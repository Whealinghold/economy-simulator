package economysim_test

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// fiscalPackTag is the checkpoint section tag of the fiscal module (FR-FIS-060).
const fiscalPackTag = 30

// fiscalEvents steps a world of the fiscal pack for ticks steps with an observer and
// returns it with the FiscalFlowApplied events it saw, in order, and the treasury
// money before the first step.
func fiscalEvents(t *testing.T, cfg economysim.Config, seed uint64, ticks int) (*economysim.World, []economysim.FiscalFlowApplied, economysim.Money) {
	t.Helper()
	rec := &recorder{}
	w := mustWorld(t, cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(rec))
	s0, err := w.Snapshot()
	if err != nil || len(s0.Jurisdictions) == 0 {
		t.Fatalf("Snapshot() before the first step = %d treasuries, %v, want at least one", len(s0.Jurisdictions), err)
	}
	stepBoth(t, ticks, w)
	var evs []economysim.FiscalFlowApplied
	for _, e := range rec.events {
		if f, ok := e.(economysim.FiscalFlowApplied); ok {
			evs = append(evs, f)
		}
	}
	return w, evs, s0.Jurisdictions[0].Money
}

// taxEventFault returns what is wrong with e as the tax of the flow poll that follows
// an event of tick last, or "" when nothing is.
func taxEventFault(e economysim.FiscalFlowApplied, last int64) string {
	switch {
	case e.Flow != "poll" || e.Kind != economysim.KindTax:
		return "want the tax of the flow poll"
	case e.Tick <= last:
		return "want one event per tick in ascending order"
	case e.Applied < 0 || e.Applied > e.Due:
		return "want 0 <= Applied <= Due"
	}
	return partiesFault(e)
}

// partiesFault returns what is wrong with the counts of e, or "".
func partiesFault(e economysim.FiscalFlowApplied) string {
	if e.Parties <= 0 || e.Short < 0 || e.Short > e.Parties {
		return "want Parties > 0 and 0 <= Short <= Parties"
	}
	if e.Short == 0 && e.Applied != e.Due {
		return "want Applied = Due when no payer fell short"
	}
	return ""
}

// stepBoth steps every world together.
func stepBoth(t *testing.T, n int, ws ...*economysim.World) {
	t.Helper()
	for range n {
		for _, w := range ws {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() error = %v, want nil", err)
			}
		}
	}
}

// flowFaults returns what is wrong with the tax events of evs (in order) and with the
// payout events, one message each.
func flowFaults(evs, payouts []economysim.FiscalFlowApplied) []string {
	var out []string
	last := int64(-1)
	for _, e := range evs {
		if e.Kind != economysim.KindTax {
			continue
		}
		if fault := taxEventFault(e, last); fault != "" {
			out = append(out, fmt.Sprintf("event %+v: %s", e, fault))
		}
		last = e.Tick
	}
	for _, e := range payouts {
		if e.Flow != "alms" || e.Applied < 0 || e.Applied > e.Due {
			out = append(out, fmt.Sprintf("payout event %+v, want the flow alms and 0 <= Applied <= Due", e))
		}
	}
	return out
}

