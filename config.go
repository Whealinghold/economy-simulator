package economysim

import (
	"cmp"
	"slices"
)

// Config is the data that defines a world: the world-level parameters, the
// calendar, the commodities, recipes, roles and regions, the population, an
// optional warm start and the optional modules (spec-config.md 3.1).
//
// The configuration types are plain structs with no tags and no embedded types,
// so a world can be built from Go alone. The zero value of an entity type is not
// a valid default: start a World, Calendar, Commodity, Recipe, Role or Region
// from its Default constructor and overwrite what differs (spec-config.md 5.2).
// Entities refer to each other by ID string; the typed IDs are assigned when a
// world is built, in the order of the IDs sorted as strings (spec-config.md 3.3).
//
// The order in which entities are written has no effect on a world or on its
// hash (spec-config.md 3.4). The field order of every struct here is the order
// of the field ids of the configuration hash (spec-config.md 8.2.2), so a field
// is appended and never inserted. At Stage 0 the types hold the core fields
// only; each module's change appends its own (architecture.md 2.10.5).
type Config struct {
	// World holds the world-level parameters.
	World WorldParams
	// Calendar is the structure of time.
	Calendar Calendar
	// Commodities are the goods, by ID in canonical order.
	Commodities []Commodity
	// Recipes are the production rules, by ID in canonical order.
	Recipes []Recipe
	// Yields are the yield profiles, one per region and recipe, in canonical
	// order (by region, then recipe).
	Yields []YieldProfile
	// Roles are the occupations, by ID in canonical order.
	Roles []Role
	// Regions are the places, by ID in canonical order.
	Regions []Region
	// Edges are the directed routes between regions, by (From, To) in canonical
	// order; none means no region can reach another (spec-regions.md 4.1). Its
	// hash id is 8, between Regions (7) and Groups (10); id 9 is RegionPrices.
	Edges []Edge
	// RegionPrices are the reference-price overrides, by (Region, Commodity) in
	// canonical order; none means every region uses the commodity's price
	// (spec-regions.md 5.2). Its hash id is 9.
	RegionPrices []RegionPrice
	// Groups are the population: the households of a role in a region, in
	// canonical order (by region, then role).
	Groups []Group
	// WarmStart is the steady-state seed table. A nil WarmStart means no warm
	// start, and a non-nil empty one is a different configuration
	// (spec-config.md 8.2.1). It switches no mechanism on.
	WarmStart *WarmStart
	// Modules holds one optional entry per module; a nil entry is off.
	Modules Modules
}

// Modules has one pointer per optional module, in the order of the Modules row
// of spec-config.md 3.1; a nil pointer is a module that is off. Each module's
// change appends its pointer, and the position of the module in that row is its
// field id in the configuration hash whatever the build order (Demand is 1).
type Modules struct {
	// Demand is the demand module: wants beyond upkeep, the tastes of the roles
	// and the regional scales (spec-demand.md). Nil is off; non-nil is on, even
	// when empty. Its hash id is 1.
	Demand *Demand
	// MoneySupply is the money supply module: configured faucets and sinks
	// (spec-money-supply.md). Nil is off; non-nil is on, even with no flow. Its
	// hash id is 2.
	MoneySupply *MoneySupply
	// Estates is the pool policy of the estates module: relief at P1.6 and the
	// pool as the target of a money flow (spec-estates.md 2.6, spec-config.md
	// 6.2.4). Nil is off; non-nil is on, even though it has no field. Its hash id
	// is 25, whatever its place in this struct.
	Estates *Estates
}

// Estates is the configuration of the pool policy (spec-config.md 6.2.4). It has
// no field: a non-nil pointer switches relief and the pool as a flow target on,
// and the tunables are Region.PoolRelease and Region.ReliefBelow.
type Estates struct{}

// Demand is the configuration of the demand module (spec-demand.md 3.2, 9). The
// zero value is a module that is on with no want. Every list is in canonical
// order after construction: wants by ID, tastes by role then want, region tastes
// by region then want (spec-config.md 3.4).
type Demand struct {
	// Wants are the want entities, by ID in canonical order.
	Wants []Want
	// Tastes are the (role, want) rows that put a want in a role's ladder, by
	// role then want in canonical order.
	Tastes []Taste
	// RegionTastes are the (region, want) rows that scale a want in a region, by
	// region then want in canonical order.
	RegionTastes []RegionTaste
}

