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
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/moneyflow"
)

// This file is the money supply module (spec-money-supply.md; architecture.md
// 2.10.1): the module's canonical form, its validator and counts, its warnings,
// its state, its step at P1.7, the audit, the removal of a carry, the scale
// setter, the snapshot rows and the checkpoint section of tag 13. Its arithmetic
// is internal/moneyflow.

// The target kinds that are recognised but not built (plan 2-6 D320): a flow
// naming one is ErrModuleDependency until its module is built.
const (
	targetPool     = "pool"
	targetStores   = "stores"
	targetTreasury = "treasury"
)

// pairTolerance is the tolerance of the warning pair_unbalanced, in basis points
// of the larger yearly total (spec-money-supply.md 12.2). It is a reporting rule
// constant, not a tunable.
const pairTolerance = 1_000

// reservedReasons is every reason name of the registry of spec-ledger.md 7.2,
// whether or not its module is built: a flow ID equal to one is
// ErrReasonCollision (spec-money-supply.md 4.5.2, plan 2-6 D325). A later module
// that declares a reason adds its name here, and the test of the reasons fails
// until it does.
var reservedReasons = []string{
	"trade", "initial", "recipe_input", "recipe_output", "disposal",
	"consumption", "want_consumption", "spoilage", "wear", "estate_money",
	"estate_goods", "start_money", "relief", "outside_budget",
	"outside_unspent", "outside_receipts", "outside_offer", "outside_unsold",
	"outside_purchases", "escrow_in", "escrow_out", "escrow_refund",
	"wage_escrow", "wage_pay", "wage_refund", "loan_out", "loan_repay",
	"loan_interest", "rent", "title_sale", "premium", "payout", "transit_out",
	"transit_in", "haul_use", "transit_use", "loss", "split", "merge", "births",
	"starvation", "deaths",
}

// canonical returns the module in canonical order with the period spelled: flows
// by ID, and an empty Per as PerTick, so the effective configuration and the hash
// do not depend on the spelling (plan 2-6 D326, NFR-MON-003). It returns nil for
// a module that is off and copies what it changes.
func (m *MoneySupply) canonical() *MoneySupply {
	if m == nil {
		return nil
	}
	out := &MoneySupply{Flows: sortedBy(m.Flows, func(a, b MoneyFlow) int { return cmp.Compare(a.ID, b.ID) })}
	for i := range out.Flows {
		if out.Flows[i].Per == "" {
			out.Flows[i].Per = PerTick
		}
	}
	return out
}

// moneyCounts is the count bound of phase 1 for the module: more than
// MaxMoneyFlows flows is ErrTooMany (spec-money-supply.md 12.1, FR-MON-034).
func moneyCounts(m *MoneySupply, over func(n, bound int, p ...string)) {
	if m != nil {
		over(len(m.Flows), MaxMoneyFlows, "modules", "money_supply", "flows")
	}
}

// validateMoneySupply is the module's part of phases 2, 3 and 5 of
// spec-config.md 7.1.6 (spec-money-supply.md 12.1): the identity of each flow,
// the bounds of its fields, its shape and its target. It runs after phase 1
// found no count over a bound, and reports nothing when the module is off.
func (v *validator) validateMoneySupply() {
	m := v.c.Modules.MoneySupply
	if m == nil {
		return
	}
	seen := make(map[string]bool, len(m.Flows))
	for _, f := range m.Flows {
		base := []string{"modules", "money_supply", "flows", f.ID}
		switch {
		case !validID(f.ID):
			v.rep.add(ErrInvalidID, path(base, "id"), f.ID, "")
		case slices.Contains(reservedReasons, f.ID):
			v.rep.add(ErrReasonCollision, path(base, "id"), f.ID, "")
		}
		if seen[f.ID] {
			v.rep.add(ErrDuplicateID, path(base, "id"), f.ID, "")
		}
		seen[f.ID] = true
		v.flowFields(base, f)
		v.flowShape(base, f)
		v.flowTarget(base, f)
	}
}

