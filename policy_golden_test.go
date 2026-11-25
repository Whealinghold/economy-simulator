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

// fivePolicy is the module-alone pack of the policy module (plan 3-plc D17, spec-reference-chain.md
// 4.5): the warm five with the trade edges, the transport modes, the governed polity and the money
// supply's faucet, and overlay/policy.toml layered last.
var fivePolicy = append(slices.Clone(warmFive), "overlay/trade-five.toml", "overlay/transport-five.toml",
	"overlay/polity-governed.toml", "overlay/faucet.toml", "overlay/policy.toml")

// fivePolicyDirected is fivePolicy with overlay/policy-directed.toml in the place of policy.toml.
var fivePolicyDirected = append(slices.Clone(fivePolicy[:len(fivePolicy)-1]), "overlay/policy-directed.toml")

// solvePolicy is the steady-state report the golden prints for the pack.
func solvePolicy(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, fivePolicy...))
	if err != nil {
		t.Fatalf("SolveSteadyState(five + policy) error = %v, want nil", err)
	}
	return newSolveReport(ss)
}

// TestGolden_PolicyAlone verifies the module-alone golden of the policy module: the warm five with
// the trade, transport, governed polity and faucet overlays and overlay/policy.toml, seed 1, 400
// ticks, run as a Step loop with the recorder and WithAudit, against testdata/golden/policy/five.txt.
//
// Requirement: FR-REF-025, FR-PLC-001, FR-PLC-043, FR-OBS-023
//
// Guard: a behaviour change of the module moves the file; it is regenerated with -update only after
// the diff against golden/polity and golden/transport is understood (testing.md 7.2).
func TestGolden_PolicyAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fivePolicy...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample, Digest: true})
	got := renderGolden("five", goldenSeed, goldenTicks, samplesOnly(res, goldenTicks), solvePolicy(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("policy", "five.txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("five.txt differs from the golden: %s (testing.md 7.1.2)", policyPositionsNote(t, cfg))
	}
	checkModuleGolden(t, "policy", "five.txt", got)
}

// TestGolden_PolicyAloneEvents verifies the golden rendering of the events of the pack, seed 1, one
// Run of 400 ticks, against testdata/golden/policy/events-five.txt, and that the Run and a bare Step
// loop end in the same snapshot.
//
// Requirement: FR-OBS-080, FR-OBS-081, FR-PLC-080
//
// Guard: the file holds the module's event and the counts of the three slot events it moves; without
// them the golden would not see the module (the cause is the probe's, TestGolden_PolicyProbeExercisesActors).
func TestGolden_PolicyAloneEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fivePolicy...)
	res := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveRun(goldenTicks), Observe: true})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: goldenTicks, Drive: driveStep})
	if !reflect.DeepEqual(res.Final, step.Final) {
		t.Errorf("final snapshot of Run(400) differs from the Step loop's (seed %d)", goldenSeed)
	}
	got := renderEvents(res)
	for _, w := range []string{"PolicyDecided Actor=", "MoneyFlowScaled", "EdgeStatusChanged", "TransitFactorChanged"} {
		if !bytes.Contains(got, []byte(w)) {
			t.Errorf("events of the policy pack hold no %q, want the module's trace in the golden", w)
		}
	}
	checkModuleGolden(t, "policy", "events-five.txt", got)
}

// policyPosLines is the number of positions that ran per tick: the 22 of the off set, P1.3 and P3.1
// of the transport module, the money supply's own and P0.3a.
const policyPosLines = 26

// TestPositionHash_PolicyGolden verifies the per-position state hash of the pack, seed 1, ticks 0 to
// 29, against testdata/golden/policy/five.positions.txt.
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-PLC-050
//
// Guard: when the golden fails, the first differing line names the tick and position at which the
// state diverged.
func TestPositionHash_PolicyGolden(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	lines := positionLines(t, loadValidPack(t, fivePolicy...), goldenSeed, posTicks)
	if want := posTicks * policyPosLines; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (%d positions per tick)", len(lines), want, policyPosLines)
	}
	checkModuleGolden(t, "policy", "five.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// policyPositionsNote replays the scripted run on cfg against the recorded positions file, so a
// failing golden says where the state first moved.
func policyPositionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("policy", "five.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return "positions of ticks 0 to 29 agree with five.positions.txt: the divergence is later"
}

// TestGolden_PolicyProbeExercisesActors verifies, over seeds 1 to 4, that the pack of the golden
// makes the actors act: each run records a PolicyDecided, a MoneyFlowScaled, a
// TransitFactorChanged and an EdgeStatusChanged with the cause policy, and every one of these
// events names a key an actor holds. A pack whose actors never moved anything would record a golden
// that proves nothing about the module (plan 3-plc Phase 8).
//
// Requirement: FR-PLC-043, FR-PLC-080, FR-PLC-085
//
// Guard: an overlay whose levers are never reached, or a rule that stays in its band, fails here
// before the golden can be recorded empty.
func TestGolden_PolicyProbeExercisesActors(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, fivePolicy...)
	for seed := uint64(1); seed <= 4; seed++ {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			res := runPack(t, cfg, runSpec{Seed: seed, Ticks: goldenTicks, Drive: driveStep, Observe: true})
			var flow, edge, transit, decided int
			for _, e := range res.Events {
				switch v := e.(type) {
				case economysim.MoneyFlowScaled:
					flow += b2i(v.Cause == economysim.CausePolicy)
				case economysim.EdgeStatusChanged:
					edge += b2i(v.Cause == economysim.CausePolicy)
				case economysim.TransitFactorChanged:
					transit += b2i(v.Cause == economysim.CausePolicy)
				case economysim.PolicyDecided:
					decided++
				}
			}
			if flow == 0 || edge == 0 || transit == 0 || decided == 0 {
				t.Errorf("seed %d: policy events = %d flow scales, %d edge statuses, %d transit factors, %d decisions, want every count above 0", seed, flow, edge, transit, decided)
			}
		})
	}
}

// b2i is 1 for true and 0 for false.
func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
