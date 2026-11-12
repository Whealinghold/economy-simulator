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

// DefaultCrew is the households a carrying unit of a self-carried transport mode
// needs when the file gives no crew (spec-transport.md 3.1). The file loader
// applies it to a mode without a haul commodity; a Config built in Go sets
// Mode.Crew itself.
const DefaultCrew = 1

// DefaultSubsistence returns the subsistence module with the placeholder
// tunables of spec-subsistence.md 12: the mode starts below 2 secure ticks, ends
// at a cover of 3 and retries a role job after 10 own-use jobs. It has no own-use
// recipe, so a pack names its own.
func DefaultSubsistence() SubsistenceConfig {
	return SubsistenceConfig{EnterTicks: 2, ExitTicks: 3, RetryJobs: 10}
}

// DefaultPopulation returns the population module with the placeholder defaults
// of spec-population.md 8: a flat birth schedule of 250 basis points a year and a
// death schedule that falls from 450 to 50 between the ratios 8 000 and 16 000,
// which cross at 12 000; one household kept per group; a full StartMoney stake.
// None is the rate of any society (N-015, N-027).
func DefaultPopulation() Population {
	return Population{
		BirthSchedule: []Knot{{Ratio: 8_000, Rate: 250}, {Ratio: 16_000, Rate: 250}},
		DeathSchedule: []Knot{{Ratio: 8_000, Rate: 450}, {Ratio: 16_000, Rate: 50}},
		MinHouseholds: 1,
		BirthStake:    10_000,
	}
}

// DefaultRegionPopulation returns the row of a region that has none: both scales
// 10 000 (the schedule as written) and no household cap.
func DefaultRegionPopulation(region string) RegionPopulation {
	return RegionPopulation{Region: region, BirthScale: 10_000, DeathScale: 10_000}
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

// DefaultInformation returns the information module with the placeholder
// tunables of spec-information.md 8: an error of 500 basis points (5%) and a
// horizon of 60 ticks. InfoSpeed has no default because a speed in a unit the
// pack chooses has none that is right for every setting, so the value does not
// validate until the caller sets it.
func DefaultInformation() Information {
	return Information{RumourErrorBP: 500, RumourMaxAge: 60}
}

// DefaultMigration returns the migration module with the placeholder tunables of
// spec-migration.md 11 (N-015, N-027): a review every 30 ticks, a gate at a
// ratio of 1.0, a margin of 1 000 basis points spread over 5 000, a cooldown of
// 60 ticks, a horizon of 180 ticks and caps of 50 and 100 basis points.
// MoveSpeed has no default because a speed in the unit of the pack's distances
// has none that is right for every setting, so the value does not validate until
// the caller sets it.
func DefaultMigration() Migration {
	return Migration{
		MoveReviewTicks: 30, MoveBelow: wholeBP, MoveGain: 1_000, MoveWidth: 5_000,
		MoveCooldown: 60, MoveHorizon: 180, MoveOutShare: 50, MoveInShare: 100,
	}
}

// DefaultMerchantFeeShare is the default Role.MerchantFeeShare, in basis points:
// a merchant lets the cost of carriage take half of the expected margin
// (spec-carriage.md 2.2, FR-CAR-012). The Go zero value of the field is a share of
// 0, which never ships (spec-merchants.md 13); the loader applies this default to
// a role that has a merchant block and no key. A placeholder (N-015, N-027).
const DefaultMerchantFeeShare = 5_000

// DefaultMerchants returns the delivery contract module with the placeholder
// tunable of spec-merchants.md 13: at most 1 000 contracts open (N-015, N-027).
func DefaultMerchants() Merchants { return Merchants{MaxOpen: 1_000} }

// DefaultMerchant returns a merchant block with the placeholder tunables of
// spec-merchants.md 13 (N-015, N-027): a lot of one carrying unit, a fee share
// spread of 2 500, a rumour acted on up to 2 ticks old, a tick away costing one
// tick of upkeep, a consignee cut of 500 and a take margin of 1 000. Modes and
// TakeUnits have no default, so the value does not validate until the caller
// sets Modes.
func DefaultMerchant() Merchant {
	return Merchant{LotUnits: 1, FeeShareSpreadBP: 2_500, MaxSourceAge: 2, TimeCostBP: wholeBP, ConsigneeCutBP: 500, TakeMarginBP: 1_000}
}

// The placeholder defaults of the external module (spec-external-sector.md 13.1;
// N-015, N-027): the clamps of the trend and walk factor, a tenth and ten times
// Base, and the neutral gate scale.
const (
	// DefaultFloorBP is the default OutsideLine.FloorBP.
	DefaultFloorBP = 1_000
	// DefaultCeilingBP is the default OutsideLine.CeilingBP.
	DefaultCeilingBP = 100_000
	// DefaultGateScaleBP is the default OutsideGate.ScaleBP.
	DefaultGateScaleBP = wholeBP
)

// DefaultOutsideLine returns a line of the commodity at the base price with the
// placeholder clamps of spec-external-sector.md 13.1 and no side, no season, no
// trend and no walk: a fixed price. A side is added by setting its Capacity.
func DefaultOutsideLine(commodity string, base Price) OutsideLine {
	return OutsideLine{Commodity: commodity, Base: base, FloorBP: DefaultFloorBP, CeilingBP: DefaultCeilingBP}
}

// DefaultOutsideGate returns a gate row at the neutral scale.
func DefaultOutsideGate(region string) OutsideGate {
	return OutsideGate{Region: region, ScaleBP: DefaultGateScaleBP}
}

// DefaultEmployer returns the employer block with the placeholder share of
// spec-employment.md 24.3.2: 5 000 basis points of the receipts (N-015, N-027).
func DefaultEmployer() Employer { return Employer{PayrollBP: 5_000} }
