package calibration

import (
	"errors"
	"math"
	"slices"
	"testing"
)

func geo() Rule                { return Rule{Kind: Geo} }
func lin(s int64) Rule         { return Rule{Kind: Lin, Step: s} }
func noOff() Off               { return Off{} }
func offAt(v int64) Off        { return Off{Has: true, Value: v} }
func bnds(lo, hi int64) Bounds { return Bounds{Lo: lo, Hi: hi} }

// window returns the levels two each side of the base.
func window(g Grid) []int64 {
	return g.Levels[max(0, g.Base-2):min(len(g.Levels), g.Base+3)]
}

// edges returns the edge of each side, None for a side that has a neighbour.
func edges(g Grid) (lower, upper Edge) {
	if !g.Lower.Present {
		lower = g.Lower.Edge
	}
	if !g.Upper.Present {
		upper = g.Upper.Edge
	}
	return lower, upper
}

// neighbourPair returns the lower and upper neighbour, -1 for a side with none.
func neighbourPair(g Grid) (lower, upper int64) {
	lower, upper = -1, -1
	if g.Lower.Present {
		lower = g.Lower.Level
	}
	if g.Upper.Present {
		upper = g.Upper.Level
	}
	return lower, upper
}

// TestGrid_Levels verifies the levels of the kinds of 5.2 step 1 on the worked
// grids of 19.1: the geometric levels b*2^k and rdiv(b, 2^-k) with 0 never a
// level, the linear levels b+k*s, a listed set in the order given, a base at a
// bound, a base of 1 and a base of 0.
//
// Requirement: FR-CLB-010, FR-CLB-011, FR-CLB-013, FR-CLB-014
func TestGrid_Levels(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		rule         Rule
		base         int64
		b            Bounds
		off          Off
		window       []int64
		lower, upper int64
	}{
		// 7000 * 2 = 14000, * 4 = 28000; rdiv(7000, 2) = 3500, rdiv(7000, 4) = 1750.
		{"CostFloor 7000", geo(), 7000, bnds(1, 1_000_000), noOff(), []int64{1750, 3500, 7000, 14000, 28000}, 3500, 14000},
		// rdiv(50, 4) = 12.5, which half to even makes 12; the level 0 is never geo.
		{"EntryShare 50", geo(), 50, bnds(0, 10_000), noOff(), []int64{12, 25, 50, 100, 200}, 25, 100},
		// lin1000 round 4000: 2000 3000 4000 5000 6000, the bound 6849 stops the grid.
		{"MinWorkShare lin1000", lin(1000), 4000, bnds(0, 6849), offAt(0), []int64{2000, 3000, 4000, 5000, 6000}, 3000, 5000},
		// A base of 1: rdiv(1, 2) = 0.5 -> 0 is below the floor of 1, so no lower level.
		{"CoverTicks base 1", geo(), 1, bnds(0, 1000), noOff(), []int64{1, 2, 4}, -1, 2},
		// A base at its upper bound has no level above it.
		{"base at the upper bound", lin(250), 1000, bnds(0, 1000), noOff(), []int64{500, 750, 1000}, 750, -1},
		// A base of 0 is allowed for lin (TargetSpread), and is never cut.
		{"TargetSpread base 0", lin(500), 0, bnds(0, 9999), offAt(0), []int64{0, 500, 1000}, -1, 500},
		// A set keeps the order given: 3, 1, 2; the neighbours of 1 are its list neighbours.
		{"set in the order given", Rule{Kind: Set, Set: []int64{3, 1, 2}}, 1, bnds(0, 10), noOff(), []int64{3, 1, 2}, 3, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g, err := Build(tt.rule, tt.base, tt.b, tt.off, nil)
			if err != nil {
				t.Fatalf("Build(%v, %d, %v) error = %v", tt.rule, tt.base, tt.b, err)
			}
			if got := window(g); !slices.Equal(got, tt.window) {
				t.Errorf("Build(%v, %d, %v) levels near the base = %v, want %v", tt.rule, tt.base, tt.b, got, tt.window)
			}
			if lo, up := neighbourPair(g); lo != tt.lower || up != tt.upper {
				t.Errorf("Build(%v, %d, %v) neighbours = %d, %d, want %d, %d", tt.rule, tt.base, tt.b, lo, up, tt.lower, tt.upper)
			}
			if g.Levels[g.Base] != tt.base {
				t.Errorf("Build(%v, %d, %v) Levels[Base] = %d, want the base", tt.rule, tt.base, tt.b, g.Levels[g.Base])
			}
		})
	}
}

