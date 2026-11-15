package config_test

import (
	"errors"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// settlementsDoc is the file form of spec-settlements.md 3.1: the band and the dwell
// (the keys here set neither, so the defaults apply), three classes with flags and
// limits, two region rows and a role with a gate. The class names are examples.
const settlementsDoc = minimal + `
[modules.settlements]
dwell = 45

[modules.settlements.class.hamlet]
enter = 0
[modules.settlements.class.village]
enter = 300
flags = ["fair"]
[modules.settlements.class.village.limits]
merchants = 2
berths = 0
[modules.settlements.class.town]
enter = 800
flags = ["hall", "fair"]
label = "a market town"
[modules.settlements.class.town.limits]
merchants = 12

[modules.settlements.region.west]
threshold_bp = 6000
[modules.settlements.region.north]
floor = "village"

[role.merchant]
needs = "fair"
ceiling = "merchants"
`

// settlementsSame is settlementsDoc written the other way: the tables in another
// order, the limits as inline tables and the band written out at its default.
const settlementsSame = minimal + `
[role.merchant]
ceiling = "merchants"
needs = "fair"
[modules.settlements.region.north]
floor = "village"
[modules.settlements.region.west]
threshold_bp = 6000
[modules.settlements.class.town]
label = "a market town"
flags = ["fair", "hall"]
limits = { merchants = 12 }
enter = 800
[modules.settlements.class.village]
limits = { berths = 0, merchants = 2 }
flags = ["fair"]
enter = 300
[modules.settlements.class.hamlet]
enter = 0
[modules.settlements]
band_bp = 1000
dwell = 45
`

// TestLoad_SettlementsToml verifies that the tables of spec-settlements.md 3.1 load:
// the band at its default and the dwell as written, the classes in ID order with their
// flags and limits (the limits in name order), the rows in region order with the
// default scale where the file omits it, the gate of a role, two spellings loading to
// one Config and one hash, and the file and the Go value hashing alike.
//
// Requirement: FR-SET-030, FR-CFG-084
func TestLoad_SettlementsToml(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(settlementsDoc)
	if err != nil {
		t.Fatalf("Parse(settlementsDoc) error = %v, want nil", err)
	}
	want := &economysim.Settlements{
		BandBP: 1_000, Dwell: 45,
		Classes: []economysim.SettlementClass{
			{ID: "hamlet", Enter: 0},
			{ID: "town", Enter: 800, Flags: []string{"hall", "fair"}, Limits: []economysim.ClassLimit{{Name: "merchants", Value: 12}}, Label: "a market town"},
			{ID: "village", Enter: 300, Flags: []string{"fair"}, Limits: []economysim.ClassLimit{{Name: "berths", Value: 0}, {Name: "merchants", Value: 2}}},
		},
		Regions: []economysim.RegionSettlement{
			{Region: "north", ThresholdBP: 10_000, Floor: "village"},
			{Region: "west", ThresholdBP: 6_000},
		},
	}
	if !reflect.DeepEqual(got.Modules.Settlements, want) {
		t.Errorf("Parse(settlementsDoc).Modules.Settlements = %+v, want %+v", got.Modules.Settlements, want)
	}
	if len(got.Roles) != 1 || got.Roles[0].Needs != "fair" || got.Roles[0].Ceiling != "merchants" {
		t.Errorf("Parse(settlementsDoc).Roles = %+v, want one role with needs fair and ceiling merchants", got.Roles)
	}
	same, err := parseStrings(settlementsSame)
	if err != nil || economysim.HashConfig(same) != economysim.HashConfig(got) {
		t.Errorf("Parse(settlementsSame) = %v, want the same hash as settlementsDoc (error %v)", economysim.HashConfig(same), err)
	}
	goCfg := got
	goCfg.Modules.Settlements = want
	if economysim.HashConfig(goCfg) != economysim.HashConfig(got) {
		t.Error("HashConfig of the file differs from that of the same Go value")
	}
}

// TestLoad_SettlementsOnAndOff verifies that a key-less [modules.settlements] table
// switches the module on with the default band and dwell and no class (the validator
// then reports no_class), and that no table leaves it off.
//
// Requirement: FR-SET-001, FR-SET-030
func TestLoad_SettlementsOnAndOff(t *testing.T) {
	t.Parallel()
	on, err := parseStrings(minimal + "[modules.settlements]\n")
	if err != nil || !reflect.DeepEqual(on.Modules.Settlements, &economysim.Settlements{BandBP: 1_000, Dwell: 30}) {
		t.Errorf("Parse(empty table).Modules.Settlements = %+v, %v, want the module on with the defaults", on.Modules.Settlements, err)
	}
	if off, _ := parseStrings(minimal); off.Modules.Settlements != nil {
		t.Errorf("Parse(minimal).Modules.Settlements = %+v, want nil: the module is off", off.Modules.Settlements)
	}
}

// TestLoad_SettlementsUnknownKey verifies the keys and types the loader rejects in
// the settlements tables, each with its rule and the path of the key.
//
// Requirement: FR-SET-030
func TestLoad_SettlementsUnknownKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"an unknown key of the module", minimal + "[modules.settlements]\nlevel = 1\n", economysim.ErrUnknownKey, []string{"modules", "settlements", "level"}},
		{"an unknown key of a class", minimal + "[modules.settlements.class.a]\nenter = 0\nrank = 1\n", economysim.ErrUnknownKey, []string{"modules", "settlements", "class", "a", "rank"}},
		{"an unknown key of a row", minimal + "[modules.settlements.region.a]\nscale = 1\n", economysim.ErrUnknownKey, []string{"modules", "settlements", "region", "a", "scale"}},
		{"a class with no enter", minimal + "[modules.settlements.class.a]\nflags = [\"x\"]\n", economysim.ErrMissingField, []string{"modules", "settlements", "class", "a", "enter"}},
		{"a float enter", minimal + "[modules.settlements.class.a]\nenter = 1.5\n", economysim.ErrNotInteger, []string{"modules", "settlements", "class", "a", "enter"}},
		{"a string dwell", minimal + "[modules.settlements]\ndwell = \"3\"\n", economysim.ErrWrongType, []string{"modules", "settlements", "dwell"}},
		{"a string limit value", minimal + "[modules.settlements.class.a]\nenter = 0\n[modules.settlements.class.a.limits]\nplaces = \"1\"\n", economysim.ErrWrongType, []string{"modules", "settlements", "class", "a", "limits", "places"}},
		{"a number in flags", minimal + "[modules.settlements.class.a]\nenter = 0\nflags = [ 1 ]\n", economysim.ErrWrongType, []string{"modules", "settlements", "class", "a", "flags", "[0]"}},
		{"a number as needs", minimal + "[role.a]\nneeds = 1\n", economysim.ErrWrongType, []string{"role", "a", "needs"}},
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