// flowFields checks the bounds of the integer fields and the closed sets of the
// strings of one flow.
func (v *validator) flowFields(base []string, f MoneyFlow) {
	v.ranges(base, []fieldRange{
		{"amount", f.Amount, 0, MaxFlowAmount},
		{"level", f.Level, 0, MaxFlowAmount},
		{"rate_bp", f.RateBP, 0, MaxFlowRateBP},
		{"exempt", f.Exempt, 0, MaxFlowAmount},
		{"start", f.Start, 0, math.MaxInt64 - 1},
		{"limit", f.Limit, 0, MaxFlowTotal},
	})
	if f.Until != 0 && f.Until <= f.Start {
		v.rep.add(ErrOutOfBounds, path(base, "until"), i64(f.Until), i64(f.Start+1))
	}
	if f.Kind != FlowGrant && f.Kind != FlowLevy {
		v.rep.add(ErrOutOfBounds, path(base, "kind"), f.Kind, "")
	}
	if f.Per != "" && f.Per != PerTick && f.Per != PerYear {
		v.rep.add(ErrOutOfBounds, path(base, "per"), f.Per, "")
	}
	v.label(base, "label", f.Label)
}

// flowShape checks that the flow is exactly one of the four shapes: a rate grant
// (Amount), a top-up (Level), a fixed levy (Amount) or a rate levy (RateBP), and
// that no field of another shape is set (spec-money-supply.md 4.1.1). A zero
// field is absent.
func (v *validator) flowShape(base []string, f MoneyFlow) {
	bad := func(field string) { v.rep.add(ErrFlowShape, path(base, field), "", "") }
	switch f.Kind {
	case FlowGrant:
		grantShape(f, bad)
	case FlowLevy:
		levyShape(f, bad)
	}
}

// grantShape reports the fields of a grant that break its shape.
func grantShape(f MoneyFlow, bad func(field string)) {
	if f.RateBP != 0 {
		bad("rate_bp")
	}
	if f.Exempt != 0 {
		bad("exempt")
	}
	if (f.Amount != 0) == (f.Level != 0) {
		bad("amount")
	}
	if f.Level != 0 && f.Per == PerYear {
		bad("per")
	}
	if f.NotIncome && f.Target.Kind != TargetHouseholds {
		bad("not_income")
	}
}

// levyShape reports the fields of a levy that break its shape.
func levyShape(f MoneyFlow, bad func(field string)) {
	if f.Level != 0 {
		bad("level")
	}
	if (f.Amount != 0) == (f.RateBP != 0) {
		bad("amount")
	}
	if f.NotIncome {
		bad("not_income")
	}
}

// flowTarget checks the target of one flow: the kind, and for households the
// role and region, which must be defined. A kind that is recognised and not
// built is ErrModuleDependency (plan 2-6 D320).
func (v *validator) flowTarget(base []string, f MoneyFlow) {
	t := f.Target
	tp := path(base, "target")
	switch t.Kind {
	case TargetHouseholds:
		switch {
		case t.Role == "":
			v.rep.add(ErrFlowShape, path(tp, "role"), "", "")
		case !v.role[t.Role]:
			v.rep.add(ErrUnknownRole, path(tp, "role"), t.Role, "")
		}
		if t.Region != "" && !v.region[t.Region] {
			v.rep.add(ErrUnknownRegion, path(tp, "region"), t.Region, "")
		}
	case targetPool, targetStores, targetTreasury:
		v.rep.add(ErrModuleDependency, path(tp, "kind"), t.Kind, "")
	default:
		v.rep.add(ErrOutOfBounds, path(tp, "kind"), t.Kind, "")
	}
}

// moneyGrantsIncome reports whether a grant to the households of the role counts
// as income: a flow of kind grant to the role's households, not NotIncome. It is
// the exemption of class_without_income (spec-demand.md 10.2, FR-DEM-033).
func moneyGrantsIncome(c *Config, ro *Role) bool {
	m := c.Modules.MoneySupply
	if m == nil {
		return false
	}
	return slices.ContainsFunc(m.Flows, func(f MoneyFlow) bool {
		return f.Kind == FlowGrant && f.Target.Kind == TargetHouseholds && f.Target.Role == ro.ID && !f.NotIncome
	})
}

// moneyWarnings returns the five warnings of the module that this build can
// raise (spec-money-supply.md 12.2, plan 2-6 D338), or nil when it is off. Like
// warnings it reads a configuration as given, skips what is not defined and
// never fails.
func moneyWarnings(c *Config, lk lookup) []Warning {
	m := c.Modules.MoneySupply
	if m == nil {
		return nil
	}
	var out []Warning
	add := func(rule string, segs ...string) { out = append(out, Warning{Path: segs, Rule: rule}) }
	if len(m.Flows) == 0 {
		add("no_flows", "modules", "money_supply")
		return out
	}
	hasLevy := slices.ContainsFunc(m.Flows, func(f MoneyFlow) bool { return f.Kind == FlowLevy })
	for _, f := range m.Flows {
		flowWarnings(c, lk, f, hasLevy, add)
	}
	if pairUnbalanced(c, m) {
		add("pair_unbalanced", "modules", "money_supply")
	}
	return out
}

