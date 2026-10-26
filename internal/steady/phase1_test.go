package steady_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// wantsOneOverN is oneOverN with a fixed term: one execution takes one labour
// tick and makes one unit, the demand is lambda units and w units are wanted
// whatever lambda is. Maximise lambda subject to  -x + lambda <= -w  (goods) and
// x <= h (labour).
func wantsOneOverN(w, h steady.Ratio) steady.Programme {
	return steady.Programme{
		Rows: 2,
		RHS:  []steady.Ratio{{Num: -w.Num, Den: w.Den}, h},
		Cols: []steady.Column{col(0, e(0, -1, 1), e(1, 1, 1)), col(1, e(0, 1, 1))},
	}
}

// TestSolve_TwoPhaseHandChecked verifies a 2x2 programme with a negative
// right-hand side against its answer worked by hand, as fractions.
//
// Maximise lambda; -x + lambda <= -1/2 and x <= 4. The goods row needs
// x >= lambda + 1/2 and x is at most 4, so x = 4 and lambda = 4 - 1/2 = 7/2.
// Duals: column lambda gives y1 >= 1 and column x gives -y1 + y2 >= 0, so
// y1 = y2 = 1; the dual objective is (-1/2) * 1 + 4 * 1 = 7/2, equal to the
// primal. The slack basis starts infeasible (-1/2 < 0), so this runs the
// first phase.
//
// Requirement: FR-VIA-055, FR-VIA-014
func TestSolve_TwoPhaseHandChecked(t *testing.T) {
	t.Parallel()
	res, err := steady.Solve(context.Background(), wantsOneOverN(r(1, 2), r(4, 1)), 100)
	if err != nil {
		t.Fatalf("Solve(1/2 wanted, 4 labour) error = %v, want nil", err)
	}
	if got, want := strs(res.Primal), []string{"4", "7/2"}; !slices.Equal(got, want) {
		t.Errorf("Solve().Primal = %v, want %v", got, want)
	}
	if got, want := strs(res.Dual), []string{"1", "1"}; !slices.Equal(got, want) {
		t.Errorf("Solve().Dual = %v, want %v", got, want)
	}
	if got := str(res.Objective); got != "7/2" {
		t.Errorf("Solve().Objective = %s, want 7/2", got)
	}
	for _, b := range res.Basis {
		if b >= 2+2 {
			t.Errorf("Solve().Basis = %v, want no artificial variable (index at least 4)", res.Basis)
		}
	}
}

// TestSolve_InfeasibleFixedTerm verifies that a fixed term the capacity cannot
// meet is ErrInfeasible and returns no result.
//
// With 5 units wanted and at most 4 made, -x + lambda <= -5 needs x >= 5 + lambda
// and x <= 4: no point. The first phase ends at -1.
//
// Requirement: FR-VIA-055
func TestSolve_InfeasibleFixedTerm(t *testing.T) {
	t.Parallel()
	res, err := steady.Solve(context.Background(), wantsOneOverN(r(5, 1), r(4, 1)), 100)
	if !errors.Is(err, steady.ErrInfeasible) {
		t.Fatalf("Solve(5 wanted, 4 labour) error = %v, want ErrInfeasible", err)
	}
	if res.Primal != nil || res.Dual != nil {
		t.Errorf("Solve(5 wanted, 4 labour) = %v with an error, want no result", res)
	}
}

// TestSolve_DegenerateArtificialLeavesBasis verifies that an artificial variable
// left in the basis at 0 by the first phase is pivoted out, and that the answer
// is the original programme's, with the pivots counted.
//
// Pinned point: maximise -5/4 x subject to -x <= -1/3 and 3x <= 1. The two rows
// leave x = 1/3 as the only feasible point. The first phase pivots x into the
// second row (one pivot) and ends at 0 with the artificial of the first row
// still basic at 0; it is pivoted out onto the slack of that row (a second
// pivot), and the second phase makes the other slack basic (a third). The
// primal is x = 1/3, the objective -5/4 * 1/3 = -5/12, and the dual (5/4, 0)
// has -1/3 * 5/4 = -5/12 and satisfies -y1 + 3 y2 = -5/4 for the column of x.
//
// Repeated bound: maximise -x; -x <= -1 twice and x <= 3, so x = 1, objective -1.
//
// Requirement: FR-VIA-055, FR-VIA-014
func TestSolve_DegenerateArtificialLeavesBasis(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		p         steady.Programme
		primal    []string
		dual      []string
		objective string
		pivots    int64
	}{
		{
			"pinned point",
			steady.Programme{Rows: 2, RHS: []steady.Ratio{r(-1, 3), r(1, 1)}, Cols: []steady.Column{{Cost: r(-5, 4), Entries: []steady.Entry{e(0, -1, 1), e(1, 3, 1)}}}},
			[]string{"1/3"}, []string{"5/4", "0"}, "-5/12", 3,
		},
		{
			"repeated bound",
			steady.Programme{Rows: 3, RHS: ints(-1, -1, 3), Cols: []steady.Column{{Cost: r(-1, 1), Entries: []steady.Entry{e(0, -1, 1), e(1, -1, 1), e(2, 1, 1)}}}},
			[]string{"1"}, nil, "-1", 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := steady.Solve(context.Background(), tt.p, 100)
			if err != nil {
				t.Fatalf("Solve() error = %v, want nil", err)
			}
			if got := strs(res.Primal); !slices.Equal(got, tt.primal) {
				t.Errorf("Solve().Primal = %v, want %v", got, tt.primal)
			}
			if got := str(res.Objective); got != tt.objective {
				t.Errorf("Solve().Objective = %s, want %s", got, tt.objective)
			}
			if tt.dual != nil && !slices.Equal(strs(res.Dual), tt.dual) {
				t.Errorf("Solve().Dual = %v, want %v", strs(res.Dual), tt.dual)
			}
			if tt.pivots != 0 && res.Pivots != tt.pivots {
				t.Errorf("Solve().Pivots = %d, want %d", res.Pivots, tt.pivots)
			}
			if !res.Degenerate {
				t.Errorf("Solve().Degenerate = false, want true (a basic variable is 0)")
			}
			cols := len(tt.p.Cols) + tt.p.Rows
			for _, b := range res.Basis {
				if b >= cols {
					t.Errorf("Solve().Basis = %v, want no artificial variable (index at least %d)", res.Basis, cols)
				}
			}
		})
	}
}

