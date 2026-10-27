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

// packOneEstates is pack one with the estates module alone on: the overlay
// overlay/estates.toml (spec-config.md 6.2.4). The module has no census of its
// own, so its module-alone golden is the reference chain with the overlay
// (testing.md 7.1.1).
var packOneEstates = append(slices.Clone(packOne), "overlay/estates.toml")

// estatesPosLines is the number of positions that ran per tick with the module
// on: the 22 of the off set and P1.6.
const estatesPosLines = 23

// TestGolden_OneEstates verifies the module-alone golden of the estates module:
// pack one + overlay/estates.toml, seed 1, 400 ticks, run as a Step loop with the
// recorder and WithAudit, against testdata/golden/estates/one.txt.
//
// Requirement: FR-REF-029, FR-EST-017, FR-OBS-023
//
// Guard: relief is the only difference from testdata/golden/off/one.txt, and it
// appears only after the first removal that leaves a pool; a behaviour change of
// the module moves the file, and it is regenerated with -update only after the
// diff is understood (testing.md 7.2).
func TestGolden_OneEstates(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneEstates...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("one", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveCold(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("estates", "one.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one.txt differs from the golden: %s (testing.md 7.1.2)", estatesPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "estates", "one.txt", got)
}

// TestGolden_OneEstatesEvents verifies the golden rendering of the events of pack
// one with the module on, seed 1, one Run of 400 ticks, against
// testdata/golden/estates/events-one.txt, and that the Run and a bare Step loop
// end in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-EST-020
//
// Guard: the file must hold ReliefPaid events; without them the golden would not
// see the module.
func TestGolden_OneEstatesEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneEstates...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEvents(res)
	if !bytes.Contains(got, []byte("ReliefPaid")) {
		t.Errorf("events of the module hold no ReliefPaid, want the module's event in the golden")
	}
	checkModuleGolden(t, "estates", "events-one.txt", got)
}

// TestPositionHash_OneEstatesGolden verifies the per-position state hash of pack
// one with the module on, seed 1, ticks 0 to 29, against
// testdata/golden/estates/one.positions.txt: one line per position that ran, 23
// per tick (P1.6 runs with the module on, and not in the off set).
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-EST-017
//
// Guard: when the golden fails, the first differing line names the tick and
// position at which the module's state diverged.
func TestPositionHash_OneEstatesGolden(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, packOneEstates...), goldenSeed, posTicks)
	if want := posTicks * estatesPosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, estatesPosLines)
	}
	checkModuleGolden(t, "estates", "one.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// estatesPositionsNote replays the scripted run on cfg against the recorded
// positions file of the module, so a failing golden says where the state first
// moved.
func estatesPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("estates", "one.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with one.positions.txt: the divergence is later", posTicks-1)
}