// TestGrid_WholeGeoGrid verifies the full geometric grid of CostFloor 7000 in
// 1 to 1 000 000: every level, with the half-to-even rounding of each halving.
//
// Requirement: FR-CLB-010, FR-CLB-013
func TestGrid_WholeGeoGrid(t *testing.T) {
	t.Parallel()
	// Halvings of 7000: 3500, 1750, 875, 437.5 -> 438 (tie to even), 218.75 -> 219,
	// 109.375 -> 109, 54.6875 -> 55, 27.34 -> 27, 13.67 -> 14, 6.84 -> 7, 3.42 -> 3,
	// 1.71 -> 2, 0.85 -> 1, which is the floor and ends the side. Doublings: 14000 to
	// 896000 (1 792 000 is above the bound).
	want := []int64{1, 2, 3, 7, 14, 27, 55, 109, 219, 438, 875, 1750, 3500, 7000,
		14000, 28000, 56000, 112000, 224000, 448000, 896000}
	g, err := Build(geo(), 7000, bnds(1, 1_000_000), noOff(), nil)
	if err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if !slices.Equal(g.Levels, want) {
		t.Errorf("Build(geo, 7000, 1..1000000) = %v, want %v", g.Levels, want)
	}
}

// TestGrid_BuildErrors verifies the errors of Build: a geo or z knob whose base is
// 0 has no grid (the record says unsearched at_off), and a base outside its
// bounds, a step below 1, bounds that are not ordered or a set without the base
// cannot make a grid.
//
// Requirement: FR-CLB-108, FR-CLB-014
func TestGrid_BuildErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		rule Rule
		base int64
		b    Bounds
		want error
	}{
		{"z base 0", geo(), 0, bnds(0, 1000), ErrNoGrid},
		{"base below the lower bound", lin(10), 5, bnds(10, 100), ErrBase},
		{"base above the upper bound", lin(10), 500, bnds(10, 100), ErrBase},
		{"bounds not ordered", lin(10), 50, bnds(100, 10), ErrBase},
		{"lin step 0", lin(0), 50, bnds(0, 100), ErrBase},
		{"geo base below 1", geo(), -1, bnds(-5, 100), ErrBase},
		{"set without the base", Rule{Kind: Set, Set: []int64{1, 2}}, 3, bnds(0, 9), ErrBase},
		{"empty set", Rule{Kind: Set}, 3, bnds(0, 9), ErrBase},
		{"unknown kind", Rule{Kind: Kind(9)}, 3, bnds(0, 9), ErrBase},
		{"too many levels", lin(1), 0, bnds(0, math.MaxInt64), ErrBase},
	}
	tests = append(tests, struct {
		name string
		rule Rule
		base int64
		b    Bounds
		want error
	}{"set above the level cap", Rule{Kind: Set, Set: make([]int64, maxLevels+1)}, 0, bnds(0, 9), ErrBase})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Build(tt.rule, tt.base, tt.b, noOff(), nil); !errors.Is(err, tt.want) {
				t.Errorf("Build(%v, %d, %v) error = %v, want %v", tt.rule, tt.base, tt.b, err, tt.want)
			}
		})
	}
}

