package technology

import (
	"math/bits"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// fractionBits is the number of bits of fraction of the fixed-point logarithm
// (spec-technology.md 4.2.1): units of 1/4096 of a doubling.
const fractionBits = 12

// Lg12 returns the fixed-point base-2 logarithm of v in units of 1/4096 of a
// doubling: 4096*k + (top12(v) - 4096) with k the floor of log2(v) and top12
// the twelve bits below the leading one, as spec-technology.md 4.2.1 defines
// it. It is exact at a power of two and linear in v between two. A v below 1
// has no logarithm and gives 0, the value of 1; Learned never passes one.
func Lg12(v int64) int64 {
	if v < 1 {
		return 0
	}
	u := uint64(v)
	k := bits.Len64(u) - 1
	var top uint64
	if k >= fractionBits {
		top = u >> (k - fractionBits)
	} else {
		top = u << (fractionBits - k)
	}
	return int64(k)<<fractionBits + int64(top) - 1<<fractionBits
}

// Learned returns the learned gain g(X) = doublingBP * (Lg12(x+base) -
// Lg12(base)) / 4096, truncating toward zero (spec-technology.md 4.2.1,
// FR-TEC-006). It is 0 at x = 0, non-decreasing in x, and exactly
// j*doublingBP when x+base = 2^j*base. ok is false for a base below 1, a
// negative x or doublingBP, or a sum or product that overflows int64.
func Learned(doublingBP, base, x int64) (gain int64, ok bool) {
	if base < 1 || x < 0 || doublingBP < 0 {
		return 0, false
	}
	v, ok := checked.Add(x, base)
	if !ok {
		return 0, false
	}
	return checked.MulDiv(doublingBP, Lg12(v)-Lg12(base), 1<<fractionBits)
}

// TrendStep applies the carried remainder of spec-technology.md 4.3.1 for one
// tick: carry+trendBP is split into add = sum / ticksPerYear and the remainder
// that is the next carry. Over any whole number of years the adds total
// exactly trendBP a year (FR-TEC-008). ok is false for a carry or trendBP
// below 0, ticksPerYear below 1, or a sum that overflows.
func TrendStep(carry, trendBP, ticksPerYear int64) (add, carry2 int64, ok bool) {
	if carry < 0 || trendBP < 0 || ticksPerYear < 1 {
		return 0, 0, false
	}
	sum, ok := checked.Add(carry, trendBP)
	if !ok {
		return 0, 0, false
	}
	return sum / ticksPerYear, sum % ticksPerYear, true
}
