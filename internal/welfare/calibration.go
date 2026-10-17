package welfare

import "github.com/Whealinghold/economy-sim/internal/checked"

// The calibration formulas run outside the tick (the viability row, tests),
// so they report ok false on an overflow or a zero divisor. The factors are
// multiplied before the one division, so truncation happens once. A product of
// two factors that leaves int64 is reported even when the final quotient would
// fit. dYear is the derived TicksPerYear and dWork the role's ExpectedWorkTicks:
// both are always arguments (FR-WEL-013).

// Wage returns the daily wage for a target ratio, r_bp * B_r * D_year /
// (10 000 * D_work) (spec-welfare-and-wages.md 3.1).
func Wage(ratioBP, basket, dYear, dWork int64) (int64, bool) {
	num, ok := checked.Mul(ratioBP, basket)
	if !ok {
		return 0, false
	}
	den, ok := checked.Mul(bp, dWork)
	if !ok {
		return 0, false
	}
	return checked.MulDiv(num, dYear, den)
}

// RatioForWage returns the ratio in basis points a daily wage produces,
// W * D_work * 10 000 / (B_r * D_year) (3.1).
func RatioForWage(wage, dWork, basket, dYear int64) (int64, bool) {
	num, ok := checked.Mul(wage, dWork)
	if !ok {
		return 0, false
	}
	den, ok := checked.Mul(basket, dYear)
	if !ok {
		return 0, false
	}
	return checked.MulDiv(num, bp, den)
}

// DaysWorked returns the days worked per year in steady state at RestBand 0,
// min(D_year, ceil(r_bp * B_r * D_year / (10 000 * W))) (5, FR-WEL-022). The
// wage must be positive: a zero or negative wage reports ok false.
func DaysWorked(ratioBP, basket, dYear, wage int64) (int64, bool) {
	if wage <= 0 {
		return 0, false
	}
	num, ok := checked.Mul(ratioBP, basket)
	if !ok {
		return 0, false
	}
	den, ok := checked.Mul(bp, wage)
	if !ok {
		return 0, false
	}
	days, ok := checked.MulDivCeil(num, dYear, den)
	if !ok {
		return 0, false
	}
	return min(days, dYear), true
}
