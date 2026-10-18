package occupation_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/occupation"
)

// TestExtinct_Cases verifies the condition of the "role extinct" warning: no
// agent and either an earlier agent or a configured group.
//
// Requirement: FR-SPN-042
func TestExtinct_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		agents              int64
		everHad, configured bool
		want                bool
	}{
		{"had agents, none now", 0, true, false, true},
		{"configured group, none now", 0, false, true, true},
		{"never had and not configured (a closed visitor role)", 0, false, false, false},
		{"one agent left", 1, true, true, false},
		{"negative count is none", -1, true, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := occupation.Extinct(tt.agents, tt.everHad, tt.configured); got != tt.want {
				t.Errorf("Extinct(%d, %v, %v) = %v, want %v", tt.agents, tt.everHad, tt.configured, got, tt.want)
			}
		})
	}
}

// TestBelowFloor_Cases verifies the group-below-floor condition at its edges: a
// weight of 1 is never a group, and the count must be strictly below the floor.
//
// Requirement: FR-SPN-047
func TestBelowFloor_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                      string
		weight, agents, minAgents int64
		want                      bool
	}{
		{"group one below the floor", 10, 2, 3, true},
		{"group at the floor", 10, 3, 3, false},
		{"weight 1 is not a group", 1, 0, 3, false},
		{"weight 2 is a group", 2, 0, 3, true},
		{"floor 0 never fires", 10, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := occupation.BelowFloor(tt.weight, tt.agents, tt.minAgents); got != tt.want {
				t.Errorf("BelowFloor(%d, %d, %d) = %v, want %v", tt.weight, tt.agents, tt.minAgents, got, tt.want)
			}
		})
	}
}

// TestUnfunded_Cases verifies that the entry-unfunded condition needs entrants
// due and none funded.
//
// Requirement: FR-SPN-036
func TestUnfunded_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		due, funded int64
		want        bool
	}{
		{"due and none funded", 3, 0, true},
		{"due and one funded", 3, 1, false},
		{"none due", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := occupation.Unfunded(tt.due, tt.funded); got != tt.want {
				t.Errorf("Unfunded(%d, %d) = %v, want %v", tt.due, tt.funded, got, tt.want)
			}
		})
	}
}