// flowWarnings raises the warnings of one flow: faucet_without_sink,
// class_empty and levy_below_upkeep.
func flowWarnings(c *Config, lk lookup, f MoneyFlow, hasLevy bool, add func(string, ...string)) {
	p := []string{"modules", "money_supply", "flows", f.ID}
	if f.Kind == FlowGrant && f.Limit == 0 && !hasLevy {
		add("faucet_without_sink", p...)
	}
	if f.Target.Kind != TargetHouseholds {
		return
	}
	if !roleHasGroup(c, f.Target) {
		add("class_empty", p...)
	}
	if f.Kind == FlowLevy && levyBelowUpkeep(c, lk, f) {
		add("levy_below_upkeep", p...)
	}
}

// targets reports whether a group is in the class of a household target.
func (t FlowTarget) targets(g Group) bool {
	return g.Role == t.Role && (t.Region == "" || g.Region == t.Region)
}

// roleHasGroup reports whether the class of a household target has households
// in any targeted region at construction.
func roleHasGroup(c *Config, t FlowTarget) bool {
	return slices.ContainsFunc(c.Groups, t.targets)
}

// levyBelowUpkeep is the condition of levy_below_upkeep: the Exempt of a levy is
// below one tick's upkeep cost of the targeted role at the reference prices of a
// region where the class has households (spec-money-supply.md 9.4).
func levyBelowUpkeep(c *Config, lk lookup, f MoneyFlow) bool {
	ro := lk.role[f.Target.Role]
	if ro == nil {
		return false
	}
	for _, g := range c.Groups {
		rg := lk.region[g.Region]
		if rg == nil || !f.Target.targets(g) {
			continue
		}
		if upkeep, ok := listCost(lk, g.Region, effectiveUpkeep(c, ro), rg.ExtraUpkeep); ok && f.Exempt < upkeep {
			return true
		}
	}
	return false
}

// pairUnbalanced is the condition of pair_unbalanced: there is a rate grant to
// households and a fixed levy on households, and their nominal yearly totals
// over the census differ by more than pairTolerance basis points of the larger
// (spec-money-supply.md 9.2). Rate levies and top-ups are not compared.
func pairUnbalanced(c *Config, m *MoneySupply) bool {
	grant, levy := yearSum{}, yearSum{}
	for _, f := range m.Flows {
		switch {
		case f.Target.Kind != TargetHouseholds || f.Amount == 0:
		case f.Kind == FlowGrant:
			grant.add(c, f)
		case f.Kind == FlowLevy:
			levy.add(c, f)
		}
	}
	return grant.usable() && levy.usable() && beyondTolerance(grant.total, levy.total)
}

// beyondTolerance reports whether two totals differ by more than pairTolerance
// basis points of the larger.
func beyondTolerance(a, b int64) bool {
	if a < 0 || b < 0 {
		return false
	}
	larger := max(a, b)
	lhs, lok := checked.Mul(larger-min(a, b), wholeBP)
	rhs, rok := checked.Mul(larger, pairTolerance)
	return lok && rok && lhs > rhs
}

// yearSum is the running yearly total of the fixed flows of one kind, and whether
// there was one and whether every sum fitted.
type yearSum struct {
	total    int64
	has, bad bool
}

// usable reports whether the sum has a flow and every sum fitted.
func (y yearSum) usable() bool { return y.has && !y.bad }

// add adds the yearly total of f; a sum that does not fit makes the pair not
// comparable.
func (y *yearSum) add(c *Config, f MoneyFlow) {
	year, yok := yearlyTotal(c, f)
	sum, sok := checked.Add(y.total, year)
	y.total, y.has = sum, true
	y.bad = y.bad || !yok || !sok
}

// yearlyTotal is the nominal amount a fixed flow moves in a year over the census
// of its class: the households of the targeted groups times Amount, times the
// ticks of a year for a period of a tick. The warnings run on a configuration
// that may not validate, so a negative field, a sum that leaves int64 and a
// calendar with no year are not comparable (false), never a wrapped total.
func yearlyTotal(c *Config, f MoneyFlow) (int64, bool) {
	var households int64
	ok := f.Amount >= 0
	for _, g := range c.Groups {
		if f.Target.targets(g) {
			var hok bool
			households, hok = checked.Add(households, g.Households)
			ok = ok && hok && g.Households >= 0
		}
	}
	total, mok := checked.Mul(households, f.Amount)
	if f.Per == PerYear {
		return total, ok && mok
	}
	ticks := c.Calendar.TicksPerYear()
	total, tok := checked.Mul(total, ticks)
	return total, ok && mok && tok && ticks > 0
}

