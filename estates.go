package economysim

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file is the estate of a region (spec-estates.md): its identity, the
// price state of its listings and the two price rules. Phase 2 holds the
// identity, the rules and the sorted-row operations; the accounts, the pool,
// the intake, the asks and the checkpoint read follow in the same file.

// listing is the price state of one held lot of an estate (spec-estates.md
// 2.7.3, 2.7.5): the lot's key, its listed price and whether an ask for it was
// admitted in the tick. asked is scratch of the tick and is never saved.
type listing struct {
	key   ledger.Key
	price Price
	asked bool
}

// estates holds the estate of every region, indexed by RegionID: the ledger
// account of each estate and the listings of its lots, sorted by key and
// without a duplicate.
type estates struct {
	accounts []int
	listed   [][]listing
	// lots is scratch for the lots of the account being read, reused across
	// calls (architecture.md 2.9).
	lots []ledger.Lot
	// scratch is the second row buffer of a reconcile; it is swapped with the
	// region's rows so that a warmed walk allocates nothing.
	scratch []listing
}

// errListingRows is the reason restore refuses listing rows (D109). The
// checkpoint row wraps it in ErrCheckpointCorrupt.
var errListingRows = errors.New("economysim: invalid estate listing rows")

// agentID returns the agent ID of the estate pseudo-agent of region r, which is
// the RegionID (spec-regions.md 3.6.1, spec-estates.md 2.7.6).
func (e *estates) agentID(r RegionID) AgentID { return AgentID(r) }

// regionOf is the reverse of agentID: ok is true when a is below the region
// count, so a is the estate pseudo-agent of the region returned.
func (e *estates) regionOf(a AgentID) (RegionID, bool) {
	if a < 0 || a >= AgentID(len(e.accounts)) {
		return 0, false
	}
	return RegionID(a), true
}

// account returns the ledger account of the estate of region r.
func (e *estates) account(r RegionID) int { return e.accounts[r] }

// firstPrice is the price of a lot with no listing (spec-estates.md 2.7.3,
// FR-EST-008): the reference price for a commodity that does not wear
// (durability 0), else ref * left / durability truncated, and in both cases not
// below the resolution floor. ok is false only when the product does not fit,
// which a validated configuration cannot produce (left <= durability and ref is
// at most MaxPrice).
func firstPrice(ref Price, durability, left, floor int64) (Price, bool) {
	base := int64(ref)
	if durability != 0 {
		var ok bool
		if base, ok = checked.MulDiv(base, left, durability); !ok {
			return 0, false
		}
	}
	return Price(max(floor, base)), true
}

// markedDown is the price after the estate markdown (spec-estates.md 2.7.3,
// FR-EST-009, D104): the price less trunc(price * mdBP / 10 000), never below
// the floor, and a price at or below the floor stays where it is. The product
// is held in 128 bits, so it cannot overflow for a price within MaxPrice.
func markedDown(price Price, mdBP, floor int64) Price {
	if int64(price) <= floor {
		return price
	}
	cut, ok := checked.MulDiv(int64(price), mdBP, wholeBP)
	if !ok {
		// shortcut: a cut that does not fit means mdBP is out of bound; the
		// floor is the only safe price then.
		return Price(floor)
	}
	return Price(max(floor, int64(price)-cut))
}

// cmpKey orders two lot keys by (commodity, born, durability), the canonical
// order of spec-ledger.md 5.1.3.
func cmpKey(a, b ledger.Key) int {
	if c := cmp.Compare(a.Commodity, b.Commodity); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Born, b.Born); c != 0 {
		return c
	}
	return cmp.Compare(a.Durability, b.Durability)
}

// findRow returns where the row with key k is or would be inserted.
func findRow(rows []listing, k ledger.Key) (int, bool) {
	return slices.BinarySearchFunc(rows, k, func(r listing, k ledger.Key) int { return cmpKey(r.key, k) })
}

