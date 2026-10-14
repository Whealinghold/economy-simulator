package economysim

import "strconv"

// This file declares the events the first slice emits (spec-observability.md
// 4.1, observability-and-logging.md 4). It declares the fields the specs give
// the core, that is every field no module of Config.Modules owns (decision D3
// of the plan of Stage 0); a module's change appends its own events and its
// own fields at the end of an existing event (architecture.md 2.10.5). An
// appended field is a count, an amount, a bool, a constant of a closed set or a
// slice, never a typed ID (FR-OBS-091).
//
// Every event is a plain struct passed by value with Tick as its first field.
// None holds a pointer, a function, a map, a float or a wall-clock time. A
// slice in an event is a copy that the event owns, in a deterministic order.
// A field that names a configured entity carries the typed integer ID of its
// kind; the strings of the IDs are in the Names table. The events are declared
// here and emitted from Stage 1 on.

// Event is implemented only by the event types of this package. An observer
// type-switches on the concrete type and ignores the types it does not know, so
// a new event is not a breaking change. Events are passed by value.
type Event interface {
	isEvent()
}

// Side is the side of an order: a bid buys and an ask sells.
type Side int

const (
	// Bid is the buying side of an order.
	Bid Side = iota
	// Ask is the selling side of an order.
	Ask
)

// String returns "bid" or "ask", the value of the side log key and of the
// golden rendering of events. Any other value gives "side(n)".
func (s Side) String() string {
	switch s {
	case Bid:
		return "bid"
	case Ask:
		return "ask"
	}
	return "side(" + strconv.Itoa(int(s)) + ")"
}

// Reason is a machine-readable reason, a member of a closed set: the rejection
// of an order, the end of a job without output or a ledger reason of a
// boundary flow. Its values are the values of the reason log key. The
// constants below are the ones the first slice can reach; each module that
// adds a reason appends its constants (`not_permitted`, `no_room`,
// `principal_left`, `region_changed` and the reasons of its flows).
type Reason string

// The rejection reasons of an order, in the order the market checks them
// (spec-clearing.md 3.2).
const (
	// ReasonUnknownAgent is an order of an agent that does not exist or is not
	// in the market's region.
	ReasonUnknownAgent Reason = "unknown_agent"
	// ReasonSideNotAllowed is an order of a pseudo-agent on a side it may not
	// place.
	ReasonSideNotAllowed Reason = "side_not_allowed"
	// ReasonUnknownCommodity is an order for a commodity that is not in the
	// configuration.
	ReasonUnknownCommodity Reason = "unknown_commodity"
	// ReasonNotTradable is an order for a commodity outside the tradable set of
	// the agent's role.
	ReasonNotTradable Reason = "not_tradable"
	// ReasonInvalidPrice is a limit price below 1 or above MaxPrice.
	ReasonInvalidPrice Reason = "invalid_price"
	// ReasonInvalidQuantity is a quantity below 1 or above MaxQuantity.
	ReasonInvalidQuantity Reason = "invalid_quantity"
	// ReasonDuplicateOrder is a second admitted order of one side for one
	// commodity of an agent in a tick.
	ReasonDuplicateOrder Reason = "duplicate_order"
	// ReasonOverflow is an order whose limit price times quantity does not fit
	// in int64.
	ReasonOverflow Reason = "overflow"
	// ReasonInsufficientFunds is a bid whose value exceeds the agent's
	// available cash.
	ReasonInsufficientFunds Reason = "insufficient_funds"
	// ReasonInsufficientStock is an ask whose quantity exceeds the agent's
	// available stock.
	ReasonInsufficientStock Reason = "insufficient_stock"
)

// The reasons a job ends without output (spec-production.md 14.1).
const (
	// ReasonAgentRemoved is a job abandoned because its agent was removed.
	ReasonAgentRemoved Reason = "agent_removed"
	// ReasonRoleChanged is a job abandoned because its agent changed role.
	ReasonRoleChanged Reason = "role_changed"
)

