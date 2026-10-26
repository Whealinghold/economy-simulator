package economysim

import (
	"fmt"
	"math/rand/v2"
	"slices"
)

// This file is the Strategy extension point (spec-agents.md 6, 10;
// plan-phase-1-7.md Phase 5): the interface, the plain-data views and plan it
// exchanges, and Directed. DefaultStrategy is in strategy_default.go. A module
// appends the fields and groups it owns to the types below with the module
// (architecture.md 2.10.5); the core shape is what the slice needs.

// Strategy decides what an agent does: plain data in, plain data out. The
// world calls it on the goroutine that steps the world, in the fixed order of
// the tick, and checks what each method returns (spec-agents.md 6.5).
//
// The world's random source rng is a parameter of every method. An
// implementation MUST NOT create its own random source, MUST NOT use a global
// one, and MUST NOT keep the rng it is given after the call returns: a
// strategy that does breaks the reproducibility of a run (architecture.md 7.4,
// 7.7). A method is a function of its view and the random source alone: it
// MUST NOT read the clock, a global, a map's iteration order or the network,
// MUST NOT write to the view, MUST NOT keep the view or a slice in it, and
// MUST NOT block, because Step waits for it. A panic in a strategy is not
// recovered and propagates to the caller of Step.
//
// An implementation that keeps state between calls MUST also implement the
// optional pair of the checkpoint (Stateful, from Stage 2 of the roadmap), so
// that a checkpoint can save it and a restore load it. SaveState MUST write
// every piece of state that can influence a result, MUST NOT change the
// strategy and MUST give equal bytes for equal states. LoadState MUST replace
// the whole state, MUST validate its input and MUST NOT keep the slice it is
// given. Neither method receives or may create a random source. An instance
// that holds state MUST NOT be shared between worlds. A strategy that keeps no
// state implements neither method.
//
// An implementation may call DefaultStrategy: hold one in a named field and
// delegate, since a public struct embeds nothing (architecture.md 4.7).
type Strategy interface {
	// Work is called once per tick, before production, for each agent that has
	// no job in progress and did not continue or complete one this tick. It
	// returns the recipe to start, or a request for none.
	Work(v *AgentView, rng *rand.Rand) JobRequest

	// Trade is called once per tick for each agent, after production. It
	// appends orders and discards to the plan, in the order they are to be
	// submitted. The plan arrives empty and is reused, so a strategy appends
	// and allocates nothing beyond it.
	Trade(v *AgentView, rng *rand.Rand, p *TradePlan)

	// Revise is called once per admitted order, after its market has cleared.
	// It returns the agent's new belief for the commodity; returning o.Belief
	// keeps it.
	Revise(o *OutcomeView, rng *rand.Rand) Belief
}

// Purpose says why a bid is placed. It classifies spending for the welfare
// ratio only; the market never sees it (spec-agents.md 6.3).
type Purpose int

// The purposes of a bid. PurposeOther is the zero value; a value outside this
// set is read as PurposeOther by the world.
const (
	// PurposeOther is a bid that is none of the others, such as a buffer.
	PurposeOther Purpose = iota
	// PurposeUpkeep is a bid for a good of the agent's upkeep.
	PurposeUpkeep
	// PurposeInput is a bid for an input of the planned recipe.
	PurposeInput
	// PurposeTool is a bid for a tool of the planned recipe.
	PurposeTool
)

// Intent is one order a strategy wants placed (spec-agents.md 6.3).
type Intent struct {
	// Commodity is the commodity to buy or sell.
	Commodity CommodityID
	// Side is Bid to buy or Ask to sell.
	Side Side
	// Quantity is whole units, a total for the agent (scaled by its weight).
	Quantity Quantity
	// Limit is the price per unit, never scaled by the weight.
	Limit Price
	// Target is the units the agent wants to hold; 0 means none. It is the
	// stock input of the belief rule (spec-beliefs.md 4.4).
	Target Quantity
	// Reserve is the units of this commodity set aside for this tick's upkeep.
	Reserve Quantity
	// UnitCost is the cost anchor passed to the belief update; 0 means unknown.
	UnitCost Price
	// Purpose says why a bid is placed.
	Purpose Purpose
	// WantUnits is the part of a bid's Quantity that is a want of the demand
	// module, the sum over the wants that have the commodity as an item
	// (spec-demand.md 4.6). It is 0 for an ask and with the module off, and the
	// world clamps it to Quantity and to the planned want units.
	WantUnits Quantity
}

// Discard is a quantity of a commodity the strategy wants disposed of.
type Discard struct {
	// Commodity is the commodity to dispose of.
	Commodity CommodityID
	// Quantity is the units to dispose of.
	Quantity Quantity
}

