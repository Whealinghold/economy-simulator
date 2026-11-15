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
