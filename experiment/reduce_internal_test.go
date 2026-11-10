package experiment

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/reduce"
)

// TestQuotients_EdgeCases verifies the rounding helpers of the comparison at
// their edges: a zero divisor is undefined (0 for rdivInt), a quotient beyond
// int64 is undefined for quotient and is cut to the clamp, with its sign, by
// clampedQuotient, and a quotient inside the clamp is exact and not flagged.
//
// Requirement: FR-EXP-040, FR-EXP-044
func TestQuotients_EdgeCases(t *testing.T) {
	t.Parallel()
	zero := reduce.FromInt64(0)
	one := reduce.FromInt64(1)
	huge, ok := reduce.FromInt64(1 << 62).Mul(reduce.FromInt64(1 << 62)) // 2^124
	if !ok {
		t.Fatal("2^124 must fit in 128 bits")
	}
	if got := rdivInt(7, 0); got != 0 {
		t.Errorf("rdivInt(7, 0) = %d, want 0", got)
	}
	if got := rdivInt(-5, 2); got != -2 {
		t.Errorf("rdivInt(-5, 2) = %d, want -2 (half to even)", got)
	}
	if q := quotient(one, zero); q.OK {
		t.Errorf("quotient by zero = %+v, want undefined", q)
	}
	if q := quotient(huge, one); q.OK {
		t.Errorf("quotient beyond int64 = %+v, want undefined", q)
	}
	if q, cut := clampedQuotient(one, zero); q.OK || cut {
		t.Errorf("clampedQuotient by zero = %+v, %v, want undefined and not cut", q, cut)
	}
	if q, cut := clampedQuotient(huge, one); q != (Reading{clampBP, true}) || !cut {
		t.Errorf("clampedQuotient(+huge) = %+v, %v, want +clamp, cut", q, cut)
	}
	neg, _ := huge.Neg()
	if q, cut := clampedQuotient(neg, one); q != (Reading{-clampBP, true}) || !cut {
		t.Errorf("clampedQuotient(-big) = %+v, %v, want -clamp, cut", q, cut)
	}
	if q, cut := clampedQuotient(reduce.FromInt64(-12), reduce.FromInt64(8)); q != (Reading{-2, true}) || cut {
		t.Errorf("clampedQuotient(-12, 8) = %+v, %v, want -2 (half to even), not cut", q, cut)
	}
}

// TestPasses_UnknownOpAndMeasure verifies that a criterion of an unknown
// operator, of an unknown measure, or over an undefined reading does not pass.
//
// Requirement: FR-EXP-043
func TestPasses_UnknownOpAndMeasure(t *testing.T) {
	t.Parallel()
	e := Spec{Measures: []Measure{{Name: "m"}}}
	final := []Reading{{Value: 5, OK: true}}
	for name, c := range map[string]Criterion{
		"unknown op":      {Measure: "m", Op: Op(99), A: 0},
		"unknown measure": {Measure: "x", Op: AtLeast, A: 0},
	} {
		if e.passes(c, final) {
			t.Errorf("%s passed", name)
		}
	}
	if e.passes(Criterion{Measure: "m", Op: AtLeast, A: 0}, []Reading{{Value: 5}}) {
		t.Error("an undefined reading passed")
	}
	if !e.passes(Criterion{Measure: "m", Op: Between, A: 5, B: 5}, final) {
		t.Error("Between is inclusive at both ends")
	}
}
