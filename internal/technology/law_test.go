package technology_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/technology"
)

// TestLg12_Table verifies the fixed-point logarithm against hand-worked
// values: exact at a power of two, linear in the twelve bits below the leading
// one between two, and the value of 1 for an argument below 1.
//
// The arithmetic of each row: lg12(v) = 4096*k + (top12(v) - 4096) with k the
// floor of log2(v). v = 3: k = 1, top12 = 3 << 11 = 6144, so 4096 + 2048 =
// 6144. v = 4097: k = 12, top12 = 4097, so 49152 + 1. v = 8191: k = 12, top12 =
// 8191, so 49152 + 4095. v = 2^13 + 1: k = 13, top12 = (2^13 + 1) >> 1 = 4096,
// so 13 * 4096 = 53248.
//
// Requirement: FR-TEC-006
func TestLg12_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		v    int64
		want int64
	}{
		{"zero is the value of one", 0, 0},
		{"negative is the value of one", -5, 0},
		{"one", 1, 0},
		{"two", 2, 4096},
		{"three", 3, 6144},
		{"a power of two", 1 << 12, 12 * 4096},
		{"just above a power of two", 4097, 49153},
		{"just below the next", 8191, 49152 + 4095},
		{"truncates the low bits above twelve", 1<<13 + 1, 13 * 4096},
		{"largest", math.MaxInt64, 62*4096 + 4095},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := technology.Lg12(tt.v); got != tt.want {
				t.Errorf("Lg12(%d) = %d, want %d", tt.v, got, tt.want)
			}
		})
	}
}

// TestLearned_ExactAtDoublings verifies that the learned gain is exactly
// j*DoublingBP when X + Base = 2^j * Base (models/tec/law.out part 1: exact at
// 21 doublings with Base 100 000 and DoublingBP 200), 0 at X = 0, and
// non-decreasing in X.
//
// Requirement: FR-TEC-006, FR-TEC-005
func TestLearned_ExactAtDoublings(t *testing.T) {
	t.Parallel()
	const base, d = 100_000, 200
	for j := int64(0); j <= 21; j++ {
		x := int64(base) * ((1 << j) - 1)
		got, ok := technology.Learned(d, base, x)
		if want := j * d; !ok || got != want {
			t.Errorf("Learned(%d, %d, %d) = %d, %v, want %d, true (doubling %d)", d, base, x, got, ok, want, j)
		}
	}
	prev := int64(-1)
	for x := int64(0); x < 3_000_000; x += 977 {
		got, _ := technology.Learned(d, base, x)
		if got < prev {
			t.Fatalf("Learned(%d, %d, %d) = %d fell below %d: the gain must not fall", d, base, x, got, prev)
		}
		prev = got
	}
}

// TestLearned_ErrorsAndBounds verifies the refusal form and the largest
// products of spec 9.5: a base below 1, a negative X or rate and an overflowing
// sum are refused, and the bounds (X = 2^50, Base 1, DoublingBP 10 000; Base
// 10^12) give their stated values with no overflow.
//
// The arithmetic: at X = 2^50 and Base 1, v = 2^50 + 1 has k = 50 and top12 =
// 4096, so lg12 = 50 * 4096 and the gain is 10 000 * 50 = 500 000 (law.out
// part 1).
//
// Requirement: NFR-TEC-005, FR-TEC-035
func TestLearned_ErrorsAndBounds(t *testing.T) {
	t.Parallel()
	refused := []struct {
		name       string
		d, base, x int64
	}{
		{"base zero", 200, 0, 5},
		{"negative x", 200, 10, -1},
		{"negative rate", -1, 10, 5},
		{"sum overflows", 200, math.MaxInt64, 1},
	}
	for _, tt := range refused {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := technology.Learned(tt.d, tt.base, tt.x); ok || got != 0 {
				t.Errorf("Learned(%d, %d, %d) = %d, %v, want 0, false", tt.d, tt.base, tt.x, got, ok)
			}
		})
	}
	if got, ok := technology.Learned(10_000, 1, 1<<50); !ok || got != 500_000 {
		t.Errorf("Learned(10000, 1, 2^50) = %d, %v, want 500000, true", got, ok)
	}
	if _, ok := technology.Learned(10_000, 1_000_000_000_000, 1<<50); !ok {
		t.Errorf("Learned(10000, 10^12, 2^50) refused: X + Base fits int64 at the bounds")
	}
}

// TestTrend_ExactOverYears verifies the carried remainder: over three years of
// 365 ticks a trend of 137 adds exactly 411 with no remainder, and a single
// tick adds floor((carry + trend) / ticks) (models/tec/law.out part 2).
//
// Requirement: FR-TEC-008
func TestTrend_ExactOverYears(t *testing.T) {
	t.Parallel()
	for _, trend := range []int64{1, 137, 200, 9999} {
		var carry, total int64
		for range 3 * 365 {
			add, c2, ok := technology.TrendStep(carry, trend, 365)
			if !ok {
				t.Fatalf("TrendStep(%d, %d, 365) refused", carry, trend)
			}
			carry, total = c2, total+add
		}
		if total != 3*trend || carry != 0 {
			t.Errorf("trend %d over three years = %d carry %d, want %d carry 0", trend, total, carry, 3*trend)
		}
	}
	add, c2, ok := technology.TrendStep(300, 100, 365)
	if !ok || add != 1 || c2 != 35 {
		t.Errorf("TrendStep(300, 100, 365) = %d, %d, %v, want 1, 35, true", add, c2, ok)
	}
}

// TestTrend_Refusals verifies that a negative carry or trend, no ticks per
// year and a sum that overflows are refused.
//
// Requirement: NFR-TEC-005
func TestTrend_Refusals(t *testing.T) {
	t.Parallel()
	for _, bad := range [][3]int64{{-1, 1, 365}, {0, -1, 365}, {0, 1, 0}, {math.MaxInt64, 1, 365}} {
		if _, _, ok := technology.TrendStep(bad[0], bad[1], bad[2]); ok {
			t.Errorf("TrendStep(%d, %d, %d) accepted, want a refusal", bad[0], bad[1], bad[2])
		}
	}
}
