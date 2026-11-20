package economysim_test

import (
	"bytes"
	"errors"
	"math/rand/v2"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The end-to-end tests of the consumer side of the policy module through the public API only
// (spec-policy.md 4.5, 8.4, 9.1; plan 3-plc Phases 6 and 7): a consumer's Policy registered
// with WithPolicy on the governed pack, and the player's DirectPolicy. The checks, the
// mailbox and the decoder have unit tests in the internal files of the same area.

// scriptPolicy is a consumer's policy that answers call n with script[n] (nothing after the
// last) and keeps what it was shown.
type scriptPolicy struct {
	script [][]economysim.LeverSetting
	ticks  []int64
	levers [][]economysim.LeverView
}

func (p *scriptPolicy) Decide(v economysim.PolicyView, _ *rand.Rand) economysim.Settings {
	n := len(p.ticks)
	p.ticks = append(p.ticks, v.Tick)
	p.levers = append(p.levers, v.Levers)
	if n >= len(p.script) {
		return economysim.Settings{}
	}
	return economysim.Settings{Levers: p.script[n]}
}

// consumerPack is policyConfig with the crown's actor decided by the consumer "council" (every
// 10 ticks, levers poll = 0 and road = 1 by ID order) and the coast directed every tick, so a
// directive is applied by the next tick. Actor IDs follow the sorted actor IDs: coast 0,
// crown 1, inland 2.
func consumerPack() economysim.Config {
	c := policyConfig()
	crown := &c.Modules.Policy.Actors[0]
	crown.Policy, crown.Schedule = "council", nil
	for i := range c.Modules.Policy.Actors {
		if a := &c.Modules.Policy.Actors[i]; a.Policy == "directed" {
			a.Every = 1
		}
	}
	return c
}

// TestPolicy_ConsumerDecidesOverTicksOnAGovernedPack verifies, as a consumer sees it, that a
// policy registered with WithPolicy is called at each due tick of its actor and no other, is
// shown the levers as they stand, and that each answer is counted by one PolicyDecided
// after TickStarted of its tick, in ascending actor order, with the outcome of each entry.
//
// Arithmetic (poll: bounds 0 to 20 000, step limit 5 000; road: 0 to 20 000, no step limit):
// the first call sets poll 12 000 and road 6 000, both applied (poll
// moves 2 000 from 10 000, inside the step limit); call 1 repeats poll 12 000, unchanged;
// call 2 asks poll 90 000, which the bounds cut to 20 000 and the step limit to 12 000 +
// 5 000 = 17 000, applied, clamped and stepped; call 3 names lever 7, which the actor does
// not hold, not held; the fifth call returns nothing and emits no event.
//
// Requirement: FR-PLC-021, FR-PLC-024, FR-PLC-040, FR-PLC-041, FR-PLC-050, FR-PLC-080, FR-PLC-082, FR-PLC-083
func TestPolicy_ConsumerDecidesOverTicksOnAGovernedPack(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	p := &scriptPolicy{script: [][]economysim.LeverSetting{
		{{Lever: 0, BP: 12_000}, {Lever: 1, BP: 6_000}},
		{{Lever: 0, BP: 12_000}},
		{{Lever: 0, BP: 90_000}},
		{{Lever: 7, BP: 1}},
	}}
	w, rec := policyRun(t, consumerPack(), 50, economysim.WithPolicy("council", p))

	// The due tick of an actor is its rotation offset plus a multiple of Every (spec-policy.md
	// 7.2): the first call falls in ticks 0 to 9, the next ones follow 10 apart.
	if len(p.ticks) != 5 || p.ticks[0] < 0 || p.ticks[0] > 9 {
		t.Fatalf("Decide called at ticks %v, want five calls, the first in 0 to 9", p.ticks)
	}
	for i, tick := range p.ticks {
		if want := p.ticks[0] + 10*int64(i); tick != want {
			t.Errorf("Decide call %d at tick %d, want %d (every 10)", i, tick, want)
		}
	}
	if got := len(p.levers[0]); got != 2 {
		t.Errorf("the view at tick 0 holds %d levers, want the actor's 2", got)
	}
	checkCrownDecisions(t, rec, p.ticks)
	checkConsumerSlots(t, w)
	assertDecidedFollowsTickStarted(t, rec)
}

// checkCrownDecisions checks the PolicyDecided of the consumer's actor (1) against the outcomes
// the script above produces at the ticks the consumer was called.
func checkCrownDecisions(t *testing.T, rec *recorder, ticks []int64) {
	t.Helper()
	var crown []economysim.PolicyDecided
	for _, d := range recordedOf[economysim.PolicyDecided](rec) {
		if d.Actor == 1 {
			crown = append(crown, d)
		}
	}
	want := []economysim.PolicyDecided{
		{Tick: ticks[0], Entries: 2, Applied: 2},
		{Tick: ticks[1], Entries: 1, Unchanged: 1},
		{Tick: ticks[2], Entries: 1, Applied: 1, Clamped: 1, Stepped: 1},
		{Tick: ticks[3], Entries: 1, NotHeld: 1},
	}
	if len(crown) != len(want) {
		t.Fatalf("PolicyDecided of the consumer's actor = %d, want %d (the fifth call returned nothing)", len(crown), len(want))
	}
	for i := range want {
		got := crown[i]
		want[i].Actor, want[i].Jurisdiction = got.Actor, got.Jurisdiction
		if got != want[i] {
			t.Errorf("PolicyDecided #%d = %+v, want %+v", i, got, want[i])
		}
	}
}

// checkConsumerSlots checks the final slots of the consumer's run.
func checkConsumerSlots(t *testing.T, w *economysim.World) {
	t.Helper()
	slots, err := w.Slots()
	if err != nil {
		t.Fatalf("Slots() error = %v, want nil", err)
	}
	if got := flowScale(slots, "poll"); got != 17_000 {
		t.Errorf("Slots() scale of poll = %d, want 17 000", got)
	}
	if len(slots.TransitFactors) != 1 || slots.TransitFactors[0].BP != 6_000 {
		t.Errorf("Slots() transit factors = %v, want the road at 6 000", slots.TransitFactors)
	}
}

// assertDecidedFollowsTickStarted checks that no PolicyDecided precedes the TickStarted of its
// tick and that, within a tick, the actors are in ascending order.
func assertDecidedFollowsTickStarted(t *testing.T, rec *recorder) {
	t.Helper()
	started, last := int64(-1), economysim.ActorID(-1)
	for _, e := range rec.events {
		switch e := e.(type) {
		case economysim.TickStarted:
			started, last = e.Tick, -1
		case economysim.PolicyDecided:
			if e.Tick != started || e.Actor <= last {
				t.Errorf("PolicyDecided of actor %d at tick %d after TickStarted %d and actor %d, want the same tick and ascending actors", e.Actor, e.Tick, started, last)
			}
			last = e.Actor
		}
	}
}

// TestPolicy_RefusedDirectiveLeavesTheWorldUnchanged verifies, through the public API, that
// each refusal of DirectPolicy (an unknown actor, an actor that is not directed, a valid entry
// followed by a bad one) returns its sentinel and leaves the world as it was: the same
// checkpoint bytes, no pending setting in the snapshot and no event.
//
// Requirement: FR-PLC-071, FR-PLC-072
func TestPolicy_RefusedDirectiveLeavesTheWorldUnchanged(t *testing.T) {
	t.Parallel()
	good := economysim.PolicyDirective{Actor: 0, Levers: []economysim.LeverSetting{{Lever: 0, BP: 20_000}}}
	for _, tt := range []struct {
		name string
		d    []economysim.PolicyDirective
		want error
	}{
		{"unknown actor", []economysim.PolicyDirective{{Actor: 9, Levers: good.Levers}}, economysim.ErrUnknownActor},
		{"actor not directed", []economysim.PolicyDirective{{Actor: 2, Levers: good.Levers}}, economysim.ErrNotDirected},
		{"good entry then a bad one", []economysim.PolicyDirective{good, {Actor: 0, Levers: good.Levers}}, economysim.ErrDuplicateID},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec := policyRun(t, consumerPack(), 5, economysim.WithPolicy("council", &scriptPolicy{}))
			before := checkpointBytes(t, w)
			rec.forget()
			if err := w.DirectPolicy(tt.d); !errors.Is(err, tt.want) {
				t.Fatalf("DirectPolicy() error = %v, want %v", err, tt.want)
			}
			if after := checkpointBytes(t, w); !bytes.Equal(before, after) {
				t.Errorf("the checkpoint changed after a refused DirectPolicy, want the same %d bytes", len(before))
			}
			snap, err := w.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() error = %v, want nil", err)
			}
			for _, a := range snap.Actors {
				if a.Pending != 0 {
					t.Errorf("actor %d has %d pending settings after a refused call, want 0", a.Actor, a.Pending)
				}
			}
			if len(rec.events) != 0 {
				t.Errorf("a refused DirectPolicy emitted %d events, want 0", len(rec.events))
			}
		})
	}
}

