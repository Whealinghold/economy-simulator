package steady

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// twoVar is "max 3x+2y; x+y<=4, x+3y<=9, x<=3": optimum (3, 1), value 11,
// duals (2, 0, 1) (worked in solve_test.go).
func twoVar() Programme {
	one := func(n int64) Ratio { return Ratio{Num: n, Den: 1} }
	return Programme{Rows: 3, RHS: []Ratio{one(4), one(9), one(3)}, Cols: []Column{
		{Cost: one(3), Entries: []Entry{{Row: 0, Val: one(1)}, {Row: 1, Val: one(1)}, {Row: 2, Val: one(1)}}},
		{Cost: one(2), Entries: []Entry{{Row: 0, Val: one(1)}, {Row: 1, Val: one(3)}}},
	}}
}

// fracOf is the integer n as a Frac.
func fracOf(n int64) Frac { return Frac{Num: big.NewInt(n), Den: big.NewInt(1)} }

// solved returns twoVar and its certified result.
func solved(t *testing.T) (Programme, Result) {
	t.Helper()
	p := twoVar()
	res, err := Solve(context.Background(), p, 100)
	if err != nil {
		t.Fatalf("Solve(twoVar) error = %v", err)
	}
	return p, res
}

// TestCertify_RejectsEachFailure verifies that every one of the checks of the
// certificate fails on its own corruption of a correct result, and that the
// correct result passes.
//
// Guard: the certificate is what stops a wrong pivot from returning a plausible
// table; a check that never fires would let a defect of the pivot through
// unseen (FR-VIA-014).
//
// Requirement: FR-VIA-013, FR-VIA-014
func TestCertify_RejectsEachFailure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(res *Result)
	}{
		{"primal value negative", func(r *Result) { r.Primal[1] = fracOf(-1) }},
		// x = 4 puts row 0 at 4 + 1 = 5 > 4.
		{"row violated", func(r *Result) { r.Primal[0] = fracOf(4) }},
		{"dual value negative", func(r *Result) { r.Dual[1] = fracOf(-1) }},
		// y1 = 1 gives column y a reduced cost 1*1 + 0*3 = 1 < 2.
		{"column dual infeasible", func(r *Result) { r.Dual[0] = fracOf(1) }},
		// The dual objective stays 4*2+3*1 = 11 but the claimed objective moves.
		{"claimed objective differs", func(r *Result) { r.Objective = fracOf(12) }},
		// x = 2 is feasible and the duals are feasible, but 3*2+2*1 = 8 != 11.
		{"primal objective differs from dual", func(r *Result) { r.Primal[0] = fracOf(2) }},
		{"primal of the wrong length", func(r *Result) { r.Primal = r.Primal[:1] }},
		{"dual of the wrong length", func(r *Result) { r.Dual = nil }},
		{"zero denominator", func(r *Result) { r.Primal[0] = Frac{Num: big.NewInt(1), Den: new(big.Int)} }},
		{"nil numerator", func(r *Result) { r.Dual[0] = Frac{Den: big.NewInt(1)} }},
		{"nil denominator in the dual", func(r *Result) { r.Dual[2] = Frac{Num: big.NewInt(1)} }},
	}
	p, ok := solved(t)
	if err := certify(p, ok); err != nil {
		t.Fatalf("certify(correct result) error = %v, want nil", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, res := solved(t)
			tt.mutate(&res)
			if err := certify(p, res); !errors.Is(err, ErrInconsistent) {
				t.Errorf("certify(%s) error = %v, want %v", tt.name, err, ErrInconsistent)
			}
		})
	}
}

// TestTableau_WrongPivotIsCaught verifies that a tableau corrupted before or
// after the pivots no longer gives a result the certificate accepts.
//
// Guard: without the certificate a bug in the pivot would return a plausible
// optimum of the wrong programme (FR-VIA-014).
//
// Requirement: FR-VIA-014
func TestTableau_WrongPivotIsCaught(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		corrupt func(tb *tableau, before bool)
		before  bool
	}{
		// Row 0 becomes 2x + y <= 4: its optimum is (3/5, 14/5), which meets the
		// original rows but has duals (7/5, 1/5, 0) with 7/5 + 1/5 = 8/5 < 3 on
		// column x, so it is not dual feasible for the original data.
		{"entry changed before the solve", func(tb *tableau, _ bool) { tb.a[0][0].SetInt64(2) }, true},
		{"objective value changed after", func(tb *tableau, _ bool) { tb.a[tb.m][tb.rhs()].Add(tb.a[tb.m][tb.rhs()], big.NewInt(1)) }, false},
		{"basic value changed after", func(tb *tableau, _ bool) { tb.a[0][tb.rhs()].Add(tb.a[0][tb.rhs()], big.NewInt(1)) }, false},
		{"dual entry changed after", func(tb *tableau, _ bool) { tb.a[tb.m][tb.n].Add(tb.a[tb.m][tb.n], big.NewInt(1)) }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := twoVar()
			tb := newTableau(p)
			if tt.before {
				tt.corrupt(tb, true)
			}
			pivots, err := tb.run(context.Background(), 100, 0)
			if err != nil {
				t.Fatalf("run(%s) error = %v", tt.name, err)
			}
			if !tt.before {
				tt.corrupt(tb, false)
			}
			if err := certify(p, tb.result(pivots)); !errors.Is(err, ErrInconsistent) {
				t.Errorf("certify(%s) error = %v, want %v", tt.name, err, ErrInconsistent)
			}
		})
	}
}

// TestTableau_InexactDivisionIsAnError verifies that a pivot whose division is
// not exact reports ErrInconsistent instead of rounding.
//
// Guard: the integer-preserving step is exact only on a correct tableau; a
// silent truncation would corrupt every later entry.
//
// Requirement: FR-VIA-014
func TestTableau_InexactDivisionIsAnError(t *testing.T) {
	t.Parallel()
	tb := newTableau(twoVar())
	tb.d = big.NewInt(7) // the true denominator is 1, so 7 does not divide
	if _, err := tb.run(context.Background(), 100, 0); !errors.Is(err, ErrInconsistent) {
		t.Errorf("run with a wrong common denominator error = %v, want %v", err, ErrInconsistent)
	}
}

// TestPackage_NoFloatingPoint verifies that no non-test file of the package
// names a floating-point type, big.Float or the math package.
//
// Guard: architecture.md 3.5 bans floating point in anything that affects
// results; math/big is the one exception and the rest of math is not needed.
//
// Requirement: FR-VIA-015
func TestPackage_NoFloatingPoint(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) < 4 {
		t.Fatalf("Glob = %v, %v, want the files of the package", files, err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(token.NewFileSet(), file, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == "math" {
				t.Errorf("%s imports math", file)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Ident:
				if n.Name == "float32" || n.Name == "float64" || n.Name == "Float" {
					t.Errorf("%s names %s", file, n.Name)
				}
			case *ast.SelectorExpr:
				if x, ok := n.X.(*ast.Ident); ok && x.Name == "big" && strings.HasPrefix(n.Sel.Name, "Float") {
					t.Errorf("%s uses big.%s", file, n.Sel.Name)
				}
			}
			return true
		})
	}
}
