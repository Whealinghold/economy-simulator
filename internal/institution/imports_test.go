package institution_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestPackage_NoRootNoName verifies that the package imports only the standard
// library (no rand, no time), internal/checked and internal/ordering, and has no
// identifier that names a body, right, role, region or commodity, so it works over
// primitive indexes and the root converts at the call.
//
// Requirement: NFR-ORG-004
//
// Guard: architecture.md 2.9 and 9.1. An import of the root or a name of a domain
// entity would break the layering, and no behavioural test would point at it.
func TestPackage_NoRootNoName(t *testing.T) {
	t.Parallel()
	for _, f := range packageFiles(t) {
		for _, imp := range f.ast.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			std := !strings.Contains(strings.SplitN(path, "/", 2)[0], ".")
			ok := path == "github.com/Whealinghold/economy-sim/internal/checked" || path == "github.com/Whealinghold/economy-sim/internal/ordering"
			if !ok && (!std || path == "math/rand" || path == "math/rand/v2" || path == "time") {
				t.Errorf("%s imports %q, want the standard library (no rand, no time), internal/checked and internal/ordering only", f.name, path)
			}
		}
		ast.Inspect(f.ast, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			low := strings.ToLower(id.Name)
			for _, bad := range []string{"body", "right", "role", "region", "commodity"} {
				if strings.Contains(low, bad) {
					t.Errorf("%s: identifier %q names a %s, want primitive names only", f.name, id.Name, bad)
				}
			}
			return true
		})
	}
}

// TestPackage_NoFloat verifies that the package declares no floating-point type and
// calls no function of the math package other than a constant.
//
// Requirement: NFR-ORG-004
//
// Guard: architecture.md 3.5. A float in the arithmetic of a range or a place would
// differ between platforms and break the golden output.
func TestPackage_NoFloat(t *testing.T) {
	t.Parallel()
	for _, f := range packageFiles(t) {
		ast.Inspect(f.ast, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.Ident:
				if low := strings.ToLower(x.Name); low == "float32" || low == "float64" {
					t.Errorf("%s: identifier %q, want no floating point (architecture.md 3.5)", f.name, x.Name)
				}
			case *ast.CallExpr:
				if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
					if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "math" {
						t.Errorf("%s: call of math.%s, want integer arithmetic only", f.name, sel.Sel.Name)
					}
				}
			}
			return true
		})
	}
}

type parsedFile struct {
	name string
	ast  *ast.File
}

// packageFiles parses the non-test files of the package.
func packageFiles(t *testing.T) []parsedFile {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob: %v, %d files", err, len(files))
	}
	var out []parsedFile
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		out = append(out, parsedFile{file, f})
	}
	return out
}
