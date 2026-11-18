package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestInfrastructureWarnings verifies the seven warnings of spec-infrastructure.md
// 13.2: each case changes the valid configuration so that exactly its warnings are
// raised, once, with the rule and the path, and the configuration still validates (a
// warning never fails validation). The quiet case is the configuration itself, which
// TestValidateInfrastructure_ConfigIsValid pins as raising none.
//
// Requirement: FR-CAP-040
func TestInfrastructureWarnings(t *testing.T) {
	t.Parallel()
	forge := economysim.FacilityClass{ID: "forge", Kind: "site", Durability: 100, Capacity: 1}
	tests := []struct {
		name   string
		change func(c *economysim.Config)
		want   []string
	}{
		{"a site no recipe needs", func(c *economysim.Config) {
			c.Modules.Infrastructure.Classes = append(c.Modules.Infrastructure.Classes, forge)
			c.Modules.Infrastructure.Initial = append(c.Modules.Infrastructure.Initial, economysim.InitialFacility{Region: "north", Class: "forge", Count: 1})
		}, []string{"facility_unused@modules.infrastructure.class.forge"}},
		{"a class nothing creates", func(c *economysim.Config) {
			c.Modules.Infrastructure.Classes = append(c.Modules.Infrastructure.Classes,
				economysim.FacilityClass{ID: "pump", Kind: "ground", Durability: 10, BoostBP: 1, Resource: "arable"})
		}, []string{"facility_unbuildable@modules.infrastructure.class.pump"}},
		{"a recipe that needs a class nothing creates", func(c *economysim.Config) {
			c.Modules.Infrastructure.Classes = append(c.Modules.Infrastructure.Classes, forge)
			recipeOf(c, "grind").Needs = []string{"forge", "mill"}
		}, []string{
			"facility_unbuildable@modules.infrastructure.class.forge",
			"unrunnable_recipe@recipes.grind.needs",
		}},
		{"decay with no repair", func(c *economysim.Config) { facilityOf(c, "drain").Decay = 1 },
			[]string{"decay_without_repair@modules.infrastructure.class.drain"}},
		{"use wear with no repair", func(c *economysim.Config) {
			c.Modules.Infrastructure.Classes = append(c.Modules.Infrastructure.Classes,
				economysim.FacilityClass{ID: "oven", Kind: "site", Durability: 100, WearPerUse: 1, Capacity: 1})
			c.Modules.Infrastructure.Initial = append(c.Modules.Infrastructure.Initial, economysim.InitialFacility{Region: "north", Class: "oven", Count: 1})
			recipeOf(c, "grind").Needs = []string{"mill", "oven"}
		}, []string{"decay_without_repair@modules.infrastructure.class.oven"}},
		{"a capacity above the heaviest weight", func(c *economysim.Config) { facilityOf(c, "mill").Capacity = 100 },
			[]string{"capacity_never_binds@modules.infrastructure.class.mill"}},
		{"a route no one carries on", func(c *economysim.Config) { facilityOf(c, "road").Modes = []string{"foot"} },
			[]string{"route_unused_mode@modules.infrastructure.class.road"}},
		{"a row in a region whose class lacks the flag", func(c *economysim.Config) {
			m := economysim.DefaultSettlements()
			m.Dwell = 400
			m.Classes = []economysim.SettlementClass{{ID: "hamlet", Enter: 0}, {ID: "village", Enter: 300}, {ID: "city", Enter: 5_000, Flags: []string{"fair"}}}
			m.Regions = []economysim.RegionSettlement{economysim.DefaultRegionSettlement("north")}
			c.Modules.Settlements = &m
			facilityOf(c, "mill").Needs = "fair"
		}, []string{"facility_gate_closed@modules.infrastructure.initial[2]"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := infrastructureConfig()
			tt.change(&c)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate() = %v, want nil: a warning never fails validation", err)
			}
			got := facilityWarningsOf(c)
			slices.Sort(got)
			if !slices.Equal(got, tt.want) {
				t.Errorf("Warnings() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestInfrastructureWarnings_OffIsSilent verifies that a configuration with the
// module off raises none of its warnings.
//
// Requirement: FR-CAP-001, FR-CAP-040
func TestInfrastructureWarnings_OffIsSilent(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	if got := facilityWarningsOf(c); len(got) != 0 {
		t.Errorf("referenceConfig().Warnings() has %v of the module, want none", got)
	}
}