// TestBuild_FixedTermWritesNegativeRHS verifies that a fixed term W(c) is the
// right-hand side -W(c) of goods row c, reduced, and that a term of 0 or a
// missing one leaves 0.
//
// Requirement: FR-VIA-055
func TestBuild_FixedTermWritesNegativeRHS(t *testing.T) {
	t.Parallel()
	m := steady.Model{
		Commodities: 3, Year: 10, Phi: r(1, 1),
		Regions: []steady.Region{{Households: 6}},
		Fixed:   []steady.Ratio{r(6, 4), r(0, 1)},
	}
	p, l, err := steady.Build(m)
	if err != nil {
		t.Fatalf("Build(fixed 3/2, 0) error = %v, want nil", err)
	}
	got := fmt.Sprint(p.RHS[:3], p.RHS[l.LabourRow(0)])
	if want := "[{-3 2} {0 1} {0 1}] {6 1}"; got != want {
		t.Errorf("Build(fixed 3/2, 0).RHS = %s, want %s (goods rows -W, labour phi*H)", got, want)
	}
}

// TestSolve_ZeroFixedUnchanged verifies that a model whose fixed terms are all 0
// solves to exactly the result of the same model with no fixed term, pivots
// included, so no warm document that was solved before the first phase existed
// changes.
//
// Guard: the first phase must not run, and so must not change the pivot path,
// when no right-hand side is negative. testdata/chains/warm-*.toml are bytes of
// this path (TestWarmDocuments_Bytes in the root package).
func TestSolve_ZeroFixedUnchanged(t *testing.T) {
	t.Parallel()
	ch := bread(breadQty{1500, 60, 100, 12, 6, 12, 18, 12, 18})
	m := ch.model([]string{"south", "north"}, []int64{1000, 1000}, full)
	base, err := steady.SolveModel(context.Background(), m, 10000)
	if err != nil {
		t.Fatalf("SolveModel(no fixed term) error = %v, want nil", err)
	}
	m.Fixed = make([]steady.Ratio, m.Commodities)
	for i := range m.Fixed {
		m.Fixed[i] = r(0, 1+int64(i))
	}
	zero, err := steady.SolveModel(context.Background(), m, 10000)
	if err != nil {
		t.Fatalf("SolveModel(zero fixed terms) error = %v, want nil", err)
	}
	if a, b := fmt.Sprintf("%+v", base), fmt.Sprintf("%+v", zero); a != b {
		t.Errorf("SolveModel(zero fixed terms) = %s, want the result with none, %s", b, a)
	}
}

// TestSolve_PivotCapCountsBothPhases verifies that the pivot cap spans the first
// and the second phase: the pivots a solve with a negative term needs succeed at
// the cap and fail one below it, with no partial result.
//
// Requirement: FR-VIA-022
func TestSolve_PivotCapCountsBothPhases(t *testing.T) {
	t.Parallel()
	p := wantsOneOverN(r(1, 2), r(4, 1))
	res, err := steady.Solve(context.Background(), p, 100)
	if err != nil {
		t.Fatalf("Solve(1/2 wanted) error = %v, want nil", err)
	}
	if res.Pivots < 2 {
		t.Fatalf("Solve(1/2 wanted).Pivots = %d, want at least 2 (a first phase and a second)", res.Pivots)
	}
	if _, err := steady.Solve(context.Background(), p, res.Pivots); err != nil {
		t.Errorf("Solve(cap = %d pivots) error = %v, want nil", res.Pivots, err)
	}
	for cap := range res.Pivots {
		got, err := steady.Solve(context.Background(), p, cap)
		var le *steady.LimitError
		if !errors.As(err, &le) || !errors.Is(err, steady.ErrLimit) {
			t.Errorf("Solve(cap = %d) error = %v, want a *LimitError", cap, err)
		}
		if got.Primal != nil {
			t.Errorf("Solve(cap = %d) returned a partial result %v", cap, got)
		}
	}
}
