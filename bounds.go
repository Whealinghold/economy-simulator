package economysim

// The validation bounds of the configuration (spec-config.md 4.1). They are
// constants, not tunables: nothing in a Config can raise them. They exist so
// that arithmetic cannot overflow (architecture.md 3.5) and so that the work of
// validation is bounded. Every value here is provisional until the slice runs
// (N-015, N-061): a value may change in a behaviour-changing release, and none
// is a result of measurement unless its comment says so.
//
// The money and quantity bounds are derived, not chosen: they are the largest
// values for which the products the simulation computes fit in int64.

// Money, price, quantity and count bounds.
const (
	// MaxPrice is the largest price, in minor units. With MaxQuantity a trade
	// fits: 10^9 * 10^9 = 10^18, below the int64 maximum of about 9.2e18.
	MaxPrice = 1_000_000_000
	// MaxQuantity is the most of one commodity in one order, stock, recipe
	// amount or upkeep.
	MaxQuantity = 1_000_000_000
	// MaxMoney is the largest balance, single transfer and total money at the
	// start, in minor units. It equals the largest trade, so one trade never
	// overflows.
	MaxMoney = 1_000_000_000_000_000_000
	// MaxWeight is the most households one agent may hold. A weight times a
	// per-household quantity must still be at most MaxQuantity.
	MaxWeight = 1_000_000
	// MaxHouseholds is the most households in one group, bounded by
	// MaxQuantity so that total upkeep fits.
	MaxHouseholds = 1_000_000_000
	// MaxBasisPoints is the largest ratio field, a factor of 100. A field with
	// a tighter range (a probability is 0 to 10 000) says so in its owner.
	MaxBasisPoints = 1_000_000
	// MaxIDLength is the longest ID, in bytes.
	MaxIDLength = 64
	// MaxLabelBytes is the longest label, in bytes, for every
	// presentation-only text field.
	MaxLabelBytes = 256
)

// Counts of entities, so that loops and validation work are bounded by the
// configuration.
const (
	// MaxCommodities is the most commodities.
	MaxCommodities = 1_000
	// MaxRoles is the most roles.
	MaxRoles = 1_000
	// MaxRegions is the most regions.
	MaxRegions = 1_000
	// MaxRecipes is the most recipes.
	MaxRecipes = 10_000
	// MaxGroups is the most groups.
	MaxGroups = 100_000
	// MaxYields is the most yield profiles: one per region and recipe, so a
	// config holds a bounded table. Like its neighbours it is provisional.
	MaxYields = MaxRegions * MaxRecipes
	// MaxRecipeItems is the most items on one side of a recipe.
	MaxRecipeItems = 16
	// MaxUpkeepItems is the most goods in one upkeep list. Each quantity is at
	// most MaxQuantity / MaxWeight and each price at most MaxPrice, so the cost
	// of a list is at most 1.6e13, far below MaxMoney.
	MaxUpkeepItems = 16
	// MaxRequires is the most tools one recipe requires; it bounds the
	// per-execution work of production.
	MaxRequires = 4
	// MaxSeedPrices is the most seed prices of a warm start: one per region and
	// commodity.
	MaxSeedPrices = MaxRegions * MaxCommodities
	// MaxDistance is the longest edge, in the pack's distance unit. A shortest
	// path has at most MaxRegions - 1 edges and an effective edge is at most
	// MaxDistance * MaxEdgeFactorBP / 10 000, so a path sum stays below 10^13
	// (spec-regions.md 4.2). Provisional (N-027).
	MaxDistance = 1_000_000_000
	// MaxEdges is the most edges: an average out-degree of 20, which bounds the
	// construction work of the region graph (spec-regions.md 4.2). Provisional.
	MaxEdges = 20 * MaxRegions
	// MaxEdgeFactorBP is the largest length factor of an edge status, in basis
	// points: ten times (spec-regions.md 7.2). Provisional.
	MaxEdgeFactorBP = 100_000
	// MaxRegionPrices is the most reference-price overrides: one per region and
	// commodity, derived as MaxSeedPrices is (spec-regions.md 4.2).
	MaxRegionPrices = MaxRegions * MaxCommodities
	// MaxViabilityPivots is the most pivots the load-time viability check and a
	// solve may use (spec-viability.md 11). Bland's rule terminates but may be
	// slow, so this caps the work; a check that reaches it is skipped with a
	// warning, a solve returns ErrSolveLimit. Provisional (N-015, N-061).
	MaxViabilityPivots = 100_000
	// MaxViabilityCells is the most rows times columns of the tableau Validate
	// builds, and the most steps of the cycle-gain enumeration; above it the
	// programmes of V2 and V3 are skipped with a warning (spec-viability.md
	// 4.3). A tableau of 4 000 000 cells is, for example, 1 000 rows by 4 000
	// columns. Provisional and unmeasured (N-015, N-061).
	MaxViabilityCells = 4_000_000
)

