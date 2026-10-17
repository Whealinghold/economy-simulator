package welfare_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// TestNextRun verifies the run counter advances by one and saturates at
// MaxInt64 without wrapping.
//
// Requirement: FR-WEL-033
func TestNextRun(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want int64 }{
		{0, 1}, {9, 10}, {math.MaxInt64 - 1, math.MaxInt64}, {math.MaxInt64, math.MaxInt64},
	}
	for _, tt := range tests {
		if got := welfare.NextRun(tt.in); got != tt.want {
			t.Errorf("NextRun(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
