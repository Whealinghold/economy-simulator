package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestScenarioS36_ScaleSeam verifies MON-S4 (spec-money-supply.md 16, 4.4.3) through
// the internal setter, the only form that exists in this plan (the public
// SetFlowScales is plan 17's, which adds the public leg of S36): two grants run 10
// ticks with the audit on; between steps the scale of one is set to half, then to
// zero, then back. Exact: only that flow's amounts change, from the next step;
// the other flow's per-tick amount never changes; a world in which no call is made
// pays the same every tick; MoneyFlowScaled is emitted once per change and not for
// an equal scale; the audit holds.
//
// Requirement: FR-MON-023, FR-MON-024, FR-MON-045, FR-MON-031
//
// Guard: a scale that applied to the tick already run, or leaked into another
// flow, would move a policy's lever by more than it asked.
func TestScenarioS36_ScaleSeam(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	// 10 farmer households: "a" pays 10 a household-tick (100 a tick), "b" pays 4
	// (40 a tick). The scale of "a" is 10 000 for ticks 0 and 1, 5 000 for ticks 2
	// and 3 (50 a tick), 0 for tick 4 and 5 (paused), 10 000 again from tick 6; a
	// repeated call with the same scale changes nothing.
	a, b := flowTo("a", economysim.FlowGrant, "farmer"), flowTo("b", economysim.FlowGrant, "farmer")
	a.Amount, b.Amount = 10, 4
	cfg := moneyWorldConfig(10, 10, a, b)
	calls := map[int]int64{2: 5_000, 4: 0, 6: 10_000}
	var wantA []economysim.Money
	for tick, scale := 0, int64(10_000); tick < 10; tick++ {
		if s, ok := calls[tick]; ok {
			scale = s
		}
		wantA = append(wantA, economysim.Money(100*scale/10_000))
	}
	for _, seed := range seedList(min(3, scenarioSeedCount())) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			rec := &recorder{}
			w := moneyWorld(t, cfg, economysim.WithSeed(seed), economysim.WithObserver(rec))
			ref := moneyWorld(t, cfg, economysim.WithSeed(seed))
			var gotA, gotB, refA []economysim.Money
			var prevA, prevB, prevRef economysim.Money
			for tick := range 10 {
				if s, ok := calls[tick]; ok {
					if err := economysim.SetFlowScale(w, 0, s); err != nil {
						t.Fatalf("seed %d: SetFlowScale at tick %d error = %v, want nil", seed, tick, err)
					}
					if err := economysim.SetFlowScale(w, 0, s); err != nil { // the same scale again: no change
						t.Fatalf("seed %d: second SetFlowScale error = %v, want nil", seed, err)
					}
				}
				step(t, w, 1)
				step(t, ref, 1)
				ta, tb, tr := flowRow(t, w, "a").Total, flowRow(t, w, "b").Total, flowRow(t, ref, "a").Total
				gotA, gotB, refA = append(gotA, ta-prevA), append(gotB, tb-prevB), append(refA, tr-prevRef)
				prevA, prevB, prevRef = ta, tb, tr
			}
			if !slices.Equal(gotA, wantA) {
				t.Errorf("seed %d: flow a pays %v a tick, want %v", seed, gotA, wantA)
			}
			for tick, v := range gotB {
				if v != 40 || refA[tick] != 100 {
					t.Errorf("seed %d: tick %d: flow b paid %d and the world with no call paid %d from a, want 40 and 100", seed, tick, v, refA[tick])
				}
			}
			var scaled []economysim.MoneyFlowScaled
			for _, e := range rec.events {
				if s, ok := e.(economysim.MoneyFlowScaled); ok {
					scaled = append(scaled, s)
				}
			}
			want := []economysim.MoneyFlowScaled{
				{Tick: 2, Flow: "a", From: 10_000, To: 5_000, Cause: economysim.CauseConsumer},
				{Tick: 4, Flow: "a", From: 5_000, To: 0, Cause: economysim.CauseConsumer},
				{Tick: 6, Flow: "a", From: 0, To: 10_000, Cause: economysim.CauseConsumer},
			}
			if !slices.Equal(scaled, want) {
				t.Errorf("seed %d: MoneyFlowScaled = %+v, want %+v", seed, scaled, want)
			}
			if err := w.Audit(); err != nil {
				t.Errorf("seed %d: Audit after the run = %v, want nil", seed, err)
			}
		})
	}
}
