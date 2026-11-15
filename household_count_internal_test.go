package economysim

import (
	"slices"
	"testing"
)

// TestHouseholdCounts_EqualSnapshotRowsAndSkipPseudoAgents verifies that the one pass
// that counts the households of every region gives the Households of each snapshot
// row, with a removed agent (weight 0) and the estate's pseudo-agent left out.
//
// Arithmetic: millConfig(1, 4) has one group of 4 households in one region, so the
// count is 4; setting the weight of the last agent to 0 (a removal that has not left
// the table) leaves the weights of the others, 4 - 1 = 3.
//
// Requirement: FR-SET-009
func TestHouseholdCounts_EqualSnapshotRowsAndSkipPseudoAgents(t *testing.T) {
	t.Parallel()
	w := newMill(t, millConfig(1, 4))
	counts := make([]int64, len(w.cfg.Regions))
	w.householdCounts(counts)
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	if want := []int64{snap.Regions[0].Households}; !slices.Equal(counts, want) || counts[0] != 4 {
		t.Errorf("householdCounts() = %v, want %v (the snapshot row) and 4", counts, want)
	}
	last := &w.agents[len(w.agents)-1]
	last.weight = 0
	w.householdCounts(counts)
	if counts[0] != 3 {
		t.Errorf("householdCounts() with the last agent at weight 0 = %v, want [3]", counts)
	}
}
