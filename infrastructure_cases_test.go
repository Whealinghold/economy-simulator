package economysim_test

import (
	"strconv"

	economysim "github.com/Whealinghold/economy-sim"
)

// facilityCase builds a failing case from infrastructureConfig: the field of the
// parameter-owner table, what breaks, the rule, the one path, the value and the
// bound, and the change to the valid configuration that breaks it.
func facilityCase(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config)) fieldCase {
	return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
		mutate: func(c *economysim.Config) {
			*c = infrastructureConfig()
			set(c)
		}}
}

// facilityClassPath is the path of a field of the class with the ID.
func facilityClassPath(id string, key ...string) []string {
	return append([]string{"modules", "infrastructure", "class", id}, key...)
}

// facilityRowPathT is the path of a field of the i-th initial row, in the order written.
func facilityRowPathT(i int, key ...string) []string {
	return append([]string{"modules", "infrastructure", "initial[" + strconv.Itoa(i) + "]"}, key...)
}

// facilityRecipePath is the path of a field of a recipe.
func facilityRecipePath(id string, key ...string) []string {
	return append([]string{"recipes", id}, key...)
}

// facilityOf returns the class of c with the ID.
func facilityOf(c *economysim.Config, id string) *economysim.FacilityClass {
	for i := range c.Modules.Infrastructure.Classes {
		if c.Modules.Infrastructure.Classes[i].ID == id {
			return &c.Modules.Infrastructure.Classes[i]
		}
	}
	return nil
}

// dropClauseRecipes removes the three recipes with a clause and their entries in the
// grower's list, so a case that changes the classes breaks no reference of theirs.
func dropClauseRecipes(c *economysim.Config) {
	keep := c.Recipes[:0]
	for _, r := range c.Recipes {
		if r.ID != "build_mill" && r.ID != "repair_mill" && r.ID != "grind" {
			keep = append(keep, r)
		}
	}
	c.Recipes = keep
	g := roleOf(c, "grower")
	g.Recipes = g.Recipes[:len(g.Recipes)-3]
}

// infrastructureFieldCases are the failing cases of the rows of the infrastructure
// module in the parameter-owner table (spec-infrastructure.md 12, 13.1, FR-CFG-031):
// each breaks one field of the valid configuration of infrastructureConfig.
// buildFieldCases appends them to the table that the registry test reads.
func infrastructureFieldCases() []fieldCase {
	var cs []fieldCase
	cs = append(cs, facilityModuleCases()...)
	cs = append(cs, facilityClassCases()...)
	cs = append(cs, facilityKindCases()...)
	cs = append(cs, facilityRowCases()...)
	cs = append(cs, facilityHolderCases()...)
	return append(cs, facilityClauseCases()...)
}

// facilityModuleCases are the cases of the module's two lists.
func facilityModuleCases() []fieldCase {
	return []fieldCase{
		facilityCase("Infrastructure.Classes", "more than MaxFacilityClasses", economysim.ErrOutOfBounds,
			[]string{"modules", "infrastructure", "classes"}, "65", "64", func(c *economysim.Config) {
				c.Modules.Infrastructure.Classes = make([]economysim.FacilityClass, economysim.MaxFacilityClasses+1)
			}),
		facilityCase("Infrastructure.Initial", "more than MaxInitialRows", economysim.ErrOutOfBounds,
			[]string{"modules", "infrastructure", "initial"}, "1025", "1024", func(c *economysim.Config) {
				c.Modules.Infrastructure.Initial = make([]economysim.InitialFacility, economysim.MaxInitialRows+1)
			}),
		facilityCase("InitialFacility.Class", "two rows with one key", economysim.ErrDuplicateID, facilityRowPathT(4), "drain", "",
			func(c *economysim.Config) {
				m := c.Modules.Infrastructure
				m.Initial = append(m.Initial, m.Initial[1])
			}),
		facilityCase("Infrastructure.Initial", "rows above MaxFacilities in all", economysim.ErrFacilityCount,
			[]string{"modules", "infrastructure", "initial"}, "101000", "100000", func(c *economysim.Config) {
				dropClauseRecipes(c)
				m := c.Modules.Infrastructure
				m.Classes, m.Initial = nil, nil
				for i := range economysim.MaxFacilityClasses {
					id := "s" + strconv.Itoa(i)
					m.Classes = append(m.Classes, economysim.FacilityClass{ID: id, Kind: "site", Durability: 10, Capacity: 1})
				}
				for i := range 101 {
					region := []string{"north", "south"}[i/economysim.MaxFacilityClasses] // 101 distinct keys over 64 classes
					m.Initial = append(m.Initial, economysim.InitialFacility{Region: region, Class: "s" + strconv.Itoa(i%64), Count: 1_000})
				}
			}),
	}
}

