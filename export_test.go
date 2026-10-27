package economysim

import (
	"slices"
	"testing"
)

// This file is the test-only seam of the root package (architecture.md 10.2):
// it reaches the unexported tables from the external test package, so nothing
// is exported from production code. Each row of Stage 1 appends its own
// aliases and wrappers below; none replaces the file.

// Canonical returns a deep copy of c in canonical order, which the table
// constructors take (config.go canonical).
func Canonical(c Config) Config { return c.canonical() }

// DeriveCalendar exposes the derivation of calendar.go.
func DeriveCalendar(c Calendar) (days, ticks int64, err error) { return c.derive() }

// DayTable is the day-of-year table of calendar.go.
type DayTable = dayTable

// NewDayTable builds the day-of-year table of a calendar.
func NewDayTable(c Calendar) DayTable { return newDayTable(c) }

// DayTableDay reads the table at a zero-based day of the year.
func DayTableDay(t DayTable, dayOfYear int64) (month int, dayOfMonth int64, season int) {
	return t.day(dayOfYear)
}

// RecipeTable is the recipe table of recipe.go.
type RecipeTable = recipeTable

// NewRecipeTable builds the recipe table of a canonical configuration.
func NewRecipeTable(c *Config) RecipeTable { return newRecipeTable(c) }

// RecipeMakers reads the makers of a commodity.
func RecipeMakers(t *RecipeTable, id CommodityID) []RecipeID { return t.makers(id) }

// RecipeRowView is a recipe row flattened to primitives, for the external
// test, so the unexported row types need no alias.
type RecipeRowView struct {
	Inputs, Tools, Outputs []int
	Source                 bool
}

// RecipeRow flattens the row of a recipe to the commodity IDs of its lists.
func RecipeRow(t *RecipeTable, id RecipeID) RecipeRowView {
	r := t.row(id)
	var v RecipeRowView
	for _, in := range r.inputs {
		v.Inputs = append(v.Inputs, int(in.commodity))
	}
	for _, tl := range r.tools {
		v.Tools = append(v.Tools, int(tl.commodity))
	}
	for _, o := range r.outputs {
		v.Outputs = append(v.Outputs, int(o.commodity))
	}
	v.Source = r.source
	return v
}

// RoleTable is the role table of role.go.
type RoleTable = roleTable

// NewRoleTable builds the role table of a canonical configuration.
func NewRoleTable(c *Config) RoleTable { return newRoleTable(c) }

// TradableSets returns the validator's tradable sets of a canonical
// configuration, the one implementation the role table must agree with.
func TradableSets(c *Config) []map[string]bool { return tradableSets(c, newLookup(c)) }

// RoleRowView is a role row flattened to primitives for the external test.
type RoleRowView struct {
	Recipes, Trades, Tradable []int
	Member                    []bool
}

// RoleTableRow flattens the row of a role.
func RoleTableRow(t *RoleTable, id RoleID) RoleRowView {
	r := t.row(id)
	var v RoleRowView
	for _, x := range r.recipes {
		v.Recipes = append(v.Recipes, int(x))
	}
	for _, x := range r.trades {
		v.Trades = append(v.Trades, int(x))
	}
	for _, x := range r.tradable {
		v.Tradable = append(v.Tradable, int(x))
	}
	v.Member = r.member
	return v
}

// RoleTradableSomewhere reads the union flag of a commodity.
func RoleTradableSomewhere(t *RoleTable, c CommodityID) bool { return t.tradableSomewhere(c) }

// BeliefParams exposes the construction of the belief parameters of beliefs.go
// as a flat view, so the external test needs no internal type.
func BeliefParams(w WorldParams) (flat [16]int64) {
	p := beliefParams(w)
	return [16]int64{p.Step, p.AdjustProb, p.Significance, p.EvidenceShare, p.StockLow, p.StockHigh,
		p.MaxAge, p.WidthInit, p.WidthMin, p.WidthMax, p.WidthStep, p.CostFloor, p.CostCeiling,
		p.RefBandLow, p.RefBandHigh, p.MaxPrice}
}

// YieldTable is the yield table of production.go.
type YieldTable = yieldTable

// NewYieldTable builds the yield table of a canonical configuration.
func NewYieldTable(c *Config) YieldTable { return newYieldTable(c) }

// YieldSite reads the site factor of a recipe in a region at a season index.
func YieldSite(t *YieldTable, region RegionID, recipe RecipeID, season int, sited bool) (int64, bool) {
	return t.site(region, recipe, season, sited)
}

// ResidueLen reads the length of the residue layout of a role's recipes.
func ResidueLen(recipes []RecipeID, t *RecipeTable) int { return residueLen(recipes, t) }

// ResidueOffset reads the offset of a recipe in the residue layout.
func ResidueOffset(recipes []RecipeID, t *RecipeTable, recipe RecipeID) (int, bool) {
	return residueOffset(recipes, t, recipe)
}

// ProductionScriptStats counts what a production script did.
type ProductionScriptStats = scriptStats

