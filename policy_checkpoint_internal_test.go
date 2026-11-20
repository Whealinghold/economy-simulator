package economysim

import (
	"bytes"
	"errors"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// The tests of the checkpoint of the policy module (spec-policy.md 13; plan 3-plc Phase 7):
// the extension entry of a Stateful consumer policy, tag 32 and its decoder.

// countingPolicy is a Stateful consumer policy that counts its calls and sets the poll to a
// value that follows the count.
type countingPolicy struct{ n int64 }

func (p *countingPolicy) Decide(PolicyView, *rand.Rand) Settings {
	p.n++
	return Settings{Levers: []LeverSetting{{Lever: 1, BP: 10_000 + 100*p.n}}}
}

func (p *countingPolicy) SaveState() (uint32, []byte, error) { return 1, []byte{byte(p.n)}, nil }

func (p *countingPolicy) LoadState(_ uint32, b []byte) error {
	p.n = int64(b[0])
	return nil
}

// TestCheckpoint_PolicyStatefulKind3 verifies that a Stateful consumer policy is saved as an
// extension of kind 3 under its name, after the entries of the lower kinds, and that a
// restored world carries on as the original does.
//
// Requirement: FR-PLC-023
func TestCheckpoint_PolicyStatefulKind3(t *testing.T) {
	t.Parallel()
	cfg := policyMillConfig(crownActor("council", 1))
	w := newMill(t, cfg, WithSeed(3), WithPolicy("council", &countingPolicy{}), WithPolicy("plain", &scriptedPolicy{}))
	entries := w.statefulEntries()
	if last := entries[len(entries)-1]; last.kind != extPolicy || last.name != "council" || len(entries) != 2 {
		t.Fatalf("statefulEntries = %+v, want the directed strategy and the one Stateful policy of kind 3", entries)
	}
	r, fresh := restoreCounting(t, w, cfg)
	if fresh.n != 3 {
		t.Errorf("restored policy count = %d, want 3", fresh.n)
	}
	for range 2 {
		if err := errors.Join(w.Step(), r.Step()); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	if a, b := errVal(w.Slots()), errVal(r.Slots()); !slices.Equal(a.FlowScales, b.FlowScales) {
		t.Errorf("flow scales after the restore = %+v, want %+v", b.FlowScales, a.FlowScales)
	}
}

// restoreCounting steps w three ticks, checkpoints it and restores the checkpoint with a new
// countingPolicy, which it returns with the restored world. It also checks that a restore
// that registers the policy as something that is not Stateful is refused.
func restoreCounting(t *testing.T, w *World, cfg Config) (*World, *countingPolicy) {
	t.Helper()
	for range 3 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	fresh := &countingPolicy{}
	r, err := RestoreWorld(cfg, ck, WithPolicy("council", fresh), WithPolicy("plain", &scriptedPolicy{}))
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	if _, err := RestoreWorld(cfg, ck, WithPolicy("council", &scriptedPolicy{})); !errors.Is(err, ErrCheckpointExtension) {
		t.Errorf("RestoreWorld() with the policy not Stateful and one missing: error = %v, want ErrCheckpointExtension", err)
	}
	return r, fresh
}

// TestExtensions_BuiltinsKeepNone verifies that a world with built-in policies and no
// registered policy adds no extension entry: the entries are the ones of a world with the
// module off.
//
// Requirement: FR-PLC-023
func TestExtensions_BuiltinsKeepNone(t *testing.T) {
	t.Parallel()
	on := newMill(t, policyMillConfig(crownActor("hold", 1), shireActor("directed", 1)), WithSeed(3))
	off := policyMillConfig()
	off.Modules.Policy = nil
	if a, b := len(on.statefulEntries()), len(newMill(t, off, WithSeed(3)).statefulEntries()); a != b {
		t.Errorf("extension entries = %d with built-in policies, %d with the module off, want equal", a, b)
	}
}

// policyBody builds a body of tag 32 for the tests of the decoder.
func policyBody(fill func(cw *codec.Writer)) []byte {
	cw := codec.NewWriter(nil)
	fill(cw)
	return cw.Out()
}

// policyRow writes one row of tag 32: the actor, the count and the settings.
func policyRow(cw *codec.Writer, actor uint64, settings ...LeverSetting) {
	cw.U(actor)
	cw.Count(len(settings))
	for _, s := range settings {
		cw.U(uint64(s.Lever))
		cw.U(uint64(s.BP))
		cw.B(s.Closed)
	}
}

// loadPolicyBody loads a body into a fresh world of directedMill and returns the error of the
// section, with the world.
func loadPolicyBody(t testing.TB, body []byte) (*World, error) {
	t.Helper()
	w, _ := directedMill(t)
	r := codec.NewReader(body)
	err := w.loadPolicy(&restoreCtx{tick: 10}, r)
	if err == nil {
		err = r.Err()
	}
	return w, err
}

// TestCheckpoint_PolicyMailboxRoundTrip verifies the worked example of spec-policy.md 14.4: a
// directive left at tick 95 for an actor due at ticks 0 mod 10, a checkpoint at 97 and a
// restore that applies it at 100 as the original does. The restored world holds the same
// mailbox, writes the same bytes when saved again and ends with the same slots.
//
// Requirement: FR-PLC-073, FR-PLC-090
func TestCheckpoint_PolicyMailboxRoundTrip(t *testing.T) {
	t.Parallel()
	w, rec := directedMill(t)
	stepWorlds(t, 95, w)
	if err := w.DirectPolicy([]PolicyDirective{{Actor: 0, Levers: []LeverSetting{{Lever: 0, BP: 7_000, Closed: true}, {Lever: 1, BP: 12_000}}}}); err != nil {
		t.Fatalf("DirectPolicy() error = %v, want nil", err)
	}
	stepWorlds(t, 2, w)
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	cfg := policyMillConfig(crownActor("directed", 10), shireActor("hold", 1))
	r, err := RestoreWorld(cfg, ck, WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	if !slices.Equal(r.policy.mailbox[0], w.policy.mailbox[0]) || len(r.policy.mailbox[0]) != 2 {
		t.Fatalf("restored mailbox = %+v, want %+v", r.policy.mailbox[0], w.policy.mailbox[0])
	}
	stepWorlds(t, 4, w, r) // ticks 97, 98, 99 and 100, the decision
	if a, b := errVal(w.Slots()), errVal(r.Slots()); !reflect.DeepEqual(a, b) || len(of[PolicyDecided](rec)) != 1 || len(w.policy.mailbox[0]) != 0 {
		t.Errorf("after tick 100 the slots are %+v restored and %+v original with %d decisions, want equal and one", b, a, len(of[PolicyDecided](rec)))
	}
	if a, b := errVal(w.Checkpoint()), errVal(r.Checkpoint()); !bytes.Equal(errVal(a.MarshalBinary()), errVal(b.MarshalBinary())) {
		t.Error("the restored world writes other bytes after the decision")
	}
}

// stepWorlds runs n steps of each world in turn.
func stepWorlds(t *testing.T, n int, ws ...*World) {
	t.Helper()
	for range n {
		for _, w := range ws {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() error = %v, want nil", err)
			}
		}
	}
}

// TestCheckpoint_PolicySavesNoLeverValue verifies that the section is empty for a world with
// no directive pending, even after an actor wrote a lever: a value is the owner's state.
//
// Requirement: FR-PLC-092
func TestCheckpoint_PolicySavesNoLeverValue(t *testing.T) {
	t.Parallel()
	w, _ := directedMill(t)
	if err := w.DirectPolicy([]PolicyDirective{{Actor: 0, Levers: []LeverSetting{{Lever: 1, BP: 11_000}}}}); err != nil {
		t.Fatalf("DirectPolicy() error = %v, want nil", err)
	}
	if err := w.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	cw := codec.NewWriter(nil)
	if err := w.savePolicy(cw); err != nil || cw.Len() != 0 {
		t.Errorf("savePolicy() = %d bytes (%v) after the directive was applied, want 0", cw.Len(), err)
	}
	if s := errVal(w.Slots()); flowScaleIn(s, "poll") != 11_000 {
		t.Errorf("poll = %d, want the 11000 the directive set, which the fiscal section holds", flowScaleIn(s, "poll"))
	}
}

// flowScaleIn is the scale of the flow with the ID in s.
func flowScaleIn(s SlotState, id string) int64 {
	for _, f := range s.FlowScales {
		if f.Flow == id {
			return f.BP
		}
	}
	return -1
}

// TestCheckpoint_PolicyRefusesCorrupt verifies one refusal for each check of spec-policy.md
// 13.2 and for a body that ends inside a row, and that a good body is read into the mailbox.
//
// Requirement: FR-PLC-090
func TestCheckpoint_PolicyRefusesCorrupt(t *testing.T) {
	t.Parallel()
	one := LeverSetting{Lever: 1, BP: 11_000}
	good := policyBody(func(cw *codec.Writer) { policyRow(cw, 0, LeverSetting{Lever: 0, BP: 5_000, Closed: true}, one) })
	w, err := loadPolicyBody(t, good)
	if err != nil || len(w.policy.mailbox[0]) != 2 {
		t.Fatalf("a good body: error = %v with mailbox %+v, want it read", err, w.policy.mailbox[0])
	}
	for _, tc := range []struct {
		name string
		body []byte
	}{
		{"actor out of range", policyBody(func(cw *codec.Writer) { policyRow(cw, 2, one) })},
		{"actor not directed", policyBody(func(cw *codec.Writer) { policyRow(cw, 1, LeverSetting{Lever: 0, BP: 6_000}) })},
		{"actor repeated", policyBody(func(cw *codec.Writer) { policyRow(cw, 0, one); policyRow(cw, 0, one) })},
		{"empty row", policyBody(func(cw *codec.Writer) { policyRow(cw, 0) })},
		{"lever out of range", policyBody(func(cw *codec.Writer) { policyRow(cw, 0, LeverSetting{Lever: 3, BP: 1}) })},
		{"lever repeated", policyBody(func(cw *codec.Writer) { policyRow(cw, 0, one, one) })},
		{"lever descending", policyBody(func(cw *codec.Writer) { policyRow(cw, 0, LeverSetting{Lever: 2, BP: 1}, one) })},
		{"value above max", policyBody(func(cw *codec.Writer) { policyRow(cw, 0, LeverSetting{Lever: 1, BP: 20_001}) })},
		{"value below min", policyBody(func(cw *codec.Writer) { policyRow(cw, 0, LeverSetting{Lever: 0, BP: 4_999}) })},
		{"closed without may close", policyBody(func(cw *codec.Writer) { policyRow(cw, 0, LeverSetting{Lever: 1, BP: 1, Closed: true}) })},
		{"count beyond the bytes", policyBody(func(cw *codec.Writer) { cw.U(0); cw.U(100) })},
		{"body ends inside a row", good[:len(good)-1]},
		{"trailing byte", append(slices.Clone(good), 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := loadPolicyBody(t, tc.body); !errors.Is(err, codec.ErrMalformed) {
				t.Errorf("loadPolicy() error = %v, want a malformed body", err)
			}
		})
	}
}

// FuzzDecodePolicySection verifies that the decoder never panics and that a body it accepts is
// the canonical one: saved again it gives the same bytes.
//
// Requirement: NFR-PLC-003, FR-PLC-090
func FuzzDecodePolicySection(f *testing.F) {
	f.Add([]byte{})
	f.Add(policyBody(func(cw *codec.Writer) { policyRow(cw, 0, LeverSetting{Lever: 1, BP: 11_000}) }))
	f.Add(policyBody(func(cw *codec.Writer) {
		policyRow(cw, 0, LeverSetting{Lever: 0, BP: 5_000, Closed: true}, LeverSetting{Lever: 2, BP: 1})
	}))
	f.Fuzz(func(t *testing.T, body []byte) {
		w, err := loadPolicyBody(t, body)
		if err != nil {
			return
		}
		cw := codec.NewWriter(nil)
		if err := w.savePolicy(cw); err != nil || !bytes.Equal(cw.Out(), body) {
			t.Errorf("an accepted body %x saves as %x (%v), want the same bytes", body, cw.Out(), err)
		}
	})
}
