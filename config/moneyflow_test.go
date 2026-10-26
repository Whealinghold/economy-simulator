package config_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// flowDoc is a document of the module with a rate grant on an inline target and
// a rate levy on a nested one, in the layout of spec-money-supply.md 17.6.
const flowDoc = minimal + `
[modules.money_supply]

[modules.money_supply.flow.stipend]
kind = "grant"
target = { kind = "households", role = "grower", region = "north" }
amount = 20
per = "tick"
label = "a stipend"

[modules.money_supply.flow.charge]
kind = "levy"
rate_bp = 3650
per = "year"
exempt = 3000
not_income = false
start = 2
until = 400
limit = 5000

[modules.money_supply.flow.charge.target]
kind = "households"
role = "cook"
`

// TestLoad_MoneySupply verifies that the table [modules.money_supply] switches
// the module on, that each flow is read from [flow.<id>] with an inline or a
// nested target, that the flows come out in ID order, and that the config the
// loader builds hashes as the same configuration written in Go.
//
// Requirement: FR-MON-005, NFR-MON-003
func TestLoad_MoneySupply(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(flowDoc)
	if err != nil {
		t.Fatalf("Parse = %v, want nil", err)
	}
	want := &economysim.MoneySupply{Flows: []economysim.MoneyFlow{
		{ID: "charge", Kind: "levy", Target: economysim.FlowTarget{Kind: "households", Role: "cook"},
			RateBP: 3_650, Per: "year", Exempt: 3_000, Start: 2, Until: 400, Limit: 5_000},
		{ID: "stipend", Kind: "grant", Target: economysim.FlowTarget{Kind: "households", Region: "north", Role: "grower"},
			Amount: 20, Per: "tick", Label: "a stipend"},
	}}
	if g := got.Modules.MoneySupply; g == nil || !slices.Equal(g.Flows, want.Flows) {
		t.Fatalf("Modules.MoneySupply = %+v, want %+v", g, want)
	}
	built, _ := parseStrings(minimal)
	built.Modules.MoneySupply = &economysim.MoneySupply{Flows: slices.Clone(want.Flows)}
	slices.Reverse(built.Modules.MoneySupply.Flows)
	built.Modules.MoneySupply.Flows[0].Per = ""
	if economysim.HashConfig(got) != economysim.HashConfig(built) {
		t.Errorf("HashConfig(loaded) differs from HashConfig(built in Go with the flows reversed and an empty Per)")
	}
}

// TestLoad_MoneySupplyPresence verifies that an empty table is a module that is
// on and no table is a module that is off.
//
// Requirement: FR-MON-005
func TestLoad_MoneySupplyPresence(t *testing.T) {
	t.Parallel()
	on, err := parseStrings(minimal + "[modules.money_supply]\n")
	if err != nil || on.Modules.MoneySupply == nil || len(on.Modules.MoneySupply.Flows) != 0 {
		t.Errorf("Parse(empty table) = %+v, %v, want the module on with no flow", on.Modules.MoneySupply, err)
	}
	off, err := parseStrings(minimal)
	if err != nil || off.Modules.MoneySupply != nil {
		t.Errorf("Parse(no table) = %+v, %v, want the module off", off.Modules.MoneySupply, err)
	}
}

// TestLoad_MoneySupplyKeys verifies the rejections of the table: an unknown key
// of the module, of a flow and of a target, a missing kind or target, a float
// where an integer, a string where a bool and a scalar where a flow table.
//
// Requirement: FR-MON-005
func TestLoad_MoneySupplyKeys(t *testing.T) {
	t.Parallel()
	const flow = "[modules.money_supply.flow.f]\n"
	const ok = "kind = \"grant\"\ntarget = { kind = \"households\", role = \"r\" }\n"
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"unknown key in the module", minimal + "[modules.money_supply]\nbogus = 1\n", economysim.ErrUnknownKey, []string{"modules", "money_supply", "bogus"}},
		{"unknown key in a flow", minimal + flow + ok + "bogus = 1\n", economysim.ErrUnknownKey, []string{"modules", "money_supply", "flow", "f", "bogus"}},
		{"unknown key in a target", minimal + flow + "kind = \"grant\"\ntarget = { kind = \"households\", store = \"s\" }\n", economysim.ErrUnknownKey, []string{"modules", "money_supply", "flow", "f", "target", "store"}},
		{"flows in the plural", minimal + "[modules.money_supply.flows.f]\n" + ok, economysim.ErrUnknownKey, []string{"modules", "money_supply", "flows"}},
		{"missing kind", minimal + flow + "target = { kind = \"households\" }\n", economysim.ErrMissingField, []string{"modules", "money_supply", "flow", "f", "kind"}},
		{"missing target", minimal + flow + "kind = \"grant\"\n", economysim.ErrMissingField, []string{"modules", "money_supply", "flow", "f", "target"}},
		{"float amount", minimal + flow + ok + "amount = 1.5\n", economysim.ErrNotInteger, []string{"modules", "money_supply", "flow", "f", "amount"}},
		{"string flag", minimal + flow + ok + "not_income = \"yes\"\n", economysim.ErrWrongType, []string{"modules", "money_supply", "flow", "f", "not_income"}},
		{"scalar where a flow table", minimal + "[modules.money_supply.flow]\nf = 1\n", economysim.ErrWrongType, []string{"modules", "money_supply", "flow", "f"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseStrings(tt.doc)
			if err == nil || !errors.Is(err, tt.rule) || !hasError(err, tt.rule, tt.path) {
				t.Errorf("Parse(%q) errors = %s, want %v at %q", tt.doc, describe(err), tt.rule, tt.path)
			}
		})
	}
}
