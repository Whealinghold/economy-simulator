package economysim

import (
	"cmp"
	"math"
	"math/rand/v2" // nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used -- only the *rand.Rand type of the injected, seeded source (architecture.md 7.3, 7.7)
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/occupation"
)

// This file declares the surface of the spawner (spec-spawner.md 3): the
// interface, the view it reads, the plan it appends and the default. A module
// appends the fields it owns with the module (architecture.md 2.10.5); the
// shape here is what the Stage 1 slice needs. The executor of a plan belongs to
// the world.

// Spawner decides who leaves, who enters and who changes role: plain data in,
// plain data out. The world calls Decide once per tick for each region in
// region order, in the entry-and-exit phase, and checks every decision it
// appends (spec-spawner.md 4).
//
// The world's random source rng is a parameter. An implementation MUST NOT
// create its own random source, MUST NOT use a global one, and MUST NOT keep
// the rng it is given after the call returns: a spawner that does breaks the
// reproducibility of a run (architecture.md 7.4, 7.7). Decide is a function of
// its view and the random source alone: it MUST NOT read the clock, a global, a
// map's iteration order or the network, MUST NOT write to the view, MUST NOT
// keep the view or a slice in it, and MUST NOT block, because Step waits for
// it. A panic in a spawner is not recovered and propagates to the caller of
// Step.
//
// An implementation that keeps state between calls MUST also implement the
// optional pair of the checkpoint (Stateful, from Stage 2 of the roadmap), so
// that a checkpoint can save it and a restore load it. SaveState MUST write
// every piece of state that can influence a result, MUST NOT change the
// spawner and MUST give equal bytes for equal states. LoadState MUST replace
// the whole state, MUST validate its input and MUST NOT keep the slice it is
// given. An instance that holds state MUST NOT be shared between worlds.
type Spawner interface {
	// Decide appends to d, in the order they are to be executed, the decisions
	// for the region of v.
	Decide(v *RegionView, rng *rand.Rand, d *Decisions)
}

// RegionView is what a spawner reads for one region and tick: a copy valid for
// the call, with plain integers only (spec-spawner.md 3.2). Roles are in
// canonical order and agents in ascending ID, with no pseudo-agent. The tunables
// are those of spec-spawner.md 11, in basis points or ticks as they are there.
type RegionView struct {
	// Tick is the tick being stepped.
	Tick int64
	// Region is the region.
	Region RegionID
	// TicksPerYear is the ticks in a year of the calendar.
	TicksPerYear int64
	// PoolMoney is the money of the region's estate account.
	PoolMoney Money
	// Budget is PoolMoney * EntryPoolShare / 10 000 at view time: advisory, a
	// ceiling a spawner may use to size its plan. The world takes the budget
	// itself (spec-spawner.md 7.3).
	Budget Money
	// ReviewTicks, EntryShare, RoleHysteresis, RoleCooldown, EntryRatio,
	// ChangePrivation, RemovePrivation, DeathStep, ExitShare, Renewal and
	// EntryPoolShare are the tunables of World (spec-spawner.md 11).
	ReviewTicks, EntryShare, RoleHysteresis, RoleCooldown, EntryRatio int64
	ChangePrivation, RemovePrivation, DeathStep, ExitShare            int64
	Renewal, EntryPoolShare                                           int64
	// Roles is the per-role group, in canonical role order.
	Roles []RoleView
	// Agents is the per-agent group, in ascending ID order.
	Agents []RegionAgent

	// observed is set once by the world when an observer is supplied; false
	// leaves the Hysteresis count at 0 (it costs a comparison per reviewer).
	observed bool
	// scratch is the default spawner's working memory, set once per world; a
	// view built without one makes Decide allocate its own for the call.
	scratch *decideScratch
}

