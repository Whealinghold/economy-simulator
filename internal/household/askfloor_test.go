package household_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/household"
)

// TestAskFloor verifies the floor of World.AskFloorBP: trunc(ref * bp / 10 000)
// is the least the limit can be, bp 0 is off, and a floor that does not fit
// leaves the limit.
//
// Arithmetic: ref 96, bp 5000: floor 48. 33 * 5000 / 10 000 truncates 16.5 to 16.
//
// Requirement: FR-AGT-087
func TestAskFloor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		limit, ref, bp int64
		want           int64
	}{
		{"off", 10, 96, 0, 10},
		{"limit below the floor is raised", 10, 96, 5000, 48},
		{"limit above the floor stays", 60, 96, 5000, 60},
		{"floor truncates", 1, 33, 5000, 16},
		{"whole reference", 10, 96, 10_000, 96},
		{"floor that overflows leaves the limit", 10, math.MaxInt64, 10_000, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := household.AskFloor(tt.limit, tt.ref, tt.bp); got != tt.want {
				t.Errorf("AskFloor(%d, %d, %d) = %d, want %d", tt.limit, tt.ref, tt.bp, got, tt.want)
			}
		})
	}
}