// facilityClassCases are the cases of the ID, kind and numbers of a class.
func facilityClassCases() []fieldCase {
	bounds := economysim.ErrOutOfBounds
	return []fieldCase{
		facilityCase("FacilityClass.ID", "not an ID", economysim.ErrInvalidID, []string{"modules", "infrastructure", "classes[2]", "id"}, "Bad ID", "",
			func(c *economysim.Config) { facilityOf(c, "mill").ID = "Bad ID" }),
		facilityCase("FacilityClass.ID", "twice", economysim.ErrDuplicateID, facilityClassPath("mill", "id"), "mill", "",
			func(c *economysim.Config) { facilityOf(c, "road").ID = "mill" }),
		facilityCase("FacilityClass.Kind", "not one of the four", bounds, facilityClassPath("road", "kind"), "bridge", "",
			func(c *economysim.Config) { facilityOf(c, "road").Kind = "bridge" }),
		facilityCase("FacilityClass.Durability", "below 2", bounds, facilityClassPath("mill", "durability"), "1", "2",
			func(c *economysim.Config) { facilityOf(c, "mill").Durability = 1 }),
		facilityCase("FacilityClass.Durability", "above MaxDurability", bounds, facilityClassPath("mill", "durability"), "1000001", "1000000",
			func(c *economysim.Config) { facilityOf(c, "mill").Durability = economysim.MaxDurability + 1 }),
		facilityCase("FacilityClass.Decay", "not below the durability", bounds, facilityClassPath("mill", "decay"), "1000", "999",
			func(c *economysim.Config) { facilityOf(c, "mill").Decay = 1_000 }),
		facilityCase("FacilityClass.WearPerUse", "above the durability", bounds, facilityClassPath("mill", "wear_per_use"), "1001", "1000",
			func(c *economysim.Config) { facilityOf(c, "mill").WearPerUse = 1_001 }),
		facilityCase("FacilityClass.WearPerUse", "on a route", economysim.ErrFacilityShape, facilityClassPath("road", "wear_per_use"), "1", "0",
			func(c *economysim.Config) { facilityOf(c, "road").WearPerUse = 1 }),
		facilityCase("FacilityClass.Capacity", "above MaxFacilityCapacity", bounds, facilityClassPath("mill", "capacity"), "100001", "100000",
			func(c *economysim.Config) { facilityOf(c, "mill").Capacity = economysim.MaxFacilityCapacity + 1 }),
		facilityCase("FacilityClass.Capacity", "0 on a site", economysim.ErrFacilityShape, facilityClassPath("mill", "capacity"), "0", "1",
			func(c *economysim.Config) { facilityOf(c, "mill").Capacity = 0 }),
		facilityCase("FacilityClass.Capacity", "on a ground", economysim.ErrFacilityShape, facilityClassPath("drain", "capacity"), "3", "0",
			func(c *economysim.Config) { facilityOf(c, "drain").Capacity = 3 }),
	}
}

