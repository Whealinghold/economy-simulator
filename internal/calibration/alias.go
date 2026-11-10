package calibration

import (
	"fmt"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// The alias test of spec-calibration.md 10.2. The bounds keep both sides of the
// comparison below 2^127 (10 000 * 16^2 * 2^104).
const (
	// AliasTolBP is alias_tol_bp: cos^2 of at least 10 000 - 500 basis points makes
	// two knobs aliased (placeholder of 10.2).
	AliasTolBP = 500
	// maxReading is the largest absolute elasticity the test reads: 2^26 (10.2).
	// EXP clamps at 2^31 - 1, a wider range, and the caller excludes a reading
	// above this one before it calls.
	maxReading = 1 << 26
	// maxMeasures is the most measures the test takes at a time (10.2).
	maxMeasures = 16
	// fullBP is 10 000, one in basis points, the scale of cos^2.
	fullBP = 10_000
)

// ErrAlias is returned for input the alias test cannot read: vectors of
// different lengths, more than 16 measures, a reading above 2^26 in absolute
// value, or a tolerance outside 0 to 10 000. A reading error names its index.
const ErrAlias = calError("calibration: alias test input out of bounds")

// Aliased reports whether the elasticity vectors ei and ej, over the measures
// where both are defined and OK, are collinear to the tolerance tolBP:
//
//	10 000 * (ei.ej)^2 >= (10 000 - tolBP) * (ei.ei) * (ej.ej)
//
// in exact integers (FR-CLB-061). A zero vector is never aliased: it is
// invisible (10.5). A vector of one measure is aliased with every other non-zero
// vector of one measure. The caller excludes a reading that is not OK, above
// 2^26 or clamped before the call; one that is still there is an error, never
// a wrapped value.
func Aliased(ei, ej []int64, tolBP int64) (bool, error) {
	if err := checkAliasInput(ei, ej, tolBP); err != nil {
		return false, err
	}
	// With |e| <= 2^26 and at most 16 measures every sum below is at most 2^56, so
	// plain int64 arithmetic cannot overflow here.
	var dot, ni, nj int64
	for k := range ei {
		dot += ei[k] * ej[k]
		ni += ei[k] * ei[k]
		nj += ej[k] * ej[k]
	}
	if ni == 0 || nj == 0 {
		return false, nil
	}
	if dot < 0 {
		dot = -dot // only (ei.ej)^2 is compared
	}
	// ok is false only for a negative factor, and none is: tolBP is at most 10 000.
	cmp, ok := checked.Cmp3(fullBP, dot, dot, fullBP-tolBP, ni, nj)
	return ok && cmp >= 0, nil
}

// checkAliasInput returns ErrAlias for vectors of different lengths, more than 16
// measures, a tolerance outside 0 to 10 000, or a reading above 2^26 in absolute
// value, which it names by index.
func checkAliasInput(ei, ej []int64, tolBP int64) error {
	if len(ei) != len(ej) || len(ei) > maxMeasures || tolBP < 0 || tolBP > fullBP {
		return ErrAlias
	}
	for k := range ei {
		for _, v := range [2]int64{ei[k], ej[k]} {
			if v > maxReading || v < -maxReading {
				return fmt.Errorf("%w: reading %d is %d", ErrAlias, k, v)
			}
		}
	}
	return nil
}
