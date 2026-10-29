package economysim

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is phases 5 and 6 of the tick (spec-world.md 6.7, 6.8;
// plan-phase-1-11.md Phase 6): the price window and the estates' markdown
// (P5.1, P5.3), the outcomes and the belief update (P5.4), the upkeep (P6.1),
// the welfare push (P6.3) and the shortfall (P6.4). The
// positions of the phases that belong to modules (the information record, wants)
// are not built.

// The values of the bound key of the Warn "price beliefs at bound": the closed
// set of that one line (spec-beliefs.md 12.2).
const (
	boundFloor   = "floor"
	boundCeiling = "ceiling"
)

// upkeepCell is what P6.1 counts for one (region, role) for P6.4: every
// household of the cell, the households that went unfed, the agents with an
// unfed household and the largest hunger. The Warn reads them, so they are
// counted whether or not an observer was supplied (D159).
type upkeepCell struct {
	households, unfed, agents, maxHunger int64
}

// outcomeTick is the scratch of phases 5 and 6, one per world: the tally of each
// market, the price of every (region, commodity) at which the welfare record
// values a tick, the scratch of the consumption step and the counts of the
// upkeep per (region, role) with, under the flag, the units short of each
// commodity. It is derived or scratch and never saved.
type outcomeTick struct {
	tallies []beliefs.Tally
	valued  []int64
	eat     consumeScratch
	cells   []upkeepCell
	short   []int64
	// clamped is, per (region, role), the households whose ring holds a field at
	// the clamp, counted at P6.3.
	clamped []int64
}

func newOutcomeTick(c *Config) outcomeTick {
	markets, cells := len(c.Regions)*len(c.Commodities), len(c.Regions)*len(c.Roles)
	return outcomeTick{
		tallies: make([]beliefs.Tally, markets),
		valued:  make([]int64, markets),
		cells:   make([]upkeepCell, cells),
		short:   make([]int64, cells*len(c.Commodities)),
		clamped: make([]int64, cells),
	}
}

// p51Window is P5.1: the published price of every market in which units traded
// in the tick goes into its window (spec-beliefs.md 7.4, FR-CLR-032).
func (w *World) p51Window() { w.markets.pushTraded() }

// p53Markdown is P5.3: for each region in ascending order, the markdown of the
// estate's listings (spec-world.md 6.7, C6).
func (w *World) p53Markdown() error {
	for r := range w.cfg.Regions {
		if err := w.estates.markdown(w.ledger, &w.cfg, RegionID(r)); err != nil { // excluded from coverage (testing.md 9.5): markdown returns a nil error today
			return fmt.Errorf("estate markdown at tick %d: %w", w.Tick(), err)
		}
	}
	return nil
}

// p54Outcomes is P5.4: the outcome of every admitted order of a household, in
// ascending (RegionID, CommodityID) and admission order, delivered once, after
// every market of the tick has cleared (spec-world.md 6.7, FR-WLD-049). After
// the last outcome of a market with one the aggregate BeliefsUpdated is emitted
// under the flag, and the run of the Warn "price beliefs at bound" of every
// market is advanced, also for a market with no outcome (N-230).
func (w *World) p54Outcomes() {
	for i := range w.markets.books {
		w.marketOutcomes(i)
	}
}

// marketOutcomes is the P5.4 step of market i. An estate's ask has no outcome
// and takes no word (ruling C21).
func (w *World) marketOutcomes(i int) {
	b, t := w.markets.books[i], &w.oc.tallies[i]
	*t = beliefs.Tally{}
	region, commodity := RegionID(i/w.markets.commodities), CommodityID(i%w.markets.commodities)
	fill, pseudo := b.filled(), AgentID(len(w.cfg.Regions))
	for j := range b.entries {
		if b.entries[j].agent < pseudo {
			continue
		}
		w.revise(i, j, commodity, fill[j], t)
	}
	if t.Outcomes > 0 && w.obs.supplied {
		w.obs.deliver(beliefsEvent(w.Tick(), region, commodity, t))
	}
	w.boundWarn(i, region, commodity, t)
}

// revise is the outcome of the order at index j of the book of market i: the
// stream of the order is keyed by (tick, agent, commodity) and its first word is
// the outcome's Draw (D8), the view is built from the book, the record of the
// intent and the market's report, Revise returns a belief, the world applies it
// inside its bounds and adds the change to the tally of the market.
func (w *World) revise(i, j int, commodity CommodityID, filled int64, t *beliefs.Tally) {
	e := &w.markets.books[i].entries[j]
	a := &w.agents[e.agent]
	rec := w.markets.intents[i][j]
	ci, ok := slices.BinarySearch(w.roles.row(a.role).tradable, commodity)
	if !ok {
		invariant("an admitted order is for a commodity outside the tradable set of its agent")
	}
	held, _, err := w.ledger.Holding(a.account, int(commodity))
	if err != nil {
		invariant("the ledger refused the holding of a live agent: " + err.Error())
	}
	bd := beliefs.Limits(int64(w.refs[i]), int64(rec.unitCost), w.bp)
	before := a.beliefs[ci]
	w.rnd.keyedPair(siteOutcome, w.Tick(), int64(a.id), int64(commodity))
	draw := w.rnd.word()
	o := w.views.outcomeView(beliefTunablesOf(w.bp))
	*o = OutcomeView{
		Agent: a.id, Commodity: commodity, Side: e.side, Limit: e.price, Quantity: e.qty, Filled: Quantity(filled),
		Belief: Belief{Centre: Price(before.Centre), Width: before.Width},
		Bounds: Bounds{Floor: Price(bd.Floor), Ceiling: Price(bd.Ceiling)},
		Market: reportView(&w.markets.reports[i]), Stock: outcomeStock(held, int64(rec.reserve)),
		Target: rec.target, UnitCost: rec.unitCost, Draw: draw, Params: o.Params,
	}
	got := w.ext.strategies[a.strategy].Revise(o, w.rnd.rnd)
	after, clamped := beliefs.Clamp(beliefs.Belief{Centre: int64(got.Centre), Width: got.Width}, bd, w.bp)
	a.beliefs[ci] = after
	c := beliefs.Change{Before: before, After: after, Bounds: bd, Open: beliefs.Gate(draw, w.bp.AdjustProb), Clamped: clamped}
	t.Add(c)
	w.logBelief(a.id, RegionID(i/w.markets.commodities), commodity, c)
}

// reportView is the market report of an OutcomeView: the published fields of a
// marketReport.
func reportView(r *marketReport) MarketReport {
	return MarketReport{
		Price: int64(r.Price), Age: r.Age, BestBid: int64(r.BestBid), BestAsk: int64(r.BestAsk),
		HasBestBid: r.HasBestBid, HasBestAsk: r.HasBestAsk,
	}
}

// logBelief writes the Debug line "belief updated" (spec-beliefs.md 12.2).
func (w *World) logBelief(agent AgentID, region RegionID, commodity CommodityID, c beliefs.Change) {
	ctx := context.Background()
	if !w.log.Enabled(ctx, slog.LevelDebug) {
		return
	}
	w.log.LogAttrs(ctx, slog.LevelDebug, "belief updated",
		slog.Int64(logkeys.Region, int64(region)), slog.Int64(logkeys.Commodity, int64(commodity)),
		slog.Int64(logkeys.Agent, int64(agent)), slog.Int64(logkeys.Previous, c.Before.Centre),
		slog.Int64(logkeys.Price, c.After.Centre), slog.String(logkeys.Reason, string(beliefReason(c))))
}

// boundWarn advances the run of the Warn "price beliefs at bound" of market i
// from the tally of the tick and logs it at the tenth tick of a run and every
// tenth after (spec-beliefs.md 12.2, D156). The run is a diagnostic.
func (w *World) boundWarn(i int, region RegionID, commodity CommodityID, t *beliefs.Tally) {
	r := beliefs.NextRun(w.markets.boundRun[i], t, warnRun)
	w.markets.boundRun[i] = r.Count
	ctx := context.Background()
	if !r.Warn || !w.log.Enabled(ctx, slog.LevelWarn) {
		return
	}
	bound := boundFloor
	if r.Ceiling {
		bound = boundCeiling
	}
	w.log.LogAttrs(ctx, slog.LevelWarn, "price beliefs at bound",
		slog.Int64(logkeys.Region, int64(region)), slog.Int64(logkeys.Commodity, int64(commodity)),
		slog.String(logkeys.Bound, bound), slog.Int64(logkeys.Agents, r.Agents), slog.Int64(logkeys.Outcomes, r.Outcomes))
}

// p61Upkeep is P6.1: for each household agent in ascending ID the consumption
// step moves what it holds of each upkeep commodity, up to the upkeep, to the
// consumption boundary and sets Unfed, Hunger and Privation (spec-agents.md 5.3,
// 5.4.1, ruling C2). The counts of P6.4 are taken from the same pass. A shortfall
// is not an error; the error is a ledger refusal, which fails the step.
func (w *World) p61Upkeep() error {
	oc := &w.oc
	clear(oc.cells)
	if w.obs.supplied {
		clear(oc.short)
	}
	roles := len(w.cfg.Roles)
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		a := &w.agents[i]
		if a.weight < 1 {
			continue
		}
		g := w.regionOf(a)
		// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
		// refusal of the world's own accounts, which a valid configuration cannot cause
		// and no test can inject from the root (D190); the branch only wraps and returns
		// it.
		if err := consume(w.ledger, a, a.account, w.roles.row(a.role).upkeepIn(g), &oc.eat); err != nil {
			return fmt.Errorf("consumption at tick %d: %w", w.Tick(), err)
		}
		w.countUpkeep(a, int(g)*roles+int(a.role))
	}
	return nil
}

