package welfare_test

import (
	"math"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// TestRatio_PaddingAndTruncation verifies the ratio of 2.1 and 2.2: the sums
// of the window plus the padding of the missing ticks, one truncating
// division, negative allowed, absent for no cost.
//
// Requirement: FR-WEL-001, FR-WEL-007, FR-WEL-008, FR-WEL-010
func TestRatio_PaddingAndTruncation(t *testing.T) {
	t.Parallel()
	const maxI = math.MaxInt64
	tests := []struct {
		name                           string
		income, cost, n, upkeep, defBP int64
		want                           int64
		present                        bool
	}{
		// 450 * 10 000 / 300 = 15 000.
		{"full window", 450, 300, 0, 10, 10_000, 15_000, true},
		// 300 * 10 000 / 300.
		{"exactly 10 000 per tick", 300, 300, 0, 10, 10_000, 10_000, true},
		// pad_cost 5 * 10 = 50, pad_income 50 * 10 000 / 10 000 = 50;
		// (100 + 50) * 10 000 / (100 + 50) = 10 000.
		{"five missing ticks at 10 000", 100, 100, 5, 10, 10_000, 10_000, true},
		// pad_cost 100, pad_income 100 * 15 000 / 10 000 = 150;
		// (200 + 150) * 10 000 / 300 = 11 666.67, truncated.
		{"ten missing ticks at 15 000", 200, 200, 10, 10, 15_000, 11_666, true},
		// -100 * 10 000 / 200.
		{"negative sum", -100, 200, 0, 10, 10_000, -5_000, true},
		// -10 000 / 3 = -3 333.33, truncated toward zero, not -3 334.
		{"negative numerator truncates toward zero", -1, 3, 0, 1, 10_000, -3_333, true},
		{"zero denominator is absent", 5, 0, 0, 0, 10_000, 0, false},
		{"zero denominator with missing ticks of zero upkeep", 5, 0, 3, 0, 10_000, 0, false},
		// T = n = 30, U = 2: pad_cost 60, pad_income 60 * 15 000 / 10 000 = 90,
		// 90 * 10 000 / 60 = 15 000 exactly.
		{"no record held, exact padding at 15 000", 0, 0, 30, 2, 15_000, 15_000, true},
		// pad_cost 2, pad_income 2 * 15 000 / 10 000 = 3, 3 * 10 000 / 2.
		{"one missing tick, exact padding at 15 000", 0, 0, 1, 2, 15_000, 15_000, true},
		// pad_cost 1, pad_income 15 000 / 10 000 = 1 (1.5 truncated),
		// 1 * 10 000 / 1 = 10 000, not 15 000.
		{"padding truncates when n*U*d is not divisible", 0, 0, 1, 1, 15_000, 10_000, true},
		// pad_cost 91, pad_income 91, 91 * 10 000 / 91.
		{"no record held at 10 000 is exact for any n and U", 0, 0, 7, 13, 10_000, 10_000, true},
		// MaxInt64 / 2 * 10 000 / 20 000 = (MaxInt64 / 2) / 2 = 2 305 843 009 213 693 951:
		// the product by 10 000 is above int64 but the quotient is in range.
		{"quotient in range from a product above int64", maxI / 2, 20_000, 0, 1, 10_000, maxI / 4, true},
		{"quotient above int64 saturates", maxI, 1, 0, 1, 10_000, maxI, true},
		{"quotient below int64 saturates", math.MinInt64, 1, 0, 1, 10_000, math.MinInt64, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, present := welfare.Ratio(tt.income, tt.cost, tt.n, tt.upkeep, tt.defBP)
			if got != tt.want || present != tt.present {
				t.Errorf("Ratio(%d, %d, %d, %d, %d) = %d, %v, want %d, %v",
					tt.income, tt.cost, tt.n, tt.upkeep, tt.defBP, got, present, tt.want, tt.present)
			}
		})
	}
}

