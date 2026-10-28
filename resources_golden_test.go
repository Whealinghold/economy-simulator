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

// packOneResources is pack one with the resources module alone on: the slack
// overlay of overlay/resources.toml (spec-resources.md 18). The module has no
// census of its own (the warm start sits at the plan stock, InitialBP equal to
// PlanShareBP, spec-resources.md 11.4), so its module-alone golden is the
// reference chain with the overlay, as testing.md 7.1.1 says.
var packOneResources = append(slices.Clone(packOne), "overlay/resources.toml")

// resourcesPosLines is the number of positions that ran per tick with the module
// on: the 22 of the off set and P2.3, P2.6 and P8.3.
const resourcesPosLines = 25

// TestGolden_OneResources verifies the module-alone golden of the resources
// module: pack one + overlay/resources.toml, seed 1, 400 ticks, run as a Step
// loop with the recorder and WithAudit (so the identity of an endowment is
// audited every tick), against testdata/golden/resources/one.txt.
//
// Requirement: FR-REF-025, FR-REF-026, FR-RES-025, FR-RES-047, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with
// -update only after the diff is understood (testing.md 7.2).
func TestGolden_OneResources(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneResources...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("one", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveCold(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("resources", "one.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one.txt differs from the golden: %s (testing.md 7.1.2)", resourcesPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "resources", "one.txt", got)
}

// TestGolden_OneResourcesEvents verifies the golden rendering of the events of
// pack one with the module on, seed 1, one Run of 400 ticks, against
// testdata/golden/resources/events-one.txt, and that the Run and a bare Step
// loop end in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-RES-026, FR-RES-027
//
// Guard: the file holds EndowmentChanged rows; without them the golden would not
// see the module's draws and regeneration.
func TestGolden_OneResourcesEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneResources...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEvents(res)
	if !bytes.Contains(got, []byte("EndowmentChanged")) {
		t.Errorf("events of pack one with resources hold no EndowmentChanged, want the module's event in the golden")
	}
	checkModuleGolden(t, "resources", "events-one.txt", got)
}

// TestPositionHash_OneResourcesGolden verifies the per-position state hash of
// pack one with the module on, seed 1, ticks 0 to 29, against
// testdata/golden/resources/one.positions.txt: one line per position that ran,
// 25 per tick (P2.3, P2.6 and P8.3 run with the module on, and not in the off
// set).
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-RES-015
//
// Guard: when the golden fails, the first differing line names the tick and
// position at which the module's state diverged.
func TestPositionHash_OneResourcesGolden(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, packOneResources...), goldenSeed, posTicks)
	if want := posTicks * resourcesPosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, resourcesPosLines)
	}
	checkModuleGolden(t, "resources", "one.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// resourcesPositionsNote replays the scripted run on cfg against the recorded
// positions file of the module, so a failing golden says where the state first
// moved.
func resourcesPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("resources", "one.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with one.positions.txt: the divergence is later", posTicks-1)
}
