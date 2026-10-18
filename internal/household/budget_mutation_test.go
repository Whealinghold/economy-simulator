package household_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/household"
)

// TestSpendCap_OverflowReportsNotOK verifies that an overflow of H (the
// half-life times the upkeep cost) or of e + H gives 0 and false, so the
// caller can reject a configuration the validator should have caught.
//
// H = MaxInt64 * 2 does not fit. With e = MaxInt64 and H = 1 * 1, e + H = 2^63
// does not fit either, while e itself does.
//
// Requirement: FR-AGT-042
func TestSpendCap_OverflowReportsNotOK(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		e    int64
		uc   int64
		p    household.Params
	}{
		{"H overflows", 1, 2, household.Params{SpendMaxShare: 5_000, SpendHalfTicks: math.MaxInt64}},
		{"e plus H overflows", math.MaxInt64, 1, household.Params{SpendMaxShare: 5_000, SpendHalfTicks: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := household.SpendCap(tt.e, tt.uc, tt.p)
			if got != 0 || ok {
				t.Errorf("SpendCap(%d, %d, %+v) = %d, %v, want 0, false", tt.e, tt.uc, tt.p, got, ok)
			}
		})
	}
}
