package design_test

import (
	"math"
	"math/big"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/design"
)

// TestGrid_IndexAndInverse verifies the grid order of doc/specs/models/exp/worked.out
// section A: 4 levels of cost_floor by 3 of belief_step make 12 points, the last
// factor fastest, point 5 is (8 000, 400) i.e. level coordinates (1, 2), and the
// coordinates of every index invert the mixed-radix number. A factor-free design
// has one point.
//
// Requirement: FR-EXP-005
func TestGrid_IndexAndInverse(t *testing.T) {
	t.Parallel()
	levels := []int{4, 3}
	n, ok := design.GridCount(levels)
	if !ok || n != 12 {
		t.Fatalf("GridCount(%v) = %d, %v, want 12, true", levels, n, ok)
	}
	// worked.out A: cost_floor 7000.. by level 0..3, belief_step 100, 200, 400 by level 0..2.
	want := [][]int{
		{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2},
		{2, 0}, {2, 1}, {2, 2}, {3, 0}, {3, 1}, {3, 2},
	}
	var dst []int
	for i, w := range want {
		got, ok := design.GridCoords(i, levels, dst)
		if !ok || !slices.Equal(got, w) {
			t.Errorf("GridCoords(%d, %v) = %v, %v, want %v", i, levels, got, ok, w)
		}
		// the inverse: index = ((l1 * L2) + l2)
		if inv := got[0]*levels[1] + got[1]; inv != i {
			t.Errorf("coordinates %v of point %d invert to %d", got, i, inv)
		}
		dst = got
	}
	if got, _ := design.GridCoords(5, levels, nil); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("GridCoords(5, %v) = %v, want [1 2]: (8 000, 400)", levels, got)
	}
	// worked.out G: 4 x 3 x 2 x 5 = 120 points.
	if n, ok := design.GridCount([]int{4, 3, 2, 5}); !ok || n != 120 {
		t.Errorf("GridCount(4 3 2 5) = %d, %v, want 120", n, ok)
	}
	if n, ok := design.GridCount(nil); !ok || n != 1 {
		t.Errorf("GridCount(none) = %d, %v, want 1: a factor-free design is one point", n, ok)
	}
	if got, ok := design.GridCoords(0, nil, nil); !ok || len(got) != 0 {
		t.Errorf("GridCoords(0, none) = %v, %v, want an empty slice", got, ok)
	}
	for _, i := range []int{-1, 12} {
		if got, ok := design.GridCoords(i, levels, nil); ok {
			t.Errorf("GridCoords(%d, %v) = %v, true, want false", i, levels, got)
		}
	}
	for _, bad := range [][]int{{0, 3}, {-1}, {math.MaxInt, 2}} {
		if got, ok := design.GridCount(bad); ok {
			t.Errorf("GridCount(%v) = %d, true, want false", bad, got)
		}
	}
}

// TestOneAtATime_Order verifies worked.out section B: the same factors give
// 1 + 3 + 2 = 6 points, point 0 the baseline, then each factor's levels above 0
// with the others at 0. A single-level factor adds none.
//
// Requirement: FR-EXP-005
func TestOneAtATime_Order(t *testing.T) {
	t.Parallel()
	levels := []int{4, 3}
	n, ok := design.OATCount(levels)
	if !ok || n != 6 {
		t.Fatalf("OATCount(%v) = %d, %v, want 6 = 1 + 3 + 2", levels, n, ok)
	}
	want := [][]int{{0, 0}, {1, 0}, {2, 0}, {3, 0}, {0, 1}, {0, 2}}
	for i, w := range want {
		if got, ok := design.OATCoords(i, levels, nil); !ok || !slices.Equal(got, w) {
			t.Errorf("OATCoords(%d, %v) = %v, %v, want %v", i, levels, got, ok, w)
		}
	}
	// worked.out G: 4 factors of 4, 3, 2 and 5 levels give 11 points.
	if n, ok := design.OATCount([]int{4, 3, 2, 5}); !ok || n != 11 {
		t.Errorf("OATCount(4 3 2 5) = %d, %v, want 11", n, ok)
	}
	if got, ok := design.OATCoords(2, []int{1, 3, 1, 2}, nil); !ok || !slices.Equal(got, []int{0, 2, 0, 0}) {
		t.Errorf("OATCoords(2, 1 3 1 2) = %v, %v, want [0 2 0 0]", got, ok)
	}
	if got, ok := design.OATCoords(3, []int{1, 3, 1, 2}, nil); !ok || !slices.Equal(got, []int{0, 0, 0, 1}) {
		t.Errorf("OATCoords(3, 1 3 1 2) = %v, %v, want [0 0 0 1]: factors of one level add no point", got, ok)
	}
	for _, i := range []int{-1, 6} {
		if got, ok := design.OATCoords(i, levels, nil); ok {
			t.Errorf("OATCoords(%d, %v) = %v, true, want false", i, levels, got)
		}
	}
	if n, ok := design.OATCount(nil); !ok || n != 1 {
		t.Errorf("OATCount(none) = %d, %v, want 1", n, ok)
	}
	for _, bad := range [][]int{{0}, {-2, 3}, {math.MaxInt, math.MaxInt}} {
		if got, ok := design.OATCount(bad); ok {
			t.Errorf("OATCount(%v) = %d, true, want false", bad, got)
		}
	}
}

