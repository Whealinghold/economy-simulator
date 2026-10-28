package endowment

import (
	"math/bits"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Regen returns the state after one tick of regeneration (spec-resources.md 4.2
// to 4.4). The numerator n of the shape is added to the carry and divided by
// D = 10^12 * TicksPerYear: the whole units are added to the stock and the rest
// is kept, so the units added over any run are exactly
// floor((carry0 + sum of numerators) / D). A stock that reaches the capacity is
// clamped to it and banks no carry; a capacity of 0 gives a stock and a carry of
// 0. ok is false for a state outside its range (a negative value, a stock above
// the capacity, a carry not below D), a TicksPerYear whose D does not fit
// int64, or a numerator beyond 128 bits; nothing is wrapped.
func Regen(p Params, in Step) (Result, bool) {
	d, ok := checked.Mul(ticksDenominator, p.TicksPerYear)
	if !ok || !valid(p, in, d) {
		return Result{}, false
	}
	if p.Capacity == 0 {
		return Result{}, true
	}
	n, ok := numerator(p, in)
	if !ok {
		return Result{}, false
	}
	total, ok := n.add(u128{lo: uint64(in.Carry)})
	if !ok {
		return Result{}, false
	}
	q, r := total.divmod64(uint64(d))
	room := uint64(p.Capacity - in.Stock)
	if q.hi != 0 || q.lo >= room {
		// A full stock banks nothing (4.4.1); the clamp is applied in 128 bits
		// before the quotient is narrowed to int64.
		return Result{Stock: p.Capacity, Added: int64(room)}, true
	}
	return Result{Stock: in.Stock + int64(q.lo), Carry: int64(r), Added: int64(q.lo)}, true
}

// valid reports whether the parameters and the state are in range. A carry
// below d also rejects a d of 0 or less, that is a TicksPerYear below 1.
func valid(p Params, in Step, d int64) bool {
	return p.Capacity >= 0 && p.Rate >= 0 && p.Refuge >= 0 && p.ScaleBP >= 0 &&
		in.SeasonBP >= 0 && in.Stock >= 0 && in.Stock <= p.Capacity &&
		in.Carry >= 0 && in.Carry < d
}

// numerator returns the numerator of one tick over D (spec-resources.md 4.2).
func numerator(p Params, in Step) (u128, bool) {
	m, s := uint64(p.ScaleBP), uint64(in.SeasonBP)
	switch p.Shape {
	case Refill:
		// A full stock has a gap of 0 and so a numerator of 0.
		return product(uint64(p.Rate), uint64(p.Capacity-in.Stock), m, s)
	case Logistic:
		return product(uint64(p.Rate), logisticW(p, in.Stock), m, s)
	case Constant:
		return product(uint64(p.Rate), m, s, BP)
	}
	// The set of shapes is closed; an unknown one is refused, not treated as Constant.
	return u128{}, false
}

// logisticW is the logistic factor w = ceil(S' * (C - S') / C) with
// S' = max(S, min(Refuge, C - 1)), 0 when S' is 0 or at the capacity
// (spec-resources.md 4.2, 4.3.3). It is rounded up so that any live stock below
// the capacity can regrow.
func logisticW(p Params, stock int64) uint64 {
	sp := max(stock, min(p.Refuge, p.Capacity-1))
	// At sp = 0 and at sp = C the product below is 0, so w is 0 there too.
	c := uint64(p.Capacity)
	prod := mul64(uint64(sp), c-uint64(sp))
	// The quotient is at most C / 4, so prod.hi < C and Div64 cannot panic.
	q, r := bits.Div64(prod.hi, prod.lo, c)
	if r != 0 {
		q++
	}
	return q
}
