package economysim_test

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode"

	economysim "github.com/Whealinghold/economy-sim"
)

// fieldCase is one configuration that breaks one field of one scope: Validate
// must return errors of the listed rules only, at exactly the paths, in the
// order of path then rule. The field is the row of the parameter-owner table
// ("Scope.Field", spec-config.md 9.2), which the registry test reads to check
// that every built row has a failing case (FR-CFG-031).
type fieldCase struct {
	field  string
	name   string
	rules  []error // the first is the rule the case is for
	mutate func(c *economysim.Config)
	paths  [][]string
	value  string // the Value of the first error, when set
	bound  string // the Bound of the first error, when set
}

// snake returns the snake case of a Go field name: the last segment of a path.
func snake(name string) string {
	var b strings.Builder
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

// setInt sets the integer field of a struct by its Go name.
func setInt(target any, field string, v int64) {
	reflect.ValueOf(target).Elem().FieldByName(field).SetInt(v)
}

// boundCase is a field that is out of its range at bad, with the one error at
// the path of the field.
func boundCase(scope, field string, path []string, mutate func(c *economysim.Config)) fieldCase {
	return fieldCase{
		field: scope + "." + field, name: "outside its range", rules: []error{economysim.ErrOutOfBounds},
		mutate: mutate, paths: [][]string{path},
	}
}

// worldBounds are the integer fields of the world that have a range of their own
// and a value outside it that breaks no other rule: the ranges of
// spec-agents.md 12, spec-beliefs.md 11, spec-spawner.md 11, spec-scale.md and
// spec-welfare-and-wages.md 12.1.
var worldBounds = []struct {
	field string
	bad   int64
}{
	{"Scale", 0}, {"MinAgents", 0}, {"BulkFactor", 0}, {"WelfareWindow", 0}, {"WindowCycles", 0},
	{"CoverTicks", 0}, {"InputJobs", 0}, {"OutputCoverTicks", -1}, {"FavourFloor", 10_001},
	{"ReserveTicks", 1_001}, {"SpendMaxShare", 10_001}, {"SpendHalfTicks", 0}, {"RestBand", 10_001},
	{"ProduceHaltRatio", 9_999}, {"BasketTolerance", 10_001}, {"BeliefStep", 0},
	{"BeliefAdjustProb", 0}, {"BeliefSignificance", 5_001}, {"BeliefStockLow", 0},
	{"BeliefStockHigh", 0}, {"BeliefMaxAge", 1_001}, {"BeliefWidthInit", 0}, {"BeliefWidthMin", 0},
	{"BeliefWidthMax", 10_000}, {"BeliefWidthStep", 0}, {"BeliefEvidenceShare", 0}, {"CostFloor", 0},
	{"CostCeiling", 0}, {"RefBandLow", 0}, {"RefBandHigh", 9_999}, {"PriceWindow", 0},
	{"ReviewTicks", 0}, {"EntryShare", 10_001}, {"RoleHysteresis", -1}, {"RoleCooldown", 3_651},
	{"EntryRatio", -1}, {"ChangePrivation", 0}, {"RemovePrivation", 0}, {"DeathStep", 0},
	{"ExitShare", 10_001}, {"Renewal", 10_001}, {"EntryPoolShare", 10_001},
}

// fieldCases is every failing case by field. It is data: the registry test of
// check 5 reads the fields from it and TestValidate_FieldRules runs it.
var fieldCases = buildFieldCases()

func buildFieldCases() []fieldCase {
	var cs []fieldCase
	add := func(c ...fieldCase) { cs = append(cs, c...) }
	bounds := []error{economysim.ErrOutOfBounds}

	for _, wb := range worldBounds {
		add(boundCase("World", wb.field, []string{"world", snake(wb.field)}, func(c *economysim.Config) {
			setInt(&c.World, wb.field, wb.bad)
		}))
	}
	add(
		// The orders between two fields of the world.
		fieldCase{field: "World.BeliefStockLow", name: "not below BeliefStockHigh", rules: bounds,
			mutate: func(c *economysim.Config) { c.World.BeliefStockLow = c.World.BeliefStockHigh },
			paths:  [][]string{{"world", "belief_stock_low"}}, value: "15000", bound: "14999"},
		fieldCase{field: "World.BeliefWidthMin", name: "above BeliefWidthInit", rules: bounds,
			mutate: func(c *economysim.Config) { c.World.BeliefWidthMin = 3_000 },
			paths:  [][]string{{"world", "belief_width_min"}}, value: "3000", bound: "2000"},
		fieldCase{field: "World.BeliefWidthInit", name: "above BeliefWidthMax", rules: bounds,
			mutate: func(c *economysim.Config) { c.World.BeliefWidthInit = 6_000 },
			paths:  [][]string{{"world", "belief_width_init"}}, value: "6000", bound: "5000"},
		fieldCase{field: "World.BeliefWidthStep", name: "above BeliefWidthMax", rules: bounds,
			mutate: func(c *economysim.Config) { c.World.BeliefWidthStep = 6_000 },
			paths:  [][]string{{"world", "belief_width_step"}}, value: "6000", bound: "5000"},
		fieldCase{field: "World.CostCeiling", name: "below CostFloor", rules: bounds,
			mutate: func(c *economysim.Config) { c.World.CostCeiling = 6_000 },
			paths:  [][]string{{"world", "cost_ceiling"}}, value: "6000", bound: "7000"},
		fieldCase{field: "World.ChangePrivation", name: "not below RemovePrivation (rule ladder_order)", rules: bounds,
			mutate: func(c *economysim.Config) { c.World.ChangePrivation = c.World.RemovePrivation },
			paths:  [][]string{{"world", "change_privation"}}, value: "10", bound: "ladder_order"},
		fieldCase{field: "World.Upkeep", name: "quantity of 0", rules: bounds,
			mutate: func(c *economysim.Config) { c.World.Upkeep[0].Quantity = 0 },
			paths:  [][]string{{"world", "upkeep", "staple", "quantity"}}, value: "0", bound: "1"},
		fieldCase{field: "World.DistanceLabel", name: "label over 256 bytes", rules: bounds,
			mutate: func(c *economysim.Config) { c.World.DistanceLabel = strings.Repeat("x", 257) },
			paths:  [][]string{{"world", "distance_label"}}, value: "257", bound: "256"},
	)

	// The calendar (spec-calendar.md 7.2).
	cal := func(field string, mutate func(cal *economysim.Calendar), rule error, paths ...[]string) fieldCase {
		return fieldCase{field: field, name: "calendar", rules: []error{rule},
			mutate: func(c *economysim.Config) { mutate(&c.Calendar) }, paths: paths}
	}
	add(
		cal("Calendar.HoursPerDay", func(c *economysim.Calendar) { c.HoursPerDay = 0 }, economysim.ErrOutOfBounds, []string{"calendar", "hours_per_day"}),
		cal("Calendar.DaysPerWeek", func(c *economysim.Calendar) { c.DaysPerWeek = 101 }, economysim.ErrOutOfBounds, []string{"calendar", "days_per_week"}),
		cal("Calendar.TicksPerDay", func(c *economysim.Calendar) { c.TicksPerDay = 0 }, economysim.ErrOutOfBounds, []string{"calendar", "ticks_per_day"}),
		cal("Calendar.DaysPerTick", func(c *economysim.Calendar) { c.DaysPerTick = 0 }, economysim.ErrOutOfBounds, []string{"calendar", "days_per_tick"}),
		cal("Calendar.StartDay", func(c *economysim.Calendar) { c.StartDay = 365 }, economysim.ErrOutOfBounds, []string{"calendar", "start_day"}),
		cal("Calendar.SeasonStart", func(c *economysim.Calendar) { c.SeasonStart = 365 }, economysim.ErrOutOfBounds, []string{"calendar", "season_start"}),
		cal("Calendar.Months", func(c *economysim.Calendar) { c.Months = nil }, economysim.ErrOutOfBounds, []string{"calendar", "months"}),
		cal("Month.Days", func(c *economysim.Calendar) { c.Months[2].Days = 0 }, economysim.ErrOutOfBounds, []string{"calendar", "months[2]", "days"}),
		cal("Calendar.Months", func(c *economysim.Calendar) {
			// Twelve months of 10 000 days: 120 000 days against 100 000.
			for i := range c.Months {
				c.Months[i].Days = economysim.MaxDaysPerMonth
			}
		}, economysim.ErrOutOfBounds, []string{"calendar", "months"}),
		cal("Calendar.DaysPerTick", func(c *economysim.Calendar) { c.TicksPerDay, c.DaysPerTick = 2, 2 },
			economysim.ErrCalendarTick, []string{"calendar", "days_per_tick"}),
		cal("Calendar.DaysPerTick", func(c *economysim.Calendar) { c.DaysPerTick = 7 },
			economysim.ErrCalendarTick, []string{"calendar", "days_per_tick"}),
		cal("Calendar.TicksPerDay", func(c *economysim.Calendar) { c.TicksPerDay = 1_000 },
			economysim.ErrOutOfBounds, []string{"calendar", "ticks_per_year"}),
		cal("Season.Days", func(c *economysim.Calendar) { c.Seasons[1].Days = 0 },
			economysim.ErrCalendarSeasons, []string{"calendar", "seasons[1]", "days"}),
		cal("Calendar.Seasons", func(c *economysim.Calendar) { c.Seasons[0].Days = 93 },
			economysim.ErrCalendarSeasons, []string{"calendar", "seasons"}),
		cal("Month.ID", func(c *economysim.Calendar) { c.Months[3].ID = "M4" }, economysim.ErrInvalidID, []string{"calendar", "months[3]", "id"}),
		cal("Season.ID", func(c *economysim.Calendar) { c.Seasons[3].ID = "Winter" }, economysim.ErrInvalidID, []string{"calendar", "seasons[3]", "id"}),
	)

	// Commodities (spec-goods.md 8).
	com := func(field string, mutate func(cm *economysim.Commodity), rule error, paths ...[]string) fieldCase {
		return fieldCase{field: field, name: "commodity fuel", rules: []error{rule},
			mutate: func(c *economysim.Config) { mutate(commodityOf(c, "fuel")) }, paths: paths}
	}
	add(
		com("Commodity.Mass", func(cm *economysim.Commodity) { cm.Mass = -1 }, economysim.ErrOutOfBounds, []string{"commodities", "fuel", "mass"}),
		com("Commodity.Bulk", func(cm *economysim.Commodity) { cm.Bulk = 1_000_001 }, economysim.ErrOutOfBounds, []string{"commodities", "fuel", "bulk"}),
		com("Commodity.ShelfLife", func(cm *economysim.Commodity) { cm.ShelfLife = -1 }, economysim.ErrOutOfBounds, []string{"commodities", "fuel", "shelf_life"}),
		com("Commodity.Durability", func(cm *economysim.Commodity) { cm.Durability = -1 }, economysim.ErrOutOfBounds, []string{"commodities", "fuel", "durability"}),
		com("Commodity.ReferencePrice", func(cm *economysim.Commodity) { cm.ReferencePrice = 0 }, economysim.ErrOutOfBounds, []string{"commodities", "fuel", "reference_price"}),
		com("Commodity.Classes", func(cm *economysim.Commodity) {
			cm.Classes = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
		}, economysim.ErrOutOfBounds, []string{"commodities", "fuel", "classes"}),
		com("Commodity.Classes", func(cm *economysim.Commodity) { cm.Classes = []string{"fuel", "Bad"} },
			economysim.ErrInvalidID, []string{"commodities", "fuel", "classes", "Bad"}),
		com("Commodity.Classes", func(cm *economysim.Commodity) { cm.Classes = []string{"fuel", "fuel"} },
			economysim.ErrDuplicateID, []string{"commodities", "fuel", "classes", "fuel"}),
		com("Commodity.Label", func(cm *economysim.Commodity) { cm.Label = strings.Repeat("x", 257) },
			economysim.ErrOutOfBounds, []string{"commodities", "fuel", "label"}),
		com("Commodity.UnitLabel", func(cm *economysim.Commodity) { cm.UnitLabel = strings.Repeat("x", 257) },
			economysim.ErrOutOfBounds, []string{"commodities", "fuel", "unit_label"}),
	)
	// An entity added with an ID that is not one: the rest of it is the defaults.
	add(
		fieldCase{field: "Commodity.ID", name: "upper case letter", rules: []error{economysim.ErrInvalidID},
			mutate: func(c *economysim.Config) {
				// waste is only the discard output of refine; it is renamed with it.
				commodityOf(c, "waste").ID = "Waste"
				recipeOf(c, "refine").Outputs[1].Commodity = "Waste"
			},
			paths: [][]string{{"commodities", "Waste", "id"}}},
		fieldCase{field: "Recipe.ID", name: "upper case letter", rules: []error{economysim.ErrInvalidID},
			mutate: func(c *economysim.Config) {
				r := economysim.DefaultRecipe()
				r.ID = "Bad"
				r.Outputs = []economysim.Output{{Commodity: "seed", Quantity: 1, CostShare: 10_000}}
				c.Recipes = append(c.Recipes, r)
			},
			paths: [][]string{{"recipes", "Bad", "id"}}},
		fieldCase{field: "Role.ID", name: "upper case letter", rules: []error{economysim.ErrInvalidID},
			mutate: func(c *economysim.Config) {
				r := economysim.DefaultRole()
				r.ID = "Bad"
				c.Roles = append(c.Roles, r)
			},
			paths: [][]string{{"roles", "Bad", "id"}}},
		fieldCase{field: "Region.ID", name: "upper case letter", rules: []error{economysim.ErrInvalidID},
			mutate: func(c *economysim.Config) {
				r := economysim.DefaultRegion()
				r.ID, r.ReferenceBasket = "Bad", 999
				c.Regions = append(c.Regions, r)
			},
			paths: [][]string{{"regions", "Bad", "id"}}},
	)
	// The belief floor: a tradable commodity cheaper than ceil(10 000 /
	// BeliefStep) = 50 (FR-BEL-029, FR-GDS-002, FR-CUR-007).
	add(fieldCase{field: "Commodity.ReferencePrice", name: "tradable and below the resolution floor", rules: bounds,
		// seed is an input of the grower's recipe, so it is tradable; it is not in
		// the upkeep, so the basket is unchanged. The seed price 222 is inside the
		// reference band of 40: 222 * 10 000 <= 40 * 100 000.
		mutate: func(c *economysim.Config) { commodityOf(c, "seed").ReferencePrice = 40 },
		paths:  [][]string{{"commodities", "seed", "reference_price"}}, value: "40", bound: "50"})

	// Recipes (spec-production.md 10.2).
	rec := func(field, recipe string, mutate func(rc *economysim.Recipe), rule error, paths ...[]string) fieldCase {
		return fieldCase{field: field, name: "recipe " + recipe, rules: []error{rule},
			mutate: func(c *economysim.Config) { mutate(recipeOf(c, recipe)) }, paths: paths}
	}
	add(
		rec("Recipe.Duration", "cut_fuel", func(r *economysim.Recipe) { r.Duration = 0 },
			economysim.ErrOutOfBounds, []string{"recipes", "cut_fuel", "duration"}),
		rec("Recipe.TraineeYield", "prepare_basic", func(r *economysim.Recipe) { r.TraineeYield = 0 },
			economysim.ErrOutOfBounds, []string{"recipes", "prepare_basic", "trainee_yield"}),
		rec("Item.Quantity", "refine", func(r *economysim.Recipe) { r.Inputs[1].Quantity = 0 },
			economysim.ErrOutOfBounds, []string{"recipes", "refine", "inputs", "fuel", "quantity"}),
		rec("Output.Quantity", "cut_fuel", func(r *economysim.Recipe) { r.Outputs[0].Quantity = 0 },
			economysim.ErrOutOfBounds, []string{"recipes", "cut_fuel", "outputs", "fuel", "quantity"}),
		rec("Tool.WearPerUse", "grow_staple", func(r *economysim.Recipe) { r.Requires[0].WearPerUse = 11 },
			economysim.ErrOutOfBounds, []string{"recipes", "grow_staple", "requires", "tool_a", "wear_per_use"}),
		rec("Tool.WearPerUse", "grow_staple", func(r *economysim.Recipe) { r.Requires[0].WearPerUse = 0 },
			economysim.ErrOutOfBounds, []string{"recipes", "grow_staple", "requires", "tool_a", "wear_per_use"}),
		rec("Recipe.Requires", "refine", func(r *economysim.Recipe) {
			// seed is a commodity of durability 0.
			r.Requires = []economysim.Tool{{Commodity: "seed", WearPerUse: 1}}
		}, economysim.ErrToolNotDurable, []string{"recipes", "refine", "requires", "seed", "commodity"}),
		rec("Recipe.TraineeEligible", "prepare_basic", func(r *economysim.Recipe) {
			r.Requires = []economysim.Tool{{Commodity: "tool_b", WearPerUse: 1}}
		}, economysim.ErrEligibleHasTool, []string{"recipes", "prepare_basic", "requires"}),
		rec("Output.CostShare", "refine", func(r *economysim.Recipe) { r.Outputs[0].CostShare = 9_000 },
			economysim.ErrCostShares, []string{"recipes", "refine", "outputs"}),
		rec("Output.CostShare", "refine", func(r *economysim.Recipe) { r.Outputs[1].CostShare = 100 },
			economysim.ErrCostShares, []string{"recipes", "refine", "outputs", "waste", "cost_share"}),
		rec("Recipe.Label", "refine", func(r *economysim.Recipe) { r.Label = strings.Repeat("x", 257) },
			economysim.ErrOutOfBounds, []string{"recipes", "refine", "label"}),
		rec("Output.Commodity", "refine", func(r *economysim.Recipe) {
			r.Outputs = append(r.Outputs, economysim.Output{Commodity: "nosuch", Quantity: 1, Discard: true})
		}, economysim.ErrUnknownCommodity, []string{"recipes", "refine", "outputs", "nosuch", "commodity"}),
		rec("Recipe.Inputs", "refine", func(r *economysim.Recipe) {
			r.Inputs = append(r.Inputs, economysim.Item{Commodity: "nosuch", Quantity: 1})
		}, economysim.ErrUnknownCommodity, []string{"recipes", "refine", "inputs", "nosuch", "commodity"}),
	)
	// A cost share of 20 000 is outside 0 to 10 000, and the shares of the
	// recipe then do not add to 10 000 either: two rules, one field.
	add(fieldCase{field: "Output.CostShare", name: "above 10 000", rules: []error{economysim.ErrOutOfBounds, economysim.ErrCostShares},
		mutate: func(c *economysim.Config) { recipeOf(c, "cut_fuel").Outputs[0].CostShare = 20_000 },
		paths: [][]string{
			{"recipes", "cut_fuel", "outputs"},
			{"recipes", "cut_fuel", "outputs", "fuel", "cost_share"},
		}})

	// Yield profiles (spec-production.md 6.1, 10.2).
	add(
		fieldCase{field: "YieldProfile.All", name: "above MaxYieldBP", rules: bounds,
			mutate: func(c *economysim.Config) { c.Yields[0].All = 100_001 },
			paths:  [][]string{{"yields", "north", "cut_fuel", "all"}}, value: "100001", bound: "100000"},
		fieldCase{field: "SeasonYield.BP", name: "below 0", rules: bounds,
			mutate: func(c *economysim.Config) { c.Yields[3].Seasons[0].BP = -1 },
			paths:  [][]string{{"yields", "north", "grow_staple", "seasons", "spring", "bp"}}},
		fieldCase{field: "SeasonYield.Season", name: "a season the calendar lacks", rules: []error{economysim.ErrUnknownSeason},
			mutate: func(c *economysim.Config) {
				c.Yields[3].Seasons = append(c.Yields[3].Seasons, economysim.SeasonYield{Season: "monsoon", BP: 1})
			},
			paths: [][]string{{"yields", "north", "grow_staple", "seasons", "monsoon", "season"}}},
		fieldCase{field: "SeasonYield.Season", name: "a season twice in one profile", rules: []error{economysim.ErrDuplicateID},
			mutate: func(c *economysim.Config) {
				c.Yields[3].Seasons = append(c.Yields[3].Seasons, economysim.SeasonYield{Season: "spring", BP: 1})
			},
			paths: [][]string{{"yields", "north", "grow_staple", "seasons", "spring", "season"}}},
		fieldCase{field: "YieldProfile.Seasons", name: "any season while the calendar has none", rules: []error{economysim.ErrUnknownSeason},
			mutate: func(c *economysim.Config) {
				c.Calendar.Seasons, c.Calendar.SeasonStart = nil, 0
			},
			paths: [][]string{{"yields", "north", "grow_staple", "seasons", "spring", "season"}}},
		fieldCase{field: "YieldProfile.Region", name: "unknown region", rules: []error{economysim.ErrUnknownRegion},
			mutate: func(c *economysim.Config) { c.Yields[0].Region = "nowhere" },
			paths:  [][]string{{"yields", "nowhere", "cut_fuel", "region"}}},
		fieldCase{field: "YieldProfile.Recipe", name: "unknown recipe", rules: []error{economysim.ErrUnknownRecipe},
			mutate: func(c *economysim.Config) { c.Yields[0].Recipe = "nosuch" },
			paths:  [][]string{{"yields", "north", "nosuch", "recipe"}}},
		fieldCase{field: "Config.Yields", name: "two profiles for one region and recipe", rules: []error{economysim.ErrDuplicateID},
			mutate: func(c *economysim.Config) { c.Yields = append(c.Yields, c.Yields[0]) },
			paths:  [][]string{{"yields", "north", "cut_fuel"}}},
	)

	// Roles (spec-agents.md 12, spec-welfare-and-wages.md 12, spec-skill-barriers.md 5).
	roleBad := []struct {
		field string
		bad   int64
	}{
		{"StartMoney", -1}, {"TargetRatio", 9_999}, {"TargetSpread", 10_000}, {"DefaultRatio", -1},
		{"InKind", 10_001}, {"ExpectedWorkTicks", 0}, {"MinWorkShare", 10_001}, {"TrainingTicks", -1},
		{"MaxTrainees", 0}, {"TeachingLoad", 10_001},
	}
	for _, rb := range roleBad {
		add(boundCase("Role", rb.field, []string{"roles", "cook", snake(rb.field)}, func(c *economysim.Config) {
			setInt(roleOf(c, "cook"), rb.field, rb.bad)
		}))
	}
	role := func(field string, mutate func(r *economysim.Role), rule error, paths ...[]string) fieldCase {
		return fieldCase{field: field, name: "role cook", rules: []error{rule},
			mutate: func(c *economysim.Config) { mutate(roleOf(c, "cook")) }, paths: paths}
	}
	add(
		// 15 000 * (10 000 - 4 000) / 10 000 = 9 000 is below 10 000; the most the
		// spread may be is 10 000 - ceil(10 000^2 / 15 000) = 10 000 - 6 667 = 3 333.
		fieldCase{field: "Role.TargetSpread", name: "lowest target below 10 000", rules: bounds,
			mutate: func(c *economysim.Config) { roleOf(c, "cook").TargetSpread = 4_000 },
			paths:  [][]string{{"roles", "cook", "target_spread"}}, value: "4000", bound: "3333"},
		// A year of 365 ticks: 366 is above it, and the error carries 365.
		fieldCase{field: "Role.ExpectedWorkTicks", name: "above TicksPerYear", rules: bounds,
			mutate: func(c *economysim.Config) { roleOf(c, "cook").ExpectedWorkTicks = 366 },
			paths:  [][]string{{"roles", "cook", "expected_work_ticks"}}, value: "366", bound: "365"},
		// 8 000 * 365 = 2 920 000 is above 250 * 10 000 = 2 500 000; the most is
		// 250 * 10 000 / 365 = 6 849.
		fieldCase{field: "Role.MinWorkShare", name: "above ExpectedWorkTicks / TicksPerYear", rules: bounds,
			mutate: func(c *economysim.Config) { roleOf(c, "cook").MinWorkShare = 8_000 },
			paths:  [][]string{{"roles", "cook", "min_work_share"}}, value: "8000", bound: "6849"},
		// Duration 2 000 on the grower's recipe: window 2 * 2 000 = 4 000 > 3 650.
		fieldCase{field: "Recipe.Duration", name: "effective window above MaxWelfareWindow", rules: bounds,
			mutate: func(c *economysim.Config) { recipeOf(c, "grow_staple").Duration = 2_000 },
			paths:  [][]string{{"roles", "grower", "recipes"}}, value: "4000", bound: "3650"},
		fieldCase{field: "Role.Upkeep", name: "quantity of 0", rules: bounds,
			mutate: func(c *economysim.Config) {
				roleOf(c, "cook").Upkeep = []economysim.Item{{Commodity: "staple", Quantity: 0}}
			},
			paths: [][]string{{"roles", "cook", "upkeep", "staple", "quantity"}}},
		fieldCase{field: "Role.StartStock", name: "above MaxQuantity / MaxWeight", rules: bounds,
			mutate: func(c *economysim.Config) { roleOf(c, "cook").StartStock[0].Quantity = 1_001 },
			paths:  [][]string{{"roles", "cook", "start_stock", "fuel", "quantity"}}, value: "1001", bound: "1000"},
		fieldCase{field: "Role.StartMoney", name: "above MaxMoney / MaxWeight", rules: bounds,
			mutate: func(c *economysim.Config) { roleOf(c, "cook").StartMoney = 1_000_000_000_001 },
			paths:  [][]string{{"roles", "cook", "start_money"}}, value: "1000000000001", bound: "1000000000000"},
		role("Role.Label", func(r *economysim.Role) { r.Label = strings.Repeat("x", 257) },
			economysim.ErrOutOfBounds, []string{"roles", "cook", "label"}),
		role("Role.Recipes", func(r *economysim.Role) { r.Recipes = append(r.Recipes, "nosuch") },
			economysim.ErrUnknownRecipe, []string{"roles", "cook", "recipes", "nosuch"}),
		role("Role.Trades", func(r *economysim.Role) { r.Trades = []string{"nosuch"} },
			economysim.ErrUnknownCommodity, []string{"roles", "cook", "trades", "nosuch"}),
		// The rule traineeship_needs_weight_one: a traineeship needs agents of
		// weight 1, and a group of scale 2 has agents of weight 2.
		fieldCase{field: "Role.TrainingTicks", name: "traineeship_needs_weight_one", rules: bounds,
			mutate: func(c *economysim.Config) {
				roleOf(c, "cook").TrainingTicks = 5
				c.Groups[0].Scale = 2
			},
			paths: [][]string{{"groups", "north", "cook", "scale"}}, value: "2", bound: "traineeship_needs_weight_one"},
	)

	// Regions.
	reg := func(field string, mutate func(r *economysim.Region), rule error, paths ...[]string) fieldCase {
		return fieldCase{field: field, name: "region north", rules: []error{rule},
			mutate: func(c *economysim.Config) { mutate(&c.Regions[0]) }, paths: paths}
	}
	add(
		reg("Region.ReferenceBasket", func(r *economysim.Region) { r.ReferenceBasket = -1 }, economysim.ErrOutOfBounds,
			[]string{"regions", "north", "reference_basket"}),
		reg("Region.CapitalReturn", func(r *economysim.Region) { r.CapitalReturn = -1 }, economysim.ErrOutOfBounds,
			[]string{"regions", "north", "capital_return"}),
		reg("Region.EstateMarkdown", func(r *economysim.Region) { r.EstateMarkdown = 10_001 }, economysim.ErrOutOfBounds,
			[]string{"regions", "north", "estate_markdown"}),
		reg("Region.ExtraUpkeep", func(r *economysim.Region) {
			r.ExtraUpkeep = []economysim.Item{{Commodity: "fuel", Quantity: 0}}
		}, economysim.ErrOutOfBounds, []string{"regions", "north", "extra_upkeep", "fuel", "quantity"}),
		reg("Region.Label", func(r *economysim.Region) { r.Label = strings.Repeat("x", 257) },
			economysim.ErrOutOfBounds, []string{"regions", "north", "label"}),
		// 1 200 against a baseline of 999: |1 200 - 999| = 201 is above
		// 1 000 * 1 200 / 10 000 = 120.
		fieldCase{field: "Region.ReferenceBasket", name: "differs from the upkeep by more than the tolerance",
			rules:  []error{economysim.ErrBasketMismatch},
			mutate: func(c *economysim.Config) { c.Regions[0].ReferenceBasket = 1_200 },
			paths:  [][]string{{"regions", "north", "reference_basket"}}, value: "1200", bound: "999"},
		fieldCase{field: "World.BasketTolerance", name: "the cost of an extra upkeep item moves the baseline",
			rules: []error{economysim.ErrBasketMismatch},
			// fuel 179 added: 999 + 179 = 1 178; |1 000 - 1 178| = 178 is above
			// 1 000 * 1 000 / 10 000 = 100.
			mutate: func(c *economysim.Config) {
				c.Regions[0].ExtraUpkeep = []economysim.Item{{Commodity: "fuel", Quantity: 1}}
			},
			paths: [][]string{{"regions", "north", "reference_basket"}}, value: "1000", bound: "1178"},
	)

	// Groups.
	add(
		boundCase("Group", "Households", []string{"groups", "north", "cook", "households"}, func(c *economysim.Config) {
			c.Groups[0].Households = 0
		}),
		boundCase("Group", "Scale", []string{"groups", "north", "cook", "scale"}, func(c *economysim.Config) {
			c.Groups[0].Scale = -1
		}),
		// 20 000 000 households at one scale: n = 1, raised to the floor K = 10,
		// so each weight is 2 000 000, above MaxWeight.
		fieldCase{field: "Group.Scale", name: "weight above MaxWeight", rules: bounds,
			mutate: func(c *economysim.Config) {
				g := &c.Groups[1] // digger
				g.Households, g.Scale = 20_000_000, 20_000_000
				// The warm start's stock would overflow its weight too; its own
				// rule is tested apart.
				c.WarmStart.Groups[1].Stock = nil
			},
			paths: [][]string{{"groups", "north", "digger", "scale"}}, value: "2000000", bound: "1000000"},
		fieldCase{field: "Group.Region", name: "unknown region", rules: []error{economysim.ErrUnknownRegion},
			mutate: func(c *economysim.Config) {
				c.Groups = append(c.Groups, economysim.Group{Region: "nowhere", Role: "visitor", Households: 1})
			},
			paths: [][]string{{"groups", "nowhere", "visitor", "region"}}},
		fieldCase{field: "Group.Role", name: "unknown role", rules: []error{economysim.ErrUnknownRole},
			mutate: func(c *economysim.Config) {
				c.Groups = append(c.Groups, economysim.Group{Region: "north", Role: "ghost", Households: 1})
			},
			paths: [][]string{{"groups", "north", "ghost", "role"}}},
	)

	// The warm start (spec-viability.md 8.2, 8.7).
	warm := func(field, name string, mutate func(w *economysim.WarmStart), rule error, paths ...[]string) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule},
			mutate: func(c *economysim.Config) { mutate(c.WarmStart) }, paths: paths}
	}
	add(
		warm("SeedPrice.Price", "below 1", func(w *economysim.WarmStart) { w.Prices[0].Price = 0 },
			economysim.ErrWarmStartPrice, []string{"warm_start", "prices", "north", "fuel", "price"}),
		warm("SeedPrice.Price", "above MaxPrice", func(w *economysim.WarmStart) { w.Prices[0].Price = 1_000_000_001 },
			economysim.ErrWarmStartPrice, []string{"warm_start", "prices", "north", "fuel", "price"}),
		// 40 is above 1 and inside the reference band of 179 (1 000 to 100 000
		// basis points: 17.9 to 1 790), and below the floor of 50.
		fieldCase{field: "SeedPrice.Price", name: "below the resolution floor", rules: []error{economysim.ErrWarmStartPrice},
			mutate: func(c *economysim.Config) { c.WarmStart.Prices[0].Price = 40 },
			paths:  [][]string{{"warm_start", "prices", "north", "fuel", "price"}}, value: "40", bound: "50"},
		// RefBandLow of 10 000: the band starts at the reference price 179; a
		// seed of 100 clears the floor and is below the band.
		fieldCase{field: "World.RefBandLow", name: "a seed price below the reference band", rules: []error{economysim.ErrWarmStartPrice},
			mutate: func(c *economysim.Config) {
				c.World.RefBandLow = 10_000
				c.WarmStart.Prices[0].Price = 100
			},
			paths: [][]string{{"warm_start", "prices", "north", "fuel", "price"}}, value: "100", bound: "179"},
		fieldCase{field: "World.RefBandHigh", name: "a seed price above the reference band", rules: []error{economysim.ErrWarmStartPrice},
			mutate: func(c *economysim.Config) {
				c.World.RefBandHigh = 10_000
				c.WarmStart.Prices[0].Price = 200
			},
			paths: [][]string{{"warm_start", "prices", "north", "fuel", "price"}}, value: "200", bound: "179"},
		warm("SeedPrice.Commodity", "unknown commodity", func(w *economysim.WarmStart) {
			w.Prices = append(w.Prices, economysim.SeedPrice{Region: "north", Commodity: "nosuch", Price: 100})
		}, economysim.ErrUnknownCommodity, []string{"warm_start", "prices", "north", "nosuch", "commodity"}),
		warm("SeedPrice.Region", "unknown region", func(w *economysim.WarmStart) {
			w.Prices = append(w.Prices, economysim.SeedPrice{Region: "nowhere", Commodity: "fuel", Price: 179})
		}, economysim.ErrUnknownRegion, []string{"warm_start", "prices", "nowhere", "fuel", "region"}),
		warm("SeedWage.Wage", "below 1", func(w *economysim.WarmStart) { w.Wages[0].Wage = 0 },
			economysim.ErrWarmStartPrice, []string{"warm_start", "wages", "north", "wage"}),
		warm("SeedWage.Wage", "above MaxMoney", func(w *economysim.WarmStart) { w.Wages[0].Wage = 1_000_000_000_000_000_001 },
			economysim.ErrWarmStartPrice, []string{"warm_start", "wages", "north", "wage"}),
		// The longest recipe lasts 60 ticks: 16 666 666 666 666 667 * 60 is above
		// MaxMoney. The roles' expected work is 1 tick so that wage * work, the
		// spawner's product, is within MaxMoney and only this rule speaks.
		fieldCase{field: "SeedWage.Wage", name: "times the longest Duration above MaxMoney", rules: []error{economysim.ErrWarmStartPrice},
			mutate: func(c *economysim.Config) {
				c.WarmStart.Wages[0].Wage = 16_666_666_666_666_667
				for i := range c.Roles {
					c.Roles[i].ExpectedWorkTicks, c.Roles[i].MinWorkShare = 1, 0
				}
			},
			paths: [][]string{{"warm_start", "wages", "north", "wage"}}, value: "16666666666666667", bound: "16666666666666666"},
		warm("SeedWage.Region", "unknown region", func(w *economysim.WarmStart) {
			w.Wages = append(w.Wages, economysim.SeedWage{Region: "nowhere", Wage: 1})
		}, economysim.ErrUnknownRegion, []string{"warm_start", "wages", "nowhere", "region"}),
		warm("WarmStart.Wages", "a second wage for one region", func(w *economysim.WarmStart) {
			w.Wages = append(w.Wages, w.Wages[0])
		}, economysim.ErrDuplicateID, []string{"warm_start", "wages", "north"}),
		warm("WarmGroup.Money", "below 0", func(w *economysim.WarmStart) { w.Groups[0].Money = -1 },
			economysim.ErrOutOfBounds, []string{"warm_start", "groups", "north", "cook", "money"}),
		warm("WarmGroup.Money", "above MaxMoney", func(w *economysim.WarmStart) { w.Groups[0].Money = 1_000_000_000_000_000_001 },
			economysim.ErrOverflow, []string{"warm_start", "groups", "north", "cook", "money"}),
		warm("WarmGroup.Stock", "quantity of 0", func(w *economysim.WarmStart) { w.Groups[0].Stock[0].Quantity = 0 },
			economysim.ErrOutOfBounds, []string{"warm_start", "groups", "north", "cook", "stock", "fuel", "quantity"}),
		warm("WarmGroup.Stock", "quantity above MaxQuantity", func(w *economysim.WarmStart) { w.Groups[0].Stock[0].Quantity = 1_000_000_001 },
			economysim.ErrOverflow, []string{"warm_start", "groups", "north", "cook", "stock", "fuel", "quantity"}),
		// 136 households at scale 136: n = 1, raised to K = 10, weight 14; a stock
		// of 100 000 000 per household is 1 400 000 000 above MaxQuantity.
		fieldCase{field: "WarmGroup.Stock", name: "quantity times the weight above MaxQuantity", rules: []error{economysim.ErrOverflow},
			mutate: func(c *economysim.Config) {
				c.Groups[0].Scale = 136
				c.WarmStart.Groups[0].Stock[0].Quantity = 100_000_000
			},
			paths: [][]string{{"warm_start", "groups", "north", "cook", "stock", "fuel", "quantity"}}, value: "1400000000", bound: "1000000000"},
		warm("WarmGroup.Stock", "unknown commodity", func(w *economysim.WarmStart) {
			g := &w.Groups[0]
			g.Stock = append(g.Stock, economysim.Item{Commodity: "nosuch", Quantity: 1})
		}, economysim.ErrUnknownCommodity, []string{"warm_start", "groups", "north", "cook", "stock", "nosuch", "commodity"}),
		warm("WarmGroup.Region", "unknown region", func(w *economysim.WarmStart) {
			w.Groups = append(w.Groups, economysim.WarmGroup{Region: "nowhere", Role: "visitor", Money: 1})
		}, economysim.ErrUnknownRegion, []string{"warm_start", "groups", "nowhere", "visitor", "region"}),
		warm("WarmGroup.Role", "unknown role", func(w *economysim.WarmStart) {
			w.Groups = append(w.Groups, economysim.WarmGroup{Region: "north", Role: "ghost", Money: 1})
		}, economysim.ErrUnknownRole, []string{"warm_start", "groups", "north", "ghost", "role"}),
		warm("WarmStart.Groups", "two rows for one region and role", func(w *economysim.WarmStart) {
			w.Groups = append(w.Groups, w.Groups[0])
		}, economysim.ErrDuplicateID, []string{"warm_start", "groups", "north", "cook"}),
		warm("JobSpec.InProgressBP", "above 10 000", func(w *economysim.WarmStart) { w.Groups[4].Jobs[0].InProgressBP = 10_001 },
			economysim.ErrWarmStartJob, []string{"warm_start", "groups", "north", "maker", "jobs", "make_tool_a", "in_progress"}),
		warm("JobSpec.InProgressBP", "group shares above 10 000", func(w *economysim.WarmStart) {
			w.Groups[4].Jobs[0].InProgressBP, w.Groups[4].Jobs[1].InProgressBP = 6_000, 6_000
		}, economysim.ErrWarmStartJob, []string{"warm_start", "groups", "north", "maker", "jobs"}),
		warm("JobSpec.Recipe", "a recipe the role does not list", func(w *economysim.WarmStart) {
			w.Groups[4].Jobs[0].Recipe = "refine"
		}, economysim.ErrWarmStartJob, []string{"warm_start", "groups", "north", "maker", "jobs", "refine", "recipe"}),
		warm("JobSpec.Recipe", "unknown recipe", func(w *economysim.WarmStart) {
			w.Groups[4].Jobs[0].Recipe = "nosuch"
		}, economysim.ErrUnknownRecipe, []string{"warm_start", "groups", "north", "maker", "jobs", "nosuch", "recipe"}),
		warm("WarmPool.Money", "below 0", func(w *economysim.WarmStart) {
			w.Pools = []economysim.WarmPool{{Region: "north", Money: -1}}
		}, economysim.ErrOutOfBounds, []string{"warm_start", "pools", "north", "money"}),
		warm("WarmPool.Money", "above MaxMoney", func(w *economysim.WarmStart) {
			w.Pools = []economysim.WarmPool{{Region: "north", Money: 1_000_000_000_000_000_001}}
		}, economysim.ErrOverflow, []string{"warm_start", "pools", "north", "money"}),
		warm("WarmPool.Region", "unknown region", func(w *economysim.WarmStart) {
			w.Pools = []economysim.WarmPool{{Region: "nowhere", Money: 1}}
		}, economysim.ErrUnknownRegion, []string{"warm_start", "pools", "nowhere", "region"}),
		warm("WarmStart.Prices", "two seed prices for one region and commodity", func(w *economysim.WarmStart) {
			w.Prices = append(w.Prices, w.Prices[0])
		}, economysim.ErrDuplicateID, []string{"warm_start", "prices", "north", "fuel"}),
		fieldCase{field: "Config.WarmStart", name: "a wage times the expected work above MaxMoney",
			rules: []error{economysim.ErrOverflow},
			// The longest recipe is 60 ticks and the wage 16 666 666 666 666 666 is
			// within MaxMoney / 60, but times 250 expected work ticks it is not.
			mutate: func(c *economysim.Config) { c.WarmStart.Wages[0].Wage = 16_666_666_666_666_666 },
			paths: [][]string{
				{"roles", "cook", "expected_work_ticks"}, {"roles", "digger", "expected_work_ticks"},
				{"roles", "forester", "expected_work_ticks"}, {"roles", "grower", "expected_work_ticks"},
				{"roles", "maker", "expected_work_ticks"}, {"roles", "refiner", "expected_work_ticks"},
			}, value: "250", bound: "60"},
	)
	// The references and shapes of the lists of items.
	add(
		fieldCase{field: "Recipe.Outputs", name: "no output", rules: []error{economysim.ErrRecipeNoOutput},
			mutate: func(c *economysim.Config) {
				r := economysim.DefaultRecipe()
				r.ID, r.Inputs = "idle", []economysim.Item{{Commodity: "staple", Quantity: 1}}
				c.Recipes = append(c.Recipes, r)
			},
			paths: [][]string{{"recipes", "idle", "outputs"}}},
		fieldCase{field: "Item.Commodity", name: "unknown commodity in the upkeep", rules: []error{economysim.ErrUnknownCommodity},
			mutate: func(c *economysim.Config) {
				c.World.Upkeep = append(c.World.Upkeep, economysim.Item{Commodity: "nosuch", Quantity: 1})
			},
			paths: [][]string{{"world", "upkeep", "nosuch", "commodity"}}},
		fieldCase{field: "Tool.Commodity", name: "unknown commodity as a tool", rules: []error{economysim.ErrUnknownCommodity},
			mutate: func(c *economysim.Config) {
				r := recipeOf(c, "refine")
				r.Requires = append(r.Requires, economysim.Tool{Commodity: "nosuch", WearPerUse: 1})
			},
			paths: [][]string{{"recipes", "refine", "requires", "nosuch", "commodity"}}},
		fieldCase{field: "WarmStart.Pools", name: "two pools for one region", rules: []error{economysim.ErrDuplicateID},
			mutate: func(c *economysim.Config) {
				c.WarmStart.Pools = []economysim.WarmPool{{Region: "north", Money: 1}, {Region: "north", Money: 2}}
			},
			paths: [][]string{{"warm_start", "pools", "north"}}},
		fieldCase{field: "WarmGroup.Jobs", name: "a recipe twice in the jobs of a group", rules: []error{economysim.ErrDuplicateID},
			mutate: func(c *economysim.Config) {
				g := &c.WarmStart.Groups[4]
				g.Jobs = append(g.Jobs, g.Jobs[0])
			},
			paths: [][]string{{"warm_start", "groups", "north", "maker", "jobs", "make_tool_a", "recipe"}}},
	)

	// A case is named by its field, its description and the path it breaks, so
	// two cases for one field are two subtests.
	seen := map[string]int{}
	for i := range cs {
		cs[i].name += " at " + strings.Join(cs[i].paths[0], ".") + " by " + cs[i].rules[0].Error()
		key := cs[i].field + "/" + cs[i].name
		if seen[key]++; seen[key] > 1 {
			cs[i].name += " #" + strconv.Itoa(seen[key])
		}
	}
	return cs
}

