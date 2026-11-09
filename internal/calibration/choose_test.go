package calibration

import (
	"math"
	"reflect"
	"slices"
	"testing"
)

// table193 is the seven points of the worked choice of 19.3 at R = 10 (need 9 of
// 10 on a B criterion): levels are (CostFloor, EvidenceShare) grid indices
// relative to the base, and BPass is (P02 passes, P04 passes).
func table193() []Point {
	return []Point{
		{0, []int{0, 0}, true, []int{7, 10}},
		{1, []int{-1, 0}, true, []int{9, 10}},
		{2, []int{1, 0}, true, []int{4, 10}},
		{3, []int{0, -1}, true, []int{8, 10}},
		{4, []int{0, 1}, true, []int{6, 9}},
		{5, []int{-1, 1}, true, []int{10, 10}},
		{6, []int{-1, -1}, true, []int{10, 10}},
	}
}

// TestChoose_Worked verifies the choice of 19.3: the base fails P02 (7 of 10, below
// 9), points 1, 5 and 6 are feasible and point 1, at grid distance 1, is chosen
// although 5 and 6 pass more seeds.
//
// Requirement: FR-CLB-030, FR-CLB-032
func TestChoose_Worked(t *testing.T) {
	t.Parallel()
	got := Choose(table193(), 0, 10)
	want := Choice{Index: 1, Distance: 1, Feasible: []int{1, 5, 6}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Choose(19.3, base 0, R 10) = %+v, want %+v", got, want)
	}
}

// TestChoose_BaseWinsWhenFeasible verifies that a feasible base is the choice
// even when another feasible point passes more seeds, and that the base is found
// by its Index and not by its position.
//
// Requirement: FR-CLB-031
func TestChoose_BaseWinsWhenFeasible(t *testing.T) {
	t.Parallel()
	pts := []Point{
		{5, []int{1}, true, []int{10}},
		{9, []int{0}, true, []int{9}}, // the base: exactly the threshold
		{2, []int{-1}, true, []int{10}},
	}
	got := Choose(pts, 9, 10)
	if got.None || got.Index != 9 || got.Distance != 0 {
		t.Errorf("Choose(base 9 feasible) = %+v, want Index 9 at distance 0", got)
	}
	// The same points with base 7, which is no point: the least distance among the
	// feasible, 0, is still point 9 by distance; with base 5 (distance 1) it is point 9 too.
	if got := Choose(pts, 5, 10); got.Index != 5 {
		t.Errorf("Choose(base 5 feasible at distance 1) = %+v, want the base 5 to win over the closer point 9", got)
	}
	if got := Choose(pts, 7, 10); got.Index != 9 {
		t.Errorf("Choose(base 7, not a point) = %+v, want the least distance, point 9", got)
	}
}

// TestChoose_LeastDistance verifies the order of 7.2.3 item 3: least grid
// distance, then fewest knobs changed, then lowest index.
//
// Requirement: FR-CLB-032
func TestChoose_LeastDistance(t *testing.T) {
	t.Parallel()
	pass := func(i int, levels ...int) Point { return Point{i, levels, true, nil} }
	tests := []struct {
		name string
		pts  []Point
		want int
	}{
		{"distance first", []Point{pass(1, 2), pass(2, 1, 1, 1)}, 1 + 0*0},           // 2 against 3
		{"fewest knobs on a distance tie", []Point{pass(1, 1, 1), pass(2, 2, 0)}, 2}, // both 2; 2 knobs against 1
		{"lowest index on a full tie", []Point{pass(7, 1, 0), pass(3, 0, -1), pass(5, 0, 1)}, 3},
		{"negative levels count by size", []Point{pass(1, -3), pass(2, 2)}, 2},
		{"base is not a point", []Point{pass(4, 0, 0), pass(1, 1, 0)}, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Choose(tt.pts, -1, 10); got.None || got.Index != tt.want {
				t.Errorf("Choose(%v) = %+v, want index %d", tt.pts, got, tt.want)
			}
		})
	}
}

