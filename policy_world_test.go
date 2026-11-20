package economysim_test

import (
	"errors"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the policy module through the public API only (spec-policy.md 6, 7, 8,
// 12; plan 3-plc Phase 5): a governed pack on which three actors move edges, a transit
// factor and flows by a schedule and a rule, observed as a consumer would observe it. The
// arithmetic is in internal/policy, the checks and the counts in
// policy_apply_internal_test.go.

// policyGoverned is policyConfig with the module on in its built-in forms only: the crown
// holds the head tax by a schedule (12 000 from tick 0, 8 000 from tick 20) and the
// footpath by a schedule (6 000); the coast holds the border with the inland province,
// which is the two edges between the harbour and the mill, by a schedule (20 000 from tick
// 0, an Against group of two keys); the inland province moves its stipend by a rule on its
// treasury (5 000 against a target of 1 000, raised by at most 1 000 a decision). Every
// actor decides every 10 ticks, so each decides four times in ticks 0 to 39.
func policyGoverned() economysim.Config {
	c := policyConfig()
	c.Modules.Policy.Actors[0].Schedule = []economysim.ScheduleStep{
		{Lever: "poll", At: 0, BP: 12_000}, {Lever: "poll", At: 20, BP: 8_000}, {Lever: "road", At: 0, BP: 6_000},
	}
	c.Modules.Policy.Actors[1].Policy = "schedule"
	c.Modules.Policy.Actors[1].Schedule = []economysim.ScheduleStep{{Lever: "gate", At: 0, BP: 20_000}}
	in := &c.Modules.Policy.Actors[2]
	in.Every, in.ReadsRegions = 10, false
	in.Rules = []economysim.Reaction{{Lever: "stipend", Gauge: economysim.GaugeTreasury, Target: 1_000, Deadband: 100, Span: 1_000, Move: 1_000, Raise: true}}
	return c
}

// policyRun builds the world of c with a recording observer, forgets the events of the
// construction and runs ticks steps.
func policyRun(t testing.TB, c economysim.Config, ticks int, opts ...economysim.Option) (*economysim.World, *recorder) {
	t.Helper()
	rec := &recorder{}
	w, err := economysim.NewWorld(c, append([]economysim.Option{economysim.WithSeed(11), economysim.WithObserver(rec)}, opts...)...)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	rec.forget()
	policyStep(t, w, ticks)
	return w, rec
}

// policyStep runs n steps.
func policyStep(t testing.TB, w *economysim.World, n int) {
	t.Helper()
	for range n {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", w.Tick(), err)
		}
	}
}

// recordedOf returns the events of one type, in order.
func recordedOf[T economysim.Event](rec *recorder) []T {
	var out []T
	for _, e := range rec.events {
		if v, ok := e.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

// TestPolicy_EndToEnd_ScheduleAndRuleOnAGovernedPack verifies, through the public API, what
// a consumer sees when three actors govern: the edges, the transit factor and the flows
// move with the cause policy, the first decision of each actor is counted and says what
// became of its entries, the sum of the counts is Entries, and Slots lists the final values.
//
// Arithmetic: the crown's first decision sets poll 10 000 to 12 000 (the step limit is
// 5 000) and road 10 000 to 6 000, two applied entries; the coast sets both edges of the
// group, harbour to mill and mill to harbour, to 20 000, one applied entry; the inland
// province, with a treasury of 5 000 against a target of 1 000, raises stipend by the
// step of 1 000 to 11 000. From tick 20 the crown's schedule takes poll to 8 000.
//
// Requirement: FR-PLC-008, FR-PLC-043, FR-PLC-060, FR-PLC-061, FR-PLC-062, FR-PLC-082, FR-PLC-085
func TestPolicy_EndToEnd_ScheduleAndRuleOnAGovernedPack(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	w, rec := policyRun(t, policyGoverned(), 40)

	checkDecisions(t, rec)
	checkEdgeEvents(t, rec)
	assertFlowMoves(t, recordedOf[economysim.MoneyFlowScaled](rec))

	slots, err := w.Slots()
	if err != nil {
		t.Fatalf("Slots() error = %v, want nil", err)
	}
	if len(slots.EdgeStatuses) != 2 || len(slots.TransitFactors) != 1 || slots.TransitFactors[0].BP != 6_000 {
		t.Errorf("Slots() edges %v, transit %v, want two edges at 20 000 and the road at 6 000", slots.EdgeStatuses, slots.TransitFactors)
	}
	if got := flowScale(slots, "poll"); got != 8_000 {
		t.Errorf("Slots() scale of poll = %d, want 8 000 from the schedule step at tick 20", got)
	}
}

// checkDecisions checks the PolicyDecided of the governed run: three actors, four decisions
// each, Entries the sum of the outcomes.
func checkDecisions(t *testing.T, rec *recorder) {
	t.Helper()
	byActor := map[economysim.ActorID]int{}
	for _, d := range recordedOf[economysim.PolicyDecided](rec) {
		byActor[d.Actor]++
		if sum := d.Applied + d.Unchanged + d.NotHeld + d.Duplicate + d.NoClose; sum != d.Entries {
			t.Errorf("PolicyDecided at tick %d = %+v, want Entries equal to the sum of its outcomes %d", d.Tick, d, sum)
		}
	}
	if len(byActor) != 3 {
		t.Fatalf("actors that emitted PolicyDecided = %d, want 3", len(byActor))
	}
	for id, n := range byActor {
		if n != 4 {
			t.Errorf("actor %d decided %d times in 40 ticks, want 4 (every 10)", id, n)
		}
	}
}

// checkEdgeEvents checks that the coast's group moved once: two events, cause policy, 20 000.
func checkEdgeEvents(t *testing.T, rec *recorder) {
	t.Helper()
	got := recordedOf[economysim.EdgeStatusChanged](rec)
	if len(got) != 2 {
		t.Errorf("EdgeStatusChanged events = %d, want 2 (the group of two keys, moved once)", len(got))
	}
	for _, e := range got {
		if e.Cause != economysim.CausePolicy || e.FactorBP != 20_000 {
			t.Errorf("EdgeStatusChanged = %+v, want cause policy and factor 20 000", e)
		}
	}
}

// assertFlowMoves checks the MoneyFlowScaled events of the governed run: all are caused by
// the policy, poll goes 12 000 then 8 000 (not before tick 20) and stipend starts at 11 000.
func assertFlowMoves(t *testing.T, moved []economysim.MoneyFlowScaled) {
	t.Helper()
	var poll, stipend []economysim.MoneyFlowScaled
	for _, m := range moved {
		if m.Cause != economysim.CausePolicy {
			t.Errorf("MoneyFlowScaled %+v has cause %v, want policy", m, m.Cause)
		}
		switch m.Flow {
		case "poll":
			poll = append(poll, m)
		case "stipend":
			stipend = append(stipend, m)
		}
	}
	if !pollMovesAsScheduled(poll) {
		t.Errorf("poll moves = %+v, want 10 000 to 12 000, then to 8 000 from tick 20", poll)
	}
	if len(stipend) == 0 || stipend[0].From != 10_000 || stipend[0].To != 11_000 {
		t.Errorf("stipend moves = %+v, want the first 10 000 to 11 000 (the step limit)", stipend)
	}
}

// pollMovesAsScheduled reports whether the moves of poll are the two steps of the crown's schedule.
func pollMovesAsScheduled(m []economysim.MoneyFlowScaled) bool {
	return len(m) == 2 && m[0].From == 10_000 && m[0].To == 12_000 && m[1].To == 8_000 && m[1].Tick >= 20
}

// flowScale returns the scale of a flow in a SlotState, 10 000 when it is not listed.
func flowScale(s economysim.SlotState, id string) int64 {
	for _, f := range s.FlowScales {
		if f.Flow == id {
			return f.BP
		}
	}
	return 10_000
}

// TestPolicy_EventOrderPerTick verifies the order a consumer reads (12.1.3): in a tick the
// slot events of the policy follow TickStarted and PolicyDecided of an actor closes its
// slot events; an actor that decides emits its edge events, then its transit events, then
// its flow events, then PolicyDecided.
//
// Requirement: FR-PLC-083, FR-PLC-043
func TestPolicy_EventOrderPerTick(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	_, rec := policyRun(t, policyGoverned(), 12)
	rank := func(e economysim.Event) int {
		switch e.(type) {
		case economysim.EdgeStatusChanged:
			return 0
		case economysim.TransitFactorChanged:
			return 1
		case economysim.MoneyFlowScaled:
			return 2
		}
		return -1
	}
	var lastRank int
	started := false
	var tick int64
	for _, e := range rec.events {
		switch v := e.(type) {
		case economysim.TickStarted:
			started, lastRank, tick = true, 0, v.Tick
		case economysim.PolicyDecided:
			lastRank = 0
		default:
			r := rank(e)
			if r < 0 {
				continue
			}
			if !started {
				t.Fatalf("%T before the first TickStarted", e)
			}
			if r < lastRank {
				t.Errorf("tick %d: %T (rank %d) after rank %d within one actor's application, want edge, transit, flow", tick, e, r, lastRank)
			}
			lastRank = r
		}
	}
}

// TestPolicy_RunsAreIdentical verifies that two runs of the same configuration and seed
// emit the same events, in the same order, and end with the same slots (NFR-PLC-004), and
// that observing changes nothing: a third run with no observer ends with the same slots
// (NFR-PLC-005).
//
// Requirement: NFR-PLC-004, NFR-PLC-005
func TestPolicy_RunsAreIdentical(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	w1, r1 := policyRun(t, policyGoverned(), 40)
	w2, r2 := policyRun(t, policyGoverned(), 40)
	if !reflect.DeepEqual(r1.events, r2.events) {
		t.Errorf("two runs with seed 11 emitted different events (%d and %d)", len(r1.events), len(r2.events))
	}
	s1, err1 := w1.Slots()
	s2, err2 := w2.Slots()
	if err1 != nil || err2 != nil || !reflect.DeepEqual(s1, s2) {
		t.Errorf("Slots() of two runs = %v (%v) and %v (%v), want equal", s1, err1, s2, err2)
	}
	w3, err := economysim.NewWorld(policyGoverned(), economysim.WithSeed(11))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	policyStep(t, w3, 40)
	if s3, err := w3.Slots(); err != nil || !reflect.DeepEqual(s1, s3) {
		t.Errorf("Slots() of the unobserved run = %v (%v), want %v", s3, err, s1)
	}
}

// TestPolicy_HoldAndAbsentTableAreNeutral verifies on the governed pack that a world with
// no policy table and a world whose actors all hold emit the same events after the
// construction (no PolicyDecided, no slot event) and end in the same slots: a hold actor
// writes nothing and says nothing (FR-PLC-081).
//
// Requirement: FR-PLC-081, FR-PLC-010
func TestPolicy_HoldAndAbsentTableAreNeutral(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	off := policyGoverned()
	off.Modules.Policy = nil
	held := policyGoverned()
	for i := range held.Modules.Policy.Actors {
		a := &held.Modules.Policy.Actors[i]
		a.Policy, a.Schedule, a.Rules = "hold", nil, nil
	}
	w1, r1 := policyRun(t, off, 25)
	w2, r2 := policyRun(t, held, 25)
	if n := len(recordedOf[economysim.PolicyDecided](r2)); n != 0 {
		t.Errorf("PolicyDecided events from hold actors = %d, want 0", n)
	}
	if !reflect.DeepEqual(r1.events, r2.events) {
		t.Errorf("events with no policy table (%d) and with hold actors (%d) differ, want equal", len(r1.events), len(r2.events))
	}
	s1, _ := w1.Slots()
	s2, _ := w2.Slots()
	if !reflect.DeepEqual(s1, s2) {
		t.Errorf("Slots() with no table = %v, with hold actors = %v, want equal", s1, s2)
	}
}

// TestPolicy_TickZeroActsOnConstruction verifies that an actor due at tick 0 acts on the
// state the construction left and that its settings take effect in tick 0: the crown's
// schedule step at tick 0 is a MoneyFlowScaled of tick 0, and the first event after
// TickStarted is a slot event of the policy.
//
// Requirement: FR-PLC-052
func TestPolicy_TickZeroActsOnConstruction(t *testing.T) {
	t.Parallel()
	c := policyGoverned()
	// Every is 1, so each actor is due at tick 0 whatever its rotation.
	for i := range c.Modules.Policy.Actors {
		c.Modules.Policy.Actors[i].Every = 1
	}
	_, rec := policyRun(t, c, 1)
	var afterStart []economysim.Event
	for i, e := range rec.events {
		if _, ok := e.(economysim.TickStarted); ok {
			afterStart = rec.events[i+1:]
			break
		}
	}
	if len(afterStart) == 0 {
		t.Fatalf("events after TickStarted = %d, want the policy events", len(afterStart))
	}
	if _, ok := afterStart[0].(economysim.EdgeStatusChanged); !ok {
		t.Errorf("first event after TickStarted = %T, want the first slot event of the policy", afterStart[0])
	}
	if m := recordedOf[economysim.MoneyFlowScaled](rec); len(m) == 0 || m[0].Tick != 0 {
		t.Errorf("MoneyFlowScaled = %+v, want the first at tick 0", m)
	}
	if d := recordedOf[economysim.PolicyDecided](rec); len(d) != 3 {
		t.Errorf("PolicyDecided at tick 0 = %d, want 3 (every actor is due)", len(d))
	}
}

// TestPolicy_ConsumerCallOnAHeldKeyStands verifies FR-PLC-045 through the public API: the
// crown's schedule holds the lever poll until tick 15, so a consumer's SetFlowScales at
// tick 0 is legal and stays through the decisions that return no entry for poll; once the
// schedule returns an entry, the actor moves the key from where the consumer left it, by
// at most the step.
//
// Arithmetic: the consumer sets 3 000 (cause consumer). The schedule step at tick 15 is
// 14 000 (the validator allows 10 000 + the step limit of 5 000 at most): the decision
// is cut by the step limit to 3 000 + 5 000 = 8 000 at the first decision from tick 15.
//
// Requirement: FR-PLC-045
func TestPolicy_ConsumerCallOnAHeldKeyStands(t *testing.T) {
	t.Parallel()
	c := policyGoverned()
	c.Modules.Policy.Actors[0].Schedule = []economysim.ScheduleStep{{Lever: "poll", At: 15, BP: 14_000}, {Lever: "road", At: 0, BP: 6_000}}
	rec := &recorder{}
	w, err := economysim.NewWorld(c, economysim.WithSeed(11), economysim.WithObserver(rec))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	if err := w.SetFlowScales([]economysim.FlowScale{{Flow: "poll", BP: 3_000}}); err != nil {
		t.Fatalf("SetFlowScales() error = %v, want nil: a held key accepts a consumer's call", err)
	}
	rec.forget()
	policyStep(t, w, 14)
	if s, _ := w.Slots(); flowScale(s, "poll") != 3_000 {
		t.Errorf("poll after 14 ticks = %d, want 3 000: the actor returned no entry for it", flowScale(s, "poll"))
	}
	policyStep(t, w, 16)
	got, ok := firstPolicyMove(rec, "poll")
	if !ok || got.Tick < 15 {
		t.Fatalf("first policy move of poll = %+v (found %v), want one at tick 15 or later", got, ok)
	}
	if got.From != 3_000 || got.To != 8_000 {
		t.Errorf("first policy move of poll = %d to %d, want 3 000 to 8 000", got.From, got.To)
	}
}

// firstPolicyMove returns the first MoneyFlowScaled of the flow with the cause policy.
func firstPolicyMove(rec *recorder, flow string) (economysim.MoneyFlowScaled, bool) {
	for _, m := range recordedOf[economysim.MoneyFlowScaled](rec) {
		if m.Flow == flow && m.Cause == economysim.CausePolicy {
			return m, true
		}
	}
	return economysim.MoneyFlowScaled{}, false
}

// TestPolicy_SlotCallsInsideAStepAreRefused verifies the refusal path through the public
// API: a consumer's slot call from an observer, while the actors apply at P0.3a, returns
// ErrInStep for every setter and for Slots, the step still completes, and the world ends
// in the slots of a twin whose observer made no call.
//
// Requirement: FR-PLC-024, FR-PLC-033
func TestPolicy_SlotCallsInsideAStepAreRefused(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	var w *economysim.World
	var errs []error
	obs := economysim.ObserverFunc(func(e economysim.Event) {
		if _, ok := e.(economysim.PolicyDecided); !ok || len(errs) > 0 {
			return
		}
		_, e1 := w.Slots()
		e2 := w.SetFlowScales([]economysim.FlowScale{{Flow: "poll", BP: 1}})
		e3 := w.SetEdgeStatuses([]economysim.EdgeStatus{{From: 0, To: 1, FactorBP: 1}})
		e4 := w.SetTransitFactors([]economysim.TransitFactor{{From: 0, To: 1, BP: 1}})
		errs = []error{e1, e2, e3, e4}
	})
	w, err := economysim.NewWorld(policyGoverned(), economysim.WithSeed(11), economysim.WithObserver(obs))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	policyStep(t, w, 25) // a refused slot call must not fail a step
	if len(errs) != 4 {
		t.Fatalf("the observer made %d slot calls inside the step, want 4", len(errs))
	}
	for i, err := range errs {
		if !errors.Is(err, economysim.ErrInStep) {
			t.Errorf("slot call %d inside a step = %v, want ErrInStep", i, err)
		}
	}
	twin, _ := policyRun(t, policyGoverned(), 25)
	got, _ := w.Slots()
	want, _ := twin.Slots()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Slots() after refused calls = %v, want %v of the twin", got, want)
	}
}

// TestPolicy_RefusedConsumerCallChangesNothing verifies that a consumer call on a key an
// actor holds, refused by its owner (an unknown edge, a flow scale out of bounds, an
// unknown flow), is refused with the owner's sentinel, emits no event and leaves the slots
// as they were, so the actor's next decision reads the same state.
//
// Requirement: FR-PLC-024, FR-PLC-045
func TestPolicy_RefusedConsumerCallChangesNothing(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		call func(w *economysim.World) error
		want error
	}{
		{"unknown edge", func(w *economysim.World) error {
			return w.SetEdgeStatuses([]economysim.EdgeStatus{{From: 0, To: 0, FactorBP: 10_000}})
		}, economysim.ErrUnknownEdge},
		{"flow scale out of bounds", func(w *economysim.World) error {
			return w.SetFlowScales([]economysim.FlowScale{{Flow: "poll", BP: -1}})
		}, economysim.ErrOutOfBounds},
		{"unknown flow", func(w *economysim.World) error {
			return w.SetFlowScales([]economysim.FlowScale{{Flow: "nowhere", BP: 100}})
		}, economysim.ErrUnknownFlow},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := &recorder{}
			w, err := economysim.NewWorld(policyGoverned(), economysim.WithSeed(11), economysim.WithObserver(rec))
			if err != nil {
				t.Fatalf("NewWorld() error = %v, want nil", err)
			}
			before, _ := w.Slots()
			rec.forget()
			if err := tt.call(w); !errors.Is(err, tt.want) {
				t.Errorf("call = %v, want %v", err, tt.want)
			}
			after, _ := w.Slots()
			if len(rec.events) != 0 || !reflect.DeepEqual(before, after) {
				t.Errorf("after a refused call: %d events, slots %v, want none and %v", len(rec.events), after, before)
			}
		})
	}
}
