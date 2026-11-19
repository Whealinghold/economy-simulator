package saving_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/saving"
)

// TestOutlay_Example16_5 checks the outlay of 16.1 (600) and with a tool of 50
// that the household does not hold (650, funded at 740).
//
// Requirement: FR-SAV-019
func TestOutlay_Example16_5(t *testing.T) {
	t.Parallel()
	if got := saving.Outlay(360, 0, 30, 5, 2); got != 600 {
		t.Fatalf("Outlay = %d, want 600", got)
	}
	tool := saving.ToolBuy(1, []int64{0}, []int64{50})
	if got := saving.Outlay(360, tool, 30, 5, 2); got != 650 {
		t.Errorf("Outlay with a tool = %d, want 650", got)
	}
	if !saving.Funded(740, 0, 650, 90) || saving.Funded(739, 0, 650, 90) {
		t.Error("with a tool the household is funded at 740 and not at 739")
	}
	if got := saving.ToolBuy(2, []int64{5, 1}, []int64{50, 10}); got != 10 {
		t.Errorf("ToolBuy of a held tool = %d, want 10 (only the second tool is short)", got)
	}
}

// TestFunded_Example16_1 checks that the household is funded at 690 and not at
// 689, and that held inputs count towards the funds.
//
// Requirement: FR-SAV-019
func TestFunded_Example16_1(t *testing.T) {
	t.Parallel()
	if !saving.Funded(690, 0, 600, 90) || saving.Funded(689, 0, 600, 90) {
		t.Error("the mill's household is funded at 690 and not at 689")
	}
	if !saving.Funded(330, 360, 600, 90) {
		t.Error("held inputs must count towards the funds")
	}
}

// TestGoal_NetOfTier2FundingTick checks the funding tick of 16.5: money 700,
// tier 2 of 360, so the goal is 240 and S* = 90 + 240 = 330, against 690 with
// the full goal.
//
// Requirement: FR-SAV-019
func TestGoal_NetOfTier2FundingTick(t *testing.T) {
	t.Parallel()
	if got := saving.Goal(600, 0, 0); got != 600 {
		t.Errorf("full goal = %d, want 600", got)
	}
	g := saving.Goal(600, 0, 360)
	if g != 240 {
		t.Errorf("goal net of tier 2 = %d, want 240", g)
	}
	if got := saving.Target(90, g); got != 330 {
		t.Errorf("S* = %d, want 330", got)
	}
	if got := saving.Target(90, saving.Goal(600, 0, 0)); got != 690 {
		t.Errorf("S* with the full goal = %d, want 690", got)
	}
	if got := saving.Goal(600, 700, 0); got != 0 {
		t.Errorf("goal of a project held in full = %d, want 0", got)
	}
}

// TestKeep_Example16_5 checks the keep of a job in progress: 180 at Remaining 3.
//
// Requirement: FR-SAV-046
func TestKeep_Example16_5(t *testing.T) {
	t.Parallel()
	if got := saving.Keep(30, 3, 2); got != 180 {
		t.Errorf("Keep = %d, want 180", got)
	}
	if got := saving.Target(saving.Floor(90, 0), saving.Keep(30, 3, 2)); got != 270 {
		t.Errorf("S* during the job = %d, want 270", got)
	}
}

// TestHeldValue_UpToWhatMExecutionsNeed checks that held stock counts only up
// to the quantity the executions need.
//
// Requirement: FR-SAV-016
func TestHeldValue_UpToWhatMExecutionsNeed(t *testing.T) {
	t.Parallel()
	got := saving.HeldValue(1, []int64{30, 10}, []int64{100, 4}, []int64{8, 12})
	if want := int64(30*8 + 4*12); got != want {
		t.Errorf("HeldValue = %d, want %d", got, want)
	}
	if got := saving.HeldValue(2, []int64{1}, []int64{1}, []int64{50}); got != 50 {
		t.Errorf("HeldValue of one tool of two = %d, want 50", got)
	}
}

// TestReach_Example16_5 checks the reach test of 5.6.5: reachable at C_n 420,
// not at 60, and a household with money 690 needs nothing.
//
// Requirement: FR-SAV-045
func TestReach_Example16_5(t *testing.T) {
	t.Parallel()
	need := saving.Need(600, 0, 90, 100)
	if need != 590 {
		t.Fatalf("Need = %d, want 590", need)
	}
	if !saving.Reach(need, 30, 420, 180) {
		t.Error("reachable at C_n 420: 17 700 <= 75 600")
	}
	if saving.Reach(need, 30, 60, 180) {
		t.Error("not reachable at C_n 60: 17 700 > 10 800")
	}
	if got := saving.Need(600, 0, 90, 690); got != 0 {
		t.Errorf("Need at money 690 = %d, want 0", got)
	}
	if !saving.Reach(0, 30, -50, 180) {
		t.Error("a need of 0 is reachable whatever the surplus")
	}
}

// TestReach_NZeroOnlyNeedZero checks that with no records only a need of 0 is
// reachable, and that a negative C_n reaches nothing.
//
// Requirement: FR-SAV-045
func TestReach_NZeroOnlyNeedZero(t *testing.T) {
	t.Parallel()
	if !saving.Reach(0, 0, 0, 180) || saving.Reach(1, 0, 1_000_000, 180) {
		t.Error("with n of 0 only a need of 0 is reachable")
	}
	if saving.Reach(1, 5, -10, 180) {
		t.Error("a negative C_n must reach nothing")
	}
}

