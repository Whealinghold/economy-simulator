package economysim_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// tenureWorldSeeds is the full seed count of TestConfigOnly_TenureOverlayBuildsAndRuns,
// reduced by invariantSeeds under -race.
const tenureWorldSeeds = 2

// TestConfigOnly_TenureOverlayBuildsAndRuns verifies, as a consumer would use it, that a
// world built from the files alone (pack one, overlay/resources.toml, overlay/tenure.toml,
// no Go code of the module) is accepted by NewWorld and runs detTicks ticks under the audit
// without an error, that the tenure module is on and its rows reach the world's
// configuration, that the run is the same on a second build (the final snapshot and the
// state hash), and that the module's configuration is part of the configuration hash.
// The module does nothing at run time yet (plan 3-lnd, phase 3): its ration, rent and
// titles come in the next pull requests, whose tests compare the results with this run.
//
// Requirement: FR-LND-001, FR-LND-038, FR-CFG-085
//
// Guard: a module that is configurable but breaks construction, the audit or
// determinism of a world would fail only here, since every other test of this change
// stops at Validate.
func TestConfigOnly_TenureOverlayBuildsAndRuns(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	layers := append(slices.Clone(packOneResources), "overlay/tenure.toml")
	cfg := loadValidPack(t, layers...)
	if cfg.Modules.Tenure == nil || len(cfg.Modules.Tenure.Holdings) != 2 {
		t.Fatalf("Modules.Tenure of the overlay = %+v, want the module on with two holdings", cfg.Modules.Tenure)
	}
	if economysim.HashConfig(cfg) == economysim.HashConfig(loadValidPack(t, packOneResources...)) {
		t.Errorf("HashConfig(with tenure) equals that of pack one with resources alone, want them to differ")
	}
	for _, seed := range seedList(invariantSeeds(tenureWorldSeeds)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			spec := detStep(runSpec{Seed: seed, Ticks: detTicks, Audit: true})
			a := runPack(t, cfg, spec)
			b := runPack(t, cfg, spec)
			if a.Final.Tick != detTicks {
				t.Errorf("seed %d: final tick = %d, want %d", seed, a.Final.Tick, detTicks)
			}
			if !reflect.DeepEqual(a.Final, b.Final) {
				t.Errorf("seed %d: final snapshots of two builds differ", seed)
			}
			if economysim.HashWorld(a.World) != economysim.HashWorld(b.World) {
				t.Errorf("seed %d: state hashes of two builds differ", seed)
			}
		})
	}
}

// secondTenure is a document of titles over pack second with the resources document
// secondResources: the vats of the crater rim are let at a rent of 3, the algae tenders hold
// 10 basis points of the vats each and the pool of the rim 2 000 more, with a step on the rent.
// The crater colony shares no word with the reference chain, so the module needs only
// configuration (architecture.md 9.4).
const secondTenure = `
[modules.tenure]
rent_step_bp = 3000

[modules.tenure.term.crater_rim.vat_volume]
rent = 3

[modules.tenure.holding.crater_rim.vat_volume.algae_tender]
share_bp = 10

[modules.tenure.holding.crater_rim.vat_volume.pool]
share_bp = 2000
`

// TestConfigOnly_TenureNeedsOnlyConfiguration verifies architecture.md 9.4 for the module
// (FR-LND-042): pack second, a setting that shares nothing with the reference chain, takes
// titles, a rent and a step from a document alone, loads, validates, runs 200 ticks for
// seeds 1 to 2 with the audit on, and the snapshot row of the vats carries the titles. The
// Go of the test names no commodity, recipe or role of the module.
//
// Requirement: FR-LND-042, FR-LND-001
//
// Guard: a branch on a commodity, recipe or role identifier in the module would fail on a
// setting it never heard of.
func TestConfigOnly_TenureNeedsOnlyConfiguration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPackBytes(t, []byte(secondResources+secondTenure), packSecond...)
	if cfg.Modules.Tenure == nil || len(cfg.Modules.Tenure.Holdings) != 2 {
		t.Fatalf("Modules.Tenure = %+v, want the module on with two holdings", cfg.Modules.Tenure)
	}
	for _, seed := range seedList(min(2, scenarioSeedCount())) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			w := newPackWorld(t, cfg, seed, economysim.WithAudit())
			for i := range 200 {
				if err := w.Step(); err != nil {
					t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, i, err)
				}
			}
			s, err := w.Snapshot()
			if err != nil {
				t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
			}
			var titled int
			for _, e := range s.Endowments {
				if e.Holders > 0 {
					titled++
				}
			}
			if titled != 1 {
				t.Errorf("seed %d: %d endowment rows with titles, want 1 (the vats)", seed, titled)
			}
		})
	}
}
