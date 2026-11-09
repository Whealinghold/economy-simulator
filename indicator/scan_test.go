package indicator_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// parseNonTest parses the non-test Go files of the package directory.
func parseNonTest(t *testing.T) map[string]*ast.File {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir error = %v", err)
	}
	files := map[string]*ast.File{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), e.Name(), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", e.Name(), err)
		}
		files[e.Name()] = f
	}
	if len(files) < 5 {
		t.Fatalf("parsed %d files, want at least 5: the scan is looking at the wrong place", len(files))
	}
	return files
}

// mentionsWorld reports whether a type expression names World, as an identifier
// or as a selector of the root package, anywhere inside it, so *World, a slice
// of Worlds and a function type that takes or returns one all count.
func mentionsWorld(e ast.Expr) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			found = found || x.Name == "World"
		case *ast.SelectorExpr:
			found = found || x.Sel.Name == "World"
		}
		return !found
	})
	return found
}

// TestSignature_NoWorld verifies, by parsing the package, that no exported
// function, method, struct field or type of indicator has a parameter, a result
// or a field whose type is World or *World or contains one, directly or in a
// function type (spec-indicators.md 3.1.2).
//
// shortcut: a type alias that hides World is not followed; add go/types if one
// appears.
//
// Requirement: FR-IND-002
func TestSignature_NoWorld(t *testing.T) {
	t.Parallel()
	for name, f := range parseNonTest(t) {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if !d.Name.IsExported() {
					continue
				}
				for _, list := range []*ast.FieldList{d.Recv, d.Type.Params, d.Type.Results} {
					for _, fld := range fieldsOf(list) {
						if mentionsWorld(fld.Type) {
							t.Errorf("%s: exported %s has a World in its signature (FR-IND-002)", name, d.Name.Name)
						}
					}
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					if ts, ok := s.(*ast.TypeSpec); ok && ts.Name.IsExported() && mentionsWorld(ts.Type) {
						t.Errorf("%s: exported type %s mentions World (FR-IND-002)", name, ts.Name.Name)
					}
				}
			}
		}
	}
}

// fieldsOf returns the fields of a list, nil for a nil list.
func fieldsOf(l *ast.FieldList) []*ast.Field {
	if l == nil {
		return nil
	}
	return l.List
}

// TestMentionsWorld_Detects verifies the detector on type expressions, so the
// scan above cannot pass because it recognises nothing.
//
// Guard: a detector that missed *World, a World in a function type or the
// qualified economysim.World would let an indicator take a world with every test
// green.
func TestMentionsWorld_Detects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expr string
		want bool
	}{
		{"int64", false},
		{"*economysim.World", true},
		{"economysim.World", true},
		{"World", true},
		{"[]*economysim.World", true},
		{"func(*economysim.World) int", true},
		{"func(int) economysim.World", true},
		{"map[string]economysim.Snapshot", false},
		{"economysim.Worldly", false},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			t.Parallel()
			e, err := parser.ParseExpr(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpr(%q) error = %v", tt.expr, err)
			}
			if got := mentionsWorld(e); got != tt.want {
				t.Errorf("mentionsWorld(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

// TestIndicator_NoGlobalStateNoGoroutineNoFloat verifies, by parsing the
// package, that it declares no package-level variable except sentinel errors
// (named Err...) and blank compile-time assertions, starts no goroutine, names
// no floating-point type, and imports neither math, the random packages, time,
// sync nor os (spec-indicators.md 3.1.4, 5, 10.3).
//
// Requirement: FR-IND-008, FR-IND-054
func TestIndicator_NoGlobalStateNoGoroutineNoFloat(t *testing.T) {
	t.Parallel()
	banned := []string{"math", "math/rand", "math/rand/v2", "crypto/rand", "time", "sync", "sync/atomic", "os"}
	for name, f := range parseNonTest(t) {
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			for _, b := range banned {
				if p == b {
					t.Errorf("%s imports %q, want none of %v", name, p, banned)
				}
			}
		}
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.VAR {
				continue
			}
			for _, s := range g.Specs {
				for _, id := range s.(*ast.ValueSpec).Names {
					if id.Name != "_" && !strings.HasPrefix(id.Name, "Err") {
						t.Errorf("%s declares package-level variable %s, want only sentinel errors (FR-IND-054)", name, id.Name)
					}
				}
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.GoStmt:
				t.Errorf("%s starts a goroutine (FR-IND-054)", name)
			case *ast.Ident:
				if x.Name == "float32" || x.Name == "float64" {
					t.Errorf("%s names %s, want no floating point (FR-IND-008)", name, x.Name)
				}
			}
			return true
		})
	}
}
