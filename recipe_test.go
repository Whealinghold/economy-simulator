package economysim

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

// bakeryConfig is a small canonical configuration for the recipe table.
// Commodities sort to ash 0, bread 1, flour 2, grain 3, oven 4. Recipes sort
// to bake 0, grind 1, mill 2, sow 3. Lists are written out of order on
// purpose, so canonical() is what puts them in commodity order.
func bakeryConfig() *Config {
	c := Config{
		Commodities: []Commodity{{ID: "oven"}, {ID: "grain"}, {ID: "flour"}, {ID: "bread"}, {ID: "ash"}},
		Recipes: []Recipe{
			{ID: "sow", Outputs: []Output{{Commodity: "grain", Quantity: 4, CostShare: 10000}}},
			{ID: "mill",
				Inputs: []Item{{Commodity: "grain", Quantity: 3}},
				// ash is a discard: created and disposed of in one step.
				Outputs: []Output{{Commodity: "flour", Quantity: 2, CostShare: 10000}, {Commodity: "ash", Quantity: 1, Discard: true}}},
			{ID: "grind",
				Inputs:  []Item{{Commodity: "grain", Quantity: 5}},
				Outputs: []Output{{Commodity: "flour", Quantity: 3, CostShare: 10000}}},
			{ID: "bake",
				Inputs: []Item{{Commodity: "grain", Quantity: 1}, {Commodity: "flour", Quantity: 2}},
				// ash is a byproduct here: non-discard, no cost share.
				Outputs:  []Output{{Commodity: "bread", Quantity: 3, CostShare: 10000}, {Commodity: "ash", Quantity: 1}},
				Requires: []Tool{{Commodity: "oven", WearPerUse: 7}}},
		},
	}
	cc := c.canonical()
	return &cc
}

// TestNewRecipeTable_RowsCarryTheRecipe verifies that a recipe with two
// inputs, a byproduct, a discard and a tool resolves every list to typed IDs
// in canonical commodity order with quantities, CostShare, Discard and
// WearPerUse carried.
//
// Requirement: FR-PRO-002
func TestNewRecipeTable_RowsCarryTheRecipe(t *testing.T) {
	t.Parallel()
	tab := newRecipeTable(bakeryConfig())
	bake := tab.row(0)
	wantIn := []itemRow{{commodity: 2, quantity: 2}, {commodity: 3, quantity: 1}} // flour, grain
	if !slices.Equal(bake.inputs, wantIn) {
		t.Errorf("bake inputs = %v, want %v", bake.inputs, wantIn)
	}
	wantOut := []outputRow{{commodity: 0, quantity: 1, costShare: 0}, {commodity: 1, quantity: 3, costShare: 10000}}
	if !slices.Equal(bake.outputs, wantOut) {
		t.Errorf("bake outputs = %v, want %v", bake.outputs, wantOut)
	}
	if want := []toolRow{{commodity: 4, wearPerUse: 7}}; !slices.Equal(bake.tools, want) {
		t.Errorf("bake tools = %v, want %v", bake.tools, want)
	}
	mill := tab.row(2)
	if want := []outputRow{{commodity: 0, quantity: 1, discard: true}, {commodity: 2, quantity: 2, costShare: 10000}}; !slices.Equal(mill.outputs, want) {
		t.Errorf("mill outputs = %v, want %v", mill.outputs, want)
	}
}

// TestNewRecipeTable_SourceFlag verifies that a recipe with no inputs is a
// source and one with inputs is not.
//
// Requirement: FR-PRO-003
func TestNewRecipeTable_SourceFlag(t *testing.T) {
	t.Parallel()
	tab := newRecipeTable(bakeryConfig())
	for id, want := range map[RecipeID]bool{0: false, 1: false, 2: false, 3: true} {
		if got := tab.row(id).source; got != want {
			t.Errorf("row(%d).source = %v, want %v", id, got, want)
		}
	}
}

// TestRecipeTable_Makers verifies the makers index: two makers list in
// ascending ID, a discard-only output has no maker, a byproduct has its recipe
// as a maker, and a commodity no recipe makes has none.
//
// Requirement: FR-PRO-032
func TestRecipeTable_Makers(t *testing.T) {
	t.Parallel()
	tab := newRecipeTable(bakeryConfig())
	tests := []struct {
		name string
		id   CommodityID
		want []RecipeID
	}{
		{"two makers ascending", 2, []RecipeID{1, 2}}, // flour: grind, mill
		// ash: discard of mill (2) is not a maker; byproduct of bake (0) is.
		{"byproduct counts, discard does not", 0, []RecipeID{0}},
		{"raw with a source recipe", 3, []RecipeID{3}},
		{"no recipe makes the tool", 4, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tab.makers(tt.id); !slices.Equal(got, tt.want) {
				t.Errorf("makers(%d) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
	// Only a discard output: no maker at all.
	c := Config{
		Commodities: []Commodity{{ID: "smoke"}, {ID: "wood"}},
		Recipes:     []Recipe{{ID: "burn", Inputs: []Item{{Commodity: "wood", Quantity: 1}}, Outputs: []Output{{Commodity: "smoke", Quantity: 1, Discard: true}}}},
	}
	cc := c.canonical()
	tab2 := newRecipeTable(&cc)
	if got := tab2.makers(0); len(got) != 0 {
		t.Errorf("makers(smoke) with a discard-only output = %v, want none", got)
	}
}

// TestNewRecipeTable_UnresolvedPanics verifies that a reference that does not
// resolve after Validate panics with the invariant message.
//
// Guard: the builder trusts Validate; a silent zero ID would name commodity 0
// and corrupt production without a sign.
func TestNewRecipeTable_UnresolvedPanics(t *testing.T) {
	t.Parallel()
	c := bakeryConfig()
	c.Recipes[0].Inputs[0].Commodity = "missing"
	defer func() {
		s, _ := recover().(string)
		if !strings.Contains(s, "invariant") {
			t.Errorf("newRecipeTable panic = %q, want an invariant message", s)
		}
	}()
	newRecipeTable(c)
}

// TestRecipe_HasNoFuncOrEmbeddedField verifies by reflection that Recipe and
// its row structs have no function field and no embedded field.
//
// Requirement: FR-PRO-001
//
// Guard: a recipe is data; a function field would let a recipe carry logic
// and an embedded field would break architecture.md 4.7 silently.
func TestRecipe_HasNoFuncOrEmbeddedField(t *testing.T) {
	t.Parallel()
	for _, typ := range []reflect.Type{
		reflect.TypeFor[Recipe](), reflect.TypeFor[Item](), reflect.TypeFor[Output](), reflect.TypeFor[Tool](),
	} {
		for i := range typ.NumField() {
			f := typ.Field(i)
			if f.Type.Kind() == reflect.Func || f.Anonymous {
				t.Errorf("%s.%s is a func or embedded field", typ.Name(), f.Name)
			}
		}
	}
}

// TestRecipeTable_ReadsAllocateNothing verifies that row and makers allocate
// nothing.
//
// Requirement: NFR-GDS-001
func TestRecipeTable_ReadsAllocateNothing(t *testing.T) {
	tab := newRecipeTable(bakeryConfig())
	var sink int
	allocs := testing.AllocsPerRun(100, func() {
		sink += len(tab.row(0).inputs) + len(tab.makers(2))
	})
	if allocs != 0 {
		t.Errorf("row and makers allocate %v per run, want 0 (sink %d)", allocs, sink)
	}
}
