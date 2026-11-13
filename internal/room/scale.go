package room

import "github.com/Whealinghold/economy-sim/internal/checked"

// Scale returns f, in basis points, by which the default strategy scales its
// targets to the room (spec-storage.md 6.2.2, FR-STO-022). targets holds, per
// commodity, the target quantity and the size of a unit. Per dimension,
//
//	Avail = max(0, room - pledge - upkeep)     what the targets may take
//	TL    = sum of target * size                the load of the targets
//
// and f is the least floor(Avail * D / TL) over the dimensions where TL > Avail,
// D if no dimension binds (so f is 0 where Avail is 0). A load that overflows
// int64 gives 0, the safe side. The caller applies ScaleQty to every target of
// the commodity set, pro rata.
func Scale(targets []Item, room, pledge, upkeep Pair) int64 {
	f := Denominator
	for _, d := range [2]func(Pair) int64{pairMass, pairBulk} {
		avail := availOf(d(room), d(pledge), d(upkeep))
		var tl sum
		for _, t := range targets {
			tl.addMul(t.Qty, d(t.Size))
		}
		if tl.bad {
			return 0
		}
		if tl.v <= avail {
			continue
		}
		v, ok := checked.MulDiv(avail, Denominator, tl.v)
		if !ok {
			return 0
		}
		f = min(f, v)
	}
	return f
}

// availOf is room less the pledge and the upkeep room, 0 where it would be
// negative: after a cut of the room the targets may take nothing (6.2.3).
func availOf(room, pledge, upkeep int64) int64 {
	v, ok := checked.Sub(room, pledge)
	if !ok {
		return 0
	}
	if v, ok = checked.Sub(v, upkeep); !ok {
		return 0
	}
	return max(v, 0)
}

// ScaleQty returns floor(qty * f / D), a target scaled by f. f is at most D, so
// the result fits whenever qty does; a negative qty or f is 0.
func ScaleQty(qty, f int64) int64 {
	if qty < 0 || f < 0 {
		return 0
	}
	v, ok := checked.MulDiv(qty, f, Denominator)
	if !ok {
		return 0
	}
	return v
}
