package insure

import "github.com/Whealinghold/economy-sim/internal/checked"

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
