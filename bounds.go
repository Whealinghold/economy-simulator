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
//
// The pool fields belong to the estates module and are ranged only while it is
// on (estatesOn), so a value outside the bound with the module off is no error
// (spec-config.md 8.2.1).
func regionRanges(r Region, estatesOn bool) []fieldRange {
	rs := []fieldRange{
		{"reference_basket", int64(r.ReferenceBasket), 0, MaxMoney},
		{"capital_return", r.CapitalReturn, 0, MaxBasisPoints},
		{"estate_markdown", r.EstateMarkdown, 0, wholeBP},
	}
	if estatesOn {
		rs = append(rs,
			fieldRange{"pool_release", r.PoolRelease, 0, wholeBP},
			fieldRange{"relief_below", r.ReliefBelow, 0, MaxBasisPoints})
	}
	return rs
}

// groupRanges returns the ranges of the integer fields of a Group
// (spec-scale.md 3.1, 5.2 and spec-config.md 4.1).
func groupRanges(g Group) []fieldRange {
	return []fieldRange{
		{"households", g.Households, 1, MaxHouseholds},
		{"scale", g.Scale, 0, MaxHouseholds},
	}
}

// Bounds of the money supply module (spec-money-supply.md 11.2, spec-config.md
// 4.1). All are provisional (N-015, N-027).
const (
	// MaxMoneyFlows is the most money flows: the step loops over the active flows
	// and their classes each tick, and it divides MaxFlowTotal.
	MaxMoneyFlows = 64
	// MaxFlowAmount is the largest Amount, Level or Exempt: for the largest
	// weight, w * Amount is at most MaxMoney.
	MaxFlowAmount = MaxMoney / MaxWeight
	// MaxFlowRateBP is the largest RateBP: the whole balance above the exemption
	// per period.
	MaxFlowRateBP = 10_000
	// MaxFlowScale is the largest flow scale in basis points: ten times the
	// configured value, so a numerator stays inside 128 bits.
	MaxFlowScale = 100_000
	// MaxFlowTotal is the largest effective limit of a flow: all flows together
	// create at most MaxMoney, so the supply stays inside int64.
	MaxFlowTotal = MaxMoney / MaxMoneyFlows
)

// Bounds of the resources module (spec-resources.md 4.6, spec-config.md 4.1).
// All are provisional (N-015, N-027). The largest regeneration numerator is
// MaxRegenBP * MaxEndowment * (MaxRegenScaleBP * MaxShockBP / 10_000) *
// MaxSeasonRegenBP, about 10^31, held in 128 bits; the largest denominator is
// 10^12 * MaxTicksPerYear, below int64.
const (
	// MaxResources is the most resources in the module.
	MaxResources = 256
	// MaxEndowments is the most rows of the endowment table.
	MaxEndowments = MaxRegions * MaxResources
	// MaxEndowment is the largest capacity, starting stock or adjustment of an
	// endowment, in units. It exceeds MaxQuantity because a stock in the ground
	// can exceed any holding.
	MaxEndowment = 1_000_000_000_000_000
	// MaxRegenRatePerYear is the largest rate of a constant stock, in units a
	// year.
	MaxRegenRatePerYear = 1_000_000_000_000_000
	// MaxRegenBP is the largest annual rate of a refill or logistic stock.
	MaxRegenBP = 100_000
	// MaxRegenScaleBP is the largest regeneration scale of an endowment row.
	MaxRegenScaleBP = 100_000
	// MaxSeasonRegenBP is the largest regeneration factor of a season.
	MaxSeasonRegenBP = 100_000
	// MaxShockBP is the largest capacity or regeneration scale of a shock.
	MaxShockBP = 100_000
	// MaxHorizonYears is the largest viability horizon of a finite stock.
	MaxHorizonYears = 1_000
	// MaxKnots is the most knots in one curve: a population schedule or the
	// abundance curve of a resource.
	MaxKnots = 8
	// MaxDraws is the most endowments one recipe draws on.
	MaxDraws = 4
)

// Bounds of the subsistence module (spec-subsistence.md 13.1), provisional
// placeholders (N-015, N-027).
const (
	// MaxOwnUseRecipes is the most entries in one own-use list; the default Work
	// scans the list of a free household once a tick.
	MaxOwnUseRecipes = 8
	// MaxOwnUseTicks is the largest EnterTicks and ExitTicks and the saturation of
	// the secure ticks; the bound of CoverTicks too.
	MaxOwnUseTicks = 1_000
	// MaxRetryJobs is the largest RetryJobs.
	MaxRetryJobs = 1_000
)

