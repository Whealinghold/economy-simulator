package verdict

import "math/bits"

const (
	maxInt64 = 1<<63 - 1
	minInt64 = -1 << 63
)

// bpScale is one whole in basis points.
const bpScale = 10_000

// magnitude returns |x| as a uint64; it is right for the minimum int64.
func magnitude(x int64) uint64 {
	u := uint64(x)
	if x < 0 {
		u = -u
	}
	return u
}

// mulSigned returns the exact 128-bit product of a and b as a sign and a
// magnitude. A zero product is not negative.
func mulSigned(a, b int64) (neg bool, hi, lo uint64) {
	hi, lo = bits.Mul64(magnitude(a), magnitude(b))
	neg = (a < 0) != (b < 0) && (hi|lo) != 0
	return neg, hi, lo
}

// cmpMul compares a*b with c*d exactly and returns -1, 0 or 1. It cannot
// overflow, so a detector never wraps (architecture.md 3.5).
func cmpMul(a, b, c, d int64) int {
	n1, h1, l1 := mulSigned(a, b)
	n2, h2, l2 := mulSigned(c, d)
	if n1 != n2 {
		if n1 {
			return -1
		}
		return 1
	}
	m := 0
	switch {
	case h1 != h2:
		m = cmpU(h1, h2)
	default:
		m = cmpU(l1, l2)
	}
	if n1 {
		return -m
	}
	return m
}

func cmpU(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// mulDivFloor returns floor(a*b/d) for a, b >= 0 and d > 0, and false if an
// argument is out of range or the quotient does not fit an int64.
func mulDivFloor(a, b, d int64) (int64, bool) {
	if a < 0 || b < 0 || d <= 0 {
		return 0, false
	}
	hi, lo := bits.Mul64(uint64(a), uint64(b))
	if hi >= uint64(d) {
		return 0, false
	}
	q, _ := bits.Div64(hi, lo, uint64(d))
	if q > maxInt64 {
		return 0, false
	}
	return int64(q), true
}

// rdiv returns n/d rounded half to even, for n >= 0 and d > 0 (IND 5.2.2).
func rdiv(n, d int64) int64 {
	q, r := n/d, n%d
	if r > d-r || (r == d-r && q%2 == 1) {
		q++
	}
	return q
}

// midUp returns floor((a+b+1)/2) without overflow: the integer midpoint
// of a and b rounded up.
func midUp(a, b int64) int64 {
	return (a >> 1) + (b >> 1) + (((a & 1) + (b & 1) + 1) >> 1)
}

// meanHalfEven returns the mean of a and b rounded half to even, without
// overflow (8.3.4: the median of an even count).
func meanHalfEven(a, b int64) int64 {
	q := (a >> 1) + (b >> 1) + (((a & 1) + (b & 1)) >> 1)
	if (a^b)&1 == 1 { // the exact mean is q + 1/2: pick the even neighbour
		if q%2 != 0 {
			q++
		}
	}
	return q
}
