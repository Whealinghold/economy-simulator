package economysim_test

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// storagePosLines is the number of positions that ran per tick with the module on:
// the 22 of the off set. The module has no position of its own until P3.6a is built.
const storagePosLines = 22

// TestGolden_StorageAlone verifies the module-alone golden of the storage
// module: pack one + overlay/storage.toml, seed 1, 400 ticks, run as a Step
// loop with the recorder and WithAudit (so the books are audited every tick),
// against testdata/golden/storage/one.txt.
//
// Requirement: FR-REF-025, FR-REF-026, FR-STO-005, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with
// -update only after the diff is understood (testing.md 7.2).
func TestGolden_StorageAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneStorage...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("one", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveCold(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("storage", "one.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one.txt differs from the golden: %s (testing.md 7.1.2)", storagePositionsNote(t, cfg))
	}
	checkModuleGolden(t, "storage", "one.txt", got)
}

// TestGolden_StorageAloneEvents verifies the golden rendering of the events of
// pack one with the module on, seed 1, one Run of 400 ticks, against
// testdata/golden/storage/events-one.txt, and that the Run and a bare Step
// loop end in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-STO-001
//
// Guard: the module's events are StoreChanged (a call) and RoomLimited (phase 7), neither
// emitted by a run of pack one with no call; the file records that the module adds no
// event to a run, so a limit that acts or an event that appears unasked moves it.
func TestGolden_StorageAloneEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneStorage...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEvents(res)
	checkModuleGolden(t, "storage", "events-one.txt", got)
}

// TestPositionHash_StorageAlone verifies the per-position state hash of pack
// one with the module on, seed 1, ticks 0 to 29, against
// testdata/golden/storage/one.positions.txt: one line per position that ran,
// 22 per tick.
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-STO-040
//
// Guard: when the golden fails, the first differing line names the tick and
// position at which the module's state diverged.
func TestPositionHash_StorageAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, packOneStorage...), goldenSeed, posTicks)
	if want := posTicks * storagePosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, storagePosLines)
	}
	checkModuleGolden(t, "storage", "one.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// storagePositionsNote replays the scripted run on cfg against the recorded
// positions file of the module, so a failing golden says where the state first
// moved.
func storagePositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("storage", "one.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with one.positions.txt: the divergence is later", posTicks-1)
}