// Bounds of the transport module (spec-transport.md 11), provisional
// placeholders (N-015, N-027).
const (
	// MaxModes is the most transport modes; a request and a view loop over them.
	MaxModes = 64
	// MaxSpeed is the largest Mode.Speed; with MaxTransitFactorBP the product of
	// the two fits int64.
	MaxSpeed = 1_000_000_000
	// MaxCapacity is the largest UnitMass and UnitBulk of one carrying unit; times
	// MaxWeight it is 10^18, which fits int64.
	MaxCapacity = 1_000_000_000_000
	// MaxFeed is the most entries in the Feed of a mode.
	MaxFeed = 4
	// MaxHaulSpan is the largest Mode.HaulSpan.
	MaxHaulSpan = 1_000_000_000_000
	// MaxShipmentLines is the most lines in one shipment request; it bounds the
	// loop and the size of an event.
	MaxShipmentLines = 16
	// MaxDispatchPerTick is the most shipment requests one agent's plan may carry.
	MaxDispatchPerTick = 8
	// MaxTravelTicks is the longest journey, a year of ticks; a duration times a
	// count does not overflow.
	MaxTravelTicks = MaxTicksPerYear
	// MaxTransitFactorBP is the largest transit factor, and the cap of the
	// effective factor (a road ten times faster).
	MaxTransitFactorBP = 100_000
	// MaxLanes is the most lanes a solve option may carry: one for each ordered
	// pair of regions and commodity.
	MaxLanes = MaxRegions * MaxRegions * MaxCommodities
)

// Bounds of the information module (spec-information.md 8), provisional
// placeholders (N-015, N-027).
const (
	// MaxInfoSpeed is the largest InfoSpeed; a distance plus it stays far below
	// int64.
	MaxInfoSpeed = 1_000_000_000
	// MaxRumourErrorBP is the largest RumourErrorBP: the factor (10 000 + e) /
	// 10 000 is at least 0.5, so a rumoured price is never 0 or negative.
	MaxRumourErrorBP = 5_000
	// MaxRumourAge is the largest RumourMaxAge, and so the depth of a ring of the
	// price record.
	MaxRumourAge = 10_000
	// MaxRumourCells is the most entries of the price record, sources times
	// RumourMaxAge.
	MaxRumourCells = 4_000_000
	// MaxRumoursPerView is the most candidates of one informed agent; it bounds
	// the loop of a view and the draws it takes.
	MaxRumoursPerView = 4_096
)

const (
	// MaxMoveReviewTicks is the largest MoveReviewTicks of the migration module:
	// a review period of at most a few years (spec-migration.md 11).
	MaxMoveReviewTicks = 1_000
)

const (
	// MaxLotUnits is the largest LotUnits of a merchant role: with MaxQuantity and
	// MaxPrice a lot's product fits int64 through the 128-bit helper, and a lot of
	// more than 64 carrying units is not a lot (spec-merchants.md 13).
	MaxLotUnits = 64
	// MaxContracts is the largest Merchants.MaxOpen: the contracts in flight, and
	// so the Escrow accounts, are at most this many (spec-merchants.md 13).
	MaxContracts = 1_000_000
	// MaxTimeCostBP is the largest TimeCostBP: ten times the upkeep for each tick
	// away, far beyond any use (spec-merchants.md 13).
	MaxTimeCostBP = 100_000
)