// TestGrid_Cut verifies the cut of 5.2 step 2: the off value and every level
// beyond it on that side are removed and the base never, with the edge recorded
// as off where the next level was cut and as a bound where there was none.
//
// Requirement: FR-CLB-108
func TestGrid_Cut(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		rule         Rule
		base         int64
		b            Bounds
		off          Off
		first, last  int64
		lower, upper Edge
	}{
		// RestBand 500, lin250, off 0 in 0..10000: the uncut levels start 0, 250, 500; the cut
		// removes 0, so the lowest kept level is 250 and the next one below it was cut.
		{"RestBand 500", lin(250), 500, bnds(0, 10000), offAt(0), 250, 10000, None, None},
		// The same at base 250: nothing is left below, and the level that was is the off value.
		{"RestBand 250", lin(250), 250, bnds(0, 10000), offAt(0), 250, 10000, OffEdge, None},
		// TargetSpread 0 is its own off value: the base is kept, nothing is cut, below 0 is a bound.
		{"TargetSpread 0", lin(500), 0, bnds(0, 9999), offAt(0), 0, 9500, Bound, None},
		// ReviewTicks 10, off 1 is the lower bound: the level 1 is cut, 2 is the lowest.
		{"ReviewTicks off=1", geo(), 10, bnds(1, 1000), offAt(1), 2, 640, None, None},
		// An off value above the base cuts it and every level above it: 7000 and 8000... go.
		{"off above the base", lin(1000), 4000, bnds(0, 10000), offAt(7000), 0, 6000, None, None},
		{"base just below the off value", lin(1000), 6000, bnds(0, 10000), offAt(7000), 0, 6000, None, OffEdge},
		// An off value that is not a level of the grid still cuts what lies beyond it.
		{"off between levels", lin(1000), 4000, bnds(0, 10000), offAt(6500), 0, 6000, None, None},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g, err := Build(tt.rule, tt.base, tt.b, tt.off, nil)
			if err != nil {
				t.Fatalf("Build error = %v", err)
			}
			if g.Levels[0] != tt.first || g.Levels[len(g.Levels)-1] != tt.last {
				t.Errorf("Build(%v, %d, %v, off %v) ends = %d..%d, want %d..%d",
					tt.rule, tt.base, tt.b, tt.off, g.Levels[0], g.Levels[len(g.Levels)-1], tt.first, tt.last)
			}
			if lo, up := edges(g); lo != tt.lower || up != tt.upper {
				t.Errorf("Build(%v, %d, %v, off %v) edges = %v, %v, want %v, %v",
					tt.rule, tt.base, tt.b, tt.off, lo, up, tt.lower, tt.upper)
			}
			if offSurvives(g, tt.off) {
				t.Errorf("Build(%v, %d, %v, off %v) kept the off value", tt.rule, tt.base, tt.b, tt.off)
			}
		})
	}
}

// TestGrid_CutSet verifies that a set is cut by position, in the order given.
//
// Requirement: FR-CLB-108
func TestGrid_CutSet(t *testing.T) {
	t.Parallel()
	// Listed 5, 7, 9, 11, 13 with the base 9 and an off value 11: 11 and 13 go.
	g, err := Build(Rule{Kind: Set, Set: []int64{5, 7, 9, 11, 13}}, 9, bnds(0, 20), offAt(11), nil)
	if err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if want := []int64{5, 7, 9}; !slices.Equal(g.Levels, want) {
		t.Errorf("Build(set, off 11) = %v, want %v", g.Levels, want)
	}
	if g.Upper.Present || g.Upper.Edge != OffEdge {
		t.Errorf("Build(set, off 11) upper = %v, want an off edge", g.Upper)
	}
	// An off value that the set does not list cuts nothing.
	g, _ = Build(Rule{Kind: Set, Set: []int64{5, 7, 9}}, 7, bnds(0, 20), offAt(2), nil)
	if want := []int64{5, 7, 9}; !slices.Equal(g.Levels, want) {
		t.Errorf("Build(set, off 2) = %v, want %v", g.Levels, want)
	}
}

// TestGrid_Overflow verifies that the checked multiply of the geometric grid
// reports overflow as an error and never wraps: a doubling that leaves int64 is
// ErrOverflow, and the grid stops at the last level that fits.
//
// Requirement: FR-CLB-013
func TestGrid_Overflow(t *testing.T) {
	t.Parallel()
	if v, err := double(math.MaxInt64); !errors.Is(err, ErrOverflow) || v != 0 {
		t.Errorf("double(MaxInt64) = %d, %v, want 0, ErrOverflow", v, err)
	}
	if v, err := double(1 << 61); err != nil || v != 1<<62 {
		t.Errorf("double(2^61) = %d, %v, want 2^62", v, err)
	}
	// 2^62 doubled is 2^63: above MaxInt64, so not a level; a wrapped value would be negative.
	g, err := Build(geo(), 1<<62, bnds(1, math.MaxInt64), noOff(), nil)
	if err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if last := g.Levels[len(g.Levels)-1]; last != 1<<62 {
		t.Errorf("Build(geo, 2^62, 1..MaxInt64) last level = %d, want 2^62", last)
	}
	if !slices.IsSorted(g.Levels) || g.Levels[0] < 1 {
		t.Errorf("Build(geo, 2^62, ...) = %v, want ascending levels of at least 1", g.Levels)
	}
}

// TestGrid_Halve verifies the halvings that the checked divide cannot take,
// 2^63 and beyond, are rounded half to even.
//
// Requirement: FR-CLB-013
func TestGrid_Halve(t *testing.T) {
	t.Parallel()
	// rdiv(2^63 - 1, 2^63) is 1 (the quotient is just under 1): 2^63 does not fit in int64.
	if got := halve(math.MaxInt64, 63); got != 1 {
		t.Errorf("halve(MaxInt64, 63) = %d, want 1", got)
	}
	// rdiv(2^62, 2^63) = 0.5, a tie, which half to even makes 0.
	if got := halve(1<<62, 63); got != 0 {
		t.Errorf("halve(2^62, 63) = %d, want 0", got)
	}
	if got := halve(math.MaxInt64, 64); got != 0 {
		t.Errorf("halve(MaxInt64, 64) = %d, want 0", got)
	}
}

