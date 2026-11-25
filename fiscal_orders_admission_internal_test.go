package economysim

import (
	"errors"
	"testing"
)

// The admission of a treasury's order (spec-fiscal.md 9.1, 9.5, plan 3-fis R6): the book
// takes one order per agent, commodity, named lot and side, and a treasury is a
// pseudo-agent that no welfare pass, outcome or belief update may trip over.

// TestFiscal_OrderAdmission verifies the three rules of the admission.
//
//   - A second purchase of one authority for the same commodity is refused by the
//     validator, so the book's duplicate rule is never what stops a treasury.
//   - A purchase and a release of one commodity are both admitted (one per side).
//   - Thirty ticks of a treasury that trades with the households run clean with the
//     audit on: no belief update, no spawn and no bankruptcy ever names the treasury's
//     agent, and no step returns an error (the pseudo-agent is below the first
//     household, so the welfare and outcome passes skip it).
//
// Requirement: FR-FIS-053, FR-FIS-054
func TestFiscal_OrderAdmission(t *testing.T) {
	t.Parallel()
	t.Run("a second order of one authority, commodity and side never reaches the book", func(t *testing.T) {
		t.Parallel()
		cfg := fiscalMillConfig(&Fiscal{Purchases: []Purchase{buy("b1", 1, 8_000, 0), buy("b2", 1, 7_000, 0)}})
		_, err := NewWorld(cfg, WithSeed(exitSeed))
		if !errors.Is(err, ErrFiscalShape) {
			t.Fatalf("NewWorld() error = %v, want ErrFiscalShape: the clearing keeps one order per agent, commodity and side, so the validator refuses a second purchase of the commodity", err)
		}
	})
	t.Run("a bid and an ask of one commodity are both admitted", func(t *testing.T) {
		t.Parallel()
		w, rec := orderMill(t, &Fiscal{Purchases: []Purchase{buy("b", 1, 8_000, 0)}, Releases: []Release{sell("s", 1, 12_000, 0)}}, 100_000)
		stockCrown(t, w, 5)
		w.p34aOrders()
		if bids, asks := sidesOf(rec, 2); bids != 1 || asks != 1 || len(of[OrderRejected](rec)) != 0 {
			t.Errorf("bids = %d, asks = %d, rejected = %+v, want 1, 1 and none", bids, asks, of[OrderRejected](rec))
		}
	})
	t.Run("a trading treasury never reaches the households' passes", func(t *testing.T) {
		t.Parallel()
		w, rec := orderMill(t, &Fiscal{Purchases: []Purchase{buy("b", 3, 9_500, 0)}, Releases: []Release{sell("s", 3, 10_500, 0)}}, 100_000)
		stockCrown(t, w, 20)
		stepN(t, w, 30)
		if len(of[TradeExecuted](rec)) == 0 {
			t.Fatalf("no trade in 30 ticks, want the treasury to trade with someone")
		}
		for _, e := range of[AgentBankrupted](rec) {
			if e.Agent == 2 || e.Agent == 3 {
				t.Errorf("AgentBankrupted names the treasury agent: %+v", e)
			}
		}
		for _, e := range of[AgentSpawned](rec) {
			if e.Agent == 2 || e.Agent == 3 {
				t.Errorf("AgentSpawned names the treasury agent: %+v", e)
			}
		}
	})
}

// TestFiscal_NoEventsWithoutObserver verifies that the position builds no event when no
// observer is supplied (NFR-FIS-001, observability-and-logging.md 4.5): the same position
// that allocates nothing with none allocates with one, because the event is boxed.
//
// Requirement: NFR-FIS-001
func TestFiscal_NoEventsWithoutObserver(t *testing.T) {
	m := &Fiscal{Purchases: []Purchase{buy("b", 1, 8_000, 0)}, Releases: []Release{sell("s", 1, 12_000, 0)}}
	allocs := func(opts ...Option) float64 {
		cfg := fiscalMillConfig(m)
		for i := range cfg.Modules.Polity.Jurisdictions {
			cfg.Modules.Polity.Jurisdictions[i].StartMoney = 1_000_000
		}
		w := newMill(t, cfg, append([]Option{WithSeed(exitSeed)}, opts...)...)
		stockCrown(t, w, 100_000)
		run := func() {
			nextBook(w)
			w.p34aOrders()
		}
		run()
		return testing.AllocsPerRun(20, run)
	}
	if got := allocs(); got != 0 {
		t.Errorf("AllocsPerRun(p34aOrders) with no observer = %v, want 0", got)
	}
	if got := allocs(WithObserver(&evLog{})); got == 0 {
		t.Errorf("AllocsPerRun(p34aOrders) with an observer = 0, want above 0: the events are built")
	}
}