// The ledger reasons of the flows that cross the boundary of the economy and
// so appear in BoundaryFlow (spec-ledger.md 7.2, spec-observability.md 4.4.1).
const (
	// ReasonInitial is the endowment of money and goods at construction.
	ReasonInitial Reason = "initial"
	// ReasonRecipeInput is goods consumed by a recipe as inputs.
	ReasonRecipeInput Reason = "recipe_input"
	// ReasonRecipeOutput is goods created by a recipe as outputs.
	ReasonRecipeOutput Reason = "recipe_output"
	// ReasonDisposal is goods discarded by a recipe or by an agent.
	ReasonDisposal Reason = "disposal"
	// ReasonConsumption is goods consumed as upkeep.
	ReasonConsumption Reason = "consumption"
	// ReasonSpoilage is goods lost to shelf life.
	ReasonSpoilage Reason = "spoilage"
	// ReasonWear is goods lost to the wear of a tool.
	ReasonWear Reason = "wear"
)

// The internal ledger reasons of the built-in group (spec-ledger.md 7.2.2):
// flows between two non-boundary accounts, so none is a faucet or a sink and
// none appears in BoundaryFlow. They are the names the ledger's reasons map to
// (ledger.go).
const (
	// ReasonTrade is money and goods moved by a matched trade.
	ReasonTrade Reason = "trade"
	// ReasonEstateMoney is the money of a removed agent moved to its estate.
	ReasonEstateMoney Reason = "estate_money"
	// ReasonEstateGoods is the goods of a removed agent moved to its estate.
	ReasonEstateGoods Reason = "estate_goods"
	// ReasonStartMoney is the starting money an estate pool gives a new agent.
	ReasonStartMoney Reason = "start_money"
	// ReasonSplit is money and goods moved when an agent is split.
	ReasonSplit Reason = "split"
	// ReasonMerge is money and goods moved when two agents are merged.
	ReasonMerge Reason = "merge"
	// ReasonBirths is the money given to households added to an existing agent.
	ReasonBirths Reason = "births"
	// ReasonStarvation is the money and goods of households cut by the failure
	// ladder, moved to the estate.
	ReasonStarvation Reason = "starvation"
)

// Cause is the machine-readable cause of a removal, a weight change or a role
// change, a member of a closed set. Its values are the values of the cause log
// key. The constants below are the ones the first slice can reach; a module
// appends its own (`population`, `births`, `traineeship`).
type Cause string

// The causes of the removal, weight change and role change events
// (spec-spawner.md 3).
const (
	// CauseInsolvent is a removal or reduction of an agent whose money was
	// below its upkeep.
	CauseInsolvent Cause = "insolvent"
	// CauseStarved is a removal or reduction of an agent that could pay its
	// upkeep but went unfed.
	CauseStarved Cause = "starved"
	// CauseConsumer is a change that the consumer's own call or spawner asked
	// for.
	CauseConsumer Cause = "consumer"
	// CauseDistress is a role change of a household in distress.
	CauseDistress Cause = "distress"
	// CauseReview is a role change made at the periodic review of roles.
	CauseReview Cause = "review"
	// CauseRenewal is households entering to replace households lost.
	CauseRenewal Cause = "renewal"
)

// Kind is the kind of a boundary flow, a member of a closed set: a faucet
// brings money or goods into the economy and a sink takes them out. Other
// events add their own sets of kinds by appending constants.
type Kind string

const (
	// KindFaucet is a flow from outside the economy into it.
	KindFaucet Kind = "faucet"
	// KindSink is a flow from the economy to outside it.
	KindSink Kind = "sink"
)

// Names maps each typed ID of a world to the string ID of its configuration.
// The slice of a kind is indexed by that kind's ID type. It is plain data: a
// copy owned by the event or snapshot that carries it. The slice declares the
// kinds the core has; the kinds of the modules (resources, transport modes,
// wants, store classes and the rest) are appended with their modules.
type Names struct {
	// Regions holds the string ID of each RegionID.
	Regions []string
	// Commodities holds the string ID of each CommodityID.
	Commodities []string
	// Roles holds the string ID of each RoleID.
	Roles []string
	// Recipes holds the string ID of each RecipeID.
	Recipes []string
	// Seasons holds the string ID of each season of the calendar.
	Seasons []string
}

// Amount is a quantity of one commodity.
type Amount struct {
	// Commodity is the commodity.
	Commodity CommodityID
	// Quantity is the number of units.
	Quantity Quantity
}

// HaltRow is the households for which the production halt skipped one recipe.
type HaltRow struct {
	// Recipe is the recipe that was skipped.
	Recipe RecipeID
	// Households is the households for which it was skipped.
	Households int64
}

