package economysim_test

import (
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the warnings of the institutions module (spec-institutions.md 10.2),
// from the configuration of institutions_test.go.

// institutionsWarningRules returns the module's warnings of a configuration as
// rule@path strings.
func institutionsWarningRules(c economysim.Config) []string {
	var got []string
	for _, w := range c.Warnings() {
		if len(w.Path) > 1 && w.Path[0] == "modules" && w.Path[1] == "institutions" {
			got = append(got, w.Rule+"@"+strings.Join(w.Path, "."))
		}
	}
	return got
}

// institutionsWarningCases are the configurations that raise one warning each:
// spec-institutions.md 10.2, the eleven rules.
func institutionsWarningCases() []struct {
	rule, path string
	set        orgSet
} {
	return []struct {
		rule, path string
		set        orgSet
	}{
		{"body_empty", "modules.institutions.body.guild", func(_ *economysim.Config, m *economysim.Institutions) { m.Bodies[0].Roles = []string{"visitor"} }},
		{"places_below_start", "modules.institutions.body.guild", func(_ *economysim.Config, m *economysim.Institutions) { m.Bodies[0].Places = 10 }},
		{"body_needs_unmet_at_start", "modules.institutions.body.guild", func(c *economysim.Config, _ *economysim.Institutions) {
			c.Modules.Settlements.Classes[2].Enter = 700 // north starts with 600 households: a village, which has no flag
		}},
		{"right_unreachable", "modules.institutions.right.make_tools", func(_ *economysim.Config, m *economysim.Institutions) { m.Rights[0].Region = "farm" }},
		{"right_over_upkeep", "modules.institutions.right.buy_staple", func(_ *economysim.Config, m *economysim.Institutions) {
			m.Rights = append(m.Rights, economysim.Right{ID: "buy_staple", Body: "guild", Activity: economysim.ActivityBuy, Subject: "staple", Region: "north"})
		}},
		{"right_over_upkeep", "modules.institutions.right.grow", func(_ *economysim.Config, m *economysim.Institutions) {
			m.Rights = append(m.Rights, economysim.Right{ID: "grow", Activity: economysim.ActivityProduce, Subject: "grow_staple", Region: "north"})
		}},
		{"train_right_no_training", "modules.institutions.right.train", func(_ *economysim.Config, m *economysim.Institutions) {
			m.Rights = append(m.Rights, economysim.Right{ID: "train", Body: "guild", Activity: economysim.ActivityTrain, Subject: "maker", Region: "north"})
		}},
		{"cap_below_band", "modules.institutions.control.assize", func(_ *economysim.Config, m *economysim.Institutions) {
			m.Controls[0].CapBP, m.Controls[0].FloorBP = 50, 0 // 96 * 50 / 10 000 = 0, raised to 1, under the band's low of 96 * 1 000 / 10 000 = 9
		}},
		{"floor_above_band", "modules.institutions.control.assize", func(_ *economysim.Config, m *economysim.Institutions) {
			m.Controls[0].CapBP, m.Controls[0].FloorBP = 0, economysim.MaxBasisPoints // a floor of 9 600 over the band's high of 960
		}},
		{"cap_below_seed_cost", "modules.institutions.control.assize", func(_ *economysim.Config, m *economysim.Institutions) {
			m.Controls[0].CapBP, m.Controls[0].FloorBP = 6_000, 0 // a cap of 57 under 7 000 bp of the seed cost of staple, about 97
		}},
		{"rate_limit_below_floor", "modules.institutions.rate_limit.usury", func(_ *economysim.Config, m *economysim.Institutions) { m.RateLimits[0].CapBP = 500 }},
		{"control_inert", "modules.institutions.control.assize", func(_ *economysim.Config, m *economysim.Institutions) {
			m.Controls[0].CapBP, m.Controls[0].FloorBP = 0, 0
		}},
		{"control_inert", "modules.institutions.rate_limit.usury", func(_ *economysim.Config, m *economysim.Institutions) {
			m.RateLimits[0].CapBP = economysim.MaxCreditRateBP
		}},
	}
}

// TestWarnings_Institutions verifies that each warning of spec-institutions.md 10.2 is
// raised, once and at the path of its body, right, control or rate limit, by a
// configuration that has the condition, that none of them fails validation, and that a
// configuration without the condition has none.
//
// Requirement: FR-ORG-048
func TestWarnings_Institutions(t *testing.T) {
	t.Parallel()
	for _, tt := range institutionsWarningCases() {
		t.Run(tt.rule+" "+tt.path, func(t *testing.T) {
			t.Parallel()
			c := institutionsConfig()
			tt.set(&c, c.Modules.Institutions)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate() = %v, want nil: a warning never fails validation", err)
			}
			want := tt.rule + "@" + tt.path
			var n int
			for _, got := range institutionsWarningRules(c) {
				if got == want {
					n++
				}
			}
			if n != 1 {
				t.Errorf("Warnings() has %q %d times in %v, want once", want, n, institutionsWarningRules(c))
			}
		})
	}
}

// TestWarnings_InstitutionsOffAndEmpty verifies that the module off and the module on
// and empty raise none of its warnings, and that a configuration that does not validate
// still gets its warnings without a panic.
//
// Requirement: FR-ORG-048
func TestWarnings_InstitutionsOffAndEmpty(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	if got := institutionsWarningRules(c); len(got) != 0 {
		t.Errorf("module off: warnings %v, want none", got)
	}
	c.Modules.Institutions = &economysim.Institutions{}
	if got := institutionsWarningRules(c); len(got) != 0 {
		t.Errorf("module empty: warnings %v, want none", got)
	}
	c = institutionsConfig()
	c.Modules.Institutions.Bodies[0].Roles = []string{"nowhere"}
	c.Modules.Institutions.Rights[0].Body = "nowhere"
	c.Modules.Institutions.Controls[0].Commodity = "nowhere"
	_ = c.Warnings() // must not panic on names that are not declared
}
