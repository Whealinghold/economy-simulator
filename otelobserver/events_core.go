package otelobserver

import (
	economysim "github.com/Whealinghold/economy-sim"
)

// Observe implements economysim.Observer. It type-switches on the three hot
// events first, then on the groups of handlers in a fixed chain; an event it
// does not know is ignored, so a new event is not a breaking change.
func (o *Observer) Observe(e economysim.Event) {
	switch ev := e.(type) {
	case economysim.OrderPlaced:
		o.orderPlaced(ev)
	case economysim.OrderRejected:
		o.orderRejected(ev)
	case economysim.TradeExecuted:
		o.tradeExecuted(ev)
	default:
		o.spanEvent(e)
		o.dispatch(e)
		o.settle()
	}
}

// dispatch gives e to the groups in turn until one handles it. Each group is a
// switch of at most ten cases, so no function passes the complexity limit.
func (o *Observer) dispatch(e economysim.Event) {
	switch {
	case o.observeLifecycle(e), o.observeMarket(e), o.observeProduction(e), o.observeAgents(e),
		o.observeFlows(e), o.observeResources(e), o.observeTransport(e), o.observeMigration(e):
	}
}

// observeLifecycle handles the events of the world and the step, and reports
// whether it handled e.
func (o *Observer) observeLifecycle(e economysim.Event) bool {
	switch ev := e.(type) {
	case economysim.WorldCreated:
		o.reset(ev.Names)
	case economysim.WorldRestored:
		o.reset(ev.Names)
		o.seed(ev.Snapshot)
	case economysim.RunStarted:
		o.runStarted(ev)
	case economysim.RunEnded:
		o.runEnded(ev)
	case economysim.TickStarted:
		o.tickStarted(ev.Tick)
	case economysim.TickEnded:
		o.tickEnded(ev)
	default:
		return false
	}
	return true
}

// observeMarket handles the events of the markets and of the boundary.
func (o *Observer) observeMarket(e economysim.Event) bool {
	switch ev := e.(type) {
	case economysim.MarketCleared:
		o.marketCleared(ev)
	case economysim.BeliefsUpdated:
		o.beliefsUpdated(ev)
	case economysim.BoundaryFlow:
		o.boundaryFlow(ev)
	case economysim.WearApplied:
		o.count(m1(iCommodityWear, ix(ev.Commodity)), ev.Units)
	default:
		return false
	}
	return true
}

// observeProduction handles the events of production and work.
func (o *Observer) observeProduction(e economysim.Event) bool {
	switch ev := e.(type) {
	case economysim.RecipeExecuted:
		o.recipeExecuted(ev)
	case economysim.RecipeRefused:
		o.recipeRefused(ev)
	case economysim.RecipeAbandoned:
		o.count(m2(iRecipeAbandoned, ix(ev.Recipe), o.reason(string(ev.Reason))), ev.Executions)
	case economysim.ProductionHalted:
		o.productionHalted(ev)
	case economysim.UpkeepShortfall:
		o.upkeepShortfall(ev)
	default:
		return false
	}
	return true
}

// observeAgents handles the events of agents, weights, roles and entry.
func (o *Observer) observeAgents(e economysim.Event) bool {
	switch ev := e.(type) {
	case economysim.AgentSpawned:
		o.agentSpawned(ev)
	case economysim.AgentBankrupted:
		o.agentBankrupted(ev)
	case economysim.WeightChanged:
		o.weightChanged(ev)
	case economysim.RoleChanged:
		o.roleChanged(ev)
	case economysim.EntryLimited:
		o.entryLimited(ev)
	default:
		return false
	}
	return true
}

// reset rebuilds the names table and the accumulators for a world, which
// forgets what a previous world left. What the old world's series still hold is
// sent first, and the levels are taken back to 0, so that a second world on the
// same adapter starts from nothing (spec-observability.md 6.4).
func (o *Observer) reset(n economysim.Names) {
	o.flushHot()
	o.flushCells(false)
	for _, s := range o.cells {
		if s.sent != 0 {
			s.n = 0
		}
	}
	o.flushCells(true)
	o.names = newNames(n)
	o.hot = newHot(&o.names)
	o.cells = map[cell]*state{}
	o.unfed = newUnfed(len(n.Regions), len(n.Roles))
	if o.inStep {
		endSpan(o.sp.stp, true, "step did not end") // a new world arrived mid-step
		o.sp.stp = nil
	}
	o.inStep = false
}