// TestRepair_Example16_2 checks the repair of 16.2: m_plan 5, restored 480,
// T_rep 180, a benefit of 2 160 at a cost of 180 (ratio 12), and the gate.
//
// Requirement: FR-SAV-022
func TestRepair_Example16_2(t *testing.T) {
	t.Parallel()
	mPlan := saving.MPlan(480, 100)
	if mPlan != 5 {
		t.Fatalf("MPlan = %d, want 5", mPlan)
	}
	if got := saving.Restored(480, mPlan, 100); got != 480 {
		t.Errorf("Restored = %d, want 480", got)
	}
	rate := saving.Rate(1, 1, 1, 1) // k = min(w, cap_now) = 1 execution
	tRep := saving.RepairTime(480, rate, 180, 1)
	if tRep != 180 {
		t.Fatalf("RepairTime = %d, want 180", tRep)
	}
	benefit := saving.SiteBenefit(120_000, tRep, maxMoney)
	cost := saving.Cost(saving.Inputs(mPlan, []int64{4}, []int64{8}), saving.Opportunity(200_000, 1), maxMoney)
	if benefit != 2_160 || cost != 180 {
		t.Errorf("benefit %d, cost %d, want 2160 and 180", benefit, cost)
	}
	if !saving.RepairDue(saving.ConditionBP(120, 600), 5_000) {
		t.Error("a mill at 20% must be due at a gate of 5 000")
	}
	if !saving.RepairDue(saving.ConditionBP(300, 600), 5_000) {
		t.Error("a mill exactly at the gate is due")
	}
	if saving.RepairDue(saving.ConditionBP(301, 600), 5_000) {
		t.Error("a mill above the gate is not due")
	}
}

// TestRepair_NeverAtZeroGate checks that a gate of 0 never repairs, even a
// facility at 0% condition, and that the repair time is bounded by the horizon
// times the facilities.
//
// Requirement: FR-SAV-022
func TestRepair_NeverAtZeroGate(t *testing.T) {
	t.Parallel()
	if saving.RepairDue(0, 0) {
		t.Error("a gate of 0 must never repair")
	}
	if got := saving.RepairTime(1_000_000, 2, 180, 3); got != 540 {
		t.Errorf("RepairTime bounded by the horizon = %d, want 540", got)
	}
	if got := saving.RepairTime(10, 0, 180, 2); got != 360 {
		t.Errorf("RepairTime at rate 0 = %d, want 360", got)
	}
	if got := saving.MPlan(1, 100); got != 1 {
		t.Errorf("MPlan rounds up: %d, want 1", got)
	}
	if got := saving.Restored(50, 1, 100); got != 50 {
		t.Errorf("Restored is at most what is missing: %d, want 50", got)
	}
}

// TestPrecaution_ShareCapAndZero checks the buffer and floor of 16.3: 2 200 at
// a share of 10 000, 1 100 at 5 000, 1 000 with a cap of 10 ticks, 0 at 0.
//
// Requirement: FR-SAV-007
func TestPrecaution_ShareCapAndZero(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                                    string
		share, capTicks, upkeep, drawdown, want int64
	}{
		{"share 10 000", 10_000, 30, 100, 2_200, 2_200},
		{"share 5 000", 5_000, 30, 100, 2_200, 1_100},
		{"cap of 10 ticks", 10_000, 10, 100, 2_200, 1_000},
		{"share 0 is no buffer", 0, 30, 100, 2_200, 0},
		{"no drawdown", 10_000, 30, 100, 0, 0},
		{"share of 20 000 at the cap", 20_000, 30, 100, 2_200, 3_000},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := saving.Precaution(tc.share, tc.capTicks, tc.upkeep, tc.drawdown); got != tc.want {
				t.Errorf("Precaution = %d, want %d", got, tc.want)
			}
		})
	}
	if got := saving.Floor(300, 2_200); got != 2_200 {
		t.Errorf("Floor = %d, want 2200", got)
	}
	if got := saving.Floor(300, 100); got != 300 {
		t.Errorf("the reserve is the floor of the buffer: %d, want 300", got)
	}
}

// TestToolBuy_NegativeStockIsZero verifies that a negative stock reads as no
// stock, as HeldValue reads it, so the difference cannot wrap or overstate.
//
// Guard: without the clamp, stock -5 and m 2 would ask for 7 tools, and a
// stock at MinInt64 would wrap m - stock and understate the outlay.
func TestToolBuy_NegativeStockIsZero(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		stock int64
	}{
		{"small negative", -5},
		{"extreme negative", math.MinInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// m = 2 tools wanted, none held, centre 1: the buy is 2 * 1.
			if got := saving.ToolBuy(2, []int64{tt.stock}, []int64{1}); got != 2 {
				t.Errorf("ToolBuy(2, [%d], [1]) = %d, want 2", tt.stock, got)
			}
		})
	}
}

// TestServiceTime_NegativePaybackIsZero verifies that a negative payback gives
// a service time of 0 with and without a wear rate.
//
// Guard: a negative service time would reach callers that use it directly.
func TestServiceTime_NegativePaybackIsZero(t *testing.T) {
	t.Parallel()
	for _, rate := range []int64{0, 4} {
		if got := saving.ServiceTime(100, rate, -3); got != 0 {
			t.Errorf("ServiceTime(100, %d, -3) = %d, want 0", rate, got)
		}
	}
}

// TestFunded_NegativeMoneySaturatesDown checks that an invalid balance at the
// bottom of int64 cannot wrap up into a funded household.
//
// Requirement: FR-SAV-019
func TestFunded_NegativeMoneySaturatesDown(t *testing.T) {
	t.Parallel()
	if saving.Funded(math.MinInt64, math.MinInt64, 0, 0) {
		t.Error("Funded with money at MinInt64 and held at MinInt64 = true, want false")
	}
	if saving.Funded(math.MinInt64, 10, 100, 0) {
		t.Error("Funded with money at MinInt64 = true, want false")
	}
}
