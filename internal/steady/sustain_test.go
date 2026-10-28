package steady_test

import (
	"context"
	"errors"
	"math/big"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// ratIs reports whether r is exactly num/den.
func ratIs(r *big.Rat, num, den int64) bool { return r.Cmp(big.NewRat(num, den)) == 0 }

// fracIs reports whether f is exactly num/den.
func fracIs(f steady.Frac, num, den int64) bool {
	return new(big.Rat).SetFrac(f.Num, f.Den).Cmp(big.NewRat(num, den)) == 0
}

// TestSustain_PhiPerShape verifies the sustainable draw a tick of each shape, in
// exact fractions.
//
// Values (spec-resources.md 11.1, 16.1, 16.2; the arithmetic is in each case):
//   - constant finite, the ice field of example B: a stock of 3 000 000 over a
//     horizon of 15 years is 200 000 a year, 200 000 / 640 ticks = 625/2 = 312.5 a
//     tick (the 313 of the text, rounded);
//   - refill, the vats of example B: 2.0 * (30 000 - 15 000) = 30 000 a year,
//     30 000 / 640 = 375/8 a tick;
//   - refill with the seasons of example A (autumn 91 and winter 90 ticks at
//     20 000, the rest 0): sbar = 181 * 20 000 / (365 * 10 000) = 362/365, a year
//     1 * 1 000 * 362/365 = 362 000/365, a tick 362 000 / 365^2 = 14 480/5 329;
//   - refill with a regeneration scale of 12 000: 1.2 times that of the vats;
//   - logistic, the woodland of example A: 0.3 * (25 000 * 25 000 / 50 000) = 3 750
//     a year (its maximum sustainable draw), 3 750/365 = 750/73 a tick;
//   - constant with a rate: 100 + 36 500/10 = 3 750 a year, again 750/73 a tick;
//   - logistic with no capacity: 0.
//
// Requirement: FR-RES-039
func TestSustain_PhiPerShape(t *testing.T) {
	t.Parallel()
	autumnWinter := []steady.Run{{Length: 92, FactorBP: 0}, {Length: 92, FactorBP: 0}, {Length: 91, FactorBP: 20000}, {Length: 90, FactorBP: 20000}}
	tests := []struct {
		name     string
		e        steady.Endow
		year     int64
		num, den int64
	}{
		{"finite constant over the horizon", steady.Endow{Shape: steady.Constant, Stock: 3000000, Horizon: 15, ScaleBP: 10000}, 640, 625, 2},
		{"refill at the plan share", steady.Endow{Shape: steady.Refill, Capacity: 30000, Rate: 20000, PlanShareBP: 5000, ScaleBP: 10000}, 640, 375, 8},
		{"refill with seasons", steady.Endow{Shape: steady.Refill, Capacity: 2000, Rate: 10000, PlanShareBP: 5000, ScaleBP: 10000, Seasons: autumnWinter}, 365, 14480, 5329},
		{"refill scaled", steady.Endow{Shape: steady.Refill, Capacity: 30000, Rate: 20000, PlanShareBP: 5000, ScaleBP: 12000}, 640, 375 * 12, 8 * 10},
		{"logistic at the maximum", steady.Endow{Shape: steady.Logistic, Capacity: 50000, Rate: 3000, PlanShareBP: 5000, ScaleBP: 10000}, 365, 750, 73},
		{"constant with a rate and a stock", steady.Endow{Shape: steady.Constant, Rate: 100, Stock: 36500, Horizon: 10, ScaleBP: 10000}, 365, 750, 73},
		{"constant reserve over one year", steady.Endow{Shape: steady.Constant, Stock: 10, Horizon: 1, ScaleBP: 10000}, 10, 1, 1},
		{"logistic with no capacity", steady.Endow{Shape: steady.Logistic, Rate: 3000, PlanShareBP: 5000, ScaleBP: 10000}, 365, 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.e.Phi(tt.year); !ratIs(got, tt.num, tt.den) {
				t.Errorf("Phi(%d) of %+v = %v, want %d/%d", tt.year, tt.e, got, tt.num, tt.den)
			}
		})
	}
}

