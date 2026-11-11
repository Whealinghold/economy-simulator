package economysim

import (
	"errors"
	"testing"
)

// TestCheapestHaulRecipe verifies that the recipe with the lowest Duration /
// Output wins and that a tie goes to the lower recipe ID, wherever it sits in the
// list.
//
// Requirement: FR-TRN-041
func TestCheapestHaulRecipe(t *testing.T) {
	t.Parallel()
	mk := func(id string, d int64, q Quantity) Recipe {
		return Recipe{ID: id, Duration: d, Outputs: []Output{{Commodity: "haul", Quantity: q}}}
	}
	for _, tc := range []struct {
		name string
		rs   []Recipe
		want haulRecipe
	}{
		{"lowest ratio", []Recipe{mk("a", 4, 4), mk("b", 2, 4), mk("c", 9, 4)}, haulRecipe{2, 4, true}},
		{"tie to the lower ID, listed last", []Recipe{mk("z", 4, 4), mk("a", 8, 8)}, haulRecipe{8, 8, true}},
		{"tie to the lower ID, listed first", []Recipe{mk("a", 4, 4), mk("z", 8, 8)}, haulRecipe{4, 4, true}},
		{"other commodity and zero output ignored", []Recipe{{ID: "x", Duration: 1, Outputs: []Output{{Commodity: "other", Quantity: 9}}}, mk("y", 1, 0)}, haulRecipe{}},
	} {
		if got := cheapest(&Config{Recipes: tc.rs}, "haul"); got != tc.want {
			t.Errorf("%s: cheapest = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

// TestTooManyLanes verifies the bound of the list: at the limit passes, above it
// is ErrOutOfBounds.
//
// Requirement: FR-TRN-041
func TestTooManyLanes(t *testing.T) {
	t.Parallel()
	if err := tooManyLanes(3, 3); err != nil {
		t.Errorf("tooManyLanes(3, 3) = %v, want nil", err)
	}
	if err := tooManyLanes(4, 3); !errors.Is(err, ErrOutOfBounds) {
		t.Errorf("tooManyLanes(4, 3) = %v, want ErrOutOfBounds", err)
	}
}
