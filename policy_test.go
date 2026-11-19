package economysim_test

import (
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the policy module's configuration surface (spec-policy.md 10, 11): the
// failing case of each rule and each row of the parameter-owner table, their order and
// the nine warnings. The arithmetic is tested in internal/policy and the hash in
// policy_hash_test.go.

// policyConfig returns fiscalConfig with the policy module on, in the invented kingdom
// of spec-policy.md 14: crown_council holds the head tax and a footpath by a schedule,
// coast_council holds the border between the coast and the inland province as a
// directed lever that may close it, and inland_council moves its stipend by a rule on
// the unfed share. The edges are the four that join the harbour and the capital to the
// mill, and the transport module has one mode. The actors are written crown, coast,
// inland, so the given index differs from the sorted one. None of the module's
// warnings is raised.
func policyConfig() economysim.Config {
	c := fiscalConfig()
	c.Edges = []economysim.Edge{
		{From: "harbour", To: "mill", Distance: 40}, {From: "mill", To: "harbour", Distance: 40},
		{From: "capital", To: "mill", Distance: 40}, {From: "mill", To: "capital", Distance: 40},
	}
	c.Modules.Transport = &economysim.Transport{Modes: []economysim.Mode{footMode()}}
	c.Modules.Fiscal.Payouts = append(c.Modules.Fiscal.Payouts, economysim.Payout{
		ID: "stipend", Authority: "inland", Amount: 1,
		Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Region: "farm"},
	})
	c.Modules.Policy = &economysim.PolicyConfig{Actors: []economysim.PolicyActor{
		{
			ID: "crown_council", Jurisdiction: "crown", Policy: "schedule", Every: 10,
			Levers: []economysim.Lever{
				{ID: "poll", Kind: economysim.LeverFlow, Flow: "poll", Min: 0, Max: 20_000, MaxStep: 5_000},
				{ID: "road", Kind: economysim.LeverTransit, From: "capital", To: "mill", Mode: "foot", Min: 0, Max: 20_000},
			},
			Schedule: []economysim.ScheduleStep{
				{Lever: "poll", At: 0, BP: 10_000}, {Lever: "poll", At: 120, BP: 15_000}, {Lever: "poll", At: 240, BP: 10_000},
			},
		},
		{
			ID: "coast_council", Jurisdiction: "coast", Policy: "directed", Every: 10,
			Levers: []economysim.Lever{
				{ID: "gate", Kind: economysim.LeverEdge, Against: "inland", Min: 10_000, Max: 40_000, MayClose: true},
			},
		},
		{
			ID: "inland_council", Jurisdiction: "inland", Policy: "rule", Every: 30, ReadsRegions: true,
			Levers: []economysim.Lever{
				{ID: "stipend", Kind: economysim.LeverFlow, Flow: "stipend", Min: 0, Max: 20_000, MaxStep: 1_000},
			},
			Rules: []economysim.Reaction{
				{Lever: "stipend", Gauge: economysim.GaugeUnfedShare, Target: 500, Deadband: 100, Span: 1_000, Move: 2_000, Raise: true},
			},
		},
	}}
	return c
}

// polityWarningRules are the nine warnings of spec-policy.md 11.2.
var policyWarningRules = []string{
	"no_actors", "actor_no_levers", "lever_fixed", "lever_no_keys", "reads_unused",
	"rule_undamped", "rule_inert", "cadence_long", "schedule_dense",
}

// polPath is the path of a key of an actor or of an element of one of its lists, as the
// validator writes it: modules.policy.actors[i], then the list and index, then the key.
func polPath(i int, rest ...string) []string {
	return append([]string{"modules", "policy", "actors[" + itoaT(i) + "]"}, rest...)
}

// itoaT writes a small int for a path.
func itoaT(n int) string { return string(rune('0' + n)) }

// polSet is the change of a case: it gets the configuration and its module.
type polSet func(c *economysim.Config, m *economysim.PolicyConfig)

