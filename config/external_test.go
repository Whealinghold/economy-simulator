package config_test

import (
	"errors"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// externalDoc is a [modules.external] table in the form of spec-external-sector.md
// 3.3.1 and 18.2: one node with a key-less gate row, two lines (an import with a
// window, an export with a season table) and a second, key-less node.
const externalDoc = minimal + `
[modules.external.node.coast]
label = "the coast"
[modules.external.node.coast.gate.north]
scale_bp = 11000
[modules.external.node.coast.line.salt]
base = 300
trend_bp = -200
walk_step_bp = 50
walk_revert_bp = 500
walk_bound_bp = 400
price_by_season = [ { season = "summer", bp = 12000 } ]
[modules.external.node.coast.line.salt.sell]
margin_bp = 800
capacity = 12
total_units = 1500
total_value = 400000
window = [ { season = "winter", bp = 0 } ]
[modules.external.node.coast.line.grain]
base = 40
floor_bp = 2000
ceiling_bp = 50000
[modules.external.node.coast.line.grain.buy]
margin_bp = 300
capacity = 20
[modules.external.node.ring]
`

// TestLoad_External verifies that the tables of spec-config.md 6.2.3 load the
// external module: a node table switches the module on and marks the node, the
// line is read over DefaultOutsideLine (the clamps 1 000 and 100 000 when no key
// is written), a gate row defaults its scale, a side is present when written
// and its keys are read, with no table the module is off, and a file and the
// same Go value hash alike.
//
// Requirement: FR-EXT-043, FR-CFG-075
func TestLoad_External(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(externalDoc)
	if err != nil {
		t.Fatalf("Parse(externalDoc) error = %v, want nil", err)
	}
	salt := economysim.DefaultOutsideLine("salt", 300)
	salt.TrendBP, salt.WalkStepBP, salt.WalkRevertBP, salt.WalkBoundBP = -200, 50, 500, 400
	salt.PriceBySeason = []economysim.SeasonFactor{{Season: "summer", BP: 12_000}}
	salt.Sell = economysim.OutsideSide{MarginBP: 800, Capacity: 12, TotalUnits: 1_500, TotalValue: 400_000,
		Window: []economysim.SeasonFactor{{Season: "winter", BP: 0}}}
	grain := economysim.DefaultOutsideLine("grain", 40)
	grain.FloorBP, grain.CeilingBP = 2_000, 50_000
	grain.Buy = economysim.OutsideSide{MarginBP: 300, Capacity: 20}
	want := &economysim.External{Nodes: []economysim.OutsideNode{
		{Region: "coast", Label: "the coast", Gates: []economysim.OutsideGate{{Region: "north", ScaleBP: 11_000}},
			Lines: []economysim.OutsideLine{grain, salt}},
		{Region: "ring"},
	}}
	if !reflect.DeepEqual(got.Modules.External, want) {
		t.Errorf("Parse(externalDoc).Modules.External = %+v, want %+v", got.Modules.External, want)
	}
	if off, _ := parseStrings(minimal); off.Modules.External != nil {
		t.Errorf("Parse(minimal).Modules.External = %+v, want nil: the module is off", off.Modules.External)
	}
	bare, err := parseStrings(minimal + "[modules.external]\n")
	if err != nil || bare.Modules.External == nil || len(bare.Modules.External.Nodes) != 0 {
		t.Errorf("Parse([modules.external]) = %+v, %v, want an empty module that is on", bare.Modules.External, err)
	}
	// The same value built in Go, rows in another order, hashes alike: the order
	// of a file is not part of the configuration.
	goValue := got
	goValue.Modules.External = &economysim.External{Nodes: []economysim.OutsideNode{
		{Region: "ring"},
		{Region: "coast", Label: "other label", Gates: want.Nodes[0].Gates, Lines: []economysim.OutsideLine{salt, grain}},
	}}
	if economysim.HashConfig(got) != economysim.HashConfig(goValue) {
		t.Errorf("HashConfig(file) != HashConfig(the same Go value in another row order), want equal (FR-EXT-043)")
	}
}

// TestLoad_ExternalErrors verifies that an unknown key in any table of the
// module is ErrUnknownKey, that a float is ErrNotInteger and that a window that
// is not an array of rows is ErrWrongType.
//
// Requirement: FR-EXT-043, FR-CFG-024
func TestLoad_ExternalErrors(t *testing.T) {
	t.Parallel()
	const line = "[modules.external.node.n.line.c]\n"
	tests := []struct {
		name string
		doc  string
		want error
	}{
		{"unknown key in the module", "[modules.external]\nbogus = 1\n", economysim.ErrUnknownKey},
		{"unknown key in a node", "[modules.external.node.n]\nbogus = 1\n", economysim.ErrUnknownKey},
		{"unknown key in a line", line + "bogus = 1\n", economysim.ErrUnknownKey},
		{"unknown key in a side", line + "[modules.external.node.n.line.c.sell]\nbogus = 1\n", economysim.ErrUnknownKey},
		{"unknown key in a gate", "[modules.external.node.n.gate.g]\nbogus = 1\n", economysim.ErrUnknownKey},
		{"a float base", line + "base = 1.5\n", economysim.ErrNotInteger},
		{"a float capacity", line + "[modules.external.node.n.line.c.buy]\ncapacity = 2.5\n", economysim.ErrNotInteger},
		{"a window that is a number", line + "[modules.external.node.n.line.c.buy]\nwindow = 3\n", economysim.ErrWrongType},
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

// FuzzLoadExternal is FuzzLoad with the seeds of the external module: the
// document above and documents that break each table of the module. It asserts
// the same properties.
//
// Requirement: FR-EXT-043, NFR-EXT-003, FR-CFG-024
func FuzzLoadExternal(f *testing.F) {
	f.Add([]byte(externalDoc), []byte("[modules.external.node.n]\nbogus = 1\n"), []byte("[modules.external.node.n.line.c.buy]\nwindow = 3\n"))
	f.Add([]byte(minimal), []byte("[modules.external.node.n.line.c]\nbase = 1.5\n"), []byte("[modules.external.node.n.gate.g]\nscale_bp = -1\n"))
	f.Fuzz(loadProperties)
}