// TestChoose_NoneAndPareto verifies that no feasible point gives no choice and the
// Pareto set of the B count vectors among the exact-passing points: the case of
// 19.3's last paragraph, where (10, 5) and (7, 9) are non-dominated and (6, 6)
// and (9, 4) are not.
//
// Requirement: FR-CLB-033
func TestChoose_NoneAndPareto(t *testing.T) {
	t.Parallel()
	pts := []Point{
		{0, []int{0}, true, []int{10, 5}},
		{1, []int{1}, true, []int{7, 9}},
		{2, []int{2}, true, []int{6, 6}},    // dominated by (7, 9)
		{3, []int{-1}, true, []int{9, 4}},   // dominated by (10, 5)
		{4, []int{3}, false, []int{10, 10}}, // fails an exact criterion: out of the set
	}
	got := Choose(pts, 0, 10)
	want := Choice{None: true, Pareto: []int{0, 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Choose(no feasible point) = %+v, want %+v", got, want)
	}
	// Equal vectors do not dominate each other: both stay.
	same := []Point{{0, nil, true, []int{3}}, {1, nil, true, []int{3}}}
	if got := Choose(same, 0, 10); !got.None || !slices.Equal(got.Pareto, []int{0, 1}) {
		t.Errorf("Choose(equal vectors) = %+v, want None with Pareto [0 1]", got)
	}
	// No point passes the exact criteria: nothing to put in the set.
	if got := Choose([]Point{{0, nil, false, []int{10}}}, 0, 10); !got.None || len(got.Pareto) != 0 {
		t.Errorf("Choose(no exact pass) = %+v, want None and an empty Pareto set", got)
	}
	if got := Choose(nil, 0, 10); !got.None || len(got.Feasible) != 0 || len(got.Pareto) != 0 {
		t.Errorf("Choose(nil) = %+v, want None and nothing else", got)
	}
}

// TestChoose_Pure verifies that the same input gives the same output and that the
// input is left as it was.
//
// Requirement: FR-CLB-034
func TestChoose_Pure(t *testing.T) {
	t.Parallel()
	pts := table193()
	before := table193()
	a, b := Choose(pts, 0, 10), Choose(pts, 0, 10)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Choose twice gave %+v then %+v", a, b)
	}
	if !reflect.DeepEqual(pts, before) {
		t.Errorf("Choose changed its input: %+v, want %+v", pts, before)
	}
}

// TestChoose_SeedThreshold verifies that the B threshold is R - R/10 in exact
// integers: 23 of 25 (half to even would give 22), 9 of 10, 45 of 50.
//
// Requirement: FR-CLB-106
func TestChoose_SeedThreshold(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ r, need int }{{25, 23}, {10, 9}, {50, 45}, {1, 1}, {0, 0}} {
		at := []Point{{0, []int{0}, true, []int{tt.need}}}
		if got := Choose(at, 0, tt.r); got.None {
			t.Errorf("Choose(count %d of %d) = %+v, want feasible", tt.need, tt.r, got)
		}
		if tt.need > 0 {
			under := []Point{{0, []int{0}, true, []int{tt.need - 1}}}
			if got := Choose(under, 0, tt.r); !got.None {
				t.Errorf("Choose(count %d of %d) = %+v, want no choice", tt.need-1, tt.r, got)
			}
		}
	}
}

// TestChoose_MissingCounts verifies that a point with fewer B counts than another
// has 0 for the criteria it lacks, so it cannot be feasible by omission.
//
// Requirement: FR-CLB-030
func TestChoose_MissingCounts(t *testing.T) {
	t.Parallel()
	pts := []Point{
		{0, []int{0}, true, []int{10}},
		{1, []int{1}, true, []int{10, 10}},
	}
	got := Choose(pts, 0, 10)
	if got.None || got.Index != 1 || !slices.Equal(got.Feasible, []int{1}) {
		t.Errorf("Choose(ragged counts) = %+v, want only point 1 feasible", got)
	}
}

// TestChoose_DistanceSaturates verifies that an absurd level index cannot wrap the
// distance into a small number, MinInt included.
//
// Guard: the distance orders the feasible points; a wrapped sum would let a far
// point win.
func TestChoose_DistanceSaturates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		levels []int
		want   int64
	}{
		{[]int{math.MinInt}, math.MaxInt64},
		{[]int{math.MaxInt, math.MaxInt}, math.MaxInt64},
		{[]int{math.MaxInt64 / 2, math.MaxInt64 / 2}, math.MaxInt64 - 1},
		{[]int{-2, 3, 0}, 5},
		{nil, 0},
	}
	for _, tt := range tests {
		if got := gridDistance(tt.levels); got != tt.want {
			t.Errorf("gridDistance(%v) = %d, want %d", tt.levels, got, tt.want)
		}
	}
	pts := []Point{{1, []int{math.MinInt}, true, nil}, {2, []int{3}, true, nil}}
	if got := Choose(pts, -1, 10); got.Index != 2 {
		t.Errorf("Choose(far and near) = %+v, want the near point 2", got)
	}
}

