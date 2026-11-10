package otelobserver_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// check compares the series of a metric with the wanted value and reports a
// series that is absent or different, naming the metric and its attributes.
func check(t *testing.T, got points, name string, want int64, kv ...string) {
	t.Helper()
	if v, ok := got.get(name, kv...); !ok || v != want {
		t.Errorf("%s%v = %d, %v, want %d, true", name, kv, v, ok, want)
	}
}

// absent reports a series that exists although no event fed it.
func absent(t *testing.T, got points, name string, kv ...string) {
	t.Helper()
	if v, ok := got.get(name, kv...); ok {
		t.Errorf("%s%v = %d, want no series", name, kv, v)
	}
}

// TestMarketCleared_PriceAgeAndUnmatched verifies that a market with no trade
// for k ticks reads a price age of k, that the price is the last one cleared,
// that the unmatched units are counted by side, and that the cost anchor is the
// last one reported.
//
// Requirement: FR-OBS-060
func TestMarketCleared_PriceAgeAndUnmatched(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	for k := int64(0); k <= 3; k++ {
		tick(h, k, economysim.MarketCleared{Tick: k, Region: 1, Commodity: 2, Price: 50, PriceAge: k,
			UnmatchedBid: 3, UnmatchedAsk: 5, CostAnchor: 40})
	}
	got := h.collect()
	check(t, got, "market.price_age", 3, "commodity", "c2", "region", "r1")
	check(t, got, "market.clearing_price", 50, "commodity", "c2", "region", "r1")
	check(t, got, "production.cost_anchor", 40, "commodity", "c2", "region", "r1")
	check(t, got, "market.unmatched", 12, "commodity", "c2", "side", "bid")
	check(t, got, "market.unmatched", 20, "commodity", "c2", "side", "ask")
}

// TestBeliefs_AtBoundHeldAcrossSilentTick verifies that all outcomes on the
// ceiling read 10 000 for the ceiling and 0 for the floor, truncated toward
// zero, and that a tick with no BeliefsUpdated records both again.
//
// 1 of 3 outcomes on the floor is 1 * 10 000 / 3 = 3 333 (truncated).
//
// Requirement: FR-OBS-060
func TestBeliefs_AtBoundHeldAcrossSilentTick(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.BeliefsUpdated{Region: 0, Commodity: 1, Outcomes: 8, AtCeiling: 8, MeanCentre: 70},
		economysim.BeliefsUpdated{Region: 0, Commodity: 0, Outcomes: 3, AtFloor: 1, MeanCentre: 20})
	before := h.calls["economysim.belief.at_bound"]
	tick(h, 1)
	if n := h.calls["economysim.belief.at_bound"] - before; n != 4 {
		t.Errorf("belief.at_bound records in a silent tick = %d, want 4 (two bounds of two markets)", n)
	}
	got := h.collect()
	check(t, got, "belief.at_bound", 10000, "commodity", "c1", "region", "r0", "bound", "ceiling")
	check(t, got, "belief.at_bound", 0, "commodity", "c1", "region", "r0", "bound", "floor")
	check(t, got, "belief.at_bound", 3333, "commodity", "c0", "region", "r0", "bound", "floor")
	check(t, got, "belief.centre", 70, "commodity", "c1", "region", "r0")
}

// TestBeliefs_OutcomesAndLimited verifies that the outcomes are added, the
// conflicts and clamps are counted by limit, and a market that never reported is
// absent, not zero.
//
// Requirement: FR-OBS-094
func TestBeliefs_OutcomesAndLimited(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.BeliefsUpdated{Commodity: 1, Region: 2, Outcomes: 5, Conflicts: 2},
		economysim.BeliefsUpdated{Commodity: 1, Region: 2, Outcomes: 4, Clamped: 1})
	got := h.collect()
	check(t, got, "belief.outcomes", 9, "commodity", "c1", "region", "r2")
	check(t, got, "belief.limited", 2, "commodity", "c1", "region", "r2", "belief_limit", "conflict")
	check(t, got, "belief.limited", 1, "commodity", "c1", "region", "r2", "belief_limit", "clamped")
	absent(t, got, "belief.outcomes", "commodity", "c0", "region", "r0")
}

