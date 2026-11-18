package config_test

import (
	"reflect"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// infrastructureDoc is the file form of spec-infrastructure.md 12.1 and 4.1.1: a
// class of each kind, with `decay = 0` written, two initial rows (one keyed by a
// role, one a route toward a region) and a recipe with each clause. The numbers are
// placeholders.
const infrastructureDoc = minimal + `
[modules.infrastructure.class.mill]
kind = "site"
durability = 1000
decay = 0
wear_per_use = 2
capacity = 6
label = "a water mill"

[modules.infrastructure.class.road]
kind = "route"
durability = 2000
boost_bp = 6000
modes = ["foot", "coach"]
needs = "fair"

[modules.infrastructure.class.drain]
kind = "ground"
durability = 500
boost_bp = 3000
resource = "arable"

[modules.infrastructure.class.barn]
kind = "store"
durability = 800
store = "bin"

[[modules.infrastructure.initial]]
region = "north"
class = "mill"
role = "grower"
count = 3

[[modules.infrastructure.initial]]
region = "north"
class = "road"
toward = "south"
count = 1

[recipe.grind]
duration = 1
needs = ["mill"]
inputs = [ { commodity = "seed", quantity = 1 } ]
outputs = [ { commodity = "staple", quantity = 2 } ]

[recipe.build_mill]
duration = 3
inputs = [ { commodity = "fuel", quantity = 2 } ]
[recipe.build_mill.builds]
class = "road"
toward = "south"

[recipe.repair_mill]
duration = 1
inputs = [ { commodity = "fuel", quantity = 1 } ]
[recipe.repair_mill.repairs]
class = "mill"
units = 100
`

// TestLoad_Infrastructure verifies that the tables of spec-infrastructure.md 12.1 load
// to the Go value: the classes of the four kinds with their fields (`decay = 0` loads
// as 0), the initial rows in the order written, a key-less table switching the module
// on, and the recipe keys needs, builds and repairs, with a build or repair recipe
// that has no outputs; and that the file and the Go value hash alike.
//
// Requirement: FR-CAP-045, FR-CAP-010, FR-CFG-031
func TestLoad_Infrastructure(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(infrastructureDoc)
	if err != nil {
		t.Fatalf("Parse(infrastructureDoc) error = %v, want nil", err)
	}
	want := &economysim.Infrastructure{
		Classes: []economysim.FacilityClass{
			{ID: "barn", Kind: "store", Durability: 800, Store: "bin"},
			{ID: "drain", Kind: "ground", Durability: 500, BoostBP: 3000, Resource: "arable"},
			{ID: "mill", Kind: "site", Durability: 1000, WearPerUse: 2, Capacity: 6, Label: "a water mill"},
			{ID: "road", Kind: "route", Durability: 2000, BoostBP: 6000, Modes: []string{"foot", "coach"}, Needs: "fair"},
		},
		Initial: []economysim.InitialFacility{
			{Region: "north", Class: "mill", Role: "grower", Count: 3},
			{Region: "north", Class: "road", Toward: "south", Count: 1},
		},
	}
	if !reflect.DeepEqual(got.Modules.Infrastructure, want) {
		t.Errorf("Parse(infrastructureDoc).Modules.Infrastructure = %+v, want %+v", got.Modules.Infrastructure, want)
	}
	clauses := map[string]economysim.Recipe{}
	for _, r := range got.Recipes {
		clauses[r.ID] = r
	}
	if r := clauses["grind"]; !reflect.DeepEqual(r.Needs, []string{"mill"}) || len(r.Outputs) != 1 {
		t.Errorf("recipe grind = Needs %v, Outputs %v, want needs [mill] and one output", r.Needs, r.Outputs)
	}
	if r := clauses["build_mill"]; !reflect.DeepEqual(r.Builds, &economysim.FacilityBuild{Class: "road", Toward: "south"}) || len(r.Outputs) != 0 {
		t.Errorf("recipe build_mill = Builds %+v, Outputs %v, want the road clause and no outputs", r.Builds, r.Outputs)
	}
	if r := clauses["repair_mill"]; !reflect.DeepEqual(r.Repairs, &economysim.FacilityRepair{Class: "mill", Units: 100}) {
		t.Errorf("recipe repair_mill.Repairs = %+v, want the mill clause of 100 units", r.Repairs)
	}
	goCfg := got
	goCfg.Modules.Infrastructure = want
	if economysim.HashConfig(goCfg) != economysim.HashConfig(got) {
		t.Error("HashConfig of the file differs from that of the same Go value")
	}
}

// TestLoad_InfrastructureOnAndOff verifies that a key-less [modules.infrastructure]
// table switches the module on with nothing in it, that no table leaves it off, and
// that a layered document naming one class gives one class, the later document's.
//
// Requirement: FR-CAP-001, FR-CFG-031
func TestLoad_InfrastructureOnAndOff(t *testing.T) {
	t.Parallel()
	if on, err := parseStrings(minimal + "[modules.infrastructure]\n"); err != nil || !reflect.DeepEqual(on.Modules.Infrastructure, &economysim.Infrastructure{}) {
		t.Errorf("Parse(empty table).Modules.Infrastructure = %+v, %v, want the module on and empty", on.Modules.Infrastructure, err)
	}
	if off, _ := parseStrings(minimal); off.Modules.Infrastructure != nil {
		t.Errorf("Parse(minimal).Modules.Infrastructure = %+v, want nil: the module is off", off.Modules.Infrastructure)
	}
	const a = "[modules.infrastructure.class.a]\nkind = \"site\"\ndurability = 10\ncapacity = 1\n"
	got, err := parseStrings(minimal+a, "[modules.infrastructure.class.a]\nkind = \"site\"\ndurability = 20\ncapacity = 2\n")
	if err != nil {
		t.Fatalf("Parse(two layers) error = %v, want nil", err)
	}
	if c := got.Modules.Infrastructure.Classes; len(c) != 1 || c[0].Durability != 20 || c[0].Capacity != 2 {
		t.Errorf("Parse(two layers).Classes = %+v, want one class of the later document", c)
	}
}

// TestLoad_InfrastructureRejections verifies the keys and types the loader rejects in
// the infrastructure tables and the recipe clauses, each with the rule and the path of
// the key.
//
// Requirement: FR-CAP-045, FR-CFG-031
func TestLoad_InfrastructureRejections(t *testing.T) {
	t.Parallel()
	const class = "[modules.infrastructure.class.a]\nkind = \"site\"\ndurability = 10\ncapacity = 1\n"
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"an unknown key of the module", minimal + "[modules.infrastructure]\nlevel = 1\n", economysim.ErrUnknownKey, []string{"modules", "infrastructure", "level"}},
		{"an unknown key of a class", minimal + "[modules.infrastructure.class.a]\nkind = \"site\"\ndurability = 10\ncost = 1\n", economysim.ErrUnknownKey, []string{"modules", "infrastructure", "class", "a", "cost"}},
		{"a class with no kind", minimal + "[modules.infrastructure.class.a]\ndurability = 10\n", economysim.ErrMissingField, []string{"modules", "infrastructure", "class", "a", "kind"}},
		{"a class with no durability", minimal + "[modules.infrastructure.class.a]\nkind = \"site\"\n", economysim.ErrMissingField, []string{"modules", "infrastructure", "class", "a", "durability"}},
		{"a float durability", minimal + "[modules.infrastructure.class.a]\nkind = \"site\"\ndurability = 1.5\n", economysim.ErrNotInteger, []string{"modules", "infrastructure", "class", "a", "durability"}},
		{"a string capacity", minimal + "[modules.infrastructure.class.a]\nkind = \"site\"\ndurability = 10\ncapacity = \"1\"\n", economysim.ErrWrongType, []string{"modules", "infrastructure", "class", "a", "capacity"}},
		{"a row with no count", minimal + class + "[[modules.infrastructure.initial]]\nregion = \"r\"\nclass = \"a\"\n", economysim.ErrMissingField, []string{"modules", "infrastructure", "initial", "[0]", "count"}},
		{"a row with no region", minimal + class + "[[modules.infrastructure.initial]]\nclass = \"a\"\ncount = 1\n", economysim.ErrMissingField, []string{"modules", "infrastructure", "initial", "[0]", "region"}},
		{"an unknown key of a row", minimal + class + "[[modules.infrastructure.initial]]\nregion = \"r\"\nclass = \"a\"\ncount = 1\nowner = \"x\"\n", economysim.ErrUnknownKey, []string{"modules", "infrastructure", "initial", "[0]", "owner"}},
		{"a build with no class", minimal + "[recipe.a.builds]\ntoward = \"x\"\n", economysim.ErrMissingField, []string{"recipe", "a", "builds", "class"}},
		{"a repair with no units", minimal + "[recipe.a.repairs]\nclass = \"x\"\n", economysim.ErrMissingField, []string{"recipe", "a", "repairs", "units"}},
		{"an unknown key of a build", minimal + "[recipe.a.builds]\nclass = \"x\"\nunits = 1\n", economysim.ErrUnknownKey, []string{"recipe", "a", "builds", "units"}},
		{"a number in needs", minimal + "[recipe.a]\nneeds = [ 1 ]\n", economysim.ErrWrongType, []string{"recipe", "a", "needs", "[0]"}},
		{"a recipe with no outputs and no clause", minimal + "[recipe.a]\nduration = 1\n", economysim.ErrMissingField, []string{"recipe", "a", "outputs"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseStrings(tt.doc)
			if err == nil || !hasError(err, tt.rule, tt.path) {
				t.Errorf("Parse(%q) errors = %s, want %v at %q", tt.doc, describe(err), tt.rule, tt.path)
			}
		})
	}
}