// insertRow puts row into the sorted rows. A key already present is one row:
// the existing row is kept, since a lot that gains units keeps its price
// (D102).
func insertRow(rows []listing, row listing) []listing {
	i, found := findRow(rows, row.key)
	if found {
		return rows
	}
	return slices.Insert(rows, i, row)
}

// reconcile merges the sorted rows with the sorted lots an account holds and
// appends the result to dst[:0]: a row whose lot is held is kept as it is (its
// price and mark), a row whose lot is gone is dropped, and a held lot with no
// row gets one at first(lot) when first is not nil (the insert-enabled walk of
// the asks, D102) or is omitted when it is nil (the filter-only walk of the
// checkpoint read, D109). dst must not share storage with rows; it allocates
// only when dst has no room.
func reconcile(dst, rows []listing, lots []ledger.Lot, first func(ledger.Lot) Price) []listing {
	dst = dst[:0]
	i, j := 0, 0
	for i < len(rows) && j < len(lots) {
		k := ledger.Key{Commodity: lots[j].Commodity, Born: lots[j].Born, Durability: lots[j].Durability}
		switch c := cmpKey(rows[i].key, k); {
		case c == 0:
			dst = append(dst, rows[i])
			i++
			j++
		case c < 0:
			i++ // the lot is gone
		default:
			if first != nil {
				dst = append(dst, listing{key: k, price: first(lots[j])})
			}
			j++
		}
	}
	if first != nil {
		for ; j < len(lots); j++ {
			k := ledger.Key{Commodity: lots[j].Commodity, Born: lots[j].Born, Durability: lots[j].Durability}
			dst = append(dst, listing{key: k, price: first(lots[j])})
		}
	}
	return dst
}

// newEstates is the estates of regions regions with no account bound yet; openEstates
// and attachEstates fill the accounts.
func newEstates(regions int) *estates {
	return &estates{accounts: make([]int, regions), listed: make([][]listing, regions)}
}

// openEstates opens the estate account of every region, in ascending RegionID
// order, and returns the estates (spec-estates.md 2.7.1, D108). Every account
// that exists must be a boundary account, since FR-EST-005 and the account
// numbering of spec-ledger.md 3.4.3 depend on the estates coming right after
// them; it is a count of kinds, not of accounts, because the declared boundary
// accounts of the modules open before the estates.
func openEstates(l *ledger.Ledger, regions int) (*estates, error) {
	for i := range l.Len() {
		if v, err := l.Account(i); err != nil || v.Kind != ledger.KindBoundary {
			invariant("an account that is not a boundary account exists before the estates are opened")
		}
	}
	e := newEstates(regions)
	for r := range regions {
		id, err := l.Open(ledger.KindEstate, r, r)
		if err != nil {
			return nil, fmt.Errorf("open the estate of region %d: %w", r, err)
		}
		e.accounts[r] = id
	}
	return e, nil
}

// seedPool creates the pool of region r by the reason initial, from the
// boundary account of initial money, while the ledger is in construction
// (spec-estates.md FR-EST-015). An amount of 0 is skipped. Outside
// construction it is ledger.ErrNotConstruction, though the ledger itself would
// accept an initial transfer during a run.
func (e *estates) seedPool(l *ledger.Ledger, r RegionID, m Money) error {
	if !l.InConstruction() {
		return fmt.Errorf("seed the pool of region %d: %w", r, ledger.ErrNotConstruction)
	}
	if m == 0 {
		return nil
	}
	src, ok := l.Boundary(ledger.Initial, ledger.Money)
	if !ok {
		invariant("the boundary account of initial money is not open")
	}
	if err := l.Transfer(src, e.accounts[r], int64(m), ledger.Initial); err != nil {
		return fmt.Errorf("seed the pool of region %d: %w", r, err)
	}
	return nil
}

// pool returns the money of the estate of region r, which is the regional pool
// (spec-estates.md FR-EST-013). The estate places no bid, so none is reserved.
func (e *estates) pool(l *ledger.Ledger, r RegionID) Money {
	v, err := l.Account(e.accounts[r])
	if err != nil {
		invariant("the estate account is unreadable: " + err.Error())
	}
	return Money(v.Balance)
}

