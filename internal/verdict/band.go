package verdict

import (
	"errors"
	"slices"
)

// Side is the side of a band that is being set.
type Side uint8

// The two sides of a one-sided band.
const (
	// Upper sets the upper bound: max + headroom, rounded up.
	Upper Side = iota
	// Lower sets the lower bound: min - headroom, rounded down.
	Lower
)

// Errors of the band rule. A refusal is a finding for review (STY 8.4.4), not
// a value.
var (
	// ErrTooFewSeeds: a band is never set from fewer than 10 seeds (8.4.6).
	ErrTooFewSeeds = errors.New("verdict: a band needs at least 10 setting seeds")
	// ErrSeedCount: the number of values is not the number of setting seeds, so
	// a seed failed; a band is never set on a run with a failed step (8.4.1).
	ErrSeedCount = errors.New("verdict: values and setting seeds differ in number")
	// ErrAboveThreshold: the bound is looser than the regime threshold or the
	// derived bound it backs (8.4.4).
	ErrAboveThreshold = errors.New("verdict: bound is looser than the threshold it backs")
	// ErrOverflow: the arithmetic of the rule does not fit an int64.
	ErrOverflow = errors.New("verdict: value out of range")
	// ErrNoRuns: there are no runs to set from.
	ErrNoRuns = errors.New("verdict: no values")
	// ErrNoSeparation: the healthy and the provoked runs do not separate (6.5.3).
	ErrNoSeparation = errors.New("verdict: healthy and provoked runs do not separate")
)

// Headroom values of STY 8.4.3, in basis points of the observed range.
// They are placeholders (P) that a first reviewed run may move.
const (
	headroomLarge = 2_500 // 40 or more setting seeds
	headroomSmall = 5_000 // 10 to 39 setting seeds
	seedsLarge    = 40
	seedsSmall    = 10
)

// Limit is the regime threshold or derived bound that a band backs. A bound
// must not be looser than it (8.4.4). The zero value is no limit.
type Limit struct {
	Set   bool
	Value int64
}

// SetBand sets one bound of a band from the observed values of m setting
// seeds (STY 8.4.3, FR-STY-077, FR-STY-078): the upper bound is
// max + h*range and the lower min - h*range, with h 2 500 bp for m >= 40 and
// 5 000 bp for 10 <= m < 40; the quotient range*h/10 000 is rounded half to
// even (91*2 500/10 000 is 23); the result is rounded outward to two
// significant digits (up for the upper bound, down for the lower; a value
// under 100 in magnitude is kept). It refuses with an error rather than wrap
// or set a bound looser than limit: m < 10 (ErrTooFewSeeds), a number of
// values other than m (ErrSeedCount), a bound above (Upper) or below (Lower)
// the limit (ErrAboveThreshold), and arithmetic out of range (ErrOverflow).
func SetBand(values []int64, side Side, m int, limit Limit) (int64, error) {
	if m < seedsSmall {
		return 0, ErrTooFewSeeds
	}
	if len(values) != m {
		return 0, ErrSeedCount
	}
	h := int64(headroomSmall)
	if m >= seedsLarge {
		h = headroomLarge
	}
	return setBand(slices.Min(values), slices.Max(values), side, h, limit)
}

// setBand finishes SetBand once the extremes and the headroom are known.
func setBand(lo, hi int64, side Side, h int64, limit Limit) (int64, error) {
	// The range check that replaces a checked multiply (no math import, O4 and
	// D587): the span and span*h must fit an int64.
	if lo < 0 && hi > maxInt64+lo {
		return 0, ErrOverflow
	}
	span := hi - lo
	if span > maxInt64/h {
		return 0, ErrOverflow
	}
	add := rdiv(span*h, bpScale)
	var b int64
	if side == Upper {
		if hi > maxInt64-add {
			return 0, ErrOverflow
		}
		b = hi + add
	} else {
		if lo < minInt64+add {
			return 0, ErrOverflow
		}
		b = lo - add
	}
	b, ok := roundOutward(b, side == Upper)
	if !ok {
		return 0, ErrOverflow
	}
	if limit.Set && (side == Upper && b > limit.Value || side == Lower && b < limit.Value) {
		return b, ErrAboveThreshold
	}
	return b, nil
}

// roundOutward rounds x to two significant digits, up when up is true and down
// otherwise: 201 becomes 210 up and 200 down, -201 becomes -200 up and -210
// down; a value under 100 in magnitude is kept (STY 8.4.3). The second result
// is false if the rounded value does not fit an int64.
func roundOutward(x int64, up bool) (int64, bool) {
	ax := magnitude(x)
	if ax < 100 {
		return x, true
	}
	step := int64(1)
	for ax >= 100*uint64(step) {
		step *= 10
	}
	q := x / step
	if x%step < 0 { // floor, so q*step <= x
		q--
	}
	if up && q*step < x {
		if q+1 > maxInt64/step {
			return 0, false
		}
		q++
	}
	if q < minInt64/step {
		return 0, false
	}
	return q * step, true
}
