package economysim_test

import (
	"bytes"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/config"
)

// This file is the exception D688 of plan 2-24 states: these tests read files under
// examples/ from the root module (testing.md 10.2 discourages it). They read bytes
// and import lines only, and they are the only tests that do.

// exampleImportProblems returns one line per import of a Go file of examples/<name>/
// that architecture.md 1.3 forbids: a package under the module's internal/, or
// another example. src is the source, or nil to read the file.
func exampleImportProblems(file string, src any) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), file, src, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	own := strings.Split(filepath.ToSlash(file), "/")[1] // examples/<name>/...
	var out []string
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(p, modulePath+"/internal/") || p == modulePath+"/internal" {
			out = append(out, file+" imports "+p+" (internal/ is not public API)")
		}
		if other, ok := strings.CutPrefix(p, modulePath+"/examples/"); ok && strings.Split(other, "/")[0] != own {
			out = append(out, file+" imports "+p+" (an example imports another example)")
		}
	}
	return out, nil
}

// TestExamples_NoInternalImports verifies that no Go file under examples/ imports a
// package under the module's internal/, and that no example imports another example.
//
// Guard: architecture.md 1.3 says the public API is enough to build a complete
// server, and the examples are the proof. Go lets a nested module whose import path
// lies under the parent's path import the parent's internal/ packages, so the
// toolchain does not enforce it (D688); without this test an example could reach
// for an internal helper and the proof would be void.
func TestExamples_NoInternalImports(t *testing.T) {
	t.Parallel()
	var problems []string
	files := 0
	err := filepath.WalkDir("examples", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		files++
		p, err := exampleImportProblems(path, nil)
		problems = append(problems, p...)
		return err
	})
	if err != nil {
		t.Fatalf("walk examples/: %v", err)
	}
	if files == 0 {
		t.Fatal("no Go file under examples/, want the examples")
	}
	for _, p := range problems {
		t.Error(p)
	}
}

// TestExamples_NoInternalImportsFlagsAViolation verifies that the guard above can
// fail: an internal import and an import of another example are each reported, an
// import of the public API and of the same example are not.
//
// Guard: a guard that never fails guards nothing (testing.md 9.3).
func TestExamples_NoInternalImportsFlagsAViolation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		src  string
		want int
	}{
		{"internal package", `package p; import _ "` + modulePath + `/internal/checked"`, 1},
		{"another example", `package p; import _ "` + modulePath + `/examples/colony"`, 1},
		{"the same example", `package p; import _ "` + modulePath + `/examples/realm/sub"`, 0},
		{"the public api", `package p; import _ "` + modulePath + `/config"; import _ "` + modulePath + `"`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := exampleImportProblems("examples/realm/x.go", tt.src)
			if err != nil {
				t.Fatalf("exampleImportProblems(%q) = %v, want nil", tt.src, err)
			}
			if len(got) != tt.want {
				t.Errorf("exampleImportProblems(%q) = %q, want %d problems", tt.src, got, tt.want)
			}
		})
	}
}

// TestExamples_PackCopiesAgree verifies that the files an example embeds as copies of
// the root fixtures are byte-equal to them: the colony's pack to testdata/chains/second/,
// and the realm's warm document and expected figures to testdata/chains/. An example
// is its own module and cannot read the root testdata (testing.md 10.2), so it holds
// a copy, and this test is what stops the copy drifting from the fixture whose own
// tests run.
//
// Guard: D687. A drifted copy would let the example run a pack the reference tests
// no longer cover.
func TestExamples_PackCopiesAgree(t *testing.T) {
	t.Parallel()
	pairs := [][2]string{
		{"examples/colony/pack/second.toml", "testdata/chains/second/second.toml"},
		{"examples/colony/pack/warm-second.toml", "testdata/chains/second/warm-second.toml"},
		{"examples/realm/testdata/warm-five.toml", "testdata/chains/warm-five.toml"},
		{"examples/realm/testdata/expected-five.toml", "testdata/chains/expected/five.toml"},
	}
	for _, p := range pairs {
		t.Run(filepath.Base(p[0]), func(t *testing.T) {
			t.Parallel()
			got, err := os.ReadFile(p[0])
			if err != nil {
				t.Fatalf("read %s: %v", p[0], err)
			}
			if len(got) == 0 {
				t.Fatalf("%s is empty, want the copy of %s", p[0], p[1])
			}
			want, err := os.ReadFile(p[1])
			if err != nil {
				t.Fatalf("read %s: %v", p[1], err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s (%d bytes) differs from %s (%d bytes), want a byte copy", p[0], len(got), p[1], len(want))
			}
		})
	}
	extra, err := filepath.Glob("examples/colony/pack/*.toml")
	if err != nil {
		t.Fatalf("glob the colony pack: %v", err)
	}
	if len(extra) != 2 {
		t.Errorf("colony pack has %d files %v, want the 2 copies of testdata/chains/second/", len(extra), extra)
	}
}

// TestColony_SharesNoIdentifierWithTheRealm verifies that the commodity, recipe and
// role identifiers of the colony's pack (testdata/chains/second/) and of the realm's
// pack are disjoint sets: the second setting is unrelated to the first.
//
// Requirement: FR-REF-029
// Guard: REF 2 row 3 and C15. A second chain that reused an identifier of the first
// would not show that the library is setting-agnostic.
func TestColony_SharesNoIdentifierWithTheRealm(t *testing.T) {
	t.Parallel()
	colony, err := config.LoadFS(os.DirFS("testdata/chains"), "second/second.toml")
	if err != nil {
		t.Fatalf("LoadFS(second) = %v, want nil", err)
	}
	// The realm's layers, in the order of examples/realm/pack.go: cold, then modules.
	realmFS := os.DirFS("examples/realm/pack")
	mods, err := fs.Glob(realmFS, "modules/*.toml")
	if err != nil {
		t.Fatalf("glob the realm modules: %v", err)
	}
	layers := append([]string{"cold/base.toml", "cold/calendar.toml", "cold/lines.toml", "cold/five.toml"}, mods...)
	realm, err := config.LoadFS(realmFS, layers...)
	if err != nil {
		t.Fatalf("LoadFS(realm pack) = %v, want nil", err)
	}
	kinds := []struct {
		name         string
		colony, real []string
	}{{"commodity", nil, nil}, {"recipe", nil, nil}, {"role", nil, nil}}
	for _, c := range colony.Commodities {
		kinds[0].colony = append(kinds[0].colony, c.ID)
	}
	for _, c := range realm.Commodities {
		kinds[0].real = append(kinds[0].real, c.ID)
	}
	for _, r := range colony.Recipes {
		kinds[1].colony = append(kinds[1].colony, r.ID)
	}
	for _, r := range realm.Recipes {
		kinds[1].real = append(kinds[1].real, r.ID)
	}
	for _, r := range colony.Roles {
		kinds[2].colony = append(kinds[2].colony, r.ID)
	}
	for _, r := range realm.Roles {
		kinds[2].real = append(kinds[2].real, r.ID)
	}
	for _, k := range kinds {
		if len(k.colony) == 0 || len(k.real) == 0 {
			t.Fatalf("%s identifiers: colony %d, realm %d, want both sets non-empty", k.name, len(k.colony), len(k.real))
		}
		for _, id := range k.colony {
			if slices.Contains(k.real, id) {
				t.Errorf("%s %q is in both the colony and the realm pack", k.name, id)
			}
		}
	}
}