// TestEarnedRatio_TakesTheReliefOut verifies that the earned ratio is the ratio
// over the income less the relief with the same padding, and absent when the
// ratio is.
//
// Requirement: FR-WEL-039
func TestEarnedRatio_TakesTheReliefOut(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                                   string
		income, relief, cost, n, upkeep, defBP int64
		want                                   int64
		present                                bool
	}{
		// (450 - 150) * 10 000 / 300.
		{"full window", 450, 150, 300, 0, 10, 10_000, 10_000, true},
		// pad_cost 100, pad_income 100; (200 - 50 + 100) * 10 000 / 300 = 8 333.33.
		{"padded", 200, 50, 200, 10, 10, 10_000, 8_333, true},
		// no relief: the same as the ratio, 450 * 10 000 / 300.
		{"no relief", 450, 0, 300, 0, 10, 10_000, 15_000, true},
		{"absent with no upkeep", 5, 1, 0, 0, 0, 10_000, 0, false},
		// 0 - MinInt64 saturates to MaxInt64 and the quotient does too.
		{"relief of minimum saturates", 0, math.MinInt64, 1, 0, 1, 10_000, math.MaxInt64, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, present := welfare.EarnedRatio(tt.income, tt.relief, tt.cost, tt.n, tt.upkeep, tt.defBP)
			if got != tt.want || present != tt.present {
				t.Errorf("EarnedRatio(%d, %d, %d, %d, %d, %d) = %d, %v, want %d, %v",
					tt.income, tt.relief, tt.cost, tt.n, tt.upkeep, tt.defBP, got, present, tt.want, tt.present)
			}
		})
	}
}

// TestWage_ReproducesTheWorkedExamples verifies the table of 3.2 (B_r 1 000,
// D_year 365) at 250 and 150 working days.
//
// Requirement: FR-WEL-012
func TestWage_ReproducesTheWorkedExamples(t *testing.T) {
	t.Parallel()
	tests := []struct {
		ratio, dWork, want int64
	}{
		// 10 000 * 1 000 * 365 / (10 000 * 250) = 1 460.
		{10_000, 250, 1_460},
		{15_000, 250, 2_190},
		{20_000, 250, 2_920},
		{30_000, 250, 4_380},
		// At 150 days 3 650 000 000 / 1 500 000 = 2 433.33, truncated.
		{10_000, 150, 2_433},
		{15_000, 150, 3_650},
		// 7 300 000 000 / 1 500 000 = 4 866.67, truncated.
		{20_000, 150, 4_866},
		{30_000, 150, 7_300},
	}
	for _, tt := range tests {
		got, ok := welfare.Wage(tt.ratio, 1_000, 365, tt.dWork)
		if !ok || got != tt.want {
			t.Errorf("Wage(%d, 1000, 365, %d) = %d, %v, want %d, true", tt.ratio, tt.dWork, got, ok, tt.want)
		}
	}
}

// TestRatioForWage_RoundTripWithinOneUnit verifies the ratio a wage produces
// and that the round trip from a ratio to a wage and back loses at most the one
// unit of truncation WEL 3.2 states.
//
// Requirement: FR-WEL-012
func TestRatioForWage_RoundTripWithinOneUnit(t *testing.T) {
	t.Parallel()
	// 1 460 * 250 * 10 000 / (1 000 * 365) = 10 000.
	if got, ok := welfare.RatioForWage(1_460, 250, 1_000, 365); !ok || got != 10_000 {
		t.Errorf("RatioForWage(1460, 250, 1000, 365) = %d, %v, want 10000, true", got, ok)
	}
	// The wage 2 433 at 150 days is 0.33 below the exact 2 433.33, so the ratio
	// is 2 433 * 150 * 10 000 / 365 000 = 9 998.6, truncated to 9 998.
	if got, ok := welfare.RatioForWage(2_433, 150, 1_000, 365); !ok || got != 9_998 {
		t.Errorf("RatioForWage(2433, 150, 1000, 365) = %d, %v, want 9998, true", got, ok)
	}
	// A wage one unit higher overshoots: 2 434 * 150 * 10 000 / 365 000 = 10 002.7.
	if got, ok := welfare.RatioForWage(2_434, 150, 1_000, 365); !ok || got != 10_002 {
		t.Errorf("RatioForWage(2434, 150, 1000, 365) = %d, %v, want 10002, true", got, ok)
	}
}

