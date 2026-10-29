package economysim

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is phase 2 of the tick (spec-world.md 6.4, spec-production.md 5;
// plan-phase-1-11.md Phase 4): the continuations and completions (P2.1), the
// calls of Work (P2.5), the starts (P2.7) and the aggregates of the production
// halt (P2.8a). Work is two-pass (spec-world.md 6.4.1): every call of P2.5 comes
// before any start of P2.7. The positions of phase 2 that belong to modules
// (teaching, the endowments, own use, the labour round) are not built.

// workCell is the count of the free households of one (region, role) in one
// tick. households is every household of the role in the region, free is the
// part passed to Work at P2.5, floorHeld the part the work floor held to work
// (spec-welfare-and-wages.md 11.3). The rest are the households of the
// aggregate ProductionHalted and are counted only when an observer was supplied
// (observability-and-logging.md 4.5).
type workCell struct {
	households, free, floorHeld, halted, rested, noRecipe int64
}

// jobAsk is the job request of one agent that P2.5 admitted and P2.7 starts.
type jobAsk struct {
	recipe  RecipeID
	maximum int64
	set     bool
	// first and draws locate the records of the request on the endowments its
	// recipe draws on (resources.go): draws is 0 for a recipe with no draw or while
	// the module is off.
	first, draws int
}

// productionTick is the scratch of phase 2, one per world: the rotation lists
// of the tick, the requests, the counts per (region, role) and, per recipe, the
// households the halt skipped, the work report and the tick of the last
// "recipe output disposed" line of each recipe (stamped tick + 1). It is
// derived or scratch and never saved.
type productionTick struct {
	rotation [][]int
	asks     []jobAsk
	cells    []workCell
	skips    []int64
	report   workReportBuf
	ovfStamp []int64
}

func newProductionTick(c *Config) productionTick {
	return productionTick{
		rotation: make([][]int, len(c.Regions)),
		cells:    make([]workCell, len(c.Regions)*len(c.Roles)),
		skips:    make([]int64, len(c.Regions)*len(c.Roles)*len(c.Recipes)),
		ovfStamp: make([]int64, len(c.Recipes)),
	}
}

// rotationAt returns the index into a region's agent list of the k-th visit of
// a walk that starts at the position tick mod n and wraps (spec-world.md 6.1.4,
// FR-WLD-043). It is the one rotation of the library. n must be at least 1.
func rotationAt(n int, tick int64, k int) int {
	return int((tick%int64(n) + int64(k)) % int64(n))
}

// buildRotation lists, per region, the household agents whose account is in the
// region, in ascending agent ID, once at the start of phase 2 (spec-world.md
// 6.1.4, D153), and counts the households of each (region, role). A pseudo-agent
// is not a household. The lists and the request table keep their capacity.
func (w *World) buildRotation() {
	p := &w.pr
	for g := range p.rotation {
		p.rotation[g] = p.rotation[g][:0]
	}
	clear(p.cells)
	for len(p.asks) < len(w.agents) {
		p.asks = append(p.asks, jobAsk{})
	}
	roles := len(w.cfg.Roles)
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		a := &w.agents[i]
		if a.weight < 1 {
			continue
		}
		g := w.regionOf(a)
		p.rotation[g] = append(p.rotation[g], i)
		p.cells[int(g)*roles+int(a.role)].households += a.weight
	}
}

// walk calls f for each agent of region g in the rotation order of the tick.
func (w *World) walk(g int, f func(a *agent) error) error {
	list, tick := w.pr.rotation[g], w.Tick()
	for k := range list {
		// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
		// refusal of the world's own accounts, which a valid configuration cannot cause
		// and no test can inject from the root (D190); the branch only wraps and returns
		// it.
		if err := f(&w.agents[list[rotationAt(len(list), tick, k)]]); err != nil {
			return err
		}
	}
	return nil
}

// p21Complete is P2.1: the continuations and completions of every job, region
// by region in the rotation order (spec-production.md 5.2, 5.3). The rotation of
// the tick is listed here, at the start of phase 2. A job continues (its
// remaining time falls by 1) and, at 0, completes: the pending output is created
// by the ledger, bounded by MaxQuantity, and RecipeExecuted is delivered. The
// agent is marked as working in this tick, once (spec-welfare-and-wages.md 4.3).
func (w *World) p21Complete() {
	w.buildRotation()
	tick := w.Tick()
	for g, list := range w.pr.rotation {
		for k := range list {
			a := &w.agents[list[rotationAt(len(list), tick, k)]]
			if !a.hasJob() {
				continue
			}
			done, working := a.job.advance()
			w.welfare.markWorking(int(a.id), working, false)
			if done {
				w.finishJob(a, RegionID(g))
			}
		}
	}
}