// countUpkeep adds the agent's households to the counts of its cell after the
// consumption step, and logs the Debug line "upkeep unmet" for each commodity it
// was short of. The units short of each commodity are summed only under the flag.
func (w *World) countUpkeep(a *agent, cell int) {
	c := &w.oc.cells[cell]
	c.households += a.weight
	c.maxHunger = max(c.maxHunger, a.hunger)
	if a.unfed == 0 {
		return
	}
	c.unfed += a.unfed
	c.agents++
	ctx := context.Background()
	debug := w.log.Enabled(ctx, slog.LevelDebug)
	for k, short := range w.oc.eat.short {
		if short < 1 {
			continue
		}
		commodity := w.oc.eat.totals[k].commodity
		if w.obs.supplied {
			w.oc.short[cell*len(w.cfg.Commodities)+int(commodity)] += short
		}
		if debug {
			w.log.LogAttrs(ctx, slog.LevelDebug, "upkeep unmet",
				slog.Int64(logkeys.Agent, int64(a.id)), slog.Int64(logkeys.Commodity, int64(commodity)),
				slog.Int64(logkeys.Quantity, short))
		}
	}
}

// p64Shortfall is P6.4: for each (region, role) in ascending order, the run of
// the Warn "households unfed" is advanced from the counts of P6.1 and, under the
// flag, UpkeepShortfall is emitted where a household went unfed (spec-agents.md
// 14.1, FR-OBS-009).
func (w *World) p64Shortfall() {
	ctx := context.Background()
	roles := len(w.cfg.Roles)
	for cell := range w.oc.cells {
		c := &w.oc.cells[cell]
		run, warn := advanceRun(w.state.unfedRun[cell], c.unfed > 0, false)
		w.state.unfedRun[cell] = run
		region, role := RegionID(cell/roles), RoleID(cell%roles)
		if warn && w.log.Enabled(ctx, slog.LevelWarn) {
			w.log.LogAttrs(ctx, slog.LevelWarn, "households unfed",
				slog.Int64(logkeys.Region, int64(region)), slog.Int64(logkeys.Role, int64(role)),
				slog.Int64(logkeys.Households, c.households), slog.Int64(logkeys.Unfed, c.unfed))
		}
		if c.unfed > 0 && w.obs.supplied {
			w.obs.deliver(UpkeepShortfall{
				Tick: w.Tick(), Region: region, Role: role, Households: c.households, Unfed: c.unfed,
				Agents: c.agents, MaxHunger: c.maxHunger, Short: w.shortRows(cell),
			})
		}
	}
}

