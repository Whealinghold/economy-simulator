package economysim

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/clearing"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// Clearing is the rule by which a market matches its admitted orders and sets
// the price of each trade (spec-clearing.md 8). The world validates the whole
// list it returns before settling any trade, so a faulty implementation can
// only fail the step with ErrInvalidTrade.
//
// rng is the world's random source. An implementation must not create its own
// source, use a global one, or keep rng after Clear returns, because that
// breaks determinism (architecture.md 7.4, 7.7). The list Clear returns is read
// by the world and not owned by it.
//
// An implementation that keeps state between calls must also implement
// Stateful so that a checkpoint can save and restore it (architecture.md 7.7):
//   - what SaveState writes must hold every piece of state that can influence
//     a result; no state lives in closures, package variables or goroutines;
//   - SaveState must not change the extension and must give equal bytes for
//     equal states;
//   - LoadState must replace the whole state, must validate its input and
//     return an error for a version or bytes it cannot read, and must not
//     keep the slice it is given;
//   - neither method receives or may create a random source.
//
// An instance that holds state must not be shared between worlds.
type Clearing interface {
	// Clear matches bids against asks and returns the trades.
	Clear(bids, asks []Order, rng *rand.Rand) []Trade
}

// AverageClearing is the default Clearing, registered as "average": bids from
// the highest limit and asks from the lowest are paired while they cross, each
// pair trades the smaller remaining quantity at the midpoint of the two limits
// rounded half to even (spec-clearing.md 4, 5.1, 5.2). It keeps no state.
type AverageClearing struct{}

var _ Clearing = AverageClearing{}

// Clear implements Clearing. It draws one word of rng per order, bids first
// and then asks, each in input order, as the tie-break key of the order, then
// matches by (limit, key, input position) (spec-clearing.md 4.2).
func (AverageClearing) Clear(bids, asks []Order, rng *rand.Rand) []Trade {
	bo := make([]clearing.Offer, len(bids))
	ao := make([]clearing.Offer, len(asks))
	for i, o := range bids {
		bo[i] = clearing.Offer{Ref: i, Price: int64(o.Price), Qty: int64(o.Quantity), Key: rng.Uint64()}
	}
	for i, o := range asks {
		ao[i] = clearing.Offer{Ref: i, Price: int64(o.Price), Qty: int64(o.Quantity), Key: rng.Uint64()}
	}
	ms := clearing.Average(nil, bo, ao)
	trades := make([]Trade, len(ms))
	for i, m := range ms {
		b, a := bids[m.BidRef], asks[m.AskRef]
		trades[i] = Trade{
			Commodity: b.Commodity, Buyer: b.Agent, Seller: a.Agent,
			Price: Price(m.Price), Quantity: Quantity(m.Qty), Lot: a.Lot,
		}
	}
	return trades
}

// codeReasons maps the admission codes of internal/clearing to the rejection
// reasons of event.go, indexed by code (spec-clearing.md 3.2). It is never
// mutated. The rows of the modules (not_permitted, no_room) are added with
// their codes.
var codeReasons = [...]Reason{
	clearing.UnknownAgent:      ReasonUnknownAgent,
	clearing.SideNotAllowed:    ReasonSideNotAllowed,
	clearing.UnknownCommodity:  ReasonUnknownCommodity,
	clearing.NotTradable:       ReasonNotTradable,
	clearing.InvalidPrice:      ReasonInvalidPrice,
	clearing.InvalidQuantity:   ReasonInvalidQuantity,
	clearing.DuplicateOrder:    ReasonDuplicateOrder,
	clearing.Overflow:          ReasonOverflow,
	clearing.HoldingLimit:      ReasonHoldingLimit,
	clearing.ReceiptLimit:      ReasonReceiptLimit,
	clearing.InsufficientFunds: ReasonInsufficientFunds,
	clearing.InsufficientStock: ReasonInsufficientStock,
}

