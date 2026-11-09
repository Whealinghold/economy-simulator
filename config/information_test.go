package config_test

import (
	"errors"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// informationDoc is a [modules.information] table in the form of
// spec-information.md 12.1, over a role flagged informed.
const informationDoc = minimal + `
[modules.information]
info_speed = 40
rumour_error_bp = 250
rumour_max_age = 30
[role.broker]
informed = true
[role.farmer]
`

// TestLoad_Information verifies that the keys of spec-config.md 6.2.1 load the
// information module: the table switches it on, its three keys are read, a key
// that is missing keeps its default (RumourErrorBP 500, RumourMaxAge 60, and no
// InfoSpeed), the flag of a role is read and defaults to false, with no table the
// module is off, and a file and the same Go value hash alike.
//
// Requirement: FR-INF-045, FR-CFG-074
func TestLoad_Information(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(informationDoc)
	if err != nil {
		t.Fatalf("Parse(informationDoc) error = %v, want nil", err)
	}
	want := &economysim.Information{InfoSpeed: 40, RumourErrorBP: 250, RumourMaxAge: 30}
	if !reflect.DeepEqual(got.Modules.Information, want) {
		t.Errorf("Parse(informationDoc).Modules.Information = %+v, want %+v", got.Modules.Information, want)
	}
	if len(got.Roles) != 2 || got.Roles[0].ID != "broker" || !got.Roles[0].Informed || got.Roles[1].Informed {
		t.Errorf("Parse(informationDoc).Roles = %+v, want broker informed and farmer not", got.Roles)
	}
	only, err := parseStrings(minimal + "[modules.information]\ninfo_speed = 7\n")
	if err != nil {
		t.Fatalf("Parse(info_speed only) error = %v, want nil", err)
	}
	if want := (economysim.Information{InfoSpeed: 7, RumourErrorBP: 500, RumourMaxAge: 60}); *only.Modules.Information != want {
		t.Errorf("Parse(info_speed only).Modules.Information = %+v, want the defaults around InfoSpeed: %+v", *only.Modules.Information, want)
	}
	none, err := parseStrings(minimal + "[modules.information]\n")
	if err != nil || none.Modules.Information == nil || none.Modules.Information.InfoSpeed != 0 {
		t.Errorf("Parse([modules.information]) = %+v, %v, want the module on with no InfoSpeed (ErrOutOfBounds at Validate)", none.Modules.Information, err)
	}
	if off, _ := parseStrings(minimal); off.Modules.Information != nil {
		t.Errorf("Parse(minimal).Modules.Information = %+v, want nil: the module is off", off.Modules.Information)
	}
	goValue := economysim.Config{
		World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar(),
		Roles: []economysim.Role{informedRole("broker", true), informedRole("farmer", false)},
	}
	goValue.Modules.Information = want
	if economysim.HashConfig(got) != economysim.HashConfig(goValue) {
		t.Errorf("HashConfig(file) != HashConfig(the same Go value), want equal (FR-INF-045)")
	}
}

// informedRole returns DefaultRole with the ID and the flag set.
func informedRole(id string, informed bool) economysim.Role {
	r := economysim.DefaultRole()
	r.ID, r.Informed = id, informed
	return r
}

// TestLoad_InformationErrors verifies that an unknown key in the module table is
// ErrUnknownKey at its path and that a value that is not an integer is
// ErrNotInteger.
//
// Requirement: FR-INF-045, FR-CFG-024
func TestLoad_InformationErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		want error
	}{
		{"unknown key", "[modules.information]\nbogus = 1\n", economysim.ErrUnknownKey},
		{"a float speed", "[modules.information]\ninfo_speed = 1.5\n", economysim.ErrNotInteger},
		{"a string error", "[modules.information]\nrumour_error_bp = \"5\"\n", economysim.ErrWrongType},
		{"informed that is not a flag", "[role.a]\ninformed = 1\n", economysim.ErrWrongType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseStrings(minimal + tt.doc); !errors.Is(err, tt.want) {
				t.Errorf("Parse(%q) error = %v, want %v", tt.doc, err, tt.want)
			}
		})
	}
}
