package policy

import "github.com/Whealinghold/economy-sim/internal/checked"

// Ratio is one role row of a covered region for the mean_ratio gauge: the mean
// welfare ratio in basis points and the households it is over. A row with no
// ratio is not passed (spec-policy.md 8.3.2).
type Ratio struct {
	BP         int64
	Households int64
}

// PriceRow is one covered region with a market row of the commodity for the
// price gauge: the households of the region, the published price and its age in
// ticks (spec-policy.md 8.3.2).
type PriceRow struct {
	Households int64
	Price      int64
	Age        int64
}

// TreasuryReading returns the balance of the treasury, and ok false when the
// jurisdiction is not governed (the gauge is then undefined).
func TreasuryReading(balance int64, governed bool) (reading int64, ok bool) {
	return balance, governed
}

// UnfedShare returns rdiv(10 000 * unfed, households), the share of households
// that went unfed in basis points, rounded half to even, and ok false when there
// are no households or an input is negative (spec-policy.md 8.3.2).
func UnfedShare(unfed, households int64) (reading int64, ok bool) {
	if unfed < 0 || households < 1 {
		return 0, false
	}
	return checked.MulDivHalfEven(unfed, Basis, households)
}

// MeanRatio returns rdiv(sum(BP * Households), sum(Households)) over the rows,
// the household-weighted mean ratio in basis points rounded half to even, and ok
// false when the households sum to 0 or an input is negative or too large
// (spec-policy.md 8.3.2). The products are summed in 128 bits, so the sum is
// exact and independent of the order of the rows.
func MeanRatio(rows []Ratio) (reading int64, ok bool) {
	var acc checked.Acc
	var total int64
	for _, r := range rows {
		if !acc.AddMul(r.BP, r.Households) {
			return 0, false
		}
		if total, ok = checked.Add(total, r.Households); !ok || r.Households < 0 {
			return 0, false
		}
	}
	return acc.DivHalfEven(total)
}

// Price returns rdiv(sum(Households * Price), sum(Households)) over the rows
// whose age is at most maxAge, the household-weighted price in minor units
// rounded half to even, and ok false when no row counts, the households of the
// rows that count sum to 0, or an input is negative or too large (spec-policy.md
// 8.3.2). A row older than maxAge is not a reading.
func Price(rows []PriceRow, maxAge int64) (reading int64, ok bool) {
	var acc checked.Acc
	var total int64
	for _, r := range rows {
		if r.Age > maxAge {
			continue
		}
		if !acc.AddMul(r.Households, r.Price) {
			return 0, false
		}
		if total, ok = checked.Add(total, r.Households); !ok {
			return 0, false
		}
	}
	return acc.DivHalfEven(total)
}
