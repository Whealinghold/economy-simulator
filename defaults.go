package economysim

// The defaults are defined once, here (architecture.md 9.8). Each constructor
// returns a fully populated value that the caller owns; no value is shared
// between calls. Every default is a provisional placeholder of its owner spec
// (N-015, N-027), not a setting: setting-specific values belong in a pack. The
// zero value of an entity is not a valid default (spec-config.md 5.2).

// DefaultWorldParams returns the world-level parameters with the defaults of their
// owner specs. Upkeep is empty and Spawner and Clearing name the built-in
// "default" and "average".
func DefaultWorldParams() WorldParams {
	return WorldParams{
		Scale:               1,
		MinAgents:           10,
		BulkFactor:          10_000,
		WelfareWindow:       30,
		WindowCycles:        2,
		CoverTicks:          3,
		InputJobs:           1,
		OutputCoverTicks:    1,
		FavourFloor:         2_000,
		ReserveTicks:        3,
		SpendMaxShare:       5_000,
		SpendHalfTicks:      10,
		RestBand:            500,
		ProduceHaltRatio:    30_000,
		BasketTolerance:     1_000,
		BeliefStep:          200,
		BeliefAdjustProb:    5_000,
		BeliefSignificance:  500,
		BeliefStockLow:      5_000,
		BeliefStockHigh:     15_000,
		BeliefMaxAge:        3,
		BeliefWidthInit:     2_000,
		BeliefWidthMin:      200,
		BeliefWidthMax:      5_000,
		BeliefWidthStep:     250,
		BeliefEvidenceShare: 5_000,
		CostFloor:           7_000,
		CostCeiling:         30_000,
		RefBandLow:          1_000,
		RefBandHigh:         100_000,
		PriceWindow:         15,
		Spawner:             "default",
		ReviewTicks:         10,
		EntryShare:          50,
		RoleHysteresis:      1_000,
		RoleCooldown:        30,
		EntryRatio:          8_000,
		ChangePrivation:     3,
		RemovePrivation:     10,
		DeathStep:           1_000,
		ExitShare:           100,
		Renewal:             10_000,
		EntryPoolShare:      5_000,
		Clearing:            "average",
	}
}

// DefaultCalendar returns the Earth calendar of spec-calendar.md 3.1: 24-hour
// days, 7-day weeks, twelve months of 365 days in all with neutral IDs m01 to
// m12, four seasons beginning on day 59, one tick a day, no leap day.
func DefaultCalendar() Calendar {
	days := [...]int64{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	months := []Month{
		{ID: "m01"}, {ID: "m02"}, {ID: "m03"}, {ID: "m04"}, {ID: "m05"}, {ID: "m06"},
		{ID: "m07"}, {ID: "m08"}, {ID: "m09"}, {ID: "m10"}, {ID: "m11"}, {ID: "m12"},
	}
	for i := range months {
		months[i].Days = days[i]
	}
	return Calendar{
		HoursPerDay: 24,
		DaysPerWeek: 7,
		Months:      months,
		TicksPerDay: 1,
		DaysPerTick: 1,
		StartDay:    0,
		Seasons: []Season{
			{ID: "spring", Days: 92},
			{ID: "summer", Days: 92},
			{ID: "autumn", Days: 91},
			{ID: "winter", Days: 90},
		},
		SeasonStart: 59,
	}
}

// DefaultCommodity returns a commodity of one mass unit and one bulk unit that
// never spoils or wears and is not raw. ID and ReferencePrice have no default
// and are required (spec-goods.md 8).
func DefaultCommodity() Commodity {
	return Commodity{Mass: 1, Bulk: 1}
}

// DefaultRecipe returns a recipe of one tick whose trainee yield is 5 000 basis
// points. ID and Outputs have no default and are required
// (spec-production.md 11).
func DefaultRecipe() Recipe {
	return Recipe{Duration: 1, TraineeYield: 5_000}
}

// DefaultRole returns a role with the defaults of spec-agents.md 12,
// spec-welfare-and-wages.md 12.1 and spec-skill-barriers.md: the "default"
// strategy, 10 000 starting money, a target ratio of 15 000 basis points and
// 250 expected working ticks. ID has no default and is required.
func DefaultRole() Role {
	return Role{
		StartMoney:        10_000,
		Strategy:          "default",
		TargetRatio:       15_000,
		DefaultRatio:      10_000,
		ExpectedWorkTicks: 250,
		MinWorkShare:      4_000,
		MaxTrainees:       3,
		TeachingLoad:      1_000,
	}
}

// DefaultRegion returns a region with a reference basket of 1 000 minor units,
// a capital return of 1 000 basis points and an estate markdown of 500 basis
// points, and, for the estates module, a pool release of 100 basis points and a
// relief line of 10 000 (the pool fields are inert and unhashed while the module
// is off, spec-config.md 8.2.3). ID has no default and is required.
func DefaultRegion() Region {
	return Region{
		ReferenceBasket: 1_000,
		CapitalReturn:   1_000,
		EstateMarkdown:  500,
		PoolRelease:     100,
		ReliefBelow:     10_000,
	}
}

// DefaultWant returns a want with a period of one tick, an elasticity and a
// ceiling of 15 000 basis points and a substitution exponent of 2
// (spec-demand.md 9). ID, Items and Quantity have no default and are required.
func DefaultWant() Want {
	return Want{Period: 1, Elasticity: 15_000, Ceiling: 15_000, Substitution: 2}
}

// DefaultWantItem returns a want item of weight 10 000 basis points. Commodity
// has no default and is required.
func DefaultWantItem() WantItem {
	return WantItem{Weight: 10_000}
}

// DefaultTaste returns a taste of scale 10 000 basis points: the want's whole
// quantity. Role and Want have no default and are required. A scale of 0 is a
// row that removes the want.
func DefaultTaste() Taste {
	return Taste{Scale: 10_000}
}

// DefaultRegionTaste returns a region taste of scale 10 000 basis points. Region
// and Want have no default and are required.
func DefaultRegionTaste() RegionTaste {
	return RegionTaste{Scale: 10_000}
}

// DefaultResource returns a constant, finite stock of no capacity that starts
// full and whose plan stock is 6 000 basis points of the capacity
// (spec-resources.md 12). ID has no default and is required.
func DefaultResource() Resource {
	return Resource{Shape: ShapeConstant, InitialBP: 10_000, PlanShareBP: 6_000}
}

// DefaultResources returns the resources module with a viability horizon of 50
// years and no resource (spec-resources.md 12).
func DefaultResources() Resources {
	return Resources{HorizonYears: 50}
}

// The defaults of the solve options (spec-viability.md 11), unexported for the
// reason the bounds are: the options are not Config fields. Provisional
// placeholders (N-015).
const (
	// defaultStockCoverTicks is the ticks of upkeep a household holds at the start.
	defaultStockCoverTicks = 5
	// defaultMoneyBufferTicks is the ticks of upkeep of money held beyond the job.
	defaultMoneyBufferTicks = 10
	// defaultResidualWarnBP is the residual above which an activity is listed.
	defaultResidualWarnBP = 1_000
)
