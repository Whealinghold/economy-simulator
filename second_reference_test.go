package economysim_test

import (
	"fmt"
	"os"
	"reflect"
	"slices"
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

// secondResources is example B of spec-resources.md 16.2 as a document over pack
// second, which is that example's colony: a deposit of ice that does not renew
// (with a curve of poorer grades as it empties) behind cut_ice, and culture vats (a
// refill of 200% a year held at half their capacity) behind grow_algae. The
// capacities are the example's, the horizon is its 15 years.
const secondResources = `
[modules.resources]
horizon_years = 15

[modules.resources.resource.ice_field]
abundance = [ { share = 0, factor = 2000 }, { share = 2000, factor = 8000 }, { share = 10000, factor = 10000 } ]

[modules.resources.resource.vat_volume]
shape = "refill"
rate_bp = 20000
plan_share_bp = 5000

[modules.resources.endowment.crater_rim.ice_field]
capacity = 3000000
[modules.resources.endowment.crater_rim.vat_volume]
capacity = 30000

[recipe.cut_ice]
draws = [ { resource = "ice_field", quantity = 30 } ]
[recipe.grow_algae]
draws = [ { resource = "vat_volume", quantity = 1 } ]
`

// TestConfigOnly_SecondSettingWithResources verifies that the second setting, the
// crater colony, takes the resources module from a document alone (example B of
// spec-resources.md 16.2, the same format as the farm of the first consumer with
// other words): it loads, validates, warns raw_source_exhaustible (ice is raw and
// cut_ice, which draws a finite deposit, is its only source), runs with the audit
// on and holds the invariants of testing.md 6.3 on every seed, and the deposit is
// drawn down.
//
// Requirement: FR-RES-049, FR-RES-033, FR-RES-001
//
// Guard: no Go code names ice, vats or the colony; a branch on a resource or
// recipe identifier would make this fail or need a change here (architecture.md
// 9.1, 9.4).
func TestConfigOnly_SecondSettingWithResources(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPackBytes(t, []byte(secondResources), packSecond...)
	var warned bool
	for _, w := range cfg.Warnings() {
		warned = warned || (w.Rule == "raw_source_exhaustible" && len(w.Path) == 2 && w.Path[1] == "ice")
	}
	if !warned {
		t.Errorf("Warnings() = %+v, want raw_source_exhaustible for the commodity ice (a raw good whose only source draws a finite stock)", cfg.Warnings())
	}
	for _, seed := range seedList(min(2, scenarioSeedCount())) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			runInvariants(t, cfg, seed, 120, nil)
			w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed))
			if err != nil {
				t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
			}
			start := w.Endowment(0, 0)
			for range 60 {
				if err := w.Step(); err != nil {
					t.Fatalf("seed %d: Step() error = %v, want nil", seed, err)
				}
			}
			if got := w.Endowment(0, 0); got >= start {
				t.Errorf("seed %d: the ice deposit after 60 ticks = %d from %d, want it drawn down", seed, got, start)
			}
		})
	}
}

// TestInvariants_ResourcesOn verifies the invariants of testing.md 6.3 over pack
// one with the resources module on and the binding overlay (the stocks sized to
// 1.1 times the draw of the reference programme), seeds 1 to scenarioSeedCount(),
// 200 ticks: goods and money are conserved with exact
// equality, since a draw is not a ledger flow, and no balance or stock is
// negative.
//
// Requirement: FR-RES-024, FR-REF-028
//
// Guard: a draw that created goods or money from the ground would break
// conservation; the table row of TestInvariants_Packs covers the slack overlay.
func TestInvariants_ResourcesOn(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, append(slices.Clone(packOne), "overlay/resources-binding.toml")...)
	for _, seed := range seedList(invariantSeeds(scenarioSeeds)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			runInvariants(t, cfg, seed, 200, nil)
		})
	}
}
