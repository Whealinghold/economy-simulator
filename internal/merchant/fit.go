package merchant

import "github.com/Whealinghold/economy-sim/internal/checked"

// Units returns the carrying units a lot of q units needs (spec-merchants.md 5.4):
// max(1, ceil(q * mass / unitMass), ceil(q * bulk / unitBulk)), a term whose
// capacity is 0 left out. ok is false on overflow.
func Units(q, mass, bulk, unitMass, unitBulk int64) (n int64, ok bool) {
	n = 1
	for _, t := range [2][2]int64{{mass, unitMass}, {bulk, unitBulk}} {
		if t[1] == 0 {
			continue
		}
		u, ok := checked.MulDivCeil(q, t[0], t[1])
		if !ok {
			return 0, false
		}
		n = max(n, u)
	}
	return n, true
}

// FitInput is what the sizing of a lot reads (spec-merchants.md 6.2.1): the cap of
// the lot, the money, the reserve, the cost at their maximum limits of the other
// inputs of the trip the merchant does not yet hold (with the toll), and the
// dearest price High at which it would bid for the cargo.
type FitInput struct {
	// Cap is the most units the lot may be: LotUnits carrying units, never more
	// than MaxQuantity.
	Cap int64
	// Money, Reserve and Other are the cash, the default's reserve and the cost of
	// the other inputs.
	Money, Reserve, Other int64
	// High is the dearest bid price for the cargo before the break-even.
	High int64
}

// Afford returns floor((money - reserve - other) / limit): the units the budget
// buys at the limit, 0 for a budget that is not positive or a limit below 1.
func (in FitInput) Afford(limit int64) int64 {
	budget, ok := checked.Sub(in.Money, in.Reserve)
	if ok {
		budget, ok = checked.Sub(budget, in.Other)
	}
	if !ok || budget <= 0 || limit < 1 {
		return 0
	}
	return budget / limit
}

// Fit sizes the lot by spec-merchants.md 6.2.2. breakEven is the break-even price
// at a given lot, ok false when the lot does not fit arithmetically. The first
// pass takes q = min(Cap, afford at High) and computes the break-even at that q.
// The second pass of the spec recomputes the affordable units at min(High,
// breakEven(q)) and takes the smaller lot; that limit is never above High, so it
// affords at least q and the smaller lot is q: the second pass lowers nothing, and
// what remains of it is the rejection of a break-even below 1 (no bid). A lot of
// 0 is returned when the first pass leaves less than one unit or the break-even
// is below 1: the caller counts the case Unaffordable. The caller evaluates the
// test and the prices at the returned lot once more. One call of breakEven.
func Fit(in FitInput, breakEven func(q int64) (int64, bool)) int64 {
	q := min(in.Cap, in.Afford(in.High))
	if q < 1 {
		return 0
	}
	if be, ok := breakEven(q); !ok || be < 1 {
		return 0
	}
	return q
}
