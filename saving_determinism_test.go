package economysim_test

import (
	"fmt"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestDeterminism_Saving verifies that a run of the saving pack is the same run however it is
// observed: for each seed, a run with a logger, an observer or the audit, and a run split into Run
// calls (the reduced case list, seeds 1 and 2), end in the snapshot and the state hash of the bare
// Step run, the events of the observed runs are equal, and a different seed gives a different run.
//
// Requirement: FR-SAV-002, FR-SAV-038, FR-SAV-039
//
// Guard: the module draws no word and keeps no state; a logger, an observer or the audit that
// changed what a household planned or asked for would change the run, and only a run with
// households that save, build and repair can show it.
func TestDeterminism_Saving(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSaving...)
	cases := detCases(true)
	seeds := seedList(2)
	finals := make([]economysim.Snapshot, len(seeds))
	for i, seed := range seeds {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			base := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks}))
			finals[i] = base.Final
			ref := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks, Observe: true}))
			refEvents := renderEvents(withoutRunBrackets(ref))
			if d := firstTickDiff(base.Samples, ref.Samples); d != "" {
				t.Fatalf("seed %d: observed Step run differs from the bare one at %s", seed, d)
			}
			for _, tc := range cases {
				spec := tc.spec()
				spec.Seed, spec.Ticks = seed, detTicks
				got := runPack(t, cfg, spec)
				if !reflect.DeepEqual(got.Final, base.Final) {
					t.Errorf("seed %d, case %q: final snapshot differs from the bare run", seed, tc.name)
				}
				if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
					t.Errorf("seed %d, case %q: state hash of the world differs from the bare run", seed, tc.name)
				}
				if spec.Observe {
					if d := firstLineDiff(refEvents, renderEvents(withoutRunBrackets(got))); d != "" {
						t.Errorf("seed %d, case %q: events differ from the reference at %s", seed, tc.name, d)
					}
				}
			}
		})
	}
	if len(seeds) > 1 && reflect.DeepEqual(finals[0], finals[1]) {
		t.Errorf("seeds %d and %d end in the same snapshot, want different runs", seeds[0], seeds[1])
	}
}