// TestRangeValue_Grid verifies the levels of a range factor: the end points are
// exact, the interior is rounded half to even, and the full int64 range does not
// wrap (checked against big.Int).
//
// Requirement: FR-EXP-012
func TestRangeValue_Grid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		k, steps     int
		lo, hi, want int64
	}{
		{"first end point", 0, 4, 7000, 10000, 7000},
		{"last end point", 3, 4, 7000, 10000, 10000},
		{"interior exact", 1, 4, 7000, 10000, 8000},
		// 7000 + rdiv(1 * 3, 2) = 7000 + 1.5 -> 2 (to the even 2)
		{"tie goes to even, up", 1, 3, 7000, 7003, 7002},
		// 100 + rdiv(1 * 5, 2) = 100 + 2.5 -> 2 (even)
		{"tie goes to even, down", 1, 3, 100, 105, 102},
		// 1 * 7 / 3 = 2.33 -> 2; 2 * 7 / 3 = 4.67 -> 5
		{"below and above a half", 1, 4, 0, 7, 2},
		{"above a half", 2, 4, 0, 7, 5},
		{"negative range", 1, 3, -10, 10, 0},
		{"single value range", 1, 3, 5, 5, 5},
		{"two steps", 1, 2, -3, 3, 3},
	}
	for _, tt := range cases {
		if got, ok := design.RangeValue(tt.k, tt.steps, tt.lo, tt.hi); !ok || got != tt.want {
			t.Errorf("%s: RangeValue(%d, %d, %d, %d) = %d, %v, want %d", tt.name, tt.k, tt.steps, tt.lo, tt.hi, got, ok, tt.want)
		}
	}
	// The whole int64 range, 2^64 - 1 wide: the 128-bit product must not wrap.
	for steps := 2; steps <= 9; steps++ {
		for k := range steps {
			width := new(big.Int).Sub(big.NewInt(math.MaxInt64), big.NewInt(math.MinInt64))
			num := new(big.Int).Mul(big.NewInt(int64(k)), width)
			q, r := new(big.Int).QuoRem(num, big.NewInt(int64(steps-1)), new(big.Int))
			twice := new(big.Int).Lsh(r, 1)
			if c := twice.Cmp(big.NewInt(int64(steps - 1))); c > 0 || (c == 0 && q.Bit(0) == 1) {
				q.Add(q, big.NewInt(1))
			}
			want := q.Add(q, big.NewInt(math.MinInt64)).Int64()
			if got, ok := design.RangeValue(k, steps, math.MinInt64, math.MaxInt64); !ok || got != want {
				t.Errorf("RangeValue(%d, %d, MinInt64, MaxInt64) = %d, %v, want %d", k, steps, got, ok, want)
			}
		}
	}
	for _, bad := range [][4]int64{{0, 1, 0, 5}, {-1, 3, 0, 5}, {3, 3, 0, 5}, {0, 3, 5, 4}} {
		if got, ok := design.RangeValue(int(bad[0]), int(bad[1]), bad[2], bad[3]); ok {
			t.Errorf("RangeValue(%v) = %d, true, want false", bad, got)
		}
	}
}

