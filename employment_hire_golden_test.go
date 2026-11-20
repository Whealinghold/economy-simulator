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

// The module-alone golden of the employment module in its hire form (plan 3-emp Phase 8, testing.md
// 7.1.1): the pack one-demand with overlay/employment-hire.toml, which carries its own census, so it
// is the module's pack and not the reference chain with an overlay. The files are in
// testdata/golden/employment_hire, beside the payroll form's testdata/golden/employment, which they
// do not touch. The default strategy offers and posts, so hires form without a scripted consumer.

// employmentHirePosLines is the number of positions that ran per tick in pack one-demand with the
// hire form on: the 25 of one-demand with the payroll form (P4.1b is a position of the
// module in both forms) and P2.5a and P2.8b.
const employmentHirePosLines = 27

// packOneDemandHireCold is the layer set of packOneDemandHire without the warm start.
var packOneDemandHireCold = slices.DeleteFunc(slices.Clone(packOneDemandHire), func(s string) bool { return s == "warm-one-demand.toml" })

// solveColdHire solves the cold layer set of one-demand with the hire overlay (no warm start). The
// overlay moves no recipe, so the census is that of one-demand; the report is the one the golden prints.
func solveColdHire(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, packOneDemandHireCold...))
	if err != nil {
		t.Fatalf("SolveSteadyState(cold one-demand + hire overlay) error = %v, want nil", err)
	}
	return newSolveReport(ss)
}

// TestGolden_EmploymentHireAlone verifies the module-alone golden of the hire form: the pack
// one-demand with overlay/employment-hire.toml, seed 1, 400 ticks, run as a Step loop with the
// recorder and WithAudit, against testdata/golden/employment_hire/one-demand.txt.
//
// Requirement: FR-REF-025, FR-REF-026, FR-EMP-001, FR-EMP-036, FR-EMP-039, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with -update only after
// the diff is understood (testing.md 7.2). DELIBERATE NEW GOLDEN: the files are new, no existing
// golden moves.
func TestGolden_EmploymentHireAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemandHire...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("one-demand", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveColdHire(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("employment_hire", "one-demand.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one-demand.txt differs from the golden: %s (testing.md 7.1.2)", employmentHirePositionsNote(t, cfg))
	}
	checkModuleGolden(t, "employment_hire", "one-demand.txt", got)
}

// TestGolden_EmploymentHireAloneEvents verifies the golden rendering of the events of one-demand
// with the hire form on, seed 1, one Run of 400 ticks, against
// testdata/golden/employment_hire/events-one-demand.txt, that the Run and a bare Step loop end in the
// same snapshot, and that the events hold the module's three events.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-EMP-053, FR-EMP-056, FR-EMP-057
//
// Guard: the file holds HireFormed, HireEnded and LabourRound rows; without them the golden would not
// see the module.
func TestGolden_EmploymentHireAloneEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemandHire...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEvents(res)
	for _, name := range []string{"HireFormed", "HireEnded", "LabourRound"} {
		if !bytes.Contains(got, []byte(name)) {
			t.Errorf("events of one-demand with the hire form hold no %s, want the module's event in the golden", name)
		}
	}
	checkModuleGolden(t, "employment_hire", "events-one-demand.txt", got)
}

// TestPositionHash_EmploymentHireGolden verifies the per-position state hash of one-demand with the
// hire form on, seed 1, ticks 0 to 29, against testdata/golden/employment_hire/one-demand.positions.txt:
// one line per position that ran, 27 per tick (P2.5a and P2.8b run with the form on).
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-EMP-001
//
// Guard: when the golden fails, the first differing line names the tick and position at which the
// state diverged.
func TestPositionHash_EmploymentHireGolden(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, packOneDemandHire...), goldenSeed, posTicks)
	if want := posTicks * employmentHirePosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, employmentHirePosLines)
	}
	checkModuleGolden(t, "employment_hire", "one-demand.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// employmentHirePositionsNote replays the scripted run on cfg against the recorded positions file of
// the hire form, so a failing golden says where the state first moved.
func employmentHirePositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("employment_hire", "one-demand.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with one-demand.positions.txt: the divergence is later", posTicks-1)
}

// TestGolden_EmploymentHireProbeExercisesHires verifies that the pack of the golden is not empty for
// the module: over seeds 1 to 4 and 400 ticks it forms hires, and every hire ends. The pack collapses
// (N-352), so a golden could end before a hire formed; the probe fails the golden's pack, not the
// golden, and the plan's fallback is the first of the packs payroll and five that forms one (D15).
//
// Requirement: FR-REF-025, FR-EMP-024, FR-EMP-036
//
// Guard: an overlay whose numbers drift so that no hire forms would leave a golden that sees nothing
// of the module and still passes.
func TestGolden_EmploymentHireProbeExercisesHires(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemandHire...)
	for _, seed := range seedList(4) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			res := runPack(t, cfg, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveStep, Observe: true})
			var formed, ended int
			for _, e := range res.Events {
				switch e.(type) {
				case economysim.HireFormed:
					formed++
				case economysim.HireEnded:
					ended++
				}
			}
			if formed == 0 {
				t.Errorf("seed %d: no HireFormed in %d ticks, want at least one (move the golden to the pack payroll or five, D15)", seed, goldenTicks)
			}
			if ended > formed {
				t.Errorf("seed %d: %d HireEnded for %d HireFormed, want no more ends than formations", seed, ended, formed)
			}
		})
	}
}
