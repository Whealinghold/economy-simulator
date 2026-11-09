package population

import (
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// Term returns exposed * rate, the numerator an agent adds to its death carry
// and the term a role adds to its birth carry (spec-population.md 3.8.1,
// 3.8.2). ok is false on overflow.
func Term(exposed, rate int64) (int64, bool) { return checked.Mul(exposed, rate) }

// Step adds numerator to carry and returns the whole events it holds and the
// remainder: events = (carry + numerator) / denom and next = (carry +
// numerator) mod denom, with next in [0, denom) (spec-population.md 3.8). Over
// any run of steps the events equal floor((carry0 + sum) / denom) less
// floor(carry0 / denom). A numerator of 0 returns the carry unchanged. ok is
// false for a denom below 1, a negative carry or numerator, or an overflow.
func Step(carry, numerator, denom int64) (events, next int64, ok bool) {
	if denom < 1 || carry < 0 || numerator < 0 {
		return 0, 0, false
	}
	total, ok := checked.Add(carry, numerator)
	if !ok {
		return 0, 0, false
	}
	return total / denom, total % denom, true
}

// Offset returns the start of the death carry of an agent: (Weyl32(id) * denom)
// / 2^32, truncated, a value in [0, denom) that is a fixed function of the id
// and draws nothing (spec-population.md 3.8.3). A denom below 1 gives 0.
func Offset(id, denom int64) int64 {
	if denom < 1 {
		return 0
	}
	// The product of a 32-bit Weyl value and a 63-bit denom is below 2^95; the
	// quotient by 2^32 is below denom, so it fits.
	v, _ := checked.MulDiv(int64(ordering.Weyl32(id)), denom, 1<<32)
	return v
}
