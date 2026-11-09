package measure

// RoundDiv returns n / d rounded to the nearest integer, an exact half going
// to the even integer and the rule symmetric for a negative n or d
// (spec-indicators.md 5.2.2): RoundDiv(-5, 2) = -2, RoundDiv(-3, 2) = -2,
// RoundDiv(25, 10) = 2, RoundDiv(35, 10) = 4. It returns false when d is 0, and
// never panics.
func RoundDiv(n, d Int128) (Int128, bool) {
	q, r, ok := n.mag.DivRem(d.mag)
	if !ok {
		return Int128{}, false
	}
	// r < d, so d - r does not wrap; q + 1 fits because r > 0 means d >= 2
	if rest, _ := d.mag.Sub(r); r.Cmp(rest) > 0 || (r.Cmp(rest) == 0 && q.Lo&1 == 1) {
		q, _ = q.Add(Uint128{0, 1})
	}
	return signed(n.neg != d.neg, q), true
}

// RoundDivU is RoundDiv for two unsigned values.
func RoundDivU(n, d Uint128) (Uint128, bool) {
	q, ok := RoundDiv(Int128{false, n}, Int128{false, d})
	return q.mag, ok
}
