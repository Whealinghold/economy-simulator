package design_test

import (
	"math"
	"math/big"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/design"
)

func cell(t testing.TB, salt uint64, axis, n, i int) int {
	t.Helper()
	c, ok := design.SpreadCell(salt, axis, n, i)
	if !ok {
		t.Fatalf("SpreadCell(%#x, %d, %d, %d) = false", salt, axis, n, i)
	}
	return c
}

// TestSpread_IsAPermutation verifies that on every axis each of the N cells is
// used by exactly one point: every N from 1 to 300 and 511, 512, 513, 1 000,
// 4 096 and 5 000, five axes, three salts. The sizes straddle the powers of four
// where the Feistel width changes.
//
// Requirement: FR-EXP-007
func TestSpread_IsAPermutation(t *testing.T) {
	t.Parallel()
	ns := []int{511, 512, 513, 1000, 4096, 5000}
	for n := 1; n <= 300; n++ {
		ns = append(ns, n)
	}
	for _, salt := range []uint64{0, 1, 0xFFFFFFFFFFFFFFFF} {
		for _, n := range ns {
			for axis := range 5 {
				seen := make([]bool, n)
				for i := range n {
					c := cell(t, salt, axis, n, i)
					if c < 0 || c >= n || seen[c] {
						t.Fatalf("salt %#x axis %d N %d: point %d has cell %d, out of range or used twice", salt, axis, n, i, c)
					}
					seen[c] = true
				}
			}
		}
	}
}

// TestSpread_WorkedPoints verifies the eight points of worked.out section C (two
// range factors, cost_floor in [6 000, 10 000] and belief_step in [100, 400],
// N = 8, salt 0): their cells and values, and the cells of lhs.out (N = 12, the
// first four points on four axes, and axis 3 for all twelve).
//
// Requirement: FR-EXP-008
func TestSpread_WorkedPoints(t *testing.T) {
	t.Parallel()
	// worked.out C.
	worked := []struct {
		cells      [2]int
		cost, step int64
	}{
		{[2]int{4, 1}, 8250, 156}, {[2]int{7, 2}, 9750, 194}, {[2]int{0, 5}, 6250, 306}, {[2]int{2, 3}, 7250, 231},
		{[2]int{1, 0}, 6750, 118}, {[2]int{5, 4}, 8750, 269}, {[2]int{3, 6}, 7750, 344}, {[2]int{6, 7}, 9250, 382},
	}
	for i, w := range worked {
		for axis, lo, hi, want := 0, int64(6000), int64(10000), w.cost; axis < 2; axis, lo, hi, want = axis+1, 100, 400, w.step {
			c := cell(t, 0, axis, 8, i)
			if c != w.cells[axis] {
				t.Errorf("point %d axis %d: cell = %d, want %d", i, axis, c, w.cells[axis])
			}
			if got, ok := design.SpreadValue(c, 8, lo, hi); !ok || got != want {
				t.Errorf("point %d axis %d: SpreadValue(%d, 8, %d, %d) = %d, %v, want %d", i, axis, c, lo, hi, got, ok, want)
			}
		}
	}
	// lhs.out: points 0..3 of N = 12 on four axes, salt 0.
	for i, want := range [][4]int{{10, 6, 0, 1}, {8, 4, 3, 9}, {6, 7, 10, 6}, {5, 0, 11, 7}} {
		for axis, w := range want {
			if got := cell(t, 0, axis, 12, i); got != w {
				t.Errorf("N 12 point %d axis %d: cell = %d, want %d (lhs.out)", i, axis, got, w)
			}
		}
	}
	// lhs.out: axis 3 by point for N = 12.
	var axis3 []int
	for i := range 12 {
		axis3 = append(axis3, cell(t, 0, 3, 12, i))
	}
	if want := []int{1, 9, 6, 7, 5, 4, 0, 3, 10, 11, 2, 8}; !slices.Equal(axis3, want) {
		t.Errorf("N 12 axis 3 cells = %v, want %v (lhs.out)", axis3, want)
	}
}

