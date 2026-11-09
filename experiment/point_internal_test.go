package experiment

import (
	"slices"
	"testing"
)

func lv(n int) []Level { return make([]Level, n) }

// TestPoint_Designs verifies the point of each design: the grid has the last
// factor fastest and the range value of a grid is half to even, one at a time
// lists the baseline and then each factor alone, a spread gives the cell and
// the level or the midpoint value, and a p outside the design is refused.
//
// The grid is 2 levels by a range [0, 10] in 3 steps: points (0,0) (0,5) (0,10)
// (1,0) (1,5) (1,10), the values of the range being 0 + rdiv(k*10, 2).
//
// Requirement: FR-EXP-005, FR-EXP-008, FR-EXP-012
func TestPoint_Designs(t *testing.T) {
	t.Parallel()
	mixed := []Factor{{Levels: lv(2)}, {Path: "a.b", Min: 0, Max: 10, Steps: 3}}
	tests := []struct {
		name       string
		e          Spec
		p          int
		wantLevels []int
		wantValues []int64
		wantN      int
	}{
		{"grid first", Spec{Factors: mixed}, 0, []int{0, 0}, []int64{0, 0}, 6},
		{"grid middle", Spec{Factors: mixed}, 4, []int{1, 1}, []int64{1, 5}, 6},
		{"grid last", Spec{Factors: mixed}, 5, []int{1, 2}, []int64{1, 10}, 6},
		{"oat baseline", Spec{Design: OneAtATime, Factors: mixed}, 0, []int{0, 0}, []int64{0, 0}, 4},
		{"oat first factor", Spec{Design: OneAtATime, Factors: mixed}, 1, []int{1, 0}, []int64{1, 0}, 4},
		{"oat range", Spec{Design: OneAtATime, Factors: mixed}, 3, []int{0, 2}, []int64{0, 10}, 4},
		{"no factors", Spec{Design: Spread, DesignN: 9}, 0, []int{}, []int64{}, 1},
	}
	for _, tt := range tests {
		n, ok := tt.e.pointCount()
		if !ok || n != tt.wantN {
			t.Errorf("%s: pointCount() = %d, %v, want %d", tt.name, n, ok, tt.wantN)
		}
		l, v, ok := tt.e.point(tt.p)
		if !ok || !slices.Equal(l, tt.wantLevels) || !slices.Equal(v, tt.wantValues) {
			t.Errorf("%s: point(%d) = %v, %v, %v, want %v, %v", tt.name, tt.p, l, v, ok, tt.wantLevels, tt.wantValues)
		}
		if _, _, ok := tt.e.point(tt.wantN); ok {
			t.Errorf("%s: point(%d) = ok, want a refusal past the last point", tt.name, tt.wantN)
		}
		if _, _, ok := tt.e.point(-1); ok {
			t.Errorf("%s: point(-1) = ok, want a refusal", tt.name)
		}
	}
}

// TestPoint_SpreadStrataAndValues verifies a spread of 10 over a range of ten
// values uses each value exactly once and over an explicit factor of 5 levels
// each level twice (the Latin hypercube stratification, spec-experiments.md
// 4.4.1), and that the cell is the Levels entry.
//
// Requirement: FR-EXP-007, FR-EXP-008
func TestPoint_SpreadStrataAndValues(t *testing.T) {
	t.Parallel()
	e := Spec{Design: Spread, DesignN: 10, DesignSalt: 3, Factors: []Factor{
		{Path: "a.b", Min: 100, Max: 109},
		{Levels: lv(5)},
	}}
	if n, ok := e.pointCount(); !ok || n != 10 {
		t.Fatalf("pointCount() = %d, %v, want 10", n, ok)
	}
	values := map[int64]int{}
	levels := map[int64]int{}
	cells := map[int]int{}
	for p := range 10 {
		l, v, ok := e.point(p)
		if !ok {
			t.Fatalf("point(%d) refused", p)
		}
		values[v[0]]++
		levels[v[1]]++
		cells[l[0]]++
	}
	for k := int64(100); k < 110; k++ {
		if values[k] != 1 {
			t.Errorf("range value %d used %d times, want once", k, values[k])
		}
	}
	for k := int64(0); k < 5; k++ {
		if levels[k] != 2 {
			t.Errorf("level %d used %d times, want twice", k, levels[k])
		}
	}
	if len(cells) != 10 {
		t.Errorf("cells of axis 0 = %v, want a permutation of 0..9", cells)
	}
}