// TestValidate_FieldRules verifies, for every field of the slice that an owner
// spec gives a bound or a reference, that a configuration that breaks only that
// field is rejected with the rule of the field, the path of the field and, where
// the case says so, the value and the bound.
//
// The cases are fieldCases, which the registry test reads to check that every
// row of the parameter-owner table has a failing case.
//
// Requirement: FR-CFG-009, FR-CFG-031, FR-CFG-035, FR-CFG-036, FR-CFG-037, FR-CFG-039, FR-CFG-040, FR-CFG-048, FR-CFG-049, FR-CFG-050, FR-CFG-051, FR-CFG-052, FR-CFG-054, FR-CFG-055, FR-CFG-057, FR-CFG-064, FR-CAL-011, FR-CAL-012, FR-CAL-013, FR-GDS-002, FR-GDS-004, FR-GDS-008, FR-GDS-013, FR-GDS-016, FR-GDS-019, FR-GDS-021, FR-PRO-004, FR-PRO-005, FR-PRO-006, FR-PRO-033, FR-PRO-034, FR-PRO-038, FR-AGT-018, FR-SCL-020, FR-WEL-014, FR-WEL-025, FR-WEL-028, FR-VIA-030, FR-VIA-033, FR-VIA-051, FR-CUR-007, FR-CUR-022, FR-BEL-027, FR-BEL-028, FR-BEL-029, FR-BEL-037, FR-SPN-037, FR-SPN-038
func TestValidate_FieldRules(t *testing.T) {
	t.Parallel()
	for _, tt := range fieldCases {
		t.Run(tt.field+" "+tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("Validate(%s: %s) = nil, want %v at %v", tt.field, tt.name, tt.rules[0], tt.paths)
			}
			if !errors.Is(err, tt.rules[0]) {
				t.Errorf("errors.Is(Validate(%s: %s), %v) = false, want true; error: %v", tt.field, tt.name, tt.rules[0], err)
			}
			var ve *economysim.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("errors.As(Validate(%s: %s), *ValidationError) = false, want true", tt.field, tt.name)
			}
			vs, other := violations(err)
			if len(other) != 0 {
				t.Errorf("Validate(%s: %s) returned entries that are not a *ValidationError: %v", tt.field, tt.name, other)
			}
			for _, v := range vs {
				if !slices.ContainsFunc(tt.rules, func(r error) bool { return errors.Is(v, r) }) {
					t.Errorf("Validate(%s: %s) returned %v, want only %v", tt.field, tt.name, v, tt.rules)
				}
			}
			if got := pathsOf(vs); !reflect.DeepEqual(got, tt.paths) {
				t.Errorf("Validate(%s: %s) paths = %q, want %q", tt.field, tt.name, got, tt.paths)
			}
			if len(vs) > 0 {
				if tt.value != "" && vs[0].Value != tt.value {
					t.Errorf("Validate(%s: %s) first value = %q, want %q", tt.field, tt.name, vs[0].Value, tt.value)
				}
				if tt.bound != "" && vs[0].Bound != tt.bound {
					t.Errorf("Validate(%s: %s) first bound = %q, want %q", tt.field, tt.name, vs[0].Bound, tt.bound)
				}
			}
		})
	}
}

