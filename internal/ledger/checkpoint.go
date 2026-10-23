package ledger

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/codec"
)

// pinnedRows is the number of named rows of the table of spec-ledger.md 7.2,
// including the rows no module has built yet. The code of a reason in a
// checkpoint is its position in that table (spec-checkpoint.md 4.2.7), so
// pinnedRows is fixed by the first checkpoint format and a module never
// changes it; the code of a configured flow would be pinnedRows plus its
// position, and the ledger has no flow reasons yet (N-319).
const pinnedRows = 42

// reasonCodes is the code of each reason this ledger knows: the position of
// its row in the table of spec-ledger.md 7.2, from 0. The numbers of Reason
// are the ledger's own and change; these never do (D246, N-225). A reason added
// to the ledger takes the code of its row here, and TestReasonCode_Pinned fails
// until it does.
var reasonCodes = [NumReasons]uint64{
	Trade: 0, Initial: 1, RecipeInput: 2, RecipeOutput: 3, Disposal: 4, Consumption: 5,
	Spoilage: 7, Wear: 8, EstateMoney: 9, EstateGoods: 10, StartMoney: 11,
	LoanOut: 25, LoanRepay: 26, LoanInterest: 27,
	Split: 37, Merge: 38, Births: 39, Starvation: 40,
}

// reasonCode returns the pinned code of reason r.
func reasonCode(r Reason) uint64 { return reasonCodes[r] }

// reasonFromCode returns the reason of a pinned code, and false for a code no
// reason of this ledger has.
func reasonFromCode(code uint64) (Reason, bool) {
	for r, c := range reasonCodes {
		if c == code {
			return Reason(r), true
		}
	}
	return 0, false
}

// reloc is one row of the pending relocation totals: the money (thing 0) or
// the units of commodity thing-1 that moved with accounts from one region to
// another since the last tick's end. Plan 3 owns the operations that record
// one; the ledger keeps what a checkpoint restores so that it is saved again
// (spec-ledger.md 7.4, N-319).
type reloc struct {
	from, to, thing int
	total           int64
}

// WriteState appends the ledger's section of a checkpoint (spec-checkpoint.md
// 4.2 tag 2) to w: the non-boundary accounts, the boundary balances, the
// claims, the pending totals and, while the ledger is in its construction
// phase, the construction bucket. The tick and the construction flag are the
// root's, in the header and in tag 1. It panics, as a violated invariant, if
// a reservation is held, if an owner reference is negative or if a boundary
// account follows another kind: none can be true of a world between ticks.
func (l *Ledger) WriteState(w *codec.Writer) {
	first := l.boundaries()
	l.writeAccounts(w, first)
	for _, a := range l.accounts[:first] {
		w.I(a.balance)
	}
	l.writeClaims(w)
	l.writeFlows(w, l.totals, l.wear)
	w.Count(len(l.relocs))
	for _, x := range l.relocs {
		w.U(uint64(x.from))
		w.U(uint64(x.to))
		w.U(uint64(x.thing))
		w.U(unsigned(x.total))
	}
	if l.construction {
		l.writeFlows(w, l.bucket, l.wearBucket)
	}
}

// unsigned returns v for a field the checkpoint writes unsigned. A negative v
// would wrap to a number the reader refuses, so it is a violated invariant, found
// when the checkpoint is made and not when it is restored.
func unsigned(v int64) uint64 {
	if v < 0 {
		invariantViolated("a negative value in a field a checkpoint writes unsigned")
	}
	return uint64(v)
}

// boundaries returns the number of leading boundary accounts: the ones a
// restore finds already open, because the configuration implies them.
func (l *Ledger) boundaries() int {
	n := 0
	for n < len(l.accounts) && l.accounts[n].kind == KindBoundary {
		n++
	}
	return n
}

// writeAccounts writes the accounts after the first boundary accounts, in ID
// order: kind and closed flag, and for an open one its region, owner, balance
// and lots.
func (l *Ledger) writeAccounts(w *codec.Writer, first int) {
	rest := l.accounts[first:]
	w.Count(len(rest))
	for i := range rest {
		a := &rest[i]
		if a.kind == KindBoundary || a.owner < 0 || a.reserved != 0 || a.earmarked() {
			invariantViolated("a boundary account follows another kind, or an owner is negative, or a reservation is held")
		}
		w.U(uint64(a.kind))
		w.B(a.closed)
		if a.closed {
			continue
		}
		w.U(uint64(a.region))
		w.U(uint64(a.owner))
		w.U(unsigned(a.balance))
		w.Count(len(a.lots))
		for _, lt := range a.lots {
			w.U(uint64(lt.Commodity))
			w.U(unsigned(lt.Quantity))
			w.I(lt.Born)
			w.U(unsigned(lt.Durability))
		}
	}
}