// Want is a need slot beyond upkeep: alternatives, a quantity per period, a rank
// and the shape of the schedule (spec-demand.md 3.2). It is not a commodity and
// not a recipe. Start from DefaultWant; ID, Items and Quantity have no default
// and are required.
type Want struct {
	// ID names the want.
	ID string
	// Items are the alternative commodities that satisfy the want, by commodity
	// in canonical order, 1 to MaxWantItems.
	Items []WantItem
	// Quantity is the units per household per Period at the reference price, 1 to
	// MaxQuantity / MaxWeight.
	Quantity int64
	// Period is the ticks over which Quantity is wanted, 1 to MaxWantPeriod.
	Period int64
	// Rank is the position on the ladder; a lower rank is funded first, 0 to
	// MaxWantRank.
	Rank int64
	// Elasticity is the slope of the schedule at the reference price in basis
	// points, 0 to MaxElasticityBP; 10 000 is unit-elastic.
	Elasticity int64
	// Ceiling is the most the schedule can reach in basis points, 10 000 to
	// MaxCeilingBP.
	Ceiling int64
	// Substitution is the integer exponent of the substitution shares, 0 to
	// MaxSubstitution.
	Substitution int64
	// Label is a display name. It is presentation only: it changes no result and
	// is not hashed.
	Label string
}

// WantItem is one alternative that can satisfy a want (spec-demand.md 3.2).
type WantItem struct {
	// Commodity is the ID of the commodity; it is not durable.
	Commodity string
	// Weight is the taste for this alternative in basis points, 1 to
	// MaxBasisPoints.
	Weight int64
}

// Taste is a role's taste for a want: the want is in the role's ladder
// (spec-demand.md 3.2). A Scale of 0 removes the want from the ladder.
type Taste struct {
	// Role is the ID of the role.
	Role string
	// Want is the ID of the want.
	Want string
	// Scale is the multiplier of the want's Quantity for the role in basis
	// points, 0 to MaxTasteScale.
	Scale int64
}

// RegionTaste scales a want in one region (spec-demand.md 3.2); a want with no
// row for a region is scaled by 10 000 there.
type RegionTaste struct {
	// Region is the ID of the region.
	Region string
	// Want is the ID of the want.
	Want string
	// Scale is the multiplier of the want's Quantity in the region in basis
	// points, 0 to MaxTasteScale.
	Scale int64
}

