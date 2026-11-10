package calibration

import "math"

// Point is one point of a screen or an explore as the choice reads it
// (spec-calibration.md 7.2.3): its index in the experiment, its grid index
// relative to the base for each knob (0 for the base level, -1 for the lower
// neighbour), whether every X, O and detector criterion passed on all seeds,
// and for each B criterion with a target the number of seeds on which it passed.
type Point struct {
	Index     int
	Levels    []int
	ExactPass bool
	BPass     []int
}

// Choice is the result of Choose. When None is true there is no choice and
// Pareto holds the non-dominated points; otherwise Index is the chosen point and
// Distance its grid distance to the base. Feasible lists the feasible point
// indices in the order of the points.
type Choice struct {
	Index    int
	Distance int64
	Feasible []int
	Pareto   []int
	None     bool
}

// Choose is the pure choice of spec-calibration.md 7.2.3 (FR-CLB-030 to
// FR-CLB-034). base is the Index of the base point and r the replicas of a
// point. A point is feasible when ExactPass holds and every B count is at least
// Threshold(r); a count a point does not list is 0. The base wins when it is
// feasible; otherwise the feasible point with the least grid distance, then the
// fewest knobs changed, then the lowest Index. With no feasible point the result
// is None and the Pareto set of the B count vectors among the points that
// pass every exact criterion. No weighted sum is computed.
func Choose(points []Point, base, r int) Choice {
	need := Threshold(r)
	width := 0
	for _, p := range points {
		width = max(width, len(p.BPass))
	}
	var feasible []Point
	for _, p := range points {
		if p.ExactPass && passesAll(p, width, need) {
			feasible = append(feasible, p)
		}
	}
	c := Choice{}
	for _, p := range feasible {
		c.Feasible = append(c.Feasible, p.Index)
	}
	if len(feasible) == 0 {
		c.None = true
		c.Pareto = pareto(points, width)
		return c
	}
	best := feasible[0]
	for _, p := range feasible {
		if p.Index == base {
			best = p
			break
		}
		if better(p, best) {
			best = p
		}
	}
	c.Index, c.Distance = best.Index, gridDistance(best.Levels)
	return c
}

// passesAll reports whether every one of width B counts reaches need.
func passesAll(p Point, width, need int) bool {
	for i := range width {
		if bCount(p, i) < need {
			return false
		}
	}
	return true
}

// bCount is the pass count of B criterion i, 0 for a count the point lacks.
func bCount(p Point, i int) int {
	if i < len(p.BPass) {
		return p.BPass[i]
	}
	return 0
}

// better orders two feasible points: less grid distance, then fewer knobs
// changed, then the lower index.
func better(a, b Point) bool {
	if da, db := gridDistance(a.Levels), gridDistance(b.Levels); da != db {
		return da < db
	}
	if ca, cb := changed(a.Levels), changed(b.Levels); ca != cb {
		return ca < cb
	}
	return a.Index < b.Index
}

// gridDistance is the sum over knobs of the absolute grid index, saturating at
// MaxInt64 (a level index is small; the saturation only keeps a fuzzed input
// from wrapping).
func gridDistance(levels []int) int64 {
	var d uint64
	for _, v := range levels {
		m := uint64(v)
		if v < 0 {
			m = -uint64(v) // two's complement: exact for MinInt as well
		}
		if d += m; d < m || d > math.MaxInt64 {
			return math.MaxInt64
		}
	}
	return int64(d)
}

// changed counts the knobs off the base level.
func changed(levels []int) int {
	n := 0
	for _, v := range levels {
		if v != 0 {
			n++
		}
	}
	return n
}

// pareto returns the Index of every point that passes the exact criteria and
// whose vector of B counts no other such point dominates (at least as many on
// every criterion and more on one), in the order of the points (7.5 item 3).
func pareto(points []Point, width int) []int {
	var exact []Point
	for _, p := range points {
		if p.ExactPass {
			exact = append(exact, p)
		}
	}
	var out []int
	for i, p := range exact {
		dominated := false
		for j, q := range exact {
			if i != j && dominates(q, p, width) {
				dominated = true
				break
			}
		}
		if !dominated {
			out = append(out, p.Index)
		}
	}
	return out
}

// dominates reports whether q is at least as good as p on every B criterion and
// better on one.
func dominates(q, p Point, width int) bool {
	better := false
	for i := range width {
		switch {
		case bCount(q, i) < bCount(p, i):
			return false
		case bCount(q, i) > bCount(p, i):
			better = true
		}
	}
	return better
}

// SpreadLevel maps a Spread value x to the index in g.Levels of the nearest
// level; a tie goes to the lower level, a rule of its own and not a division
// (7.2.3, FR-CLB-013). It returns -1 for a grid with no level.
func SpreadLevel(g Grid, x int64) int {
	best := -1
	var bestGap uint64
	for i, l := range g.Levels {
		gap := uint64(x) - uint64(l) // two's complement: the exact distance as uint64
		if l > x {
			gap = uint64(l) - uint64(x)
		}
		if best < 0 || gap < bestGap || (gap == bestGap && l < g.Levels[best]) {
			best, bestGap = i, gap
		}
	}
	return best
}
