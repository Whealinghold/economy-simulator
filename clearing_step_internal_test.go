package economysim

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/clearing"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// TestClearMarket_ReleasesNothingAtZero verifies that a release of amount 0 is
// never made (the ledger refuses an amount below 1): a bid that trades at its
// own limit (10 against 10, refund (10-10)*2 = 0), a fully filled bid, ask and
// named-lot ask do not panic, and a partly filled bid releases exactly its
// remainder.
//
// Requirement: FR-CLR-008, FR-CLR-016
func TestClearMarket_ReleasesNothingAtZero(t *testing.T) {
	t.Parallel()
	t.Run("bid at its own limit, all filled", func(t *testing.T) {
		t.Parallel()
		m := newMkMarket(t, 2)
		m.fund(0, 100)
		m.stock(1, 0, 5)
		b := newBook(0, 0, 0, &m.sc)
		m.admitAll(b, m.bid(0, 0, 10, 2), m.ask(1, 0, 10, 2))
		_, tr, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(1, 0, 0, 0), seedReport(1), 0, &quiet, discardLogger(), nil)
		if err != nil || len(tr) != 1 || tr[0].Price != 10 {
			t.Fatalf("clearMarket = %v, %v, want one trade at 10", tr, err)
		}
		if bal, res := m.money(0); bal != 80 || res != 0 {
			t.Errorf("buyer balance %d reserved %d, want 80 and 0", bal, res)
		}
		if err := m.l.Audit(); err != nil {
			t.Errorf("Audit() = %v, want nil", err)
		}
		if m.l.HasReservation() {
			t.Errorf("HasReservation() = true after the market step, want false")
		}
	})
	t.Run("partly filled bid releases its remainder", func(t *testing.T) {
		t.Parallel()
		m := newMkMarket(t, 2)
		m.fund(0, 100)
		m.stock(1, 0, 5)
		b := newBook(0, 0, 0, &m.sc)
		m.admitAll(b, m.bid(0, 0, 10, 3), m.ask(1, 0, 10, 1))
		if _, res := m.money(0); res != 30 {
			t.Fatalf("reserved %d, want 30 (10*3)", res)
		}
		_, _, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(1, 0, 0, 0), seedReport(1), 0, &quiet, discardLogger(), nil)
		if err != nil {
			t.Fatalf("clearMarket: %v", err)
		}
		// One unit at 10 spent; 2 units * 10 = 20 of the 30 released.
		if bal, res := m.money(0); bal != 90 || res != 0 {
			t.Errorf("buyer balance %d reserved %d, want 90 and 0", bal, res)
		}
		if err := m.l.Audit(); err != nil {
			t.Errorf("Audit() = %v, want nil", err)
		}
		if m.l.HasReservation() {
			t.Errorf("HasReservation() = true after the market step, want false")
		}
	})
	t.Run("named-lot ask fully filled", func(t *testing.T) {
		t.Parallel()
		m := newMkMarket(t, 2)
		m.fund(0, 100)
		m.lot(1, 3, 2)
		b := newBook(0, 1, 0, &m.sc)
		m.admitAll(b, m.bid(0, 1, 7, 2), m.askLot(1, 1, 7, 2, 3))
		_, tr, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(1, 0, 0, 1), seedReport(1), 0, &quiet, discardLogger(), nil)
		if err != nil || len(tr) != 1 {
			t.Fatalf("clearMarket = %v, %v, want one trade", tr, err)
		}
		if err := m.l.Audit(); err != nil {
			t.Errorf("Audit() = %v, want nil", err)
		}
		if m.l.HasReservation() {
			t.Errorf("HasReservation() = true after the market step, want false")
		}
	})
	t.Run("partly filled ask releases its units", func(t *testing.T) {
		t.Parallel()
		m := newMkMarket(t, 2)
		m.fund(0, 100)
		m.stock(1, 0, 5)
		b := newBook(0, 0, 0, &m.sc)
		m.admitAll(b, m.bid(0, 0, 10, 1), m.ask(1, 0, 4, 3))
		if _, _, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(1, 0, 0, 0), seedReport(1), 0, &quiet, discardLogger(), nil); err != nil {
			t.Fatalf("clearMarket: %v", err)
		}
		if held, res := m.units(1, 0); held != 4 || res != 0 {
			t.Errorf("seller holds %d earmarked %d, want 4 and 0", held, res)
		}
		if err := m.l.Audit(); err != nil {
			t.Errorf("Audit() = %v, want nil", err)
		}
		if m.l.HasReservation() {
			t.Errorf("HasReservation() = true after the market step, want false")
		}
	})
}