// TestSpread_ValueRules verifies the value of a cell: a range narrower than N
// repeats values (lhs.out: [10, 19] at N = 25 gives 2 or 3 points a value), an
// explicit factor uses floor((2*cell+1)*L/(2*N)), and Max - Min + 1 near 2^64
// does not wrap (checked against big.Int).
//
// Requirement: FR-EXP-008
func TestSpread_ValueRules(t *testing.T) {
	t.Parallel()
	count := func(n int) map[int64]int {
		m := map[int64]int{}
		for i := range n {
			v, _ := design.SpreadValue(cell(t, 0, 0, n, i), n, 10, 19)
			m[v]++
		}
		return m
	}
	for v, c := range count(10) {
		if c != 1 {
			t.Errorf("N 10 over [10, 19]: value %d used %d times, want once", v, c)
		}
	}
	// lhs.out: N = 25 gives {10: 2, 11: 3, 12: 2, 13: 3, ...}.
	got25 := count(25)
	for v := int64(10); v <= 19; v++ {
		if want := 2 + int((v-10)%2); got25[v] != want {
			t.Errorf("N 25 over [10, 19]: value %d used %d times, want %d", v, got25[v], want)
		}
	}
	// explicit levels: N = 8, L = 3: cells 0..7 -> floor((2c+1)*3/16) = 0,0,0,1,1,2,2,2 (21/16 = 1, 33/16 = 2).
	for c, want := range []int{0, 0, 0, 1, 1, 2, 2, 2} {
		if got, ok := design.SpreadLevel(c, 8, 3); !ok || got != want {
			t.Errorf("SpreadLevel(%d, 8, 3) = %d, %v, want %d", c, got, ok, want)
		}
	}
	for n := 1; n <= 40; n++ { // never past the last level
		for c := range n {
			if got, _ := design.SpreadLevel(c, n, 7); got < 0 || got > 6 {
				t.Errorf("SpreadLevel(%d, %d, 7) = %d, want within [0, 6]", c, n, got)
			}
		}
	}
	// The whole int64 range: width + 1 = 2^64.
	lo, hi := int64(math.MinInt64), int64(math.MaxInt64)
	for _, nc := range [][2]int{{1, 0}, {2, 0}, {2, 1}, {7, 3}, {100000, 99999}, {100000, 0}} {
		n, c := nc[0], nc[1]
		num := new(big.Int).Mul(big.NewInt(int64(2*c+1)), new(big.Int).Lsh(big.NewInt(1), 64))
		want := new(big.Int).Add(new(big.Int).Div(num, big.NewInt(int64(2*n))), big.NewInt(lo)).Int64()
		if got, ok := design.SpreadValue(c, n, lo, hi); !ok || got != want {
			t.Errorf("SpreadValue(%d, %d, MinInt64, MaxInt64) = %d, %v, want %d", c, n, got, ok, want)
		}
	}
	if got, _ := design.SpreadValue(7, 8, 5, 5); got != 5 {
		t.Errorf("SpreadValue over a single value = %d, want 5", got)
	}
}