const (
	// MaxOutsideNodes is the most outside nodes of the external module: the
	// counterparties and the loops of the module are bounded by it
	// (spec-external-sector.md 13.2).
	MaxOutsideNodes = 16
	// MaxOutsideLines is the most lines of one outside node.
	MaxOutsideLines = 64
	// MaxCounterparties is the most counterparties of the module (one for each
	// node, gate and side): agent IDs and boundary accounts are bounded by it.
	MaxCounterparties = 256
	// MaxSellMarginBP is the largest margin of a sell side: an ask of at most
	// eleven times the mid.
	MaxSellMarginBP = 100_000
	// MaxBuyMarginBP is the largest margin of a buy side: a bid is at least a
	// basis point of the mid.
	MaxBuyMarginBP = 9_999
	// MinQuoteSpreadBP is the smallest sum of the two margins of a line with both
	// sides, so that no loop of trades with the outside earns money.
	MinQuoteSpreadBP = 1
	// MaxPriceFactorBP is the largest seasonal price factor: ten times.
	MaxPriceFactorBP = 100_000
	// MaxTrendBP is the largest TrendBP in either direction: ten times Base a year.
	MaxTrendBP = 100_000
	// MaxFactorBP is the largest CeilingBP: a hundred times Base.
	MaxFactorBP = 1_000_000
	// MaxWalkStepBP is the largest step of the walk: half the mid in one tick is
	// already a jump, not a walk.
	MaxWalkStepBP = 5_000
	// MaxDriftBP is the largest WalkBoundBP: it keeps the walk factor positive.
	MaxDriftBP = 9_999
	// MaxGateScaleBP is the largest gate scale: ten times.
	MaxGateScaleBP = 100_000
	// MaxLineTotal is the largest TotalUnits and TotalValue of a side row. The cap
	// is per side row (spec-external-sector.md 3.3.4) and there are at most
	// MaxCounterparties counterparties of at most MaxOutsideLines rows each, so all
	// the rows together move at most MaxMoney of list value.
	MaxLineTotal = MaxMoney / (MaxCounterparties * MaxOutsideLines)
)

// The bounds of the polity module (spec-polity.md 8). Both are placeholders
// (N-015, spec-polity.md 17.7).
const (
	// MaxJurisdictions is the most jurisdictions: the value of MaxRegions, since a
	// jurisdiction that covers no region draws a warning and more is pointless.
	MaxJurisdictions = MaxRegions
	// MaxPolityDepth is the most levels of a chain of jurisdictions: a chain has at
	// most this many entries and a jurisdiction at most this many less one
	// ancestors.
	MaxPolityDepth = 8
)

// The bounds of the storage module (spec-storage.md 12.2). All three are
// placeholders (N-015, N-027).
const (
	// MaxRoom is the largest mass room and bulk room, of a base room, of a store
	// class and of one dimension of a Capacity: 10^12, so that MaxWeight * MaxRoom
	// is 10^18 and the largest room of an account (the base room of a cohort and
	// MaxStoresPerAgent stores) is below int64.
	MaxRoom = 1_000_000_000_000
	// MaxStoresPerAgent is the most stores an agent owns, in a role row's Stores
	// and by calls.
	MaxStoresPerAgent = 8
	// MaxStoreClasses is the most store classes.
	MaxStoreClasses = 64
)

// The bounds of the settlements module (spec-settlements.md 8.1). All five are
// placeholders (N-015, N-027).
const (
	// MaxSettlementClasses is the most settlement classes: a ranking of settlements
	// is short, and each region holds one threshold per class.
	MaxSettlementClasses = 16
	// MaxClassFlags is the most distinct flag names over the classes of the module,
	// so that the flags of a class fit one 64-bit word.
	MaxClassFlags = 64
	// MaxClassLimits is the most distinct limit names over the classes of the module,
	// so that which limits a class carries fits one 64-bit word.
	MaxClassLimits = 64
	// MaxBandBP is the largest band, in basis points of an enter threshold: a leave
	// threshold is at least half the enter threshold, so a region that has just
	// reached a class is not a few households from losing it.
	MaxBandBP = 5_000
	// MaxThresholdBP is the largest scale of the enter thresholds of a region, in
	// basis points: ten times, as MaxYieldBP.
	MaxThresholdBP = 100_000
)

// The bounds of the technology module (spec-technology.md 9.2). All are
// placeholders (N-015, N-027), chosen so that every product of spec-technology.md
// 9.5 fits int64 or the checked 128-bit helper.
const (
	// MaxTechFields is the most technology fields: fields are authored by hand.
	MaxTechFields = 64
	// MaxFieldRecipes is the most recipes of one field.
	MaxFieldRecipes = 64
	// MaxTechBase is the largest Base of a field, in executions: 10^12, so that
	// X + Base fits int64 with MaxExperience.
	MaxTechBase = 1_000_000_000_000
	// MaxDoublingBP is the largest gain per doubling of the cumulative work: a
	// doubling doubles the table's yield at most.
	MaxDoublingBP = 10_000
	// MaxTechTrendBP is the largest trend a year: a year adds at most the table's
	// yield again.
	MaxTechTrendBP = 10_000
	// MaxDiscoveryBP is the largest discovery rate: ten discoveries a year.
	MaxDiscoveryBP = 100_000
	// MaxGainBP is the largest cap of a field: a technique raises the table's
	// yield at most elevenfold.
	MaxGainBP = 100_000
	// MaxStepBP is the largest gain of one discovery: the largest cap.
	MaxStepBP = MaxGainBP
	// MaxCatchUpBP is the largest share of the gap closed a year: the whole gap.
	MaxCatchUpBP = 10_000
	// MaxTechBP is the largest technology factor, in basis points: 10 000 plus the
	// largest cap.
	MaxTechBP = 10_000 + MaxGainBP
	// MaxExperience is the most executions a cell counts, 2^50: unreachable in any
	// run, it only keeps X + Base inside int64 (spec-technology.md 4.2.4).
	MaxExperience = 1 << 50
)

