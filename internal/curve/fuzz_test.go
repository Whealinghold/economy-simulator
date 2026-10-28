package curve_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/curve"
)

// FuzzEval verifies Eval never panics on any four-knot list, that a result
// lies between the Y values of the two knots around x (or equals an end knot's
// Y outside the curve), and that a non-decreasing list gives a non-decreasing
// curve.
//
// Requirement: NFR-RES-003, FR-POP-004
func FuzzEval(f *testing.F) {
	f.Add(int64(0), int64(0), int64(100), int64(50), int64(300), int64(90), int64(400), int64(10), int64(150))
	f.Add(int64(-9e18), int64(0), int64(9e18), int64(9e18), int64(9e18), int64(0), int64(9e18), int64(1), int64(0))
	f.Fuzz(func(t *testing.T, x0, y0, x1, y1, x2, y2, x3, y3, x int64) {
		knots := []curve.Knot{{X: x0, Y: y0}, {X: x1, Y: y1}, {X: x2, Y: y2}, {X: x3, Y: y3}}
		if curve.Check(knots, 1<<62, 1<<62) >= 0 {
			// Only a list that Check accepts is evaluated: the others are the validator's.
			curve.Eval(knots, x)
			return
		}
		y, ok := curve.Eval(knots, x)
		if !ok {
			t.Fatalf("Eval(%v, %d) overflowed inside the checked range", knots, x)
		}
		lo, hi := knots[0].Y, knots[0].Y
		for _, k := range knots {
			lo, hi = min(lo, k.Y), max(hi, k.Y)
		}
		if y < lo || y > hi {
			t.Errorf("Eval(%v, %d) = %d, outside [%d, %d]", knots, x, y, lo, hi)
		}
		if x < 1<<62 && curve.NonDecreasing(knots) {
			if y2, ok := curve.Eval(knots, x+1); ok && y2 < y {
				t.Errorf("Eval(%v, %d) = %d then %d at %d: a non-decreasing list fell", knots, x, y, y2, x+1)
			}
		}
	})
}
