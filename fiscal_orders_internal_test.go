package economysim

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The tests of the standing orders of the fiscal module at P3.4a (spec-fiscal.md 9,
// plan 3-fis Phase 8). They build the fiscal mill (fiscal_tax_internal_test.go: the
// crown seated at home with agent 2, the shire seated at yonder with agent 3, grain
// at a reference price of 1 000), call the position and read the book, the ledger and
// the events: the arithmetic of a limit, a wish and a cap is tested in internal/fiscal.

// orderMill builds the fiscal mill with the module m; the crown and the shire hold
// crownCash each. The events of construction are forgotten.
func orderMill(t *testing.T, m *Fiscal, crownCash Money) (*World, *evLog) {
	t.Helper()
	return orderMillWith(t, m, func(c *Config) {
		for i := range c.Modules.Polity.Jurisdictions {
			c.Modules.Polity.Jurisdictions[i].StartMoney = crownCash
		}
	})
}

// orderMillWith is orderMill with mutate applied to the configuration first.
func orderMillWith(t *testing.T, m *Fiscal, mutate func(*Config), opts ...Option) (*World, *evLog) {
	t.Helper()
	cfg := fiscalMillConfig(m)
	mutate(&cfg)
	rec := &evLog{}
	w := newMill(t, cfg, append([]Option{WithSeed(exitSeed), WithAudit(), WithObserver(rec)}, opts...)...)
	rec.events = nil
	return w, rec
}

// buy and sell are a purchase and a release of the crown of grain.
func buy(id string, qty, limitBP, hold int64) Purchase {
	return Purchase{ID: id, Authority: "crown", Commodity: "grain", Quantity: qty, LimitBP: limitBP, Hold: hold}
}

func sell(id string, qty, limitBP, reserve int64) Release {
	return Release{ID: id, Authority: "crown", Commodity: "grain", Quantity: qty, LimitBP: limitBP, Reserve: reserve}
}

// nextBook empties the books and the receipt tally, as the start of P3.2 does, so that
// the position can run again in one test without a second step.
func nextBook(w *World) {
	w.markets.reset(w.Tick())
	w.receipts.reset()
}

// placed returns the OrderPlaced events of the tick in order.
func placed(rec *evLog) []OrderPlaced { return of[OrderPlaced](rec) }

// TestFiscal_OrderLimit verifies the price rule of spec-fiscal.md 9.2: a purchase bids
// and a release asks at LimitBP of the effective reference price of the seat region,
// truncated toward zero and clamped to at least 1, and nothing but the configuration
// moves it.
//
// Arithmetic: the reference price of grain at home is 1 000. 8 000 bp gives
// 1 000 * 8 000 / 10 000 = 800; 12 000 bp gives 1 200; 1 bp gives 1 000 * 1 / 10 000 =
// 0.1, truncated to 0 and clamped to 1; 15 bp gives 1.5, truncated to 1.
//
// Requirement: FR-FIS-050
func TestFiscal_OrderLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		buy  []Purchase
		sell []Release
		side Side
		want Price
	}{
		{"bid at 80 percent", []Purchase{buy("b", 1, 8_000, 0)}, nil, Bid, 800},
		{"ask at 120 percent", nil, []Release{sell("s", 1, 12_000, 0)}, Ask, 1_200},
		{"a limit below 1 is clamped to 1", []Purchase{buy("b", 1, 1, 0)}, nil, Bid, 1},
		{"a limit of 1.5 is truncated to 1", []Purchase{buy("b", 1, 15, 0)}, nil, Bid, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec := orderMill(t, &Fiscal{Purchases: tt.buy, Releases: tt.sell}, 100_000)
			if tt.side == Ask {
				stockCrown(t, w, 10)
			}
			w.p34aOrders()
			got := placed(rec)
			if len(got) != 1 || got[0].Side != tt.side || got[0].Limit != tt.want || got[0].Agent != 2 || got[0].Region != 0 {
				t.Fatalf("p34aOrders() placed %+v, want one %v of the crown (agent 2) at home with limit %d", got, tt.side, tt.want)
			}
		})
	}
}

