package facility_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/facility"
)

// TestRoundRobin_Split verifies the round robin of an initial row: 3 over 5
// gives the first three agents one each, 8 over 5 gives the first three two each.
//
// Requirement: FR-CAP-009
func TestRoundRobin_Split(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		n      int64
		agents int
		want   []int64
	}{
		{"three over five", 3, 5, []int64{1, 1, 1, 0, 0}},
		{"eight over five", 8, 5, []int64{2, 2, 2, 1, 1}},
		{"even", 10, 5, []int64{2, 2, 2, 2, 2}},
		{"none", 0, 3, []int64{0, 0, 0}},
		{"one agent", 7, 1, []int64{7}},
		{"no agents", 3, 0, nil},
		{"negative count", -1, 3, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := facility.RoundRobin(tt.n, tt.agents); !slices.Equal(got, tt.want) {
				t.Errorf("RoundRobin(%d, %d) = %v, want %v", tt.n, tt.agents, got, tt.want)
			}
		})
	}
}

// TestRoundRobin_Sums verifies that the parts always add up to n and differ by
// at most one, for a range of sizes.
//
// Requirement: FR-CAP-009
func TestRoundRobin_Sums(t *testing.T) {
	t.Parallel()
	for agents := 1; agents <= 12; agents++ {
		for n := int64(0); n <= 40; n++ {
			got := facility.RoundRobin(n, agents)
			var sum int64
			for _, g := range got {
				sum += g
			}
			if sum != n || got[0]-got[len(got)-1] > 1 {
				t.Fatalf("RoundRobin(%d, %d) = %v: sum %d, spread %d", n, agents, got, sum, got[0]-got[len(got)-1])
			}
		}
	}
}