// Bounds of the demand module (spec-demand.md 9, spec-config.md 4.1). All are
// provisional (N-015, N-027).
const (
	// MaxWants is the most want entities, as MaxCommodities.
	MaxWants = 1_000
	// MaxWantItems is the most alternatives of one want: the loop over
	// alternatives and the carry state.
	MaxWantItems = 4
	// MaxWantsPerRole is the most taste rows of one role: the per-agent loop and
	// the carry state.
	MaxWantsPerRole = 32
	// MaxTastes is the most taste rows in all; derived, so the list is bounded
	// before the per-role count is read.
	MaxTastes = MaxRoles * MaxWantsPerRole
	// MaxRegionTastes is the most region taste rows in all; derived.
	MaxRegionTastes = MaxRegions * MaxWants
	// MaxWantPeriod is the longest period of a want in ticks, equal to
	// MaxTicksPerYear.
	MaxWantPeriod = 100_000
	// MaxWantRank is the highest rank on a ladder.
	MaxWantRank = 1_000
	// MaxElasticityBP is the steepest slope of a schedule: a slope of ten.
	MaxElasticityBP = 100_000
	// MaxCeilingBP is the highest ceiling of a schedule: five times the base
	// quantity.
	MaxCeilingBP = 50_000
	// MaxSubstitution is the largest integer exponent of the substitution shares.
	MaxSubstitution = 4
	// MaxTasteScale is the largest taste or region scale: ten times.
	MaxTasteScale = 100_000
)

// Calendar and time bounds (spec-calendar.md 7.1).
const (
	// MaxTicksPerYear is the most ticks in a year. TicksPerYear is derived from
	// the calendar and multiplies money and basis points in the calibration
	// formulas; the bound keeps the checked 128-bit helper far from its limit.
	MaxTicksPerYear = 100_000
	// MaxHoursPerDay is the most hours in a calendar day.
	MaxHoursPerDay = 1_000
	// MaxDaysPerWeek is the most days in a week.
	MaxDaysPerWeek = 100
	// MaxMonths is the most months in a year.
	MaxMonths = 100
	// MaxDaysPerMonth is the most days in a month.
	MaxDaysPerMonth = 10_000
	// MaxDaysPerYear is the most days in a year.
	MaxDaysPerYear = 100_000
	// MaxSeasons is the most seasons in a year.
	MaxSeasons = 100
	// MaxTicksPerDay is the most ticks in a calendar day.
	MaxTicksPerDay = 1_000
	// MaxWelfareWindow is the longest effective welfare window of a role, ten
	// Earth years of daily ticks; the window is the length of a per-agent
	// record, so it bounds memory and the window-times-income product.
	MaxWelfareWindow = 3_650
)

// Production bounds (spec-production.md 10).
const (
	// MaxRecipeTicks is the longest job duration; it equals MaxTicksPerYear, so
	// a job of a year is possible.
	MaxRecipeTicks = 100_000
	// MaxYieldBP is the largest yield factor, in basis points: ten times.
	MaxYieldBP = 100_000
)

