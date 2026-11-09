package indicator

// IncomeResult is the booked income of households (spec-indicators.md 9.1 row 21).
// In version 1 it is never available: OK is false and every figure is 0, because
// the snapshot carries no income, relief or cost fields (plan 2-18 D574, N-130
// item 1). The fields are those the preset will fill when they exist.
type IncomeResult struct {
	ConfigHash [32]byte
	From, To   int64
	OK         bool
	// Income, Relief and Cost are the booked sums in minor units; Ratio is the
	// aggregate ratio sum(income) / sum(cost) and Gap the difference of 7.5.2.
	Income, Relief, Cost int64
	Ratio                BP
	Gap                  int64
}

// HouseholdIncome returns OK false in version 1: the income fields of the role
// row (IncomeLast, ReliefLast, CostLast) do not exist, so the preset has nothing
// to book (FR-IND-050 first clause). It does not fail.
func HouseholdIncome(w Window) (IncomeResult, error) {
	return IncomeResult{ConfigHash: w.ConfigHash, From: w.From, To: w.To}, nil
}
