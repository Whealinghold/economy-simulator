package measure

import (
	"cmp"
	"math/bits"
	"slices"
)

// Row is a household row: Weight households that together hold Total, each
// holding Total / Weight (spec-indicators.md 4.5.1). A row of weight w and total
// T stands for w households of value T / w.
type Row struct{ Weight, Total int64 }

// order returns the positions of rows sorted ascending by Total / Weight, by
// cross multiplication in 128 bits (Total_i * Weight_j against Total_j *
// Weight_i), so no division and no rounding decides the order. Ties end in the
// row position, a total order (architecture.md 7.6).
func order(rows []Row) []int {
	idx := make([]int, len(rows))
	for i := range idx {
		idx[i] = i
	}
	slices.SortFunc(idx, func(i, j int) int {
		a := MulInt64(rows[i].Total, rows[j].Weight)
		b := MulInt64(rows[j].Total, rows[i].Weight)
		if c := a.Cmp(b); c != 0 {
			return c
		}
		return cmp.Compare(i, j)
	})
	return idx
}

// weightSum returns the sum of the weights, and ErrRange for a weight below 1 or
// ErrOverflow for a sum past uint64.
func weightSum(rows []Row) (uint64, error) {
	var w uint64
	for _, r := range rows {
		if r.Weight < 1 {
			return 0, ErrRange
		}
		var c uint64
		if w, c = bits.Add64(w, uint64(r.Weight), 0); c != 0 {
			return 0, ErrOverflow
		}
	}
	return w, nil
}

// totalSum returns the sum of the totals of rows that must hold none below 0,
// and ErrNegative for a negative total.
func totalSum(rows []Row) (Uint128, error) {
	var s Uint128
	for _, r := range rows {
		if r.Total < 0 {
			return Uint128{}, ErrNegative
		}
		s, _ = s.Add(Uint128{0, uint64(r.Total)}) // 2^64 rows would be needed to pass 128 bits
	}
	return s, nil
}

// rankOf returns k = max(1, ceil(q * W / 10 000)), the rank of the quantile q in
// basis points of w households (spec-indicators.md 5.5.2), which is at most w.
func rankOf(q int64, w uint64) uint64 {
	num, _ := Mul64(uint64(q), w).Add(Uint128{0, bpScale - 1})
	k, _, _ := num.DivRem(Uint128{0, bpScale})
	return max(1, k.Lo)
}

// Quantile returns the weighted nearest-rank quantile q (basis points, in
// [0, 10 000]) of the household values: the value Total / Weight, half to even,
// of the first row, in ascending order, whose cumulative weight reaches
// k = max(1, ceil(q * W / 10 000)) (spec-indicators.md 5.5.2). A total may be
// negative. ok is false when there are no rows; err is ErrRange for a weight
// below 1 or q outside the range and ErrOverflow for a weight sum past uint64.
func Quantile(rows []Row, q int64) (value int64, ok bool, err error) {
	vs, ok, err := Quantiles(rows, q)
	if !ok || err != nil {
		return 0, false, err
	}
	return vs[0], true, nil
}

// Quantiles returns the quantile of each q, as Quantile does, sorting the rows
// once for all of them. The values are in the order of qs.
func Quantiles(rows []Row, qs ...int64) (values []int64, ok bool, err error) {
	w, err := weightSum(rows)
	if err != nil {
		return nil, false, err
	}
	for _, q := range qs {
		if q < 0 || q > bpScale {
			return nil, false, ErrRange
		}
	}
	if len(rows) == 0 {
		return nil, false, nil
	}
	idx := order(rows)
	values = make([]int64, len(qs))
	for n, q := range qs {
		k, cum := rankOf(q, w), uint64(0)
		for _, i := range idx {
			if cum += uint64(rows[i].Weight); cum >= k {
				v, _ := RoundDiv(FromInt64(rows[i].Total), FromInt64(rows[i].Weight))
				values[n], _ = ToInt64(v)
				break
			}
		}
	}
	return values, true, nil
}

// ShareBelow returns the share, in basis points and half to even, of the
// households whose value is strictly below x: the rows with Total < x * Weight
// (spec-indicators.md 5.5.3). A total and x may be negative. ok is false when
// there are no rows; err is as for Quantile.
func ShareBelow(rows []Row, x int64) (bp int64, ok bool, err error) {
	w, err := weightSum(rows)
	if err != nil || w == 0 {
		return 0, false, err
	}
	var below uint64
	for _, r := range rows {
		if MulInt64(x, r.Weight).Cmp(FromInt64(r.Total)) > 0 {
			below += uint64(r.Weight)
		}
	}
	q, _ := RoundDivU(Mul64(below, bpScale), Uint128{0, w})
	return int64(q.Lo), true, nil
}

// heldShare returns the share, in basis points and half to even, of the total
// s held by the first k households in the order idx; the row at the boundary is
// split by weight, k' of its w households holding k' * T / w
// (spec-indicators.md 5.4.4).
func heldShare(rows []Row, idx []int, k uint64, s Uint128) (int64, error) {
	var full Uint128 // the totals of the rows taken whole
	got, w, t := uint64(0), uint64(1), uint64(0)
	for _, i := range idx {
		rw, rt := uint64(rows[i].Weight), uint64(rows[i].Total)
		if got+rw > k {
			w, t = rw, rt // the boundary row
			break
		}
		got += rw
		full, _ = full.Add(Uint128{0, rt}) // below s, which fit
	}
	var o sticky
	num := o.mulBP(o.add(o.mul(full, w), Mul64(k-got, t)))
	den := o.mul(s, w)
	if o.over {
		return 0, ErrOverflow
	}
	q, _ := RoundDivU(num, den)
	return int64(q.Lo), nil
}

// TopBottom returns the share of the total held by the top and by the bottom
// k = max(1, ceil(q * W / 10 000)) households (spec-indicators.md 5.4.4), in
// basis points. Every total must be at least 0 (ErrNegative). ok is false when
// W or the total is 0; err is ErrRange for a weight below 1 or q outside
// [0, 10 000] and ErrOverflow past 128 bits.
func TopBottom(rows []Row, q int64) (top, bottom int64, ok bool, err error) {
	w, err := weightSum(rows)
	if err != nil {
		return 0, 0, false, err
	}
	s, err := totalSum(rows)
	switch {
	case err != nil:
		return 0, 0, false, err
	case q < 0 || q > bpScale:
		return 0, 0, false, ErrRange
	case w == 0 || s.IsZero():
		return 0, 0, false, nil
	}
	asc := order(rows)
	k := rankOf(q, w)
	bottom, errB := heldShare(rows, asc, k, s)
	slices.Reverse(asc)
	top, errT := heldShare(rows, asc, k, s)
	if err := cmp.Or(errB, errT); err != nil {
		return 0, 0, false, err
	}
	return top, bottom, true, nil
}
