package config_test

import (
	"errors"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// populationDoc is a [modules.population] table in the form of
// spec-population.md 8: both schedules written, one tunable, and two region rows
// in an order that is not the canonical one.
const populationDoc = minimal + `
[modules.population]
birth_schedule = [ { ratio = 6000, rate = 300 }, { ratio = 18000, rate = 700 } ]
death_schedule = [ { ratio = 6000, rate = 900 }, { ratio = 18000, rate = 100 } ]
birth_stake = 5000
[modules.population.region.b]
death_scale = 30000
[modules.population.region.a]
household_cap = 600
`

// TestLoad_Population verifies that the keys of spec-population.md 8 load the
// module: [modules.population] switches it on over DefaultPopulation (a key not
// written keeps its default), the schedules come out as written, the rows in
// region order over DefaultRegionPopulation, and with no table the module is off.
//
// Requirement: FR-POP-001, FR-POP-025
func TestLoad_Population(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(populationDoc)
	if err != nil {
		t.Fatalf("Parse(populationDoc) error = %v, want nil", err)
	}
	want := &economysim.Population{
		BirthSchedule: []economysim.Knot{{Ratio: 6_000, Rate: 300}, {Ratio: 18_000, Rate: 700}},
		DeathSchedule: []economysim.Knot{{Ratio: 6_000, Rate: 900}, {Ratio: 18_000, Rate: 100}},
		MinHouseholds: 1, BirthStake: 5_000,
		Regions: []economysim.RegionPopulation{
			{Region: "a", BirthScale: 10_000, DeathScale: 10_000, HouseholdCap: 600},
			{Region: "b", BirthScale: 10_000, DeathScale: 30_000},
		},
	}
	if !reflect.DeepEqual(got.Modules.Population, want) {
		t.Errorf("Parse(populationDoc).Modules.Population = %+v, want %+v", got.Modules.Population, want)
	}
	on, _ := parseStrings(minimal + "[modules.population]\n")
	if d := economysim.DefaultPopulation(); on.Modules.Population == nil || !reflect.DeepEqual(*on.Modules.Population, d) {
		t.Errorf("Parse([modules.population]).Modules.Population = %+v, want the module on with %+v", on.Modules.Population, d)
	}
	if off, _ := parseStrings(minimal); off.Modules.Population != nil {
		t.Errorf("Parse(minimal).Modules.Population = %+v, want nil: the module is off", off.Modules.Population)
	}
	zero, _ := parseStrings(minimal + "[modules.population]\nmin_households = 0\nbirth_stake = 0\n")
	if m := zero.Modules.Population; m.MinHouseholds != 0 || m.BirthStake != 0 {
		t.Errorf("Parse(min_households = 0, birth_stake = 0) = %+v, want the zeros the file gives kept", m)
	}
}

// TestLoad_PopulationLayers verifies that two documents merge by key
// (spec-config.md 6.4): a schedule of the second layer replaces the first
// whole, a region row merges by region, and the keys the second does not write
// remain.
//
// Requirement: FR-POP-025, FR-CFG-077
func TestLoad_PopulationLayers(t *testing.T) {
	t.Parallel()
	second := "[modules.population]\ndeath_schedule = [ { ratio = 0, rate = 10 } ]\n[modules.population.region.a]\nbirth_scale = 20000\n"
	got, err := parseStrings(populationDoc, second)
	if err != nil {
		t.Fatalf("Parse(layers) error = %v, want nil", err)
	}
	m := got.Modules.Population
	if want := []economysim.Knot{{Ratio: 0, Rate: 10}}; !reflect.DeepEqual(m.DeathSchedule, want) {
		t.Errorf("merged death schedule = %v, want %v: replaced whole, not merged knot by knot", m.DeathSchedule, want)
	}
	if want := []economysim.Knot{{Ratio: 6_000, Rate: 300}, {Ratio: 18_000, Rate: 700}}; !reflect.DeepEqual(m.BirthSchedule, want) || m.BirthStake != 5_000 {
		t.Errorf("merged module = %+v, want the first layer's birth schedule and stake", m)
	}
	if want := (economysim.RegionPopulation{Region: "a", BirthScale: 20_000, DeathScale: 10_000, HouseholdCap: 600}); m.Regions[0] != want {
		t.Errorf("merged row a = %+v, want %+v: the second layer's scale on the first layer's cap", m.Regions[0], want)
	}
}

// TestLoad_PopulationRejects verifies that an unknown key, a knot with a missing
// key, a float and a scalar where a table belongs are rejected with the sentinel
// and the path of the key, and that a value out of its bound loads and is left
// to Validate.
//
// Requirement: FR-CFG-008, FR-CFG-019, FR-POP-025
func TestLoad_PopulationRejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"unknown key in the module table", "[modules.population]\nbogus = 1\n", economysim.ErrUnknownKey, []string{"modules", "population", "bogus"}},
		{"unknown key in a row", "[modules.population.region.a]\nbogus = 1\n", economysim.ErrUnknownKey, []string{"modules", "population", "region", "a", "bogus"}},
		{"unknown key in a knot", "[modules.population]\nbirth_schedule = [ { ratio = 1, rate = 1, bogus = 2 } ]\n", economysim.ErrUnknownKey, []string{"modules", "population", "birth_schedule", "[0]", "bogus"}},
		{"a knot with no rate", "[modules.population]\ndeath_schedule = [ { ratio = 1 } ]\n", economysim.ErrMissingField, []string{"modules", "population", "death_schedule", "[0]", "rate"}},
		{"a knot with no ratio", "[modules.population]\ndeath_schedule = [ { rate = 1 } ]\n", economysim.ErrMissingField, []string{"modules", "population", "death_schedule", "[0]", "ratio"}},
		{"float stake", "[modules.population]\nbirth_stake = 1.5\n", economysim.ErrNotInteger, []string{"modules", "population", "birth_stake"}},
		{"schedule not an array", "[modules.population]\nbirth_schedule = 5\n", economysim.ErrWrongType, []string{"modules", "population", "birth_schedule"}},
		{"scalar where a row table", "[modules.population.region]\na = 1\n", economysim.ErrWrongType, []string{"modules", "population", "region", "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseStrings(minimal + tt.doc)
			if err == nil || !errors.Is(err, tt.rule) || !hasError(err, tt.rule, tt.path) {
				t.Errorf("Parse(%q) errors = %s, want %v at %q", tt.doc, describe(err), tt.rule, tt.path)
			}
		})
	}
	got, err := parseStrings(minimal + "[modules.population]\nbirth_stake = 99999\n")
	if err != nil || got.Modules.Population.BirthStake != 99999 {
		t.Errorf("Parse(birth_stake 99999) = %+v, %v, want the value loaded for Validate to reject", got.Modules.Population, err)
	}
}

