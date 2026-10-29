package ownuse_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ownuse"
)

// TestCov_Truncation128 verifies the coverage q * E / (Duration * u) in basis
// points: the worked values of spec-subsistence.md 16.1 and 16.2, truncation
// toward zero, a product that needs 128 bits (and a divisor that does not fit
// int64, never formed), and the refusal of bad inputs and of a result beyond
// int64, which saturates.
//
// Requirement: FR-SUB-039
func TestCov_Truncation128(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                        string
		q, factor, duration, upkeep int64
		want                        int64
		wantOK                      bool
	}{
		// 5 * 7 500 / (1 * 3) = 12 500 (16.1).
		{"example A", 5, 7500, 1, 3, 12_500, true},
		// 15 * 8 000 / (3 * 4) = 10 000 (16.2).
		{"example B", 15, 8000, 3, 4, 10_000, true},
		// 1 * 10 000 / (3 * 1) = 3 333.33, truncated.
		{"truncates", 1, 10_000, 3, 1, 3333, true},
		// 7 * 1 / (2 * 5) = 0.7, truncated to 0.
		{"below one is zero", 7, 1, 2, 5, 0, true},
		// 2^60 * 10 000 = 1.15 * 10^22 exceeds 64 bits; / (2^20 * 2^20) = 2^20 * 10 000 = 10 485 760 000.
		{"product beyond 64 bits", 1 << 60, 10_000, 1 << 20, 1 << 20, 10_485_760_000, true},
		// 2^40 * 2^40 = 2^80 does not fit int64: the divisor is never formed.
		{"divisor beyond int64", 1 << 60, 10_000, 1 << 40, 1 << 40, 0, true},
		{"result beyond int64 saturates", math.MaxInt64, math.MaxInt64, 1, 1, math.MaxInt64, false},
		{"zero output", 0, 10_000, 1, 3, 0, true},
		{"zero factor", 5, 0, 1, 3, 0, true},
		{"zero duration", 5, 10_000, 0, 3, 0, false},
		{"zero upkeep", 5, 10_000, 1, 0, 0, false},
		{"negative output", -1, 10_000, 1, 3, 0, false},
		{"negative factor", 5, -1, 1, 3, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ownuse.Cov(tt.q, tt.factor, tt.duration, tt.upkeep)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Cov(%d, %d, %d, %d) = %d, %v, want %d, %v", tt.q, tt.factor, tt.duration, tt.upkeep, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestFloorBP_MaxOverCovered verifies that the floor figure of a recipe is the
// greatest coverage over the covered goods, and 0 for none.
//
// Requirement: FR-SUB-039
func TestFloorBP_MaxOverCovered(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		covs []int64
		want int64
	}{
		{"one", []int64{12_500}, 12_500},
		{"greatest of three", []int64{3000, 12_500, 9000}, 12_500},
		{"a good the recipe does not make counts 0", []int64{0, 4000}, 4000},
		{"none", nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ownuse.FloorBP(tt.covs); got != tt.want {
				t.Errorf("FloorBP(%v) = %d, want %d", tt.covs, got, tt.want)
			}
		})
	}
}

