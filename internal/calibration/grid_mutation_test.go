package calibration

import (
	"errors"
	"math"
	"slices"
	"strings"
	"testing"
)

// The tests of this file pin the boundaries and branches that the mutation pass
// of Phase 3 found unasserted (gomu, Docker, 4 CPUs): each is a case where a
// one-token change of the code gave the same result on every earlier test.

// TestErrors_Text verifies that every sentinel of the package has a non-empty
// message that names the package, so an error a caller prints says where it
// came from.
//
// Guard: the sentinels are constants of an error type; an emptied message would
// still match errors.Is and no other test would see it.
func TestErrors_Text(t *testing.T) {
	t.Parallel()
	for _, err := range []error{ErrNoGrid, ErrBase, ErrOverflow, ErrCount, ErrBlock} {
		if !strings.HasPrefix(err.Error(), "calibration: ") || len(err.Error()) < 20 {
			t.Errorf("%T(%q).Error() is not a message of the package", err, string(err.(calError)))
		}
	}
}

// TestGrid_LevelCap verifies the cap on the levels of one grid at its edge: a
// grid of exactly maxLevels levels is built, one more is ErrBase, whether the
// levels are below the base, above it, or split between the two.
//
// Requirement: FR-CLB-013
func TestGrid_LevelCap(t *testing.T) {
	t.Parallel()
	last := int64(maxLevels - 1)
	tests := []struct {
		name string
		base int64
		b    Bounds
		ok   bool
	}{
		{"all above, the cap exactly", 0, bnds(0, last), true},
		{"all above, one over", 0, bnds(0, last+1), false},
		{"all below, the cap exactly", last, bnds(0, last), true},
		{"all below, one over", last + 1, bnds(0, last+1), false},
		// 32 768 levels below the base, 32 767 above and the base itself: 65 536.
		{"split, the cap exactly", 32768, bnds(0, last), true},
		{"split, one over", 32768, bnds(0, last+1), false},
	}
	for _, tt := range tests {
		g, err := Build(lin(1), tt.base, tt.b, noOff(), nil)
		if (err == nil) != tt.ok || (err != nil && !errors.Is(err, ErrBase)) || (tt.ok && len(g.Levels) != maxLevels) {
			t.Errorf("Build(lin1, %d, %v) = %d levels, %v, want ok %v and %d levels", tt.base, tt.b, len(g.Levels), err, tt.ok, maxLevels)
		}
	}
	listed := make([]int64, maxLevels)
	for i := range listed {
		listed[i] = int64(i)
	}
	if g, err := Build(Rule{Kind: Set, Set: listed}, 0, bnds(0, maxLevels), noOff(), nil); err != nil || len(g.Levels) != maxLevels {
		t.Errorf("Build(set of %d) = %d levels, %v, want all of them", maxLevels, len(g.Levels), err)
	}
}

// TestGrid_BoundsBoundaries verifies that a base on either bound, and a single
// level where the bounds are equal, are accepted, and that a base outside the
// bounds, or bounds in the wrong order, are ErrBase for every kind of rule.
//
// Requirement: FR-CLB-014
func TestGrid_BoundsBoundaries(t *testing.T) {
	t.Parallel()
	for _, r := range []Rule{geo(), lin(10), {Kind: Set, Set: []int64{20, 30, 40}}} {
		for _, base := range []int64{20, 40} { // each bound
			if g, err := Build(r, base, bnds(20, 40), noOff(), nil); err != nil || g.Levels[g.Base] != base {
				t.Errorf("Build(%v, %d, 20..40) = %v, %v, want a grid on the bound", r, base, g.Levels, err)
			}
		}
		if g, err := Build(r, 20, bnds(20, 20), noOff(), nil); err != nil || !slices.Equal(g.Levels, []int64{20}) {
			t.Errorf("Build(%v, 20, 20..20) = %v, %v, want the one level 20", r, g.Levels, err)
		}
		if got := outsideBounds(r); len(got) > 0 {
			t.Errorf("Build(%v) outside its bounds did not return ErrBase for %v", r, got)
		}
	}
	// The single level has both neighbours missing, both bounds.
	g, _ := Build(lin(10), 20, bnds(20, 20), noOff(), nil)
	if lo, up := edges(g); lo != Bound || up != Bound {
		t.Errorf("Build(lin10, 20, 20..20) edges = %v, %v, want both Bound", lo, up)
	}
}