// TestFiscal_OrderQuantity verifies the wish and the caps of spec-fiscal.md 9.3: a
// purchase is cut first by the cash and then by the hold, a release by the stock less
// the reserve, each cut is counted in TreasuryOrdered, and an order is placed for the
// units left only if there is at least one.
//
// Arithmetic (crown cash C, limit 800 for a purchase at 8 000 bp of 1 000): cash 4 000
// buys floor(4 000 / 800) = 5 of a wish of 8, so CutCash 3; a hold of 5 over 3 held
// leaves room for 2 of a wish of 5, CutHold 3; cash 1 000 buys 1 and the hold of 5 over
// 0 held allows 5, so only the cash cuts (CutCash 4 of 5, then nothing at the hold); a
// release of wish 5 with 6 held and reserve 4 sells 6 - 4 = 2, CutStock 3; with 3 held
// and reserve 4 nothing is left and no order is placed.
//
// Requirement: FR-FIS-051
func TestFiscal_OrderQuantity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		buy   []Purchase
		sell  []Release
		cash  Money
		stock int64
		want  TreasuryOrdered
		units Quantity
	}{
		{"the cash cuts a purchase", []Purchase{buy("b", 8, 8_000, 0)}, nil, 4_000, 0,
			TreasuryOrdered{Bids: 1, WantedBuy: 8, PlacedBuy: 5, CutCash: 3}, 5},
		{"the hold cuts a purchase", []Purchase{buy("b", 5, 8_000, 5)}, nil, 100_000, 3,
			TreasuryOrdered{Bids: 1, WantedBuy: 5, PlacedBuy: 2, CutHold: 3}, 2},
		{"the cash cuts before the hold", []Purchase{buy("b", 5, 8_000, 5)}, nil, 1_000, 0,
			TreasuryOrdered{Bids: 1, WantedBuy: 5, PlacedBuy: 1, CutCash: 4}, 1},
		{"the reserve cuts a release", nil, []Release{sell("s", 5, 12_000, 4)}, 100_000, 6,
			TreasuryOrdered{Asks: 1, WantedSell: 5, PlacedSell: 2, CutStock: 3}, 2},
		{"a release with nothing above the reserve places no order", nil, []Release{sell("s", 5, 12_000, 4)}, 100_000, 3,
			TreasuryOrdered{WantedSell: 5, CutStock: 5}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec := orderMill(t, &Fiscal{Purchases: tt.buy, Releases: tt.sell}, tt.cash)
			if tt.stock > 0 {
				stockCrown(t, w, tt.stock)
			}
			w.p34aOrders()
			got := of[TreasuryOrdered](rec)
			if len(got) != 1 {
				t.Fatalf("p34aOrders() emitted %d TreasuryOrdered, want 1", len(got))
			}
			tt.want.Authority, tt.want.Region = 0, 0
			if got[0] != tt.want {
				t.Errorf("TreasuryOrdered = %+v, want %+v", got[0], tt.want)
			}
			var units Quantity
			for _, o := range placed(rec) {
				units += o.Quantity
			}
			if units != tt.units {
				t.Errorf("OrderPlaced units = %d, want %d", units, tt.units)
			}
		})
	}
}

// TestFiscal_OrderWishCarry verifies the carry of the wish (spec-fiscal.md 9.3.1): a
// Quantity of 1 at a scale of 5 000 wants one unit every second tick, and the carry
// advances in a tick where no order is placed.
//
// Arithmetic: (1 * 5 000 + 0) / 10 000 = 0, carry 5 000; then (5 000 + 5 000) / 10 000 =
// 1, carry 0; and so on, wishes 0, 1, 0, 1.
//
// Requirement: FR-FIS-051
func TestFiscal_OrderWishCarry(t *testing.T) {
	t.Parallel()
	w, rec := orderMill(t, &Fiscal{Purchases: []Purchase{buy("b", 1, 8_000, 0)}}, 100_000)
	if err := w.setScaleOf("b", 5_000, CauseConsumer); err != nil {
		t.Fatalf("setScaleOf() error = %v, want nil", err)
	}
	var got []int64
	for range 4 {
		rec.events = nil
		nextBook(w)
		w.p34aOrders()
		var n int64
		for _, e := range of[TreasuryOrdered](rec) {
			n += e.WantedBuy
		}
		got = append(got, n)
	}
	if want := []int64{0, 1, 0, 1}; !slices.Equal(got, want) {
		t.Errorf("wishes over four ticks at scale 5 000 = %v, want %v", got, want)
	}
}

// TestFiscal_OrderPair verifies the pair of spec-fiscal.md 9.4.1: a purchase and a
// release of one commodity of one authority are both admitted, the bid before the ask,
// because the clearing keeps one order per agent, commodity, lot and side.
//
// Arithmetic: the limits are 800 and 1 200 on a reference price of 1 000, strictly
// ordered, so the two never cross.
//
// Requirement: FR-FIS-053
func TestFiscal_OrderPair(t *testing.T) {
	t.Parallel()
	w, rec := orderMill(t, &Fiscal{Purchases: []Purchase{buy("b", 2, 8_000, 0)}, Releases: []Release{sell("s", 2, 12_000, 0)}}, 100_000)
	stockCrown(t, w, 10)
	w.p34aOrders()
	got := placed(rec)
	if len(got) != 2 || got[0].Side != Bid || got[1].Side != Ask {
		t.Fatalf("OrderPlaced = %+v, want the crown's bid then its ask", got)
	}
	if o := of[OrderRejected](rec); len(o) != 0 {
		t.Errorf("OrderRejected = %+v, want none: the pair is admitted", o)
	}
	if te := of[TreasuryOrdered](rec); len(te) != 1 || te[0].Bids != 1 || te[0].Asks != 1 {
		t.Errorf("TreasuryOrdered = %+v, want one with 1 bid and 1 ask", te)
	}
}

