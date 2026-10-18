package economysim

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// Commodity indexes used below (packOneCommodities): staple (5) does not wear,
// tool_a (6) wears with a full durability of 10.
const (
	lstStaple = 5
	lstTool   = 6
)

// listFixture is a real ledger, estates and books for the asks, the markdown and
// the checkpoint read. Reference price 100 for every commodity, BeliefStep 2500
// so the floor is ceil(10 000 / 2500) = 4, EstateMarkdown 500.
type listFixture struct {
	*poolFixture
	cfg   *Config
	buyer int
	sc    clearScratch
	tally receiptTally
	tick  int64
}

func newListFixture(t *testing.T, regions int) *listFixture {
	t.Helper()
	cfg := &Config{Commodities: packOneCommodities()}
	for i := range cfg.Commodities {
		cfg.Commodities[i].ReferencePrice = 100
	}
	for r := range regions {
		cfg.Regions = append(cfg.Regions, Region{ID: string(rune('a' + r)), EstateMarkdown: 500})
	}
	cfg.World.BeliefStep = 2500
	f := &listFixture{poolFixture: &poolFixture{l: mustLedger(t, *cfg)}, cfg: cfg}
	e, err := openEstates(f.l, regions)
	if err != nil {
		t.Fatalf("openEstates: %v", err)
	}
	f.e = e
	if f.buyer, err = f.l.Open(ledger.KindAgent, 100, 0); err != nil {
		t.Fatalf("Open: %v", err)
	}
	f.fund(t, f.buyer, 100_000)
	return f
}

// give puts qty units of the lot (commodity, durability) in the estate of r.
func (f *listFixture) give(t *testing.T, r RegionID, commodity int, dur, qty int64) {
	t.Helper()
	src, _ := f.l.Boundary(ledger.Initial, commodity)
	if err := f.l.MoveStated(src, f.e.account(r), ledger.Key{Commodity: commodity, Durability: dur}, qty, ledger.Initial); err != nil {
		t.Fatalf("give: %v", err)
	}
	f.minted[commodity] += qty
}

// threeLots is the estate of region 0 holding the non-durable staple (5 units)
// and two lots of tool_a with remaining durability 3 (2 units) and 6 (4 units).
func (f *listFixture) threeLots(t *testing.T) {
	t.Helper()
	f.give(t, 0, lstStaple, 0, 5)
	f.give(t, 0, lstTool, 3, 2)
	f.give(t, 0, lstTool, 6, 4)
}

// round runs P3.4 to P5.3 for region 0: the estate's asks are admitted through
// the real books, the buyer bids bidQty units at bidLimit in the market of
// bidIn (0 for none), both markets clear and the estate marks down. It returns
// the orders of the estate and the reason of each admission ("" for admitted).
func (f *listFixture) round(t *testing.T, bidIn int, bidLimit, bidQty int64) ([]Order, []Reason) {
	t.Helper()
	f.tick++
	f.tally.reset()
	orders, err := f.e.asks(f.l, f.cfg, 0, nil)
	if err != nil {
		t.Fatalf("asks: %v", err)
	}
	books := map[int]*book{}
	for _, c := range []int{lstStaple, lstTool} {
		books[c] = newBook(0, CommodityID(c), f.tick, &f.sc)
	}
	var reasons []Reason
	for _, o := range orders {
		r, ok := books[int(o.Commodity)].admit(f.l, f.e.facts(0, Ask, &f.tally), o)
		if ok {
			f.e.admitted(0, o)
			r = ""
		}
		reasons = append(reasons, r)
	}
	if bidIn != 0 {
		bid := Order{Agent: 99, Commodity: CommodityID(bidIn), Side: Bid, Price: Price(bidLimit), Quantity: Quantity(bidQty)}
		facts := orderFacts{Account: f.buyer, AgentKnown: true, SideAllowed: true, CommodityKnown: true, Tradable: true, Tally: &f.tally}
		if r, ok := books[bidIn].admit(f.l, facts, bid); !ok {
			t.Fatalf("buyer bid refused: %s", r)
		}
	}
	for _, c := range []int{lstStaple, lstTool} {
		if _, _, err := clearMarket(f.l, books[c], AverageClearing{}, avgRNG(1, f.tick, 0, int64(c)), seedReport(100), &quiet, discardLogger(), nil); err != nil {
			t.Fatalf("clearMarket(%d): %v", c, err)
		}
	}
	if err := f.e.markdown(f.l, f.cfg, 0); err != nil {
		t.Fatalf("markdown: %v", err)
	}
	return orders, reasons
}