// TestCalibration_ArgumentsAreTheOnlySource verifies that the year length, the
// days worked and the basket are arguments: changing each changes the result
// as the formula says.
//
// Requirement: FR-WEL-013
func TestCalibration_ArgumentsAreTheOnlySource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  func() (int64, bool)
		want int64
	}{
		// 10 000 * 1 000 * 360 / (10 000 * 200) = 1 800.
		{"wage with a 360-tick year and 200 days", func() (int64, bool) { return welfare.Wage(10_000, 1_000, 360, 200) }, 1_800},
		// 10 000 * 2 000 * 365 / (10 000 * 250) = 2 920.
		{"wage with a basket of 2 000", func() (int64, bool) { return welfare.Wage(10_000, 2_000, 365, 250) }, 2_920},
		// 1 000 * 100 * 10 000 / (500 * 200) = 10 000.
		{"ratio with another year and basket", func() (int64, bool) { return welfare.RatioForWage(1_000, 100, 500, 200) }, 10_000},
		// 15 000 * 1 000 * 100 / (10 000 * 3 000) = 50.
		{"days with a 100-tick year", func() (int64, bool) { return welfare.DaysWorked(15_000, 1_000, 100, 3_000) }, 50},
	}
	for _, tt := range tests {
		if got, ok := tt.got(); !ok || got != tt.want {
			t.Errorf("%s = %d, %v, want %d, true", tt.name, got, ok, tt.want)
		}
	}
}

// TestDaysWorked_ReproducesTheClosedForm verifies the illustrative table of
// section 5 (B_r 1 000, target 15 000, 365 ticks a year): the days are rounded
// up and capped at the year.
//
// Requirement: FR-WEL-022
func TestDaysWorked_ReproducesTheClosedForm(t *testing.T) {
	t.Parallel()
	tests := []struct {
		wage, want int64
	}{
		// numerator 15 000 * 1 000 * 365 = 5 475 000 000; over 10 000 * W.
		{1_000, 365}, // 547.5 -> 548, capped at 365
		{1_500, 365}, // 365 exactly
		{2_190, 250}, // 250 exactly
		{3_285, 167}, // 166.67 rounded up
		{4_380, 125}, // 125 exactly
		{6_570, 84},  // 83.33 rounded up
	}
	for _, tt := range tests {
		got, ok := welfare.DaysWorked(15_000, 1_000, 365, tt.wage)
		if !ok || got != tt.want {
			t.Errorf("DaysWorked(15000, 1000, 365, %d) = %d, %v, want %d, true", tt.wage, got, ok, tt.want)
		}
	}
}

// TestCalibration_ReportsOverflowAndZeroDivisors verifies that the formulas
// that run outside the tick report ok false instead of wrapping, saturating or
// dividing by zero.
//
// Requirement: FR-WEL-010
func TestCalibration_ReportsOverflowAndZeroDivisors(t *testing.T) {
	t.Parallel()
	const maxI = math.MaxInt64
	tests := []struct {
		name string
		f    func() (int64, bool)
	}{
		{"wage with no work days", func() (int64, bool) { return welfare.Wage(10_000, 1_000, 365, 0) }},
		{"wage numerator overflows", func() (int64, bool) { return welfare.Wage(maxI, 2, 365, 250) }},
		{"wage denominator overflows", func() (int64, bool) { return welfare.Wage(1, 1, 1, maxI) }},
		{"wage quotient overflows", func() (int64, bool) { return welfare.Wage(maxI, 1, maxI, 1) }},
		{"ratio for a wage with an empty basket", func() (int64, bool) { return welfare.RatioForWage(1, 1, 0, 365) }},
		{"ratio for a wage numerator overflows", func() (int64, bool) { return welfare.RatioForWage(maxI, 2, 1, 1) }},
		{"ratio for a wage denominator overflows", func() (int64, bool) { return welfare.RatioForWage(1, 1, maxI, 2) }},
		{"ratio for a wage quotient overflows", func() (int64, bool) { return welfare.RatioForWage(maxI, 1, 1, 1) }},
		{"days with a zero wage", func() (int64, bool) { return welfare.DaysWorked(15_000, 1_000, 365, 0) }},
		{"days with a negative wage", func() (int64, bool) { return welfare.DaysWorked(15_000, 1_000, 365, -5) }},
		{"days numerator overflows", func() (int64, bool) { return welfare.DaysWorked(maxI, 2, 365, 1) }},
		{"days denominator overflows", func() (int64, bool) { return welfare.DaysWorked(1, 1, 365, maxI) }},
		{"days quotient overflows", func() (int64, bool) { return welfare.DaysWorked(maxI, 1, maxI, 1) }},
	}
	for _, tt := range tests {
		if got, ok := tt.f(); ok || got != 0 {
			t.Errorf("%s = %d, %v, want 0, false", tt.name, got, ok)
		}
	}
}