// WorldCreated is the first event of a constructed world, with Tick 0. It
// precedes the AgentSpawned event of each initial agent. The jurisdictions of
// the polity module are appended with that module.
type WorldCreated struct {
	Tick int64
	// Seed is the world's seed.
	Seed uint64
	// ConfigHash is the whole configuration hash.
	ConfigHash [32]byte
	// WarmStart is whether a warm start was present.
	WarmStart bool
	// Names maps every typed ID to its string ID.
	Names Names
}

// RunStarted is emitted when Run begins, before its first tick.
type RunStarted struct {
	Tick int64
	// Ticks is the number of ticks requested.
	Ticks int64
}

// RunEnded is emitted when Run returns, after its last completed tick. Cancelled
// and Failed are never both true.
type RunEnded struct {
	Tick int64
	// Completed is the number of steps that finished without error.
	Completed int64
	// Cancelled is true only when the context was cancelled.
	Cancelled bool
	// Failed is true only when a step failed.
	Failed bool
}

// TickStarted is emitted when a step begins.
type TickStarted struct {
	Tick int64
}

// TickEnded is emitted when a step that began ends.
type TickEnded struct {
	Tick int64
	// Failed is true when the step returned an error.
	Failed bool
}

// OrderPlaced is emitted for each admitted order.
type OrderPlaced struct {
	Tick int64
	// Region is the market's region.
	Region RegionID
	// Commodity is the commodity ordered.
	Commodity CommodityID
	// Agent is the agent that placed the order.
	Agent AgentID
	// Side is bid or ask.
	Side Side
	// Limit is the admitted limit price, after any legal clamp.
	Limit Price
	// Quantity is the number of units.
	Quantity Quantity
}

// OrderRejected is emitted for each order that was not admitted.
type OrderRejected struct {
	Tick int64
	// Region is the market's region.
	Region RegionID
	// Commodity is the commodity ordered.
	Commodity CommodityID
	// Agent is the agent that placed the order.
	Agent AgentID
	// Side is bid or ask.
	Side Side
	// Reason is why the order was rejected.
	Reason Reason
}

// TradeExecuted is emitted when a bid and an ask are matched.
type TradeExecuted struct {
	Tick int64
	// Region is the market's region.
	Region RegionID
	// Commodity is the commodity traded.
	Commodity CommodityID
	// Buyer is the agent that bought.
	Buyer AgentID
	// Seller is the agent that sold.
	Seller AgentID
	// Price is the price of one unit.
	Price Price
	// Quantity is the number of units.
	Quantity Quantity
}

// MarketCleared is emitted once per market per tick, after its trades, also
// when nothing traded.
type MarketCleared struct {
	Tick int64
	// Region is the market's region.
	Region RegionID
	// Commodity is the commodity of the market.
	Commodity CommodityID
	// Price is the published price.
	Price Price
	// PriceAge is the ticks since the price was last set by a trade.
	PriceAge int64
	// Volume is the units traded.
	Volume Quantity
	// UnmatchedBid is the bid units left unmatched.
	UnmatchedBid Quantity
	// UnmatchedAsk is the ask units left unmatched.
	UnmatchedAsk Quantity
	// BestBid is the highest unmatched bid; valid when HasBestBid.
	BestBid Price
	// BestAsk is the lowest unmatched ask; valid when HasBestAsk.
	BestAsk Price
	// HasBestBid is whether an unmatched bid remained.
	HasBestBid bool
	// HasBestAsk is whether an unmatched ask remained.
	HasBestAsk bool
}

// BeliefsUpdated is emitted once per market per tick that had at least one
// order outcome, after the outcomes are processed. Every count is of outcomes
// or of agents.
type BeliefsUpdated struct {
	Tick int64
	// Region is the market's region.
	Region RegionID
	// Commodity is the commodity of the market.
	Commodity CommodityID
	// Outcomes is the number of order outcomes processed.
	Outcomes int64
	// Adjusted is the number of beliefs that moved.
	Adjusted int64
	// Raised is the number of beliefs raised.
	Raised int64
	// Lowered is the number of beliefs lowered.
	Lowered int64
	// AtFloor is the number of beliefs on the floor of the price band.
	AtFloor int64
	// AtCeiling is the number of beliefs on the ceiling of the price band.
	AtCeiling int64
	// Conflicts is the number of adjustments that conflicted.
	Conflicts int64
	// Clamped is the number of adjustments cut by a bound.
	Clamped int64
	// MeanCentre is the mean centre of the beliefs.
	MeanCentre Price
	// LowCentre is the lowest centre.
	LowCentre Price
	// HighCentre is the highest centre.
	HighCentre Price
}

