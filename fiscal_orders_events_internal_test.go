package economysim

import (
	"slices"
	"testing"
)

// The events, the refusals and the limits of the standing orders of the fiscal module at
// P3.4a (spec-fiscal.md 9, 16.1.2, plan 3-fis Phase 8).

// TestFiscal_Events_P34a verifies when TreasuryOrdered is emitted (FR-FIS-045): once per
// authority after its OrderPlaced events when a standing order of it was in its window
// with a wish of at least 1, and never otherwise: not before the window starts, not at
// scale 0, not when the wish is below 1, and not for an authority with no order.
//
// Arithmetic: a window that starts at tick 5 is closed at tick 0; a scale of 5 000 on a
// Quantity of 1 gives a wish of (5 000 + 0) / 10 000 = 0 in the first tick; the shire has
// no order, so only the crown's flows can emit.
//
// Requirement: FR-FIS-045
func TestFiscal_Events_P34a(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		buy   Purchase
		scale int64
		want  int
	}{
		{"in its window with a wish", buy("b", 1, 8_000, 0), 10_000, 1},
		{"before the window starts", Purchase{ID: "b", Authority: "crown", Commodity: "grain", Quantity: 1, LimitBP: 8_000, Window: Window{Start: 5}}, 10_000, 0},
		{"at scale 0", buy("b", 1, 8_000, 0), 0, 0},
		{"a wish below 1", buy("b", 1, 8_000, 0), 5_000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec := orderMill(t, &Fiscal{Purchases: []Purchase{tt.buy}}, 100_000)
			if err := w.setScaleOf("b", tt.scale, CauseConsumer); err != nil {
				t.Fatalf("setScaleOf() error = %v, want nil", err)
			}
			rec.events = nil
			w.p34aOrders()
			if got := of[TreasuryOrdered](rec); len(got) != tt.want {
				t.Errorf("TreasuryOrdered events = %d, want %d", len(got), tt.want)
			}
		})
	}
}

// TestFiscal_OrderRejected verifies that an order the book refuses is an OrderRejected
// with its reason, in placing order and before the authority's TreasuryOrdered, which
// counts the wish and places none (FR-FIS-045, FR-FIS-051).
//
// Arithmetic: the crown holds MaxMoney less 1 000 and releases 2 units; the worst-case
// receipt of an ask is 2 * MaxPrice = 2 * 10^9, which with the balance is above MaxMoney,
// so the ask is refused as receipt_limit.
//
// Requirement: FR-FIS-045, FR-FIS-051
func TestFiscal_OrderRejected(t *testing.T) {
	t.Parallel()
	w, rec := orderMillWith(t, &Fiscal{Releases: []Release{sell("s", 2, 12_000, 0)}}, func(c *Config) {
		c.Modules.Polity.Jurisdictions[0].StartMoney = MaxMoney - 1_000
		c.Modules.Polity.Jurisdictions[1].StartMoney = 0
	})
	stockCrown(t, w, 5)
	w.p34aOrders()
	rej, ordered := of[OrderRejected](rec), of[TreasuryOrdered](rec)
	if len(rej) != 1 || rej[0].Reason != ReasonReceiptLimit || rej[0].Agent != 2 || len(placed(rec)) != 0 {
		t.Fatalf("OrderRejected = %+v, OrderPlaced = %+v, want one rejection receipt_limit of the crown and no order placed", rej, placed(rec))
	}
	if len(ordered) != 1 || ordered[0].WantedSell != 2 || ordered[0].PlacedSell != 0 || ordered[0].Asks != 0 {
		t.Errorf("TreasuryOrdered = %+v, want a wish of 2, none placed", ordered)
	}
	if last := rec.events[len(rec.events)-1]; last != Event(ordered[0]) {
		t.Errorf("last event = %+v, want TreasuryOrdered after the OrderRejected", last)
	}
}

