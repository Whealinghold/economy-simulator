package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packForkCells are the boundaries of TestFork_InfrastructurePack: the ticks run before it and
// the call made on the original there (nil: none).
func packForkCells(t *testing.T, cfg economysim.Config) []struct {
	name   string
	n      int
	before func(*economysim.World)
} {
	t.Helper()
	build := func(w *economysim.World) {
		cook := capAddCook(t, w, cfg, 0)
		capDirect(t, w, cook, capRecipe(cfg, "build_mill"), 1)
	}
	adjust := func(w *economysim.World) {
		rows, err := w.Facilities(0)
		if err != nil || len(rows) == 0 {
			t.Fatalf("Facilities(0) = %d rows, %v, want the initial mills and nil", len(rows), err)
		}
		if err := w.AdjustFacilities([]economysim.FacilityAdjustment{{ID: rows[0].ID, Delta: -400}}); err != nil {
			t.Fatalf("AdjustFacilities error = %v, want nil", err)
		}
	}
	return []struct {
		name   string
		n      int
		before func(*economysim.World)
	}{
		{"the first boundary", 0, nil},
		{"a build told at tick 0", 0, build},
		{"after an adjustment", 40, adjust},
		{"in the ruins of the mills", 150, nil},
		{"a build told among the ruins", 150, build},
	}
}

// TestFork_InfrastructurePack verifies that a world of pack one with overlay/infrastructure.toml
// (60 mills decaying, worn by the cooks and ruined from about tick 100) forked at the boundaries
// of packForkCells runs on exactly as an uninterrupted one: the same events, snapshot, audit
// and checkpoint bytes, and so does a restore of the restore. The cells are the first boundary,
// a build told to a directed cook there (the job and the record not yet made), an adjustment
// of a facility's condition between ticks, and two boundaries among the ruins, the last with a
// build told there. TestScenarioS125_Checkpoint takes the checkpoint with a build in
// progress.
//
// Requirement: FR-CAP-041
//
// Guard: the records, the tables and the per-tick counters of the module are rebuilt, not
// saved, at a restore; a rebuilt value that drifted from a run's would show here in a world
// with real decay, wear and ruin.
func TestFork_InfrastructurePack(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	cfg := loadValidPack(t, packOneInfrastructure...)
	opts := func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} }
	for _, c := range packForkCells(t, cfg) {
		forkAtN(t, forkCase{Name: "infrastructure pack " + c.name, Cfg: cfg, Seed: 1, N: c.n, M: forkM, Logs: true, Opts: opts, Before: c.before})
	}
}