// TestSpread_Stateless verifies that point i computed alone equals point i in the
// full design, whatever order the points are computed in, that another salt gives
// another batch, and that out-of-range arguments report false.
//
// Requirement: FR-EXP-009
// Guard: a design that depended on the points computed before it would give a
// resumed, skipped or re-ordered sweep different points (spec-experiments.md 4.5.3).
func TestSpread_Stateless(t *testing.T) {
	t.Parallel()
	const n = 37
	full := make([]int, n)
	for i := range n {
		full[i] = cell(t, 5, 2, n, i)
	}
	for _, i := range []int{36, 0, 18, 5, 36} { // any order, a repeat
		if got := cell(t, 5, 2, n, i); got != full[i] {
			t.Errorf("point %d alone = %d, in the full design = %d", i, got, full[i])
		}
	}
	other := make([]int, n)
	for i := range n {
		other[i] = cell(t, 6, 2, n, i)
	}
	if slices.Equal(full, other) {
		t.Errorf("salts 5 and 6 give the same batch %v, want different batches", full)
	}
	for _, c := range [][4]int{{0, 0, 0, 0}, {0, -1, 5, 0}, {0, 0, 5, 5}, {0, 0, 5, -1}} {
		if got, ok := design.SpreadCell(uint64(c[0]), c[1], c[2], c[3]); ok {
			t.Errorf("SpreadCell%v = %d, true, want false", c, got)
		}
	}
	if _, ok := design.SpreadCell(0, 0, design.MaxSpread+1, 0); ok {
		t.Errorf("SpreadCell with N above MaxSpread ok = true, want false")
	}
	if _, ok := design.SpreadLevel(0, 0, 3); ok {
		t.Errorf("SpreadLevel with N 0 ok = true, want false")
	}
	if _, ok := design.SpreadLevel(0, 4, 0); ok {
		t.Errorf("SpreadLevel with 0 levels ok = true, want false")
	}
	if _, ok := design.SpreadValue(0, 4, 5, 4); ok {
		t.Errorf("SpreadValue with Min above Max ok = true, want false")
	}
	if _, ok := design.SpreadValue(4, 4, 0, 4); ok {
		t.Errorf("SpreadValue with the cell outside N ok = true, want false")
	}
}

// FuzzPermutation checks, for any salt, axis and N up to 200, that every cell is
// in range and the cells are a bijection of [0, N); for a larger N only that
// every point is in range.
//
// Requirement: NFR-EXP-005
func FuzzPermutation(f *testing.F) {
	f.Add(uint64(0), 0, 12)
	f.Add(uint64(1<<63), 7, 200)
	f.Add(uint64(99), 3, 4097)
	f.Fuzz(func(t *testing.T, salt uint64, axis, n int) {
		if axis < 0 || axis > 1000 || n < 1 || n > 5000 {
			return
		}
		seen := map[int]bool{}
		for i := range n {
			c, ok := design.SpreadCell(salt, axis, n, i)
			if !ok || c < 0 || c >= n {
				t.Fatalf("SpreadCell(%#x, %d, %d, %d) = %d, %v: out of range", salt, axis, n, i, c, ok)
			}
			if n <= 200 {
				if seen[c] {
					t.Fatalf("SpreadCell(%#x, %d, %d, _) used cell %d twice", salt, axis, n, c)
				}
				seen[c] = true
			}
		}
	})
}

// TestSpread_SmallSizesAgainstTheModel verifies the cells of small N, where the
// Feistel half-width w changes (w = 1 up to N = 4, 2 up to 16, 3 up to 64),
// against doc/specs/models/exp/lhs.py perm() with the default design salt on
// axes 0 and 1. Sizes 4 and 16 sit exactly on a width boundary, 5 and 17 just
// past it.
//
// Requirement: FR-EXP-007, FR-EXP-008
func TestSpread_SmallSizesAgainstTheModel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		n    int
		axes [2][]int
	}{
		{2, [2][]int{{0, 1}, {1, 0}}},
		{3, [2][]int{{2, 1, 0}, {2, 1, 0}}},
		{4, [2][]int{{1, 0, 3, 2}, {0, 1, 3, 2}}},
		{5, [2][]int{{2, 4, 1, 3, 0}, {4, 3, 0, 1, 2}}},
		{9, [2][]int{{2, 3, 5, 0, 8, 6, 1, 7, 4}, {0, 7, 2, 1, 4, 3, 6, 8, 5}}},
		{16, [2][]int{{0, 10, 4, 11, 15, 6, 5, 7, 8, 12, 1, 3, 9, 13, 14, 2}, {6, 10, 15, 0, 3, 12, 9, 8, 13, 2, 4, 5, 11, 7, 14, 1}}},
		{17, [2][]int{{5, 7, 0, 15, 11, 6, 9, 3, 4, 8, 2, 14, 13, 10, 16, 12, 1}, {4, 14, 7, 5, 1, 0, 15, 3, 16, 8, 6, 10, 12, 11, 13, 9, 2}}},
	}
	for _, tt := range cases {
		for axis, want := range tt.axes {
			var got []int
			for i := range tt.n {
				got = append(got, cell(t, 0, axis, tt.n, i))
			}
			if !slices.Equal(got, want) {
				t.Errorf("N %d axis %d: cells = %v, want %v (lhs.py)", tt.n, axis, got, want)
			}
		}
	}
	if got := cell(t, 0, 0, 1, 0); got != 0 {
		t.Errorf("N 1: cell = %d, want 0", got)
	}
}

