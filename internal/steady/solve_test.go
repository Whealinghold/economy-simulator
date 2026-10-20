package steady_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// r is the ratio n/d.
func r(n, d int64) steady.Ratio { return steady.Ratio{Num: n, Den: d} }

// e is the entry n/d in a row.
func e(row int, n, d int64) steady.Entry { return steady.Entry{Row: row, Val: r(n, d)} }

// col is a column with an integer cost.
func col(cost int64, entries ...steady.Entry) steady.Column {
	return steady.Column{Cost: r(cost, 1), Entries: entries}
}

// ints returns the rhs ratios n/1.
func ints(ns ...int64) []steady.Ratio {
	out := make([]steady.Ratio, len(ns))
	for i, n := range ns {
		out[i] = r(n, 1)
	}
	return out
}

// str renders a fraction as num/den, or num when the denominator is 1.
func str(f steady.Frac) string {
	if f.Den.IsInt64() && f.Den.Int64() == 1 {
		return f.Num.String()
	}
	return f.Num.String() + "/" + f.Den.String()
}

// strs renders a slice of fractions.
func strs(fs []steady.Frac) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = str(f)
	}
	return out
}

// render is a byte-comparable rendering of a result.
func render(res steady.Result) string {
	return fmt.Sprint(strs(res.Primal), strs(res.Dual), str(res.Objective), res.Basis, res.Pivots, res.Degenerate, res.Alternative)
}

// oneOverN is the programme "one execution takes n labour ticks and makes one
// unit; the demand is one unit": maximise lambda subject to -x + lambda <= 0
// (goods) and n x <= 1 (labour), columns x then lambda. Its answer is 1/n.
func oneOverN(n int64) steady.Programme {
	return steady.Programme{
		Rows: 2,
		RHS:  ints(0, 1),
		Cols: []steady.Column{col(0, e(0, -1, 1), e(1, n, 1)), col(1, e(0, 1, 1))},
	}
}

// beale is the classic cycling programme: with the largest-coefficient rule it
// cycles forever. Maximise 3/4 x4 - 20 x5 + 1/2 x6 - 6 x7 subject to
// 1/4 x4 - 8 x5 - x6 + 9 x7 <= 0, 1/2 x4 - 12 x5 - 1/2 x6 + 3 x7 <= 0, x6 <= 1.
func beale() steady.Programme {
	return steady.Programme{
		Rows: 3,
		RHS:  ints(0, 0, 1),
		Cols: []steady.Column{
			{Cost: r(3, 4), Entries: []steady.Entry{e(0, 1, 4), e(1, 1, 2)}},
			{Cost: r(-20, 1), Entries: []steady.Entry{e(0, -8, 1), e(1, -12, 1)}},
			{Cost: r(1, 2), Entries: []steady.Entry{e(0, -1, 1), e(1, -1, 2), e(2, 1, 1)}},
			{Cost: r(-6, 1), Entries: []steady.Entry{e(0, 9, 1), e(1, 3, 1)}},
		},
	}
}

// handCase is a programme solved by hand: its exact answer and flags.
type handCase struct {
	name        string
	p           steady.Programme
	primal      []string
	dual        []string
	objective   string
	degenerate  bool
	alternative bool
}

