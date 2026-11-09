package transit

import (
	"math"
	"math/bits"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Labour returns the carrier labour of a lane in household-ticks per 10 000
// units of a commodity (spec-transport.md 9.4):
//
//	f      = max(mass / unitMass, bulk / unitBulk)   the binding fill of one unit
//	s      = ceil(distance / span)
//	labour = ceil(10 000 * f * s * duration / output)
//
// mass and bulk are those of one unit of the commodity. f is a rational: the
// two terms are compared by cross-multiplication in 128 bits and a term whose
// unit capacity is 0 is left out. The numerator is held in 128 bits and the
// division is exact, so the result is the ceiling of the true value. ok is false
// when an argument is negative, both capacities are 0, span or output is not
// positive, or the result does not fit in int64.
func Labour(mass, bulk, unitMass, unitBulk, span, distance, duration, output int64) (labour int64, ok bool) {
	if duration < 0 || output <= 0 || !valid(mass, bulk, unitMass, unitBulk) {
		return 0, false
	}
	s, ok := HaulPerUnit(distance, span)
	if !ok {
		return 0, false
	}
	fn, fd := binding(mass, bulk, unitMass, unitBulk)
	k, kok := checked.Mul(s, duration)
	if kok {
		k, kok = checked.Mul(k, 10_000)
	}
	den, dok := checked.Mul(fd, output)
	if !kok || !dok {
		return 0, false
	}
	return ceilDiv128(fn, k, den)
}

// binding returns the numerator and denominator of the binding fill f =
// max(mass / unitMass, bulk / unitBulk): the two fractions compared by
// cross-multiplication in 128 bits, a term with a unit capacity of 0 left out.
// The arguments are valid, so the denominator is above 0.
func binding(mass, bulk, unitMass, unitBulk int64) (fn, fd int64) {
	if unitMass == 0 || (unitBulk > 0 && mulCmp(uint64(bulk), uint64(unitMass), uint64(mass), uint64(unitBulk)) > 0) {
		return bulk, unitBulk
	}
	return mass, unitMass
}

// ceilDiv128 returns ceil(a * b / d) for a, b >= 0 and d > 0 with the product
// held in 128 bits; ok is false when the quotient does not fit in int64.
func ceilDiv128(a, b, d int64) (q int64, ok bool) {
	hi, lo := bits.Mul64(uint64(a), uint64(b))
	if hi >= uint64(d) {
		return 0, false
	}
	uq, r := bits.Div64(hi, lo, uint64(d))
	if r != 0 {
		if uq >= math.MaxInt64 {
			return 0, false
		}
		uq++
	}
	if uq > math.MaxInt64 {
		return 0, false
	}
	return int64(uq), true
}