// TestValidate_FieldCasesAreTheSingleBreak verifies that the table has no two
// cases with the same field, name and rule list, so a subtest name is not
// shared, and that every case names a path.
//
// Guard: the registry test counts a field as covered when it is in this table.
// A case with no path, or one that a copy-paste left identical, would count a
// field as tested without testing it.
func TestValidate_FieldCasesAreTheSingleBreak(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for _, tt := range fieldCases {
		key := tt.field + "/" + tt.name + "/" + tt.rules[0].Error()
		if seen[key] {
			t.Errorf("two cases for %q", key)
		}
		seen[key] = true
		if len(tt.paths) == 0 {
			t.Errorf("case %q has no path", key)
		}
	}
}

// TestValidate_BasketToleranceBoundary verifies that a basket exactly at the
// tolerance is accepted and one unit beyond it is not, above the cost and below
// it: the allowed difference is the basket times 1 000 / 10 000, rounded down.
//
// Requirement: FR-CUR-022, FR-AGT-018
func TestValidate_BasketToleranceBoundary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		basket economysim.Money
		want   error
	}{
		// The cost is 999 and the tolerance 1 000 basis points of the basket:
		// 1 110 * 1 000 / 10 000 = 111, and |1 110 - 999| = 111.
		{"at the tolerance", 1_110, nil},
		{"one beyond", 1_111, economysim.ErrBasketMismatch},
		// Below the cost: 909 * 1 000 / 10 000 = 90 and |909 - 999| = 90; 908 gives
		// 90 and 91.
		{"below at the tolerance", 909, nil},
		{"below one beyond", 908, economysim.ErrBasketMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			c.Regions[0].ReferenceBasket = tt.basket
			err := c.Validate()
			if tt.want == nil && err != nil || tt.want != nil && !errors.Is(err, tt.want) {
				t.Errorf("Validate(basket %d) = %v, want %v", tt.basket, err, tt.want)
			}
		})
	}
}