// resourceModel is one region of 10 households with an upkeep of 5 of commodity
// 0 a tick, a year of 10 ticks, and one activity of duration 1 that outputs one
// unit of commodity 0 and draws one unit of resource 7 (J 10, so it has no
// capacity row). Labour allows 10 executions a tick; the endowment is the
// parameter.
func resourceModel(e steady.Endow) steady.Model {
	e.Resource, e.Region = 7, 0
	return steady.Model{
		Commodities: 1,
		Regions:     []steady.Region{{Households: 10, Upkeep: []int64{5}}},
		Year:        10,
		Phi:         steady.Ratio{Num: 1, Den: 1},
		Activities: []steady.Activity{{
			Duration: 1, Starts: 10, FactorSum: 10 * 10000,
			Outputs: []steady.Item{{Commodity: 0, Qty: 1}},
			Draws:   []steady.Draw{{Resource: 7, Qty: 1}}, AbundanceBP: 10000,
		}},
		Resources: []steady.Endow{e},
	}
}

// TestSolve_ResourceRowBinds verifies that a resource row below what the basket
// needs binds: the surplus factor is below 1, the row is tight and carries a
// positive rent, and the draw at lambda = 1 exceeds the sustainable draw.
//
// Arithmetic: refill, capacity 100, plan share 5 000, rate 6 000: 0.6 * 50 = 30 a
// year over 10 ticks = 3 a tick. The basket needs 5 executions a tick, so
// lambda* = 3/5 (6 000 bp). One more unit of draw is worth 1/5 of lambda, the
// rent; the draw at lambda = 1 is 3 / (3/5) = 5 against a sustainable 3.
//
// Requirement: FR-RES-040
func TestSolve_ResourceRowBinds(t *testing.T) {
	t.Parallel()
	m := resourceModel(steady.Endow{Shape: steady.Refill, Capacity: 100, Rate: 6000, PlanShareBP: 5000, ScaleBP: 10000})
	out, err := steady.SolveModel(context.Background(), m, 10000)
	if err != nil {
		t.Fatalf("SolveModel() error = %v", err)
	}
	if out.SurplusBP != 6000 {
		t.Errorf("SurplusBP = %d, want 6000", out.SurplusBP)
	}
	if !out.ResTight[0] {
		t.Errorf("ResTight[0] = false, want true")
	}
	if !fracIs(out.ResRent[0], 1, 5) {
		t.Errorf("ResRent[0] = %v/%v, want 1/5", out.ResRent[0].Num, out.ResRent[0].Den)
	}
	if !fracIs(out.ResPhi[0], 3, 1) || !fracIs(out.ResDraw[0], 5, 1) {
		t.Errorf("ResPhi, ResDraw = %v, %v, want 3 and 5", out.ResPhi[0], out.ResDraw[0])
	}
}

// TestSolve_RowSlack verifies that a resource row above what labour allows is
// slack: no rent, not tight, and a headroom (the sustainable draw over the draw at
// lambda = 1) above 1.
//
// Arithmetic: refill, capacity 100, plan share 5 000, rate 40 000: 4 * 50 = 200 a
// year, 20 a tick. Labour binds at 10 executions, lambda* = 2; the draw at
// lambda = 1 is 10 / 2 = 5, so the headroom is 20 / 5 = 4.
//
// Requirement: FR-RES-040
func TestSolve_RowSlack(t *testing.T) {
	t.Parallel()
	m := resourceModel(steady.Endow{Shape: steady.Refill, Capacity: 100, Rate: 40000, PlanShareBP: 5000, ScaleBP: 10000})
	out, err := steady.SolveModel(context.Background(), m, 10000)
	if err != nil {
		t.Fatalf("SolveModel() error = %v", err)
	}
	if out.SurplusBP != 20000 || out.ResTight[0] {
		t.Errorf("SurplusBP, ResTight[0] = %d, %v, want 20000, false", out.SurplusBP, out.ResTight[0])
	}
	if out.ResRent[0].Sign() != 0 {
		t.Errorf("ResRent[0] = %v, want 0", out.ResRent[0])
	}
	if !fracIs(out.ResPhi[0], 20, 1) || !fracIs(out.ResDraw[0], 5, 1) {
		t.Errorf("ResPhi, ResDraw = %v, %v, want 20 and 5", out.ResPhi[0], out.ResDraw[0])
	}
}