// TestClearMarket_NamedLot verifies that an ask that names a lot reserves and
// moves exactly that lot and no other of the seller's, and that an ask naming
// a lot the seller does not hold or holds too few units of is
// insufficient_stock and reserves nothing.
//
// Requirement: FR-CLR-029, FR-CLR-006
func TestClearMarket_NamedLot(t *testing.T) {
	t.Parallel()
	m := newMkMarket(t, 3)
	m.fund(0, 1000)
	m.lot(1, 3, 5)
	m.lot(1, 5, 5)
	b := newBook(0, 1, 0, &m.sc)
	got := m.admitAll(b,
		m.bid(0, 1, 10, 2),
		m.askLot(1, 1, 8, 2, 3),
	)
	if !slices.Equal(got, []Reason{"", ""}) {
		t.Fatalf("admission = %v, want both admitted", got)
	}
	// Dur 3 holds 5: 2 are earmarked; dur 5 none.
	lots, _ := m.l.AppendLots(nil, m.acct[1])
	for _, lt := range lots {
		want := int64(0)
		if lt.Durability == 3 {
			want = 2
		}
		if lt.Reserved != want {
			t.Errorf("lot of durability %d earmarked %d, want %d", lt.Durability, lt.Reserved, want)
		}
	}
	// Another agent holding no lot of durability 2, and one holding too few.
	m.lot(2, 4, 1)
	before := m.state()
	tests := []struct {
		name string
		o    Order
	}{
		{"lot not held", m.askLot(2, 1, 8, 1, 2)},
		{"too few units of the lot", m.askLot(2, 1, 8, 2, 4)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, ok := b.admit(m.l, m.facts(int(tt.o.Agent)-10), tt.o)
			if ok || r != ReasonInsufficientStock {
				t.Errorf("admit(%+v) = %q, %v, want insufficient_stock, false", tt.o, r, ok)
			}
		})
	}
	if !reflect.DeepEqual(before, m.state()) {
		t.Errorf("a rejected named ask changed the ledger")
	}
	_, tr, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(1, 0, 0, 1), seedReport(1), 0, &quiet, discardLogger(), nil)
	if err != nil || len(tr) != 1 || tr[0].Lot != (LotKey{Named: true, Durability: 3}) {
		t.Fatalf("clearMarket = %v, %v, want one trade of the lot of durability 3", tr, err)
	}
	// Seller: dur 3 has 3 left, dur 5 untouched at 5; buyer holds 2 of dur 3.
	for _, c := range []struct {
		acct int
		dur  int64
		want int64
	}{{1, 3, 3}, {1, 5, 5}, {0, 3, 2}, {0, 5, 0}} {
		lots, _ := m.l.AppendLots(nil, m.acct[c.acct])
		var q int64
		for _, lt := range lots {
			if lt.Commodity == 1 && lt.Durability == c.dur {
				q = lt.Quantity
			}
		}
		if q != c.want {
			t.Errorf("agent %d holds %d units of the lot of durability %d, want %d", c.acct, q, c.dur, c.want)
		}
	}
	if err := m.l.Audit(); err != nil {
		t.Errorf("Audit() = %v, want nil", err)
	}
	if m.l.HasReservation() {
		t.Errorf("HasReservation() = true after the market step, want false")
	}
}

// faultyTrades are the six breaks of spec-clearing.md 8.2 against the book
// built by faultyBook: bids by agents 0 (10x3), 1 (8x2) and 4 (3x1), asks by 2
// (5x2), 3 (6x2) and 4 (3x1).
func faultyTrades(m *mkMarket) map[string]Trade {
	return map[string]Trade{
		"another commodity": {Commodity: 1, Buyer: m.agent(0), Seller: m.agent(2), Price: 7, Quantity: 1},
		"no bid held":       {Commodity: 0, Buyer: m.agent(2), Seller: m.agent(3), Price: 7, Quantity: 1},
		"quantity zero":     {Commodity: 0, Buyer: m.agent(0), Seller: m.agent(2), Price: 7, Quantity: 0},
		"price above bid":   {Commodity: 0, Buyer: m.agent(0), Seller: m.agent(2), Price: 11, Quantity: 1},
		"price below ask":   {Commodity: 0, Buyer: m.agent(0), Seller: m.agent(2), Price: 4, Quantity: 1},
		"same agent":        {Commodity: 0, Buyer: m.agent(4), Seller: m.agent(4), Price: 3, Quantity: 1},
		"over the quantity": {Commodity: 0, Buyer: m.agent(0), Seller: m.agent(2), Price: 7, Quantity: 3},
	}
}

func faultyBook(t testing.TB) (*mkMarket, []Order) {
	m := newMkMarket(t, 5)
	m.fund(0, 1000)
	m.fund(1, 1000)
	m.fund(4, 1000)
	m.stock(2, 0, 10)
	m.stock(3, 0, 10)
	m.stock(4, 0, 10)
	return m, []Order{
		m.bid(0, 0, 10, 3), m.bid(1, 0, 8, 2), m.bid(4, 0, 3, 1),
		m.ask(2, 0, 5, 2), m.ask(3, 0, 6, 2), m.ask(4, 0, 3, 1),
	}
}

