package economysim_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// randConstructors are the names of math/rand and math/rand/v2 that build a
// seeded source or name a type: the only uses architecture.md 7.3 allows.
var randConstructors = []string{
	"New", "NewPCG", "NewChaCha8", "NewSource", "NewZipf",
	"PCG", "ChaCha8", "Rand", "Source", "Source64", "Zipf",
}

// bannedUses returns one line per use in src of what architecture.md 7.1 to
// 7.3, 11.1 and 11.2 forbid: the global math/rand functions, crypto/rand,
// time.Now and time.Since, slog.Default, fmt.Print*, log.Print*, Fatal* and
// Panic*, the builtins print and println, os.Stdout and os.Stderr, an
// init function, and a panic whose message has no string literal containing
// "invariant" (architecture.md 8.1). It reads the syntax tree, so a mention in a comment or a
// string is not a use, and an import renamed to another name is still found.
func bannedUses(name string, src any) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var out []string
	report := func(pos token.Pos, what string) {
		out = append(out, fmt.Sprintf("%s: %s", fset.Position(pos), what))
	}
	pkgs := map[string]string{} // local import name -> import path
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, err
		}
		local := path[strings.LastIndex(path, "/")+1:]
		if path == "math/rand/v2" {
			local = "rand"
		}
		if imp.Name != nil {
			local = imp.Name.Name
		}
		pkgs[local] = path
		if path == "crypto/rand" {
			report(imp.Pos(), "import of crypto/rand")
		}
	}
	// A local variable that shadows an import or a builtin makes the check
	// over-report, never under-report; no file of the module does it.
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncDecl:
			if n.Recv == nil && n.Name.Name == "init" {
				report(n.Pos(), "init function")
			}
		case *ast.SelectorExpr:
			id, ok := n.X.(*ast.Ident)
			if !ok {
				return true
			}
			path, sel := pkgs[id.Name], n.Sel.Name
			switch {
			case (path == "math/rand" || path == "math/rand/v2") && !slices.Contains(randConstructors, sel):
				report(n.Pos(), "global "+path+"."+sel)
			case path == "time" && (sel == "Now" || sel == "Since"):
				report(n.Pos(), "time."+sel)
			case path == "log/slog" && sel == "Default":
				report(n.Pos(), "slog.Default")
			case path == "fmt" && strings.HasPrefix(sel, "Print"):
				report(n.Pos(), "fmt."+sel)
			case path == "log" && (strings.HasPrefix(sel, "Print") || strings.HasPrefix(sel, "Fatal") || strings.HasPrefix(sel, "Panic")):
				report(n.Pos(), "log."+sel)
			case path == "os" && (sel == "Stdout" || sel == "Stderr"):
				report(n.Pos(), "os."+sel)
			}
		case *ast.CallExpr:
			id, ok := n.Fun.(*ast.Ident)
			switch {
			case ok && (id.Name == "print" || id.Name == "println"):
				report(n.Pos(), "builtin "+id.Name)
			case ok && id.Name == "panic" && !panicSaysInvariant(n):
				report(n.Pos(), `panic without "invariant" in its message`)
			}
		}
		return true
	})
	return out, nil
}

// panicSaysInvariant reports whether a call of panic holds a string literal
// that contains "invariant", in any argument expression (a literal, a
// concatenation, or the format of fmt.Sprintf or errors.New).
func panicSaysInvariant(call *ast.CallExpr) bool {
	found := false
	for _, arg := range call.Args {
		ast.Inspect(arg, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING && strings.Contains(lit.Value, "invariant") {
				found = true
			}
			return !found
		})
	}
	return found
}