// TestGrid_Neighbours verifies that an invalid level is skipped, that the
// neighbour is the nearest kept valid level, and that a side with none is an
// edge: rule where the next level was invalid.
//
// Requirement: FR-CLB-012
func TestGrid_Neighbours(t *testing.T) {
	t.Parallel()
	// BeliefStep 200 in 1..2500: the resolution floor is ceil(10000 / v), and the cheapest
	// price is 96 (19.1). 10000/105 = 95.2 -> 96 is valid; 10000/104 = 96.2 -> 97 is not.
	// So a level below 105 is invalid, and 100 (floor 100) is the next level below 200.
	floor := func(v int64) bool { return (10000+v-1)/v <= 96 }
	g, err := Build(geo(), 200, bnds(1, 2500), noOff(), floor)
	if err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if g.Lower.Present || g.Lower.Edge != RuleEdge {
		t.Errorf("BeliefStep lower = %v, want a rule edge", g.Lower)
	}
	if !g.Upper.Present || g.Upper.Level != 400 {
		t.Errorf("BeliefStep upper = %v, want 400", g.Upper)
	}
	if want := []int64{200, 400, 800, 1600}; !slices.Equal(g.Levels, want) {
		t.Errorf("BeliefStep levels = %v, want %v", g.Levels, want)
	}
}

// TestGrid_NeighboursSkip verifies that an invalid level is skipped and not
// stopped at, and that the pack's own value is never skipped.
//
// Requirement: FR-CLB-012
func TestGrid_NeighboursSkip(t *testing.T) {
	t.Parallel()
	// ExitTicks 3, lin1 with EnterTicks 2: the level 2 is invalid (and so is 1).
	g, _ := Build(lin(1), 3, bnds(1, 1000), noOff(), func(v int64) bool { return v > 2 })
	if g.Lower.Present || g.Lower.Edge != RuleEdge || g.Upper.Level != 4 {
		t.Errorf("ExitTicks lower, upper = %v, %v, want a rule edge and 4", g.Lower, g.Upper)
	}
	// An invalid level in the middle is skipped, not stopped at: 2 and 4 invalid, base 3.
	g, _ = Build(lin(1), 3, bnds(0, 6), noOff(), func(v int64) bool { return v%2 == 1 })
	if lo, up := neighbourPair(g); lo != 1 || up != 5 {
		t.Errorf("neighbours past invalid levels = %d, %d, want 1, 5", lo, up)
	}
	// An invalid base is still a level (the pack's own value is never skipped).
	g, _ = Build(lin(1), 3, bnds(0, 6), noOff(), func(v int64) bool { return v != 3 })
	if g.Levels[g.Base] != 3 {
		t.Errorf("an invalid base was skipped: %v", g.Levels)
	}
}

// TestGrid_Distance verifies that the grid distance counts index steps over the
// kept valid levels, and is -1 for an index outside the grid.
//
// Requirement: FR-CLB-032
func TestGrid_Distance(t *testing.T) {
	t.Parallel()
	g, _ := Build(lin(10), 50, bnds(0, 100), noOff(), nil)
	for _, tt := range []struct {
		i, j int
		want int64
	}{{5, 5, 0}, {3, 6, 3}, {6, 3, 3}, {0, 10, 10}, {-1, 2, -1}, {2, 11, -1}} {
		if got := Distance(g, tt.i, tt.j); got != tt.want {
			t.Errorf("Distance(g, %d, %d) = %d, want %d", tt.i, tt.j, got, tt.want)
		}
	}
}

