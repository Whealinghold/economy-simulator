package insure_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// sourceFiles parses the non-test files of the package.
func sourceFiles(t *testing.T) (files []*ast.File, names []string) {
	t.Helper()
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	for _, p := range paths {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), p, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", p, err)
		}
		files, names = append(files, f), append(names, p)
	}
	return files, names
}

// TestPackage_NoRootNoName verifies that the package imports only the standard
// library (no rand, no time) and internal/checked, and has no identifier that names a
// place, a transport kind, a good, an agent or a role, so it works over primitive
// values and the root converts at the call.
//
// Requirement: NFR-INS-005
//
// Guard: architecture.md 2.9 and 9.1. An import of the root or a name of a domain
// entity would break the layering, and no behavioural test would point at it.
func TestPackage_NoRootNoName(t *testing.T) {
	t.Parallel()
	files, names := sourceFiles(t)
	for i, f := range files {
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			std := !strings.Contains(strings.SplitN(path, "/", 2)[0], ".")
			ok := path == "github.com/Whealinghold/economy-sim/internal/checked"
			if !ok && (!std || path == "math/rand" || path == "math/rand/v2" || path == "time") {
				t.Errorf("%s imports %q, want the standard library (no rand, no time) and internal/checked only", names[i], path)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			low := strings.ToLower(id.Name)
			for _, bad := range []string{"region", "commodity", "agent", "role", "shipment", "mode"} {
				if strings.Contains(low, bad) {
					t.Errorf("%s: identifier %q names a %s, want primitive names only", names[i], id.Name, bad)
				}
			}
			return true
		})
	}
}

// TestPackage_NoFloat verifies that the package declares no floating-point type and
// imports no math package.
//
// Requirement: NFR-INS-003, NFR-INS-005
//
// Guard: architecture.md 3.5. A float in the premium or the due would differ
// between platforms and break the golden output.
func TestPackage_NoFloat(t *testing.T) {
	t.Parallel()
	files, names := sourceFiles(t)
	for i, f := range files {
		for _, imp := range f.Imports {
			if strings.Trim(imp.Path.Value, `"`) == "math" {
				t.Errorf("%s imports math, want integer arithmetic through internal/checked", names[i])
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && (id.Name == "float32" || id.Name == "float64") {
				t.Errorf("%s uses %s, want fixed-point integers", names[i], id.Name)
			}
			return true
		})
	}
}
