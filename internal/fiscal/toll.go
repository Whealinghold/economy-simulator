package fiscal

import "github.com/Whealinghold/economy-sim/internal/checked"

// Toll returns the toll of one crossing for a shipment of n carrying units
// (spec-fiscal.md 7.3): floor(amount * scale * max(1, n) / 10^4), one
// truncation and no carry, a toll having no payer state. ok is false for a
// negative argument or a toll that leaves int64.
func Toll(amount, scale, n int64) (due int64, ok bool) {
	if amount < 0 || scale < 0 || n < 0 {
		return 0, false
	}
	a, aok := checked.Mul(amount, scale)
	if !aok {
		return 0, false
	}
	return checked.MulDiv(a, max(1, n), Basis)
}

// Total returns the sum of the amounts of a shipment's crossings, in the order
// the caller lists them (path order, then ascending toll ID): the one number the
// refusal of spec-fiscal.md 7.4 checks. ok is false for a negative amount or a
// sum that leaves int64, which TRN reports as the refusal overflow.
func Total(amounts []int64) (total int64, ok bool) {
	for _, a := range amounts {
		if a < 0 {
			return 0, false
		}
		if total, ok = checked.Add(total, a); !ok {
			return 0, false
		}
	}
	return total, true
}
