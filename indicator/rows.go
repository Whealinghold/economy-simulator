package indicator

import (
	"fmt"

	"github.com/Whealinghold/economy-sim/internal/measure"
)

// Household is a household row: Weight households that together hold Total, each
// holding Total / Weight, which need not be an integer (spec-indicators.md
// 4.5.1). Weight is at least 1.
type Household struct{ Weight, Total int64 }

// StatOp selects the statistic of Stat. The set is closed and its members are
// rule constants of the package.
type StatOp int

// The statistics of Stat (spec-indicators.md 3.6).
const (
	// StatSum is the sum of the totals, S.
	StatSum StatOp = iota + 1
	// StatMean is the household-weighted mean rdiv(S, W), half to even.
	StatMean
	// StatQuantile is the weighted nearest-rank quantile; arg is q in basis points.
	StatQuantile
	// StatShareBelow is the share, in basis points, of households whose value is
	// strictly below arg.
	StatShareBelow
	// StatGini is the population Gini coefficient in basis points, for
	// non-negative totals.
	StatGini
)

// String returns the name of the statistic, as the error of a call reports it.
func (op StatOp) String() string {
	switch op {
	case StatSum:
		return "sum"
	case StatMean:
		return "mean"
	case StatQuantile:
		return "quantile"
	case StatShareBelow:
		return "share_below"
	case StatGini:
		return "gini"
	}
	return fmt.Sprintf("stat_op_%d", int(op))
}

// StatResult is the value of a statistic. OK is false when it is undefined (no
// households, or a zero total for the Gini coefficient), and then Value is 0.
type StatResult struct {
	OK bool
	// Value is in the unit of the totals, or in basis points for a share or the
	// Gini coefficient.
	Value int64
	// Households is the sum of the weights, W.
	Households int64
}

// toMeasure converts rows to the primitive rows of internal/measure, inline at
// the call (architecture.md 2.9).
func toMeasure(rows []Household) []measure.Row {
	out := make([]measure.Row, len(rows))
	for i, r := range rows {
		out[i] = measure.Row{Weight: r.Weight, Total: r.Total}
	}
	return out
}

// Stat returns a weighted statistic of household rows (spec-indicators.md 3.6).
// The result does not depend on the order of the rows. err is ErrRange for a
// weight below 1 or a quantile outside [0, 10 000], ErrNegativeValue for the
// Gini coefficient of a negative total, and ErrOverflow for a sum or a guard
// past its bound.
func Stat(rows []Household, op StatOp, arg int64) (StatResult, error) {
	mr := toMeasure(rows)
	fail := func(err error) (StatResult, error) {
		return StatResult{}, wrap("Stat", op.String(), fromMeasure(err))
	}
	w, s, err := sums(mr)
	if err != nil {
		return fail(err)
	}
	res := StatResult{Households: w}
	var ok bool
	switch op {
	case StatSum:
		res.Value, ok = s, true
	case StatMean:
		res.Value, ok, err = mean(s, w)
	case StatQuantile:
		res.Value, ok, err = measure.Quantile(mr, arg)
	case StatShareBelow:
		res.Value, ok, err = measure.ShareBelow(mr, arg)
	case StatGini:
		res.Value, ok, err = measure.Gini(mr)
	default:
		return StatResult{}, wrap("Stat", op.String(), ErrRange)
	}
	if err != nil {
		return fail(err)
	}
	res.OK = ok
	return res, nil
}

// sums returns the sum of the weights W and of the totals S, checked: a weight
// below 1 is ErrRange and a sum past int64 is ErrOverflow.
func sums(rows []measure.Row) (w, s int64, err error) {
	ws, ss := measure.FromInt64(0), measure.FromInt64(0)
	for _, r := range rows {
		if r.Weight < 1 {
			return 0, 0, measure.ErrRange
		}
		ws, _ = ws.Add(measure.FromInt64(r.Weight)) // 2^64 rows would be needed to pass 2^127
		ss, _ = ss.Add(measure.FromInt64(r.Total))
	}
	w, wok := measure.ToInt64(ws)
	s, sok := measure.ToInt64(ss)
	if !wok || !sok {
		return 0, 0, measure.ErrOverflow
	}
	return w, s, nil
}

// mean returns rdiv(s, w), and ok false when there are no households.
func mean(s, w int64) (int64, bool, error) {
	if w == 0 {
		return 0, false, nil
	}
	q, _ := measure.RoundDiv(measure.FromInt64(s), measure.FromInt64(w))
	v, _ := measure.ToInt64(q) // a mean lies between the least and greatest total
	return v, true, nil
}