// grant gives min(want, pool) of region r to account to by reason, and returns
// what it gave (spec-estates.md 2.3, FR-EST-014, FR-EST-022). It gives nothing
// for an amount below 1, which the ledger would refuse. The caller decides the
// budget and the whole households; reason is start_money for a new agent and
// births for households added to one.
func (e *estates) grant(l *ledger.Ledger, r RegionID, to int, want Money, reason ledger.Reason) (Money, error) {
	given := min(want, e.pool(l, r))
	if given < 1 {
		return 0, nil
	}
	if err := l.Transfer(e.accounts[r], to, int64(given), reason); err != nil {
		return 0, fmt.Errorf("grant from the pool of region %d: %w", r, err)
	}
	return given, nil
}

// acceptMoney moves amount from account from to the estate of region r by
// reason, for a weight reduction (split, merge, starvation). It moves nothing
// for an amount below 1.
func (e *estates) acceptMoney(l *ledger.Ledger, from int, r RegionID, amount Money, reason ledger.Reason) error {
	if amount < 1 {
		return nil
	}
	if err := l.Transfer(from, e.accounts[r], int64(amount), reason); err != nil {
		return fmt.Errorf("pay the estate of region %d: %w", r, err)
	}
	return nil
}

// refused handles a refusal of an intake. No reservation is open when an
// intake runs, so a refusal is a violated invariant (D106), except an overflow
// of the estate's balance, which is returned for the world to fail the step.
func refused(what string, r RegionID, from int, err error) error {
	where := fmt.Sprintf("%s (estate of region %d, account %d)", what, r, from)
	if errors.Is(err, ledger.ErrBalanceOverflow) {
		return fmt.Errorf("%s: %w", where, err)
	}
	invariant(where + ": " + err.Error())
	return nil
}

// acceptUnits moves qty units of one commodity from account from to the estate
// of region r by reason, for the goods of a weight reduction (the ledger takes
// the oldest lots first and keeps each lot's key). It returns the units moved,
// which feed WeightChanged.Units, and moves nothing for qty below 1.
func (e *estates) acceptUnits(l *ledger.Ledger, from int, r RegionID, commodity int, qty int64, reason ledger.Reason) (int64, error) {
	if qty < 1 {
		return 0, nil
	}
	if err := l.Move(from, e.accounts[r], commodity, qty, reason); err != nil {
		return 0, refused("move units to the estate", r, from, err)
	}
	return qty, nil
}

// acceptGoods moves the whole lots of the commodities only selects (nil: all)
// from account from to the estate of region r, in key order and keeping the
// keys, and returns the number of lots moved (GoodsListed, D101).
func (e *estates) acceptGoods(l *ledger.Ledger, from int, r RegionID, reason ledger.Reason, only func(commodity int) bool) (int64, error) {
	lots, err := l.AppendLots(e.lots[:0], from)
	e.lots = lots
	if err != nil {
		invariant("the account handing goods to the estate is unreadable: " + err.Error())
	}
	var n int64
	for _, lt := range lots {
		if only != nil && !only(lt.Commodity) {
			continue
		}
		key := ledger.Key{Commodity: lt.Commodity, Born: lt.Born, Durability: lt.Durability}
		if err := l.MoveLot(from, e.accounts[r], key, lt.Quantity, reason); err != nil {
			return n, refused("move a lot to the estate", r, from, err)
		}
		n++
	}
	return n, nil
}