// shortRows is the Short of an UpkeepShortfall of a cell: a row for each
// commodity short, in ascending CommodityID. The event owns the slice.
func (w *World) shortRows(cell int) []ShortRow {
	nc := len(w.cfg.Commodities)
	var rows []ShortRow
	for k, units := range w.oc.short[cell*nc : (cell+1)*nc] {
		if units > 0 {
			rows = append(rows, ShortRow{Commodity: CommodityID(k), Units: units})
		}
	}
	return rows
}

// p63Welfare is P6.3: the welfare record of the tick of each household agent in
// ascending ID (spec-welfare-and-wages.md 4.3, 9, D158): the working tick from
// the scratch of the tick, the record valued at the prices of the tick, Push into
// the agent's ring and the reset of its scratch. The run of the Warn "welfare
// record clamped" of every (region, role) is advanced from the households whose
// ring holds a field at the clamp (pattern A, FR-WEL-042).
func (w *World) p63Welfare() {
	w.valuePrices()
	if w.sub != nil {
		w.sub.priceReceipts(w)
	}
	clear(w.oc.clamped)
	roles, nc := len(w.cfg.Roles), len(w.cfg.Commodities)
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		a := &w.agents[i]
		if a.weight < 1 {
			continue
		}
		g, row := w.regionOf(a), w.roles.row(a.role)
		k, ownUse, traded := w.welfare.workRead(i)
		working, workHH, ownHH := workingTick(len(row.recipes) > 0, k, ownUse, traded, false, a.weight)
		rec := w.welfare.record(i, a.weight, working, workHH, ownHH, w.oc.valued[int(g)*nc:(int(g)+1)*nc], row.upkeepIn(g))
		a.history.Push(rec)
		if a.history.AtBound() {
			w.oc.clamped[int(g)*roles+int(a.role)] += a.weight
		}
		w.welfare.reset(i)
	}
	w.clampWarns()
}

