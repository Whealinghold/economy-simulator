package occupation

import (
	"math"
	"math/bits"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Expected returns E, the ratio a household would reach working its expected
// ticks, wage * expectedWork * 10 000 / (upkeep * ticksPerYear), truncated
// toward zero (spec-spawner.md 6.1, 6.1.1). has is false, and e 0, when upkeep
// or ticksPerYear is not above 0: there is nothing to divide by, and the role
// has no signal. The caller also gives no signal to a role with no observed
// wage (HasWage false) and does not call this for it.
//
// A quotient beyond int64 saturates to the nearest extreme by the sign of wage
// and is never an absence (6.1.2). It saturates only where the exact value
// leaves int64: when upkeep * ticksPerYear does not fit, upkeep is above
// MaxInt64 / MaxTicksPerYear and the exact value is below 1e14, so the quotient
// by upkeep is formed first and then divided by ticksPerYear, which equals the
// single quotient for a positive divisor. expectedWork is at most
// MaxTicksPerYear, so expectedWork * 10 000 fits; a value for which it does not
// (a hand-built view) saturates by the sign of wage.
func Expected(wage, expectedWork, upkeep, ticksPerYear int64) (e int64, has bool) {
	if upkeep <= 0 || ticksPerYear <= 0 {
		return 0, false
	}
	num, ok := checked.Mul(expectedWork, Basis)
	if !ok {
		return saturate(wage), true
	}
	if den, ok := checked.Mul(upkeep, ticksPerYear); ok {
		if e, ok = checked.MulDiv(wage, num, den); !ok {
			return saturate(wage), true
		}
		return e, true
	}
	q, ok := checked.MulDiv(wage, num, upkeep)
	if !ok {
		return saturate(wage), true
	}
	return q / ticksPerYear, true
}

// Viability returns V, the ratio a household would reach working every tick,
// wage * 10 000 / upkeep, truncated toward zero and saturated by the sign of
// wage when it leaves int64 (spec-spawner.md 6.1, 6.1.2). has is false when
// upkeep is not above 0.
func Viability(wage, upkeep int64) (v int64, has bool) {
	if upkeep <= 0 {
		return 0, false
	}
	v, ok := checked.MulDiv(wage, Basis, upkeep)
	if !ok {
		return saturate(wage), true
	}
	return v, true
}

// Beats reports whether a destination with signal eD clears the hysteresis
// band over the agent's own signal eO: eD * 10 000 >= eO * (10 000 +
// hysteresisBP), computed as eD >= ceil(eO * (10 000 + hysteresisBP) / 10 000),
// which is exactly equal for integers (spec-spawner.md 6.1.3, 15.1). The ceiling
// beyond int64 decides by its sign, which is what the exact inequality says
// since eD is within int64: a positive one is never reached, a negative one
// always is. hysteresisBP is at least 0 (validation).
func Beats(eD, eO, hysteresisBP int64) bool {
	t, ok := checked.MulDivCeil(eO, Basis+hysteresisBP, Basis)
	if !ok {
		return eO < 0
	}
	return eD >= t
}

// CapitalOK is the tool half of the capital test of spec-skill-barriers.md 2.5,
// the exact inequality diff * ut >= toolPrice * capitalReturn with diff = eD -
// eO and ut = upkeep * ticksPerYear (spec-spawner.md 6.2, FR-SPN-058). A
// toolPrice of 0 passes first, before any product is formed. The test is denied,
// never wrapped, when ut does not fit int64 or is not above 0, when diff is
// below 0, and for a negative toolPrice or capitalReturn (outside validation).
// Otherwise both sides are exact products in 128 bits: |diff| is below 2^64 and
// ut below 2^63, so neither product overflows.
func CapitalOK(eD, eO, toolPrice, capitalReturn, upkeep, ticksPerYear int64) bool {
	if toolPrice == 0 {
		return true
	}
	if toolPrice < 0 || capitalReturn < 0 || eD < eO {
		return false
	}
	ut, ok := checked.Mul(upkeep, ticksPerYear)
	if !ok || ut <= 0 {
		return false
	}
	// eD >= eO, so the wrapped uint64 difference is the exact |diff|.
	lh, ll := bits.Mul64(uint64(eD)-uint64(eO), uint64(ut))
	rh, rl := bits.Mul64(uint64(toolPrice), uint64(capitalReturn))
	return lh > rh || (lh == rh && ll >= rl)
}

// saturate returns the extreme of int64 on the side of v's sign, 0 for 0.
func saturate(v int64) int64 {
	switch {
	case v > 0:
		return math.MaxInt64
	case v < 0:
		return math.MinInt64
	}
	return 0
}
