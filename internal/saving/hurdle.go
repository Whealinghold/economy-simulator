package saving

import "github.com/Whealinghold/economy-sim/internal/checked"

// Hurdle returns base + floor(weyl * spread / 2^32), in [base, base + spread)
// (spec 5.5.2, FR-SAV-015): the agent's fixed hurdle, a function of the agent's
// Weyl value alone. No draw is taken.
func Hurdle(base, spread int64, weyl uint32) int64 {
	return satAdd(base, satMulDiv(nonneg(spread), int64(weyl), 1<<32))
}

// Committed reports whether the household holds at least half of the project's
// inputs by value, 2 * held >= inputs (spec 5.5.3, FR-SAV-016).
func Committed(held, inputs int64) bool {
	return satMul(2, nonneg(held)) >= nonneg(inputs)
}

// Lowered returns the hurdle of a committed household, hurdle * (10 000 -
// band) / 10 000 (spec 5.5.3).
func Lowered(hurdle, band int64) int64 {
	return satMulDiv(nonneg(hurdle), nonneg(scale-band), scale)
}

// Passes reports whether a project passes the test of spec 5.5.1
// (FR-SAV-014): benefit above 0 and benefit * 10 000 >= hurdle * cost in 128
// bits. A saturated benefit (at maxMoney) passes and a saturated cost fails.
func Passes(benefit, cost, hurdle, maxMoney int64) bool {
	if benefit <= 0 || cost >= maxMoney {
		return false
	}
	if benefit >= maxMoney {
		return true
	}
	cmp, ok := checked.Cmp3(benefit, scale, 1, nonneg(hurdle), nonneg(cost), 1)
	return ok && cmp >= 0
}

// Candidate is a project that passed, for the choice of spec 5.5.4.
type Candidate struct {
	Benefit, Cost int64
	Repair        bool
	ID            int64 // the recipe's ID: the final tie-break
}

// Better reports whether a is chosen over b (spec 5.5.4, FR-SAV-017): the
// greater benefit over cost by cross-multiplication in 128 bits, then a repair
// before a build, then the lower ID.
func Better(a, b Candidate) bool {
	cmp, _ := checked.Cmp3(nonneg(a.Benefit), nonneg(b.Cost), 1, nonneg(b.Benefit), nonneg(a.Cost), 1)
	if cmp != 0 {
		return cmp > 0
	}
	if a.Repair != b.Repair {
		return a.Repair
	}
	return a.ID < b.ID
}