// writeClaims writes the next claim ID and the count of open claims, and, when
// the ledger can hold a claim, the seven totals and the claims (N-319).
func (l *Ledger) writeClaims(w *codec.Writer) {
	w.U(unsigned(l.nextClaim))
	w.Count(len(l.claims))
	if l.maxClaims == 0 {
		return
	}
	t := l.claimTot
	for _, v := range [...]int64{t.Lent, t.Awarded, t.Repaid, t.InterestPaid, t.InterestFixed, t.LostPrincipal, t.LostInterest} {
		w.U(unsigned(v))
	}
	for i := range l.claims {
		c := &l.claims[i]
		for _, v := range [...]int64{c.ID, int64(c.Creditor), int64(c.Debtor), c.Principal, c.Interest, c.PrincipalOwed, c.InterestOwed, c.RateBP, c.Formed, c.Due} {
			w.U(unsigned(v))
		}
		w.I(c.Overdue)
		w.U(uint64(c.Origin))
	}
}

// writeFlows writes the non-zero flow totals of src as rows (region, reason
// code, thing, total) in ascending order of that key, then the non-zero wear
// units as rows (commodity, units).
func (l *Ledger) writeFlows(w *codec.Writer, src, wear []int64) {
	n := 0
	l.eachFlow(src, func(int, uint64, uint64, int64) { n++ })
	w.Count(n)
	l.eachFlow(src, func(region int, code, thing uint64, v int64) {
		w.U(uint64(region))
		w.U(code)
		w.U(thing)
		w.U(unsigned(v))
	})
	n = 0
	for _, v := range wear {
		if v != 0 {
			n++
		}
	}
	w.Count(n)
	for c, v := range wear {
		if v != 0 {
			w.U(uint64(c))
			w.U(unsigned(v))
		}
	}
}

// eachFlow calls f for every non-zero total of src, ordered by region, then by
// the pinned code of the reason, then by thing (0 for money, 1+c for a
// commodity), which is the order of the rows of a checkpoint.
func (l *Ledger) eachFlow(src []int64, f func(region int, code, thing uint64, v int64)) {
	for region := range l.regions {
		base := src[region*l.stride:]
		for code := range uint64(pinnedRows) {
			if r, ok := reasonFromCode(code); ok {
				l.eachThing(base, r, func(thing uint64, v int64) { f(region, code, thing, v) })
			}
		}
	}
}

// eachThing calls f for the non-zero totals of reason r in one region's block:
// money first, then the commodities in order.
func (l *Ledger) eachThing(base []int64, r Reason, f func(thing uint64, v int64)) {
	if o := l.off[r][thingMoney]; o >= 0 && base[o] != 0 {
		f(0, base[o])
	}
	if o := l.off[r][thingGoods]; o >= 0 {
		for c := range l.comms {
			if v := base[o+c]; v != 0 {
				f(uint64(c)+1, v)
			}
		}
	}
}

// ReadState reads the ledger's section of a checkpoint from r, which holds
// exactly the body of the section, into a ledger that New built and whose
// boundary accounts are open and are all it has: the configuration implies
// them (spec-checkpoint.md 5.5.1). tick is the header's tick and construction
// the flag of tag 1. It checks the rules R1 to R13 as it reads and returns an
// error that matches codec.ErrMalformed; the identities the audit checks
// (the claim parties and ranges, conservation) are the audit's, which a
// restore always runs. After an error the ledger is unusable and the caller
// drops it. It panics, as a violated invariant, on a ledger that is not as
// described.
func (l *Ledger) ReadState(r *codec.Reader, tick int64, construction bool) error {
	first := len(l.accounts)
	if l.boundaries() != first || l.tick != 0 || !l.construction {
		invariantViolated("a restore needs a ledger with only its boundary accounts, still in construction")
	}
	switch {
	case tick < 0:
		r.Failf("tick %d is below 0", tick)
	case construction && tick != 0:
		r.Failf("tick %d in the construction phase, which ends at the first step", tick)
	case !construction && tick == 0:
		r.Failf("tick 0 after the construction phase, which a tick ends")
	}
	if r.Err() != nil {
		return r.Err()
	}
	l.tick, l.construction, l.cleared = tick, construction, !construction
	l.readBody(r, first, construction)
	l.finish(r, construction)
	return r.Err()
}

// finish does what a whole section allows: the derived hints, and for a world
// past construction the baseline of the audit.
func (l *Ledger) finish(r *codec.Reader, construction bool) {
	if r.Done() != nil {
		return
	}
	l.RebuildHints()
	if construction {
		return
	}
	if err := l.restoreBaseline(); err != nil {
		r.Failf("%v", err)
	}
}