// TestClearMarket_FaultyClearing verifies that a Clearing that breaks any rule
// of spec-clearing.md 8.2 gets ErrInvalidTrade: nothing is settled, every
// reservation is released and the ledger equals its state before the market
// step, and no trade or event is produced.
//
// Requirement: FR-CLR-014, FR-CLR-018, FR-CLR-024
func TestClearMarket_FaultyClearing(t *testing.T) {
	t.Parallel()
	names := []string{"another commodity", "no bid held", "quantity zero", "price above bid", "price below ask", "same agent", "over the quantity"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m, orders := faultyBook(t)
			before := m.state()
			b := newBook(0, 0, 0, &m.sc)
			if got := m.admitAll(b, orders...); !slices.Equal(got, make([]Reason, len(orders))) {
				t.Fatalf("admission = %v, want all admitted", got)
			}
			bad := faultyTrades(m)[name]
			// A valid trade ahead of the bad one: the whole list is refused.
			good := Trade{Commodity: 0, Buyer: m.agent(0), Seller: m.agent(2), Price: 7, Quantity: 1}
			rec := &mkRecorder{}
			prev := seedReport(42)
			rep, tr, err := clearMarket(m.l, b, fakeClearing{[]Trade{good, bad}}, rand.New(rand.NewPCG(1, 1)), prev, 0, rec.observers(), discardLogger(), nil)
			if !errors.Is(err, ErrInvalidTrade) {
				t.Fatalf("clearMarket error = %v, want ErrInvalidTrade", err)
			}
			if len(tr) != 0 || rep != prev || len(rec.events) != 0 {
				t.Errorf("trades %v report %+v events %v, want none, the previous report and no event", tr, rep, rec.events)
			}
			if after := m.state(); !reflect.DeepEqual(before, after) {
				t.Errorf("the ledger differs from its state before the step:\nbefore %+v\nafter  %+v", before, after)
			}
			if !slices.Equal(b.filled(), make([]int64, len(orders))) {
				t.Errorf("filled() = %v, want zeros after a failed step", b.filled())
			}
			if !strings.Contains(err.Error(), "tick 0 region 0 commodity 0") {
				t.Errorf("error %q does not name the tick, region and commodity", err)
			}
			if err := m.l.Audit(); err != nil {
				t.Errorf("Audit() = %v, want nil", err)
			}
			if m.l.HasReservation() {
				t.Errorf("HasReservation() = true after the market step, want false")
			}
		})
	}
}

// TestClearMarket_TradesAndFills verifies that the trades clearMarket appends
// equal the TradeExecuted events in order, that the filled units of the
// admitted orders summed per side equal the units of the trades, and that a
// failed list appends no trade to the caller's slice.
//
// Requirement: FR-CLR-016, FR-CLR-017, FR-CLR-026, FR-CLR-027
func TestClearMarket_TradesAndFills(t *testing.T) {
	t.Parallel()
	m, orders := mkWorked(t)
	b := newBook(0, 0, 0, &m.sc)
	m.admitAll(b, orders...)
	rec := &mkRecorder{}
	pre := []Trade{{Commodity: 9}} // the caller's slice already holds a trade
	_, tr, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(5, 0, 0, 0), seedReport(1), 0, rec.observers(), discardLogger(), pre)
	if err != nil {
		t.Fatalf("clearMarket: %v", err)
	}
	if tr[0] != pre[0] || len(tr) != 4 {
		t.Fatalf("trades = %v, want the caller's trade then 3 appended", tr)
	}
	var evs []Trade
	for _, e := range rec.events[:3] {
		x, ok := e.(TradeExecuted)
		if !ok {
			t.Fatalf("event %T before the market cleared, want TradeExecuted", e)
		}
		evs = append(evs, Trade{Commodity: x.Commodity, Buyer: x.Buyer, Seller: x.Seller, Price: x.Price, Quantity: x.Quantity})
	}
	if !slices.Equal(evs, tr[1:]) {
		t.Errorf("TradeExecuted events = %v, want the trades %v", evs, tr[1:])
	}
	var units, bidFill, askFill int64
	for _, x := range tr[1:] {
		units += int64(x.Quantity)
	}
	for i, f := range b.filled() {
		if i < 3 {
			bidFill += f
		} else {
			askFill += f
		}
	}
	if bidFill != units || askFill != units {
		t.Errorf("filled units bids %d asks %d, want %d on both sides", bidFill, askFill, units)
	}

	// A failed list appends nothing to a caller's slice that already holds a
	// trade, and returns it as it was.
	m2, orders2 := faultyBook(t)
	b2 := newBook(0, 0, 0, &m2.sc)
	m2.admitAll(b2, orders2...)
	pre2 := []Trade{{Commodity: 9}}
	bad := fakeClearing{[]Trade{faultyTrades(m2)["quantity zero"]}}
	_, tr2, err := clearMarket(m2.l, b2, bad, nil, seedReport(1), 0, &quiet, discardLogger(), pre2)
	if !errors.Is(err, ErrInvalidTrade) || !slices.Equal(tr2, pre2) {
		t.Errorf("clearMarket(invalid list, %v) = %v, %v, want the caller's slice unchanged and ErrInvalidTrade", pre2, tr2, err)
	}
}

