package config_test

import (
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// policyDoc is the file form of spec-policy.md 3.1.3 and 14: the crown council with a
// schedule, the coast council with an Against lever and the inland council with a rule.
// The tables are written out of canonical order.
const policyDoc = minimal + `
[modules.policy.actor.inland_council]
jurisdiction  = "inland"
policy        = "rule"
every         = 30
reads_regions = true
label         = "Inland"

[modules.policy.actor.inland_council.lever.stipend]
kind     = "flow"
flow     = "bread_stipend"
min      = 0
max      = 20000
max_step = 1000

[modules.policy.actor.inland_council.rule.stipend]
gauge    = "unfed_share"
target   = 500
deadband = 100
span     = 1000
move     = 2000
raise    = true

[modules.policy.actor.crown_council]
jurisdiction = "crown"
policy       = "schedule"
every        = 10
cycle        = 250

[modules.policy.actor.crown_council.lever.poll]
kind     = "flow"
flow     = "head_tax"
min      = 0
max      = 20000
max_step = 5000

[modules.policy.actor.crown_council.schedule.poll.120]
bp = 15000

[modules.policy.actor.crown_council.schedule.poll.0]
bp = 10000

[modules.policy.actor.coast_council]
jurisdiction = "coast"
policy       = "directed"
every        = 10

[modules.policy.actor.coast_council.lever.gate]
kind      = "edge"
against   = "inland"
min       = 10000
max       = 40000
may_close = true

[modules.policy.actor.coast_council.lever.road]
kind = "transit"
from = "capital"
to   = "mill"
mode = "foot"
min  = 0
max  = 20000
`

// policyWant is the module that policyDoc loads, in the order the loader reads the keys.
func policyWant() *economysim.PolicyConfig {
	return &economysim.PolicyConfig{Actors: []economysim.PolicyActor{
		{
			ID: "coast_council", Jurisdiction: "coast", Policy: "directed", Every: 10,
			Levers: []economysim.Lever{
				{ID: "gate", Kind: economysim.LeverEdge, Against: "inland", Min: 10_000, Max: 40_000, MayClose: true},
				{ID: "road", Kind: economysim.LeverTransit, From: "capital", To: "mill", Mode: "foot", Min: 0, Max: 20_000},
			},
		},
		{
			ID: "crown_council", Jurisdiction: "crown", Policy: "schedule", Every: 10, Cycle: 250,
			Levers:   []economysim.Lever{{ID: "poll", Kind: economysim.LeverFlow, Flow: "head_tax", Min: 0, Max: 20_000, MaxStep: 5_000}},
			Schedule: []economysim.ScheduleStep{{Lever: "poll", At: 0, BP: 10_000}, {Lever: "poll", At: 120, BP: 15_000}},
		},
		{
			ID: "inland_council", Jurisdiction: "inland", Policy: "rule", Every: 30, ReadsRegions: true, Label: "Inland",
			Levers: []economysim.Lever{{ID: "stipend", Kind: economysim.LeverFlow, Flow: "bread_stipend", Min: 0, Max: 20_000, MaxStep: 1_000}},
			Rules: []economysim.Reaction{
				{Lever: "stipend", Gauge: economysim.GaugeUnfedShare, Target: 500, Deadband: 100, Span: 1_000, Move: 2_000, Raise: true},
			},
		},
	}}
}

// TestLoad_Policy verifies that the tables of spec-policy.md 3.1.3 load: an actor keyed
// by ID with its levers, schedule steps keyed by (lever, tick) and reactions keyed by
// lever, a key-less [modules.policy] table switching the module on with no actor, no table
// leaving it off, the file and the Go value hashing alike, and the label not hashed.
//
// Requirement: FR-PLC-003, FR-CFG-089
func TestLoad_Policy(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(policyDoc)
	if err != nil {
		t.Fatalf("Parse(policyDoc) error = %v, want nil", err)
	}
	want := policyWant()
	if !reflect.DeepEqual(got.Modules.Policy, want) {
		t.Errorf("Parse(policyDoc).Modules.Policy = %+v, want %+v", got.Modules.Policy, want)
	}
	goCfg := economysim.Config{World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar()}
	goCfg.Modules.Policy = want
	if economysim.HashConfig(goCfg) != economysim.HashConfig(got) {
		t.Error("HashConfig of the file differs from that of the same Go value")
	}
	if on, err := parseStrings(minimal + "[modules.policy]\n"); err != nil || !reflect.DeepEqual(on.Modules.Policy, &economysim.PolicyConfig{}) {
		t.Errorf("Parse(empty table).Modules.Policy = %+v, %v, want the module on and empty", on.Modules.Policy, err)
	}
	if off, _ := parseStrings(minimal); off.Modules.Policy != nil {
		t.Errorf("Parse(minimal).Modules.Policy = %+v, want nil: the module is off", off.Modules.Policy)
	}
}

// TestLoad_PolicyRejections verifies the keys and types the loader rejects in the policy
// tables, each with the rule and the path of the key.
//
// Requirement: FR-CFG-089
func TestLoad_PolicyRejections(t *testing.T) {
	t.Parallel()
	actor := "[modules.policy.actor.a]\njurisdiction = \"j\"\npolicy = \"hold\"\nevery = 1\n"
	lever := actor + "[modules.policy.actor.a.lever.l]\nkind = \"flow\"\nflow = \"f\"\nmin = 0\nmax = 1\n"
	at := func(more ...string) []string { return append([]string{"modules", "policy", "actor", "a"}, more...) }
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"an unknown key of the module", minimal + "[modules.policy]\nlevel = 1\n", economysim.ErrUnknownKey, []string{"modules", "policy", "level"}},
		{"an unknown key of an actor", minimal + actor + "label2 = \"x\"\n", economysim.ErrUnknownKey, at("label2")},
		{"an actor with no jurisdiction", minimal + "[modules.policy.actor.a]\npolicy = \"hold\"\nevery = 1\n", economysim.ErrMissingField, at("jurisdiction")},
		{"an actor with no policy", minimal + "[modules.policy.actor.a]\njurisdiction = \"j\"\nevery = 1\n", economysim.ErrMissingField, at("policy")},
		{"an actor with no cadence", minimal + "[modules.policy.actor.a]\njurisdiction = \"j\"\npolicy = \"hold\"\n", economysim.ErrMissingField, at("every")},
		{"a float cadence", minimal + "[modules.policy.actor.a]\njurisdiction = \"j\"\npolicy = \"hold\"\nevery = 1.5\n", economysim.ErrNotInteger, at("every")},
		{"a string reads_regions", minimal + "[modules.policy.actor.a]\njurisdiction = \"j\"\npolicy = \"hold\"\nevery = 1\nreads_regions = \"yes\"\n", economysim.ErrWrongType, at("reads_regions")},
		{"a lever with no kind", minimal + actor + "[modules.policy.actor.a.lever.l]\nmin = 0\nmax = 1\n", economysim.ErrMissingField, at("lever", "l", "kind")},
		{"a lever with no min", minimal + actor + "[modules.policy.actor.a.lever.l]\nkind = \"flow\"\nmax = 1\n", economysim.ErrMissingField, at("lever", "l", "min")},
		{"a lever with no max", minimal + actor + "[modules.policy.actor.a.lever.l]\nkind = \"flow\"\nmin = 1\n", economysim.ErrMissingField, at("lever", "l", "max")},
		{"an unknown key of a lever", minimal + lever + "step = 1\n", economysim.ErrUnknownKey, at("lever", "l", "step")},
		{"an integer may_close", minimal + lever + "may_close = 1\n", economysim.ErrWrongType, at("lever", "l", "may_close")},
		{"a schedule step with no value", minimal + lever + "[modules.policy.actor.a.schedule.l.5]\nclosed = true\n", economysim.ErrMissingField, at("schedule", "l", "5", "bp")},
		{"a schedule step whose tick is not a number", minimal + lever + "[modules.policy.actor.a.schedule.l.soon]\nbp = 1\n", economysim.ErrWrongType, at("schedule", "l", "soon")},
		{"an unknown key of a schedule step", minimal + lever + "[modules.policy.actor.a.schedule.l.5]\nbp = 1\nat = 5\n", economysim.ErrUnknownKey, at("schedule", "l", "5", "at")},
		{"a reaction with no gauge", minimal + lever + "[modules.policy.actor.a.rule.l]\ntarget = 1\n", economysim.ErrMissingField, at("rule", "l", "gauge")},
		{"a float target", minimal + lever + "[modules.policy.actor.a.rule.l]\ngauge = \"treasury\"\ntarget = 1.5\n", economysim.ErrNotInteger, at("rule", "l", "target")},
		{"an unknown key of a reaction", minimal + lever + "[modules.policy.actor.a.rule.l]\ngauge = \"treasury\"\ngain = 3\n", economysim.ErrUnknownKey, at("rule", "l", "gain")},
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
	if !slices.Equal(at("x"), []string{"modules", "policy", "actor", "a", "x"}) {
		t.Fatal("the path helper is wrong")
	}
}
