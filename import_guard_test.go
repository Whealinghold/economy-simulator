package economysim_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const (
	modulePath = "github.com/Whealinghold/economy-sim"
	tomlPath   = "github.com/pelletier/go-toml/v2"
)

// importProblems returns one line per import of the file name (a path
// relative to the module root, with slashes) that the hard constraints of
// AGENTS.md and architecture.md 2.7 and 2.9 forbid: a path outside the
// standard library and the module, a third-party parser anywhere but in
// config/, a package of the module that the importing package may not import
// (internal/ never imports the root, config/, strategy/, indicator/ or
// experiment/; the root imports none of config/, strategy/, indicator/,
// experiment/), and math/big outside the non-test files of internal/steady
// (architecture.md 3.5). It reads the imports only.
func importProblems(name string, src any) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	top := "."
	if i := strings.Index(name, "/"); i >= 0 {
		top = name[:i]
	}
	// the packages of the module that the importer may not import
	var banned []string
	switch top {
	case ".":
		banned = []string{"config", "strategy", "indicator", "experiment"}
	case "internal":
		banned = []string{"", "config", "strategy", "indicator", "experiment"}
	}
	var out []string
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, err
		}
		report := func(why string) {
			out = append(out, fmt.Sprintf("%s: import %q: %s", name, p, why))
		}
		first, _, _ := strings.Cut(p, "/")
		switch {
		case p == "math/big":
			if !strings.HasPrefix(name, "internal/steady/") {
				report("math/big is imported by internal/steady only (architecture.md 3.5)")
			}
		case p == tomlPath || strings.HasPrefix(p, tomlPath+"/"):
			if top != "config" {
				report("a file-format parser is imported by config/ only")
			}
		case p == modulePath || strings.HasPrefix(p, modulePath+"/"):
			rel := strings.TrimPrefix(strings.TrimPrefix(p, modulePath), "/")
			if i := strings.Index(rel, "/"); i >= 0 {
				rel = rel[:i]
			}
			for _, b := range banned {
				if rel == b {
					report("a package of the module that " + top + " may not import")
				}
			}
		case strings.Contains(first, "."):
			report("not the standard library or the module")
		}
	}
	return out, nil
}

// TestImportGuard_Detects verifies the checker on in-memory sources, so the
// guard below cannot pass because it looks at nothing.
//
// Guard: a checker that missed a third-party import, a layer inversion or a
// parser outside config/ would let the standard-library-only rule lapse with
// every test green.
func TestImportGuard_Detects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		file string
		src  string
		want int
	}{
		{"root stdlib", "a.go", `package p; import "fmt"; import "slices"`, 0},
		{"root third party", "a.go", `package p; import "github.com/google/uuid"`, 1},
		{"root parser", "a.go", `package p; import "` + tomlPath + `"`, 1},
		{"root imports config", "a.go", `package p; import "` + modulePath + `/config"`, 1},
		{"root imports internal", "a.go", `package p; import "` + modulePath + `/internal/checked"`, 0},
		{"internal third party", "internal/x/x.go", `package p; import "golang.org/x/exp/slices"`, 1},
		{"internal imports root", "internal/x/x.go", `package p; import "` + modulePath + `"`, 1},
		{"internal imports config", "internal/x/x.go", `package p; import "` + modulePath + `/config"`, 1},
		{"internal imports indicator", "internal/x/x.go", `package p; import "` + modulePath + `/indicator"`, 1},
		{"internal imports internal", "internal/x/x.go", `package p; import "` + modulePath + `/internal/y"`, 0},
		{"internal parser", "internal/x/x.go", `package p; import "` + tomlPath + `"`, 1},
		{"config parser", "config/c.go", `package p; import "` + tomlPath + `"; import "` + modulePath + `"`, 0},
		{"config other third party", "config/c.go", `package p; import "gopkg.in/yaml.v3"`, 1},
		{"indicator parser", "indicator/i.go", `package p; import "` + tomlPath + `"`, 1},
		{"steady big", "internal/steady/s.go", `package p; import "math/big"`, 0},
		{"steady subdirectory big", "internal/steady/x/s.go", `package p; import "math/big"`, 0},
		{"root big", "a.go", `package p; import "math/big"`, 1},
		{"other internal big", "internal/checked/c.go", `package p; import "math/big"`, 1},
		{"lookalike steady big", "internal/steadyx/s.go", `package p; import "math/big"`, 1},
		{"config big", "config/c.go", `package p; import "math/big"`, 1},
		{"big sub-package is not math/big", "a.go", `package p; import "math/bits"`, 0},
		{"lookalike module prefix", "a.go", `package p; import "` + modulePath + `-evil/x"`, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := importProblems(tt.file, tt.src)
			if err != nil {
				t.Fatalf("importProblems(%q) error = %v", tt.file, err)
			}
			if len(got) != tt.want {
				t.Errorf("importProblems(%q, %q) = %v, want %d problems", tt.file, tt.src, got, tt.want)
			}
		})
	}
}

// TestImports_StdlibAndLayers verifies, for every non-test file in the scope
// of guardedFiles, that the root package, internal/**, strategy/, indicator/
// and experiment/ import the standard library and the module only, that
// internal/ never imports a package above it, that the root imports none of
// config/, strategy/, indicator/ and experiment/, and that only config/
// imports the TOML parser.
//
// Guard: AGENTS.md "Standard library only in the core" and architecture.md
// 2.7 and 2.9. A dependency or an inverted import breaks the consumer's
// build graph, and no behavioural test would point at it.
func TestImports_StdlibAndLayers(t *testing.T) {
	t.Parallel()
	files, err := guardedFiles(".")
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(files) < 10 {
		t.Fatalf("checked %d files, want at least 10: the walk is looking at the wrong place", len(files))
	}
	for _, file := range files {
		file = filepath.ToSlash(file)
		problems, err := importProblems(file, nil)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, p := range problems {
			t.Errorf("%s", p)
		}
	}
}