// TestSolve_EmptyRow verifies a resource with no sustainable draw: the row is
// 0 on the right, no activity that draws can run, and the surplus is 0.
//
// Requirement: FR-RES-039
func TestSolve_EmptyRow(t *testing.T) {
	t.Parallel()
	m := resourceModel(steady.Endow{Shape: steady.Refill, Capacity: 100, Rate: 0, PlanShareBP: 5000, ScaleBP: 10000})
	out, err := steady.SolveModel(context.Background(), m, 10000)
	if err != nil {
		t.Fatalf("SolveModel() error = %v", err)
	}
	if out.SurplusBP != 0 || !out.ResTight[0] || out.ResPhi[0].Sign() != 0 || out.ResDraw[0].Sign() != 0 {
		t.Errorf("SurplusBP, ResTight, ResPhi, ResDraw = %d, %v, %v, %v, want 0, true, 0, 0", out.SurplusBP, out.ResTight[0], out.ResPhi[0], out.ResDraw[0])
	}
}

// TestSolve_AbundanceScalesTheYield verifies that the abundance at the plan share
// multiplies the mean factor of an activity with draws.
//
// Arithmetic: the slack model of TestSolve_RowSlack with an abundance of 5 000:
// an execution outputs 1 * 0.5 = 1/2 unit, labour allows 10 executions = 5 units,
// so lambda* = 5/5 = 1 (10 000 bp) instead of 2.
//
// Requirement: FR-RES-039
func TestSolve_AbundanceScalesTheYield(t *testing.T) {
	t.Parallel()
	m := resourceModel(steady.Endow{Shape: steady.Refill, Capacity: 100, Rate: 40000, PlanShareBP: 5000, ScaleBP: 10000})
	m.Activities[0].AbundanceBP = 5000
	out, err := steady.SolveModel(context.Background(), m, 10000)
	if err != nil {
		t.Fatalf("SolveModel() error = %v", err)
	}
	if out.SurplusBP != 10000 {
		t.Errorf("SurplusBP = %d, want 10000", out.SurplusBP)
	}
	if got := m.Activities[0].Yield(0); !fracIs(got, 1, 2) {
		t.Errorf("Yield(0) = %v, want 1/2", got)
	}
}

// TestSolve_OffUnchanged verifies that a model with no resource rows builds the
// programme it built before the module: the same rows, columns and layout, with
// an empty list the same as none.
//
// Guard: the module off must leave the programme, and so every table the solver
// makes, as it was; the rows are appended after the capacity rows.
//
// Requirement: FR-RES-039
func TestSolve_OffUnchanged(t *testing.T) {
	t.Parallel()
	base := oneRegion()
	empty := oneRegion()
	empty.Resources = []steady.Endow{}
	p1, l1, err1 := steady.Build(base)
	p2, l2, err2 := steady.Build(empty)
	if err1 != nil || err2 != nil {
		t.Fatalf("Build() errors = %v, %v", err1, err2)
	}
	if !reflect.DeepEqual(p1, p2) || !reflect.DeepEqual(l1, l2) {
		t.Errorf("Build with an empty Resources differs from Build without")
	}
	r1, c1 := steady.Size(base)
	if r1 != 4 || c1 != 7 || len(l1.ResRow) != 0 {
		t.Errorf("Size, ResRow = %d, %d, %v, want 4 rows, 7 columns and no resource row", r1, c1, l1.ResRow)
	}
}

