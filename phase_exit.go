package economysim

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/occupation"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file is the entry step of phase 7 (spec-world.md 6.9, plan-phase-1-11.md
// Phase 7): the view of a region that the spawner reads (P7.1(c)), the call of
// Decide and the primitives of a removal that fanin.go composes, then the
// executor of the plan (P7.1(d)). Nothing here draws a random number except the
// keyed stream of a consumer's spawner and the two creation words of each new
// agent (D10).

// exitTick is the scratch of the entry step, one per world: the view, the plan,
// and the buffers the view is built in. It is derived or scratch and never saved
// (D154); the world never shrinks it.
type exitTick struct {
	view RegionView
	plan Decisions
	// prices is the published price of each commodity in the region being
	// viewed, and upkeep the cost of one household's upkeep per role at them.
	prices []int64
	upkeep []Money
	// sets is the agents of each role of the region, for the welfare summary,
	// and unskilled the rings of the agents of the Unskilled roles.
	sets      []roleSet
	sums      []roleSummary
	unskilled []*welfare.Ring
	// seen marks the tool commodities counted in the ToolPrice of a role.
	seen         []bool
	ticksPerYear int64
	// The executor's scratch: the entry budget of the region and whether it was
	// taken, the households removed per role in the tick, the hosts an entry can
	// fill (hostAt maps a host to its slot in the agent table), the belief seeds, the
	// households of each role, and per (region, role) the scale of its group and
	// whether a group is configured.
	budget      Money
	budgetTaken bool
	removed     []int64
	hosts       []occupation.Host
	hostAt      []int
	seeds       []beliefSeed
	hh          []int64
	scale       []int64
	configured  []bool
}

// newExitTick sizes the scratch for a canonical configuration. observed is
// whether an observer was supplied, copied to the view once (plan 1-9 D119).
func newExitTick(c *Config, observed bool) exitTick {
	x := exitTick{
		prices:       make([]int64, len(c.Commodities)),
		upkeep:       make([]Money, len(c.Roles)),
		sets:         make([]roleSet, len(c.Roles)),
		sums:         make([]roleSummary, len(c.Roles)),
		seen:         make([]bool, len(c.Commodities)),
		ticksPerYear: c.Calendar.TicksPerYear(),
	}
	x.removed, x.hh = make([]int64, len(c.Roles)), make([]int64, len(c.Roles))
	x.scale, x.configured = make([]int64, len(c.Regions)*len(c.Roles)), make([]bool, len(c.Regions)*len(c.Roles))
	for i := range x.scale {
		x.scale[i] = c.World.Scale
	}
	for _, g := range c.Groups {
		r, rok := c.regionID(g.Region)
		ro, ook := c.roleID(g.Role)
		if rok && ook {
			x.scale[int(r)*len(c.Roles)+int(ro)] = cmp.Or(g.Scale, c.World.Scale)
			x.configured[int(r)*len(c.Roles)+int(ro)] = true
		}
	}
	x.view.observed = observed
	x.view.scratch = new(decideScratch)
	x.view.Roles = make([]RoleView, len(c.Roles))
	return x
}

// decideRegion is P7.1(c) for region g: it builds the view and calls the
// spawner's Decide with an empty plan, and returns the plan. A consumer's
// spawner gets the keyed stream (siteSpawner, tick, region); the default takes no
// word, so the world skips the reseed (D155).
func (w *World) decideRegion(g RegionID) *Decisions {
	v := w.buildRegionView(g)
	w.ex.plan.reset()
	if _, builtin := w.ext.spawner.(DefaultSpawner); !builtin {
		w.rnd.keyed(siteSpawner, w.Tick(), int64(g))
	}
	w.ext.spawner.Decide(v, w.rnd.rnd, &w.ex.plan)
	return &w.ex.plan
}