// TestGrid_Widen verifies the widening of 5.3: two more kept levels on a side
// that is not an edge, never past the grid (so never outside [Lo, Hi] and never
// onto a cut off value), and none on an edge.
//
// Requirement: FR-CLB-026
func TestGrid_Widen(t *testing.T) {
	t.Parallel()
	// RestBand 500: kept 250..10000 step 250, base index 1; the explore covers indices 0..3.
	g, _ := Build(lin(250), 500, bnds(0, 10000), offAt(0), nil)
	s := Explore(g)
	if s != (Span{0, 3}) {
		t.Fatalf("Explore = %v, want {0 3}: the second level below does not exist", s)
	}
	if !AtEdge(g, s, false) {
		t.Errorf("AtEdge(lower) = false, want true: the level below 250 was cut")
	}
	if _, ok := Widen(g, s, false); ok {
		t.Errorf("Widen below an edge reported ok")
	}
	w, ok := Widen(g, s, true)
	if !ok || w != (Span{0, 5}) {
		t.Errorf("Widen above = %v, %v, want {0 5}, true", w, ok)
	}
	for ok { // widening to the end never leaves the grid
		w, ok = Widen(g, w, true)
	}
	if w.Hi != len(g.Levels)-1 || g.Levels[w.Hi] != 10000 {
		t.Errorf("widened to %v, want the last level 10000", w)
	}
}

// TestGrid_WidenToTheEnd verifies that a widening with one level left stops at
// the end of the grid, and that downward widening has room below the explore.
//
// Requirement: FR-CLB-026
func TestGrid_WidenToTheEnd(t *testing.T) {
	t.Parallel()
	g, _ := Build(lin(250), 500, bnds(0, 10000), offAt(0), nil)
	// One level left after a widening of two: it stops at the end, not past it.
	w, ok := Widen(g, Span{0, len(g.Levels) - 2}, true)
	if !ok || w.Hi != len(g.Levels)-1 {
		t.Errorf("Widen with one level left = %v, %v, want to the end", w, ok)
	}
	// Downward on a grid with room below the explore.
	g, _ = Build(geo(), 7000, bnds(1, 1_000_000), noOff(), nil)
	s := Explore(g)
	w, ok = Widen(g, s, false)
	if !ok || w.Lo != s.Lo-2 {
		t.Errorf("Widen below = %v, %v, want Lo %d", w, ok, s.Lo-2)
	}
}

// FuzzGrid verifies, for any rule, base, bounds and off value, that Build does
// not panic and that a grid it returns is ascending (geo and lin), without
// duplicates, inside the bounds, holds the base at Base, has neighbours adjacent
// to the base, an edge for each missing neighbour and no off value except the
// base.
//
// Requirement: NFR-CLB-004
//
// #lizard forgives: one fuzz body, one property per block; the seven inputs are the corpus format.
func FuzzGrid(f *testing.F) {
	f.Add(uint8(0), int64(7000), int64(1), int64(1_000_000), int64(0), int64(0), false)
	f.Add(uint8(0), int64(50), int64(0), int64(10_000), int64(0), int64(0), false)
	f.Add(uint8(1), int64(500), int64(0), int64(10_000), int64(250), int64(0), true)
	f.Add(uint8(1), int64(0), int64(0), int64(9999), int64(500), int64(0), true)
	f.Add(uint8(0), int64(10), int64(1), int64(1000), int64(0), int64(1), true)
	f.Add(uint8(0), int64(1<<62), int64(1), int64(math.MaxInt64), int64(0), int64(0), false)
	f.Fuzz(func(t *testing.T, kind uint8, base, lo, hi, step, off int64, hasOff bool) {
		r := Rule{Kind: Kind(kind % 2), Step: step}
		g, err := Build(r, base, Bounds{lo, hi}, Off{hasOff, off}, nil)
		if err != nil {
			return
		}
		if !slices.IsSorted(g.Levels) || len(slices.Compact(slices.Clone(g.Levels))) != len(g.Levels) {
			t.Fatalf("Build(%v, %d, %d..%d) levels %v not strictly ascending", r, base, lo, hi, g.Levels)
		}
		if g.Levels[g.Base] != base {
			t.Fatalf("Build(%v, %d, %d..%d) Levels[Base] = %d, want the base", r, base, lo, hi, g.Levels[g.Base])
		}
		for _, v := range g.Levels {
			if v < lo || v > hi || (r.Kind == Geo && v < 1) || (hasOff && v == off && v != base) {
				t.Fatalf("Build(%v, %d, %d..%d, off %d/%v) has level %d", r, base, lo, hi, off, hasOff, v)
			}
		}
		if (g.Base > 0) != g.Lower.Present || (g.Base+1 < len(g.Levels)) != g.Upper.Present {
			t.Fatalf("Build(%v, %d, %d..%d) neighbours %v, %v do not match the levels %v", r, base, lo, hi, g.Lower, g.Upper, g.Levels)
		}
		if (!g.Lower.Present && g.Lower.Edge == None) || (!g.Upper.Present && g.Upper.Edge == None) {
			t.Fatalf("Build(%v, %d, %d..%d) a missing neighbour has no edge", r, base, lo, hi)
		}
	})
}
