package endowment

import "math/bits"

// u128 is an unsigned 128-bit value. The regeneration numerator reaches 10^31
// (spec-resources.md 4.6), beyond int64 and beyond internal/checked, which takes
// and returns int64; the need is this package's, so the helper is local.
type u128 struct{ hi, lo uint64 }

// mul64 returns the full 128-bit product of a and b.
func mul64(a, b uint64) u128 {
	hi, lo := bits.Mul64(a, b)
	return u128{hi, lo}
}

// mul returns x * c and whether it fits in 128 bits.
func (x u128) mul(c uint64) (u128, bool) {
	h1, lo := bits.Mul64(x.lo, c)
	h2, l2 := bits.Mul64(x.hi, c)
	hi, carry := bits.Add64(h1, l2, 0)
	return u128{hi, lo}, h2 == 0 && carry == 0
}

// add returns x + y and whether it fits in 128 bits.
func (x u128) add(y u128) (u128, bool) {
	lo, c := bits.Add64(x.lo, y.lo, 0)
	hi, c2 := bits.Add64(x.hi, y.hi, c)
	return u128{hi, lo}, c2 == 0
}

// divmod64 returns x / d and x mod d for d above 0. The quotient can be wider
// than 64 bits; the remainder is below d.
func (x u128) divmod64(d uint64) (q u128, r uint64) {
	qh, rh := x.hi/d, x.hi%d
	ql, r := bits.Div64(rh, x.lo, d)
	return u128{qh, ql}, r
}

// cmp returns -1, 0 or 1 as x is below, equal to or above y.
func (x u128) cmp(y u128) int {
	switch {
	case x.hi != y.hi:
		if x.hi < y.hi {
			return -1
		}
		return 1
	case x.lo < y.lo:
		return -1
	case x.lo > y.lo:
		return 1
	}
	return 0
}

// product returns a * b * c * d and whether it fits in 128 bits.
func product(a, b, c, d uint64) (u128, bool) {
	x, ok := mul64(a, b).mul(c)
	if !ok {
		return u128{}, false
	}
	return x.mul(d)
}
