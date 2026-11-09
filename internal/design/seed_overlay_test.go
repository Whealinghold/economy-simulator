package design_test

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/design"
)

// TestSeed_SameAndPerPoint verifies the seeds of worked.out section E (Seed0 1,
// 4 replicas, 3 points) for both rules, and the wrap modulo 2^64 at the largest
// Seed0.
//
// Requirement: FR-EXP-010
func TestSeed_SameAndPerPoint(t *testing.T) {
	t.Parallel()
	for _, perPoint := range []bool{false, true} {
		for p := range 3 {
			var got []uint64
			for r := range 4 {
				got = append(got, design.Seed(1, perPoint, p, r, 4))
			}
			// worked.out E: same [1 2 3 4] at every point; per point [1..4], [5..8], [9..12].
			want := []uint64{1, 2, 3, 4}
			if perPoint {
				base := uint64(4*p) + 1
				want = []uint64{base, base + 1, base + 2, base + 3}
			}
			if !slices.Equal(got, want) {
				t.Errorf("Seed(1, perPoint %v, p %d, r 0..3, 4) = %v, want %v", perPoint, p, got, want)
			}
		}
	}
	// Seed0 = 2^64 - 1: the next seed wraps to 0, and the per-point seed of point 2, replica 3 of 4 is 2^64 - 1 + 11 = 10.
	if got := design.Seed(math.MaxUint64, false, 0, 1, 4); got != 0 {
		t.Errorf("Seed(MaxUint64, same, 0, 1, 4) = %d, want 0 (modulo 2^64)", got)
	}
	if got := design.Seed(math.MaxUint64, true, 2, 3, 4); got != 10 {
		t.Errorf("Seed(MaxUint64, per point, 2, 3, 4) = %d, want 10 (modulo 2^64)", got)
	}
}

// TestOverlay_Bytes verifies the bytes of the document of a change: worked.out
// section D, a path of three segments, a negative value, zero and
// math.MinInt64. The text is fixed byte for byte, with LF ends.
//
// Requirement: FR-EXP-003
func TestOverlay_Bytes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path  string
		value int64
		want  string
	}{
		{"world.cost_floor", 8000, "[world]\ncost_floor = 8000\n"},
		{"world.belief_step", 400, "[world]\nbelief_step = 400\n"},
		{"role.worker_a.target_ratio", 12500, "[role.worker_a]\ntarget_ratio = 12500\n"},
		{"modules.migration.move_below", -3, "[modules.migration]\nmove_below = -3\n"},
		{"world.x", 0, "[world]\nx = 0\n"},
		{"world.x", math.MinInt64, "[world]\nx = -9223372036854775808\n"},
		{"world.x", math.MaxInt64, "[world]\nx = 9223372036854775807\n"},
		{"a.b.c.d.e.f.g.h", 1, "[a.b.c.d.e.f.g]\nh = 1\n"},
	}
	for _, tt := range cases {
		got, err := design.Overlay(tt.path, tt.value)
		if err != nil || string(got) != tt.want {
			t.Errorf("Overlay(%q, %d) = %q, %v, want %q", tt.path, tt.value, got, err, tt.want)
		}
	}
	if got, err := design.Overlay("World.x", 1); err == nil {
		t.Errorf("Overlay(%q) = %q, nil, want ErrPath", "World.x", got)
	} else if err != design.ErrPath {
		t.Errorf("Overlay(%q) error = %v, want ErrPath", "World.x", err)
	}
}

// TestValidPath_Syntax verifies the path syntax of spec-experiments.md 3.2.1: two
// to eight segments of [a-z][a-z0-9_]*, at most 64 bytes each; and the ways to
// break each part of it.
//
// Requirement: FR-EXP-004
func TestValidPath_Syntax(t *testing.T) {
	t.Parallel()
	seg64 := strings.Repeat("a", 64)
	cases := []struct {
		name string
		path string
		want bool
	}{
		{"two segments", "world.cost_floor", true},
		{"eight segments", "a.b.c.d.e.f.g.h", true},
		{"digits and underscores after the first", "a1_b.c_2", true},
		{"a 64-byte segment", seg64 + ".x", true},
		{"a 65-byte segment", seg64 + "a.x", false},
		{"one segment", "world", false},
		{"nine segments", "a.b.c.d.e.f.g.h.i", false},
		{"empty", "", false},
		{"empty segment", "world..x", false},
		{"leading dot", ".world.x", false},
		{"trailing dot", "world.x.", false},
		{"uppercase", "World.x", false},
		{"leading digit", "world.1x", false},
		{"leading underscore", "world._x", false},
		{"hyphen", "world.cost-floor", false},
		{"space", "world. x", false},
		{"non-ASCII letter", "world.é", false},
	}
	for _, tt := range cases {
		if got := design.ValidPath(tt.path); got != tt.want {
			t.Errorf("%s: ValidPath(%q) = %v, want %v", tt.name, tt.path, got, tt.want)
		}
	}
}

// TestCheckedCount_Overflow verifies that products that wrap return false, that
// the empty product is 1, and that a zero or a negative factor behaves.
//
// Requirement: FR-EXP-064
func TestCheckedCount_Overflow(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		in     []int
		want   int
		wantOK bool
	}{
		{"none", nil, 1, true},
		{"small", []int{4, 3, 2, 5}, 120, true},
		{"zero", []int{math.MaxInt, 0}, 0, true},
		{"largest", []int{math.MaxInt, 1}, math.MaxInt, true},
		{"wraps 2^64", []int{1 << 32, 1 << 32}, 0, false},
		{"just over int", []int{math.MaxInt/2 + 1, 2}, 0, false},
		{"three that wrap", []int{1 << 30, 1 << 30, 1 << 30}, 0, false},
		{"negative", []int{3, -1}, 0, false},
	}
	for _, tt := range cases {
		if got, ok := design.CheckedCount(tt.in...); ok != tt.wantOK || (ok && got != tt.want) {
			t.Errorf("%s: CheckedCount(%v) = %d, %v, want %d, %v", tt.name, tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

// FuzzOverlay checks, for any path and value, that Overlay does not panic, that it
// succeeds exactly when ValidPath does, and that a document has the shape of two
// lines: a header in brackets and a key set to the decimal value. That the loader
// accepts or rejects such a document, never both outcomes, is tested in the
// experiment package, which may import it.
//
// Requirement: NFR-EXP-005
func FuzzOverlay(f *testing.F) {
	f.Add("world.cost_floor", int64(8000))
	f.Add("a.b", int64(math.MinInt64))
	f.Add("A.b", int64(1))
	f.Add("", int64(0))
	f.Fuzz(func(t *testing.T, path string, value int64) {
		doc, err := design.Overlay(path, value)
		if valid := design.ValidPath(path); valid != (err == nil) {
			t.Fatalf("ValidPath(%q) = %v but Overlay error = %v", path, valid, err)
		}
		if err != nil {
			return
		}
		lines := strings.Split(string(doc), "\n")
		if len(lines) != 3 || lines[2] != "" || !strings.HasPrefix(lines[0], "[") || !strings.HasSuffix(lines[0], "]") ||
			!strings.Contains(lines[1], " = ") {
			t.Fatalf("Overlay(%q, %d) = %q, want a header line and a key line", path, value, doc)
		}
	})
}