// TestClearMarket_ZeroAllocations verifies that, with a hand-written Clearing
// that returns a preallocated slice, a caller slice with room and a warmed
// scratch and book, admission and the market step allocate nothing, for a
// valid list and for a rejected one. AverageClearing.Clear is outside this
// count: the Clearing interface returns a fresh slice.
//
// Requirement: NFR-CLR-002
func TestClearMarket_ZeroAllocations(t *testing.T) {
	// Not parallel: AllocsPerRun is process-wide.
	for _, name := range []string{"valid", "invalid"} {
		t.Run(name, func(t *testing.T) {
			m := newMkMarket(t, 4)
			m.fund(0, 1e12)
			m.fund(1, 1e12)
			m.stock(2, 0, 1e6)
			m.stock(3, 0, 1e6)
			list := []Trade{
				{Commodity: 0, Buyer: m.agent(0), Seller: m.agent(2), Price: 8, Quantity: 2},
				{Commodity: 0, Buyer: m.agent(1), Seller: m.agent(3), Price: 7, Quantity: 1},
			}
			if name == "invalid" {
				list[1].Price = 100 // above the bid's limit of 9
			}
			orders := []Order{m.bid(0, 0, 10, 2), m.bid(1, 0, 9, 1), m.ask(2, 0, 6, 2), m.ask(3, 0, 5, 1)}
			facts := make([]orderFacts, len(orders))
			for i, o := range orders {
				facts[i] = m.facts(int(o.Agent) - 10)
			}
			b := newBook(0, 0, 0, &m.sc)
			trades := make([]Trade, 0, 8)
			var c Clearing = fakeClearing{list}
			log := discardLogger()
			step := func() {
				b.reset(0)
				m.tally.reset()
				for i, o := range orders {
					if r, ok := b.admit(m.l, facts[i], o); !ok {
						t.Fatalf("admit(%+v) = %q", o, r)
					}
				}
				_, _, err := clearMarket(m.l, b, c, nil, seedReport(5), 0, &quiet, log, trades[:0])
				if wantInvalid := name == "invalid"; wantInvalid != (err != nil) || (wantInvalid && !errors.Is(err, ErrInvalidTrade)) {
					t.Fatalf("clearMarket(%s list) error = %v, want invalid %v", name, err, wantInvalid)
				}
			}
			step() // warm the scratch, the book and the ledger's lots
			if got := testing.AllocsPerRun(50, step); got != 0 {
				t.Errorf("AllocsPerRun(market step) = %v, want 0", got)
			}
		})
	}
}

// TestEmit_ZeroAllocationsWithDefaults verifies that, with no observer
// supplied and the default logger, the OrderPlaced and OrderRejected call
// sites and their Debug line allocate nothing.
//
// Requirement: NFR-CLR-002, FR-CLR-033
func TestEmit_ZeroAllocationsWithDefaults(t *testing.T) {
	b := newBook(1, 2, 3, &clearScratch{})
	o := Order{Agent: 4, Commodity: 2, Side: Ask, Price: 5, Quantity: 6}
	log := discardLogger()
	if got := testing.AllocsPerRun(100, func() { b.emitPlaced(&quiet, o) }); got != 0 {
		t.Errorf("AllocsPerRun(emitPlaced) = %v, want 0", got)
	}
	if got := testing.AllocsPerRun(100, func() { b.emitRejected(&quiet, log, o, ReasonOverflow) }); got != 0 {
		t.Errorf("AllocsPerRun(emitRejected) = %v, want 0", got)
	}
}

