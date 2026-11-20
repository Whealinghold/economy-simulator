package economysim

import (
	"errors"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// The tests of a consumer's policy (spec-policy.md 4.5, 4.6, 7.3; plan 3-plc Phase 6):
// registration by name, the view it is given, the stream of its key and what a panic does.

// panicOf runs f and returns the value it panicked with, or nil.
func panicOf(f func()) (v any) {
	defer func() { v = recover() }()
	f()
	return nil
}

// drawingPolicy is a consumer's policy that records, per call, the actor, the tick and the
// first words of the stream it was handed, and draws extra words for the actors in extra.
type drawingPolicy struct {
	extra map[ActorID]int
	calls []drawn
}

type drawn struct {
	tick  int64
	actor ActorID
	word  uint64
}

func (p *drawingPolicy) Decide(v PolicyView, rng *rand.Rand) Settings {
	p.calls = append(p.calls, drawn{tick: v.Tick, actor: v.Actor, word: rng.Uint64()})
	for range p.extra[v.Actor] {
		rng.Uint64()
	}
	return Settings{}
}

// keyedWord is the first word of the stream (sitePolicy, tick, actor) of a world with seed.
func keyedWord(seed uint64, tick int64, actor ActorID) uint64 {
	hi, lo := ordering.StreamKey(seed, sitePolicy, tick, int64(actor))
	return rand.New(rand.NewPCG(hi, lo)).Uint64()
}

// TestWithPolicy_ConsumerIsCalledAndItsSettingsApplied verifies that an actor whose policy is
// a registered name is decided by it at its due ticks, and that what it returns goes through
// the checks and the owners' setters with the cause policy.
//
// Requirement: FR-PLC-021
func TestWithPolicy_ConsumerIsCalledAndItsSettingsApplied(t *testing.T) {
	t.Parallel()
	p := &scriptedPolicy{set: Settings{Levers: []LeverSetting{{Lever: 1, BP: 12_000}}}}
	rec := &evLog{}
	w := newMill(t, policyMillConfig(crownActor("council", 2)), WithSeed(3), WithPolicy("council", p), WithObserver(rec))
	for range 3 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	if p.view.Actor != 0 || p.view.Tick != 2 || len(p.view.Levers) != 3 {
		t.Errorf("last view = actor %d tick %d with %d levers, want actor 0 at tick 2 with 3 levers", p.view.Actor, p.view.Tick, len(p.view.Levers))
	}
	moved := of[MoneyFlowScaled](rec)
	if len(moved) != 1 || moved[0].Cause != CausePolicy || moved[0].To != 12_000 {
		t.Errorf("MoneyFlowScaled = %+v, want one to 12000 with the cause policy", moved)
	}
}

// TestNewWorld_UnknownPolicy verifies that an actor that names a policy nobody registered is
// ErrUnknownPolicy at its path, raised after Validate succeeded.
//
// Requirement: FR-PLC-021
func TestNewWorld_UnknownPolicy(t *testing.T) {
	t.Parallel()
	_, err := NewWorld(policyMillConfig(crownActor("council", 2)), WithSeed(3))
	var ve *ValidationError
	if !errors.Is(err, ErrUnknownPolicy) || !errors.As(err, &ve) || !slices.Equal(ve.Path, policyActorPath(0, "policy")) {
		t.Errorf("NewWorld() error = %v, want ErrUnknownPolicy at modules.policy.actors[0].policy", err)
	}
}

// TestWithPolicy_RegistrationErrors verifies that a reserved name, a repeat and a nil policy
// are refused by the option, in that order of kind.
//
// Requirement: FR-PLC-021
func TestWithPolicy_RegistrationErrors(t *testing.T) {
	t.Parallel()
	c := policyMillConfig(crownActor("hold", 2))
	for _, tc := range []struct {
		name string
		opts []Option
		want error
	}{
		{"reserved name", []Option{WithPolicy("rule", HoldPolicy{})}, ErrDuplicateID},
		{"repeated name", []Option{WithPolicy("council", HoldPolicy{}), WithPolicy("council", HoldPolicy{})}, ErrDuplicateID},
		{"nil policy", []Option{WithPolicy("council", nil)}, ErrNilExtension},
		{"bad name", []Option{WithPolicy("Bad Name", HoldPolicy{})}, ErrInvalidID},
	} {
		if _, err := NewWorld(c, tc.opts...); !errors.Is(err, tc.want) {
			t.Errorf("%s: NewWorld() error = %v, want %v", tc.name, err, tc.want)
		}
	}
}

// TestView_LeversAreACopyForConsumers verifies that a consumer's policy is handed its own
// copy of the levers' rows: it may change them or keep them, and neither the world's scratch
// nor the next view shows it.
//
// Requirement: FR-PLC-025
// Guard: handing the world's scratch slice to a consumer lets it corrupt the inputs of the checks.
func TestView_LeversAreACopyForConsumers(t *testing.T) {
	t.Parallel()
	p := &scriptedPolicy{}
	w := newMill(t, policyMillConfig(crownActor("council", 1)), WithSeed(3), WithPolicy("council", p))
	if err := w.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	kept := p.view.Levers
	if len(kept) == 0 || &kept[0] == &w.policy.scratch.levers[0] {
		t.Fatalf("the consumer's levers share the world's scratch (len %d), want a copy", len(kept))
	}
	want := slices.Clone(w.policy.scratch.levers)
	for i := range kept {
		kept[i] = LeverView{Min: -1, Max: -1, BP: -1}
	}
	if !slices.Equal(w.policy.scratch.levers, want) {
		t.Errorf("scratch levers = %+v after the consumer changed its view, want %+v", w.policy.scratch.levers, want)
	}
	if err := w.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	if p.view.Levers[0].Max != 20_000 || p.view.Tick != 1 {
		t.Errorf("next view = %+v at tick %d, want a fresh view with the limits of the lever", p.view.Levers[0], p.view.Tick)
	}
}

// TestDraws_BuiltinsTakeNone verifies that a due built-in policy never keys the stream: the
// source holds the state it had before P0.3a.
//
// Requirement: FR-PLC-053
func TestDraws_BuiltinsTakeNone(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"hold", "schedule", "rule", "directed"} {
		a := crownActor(name, 1)
		if name == "schedule" {
			a.Schedule = []ScheduleStep{{Lever: "poll", At: 0, BP: 12_000}}
		}
		if name == "rule" {
			a.Rules = []Reaction{{Lever: "poll", Gauge: GaugeTreasury, Target: 1, Span: 1, Move: 100}}
		}
		w := newMill(t, policyMillConfig(a), WithSeed(3))
		if err := w.DirectPolicy(directFor(name)); err != nil {
			t.Fatalf("%s: DirectPolicy() error = %v, want nil", name, err)
		}
		w.rnd.keyed(sitePolicy, 99, 99)
		before, _ := w.rnd.pcg.MarshalBinary()
		w.p03aPolicy()
		after, _ := w.rnd.pcg.MarshalBinary()
		if !slices.Equal(before, after) {
			t.Errorf("%s: the stream changed during P0.3a, want no key and no word from a built-in", name)
		}
	}
}