// polCase builds one failing case that starts from policyConfig.
func polCase(field, name string, rule error, p []string, value, bound string, set polSet) fieldCase {
	return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
		mutate: func(c *economysim.Config) {
			*c = policyConfig()
			set(c, c.Modules.Policy)
		}}
}

// TestValidatePolicy_ConfigIsValid verifies that the configuration the policy tests
// start from validates and raises none of the module's warnings, so every failing case
// breaks one rule.
//
// Guard: the field cases start from this configuration; if it did not validate, each
// case would pass for the wrong reason.
func TestValidatePolicy_ConfigIsValid(t *testing.T) {
	t.Parallel()
	c := policyConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("policyConfig().Validate() = %v, want nil", err)
	}
	for _, w := range c.Warnings() {
		if slices.Contains(policyWarningRules, w.Rule) {
			t.Errorf("policyConfig().Warnings() has %v, want none of the module's", w)
		}
	}
}

// policyFieldCases are the failing cases of the rows of the policy module in the
// parameter-owner table (spec-policy.md 10, 11.1, FR-PLC-002 to 012): each breaks one
// field of the valid configuration of policyConfig. buildFieldCases appends them to the
// table that the registry test reads.
func policyFieldCases() []fieldCase {
	var cs []fieldCase
	cs = append(cs, polActorCases()...)
	cs = append(cs, polLeverCases()...)
	cs = append(cs, polKeyCases()...)
	cs = append(cs, polScheduleCases()...)
	return append(cs, polRuleCases()...)
}

// polActorCases are the cases of the module and the actor rows.
func polActorCases() []fieldCase {
	bounds := economysim.ErrOutOfBounds
	many := func(c *economysim.Config, m *economysim.PolicyConfig) {
		for len(m.Actors) <= economysim.MaxActors {
			m.Actors = append(m.Actors, economysim.PolicyActor{ID: "a" + strings.Repeat("x", len(m.Actors)), Jurisdiction: "crown", Policy: "hold", Every: 1})
		}
	}
	return []fieldCase{
		{field: "Modules.Policy", name: "needs the polity module", rules: []error{economysim.ErrModuleDependency},
			mutate: func(c *economysim.Config) { *c = policyConfig(); c.Modules.Polity = nil; c.Modules.Fiscal = nil },
			paths:  [][]string{{"modules", "policy"}}, value: "polity"},
		polCase("PolicyConfig.Actors", "more than MaxActors", economysim.ErrTooMany, []string{"modules", "policy", "actors"}, "65", "64", many),
		polCase("PolicyActor.ID", "invalid", economysim.ErrInvalidID, polPath(0, "id"), "Crown", "", func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[0].ID = "Crown"
		}),
		polCase("PolicyActor.ID", "repeated", economysim.ErrDuplicateID, polPath(1, "id"), "crown_council", "", func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[1].ID = "crown_council"
		}),
		polCase("PolicyActor.Jurisdiction", "undefined", economysim.ErrUnknownJurisdiction, polPath(0, "jurisdiction"), "nowhere", "", func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[0].Jurisdiction = "nowhere"
		}),
		polCase("PolicyActor.Jurisdiction", "two actors on one jurisdiction (rule one_actor_per_jurisdiction)", economysim.ErrPolicyShape, polPath(1, "jurisdiction"), "crown", "one_actor_per_jurisdiction", func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[1].Jurisdiction = "crown"
		}),
		polCase("PolicyActor.Every", "below 1", bounds, polPath(0, "every"), "0", "1", func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[0].Every = 0
		}),
		polCase("PolicyActor.Every", "above MaxPolicyEvery", bounds, polPath(0, "every"), "3651", "3650", func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[0].Every = economysim.MaxPolicyEvery + 1
		}),
		polCase("PolicyActor.Cycle", "above MaxPolicyCycle", bounds, polPath(0, "cycle"), "3651", "3650", func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[0].Cycle = economysim.MaxPolicyCycle + 1
		}),
		polCase("PolicyActor.Cycle", "with a policy that is not schedule (rule policy_data)", economysim.ErrPolicyShape, polPath(1, "schedule"), "directed", "policy_data", func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[1].Cycle = 100
		}),
		polCase("PolicyActor.Schedule", "with a policy that is not schedule (rule policy_data)", economysim.ErrPolicyShape, polPath(2, "schedule"), "rule", "policy_data", func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[2].Schedule = []economysim.ScheduleStep{{Lever: "stipend", At: 0, BP: 10_000}}
		}),
		polCase("PolicyActor.Rules", "with a policy that is not rule (rule policy_data)", economysim.ErrPolicyShape, polPath(0, "rules"), "schedule", "policy_data", func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[0].Rules = []economysim.Reaction{{Lever: "poll", Gauge: economysim.GaugeTreasury, Span: 1}}
		}),
		polCase("PolicyActor.ReadsRegions", "off with a region gauge (rule rule_reads)", economysim.ErrPolicyShape, polPath(2, "rules[0]", "gauge"), "unfed_share", "rule_reads", func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[2].ReadsRegions = false
		}),
		polCase("PolicyActor.Levers", "more than MaxLevers", economysim.ErrTooMany, polPath(1, "levers"), "33", "32", func(c *economysim.Config, m *economysim.PolicyConfig) {
			for len(m.Actors[1].Levers) <= economysim.MaxLevers {
				l := m.Actors[1].Levers[0]
				l.ID = "g" + strings.Repeat("x", len(m.Actors[1].Levers))
				m.Actors[1].Levers = append(m.Actors[1].Levers, l)
			}
		}),
		polCase("PolicyActor.Schedule", "more than MaxScheduleSteps", economysim.ErrTooMany, polPath(0, "schedule"), "257", "256", func(c *economysim.Config, m *economysim.PolicyConfig) {
			for at := int64(1_000); len(m.Actors[0].Schedule) <= economysim.MaxScheduleSteps; at++ {
				m.Actors[0].Schedule = append(m.Actors[0].Schedule, economysim.ScheduleStep{Lever: "poll", At: at, BP: 10_000})
			}
		}),
	}
}

