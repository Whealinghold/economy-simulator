// Package checked is the one shared helper for integer arithmetic that can
// exceed int64 (architecture.md 3.5): a product or a ratio is computed in 128
// bits and reports, instead of wrapping, a result that does not fit.
//
// Every function returns an ok result that is false on overflow and, for the
// ratios, on a zero divisor. When ok is false the other results are zero and
// must not be used. The package imports the standard library only and holds no
// type of the root.
//
// Rounding: truncation toward zero is the default (MulDiv, MulDivRem, as Go's
// integer division). MulDivFloor rounds toward negative infinity, MulDivCeil
// toward positive infinity, and MulDivHalfEven to the nearest integer with ties
// to the even quotient.
package checked

import (
	"math"
	"math/bits"
)

// mode is a rounding rule of the quotient.
type mode uint8

const (
	modeTrunc mode = iota
	modeFloor
	modeCeil
	modeHalfEven
)

// Add returns a + b and whether the sum fits in int64.
func Add(a, b int64) (int64, bool) {
	s := a + b
	// Overflow occurred when both operands share a sign and the sum does not.
	if (a >= 0) == (b >= 0) && (s >= 0) != (a >= 0) {
		return 0, false
	}
	return s, true
}

// Sub returns a - b and whether the difference fits in int64.
func Sub(a, b int64) (int64, bool) {
	d := a - b
	// Overflow occurred when the operands differ in sign and the result's sign
	// differs from a's.
	if (a >= 0) != (b >= 0) && (d >= 0) != (a >= 0) {
		return 0, false
	}
	return d, true
}

// Mul returns a * b and whether the product fits in int64.
func Mul(a, b int64) (int64, bool) {
	hi, lo := bits.Mul64(abs(a), abs(b))
	if hi != 0 {
		return 0, false
	}
	return signed(lo, (a < 0) != (b < 0))
}

// MulDiv returns a * b / c, with the product held in 128 bits, truncated toward
// zero. ok is false when c is 0 or the quotient does not fit in int64.
func MulDiv(a, b, c int64) (q int64, ok bool) {
	q, _, ok = muldiv(a, b, c, modeTrunc)
	return q, ok
}

// MulDivRem returns the quotient of a * b / c truncated toward zero and the
// remainder r with a * b = q * c + r, which has the sign of a * b and magnitude
// below |c|. ok is false when c is 0 or the quotient does not fit in int64.
func MulDivRem(a, b, c int64) (q, r int64, ok bool) {
	return muldiv(a, b, c, modeTrunc)
}

// MulDivFloor returns a * b / c rounded toward negative infinity. ok is false
// when c is 0 or the result does not fit in int64.
func MulDivFloor(a, b, c int64) (q int64, ok bool) {
	q, _, ok = muldiv(a, b, c, modeFloor)
	return q, ok
}

// MulDivCeil returns a * b / c rounded toward positive infinity. ok is false
// when c is 0 or the result does not fit in int64.
func MulDivCeil(a, b, c int64) (q int64, ok bool) {
	q, _, ok = muldiv(a, b, c, modeCeil)
	return q, ok
}

// MulDivHalfEven returns a * b / c rounded to the nearest integer, a tie going
// to the even quotient. ok is false when c is 0 or the result does not fit in
// int64.
func MulDivHalfEven(a, b, c int64) (q int64, ok bool) {
	q, _, ok = muldiv(a, b, c, modeHalfEven)
	return q, ok
}

// muldiv works on magnitudes in uint64 (|MinInt64| is 2^63) and applies the
// sign and the rounding rule at the end. The remainder returned is that of the
// truncated division, whatever the mode.
func muldiv(a, b, c int64, m mode) (q, r int64, ok bool) {
	if c == 0 {
		return 0, 0, false
	}
	uc := abs(c)
	hi, lo := bits.Mul64(abs(a), abs(b))
	if hi >= uc {
		// The quotient needs 65 bits or more.
		return 0, 0, false
	}
	uq, ur := bits.Div64(hi, lo, uc)
	prodNeg := (a < 0) != (b < 0)
	neg := prodNeg != (c < 0)

	up := false // round the magnitude away from zero
	if ur != 0 {
		switch m {
		case modeFloor:
			up = neg
		case modeCeil:
			up = !neg
		case modeHalfEven:
			// 2*ur versus uc without overflowing: ur > uc-ur means 2*ur > uc.
			rest := uc - ur
			up = ur > rest || (ur == rest && uq&1 == 1)
		}
	}
	if up {
		var carry uint64
		uq, carry = bits.Add64(uq, 1, 0)
		if carry != 0 {
			return 0, 0, false
		}
	}
	q, ok = signed(uq, neg)
	if !ok {
		return 0, 0, false
	}
	r = int64(ur) // ur < uc <= 2^63, so it fits
	if prodNeg {
		r = -r
	}
	return q, r, true
}

// abs returns |v| as a uint64; |MinInt64| is 2^63 and does not overflow.
func abs(v int64) uint64 {
	if v < 0 {
		return -uint64(v)
	}
	return uint64(v)
}

// signed converts a magnitude and a sign to int64, reporting whether it fits.
func signed(mag uint64, neg bool) (int64, bool) {
	if neg {
		if mag > 1<<63 {
			return 0, false
		}
		return int64(-mag), true
	}
	if mag > math.MaxInt64 {
		return 0, false
	}
	return int64(mag), true
}
