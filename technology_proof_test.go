package economysim_test

import (
	"fmt"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestDeterminism_Technology verifies the cases of TestDeterminism_SameBytes that
// detCases(true) keeps on pack three with the four-mechanism overlay, seeds 1 and 2, 400
// ticks: a run split into Run calls, with a logger, an observer or the audit, ends in the
// snapshot, the state hash and the config hash of the bare Step run, and its events and
// rendering equal the reference's. The 400 ticks hold learning, trend, discoveries and
// catch-up adds, so every part of the step is under it.
//
// Requirement: NFR-TEC-001, FR-TEC-033, FR-CFG-026, FR-OBS-023, FR-WLD-063
//
// Guard: the step draws no random number and reads no wall clock; a draw or a dependence
// on the observer would fail here.
func TestDeterminism_Technology(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, technologyThree...)
	report := solveTechnology(t)
	render := func(seed uint64, res result) []byte {
		return renderGolden("three", seed, detTicks, samplesOnly(res, detTicks), report, cfg.Warnings())
	}
	for _, seed := range seedList(min(2, scenarioSeedCount())) {
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

// TestConfigOnly_TechnologyNeedsOnlyConfiguration verifies architecture.md 9.4 for the
// module: pack second, a setting that shares nothing with the reference chain, is given a
// technology field by data alone, naming the first recipe of its first role, and runs 400
// ticks for seeds 1 to 3 with the audit; the field learns (a row with experience and a gain
// above 0). The test names no commodity, recipe or role.
//
// Requirement: FR-TEC-002, FR-TEC-003, FR-REF-031
//
// Guard: a branch on a commodity, recipe or role identifier in the module would fail on a
// setting it never heard of.
func TestConfigOnly_TechnologyNeedsOnlyConfiguration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, packSecond...)
	cfg.Modules.Technology = &economysim.Technology{Fields: []economysim.TechField{
		{ID: "craft", Recipes: cfg.Roles[0].Recipes[:1], CapBP: 3_000, Base: 10, DoublingBP: 300},
	}}
	for _, seed := range seedList(min(3, scenarioSeedCount())) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			w := newPackWorld(t, cfg, seed, economysim.WithAudit())
			for i := range 400 {
				if err := w.Step(); err != nil {
					t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, i, err)
				}
			}
			s, err := w.Snapshot()
			if err != nil {
				t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
			}
			if len(s.Tech) == 0 || s.Tech[0].Experience == 0 || s.Tech[0].GainBP == 0 {
				t.Errorf("seed %d: Tech rows after 400 ticks = %+v, want a field that learned", seed, s.Tech)
			}
		})
	}
}
