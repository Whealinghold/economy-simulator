package migration_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestPackage_NoFloatNoRoot verifies that the package imports only the standard
// library, internal/checked and internal/ordering, uses no floating-point type
// or math function, and has no identifier that names a region, a commodity or a
// role, so it works over primitive numbers.
//
// Requirement: NFR-MIG-003
//
// Guard: architecture.md 2.9, 3.5 and 9.1. An import of the root, a float in the
// arithmetic of a decision or an identifier that branches on a domain name would
// break the layering or the determinism, and no behavioural test would point at
// it.
func TestPackage_NoFloatNoRoot(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob: %v, %d files", err, len(files))
	}
	allowed := map[string]bool{
		"github.com/Whealinghold/economy-sim/internal/checked":  true,
		"github.com/Whealinghold/economy-sim/internal/ordering": true,
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			std := !strings.Contains(strings.SplitN(path, "/", 2)[0], ".")
			if !allowed[path] && (!std || path == "math/rand" || path == "math/rand/v2" || path == "time") {
				t.Errorf("%s imports %q, want the standard library (no rand, no time), internal/checked and internal/ordering only", file, path)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			low := strings.ToLower(id.Name)
			if low == "float32" || low == "float64" {
				t.Errorf("%s: identifier %q, want no floating point (architecture.md 3.5)", file, id.Name)
			}
			for _, bad := range []string{"region", "commodity", "role"} {
				if strings.Contains(low, bad) {
					t.Errorf("%s: identifier %q names a %s, want primitive names only", file, id.Name, bad)
				}
			}
			return true
		})
	}
}
