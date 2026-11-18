package employment

import "github.com/Whealinghold/economy-sim/internal/checked"

// Wage returns the wage per household-tick of a pair of a worker's reservation
// and an employer's bid (spec-employment.md 5.4.2):
//
//	wage = reserve + halfEven(shareBP * (bid - reserve) / 10 000)
//
// The division rounds half to even, as clearing does (spec-clearing.md 5.2),
// so the result lies in [reserve, bid]: the worker never gets less than it
// asked and the employer never pays more than it bid. The wage depends on the
// two limits and the share only, never on how many offers or vacancies there
// are (5.4.3). It reports false when reserve is below 0, bid is below reserve
// or shareBP is outside 0 to 10 000, so a caller that admitted its inputs never
// sees it.
func Wage(reserve, bid, shareBP int64) (int64, bool) {
	if reserve < 0 || bid < reserve || shareBP < 0 || shareBP > WholeBP {
		return 0, false
	}
	cut, ok := checked.MulDivHalfEven(shareBP, bid-reserve, WholeBP)
	if !ok {
		return 0, false
	}
	return checked.Add(reserve, cut)
}

// Bill returns the money a hire escrows at its formation (spec-employment.md
// 5.5): wage * ticks * households. A product that does not fit int64 makes the
// formation fail with employer_funds (14.1), so the caller treats ok false as
// "the employer cannot pay". A factor below 0 reports false. Each product is a
// product of non-negative factors, so one that overflows int64 means the whole
// bill does, and the two checked products are exact.
func Bill(wage, ticks, households int64) (int64, bool) {
	if wage < 0 || ticks < 0 || households < 0 {
		return 0, false
	}
	perHousehold, ok := checked.Mul(wage, ticks)
	if !ok {
		return 0, false
	}
	return checked.Mul(perHousehold, households)
}