// BenchmarkClearMarket reports the cost and allocations of one market step of
// 100 orders with a preallocated list.
func BenchmarkClearMarket(b *testing.B) {
	m := newMkMarket(b, 100)
	var orders []Order
	for i := range 50 {
		m.fund(i, 1e9)
		m.stock(i+50, 0, 1e6)
		orders = append(orders, m.bid(i, 0, int64(10+i%5), 2), m.ask(i+50, 0, int64(5+i%5), 2))
	}
	facts := make([]orderFacts, len(orders))
	for i, o := range orders {
		facts[i] = m.facts(int(o.Agent) - 10)
	}
	bk := newBook(0, 0, 0, &m.sc)
	list := []Trade{{Commodity: 0, Buyer: m.agent(0), Seller: m.agent(50), Price: 8, Quantity: 2}}
	var c Clearing = fakeClearing{list}
	trades := make([]Trade, 0, 4)
	b.ReportAllocs()
	for b.Loop() {
		bk.reset(0)
		m.tally.reset()
		for i, o := range orders {
			bk.admit(m.l, facts[i], o)
		}
		clearMarket(m.l, bk, c, nil, seedReport(5), 0, &quiet, discardLogger(), trades[:0])
	}
}

// TestClearMarket_OrderOfMarkets verifies that two markets cleared in either
// order against one ledger give the same trades and balances, given the keys
// of each market from its own stream, with one agent in both markets.
//
// Requirement: FR-CLR-023
func TestClearMarket_OrderOfMarkets(t *testing.T) {
	t.Parallel()
	run := func(zeroFirst bool) ([]Trade, ledgerState) {
		m := newMkMarket(t, 6)
		for i := range 3 {
			m.fund(i, 500)
			m.stock(i+3, 0, 20)
			m.stock(i+3, 1, 20)
		}
		b0, b1 := newBook(0, 0, 4, &m.sc), newBook(0, 1, 4, &m.sc)
		// Equal limits, so the tie-break keys decide. Agent 0 buys in both.
		m.admitAll(b0, m.bid(0, 0, 10, 2), m.bid(1, 0, 10, 2), m.ask(3, 0, 10, 2), m.ask(4, 0, 10, 2))
		m.tally.reset()
		m.admitAll(b1, m.bid(0, 1, 10, 3), m.bid(2, 1, 10, 3), m.ask(3, 1, 10, 3), m.ask(5, 1, 10, 3))
		var tr [2][]Trade
		clear := func(i int, b *book) {
			_, got, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(77, 4, 0, int64(i)), seedReport(1), 4, &quiet, discardLogger(), nil)
			if err != nil {
				t.Fatalf("clearMarket(%d): %v", i, err)
			}
			tr[i] = got
		}
		if zeroFirst {
			clear(0, b0)
			clear(1, b1)
		} else {
			clear(1, b1)
			clear(0, b0)
		}
		if err := m.l.Audit(); err != nil {
			t.Errorf("Audit() = %v, want nil", err)
		}
		if m.l.HasReservation() {
			t.Errorf("HasReservation() = true after the market step, want false")
		}
		return append(tr[0], tr[1]...), m.state()
	}
	tradesA, stateA := run(true)
	tradesB, stateB := run(false)
	if !slices.Equal(tradesA, tradesB) || !reflect.DeepEqual(stateA, stateB) {
		t.Errorf("market order changed the result:\ntrades %v vs %v\nstate %+v vs %+v", tradesA, tradesB, stateA, stateB)
	}
}

// TestClearMarket_Events verifies the emission order of a scripted market:
// OrderPlaced and OrderRejected in the order submitted with the admitted limit
// and the reason, then the TradeExecuted events in matching order, then one
// MarketCleared, which is also emitted for a market with no order. No event
// holds a map or a pointer.
//
// Requirement: FR-CLR-026, FR-CLR-027
func TestClearMarket_Events(t *testing.T) {
	t.Parallel()
	m := newMkMarket(t, 3)
	m.fund(0, 100)
	m.stock(1, 0, 5)
	b := newBook(2, 0, 6, &m.sc)
	rec := &mkRecorder{}
	obs := rec.observers()
	submit := func(o Order) {
		r, ok := b.admit(m.l, m.facts(int(o.Agent)-10), o)
		if ok {
			b.emitPlaced(obs, o)
			return
		}
		b.emitRejected(obs, discardLogger(), o, r)
	}
	submit(m.bid(0, 0, 9, 2))
	submit(m.bid(0, 0, 9, 2)) // duplicate
	submit(m.ask(1, 0, 5, 2))
	submit(m.bid(2, 0, 0, 1)) // invalid price
	_, _, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(2, 6, 2, 0), seedReport(3), 6, obs, discardLogger(), nil)
	if err != nil {
		t.Fatalf("clearMarket: %v", err)
	}
	// 9 and 5 meet at (9+5)/2 = 7 for 2 units; nothing is unmatched.
	want := []Event{
		OrderPlaced{Tick: 6, Region: 2, Commodity: 0, Agent: 10, Side: Bid, Limit: 9, Quantity: 2},
		OrderRejected{Tick: 6, Region: 2, Commodity: 0, Agent: 10, Side: Bid, Reason: ReasonDuplicateOrder},
		OrderPlaced{Tick: 6, Region: 2, Commodity: 0, Agent: 11, Side: Ask, Limit: 5, Quantity: 2},
		OrderRejected{Tick: 6, Region: 2, Commodity: 0, Agent: 12, Side: Bid, Reason: ReasonInvalidPrice},
		TradeExecuted{Tick: 6, Region: 2, Commodity: 0, Buyer: 10, Seller: 11, Price: 7, Quantity: 2},
		MarketCleared{Tick: 6, Region: 2, Commodity: 0, Price: 7, PriceAge: 0, Volume: 2},
	}
	if !reflect.DeepEqual(rec.events, want) {
		t.Errorf("events:\n got %+v\nwant %+v", rec.events, want)
	}
	for _, e := range want {
		rt := reflect.TypeOf(e)
		for i := range rt.NumField() {
			switch k := rt.Field(i).Type.Kind(); k {
			case reflect.Map, reflect.Pointer, reflect.Func, reflect.Chan, reflect.Interface:
				t.Errorf("%s.%s is a %v, want no map or pointer in an event", rt.Name(), rt.Field(i).Name, k)
			}
		}
	}
	// An empty market still reports.
	b.reset(7)
	rec.events = nil
	_, _, err = clearMarket(m.l, b, AverageClearing{}, avgRNG(2, 7, 2, 0), seedReport(3), 7, obs, discardLogger(), nil)
	if err != nil {
		t.Fatalf("clearMarket(empty): %v", err)
	}
	wantEmpty := []Event{MarketCleared{Tick: 7, Region: 2, Commodity: 0, Price: 3, PriceAge: 1}}
	if !reflect.DeepEqual(rec.events, wantEmpty) {
		t.Errorf("events of an empty market = %+v, want %+v", rec.events, wantEmpty)
	}
}

