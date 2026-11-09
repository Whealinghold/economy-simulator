// Package measure is the checked integer arithmetic under the indicator
// package (spec-indicators.md 5): signed and unsigned 128-bit sums and
// products, the half-to-even quotient of two 128-bit values, the nearest
// integer square root, the Gini accumulation and the nearest-rank quantile.
//
// It works on primitive numbers only and knows no commodity, region or agent
// (architecture.md 2.9). It imports the standard library and internal/checked,
// the one shared 128-bit helper (plan 2-18 D562), and no floating-point type:
// every quotient is rounded half to even at its last division.
package measure

import "math/bits"

// Uint128 is an unsigned 128-bit integer. The zero value is 0.
type Uint128 struct{ Hi, Lo uint64 }

// IsZero reports whether a is 0.
func (a Uint128) IsZero() bool { return a.Hi == 0 && a.Lo == 0 }

// Cmp returns -1, 0 or 1 as a is below, equal to or above b.
func (a Uint128) Cmp(b Uint128) int {
	switch {
	case a.Hi != b.Hi:
		if a.Hi < b.Hi {
			return -1
		}
		return 1
	case a.Lo != b.Lo:
		if a.Lo < b.Lo {
			return -1
		}
		return 1
	}
	return 0
}

// Add returns a + b, and false when the sum does not fit in 128 bits.
func (a Uint128) Add(b Uint128) (Uint128, bool) {
	lo, c := bits.Add64(a.Lo, b.Lo, 0)
	hi, c := bits.Add64(a.Hi, b.Hi, c)
	return Uint128{hi, lo}, c == 0
}

// Sub returns a - b, and false when b is above a.
func (a Uint128) Sub(b Uint128) (Uint128, bool) {
	lo, br := bits.Sub64(a.Lo, b.Lo, 0)
	hi, br := bits.Sub64(a.Hi, b.Hi, br)
	return Uint128{hi, lo}, br == 0
}

// Mul64 returns the exact product a * b, which always fits.
func Mul64(a, b uint64) Uint128 {
	hi, lo := bits.Mul64(a, b)
	return Uint128{hi, lo}
}

// MulU64 returns a * m, and false when the product does not fit in 128 bits.
func (a Uint128) MulU64(m uint64) (Uint128, bool) {
	hi, lo := bits.Mul64(a.Lo, m)
	over, cross := bits.Mul64(a.Hi, m)
	hi, c := bits.Add64(hi, cross, 0)
	return Uint128{hi, lo}, over == 0 && c == 0
}

// Mul returns a * b, and false when the product does not fit in 128 bits.
func (a Uint128) Mul(b Uint128) (Uint128, bool) {
	switch {
	case a.Hi != 0 && b.Hi != 0:
		return Uint128{}, false
	case a.Hi == 0:
		return b.MulU64(a.Lo)
	}
	return a.MulU64(b.Lo)
}

// DivRem returns the quotient and remainder of a / b, and false when b is 0.
func (a Uint128) DivRem(b Uint128) (q, r Uint128, ok bool) {
	if b.IsZero() {
		return Uint128{}, Uint128{}, false
	}
	if b.Hi == 0 {
		var rem uint64
		q.Hi, rem = bits.Div64(0, a.Hi, b.Lo)
		q.Lo, rem = bits.Div64(rem, a.Lo, b.Lo)
		return q, Uint128{0, rem}, true
	}
	// shift and subtract: the divisor has a high word, so the quotient is below
	// 2^64. The running remainder is a prefix of a, at most 127 bits before the
	// shift of the last step, so the shift cannot carry out of 128 bits.
	for i := 127; i >= 0; i-- {
		r = Uint128{r.Hi<<1 | r.Lo>>63, r.Lo<<1 | a.bit(i)}
		if r.Cmp(b) >= 0 {
			r, _ = r.Sub(b)
			q = q.setBit(i) // i < 64 whenever the divisor has a high word
		}
	}
	return q, r, true
}

// bit returns bit i of a, 0 or 1.
func (a Uint128) bit(i int) uint64 {
	if i >= 64 {
		return a.Hi >> (i - 64) & 1
	}
	return a.Lo >> i & 1
}

// setBit returns a with bit i of the low word set; the quotient of the
// shift-and-subtract path is below 2^64, so the high word is never set.
func (a Uint128) setBit(i int) Uint128 {
	a.Lo |= 1 << i
	return a
}

// Int128 is a signed integer of magnitude below 2^127, held as a sign and a
// magnitude so that no sum or product of two int64 values can wrap. The zero
// value is 0.
type Int128 struct {
	neg bool
	mag Uint128
}

// maxMag is the largest magnitude of an Int128, 2^127 - 1.
var maxMag = Uint128{1<<63 - 1, 1<<64 - 1}

// FromInt64 returns v as an Int128.
func FromInt64(v int64) Int128 {
	if v < 0 {
		return Int128{true, Uint128{0, uint64(^v) + 1}}
	}
	return Int128{false, Uint128{0, uint64(v)}}
}

// FromUint128 returns u as an Int128, and false when u is above 2^127 - 1.
func FromUint128(u Uint128) (Int128, bool) {
	return Int128{false, u}, u.Cmp(maxMag) <= 0
}

// signed builds a value with the sign normalised: zero is never negative.
func signed(neg bool, mag Uint128) Int128 {
	return Int128{neg && !mag.IsZero(), mag}
}

// MulInt64 returns the exact product a * b, which always fits (below 2^126).
func MulInt64(a, b int64) Int128 {
	x, y := FromInt64(a), FromInt64(b)
	return signed(x.neg != y.neg, Mul64(x.mag.Lo, y.mag.Lo))
}

// Sign returns -1, 0 or 1.
func (v Int128) Sign() int {
	switch {
	case v.mag.IsZero():
		return 0
	case v.neg:
		return -1
	}
	return 1
}

// Mag returns the magnitude |v|.
func (v Int128) Mag() Uint128 { return v.mag }

// Neg returns -v.
func (v Int128) Neg() Int128 { return signed(!v.neg, v.mag) }

// Cmp returns -1, 0 or 1 as v is below, equal to or above w.
func (v Int128) Cmp(w Int128) int {
	switch {
	case v.Sign() != w.Sign():
		if v.Sign() < w.Sign() {
			return -1
		}
		return 1
	case v.neg:
		return w.mag.Cmp(v.mag)
	}
	return v.mag.Cmp(w.mag)
}

// Add returns v + w, and false when the sum has a magnitude of 2^127 or more.
func (v Int128) Add(w Int128) (Int128, bool) {
	if v.neg == w.neg {
		m, _ := v.mag.Add(w.mag) // two magnitudes below 2^127 sum below 2^128
		return signed(v.neg, m), m.Cmp(maxMag) <= 0
	}
	if v.mag.Cmp(w.mag) >= 0 {
		m, _ := v.mag.Sub(w.mag)
		return signed(v.neg, m), true
	}
	m, _ := w.mag.Sub(v.mag)
	return signed(w.neg, m), true
}

// Sub returns v - w, and false when the difference has a magnitude of 2^127 or more.
func (v Int128) Sub(w Int128) (Int128, bool) { return v.Add(w.Neg()) }