// prices lists the listed price of each row of region 0 by key.
func (f *listFixture) prices() map[ledger.Key]Price {
	m := map[ledger.Key]Price{}
	for _, r := range f.e.listed[0] {
		m[r.key] = r.price
	}
	return m
}

func ordersOf(t *testing.T, f *listFixture) []Order {
	t.Helper()
	o, err := f.e.asks(f.l, f.cfg, 0, nil)
	if err != nil {
		t.Fatalf("asks: %v", err)
	}
	return o
}

// TestEstateAsks_PricesAndAdmission verifies the asks of an estate holding
// three lots, admitted by the real books.
//
// Arithmetic: reference 100, floor ceil(10 000 / 2500) = 4. The staple does not
// wear: 100. The tool lots: 100 * 3 / 10 = 30 and 100 * 6 / 10 = 60. The order
// is the key order of the ledger: staple (5) then tool 3 then tool 6. A bid of
// the estate is side_not_allowed; the asks are admitted with a reservation of
// goods, no cash, and the estate keeps 0 money.
//
// Requirement: FR-EST-006, FR-EST-007, FR-EST-008, FR-EST-012
func TestEstateAsks_PricesAndAdmission(t *testing.T) {
	t.Parallel()
	f := newListFixture(t, 1)
	f.threeLots(t)
	got := ordersOf(t, f)
	want := []Order{
		{Agent: 0, Commodity: lstStaple, Side: Ask, Price: 100, Quantity: 5, Lot: LotKey{Named: true}},
		{Agent: 0, Commodity: lstTool, Side: Ask, Price: 30, Quantity: 2, Lot: LotKey{Named: true, Durability: 3}},
		{Agent: 0, Commodity: lstTool, Side: Ask, Price: 60, Quantity: 4, Lot: LotKey{Named: true, Durability: 6}},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("asks = %+v, want %+v", got, want)
	}
	b := newBook(0, lstTool, 1, &f.sc)
	bid := Order{Agent: 0, Commodity: lstTool, Side: Bid, Price: 10, Quantity: 1}
	if r, ok := b.admit(f.l, f.e.facts(0, Bid, &f.tally), bid); ok || r != ReasonSideNotAllowed {
		t.Errorf("estate bid admit = %q, %v, want %q, false", r, ok, ReasonSideNotAllowed)
	}
	for _, o := range want[1:] {
		if r, ok := b.admit(f.l, f.e.facts(0, Ask, &f.tally), o); !ok {
			t.Fatalf("ask %+v refused: %s", o, r)
		}
	}
	v, _ := f.l.Account(f.e.account(0))
	if v.Balance != 0 || v.Reserved != 0 || f.e.pool(f.l, 0) != 0 {
		t.Errorf("estate account %+v after admission, want no money and no cash reserved", v)
	}
	var reserved int64
	for _, lt := range f.lots(t, f.e.account(0)) {
		reserved += lt.Reserved
	}
	if reserved != 6 {
		t.Errorf("goods reserved %d, want 2 + 4 = 6", reserved)
	}
}

