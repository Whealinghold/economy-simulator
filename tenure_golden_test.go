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

// packFiveTenure is the warm five with the binding resources overlay and the tenure overlay
// layered last: the module-alone pack of the tenure module (spec-reference-chain.md 4.5,
// D10 and D12 of plan-3-lnd.md). Pack one with the same overlays charges no rent in 400
// ticks (the growers draw on their own entitlements and the pool's let units lie idle), so
// the golden moves to the first pack that does, as the plan says: the five, whose binding
// stock makes the commons short and lets units.
var packFiveTenure = append(slices.Clone(packFiveWarm), "overlay/resources-five-binding.toml", "overlay/tenure.toml")

// tenurePosLines is the number of positions that ran per tick with the module on: the 25
// of the resources set plus P8.3a, the position of the module (RentPaid).
const tenurePosLines = 26

// solveTenure is the steady-state report the golden prints for the pack: the solver does
// not read a title or a rent (spec-land.md 12.1).
func solveTenure(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, packFiveTenure...))
	if err != nil {
		t.Fatalf("SolveSteadyState(five + resources + tenure) error = %v, want nil", err)
	}
	return newSolveReport(ss)
}

// TestGolden_TenureAlone verifies the module-alone golden of the tenure module: the warm
// five with overlay/resources-five-binding.toml and overlay/tenure.toml, seed 1, 400
// ticks, run as a Step loop with the recorder and WithAudit, against
// testdata/golden/tenure/five.txt. The file is a new set, not a diff of the off five or of
// the resources golden: the module changes the ration from the first tick.
//
// Requirement: FR-REF-025, FR-LND-001, FR-LND-006, FR-LND-037, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with
// -update only after the diff is understood (testing.md 7.2).
func TestGolden_TenureAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packFiveTenure...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("five", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveTenure(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("tenure", "five.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("five.txt differs from the golden: %s (testing.md 7.1.2)", tenurePositionsNote(t, cfg))
	}
	checkModuleGolden(t, "tenure", "five.txt", got)
}

// TestGolden_TenureAloneEvents verifies the golden rendering of the events of the five
// with the module on, seed 1, one Run of 400 ticks, against
// testdata/golden/tenure/events-five.txt, and that the Run and a bare Step loop end in the
// same snapshot. The rendering leaves out the full block of ticks 0 to 4, as the other
// five-sized goldens do.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-LND-028, FR-LND-044
//
// Guard: the file holds RentPaid rows; without them the golden would not see the module.
func TestGolden_TenureAloneEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packFiveTenure...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEventsFull(res, -1)
	if !bytes.Contains(got, []byte("RentPaid")) {
		t.Errorf("events of the five with the module hold no RentPaid row, want the module's event in the golden")
	}
	checkModuleGolden(t, "tenure", "events-five.txt", got)
}

// TestPositionHash_TenureGolden verifies the per-position state hash of the five with the
// module on, seed 1, ticks 0 to 29, against testdata/golden/tenure/five.positions.txt: one
// line per position that ran, 26 per tick (P8.3a is the module's).
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-LND-036
//
// Guard: when the golden fails, the first differing line names the tick and position at
// which the state diverged.
func TestPositionHash_TenureGolden(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, packFiveTenure...), goldenSeed, posTicks)
	if want := posTicks * tenurePosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, tenurePosLines)
	}
	checkModuleGolden(t, "tenure", "five.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// TestGolden_TenureProbeExercisesRent verifies decision D12 of plan-3-lnd.md: the golden
// run of the module charges rent. It runs the golden pack and fails when no RentPaid has
// LetUnits above 0 and Paid above 0, because a golden that never let a unit would not see
// the rent circuit, the tiers' third tier or the transfer. The pack collapses
// (criterion 1, N-352), so the amount is reported and not asserted.
//
// Requirement: FR-LND-010, FR-LND-017
//
// Guard: if the overlays change so that no unit is let, the golden would stay green and
// show nothing; this test names the moment the pack has to move.
func TestGolden_TenureProbeExercisesRent(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packFiveTenure...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true})
	var events, let, paid int64
	for _, e := range res.Events {
		if r, ok := e.(economysim.RentPaid); ok {
			events++
			let += r.LetUnits
			paid += int64(r.Paid)
		}
	}
	if let == 0 || paid == 0 {
		t.Errorf("%d RentPaid events with %d let units and %d paid in %d ticks, want rent charged (D12: move the golden pack)", events, let, paid, goldenTicks)
	}
	t.Logf("%d RentPaid events, %d units let, %d paid in %d ticks (reported, not asserted: N-352)", events, let, paid, goldenTicks)
}

// tenurePositionsNote replays the run on cfg against the recorded positions file of the
// module, so a failing golden says where the state first moved.
func tenurePositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("tenure", "five.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with five.positions.txt: the divergence is later", posTicks-1)
}