// buildRegionView fills the view of region g from the state as it stands (the
// agents already changed in earlier regions of the tick included) and returns
// it: the tunables of WorldParams, the pool, then the roles and the agents
// (spec-spawner.md 3.2). The view is a copy valid for the call.
func (w *World) buildRegionView(g RegionID) *RegionView {
	v, p := &w.ex.view, &w.cfg.World
	v.Tick, v.Region, v.TicksPerYear = w.Tick(), g, w.ex.ticksPerYear
	v.PoolMoney = w.estates.pool(w.ledger, g)
	v.Budget = Money(satAllowance(int64(v.PoolMoney), p.EntryPoolShare))
	v.ReviewTicks, v.EntryShare, v.RoleHysteresis, v.RoleCooldown = p.ReviewTicks, p.EntryShare, p.RoleHysteresis, p.RoleCooldown
	v.EntryRatio, v.ChangePrivation, v.RemovePrivation = p.EntryRatio, p.ChangePrivation, p.RemovePrivation
	v.DeathStep, v.ExitShare, v.Renewal, v.EntryPoolShare = p.DeathStep, p.ExitShare, p.Renewal, p.EntryPoolShare
	w.pricesOf(g)
	w.scanRegion(g)
	w.fillRoles(g)
	return v
}

// pricesOf fills the published price of every commodity in region g and the cost
// of one household's upkeep per role at them (spec-spawner.md 3.2).
func (w *World) pricesOf(g RegionID) {
	x, nc := &w.ex, len(w.cfg.Commodities)
	for c := range x.prices {
		x.prices[c] = int64(w.markets.reports[int(g)*nc+c].Price)
	}
	for r := range x.upkeep {
		x.upkeep[r] = Money(w.reads.cost.tick(w.roles.row(RoleID(r)).upkeepIn(g), x.prices, 1))
	}
}

// scanRegion lists the household agents of region g in ascending ID into the
// view and into the per-role sets of the welfare summary. A pseudo-agent and a
// removed agent (weight 0) are skipped.
func (w *World) scanRegion(g RegionID) {
	x := &w.ex
	x.view.Agents = x.view.Agents[:0]
	x.unskilled = x.unskilled[:0]
	for r := range x.sets {
		s := &x.sets[r]
		s.rings, s.weights, s.targets, s.free = s.rings[:0], s.weights[:0], s.targets[:0], s.free[:0]
	}
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		if a := &w.agents[i]; a.weight >= 1 && w.regionOf(a) == g {
			w.addAgentRow(a)
		}
	}
}

// addAgentRow adds agent a to the view and to the set of its role.
func (w *World) addAgentRow(a *agent) {
	x := &w.ex
	acct, err := w.ledger.Account(a.account)
	if err != nil {
		invariant("the account of an agent is unreadable: " + err.Error())
	}
	s := &x.sets[a.role]
	s.rings, s.weights = append(s.rings, &a.history), append(s.weights, a.weight)
	s.targets, s.free = append(s.targets, a.target), append(s.free, !a.hasJob())
	if w.cfg.Roles[a.role].Unskilled {
		x.unskilled = append(x.unskilled, &a.history)
	}
	x.view.Agents = append(x.view.Agents, RegionAgent{
		Agent: a.id, Role: a.role, Weight: a.weight, External: a.external, Mortal: a.mortal,
		Money: Money(acct.Balance), Unfed: a.unfed, Hunger: a.hunger, Privation: a.privation,
		RoleSince: a.roleSince, Upkeep: x.upkeep[a.role],
	})
}

// fillRoles fills the per-role group of the view of region g, in canonical role
// order. The unskilled wage of the region is read first, because the welfare
// summary of every role compares with it (premium).
func (w *World) fillRoles(g RegionID) {
	x := &w.ex
	unskilled, hasUnskilled := w.reads.observedWage(x.unskilled)
	for r := range x.view.Roles {
		id, cfg := RoleID(r), &w.cfg.Roles[r]
		sum := w.reads.summarise(x.sets[r], paramsOf(&w.cfg, id, &w.roles), w.roles.row(id).upkeepIn(g), x.prices, unskilled, hasUnskilled)
		x.sums[r] = sum
		x.view.Roles[r] = RoleView{
			Role: id, Closed: cfg.Closed, Households: households(x.sets[r]), Agents: int64(len(x.sets[r].rings)),
			Wage: Money(sum.Wage), HasWage: sum.HasWage, Upkeep: x.upkeep[r],
			ExpectedWorkTicks: cfg.ExpectedWorkTicks, StartMoney: cfg.StartMoney, ToolPrice: w.toolPrice(id),
			TrainingTicks: cfg.TrainingTicks, CapitalReturn: w.cfg.Regions[g].CapitalReturn,
			RenewalCarry: w.state.renewalCarry[int(g)*len(w.cfg.Roles)+r],
		}
	}
}

