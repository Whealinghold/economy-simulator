package economysim

import (
	"fmt"
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/household"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file is the agent (spec-agents.md 3, 5; plan-phase-1-7.md Phase 4): the
// type, its creation and the creation draws, the initial endowment, the
// belief reseed, the upkeep in total quantities and the consumption step of
// one agent. The loops over agents, the removal and the role and weight
// changes are the world row's and compose the pieces below. Nothing here logs,
// emits an event or draws a random number beyond the two creation words.

// residueBound is the exclusive upper bound of the residue draw of a new agent,
// copied to every residue (spec-production.md 6.6.2, FR-PRO-026). It is a rule
// constant of the draw, not a tunable.
const residueBound int64 = 10_000

// agent is one household or a cohort of identical households (spec-agents.md
// 3.1). It holds no money and no quantity: those are in its ledger account, and
// its region is the account's (FR-AGT-002). The per-agent state of a module is
// in that module's slices by agent slot (architecture.md 2.10.1), not here.
type agent struct {
	// id is the agent's ID, never reused (FR-AGT-001).
	id AgentID
	// account is the agent's ledger account; the region lives on the account.
	account int
	// role is the agent's role.
	role RoleID
	// weight is the households this agent stands for, at least 1.
	weight int64
	// external says the agent was added through the integration interface.
	external bool
	// mortal says an external agent may be removed by the library's rules.
	mortal bool
	// strategy is the index into the world's strategy table (spec-agents.md 6.6).
	strategy int
	// beliefs holds one price belief per commodity of the role's tradable set,
	// in canonical order (spec-beliefs.md 3.1).
	beliefs []beliefs.Belief
	// job is the job in progress; hasJob says whether there is one
	// (spec-production.md 5.1).
	job job
	// residues is the output residues of the role's recipes (spec-production.md 6.6).
	residues []int64
	// history is the welfare ring (spec-welfare-and-wages.md 13).
	history history
	// target is the target ratio drawn at creation, in basis points (spec-welfare-and-wages.md 5.1).
	target int64
	// unfed is the households not fully fed in the last consumption phase (5.4).
	unfed int64
	// hunger is the consecutive ticks with unfed above 0 (5.4.1).
	hunger int64
	// privation is the leaky count of unfed ticks the failure ladder reads (5.4.1).
	privation int64
	// roleSince is the tick the agent took up its present role or place
	// (spec-spawner.md 5.2.2a).
	roleSince int64
	// directive is the pending directive of a directed agent, set between steps
	// (spec-agents.md 10.3); nil when there is none.
	directive *Directive
	// live is the directive of the tick: P1.9 copies the mailbox into it
	// (spec-integration.md 4.2.3) and the view reads it. It is scratch, derived at
	// P1.9 from the mailbox, so it is not saved.
	live *Directive
}

// Directive is what a consumer asks a directed agent to do in the next tick
// (spec-agents.md 10.3): plain data, kept in the agent's mailbox. The credit
// fields are appended with the credit module.
type Directive struct {
	// Job is used by Work; nil means the directive has no job and the
	// fallback's Work decides. A non-nil request with NoJob set requests
	// nothing.
	Job *JobRequest
	// Intents are used by Trade: each replaces the fallback's intent for the
	// same commodity.
	Intents []Intent
	// Discards are used by Trade: they are added to the plan.
	Discards []Discard
	// Hold lists the commodities the fallback must not trade. It removes only
	// the fallback's intent: the directive's own intent for a held commodity
	// stays.
	Hold []CommodityID
	// Sticky keeps the directive after the tick it applies to.
	Sticky bool
}

// hasJob reports whether the agent has a job in progress.
func (a *agent) hasJob() bool { return a.job.executions > 0 && a.job.remaining > 0 }

// agentInit is what newAgent needs besides the role row (N-243: few parameters).
type agentInit struct {
	id       AgentID
	account  int
	role     RoleID
	weight   int64
	strategy int
	external bool
	mortal   bool
	// tick is the creation tick, the agent's roleSince.
	tick int64
}

// newAgent builds an agent from a role row (spec-agents.md 3.5). It takes no
// amount, opens no account and draws no random number: it is called for every
// way of creating an agent (FR-AGT-005), and the caller funds the account,
// seeds the beliefs (reseedBeliefs) and draws the creation words
// (drawCreation). A weight below 1 is a programmer error and panics.
func newAgent(p agentInit, row *roleRow, rt *recipeTable) agent {
	if p.weight < 1 {
		panic(fmt.Sprintf("economysim: invariant violated: agent %d has weight %d, want at least 1", p.id, p.weight))
	}
	return agent{
		id: p.id, account: p.account, role: p.role, weight: p.weight,
		external: p.external, mortal: p.mortal, strategy: p.strategy,
		beliefs:   make([]beliefs.Belief, 0, len(row.tradable)),
		residues:  make([]int64, residueLen(row.jobs, rt)),
		history:   newHistory(row),
		roleSince: p.tick,
	}
}

// drawCreation takes the two creation words of an agent, in the order of
// spec-agents.md 3.5.2: the target ratio (newTarget, one word whatever the
// spread) and then the residue value, copied to every residue (FR-PRO-026). It
// takes nothing else (FR-AGT-006). The caller has keyed s to the agent's
// creation stream.
func drawCreation(s *stream, a *agent, targetRatio, spread int64) {
	a.target = newTarget(s, targetRatio, spread)
	setResidues(a.residues, s.draw(residueBound))
}

// beliefSeed is one commodity of the tradable set to seed a belief for
// (spec-beliefs.md 8.2, 8.3): the caller fills Ref, UnitCost, Mean, Empty and
// Price, and reseedBeliefs sets Kept and Belief.
type beliefSeed = beliefs.Seed

// reseedBeliefs replaces the agent's beliefs by one per commodity of row's
// tradable set, in canonical order (spec-beliefs.md 8.3, the one helper). src
// holds one seed per commodity of row.tradable, in the same order, and is
// scratch: its Kept and Belief are set here. A commodity in the tradable sets of
// both roles keeps its stored belief (re-clamped to the new bounds); oldRow is
// nil at a creation, when every belief is new. It takes no random word and
// reuses the agent's slice.
func reseedBeliefs(a *agent, row, oldRow *roleRow, src []beliefSeed, p beliefs.Params) {
	n := len(row.tradable)
	if len(src) < n {
		panic("economysim: invariant violated: fewer belief seeds than commodities in the tradable set")
	}
	src = src[:n]
	for i, c := range row.tradable {
		s := &src[i]
		s.Kept, s.Belief = false, beliefs.Belief{}
		if oldRow == nil || !oldRow.isTradable(c) {
			continue
		}
		j, ok := slices.BinarySearch(oldRow.tradable, c)
		if !ok || j >= len(a.beliefs) {
			panic("economysim: invariant violated: agent beliefs do not match the old role's tradable set")
		}
		s.Kept, s.Belief = true, a.beliefs[j]
	}
	// src holds every kept belief by value, so Reseed may append over the slice it replaces.
	a.beliefs = beliefs.Reseed(a.beliefs[:0], src, p)
}

// defaultEndowment returns the money and the goods an initial agent of the
// weight receives (spec-agents.md 3.5.1): weight times StartMoney and weight
// times each StartStock quantity, with checked arithmetic. ok is false on
// overflow, which validation rules out, so the world's path treats it as an
// invariant violation. The role is a validated one.
func defaultEndowment(cfg *Config, r *Role, weight int64) (Money, []itemRow, bool) {
	money, ok := checked.Mul(weight, int64(r.StartMoney))
	if !ok {
		return 0, nil, false
	}
	stock := make([]itemRow, 0, len(r.StartStock))
	for _, it := range r.StartStock {
		q, ok := checked.Mul(weight, int64(it.Quantity))
		if !ok {
			return 0, nil, false
		}
		stock = append(stock, itemRow{commodity: roleCommodity(cfg, r.ID, it.Commodity), quantity: Quantity(q)})
	}
	return Money(money), stock, true
}

// fundInitial gives an account its initial money and goods from the boundary
// accounts of the reason initial (spec-agents.md 3.5.1, FR-AGT-005), with
// unstated lot keys. It is the uniform endowment of an initial agent; a later
// agent is funded from the pool (FR-AGT-007), not here. An amount of 0 is
// skipped, since the ledger refuses an amount below 1.
func fundInitial(l *ledger.Ledger, account int, money Money, stock []itemRow) error {
	if money >= 1 {
		from, ok := l.Boundary(ledger.Initial, ledger.Money)
		if !ok {
			return fmt.Errorf("fund account %d: no initial boundary account for money", account)
		}
		if err := l.Transfer(from, account, int64(money), ledger.Initial); err != nil {
			return fmt.Errorf("fund account %d with %d money: %w", account, money, err)
		}
	}
	for _, it := range stock {
		if it.quantity < 1 {
			continue
		}
		from, ok := l.Boundary(ledger.Initial, int(it.commodity))
		if !ok {
			return fmt.Errorf("fund account %d: no initial boundary account for commodity %d", account, it.commodity)
		}
		if err := l.Move(from, account, int(it.commodity), int64(it.quantity), ledger.Initial); err != nil {
			return fmt.Errorf("fund account %d with %d units of commodity %d: %w", account, it.quantity, it.commodity, err)
		}
	}
	return nil
}

// upkeepTotals appends to dst[:0] the effective upkeep of an agent of the
// weight: each per-household quantity times the weight, in the order of per
// (canonical already), with checked arithmetic (spec-agents.md 5.2, FR-AGT-011).
// It is the one place the agent's upkeep is multiplied by the weight. ok is
// false on overflow, which validation rules out (plan-phase-1-7.md D96).
func upkeepTotals(dst []itemRow, per []itemRow, weight int64) ([]itemRow, bool) {
	dst = dst[:0]
	for _, it := range per {
		t, ok := checked.Mul(weight, int64(it.quantity))
		if !ok {
			return dst, false
		}
		dst = append(dst, itemRow{commodity: it.commodity, quantity: Quantity(t)})
	}
	return dst, true
}

// consumeScratch is the scratch of the consumption step. After consume, totals
// holds the upkeep rows in total quantities and short the units of each row
// that were not held (total less consumed), parallel to the rows, for the
// "upkeep unmet" Debug line of the world. The zero value works; it grows once.
type consumeScratch struct {
	totals   []itemRow
	short    []int64
	consumed []int64
	perHH    []int64
}

// consume is the consumption step of one agent (spec-agents.md 5.3, 5.4, 5.4.1;
// FR-AGT-013, 014, 015, 075). For each upkeep row in canonical order it moves
// the lesser of what the account from holds and the row's total to the
// consumption boundary, oldest lots first; then it sets unfed, hunger and
// privation from the quantities consumed. from is the agent's own account, or a
// trainer's for a trainee (FR-AGT-068). per is the upkeep of one household in
// the agent's region (roleRow.upkeepIn). A shortfall is not an error. The
// error returned is a ledger refusal, which a move of at most the holding
// cannot cause, so it is an invariant violation the world fails the step on.
func consume(l *ledger.Ledger, a *agent, from int, per []itemRow, sc *consumeScratch) error {
	totals, ok := upkeepTotals(sc.totals, per, a.weight)
	sc.totals = totals
	if !ok {
		panic("economysim: invariant violated: upkeep times weight overflows after validation")
	}
	sc.short, sc.consumed, sc.perHH = sc.short[:0], sc.consumed[:0], sc.perHH[:0]
	for i, it := range totals {
		c := int(it.commodity)
		held, _, err := l.Holding(from, c)
		if err != nil {
			return fmt.Errorf("consume: holding of commodity %d in account %d: %w", c, from, err)
		}
		eaten := min(held, int64(it.quantity))
		if eaten >= 1 {
			sink, ok := l.Boundary(ledger.Consumption, c)
			if !ok {
				return fmt.Errorf("consume: no consumption boundary account for commodity %d", c)
			}
			// Untested: eaten is at most the holding just read, so the ledger cannot
			// refuse the move; the check keeps an invariant break from being silent.
			if err := l.Move(from, sink, c, eaten, ledger.Consumption); err != nil {
				return fmt.Errorf("consume: %d units of commodity %d from account %d: %w", eaten, c, from, err)
			}
		}
		sc.consumed = append(sc.consumed, eaten)
		sc.perHH = append(sc.perHH, int64(per[i].quantity))
		sc.short = append(sc.short, int64(it.quantity)-eaten)
	}
	a.unfed = a.weight - household.Fed(sc.consumed, sc.perHH, a.weight)
	a.hunger = household.NextHunger(a.hunger, a.unfed)
	a.privation = household.NextPrivation(a.privation, a.unfed)
	return nil
}

// The bits of the flag word of an agent record (spec-checkpoint.md 4.2.4).
const (
	flagExternal uint64 = 1 << iota
	flagMortal
)

// saveAgents is tag 3 (spec-checkpoint.md 4.2; plan 2-2 D247): the table of the
// strategy names the live agents use, sorted, a count of agents, each live agent
// in ascending ID, the blocks of the modules that follow the list, and the sparse
// run of unfed ticks. A removed agent is a zero slot and writes nothing.
func (w *World) saveAgents(cw *codec.Writer) error {
	var names []string
	w.eachLive(func(a *agent) { names = append(names, w.strategyName(a)) })
	slices.Sort(names)
	names = slices.Compact(names)
	cw.Count(len(names))
	for _, n := range names {
		cw.Bytes([]byte(n))
	}
	cw.Count(int(w.live.agents))
	w.eachLive(func(a *agent) { w.saveAgent(cw, a, names) })
	// Excluded from coverage (testing.md 9.5): agentTails is empty until the storage
	// and institutions modules append to it, and their change tests the loop.
	for _, t := range agentTails {
		if t.present(&w.cfg) {
			t.save(w, cw)
		}
	}
	writeSparse(cw, w.state.unfedRun, len(w.cfg.Roles))
	return nil
}

// saveAgent writes the record of one agent. The fields a module hosts are always
// written (spec-checkpoint.md R10, plan 2-2 D247): OwnUse and OwnUseJobs as 0 and
// the journey as absent while their modules are off, DeathCarry as its start; the
// module that builds the field replaces the value and the module-on check here and
// moves nothing.
func (w *World) saveAgent(cw *codec.Writer, a *agent, names []string) {
	var flags uint64
	if a.external {
		flags |= flagExternal
	}
	if a.mortal {
		flags |= flagMortal
	}
	idx, _ := slices.BinarySearch(names, w.strategyName(a))
	cw.U(uint64(a.id))
	cw.U(uint64(a.account))
	cw.U(uint64(a.role))
	cw.U(unsigned(a.weight))
	cw.U(flags)
	cw.U(uint64(idx))
	cw.U(unsigned(a.unfed))
	cw.U(unsigned(a.hunger))
	cw.U(unsigned(a.privation))
	cw.I(a.roleSince)
	cw.I(w.deathCarry(a.id))
	on, jobs := w.sub.recordFields(a.id)
	cw.U(on) // OwnUse: 0 while the subsistence module is off
	cw.U(jobs)
	to, arrives, journey := w.migration.journeyOf(a.id)
	cw.B(journey)
	if journey {
		cw.U(uint64(to))
		cw.U(unsigned(arrives))
	}
	saveDirective(cw, a.directive)
}

// saveDirective writes the directive of an agent: whether there is one and, when
// there is, whether it carries a job request (Job is a pointer) and the request,
// the intents, the discards, the holds and Sticky. A negative amount a consumer
// put in a directive is written as the unsigned number it converts to and refused
// by the restore, which also holds the record to the rules Direct holds it to
// (checkLoadedDirective).
func saveDirective(cw *codec.Writer, d *Directive) {
	cw.B(d != nil)
	if d == nil {
		return
	}
	cw.B(d.Job != nil)
	if d.Job != nil {
		cw.B(d.Job.NoJob)
		if d.Job.NoJob {
			// The other fields are ignored with NoJob and may hold any value, which the
			// restore would refuse; write the neutral ones.
			cw.U(0)
			cw.U(0)
		} else {
			cw.U(uint64(d.Job.Recipe))
			cw.U(uint64(d.Job.Executions))
		}
	}
	cw.Count(len(d.Intents))
	for _, in := range d.Intents {
		cw.U(uint64(in.Commodity))
		cw.U(uint64(in.Side))
		cw.U(uint64(in.Quantity))
		cw.U(uint64(in.Limit))
		cw.U(uint64(in.Target))
		cw.U(uint64(in.Reserve))
		cw.U(uint64(in.UnitCost))
		cw.U(uint64(in.Purpose))
	}
	cw.Count(len(d.Discards))
	for _, ds := range d.Discards {
		cw.U(uint64(ds.Commodity))
		cw.U(uint64(ds.Quantity))
	}
	cw.Count(len(d.Hold))
	for _, c := range d.Hold {
		cw.U(uint64(c))
	}
	cw.B(d.Sticky)
}

// agentRecordMin is the fewest bytes the record of an agent takes: fifteen fields
// of at least one byte each, which bounds the count of agents by the bytes left
// (R8).
const agentRecordMin = 15

// loadAgents is tag 3 (plan 2-2 D247): the table of strategy names, the agents in
// strictly ascending ID, the blocks of the modules that follow the list and the
// sparse run of unfed ticks. The agent table is allocated here, once the count of
// slots the ledger can account for is known (spec-checkpoint.md 6.6.1): every
// agent ever created has an account, open or closed, so the next ID cannot exceed
// the regions plus the accounts.
func (w *World) loadAgents(rs *restoreCtx, r *codec.Reader) error {
	regions, roles := len(w.cfg.Regions), len(w.cfg.Roles)
	names := readStrategyNames(r)
	if r.Err() == nil && int(w.state.nextAgent)-w.firstHousehold > w.ledger.Len() {
		r.Failf("next agent ID %d, but the ledger has %d accounts", w.state.nextAgent, w.ledger.Len())
	}
	if r.Err() != nil {
		return nil
	}
	w.agents = make([]agent, w.state.nextAgent)
	prev := uint64(0)
	for i := range r.Count(agentRecordMin) {
		if r.Err() != nil {
			return nil
		}
		prev = w.loadAgent(rs, r, names, prev, i > 0)
	}
	w.rebuildExternals()
	// Every name in the table is a live agent's strategy (the table is the sorted
	// set saveAgents writes), so an unused name is a non-canonical encoding (R10).
	if r.Err() == nil {
		used := map[string]struct{}{}
		w.eachLive(func(a *agent) { used[w.strategyName(a)] = struct{}{} })
		if len(used) != len(names) {
			r.Failf("the table of strategy names has %d names, the agents use %d", len(names), len(used))
		}
	}
	for _, t := range agentTails {
		if t.present(&w.cfg) {
			t.load(w, r)
		}
	}
	readSparse(r, w.state.unfedRun, regions, roles, rs.tick)
	return nil
}

// readStrategyNames reads the table of strategy names that heads tag 3: a count and
// the names in strictly ascending order.
func readStrategyNames(r *codec.Reader) []string {
	names := make([]string, 0, 1)
	for range r.Count(1) {
		n := string(r.Bytes())
		if k := len(names) - 1; k >= 0 && r.Err() == nil && names[k] >= n {
			r.Failf("strategy name %q is not above the one before it", n)
		}
		names = append(names, n)
	}
	return names
}

// agentRec is the record of one agent as read, before any field is checked against
// the world: the fields in the order of spec-checkpoint.md 4.2 tag 3. nameIdx is
// the index into the table of strategy names.
type agentRec struct {
	id                       uint64
	acct                     int
	role                     uint64
	weight, flags, nameIdx   int64
	unfed, hunger, privation int64
	since, carry             int64
	ownUse, ownUseJobs       uint64
	journey                  bool
	jto                      uint64
	jarrives                 int64
	directive                *Directive
}

// readAgentRec reads the 15 fields of an agent record, each against its own bound
// (R6, R13). ownUse is 0 or 1 and ownUseJobs at most the ticks elapsed, a start a
// tick being the most the counter can have counted.
func (w *World) readAgentRec(rs *restoreCtx, r *codec.Reader, names []string, prev uint64, started bool) agentRec {
	rec := agentRec{id: r.Strict(prev, started)}
	rec.acct = int(r.Index(uint64(w.ledger.Len())))
	rec.role = r.Index(uint64(len(w.cfg.Roles)))
	rec.weight = r.Bounded("weight", MaxWeight)
	rec.flags = int64(r.Index((flagExternal | flagMortal) + 1))
	rec.nameIdx = int64(r.Index(uint64(len(names))))
	rec.unfed = r.Bounded("unfed", rec.weight)
	rec.hunger = r.Bounded("hunger", rs.tick)
	rec.privation = r.Bounded("privation", rs.tick)
	rec.since, rec.carry = r.I(), r.I()
	rec.ownUse, rec.ownUseJobs, rec.journey = r.Index(2), uint64(r.Bounded("own-use jobs", rs.tick)), r.B()
	if rec.journey {
		rec.jto = r.Index(uint64(len(w.cfg.Regions)))
		rec.jarrives = r.Bounded("arrival", math.MaxInt64)
	}
	rec.directive = w.loadDirective(r)
	return rec
}

// checkAgentRec requires the record to describe an agent this world can hold. It
// returns the agent's account.
func (w *World) checkAgentRec(rs *restoreCtx, rec *agentRec, names []string) (ledger.View, error) {
	v, err := w.ledger.Account(rec.acct)
	switch {
	case rec.id < uint64(w.firstHousehold) || rec.id >= uint64(w.state.nextAgent):
		err = fmt.Errorf("agent ID %d outside %d to %d", rec.id, w.firstHousehold, w.state.nextAgent)
	case err != nil || v.Kind != ledger.KindAgent || v.Closed || v.Owner != int(rec.id):
		err = fmt.Errorf("agent %d names account %d, which is not its open Agent account", rec.id, rec.acct)
	case rec.journey && w.badJourney(rs, rec, v.Region):
		err = fmt.Errorf("agent %d has a journey to region %d arriving at %d, which its account in region %d cannot have", rec.id, rec.jto, rec.jarrives, v.Region)
	default:
		err = w.checkAgentFields(rs, rec, names)
	}
	return v, err
}

// checkAgentFields requires a weight, the strategy of the agent's role, a
// role-since within the run, the death carry that is its start and none of the
// state of a module that is off.
func (w *World) checkAgentFields(rs *restoreCtx, rec *agentRec, names []string) error {
	switch id := AgentID(rec.id); {
	case rec.weight < 1 || w.badStrategy(rec, names[rec.nameIdx]):
		return fmt.Errorf("agent %d has weight %d, flags %d or a strategy %q that is not its role's or an external agent's", rec.id, rec.weight, rec.flags, names[rec.nameIdx])
	case rec.since < 0 || rec.since > rs.tick:
		return fmt.Errorf("agent %d has RoleSince %d outside 0 to %d", rec.id, rec.since, rs.tick)
	case w.badDeathCarry(id, rec.carry):
		return fmt.Errorf("agent %d has DeathCarry %d, which is not its start while the population module is off and outside 0 to D with it on", rec.id, rec.carry)
	case w.sub.recordBad(rec.ownUse, rec.ownUseJobs) || (rec.journey && (w.migration == nil || w.cfg.Roles[rec.role].Fixed)):
		return fmt.Errorf("agent %d has own-use or journey state its module does not allow", rec.id)
	}
	return nil
}

// badStrategy reports an agent record whose flags or strategy cannot be one: a
// household has its role's strategy and neither flag; an external agent (the flag
// word has External, and Mortal only with it) has a strategy that resolves, "default",
// "directed" or a registered name (spec-checkpoint.md 4.2.4, spec-integration.md 4.8).
func (w *World) badStrategy(rec *agentRec, name string) bool {
	external, mortal := rec.flags&int64(flagExternal) != 0, rec.flags&int64(flagMortal) != 0
	if !external {
		return mortal || name != w.cfg.Roles[rec.role].Strategy
	}
	_, ok := w.strategyIndex(name)
	return !ok
}

// checkLoadedDirective holds a restored mailbox to the rules Direct holds a set one
// to (spec-integration.md 4.3.2): a mailbox is never empty, and its job, intents,
// discards and holds name what the agent's role and weight allow.
func (w *World) checkLoadedDirective(rec *agentRec) error {
	d := rec.directive
	if d == nil {
		return nil
	}
	if d.mailbox() == nil {
		return fmt.Errorf("agent %d has an empty directive", rec.id)
	}
	if err := w.checkDirective(&agent{role: RoleID(rec.role), weight: rec.weight}, d); err != nil {
		return fmt.Errorf("agent %d has an invalid directive: %w", rec.id, err)
	}
	return nil
}

// loadAgent reads the record of one agent, checks it and puts the agent in its
// slot. It returns the agent's ID, the key the next record must exceed. The fields
// a module hosts are read always (R10): while the module is off, OwnUse and
// OwnUseJobs must be 0, there is no journey (and none for a Fixed role), and DeathCarry must be its start.
func (w *World) loadAgent(rs *restoreCtx, r *codec.Reader, names []string, prev uint64, started bool) uint64 {
	rec := w.readAgentRec(rs, r, names, prev, started)
	if r.Err() != nil {
		return rec.id
	}
	v, err := w.checkAgentRec(rs, &rec, names)
	if err != nil {
		r.Failf("%v", err)
		return rec.id
	}
	if err := w.checkLoadedDirective(&rec); err != nil {
		r.Failf("%v", err)
		return rec.id
	}
	init := agentInit{
		id: AgentID(rec.id), account: rec.acct, role: RoleID(rec.role), weight: rec.weight, strategy: int(rec.role), tick: rec.since,
		external: rec.flags&int64(flagExternal) != 0, mortal: rec.flags&int64(flagMortal) != 0,
	}
	if init.external {
		init.strategy, _ = w.strategyIndex(names[rec.nameIdx])
	}
	a := newAgent(init, w.roles.row(RoleID(rec.role)), &w.recipes)
	a.unfed, a.hunger, a.privation, a.directive = rec.unfed, rec.hunger, rec.privation, rec.directive
	w.agents[rec.id] = a
	if rec.journey {
		w.migration.depart(a.id, RegionID(rec.jto), rec.jarrives)
	}
	if w.sub != nil {
		w.sub.restoreAgent(a.id, rec.ownUse, rec.ownUseJobs)
	}
	if w.pop != nil {
		w.pop.restoreCarry(a.id, rec.carry)
	}
	w.live.agents++
	w.live.households += rec.weight
	w.live.inRole[v.Region*len(w.cfg.Roles)+int(rec.role)]++
	return rec.id
}

// loadDirective reads saveDirective's bytes: whether there is a directive, its
// optional job request, intents, discards, holds and Sticky. Every amount is
// checked against its bound (R13), which also refuses the number a negative one
// was written as; the rules of Direct are checked after the record is read.
func (w *World) loadDirective(r *codec.Reader) *Directive {
	if !r.B() {
		return nil
	}
	nc, nr := uint64(len(w.cfg.Commodities)), uint64(len(w.cfg.Recipes))
	d := &Directive{}
	if r.B() {
		d.Job = &JobRequest{NoJob: r.B()}
		if d.Job.NoJob { // the other fields are ignored and written as 0, even in a world with no recipe
			r.Bounded("recipe", 0)
			r.Bounded("executions", 0)
		} else {
			d.Job.Recipe, d.Job.Executions = RecipeID(r.Index(nr)), r.Bounded("executions", math.MaxInt64)
		}
	}
	for range r.Count(8) {
		if r.Err() != nil {
			return nil
		}
		d.Intents = append(d.Intents, Intent{
			Commodity: CommodityID(r.Index(nc)), Side: Side(r.Index(2)),
			Quantity: Quantity(r.Bounded("quantity", MaxQuantity)), Limit: Price(r.Bounded("limit", MaxPrice)),
			Target: Quantity(r.Bounded("target", MaxQuantity)), Reserve: Quantity(r.Bounded("reserve", MaxQuantity)),
			UnitCost: Price(r.Bounded("unit cost", MaxPrice)), Purpose: Purpose(r.Index(uint64(PurposeTool) + 1)),
		})
	}
	for range r.Count(2) {
		if r.Err() != nil {
			return nil
		}
		d.Discards = append(d.Discards, Discard{Commodity: CommodityID(r.Index(nc)), Quantity: Quantity(r.Bounded("quantity", MaxQuantity))})
	}
	for range r.Count(1) {
		if r.Err() != nil {
			return nil
		}
		d.Hold = append(d.Hold, CommodityID(r.Index(nc)))
	}
	d.Sticky = r.B()
	return d
}

// AgentState is the state of one agent at the tick boundary (spec-integration.md
// 4.7.2): a function of the saved state alone, a deep copy of plain data with no
// pointer, function or map. It is not the AgentView of a strategy: it omits what
// only a position of the next tick can say (the maximum executions and the refusal
// reason of a recipe, the stock after the next completions). The groups of a module
// that is off are at their zero value, and a later module appends its group.
type AgentState struct {
	// Identity.
	ID        AgentID
	Region    RegionID
	Role      RoleID
	Weight    int64
	External  bool
	Mortal    bool
	Strategy  string
	RoleSince int64

	// Standing. Money is the available balance (no reservation exists). Ratio and
	// EarnedRatio are the welfare ratio and the ratio without relief, each valid when
	// its Has flag is set.
	Money                    Money
	Hunger, Unfed, Privation int64
	Ratio, EarnedRatio       int64
	HasRatio, HasEarned      bool
	TargetRatio              int64
	WorkingTicks, Window     int64

	// Holdings are the commodities held, ascending.
	Holdings []HoldingState

	// Job: whether one is active, its recipe, the ticks it has left, its
	// executions, and the output it will create (Pending, by commodity).
	HasJob                bool
	Recipe                RecipeID
	Remaining, Executions int64
	Pending               []Amount

	// Beliefs and Markets hold one entry per commodity of the role's tradable set, in
	// canonical order: the agent's belief with the bounds of its range, and the
	// market report of its region.
	Beliefs []BeliefState
	Markets []MarketState

	// Upkeep is the effective upkeep of one household in the agent's region.
	Upkeep []UpkeepRow

	// HasDirective says Directive is the mailbox: the directive pending, if any.
	HasDirective bool
	Directive    Directive

	// Rumours are the rumours of the last view built for the agent and RumoursTick
	// the tick of that view, or none and -1 (spec-integration.md 4.7.3).
	Rumours     []Rumour
	RumoursTick int64

	// Wants are the carries of the agent's ladder, in rank order (demand module).
	Wants []WantCarry
	// OwnUse, OwnUseJobs, Cover and Secure are the own-use group (subsistence module).
	OwnUse        bool
	OwnUseJobs    int64
	Cover, Secure int64
	// Away says the agent is away on a trip (transport module); HasJourney, JourneyTo
	// and JourneyArrives are its journey (migration module).
	Away           bool
	HasJourney     bool
	JourneyTo      RegionID
	JourneyArrives int64
}

// HoldingState is a commodity an agent holds: its units and the units of them that
// spoil at the end of the tick the next step runs.
type HoldingState struct {
	Commodity CommodityID
	Quantity  Quantity
	Expiring  Quantity
}

// BeliefState is the price belief of an agent for a commodity: the centre and the
// width, and the floor and ceiling of the range they sit in (spec-beliefs.md 3).
type BeliefState struct {
	Commodity      CommodityID
	Centre         Price
	Width          int64
	Floor, Ceiling Price
}

// WantCarry is the carry of one want of an agent's ladder, in basis points of a
// unit (spec-demand.md 4.4).
type WantCarry struct {
	Want  WantID
	Carry int64
}

// Agent returns the state of one live agent at the tick boundary, for a player's
// screen or a designer's tool (spec-integration.md 4.7). It returns ErrInStep from
// inside a step, ErrWorldFailed on a failed world, ErrUnknownAgent for an ID never
// issued or a pseudo-agent and ErrAgentRemoved for a removed agent. It draws no
// random number, emits no event and changes no saved state. It derives the cost
// anchors of the boundary as Snapshot does (shortcut: they are not saved, so it
// reads every agent once; keep them in the saved state to make this O(entity)).
func (w *World) Agent(id AgentID) (AgentState, error) {
	if err := w.guard(); err != nil {
		return AgentState{}, fmt.Errorf("Agent: %w", err)
	}
	a, err := w.queryAgent(id)
	if err != nil {
		return AgentState{}, fmt.Errorf("Agent: agent %d: %w", id, err)
	}
	if _, err := w.prepareAnchors(); err != nil {
		return AgentState{}, fmt.Errorf("Agent: %w", err)
	}
	st := AgentState{
		ID: a.id, Region: w.regionOf(a), Role: a.role, Weight: a.weight, External: a.external, Mortal: a.mortal,
		Strategy: w.strategyName(a), RoleSince: a.roleSince, RumoursTick: -1,
		Hunger: a.hunger, Unfed: a.unfed, Privation: a.privation, TargetRatio: a.target,
		WorkingTicks: a.history.WorkingTicks(), Window: w.roles.row(a.role).window,
	}
	if err := w.fillStanding(&st, a); err != nil {
		return AgentState{}, fmt.Errorf("Agent: agent %d: %w", id, err)
	}
	w.fillJob(&st, a)
	w.fillBeliefs(&st, a)
	if a.directive != nil {
		st.HasDirective, st.Directive = true, *a.directive.mailbox()
	}
	if rv, ok := w.lastView(a); ok {
		st.Rumours, st.RumoursTick = slices.Clone(rv.rumours), rv.tick
	}
	w.fillModuleGroups(&st, a)
	return st, nil
}

// queryAgent finds a live agent of any kind: the lookup of Direct without the
// external test (spec-integration.md 3.4.3).
func (w *World) queryAgent(id AgentID) (*agent, error) {
	switch {
	case id < 0 || id >= w.state.nextAgent || int(id) < w.firstHousehold:
		return nil, ErrUnknownAgent
	case w.agents[id].weight < 1:
		return nil, ErrAgentRemoved
	}
	return &w.agents[id], nil
}

// fillStanding sets the money, the welfare ratios, the holdings and the upkeep.
func (w *World) fillStanding(st *AgentState, a *agent) error {
	acct, err := w.ledger.Account(a.account)
	if err != nil {
		return fmt.Errorf("account %d: %w", a.account, err)
	}
	st.Money = Money(acct.Balance - acct.Reserved)
	items := w.roles.row(a.role).upkeepIn(st.Region)
	qty, price := make([]int64, len(items)), make([]int64, len(items))
	st.Upkeep = make([]UpkeepRow, len(items))
	for i, it := range items {
		st.Upkeep[i] = UpkeepRow{Commodity: it.commodity, Quantity: int64(it.quantity)}
		qty[i], price[i] = int64(it.quantity), int64(w.markets.reports[w.markets.index(st.Region, it.commodity)].Price)
	}
	cost, def := welfare.TickCost(qty, price, a.weight), w.cfg.Roles[a.role].DefaultRatio
	st.Ratio, st.HasRatio = a.history.Ratio(cost, def)
	st.EarnedRatio, st.HasEarned = a.history.EarnedRatio(cost, def)
	for c := range w.cfg.Commodities {
		held, _, err := w.ledger.Holding(a.account, c)
		if err != nil {
			return fmt.Errorf("stock of commodity %d: %w", c, err)
		}
		if held == 0 {
			continue
		}
		exp, err := w.ledger.Expiring(a.account, c, w.Tick())
		if err != nil {
			return fmt.Errorf("expiring units of commodity %d: %w", c, err)
		}
		st.Holdings = append(st.Holdings, HoldingState{Commodity: CommodityID(c), Quantity: Quantity(held), Expiring: Quantity(exp)})
	}
	return nil
}

// fillJob sets the job group: the recipe in progress and the output it will create.
func (w *World) fillJob(st *AgentState, a *agent) {
	if !a.hasJob() {
		return
	}
	st.HasJob, st.Recipe, st.Remaining, st.Executions = true, a.job.recipe, a.job.remaining, a.job.executions
	for i, o := range w.recipes.row(a.job.recipe).outputs {
		st.Pending = append(st.Pending, Amount{Commodity: o.commodity, Quantity: Quantity(a.job.pending[i])})
	}
}

// fillBeliefs sets the beliefs and the market reports of the tradable set, with the
// range of each belief: the reference band, or the cost band for a commodity the
// role makes.
func (w *World) fillBeliefs(st *AgentState, a *agent) {
	row := w.roles.row(a.role)
	for i, id := range row.tradable {
		var unitCost int64
		if w.roleMakes(row, id) {
			unitCost = w.prod.unitCost(st.Region, id)
		}
		bd := beliefs.Limits(int64(w.refPrice(st.Region, id)), unitCost, w.bp)
		b := a.beliefs[i]
		st.Beliefs = append(st.Beliefs, BeliefState{Commodity: id, Centre: Price(b.Centre), Width: b.Width, Floor: Price(bd.Floor), Ceiling: Price(bd.Ceiling)})
		st.Markets = append(st.Markets, w.marketState(w.markets.index(st.Region, id), st.Region, id))
	}
}

// roleMakes says a recipe of the role makes c as a non-discard output: the rule
// under which a belief's bounds use the cost anchor.
func (w *World) roleMakes(row *roleRow, c CommodityID) bool {
	for _, rid := range row.recipes {
		for _, o := range w.recipes.row(rid).outputs {
			if o.commodity == c && !o.discard {
				return true
			}
		}
	}
	return false
}

// fillModuleGroups sets the groups of the modules that are on: the carries of the
// wants, the own-use figures, and the trip and the journey.
func (w *World) fillModuleGroups(st *AgentState, a *agent) {
	if d := w.demand; d != nil && int(a.id) < len(d.carry) {
		for i, rg := range d.ladder[a.role] {
			st.Wants = append(st.Wants, WantCarry{Want: rg.want, Carry: d.carry[a.id][i]})
		}
	}
	if s := w.sub; s != nil {
		st.OwnUse, st.OwnUseJobs = s.agentFields(int(a.id))
		if goods, ok := s.goodsOf(w, a, st.Region); ok && len(goods) > 0 {
			st.Cover, st.Secure = s.figures(goods, a.hunger > 0, int64(st.Money))
		}
	}
	st.Away = w.Away(a.id)
	if m := w.migration; m != nil && m.on(a.id) {
		st.HasJourney, st.JourneyTo, st.JourneyArrives = true, m.journey[a.id].to, m.journey[a.id].arrives
	}
}