// mkLogs is a slog.Handler that records every record, at every level.
type mkLogs struct{ recs *[]slog.Record }

func (mkLogs) Enabled(context.Context, slog.Level) bool { return true }
func (h mkLogs) Handle(_ context.Context, r slog.Record) error {
	*h.recs = append(*h.recs, r)
	return nil
}
func (h mkLogs) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h mkLogs) WithGroup(string) slog.Handler      { return h }

// TestClearMarket_LogsOnlyDebug verifies that a scripted market with a
// rejected order and an invalid list writes only the two Debug lines with
// their constant messages and keys, each carrying its tick and region: no
// Info, Warn or Error.
//
// Requirement: FR-CLR-033
func TestClearMarket_LogsOnlyDebug(t *testing.T) {
	t.Parallel()
	var recs []slog.Record
	log := slog.New(mkLogs{&recs})
	m, orders := faultyBook(t)
	// Region 2 so that a line which logged a default 0 would fail the value
	// check below.
	b := newBook(2, 0, 0, &m.sc)
	for _, o := range append(orders, m.bid(0, 0, 1, 1)) { // the last is a duplicate
		if r, ok := b.admit(m.l, m.facts(int(o.Agent)-10), o); !ok {
			b.emitRejected(&quiet, log, o, r)
		}
	}
	bad := faultyTrades(m)["price above bid"]
	if _, _, err := clearMarket(m.l, b, fakeClearing{[]Trade{bad}}, nil, seedReport(1), 0, &quiet, log, nil); !errors.Is(err, ErrInvalidTrade) {
		t.Fatalf("clearMarket error = %v, want ErrInvalidTrade", err)
	}
	b.reset(1)
	m.admitAll(b, m.bid(0, 0, 10, 1), m.ask(2, 0, 5, 1))
	if _, _, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(1, 1, 0, 0), seedReport(1), 0+1, &quiet, log, nil); err != nil {
		t.Fatalf("clearMarket: %v", err)
	}
	keys := map[string][]string{
		"order rejected": {logkeys.Tick, logkeys.Region, logkeys.Commodity, logkeys.Agent, logkeys.Side, logkeys.Reason},
		"market cleared": {logkeys.Tick, logkeys.Region, logkeys.Commodity, logkeys.Price, logkeys.Quantity},
	}
	seen := map[string]int{}
	for _, r := range recs {
		want, ok := keys[r.Message]
		if r.Level != slog.LevelDebug || !ok {
			t.Errorf("record %q at %v, want only the Debug lines %v", r.Message, r.Level, keys)
			continue
		}
		seen[r.Message]++
		var got []string
		vals := map[string]int64{}
		r.Attrs(func(a slog.Attr) bool {
			got = append(got, a.Key)
			if a.Key == logkeys.Tick || a.Key == logkeys.Region {
				vals[a.Key] = a.Value.Int64()
			}
			return true
		})
		if !slices.Equal(got, want) {
			t.Errorf("%q keys = %v, want %v", r.Message, got, want)
		}
		// The rejection is of the book's tick 0 and the clearing of tick 1; both
		// are region 2. Without them a line cannot be tied to its market.
		wantTick := map[string]int64{"order rejected": 0, "market cleared": 1}[r.Message]
		if vals[logkeys.Tick] != wantTick || vals[logkeys.Region] != 2 {
			t.Errorf("%q tick, region = %d, %d, want %d, 2", r.Message, vals[logkeys.Tick], vals[logkeys.Region], wantTick)
		}
	}
	if seen["order rejected"] != 1 || seen["market cleared"] != 1 {
		t.Errorf("lines seen = %v, want one of each (the invalid list logs nothing)", seen)
	}
}