// TestWorkShare_BandAndThreshold verifies the share of section 5 below the
// target, at it, inside the rest band, at its top and above, and the empty
// band.
//
// Requirement: FR-WEL-021
func TestWorkShare_BandAndThreshold(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		ratio, lo, band int64
		want            int64
	}{
		// T_hi = 15 000 * 10 500 / 10 000 = 15 750 for the first six rows.
		{"below the target", 14_999, 15_000, 500, 10_000},
		// (15 750 - 15 000) * 10 000 / 750.
		{"at the target", 15_000, 15_000, 500, 10_000},
		// (15 750 - 15 600) * 10 000 / 750 = 2 000.
		{"a fifth of the way", 15_600, 15_000, 500, 2_000},
		// (15 750 - 15 375) * 10 000 / 750 = 5 000.
		{"midway in the band", 15_375, 15_000, 500, 5_000},
		{"at the top of the band", 15_750, 15_000, 500, 0},
		{"above the band", 20_000, 15_000, 500, 0},
		{"band 0 below", 14_999, 15_000, 0, 10_000},
		{"band 0 at the target", 15_000, 15_000, 0, 0},
		// T_hi = 1 * 10 500 / 10 000 = 1 equals T_lo: the band is empty.
		{"band truncates to empty, below", 0, 1, 500, 10_000},
		{"band truncates to empty, at", 1, 1, 500, 0},
		{"negative ratio", -5_000, 15_000, 500, 10_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := welfare.WorkShare(tt.ratio, tt.lo, tt.band); got != tt.want {
				t.Errorf("WorkShare(%d, %d, %d) = %d, want %d", tt.ratio, tt.lo, tt.band, got, tt.want)
			}
		})
	}
}

// TestExecutions_CeilingOfTheShare verifies the executions requested by a
// cohort: the weight times the share, rounded up.
//
// Requirement: FR-WEL-021
func TestExecutions_CeilingOfTheShare(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		weight, share int64
		want          int64
	}{
		{"household at half rounds up to one", 1, 5_000, 1},
		{"household at zero works not", 1, 0, 0},
		{"household at full works", 1, 10_000, 1},
		// 40 * 5 000 / 10 000 = 20 exactly.
		{"cohort of 40 at half", 40, 5_000, 20},
		{"cohort of 40 at full", 40, 10_000, 40},
		// 40 * 1 / 10 000 = 0.004, rounded up.
		{"cohort of 40 at the least share", 40, 1, 1},
		// 3 * 3 333 / 10 000 = 0.9999, rounded up.
		{"cohort of 3 just below one", 3, 3_333, 1},
		{"cohort of 3 at zero", 3, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := welfare.Executions(tt.weight, tt.share); got != tt.want {
				t.Errorf("Executions(%d, %d) = %d, want %d", tt.weight, tt.share, got, tt.want)
			}
		})
	}
}

// TestFloorBinds_AtTheThreshold verifies the work floor: fewer working ticks
// than MinWorkShare * T / 10 000 in integer division binds, exactly the
// threshold does not, and a share of 0 turns it off.
//
// Requirement: FR-WEL-027
func TestFloorBinds_AtTheThreshold(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                   string
		working, window, minBP int64
		want                   bool
	}{
		// 4 000 * 30 / 10 000 = 12.
		{"exactly the threshold", 12, 30, 4_000, false},
		{"one below the threshold", 11, 30, 4_000, true},
		{"above the threshold", 30, 30, 4_000, false},
		{"no working ticks", 0, 30, 4_000, true},
		{"floor off", 0, 30, 0, false},
		// 4 000 * 365 / 10 000 = 146.
		{"year window", 145, 365, 4_000, true},
		// 4 000 * 1 / 10 000 = 0 in integer division: nothing is below 0.
		{"window of one truncates to zero", 0, 1, 4_000, false},
		// 3 333 * 3 / 10 000 = 0.99 truncated to 0.
		{"truncation toward zero", 0, 3, 3_333, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := welfare.FloorBinds(tt.working, tt.window, tt.minBP); got != tt.want {
				t.Errorf("FloorBinds(%d, %d, %d) = %v, want %v", tt.working, tt.window, tt.minBP, got, tt.want)
			}
		})
	}
}