// RoleView is one role in a region as a spawner sees it.
type RoleView struct {
	// Role is the role.
	Role RoleID
	// Closed is true when the role is closed to entry (Role.Closed).
	Closed bool
	// Households is the sum of the weights of the role's agents.
	Households int64
	// Agents is the number of the role's agents.
	Agents int64
	// Wage is the observed wage per working household-tick; it is meaningful
	// only when HasWage is true.
	Wage Money
	// HasWage is whether the role has an observed wage.
	HasWage bool
	// Upkeep is the cost of one household's effective upkeep per tick at the
	// region's prices.
	Upkeep Money
	// ExpectedWorkTicks is the ticks a household of the role is expected to
	// work in a year.
	ExpectedWorkTicks int64
	// StartMoney is the money an entrant starts with.
	StartMoney Money
	// ToolPrice is the market price of the tools the role's recipes require, 0
	// if none.
	ToolPrice Money
	// TrainingTicks is the ticks of training the role needs; a role above 0 is
	// not entered by the default spawner.
	TrainingTicks int64
	// CapitalReturn is the years in which a tool must pay for itself.
	CapitalReturn int64
	// RenewalCarry is the world's carry of the renewal rule
	// (spec-spawner.md 7.1), read by Decide.
	RenewalCarry int64
}

// RegionAgent is one agent in a region as a spawner sees it. The name is not
// AgentView, which is the strategy's view.
type RegionAgent struct {
	// Agent is the agent.
	Agent AgentID
	// Role is the agent's role.
	Role RoleID
	// Weight is the households the agent stands for.
	Weight int64
	// External is true for an external agent; Mortal is true for one the
	// world may remove (spec-spawner.md 9).
	External, Mortal bool
	// Money is the agent's money.
	Money Money
	// Unfed is the households of the agent that went unfed this tick.
	Unfed int64
	// Hunger is the agent's hunger.
	Hunger int64
	// Privation is the agent's privation, set by the consumption step.
	Privation int64
	// RoleSince is the tick the agent took its role.
	RoleSince int64
	// Upkeep is one household's upkeep per tick.
	Upkeep Money
}

// DecisionKind is the kind of a decision of a spawner.
type DecisionKind int

// The kinds of decision. A module appends its own (DecideApply, by the skills
// module).
const (
	// DecideRemove removes every household of an agent.
	DecideRemove DecisionKind = iota
	// DecideReduce lowers an agent's weight by Households.
	DecideReduce
	// DecideSwitch changes an agent's role.
	DecideSwitch
	// DecideEnter adds Households to a role in the region.
	DecideEnter
)

// Decision is one step of a plan.
type Decision struct {
	// Kind is the kind of decision.
	Kind DecisionKind
	// Agent is the agent of a Remove, Reduce or Switch.
	Agent AgentID
	// Role is the role of a Switch or an Enter.
	Role RoleID
	// Households is the households of a Reduce or an Enter, at least 1.
	Households int64
	// Cause is carried to the event.
	Cause Cause
}

// Decisions is the plan a spawner appends to, in the order to be executed. The
// unexported counts are the default's own dampers, which the world merges into
// EntryLimited; a consumer's spawner leaves them 0.
type Decisions struct {
	// List is the decisions, in execution order.
	List []Decision

	regionCap, roleCap, cooldown, hysteresis, exitCap int64
	noViableRole, stranded                            int64
	noViableByRole                                    []int64
}

// reset empties the plan and zeroes the counts, keeping the backing arrays. The
// world calls it before each region's Decide.
func (d *Decisions) reset() {
	d.List = d.List[:0]
	d.regionCap, d.roleCap, d.cooldown, d.hysteresis, d.exitCap = 0, 0, 0, 0, 0
	d.noViableRole, d.stranded = 0, 0
	clear(d.noViableByRole)
}

// DefaultSpawner is the built-in spawner: the failure ladder, the entry rule
// and renewal of spec-spawner.md 5 to 7. It has no fields, keeps no state and
// draws no random word.
type DefaultSpawner struct{}

var _ Spawner = DefaultSpawner{}

// Decide appends the default's plan for v to d, in the order of
// spec-spawner.md 10.2: rung-3 reductions and removals, then the switches of
// rung 1 and of the review, then the renewal entries. It draws nothing and
// reads nothing but v. The counts of d are added to, so the world resets d
// before each region.
func (DefaultSpawner) Decide(v *RegionView, _ *rand.Rand, d *Decisions) {
	s := v.scratch
	if s == nil {
		s = new(decideScratch)
	}
	c := decider{v: v, d: d, s: s}
	c.prepare()
	c.exits()
	c.switches()
	c.renewals()
}