// TestEstateRound_Markdown verifies the tick of an estate through the real
// books: a trade pays the pool by trade, an unsold lot falls at P5.3, a wholly
// sold lot has no row afterwards and a partly sold lot falls.
//
// Arithmetic: asks 100, 30, 60 (see TestEstateAsks_PricesAndAdmission),
// markdown 500 basis points. Unsold: 100 -> 100 - trunc(100 * 500 / 10 000) =
// 95; 60 -> 60 - 3 = 57. A bid of 40 for 2 tools meets the ask 30 (the lowest)
// at the midpoint (30 + 40) / 2 = 35: the pool gets 2 * 35 = 70, and the lot
// is sold wholly. A bid of 40 for 1 tool leaves 1 unit of the lot, which falls
// 30 - trunc(30 * 500 / 10 000) = 30 - 1 = 29, and the pool gets 35.
//
// Requirement: FR-EST-009, FR-EST-013
func TestEstateRound_Markdown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		bidQty int64
		pool   Money
		want   map[ledger.Key]Price
	}{
		{"nothing sold", 0, 0, map[ledger.Key]Price{k(lstStaple, 0, 0): 95, k(lstTool, 0, 3): 29, k(lstTool, 0, 6): 57}},
		{"a lot sold wholly", 2, 70, map[ledger.Key]Price{k(lstStaple, 0, 0): 95, k(lstTool, 0, 6): 57}},
		{"a lot sold partly", 1, 35, map[ledger.Key]Price{k(lstStaple, 0, 0): 95, k(lstTool, 0, 3): 29, k(lstTool, 0, 6): 57}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newListFixture(t, 1)
			f.threeLots(t)
			if tc.bidQty == 0 {
				f.round(t, 0, 0, 0)
			} else {
				f.round(t, lstTool, 40, tc.bidQty)
			}
			if got := f.e.pool(f.l, 0); got != tc.pool {
				t.Errorf("pool = %d, want %d", got, tc.pool)
			}
			if got := f.prices(); len(got) != len(tc.want) {
				t.Errorf("prices = %v, want %v", got, tc.want)
			} else {
				for key, p := range tc.want {
					if got[key] != p {
						t.Errorf("price of %v = %d, want %d (all %v)", key, got[key], p, got)
					}
				}
			}
			f.audit(t, tc.name)
		})
	}
}

// TestEstateRound_RejectedAskNotMarkedDown verifies that an ask the clearing
// rejects is not an ask and its lot keeps its price (D103).
//
// Arithmetic: the receipt tally of the estate is MaxMoney - 1, so the worst-case
// receipt of any ask (units * MaxPrice) does not fit under MaxMoney: all three
// are receipt_limit and no row is marked, so the prices stay 100, 30 and 60.
//
// Requirement: FR-EST-009
func TestEstateRound_RejectedAskNotMarkedDown(t *testing.T) {
	t.Parallel()
	f := newListFixture(t, 1)
	f.threeLots(t)
	f.tally.set(f.e.account(0), MaxMoney-1)
	orders := ordersOf(t, f)
	b := newBook(0, lstStaple, 1, &f.sc)
	r, ok := b.admit(f.l, f.e.facts(0, Ask, &f.tally), orders[0])
	if ok || r != ReasonReceiptLimit {
		t.Fatalf("admit = %q, %v, want %q, false", r, ok, ReasonReceiptLimit)
	}
	if err := f.e.markdown(f.l, f.cfg, 0); err != nil {
		t.Fatal(err)
	}
	if got := f.prices()[k(lstStaple, 0, 0)]; got != 100 {
		t.Errorf("price after a rejected ask = %d, want 100", got)
	}
}

// TestEstateRound_ArrivingUnitsAndRepricing verifies that units joining a
// marked-down lot are listed at its current price, and that goods arriving
// under the key of a lot sold wholly are listed at the first price, as a
// restored world lists them (D102, FR-EST-024).
//
// Arithmetic: tick 1 unsold, staple 100 -> 95. Three more units join the lot
// (8 units at 95). Tick 2 a bid of 95 for 8 buys it wholly at 95; its row is
// dropped at P5.3; 2 units under the same key list at the first price, 100.
//
// Requirement: FR-EST-008, FR-EST-024
func TestEstateRound_ArrivingUnitsAndRepricing(t *testing.T) {
	t.Parallel()
	f := newListFixture(t, 1)
	f.give(t, 0, lstStaple, 0, 5)
	f.round(t, 0, 0, 0)
	f.give(t, 0, lstStaple, 0, 3)
	if got := ordersOf(t, f); len(got) != 1 || got[0].Price != 95 || got[0].Quantity != 8 {
		t.Fatalf("asks after units joined = %+v, want one ask of 8 at 95", got)
	}
	f.round(t, lstStaple, 95, 8)
	if len(f.e.listed[0]) != 0 {
		t.Fatalf("rows after a wholesale sale = %+v, want none", f.e.listed[0])
	}
	f.give(t, 0, lstStaple, 0, 2)
	if got := ordersOf(t, f); len(got) != 1 || got[0].Price != 100 || got[0].Quantity != 2 {
		t.Fatalf("asks after the key came back = %+v, want one ask of 2 at 100", got)
	}
}

