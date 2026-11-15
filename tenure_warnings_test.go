package economysim_test

import (
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// tenureWarned returns the warnings of the tenure module of c as "path rule" lines, in
// the order Warnings returns them. The warnings of other modules are left out.
func tenureWarned(c economysim.Config) []string {
	var out []string
	for _, w := range c.Warnings() {
		if len(w.Path) > 1 && w.Path[0] == "modules" && w.Path[1] == "tenure" {
			out = append(out, strings.Join(w.Path, ".")+" "+w.Rule)
		}
	}
	return out
}

// TestWarnings_Tenure verifies each warning of spec-land.md 11.2 that needs no solver,
// with a configuration that breaks only its condition: the warning is raised at the
// path of the row it is about, and no other warning of the module with it.
//
// Requirement: FR-LND-039
func TestWarnings_Tenure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		want   []string
	}{
		{"no titles and no term with a rent or a closing", func(c *economysim.Config) {
			c.Modules.Tenure = &economysim.TenureConfig{}
		}, []string{"modules.tenure no_titles"}},
		{"a rent on an endowment no holding gives a title", func(c *economysim.Config) {
			c.Modules.Tenure.Holdings = nil
		}, []string{"modules.tenure.term[0] rent_without_titles"}},
		{"titles that pay no rent and do not close", func(c *economysim.Config) {
			c.Modules.Tenure.Terms = nil
			c.Modules.Tenure.RentStepBP = 0
		}, []string{"modules.tenure.holding[0] titles_without_rent"}},
		{"a holding for a role with no group in the region", func(c *economysim.Config) {
			c.Modules.Tenure.Holdings[0].Holder.Role = "visitor"
		}, []string{"modules.tenure.holding[0] holding_unused"}},
		{"a holding on an endowment of capacity 0", func(c *economysim.Config) {
			e := &c.Modules.Resources.Endowments[0]
			e.Capacity, e.HasInitial, e.Initial = 0, false, 0
		}, []string{"modules.tenure.holding[0] holding_unused", "modules.tenure.holding[1] holding_unused"}},
		{"a closed endowment fully titled and a role that plays on it with no title", func(c *economysim.Config) {
			c.Modules.Tenure.Terms[0].HasClosed, c.Modules.Tenure.Terms[0].Closed = true, true
			c.Modules.Tenure.Holdings[1].ShareBP = 4_940 // 253 * 20 + 4 940 = 10 000
			c.Roles[tenureRoleIndex(c, "maker")].Recipes = append(c.Roles[tenureRoleIndex(c, "maker")].Recipes, "grow_staple")
			// the closed term also stops the posted rent from moving
		}, []string{"modules.tenure.rent_step_bp rent_step_inert", "modules.tenure.term[0].role.maker role_locked_out"}},
		{"a title so small that its entitlement is below a unit", func(c *economysim.Config) {
			e := &c.Modules.Resources.Endowments[0]
			e.Capacity, e.Initial = 1_000, 500 // the plan stock is 600
			c.Modules.Tenure.Holdings[1].ShareBP = 1
		}, []string{"modules.tenure.holding[1] entitlement_below_unit"}},
		{"a rent above the value of the output of the recipe", func(c *economysim.Config) {
			c.Modules.Tenure.Terms[0].Rent = 1_000_000 // the staple of one execution is worth less
		}, []string{"modules.tenure.term[0].recipe.grow_staple rent_above_output"}},
		{"a title the pool may hold and the estates module off", func(c *economysim.Config) {
			c.Modules.Estates = nil
		}, []string{"modules.tenure.holding[0] rent_hoarded"}},
		{"a step with no rent that can move", func(c *economysim.Config) {
			c.Modules.Tenure.Terms[0].Rent = 0
		}, []string{"modules.tenure.holding[0] titles_without_rent", "modules.tenure.rent_step_bp rent_step_inert"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tenureConfig()
			tt.mutate(&c)
			if got := tenureWarned(c); !slices.Equal(got, tt.want) {
				t.Errorf("Warnings() of the tenure module = %q, want %q", got, tt.want)
			}
		})
	}
}

// roleIndex returns the index of the role with the given ID in c.
func tenureRoleIndex(c *economysim.Config, id string) int {
	return slices.IndexFunc(c.Roles, func(r economysim.Role) bool { return r.ID == id })
}

// TestWarnings_TenureOffAndSilent verifies that the module off raises none of its
// warnings, and that the valid configuration of tenureConfig raises none: the silent case
// each rule above is measured against.
//
// Requirement: FR-LND-039
func TestWarnings_TenureOffAndSilent(t *testing.T) {
	t.Parallel()
	c := tenureConfig()
	if got := tenureWarned(c); len(got) != 0 {
		t.Errorf("Warnings(tenureConfig) = %q, want none of the module's", got)
	}
	c.Modules.Tenure = nil
	if got := tenureWarned(c); len(got) != 0 {
		t.Errorf("Warnings(tenure off) = %q, want none", got)
	}
}

// TestWarnings_TenureSafeOnInvalid verifies that the warnings are safe to compute on a
// configuration that does not validate: an undeclared region, resource or role is
// skipped and nothing panics.
//
// Requirement: FR-LND-039
//
// Guard: Warnings is documented as safe on a configuration that does not validate.
func TestWarnings_TenureSafeOnInvalid(t *testing.T) {
	t.Parallel()
	c := tenureConfig()
	m := c.Modules.Tenure
	m.Terms = append(m.Terms, economysim.TenureTerm{Region: "nowhere", Resource: "nothing", HasRent: true, Rent: -5, HasClosed: true, Closed: true})
	m.Holdings = append(m.Holdings,
		economysim.Holding{Region: "nowhere", Resource: "nothing", Holder: economysim.HolderRef{Kind: "guild", Role: "serf"}, ShareBP: -4},
		economysim.Holding{Region: "north", Resource: "arable", Holder: economysim.HolderRef{Kind: economysim.HolderKindRole, Role: "serf"}, ShareBP: 1 << 40})
	_ = c.Warnings()
}
