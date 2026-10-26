package steady

import (
	"context"
	"errors"
	"math/big"
	"slices"
	"testing"
)

// signedProgramme decodes up to 4 rows and 4 columns of small rationals whose
// right-hand sides run from -5 to 5, so about half the rows need the first
// phase.
func signedProgramme(b *byteReader) Programme {
	p := randomProgramme(b)
	for i := range p.RHS {
		p.RHS[i].Num -= 5
	}
	return p
}

// FuzzSolveTwoPhase verifies, over small random programmes with right-hand
// sides of either sign, that the solver never panics, that the only failures it
// reports are an infeasible or an unbounded programme, and that every result
// passes the certificate against the original programme.
//
// Requirement: FR-VIA-014, FR-VIA-022, FR-VIA-055
func FuzzSolveTwoPhase(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{3, 3, 4, 9, 3, 10, 2, 8, 7, 5, 3, 1, 6, 9, 2})
	f.Add([]byte{2, 1, 0, 0, 200, 7, 7, 1, 1})
	f.Add([]byte{1, 1, 0, 0, 5, 5, 5})
	f.Add([]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255})
	f.Fuzz(func(t *testing.T, data []byte) {
		p := signedProgramme(&byteReader{data: data})
		res, err := Solve(context.Background(), p, 100_000)
		switch {
		case errors.Is(err, ErrUnbounded), errors.Is(err, ErrInfeasible):
			if res.Primal != nil {
				t.Fatalf("Solve(%v) = %v with error %v, want no result", p, res, err)
			}
		case err != nil:
			t.Fatalf("Solve(%v) error = %v, want nil, ErrInfeasible or ErrUnbounded", p, err)
		default:
			if cerr := certify(p, res); cerr != nil {
				t.Fatalf("certify(Solve(%v)) = %v", p, cerr)
			}
			for _, b := range res.Basis {
				if b >= len(p.Cols)+p.Rows {
					t.Fatalf("Solve(%v).Basis = %v, want no artificial variable", p, res.Basis)
				}
			}
		}
	})
}

// TestSolve_CertificateHoldsWithWants verifies that the certificate of a
// result from the first phase is checked against the original programme, with
// its negative right-hand side, and that it fails when a dual is tampered with.
//
// Requirement: FR-VIA-014
func TestSolve_CertificateHoldsWithWants(t *testing.T) {
	t.Parallel()
	m := Model{
		Commodities: 1, Year: 10, Phi: Ratio{Num: 1, Den: 1},
		Regions:    []Region{{Households: 8, Upkeep: []int64{1}}},
		Fixed:      []Ratio{{Num: 3, Den: 2}},
		Activities: []Activity{{Duration: 1, Starts: 10, FactorSum: 100000, Outputs: []Item{{Commodity: 0, Qty: 1}}}},
	}
	p, _, err := Build(m)
	if err != nil {
		t.Fatalf("Build() error = %v, want nil", err)
	}
	if p.RHS[0].Num >= 0 {
		t.Fatalf("Build().RHS[0] = %v, want a negative right-hand side", p.RHS[0])
	}
	res, err := Solve(context.Background(), p, 1000)
	if err != nil {
		t.Fatalf("Solve() error = %v, want nil", err)
	}
	if err := certify(p, res); err != nil {
		t.Errorf("certify(Solve()) = %v, want nil", err)
	}
	res.Dual[0].Num = new(big.Int).Add(res.Dual[0].Num, res.Dual[0].Den)
	if err := certify(p, res); !errors.Is(err, ErrInconsistent) {
		t.Errorf("certify(tampered dual) = %v, want ErrInconsistent", err)
	}
}

// TestEndPhase1_RowWithoutAColumnIsInconsistent verifies that an artificial
// variable basic at 0 in a row with no non-zero structural or slack entry, which
// cannot occur because every row has its own slack, is reported as
// ErrInconsistent and not as a pivot on nothing.
//
// Guard: the drive-out step depends on that fact; without the check a corrupt
// tableau would loop or pivot on a zero.
func TestEndPhase1_RowWithoutAColumnIsInconsistent(t *testing.T) {
	t.Parallel()
	p := Programme{Rows: 1, RHS: []Ratio{{Num: -1, Den: 1}}, Cols: []Column{{Cost: Ratio{Num: 1, Den: 1}}}}
	tb := newTableau(p)
	obj := tb.startPhase1()
	for j := 0; j < tb.n+tb.m; j++ {
		tb.a[0][j].SetInt64(0)
	}
	tb.a[0][tb.rhs()].SetInt64(0)
	tb.a[tb.m][tb.rhs()].SetInt64(0)
	if _, err := tb.endPhase1(context.Background(), 1<<20, 0, obj); !errors.Is(err, ErrInconsistent) {
		t.Errorf("endPhase1(row of zeros) error = %v, want ErrInconsistent", err)
	}
}