// carryUnset marks a recipient a flow has not visited yet: its carry starts at
// the first visit, and the start value is a function of the account ID alone, so
// a lazy start equals a start at creation (plan 2-6 D329).
const carryUnset = -1

// flowState is the state of one flow and what the step derives from its
// configuration once: the ledger reason and boundary account, the IDs of the
// targeted role and region, the period and the effective limit.
type flowState struct {
	flow MoneyFlow
	// reason and boundary are the flow's ledger reason and money boundary account.
	reason   ledger.Reason
	boundary int
	// role is the targeted role and region the targeted region, or -1 for every
	// region.
	role   RoleID
	region int
	// period is the ticks of the flow's period (spec-money-supply.md 4.2) and
	// limit its effective limit: the smaller of Limit and MaxFlowTotal.
	period, limit int64
	// total, scale and ended are saved: what the flow applied, its scale in
	// basis points and 0 not ended, 1 ended at its limit, 2 at the cap.
	total, scale int64
	ended        uint8
	// endedNow says the flow ended in this tick, until its event is reported.
	endedNow bool
	// carry is the carry of each recipient by AgentID, carryUnset until the first
	// visit; it is kept for an agent that leaves the class and dropped with the
	// agent (spec-money-supply.md 4.3.1, 14).
	carry []int64
}

// flowAgg is the per-tick aggregate of one flow in one region: the amounts and
// the households cut by the room of the account (short for a levy, headroom for
// a grant) and by the limit (spec-money-supply.md 13.1).
type flowAgg struct {
	due, applied, room, limit int64
}

// moneyFlows is the state of the module, held by World.money and nil when it is
// off. The scratch (cells, agg, on) is reused every tick and never saved.
type moneyFlows struct {
	flows          []flowState
	regions, roles int
	// cells lists, per (region, role), the AgentIDs of the live agents in
	// ascending order: the class scan of plan 2-6 D330.
	cells [][]AgentID
	// agg is the aggregate of flow i in region g at i*regions+g; on says which
	// flows ran in this tick.
	agg []flowAgg
	on  []bool
}

// flowClass is the ledger class of the reason of a flow: a grant is a faucet and
// a levy a sink (spec-money-supply.md 4.5.1).
func flowClass(f MoneyFlow) ledger.Class {
	if f.Kind == FlowLevy {
		return ledger.Sink
	}
	return ledger.Faucet
}

// flowClasses returns the class of each flow in ID order for the ledger's
// Config.Flows, or nil with the module off.
func flowClasses(c *Config) []ledger.Class {
	if c.Modules.MoneySupply == nil {
		return nil
	}
	out := make([]ledger.Class, len(c.Modules.MoneySupply.Flows))
	for i, f := range c.Modules.MoneySupply.Flows {
		out[i] = flowClass(f)
	}
	return out
}

// moneyFlowList returns the configured flows in ID order, or nil with the module
// off: the world's name and class table for the dynamic reasons.
func (w *World) moneyFlowList() []MoneyFlow {
	if w.cfg.Modules.MoneySupply == nil {
		return nil
	}
	return w.cfg.Modules.MoneySupply.Flows
}

// newMoneyFlows builds the state of the module for a canonical configuration
// whose ledger was built by newLedger, or returns nil when it is off. A flow whose
// role, region or boundary does not resolve is a programmer error after
// validation and panics.
func newMoneyFlows(c *Config, l *ledger.Ledger) *moneyFlows {
	ms := c.Modules.MoneySupply
	if ms == nil {
		return nil
	}
	m := &moneyFlows{
		flows:   make([]flowState, len(ms.Flows)),
		regions: len(c.Regions), roles: len(c.Roles),
		cells: make([][]AgentID, len(c.Regions)*len(c.Roles)),
		agg:   make([]flowAgg, len(ms.Flows)*len(c.Regions)),
		on:    make([]bool, len(ms.Flows)),
	}
	for i, f := range ms.Flows {
		m.flows[i] = newFlowState(c, l, i, f)
	}
	return m
}