// polLeverCases are the cases of the lever rows other than the keys and the reach.
func polLeverCases() []fieldCase {
	bounds := economysim.ErrOutOfBounds
	lever := func(i, j int, f func(l *economysim.Lever)) polSet {
		return func(c *economysim.Config, m *economysim.PolicyConfig) { f(&m.Actors[i].Levers[j]) }
	}
	lp := func(i, j int, key string) []string { return polPath(i, "levers["+itoaT(j)+"]", key) }
	return []fieldCase{
		polCase("Lever.ID", "invalid", economysim.ErrInvalidID, lp(0, 0, "id"), "Poll", "", lever(0, 0, func(l *economysim.Lever) { l.ID = "Poll" })),
		polCase("Lever.ID", "repeated within the actor", economysim.ErrDuplicateID, lp(0, 1, "id"), "poll", "", lever(0, 1, func(l *economysim.Lever) { l.ID = "poll" })),
		polCase("Lever.Kind", "reserved kind rate", bounds, lp(0, 0, "kind"), "rate", "flow, edge or transit", lever(0, 0, func(l *economysim.Lever) { l.Kind = "rate" })),
		polCase("Lever.Kind", "a kind that is not offered", bounds, lp(0, 0, "kind"), "shock", "flow, edge or transit", lever(0, 0, func(l *economysim.Lever) { l.Kind = "shock" })),
		polCase("Lever.Min", "above the slot's bound", bounds, lp(0, 0, "min"), "100001", "100000", lever(0, 0, func(l *economysim.Lever) { l.Min = 100_001 })),
		polCase("Lever.Min", "above Max", bounds, lp(0, 0, "min"), "20001", "20000", lever(0, 0, func(l *economysim.Lever) { l.Min = 20_001 })),
		polCase("Lever.Min", "above the default", bounds, lp(0, 0, "min"), "10001", "10000", lever(0, 0, func(l *economysim.Lever) { l.Min = 10_001 })),
		polCase("Lever.Max", "below the default", bounds, lp(0, 0, "max"), "9999", "10000", lever(0, 0, func(l *economysim.Lever) { l.Max = 9_999 })),
		polCase("Lever.Max", "above the slot's bound", bounds, lp(0, 0, "max"), "100001", "100000", lever(0, 0, func(l *economysim.Lever) { l.Max = 100_001 })),
		polCase("Lever.MaxStep", "above the slot's bound", bounds, lp(0, 0, "max_step"), "100001", "100000", lever(0, 0, func(l *economysim.Lever) { l.MaxStep = 100_001 })),
		polCase("Lever.MaxStep", "below 0", bounds, lp(0, 0, "max_step"), "-1", "0", lever(0, 0, func(l *economysim.Lever) { l.MaxStep = -1 })),
		polCase("Lever.MayClose", "on a kind other than edge (rule lever_key)", economysim.ErrPolicyShape, lp(0, 0, "kind"), "flow", "lever_key", lever(0, 0, func(l *economysim.Lever) { l.MayClose = true })),
	}
}