// toolPrice is the published price, in the region being viewed, of the tool
// commodities the recipes of role id require, each counted once; 0 when the role
// needs none. The sum saturates.
func (w *World) toolPrice(id RoleID) Money {
	x := &w.ex
	clear(x.seen)
	var total int64
	for _, rid := range w.roles.row(id).recipes {
		for _, t := range w.recipes.row(rid).tools {
			if !x.seen[t.commodity] {
				x.seen[t.commodity] = true
				total = satAdd(total, x.prices[t.commodity])
			}
		}
	}
	return Money(total)
}

// abandonJob ends the job of agent a, which moves nothing, and emits
// RecipeAbandoned with the reason under the flag (spec-production.md 5.4). The
// caller has checked that there is a job.
func (w *World) abandonJob(a *agent, region RegionID, reason Reason) {
	recipe, executions := a.job.abandon()
	if w.obs.supplied {
		w.obs.deliver(recipeAbandoned(w.Tick(), a.id, region, recipe, executions, reason))
	}
}

// logRemoval writes the Debug line "agent bankrupt" (spec-spawner.md 13.3).
func (w *World) logRemoval(a *agent, cause Cause) {
	ctx := context.Background()
	if !w.log.Enabled(ctx, slog.LevelDebug) {
		return
	}
	w.log.LogAttrs(ctx, slog.LevelDebug, "agent bankrupt",
		slog.Int64(logkeys.Agent, int64(a.id)), slog.Int64(logkeys.Role, int64(a.role)),
		slog.String(logkeys.Reason, string(cause)))
}

// forget takes a removed agent out of the world's counters and gives its slot
// the zero agent, as the estate pseudo-agents hold: weight 0, so every loop
// skips it. The ID is never reused (FR-AGT-001).
func (w *World) forget(a *agent, region RegionID) {
	w.live.agents--
	w.live.households -= a.weight
	w.live.inRole[int(region)*len(w.cfg.Roles)+int(a.role)]--
	*a = agent{}
}

// reasonUnknownKind is the Debug reason of a decision whose kind the slice does
// not know (a module's kind with its module off). It is counted as a refusal.
const reasonUnknownKind Reason = "unknown_kind"

// p71Entry is P7.1 (spec-world.md 6): for each region in ascending RegionID the
// view, Decide and the execution of the plan, one region fully before the next,
// so the entrants of region r exist when region r+1 is viewed. The error is the
// failure of a ledger move the world cannot recover from.
func (w *World) p71Entry() error {
	w.tally.reset()
	for g := range w.cfg.Regions {
		// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
		// refusal of the world's own accounts, which a valid configuration cannot cause
		// and no test can inject from the root (D190); the branch only wraps and returns
		// it.
		if err := w.entryRegion(RegionID(g)); err != nil {
			return err
		}
	}
	return nil
}

// entryRegion runs the entry step of region g: Decide, the plan in order, the
// renewal carry, the logs and the one EntryLimited of the region (FR-SPN-035).
func (w *World) entryRegion(g RegionID) error {
	plan := w.decideRegion(g)
	w.tally.addPlan(g, plan)
	w.ex.budgetTaken = false
	clear(w.ex.removed)
	for _, d := range plan.List {
		// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
		// refusal of the world's own accounts, which a valid configuration cannot cause
		// and no test can inject from the root (D190); the branch only wraps and returns
		// it.
		if err := w.execute(g, d); err != nil {
			return fmt.Errorf("execute the plan of region %d at tick %d: %w", g, w.Tick(), err)
		}
	}
	w.advanceRenewal(g)
	w.logNoViable(plan)
	w.entryWarns(g)
	if lim, ok := w.tally.limited(w.Tick(), g); ok && w.obs.supplied {
		w.obs.deliver(lim)
	}
	return nil
}