// vertexCases are the programmes with a vertex answer worked in the comments.
func vertexCases() []handCase {
	return []handCase{
		{
			// max 3x+2y; x+y<=4, x+3y<=9, x<=3. Rows 1 and 3 are tight at
			// (3, 1): 3*3+2*1 = 11. Duals: y1+y3 = 3 (column x) and y1 = 2
			// (column y), so y1 = 2, y3 = 1, y2 = 0; 4*2 + 3*1 = 11. Row 2 has
			// slack 9-6 = 3, so the vertex is not degenerate.
			name: "two variables",
			p: steady.Programme{Rows: 3, RHS: ints(4, 9, 3), Cols: []steady.Column{
				col(3, e(0, 1, 1), e(1, 1, 1), e(2, 1, 1)),
				col(2, e(0, 1, 1), e(1, 3, 1)),
			}},
			primal: []string{"3", "1"}, dual: []string{"2", "0", "1"}, objective: "11",
		},
		{
			// max x; x+y<=1, x-y<=1. x enters, the ratios tie at 1 and the
			// lower basic index (the slack of row 0) leaves, so row 1 keeps its
			// slack at 1-1 = 0: degenerate. x = 1, duals (1, 0), objective 1.
			name: "degenerate",
			p: steady.Programme{Rows: 2, RHS: ints(1, 1), Cols: []steady.Column{
				col(1, e(0, 1, 1), e(1, 1, 1)),
				col(0, e(0, 1, 1), e(1, -1, 1)),
			}},
			primal: []string{"1", "0"}, dual: []string{"1", "0"}, objective: "1", degenerate: true,
		},
		{
			// max x+y; x+y<=2: x enters first (lowest index) and takes 2; y has
			// reduced cost 0, so there is a second optimum (0, 2).
			name: "alternative optima",
			p: steady.Programme{Rows: 1, RHS: ints(2), Cols: []steady.Column{
				col(1, e(0, 1, 1)), col(1, e(0, 1, 1)),
			}},
			primal: []string{"2", "0"}, dual: []string{"1"}, objective: "2", alternative: true,
		},
		{
			// x/2 <= 3/4 gives x = 3/2; cost 2/3 gives 2/3 * 3/2 = 1; the dual y
			// solves y/2 = 2/3, y = 4/3, and 3/4 * 4/3 = 1.
			name: "fractions",
			p: steady.Programme{Rows: 1, RHS: []steady.Ratio{r(3, 4)}, Cols: []steady.Column{
				{Cost: r(2, 3), Entries: []steady.Entry{e(0, 1, 2)}},
			}},
			primal: []string{"3/2"}, dual: []string{"4/3"}, objective: "1",
		},
	}
}

// flowCases are the programmes of the solver's own shape: 1/n, Beale and the
// transport case.
func flowCases() []handCase {
	transport := steady.Programme{Rows: 4, RHS: ints(3, 5, 4, 6)}
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			transport.Cols = append(transport.Cols, col(1, e(i, 1, 1), e(2+j, 1, 1)))
		}
	}
	return []handCase{
		{
			// n = 3: lambda = x = 1/3 (3 ticks of labour, one unit made, one unit
			// demanded). Duals: goods 1, labour 1/n = 1/3; 0*1 + 1*(1/3) = 1/3.
			name:   "one activity one commodity n=3",
			p:      oneOverN(3),
			primal: []string{"1/3", "1/3"}, dual: []string{"1", "1/3"}, objective: "1/3",
		},
		{
			name:   "one activity one commodity n=7",
			p:      oneOverN(7),
			primal: []string{"1/7", "1/7"}, dual: []string{"1", "1/7"}, objective: "1/7",
		},
		{
			// Beale: x4 <= x6 from row 2 and x6 <= 1, so 3/4 + 1/2 = 5/4 at
			// x4 = x6 = 1 (x5 = x7 = 0); row 1 keeps a slack of 3/4.
			name:   "beale cycling example",
			p:      beale(),
			primal: []string{"1", "0", "1", "0"}, dual: nil, objective: "5/4",
		},
		{
			// Complete 2x2 transport, one unit of value per unit moved: the most
			// that can move is min(supply 3+5, demand 4+6) = 8.
			name: "unit-cost transport",
			p:    transport, objective: "8", alternative: true,
		},
		{name: "no rows no columns", p: steady.Programme{}, primal: []string{}, dual: []string{}, objective: "0"},
		{
			name:   "zero cost column",
			p:      steady.Programme{Rows: 1, RHS: ints(5), Cols: []steady.Column{col(0, e(0, 1, 1))}},
			primal: []string{"0"}, dual: []string{"0"}, objective: "0", alternative: true,
		},
	}
}

