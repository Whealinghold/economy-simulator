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

// packOnePopulation is pack one with the population module alone on: the overlay
// of overlay/population-low.toml (spec-reference-chain.md 7.1.5, plan 2-11 D438).
// The module has no census of its own, so its module-alone golden is the
// reference chain with the overlay, as testing.md 7.1.1 says.
var packOnePopulation = append(slices.Clone(packOne), "overlay/population-low.toml")

// populationPosLines is the number of positions that ran per tick with the module
// on: the 22 of the off set. The P7.1(b) step is a sub-step of P7.1, not a
// position (architecture.md 2.10.3).
const populationPosLines = 22

// TestGolden_PopulationAlone verifies the module-alone golden of the population
// module: pack one + overlay/population-low.toml, seed 1, 400 ticks, run as a Step
// loop with the recorder and WithAudit (so the books are audited every tick),
// against testdata/golden/population/one.txt.
//
// Requirement: FR-REF-025, FR-REF-026, FR-POP-001, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with
// -update only after the diff is understood (testing.md 7.2).
func TestGolden_PopulationAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOnePopulation...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("one", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveCold(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("population", "one.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one.txt differs from the golden: %s (testing.md 7.1.2)", populationPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "population", "one.txt", got)
}

// TestGolden_PopulationAloneEvents verifies the golden rendering of the events of
// pack one with the module on, seed 1, one Run of 400 ticks, against
// testdata/golden/population/events-one.txt, and that the Run and a bare Step
// loop end in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-POP-031
//
// Guard: the file holds PopulationChanged rows; without them the golden would not
// see the module's births and deaths.
func TestGolden_PopulationAloneEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOnePopulation...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEvents(res)
	if !bytes.Contains(got, []byte("PopulationChanged")) {
		t.Errorf("events of pack one with the module hold no PopulationChanged, want the module's event in the golden")
	}
	checkModuleGolden(t, "population", "events-one.txt", got)
}

// TestPositionHash_PopulationAlone verifies the per-position state hash of pack
// one with the module on, seed 1, ticks 0 to 29, against
// testdata/golden/population/one.positions.txt: one line per position that ran,
// 22 per tick.
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-POP-002
//
// Guard: when the golden fails, the first differing line names the tick and
// position at which the module's state diverged.
func TestPositionHash_PopulationAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, packOnePopulation...), goldenSeed, posTicks)
	if want := posTicks * populationPosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, populationPosLines)
	}
	checkModuleGolden(t, "population", "one.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// populationPositionsNote replays the scripted run on cfg against the recorded
// positions file of the module, so a failing golden says where the state first
// moved.
func populationPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("population", "one.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with one.positions.txt: the divergence is later", posTicks-1)
}