// TestEstateRows_StaleAndRoundTrip verifies that rows skips a stale row and a
// held lot with no row, that restore of rows gives a world whose next asks
// equal the uninterrupted one's, and that two runs agree.
//
// Arithmetic: after one unsold round the prices are 95, 29 (30 - trunc(30 * 500
// / 10 000) = 30 - 1) and 57. The tool lot of durability 3 then leaves the
// estate between ticks, so its row is stale and rows omits it.
//
// Requirement: FR-EST-024, FR-EST-012
func TestEstateRows_StaleAndRoundTrip(t *testing.T) {
	t.Parallel()
	f := newListFixture(t, 1)
	if got, err := f.e.rows(f.l, 0, nil); err != nil || len(got) != 0 {
		t.Fatalf("rows of an empty estate = %v, %v", got, err)
	}
	f.threeLots(t)
	if got, _ := f.e.rows(f.l, 0, nil); len(got) != 0 {
		t.Fatalf("rows with no listing yet = %v, want none (omitted, priced when listed)", got)
	}
	f.round(t, 0, 0, 0)
	if got := f.prices(); got[k(lstTool, 0, 3)] != 29 {
		t.Fatalf("price of the small tool lot = %d, want 29", got[k(lstTool, 0, 3)])
	}
	// Remove the small tool lot between ticks: its row is stale.
	if err := f.l.MoveLot(f.e.account(0), f.buyer, k(lstTool, 0, 3), 2, ledger.Trade); err != nil {
		t.Fatalf("MoveLot: %v", err)
	}
	saved, err := f.e.rows(f.l, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []listing{{key: k(lstStaple, 0, 0), price: 95}, {key: k(lstTool, 0, 6), price: 57}}
	if !slices.Equal(saved, want) {
		t.Fatalf("rows = %+v, want %+v", saved, want)
	}
	g := newListFixture(t, 1)
	g.give(t, 0, lstStaple, 0, 5)
	g.give(t, 0, lstTool, 6, 4)
	if err := g.e.restore(g.l, 0, saved); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if a, b := ordersOf(t, f), ordersOf(t, g); !slices.Equal(a, b) {
		t.Fatalf("asks of the restored world %+v, uninterrupted %+v", b, a)
	}
	f.round(t, 0, 0, 0)
	g.round(t, 0, 0, 0)
	if a, b := f.prices(), g.prices(); len(a) != len(b) || a[k(lstTool, 0, 6)] != b[k(lstTool, 0, 6)] || a[k(lstStaple, 0, 0)] != b[k(lstStaple, 0, 0)] {
		t.Fatalf("prices after another round %v, restored %v", b, a)
	}
}

// TestEstateRestore_Refusals verifies each refusal of D109 is errListingRows
// and leaves the listings as they were.
//
// Arithmetic: the estate holds staple (5, 0, 0) and tool (6, 0, 3); MaxPrice is
// 1 000 000 000, so MaxPrice + 1 is out of range and 0 is below 1.
//
// Requirement: FR-EST-024
func TestEstateRestore_Refusals(t *testing.T) {
	t.Parallel()
	a, b := k(lstStaple, 0, 0), k(lstTool, 0, 3)
	tests := []struct {
		name string
		rows []listing
		ok   bool
	}{
		{"valid", []listing{{key: a, price: 7}, {key: b, price: MaxPrice}}, true},
		{"empty", nil, true},
		{"out of order", []listing{{key: b, price: 7}, {key: a, price: 7}}, false},
		{"repeated", []listing{{key: a, price: 7}, {key: a, price: 7}}, false},
		{"price zero", []listing{{key: a, price: 0}}, false},
		{"price above the maximum", []listing{{key: a, price: MaxPrice + 1}}, false},
		{"lot not held", []listing{{key: k(lstTool, 0, 6), price: 7}}, false},
		{"key outside the ranges", []listing{{key: k(99, -5, 77), price: 7}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newListFixture(t, 1)
			f.give(t, 0, lstStaple, 0, 1)
			f.give(t, 0, lstTool, 3, 1)
			prior := []listing{{key: a, price: 11}}
			f.e.listed[0] = slices.Clone(prior)
			err := f.e.restore(f.l, 0, tc.rows)
			if tc.ok {
				if err != nil || !slices.Equal(f.e.listed[0], tc.rows) && len(tc.rows) != 0 {
					t.Fatalf("restore = %v, rows %+v, want nil and %+v", err, f.e.listed[0], tc.rows)
				}
				return
			}
			if !errors.Is(err, errListingRows) {
				t.Fatalf("restore error = %v, want errListingRows", err)
			}
			if !slices.Equal(f.e.listed[0], prior) {
				t.Errorf("listings after a refusal = %+v, want unchanged %+v", f.e.listed[0], prior)
			}
		})
	}
}

// TestEstateAsks_Deterministic verifies that the asks do not depend on the
// order in which the lots arrived, and that two runs are equal.
//
// Requirement: FR-EST-012
func TestEstateAsks_Deterministic(t *testing.T) {
	t.Parallel()
	f, g := newListFixture(t, 1), newListFixture(t, 1)
	f.threeLots(t)
	g.give(t, 0, lstTool, 6, 4)
	g.give(t, 0, lstStaple, 0, 5)
	g.give(t, 0, lstTool, 3, 2)
	for i := range 3 {
		f.round(t, 0, 0, 0)
		g.round(t, 0, 0, 0)
		if a, b := ordersOf(t, f), ordersOf(t, g); !slices.Equal(a, b) {
			t.Fatalf("round %d: asks %+v and %+v differ", i, a, b)
		}
	}
}

// TestEstates_WarmedWalkAllocatesNothing verifies that a warmed asks and
// markdown over three regions allocate nothing.
//
// Requirement: NFR-EST-001
func TestEstates_WarmedWalkAllocatesNothing(t *testing.T) {
	f := newListFixture(t, 3)
	for r := range RegionID(3) {
		f.give(t, r, lstStaple, 0, 5)
		f.give(t, r, lstTool, 3, 2)
		f.give(t, r, lstTool, 6, 4)
	}
	dst := make([]Order, 0, 16)
	walk := func() {
		for r := range RegionID(3) {
			dst, _ = f.e.asks(f.l, f.cfg, r, dst[:0])
			for _, o := range dst {
				f.e.admitted(r, o)
			}
			_ = f.e.markdown(f.l, f.cfg, r)
		}
	}
	walk()
	walk()
	if n := testing.AllocsPerRun(50, walk); n != 0 {
		t.Errorf("warmed asks and markdown allocated %v times per run, want 0", n)
	}
}

// FuzzRestore checks that restore never panics, that rows it accepts round-trip
// through rows, and that rows it refuses are errListingRows and change nothing.
// The estate holds the keys 0 to 2 below; key 3 is not held.
//
// Requirement: FR-EST-024
func FuzzRestore(f *testing.F) {
	f.Add([]byte{0, 1, 2}, int64(100))
	f.Add([]byte{0, 1, 2}, int64(MaxPrice-2))
	f.Add([]byte{2, 1}, int64(5))
	f.Add([]byte{0, 3}, int64(5))
	f.Add([]byte{1, 1}, int64(0))
	f.Add([]byte{}, int64(-1))
	f.Fuzz(func(t *testing.T, sel []byte, price int64) {
		keys := []ledger.Key{k(lstStaple, 0, 0), k(lstTool, 0, 3), k(lstTool, 0, 6), k(lstTool, 0, 9)}
		fx := newListFixture(t, 1)
		fx.give(t, 0, lstStaple, 0, 1)
		fx.give(t, 0, lstTool, 3, 1)
		fx.give(t, 0, lstTool, 6, 1)
		var rows []listing
		valid := true
		for i, s := range sel {
			row := listing{key: keys[int(s)%len(keys)], price: Price(price + int64(i))}
			if int(s)%len(keys) == 3 || row.price < 1 || row.price > MaxPrice || (i > 0 && cmpKey(rows[i-1].key, row.key) >= 0) {
				valid = false
			}
			rows = append(rows, row)
		}
		err := fx.e.restore(fx.l, 0, rows)
		if valid != (err == nil) || (err != nil && !errors.Is(err, errListingRows)) {
			t.Fatalf("restore(%+v) = %v, valid %v", rows, err, valid)
		}
		if err != nil {
			return
		}
		got, err := fx.e.rows(fx.l, 0, nil)
		if err != nil || !slices.Equal(got, rows) && len(rows) != 0 {
			t.Fatalf("rows = %+v, %v, want %+v", got, err, rows)
		}
	})
}