// minimalConfig returns a configuration built from the Default constructors and
// the fields they leave required: one raw commodity that is the household's
// upkeep, one region, one role and one group. Its basket is the default 1 000
// and the upkeep costs 1 * 1 000, so it is valid with every default.
func minimalConfig() economysim.Config {
	world := economysim.DefaultWorld()
	world.Upkeep = []economysim.Item{{Commodity: "grain", Quantity: 1}}
	grain := economysim.DefaultCommodity()
	grain.ID, grain.ReferencePrice, grain.Raw = "grain", 1_000, true
	region := economysim.DefaultRegion()
	region.ID = "home"
	role := economysim.DefaultRole()
	role.ID = "farmer"
	return economysim.Config{
		World:       world,
		Calendar:    economysim.DefaultCalendar(),
		Commodities: []economysim.Commodity{grain},
		Roles:       []economysim.Role{role},
		Regions:     []economysim.Region{region},
		Groups:      []economysim.Group{{Region: "home", Role: "farmer", Households: 10}},
	}
}

// TestValidate_MinimalConfigurationFromDefaults verifies that a configuration
// that is only the defaults and the required fields validates (FR-CFG-029), and
// that the region's default basket is checked like a stated one: without the
// upkeep that makes it 1 000 the same configuration is rejected.
//
// Requirement: FR-CFG-029, FR-CUR-021
func TestValidate_MinimalConfigurationFromDefaults(t *testing.T) {
	t.Parallel()
	c := minimalConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate(minimal) = %v, want nil", err)
	}
	c.World.Upkeep = nil
	err := c.Validate()
	if !errors.Is(err, economysim.ErrBasketMismatch) {
		t.Errorf("Validate(minimal with no upkeep) = %v, want ErrBasketMismatch: the default basket of 1 000 is checked", err)
	}
}

