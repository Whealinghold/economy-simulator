package tenure_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sources returns the non-test Go files of the package, parsed with imports.
func sources(t *testing.T) map[string][]string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	imports := map[string][]string{}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", name, err)
		}
		for _, imp := range f.Imports {
			imports[name] = append(imports[name], strings.Trim(imp.Path.Value, `"`))
		}
	}
	return imports
}

// TestPackage_NoFloatNoRoot verifies the package imports the standard library,
// internal/checked and internal/endowment only, never the root, config, strategy,
// indicator or experiment, and uses no floating point and no math package: the
// rules are integer rules (architecture.md 2.9, 3.5, 5.1).
//
// Requirement: FR-LND-036, NFR-LND-003
func TestPackage_NoFloatNoRoot(t *testing.T) {
	t.Parallel()
	allowed := map[string]bool{
		"github.com/Whealinghold/economy-sim/internal/checked":   true,
		"github.com/Whealinghold/economy-sim/internal/endowment": true,
	}
	for name, imps := range sources(t) {
		for _, imp := range imps {
			if strings.Contains(imp, ".") && !allowed[imp] {
				t.Errorf("%s imports %s, want the standard library, internal/checked or internal/endowment only", name, imp)
			}
			if imp == "math" || imp == "math/big" {
				t.Errorf("%s imports %s, want integer rules without the math package", name, imp)
			}
		}
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", name, err)
		}
		for _, word := range []string{"float32", "float64"} {
			if strings.Contains(string(body), word) {
				t.Errorf("%s mentions %s, want integer arithmetic", name, word)
			}
		}
	}
}

// TestPackage_TakesNoDraw verifies no file imports a random or clock package: the
// tenure rules take no draw and every tie goes by the key the caller passes in
// (spec-land.md 8.2, FR-LND-036).
//
// Requirement: FR-LND-036
func TestPackage_TakesNoDraw(t *testing.T) {
	t.Parallel()
	for name, imps := range sources(t) {
		for _, imp := range imps {
			switch imp {
			case "math/rand", "math/rand/v2", "crypto/rand", "time":
				t.Errorf("%s imports %s, want no draw and no clock", name, imp)
			}
		}
	}
}