// TestLoad_SettlementsDuplicateClass verifies that a class written twice in one
// document is a syntax error, and that two layered documents naming one class give
// one class, the later document's, with the classes of both documents kept.
//
// Requirement: FR-SET-030
//
// Guard: the classes are a table keyed by ID, so a repeat can only be a typo in one
// file or an override in a layer; a loader that appended would give a list with a
// duplicate ID for the validator to report on a file that has none.
func TestLoad_SettlementsDuplicateClass(t *testing.T) {
	t.Parallel()
	const a = "[modules.settlements.class.a]\nenter = 0\n"
	if _, err := parseStrings(minimal + a + a); !errors.Is(err, economysim.ErrSyntax) {
		t.Errorf("Parse(class a twice) error = %v, want errors.Is ErrSyntax", err)
	}
	got, err := parseStrings(minimal+a, "[modules.settlements.class.a]\nenter = 0\nflags = [\"f\"]\n[modules.settlements.class.b]\nenter = 5\n")
	if err != nil {
		t.Fatalf("Parse(two layers) error = %v, want nil", err)
	}
	want := []economysim.SettlementClass{{ID: "a", Flags: []string{"f"}}, {ID: "b", Enter: 5}}
	if !reflect.DeepEqual(got.Modules.Settlements.Classes, want) {
		t.Errorf("Parse(two layers).Classes = %+v, want %+v", got.Modules.Settlements.Classes, want)
	}
}

// TestLoad_SettlementsOverBounds verifies that a document that loads and breaks a rule
// of spec-settlements.md 9.1 is reported by Validate with the rule and the path of the
// field: the loader checks the shape of the file, not the bounds.
//
// Requirement: FR-SET-003, FR-SET-030
func TestLoad_SettlementsOverBounds(t *testing.T) {
	t.Parallel()
	const first = "[modules.settlements.class.a]\nenter = 0\n"
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"a band over MaxBandBP", minimal + "[modules.settlements]\nband_bp = 5001\n" + first, economysim.ErrOutOfBounds, []string{"modules", "settlements", "band_bp"}},
		{"a first class that is not open", minimal + "[modules.settlements.class.a]\nenter = 3\n", economysim.ErrOutOfBounds, []string{"modules", "settlements", "classes[0]", "enter"}},
		{"a floor that is no class", minimal + first + "[modules.settlements.region.x]\nfloor = \"b\"\n", economysim.ErrUnknownRegion, []string{"modules", "settlements", "regions[0]", "region"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := parseStrings(tt.doc)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v, want nil", tt.doc, err)
			}
			if verr := cfg.Validate(); !hasError(verr, tt.rule, tt.path) {
				t.Errorf("Validate() errors = %s, want %v at %q", describe(verr), tt.rule, tt.path)
			}
		})
	}
}