// polKeyCases are the cases of the key fields of a lever and of its reach.
func polKeyCases() []fieldCase {
	lever := func(i, j int, f func(l *economysim.Lever)) polSet {
		return func(c *economysim.Config, m *economysim.PolicyConfig) { f(&m.Actors[i].Levers[j]) }
	}
	lp := func(i, j int, key string) []string { return polPath(i, "levers["+itoaT(j)+"]", key) }
	shape := economysim.ErrPolicyShape
	return []fieldCase{
		polCase("Lever.Flow", "on an edge (rule lever_key)", shape, lp(1, 0, "kind"), "edge", "lever_key", lever(1, 0, func(l *economysim.Lever) { l.Flow = "poll" })),
		polCase("Lever.Flow", "undefined", economysim.ErrUnknownFlow, lp(0, 0, "flow"), "nothing", "", lever(0, 0, func(l *economysim.Lever) { l.Flow = "nothing" })),
		polCase("Lever.Flow", "outside the reach of the jurisdiction", economysim.ErrLeverNotInReach, lp(2, 0, "kind"), "alms", "inland", lever(2, 0, func(l *economysim.Lever) { l.Flow = "alms" })),
		polCase("Lever.From", "without To (rule lever_key)", shape, lp(0, 1, "kind"), "transit", "lever_key", lever(0, 1, func(l *economysim.Lever) { l.To = "" })),
		polCase("Lever.From", "undefined region", economysim.ErrUnknownRegion, lp(0, 1, "from"), "nowhere", "", lever(0, 1, func(l *economysim.Lever) { l.From = "nowhere" })),
		polCase("Lever.From", "an outside edge pair that is not configured", economysim.ErrUnknownEdge, lp(1, 0, "from"), "harbour>capital", "", lever(1, 0, func(l *economysim.Lever) { l.Against, l.From, l.To = "", "harbour", "capital" })),
		polCase("Lever.To", "equal to From (rule transit_pair)", shape, lp(0, 1, "to"), "capital>capital", "transit_pair", lever(0, 1, func(l *economysim.Lever) { l.To = "capital" })),
		polCase("Lever.To", "undefined region", economysim.ErrUnknownRegion, lp(0, 1, "to"), "nowhere", "", lever(0, 1, func(l *economysim.Lever) { l.To = "nowhere" })),
		polCase("Lever.Mode", "on a flow (rule lever_key)", shape, lp(0, 0, "kind"), "flow", "lever_key", lever(0, 0, func(l *economysim.Lever) { l.Mode = "foot" })),
		polCase("Lever.Mode", "undefined", economysim.ErrUnknownMode, lp(0, 1, "mode"), "boat", "", lever(0, 1, func(l *economysim.Lever) { l.Mode = "boat" })),
		polCase("Lever.Against", "with From (rule lever_key)", shape, lp(1, 0, "kind"), "edge", "lever_key", lever(1, 0, func(l *economysim.Lever) { l.From = "harbour" })),
		polCase("Lever.Against", "undefined jurisdiction", economysim.ErrUnknownJurisdiction, lp(1, 0, "against"), "nowhere", "", lever(1, 0, func(l *economysim.Lever) { l.Against = "nowhere" })),
		polCase("Lever.Against", "the actor's own jurisdiction (rule against_nested)", shape, lp(1, 0, "against"), "coast", "against_nested", lever(1, 0, func(l *economysim.Lever) { l.Against = "coast" })),
		polCase("Lever.Against", "an ancestor (rule against_nested)", shape, lp(1, 0, "against"), "crown", "against_nested", lever(1, 0, func(l *economysim.Lever) { l.Against = "crown" })),
		polCase("Lever.Against", "a descendant (rule against_nested)", shape, lp(2, 0, "against"), "uplands", "against_nested", func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[2].Levers = append(m.Actors[2].Levers, economysim.Lever{ID: "wall", Kind: economysim.LeverEdge, Against: "uplands", Min: 10_000, Max: 20_000})
			m.Actors[2].Levers[0], m.Actors[2].Levers[1] = m.Actors[2].Levers[1], m.Actors[2].Levers[0]
		}),
	}
}

