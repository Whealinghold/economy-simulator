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

// packOneFaucetSunk is pack one with the money supply module alone on: the
// stipend of overlay/faucet.toml and the rate levy of overlay/faucet-sunk.toml
// (spec-money-supply.md 17.6). The module has no census of its own in this plan
// (the warm start with flows on is plan 8's, D339), so its module-alone golden is
// the reference chain with the overlay, as testing.md 7.1.1 says.
var packOneFaucetSunk = append(slices.Clone(packOne), "overlay/faucet-sunk.toml")

// moneyPosLines is the number of positions that ran per tick with the module on:
// the 22 of the off set and P1.7.
const moneyPosLines = 23

// TestGolden_OneMoneySupply verifies the module-alone golden of the money supply
// module: pack one + overlay/faucet-sunk.toml, seed 1, 400 ticks, run as a Step
// loop with the recorder and WithAudit, against
// testdata/golden/money_supply/one.txt.
//
// Requirement: FR-REF-025, FR-REF-026, FR-MON-029, FR-MON-042, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with
// -update only after the diff is understood (testing.md 7.2).
func TestGolden_OneMoneySupply(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneFaucetSunk...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("one", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solveCold(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("money_supply", "one.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("one.txt differs from the golden: %s (testing.md 7.1.2)", moneyPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "money_supply", "one.txt", got)
}

// TestGolden_OneMoneySupplyEvents verifies the golden rendering of the events of
// pack one with the module on, seed 1, one Run of 400 ticks, against
// testdata/golden/money_supply/events-one.txt, and that the Run and a bare Step
// loop end in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-MON-029, FR-MON-030
//
// Guard: the file holds BoundaryFlow rows under the flow IDs; without them the
// golden would not see the module's payments.
func TestGolden_OneMoneySupplyEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneFaucetSunk...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEvents(res)
	for _, reason := range []string{"stipend", "charge"} {
		if !bytes.Contains(got, []byte(reason)) {
			t.Errorf("events of the module hold no %q row, want the flow's BoundaryFlow in the golden", reason)
		}
	}
	checkModuleGolden(t, "money_supply", "events-one.txt", got)
}

// TestPositionHash_OneMoneySupplyGolden verifies the per-position state hash of
// pack one with the module on, seed 1, ticks 0 to 29, against
// testdata/golden/money_supply/one.positions.txt: one line per position that ran,
// 23 per tick (P1.7 runs with the module on, and not in the off set).
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-MON-019
//
// Guard: when the golden fails, the first differing line names the tick and
// position at which the module's state diverged.
func TestPositionHash_OneMoneySupplyGolden(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, packOneFaucetSunk...), goldenSeed, posTicks)
	if want := posTicks * moneyPosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, moneyPosLines)
	}
	checkModuleGolden(t, "money_supply", "one.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// moneyPositionsNote replays the scripted run on cfg against the recorded
// positions file of the module, so a failing golden says where the state first
// moved.
func moneyPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("money_supply", "one.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with one.positions.txt: the divergence is later", posTicks-1)
}