// checkpointBytes is the marshalled checkpoint of w.
func checkpointBytes(t testing.TB, w *economysim.World) []byte {
	t.Helper()
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	raw, err := ck.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v, want nil", err)
	}
	return raw
}

// TestPolicy_DirectiveIsAppliedByTheNextTick verifies the player's path end to end: a
// directive left between ticks shows as Pending in the snapshot, the actor (due every tick)
// applies it in the next step with one PolicyDecided and the two edge events of its group with
// the cause policy, the mailbox is empty afterwards, and later ticks emit nothing for it.
//
// Requirement: FR-PLC-070, FR-PLC-072, FR-PLC-073, FR-PLC-074, FR-PLC-080
func TestPolicy_DirectiveIsAppliedByTheNextTick(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	w, rec := policyRun(t, consumerPack(), 3, economysim.WithPolicy("council", &scriptPolicy{}))
	if n := len(recordedOf[economysim.PolicyDecided](rec)); n != 0 {
		t.Fatalf("PolicyDecided before any directive = %d, want 0 (the coast is directed and its mailbox empty)", n)
	}
	if err := w.DirectPolicy([]economysim.PolicyDirective{{Actor: 0, Levers: []economysim.LeverSetting{{Lever: 0, BP: 20_000}}}}); err != nil {
		t.Fatalf("DirectPolicy() error = %v, want nil", err)
	}
	if got := pendingOf(t, w, 0); got != 1 {
		t.Errorf("Snapshot().Actors[0].Pending = %d, want 1", got)
	}
	rec.forget()
	policyStep(t, w, 1)
	checkDirectiveApplied(t, rec)
	if got := pendingOf(t, w, 0); got != 0 {
		t.Errorf("Snapshot().Actors[0].Pending = %d after the decision, want 0", got)
	}
	rec.forget()
	policyStep(t, w, 3)
	for _, d := range recordedOf[economysim.PolicyDecided](rec) {
		if d.Actor == 0 {
			t.Errorf("PolicyDecided of the directed actor after the directive was applied = %+v, want none", d)
		}
	}
}