// roleSignal is the signal of one role: E, V, whether E exists and whether the
// role is viable at EntryRatio.
type roleSignal struct {
	e, v        int64
	has, viable bool
}

// decideScratch holds the slices Decide works in, so a call allocates nothing
// once they have grown to the region.
type decideScratch struct {
	sig        []roleSignal
	rank       []int
	leavers    []occupation.Leaver
	exits      []occupation.Exit
	byID       []int
	acted      []bool
	removed    []int64
	lfl        []int64
	cands      []occupation.Candidate
	dests      []occupation.Dest
	adm        []occupation.Admission
	dc         []occupation.DestCount
	used       []int64
	roleCap    []int64
	regionUsed int64
}

// decider is one call of the default's Decide.
type decider struct {
	v          *RegionView
	d          *Decisions
	s          *decideScratch
	households int64
	regionCap  int64
}

// roleSignals returns E and V of a role and whether it is viable at entryRatio
// (V >= entryRatio). A role has no signal without an observed wage, or without
// an upkeep or a year to divide by (spec-spawner.md 6.1.1); hasE is false then
// and the role is never viable.
func roleSignals(r *RoleView, ticksPerYear, entryRatio int64) (e, v int64, hasE, viable bool) {
	if !r.HasWage {
		return 0, 0, false, false
	}
	e, hasE = occupation.Expected(int64(r.Wage), r.ExpectedWorkTicks, int64(r.Upkeep), ticksPerYear)
	if !hasE {
		return 0, 0, false, false
	}
	v, _ = occupation.Viability(int64(r.Wage), int64(r.Upkeep))
	return e, v, true, v >= entryRatio
}

// satAllowance is the allowance of a count of households, saturating where the
// product leaves int64.
func satAllowance(households, shareBP int64) int64 {
	n, ok := occupation.Allowance(households, shareBP)
	if !ok {
		return math.MaxInt64
	}
	return n
}

// regrow returns s with length n and every element zero, allocating only when
// its capacity is short.
func regrow[T any](s []T, n int) []T {
	s = slices.Grow(s[:0], n)[:n]
	clear(s)
	return s
}

// prepare sizes the scratch, computes the signals and the rank of the open
// roles, and the caps of the region and of each role.
func (c *decider) prepare() {
	v, s := c.v, c.s
	nr, na := len(v.Roles), len(v.Agents)
	s.sig = regrow(s.sig, nr)
	s.acted = regrow(s.acted, na)
	s.removed = regrow(s.removed, nr)
	s.lfl = regrow(s.lfl, nr)
	s.used = regrow(s.used, nr)
	s.roleCap = regrow(s.roleCap, nr)
	c.d.noViableByRole = regrow(c.d.noViableByRole, nr)
	s.regionUsed = 0
	s.rank = s.rank[:0]
	s.dests = slices.Grow(s.dests[:0], na*nr+nr)
	c.households = 0
	for i := range v.Roles {
		r := &v.Roles[i]
		sg := &s.sig[i]
		sg.e, sg.v, sg.has, sg.viable = roleSignals(r, v.TicksPerYear, v.EntryRatio)
		c.households = satAdd(c.households, r.Households)
		s.roleCap[i] = satAllowance(r.Households, v.EntryShare)
		if c.open(i) && sg.has {
			s.rank = append(s.rank, i)
		}
	}
	c.regionCap = satAllowance(c.households, v.EntryShare)
	slices.SortFunc(s.rank, func(a, b int) int {
		return cmp.Or(cmp.Compare(s.sig[b].e, s.sig[a].e), cmp.Compare(v.Roles[a].Role, v.Roles[b].Role), cmp.Compare(a, b))
	})
}