// TestSolve_HandMadeProgrammes verifies the exact primal values, duals,
// objective and flags of programmes small enough to solve by hand.
//
// Requirement: FR-VIA-013, FR-VIA-014, FR-VIA-016
func TestSolve_HandMadeProgrammes(t *testing.T) {
	t.Parallel()
	tests := append(vertexCases(), flowCases()...)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := steady.Solve(context.Background(), tt.p, 1000)
			if err != nil {
				t.Fatalf("Solve(%s) error = %v", tt.name, err)
			}
			if got := str(res.Objective); got != tt.objective {
				t.Errorf("Solve(%s) objective = %s, want %s", tt.name, got, tt.objective)
			}
			if tt.primal != nil {
				if got := strs(res.Primal); !slices.Equal(got, tt.primal) {
					t.Errorf("Solve(%s) primal = %v, want %v", tt.name, got, tt.primal)
				}
			}
			if tt.dual != nil {
				if got := strs(res.Dual); !slices.Equal(got, tt.dual) {
					t.Errorf("Solve(%s) dual = %v, want %v", tt.name, got, tt.dual)
				}
			}
			if res.Degenerate != tt.degenerate || res.Alternative != tt.alternative {
				t.Errorf("Solve(%s) degenerate, alternative = %v, %v, want %v, %v",
					tt.name, res.Degenerate, res.Alternative, tt.degenerate, tt.alternative)
			}
		})
	}
}

// TestSolve_BlandTieBreakLeavesLowestIndex verifies the leaving variable on a
// tied ratio test, through the final basis of the degenerate programme.
//
// Guard: a tie broken the other way still gives the right optimum on this
// programme but a different basis, and with it a different vertex on a
// programme with alternative optima; the basis is what makes the answer a
// function of the column order (FR-VIA-022).
//
// Requirement: FR-VIA-016, FR-VIA-022
func TestSolve_BlandTieBreakLeavesLowestIndex(t *testing.T) {
	t.Parallel()
	p := steady.Programme{Rows: 2, RHS: ints(1, 1), Cols: []steady.Column{
		col(1, e(0, 1, 1), e(1, 1, 1)),
		col(0, e(0, 1, 1), e(1, -1, 1)),
	}}
	res, err := steady.Solve(context.Background(), p, 10)
	if err != nil {
		t.Fatalf("Solve error = %v", err)
	}
	// Variables: 0 x, 1 y, 2 slack of row 0, 3 slack of row 1. The slack of row
	// 0 (index 2) leaves, so row 0 holds x (0) and row 1 keeps its slack (3).
	if want := []int{0, 3}; !slices.Equal(res.Basis, want) {
		t.Errorf("Solve basis = %v, want %v", res.Basis, want)
	}
	if res.Pivots != 1 {
		t.Errorf("Solve pivots = %d, want 1", res.Pivots)
	}
}

// TestSolve_ColumnOrderIsPartOfTheInput verifies that the same two columns in
// the two orders give different vertices when the optima are alternative, so
// the caller's canonical order is what fixes the answer.
//
// Requirement: FR-VIA-022
func TestSolve_ColumnOrderIsPartOfTheInput(t *testing.T) {
	t.Parallel()
	// Column a uses row 0; column b uses rows 0 and 1; both earn 1. Rows:
	// a + b <= 2 and b <= 5. Either column alone can take the 2 units, so the
	// optimum 2 is reached at (a, b) = (2, 0) and at (0, 2); Bland's rule gives
	// the vertex of whichever column has the lower index.
	a := col(1, e(0, 1, 1))
	b := col(1, e(0, 1, 1), e(1, 1, 1))
	tests := []struct {
		name string
		cols []steady.Column
		// index of column a in the order given
		aAt  int
		want string
	}{
		{"a first", []steady.Column{a, b}, 0, "2"},
		{"b first", []steady.Column{b, a}, 1, "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := steady.Solve(context.Background(), steady.Programme{Rows: 2, RHS: ints(2, 5), Cols: tt.cols}, 10)
			if err != nil {
				t.Fatalf("Solve error = %v", err)
			}
			if got := str(res.Primal[tt.aAt]); got != tt.want {
				t.Errorf("Solve(%s) value of column a = %s, want %s", tt.name, got, tt.want)
			}
		})
	}
}

