package fiscal

import (
	"math/bits"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// dutyDen is 10^8: the basis-point denominator twice, the unit of a duty's carry
// (spec-fiscal.md 6.3). A rule constant, not a setting.
const dutyDen = 100_000_000

// Base is the sum of the values of the trades of one (duty, payer) in one tick,
// held in 128 bits so that the sum of any number of int64 values cannot wrap
// (spec-fiscal.md 6.2.2, 11.2). The zero value is an empty base.
type Base struct{ hi, lo uint64 }

// Add adds the value of one trade, price times quantity, already checked into
// int64 by the clearing. ok is false for a negative value, which is left out.
func (b *Base) Add(v int64) (ok bool) {
	if v < 0 {
		return false
	}
	lo, c := bits.Add64(b.lo, uint64(v), 0)
	b.hi += c // the sum of 2^64 values below 2^63 cannot carry out of 128 bits
	b.lo = lo
	return true
}

// IsZero reports whether the base is empty.
func (b Base) IsZero() bool { return b.hi == 0 && b.lo == 0 }

// DutyDue returns the due of a duty on the base and the carry that remains
// (spec-fiscal.md 6.3):
//
//	num = base*rateBP*scale + carry, due = floor(num / 10^8), carry = num mod 10^8
//
// with a 192-bit intermediate. The due is at most maxMoney; a quotient above it
// is cut to maxMoney and the excess is neither applied nor carried, as
// internal/moneyflow does. The carry advances by the due, not by what the payer
// could pay. ok is false for a negative argument, a rate times scale that leaves
// int64, or a carry of 10^8 or more.
func DutyDue(b Base, rateBP, scale, carry, maxMoney int64) (due, newCarry int64, ok bool) {
	rate, rok := checked.Mul(rateBP, scale)
	if !rok || rateBP < 0 || scale < 0 || carry < 0 || carry >= dutyDen || maxMoney < 0 {
		return 0, 0, false
	}
	// [top mid low] = base * rate + carry, 192 bits.
	h0, low := bits.Mul64(b.lo, uint64(rate))
	h1, m1 := bits.Mul64(b.hi, uint64(rate))
	mid, c := bits.Add64(h0, m1, 0)
	top := h1 + c
	low, c = bits.Add64(low, uint64(carry), 0)
	mid, c = bits.Add64(mid, 0, c)
	top += c
	// Divide by 10^8, three words at a time; each remainder is below the divisor,
	// as bits.Div64 needs.
	qTop, r := top/dutyDen, top%dutyDen
	qMid, r := bits.Div64(r, mid, dutyDen)
	qLow, r := bits.Div64(r, low, dutyDen)
	newCarry = int64(r)
	if qTop != 0 || qMid != 0 || qLow > uint64(maxMoney) {
		return maxMoney, newCarry, true
	}
	return int64(qLow), newCarry, true
}