// TestPick_GreatestSumTiesToLowerID verifies the choice among own-use recipes:
// the greatest sum of the truncated coverages over the short goods, a tie to
// the lower reference, only short goods counted, and a sum that overflows
// saturating instead of wrapping into a loss.
//
// The goods are (grain, fuel): grain is short, fuel is not.
//
// Requirement: FR-SUB-020
func TestPick_GreatestSumTiesToLowerID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		opts  []ownuse.Option
		short []bool
		want  int
	}{
		{"greatest sum", []ownuse.Option{
			{Ref: 3, Max: 1, Cov: []int64{4000, 0}},
			{Ref: 5, Max: 1, Cov: []int64{12_500, 0}},
			{Ref: 7, Max: 1, Cov: []int64{9000, 0}},
		}, []bool{true, false}, 5},
		// The better recipe comes first and has the higher reference; the weaker one
		// after it has the lower reference but must not win on that.
		{"a lower reference does not beat a greater sum", []ownuse.Option{
			{Ref: 5, Max: 1, Cov: []int64{12_500, 0}},
			{Ref: 3, Max: 1, Cov: []int64{4000, 0}},
		}, []bool{true, false}, 5},
		{"several executions can start", []ownuse.Option{
			{Ref: 1, Max: 3, Cov: []int64{7000, 0}},
		}, []bool{true, false}, 1},
		{"tie goes to the lower reference, negative references", []ownuse.Option{
			{Ref: 3, Max: 1, Cov: []int64{5000, 0}},
			{Ref: -2, Max: 1, Cov: []int64{5000, 0}},
		}, []bool{true, false}, -2},
		// Fuel (the first good) is not short and grain (the second) is: a recipe is
		// judged on the short goods wherever they sit in the covered set.
		{"a good that is not short comes first", []ownuse.Option{
			{Ref: 1, Max: 1, Cov: []int64{10_000, 3000}},
			{Ref: 2, Max: 1, Cov: []int64{0, 2000}},
		}, []bool{false, true}, 1},
		{"tie goes to the lower reference", []ownuse.Option{
			{Ref: 9, Max: 1, Cov: []int64{5000, 0}},
			{Ref: 2, Max: 1, Cov: []int64{5000, 0}},
			{Ref: 6, Max: 1, Cov: []int64{5000, 0}},
		}, []bool{true, false}, 2},
		// Only grain is short: recipe 1 covers fuel (10 000), recipe 4 grain (3 000).
		{"a good that is not short does not count", []ownuse.Option{
			{Ref: 1, Max: 1, Cov: []int64{0, 10_000}},
			{Ref: 4, Max: 1, Cov: []int64{3000, 0}},
		}, []bool{true, false}, 4},
		// Both short: 6 000 + 5 000 = 11 000 beats 10 000.
		{"sums over the short goods", []ownuse.Option{
			{Ref: 1, Max: 1, Cov: []int64{10_000, 0}},
			{Ref: 2, Max: 1, Cov: []int64{6000, 5000}},
		}, []bool{true, true}, 2},
		// 4 999 + 4 999 truncated is 9 998 for the sum; the exact fractions would not be used.
		{"truncated coverages are summed as integers", []ownuse.Option{
			{Ref: 1, Max: 1, Cov: []int64{9999, 0}},
			{Ref: 2, Max: 1, Cov: []int64{4999, 4999}},
		}, []bool{true, true}, 1},
		{"a recipe that cannot start is skipped", []ownuse.Option{
			{Ref: 1, Max: 0, Cov: []int64{20_000, 0}},
			{Ref: 2, Max: 1, Cov: []int64{1, 0}},
		}, []bool{true, false}, 2},
		{"a sum beyond int64 saturates", []ownuse.Option{
			{Ref: 1, Max: 1, Cov: []int64{math.MaxInt64, math.MaxInt64}},
			{Ref: 2, Max: 1, Cov: []int64{5000, 0}},
		}, []bool{true, true}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ownuse.Pick(tt.opts, tt.short)
			if !ok || got != tt.want {
				t.Errorf("Pick(%v, %v) = %d, %v, want %d, true", tt.opts, tt.short, got, ok, tt.want)
			}
		})
	}
}

// TestPick_NeverZero verifies that no recipe is returned when none has a
// positive sum over the short goods or an execution to start: the household is
// then blocked (5.5 item 6 case a), and a recipe of value 0 is never requested.
//
// Requirement: FR-SUB-020, FR-SUB-023
func TestPick_NeverZero(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		opts  []ownuse.Option
		short []bool
	}{
		{"no option", nil, []bool{true}},
		{"sum zero", []ownuse.Option{{Ref: 1, Max: 3, Cov: []int64{0, 0}}}, []bool{true, true}},
		{"coverage only of goods that are not short", []ownuse.Option{{Ref: 1, Max: 3, Cov: []int64{8000}}}, []bool{false}},
		{"no execution", []ownuse.Option{{Ref: 1, Max: 0, Cov: []int64{8000}}}, []bool{true}},
		{"negative executions", []ownuse.Option{{Ref: 1, Max: -2, Cov: []int64{8000}}}, []bool{true}},
		{"short shorter than cov", []ownuse.Option{{Ref: 1, Max: 1, Cov: []int64{0, 8000}}}, []bool{true}},
		{"short much shorter than cov", []ownuse.Option{{Ref: 1, Max: 1, Cov: []int64{0, 0, 8000}}}, []bool{true}},
		{"a zero-sum recipe with a negative reference", []ownuse.Option{{Ref: -1, Max: 1, Cov: []int64{0}}}, []bool{true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := ownuse.Pick(tt.opts, tt.short); ok {
				t.Errorf("Pick(%v, %v) = %d, true, want no recipe", tt.opts, tt.short, got)
			}
		})
	}
}

