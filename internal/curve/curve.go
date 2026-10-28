// Package curve evaluates a piecewise-linear curve given as a list of knots. It
// is the one evaluation shared by the abundance curve of the resources module
// (spec-resources.md 4.7.1) and the schedules of the population module
// (spec-population.md 3.5.1). It knows no share, factor, ratio or rate: the
// ranges are the caller's.
package curve

import (
	"sort"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Knot is a point of a curve. Knots are listed in strictly increasing X.
type Knot struct{ X, Y int64 }

// Eval returns the curve at x. It is flat at the first knot's Y below the first
// knot and at the last knot's Y above the last. Between knot i and i+1, with
// dx = X[i+1] - X[i], it is Y[i] + trunc((Y[i+1] - Y[i]) * (x - X[i]) / dx),
// truncating toward zero (also for a falling segment), so a result always lies
// between the two Y values. ok is false for an empty list or on overflow.
func Eval(knots []Knot, x int64) (y int64, ok bool) {
	n := len(knots)
	switch {
	case n == 0:
		return 0, false
	case x <= knots[0].X:
		return knots[0].Y, true
	case x >= knots[n-1].X:
		return knots[n-1].Y, true
	}
	x0, y0, x1, y1, _ := Segment(knots, x)
	dy, ok := checked.Sub(y1, y0)
	if !ok {
		return 0, false
	}
	dx, ok := checked.Sub(x1, x0)
	if !ok {
		return 0, false
	}
	off, ok := checked.Sub(x, x0)
	if !ok {
		return 0, false
	}
	step, ok := checked.MulDiv(dy, off, dx)
	if !ok {
		return 0, false
	}
	return checked.Add(y0, step)
}

// Segment returns the segment that contains x, with x0 <= x < x1: a knot belongs
// to the segment that starts at it, so exactly one segment holds every x. found
// is false below the first knot, at or above the last, and for fewer than two
// knots, where the curve is flat.
func Segment(knots []Knot, x int64) (x0, y0, x1, y1 int64, found bool) {
	n := len(knots)
	if n < 2 || x < knots[0].X || x >= knots[n-1].X {
		return 0, 0, 0, 0, false
	}
	// i is the first knot above x, so knots[i-1].X <= x < knots[i].X.
	i := sort.Search(n, func(i int) bool { return knots[i].X > x })
	a, b := knots[i-1], knots[i]
	return a.X, a.Y, b.X, b.Y, true
}

// Check returns the index of the first knot that is out of range (X outside
// [0, xMax] or Y outside [0, yMax]) or whose X is not above the one before, and
// -1 if there is none, including for an empty list.
func Check(knots []Knot, xMax, yMax int64) (bad int) {
	for i, k := range knots {
		switch {
		case k.X < 0 || k.X > xMax || k.Y < 0 || k.Y > yMax:
			return i
		case i > 0 && k.X <= knots[i-1].X:
			return i
		}
	}
	return -1
}

// NonDecreasing reports whether Y never falls from one knot to the next.
func NonDecreasing(knots []Knot) bool {
	for i := 1; i < len(knots); i++ {
		if knots[i].Y < knots[i-1].Y {
			return false
		}
	}
	return true
}