// ProductionScript runs the scripted run of the production steps of
// production_script_test.go over a canonical configuration against a real
// ledger and returns the first failed check, naming the seed and the
// operation.
func ProductionScript(c *Config, seed uint64, ops int) (ProductionScriptStats, error) {
	return productionScript(c, seed, ops)
}

// ProductionAnchors builds the recipe and yield tables of a canonical
// configuration, runs the cost anchor step of production.go with no yield shock
// and returns a copy of the table (region x commodity) and the root's read of
// it.
func ProductionAnchors(c *Config, prices, wages []int64, runnable [][]RecipeID, season int) (table []int64, read func(RegionID, CommodityID) int64) {
	rt := newRecipeTable(c)
	yt := newYieldTable(c)
	var sh yieldShocks
	s := newProductionScratch(c, &rt)
	s.anchors(prices, wages, runnable, &yt, &sh, season)
	return append([]int64(nil), s.costs...), s.unitCost
}

// RoleWindow reads the stored effective welfare window of a role.
func RoleWindow(t *RoleTable, id RoleID) int64 { return t.row(id).window }

// EstateAccountIDs builds the ledger of a canonical configuration, opens the
// estates and returns the account ID of each estate and the ID the next opened
// account (the first household) would receive.
func EstateAccountIDs(c Config) (estates []int, next int, err error) {
	l, err := newLedger(c)
	if err != nil {
		return nil, 0, err
	}
	e, err := openEstates(l, len(c.Regions))
	if err != nil {
		return nil, 0, err
	}
	return e.accounts, l.Len(), nil
}

// SetAgentDirective sets the pending directive of an agent between ticks. The public
// call is the integration change's (plan 2-17); until it lands the fork tests reach
// the field here (architecture.md 10.2).
func SetAgentDirective(w *World, id AgentID, d *Directive) { w.agents[id].directive = d }

// RelocateForTest moves the account of agent a to region to with the cause, by
// the world's one relocation path (regions_relocate.go). The public Relocate is
// INT's; this seam goes with it.
func RelocateForTest(w *World, a AgentID, to RegionID, cause Cause) error {
	return w.relocate(a, to, cause)
}

// RegisterBoundForTest registers a region-bound obligation of the given name,
// held by the agent when held returns true. held names each instance it holds
// through SkipHasForTest, as a module does. No module is on in Stage 2, so the
// table of obligations would otherwise be empty.
func RegisterBoundForTest(w *World, name string, held func(a AgentID, skip []string) bool) {
	w.bounds.add(name, func(a *agent, skip []string) bool { return held(a.id, skip) })
}

// SkipHasForTest reports whether skip names the instance key of an obligation,
// by name or as "name:key" (spec-regions.md 6.5.2).
func SkipHasForTest(skip []string, name string, key int64) bool { return skipHas(skip, name, key) }

// SetEdgeStatusesForTest applies a batch of edge statuses with the cause by the
// world's one internal form (regions_status.go). The public SetEdgeStatuses is
// INT's (plan 2-17); this seam goes with it.
func SetEdgeStatusesForTest(w *World, changes []EdgeStatus, cause Cause) error {
	return w.setEdgeStatuses(changes, cause)
}

// DemandLadders flattens the ladder of each role of a canonical, validated
// configuration to the want IDs in funding order, indexed by RoleID. The second
// result is false when the demand module is off, where there is no state.
func DemandLadders(c *Config) ([][]WantID, bool) {
	st := newDemandState(c)
	if st == nil {
		return nil, false
	}
	out := make([][]WantID, len(st.ladder))
	for i, l := range st.ladder {
		for _, r := range l {
			out[i] = append(out[i], r.want)
		}
	}
	return out, true
}

// WorldHasDemandState reports whether the world holds the state of the demand
// module, which it does if and only if the module is on.
func WorldHasDemandState(w *World) bool { return w.demand != nil }

// WantCarries returns a copy of the carries of an agent's wants, in ladder order.
func WantCarries(w *World, id AgentID) []int64 { return slices.Clone(w.demand.carry[id]) }

// LadderOf returns the want IDs of a role's ladder in rank order.
func LadderOf(w *World, role RoleID) []WantID {
	var out []WantID
	for _, rg := range w.demand.ladder[role] {
		out = append(out, rg.want)
	}
	return out
}

// AgentRole returns the role and weight of a live agent, and the number of agents.
func AgentRole(w *World, id AgentID) (RoleID, int64) { return w.agents[id].role, w.agents[id].weight }

// AgentCount is the number of agent slots, estates included.
func AgentCount(w *World) int { return len(w.agents) }

// ChangeRoleForTest gives agent id the role to as the spawner does, with the
// fan-in call (takeRole).
func ChangeRoleForTest(w *World, id AgentID, to RoleID) {
	a := &w.agents[id]
	w.takeRole(a, w.regionOf(a), to)
}

