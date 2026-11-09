package rumour_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestPackage_StdlibOnlyAndKnowsNoDomainNames verifies that the package imports
// only the standard library and internal/checked, and that no identifier in it
// names a region, a commodity or a role, so it works over primitive slices.
//
// Requirement: NFR-INF-003
//
// Guard: architecture.md 2.9 and 9.1. An import of the root or an identifier
// that branches on a domain name would break the layering, and no behavioural
// test would point at it.
func TestPackage_StdlibOnlyAndKnowsNoDomainNames(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob: %v, %d files", err, len(files))
	}
	const checked = "github.com/Whealinghold/economy-sim/internal/checked"
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
			if path != checked && !std {
				t.Errorf("%s imports %q, want the standard library and internal/checked only", file, path)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			low := strings.ToLower(id.Name)
			for _, bad := range []string{"region", "commodity", "role"} {
				if strings.Contains(low, bad) {
					t.Errorf("%s: identifier %q names a %s, want primitive names only", file, id.Name, bad)
				}
			}
			return true
		})
	}
}