// TestValidate_DerivedProducts verifies the derived checks of spec-config.md 4.2
// that the field bounds leave possible: a weight times a recipe amount, the cost
// of the inputs, the welfare window times the largest income, the calibration
// product, the money at the start and the warm start's products. Each case
// starts from the minimal configuration and breaks one product; the paths are
// those of the ErrOverflow errors (another rule a case touches is not asserted).
//
// Requirement: FR-CFG-010, FR-CFG-036, FR-PRO-039, FR-SCL-011, FR-WEL-010
func TestValidate_DerivedProducts(t *testing.T) {
	t.Parallel()
	// withSmelt adds a recipe that makes metal from two raw goods of the given
	// reference price and quantity, and gives it to the farmer.
	withSmelt := func(c *economysim.Config, rawPrice, metalPrice economysim.Price, in, out int64) {
		for _, id := range []string{"ore", "coal"} {
			cm := economysim.DefaultCommodity()
			cm.ID, cm.ReferencePrice, cm.Raw = id, rawPrice, true
			c.Commodities = append(c.Commodities, cm)
		}
		metal := economysim.DefaultCommodity()
		metal.ID, metal.ReferencePrice = "metal", metalPrice
		c.Commodities = append(c.Commodities, metal)
		r := economysim.DefaultRecipe()
		r.ID = "smelt"
		r.Inputs = []economysim.Item{{Commodity: "coal", Quantity: economysim.Quantity(in)}, {Commodity: "ore", Quantity: economysim.Quantity(in)}}
		r.Outputs = []economysim.Output{{Commodity: "metal", Quantity: economysim.Quantity(out), CostShare: 10_000}}
		c.Recipes = append(c.Recipes, r)
		c.Roles[0].Recipes = []string{"smelt"}
	}
	// heavy makes 10 000 households one group of 10 agents of weight 1 000.
	heavy := func(c *economysim.Config) { c.Groups[0].Households, c.Groups[0].Scale = 10_000, 10_000 }
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		paths  [][]string // of the ErrOverflow errors
		value  string
	}{
		{
			// Two inputs of 10^9 units at 10^9 each: 2 * 10^18 above MaxMoney.
			name:   "cost of the inputs",
			mutate: func(c *economysim.Config) { withSmelt(c, economysim.MaxPrice, 1_000, 1_000_000_000, 1) },
			paths:  [][]string{{"recipes", "smelt", "inputs"}},
			value:  "2000000000000000000",
		},
		{
			// An output of 10^6 at ten times the yield (10^7) with a weight of 1 000
			// is 10^10, above MaxQuantity.
			name: "weight times an output at the largest yield",
			mutate: func(c *economysim.Config) {
				withSmelt(c, 1_000, 1_000, 1, 1_000_000)
				heavy(c)
			},
			paths: [][]string{{"recipes", "smelt", "outputs", "metal", "quantity"}},
			value: "10000000000",
		},
		{
			// 10^9 units of an input at a weight of 1 000: 10^12 above MaxQuantity;
			// the role holds the same units as its job target.
			name: "weight times an input quantity",
			mutate: func(c *economysim.Config) {
				withSmelt(c, 1_000, 1_000, 1_000_000_000, 1)
				heavy(c)
			},
			paths: [][]string{
				{"recipes", "smelt", "inputs", "coal", "quantity"},
				{"recipes", "smelt", "inputs", "ore", "quantity"},
				{"roles", "farmer", "targets", "coal"},
				{"roles", "farmer", "targets", "ore"},
			},
			value: "1000000000000",
		},
		{
			// 1 000 units a household over 1 000 ticks of cover and the tick:
			// 1 001 000 units, times a weight of 1 000, is above MaxQuantity.
			name: "targets of the upkeep",
			mutate: func(c *economysim.Config) {
				c.World.Upkeep[0].Quantity = 1_000
				c.World.CoverTicks = 1_000
				c.Regions[0].ReferenceBasket = 1_000_000 // 1 000 units at 1 000
				heavy(c)
			},
			paths: [][]string{{"roles", "farmer", "targets", "grain"}},
			value: "1001000000",
		},
		{
			// An output of 10^6 at the largest price and ten times the yield is
			// 10^16 a tick; a weight of 100 (1 000 households at scale 1 000) and
			// a window of 1 000 ticks make 10^21, above int64.
			name: "window times the largest income",
			mutate: func(c *economysim.Config) {
				withSmelt(c, 1_000, economysim.MaxPrice, 1, 1_000_000)
				c.Groups[0].Households, c.Groups[0].Scale = 1_000, 1_000
				c.World.WelfareWindow = 1_000
			},
			paths: [][]string{{"roles", "farmer", "recipes"}},
			value: "overflow",
		},
		{
			// The wage of the target ratio at the most: 10^6 * 99 645 ticks of a
			// year * 10^18 / (10^4 * 1) does not fit. The basket 10^18 is also not
			// the cost of the upkeep, which is another rule.
			name: "calibration product",
			mutate: func(c *economysim.Config) {
				c.Calendar.TicksPerDay = 273 // 365 * 273 = 99 645 <= MaxTicksPerYear
				c.Regions[0].ReferenceBasket = economysim.MaxMoney
				c.Roles[0].TargetRatio, c.Roles[0].ExpectedWorkTicks, c.Roles[0].MinWorkShare = economysim.MaxBasisPoints, 1, 0
			},
			paths: [][]string{{"roles", "farmer", "target_ratio"}},
			value: "overflow",
		},
		{
			// Two households with the most money each: 2 * 10^18 above MaxMoney.
			name: "money at the start",
			mutate: func(c *economysim.Config) {
				c.Groups[0].Households = 2
				c.WarmStart = &economysim.WarmStart{Groups: []economysim.WarmGroup{{Region: "home", Role: "farmer", Money: economysim.MaxMoney}}}
			},
			paths: [][]string{{"groups"}},
			value: "2000000000000000000",
		},
		{
			// 10^9 households at 10^18 do not even fit int64.
			name: "money at the start beyond int64",
			mutate: func(c *economysim.Config) {
				c.Groups[0].Households = economysim.MaxHouseholds
				c.WarmStart = &economysim.WarmStart{Groups: []economysim.WarmGroup{{Region: "home", Role: "farmer", Money: economysim.MaxMoney}}}
			},
			paths: [][]string{{"groups"}},
			value: "overflow",
		},
		{
			// Ten pools of 10^18 add to 10^19, above int64 (9.2 * 10^18).
			name: "pools beyond int64",
			mutate: func(c *economysim.Config) {
				w := &economysim.WarmStart{}
				for i := 0; i < 10; i++ {
					id := "r" + strconv.Itoa(i)
					r := economysim.DefaultRegion()
					r.ID = id
					c.Regions = append(c.Regions, r)
					w.Pools = append(w.Pools, economysim.WarmPool{Region: id, Money: economysim.MaxMoney})
				}
				c.WarmStart = w
			},
			paths: [][]string{{"groups"}},
			value: "overflow",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := minimalConfig()
			tt.mutate(&c)
			err := c.Validate()
			vs, _ := violations(err)
			var overflow []*economysim.ValidationError
			for _, v := range vs {
				if errors.Is(v, economysim.ErrOverflow) {
					overflow = append(overflow, v)
				}
			}
			if got := pathsOf(overflow); !reflect.DeepEqual(got, tt.paths) {
				t.Fatalf("ErrOverflow paths of Validate(%s) = %q, want %q (all errors: %v)", tt.name, got, tt.paths, err)
			}
			if overflow[0].Value != tt.value {
				t.Errorf("Validate(%s) first overflow value = %q, want %q", tt.name, overflow[0].Value, tt.value)
			}
		})
	}
}