// TestBuild_ResourceRowsAppended verifies the layout with resource rows: they come
// after the capacity rows, a row with a positive draw is written divided by it
// (right-hand side 1) and one with none keeps the plain coefficient.
//
// Requirement: FR-RES-039
func TestBuild_ResourceRowsAppended(t *testing.T) {
	t.Parallel()
	m := resourceModel(steady.Endow{Shape: steady.Refill, Capacity: 100, Rate: 6000, PlanShareBP: 5000, ScaleBP: 10000})
	p, l, err := steady.Build(m)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	// goods row 0, labour row 1, no capacity row (J d = year), resource row 2.
	if p.Rows != 3 || !reflect.DeepEqual(l.ResRow, []int{2}) {
		t.Fatalf("Rows, ResRow = %d, %v, want 3, [2]", p.Rows, l.ResRow)
	}
	if p.RHS[2] != (steady.Ratio{Num: 1, Den: 1}) {
		t.Errorf("RHS[2] = %v, want 1/1", p.RHS[2])
	}
	e := p.Cols[0].Entries[len(p.Cols[0].Entries)-1]
	if e.Row != 2 || e.Wide == nil || !ratIs(e.Wide, 1, 3) {
		t.Errorf("last entry of the activity = %+v, want row 2 with 1/3", e)
	}
	m.Resources[0].Rate = 0
	p, _, err = steady.Build(m)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	e = p.Cols[0].Entries[len(p.Cols[0].Entries)-1]
	if p.RHS[2].Num != 0 || e.Wide != nil || e.Val != (steady.Ratio{Num: 1, Den: 1}) {
		t.Errorf("RHS[2], last entry = %v, %+v, want 0 and the plain 1/1", p.RHS[2], e)
	}
}

// TestBuild_ResourceRejections verifies the data errors and the invariants of a
// resource row.
//
// Requirement: FR-RES-039
func TestBuild_ResourceRejections(t *testing.T) {
	t.Parallel()
	ok := steady.Endow{Shape: steady.Refill, Capacity: 100, Rate: 6000, PlanShareBP: 5000, ScaleBP: 10000}
	tests := []struct {
		name   string
		mutate func(m *steady.Model)
		bad    bool // ErrBadProgramme; else a panic
	}{
		{"negative capacity", func(m *steady.Model) { m.Resources[0].Capacity = -1 }, true},
		{"plan share above one", func(m *steady.Model) { m.Resources[0].PlanShareBP = 10001 }, true},
		{"unknown shape", func(m *steady.Model) { m.Resources[0].Shape = 9 }, true},
		{"constant with no horizon", func(m *steady.Model) { m.Resources[0].Shape, m.Resources[0].Horizon = steady.Constant, 0 }, true},
		{"draw of no units", func(m *steady.Model) { m.Activities[0].Draws[0].Qty = 0 }, true},
		{"negative abundance", func(m *steady.Model) { m.Activities[0].AbundanceBP = -1 }, true},
		{"draw with no row", func(m *steady.Model) { m.Activities[0].Draws[0].Resource = 8 }, false},
		{"row in no region", func(m *steady.Model) { m.Resources[0].Region = 3 }, false},
		{"repeated row", func(m *steady.Model) { m.Resources = append(m.Resources, m.Resources[0]) }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := resourceModel(ok)
			tt.mutate(&m)
			defer func() {
				if r := recover(); r != nil != !tt.bad {
					t.Errorf("Build() panicked = %v, want %v", r != nil, !tt.bad)
				}
			}()
			if _, _, err := steady.Build(m); tt.bad && !errors.Is(err, steady.ErrBadProgramme) {
				t.Errorf("Build() error = %v, want ErrBadProgramme", err)
			}
		})
	}
}