// newFlowState resolves flow i of the configuration.
func newFlowState(c *Config, l *ledger.Ledger, i int, f MoneyFlow) flowState {
	reason := ledger.FlowReason(i)
	boundary, ok := l.Boundary(reason, ledger.Money)
	role := slices.IndexFunc(c.Roles, func(r Role) bool { return r.ID == f.Target.Role })
	region := -1
	if f.Target.Region != "" {
		region = slices.IndexFunc(c.Regions, func(r Region) bool { return r.ID == f.Target.Region })
	}
	if !ok || role < 0 || (f.Target.Region != "" && region < 0) {
		invariant("a money flow's boundary, role or region does not resolve after validation: " + f.ID)
	}
	st := flowState{
		flow: f, reason: reason, boundary: boundary, role: RoleID(role), region: region,
		period: 1, limit: MaxFlowTotal, scale: wholeBP,
	}
	if f.Per == PerYear {
		st.period = c.Calendar.TicksPerYear()
	}
	if f.Limit > 0 {
		st.limit = min(f.Limit, MaxFlowTotal)
	}
	return st
}

// active reports whether the flow runs in the tick: inside its window, not ended
// and not paused at scale 0 (FR-MON-010, 045).
func (f *flowState) active(tick int64) bool {
	return f.ended == 0 && f.scale != 0 && tick >= f.flow.Start && (f.flow.Until == 0 || tick < f.flow.Until)
}

// hasCarry reports whether the flow keeps a carry per recipient: a rate grant, a
// fixed levy and a rate levy do, a top-up does not.
func (f *flowState) hasCarry() bool { return f.flow.Kind == FlowLevy || f.flow.Level == 0 }

// carryDen is the bound of a carry: P*10^4 for a rate grant and a fixed levy,
// P*10^8 for a rate levy.
func (f *flowState) carryDen() int64 {
	den := f.period * wholeBP
	if f.flow.Kind == FlowLevy && f.flow.RateBP != 0 {
		den *= wholeBP
	}
	return den
}

// fit grows the carry table to n agents, the new ones unstarted.
func (f *flowState) fit(n int) {
	for len(f.carry) < n {
		f.carry = append(f.carry, carryUnset)
	}
}

// p17Flows is P1.7 (spec-world.md 6, spec-money-supply.md 5): the grants then the
// levies, each flow in ascending ID, its regions in order and its class in
// ascending account ID, then the events of the tick. Nothing is drawn. A ledger
// refusal of an amount checked against the balance, the headroom and the limit is
// a violated invariant (plan 2-6 D331).
func (w *World) p17Flows() {
	m := w.money
	if m == nil {
		return
	}
	tick, n := w.Tick(), 0
	for i := range m.flows {
		m.on[i] = m.flows[i].active(tick)
		if m.on[i] {
			n++
		}
	}
	if n == 0 {
		return
	}
	m.scan(w)
	clear(m.agg)
	for _, kind := range [...]string{FlowGrant, FlowLevy} {
		for i := range m.flows {
			if m.on[i] && m.flows[i].flow.Kind == kind {
				w.runFlow(m, i)
			}
		}
	}
	w.reportFlows(m, tick)
}

// scan builds the class lists: one pass over the agents in ascending AgentID,
// which is ascending account ID, skipping the estates and the removed slots
// (weight 0). Membership is read from the agent's role and region now
// (FR-MON-007, 009).
func (m *moneyFlows) scan(w *World) {
	for i := range m.cells {
		m.cells[i] = m.cells[i][:0]
	}
	for id := m.regions; id < len(w.agents); id++ {
		if a := &w.agents[id]; a.weight >= 1 {
			cell := int(w.regionOf(a))*m.roles + int(a.role)
			m.cells[cell] = append(m.cells[cell], AgentID(id))
		}
	}
	for i := range m.flows {
		m.flows[i].fit(len(w.agents))
	}
}

// runFlow applies flow i to the accounts of its class.
func (w *World) runFlow(m *moneyFlows, i int) {
	f := &m.flows[i]
	lo, hi := 0, m.regions
	if f.region >= 0 {
		lo, hi = f.region, f.region+1
	}
	for g := lo; g < hi; g++ {
		for _, id := range m.cells[g*m.roles+int(f.role)] {
			w.visitFlow(m, i, g, &w.agents[id])
		}
	}
}

