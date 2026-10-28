package curve_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/curve"
)

// TestEval_InteriorTruncatesTowardZero verifies the interior rule of
// spec-resources.md 4.7.1 on a rising and a falling segment: the product is
// divided with truncation toward zero, so the result never leaves the two Y values.
//
// Requirement: FR-RES-013, FR-POP-004
func TestEval_InteriorTruncatesTowardZero(t *testing.T) {
	t.Parallel()
	rise := []curve.Knot{{X: 1000, Y: 9000}, {X: 6000, Y: 10000}}
	fall := []curve.Knot{{X: 0, Y: 10000}, {X: 5000, Y: 9000}}
	tests := []struct {
		name  string
		knots []curve.Knot
		x     int64
		want  int64
	}{
		// RES 16.1: 9 000 + trunc(1 000 * 3 321 / 5 000) = 9 000 + trunc(664.2) = 9 664.
		{"rising segment", rise, 4321, 9664},
		// 10 000 + trunc(-1 000 * 3 321 / 5 000) = 10 000 + trunc(-664.2) = 10 000 - 664 =
		// 9 336; flooring would give 9 335.
		{"falling segment truncates toward zero", fall, 3321, 9336},
		// 9 000 + trunc(1 000 * 1 / 5 000) = 9 000: a fraction below one unit adds nothing.
		{"first step of a rising segment adds nothing", rise, 1001, 9000},
		// 9 000 + trunc(1 000 * 4 999 / 5 000) = 9 000 + 999.
		{"last step before the upper knot", rise, 5999, 9999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := curve.Eval(tt.knots, tt.x)
			if !ok || got != tt.want {
				t.Errorf("Eval(%v, %d) = %d, %v, want %d, true", tt.knots, tt.x, got, ok, tt.want)
			}
		})
	}
}

// TestEval_FlatOutsideAndAtKnots verifies the curve is flat below the first
// knot and above the last, and equals a knot's Y exactly at the knot.
//
// Requirement: FR-RES-013, FR-POP-004
func TestEval_FlatOutsideAndAtKnots(t *testing.T) {
	t.Parallel()
	knots := []curve.Knot{{X: 2000, Y: 500}, {X: 5000, Y: 8000}, {X: 9000, Y: 8000}}
	tests := []struct {
		name string
		x    int64
		want int64
	}{
		{"below the first knot", -5, 500},
		{"at the first knot", 2000, 500},
		{"at the middle knot", 5000, 8000},
		{"on a flat segment", 7000, 8000},
		{"at the last knot", 9000, 8000},
		{"above the last knot", 10000, 8000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := curve.Eval(knots, tt.x)
			if !ok || got != tt.want {
				t.Errorf("Eval(%v, %d) = %d, %v, want %d, true", knots, tt.x, got, ok, tt.want)
			}
		})
	}
}

// TestEval_SingleKnot verifies one knot is a constant curve, and an empty list
// is reported rather than read.
//
// Requirement: FR-RES-013, FR-POP-004
func TestEval_SingleKnot(t *testing.T) {
	t.Parallel()
	one := []curve.Knot{{X: 3000, Y: 7000}}
	for _, x := range []int64{-1, 3000, 99999} {
		if got, ok := curve.Eval(one, x); !ok || got != 7000 {
			t.Errorf("Eval(%v, %d) = %d, %v, want 7000, true", one, x, got, ok)
		}
	}
	if got, ok := curve.Eval(nil, 5); ok || got != 0 {
		t.Errorf("Eval(nil, 5) = %d, %v, want 0, false", got, ok)
	}
}