// polScheduleCases are the cases of the schedule step rows.
func polScheduleCases() []fieldCase {
	bounds := economysim.ErrOutOfBounds
	shape := economysim.ErrPolicyShape
	step := func(j int, f func(s *economysim.ScheduleStep)) polSet {
		return func(c *economysim.Config, m *economysim.PolicyConfig) { f(&m.Actors[0].Schedule[j]) }
	}
	sp := func(j int, key string) []string { return polPath(0, "schedule["+itoaT(j)+"]", key) }
	return []fieldCase{
		polCase("ScheduleStep.Lever", "names no lever of the actor (rule schedule_lever)", shape, sp(1, "lever"), "nothing", "schedule_lever", step(1, func(s *economysim.ScheduleStep) { s.Lever = "nothing" })),
		polCase("ScheduleStep.At", "above MaxScheduleTick", bounds, sp(1, "at"), "1000000001", "1000000000", step(1, func(s *economysim.ScheduleStep) { s.At = economysim.MaxScheduleTick + 1 })),
		polCase("ScheduleStep.At", "not below the cycle", bounds, sp(2, "at"), "240", "199", func(c *economysim.Config, m *economysim.PolicyConfig) { m.Actors[0].Cycle = 200 }),
		polCase("ScheduleStep.At", "repeated for a lever", economysim.ErrDuplicateID, sp(1, "at"), "0", "", step(1, func(s *economysim.ScheduleStep) { s.At = 0 })),
		polCase("ScheduleStep.BP", "outside the slot's bound", bounds, sp(1, "bp"), "100001", "100000", step(1, func(s *economysim.ScheduleStep) { s.BP = 100_001 })),
		polCase("ScheduleStep.BP", "outside the lever's bounds (rule schedule_value)", shape, sp(1, "bp"), "20001", "schedule_value", step(1, func(s *economysim.ScheduleStep) { s.BP = 20_001 })),
		polCase("ScheduleStep.BP", "two consecutive values further apart than MaxStep (rule schedule_step)", shape, sp(1, "bp"), "16000", "schedule_step", func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[0].Schedule[1].BP, m.Actors[0].Schedule[2].BP = 16_000, 16_000
		}),
		polCase("ScheduleStep.Closed", "without MayClose (rule schedule_value)", shape, sp(1, "bp"), "15000", "schedule_value", step(1, func(s *economysim.ScheduleStep) { s.Closed = true })),
	}
}