// settle sends the accumulators at once when the event arrived outside a step:
// a call between ticks has no TickEnded to flush it (plan 2-22 D647).
func (o *Observer) settle() {
	if !o.inStep {
		o.flushHot()
		o.flushCells(false)
	}
}

func (o *Observer) orderPlaced(ev economysim.OrderPlaced) {
	if ev.Side != economysim.Bid && ev.Side != economysim.Ask {
		return
	}
	nr := o.hot.nRegion
	if ev.Commodity < 0 || int(ev.Commodity) >= o.hot.nCommodity || ev.Region < 0 || int(ev.Region) >= nr {
		return
	}
	o.hot.orders.add((int(ev.Commodity)*nr+int(ev.Region))*2+int(ev.Side), 1)
	o.settle()
}

func (o *Observer) orderRejected(ev economysim.OrderRejected) {
	if ev.Commodity < 0 || int(ev.Commodity) >= o.hot.nCommodity {
		return
	}
	r := o.names.reasons.id(string(ev.Reason))
	if r < 0 {
		return
	}
	o.hot.rejected.add(int(ev.Commodity)*maxReasons+r, 1)
	o.settle()
}

// tradeExecuted counts a trade and its volume, and the volume again as an
// estate's sale where the seller is below the region count: an estate's account
// number is its region's (spec-observability.md 4.3.3, N-310).
func (o *Observer) tradeExecuted(ev economysim.TradeExecuted) {
	nr := o.hot.nRegion
	if ev.Commodity < 0 || int(ev.Commodity) >= o.hot.nCommodity || ev.Quantity <= 0 {
		return
	}
	c := int(ev.Commodity)
	o.hot.trades.add(c, 1)
	o.hot.volume.add(c, int64(ev.Quantity))
	if ev.Seller >= 0 && int(ev.Seller) < nr && ev.Region >= 0 && int(ev.Region) < nr {
		o.hot.estateSold.add(c*nr+int(ev.Region), int64(ev.Quantity))
	}
	o.settle()
}

func (o *Observer) tickStarted(tick int64) {
	o.stepStarted(tick)
	o.inStep = true
	o.stepStart = o.clock()
}

// tickEnded flushes what the step accumulated. A failed step records no
// duration and counts one failure; the other records its duration (FR-OBS-058).
func (o *Observer) tickEnded(ev economysim.TickEnded) {
	o.stepEnded(ev.Failed)
	switch {
	case ev.Failed:
		o.inst[iWorldStepFailed].counter.Add(bg, 1)
	case o.inStep:
		o.inst[iWorldStepDuration].histogram.Record(bg, o.clock().Sub(o.stepStart).Seconds())
	}
	o.inStep = false
	if !ev.Failed {
		o.endUnfed()
	} else {
		clear(o.unfed.seen) // a partial tick's reports must not mask a role the next tick omits
	}
	o.flushHot()
	o.flushCells(true)
}

func (o *Observer) marketCleared(ev economysim.MarketCleared) {
	c, r := ix(ev.Commodity), ix(ev.Region)
	o.set(m2(iMarketClearingPrice, c, r), int64(ev.Price))
	o.set(m2(iMarketPriceAge, c, r), ev.PriceAge)
	o.set(m2(iProductionCostAnchor, c, r), int64(ev.CostAnchor))
	o.counts(m2(iMarketUnmatched, c, 0), 1, int64(ev.UnmatchedBid), int64(ev.UnmatchedAsk))
}

// beliefsUpdated records the mean centre, the share of outcomes on each bound,
// the outcomes (the denominator of that share) and the updates cut.
func (o *Observer) beliefsUpdated(ev economysim.BeliefsUpdated) {
	c, r := ix(ev.Commodity), ix(ev.Region)
	o.set(m2(iBeliefCentre, c, r), int64(ev.MeanCentre))
	if ev.Outcomes > 0 {
		o.set(m3(iBeliefAtBound, c, r, boundFloor), ratioBP(ev.AtFloor, ev.Outcomes))
		o.set(m3(iBeliefAtBound, c, r, boundCeiling), ratioBP(ev.AtCeiling, ev.Outcomes))
	}
	o.count(m2(iBeliefOutcomes, c, r), ev.Outcomes)
	o.counts(m3(iBeliefLimited, c, r, 0), 2, ev.Conflicts, ev.Clamped)
}

