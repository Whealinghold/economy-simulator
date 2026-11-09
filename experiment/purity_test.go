package experiment_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestPackage_PurityAST verifies, over the non-test files of experiment/,
// internal/design and internal/reduce, that none starts a goroutine, uses a
// float type or literal, imports a source of time, randomness or the operating
// system, or declares a package variable other than a sentinel error, a
// compile-time assertion or the quantile table.
//
// Requirement: NFR-EXP-002
//
// Guard: the existing guards cover imports and time.Now but not goroutines or
// floats, and a float in a reduction would make a result depend on the platform
// (architecture.md 3.5, 12.2).
func TestPackage_PurityAST(t *testing.T) {
	t.Parallel()
	banned := map[string]bool{"math/rand": true, "math/rand/v2": true, "crypto/rand": true, "os": true, "time": true}
	scanned := 0
	for _, dir := range []string{".", "../internal/design", "../internal/reduce"} {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range files {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			scanned++
			f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.SkipObjectResolution)
			if err != nil {
				t.Fatalf("parse %s: %v", name, err)
			}
			for _, p := range impurities(f, banned) {
				t.Errorf("%s: %s", name, p)
			}
		}
	}
	if scanned < 10 {
		t.Errorf("scanned %d files, want at least 10: the walk looks at the wrong place", scanned)
	}
}

// impurities returns the reasons a parsed file breaks the rules of
// TestPackage_PurityAST.
func impurities(f *ast.File, banned map[string]bool) []string {
	var out []string
	for _, imp := range f.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); banned[p] {
			out = append(out, "imports "+p)
		}
	}
	for _, d := range f.Decls {
		if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.VAR {
			for _, s := range g.Specs {
				for _, n := range s.(*ast.ValueSpec).Names {
					if n.Name != "_" && !strings.HasPrefix(n.Name, "Err") && n.Name != "quantiles" {
						out = append(out, "package variable "+n.Name)
					}
				}
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GoStmt:
			out = append(out, "go statement")
		case *ast.SelectorExpr:
			// math constants (MaxInt64) are exact; its functions are float.
			if id, ok := x.X.(*ast.Ident); ok && id.Name == "math" && !strings.HasPrefix(x.Sel.Name, "Max") && !strings.HasPrefix(x.Sel.Name, "Min") {
				out = append(out, "math."+x.Sel.Name)
			}
		case *ast.Ident:
			if x.Name == "float32" || x.Name == "float64" {
				out = append(out, "float type")
			}
		case *ast.BasicLit:
			if x.Kind == token.FLOAT || x.Kind == token.IMAG {
				out = append(out, "float literal "+x.Value)
			}
		}
		return true
	})
	return out
}

// TestPackage_PurityDetects verifies the checker on in-memory sources, so the
// guard above cannot pass because it looks at nothing.
//
// Requirement: NFR-EXP-002
//
// Guard: a checker that missed a goroutine, a float or a variable would let the
// rules lapse with every test green.
func TestPackage_PurityDetects(t *testing.T) {
	t.Parallel()
	banned := map[string]bool{"time": true}
	tests := []struct {
		name, src string
		want      int
	}{
		{"clean", `package p; var ErrX = 1; var _ = 2; var quantiles = 3; func f() { x := 1; _ = x }`, 0},
		{"goroutine", `package p; func f() { go f() }`, 1},
		{"float type", `package p; func f(x float64) {}`, 1},
		{"math function", `package p; import "math"; var _ = math.Sqrt`, 1},
		{"math constant", `package p; import "math"; var _ = math.MaxInt64`, 0},
		{"float literal", `package p; var _ = 1.5`, 1},
		{"time import", `package p; import "time"`, 1},
		{"package variable", `package p; var cache = map[int]int{}`, 1},
	}
	for _, tt := range tests {
		f, err := parser.ParseFile(token.NewFileSet(), "x.go", tt.src, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if got := impurities(f, banned); len(got) != tt.want {
			t.Errorf("%s: impurities = %v, want %d", tt.name, got, tt.want)
		}
	}
}
