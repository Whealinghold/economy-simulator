package economysim_test

import (
	"fmt"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// populationDetSeeds is the full seed count of TestDeterminism_Population, reduced
// by invariantSeeds under -race (plan 2-1 D226). Two seeds, like the other module
// determinism tests: the module draws no number, so seeds differ only in the world.
const populationDetSeeds = 2

// TestDeterminism_Population verifies the cases of TestDeterminism_SameBytes that
// detCases(true) keeps on pack one with the population module on
// (overlay/population-low.toml), seeds 1 to invariantSeeds(2), 400 ticks: a run split
// into Run calls, with a logger, an observer or the audit, ends in the snapshot,
// the state hash and the config hash of the bare Step run, and its events and
// rendering equal the reference's.
//
// Requirement: FR-POP-035, FR-POP-036, FR-OBS-023, FR-WLD-063
//
// Guard: the carries, scales and runs are state, and the counts of the tick are
// scratch; a module that let an observer, a logger or the chunking of Run move
// them would fail here.
func TestDeterminism_Population(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOnePopulation...)
	report := solveCold(t)
	render := func(seed uint64, res result) []byte {
		return renderGolden("one", seed, detTicks, samplesOnly(res, detTicks), report, cfg.Warnings())
	}
	for _, seed := range seedList(invariantSeeds(populationDetSeeds)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			base := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks}))
			ref := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks, Observe: true}))
			refEvents := renderEvents(withoutRunBrackets(ref))
			refGolden := render(seed, ref)
			if d := firstTickDiff(base.Samples, ref.Samples); d != "" {
				t.Fatalf("seed %d: observed Step run differs from the bare one at %s (FR-OBS-023)", seed, d)
			}
			for _, tc := range detCases(true) {
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
				if spec.Observe && len(spec.Drive.chunks) == 0 {
					if d := firstLineDiff(refGolden, render(seed, got)); d != "" {
						t.Errorf("seed %d, case %q: rendering differs from the reference at %s", seed, tc.name, d)
					}
				}
			}
		})
	}
}
