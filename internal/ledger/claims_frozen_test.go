package ledger_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestClaim_FormationFieldsAreFrozen verifies that the Principal, Interest,
// RateBP, Formed and Due of a claim stay as they were formed through every
// operation that touches the claim (a payment, a reassignment, an overdue
// mark, a refused payment), and that PrincipalOwed and InterestOwed only
// fall.
//
// Requirement: FR-CRD-009
func TestClaim_FormationFieldsAreFrozen(t *testing.T) {
	t.Parallel()
	c := newCfix(t)
	id := c.lend(t)
	formed, _ := c.l.Claim(id)
	steps := []struct {
		name    string
		do      func() error
		wantErr error
	}{
		{"interest part paid", func() error { return c.l.Settle(id, 6, 0) }, nil},
		{"overdue marked", func() error { return c.l.MarkOverdue(id, 11) }, nil},
		{"reassigned to an estate", func() error { return c.l.Reassign(id, c.e0) }, nil},
		{"payment above what is owed refused", func() error { return c.l.Settle(id, 0, 61) }, ledger.ErrInvalidAmount},
		{"principal part paid", func() error { return c.l.Settle(id, 0, 20) }, nil},
	}
	prev := formed
	for _, st := range steps {
		if err := st.do(); !errors.Is(err, st.wantErr) && (st.wantErr != nil || err != nil) {
			t.Fatalf("%s: err = %v, want %v", st.name, err, st.wantErr)
		}
		got, ok := c.l.Claim(id)
		if !ok {
			t.Fatalf("%s: the claim ended before the last step, want it open", st.name)
		}
		if got.Principal != formed.Principal || got.Interest != formed.Interest || got.RateBP != formed.RateBP ||
			got.Formed != formed.Formed || got.Due != formed.Due {
			t.Errorf("%s: formation fields = %+v, want those of %+v", st.name, got, formed)
		}
		if got.PrincipalOwed > prev.PrincipalOwed || got.InterestOwed > prev.InterestOwed {
			t.Errorf("%s: owed rose from %d+%d to %d+%d", st.name,
				prev.PrincipalOwed, prev.InterestOwed, got.PrincipalOwed, got.InterestOwed)
		}
		prev = got
	}
	if prev.PrincipalOwed != 40 || prev.InterestOwed != 0 {
		t.Errorf("owed after the steps = %d principal, %d interest, want 40, 0", prev.PrincipalOwed, prev.InterestOwed)
	}
}