// execute checks one decision against live state and applies it. A refused
// decision changes nothing: it is counted and logged, and the plan goes on
// (spec-spawner.md 4, D196).
func (w *World) execute(g RegionID, d Decision) error {
	if reason := w.refusal(g, &d); reason != "" {
		w.tally.addExecuted(g, 0, 0, 0, 0, 1)
		w.logRefused(d, reason)
		return nil
	}
	if d.Kind == DecideSwitch || d.Kind == DecideEnter {
		w.takeBudget(g)
	}
	switch d.Kind {
	case DecideRemove:
		return w.execRemove(g, &w.agents[d.Agent], d.Cause)
	case DecideReduce:
		return w.execReduce(g, &w.agents[d.Agent], d)
	case DecideSwitch:
		return w.execSwitch(g, &w.agents[d.Agent], d)
	}
	return w.execEnter(g, d)
}

// refusal returns the reason a decision is refused, or "" when it passes. The
// kinds of the slice map one to one onto internal/occupation's.
func (w *World) refusal(g RegionID, d *Decision) Reason {
	if d.Kind < DecideRemove || d.Kind > DecideEnter {
		return reasonUnknownKind
	}
	f := occupation.Facts{
		RoleKnown: d.Role >= 0 && int(d.Role) < len(w.cfg.Roles), Households: d.Households,
		CauseSupported: d.Kind != DecideReduce || d.Cause == CauseStarved || d.Cause == CauseInsolvent,
	}
	if a := w.agentIn(g, d.Agent); a != nil {
		f.AgentKnown, f.External, f.SameRole, f.Weight = true, a.external && !a.mortal, a.role == d.Role, a.weight
	}
	return refusalReason(occupation.Refuse(occupation.Kind(d.Kind), f))
}

// agentIn returns the household agent id when it exists and its account is in
// region g, else nil. An estate pseudo-agent and a removed agent are not found.
func (w *World) agentIn(g RegionID, id AgentID) *agent {
	if id < AgentID(len(w.cfg.Regions)) || id >= AgentID(len(w.agents)) {
		return nil
	}
	if a := &w.agents[id]; a.weight >= 1 && w.regionOf(a) == g {
		return a
	}
	return nil
}

// takeBudget takes the entry budget of region g once, at the first switch or
// entry of the plan, so the estates of the removals before it fund the births
// (spec-world.md C15, FR-SPN-026).
func (w *World) takeBudget(g RegionID) {
	x := &w.ex
	if x.budgetTaken {
		return
	}
	x.budgetTaken = true
	x.budget = Money(satAllowance(int64(w.estates.pool(w.ledger, g)), w.cfg.World.EntryPoolShare))
}

// execRemove removes agent a with the cause the decision carries and counts its
// households as removed.
func (w *World) execRemove(g RegionID, a *agent, cause Cause) error {
	role, weight := a.role, a.weight
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err := w.removeAgent(a, cause); err != nil {
		return err
	}
	w.ex.removed[role] = satAdd(w.ex.removed[role], weight)
	w.tally.addExecuted(g, 0, 0, weight, 0, 0)
	return nil
}

// execReduce lowers the weight of agent a by d.Households (spec-spawner.md 4.1,
// spec-agents.md 3.7): the money and each good fall to floor(x * w' / w) and the
// difference goes to the estate by starvation; the job is re-weighed. A
// reduction by the whole weight is a removal and keeps the decision's cause.
func (w *World) execReduce(g RegionID, a *agent, d Decision) error {
	from, to := a.weight, a.weight-d.Households
	if to == 0 {
		return w.execRemove(g, a, d.Cause)
	}
	money, units, err := w.shed(g, a, to)
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err != nil {
		return err
	}
	a.job.reweigh(to)
	a.weight = to
	w.live.households -= d.Households
	w.ex.removed[a.role] = satAdd(w.ex.removed[a.role], d.Households)
	w.tally.addExecuted(g, 0, 0, d.Households, 0, 0)
	if w.obs.supplied {
		w.obs.deliver(WeightChanged{Tick: w.Tick(), Agent: a.id, Region: g, Role: a.role, From: from, To: to, Cause: d.Cause, Money: -money, Units: units})
	}
	return nil
}

