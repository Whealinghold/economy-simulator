package production

import "github.com/Whealinghold/economy-sim/internal/checked"

// Term is a commodity index and a quantity.
type Term struct {
	Commodity int
	Quantity  int64
}

// Wear is a tool: commodity index, wear per use and full durability.
type Wear struct {
	Commodity  int
	WearPerUse int64
	Durability int64
}

// WearCost returns ceil(WearPerUse * price / Durability), the wear of one
// execution on a tool of the given price (PRO 9.1). ok is false when
// Durability is not positive, a factor is negative or the result overflows.
func WearCost(w Wear, price int64) (cost int64, ok bool) {
	if w.Durability <= 0 || w.WearPerUse < 0 || price < 0 {
		return 0, false
	}
	return checked.MulDivCeil(w.WearPerUse, price, w.Durability)
}

// UnitCost returns the cost of one unit of an output, in minor units, rounded
// up, or 0 if unknown (PRO 9.1.1):
//
//	ceil((sum q*price + sum WearCost + duration*wage + rent) * share / (outQty * yieldBP))
//
// prices is indexed by commodity index; 0 means unknown. It is 0 when any input
// or tool price is 0 (or the commodity is outside prices), the wage is 0 or
// less, yieldBP or share is 0 or less, or any step overflows. No side effect,
// no allocation.
func UnitCost(inputs []Term, tools []Wear, prices []int64, share, outQty, yieldBP, duration, wage, rent int64) int64 {
	if wage <= 0 || share <= 0 || yieldBP <= 0 || outQty <= 0 || duration < 0 || rent < 0 {
		return 0
	}
	var cost int64
	var ok bool
	for _, in := range inputs {
		if in.Commodity < 0 || in.Commodity >= len(prices) || prices[in.Commodity] <= 0 || in.Quantity < 0 {
			return 0
		}
		var v int64
		if v, ok = checked.Mul(in.Quantity, prices[in.Commodity]); !ok {
			return 0
		}
		if cost, ok = checked.Add(cost, v); !ok {
			return 0
		}
	}
	for _, t := range tools {
		if t.Commodity < 0 || t.Commodity >= len(prices) || prices[t.Commodity] <= 0 {
			return 0
		}
		var v int64
		if v, ok = WearCost(t, prices[t.Commodity]); !ok {
			return 0
		}
		if cost, ok = checked.Add(cost, v); !ok {
			return 0
		}
	}
	labour, ok := checked.Mul(duration, wage)
	if !ok {
		return 0
	}
	if cost, ok = checked.Add(cost, labour); !ok {
		return 0
	}
	if cost, ok = checked.Add(cost, rent); !ok {
		return 0
	}
	den, ok := checked.Mul(outQty, yieldBP)
	if !ok {
		return 0
	}
	unit, ok := checked.MulDivCeil(cost, share, den)
	if !ok {
		return 0
	}
	return unit
}

// Out is one output of a recipe for the anchor: its commodity, quantity per
// execution, cost share in basis points and whether it is discarded.
type Out struct {
	Commodity int
	Quantity  int64
	ShareBP   int64
	Discard   bool
}

// Recipe is the per-recipe argument of Anchors, filled by the caller. YieldBP
// is the effective factor of PRO 6.3 for the region and season, and Runnable
// says some agent of the region has the recipe on its role's list.
type Recipe struct {
	Inputs   []Term
	Tools    []Wear
	Outputs  []Out
	Duration int64
	Rent     int64
	YieldBP  int64
	Runnable bool
}

// Anchors writes into row, indexed by commodity, the cost anchor of every
// commodity in one region (PRO 9.3): the least non-zero UnitCost over the
// recipes in makers[c] that are runnable, have a yield above 0 and output c
// with a share above 0 (a byproduct or a discard never sets an anchor); 0 when
// none. makers[c] lists the indexes (any int type, so the root passes its
// typed IDs in place) into recipes of the recipes that make
// commodity c, ascending. A wage of 0 or less gives 0 everywhere. A commodity
// beyond makers, or a maker index outside recipes, gets no anchor from it.
func Anchors[I ~int](row []int64, recipes []Recipe, makers [][]I, prices []int64, wage int64) {
	clear(row)
	if wage <= 0 {
		return
	}
	for c := range min(len(row), len(makers)) {
		for _, m := range makers[c] {
			ri := int(m)
			if ri < 0 || ri >= len(recipes) {
				continue
			}
			r := &recipes[ri]
			if !r.Runnable || r.YieldBP <= 0 {
				continue
			}
			for _, o := range r.Outputs {
				if o.Commodity != c || o.Discard {
					continue
				}
				u := UnitCost(r.Inputs, r.Tools, prices, o.ShareBP, o.Quantity, r.YieldBP, r.Duration, wage, r.Rent)
				if u > 0 && (row[c] == 0 || u < row[c]) {
					row[c] = u
				}
			}
		}
	}
}
