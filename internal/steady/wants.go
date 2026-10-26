package steady

// WantWeight is one item of a want: a commodity and its weight.
type WantWeight struct {
	Commodity int
	Weight    int64
}

// WantRow is one taste row of one group at the reference (spec-viability.md
// 3.2): the households, the taste and region scales in basis points, and the
// want's quantity per period and its period and items.
type WantRow struct {
	Households, Taste, Region, Quantity, Period int64
	Items                                       []WantWeight
}

// wantsBP is the divisor of the two scales, 10 000 each.
const wantsBP = 100_000_000

// FixedTerm returns the exact fixed term W(c) of the goods rows of the rows:
// for each row Households * Taste * Region * Quantity / (10^8 * Period), split
// among its items in proportion to their weights, summed by commodity. The
// terms are exact fractions in lowest terms; none is rounded. A row with no
// positive weight, a period below 1 or a negative factor contributes nothing. A
// term that does not fit two int64 is ErrOverflow.
func FixedTerm(commodities int, rows []WantRow) ([]Ratio, error) {
	total := make([]Frac, commodities)
	for c := range total {
		total[c] = Rat(0, 1)
	}
	for _, r := range rows {
		r.addTo(total)
	}
	out := make([]Ratio, commodities)
	for c, f := range total {
		if !f.Num.IsInt64() || !f.Den.IsInt64() {
			return nil, ErrOverflow
		}
		out[c] = Ratio{Num: f.Num.Int64(), Den: f.Den.Int64()}
	}
	return out, nil
}

// addTo adds the row's demand, split among its items by weight, to total.
func (r WantRow) addTo(total []Frac) {
	var sum int64
	for _, it := range r.Items {
		sum += max(it.Weight, 0)
	}
	if sum < 1 || r.Period < 1 || r.Households < 0 || r.Taste < 0 || r.Region < 0 || r.Quantity < 0 {
		return
	}
	whole := Rat(r.Households, 1).Mul(r.Taste, 1).Mul(r.Region, 1).Mul(r.Quantity, 1).Mul(1, wantsBP).Mul(1, r.Period).Mul(1, sum)
	for _, it := range r.Items {
		if it.Weight > 0 {
			total[it.Commodity] = total[it.Commodity].Add(whole.Mul(it.Weight, 1))
		}
	}
}