// open is whether the default may enter role i: not closed and with no
// training (a traineeship is the skills module's, spec-spawner.md 6.2).
func (c *decider) open(i int) bool {
	r := &c.v.Roles[i]
	return !r.Closed && r.TrainingTicks <= 0
}

// roleIdx is the index in Roles of the role id, or -1.
func (c *decider) roleIdx(id RoleID) int {
	for i := range c.v.Roles {
		if c.v.Roles[i].Role == id {
			return i
		}
	}
	return -1
}

// exits appends the rung-3 reductions and removals (spec-spawner.md 5.3) and
// remembers the agents acted on and the households removed per role.
func (c *decider) exits() {
	v, s := c.v, c.s
	if v.ExitShare <= 0 {
		return
	}
	s.leavers = s.leavers[:0]
	for i := range v.Agents {
		a := &v.Agents[i]
		if !a.External && a.Unfed > 0 && occupation.Rung3(a.Privation, v.Tick, a.RoleSince, v.RemovePrivation) {
			s.leavers = append(s.leavers, occupation.Leaver{
				Ref: int(a.Agent), Privation: a.Privation, Weight: a.Weight,
				Unfed: a.Unfed, Money: int64(a.Money), Upkeep: int64(a.Upkeep),
			})
		}
	}
	var cut int64
	s.exits, cut = occupation.Exits(s.exits, s.leavers, satAllowance(c.households, v.ExitShare), v.DeathStep)
	c.d.exitCap = satAdd(c.d.exitCap, cut)
	if len(s.exits) == 0 {
		return
	}
	s.byID = s.byID[:0]
	for i := range v.Agents {
		s.byID = append(s.byID, i)
	}
	slices.SortFunc(s.byID, func(a, b int) int { return cmp.Compare(v.Agents[a].Agent, v.Agents[b].Agent) })
	for _, e := range s.exits {
		c.emitExit(e)
	}
}

// emitExit appends one reduction or removal and books it.
func (c *decider) emitExit(e occupation.Exit) {
	v, s := c.v, c.s
	pos, _ := slices.BinarySearchFunc(s.byID, e.Ref, func(i, ref int) int { return cmp.Compare(int(v.Agents[i].Agent), ref) })
	i := s.byID[pos]
	s.acted[i] = true
	cause, kind := CauseStarved, DecideReduce
	if e.Insolvent {
		cause = CauseInsolvent
	}
	if e.Remove {
		kind = DecideRemove
	}
	c.d.List = append(c.d.List, Decision{Kind: kind, Agent: v.Agents[i].Agent, Households: e.Households, Cause: cause})
	if ri := c.roleIdx(v.Agents[i].Role); ri >= 0 {
		s.removed[ri] = satAdd(s.removed[ri], e.Households)
	}
}

// switches admits the rung-1 and review candidates under the caps and appends
// their Switch decisions (spec-spawner.md 6.2 to 6.4).
func (c *decider) switches() {
	v, s := c.v, c.s
	s.cands = s.cands[:0]
	for i := range v.Agents {
		c.consider(i)
	}
	if len(s.cands) == 0 {
		return
	}
	c.admit()
	for _, a := range s.adm {
		ag := &v.Agents[a.Ref]
		cause := CauseReview
		if ag.Privation >= v.ChangePrivation {
			cause = CauseDistress
		}
		c.d.List = append(c.d.List, Decision{Kind: DecideSwitch, Agent: ag.Agent, Role: v.Roles[a.Admitted[0].Dest].Role, Cause: cause})
	}
}

// admit runs Admit on the candidates and adds its counts.
func (c *decider) admit() {
	s := c.s
	var n occupation.Counts
	s.adm, s.dc, n = occupation.Admit(s.adm, s.dc, s.cands, occupation.Limits{RegionCap: c.regionCap, RoleCap: s.roleCap}, &s.regionUsed, s.used)
	c.d.regionCap = satAdd(c.d.regionCap, n.RegionCap)
	c.d.roleCap = satAdd(c.d.roleCap, n.RoleCap)
}

