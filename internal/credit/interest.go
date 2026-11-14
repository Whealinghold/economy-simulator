package credit

import "github.com/Whealinghold/economy-sim/internal/checked"

// Interest returns the interest of a claim of principal x at the yearly rate
// rateBP, in basis points, for a term of term ticks in a year of ticksPerYear
// ticks: x * rateBP * term / (10 000 * ticksPerYear), rounded half to even
// (spec-credit.md 7.1.1, 7.1.2, FR-CRD-027). The numerator is held in 128 bits,
// so the largest legal product, MaxMoney * MaxCreditRateBP * MaxCreditTerm, is
// exact. ok is false, and the interest 0, for a negative x, rateBP or term, a
// ticksPerYear below 1, or an interest that does not fit in int64; it never wraps.
func Interest(x, rateBP, term, ticksPerYear int64) (interest int64, ok bool) {
	if x < 0 || rateBP < 0 || term < 0 || ticksPerYear < 1 {
		return 0, false
	}
	rn, ok := checked.Mul(rateBP, term)
	if !ok {
		return 0, false
	}
	d, ok := checked.Mul(Basis, ticksPerYear)
	if !ok {
		return 0, false
	}
	q, ok := checked.MulDivHalfEven(x, rn, d)
	if !ok {
		return 0, false
	}
	return q, true
}

// Effective returns the effective rate of a posted rate: rateBP * scaleBP /
// 10 000 rounded half to even, clamped to maxRateBP (spec-credit.md 6.3.1,
// FR-CRD-024). ok is false for a negative rate or scale or a maxRateBP below 0.
// A scale of 0 gives 0 (interest-free).
func Effective(rateBP, scaleBP, maxRateBP int64) (rate int64, ok bool) {
	if rateBP < 0 || scaleBP < 0 || maxRateBP < 0 {
		return 0, false
	}
	q, ok := checked.MulDivHalfEven(rateBP, scaleBP, Basis)
	if !ok {
		return maxRateBP, true // the quotient does not fit in int64: far above any cap
	}
	return min(q, maxRateBP), true
}
