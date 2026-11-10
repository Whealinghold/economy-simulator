package otelobserver_test

import (
	economysim "github.com/Whealinghold/economy-sim"
)

// miniConfig is a world built in Go, because the module cannot read the core's
// testdata (testing.md 10.2): two regions, two commodities and two roles. A
// farmer makes grain and a weaver makes cloth, and every household eats one of
// each a tick, so the two roles must trade with each other.
func miniConfig() economysim.Config {
	world := economysim.DefaultWorldParams()
	world.Upkeep = []economysim.Item{{Commodity: "grain", Quantity: 1}, {Commodity: "cloth", Quantity: 1}}
	var commodities []economysim.Commodity
	var recipes []economysim.Recipe
	var roles []economysim.Role
	for _, p := range []struct{ good, recipe, role string }{{"grain", "farm", "farmer"}, {"cloth", "weave", "weaver"}} {
		c := economysim.DefaultCommodity()
		c.ID, c.ReferencePrice = p.good, 1_000
		commodities = append(commodities, c)
		r := economysim.DefaultRecipe()
		r.ID = p.recipe
		r.Outputs = []economysim.Output{{Commodity: p.good, Quantity: 4, CostShare: 10_000}}
		recipes = append(recipes, r)
		k := economysim.DefaultRole()
		k.ID, k.Recipes = p.role, []string{p.recipe}
		roles = append(roles, k)
	}
	var regions []economysim.Region
	var groups []economysim.Group
	for _, id := range []string{"north", "south"} {
		r := economysim.DefaultRegion()
		r.ID, r.ReferenceBasket = id, 2_000
		regions = append(regions, r)
		for _, k := range roles {
			groups = append(groups, economysim.Group{Region: id, Role: k.ID, Households: 10})
		}
	}
	return economysim.Config{World: world, Calendar: economysim.DefaultCalendar(), Commodities: commodities,
		Recipes: recipes, Roles: roles, Regions: regions, Groups: groups}
}