// finishJob completes the job of a that has run out of time: the ledger creates
// the output, RecipeExecuted is delivered under the flag and the overflow Warn
// is considered.
func (w *World) finishJob(a *agent, region RegionID) {
	c := w.prod.complete(w.ledger, &a.job, a.account, MaxQuantity)
	if w.obs.supplied {
		w.obs.deliver(w.prod.recipeExecuted(nil, nil, nil, w.Tick(), a.id, region, &c, w.cfg.Recipes[c.recipe].Duration))
	}
	w.warnOverflow(&c)
}

// warnOverflow logs the Warn "recipe output disposed" (spec-production.md
// 14.2): the part of an output that no room held. It has no run and is logged
// at most once per recipe per tick, for the first output that overflowed in the
// first completion of the recipe in the tick, so its volume is bounded by the
// recipes of the configuration.
func (w *World) warnOverflow(c *completion) {
	ctx := context.Background()
	stamp := w.Tick() + 1
	if !w.log.Enabled(ctx, slog.LevelWarn) || w.pr.ovfStamp[c.recipe] == stamp {
		return
	}
	for i, q := range c.overflow {
		if q > 0 {
			w.pr.ovfStamp[c.recipe] = stamp
			w.log.LogAttrs(ctx, slog.LevelWarn, "recipe output disposed",
				slog.Int64(logkeys.Recipe, int64(c.recipe)),
				slog.Int64(logkeys.Commodity, int64(w.recipes.row(c.recipe).outputs[i].commodity)),
				slog.Int64(logkeys.Quantity, q))
			return
		}
	}
}

// workDraws reports whether a call of Work on s takes random words: the default
// strategy and a Directed that falls back to it take none, so the world does not
// reseed the stream for them (spec-world.md 7.3.3, D155).
func workDraws(s Strategy) bool {
	switch s := s.(type) {
	case DefaultStrategy:
		return false
	case Directed:
		return s.Fallback != nil && workDraws(s.Fallback)
	}
	return true
}

// p25Work is P2.5: Work for every free agent, region by region in the rotation
// order (spec-world.md 6.4, 6.4.1). An agent is free when it has no job in
// progress and none was continued or completed in P2.1. The view is built from
// the state after P2.1 by the one builder of the world (fillAgentView), and the
// work report, which the counts read, is taken from the view as filled and before
// the strategy has it in hand, so a strategy cannot hide the floor, the halt or
// the rest. The request is admitted here: a recipe off the role's list or one
// that cannot start is refused and counted, and the rest wait for P2.7. The runs
// of the Warn "work floor binding" advance for every cell.
func (w *World) p25Work() error {
	w.refreshViewCtx()
	for g := range w.pr.rotation {
		// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
		// refusal of the world's own accounts, which a valid configuration cannot cause
		// and no test can inject from the root (D190); the branch only wraps and returns
		// it.
		if err := w.walk(g, func(a *agent) error { return w.workAgent(a, RegionID(g)) }); err != nil {
			return err
		}
	}
	w.floorWarns()
	return nil
}

// workAgent is the P2.5 step of one agent: the view, the report and its counts,
// the call of Work and the admission of the request.
func (w *World) workAgent(a *agent, region RegionID) error {
	if a.hasJob() || w.welfare.at(int(a.id)).jobSet {
		return nil
	}
	v := &w.tk.view
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err := w.fillAgentView(v, a); err != nil {
		return fmt.Errorf("work at tick %d: %w", w.Tick(), err)
	}
	cell := int(region)*len(w.cfg.Roles) + int(a.role)
	w.countWork(cell, region, a, v)
	s := w.ext.strategies[a.strategy]
	if workDraws(s) {
		w.rnd.keyed(siteWork, w.Tick(), int64(a.id))
	}
	if req := s.Work(v, w.rnd.rnd); !req.NoJob {
		w.admitJob(a, region, req)
	}
	return nil
}

