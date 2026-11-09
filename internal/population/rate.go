package population

import (
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/curve"
)

// MaxRate is the largest annual rate: every household once a year, in basis
// points. It is the largest probability per year, not a tunable
// (spec-population.md 8).
const MaxRate = 10_000

// Knot is a point of a rate schedule: a welfare ratio and the annual rate at it,
// both in basis points (spec-population.md 3.5).
type Knot struct{ Ratio, Rate int64 }

// stackKnots is the length of the knot list converted on the stack; a longer
// list (the validator bounds a schedule to MaxKnots) allocates.
const stackKnots = 16

// eval returns the schedule at ratio through the one piecewise-linear
// evaluator of internal/curve.
func eval(knots []Knot, ratio int64) (int64, bool) {
	var buf [stackKnots]curve.Knot
	ks := buf[:0]
	for _, k := range knots {
		ks = append(ks, curve.Knot{X: k.Ratio, Y: k.Rate})
	}
	return curve.Eval(ks, ratio)
}

// Rate returns the effective annual rate of a household at the given welfare
// ratio: min(10 000, trunc(schedule(ratio) * scaleBP / 10 000)), with the
// schedule flat outside its first and last knot and the increment between two
// knots truncated toward zero (spec-population.md 3.5.1, 3.6). clamped says the
// 10 000 cap acted. ok is false for an empty schedule, a negative scale or
// result, or an overflow.
func Rate(knots []Knot, ratio, scaleBP int64) (rate int64, clamped, ok bool) {
	base, ok := eval(knots, ratio)
	if !ok || base < 0 || scaleBP < 0 {
		return 0, false, false
	}
	scaled, ok := checked.MulDiv(base, scaleBP, 10_000)
	if !ok {
		return 0, false, false
	}
	if scaled > MaxRate {
		return MaxRate, true, true
	}
	return scaled, false, true
}
