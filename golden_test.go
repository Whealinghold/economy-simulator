package economysim_test

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// goldenTicks and goldenSeed are the length and seed REF 8.2 fixes for the golden.
const (
	goldenTicks int64  = 400
	goldenSeed  uint64 = 1
)

// solveCold solves the cold layer set (base, one: no warm start), as FR-REF-007
// solves "the layered configuration without" the warm start, and adapts the
// report to the renderer (the one place a rename of the solver report is edited).
func solveCold(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, packOneCold...))
	if err != nil {
		t.Fatalf("SolveSteadyState(cold one) error = %v, want nil", err)
	}
	return newSolveReport(ss)
}

// samplesOnly returns res with only the samples of REF 8.5 (a snapEvery run keeps
// every tick), so a per-tick run renders the same one.txt as a snapSample run.
func samplesOnly(res result, ticks int64) result {
	out := res
	out.Samples = map[int64]economysim.Snapshot{}
	for tick, s := range res.Samples {
		if (runSpec{Ticks: ticks, Snapshots: snapSample}).isSample(tick) {
			out.Samples[tick] = s
		}
	}
	return out
}

// oneTxt renders one.txt for a pack-one run.
func oneTxt(cfg economysim.Config, seed uint64, ticks int64, res result, report solveReport) []byte {
	return renderGolden("one", seed, ticks, samplesOnly(res, ticks), report, cfg.Warnings())
}

// TestGolden_OneOff verifies the end-to-end golden of pack one (the three layers),
// seed 1, 400 ticks, run as a Step loop with the recorder and WithAudit, against
// testdata/golden/off/one.txt (REF 8.5 items 1 to 5).
//
// Requirement: FR-REF-029, FR-REF-025, FR-REF-026, FR-OBS-023
//
// Guard: a behaviour change of any row moves the file; it is regenerated with
// -update only after the diff is understood (testing.md 7.2).
func TestGolden_OneOff(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample})
	got := oneTxt(cfg, goldenSeed, goldenTicks, res, solveCold(t))
	if want, err := os.ReadFile(goldenPath("one.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one.txt differs from the golden: %s (testing.md 7.1.2)", positionsNote(t, cfg))
	}
	checkGolden(t, "one.txt", got)
}

// TestGolden_OneEvents verifies the golden rendering of the events of pack one,
// seed 1, one Run of 400 ticks (RunStarted and RunEnded included), against
// testdata/golden/off/events-one.txt (OBS 9.5.3), and that this Run and a bare
// Step loop end in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-REF-029, FR-REF-025
//
// Guard: the two runs of the golden cannot be one (Snapshot is not callable from
// an observer), so their final snapshots are compared here.
func TestGolden_OneEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	var started, ended int
	for _, e := range res.Events {
		switch eventType(e) {
		case "RunStarted":
			started++
		case "RunEnded":
			ended++
		}
	}
	if started != 1 || ended != 1 {
		t.Errorf("RunStarted = %d, RunEnded = %d, want 1 and 1", started, ended)
	}
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	checkGolden(t, "events-one.txt", renderEvents(res))
}
