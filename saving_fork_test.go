package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// savingCell is a tick of the saving pack at which a fork is taken, and what it stands for.
type savingCell struct {
	name string
	n    int
}

// savingForkCells probes the saving pack at seed 1 and returns the ticks at which the module
// is in each of the states the fork must carry across a checkpoint (plan 3-sav Phase 7): before the
// first step, while households save, after one is funded, at its request, in the middle of the
// build, at completion and after a repair. The ticks are read from the events of one run, so
// the cells follow the pack and a cell that does not exist fails the test.
func savingForkCells(t testing.TB) []savingCell {
	t.Helper()
	cfg := loadValidPack(t, packOneSaving...)
	res := runPack(t, cfg, runSpec{Seed: 1, Ticks: goldenTicks, Drive: driveStep, Observe: true})
	first, last := map[string]int64{}, map[string]int64{}
	note := func(k string, tick int64) {
		if _, ok := first[k]; !ok {
			first[k] = tick
		}
		last[k] = tick
	}
	for _, e := range res.Events {
		switch v := e.(type) {
		case economysim.SavingPlanned:
			if v.Saving > 0 {
				note("saving", v.Tick)
			}
			if v.Funded > 0 {
				note("funded", v.Tick)
			}
			if v.Builds > 0 {
				note("request", v.Tick)
			}
			if v.Repairs > 0 {
				note("repair request", v.Tick)
			}
		case economysim.FacilityChanged:
			if v.Kind == economysim.FacilityBuilt {
				note("built", v.Tick)
			}
			if v.Kind == economysim.FacilityRepaired {
				note("repaired", v.Tick)
			}
		}
	}
	for _, k := range []string{"saving", "funded", "request", "repair request", "built", "repaired"} {
		if _, ok := first[k]; !ok {
			t.Fatalf("the saving pack at seed 1 has no %q tick in %d ticks, want one for the fork matrix", k, goldenTicks)
		}
	}
	if first["request"] >= first["built"] || last["request"] >= last["built"] {
		t.Fatalf("first request %d, built %d; last request %d, built %d: want each request before its completion so the mid-build cells fall inside a build",
			first["request"], first["built"], last["request"], last["built"])
	}
	mid := (first["request"] + first["built"]) / 2
	return []savingCell{
		{"before the first step", 0},
		{"while saving", int(first["saving"])},
		{"after funding", int(first["funded"])},
		{"at the request", int(first["request"])},
		{"mid-build", int(mid)},
		{"at completion", int(first["built"])},
		{"after a repair", int(first["repaired"])},
		{"at a repair request", int(first["repair request"])},
		{"at the last request", int(last["request"])},
		{"mid-build of the last", int(last["request"]+last["built"]) / 2},
		{"at the last completion", int(last["built"])},
	}
}

// TestFork_Saving verifies the fork-at-N of the saving pack (FR-SAV-026, FR-SAV-040, CHK 8.1): at N
// in {0, while saving, after funding, at the request, mid-build, at completion, after a repair} the
// original and two restores (one a restore of a restore) give the same events, snapshot, audit and
// checkpoint bytes for 30 ticks. The module keeps no state, so a world restored from the checkpoint
// of the other modules must plan, request and count as the original does.
//
// Requirement: FR-SAV-026, FR-SAV-040
//
// Guard: a field of the module that the restore did not rebuild (a table of resolved recipes, a
// scratch count, a cached choice) moves the events or the snapshot rows after the boundary.
func TestFork_Saving(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSaving...)
	cells := savingForkCells(t)
	for _, cell := range cells {
		if cell.n < 0 || int64(cell.n) >= goldenTicks {
			t.Fatalf("cell %q at N %d, want 0 <= N < %d", cell.name, cell.n, goldenTicks)
		}
	}
	for _, cell := range cells {
		t.Run(fmt.Sprintf("%s (N %d)", cell.name, cell.n), func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{
				Name: "saving pack", Cfg: cfg, Seed: 1, N: cell.n, M: forkM, Logs: true,
				Opts: func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} },
			})
		})
	}
}