// facilityKindCases are the cases of the fields a kind uses and the names they hold.
func facilityKindCases() []fieldCase {
	shape := economysim.ErrFacilityShape
	return []fieldCase{
		facilityCase("FacilityClass.BoostBP", "above MaxBoostBP", economysim.ErrOutOfBounds, facilityClassPath("road", "boost_bp"), "100001", "100000",
			func(c *economysim.Config) { facilityOf(c, "road").BoostBP = economysim.MaxBoostBP + 1 }),
		facilityCase("FacilityClass.BoostBP", "0 on a route", shape, facilityClassPath("road", "boost_bp"), "0", "1",
			func(c *economysim.Config) { facilityOf(c, "road").BoostBP = 0 }),
		facilityCase("FacilityClass.BoostBP", "on a site", shape, facilityClassPath("mill", "boost_bp"), "5", "0",
			func(c *economysim.Config) { facilityOf(c, "mill").BoostBP = 5 }),
		facilityCase("FacilityClass.Modes", "more than MaxRouteModes", economysim.ErrOutOfBounds, facilityClassPath("road", "modes"), "9", "8",
			func(c *economysim.Config) { facilityOf(c, "road").Modes = make([]string, economysim.MaxRouteModes+1) }),
		facilityCase("FacilityClass.Modes", "none on a route", shape, facilityClassPath("road", "modes"), "0", "1",
			func(c *economysim.Config) { facilityOf(c, "road").Modes = nil }),
		facilityCase("FacilityClass.Modes", "a mode that is not defined", economysim.ErrUnknownMode, facilityClassPath("road", "modes[0]"), "nowhere", "",
			func(c *economysim.Config) { facilityOf(c, "road").Modes = []string{"nowhere"} }),
		facilityCase("FacilityClass.Resource", "a resource that is not defined", economysim.ErrUnknownResource, facilityClassPath("drain", "resource"), "nowhere", "",
			func(c *economysim.Config) { facilityOf(c, "drain").Resource = "nowhere" }),
		facilityCase("FacilityClass.Resource", "none on a ground", shape, facilityClassPath("drain", "resource"), "", "1",
			func(c *economysim.Config) { facilityOf(c, "drain").Resource = "" }),
		facilityCase("FacilityClass.Store", "a store class that is not defined", economysim.ErrUnknownStoreClass, facilityClassPath("barn", "store"), "nowhere", "",
			func(c *economysim.Config) { facilityOf(c, "barn").Store = "nowhere" }),
		facilityCase("FacilityClass.Store", "none on a store", shape, facilityClassPath("barn", "store"), "", "1",
			func(c *economysim.Config) { facilityOf(c, "barn").Store = "" }),
		facilityCase("FacilityClass.Needs", "a flag with the settlements module off", economysim.ErrModuleDependency, facilityClassPath("mill", "needs"), "fair", "settlements",
			func(c *economysim.Config) { facilityOf(c, "mill").Needs = "fair" }),
	}
}

// facilityRowCases are the cases of the fields of an initial row.
func facilityRowCases() []fieldCase {
	shape := economysim.ErrFacilityShape
	return []fieldCase{
		facilityCase("InitialFacility.Region", "a region that is not defined", economysim.ErrUnknownRegion, facilityRowPathT(1, "region"), "nowhere", "",
			func(c *economysim.Config) { c.Modules.Infrastructure.Initial[1].Region = "nowhere" }),
		facilityCase("InitialFacility.Class", "a class that is not defined", economysim.ErrUnknownFacilityClass, facilityRowPathT(1, "class"), "nowhere", "",
			func(c *economysim.Config) { c.Modules.Infrastructure.Initial[1].Class = "nowhere" }),
		facilityCase("InitialFacility.Toward", "the row's own region", economysim.ErrSameRegion, facilityRowPathT(3, "toward"), "north", "",
			func(c *economysim.Config) { c.Modules.Infrastructure.Initial[3].Toward = "north" }),
		facilityCase("InitialFacility.Toward", "on a site", shape, facilityRowPathT(2, "toward"), "south", "",
			func(c *economysim.Config) { c.Modules.Infrastructure.Initial[2].Toward = "south" }),
		facilityCase("InitialFacility.Toward", "a region with no edge to the row's", economysim.ErrUnknownEdge, facilityRowPathT(3, "toward"), "east", "north",
			func(c *economysim.Config) {
				east := economysim.DefaultRegion()
				east.ID = "east"
				c.Regions = append(c.Regions, east)
				c.Edges = append(c.Edges, economysim.Edge{From: "south", To: "east", Distance: 40}, economysim.Edge{From: "east", To: "south", Distance: 40})
				c.Modules.Infrastructure.Initial[3].Toward = "east"
			}),
		facilityCase("InitialFacility.Role", "a role that is not defined", economysim.ErrUnknownRole, facilityRowPathT(2, "role"), "nowhere", "",
			func(c *economysim.Config) { c.Modules.Infrastructure.Initial[2].Role = "nowhere" }),
		facilityCase("InitialFacility.Role", "none on a store row", shape, facilityRowPathT(0, "role"), "", "1",
			func(c *economysim.Config) { c.Modules.Infrastructure.Initial[0].Role = "" }),
		facilityCase("InitialFacility.Count", "0", economysim.ErrOutOfBounds, facilityRowPathT(1, "count"), "0", "1",
			func(c *economysim.Config) { c.Modules.Infrastructure.Initial[1].Count = 0 }),
		facilityCase("InitialFacility.Count", "above MaxInitialCount", economysim.ErrOutOfBounds, facilityRowPathT(1, "count"), "1001", "1000",
			func(c *economysim.Config) { c.Modules.Infrastructure.Initial[1].Count = economysim.MaxInitialCount + 1 }),
	}
}

