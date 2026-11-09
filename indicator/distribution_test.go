package indicator_test

import (
	"errors"
	"math"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// moneyAgents are the cohorts of 18.8: weights 1, 2, 4, 1 holding 0, 40, 160, 60
// in all (so 20 and 40 each in the two cohorts).
func moneyAgents() []indicator.AgentRow {
	return []indicator.AgentRow{
		{Weight: 1, Money: 0}, {Weight: 2, Money: 40}, {Weight: 4, Money: 160}, {Weight: 1, Money: 60},
	}
}

// ratioAgents are the welfare ratios of 18.8: weights 1, 2, 4, 1 at -5 000,
// 8 000, 10 000 and 25 000 per household.
func ratioAgents() []indicator.AgentRow {
	r := []int64{-5000, 8000, 10_000, 25_000}
	w := []int64{1, 2, 4, 1}
	var out []indicator.AgentRow
	for i := range r {
		out = append(out, indicator.AgentRow{Weight: w[i], WelfareRatioBP: r[i], HasWelfareRatio: true, EarnedRatioBP: r[i] - 1000, HasEarnedRatio: true})
	}
	return out
}

func rowsOf(t *testing.T, st []indicator.AgentRow, col indicator.Column, opt indicator.RowOpt) indicator.RowSet {
	t.Helper()
	set, err := indicator.RowsOf(st, col, opt)
	if err != nil {
		t.Fatalf("RowsOf(column %d) error = %v, want nil", col, err)
	}
	return set
}

// TestRowsOf_ExcludesPseudoAgentsAndAbsent verifies 4.5: a pseudo-agent has no
// row (the type has no place for one); External is dropped only when asked; an
// agent with no upkeep has no welfare ratio and is counted in Absent with its
// weight; a state column is weight times the value; and the coverage is reported.
//
// Requirement: FR-IND-026, FR-IND-027
func TestRowsOf_ExcludesPseudoAgentsAndAbsent(t *testing.T) {
	st := append(ratioAgents(), indicator.AgentRow{Weight: 3, External: true, WelfareRatioBP: 100, HasWelfareRatio: true},
		indicator.AgentRow{Weight: 5}, indicator.AgentRow{Weight: 2, Hunger: 4})
	all := rowsOf(t, st, indicator.ColWelfareRatio, indicator.RowOpt{})
	if all.Agents != 5 || all.Households != 11 || all.Absent != 7 || all.Cohort != 9 {
		t.Errorf("RowsOf(all) = agents %d households %d absent %d cohort %d, want 5, 11 (the external agent is a household), 7 (weights 5 + 2 without a ratio), 9", all.Agents, all.Households, all.Absent, all.Cohort)
	}
	if all.CohortShare.Value != 8182 {
		t.Errorf("RowsOf(all).CohortShare = %+v, want rdiv(10 000 * 9, 11) = 8 182", all.CohortShare)
	}
	if all.Rows[1] != (indicator.Household{Weight: 2, Total: 16_000}) {
		t.Errorf("RowsOf() row 1 = %+v, want weight 2 total 16 000: a state is weight times the value", all.Rows[1])
	}
	noExt := rowsOf(t, st, indicator.ColWelfareRatio, indicator.RowOpt{ExcludeExternal: true})
	if noExt.Households != 8 || noExt.Agents != 4 || noExt.Absent != 7 {
		t.Errorf("RowsOf(ExcludeExternal) = households %d agents %d absent %d, want 8, 4, 7", noExt.Households, noExt.Agents, noExt.Absent)
	}
	if h := rowsOf(t, st, indicator.ColHunger, indicator.RowOpt{}); h.Absent != 0 || h.Households != 18 || h.Rows[len(h.Rows)-1].Total != 8 {
		t.Errorf("RowsOf(hunger) = %+v, want every agent a row, the last of total 2 * 4", h)
	}
	if e := rowsOf(t, st, indicator.ColEarnedRatio, indicator.RowOpt{}); e.Absent != 10 || e.Households != 8+3-3 {
		t.Errorf("RowsOf(earned) = absent %d households %d, want the earned ratio of 3 + 5 + 2 absent and 8 households", e.Absent, e.Households)
	}
}

// TestRowsOf_Wealth verifies that wealth is money plus holdings valued at the
// prices of the agent's own region, and that a table is needed only for it.
//
// Requirement: FR-IND-026
func TestRowsOf_Wealth(t *testing.T) {
	w, _, _ := workedWindow(t)
	p := valleyTable(t, w, alphaP, betaP)
	st := []indicator.AgentRow{
		{Region: alpha, Weight: 1, Money: 100, Holdings: []indicator.Holding{{Commodity: grain, Quantity: 3}, {Commodity: loaf, Quantity: 1}}},
		{Region: beta, Weight: 2, Money: 10, Holdings: []indicator.Holding{{Commodity: grain, Quantity: 3}}},
	}
	got := rowsOf(t, st, indicator.ColWealth, indicator.RowOpt{Prices: p})
	if got.Rows[0].Total != 155 || got.Rows[1].Total != 49 {
		t.Errorf("RowsOf(wealth) totals = %d, %d, want 100 + 30 + 25 = 155 and 10 + 39 = 49 (beta grain is 13)", got.Rows[0].Total, got.Rows[1].Total)
	}
	m := rowsOf(t, st, indicator.ColMoney, indicator.RowOpt{})
	if m.Rows[0].Total != 100 || m.Rows[1].Total != 10 {
		t.Errorf("RowsOf(money) totals = %d, %d, want 100, 10: money needs no table", m.Rows[0].Total, m.Rows[1].Total)
	}
}

// TestRowsOf_Misuse verifies the sentinels of RowsOf.
//
// Requirement: FR-IND-026
func TestRowsOf_Misuse(t *testing.T) {
	if _, err := indicator.RowsOf([]indicator.AgentRow{{Weight: 0}}, indicator.ColMoney, indicator.RowOpt{}); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("RowsOf(weight 0) error = %v, want ErrRange", err)
	}
	if _, err := indicator.RowsOf([]indicator.AgentRow{{Weight: 1}}, indicator.Column(0), indicator.RowOpt{}); !errors.Is(err, indicator.ErrRange) {
		t.Errorf("RowsOf(column 0) error = %v, want ErrRange", err)
	}
	big := []indicator.AgentRow{{Weight: 4, WelfareRatioBP: math.MaxInt64 / 2, HasWelfareRatio: true}}
	if _, err := indicator.RowsOf(big, indicator.ColWelfareRatio, indicator.RowOpt{}); !errors.Is(err, indicator.ErrOverflow) {
		t.Errorf("RowsOf(weight * ratio past int64) error = %v, want ErrOverflow", err)
	}
	huge := []indicator.AgentRow{{Weight: 1, Money: math.MaxInt64, Holdings: []indicator.Holding{{Quantity: 5}}}}
	p, _ := indicator.NewPrices([32]byte{}, 0, 0, 1, 1, []economysim.Price{9})
	if _, err := indicator.RowsOf(huge, indicator.ColWealth, indicator.RowOpt{Prices: p}); !errors.Is(err, indicator.ErrOverflow) {
		t.Errorf("RowsOf(wealth past int64) error = %v, want ErrOverflow", err)
	}
	if _, err := indicator.RowsOf([]indicator.AgentRow{{Weight: math.MaxInt64}, {Weight: math.MaxInt64}}, indicator.ColMoney, indicator.RowOpt{}); !errors.Is(err, indicator.ErrOverflow) {
		t.Errorf("RowsOf(households past int64) error = %v, want ErrOverflow", err)
	}
	if got := rowsOf(t, nil, indicator.ColMoney, indicator.RowOpt{}); got.Households != 0 || got.CohortShare.OK {
		t.Errorf("RowsOf(nothing) = %+v, want no households and a cohort share undefined", got)
	}
}