// receive is the intake of a removal (spec-agents.md 3.6 steps 3 and 4): the
// whole balance of account from by estate_money, then every lot by
// estate_goods. It returns the money passed and the lots moved. The caller
// closes the account and emits the events. It is not atomic: an error here
// fails the step, and a failed world is never observed or saved half-way
// (spec-world.md 3.5), so a rollback would only add a second path to test.
func (e *estates) receive(l *ledger.Ledger, from int, r RegionID) (Money, int64, error) {
	v, err := l.Account(from)
	if err != nil {
		invariant("the account of a removed agent is unreadable: " + err.Error())
	}
	money := Money(v.Balance)
	if money > 0 {
		if err := l.Transfer(from, e.accounts[r], int64(money), ledger.EstateMoney); err != nil {
			return 0, 0, refused("move money to the estate", r, from, err)
		}
	}
	lots, err := e.acceptGoods(l, from, r, ledger.EstateGoods, nil)
	return money, lots, err
}

// keyOf is the ledger key of a lot.
func keyOf(lt ledger.Lot) ledger.Key {
	return ledger.Key{Commodity: lt.Commodity, Born: lt.Born, Durability: lt.Durability}
}

// heldLots reads the lots the estate of region r holds into the scratch.
func (e *estates) heldLots(l *ledger.Ledger, r RegionID) []ledger.Lot {
	lots, err := l.AppendLots(e.lots[:0], e.accounts[r])
	e.lots = lots
	if err != nil {
		invariant("the estate account is unreadable: " + err.Error())
	}
	return lots
}

// asks appends to dst one named-lot ask per lot the estate of region r holds,
// in key order, at the lot's listed price, for all its units (spec-estates.md
// 2.7.3, D103), refs being the world's table of effective reference prices. A held lot with no listing gets one at the first price. The
// marks of the last tick are cleared. It allocates only when dst has no room.
func (e *estates) asks(l *ledger.Ledger, cfg *Config, refs []Price, r RegionID, dst []Order) ([]Order, error) {
	floor, ok := resolutionFloor(cfg.World)
	if !ok {
		invariant("the belief step is outside its bound")
	}
	lots := e.heldLots(l, r)
	first := func(lt ledger.Lot) Price {
		c := &cfg.Commodities[lt.Commodity]
		p, ok := firstPrice(refs[int(r)*len(cfg.Commodities)+lt.Commodity], c.Durability, lt.Durability, floor)
		if !ok {
			invariant("the first price of an estate lot overflows")
		}
		return p
	}
	rows := reconcile(e.scratch, e.listed[r], lots, first)
	e.scratch, e.listed[r] = e.listed[r], rows
	for i, lt := range lots {
		rows[i].asked = false
		dst = append(dst, Order{
			Agent: e.agentID(r), Commodity: CommodityID(lt.Commodity), Side: Ask,
			Price: rows[i].price, Quantity: Quantity(lt.Quantity),
			Lot: LotKey{Named: true, Born: lt.Born, Durability: lt.Durability},
		})
	}
	return dst, nil
}

// facts is what the world resolves for an order of the estate of region r: a
// known agent that may place asks only, with a commodity that is always
// tradable (spec-estates.md 2.7.2, FR-EST-006).
func (e *estates) facts(r RegionID, side Side, tally *receiptTally) orderFacts {
	return orderFacts{
		Account: e.accounts[r], AgentKnown: true, SideAllowed: side == Ask,
		CommodityKnown: true, Tradable: true, Tally: tally,
	}
}

// admitted marks the lot of an ask that the book admitted, so that the
// markdown of the tick reads it. A rejected ask is never marked (D103).
func (e *estates) admitted(r RegionID, o Order) {
	i, found := findRow(e.listed[r], ledger.Key{Commodity: int(o.Commodity), Born: o.Lot.Born, Durability: o.Lot.Durability})
	if !found {
		invariant("an admitted estate ask names a lot with no listing")
	}
	e.listed[r][i].asked = true
}

// markdown is P5.3 for region r: a lot with an admitted ask that is still held
// falls by EstateMarkdown (D104), the row of a lot that is gone is dropped
// (D102), and the marks are cleared.
func (e *estates) markdown(l *ledger.Ledger, cfg *Config, r RegionID) error {
	floor, ok := resolutionFloor(cfg.World)
	if !ok {
		invariant("the belief step is outside its bound")
	}
	rows := reconcile(e.scratch, e.listed[r], e.heldLots(l, r), nil)
	e.scratch, e.listed[r] = e.listed[r], rows
	for i := range rows {
		if rows[i].asked {
			rows[i].price = markedDown(rows[i].price, cfg.Regions[r].EstateMarkdown, floor)
			rows[i].asked = false
		}
	}
	return nil
}

