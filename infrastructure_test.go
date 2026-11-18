package economysim_test

import (
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// infrastructureConfig returns referenceConfig with the transport, resources and
// storage modules on and the infrastructure module with one class of each kind and
// one initial row of each: a mill (site, capacity 2, wear 2 a use, decay 1, the only class that decays: a recipe repairs it), a road
// (route, the hired mode coach, from north to south), a drain (ground, on arable) and a barn
// (store, installing a bin, held by the growers). The grower recipes draw on arable
// and bins give the growers room. Two recipes with the clauses are on the maker's
// list: one builds a mill and one repairs it. The numbers are placeholders (N-015).
func infrastructureConfig() economysim.Config {
	c := transportConfig()
	c.World.Scale = 100 // heavy agents, so the capacity of the mill binds
	c.Modules.Transport.Modes = append(c.Modules.Transport.Modes, hiredMode())
	withResources(&c)
	c.Modules.Storage = &economysim.Storage{
		Roles: []economysim.RoleRoom{
			{Role: "grower", Base: &economysim.Capacity{Mass: 2_000, Bulk: 2_000}, Stores: []string{"bin"}},
		},
		Classes: []economysim.StoreClass{{ID: "bin", Mass: 400, Bulk: 300}},
	}
	c.Modules.Infrastructure = &economysim.Infrastructure{
		Classes: []economysim.FacilityClass{
			{ID: "barn", Kind: "store", Durability: 800, Store: "bin"},
			{ID: "drain", Kind: "ground", Durability: 500, BoostBP: 3_000, Resource: "arable"},
			{ID: "mill", Kind: "site", Durability: 1_000, Decay: 1, WearPerUse: 2, Capacity: 2},
			{ID: "road", Kind: "route", Durability: 2_000, BoostBP: 6_000, Modes: []string{"coach"}},
		},
		Initial: []economysim.InitialFacility{
			{Region: "north", Class: "barn", Role: "grower", Count: 1},
			{Region: "north", Class: "drain", Count: 1},
			{Region: "north", Class: "mill", Role: "grower", Count: 2},
			{Region: "north", Class: "road", Toward: "south", Count: 1},
		},
	}
	build := economysim.DefaultRecipe()
	build.ID, build.Duration = "build_mill", 3
	build.Inputs = []economysim.Item{{Commodity: "fuel", Quantity: 2}}
	build.Builds = &economysim.FacilityBuild{Class: "mill"}
	repair := economysim.DefaultRecipe()
	repair.ID, repair.Duration = "repair_mill", 1
	repair.Inputs = []economysim.Item{{Commodity: "fuel", Quantity: 1}}
	repair.Repairs = &economysim.FacilityRepair{Class: "mill", Units: 100}
	grind := economysim.DefaultRecipe()
	grind.ID, grind.Duration = "grind", 1
	grind.Inputs = []economysim.Item{{Commodity: "seed", Quantity: 1}}
	grind.Outputs = []economysim.Output{{Commodity: "staple", Quantity: 2, CostShare: 10_000}}
	grind.Needs = []string{"mill"}
	c.Recipes = append(c.Recipes, build, repair, grind)
	m := roleOf(&c, "grower")
	m.Recipes = append(m.Recipes, "build_mill", "repair_mill", "grind")
	return c
}

// facilityWarningsOf returns the rule@path strings of the warnings of c that are about
// the module (its blocks) or about the clauses of a recipe.
func facilityWarningsOf(c economysim.Config) []string {
	var got []string
	for _, w := range c.Warnings() {
		p := strings.Join(w.Path, ".")
		if strings.HasPrefix(p, "modules.infrastructure") || (len(w.Path) == 3 && w.Path[0] == "recipes" && w.Path[2] == "needs") {
			got = append(got, w.Rule+"@"+p)
		}
	}
	return got
}

// TestValidateInfrastructure_ConfigIsValid verifies that the configuration the
// infrastructure tests start from validates and raises none of the module's
// warnings, and that the empty module is valid and inert.
//
// Requirement: FR-CAP-003, FR-CAP-001
//
// Guard: the field cases start from infrastructureConfig; if it did not validate,
// each case would pass for the wrong reason.
func TestValidateInfrastructure_ConfigIsValid(t *testing.T) {
	t.Parallel()
	c := infrastructureConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("infrastructureConfig().Validate() = %v, want nil", err)
	}
	if got := facilityWarningsOf(c); len(got) != 0 {
		t.Errorf("infrastructureConfig().Warnings() has %v, want none of the module's", got)
	}
	empty := referenceConfig()
	empty.Modules.Infrastructure = &economysim.Infrastructure{}
	if err := empty.Validate(); err != nil {
		t.Errorf("the empty module .Validate() = %v, want nil: it is on, has no class and does nothing", err)
	}
}
