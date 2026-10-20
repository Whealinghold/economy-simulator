package steady

import "math/big"

// floorDiv returns the floor of num/den and the remainder num - q*den, which is
// in [0, den) after den is made positive. It panics on a zero denominator, a
// violated programmer invariant that no input can cause.
func floorDiv(num, den *big.Int) (q, rem, d *big.Int) {
	if den.Sign() == 0 {
		panic("steady: invariant violated: zero denominator in a rounding")
	}
	n, d := new(big.Int).Set(num), new(big.Int).Set(den)
	if d.Sign() < 0 {
		n.Neg(n)
		d.Neg(d)
	}
	q, rem = new(big.Int), new(big.Int)
	q.DivMod(n, d, rem)
	return q, rem, d
}

// fit returns q as an int64 when it lies in [lo, hi], and 0, false otherwise.
func fit(q *big.Int, lo, hi int64) (int64, bool) {
	if !q.IsInt64() {
		return 0, false
	}
	if v := q.Int64(); v >= lo && v <= hi {
		return v, true
	}
	return 0, false
}

// RoundHalfEven returns num/den rounded to the nearest integer, an exact half
// going to the even neighbour (5/2 is 2, 7/2 is 4, -5/2 is -2, as clearing
// rounds, architecture.md 3.5). ok is false, and the value 0, when the result
// is outside [lo, hi] or does not fit int64. A zero den panics: it is a
// programmer invariant, not a result.
func RoundHalfEven(num, den *big.Int, lo, hi int64) (int64, bool) {
	q, rem, d := floorDiv(num, den)
	twice := rem.Lsh(rem, 1)
	switch cmp := twice.Cmp(d); {
	case cmp > 0, cmp == 0 && q.Bit(0) == 1:
		q.Add(q, big.NewInt(1))
	}
	return fit(q, lo, hi)
}

// Floor returns num/den rounded toward negative infinity under the same bound
// rule as RoundHalfEven. A zero den panics.
func Floor(num, den *big.Int, lo, hi int64) (int64, bool) {
	q, _, _ := floorDiv(num, den)
	return fit(q, lo, hi)
}

// Ceil returns num/den rounded toward positive infinity under the same bound
// rule as RoundHalfEven. A zero den panics.
func Ceil(num, den *big.Int, lo, hi int64) (int64, bool) {
	q, rem, _ := floorDiv(num, den)
	if rem.Sign() != 0 {
		q.Add(q, big.NewInt(1))
	}
	return fit(q, lo, hi)
}
