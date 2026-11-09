package indicator

import (
	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/measure"
)

// Wide is an unsigned 128-bit integer, high word first. It is the type of a sum
// that can pass int64 (the sum of Price * Volume of a window, spec-indicators.md
// 5.2.1). Its fields are plain so that a consumer can store a Window.
type Wide struct{ Hi, Lo uint64 }

// u converts to the type of internal/measure, inline at the call.
func (a Wide) u() measure.Uint128 { return measure.Uint128{Hi: a.Hi, Lo: a.Lo} }

// Window is a closed window of whole steps, [From, To), as plain data: the
// running sums a Tally kept, the households and pending totals at the two
// boundaries, and nothing else (spec-indicators.md 4.2.6). It holds no stocks: a
// preset that needs them takes the two snapshots beside the Window (plan 2-18
// D566). Every row is in the canonical order of its key; a table that is dense
// (Markets, Beliefs, Recipe, People, Regional) has one row per cell of the
// configuration, so its size is a function of the configuration alone
// (FR-IND-004).
//
// The rows of Dispatches, Arrivals and Settlements are filled by the events of
// the transport and merchant modules (ShipmentDispatched, ShipmentArrived,
// ContractSettled and ContractFormed), which the Tally does not read until those
// modules are in the tree; a consumer may build them itself.
type Window struct {
	// From and To are the ticks of the window, Steps the steps it ran (To - From).
	From, To, Steps int64
	// ConfigHash is the configuration hash of the world the events came from.
	ConfigHash [32]byte
	// Regions, Commodities, Roles and Recipes are the lengths of the names table,
	// which index the dense tables.
	Regions, Commodities, Roles, Recipes int
	// OpenHouseholds and CloseHouseholds are the households by (region, role) of
	// the two snapshots, at region * Roles + role.
	OpenHouseholds, CloseHouseholds []int64
	// OpenPending and ClosePending are the pending totals of the two snapshots:
	// what a call between ticks added and the next tick's BoundaryFlow will carry
	// (spec-indicators.md 7.1.5).
	OpenPending, ClosePending []economysim.FlowTotal
	// Flows is the sum of the BoundaryFlow totals of the window by (region,
	// reason, commodity), in that order, as the events carried them. Corrected
	// applies 7.1.5. Construction is the bucket of Tick -1, kept apart (7.1.4).
	Flows, Construction []economysim.FlowTotal
	// Markets and Beliefs are the sums of MarketCleared and BeliefsUpdated by
	// (region, commodity), at region * Commodities + commodity.
	Markets []MarketTally
	Beliefs []BeliefTally
	// Recipe is the executions by (region, recipe), at region * Recipes + recipe;
	// Outputs the gross and disposed units by (region, recipe, commodity).
	Recipe  []RecipeTally
	Outputs []OutputTally
	// People is the households' account by (region, role), at region * Roles +
	// role.
	People []PeopleTally
	// Regional is relief and the binding counters by region.
	Regional []RegionTally
	// Relocations is AgentRelocated by (from, to).
	Relocations []RelocTally
	// Dispatches, Arrivals and Settlements are the transport and contract rows.
	Dispatches  []DispatchTally
	Arrivals    []ArrivalTally
	Settlements []SettlementTally
}

// MarketTally is the sums of MarketCleared of one market over the window.
type MarketTally struct {
	// Ticks is the clearings seen and Active those with a volume above 0; Stuck is
	// the ticks in which units were left unmatched on both sides and nothing
	// traded (spec-indicators.md 4.2.2).
	Ticks, Active, Stuck int64
	// SumPV is the sum of Price * Volume and SumV the sum of the volumes.
	SumPV Wide
	SumV  int64
	// Price is the last published price, Low and High the lowest and highest.
	Price, Low, High economysim.Price
	// PriceAge is the last price age.
	PriceAge int64
}

// BeliefTally is the sums of BeliefsUpdated of one market over the window.
type BeliefTally struct {
	// Ticks is the events seen; Outcomes, AtFloor and AtCeiling are summed.
	Ticks, Outcomes, AtFloor, AtCeiling int64
	// LowCentre, HighCentre and MeanCentre are those of the last event.
	LowCentre, HighCentre, MeanCentre economysim.Price
	// Lowest is the lowest over the window of rdiv(10 000 * (High - Low), Mean),
	// defined where HasLowest (some event had a mean above 0).
	Lowest    BP
	HasLowest bool
}