// outsideBounds returns the cases of a base below, above or bounds in the wrong
// order for which Build did not return ErrBase.
func outsideBounds(r Rule) []string {
	var missed []string
	for _, bad := range []struct {
		name string
		base int64
		b    Bounds
	}{{"below", 19, bnds(20, 40)}, {"above", 41, bnds(20, 40)}, {"unordered", 30, bnds(40, 20)}} {
		if _, err := Build(r, bad.base, bad.b, noOff(), nil); !errors.Is(err, ErrBase) {
			missed = append(missed, bad.name)
		}
	}
	return missed
}

// TestGrid_GeoBoundaries verifies the geometric grid at its limits: a level equal
// to the upper bound is kept, the floor is repeated by the halvings without being
// listed twice, and a lower bound above 1 is the floor.
//
// Requirement: FR-CLB-010
func TestGrid_GeoBoundaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		base int64
		b    Bounds
		want []int64
	}{
		// 4 doubled is 8, which is the upper bound: kept.
		{"upper bound is a level", 4, bnds(1, 8), []int64{1, 2, 4, 8}},
		// Halvings of 5: 2.5 -> 2, 1.25 -> 1, 0.625 -> 1 (not a tie), 0.31 -> 0: the 1 repeats.
		{"the floor repeats", 5, bnds(1, 100), []int64{1, 2, 5, 10, 20, 40, 80}},
		// Lower bound 3: halvings of 20 are 10, 5, 2.5 -> 2 (below 3), so 5 is the last.
		{"lower bound above 1", 20, bnds(3, 100), []int64{5, 10, 20, 40, 80}},
	}
	for _, tt := range tests {
		g, err := Build(geo(), tt.base, tt.b, noOff(), nil)
		if err != nil || !slices.Equal(g.Levels, tt.want) {
			t.Errorf("Build(geo, %d, %v) = %v, %v, want %v", tt.base, tt.b, g.Levels, err, tt.want)
		}
	}
}

// TestGrid_HalveRounding verifies rdiv(v, 2^k), half to even, at every rounding case.
//
// Requirement: FR-CLB-013
func TestGrid_HalveRounding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		v    int64
		k    uint
		want int64
	}{
		{7000, 1, 3500}, // exact
		{7000, 4, 438},  // 437.5, a tie: the even neighbour
		{3, 1, 2},       // 1.5, a tie: 2 is even
		{5, 1, 2},       // 2.5, a tie: 2 is even
		{5, 3, 1},       // 0.625, nearer 1
		{1 << 62, 62, 1},
		{math.MaxInt64, 62, 2}, // 1.99999..., nearer 2
		{5, 63, 0},
		{math.MaxInt64, 64, 0},
	}
	for _, tt := range tests {
		if got := halve(tt.v, tt.k); got != tt.want {
			t.Errorf("halve(%d, %d) = %d, want %d", tt.v, tt.k, got, tt.want)
		}
	}
}

// TestGrid_LinFromLowerBound verifies that the levels below the base stop at the
// lower bound, which is not zero: base 50, step 10 in 20 to 100 has three levels
// below it, (50 - 20) / 10.
//
// Requirement: FR-CLB-011
func TestGrid_LinFromLowerBound(t *testing.T) {
	t.Parallel()
	g, err := Build(lin(10), 50, bnds(20, 100), noOff(), nil)
	if want := []int64{20, 30, 40, 50, 60, 70, 80, 90, 100}; err != nil || !slices.Equal(g.Levels, want) {
		t.Errorf("Build(lin10, 50, 20..100) = %v, %v, want %v", g.Levels, err, want)
	}
}

// TestGrid_SetFilter verifies that a set keeps the listed values inside the
// bounds, bounds included, in the order given, with a repeated value once.
//
// Requirement: FR-CLB-011
func TestGrid_SetFilter(t *testing.T) {
	t.Parallel()
	g, err := Build(Rule{Kind: Set, Set: []int64{0, 5, 5, 11, 3, 10}}, 3, bnds(0, 10), noOff(), nil)
	if want := []int64{0, 5, 3, 10}; err != nil || !slices.Equal(g.Levels, want) {
		t.Errorf("Build(set with 11 and a repeat) = %v, %v, want %v", g.Levels, err, want)
	}
}