// valuePrices fills the price at which the welfare record values a tick for every
// (region, commodity): the published price of the market, or the effective
// reference price while its window is empty, that is while it has never traded
// (spec-welfare-and-wages.md 2.1, FR-WEL-003). The window is pushed at P5.1 in
// every tick with a trade and never shrinks, so Len() == 0 says never traded.
func (w *World) valuePrices() {
	for i := range w.oc.valued {
		if w.markets.windows[i].Len() > 0 {
			w.oc.valued[i] = int64(w.markets.reports[i].Price)
			continue
		}
		w.oc.valued[i] = int64(w.refs[i])
	}
}

// clampWarns advances the run of the Warn "welfare record clamped" of every
// (region, role) from the clamped households counted in P6.3 and logs it on the
// first tick of a run and every tenth after (D156).
func (w *World) clampWarns() {
	ctx := context.Background()
	roles := len(w.cfg.Roles)
	for cell, n := range w.oc.clamped {
		run, warn := clampRun(w.state.clampRun[cell], n)
		w.state.clampRun[cell] = run
		if warn && w.log.Enabled(ctx, slog.LevelWarn) {
			w.log.LogAttrs(ctx, slog.LevelWarn, "welfare record clamped",
				slog.Int64(logkeys.Region, int64(cell/roles)), slog.Int64(logkeys.Role, int64(cell%roles)),
				slog.Int64(logkeys.Households, n))
		}
	}
}
