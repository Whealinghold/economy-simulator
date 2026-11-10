package reduce

import "math/bits"

// RDiv returns n / d rounded to the nearest integer, an exact half going to the
// even neighbour (spec-experiments.md 8.3, the rule of spec-indicators.md 5.2.2):
// RDiv(-5, 2) = -2, RDiv(-3, 2) = -2, RDiv(25, 10) = 2, RDiv(35, 10) = 4. It
// returns false for a zero divisor, and for a quotient out of range, which is
// only -2^127 / -1.
func RDiv(n, d Int128) (Int128, bool) {
	if d.Sign() == 0 {
		return Int128{}, false
	}
	mn, nn := n.mag()
	md, nd := d.mag()
	q, r := mn.divmod(md)
	// r < md <= 2^127, so 2r fits the word pair.
	twice := u128{r.hi<<1 | r.lo>>63, r.lo << 1}
	if c := twice.cmp(md); c > 0 || (c == 0 && q.lo&1 == 1) {
		lo, carry := bits.Add64(q.lo, 1, 0)
		q = u128{q.hi + carry, lo}
	}
	return fromMag(q, nn != nd)
}

// IsqrtNearest returns the integer nearest to the square root of n: the floor
// root f, and f + 1 when n - f*f is above f, that is when n >= f*f + f + 1.
// There is no tie, because (f + 1/2)^2 is never an integer. The root of a
// 127-bit value is below 2^64. A negative n has no root and returns 0; the
// callers pass a variance, which is never negative.
func IsqrtNearest(n Int128) uint64 {
	if n.IsNeg() {
		return 0
	}
	m, _ := n.mag()
	var f uint64 // the floor root, built from its top bit down
	for b := 63; b >= 0; b-- {
		t := f | 1<<uint(b)
		sh, sl := bits.Mul64(t, t)
		if (u128{sh, sl}).cmp(m) <= 0 {
			f = t
		}
	}
	sh, sl := bits.Mul64(f, f)
	rl, c := bits.Sub64(m.lo, sl, 0)
	rh, _ := bits.Sub64(m.hi, sh, c)
	if rh != 0 || rl > f {
		return f + 1
	}
	return f
}