// TestSolve_Errors verifies the failures a caller can branch on.
//
// Requirement: FR-VIA-014, FR-VIA-016
func TestSolve_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		p    steady.Programme
		want error
	}{
		// max x; x - y <= 1: x grows with y and nothing stops it.
		{"unbounded column", steady.Programme{Rows: 1, RHS: ints(1), Cols: []steady.Column{
			col(1, e(0, 1, 1)), col(0, e(0, -1, 1))}}, steady.ErrUnbounded},
		{"unbounded with no rows", steady.Programme{Cols: []steady.Column{col(1)}}, steady.ErrUnbounded},
		{"negative right-hand side", steady.Programme{Rows: 1, RHS: ints(-1), Cols: []steady.Column{col(1, e(0, 1, 1))}}, steady.ErrInfeasibleStart},
		{"rhs of the wrong length", steady.Programme{Rows: 2, RHS: ints(1)}, steady.ErrBadProgramme},
		{"negative rows", steady.Programme{Rows: -1}, steady.ErrBadProgramme},
		{"zero rhs denominator", steady.Programme{Rows: 1, RHS: []steady.Ratio{r(1, 0)}}, steady.ErrBadProgramme},
		{"zero cost denominator", steady.Programme{Cols: []steady.Column{{Cost: r(1, 0)}}}, steady.ErrBadProgramme},
		{"negative entry denominator", steady.Programme{Rows: 1, RHS: ints(1), Cols: []steady.Column{col(1, e(0, 1, -1))}}, steady.ErrBadProgramme},
		{"row out of range", steady.Programme{Rows: 1, RHS: ints(1), Cols: []steady.Column{col(1, e(1, 1, 1))}}, steady.ErrBadProgramme},
		{"negative row", steady.Programme{Rows: 1, RHS: ints(1), Cols: []steady.Column{col(1, e(-1, 1, 1))}}, steady.ErrBadProgramme},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := steady.Solve(context.Background(), tt.p, 100)
			if !errors.Is(err, tt.want) {
				t.Errorf("Solve(%s) error = %v, want %v", tt.name, err, tt.want)
			}
			if res.Primal != nil || res.Dual != nil {
				t.Errorf("Solve(%s) returned a partial result %v with an error", tt.name, render(res))
			}
		})
	}
}

// TestSolve_BealeTerminates verifies that Bland's rule ends on the programme
// that cycles under the largest-coefficient rule, with the right optimum.
//
// Requirement: FR-VIA-016
func TestSolve_BealeTerminates(t *testing.T) {
	t.Parallel()
	res, err := steady.Solve(context.Background(), beale(), 1000)
	if err != nil {
		t.Fatalf("Solve(beale) error = %v", err)
	}
	if got := str(res.Objective); got != "5/4" {
		t.Errorf("Solve(beale) objective = %s, want 5/4", got)
	}
}

// countCtx is a context that reports cancellation from its n+1th Err call on.
type countCtx struct {
	context.Context
	left int
}

// Err counts down and then reports context.Canceled.
func (c *countCtx) Err() error {
	if c.left <= 0 {
		return context.Canceled
	}
	c.left--
	return nil
}

