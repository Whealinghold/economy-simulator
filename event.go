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
// boundary flow, or the outcome of a belief update in the "belief updated"
// Debug line. Its values are the values of the reason log key. The
// constants below are the ones the first slice can reach; each module that
// adds a reason appends its constants (`not_permitted`, `no_room`,
// `principal_left`, `region_changed` and the reasons of its flows).
type Reason string

// The rejection reasons of an order, in the order the market checks them
// (spec-clearing.md 3.2).
const (
	// ReasonUnknownAgent is an order of an agent that does not exist or is not
	// in the market's region, or a decision of a spawner that names such an
	// agent (spec-spawner.md 4).
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
	// in int64, or a relocation whose pending totals would pass int64.
	ReasonOverflow Reason = "overflow"
	// ReasonHoldingLimit is a bid whose quantity added to the buyer's holding
	// of the commodity exceeds MaxQuantity, so settling it would break the
	// ledger's stock bound (spec-clearing.md 3.2.2).
	ReasonHoldingLimit Reason = "holding_limit"
	// ReasonReceiptLimit is an ask whose worst-case receipt, added to the
	// seller's balance and to the receipts of its asks already admitted in the
	// round, exceeds MaxMoney, so settling it could break the ledger's balance
	// bound (spec-clearing.md 3.2.2).
	ReasonReceiptLimit Reason = "receipt_limit"
	// ReasonInsufficientFunds is a bid whose value exceeds the agent's
	// available cash.
	ReasonInsufficientFunds Reason = "insufficient_funds"
	// ReasonInsufficientStock is an ask whose quantity exceeds the agent's
	// available stock.
	ReasonInsufficientStock Reason = "insufficient_stock"
)

// The reasons a shipment request is refused, in the order of the checks
// (spec-transport.md 5.4). ReasonUnknownRegion, ReasonSameRegion,
// ReasonUnknownCommodity, ReasonInvalidQuantity, ReasonInsufficientStock and
// ReasonOverflow, above, are the others of the table. The reasons of the modules
// that are not built (has_store, outside_destination, consignee_full,
// toll_unaffordable) join with their modules.
const (
	// ReasonTooManyRequests is the request number MaxDispatchPerTick + 1 or later of
	// a plan.
	ReasonTooManyRequests Reason = "too_many_requests"
	// ReasonUnknownMode is a mode that is not in the configuration, and every
	// request while the transport module is off.
	ReasonUnknownMode Reason = "unknown_mode"
	// ReasonTravellerAway is a self-carried request of an agent that is already
	// away, also by an earlier request of the same plan.
	ReasonTravellerAway Reason = "traveller_away"
	// ReasonJobInProgress is a self-carried request of an agent with a job in
	// progress.
	ReasonJobInProgress Reason = "job_in_progress"
	// ReasonNoRoute is a pair of regions with no path.
	ReasonNoRoute Reason = "no_route"
	// ReasonBlocked is a pair whose transit factor is 0.
	ReasonBlocked Reason = "blocked"
	// ReasonTooFar is a journey of more than MaxTravelTicks.
	ReasonTooFar Reason = "too_far"
	// ReasonUnknownConsignee is a consignee that is not an existing agent.
	ReasonUnknownConsignee Reason = "unknown_consignee"
	// ReasonConsigneeElsewhere is a consignee that is not in the destination.
	ReasonConsigneeElsewhere Reason = "consignee_elsewhere"
	// ReasonLinesWithSelf is a request to oneself that carries lines.
	ReasonLinesWithSelf Reason = "lines_with_self"
	// ReasonEmptyShipment is a request that ships nothing to another agent.
	ReasonEmptyShipment Reason = "empty_shipment"
	// ReasonTooManyLines is a request of more than MaxShipmentLines lines.
	ReasonTooManyLines Reason = "too_many_lines"
	// ReasonDuplicateLine is a request that names a commodity twice.
	ReasonDuplicateLine Reason = "duplicate_line"
	// ReasonVehicleInCargo is a line that names the mode's own vehicle.
	ReasonVehicleInCargo Reason = "vehicle_in_cargo"
	// ReasonCrewShort is a self-carried request of an agent with fewer households
	// than the crew.
	ReasonCrewShort Reason = "crew_short"
	// ReasonNoVehicle is a self-carried request of an agent with no available
	// vehicle.
	ReasonNoVehicle Reason = "no_vehicle"
	// ReasonCapacityExceeded is a load that does not fit the pooled capacity.
	ReasonCapacityExceeded Reason = "capacity_exceeded"
	// ReasonInsufficientHaul is a hired request of an agent with fewer haul units
	// than the carriage needs.
	ReasonInsufficientHaul Reason = "insufficient_haul"
	// ReasonInsufficientFeed is a self-carried request of an agent that lacks the
	// feed of the trip.
	ReasonInsufficientFeed Reason = "insufficient_feed"
	// ReasonUnprovisioned is a self-carried request after which the traveller would
	// be unfed on the way.
	ReasonUnprovisioned Reason = "unprovisioned"
)

// The reasons a job ends without output (spec-production.md 14.1).
const (
	// ReasonAgentRemoved is a job abandoned because its agent was removed.
	ReasonAgentRemoved Reason = "agent_removed"
	// ReasonRoleChanged is a job abandoned because its agent changed role.
	ReasonRoleChanged Reason = "role_changed"
	// ReasonRegionChanged is a job abandoned because its agent left its region on
	// a migration journey (spec-production.md 14.1, spec-migration.md 8.2).
	ReasonRegionChanged Reason = "region_changed"
)