// TestWelfareDistribution_NoGini verifies 18.8 and 18.9: the mean 9 500, the
// quantiles -5 000, 10 000 and 25 000, the share below one 3 750, negative
// ratios allowed; the agents without a ratio are Absent and the cohort share is
// reported; the earned ratio has its own distribution; and the mean of ratios
// (12 500) is not the aggregate ratio of the sums (8 000).
//
// Requirement: FR-IND-039
func TestWelfareDistribution_NoGini(t *testing.T) {
	st := append(ratioAgents(), indicator.AgentRow{Weight: 2})
	welfare, earned := rowsOf(t, st, indicator.ColWelfareRatio, indicator.RowOpt{}), rowsOf(t, st, indicator.ColEarnedRatio, indicator.RowOpt{})
	got, err := indicator.WelfareDistribution(welfare, earned, 9000)
	if err != nil {
		t.Fatalf("WelfareDistribution() error = %v, want nil", err)
	}
	g := got.Welfare
	if !g.OK || g.Mean != 9500 || g.P10 != -5000 || g.P25 != 8000 || g.P50 != 10_000 || g.P75 != 10_000 || g.P90 != 25_000 {
		t.Errorf("WelfareDistribution().Welfare = %+v, want mean 9 500, P10 -5 000, P25 8 000, P50 10 000, P75 10 000, P90 25 000", g)
	}
	if g.Below.Value != 3750 || g.BelowTarget.Value != 3750 || g.Absent != 2 || g.Cohort != 6 || g.CohortShare.Value != 7500 {
		t.Errorf("WelfareDistribution().Welfare = %+v, want 3 750 below one and below 9 000, absent 2, cohort 6 of 8 = 7 500", g)
	}
	if len(g.Warnings) != 2 || g.Warnings[0] != indicator.WarnCohort || g.Warnings[1] != indicator.WarnAbsent {
		t.Errorf("WelfareDistribution().Welfare.Warnings = %v, want cohort and absent", g.Warnings)
	}
	if e := got.Earned; e.Mean != 8500 || e.P90 != 24_000 || e.Below.Value != 8750 {
		t.Errorf("WelfareDistribution().Earned = %+v, want the earned ratio 1 000 lower: mean 8 500, P90 24 000, 8 750 below one", e)
	}
	if hi, _ := indicator.WelfareDistribution(welfare, earned, 25_001); hi.Welfare.BelowTarget.Value != 10_000 {
		t.Errorf("WelfareDistribution(target 25 001).BelowTarget = %+v, want every household", hi.Welfare.BelowTarget)
	}

	// 18.9: ratios 20 000 and 5 000 for (income, cost) (20, 10) and (20, 40).
	pair := rowsOf(t, []indicator.AgentRow{{Weight: 1, WelfareRatioBP: 20_000, HasWelfareRatio: true}, {Weight: 1, WelfareRatioBP: 5000, HasWelfareRatio: true}}, indicator.ColWelfareRatio, indicator.RowOpt{})
	d, err := indicator.WelfareDistribution(pair, indicator.RowSet{}, 10_000)
	agg, ok, _ := indicator.Quotient(10_000*40, 50)
	if err != nil || d.Welfare.Mean != 12_500 || !ok || agg != 8000 || d.Welfare.Mean == agg {
		t.Errorf("mean of ratios = %d (%v), aggregate ratio = %d, want 12 500 against 8 000: they differ by the covariance", d.Welfare.Mean, err, agg)
	}
	if d.Earned.OK {
		t.Errorf("WelfareDistribution(no earned rows).Earned = %+v, want undefined", d.Earned)
	}
}

