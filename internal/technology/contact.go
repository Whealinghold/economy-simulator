package technology

import (
	"math/bits"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// DistanceFunc returns the shortest distance from one region to another over
// the effective graph, as an index into the caller's regions, and ok false
// when there is no path (spec-technology.md 4.5.1). The caller's graph owns
// closed and slowed edges; this package never sees them.
type DistanceFunc func(from, to int) (d int64, ok bool)

// Weight returns the contact weight in basis points from a region at distance
// d, with reach the field's reach (spec-technology.md 4.5.1, FR-TEC-014):
// (reach-d)*10 000/reach for 1 <= d < reach, truncating, and 0 for d >= reach,
// when there is no path (ok false), for a reach below 1 and for a d below 1
// (two distinct regions are at least 1 apart, so a smaller d is no contact).
// The product is held in 128 bits and the quotient is below 10 000, so it
// never overflows.
func Weight(reach, d int64, ok bool) int64 {
	if !ok || reach < 1 || d < 1 || d >= reach {
		return 0
	}
	hi, lo := bits.Mul64(uint64(reach-d), uint64(Basis))
	w, _ := bits.Div64(hi, lo, uint64(reach))
	return int64(w)
}

// Frontier returns F(r) = max over every region s other than r of
// gains[s] * Weight(reach, dist(s, r)) / 10 000 (spec-technology.md 4.5.2),
// from a snapshot of gains of one field taken once for the step. It is 0 when
// no region is in contact. The product is held in 128 bits and the weight is
// below 10 000 whenever it is not 0, so the quotient is below the gain and
// always fits: there is no overflow to report.
func Frontier(r int, gains []int64, reach int64, dist DistanceFunc) int64 {
	var best int64
	for s, g := range gains {
		if s == r || g <= 0 {
			continue
		}
		d, has := dist(s, r)
		w := Weight(reach, d, has)
		if w == 0 {
			continue
		}
		hi, lo := bits.Mul64(uint64(g), uint64(w))
		q, _ := bits.Div64(hi, lo, uint64(Basis))
		best = max(best, int64(q))
	}
	return best
}

// CatchUp applies the catch-up of spec-technology.md 4.5.4 for one cell:
// when gap is above 0, carry+gap*catchUpBP is split by 10 000*ticksPerYear
// into the add and the next carry; a gap of 0 or below adds nothing and keeps
// the carry (FR-TEC-015). ok is false for a carry or catchUpBP below 0, a
// ticksPerYear below 1, or a product that overflows.
func CatchUp(carry, gap, catchUpBP, ticksPerYear int64) (add, carry2 int64, ok bool) {
	if carry < 0 || catchUpBP < 0 || ticksPerYear < 1 {
		return 0, 0, false
	}
	if gap <= 0 {
		return 0, carry, true
	}
	denom, ok := checked.Mul(Basis, ticksPerYear)
	if !ok {
		return 0, 0, false
	}
	prod, ok := checked.Mul(gap, catchUpBP)
	if !ok {
		return 0, 0, false
	}
	sum, ok := checked.Add(carry, prod)
	if !ok {
		return 0, 0, false
	}
	return sum / denom, sum % denom, true
}
