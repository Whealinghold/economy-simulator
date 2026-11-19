package saving_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/saving"
)

const maxMoney = 1_000_000_000_000_000_000

// TestPv_TruncatesTowardZero checks the value per tick of spec 16.1 and the
// truncation of a quotient that is not whole.
//
// Requirement: FR-SAV-009
func TestPv_TruncatesTowardZero(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		net, duration int64
		want          int64
	}{
		{"grind of 16.1", 20, 1, 200_000},
		{"carry of 16.1", 8, 1, 80_000},
		{"truncated", 7, 3, 23_333},
		{"negative truncates toward zero", -7, 3, -23_333},
		{"duration below one counts as one", 5, 0, 50_000},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := saving.PerTick(tc.net, tc.duration); got != tc.want {
				t.Errorf("PerTick(%d, %d) = %d, want %d", tc.net, tc.duration, got, tc.want)
			}
		})
	}
}

// TestJobValue_Example16_1 checks J(mill, 0), J(mill, 6) and the gain of a
// build with the numbers of spec 16.1, and that a gain is never negative.
//
// Requirement: FR-SAV-010
func TestJobValue_Example16_1(t *testing.T) {
	t.Parallel()
	const pvB, pvA = 200_000, 80_000
	j0 := saving.JobValue(pvB, pvA, 1, 0)
	j6 := saving.JobValue(pvB, pvA, 1, 6)
	if j0 != 80_000 || j6 != 200_000 {
		t.Fatalf("J(0) = %d, J(6) = %d, want 80000 and 200000", j0, j6)
	}
	if got := saving.BuildGain(j0, j6); got != 120_000 {
		t.Errorf("BuildGain = %d, want 120000", got)
	}
	// 16.2: at cap_now 2 a further build adds nothing for a weight-1 household.
	if got := saving.BuildGain(saving.JobValue(pvB, pvA, 1, 2), saving.JobValue(pvB, pvA, 1, 8)); got != 0 {
		t.Errorf("BuildGain at cap_now 2 = %d, want 0", got)
	}
}

// TestBuildGain_NeverNegative checks that a job value that falls gives 0.
//
// Requirement: FR-SAV-010
func TestBuildGain_NeverNegative(t *testing.T) {
	t.Parallel()
	if got := saving.BuildGain(200, 100); got != 0 {
		t.Errorf("BuildGain(200, 100) = %d, want 0", got)
	}
}

// TestJobValue_WeightAndCapacity checks that a cohort's job is limited by the
// executions of capacity and that the better of the two jobs wins.
//
// Requirement: FR-SAV-010
func TestJobValue_WeightAndCapacity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		pvB, pvA, w, k int64
		want           int64
	}{
		{"capacity limits the job", 100, 10, 5, 2, 200 + 0},
		{"the job without the site wins", 10, 100, 5, 5, 500},
		{"no capacity", 100, 10, 3, 0, 30},
		{"negative inputs count as zero", -5, -5, 3, 3, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := saving.JobValue(tc.pvB, tc.pvA, tc.w, tc.k); got != tc.want {
				t.Errorf("JobValue(%d, %d, %d, %d) = %d, want %d", tc.pvB, tc.pvA, tc.w, tc.k, got, tc.want)
			}
		})
	}
}

// TestRate_ServiceTime_RateZero checks the rate, the ceiling of the wear and
// the service time of spec 5.3.1, including a rate of 0.
//
// Requirement: FR-SAV-011
func TestRate_ServiceTime_RateZero(t *testing.T) {
	t.Parallel()
	if got := saving.Rate(1, 1, 1, 1); got != 2 {
		t.Errorf("Rate(1, 1, 1, 1) = %d, want 2", got)
	}
	if got := saving.Rate(1, 1, 3, 2); got != 3 { // 1 + ceil(3 / 2)
		t.Errorf("Rate(1, 1, 3, 2) = %d, want 3", got)
	}
	if got := saving.WorthRate(0, 2, 5, 3); got != 6 { // wear * min(w, capacity)
		t.Errorf("WorthRate = %d, want 6", got)
	}
	tests := []struct{ durability, rate, payback, want int64 }{
		{600, 2, 180, 180}, // 300 ticks of life, capped by the horizon
		{100, 2, 180, 50},  // the life binds
		{600, 0, 180, 180}, // rate 0: the horizon
		{7, 2, 180, 3},     // floor
	}
	for _, tc := range tests {
		if got := saving.ServiceTime(tc.durability, tc.rate, tc.payback); got != tc.want {
			t.Errorf("ServiceTime(%d, %d, %d) = %d, want %d", tc.durability, tc.rate, tc.payback, got, tc.want)
		}
	}
}