// reasonOfCode returns the Reason of a rejection code, and false for Admitted
// and for a code with no reason.
func reasonOfCode(c clearing.Code) (Reason, bool) {
	if c == clearing.Admitted || int(c) >= len(codeReasons) || codeReasons[c] == "" {
		return "", false
	}
	return codeReasons[c], true
}

// codeOfReason is the inverse of reasonOfCode, and false for a Reason that is
// no rejection code of internal/clearing.
func codeOfReason(r Reason) (clearing.Code, bool) {
	for c := range codeReasons {
		if r != "" && codeReasons[c] == r {
			return clearing.Code(c), true
		}
	}
	return clearing.Admitted, false
}

// marketReport is what one market published at its last clearing: exactly the
// fields of spec-clearing.md 16, which the checkpoint saves. The world holds
// one per (region, commodity).
type marketReport struct {
	// Price is the published market price.
	Price Price
	// Age is the ticks since a unit last traded; 0 after a trade.
	Age int64
	// Volume is the units traded in the tick.
	Volume Quantity
	// UnmatchedBid and UnmatchedAsk are the unfilled units of each side.
	UnmatchedBid, UnmatchedAsk Quantity
	// BestBid is the highest unmatched bid, valid when HasBestBid; BestAsk is
	// the lowest unmatched ask, valid when HasBestAsk.
	BestBid, BestAsk       Price
	HasBestBid, HasBestAsk bool
}

// seedReport is the report of a market before its first clearing: the seed
// price, age 0 and no volume or unmatched units (spec-clearing.md 5.4).
func seedReport(price Price) marketReport { return marketReport{Price: price} }

// receiptTally is the worst-case money each ledger account may receive from
// the asks admitted in the round, indexed by account (spec-clearing.md 3.2.2).
// The world owns one and resets it at the start of P3.2.
type receiptTally struct{ v []int64 }

// reset zeroes every account's tally and keeps the storage.
func (t *receiptTally) reset() { clear(t.v) }

// of returns the tally of account acct.
func (t *receiptTally) of(acct int) int64 {
	if acct < len(t.v) {
		return t.v[acct]
	}
	return 0
}

// set stores the tally of account acct, growing the table if needed.
func (t *receiptTally) set(acct int, v int64) {
	for acct >= len(t.v) {
		t.v = append(t.v, 0)
	}
	t.v[acct] = v
}

// orderFacts is what the world resolves for one order and the ledger does not
// know (spec-clearing.md 3.2): the agent's ledger account and the table rows.
type orderFacts struct {
	// Account is the ledger account of the agent, read only when AgentKnown.
	Account int
	// AgentKnown is whether the agent exists and is in the book's region.
	AgentKnown bool
	// SideAllowed is false for a pseudo-agent that placed the side it may not.
	SideAllowed bool
	// CommodityKnown is whether the commodity is in the configuration.
	CommodityKnown bool
	// Tradable is whether the commodity is in the agent's role's tradable set;
	// true for a pseudo-agent.
	Tradable bool
	// Tally is the world's receipt tally; required for an ask.
	Tally *receiptTally
}

// bookEntry is one admitted order and the reservation it holds.
type bookEntry struct {
	agent   AgentID
	account int
	side    Side
	price   Price
	qty     Quantity
	lot     LotKey
	// slot is the position of the order in Sums for the current clearing:
	// bids first, then asks.
	slot int
}

// clearScratch is the working memory of one market step, owned by the world
// and reused by every step so that a warmed step allocates nothing. It is used
// only inside admit (lots) and clearMarket, so one scratch serves every book of
// a world; nothing in it outlives the call.
type clearScratch struct {
	ordB, ordA []Order
	bids, asks []clearing.Offer
	fills      []clearing.Fill
	matches    []clearing.Match
	lots       []ledger.Lot
	sums       []int64
	index      []int32
}

// resize returns s with length n, allocating only when the capacity is short.
func resize[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	return s[:n]
}

