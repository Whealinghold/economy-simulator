package indicator

import (
	"fmt"

	economysim "github.com/Whealinghold/economy-sim"
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
		var fits bool
		if res.Value, fits = measure.ToInt64(s); !fits { // only a sum is stated as int64
			return fail(measure.ErrOverflow)
		}
		ok = true
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
// below 1 is ErrRange and a weight sum past int64 is ErrOverflow. S is 128 bits,
// because only a sum or a mean states it, and a quantile or a Gini coefficient
// of totals that sum past int64 is still defined.
func sums(rows []measure.Row) (w int64, s measure.Int128, err error) {
	ws, ss := measure.FromInt64(0), measure.FromInt64(0)
	for _, r := range rows {
		if r.Weight < 1 {
			return 0, measure.Int128{}, measure.ErrRange
		}
		ws, _ = ws.Add(measure.FromInt64(r.Weight)) // 2^64 rows would be needed to pass 2^127
		ss, _ = ss.Add(measure.FromInt64(r.Total))
	}
	w, wok := measure.ToInt64(ws)
	if !wok {
		return 0, measure.Int128{}, measure.ErrOverflow
	}
	return w, ss, nil
}

// mean returns rdiv(s, w), and ok false when there are no households.
func mean(s measure.Int128, w int64) (int64, bool, error) {
	if w == 0 {
		return 0, false, nil
	}
	q, _ := measure.RoundDiv(s, measure.FromInt64(w))
	v, _ := measure.ToInt64(q) // a mean lies between the least and greatest total
	return v, true, nil
}

// Holding is the units of one commodity an agent holds.
type Holding struct {
	Commodity economysim.CommodityID
	Quantity  economysim.Quantity
}

// AgentRow is the part of an agent's state at a boundary that the household rows
// read: the fields of AgentState named in spec-integration.md 4.7.2 (identity,
// standing, holdings). It is plain data. A pseudo-agent (an estate, an outside
// counterparty) has no AgentRow, because the world lists none in Agents(r), so
// no row set holds one (FR-IND-027).
//
// shortcut: AgentState (plan 2-17) is not in the tree yet; when it merges, replace
// this type by it, field for field, in RowsOf.
type AgentRow struct {
	Region economysim.RegionID
	// Weight is the households the agent stands for, at least 1.
	Weight int64
	// External is true for an agent a consumer drives (a player).
	External bool
	Money    economysim.Money
	Hunger   int64
	// WelfareRatioBP and EarnedRatioBP are the ratios in basis points; they exist
	// only where the Has field is true (the agent has an upkeep).
	WelfareRatioBP, EarnedRatioBP   int64
	HasWelfareRatio, HasEarnedRatio bool
	Holdings                        []Holding
}

// Column selects the quantity of a household row. The set is closed.
type Column int

// The columns of RowsOf (spec-indicators.md 4.5.2).
const (
	// ColMoney is the agent's money, the cohort's sum.
	ColMoney Column = iota + 1
	// ColWelfareRatio and ColEarnedRatio are the ratios; a state, so Total is
	// Weight times the ratio.
	ColWelfareRatio
	ColEarnedRatio
	// ColHunger is Weight times the hunger count.
	ColHunger
	// ColWealth is money plus the holdings valued at the region's prices of
	// RowOpt.Prices (goods valued as new, 4.5.5).
	ColWealth
)

// RowOpt are the options of RowsOf.
type RowOpt struct {
	// ExcludeExternal drops the agents with External true (4.5.2).
	ExcludeExternal bool
	// Prices values the holdings of ColWealth.
	Prices Prices
}

// Coverage is what every result over household rows reports beside its value
// (spec-indicators.md 4.5.4, 6.2): the households used, the agents they come
// from, the households left out because the value does not exist, and the
// households of rows with weight above 1, which are held identical by
// construction, with their share in basis points.
type Coverage struct {
	Agents, Households, Absent, Cohort int64
	CohortShare                        Index
}

// RowSet is the household rows of one column with their coverage.
type RowSet struct {
	Rows []Household
	Coverage
}

// RowsOf builds the household rows of one column from the states a consumer
// already holds (spec-indicators.md 4.5.2). A row whose value does not exist is
// left out and counted in Absent with its weight. err is ErrRange for a weight
// below 1 or an unknown column, and ErrOverflow for a total past int64.
func RowsOf(states []AgentRow, col Column, opt RowOpt) (RowSet, error) {
	var set RowSet
	var cohort, hh, absent acc
	for _, a := range states {
		if a.Weight < 1 {
			return RowSet{}, wrap("RowsOf", "weight", ErrRange)
		}
		if opt.ExcludeExternal && a.External {
			continue
		}
		total, ok, err := a.total(col, opt.Prices)
		if err != nil {
			return RowSet{}, err
		}
		if !ok {
			absent.addInt(measure.FromInt64(a.Weight))
			continue
		}
		set.Rows = append(set.Rows, Household{Weight: a.Weight, Total: total})
		set.Agents++
		hh.addInt(measure.FromInt64(a.Weight))
		if a.Weight > 1 {
			cohort.addInt(measure.FromInt64(a.Weight))
		}
	}
	return set.finish(hh, cohort, absent)
}

// finish sets the counts and the cohort share from the sums.
func (set *RowSet) finish(hh, cohort, absent acc) (RowSet, error) {
	var err error
	if set.Households, err = hh.int64("RowsOf", "households"); err != nil {
		return RowSet{}, err
	}
	if set.Cohort, err = cohort.int64("RowsOf", "cohort"); err != nil {
		return RowSet{}, err
	}
	if set.Absent, err = absent.int64("RowsOf", "absent"); err != nil {
		return RowSet{}, err
	}
	set.CohortShare, err = ratioWide("RowsOf", "cohort_share", cohort.v, hh.v)
	return *set, err
}

// total returns the row total of a column, and ok false where the value does not
// exist.
func (a AgentRow) total(col Column, p Prices) (int64, bool, error) {
	switch col {
	case ColMoney:
		return int64(a.Money), true, nil
	case ColWelfareRatio:
		if !a.HasWelfareRatio { // absent: the value is not read, so it cannot overflow
			return 0, false, nil
		}
		t, err := a.times(a.WelfareRatioBP)
		return t, true, err
	case ColEarnedRatio:
		if !a.HasEarnedRatio {
			return 0, false, nil
		}
		t, err := a.times(a.EarnedRatioBP)
		return t, true, err
	case ColHunger:
		t, err := a.times(a.Hunger)
		return t, true, err
	case ColWealth:
		t, err := a.wealth(p)
		return t, true, err
	}
	return 0, false, wrap("RowsOf", "column", ErrRange)
}

// times returns Weight * v, checked.
func (a AgentRow) times(v int64) (int64, error) {
	t, ok := measure.ToInt64(measure.MulInt64(a.Weight, v))
	if !ok {
		return 0, wrap("RowsOf", "total", ErrOverflow)
	}
	return t, nil
}

// wealth returns the money plus the holdings at the prices of the agent's region,
// checked.
func (a AgentRow) wealth(p Prices) (int64, error) {
	var sum acc
	sum.addInt(measure.FromInt64(int64(a.Money)))
	for _, h := range a.Holdings {
		if h.Quantity != 0 && !p.has(a.Region, h.Commodity) { // a missing price would value the goods at 0, silently
			return 0, wrap("RowsOf", "wealth", ErrRange)
		}
		sum.add(int64(h.Quantity), p.At(a.Region, h.Commodity))
	}
	return sum.int64("RowsOf", "wealth")
}