// checkDirectiveApplied checks the events of the step that applied the coast's directive.
func checkDirectiveApplied(t *testing.T, rec *recorder) {
	t.Helper()
	d := recordedOf[economysim.PolicyDecided](rec)
	if len(d) != 1 || d[0].Tick != 3 || d[0].Actor != 0 || d[0].Applied != 1 || d[0].Entries != 1 {
		t.Errorf("PolicyDecided after the next step = %+v, want one at tick 3 for actor 0 with one applied entry", d)
	}
	edges := recordedOf[economysim.EdgeStatusChanged](rec)
	if len(edges) != 2 || edges[0].Cause != economysim.CausePolicy || edges[0].FactorBP != 20_000 {
		t.Errorf("EdgeStatusChanged = %+v, want the two edges of the group at 20 000 with cause policy", edges)
	}
}

// pendingOf is the Pending count of the actor in a snapshot of w.
func pendingOf(t *testing.T, w *economysim.World, actor economysim.ActorID) int64 {
	t.Helper()
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	for _, a := range snap.Actors {
		if a.Actor == actor {
			return a.Pending
		}
	}
	t.Fatalf("Snapshot().Actors has no actor %d: %+v", actor, snap.Actors)
	return 0
}

// panicDecide is a policy whose Decide panics.
type panicDecide struct{}

func (panicDecide) Decide(economysim.PolicyView, *rand.Rand) economysim.Settings {
	panic("consumer policy failed")
}