// book is the admitted orders of one market for one tick (D39: one region and
// one commodity). It holds the orders in admission order and the reservations
// they hold until clearMarket releases them.
type book struct {
	region    RegionID
	commodity CommodityID
	tick      int64
	sc        *clearScratch
	entries   []bookEntry
	// fill is the units filled per entry, in admission order, set by
	// clearMarket and kept until the next one (the outcomes of spec-clearing.md
	// 7.3 are delivered after every market has cleared).
	fill []int64
	// invalid is returned, by pointer, by a failed clearMarket so that the
	// failure allocates nothing; it is overwritten by the next failure.
	invalid tradeListError
}

// newBook returns an empty book of the market (region, commodity) of a tick.
func newBook(region RegionID, commodity CommodityID, tick int64, sc *clearScratch) *book {
	return &book{region: region, commodity: commodity, tick: tick, sc: sc}
}

// reset empties the book for a new tick and keeps its storage. It must be
// called only when the reservations of the last tick are released, which
// clearMarket does.
func (b *book) reset(tick int64) {
	b.tick = tick
	b.entries = b.entries[:0]
	b.fill = b.fill[:0]
}

// filled returns the units filled of each admitted order, in admission order,
// as the last clearMarket left them (all 0 after a failed one). The slice is
// the book's and is valid until the book is reset.
func (b *book) filled() []int64 { return b.fill }

// invariant panics with a message that says the programmer violated an
// invariant (architecture.md 8.1). Every call guards a condition that
// admission or the trade-list check has already ruled out (D33, D34), so the
// branches that call it are unreachable through any input and are left out of
// the coverage count on purpose (testing.md 9.5).
func invariant(what string) {
	panic("economysim: violated programmer invariant: " + what)
}

// stock reads the units of the book's commodity an account holds and has
// earmarked and, for a named lot, the units of that lot not earmarked (0 when
// the lot is absent). It uses the scratch's lot slice.
func (b *book) stock(l *ledger.Ledger, acct int, lot LotKey) (held, reserved, lotAvail int64) {
	lots, err := l.AppendLots(b.sc.lots[:0], acct)
	if err != nil {
		invariant("the ledger account of a known agent is unreadable: " + err.Error())
	}
	b.sc.lots = lots
	for _, lt := range lots {
		if lt.Commodity != int(b.commodity) {
			continue
		}
		held += lt.Quantity
		reserved += lt.Reserved
		if lot.Named && lt.Born == lot.Born && lt.Durability == lot.Durability {
			lotAvail = lt.Quantity - lt.Reserved
		}
	}
	return held, reserved, lotAvail
}

// admit runs the admission of spec-clearing.md 3.2 for one order and, if it
// passes, reserves its cash or stock and appends it to the book. It returns
// the reason of the first failing check and false, or "" and true. The caller
// emits the event (emitPlaced, emitRejected). An order of another commodity
// than the book's panics: the world routes by commodity (D39). A bid names no
// lot; the Lot of a bid is ignored. An agent that places an ordinary ask and a
// named-lot ask of one commodity in one book is also a violated invariant, as
// only an estate names a lot and it names one always (spec-clearing.md 3.1.2).
func (b *book) admit(l *ledger.Ledger, f orderFacts, o Order) (Reason, bool) {
	if o.Commodity != b.commodity {
		invariant("an order of another commodity reached the book of a market")
	}
	if o.Side != Bid && o.Side != Ask {
		invariant("an order has a side that is neither bid nor ask")
	}
	bid := o.Side == Bid
	if bid {
		o.Lot = LotKey{}
	}
	cf := clearing.Facts{
		AgentKnown: f.AgentKnown, SideAllowed: f.SideAllowed,
		CommodityKnown: f.CommodityKnown, Tradable: f.Tradable,
		Bid: bid, Price: int64(o.Price), Qty: int64(o.Quantity),
		MaxPrice: MaxPrice, MaxQuantity: MaxQuantity, MaxMoney: MaxMoney,
		Duplicate: b.duplicate(o),
	}
	if f.AgentKnown {
		b.readFacts(l, f, o, &cf)
	}
	if code := clearing.Admit(cf); code != clearing.Admitted {
		r, ok := reasonOfCode(code)
		if !ok {
			invariant("admission returned a code with no reason")
		}
		return r, false
	}
	// The check passed, so none of the products below can fail.
	if r, ok := b.reserve(l, f, o, cf.Tally); !ok {
		return r, false
	}
	b.entries = append(b.entries, bookEntry{
		agent: o.Agent, account: f.Account, side: o.Side,
		price: o.Price, qty: o.Quantity, lot: o.Lot,
	})
	return "", true
}

