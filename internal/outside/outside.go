package outside

import "github.com/Whealinghold/economy-sim/internal/checked"

// bp is ten thousand: one in basis points.
const bp = 10_000

// scaleProduct is 10^12, the divisor of the single rounding of the mid
// (spec-external-sector.md 6.1): the season and the shock are each in basis
// points, the factor is in basis points, and the base is a price.
const scaleProduct = 1_000_000_000_000

// MidIn is the input of Mid. The first group is the line, the second the tick
// and the state of the line.
type MidIn struct {
	// Base is the mid at tick 0, at least 1.
	Base int64
	// Season is the seasonal factor in basis points, 10 000 for none.
	Season int64
	// TrendBP is the signed trend in basis points of the base a year.
	TrendBP int64
	// FloorBP and CeilBP bound the factor of the trend and the walk, in
	// basis points.
	FloorBP, CeilBP int64
	// MaxPrice bounds the mid, at least 1.
	MaxPrice int64

	// Tick is the tick, at least 0, and TicksPerYear the ticks in a year, at
	// least 1.
	Tick, TicksPerYear int64
	// Drift is the state of the walk in basis points, above -10 000.
	Drift int64
	// Shock is the shock slot in basis points, 10 000 by default.
	Shock int64
}

// Mid returns the mid of a line at a tick (spec-external-sector.md 6.1):
//
//	F_trend = max(0, 10 000 + trunc(TrendBP * Tick / TicksPerYear))
//	F_walk  = 10 000 + Drift
//	F       = clamp(round_half_even(F_trend * F_walk / 10 000), FloorBP, CeilBP)
//	Mid     = clamp(round_half_even(Base * Season * F * Shock / 10^12), 1, MaxPrice)
//
// The rounding of the factor and the single rounding of the mid are half to
// even (the rule of the clearing); the trend truncates toward zero. clamped
// reports that a clamp changed the factor or the mid. ok is false for an input
// outside its domain (a base or a price bound below 1, a negative season, shock,
// tick or floor, a ceiling below the floor, fewer than one tick a year, a drift
// of -10 000 or less) and for a product that does not fit in int64; the
// validated bounds of the module never reach either.
func Mid(m MidIn) (mid int64, clamped, ok bool) {
	if !m.valid() {
		return 0, false, false
	}
	raw, ok := m.rawFactor()
	if !ok {
		return 0, false, false
	}
	f := min(max(raw, m.FloorBP), m.CeilBP)
	baseSeason, ok := checked.Mul(m.Base, m.Season)
	if !ok {
		return 0, false, false
	}
	factorShock, ok := checked.Mul(f, m.Shock)
	if !ok {
		return 0, false, false
	}
	q, ok := checked.MulDivHalfEven(baseSeason, factorShock, scaleProduct)
	if !ok {
		return 0, false, false
	}
	mid = min(max(q, 1), m.MaxPrice)
	return mid, f != raw || mid != q, true
}

// valid reports whether the input is inside the domain of Mid.
func (m MidIn) valid() bool {
	positive := m.Base >= 1 && m.MaxPrice >= 1 && m.TicksPerYear >= 1
	signs := m.Season >= 0 && m.Shock >= 0 && m.Tick >= 0 && m.FloorBP >= 0
	return positive && signs && m.CeilBP >= m.FloorBP && m.Drift > -bp
}

// rawFactor returns the factor of the trend and the walk before the clamp,
// round_half_even(F_trend * F_walk / 10 000).
func (m MidIn) rawFactor() (int64, bool) {
	trend, ok := checked.MulDiv(m.TrendBP, m.Tick, m.TicksPerYear)
	if !ok {
		return 0, false
	}
	fTrend, ok := checked.Add(bp, trend)
	if !ok {
		return 0, false
	}
	fWalk, ok := checked.Add(bp, m.Drift)
	if !ok {
		return 0, false
	}
	return checked.MulDivHalfEven(max(fTrend, 0), fWalk, bp)
}

// Walk returns the next drift of a line (spec-external-sector.md 6.4):
//
//	drift' = clamp(drift - trunc(drift * revertBP / 10 000) + d, -bound, bound)
//
// d is the draw already shifted into [-step, step]. The domain is the validated
// one: 0 <= revertBP <= 10 000 (a larger one is held at 10 000, the whole drift),
// |drift| <= bound and |d| <= bound <= 9 999, so no sum can overflow.
func Walk(drift, revertBP, d, bound int64) int64 {
	revertBP = min(max(revertBP, 0), bp)
	// |drift * revertBP| < 2^63 * 10^4 fits the 128-bit product and the
	// quotient is at most |drift|, so the helper cannot report an overflow.
	undone, _ := checked.MulDiv(drift, revertBP, bp)
	return min(max(drift-undone+d, -bound), bound)
}

// Quote returns the limit price of a side at a gate (spec-external-sector.md 6.6).
// A sell side (an ask) rounds up and a buy side (a bid) rounds down, in the
// margin and again in the gate scale:
//
//	ask = ceil(ceil(mid * (10 000 + marginBP) / 10 000) * scaleBP / 10 000)
//	bid = floor(floor(mid * (10 000 - marginBP) / 10 000) * scaleBP / 10 000)
//
// and each is held at maxPrice at most. The rounding widens the spread against
// the realm, so it never closes it. A bid can be 0: the caller places nothing
// for a price below 1 and counts the units as cut by the price. ok is false for
// a mid or a price bound below 1, a negative margin or scale, a buy margin above
// 10 000, and a product that does not fit in int64.
func Quote(mid, marginBP, scaleBP, maxPrice int64, sell bool) (price int64, ok bool) {
	if mid < 1 || maxPrice < 1 || marginBP < 0 || scaleBP < 0 || (!sell && marginBP > bp) {
		return 0, false
	}
	round, factor := checked.MulDivFloor, bp-marginBP
	if sell {
		round, factor = checked.MulDivCeil, bp+marginBP
	}
	p, ok := round(mid, factor, bp)
	if ok {
		p, ok = round(p, scaleBP, bp)
	}
	if !ok {
		return 0, false
	}
	return min(p, maxPrice), true
}

// Want returns the units a side wants in a tick and the new carry
// (spec-external-sector.md 7.1):
//
//	units = floor((capacity * avail + carry) / 10 000), carry' = the remainder
//
// so a fraction of a unit is never lost: a capacity of 3 at an availability of
// 2 500 wants 0, 1, 1, 1, 0, 1, 1, 1 units. The carry is in [0, 10 000) and avail
// in [0, 10 000]. Outside that domain, or if the product does not fit in int64,
// nothing is wanted and the carry is returned unchanged.
func Want(capacity, avail, carry int64) (units, newCarry int64) {
	if capacity < 0 || avail < 0 || carry < 0 || carry >= bp {
		return 0, carry
	}
	total, ok := checked.Mul(capacity, avail)
	if !ok {
		return 0, carry
	}
	total, ok = checked.Add(total, carry)
	if !ok {
		return 0, carry
	}
	return total / bp, total % bp
}
