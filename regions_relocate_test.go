package economysim_test

import (
	"errors"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestRelocate_RegionBound verifies, through the public query and the test seam
// of the relocation, that a registered obligation refuses with a
// *RegionBoundError that names it and matches ErrRegionBound, that skipping it
// by name or by every instance key lifts the refusal, that skipping one of two
// instances does not, and that no other refusal is skipped.
//
// Requirement: FR-REG-047, FR-REG-060
func TestRelocate_RegionBound(t *testing.T) {
	t.Parallel()
	c := withRegions(referenceConfig(), "south")
	c.Groups = append(c.Groups, economysim.Group{Region: "south", Role: "cook", Households: 10})
	w, err := economysim.NewWorld(c, economysim.WithSeed(1))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	// The agent holds two instances of the obligation, keys 3 and 4.
	holder := economysim.AgentID(len(c.Regions)) // the first agent of the first region
	economysim.RegisterBoundForTest(w, "open_contract", func(a economysim.AgentID, skip []string) bool {
		return a == holder && (!economysim.SkipHasForTest(skip, "open_contract", 3) || !economysim.SkipHasForTest(skip, "open_contract", 4))
	})
	to := economysim.RegionID(1)
	tests := []struct {
		name string
		skip []string
		want bool
	}{
		{"no skip", nil, true},
		{"skip by name", []string{"open_contract"}, false},
		{"skip by both keys", []string{"open_contract:3", "open_contract:4"}, false},
		{"skip by one key", []string{"open_contract:3"}, true},
		{"skip of another obligation", []string{"on_journey"}, true},
	}
	for _, tt := range tests {
		reason, err := w.RelocationRefusal(holder, to, tt.skip...)
		var rb *economysim.RegionBoundError
		got := errors.Is(err, economysim.ErrRegionBound) && errors.As(err, &rb) && rb.Obligation == "open_contract" && reason == string(economysim.ReasonRegionBound)
		if got != tt.want || (!tt.want && (reason != "" || err != nil)) {
			t.Errorf("%s: RelocationRefusal(%d, %d, %v) = %q, %v, want a region_bound refusal naming open_contract: %v", tt.name, holder, to, tt.skip, reason, err, tt.want)
		}
	}
	if err := economysim.RelocateForTest(w, holder, to, economysim.CauseConsumer); !errors.Is(err, economysim.ErrRegionBound) {
		t.Errorf("RelocateForTest = %v, want ErrRegionBound: the guarded path takes no skip", err)
	}
}