// duplicate reports whether the book holds an order of the same (agent, lot,
// side). An agent that mixes an ordinary and a named-lot ask panics.
func (b *book) duplicate(o Order) bool {
	for i := range b.entries { // shortcut: linear scan, O(n^2) a market; index by (agent, lot, side) if a book passes a few thousand orders
		e := &b.entries[i]
		if e.agent != o.Agent || e.side != o.Side {
			continue
		}
		if e.lot == o.Lot {
			return true
		}
		if e.lot.Named != o.Lot.Named {
			// The ledger earmarks an ordinary ask's units by commodity, oldest
			// lot first, so they could be those a named ask earmarked.
			invariant("an agent mixes an ordinary ask and a named-lot ask of one commodity in one book")
		}
	}
	return false
}

// readFacts fills the ledger-derived facts of cf for a known agent.
func (b *book) readFacts(l *ledger.Ledger, f orderFacts, o Order, cf *clearing.Facts) {
	v, err := l.Account(f.Account)
	if err != nil {
		invariant("the ledger account of a known agent is unreadable: " + err.Error())
	}
	held, reserved, lotAvail := b.stock(l, f.Account, o.Lot)
	// A balance or a holding below 0 is a ledger bug. Admission would refuse it
	// as an ordinary rejection and hide it among the rejections of a market
	// under pressure, so it stops here, loudly (spec-clearing.md 3.2.2).
	if v.Balance < 0 || held < 0 {
		invariant("a known agent's ledger account holds a negative balance or stock")
	}
	if o.Side == Bid {
		cf.Available, cf.Held = v.Balance-v.Reserved, held
		return
	}
	if f.Tally == nil {
		invariant("an ask was admitted with no receipt tally")
	}
	cf.Balance, cf.Tally = v.Balance, f.Tally.of(f.Account)
	if cf.Tally < 0 {
		invariant("a receipt tally is negative")
	}
	cf.Available = held - reserved
	if o.Lot.Named {
		cf.Available = lotAvail
	}
}

// reserve reserves the cash of an admitted bid or the goods of an admitted ask
// and, for an ask, adds its worst-case receipt to the account's tally. It
// returns ReasonReceiptLimit and false if the tally does not fit int64, which
// admission's own test of the receipt against MaxMoney should already have
// refused.
func (b *book) reserve(l *ledger.Ledger, f orderFacts, o Order, tally int64) (Reason, bool) {
	if o.Side == Bid {
		cost, ok := checked.Mul(int64(o.Price), int64(o.Quantity))
		if !ok {
			invariant("a bid that passed admission has a product that overflows")
		}
		if err := l.ReserveMoney(f.Account, cost); err != nil {
			invariant("the ledger refused the reservation of an admitted bid: " + err.Error())
		}
		return "", true
	}
	worst, ok := checked.Mul(int64(o.Quantity), MaxPrice)
	sum, ok2 := checked.Add(tally, worst)
	if !ok || !ok2 {
		return ReasonReceiptLimit, false
	}
	var err error
	if o.Lot.Named {
		err = l.ReserveLot(f.Account, ledger.Key{Commodity: int(b.commodity), Born: o.Lot.Born, Durability: o.Lot.Durability}, int64(o.Quantity))
	} else {
		err = l.ReserveGoods(f.Account, int(b.commodity), int64(o.Quantity))
	}
	if err != nil {
		invariant("the ledger refused the reservation of an admitted ask: " + err.Error())
	}
	f.Tally.set(f.Account, sum)
	return "", true
}

