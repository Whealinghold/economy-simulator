package ledger

import "testing"

// TestAge_SpoilsALotInATreasury verifies that a lot planted in a Treasury account
// spoils at the end of its shelf life to the Spoilage boundary of the account's
// region, like a lot of any holder (spec-ledger.md 5.3, spec-polity.md 5.3.1). No
// registry row moves goods into a treasury yet (the fiscal module's), so the lot is
// planted, as the ageing tests above do.
//
// Requirement: FR-POL-034
//
// Wear is not asserted: the Wear row runs agent to boundary, so a treasury cannot
// wear an item until a module widens that row (reported to the owner).
func TestAge_SpoilsALotInATreasury(t *testing.T) {
	t.Parallel()
	l, _ := iledger(t)
	tr, err := l.Open(KindTreasury, 0, 1)
	if err != nil {
		t.Fatalf("Open(KindTreasury) error = %v, want nil", err)
	}
	// Commodity 0 has a shelf life of 3: a lot born at tick -7 expired at tick -5.
	a := &l.accounts[tr]
	a.lots = []Lot{{Commodity: 0, Quantity: 4, Born: -7}}
	a.hint, a.hasHint = -5, true
	if err := l.Age(); err != nil {
		t.Fatalf("Age() error = %v, want nil", err)
	}
	if len(a.lots) != 0 {
		t.Errorf("the treasury holds %v after ageing, want the expired lot gone", a.lots)
	}
	b, ok := l.Boundary(Spoilage, 0)
	if !ok {
		t.Fatal("no Spoilage boundary for commodity 0")
	}
	if v, _ := l.Account(b); v.Balance != 4 {
		t.Errorf("Spoilage boundary balance = %d, want 4 (a sink adds the 4 units that spoiled in the treasury)", v.Balance)
	}
}