// polRuleCases are the cases of the reaction rows.
func polRuleCases() []fieldCase {
	bounds := economysim.ErrOutOfBounds
	shape := economysim.ErrPolicyShape
	rule := func(f func(r *economysim.Reaction)) polSet {
		return func(c *economysim.Config, m *economysim.PolicyConfig) { f(&m.Actors[2].Rules[0]) }
	}
	rp := func(key string) []string { return polPath(2, "rules[0]", key) }
	return []fieldCase{
		polCase("Reaction.Lever", "names no lever (rule rule_lever)", shape, rp("lever"), "nothing", "rule_lever", rule(func(r *economysim.Reaction) { r.Lever = "nothing" })),
		polCase("Reaction.Lever", "two reactions on one lever", economysim.ErrDuplicateID, polPath(2, "rules[1]", "lever"), "stipend", "", func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[2].Rules = append(m.Actors[2].Rules, m.Actors[2].Rules[0])
		}),
		polCase("Reaction.Gauge", "not one of the four (rule rule_gauge)", shape, rp("gauge"), "gini", "rule_gauge", rule(func(r *economysim.Reaction) { r.Gauge = "gini" })),
		polCase("Reaction.Gauge", "treasury on a grouping (ErrNotGoverned)", economysim.ErrNotGoverned, polPath(1, "rules[0]", "gauge"), "coast", "", func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[1].Policy = "rule"
			m.Actors[1].Rules = []economysim.Reaction{{Lever: "gate", Gauge: economysim.GaugeTreasury, Span: 1}}
		}),
		polCase("Reaction.Commodity", "on a gauge that is not price (rule rule_gauge)", shape, rp("gauge"), "unfed_share", "rule_gauge", rule(func(r *economysim.Reaction) { r.Commodity = "staple" })),
		polCase("Reaction.Commodity", "missing for a price gauge (rule rule_gauge)", shape, rp("gauge"), "price", "rule_gauge", rule(func(r *economysim.Reaction) { r.Gauge, r.MaxAge = economysim.GaugePrice, 3 })),
		polCase("Reaction.Commodity", "undefined", economysim.ErrUnknownCommodity, rp("commodity"), "nothing", "", rule(func(r *economysim.Reaction) {
			r.Gauge, r.Commodity, r.MaxAge = economysim.GaugePrice, "nothing", 3
		})),
		polCase("Reaction.MaxAge", "above MaxPolicyEvery", bounds, rp("max_age"), "3651", "3650", rule(func(r *economysim.Reaction) {
			r.Gauge, r.Commodity, r.MaxAge = economysim.GaugePrice, "staple", economysim.MaxPolicyEvery+1
		})),
		polCase("Reaction.MaxAge", "missing for a price gauge (rule rule_gauge)", shape, rp("gauge"), "price", "rule_gauge", rule(func(r *economysim.Reaction) { r.Gauge, r.Commodity = economysim.GaugePrice, "staple" })),
		polCase("Reaction.Target", "above MaxMoney", bounds, rp("target"), "1000000000000000001", "1000000000000000000", rule(func(r *economysim.Reaction) { r.Target = economysim.MaxMoney + 1 })),
		polCase("Reaction.Deadband", "below 0", bounds, rp("deadband"), "-1", "0", rule(func(r *economysim.Reaction) { r.Deadband = -1 })),
		polCase("Reaction.Span", "below 1", bounds, rp("span"), "0", "1", rule(func(r *economysim.Reaction) { r.Span = 0 })),
		polCase("Reaction.Move", "above MaxMoveBP", bounds, rp("move"), "100001", "100000", rule(func(r *economysim.Reaction) { r.Move = economysim.MaxMoveBP + 1 })),
	}
}