// emitPlaced delivers OrderPlaced for an admitted order, when an observer was
// supplied (spec-clearing.md 9.1).
func (b *book) emitPlaced(obs *observers, o Order) {
	if !obs.supplied {
		return
	}
	obs.deliver(OrderPlaced{
		Tick: b.tick, Region: b.region, Commodity: o.Commodity, Agent: o.Agent,
		Side: o.Side, Limit: o.Price, Quantity: o.Quantity,
	})
}

// emitRejected delivers OrderRejected for a rejected order, when an observer
// was supplied, and writes the Debug line "order rejected", when the logger
// has Debug enabled (spec-clearing.md 9.2).
func (b *book) emitRejected(obs *observers, log *slog.Logger, o Order, r Reason) {
	if obs.supplied {
		obs.deliver(OrderRejected{
			Tick: b.tick, Region: b.region, Commodity: o.Commodity, Agent: o.Agent,
			Side: o.Side, Reason: r,
		})
	}
	if ctx := context.Background(); log != nil && log.Enabled(ctx, slog.LevelDebug) {
		log.LogAttrs(ctx, slog.LevelDebug, "order rejected",
			slog.Int64(logkeys.Tick, b.tick),
			slog.Int(logkeys.Region, int(b.region)),
			slog.Int(logkeys.Commodity, int(o.Commodity)),
			slog.Int64(logkeys.Agent, int64(o.Agent)),
			slog.String(logkeys.Side, o.Side.String()),
			slog.String(logkeys.Reason, string(r)),
		)
	}
}

// load splits the book into the two sides, in admission order, for the
// Clearing (orders) and for the check (offers, whose Ref is the entry index),
// and assigns each entry its slot in Sums: bids first, then asks.
func (sc *clearScratch) load(b *book) {
	sc.ordB, sc.ordA = sc.ordB[:0], sc.ordA[:0]
	sc.bids, sc.asks = sc.bids[:0], sc.asks[:0]
	for i := range b.entries {
		e := &b.entries[i]
		o := Order{Agent: e.agent, Commodity: b.commodity, Side: e.side, Price: e.price, Quantity: e.qty, Lot: e.lot}
		of := clearing.Offer{
			Ref: i, Price: int64(e.price), Qty: int64(e.qty), Owner: int64(e.agent),
			Lot: clearing.Lot{Named: e.lot.Named, Born: e.lot.Born, Durability: e.lot.Durability},
		}
		if e.side == Bid {
			e.slot = len(sc.bids)
			sc.ordB, sc.bids = append(sc.ordB, o), append(sc.bids, of)
		} else {
			e.slot = len(sc.asks)
			sc.ordA, sc.asks = append(sc.ordA, o), append(sc.asks, of)
		}
	}
	nb := len(sc.bids)
	for i := range b.entries {
		if e := &b.entries[i]; e.side != Bid {
			e.slot += nb
		}
	}
}

// release returns the remainder of every reservation of the book: the money of
// a bid not spent and the units of an ask not sold, given the units filled per
// slot (nil: nothing was filled). A remainder of 0 is not released, because
// the ledger refuses an amount below 1 (D33).
func (b *book) release(l *ledger.Ledger, sums []int64) {
	for i := range b.entries {
		e := &b.entries[i]
		var filled int64
		if sums != nil {
			filled = sums[e.slot]
		}
		rem := int64(e.qty) - filled
		if rem < 1 {
			continue
		}
		var err error
		switch {
		case e.side == Bid:
			// limit * remaining <= limit * quantity, which fit at admission.
			money, ok := checked.Mul(int64(e.price), rem)
			if !ok {
				invariant("the remainder of an admitted bid has a product that overflows")
			}
			err = l.ReleaseMoney(e.account, money)
		case e.lot.Named:
			err = l.ReleaseLot(e.account, ledger.Key{Commodity: int(b.commodity), Born: e.lot.Born, Durability: e.lot.Durability}, rem)
		default:
			err = l.ReleaseGoods(e.account, int(b.commodity), rem)
		}
		if err != nil {
			invariant("the ledger refused the release of a reservation the book holds: " + err.Error())
		}
	}
}

