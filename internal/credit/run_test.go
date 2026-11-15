package credit_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/credit"
)

// TestNextRun_WarnsAtTheTenthAndEveryTenth verifies that a run of ticks in which
// the condition held warns at the tenth and at every tenth after it, that an
// inactive tick resets the run, and that a saturated run stays put and no longer
// warns.
//
// Requirement: FR-CRD-076
func TestNextRun_WarnsAtTheTenthAndEveryTenth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		run      int64
		active   bool
		wantRun  int64
		wantWarn bool
	}{
		{"first tick", 0, true, 1, false},
		{"ninth tick", 8, true, 9, false},
		{"tenth tick", 9, true, 10, true},
		{"eleventh tick", 10, true, 11, false},
		{"twentieth tick", 19, true, 20, true},
		{"reset by an inactive tick", 7, false, 0, false},
		{"saturated", math.MaxInt64, true, math.MaxInt64, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			run, warn := credit.NextRun(tt.run, tt.active)
			if run != tt.wantRun || warn != tt.wantWarn {
				t.Errorf("NextRun(%d, %v) = %d, %v, want %d, %v", tt.run, tt.active, run, warn, tt.wantRun, tt.wantWarn)
			}
		})
	}
}