// Loader limits (spec-config.md 6.3.1). They bound untrusted input before it is
// decoded, and each is checked in the order of the spec. They come from
// measurements of the chosen TOML library (ADR 0009) and are placeholders in
// the sense of N-015: a pack that needs more splits into documents.
const (
	// MaxDocuments is the most documents of one Parse or LoadFS call.
	MaxDocuments = 16
	// MaxDocumentBytes is the largest document: 4 MiB.
	MaxDocumentBytes = 4 << 20
	// MaxMergedBytes is the largest sum of the sizes of the documents of one
	// call: 16 MiB.
	MaxMergedBytes = 16 << 20
	// MaxLineBytes is the longest line of a document: 256 KiB.
	MaxLineBytes = 256 << 10
	// MaxNestingDepth is the most segments of a dotted key or a table header
	// plus the nesting of arrays and inline tables in a value, counted
	// together. It is also the most segments of a path in an error.
	MaxNestingDepth = 16
	// MaxTableKeys is the most direct children of one table of a document; it
	// equals MaxRecipes, the largest table a pack writes.
	MaxTableKeys = MaxRecipes
)

// fieldRange is the documented range of one integer field of a configuration
// struct: the field's name as a path segment (the snake case of the Go name),
// its value and the inclusive range its owner spec gives it. The tables below
// are lookup tables over a value, never mutated; every row cites its owner.
// Like the constants above they are provisional (N-015, N-027).
type fieldRange struct {
	name   string
	value  int64
	lo, hi int64
}

// wholeBP is 10 000 basis points: a ratio of 1.
const wholeBP = 10_000

// The bounds of the solve options (spec-viability.md 11), unexported: an option
// is a function of SolveSteadyState and not a Config field. The bound of
// WithMaxPivots is MaxViabilityPivots. Provisional (N-015).
const (
	// maxStockCoverTicks is the most ticks of upkeep a warm stock may cover.
	maxStockCoverTicks = 1_000
	// maxMoneyBufferTicks is the most ticks of upkeep a warm money may add.
	maxMoneyBufferTicks = 10_000
	// maxResidualWarnBP is the largest residual threshold, in basis points.
	maxResidualWarnBP = 100_000
)

// worldRanges returns the ranges of the integer fields of a World that do not
// depend on another field: spec-agents.md 12, spec-beliefs.md 11,
// spec-spawner.md 11, spec-scale.md 3.1 and 5.2, spec-goods.md 8 and
// spec-welfare-and-wages.md 12.1. The orders between two fields (a low below a
// high) are rules of validate.go.
func worldRanges(w WorldParams) []fieldRange {
	return []fieldRange{
		{"scale", w.Scale, 1, MaxHouseholds},
		{"min_agents", w.MinAgents, 1, MaxHouseholds},
		{"bulk_factor", w.BulkFactor, 1, MaxBasisPoints},
		{"welfare_window", w.WelfareWindow, 1, 1_000},
		{"window_cycles", w.WindowCycles, 1, 100},
		{"cover_ticks", w.CoverTicks, 1, 1_000},
		{"input_jobs", w.InputJobs, 1, 100},
		{"output_cover_ticks", w.OutputCoverTicks, 0, 1_000},
		{"favour_floor", w.FavourFloor, 0, wholeBP},
		{"reserve_ticks", w.ReserveTicks, 0, 1_000},
		{"spend_max_share", w.SpendMaxShare, 0, wholeBP},
		{"spend_half_ticks", w.SpendHalfTicks, 1, 10_000},
		{"rest_band", w.RestBand, 0, wholeBP},
		{"produce_halt_ratio", w.ProduceHaltRatio, wholeBP, MaxBasisPoints},
		{"basket_tolerance", w.BasketTolerance, 0, wholeBP},
		{"belief_step", w.BeliefStep, 1, 2_500},
		{"belief_adjust_prob", w.BeliefAdjustProb, 1, wholeBP},
		{"belief_significance", w.BeliefSignificance, 0, 5_000},
		{"belief_stock_low", w.BeliefStockLow, 1, MaxBasisPoints},
		{"belief_stock_high", w.BeliefStockHigh, 1, MaxBasisPoints},
		{"belief_max_age", w.BeliefMaxAge, 0, 1_000},
		{"belief_width_init", w.BeliefWidthInit, 1, 9_999},
		{"belief_width_min", w.BeliefWidthMin, 1, 9_999},
		{"belief_width_max", w.BeliefWidthMax, 1, 9_999},
		{"belief_width_step", w.BeliefWidthStep, 1, 9_999},
		{"belief_evidence_share", w.BeliefEvidenceShare, 1, wholeBP},
		{"cost_floor", w.CostFloor, 1, MaxBasisPoints},
		{"cost_ceiling", w.CostCeiling, 1, MaxBasisPoints},
		{"ref_band_low", w.RefBandLow, 1, wholeBP},
		{"ref_band_high", w.RefBandHigh, wholeBP, MaxBasisPoints},
		{"price_window", w.PriceWindow, 1, 1_000},
		{"review_ticks", w.ReviewTicks, 1, 1_000},
		{"entry_share", w.EntryShare, 0, wholeBP},
		{"role_hysteresis", w.RoleHysteresis, 0, MaxBasisPoints},
		{"role_cooldown", w.RoleCooldown, 0, MaxWelfareWindow},
		{"entry_ratio", w.EntryRatio, 0, MaxBasisPoints},
		{"change_privation", w.ChangePrivation, 1, 1_000},
		{"remove_privation", w.RemovePrivation, 1, 1_000},
		{"death_step", w.DeathStep, 1, wholeBP},
		{"exit_share", w.ExitShare, 0, wholeBP},
		{"renewal", w.Renewal, 0, wholeBP},
		{"entry_pool_share", w.EntryPoolShare, 0, wholeBP},
		{"anchor_hold", w.AnchorHold, 0, wholeBP},
	}
}