// TestInequality_CohortShare verifies 18.8: the Gini of money 2 788, Top(1 000)
// 2 308, Bottom(4 000) 3 077, P90 60 and P10 0 so that P90 / P10 is undefined,
// and the cohort share 7 500 for the rows of weight above 1 against 0 for the same
// holdings as agents of weight 1; a negative balance is an error; and an empty
// set is undefined.
//
// Requirement: FR-IND-040, FR-IND-026
func TestInequality_CohortShare(t *testing.T) {
	got, err := indicator.Inequality(rowsOf(t, moneyAgents(), indicator.ColMoney, indicator.RowOpt{}))
	if err != nil {
		t.Fatalf("Inequality() error = %v, want nil", err)
	}
	if !got.GiniOK || got.Gini != 2788 || !got.SharesOK || got.Top != 2308 || got.Bottom != 3077 || got.P10 != 0 || got.P90 != 60 || got.P90P10OK {
		t.Errorf("Inequality() = %+v, want Gini 2 788, Top 2 308, Bottom 3 077, P90 60, P10 0 and no ratio", got)
	}
	if got.CohortShare.Value != 7500 || got.Cohort != 6 || got.Households != 8 || got.Agents != 4 || len(got.Warnings) != 1 || got.Warnings[0] != indicator.WarnCohort {
		t.Errorf("Inequality() coverage = %+v, want cohort 6 of 8 households = 7 500, warned", got.Coverage)
	}

	// The same four agents at scale 1 (weights 1): no cohort, so CohortShare is 0.
	one := []indicator.AgentRow{{Weight: 1, Money: 0}, {Weight: 1, Money: 40}, {Weight: 1, Money: 160}, {Weight: 1, Money: 60}}
	g1, err := indicator.Inequality(rowsOf(t, one, indicator.ColMoney, indicator.RowOpt{}))
	if err != nil || g1.Cohort != 0 || !g1.CohortShare.OK || g1.CohortShare.Value != 0 || len(g1.Warnings) != 0 {
		t.Errorf("Inequality(scale 1) = %+v, %v, want no cohort and no warning", g1, err)
	}

	// A P10 above 0 gives the ratio: rows 10, 20, 30 of weight 1: P10 10, P90 30, 30 000.
	r, err := indicator.Inequality(rowsOf(t, []indicator.AgentRow{{Weight: 1, Money: 10}, {Weight: 1, Money: 20}, {Weight: 1, Money: 30}}, indicator.ColMoney, indicator.RowOpt{}))
	if err != nil || !r.P90P10OK || r.P90P10 != 30_000 {
		t.Errorf("Inequality(10, 20, 30) = %+v, %v, want P90 / P10 = 30 000", r, err)
	}

	if _, err := indicator.Inequality(rowsOf(t, []indicator.AgentRow{{Weight: 1, Money: -5}, {Weight: 1, Money: 9}}, indicator.ColMoney, indicator.RowOpt{})); !errors.Is(err, indicator.ErrNegativeValue) {
		t.Errorf("Inequality(a negative balance) error = %v, want ErrNegativeValue", err)
	}
	if e, err := indicator.Inequality(indicator.RowSet{}); err != nil || e.GiniOK || e.SharesOK || e.P90P10OK {
		t.Errorf("Inequality(nothing) = %+v, %v, want everything undefined and no error", e, err)
	}
	if z, err := indicator.Inequality(rowsOf(t, []indicator.AgentRow{{Weight: 2, Money: 0}}, indicator.ColMoney, indicator.RowOpt{})); err != nil || z.GiniOK || z.SharesOK {
		t.Errorf("Inequality(all zero) = %+v, %v, want undefined: nobody holds anything", z, err)
	}
}