// atFloor reports whether the estate of region r lists lot lt at or below the
// resolution floor. A held lot with no row is not listed yet, so it is not.
func (e *estates) atFloor(r RegionID, lt ledger.Lot, floor int64) bool {
	i, found := findRow(e.listed[r], keyOf(lt))
	return found && int64(e.listed[r][i].price) <= floor
}

// rows appends to dst[:0] the listings of the lots the estate of region r
// holds, in key order, for the checkpoint (D109). It is filter-only: a stale
// row is not returned and a held lot with no row is omitted.
func (e *estates) rows(l *ledger.Ledger, r RegionID, dst []listing) ([]listing, error) {
	out := reconcile(dst, e.listed[r], e.heldLots(l, r), nil)
	for i := range out {
		out[i].asked = false
	}
	return out, nil
}

// restore rebuilds the listings of region r from rows, or refuses them with
// errListingRows and changes nothing: rows out of order or repeated, a price
// outside 1 to MaxPrice, or a key that is not a lot the estate holds (D109,
// FR-CHK-023).
func (e *estates) restore(l *ledger.Ledger, r RegionID, rows []listing) error {
	lots := e.heldLots(l, r)
	for i, row := range rows {
		switch {
		case i > 0 && cmpKey(rows[i-1].key, row.key) >= 0:
			return fmt.Errorf("%w: row %d is out of order or repeated (%+v)", errListingRows, i, row.key)
		case row.price < 1 || row.price > MaxPrice:
			return fmt.Errorf("%w: row %d has price %d outside 1 to %d", errListingRows, i, row.price, MaxPrice)
		}
		if _, held := slices.BinarySearchFunc(lots, row.key, func(lt ledger.Lot, k ledger.Key) int { return cmpKey(keyOf(lt), k) }); !held {
			return fmt.Errorf("%w: row %d names a lot the estate does not hold (%+v)", errListingRows, i, row.key)
		}
	}
	e.listed[r] = append(e.listed[r][:0], rows...)
	for i := range e.listed[r] {
		e.listed[r][i].asked = false
	}
	return nil
}

// The pool policy of the Estates module (relief at P1.6, the pool as a
// money-flow target) is at the end of the file, after the checkpoint section.

// saveEstates is tag 8 (spec-checkpoint.md 4.2; plan 2-2 D249): for each region in
// ascending RegionID the count and the rows of the listings of the lots its estate
// holds, each as the commodity, the born tick (signed), the durability and the
// listed price. The asked mark is scratch and is not written.
func (w *World) saveEstates(cw *codec.Writer) error {
	var rows []listing
	for r := range w.cfg.Regions {
		var err error
		// Excluded from coverage (testing.md 9.5): rows has no way to fail today and
		// keeps the error result of the restore's counterpart; the branch only wraps it.
		if rows, err = w.estates.rows(w.ledger, RegionID(r), rows[:0]); err != nil {
			return fmt.Errorf("checkpoint: listings of region %d: %w", r, err)
		}
		cw.Count(len(rows))
		for _, l := range rows {
			cw.U(uint64(l.key.Commodity))
			cw.I(l.key.Born)
			cw.U(unsigned(l.key.Durability))
			cw.U(unsigned(l.price))
		}
	}
	return nil
}