// facilityHolderCases are the cases of the rows that have no holder or too many.
func facilityHolderCases() []fieldCase {
	return []fieldCase{
		facilityCase("InitialFacility.Role", "a role with no group in the region", economysim.ErrFacilityHolder, facilityRowPathT(2, "role"), "visitor", "north",
			func(c *economysim.Config) { c.Modules.Infrastructure.Initial[2].Role = "visitor" }),
		facilityCase("InitialFacility.Role", "a store row whose role has no base room", economysim.ErrFacilityHolder, facilityRowPathT(0, "role"), "cook", "north",
			func(c *economysim.Config) { c.Modules.Infrastructure.Initial[0].Role = "cook" }),
		facilityCase("InitialFacility.Count", "more than an owner may hold", economysim.ErrFacilityCount, facilityRowPathT(2, "count"), "101", "64",
			func(c *economysim.Config) { c.Modules.Infrastructure.Initial[2].Count = 1_000 }),
		facilityCase("FacilityClass.Kind", "a route with the transport module off", economysim.ErrModuleDependency, facilityClassPath("road", "kind"), "route", "transport",
			func(c *economysim.Config) { c.Modules.Transport = nil }),
		facilityCase("FacilityClass.Kind", "a ground with the resources module off", economysim.ErrModuleDependency, facilityClassPath("drain", "kind"), "ground", "resources",
			func(c *economysim.Config) {
				c.Modules.Resources = nil
				recipeOf(c, "grow_staple").Draws, recipeOf(c, "cut_fuel").Draws = nil, nil
			}),
		facilityCase("FacilityClass.Kind", "a store with the storage module off", economysim.ErrModuleDependency, facilityClassPath("barn", "kind"), "store", "storage",
			func(c *economysim.Config) { c.Modules.Storage = nil }),
	}
}