// visitFlow computes the due of one recipient, cuts it by the room bound and then
// the limit, moves what is left and ends the flow when it reaches its limit.
// The cells are scanned at the start of P1.7 and nothing relocates an account
// during the pass, so region is the recipient's region (a module that moved one
// here would credit the aggregate row to the wrong region).
func (w *World) visitFlow(m *moneyFlows, i, region int, a *agent) {
	f := &m.flows[i]
	v, err := w.ledger.Account(a.account)
	if err != nil {
		invariant("the account of an agent is unreadable: " + err.Error())
	}
	bal := v.Balance
	if f.flow.Kind == FlowLevy {
		bal -= v.Reserved // Transfer spends only balance less reserved
	}
	due, room := f.plan(a, bal)
	applied, cut := moneyflow.Clamp(due, room, f.limit-f.total)
	if applied > 0 {
		w.moveFlow(f, a, applied)
	}
	g := &m.agg[i*m.regions+region]
	g.due, g.applied = satAdd(g.due, due), satAdd(g.applied, applied)
	switch cut {
	case moneyflow.CutRoom:
		g.room = satAdd(g.room, a.weight)
	case moneyflow.CutLimit:
		g.limit = satAdd(g.limit, a.weight)
	}
	if f.total >= f.limit && f.ended == 0 {
		f.ended, f.endedNow = 2, true
		if f.flow.Limit > 0 {
			f.ended = 1
		}
	}
}

// plan returns the due of a recipient with the balance and the room its bound
// leaves, and advances the recipient's carry by the due (FR-MON-006, 022).
func (f *flowState) plan(a *agent, bal int64) (due, room int64) {
	fl := &f.flow
	if fl.Kind == FlowGrant && fl.Level != 0 {
		due, ok := moneyflow.TopUp(a.weight, fl.Level, f.scale, bal, MaxMoney)
		if !ok {
			invariant("a money flow's top-up left its validated bounds: " + fl.ID)
		}
		return due, MaxMoney - bal
	}
	c := f.carryOf(a)
	avail := availableAbove(bal, a.weight, fl.Exempt)
	var ok bool
	switch {
	case fl.Kind == FlowLevy && fl.RateBP != 0:
		due, *c, ok = moneyflow.RateLevy(avail, fl.RateBP, f.scale, *c, f.period, MaxMoney)
		room = avail
	default:
		due, *c, ok = moneyflow.Spread(a.weight, fl.Amount, f.scale, *c, f.period, MaxMoney)
		room = avail
		if fl.Kind == FlowGrant {
			room = MaxMoney - bal
		}
	}
	if !ok {
		invariant("a money flow's arithmetic left its validated bounds: " + fl.ID)
	}
	return due, room
}

// carryOf returns the carry of a recipient, starting it at the first visit.
func (f *flowState) carryOf(a *agent) *int64 {
	c := &f.carry[a.id]
	if *c == carryUnset {
		*c = 0
		if !(f.flow.Kind == FlowLevy && f.flow.RateBP != 0) {
			*c = moneyflow.StartCarry(int64(a.account), f.period)
		}
	}
	return c
}

// availableAbove is max(0, bal - w*exempt): the balance a levy may take. A
// product that does not fit leaves nothing to take.
func availableAbove(bal, weight, exempt int64) int64 {
	held, ok := checked.Mul(weight, exempt)
	if !ok {
		return 0
	}
	return max(bal-held, 0)
}

// moveFlow transfers an applied amount between the flow's boundary and the
// recipient, adds it to the flow's total and, for a grant that is income, to the
// recipient's unearned income of the tick (plan 2-6 D332).
func (w *World) moveFlow(f *flowState, a *agent, amount int64) {
	from, to := f.boundary, a.account
	if f.flow.Kind == FlowLevy {
		from, to = to, from
	}
	if err := w.ledger.Transfer(from, to, amount, f.reason); err != nil {
		invariant("the ledger refused a money flow's transfer: " + err.Error())
	}
	f.total += amount
	if f.flow.Kind == FlowGrant && !f.flow.NotIncome {
		w.welfare.addUnearned(int(a.id), amount)
	}
}

// reportFlows emits, after both passes, per flow in ascending ID and per region in
// ascending order the MoneyFlowLimited of a region where a due was not applied,
// then the flow's MoneyFlowEnded; events are built only with an observer, and the
// Debug lines only when enabled (plan 2-6 D334).
func (w *World) reportFlows(m *moneyFlows, tick int64) {
	for i := range m.flows {
		if !m.on[i] {
			continue
		}
		f := &m.flows[i]
		for g := range m.regions {
			if a := m.agg[i*m.regions+g]; a.due != a.applied {
				w.flowLimited(f, tick, RegionID(g), a)
			}
		}
		if f.endedNow {
			f.endedNow = false
			w.flowEnded(f, tick)
		}
	}
}

