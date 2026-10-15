package economysim

import (
	"slices"
	"testing"
)

// Equivalent mutants of recipe.go that no test can kill (gomu, one-line why each):
//   - line 65 `len(r.Inputs) == 0` -> `<= 0`: a length is never negative, so <= 0 is == 0.
//   - line 79 `len(m) == 0` -> `<= 0`: same reason.
//   - line 79 `m[len(m)-1] != id` -> `< id`: recipes are visited in ascending ID, so the
//     last maker is never greater than id and `<` agrees with `!=` on every reachable input.

// TestNewRecipeTable_RepeatedOutputNamesRecipeOnce pins the byOutput de-duplication: a
// recipe that lists the same non-discard commodity twice appears once in its makers, while
// both output rows are kept; a later recipe that makes it too is still appended.
//
// Guard: without the check the maker list would repeat the recipe, and a consumer that
// walks makers would count it twice.
//
// Requirement: FR-PRO-032
func TestNewRecipeTable_RepeatedOutputNamesRecipeOnce(t *testing.T) {
	t.Parallel()
	c := Config{
		Commodities: []Commodity{{ID: "ash"}, {ID: "bread"}},
		Recipes: []Recipe{
			{ID: "bake", Outputs: []Output{
				{Commodity: "bread", Quantity: 2, CostShare: 5000},
				{Commodity: "bread", Quantity: 1, CostShare: 5000},
				{Commodity: "ash", Quantity: 1}}},
			{ID: "fire", Outputs: []Output{{Commodity: "bread", Quantity: 1, CostShare: 10000}}},
		},
	}
	cc := c.canonical()
	tab := newRecipeTable(&cc)
	if got, want := tab.makers(1), []RecipeID{0, 1}; !slices.Equal(got, want) {
		t.Errorf("makers(bread) = %v, want %v", got, want)
	}
	if got, want := tab.makers(0), []RecipeID{0}; !slices.Equal(got, want) {
		t.Errorf("makers(ash) = %v, want %v", got, want)
	}
	if got := len(tab.row(0).outputs); got != 3 {
		t.Errorf("row(bake) has %d outputs, want 3 (the repeat is kept)", got)
	}
}