// shed sends to the estate of region g what agent a holds above floor(x * to /
// weight), money then each commodity in canonical order, by the reason
// starvation, and returns the money and the units sent.
func (w *World) shed(g RegionID, a *agent, to int64) (money Money, units int64, err error) {
	acct, aerr := w.ledger.Account(a.account)
	if aerr != nil {
		invariant("the account of a reduced agent is unreadable: " + aerr.Error())
	}
	keep := mustRemain(acct.Balance, a.weight, to)
	money = Money(acct.Balance - keep)
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err = w.estates.acceptMoney(w.ledger, a.account, g, money, ledger.Starvation); err != nil {
		return 0, 0, err
	}
	for c := range w.cfg.Commodities {
		held, _, herr := w.ledger.Holding(a.account, c)
		if herr != nil {
			invariant("the holding of a reduced agent is unreadable: " + herr.Error())
		}
		n, uerr := w.estates.acceptUnits(w.ledger, a.account, g, c, held-mustRemain(held, a.weight, to), ledger.Starvation)
		// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
		// refusal of the world's own accounts, which a valid configuration cannot cause
		// and no test can inject from the root (D190); the branch only wraps and returns
		// it.
		if uerr != nil {
			return money, units, uerr
		}
		units = satAdd(units, n)
	}
	return money, units, nil
}

// mustRemain is occupation.Remaining for a held amount and a weight the
// executor has checked: a refusal is a violated invariant.
func mustRemain(x, w, to int64) int64 {
	v, ok := occupation.Remaining(x, w, to)
	if !ok {
		invariant("an amount held by an agent cannot be scaled to its new weight")
	}
	return v
}

// execSwitch changes the role of agent a (spec-spawner.md 4.2, spec-agents.md
// 3.8): the job is abandoned, the goods outside the new role's tradable set go
// to the estate whole lots at a time, and the agent takes up the new role. The
// account and the money stay, so no account is opened or closed.
func (w *World) execSwitch(g RegionID, a *agent, d Decision) error {
	from, to := a.role, d.Role
	if a.hasJob() {
		w.abandonJob(a, g, ReasonRoleChanged)
	}
	newRow := w.roles.row(to)
	lots, err := w.estates.acceptGoods(w.ledger, a.account, g, ledger.EstateGoods, func(c int) bool { return !newRow.isTradable(CommodityID(c)) })
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err != nil {
		return err
	}
	w.takeRole(a, g, to)
	roles := len(w.cfg.Roles)
	w.live.inRole[int(g)*roles+int(from)]--
	w.live.inRole[int(g)*roles+int(to)]++
	w.state.hadAgent[int(g)*roles+int(to)] = true
	w.tally.addExecuted(g, a.weight, 0, 0, 0, 0)
	if w.obs.supplied {
		w.obs.deliver(RoleChanged{Tick: w.Tick(), Agent: a.id, Region: g, From: from, To: to, Weight: a.weight, Cause: d.Cause, GoodsListed: lots})
	}
	w.logRoleChanged(a, d.Cause)
	return nil
}

// takeRole sets what a role change changes in agent a: the target ratio carried
// by its offset (no draw), the welfare ring resized to the new window, the
// residues reset to 0, the beliefs kept for the commodities both roles trade and
// seeded for the others, the strategy and RoleSince.
func (w *World) takeRole(a *agent, g RegionID, to RoleID) {
	oldCfg, newCfg := &w.cfg.Roles[a.role], &w.cfg.Roles[to]
	oldRow, newRow := w.roles.row(a.role), w.roles.row(to)
	lo, hi := welfare.TargetRange(newCfg.TargetRatio, newCfg.TargetSpread)
	target, ok := occupation.Retarget(a.target, oldCfg.TargetRatio, newCfg.TargetRatio, lo, hi)
	if !ok {
		invariant("the target ratio of a role change overflows")
	}
	a.target = target
	if err := a.history.Resize(newRow.window); err != nil {
		invariant("the welfare window of a role is outside its bound after validation: " + err.Error())
	}
	a.residues = slices.Grow(a.residues[:0], residueLen(newRow.recipes, &w.recipes))[:residueLen(newRow.recipes, &w.recipes)]
	setResidues(a.residues, 0)
	w.fillSeeds(g, newRow)
	reseedBeliefs(a, newRow, oldRow, w.ex.seeds, w.bp)
	a.role, a.strategy, a.roleSince = to, int(to), w.Tick()
}