// TestBenefit_SiteAndWorthAndNone checks the benefit of a site (2 160 in 16.1)
// and of a class with a Worth row (1 500 in 16.4, 0 at Count).
//
// Requirement: FR-SAV-012
func TestBenefit_SiteAndWorthAndNone(t *testing.T) {
	t.Parallel()
	if got := saving.SiteBenefit(120_000, 180, maxMoney); got != 2_160 {
		t.Errorf("SiteBenefit = %d, want 2160", got)
	}
	if got := saving.WorthBenefit(15, 100, 0, 4, maxMoney); got != 1_500 {
		t.Errorf("WorthBenefit below Count = %d, want 1500", got)
	}
	if got := saving.WorthBenefit(15, 100, 4, 4, maxMoney); got != 0 {
		t.Errorf("WorthBenefit at Count = %d, want 0", got)
	}
	if got := saving.SiteBenefit(0, 180, maxMoney); got != 0 {
		t.Errorf("SiteBenefit of a zero gain = %d, want 0", got)
	}
}

// TestSaturation_BenefitPassesCostFails checks that a benefit and a cost that
// leave int64 saturate at the money bound and never wrap.
//
// Requirement: FR-SAV-014, FR-SAV-043
func TestSaturation_BenefitPassesCostFails(t *testing.T) {
	t.Parallel()
	b := saving.SiteBenefit(math.MaxInt64, math.MaxInt64, maxMoney)
	if b != maxMoney {
		t.Fatalf("SiteBenefit at the extremes = %d, want %d", b, maxMoney)
	}
	if !saving.Passes(b, 1, 200_000, maxMoney) {
		t.Error("a saturated benefit must pass")
	}
	c := saving.Cost(math.MaxInt64, math.MaxInt64, maxMoney)
	if c != maxMoney {
		t.Fatalf("Cost at the extremes = %d, want %d", c, maxMoney)
	}
	if saving.Passes(maxMoney-1, c, 10_000, maxMoney) {
		t.Error("a saturated cost must fail")
	}
	if got := saving.PerTick(maxMoney, 1); got != math.MaxInt64 {
		t.Errorf("PerTick(MaxMoney, 1) = %d, want it saturated at MaxInt64", got)
	}
}

// TestEdges_DegenerateArguments checks the guards of the pure rules: a
// duration or a unit count below 1, a quantity that leaves int64 in either
// direction, and a negative argument.
//
// Requirement: FR-SAV-043
func TestEdges_DegenerateArguments(t *testing.T) {
	t.Parallel()
	if got := saving.Rate(1, 1, 3, 0); got != 4 { // a duration of 0 counts as 1
		t.Errorf("Rate with a duration of 0 = %d, want 4", got)
	}
	if got := saving.Rate(0, math.MaxInt64, math.MaxInt64, 1); got != math.MaxInt64 {
		t.Errorf("Rate beyond int64 = %d, want it saturated", got)
	}
	if got := saving.PerTick(-maxMoney, 1); got != math.MinInt64 {
		t.Errorf("PerTick(-MaxMoney, 1) = %d, want it saturated at MinInt64", got)
	}
	if got := saving.ConditionBP(5, 0); got != 0 {
		t.Errorf("ConditionBP with no durability = %d, want 0", got)
	}
	if got := saving.MPlan(10, 0); got != 10 { // units of 0 count as 1
		t.Errorf("MPlan with units of 0 = %d, want 10", got)
	}
	if got := saving.Need(1, 0, 0, math.MinInt64); got != math.MaxInt64 {
		t.Errorf("Need beyond int64 = %d, want it saturated", got)
	}
	if got := saving.Goal(math.MinInt64, math.MaxInt64, 0); got != 0 {
		t.Errorf("Goal of a negative outlay = %d, want 0", got)
	}
	if got := saving.Outlay(math.MaxInt64, math.MaxInt64, 1, 1, 1); got != math.MaxInt64 {
		t.Errorf("Outlay beyond int64 = %d, want it saturated", got)
	}
}