// countWork takes the work report of the view as filled and adds the agent's
// households to the counts of its cell: the free and floor-held households
// always, because the Warn reads them, and the halt, rest, no-recipe and
// per-recipe skip counts and the blocked refusals only under the flag.
func (w *World) countWork(cell int, region RegionID, a *agent, v *AgentView) {
	r := &w.pr.report
	workReport(v, r)
	c := &w.pr.cells[cell]
	c.free += a.weight
	if w.sub != nil {
		w.sub.countWork(cell, a.weight, r)
	}
	if r.floorChanged {
		c.floorHeld += a.weight
	}
	if !w.obs.supplied {
		return
	}
	switch {
	case r.rest:
		c.rested += a.weight
	case r.reason == ReasonHalted:
		c.halted += a.weight
	case r.blocked >= 0:
		w.refuse(a, region, v.Recipes[r.blocked].Recipe, r.reason)
	case r.reason == ReasonNoRecipe:
		c.noRecipe += a.weight
	}
	for _, id := range r.skipped {
		w.pr.skips[cell*len(w.cfg.Recipes)+int(id)] += a.weight
	}
}

// admitJob is the world's check of a request (spec-agents.md 6.5.1,
// spec-production.md 4.2): the recipe is checked against the role's list first
// and refused not_on_role, then the executions k0 the request could start are
// derived and a count of 0 is refused with the first applicable reason. An
// admitted request is kept for P2.7, and its draws on the endowments are recorded
// for the ration of P2.6, which ignores them here (spec-resources.md 5.2).
func (w *World) admitJob(a *agent, region RegionID, req JobRequest) {
	row := w.roles.row(a.role)
	if reason := admit(row.jobs, req.Recipe); reason != "" {
		w.refuse(a, region, req.Recipe, reason)
		return
	}
	q := w.startReq(a, region, req.Recipe, req.Executions)
	k0, reason, err := w.prod.maxExecutions(w.ledger, &q)
	if err != nil {
		// Excluded from coverage (testing.md 9.5): the account and the commodities
		// of a request on the role's list are the world's own, so the ledger
		// cannot refuse the read.
		invariant("the ledger refused the count of a request: " + err.Error())
	}
	if k0 < 1 {
		w.refuse(a, region, req.Recipe, reason)
		return
	}
	ask := jobAsk{recipe: req.Recipe, maximum: req.Executions, set: true}
	ask.first, ask.draws = w.recordDraws(a, region, req.Recipe, k0)
	w.pr.asks[a.id] = ask
}

// startReq is the request to start a recipe for agent a in a region at the
// moment of the tick: the effective factor of the region and the season, with the
// abundance of the snapshot for a recipe that draws (spec-production.md 6.3), and
// the weight of the agent.
func (w *World) startReq(a *agent, region RegionID, recipe RecipeID, maximum int64) startReq {
	e, has, ok := w.yields.effectiveAt(&w.shocks, region, recipe, w.tk.moment.Season, w.cfg.Recipes[recipe].Sited,
		w.prod.abundance(region, recipe, false))
	if !ok {
		// Excluded from coverage (testing.md 9.5): the bounds of validation keep
		// the effective yield in range.
		invariant("the effective yield does not fit after validation")
	}
	return startReq{recipe: recipe, account: a.account, principal: a.id, weight: a.weight,
		maximum: maximum, effective: e, hasProfile: has}
}

// refuse records a request that starts nothing: a Debug line "recipe not
// started" behind Enabled and, under the flag, one count in the refusal tally. A
// recipe outside the configuration has no row in the tally and is only logged.
func (w *World) refuse(a *agent, region RegionID, recipe RecipeID, reason Reason) {
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "recipe not started",
			slog.Int64(logkeys.Agent, int64(a.id)), slog.Int64(logkeys.Recipe, int64(recipe)),
			slog.String(logkeys.Reason, string(reason)))
	}
	if w.obs.supplied && reason != "" && recipe >= 0 && int(recipe) < len(w.cfg.Recipes) {
		w.refused.add(region, recipe, reason, a.weight)
	}
}

// floorWarns advances the run of the Warn "work floor binding" of every (region,
// role) with the households the floor held at P2.5, and logs the Warn at the
// tenth tick of a run and every tenth after (spec-welfare-and-wages.md 11.3,
// FR-WEL-033, D156). The run is a diagnostic and changes no result.
func (w *World) floorWarns() {
	ctx := context.Background()
	roles := len(w.cfg.Roles)
	for cell := range w.pr.cells {
		c := &w.pr.cells[cell]
		run, warn := floorRun(w.state.floorRun[cell], c.floorHeld, c.households)
		w.state.floorRun[cell] = run
		if warn && w.log.Enabled(ctx, slog.LevelWarn) {
			w.log.LogAttrs(ctx, slog.LevelWarn, "work floor binding",
				slog.Int64(logkeys.Region, int64(cell/roles)), slog.Int64(logkeys.Role, int64(cell%roles)),
				slog.Int64(logkeys.Households, c.floorHeld))
		}
	}
}