// The reasons of the migration module (spec-migration.md 7.6, 8.3, 13.2): a
// spawner refusal and the name of a region-bound obligation, and the outcomes of
// a pushed household. ReasonTooFar and ReasonUnprovisioned are the transport
// module's strings and ReasonHeld the belief's, which the outcomes of the same
// names consume (a repeated value is forbidden in one type). They are values of
// the reason log key and of the limit attribute of the metrics.
const (
	// ReasonOnJourney is a switch of an agent on a migration journey, and the name
	// of the region-bound obligation the module registers.
	ReasonOnJourney Reason = "on_journey"
	// ReasonStarted is a pushed household that started a journey.
	ReasonStarted Reason = "started"
	// ReasonNoEstimate is a pushed household whose role has no recipe to estimate.
	ReasonNoEstimate Reason = "no_estimate"
	// ReasonNoDestination is a pushed household with no candidate region.
	ReasonNoDestination Reason = "no_destination"
	// ReasonNoRumour is a pushed household with no candidate that has a complete
	// set of rumoured prices.
	ReasonNoRumour Reason = "no_rumour"
	// ReasonNoViable is a pushed household with no candidate in which its role
	// earns more than its upkeep costs.
	ReasonNoViable Reason = "no_viable"
	// ReasonNoGain is a pushed household whose best candidate does not clear the
	// margin over its own measured figure.
	ReasonNoGain Reason = "no_gain"
	// ReasonBound is a pushed household refused by a region-bound obligation.
	ReasonBound Reason = "bound"
	// ReasonOutCap is a pushed household held back by the outflow cap of its region.
	ReasonOutCap Reason = "out_cap"
	// ReasonInCap is a pushed household held back by the pipeline cap of every
	// destination it could reach.
	ReasonInCap Reason = "in_cap"
	// ReasonEnded is a journey that ended by a removal or a cancel.
	ReasonEnded Reason = "ended"
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

// The declared ledger reasons that cross the boundary: rows a module owns, in
// the ledger only while the module is on (spec-ledger.md 7.2.2). They are
// appended in build order, one per module change (architecture.md 2.10.5).
const (
	// ReasonWantConsumption is goods consumed as a want beyond upkeep, with the
	// demand module on (spec-demand.md 4.7).
	ReasonWantConsumption Reason = "want_consumption"
	// ReasonRelief is the money the pool of a region pays a household below the
	// relief line, with the estates module on (spec-estates.md 2.6).
	ReasonRelief Reason = "relief"
	// ReasonDeaths is the money and goods of households the population module
	// removed from an agent that keeps others, moved to the estate (spec-population.md
	// 4.7).
	ReasonDeaths Reason = "deaths"
	// ReasonTransitOut is the cargo of a shipment moved from its shipper into a
	// Transit account at dispatch, with the transport module on
	// (spec-transport.md 7.2).
	ReasonTransitOut Reason = "transit_out"
	// ReasonTransitIn is the cargo of a shipment moved from its Transit account to
	// the consignee at arrival, or to the estate of the destination when it is
	// diverted (spec-transport.md 7.2).
	ReasonTransitIn Reason = "transit_in"
	// ReasonHaulUse is the haul units a hired shipment redeems at dispatch, a goods
	// sink (spec-transport.md 3.3.3, 7.2).
	ReasonHaulUse Reason = "haul_use"
	// ReasonTransitUse is the feed a self-carried trip consumes at dispatch, a
	// goods sink (spec-transport.md 3.3.2, 7.2).
	ReasonTransitUse Reason = "transit_use"
	// ReasonEscrowIn is the money a payer puts into an Escrow account when a
	// delivery contract is formed, ReasonEscrowOut the money the escrow pays the
	// deliverer for the units that arrived, and ReasonEscrowRefund the money it
	// gives back to the payer, or to the estate of the destination when the payer is
	// gone (spec-merchants.md 9.6). All three are internal transfers of money.
	ReasonEscrowIn     Reason = "escrow_in"
	ReasonEscrowOut    Reason = "escrow_out"
	ReasonEscrowRefund Reason = "escrow_refund"
	// ReasonWagePay is the money an employer pays a worker as the wage of a tick:
	// an internal transfer of money, agent to agent in the payroll form of the
	// employment module (spec-employment.md 24.1.6), no faucet and no sink.
	ReasonWagePay Reason = "wage_pay"
)

// The declared ledger reasons of the external module, six rows declared while
// it is on (spec-ledger.md 7.2, spec-external-sector.md 8.1). The first three move money
// and the last three goods; ReasonOutsideBudget and ReasonOutsideOffer are
// faucets, the other four sinks.
const (
	// ReasonOutsideBudget is the money a buyer counterparty's account receives
	// from the boundary before its bid.
	ReasonOutsideBudget Reason = "outside_budget"
	// ReasonOutsideUnspent is the part of that budget the sweep gives back.
	ReasonOutsideUnspent Reason = "outside_unspent"
	// ReasonOutsideReceipts is the money a seller counterparty's account holds
	// after the markets, swept to the boundary.
	ReasonOutsideReceipts Reason = "outside_receipts"
	// ReasonOutsideOffer is the goods a seller counterparty's account receives from
	// the boundary before its ask, born in the tick at full durability.
	ReasonOutsideOffer Reason = "outside_offer"
	// ReasonOutsideUnsold is the goods of a seller's account that were not bought,
	// swept to the boundary.
	ReasonOutsideUnsold Reason = "outside_unsold"
	// ReasonOutsidePurchases is the goods a buyer's account bought, swept to the
	// boundary.
	ReasonOutsidePurchases Reason = "outside_purchases"
)

// The refusals of the contract round, the reasons of a delivery offer or take that
// the admission refuses and of a formation that fails (spec-merchants.md 9.3,
// 9.5). An admission also uses the reasons of an order and of a shipment request.
const (
	// ReasonContractsOff is an offer or a take while the merchants module is absent.
	ReasonContractsOff Reason = "contracts_off"
	// ReasonDuplicateTake is a second take for a commodity by the same agent in a
	// tick.
	ReasonDuplicateTake Reason = "duplicate_take"
	// ReasonTooManyContracts is a formation that finds MaxOpen contracts in flight.
	ReasonTooManyContracts Reason = "too_many_contracts"
	// ReasonPayerFunds is a formation whose payer no longer has the money its take
	// was sized on.
	ReasonPayerFunds Reason = "payer_funds"
	// ReasonNoTaker is an offer that finds no take to match.
	ReasonNoTaker Reason = "no_taker"
)

// The reasons of the "want skipped" Debug line of the demand module
// (spec-demand.md 11.2). They are log values only: no event carries them.
const (
	// ReasonChoked is a want whose price ratio is at or above its choke price.
	ReasonChoked Reason = "choked"
	// ReasonOverBudget is a want funded in part: the budget ran out inside it.
	ReasonOverBudget Reason = "over_budget"
	// ReasonNoBudget is a want with units the strategy funded none of, though
	// a bid was placed for one of its items.
	ReasonNoBudget Reason = "no_budget"
	// ReasonNoStock is a want with units for which the plan holds no bid of any
	// of its items: the agent already holds what it keeps.
	ReasonNoStock Reason = "no_stock"
)

// The reasons of the "belief updated" Debug line, in the precedence of
// spec-beliefs.md 12.2. They are log values only: no event carries them.
const (
	// ReasonGateClosed is a belief whose adjustment gate was closed: nothing
	// was evaluated.
	ReasonGateClosed Reason = "gate_closed"
	// ReasonMoved is a belief whose centre changed and ended inside its bounds.
	ReasonMoved Reason = "moved"
	// ReasonHeld is a belief whose gate was open and whose centre did not change.
	// It is also the outcome of a migration arrival held by a region-bound
	// obligation (spec-migration.md 8.4, 13.2); the belief owns the constant.
	ReasonHeld Reason = "held"
	// ReasonAtFloor is a belief whose centre ended on its floor, moved or not.
	ReasonAtFloor Reason = "at_floor"
	// ReasonAtCeiling is a belief whose centre ended on its ceiling, moved or not.
	ReasonAtCeiling Reason = "at_ceiling"
)

// The reasons a start of a recipe is refused in the first slice, in the
// master order of spec-production.md 4.2.1. The rest of the order arrives
// with the module that owns each reason, appended here in that order and
// never inserted.
const (
	// ReasonNotOnRole is a request for a recipe that is not on the agent's
	// role list.
	ReasonNotOnRole Reason = "not_on_role"
	// ReasonZeroYield is a start whose effective yield factor is 0 and whose
	// recipe is not refused as having no site.
	ReasonZeroYield Reason = "zero_yield"
	// ReasonNoSite is a start of a sited recipe that the region has no yield
	// profile for.
	ReasonNoSite Reason = "no_site"
	// ReasonNoTool is a start for which a tool item is missing.
	ReasonNoTool Reason = "no_tool"
	// ReasonNoInput is a start for which an input is not available.
	ReasonNoInput Reason = "no_input"
	// ReasonNoEndowment is a start for which the ration granted no execution of
	// a draw, or the live stock does not cover one (resources module).
	ReasonNoEndowment Reason = "no_endowment"
)

// The reasons of the "work declined" Debug line that no start refusal covers
// (spec-agents.md 14.3). They are log values only: no event carries them, and
// they are not values of the metric attribute economysim.reason.
const (
	// ReasonRest is a work share of 0: the agent rests.
	ReasonRest Reason = "rest"
	// ReasonHalted is no job because the production halt skipped every
	// eligible recipe (spec-agents.md 7.1.5).
	ReasonHalted Reason = "halted"
	// ReasonNoRecipe is no job because no recipe is eligible.
	ReasonNoRecipe Reason = "no_recipe"
)

// The reasons of a spawner decision the world refuses, in the order it checks
// them, then the two "entry dropped" Debug reasons (spec-spawner.md 4, 13.3).
// ReasonUnknownAgent, above, is the first check. on_journey is appended with the
// migration module, and in_traineeship, needs_weight_one, needs_traineeship and
// already_queued with the skill barriers.
const (
	// ReasonUnknownRole is a decision that names a role that does not exist.
	ReasonUnknownRole Reason = "unknown_role"
	// ReasonExternalAgent is a decision on an agent that is external and not
	// mortal.
	ReasonExternalAgent Reason = "external_agent"
	// ReasonSameRole is a switch to the agent's own role.
	ReasonSameRole Reason = "same_role"
	// ReasonBadHouseholds is a reduction by 0 or by more than the weight, or an
	// entry of fewer than 1 household.
	ReasonBadHouseholds Reason = "bad_households"
	// ReasonUnsupportedCause is a reduction whose cause is neither starved nor
	// insolvent. It is the last check.
	ReasonUnsupportedCause Reason = "unsupported_cause"
	// ReasonUnfunded is an "entry dropped" Debug reason, not a refusal: the
	// households of an entry the pool could not fund. Refused does not count it.
	ReasonUnfunded Reason = "unfunded"
	// ReasonNoViableRole is an "entry dropped" Debug reason, not a refusal:
	// households that replace households lost and have no viable role to enter.
	// Refused does not count it.
	ReasonNoViableRole Reason = "no_viable_role"
)

// The reasons of a spawner decision that the skill barriers refuse, appended
// after the others (spec-spawner.md 4, spec-skill-barriers.md 4.6.2), and the
// reasons of TraineeshipEnded (spec-skill-barriers.md 8.1).
const (
	// ReasonInTraineeship is a switch or an apply of a trainee, or a switch of a
	// trainer with trainees. It is also the name of the region_bound obligation
	// of the skill barriers.
	ReasonInTraineeship Reason = "in_traineeship"
	// ReasonNeedsWeightOne is an apply for an agent of weight above 1.
	ReasonNeedsWeightOne Reason = "needs_weight_one"
	// ReasonNeedsTraineeship is a switch or an entry into a role with
	// TrainingTicks above 0, which is entered only by an apply.
	ReasonNeedsTraineeship Reason = "needs_traineeship"
	// ReasonAlreadyQueued is an apply by an agent that is already in a queue.
	ReasonAlreadyQueued Reason = "already_queued"
	// ReasonTrainerLeft is a traineeship that ended because its trainer was
	// removed.
	ReasonTrainerLeft Reason = "trainer_left"
	// ReasonAgentLeft is a traineeship that ended because the trainee was
	// removed.
	ReasonAgentLeft Reason = "agent_left"
	// ReasonRoleRemoved is a traineeship that ended because its role was removed.
	// No role can be removed at run time, so it is declared and not emitted.
	ReasonRoleRemoved Reason = "role_removed"
)

// The reasons of a refused relocation, in the order of the checks (spec-regions.md
// 6.1.3). ReasonUnknownAgent, above, is the second. outside_node is appended with
// the external module.
const (
	// ReasonUnknownRegion is a destination that is not a region.
	ReasonUnknownRegion Reason = "unknown_region"
	// ReasonClosedAccount is an agent that was removed.
	ReasonClosedAccount Reason = "closed_account"
	// ReasonNotRelocatable is an account that is not an agent's: an estate
	// pseudo-agent.
	ReasonNotRelocatable Reason = "not_relocatable"
	// ReasonSameRegion is a destination that is the agent's own region.
	ReasonSameRegion Reason = "same_region"
	// ReasonOpenReservation is an account with a reservation of money or an
	// earmark of goods.
	ReasonOpenReservation Reason = "open_reservation"
	// ReasonRegionBound is an obligation, registered by a module, that is tied to
	// the region of the account; the error names it.
	ReasonRegionBound Reason = "region_bound"
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
	// CauseTransport is a relocation at the arrival of a traveller.
	CauseTransport Cause = "transport"
	// CauseMigration is a relocation at the arrival of a migration journey.
	CauseMigration Cause = "migration"
	// CauseLimit is a money flow that ended at its limit.
	CauseLimit Cause = "limit"
	// CauseCap is a money flow that ended at the cap of MaxFlowTotal.
	CauseCap Cause = "cap"
	// CausePopulation is a death of the population module: a reduction or a
	// removal by the schedule (spec-population.md 4.7).
	CausePopulation Cause = "population"
	// CauseBirths is households born through the population module, entering a
	// role (spec-population.md 4.3).
	CauseBirths Cause = "births"
	// CauseTraineeship is a role change at the completion of a traineeship.
	CauseTraineeship Cause = "traineeship"
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
	// Wants holds the string ID of each WantID; nil when the demand module is off.
	Wants []string
	// Resources holds the string ID of each ResourceID; nil when the resources
	// module is off.
	Resources []string
	// Modes holds the string ID of each ModeID; nil when the transport module is
	// off.
	Modes []string
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

// ShortRow is the units of one upkeep commodity that the households of a role
// in a region needed in a tick and did not hold.
type ShortRow struct {
	// Commodity is the upkeep commodity.
	Commodity CommodityID
	// Units is the units short, summed over the role's agents (the weight
	// folded in), above 0.
	Units int64
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
	// CostAnchor is the cost anchor of the commodity in the region in force
	// for the tick (spec-production.md 9.3), in minor units; 0 when unknown.
	CostAnchor Price
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
	// Adjusted is the number of outcomes whose adjustment gate was open.
	Adjusted int64
	// Raised is the number of outcomes whose centre ended above the stored one.
	Raised int64
	// Lowered is the number of outcomes whose centre ended below the stored one.
	Lowered int64
	// AtFloor is the number of outcomes whose centre ended on the belief floor.
	AtFloor int64
	// AtCeiling is the number of outcomes whose centre ended on the belief ceiling.
	AtCeiling int64
	// Conflicts is the number of outcomes for which the cost floor exceeded the
	// reference band.
	Conflicts int64
	// Clamped is the number of outcomes whose returned belief the world had to
	// clamp.
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
	// Outputs is the gross units created per output item, in output order:
	// discard outputs and the part of an output that had no room are included,
	// a quantity of 0 has no row.
	Outputs []Amount
	// Disposed is the part of Outputs that was never held, in output order:
	// the whole quantity of each discard output and, for any other output,
	// the part the maximum quantity left no room for. The units added to the
	// principal's account are Outputs less Disposed.
	Disposed []Amount
	// EffectiveYield is the yield applied, in basis points.
	EffectiveYield int64
	// Overflow is the part of Disposed that is not a discard output, in output
	// order: the units an output lost because the maximum quantity left no
	// room. Disposed less Overflow is the discard outputs, which are waste by
	// design; Overflow is the anomaly the "recipe output disposed" Warn and
	// the overflow metric count.
	Overflow []Amount
}

// RecipeRefused is emitted once per region and recipe per tick in which the job
// request of at least one free household was refused, so that the count of
// executions was 0 (spec-production.md 4.2, 14.1); it is never emitted when
// none was. Each count is the households refused for the reason (the sum of
// the weights of the agents), and a request is counted under its first
// applicable reason of the master order, so the counts add to the households
// refused. A request declined by the production halt is in ProductionHalted
// and a household that started fewer executions than its weight is in neither.
type RecipeRefused struct {
	Tick int64
	// Region is the region.
	Region RegionID
	// Recipe is the recipe requested.
	Recipe RecipeID
	// NotOnRole is the households whose role does not list the recipe.
	NotOnRole int64
	// ZeroYield is the households for which the effective yield was 0.
	ZeroYield int64
	// NoSite is the households for which a sited recipe had no yield profile
	// in the region.
	NoSite int64
	// NoTool is the households for which a tool item was missing.
	NoTool int64
	// NoInput is the households for which an input was not available.
	NoInput int64
	// NoEndowment is the households for which the ration granted no execution of
	// a draw (resources module).
	NoEndowment int64
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
// production halt declined the job request of at least one free household, or
// at least one free household rested, or found no recipe worth running; it is
// never emitted when none did. The name is the halt's, the first of the
// three; the event is the aggregate of the work decision.
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
	// Rested is the free households that requested no job because their work
	// share was 0: the target-income rule at work, not a clamp.
	Rested int64
	// NoRecipe is the free households that requested no job because no recipe
	// was worth running: none was eligible, and the halt skipped none and no
	// recipe was blocked.
	NoRecipe int64
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
	// Short has one row for each upkeep commodity the role's households
	// needed and did not hold, in ascending CommodityID.
	Short []ShortRow
}

// WantShortfall is emitted once per region, role and want per tick in which the
// want was choked or unfunded for at least one household (spec-demand.md 11.1),
// in order of region, role and want; it is never emitted when every household
// is fully funded.
type WantShortfall struct {
	Tick int64
	// Region, Role and Want identify the cell.
	Region RegionID
	Role   RoleID
	Want   WantID
	// Households is the households of the role in the region that have the want
	// in their ladder, short or not.
	Households int64
	// Choked is the households whose price ratio was at the choke price.
	Choked int64
	// Planned is the units wanted from the carry, over the households.
	Planned int64
	// Funded is the units the strategy placed in bids (TradePlan.WantFunded).
	Funded int64
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
	// GoodsListed is the number of lots passed to the estate.
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
	// GoodsListed is the number of lots listed.
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
	// GoodsListed is the number of lots passed to the estate.
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
	// Cooldown is the households of agents due for a change that a cooldown
	// held.
	Cooldown int64
	// Hysteresis is the households of reviewers held by the hysteresis band.
	Hysteresis int64
	// ExitCap is the households the exit allowance kept from leaving.
	ExitCap int64
	// Refused is the decisions the world refused.
	Refused int64
	// Stranded is the households at rung 1 of the failure ladder that had no
	// role to move to: none open, viable and better than their own, or none they
	// can afford. They stay and go on to rung 3.
	Stranded int64
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

// WorldRestored is the first event a successful RestoreWorld delivers to the
// observers given to it, with Tick the restored tick; a restore that fails
// delivers nothing. It is not a replay: no AgentSpawned follows it for the agents
// that exist, so an observer that counts from deltas starts from the levels of
// Snapshot, the state at the restored boundary (spec-observability.md 4.4.5).
type WorldRestored struct {
	Tick int64
	// ConfigHash is the whole hash of the configuration, the checkpoint's.
	ConfigHash [32]byte
	// Format is the format version of the checkpoint, as CheckpointInfo.Format.
	Format uint32
	// Reseeded says the restore was given WithReseed; Salt is its salt and is
	// meaningful only then.
	Reseeded bool
	Salt     uint64
	// Names maps every typed ID to its string ID.
	Names Names
	// Snapshot is the snapshot at the restored boundary; its header holds the
	// effective seed, the agents and the households.
	Snapshot Snapshot
}

// AgentRelocated is emitted once per relocation, after the move is complete: an
// agent's account changed region (spec-regions.md 6.2, 10.1). A household leaves
// From and joins To, so a consumer that counts households by region needs it
// beside AgentSpawned, AgentBankrupted, WeightChanged and RoleChanged. An estate
// pseudo-agent is not an agent and causes none.
type AgentRelocated struct {
	Tick int64
	// Agent is the agent that moved.
	Agent AgentID
	// Role is the agent's role.
	Role RoleID
	// From and To are the regions the account left and joined.
	From, To RegionID
	// Weight is the households the agent stands for.
	Weight int64
	// Money is what the account held, and Units the units of goods it held.
	Money Money
	Units int64
	// Cause is why the agent moved: transport, migration or consumer.
	Cause Cause
}

// EdgeStatusChanged is emitted once per edge whose status changed, in ascending
// EdgeID, after the world's tables for the new status are in place (spec-regions.md
// 7.3.2, 10.1). A change equal to the status in force is no event. The names of
// the regions are in the names table.
type EdgeStatusChanged struct {
	Tick int64
	// From and To are the ends of the edge.
	From, To RegionID
	// Closed and FactorBP are the new status: closed or open, and the factor of
	// the configured distance in basis points.
	Closed   bool
	FactorBP int64
	// Cause is who set the status: consumer, or policy for a policy actor.
	Cause Cause
}

// ReliefPaid is emitted once per region and tick in which the estates module paid
// any relief, in ascending RegionID, at P1.6 and so before the MoneyFlowLimited
// events of the same tick; never when nothing was paid and never with the
// module off (spec-estates.md 2.6, FR-EST-020). It carries no per-agent rows.
type ReliefPaid struct {
	Tick   int64
	Region RegionID
	// Paid is the money the pool paid, by the reason relief.
	Paid Money
	// Recipients is the number of agents paid and Households the sum of their
	// weights.
	Recipients, Households int64
}

// MoneyFlowLimited is emitted once per money flow and region in a tick in which
// part of a due was not applied, after the step of P1.7, flows in ID order and
// regions in ascending order; never when all was applied (spec-money-supply.md
// 13.1). A household is counted in the first count that cut it only.
type MoneyFlowLimited struct {
	Tick int64
	// Flow is the flow's ID, also the Reason of its BoundaryFlow rows.
	Flow   string
	Region RegionID
	// Due and Applied are the amounts the flow computed and applied in the region.
	Due, Applied Money
	// Short, Headroom and Limit are counts of households (the sum of agent
	// weights), not money. Short is the households whose levy found too little, Headroom the
	// households whose credit was cut at MaxMoney and Limit the households cut by
	// the flow's limit.
	Short, Headroom, Limit int64
}

// MoneyFlowEnded is emitted once, in the tick a money flow reaches its effective
// limit and ends, after the MoneyFlowLimited events of the same flow
// (spec-money-supply.md 13.1).
type MoneyFlowEnded struct {
	Tick int64
	Flow string
	// Cause is CauseLimit or CauseCap.
	Cause Cause
	// Total is the cumulative amount the flow applied.
	Total Money
}

// MoneyFlowScaled is emitted once per money flow whose scale is changed, at the
// call or at P0.3a (spec-money-supply.md 13.1).
type MoneyFlowScaled struct {
	Tick int64
	Flow string
	// From and To are the scale before and after, in basis points.
	From, To int64
	// Cause is CauseConsumer or, for a policy actor, CausePolicy.
	Cause Cause
}

// EndowmentChanged is emitted once per endowment in a tick in which any of the units
// drawn, regenerated, short or lost is non-zero, after its regeneration at P8.3
// and in ascending (RegionID, ResourceID) order; never otherwise and never with
// the resources module off (spec-resources.md 14.1, FR-RES-026). It carries no
// per-agent rows.
type EndowmentChanged struct {
	Tick     int64
	Region   RegionID
	Resource ResourceID
	// Stock is the units held after the tick and Capacity the effective capacity;
	// AbundanceBP is the abundance of the snapshot of the tick.
	Stock, Capacity, AbundanceBP int64
	// Requested is the units asked (the sum of executions times draw), Granted the
	// units the ration granted, Drawn the units the starts took (at most Granted)
	// and Short is Requested less Granted.
	Requested, Granted, Drawn, Short int64
	// Regenerated is the units kept by the regeneration after the clamp at the
	// capacity, and Lost the units a capacity clamp removed this tick.
	Regenerated, Lost int64
	// AgentsDenied counts the requests granted 0 and AgentsReduced those granted
	// less than they asked but more than 0.
	AgentsDenied, AgentsReduced int64
}

// AdjustKind says what changed an endowment between ticks (spec-resources.md 14.1).
type AdjustKind string

// The kinds of EndowmentAdjusted, a closed set.
const (
	// AdjustKindAdjust is a signed change of the stock by the consumer.
	AdjustKindAdjust AdjustKind = "adjust"
	// AdjustKindShock is a change of a capacity or regeneration scale.
	AdjustKindShock AdjustKind = "shock"
	// AdjustKindFacility is a clamp of the stock by a facility that lowers the
	// effective capacity.
	AdjustKindFacility AdjustKind = "facility"
)

// EndowmentAdjusted is emitted once per changed endowment of a call that adjusts a
// stock or sets a shock scale between ticks (spec-resources.md 14.1, 6.3). Its
// Tick is the tick the next step runs.
type EndowmentAdjusted struct {
	Tick     int64
	Region   RegionID
	Resource ResourceID
	Kind     AdjustKind
	// Delta is the signed change of the stock for the kind adjust, after the clamp
	// to [0, capacity]; 0 for the others.
	Delta int64
	// CapacityBP and RegenBP are the scales after the call, in basis points.
	CapacityBP, RegenBP int64
	// StockBefore and StockAfter are the stock around the call, and Lost the units
	// a capacity clamp removed.
	StockBefore, StockAfter, Lost int64
}

func (WorldCreated) isEvent() {}

// OwnUseChanged is emitted once per region and role per tick in phase 2, after the
// starts, in ascending RegionID then role order, if any of Entered, Left, Started,
// Blocked, Retried or Output is non-zero; it is never emitted otherwise
// (spec-subsistence.md 14.1, FR-SUB-040, 041). It is emitted only when the
// subsistence module is on. A change of OnOwnUse always carries Entered or Left.
type OwnUseChanged struct {
	Tick int64
	// Region and Role are the cell.
	Region RegionID
	Role   RoleID
	// Households is the sum of the weights of the role in the region and OnOwnUse
	// the households with the flag at the end of the flag step.
	Households, OnOwnUse int64
	// Entered and Left are the households whose flag changed in the tick.
	Entered, Left int64
	// Started is the executions of own-use recipes started.
	Started int64
	// Blocked is the households in the mode that were free and could run no
	// own-use recipe at P2.5, or whose own-use requests the ration or the check of
	// the start refused at P2.6 or P2.7, and which started nothing; a household
	// counts once.
	Blocked int64
	// Retried is the households for which the retry chose a role recipe over a
	// runnable own-use recipe.
	Retried int64
	// Output is the units created at the completion of own-use jobs in the tick,
	// in commodity order.
	Output []Amount
}

// ScaleKind names which of a region's two population scales a call sets, a member
// of a closed set (spec-observability.md 4.1; spec-population.md 3.6).
type ScaleKind string

// The two scales of a region.
const (
	// ScaleBirth is the multiplier of the birth schedule.
	ScaleBirth ScaleKind = "birth"
	// ScaleDeath is the multiplier of the death schedule.
	ScaleDeath ScaleKind = "death"
)

// PopulationChanged is the aggregate of the population module for one region and
// role: emitted once per region and role per tick, in ascending role order, after
// the region's plan and after EntryLimited, when any count other than Exposed,
// Unfed and NoRatio is non-zero, and never otherwise (spec-population.md 10.1,
// FR-POP-031). It is emitted only when the population module is on. The counts
// are households. BirthsCeiling, the births a role ceiling cut, is appended with
// the settlements module.
type PopulationChanged struct {
	Tick int64
	// Region and Role are the cell.
	Region RegionID
	Role   RoleID
	// Exposed is the households the schedule saw; Unfed those it did not because
	// they were unfed, and NoRatio those whose agent has no welfare ratio.
	Exposed, Unfed, NoRatio int64
	// BirthsDue and DeathsDue are the whole households the carries produced,
	// before the floors and the cap.
	BirthsDue, DeathsDue int64
	// DeathsCut is the deaths cut to an agent's fed households and DeathsHeld
	// those dropped by MinHouseholds.
	DeathsCut, DeathsHeld int64
	// BirthsCapped is the births dropped by the region's HouseholdCap.
	BirthsCapped int64
	// BirthsUnfunded is the births the pool could not fund.
	BirthsUnfunded int64
	// RateClamped is the exposed households whose effective rate was clamped at
	// 10 000.
	RateClamped int64
	// BirthsDone and DeathsDone are the households the module added and removed.
	BirthsDone, DeathsDone int64
}

// PopulationScaled is emitted by a call that sets the birth or death scale of a
// region, once per changed key, with the next step's Tick (spec-observability.md
// 4.4.4). It is emitted only when the population module is on.
type PopulationScaled struct {
	Tick int64
	// Region is the region and Which the scale set.
	Region RegionID
	Which  ScaleKind
	// BeforeBP and AfterBP are the scale before and after, in basis points.
	BeforeBP, AfterBP int64
}

// ShipmentDispatched is emitted when a shipment is dispatched, in phase 3 in the
// plan order of the shipper, before its OrderPlaced events (spec-transport.md
// 13.1). It is emitted only when the transport module is on. Lines is a copy in
// commodity order, at most MaxShipmentLines rows; Toll and LossBP are appended
// with the fiscal and risk modules.
type ShipmentDispatched struct {
	Tick int64
	// Shipment is the ID of the shipment and Mode its mode.
	Shipment ShipmentID
	Mode     ModeID
	// Region is the origin and To the destination.
	Region, To RegionID
	// Shipper is the agent that asked and Traveller the agent that goes (the
	// shipper, for a self-carried trip; unused for a hired one).
	Shipper, Traveller AgentID
	// Hired says the trip is by a hired mode.
	Hired bool
	// Consignee receives the cargo.
	Consignee AgentID
	// Arrives is the tick of the arrival.
	Arrives int64
	// Units is the carrying units used and HaulUnits the haul units redeemed.
	Units, HaulUnits int64
	// Lines is what is shipped, by commodity.
	Lines []Amount
}

// ShipmentRejected is emitted when a shipment request is refused, in the plan
// order of the agent, before its OrderPlaced events (spec-transport.md 13.1). It
// is emitted only when the transport module is on.
type ShipmentRejected struct {
	Tick int64
	// Region is the origin region and Agent the shipper.
	Region RegionID
	Agent  AgentID
	// Mode and To are the request's mode and destination as given.
	Mode ModeID
	To   RegionID
	// Reason is why the request was refused.
	Reason Reason
}

func (RunStarted) isEvent()         {}
func (RunEnded) isEvent()           {}
func (TickStarted) isEvent()        {}
func (TickEnded) isEvent()          {}
func (OrderPlaced) isEvent()        {}
func (OrderRejected) isEvent()      {}
func (TradeExecuted) isEvent()      {}
func (MarketCleared) isEvent()      {}
func (BeliefsUpdated) isEvent()     {}
func (RecipeExecuted) isEvent()     {}
func (RecipeRefused) isEvent()      {}
func (RecipeAbandoned) isEvent()    {}
func (ProductionHalted) isEvent()   {}
func (UpkeepShortfall) isEvent()    {}
func (WantShortfall) isEvent()      {}
func (AgentSpawned) isEvent()       {}
func (AgentBankrupted) isEvent()    {}
func (EstateOpened) isEvent()       {}
func (WeightChanged) isEvent()      {}
func (RoleChanged) isEvent()        {}
func (EntryLimited) isEvent()       {}
func (BoundaryFlow) isEvent()       {}
func (WearApplied) isEvent()        {}
func (WorldRestored) isEvent()      {}
func (AgentRelocated) isEvent()     {}
func (EdgeStatusChanged) isEvent()  {}
func (ReliefPaid) isEvent()         {}
func (MoneyFlowLimited) isEvent()   {}
func (MoneyFlowEnded) isEvent()     {}
func (MoneyFlowScaled) isEvent()    {}
func (EndowmentChanged) isEvent()   {}
func (EndowmentAdjusted) isEvent()  {}
func (OwnUseChanged) isEvent()      {}
func (PopulationChanged) isEvent()  {}
func (PopulationScaled) isEvent()   {}
func (ShipmentDispatched) isEvent() {}
func (ShipmentRejected) isEvent()   {}
func (ShipmentArrived) isEvent()    {}

// ShipmentArrived is emitted when a shipment arrives, after its delivery and after
// the AgentRelocated of a traveller, in shipment order in phase 1 (spec-transport.md
// 13.1). It is emitted only when the transport module is on. Lines are copies in
// commodity order; AccountLost is appended with the risk module.
type ShipmentArrived struct {
	Tick int64
	// Shipment and Mode identify the shipment; Region is the origin and To the
	// destination.
	Shipment ShipmentID
	Mode     ModeID
	Region   RegionID
	To       RegionID
	// Receiver is the consignee, or the estate pseudo-agent of the destination
	// when the cargo was diverted.
	Receiver AgentID
	// Diverted says some of the cargo went to the estate.
	Diverted bool
	// Lines are what each commodity did.
	Lines []ArrivedLine
}

// ArrivedLine is what became of one line of a shipment: the units shipped, the
// units delivered to the consignee, diverted to the estate of the destination and
// spoiled in transit. They add up to Shipped. Lost is appended with the risk
// module.
type ArrivedLine struct {
	Commodity CommodityID
	Shipped   Quantity
	Delivered Quantity
	Diverted  Quantity
	Spoiled   Quantity
}

// TransitFactorChanged is emitted by a call that sets the transit factor of an
// (origin, destination, mode), once per changed key, with the next step's Tick, or
// by a policy actor at P0.3a (spec-observability.md 4.4.4). It is emitted only when
// the transport module is on.
type TransitFactorChanged struct {
	Tick int64
	// From and To are the regions and Mode the mode.
	From, To RegionID
	Mode     ModeID
	// BeforeBP and AfterBP are the factor before and after, in basis points.
	BeforeBP, AfterBP int64
	// Cause is CauseConsumer; the policy cause joins with the policy module.
	Cause Cause
}

func (TransitFactorChanged) isEvent() {}

// The results of a candidate rumour that is not offered, counted by RumoursServed
// (spec-information.md 5.3.1, 8). The fourth result, Offered, is the field of that
// name and the literal "offered" of the metric attribute, which the adapter
// spells; so the four attribute values are these three and "offered".
const (
	// ReasonNoPath is a candidate with no walk over open edges from its source
	// region to the reader's.
	ReasonNoPath Reason = "no_path"
	// ReasonTooOld is a candidate whose lag is above the horizon RumourMaxAge.
	ReasonTooOld Reason = "too_old"
	// ReasonNotYet is a candidate whose vintage tick would be before tick 0.
	ReasonNotYet Reason = "not_yet"
)

// RumoursServed is emitted once per reader region per tick in which the
// information module is on and at least one informed agent was given a view in
// that region, in ascending RegionID, at P3.6, after the last plan of phase 3 and
// before phase 4 (spec-information.md 10.1). It is the counter of the three
// silent limits of the module: the closed road, the horizon and the young run.
// The calls of the migration pass are not counted. It is bounded by the regions,
// so it is not per agent.
type RumoursServed struct {
	Tick int64
	// Region is the reader's region.
	Region RegionID
	// Readers is the informed agents given a view in the region.
	Readers int64
	// Offered, NoPath, TooOld and NotYet are the candidates of those views by
	// their result; they add up to the candidates, and Offered is the rumours
	// placed in the views.
	Offered, NoPath, TooOld, NotYet int64
	// MaxLag is the largest lag among the offered rumours, 0 if none.
	MaxLag int64
}

func (RumoursServed) isEvent() {}

// MigrationStarted is emitted once per departure of the migration pass, in the
// order of admission, after the last departure of the pass and after the
// RecipeAbandoned events of the departures (spec-migration.md 8.2, 10.2, 13.1,
// FR-MIG-035). It is per agent, like AgentRelocated; the arrival is that event
// with the cause migration.
type MigrationStarted struct {
	Tick int64
	// Agent and Role are the agent that left and its role.
	Agent AgentID
	Role  RoleID
	// From and To are the region of departure and of arrival.
	From, To RegionID
	// Weight is the households the agent stands for.
	Weight int64
	// Arrives is the arrival tick, fixed now; Distance is the graph's distance and
	// Journey the ticks of the journey.
	Arrives, Distance, Journey int64
	// EarnedRatio is the push (basis points, signed), Home the household's own
	// measured figure and Estimate the figure of the destination (basis points).
	EarnedRatio, Home, Estimate int64
	// JobAbandoned is whether a job in progress was abandoned by the departure.
	JobAbandoned bool
}

func (MigrationStarted) isEvent() {}

// MigrationReviewed is emitted once per region per tick, in ascending RegionID
// after the MigrationStarted events, in which any household was pushed or held
// back by the settling rule, any arrival was held or any journey ended by a
// removal or a cancel, and never otherwise (spec-migration.md 13.1, FR-MIG-036).
// It counts households, attributed to the region of the account, and is the
// counter of every silent limit of the module.
type MigrationReviewed struct {
	Tick   int64
	Region RegionID
	// Unsettled is the households whose turn came, that the push would have chosen
	// and that were not settled; Reviewed those eligible and settled; Pushed those
	// among them that were pushed.
	Unsettled, Reviewed, Pushed int64
	// The outcomes of the pushed households (spec-migration.md 7.6); they add up to
	// Pushed.
	Started, NoEstimate, NoDestination, NoRumour, TooFar, NoViable, NoGain int64
	Bound, OutCap, Unprovisioned, InCap                                    int64
	// Held is the arrivals held by an obligation and Ended the journeys ended by a
	// removal or a cancel since the last pass.
	Held, Ended int64
}

func (MigrationReviewed) isEvent() {}

// LaneDecision is the result of a merchant's lane decision, a closed set of the
// attribute economysim.decision (spec-merchants.md 5.1.2, 15.3). It is a type of
// its own, distinct from Reason, so that DecisionNoRumour does not repeat the
// value of ReasonNoRumour.
type LaneDecision string

// The eleven decisions of a merchant's tick, the names of the counts of PlanNotes
// and of MerchantsPlanned (spec-merchants.md 5.1.2, 6.4, 6.5, 7.1).
const (
	// DecisionNoRumour is a candidate with no rumour for its lane.
	DecisionNoRumour LaneDecision = "no_rumour"
	// DecisionStale is a candidate whose rumour is older at its source than the
	// block allows.
	DecisionStale LaneDecision = "stale"
	// DecisionLocked is a candidate whose lane has units on the way already.
	DecisionLocked LaneDecision = "locked"
	// DecisionPerishable is a candidate whose cargo would spoil on the way.
	DecisionPerishable LaneDecision = "perishable"
	// DecisionNoMargin is a candidate whose cautious revenue is not above the local
	// price.
	DecisionNoMargin LaneDecision = "no_margin"
	// DecisionUnaffordable is a candidate whose lot is 0.
	DecisionUnaffordable LaneDecision = "unaffordable"
	// DecisionBelowShare is a candidate whose carriage costs more than the fee
	// share of its gross margin.
	DecisionBelowShare LaneDecision = "below_share"
	// DecisionPlanned is a passing candidate that was planned.
	DecisionPlanned LaneDecision = "planned"
	// DecisionRested is a merchant at its target income that planned nothing.
	DecisionRested LaneDecision = "rested"
	// DecisionWaiting is a merchant that stayed: goods inbound, a store, a job.
	DecisionWaiting LaneDecision = "waiting"
	// DecisionLotCapped is a lot cut by the cap of the block.
	DecisionLotCapped LaneDecision = "lot_capped"
)

// MerchantsPlanned is emitted once per region per tick in which at least one
// agent with a merchant block was given a view, in ascending RegionID at P3.3a,
// after ContractRound and before RumoursServed, also with the module absent
// (spec-merchants.md 15.1.1). It is the sum of the PlanNotes of the plans of the
// region, a count of lanes and cases, and is bounded by the regions, not per
// agent.
type MerchantsPlanned struct {
	Tick   int64
	Region RegionID
	// Merchants is the agents with a merchant block given a view in the region.
	Merchants int64
	// Considered is the candidates, and NoRumour to BelowShare how they were
	// decided; they add up to Considered less the passing candidates.
	Considered, NoRumour, Stale, Locked, Perishable, NoMargin, Unaffordable, BelowShare int64
	// Planned is the trips planned, Dispatched the shipment requests, Offered the
	// delivery offers and Taken the delivery takes posted.
	Planned, Dispatched, Offered, Taken int64
	// Rested, Waiting and LotCapped count a rest, a stay and a lot cut by the cap.
	Rested, Waiting, LotCapped int64
}

func (MerchantsPlanned) isEvent() {}

// ContractFormed is emitted when a delivery contract is formed, at P3.3 in
// formation order, after the ShipmentDispatched of its shipment (spec-merchants.md
// 9.5, 15.1.1). It is per contract, like ShipmentDispatched.
type ContractFormed struct {
	Tick int64
	// Contract is the contract, Payer the consignee that bought the goods and
	// Deliverer the shipper that sells them.
	Contract  ContractID
	Payer     AgentID
	Deliverer AgentID
	// Region is the deliverer's region, the origin, and To the destination, where
	// the payer's account is.
	Region RegionID
	To     RegionID
	// Commodity, Quantity and Price are the goods and the price per unit.
	Commodity CommodityID
	Quantity  Quantity
	Price     Price
	// Escrowed is Price times Quantity, the money moved to the Escrow account.
	Escrowed Money
	// Shipment is the shipment that carries the goods, Mode its mode and Arrives
	// the tick it arrives, which is the deadline of the contract.
	Shipment ShipmentID
	Mode     ModeID
	Arrives  int64
}

func (ContractFormed) isEvent() {}

// The causes of ContractSettled (spec-merchants.md 15.1.1), constants of Cause.
const (
	// CauseDelivered is a contract whose goods all arrived.
	CauseDelivered Cause = "delivered"
	// CauseShort is a contract of which units spoiled or were lost on the way.
	CauseShort Cause = "short"
	// CausePayerRemoved is a contract whose payer was removed before the arrival:
	// the cargo went to the estate and so did the refund.
	CausePayerRemoved Cause = "payer_removed"
	// CauseDelivererRemoved is a contract that ended at the removal of its
	// deliverer, the whole escrow returned.
	CauseDelivererRemoved Cause = "deliverer_removed"
)

// ContractSettled is emitted when a delivery contract ends: in phase 1 at P1.4,
// after the ShipmentArrived events, in contract ID order, or with the removal of
// its deliverer in phase 7 (spec-merchants.md 9.7, 15.1.1). It is per contract.
type ContractSettled struct {
	Tick int64
	// Contract is the contract, Payer the consignee and Deliverer the shipper.
	Contract  ContractID
	Payer     AgentID
	Deliverer AgentID
	// Region is the deliverer's region at the event and To the destination.
	Region RegionID
	To     RegionID
	// Commodity and Quantity are the goods the contract named.
	Commodity CommodityID
	Quantity  Quantity
	// Reached is the units that reached the destination, delivered or diverted;
	// Spoiled the units spoiled in transit and Lost the units lost (0 with the risk
	// module off). Reached is Quantity less Spoiled and Lost.
	Reached, Spoiled, Lost Quantity
	// Paid is the money paid to the deliverer and Refunded the money returned to the
	// payer, or to the estate of To when the payer was removed.
	Paid, Refunded Money
	// Cause says why the contract ended.
	Cause Cause
}

func (ContractSettled) isEvent() {}

// ContractRound is emitted once per destination region and tick in which an offer
// or a take was submitted for it, in ascending RegionID at P3.3 after the last
// ContractFormed of the round (spec-merchants.md 15.1.1). It is bounded by the
// regions, not per contract.
type ContractRound struct {
	Tick int64
	// Region is the destination.
	Region RegionID
	// Offers and Takes are the admitted, OffersRefused and TakesRefused the refused.
	Offers, Takes, OffersRefused, TakesRefused int64
	// Matched is the tentative matches, Formed the contracts formed, NoTaker the
	// offers that found no take, and PayerFunds, TooMany and DispatchRefused the
	// matches that failed in formation by that reason.
	Matched, Formed, NoTaker, PayerFunds, TooMany, DispatchRefused int64
	// Units and Value are the units and the money of the contracts formed.
	Units int64
	Value Money
}

func (ContractRound) isEvent() {}

// The reasons the external module adds to the refusals of other modules
// (spec-external-sector.md 4.4, D533).
const (
	// ReasonOutsideNode is a relocation into an outside node: row 5a of
	// spec-regions.md 6.1.3, between same_region and open_reservation.
	ReasonOutsideNode Reason = "outside_node"
	// ReasonOutsideDestination is a shipment request whose destination is an
	// outside node: row 6a of spec-transport.md 5.4, between unknown_region and
	// no_route.
	ReasonOutsideDestination Reason = "outside_destination"
)

// The causes that end a side of an outside node (spec-external-sector.md 7.3, D533).
const (
	// CauseUnits is a side whose cumulative units reached the cap TotalUnits.
	CauseUnits Cause = "units"
	// CauseValue is a side whose remaining list value fell below the limit price of
	// the tick.
	CauseValue Cause = "value"
)

// OutsideTraded is emitted once per (node, gate region, commodity, side) per tick
// in which the side wanted any unit or any unit was cut, after the sweep of P4.2,
// in ascending (Node, Region, Commodity, buy before sell); it is never emitted when
// nothing was wanted or cut, and only while the external module is on
// (spec-external-sector.md 15.1, FR-EXT-036). The counts are units: Wanted is what
// the side wanted, Offered what it placed after the cuts, Traded what changed hands
// (the sweep), and the three Cut fields the wanted units not offered, by cause.
type OutsideTraded struct {
	Tick int64
	// Node is the outside node and Region the gate; Commodity and Side name the side
	// (Bid for a buy side, Ask for a sell side).
	Node, Region RegionID
	Commodity    CommodityID
	Side         Side
	// Mid is the mid of the line this tick and Price the limit of the order, the
	// quote at the gate; Price is 0 when no quote was made (a closed edge).
	Mid, Price Price
	// Wanted, Offered and Traded are units.
	Wanted, Offered, Traded int64
	// Value is the money paid or received in the trades and ListValue the units
	// traded times the limit (7.4).
	Value, ListValue Money
	// CutClosed, CutCap and CutPrice are the wanted units not offered because the
	// edge was closed, the caps cut them, or the bid fell below 1.
	CutClosed, CutCap, CutPrice int64
	// PriceClamped says a clamp of the factor or of the price acted (6.3).
	PriceClamped bool
}

func (OutsideTraded) isEvent() {}

// OutsideEnded is emitted once when a side row reaches its cumulative cap and ends,
// after the OutsideTraded events of the sweep (spec-external-sector.md 7.3, 15.1,
// FR-EXT-036). Units and ListValue are the cumulative totals.
type OutsideEnded struct {
	Tick int64
	// Node is the outside node and Region the gate; Commodity and Side name the side.
	Node, Region RegionID
	Commodity    CommodityID
	Side         Side
	// Cause is CauseUnits or CauseValue.
	Cause Cause
	// Units and ListValue are the cumulative units and list value of the side row.
	Units     int64
	ListValue Money
}

func (OutsideEnded) isEvent() {}

// OutsideShocked is emitted once per changed key of a call that sets the shock of an
// outside line between ticks (spec-external-sector.md 6.5, 15.1, FR-EXT-037). From
// and To are the factor before and after, in basis points. Its Tick is the tick the
// next step runs.
type OutsideShocked struct {
	Tick int64
	// Node is the outside node and Commodity the line.
	Node      RegionID
	Commodity CommodityID
	// From and To are the shock before and after, in basis points.
	From, To int64
}

func (OutsideShocked) isEvent() {}

// YieldShocked is emitted once per changed key of a call that sets the yield shock
// of a recipe in a region between ticks (spec-integration.md 5.1, spec-production.md
// 6.8). BeforeBP and AfterBP are the factor before and after, in basis points. Its
// Tick is the tick the next step runs. It depends on no module.
type YieldShocked struct {
	Tick int64
	// Region and Recipe are the key.
	Region RegionID
	Recipe RecipeID
	// BeforeBP and AfterBP are the factor before and after, in basis points.
	BeforeBP, AfterBP int64
}

func (YieldShocked) isEvent() {}

// PayrollPaid is the aggregate of the payroll form of the employment module for
// one region and employer role: emitted once per region and role per tick at
// P4.1b, in ascending (RegionID, role) order, after the FiscalFlowApplied events
// of P4.1a, when Due or Paid is non-zero, and never otherwise and never with the
// module off (spec-employment.md 24.6.1, FR-EMP-076). It carries no per-agent
// rows. Money is in minor units.
type PayrollPaid struct {
	Tick int64
	// Region is the region of the payers and the workers, and Role the payers' role.
	Region RegionID
	Role   RoleID
	// Payers is the payers of the role with a budget of at least 1.
	Payers int64
	// Workers is the households paid by the payers of the role, counted once per
	// execution (spec-employment.md 24.6.1): the sum of the executions of the
	// workers that at least one payer of the role paid, not a head count.
	Workers int64
	// Due is the share of the payers' receipts the rule asked for, Paid what was
	// paid by the reason wage_pay, and Cut Due less Paid: the part cut at a
	// payer's balance, or all of Due when the region had no worker to pay.
	Due, Paid, Cut Money
}

func (PayrollPaid) isEvent() {}

// TraineeshipStarted is emitted when a place is filled from the queue and an
// applicant becomes a trainee, in a region's entry step at P7.1(a), after the
// completions of the step (spec-skill-barriers.md 4.7, 8.1; spec-observability.md
// 4.4.8). It carries identifiers only. Region is the trainee's region at the
// event.
type TraineeshipStarted struct {
	Tick int64
	// Trainee is the agent that became a trainee, and Trainer the agent that
	// teaches it.
	Trainee, Trainer AgentID
	// Role is the role the trainee is training for, the trainer's role.
	Role RoleID
	// Region is the region of the trainee.
	Region RegionID
}

func (TraineeshipStarted) isEvent() {}

// TraineeshipCompleted is emitted when a trainee completes its course and
// changes role, after the RecipeAbandoned of its job if it had one and before the
// RoleChanged it causes (spec-skill-barriers.md 4.2, 4.7; spec-observability.md
// 4.4.8). Region is the agent's region at the event.
type TraineeshipCompleted struct {
	Tick int64
	// Agent is the agent that completed.
	Agent AgentID
	// Region is the agent's region.
	Region RegionID
	// From is the role the agent held while it trained, and To the role it
	// takes.
	From, To RoleID
}

func (TraineeshipCompleted) isEvent() {}

// TraineeshipEnded is emitted when a traineeship ends without completing: with
// the removal of the trainer or of the trainee, before the events of the removal
// (spec-skill-barriers.md 4.6, 8.1; spec-observability.md 4.4.8). Region is the
// agent's region at the event. Reason is ReasonTrainerLeft, ReasonAgentLeft or
// ReasonRoleRemoved.
type TraineeshipEnded struct {
	Tick int64
	// Agent is the trainee whose traineeship ended.
	Agent AgentID
	// Region is the agent's region.
	Region RegionID
	// Reason is why it ended.
	Reason Reason
}

func (TraineeshipEnded) isEvent() {}
