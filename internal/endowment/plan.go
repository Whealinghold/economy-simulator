package endowment

import (
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/curve"
)

// Plan is the input of the two stability conditions of spec-resources.md 8.3.1,
// all primitive. RateBPEff is the effective annual rate in basis points,
// RateBP * m / BP times the mean season factor / BP. ShareBP is the plan share
// and ElasticityBP the elasticity of the abundance curve there. RStar, DR and DX
// are the replacement ratio and the slope of birth minus death (rate over
// ratio) at it, which the population module supplies; the conditions apply only
// when it is on.
type Plan struct {
	Shape         Shape
	RateBPEff     int64
	ShareBP       int64
	ElasticityBP  int64
	RStar, DR, DX int64
}

// Elasticity returns eps_bp = sig * slope / a(sig) in basis points: the
// elasticity of the abundance curve at the share, with the slope of the
// segment that contains it (factor over share), rounded half to even. It is 0
// for a curve that is flat there, outside the knots, or whose factor is 0
// (spec-resources.md 8.3.1). A falling segment gives a negative value, which
// Overshoot and UnstablePlan treat as out of range (false). ok is false on overflow.
func Elasticity(knots []curve.Knot, shareBP int64) (eps int64, ok bool) {
	x0, y0, x1, y1, found := curve.Segment(knots, shareBP)
	if !found || y1 == y0 {
		return 0, true
	}
	a, ok := curve.Eval(knots, shareBP)
	if !ok || a <= 0 {
		return 0, ok
	}
	dy, ok := checked.Sub(y1, y0)
	if !ok {
		return 0, false
	}
	dx, ok := checked.Sub(x1, x0)
	if !ok {
		return 0, false
	}
	den, ok := checked.Mul(dx, a)
	if !ok {
		return 0, false
	}
	num, ok := checked.Mul(shareBP, BP)
	if !ok {
		return 0, false
	}
	return checked.MulDivHalfEven(num, dy, den)
}

// Overshoot reports the warning resource_overshoot: a refill, or a logistic
// stock above half its capacity at the plan share, refills so slowly against the
// population's response that the households overshoot (spec-resources.md 8.3.1).
//
//	refill:    RateBPEff * BP * dX * sig < 4 * r* * dR * eps * (BP - sig)
//	logistic:  RateBPEff * (2 * sig - BP)^2 * dX < 4 * r* * dR * eps * (BP - sig),  sig > BP/2
//
// A constant stock, and a logistic one at or below half, never overshoot (the
// second is unstable, UnstablePlan). It is false when an input is negative or a
// product leaves 128 bits on the left; a right side beyond 128 bits is above
// any left.
func Overshoot(p Plan) bool {
	if !p.inRange() {
		return false
	}
	var left, right u128
	var ok bool
	switch {
	case p.Shape == Refill:
		left, ok = product(uint64(p.RateBPEff), BP, uint64(p.DX), uint64(p.ShareBP))
	case p.Shape == Logistic && p.ShareBP > BP/2:
		d := uint64(2*p.ShareBP - BP)
		left, ok = product(uint64(p.RateBPEff), d*d, uint64(p.DX), 1)
	default:
		return false
	}
	if !ok {
		return false
	}
	// 4 * r* * dR * eps * (BP - sig), the factor 4 taken last so no step wraps.
	right, ok = product(uint64(p.RStar), uint64(p.DR), uint64(p.ElasticityBP), uint64(BP-p.ShareBP))
	if ok {
		right, ok = right.mul(4)
	}
	return !ok || left.cmp(right) < 0
}

// UnstablePlan reports the warning resource_unstable_plan: the plan share sits
// where the stock feeds the population's response with nothing to damp it, a
// logistic stock at or below half its capacity, or a constant flow, with an
// elasticity above 0 (spec-resources.md 8.3.1).
func UnstablePlan(p Plan) bool {
	if !p.inRange() || p.ElasticityBP <= 0 {
		return false
	}
	switch p.Shape {
	case Logistic:
		return p.ShareBP <= BP/2
	case Constant:
		return true
	}
	return false
}

// inRange reports whether every input is non-negative and the share is a share.
func (p Plan) inRange() bool {
	return p.RateBPEff >= 0 && p.ElasticityBP >= 0 && p.RStar >= 0 && p.DR >= 0 && p.DX >= 0 &&
		p.ShareBP >= 0 && p.ShareBP <= BP
}
