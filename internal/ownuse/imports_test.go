package ownuse_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOwnuse_ImportsOnly verifies that no non-test file of the package imports
// anything but the standard library and internal/checked, and none imports
// math/rand, time or a floating-point package of math (the rules take no
// random number and no float).
//
// Requirement: NFR-SUB-002
//
// Guard: architecture.md 2.9 and 3.5. An import of the root or of a random
// source would break the layering or the determinism, and no behavioural test
// would point at it.
func TestOwnuse_ImportsOnly(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob: %v, %d files", err, len(files))
	}
	const checked = "github.com/Whealinghold/economy-sim/internal/checked"
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		src, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, imp := range src.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			std := !strings.Contains(strings.SplitN(path, "/", 2)[0], ".")
			switch {
			case path == checked:
			case std && path != "time" && !strings.HasPrefix(path, "math/rand") && !strings.HasPrefix(path, "crypto"):
			default:
				t.Errorf("%s imports %q, want the standard library and internal/checked only", file, path)
			}
		}
	}
	if _, err := os.Stat("ownuse.go"); err != nil {
		t.Fatalf("ownuse.go: %v", err)
	}
}