// TestPeakExceeds verifies the season peak rule: the starts of a window against
// the stock at its opening.
//
// Arithmetic: refill, capacity 2 000, plan share 5 000 (S_p = 1 000), rate
// 10 000 a year, seasons autumn and winter only, a year of 365 ticks. An activity
// draws 2 a tick on average (730 a year) and has 10 starts, 8 in a window of the
// first 20 ticks of spring: 730 * 8/10 = 584 drawn there. The window regrows
// nothing (spring is 0), so the stock is S_p = 1 000: 584 does not exceed it. With
// 10 starts all in the window the draw is 730, still below; at a draw of 3 a tick
// (1 095) it exceeds 1 000. A window of the 91 autumn ticks (factor 2.0) adds a
// regrowth of 1 000 * 91 * 20 000 / (365 * 10 000) = 498.6 to the 1 000, so
// 1 498.6 is held: a draw of 4 a tick (1 460) fits and one of 5 (1 825) does not.
// With a rate of 40 000 the regrowth is 1 994.5, the stock held is capped at the
// capacity 2 000, and a draw of 5 a tick (1 825) fits while 6 (2 190) does not.
//
// Requirement: FR-RES-040
func TestPeakExceeds(t *testing.T) {
	t.Parallel()
	seasons := []steady.Run{{Length: 92, FactorBP: 0}, {Length: 92, FactorBP: 0}, {Length: 91, FactorBP: 20000}, {Length: 90, FactorBP: 20000}}
	e := steady.Endow{Shape: steady.Refill, Capacity: 2000, Rate: 10000, PlanShareBP: 5000, ScaleBP: 10000, Seasons: seasons}
	spring := steady.Window{Start: 0, End: 20, Starts: 8}
	autumn := steady.Window{Start: 184, End: 275, Starts: 10}
	tests := []struct {
		name    string
		w       steady.Window
		starts  int64
		perTick int64
		want    bool
	}{
		{"spring window below the stock", spring, 10, 2, false},
		{"spring window above the stock", steady.Window{Start: 0, End: 20, Starts: 10}, 10, 3, true},
		{"autumn window regrows", autumn, 10, 4, false},
		{"autumn window above the regrowth", autumn, 10, 5, true},
		{"no start in the year", spring, 0, 100, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := e.PeakExceeds(365, tt.w, tt.starts, steady.Rat(tt.perTick, 1)); got != tt.want {
				t.Errorf("PeakExceeds(365, %+v, %d, %d) = %v, want %v", tt.w, tt.starts, tt.perTick, got, tt.want)
			}
		})
	}
	// A fast refill would hold more than the capacity, which the stock cannot.
	fast := e
	fast.Rate = 40000
	for perTick, want := range map[int64]bool{5: false, 6: true} {
		if got := fast.PeakExceeds(365, autumn, 10, steady.Rat(perTick, 1)); got != want {
			t.Errorf("PeakExceeds(365, autumn, 10, %d) at rate 40 000 = %v, want %v", perTick, got, want)
		}
	}
	// With no season profile the window regrows at the plain rate.
	flat := e
	flat.Seasons = nil
	if got := flat.PeakExceeds(365, spring, 10, steady.Rat(3, 1)); got {
		t.Errorf("PeakExceeds with a flat profile = true, want false: 3 * 365 * 8/10 = 876 against 1 000 + 20 * 1 000 / 365 = 1 054")
	}
}