// TestValidate_BoundsAreInclusive verifies that a value exactly at a bound is
// valid for the bounds that the cases above break by one: MaxQuantity as an
// input, MaxMoney of money at the start, the expected work at the year.
//
// Requirement: FR-CFG-009, FR-CFG-010
func TestValidate_BoundsAreInclusive(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
	}{
		{"input of MaxQuantity at weight 1", func(c *economysim.Config) {
			c.Recipes[0].Inputs[0].Quantity = economysim.MaxQuantity
		}},
		{"money at the start of MaxMoney", func(c *economysim.Config) {
			c.Groups[0].Households = 1
			c.WarmStart = &economysim.WarmStart{Groups: []economysim.WarmGroup{{Region: "home", Role: "farmer", Money: economysim.MaxMoney}}}
		}},
		{"expected work of TicksPerYear", func(c *economysim.Config) {
			c.Roles[0].ExpectedWorkTicks, c.Roles[0].MinWorkShare = 365, 0
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := minimalConfig()
			// A recipe that mills salt, a raw good that is not the upkeep, into flour.
			salt, flour := economysim.DefaultCommodity(), economysim.DefaultCommodity()
			salt.ID, salt.ReferencePrice, salt.Raw = "salt", 1_000, true
			flour.ID, flour.ReferencePrice = "flour", 1_000
			r := economysim.DefaultRecipe()
			r.ID = "mill"
			r.Inputs = []economysim.Item{{Commodity: "salt", Quantity: 1}}
			r.Outputs = []economysim.Output{{Commodity: "flour", Quantity: 1, CostShare: 10_000}}
			c.Commodities = append(c.Commodities, salt, flour)
			c.Recipes = append(c.Recipes, r)
			c.Roles[0].Recipes = []string{"mill"}
			tt.mutate(&c)
			if err := c.Validate(); err != nil {
				t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
			}
		})
	}
}

// TestValidate_DefaultsAreInTheirBounds verifies that no default constructor
// returns a value that is outside a range of its own field: it builds a
// configuration of nothing but the defaults and the required fields and checks
// that no error is at a path of a defaulted field.
//
// Requirement: FR-CFG-029
func TestValidate_DefaultsAreInTheirBounds(t *testing.T) {
	t.Parallel()
	c := minimalConfig()
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(defaults) = %v, want nil", err)
	}
	if got := economysim.DefaultWorld().Upkeep; len(got) != 0 {
		t.Errorf("DefaultWorld().Upkeep = %v, want none: upkeep is a pack's", got)
	}
	// The calendar default is the Earth year the specs state.
	if cal := economysim.DefaultCalendar(); len(cal.Months) != 12 || cal.SeasonStart != 59 {
		t.Errorf("DefaultCalendar() = %d months from season day %d, want 12 and 59", len(cal.Months), cal.SeasonStart)
	}
	// math.MaxInt64 is not a configuration value anywhere: a value of it is out
	// of every range (no overflow in the range checks).
	w := economysim.DefaultWorld()
	w.WelfareWindow = math.MaxInt64
	c.World = w
	c.World.Upkeep = []economysim.Item{{Commodity: "grain", Quantity: 1}}
	if err := c.Validate(); !errors.Is(err, economysim.ErrOutOfBounds) {
		t.Errorf("Validate(WelfareWindow = MaxInt64) = %v, want ErrOutOfBounds", err)
	}
}

// sortedWarningsEqual reports whether two lists hold the same warnings.
func sortedWarningsEqual(a, b []economysim.Warning) bool {
	return slices.EqualFunc(a, b, func(x, y economysim.Warning) bool {
		return x.Rule == y.Rule && slices.Equal(x.Path, y.Path)
	})
}

// hasWarning reports whether ws holds a warning of the rule at the path.
func hasWarning(ws []economysim.Warning, rule string, path ...string) bool {
	return slices.ContainsFunc(ws, func(w economysim.Warning) bool {
		return w.Rule == rule && slices.Equal(w.Path, path)
	})
}

