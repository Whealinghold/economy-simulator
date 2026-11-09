package measure

import (
	"errors"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// MaxIndexBP is the largest index, ratio or growth in basis points: an index at
// or above it is clamped to it and flagged, never an error
// (spec-indicators.md 5.3.7).
const MaxIndexBP = 1<<31 - 1

// bpScale is the number of basis points in 1.0.
const bpScale = 10_000

var (
	// ErrOverflow reports a sum, product or conversion that does not fit.
	ErrOverflow = errors.New("measure: overflow")
	// ErrNegative reports a negative total where the measure needs none.
	ErrNegative = errors.New("measure: negative value")
	// ErrRange reports a weight below 1 or a quantile outside [0, 10 000].
	ErrRange = errors.New("measure: value out of range")
)

// maxTimesBP is floor(2^127 / 10 000), the largest magnitude that can be
// multiplied by 10 000 and stay below 2^127 (spec-indicators.md 5.2.1).
var maxTimesBP = Uint128{0x346dc5d638865, 0x94af4f0d844d013a}

// MulBP returns v * 10 000, and false when |v| is above floor(2^127 / 10 000):
// the guard that stands before every ratio, share and Gini numerator.
func MulBP(v Int128) (Int128, bool) {
	m, ok := MulBPU(v.mag)
	return signed(v.neg, m), ok
}

// MulBPU is MulBP for an unsigned value.
func MulBPU(u Uint128) (Uint128, bool) {
	if u.Cmp(maxTimesBP) > 0 {
		return Uint128{}, false
	}
	return u.MulU64(bpScale)
}

// ToInt64 returns v as an int64, and 0, false when it does not fit.
func ToInt64(v Int128) (int64, bool) {
	switch {
	case v.mag.Hi != 0:
		return 0, false
	case v.neg && v.mag.Lo <= 1<<63:
		return int64(-v.mag.Lo), true
	case !v.neg && v.mag.Lo <= 1<<63-1:
		return int64(v.mag.Lo), true
	}
	return 0, false
}

// ClampBP returns v limited to [-MaxIndexBP, MaxIndexBP] and whether it was cut.
func ClampBP(v Int128) (int64, bool) {
	if v.mag.Hi != 0 || v.mag.Lo >= MaxIndexBP {
		if v.neg {
			return -MaxIndexBP, true
		}
		return MaxIndexBP, true
	}
	n, _ := ToInt64(v)
	return n, false
}

// PV accumulates a sum of price times volume, for the volume-weighted price of
// a window (spec-indicators.md 4.3.2). It wraps checked.Acc, the one shared
// helper, so there is no second copy of the rule. The zero value is empty.
type PV struct{ acc checked.Acc }

// Add adds price * volume, and false when either is negative or the sum would
// pass 128 bits; the sum is then unchanged.
func (p *PV) Add(price, volume int64) bool { return p.acc.AddMul(price, volume) }

// Div returns the sum divided by volume, half to even, and false when volume is
// below 1 or the quotient does not fit.
func (p PV) Div(volume int64) (int64, bool) { return p.acc.DivHalfEven(volume) }