// fillSeeds fills the belief seeds of the tradable set of row in region g: the
// window mean of each market, or its published price while the window is empty
// (spec-beliefs.md 8.3), and the cost anchor as the unit cost.
func (w *World) fillSeeds(g RegionID, row *roleRow) {
	x := &w.ex
	n := len(row.tradable)
	x.seeds = slices.Grow(x.seeds[:0], n)[:n]
	for i, c := range row.tradable {
		m := w.markets.index(g, c)
		mean, has := w.markets.windows[m].Mean()
		x.seeds[i] = beliefSeed{
			Ref: int64(w.cfg.Commodities[c].ReferencePrice), UnitCost: w.prod.unitCost(g, c),
			Mean: mean, Empty: !has, Price: int64(w.markets.reports[m].Price),
		}
	}
}

// execEnter adds d.Households households to a role (spec-spawner.md 4.3): as
// many as the one budget of the region buys, each placed in the host of
// smallest weight below the group's scale or in a new agent of weight 1. The
// rest are neither created nor deferred, only counted (FR-SPN-008).
func (w *World) execEnter(g RegionID, d Decision) error {
	x, start := &w.ex, w.cfg.Roles[d.Role].StartMoney
	funded := occupation.Funded(d.Households, int64(min(x.budget, w.estates.pool(w.ledger, g))), int64(start))
	x.budget -= Money(funded) * start
	unfunded := d.Households - funded
	w.tally.addExecuted(g, 0, funded, 0, unfunded, 0)
	if unfunded > 0 {
		w.logDropped(d.Role, ReasonUnfunded)
	}
	w.gatherHosts(g, d.Role)
	for left := funded; left > 0; {
		n, err := w.placeOne(g, d, left)
		// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
		// refusal of the world's own accounts, which a valid configuration cannot cause
		// and no test can inject from the root (D190); the branch only wraps and returns
		// it.
		if err != nil {
			return err
		}
		left -= n
	}
	return nil
}

// gatherHosts lists the agents of role in region g in ascending ID, the
// candidates of Place, into the executor's scratch.
func (w *World) gatherHosts(g RegionID, role RoleID) {
	x := &w.ex
	x.hosts, x.hostAt = x.hosts[:0], x.hostAt[:0]
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		if a := &w.agents[i]; a.weight >= 1 && a.role == role && w.regionOf(a) == g {
			x.hosts = append(x.hosts, occupation.Host{Ref: len(x.hostAt), Weight: a.weight})
			x.hostAt = append(x.hostAt, i)
		}
	}
}

// placeOne places some of the left households of an entry: into the host
// Place chooses, up to its room, or as one new agent of weight 1. It returns the
// households placed.
func (w *World) placeOne(g RegionID, d Decision, left int64) (int64, error) {
	x := &w.ex
	scale := x.scale[int(g)*len(w.cfg.Roles)+int(d.Role)]
	ref, room, ok := occupation.Place(x.hosts, scale)
	if !ok {
		return 1, w.spawnEntrant(g, d)
	}
	take := min(left, room)
	x.hosts[ref].Weight += take
	return take, w.enlarge(g, &w.agents[x.hostAt[ref]], take, d.Cause)
}

// enlarge raises the weight of an existing agent by take households, each
// bringing the role's StartMoney from the pool by births, and opens no account.
func (w *World) enlarge(g RegionID, a *agent, take int64, cause Cause) error {
	given, err := w.estates.grant(w.ledger, g, a.account, Money(take)*w.cfg.Roles[a.role].StartMoney, ledger.Births)
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err != nil {
		return err
	}
	from := a.weight
	a.weight += take
	w.live.households += take
	if w.obs.supplied {
		w.obs.deliver(WeightChanged{Tick: w.Tick(), Agent: a.id, Region: g, Role: a.role, From: from, To: a.weight, Cause: cause, Money: given})
	}
	return nil
}

