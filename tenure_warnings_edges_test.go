package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestWarnings_TenureRentAboveOutputIsNotJudged verifies the cases in which
// rent_above_output stays silent although the rent of a term is far above the value of
// what the recipe makes: the term has no rent (a closing alone), the recipe has no
// output or an output of an unknown commodity or of quantity 0, no role of the region
// runs the recipe, and a sum of rent lines that does not fit in int64. Each is a case the recipe
// cannot be judged on, so the warning must not guess.
//
// Requirement: FR-LND-039
//
// Guard: without the cases a nil output or an overflow would panic or raise a warning
// about a recipe that has no value to compare, in a warning that must be safe on any
// configuration (the rule of Warnings).
func TestWarnings_TenureRentAboveOutputIsNotJudged(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
	}{
		{"a term that closes and sets no rent", func(c *economysim.Config) {
			tm := &c.Modules.Tenure.Terms[0]
			tm.HasRent, tm.Rent, tm.HasClosed, tm.Closed = false, 0, true, true
		}},
		{"a recipe with no output", func(c *economysim.Config) { recipeOf(c, "grow_staple").Outputs = nil }},
		{"an output of an unknown commodity", func(c *economysim.Config) {
			recipeOf(c, "grow_staple").Outputs[0].Commodity = "ghost"
		}},
		{"an output of quantity 0", func(c *economysim.Config) { recipeOf(c, "grow_staple").Outputs[0].Quantity = 0 }},
		{"a recipe no role of the region runs", func(c *economysim.Config) {
			for i := range c.Roles {
				c.Roles[i].Recipes = slices.DeleteFunc(slices.Clone(c.Roles[i].Recipes), func(id string) bool { return id == "grow_staple" })
			}
		}},
		{"a sum of rent lines that overflows", func(c *economysim.Config) {
			// One line is MaxQuantity * MaxPrice = 1e18, which fits in int64; ten of them sum to
			// 1e19, above its largest value of about 9.2e18.
			c.Modules.Tenure.Terms[0].Rent = economysim.MaxPrice
			draw := economysim.Draw{Resource: "arable", Quantity: economysim.MaxQuantity}
			recipeOf(c, "grow_staple").Draws = slices.Repeat([]economysim.Draw{draw}, 10)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tenureConfig()
			c.Modules.Tenure.Terms[0].Rent = 1_000_000 // above the value of one execution (see TestWarnings_Tenure)
			tt.mutate(&c)
			for _, w := range tenureWarned(c) {
				if w == "modules.tenure.term[0].recipe.grow_staple rent_above_output" {
					t.Errorf("Warnings(%s) holds %q, want no rent_above_output", tt.name, w)
				}
			}
		})
	}
}

// TestWarnings_TenureCapacityDefaultsToTheResource verifies that an endowment with no
// row of the resources module takes the resource's default capacity for the warnings:
// with a default of 0 both holdings are unused, with a positive default neither is.
//
// Requirement: FR-LND-039
func TestWarnings_TenureCapacityDefaultsToTheResource(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		capacity int64
		want     []string
	}{
		{"a default of 0", 0, []string{"modules.tenure.holding[0] holding_unused", "modules.tenure.holding[1] holding_unused"}},
		{"a positive default", 600, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tenureConfig()
			c.Modules.Resources.Endowments = nil
			c.Modules.Resources.Resources[0].Capacity = tt.capacity
			var got []string
			for _, w := range tenureWarned(c) {
				if slices.Contains([]string{"modules.tenure.holding[0] holding_unused", "modules.tenure.holding[1] holding_unused"}, w) {
					got = append(got, w)
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Warnings(no endowment row, default capacity %d) holding_unused = %q, want %q", tt.capacity, got, tt.want)
			}
		})
	}
}
