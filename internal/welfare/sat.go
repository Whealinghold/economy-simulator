package welfare

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// The three saturating helpers are the only ones in the package: a value of
// the tick that leaves int64 becomes the nearest extreme (spec-welfare-and-
// wages.md 2.3.2). Only the final value of a quotient saturates.

// satAdd returns a + b, or the extreme of the operands' sign on overflow.
func satAdd(a, b int64) int64 {
	if s, ok := checked.Add(a, b); ok {
		return s
	}
	if a < 0 {
		return math.MinInt64
	}
	return math.MaxInt64
}

// satSub returns a - b, or the extreme of the sign of the exact result on
// overflow. It is never a sum with -b, because -MinInt64 is not an int64.
func satSub(a, b int64) int64 {
	if d, ok := checked.Sub(a, b); ok {
		return d
	}
	if a < 0 {
		return math.MinInt64
	}
	return math.MaxInt64
}

// satQuot returns the extreme for a quotient a * b / c that does not fit: the
// sign of a * b * c decides, and a zero is non-negative.
func satQuot(a, b, c int64) int64 {
	if (a < 0) != (b < 0) != (c < 0) {
		return math.MinInt64
	}
	return math.MaxInt64
}

// satMulDiv returns a * b / c, the product in 128 bits and one truncating
// division, saturating only the final value. A zero c is a violated
// programmer invariant: the caller handles its absent case first.
func satMulDiv(a, b, c int64) int64 {
	if c == 0 {
		panic("welfare: zero divisor passed to a saturating quotient (violated programmer invariant)")
	}
	if q, ok := checked.MulDiv(a, b, c); ok {
		return q
	}
	return satQuot(a, b, c)
}

// satMul returns a * b, saturating.
func satMul(a, b int64) int64 { return satMulDiv(a, b, 1) }

// satMulDivCeil is satMulDiv rounded toward positive infinity, for the one
// ceiling of the package, the wear cost.
func satMulDivCeil(a, b, c int64) int64 {
	if c == 0 {
		panic("welfare: zero divisor passed to a saturating quotient (violated programmer invariant)")
	}
	if q, ok := checked.MulDivCeil(a, b, c); ok {
		return q
	}
	return satQuot(a, b, c)
}