// readBody reads the sections of the body in the order WriteState wrote them:
// the accounts, the boundary balances of the first accounts, the claims, the
// pending totals and, in the construction phase, the bucket.
func (l *Ledger) readBody(r *codec.Reader, first int, construction bool) {
	l.readAccounts(r)
	for i := range r.Implied(first, 1) {
		l.accounts[i].balance = r.I()
	}
	l.readClaims(r)
	rows := l.readFlows(r, l.totals, l.wear)
	rows += l.readRelocs(r)
	if construction {
		if rows != 0 {
			r.Failf("pending totals in the construction phase")
		}
		l.readFlows(r, l.bucket, l.wearBucket)
	}
}

// readInt reads a U that is at most max, or records a violation of R13 and
// returns 0.
func readInt(r *codec.Reader, what string, max int64) int64 {
	v := r.U()
	if v > uint64(max) {
		r.Failf("%s %d is above %d", what, v, max)
		return 0
	}
	return int64(v)
}

// readAccounts reads the count of non-boundary accounts and each account in
// ID order, appending them to the ledger.
func (l *Ledger) readAccounts(r *codec.Reader) {
	n := r.Count(2)
	for range n {
		if r.Err() != nil {
			break
		}
		kind := Kind(r.Index(uint64(KindBoundary)))
		if r.B() {
			l.accounts = append(l.accounts, account{kind: kind, closed: true})
			continue
		}
		a := account{kind: kind}
		a.region = int(r.Index(uint64(l.regions)))
		a.owner = int(readInt(r, "owner", math.MaxInt))
		a.balance = readInt(r, "balance", l.maxMoney)
		a.lots = l.readLots(r)
		l.accounts = append(l.accounts, a)
	}
}

// readLots reads the lots of one account: each in canonical order with no
// repeated key, its quantity in [1, MaxQuantity] and the quantity of a
// commodity within MaxQuantity, its born and durability within the ranges of
// the commodity (the ranges of the audit, spec-ledger.md 5.1.3, 9.1).
func (l *Ledger) readLots(r *codec.Reader) []Lot {
	n := r.Count(4)
	lots := make([]Lot, 0, n)
	var sum int64
	for i := range n {
		if r.Err() != nil {
			break
		}
		lt := l.readLot(r)
		if i > 0 && cmpLot(lots[i-1], lt.key()) >= 0 {
			r.Failf("lot %d is not after the lot before it", i)
		}
		if i == 0 || lots[i-1].Commodity != lt.Commodity {
			sum = 0
		}
		var ok bool
		if sum, ok = checked.Add(sum, lt.Quantity); !ok || sum > l.maxQty {
			r.Failf("commodity %d is held above MaxQuantity %d", lt.Commodity, l.maxQty)
		}
		lots = append(lots, lt)
	}
	return lots
}

// readLot reads the four fields of a lot and checks them against its
// commodity.
func (l *Ledger) readLot(r *codec.Reader) Lot {
	lt := Lot{Commodity: int(r.Index(uint64(len(l.comms))))}
	lt.Quantity = readInt(r, "quantity", l.maxQty)
	lt.Born = r.I()
	lt.Durability = readInt(r, "durability", math.MaxInt64)
	if r.Err() != nil {
		return lt
	}
	c := l.comms[lt.Commodity]
	switch {
	case lt.Quantity < 1:
		r.Failf("a lot of commodity %d has quantity 0", lt.Commodity)
	case outsideRange(c.ShelfLife, lt.Born, -(c.ShelfLife - 1), l.tick):
		r.Failf("a lot of commodity %d has born %d, outside its range", lt.Commodity, lt.Born)
	case outsideRange(c.Durability, lt.Durability, 1, c.Durability):
		r.Failf("a lot of commodity %d has durability %d, outside its range", lt.Commodity, lt.Durability)
	}
	return lt
}

// readClaims reads the next claim ID, the count and, when the ledger can hold
// a claim, the totals and the claims in ascending ID.
func (l *Ledger) readClaims(r *codec.Reader) {
	l.nextClaim = readInt(r, "next claim ID", math.MaxInt64)
	n := r.Count(12)
	if n > l.maxClaims {
		r.Failf("%d claims are open, MaxClaims is %d", n, l.maxClaims)
		return
	}
	if l.maxClaims == 0 {
		return
	}
	t := &l.claimTot
	for _, p := range [...]*int64{&t.Lent, &t.Awarded, &t.Repaid, &t.InterestPaid, &t.InterestFixed, &t.LostPrincipal, &t.LostInterest} {
		*p = readInt(r, "claim total", math.MaxInt64)
	}
	l.claims = make([]Claim, 0, n)
	var prev uint64
	for i := range n {
		if r.Err() != nil {
			break
		}
		prev = r.Strict(prev, i > 0)
		c := l.readClaim(r, prev)
		l.claims = append(l.claims, c)
	}
}