// SetWeightForTest changes an agent's weight without touching anything else, as
// a merge or a birth does to its host.
func SetWeightForTest(w *World, id AgentID, weight int64) { w.agents[id].weight = weight }

// DemandAgentIndex is the 0-based position of an agent in its region's list.
func DemandAgentIndex(w *World, id AgentID) int64 {
	a := &w.agents[id]
	return w.agentIndex(a, w.regionOf(a))
}

// DemandPlanAllocs measures the allocations of the filler and the checker for
// one live agent of the role over runs calls, with the plan the default strategy
// made for it.
func DemandPlanAllocs(w *World, id AgentID, runs int) float64 {
	a := &w.agents[id]
	w.refreshViewCtx()
	if err := w.fillAgentView(&w.tk.view, a); err != nil {
		panic(err)
	}
	p := w.views.tradePlan()
	DefaultStrategy{}.Trade(&w.tk.view, w.rnd.rnd, p)
	return testing.AllocsPerRun(runs, func() {
		_ = w.fillAgentView(&w.tk.view, a)
		w.checkWants(a, p)
		w.demand.use = w.demand.use[:0]
	})
}

// PlannedWants returns the planned units of the agent's view, in ladder order.
func PlannedWants(w *World, id AgentID) []Quantity {
	a := &w.agents[id]
	w.refreshViewCtx()
	if err := w.fillAgentView(&w.tk.view, a); err != nil {
		panic(err)
	}
	var out []Quantity
	for _, wv := range w.tk.view.Wants {
		out = append(out, wv.Planned)
	}
	w.demand.cur.valid = false
	return out
}

// ConsumeWantForTest plans units of the commodity for the agent and runs P6.2. It
// returns what the agent held before and after, its upkeep of the commodity next
// tick, and its unfed, hunger and privation before and after.
func ConsumeWantForTest(w *World, id AgentID, c CommodityID, units int64) (before, after, upkeep int64, welfareBefore, welfareAfter [3]int64) {
	a := &w.agents[id]
	before, _, _ = w.ledger.Holding(a.account, int(c))
	upkeep = w.nextUpkeep(a, w.regionOf(a), c)
	welfareBefore = [3]int64{a.unfed, a.hunger, a.privation}
	w.demand.use = append(w.demand.use[:0], wantUse{agent: id, commodity: c, units: units})
	if err := w.p62Wants(); err != nil {
		panic(err)
	}
	after, _, _ = w.ledger.Holding(a.account, int(c))
	welfareAfter = [3]int64{a.unfed, a.hunger, a.privation}
	return before, after, upkeep, welfareBefore, welfareAfter
}

// UpkeepCommodity returns the first upkeep commodity of an agent's role.
func UpkeepCommodity(w *World, id AgentID) CommodityID {
	a := &w.agents[id]
	return w.roles.row(a.role).upkeepIn(w.regionOf(a))[0].commodity
}

// SetWantTally sets the P3.2 tally of the want in the (region, role) cell, as the
// checker would have left it, so a test of P3.6 can pose a tick. The want must be in
// the role's ladder.
func SetWantTally(w *World, region RegionID, role RoleID, want WantID, households, choked, planned, funded int64) {
	d := w.demand
	k := slices.IndexFunc(d.ladder[role], func(rg rung) bool { return rg.want == want })
	d.tally[int(region)*d.block+d.off[role]+k] = wantTally{households, choked, planned, funded}
}

// P36WantsForTest runs the demand module's part of P3.6 once.
func P36WantsForTest(w *World) { w.p36Wants() }

// SetChokedRun sets the run of the Warn "wants choked" of a cell.
func SetChokedRun(w *World, region RegionID, role RoleID, want WantID, run int64) {
	d := w.demand
	k := slices.IndexFunc(d.ladder[role], func(rg rung) bool { return rg.want == want })
	d.chokedRun[int(region)*d.block+d.off[role]+k] = run
}

// ChokedRun returns the run of the Warn "wants choked" of a cell.
func ChokedRun(w *World, region RegionID, role RoleID, want WantID) int64 {
	d := w.demand
	k := slices.IndexFunc(d.ladder[role], func(rg rung) bool { return rg.want == want })
	return d.chokedRun[int(region)*d.block+d.off[role]+k]
}

// WantShortReason is wantShortReason for a plan with the intents given and a want
// with the item commodities given.
func WantShortReason(intents []Intent, items []CommodityID, funded int64) Reason {
	wt := wantRT{}
	for _, c := range items {
		wt.items = append(wt.items, wantItemRT{commodity: c})
	}
	return wantShortReason(&TradePlan{Intents: intents}, &wt, funded)
}

// P17Flows runs P1.7 alone, for the benchmark of the money supply module.
func P17Flows(w *World) { w.p17Flows() }

// SetFlowScale sets the scale of the money flow at index i in ID order, as the
// public call of the integration module will (plan 17).
func SetFlowScale(w *World, i int, bp int64) error { return w.setFlowScale(i, bp, CauseConsumer) }
