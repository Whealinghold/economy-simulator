package welfare

// bp is one in basis points: 10 000 is a ratio of 1.0.
const bp = 10_000

// Ratio returns the welfare ratio in basis points of a window whose records
// sum to incomeSum and costSum, with n missing ticks padded at read
// (spec-welfare-and-wages.md 2.1, 2.2): upkeep is the agent's upkeep cost U in
// the current tick and defaultBP its DefaultRatio. The padding is
// pad_cost = n * U and pad_income = pad_cost * defaultBP / 10 000, truncated,
// so the padded ratio is DefaultRatio only when that division is exact. The
// result is truncated toward zero, may be negative, and is absent (present
// false) when the padded cost is 0. Nothing wraps: every named value
// saturates, and the ratio is one quotient of the unclamped product.
func Ratio(incomeSum, costSum, n, upkeep, defaultBP int64) (ratioBP int64, present bool) {
	padCost := satMul(n, upkeep)
	padIncome := satMulDiv(padCost, defaultBP, bp)
	den := satAdd(costSum, padCost)
	if den == 0 {
		return 0, false
	}
	return satMulDiv(satAdd(incomeSum, padIncome), bp, den), true
}

// EarnedRatio is the earned ratio of 2.1.7: the ratio over the income less the
// relief, with the same padding and truncation, absent when the ratio is.
func EarnedRatio(incomeSum, reliefSum, costSum, n, upkeep, defaultBP int64) (ratioBP int64, present bool) {
	return Ratio(satSub(incomeSum, reliefSum), costSum, n, upkeep, defaultBP)
}
