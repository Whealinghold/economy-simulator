package config_test

import (
	"errors"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// subsistenceDoc is a [modules.subsistence] table written in the form of
// spec-subsistence.md 3.2.2 and 16.1, with the default list and the rows in an
// order that is not the canonical one, one tunable written and one row with no
// key at all.
const subsistenceDoc = minimal + `
[modules.subsistence]
recipes = [ "glean", "forage" ]
exit_ticks = 5
[modules.subsistence.role.farmer]
recipes = []
[modules.subsistence.role.cutter]
recipes = [ "glean" ]
[modules.subsistence.role.carrier]
`

// TestLoad_SubsistenceTables verifies that the keys of spec-subsistence.md 3.2.2
// load the module: [modules.subsistence] switches it on over DefaultSubsistence
// (a key not written keeps its default), the role rows come out in role order,
// the lists as written, a row with no key is an empty row, and with no table the
// module is off.
//
// Requirement: FR-SUB-001, FR-SUB-002, FR-CFG-077
func TestLoad_SubsistenceTables(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(subsistenceDoc)
	if err != nil {
		t.Fatalf("Parse(subsistenceDoc) error = %v, want nil", err)
	}
	want := &economysim.SubsistenceConfig{
		Recipes:    []string{"glean", "forage"},
		Roles:      []economysim.RoleOwnUse{{Role: "carrier"}, {Role: "cutter", Recipes: []string{"glean"}}, {Role: "farmer"}},
		EnterTicks: 2, ExitTicks: 5, RetryJobs: 10,
	}
	if !reflect.DeepEqual(got.Modules.Subsistence, want) {
		t.Errorf("Parse(subsistenceDoc).Modules.Subsistence = %+v, want %+v", got.Modules.Subsistence, want)
	}
	on, _ := parseStrings(minimal + "[modules.subsistence]\n")
	if d := economysim.DefaultSubsistence(); on.Modules.Subsistence == nil || !reflect.DeepEqual(*on.Modules.Subsistence, d) {
		t.Errorf("Parse([modules.subsistence]).Modules.Subsistence = %+v, want the module on with %+v", on.Modules.Subsistence, d)
	}
	if off, _ := parseStrings(minimal); off.Modules.Subsistence != nil {
		t.Errorf("Parse(minimal).Modules.Subsistence = %+v, want nil: the module is off", off.Modules.Subsistence)
	}
	zero, _ := parseStrings(minimal + "[modules.subsistence]\nretry_jobs = 0\n")
	if zero.Modules.Subsistence.RetryJobs != 0 {
		t.Errorf("Parse(retry_jobs = 0) = %d, want 0: a 0 the file gives is kept", zero.Modules.Subsistence.RetryJobs)
	}
}

// TestLoad_SubsistenceLayersMergeByKey verifies that two documents merge by key
// (spec-config.md 6.4): the second sets one tunable and one role row, and the
// default list and the other keys of the first remain.
//
// Requirement: FR-SUB-002, FR-CFG-077
func TestLoad_SubsistenceLayersMergeByKey(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(subsistenceDoc, "[modules.subsistence]\nenter_ticks = 1\n[modules.subsistence.role.cutter]\nrecipes = [ \"forage\" ]\n")
	if err != nil {
		t.Fatalf("Parse(layers) error = %v, want nil", err)
	}
	m := got.Modules.Subsistence
	if m.EnterTicks != 1 || m.ExitTicks != 5 || !reflect.DeepEqual(m.Recipes, []string{"glean", "forage"}) {
		t.Errorf("merged module = %+v, want enter_ticks 1 from the second layer and the rest of the first", m)
	}
	if want := []string{"forage"}; len(m.Roles) != 3 || !reflect.DeepEqual(m.Roles[1].Recipes, want) {
		t.Errorf("merged rows = %+v, want the cutter's row set by the second layer to %v", m.Roles, want)
	}
}

// TestLoad_SubsistenceRejects verifies that a key that is not one of the
// module's, a value of the wrong type and a list that holds a non-string are
// rejected with the sentinel and the path of the key, and that a value out of its
// bound loads and is left to Validate.
//
// Requirement: FR-SUB-007, FR-CFG-008, FR-CFG-019
func TestLoad_SubsistenceRejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"unknown key in the module table", "[modules.subsistence]\nbogus = 1\n", economysim.ErrUnknownKey, []string{"modules", "subsistence", "bogus"}},
		{"unknown key in a row", "[modules.subsistence.role.r]\nbogus = 1\n", economysim.ErrUnknownKey, []string{"modules", "subsistence", "role", "r", "bogus"}},
		{"roles in the plural", "[modules.subsistence.roles.r]\n", economysim.ErrUnknownKey, []string{"modules", "subsistence", "roles"}},
		{"float enter ticks", "[modules.subsistence]\nenter_ticks = 1.5\n", economysim.ErrNotInteger, []string{"modules", "subsistence", "enter_ticks"}},
		{"recipes not an array", "[modules.subsistence]\nrecipes = \"glean\"\n", economysim.ErrWrongType, []string{"modules", "subsistence", "recipes"}},
		{"a number in the list", "[modules.subsistence.role.r]\nrecipes = [ 1 ]\n", economysim.ErrWrongType, []string{"modules", "subsistence", "role", "r", "recipes", "[0]"}},
		{"scalar where a row table", "[modules.subsistence.role]\nr = 1\n", economysim.ErrWrongType, []string{"modules", "subsistence", "role", "r"}},
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
	got, err := parseStrings(minimal + "[modules.subsistence]\nenter_ticks = 99999\n")
	if err != nil || got.Modules.Subsistence.EnterTicks != 99999 {
		t.Errorf("Parse(enter_ticks 99999) = %+v, %v, want the value loaded for Validate to reject", got.Modules.Subsistence, err)
	}
}