// WorldParams holds the world-level parameters. Start from DefaultWorldParams: the zero
// value is not a valid configuration. Every default is a provisional
// placeholder, and the bounds are checked by Validate. All integer fields are
// counts, ticks or basis points of 1/10 000 unless said otherwise.
type WorldParams struct {
	// Scale is the number of households per agent of a group that gives no
	// scale of its own (spec-scale.md 3.1).
	Scale int64
	// MinAgents is the smallest number of agents a group may be scaled down to
	// (spec-scale.md 5.2).
	MinAgents int64
	// BulkFactor is how heavily bulk counts in a chargeable size, in basis
	// points (spec-goods.md 8).
	BulkFactor int64
	// WelfareWindow is the base welfare window in ticks (spec-welfare-and-wages.md
	// 2.3.1); the effective window of a role is derived from it.
	WelfareWindow int64
	// WindowCycles is the number of longest jobs the effective welfare window
	// must hold (spec-welfare-and-wages.md 2.3.3).
	WindowCycles int64
	// Upkeep is the baseline upkeep of one household per tick: a list of
	// (commodity, quantity) (spec-agents.md 5.2).
	Upkeep []Item
	// CoverTicks is the ticks of use held as the target (spec-agents.md 7.3).
	CoverTicks int64
	// InputJobs is the jobs' worth of inputs held when the next job is about
	// to start (spec-agents.md 7.3).
	InputJobs int64
	// OutputCoverTicks is the ticks of own output regarded as the normal
	// inventory; 0 gives a seller no target (spec-agents.md 7.3).
	OutputCoverTicks int64
	// FavourFloor is the least share of a buffer want bought, in basis points
	// (spec-agents.md 7.5.2).
	FavourFloor int64
	// ReserveTicks is the ticks of upkeep cost kept as savings before any
	// buffer is bought (spec-agents.md 7.4.2).
	ReserveTicks int64
	// SpendMaxShare is the largest share spent on buffers, in basis points
	// (spec-agents.md 7.4.4).
	SpendMaxShare int64
	// SpendHalfTicks is the ticks of upkeep cost at which the spend share is
	// half its maximum (spec-agents.md 7.4.4).
	SpendHalfTicks int64
	// RestBand is how far above its target the ratio must be before an agent
	// rests, in basis points of the target (spec-agents.md 7.1.2).
	RestBand int64
	// ProduceHaltRatio is the stock, in basis points of the output target, at
	// which a recipe is not started (spec-agents.md 7.1.5).
	ProduceHaltRatio int64
	// BasketTolerance is how far a region's reference basket may differ from
	// the cost of its baseline upkeep, in basis points (spec-agents.md 5.6.1).
	BasketTolerance int64
	// BeliefStep is one unit of price adjustment, in basis points of the centre
	// (spec-beliefs.md 11).
	BeliefStep int64
	// BeliefAdjustProb is the probability that an agent adjusts after an
	// outcome, in basis points (spec-beliefs.md 11).
	BeliefAdjustProb int64
	// BeliefSignificance is how far a limit price must be beyond the market
	// price before a fully filled order is corrected, in basis points
	// (spec-beliefs.md 11).
	BeliefSignificance int64
	// BeliefStockLow is the stock ratio below which the stock component pushes
	// a price up, in basis points of the target (spec-beliefs.md 11).
	BeliefStockLow int64
	// BeliefStockHigh is the stock ratio above which the stock component
	// pushes a price down, in basis points of the target (spec-beliefs.md 11).
	BeliefStockHigh int64
	// BeliefMaxAge is the ticks for which a market price is fresh evidence
	// (spec-beliefs.md 11).
	BeliefMaxAge int64
	// BeliefWidthInit is the starting half-width of a belief, in basis points
	// of the centre (spec-beliefs.md 11).
	BeliefWidthInit int64
	// BeliefWidthMin is the smallest half-width of a belief (spec-beliefs.md 11).
	BeliefWidthMin int64
	// BeliefWidthMax is the largest half-width of a belief (spec-beliefs.md 11).
	BeliefWidthMax int64
	// BeliefWidthStep is the change of the half-width per adjustment, in basis
	// points (spec-beliefs.md 11).
	BeliefWidthStep int64
	// BeliefEvidenceShare is the largest evidence move as a share of one unit
	// of adjustment, in basis points (spec-beliefs.md 11).
	BeliefEvidenceShare int64
	// CostFloor is the lowest belief for a commodity with a known unit cost,
	// in basis points of that cost (spec-beliefs.md 11).
	CostFloor int64
	// CostCeiling is the highest belief for such a commodity, in basis points
	// of the cost (spec-beliefs.md 11).
	CostCeiling int64
	// RefBandLow is the lowest belief, in basis points of the effective
	// reference price (spec-beliefs.md 11).
	RefBandLow int64
	// RefBandHigh is the highest belief, in basis points of the effective
	// reference price (spec-beliefs.md 11).
	RefBandHigh int64
	// PriceWindow is the ticks of trading kept in a market's price window
	// (spec-beliefs.md 11).
	PriceWindow int64
	// Spawner is the registered name of the spawner; "default" is the built-in
	// one. It is hashed as a name (spec-spawner.md 3.1.3).
	Spawner string
	// ReviewTicks is the period of the occupation review of an agent, in ticks
	// (spec-spawner.md 11).
	ReviewTicks int64
	// EntryShare is the most households that may change role per tick, in
	// basis points; 0 turns role change off (spec-spawner.md 11).
	EntryShare int64
	// RoleHysteresis is the margin a role's expected ratio must exceed the
	// agent's own by, in basis points (spec-spawner.md 11).
	RoleHysteresis int64
	// RoleCooldown is the ticks after entering a role before an agent may
	// change role (spec-spawner.md 11).
	RoleCooldown int64
	// EntryRatio is the viability gate: the lowest role viability an entrant
	// may enter, in basis points (spec-spawner.md 11).
	EntryRatio int64
	// ChangePrivation is the privation at which rung 1 of the failure ladder
	// applies (spec-spawner.md 11).
	ChangePrivation int64
	// RemovePrivation is the privation at which rung 3 of the failure ladder
	// applies (spec-spawner.md 11).
	RemovePrivation int64
	// DeathStep is the largest share of an agent's weight that leaves per tick
	// by rung 3, in basis points (spec-spawner.md 11).
	DeathStep int64
	// ExitShare is the most households that may leave a region per tick by
	// rung 3, in basis points; 0 turns the rung off (spec-spawner.md 11).
	ExitShare int64
	// Renewal is the share of the households lost by rung 3 that are replaced,
	// in basis points (spec-spawner.md 11).
	Renewal int64
	// EntryPoolShare is the share of a regional pool that may fund entrants in
	// one tick, in basis points (spec-spawner.md 11).
	EntryPoolShare int64
	// Clearing is the registered name of the clearing rule; "average" is the
	// built-in AverageClearing. It is hashed as a name (spec-clearing.md 8).
	Clearing string
	// DistanceLabel names the pack's distance unit. It is presentation only: it
	// changes no result and is not hashed (spec-regions.md 4.1.2).
	DistanceLabel string
	// AnchorHold is the share of a region's starting wage (the warm start's) below
	// which the cost anchor does not read the pooled observed wage, in basis
	// points; 0 turns it off, which is the default (spec-production.md 9.3). It
	// bounds how far a measured wage pulls the anchor and sets no wage (FR-WEL-015).
	// It is the 46th field, appended after the presentation-only DistanceLabel.
	AnchorHold int64
}

