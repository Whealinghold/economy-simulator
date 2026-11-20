package economysim

import (
	"errors"
	"testing"
)

// The tests of hostile input to the policy module (spec-policy.md 6.2, 9.1.2): what a
// consumer's policy may return and what a call of DirectPolicy may carry, in the sizes a
// careless or hostile caller can send.

// hostileBatch is n settings of which the first is valid and the rest name a lever the
// actor does not hold.
func hostileBatch(n int) []LeverSetting {
	b := make([]LeverSetting, n)
	b[0] = LeverSetting{Lever: 1, BP: 12_000}
	for i := 1; i < n; i++ {
		b[i] = LeverSetting{Lever: LeverID(7 + i%3), BP: -1}
	}
	return b
}

// TestDecide_HostileConsumerSettingsAreCountedNotFatal verifies that a consumer's policy that
// returns a hundred thousand settings, almost all of them naming a lever the actor does not
// hold, fails no step: the unknown ones are counted as not held, the valid one is applied, and
// the tally adds up.
//
// Requirement: FR-PLC-024
// Guard: the checks are linear in the entries and never index with a lever the consumer made up.
func TestDecide_HostileConsumerSettingsAreCountedNotFatal(t *testing.T) {
	t.Parallel()
	const n = 100_000
	p := &scriptedPolicy{set: Settings{Levers: hostileBatch(n)}}
	rec := &evLog{}
	w := newMill(t, policyMillConfig(crownActor("council", 1)), WithSeed(3), WithPolicy("council", p), WithObserver(rec))
	if err := w.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	ds := of[PolicyDecided](rec)
	if len(ds) != 1 {
		t.Fatalf("PolicyDecided events = %d, want 1", len(ds))
	}
	d := ds[0]
	// One valid entry and n-1 for levers 7 to 9, none of which exists (the mill's actor holds 3).
	if d.Entries != n || d.NotHeld != n-1 || d.Applied+d.Unchanged != 1 {
		t.Errorf("PolicyDecided = entries %d, not held %d, applied %d, unchanged %d, want %d, %d and one of the last two", d.Entries, d.NotHeld, d.Applied, d.Unchanged, n, n-1)
	}
}

// TestDirectPolicy_HugeBatchesFailFast verifies that a call with a hundred thousand entries,
// and one entry with a hundred thousand settings, is refused at the first repeat or unknown
// name, with nothing pending: the quadratic scans for repeats cannot run past the number of
// actors or levers the world holds.
//
// Requirement: FR-PLC-071
// Guard: a valid prefix is at most as long as the actors (MaxActors) or levers, so the
// repeat scans stay small whatever the caller sends; a change that scans before it checks
// the range would make this call take seconds.
func TestDirectPolicy_HugeBatchesFailFast(t *testing.T) {
	t.Parallel()
	const n = 100_000
	w, _ := directedMill(t)
	many := make([]PolicyDirective, n)
	for _, tc := range []struct {
		name string
		d    []PolicyDirective
		want error
	}{
		{"every entry the same actor", many, ErrDuplicateID},
		{"one entry, one lever repeated", []PolicyDirective{{Actor: 0, Levers: repeated(LeverSetting{Lever: 1, BP: 11_000}, n)}}, ErrDuplicateID},
		{"one entry, unknown levers", []PolicyDirective{{Actor: 0, Levers: hostileBatch(n)}}, ErrUnknownLever},
	} {
		if err := w.DirectPolicy(tc.d); !errors.Is(err, tc.want) {
			t.Errorf("%s: DirectPolicy() error = %v, want %v", tc.name, err, tc.want)
		}
		if got := w.policy.pending(); got != 0 {
			t.Errorf("%s: %d settings pending after a refused call, want 0", tc.name, got)
		}
	}
}

// repeated is n copies of s.
func repeated(s LeverSetting, n int) []LeverSetting {
	out := make([]LeverSetting, n)
	for i := range out {
		out[i] = s
	}
	return out
}
