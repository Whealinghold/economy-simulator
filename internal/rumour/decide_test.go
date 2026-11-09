package rumour_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/rumour"
)

// TestDecide_FirstCaseWins verifies the four cases of spec-information.md 5.3.1
// and their order: no path before a lag above the horizon, the horizon before
// the young run, and the boundaries (a lag equal to the horizon or to the tick
// is offered).
//
// Requirement: FR-INF-010, FR-INF-011, FR-INF-012, FR-INF-013
func TestDecide_FirstCaseWins(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		hasPath      bool
		lag          int64
		maxAge, asOf int64
		want         rumour.Outcome
	}{
		{"no walk is no path", false, 3, 60, 100, rumour.NoPath},
		{"no path wins over a lag above the horizon", false, 99, 60, 100, rumour.NoPath},
		{"no path wins over a young run", false, 99, 60, 0, rumour.NoPath},
		{"lag above the horizon is too old", true, 61, 60, 100, rumour.TooOld},
		{"too old wins over not yet", true, 61, 60, 5, rumour.TooOld},
		{"lag equal to the horizon is offered", true, 60, 60, 100, rumour.Offered},
		{"lag above the tick is not yet", true, 3, 60, 2, rumour.NotYet},
		{"tick 0 is not yet for the least lag", true, 2, 60, 0, rumour.NotYet},
		{"lag equal to the tick is offered", true, 3, 60, 3, rumour.Offered},
		{"the migration pass reads as of the next tick", true, 3, 60, 3, rumour.Offered},
		{"ordinary candidate", true, 4, 60, 100, rumour.Offered},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := rumour.Decide(tt.hasPath, tt.lag, tt.maxAge, tt.asOf); got != tt.want {
				t.Errorf("Decide(%v, %d, %d, %d) = %d, want %d", tt.hasPath, tt.lag, tt.maxAge, tt.asOf, got, tt.want)
			}
		})
	}
	if rumour.Outcome(0) != rumour.NoPath {
		t.Errorf("the zero Outcome = %d, want NoPath, the outcome that offers nothing", rumour.Outcome(0))
	}
}
