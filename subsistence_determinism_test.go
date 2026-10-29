package economysim_test

import (
	"fmt"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// subsistenceDetSeeds is the full seed count of TestDeterminism_Subsistence, reduced
// by invariantSeeds under -race (plan 2-1 D226). Two seeds, like the other module
// determinism tests: the module draws no number, so seeds differ only in the world.
const subsistenceDetSeeds = 2

// TestDeterminism_Subsistence verifies the cases of TestDeterminism_SameBytes that
// detCases(true) keeps on pack one with the subsistence module on
// (overlay/own-use.toml), seeds 1 to invariantSeeds(2), 400 ticks: a run split
// into Run calls, with a logger, an observer or the audit, ends in the snapshot,
// the state hash and the config hash of the bare Step run, and its events and
// rendering equal the reference's.
//
// Requirement: FR-SUB-035, FR-SUB-045, FR-OBS-023, FR-WLD-063
//
// Guard: the flags, counters and runs are state, and the counts of the tick are
// scratch; a module that let an observer, a logger or the chunking of Run move
// them would fail here.
func TestDeterminism_Subsistence(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSubsistence...)
	report := solveCold(t)
	render := func(seed uint64, res result) []byte {
		return renderGolden("one", seed, detTicks, samplesOnly(res, detTicks), report, cfg.Warnings())
	}
	for _, seed := range seedList(invariantSeeds(subsistenceDetSeeds)) {
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

// secondSubsistence is the subsistence module as a document over pack second, the
// crater colony: its households eat 2 biomass and 1 water a tick, and two sited
// source recipes, forage_algae and skim_water, make exactly that in the rim at a
// factor of 10 000 (2 * 10 000 / (1 * 2) and 1 * 10 000 / (1 * 1): one tick of
// upkeep for one tick of work). No word of the first consumer's setting is in it.
const secondSubsistence = `
[recipe.forage_algae]
duration = 1
sited = true
outputs = [ { commodity = "biomass", quantity = 2, cost_share = 10000 } ]
[recipe.skim_water]
duration = 1
sited = true
outputs = [ { commodity = "water", quantity = 1, cost_share = 10000 } ]

[region.crater_rim.yield.forage_algae]
all = 10000
[region.crater_rim.yield.skim_water]
all = 10000

[modules.subsistence]
recipes = [ "forage_algae", "skim_water" ]
`

// subsistenceEntered returns the households that entered the mode in the events.
func subsistenceEntered(events []economysim.Event) (n int64) {
	for _, e := range events {
		if o, ok := e.(economysim.OwnUseChanged); ok {
			n += o.Entered
		}
	}
	return n
}

// TestSubsistence_ConfigOnlyPack verifies architecture.md 9.4 for the module: pack
// one with overlay/own-use.toml, and the crater colony of pack second with the
// document above, build from files alone, validate, hold the invariants of
// testing.md 6.3 and the audit for 120 ticks and put households in the mode. The
// second setting names no good, recipe or role of the first.
//
// Requirement: FR-SUB-047
func TestSubsistence_ConfigOnlyPack(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, p := range []struct {
		name string
		cfg  economysim.Config
	}{
		{"one", loadValidPack(t, packOneSubsistence...)},
		{"second", loadPackBytes(t, []byte(secondSubsistence), packSecond...)},
	} {
		for _, seed := range seedList(min(2, scenarioSeedCount())) {
			t.Run(fmt.Sprintf("%s seed %d", p.name, seed), func(t *testing.T) {
				t.Parallel()
				runInvariants(t, p.cfg, seed, 120, nil)
				rec := &recorder{}
				w, err := economysim.NewWorld(p.cfg, economysim.WithSeed(seed), economysim.WithObserver(rec))
				if err != nil {
					t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
				}
				step(t, w, 120)
				if n := subsistenceEntered(rec.events); n == 0 {
					t.Errorf("%s seed %d: households entered the mode = %d over 120 ticks, want some so the overlay is exercised", p.name, seed, n)
				}
			})
		}
	}
}
