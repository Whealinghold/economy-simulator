package economysim_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// employmentPosLines is the number of positions that ran per tick in pack
// one-demand with the employment module on: the 24 of one-demand and P4.1b.
const employmentPosLines = 25

// solveColdEmployment solves the cold layer set of one-demand with the overlay of the
// employment module (no warm start). The overlay moves no recipe, so the census is
// that of one-demand; the report is the one the golden prints.
func solveColdEmployment(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, packOneDemandEmploymentCold...))
	if err != nil {
		t.Fatalf("SolveSteadyState(cold one-demand + employment) error = %v, want nil", err)
	}
	return newSolveReport(ss)
}

// TestGolden_EmploymentAlone verifies the module-alone golden of the employment
// module in its payroll form: the pack one-demand with overlay/employment.toml (the
// pack carries its own census, so it is the module's pack, not the reference chain
// with an overlay: testing.md 7.1.1), seed 1, 400 ticks, run as a Step loop with the
// recorder and WithAudit, against testdata/golden/employment/one-demand.txt.
//
// Requirement: FR-REF-025, FR-REF-026, FR-EMP-064, FR-EMP-072, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with
// -update only after the diff is understood (testing.md 7.2).
func TestGolden_EmploymentAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemandEmployment...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("one-demand", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveColdEmployment(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("employment", "one-demand.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one-demand.txt differs from the golden: %s (testing.md 7.1.2)", employmentPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "employment", "one-demand.txt", got)
}

// TestGolden_EmploymentAloneEvents verifies the golden rendering of the events of
// one-demand with the module on, seed 1, one Run of 400 ticks, against
// testdata/golden/employment/events-one-demand.txt, and that the Run and a bare Step
// loop end in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-EMP-076
//
// Guard: the file holds PayrollPaid rows; without them the golden would not see the
// module's event.
func TestGolden_EmploymentAloneEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemandEmployment...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEvents(res)
	if !bytes.Contains(got, []byte("PayrollPaid")) {
		t.Errorf("events of one-demand with the module hold no PayrollPaid, want the module's event in the golden")
	}
	checkModuleGolden(t, "employment", "events-one-demand.txt", got)
}

// TestPositionHash_EmploymentGolden verifies the per-position state hash of
// one-demand with the module on, seed 1, ticks 0 to 29, against
// testdata/golden/employment/one-demand.positions.txt: one line per position that
// ran, 25 per tick (P4.1b runs with the module on, and not in the one-demand set).
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-EMP-064
//
// Guard: when the golden fails, the first differing line names the tick and position
// at which the state diverged.
func TestPositionHash_EmploymentGolden(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, packOneDemandEmployment...), goldenSeed, posTicks)
	if want := posTicks * employmentPosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, employmentPosLines)
	}
	checkModuleGolden(t, "employment", "one-demand.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// employmentPositionsNote replays the scripted run on cfg against the recorded
// positions file of the module, so a failing golden says where the state first moved.
func employmentPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("employment", "one-demand.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with one-demand.positions.txt: the divergence is later", posTicks-1)
}

// TestFork_Employment verifies that a world with the payroll form on, forked at a
// boundary, runs on exactly as an uninterrupted one in every event, in the snapshot,
// in the checkpoint bytes and in the log records. The module keeps no state and saves
// no section (FR-EMP-074), so the fork is bit-identical by construction; the test
// proves it, at tick 0, early in the warm start, and after the money has circulated
// for 150 ticks.
//
// Setup: pack one-demand with overlay/employment.toml, seed 3, audit on. Inspect
// requires PayrollPaid events after the boundary, so the fork carries a running
// payroll and not an idle module.
//
// Requirement: FR-EMP-074, FR-CHK-001
func TestFork_Employment(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemandEmployment...)
	for _, n := range []int{0, 7, 150} {
		forkAtN(t, forkCase{Name: fmt.Sprintf("employment at %d", n), Cfg: cfg, Seed: 3, N: n, M: forkM, Logs: true,
			Opts: func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} },
			Inspect: func(t *testing.T, restored, since []economysim.Event) {
				paid := 0
				for _, e := range since {
					if _, ok := e.(economysim.PayrollPaid); ok {
						paid++
					}
				}
				if paid == 0 {
					t.Errorf("employment at %d: no PayrollPaid after the boundary, want some so the fork carries the payroll", n)
				}
			}})
	}
}

// TestEmployment_OffMovesNothing verifies FR-EMP-064: the same pack without the
// overlay emits no PayrollPaid and has no wage_pay flow in its ledger-flow rows, and
// the payroll's money is an internal transfer, so a run with the module on has no
// BoundaryFlow row of the reason wage_pay either.
//
// Requirement: FR-EMP-064, FR-EMP-072
func TestEmployment_OffMovesNothing(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, tc := range []struct {
		name   string
		layers []string
		want   bool
	}{{"off", packOneDemand, false}, {"on", packOneDemandEmployment, true}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := runPack(t, loadValidPack(t, tc.layers...), runSpec{Seed: 1, Ticks: 60, Drive: driveStep, Observe: true, Audit: true})
			var payroll, flow int
			for _, e := range res.Events {
				switch ev := e.(type) {
				case economysim.PayrollPaid:
					payroll++
				case economysim.BoundaryFlow:
					if ev.Reason == economysim.ReasonWagePay {
						flow++
					}
				}
			}
			if (payroll > 0) != tc.want {
				t.Errorf("PayrollPaid events = %d, want some = %v", payroll, tc.want)
			}
			if flow != 0 {
				t.Errorf("BoundaryFlow rows of wage_pay = %d, want 0: it is no faucet and no sink", flow)
			}
		})
	}
}

// TestDeterminism_Employment verifies the cases of TestDeterminism_SameBytes that
// detCases(true) keeps, on one-demand with the payroll on, seeds 1 to invariantSeeds(2),
// 400 ticks: a run split into Run calls, with a logger, an observer or the audit, ends in
// the snapshot, the state hash and the config hash of the bare Step run, and the module
// takes no word of the stream (FR-EMP-075).
//
// Requirement: FR-EMP-075, FR-OBS-023, FR-WLD-063
//
// Guard: an observer or logger that moved the payroll's result, or a draw taken by the
// position, would fail here.
func TestDeterminism_Employment(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemandEmployment...)
	for _, seed := range seedList(invariantSeeds(2)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
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
		})
	}
}