// attachEstates binds the estate of every region to the account a restored ledger
// holds for it: the estates are the first non-boundary accounts, in region order,
// each open, of kind Estate, with its region as owner and region (spec-checkpoint.md
// 4.2.2, FR-CHK-049). It is openEstates for a ledger that already has them.
func attachEstates(l *ledger.Ledger, regions int) (*estates, error) {
	first := 0
	for first < l.Len() {
		if v, _ := l.Account(first); v.Kind != ledger.KindBoundary {
			break
		}
		first++
	}
	e := newEstates(regions)
	for r := range regions {
		v, err := l.Account(first + r)
		if err != nil || v.Kind != ledger.KindEstate || v.Closed || v.Owner != r || v.Region != r {
			return nil, fmt.Errorf("account %d is not the open estate account of region %d", first+r, r)
		}
		e.accounts[r] = first + r
	}
	return e, nil
}

// loadEstates is tag 8: for each region the listings of the lots its estate holds.
// estates.restore refuses rows out of order, a price outside its bounds or a lot the
// estate does not hold.
func (w *World) loadEstates(_ *restoreCtx, r *codec.Reader) error {
	var rows []listing
	for reg := range w.cfg.Regions {
		rows = rows[:0]
		for range r.Count(4) {
			c := r.Index(uint64(len(w.cfg.Commodities)))
			born := r.I()
			dur := r.Bounded("durability", math.MaxInt64)
			price := r.Bounded("price", MaxPrice)
			if r.Err() != nil {
				return nil
			}
			rows = append(rows, listing{key: ledger.Key{Commodity: int(c), Born: born, Durability: dur}, price: Price(price)})
		}
		if r.Err() != nil {
			return nil
		}
		if err := w.estates.restore(w.ledger, RegionID(reg), rows); err != nil {
			return err
		}
	}
	return nil
}

// reliefRelease is the money region's pool releases as relief this tick,
// pool * PoolRelease / 10 000 truncated (spec-estates.md 2.6, FR-EST-017). ok is
// false only when the product does not fit, which a pool below MaxMoney and a
// PoolRelease within its bound cannot cause.
func reliefRelease(pool Money, poolRelease int64) (release int64, ok bool) {
	return checked.MulDiv(int64(pool), poolRelease, wholeBP)
}

// reliefNeed is the money that lifts an agent's income to the relief line,
// upkeep * (line - ratio) / 10 000 truncated, for an agent whose ratio is
// present and below the line, else 0 (FR-EST-018). ok is false when the
// subtraction or the product does not fit.
func reliefNeed(upkeep, line, ratio int64, present bool) (need int64, ok bool) {
	if !present || ratio >= line {
		return 0, true
	}
	gap, ok := checked.Sub(line, ratio)
	if !ok {
		return 0, false
	}
	return checked.MulDiv(upkeep, gap, wholeBP)
}

// reliefShare is the payment of an agent of need need among agents whose needs
// sum to sumNeed, from a release of release: release * need / sumNeed
// truncated, and never more than need (FR-EST-018). It depends on the agent's
// need and the totals only, so no order of agents changes it (FR-EST-019). A
// need or a sum below 1 pays nothing.
func reliefShare(release, need, sumNeed int64) (pay int64, ok bool) {
	if need < 1 || sumNeed < 1 {
		return 0, true
	}
	share, ok := checked.MulDiv(release, need, sumNeed)
	return min(share, need), ok
}

// reliefRow is what P1.6 keeps of one agent between its two passes: the region
// and the need.
type reliefRow struct {
	region RegionID
	need   int64
}

// reliefPass is the scratch of P1.6, built when the module is on and reused
// every tick. rows is by agent slot and grows only when agents are added.
// release and sumNeed are by region; paid, recipients and households are what
// the pass paid in each region, for the event.
type reliefPass struct {
	rows                         []reliefRow
	release, sumNeed             []int64
	paid, recipients, households []int64
}

// newReliefPass builds the scratch of a configuration.
func newReliefPass(regions int) *reliefPass {
	return &reliefPass{
		release: make([]int64, regions), sumNeed: make([]int64, regions),
		paid: make([]int64, regions), recipients: make([]int64, regions),
		households: make([]int64, regions),
	}
}

// estatesReasons is the ledger reasons the estates module declares.
func estatesReasons() []ledger.Reason { return []ledger.Reason{ledger.Relief} }