// TestRamp_Formula verifies k = min(w, maxExec, max(1, ceil(w * (exit - cover)
// / exit))): a weight-1 agent runs 1, the execution cap binds, a cover at or
// above the exit still runs 1, and a bad weight or exit is refused.
//
// Requirement: FR-SUB-021
func TestRamp_Formula(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                 string
		w, maxExec, cover, x int64
		want                 int64
		wantOK               bool
	}{
		{"weight one", 1, 5, 1, 3, 1, true},
		{"weight one empty stock", 1, 5, 0, 3, 1, true},
		// ceil(40 * 3 / 3) = 40.
		{"cohort with no cover", 40, 100, 0, 3, 40, true},
		{"the execution cap binds", 40, 10, 0, 3, 10, true},
		{"cover at the exit still runs one", 40, 100, 3, 3, 1, true},
		{"cover above the exit still runs one", 40, 100, 9, 3, 1, true},
		{"weight bounds the executions", 3, 100, 0, 3, 3, true},
		// ceil(5 * 1 / 1) = 5: an exit cover of 1 is valid.
		{"exit of one", 5, 9, 0, 1, 5, true},
		{"weight of one with an exit of one", 1, 9, 0, 1, 1, true},
		{"zero weight", 0, 5, 0, 3, 0, false},
		{"zero exit", 5, 5, 0, 0, 0, false},
		{"overflow of the share", math.MaxInt64, math.MaxInt64, -1_000_000, 1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ownuse.Ramp(tt.w, tt.maxExec, tt.cover, tt.x)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Ramp(%d, %d, %d, %d) = %d, %v, want %d, %v", tt.w, tt.maxExec, tt.cover, tt.x, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestRamp_CohortSteps verifies the steps of a cohort of 40 with an exit cover
// of 3 (spec-subsistence.md 16.1, 16.2; SUB-D5): cover 0 puts all 40 on the
// floor, cover 1 puts ceil(40 * 2 / 3) = 27, cover 2 puts ceil(40 / 3) = 14, so
// its labour moves in steps and not all at once; the 27 requests draw 54 units
// of a resource that takes 2 an execution.
//
// Requirement: FR-SUB-021
func TestRamp_CohortSteps(t *testing.T) {
	t.Parallel()
	want := map[int64]int64{0: 40, 1: 27, 2: 14}
	for cover := int64(0); cover <= 2; cover++ {
		got, ok := ownuse.Ramp(40, 100, cover, 3)
		if !ok || got != want[cover] {
			t.Errorf("Ramp(40, 100, %d, 3) = %d, %v, want %d", cover, got, ok, want[cover])
		}
	}
	if k, _ := ownuse.Ramp(40, 100, 1, 3); k*2 != 54 {
		t.Errorf("27 executions draw %d units, want 54", k*2)
	}
}

// TestRetryDue verifies that the retry fires at RetryJobs own-use jobs in a
// row and after, never when RetryJobs is 0.
//
// Requirement: FR-SUB-022
func TestRetryDue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		retry, jobs int64
		want        bool
	}{
		{"below", 10, 9, false},
		{"at", 10, 10, true},
		{"above", 10, 25, true},
		{"never when 0", 0, 1000, false},
		{"one", 1, 1, true},
		{"one with none started", 1, 0, false},
		{"negative retry never fires", -1, 5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ownuse.RetryDue(tt.retry, tt.jobs); got != tt.want {
				t.Errorf("RetryDue(%d, %d) = %v, want %v", tt.retry, tt.jobs, got, tt.want)
			}
		})
	}
}