// TestWarnings verifies each warning of spec-config.md 7.4.2 that the slice
// raises without a solver: it fires on its condition at its path, does not fire
// on the case beside it, and never fails Validate (FR-CFG-018). A case that
// names invalid is a configuration that Validate rejects for another reason,
// because the rule is unreachable on a valid one.
//
// Requirement: FR-CFG-018, FR-CFG-053, FR-CFG-056, FR-CFG-059, FR-CFG-065, FR-CFG-066, FR-CFG-082, FR-AGT-060, FR-GDS-023, FR-PRO-040, FR-CAL-016, FR-CAL-021, FR-SKL-004, FR-WEL-009, FR-BEL-038, FR-SPN-039
func TestWarnings(t *testing.T) {
	t.Parallel()
	// A recipe with no inputs that no role lists, and what it makes.
	addSource := func(c *economysim.Config, recipe, out string, durability int64) {
		cm := economysim.DefaultCommodity()
		cm.ID, cm.ReferencePrice, cm.Durability = out, 100, durability
		c.Commodities = append(c.Commodities, cm)
		r := economysim.DefaultRecipe()
		r.ID = recipe
		r.Outputs = []economysim.Output{{Commodity: out, Quantity: 1, CostShare: 10_000}}
		c.Recipes = append(c.Recipes, r)
	}
	// calendarOf puts the reference configuration on another calendar: the roles
	// expect 50 ticks of work a year, which a year of 73 ticks or more holds, and
	// have no floor of work.
	//
	// The staple's shelf life of 700 ticks is nearly two years of daily ticks; on a
	// finer tick it falls below the 304-day gap between harvests (V4, storage_gap),
	// so it is raised to keep the pack viable.
	calendarOf := func(c *economysim.Config, edit func(cal *economysim.Calendar)) {
		edit(&c.Calendar)
		commodityOf(c, "staple").ShelfLife = 2_000
		for i := range c.Roles {
			c.Roles[i].ExpectedWorkTicks, c.Roles[i].MinWorkShare = 50, 0
		}
	}
	tests := []struct {
		name    string
		mutate  func(c *economysim.Config)
		rule    string
		path    []string
		fires   bool
		invalid bool // Validate rejects the configuration
	}{
		{
			name: "no_upkeep fires with no upkeep anywhere",
			mutate: func(c *economysim.Config) {
				c.World.Upkeep = nil
				c.Regions[0].ReferenceBasket = 0
			},
			rule: "no_upkeep", path: []string{"world", "upkeep"}, fires: true,
		},
		{
			name: "no_upkeep is not raised for a role upkeep",
			mutate: func(c *economysim.Config) {
				c.World.Upkeep = nil
				roleOf(c, "cook").Upkeep = []economysim.Item{{Commodity: "staple", Quantity: 1}}
				c.Regions[0].ReferenceBasket = 0
			},
			rule: "no_upkeep", path: []string{"world", "upkeep"}, fires: false,
		},
		{
			name: "no_open_role fires when every role is closed",
			mutate: func(c *economysim.Config) {
				for i := range c.Roles {
					c.Roles[i].Closed = true
				}
			},
			rule: "no_open_role", path: []string{"roles"}, fires: true,
		},
		{
			name: "commodity_untraded fires for a good no role uses",
			mutate: func(c *economysim.Config) {
				addSource(c, "make_x", "good_x", 0)
			},
			rule: "commodity_untraded", path: []string{"commodities", "good_x"}, fires: true,
		},
		{
			name: "commodity_untraded is not raised for a discard output only",
			mutate: func(c *economysim.Config) {
			},
			rule: "commodity_untraded", path: []string{"commodities", "waste"}, fires: false,
		},
		{
			// A good that is upkeep, not raw, made by no recipe and in no stock: the
			// structural rule rejects it first (it is consumed and not producible),
			// so the warning is raised only on a configuration that is invalid.
			name: "upkeep_unsourced fires for an upkeep nothing makes",
			mutate: func(c *economysim.Config) {
				cm := economysim.DefaultCommodity()
				cm.ID, cm.ReferencePrice = "ore_y", 100
				c.Commodities = append(c.Commodities, cm)
				c.World.Upkeep = append(c.World.Upkeep, economysim.Item{Commodity: "ore_y", Quantity: 1})
			},
			rule: "upkeep_unsourced", path: []string{"commodities", "ore_y"}, fires: true, invalid: true,
		},
		{
			name: "upkeep_unsourced is not raised for a raw good",
			mutate: func(c *economysim.Config) {
				cm := economysim.DefaultCommodity()
				cm.ID, cm.ReferencePrice, cm.Raw = "ore_y", 100, true
				c.Commodities = append(c.Commodities, cm)
				c.World.Upkeep = append(c.World.Upkeep, economysim.Item{Commodity: "ore_y", Quantity: 1})
			},
			// A Raw upkeep good that nothing real makes beside a produced basket is the
			// documented consequence of spec-viability.md 4.2.1: the pooled programme
			// is 0 and labour_short rejects it, and raw_without_source names the good.
			rule: "upkeep_unsourced", path: []string{"commodities", "ore_y"}, fires: false, invalid: true,
		},
		{
			// The upkeep of staple is 2 a household and the start stock 1.
			name: "start_stock_below_target fires below one tick of upkeep",
			mutate: func(c *economysim.Config) {
				roleOf(c, "digger").StartStock = []economysim.Item{{Commodity: "fuel", Quantity: 5}, {Commodity: "prepared", Quantity: 4}, {Commodity: "staple", Quantity: 1}}
			},
			rule: "start_stock_below_target", path: []string{"roles", "digger", "start_stock"}, fires: true,
		},
		{
			// make_tool_a needs 2 refined and make_tool_b 3; the maker holds none.
			name: "start_stock_below_target fires when no recipe can start",
			mutate: func(c *economysim.Config) {
				roleOf(c, "maker").StartStock = []economysim.Item{{Commodity: "fuel", Quantity: 5}, {Commodity: "prepared", Quantity: 4}, {Commodity: "staple", Quantity: 10}}
			},
			rule: "start_stock_below_target", path: []string{"roles", "maker", "start_stock"}, fires: true,
		},
		{
			// 3 ticks of reserve times 999 is 2 997; the cook has 2 000.
			name: "reserve_exceeds_money fires below the reserve",
			mutate: func(c *economysim.Config) {
				roleOf(c, "cook").StartMoney = 2_000
			},
			rule: "reserve_exceeds_money", path: []string{"roles", "cook", "start_money"}, fires: true,
		},
		{
			name: "reserve_exceeds_money is not raised at 2 997",
			mutate: func(c *economysim.Config) {
				roleOf(c, "cook").StartMoney = 2_997
			},
			rule: "reserve_exceeds_money", path: []string{"roles", "cook", "start_money"}, fires: false,
		},
		{
			name: "strategy_default_unused_trades fires for a role with households",
			mutate: func(c *economysim.Config) {
				roleOf(c, "digger").Trades = []string{"fuel"}
			},
			rule: "strategy_default_unused_trades", path: []string{"roles", "digger", "trades"}, fires: true,
		},
		{
			name: "strategy_default_unused_trades is not raised for a role with no group",
			mutate: func(c *economysim.Config) {
			},
			rule: "strategy_default_unused_trades", path: []string{"roles", "visitor", "trades"}, fires: false,
		},
		{
			name: "unused_recipe fires for a recipe no role lists",
			mutate: func(c *economysim.Config) {
				addSource(c, "spare", "good_x", 0)
			},
			rule: "unused_recipe", path: []string{"recipes", "spare"}, fires: true,
		},
		{
			name: "unused_recipe is not raised for a trainee-eligible recipe",
			mutate: func(c *economysim.Config) {
				addSource(c, "spare", "good_x", 0)
				recipeOf(c, "spare").TraineeEligible = true
			},
			rule: "unused_recipe", path: []string{"recipes", "spare"}, fires: false,
		},
		{
			name: "unrunnable_recipe fires when every profile entry is 0",
			mutate: func(c *economysim.Config) {
				c.Yields[0].All = 0
			},
			// A fuel that no region can make is also not viable (phase 6,
			// unproducible_in_realm and role_cannot_work): the warning is raised on a
			// configuration that is rejected.
			rule: "unrunnable_recipe", path: []string{"recipes", "cut_fuel"}, fires: true, invalid: true,
		},
		{
			name: "unrunnable_recipe fires for a Sited recipe with no profile",
			mutate: func(c *economysim.Config) {
				c.Yields = slices.Delete(c.Yields, 2, 3) // gather_seed
			},
			// Without gather_seed the seed cycle has no way in (phase 6,
			// unproducible_in_realm): rejected, and the warning still fires.
			rule: "unrunnable_recipe", path: []string{"recipes", "gather_seed"}, fires: true, invalid: true,
		},
		{
			name: "durable_not_required fires for a durable no recipe requires",
			mutate: func(c *economysim.Config) {
				addSource(c, "make_ring", "ring", 5)
			},
			rule: "durable_not_required", path: []string{"commodities", "ring"}, fires: true,
		},
		{
			name: "source_unbounded fires for a source that is not Sited",
			mutate: func(c *economysim.Config) {
				addSource(c, "make_x", "good_x", 0)
			},
			rule: "source_unbounded", path: []string{"recipes", "make_x"}, fires: true,
		},
		{
			name: "window_extended is not raised when the job is short",
			mutate: func(c *economysim.Config) {
				recipeOf(c, "grow_staple").Duration = 10
			},
			rule: "window_extended", path: []string{"roles", "grower", "recipes"}, fires: false,
		},
		{
			name: "cost_floor_at_full_cost fires at 10 000",
			mutate: func(c *economysim.Config) {
				c.World.CostFloor = 10_000
			},
			rule: "cost_floor_at_full_cost", path: []string{"world", "cost_floor"}, fires: true,
		},
		{
			name: "cost_floor_at_full_cost is not raised at 9 999",
			mutate: func(c *economysim.Config) {
				c.World.CostFloor = 9_999
			},
			rule: "cost_floor_at_full_cost", path: []string{"world", "cost_floor"}, fires: false,
		},
		{
			// 24 hours over 5 ticks a day is 4.8: not exact.
			name: "inexact_hour fires when TicksPerDay does not divide the hours",
			mutate: func(c *economysim.Config) {
				calendarOf(c, func(cal *economysim.Calendar) { cal.TicksPerDay = 5 })
			},
			rule: "inexact_hour", path: []string{"calendar", "ticks_per_day"}, fires: true,
		},
		{
			name: "inexact_hour is not raised for 4 ticks a day",
			mutate: func(c *economysim.Config) {
				calendarOf(c, func(cal *economysim.Calendar) { cal.TicksPerDay = 4 })
			},
			rule: "inexact_hour", path: []string{"calendar", "ticks_per_day"}, fires: false,
		},
		{
			// A tick of 5 days over seasons of 3 and 362: the first is shorter.
			name: "season_shorter_than_tick fires for a season below a tick",
			mutate: func(c *economysim.Config) {
				calendarOf(c, func(cal *economysim.Calendar) {
					cal.DaysPerTick = 5
					cal.Seasons = []economysim.Season{{ID: "spring", Days: 3}, {ID: "winter", Days: 362}}
				})
			},
			rule: "season_shorter_than_tick", path: []string{"calendar", "seasons"}, fires: true,
		},
		{
			name: "season_shorter_than_tick is not raised at a tick of 1 day",
			mutate: func(c *economysim.Config) {
				c.Calendar.Seasons = []economysim.Season{{ID: "spring", Days: 3}, {ID: "winter", Days: 362}}
			},
			rule: "season_shorter_than_tick", path: []string{"calendar", "seasons"}, fires: false,
		},
		{
			name: "tool_unaffordable fires below the price of the tool",
			mutate: func(c *economysim.Config) {
				roleOf(c, "cook").StartMoney = 17_104
			},
			rule: "tool_unaffordable", path: []string{"roles", "cook", "start_money"}, fires: true,
		},
		{
			name: "tool_unaffordable is not raised at the price of the tool",
			mutate: func(c *economysim.Config) {
				roleOf(c, "cook").StartMoney = 17_105
			},
			rule: "tool_unaffordable", path: []string{"roles", "cook", "start_money"}, fires: false,
		},
		{
			name: "entry_share_window is not raised at 2 500",
			mutate: func(c *economysim.Config) {
				c.World.EntryShare = 20 // 20 * 120 = 2 400
			},
			rule: "entry_share_window", path: []string{"world", "entry_share"}, fires: false,
		},
		{
			name: "cold_start_lockstep is not raised with a job row",
			mutate: func(c *economysim.Config) {
				// grow_staple is spring only, so a job on it has no virtual start offset at
				// tick 0 (phase 6, ErrWarmStartJob); save_seed runs all year.
				c.WarmStart.Groups[3].Jobs = []economysim.JobSpec{{Recipe: "save_seed", InProgressBP: 100}}
			},
			rule: "cold_start_lockstep", path: []string{"roles", "grower", "recipes"}, fires: false,
		},
		{
			name: "cold_start_lockstep fires with no warm start at all",
			mutate: func(c *economysim.Config) {
				c.WarmStart = nil
			},
			rule: "cold_start_lockstep", path: []string{"roles", "maker", "recipes"}, fires: true,
		},
		{
			name: "cold_start_lockstep is not raised for a group of one agent",
			mutate: func(c *economysim.Config) {
				c.WarmStart = nil
				c.Groups[3].Households = 1
			},
			rule: "cold_start_lockstep", path: []string{"roles", "grower", "recipes"}, fires: false,
		},
		{
			// The only recipe with no inputs that makes fuel is cut_fuel; with no
			// role that lists it, the raw marker stands on nothing.
			name: "raw_without_source fires when no listed source makes the good",
			mutate: func(c *economysim.Config) {
				roleOf(c, "forester").Recipes = nil
			},
			rule: "raw_without_source", path: []string{"commodities", "fuel"}, fires: true,
		},
		{
			name: "raw_without_source is not raised when a source runs",
			mutate: func(c *economysim.Config) {
			},
			rule: "raw_without_source", path: []string{"commodities", "fuel"}, fires: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			got := c.Warnings()
			if has := hasWarning(got, tt.rule, tt.path...); has != tt.fires {
				t.Errorf("Warnings(%s) has %s at %v = %v, want %v; warnings: %v", tt.name, tt.rule, tt.path, has, tt.fires, got)
			}
			err := c.Validate()
			if tt.invalid && err == nil {
				t.Errorf("Validate(%s) = nil, want an error: the case is a configuration that is rejected", tt.name)
			}
			if !tt.invalid && err != nil {
				t.Errorf("Validate(%s) = %v, want nil: a warning never fails validation", tt.name, err)
			}
		})
	}
}

// TestWarnings_AreSortedAndStable verifies that the warnings of one
// configuration come in order of path then rule, the same on every call, and
// that two warnings at one path are ordered by rule.
//
// Requirement: FR-CFG-012, FR-CFG-018
func TestWarnings_AreSortedAndStable(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	c.World.Upkeep = nil
	c.World.CostFloor = 10_000
	for i := range c.Roles {
		c.Roles[i].Closed = true
	}
	first := c.Warnings()
	if !sortedWarningsEqual(first, c.Warnings()) {
		t.Errorf("Warnings differ between two calls: %v and %v", first, c.Warnings())
	}
	for i := 1; i < len(first); i++ {
		a, b := first[i-1], first[i]
		cmp := slices.Compare(a.Path, b.Path)
		if cmp > 0 || (cmp == 0 && a.Rule > b.Rule) {
			t.Errorf("warning %d (%v %s) comes before warning %d (%v %s), want the order of path then rule", i-1, a.Path, a.Rule, i, b.Path, b.Rule)
		}
	}
	if !hasWarning(first, "no_upkeep", "world", "upkeep") || !hasWarning(first, "cost_floor_at_full_cost", "world", "cost_floor") {
		t.Errorf("Warnings = %v, want no_upkeep and cost_floor_at_full_cost among them", first)
	}
}

// TestWarnings_OnAConfigurationThatDoesNotValidate verifies that Warnings does
// not panic and still returns the warnings of a configuration that is broken in
// many ways: unknown references, a zero value, and values outside every range.
//
// Guard: NewWorld calls Warnings only after Validate succeeds, but a consumer
// may call it on anything (spec-config.md 7.2). A nil lookup or a division by a
// zero field in a warning would panic there.
func TestWarnings_OnAConfigurationThatDoesNotValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		config economysim.Config
	}{
		{"the zero value", economysim.Config{}},
		{"unknown references everywhere", func() economysim.Config {
			c := referenceConfig()
			c.Roles[0].Recipes = []string{"nosuch"}
			c.Roles[0].Upkeep = []economysim.Item{{Commodity: "nosuch", Quantity: -5}}
			c.Groups = append(c.Groups, economysim.Group{Region: "nowhere", Role: "ghost", Households: -1, Scale: -4})
			c.Yields = append(c.Yields, economysim.YieldProfile{Region: "nowhere", Recipe: "nosuch"})
			c.World.MinAgents, c.World.Scale, c.World.WindowCycles = 0, 0, math.MaxInt64
			c.Calendar.TicksPerDay, c.Calendar.HoursPerDay = 0, 0
			return c
		}()},
		{"values at the edges of int64", func() economysim.Config {
			c := referenceConfig()
			c.Recipes[0].Duration = math.MaxInt64
			c.Roles[1].StartMoney = math.MaxInt64
			c.Commodities[0].ReferencePrice = math.MaxInt64
			c.World.EntryShare = math.MaxInt64
			return c
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_ = tt.config.Warnings() // must not panic
			_ = tt.config.Validate()
		})
	}
}

// intFields returns the settable integer fields of a struct value, in the order
// of their declaration, so a fuzz input can pick one by index.
func intFields(v reflect.Value) []reflect.Value {
	var out []reflect.Value
	for i := 0; i < v.NumField(); i++ {
		if f := v.Field(i); f.Kind() == reflect.Int64 {
			out = append(out, f)
		}
	}
	return out
}