// Calendar is the structure of time: one section, not an entity
// (spec-calendar.md 3.1). Its zero value is not a valid calendar (it has no
// month); start from DefaultCalendar, which is Earth. Months and Seasons are
// ordered lists: their order is data, they are kept as written and are never
// sorted.
type Calendar struct {
	// HoursPerDay is the hours in a calendar day.
	HoursPerDay int64
	// DaysPerWeek is the calendar days in a week.
	DaysPerWeek int64
	// Months are the months of a year in order.
	Months []Month
	// TicksPerDay is the ticks in a calendar day; a tick is a part of a day or
	// a number of whole days, never both.
	TicksPerDay int64
	// DaysPerTick is the calendar days in a tick.
	DaysPerTick int64
	// StartDay is the zero-based day of the year on which tick 0 falls.
	StartDay int64
	// Seasons are the seasons of a year in order; the list may be empty.
	Seasons []Season
	// SeasonStart is the zero-based day of the year on which the first season
	// begins.
	SeasonStart int64
}

// Month is one month of the calendar's ordered list.
type Month struct {
	// ID names the month.
	ID string
	// Days is the calendar days in the month.
	Days int64
}

// Season is one season of the calendar's ordered list.
type Season struct {
	// ID names the season.
	ID string
	// Days is the calendar days in the season.
	Days int64
}

// Commodity is a good that can be held and traded (spec-goods.md 8). Start
// from DefaultCommodity; ReferencePrice has no default and is required.
type Commodity struct {
	// ID names the commodity; it matches [a-z][a-z0-9_]* and is at most
	// MaxIDLength bytes.
	ID string
	// Mass is the load-bearing capacity one unit uses; 0 uses none.
	Mass int64
	// Bulk is the room one unit takes up; 0 takes none.
	Bulk int64
	// ShelfLife is the ticks a lot lasts before it spoils; 0 never spoils.
	ShelfLife int64
	// Durability is the wear units of a new unit; 0 does not wear.
	Durability int64
	// Classes are class tags for measurement, in canonical order.
	Classes []string
	// Raw says the commodity can be supplied without being made from others.
	Raw bool
	// ReferencePrice is the price per unit in minor units; required.
	ReferencePrice Price
	// Label is a display name. It is presentation only: it changes no result
	// and is not hashed.
	Label string
	// UnitLabel is the display name of the unit. It is presentation only: it
	// changes no result and is not hashed.
	UnitLabel string
}

// Item is a (commodity, quantity) pair: an upkeep, a start stock, a recipe
// input or a warm group's stock (spec-production.md 3.1).
type Item struct {
	// Commodity is the ID of the commodity.
	Commodity string
	// Quantity is the number of units.
	Quantity Quantity
}

// Output is one output of a recipe (spec-production.md 3.2). It repeats the
// fields of Item because no struct embeds another.
type Output struct {
	// Commodity is the ID of the commodity produced.
	Commodity string
	// Quantity is the units produced per execution.
	Quantity Quantity
	// CostShare is the share of the input cost allocated to the output, in
	// basis points; the non-discard shares of a recipe add to 10 000.
	CostShare int64
	// Discard says the output is created and disposed of in one step.
	Discard bool
}

// Tool is a durable commodity a recipe requires (spec-production.md 4.4).
type Tool struct {
	// Commodity is the ID of the tool commodity.
	Commodity string
	// WearPerUse is the wear units one execution puts on the tool.
	WearPerUse int64
}

