package economysim_test

import (
	economysim "github.com/Whealinghold/economy-sim"
)

// referenceConfig returns the reference configuration: the pack "one" of
// spec-reference-chain.md built in Go (the layers testdata/chains/base.toml,
// one.toml and warm-one.toml), with no module on and a warm start present.
//
// It is the shared fixture of the configuration tests: the hash tests pin its
// hash, the validation tests of Phase 4 start every failing case from it and
// break one field, and the loader's "file and Go agree" test of Phase 5
// compares the pack loaded from those three files with it.
//
// It is written the way the loader produces it: each entity starts from its
// Default constructor, then sets what the file sets; entities are in ID order
// (a table of a file is a map, so a loader sorts it) and the lists a file
// writes as arrays (an upkeep, the inputs of a recipe, the stock of a group,
// the recipes of a role) are in the order of the file. Canonical order
// (spec-config.md 3.4) is applied by HashConfig and by a world, not here. It has
// no Pools row, because the file has no pool key (WarmPool.Money defaults to 0),
// and no Label, because the packs have none.
//
// Each call returns a fresh value that the caller may change.
func referenceConfig() economysim.Config {
	item := func(commodity string, q economysim.Quantity) economysim.Item {
		return economysim.Item{Commodity: commodity, Quantity: q}
	}
	commodity := func(id string, price economysim.Price, mutate func(*economysim.Commodity)) economysim.Commodity {
		c := economysim.DefaultCommodity()
		c.ID, c.ReferencePrice = id, price
		if mutate != nil {
			mutate(&c)
		}
		return c
	}
	recipe := func(id string, mutate func(*economysim.Recipe)) economysim.Recipe {
		r := economysim.DefaultRecipe()
		r.ID = id
		mutate(&r)
		return r
	}
	role := func(id string, mutate func(*economysim.Role)) economysim.Role {
		r := economysim.DefaultRole()
		r.ID = id
		mutate(&r)
		return r
	}
	// output is a recipe output with the cost share the pack writes.
	output := func(commodity string, q economysim.Quantity, share int64, discard bool) economysim.Output {
		return economysim.Output{Commodity: commodity, Quantity: q, CostShare: share, Discard: discard}
	}
	// The start stock every role of the pack shares, then what each adds.
	baseStock := func(extra ...economysim.Item) []economysim.Item {
		return append([]economysim.Item{item("fuel", 5), item("prepared", 4)}, extra...)
	}

	world := economysim.DefaultWorldParams()
	world.Upkeep = []economysim.Item{item("staple", 2), item("prepared", 1), item("fuel", 1)}

	region := economysim.DefaultRegion()
	region.ID = "north"

	return economysim.Config{
		World:    world,
		Calendar: economysim.DefaultCalendar(),
		Commodities: []economysim.Commodity{
			commodity("fuel", 179, func(c *economysim.Commodity) { c.Raw, c.Classes = true, []string{"fuel"} }),
			commodity("mineral", 837, func(c *economysim.Commodity) { c.Raw, c.Classes = true, []string{"material"} }),
			commodity("prepared", 628, func(c *economysim.Commodity) { c.ShelfLife, c.Classes = 4, []string{"food"} }),
			commodity("refined", 3190, func(c *economysim.Commodity) { c.Classes = []string{"material"} }),
			commodity("seed", 222, nil),
			commodity("staple", 96, func(c *economysim.Commodity) { c.ShelfLife, c.Classes = 700, []string{"food"} }),
			commodity("tool_a", 13915, func(c *economysim.Commodity) { c.Durability, c.Classes = 10, []string{"tool"} }),
			commodity("tool_b", 17105, func(c *economysim.Commodity) { c.Durability, c.Classes = 8, []string{"tool"} }),
			commodity("waste", 50, nil),
		},
		Recipes: []economysim.Recipe{
			recipe("cut_fuel", func(r *economysim.Recipe) {
				r.Sited = true
				r.Outputs = []economysim.Output{output("fuel", 14, 10_000, false)}
			}),
			recipe("dig_mineral", func(r *economysim.Recipe) {
				r.Sited = true
				r.Outputs = []economysim.Output{output("mineral", 3, 10_000, false)}
			}),
			recipe("gather_seed", func(r *economysim.Recipe) {
				r.Sited = true
				r.Outputs = []economysim.Output{output("seed", 2, 10_000, false)}
			}),
			recipe("grow_staple", func(r *economysim.Recipe) {
				r.Duration, r.Sited = 60, true
				r.Inputs = []economysim.Item{item("seed", 40)}
				r.Outputs = []economysim.Output{output("staple", 1800, 10_000, false)}
				r.Requires = []economysim.Tool{{Commodity: "tool_a", WearPerUse: 10}}
			}),
			recipe("make_tool_a", func(r *economysim.Recipe) {
				r.Duration = 3
				r.Inputs = []economysim.Item{item("refined", 2)}
				r.Outputs = []economysim.Output{output("tool_a", 1, 10_000, false)}
			}),
			recipe("make_tool_b", func(r *economysim.Recipe) {
				r.Duration = 3
				r.Inputs = []economysim.Item{item("refined", 3)}
				r.Outputs = []economysim.Output{output("tool_b", 1, 10_000, false)}
			}),
			recipe("prepare", func(r *economysim.Recipe) {
				r.Inputs = []economysim.Item{item("staple", 2), item("fuel", 1)}
				r.Outputs = []economysim.Output{output("prepared", 8, 10_000, false)}
				r.Requires = []economysim.Tool{{Commodity: "tool_b", WearPerUse: 1}}
			}),
			recipe("prepare_basic", func(r *economysim.Recipe) {
				r.TraineeEligible, r.TraineeYield = true, 7_000
				r.Inputs = []economysim.Item{item("staple", 2), item("fuel", 1)}
				r.Outputs = []economysim.Output{output("prepared", 4, 10_000, false)}
			}),
			recipe("refine", func(r *economysim.Recipe) {
				r.Duration = 2
				r.Inputs = []economysim.Item{item("mineral", 5), item("fuel", 2)}
				r.Outputs = []economysim.Output{output("refined", 3, 10_000, false), output("waste", 1, 0, true)}
			}),
			recipe("save_seed", func(r *economysim.Recipe) {
				r.Inputs = []economysim.Item{item("staple", 20)}
				r.Outputs = []economysim.Output{output("seed", 20, 10_000, false)}
			}),
		},
		Yields: []economysim.YieldProfile{
			{Region: "north", Recipe: "cut_fuel", HasAll: true, All: 10_000},
			{Region: "north", Recipe: "dig_mineral", HasAll: true, All: 10_000},
			{Region: "north", Recipe: "gather_seed", HasAll: true, All: 10_000},
			{Region: "north", Recipe: "grow_staple", Seasons: []economysim.SeasonYield{{Season: "spring", BP: 10_000}}},
		},
		Roles: []economysim.Role{
			role("cook", func(r *economysim.Role) {
				r.Recipes = []string{"prepare", "prepare_basic"}
				r.StartStock = baseStock(item("staple", 10), item("tool_b", 1))
			}),
			role("digger", func(r *economysim.Role) {
				r.Unskilled = true
				r.Recipes = []string{"dig_mineral"}
				r.StartStock = baseStock(item("staple", 10))
			}),
			role("forester", func(r *economysim.Role) {
				r.Unskilled = true
				r.Recipes = []string{"cut_fuel"}
				r.StartStock = baseStock(item("staple", 10))
			}),
			role("grower", func(r *economysim.Role) {
				r.Recipes = []string{"grow_staple", "save_seed", "gather_seed"}
				r.StartStock = baseStock(item("seed", 40), item("staple", 10), item("tool_a", 1))
			}),
			role("maker", func(r *economysim.Role) {
				r.Recipes = []string{"make_tool_a", "make_tool_b"}
				r.StartStock = baseStock(item("refined", 5), item("staple", 10))
			}),
			role("refiner", func(r *economysim.Role) {
				r.Recipes = []string{"refine"}
				r.StartStock = []economysim.Item{item("fuel", 5), item("mineral", 5), item("prepared", 4), item("staple", 10)}
			}),
			role("visitor", func(r *economysim.Role) {
				r.Closed = true
				r.Trades = []string{"fuel", "prepared", "staple"}
			}),
		},
		Regions: []economysim.Region{region},
		Groups: []economysim.Group{
			{Region: "north", Role: "cook", Households: 136},
			{Region: "north", Role: "digger", Households: 30},
			{Region: "north", Role: "forester", Households: 90},
			{Region: "north", Role: "grower", Households: 253},
			{Region: "north", Role: "maker", Households: 55},
			{Region: "north", Role: "refiner", Households: 36},
		},
		WarmStart: &economysim.WarmStart{
			Prices: []economysim.SeedPrice{
				{Region: "north", Commodity: "fuel", Price: 179},
				{Region: "north", Commodity: "mineral", Price: 837},
				{Region: "north", Commodity: "prepared", Price: 628},
				{Region: "north", Commodity: "refined", Price: 3190},
				{Region: "north", Commodity: "seed", Price: 222},
				{Region: "north", Commodity: "staple", Price: 96},
				{Region: "north", Commodity: "tool_a", Price: 13915},
				{Region: "north", Commodity: "tool_b", Price: 17105},
			},
			Wages: []economysim.SeedWage{{Region: "north", Wage: 2512}},
			Groups: []economysim.WarmGroup{
				{Region: "north", Role: "cook", Money: 11360,
					Stock: []economysim.Item{item("fuel", 5), item("prepared", 4), item("staple", 10), item("tool_b", 1)}},
				{Region: "north", Role: "digger", Money: 10989,
					Stock: []economysim.Item{item("fuel", 5), item("prepared", 4), item("staple", 10)}},
				{Region: "north", Role: "forester", Money: 10989,
					Stock: []economysim.Item{item("fuel", 5), item("prepared", 4), item("staple", 10)}},
				{Region: "north", Role: "grower", Money: 78810,
					Stock: []economysim.Item{item("fuel", 5), item("prepared", 4), item("seed", 40), item("staple", 664), item("tool_a", 1)}},
				{Region: "north", Role: "maker", Money: 22557,
					Stock: []economysim.Item{item("fuel", 5), item("prepared", 4), item("refined", 5), item("staple", 10)},
					Jobs:  []economysim.JobSpec{{Recipe: "make_tool_a", InProgressBP: 418}, {Recipe: "make_tool_b", InProgressBP: 5113}}},
				{Region: "north", Role: "refiner", Money: 16531,
					Stock: []economysim.Item{item("fuel", 5), item("mineral", 5), item("prepared", 4), item("staple", 10)},
					Jobs:  []economysim.JobSpec{{Recipe: "refine", InProgressBP: 5492}}},
			},
		},
	}
}