// basicSolutions enumerates, exactly, every basic solution of p with its slacks
// (choose Rows columns of [A | I], solve, keep the non-negative ones) and
// returns whether any exists and the best objective among them. For a programme
// of "maximise Cost . x, A x <= RHS, x >= 0" a feasible region always has a
// vertex, so "no basic feasible solution" is exactly "infeasible", and a bounded
// optimum is the best vertex.
func basicSolutions(p Programme) (best *big.Rat, feasible bool) {
	n, m := len(p.Cols), p.Rows
	cols := make([][]*big.Rat, n+m)
	cost := make([]*big.Rat, n+m)
	for j := range cols {
		cols[j] = make([]*big.Rat, m)
		for i := range cols[j] {
			cols[j][i] = new(big.Rat)
		}
		cost[j] = new(big.Rat)
	}
	for j, c := range p.Cols {
		cost[j] = rat(c.Cost)
		for _, e := range c.Entries {
			cols[j][e.Row].Add(cols[j][e.Row], e.rat())
		}
	}
	for i := 0; i < m; i++ {
		cols[n+i][i].SetInt64(1)
	}
	var pick func(from int, chosen []int)
	pick = func(from int, chosen []int) {
		if len(chosen) == m {
			x, ok := solveBasis(cols, p, chosen)
			if !ok {
				return
			}
			obj := new(big.Rat)
			for k, j := range chosen {
				obj.Add(obj, new(big.Rat).Mul(cost[j], x[k]))
			}
			if best == nil || obj.Cmp(best) > 0 {
				best = obj
			}
			feasible = true
			return
		}
		for j := from; j < n+m; j++ {
			pick(j+1, append(chosen, j))
		}
	}
	pick(0, nil)
	return best, feasible
}

// solveBasis solves B x = RHS for the chosen columns by Gauss-Jordan over the
// rationals and reports whether B is regular and x is non-negative.
func solveBasis(cols [][]*big.Rat, p Programme, chosen []int) ([]*big.Rat, bool) {
	m := p.Rows
	a := make([][]*big.Rat, m)
	for i := range a {
		a[i] = make([]*big.Rat, m+1)
		for k, j := range chosen {
			a[i][k] = new(big.Rat).Set(cols[j][i])
		}
		a[i][m] = rat(p.RHS[i])
	}
	for c := 0; c < m; c++ {
		piv := -1
		for r := c; r < m; r++ {
			if a[r][c].Sign() != 0 {
				piv = r
				break
			}
		}
		if piv < 0 {
			return nil, false
		}
		a[c], a[piv] = a[piv], a[c]
		inv := new(big.Rat).Inv(a[c][c])
		for k := range a[c] {
			a[c][k].Mul(a[c][k], inv)
		}
		for r := 0; r < m; r++ {
			if r == c || a[r][c].Sign() == 0 {
				continue
			}
			f := new(big.Rat).Set(a[r][c])
			for k := range a[r] {
				a[r][k].Sub(a[r][k], new(big.Rat).Mul(f, a[c][k]))
			}
		}
	}
	x := make([]*big.Rat, m)
	for i := range x {
		x[i] = a[i][m]
		if x[i].Sign() < 0 {
			return nil, false
		}
	}
	return x, true
}

