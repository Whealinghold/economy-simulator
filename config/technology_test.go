package config_test

import (
	"os"
	"reflect"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// loadTechnologyPack loads the layers of the chains directory and validates the
// result.
func loadTechnologyPack(t *testing.T, layers ...string) economysim.Config {
	t.Helper()
	cfg, err := config.LoadFS(os.DirFS(chains), layers...)
	if err != nil {
		t.Fatalf("LoadFS(%q) error = %v, want nil", layers, err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate(%q) error = %v, want nil", layers, err)
	}
	return cfg
}

// technologyDoc is the file form of spec-technology.md 9.1: two fields, one with
// learning, a trend and a label, one with discovery and diffusion.
const technologyDoc = minimal + `
[modules.technology.field.farming]
recipes = ["grow_staple", "gather_seed"]
label = "Farming"
cap_bp = 3000
base = 1000
doubling_bp = 300
trend_bp = 100

[modules.technology.field.crafts]
recipes = ["make_tool_a"]
cap_bp = 2000
discovery_bp = 5000
step_bp = 100
catch_up_bp = 2000
reach = 250
`

// technologySame is technologyDoc written the other way: the fields in the other
// order, the keys in another order and a field as a dotted table.
const technologySame = minimal + `
[modules.technology.field.crafts]
reach = 250
catch_up_bp = 2000
step_bp = 100
discovery_bp = 5000
cap_bp = 2000
recipes = ["make_tool_a"]
[modules.technology.field.farming]
trend_bp = 100
doubling_bp = 300
base = 1000
cap_bp = 3000
label = "Farming"
recipes = ["gather_seed", "grow_staple"]
`

// TestLoad_Technology verifies that the tables of spec-technology.md 9.1 load: the
// fields in ID order with every key, a key-less [modules.technology] table switching
// the module on with no field, no table leaving it off, two spellings of one module
// hashing alike, the loader keeping a list as written, and the file and the Go value hashing alike.
//
// Requirement: FR-TEC-002, FR-CFG-085
func TestLoad_Technology(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(technologyDoc)
	if err != nil {
		t.Fatalf("Parse(technologyDoc) error = %v, want nil", err)
	}
	want := &economysim.Technology{Fields: []economysim.TechField{
		{ID: "crafts", Recipes: []string{"make_tool_a"}, CapBP: 2000, DiscoveryBP: 5000, StepBP: 100, CatchUpBP: 2000, Reach: 250},
		{ID: "farming", Recipes: []string{"grow_staple", "gather_seed"}, CapBP: 3000, Base: 1000, DoublingBP: 300, TrendBP: 100, Label: "Farming"},
	}}
	if !reflect.DeepEqual(got.Modules.Technology, want) {
		t.Errorf("Parse(technologyDoc).Modules.Technology = %+v, want %+v", got.Modules.Technology, want)
	}
	if same, err := parseStrings(technologySame); err != nil || economysim.HashConfig(same) != economysim.HashConfig(got) {
		t.Errorf("Parse(technologySame) = %+v, %v, want the same hash as technologyDoc: the lists hash in canonical order", same.Modules.Technology, err)
	}
	goCfg := economysim.Config{World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar()}
	goCfg.Modules.Technology = want
	if economysim.HashConfig(goCfg) != economysim.HashConfig(got) {
		t.Error("HashConfig of the file differs from that of the same Go value")
	}
	if on, err := parseStrings(minimal + "[modules.technology]\n"); err != nil || !reflect.DeepEqual(on.Modules.Technology, &economysim.Technology{}) {
		t.Errorf("Parse(empty table).Modules.Technology = %+v, %v, want the module on and empty", on.Modules.Technology, err)
	}
	if off, _ := parseStrings(minimal); off.Modules.Technology != nil {
		t.Errorf("Parse(minimal).Modules.Technology = %+v, want nil: the module is off", off.Modules.Technology)
	}
}

// TestLoad_TechnologyRejections verifies the keys and types the loader rejects in the
// technology tables, each with the rule and the path of the key.
//
// Requirement: FR-CFG-085
func TestLoad_TechnologyRejections(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"an unknown key of the module", minimal + "[modules.technology]\nlevel = 1\n", economysim.ErrUnknownKey, []string{"modules", "technology", "level"}},
		{"an unknown key of a field", minimal + "[modules.technology.field.a]\ncost = 1\n", economysim.ErrUnknownKey, []string{"modules", "technology", "field", "a", "cost"}},
		{"a float cap", minimal + "[modules.technology.field.a]\ncap_bp = 1.5\n", economysim.ErrNotInteger, []string{"modules", "technology", "field", "a", "cap_bp"}},
		{"a string reach", minimal + "[modules.technology.field.a]\nreach = \"1\"\n", economysim.ErrWrongType, []string{"modules", "technology", "field", "a", "reach"}},
		{"a number in recipes", minimal + "[modules.technology.field.a]\nrecipes = [ 1 ]\n", economysim.ErrWrongType, []string{"modules", "technology", "field", "a", "recipes", "[0]"}},
		{"a scalar where a field table", minimal + "[modules.technology.field]\na = 1\n", economysim.ErrWrongType, []string{"modules", "technology", "field", "a"}},
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

// TestLoad_TechnologyOverBounds verifies that a document that loads and breaks a rule
// of spec-technology.md 10.1 is reported by Validate, not by the loader: the loader
// checks the shape of the file and the bounds are the validator's.
//
// Requirement: FR-TEC-002, FR-TEC-003
func TestLoad_TechnologyOverBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"a cap over MaxGainBP", "[modules.technology.field.a]\nrecipes = [\"grow_staple\"]\ncap_bp = 100001\n",
			economysim.ErrOutOfBounds, []string{"modules", "technology", "fields[0]", "cap_bp"}},
		{"a field with no recipe", "[modules.technology.field.a]\ncap_bp = 5\n",
			economysim.ErrTechShape, []string{"modules", "technology", "fields[0]", "recipes"}},
		{"a recipe in two fields", "[modules.technology.field.a]\nrecipes = [\"grow_staple\"]\ncap_bp = 5\n[modules.technology.field.b]\nrecipes = [\"grow_staple\"]\ncap_bp = 5\n",
			economysim.ErrTechRecipeOverlap, []string{"modules", "technology", "fields[1]", "recipes[0]"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := parseStrings(minimal, tt.doc)
			if err != nil {
				t.Fatalf("Parse() error = %v, want nil: the loader does not check bounds", err)
			}
			cfg, err := config.LoadFS(os.DirFS(chains), "base.toml", "three.toml", "warm-three.toml")
			if err != nil {
				t.Fatalf("LoadFS(pack three) error = %v, want nil", err)
			}
			cfg.Modules.Technology = doc.Modules.Technology
			if err := cfg.Validate(); err == nil || !hasError(err, tt.rule, tt.path) {
				t.Errorf("Validate() errors = %s, want %v at %q", describe(err), tt.rule, tt.path)
			}
		})
	}
}

