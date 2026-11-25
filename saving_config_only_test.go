package economysim_test

import (
	"strconv"
	"testing"
)

// packSecondSaving is the second setting (the crater colony) with its own site class, build and repair recipes
// and a Saver block on the driller: testdata/chains/second/saving.toml, layered after the pack.
var packSecondSaving = append(append([]string(nil), packSecond...), "second/saving.toml")

// TestConfigOnly_SavingBuildsItsOwnCapital verifies FR-SAV-041: a facility chain and a saver role
// written from a file alone, on a setting that shares no commodity, recipe or role with pack one,
// load, validate and run 300 ticks with the audit on, seeds 1 to 3. The drillers that hold no kiln
// save for one, ask for a build, complete it, and no step fails: nothing in the module names the
// identifier of a commodity, a recipe or a role.
//
// Requirement: FR-SAV-041, FR-SAV-019, FR-SAV-021
//
// Guard: a branch on one of the first pack's identifiers in the saver's rules would not find the
// kiln, the recipes or the role on another setting, and the household would never save or build.
func TestConfigOnly_SavingBuildsItsOwnCapital(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	cfg := loadValidPack(t, packSecondSaving...)
	for _, seed := range seedList(min(3, scenarioSeedCount())) {
		t.Run("seed "+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			res := runPack(t, cfg, runSpec{Seed: seed, Ticks: 300, Drive: driveStep, Observe: true, Audit: true})
			p := savingProbeOf(res.Events)
			if p.Saving == 0 || p.Builds == 0 || p.Built == 0 {
				t.Errorf("seed %d: %+v, want households saving, a build request and a built facility on the second setting", seed, p)
			}
		})
	}
}
