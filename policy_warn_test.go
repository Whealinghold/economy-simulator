package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestWarnings_Policy verifies FR-PLC-013: one case for each of the nine warnings of
// spec-policy.md 11.2, with the rule and the path, and that none of them fails validation.
//
// Requirement: FR-PLC-013
func TestWarnings_Policy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		rule string
		path []string
		set  func(c *economysim.Config, m *economysim.PolicyConfig)
	}{
		{"no actors", "no_actors", []string{"modules", "policy"}, func(c *economysim.Config, m *economysim.PolicyConfig) { m.Actors = nil }},
		{"an actor with no lever", "actor_no_levers", polPath(1), func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[1].Levers = nil
		}},
		{"a lever that cannot move", "lever_fixed", polPath(0, "levers[0]", "min"), func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[0].Levers[0].Min, m.Actors[0].Levers[0].Max = 10_000, 10_000
			m.Actors[0].Schedule = nil
		}},
		{"an Against that expands to no edge", "lever_no_keys", polPath(1, "levers[0]", "against"), func(c *economysim.Config, m *economysim.PolicyConfig) {
			c.Edges = c.Edges[2:]
		}},
		{"reads_regions with a policy that reads none", "reads_unused", polPath(0, "reads_regions"), func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[0].ReadsRegions = true
		}},
		{"a reaction with no step limit and no deadband", "rule_undamped", polPath(2, "rules[0]", "deadband"), func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[2].Levers[0].MaxStep, m.Actors[2].Rules[0].Deadband = 0, 0
		}},
		{"a reaction with a move of 0", "rule_inert", polPath(2, "rules[0]", "move"), func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[2].Rules[0].Move = 0
		}},
		{"a cadence above the year", "cadence_long", polPath(0, "every"), func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[0].Every, m.Actors[0].Schedule = 3_000, nil
		}},
		{"two steps closer than the cadence", "schedule_dense", polPath(0, "levers[0]", "max_step"), func(c *economysim.Config, m *economysim.PolicyConfig) {
			m.Actors[0].Schedule[1].At = 235
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := policyConfig()
			tt.set(&c, c.Modules.Policy)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate() = %v, want nil: a warning never fails validation", err)
			}
			var got []economysim.Warning
			for _, w := range c.Warnings() {
				if slices.Contains(policyWarningRules, w.Rule) {
					got = append(got, w)
				}
			}
			if len(got) != 1 || got[0].Rule != tt.rule || !slices.Equal(got[0].Path, tt.path) {
				t.Errorf("policy warnings = %v, want one %s at %v", got, tt.rule, tt.path)
			}
		})
	}
}

// TestWarnings_PolicyOffAndDense verifies that a configuration with the module off has
// none of the nine warnings, that a consumer's policy that may read regions raises no
// reads_unused, and that schedule_dense counts the wrap of a cycle as well.
//
// Requirement: FR-PLC-013
func TestWarnings_PolicyOffAndDense(t *testing.T) {
	t.Parallel()
	off := fiscalConfig()
	for _, w := range off.Warnings() {
		if slices.Contains(policyWarningRules, w.Rule) {
			t.Errorf("module off has warning %v", w)
		}
	}
	c := policyConfig()
	c.Modules.Policy.Actors[0].Policy = "consumer_policy"
	c.Modules.Policy.Actors[0].ReadsRegions = true
	c.Modules.Policy.Actors[0].Schedule, c.Modules.Policy.Actors[0].Cycle = nil, 0
	for _, w := range c.Warnings() {
		if w.Rule == "reads_unused" {
			t.Errorf("a consumer's policy raised %v", w)
		}
	}
	d := policyConfig()
	a := &d.Modules.Policy.Actors[0]
	a.Cycle = 250 // the last step at 240 and the first at 0 + 250 are 10 apart: not below Every 10
	a.Schedule[2].At = 245
	var dense int
	for _, w := range d.Warnings() {
		if w.Rule == "schedule_dense" {
			dense++
		}
	}
	if dense != 1 {
		t.Errorf("schedule_dense raised %d times for a wrap 5 ticks wide, want 1", dense)
	}
}
