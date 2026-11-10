package ledger

import "testing"

// TestLedger_OutsideReasonsLeaveEarlierLayoutUnchanged verifies that declaring
// the outside reasons appends their flow slots after every earlier row: the
// offsets of the earlier declared reasons do not move, and a ledger that does
// not declare them has the stride and the offsets it had before the six rows
// existed (the fields the per-position state hash walks).
//
// Guard: N-217. The off golden files and the pinned hash must not move when a
// module is added; a row inserted ahead of an earlier one would shift the slots
// of a ledger whose module is off.
func TestLedger_OutsideReasonsLeaveEarlierLayoutUnchanged(t *testing.T) {
	t.Parallel()
	cfg := func(declared ...Reason) Config {
		return Config{Commodities: []Commodity{{}, {}}, Regions: 1, MaxMoney: 100, MaxQuantity: 100, Declared: declared}
	}
	base, err := New(cfg(Relief, EscrowIn))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	with, err := New(cfg(Relief, EscrowIn, OutsideBudget, OutsideOffer))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// outside_budget is one money slot and outside_offer one slot per commodity.
	if want := base.stride + 1 + 2; with.stride != want {
		t.Errorf("stride with the outside reasons = %d, want %d (the earlier stride %d plus 1 money and 2 goods slots)", with.stride, want, base.stride)
	}
	for _, r := range []Reason{Trade, Relief, EscrowIn} {
		for th := range 2 {
			if a, b := base.offOf(r, th), with.offOf(r, th); a != b {
				t.Errorf("offset of reason %d thing %d = %d with the outside reasons, %d without: an earlier row moved", r, th, b, a)
			}
		}
	}
	if base.offOf(OutsideBudget, thingMoney) != -1 || base.offOf(OutsideOffer, thingGoods) != -1 {
		t.Errorf("a ledger that does not declare the outside reasons has a slot for them")
	}
	none, err := New(cfg())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if none.later != nil {
		t.Errorf("later offsets = %v with no declared reason, want nil", none.later)
	}
}
