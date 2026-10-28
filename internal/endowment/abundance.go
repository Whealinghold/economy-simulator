package endowment

import (
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/curve"
)

// Abundance returns the yield factor, in basis points, of a stock
// (spec-resources.md 4.7.1): the curve at share = floor(stock * BP / capacity).
// No knots mean a flat BP, except that an endowment with no capacity can supply
// nothing and has 0 whatever the curve. A stock above the capacity reads as the
// capacity. ok is false for a negative stock or capacity, or an evaluation that
// overflows; the knots are the validator's to bound.
func Abundance(knots []curve.Knot, stock, capacity int64) (factor int64, ok bool) {
	if stock < 0 || capacity < 0 {
		return 0, false
	}
	if capacity == 0 {
		return 0, true
	}
	if len(knots) == 0 {
		return BP, true
	}
	share, ok := checked.MulDiv(min(stock, capacity), BP, capacity)
	if !ok {
		return 0, false
	}
	return curve.Eval(knots, share)
}
