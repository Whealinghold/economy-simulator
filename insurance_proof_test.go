package economysim_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// packSecondCover is pack second with the setting-free cover overlay of testdata/chains/second.
var packSecondCover = append(slices.Clone(packSecond), "second/cover.toml")

// TestConfigOnly_InsuranceNeedsOnlyConfiguration verifies architecture.md 9.4 for the module
// (NFR-INS-005): pack second, a setting that shares nothing with the reference chain, takes a
// cellar, a risk row, the credit module and the two cover blocks from a document alone, loads,
// validates, raises no insurance warning, runs 200 ticks for seeds 1 to 2 with the audit on,
// forms store policies, and the premiums of the events are the totals of
// the snapshot. The Go of the test names no commodity, role or region of the module.
//
// Requirement: NFR-INS-005, FR-INS-001, FR-INS-021, FR-INS-022
//
// Guard: a branch on a commodity, recipe or role identifier in the module would fail on a
// setting it never heard of.
func TestConfigOnly_InsuranceNeedsOnlyConfiguration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, packSecondCover...)
	if cfg.Modules.Insurance == nil || !slices.ContainsFunc(cfg.Roles, func(r economysim.Role) bool { return r.Cover != nil && r.Cover.Insure != nil }) {
		t.Fatalf("Modules.Insurance = %+v, want the module on with an insured role", cfg.Modules.Insurance)
	}
	for _, w := range cfg.Warnings() {
		if strings.HasPrefix(w.Rule, "cover") || strings.HasPrefix(w.Rule, "insur") {
			t.Errorf("Warnings() has %s at %v, want no insurance warning on a consistent file", w.Rule, w.Path)
		}
	}
	for _, seed := range seedList(min(2, scenarioSeedCount())) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			res := runPack(t, cfg, runSpec{Seed: seed, Ticks: 200, Drive: driveStep, Observe: true, Audit: true})
			var tl insTally
			tl.add(res.Events)
			if tl.storeFormed == 0 {
				t.Errorf("seed %d: no store policy formed in 200 ticks, want the module to run (the test would pass for nothing)", seed)
			}
			var premiums int64
			for _, e := range res.Events {
				if f, ok := e.(economysim.CoverFormed); ok {
					premiums += int64(f.Premium)
				}
			}
			if got := int64(res.Final.Insurance.Premiums); got != premiums {
				t.Errorf("seed %d: the snapshot totals %d of premiums, the events %d", seed, got, premiums)
			}
		})
	}
}

// TestInvariants_InsurancePacks runs the invariants of testing.md 6.3 over the two packs of the
// module-alone goldens, 100 ticks, seeds 1 to 4, with the audit on: goods and money are conserved
// (the premium and the payout move money between two agents and create none), no balance or stock
// is negative, and a twin world of the same seed agrees after the first step.
//
// Requirement: FR-INS-002, FR-INS-003, FR-INS-006, FR-LED-035
//
// Guard: a payout that left a balance negative, or a premium that moved money to nobody, would
// break the conservation of money at the tick it happened.
func TestInvariants_InsurancePacks(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end: 100 ticks per seed; run without -short")
	}
	t.Parallel()
	for _, g := range []insGolden{insGoldenTwo(t), insGoldenStore(t)} {
		opts := func() []economysim.Option { return g.opts }
		for seed := uint64(1); seed <= uint64(invariantSeeds(4)); seed++ {
			t.Run(fmt.Sprintf("%s seed %d", g.name, seed), func(t *testing.T) {
				t.Parallel()
				runInvariantsWith(t, g.cfg, seed, 100, nil, opts)
			})
		}
	}
}

// TestDeterminism_Insurance verifies, for the two packs of the module-alone goldens, that the final
// state does not depend on the way the run is driven or watched: the cases of detCases(true), seed
// 1, end in the snapshot and state hash of a bare Step run. The module takes no random word, so an
// observer, a logger, a split Run, an audit or a snapshot cannot move a policy (NFR-INS-003).
//
// Requirement: NFR-INS-003, NFR-INS-004, FR-INS-033
//
// Guard: a branch on whether an observer is attached, or a state kept per call of Run, would
// show first in the case that differs by observation or chunking.
func TestDeterminism_Insurance(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, g := range []insGolden{insGoldenTwo(t), insGoldenStore(t)} {
		t.Run(g.name, func(t *testing.T) {
			t.Parallel()
			base := runPack(t, g.cfg, detStep(runSpec{Seed: 1, Ticks: detTicks, Extra: g.opts}))
			for _, tc := range detCases(true) {
				spec := tc.spec()
				spec.Seed, spec.Ticks = 1, detTicks
				spec.Extra = append(slices.Clone(spec.Extra), g.opts...)
				got := runPack(t, g.cfg, spec)
				if !reflect.DeepEqual(got.Final, base.Final) {
					t.Errorf("%s case %q: final snapshot differs from the bare run (seed 1)", g.name, tc.name)
				}
				if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
					t.Errorf("%s case %q: state hash differs from the bare run (seed 1)", g.name, tc.name)
				}
			}
		})
	}
}

// TestConfig_InsuranceOverlaysRefuseBadRows verifies that the cover overlays go through the loader and
// the validator of production, which refuse a bad row: each of the three overlays loads and validates
// over its layers, and the same layers with one row appended that is out of range, names a key the
// block does not have, or names a role the pack lacks fail in config.Parse or Validate.
//
// Requirement: FR-INS-001, FR-INS-021, FR-INS-022, FR-CFG-001
//
// Guard: an overlay that a loader read without checking (an unknown key skipped, a bound not
// tested) would leave a golden that runs on numbers nobody validated.
func TestConfig_InsuranceOverlaysRefuseBadRows(t *testing.T) {
	t.Parallel()
	sets := []struct {
		name   string
		layers []string
	}{{"two", packTwoCover}, {"one-store", packOneCoverStore}, {"second", packSecondCover}}
	for _, s := range sets {
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()
			_ = loadValidPack(t, s.layers...) // the overlay itself is accepted
			bad := map[string]string{
				"unknown key in the module block": "[modules.insurance]\nno_such_key = 1\n",
				"value cap above its bound":       "[modules.insurance]\nvalue_cap_bp = 9223372036854775807\n",
				"negative minimum sum":            "[modules.insurance]\nmin_sum = -1\n",
				"block on a role the pack lacks":  "[role.no_such_role.cover.underwrite]\ncap_bp = 1\n",
			}
			for name, doc := range bad {
				err := loadRefused(s.layers, doc)
				if err == nil {
					t.Errorf("%s: an overlay with %q loaded and validated, want a refusal", s.name, name)
				}
			}
		})
	}
}

// loadRefused layers the files and then doc, and returns the error of the parse or of the validation.
func loadRefused(layers []string, doc string) error {
	docs := make([][]byte, 0, len(layers)+1)
	for _, l := range layers {
		b, err := os.ReadFile(filepath.Join(chainsDir, l))
		if err != nil {
			return fmt.Errorf("read %s: %w", l, err)
		}
		docs = append(docs, b)
	}
	cfg, err := config.Parse(append(docs, []byte(doc))...)
	if err != nil {
		return err
	}
	return cfg.Validate()
}