// sidesOf counts the bids and the asks the agent placed.
func sidesOf(rec *evLog, agent AgentID) (bids, asks int) {
	for _, o := range placed(rec) {
		switch {
		case o.Agent != agent:
		case o.Side == Bid:
			bids++
		default:
			asks++
		}
	}
	return bids, asks
}

// TestFiscal_TreasuryNeverMatchesItself verifies, over a run of the audited world with
// a band of the crown on the one commodity the households trade, that no trade has the
// treasury on both sides and that the books accept the pair every tick (spec-fiscal.md
// 9.4.1, FR-FIS-053).
//
// Guard: the one-order rule of the clearing exists so an agent never trades with
// itself; a treasury places two orders of one commodity, which is safe only because
// its limits are strictly ordered. A change that let them touch would make a trade of
// the crown with itself, which the sectoral audit would not see.
//
// Requirement: FR-FIS-053
func TestFiscal_TreasuryNeverMatchesItself(t *testing.T) {
	t.Parallel()
	w, rec := orderMill(t, &Fiscal{Purchases: []Purchase{buy("b", 3, 9_500, 0)}, Releases: []Release{sell("s", 3, 10_500, 0)}}, 100_000)
	stockCrown(t, w, 20)
	stepN(t, w, 30)
	bids, asks := sidesOf(rec, 2)
	if bids == 0 || asks == 0 {
		t.Fatalf("the crown placed %d bids and %d asks in 30 ticks, want both above 0", bids, asks)
	}
	for _, tr := range of[TradeExecuted](rec) {
		if tr.Buyer == 2 && tr.Seller == 2 {
			t.Fatalf("tick %d: the treasury traded with itself: %+v", tr.Tick, tr)
		}
	}
	if o := of[OrderRejected](rec); len(o) != 0 {
		t.Errorf("OrderRejected = %+v, want none: a duplicate would mean one order per commodity was broken", o)
	}
}

// TestFiscal_OrderInputOrder verifies the position of a treasury's orders in the book of
// a market (spec-fiscal.md 9.4.2): after the estate's ask and in the order the position
// places them, the bid before the ask; and in the events, authorities ascending, each
// one's OrderPlaced before its TreasuryOrdered.
//
// Requirement: FR-FIS-054
func TestFiscal_OrderInputOrder(t *testing.T) {
	t.Parallel()
	m := &Fiscal{
		Purchases: []Purchase{buy("b", 1, 8_000, 0), {ID: "bs", Authority: "shire", Commodity: "grain", Quantity: 1, LimitBP: 8_000}},
		Releases:  []Release{sell("s", 1, 12_000, 0)},
	}
	w, rec := orderMill(t, m, 100_000)
	stockCrown(t, w, 5)
	src, ok := w.ledger.Boundary(ledger.Initial, int(payoutGrain(w)))
	if !ok {
		t.Fatalf("no initial boundary for grain")
	}
	// four units in the home estate, which lists and asks them at P3.4
	if err := w.ledger.Move(src, w.estates.account(0), int(payoutGrain(w)), 4, ledger.Initial); err != nil {
		t.Fatalf("stock the home estate: %v", err)
	}
	if err := w.p34Estates(); err != nil {
		t.Fatalf("p34Estates() error = %v, want nil", err)
	}
	w.p34aOrders()
	var order []AgentID
	for _, e := range w.markets.books[w.markets.index(0, payoutGrain(w))].entries {
		order = append(order, e.agent)
	}
	if want := []AgentID{0, 2, 2}; !slices.Equal(order, want) {
		t.Errorf("book of home grain = %v, want the estate's ask, then the crown's bid and its ask %v", order, want)
	}
	var kinds []string
	for _, e := range rec.events {
		switch e := e.(type) {
		case OrderPlaced:
			kinds = append(kinds, "placed:"+string(rune('0'+e.Agent)))
		case TreasuryOrdered:
			kinds = append(kinds, "ordered:"+string(rune('0'+e.Authority)))
		}
	}
	if want := []string{"placed:0", "placed:2", "placed:2", "ordered:0", "placed:3", "ordered:1"}; !slices.Equal(kinds, want) {
		t.Errorf("events = %v, want %v", kinds, want)
	}
}