// perturb changes the reference configuration by one of a few primitives: kind
// picks the section, field the integer field of its first entity, value the
// number it is set to, and count how many entities of a kind are added. It is
// the one place a fuzz input becomes a configuration, so a failing input is a
// handful of numbers.
func perturb(c *economysim.Config, kind, field uint8, value int64, count uint8) {
	pick := func(v reflect.Value) {
		if fs := intFields(v); len(fs) > 0 {
			fs[int(field)%len(fs)].SetInt(value)
		}
	}
	n := int(count)
	switch kind % 14 {
	case 0:
		pick(reflect.ValueOf(&c.World).Elem())
	case 1:
		pick(reflect.ValueOf(&c.Calendar).Elem())
	case 2:
		if len(c.Calendar.Months) > 0 {
			c.Calendar.Months[int(field)%len(c.Calendar.Months)].Days = value
		}
	case 3:
		if len(c.Calendar.Seasons) > 0 {
			c.Calendar.Seasons[int(field)%len(c.Calendar.Seasons)].Days = value
		}
	case 4:
		pick(reflect.ValueOf(&c.Commodities[int(field)%len(c.Commodities)]).Elem())
	case 5:
		pick(reflect.ValueOf(&c.Recipes[int(field)%len(c.Recipes)]).Elem())
	case 6:
		pick(reflect.ValueOf(&c.Roles[int(field)%len(c.Roles)]).Elem())
	case 7:
		pick(reflect.ValueOf(&c.Regions[0]).Elem())
	case 8:
		pick(reflect.ValueOf(&c.Groups[int(field)%len(c.Groups)]).Elem())
	case 9:
		// The yield profiles: a factor of a profile or of one of its seasons.
		y := &c.Yields[int(field)%len(c.Yields)]
		if len(y.Seasons) > 0 && field%2 == 0 {
			y.Seasons[0].BP = value
		} else {
			y.All = value
		}
	case 10:
		// More groups, over unknown regions and roles as the field says.
		for i := 0; i < n; i++ {
			g := economysim.Group{Region: "north", Role: "cook", Households: value}
			if field%2 == 1 {
				g.Region = "r" + strconv.Itoa(i)
			}
			c.Groups = append(c.Groups, g)
		}
	case 11:
		// More commodities with the same or a changed ID: past MaxCommodities
		// for count above 200.
		for i := 0; i < n*5; i++ {
			id := "x"
			if field%2 == 1 {
				id += strconv.Itoa(i)
			}
			c.Commodities = append(c.Commodities, economysim.Commodity{ID: id, ReferencePrice: economysim.Price(value)})
		}
	case 12:
		for i := 0; i < n; i++ {
			c.Recipes = append(c.Recipes, economysim.Recipe{ID: "r" + strconv.Itoa(i%7), Duration: value,
				Outputs: []economysim.Output{{Commodity: "fuel", Quantity: economysim.Quantity(value), CostShare: value}}})
		}
	case 13:
		if w := c.WarmStart; w != nil {
			switch field % 4 {
			case 0:
				w.Prices[int(field)%len(w.Prices)].Price = economysim.Price(value)
			case 1:
				w.Wages[0].Wage = economysim.Money(value)
			case 2:
				w.Groups[int(field)%len(w.Groups)].Money = economysim.Money(value)
			default:
				for i := 0; i < n; i++ {
					w.Pools = append(w.Pools, economysim.WarmPool{Region: "north", Money: economysim.Money(value)})
				}
			}
		}
	}
}

// FuzzValidate verifies that Validate and Warnings never panic on a
// configuration that is the reference with one section perturbed, that at most
// 100 errors and one ErrTooManyErrors come back, that every error is a
// *ValidationError, a *ViabilityError of ErrNotViable or ErrTooManyErrors, that
// equal input gives equal output and that the receiver is not changed. A
// perturbation that leaves the structural phases clean runs phase 6 as well. The calendar's rules and the yield profile's
// are run through it by the section selectors 1 to 3 and 9.
//
// Requirement: NFR-CFG-002, NFR-CFG-003, NFR-CAL-002, NFR-PRO-004
func FuzzValidate(f *testing.F) {
	seeds := []struct {
		kind, field uint8
		value       int64
		count       uint8
	}{
		{0, 0, 0, 0}, {0, 5, -1, 0}, {0, 40, math.MaxInt64, 0}, {1, 0, 0, 0}, {1, 3, math.MinInt64, 0},
		{2, 1, 0, 0}, {2, 1, 100_000, 0}, {3, 2, -5, 0}, {4, 0, 0, 0}, {4, 4, math.MaxInt64, 0},
		{5, 1, 0, 0}, {6, 4, 9_999, 0}, {6, 7, math.MinInt64, 0}, {7, 1, 5, 0}, {8, 0, 0, 0},
		{9, 0, 100_001, 0}, {9, 1, -1, 0}, {10, 0, 1, 120}, {10, 1, 1, 120}, {11, 0, 1, 250},
		{11, 1, 100, 10}, {12, 0, 5, 20}, {13, 0, 0, 0}, {13, 1, math.MaxInt64, 0}, {13, 3, math.MaxInt64, 12},
		// Structurally clean, so phase 6 runs: a yield of 0 and a short year of
		// seasons (the viability rules fire), a shelf life under a supply gap, and
		// a recipe that is longer than the pack was written for.
		{9, 2, 0, 0}, {9, 0, 0, 0}, {4, 2, 1, 0}, {4, 2, 100, 0}, {5, 0, 1_000, 0}, {5, 0, 2, 0}, {8, 2, 1, 0},
	}
	for _, s := range seeds {
		f.Add(s.kind, s.field, s.value, s.count)
	}
	f.Fuzz(func(t *testing.T, kind, field uint8, value int64, count uint8) {
		build := func() economysim.Config {
			c := referenceConfig()
			perturb(&c, kind, field, value, count)
			return c
		}
		c := build()
		err := c.Validate()
		again := build()
		err2 := again.Validate()
		if (err == nil) != (err2 == nil) || (err != nil && err.Error() != err2.Error()) {
			t.Fatalf("Validate(kind %d field %d value %d count %d) is not deterministic:\n%v\n%v", kind, field, value, count, err, err2)
		}
		if !reflect.DeepEqual(c, build()) {
			t.Errorf("Validate(kind %d field %d value %d count %d) changed its receiver", kind, field, value, count)
		}
		if err != nil {
			// violations puts a *ViabilityError among the others: it is not a
			// *ValidationError.
			vs, other := violations(err)
			cuts, via := 0, 0
			for _, o := range other {
				var ve *economysim.ViabilityError
				switch {
				case errors.Is(o, economysim.ErrTooManyErrors):
					cuts++
				case errors.As(o, &ve) && errors.Is(o, economysim.ErrNotViable) && ve.Rule != "" && len(ve.Path) > 0 && ve.Error() != "":
					via++
				default:
					t.Errorf("Validate returned %v that is neither a *ValidationError, a *ViabilityError nor ErrTooManyErrors", o)
				}
			}
			if len(vs)+via > 100 || cuts > 1 {
				t.Errorf("Validate(kind %d field %d value %d count %d) = %d errors and %d cuts, want at most 100 and 1", kind, field, value, count, len(vs)+via, cuts)
			}
			for _, v := range vs {
				if v.Rule == nil || len(v.Path) == 0 || v.Error() == "" {
					t.Errorf("Validate returned an error with no rule or no path: %+v", v)
				}
			}
		}
		// Warnings: no panic, in order, and the same twice.
		ws := c.Warnings()
		if !sortedWarningsEqual(ws, c.Warnings()) {
			t.Errorf("Warnings(kind %d field %d value %d count %d) differ between two calls", kind, field, value, count)
		}
		for i := 1; i < len(ws); i++ {
			if cmp := slices.Compare(ws[i-1].Path, ws[i].Path); cmp > 0 || (cmp == 0 && ws[i-1].Rule > ws[i].Rule) {
				t.Errorf("Warnings out of order at %d: %v then %v", i, ws[i-1], ws[i])
			}
		}
	})
}

// BenchmarkValidate_Reference measures Validate and Warnings on the reference
// configuration: the cost of the phases for a pack of the size of `one`
// (NFR-CFG-001 is measured at the bounds of 4.1 in the slice).
func BenchmarkValidate_Reference(b *testing.B) {
	c := referenceConfig()
	b.ReportAllocs()
	for b.Loop() {
		if err := c.Validate(); err != nil {
			b.Fatalf("Validate(reference) = %v, want nil", err)
		}
	}
}

// BenchmarkWarnings_Reference measures Warnings on the reference configuration.
func BenchmarkWarnings_Reference(b *testing.B) {
	c := referenceConfig()
	b.ReportAllocs()
	for b.Loop() {
		if len(c.Warnings()) == 0 {
			b.Fatal("Warnings(reference) = none, want the reference warnings")
		}
	}
}

// TestValidate_ValueAtTheBoundIsValid verifies that a value exactly at the bound
// that the failing cases break by one is accepted, so no rule is off by one.
// Each case states the arithmetic that puts it on the bound.
//
// Requirement: FR-CFG-009, FR-CFG-051, FR-CFG-052, FR-CFG-055, FR-WEL-025, FR-VIA-030
func TestValidate_ValueAtTheBoundIsValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
	}{
		{"seed price at the resolution floor", func(c *economysim.Config) {
			// ceil(10 000 / 200) = 50, and 50 is inside the band of 179.
			c.WarmStart.Prices[0].Price = 50
		}},
		{"lowest target of exactly 10 000", func(c *economysim.Config) {
			// 15 000 * (10 000 - 3 333) / 10 000 = 10 000.5, truncated to 10 000.
			roleOf(c, "cook").TargetSpread = 3_333
		}},
		{"work floor of exactly the expected work", func(c *economysim.Config) {
			// 10 000 * 365 = 365 * 10 000.
			r := roleOf(c, "cook")
			r.MinWorkShare, r.ExpectedWorkTicks = 10_000, 365
		}},
		{"wage times expected work of exactly MaxMoney", func(c *economysim.Config) {
			// 4 * 10^15 * 250 = 10^18; the longest recipe, 60 ticks, is 2.4 * 10^17.
			c.WarmStart.Wages[0].Wage = 4_000_000_000_000_000
		}},
		{"effective window of exactly MaxWelfareWindow", func(c *economysim.Config) {
			// 2 cycles of 1 825 ticks is 3 650. The job is 30 times longer than the
			// 60 ticks it was written for, so its output is scaled to keep the chain
			// productive (V3): 1 800 * 1 825 / 60 = 54 750.
			recipeOf(c, "grow_staple").Duration = 1_825
			recipeOf(c, "grow_staple").Outputs[0].Quantity = 54_750
		}},
		{"belief floor met exactly", func(c *economysim.Config) {
			// 50 is the floor at BeliefStep 200; seed (tradable) is at 50 and the
			// warm start's seed price for it is moved with it.
			commodityOf(c, "seed").ReferencePrice = 50
			for i := range c.WarmStart.Prices {
				if c.WarmStart.Prices[i].Commodity == "seed" {
					c.WarmStart.Prices[i].Price = 50
				}
			}
		}},
		{"seed price of exactly MaxPrice", func(c *economysim.Config) {
			// The reference price is at the bound too, so the price is inside the
			// reference band.
			commodityOf(c, "seed").ReferencePrice = economysim.MaxPrice
			for i := range c.WarmStart.Prices {
				if c.WarmStart.Prices[i].Commodity == "seed" {
					c.WarmStart.Prices[i].Price = economysim.MaxPrice
				}
			}
		}},
		{"lowest-priced untradable commodity below the floor", func(c *economysim.Config) {
			// waste is only a discard output: no role trades it, so the floor
			// does not apply (spec-goods.md 6.4).
			commodityOf(c, "waste").ReferencePrice = 1
		}},
		{"group of weight exactly MaxWeight", func(c *economysim.Config) {
			// 10 000 000 households at scale 10 000 000: n = 1 raised to K = 10,
			// weight 1 000 000.
			c.Groups[1].Households, c.Groups[1].Scale = 10_000_000, 10_000_000
			c.WarmStart.Groups[1].Stock = nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			if err := c.Validate(); err != nil {
				t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
			}
		})
	}
}
