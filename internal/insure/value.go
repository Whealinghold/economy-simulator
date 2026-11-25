package insure

import (
	"math/bits"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// UnitValue returns v = min(published, trunc(valueCapBP * reference / 10 000)), the
// unit value of spec-insurance.md 6.2.1 (FR-INS-020): the published price capped at
// a multiple of the reference price. ok is false for a negative input or a product
// that overflows.
func UnitValue(published, reference, valueCapBP int64) (v int64, ok bool) {
	if published < 0 || reference < 0 || valueCapBP < 0 {
		return 0, false
	}
	ceiling, ok := checked.MulDiv(valueCapBP, reference, Basis)
	if !ok {
		return 0, false
	}
	return min(published, ceiling), true
}

// Free returns the room an underwriter has for new open liability (spec-insurance.md
// 5.2 step 8, 10.1): max(0, trunc(capBP * money / 10 000) - open), saturated at limit
// (spec-insurance.md 6.3.5). The product is made in 128 bits, so a capBP above 10 000
// and a money near the bound cannot wrap. It reports ok false for a negative input.
// An offer is refused `exposure` when its liability passes Free.
func Free(capBP, money, open, limit int64) (free int64, ok bool) {
	if capBP < 0 || money < 0 || open < 0 || limit < 0 {
		return 0, false
	}
	whole, fits := checked.MulDiv(capBP, money, Basis)
	if !fits {
		whole = limit // the 128-bit quotient passes int64: saturate
	}
	return max(0, min(whole, limit)-open), true
}

// AtRisk reports whether value is at least ruinShareBP basis points of money, that is
// 10 000 * value >= ruinShareBP * money, compared in 128 bits (spec-insurance.md 10.2,
// 6.3.5). It is the default insured's test that a cargo is large against what the
// household holds. A negative input is not at risk.
func AtRisk(value, ruinShareBP, money int64) bool {
	if value < 0 || ruinShareBP < 0 || money < 0 {
		return false
	}
	lh, ll := bits.Mul64(uint64(value), uint64(Basis))
	rh, rl := bits.Mul64(uint64(ruinShareBP), uint64(money))
	return lh > rh || (lh == rh && ll >= rl)
}
