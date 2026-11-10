package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s66ReopenAfter is the number of ticks the edge stays closed.
const s66ReopenAfter = 20

// TestScenarioS66_ClosureDuringJourney is the closure leg of MIG-S6 (spec-migration.md,
// S66): the migration pack runs with the audit on and, at the first boundary at which a
// household is on a journey, the public SetEdgeStatuses closes the edge between the two
// regions; it reopens s66ReopenAfter ticks later. Exact: the closure is made while a
// journey is in flight; EdgeStatusChanged is emitted once for the closure and once for
// the reopening; and the invariants of the other leg hold across both: every household
// started is accounted for, the outcomes add to Pushed, the events rebuild the
// household counts and the pipeline stays within its cap.
//
// Requirement: FR-INT-041, FR-INT-042, FR-MIG-033, FR-MIG-037, FR-MIG-025
//
// Guard: a closure that dropped or duplicated a household on the way, or left the
// pipeline over its cap, would break the audit or one of the four counts.
func TestScenarioS66_ClosureDuringJourney(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := movingPack(t, packMigration("two"), nil)
	rep := newPackScenReport("two", "S66 the closure of an edge during a journey (MIG-S6)")
	for _, seed := range seedList(scenarioSeedCount()) {
		closedAt := int64(-1)
		var hadJourney bool
		tl := runMigration(t, cfg, seed, migS66Tick, func(w *economysim.World, tick int64) {
			closed := []economysim.EdgeStatus{{From: 0, To: 1, Closed: true, FactorBP: 10_000}}
			switch {
			case closedAt < 0 && tick > 0:
				s, err := w.Snapshot()
				if err != nil {
					t.Fatalf("seed %d: Snapshot() at tick %d error = %v, want nil", seed, tick, err)
				}
				var onWay int64
				for _, r := range s.Regions {
					onWay += r.OnJourneyFrom
				}
				if onWay == 0 {
					return
				}
				closedAt, hadJourney = tick, true
				if err := w.SetEdgeStatuses(closed); err != nil {
					t.Fatalf("seed %d: SetEdgeStatuses(close) error = %v, want nil", seed, err)
				}
			case closedAt >= 0 && tick == closedAt+s66ReopenAfter:
				closed[0].Closed = false
				if err := w.SetEdgeStatuses(closed); err != nil {
					t.Fatalf("seed %d: SetEdgeStatuses(open) error = %v, want nil", seed, err)
				}
			}
		})
		migCheck(t, rep, seed, "s66c.closed_during_a_journey", hadJourney, "no household was on a journey in 200 ticks, so no edge was closed")
		var changes int
		for _, e := range tl.events {
			if _, ok := e.(economysim.EdgeStatusChanged); ok {
				changes++
			}
		}
		migCheck(t, rep, seed, "s66c.status_events", changes == 2, fmt.Sprintf("%d EdgeStatusChanged, want 2 (the closure and the reopening)", changes))
		migCheck(t, rep, seed, "s66c.reconciles", tl.started >= tl.arrived+tl.endFlight+tl.ended, fmt.Sprintf("started %d, arrived %d, on the way %d, ended %d", tl.started, tl.arrived, tl.endFlight, tl.ended))
		migCheck(t, rep, seed, "s66c.outcomes_add_to_pushed", tl.reviewMismatches == 0, fmt.Sprintf("%d reviews whose outcomes do not add to Pushed", tl.reviewMismatches))
		migCheck(t, rep, seed, "s66c.counts_rebuilt_from_events", tl.rebuildMismatches == 0, fmt.Sprintf("%d region boundaries where the events rebuild a household count the snapshot does not hold", tl.rebuildMismatches))
		migCheck(t, rep, seed, "s66c.pipeline_within_cap", tl.capBreaches == 0, fmt.Sprintf("%d boundaries over the pipeline cap", tl.capBreaches))
	}
}
