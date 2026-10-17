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

// Basis is the scalar part of a UnitCost: the cost share of the output in basis
// points, its quantity per execution, the effective yield factor in basis
// points, the duration of the job, the wage per unit of duration and the rent.
type Basis struct {
	Share, OutQty, YieldBP, Duration, Wage, Rent int64
}

func (b Basis) valid() bool {
	return b.Wage > 0 && b.Share > 0 && b.YieldBP > 0 && b.OutQty > 0 && b.Duration >= 0 && b.Rent >= 0
}

// sum is a checked running total that stays bad once a step fails.
type sum struct {
	v   int64
	bad bool
}

func (s *sum) add(v int64, ok bool) {
	if s.bad || !ok {
		s.bad = true
		return
	}
	s.v, ok = checked.Add(s.v, v)
	s.bad = !ok
}

// priceOf returns the known price of a commodity; ok is false when the
// commodity is outside prices or its price is 0 or less.
func priceOf(prices []int64, c int) (price int64, ok bool) {
	if c < 0 || c >= len(prices) || prices[c] <= 0 {
		return 0, false
	}
	return prices[c], true
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
func UnitCost(inputs []Term, tools []Wear, prices []int64, b Basis) int64 {
	if !b.valid() {
		return 0
	}
	var s sum
	for _, in := range inputs {
		p, ok := priceOf(prices, in.Commodity)
		if !ok || in.Quantity < 0 {
			return 0
		}
		s.add(checked.Mul(in.Quantity, p))
	}
	for _, t := range tools {
		p, ok := priceOf(prices, t.Commodity)
		if !ok {
			return 0
		}
		s.add(WearCost(t, p))
	}
	s.add(checked.Mul(b.Duration, b.Wage))
	s.add(b.Rent, true)
	den, ok := checked.Mul(b.OutQty, b.YieldBP)
	if s.bad || !ok {
		return 0
	}
	unit, ok := checked.MulDivCeil(s.v, b.Share, den)
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
			if ri := int(m); ri >= 0 && ri < len(recipes) {
				row[c] = lower(row[c], makerCost(&recipes[ri], c, prices, wage))
			}
		}
	}
}

// lower returns u when it is non-zero and below cur (or cur is 0), else cur.
func lower(cur, u int64) int64 {
	if u > 0 && (cur == 0 || u < cur) {
		return u
	}
	return cur
}

// makerCost is the least non-zero UnitCost of commodity c over the outputs of
// r that set an anchor; 0 when r cannot run or has none.
func makerCost(r *Recipe, c int, prices []int64, wage int64) int64 {
	if !r.Runnable || r.YieldBP <= 0 {
		return 0
	}
	var best int64
	for _, o := range r.Outputs {
		if o.Commodity == c && !o.Discard {
			b := Basis{Share: o.ShareBP, OutQty: o.Quantity, YieldBP: r.YieldBP, Duration: r.Duration, Wage: wage, Rent: r.Rent}
			best = lower(best, UnitCost(r.Inputs, r.Tools, prices, b))
		}
	}
	return best
}
