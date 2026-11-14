package tenure

import "github.com/Whealinghold/economy-sim/internal/checked"

// BasisPoints is the scale of the posted rent's slot (spec-land.md 5.2.2):
// 10 000 basis points are 1.
const BasisPoints = 10_000

// RentInForce is the rent per unit let in a tick: posted * scale / 10 000,
// truncated toward zero, and at least 1 when posted and scale are both positive
// (spec-land.md 5.2.2). A scale of 0 makes the rent 0. ok is false for a
// negative number or a result beyond int64, with 0 returned.
func RentInForce(posted, scale int64) (rent int64, ok bool) {
	if posted < 0 || scale < 0 {
		return 0, false
	}
	if posted == 0 || scale == 0 {
		return 0, true
	}
	rent, ok = checked.MulDiv(posted, scale, BasisPoints)
	if !ok {
		return 0, false
	}
	return max(rent, 1), true
}

// LeastRent is the least rent a job of executions executions of draw units each
// could owe: r times the units left after its own entitlement and the whole
// commons entitlement, max(0, executions * draw - own - commons), which is 0
// when its own entitlement covers the draw (spec-land.md 4.7.2). It is the
// bound the formation of a hire checks. ok is false for a negative number or a
// product beyond int64.
func LeastRent(executions, draw, own, commons, rent int64) (least int64, ok bool) {
	if executions < 0 || draw < 0 || own < 0 || commons < 0 || rent < 0 {
		return 0, false
	}
	units, ok := checked.Mul(executions, draw)
	if !ok {
		return 0, false
	}
	units = max(0, units-own-commons) // the two sums cannot go below -2 * MaxInt64: both are at least 0
	return checked.Mul(units, rent)
}