// TestFiscal_PackTaxIsCollectedAndRemitted verifies, end to end on the governed warm
// five with the fiscal file layer, that the head tax and the payout are reported and
// that what is reported is what happened: every tax event is of the flow "poll" of the
// root, in ascending tick order and at most once per tick, with Applied no more than
// Due and Due less Applied no more than what the payers fell short; every payout event
// is of the flow "alms" with Applied no more than Due; and the root's treasury ends at
// exactly its start money plus the taxes and the duties applied less the payouts applied
// (the standing orders of the overlay are not built, so nothing else moves it). The audit, on every
// tick, is what proves money is conserved: a tax or a payout moves money between
// sectors and mints none.
//
// Arithmetic: the treasury starts at s0 and ends at s0 + sum(tax Applied) - sum(payout
// Applied); the test compares integers with no tolerance (testing.md 6.3).
//
// Requirement: FR-FIS-002, FR-FIS-021, FR-FIS-024, FR-FIS-039, FR-FIS-044
//
// Guard: a flow collected or paid without its event, or an event reporting money that
// did not move, leaves the observer's rebuild of the treasury wrong and passes every
// unit test of the step on its own.
func TestFiscal_PackTaxIsCollectedAndRemitted(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	const seed, ticks = 1, 30
	w, evs, start := fiscalEvents(t, fiscalPackConfig(t, false), seed, ticks)
	taxed, paid, payouts := fiscalFlowSums(evs)
	if taxed <= 0 || len(payouts) == 0 {
		t.Fatalf("taxes applied = %d and payout events = %d over %d ticks (seed %d), want both positive", taxed, len(payouts), ticks, seed)
	}
	for _, fault := range flowFaults(evs, payouts) {
		t.Errorf("%s (seed %d)", fault, seed)
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if got, want := snap.Jurisdictions[0].Money, start+taxed-paid; got != want {
		t.Errorf("treasury after %d ticks (seed %d) = %d, want %d = start %d + taxed and dutied %d - paid %d", ticks, seed, got, want, start, taxed, paid)
	}
}

// TestFiscal_PackDeterminismAndNonInterference verifies that the pack with the module
// on gives the same events and the same snapshot on two runs of one seed, and the same
// snapshot with the observer off: the fiscal step reports through the observer and
// never reads it back (NFR-FIS-004, NFR-FIS-005).
//
// Requirement: NFR-FIS-004, NFR-FIS-005
//
// Guard: an event built from map order or a step that branched on the presence of an
// observer would differ only on some runs, or only when observed.
func TestFiscal_PackDeterminismAndNonInterference(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	const seed, ticks = 3, 20
	cfg := fiscalPackConfig(t, false)
	w1, ev1, _ := fiscalEvents(t, cfg, seed, ticks)
	w2, ev2, _ := fiscalEvents(t, cfg, seed, ticks)
	if !reflect.DeepEqual(ev1, ev2) {
		t.Errorf("FiscalFlowApplied events of two runs of seed %d differ, want equal", seed)
	}
	bare := runPack(t, cfg, runSpec{Seed: seed, Ticks: ticks, Audit: true, Drive: driveStep, Snapshots: snapNone})
	want, _ := w1.Snapshot()
	again, _ := w2.Snapshot()
	got, err := bare.World.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() of the unobserved world error = %v, want nil", err)
	}
	if !reflect.DeepEqual(want, again) {
		t.Errorf("snapshots of two runs of seed %d differ, want equal", seed)
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("snapshot with an observer differs from the one without (seed %d), want equal", seed)
	}
}

// TestFiscal_PackCheckpointContinuesTheEvents verifies the round trip of tag 30 on the
// pack: the checkpoint after 10 ticks carries the section, and the restored world
// emits, in the 10 ticks that follow, exactly the FiscalFlowApplied events the
// uninterrupted world emits and ends on the same checkpoint bytes (the carries and the
// scales are in the bytes).
//
// Requirement: FR-FIS-044, FR-FIS-060
//
// Guard: the pack's snapshot test compares money only; a carry restored one account off
// or reset to zero changes the due of a later tick, which only the events show.
func TestFiscal_PackCheckpointContinuesTheEvents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := fiscalPackConfig(t, false)
	recA := &recorder{}
	a := mustWorld(t, cfg, economysim.WithSeed(2), economysim.WithAudit(), economysim.WithObserver(recA))
	stepBoth(t, 10, a)
	cp := mustCheckpoint(t, a)
	if !slices.ContainsFunc(decodeFrame(t, mustMarshal(t, cp)).sections, func(s section) bool { return s.tag == fiscalPackTag }) {
		t.Fatalf("checkpoint of the pack with the module on has no section %d, want one", fiscalPackTag)
	}
	recB := &recorder{}
	b, err := economysim.RestoreWorld(cfg, cp, economysim.WithAudit(), economysim.WithObserver(recB))
	if err != nil {
		t.Fatalf("RestoreWorld(module on) error = %v, want nil", err)
	}
	recA.forget()
	recB.forget()
	stepBoth(t, 10, a, b)
	if !reflect.DeepEqual(recA.events, recB.events) {
		t.Errorf("events of the restored world over 10 ticks (%d) differ from the uninterrupted world's (%d), want equal", len(recB.events), len(recA.events))
	}
	ca, cb := mustMarshal(t, mustCheckpoint(t, a)), mustMarshal(t, mustCheckpoint(t, b))
	if !bytes.Equal(ca, cb) {
		t.Errorf("checkpoint bytes of the restored world (%d) differ from the uninterrupted world's (%d), want equal", len(cb), len(ca))
	}
}

