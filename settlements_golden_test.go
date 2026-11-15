package economysim_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// fiveSettlements is the warm five with the settlements overlay layered last: the
// module-alone pack of the settlements module (spec-reference-chain.md 4.4.4, 4.5, spec-settlements.md
// 12.4). The pack collapses (criterion 1 of Stage 2), so the golden records the classes
// shrinking, the holds under the dwell and the floor, and no change upwards: the
// staircase up is asserted by TestScenarioSET1_GrowingRegion on a layered world.
var fiveSettlements = append(slices.Clone(warmFive), "overlay/settlements.toml")

// settlementsPosLines is the number of positions that ran per tick with the module on:
// the 22 of the off set and P7.2a.
const settlementsPosLines = 23

// solveSettlements is the steady-state report the golden prints for the pack: the
// solver does not read the class (spec-settlements.md 6.4.4).
func solveSettlements(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, fiveSettlements...))
	if err != nil {
		t.Fatalf("SolveSteadyState(five + settlements) error = %v, want nil", err)
	}
	return newSolveReport(ss)
}

// TestGolden_SettlementsAlone verifies the module-alone golden of the settlements
// module: the warm five with overlay/settlements.toml, seed 1, 400 ticks, run as a Step
// loop with the recorder and WithAudit, against testdata/golden/settlements/five.txt.
// The file is a new set, not a diff of the off five: the maker is gated and the refiner
// bounded, so the households differ from the first change of class.
//
// Requirement: FR-REF-025, FR-SET-013, FR-SET-026, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with
// -update only after the diff is understood (testing.md 7.2).
func TestGolden_SettlementsAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fiveSettlements...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("five", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveSettlements(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("settlements", "five.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("five.txt differs from the golden: %s (testing.md 7.1.2)", settlementsPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "settlements", "five.txt", got)
}

// TestGolden_SettlementsAloneEvents verifies the golden rendering of the events of the
// five with the module on, seed 1, one Run of 400 ticks, against
// testdata/golden/settlements/events-five.txt, and that the Run and a bare Step loop
// end in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-SET-024, FR-SET-025
//
// Guard: the file holds the founded events, the changes down and the held ticks; without
// them the golden would not see the module.
func TestGolden_SettlementsAloneEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fiveSettlements...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEvents(res)
	for _, row := range []string{"SettlementClassChanged", "SettlementHeld"} {
		if !bytes.Contains(got, []byte(row)) {
			t.Errorf("events of the five with the module hold no %s row, want the module's events in the golden", row)
		}
	}
	checkModuleGolden(t, "settlements", "events-five.txt", got)
}

// TestPositionHash_SettlementsGolden verifies the per-position state hash of the five
// with the module on, seed 1, ticks 0 to 29, against
// testdata/golden/settlements/five.positions.txt: one line per position that ran, 23
// per tick (P7.2a is the module's).
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-SET-014
//
// Guard: when the golden fails, the first differing line names the tick and position at
// which the state diverged.
func TestPositionHash_SettlementsGolden(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, fiveSettlements...), goldenSeed, posTicks)
	if want := posTicks * settlementsPosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, settlementsPosLines)
	}
	checkModuleGolden(t, "settlements", "five.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// settlementsPositionsNote replays the run on cfg against the recorded positions file
// of the module, so a failing golden says where the state first moved.
func settlementsPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("settlements", "five.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return "positions of ticks 0 to 29 agree with five.positions.txt: the divergence is later"
}

// TestGolden_SettlementsExercisesTheModule verifies, over seeds 1 to 4, that the pack of
// the golden exercises the module: every run records a founded event per region, at
// least one class that shrank, at least one held change and a region standing on its
// floor, so the file cannot be an empty record of a module that does nothing (plan 3-set
// Phase 7, R2). The pack never records a change upwards: it collapses.
//
// Requirement: FR-SET-010, FR-SET-011, FR-SET-007, FR-SET-024
func TestGolden_SettlementsExercisesTheModule(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fiveSettlements...)
	for seed := uint64(1); seed <= 4; seed++ {
		res := runPack(t, cfg, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
		founded, shrank, held, atFloor := settlementsTally(res)
		if founded != 5 || shrank == 0 || held == 0 || atFloor == 0 {
			t.Errorf("seed %d: founded %d, shrank %d, held %d, regions at the floor %d, want 5 and each of the others above 0", seed, founded, shrank, held, atFloor)
		}
	}
}

// settlementsTally counts the founded events, the changes down and the held ticks of a
// run, and the regions of its final snapshot that stand on their floor.
func settlementsTally(res result) (founded, shrank, held, atFloor int) {
	for _, e := range res.Events {
		switch x := e.(type) {
		case economysim.SettlementClassChanged:
			switch x.Cause {
			case economysim.CauseFounded:
				founded++
			case economysim.CauseShrank:
				shrank++
			}
		case economysim.SettlementHeld:
			held++
		}
	}
	for _, r := range res.Final.Regions {
		if r.ClassAtFloor {
			atFloor++
		}
	}
	return founded, shrank, held, atFloor
}

// TestGolden_SettlementsOffUnmoved verifies that the module leaves no trace while it is
// off: the pack with the module removed and the two role fields cleared hashes as the
// warm five does, and the all-off reference configuration still has its pinned hash
// (N-217). The files of testdata/golden/off/ are the ones of TestGolden_Off, which this
// module's change does not touch.
//
// Requirement: FR-SET-001, FR-CFG-105
//
// Guard: an append that moved the off hash would break the rule of architecture.md
// 2.10.5; it is not a golden to regenerate.
func TestGolden_SettlementsOffUnmoved(t *testing.T) {
	t.Parallel()
	cfg := loadValidPack(t, fiveSettlements...)
	cfg.Modules.Settlements = nil
	for i := range cfg.Roles {
		cfg.Roles[i].Needs, cfg.Roles[i].Ceiling = "", ""
	}
	if got, want := hexHash(cfg), hexHash(loadValidPack(t, warmFive...)); got != want {
		t.Errorf("hash of the pack without the module = %s, want the warm five's %s", got, want)
	}
	if got := hexHash(referenceConfig()); got != pinnedReferenceHash {
		t.Errorf("HashConfig(referenceConfig()) = %s, want the pin %s", got, pinnedReferenceHash)
	}
}

// TestDeterminism_SettlementsAlone verifies the determinism row for the five with the
// module on, seed 1, detTicks ticks: a run split into Run calls, with a logger, an
// observer or the audit, ends in the snapshot and the state hash of the bare Step run,
// and its events equal the reference's (FR-SET-033).
//
// Requirement: FR-SET-033, FR-OBS-023, FR-WLD-063
//
// Guard: the class and its last change are state; a logger, an observer or the
// chunking of Run that moved them would fail here.
func TestDeterminism_SettlementsAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fiveSettlements...)
	seed := uint64(1)
	base := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks}))
	ref := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks, Observe: true}))
	refEvents := renderEvents(withoutRunBrackets(ref))
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
	}
}

// TestFork_SettlementsFive verifies the fork at N of CHK 8.1 on the five with the module
// on (FR-SET-029): the original and two restores give the same events, snapshot, audit
// and checkpoint bytes after N ticks of 0, 60 (inside the hold of the founding), 209 (the
// tick after the first change down) and 250.
//
// Requirement: FR-SET-029, FR-CHK-048
//
// Guard: a restore that lost the tick of the last change would release a held change
// early or late and differ at the first snapshot after it.
func TestFork_SettlementsFive(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, fiveSettlements...)
	for _, n := range []int{0, 60, 209, 250} {
		forkAtN(t, forkCase{Name: fmt.Sprintf("settlements at %d", n), Cfg: cfg, Seed: 3, N: n, M: forkM, Logs: true,
			Opts: func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} }})
	}
}