// directFor is a directive for the directed case of TestDraws_BuiltinsTakeNone and none for
// the others.
func directFor(name string) []PolicyDirective {
	if name != "directed" {
		return nil
	}
	return []PolicyDirective{{Actor: 0, Levers: []LeverSetting{{Lever: 1, BP: 11_000}}}}
}

// TestDraws_ConsumerPolicyKeyedByActor verifies that the stream a consumer's Decide receives is
// the one of (sitePolicy, tick, ActorID): its first word is the first word of that key, for
// each due actor.
//
// Requirement: FR-PLC-022, FR-PLC-053
func TestDraws_ConsumerPolicyKeyedByActor(t *testing.T) {
	t.Parallel()
	p := &drawingPolicy{}
	w := newMill(t, policyMillConfig(crownActor("council", 1), shireActor("council", 1)), WithSeed(5), WithPolicy("council", p))
	for range 3 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	if len(p.calls) != 6 {
		t.Fatalf("calls = %d, want 6 (two actors, three ticks)", len(p.calls))
	}
	for _, c := range p.calls {
		if want := keyedWord(w.rnd.seed, c.tick, c.actor); c.word != want {
			t.Errorf("tick %d actor %d: first word = %#x, want %#x of the key (sitePolicy, tick, actor)", c.tick, c.actor, c.word, want)
		}
	}
}

// TestDraws_PolicyKeyedNotPositional verifies that an actor's draws depend on its key and on no
// other draw: the words the first actor takes do not move the words of the second.
//
// Requirement: FR-PLC-053
// Guard: one stream positioned by call order would shift the second actor's words with the first's count.
func TestDraws_PolicyKeyedNotPositional(t *testing.T) {
	t.Parallel()
	run := func(extra int) []drawn {
		p := &drawingPolicy{extra: map[ActorID]int{0: extra}}
		w := newMill(t, policyMillConfig(crownActor("council", 1), shireActor("council", 1)), WithSeed(5), WithPolicy("council", p))
		for range 2 {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() error = %v, want nil", err)
			}
		}
		return p.calls
	}
	a, b := run(0), run(7)
	if !slices.Equal(a, b) {
		t.Errorf("words with 0 extra draws = %+v, with 7 = %+v, want the same: keyed, not positional", a, b)
	}
}

// panicPolicy panics in Decide.
type panicPolicy struct{}

func (panicPolicy) Decide(PolicyView, *rand.Rand) Settings { panic("policy panic") }

// TestDecide_PanicFailsWorld verifies that a panic of a consumer's Decide propagates and
// leaves the world failed, as any extension's does.
//
// Requirement: FR-PLC-024
func TestDecide_PanicFailsWorld(t *testing.T) {
	t.Parallel()
	w := newMill(t, policyMillConfig(crownActor("council", 1)), WithSeed(3), WithPolicy("council", panicPolicy{}))
	if v := panicOf(func() { _ = w.Step() }); v != "policy panic" {
		t.Fatalf("Step() panicked with %v, want the policy's panic", v)
	}
	if err := w.Step(); !errors.Is(err, ErrWorldFailed) {
		t.Errorf("Step() error = %v after the panic, want ErrWorldFailed", err)
	}
}