// The bounds of the credit module (spec-credit.md 14). All three are placeholders
// (N-015, N-027).
const (
	// MaxCreditRateBP is the largest yearly rate of a lender, of a borrower's limit
	// and of an effective rate: ten times the principal a year. A rate above it is a
	// typing error, and with MaxCreditTerm and MaxMoney the interest product fits 128
	// bits.
	MaxCreditRateBP = 100_000
	// MaxCreditTerm is the longest term, shield, write-off, keep and slack, in ticks:
	// ten years of daily ticks. spec-insurance.md reuses it for PayoutTicks.
	MaxCreditTerm = 3_650
	// MaxOpenClaims is the most open claims at once, the bound of Credit.MaxClaims.
	MaxOpenClaims = 1_000_000
)

// The bounds of the insurance module (spec-insurance.md 13). All are placeholders
// (N-015, N-027). Insurance.PayoutTicks reuses MaxCreditTerm.
const (
	// MaxValueCapBP is the largest cap on a unit's value, ten times the reference
	// price: a larger cap is a typing error, and the product with MaxPrice fits int64.
	MaxValueCapBP = 100_000
	// MaxLoadingBound is the largest loading, ten times the fair premium; with
	// MaxMoney, 10 000 and 10 000 the premium product fits 128 bits.
	MaxLoadingBound = 100_000
	// MaxStoreRateBound is the largest store rate for one tick: the whole sum insured.
	MaxStoreRateBound = 10_000
	// MaxCapBound is the largest cap on an underwriter's open liability, ten times
	// its money: a leverage above it is a typing error.
	MaxCapBound = 100_000
	// MaxRuinBound is the largest ruin share, ten times the household's money.
	MaxRuinBound = 100_000
	// MaxOpenPolicies is the most policies open at once, the bound of
	// Insurance.MaxPolicies.
	MaxOpenPolicies = 1_000_000
	// MaxCoverRequestsBound is the most requests in one plan: the loop of a plan.
	MaxCoverRequestsBound = 64
)

// The bounds of the tenure module (spec-land.md 10). All are placeholders (N-015,
// N-027).
const (
	// MaxRentStepBP is the largest step of the posted rent: a tick moves it by at
	// most the configured rate.
	MaxRentStepBP = 10_000
	// MaxRentCeilingBP is the largest ceiling of the posted rent: ten times the
	// configured rate.
	MaxRentCeilingBP = 100_000
	// MaxHoldings is the most term rows, and the most holding rows, of the module.
	MaxHoldings = 4_096
	// MaxTitleTransfers is the most entries of one call of TransferTitles.
	MaxTitleTransfers = 1_024
)

// The bound of the risk module (spec-risk.md 12.2). It is a placeholder (N-015, N-027).
const (
	// MaxRiskRows is the most rows of each table of the module, the route rows and
	// the store rows.
	MaxRiskRows = 4_096
)