// Recipe is a production rule (spec-production.md 3.1). Start from
// DefaultRecipe; Outputs is required.
type Recipe struct {
	// ID names the recipe.
	ID string
	// Duration is the ticks one job takes; at least 1.
	Duration int64
	// Inputs are consumed per execution, in canonical order.
	Inputs []Item
	// Outputs are produced per execution, in canonical order.
	Outputs []Output
	// Requires are the tools needed, in canonical order.
	Requires []Tool
	// Sited says the recipe runs only where the yield table gives a positive
	// factor.
	Sited bool
	// TraineeEligible says a trainee may run the recipe.
	TraineeEligible bool
	// TraineeYield is a trainee's share of a full execution's output, in basis
	// points.
	TraineeYield int64
	// Label is a display name. It is presentation only: it changes no result
	// and is not hashed.
	Label string
}

// YieldProfile is the yield of one recipe in one region (spec-production.md
// 6.1): a basis-point factor for all seasons, by season, or both.
type YieldProfile struct {
	// Region is the ID of the region.
	Region string
	// Recipe is the ID of the recipe.
	Recipe string
	// HasAll says All is set; the file says it by writing the key.
	HasAll bool
	// All is the factor for every season without an entry of its own, in basis
	// points.
	All int64
	// Seasons are the seasonal factors, in canonical order (by season ID).
	Seasons []SeasonYield
}

// SeasonYield is the yield factor of one season.
type SeasonYield struct {
	// Season is the ID of a season of the calendar.
	Season string
	// BP is the factor in basis points.
	BP int64
}

// Role is an occupation (spec-agents.md 3.3). Start from DefaultRole: the zero
// value has a TargetRatio of 0, which validation rejects.
type Role struct {
	// ID names the role. The word "pool" is reserved.
	ID string
	// Recipes are the recipes an agent of the role may start, in canonical
	// order.
	Recipes []string
	// Upkeep replaces the world's upkeep for the role when it is not empty.
	Upkeep []Item
	// Trades are extra tradable commodities, in canonical order.
	Trades []string
	// StartMoney is the money per household at creation.
	StartMoney Money
	// StartStock is the goods per household of an initial agent, in canonical
	// order.
	StartStock []Item
	// Strategy is the registered strategy name; "default" is built in.
	Strategy string
	// Unskilled says the role needs no tool or training to enter
	// (spec-welfare-and-wages.md 4.2).
	Unskilled bool
	// TargetRatio is the target income ratio in basis points, at least 10 000
	// (spec-welfare-and-wages.md 5).
	TargetRatio int64
	// TargetSpread is the spread of drawn targets, in basis points
	// (spec-welfare-and-wages.md 5.1).
	TargetSpread int64
	// DefaultRatio is the ratio assumed before any record exists, in basis
	// points (spec-welfare-and-wages.md 2.2).
	DefaultRatio int64
	// InKind is the share of income valued in kind, in basis points
	// (spec-welfare-and-wages.md 6).
	InKind int64
	// ExpectedWorkTicks is the expected working ticks a year
	// (spec-welfare-and-wages.md 4.2).
	ExpectedWorkTicks int64
	// MinWorkShare is the floor on the work share, in basis points
	// (spec-welfare-and-wages.md 5.3).
	MinWorkShare int64
	// TrainingTicks is the ticks of traineeship needed to enter the role; 0
	// means no training (spec-skill-barriers.md 4.1).
	TrainingTicks int64
	// MaxTrainees is the trainees one trainer may hold at once
	// (spec-skill-barriers.md 4.4).
	MaxTrainees int64
	// TeachingLoad is the share of a trainer's working time one trainee takes,
	// in basis points (spec-skill-barriers.md 4.2).
	TeachingLoad int64
	// Closed says the default spawner places nobody in the role
	// (spec-spawner.md 6.2).
	Closed bool
	// Label is a display name. It is presentation only: it changes no result
	// and is not hashed.
	Label string
}

// Region is a place with its own markets (spec-regions.md 3.2). Start from
// DefaultRegion.
type Region struct {
	// ID names the region.
	ID string
	// ReferenceBasket is the cost of one household's subsistence for one tick
	// at the region's reference prices, in minor units (spec-currency.md 5).
	ReferenceBasket Money
	// ExtraUpkeep is the goods added to every household's upkeep in the
	// region, in canonical order.
	ExtraUpkeep []Item
	// CapitalReturn is the required annual return on a tool, in basis points
	// (spec-skill-barriers.md 2.5).
	CapitalReturn int64
	// EstateMarkdown is the per-tick price reduction of an unsold estate
	// listing, in basis points (spec-estates.md 3).
	EstateMarkdown int64
	// PoolRelease is the share of the region's estate pool released as relief
	// each tick, in basis points, 0 to 10 000 (spec-estates.md 2.6, 3). It is
	// read, hashed and validated only while Modules.Estates is on.
	PoolRelease int64
	// ReliefBelow is the welfare ratio, in basis points, under which a
	// household draws relief, 0 to MaxBasisPoints (spec-estates.md 2.6, 3). It is
	// read, hashed and validated only while Modules.Estates is on.
	ReliefBelow int64
	// Label is a display name. It is presentation only: it changes no result
	// and is not hashed.
	Label string
}