// boundaryFlow records the money or the goods that crossed the boundary, the
// construction bucket at Tick -1 included.
func (o *Observer) boundaryFlow(ev economysim.BoundaryFlow) {
	reason, kind := o.reason(string(ev.Reason)), closedIndex(keyKind, string(ev.Kind))
	if ev.Commodity == economysim.NoCommodity {
		o.count(m2(iMoneyFlow, reason, kind), ev.Total)
		return
	}
	o.count(m3(iCommodityFlow, reason, kind, ix(ev.Commodity)), ev.Total)
}

func (o *Observer) recipeExecuted(ev economysim.RecipeExecuted) {
	o.count(m1(iRecipeExecutions, ix(ev.Recipe)), ev.Executions)
	for _, a := range ev.Overflow {
		o.count(m1(iRecipeOverflow, ix(a.Commodity)), int64(a.Quantity))
	}
}

// refusalReasons are the reasons of a refused start, in the order of the fields
// of RecipeRefused. The sixth, no_endowment, is the module of resources'.
var refusalReasons = [...]string{"not_on_role", "zero_yield", "no_site", "no_tool", "no_input", "no_endowment"}

func (o *Observer) recipeRefused(ev economysim.RecipeRefused) {
	counts := [len(refusalReasons)]int64{ev.NotOnRole, ev.ZeroYield, ev.NoSite, ev.NoTool, ev.NoInput, ev.NoEndowment}
	for i, v := range counts {
		if v != 0 {
			o.count(m2(iRecipeRefused, ix(ev.Recipe), o.reason(refusalReasons[i])), v)
		}
	}
}

// productionHalted counts a household once for each recipe skipped, and the
// households that requested no job, by why.
func (o *Observer) productionHalted(ev economysim.ProductionHalted) {
	role := ix(ev.Role)
	for _, row := range ev.Skipped {
		o.count(m2(iProductionHalted, role, ix(row.Recipe)), row.Households)
	}
	o.counts(m2(iWorkDeclined, role, 0), 1, ev.Rested, ev.NoRecipe)
}

func (o *Observer) agentSpawned(ev economysim.AgentSpawned) {
	role := ix(ev.Role)
	o.count(m1(iAgentCount, role), 1)
	o.count(m1(iHouseholdCount, role), ev.Weight)
}

// agentBankrupted removes the agent from the counts, counts the removal by its
// cause and, for a failure or a consumer's call, the households that left.
func (o *Observer) agentBankrupted(ev economysim.AgentBankrupted) {
	role := ix(ev.Role)
	o.count(m1(iAgentCount, role), -1)
	o.count(m1(iHouseholdCount, role), -ev.Weight)
	o.count(m2(iAgentRemovals, role, closedIndex(keyCause, string(ev.Cause))), 1)
	o.exit(role, ev.Cause, ev.Weight)
}

// weightChanged moves the household count, and counts a reduction as an exit.
func (o *Observer) weightChanged(ev economysim.WeightChanged) {
	role := ix(ev.Role)
	o.count(m1(iHouseholdCount, role), ev.To-ev.From)
	if ev.To < ev.From {
		o.exit(role, ev.Cause, ev.From-ev.To)
	}
}

// exit counts households that left by the failure ladder or by a consumer's
// removal. The cause population is left out: household.deaths counts it
// (FR-OBS-059).
func (o *Observer) exit(role int32, cause economysim.Cause, households int64) {
	if c := closedIndex(keyCause, string(cause)); c >= causeInsolvent && c <= causeConsumer {
		o.count(m2(iHouseholdExits, role, c), households)
	}
}

func (o *Observer) roleChanged(ev economysim.RoleChanged) {
	from, to := ix(ev.From), ix(ev.To)
	o.count(m1(iAgentCount, from), -1)
	o.count(m1(iAgentCount, to), 1)
	o.count(m1(iHouseholdCount, from), -ev.Weight)
	o.count(m1(iHouseholdCount, to), ev.Weight)
	o.count(m1(iRoleChanges, to), 1)
}

// entryLimited counts the cases that limited entry, by limit, and the gross
// entry. EntryLimited has no field for the role ceiling, so that place is 0.
func (o *Observer) entryLimited(ev economysim.EntryLimited) {
	o.counts(m1(iEntryLimited, 0), 0, ev.Unfunded, ev.NoViableRole, ev.RegionCap, ev.RoleCap, 0,
		ev.Cooldown, ev.Hysteresis, ev.ExitCap, ev.Refused, ev.Stranded)
	o.count(m1(iHouseholdEntries, ix(ev.Region)), ev.Entered)
}
