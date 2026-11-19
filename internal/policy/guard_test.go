package policy_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// allowedImports are the only imports of the package outside the standard
// library's non-banned packages (architecture.md 2.9, 5.1).
var allowedImports = map[string]bool{
	"github.com/Whealinghold/economy-sim/internal/checked":  true,
	"github.com/Whealinghold/economy-sim/internal/ordering": true,
}

// TestPackage_NoFloatNoGlobalNoGo verifies, over the syntax of every non-test
// file, that the package imports internal/checked, internal/ordering and the
// standard library only (and never time, math, reflect, crypto or unsafe),
// declares no float type, starts no goroutine, has no init and no package-level
// variable, so it keeps no state and takes no random source (architecture.md
// 3.5, 7.1, 7.2, 12.2).
//
// Requirement: FR-PLC-091
func TestPackage_NoFloatNoGlobalNoGo(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	checked := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		checked++
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", name, err)
		}
		checkImports(t, name, f)
		checkSyntax(t, name, f)
		checkNoGlobals(t, name, f)
	}
	if checked == 0 {
		t.Fatal("no source file parsed: the glob is looking at the wrong place")
	}
}

// checkImports reports an import outside the allowed set or in the banned list.
func checkImports(t *testing.T, name string, f *ast.File) {
	t.Helper()
	banned := []string{"time", "math", "reflect", "unsafe", "os", "log", "sync", "crypto"}
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if strings.Contains(path, ".") && !allowedImports[path] {
			t.Errorf("%s imports %s, want the standard library, internal/checked or internal/ordering", name, path)
		}
		for _, b := range banned {
			if path == b || strings.HasPrefix(path, b+"/") {
				t.Errorf("%s imports %s, which this package must not use", name, path)
			}
		}
	}
}

// checkSyntax reports a goroutine, a float type or an init function.
func checkSyntax(t *testing.T, name string, f *ast.File) {
	t.Helper()
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GoStmt:
			t.Errorf("%s starts a goroutine", name)
		case *ast.Ident:
			if x.Name == "float32" || x.Name == "float64" {
				t.Errorf("%s uses %s, want integer arithmetic only", name, x.Name)
			}
		case *ast.FuncDecl:
			if x.Name.Name == "init" && x.Recv == nil {
				t.Errorf("%s declares init", name)
			}
		}
		return true
	})
}

// checkNoGlobals reports every package-level variable.
func checkNoGlobals(t *testing.T, name string, f *ast.File) {
	t.Helper()
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.VAR {
			continue
		}
		for _, s := range g.Specs {
			for _, id := range s.(*ast.ValueSpec).Names {
				t.Errorf("%s declares the package variable %s", name, id.Name)
			}
		}
	}
}