// TradePlan collects what Trade returns. The world gives the strategy a plan
// that is empty and reused. A module appends the fields it owns (shipments,
// offers, credit and cover requests) with the module.
type TradePlan struct {
	// Intents are the orders to place, in the order they are to be submitted.
	Intents []Intent
	// Discards are the quantities to dispose of, executed before the intents.
	Discards []Discard
	// WantFunded is, for each want the strategy funded, the units it placed in
	// bids (spec-demand.md 11.1). A strategy that ignores wants leaves it empty.
	WantFunded []WantFunding
}

// WantFunding is the units of one want that a plan funded.
type WantFunding struct {
	// Want is the want and Units the units funded across its items.
	Want  WantID
	Units Quantity
}

// JobRequest names the recipe to start. Executions is a maximum; 0 means the
// most the agent can run. The zero value requests a job; set NoJob to request
// none (spec-agents.md 6.3).
type JobRequest struct {
	// NoJob requests nothing; the other fields are then ignored.
	NoJob bool
	// Recipe is the recipe to start; it must be on the role's list.
	Recipe RecipeID
	// Executions is the most executions to start; 0 means the most it can run.
	Executions int64
}

// Directed is a Strategy that does what the agent's directive says and leaves
// the rest to Fallback (spec-agents.md 10.3, 10.4). A nil Fallback means:
// nothing without a directive, and Revise keeps the belief. The directive is
// read from the view (HasDirective), so Directed keeps no state of its own.
type Directed struct {
	// Fallback decides everything the directive leaves out.
	Fallback Strategy
}

// Work returns the directive's job if it has one (Job is non-nil), else the
// fallback's, else a request for none.
func (d Directed) Work(v *AgentView, rng *rand.Rand) JobRequest {
	if v.HasDirective && v.Directive.Job != nil {
		return *v.Directive.Job
	}
	if d.Fallback == nil {
		return JobRequest{NoJob: true}
	}
	return d.Fallback.Work(v, rng)
}

// Trade runs the fallback, then removes the fallback's intents for the
// commodities in Hold, replaces its intent for a commodity the directive has an
// intent for, adds the other intents of the directive in canonical commodity
// order and appends its discards. It allocates only what the plan's slices grow
// by.
func (d Directed) Trade(v *AgentView, rng *rand.Rand, p *TradePlan) {
	n0 := len(p.Intents)
	if d.Fallback != nil {
		d.Fallback.Trade(v, rng, p)
	}
	if !v.HasDirective {
		return
	}
	dir := &v.Directive
	kept := p.Intents[:n0]
	for _, in := range p.Intents[n0:] {
		if j := firstIntentFor(dir.Intents, in.Commodity); j >= 0 {
			in = dir.Intents[j]
		} else if slices.Contains(dir.Hold, in.Commodity) {
			continue
		}
		kept = append(kept, in)
	}
	p.Intents = kept
	for k, in := range dir.Intents {
		// A first intent whose commodity the fallback planned has replaced it
		// above. Any other is added; a repeat is added too, so that the
		// admission refuses it as duplicate_order rather than it vanishing.
		if firstIntentFor(dir.Intents, in.Commodity) == k && hasIntentFor(p.Intents[n0:], in.Commodity) {
			continue
		}
		p.Intents = insertIntent(p.Intents, n0, in)
	}
	p.Discards = append(p.Discards, dir.Discards...)
}

// Revise is the fallback's; with a nil Fallback it keeps the belief.
func (d Directed) Revise(o *OutcomeView, rng *rand.Rand) Belief {
	if d.Fallback == nil {
		return o.Belief
	}
	return d.Fallback.Revise(o, rng)
}

// firstIntentFor returns the index of the first intent for c, or -1.
func firstIntentFor(in []Intent, c CommodityID) int {
	for i := range in {
		if in[i].Commodity == c {
			return i
		}
	}
	return -1
}

func hasIntentFor(in []Intent, c CommodityID) bool { return firstIntentFor(in, c) >= 0 }

// insertIntent inserts in into list[from:] after the last intent whose
// commodity is not above its own, so a sorted tail stays sorted and an equal
// commodity follows its twin.
func insertIntent(list []Intent, from int, in Intent) []Intent {
	list = append(list, Intent{})
	i := len(list) - 1
	for i > from && list[i-1].Commodity > in.Commodity {
		list[i] = list[i-1]
		i--
	}
	list[i] = in
	return list
}

// UpkeepRow is one good of the effective upkeep of one household (spec-agents.md
// 5.2).
type UpkeepRow struct {
	// Commodity is the good eaten.
	Commodity CommodityID
	// Quantity is the units one household eats each tick.
	Quantity int64
}

// RecipeLine is one input of a recipe: the units of a commodity one execution
// uses.
type RecipeLine struct {
	// Commodity is the input.
	Commodity CommodityID
	// Quantity is the units one execution uses.
	Quantity int64
}