// TestPolicy_PanickingDecideFailsTheWorld verifies that a panic of a consumer's Decide is not
// swallowed, and that afterwards every call on the world reports ErrWorldFailed, a stable
// sentinel a consumer can test with errors.Is.
//
// Requirement: FR-PLC-024
func TestPolicy_PanickingDecideFailsTheWorld(t *testing.T) {
	t.Parallel()
	w, err := economysim.NewWorld(consumerPack(), economysim.WithSeed(11), economysim.WithPolicy("council", panicDecide{}))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	// The actor is due once in its first 10 ticks, so the panic comes within them.
	var v any
	for range 10 {
		func() {
			defer func() { v = recover() }()
			_ = w.Step()
		}()
		if v != nil {
			break
		}
	}
	if v != "consumer policy failed" {
		t.Fatalf("Step() panicked with %v within 10 ticks, want the policy's own value", v)
	}
	if err := w.Step(); !errors.Is(err, economysim.ErrWorldFailed) {
		t.Errorf("Step() error = %v after the panic, want ErrWorldFailed", err)
	}
	if _, err := w.Checkpoint(); !errors.Is(err, economysim.ErrWorldFailed) {
		t.Errorf("Checkpoint() error = %v after the panic, want ErrWorldFailed", err)
	}
}

// drawPolicy answers with a poll value taken from its stream, so that the draws change the
// world, and keeps the words it drew.
type drawPolicy struct{ words []uint64 }

func (p *drawPolicy) Decide(_ economysim.PolicyView, rng *rand.Rand) economysim.Settings {
	w := rng.Uint64()
	p.words = append(p.words, w)
	return economysim.Settings{Levers: []economysim.LeverSetting{{Lever: 0, BP: int64(w % 20_000)}}}
}

// TestPolicy_ConsumerDrawsAreDeterministicPerSeed verifies that the words a consumer draws, the
// events and the slots they cause are equal over two runs of the same seed, and that another
// seed gives other words: the stream is the world's, keyed, and nothing else.
//
// Requirement: FR-PLC-022, FR-PLC-053, NFR-PLC-004
func TestPolicy_ConsumerDrawsAreDeterministicPerSeed(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	run := func(seed uint64) (*drawPolicy, *recorder, economysim.SlotState) {
		p := &drawPolicy{}
		rec := &recorder{}
		w, err := economysim.NewWorld(consumerPack(), economysim.WithSeed(seed), economysim.WithObserver(rec), economysim.WithPolicy("council", p))
		if err != nil {
			t.Fatalf("NewWorld() error = %v, want nil", err)
		}
		policyStep(t, w, 40)
		s, err := w.Slots()
		if err != nil {
			t.Fatalf("Slots() error = %v, want nil", err)
		}
		return p, rec, s
	}
	p1, r1, s1 := run(11)
	p2, r2, s2 := run(11)
	if !reflect.DeepEqual(p1.words, p2.words) || len(p1.words) != 4 {
		t.Errorf("words of seed 11 = %v then %v, want the same four", p1.words, p2.words)
	}
	if !reflect.DeepEqual(r1.events, r2.events) || !reflect.DeepEqual(s1, s2) {
		t.Errorf("two runs of seed 11 differ in events (%d, %d) or slots", len(r1.events), len(r2.events))
	}
	if p3, _, _ := run(12); reflect.DeepEqual(p1.words, p3.words) {
		t.Errorf("words of seed 12 = %v, want others than those of seed 11", p3.words)
	}
}

// TestPolicy_UnusedRegistrationLeavesNoTrace verifies that a policy registered with WithPolicy
// that no actor names changes nothing: the same events and the same checkpoint bytes, with the
// module off (no Policy table) and with it on (built-in actors only).
//
// Requirement: FR-PLC-001, FR-PLC-023, NFR-PLC-004
func TestPolicy_UnusedRegistrationLeavesNoTrace(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	off := policyGoverned()
	off.Modules.Policy = nil
	for _, tt := range []struct {
		name string
		c    economysim.Config
	}{
		{"module off", off},
		{"built-in actors only", policyGoverned()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			plain, r1 := policyRun(t, tt.c, 25)
			spare, r2 := policyRun(t, tt.c, 25, economysim.WithPolicy("unused", &scriptPolicy{}))
			if !reflect.DeepEqual(r1.events, r2.events) {
				t.Errorf("events with an unused registration (%d) differ from those without (%d)", len(r2.events), len(r1.events))
			}
			if !bytes.Equal(checkpointBytes(t, plain), checkpointBytes(t, spare)) {
				t.Errorf("checkpoint bytes with an unused registration differ from those without, want equal")
			}
		})
	}
}