// TestTargetRange_Spread verifies the range of the drawn target: lo equals hi
// without a spread, and the range is the truncated bounds of 5.1 with one.
//
// Requirement: FR-WEL-024, FR-WEL-025
func TestTargetRange_Spread(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		target, spread int64
		lo, hi         int64
	}{
		{"spread 0 is one value", 15_000, 0, 15_000, 15_000},
		// 15 000 * 8 500 / 10 000 = 12 750; 15 000 * 11 500 / 10 000 = 17 250.
		{"spread 1 500", 15_000, 1_500, 12_750, 17_250},
		// 15 000 * 1 / 10 000 = 1.5 truncated to 1; 15 000 * 19 999 / 10 000 = 29 998.5 truncated.
		{"spread 9 999 is the lowest target", 15_000, 9_999, 1, 29_998},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lo, hi := welfare.TargetRange(tt.target, tt.spread)
			if lo != tt.lo || hi != tt.hi {
				t.Errorf("TargetRange(%d, %d) = %d, %d, want %d, %d", tt.target, tt.spread, lo, hi, tt.lo, tt.hi)
			}
		})
	}
}

// TestWearCost_OneCeilingPerExecution verifies the wear cost of a row of
// executions: one ceiling per execution and then the product, with no weight.
//
// Requirement: FR-WEL-003, FR-WEL-010
//
// Guard: ceil(n * WearPerUse * price / Durability) is the tempting rewrite and
// differs from the rule.
func TestWearCost_OneCeilingPerExecution(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                                      string
		wear, price, durability, executions, want int64
	}{
		// 3 * 10 / 4 = 7.5 -> 8; 2 * 8 = 16, where ceil(2 * 30 / 4) = 15 would be wrong.
		{"two executions", 3, 10, 4, 2, 16},
		{"one execution", 3, 10, 4, 1, 8},
		// 1 * 7 / 10 = 0.7 -> 1; 3 * 1.
		{"a small wear still costs a unit", 1, 7, 10, 3, 3},
		// 5 * 20 / 10 = 10 exactly; 4 * 10.
		{"exact division", 5, 20, 10, 4, 40},
		{"no executions", 3, 10, 4, 0, 0},
		{"no price", 3, 0, 4, 5, 0},
		{"ceiling overflow saturates", math.MaxInt64, 2, 1, 1, math.MaxInt64},
		{"product overflow saturates", 1, 1, 1, math.MaxInt64, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := welfare.WearCost(tt.wear, tt.price, tt.durability, tt.executions); got != tt.want {
				t.Errorf("WearCost(%d, %d, %d, %d) = %d, want %d", tt.wear, tt.price, tt.durability, tt.executions, got, tt.want)
			}
		})
	}
}

// TestWearCost_ZeroDurabilityIsAnInvariant verifies that a zero durability,
// which validation refuses, panics with a message that says it is a violated
// programmer invariant.
//
// Guard: a zero divisor must not be read as an overflow and returned as a cost.
func TestWearCost_ZeroDurabilityIsAnInvariant(t *testing.T) {
	t.Parallel()
	defer func() {
		msg, _ := recover().(string)
		if !strings.Contains(msg, "violated programmer invariant") {
			t.Errorf("WearCost(3, 10, 0, 1) panicked with %q, want a violated programmer invariant", msg)
		}
	}()
	welfare.WearCost(3, 10, 0, 1)
}

// TestWearCost_ZeroDurabilityNamesItsFunction verifies that the zero
// durability is refused by WearCost's own guard, not by the divisor check of
// the saturating helper below it, so the message names WearCost.
//
// Guard: the caller must see which function's invariant was violated.
func TestWearCost_ZeroDurabilityNamesItsFunction(t *testing.T) {
	t.Parallel()
	defer func() {
		msg, _ := recover().(string)
		if !strings.Contains(msg, "WearCost") {
			t.Errorf("WearCost(3, 10, 0, 1) panicked with %q, want a message naming WearCost", msg)
		}
	}()
	welfare.WearCost(3, 10, 0, 1)
}