// flowLimited reports the unapplied part of a flow's dues in one region.
func (w *World) flowLimited(f *flowState, tick int64, region RegionID, a flowAgg) {
	short, headroom := a.room, int64(0)
	if f.flow.Kind == FlowGrant {
		short, headroom = 0, a.room
	}
	if w.obs.supplied {
		w.obs.deliver(MoneyFlowLimited{Tick: tick, Flow: f.flow.ID, Region: region, Due: Money(a.due), Applied: Money(a.applied),
			Short: short, Headroom: headroom, Limit: a.limit})
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "money flow limited", slog.String(logkeys.Flow, f.flow.ID),
			slog.Int64(logkeys.Region, int64(region)), slog.Int64(logkeys.Due, a.due), slog.Int64(logkeys.Applied, a.applied))
	}
}

// flowEnded reports a flow that reached its effective limit.
func (w *World) flowEnded(f *flowState, tick int64) {
	cause := CauseCap
	if f.ended == 1 {
		cause = CauseLimit
	}
	if w.obs.supplied {
		w.obs.deliver(MoneyFlowEnded{Tick: tick, Flow: f.flow.ID, Cause: cause, Total: Money(f.total)})
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "money flow ended", slog.String(logkeys.Flow, f.flow.ID), slog.String(logkeys.Cause, string(cause)))
	}
}

// setFlowScale sets the scale of flow i (the index in ID order) to bp basis
// points, from the next P1.7: ErrUnknownFlow for an index that is no flow and
// ErrOutOfBounds outside 0 to MaxFlowScale, changing nothing, and no event when
// the scale is already bp. It emits MoneyFlowScaled with the cause. The public
// call SetFlowScales is the integration module's (plan 2-6 D335).
func (w *World) setFlowScale(i int, bp int64, cause Cause) error {
	if w.money == nil || i < 0 || i >= len(w.money.flows) {
		return fmt.Errorf("money flow %d: %w", i, ErrUnknownFlow)
	}
	if bp < 0 || bp > MaxFlowScale {
		return fmt.Errorf("flow scale %d outside 0 to %d: %w", bp, int64(MaxFlowScale), ErrOutOfBounds)
	}
	f := &w.money.flows[i]
	from := f.scale
	if from == bp {
		return nil
	}
	f.scale = bp
	if w.obs.supplied {
		w.obs.deliver(MoneyFlowScaled{Tick: w.Tick(), Flow: f.flow.ID, From: from, To: bp, Cause: cause})
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "money flow scaled", slog.String(logkeys.Flow, f.flow.ID),
			slog.Int64(logkeys.From, from), slog.Int64(logkeys.To, bp))
	}
	return nil
}

// auditModules is the modules' part of the audit (plan 2-6 D333): for each flow
// the boundary balance of its reason is the total with the sign of its kind, and
// the first flow that disagrees is named (FR-MON-026, 027). It is nil with no
// module on.
func (w *World) auditModules() error {
	if w.money == nil {
		return nil
	}
	for _, f := range w.money.flows {
		v, err := w.ledger.Account(f.boundary)
		if err != nil {
			invariant("a boundary account is unreadable: " + err.Error())
		}
		want := -f.total
		if f.flow.Kind == FlowLevy {
			want = f.total
		}
		if v.Balance != want {
			return auditFailure(w.Tick(), fmt.Errorf("money flow %q: boundary balance %d, want %d from its total %d", f.flow.ID, v.Balance, want, f.total))
		}
	}
	return nil
}

// dropAgent drops the carries of a removed agent (spec-money-supply.md 14).
func (m *moneyFlows) dropAgent(a *agent) {
	for i := range m.flows {
		if f := &m.flows[i]; int(a.id) < len(f.carry) {
			f.carry[a.id] = carryUnset
		}
	}
}

// snapshotFlows returns the Flows rows of the snapshot, one per flow in ID order,
// empty with the module off.
func (w *World) snapshotFlows() []FlowRow {
	if w.money == nil {
		return nil
	}
	rows := make([]FlowRow, len(w.money.flows))
	for i, f := range w.money.flows {
		rows[i] = FlowRow{Flow: f.flow.ID, Total: Money(f.total), ScaleBP: f.scale, Ended: f.ended != 0}
	}
	return rows
}

// flowBoundaryRows appends the BoundaryRow of each flow's boundary account, in
// flow-ID order: the block between the built-in rows and the declared ones
// (spec-observability.md 4.4.1.2).
func (w *World) flowBoundaryRows(rows []BoundaryRow) []BoundaryRow {
	if w.money == nil {
		return rows
	}
	for _, f := range w.money.flows {
		v, err := w.ledger.Account(f.boundary)
		if err != nil {
			invariant("a boundary account is unreadable: " + err.Error())
		}
		kind := KindFaucet
		if f.flow.Kind == FlowLevy {
			kind = KindSink
		}
		rows = append(rows, BoundaryRow{Reason: Reason(f.flow.ID), Kind: kind, Commodity: NoCommodity, Balance: v.Balance})
	}
	return rows
}