// TestPeakExceeds_Boundaries verifies the comparison and the window arithmetic at
// their edges, on a year of 100 ticks.
//
// Arithmetic: refill, capacity 2 000, plan share 5 000 (S_p = 1 000), rate 10 000
// a year. A flat profile over a window of 25 ticks regrows 1 000 * 25/100 = 250,
// so 1 250 is held. One start of one a year drawing 12.5 a tick draws 12.5 * 100 =
// 1 250: equal, so no peak; 12.51 draws 1 251: a peak. With a profile of 50 ticks
// at 0 and 50 at 10 000, the window [25, 75) has 25 ticks at 0 and 25 at 10 000,
// the same 250. A profile of two runs of 10 000 under the window [25, 75) regrows
// 1 000 * 50/100 = 500, so 1 500 is held and 12.51 a tick (1 251) is no peak. A
// profile of 100 ticks at 20 000 gives a window of 25 ticks 500.
//
// Requirement: FR-RES-040
func TestPeakExceeds_Boundaries(t *testing.T) {
	t.Parallel()
	base := steady.Endow{Shape: steady.Refill, Capacity: 2000, Rate: 10000, PlanShareBP: 5000, ScaleBP: 10000}
	half := base
	half.Seasons = []steady.Run{{Length: 50, FactorBP: 0}, {Length: 50, FactorBP: 10000}}
	double := base
	double.Seasons = []steady.Run{{Length: 100, FactorBP: 20000}}
	two := base
	two.Seasons = []steady.Run{{Length: 50, FactorBP: 10000}, {Length: 50, FactorBP: 10000}}
	tests := []struct {
		name    string
		e       steady.Endow
		w       steady.Window
		perTick steady.Frac
		want    bool
	}{
		{"flat, equal to the stock held", base, steady.Window{Start: 0, End: 25, Starts: 1}, steady.Rat(25, 2), false},
		{"flat, a hundredth above", base, steady.Window{Start: 0, End: 25, Starts: 1}, steady.Rat(1251, 100), true},
		{"profile, equal", half, steady.Window{Start: 25, End: 75, Starts: 1}, steady.Rat(25, 2), false},
		{"profile, above", half, steady.Window{Start: 25, End: 75, Starts: 1}, steady.Rat(1251, 100), true},
		{"flat, window past the start", base, steady.Window{Start: 50, End: 75, Starts: 1}, steady.Rat(25, 2), false},
		{"two runs under one window", two, steady.Window{Start: 25, End: 75, Starts: 1}, steady.Rat(1251, 100), false},
		{"double profile, equal to 1 500", double, steady.Window{Start: 0, End: 25, Starts: 1}, steady.Rat(15, 1), false},
		{"double profile, above 1 500", double, steady.Window{Start: 0, End: 25, Starts: 1}, steady.Rat(1501, 100), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.e.PeakExceeds(100, tt.w, 1, tt.perTick); got != tt.want {
				t.Errorf("PeakExceeds(100, %+v, 1, %v) = %v, want %v", tt.w, tt.perTick, got, tt.want)
			}
		})
	}
}

// TestPhi_ConstantWithoutHorizon verifies that a constant stock with no horizon
// adds no reserve term and does not divide by it: only its flow.
//
// Requirement: FR-RES-039
func TestPhi_ConstantWithoutHorizon(t *testing.T) {
	t.Parallel()
	e := steady.Endow{Shape: steady.Constant, Rate: 100, Stock: 5000, ScaleBP: 10000}
	if got := e.Phi(10); !ratIs(got, 10, 1) {
		t.Errorf("Phi(10) of %+v = %v, want 10: 100 a year over 10 ticks", e, got)
	}
}

// TestBuild_ResourceBoundaries verifies that the edges of a resource row are
// accepted and one past them is not: zero quantities, a plan share of 10 000, a
// horizon of 1 and a shape at either end.
//
// Requirement: FR-RES-039
func TestBuild_ResourceBoundaries(t *testing.T) {
	t.Parallel()
	ok := steady.Endow{Shape: steady.Refill, Capacity: 100, Rate: 6000, PlanShareBP: 5000, ScaleBP: 10000}
	tests := []struct {
		name   string
		mutate func(e *steady.Endow)
		bad    bool
	}{
		{"all zero", func(e *steady.Endow) { *e = steady.Endow{Shape: steady.Refill} }, false},
		{"plan share of one", func(e *steady.Endow) { e.PlanShareBP = 10000 }, false},
		{"logistic with no horizon", func(e *steady.Endow) { e.Shape = steady.Logistic }, false},
		{"constant with a horizon of 1", func(e *steady.Endow) { e.Shape, e.Horizon = steady.Constant, 1 }, false},
		{"negative stock", func(e *steady.Endow) { e.Stock = -1 }, true},
		{"negative rate", func(e *steady.Endow) { e.Rate = -1 }, true},
		{"negative scale", func(e *steady.Endow) { e.ScaleBP = -1 }, true},
		{"negative plan share", func(e *steady.Endow) { e.PlanShareBP = -1 }, true},
		{"negative shape", func(e *steady.Endow) { e.Shape = -1 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := ok
			tt.mutate(&e)
			_, _, err := steady.Build(resourceModel(e))
			if got := errors.Is(err, steady.ErrBadProgramme); got != tt.bad || (!tt.bad && err != nil) {
				t.Errorf("Build(%+v) error = %v, want bad = %v", e, err, tt.bad)
			}
		})
	}
}