// TestSpreadLevel_TieLower verifies that a Spread value maps to the nearest
// level and a tie goes to the lower one, not to the even one.
//
// Requirement: FR-CLB-013
func TestSpreadLevel_TieLower(t *testing.T) {
	t.Parallel()
	g := Grid{Levels: []int64{100, 200, 400}}
	tests := []struct {
		x    int64
		want int
	}{
		{300, 1}, // 100 from 200 and 100 from 400: the lower, 200
		{290, 1},
		{310, 2},
		{150, 0}, // 50 from 100 and 50 from 200: the lower, 100
		{200, 1},
		{-50, 0},
		{9000, 2},
		{math.MinInt64, 0},
		{math.MaxInt64, 2},
	}
	for _, tt := range tests {
		if got := SpreadLevel(g, tt.x); got != tt.want {
			t.Errorf("SpreadLevel(%v, %d) = %d, want %d", g.Levels, tt.x, got, tt.want)
		}
	}
	// A tie is the lower level whichever order the levels are listed in (a set grid).
	if got := SpreadLevel(Grid{Levels: []int64{400, 200}}, 300); got != 1 {
		t.Errorf("SpreadLevel([400 200], 300) = %d, want index 1 (level 200)", got)
	}
	if got := SpreadLevel(Grid{}, 5); got != -1 {
		t.Errorf("SpreadLevel(empty, 5) = %d, want -1", got)
	}
	// Distances between extreme values do not wrap: MinInt64 is 2^64-1 from MaxInt64.
	if got := SpreadLevel(Grid{Levels: []int64{math.MaxInt64, 0}}, math.MinInt64); got != 1 {
		t.Errorf("SpreadLevel([max 0], min) = %d, want index 1 (level 0)", got)
	}
}

// FuzzChoose verifies, for any set of points, that Choose does not panic, that a
// chosen point is feasible, that a feasible base is the choice, that no choice
// means no feasible point, and that the Pareto set holds only exact-passing
// points that no other such point dominates.
//
// #lizard forgives: one fuzz body, one property per block.
func FuzzChoose(f *testing.F) {
	f.Add([]byte{1, 0, 9, 10, 1, 1, 9, 10, 1, 2, 9, 9}, uint8(0), uint8(10))
	f.Add([]byte{1, 0, 5, 1, 1, 7, 9, 0, 3, 10}, uint8(1), uint8(10))
	f.Add([]byte{}, uint8(0), uint8(0))
	f.Fuzz(func(t *testing.T, data []byte, base, r uint8) {
		var pts []Point
		for len(data) >= 4 {
			pts = append(pts, Point{Index: len(pts), Levels: []int{int(int8(data[1]))}, ExactPass: data[0]&1 == 1, BPass: []int{int(data[2] % 51), int(data[3] % 51)}})
			data = data[4:]
		}
		c := Choose(pts, int(base), int(r%60))
		if c.None != (len(c.Feasible) == 0) {
			t.Fatalf("Choose(%v) = %+v: None and Feasible disagree", pts, c)
		}
		if !c.None {
			if !slices.Contains(c.Feasible, c.Index) {
				t.Fatalf("Choose(%v) = %+v: the choice is not feasible", pts, c)
			}
			if slices.Contains(c.Feasible, int(base)) && c.Index != int(base) {
				t.Fatalf("Choose(%v, base %d) = %+v: a feasible base did not win", pts, base, c)
			}
			return
		}
		for _, i := range c.Pareto {
			if !pts[i].ExactPass {
				t.Fatalf("Choose(%v) Pareto point %d fails an exact criterion", pts, i)
			}
			for _, q := range pts {
				if q.ExactPass && dominates(q, pts[i], 2) {
					t.Fatalf("Choose(%v) Pareto point %d is dominated by %d", pts, i, q.Index)
				}
			}
		}
	})
}
