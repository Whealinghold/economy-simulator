package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestValidate_OverflowReadsHireYieldBP verifies the overflow term of
// spec-employment.md 14.1 (the debt N-430 of the plan): a recipe on the hire list of an
// employer role has its largest yield factor multiplied by HireYieldBP over 10 000, so
// a product that is at the bound with the default is above it with the largest
// HireYieldBP, and a recipe on no hire list, the payroll form and a HireYieldBP of
// 10 000 are as they were.
//
// The arithmetic (one employer of weight 1): the largest yield of a recipe that is not
// Sited is 10 000 * MaxYieldBP / 10 000 = 100 000, so 5 * 10^7 units of output give
// 5 * 10^7 * 100 000 / 10 000 = 5 * 10^8, below MaxQuantity (10^9) and valid; with
// HireYieldBP at MaxYieldBP the factor is 10 times larger, 5 * 10^9, and fails.
//
// Requirement: FR-EMP-003, FR-EMP-007
func TestValidate_OverflowReadsHireYieldBP(t *testing.T) {
	t.Parallel()
	outPath := []string{"recipes", "mine", "outputs", "grain", "quantity"}
	build := func(m *economysim.Employment, hireYield int64, list ...string) economysim.Config {
		c := minimalConfig()
		c.Groups[0].Households = 1
		r := economysim.DefaultRecipe()
		r.ID = "mine"
		r.Outputs = []economysim.Output{{Commodity: "grain", Quantity: 50_000_000, CostShare: 10_000}}
		c.Recipes = append(c.Recipes, r)
		c.Roles[0].Recipes = []string{"mine"}
		c.Roles[0].Employer = &economysim.Employer{Recipes: list, MaxHires: 1, WorkerShareBP: 5_000, HireYieldBP: hireYield}
		c.Modules.Employment = m
		return c
	}
	hire := &economysim.Employment{Hire: true, MaxOpen: 10}
	tests := []struct {
		name string
		c    economysim.Config
		want bool
	}{
		{"the largest HireYieldBP on the hire list", build(hire, economysim.MaxYieldBP, "mine"), true},
		{"the default HireYieldBP", build(hire, 10_000, "mine"), false},
		{"a HireYieldBP of 1", build(hire, 1, "mine"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vs, _ := violations(tt.c.Validate())
			if got := hasPath(vs, economysim.ErrOverflow, outPath) != nil; got != tt.want {
				t.Errorf("Validate(%s): ErrOverflow at %v = %t, want %t; paths: %v", tt.name, outPath, got, tt.want, pathsOf(vs))
			}
		})
	}
}
