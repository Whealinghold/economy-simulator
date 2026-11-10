package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestScenarioS52_PublicClosure is the public leg of EXT-S2 (spec-stability.md 8, S52):
// pack one with the external overlay runs 16 ticks with the audit on and, from tick 6,
// the public SetEdgeStatuses closes both edges of the gate between `north` and the
// outside node `outer`. Exact: before the closure no unit is cut by it and the fuel line
// offers; from the tick after the call every unit the line would offer is counted as
// CutClosed and none is offered or traded; one EdgeStatusChanged is emitted for each
// edge; the audit holds.
//
// Requirement: FR-INT-041, FR-INT-042, FR-EXT-032, FR-EXT-035
//
// Guard: a closed edge that still placed an order, or a call that did not reach the
// outside sector, would show in the counters of the line.
func TestScenarioS52_PublicClosure(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const closeAt, ticks = 6, 16
	cfg := loadValidPack(t, packExternal()...)
	north, outer := regionNamed(t, cfg, "north"), regionNamed(t, cfg, "outer")
	for _, seed := range seedList(scenarioSeedCount()) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			rec := &recorder{}
			w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(rec))
			if err != nil {
				t.Fatalf("seed %d: NewWorld() error = %v, want nil", seed, err)
			}
			for i := range ticks {
				if i == closeAt {
					gate := []economysim.EdgeStatus{
						{From: north, To: outer, Closed: true, FactorBP: 10_000},
						{From: outer, To: north, Closed: true, FactorBP: 10_000},
					}
					if err := w.SetEdgeStatuses(gate); err != nil {
						t.Fatalf("seed %d: SetEdgeStatuses() error = %v, want nil", seed, err)
					}
				}
				step(t, w, 1)
			}
			if got := eventsAs[economysim.EdgeStatusChanged](rec.events); len(got) != 2 || !got[0].Closed || !got[1].Closed {
				t.Errorf("seed %d: EdgeStatusChanged = %+v, want two closures", seed, got)
			}
			var cutAfter int64
			for _, e := range eventsAs[economysim.OutsideTraded](rec.events) {
				if e.Tick < closeAt && e.CutClosed != 0 {
					t.Errorf("seed %d: tick %d: %+v, want nothing cut while the gate is open", seed, e.Tick, e)
				}
				if e.Tick >= closeAt {
					cutAfter += e.CutClosed
					if e.Offered != 0 || e.Traded != 0 {
						t.Errorf("seed %d: tick %d: %+v, want nothing offered or traded through the closed gate", seed, e.Tick, e)
					}
				}
			}
			if cutAfter == 0 {
				t.Errorf("seed %d: CutClosed over the closure = 0, want the capacity of the lines counted", seed)
			}
		})
	}
}
