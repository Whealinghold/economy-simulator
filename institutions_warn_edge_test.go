package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestWarnings_InstitutionsEdges verifies the edges of the module's warnings that the
// table of TestWarnings_Institutions does not reach: a world with no warm start has no
// seed cost, so cap_below_seed_cost is silent while the other warnings still fire; a
// second control on the same commodity, or on a second commodity of the same
// jurisdiction, reads the memoised seeds and floors and warns as the first does; a
// recipe that needs a tool that is not declared has no cost, so no warning, and a role
// that lists a recipe that is not declared is skipped, not read as a nil recipe; a lender
// with no household in the scope of a rate limit does not count, so the limit still
// warns for the lender that has one.
//
// Requirement: FR-ORG-048
//
// Guard: the seed table and the floors are cached per region, and each cache has a
// miss and a hit branch. Without the test a change of the key, or a cached value of an
// unfinished table, would change the warning of the second control and nothing else.
func TestWarnings_InstitutionsEdges(t *testing.T) {
	t.Parallel()
	const seedCost = "cap_below_seed_cost@modules.institutions.control."
	lowCap := func(m *economysim.Institutions) {
		m.Controls[0].CapBP, m.Controls[0].FloorBP = 6_000, 0
	}
	tests := []struct {
		name string
		set  orgSet
		want []string
		not  []string
	}{
		{"no warm start", func(c *economysim.Config, m *economysim.Institutions) {
			lowCap(m)
			c.WarmStart = nil
			m.RateLimits[0].CapBP = 500
		}, []string{"rate_limit_below_floor@modules.institutions.rate_limit.usury"}, []string{seedCost + "assize"}},
		{"second control on the same commodity", func(_ *economysim.Config, m *economysim.Institutions) {
			lowCap(m)
			second := m.Controls[0]
			second.ID = "assize_north"
			second.Jurisdiction, second.Region = "", "north"
			m.Controls = append(m.Controls, second)
		}, []string{seedCost + "assize", seedCost + "assize_north"}, nil},
		{"second control on another commodity", func(_ *economysim.Config, m *economysim.Institutions) {
			lowCap(m)
			second := m.Controls[0]
			second.ID = "assize_seed"
			second.Commodity = "seed"
			m.Controls = append(m.Controls, second)
		}, []string{seedCost + "assize"}, nil},
		{"recipe needs an undeclared tool", func(c *economysim.Config, m *economysim.Institutions) {
			lowCap(m)
			for i := range c.Recipes {
				c.Recipes[i].Requires = append(c.Recipes[i].Requires, economysim.Tool{Commodity: "nowhere", WearPerUse: 1})
			}
		}, nil, []string{seedCost + "assize"}},
		{"role lists an undeclared recipe", func(c *economysim.Config, m *economysim.Institutions) {
			lowCap(m)
			for i := range c.Roles {
				c.Roles[i].Recipes = append(c.Roles[i].Recipes, "nowhere")
			}
		}, []string{seedCost + "assize"}, nil},
		{"lender with no household in the scope", func(c *economysim.Config, m *economysim.Institutions) {
			m.RateLimits[0].CapBP = 500
			for i := range c.Roles {
				if c.Roles[i].ID == "visitor" {
					c.Roles[i].Credit = &economysim.CreditTerms{Lend: &economysim.LendTerms{RateBP: 400, MaxTermTicks: 90, KeepTicks: 3, ShareBP: 5_000}}
				}
			}
		}, []string{"rate_limit_below_floor@modules.institutions.rate_limit.usury"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := institutionsConfig()
			tt.set(&c, c.Modules.Institutions)
			got := institutionsWarningRules(c)
			for _, w := range tt.want {
				if n := occurrences(got, w); n != 1 {
					t.Errorf("Warnings() has %q %d times in %v, want once", w, n, got)
				}
			}
			for _, w := range tt.not {
				if slices.Contains(got, w) {
					t.Errorf("Warnings() = %v, want no %q", got, w)
				}
			}
		})
	}
}

// occurrences returns how many elements of list equal v.
func occurrences(list []string, v string) int {
	n := 0
	for _, s := range list {
		if s == v {
			n++
		}
	}
	return n
}

// TestWarnings_InstitutionsOverflowingCensus verifies that a settlements module whose
// thresholds cannot be computed (a band above 10 000, which Validate refuses) leaves the
// body warnings that read the census silent and does not panic.
//
// Requirement: FR-ORG-048
//
// Guard: Warnings runs on a configuration that failed validation, so a census that
// cannot be built must read as "no census", not as an empty one that raises
// body_needs_unmet_at_start for every body.
func TestWarnings_InstitutionsOverflowingCensus(t *testing.T) {
	t.Parallel()
	c := institutionsConfig()
	c.Modules.Settlements.BandBP = 20_000
	if err := c.Validate(); err == nil {
		t.Fatalf("Validate() = nil, want an error for a band above 10 000")
	}
	if got := institutionsWarningRules(c); slices.Contains(got, "body_needs_unmet_at_start@modules.institutions.body.guild") {
		t.Errorf("Warnings() = %v, want no body_needs_unmet_at_start without a census", got)
	}
}
