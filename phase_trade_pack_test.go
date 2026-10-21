package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestStep_NoReservationIsOpenAtAnyTickBoundary verifies, over the pack one and a
// range of seeds, that no reservation is open at any tick boundary
// (spec-world.md 6.1.6, FR-WLD-045) while the market is live: every run trades, so
// the check is not made over a world that placed no order. A failing seed is named
// in the message (testing.md 6.2). The direct check after P4.1 is the internal
// test TestPositions_NoReservationIsOpenAfterP41.
//
// Requirement: FR-WLD-045, FR-LED-020
func TestStep_NoReservationIsOpenAtAnyTickBoundary(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	cfg := loadPack(t, "base.toml", "one.toml")
	for seed := uint64(1); seed <= 4; seed++ {
		rec := &recorder{}
		w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithObserver(rec), economysim.WithAudit())
		if err != nil {
			t.Fatalf("seed %d: NewWorld(one) error = %v, want nil", seed, err)
		}
		for tick := range 50 {
			if err := w.Step(); err != nil {
				t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, tick, err)
			}
			if economysim.HasReservation(w) {
				t.Fatalf("seed %d: a reservation is open at the boundary after tick %d, want none", seed, tick)
			}
		}
		trades := 0
		for _, e := range rec.events {
			if _, ok := e.(economysim.TradeExecuted); ok {
				trades++
			}
		}
		if trades == 0 {
			t.Errorf("seed %d: no trade in 50 ticks of the pack one, want some, or the window was never open", seed)
		}
	}
}

// TestStep_MarketClearedCarriesTheCostAnchorInForce verifies over warm-one that the
// CostAnchor of every MarketCleared equals the anchor of P1.2 for its region and
// commodity (spec-observability.md, FR-OBS-097, D157), and that some anchor is
// above 0, so the check is not made over a world with no anchor.
//
// Arithmetic: at tick 0 the cost anchors read the starting wage 2 512 and give 180
// for fuel, 222 for seed and 838 for mineral (TestStep_FirstTickCostAnchorsReadTheStartingWage), so the first step's MarketCleared for those carry them.
//
// Requirement: FR-OBS-097, FR-PRO-037
func TestStep_MarketClearedCarriesTheCostAnchorInForce(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	rec := &recorder{}
	w, _ := buildWarm(t, []string{"base.toml", "one.toml", "warm-one.toml"}, economysim.WithObserver(rec))
	rec.forget()
	positive := 0
	for tick := range 6 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", tick, err)
		}
		for _, e := range rec.events {
			mc, ok := e.(economysim.MarketCleared)
			if !ok {
				continue
			}
			if want := economysim.UnitCostOf(w, mc.Region, mc.Commodity); int64(mc.CostAnchor) != want {
				t.Errorf("tick %d: MarketCleared of region %d commodity %d has CostAnchor %d, want %d", tick, mc.Region, mc.Commodity, mc.CostAnchor, want)
			}
			if mc.CostAnchor > 0 {
				positive++
			}
		}
		rec.forget()
	}
	if positive == 0 {
		t.Errorf("no MarketCleared carried a cost anchor above 0, want some")
	}
}
