package checked

import "math/bits"

// Acc accumulates a sum of non-negative products in 128 bits, for the
// volume-weighted mean of CLR 5.4. The zero value is an empty sum. It exists so
// no second copy of the 128-bit helper appears (N-019).
type Acc struct{ hi, lo uint64 }

// AddMul adds a*b to the sum and reports whether it fit. It returns false and
// leaves the sum unchanged when a or b is negative or the sum would exceed
// 128 bits.
func (s *Acc) AddMul(a, b int64) bool {
	if a < 0 || b < 0 {
		return false
	}
	hi, lo := bits.Mul64(uint64(a), uint64(b))
	lo, c := bits.Add64(s.lo, lo, 0)
	hi, c = bits.Add64(s.hi, hi, c)
	if c != 0 {
		return false
	}
	s.hi, s.lo = hi, lo
	return true
}

// DivHalfEven returns the sum divided by c, rounded to the nearest integer with
// a tie going to the even quotient. ok is false when c is below 1 or the
// quotient does not fit in int64.
func (s *Acc) DivHalfEven(c int64) (int64, bool) {
	if c < 1 {
		return 0, false
	}
	uc := uint64(c)
	if s.hi >= uc {
		return 0, false // the quotient needs 64 bits or more
	}
	q, r := bits.Div64(s.hi, s.lo, uc)
	rest := uc - r
	if r > rest || (r == rest && q&1 == 1) {
		if q == 1<<64-1 {
			return 0, false
		}
		q++
	}
	if q > 1<<63-1 {
		return 0, false
	}
	return int64(q), true
}
