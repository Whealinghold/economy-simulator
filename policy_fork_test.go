package economysim_test

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/codec"
)

// The fork test and the restore refusals of tag 32 through the public API and a real frame
// (spec-policy.md 13, 14.4; plan 3-plc Phase 7): a world restored with a directive pending
// continues as the uninterrupted world does, and each check of the decoder refuses a section
// that a real checkpoint was edited to carry.

// directedPack is policyConfig with the coast and the crown both directed (actors 0 and 1;
// the inland actor 2 is a rule), each every 10 ticks. The coast's only lever is the edge
// group gate (bounds 10 000 to 40 000, may close); the crown's are poll (0 to 20 000) and
// road, neither of which may close.
func directedPack() economysim.Config {
	c := policyConfig()
	crown := &c.Modules.Policy.Actors[0]
	crown.Policy, crown.Schedule = "directed", nil
	return c
}

// pendingDirectives runs c to tick 14, leaves one directive for each directed actor, runs to
// tick 17 and returns the world, its recorder (emptied) and its checkpoint.
func pendingDirectives(t *testing.T, c economysim.Config) (*economysim.World, *recorder, economysim.Checkpoint) {
	t.Helper()
	w, orig := policyRun(t, c, 14)
	d := []economysim.PolicyDirective{
		{Actor: 0, Levers: []economysim.LeverSetting{{Lever: 0, BP: 30_000, Closed: true}}},
		{Actor: 1, Levers: []economysim.LeverSetting{{Lever: 0, BP: 9_000}, {Lever: 1, BP: 7_000}}},
	}
	if err := w.DirectPolicy(d); err != nil {
		t.Fatalf("DirectPolicy() error = %v, want nil", err)
	}
	policyStep(t, w, 3)
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	orig.forget() // the original is compared from the checkpoint on
	return w, orig, ck
}