// RecipeExecuted is emitted when a job completes; a one-tick job completes in
// its start tick.
type RecipeExecuted struct {
	Tick int64
	// Agent is the agent that ran the recipe.
	Agent AgentID
	// Region is the agent's region at the event.
	Region RegionID
	// Recipe is the recipe.
	Recipe RecipeID
	// StartTick is the tick the job started.
	StartTick int64
	// Executions is the number of executions of the recipe, the sum of
	// weights of the households that ran it.
	Executions int64
	// Outputs is the units created, in output order.
	Outputs []Amount
	// Disposed is the units of the outputs that were discarded.
	Disposed []Amount
	// EffectiveYield is the yield applied, in basis points.
	EffectiveYield int64
}

// RecipeAbandoned is emitted when a job ends without output.
type RecipeAbandoned struct {
	Tick int64
	// Agent is the agent that held the job.
	Agent AgentID
	// Region is the agent's region at the event.
	Region RegionID
	// Recipe is the recipe.
	Recipe RecipeID
	// Executions is the number of executions that were abandoned.
	Executions int64
	// Reason is why the job ended.
	Reason Reason
}

// ProductionHalted is emitted once per region and role per tick in which the
// production halt declined the job request of at least one free household; it
// is never emitted when none was.
type ProductionHalted struct {
	Tick int64
	// Region is the region.
	Region RegionID
	// Role is the role.
	Role RoleID
	// Free is the households of the role that were free to work.
	Free int64
	// Halted is the free households that requested no job because of the halt.
	Halted int64
	// Skipped has one row for each recipe the halt skipped, in ascending
	// RecipeID.
	Skipped []HaltRow
}

// UpkeepShortfall is emitted once per region and role per tick in which at
// least one household went unfed; it is never emitted when all were fed.
type UpkeepShortfall struct {
	Tick int64
	// Region is the region.
	Region RegionID
	// Role is the role.
	Role RoleID
	// Households is the households of the role in the region.
	Households int64
	// Unfed is the households that went unfed.
	Unfed int64
	// Agents is the agents with unfed households.
	Agents int64
	// MaxHunger is the largest hunger of an agent of the role.
	MaxHunger int64
}

// AgentSpawned is emitted when an agent enters the world, including each
// initial agent after WorldCreated. The stores withheld by the storage module
// are appended with that module.
type AgentSpawned struct {
	Tick int64
	// Agent is the new agent.
	Agent AgentID
	// Region is the agent's region.
	Region RegionID
	// Role is the agent's role.
	Role RoleID
	// Weight is the households the agent stands for.
	Weight int64
	// StartMoney is the money the agent received.
	StartMoney Money
	// External is whether the agent is driven by the consumer.
	External bool
}

// AgentBankrupted is emitted when an agent is removed entirely. It is the only
// event that removes an agent, whatever the cause; the others of a removal
// accompany it.
type AgentBankrupted struct {
	Tick int64
	// Agent is the removed agent.
	Agent AgentID
	// Region is the agent's region.
	Region RegionID
	// Role is the agent's role.
	Role RoleID
	// Weight is the agent's weight at removal.
	Weight int64
	// Cause is why the agent was removed.
	Cause Cause
	// Money is the agent's money passed to the estate.
	Money Money
	// GoodsListed is the number of goods passed to the estate.
	GoodsListed int64
}

// EstateOpened is emitted with a removal, after AgentBankrupted.
type EstateOpened struct {
	Tick int64
	// Agent is the removed agent whose estate was opened.
	Agent AgentID
	// Region is the region of the estate.
	Region RegionID
	// Balance is the money the estate holds.
	Balance Money
	// GoodsListed is the number of goods listed.
	GoodsListed int64
}

