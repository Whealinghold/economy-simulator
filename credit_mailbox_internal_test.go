package economysim

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// TestCheckpoint_CreditDirective verifies that the Borrow and the Lend of a sticky
// directive are saved with the agent's record and restored: the mailbox of the
// restored world equals the original's, the checkpoint bytes are the same, and the
// two worlds form the same loans in the next step, so a request left in a mailbox
// is not lost at a fork.
//
// Requirement: FR-CRD-055, FR-CRD-087
func TestCheckpoint_CreditDirective(t *testing.T) {
	t.Parallel()
	w, rec, miller, weaver := directWorldCredit(t)
	req, off := CreditRequest{Principal: 500, Term: 30, MaxRateBP: 1_500}, CreditOffer{Principal: 2_000, MaxTerm: 90, RateBP: 600}
	if err := errors.Join(w.Direct(miller, Directive{Borrow: &req, Sticky: true}), w.Direct(weaver, Directive{Lend: &off, Sticky: true})); err != nil {
		t.Fatalf("Direct error = %v, want nil", err)
	}
	cp, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint error = %v, want nil", err)
	}
	other := &evLog{}
	restored, err := RestoreWorld(w.Config(), cp, WithStrategy("script", newLoanScript()), WithObserver(other), WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	for _, id := range []AgentID{miller, weaver} {
		a, _ := w.Agent(id)
		b, _ := restored.Agent(id)
		if !a.HasDirective || !reflect.DeepEqual(a.Directive, b.Directive) {
			t.Errorf("mailbox of agent %d after the restore = %+v, want %+v", id, b.Directive, a.Directive)
		}
	}
	if again, _ := restored.Checkpoint(); !bytes.Equal(again.b, cp.b) {
		t.Error("checkpoint of the restored world differs from the one it was restored from, want the same bytes")
	}
	step(t, w, 1)
	step(t, restored, 1)
	if a, b := formedOf(rec), formedOf(other); len(a) == 0 || !reflect.DeepEqual(a, b) {
		t.Errorf("loans formed by the original %+v and by the restored world %+v, want the same, at least one", a, b)
	}
}

// TestLoadDirectiveCredit_RefusesABadRecord verifies the reading of the credit fields
// of a mailbox: a request or an offer above its upper bound, or a negative amount
// written as the unsigned number it converts to, is malformed.
//
// Requirement: FR-CRD-087
func TestLoadDirectiveCredit_RefusesABadRecord(t *testing.T) {
	t.Parallel()
	body := func(b *[3]uint64, o *[3]uint64) []byte {
		w := codec.NewWriter(nil)
		for _, x := range []*[3]uint64{b, o} {
			w.B(x != nil)
			if x != nil {
				w.U(x[0])
				w.U(x[1])
				w.U(x[2])
			}
		}
		return w.Out()
	}
	good := &[3]uint64{500, 30, 1_500}
	tests := []struct {
		name string
		body []byte
		ok   bool
	}{
		{"both", body(good, good), true},
		{"neither", body(nil, nil), true},
		{"a principal above MaxMoney", body(&[3]uint64{MaxMoney + 1, 30, 1}, nil), false},
		{"a term above the bound", body(nil, &[3]uint64{5, MaxCreditTerm + 1, 1}), false},
		{"a rate above the bound", body(good, &[3]uint64{5, 5, MaxCreditRateBP + 1}), false},
		{"a negative principal written as unsigned", body(&[3]uint64{^uint64(0), 30, 1}, nil), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := codec.NewReader(tt.body)
			d := &Directive{}
			loadDirectiveCredit(r, d)
			err := r.Done()
			if ok := err == nil && r.Err() == nil; ok != tt.ok {
				t.Errorf("loadDirectiveCredit(%s) = %+v, error %v, want ok %v", tt.name, d, err, tt.ok)
			}
		})
	}
}

// TestCheckLoadedDirective_HoldsTheCreditFieldsToDirect verifies that a restored
// mailbox is held to the rules Direct holds a set one to: a principal of 0 is out of
// bounds, and a Borrow for a role with a Lend block, or with none, is not on the role.
//
// Requirement: FR-CRD-087, FR-CRD-055
func TestCheckLoadedDirective_HoldsTheCreditFieldsToDirect(t *testing.T) {
	t.Parallel()
	w, _, _ := loanWorld(t, newLoanScript())
	tests := []struct {
		name string
		role uint64
		d    Directive
		want error
	}{
		{"a principal of 0", 0, Directive{Borrow: &CreditRequest{Term: 5}}, ErrOutOfBounds},
		{"a Borrow for a lender", 1, Directive{Borrow: &CreditRequest{Principal: 5, Term: 5}}, ErrNotOnRole},
		{"a good Borrow", 0, Directive{Borrow: &CreditRequest{Principal: 5, Term: 5}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := w.checkLoadedDirective(&agentRec{role: tt.role, weight: 1, directive: &tt.d})
			if !errors.Is(err, tt.want) || (tt.want == nil && err != nil) {
				t.Errorf("checkLoadedDirective(%+v) = %v, want %v", tt.d, err, tt.want)
			}
		})
	}
}