// TestTickCost_SumsTheUpkeepRows verifies the cost of a tick: quantity times
// price times weight per row, summed in slice order, saturating.
//
// Requirement: FR-WEL-004, FR-WEL-010
func TestTickCost_SumsTheUpkeepRows(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		qty, price []int64
		weight     int64
		want       int64
	}{
		// (2 * 5 + 1 * 7) * 3 = 51.
		{"two rows at weight 3", []int64{2, 1}, []int64{5, 7}, 3, 51},
		{"no rows", nil, nil, 3, 0},
		{"weight 0", []int64{2}, []int64{5}, 0, 0},
		{"a row that leaves int64 reads the maximum", []int64{math.MaxInt64}, []int64{2}, 1, math.MaxInt64},
		{"a weight that leaves int64 reads the maximum", []int64{2}, []int64{math.MaxInt64}, 2, math.MaxInt64},
		// The first row saturates, so the later rows keep the sum at the maximum.
		{"later rows keep a saturated sum", []int64{math.MaxInt64, 1, 1}, []int64{2, 1, 1}, 1, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := welfare.TickCost(tt.qty, tt.price, tt.weight); got != tt.want {
				t.Errorf("TickCost(%v, %v, %d) = %d, want %d", tt.qty, tt.price, tt.weight, got, tt.want)
			}
		})
	}
}

// TestTickCost_UnequalSlicesAreAnInvariant verifies that slices of unequal
// length panic with a message that says it is a violated programmer invariant.
//
// Guard: the slices are equal by construction; a silent truncation would drop
// an upkeep row from the cost.
func TestTickCost_UnequalSlicesAreAnInvariant(t *testing.T) {
	t.Parallel()
	defer func() {
		msg, _ := recover().(string)
		if !strings.Contains(msg, "violated programmer invariant") {
			t.Errorf("TickCost with unequal slices panicked with %q, want a violated programmer invariant", msg)
		}
	}()
	welfare.TickCost([]int64{1, 2}, []int64{1}, 1)
}

// TestTickCost_ShorterQuantitiesAreAnInvariant verifies that more prices than
// quantities also panic, not only more quantities than prices.
//
// Guard: a silent truncation would drop a priced row from the cost.
func TestTickCost_ShorterQuantitiesAreAnInvariant(t *testing.T) {
	t.Parallel()
	defer func() {
		msg, _ := recover().(string)
		if !strings.Contains(msg, "violated programmer invariant") {
			t.Errorf("TickCost with fewer quantities panicked with %q, want a violated programmer invariant", msg)
		}
	}()
	welfare.TickCost([]int64{1}, []int64{1, 2}, 1)
}

// TestTickIncome_CreditsLessDebits verifies the income of a tick: the credit
// sum less the debit sum, each sum in slice order, saturating by the sign rules.
//
// Requirement: FR-WEL-002, FR-WEL-010
func TestTickIncome_CreditsLessDebits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		credits, debits []int64
		want            int64
	}{
		// (900 + 40) - (200 + 40) = 700.
		{"hand case", []int64{900, 40}, []int64{200, 40}, 700},
		{"empty slices", nil, nil, 0},
		{"debits only", nil, []int64{30}, -30},
		{"credit at the maximum less a debit at the minimum", []int64{math.MaxInt64}, []int64{math.MinInt64}, math.MaxInt64},
		// -5 - MinInt64 = MaxInt64 - 4, in range: a sum with the negated debit would give MaxInt64 - 5.
		{"minus five less the minimum", []int64{-5}, []int64{math.MinInt64}, math.MaxInt64 - 4},
		// The sum is taken in order: MaxInt64 + 1 saturates, then - 5 gives MaxInt64 - 5, not MaxInt64 - 4.
		{"credits saturate in slice order", []int64{math.MaxInt64, 1, -5}, nil, math.MaxInt64 - 5},
		{"credits saturate down", []int64{math.MinInt64, -1}, nil, math.MinInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := welfare.TickIncome(tt.credits, tt.debits); got != tt.want {
				t.Errorf("TickIncome(%v, %v) = %d, want %d", tt.credits, tt.debits, got, tt.want)
			}
		})
	}
}

// TestHotPath_AllocatesNothing verifies that the ratio and the work share, the
// functions a tick calls per agent, allocate nothing.
//
// Requirement: NFR-WEL-001
func TestHotPath_AllocatesNothing(t *testing.T) {
	// No t.Parallel: AllocsPerRun must not overlap other tests' allocations.
	if n := testing.AllocsPerRun(100, func() {
		welfare.Ratio(450, 300, 3, 10, 15_000)
		welfare.WorkShare(15_300, 15_000, 500)
		welfare.Executions(40, 5_000)
		welfare.FloorBinds(5, 30, 4_000)
	}); n != 0 {
		t.Errorf("Ratio, WorkShare, Executions and FloorBinds allocated %v times per run, want 0", n)
	}
}