// TestPolicy_ForkWithAPendingDirectiveEqualsTheUninterruptedRun verifies, through the public
// API, that a checkpoint taken with a directive pending, restored into a fresh world, applies
// it as the original does: the same events from the restore on, the same slots and the same
// checkpoint bytes at the end, and the pending count is carried across.
//
// Arithmetic: the directive is left at tick 14 and the checkpoint taken at tick 17; the
// actors decide every 10 ticks on their rotation offset, so each is due once in ticks 17 to
// 26 and the restored run applies it there; both worlds run to tick 40.
//
// Requirement: FR-PLC-055, FR-PLC-073, FR-PLC-090
func TestPolicy_ForkWithAPendingDirectiveEqualsTheUninterruptedRun(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	c := directedPack()
	w, orig, ck := pendingDirectives(t, c)
	rec := &recorder{}
	r, err := economysim.RestoreWorld(c, ck, economysim.WithObserver(rec), economysim.WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	if a, b := pendingOf(t, r, 0), pendingOf(t, w, 0); a != 1 || b != 1 {
		t.Errorf("Pending of the coast = %d restored and %d original, want 1 and 1", a, b)
	}
	rec.forget() // the WorldRestored event of the restore itself
	policyStep(t, w, 23)
	policyStep(t, r, 23)
	if !reflect.DeepEqual(rec.events, orig.events) || len(recordedOf[economysim.PolicyDecided](rec)) < 2 {
		t.Errorf("events after the restore = %d, of the original %d, want equal with both directives applied", len(rec.events), len(orig.events))
	}
	if !bytes.Equal(checkpointBytes(t, w), checkpointBytes(t, r)) {
		t.Errorf("the checkpoint at tick 40 of the uninterrupted run differs from that of the restored run, want the same bytes")
	}
	if a, b := pendingOf(t, r, 0), pendingOf(t, r, 1); a != 0 || b != 0 {
		t.Errorf("Pending at tick 40 = %d and %d, want 0 and 0 (both directives applied)", a, b)
	}
}

// policyBodyOf is a body of tag 32 written by fill.
func policyBodyOf(fill func(cw *codec.Writer)) []byte {
	cw := codec.NewWriter(nil)
	fill(cw)
	return cw.Out()
}

// policyRowOf writes one row of tag 32: the actor, the count and each setting.
func policyRowOf(cw *codec.Writer, actor uint64, settings ...economysim.LeverSetting) {
	cw.U(actor)
	cw.Count(len(settings))
	for _, s := range settings {
		cw.U(uint64(s.Lever))
		cw.U(uint64(s.BP))
		cw.B(s.Closed)
	}
}

// TestCheckpoint_PolicyRestoreRefusesEachCheck verifies, one subtest per check of spec-policy.md
// 13.2 and one per way a body can end badly, that a real checkpoint whose section of tag 32
// was replaced by a body that breaks that check is refused with ErrCheckpointCorrupt, and that
// the same edit with a good body restores.
//
// Requirement: FR-PLC-090
func TestCheckpoint_PolicyRestoreRefusesEachCheck(t *testing.T) {
	t.Parallel()
	c := directedPack()
	raw := policyCheckpointBytes(t, c)
	gate := economysim.LeverSetting{Lever: 0, BP: 20_000}
	good := policyBodyOf(func(cw *codec.Writer) {
		policyRowOf(cw, 0, gate)
		policyRowOf(cw, 1, economysim.LeverSetting{Lever: 0, BP: 9_000}, economysim.LeverSetting{Lever: 1, BP: 7_000})
	})
	restore := func(body []byte) error {
		ck := reframe(t, raw, func(secs []section) []section {
			return editSection(secs, 32, func([]byte) []byte { return body })
		})
		_, err := economysim.RestoreWorld(c, ck)
		return err
	}
	if err := restore(good); err != nil {
		t.Fatalf("RestoreWorld() with a good body error = %v, want nil", err)
	}
	for _, tt := range []struct {
		name string
		body []byte
	}{
		{"actor out of range", policyBodyOf(func(cw *codec.Writer) { policyRowOf(cw, 3, gate) })},
		{"actor not directed", policyBodyOf(func(cw *codec.Writer) { policyRowOf(cw, 2, gate) })},
		{"actor repeated", policyBodyOf(func(cw *codec.Writer) { policyRowOf(cw, 0, gate); policyRowOf(cw, 0, gate) })},
		{"actors descending", policyBodyOf(func(cw *codec.Writer) { policyRowOf(cw, 1, gate); policyRowOf(cw, 0, gate) })},
		{"row with no setting", policyBodyOf(func(cw *codec.Writer) { policyRowOf(cw, 0) })},
		{"lever out of range", policyBodyOf(func(cw *codec.Writer) { policyRowOf(cw, 0, economysim.LeverSetting{Lever: 1, BP: 20_000}) })},
		{"lever repeated", policyBodyOf(func(cw *codec.Writer) { policyRowOf(cw, 1, gate, gate) })},
		{"levers descending", policyBodyOf(func(cw *codec.Writer) {
			policyRowOf(cw, 1, economysim.LeverSetting{Lever: 1, BP: 1}, economysim.LeverSetting{Lever: 0, BP: 1})
		})},
		{"value above the maximum", policyBodyOf(func(cw *codec.Writer) { policyRowOf(cw, 0, economysim.LeverSetting{Lever: 0, BP: 40_001}) })},
		{"value below the minimum", policyBodyOf(func(cw *codec.Writer) { policyRowOf(cw, 0, economysim.LeverSetting{Lever: 0, BP: 9_999}) })},
		{"closed on a lever that may not close", policyBodyOf(func(cw *codec.Writer) {
			policyRowOf(cw, 1, economysim.LeverSetting{Lever: 0, BP: 1, Closed: true})
		})},
		{"count beyond the bytes", policyBodyOf(func(cw *codec.Writer) { cw.U(0); cw.U(100) })},
		{"body ends inside a row", good[:len(good)-1]},
		{"trailing byte", append(slices.Clone(good), 0)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := restore(tt.body); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
				t.Errorf("RestoreWorld() error = %v, want ErrCheckpointCorrupt", err)
			}
		})
	}
}
