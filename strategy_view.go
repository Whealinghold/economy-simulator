package economysim

import (
	"fmt"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file is the world's side of the Strategy contract (spec-agents.md 6.4,
// 6.5.2; plan-phase-1-7.md Phase 7): the scratch the views are built in, the
// core filler of AgentView, and the conversion of an intent to an order. The
// loops over agents, the module groups of a view and the checks of a plan
// against the ledger are the world row's, which calls fillCore first
// (architecture.md 2.10.4). Nothing here logs, emits an event or draws a
// random number.

// viewScratch is the working memory of the filler, sized once from the largest
// role and reused for every agent; it is not saved and not shared between
// worlds. A view built by hand has none and Trade allocates its own.
type viewScratch struct {
	trade tradeScratch

	commodities []CommodityView
	recipes     []RecipeView
	upkeep      []UpkeepRow
	inputs      []RecipeLine
	outputs     []RecipeOutput
	tools       []RecipeTool

	// The copy of the agent's pending directive (D97).
	dirJob      JobRequest
	dirIntents  []Intent
	dirDiscards []Discard
	dirHold     []CommodityID

	// costQ and costP hold the upkeep rows and their prices for the welfare cost.
	costQ, costP []int64

	// plan and outcome are the plan and the outcome view the world passes to
	// Trade and Revise; they are reset, not reallocated, for each call.
	plan    TradePlan
	outcome OutcomeView
}

// newViewScratch sizes the scratch once from the largest role: its tradable
// set, recipes, upkeep (in any region) and the lines of its recipes. It
// converts nothing.
func newViewScratch(cfg *Config, rt *roleTable, rc *recipeTable) *viewScratch {
	var com, rec, up, in, out, tl int
	for i := range rt.rows {
		row := &rt.rows[i]
		com, rec, up = max(com, len(row.tradable)), max(rec, len(row.recipes)), max(up, len(row.upkeep))
		for _, g := range row.regionUpkeep {
			up = max(up, len(g.items))
		}
		var ni, no, nt int
		for _, id := range row.recipes {
			rr := rc.row(id)
			ni, no, nt = ni+len(rr.inputs), no+len(rr.outputs), nt+len(rr.tools)
		}
		in, out, tl = max(in, ni), max(out, no), max(tl, nt)
	}
	return &viewScratch{
		trade:       tradeScratch{rows: make([]tradeRow, 0, com)},
		commodities: make([]CommodityView, 0, com),
		recipes:     make([]RecipeView, 0, rec),
		upkeep:      make([]UpkeepRow, 0, up),
		inputs:      make([]RecipeLine, 0, in),
		outputs:     make([]RecipeOutput, 0, out),
		tools:       make([]RecipeTool, 0, tl),
		costQ:       make([]int64, 0, len(cfg.Commodities)),
		costP:       make([]int64, 0, len(cfg.Commodities)),
		plan:        TradePlan{Intents: make([]Intent, 0, com)},
	}
}

// tradePlan returns the empty, reused plan Trade appends to.
func (sc *viewScratch) tradePlan() *TradePlan {
	sc.plan.Intents, sc.plan.Discards = sc.plan.Intents[:0], sc.plan.Discards[:0]
	return &sc.plan
}

// outcomeView returns the zeroed, reused outcome view with the belief group set;
// the world fills the rest from the book and the window (FR-AGT-049).
func (sc *viewScratch) outcomeView(params BeliefTunables) *OutcomeView {
	sc.outcome = OutcomeView{Params: params}
	return &sc.outcome
}

// viewCtx is what the core filler reads, assembled by the world: pointers to
// the tables it already holds and the per-market state, region-major (index
// region * commodities + commodity). Nothing in it is written by the filler.
type viewCtx struct {
	cfg     *Config
	roles   *roleTable
	recipes *recipeTable
	l       *ledger.Ledger
	// prod gives the maximum executions and the cost anchors of the tick.
	prod   *productionScratch
	yields *yieldTable
	shocks *yieldShocks
	// bp is the belief parameters, beliefParams of the configuration's section.
	bp beliefs.Params
	// moment is the moment of the tick.
	moment Moment
	// reports and windows are the market report and the price window of every
	// (region, commodity).
	reports []marketReport
	windows []beliefs.Window
}

// market returns the report and the price window of a market.
func (c *viewCtx) market(region RegionID, commodity CommodityID) (*marketReport, *beliefs.Window) {
	i := int(region)*len(c.cfg.Commodities) + int(commodity)
	return &c.reports[i], &c.windows[i]
}

// beliefTunablesOf is the belief group of a view: the fifteen parameters that
// beliefs.Params holds, without the root bound MaxPrice (D82).
func beliefTunablesOf(p beliefs.Params) BeliefTunables {
	return BeliefTunables{
		Step: p.Step, AdjustProb: p.AdjustProb, Significance: p.Significance,
		EvidenceShare: p.EvidenceShare, StockLow: p.StockLow, StockHigh: p.StockHigh,
		MaxAge: p.MaxAge, WidthInit: p.WidthInit, WidthMin: p.WidthMin,
		WidthMax: p.WidthMax, WidthStep: p.WidthStep, CostFloor: p.CostFloor,
		CostCeiling: p.CostCeiling, RefBandLow: p.RefBandLow, RefBandHigh: p.RefBandHigh,
	}
}

// tunablesOf is the Tunables group of a view (spec-agents.md 12).
func (c *viewCtx) tunablesOf() Tunables {
	w := &c.cfg.World
	return Tunables{
		CoverTicks: w.CoverTicks, InputJobs: w.InputJobs, OutputCoverTicks: w.OutputCoverTicks,
		FavourFloor: w.FavourFloor, ReserveTicks: w.ReserveTicks, SpendMaxShare: w.SpendMaxShare,
		SpendHalfTicks: w.SpendHalfTicks, RestBand: w.RestBand, ProduceHaltRatio: w.ProduceHaltRatio,
		Beliefs: beliefTunablesOf(c.bp),
	}
}

// fillCore fills the core groups of v for agent a (spec-agents.md 6.4.1): the
// identity, money, state, upkeep, job, per-commodity, per-recipe, tunables and
// mailbox groups. Every slice is a scratch copy with its capacity capped, so a
// strategy that writes or appends to a view changes no state and no later view
// (FR-AGT-024). The cost is proportional to the tradable set plus the recipes
// and does not depend on the weight (NFR-AGT-002). An error is a ledger refusal
// or a yield that does not fit, which a validated world does not produce; the
// world fails the step on it. The module fillers run after this one.
func (sc *viewScratch) fillCore(v *AgentView, a *agent, c *viewCtx) error {
	row := c.roles.row(a.role)
	acct, err := c.l.Account(a.account)
	if err != nil {
		return fmt.Errorf("fill view of agent %d: account %d: %w", a.id, a.account, err)
	}
	region := RegionID(acct.Region)
	role := &c.cfg.Roles[a.role]
	*v = AgentView{
		Tick: c.moment, Agent: a.id, Role: a.role, Region: region, Weight: a.weight, External: a.external,
		Money: acct.Balance - acct.Reserved, Hunger: a.hunger, Unfed: a.unfed, TargetRatio: a.target,
		WorkingTicks: a.history.WorkingTicks(), Window: row.window, StartsIn: 1, MinWorkShare: role.MinWorkShare,
		Tunables: c.tunablesOf(), scratch: sc,
	}
	if a.hasJob() {
		v.HasJob, v.Recipe, v.Remaining, v.Executions = true, a.job.recipe, a.job.remaining, a.job.executions
		v.StartsIn = a.job.remaining + 1
	}
	sc.fillUpkeep(v, row.upkeepIn(region), role.DefaultRatio, a, c, region)
	if err := sc.fillRecipes(v, row, a, c, region); err != nil {
		return err
	}
	if err := sc.fillCommodities(v, row, a, c, region); err != nil {
		return err
	}
	sc.fillDirective(v, a)
	return nil
}

// fillUpkeep sets the upkeep group and the welfare ratio, which is read at the
// published prices of the region: the cost of one tick of the upkeep of the
// agent's weight (spec-welfare-and-wages.md 2.1).
func (sc *viewScratch) fillUpkeep(v *AgentView, per []itemRow, defaultBP int64, a *agent, c *viewCtx, region RegionID) {
	sc.upkeep, sc.costQ, sc.costP = sc.upkeep[:0], sc.costQ[:0], sc.costP[:0]
	for _, it := range per {
		rep, _ := c.market(region, it.commodity)
		sc.upkeep = append(sc.upkeep, UpkeepRow{Commodity: it.commodity, Quantity: int64(it.quantity)})
		sc.costQ, sc.costP = append(sc.costQ, int64(it.quantity)), append(sc.costP, int64(rep.Price))
	}
	v.Upkeep = sc.upkeep[:len(sc.upkeep):len(sc.upkeep)]
	v.Ratio, v.HasRatio = a.history.Ratio(welfare.TickCost(sc.costQ, sc.costP, a.weight), defaultBP)
}

// fillRecipes sets the per-recipe group: the lines of each recipe of the role in
// ascending RecipeID, the effective yield of the region and the season, and the
// executions the agent could start now with the reason if there are none, read
// by the same count the start uses (spec-production.md 4.2).
func (sc *viewScratch) fillRecipes(v *AgentView, row *roleRow, a *agent, c *viewCtx, region RegionID) error {
	sc.recipes, sc.inputs, sc.outputs, sc.tools = sc.recipes[:0], sc.inputs[:0], sc.outputs[:0], sc.tools[:0]
	for _, id := range row.recipes {
		rr, cr := c.recipes.row(id), &c.cfg.Recipes[id]
		i0, o0, t0 := len(sc.inputs), len(sc.outputs), len(sc.tools)
		for _, in := range rr.inputs {
			sc.inputs = append(sc.inputs, RecipeLine{Commodity: in.commodity, Quantity: int64(in.quantity)})
		}
		for _, o := range rr.outputs {
			sc.outputs = append(sc.outputs, RecipeOutput{Commodity: o.commodity, Quantity: int64(o.quantity), CostShare: o.costShare, Discard: o.discard})
		}
		for _, t := range rr.tools {
			sc.tools = append(sc.tools, RecipeTool{Commodity: t.commodity, WearPerUse: t.wearPerUse, Durability: c.cfg.Commodities[t.commodity].Durability})
		}
		// Untested: the yield, shock and profile bounds of validation keep the
		// effective yield in range, and an unknown account is refused in fillCore
		// before this loop, so neither error below is reachable from a built world.
		e, has, ok := c.yields.effective(c.shocks, region, id, c.moment.Season, cr.Sited)
		if !ok {
			return fmt.Errorf("fill view of agent %d: effective yield of recipe %d does not fit", a.id, id)
		}
		k0, reason, err := c.prod.maxExecutions(c.l, &startReq{
			recipe: id, account: a.account, principal: a.id, weight: a.weight, effective: e, hasProfile: has,
		})
		if err != nil {
			return fmt.Errorf("fill view of agent %d: executions of recipe %d: %w", a.id, id, err)
		}
		sc.recipes = append(sc.recipes, RecipeView{
			Recipe: id, Duration: cr.Duration, Effective: e, MaxExecutions: k0, Reason: reason,
			Inputs:  sc.inputs[i0:len(sc.inputs):len(sc.inputs)],
			Outputs: sc.outputs[o0:len(sc.outputs):len(sc.outputs)],
			Tools:   sc.tools[t0:len(sc.tools):len(sc.tools)],
		})
	}
	v.Recipes = sc.recipes[:len(sc.recipes):len(sc.recipes)]
	return nil
}

// makesIn reports whether a recipe of the view makes c as a non-discard output.
func makesIn(recipes []RecipeView, c CommodityID) bool {
	for i := range recipes {
		for _, o := range recipes[i].Outputs {
			if o.Commodity == c && !o.Discard {
				return true
			}
		}
	}
	return false
}

// fillCommodities sets the per-commodity group over the tradable set in
// canonical order. UnitCost is the cost anchor of the region only for a
// commodity the role makes, and 0 otherwise and while the region has no
// observed wage, when the anchor table is 0 (FR-AGT-048, spec-agents.md 7.3.6).
// The bounds use that same UnitCost (spec-beliefs.md 3.2.4).
func (sc *viewScratch) fillCommodities(v *AgentView, row *roleRow, a *agent, c *viewCtx, region RegionID) error {
	if len(a.beliefs) != len(row.tradable) {
		return fmt.Errorf("fill view of agent %d: %d beliefs for %d tradable commodities", a.id, len(a.beliefs), len(row.tradable))
	}
	sc.commodities = sc.commodities[:0]
	for i, id := range row.tradable {
		held, _, err := c.l.Holding(a.account, int(id))
		if err != nil {
			return fmt.Errorf("fill view of agent %d: stock of commodity %d: %w", a.id, id, err)
		}
		// Untested: Holding above already refused an unknown account or commodity,
		// which is all Expiring can refuse.
		expiring, err := c.l.Expiring(a.account, int(id), c.moment.Tick)
		if err != nil {
			return fmt.Errorf("fill view of agent %d: expiring units of commodity %d: %w", a.id, id, err)
		}
		rep, win := c.market(region, id)
		ref := c.cfg.refPrice(region, id)
		var unitCost int64
		if makesIn(v.Recipes, id) {
			unitCost = c.prod.unitCost(region, id)
		}
		b, bd := a.beliefs[i], beliefs.Limits(int64(ref), unitCost, c.bp)
		cv := CommodityView{
			Commodity: id, Stock: held, Expiring: expiring, Ref: ref, Tool: c.recipes.tool(id), UnitCost: Price(unitCost),
			Belief: Belief{Centre: Price(b.Centre), Width: b.Width},
			Bounds: Bounds{Floor: Price(bd.Floor), Ceiling: Price(bd.Ceiling)},
			Market: MarketReport{
				Price: int64(rep.Price), Age: rep.Age, BestBid: int64(rep.BestBid), BestAsk: int64(rep.BestAsk),
				HasBestBid: rep.HasBestBid, HasBestAsk: rep.HasBestAsk,
			},
		}
		if mean, ok := win.Mean(); ok {
			lo, _ := win.Low()
			hi, _ := win.High()
			cv.Window = PriceWindowView{Mean: mean, Low: lo, High: hi, Position: beliefs.Position(int64(rep.Price), lo, hi), HasWindow: true}
		}
		sc.commodities = append(sc.commodities, cv)
	}
	v.Commodities = sc.commodities[:len(sc.commodities):len(sc.commodities)]
	return nil
}

// fillDirective copies the pending directive into the view by value, its slices
// and its job into the scratch, so that nothing in the view points into the
// agent (D97, spec-agents.md 6.4.2).
func (sc *viewScratch) fillDirective(v *AgentView, a *agent) {
	d := a.directive
	if d == nil {
		return
	}
	sc.dirIntents = append(sc.dirIntents[:0], d.Intents...)
	sc.dirDiscards = append(sc.dirDiscards[:0], d.Discards...)
	sc.dirHold = append(sc.dirHold[:0], d.Hold...)
	v.HasDirective = true
	v.Directive = Directive{
		Intents:  sc.dirIntents[:len(sc.dirIntents):len(sc.dirIntents)],
		Discards: sc.dirDiscards[:len(sc.dirDiscards):len(sc.dirDiscards)],
		Hold:     sc.dirHold[:len(sc.dirHold):len(sc.dirHold)],
		Sticky:   d.Sticky,
	}
	if d.Job != nil {
		sc.dirJob = *d.Job
		v.Directive.Job = &sc.dirJob
	}
}

// normalise applies the clamps of spec-agents.md 6.5.2 to an intent in place and
// reports whether it changed anything: a Target, Reserve or UnitCost below 0 or
// a Target or Reserve above MaxQuantity is read as 0, and a Purpose outside the
// enumeration as PurposeOther. They are inputs to a belief rule, not orders, so
// they never reject the order; the world logs a Debug line when it changed. A
// Target from the output target above MaxQuantity is cut to 0 here as well.
func normalise(in *Intent) (changed bool) {
	if in.Target < 0 || in.Target > MaxQuantity {
		in.Target, changed = 0, true
	}
	if in.Reserve < 0 || in.Reserve > MaxQuantity {
		in.Reserve, changed = 0, true
	}
	if in.UnitCost < 0 {
		in.UnitCost, changed = 0, true
	}
	if in.Purpose < PurposeOther || in.Purpose > PurposeTool {
		in.Purpose, changed = PurposeOther, true
	}
	return changed
}

// orderOf turns an intent of agent a into the order the market admits: the
// intent's Limit is the order's Price and its Quantity the order's. The
// admission checks (not_tradable, duplicate_order and the market's) are the
// book's, not repeated here (FR-AGT-025, 026).
func orderOf(a *agent, in Intent) Order {
	return Order{Agent: a.id, Commodity: in.Commodity, Side: in.Side, Price: in.Limit, Quantity: in.Quantity}
}

// outcomeStock is the Stock of an OutcomeView: the units held after the trades
// less those reserved for this tick's upkeep, not below 0 (FR-AGT-049). reserve
// is the intent's Reserve after normalise, so it is not negative.
func outcomeStock(held, reserve int64) int64 { return max(0, held-reserve) }
