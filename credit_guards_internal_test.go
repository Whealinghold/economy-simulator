package economysim

import (
	"errors"
	"strings"
	"testing"
)

// TestCredit_ViolatedInvariantsPanicWithTheirMessage verifies the three guards of
// the settlement and the waterfall that turn a broken invariant (a refused ledger
// operation, an account or an indexed claim that is gone) into a panic naming it,
// and that a nil error from the ledger passes without one.
//
// Guard: the module reads claims and accounts the indexes say are open. Without the
// panic a stale index would read a zero value and settle a claim that does not
// exist, moving no money and hiding the corruption until the audit.
func TestCredit_ViolatedInvariantsPanicWithTheirMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(w *World)
		want string // "" when no panic is expected
	}{
		{"a refused operation", func(*World) { mustClaim(errors.New("boom"), "the payment") }, "the ledger refused the payment of a checked claim: boom"},
		{"an operation that went through", func(*World) { mustClaim(nil, "the payment") }, ""},
		{"an account that is not there", func(w *World) { w.accountView(-1) }, "the account of a claim is unreadable"},
		{"an indexed claim that is not open", func(w *World) { w.indexed(1 << 40) }, "an indexed claim is not in the ledger's list"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := creditWorld(t, 0, 0)
			got := invariantMessage(func() { tt.run(w) })
			if tt.want == "" {
				if got != "" {
					t.Errorf("%s panicked with %q, want no panic", tt.name, got)
				}
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("%s panicked with %q, want a message containing %q", tt.name, got, tt.want)
			}
		})
	}
}

// TestCreditRoom_IsZeroWithoutABorrowBlock verifies that an agent whose role has no
// credit block, or a Lend block alone, has no exposure room (spec-credit.md 5.3.3),
// whatever its debt.
//
// Requirement: FR-CRD-015
// Guard: the room is the one place a role's Borrow block is dereferenced outside the
// round. A lender role asking for room must read 0, not fault on a nil block.
func TestCreditRoom_IsZeroWithoutABorrowBlock(t *testing.T) {
	t.Parallel()
	w := creditWorld(t, 0, 0)
	a := &w.agents[homeA]
	if got := w.creditRoom(a, 250); got != 0 {
		t.Errorf("creditRoom(role with no credit block, debt 250) = %d, want 0", got)
	}
	w.cfg.Roles[a.role].Credit = &CreditTerms{Lend: &LendTerms{}}
	if got := w.creditRoom(a, 250); got != 0 {
		t.Errorf("creditRoom(role with a Lend block only, debt 250) = %d, want 0", got)
	}
}

// TestCreditAgents_CountsTheRolesWithABlockAndSkipsTheRemoved verifies the counts of
// the construction summary: one lender per live agent of a role with a Lend block, one
// borrower per live agent of a role with a Borrow block, and an agent whose weight
// fell below 1 (removed) counts for neither.
//
// Requirement: FR-CRD-001
// Guard: the summary line is read by operators to see who can lend. A removed agent
// still counted would make the line disagree with the round that skips it.
func TestCreditAgents_CountsTheRolesWithABlockAndSkipsTheRemoved(t *testing.T) {
	t.Parallel()
	w := creditWorld(t, 0, 0)
	if l, b := w.creditAgents(); l != 0 || b != 0 {
		t.Fatalf("creditAgents() = %d, %d with no block, want 0, 0", l, b)
	}
	// Role 0 is the millers (agents 2 to 4 and 6, 7); role 1 the weavers (5 and 8).
	w.cfg.Roles[0].Credit = &CreditTerms{Lend: &LendTerms{}}
	w.cfg.Roles[1].Credit = &CreditTerms{Borrow: &BorrowTerms{}}
	var millers, weavers int64
	for i := w.firstHousehold; i < len(w.agents); i++ {
		switch w.agents[i].role {
		case 0:
			millers++
		case 1:
			weavers++
		}
	}
	if l, b := w.creditAgents(); l != millers || b != weavers {
		t.Errorf("creditAgents() = %d, %d, want %d lenders and %d borrowers", l, b, millers, weavers)
	}
	w.agents[homeA].weight = 0
	if l, _ := w.creditAgents(); l != millers-1 {
		t.Errorf("creditAgents() lenders = %d after one miller is removed, want %d", l, millers-1)
	}
}