// The bounds of the infrastructure module (spec-infrastructure.md 12.2). All are
// placeholders (N-015, N-027), chosen so that every product of the spec fits in 128
// bits (spec-infrastructure.md 13.3).
const (
	// MaxFacilityClasses is the most facility classes: classes are authored by hand.
	MaxFacilityClasses = 64
	// MaxDurability is the largest Durability of a class. A unit is the author's, and a
	// product with a capacity or a boost fits.
	MaxDurability = 1_000_000
	// MaxFacilityCapacity is the largest Capacity of a site class, the executions a
	// whole facility serves for one principal in a tick.
	MaxFacilityCapacity = 100_000
	// MaxBoostBP is the largest BoostBP of a route or ground class: a facility can at
	// most raise a factor elevenfold.
	MaxBoostBP = 100_000
	// MaxTotalBoostBP is the cap of the sum of the gains that apply to one pair or one
	// endowment.
	MaxTotalBoostBP = 200_000
	// MaxRouteModes is the most transport modes one route class speeds.
	MaxRouteModes = 8
	// MaxRecipeNeeds is the most site classes one recipe needs.
	MaxRecipeNeeds = 2
	// MaxFacilities is the most facilities in the world at once: the list is scanned
	// once a tick.
	MaxFacilities = 100_000
	// MaxFacilitiesPerAgent is the most facilities one owner other than an estate
	// holds: a household does not own a city, and its view carries them.
	MaxFacilitiesPerAgent = 64
	// MaxInitialCount is the largest Count of an initial row: construction is bounded.
	MaxInitialCount = 1_000
	// MaxInitialRows is the most initial rows of the module.
	MaxInitialRows = 1_024
)

// The bounds of the institutions module (spec-institutions.md 9). All are
// placeholders (N-015, N-027).
const (
	// MaxBodies is the most bodies: bodies are authored by hand.
	MaxBodies = 256
	// MaxRights is the most rights.
	MaxRights = 4_096
	// MaxControls is the most controls and rate limits together, one namespace.
	MaxControls = 4_096
	// MaxBodyTrainees is the largest MaxTrainees of a body, the bound of a role's own
	// (the skills module's, spec-institutions.md 9).
	MaxBodyTrainees = 100
)

// The bounds of the fiscal module (spec-fiscal.md 11.4, 14). All are placeholders
// (N-015, N-027). The module also reuses MaxFlowAmount, MaxFlowRateBP, MaxFlowScale,
// MaxMoney, MaxWeight, MaxQuantity and MaxPrice.
const (
	// MaxFiscalFlows is the most fiscal flows of all six kinds together: the step loops
	// over the active flows and their classes each tick, so its cost is bounded by the
	// configuration, as MaxMoneyFlows is.
	MaxFiscalFlows = 64
	// MaxDutyBP is the largest RateBP of a duty: the whole value of a trade at a scale
	// of 10 000. A scale above 10 000 can lift the applied rate above it.
	MaxDutyBP = 10_000
	// MaxLimitBP is the largest LimitBP of a standing order: ten times the effective
	// reference price.
	MaxLimitBP = 100_000
)

// The bounds of the hire form of the employment module (spec-employment.md 13).
// Both are provisional placeholders (N-015, N-027).
const (
	// MaxEstablishment is the largest Employer.MaxHires: the hired executions of one
	// employer in progress at once. With MaxQuantity and MaxPrice the product of a
	// bill (wage, Duration, executions) fits int64 through the 128-bit helper, and
	// an establishment above a thousand households is not one agent.
	MaxEstablishment = 1_000
	// MaxHireRecords is the largest Employment.MaxOpen: the hires open at once in a
	// region, and so the Escrow accounts of the hires are at most this many (as
	// MaxContracts).
	MaxHireRecords = 1_000_000
)

// The bounds of the policy module (spec-policy.md 10). All are placeholders (N-015,
// N-027). The module also reuses MaxMoney, MaxFlowScale, MaxEdgeFactorBP,
// MaxTransitFactorBP and MaxLabelBytes, which are the owners'.
const (
	// MaxActors is the most policy actors: at most one per jurisdiction, and a pack has
	// few authorities.
	MaxActors = 64
	// MaxLevers is the most levers of one actor.
	MaxLevers = 32
	// MaxScheduleSteps is the most steps of one actor's schedule.
	MaxScheduleSteps = 256
	// MaxPolicyEvery is the largest Every: ten years of a 365-tick year.
	MaxPolicyEvery = 3_650
	// MaxPolicyCycle is the largest Cycle of a schedule, in ticks.
	MaxPolicyCycle = 3_650
	// MaxScheduleTick is the largest At of a schedule step.
	MaxScheduleTick = 1_000_000_000
	// MaxLeverKeys is the most keys a single Against lever expands to.
	MaxLeverKeys = 256
	// MaxMoveBP is the largest Move of a reaction: basis points of a lever value per
	// Span, so the product of an error and a move fits in 128 bits.
	MaxMoveBP = 100_000
)