// guardedFiles returns the non-test Go files of the module at dir that the
// guards cover, relative to dir: the root package (not its subdirectories),
// internal/**, config/**, and strategy/, indicator/ and experiment/ when they
// exist. examples/ and otelobserver/ are separate modules and testdata holds
// no source of the module.
func guardedFiles(dir string) ([]string, error) {
	var files []string
	for _, root := range []string{".", "internal", "config", "strategy", "indicator", "experiment"} {
		err := filepath.WalkDir(filepath.Join(dir, root), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if root != "." && os.IsNotExist(err) {
					return nil // an optional directory that does not exist yet
				}
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" || (root == "." && path != filepath.Join(dir, root)) {
					return fs.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
				rel, err := filepath.Rel(dir, path)
				if err != nil {
					return err
				}
				files = append(files, rel)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

// TestGuardedFiles_Scope verifies the scope function on a small tree.
//
// Guard: a walk that missed strategy/, indicator/ or experiment/, or that
// descended into a separate module, would let a violation through or fail on
// code the rules do not cover.
func TestGuardedFiles_Scope(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, name := range []string{
		"a.go", "a_test.go", "doc/x.go", "examples/e/e.go", "otelobserver/o.go",
		"internal/p/p.go", "internal/p/p_test.go", "internal/p/q/q.go",
		"internal/p/testdata/t.go", "config/c.go", "indicator/i.go",
		"experiment/e/e.go", "testdata/t.go",
	} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("package p\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := guardedFiles(dir)
	if err != nil {
		t.Fatalf("guardedFiles error = %v", err)
	}
	want := []string{
		"a.go", "internal/p/p.go", "internal/p/q/q.go", "config/c.go",
		"indicator/i.go", "experiment/e/e.go",
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("guardedFiles = %v, want %v", got, want)
	}
}

// TestBannedUses_Detects verifies the checker on small sources, so the guard
// below cannot pass because it looks at nothing.
//
// Guard: a checker that silently misses a use would let the determinism and
// observability rules lapse with every test green.
//
// Requirement: NFR-WLD-003
func TestBannedUses_Detects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		src  string
		want int
	}{
		{"clean", `package p; import "math/rand/v2"; func f() { _ = rand.New(rand.NewPCG(1, 2)) }`, 0},
		{"global rand function", `package p; import "math/rand/v2"; func f() { _ = rand.IntN(3) }`, 1},
		{"old global rand", `package p; import "math/rand"; func f() { _ = rand.Intn(3) }`, 1},
		{"renamed rand import", `package p; import r "math/rand/v2"; func f() { _ = r.Float64() }`, 1},
		{"crypto rand", `package p; import "crypto/rand"; func f() { _ = rand.Reader }`, 1},
		{"time now", `package p; import "time"; func f() { _ = time.Now() }`, 1},
		{"time since", `package p; import "time"; func f() { _ = time.Since(time.Time{}) }`, 1},
		{"time duration is fine", `package p; import "time"; var _ = time.Second`, 0},
		{"slog default", `package p; import "log/slog"; func f() { _ = slog.Default() }`, 1},
		{"slog new is fine", `package p; import "log/slog"; func f() { _ = slog.New(slog.DiscardHandler) }`, 0},
		{"fmt print", `package p; import "fmt"; func f() { fmt.Println(1); fmt.Printf("x") }`, 2},
		{"fmt sprintf is fine", `package p; import "fmt"; func f() { _ = fmt.Sprintf("x") }`, 0},
		{"log print", `package p; import "log"; func f() { log.Printf("x"); log.Fatal(1) }`, 2},
		{"builtin print", `package p; func f() { print(1); println(2) }`, 2},
		{"os std streams", `package p; import "os"; func f() { _ = os.Stdout; _ = os.Stderr }`, 2},
		{"init", `package p; func init() {}`, 1},
		{"method named init is fine", `package p; type T struct{}; func (T) init() {}`, 0},
		{"panic with invariant", `package p; func f() { panic("economysim: invariant violated: x") }`, 0},
		{"panic with invariant in sprintf", `package p; import "fmt"; func f() { panic(fmt.Sprintf("invariant: %d", 1)) }`, 0},
		{"panic without invariant", `package p; func f() { panic("bad") }`, 1},
		{"panic of an error value", `package p; func f(err error) { panic(err) }`, 1},
		{"mention in comment and string", `package p; // time.Now() and fmt.Println
var s = "rand.Intn(3) time.Now()"`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := bannedUses("src.go", tt.src)
			if err != nil {
				t.Fatalf("bannedUses(%q) error = %v", tt.src, err)
			}
			if len(got) != tt.want {
				t.Errorf("bannedUses(%q) = %v, want %d uses", tt.src, got, tt.want)
			}
		})
	}
}

// TestNoBannedUses_InNonTestFiles verifies that no non-test file of the root
// package, internal/, config/, strategy/, indicator/ or experiment/ uses the global math/rand functions,
// crypto/rand, time.Now, time.Since, slog.Default, fmt.Print*, log.Print*,
// the builtins print and println, os.Stdout or os.Stderr, that none
// declares an init function, and that every panic says "invariant" (the scope
// is guardedFiles).
//
// Guard: these are the rules that keep a run reproducible and the library
// silent (architecture.md 7.1 to 7.3, 11.1, 11.2). A single stray call
// breaks bit-identical output on one platform only, or writes to a
// consumer's terminal, and no behavioural test would point at it.
//
// Requirement: NFR-WLD-003
func TestNoBannedUses_InNonTestFiles(t *testing.T) {
	t.Parallel()
	files, err := guardedFiles(".")
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	checked := len(files)
	for _, path := range files {
		uses, err := bannedUses(path, nil)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, u := range uses {
			t.Errorf("forbidden use: %s", u)
		}
	}
	if checked < 10 {
		t.Errorf("checked %d files, want at least 10: the walk is looking at the wrong place", checked)
	}
}