// RecipeOutput is one output of a recipe.
type RecipeOutput struct {
	// Commodity is the output.
	Commodity CommodityID
	// Quantity is the units one execution makes at a yield of 10 000.
	Quantity int64
	// CostShare is the share of the cost the output carries, in basis points.
	CostShare int64
	// Discard says the output is made and disposed of in one step.
	Discard bool
}

// RecipeTool is one tool of a recipe: it is held, not consumed, and wears.
type RecipeTool struct {
	// Commodity is the tool.
	Commodity CommodityID
	// WearPerUse is the durability units one execution removes.
	WearPerUse int64
	// Durability is the tool commodity's durability, in the units of WearPerUse.
	Durability int64
}

// MarketReport is what one market published for the view: the price, its age
// and the best unmatched bid and ask (spec-clearing.md 16).
type MarketReport struct {
	// Price is the market price; Age is the ticks since a unit last traded.
	Price, Age int64
	// BestBid and BestAsk are the best unmatched prices, valid when the flag is
	// set.
	BestBid, BestAsk       int64
	HasBestBid, HasBestAsk bool
}

// PriceWindowView is the price window of a market (spec-beliefs.md 7): its
// mean, low and high, and the position of the price in it, in basis points.
type PriceWindowView struct {
	// Mean, Low, High and Position are valid when HasWindow is set.
	Mean, Low, High, Position int64
	HasWindow                 bool
}

// CommodityView is the view of one commodity of the agent's tradable set.
type CommodityView struct {
	// Commodity is the commodity.
	Commodity CommodityID
	// Stock is the whole balance; no reservation exists yet.
	Stock int64
	// Expiring is the units that spoil at the end of this tick.
	Expiring int64
	// Belief and Bounds are the agent's price belief and its floor and ceiling.
	Belief Belief
	Bounds Bounds
	// Ref is the effective reference price.
	Ref Price
	// Tool says the commodity is a required tool of some recipe of the
	// configuration (read by the ask rule of spec-agents.md 7.3.5).
	Tool bool
	// UnitCost is the cost anchor; 0 is unknown, and the filler zeroes it for a
	// commodity the role has no recipe making (FR-AGT-048).
	UnitCost Price
	// Market is the market report of the agent's region.
	Market MarketReport
	// Window is the price window of the agent's region.
	Window PriceWindowView
}

// RecipeView is the view of one recipe of the agent's role.
type RecipeView struct {
	// Recipe is the recipe; Duration its length in ticks.
	Recipe   RecipeID
	Duration int64
	// Inputs, Outputs and Tools are the recipe's lines, in canonical order.
	Inputs  []RecipeLine
	Outputs []RecipeOutput
	Tools   []RecipeTool
	// Effective is the yield E for this region and season, in basis points.
	Effective int64
	// MaxExecutions is the most executions the agent could start now.
	MaxExecutions int64
	// Reason says why MaxExecutions is 0 (spec-production.md 4.2).
	Reason Reason
}

// BeliefTunables is the group of the fifteen belief parameters that
// beliefs.Params holds (spec-beliefs.md 11), as plain data: the world fills it
// and DefaultStrategy.Revise reads it from the view alone. MaxPrice and the
// window capacity are not in it.
type BeliefTunables struct {
	Step, AdjustProb, Significance, EvidenceShare int64
	StockLow, StockHigh, MaxAge                   int64
	WidthInit, WidthMin, WidthMax, WidthStep      int64
	CostFloor, CostCeiling                        int64
	RefBandLow, RefBandHigh                       int64
}

// Tunables are the values of spec-agents.md 12 that a strategy reads, plus the
// belief parameters. A view built by hand fills them as the world does.
type Tunables struct {
	// CoverTicks, InputJobs and OutputCoverTicks size the targets (7.3).
	CoverTicks, InputJobs, OutputCoverTicks int64
	// FavourFloor, ReserveTicks, SpendMaxShare and SpendHalfTicks shape the
	// money stack (7.4, 7.5.2).
	FavourFloor, ReserveTicks, SpendMaxShare, SpendHalfTicks int64
	// RestBand and ProduceHaltRatio shape the work decision (7.1).
	RestBand, ProduceHaltRatio int64
	// Beliefs are the belief parameters.
	Beliefs BeliefTunables
}

