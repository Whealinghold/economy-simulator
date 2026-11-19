package ledger_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestRegistry_InitialToTreasury verifies that the reason initial moves money from
// the boundary to a Treasury account and that nothing else is widened: goods still
// refuse a treasury under initial, no other built-in reason or declared row has one
// at either end (spoilage, which runs from any holder, and trade, which the fiscal
// module's standing orders use, apart), and a trade between an agent and a treasury
// is allowed in both directions for money and goods. A treasury never goes below zero.
//
// Requirement: FR-LED-065, FR-POL-017, FR-POL-018, FR-FIS-052
func TestRegistry_InitialToTreasury(t *testing.T) {
	t.Parallel()
	f := newFix(t, testMaxMoney)
	tr, err := f.l.Open(ledger.KindTreasury, 0, 1)
	if err != nil {
		t.Fatalf("Open(KindTreasury) error = %v, want nil", err)
	}
	if err := f.l.Transfer(f.mint, tr, 40, ledger.Initial); err != nil {
		t.Fatalf("Transfer(boundary, treasury, 40, initial) error = %v, want nil", err)
	}
	if v, _ := f.l.Account(tr); v.Balance != 40 || v.Kind != ledger.KindTreasury || v.Region != 1 {
		t.Errorf("Account(treasury) = %+v, want balance 40, kind Treasury, region 1", v)
	}
	if err := f.l.Transfer(tr, f.a0, 41, ledger.Trade); !errors.Is(err, ledger.ErrInsufficientFunds) {
		t.Errorf("Transfer(treasury, agent, 41, trade) error = %v, want ErrInsufficient (the treasury holds 40)", err)
	}
	if err := f.l.Transfer(tr, f.a0, 10, ledger.Trade); err != nil {
		t.Errorf("Transfer(treasury, agent, 10, trade) error = %v, want nil", err)
	}
	if err := f.l.Transfer(f.a0, tr, 1, ledger.Trade); err != nil {
		t.Errorf("Transfer(agent, treasury, 1, trade) error = %v, want nil", err)
	}
	if err := f.l.Move(f.goodsBoundary, tr, 0, 1, ledger.Initial); !errors.Is(err, ledger.ErrReasonMismatch) {
		t.Errorf("Move(boundary, treasury, goods, initial) error = %v, want ErrReasonMismatch", err)
	}
	onlyInitialMoneyNamesATreasury(t)
}

// onlyInitialMoneyNamesATreasury checks every row of the registry for a Treasury at
// either end.
func onlyInitialMoneyNamesATreasury(t *testing.T) {
	t.Helper()
	for r := ledger.Reason(0); r < ledger.NumReasons; r++ {
		for _, goods := range []bool{false, true} {
			if r == ledger.Initial && !goods || r == ledger.Spoilage || r == ledger.Trade {
				continue // spoilage runs from any holder, a treasury included (FR-POL-034)
			}
			for k := ledger.KindAgent; k <= ledger.KindBoundary; k++ {
				if ledger.Allowed(r, goods, ledger.KindTreasury, k) || ledger.Allowed(r, goods, k, ledger.KindTreasury) {
					t.Errorf("Allowed(reason %d, goods %v) names a treasury with kind %d, want it refused: only initial money, spoilage and trade have a treasury row", r, goods, k)
				}
			}
		}
	}
}
