package technology_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/technology"
)

// TestWeight_Table verifies the contact weight: (reach - d) * 10 000 / reach
// for 1 <= d < reach, truncating, and 0 at or past the reach, with no path,
// and for a distance below 1 or a reach below 1.
//
// The arithmetic at reach 100: d 20 gives 80 * 10 000 / 100 = 8 000, d 60 gives
// 4 000, d 50 gives 5 000, d 99 gives 100; at reach 7 and d 3, 4 * 10 000 / 7
// = 5 714 (truncated). At the largest reach 10^9 and d 1 the product is (10^9 - 1) * 10^4 < 2^63 and the weight is 9 999 (truncated).
//
// Requirement: FR-TEC-014
func TestWeight_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		reach, d int64
		ok       bool
		want     int64
	}{
		{"near", 100, 20, true, 8000},
		{"far", 100, 60, true, 4000},
		{"middle", 100, 50, true, 5000},
		{"just inside the reach", 100, 99, true, 100},
		{"at the reach", 100, 100, true, 0},
		{"past the reach", 100, 101, true, 0},
		{"no path", 100, 20, false, 0},
		{"distance zero", 100, 0, true, 0},
		{"reach zero", 0, 5, true, 0},
		{"truncates", 7, 3, true, 5714},
		{"largest reach", 1_000_000_000, 1, true, 9999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := technology.Weight(tt.reach, tt.d, tt.ok); got != tt.want {
				t.Errorf("Weight(%d, %d, %v) = %d, want %d", tt.reach, tt.d, tt.ok, got, tt.want)
			}
		})
	}
}

// TestCatchUp_CarryAndNoGap verifies the catch-up: the carry takes
// gap * CatchUpBP, the add is the whole units of 10 000 * ticksPerYear, and a
// gap of 0 or below adds nothing and keeps the carry.
//
// The arithmetic: gap 1 200 at 2 000 BP is 2 400 000; over 10 000 * 365 =
// 3 650 000 that is 0 whole units and a carry of 2 400 000; a second tick with
// the same gap gives 4 800 000, so add 1 and carry 1 150 000.
//
// Requirement: FR-TEC-015
func TestCatchUp_CarryAndNoGap(t *testing.T) {
	t.Parallel()
	add, c, ok := technology.CatchUp(0, 1200, 2000, 365)
	if !ok || add != 0 || c != 2_400_000 {
		t.Errorf("CatchUp(0, 1200, 2000, 365) = %d, %d, %v, want 0, 2400000, true", add, c, ok)
	}
	add, c, ok = technology.CatchUp(c, 1200, 2000, 365)
	if !ok || add != 1 || c != 1_150_000 {
		t.Errorf("second tick = %d, %d, %v, want 1, 1150000, true", add, c, ok)
	}
	for _, gap := range []int64{0, -50} {
		add, c, ok = technology.CatchUp(777, gap, 2000, 365)
		if !ok || add != 0 || c != 777 {
			t.Errorf("CatchUp(777, %d, 2000, 365) = %d, %d, %v, want 0, 777, true", gap, add, c, ok)
		}
	}
}

// TestCatchUp_Refusals verifies that a negative carry or rate, no ticks per
// year and a product that overflows are refused.
//
// Requirement: NFR-TEC-005
func TestCatchUp_Refusals(t *testing.T) {
	t.Parallel()
	for _, bad := range [][4]int64{{-1, 1, 1, 365}, {0, 1, -1, 365}, {0, 1, 1, 0}, {0, math.MaxInt64, 2, 365}} {
		if _, _, ok := technology.CatchUp(bad[0], bad[1], bad[2], bad[3]); ok {
			t.Errorf("CatchUp(%v) accepted, want a refusal", bad)
		}
	}
}

// TestFrontier_MaxOverOthers verifies the frontier: the maximum over the other
// regions of gain * weight / 10 000, excluding the region itself, with no path
// as no contact, and a refusal on overflow.
//
// The arithmetic: gains 3 000 (r0), 2 000 (r1), 0 (r2); reach 100; d(0,2) 60
// gives 3 000 * 4 000 / 10 000 = 1 200 and d(1,2) 50 gives 2 000 * 5 000 /
// 10 000 = 1 000, so the frontier of r2 is 1 200. A gain at MaxInt64 times the
// largest weight, 9 999, stays below the gain: the 128-bit product cannot overflow.
//
// Requirement: FR-TEC-013, FR-TEC-014
func TestFrontier_MaxOverOthers(t *testing.T) {
	t.Parallel()
	d := [3][3]int64{{0, 20, 60}, {20, 0, 50}, {60, 50, 0}}
	dist := func(a, b int) (int64, bool) { return d[a][b], true }
	gains := []int64{3000, 2000, 0}
	if got := technology.Frontier(2, gains, 100, dist); got != 1200 {
		t.Errorf("Frontier(2) = %d, want 1200", got)
	}
	if got := technology.Frontier(0, gains, 100, dist); got != 1600 {
		t.Errorf("Frontier(0) = %d, want 1600 (2000 * 8000 / 10000)", got)
	}
	none := func(a, b int) (int64, bool) { return 0, false }
	if got := technology.Frontier(2, gains, 100, none); got != 0 {
		t.Errorf("Frontier with no path = %d, want 0", got)
	}
	big := []int64{math.MaxInt64, 0}
	near := func(a, b int) (int64, bool) { return 1, true }
	if got, want := technology.Frontier(1, big, 1_000_000_000, near), int64(math.MaxInt64/10_000*9_999); got < want {
		t.Errorf("Frontier at MaxInt64 = %d, want at least %d (no overflow, weight 9999)", got, want)
	}
}
