package config_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestLoad_SettlementsShapeErrors verifies the shapes of the settlements tables the
// loader rejects beyond TestLoad_SettlementsUnknownKey: a table where a value is
// expected and the reverse, a float where an integer is, a string where a list is,
// and the type of a row's keys, each with its rule and the path of the key.
//
// Requirement: FR-SET-030
func TestLoad_SettlementsShapeErrors(t *testing.T) {
	t.Parallel()
	const first = "[modules.settlements.class.a]\nenter = 0\n"
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"the module as a value", minimal + "[modules]\nsettlements = 3\n", economysim.ErrWrongType, []string{"modules", "settlements"}},
		{"the classes as a value", minimal + "[modules.settlements]\nclass = 3\n", economysim.ErrWrongType, []string{"modules", "settlements", "class"}},
		{"a class as a value", minimal + "[modules.settlements.class]\na = 3\n", economysim.ErrWrongType, []string{"modules", "settlements", "class", "a"}},
		{"a row as a value", minimal + first + "[modules.settlements.region]\nwest = 3\n", economysim.ErrWrongType, []string{"modules", "settlements", "region", "west"}},
		{"a float band", minimal + "[modules.settlements]\nband_bp = 1.5\n" + first, economysim.ErrNotInteger, []string{"modules", "settlements", "band_bp"}},
		{"a float dwell", minimal + "[modules.settlements]\ndwell = 2.5\n" + first, economysim.ErrNotInteger, []string{"modules", "settlements", "dwell"}},
		{"a string enter", minimal + "[modules.settlements.class.a]\nenter = \"0\"\n", economysim.ErrWrongType, []string{"modules", "settlements", "class", "a", "enter"}},
		{"flags as a string", minimal + first + "flags = \"fair\"\n", economysim.ErrWrongType, []string{"modules", "settlements", "class", "a", "flags"}},
		{"limits as a list", minimal + first + "limits = [ 1 ]\n", economysim.ErrWrongType, []string{"modules", "settlements", "class", "a", "limits"}},
		{"a float limit", minimal + first + "[modules.settlements.class.a.limits]\nplaces = 1.5\n", economysim.ErrNotInteger, []string{"modules", "settlements", "class", "a", "limits", "places"}},
		{"a float threshold", minimal + first + "[modules.settlements.region.west]\nthreshold_bp = 1.5\n", economysim.ErrNotInteger, []string{"modules", "settlements", "region", "west", "threshold_bp"}},
		{"a number as floor", minimal + first + "[modules.settlements.region.west]\nfloor = 1\n", economysim.ErrWrongType, []string{"modules", "settlements", "region", "west", "floor"}},
		{"a number as ceiling", minimal + "[role.a]\nceiling = 1\n", economysim.ErrWrongType, []string{"role", "a", "ceiling"}},
		{"a number as label", minimal + first + "label = 1\n", economysim.ErrWrongType, []string{"modules", "settlements", "class", "a", "label"}},
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

// TestLoad_SettlementsEmptyModuleLoadsThenFails verifies that a bare
// [modules.settlements] table loads with the module on and the defaults (a table with
// no key is a present module), and that Validate then reports the missing class: the
// loader checks shape, the validator checks content.
//
// Requirement: FR-SET-001, FR-SET-002, FR-SET-030
func TestLoad_SettlementsEmptyModuleLoadsThenFails(t *testing.T) {
	t.Parallel()
	cfg, err := parseStrings(minimal + "[modules.settlements]\n")
	if err != nil {
		t.Fatalf("Parse(bare module) error = %v, want nil", err)
	}
	m := cfg.Modules.Settlements
	if m == nil || m.BandBP != economysim.DefaultSettlements().BandBP || m.Dwell != economysim.DefaultSettlements().Dwell || len(m.Classes) != 0 {
		t.Fatalf("Parse(bare module).Modules.Settlements = %+v, want the module on with the defaults and no class", m)
	}
	if verr := cfg.Validate(); !hasError(verr, economysim.ErrTooMany, []string{"modules", "settlements", "classes"}) {
		t.Errorf("Validate() errors = %s, want ErrTooMany at modules.settlements.classes", describe(verr))
	}
}