// AgentView is what Work and Trade read: a struct of plain values the world
// fills for one agent (spec-agents.md 6.4.1). It is valid for the call only; a
// strategy MUST NOT keep the pointer or a slice in it, and MUST NOT write to it.
// A module appends its own groups with the module.
type AgentView struct {
	// Tick is the moment of the tick; Agent, Role, Region and Weight identify
	// the agent, and External says it was added through the integration
	// interface.
	Tick     Moment
	Agent    AgentID
	Role     RoleID
	Region   RegionID
	Weight   int64
	External bool
	// Money is the agent's available cash.
	Money int64
	// Hunger and Unfed are the consecutive unfed ticks and the households unfed
	// in the last consumption phase (5.4).
	Hunger, Unfed int64
	// Ratio is the welfare ratio in basis points; it is valid when HasRatio is
	// set, and absent for an agent with no upkeep.
	Ratio    int64
	HasRatio bool
	// TargetRatio is the agent's target ratio in basis points.
	TargetRatio int64
	// WorkingTicks is the working ticks in the effective window Window.
	WorkingTicks, Window int64
	// StartsIn is the ticks until the agent can next start a job (6.4.3).
	StartsIn int64
	// MinWorkShare is the role's floor on the share of working ticks, in basis
	// points; it is owned by spec-welfare-and-wages.md and read here.
	MinWorkShare int64
	// Upkeep is the effective upkeep of one household (5.2).
	Upkeep []UpkeepRow
	// HasJob says a job is in progress, of Recipe, with Remaining ticks and
	// Executions.
	HasJob                bool
	Recipe                RecipeID
	Remaining, Executions int64
	// Commodities is the tradable set in canonical order; Recipes the role's
	// recipes in ascending RecipeID.
	Commodities []CommodityView
	Recipes     []RecipeView
	// Tunables are the values of spec-agents.md 12 and the belief parameters.
	Tunables Tunables
	// HasDirective says Directive is the agent's pending directive, copied by
	// value into the view (6.4.2).
	HasDirective bool
	Directive    Directive
	// Wants is the ladder of the agent's role in rank order, with the planned
	// units of each want this tick; nil when the demand module is off or the
	// role has no ladder (spec-demand.md 4.4).
	Wants []WantView
	// scratch is the working arrays of DefaultStrategy.Trade, set by the world's
	// filler and nil in a view built by hand, for which Trade allocates its own
	// (spec-agents.md 6.4.1; NFR-AGT-001 binds the world's views only).
	scratch *viewScratch
}

// WantView is one want of the agent's ladder at this tick, computed by the
// filler from the agent's carry and its beliefs; it commits nothing.
type WantView struct {
	// Want is the want.
	Want WantID
	// Planned is the whole units wanted this tick, over all items and the weight
	// of the agent; 0 when Choked.
	Planned Quantity
	// Choked says the price ratio is at or above the choke price, so the want is
	// skipped.
	Choked bool
	// Items splits Planned among the want's items by largest remainder, in
	// ascending CommodityID.
	Items []WantItemView
}

// WantItemView is one item of a want: its commodity, its substitution share in
// basis points and the planned units.
type WantItemView struct {
	Commodity CommodityID
	Share     int64
	Units     Quantity
}

// OutcomeView is what Revise reads for one admitted order, after its market
// cleared (spec-agents.md 6.4.4).
type OutcomeView struct {
	// Agent, Commodity and Side say which order.
	Agent     AgentID
	Commodity CommodityID
	Side      Side
	// Limit, Quantity and Filled are the order and its outcome.
	Limit            Price
	Quantity, Filled Quantity
	// Belief and Bounds are the agent's belief and its floor and ceiling.
	Belief Belief
	Bounds Bounds
	// Market is the market report after the clearing.
	Market MarketReport
	// Stock is the units held after the trades, less those reserved for this
	// tick's upkeep, not below 0.
	Stock int64
	// Target and UnitCost are the values the intent carried.
	Target   Quantity
	UnitCost Price
	// Draw is the one word the world drew for this outcome (spec-beliefs.md 4.5).
	Draw uint64
	// Params are the belief parameters, so that DefaultStrategy.Revise can call
	// the update from the view alone.
	Params BeliefTunables
}

var _ Strategy = Directed{}

var _ Stateful = Directed{}

// SaveState implements Stateful by delegating: it returns the fallback's state
// when the fallback is Stateful and (0, nil, nil) otherwise, since Directed keeps
// no state of its own (the directive is in the agent, spec-checkpoint.md 7.5). So
// Directed{Fallback: x} is saved exactly when x is.
func (d Directed) SaveState() (version uint32, state []byte, err error) {
	if s, ok := d.Fallback.(Stateful); ok {
		return s.SaveState()
	}
	return 0, nil, nil
}

// LoadState implements Stateful by delegating: it hands the version and the bytes
// to the fallback when the fallback is Stateful. When it is not, there is nothing
// to load, and anything but version 0 and no bytes is an error: it was written by
// another fallback.
func (d Directed) LoadState(version uint32, state []byte) error {
	if s, ok := d.Fallback.(Stateful); ok {
		return s.LoadState(version, state)
	}
	if version != 0 || len(state) != 0 {
		return fmt.Errorf("directed has no state to load: version %d with %d bytes", version, len(state))
	}
	return nil
}