// p27Start is P2.7: the starts of the admitted requests, region by region in
// the rotation order. Each start takes the inputs by one ledger Transform, wears
// the tools and gives the job its pending output (spec-production.md 4.3); the
// wear facts go to the welfare scratch and the agent is marked as working, once,
// at its start. The job then takes this tick's step: a job of duration 1
// completes in its start tick, as in P2.1. A start that cannot go ahead is
// refused. The error is a ledger refusal of the inputs, which fails the step.
func (w *World) p27Start() error {
	for g := range w.pr.rotation {
		err := w.walk(g, func(a *agent) error {
			ask := &w.pr.asks[a.id]
			if !ask.set {
				return nil
			}
			ask.set = false
			return w.startJob(a, RegionID(g), ask)
		})
		// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
		// refusal of the world's own accounts, which a valid configuration cannot cause
		// and no test can inject from the root (D190); the branch only wraps and returns
		// it.
		if err != nil {
			return err
		}
	}
	return nil
}

// startJob starts the job of an admitted request.
func (w *World) startJob(a *agent, region RegionID, ask *jobAsk) error {
	row := w.roles.row(a.role)
	off, _ := residueOffset(row.jobs, &w.recipes, ask.recipe)
	q := w.startReq(a, region, ask.recipe, ask.maximum)
	if w.res != nil {
		w.capDraws(&q, ask.first, ask.draws)
	}
	res, err := w.prod.start(w.ledger, &a.job, a.residues[off:off+len(w.recipes.row(ask.recipe).outputs)], &q)
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err != nil {
		return fmt.Errorf("start of recipe %d for agent %d at tick %d: %w", ask.recipe, a.id, w.Tick(), err)
	}
	if res.k == 0 {
		// Reached only when the cap of a draw is 0: the ration granted nothing, or the
		// live stock is below one execution (reason no_endowment).
		w.refuse(a, region, ask.recipe, res.reason)
		if w.sub != nil {
			w.sub.blockedStart(w, a, int(region)*len(w.cfg.Roles)+int(a.role), ask.recipe, res.reason)
		}
		return nil
	}
	if w.res != nil {
		w.takeDraws(ask.first, ask.draws, res.k)
	}
	if w.sub != nil {
		w.sub.noteStart(w, a, int(region)*len(w.cfg.Roles)+int(a.role), ask.recipe, res.k)
	}
	for _, f := range w.prod.wear {
		w.welfare.addWear(int(a.id), f.commodity, f.items, f.units)
	}
	w.welfare.markWorking(int(a.id), res.k, false)
	if done, _ := a.job.advance(); done {
		w.finishJob(a, region)
	}
	return nil
}

// p28aHalted is P2.8a: the aggregates of the work decision, only when an
// observer was supplied. ProductionHalted comes per region and role in
// ascending order where some household was halted, rested or had no recipe
// worth running, then RecipeRefused per region and recipe in ascending order
// where some request was refused (spec-observability.md 4.4.6, 4.4.6a). The
// counts are cleared at the next P2.1, and the tally by its flush.
func (w *World) p28aHalted() {
	if !w.obs.supplied {
		return
	}
	roles, recipes, tick := len(w.cfg.Roles), len(w.cfg.Recipes), w.Tick()
	for cell := range w.pr.cells {
		c := &w.pr.cells[cell]
		if c.halted == 0 && c.rested == 0 && c.noRecipe == 0 {
			continue
		}
		var rows []HaltRow
		for r, n := range w.pr.skips[cell*recipes : (cell+1)*recipes] {
			if n > 0 {
				rows = append(rows, HaltRow{Recipe: RecipeID(r), Households: n})
			}
		}
		w.obs.deliver(ProductionHalted{Tick: tick, Region: RegionID(cell / roles), Role: RoleID(cell % roles),
			Free: c.free, Halted: c.halted, Skipped: rows, Rested: c.rested, NoRecipe: c.noRecipe})
	}
	clear(w.pr.skips)
	w.refused.flush(tick, func(e RecipeRefused) { w.obs.deliver(e) })
}
