package beliefs

import (
	"math"
	"math/bits"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// bpDen is the denominator of a basis-point ratio (architecture.md 3.5).
const bpDen = 10_000

// halfFill is the fill share, in basis points, from which the width narrows
// (spec-beliefs.md 5.7).
const halfFill = 5_000

// Belief is the price belief of one agent for one commodity (spec-beliefs.md
// 3.1): a centre in minor units per unit and a half-width in basis points of
// the centre.
type Belief struct {
	Centre, Width int64
}

// Bounds is the interval a belief's centre is kept in (spec-beliefs.md 3.2).
// Conflict is true when the cost floor exceeded the upper end of the
// reference band, so the floor won and the ceiling was raised to meet it
// (3.2.3).
type Bounds struct {
	Floor, Ceiling int64
	Conflict       bool
}

// Params are the belief parameters the package reads, in basis points unless
// stated (spec-beliefs.md 11), and MaxPrice, the bound of spec-config.md 4.1,
// passed in because internal packages never import the root (architecture.md
// 2.9). PriceWindow is the capacity of a Window and is not a field.
type Params struct {
	Step, AdjustProb, Significance int64
	EvidenceShare                  int64 // basis points of Step: the largest evidence move
	StockLow, StockHigh, MaxAge    int64
	WidthInit, WidthMin            int64
	WidthMax, WidthStep            int64
	CostFloor, CostCeiling         int64
	RefBandLow, RefBandHigh        int64
	MaxPrice                       int64
}

// mulBP returns v * bp / 10 000, truncated toward zero, and false when the
// result does not fit int64.
func mulBP(v, bp int64) (int64, bool) {
	return checked.MulDiv(v, bp, bpDen)
}

// mulBPSat is mulBP with an overflowing result read as limit.
func mulBPSat(v, bp, limit int64) int64 {
	q, ok := mulBP(v, bp)
	if !ok {
		return limit
	}
	return q
}

// Limits returns the bounds of a belief (spec-beliefs.md 3.2) from the
// effective reference price ref and the caller's unit cost unitCost, 0 when
// unknown. A ref below 1 is read as 1 and a negative unitCost as 0. Every
// ratio is truncated toward zero; one that does not fit int64 is read as
// MaxPrice, so the floor becomes MaxPrice (3.2.5). Floor never exceeds
// Ceiling, and both lie in [1, MaxPrice] when MaxPrice is at least 1.
func Limits(ref, unitCost int64, p Params) Bounds {
	ref = max(ref, 1)
	unitCost = max(unitCost, 0)

	refLow := max(1, mulBPSat(ref, p.RefBandLow, p.MaxPrice))
	refHigh := min(p.MaxPrice, mulBPSat(ref, p.RefBandHigh, p.MaxPrice))
	costLow, costHigh := int64(0), p.MaxPrice
	if unitCost > 0 {
		costLow = mulBPSat(unitCost, p.CostFloor, p.MaxPrice)
		costHigh = mulBPSat(unitCost, p.CostCeiling, p.MaxPrice)
	}

	floor := min(p.MaxPrice, max(refLow, costLow))
	return Bounds{
		Floor:    floor,
		Ceiling:  max(floor, min(refHigh, costHigh)),
		Conflict: costLow > refHigh,
	}
}

// Clamp returns b with its centre moved into [bd.Floor, bd.Ceiling] and its
// width into [p.WidthMin, p.WidthMax] (spec-beliefs.md 3.3, 10.3), and whether
// anything changed. The bounds and the width limits must be valid.
func Clamp(b Belief, bd Bounds, p Params) (Belief, bool) {
	c := Belief{
		Centre: min(max(b.Centre, bd.Floor), bd.Ceiling),
		Width:  min(max(b.Width, p.WidthMin), p.WidthMax),
	}
	return c, c != b
}

// Draw returns the limit price drawn from b's effective range (spec-beliefs.md
// 3.3, 6). The belief is re-clamped first, so a stored belief outside its
// bounds or width limits still gives a price in [bd.Floor, bd.Ceiling]. It
// consumes exactly the one word u, also when the range holds one price: the
// result is Low plus the high word of u * n, with n = High - Low + 1 and
// half = Centre * Width / 10 000 truncated.
func Draw(b Belief, bd Bounds, p Params, u uint64) int64 {
	b, _ = Clamp(b, bd, p)
	half := mulBPSat(b.Centre, b.Width, b.Centre)
	low := max(bd.Floor, b.Centre-half)
	high := min(bd.Ceiling, b.Centre+half)
	hi, _ := bits.Mul64(u, uint64(high-low+1))
	return low + int64(hi)
}

// Outcome is what happened to one order as a whole (spec-beliefs.md 4.2). An
// order whose Quantity is below 1 has a fill share of 0 and counts as not
// fully filled; Filled is read inside [0, Quantity] and Limit below 1 as 1
// (5.9).
type Outcome struct {
	Bid      bool
	Limit    int64
	Quantity int64
	Filled   int64
}

// Quote is the market report of the order's market (spec-beliefs.md 4.3): the
// market price and its age, and the best unmatched bid and ask with a flag each
// for "none".
type Quote struct {
	Price, Age       int64
	BestBid, BestAsk int64
	HasBid, HasAsk   bool
}

// Result is what Update alone knows: whether the adjustment gate was open. The
// world never sees it; it builds a Change instead (spec-beliefs.md 12.1.1).
type Result struct {
	Open bool
}

// Change is what the world holds for one outcome: the stored belief before,
// the belief after Revise and the clamp, the bounds, the gate of the outcome's
// draw and whether the clamp changed the strategy's belief. The tally and the
// reason of spec-beliefs.md 12 are functions of it, so a strategy that ignores
// the default rule is counted the same way.
type Change struct {
	Before, After Belief
	Bounds        Bounds
	Open          bool
	Clamped       bool
}

// Raised reports whether the centre after is above the stored centre.
func (c Change) Raised() bool { return c.After.Centre > c.Before.Centre }

// Lowered reports whether the centre after is below the stored centre.
func (c Change) Lowered() bool { return c.After.Centre < c.Before.Centre }

// Moved reports whether the centre after differs from the stored centre.
func (c Change) Moved() bool { return c.After.Centre != c.Before.Centre }

// AtFloor reports whether the centre after is on the floor. A belief whose
// Floor equals its Ceiling is both AtFloor and AtCeiling.
func (c Change) AtFloor() bool { return c.After.Centre == c.Bounds.Floor }

// AtCeiling reports whether the centre after is on the ceiling.
func (c Change) AtCeiling() bool { return c.After.Centre == c.Bounds.Ceiling }

// Conflict reports whether the cost floor exceeded the reference band.
func (c Change) Conflict() bool { return c.Bounds.Conflict }

// Gate reports whether the adjustment gate is open for the draw u
// (spec-beliefs.md 5.2): the high word of u * 10 000 is below adjustProb. It
// consumes the one word u, with no loop and no rejection. An adjustProb of 0
// is never open and 10 000 always.
func Gate(u uint64, adjustProb int64) bool {
	hi, _ := bits.Mul64(u, bpDen)
	return int64(hi) < adjustProb
}

// addSat returns a + b, or the int64 limit of the sign of b on overflow.
func addSat(a, b int64) int64 {
	if s, ok := checked.Add(a, b); ok {
		return s
	}
	if b > 0 {
		return math.MaxInt64
	}
	return math.MinInt64
}

// gap returns a - b, saturated to [-MaxInt64, MaxInt64] so it can be negated.
func gap(a, b int64) int64 {
	s, ok := checked.Sub(a, b)
	switch {
	case !ok && a > b:
		return math.MaxInt64
	case !ok || s == math.MinInt64:
		return -math.MaxInt64
	}
	return s
}

// fillShare returns the filled units, clamped to [0, Quantity], and the fill
// share in basis points, truncated (spec-beliefs.md 4.2, 5.9).
func fillShare(o Outcome) (filled, share int64) {
	if o.Quantity < 1 {
		return 0, 0
	}
	filled = min(max(o.Filled, 0), o.Quantity)
	share, _ = checked.MulDiv(filled, bpDen, o.Quantity) // filled <= Quantity: cannot overflow
	return filled, share
}

// target returns the target price of spec-beliefs.md 5.4 and whether there is
// one. limit is at least 1.
func target(o Outcome, limit, filled int64, q Quote, centre int64, p Params) (int64, bool) {
	fresh := q.Age <= p.MaxAge
	if o.Quantity < 1 || filled < o.Quantity { // not fully filled
		return unfilledTarget(o.Bid, limit, q, fresh)
	}
	threshold := mulBPSat(centre, p.Significance, math.MaxInt64)
	if fresh && gapFor(o.Bid, limit, q.Price) > threshold {
		return q.Price, true
	}
	return 0, false
}

// gapFor is how far the limit lies beyond the market price on the unfavourable
// side: limit - price for a bid, price - limit for an ask.
func gapFor(bid bool, limit, price int64) int64 {
	if bid {
		return gap(limit, price)
	}
	return gap(price, limit)
}

// unfilledTarget is the target of an order that was not fully filled: the
// smaller of the best unmatched ask and the fresh price above the limit for a
// bid, the larger of the best unmatched bid and the fresh price below the limit
// for an ask.
func unfilledTarget(bid bool, limit int64, q Quote, fresh bool) (int64, bool) {
	var t int64
	var ok bool
	pick := func(c int64) {
		switch {
		case !ok:
			t, ok = c, true
		case bid && c < t, !bid && c > t:
			t = c
		}
	}
	if bid && q.HasAsk {
		pick(q.BestAsk)
	}
	if !bid && q.HasBid {
		pick(q.BestBid)
	}
	if fresh && (bid && q.Price > limit || !bid && q.Price < limit) {
		pick(q.Price)
	}
	return t, ok
}

// evidenceMove is the evidence move of spec-beliefs.md 5.4: toward the target,
// by at most de, never past it from the centre and never more than the distance
// from the limit, and 0 when the two gaps differ in sign or either is 0.
func evidenceMove(t, limit, centre, de int64) int64 {
	gL, gC := gap(t, limit), gap(t, centre)
	if gL == 0 || gC == 0 || (gL > 0) != (gC > 0) {
		return 0
	}
	if gC < 0 {
		return -min(-gL, -gC, de)
	}
	return min(gL, gC, de)
}

// stockStep is the stock component's direction of spec-beliefs.md 5.5: +1 when
// short, -1 when long, 0 inside the band or with no target. A negative stock or
// target is read as 0 and a ratio that overflows is above StockHigh (5.9).
func stockStep(stock, target int64, p Params) int64 {
	stock, target = max(stock, 0), max(target, 0)
	if target == 0 {
		return 0
	}
	ratio, ok := checked.MulDiv(stock, bpDen, target)
	switch {
	case !ok || ratio > p.StockHigh:
		return -1
	case ratio < p.StockLow:
		return 1
	}
	return 0
}

// Update applies the update rule of spec-beliefs.md 5 to one order's outcome
// and returns the new belief and the result. The stored belief is re-clamped
// first (3.3); a closed gate (5.2, one word u) leaves it at that. Otherwise the
// centre moves by the evidence move (5.4) plus the stock step (5.5) of
// delta = max(1, Centre * Step / 10 000), truncated, and is clamped to the
// bounds; the width narrows by WidthStep when the fill share is at least 5 000
// and widens otherwise, clamped (5.7). A sum or product that does not fit int64
// saturates and never wraps (5.9). The bounds must be valid.
func Update(b Belief, bd Bounds, o Outcome, q Quote, stock, tgt int64, p Params, u uint64) (Belief, Result) {
	b, _ = Clamp(b, bd, p)
	if !Gate(u, p.AdjustProb) {
		return b, Result{}
	}
	delta := max(1, mulBPSat(b.Centre, p.Step, math.MaxInt64))
	de := max(1, mulBPSat(delta, p.EvidenceShare, math.MaxInt64))
	limit := max(o.Limit, 1)
	filled, share := fillShare(o)

	var move int64
	if t, ok := target(o, limit, filled, q, b.Centre, p); ok {
		move = evidenceMove(t, limit, b.Centre, de)
	}
	move = addSat(move, stockStep(stock, tgt, p)*delta)

	w := p.WidthStep
	if share >= halfFill { // a half or more of the order was filled: narrower
		w = -w
	}
	nb, _ := Clamp(Belief{Centre: addSat(b.Centre, move), Width: addSat(b.Width, w)}, bd, p)
	return nb, Result{Open: true}
}