// TestSolve_TwoPhaseAgainstVertexEnumeration verifies the first phase against an
// exact enumeration of the basic solutions, over many small programmes with
// right-hand sides of either sign: the solver says ErrInfeasible exactly when no
// basic feasible solution exists, never says it for a feasible one, and when it
// returns an optimum that optimum is the best vertex.
//
// The programmes come from a fixed linear congruential sequence, so a failure is
// reproduced by its index.
//
// Requirement: FR-VIA-014, FR-VIA-055
func TestSolve_TwoPhaseAgainstVertexEnumeration(t *testing.T) {
	t.Parallel()
	state := uint64(0x9E3779B97F4A7C15)
	var infeasible, negative, optimal, unbounded int
	for i := 0; i < 4000; i++ {
		data := make([]byte, 48)
		for k := range data {
			state = state*6364136223846793005 + 1442695040888963407
			data[k] = byte(state >> 56)
		}
		p := signedProgramme(&byteReader{data: data})
		best, feasible := basicSolutions(p)
		res, err := Solve(context.Background(), p, 100_000)
		switch {
		case !feasible:
			infeasible++
			if !errors.Is(err, ErrInfeasible) {
				t.Fatalf("case %d: Solve(%v) error = %v, want ErrInfeasible (no basic feasible solution)", i, p, err)
			}
		case errors.Is(err, ErrUnbounded):
			unbounded++
		case err != nil:
			t.Fatalf("case %d: Solve(%v) error = %v, want nil or ErrUnbounded (a vertex exists)", i, p, err)
		default:
			optimal++
			if got := frat(res.Objective); got.Cmp(best) != 0 {
				t.Fatalf("case %d: Solve(%v).Objective = %v, want the best vertex %v", i, p, got, best)
			}
			for _, b := range res.Basis {
				if b >= len(p.Cols)+p.Rows {
					t.Fatalf("case %d: Solve(%v).Basis = %v, want no artificial variable", i, p, res.Basis)
				}
			}
			if negativeRHS(p) {
				negative++
			}
		}
	}
	// The cases must exercise the first phase, or the comparison proves nothing.
	if infeasible < 100 || negative < 500 || optimal < 1000 || unbounded < 10 {
		t.Errorf("cases: %d infeasible, %d optimal with a negative right-hand side of %d optimal, %d unbounded; want at least 100, 500 of 1000, 10", infeasible, negative, optimal, unbounded)
	}
}

// negativeRHS reports whether some right-hand side of p is negative.
func negativeRHS(p Programme) bool {
	for _, b := range p.RHS {
		if b.Num < 0 {
			return true
		}
	}
	return false
}

// TestEndPhase1_PivotsOnEitherSign verifies the drive-out step on a tableau built
// by hand: an artificial variable basic at 0 is pivoted out onto the first
// non-zero structural or slack entry of its row, a negative entry after the row
// is negated, so that the pivot is positive and the new basic column is d.
//
// One row, RHS -1, the column of x with entry v in it. The row is negated for
// the first phase, so its entry for x is -v and its slack entry is -1. The test
// sets the right-hand sides to 0, as the first phase leaves them for a
// feasible programme, and calls endPhase1 directly. With v = -1 the entry of x is
// +1 and is used as it stands; with v = 1 it is -1 and the row is negated first.
//
// Guard: a pivot on a negative entry would leave d negative and flip the sign of
// every value read off the tableau; the random programmes of the oracle test only
// ever meet the negative case, so this one pins the other.
func TestEndPhase1_PivotsOnEitherSign(t *testing.T) {
	t.Parallel()
	for _, v := range []int64{-1, 1} {
		p := Programme{Rows: 1, RHS: []Ratio{{Num: -1, Den: 1}}, Cols: []Column{{Cost: Ratio{Num: 1, Den: 1}, Entries: []Entry{{Row: 0, Val: Ratio{Num: v, Den: 1}}}}}}
		tb := newTableau(p)
		obj := tb.startPhase1()
		tb.a[0][tb.rhs()].SetInt64(0)
		tb.a[tb.m][tb.rhs()].SetInt64(0)
		pivots, err := tb.endPhase1(context.Background(), 1<<20, 0, obj)
		if err != nil || pivots != 1 {
			t.Fatalf("endPhase1(entry %d) = %d pivots, error %v, want 1, nil", v, pivots, err)
		}
		if tb.basis[0] != 0 || tb.d.Sign() <= 0 || tb.a[0][0].Cmp(tb.d) != 0 {
			t.Errorf("endPhase1(entry %d): basis = %v, d = %v, a[0][0] = %v, want x basic with d > 0 and a[0][0] = d", v, tb.basis, tb.d, tb.a[0][0])
		}
	}
}