// Edge is a directed route. It says that a carrier can go directly from From to
// To, over Distance. It says nothing about who, how fast or how much: those are
// the transport module's (spec-regions.md 4.1).
type Edge struct {
	// From is the ID of the region the route leaves.
	From string
	// To is the ID of the region the route enters; it differs from From.
	To string
	// Distance is the length of the route in the pack's distance unit, 1 to
	// MaxDistance. It has no default.
	Distance int64
}

// RegionPrice replaces, in one region, the reference price of one commodity
// (spec-regions.md 5.2).
type RegionPrice struct {
	// Region is the ID of the region whose reference price differs.
	Region string
	// Commodity is the ID of the commodity whose reference price differs.
	Commodity string
	// Price is the reference price in that region, in minor units, 1 to
	// MaxPrice. It has no default.
	Price Price
}

// Group is the population of one role in one region (spec-scale.md 3.1).
type Group struct {
	// Region is the ID of the region.
	Region string
	// Role is the ID of the role.
	Role string
	// Households is the number of households the group models.
	Households int64
	// Scale is the households per agent of the group; 0 means the world's
	// Scale.
	Scale int64
}

// WarmStart is the steady-state seed table (spec-viability.md 8.1): plain data,
// every list in canonical order, no maps. It switches no mechanism on.
type WarmStart struct {
	// Prices are the seed prices, by region then commodity.
	Prices []SeedPrice
	// Wages are the starting wages, by region.
	Wages []SeedWage
	// Groups are the seeds of a group's money, stock and jobs, by region then
	// role.
	Groups []WarmGroup
	// Pools are the regional estate pools at the start, by region.
	Pools []WarmPool
}

// SeedPrice is the seed price of a commodity in a region.
type SeedPrice struct {
	// Region is the ID of the region.
	Region string
	// Commodity is the ID of the commodity.
	Commodity string
	// Price is the market's initial published price, in minor units.
	Price Price
}

// SeedWage is the starting wage of a region: money per working
// household-tick, used as the pooled observed wage whenever that figure is
// unknown (spec-viability.md 8.2.5).
type SeedWage struct {
	// Region is the ID of the region.
	Region string
	// Wage is minor units per working household-tick.
	Wage Money
}

// WarmGroup is the seed of one group.
type WarmGroup struct {
	// Region is the ID of the region.
	Region string
	// Role is the ID of the role.
	Role string
	// Money is the money per household at the start, in minor units.
	Money Money
	// Stock is the goods per household, in canonical order.
	Stock []Item
	// Jobs are the jobs in progress at the start, in canonical order (by
	// recipe).
	Jobs []JobSpec
}

// JobSpec says what share of a group's agents start in progress on a recipe.
type JobSpec struct {
	// Recipe is the ID of the recipe.
	Recipe string
	// InProgressBP is the share of the group's agents, in basis points; the
	// file key is "in_progress".
	InProgressBP int64
}

// WarmPool is the estate pool of a region at the start.
type WarmPool struct {
	// Region is the ID of the region.
	Region string
	// Money is the pool in minor units; 0 when absent.
	Money Money
}