// spawnEntrant creates one new agent of weight 1 for an entry (spec-agents.md
// 3.5.3): its account, the role's StartMoney from the pool by start_money and no
// goods, beliefs from the window mean or the price, the two creation draws of D10
// and the counters. It joins the hosts of the entry in progress.
func (w *World) spawnEntrant(g RegionID, d Decision) error {
	id, role, row := w.state.nextAgent, &w.cfg.Roles[d.Role], w.roles.row(d.Role)
	if int(id) != len(w.agents) {
		invariant("the agent ID space is not dense")
	}
	acct, err := w.ledger.Open(ledger.KindAgent, int(id), int(g))
	if err != nil {
		invariant("the ledger refused the opening of an entrant's account: " + err.Error())
	}
	a := newAgent(agentInit{id: id, account: acct, role: d.Role, weight: 1, strategy: int(d.Role), tick: w.Tick()}, row, &w.recipes)
	given, err := w.estates.grant(w.ledger, g, acct, role.StartMoney, ledger.StartMoney)
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err != nil {
		return err
	}
	w.fillSeeds(g, row)
	reseedBeliefs(&a, row, nil, w.ex.seeds, w.bp)
	w.rnd.keyed(siteCreation, w.Tick(), int64(id))
	drawCreation(w.rnd, &a, role.TargetRatio, role.TargetSpread)
	x, cell := &w.ex, int(g)*len(w.cfg.Roles)+int(d.Role)
	x.hosts = append(x.hosts, occupation.Host{Ref: len(x.hostAt), Weight: 1})
	x.hostAt = append(x.hostAt, len(w.agents))
	w.agents = append(w.agents, a)
	w.state.nextAgent++
	w.live.agents++
	w.live.households++
	w.live.inRole[cell]++
	w.state.hadAgent[cell] = true
	if w.obs.supplied {
		w.obs.deliver(AgentSpawned{Tick: w.Tick(), Agent: id, Region: g, Role: d.Role, Weight: 1, StartMoney: given})
	}
	return nil
}

// advanceRenewal applies the renewal rule to the households the executor
// removed in region g, per role, and stores the carry (spec-spawner.md 7.1,
// D145). Decide sized its entries by the same rule on a copy.
func (w *World) advanceRenewal(g RegionID) {
	roles := len(w.cfg.Roles)
	for r, removed := range w.ex.removed {
		cell := int(g)*roles + r
		_, carry, ok := occupation.Renewal(w.state.renewalCarry[cell], removed, w.cfg.World.Renewal)
		if !ok {
			invariant("the renewal carry overflows")
		}
		w.state.renewalCarry[cell] = carry
	}
}

// entryWarns advances the runs of the three bounded Warns of the entry step for
// region g, after its plan ran, and logs them on the tick the run says
// (spec-spawner.md 13.3, D156): "entry unfunded" per region, "role extinct" and
// "group below floor" per role.
func (w *World) entryWarns(g RegionID) {
	ctx, x := context.Background(), &w.ex
	row := w.tally.row(g)
	run, warn := unfundedRun(w.state.unfundedRun[g], satAdd(row.entered, row.unfunded), row.entered)
	w.state.unfundedRun[g] = run
	if warn && w.log.Enabled(ctx, slog.LevelWarn) {
		w.log.LogAttrs(ctx, slog.LevelWarn, "entry unfunded", slog.Int64(logkeys.Region, int64(g)), slog.Int64(logkeys.Households, row.unfunded))
	}
	clear(x.hh)
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		if a := &w.agents[i]; a.weight >= 1 && w.regionOf(a) == g {
			x.hh[a.role] += a.weight
		}
	}
	for r := range x.hh {
		w.roleWarns(ctx, g, RoleID(r))
	}
}