// TestNewTableau_ArtificialForEachNegativeRow verifies that the starting tableau
// has one artificial variable for each negative right-hand side and no more, that
// each negative row starts from its own artificial and every other row from its
// slack, and that the negated rows carry a positive right-hand side.
//
// Programme: RHS 0, -1, 1, -1/2 over one column. Rows 1 and 3 are negative, so
// the tableau has two artificials, in columns n+m = 5 and 6.
//
// Guard: with a surplus artificial the first phase runs for a programme that
// needs none, and with a shared one two rows would start from one variable.
func TestNewTableau_ArtificialForEachNegativeRow(t *testing.T) {
	t.Parallel()
	p := Programme{
		Rows: 4,
		RHS:  []Ratio{{Num: 0, Den: 1}, {Num: -1, Den: 1}, {Num: 1, Den: 1}, {Num: -1, Den: 2}},
		Cols: []Column{{Cost: Ratio{Num: 1, Den: 1}, Entries: []Entry{{Row: 0, Val: Ratio{Num: 1, Den: 1}}}}},
	}
	tb := newTableau(p)
	if tb.na != 2 || len(tb.a[0]) != 1+4+2+1 {
		t.Fatalf("newTableau(RHS 0, -1, 1, -1/2): na = %d, columns = %d, want 2 and 8", tb.na, len(tb.a[0]))
	}
	if want := []int{1, 5, 3, 6}; !slices.Equal(tb.basis, want) {
		t.Errorf("newTableau(...).basis = %v, want %v", tb.basis, want)
	}
	for i := range p.RHS {
		if tb.a[i][tb.rhs()].Sign() < 0 {
			t.Errorf("newTableau(...) row %d has a negative right-hand side %v", i, tb.a[i][tb.rhs()])
		}
	}
	if q := (Programme{Rows: 1, RHS: []Ratio{{Num: 0, Den: 1}}}); newTableau(q).na != 0 {
		t.Errorf("newTableau(RHS 0).na = %d, want 0 (zero is not negative)", newTableau(q).na)
	}
}

// TestFirstNonzero_ScansStructuralAndSlackColumnsOnly verifies the column scan of
// the drive-out step on a tableau of one structural column and two rows (columns 0,
// 1 and 2 are x and the two slacks, column 3 the artificial of row 0, column 4 the
// right-hand side): it finds the lowest non-zero entry, the last slack included,
// and ignores the artificial column.
//
// Guard: a scan that stopped one column early would miss a row whose only entry is
// its slack, and one that read the artificial column would pivot a variable that
// must leave the basis back onto itself.
func TestFirstNonzero_ScansStructuralAndSlackColumnsOnly(t *testing.T) {
	t.Parallel()
	p := Programme{Rows: 2, RHS: []Ratio{{Num: -1, Den: 1}, {Num: 1, Den: 1}}, Cols: []Column{{Cost: Ratio{Num: 1, Den: 1}}}}
	for _, tc := range []struct {
		name string
		set  []int
		want int
	}{
		{"structural column", []int{0, 2}, 0},
		{"first slack", []int{1, 2}, 1},
		{"last slack only", []int{2}, 2},
		{"only the artificial column", nil, -1},
	} {
		tb := newTableau(p)
		for j := range tb.a[0] {
			tb.a[0][j].SetInt64(0)
		}
		tb.a[0][tb.n+tb.m].SetInt64(1) // the artificial column, always non-zero
		for _, j := range tc.set {
			tb.a[0][j].SetInt64(1)
		}
		if got := tb.firstNonzero(0); got != tc.want {
			t.Errorf("firstNonzero(%s) = %d, want %d", tc.name, got, tc.want)
		}
	}
}

// TestEndPhase1_HonoursCapAndContext verifies that the pivots that remove an
// artificial from the basis count against the cap and notice a cancelled
// context, and that the LimitError counts the artificial columns.
//
// Same programme as TestEndPhase1_PivotsOnEitherSign: one pivot is needed.
//
// Guard: without the checks a solve could exceed maxPivots by one pivot per
// negative row, each a big.Int pass over the whole tableau.
func TestEndPhase1_HonoursCapAndContext(t *testing.T) {
	t.Parallel()
	p := Programme{Rows: 1, RHS: []Ratio{{Num: -1, Den: 1}}, Cols: []Column{{Cost: Ratio{Num: 1, Den: 1}, Entries: []Entry{{Row: 0, Val: Ratio{Num: -1, Den: 1}}}}}}
	prepare := func() (*tableau, []*big.Int) {
		tb := newTableau(p)
		obj := tb.startPhase1()
		tb.a[0][tb.rhs()].SetInt64(0)
		tb.a[tb.m][tb.rhs()].SetInt64(0)
		return tb, obj
	}
	tb, obj := prepare()
	_, err := tb.endPhase1(context.Background(), 3, 3, obj)
	var le *LimitError
	if !errors.As(err, &le) || le.Pivots != 3 || le.Rows != 1 || le.Cols != 3 {
		t.Errorf("endPhase1(cap 3, done 3) error = %v, want LimitError{3 pivots, 1 row, 3 columns}", err)
	}
	tb, obj = prepare()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := tb.endPhase1(ctx, 1<<20, 0, obj); !errors.Is(err, context.Canceled) {
		t.Errorf("endPhase1(cancelled) error = %v, want context.Canceled", err)
	}
}
