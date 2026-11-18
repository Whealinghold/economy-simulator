package economysim

import (
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestAudit_NamesEachFaultOfTheHireList verifies that the part of the hires of the audit
// fails with ErrAuditFailed for each fault of spec-employment.md 7.4.2 that the
// other audit tests do not reach: hire IDs out of order, an ID at or above the next,
// two hires on one account, an account of another region or of another kind, a
// worker whose job names another principal, and money left in a free Escrow account.
// Each case corrupts one thing in a sound world of two hires (two farms and two
// hands, a recipe of 3 ticks, so both hires are open after one step).
//
// Requirement: FR-EMP-041
//
// Guard: a branch of the audit that is never exercised is a check that a refactor
// can drop with no test noticing.
func TestAudit_NamesEachFaultOfTheHireList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		corrupt func(t *testing.T, w *World)
	}{
		{"ids out of order", func(_ *testing.T, w *World) {
			h := w.employment.hires
			h[0], h[1] = h[1], h[0]
		}},
		{"id not below the next", func(_ *testing.T, w *World) {
			w.employment.next = w.employment.hires[1].id
		}},
		{"two hires on one account", func(_ *testing.T, w *World) {
			w.employment.hires[1].escrow = w.employment.hires[0].escrow
		}},
		{"account of another region", func(_ *testing.T, w *World) {
			w.employment.hires[0].region++
		}},
		{"account that is not an Escrow account", func(_ *testing.T, w *World) {
			w.employment.hires[0].escrow = w.agents[w.employment.hires[0].worker].account
		}},
		{"job of another principal", func(_ *testing.T, w *World) {
			h := &w.employment.hires[0]
			w.agents[h.worker].job.principal = h.worker
		}},
		{"money in a free Escrow account", func(t *testing.T, w *World) {
			id, err := w.ledger.Open(ledger.KindEscrow, 0, 0)
			if err != nil {
				t.Fatalf("Open(Escrow): %v", err)
			}
			from := w.agents[slotsOf(w, "farm")[0]].account
			if err := w.ledger.Transfer(from, id, 1, ledger.WageEscrow); err != nil {
				t.Fatalf("Transfer(): %v", err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := hireWorld(t, defaultHire(hireOpts{farms: 2, hands: 2}), farmWants(1, 100, 40))
			stepN(t, w, 1)
			if got := len(w.employment.hires); got != 2 {
				t.Fatalf("open hires after one step = %d, want 2", got)
			}
			if err := w.auditHires(); err != nil {
				t.Fatalf("auditHires() on a sound world = %v, want nil", err)
			}
			tt.corrupt(t, w)
			if err := w.auditHires(); !errors.Is(err, ErrAuditFailed) {
				t.Errorf("auditHires() after %q = %v, want ErrAuditFailed", tt.name, err)
			}
		})
	}
}
