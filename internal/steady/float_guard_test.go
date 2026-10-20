package steady

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// floatProblems parses one non-test source file and returns a description of
// each use of floating point: the types float32 and float64 (and complex ones), a
// floating literal, any identifier or selector with Float in its name (big.Float,
// Rat.Float64, SetFloat64, NewFloat) and an import of math (FR-VIA-015).
func floatProblems(name string, src any) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), name, src, 0)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, imp := range f.Imports {
		if imp.Path.Value == `"math"` {
			out = append(out, name+": imports math")
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			switch n.Name {
			case "float32", "float64", "complex64", "complex128":
				out = append(out, name+": uses "+n.Name)
			default:
				if strings.Contains(n.Name, "Float") {
					out = append(out, name+": uses "+n.Name)
				}
			}
		case *ast.BasicLit:
			if n.Kind == token.FLOAT || n.Kind == token.IMAG {
				out = append(out, name+": floating literal "+n.Value)
			}
		}
		return true
	})
	return out, nil
}

// TestNoFloatingPoint verifies that no non-test file of the package uses a
// floating-point type, literal or function, nor imports math.
//
// Requirement: FR-VIA-015
// Guard: the determinism guard of the root does not detect float types; the exact
// solver's results are fractions, and one float64 conversion would make the table
// depend on the platform.
func TestNoFloatingPoint(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir(.) error = %v", err)
	}
	checked := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		checked++
		got, err := floatProblems(name, nil)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if len(got) > 0 {
			t.Errorf("floatProblems(%s) = %v, want none", name, got)
		}
	}
	if checked < 8 {
		t.Errorf("checked %d files, want at least 8 (the guard must read the package)", checked)
	}
}

// TestFloatProblems_Detects verifies that the guard reports each kind of
// floating-point use and accepts exact arithmetic.
//
// Guard: a guard that never fires proves nothing; each case is a way the
// package could drift.
func TestFloatProblems_Detects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		src  string
		want int
	}{
		{"float64 type", `package p; var x float64`, 1},
		{"float32 conversion", `package p; var x = float32(1)`, 1},
		{"float literal", `package p; var x = 0.5`, 1},
		{"big.Float", `package p; import "math/big"; var x *big.Float`, 1},
		{"Rat.Float64", `package p; import "math/big"; func f(r *big.Rat) { r.Float64() }`, 1},
		{"math import", `package p; import "math"; var x = math.MaxInt64`, 1},
		{"big.Int and Rat", `package p; import "math/big"; var x = big.NewRat(1, 2); var y = new(big.Int)`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := floatProblems("x.go", tt.src)
			if err != nil {
				t.Fatalf("floatProblems(%s) error = %v", tt.name, err)
			}
			if len(got) != tt.want {
				t.Errorf("floatProblems(%s) = %v, want %d problems", tt.name, got, tt.want)
			}
		})
	}
}