// TestGrid_CutPositions verifies the cut where the off value lies between levels
// or cuts several: below the base all the levels at or under it go, between two
// levels only those beyond it go, and a set is cut by position even when its
// order is not ascending.
//
// Requirement: FR-CLB-108
func TestGrid_CutPositions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		rule Rule
		base int64
		b    Bounds
		off  Off
		want []int64
	}{
		{"several levels below", lin(1000), 4000, bnds(0, 6000), offAt(2000), []int64{3000, 4000, 5000, 6000}},
		// 100 is no level: 0 is beyond it (at or under), 250 is not.
		{"off between two levels below", lin(250), 1000, bnds(0, 2000), offAt(100), []int64{250, 500, 750, 1000, 1250, 1500, 1750, 2000}},
		// 6500 is no level: 7000 and 8000 are beyond it, 6000 is not.
		{"off between two levels above", lin(1000), 4000, bnds(0, 8000), offAt(6500), []int64{0, 1000, 2000, 3000, 4000, 5000, 6000}},
		// Unsorted set: by value 3 is below the base 9, by position it is after it.
		{"set by position", Rule{Kind: Set, Set: []int64{9, 3, 11, 5}}, 9, bnds(0, 20), offAt(3), []int64{9}},
		{"set off at position 0", Rule{Kind: Set, Set: []int64{3, 5, 7}}, 5, bnds(0, 20), offAt(3), []int64{5, 7}},
		{"off beyond every level", lin(1000), 4000, bnds(0, 6849), offAt(9000), []int64{0, 1000, 2000, 3000, 4000, 5000, 6000}},
	}
	for _, tt := range tests {
		g, err := Build(tt.rule, tt.base, tt.b, tt.off, nil)
		if err != nil || !slices.Equal(g.Levels, tt.want) {
			t.Errorf("Build(%s) = %v, %v, want %v", tt.name, g.Levels, err, tt.want)
		}
	}
}

// TestGrid_EdgeOfTheLastLevel verifies the edge of a side whose next level is the
// last of the grid: cut (the off value) or invalid (the rule), not a bound.
//
// Requirement: FR-CLB-012
func TestGrid_EdgeOfTheLastLevel(t *testing.T) {
	t.Parallel()
	g, _ := Build(lin(250), 750, bnds(0, 1000), offAt(1000), nil)
	if g.Upper.Present || g.Upper.Edge != OffEdge {
		t.Errorf("upper of 750 with the off value 1000 = %v, want an off edge", g.Upper)
	}
	g, _ = Build(lin(250), 750, bnds(0, 1000), noOff(), func(v int64) bool { return v != 1000 })
	if g.Upper.Present || g.Upper.Edge != RuleEdge {
		t.Errorf("upper of 750 with 1000 invalid = %v, want a rule edge", g.Upper)
	}
	g, _ = Build(lin(250), 250, bnds(0, 1000), noOff(), func(v int64) bool { return v != 0 })
	if g.Lower.Present || g.Lower.Edge != RuleEdge {
		t.Errorf("lower of 250 with 0 invalid = %v, want a rule edge", g.Lower)
	}
}

// TestGrid_DistanceIndices verifies the distance at the first and last index and
// just past the end.
//
// Requirement: FR-CLB-032
func TestGrid_DistanceIndices(t *testing.T) {
	t.Parallel()
	g, _ := Build(lin(10), 50, bnds(0, 100), noOff(), nil) // 11 levels, indices 0 to 10
	for _, tt := range []struct {
		i, j int
		want int64
	}{{0, 0, 0}, {5, 0, 5}, {0, 5, 5}, {10, 10, 0}, {10, 11, -1}, {11, 10, -1}, {12, 10, -1}, {10, 12, -1}, {3, -1, -1}} {
		if got := Distance(g, tt.i, tt.j); got != tt.want {
			t.Errorf("Distance(g, %d, %d) = %d, want %d", tt.i, tt.j, got, tt.want)
		}
	}
}
