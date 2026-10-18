package economysim

import "testing"

// TestEntryTally_RegionAtLength verifies a region equal to the number of regions
// is outside the tally: adding to it neither panics nor changes a row.
//
// Requirement: FR-SPN-035
func TestEntryTally_RegionAtLength(t *testing.T) {
	t.Parallel()
	tl := newEntryTally(2)
	if got := tl.row(2); got != nil {
		t.Fatalf("row(2) of a 2-region tally = %+v, want nil", got)
	}
	tl.addPlan(2, &Decisions{regionCap: 1})
	tl.addExecuted(2, 1, 1, 1, 1, 1)
	for r := RegionID(0); r < 2; r++ {
		if _, ok := tl.limited(0, r); ok {
			t.Errorf("limited(region %d) = true, want false", r)
		}
	}
}
