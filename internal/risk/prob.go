package risk

import "github.com/Whealinghold/economy-sim/internal/checked"

// BP is the basis-point denominator: a share of 10 000 is the whole.
const BP = 10_000

// rhe returns a * b / BP rounded half to even. ok is false when a or b is
// negative or the quotient does not fit.
func rhe(a, b int64) (int64, bool) {
	if a < 0 || b < 0 {
		return 0, false
	}
	return checked.MulDivHalfEven(a, b, BP)
}

// Prob returns the effective probability of a row, P = min(10 000,
// rhe(strikeBP * scaleBP, 10 000)), in basis points (spec-risk.md 3.4.1). A
// scale of 0 gives 0, a row that takes no word. ok is false when an argument is
// negative.
func Prob(strikeBP, scaleBP int64) (p int64, ok bool) {
	p, ok = rhe(strikeBP, scaleBP)
	if !ok {
		return 0, false
	}
	return min(p, BP), true
}

// Struck reports whether a draw in [0, 10 000) strikes a row of effective
// probability p: the draw is below p (spec-risk.md 3.4.2). A p of 10 000
// always strikes and a p of 0 never does. A draw below 0 is outside the range the
// world gives and spares, so a broken reduction cannot strike every row.
func Struck(draw, p int64) bool {
	return draw >= 0 && draw < p
}
