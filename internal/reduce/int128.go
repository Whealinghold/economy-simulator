// Package reduce holds the integer arithmetic of the experiment harness
// (spec-experiments.md 8.3, 8.3.2): 128-bit sums and products with an overflow
// flag, division rounded half to even, the nearest integer root, the
// nearest-rank quantile and the exact sign test. It uses the standard library
// and math/bits only, no floating point, and imports nothing of the module
// (IND's internal/measure is IND's alone, FR-IND-001).
package reduce

import (
	"math/bits"
	"strconv"
)

// Int128 is a signed 128-bit integer in two's complement. The zero value is 0.
// Every operation that can leave the range [-2^127, 2^127 - 1] reports it with a
// false result and never wraps (architecture.md 3.5).
type Int128 struct{ hi, lo uint64 }

// u128 is an unsigned 128-bit magnitude, the working form of Mul and RDiv.
type u128 struct{ hi, lo uint64 }

// FromInt64 returns v as an Int128.
func FromInt64(v int64) Int128 { return Int128{uint64(v >> 63), uint64(v)} }

// FromWords returns the Int128 whose two's complement words are hi and lo.
func FromWords(hi int64, lo uint64) Int128 { return Int128{uint64(hi), lo} }

// Words returns the two's complement words of a, high first.
func (a Int128) Words() (hi int64, lo uint64) { return int64(a.hi), a.lo }

// IsNeg reports whether a is below 0.
func (a Int128) IsNeg() bool { return a.hi>>63 == 1 }

// Sign returns -1, 0 or 1.
func (a Int128) Sign() int {
	switch {
	case a.IsNeg():
		return -1
	case a.hi == 0 && a.lo == 0:
		return 0
	}
	return 1
}

// Cmp returns -1, 0 or 1 as a is below, equal to or above b.
func (a Int128) Cmp(b Int128) int {
	if a.hi != b.hi {
		if int64(a.hi) < int64(b.hi) {
			return -1
		}
		return 1
	}
	switch {
	case a.lo < b.lo:
		return -1
	case a.lo > b.lo:
		return 1
	}
	return 0
}

// Int64 returns a as an int64, and false when it does not fit.
func (a Int128) Int64() (int64, bool) {
	return int64(a.lo), a.hi == uint64(int64(a.lo)>>63)
}

// Add returns a + b, and false on overflow.
func (a Int128) Add(b Int128) (Int128, bool) {
	lo, c := bits.Add64(a.lo, b.lo, 0)
	hi, _ := bits.Add64(a.hi, b.hi, c)
	r := Int128{hi, lo}
	return r, a.IsNeg() != b.IsNeg() || r.IsNeg() == a.IsNeg()
}

// Sub returns a - b, and false on overflow.
func (a Int128) Sub(b Int128) (Int128, bool) {
	lo, c := bits.Sub64(a.lo, b.lo, 0)
	hi, _ := bits.Sub64(a.hi, b.hi, c)
	r := Int128{hi, lo}
	return r, a.IsNeg() == b.IsNeg() || r.IsNeg() == a.IsNeg()
}

// Neg returns -a, and false for the minimum, which has no positive twin.
func (a Int128) Neg() (Int128, bool) {
	m, neg := a.mag()
	return fromMag(m, !neg)
}

// Mul returns a * b, and false on overflow.
func (a Int128) Mul(b Int128) (Int128, bool) {
	ma, na := a.mag()
	mb, nb := b.mag()
	p, ok := ma.mul(mb)
	if !ok {
		return Int128{}, false
	}
	return fromMag(p, na != nb)
}

// String returns the decimal text of a, for messages.
func (a Int128) String() string {
	m, neg := a.mag()
	if m.hi == 0 {
		s := strconv.FormatUint(m.lo, 10)
		if neg {
			return "-" + s
		}
		return s
	}
	// Peel 19 decimal digits at a time: 10^19 is the largest power of ten in a word.
	const chunk = 10_000_000_000_000_000_000
	var parts []uint64
	for m.hi != 0 || m.lo != 0 {
		var r uint64
		m, r = m.divWord(chunk)
		parts = append(parts, r)
	}
	s := strconv.FormatUint(parts[len(parts)-1], 10)
	for i := len(parts) - 2; i >= 0; i-- {
		d := strconv.FormatUint(parts[i], 10)
		s += "0000000000000000000"[len(d):] + d
	}
	if neg {
		return "-" + s
	}
	return s
}

// mag returns the magnitude of a and whether a is negative. The magnitude of the
// minimum, 2^127, fits an unsigned word pair.
func (a Int128) mag() (u128, bool) {
	if !a.IsNeg() {
		return u128(a), false
	}
	lo, c := bits.Sub64(0, a.lo, 0)
	hi, _ := bits.Sub64(0, a.hi, c)
	return u128{hi, lo}, true
}

// fromMag builds the Int128 of a magnitude and a sign, and false when it does
// not fit: 2^127 fits only as a negative number.
func fromMag(m u128, neg bool) (Int128, bool) {
	const top = uint64(1) << 63
	if !neg {
		return Int128(m), m.hi < top
	}
	if m.hi > top || (m.hi == top && m.lo != 0) {
		return Int128{}, false
	}
	lo, c := bits.Sub64(0, m.lo, 0)
	hi, _ := bits.Sub64(0, m.hi, c)
	return Int128{hi, lo}, true
}

func (a u128) mul(b u128) (u128, bool) {
	if a.hi != 0 && b.hi != 0 {
		return u128{}, false
	}
	hi, lo := bits.Mul64(a.lo, b.lo)
	h1, l1 := bits.Mul64(a.hi, b.lo)
	h2, l2 := bits.Mul64(a.lo, b.hi)
	if h1 != 0 || h2 != 0 {
		return u128{}, false
	}
	hi, c1 := bits.Add64(hi, l1, 0)
	hi, c2 := bits.Add64(hi, l2, 0)
	return u128{hi, lo}, c1|c2 == 0
}

func (a u128) cmp(b u128) int {
	switch {
	case a.hi != b.hi:
		if a.hi < b.hi {
			return -1
		}
		return 1
	case a.lo < b.lo:
		return -1
	case a.lo > b.lo:
		return 1
	}
	return 0
}

// divWord divides a by a nonzero word d, exactly.
func (a u128) divWord(d uint64) (u128, uint64) {
	qh, r := a.hi/d, a.hi%d
	ql, r := bits.Div64(r, a.lo, d)
	return u128{qh, ql}, r
}

// divmod returns the quotient and remainder of a by a nonzero b. A divisor of
// one word takes two hardware divisions; a wider divisor takes the bit loop,
// whose quotient is below 2^64.
func (a u128) divmod(b u128) (q, r u128) {
	if b.hi == 0 {
		q, rem := a.divWord(b.lo)
		return q, u128{0, rem}
	}
	// r < b <= 2^127 at the top of the loop, so the shift cannot lose a bit.
	for i := 127; i >= 0; i-- {
		r.hi = r.hi<<1 | r.lo>>63
		r.lo = r.lo<<1 | a.bit(i)
		if r.cmp(b) >= 0 {
			lo, c := bits.Sub64(r.lo, b.lo, 0)
			hi, _ := bits.Sub64(r.hi, b.hi, c)
			r = u128{hi, lo}
			q.lo |= 1 << uint(i) // i < 64 here: b >= 2^64 makes the quotient < 2^64
		}
	}
	return q, r
}

func (a u128) bit(i int) uint64 {
	if i >= 64 {
		return a.hi >> uint(i-64) & 1
	}
	return a.lo >> uint(i) & 1
}