// TestFileAndGo_Population verifies that the file and the Go structs give one
// configuration and one hash: the document loads to the Config a caller builds
// in Go (spec-config.md 3.7, FR-CFG-020).
//
// Requirement: FR-CFG-020, FR-CFG-069
func TestFileAndGo_Population(t *testing.T) {
	t.Parallel()
	fromFile, err := parseStrings(populationDoc)
	if err != nil {
		t.Fatalf("Parse(populationDoc) error = %v, want nil", err)
	}
	inGo, err := parseStrings(minimal)
	if err != nil {
		t.Fatalf("Parse(minimal) error = %v, want nil", err)
	}
	inGo.Modules.Population = &economysim.Population{
		BirthSchedule: []economysim.Knot{{Ratio: 6_000, Rate: 300}, {Ratio: 18_000, Rate: 700}},
		DeathSchedule: []economysim.Knot{{Ratio: 6_000, Rate: 900}, {Ratio: 18_000, Rate: 100}},
		MinHouseholds: 1, BirthStake: 5_000,
		Regions: []economysim.RegionPopulation{
			{Region: "b", BirthScale: 10_000, DeathScale: 30_000},
			{Region: "a", BirthScale: 10_000, DeathScale: 10_000, HouseholdCap: 600},
		},
	}
	if a, b := economysim.HashConfig(fromFile), economysim.HashConfig(inGo); a != b {
		t.Errorf("HashConfig(file) = %x, HashConfig(Go) = %x, want equal (rows in either order)", a, b)
	}
}
