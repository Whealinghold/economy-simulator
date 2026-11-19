package saving

import (
	"math"
	"testing"
)

// TestSatAdd_SaturatesBySign checks that a sum that leaves int64 becomes the
// extreme of the sign of the operands and any other sum is exact.
//
// The exported rules add non-negative terms only, so the downward branch is
// unreachable through the package's API; this test pins it because a wrap
// there would turn a very negative balance into a huge positive one.
//
// Requirement: FR-SAV-043
func TestSatAdd_SaturatesBySign(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b int64
		want int64
	}{
		{"in range", 2, 3, 5},
		{"max plus one saturates up", math.MaxInt64, 1, math.MaxInt64},
		{"min plus minus one saturates down", math.MinInt64, -1, math.MinInt64},
		{"both min saturates down", math.MinInt64, math.MinInt64, math.MinInt64},
		{"max plus min is minus one", math.MaxInt64, math.MinInt64, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := satAdd(tt.a, tt.b); got != tt.want {
				t.Errorf("satAdd(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