// TestEval_OverflowReported verifies a segment whose difference does not fit in
// int64 is reported with ok false and never wrapped, while a wide segment that
// fits is exact through the 128-bit product.
//
// Requirement: NFR-RES-003
func TestEval_OverflowReported(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		knots  []curve.Knot
		x      int64
		want   int64
		wantOK bool
	}{
		// dy = MaxInt64 - (-5) leaves int64.
		{"dy overflows", []curve.Knot{{X: 0, Y: -5}, {X: 10, Y: math.MaxInt64}}, 5, 0, false},
		// dx = MaxInt64 - (-MaxInt64) leaves int64.
		{"dx overflows", []curve.Knot{{X: -math.MaxInt64, Y: 0}, {X: math.MaxInt64, Y: 10}}, 0, 0, false},
		// Equal Y over a span of 2^64 - 2: a wrapped dx would give a quotient that fits and hide the overflow.
		{"dx overflows with a flat segment", []curve.Knot{{X: -math.MaxInt64, Y: 5}, {X: math.MaxInt64, Y: 5}}, 0, 0, false},
		// dy = MaxInt64, x - x0 = 1, dx = 2: the product needs 128 bits and the
		// quotient trunc((2^63 - 1) / 2) = 2^62 - 1 fits.
		{"wide product fits", []curve.Knot{{X: 0, Y: 0}, {X: 2, Y: math.MaxInt64}}, 1, math.MaxInt64 / 2, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := curve.Eval(tt.knots, tt.x)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("Eval(%v, %d) = %d, %v, want %d, %v", tt.knots, tt.x, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestCheck_Rejections verifies Check names the first bad knot: out of range
// in X or in Y, equal or falling X, and -1 for a good or empty list.
//
// Requirement: FR-RES-013, FR-POP-004
func TestCheck_Rejections(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		knots []curve.Knot
		want  int
	}{
		{"empty list", nil, -1},
		{"good list", []curve.Knot{{0, 0}, {5000, 100}, {10000, 100000}}, -1},
		{"negative X", []curve.Knot{{-1, 0}}, 0},
		{"X above the maximum", []curve.Knot{{0, 0}, {10001, 0}}, 1},
		{"negative Y", []curve.Knot{{0, 0}, {1, -1}}, 1},
		{"Y above the maximum", []curve.Knot{{0, 100001}}, 0},
		{"equal X", []curve.Knot{{0, 0}, {5, 0}, {5, 1}}, 2},
		{"equal X at the second knot", []curve.Knot{{5, 0}, {5, 1}}, 1},
		{"falling X", []curve.Knot{{0, 0}, {5, 0}, {4, 1}}, 2},
		{"first bad knot wins", []curve.Knot{{0, 0}, {5, 200000}, {4, 0}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := curve.Check(tt.knots, 10000, 100000); got != tt.want {
				t.Errorf("Check(%v, 10000, 100000) = %d, want %d", tt.knots, got, tt.want)
			}
		})
	}
}

// TestNonDecreasing verifies equal Y values count as non-decreasing and a
// single fall anywhere does not.
//
// Requirement: FR-RES-013
func TestNonDecreasing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		knots []curve.Knot
		want  bool
	}{
		{"empty", nil, true},
		{"single", []curve.Knot{{1, 5}}, true},
		{"rising and flat", []curve.Knot{{0, 1}, {1, 1}, {2, 3}}, true},
		{"one fall", []curve.Knot{{0, 1}, {1, 3}, {2, 2}}, false},
		{"fall at the second knot", []curve.Knot{{0, 3}, {1, 1}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := curve.NonDecreasing(tt.knots); got != tt.want {
				t.Errorf("NonDecreasing(%v) = %v, want %v", tt.knots, got, tt.want)
			}
		})
	}
}

// TestSegment_BoundaryBelongsToOneSegment verifies a knot belongs to the
// segment that starts at it, so exactly one segment holds every x inside the
// curve, and that the points outside it, and the last knot, have none.
//
// Requirement: FR-RES-044
func TestSegment_BoundaryBelongsToOneSegment(t *testing.T) {
	t.Parallel()
	knots := []curve.Knot{{X: 0, Y: 0}, {X: 100, Y: 50}, {X: 300, Y: 90}}
	type seg struct {
		x0, y0, x1, y1 int64
		found          bool
	}
	tests := []struct {
		name string
		x    int64
		want seg
	}{
		{"below the first knot", -1, seg{}},
		{"at the first knot", 0, seg{0, 0, 100, 50, true}},
		{"inside the first segment", 99, seg{0, 0, 100, 50, true}},
		{"at the middle knot belongs to the second", 100, seg{100, 50, 300, 90, true}},
		{"inside the second segment", 299, seg{100, 50, 300, 90, true}},
		{"at the last knot has none", 300, seg{}},
		{"above the last knot has none", 301, seg{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got seg
			got.x0, got.y0, got.x1, got.y1, got.found = curve.Segment(knots, tt.x)
			if got != tt.want {
				t.Errorf("Segment(%v, %d) = %+v, want %+v", knots, tt.x, got, tt.want)
			}
		})
	}
	if _, _, _, _, found := curve.Segment(knots[:1], 0); found {
		t.Errorf("Segment of one knot found a segment, want none")
	}
}
