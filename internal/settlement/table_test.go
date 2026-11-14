package settlement_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/settlement"
)

// TestTable_NoLimitVersusZero verifies that an absent limit row is no limit and a
// row of 0 is a closed gate, and that a flag is read per class
// (spec-settlements.md 3.3).
//
// Requirement: FR-SET-005
func TestTable_NoLimitVersusZero(t *testing.T) {
	t.Parallel()
	tb, ok := settlement.NewTable(2, []settlement.Class{
		{},
		{Flags: []int{0}, Limits: []settlement.Limit{{Index: 0, Value: 2}, {Index: 1, Value: 0}}},
		{Flags: []int{0, 63}},
	})
	if !ok {
		t.Fatal("NewTable not ok")
	}
	if tb.Classes() != 3 {
		t.Errorf("Classes() = %d, want 3", tb.Classes())
	}
	tests := []struct {
		name         string
		class, limit int
		want         int64
		wantOK       bool
	}{
		{"class without the row has no limit", 0, 0, 0, false},
		{"row of 2", 1, 0, 2, true},
		{"row of 0 is a closed gate", 1, 1, 0, true},
		{"larger class omits the limit", 2, 0, 0, false},
		{"limit out of range", 1, 2, 0, false},
		{"class out of range", 3, 0, 0, false},
		{"negative class", -1, 0, 0, false},
		{"negative limit", 1, -1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := tb.Limit(tt.class, tt.limit); got != tt.want || ok != tt.wantOK {
				t.Errorf("Limit(%d, %d) = %d, %v, want %d, %v", tt.class, tt.limit, got, ok, tt.want, tt.wantOK)
			}
		})
	}
	flags := []struct {
		class, flag int
		want        bool
	}{{0, 0, false}, {1, 0, true}, {1, 1, false}, {2, 63, true}, {2, 64, false}, {3, 0, false}, {-1, 0, false}, {1, -1, false}}
	for _, f := range flags {
		if got := tb.Has(f.class, f.flag); got != f.want {
			t.Errorf("Has(%d, %d) = %v, want %v", f.class, f.flag, got, f.want)
		}
	}
}

// TestTable_RefusesBadInput verifies that a table is refused for a flag or limit
// index outside the word, a negative value, a limit twice in one class and a
// limit count above the word.
//
// Requirement: FR-SET-004
func TestTable_RefusesBadInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		limits  int
		classes []settlement.Class
	}{
		{"flag too large", 1, []settlement.Class{{Flags: []int{64}}}},
		{"negative flag", 1, []settlement.Class{{Flags: []int{-1}}}},
		{"limit index too large", 1, []settlement.Class{{Limits: []settlement.Limit{{Index: 1}}}}},
		{"negative limit index", 1, []settlement.Class{{Limits: []settlement.Limit{{Index: -1}}}}},
		{"negative value", 1, []settlement.Class{{Limits: []settlement.Limit{{Index: 0, Value: -1}}}}},
		{"limit twice", 1, []settlement.Class{{Limits: []settlement.Limit{{Index: 0}, {Index: 0, Value: 3}}}}},
		{"too many limits", 65, nil},
		{"negative limit count", -1, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tb, ok := settlement.NewTable(tt.limits, tt.classes); ok || tb != nil {
				t.Errorf("NewTable(%d, %v) = %v, %v, want nil, false", tt.limits, tt.classes, tb, ok)
			}
		})
	}
	if tb, ok := settlement.NewTable(64, []settlement.Class{{Limits: []settlement.Limit{{Index: 63, Value: 9}}}}); !ok {
		t.Error("NewTable(64 limits) not ok, want ok")
	} else if v, ok := tb.Limit(0, 63); !ok || v != 9 {
		t.Errorf("Limit(0, 63) = %d, %v, want 9, true", v, ok)
	}
}

// TestTable_ReadsAllocateNothing verifies that Has and Limit allocate nothing.
//
// Requirement: FR-SET-020
func TestTable_ReadsAllocateNothing(t *testing.T) {
	tb, ok := settlement.NewTable(1, []settlement.Class{{Flags: []int{3}, Limits: []settlement.Limit{{Index: 0, Value: 5}}}})
	if !ok {
		t.Fatal("NewTable not ok")
	}
	var sink bool
	var n int64
	allocs := testing.AllocsPerRun(100, func() {
		sink = tb.Has(0, 3)
		n, _ = tb.Limit(0, 0)
	})
	if allocs != 0 || !sink || n != 5 {
		t.Errorf("Has and Limit allocate %v per run (sink %v, n %d), want 0", allocs, sink, n)
	}
}

// TestHeadroom_NeverNegative verifies max(0, limit - households), the clamp of
// negative inputs and the largest values.
//
// Requirement: FR-SET-023
func TestHeadroom_NeverNegative(t *testing.T) {
	t.Parallel()
	const maxInt64 = 1<<63 - 1
	tests := []struct {
		name         string
		limit, house int64
		want         int64
	}{
		{"room left", 12, 9, 3},
		{"exactly full", 12, 12, 0},
		{"above the ceiling has none", 12, 15, 0},
		{"closed gate", 0, 0, 0},
		{"negative limit", -3, 0, 0},
		{"negative households", 4, -2, 4},
		{"largest", maxInt64, 0, maxInt64},
		{"largest against largest", maxInt64, maxInt64, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := settlement.Headroom(tt.limit, tt.house); got != tt.want {
				t.Errorf("Headroom(%d, %d) = %d, want %d", tt.limit, tt.house, got, tt.want)
			}
		})
	}
}

// TestPackage_NoRootNoName verifies that the package imports only the standard
// library and internal/checked, uses no floating-point type, no time or random
// import, and has no identifier that names a region, a commodity or a role, so
// the rules work over primitive numbers.
//
// Requirement: NFR-SET-003
//
// Guard: architecture.md 2.9, 3.5 and 9.1. An import of the root, a float in the
// arithmetic or an identifier that branches on a domain name would break the
// layering or the determinism, and no behavioural test would point at it.
func TestPackage_NoRootNoName(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob: %v, %d files", err, len(files))
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, imp := range f.Imports {
			if msg := badImport(strings.Trim(imp.Path.Value, `"`)); msg != "" {
				t.Errorf("%s imports %s", file, msg)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				if msg := badIdent(id.Name); msg != "" {
					t.Errorf("%s: identifier %q %s", file, id.Name, msg)
				}
			}
			return true
		})
	}
}

// badImport returns why an import path is not allowed in the package, or "".
func badImport(path string) string {
	std := !strings.Contains(strings.SplitN(path, "/", 2)[0], ".")
	if path != "github.com/Whealinghold/economy-sim/internal/checked" && (!std || path == "math/rand" || path == "math/rand/v2" || path == "time") {
		return path + ", want the standard library (no rand, no time) and internal/checked only"
	}
	return ""
}

// badIdent returns why an identifier is not allowed in the package, or "".
func badIdent(name string) string {
	low := strings.ToLower(name)
	if low == "float32" || low == "float64" {
		return "is floating point (architecture.md 3.5)"
	}
	for _, bad := range []string{"region", "commodity", "role", "agent"} {
		if strings.Contains(low, bad) {
			return "names a " + bad + ", want primitive names only"
		}
	}
	return ""
}