// eligible reports whether agent i is a candidate this tick and the index of its
// role: not external, not acted on, at rung 1 or due for a review, and past its
// cooldown (an agent held by the cooldown is counted in Cooldown).
func (c *decider) eligible(i int) (ri int, ok bool) {
	v := c.v
	a := &v.Agents[i]
	if a.External || c.s.acted[i] || a.Weight <= 0 {
		return 0, false
	}
	if a.Privation < v.ChangePrivation && !occupation.ReviewDue(int64(a.Agent), v.Tick, v.ReviewTicks) {
		return 0, false
	}
	if ri = c.roleIdx(a.Role); ri < 0 {
		return 0, false
	}
	if !occupation.PastCooldown(v.Tick, a.RoleSince, v.RoleCooldown) {
		c.d.cooldown = satAdd(c.d.cooldown, a.Weight)
		return 0, false
	}
	return ri, true
}

// consider makes agent i a candidate if it is eligible and has a destination.
// An agent at rung 1 is judged by the rung-1 rule only (spec-spawner.md 6.4).
func (c *decider) consider(i int) {
	v, s := c.v, c.s
	ri, ok := c.eligible(i)
	if !ok {
		return
	}
	a := &v.Agents[i]
	var eo int64
	if s.sig[ri].has {
		eo = s.sig[ri].e
	}
	rung1 := a.Privation >= v.ChangePrivation
	start := len(s.dests)
	gap := c.destinations(a, ri, eo, rung1)
	if len(s.dests) == start {
		if rung1 {
			c.d.stranded = satAdd(c.d.stranded, a.Weight)
		}
		return
	}
	cand := occupation.Candidate{
		Kind: occupation.CandReview, Ref: i, Gap: gap, Seq: int64(a.Agent), Households: a.Weight,
		Dests: s.dests[start:len(s.dests):len(s.dests)],
	}
	if rung1 {
		cand.Kind, cand.Privation = occupation.CandRung1, a.Privation
	}
	s.cands = append(s.cands, cand)
}

// wants is whether a destination of signal sg is good enough for an agent whose
// own E is eo: rung 1 needs a viable role that beats its own, a review the
// hysteresis band and no viability (spec-spawner.md 6.2, 6.4).
func (c *decider) wants(sg roleSignal, eo int64, rung1 bool) bool {
	if rung1 {
		return sg.viable && sg.e > eo
	}
	return occupation.Beats(sg.e, eo, c.v.RoleHysteresis)
}

// destinations appends the roles the agent may move to, best first, and
// returns the gap E_d - E_o of the best. When an observer is attached it also
// counts a reviewer whose best candidate role fails only the band.
func (c *decider) destinations(a *RegionAgent, ri int, eo int64, rung1 bool) (gap int64) {
	s := c.s
	start := len(s.dests)
	first := true
	for _, d := range s.rank {
		if d == ri || !c.affordable(a, d, eo) {
			continue
		}
		sg := s.sig[d]
		ok := c.wants(sg, eo, rung1)
		if first && !ok && !rung1 {
			c.noteHysteresis(a, sg.e, eo)
		}
		first = false
		if !ok {
			continue
		}
		if len(s.dests) == start {
			gap = gapOf(sg.e, eo)
		}
		s.dests = append(s.dests, occupation.Dest{Role: d, Qualifies: true})
	}
	return gap
}

// noteHysteresis counts the households of a reviewer whose best candidate role
// is better than its own but fails the band, when an observer is attached.
func (c *decider) noteHysteresis(a *RegionAgent, eD, eo int64) {
	if c.v.observed && eD > eo {
		c.d.hysteresis = satAdd(c.d.hysteresis, a.Weight)
	}
}

// affordable is the tool half of the capital test (spec-skill-barriers.md 2.5):
// the agent's money covers ToolPrice and the tool pays back in CapitalReturn.
func (c *decider) affordable(a *RegionAgent, d int, eo int64) bool {
	r := &c.v.Roles[d]
	if r.ToolPrice > 0 && a.Money < r.ToolPrice {
		return false
	}
	return occupation.CapitalOK(c.s.sig[d].e, eo, int64(r.ToolPrice), r.CapitalReturn, int64(a.Upkeep), c.v.TicksPerYear)
}