// TestClearMarket_NonInterference verifies that the same book with and without
// an observer and a logger gives the same ledger, report and trades.
//
// Requirement: FR-CLR-033
func TestClearMarket_NonInterference(t *testing.T) {
	t.Parallel()
	run := func(obs *observers, log *slog.Logger) (marketReport, []Trade, ledgerState) {
		m, orders := mkWorked(t)
		b := newBook(0, 0, 0, &m.sc)
		for _, o := range orders {
			r, ok := b.admit(m.l, m.facts(int(o.Agent)-10), o)
			if !ok {
				b.emitRejected(obs, log, o, r)
			} else {
				b.emitPlaced(obs, o)
			}
		}
		rep, tr, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(8, 0, 0, 0), seedReport(1), 0, obs, log, nil)
		if err != nil {
			t.Fatalf("clearMarket: %v", err)
		}
		return rep, tr, m.state()
	}
	var recs []slog.Record
	rec := &mkRecorder{}
	r1, t1, s1 := run(&quiet, discardLogger())
	r2, t2, s2 := run(rec.observers(), slog.New(mkLogs{&recs}))
	if r1 != r2 || !slices.Equal(t1, t2) || !reflect.DeepEqual(s1, s2) {
		t.Errorf("observation changed the result: %+v %v vs %+v %v", r1, t1, r2, t2)
	}
	if len(rec.events) == 0 || len(recs) == 0 {
		t.Errorf("the observed run saw %d events and %d records, want some", len(rec.events), len(recs))
	}
}

// TestReasonCodes_Guard verifies the mapping of admission codes to rejection
// reasons both ways: every code of internal/clearing has a constant, every
// slice constant of the table of spec-clearing.md 3.2 has a code, and the
// rows owned by a module are the only ones without.
//
// Guard: the root maps a number it does not own to a closed set of strings. A
// code added in internal/clearing with no row here would reach OrderRejected
// as an empty reason, and a constant with no code could never be produced.
func TestReasonCodes_Guard(t *testing.T) {
	t.Parallel()
	for c := clearing.UnknownAgent; c <= clearing.InsufficientStock; c++ {
		t.Run(fmt.Sprintf("code %d", c), func(t *testing.T) {
			r, ok := reasonOfCode(c)
			if !ok {
				t.Fatalf("reasonOfCode(%d) has no reason", c)
			}
			if back, ok := codeOfReason(r); !ok || back != c {
				t.Errorf("codeOfReason(%q) = %d, %v, want %d", r, back, ok, c)
			}
		})
	}
	if r, ok := reasonOfCode(clearing.Admitted); ok {
		t.Errorf("reasonOfCode(Admitted) = %q, want none", r)
	}
	if r, ok := reasonOfCode(clearing.InsufficientStock + 1); ok {
		t.Errorf("reasonOfCode(%d) = %q, want none", clearing.InsufficientStock+1, r)
	}
	if c, ok := codeOfReason(ReasonTrade); ok {
		t.Errorf("codeOfReason(trade) = %d, want none", c)
	}
	if c, ok := codeOfReason(""); ok {
		t.Errorf("codeOfReason(\"\") = %d, want none", c)
	}
	// Every row of the table of 3.2 is a code or a module's deferred row.
	spec := mustRead(t, "doc/specs/spec-clearing.md")
	from := strings.Index(spec, "### 3.2 Validation")
	to := strings.Index(spec, "Checks run in the order of the table")
	if from < 0 || to < from {
		t.Fatalf("table of 3.2 not found in spec-clearing.md")
	}
	deferred := []string{"not_permitted", "no_room"}
	for _, line := range strings.Split(spec[from:to], "\n") {
		var name string
		if _, rest, ok := strings.Cut(line, "| `"); ok {
			name, _, _ = strings.Cut(rest, "`")
		}
		if name == "" || name == "reason" {
			continue
		}
		_, has := codeOfReason(Reason(name))
		if has == slices.Contains(deferred, name) {
			t.Errorf("reason %q: has a code %v, deferred %v, want exactly one", name, has, slices.Contains(deferred, name))
		}
	}
}