// TestRecipe_ExecutionsRefusedAbandonedOverflow verifies the executions (not
// one per event), the six refusal reasons, the abandon reasons and the overflow
// units by commodity.
//
// Requirement: FR-OBS-096, FR-OBS-097
func TestRecipe_ExecutionsRefusedAbandonedOverflow(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0,
		economysim.RecipeExecuted{Recipe: 0, Executions: 3, Overflow: []economysim.Amount{{Commodity: 2, Quantity: 4}}},
		economysim.RecipeExecuted{Recipe: 0, Executions: 2},
		economysim.RecipeRefused{Recipe: 0, NotOnRole: 1, ZeroYield: 2, NoSite: 3, NoTool: 4, NoInput: 5, NoEndowment: 6},
		economysim.RecipeAbandoned{Recipe: 0, Executions: 2, Reason: economysim.ReasonAgentRemoved},
		economysim.RecipeAbandoned{Recipe: 0, Executions: 1, Reason: economysim.ReasonRegionChanged})
	got := h.collect()
	check(t, got, "recipe.executions", 5, "recipe", "p0")
	check(t, got, "recipe.overflow", 4, "commodity", "c2")
	for i, r := range []string{"not_on_role", "zero_yield", "no_site", "no_tool", "no_input", "no_endowment"} {
		check(t, got, "recipe.refused", int64(i+1), "recipe", "p0", "reason", r)
	}
	check(t, got, "recipe.abandoned", 2, "recipe", "p0", "reason", "agent_removed")
	check(t, got, "recipe.abandoned", 1, "recipe", "p0", "reason", "region_changed")
}

// TestProduction_HaltedAndDeclined verifies that a household counts once for
// each recipe skipped, and that the households that rested or found no recipe
// are counted once per event, apart from the halted ones.
//
// Requirement: FR-OBS-099
func TestProduction_HaltedAndDeclined(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.ProductionHalted{Role: 1, Halted: 4, Rested: 2, NoRecipe: 1,
		Skipped: []economysim.HaltRow{{Recipe: 0, Households: 4}, {Recipe: 0, Households: 3}}})
	got := h.collect()
	check(t, got, "production.halted", 7, "role", "k1", "recipe", "p0")
	check(t, got, "work.declined", 2, "role", "k1", "work_decline", "rest")
	check(t, got, "work.declined", 1, "role", "k1", "work_decline", "no_recipe")
}

// TestAgents_CountsAndRemovals verifies that agent.removals by cause sums to
// the number of AgentBankrupted events, and that the agent and household counts
// net the spawns, removals, weight changes and role changes.
//
// Household count of k0: +3 +2 (spawns), -2 (removal), +1 (weight 2 to 3),
// -3 (role change out) = 1; of k1: +4 (spawn), +3 (role change in) = 7.
//
// Requirement: FR-OBS-057
func TestAgents_CountsAndRemovals(t *testing.T) {
	h := newHarness(t)
	h.send(created(),
		economysim.AgentSpawned{Agent: 1, Role: 0, Weight: 3}, economysim.AgentSpawned{Agent: 2, Role: 0, Weight: 2},
		economysim.AgentSpawned{Agent: 3, Role: 1, Weight: 4})
	tick(h, 0,
		economysim.AgentBankrupted{Agent: 2, Role: 0, Weight: 2, Cause: economysim.CauseInsolvent},
		economysim.WeightChanged{Agent: 1, Role: 0, From: 2, To: 3},
		economysim.RoleChanged{Agent: 1, From: 0, To: 1, Weight: 3})
	got := h.collect()
	check(t, got, "agent.count", 0, "role", "k0")
	check(t, got, "agent.count", 2, "role", "k1")
	check(t, got, "household.count", 1, "role", "k0")
	check(t, got, "household.count", 7, "role", "k1")
	check(t, got, "role.changes", 1, "role", "k1")
	check(t, got, "household.exits", 2, "role", "k0", "cause", "insolvent")
	for _, c := range []economysim.Cause{economysim.CauseStarved, economysim.CauseConsumer, economysim.CausePopulation} {
		tick(h, 1, economysim.AgentBankrupted{Role: 1, Weight: 1, Cause: c})
	}
	got = h.collect()
	check(t, got, "household.exits", 1, "role", "k1", "cause", "starved")
	check(t, got, "household.exits", 1, "role", "k1", "cause", "consumer")
	absent(t, got, "household.exits", "role", "k1", "cause", "population")
	if v := got.sum("agent.removals"); v != 4 {
		t.Errorf("agent.removals = %d, want 4, the AgentBankrupted events", v)
	}
	check(t, got, "agent.removals", 1, "role", "k0", "cause", "insolvent")
	check(t, got, "agent.removals", 1, "role", "k1", "cause", "population")
}

