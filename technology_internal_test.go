package economysim

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/curve"
)

// The tests of the unexported helpers of technology_validate.go that the public
// validation reaches only through the whole overflow check.

// TestWithTech_OnlyAFieldRecipeGrows verifies the term of the overflow check of
// spec-technology.md 9.5: with the module off, or for a recipe in no field, the
// largest yield factor is returned as it was; for a recipe in a field it is
// multiplied by MaxTechBP over 10 000 (eleven times, with MaxGainBP at 100 000), and
// a recipe named in the second field is found as well as one in the first.
//
// The arithmetic: MaxTechBP is 10 000 + 100 000 = 110 000, so y * 110 000 / 10 000 is
// 11 y; 100 000 (the largest yield factor) gives 1 100 000, and 0 stays 0.
//
// Requirement: FR-TEC-020, NFR-TEC-005
func TestWithTech_OnlyAFieldRecipeGrows(t *testing.T) {
	t.Parallel()
	on := Config{}
	on.Modules.Technology = &Technology{Fields: []TechField{
		{ID: "a", Recipes: []string{"r1"}, CapBP: 1},
		{ID: "b", Recipes: []string{"r2", "r3"}, CapBP: 1},
	}}
	tests := []struct {
		name   string
		c      Config
		recipe string
		y      int64
		want   int64
	}{
		{"module off", Config{}, "r1", 100_000, 100_000},
		{"a recipe in no field", on, "r9", 100_000, 100_000},
		{"a recipe of the first field", on, "r1", 100_000, 1_100_000},
		{"a recipe of the second field", on, "r3", 7, 77},
		{"a yield of 0", on, "r2", 0, 0},
		{"the largest yield factor", on, "r1", MaxYieldBP, MaxYieldBP * MaxTechBP / 10_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := &validator{c: tt.c}
			if got := v.withTech(tt.recipe, tt.y); got != tt.want {
				t.Errorf("withTech(%q, %d) = %d, want %d", tt.recipe, tt.y, got, tt.want)
			}
		})
	}
}

// TestAbundanceAt_NoKnotsIsWhole verifies that a stock with no abundance curve reads
// 10 000 at every share, and that one with knots reads the curve, so the inequality
// of 10.3.1 compares like with like.
//
// Requirement: FR-TEC-030
func TestAbundanceAt_NoKnotsIsWhole(t *testing.T) {
	t.Parallel()
	for _, share := range []int64{0, 3_000, 10_000} {
		if got, ok := abundanceAt(nil, share); !ok || got != wholeBP {
			t.Errorf("abundanceAt(nil, %d) = %d, %v, want %d, true", share, got, ok, wholeBP)
		}
	}
	line := []curve.Knot{{X: 0, Y: 0}, {X: 10_000, Y: 10_000}}
	if got, ok := abundanceAt(line, 6_000); !ok || got != 6_000 {
		t.Errorf("abundanceAt(line, 6000) = %d, %v, want 6000, true", got, ok)
	}
}