// facilityClauseCases are the cases of the fields of the recipe clauses.
func facilityClauseCases() []fieldCase {
	shape, bounds := economysim.ErrFacilityShape, economysim.ErrOutOfBounds
	return []fieldCase{
		facilityCase("Recipe.Needs", "more than MaxRecipeNeeds", bounds, facilityRecipePath("grind", "needs"), "3", "2",
			func(c *economysim.Config) { recipeOf(c, "grind").Needs = []string{"mill", "mill", "mill"} }),
		facilityCase("Recipe.Needs", "a class twice", economysim.ErrDuplicateID, facilityRecipePath("grind", "needs"), "mill", "",
			func(c *economysim.Config) { recipeOf(c, "grind").Needs = []string{"mill", "mill"} }),
		facilityCase("Recipe.Needs", "a class that is not a site", shape, facilityRecipePath("grind", "needs"), "road", "site",
			func(c *economysim.Config) { recipeOf(c, "grind").Needs = []string{"road"} }),
		facilityCase("Recipe.Needs", "a class that is not defined", economysim.ErrUnknownFacilityClass, facilityRecipePath("grind", "needs"), "nowhere", "",
			func(c *economysim.Config) { recipeOf(c, "grind").Needs = []string{"nowhere"} }),
		facilityCase("Recipe.Needs", "with the module off", economysim.ErrModuleDependency, facilityRecipePath("grind", "needs"), "1", "infrastructure",
			func(c *economysim.Config) {
				dropClauseRecipes(c)
				r := economysim.DefaultRecipe()
				r.ID, r.Needs = "grind", []string{"mill"}
				r.Inputs = []economysim.Item{{Commodity: "seed", Quantity: 1}}
				r.Outputs = []economysim.Output{{Commodity: "staple", Quantity: 2, CostShare: 10_000}}
				c.Recipes = append(c.Recipes, r)
				c.Modules.Infrastructure = nil
			}),
		facilityCase("Recipe.Builds", "together with Repairs", shape, facilityRecipePath("build_mill"), "builds and repairs", "one",
			func(c *economysim.Config) {
				recipeOf(c, "build_mill").Repairs = &economysim.FacilityRepair{Class: "mill", Units: 5}
			}),
		facilityCase("Recipe.Builds", "with an output that is not a discard", shape, facilityRecipePath("build_mill", "outputs", "staple"), "staple", "discard",
			func(c *economysim.Config) {
				recipeOf(c, "build_mill").Outputs = []economysim.Output{{Commodity: "staple", Quantity: 1, CostShare: 10_000}}
			}),
		facilityCase("Recipe.Builds", "with no input", economysim.ErrFacilityNoCost, facilityRecipePath("build_mill", "inputs"), "0", "1",
			func(c *economysim.Config) { recipeOf(c, "build_mill").Inputs = nil }),
		facilityCase("Recipe.Repairs", "with no input", economysim.ErrFacilityNoCost, facilityRecipePath("repair_mill", "inputs"), "0", "1",
			func(c *economysim.Config) { recipeOf(c, "repair_mill").Inputs = nil }),
		facilityCase("Recipe.Repairs", "on a trainee-eligible recipe", economysim.ErrEligibleHasTool, facilityRecipePath("repair_mill", "trainee_eligible"), "true", "false",
			func(c *economysim.Config) { recipeOf(c, "repair_mill").TraineeEligible = true }),
		facilityCase("FacilityBuild.Class", "a class that is not defined", economysim.ErrUnknownFacilityClass, facilityRecipePath("build_mill", "builds", "class"), "nowhere", "",
			func(c *economysim.Config) { recipeOf(c, "build_mill").Builds.Class = "nowhere" }),
		facilityCase("FacilityBuild.Toward", "on a class that is not a route", shape, facilityRecipePath("build_mill", "builds", "toward"), "south", "",
			func(c *economysim.Config) { recipeOf(c, "build_mill").Builds.Toward = "south" }),
		facilityCase("FacilityBuild.Toward", "a region that is not defined", economysim.ErrUnknownRegion, facilityRecipePath("build_mill", "builds", "toward"), "nowhere", "",
			func(c *economysim.Config) {
				recipeOf(c, "build_mill").Builds = &economysim.FacilityBuild{Class: "road", Toward: "nowhere"}
			}),
		facilityCase("FacilityRepair.Class", "a class that is not defined", economysim.ErrUnknownFacilityClass, facilityRecipePath("repair_mill", "repairs", "class"), "nowhere", "",
			func(c *economysim.Config) { recipeOf(c, "repair_mill").Repairs.Class = "nowhere" }),
		facilityCase("FacilityRepair.Units", "0", bounds, facilityRecipePath("repair_mill", "repairs", "units"), "0", "1",
			func(c *economysim.Config) { recipeOf(c, "repair_mill").Repairs.Units = 0 }),
		facilityCase("FacilityRepair.Units", "above the durability of the class", bounds, facilityRecipePath("repair_mill", "repairs", "units"), "1001", "1000",
			func(c *economysim.Config) { recipeOf(c, "repair_mill").Repairs.Units = 1_001 }),
		facilityCase("FacilityClass.Label", "longer than MaxLabelBytes", bounds, facilityClassPath("mill", "label"), "257", "256",
			func(c *economysim.Config) {
				facilityOf(c, "mill").Label = string(make([]byte, economysim.MaxLabelBytes+1))
			}),
	}
}
