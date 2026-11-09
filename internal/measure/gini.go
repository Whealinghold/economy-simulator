package measure

// lorenzSum returns the sum over the rows, taken in the order idx, of
// Weight_i * (2 * C_{i-1} + Total_i), with C the running total, in 128 bits
// (spec-indicators.md 5.4.1). The caller has checked that 10 000 * W * S is
// below 2^127, and every term is at most 3 * W * S, so no addition or product
// here can leave 128 bits and none is checked.
func lorenzSum(rows []Row, idx []int) Uint128 {
	var c, sum Uint128
	for _, i := range idx {
		t := Uint128{0, uint64(rows[i].Total)}
		inner, _ := c.Add(c)
		inner, _ = inner.Add(t)
		term, _ := inner.MulU64(uint64(rows[i].Weight))
		sum, _ = sum.Add(term)
		c, _ = c.Add(t)
	}
	return sum
}

// Gini returns the population Gini coefficient of the households in basis
// points, rounded once half to even (spec-indicators.md 5.4.1): with W the sum
// of weights, S the sum of totals, C the running total over the rows ascending
// by Total / Weight, D = W * S and N = D - sum Weight_i * (2 * C_{i-1} + Total_i),
// the coefficient is 10 000 * N / D. A row of weight w counts as w households
// each holding Total / w; the maximum for W households is (W - 1) / W.
//
// ok is false when W or S is 0. err is ErrNegative for a negative total,
// ErrRange for a weight below 1 and ErrOverflow when 10 000 * W * S reaches
// 2^127 (the guard of spec-indicators.md 5.2.1).
func Gini(rows []Row) (bp int64, ok bool, err error) {
	w, err := weightSum(rows)
	if err != nil {
		return 0, false, err
	}
	s, err := totalSum(rows)
	if err != nil {
		return 0, false, err
	}
	if w == 0 || s.IsZero() {
		return 0, false, nil
	}
	d, ok := Uint128{0, w}.Mul(s)
	if _, guard := MulBPU(d); !ok || !guard {
		return 0, false, ErrOverflow
	}
	// the guard keeps the sum below 2 * D, and a sort ascending by value keeps it
	// at most D, so N is not negative
	n, _ := d.Sub(lorenzSum(rows, order(rows)))
	num, _ := MulBPU(n) // n <= d, so the guard above holds
	q, _ := RoundDivU(num, d)
	return int64(q.Lo), true, nil
}
