package risk

import "github.com/Whealinghold/economy-sim/internal/checked"

// Phi returns the part of an account's load in one dimension that a store of
// capacity capK holds, in basis points of the load (spec-risk.md 6.1.2):
//
//	x   = floor(stored * capK / sumCap)    0 when sumCap is 0
//	phi = floor(x * 10 000 / load)         0 when load is 0
//
// The first product is held in 128 bits. ok is false when an argument is
// negative or the second product does not fit.
func Phi(stored, sumCap, capK, load int64) (int64, bool) {
	if stored < 0 || sumCap < 0 || capK < 0 || load < 0 {
		return 0, false
	}
	if sumCap == 0 || load == 0 {
		return 0, true
	}
	x, ok := checked.MulDiv(stored, capK, sumCap)
	if !ok {
		return 0, false
	}
	return checked.MulDiv(x, BP, load)
}

// Larger returns the larger of the two dimensions' fractions, the cautious
// reading that exposes more (spec-risk.md 6.1.3).
func Larger(a, b int64) int64 {
	return max(a, b)
}

// Share returns the share of a store class, min(10 000, n * rhe(phi *
// severityBP, 10 000)), for n struck stores (spec-risk.md 6.3.2). A product that
// leaves int64 is far above 10 000 and gives 10 000. ok is false when an
// argument is negative.
func Share(n, phi, severityBP int64) (int64, bool) {
	if n < 0 {
		return 0, false
	}
	each, ok := rhe(phi, severityBP)
	if !ok {
		return 0, false
	}
	total, ok := checked.Mul(n, each)
	if !ok {
		return BP, true
	}
	return min(total, BP), true
}

// Apply returns the units a class takes of a commodity held at q0 units at the
// start of the position, rhe(q0 * share, 10 000) (spec-risk.md 6.3.2). ok is
// false when q0 is negative or share is outside 0 to 10 000.
func Apply(q0, share int64) (int64, bool) {
	if share > BP {
		return 0, false
	}
	return rhe(q0, share)
}

// Capped returns the units of loss that may still be taken of a commodity: the
// total taken of it never exceeds q0 (spec-risk.md 6.3.2). taken is what the
// earlier classes took.
func Capped(q0, taken, loss int64) int64 {
	return max(0, min(loss, q0-taken))
}