// tradeListError is the error of a list of trades that fails the check of
// spec-clearing.md 8.2. It wraps ErrInvalidTrade and names the market.
type tradeListError struct {
	tick      int64
	region    RegionID
	commodity CommodityID
	what      string
}

// Error describes the market and the rule broken.
func (e *tradeListError) Error() string {
	return fmt.Sprintf("%s: tick %d region %d commodity %d: %s",
		ErrInvalidTrade.Error(), e.tick, e.region, e.commodity, e.what)
}

// Unwrap returns ErrInvalidTrade.
func (e *tradeListError) Unwrap() error { return ErrInvalidTrade }

// listFailure describes a failure code of clearing.Check.
func listFailure(c clearing.Code) string {
	switch c {
	case clearing.BadCommodity:
		return "a trade names another commodity than the market's"
	case clearing.NoOrder:
		return "a trade names a buyer with no bid or a seller with no ask"
	case clearing.BadQuantity:
		return "a trade has a quantity below 1"
	case clearing.BadPrice:
		return "a trade price is outside the limits of its orders"
	case clearing.SameOwner:
		return "a trade has one agent as buyer and seller"
	case clearing.OverFilled:
		return "the trades of an order exceed its quantity"
	}
	return "unknown failure" // a Code Check does not return; kept so a new code is not silent
}

// mustSettle panics when the ledger refuses a step of settlement. The list
// passed the check and admission closed the overflow cases (D33, D34), so a
// refusal is a violated invariant, never an input.
func mustSettle(err error, step string) {
	if err != nil {
		invariant("the ledger refused " + step + " of a checked trade: " + err.Error())
	}
}

