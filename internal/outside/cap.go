package outside

import "github.com/Whealinghold/economy-sim/internal/checked"

// The causes that end a side (spec-external-sector.md 7.3). The root names them
// in the event.
const (
	// EndUnits means the remaining units reached 0.
	EndUnits = 1
	// EndValue means the remaining list value fell below the limit price.
	EndValue = 2
)

// Cap is what is left of the cumulative caps of one side: the units and the list
// value it may still trade over the run (spec-external-sector.md 7.3). A cap that
// is not set has its flag false and its remainder is not read.
type Cap struct {
	// UnitsLeft is the remaining units, read when UnitCap is true.
	UnitsLeft int64
	// ValueLeft is the remaining list value, read when ValueCap is true.
	ValueLeft int64
	// UnitCap and ValueCap say which caps are set.
	UnitCap, ValueCap bool
}

// Offer returns the units a side offers at a limit price and the units the caps
// cut (spec-external-sector.md 7.3):
//
//	offered = max(0, min(wanted, UnitsLeft, floor(ValueLeft / limit)))
//
// over the caps that are set; cut = wanted - offered. A limit below 1 offers
// nothing and cuts the whole wanted quantity, and a negative wanted quantity
// offers and cuts nothing.
func Offer(c Cap, wanted, limit int64) (offered, cut int64) {
	if wanted < 1 {
		return 0, 0
	}
	if limit < 1 {
		return 0, wanted
	}
	offered = wanted
	if c.UnitCap {
		offered = min(offered, c.UnitsLeft)
	}
	if c.ValueCap {
		offered = min(offered, c.ValueLeft/limit)
	}
	offered = max(offered, 0)
	return offered, wanted - offered
}

// Settle returns the cap after units traded at a limit price, and the list value
// of the trade, the units times the limit (spec-external-sector.md 7.4), not the
// money that changed hands. ok is false for a negative quantity or limit, a list
// value that does not fit in int64, and a trade the cap did not allow (more
// units than are left, or a list value above what is left); the cap is then
// returned unchanged.
func (c Cap) Settle(units, limit int64) (next Cap, listValue int64, ok bool) {
	if units < 0 || limit < 0 {
		return c, 0, false
	}
	listValue, ok = checked.Mul(units, limit)
	if !ok || (c.UnitCap && units > c.UnitsLeft) || (c.ValueCap && listValue > c.ValueLeft) {
		return c, 0, false
	}
	next = c
	if c.UnitCap {
		next.UnitsLeft -= units
	}
	if c.ValueCap {
		next.ValueLeft -= listValue
	}
	return next, listValue, true
}

// Ended reports whether a side ends after settlement, and why: EndUnits when
// the remaining units are 0, else EndValue when the remaining list value is below
// the limit price of the tick, which no unit can then be bought with
// (spec-external-sector.md 7.3). With both true the units are the cause. A side
// with no cap set never ends.
func (c Cap) Ended(limit int64) (cause int, ended bool) {
	switch {
	case c.UnitCap && c.UnitsLeft <= 0:
		return EndUnits, true
	case c.ValueCap && c.ValueLeft < limit:
		return EndValue, true
	}
	return 0, false
}
