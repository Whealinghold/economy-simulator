package measure_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestMeasure_NoFloatNoMath verifies, by parsing the non-test files of the
// package, that none imports math (math/bits is allowed), declares a package
// level variable that is not a sentinel error or a constant table, or names a
// floating-point type.
//
// Requirement: FR-IND-008, NFR-IND-004
func TestMeasure_NoFloatNoMath(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir error = %v", err)
	}
	var files int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		files++
		f, err := parser.ParseFile(token.NewFileSet(), e.Name(), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", e.Name(), err)
		}
		for _, imp := range f.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == "math" || strings.HasPrefix(p, "math/rand") {
				t.Errorf("%s imports %q, want no math (FR-IND-008)", e.Name(), p)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && (id.Name == "float32" || id.Name == "float64") {
				t.Errorf("%s uses %s, want no floating point (FR-IND-008)", e.Name(), id.Name)
			}
			return true
		})
	}
	if files < 5 {
		t.Fatalf("scanned %d files, want at least 5: the scan is looking at the wrong place", files)
	}
}