// FuzzLoadInfrastructure verifies, for random class, row and clause values, that the
// loader never panics, that a rejected document returns an error, that an accepted one
// loads again to an equal Config with an equal hash, and that Validate and Warnings do
// not panic.
//
// Requirement: FR-CAP-045, FR-CFG-024
func FuzzLoadInfrastructure(f *testing.F) {
	f.Add("mill", "site", "north", "grower", int64(1000), int64(6), int64(3))
	f.Add("road", "route", "south", "", int64(2000), int64(6000), int64(1))
	f.Add("", "", "", "", int64(0), int64(-1), int64(-1))
	f.Add("x", "store", "y", "z", int64(1)<<62, int64(1)<<62, int64(1)<<62)
	f.Fuzz(func(t *testing.T, id, kind, region, role string, durability, capacity, count int64) {
		n := func(v int64) string { return strconv.FormatInt(v, 10) }
		doc := minimal + "[modules.infrastructure.class." + strconv.Quote(id) + "]\nkind = " + strconv.Quote(kind) +
			"\ndurability = " + n(durability) + "\ncapacity = " + n(capacity) + "\nboost_bp = " + n(capacity) +
			"\n[[modules.infrastructure.initial]]\nregion = " + strconv.Quote(region) + "\nclass = " + strconv.Quote(id) +
			"\nrole = " + strconv.Quote(role) + "\ncount = " + n(count) +
			"\n[recipe.r.repairs]\nclass = " + strconv.Quote(id) + "\nunits = " + n(count) + "\n"
		got, err := parseStrings(doc)
		if err != nil {
			return
		}
		again, err := parseStrings(doc)
		if err != nil || !reflect.DeepEqual(got, again) || economysim.HashConfig(got) != economysim.HashConfig(again) {
			t.Fatalf("Parse(%q) is not repeatable: %v", doc, err)
		}
		_ = got.Warnings()
		_ = got.Validate()
	})
}