// WeightChanged is emitted when an agent's weight changes without removing it.
type WeightChanged struct {
	Tick int64
	// Agent is the agent.
	Agent AgentID
	// Region is the agent's region.
	Region RegionID
	// Role is the agent's role.
	Role RoleID
	// From is the weight before.
	From int64
	// To is the weight after.
	To int64
	// Cause is why the weight changed.
	Cause Cause
	// Money is the money moved, signed, positive to the agent.
	Money Money
	// Units is the goods moved to the estate.
	Units int64
}

// RoleChanged is emitted when an agent changes role.
type RoleChanged struct {
	Tick int64
	// Agent is the agent.
	Agent AgentID
	// Region is the agent's region.
	Region RegionID
	// From is the role before.
	From RoleID
	// To is the role after.
	To RoleID
	// Weight is the agent's weight.
	Weight int64
	// Cause is why the role changed.
	Cause Cause
	// GoodsListed is the number of goods passed to the estate.
	GoodsListed int64
}

// EntryLimited is emitted once per region per tick in which a limit or refusal
// of the spawner acted; it is never emitted when every count is zero. The
// refusal by a role ceiling of the settlement module is appended with that
// module.
type EntryLimited struct {
	Tick int64
	// Region is the region.
	Region RegionID
	// Changed is the role changes made.
	Changed int64
	// Entered is the households that entered.
	Entered int64
	// Removed is the households removed.
	Removed int64
	// Unfunded is the households whose entry the pool could not fund.
	Unfunded int64
	// NoViableRole is the households with no viable role.
	NoViableRole int64
	// RegionCap is the entries cut by the region's cap.
	RegionCap int64
	// RoleCap is the entries cut by a role's cap.
	RoleCap int64
	// Cooldown is the changes held by a cooldown.
	Cooldown int64
	// Hysteresis is the changes held by the hysteresis band.
	Hysteresis int64
	// ExitCap is the removals cut by the exit cap.
	ExitCap int64
	// Refused is the decisions the world refused.
	Refused int64
}

// BoundaryFlow is emitted once per tick for each (region, reason, money or
// commodity) with a non-zero total whose reason is a faucet or a sink. The
// construction bucket of a world that has not yet stepped is emitted once with
// Tick -1, before its first TickStarted.
type BoundaryFlow struct {
	Tick int64
	// Region is the region the flow is attributed to.
	Region RegionID
	// Reason is the ledger reason of the flow.
	Reason Reason
	// Kind is faucet or sink.
	Kind Kind
	// Commodity is the commodity, or NoCommodity for money.
	Commodity CommodityID
	// Total is the non-zero, non-negative total: money in minor units or units
	// of the commodity. The direction is Kind.
	Total int64
}

// WearApplied is emitted once per tick for each commodity whose wear units of
// the tick are above 0, in ascending commodity order, after the tick's
// BoundaryFlow rows (spec-ledger.md 12.1.1). The construction bucket's wear
// units are emitted in the same form with Tick -1, after the construction
// BoundaryFlow rows. It is not a flow of a reason: an item reaches the wear
// boundary only when its durability is used up, so BoundaryFlow counts the
// retirement and this counts the rate.
type WearApplied struct {
	Tick int64
	// Commodity is the commodity whose items wore.
	Commodity CommodityID
	// Units is the durability actually removed, at least 1.
	Units int64
}

func (WorldCreated) isEvent()     {}
func (RunStarted) isEvent()       {}
func (RunEnded) isEvent()         {}
func (TickStarted) isEvent()      {}
func (TickEnded) isEvent()        {}
func (OrderPlaced) isEvent()      {}
func (OrderRejected) isEvent()    {}
func (TradeExecuted) isEvent()    {}
func (MarketCleared) isEvent()    {}
func (BeliefsUpdated) isEvent()   {}
func (RecipeExecuted) isEvent()   {}
func (RecipeAbandoned) isEvent()  {}
func (ProductionHalted) isEvent() {}
func (UpkeepShortfall) isEvent()  {}
func (AgentSpawned) isEvent()     {}
func (AgentBankrupted) isEvent()  {}
func (EstateOpened) isEvent()     {}
func (WeightChanged) isEvent()    {}
func (RoleChanged) isEvent()      {}
func (EntryLimited) isEvent()     {}
func (BoundaryFlow) isEvent()     {}
func (WearApplied) isEvent()      {}
