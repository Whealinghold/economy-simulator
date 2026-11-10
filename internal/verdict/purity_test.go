package verdict_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// packageFiles returns the non-test Go files of the package.
func packageFiles(t *testing.T) []string {
	t.Helper()
	all, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	var out []string
	for _, f := range all {
		if !strings.HasSuffix(f, "_test.go") {
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		t.Fatalf("no non-test files found")
	}
	return out
}

// TestVerdicts_IntegerOnly verifies by a source text check that the package
// has no float type and no float function of math, and no math import at all
// (D587: a range check replaces it); math/bits is allowed.
//
// Requirement: FR-STY-006
func TestVerdicts_IntegerOnly(t *testing.T) {
	t.Parallel()
	floatRe := regexp.MustCompile(`\bfloat(32|64)\b|\bmath\.[A-Za-z]`)
	for _, f := range packageFiles(t) {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			code, _, _ := strings.Cut(line, "//")
			if floatRe.MatchString(code) {
				t.Errorf("%s:%d uses a float type or a function of math: %q", f, i+1, strings.TrimSpace(line))
			}
		}
		fs, err := parser.ParseFile(token.NewFileSet(), f, src, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		for _, imp := range fs.Imports {
			if imp.Path.Value == `"math"` {
				t.Errorf("%s imports math; the range check of D587 replaces it", f)
			}
		}
	}
}

// TestPurity_NoWorld verifies that the package imports the standard library
// only (no path with a dot in its first element, so nothing of the module) and
// that no exported function takes or returns a type of another package apart
// from the standard library.
//
// Requirement: FR-STY-080, NFR-STY-004
func TestPurity_NoWorld(t *testing.T) {
	t.Parallel()
	for _, f := range packageFiles(t) {
		fs, err := parser.ParseFile(token.NewFileSet(), f, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		for _, imp := range fs.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			first, _, _ := strings.Cut(path, "/")
			if strings.Contains(first, ".") || path == "log/slog" || path == "time" {
				t.Errorf("%s imports %s; the package is the standard library without logging or the clock", f, path)
			}
		}
		for _, d := range fs.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || !fd.Name.IsExported() {
				continue
			}
			ast.Inspect(fd.Type, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok {
					t.Errorf("%s: exported %s mentions a type of another package: %v", f, fd.Name.Name, sel.Sel.Name)
				}
				return true
			})
		}
	}
}
