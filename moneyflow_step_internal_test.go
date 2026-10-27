package economysim

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The internals of the money supply module's step: the scale setter, the removal of
// a carry, the audit, the welfare income term and the cases a world of the public
// API cannot reach (a balance at MaxMoney, a total at the cap). The step through
// the public API is moneyflow_step_test.go.

// obsRec is a recording Observer of the internal tests.
type obsRec struct{ events []Event }

func (r *obsRec) Observe(e Event) { r.events = append(r.events, e) }

// moneyMill builds the one-region world of millConfig with households of weight
// 1 and the flows of the module on, the audit on, ready for p17Flows at tick 0.
func moneyMill(t *testing.T, households int64, flows ...MoneyFlow) (*World, *obsRec) {
	t.Helper()
	c := millConfig(1, households)
	c.Roles[0].StartMoney = 1_000
	c.Modules.MoneySupply = &MoneySupply{Flows: flows}
	rec := &obsRec{}
	return newMill(t, c, WithSeed(1), WithAudit(), WithObserver(rec)), rec
}

// millerFlow is a flow of the kind to the millers.
func millerFlow(id, kind string, amount int64) MoneyFlow {
	return MoneyFlow{ID: id, Kind: kind, Amount: amount, Target: FlowTarget{Kind: TargetHouseholds, Role: "miller"}}
}

// holds returns the money of an agent.
func holds(t *testing.T, w *World, id AgentID) int64 {
	t.Helper()
	held, _ := balance(t, w, id)
	return held
}

