package economysim_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// secondRisk is the risk module on pack second (the crater colony), whose commodities, roles and
// region share no word with the reference chain: a barn-like store class for the algae tenders
// and one store row on it. The role's own room is small, so the stock it holds overflows into the
// store. The module needs only configuration (architecture.md 9.4, FR-RSK-039).
const secondRisk = `
[modules.storage.base]
mass = 200
bulk = 200

[modules.storage.role.algae_tender]
stores = [ "cellar" ]

[modules.storage.role.algae_tender.base]
mass = 30
bulk = 30

[modules.storage.class.cellar]
mass = 1000
bulk = 1000

[modules.risk.store.crater_rim.cellar]
strike_bp = 2500
severity_bp = 3000
`

// TestConfigOnly_RiskNeedsOnlyConfiguration verifies architecture.md 9.4 for the module
// (FR-RSK-039): pack second, a setting that shares nothing with the reference chain, takes a
// store class and a risk row from a document alone, loads, validates, raises no risk warning, runs
// 200 ticks for seeds 1 to 2 with the audit on, the store row strikes, and the books of the loss balance every tick. The
// Go of the test names no commodity, role or region of the module.
//
// Requirement: FR-RSK-039, FR-RSK-001, FR-RSK-022
//
// Guard: a branch on a commodity, recipe or role identifier in the module would fail on a
// setting it never heard of.
func TestConfigOnly_RiskNeedsOnlyConfiguration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPackBytes(t, []byte(secondRisk), packSecond...)
	if cfg.Modules.Risk == nil || len(cfg.Modules.Risk.Stores) != 1 || len(cfg.Modules.Risk.Routes) != 0 {
		t.Fatalf("Modules.Risk = %+v, want the module on with one store row", cfg.Modules.Risk)
	}
	for _, w := range cfg.Warnings() {
		if slices.Contains([]string{"no_risk_rows", "risk_edge_off_path", "risk_store_class_unowned", "risk_certain"}, w.Rule) {
			t.Errorf("Warnings() has %s at %v, want no risk warning on a consistent file", w.Rule, w.Path)
		}
	}
	for _, seed := range seedList(min(2, scenarioSeedCount())) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			rw := newRiskWatch(t, cfg, seed, nil)
			if msg := rw.run(t, 200); msg != "" {
				t.Errorf("seed %d: %s, want the books of the loss to balance", seed, msg)
			}
			if rw.tally.storeEvents == 0 {
				t.Errorf("seed %d: no StoreLoss in 200 ticks, want the row to strike the cellars (the test would pass for nothing)", seed)
			}
		})
	}
}

// TestInvariants_RiskPacks runs the invariants of testing.md 6.3 over the two packs of the
// module-alone goldens, 100 ticks, seeds 1 to 4, with the audit on: goods and money are conserved
// but for recipes and the declared sinks (the reason loss is one), no balance or stock is negative,
// and a twin world of the same seed agrees after the first step.
//
// Requirement: FR-RSK-014, FR-RSK-022, FR-RSK-023, FR-LED-035
//
// Guard: a loss that left a goods account negative, or took units that no event or boundary account
// shows, would break the conservation of goods at the tick it happened.
func TestInvariants_RiskPacks(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end: 100 ticks per seed; run without -short")
	}
	t.Parallel()
	for _, g := range []riskGolden{riskGoldenTwo(t), riskGoldenStore(t)} {
		opts := func() []economysim.Option { return g.opts }
		for seed := uint64(1); seed <= uint64(invariantSeeds(4)); seed++ {
			t.Run(fmt.Sprintf("%s seed %d", g.name, seed), func(t *testing.T) {
				t.Parallel()
				runInvariantsWith(t, g.cfg, seed, 100, nil, opts)
			})
		}
	}
}

// TestDeterminism_Risk verifies, for the two packs of the module-alone goldens, that the final
// state does not depend on the way the run is driven or watched: the cases of detCases(true), seed
// 1, end in the snapshot and state hash of a bare Step run. The strikes are keyed draws that
// depend on no other draw, so an observer, a logger, a split Run, an audit or a snapshot cannot
// move one (FR-RSK-033).
//
// Requirement: FR-RSK-033, NFR-RSK-003
//
// Guard: a draw taken only when an observer is attached, or per call of Run, would show first in
// the case that differs by observation or chunking.
func TestDeterminism_Risk(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, g := range []riskGolden{riskGoldenTwo(t), riskGoldenStore(t)} {
		t.Run(g.name, func(t *testing.T) {
			t.Parallel()
			base := runPack(t, g.cfg, detStep(runSpec{Seed: 1, Ticks: detTicks, Extra: g.opts}))
			for _, tc := range detCases(true) {
				spec := tc.spec()
				spec.Seed, spec.Ticks = 1, detTicks
				spec.Extra = append(slices.Clone(spec.Extra), g.opts...)
				got := runPack(t, g.cfg, spec)
				if !reflect.DeepEqual(got.Final, base.Final) {
					t.Errorf("%s case %q: final snapshot differs from the bare run", g.name, tc.name)
				}
				if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
					t.Errorf("%s case %q: state hash differs from the bare run", g.name, tc.name)
				}
			}
		})
	}
}