// commodityRanges returns the ranges of the integer fields of a Commodity
// (spec-goods.md 8). The mass and bulk bounds and the shelf life bound are the
// provisional figures of that table.
func commodityRanges(c Commodity) []fieldRange {
	return []fieldRange{
		{"mass", c.Mass, 0, 1_000_000},
		{"bulk", c.Bulk, 0, 1_000_000},
		{"shelf_life", c.ShelfLife, 0, 10_000_000},
		{"durability", c.Durability, 0, MaxQuantity},
		{"reference_price", int64(c.ReferencePrice), 1, MaxPrice},
	}
}

// maxClasses is the most classes of a commodity (spec-goods.md 6.1).
const maxClasses = 8

// recipeRanges returns the ranges of the integer fields of a Recipe
// (spec-production.md 10.2).
func recipeRanges(r Recipe) []fieldRange {
	return []fieldRange{
		{"duration", r.Duration, 1, MaxRecipeTicks},
		{"trainee_yield", r.TraineeYield, 1, wholeBP},
	}
}

// roleRanges returns the ranges of the integer fields of a Role that do not
// depend on the calendar (spec-agents.md 12, spec-welfare-and-wages.md 12.1,
// spec-skill-barriers.md 5). ExpectedWorkTicks and MinWorkShare depend on the
// derived TicksPerYear and are rules of validate.go.
func roleRanges(r Role) []fieldRange {
	return []fieldRange{
		{"start_money", int64(r.StartMoney), 0, MaxMoney / MaxWeight},
		{"target_ratio", r.TargetRatio, wholeBP, MaxBasisPoints},
		{"target_spread", r.TargetSpread, 0, wholeBP - 1},
		{"default_ratio", r.DefaultRatio, 0, MaxBasisPoints},
		{"in_kind", r.InKind, 0, wholeBP},
		{"training_ticks", r.TrainingTicks, 0, 1_000_000},
		{"max_trainees", r.MaxTrainees, 1, 100},
		{"teaching_load", r.TeachingLoad, 0, wholeBP},
	}
}

// regionRanges returns the ranges of the integer fields of a Region
// (spec-currency.md 5, spec-skill-barriers.md 5, spec-estates.md 3).
func regionRanges(r Region) []fieldRange {
	return []fieldRange{
		{"reference_basket", int64(r.ReferenceBasket), 0, MaxMoney},
		{"capital_return", r.CapitalReturn, 0, MaxBasisPoints},
		{"estate_markdown", r.EstateMarkdown, 0, wholeBP},
	}
}

// groupRanges returns the ranges of the integer fields of a Group
// (spec-scale.md 3.1, 5.2 and spec-config.md 4.1).
func groupRanges(g Group) []fieldRange {
	return []fieldRange{
		{"households", g.Households, 1, MaxHouseholds},
		{"scale", g.Scale, 0, MaxHouseholds},
	}
}
