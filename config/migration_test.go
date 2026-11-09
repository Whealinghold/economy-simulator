package config_test

import (
	"errors"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// migrationDoc is a [modules.migration] table in the form of spec-migration.md
// 15.3, with two region rows and a role the module never moves.
const migrationDoc = minimal + `
[modules.migration]
move_speed = 20
move_review_ticks = 15
move_below = 9000
move_gain = 800
move_width = 4000
move_cooldown = 45
move_horizon = 120
move_out_share = 40
move_in_share = 80
[modules.migration.region.north]
out_share = 0
[modules.migration.region.south]
out_share = 10
in_share = 20
[role.broker]
fixed = true
[role.farmer]
`

// TestLoad_Migration verifies that the keys of spec-config.md 6.2.1 load the
// migration module: the table switches it on, its nine keys are read, a key that
// is missing keeps its default (and no MoveSpeed), a region row replaces the
// shares it names and keeps the world's for the key it omits, the rows come out
// in region order, the flag of a role is read and defaults to false, with no
// table the module is off, and a file and the same Go value hash alike.
//
// Requirement: FR-MIG-001, FR-MIG-004, FR-CFG-078
func TestLoad_Migration(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(migrationDoc)
	if err != nil {
		t.Fatalf("Parse(migrationDoc) error = %v, want nil", err)
	}
	want := &economysim.Migration{
		MoveSpeed: 20, MoveReviewTicks: 15, MoveBelow: 9_000, MoveGain: 800, MoveWidth: 4_000,
		MoveCooldown: 45, MoveHorizon: 120, MoveOutShare: 40, MoveInShare: 80,
		Regions: []economysim.RegionMove{{Region: "north", OutShare: 0, InShare: 80}, {Region: "south", OutShare: 10, InShare: 20}},
	}
	if !reflect.DeepEqual(got.Modules.Migration, want) {
		t.Errorf("Parse(migrationDoc).Modules.Migration = %+v, want %+v", got.Modules.Migration, want)
	}
	if len(got.Roles) != 2 || got.Roles[0].ID != "broker" || !got.Roles[0].Fixed || got.Roles[1].Fixed {
		t.Errorf("Parse(migrationDoc).Roles = %+v, want broker fixed and farmer not", got.Roles)
	}
	only, err := parseStrings(minimal + "[modules.migration]\nmove_speed = 7\n")
	if err != nil {
		t.Fatalf("Parse(move_speed only) error = %v, want nil", err)
	}
	if def := economysim.DefaultMigration(); only.Modules.Migration.MoveSpeed != 7 || only.Modules.Migration.MoveHorizon != def.MoveHorizon ||
		only.Modules.Migration.MoveOutShare != def.MoveOutShare {
		t.Errorf("Parse(move_speed only).Modules.Migration = %+v, want the defaults around MoveSpeed: %+v", *only.Modules.Migration, def)
	}
	if none, err := parseStrings(minimal + "[modules.migration]\n"); err != nil || none.Modules.Migration == nil || none.Modules.Migration.MoveSpeed != 0 {
		t.Errorf("Parse([modules.migration]) = %+v, %v, want the module on with no MoveSpeed (ErrOutOfBounds at Validate)", none.Modules.Migration, err)
	}
	if off, _ := parseStrings(minimal); off.Modules.Migration != nil {
		t.Errorf("Parse(minimal).Modules.Migration = %+v, want nil: the module is off", off.Modules.Migration)
	}
	role := func(id string, fixed bool) economysim.Role {
		r := economysim.DefaultRole()
		r.ID, r.Fixed = id, fixed
		return r
	}
	goValue := economysim.Config{
		World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar(),
		Roles: []economysim.Role{role("broker", true), role("farmer", false)},
	}
	goValue.Modules.Migration = want
	if economysim.HashConfig(got) != economysim.HashConfig(goValue) {
		t.Errorf("HashConfig(file) != HashConfig(the same Go value), want equal (FR-CFG-078)")
	}
}

// TestLoad_MigrationErrors verifies that an unknown key in the module table or in
// a region row is ErrUnknownKey, that a float is ErrNotInteger and that a flag
// that is not a boolean is ErrWrongType.
//
// Requirement: FR-MIG-001, FR-CFG-024
func TestLoad_MigrationErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		want error
	}{
		{"unknown key", "[modules.migration]\nbogus = 1\n", economysim.ErrUnknownKey},
		{"unknown key in a row", "[modules.migration.region.a]\nshare = 1\n", economysim.ErrUnknownKey},
		{"a float speed", "[modules.migration]\nmove_speed = 1.5\n", economysim.ErrNotInteger},
		{"a string share", "[modules.migration.region.a]\nout_share = \"5\"\n", economysim.ErrWrongType},
		{"fixed that is not a flag", "[role.a]\nfixed = 1\n", economysim.ErrWrongType},
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