// TestGrid_GuardsAndScratch verifies the argument guards at their edges and the
// reuse of the scratch slice: a level count of exactly 1 is valid, an index just
// past the last point is refused (not only the first one beyond), a refusal
// returns nothing, a dirty scratch slice is cleared, and one with room is reused
// rather than reallocated.
//
// Requirement: FR-EXP-005
// Guard: a stale coordinate left in a reused slice would put a point on the wrong level.
func TestGrid_GuardsAndScratch(t *testing.T) {
	t.Parallel()
	if n, ok := design.GridCount([]int{1, 3}); !ok || n != 3 {
		t.Errorf("GridCount(1 3) = %d, %v, want 3: one level is a factor held fixed", n, ok)
	}
	if n, ok := design.GridCount([]int{0}); ok || n != 0 {
		t.Errorf("GridCount(0) = %d, %v, want 0, false", n, ok)
	}
	if n, ok := design.OATCount([]int{1, 1}); !ok || n != 1 {
		t.Errorf("OATCount(1 1) = %d, %v, want 1", n, ok)
	}
	if n, ok := design.OATCount([]int{math.MaxInt}); !ok || n != math.MaxInt {
		t.Errorf("OATCount(MaxInt) = %d, %v, want MaxInt: 1 + (MaxInt - 1) fits", n, ok)
	}
	if n, ok := design.OATCount([]int{math.MaxInt, 2}); ok || n != 0 {
		t.Errorf("OATCount(MaxInt 2) = %d, %v, want 0, false: one past MaxInt", n, ok)
	}
	for _, i := range []int{12, 13, 100} {
		if got, ok := design.GridCoords(i, []int{4, 3}, nil); ok || got != nil {
			t.Errorf("GridCoords(%d, 4 3) = %v, %v, want nil, false", i, got, ok)
		}
	}
	for _, i := range []int{6, 7, 100} {
		if got, ok := design.OATCoords(i, []int{4, 3}, nil); ok || got != nil {
			t.Errorf("OATCoords(%d, 4 3) = %v, %v, want nil, false", i, got, ok)
		}
	}
	dirty := []int{9, 9, 9}
	if got, _ := design.OATCoords(0, []int{2, 2}, dirty); !slices.Equal(got, []int{0, 0}) {
		t.Errorf("OATCoords(0, 2 2, dirty) = %v, want [0 0]: the baseline clears the scratch", got)
	}
	if got, _ := design.OATCoords(3, []int{3, 2}, dirty); !slices.Equal(got, []int{0, 1}) {
		t.Errorf("OATCoords(3, 3 2, dirty) = %v, want [0 1]: a stale 9 must not survive", got)
	}
	room := make([]int, 0, 2)
	got, _ := design.GridCoords(3, []int{2, 2}, room)
	if &got[0] != &room[:1][0] {
		t.Errorf("GridCoords did not reuse a scratch slice with exactly enough capacity")
	}
	roomy := make([]int, 0, 5)
	got, _ = design.GridCoords(3, []int{2, 2}, roomy)
	if &got[0] != &roomy[:1][0] {
		t.Errorf("GridCoords did not reuse a scratch slice with spare capacity")
	}
	small := make([]int, 0, 1)
	got, _ = design.GridCoords(3, []int{2, 2}, small)
	if len(got) != 2 || !slices.Equal(got, []int{1, 1}) {
		t.Errorf("GridCoords(3, 2 2, small scratch) = %v, want [1 1]", got)
	}
	for _, bad := range []struct{ k, steps int }{{4, 4}, {5, 4}, {0, 1}, {0, 0}} {
		if got, ok := design.RangeValue(bad.k, bad.steps, 0, 9); ok || got != 0 {
			t.Errorf("RangeValue(%d, %d, 0, 9) = %d, %v, want 0, false", bad.k, bad.steps, got, ok)
		}
	}
}
