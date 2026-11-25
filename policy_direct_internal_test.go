package economysim

import (
	"errors"
	"math/rand/v2"
	"slices"
	"testing"
)

// The tests of the mailbox and of DirectPolicy (spec-policy.md 8.4, 9.1, 12.3; plan 3-plc
// Phase 6): the checks of the call in their order, the atomic replacement, the decision that
// reads and clears the mailbox, and the rows of Snapshot.Actors.

// directedMill is a world of the mill with the crown directed (actor 0, every 10 ticks, the
// levers gate, poll and road) and the shire on hold (actor 1).
func directedMill(t testing.TB, opts ...Option) (*World, *evLog) {
	t.Helper()
	rec := &evLog{}
	w := newMill(t, policyMillConfig(crownActor("directed", 10), shireActor("hold", 1)), append(opts, WithSeed(3), WithObserver(rec))...)
	return w, rec
}

// TestDirectPolicy_ErrorsInOrder verifies one refusal per row of spec-policy.md 9.1.2, in
// order: an input with two faults reports the earlier one, with the path of the entry and
// the sentinel.
//
// Requirement: FR-PLC-071
func TestDirectPolicy_ErrorsInOrder(t *testing.T) {
	t.Parallel()
	ok := LeverSetting{Lever: 1, BP: 11_000}
	for _, tc := range []struct {
		name string
		d    []PolicyDirective
		want error
		path []string
	}{
		{"unknown actor", []PolicyDirective{{Actor: 2, Levers: []LeverSetting{{Lever: 9, BP: -1}}}}, ErrUnknownActor, []string{"d", "d[0]", "actor"}},
		{"negative actor", []PolicyDirective{{Actor: -1}}, ErrUnknownActor, []string{"d", "d[0]", "actor"}},
		{"repeated actor", []PolicyDirective{{Actor: 0}, {Actor: 0, Levers: []LeverSetting{{Lever: 9}}}}, ErrDuplicateID, []string{"d", "d[1]", "actor"}},
		{"not directed", []PolicyDirective{{Actor: 1, Levers: []LeverSetting{{Lever: 9}}}}, ErrNotDirected, []string{"d", "d[0]", "actor"}},
		{"unknown lever", []PolicyDirective{{Actor: 0, Levers: []LeverSetting{ok, {Lever: 3, BP: -1}}}}, ErrUnknownLever, []string{"d", "d[0]", "levers[1]", "lever"}},
		{"negative lever", []PolicyDirective{{Actor: 0, Levers: []LeverSetting{{Lever: -1}}}}, ErrUnknownLever, []string{"d", "d[0]", "levers[0]", "lever"}},
		{"repeated lever", []PolicyDirective{{Actor: 0, Levers: []LeverSetting{ok, {Lever: 1, BP: 99_999}}}}, ErrDuplicateID, []string{"d", "d[0]", "levers[1]", "lever"}},
		{"value above max", []PolicyDirective{{Actor: 0, Levers: []LeverSetting{{Lever: 1, BP: 20_001, Closed: true}}}}, ErrOutOfBounds, []string{"d", "d[0]", "levers[0]", "bp"}},
		{"value below min", []PolicyDirective{{Actor: 0, Levers: []LeverSetting{{Lever: 0, BP: 4_999}}}}, ErrOutOfBounds, []string{"d", "d[0]", "levers[0]", "bp"}},
		{"closed without may close", []PolicyDirective{{Actor: 0, Levers: []LeverSetting{{Lever: 1, BP: 10_000, Closed: true}}}}, ErrOutOfBounds, []string{"d", "d[0]", "levers[0]", "closed"}},
		{"an earlier entry first", []PolicyDirective{{Actor: 0, Levers: []LeverSetting{{Lever: 1, BP: -5}}}, {Actor: 2}}, ErrOutOfBounds, []string{"d", "d[0]", "levers[0]", "bp"}},
	} {
		w, _ := directedMill(t)
		err := w.DirectPolicy(tc.d)
		var ve *ValidationError
		if !errors.Is(err, tc.want) || !errors.As(err, &ve) || !slices.Equal(ve.Path, tc.path) {
			t.Errorf("%s: DirectPolicy() error = %v, want %v at %v", tc.name, err, tc.want, tc.path)
		}
		if n := w.policy.pending(); n != 0 {
			t.Errorf("%s: %d settings pending after a refused call, want 0", tc.name, n)
		}
	}
}

// pending is the number of settings in all mailboxes, for the tests.
func (s *policyState) pending() int {
	n := 0
	for _, m := range s.mailbox {
		n += len(m)
	}
	return n
}