// BenchmarkRatio reports the cost of a padded ratio.
//
// Requirement: NFR-WEL-003
func BenchmarkRatio(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		welfare.Ratio(450, 300, 3, 10, 15_000)
	}
}

// BenchmarkWorkShare reports the cost of the share and the executions inside
// the rest band.
//
// Requirement: NFR-WEL-003
func BenchmarkWorkShare(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		welfare.Executions(40, welfare.WorkShare(15_300, 15_000, 500))
	}
}

// TestSetFunctions_PanicOnMismatchedSlices verifies that a caller that passes
// slices of different lengths to the set functions is a violated programmer
// invariant: the panic message says so, and nothing is computed from the
// shorter slice.
//
// Guard: the lengths are the caller's contract (parallel slices over one set of
// agents); a silent truncation would drop agents from a wage or a mean and show
// only as a wrong number. The panic is the intended loud failure, and
// testing.md 9.5 asks that a branch that cannot be reached by valid input is
// still exercised or explained.
//
// Requirement: FR-WEL-037, FR-WEL-036
func TestSetFunctions_PanicOnMismatchedSlices(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		f    func()
	}{
		{"SetWage numerators longer", func() { welfare.SetWage([]int64{1, 2}, []int64{1}) }},
		{"PooledWage denominators longer", func() { welfare.PooledWage([]int64{1}, []int64{1, 2}) }},
		{"MeanRatio ratios shorter", func() { welfare.MeanRatio([]int64{1, 1}, []int64{5}, []bool{true, true}) }},
		{"MeanRatio present shorter", func() { welfare.MeanRatio([]int64{1, 1}, []int64{5, 5}, []bool{true}) }},
		{"MeanRatio weights shorter than ratios", func() { welfare.MeanRatio([]int64{1}, []int64{5, 5}, []bool{true}) }},
		{"MeanRatio weights shorter than present", func() { welfare.MeanRatio([]int64{1}, []int64{5}, []bool{true, true}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				r := recover()
				msg, _ := r.(string)
				if !strings.Contains(msg, "violated programmer invariant") {
					t.Errorf("%s: recovered %v, want a panic saying \"violated programmer invariant\"", tt.name, r)
				}
			}()
			tt.f()
		})
	}
}

// TestFloorWage_RaisesTheWageToAShareOfTheSeed verifies that the wage the cost
// anchor reads is at least floorBP basis points of the starting wage, truncated
// toward zero, and is the wage itself when the floor is off (floorBP 0), when
// there is no starting wage, or when the wage is already above the floor.
//
// Arithmetic: seed 2 512 at 5 000 bp is 2 512 * 5 000 / 10 000 = 1 256; at 3 333
// bp it is 2 512 * 3 333 / 10 000 = 837.3, truncated to 837. A seed near the top
// of int64 at 10 000 bp is the seed itself, not an overflow.
//
// Requirement: FR-PRO-055
//
// Fix: the cost anchor read the pooled observed wage alone, so a price crash that
// lowered the wage lowered the anchor and the belief floor with it.
func TestFloorWage_RaisesTheWageToAShareOfTheSeed(t *testing.T) {
	t.Parallel()
	const maxI = math.MaxInt64
	tests := []struct {
		name                string
		wage, seed, floorBP int64
		want                int64
	}{
		{"below the floor is raised", 80, 2_512, 5_000, 1_256},
		{"at the floor is unchanged", 1_256, 2_512, 5_000, 1_256},
		{"above the floor is unchanged", 3_000, 2_512, 5_000, 3_000},
		{"the floor truncates toward zero", 1, 2_512, 3_333, 837},
		{"floor off", 80, 2_512, 0, 80},
		{"no starting wage", 80, 0, 5_000, 80},
		{"negative starting wage", 80, -1, 5_000, 80},
		{"whole seed at 10 000", 80, maxI - 1, 10_000, maxI - 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := welfare.FloorWage(tt.wage, tt.seed, tt.floorBP); got != tt.want {
				t.Errorf("FloorWage(%d, %d, %d) = %d, want %d", tt.wage, tt.seed, tt.floorBP, got, tt.want)
			}
		})
	}
}