// TestSolve_PivotCapAndContext verifies that the pivot cap and a cancelled
// context each return their error and no result, at the first pivot and at a
// later one.
//
// Requirement: FR-VIA-017
func TestSolve_PivotCapAndContext(t *testing.T) {
	t.Parallel()
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	// oneOverN(3) needs two pivots, so a cap of 0 stops before the first, a cap
	// of 1 before the second and a cap of 2 is enough. The tableau is 2 rows by
	// 2 structural + 2 slack columns.
	tests := []struct {
		name      string
		ctx       context.Context
		max       int64
		wantErr   error
		wantLimit *steady.LimitError
	}{
		{"cap before the first pivot", context.Background(), 0, steady.ErrLimit, &steady.LimitError{Pivots: 0, Rows: 2, Cols: 4}},
		{"cap before a later pivot", context.Background(), 1, steady.ErrLimit, &steady.LimitError{Pivots: 1, Rows: 2, Cols: 4}},
		{"cap exactly enough", context.Background(), 2, nil, nil},
		{"cancelled before the first pivot", cancelled, 100, context.Canceled, nil},
		{"cancelled before a later pivot", &countCtx{Context: context.Background(), left: 1}, 100, context.Canceled, nil},
		{"cancelled before the optimum is declared", &countCtx{Context: context.Background(), left: 2}, 100, context.Canceled, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := steady.Solve(tt.ctx, oneOverN(3), tt.max)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Solve(%s) error = %v, want %v", tt.name, err, tt.wantErr)
			}
			if tt.wantErr == nil {
				return
			}
			if res.Primal != nil {
				t.Errorf("Solve(%s) returned a partial result with an error", tt.name)
			}
			var le *steady.LimitError
			if tt.wantLimit != nil {
				if !errors.As(err, &le) || *le != *tt.wantLimit {
					t.Errorf("Solve(%s) error = %#v, want %#v", tt.name, le, tt.wantLimit)
				}
				if !strings.Contains(err.Error(), "pivot limit") {
					t.Errorf("Solve(%s) message = %q, want it to name the pivot limit", tt.name, err.Error())
				}
			}
		})
	}
}

// TestSolve_IsDeterministic verifies that the same programme solved twice gives
// the same rendering, bit for bit.
//
// Requirement: FR-VIA-020
func TestSolve_IsDeterministic(t *testing.T) {
	t.Parallel()
	for name, p := range map[string]steady.Programme{"beale": beale(), "one over n": oneOverN(7)} {
		a, errA := steady.Solve(context.Background(), p, 1000)
		b, errB := steady.Solve(context.Background(), p, 1000)
		if errA != nil || errB != nil {
			t.Fatalf("Solve(%s) errors = %v, %v", name, errA, errB)
		}
		if ra, rb := render(a), render(b); ra != rb {
			t.Errorf("Solve(%s) twice = %s then %s, want equal", name, ra, rb)
		}
	}
}

// TestSolve_LargeCoefficientsStayExact verifies that data whose products exceed
// int64 solve exactly, the reason the package computes in unbounded integers.
//
// Requirement: FR-VIA-015
func TestSolve_LargeCoefficientsStayExact(t *testing.T) {
	t.Parallel()
	// max x; M x <= M-1 with M = MaxInt64: x = (M-1)/M, an exact fraction whose
	// numerator and denominator are near 2^63, and the dual is 1/M.
	m := int64(math.MaxInt64)
	p := steady.Programme{Rows: 1, RHS: []steady.Ratio{r(m-1, 1)}, Cols: []steady.Column{col(1, e(0, m, 1))}}
	res, err := steady.Solve(context.Background(), p, 10)
	if err != nil {
		t.Fatalf("Solve error = %v", err)
	}
	if got, want := str(res.Primal[0]), fmt.Sprintf("%d/%d", m-1, m); got != want {
		t.Errorf("Solve primal = %s, want %s", got, want)
	}
	if got, want := str(res.Dual[0]), fmt.Sprintf("1/%d", m); got != want {
		t.Errorf("Solve dual = %s, want %s", got, want)
	}
}