// gapOf is e - eo, saturating by sign where it leaves int64.
func gapOf(e, eo int64) int64 {
	g, ok := checked.Sub(e, eo)
	switch {
	case ok:
		return g
	case e > eo:
		return math.MaxInt64
	}
	return math.MinInt64
}

// renewals appends the replacement entries (spec-spawner.md 7): per role in
// role order, the like-for-like entry and then the entries of its fallback
// households, which are one batch for the caps.
func (c *decider) renewals() {
	v, s := c.v, c.s
	s.cands = s.cands[:0]
	s.dests = s.dests[:0]
	for _, d := range s.rank {
		if s.sig[d].viable {
			s.dests = append(s.dests, occupation.Dest{Role: d, Qualifies: true})
		}
	}
	for r := range v.Roles {
		c.size(r)
	}
	if len(s.cands) > 0 {
		c.admit()
	} else {
		s.adm = s.adm[:0]
	}
	p := 0
	for r := range v.Roles {
		if n := s.lfl[r]; n > 0 {
			c.enter(r, n)
		}
		for ; p < len(s.adm) && s.adm[p].Ref == r; p++ {
			for _, dc := range s.adm[p].Admitted {
				c.enter(dc.Dest, dc.Households)
			}
		}
	}
}

// size sizes the replacement of role r from the view's carry: like for like if
// the role is open and viable, else a batch for the best viable roles, else
// dropped and counted.
func (c *decider) size(r int) {
	v, s := c.v, c.s
	n, _, ok := occupation.Renewal(v.Roles[r].RenewalCarry, s.removed[r], v.Renewal)
	switch {
	case !ok || n <= 0:
	case c.open(r) && s.sig[r].viable:
		s.lfl[r] = n
	case len(s.dests) == 0:
		c.d.noViableRole = satAdd(c.d.noViableRole, n)
		c.d.noViableByRole[r] = satAdd(c.d.noViableByRole[r], n)
	default:
		s.cands = append(s.cands, occupation.Candidate{Kind: occupation.CandReplace, Ref: r, Seq: int64(r), Households: n, Dests: s.dests})
	}
}

// enter appends the renewal entry of n households into role index r.
func (c *decider) enter(r int, n int64) {
	c.d.List = append(c.d.List, Decision{Kind: DecideEnter, Role: c.v.Roles[r].Role, Households: n, Cause: CauseRenewal})
}

// refusalReason maps a refusal of internal/occupation to its Reason. It is
// total over the refusals of the slice; RefusalNone has no reason and gives "".
func refusalReason(r occupation.Refusal) Reason {
	switch r {
	case occupation.UnknownAgent:
		return ReasonUnknownAgent
	case occupation.UnknownRole:
		return ReasonUnknownRole
	case occupation.ExternalAgent:
		return ReasonExternalAgent
	case occupation.SameRole:
		return ReasonSameRole
	case occupation.BadHouseholds:
		return ReasonBadHouseholds
	case occupation.UnsupportedCause:
		return ReasonUnsupportedCause
	}
	return ""
}

// saveSpawner is tag 9 (spec-checkpoint.md 4.2; plan 2-2 D250): the sparse renewal
// carry of each (region, role), the (region, role) pairs that ever had an agent
// (only the set bits), the sparse run of ticks with unfunded entrants of each
// region, and the sparse runs of empty ticks and of ticks below the floor of each
// (region, role).
func (w *World) saveSpawner(cw *codec.Writer) error {
	roles := len(w.cfg.Roles)
	writeSparse(cw, w.state.renewalCarry, roles)
	had := 0
	for _, b := range w.state.hadAgent {
		if b {
			had++
		}
	}
	cw.Count(had)
	for i, b := range w.state.hadAgent {
		if b {
			cw.U(uint64(i / roles))
			cw.U(uint64(i % roles))
		}
	}
	writeSparseByRegion(cw, w.state.unfundedRun)
	writeSparse(cw, w.state.extinctRun, roles)
	writeSparse(cw, w.state.belowFloorRun, roles)
	return nil
}