// RecipeTally is the executions of one recipe in one region.
type RecipeTally struct{ Executions int64 }

// OutputTally is the gross units created and the units disposed of, by (region,
// recipe, commodity), summed over RecipeExecuted events.
type OutputTally struct {
	Region    economysim.RegionID
	Recipe    economysim.RecipeID
	Commodity economysim.CommodityID
	Output    int64
	Disposed  int64
}

// PeopleTally is the account of the households of one (region, role): the
// running count, the household-ticks, and the changes the events reported.
type PeopleTally struct {
	// Households is the running count at the end of the window: the count of the
	// opening snapshot moved by every event (spec-indicators.md 6.3).
	Households int64
	// HouseholdTicks is the sum over the steps of the running count at the
	// tick's start, the denominator of the unfed share (6.3).
	HouseholdTicks int64
	// Spawned and Bankrupted are the weights of AgentSpawned and AgentBankrupted;
	// WeightDelta the sum of To - From of WeightChanged; RoleIn and RoleOut the
	// weights that changed role into and out of the role; RelocIn and RelocOut
	// the weights relocated into and out of the region.
	Spawned, Bankrupted, WeightDelta, RoleIn, RoleOut, RelocIn, RelocOut int64
	// Unfed is the sum of UpkeepShortfall.Unfed and MaxHunger the largest.
	Unfed, MaxHunger int64
	// BirthsDone and DeathsDone are the households of PopulationChanged.
	BirthsDone, DeathsDone int64
}

// RegionTally is relief and the binding counters of one region, with the money
// that crosses the region under a fiscal flow (always 0 until the fiscal module,
// which adds the FiscalFlowApplied case).
type RegionTally struct {
	// ReliefPaid, ReliefRecipients and ReliefHouseholds sum ReliefPaid.
	ReliefPaid                         economysim.Money
	ReliefRecipients, ReliefHouseholds int64
	FiscalIn, FiscalOut                economysim.Money
	Binding                            BindingCounts
}

// BindingCounts are the counters of the caps, limits and refusals that acted in
// a region over the window (spec-indicators.md 8.7): the entry budget and births
// (Unfunded, BirthsCapped, BirthsUnfunded), the role-change and entry caps
// (RegionCap, RoleCap, Cooldown, Hysteresis), the ladder (ExitCap), the deaths
// (DeathsCut, DeathsHeld), the refusals (Refused) and a money flow's limit
// (FlowShort, FlowLimit).
type BindingCounts struct {
	Unfunded, RegionCap, RoleCap, Cooldown, Hysteresis, ExitCap, Refused int64
	BirthsCapped, BirthsUnfunded, DeathsCut, DeathsHeld                  int64
	FlowShort, FlowLimit                                                 int64
}

// Any reports whether any counter is above 0.
func (b BindingCounts) Any() bool { return b != BindingCounts{} }

// RelocTally is the money, units and weight relocated from one region to another.
type RelocTally struct {
	From, To economysim.RegionID
	Money    economysim.Money
	Units    int64
	Weight   int64
}

// DispatchTally is ShipmentDispatched summed by (origin, destination, mode):
// shipments, how many were hired, the haul units redeemed, the units of the
// lines and the shipments struck by a loss.
type DispatchTally struct {
	Origin, Destination economysim.RegionID
	Mode                int64
	Shipments, Hired    int64
	HaulUnits, Units    int64
	Struck              int64
}

// ArrivalTally is ShipmentArrived summed by (origin, destination, commodity).
// Shipped = Delivered + Diverted + Spoiled + Lost; AccountLost is the amount a
// self-carried trip lost to an account.
type ArrivalTally struct {
	Origin, Destination                         economysim.RegionID
	Commodity                                   economysim.CommodityID
	Shipped, Delivered, Diverted, Spoiled, Lost int64
	AccountLost                                 int64
}

// SettlementTally is the contracts' money by (deliverer's region, destination):
// Paid and Refunded from ContractSettled and Escrowed from ContractFormed.
type SettlementTally struct {
	Region, To               economysim.RegionID
	Paid, Refunded, Escrowed economysim.Money
}
