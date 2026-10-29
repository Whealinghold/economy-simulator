package economysim

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/ownuse"
)

// This file is the subsistence module (spec-subsistence.md): the canonical copy
// of its configuration, the effective own-use list of a role and its join into
// the role table, the state (the flag and the counter of each agent), the flag
// step P2.4, the filler of the view, the choice of the default Work in the mode
// and the hooks the world calls at a start, a drop and a role change. The
// validator and the warnings are in subsistence_validate.go, the types in
// config.go and the pure rules in internal/ownuse.

// canonical returns a copy with the default list and every row's list sorted by
// recipe ID and the rows sorted by role (spec-subsistence.md 3.2.3), so two
// spellings of one configuration hash alike. A nil module stays nil; a module
// that is on and empty stays on.
func (s *SubsistenceConfig) canonical() *SubsistenceConfig {
	if s == nil {
		return nil
	}
	out := &SubsistenceConfig{
		Recipes:    sortedBy(s.Recipes, cmp.Compare[string]),
		Roles:      sortedBy(s.Roles, func(a, b RoleOwnUse) int { return cmp.Compare(a.Role, b.Role) }),
		EnterTicks: s.EnterTicks,
		ExitTicks:  s.ExitTicks,
		RetryJobs:  s.RetryJobs,
	}
	for i := range out.Roles {
		out.Roles[i].Recipes = sortedBy(out.Roles[i].Recipes, cmp.Compare[string])
	}
	return out
}

// ownUseList returns the effective own-use list of a role (spec-subsistence.md
// 3.2.1, 3.3.1): the list of its row if it has one, an empty row meaning none,
// and the module's default list otherwise, sorted by recipe ID. It is nil when
// the module is off or the list is empty, and a copy the caller may keep.
func (s *SubsistenceConfig) ownUseList(role string) []string {
	if s == nil {
		return nil
	}
	list := s.Recipes
	for i := range s.Roles {
		if s.Roles[i].Role == role {
			list = s.Roles[i].Recipes
			break
		}
	}
	return sortedBy(list, cmp.Compare[string])
}

// ownUseJobs resolves the effective own-use list of a role of a configuration
// that passed Validate and returns it with the union of the role's own recipes
// and that list, both ascending by RecipeID (architecture.md 2.10.1: the join
// into the role table). own is nil and jobs is recipes itself when the module is
// off (a nil receiver) or the list is empty, so a module that does nothing
// changes no slice.
func (s *SubsistenceConfig) ownUseJobs(c *Config, role string, recipes []RecipeID) (own, jobs []RecipeID) {
	list := s.ownUseList(role)
	if len(list) == 0 {
		return nil, recipes
	}
	for _, id := range list {
		rid, ok := c.recipeID(id)
		if !ok {
			// Not covered by a test: validation rejects an unknown recipe before this
			// runs, so no valid configuration reaches the panic (testing.md 9.5).
			panic("economysim: invariant violated: role " + role + " has unknown own-use recipe " + id + " after validation")
		}
		own = append(own, rid)
	}
	slices.Sort(own)
	jobs = append(slices.Clone(recipes), own...)
	slices.Sort(jobs)
	return own, jobs
}

// subsistenceItems adds the non-discard outputs of the effective own-use list
// of a role to its tradable set, so a household may hold and sell what it makes
// (spec-subsistence.md 3.6, spec-agents.md 3.4). It is the module's call in the
// fifth fan-in, tradableItems.
func subsistenceItems(c *Config, lk lookup, role *Role, set map[string]bool) {
	for _, id := range c.Modules.Subsistence.ownUseList(role.ID) {
		rc := lk.recipe[id]
		if rc == nil {
			continue
		}
		for _, o := range rc.Outputs {
			if !o.Discard {
				set[o.Commodity] = true
			}
		}
	}
}