// TestHouseholdIncome_NotAvailableInVersion1 verifies that the preset returns
// OK false with the window it was asked about and no figure (D574, N-130 item 1).
//
// Requirement: FR-IND-050
func TestHouseholdIncome_NotAvailableInVersion1(t *testing.T) {
	w, _, _ := peopleWindow(t)
	got, err := indicator.HouseholdIncome(w)
	if err != nil {
		t.Fatalf("HouseholdIncome() error = %v, want nil", err)
	}
	if got.OK || got.Income != 0 || got.Relief != 0 || got.Cost != 0 || got.Ratio != 0 || got.Gap != 0 || got.From != 10 || got.To != 12 || got.ConfigHash != w.ConfigHash {
		t.Errorf("HouseholdIncome() = %+v, want OK false, no figure, the window 10 to 12", got)
	}
}

// TestHouseholdIncome_BookedFromRoleRowFields is the second clause of FR-IND-050
// for the day the income fields exist: the stationary world of 18.9 must give an
// income of 50, a cost equal to it and a Gap of 0.
//
// Requirement: FR-IND-050
func TestHouseholdIncome_BookedFromRoleRowFields(t *testing.T) {
	t.Skip("N-130 item 1: RoleRow has no IncomeLast, ReliefLast or CostLast yet; owner Stage 3 (D574)")
}