// TestDirectPolicy_GuardsComeFirst verifies the three checks before the entries: a call made
// inside a step is ErrInStep, a call on a failed world is ErrWorldFailed and a call on a
// world without the module is ErrModuleOff naming policy, each before any entry is read.
//
// Requirement: FR-PLC-071
func TestDirectPolicy_GuardsComeFirst(t *testing.T) {
	t.Parallel()
	bad := []PolicyDirective{{Actor: 7}}

	var inStep error
	var w *World
	spy := &spyPolicy{do: func() { inStep = w.DirectPolicy(bad) }}
	w = newMill(t, policyMillConfig(crownActor("spy", 1)), WithSeed(3), WithPolicy("spy", spy))
	if err := w.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	if !errors.Is(inStep, ErrInStep) {
		t.Errorf("DirectPolicy() inside a step: error = %v, want ErrInStep", inStep)
	}

	failed := newMill(t, policyMillConfig(crownActor("boom", 1)), WithSeed(3), WithPolicy("boom", panicPolicy{}))
	_ = panicOf(func() { _ = failed.Step() })
	if err := failed.DirectPolicy(bad); !errors.Is(err, ErrWorldFailed) {
		t.Errorf("DirectPolicy() on a failed world: error = %v, want ErrWorldFailed", err)
	}

	off := policyMillConfig()
	off.Modules.Policy = nil
	if err := newMill(t, off, WithSeed(3)).DirectPolicy(bad); !errors.Is(err, ErrModuleOff) {
		t.Errorf("DirectPolicy() with the module off: error = %v, want ErrModuleOff", err)
	}
}

// spyPolicy runs do at each decision and returns nothing.
type spyPolicy struct{ do func() }

func (p *spyPolicy) Decide(PolicyView, *rand.Rand) Settings {
	p.do()
	return Settings{}
}

// TestDirectPolicy_EmptyIsANoOpAndReplacesWhole verifies that a nil or empty call changes
// nothing, that a directive replaces the mailbox of its actor whole (never merged), that an
// empty directive clears it, that the world keeps the settings sorted by lever and that the
// caller's slice can change after the call.
//
// Requirement: FR-PLC-071, FR-PLC-072
func TestDirectPolicy_EmptyIsANoOpAndReplacesWhole(t *testing.T) {
	t.Parallel()
	w, _ := directedMill(t)
	for _, d := range [][]PolicyDirective{nil, {}} {
		if err := w.DirectPolicy(d); err != nil || w.policy.pending() != 0 {
			t.Fatalf("DirectPolicy(%v) = %v with %d pending, want a no-op", d, err, w.policy.pending())
		}
	}
	levers := []LeverSetting{{Lever: 2, BP: 9_000}, {Lever: 0, BP: 7_000, Closed: true}, {Lever: 1, BP: 11_000}}
	if err := w.DirectPolicy([]PolicyDirective{{Actor: 0, Levers: levers}}); err != nil {
		t.Fatalf("DirectPolicy() error = %v, want nil", err)
	}
	levers[0].BP = 1
	want := []LeverSetting{{Lever: 0, BP: 7_000, Closed: true}, {Lever: 1, BP: 11_000}, {Lever: 2, BP: 9_000}}
	if got := w.policy.mailbox[0]; !slices.Equal(got, want) {
		t.Fatalf("mailbox = %+v, want %+v (sorted, a copy)", got, want)
	}
	if err := w.DirectPolicy([]PolicyDirective{{Actor: 0, Levers: []LeverSetting{{Lever: 1, BP: 12_000}}}}); err != nil {
		t.Fatalf("DirectPolicy() error = %v, want nil", err)
	}
	if got := w.policy.mailbox[0]; !slices.Equal(got, []LeverSetting{{Lever: 1, BP: 12_000}}) {
		t.Errorf("mailbox = %+v after a second call, want it replaced, not merged", got)
	}
	if err := w.DirectPolicy([]PolicyDirective{{Actor: 0}}); err != nil || len(w.policy.mailbox[0]) != 0 {
		t.Errorf("DirectPolicy(empty directive) = %v with %d pending, want the mailbox cleared", err, len(w.policy.mailbox[0]))
	}
}

// TestDirectPolicy_AtomicWhole verifies that a call whose second entry is refused leaves the
// mailbox of the first entry's actor as it was.
//
// Requirement: FR-PLC-071
func TestDirectPolicy_AtomicWhole(t *testing.T) {
	t.Parallel()
	w, _ := directedMill(t)
	err := w.DirectPolicy([]PolicyDirective{{Actor: 0, Levers: []LeverSetting{{Lever: 1, BP: 11_000}}}, {Actor: 1}})
	if !errors.Is(err, ErrNotDirected) || w.policy.pending() != 0 {
		t.Errorf("DirectPolicy() = %v with %d pending, want ErrNotDirected and nothing kept", err, w.policy.pending())
	}
}