// p16Relief is P1.6 (spec-estates.md 2.6, plan 2-7 D347 to D350): in ascending
// RegionID each pool releases pool * PoolRelease / 10 000, shared among the
// agents of the region whose welfare ratio is below ReliefBelow in proportion to
// their needs. Every need is computed before the first payment, so a refusal
// leaves the pool as it was; what is not paid stays in the pool. It keeps no
// state across ticks. It returns the first overflow or ledger refusal, which
// fails the step.
func (w *World) p16Relief() error {
	rp := w.relief
	if rp == nil || !w.reliefReleases() {
		return nil
	}
	w.valuePrices()
	if err := w.reliefNeeds(); err != nil {
		return err
	}
	return w.reliefPay()
}

// reliefReleases fills the release of every region and clears the totals of the
// pass. It reports whether any region releases at least 1.
func (w *World) reliefReleases() bool {
	rp, any := w.relief, false
	for r := range w.cfg.Regions {
		rel, ok := reliefRelease(w.estates.pool(w.ledger, RegionID(r)), w.cfg.Regions[r].PoolRelease)
		if !ok {
			invariant("a release of the pool does not fit in int64")
		}
		rp.release[r], rp.sumNeed[r] = rel, 0
		rp.paid[r], rp.recipients[r], rp.households[r] = 0, 0, 0
		any = any || rel >= 1
	}
	return any
}

// reliefNeeds is the first pass over the agents: the need of every household of a
// region that releases, and the sum of the needs of each region. The estate
// pseudo-agents, below the region count, are not households and have none.
func (w *World) reliefNeeds() error {
	rp, nc := w.relief, len(w.cfg.Commodities)
	if cap(rp.rows) < len(w.agents) {
		rp.rows = make([]reliefRow, len(w.agents))
	}
	rp.rows = rp.rows[:len(w.agents)]
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		a := &w.agents[i]
		rp.rows[i] = reliefRow{}
		if a.weight < 1 {
			continue
		}
		g := w.regionOf(a)
		if rp.release[g] < 1 {
			continue
		}
		role := &w.cfg.Roles[a.role]
		u := w.welfare.cost.tick(w.roles.row(a.role).upkeepIn(g), w.oc.valued[int(g)*nc:(int(g)+1)*nc], a.weight)
		ratio, present := a.history.Ratio(u, role.DefaultRatio)
		need, ok := reliefNeed(u, w.cfg.Regions[g].ReliefBelow, ratio, present)
		sum, ok2 := checked.Add(rp.sumNeed[g], need)
		if !ok || !ok2 {
			return fmt.Errorf("relief need of agent %d in region %d: %w", a.id, g, ErrOverflow)
		}
		rp.rows[i], rp.sumNeed[g] = reliefRow{region: g, need: need}, sum
	}
	return nil
}

// reliefPay is the second pass: each agent with a need is paid its share by the
// reason relief, which is income of the tick in its welfare record, in ascending
// agent ID.
func (w *World) reliefPay() error {
	rp := w.relief
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		row := rp.rows[i]
		if row.need < 1 {
			continue
		}
		a := &w.agents[i]
		pay, ok := reliefShare(rp.release[row.region], row.need, rp.sumNeed[row.region])
		if !ok {
			return fmt.Errorf("relief share of agent %d in region %d: %w", a.id, row.region, ErrOverflow)
		}
		if pay < 1 {
			continue
		}
		given, err := w.estates.grant(w.ledger, row.region, a.account, Money(pay), ledger.Relief)
		if err != nil {
			return fmt.Errorf("relief to agent %d in region %d: %w", a.id, row.region, err)
		}
		if given != Money(pay) {
			invariant("the pool of a region cannot pay a relief within its release")
		}
		f := w.welfare.at(i)
		f.relief = satAdd(f.relief, pay)
		rp.paid[row.region], rp.recipients[row.region] = rp.paid[row.region]+pay, rp.recipients[row.region]+1
		rp.households[row.region] += a.weight
	}
	return nil
}
