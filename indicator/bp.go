package indicator

import "github.com/Whealinghold/economy-sim/internal/measure"

// BP is a ratio, an index, a share or a rate in basis points: 10 000 is 1.0
// (spec-indicators.md 5.1).
type BP int64

// MaxIndexBP is the largest index, ratio or growth: a value at or above it is
// clamped to it and flagged, never an error (spec-indicators.md 5.3.7).
const MaxIndexBP = measure.MaxIndexBP

// Quotient returns n / d rounded half to even and symmetric for a negative n
// (Quotient(-5, 2) = -2, Quotient(25, 10) = 2, Quotient(35, 10) = 4). A zero
// divisor makes the quotient undefined: ok is false and q is 0, and it is not an
// error and not a panic (spec-indicators.md 5.2.2, 5.2.4). err is ErrOverflow
// for the one quotient that does not fit int64 (the minimum over -1).
func Quotient(n, d int64) (q int64, ok bool, err error) {
	v, ok := measure.RoundDiv(measure.FromInt64(n), measure.FromInt64(d))
	if !ok {
		return 0, false, nil
	}
	if q, ok = measure.ToInt64(v); !ok {
		return 0, false, wrap("Quotient", "result", ErrOverflow)
	}
	return q, true, nil
}

// Annualise returns a total over a window of n ticks stated per year, the exact
// rdiv(total * ticksPerYear, n) with the product in 128 bits and one rounding
// (spec-indicators.md 5.7.2). ok is false when n is below 1 (a window with no
// ticks has no rate); err is ErrRange for ticksPerYear below 1 and ErrOverflow
// when the result does not fit int64.
func Annualise(total, ticksPerYear, n int64) (annual int64, ok bool, err error) {
	if ticksPerYear < 1 {
		return 0, false, wrap("Annualise", "ticks per year", ErrRange)
	}
	if n < 1 {
		return 0, false, nil
	}
	v, _ := measure.RoundDiv(measure.MulInt64(total, ticksPerYear), measure.FromInt64(n))
	if annual, ok = measure.ToInt64(v); !ok {
		return 0, false, wrap("Annualise", "result", ErrOverflow)
	}
	return annual, true, nil
}
