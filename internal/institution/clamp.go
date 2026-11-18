package institution

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Clamp returns the limit price an order is admitted with under a legal range
// (spec-institutions.md 6.3.1, FR-ORG-030): a bid whose limit is above the cap is
// replaced by the cap, an ask whose limit is below the floor is replaced by the
// floor, and an order inside the range is unchanged. A cap or floor of 0 is none.
// changed is true when the limit was replaced. The quantity, the order's place in a
// book and its key are not touched (FR-ORG-032).
func Clamp(limit int64, bid bool, capPrice, floorPrice int64) (admitted int64, changed bool) {
	if bid && capPrice > 0 && limit > capPrice {
		return capPrice, true
	}
	if !bid && floorPrice > 0 && limit < floorPrice {
		return floorPrice, true
	}
	return limit, false
}

// Tally counts the clamps of one control in one region and one tick: the orders
// lowered to a cap, the orders raised to a floor, and their units (the count that
// `PriceControlled` tells, spec-institutions.md 11.1).
type Tally struct {
	Lowered, Raised int64
	Units           int64
}

// Add counts one clamp of an order of qty units; an order that was not changed
// counts nothing. It reports whether anything was counted. A quantity that would
// overflow the unit count saturates it at the largest int64, so the count never
// wraps.
func (t *Tally) Add(bid, changed bool, qty int64) bool {
	if !changed {
		return false
	}
	if bid {
		t.Lowered++
	} else {
		t.Raised++
	}
	units, ok := checked.Add(t.Units, max(qty, 0))
	if !ok {
		units = math.MaxInt64
	}
	t.Units = units
	return true
}
