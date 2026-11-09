package indicator

import (
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// RatioStats is the distribution of one ratio over household rows, in basis
// points. OK is false when there are no rows. A negative ratio is allowed, so
// there is no Gini coefficient (spec-indicators.md 5.4.3).
type RatioStats struct {
	OK                      bool
	Mean                    int64
	P10, P25, P50, P75, P90 int64
	// Below is the share of households under 10 000 (a ratio under one) and
	// BelowTarget the share under the target the caller gave.
	Below, BelowTarget Index
	Coverage
	Warnings []Warning
}

// WelfareResult is the distribution of the welfare ratio and of the earned
// ratio (the ratio without relief).
type WelfareResult struct {
	Welfare, Earned RatioStats
}

// WelfareDistribution returns the household-weighted mean, the quantiles P10 to
// P90, the share below one and the share below target of the welfare ratio rows,
// and the same for the earned ratio rows (spec-indicators.md 9.1 row 10). The
// rows are those of RowsOf with ColWelfareRatio and ColEarnedRatio, whose Absent
// households (no upkeep) and Cohort are carried into the result. The mean of
// ratios is not the aggregate ratio of the sums (7.7.2). target is the role's
// target ratio in basis points.
func WelfareDistribution(welfare, earned RowSet, target int64) (WelfareResult, error) {
	var res WelfareResult
	var err error
	if res.Welfare, err = ratioStats(welfare, target); err != nil {
		return WelfareResult{}, err
	}
	res.Earned, err = ratioStats(earned, target)
	return res, err
}

// ratioStats is the distribution of one row set.
func ratioStats(set RowSet, target int64) (RatioStats, error) {
	out := RatioStats{Coverage: set.Coverage, Warnings: coverageWarnings(set.Coverage)}
	if len(set.Rows) == 0 {
		return out, nil
	}
	mr := toMeasure(set.Rows)
	w, s, err := sums(mr)
	if err != nil {
		return RatioStats{}, wrap("WelfareDistribution", "sums", fromMeasure(err))
	}
	out.OK = true
	if out.Mean, _, err = mean(s, w); err != nil {
		return RatioStats{}, wrap("WelfareDistribution", "mean", err)
	}
	for _, q := range []struct {
		dst *int64
		at  int64
	}{{&out.P10, 1000}, {&out.P25, 2500}, {&out.P50, 5000}, {&out.P75, 7500}, {&out.P90, 9000}} {
		if *q.dst, _, err = measure.Quantile(mr, q.at); err != nil {
			return RatioStats{}, wrap("WelfareDistribution", "quantile", fromMeasure(err))
		}
	}
	if out.Below, err = shareBelow(mr, 10_000); err != nil {
		return RatioStats{}, err
	}
	out.BelowTarget, err = shareBelow(mr, target)
	return out, err
}

// shareBelow is the share of households strictly below x as an Index.
func shareBelow(rows []measure.Row, x int64) (Index, error) {
	v, ok, err := measure.ShareBelow(rows, x)
	if err != nil {
		return Index{}, wrap("WelfareDistribution", "share_below", fromMeasure(err))
	}
	return Index{Value: BP(v), OK: ok}, nil
}

// coverageWarnings lists the cautions of a row set: WarnCohort where households
// are held identical by construction, WarnAbsent where some were left out.
func coverageWarnings(c Coverage) []Warning {
	var out []Warning
	if c.Cohort > 0 {
		out = append(out, WarnCohort)
	}
	if c.Absent > 0 {
		out = append(out, WarnAbsent)
	}
	return out
}

// InequalityResult is the dispersion of one column of household rows. Gini, Top
// (the share held by the top 10% of households) and Bottom (the share held by the
// bottom 40%) are in basis points; P90P10 is rdiv(10 000 * P90, P10), undefined
// where P10 is not above 0. A row set of one agent of weight above 1 is not an
// error: CohortShare says how much of the world is held identical (6.2).
type InequalityResult struct {
	GiniOK, SharesOK, P90P10OK bool
	Gini, Top, Bottom          int64
	P10, P90                   int64
	P90P10                     BP
	Coverage
	Warnings []Warning
}

// Inequality returns the Gini coefficient, the top and bottom shares and the
// ratio of the 90th to the 10th percentile of the rows (spec-indicators.md 9.1
// row 11): ColMoney for money, ColWealth for wealth with a table. The rows must
// have no negative total (ErrNegativeValue). The estate and pool are outside the
// households and never rows.
func Inequality(set RowSet) (InequalityResult, error) {
	out := InequalityResult{Coverage: set.Coverage, Warnings: coverageWarnings(set.Coverage)}
	mr := toMeasure(set.Rows)
	var err error
	if out.Gini, out.GiniOK, err = measure.Gini(mr); err != nil {
		return InequalityResult{}, wrap("Inequality", "gini", fromMeasure(err))
	}
	if out.Top, _, out.SharesOK, err = measure.TopBottom(mr, 1000); err != nil {
		return InequalityResult{}, wrap("Inequality", "top", fromMeasure(err))
	}
	if out.Bottom, err = bottomShare(mr, out); err != nil {
		return InequalityResult{}, err
	}
	var ok1, ok2 bool
	if out.P10, ok1, err = measure.Quantile(mr, 1000); err != nil {
		return InequalityResult{}, wrap("Inequality", "p10", fromMeasure(err))
	}
	if out.P90, ok2, err = measure.Quantile(mr, 9000); err != nil {
		return InequalityResult{}, wrap("Inequality", "p90", fromMeasure(err))
	}
	if ok1 && ok2 && out.P10 > 0 {
		var q Index
		if q, err = ratioWide("Inequality", "p90p10", measure.FromInt64(out.P90), measure.FromInt64(out.P10)); err != nil {
			return InequalityResult{}, err
		}
		out.P90P10, out.P90P10OK = q.Value, q.OK
	}
	return out, nil
}

// bottomShare returns the share held by the bottom 40%: TopBottom takes one
// fraction for both ends, so the top figure of the call at 1 000 and the bottom
// figure of a call at 4 000 are taken from two calls.
func bottomShare(rows []measure.Row, out InequalityResult) (int64, error) {
	if !out.SharesOK {
		return 0, nil
	}
	_, bottom, _, err := measure.TopBottom(rows, 4000)
	return bottom, wrap("Inequality", "bottom", fromMeasure(err))
}
