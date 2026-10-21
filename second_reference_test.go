package economysim_test

import (
	"os"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// secondConfig returns the second setting, pack `second`
// (testdata/chains/second/second.toml and warm-second.toml), built in Go. It is
// the Go half of the file-and-Go agreement of the configuration-only chain test
// (X4, FR-CFG-004). Like referenceConfig it starts every entity from its
// Default constructor and lists entities in ID order and arrays in file order.
// Each call returns a fresh value.
func secondConfig() economysim.Config {
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
	role := func(id string, recipes ...string) economysim.Role {
		r := economysim.DefaultRole()
		r.ID, r.Recipes = id, recipes
		return r
	}
	output := func(commodity string, q economysim.Quantity, share int64, discard bool) economysim.Output {
		return economysim.Output{Commodity: commodity, Quantity: q, CostShare: share, Discard: discard}
	}

	world := economysim.DefaultWorldParams()
	world.Upkeep = []economysim.Item{item("biomass", 2), item("water", 1)}

	cal := economysim.DefaultCalendar()
	cal.Seasons = []economysim.Season{{ID: "lit", Days: 184}, {ID: "dark", Days: 181}}
	cal.SeasonStart = 59

	region := economysim.DefaultRegion()
	region.ID, region.ReferenceBasket = "crater_rim", 1000

	return economysim.Config{
		World:    world,
		Calendar: cal,
		Commodities: []economysim.Commodity{
			commodity("biomass", 327, func(c *economysim.Commodity) { c.ShelfLife = 60 }),
			commodity("board", 12364, nil),
			commodity("drill", 14330, func(c *economysim.Commodity) { c.Durability = 500 }),
			commodity("fabricator", 41155, func(c *economysim.Commodity) { c.Durability = 1000 }),
			commodity("ice", 112, func(c *economysim.Commodity) { c.Raw = true }),
			commodity("regolith", 66, func(c *economysim.Commodity) { c.Raw = true }),
			commodity("slag", 50, nil),
			commodity("solder", 2097, nil),
			commodity("wafer", 2163, nil),
			commodity("water", 346, nil),
		},
		Recipes: []economysim.Recipe{
			recipe("cut_ice", func(r *economysim.Recipe) {
				r.Sited = true
				r.Outputs = []economysim.Output{output("ice", 30, 10_000, false)}
				r.Requires = []economysim.Tool{{Commodity: "drill", WearPerUse: 2}}
			}),
			recipe("dig_regolith", func(r *economysim.Recipe) {
				r.Outputs = []economysim.Output{output("regolith", 30, 10_000, false)}
			}),
			recipe("fab_board", func(r *economysim.Recipe) {
				r.Duration = 3
				r.Inputs = []economysim.Item{item("wafer", 2), item("solder", 1)}
				r.Outputs = []economysim.Output{output("board", 1, 10_000, false), output("slag", 1, 0, true)}
				r.Requires = []economysim.Tool{{Commodity: "fabricator", WearPerUse: 1}}
			}),
			recipe("grow_algae", func(r *economysim.Recipe) {
				r.Duration, r.Sited = 12, true
				r.Inputs = []economysim.Item{item("water", 4)}
				r.Outputs = []economysim.Output{output("biomass", 120, 10_000, false)}
			}),
			recipe("make_drill", func(r *economysim.Recipe) {
				r.Inputs = []economysim.Item{item("board", 1)}
				r.Outputs = []economysim.Output{output("drill", 1, 10_000, false)}
			}),
			recipe("make_fabricator", func(r *economysim.Recipe) {
				r.Inputs = []economysim.Item{item("board", 3), item("solder", 1)}
				r.Outputs = []economysim.Output{output("fabricator", 1, 10_000, false)}
			}),
			recipe("melt", func(r *economysim.Recipe) {
				r.Inputs = []economysim.Item{item("ice", 1)}
				r.Outputs = []economysim.Output{output("water", 6, 10_000, false)}
			}),
			recipe("refine_solder", func(r *economysim.Recipe) {
				r.Inputs = []economysim.Item{item("regolith", 2)}
				r.Outputs = []economysim.Output{output("solder", 1, 10_000, false)}
			}),
			recipe("refine_wafer", func(r *economysim.Recipe) {
				r.Inputs = []economysim.Item{item("regolith", 3)}
				r.Outputs = []economysim.Output{output("wafer", 1, 10_000, false)}
			}),
			recipe("salvage", func(r *economysim.Recipe) {
				r.Duration = 50
				r.Outputs = []economysim.Output{output("fabricator", 1, 10_000, false)}
			}),
		},
		Yields: []economysim.YieldProfile{
			{Region: "crater_rim", Recipe: "cut_ice", HasAll: true, All: 6_000},
			{Region: "crater_rim", Recipe: "grow_algae", Seasons: []economysim.SeasonYield{{Season: "dark", BP: 2_500}, {Season: "lit", BP: 10_000}}},
		},
		Roles: []economysim.Role{
			role("algae_tender", "grow_algae"),
			role("driller", "cut_ice", "melt"),
			role("fabber", "fab_board", "make_drill", "make_fabricator", "salvage"),
			role("regolith_miner", "dig_regolith"),
			role("smelter", "refine_wafer", "refine_solder"),
		},
		Regions: []economysim.Region{region},
		Groups: []economysim.Group{
			{Region: "crater_rim", Role: "algae_tender", Households: 600},
			{Region: "crater_rim", Role: "driller", Households: 385},
			{Region: "crater_rim", Role: "fabber", Households: 5},
			{Region: "crater_rim", Role: "regolith_miner", Households: 5},
			{Region: "crater_rim", Role: "smelter", Households: 5},
		},
		WarmStart: &economysim.WarmStart{
			Prices: []economysim.SeedPrice{
				{Region: "crater_rim", Commodity: "biomass", Price: 327},
				{Region: "crater_rim", Commodity: "board", Price: 12364},
				{Region: "crater_rim", Commodity: "drill", Price: 14330},
				{Region: "crater_rim", Commodity: "fabricator", Price: 41155},
				{Region: "crater_rim", Commodity: "ice", Price: 112},
				{Region: "crater_rim", Commodity: "regolith", Price: 66},
				{Region: "crater_rim", Commodity: "solder", Price: 2097},
				{Region: "crater_rim", Commodity: "wafer", Price: 2163},
				{Region: "crater_rim", Commodity: "water", Price: 346},
			},
			Wages: []economysim.SeedWage{{Region: "crater_rim", Wage: 1966}},
			Groups: []economysim.WarmGroup{
				{Region: "crater_rim", Role: "algae_tender", Money: 23384,
					Stock: []economysim.Item{item("biomass", 10), item("water", 5)},
					Jobs:  []economysim.JobSpec{{Recipe: "grow_algae", InProgressBP: 5232}}},
				{Region: "crater_rim", Role: "driller", Money: 11112,
					Stock: []economysim.Item{item("biomass", 10), item("drill", 1), item("ice", 1), item("water", 5)}},
				{Region: "crater_rim", Role: "fabber", Money: 19423,
					Stock: []economysim.Item{item("biomass", 10), item("board", 4), item("fabricator", 1), item("solder", 2), item("wafer", 2), item("water", 5)},
					Jobs:  []economysim.JobSpec{{Recipe: "fab_board", InProgressBP: 246}}},
				{Region: "crater_rim", Role: "regolith_miner", Money: 11000,
					Stock: []economysim.Item{item("biomass", 10), item("water", 5)}},
				{Region: "crater_rim", Role: "smelter", Money: 11198,
					Stock: []economysim.Item{item("biomass", 10), item("regolith", 5), item("water", 5)}},
			},
		},
	}
}

// secondFromFiles loads pack `second` from its two layers.
func secondFromFiles(t testing.TB) economysim.Config {
	t.Helper()
	cfg, err := config.LoadFS(os.DirFS(chainsDir), "second/second.toml", "second/warm-second.toml")
	if err != nil {
		t.Fatalf("LoadFS(second) error = %v, want nil", err)
	}
	return cfg
}

// TestConfig_SecondFileAndGoAgree verifies that pack second loaded from its two
// layers is, field for field, the Go-built secondConfig, and that the two give
// the same hash: the second setting is data, and the Go builder is only another
// way to write it (X4: the test builds second both ways).
//
// Requirement: FR-CFG-004, FR-REF-031
//
// Guard: a field of the second pack that the file sets and the Go builder does
// not (or the reverse) would hide a default that moved.
func TestConfig_SecondFileAndGoAgree(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	file, goV := reflect.ValueOf(secondFromFiles(t)), reflect.ValueOf(secondConfig())
	for i := range file.NumField() {
		name := file.Type().Field(i).Name
		if !reflect.DeepEqual(file.Field(i).Interface(), goV.Field(i).Interface()) {
			t.Errorf("Config.%s from the file = %+v, want the Go build %+v", name, file.Field(i).Interface(), goV.Field(i).Interface())
		}
	}
	if got, want := economysim.HashConfig(secondFromFiles(t)), economysim.HashConfig(secondConfig()); got != want {
		t.Errorf("HashConfig(file) = %x, want %x (HashConfig of secondConfig)", got, want)
	}
	if err := secondConfig().Validate(); err != nil {
		t.Errorf("secondConfig().Validate() = %v, want nil", err)
	}
}