// TestFiscal_PackCheckpointHostileBytes verifies that a checkpoint of the pack whose
// section 30 is damaged (and whose digest is made right again, so the frame is valid) is
// refused by RestoreWorld as ErrCheckpointCorrupt and never panics, that a frame cut
// short or with a flipped byte is refused by UnmarshalBinary, and that the same bytes
// restored under the pack with the module off are ErrCheckpointConfig.
//
// Requirement: FR-FIS-061, FR-FIS-060
//
// Guard: the loader of tag 30 reads counts and account IDs from the bytes; a section
// that trusted them would panic or restore a carry on an account that cannot hold one.
func TestFiscal_PackCheckpointHostileBytes(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := fiscalPackConfig(t, false)
	w := mustWorld(t, cfg, economysim.WithSeed(1), economysim.WithAudit())
	stepBoth(t, 10, w)
	raw := mustMarshal(t, mustCheckpoint(t, w))
	if r, err := economysim.RestoreWorld(cfg, reframe(t, raw, func(s []section) []section { return s })); err != nil || r == nil {
		t.Fatalf("RestoreWorld(unedited) = %v, %v, want a world", r, err)
	}
	for _, tt := range []struct {
		name string
		edit func(b []byte) []byte
	}{
		{"a trailing byte", func(b []byte) []byte { return append(b, 0) }},
		{"an empty section", func(b []byte) []byte { return nil }},
		{"a count far beyond the bytes", func(b []byte) []byte { return []byte{0xff, 0xff, 0xff, 0xff, 0x0f} }},
		{"all bytes set", func(b []byte) []byte { return bytes.Repeat([]byte{0xff}, len(b)+1) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := reframe(t, raw, func(s []section) []section { return editSection(s, fiscalPackTag, tt.edit) })
			if _, err := economysim.RestoreWorld(cfg, c); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
				t.Errorf("RestoreWorld(section %d with %s) error = %v, want ErrCheckpointCorrupt", fiscalPackTag, tt.name, err)
			}
		})
	}
	t.Run("cut short", func(t *testing.T) {
		t.Parallel()
		var c economysim.Checkpoint
		if err := c.UnmarshalBinary(raw[:len(raw)/2]); err == nil {
			t.Error("UnmarshalBinary(half of the frame) error = nil, want an error")
		}
	})
	t.Run("a flipped byte", func(t *testing.T) {
		t.Parallel()
		bad := slices.Clone(raw)
		bad[len(bad)/2] ^= 0xff
		var c economysim.Checkpoint
		if err := c.UnmarshalBinary(bad); err == nil {
			t.Error("UnmarshalBinary(frame with a flipped byte) error = nil, want an error")
		}
	})
	t.Run("module off", func(t *testing.T) {
		t.Parallel()
		var c economysim.Checkpoint
		if err := c.UnmarshalBinary(raw); err != nil {
			t.Fatalf("UnmarshalBinary(raw) error = %v, want nil", err)
		}
		if _, err := economysim.RestoreWorld(fiscalPackConfig(t, true), c); !errors.Is(err, economysim.ErrCheckpointConfig) {
			t.Errorf("RestoreWorld(module off) error = %v, want ErrCheckpointConfig", err)
		}
	})
}