// TestSpread_GuardsAtTheEdges verifies each argument guard of the three value
// functions on its own boundary, and that a refusal returns the zero value; and
// that the largest accepted spread, 2^62, works without wrapping.
//
// Requirement: FR-EXP-007, FR-EXP-008
func TestSpread_GuardsAtTheEdges(t *testing.T) {
	t.Parallel()
	const max = design.MaxSpread
	for _, c := range []struct{ salt, axis, n, i int }{
		{0, 0, 0, 0}, {0, 0, -1, 0}, {0, 0, 5, 6}, {0, 0, 5, 5}, {0, -1, 5, 0}, {0, 0, 5, -1}, {0, 0, max + 1, 0},
	} {
		if got, ok := design.SpreadCell(uint64(c.salt), c.axis, c.n, c.i); ok || got != 0 {
			t.Errorf("SpreadCell%v = %d, %v, want 0, false", c, got, ok)
		}
	}
	for _, c := range []struct{ cell, n, levels int }{
		{0, 0, 3}, {0, -1, 3}, {0, 4, 0}, {0, 4, -1}, {-1, 4, 3}, {4, 4, 3}, {5, 4, 3}, {0, max + 1, 3},
	} {
		if got, ok := design.SpreadLevel(c.cell, c.n, c.levels); ok || got != 0 {
			t.Errorf("SpreadLevel%v = %d, %v, want 0, false", c, got, ok)
		}
	}
	for _, c := range []struct {
		cell, n int
		lo, hi  int64
	}{
		{0, 0, 0, 5}, {0, -1, 0, 5}, {-1, 4, 0, 5}, {4, 4, 0, 5}, {5, 4, 0, 5}, {0, 4, 6, 5}, {0, max + 1, 0, 5},
	} {
		if got, ok := design.SpreadValue(c.cell, c.n, c.lo, c.hi); ok || got != 0 {
			t.Errorf("SpreadValue%v = %d, %v, want 0, false", c, got, ok)
		}
	}
	// The edges that are valid: one level, one point, the last cell, the largest spread.
	if got, ok := design.SpreadLevel(0, 1, 1); !ok || got != 0 {
		t.Errorf("SpreadLevel(0, 1, 1) = %d, %v, want 0, true", got, ok)
	}
	if got, ok := design.SpreadLevel(max-1, max, 10); !ok || got != 9 {
		t.Errorf("SpreadLevel(last, MaxSpread, 10) = %d, %v, want 9, true", got, ok)
	}
	if got, ok := design.SpreadValue(max-1, max, 0, 1_000_000); !ok || got != 1_000_000 {
		t.Errorf("SpreadValue(last, MaxSpread, 0, 10^6) = %d, %v, want 10^6, true", got, ok)
	}
	if got, ok := design.SpreadCell(3, 1, max, max-1); !ok || got < 0 || got >= max {
		t.Errorf("SpreadCell(…, MaxSpread, last) = %d, %v, want a cell in range", got, ok)
	}
}