// TestExits_CohortCutDoesNotMoveRemovals verifies that a cohort cut by the
// unfed share raises household.exits while agent.removals stays at 0, that the
// removal at weight 0 adds the weight left, and that a population death is not in
// household.exits (household.deaths counts it: TestPopulation_BirthsAndDeaths).
//
// The cohort of 40 is cut 4 a tick: three ticks leave 28, exits 12; a death of the
// population module leaves 27, and its removal at weight 27 brings exits to 39.
//
// Requirement: FR-OBS-059
func TestExits_CohortCutDoesNotMoveRemovals(t *testing.T) {
	h := newHarness(t)
	h.send(created(), economysim.AgentSpawned{Agent: 1, Role: 0, Weight: 40})
	for k, w := range []int64{36, 32, 28} {
		tick(h, int64(k), economysim.WeightChanged{Agent: 1, Role: 0, From: w + 4, To: w, Cause: economysim.CauseStarved})
	}
	got := h.collect()
	check(t, got, "household.exits", 12, "role", "k0", "cause", "starved")
	if v := got.sum("agent.removals"); v != 0 {
		t.Errorf("agent.removals = %d, want 0 while the cohort is cut", v)
	}
	tick(h, 3,
		economysim.WeightChanged{Agent: 1, Role: 0, From: 28, To: 27, Cause: economysim.CausePopulation},
		economysim.AgentBankrupted{Agent: 1, Role: 0, Weight: 27, Cause: economysim.CauseStarved})
	got = h.collect()
	check(t, got, "household.exits", 39, "role", "k0", "cause", "starved")
	check(t, got, "household.count", 0, "role", "k0")
	absent(t, got, "household.exits", "role", "k0", "cause", "population")
}

// TestEntry_LimitedAndEntries verifies that every limit and refusal that acted
// is counted by limit, with the stranded households, and that the gross entry is
// counted by region.
//
// Requirement: FR-OBS-103
func TestEntry_LimitedAndEntries(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.EntryLimited{Region: 2, Entered: 6, Unfunded: 1, NoViableRole: 2, RegionCap: 3, RoleCap: 4,
		Cooldown: 5, Hysteresis: 6, ExitCap: 7, Refused: 8, Stranded: 9})
	got := h.collect()
	check(t, got, "household.entries", 6, "region", "r2")
	for i, l := range []string{"unfunded", "no_viable_role", "region_cap", "role_cap"} {
		check(t, got, "entry.limited", int64(i+1), "entry_limit", l)
	}
	for i, l := range []string{"cooldown", "hysteresis", "exit_cap", "refused", "stranded"} {
		check(t, got, "entry.limited", int64(i+5), "entry_limit", l)
	}
	absent(t, got, "entry.limited", "entry_limit", "role_ceiling")
}

// TestBoundaryFlow_BucketIsRecorded verifies that the construction bucket at
// Tick -1, emitted outside a step, reaches money.flow and commodity.flow, and
// that WearApplied reaches commodity.wear.
//
// Requirement: FR-OBS-050
func TestBoundaryFlow_BucketIsRecorded(t *testing.T) {
	h := newHarness(t)
	h.send(created(),
		economysim.BoundaryFlow{Tick: -1, Reason: "initial", Kind: economysim.KindFaucet, Commodity: economysim.NoCommodity, Total: 900},
		economysim.BoundaryFlow{Tick: -1, Reason: economysim.ReasonInitial, Kind: economysim.KindFaucet, Commodity: 1, Total: 30},
		economysim.WearApplied{Tick: -1, Commodity: 1, Units: 7})
	got := h.collect()
	check(t, got, "money.flow", 900, "reason", "initial", "kind", "faucet")
	check(t, got, "commodity.flow", 30, "reason", "initial", "kind", "faucet", "commodity", "c1")
	check(t, got, "commodity.wear", 7, "commodity", "c1")
	tick(h, 0, economysim.BoundaryFlow{Reason: "consumption", Kind: economysim.KindSink, Commodity: 0, Total: 5},
		economysim.BoundaryFlow{Reason: "consumption", Kind: "other", Commodity: 0, Total: 5})
	check(t, h.collect(), "commodity.flow", 5, "reason", "consumption", "kind", "sink", "commodity", "c0")
}

// TestUpkeepShort_ByCommodityAndRole verifies that the units short are counted
// by commodity and role, one series for each pair.
//
// Requirement: FR-OBS-100
func TestUpkeepShort_ByCommodityAndRole(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	short := []economysim.ShortRow{{Commodity: 0, Units: 3}, {Commodity: 2, Units: 8}}
	tick(h, 0, economysim.UpkeepShortfall{Region: 0, Role: 1, Unfed: 2, Short: short},
		economysim.UpkeepShortfall{Region: 1, Role: 1, Unfed: 1, Short: short[:1]})
	got := h.collect()
	check(t, got, "upkeep.short", 6, "commodity", "c0", "role", "k1")
	check(t, got, "upkeep.short", 8, "commodity", "c2", "role", "k1")
	absent(t, got, "upkeep.short", "commodity", "c2", "role", "k0")
}