// TestFiscal_OrderSeatOnly verifies that a treasury's order goes to the market of its
// seat region and no other (spec-fiscal.md 9.1.1, FR-FIS-055): the shire, seated at
// yonder, buys in yonder's book and leaves home's untouched.
//
// Requirement: FR-FIS-055
func TestFiscal_OrderSeatOnly(t *testing.T) {
	t.Parallel()
	shire := Purchase{ID: "bs", Authority: "shire", Commodity: "grain", Quantity: 1, LimitBP: 8_000}
	w, rec := orderMill(t, &Fiscal{Purchases: []Purchase{shire}}, 100_000)
	w.p34aOrders()
	got := placed(rec)
	if len(got) != 1 || got[0].Region != 1 || got[0].Agent != 3 {
		t.Fatalf("OrderPlaced = %+v, want one order of the shire (agent 3) in yonder (region 1)", got)
	}
	g := payoutGrain(w)
	if n := len(w.markets.books[w.markets.index(0, g)].entries); n != 0 {
		t.Errorf("the book of home grain holds %d orders, want 0: a seat's order is in no other market", n)
	}
	if n := len(w.markets.books[w.markets.index(1, g)].entries); n != 1 {
		t.Errorf("the book of yonder grain holds %d orders, want the shire's 1", n)
	}
}

// TestFiscal_TreasuryGoodsSpoil verifies that the units a treasury holds age and spoil as
// any (spec-fiscal.md 9.6.2, FR-FIS-056): a lot of a perishable commodity the crown holds
// moves whole to the spoilage boundary at the end of the tick in which its shelf life
// ends, and the audit, on at every step, accepts it.
//
// Arithmetic: a shelf life of 2 and a lot born at tick 0 spoil at the end of tick 1:
// Born + ShelfLife - 1 = 1.
//
// Requirement: FR-FIS-056
func TestFiscal_TreasuryGoodsSpoil(t *testing.T) {
	t.Parallel()
	w, _ := orderMillWith(t, &Fiscal{Purchases: []Purchase{buy("b", 1, 8_000, 0)}}, func(c *Config) {
		i := slices.IndexFunc(c.Commodities, func(x Commodity) bool { return x.ID == "grain" })
		c.Commodities[i].ShelfLife = 2
	})
	stockCrown(t, w, 5)
	account, g := w.polity.tr[0].account, int(payoutGrain(w))
	stepN(t, w, 1)
	if held, _, _ := w.ledger.Holding(account, g); held != 5 {
		t.Fatalf("the crown holds %d grain after tick 0, want its 5 (the lot has a tick left)", held)
	}
	stepN(t, w, 1)
	if held, _, _ := w.ledger.Holding(account, g); held != 0 {
		t.Errorf("the crown holds %d grain after tick 1, want 0: the lot spoiled whole", held)
	}
}

// TestFiscal_OrderAllocsPerRun verifies that the position allocates nothing in steady
// state with no observer supplied (NFR-FIS-001, observability-and-logging.md 4.5): the
// books and the records keep their capacity from a tick to the next, and no event is
// built.
//
// Requirement: NFR-FIS-001
func TestFiscal_OrderAllocsPerRun(t *testing.T) {
	m := &Fiscal{Purchases: []Purchase{buy("b", 1, 8_000, 0)}, Releases: []Release{sell("s", 1, 12_000, 0)}}
	cfg := fiscalMillConfig(m)
	for i := range cfg.Modules.Polity.Jurisdictions {
		cfg.Modules.Polity.Jurisdictions[i].StartMoney = 1_000_000
	}
	w := newMill(t, cfg, WithSeed(exitSeed))
	stockCrown(t, w, 100_000)
	run := func() {
		nextBook(w)
		w.p34aOrders()
	}
	run() // grows the books and the records once
	if got := testing.AllocsPerRun(20, run); got != 0 {
		t.Errorf("AllocsPerRun(p34aOrders) = %v, want 0", got)
	}
}
