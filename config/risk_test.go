package config_test

import (
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// riskDoc is the file form of spec-risk.md 12.1 and spec-config.md 6.2: a unit route
// row, a common route row, a unit store row and a common store row. The rows are
// written out of canonical order.
const riskDoc = minimal + `
[modules.risk.route.south.north.foot]
strike_bp = 1000
severity_bp = 6000
common = true

[modules.risk.route.north.south.foot]
strike_bp = 3000
severity_bp = 2500

[modules.risk.store.north.shed]
strike_bp = 200
severity_bp = 10000
common = true

[modules.risk.store.north.bin]
strike_bp = 500
severity_bp = 4000
`

// riskSame is riskDoc with the tables in another order.
const riskSame = minimal + `
[modules.risk.store.north.bin]
severity_bp = 4000
strike_bp = 500
[modules.risk.route.north.south.foot]
severity_bp = 2500
strike_bp = 3000
[modules.risk.store.north.shed]
common = true
severity_bp = 10000
strike_bp = 200
[modules.risk.route.south.north.foot]
common = true
severity_bp = 6000
strike_bp = 1000
`

// TestLoad_Risk verifies that the tables of spec-risk.md 12.1 load: a route row keyed
// by (from, to, mode) and a store row keyed by (region, class), common false when the
// key is absent, a key-less [modules.risk] table switching the module on with no row,
// no table leaving it off, two spellings of one module hashing alike, and the file and
// the Go value hashing alike.
//
// Requirement: FR-RSK-001, FR-CFG-089
func TestLoad_Risk(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(riskDoc)
	if err != nil {
		t.Fatalf("Parse(riskDoc) error = %v, want nil", err)
	}
	// The loader reads each level in the order of the keys: from, then to, then mode.
	want := &economysim.Risk{
		Routes: []economysim.RouteRisk{
			{From: "north", To: "south", Mode: "foot", StrikeBP: 3_000, SeverityBP: 2_500},
			{From: "south", To: "north", Mode: "foot", StrikeBP: 1_000, SeverityBP: 6_000, Common: true},
		},
		Stores: []economysim.StoreRisk{
			{Region: "north", Class: "bin", StrikeBP: 500, SeverityBP: 4_000},
			{Region: "north", Class: "shed", StrikeBP: 200, SeverityBP: 10_000, Common: true},
		},
	}
	if !reflect.DeepEqual(got.Modules.Risk, want) {
		t.Errorf("Parse(riskDoc).Modules.Risk = %+v, want %+v", got.Modules.Risk, want)
	}
	if same, err := parseStrings(riskSame); err != nil || economysim.HashConfig(same) != economysim.HashConfig(got) {
		t.Errorf("Parse(riskSame) = %+v, %v, want the same hash as riskDoc", same.Modules.Risk, err)
	}
	goCfg := economysim.Config{World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar()}
	goCfg.Modules.Risk = want
	if economysim.HashConfig(goCfg) != economysim.HashConfig(got) {
		t.Error("HashConfig of the file differs from that of the same Go value")
	}
	if on, err := parseStrings(minimal + "[modules.risk]\n"); err != nil || !reflect.DeepEqual(on.Modules.Risk, &economysim.Risk{}) {
		t.Errorf("Parse(empty table).Modules.Risk = %+v, %v, want the module on and empty", on.Modules.Risk, err)
	}
	if off, _ := parseStrings(minimal); off.Modules.Risk != nil {
		t.Errorf("Parse(minimal).Modules.Risk = %+v, want nil: the module is off", off.Modules.Risk)
	}
}

// TestLoad_RiskRejections verifies the keys and types the loader rejects in the risk
// tables, each with the rule and the path of the key.
//
// Requirement: FR-CFG-089
func TestLoad_RiskRejections(t *testing.T) {
	t.Parallel()
	base := []string{"modules", "risk"}
	at := func(more ...string) []string { return append(slices.Clone(base), more...) }
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"an unknown key of the module", minimal + "[modules.risk]\nlevel = 1\n", economysim.ErrUnknownKey, at("level")},
		{"an unknown key of a route row", minimal + "[modules.risk.route.a.b.m]\nstrike_bp = 1\nseverity_bp = 1\nscale = 2\n", economysim.ErrUnknownKey, at("route", "a", "b", "m", "scale")},
		{"an unknown key of a store row", minimal + "[modules.risk.store.a.k]\nstrike_bp = 1\nseverity_bp = 1\nlabel = \"x\"\n", economysim.ErrUnknownKey, at("store", "a", "k", "label")},
		{"a route row with no strike", minimal + "[modules.risk.route.a.b.m]\nseverity_bp = 1\n", economysim.ErrMissingField, at("route", "a", "b", "m", "strike_bp")},
		{"a route row with no severity", minimal + "[modules.risk.route.a.b.m]\nstrike_bp = 1\n", economysim.ErrMissingField, at("route", "a", "b", "m", "severity_bp")},
		{"a store row with no strike", minimal + "[modules.risk.store.a.k]\nseverity_bp = 1\n", economysim.ErrMissingField, at("store", "a", "k", "strike_bp")},
		{"a store row with no severity", minimal + "[modules.risk.store.a.k]\nstrike_bp = 1\n", economysim.ErrMissingField, at("store", "a", "k", "severity_bp")},
		{"a float strike", minimal + "[modules.risk.route.a.b.m]\nstrike_bp = 1.5\nseverity_bp = 1\n", economysim.ErrNotInteger, at("route", "a", "b", "m", "strike_bp")},
		{"a string severity", minimal + "[modules.risk.store.a.k]\nstrike_bp = 1\nseverity_bp = \"1\"\n", economysim.ErrWrongType, at("store", "a", "k", "severity_bp")},
		{"an integer common", minimal + "[modules.risk.route.a.b.m]\nstrike_bp = 1\nseverity_bp = 1\ncommon = 1\n", economysim.ErrWrongType, at("route", "a", "b", "m", "common")},
		{"a scalar where a route table", minimal + "[modules.risk.route]\na = 1\n", economysim.ErrWrongType, at("route", "a")},
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
