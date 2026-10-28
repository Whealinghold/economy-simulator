package config_test

import (
	"errors"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// resourcesDoc is a [modules.resources] table written in the form of
// spec-resources.md 3.3.2 and 16.1, with the resources and rows in an order that
// is not the canonical one, and a recipe that draws on two of them.
const resourcesDoc = minimal + `
[recipe.dig]
outputs = [ { commodity = "ore", quantity = 1 } ]
draws = [ { resource = "seam", quantity = 30 }, { resource = "arable", quantity = 1 } ]

[modules.resources]
horizon_years = 15
[modules.resources.resource.seam]
abundance = [ { share = 0, factor = 2000 }, { share = 10000, factor = 10000 } ]
capacity = 900
[modules.resources.resource.arable]
shape = "refill"
rate_bp = 10000
initial_bp = 8500
plan_share_bp = 5000
seasons = [ { season = "autumn", bp = 20000 }, { season = "winter", bp = 0 } ]
label = "Arable land"
[modules.resources.endowment.north.seam]
capacity = 3000
[modules.resources.endowment.north.arable]
initial = 0
regen_scale_bp = 5000
[modules.resources.endowment.centre.arable]
`

// TestLoad_ResourcesKeys verifies that the keys of spec-resources.md 3.3 load the
// module: [modules.resources] switches it on, resource.<id> and
// endowment.<region>.<resource> fill the two lists in canonical order over their
// constructors, a key a row writes sets its Has flag (also a value of 0), and
// the draws key of a recipe loads in the order written.
//
// Requirement: FR-RES-004, FR-RES-032, FR-CFG-073, FR-CFG-060
func TestLoad_ResourcesKeys(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(resourcesDoc)
	if err != nil {
		t.Fatalf("Parse(resourcesDoc) error = %v, want nil", err)
	}
	want := &economysim.Resources{
		HorizonYears: 15,
		Resources: []economysim.Resource{
			{ID: "arable", Shape: economysim.ShapeRefill, RateBP: 10_000, InitialBP: 8_500, PlanShareBP: 5_000,
				Seasons: []economysim.SeasonRegen{{Season: "autumn", BP: 20_000}, {Season: "winter", BP: 0}}, Label: "Arable land"},
			{ID: "seam", Shape: economysim.ShapeConstant, Capacity: 900, InitialBP: 10_000, PlanShareBP: 6_000,
				Abundance: []economysim.AbundanceKnot{{Share: 0, Factor: 2_000}, {Share: 10_000, Factor: 10_000}}},
		},
		Endowments: []economysim.Endowment{
			{Region: "centre", Resource: "arable"},
			{Region: "north", Resource: "arable", HasInitial: true, HasRegenScale: true, RegenScaleBP: 5_000},
			{Region: "north", Resource: "seam", HasCapacity: true, Capacity: 3_000},
		},
	}
	if !reflect.DeepEqual(got.Modules.Resources, want) {
		t.Errorf("Parse(resourcesDoc).Modules.Resources = %+v, want %+v", got.Modules.Resources, want)
	}
	draws := []economysim.Draw{{Resource: "seam", Quantity: 30}, {Resource: "arable", Quantity: 1}}
	if !reflect.DeepEqual(got.Recipes[0].Draws, draws) {
		t.Errorf("Parse(resourcesDoc) draws = %+v, want %+v in the order written", got.Recipes[0].Draws, draws)
	}
	if on, _ := parseStrings(minimal + "[modules.resources]\n"); on.Modules.Resources == nil || !reflect.DeepEqual(*on.Modules.Resources, economysim.DefaultResources()) {
		t.Errorf("Parse([modules.resources]).Modules.Resources = %+v, want the module on with its default horizon", on.Modules.Resources)
	}
	if off, _ := parseStrings(minimal); off.Modules.Resources != nil {
		t.Errorf("Parse(minimal).Modules.Resources = %+v, want nil: the module is off", off.Modules.Resources)
	}
}

// TestLoad_ResourcesRejects verifies that a key that is not one of the module's,
// a missing required key and a value of the wrong type are rejected with the
// sentinel and the path of the key, and that a bad shape loads and is left to
// Validate.
//
// Requirement: FR-RES-032, FR-CFG-008, FR-CFG-019, FR-CFG-098
func TestLoad_ResourcesRejects(t *testing.T) {
	t.Parallel()
	rec := minimal + "[recipe.dig]\noutputs = [ { commodity = \"ore\" } ]\n"
	tests := []struct {
		name string
		docs []string
		rule error
		path []string
	}{
		{"unknown key in the module table", []string{minimal + "[modules.resources]\nbogus = 1\n"}, economysim.ErrUnknownKey, []string{"modules", "resources", "bogus"}},
		{"unknown key in a resource", []string{minimal + "[modules.resources.resource.r]\nbogus = 1\n"}, economysim.ErrUnknownKey, []string{"modules", "resources", "resource", "r", "bogus"}},
		{"unknown key in a knot", []string{minimal + "[modules.resources.resource.r]\nabundance = [ { share = 0, factor = 1, bogus = 1 } ]\n"}, economysim.ErrUnknownKey, []string{"modules", "resources", "resource", "r", "abundance", "[0]", "bogus"}},
		{"unknown key in a row", []string{minimal + "[modules.resources.endowment.n.r]\nstock = 1\n"}, economysim.ErrUnknownKey, []string{"modules", "resources", "endowment", "n", "r", "stock"}},
		{"resources in the plural", []string{minimal + "[modules.resources.resources.r]\n"}, economysim.ErrUnknownKey, []string{"modules", "resources", "resources"}},
		{"missing knot factor", []string{minimal + "[modules.resources.resource.r]\nabundance = [ { share = 0 } ]\n"}, economysim.ErrMissingField, []string{"modules", "resources", "resource", "r", "abundance", "[0]", "factor"}},
		{"missing season bp", []string{minimal + "[modules.resources.resource.r]\nseasons = [ { season = \"a\" } ]\n"}, economysim.ErrMissingField, []string{"modules", "resources", "resource", "r", "seasons", "[0]", "bp"}},
		{"missing draw quantity", []string{rec + "draws = [ { resource = \"r\" } ]\n"}, economysim.ErrMissingField, []string{"recipe", "dig", "draws", "[0]", "quantity"}},
		{"missing draw resource", []string{rec + "draws = [ { quantity = 1 } ]\n"}, economysim.ErrMissingField, []string{"recipe", "dig", "draws", "[0]", "resource"}},
		{"float capacity", []string{minimal + "[modules.resources.resource.r]\ncapacity = 1.5\n"}, economysim.ErrNotInteger, []string{"modules", "resources", "resource", "r", "capacity"}},
		{"string shape", []string{minimal + "[modules.resources.resource.r]\nshape = 1\n"}, economysim.ErrWrongType, []string{"modules", "resources", "resource", "r", "shape"}},
		{"scalar where a row table", []string{minimal + "[modules.resources.endowment.n]\nr = 1\n"}, economysim.ErrWrongType, []string{"modules", "resources", "endowment", "n", "r"}},
		{"draws not an array", []string{rec + "draws = 1\n"}, economysim.ErrWrongType, []string{"recipe", "dig", "draws"}},
		{"table then scalar across documents", []string{minimal + "[modules.resources]\n", "[modules]\nresources = 1\n"}, economysim.ErrMergeConflict, []string{"modules", "resources"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseStrings(tt.docs...)
			if err == nil || !errors.Is(err, tt.rule) || !hasError(err, tt.rule, tt.path) {
				t.Errorf("Parse(%q) errors = %s, want %v at %q", tt.docs, describe(err), tt.rule, tt.path)
			}
		})
	}
	got, err := parseStrings(minimal + "[modules.resources.resource.r]\nshape = \"tidal\"\n")
	if err != nil || got.Modules.Resources.Resources[0].Shape != "tidal" {
		t.Errorf("Parse(shape tidal) = %v, %v, want the shape loaded for Validate to reject", got.Modules.Resources, err)
	}
}
