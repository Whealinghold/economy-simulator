package economysim_test

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// fiveInstitutions is the warm five with the polity overlay and then the institutions
// overlay layered last: the module-alone pack of the institutions module
// (spec-reference-chain.md 4.5, spec-institutions.md 13.5). The polity overlay is the
// ungoverned one, which adds no account and no agent, so the pack differs from the off
// five by the module's own outputs and by what a body, a right and a control do.
var fiveInstitutions = append(slices.Clone(fivePolity), "overlay/institutions.toml")

// institutionsPosLines is the number of positions that ran per tick with the module on:
// the 22 of the off set, the polity's one, P1.9a and P7.2b.
const institutionsPosLines = 25

// solveInstitutions is the steady-state report the golden prints for the pack: the solver
// does not read a body, a right or a control (spec-institutions.md 6.5).
func solveInstitutions(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, fiveInstitutions...))
	if err != nil {
		t.Fatalf("SolveSteadyState(five + institutions) error = %v, want nil", err)
	}
	return newSolveReport(ss)
}

// TestGolden_InstitutionsAlone verifies the module-alone golden of the institutions module:
// the warm five with the polity and institutions overlays, seed 1, 400 ticks, run as a Step
// loop with the recorder and WithAudit, against testdata/golden/institutions/five.txt. The
// file is a new set, not a diff of the off five: the guild's places, the produce right and
// the two controls change what the cooks of the centre and the staple markets do.
//
// Requirement: FR-REF-025, FR-ORG-001, FR-ORG-044, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with
// -update only after the diff is understood (testing.md 7.2).
func TestGolden_InstitutionsAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fiveInstitutions...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("five", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveInstitutions(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("institutions", "five.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("five.txt differs from the golden: %s (testing.md 7.1.2)", institutionsPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "institutions", "five.txt", got)
}

// TestGolden_InstitutionsAloneEvents verifies the golden rendering of the events of the
// five with the module on, seed 1, one Run of 400 ticks, against
// testdata/golden/institutions/events-five.txt, and that the Run and a bare Step loop end
// in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-ORG-039, FR-ORG-040, FR-ORG-041
//
// Guard: the file holds the membership, refusal and price-control rows; without them the
// golden would not see the module.
func TestGolden_InstitutionsAloneEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fiveInstitutions...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEvents(res)
	for _, row := range []string{"MembershipChanged", "PriceControlled"} {
		if !bytes.Contains(got, []byte(row)) {
			t.Errorf("events of the five with the module hold no %s row, want the module's events in the golden", row)
		}
	}
	checkModuleGolden(t, "institutions", "events-five.txt", got)
}

// TestPositionHash_InstitutionsGolden verifies the per-position state hash of the five with
// the module on, seed 1, ticks 0 to 29, against testdata/golden/institutions/five.positions.txt:
// one line per position that ran, 25 per tick (P1.9a and P7.2b are the module's, one is the polity's).
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-WLD-040, FR-ORG-013
//
// Guard: when the golden fails, the first differing line names the tick and position at
// which the state diverged.
func TestPositionHash_InstitutionsGolden(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, fiveInstitutions...), goldenSeed, posTicks)
	if want := posTicks * institutionsPosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, institutionsPosLines)
	}
	checkModuleGolden(t, "institutions", "five.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// institutionsPositionsNote replays the run on cfg against the recorded positions file of
// the module, so a failing golden says where the state first moved.
func institutionsPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("institutions", "five.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return "positions of ticks 0 to 29 agree with five.positions.txt: the divergence is later"
}

// institutionsTally counts the events of a run that show the module at work: the
// membership events that carry households waiting, the rights that refused something and
// the controls that replaced a limit.
func institutionsTally(res result) (waiting, refused, controlled int) {
	for _, e := range res.Events {
		switch x := e.(type) {
		case economysim.MembershipChanged:
			if x.Waiting > 0 {
				waiting++
			}
		case economysim.RightRefused:
			if x.Produce+x.Sell+x.Buy+x.Train > 0 {
				refused++
			}
		case economysim.PriceControlled:
			if x.Bids+x.Asks > 0 {
				controlled++
			}
		}
	}
	return waiting, refused, controlled
}

// TestGolden_InstitutionsExercisesTheModule verifies, over seeds 1 to 4, that the pack of
// the golden exercises the module: every run records a MembershipChanged with households
// waiting, a RightRefused with a count and a PriceControlled with a clamped order, so the
// file cannot be an empty record of a module that does nothing (plan 3-org Phase 10, R5).
//
// Requirement: FR-ORG-006, FR-ORG-024, FR-ORG-030, FR-ORG-039, FR-ORG-040, FR-ORG-041
func TestGolden_InstitutionsExercisesTheModule(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fiveInstitutions...)
	for seed := uint64(1); seed <= 4; seed++ {
		res := runPack(t, cfg, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
		waiting, refused, controlled := institutionsTally(res)
		t.Logf("seed %d: waiting rows %d, refusal rows %d, control rows %d", seed, waiting, refused, controlled)
		if waiting == 0 || refused == 0 || controlled == 0 {
			t.Errorf("seed %d: waiting %d, refused %d, controlled %d, want each above 0", seed, waiting, refused, controlled)
		}
	}
}

// TestGolden_InstitutionsOffLeavesNoTrace verifies that the module leaves no trace while it
// is off: the pack with the module removed and the cook's course (the one role field the overlay changes) cleared hashes as the
// polity five does, and the all-off reference configuration still has its pinned hash
// (N-217). The files of testdata/golden/off/ are the ones of TestGolden_Off, which this
// module's change does not touch.
//
// Requirement: FR-ORG-001, FR-CFG-105
//
// Guard: an append that moved the off hash would break the rule of architecture.md
// 2.10.5; it is not a golden to regenerate.
func TestGolden_InstitutionsOffLeavesNoTrace(t *testing.T) {
	t.Parallel()
	cfg := loadValidPack(t, fiveInstitutions...)
	cfg.Modules.Institutions = nil
	for i := range cfg.Roles {
		if cfg.Roles[i].ID == "cook" {
			cfg.Roles[i].TrainingTicks = 0
		}
	}
	if got, want := hexHash(cfg), hexHash(loadValidPack(t, fivePolity...)); got != want {
		t.Errorf("hash of the pack without the module = %s, want the polity five's %s", got, want)
	}
	if got := hexHash(referenceConfig()); got != pinnedReferenceHash {
		t.Errorf("HashConfig(referenceConfig()) = %s, want the pin %s", got, pinnedReferenceHash)
	}
}
