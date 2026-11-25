package economysim

import "testing"

// TestCoverItemOf_HostileLostEntries verifies the defensive reading of the units lost (plan
// 3-ins Phase 6): a commodity outside the configuration, a negative quantity and a
// quantity of MaxInt64 are no loss or a saturated one, and never an index panic or a wrap.
//
// Worked: the entries (commodity 99, 5), (0, -3) are skipped, so the loss value is 0 and
// the due is 0; (0, MaxInt64) saturates the loss value and the due is the maximum payout.
//
// Requirement: FR-INS-041, FR-INS-042
func TestCoverItemOf_HostileLostEntries(t *testing.T) {
	t.Parallel()
	w, _ := settleWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), 8)
	p := &coverPolicy{id: 1, maxPayout: 1_000, coverBP: 10_000}
	it := w.coverItemOf(p, 0, []Amount{{Commodity: 99, Quantity: 5}, {Commodity: 0, Quantity: -3}})
	if it.lossValue != 0 || it.due != 0 {
		t.Errorf("hostile entries gave loss %d due %d, want 0 and 0", it.lossValue, it.due)
	}
	it = w.coverItemOf(p, 0, []Amount{{Commodity: 0, Quantity: Quantity(1<<63 - 1)}, {Commodity: 0, Quantity: Quantity(1<<63 - 1)}})
	if it.due < 0 || it.due > p.maxPayout {
		t.Errorf("saturated loss gave due %d, want within [0, %d]", it.due, p.maxPayout)
	}
}