// canonical returns a deep copy of c in canonical order (spec-config.md 3.4):
// every entity slice sorted by ID (groups by region then role), and the lists
// of (commodity, quantity) by commodity. The calendar's months and seasons are
// ordered lists and are copied as written. Entities with an equal key keep
// their given order; such a Config fails validation (ErrDuplicateID). The
// receiver is not changed, so a World can own the result.
func (c Config) canonical() Config {
	out := c
	out.World.Upkeep = sortedBy(c.World.Upkeep, func(a, b Item) int { return cmp.Compare(a.Commodity, b.Commodity) })
	out.Calendar.Months = slices.Clone(c.Calendar.Months)
	out.Calendar.Seasons = slices.Clone(c.Calendar.Seasons)

	out.Commodities = sortedBy(c.Commodities, func(a, b Commodity) int { return cmp.Compare(a.ID, b.ID) })
	for i := range out.Commodities {
		out.Commodities[i].Classes = sortedBy(out.Commodities[i].Classes, cmp.Compare[string])
	}
	out.Recipes = sortedBy(c.Recipes, func(a, b Recipe) int { return cmp.Compare(a.ID, b.ID) })
	for i := range out.Recipes {
		r := &out.Recipes[i]
		r.Inputs = sortedBy(r.Inputs, func(a, b Item) int { return cmp.Compare(a.Commodity, b.Commodity) })
		r.Outputs = sortedBy(r.Outputs, func(a, b Output) int { return cmp.Compare(a.Commodity, b.Commodity) })
		r.Requires = sortedBy(r.Requires, func(a, b Tool) int { return cmp.Compare(a.Commodity, b.Commodity) })
	}
	out.Yields = sortedBy(c.Yields, func(a, b YieldProfile) int {
		return cmp.Or(cmp.Compare(a.Region, b.Region), cmp.Compare(a.Recipe, b.Recipe))
	})
	for i := range out.Yields {
		out.Yields[i].Seasons = sortedBy(out.Yields[i].Seasons, func(a, b SeasonYield) int { return cmp.Compare(a.Season, b.Season) })
	}
	out.Roles = sortedBy(c.Roles, func(a, b Role) int { return cmp.Compare(a.ID, b.ID) })
	for i := range out.Roles {
		r := &out.Roles[i]
		r.Recipes = sortedBy(r.Recipes, cmp.Compare[string])
		r.Upkeep = sortedBy(r.Upkeep, byCommodity)
		r.Trades = sortedBy(r.Trades, cmp.Compare[string])
		r.StartStock = sortedBy(r.StartStock, byCommodity)
	}
	out.Regions = sortedBy(c.Regions, func(a, b Region) int { return cmp.Compare(a.ID, b.ID) })
	for i := range out.Regions {
		out.Regions[i].ExtraUpkeep = sortedBy(out.Regions[i].ExtraUpkeep, byCommodity)
	}
	out.Edges = sortedBy(c.Edges, func(a, b Edge) int {
		return cmp.Or(cmp.Compare(a.From, b.From), cmp.Compare(a.To, b.To))
	})
	out.RegionPrices = sortedBy(c.RegionPrices, func(a, b RegionPrice) int {
		return cmp.Or(cmp.Compare(a.Region, b.Region), cmp.Compare(a.Commodity, b.Commodity))
	})
	out.Groups = sortedBy(c.Groups, func(a, b Group) int {
		return cmp.Or(cmp.Compare(a.Region, b.Region), cmp.Compare(a.Role, b.Role))
	})
	if c.WarmStart != nil {
		w := *c.WarmStart
		w.Prices = sortedBy(w.Prices, func(a, b SeedPrice) int {
			return cmp.Or(cmp.Compare(a.Region, b.Region), cmp.Compare(a.Commodity, b.Commodity))
		})
		w.Wages = sortedBy(w.Wages, func(a, b SeedWage) int { return cmp.Compare(a.Region, b.Region) })
		w.Groups = sortedBy(w.Groups, func(a, b WarmGroup) int {
			return cmp.Or(cmp.Compare(a.Region, b.Region), cmp.Compare(a.Role, b.Role))
		})
		for i := range w.Groups {
			w.Groups[i].Stock = sortedBy(w.Groups[i].Stock, byCommodity)
			w.Groups[i].Jobs = sortedBy(w.Groups[i].Jobs, func(a, b JobSpec) int { return cmp.Compare(a.Recipe, b.Recipe) })
		}
		w.Pools = sortedBy(w.Pools, func(a, b WarmPool) int { return cmp.Compare(a.Region, b.Region) })
		out.WarmStart = &w
	}
	out.Modules.Demand = c.Modules.Demand.canonical()
	out.Modules.MoneySupply = c.Modules.MoneySupply.canonical()
	return out
}

// byCommodity orders two items by the ID of their commodity.
func byCommodity(a, b Item) int { return cmp.Compare(a.Commodity, b.Commodity) }

// sortedBy returns a sorted copy of s, nil when s is empty. The sort is stable,
// so equal keys keep their given order (architecture.md 7.6).
func sortedBy[T any](s []T, compare func(a, b T) int) []T {
	if len(s) == 0 {
		return nil
	}
	out := slices.Clone(s)
	slices.SortStableFunc(out, compare)
	return out
}