// ownUseOnly returns, by recipe, whether it is on the own-use list of some role
// and on no role's Recipes: such a recipe is no activity of the reference
// programme and makes no commodity producible (spec-viability.md 3.1, 4.2,
// FR-SUB-037). The floor is for when the markets fail, so the steady state does
// not stand on it.
func (m *viaModel) ownUseOnly() []bool {
	own, market := make([]bool, len(m.c.Recipes)), make([]bool, len(m.c.Recipes))
	for i := range m.c.Roles {
		row := m.roles.row(RoleID(i))
		for _, k := range row.ownUse {
			own[k] = true
		}
		for _, k := range row.recipes {
			market[k] = true
		}
	}
	for k := range own {
		own[k] = own[k] && !market[k]
	}
	return own
}

// blockReasons are the reasons a household in the mode is blocked, the last being
// any other; the Warn "own use blocked" names the commonest (spec-subsistence.md
// 14.2). A later module that adds a refusal appends its reason here (D418).
var blockReasons = [...]Reason{ReasonZeroYield, ReasonNoSite, ReasonNoEndowment, ReasonNoRecipe}

// blockReasonAt is the index of a reason in blockReasons, the last for any other.
func blockReasonAt(r Reason) int {
	if i := slices.Index(blockReasons[:], r); i >= 0 {
		return i
	}
	return len(blockReasons) - 1
}

// subsistence is the state of the module, nil in the world when it is off
// (spec-subsistence.md 15, plan 2-10 D403). The flag and the counter of an agent
// are in slices indexed by agent ID, never fields of the agent (architecture.md
// 2.10.1), and grown like the welfare scratch. covered and the parameters are
// derived from the configuration and never saved; the counts of the tick are
// scratch that the event of P2.8 reads (D409).
type subsistence struct {
	params    ownuse.Params
	retryJobs int64
	// covered is, per role, the upkeep commodities of the role in any region that
	// are a non-discard output of a recipe of its own-use list, in canonical order
	// (spec-subsistence.md 3.3.2, D405).
	covered [][]CommodityID

	ownUse     []bool
	ownUseJobs []int64

	// The counts of the tick per (region, role) cell, in households: made always,
	// because the Warns need them with no observer (D409).
	entered, left, started, blocked, retried []int64
	// inMode is the households with the flag at the end of the flag step, per cell;
	// blockedBy splits blocked by the reason that blocked it (blockReasons); output
	// is the units created at the completion of own-use jobs, per cell and
	// commodity, kept only under an observer (observability-and-logging.md 4.5).
	inMode    []int64
	blockedBy [][len(blockReasons)]int64
	output    []int64
	// runOn and runBlocked are the consecutive ticks of the two Warns per cell:
	// diagnostics that change no result and are saved in section 19.
	runOn, runBlocked []int64

	// goods is the scratch of the covered goods of the agent being evaluated.
	goods []ownuse.Good

	// receipts are the units created at the completion of own-use jobs in the
	// tick, valued and added to the welfare scratch at P6.3 (D410).
	receipts []ownReceipt
}

// ownReceipt is the units of one commodity an own-use job of an agent created.
type ownReceipt struct {
	slot      int
	commodity CommodityID
	units     int64
}

// newSubsistence builds the state from a canonical configuration that passed
// Validate; it is nil while the module is off.
func newSubsistence(c *Config, rt *roleTable, rc *recipeTable) *subsistence {
	m := c.Modules.Subsistence
	if m == nil {
		return nil
	}
	cells := len(c.Regions) * len(c.Roles)
	s := &subsistence{
		params:    ownuse.Params{Enter: m.EnterTicks, Exit: m.ExitTicks, Max: MaxOwnUseTicks},
		retryJobs: m.RetryJobs,
		covered:   make([][]CommodityID, len(c.Roles)),
		goods:     make([]ownuse.Good, 0, MaxUpkeepItems),
		entered:   make([]int64, cells), left: make([]int64, cells), started: make([]int64, cells),
		blocked: make([]int64, cells), retried: make([]int64, cells),
		inMode: make([]int64, cells), blockedBy: make([][len(blockReasons)]int64, cells),
		output: make([]int64, cells*len(c.Commodities)), runOn: make([]int64, cells), runBlocked: make([]int64, cells),
	}
	for i := range s.covered {
		row := rt.row(RoleID(i))
		makes := map[CommodityID]bool{}
		for _, k := range row.ownUse {
			for _, o := range rc.row(k).outputs {
				if !o.discard {
					makes[o.commodity] = true
				}
			}
		}
		for _, c := range upkeepCommodities(row) {
			if makes[c] {
				s.covered[i] = append(s.covered[i], c)
			}
		}
	}
	return s
}