// saveMoneySupply is tag 13 (spec-checkpoint.md 4.2, plan 2-6 D337): per flow in
// ID order its total, scale and end, then per flow that keeps a carry the started
// carries as (account ID, carry) pairs in ascending account ID. An agent has one
// account and the IDs rise with the account, so the pairs are in ascending order;
// the carries of removed agents were dropped.
func (w *World) saveMoneySupply(cw *codec.Writer) error {
	for _, f := range w.money.flows {
		cw.U(uint64(f.total))
		cw.U(uint64(f.scale))
		cw.U(uint64(f.ended))
	}
	for i := range w.money.flows {
		f := &w.money.flows[i]
		if f.hasCarry() {
			w.saveCarries(cw, f)
		}
	}
	return nil
}

// saveCarries writes the started carries of one flow: a count, then the pairs.
func (w *World) saveCarries(cw *codec.Writer, f *flowState) {
	n := 0
	w.eachStarted(f, func(*agent, int64) { n++ })
	cw.Count(n)
	w.eachStarted(f, func(a *agent, c int64) {
		cw.U(uint64(a.account))
		cw.U(uint64(c))
	})
}

// eachStarted calls fn for each live agent whose carry in f is started, in
// ascending AgentID.
func (w *World) eachStarted(f *flowState, fn func(a *agent, carry int64)) {
	w.eachLive(func(a *agent) {
		if int(a.id) < len(f.carry) && f.carry[a.id] != carryUnset {
			fn(a, f.carry[a.id])
		}
	})
}

// loadMoneySupply is tag 13: the totals, scales and ends, each in its bound and
// consistent with the flow (an ended flow is at its effective limit, with the
// cause its configuration gives), then the carry pairs of each flow that keeps
// one, with the account strictly ascending and the carry inside its denominator.
// That the account is the account of a live agent is cross-check 7.
func (w *World) loadMoneySupply(_ *restoreCtx, r *codec.Reader) error {
	m := w.money
	for i := range m.flows {
		f := &m.flows[i]
		f.total = r.Bounded("flow total", f.limit)
		f.scale = r.Bounded("flow scale", MaxFlowScale)
		f.ended = uint8(r.Index(3))
		if r.Err() == nil && !f.endConsistent() {
			r.Failf("flow %q has total %d and end %d against its limit %d", f.flow.ID, f.total, f.ended, f.limit)
		}
	}
	for i := range m.flows {
		if f := &m.flows[i]; f.hasCarry() && r.Err() == nil {
			w.loadCarries(r, f)
		}
	}
	return nil
}

// endConsistent reports whether the end of a flow matches its total: it is ended
// exactly when the total is the effective limit, by the limit when Limit is set
// and by the cap otherwise.
func (f *flowState) endConsistent() bool {
	switch f.ended {
	case 0:
		return f.total < f.limit
	case 1:
		return f.total == f.limit && f.flow.Limit > 0
	default:
		return f.total == f.limit && f.flow.Limit == 0
	}
}

// loadCarries reads the pairs of one flow into its carry table.
func (w *World) loadCarries(r *codec.Reader, f *flowState) {
	f.carry = nil
	f.fit(len(w.agents))
	den, prev := f.carryDen(), uint64(0)
	for i := range r.Count(2) {
		acct := r.Strict(prev, i > 0)
		carry := r.Bounded("flow carry", den-1)
		if r.Err() != nil {
			return
		}
		prev = acct
		v, err := w.ledger.Account(int(acct))
		if err != nil || v.Kind != ledger.KindAgent || v.Owner < w.money.regions || v.Owner >= len(w.agents) {
			r.Failf("flow carry names account %d, which is not an agent's", acct)
			return
		}
		f.carry[v.Owner] = carry
	}
}

// moneyCross is the module's part of step 7 (cross-check 7): every started carry
// is the carry of a live agent whose account is the one the pair named.
func (w *World) moneyCross() error {
	for _, f := range w.money.flows {
		for id, c := range f.carry {
			if a := &w.agents[id]; c != carryUnset && a.weight < 1 {
				return fmt.Errorf("%w: flow %q has a carry for agent %d, whose account is not open", ErrCheckpointCorrupt, f.flow.ID, id)
			}
		}
	}
	return nil
}
