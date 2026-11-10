package calibration

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const module = "github.com/Whealinghold/economy-sim"

// allowedImports are the module's packages that internal/calibration may import
// (plan 2-21 D620): the checked arithmetic and the design's overlay text. A
// path without a dot in its first element is the standard library.
func allowedImports() []string {
	return []string{module + "/internal/checked", module + "/internal/design"}
}

// bannedStd are the standard-library packages that break NFR-CLB-003: a clock,
// a random source, and math/bits (overflow goes through internal/checked, D621).
func bannedStd() []string {
	return []string{"time", "math/rand", "math/rand/v2", "crypto/rand", "math/bits", "sync", "sync/atomic", "os", "log", "log/slog"}
}

// sourceProblems parses one Go source and returns what breaks NFR-CLB-001 and
// NFR-CLB-003: an import outside the standard library and allowedImports, a
// banned package, a floating-point type, a goroutine, or a package-level var.
func sourceProblems(name, src string) []string {
	f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.SkipObjectResolution)
	if err != nil {
		return []string{name + ": " + err.Error()}
	}
	var out []string
	for _, im := range f.Imports {
		p, _ := strconv.Unquote(im.Path.Value)
		first, _, _ := strings.Cut(p, "/")
		switch {
		case slices.Contains(bannedStd(), p):
			out = append(out, name+": import of "+p)
		case strings.Contains(first, ".") && !slices.Contains(allowedImports(), p):
			out = append(out, name+": import of "+p+" outside the standard library, internal/checked and internal/design")
		}
	}
	for _, d := range f.Decls {
		if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.VAR {
			out = append(out, name+": package-level var")
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			if x.Name == "float32" || x.Name == "float64" {
				out = append(out, name+": "+x.Name)
			}
		case *ast.GoStmt:
			out = append(out, name+": go statement")
		}
		return true
	})
	return out
}

func packageSources(t *testing.T) map[string]string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		out[f] = string(b)
	}
	return out
}

// TestImports_CalibrationStdlibAndDesign verifies that the non-test files of
// internal/calibration import only the standard library, internal/checked and
// internal/design, so the package cannot reach the root, config/, strategy/,
// indicator/ or experiment/ (architecture.md 2.9).
//
// Requirement: NFR-CLB-001, NFR-CLB-003
func TestImports_CalibrationStdlibAndDesign(t *testing.T) {
	t.Parallel()
	for name, src := range packageSources(t) {
		for _, p := range sourceProblems(name, src) {
			if strings.Contains(p, "outside the standard library") {
				t.Errorf("%s", p)
			}
		}
	}
	for _, imp := range []string{module, module + "/config", module + "/strategy", module + "/indicator", module + "/experiment", module + "/internal/ordering"} {
		src := "package x\n\nimport _ \"" + imp + "\"\n"
		if got := sourceProblems("x.go", src); len(got) == 0 {
			t.Errorf("sourceProblems(import %s) found nothing, want a problem", imp)
		}
	}
	if got := sourceProblems("x.go", "package x\n\nimport _ \""+module+"/internal/checked\"\nimport _ \"sort\"\n"); len(got) > 0 {
		t.Errorf("sourceProblems(internal/checked and sort) = %v, want none", got)
	}
}

// TestNoFloatClockGoroutine verifies that the package has no floating point, no
// clock, no random source, no math/bits, no goroutine and no package-level
// variable (NFR-CLB-003, architecture.md 3.5, 7.1, 7.3, 12.2): the grid, the
// choice and the record are integer rules whose bytes are the same on every
// platform.
//
// Requirement: NFR-CLB-003
func TestNoFloatClockGoroutine(t *testing.T) {
	t.Parallel()
	for name, src := range packageSources(t) {
		if got := sourceProblems(name, src); len(got) > 0 {
			t.Errorf("sourceProblems(%s) = %v, want none", name, got)
		}
	}
	bad := map[string]string{
		"float64":      "package x\n\nfunc f() float64 { return 0 }\n",
		"float32":      "package x\n\nvar _ float32\n",
		"clock":        "package x\n\nimport \"time\"\n\nvar _ = time.Now\n",
		"math/rand":    "package x\n\nimport \"math/rand\"\n\nvar _ = rand.Int\n",
		"crypto/rand":  "package x\n\nimport _ \"crypto/rand\"\n",
		"math/bits":    "package x\n\nimport _ \"math/bits\"\n",
		"goroutine":    "package x\n\nfunc f() { go f() }\n",
		"package var":  "package x\n\nvar a = 1\n",
		"package vars": "package x\n\nvar (\n\ta = 1\n)\n",
	}
	for what, src := range bad {
		if got := sourceProblems("x.go", src); len(got) == 0 {
			t.Errorf("sourceProblems(%s) found nothing, want a problem", what)
		}
	}
}