// readClaim reads the fields of a claim after its ID. The parties must be
// accounts of the ledger; their kinds, the ranges and the owed amounts are the
// audit's.
func (l *Ledger) readClaim(r *codec.Reader, id uint64) Claim {
	c := Claim{ID: int64(id)}
	if id >= uint64(l.nextClaim) {
		r.Failf("claim %d is not below the next claim ID %d", id, l.nextClaim)
	}
	n := uint64(len(l.accounts))
	c.Creditor, c.Debtor = int(r.Index(n)), int(r.Index(n))
	for _, p := range [...]*int64{&c.Principal, &c.Interest, &c.PrincipalOwed, &c.InterestOwed, &c.RateBP, &c.Formed, &c.Due} {
		*p = readInt(r, "claim field", math.MaxInt64)
	}
	c.Overdue = r.I()
	c.Origin = int(r.Index(2))
	return c
}

// readFlows reads the rows of flow totals into dst and the rows of wear units
// into wear, each in strictly ascending key order with no zero total, and
// returns the number of rows.
func (l *Ledger) readFlows(r *codec.Reader, dst, wear []int64) int {
	nc := uint64(len(l.comms))
	n := r.Count(4)
	var prev uint64
	for i := range n {
		if r.Err() != nil {
			break
		}
		region := r.Index(uint64(l.regions))
		reason, ok := reasonFromCode(r.U())
		thing := r.Index(nc + 1)
		total := readInt(r, "flow total", math.MaxInt64)
		slot, err := l.slotOf(reason, int(thing)-1)
		key := (region*pinnedRows+reasonCode(reason))*(nc+1) + thing
		switch {
		case r.Err() != nil:
		case !ok || err != nil:
			r.Failf("row %d names a reason that does not move that thing, or a code no reason has", i)
		case total == 0 || i > 0 && key <= prev:
			r.Failf("row %d has a total of 0 or is not after the row before it", i)
		default:
			dst[int(region)*l.stride+slot] = total
		}
		prev = key
	}
	return n + l.readWear(r, wear, nc)
}

// readWear reads the rows (commodity, units) of the wear units into wear and
// returns the number of rows.
func (l *Ledger) readWear(r *codec.Reader, wear []int64, nc uint64) int {
	n := r.Count(2)
	var prev uint64
	for i := range n {
		if r.Err() != nil {
			break
		}
		prev = r.Strict(prev, i > 0)
		units := readInt(r, "wear units", math.MaxInt64)
		switch {
		case r.Err() != nil:
		case prev >= nc || units == 0:
			r.Failf("wear row %d names a commodity outside the table or has 0 units", i)
		default:
			wear[prev] = units
		}
	}
	return n
}

// readRelocs reads the pending relocation totals: rows (from, to, thing,
// total) sorted by that key, between two different regions, with a total of at
// least 1. It returns the number of rows.
func (l *Ledger) readRelocs(r *codec.Reader) int {
	nr, nc := uint64(l.regions), uint64(len(l.comms))
	n := r.Count(4)
	l.relocs = l.relocs[:0]
	var prev uint64
	for i := range n {
		if r.Err() != nil {
			break
		}
		from, to := r.Index(nr), r.Index(nr)
		thing := r.Index(nc + 1)
		total := readInt(r, "relocation total", math.MaxInt64)
		key := (from*nr+to)*(nc+1) + thing
		switch {
		case r.Err() != nil:
		case from == to || total == 0 || i > 0 && key <= prev:
			r.Failf("relocation row %d is within a region, has a total of 0 or is not after the row before it", i)
		default:
			l.relocs = append(l.relocs, reloc{int(from), int(to), int(thing), total})
		}
		prev = key
	}
	return n
}

// restoreBaseline sets the per-boundary baseline of a world restored after its
// first step: the restored balance less what the pending totals say the
// boundary has moved since the last tick's end (spec-ledger.md 9.1). A world
// restored in construction keeps the baseline of 0 that construction has.
func (l *Ledger) restoreBaseline() error {
	return l.eachBoundarySlot(func(_ Reason, class Class, _, slot int) error {
		id := l.boundary[slot]
		if id < 0 {
			return nil
		}
		var sum int64
		for region := range l.regions {
			var ok bool
			if sum, ok = checked.Add(sum, l.totals[region*l.stride+slot]); !ok {
				return ErrBalanceOverflow
			}
		}
		if class == Faucet {
			sum = -sum
		}
		base, ok := checked.Sub(l.accounts[id].balance, sum)
		if !ok {
			return ErrBalanceOverflow
		}
		l.base[slot] = base
		return nil
	})
}