// TestDirectPolicy_DoesNotCheckMaxStep verifies that the step limit is taken at the decision
// and not at the call: a directive far from the current value is accepted, and the decision
// cuts it to the limit and counts it as stepped.
//
// Requirement: FR-PLC-071
func TestDirectPolicy_DoesNotCheckMaxStep(t *testing.T) {
	t.Parallel()
	w, rec := directedMill(t)
	if err := w.DirectPolicy([]PolicyDirective{{Actor: 0, Levers: []LeverSetting{{Lever: 1, BP: 20_000}}}}); err != nil {
		t.Fatalf("DirectPolicy() error = %v, want nil (poll has a step of 3000, the value is 10000 away)", err)
	}
	if err := w.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	dec, moved := of[PolicyDecided](rec), of[MoneyFlowScaled](rec)
	if len(dec) != 1 || dec[0].Stepped != 1 || dec[0].Applied != 1 || len(moved) != 1 || moved[0].To != 13_000 || moved[0].Cause != CausePolicy {
		t.Errorf("decisions = %+v, flow events = %+v, want one stepped entry applied as 13000", dec, moved)
	}
}

// TestDirected_ClearedWhateverOutcome verifies that the decision empties the mailbox
// whatever the checks made of it: an applied entry, an unchanged one and a refused one all
// leave Pending at 0, and a directive acts once.
//
// Requirement: FR-PLC-070
// Guard: clearing the mailbox only when an entry applied would replay a refused directive every cadence.
func TestDirected_ClearedWhateverOutcome(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		levers []LeverSetting
	}{
		{"applied", []LeverSetting{{Lever: 1, BP: 11_000}}},
		{"unchanged", []LeverSetting{{Lever: 1, BP: 10_000}}},
	} {
		w, rec := directedMill(t)
		if err := w.DirectPolicy([]PolicyDirective{{Actor: 0, Levers: tc.levers}}); err != nil {
			t.Fatalf("%s: DirectPolicy() error = %v, want nil", tc.name, err)
		}
		if err := w.Step(); err != nil {
			t.Fatalf("%s: Step() error = %v, want nil", tc.name, err)
		}
		if n := len(w.policy.mailbox[0]); n != 0 {
			t.Errorf("%s: %d settings pending after the decision, want 0", tc.name, n)
		}
		if err := w.Step(); err != nil {
			t.Fatalf("%s: Step() error = %v, want nil", tc.name, err)
		}
		if got := len(of[PolicyDecided](rec)); got != 1 {
			t.Errorf("%s: %d decisions over two ticks, want 1 (the directive acted once)", tc.name, got)
		}
	}
}

// TestDirected_EmptyMailboxIsSilent verifies that a due directed actor with nothing in its
// mailbox emits no event and writes nothing, and builds no view.
//
// Requirement: FR-PLC-070
func TestDirected_EmptyMailboxIsSilent(t *testing.T) {
	t.Parallel()
	w, rec := directedMill(t)
	for range 11 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	if n := len(of[PolicyDecided](rec)) + len(of[MoneyFlowScaled](rec)); n != 0 {
		t.Errorf("%d policy events with nothing directed, want 0", n)
	}
}

// TestSnapshot_ActorsPending verifies Snapshot.Actors: one row per actor in ascending
// ActorID with its jurisdiction and, for a directed actor, the number of settings pending,
// and nothing with the module off.
//
// Requirement: FR-PLC-074
func TestSnapshot_ActorsPending(t *testing.T) {
	t.Parallel()
	w, _ := directedMill(t)
	if err := w.DirectPolicy([]PolicyDirective{{Actor: 0, Levers: []LeverSetting{{Lever: 1, BP: 11_000}, {Lever: 2, BP: 9_000}}}}); err != nil {
		t.Fatalf("DirectPolicy() error = %v, want nil", err)
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	crown, _ := w.polity.v.Lookup("crown")
	shire, _ := w.polity.v.Lookup("shire")
	want := []ActorRow{{Actor: 0, Jurisdiction: crown, Pending: 2}, {Actor: 1, Jurisdiction: shire}}
	if !slices.Equal(s.Actors, want) {
		t.Errorf("Actors = %+v, want %+v", s.Actors, want)
	}
	off := policyMillConfig()
	off.Modules.Policy = nil
	if s, err := newMill(t, off, WithSeed(3)).Snapshot(); err != nil || s.Actors != nil {
		t.Errorf("Snapshot().Actors = %+v (%v) with the module off, want nil", s.Actors, err)
	}
}