// clearMarket clears one market: it hands the admitted orders to c, checks the
// whole list it returns, and only then settles each trade, emits its event,
// releases the remainder of every reservation and reports the market
// (spec-clearing.md 6, 8.2, 9). The settled trades are appended to trades.
//
// On an invalid list nothing is settled or emitted, every reservation is
// released and the error wraps ErrInvalidTrade; it is not logged, the caller
// returns it (architecture.md 8.3). The error is the book's own value and is
// overwritten by the book's next failure. The units filled per order are
// b.filled() afterwards. tick must be the tick of the book.
//
// The per-trade money flow is an int64 sum at settlement and cannot overflow
// unless total money passes int64, which MaxMoney per balance and the audit
// exclude, so no check is made.
func clearMarket(l *ledger.Ledger, b *book, c Clearing, rng *rand.Rand, prev marketReport,
	tick int64, obs *observers, log *slog.Logger, trades []Trade) (marketReport, []Trade, error) {
	if tick != b.tick {
		invariant("clearMarket was given a tick other than the book's")
	}
	sc, n := b.sc, len(b.entries)
	sc.load(b)
	listed := c.Clear(sc.ordB, sc.ordA, rng)
	sc.fills = sc.fills[:0]
	for _, t := range listed {
		sc.fills = append(sc.fills, clearing.Fill{
			Commodity: int64(t.Commodity), Buyer: int64(t.Buyer), Seller: int64(t.Seller),
			Lot:   clearing.Lot{Named: t.Lot.Named, Born: t.Lot.Born, Durability: t.Lot.Durability},
			Price: int64(t.Price), Qty: int64(t.Quantity),
		})
	}
	sc.sums, sc.index = resize(sc.sums, n), resize(sc.index, n)
	ms, code := clearing.Check(sc.matches[:0], clearing.Scratch{Sums: sc.sums, Index: sc.index},
		int64(b.commodity), sc.bids, sc.asks, sc.fills)
	sc.matches = ms
	b.fill = resize(b.fill, n)
	if code != 0 {
		clear(b.fill)
		b.release(l, nil)
		b.invalid = tradeListError{tick: tick, region: b.region, commodity: b.commodity, what: listFailure(code)}
		return prev, trades, &b.invalid
	}
	for _, m := range ms {
		trades = b.settle(l, m, tick, obs, trades)
	}
	b.release(l, sc.sums)
	for i := range b.entries {
		b.fill[i] = sc.sums[b.entries[i].slot]
	}
	r, ok := clearing.Report(sc.bids, sc.asks, sc.sums, ms, int64(prev.Price), prev.Age)
	if !ok {
		invariant("the market report of a checked list does not fit int64")
	}
	rep := marketReport{
		Price: Price(r.Price), Age: r.Age, Volume: Quantity(r.Volume),
		UnmatchedBid: Quantity(r.UnmatchedBids), UnmatchedAsk: Quantity(r.UnmatchedAsks),
		BestBid: Price(r.BestBid), BestAsk: Price(r.BestAsk),
		HasBestBid: r.HasBestBid, HasBestAsk: r.HasBestAsk,
	}
	if obs.supplied {
		obs.deliver(MarketCleared{
			Tick: tick, Region: b.region, Commodity: b.commodity, Price: rep.Price,
			PriceAge: rep.Age, Volume: rep.Volume, UnmatchedBid: rep.UnmatchedBid,
			UnmatchedAsk: rep.UnmatchedAsk, BestBid: rep.BestBid, BestAsk: rep.BestAsk,
			HasBestBid: rep.HasBestBid, HasBestAsk: rep.HasBestAsk,
		})
	}
	if ctx := context.Background(); log != nil && log.Enabled(ctx, slog.LevelDebug) {
		log.LogAttrs(ctx, slog.LevelDebug, "market cleared",
			slog.Int64(logkeys.Tick, tick),
			slog.Int(logkeys.Region, int(b.region)),
			slog.Int(logkeys.Commodity, int(b.commodity)),
			slog.Int64(logkeys.Price, int64(rep.Price)),
			slog.Int64(logkeys.Quantity, int64(rep.Volume)),
		)
	}
	return rep, trades, nil
}

// settle moves the money and goods of one checked match and emits its event
// (spec-clearing.md 6, D33). The goods are those of the book's commodity, and
// the lot is the ask's as admitted.
func (b *book) settle(l *ledger.Ledger, m clearing.Match, tick int64, obs *observers, trades []Trade) []Trade {
	bid, ask := &b.entries[m.BidRef], &b.entries[m.AskRef]
	pay, ok := checked.Mul(m.Price, m.Qty)
	if !ok {
		invariant("the value of a checked trade overflows")
	}
	mustSettle(l.TransferReserved(bid.account, ask.account, pay, ledger.Trade), "the payment")
	if refund := (int64(bid.price) - m.Price) * m.Qty; refund > 0 { // <= limit * qty, fits
		mustSettle(l.ReleaseMoney(bid.account, refund), "the refund")
	}
	if ask.lot.Named {
		key := ledger.Key{Commodity: int(b.commodity), Born: ask.lot.Born, Durability: ask.lot.Durability}
		mustSettle(l.MoveLotReserved(ask.account, bid.account, key, m.Qty, ledger.Trade), "the delivery of a lot")
	} else {
		mustSettle(l.MoveReserved(ask.account, bid.account, int(b.commodity), m.Qty, ledger.Trade), "the delivery")
	}
	t := Trade{
		Commodity: b.commodity, Buyer: bid.agent, Seller: ask.agent,
		Price: Price(m.Price), Quantity: Quantity(m.Qty), Lot: ask.lot,
	}
	if obs.supplied {
		obs.deliver(TradeExecuted{
			Tick: tick, Region: b.region, Commodity: t.Commodity, Buyer: t.Buyer,
			Seller: t.Seller, Price: t.Price, Quantity: t.Quantity,
		})
	}
	return append(trades, t)
}