// commodityID returns the typed ID of the commodity with the given ID string,
// which is its index in the canonical Commodities (spec-config.md 3.3). The
// receiver must already be canonical.
func (c Config) commodityID(id string) (CommodityID, bool) {
	i, ok := slices.BinarySearchFunc(c.Commodities, id, func(e Commodity, t string) int { return cmp.Compare(e.ID, t) })
	return CommodityID(i), ok
}

// roleID returns the typed ID of a role, its index in the canonical Roles.
func (c Config) roleID(id string) (RoleID, bool) {
	i, ok := slices.BinarySearchFunc(c.Roles, id, func(e Role, t string) int { return cmp.Compare(e.ID, t) })
	return RoleID(i), ok
}

// recipeID returns the typed ID of a recipe, its index in the canonical Recipes.
func (c Config) recipeID(id string) (RecipeID, bool) {
	i, ok := slices.BinarySearchFunc(c.Recipes, id, func(e Recipe, t string) int { return cmp.Compare(e.ID, t) })
	return RecipeID(i), ok
}

// regionID returns the typed ID of a region, its index in the canonical Regions.
func (c Config) regionID(id string) (RegionID, bool) {
	i, ok := slices.BinarySearchFunc(c.Regions, id, func(e Region, t string) int { return cmp.Compare(e.ID, t) })
	return RegionID(i), ok
}

// MoneySupply is the configuration of the money supply module
// (spec-money-supply.md 4.1): the money flows, each a grant (a faucet) or a levy
// (a sink) on a class of accounts. The zero value is a module that is on with no
// flow. After construction the flows are in ID order (spec-config.md 3.4).
type MoneySupply struct {
	// Flows are the money flows, by ID in canonical order, at most MaxMoneyFlows.
	Flows []MoneyFlow
}

// The values of MoneyFlow.Kind, MoneyFlow.Per and FlowTarget.Kind that are
// built.
const (
	// FlowGrant is a flow that issues money to its class: a faucet.
	FlowGrant = "grant"
	// FlowLevy is a flow that collects money from its class: a sink.
	FlowLevy = "levy"
	// PerTick is a period of one tick, the default of MoneyFlow.Per.
	PerTick = "tick"
	// PerYear is a period of the calendar's ticks in a year.
	PerYear = "year"
	// TargetHouseholds is the class of the households of a role.
	TargetHouseholds = "households"
	// TargetPool is the estate pool of a region, or of every region when the
	// target names none. It needs the estates module.
	TargetPool = "pool"
)

// MoneyFlow is one configured faucet (Kind FlowGrant) or sink (Kind FlowLevy)
// (spec-money-supply.md 4.1, 11.1). It has exactly one shape: a rate grant
// (Amount), a top-up (Level), a fixed levy (Amount) or a rate levy (RateBP).
// Every default is the zero value, and ID, Kind and Target have no default.
type MoneyFlow struct {
	// ID names the flow and is its ledger reason; it is not a built-in reason.
	ID string
	// Kind is FlowGrant or FlowLevy.
	Kind string
	// Target is the class of accounts that receives or pays.
	Target FlowTarget
	// Amount is the minor units per household per Per: a grant's rate or a fixed
	// levy, 0 to MaxFlowAmount.
	Amount int64
	// Level is a top-up grant's target balance per household, 0 to MaxFlowAmount.
	Level int64
	// RateBP is a rate levy's basis points per Per of the balance above Exempt,
	// 0 to MaxFlowRateBP.
	RateBP int64
	// Exempt is the minor units per household a levy never collects, 0 to
	// MaxFlowAmount.
	Exempt int64
	// Per is PerTick (also ""), the default, or PerYear.
	Per string
	// Start is the first tick the flow is active, 0 to math.MaxInt64 - 1.
	Start int64
	// Until is the first tick the flow is no longer active; 0 is never, else it
	// is above Start.
	Until int64
	// Limit is the most the flow ever applies, in minor units; 0 is none, else up
	// to MaxFlowTotal.
	Limit int64
	// NotIncome makes a grant to households not income for the welfare ratio.
	NotIncome bool
	// Label is a display name. It is presentation only: it changes no result and
	// is not hashed.
	Label string
}

// FlowTarget names a class of accounts (spec-money-supply.md 4.3). The kinds
// TargetHouseholds and TargetPool are built; the other kinds are recognised and
// refused with ErrModuleDependency until their modules are built.
type FlowTarget struct {
	// Kind is TargetHouseholds or TargetPool.
	Kind string
	// Region is the region of the class; empty is every region.
	Region string
	// Role is the role of the households; a pool has none.
	Role string
}
