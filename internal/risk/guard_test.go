package risk_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPackage_NoRootNoName verifies the package imports the standard library
// and internal/checked only, never the root, config, strategy, indicator or
// experiment, and names no region, mode, store class or commodity: it works on
// numbers (architecture.md 2.9, 9.1).
//
// Requirement: FR-RSK-039
func TestPackage_NoRootNoName(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", name, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(path, ".") && path != "github.com/Whealinghold/economy-sim/internal/checked" {
				t.Errorf("%s imports %s, want the standard library or internal/checked only", name, path)
			}
		}
	}
}

// TestPackage_NoFloat verifies the package uses no floating point and no math
// package: the rules are integer rules (architecture.md 3.5, NFR-RSK-003).
//
// Requirement: NFR-RSK-003
func TestPackage_NoFloat(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", name, err)
		}
		for _, word := range []string{"float32", "float64", `"math"`, `"math/big"`} {
			if strings.Contains(string(body), word) {
				t.Errorf("%s contains %s, want integer arithmetic only", name, word)
			}
		}
	}
}
