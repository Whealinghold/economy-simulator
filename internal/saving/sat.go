package saving

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// scale is the scale of a value per tick and of a basis-point share
// (spec-saving-and-investment.md 5.3.1).
const scale = 10_000

// nonneg returns v, or 0 when v is negative.
func nonneg(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}

// satAdd returns a + b, or the extreme of the sign of the operands on overflow.
// Callers add non-negative terms, so an overflow is upward in practice; a
// negative a (an invalid balance) still saturates downward, never wraps up.
func satAdd(a, b int64) int64 {
	if s, ok := checked.Add(a, b); ok {
		return s
	}
	if a < 0 {
		return math.MinInt64
	}
	return math.MaxInt64
}

// satSub returns a - b, or the extreme of the sign of the exact result.
func satSub(a, b int64) int64 {
	if d, ok := checked.Sub(a, b); ok {
		return d
	}
	if a < 0 {
		return math.MinInt64
	}
	return math.MaxInt64
}

// satMulDiv returns a * b / c truncated toward zero with the product in 128
// bits, and the extreme of the exact sign when the quotient does not fit. The
// divisor c is positive at every call: a constant, a duration or a unit count
// of at least 1.
func satMulDiv(a, b, c int64) int64 {
	if q, ok := checked.MulDiv(a, b, c); ok {
		return q
	}
	if (a < 0) != (b < 0) {
		return math.MinInt64
	}
	return math.MaxInt64
}

// satMul returns a * b, saturating.
func satMul(a, b int64) int64 { return satMulDiv(a, b, 1) }

// satMulDivCeil is satMulDiv of non-negative operands rounded toward positive
// infinity; it saturates at MaxInt64.
func satMulDivCeil(a, b, c int64) int64 {
	if q, ok := checked.MulDivCeil(a, b, c); ok {
		return q
	}
	return math.MaxInt64
}

// capAt returns v limited to limit; v is never negative at a call.
func capAt(v, limit int64) int64 {
	if v > limit {
		return limit
	}
	return v
}