// mustRead returns a file of the module, which is the working directory of a
// test of the root package.
func mustRead(t testing.TB, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// TestClearMarket_NamedLotRemainderReleasedOnItsLot verifies that the unfilled
// remainder of a named-lot ask is released on its own lot and not by commodity
// (which would take the units earmarked last, here another lot's): the seller
// has 2 units of the lot of durability 5 earmarked outside the book and asks 3
// of the lot of durability 3, of which 1 sells. Afterwards the lot of
// durability 3 has no earmark and the outside earmark of 2 is untouched.
//
// Requirement: FR-CLR-029, FR-CLR-016
func TestClearMarket_NamedLotRemainderReleasedOnItsLot(t *testing.T) {
	t.Parallel()
	m := newMkMarket(t, 2)
	m.fund(0, 1000)
	m.lot(1, 3, 5)
	m.lot(1, 5, 5)
	if err := m.l.ReserveLot(m.acct[1], ledger.Key{Commodity: 1, Durability: 5}, 2); err != nil {
		t.Fatalf("outside earmark: %v", err)
	}
	b := newBook(0, 1, 0, &m.sc)
	got := m.admitAll(b, m.bid(0, 1, 10, 1), m.askLot(1, 1, 8, 3, 3))
	if !slices.Equal(got, []Reason{"", ""}) {
		t.Fatalf("admission = %v, want both admitted", got)
	}
	if _, tr, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(1, 0, 0, 1), seedReport(1), 0, &quiet, discardLogger(), nil); err != nil || len(tr) != 1 {
		t.Fatalf("clearMarket = %v, %v, want one trade", tr, err)
	}
	lots, _ := m.l.AppendLots(nil, m.acct[1])
	for _, lt := range lots {
		want := int64(0)
		if lt.Durability == 5 {
			want = 2
		}
		if lt.Reserved != want {
			t.Errorf("lot of durability %d earmarked %d, want %d", lt.Durability, lt.Reserved, want)
		}
	}
}

// TestClearMarket_NamedLotAsksRemaindersReleasedPerLot verifies that one
// seller with two named-lot asks, the lot of durability 3 not filled and the
// lot of durability 5 partly filled, ends with each remainder released on its
// own lot. Seller holds 5 units of each lot and
// asks 2 of durability 3 at 9 and 3 of durability 5 at 8; a bid of 2 at 10
// clears against the cheaper ask, so durability 5 sells 2 and keeps 1 earmarked
// until the release, and durability 3 releases all 2. One unit of the lot of
// durability 5 is earmarked outside the book first, so a release by commodity
// (which takes the earmark of the lot named last) is not masked by every
// earmark ending at 0: afterwards that lot keeps exactly this 1 and the lot of
// durability 3 none.
//
// Guard: release must name the lot of a named ask. Releasing by commodity
// takes the units earmarked last, which can be the other lot's, and leaves a
// lot earmarked for an order that no longer exists.
func TestClearMarket_NamedLotAsksRemaindersReleasedPerLot(t *testing.T) {
	t.Parallel()
	m := newMkMarket(t, 2)
	m.fund(0, 1000)
	m.lot(1, 3, 5)
	m.lot(1, 5, 5)
	if err := m.l.ReserveLot(m.acct[1], ledger.Key{Commodity: 1, Durability: 5}, 1); err != nil {
		t.Fatalf("outside earmark: %v", err)
	}
	b := newBook(0, 1, 0, &m.sc)
	got := m.admitAll(b, m.bid(0, 1, 10, 2), m.askLot(1, 1, 9, 2, 3), m.askLot(1, 1, 8, 3, 5))
	if !slices.Equal(got, []Reason{"", "", ""}) {
		t.Fatalf("admission = %v, want all admitted", got)
	}
	_, tr, err := clearMarket(m.l, b, AverageClearing{}, avgRNG(1, 0, 0, 1), seedReport(1), 0, &quiet, discardLogger(), nil)
	if err != nil || len(tr) != 1 || tr[0].Lot != (LotKey{Named: true, Durability: 5}) || tr[0].Quantity != 2 {
		t.Fatalf("clearMarket = %v, %v, want one trade of 2 units of the lot of durability 5", tr, err)
	}
	lots, _ := m.l.AppendLots(nil, m.acct[1])
	held := map[int64]int64{}
	for _, lt := range lots {
		held[lt.Durability] = lt.Quantity
		want := int64(0)
		if lt.Durability == 5 {
			want = 1 // the outside earmark
		}
		if lt.Reserved != want {
			t.Errorf("lot of durability %d earmarked %d, want %d", lt.Durability, lt.Reserved, want)
		}
	}
	if held[3] != 5 || held[5] != 3 {
		t.Errorf("seller holds %v by durability, want 5 of 3 and 3 of 5", held)
	}
}