// eventsOf returns the events of the recorder that are of the type T.
func eventsOf[T Event](r *obsRec) []T {
	var out []T
	for _, e := range r.events {
		if v, ok := e.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

// TestSetFlowScale_BoundsAndEvent verifies that the internal scale setter
// refuses an unknown flow and a scale outside 0 to MaxFlowScale without changing
// anything, emits MoneyFlowScaled once for a change and nothing for an equal
// scale, and that the new scale applies from the next P1.7.
//
// Requirement: FR-MON-023, FR-MON-045
func TestSetFlowScale_BoundsAndEvent(t *testing.T) {
	t.Parallel()
	w, rec := moneyMill(t, 4, millerFlow("stipend", FlowGrant, 10))
	for _, tt := range []struct {
		name string
		i    int
		bp   int64
		want error
	}{
		{"unknown flow", 1, 5_000, ErrUnknownFlow},
		{"negative index", -1, 5_000, ErrUnknownFlow},
		{"above the bound", 0, MaxFlowScale + 1, ErrOutOfBounds},
		{"below zero", 0, -1, ErrOutOfBounds},
	} {
		if err := w.setFlowScale(tt.i, tt.bp, CauseConsumer); !errors.Is(err, tt.want) {
			t.Errorf("%s: setFlowScale(%d, %d) = %v, want %v", tt.name, tt.i, tt.bp, err, tt.want)
		}
	}
	if err := w.setFlowScale(0, wholeBP, CauseConsumer); err != nil || len(rec.events) != 0 && len(eventsOf[MoneyFlowScaled](rec)) != 0 {
		t.Errorf("setFlowScale to the same scale = %v with %d scaled events, want nil and none", err, len(eventsOf[MoneyFlowScaled](rec)))
	}
	if err := w.setFlowScale(0, 5_000, CauseConsumer); err != nil {
		t.Fatalf("setFlowScale(0, 5000) = %v, want nil", err)
	}
	got := eventsOf[MoneyFlowScaled](rec)
	if want := []MoneyFlowScaled{{Tick: 0, Flow: "stipend", From: 10_000, To: 5_000, Cause: CauseConsumer}}; !slices.Equal(got, want) {
		t.Errorf("MoneyFlowScaled events = %+v, want %+v", got, want)
	}
	w.p17Flows()
	// 10 a household-tick at half scale is 5 each, 4 households: 20.
	if tot := w.money.flows[0].total; tot != 20 {
		t.Errorf("total after one P1.7 at half scale = %d, want 5 * 4 = 20", tot)
	}
	if err := w.setFlowScale(0, 0, CauseConsumer); err != nil {
		t.Fatalf("setFlowScale(0, 0) = %v, want nil", err)
	}
	w.p17Flows()
	if tot := w.money.flows[0].total; tot != 20 {
		t.Errorf("total after a P1.7 at scale 0 = %d, want 20: a paused flow applies nothing", tot)
	}
	off := newMill(t, millConfig(1, 4))
	if err := off.setFlowScale(0, 1, CauseConsumer); !errors.Is(err, ErrUnknownFlow) {
		t.Errorf("setFlowScale with the module off = %v, want ErrUnknownFlow", err)
	}
}

// TestAudit_NamesTheFirstFlowThatDisagrees verifies that the audit passes while
// every flow's boundary balance is minus its total (a grant) or its total (a
// levy), and that after a total is corrupted World.Audit wraps ErrAuditFailed and
// names the first flow, in ID order, that disagrees.
//
// Requirement: FR-MON-002, FR-MON-026, FR-MON-027
func TestAudit_NamesTheFirstFlowThatDisagrees(t *testing.T) {
	t.Parallel()
	w, _ := moneyMill(t, 4, millerFlow("aid", FlowGrant, 10), millerFlow("tax", FlowLevy, 5))
	w.p17Flows()
	if err := w.Audit(); err != nil {
		t.Fatalf("Audit after P1.7 = %v, want nil", err)
	}
	w.money.flows[1].total++
	w.money.flows[0].total++
	err := w.Audit()
	if !errors.Is(err, ErrAuditFailed) || err == nil || !strings.Contains(err.Error(), `"aid"`) || strings.Contains(err.Error(), `"tax"`) {
		t.Errorf("Audit with both totals corrupted = %v, want ErrAuditFailed naming only the first flow aid", err)
	}
	w.money.flows[0].total--
	if err := w.p84AuditForce(); err == nil || !strings.Contains(err.Error(), `"tax"`) {
		t.Errorf("p84Audit with tax corrupted = %v, want an error naming tax", err)
	}
}

// p84AuditForce runs the audit of P8.4 as a world with WithAudit does.
func (w *World) p84AuditForce() error {
	w.audit = true
	return w.p84Audit()
}

// TestWelfare_GrantIsUnearnedIncome verifies that a grant that is income adds its
// applied amount to the recipient's unearned income of the tick, that a grant with
// NotIncome and a levy add nothing, and so that the observed wage excludes it.
//
// Requirement: FR-MON-025
func TestWelfare_GrantIsUnearnedIncome(t *testing.T) {
	t.Parallel()
	income := millerFlow("income", FlowGrant, 10)
	gift := millerFlow("gift", FlowGrant, 100)
	gift.NotIncome = true
	levy := millerFlow("levy", FlowLevy, 7)
	w, _ := moneyMill(t, 4, income, gift, levy)
	w.p17Flows()
	for id := AgentID(1); id < AgentID(len(w.agents)); id++ {
		if got, want := w.welfare.slots[id].unearned, 10*w.agents[id].weight; got != want {
			t.Errorf("agent %d unearned income = %d, want %d: only the grant that is income counts", id, got, want)
		}
	}
}

// TestFlow_RemovalDropsCarry verifies that a rate flow starts the carry of an
// account at its first visit, and that removing the agent drops the carry, so a
// started carry is only ever the carry of a live agent.
//
// Requirement: FR-MON-013, FR-MON-040
func TestFlow_RemovalDropsCarry(t *testing.T) {
	t.Parallel()
	f := millerFlow("slow", FlowGrant, 7)
	f.Per = PerYear
	w, _ := moneyMill(t, 4, f)
	fs := &w.money.flows[0]
	if len(fs.carry) != 0 {
		t.Fatalf("carry table before the first P1.7 has %d entries, want none", len(fs.carry))
	}
	w.p17Flows()
	a := &w.agents[1]
	// The carry starts at (account mod P) * 10^4 and has paid one tick: 7 * 10^4
	// more, less the whole units paid, so it is below P * 10^4.
	if c := fs.carry[1]; c < 0 || c >= fs.period*wholeBP {
		t.Fatalf("carry of agent 1 = %d, want in [0, %d)", c, fs.period*wholeBP)
	}
	if err := w.removeAgent(a, CauseInsolvent); err != nil {
		t.Fatalf("removeAgent = %v, want nil", err)
	}
	if got := fs.carry[1]; got != carryUnset {
		t.Errorf("carry of the removed agent = %d, want unset (%d)", got, carryUnset)
	}
	if got := fs.carry[2]; got == carryUnset {
		t.Error("carry of a live agent was dropped with another's removal")
	}
}

// TestFlow_HeadroomAndCap verifies the two cuts a world of the public API cannot
// reach cheaply: a grant that would take a balance past MaxMoney pays up to it and
// counts the households under Headroom, and a flow whose total reaches
// MaxFlowTotal ends with the cause cap.
//
// Requirement: FR-MON-021, FR-MON-011, FR-MON-030
func TestFlow_HeadroomAndCap(t *testing.T) {
	t.Parallel()
	g := millerFlow("big", FlowGrant, 1_000)
	w, rec := moneyMill(t, 1, g)
	a := &w.agents[1]
	initial, ok := w.ledger.Boundary(ledger.Initial, ledger.Money)
	if !ok {
		t.Fatal("no boundary account of the initial reason")
	}
	// Leave 10 of room below MaxMoney: the grant of 1 000 * weight is cut to 10.
	room := int64(10)
	if err := w.ledger.Transfer(initial, a.account, MaxMoney-holds(t, w, 1)-room, ledger.Initial); err != nil {
		t.Fatalf("raise the balance: %v", err)
	}
	w.p17Flows()
	want := []MoneyFlowLimited{{Tick: 0, Flow: "big", Due: Money(1_000 * a.weight), Applied: 10, Headroom: a.weight}}
	if got := eventsOf[MoneyFlowLimited](rec); !slices.Equal(got, want) {
		t.Errorf("MoneyFlowLimited = %+v, want %+v", got, want)
	}
	if holds(t, w, 1) != MaxMoney {
		t.Errorf("balance after the cut grant = %d, want MaxMoney", holds(t, w, 1))
	}
	// A flow 5 below the cap ends on the cap when the next payment reaches it.
	w2, rec2 := moneyMill(t, 1, millerFlow("cap", FlowGrant, 1_000))
	w2.money.flows[0].total = MaxFlowTotal - 5
	w2.p17Flows()
	ended := eventsOf[MoneyFlowEnded](rec2)
	if len(ended) != 1 || ended[0].Cause != CauseCap || ended[0].Total != MaxFlowTotal {
		t.Errorf("MoneyFlowEnded = %+v, want one with the cause cap and the total MaxFlowTotal", ended)
	}
}

// TestFlow_MembershipFollowsRelocate verifies that a class is read each tick from
// the agent's current role and region: a flow aimed at one region pays an agent
// that is there and stops paying it when it moves away, keeping its carry.
//
// Requirement: FR-MON-009, FR-MON-007
func TestFlow_MembershipFollowsRelocate(t *testing.T) {
	t.Parallel()
	c := millConfig(1, 4)
	away := DefaultRegion()
	away.ID = "away"
	c.Regions = append(c.Regions, away)
	f := millerFlow("local", FlowGrant, 10)
	f.Target.Region = "home"
	c.Modules.MoneySupply = &MoneySupply{Flows: []MoneyFlow{f}}
	w := newMill(t, c, WithSeed(1), WithAudit())
	w.p17Flows()
	before := w.money.flows[0].total
	if before != 40 {
		t.Fatalf("total in the home region = %d, want 10 * 4 = 40", before)
	}
	if err := w.moveAgent(&w.agents[2], 0, CauseConsumer); err != nil {
		t.Fatalf("moveAgent = %v, want nil", err)
	}
	w.p17Flows()
	if got := w.money.flows[0].total - before; got != 30 {
		t.Errorf("paid after one agent moved out = %d, want 30: the mover is no longer in the class", got)
	}
	if w.money.flows[0].carry[2] == carryUnset {
		t.Error("the mover's carry was dropped, want it kept for its return")
	}
}

// TestFlow_NoRuleChangesAFlowItself verifies that a run changes only the totals,
// the carries and the end of a flow: the scale and the configuration stay as
// they were.
//
// Requirement: FR-MON-024
//
// Guard: the engine must not adjust a flow by itself (a control law in the
// engine); without this test such a rule would show only as a changed money path.
func TestFlow_NoRuleChangesAFlowItself(t *testing.T) {
	t.Parallel()
	w, _ := moneyMill(t, 6, millerFlow("aid", FlowGrant, 3), millerFlow("tax", FlowLevy, 2))
	cfg := w.money.flows[0].flow
	for range 30 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
	}
	for i, f := range w.money.flows {
		if f.scale != wholeBP || f.limit != MaxFlowTotal {
			t.Errorf("flow %d scale %d limit %d after a run, want %d and %d unchanged", i, f.scale, f.limit, wholeBP, int64(MaxFlowTotal))
		}
	}
	if w.money.flows[0].flow != cfg {
		t.Error("the configuration of a flow changed in a run")
	}
}

// TestFlow_CreationDrawsIndependentOfTheModule verifies that the draws made at
// agent creation do not depend on whether the module is on: two worlds of one
// seed, with and without it, hold the same target ratio for every agent.
//
// Requirement: FR-MON-041
func TestFlow_CreationDrawsIndependentOfTheModule(t *testing.T) {
	t.Parallel()
	on, _ := moneyMill(t, 6, millerFlow("aid", FlowGrant, 3))
	c := millConfig(1, 6)
	c.Roles[0].StartMoney = 1_000
	off := newMill(t, c, WithSeed(1))
	if len(on.agents) != len(off.agents) {
		t.Fatalf("agents: %d with the module on, %d off", len(on.agents), len(off.agents))
	}
	for i := range on.agents {
		if on.agents[i].target != off.agents[i].target {
			t.Errorf("agent %d target %d on, %d off, want equal draws", i, on.agents[i].target, off.agents[i].target)
		}
	}
}

// TestPlanFlow_DueIgnoresEverythingElse verifies that the due of a recipient of a
// rate grant is a function of its weight, carry, the flow and its scale alone: the
// same recipient state gives the same due whatever the balance, and a top-up gives
// the due that only its own balance sets.
//
// Requirement: FR-MON-006, FR-MON-016
func TestPlanFlow_DueIgnoresEverythingElse(t *testing.T) {
	t.Parallel()
	w, _ := moneyMill(t, 4, millerFlow("aid", FlowGrant, 13))
	topUp := MoneyFlow{ID: "top", Kind: FlowGrant, Level: 500, Target: FlowTarget{Kind: TargetHouseholds, Role: "miller"}}
	w2, _ := moneyMill(t, 4, topUp)
	a := &w.agents[1]
	w.money.scan(w)
	f := &w.money.flows[0]
	f.carry[1] = 0
	first, _ := f.plan(a.weight, &f.carry[1], 0)
	f.carry[1] = 0
	second, _ := f.plan(a.weight, &f.carry[1], 777)
	if first != second || first != 13*a.weight {
		t.Errorf("rate grant due = %d at balance 0 and %d at 777, want 13 * %d = %d for both", first, second, a.weight, 13*a.weight)
	}
	b := &w2.agents[1]
	w2.money.scan(w2)
	g := &w2.money.flows[0]
	if due, _ := g.plan(b.weight, &g.carry[1], 100); due != 500*b.weight-100 {
		t.Errorf("top-up due at balance 100 = %d, want 500 * %d - 100", due, b.weight)
	}
	if due, _ := g.plan(b.weight, &g.carry[1], 500*b.weight+9); due != 0 {
		t.Errorf("top-up due above the level = %d, want 0", due)
	}
}

// TestP17Flows_AllocatesNothing verifies that the step of the module allocates
// nothing once its scratch has grown, with the default logger and observer: the
// class lists, the carries and the aggregates are reused, no event is built and no
// log argument is evaluated.
//
// Requirement: FR-MON-033, NFR-MON-001
//
// Guard: a slice or a boxed event built per recipient would make the step cost
// grow with the households; the world's baseline (allocation_test.go) would show
// it only as a sum with the rest of the tick.
func TestP17Flows_AllocatesNothing(t *testing.T) {
	// Not parallel: testing.AllocsPerRun panics in a parallel test.
	short := millerFlow("tax", FlowLevy, 100) // short: the MoneyFlowLimited path runs
	rate := MoneyFlow{ID: "rate", Kind: FlowLevy, RateBP: 500, Per: PerYear, Target: FlowTarget{Kind: TargetHouseholds, Role: "miller"}}
	w, _ := moneyMill(t, 50, millerFlow("aid", FlowGrant, 7), short, rate)
	w.obs = observers{}
	for range 5 {
		w.p17Flows()
	}
	if got := testing.AllocsPerRun(50, func() { w.p17Flows() }); got != 0 {
		t.Errorf("allocations per P1.7 = %v, want 0", got)
	}
}

// TestMoneyCross_RefusesACarryOfAClosedAccount verifies cross-check 7: a started
// carry of an agent that is not live is refused with ErrCheckpointCorrupt, and the
// carries of live agents pass.
//
// Requirement: FR-MON-040
func TestMoneyCross_RefusesACarryOfAClosedAccount(t *testing.T) {
	t.Parallel()
	w, _ := moneyMill(t, 4, millerFlow("aid", FlowGrant, 7))
	w.p17Flows()
	if err := w.moneyCross(); err != nil {
		t.Fatalf("moneyCross with live agents = %v, want nil", err)
	}
	w.agents[2] = agent{}
	if err := w.moneyCross(); !errors.Is(err, ErrCheckpointCorrupt) {
		t.Errorf("moneyCross with a carry of a removed agent = %v, want ErrCheckpointCorrupt", err)
	}
}

// TestFlow_LevyIgnoresReservedMoney verifies that a levy sizes its due from the
// balance less the part a reservation holds, so a household whose money is all
// reserved pays nothing instead of the transfer being refused.
//
// Requirement: FR-MON-009
//
// Fix: stage-2-6 review. The levy read the raw balance, so Ledger.Transfer, which
// spends only balance less reserved, refused it and moveFlow panicked.
func TestFlow_LevyIgnoresReservedMoney(t *testing.T) {
	t.Parallel()
	w, _ := moneyMill(t, 1, millerFlow("tax", FlowLevy, 5))
	a := &w.agents[1]
	held := holds(t, w, 1)
	if err := w.ledger.ReserveMoney(a.account, held); err != nil {
		t.Fatalf("reserve the whole balance: %v", err)
	}
	w.p17Flows()
	if got := holds(t, w, 1); got != held {
		t.Errorf("balance after a levy on fully reserved money = %d, want %d unchanged", got, held)
	}
}