// TestLoad_TechnologyOverlays verifies that the two overlays of plan 3-tec load on pack
// three, validate, and hold the fields they say: technology.toml one learning-only field
// of the growers' recipe, technology-four.toml all four mechanisms on that field, with
// the edges of trade-three.toml in contact at its reach.
//
// Requirement: FR-TEC-002, FR-CFG-085
func TestLoad_TechnologyOverlays(t *testing.T) {
	t.Parallel()
	pack := []string{"base.toml", "three.toml", "warm-three.toml"}
	learn := loadTechnologyPack(t, append(append([]string{}, pack...), "overlay/technology.toml")...)
	want := economysim.TechField{ID: "farming", Recipes: []string{"grow_staple"}, CapBP: 3000, Base: 1000, DoublingBP: 300, Label: "Farming"}
	if m := learn.Modules.Technology; m == nil || !reflect.DeepEqual(m.Fields, []economysim.TechField{want}) {
		t.Errorf("overlay/technology.toml loaded %+v, want the one field %+v", m, want)
	}
	four := loadTechnologyPack(t, append(append([]string{}, pack...), "overlay/trade-three.toml", "overlay/technology-four.toml")...)
	want.TrendBP, want.DiscoveryBP, want.StepBP, want.CatchUpBP, want.Reach = 100, 100_000, 100, 2000, 250
	if m := four.Modules.Technology; m == nil || !reflect.DeepEqual(m.Fields, []economysim.TechField{want}) {
		t.Errorf("overlay/technology-four.toml loaded %+v, want the one field %+v", m, want)
	}
	for _, w := range four.Warnings() {
		if len(w.Path) > 1 && w.Path[1] == "technology" {
			t.Errorf("technology-four on three with the trade edges raises %v, want none of the module's", w)
		}
	}
}

// FuzzLoadTechnology verifies, for random field tables, that the loader never
// panics, that a rejected document returns an error, that an accepted one loads
// again to an equal Config with an equal hash, that Validate does not panic, and that
// a field that validates has a recipe, a cap above 0 and no value an off mechanism
// does not use (spec-technology.md 3.1.2).
//
// Requirement: FR-TEC-002, FR-TEC-003, FR-CFG-024
func FuzzLoadTechnology(f *testing.F) {
	f.Add("farming", "grow_staple", int64(3000), int64(1000), int64(300), int64(100), int64(0), int64(0), int64(0), int64(0))
	f.Add("crafts", "make_tool_a", int64(2000), int64(0), int64(0), int64(0), int64(5000), int64(100), int64(2000), int64(250))
	f.Add("", "", int64(0), int64(-1), int64(-1), int64(-1), int64(-1), int64(-1), int64(-1), int64(-1))
	f.Add("x", "nowhere", int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62)
	f.Fuzz(func(t *testing.T, id, recipe string, capBP, base, doubling, trend, discovery, step, catchUp, reach int64) {
		doc := minimal + "[modules.technology.field." + strconv.Quote(id) + "]\nrecipes = [ " + strconv.Quote(recipe) + " ]\ncap_bp = " + strconv.FormatInt(capBP, 10) +
			"\nbase = " + strconv.FormatInt(base, 10) + "\ndoubling_bp = " + strconv.FormatInt(doubling, 10) + "\ntrend_bp = " + strconv.FormatInt(trend, 10) +
			"\ndiscovery_bp = " + strconv.FormatInt(discovery, 10) + "\nstep_bp = " + strconv.FormatInt(step, 10) +
			"\ncatch_up_bp = " + strconv.FormatInt(catchUp, 10) + "\nreach = " + strconv.FormatInt(reach, 10) + "\n"
		got, err := parseStrings(doc)
		if err != nil {
			return
		}
		again, err := parseStrings(doc)
		if err != nil || !reflect.DeepEqual(got, again) || economysim.HashConfig(got) != economysim.HashConfig(again) {
			t.Fatalf("Parse(%q) is not repeatable: %v", doc, err)
		}
		if got.Validate() != nil || got.Modules.Technology == nil {
			return
		}
		for _, fd := range got.Modules.Technology.Fields {
			if len(fd.Recipes) == 0 || fd.CapBP <= 0 || (fd.DoublingBP == 0) != (fd.Base == 0) ||
				(fd.DiscoveryBP == 0) != (fd.StepBP == 0) || (fd.CatchUpBP == 0) != (fd.Reach == 0) {
				t.Fatalf("Validate accepted the field %+v of %q, want a shape that 3.1.2 allows", fd, doc)
			}
		}
	})
}
