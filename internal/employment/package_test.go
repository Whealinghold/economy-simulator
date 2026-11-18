package employment_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestPackage_NoFloatNoRootNoDraw verifies that the sources of the package
// import only the standard library without a random or time package and
// internal/checked, and use no float type: the hire arithmetic is integer, it
// does not read the root and it takes no draw.
//
// Requirement: FR-EMP-018, FR-EMP-061, NFR-EMP-003
//
// Guard: architecture.md 2.9, 3.5 and 7.3. An import of the root would be a
// cycle, a float would differ by platform, and a draw would move every later
// word of a run.
func TestPackage_NoFloatNoRootNoDraw(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the package directory: %v", err)
	}
	forbidden := []string{"math/rand", "math/rand/v2", "crypto/rand", "time", "math"}
	sources := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		sources++
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			switch {
			case strings.Contains(path, "."):
				if path != "github.com/Whealinghold/economy-sim/internal/checked" {
					t.Errorf("%s imports %s, want the standard library and internal/checked only", name, path)
				}
			default:
				for _, bad := range forbidden {
					if path == bad {
						t.Errorf("%s imports %s", name, path)
					}
				}
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && (id.Name == "float32" || id.Name == "float64") {
				t.Errorf("%s uses %s", name, id.Name)
			}
			return true
		})
	}
	if sources < 5 {
		t.Errorf("parsed %d source files, want at least 5: the guard is looking at the wrong place", sources)
	}
}