// roleWarns advances the runs of "role extinct" and "group below floor" of
// (region g, role r) and logs them.
func (w *World) roleWarns(ctx context.Context, g RegionID, r RoleID) {
	cell := int(g)*len(w.cfg.Roles) + int(r)
	agents := w.live.inRole[cell]
	run, warn := extinctRun(w.state.extinctRun[cell], agents, w.state.hadAgent[cell], w.ex.configured[cell])
	w.state.extinctRun[cell] = run
	if warn && w.log.Enabled(ctx, slog.LevelWarn) {
		w.log.LogAttrs(ctx, slog.LevelWarn, "role extinct", slog.Int64(logkeys.Region, int64(g)), slog.Int64(logkeys.Role, int64(r)))
	}
	run, warn = belowFloorRun(w.state.belowFloorRun[cell], w.ex.hh[r], agents, w.cfg.World.MinAgents)
	w.state.belowFloorRun[cell] = run
	if warn && w.log.Enabled(ctx, slog.LevelWarn) {
		w.log.LogAttrs(ctx, slog.LevelWarn, "group below floor",
			slog.Int64(logkeys.Region, int64(g)), slog.Int64(logkeys.Role, int64(r)), slog.Int64(logkeys.Agents, agents))
	}
}

// logRefused writes the Debug line "decision refused" (spec-spawner.md 13.3).
func (w *World) logRefused(d Decision, reason Reason) {
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "decision refused", slog.Int64(logkeys.Agent, int64(d.Agent)), slog.String(logkeys.Reason, string(reason)))
	}
}

// logRoleChanged writes the Debug line "role changed" (spec-spawner.md 13.3).
func (w *World) logRoleChanged(a *agent, cause Cause) {
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "role changed",
			slog.Int64(logkeys.Agent, int64(a.id)), slog.Int64(logkeys.Role, int64(a.role)), slog.String(logkeys.Reason, string(cause)))
	}
}

// logDropped writes the Debug line "entry dropped" for a role and a reason.
func (w *World) logDropped(role RoleID, reason Reason) {
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "entry dropped", slog.Int64(logkeys.Role, int64(role)), slog.String(logkeys.Reason, string(reason)))
	}
}

// logNoViable writes the "entry dropped" lines of the renewals the default
// found no viable role for, one per role, in role order.
func (w *World) logNoViable(plan *Decisions) {
	for r, n := range plan.noViableByRole {
		if n > 0 {
			w.logDropped(RoleID(r), ReasonNoViableRole)
		}
	}
}

// p81Age is P8.1 (spec-world.md 6, spec-ledger.md 5.3): the ageing of the tick.
// Every lot that spoils by this tick moves whole to the spoilage boundary. The
// ledger refuses only a configuration the world never builds, so an error is
// returned as the failure of the step, naming the tick.
func (w *World) p81Age() error {
	// Excluded from coverage (testing.md 9.5): the error is a ledger or overflow
	// refusal of the world's own accounts, which a valid configuration cannot cause
	// and no test can inject from the root (D190); the branch only wraps and returns
	// it.
	if err := w.ledger.Age(); err != nil {
		return fmt.Errorf("ageing at tick %d: %w", w.Tick(), err)
	}
	return nil
}

// p82Flows is P8.2 (D160): with an observer, the tick's BoundaryFlow rows of the
// faucet and sink reasons, then the WearApplied rows in ascending commodity. The
// totals are read, not cleared; P8.6 clears them.
func (w *World) p82Flows() {
	if !w.obs.supplied {
		return
	}
	t := &w.tk
	t.flows = w.ledger.AppendFlows(t.flows[:0], false)
	t.boundary = boundaryFlows(t.boundary[:0], t.flows, w.Tick())
	for _, e := range t.boundary {
		w.obs.deliver(e)
	}
	t.wear = wearApplied(t.wear[:0], w.ledger, len(w.cfg.Commodities), w.Tick(), false)
	for _, e := range t.wear {
		w.obs.deliver(e)
	}
}

// p84Audit is P8.4: with WithAudit, the ledger's audit mapped by auditFailure
// (D149, D150, D190). Its error is the argument of endStep, which fails the
// world before TickEnded. It writes nothing.
func (w *World) p84Audit() error {
	if !w.audit {
		return nil
	}
	return auditFailure(w.Tick(), w.ledger.Audit())
}

// logTickEnded writes the Debug line "tick ended" (D163) from the live
// counters, not by a scan.
func (w *World) logTickEnded() {
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "tick ended", slog.Int64(logkeys.Tick, w.Tick()),
			slog.Int64(logkeys.Agents, w.live.agents), slog.Int64(logkeys.Households, w.live.households))
	}
}
