package employment_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
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
		for _, problem := range sourceProblems(name, f, forbidden) {
			t.Error(problem)
		}
	}
	if sources < 5 {
		t.Errorf("parsed %d source files, want at least 5: the guard is looking at the wrong place", sources)
	}
}

// sourceProblems lists what a parsed source file breaks: an import outside the
// standard library and internal/checked, a forbidden standard package, or a
// float type.
func sourceProblems(name string, f *ast.File, forbidden []string) []string {
	var problems []string
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if problem := importProblem(name, path, forbidden); problem != "" {
			problems = append(problems, problem)
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && (id.Name == "float32" || id.Name == "float64") {
			problems = append(problems, name+" uses "+id.Name)
		}
		return true
	})
	return problems
}

// importProblem returns the complaint about one import path, or "" when the
// path is allowed.
func importProblem(name, path string, forbidden []string) string {
	if strings.Contains(path, ".") {
		if path != "github.com/Whealinghold/economy-sim/internal/checked" {
			return name + " imports " + path + ", want the standard library and internal/checked only"
		}
		return ""
	}
	if slices.Contains(forbidden, path) {
		return name + " imports " + path
	}
	return ""
}
