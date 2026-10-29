package economysim_test

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packOneSubsistence is pack one with the subsistence module alone on: the three
// source recipes of overlay/own-use.toml (spec-subsistence.md 16, plan 2-10 D416).
// The module has no census of its own, so its module-alone golden is the
// reference chain with the overlay, as testing.md 7.1.1 says.
var packOneSubsistence = append(slices.Clone(packOne), "overlay/own-use.toml")

// subsistencePosLines is the number of positions that ran per tick with the module
// on: the 22 of the off set and P2.4 and P2.8.
const subsistencePosLines = 24

// TestGolden_SubsistenceAlone verifies the module-alone golden of the subsistence
// module: pack one + overlay/own-use.toml, seed 1, 400 ticks, run as a Step
// loop with the recorder and WithAudit (so the books are audited every tick),
// against testdata/golden/subsistence/one.txt.
//
// Requirement: FR-REF-025, FR-REF-026, FR-SUB-001, FR-SUB-003, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with
// -update only after the diff is understood (testing.md 7.2).
func TestGolden_SubsistenceAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSubsistence...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("one", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveCold(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("subsistence", "one.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one.txt differs from the golden: %s (testing.md 7.1.2)", subsistencePositionsNote(t, cfg))
	}
	checkModuleGolden(t, "subsistence", "one.txt", got)
}

// TestGolden_SubsistenceAloneEvents verifies the golden rendering of the events of
// pack one with the module on, seed 1, one Run of 400 ticks, against
// testdata/golden/subsistence/events-one.txt, and that the Run and a bare Step
// loop end in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-SUB-040
//
// Guard: the file holds OwnUseChanged rows; without them the golden would not
// see the mode.
func TestGolden_SubsistenceAloneEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSubsistence...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEvents(res)
	if !bytes.Contains(got, []byte("OwnUseChanged")) {
		t.Errorf("events of pack one with the module hold no OwnUseChanged, want the module's event in the golden")
	}
	checkModuleGolden(t, "subsistence", "events-one.txt", got)
}

// TestPositionHash_SubsistenceGolden verifies the per-position state hash of
// pack one with the module on, seed 1, ticks 0 to 29, against
// testdata/golden/subsistence/one.positions.txt: one line per position that ran,
// 24 per tick (P2.4 and P2.8 run with the module on, and not in the off set).
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-SUB-015
//
// Guard: when the golden fails, the first differing line names the tick and
// position at which the module's state diverged.
func TestPositionHash_SubsistenceGolden(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, packOneSubsistence...), goldenSeed, posTicks)
	if want := posTicks * subsistencePosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, subsistencePosLines)
	}
	checkModuleGolden(t, "subsistence", "one.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// subsistencePositionsNote replays the scripted run on cfg against the recorded
// positions file of the module, so a failing golden says where the state first
// moved.
func subsistencePositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("subsistence", "one.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with one.positions.txt: the divergence is later", posTicks-1)
}