// upkeepCommodities returns the commodities of the upkeep of a role in any
// region, ascending.
func upkeepCommodities(row *roleRow) []CommodityID {
	var out []CommodityID
	for _, it := range row.upkeep {
		out = append(out, it.commodity)
	}
	for _, g := range row.regionUpkeep {
		for _, it := range g.items {
			out = append(out, it.commodity)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// grow makes the slices cover n agent slots, outside the hot path of a tick (an
// agent entering).
func (s *subsistence) grow(n int) {
	for len(s.ownUse) < n {
		s.ownUse, s.ownUseJobs = append(s.ownUse, false), append(s.ownUseJobs, 0)
	}
}

// drop sets the flag and the counter of an agent to their defaults: a removal,
// a relocation (spec-subsistence.md 5.8, FR-SUB-031, 032). Plans 14 and 17 call
// it too.
func (s *subsistence) drop(id AgentID) {
	if int(id) < len(s.ownUse) {
		s.ownUse[id], s.ownUseJobs[id] = false, 0
	}
}

// roleChanged resets the counter and keeps the flag, which the next flag step
// evaluates against the new role (FR-SUB-032). It is the module's call in the
// fan-in agentRoleChanged.
func (s *subsistence) roleChanged(a *agent) {
	if int(a.id) < len(s.ownUseJobs) {
		s.ownUseJobs[a.id] = 0
	}
}

// agentFields returns the flag and the counter of an agent slot, for the
// per-agent query of plan 17.
func (s *subsistence) agentFields(slot int) (on bool, jobs int64) {
	if slot < len(s.ownUse) {
		return s.ownUse[slot], s.ownUseJobs[slot]
	}
	return false, 0
}

// on reports whether the agent is in the mode; it is false for a nil module.
func (s *subsistence) on(a *agent) bool {
	return s != nil && int(a.id) < len(s.ownUse) && s.ownUse[a.id]
}

// isOwn reports whether recipe is on the own-use list of role.
func isOwn(row *roleRow, recipe RecipeID) bool {
	_, ok := slices.BinarySearch(row.ownUse, recipe)
	return ok
}

// ownJob reports whether the job of the agent runs an own-use recipe; it is
// false for a nil module and for an agent with no job.
func (s *subsistence) ownJob(w *World, a *agent) bool {
	return s != nil && a.hasJob() && isOwn(w.roles.row(a.role), a.job.recipe)
}

// goodsOf fills the covered goods of an agent in region g (spec-subsistence.md
// 5.2): the covered commodities whose upkeep in the region is above 0, with the
// whole stock, the upkeep of the whole agent and the published price. ok is
// false when the upkeep of the weight overflows. The slice is the module's
// scratch, valid until the next call.
func (s *subsistence) goodsOf(w *World, a *agent, g RegionID) (goods []ownuse.Good, ok bool) {
	items := w.roles.row(a.role).upkeepIn(g)
	s.goods = s.goods[:0]
	for _, c := range s.covered[a.role] {
		var q Quantity
		for _, it := range items {
			if it.commodity == c {
				q = it.quantity
			}
		}
		if q < 1 {
			continue
		}
		held, _, err := w.ledger.Holding(a.account, int(c))
		if err != nil {
			invariant("the stock of a covered commodity is unreadable: " + err.Error())
		}
		u, fits := checked.Mul(a.weight, int64(q))
		if !fits {
			return nil, false
		}
		s.goods = append(s.goods, ownuse.Good{Stock: held, Upkeep: u, Price: int64(w.markets.reports[w.markets.index(g, c)].Price)})
	}
	return s.goods, true
}

// p24OwnUse is P2.4: the flag step. For every household agent in ascending ID it
// evaluates the mode from the agent's own stock, money, Hunger, upkeep, counter
// and the prices of its own region, after the continuations, the completions and
// the snapshot of the endowments and before the first Work (spec-subsistence.md
// 5.3, 5.4, FR-SUB-017). It draws nothing and reads no other agent. An agent
// away is skipped, its flag frozen; an agent with no covered good in its region
// drops out of the mode, counted as having left.
func (w *World) p24OwnUse() {
	s := w.sub
	s.grow(len(w.agents))
	clear(s.inMode)
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		a := &w.agents[i]
		if a.weight < 1 {
			continue
		}
		if !w.offStage(a) {
			s.evaluate(w, a)
		}
		if s.ownUse[i] {
			s.inMode[int(w.regionOf(a))*len(w.cfg.Roles)+int(a.role)] += a.weight
		}
	}
}

// evaluate is the flag step of one agent.
func (s *subsistence) evaluate(w *World, a *agent) {
	g := w.regionOf(a)
	goods, ok := s.goodsOf(w, a, g)
	if !ok {
		return
	}
	cell := int(g)*len(w.cfg.Roles) + int(a.role)
	if len(goods) == 0 {
		if s.ownUse[a.id] {
			s.left[cell] += a.weight
			s.debug(w, "own use left", a, "cover")
		}
		s.ownUse[a.id], s.ownUseJobs[a.id] = false, 0
		return
	}
	acct, err := w.ledger.Account(a.account)
	if err != nil {
		invariant("the account of an agent is unreadable: " + err.Error())
	}
	r, ok := ownuse.Step(s.params, s.ownUse[a.id], s.ownUseJobs[a.id], goods, a.hunger > 0, acct.Balance-acct.Reserved)
	if !ok {
		return
	}
	s.ownUse[a.id], s.ownUseJobs[a.id] = r.On, r.Jobs
	if r.Entered {
		s.entered[cell] += a.weight
		s.debug(w, "own use entered", a, enterReason(a.hunger))
	}
	if r.Left {
		s.left[cell] += a.weight
		s.debug(w, "own use left", a, "cover")
	}
}

// enterReason is why a household entered the mode: unfed when its Hunger decided
// it (it counts no money), short_cover otherwise (spec-subsistence.md 14.2).
func enterReason(hunger int64) string {
	if hunger > 0 {
		return "unfed"
	}
	return "short_cover"
}

// debug logs a per-agent Debug line of the module behind Enabled; an empty reason
// leaves the attribute out.
func (s *subsistence) debug(w *World, msg string, a *agent, reason string) {
	ctx := context.Background()
	if !w.log.Enabled(ctx, slog.LevelDebug) {
		return
	}
	attrs := []slog.Attr{slog.Int64(logkeys.Agent, int64(a.id)), slog.Int64(logkeys.Role, int64(a.role))}
	if reason != "" {
		attrs = append(attrs, slog.String(logkeys.Reason, reason))
	}
	w.log.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
}

// fillView is the module's filler of the view (spec-subsistence.md 5.4,
// FR-SUB-018): the flag and the counter from the state, Cover and Secure
// recomputed from the stock and the prices at this moment, the two tunables a
// pure Work needs, and the own-use recipes of the role appended to the per-recipe
// group, marked, in ascending RecipeID. It is the module's call in the fan-in
// fillAgentView.
func (s *subsistence) fillView(w *World, v *AgentView, a *agent, c *viewCtx) error {
	v.OwnUse, v.OwnUseJobs = s.agentFields(int(a.id))
	v.ExitTicks, v.RetryJobs = s.params.Exit, s.retryJobs
	if goods, ok := s.goodsOf(w, a, v.Region); ok && len(goods) > 0 {
		v.Cover, v.Secure = s.figures(goods, a.hunger > 0, v.Money)
	}
	sc := v.scratch
	for _, id := range w.roles.row(a.role).ownUse {
		if err := sc.appendRecipe(v, a, c, id, true); err != nil {
			return err
		}
	}
	v.Recipes = sc.recipes[:len(sc.recipes):len(sc.recipes)]
	return nil
}

// figures returns the cover and the secure ticks of the goods; a figure that
// cannot be formed is saturated at the largest, as Step reads it.
func (s *subsistence) figures(goods []ownuse.Good, hungry bool, money int64) (cover, secure int64) {
	cover, secure = s.params.Max, s.params.Max
	c, ok := ownuse.Cover(goods)
	if !ok {
		return cover, secure
	}
	cost, ok := ownuse.Cost(goods)
	if !ok {
		return c, secure
	}
	af, ok := ownuse.Afford(hungry, money, cost)
	if !ok {
		return c, secure
	}
	return c, ownuse.Secure(c, af, s.params.Max)
}

// countWork takes the blocked and retried households of the work report of an
// agent into the counts of its cell (D409).
func (s *subsistence) countWork(w *World, cell int, a *agent, r *workReportBuf) {
	if r.ownBlocked {
		s.blocked[cell] += a.weight
		s.blockedBy[cell][blockReasonAt(r.ownReason)] += a.weight
		s.debug(w, "own use blocked", a, string(r.ownReason))
	}
	if r.ownRetried {
		s.retried[cell] += a.weight
		s.debug(w, "own use retry", a, "")
	}
}

// noteStart records a start at P2.7: an own-use start adds 1 to the counter of
// the agent and k to the executions started of its cell, a start of any other
// recipe resets the counter (FR-SUB-022).
func (s *subsistence) noteStart(w *World, a *agent, cell int, recipe RecipeID, k int64) {
	s.grow(int(a.id) + 1) // an agent created after P2.4 has no slot yet
	if isOwn(w.roles.row(a.role), recipe) {
		s.ownUseJobs[a.id]++
		s.started[cell] += k
		return
	}
	s.ownUseJobs[a.id] = 0
}

// blockedStart counts an own-use request that the start refused for want of an
// endowment (FR-SUB-023, case b). The reasons that later modules add are
// appended to the test here (D418).
func (s *subsistence) blockedStart(w *World, a *agent, cell int, recipe RecipeID, reason Reason) {
	if reason == ReasonNoEndowment && s.on(a) && isOwn(w.roles.row(a.role), recipe) {
		s.blocked[cell] += a.weight
		s.blockedBy[cell][blockReasonAt(reason)] += a.weight
		s.debug(w, "own use blocked", a, string(reason))
	}
}

// ownGoods lists the covered goods of the view in the order of its upkeep rows
// (spec-subsistence.md 5.5 item 3): an upkeep row whose commodity is a
// non-discard output of a marked recipe, and whether its cover is short of
// ExitTicks. It appends to the two slices and returns them; an upkeep of the
// weight that overflows reads as a cover that is not short.
func (v *AgentView) ownGoods(rows []UpkeepRow, short []bool) ([]UpkeepRow, []bool) {
	for _, row := range v.Upkeep {
		if !v.ownMakes(row.Commodity) {
			continue
		}
		u, fits := checked.Mul(v.Weight, row.Quantity)
		rows = append(rows, row)
		short = append(short, fits && u > 0 && v.stockOf(row.Commodity)/u < v.ExitTicks)
	}
	return rows, short
}

// ownMakes reports whether a marked recipe of the view makes c as a non-discard
// output.
func (v *AgentView) ownMakes(c CommodityID) bool {
	for i := range v.Recipes {
		if !v.Recipes[i].OwnUse {
			continue
		}
		for _, o := range v.Recipes[i].Outputs {
			if o.Commodity == c && !o.Discard {
				return true
			}
		}
	}
	return false
}

// ownChoice is the own-use recipe the default Work requests in the mode and the
// executions to request of it (spec-subsistence.md 5.5 items 3 and 4,
// FR-SUB-020, 021): the marked recipe with MaxExecutions of 1 or more and the
// greatest sum of Cov over the goods short of ExitTicks, ties to the lower
// RecipeID, and the ramp of the cohort. idx indexes v.Recipes. ok is false when
// no marked recipe qualifies, which is the blocked case (a). It takes no random
// word and, for the few goods and recipes a role has, allocates nothing.
func (v *AgentView) ownChoice() (idx int, k int64, ok bool) {
	var rowBuf [MaxUpkeepItems]UpkeepRow
	var shortBuf [MaxUpkeepItems]bool
	var optBuf [MaxOwnUseRecipes]ownuse.Option
	var covBuf [MaxOwnUseRecipes * MaxUpkeepItems]int64
	rows, short := v.ownGoods(rowBuf[:0], shortBuf[:0])
	n, used := 0, 0
	for i := range v.Recipes {
		r := &v.Recipes[i]
		// A role has at most MaxOwnUseRecipes of them (validation), so the arrays
		// hold every option; a hand-built view with more is cut to the first.
		if !r.OwnUse || n == len(optBuf) || used+len(rows) > len(covBuf) {
			continue
		}
		covs := covBuf[used : used+len(rows) : used+len(rows)]
		r.ownCovs(covs, rows)
		optBuf[n] = ownuse.Option{Ref: int(r.Recipe), Max: r.MaxExecutions, Cov: covs}
		n++
		used += len(rows)
	}
	ref, ok := ownuse.Pick(optBuf[:n], short)
	if !ok {
		return 0, 0, false
	}
	i := v.recipeAt(RecipeID(ref))
	k, ok = ownuse.Ramp(v.Weight, v.Recipes[i].MaxExecutions, v.Cover, v.ExitTicks)
	return i, k, ok
}

// ownCovs fills dst, one entry per good of rows, with the coverage of the good
// by the recipe (spec-subsistence.md 4.1, FR-SUB-039), the sum over its
// non-discard outputs of the commodity, saturated at MaxInt64.
func (r *RecipeView) ownCovs(dst []int64, rows []UpkeepRow) {
	for j, row := range rows {
		var cov int64
		for _, o := range r.Outputs {
			if o.Commodity != row.Commodity || o.Discard {
				continue
			}
			c, _ := ownuse.Cov(o.Quantity, r.Effective, r.Duration, row.Quantity) // saturated at MaxInt64 when too large
			if sum, fits := checked.Add(cov, c); fits {
				cov = sum
			} else {
				cov = math.MaxInt64
			}
		}
		dst[j] = cov
	}
}

// recipeAt is the index into the view's Recipes of the marked recipe with the
// ID; Pick returns an ID it was given, so it is there.
func (v *AgentView) recipeAt(id RecipeID) int {
	for i := range v.Recipes {
		if v.Recipes[i].OwnUse && v.Recipes[i].Recipe == id {
			return i
		}
	}
	// Not covered by a test (testing.md 9.5): Pick returns a Ref it was given.
	panic("economysim: invariant violated: Pick returned a recipe that is not in the view")
}

// ownBlockReason is why no own-use recipe can run: the production reason of the
// first marked recipe the agent cannot start, no_recipe when every one could and
// none makes a good that is short.
func (v *AgentView) ownBlockReason() Reason {
	for i := range v.Recipes {
		if r := &v.Recipes[i]; r.OwnUse && r.MaxExecutions < 1 {
			return r.Reason
		}
	}
	return ReasonNoRecipe
}

// ownUseWork is the default Work in the own-use mode (spec-subsistence.md 5.5),
// called for an agent in the mode before the normal decision. It reports whether
// it filled the request. With no marked recipe it can run, the agent is blocked
// at P2.5 (case a) and the normal decision stands as if the agent were out of
// the mode. With one, the work share is 10 000 and the halt does not apply; when
// the retry is due the normal decision runs first and a role recipe it chooses
// wins, the own-use recipe going on otherwise.
func (dst *workReportBuf) ownUseWork(v *AgentView) bool {
	idx, k, ok := v.ownChoice()
	if !ok {
		dst.ownBlocked, dst.ownReason = true, v.ownBlockReason()
		return false
	}
	if ownuse.RetryDue(v.RetryJobs, v.OwnUseJobs) {
		dst.normal(v)
		if dst.chosen >= 0 {
			dst.ownRetried = true
			return true
		}
		*dst = workReportBuf{skipped: dst.skipped[:0], noList: dst.noList, chosen: -1, blocked: -1}
	}
	dst.share, dst.chosen, dst.executions = wholeBP, idx, k
	return true
}

// completed records a finished job of agent a in region g: when it ran an
// own-use recipe, the units it created of each non-discard output are an in-kind
// receipt of the tick, valued at P6.3 (spec-subsistence.md 7.1, FR-SUB-028, 051).
// The units are those actually created, after the carried remainder and the room
// that was left. It is the module's call in finishJob.
func (s *subsistence) completed(w *World, a *agent, cell int, c *completion) {
	row := w.roles.row(a.role)
	if !isOwn(row, c.recipe) {
		return
	}
	for i, o := range w.recipes.row(c.recipe).outputs {
		if units := c.created[i] - c.disposed[i]; !o.discard && units > 0 {
			s.receipts = append(s.receipts, ownReceipt{slot: int(a.id), commodity: o.commodity, units: units})
			if w.obs.supplied {
				s.output[cell*len(w.cfg.Commodities)+int(o.commodity)] += units
			}
		}
	}
}

// priceReceipts values the receipts of the tick at the prices of cost_t, the
// published price of the commodity in the agent's region or the effective
// reference price while it has never traded, and adds them to the welfare scratch
// (spec-subsistence.md 7.2). It runs at P6.3 right after the prices are valued and
// empties the list. A product beyond int64 is read as the largest value, as every
// sum of the record is.
func (s *subsistence) priceReceipts(w *World) {
	nc := len(w.cfg.Commodities)
	for _, r := range s.receipts {
		g := w.regionOf(&w.agents[r.slot])
		amount, ok := checked.Mul(r.units, w.oc.valued[int(g)*nc+int(r.commodity)])
		if !ok {
			amount = math.MaxInt64
		}
		w.welfare.addOwnReceipt(r.slot, amount)
	}
	s.receipts = s.receipts[:0]
}

// addOwnReceipt adds the value of the units an own-use job created to the income
// of the tick (K^own_t).
func (s *welfareScratch) addOwnReceipt(slot int, amount int64) {
	f := s.at(slot)
	f.ownReceipt = satAdd(f.ownReceipt, amount)
}

// p28OwnUse is P2.8: the module's report of the tick, before P2.8a. For each
// (region, role) cell in ascending order it advances the two runs of the Warns
// and logs them at the tenth tick of a run and every tenth after (spec-subsistence.md
// 14.2, FR-SUB-042), emits OwnUseChanged under the observer flag when any count
// is non-zero (FR-SUB-040, 041), and clears the counts. The runs are
// diagnostics and change no result.
func (w *World) p28OwnUse() {
	s := w.sub
	for cell := range s.entered {
		households := w.pr.cells[cell].households
		var warnOn, warnBlocked bool
		s.runOn[cell], warnOn = floorRun(s.runOn[cell], s.inMode[cell], households)
		s.runBlocked[cell], warnBlocked = advanceRun(s.runBlocked[cell], s.blocked[cell] > 0, false)
		if warnOn || warnBlocked {
			s.warn(w, cell, households, warnOn, warnBlocked)
		}
		if w.obs.supplied {
			s.emit(w, cell, households)
		}
	}
	clear(s.entered)
	clear(s.left)
	clear(s.started)
	clear(s.blocked)
	clear(s.retried)
	clear(s.blockedBy)
	clear(s.output)
}

// warn logs the Warns "role on own use" and "own use blocked" of a cell, which
// are bounded by regions times roles (observability-and-logging.md 3.2.2).
func (s *subsistence) warn(w *World, cell int, households int64, on, blocked bool) {
	ctx := context.Background()
	if !w.log.Enabled(ctx, slog.LevelWarn) {
		return
	}
	roles := len(w.cfg.Roles)
	region, role := slog.Int64(logkeys.Region, int64(cell/roles)), slog.Int64(logkeys.Role, int64(cell%roles))
	if on {
		w.log.LogAttrs(ctx, slog.LevelWarn, "role on own use", region, role,
			slog.Int64(logkeys.Households, households), slog.Int64(logkeys.OwnUse, s.inMode[cell]))
	}
	if blocked {
		best := 0
		for i, n := range s.blockedBy[cell] {
			if n > s.blockedBy[cell][best] {
				best = i
			}
		}
		w.log.LogAttrs(ctx, slog.LevelWarn, "own use blocked", region, role,
			slog.Int64(logkeys.Households, s.blocked[cell]), slog.String(logkeys.Reason, string(blockReasons[best])))
	}
}

// emit delivers OwnUseChanged for a cell when any of its counts is non-zero.
func (s *subsistence) emit(w *World, cell int, households int64) {
	nc, roles := len(w.cfg.Commodities), len(w.cfg.Roles)
	var out []Amount
	for c, units := range s.output[cell*nc : (cell+1)*nc] {
		if units > 0 {
			out = append(out, Amount{Commodity: CommodityID(c), Quantity: Quantity(units)})
		}
	}
	if s.entered[cell]|s.left[cell]|s.started[cell]|s.blocked[cell]|s.retried[cell] == 0 && len(out) == 0 {
		return
	}
	w.obs.deliver(OwnUseChanged{
		Tick: w.Tick(), Region: RegionID(cell / roles), Role: RoleID(cell % roles),
		Households: households, OnOwnUse: s.inMode[cell], Entered: s.entered[cell], Left: s.left[cell],
		Started: s.started[cell], Blocked: s.blocked[cell], Retried: s.retried[cell], Output: out,
	})
}

// recordFields returns the two numbers of the agent record that the module hosts
// (spec-checkpoint.md 4.2, tag 3): the flag as 0 or 1 and the counter. They are 0
// while the module is off, which a nil module also answers.
func (s *subsistence) recordFields(id AgentID) (on, jobs uint64) {
	if s == nil || int(id) >= len(s.ownUse) {
		return 0, 0
	}
	if s.ownUse[id] {
		on = 1
	}
	return on, uint64(s.ownUseJobs[id])
}

// recordBad reports whether the flag and the counter read from an agent record are
// not a state this world can hold: any non-zero value while the module is off
// (R10), and a counter without the flag (the counter is reset to 0 whenever the
// flag drops, FR-SUB-032). A flag on a role whose covered set is empty is lawful:
// a role change keeps the flag and the next flag step drops it.
func (s *subsistence) recordBad(on, jobs uint64) bool {
	if s == nil {
		return on|jobs != 0
	}
	return on == 0 && jobs != 0
}

// restoreAgent puts the flag and the counter read from an agent record in the
// slots of the agent, growing them as the flag step does.
func (s *subsistence) restoreAgent(id AgentID, on, jobs uint64) {
	s.grow(int(id) + 1)
	s.ownUse[id], s.ownUseJobs[id] = on == 1, int64(jobs)
}

// saveSubsistence is tag 19 (spec-subsistence.md 15, spec-checkpoint.md 4.2): the two
// runs of consecutive ticks of the two Warns, each sparse by (region, role). They
// are diagnostics that change no result and are saved so that a resumed run logs
// as the original did. The flag and the counter are in the agent record (tag 3).
func (w *World) saveSubsistence(cw *codec.Writer) error {
	roles := len(w.cfg.Roles)
	writeSparse(cw, w.sub.runOn, roles)
	writeSparse(cw, w.sub.runBlocked, roles)
	return nil
}

// loadSubsistence reads tag 19: each run from 1 to the ticks elapsed, a row at 0
// being corrupt (R10). That a run belongs to a role with something to cover is
// the cross-check (subsistenceCross).
func (w *World) loadSubsistence(rs *restoreCtx, r *codec.Reader) error {
	regions, roles := len(w.cfg.Regions), len(w.cfg.Roles)
	readSparse(r, w.sub.runOn, regions, roles, rs.tick)
	readSparse(r, w.sub.runBlocked, regions, roles, rs.tick)
	return nil
}

// subsistenceCross is the module's part of the cross-checks of a restore: a run of
// either Warn needs households in the mode at P2.8 of the last tick, and the flag
// step drops every household of a role whose covered set is empty before P2.8, so
// a role with an empty covered set has no run.
func (w *World) subsistenceCross() error {
	s, roles := w.sub, len(w.cfg.Roles)
	for cell := range s.runOn {
		if len(s.covered[cell%roles]) == 0 && (s.runOn[cell] != 0 || s.runBlocked[cell] != 0) {
			return fmt.Errorf("%w: region %d, role %d has a run of own-use warnings, but its covered set is empty", ErrCheckpointCorrupt, cell/roles, cell%roles)
		}
	}
	return nil
}