// TestBeliefs_NoOutcomesNoBoundShare verifies that a BeliefsUpdated with no
// outcomes records the centre but no share on either bound, which is undefined.
//
// Requirement: FR-OBS-060
func TestBeliefs_NoOutcomesNoBoundShare(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.BeliefsUpdated{Commodity: 1, Region: 1, MeanCentre: 9})
	got := h.collect()
	check(t, got, "belief.centre", 9, "commodity", "c1", "region", "r1")
	if n := len(got["economysim.belief.at_bound"]); n != 0 {
		t.Errorf("belief.at_bound series = %d, want 0 for an event with no outcomes", n)
	}
}

// TestBoundaryFlow_OnlyNoCommodityIsMoney verifies that a commodity outside the
// table is not taken for money: only NoCommodity is.
//
// Requirement: FR-OBS-050
func TestBoundaryFlow_OnlyNoCommodityIsMoney(t *testing.T) {
	h := newHarness(t)
	h.send(created(), economysim.BoundaryFlow{Reason: "x", Kind: economysim.KindSink, Commodity: -2, Total: 4})
	if v := h.collect().sum("money.flow"); v != 0 {
		t.Errorf("money.flow = %d, want 0 for a commodity of -2", v)
	}
}

// TestCells_EachCounterAndLevelIsSentOnlyWhenItMoved verifies that the SDK is not
// called for a counter with nothing pending or a level that did not change, so a
// quiet tick costs only the held gauges.
//
// Requirement: NFR-OBS-005
func TestCells_EachCounterAndLevelIsSentOnlyWhenItMoved(t *testing.T) {
	h := newHarness(t)
	h.send(created(), economysim.AgentSpawned{Role: 0, Weight: 1})
	tick(h, 0, economysim.BoundaryFlow{Reason: "r", Kind: economysim.KindFaucet, Commodity: economysim.NoCommodity, Total: 3})
	tick(h, 1)
	tick(h, 2)
	for _, name := range []string{"economysim.money.flow", "economysim.agent.count", "economysim.household.count"} {
		if n := h.calls[name]; n != 1 {
			t.Errorf("SDK calls for %s = %d, want 1: one send, none for the quiet ticks", name, n)
		}
	}
}

// TestHot_OutsideAStepIsSentAtOnce verifies that an order, a rejection and a trade
// that arrive between ticks reach the SDK without a TickEnded.
//
// Requirement: NFR-OBS-005
func TestHot_OutsideAStepIsSentAtOnce(t *testing.T) {
	h := newHarness(t)
	h.send(created(), economysim.OrderPlaced{Commodity: 1, Region: 1}, economysim.OrderRejected{Commodity: 1, Reason: "x"},
		economysim.TradeExecuted{Commodity: 1, Region: 1, Seller: 0, Quantity: 2})
	got := h.collect()
	for name, want := range map[string]int64{"market.orders": 1, "market.orders.rejected": 1, "market.trades": 1,
		"market.volume": 2, "estate.sold": 2} {
		if v := got.sum(name); v != want {
			t.Errorf("%s = %d, want %d without a TickEnded", name, v, want)
		}
	}
}

// TestHot_IDsOnTheEdgeOfTheTablesAreDropped verifies the edges of the hot
// accumulators: a negative side, a trade in a region equal to the region count
// (which would land in the next commodity's cells), and a rejection reason beyond
// the cap of a commodity that is not the first (which would land in the previous
// commodity's last cell).
//
// Requirement: FR-OBS-052
func TestHot_IDsOnTheEdgeOfTheTablesAreDropped(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	es := []economysim.Event{
		economysim.OrderPlaced{Commodity: 0, Region: 0, Side: -1},
		economysim.TradeExecuted{Commodity: 0, Region: 3, Seller: 0, Quantity: 4},
		economysim.TradeExecuted{Commodity: 0, Region: 0, Seller: 3, Quantity: 4},
	}
	for i := 0; i < 32; i++ {
		es = append(es, economysim.OrderRejected{Commodity: 0, Reason: economysim.Reason(string(rune('A' + i)))})
	}
	es = append(es, economysim.OrderRejected{Commodity: 1, Reason: "beyond the cap"})
	tick(h, 0, es...)
	got := h.collect()
	if v := got.sum("market.orders"); v != 0 {
		t.Errorf("market.orders = %d, want 0 for a side of -1", v)
	}
	if v := got.sum("estate.sold"); v != 0 {
		t.Errorf("estate.sold = %d, want 0: region 3 is outside the table and seller 3 is an agent", v)
	}
	if v := got.sum("market.orders.rejected"); v != 32 {
		t.Errorf("market.orders.rejected = %d, want 32: the reason beyond the cap is dropped", v)
	}
	absent(t, got, "market.orders.rejected", "commodity", "c0", "reason", "beyond the cap")
}
