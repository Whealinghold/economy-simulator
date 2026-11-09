package merchant_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/merchant"
)

// TestFeeShare_Weyl verifies the fee share of spec-merchants.md 5.7.1 against the
// worked examples of 19 (models/mer/worked.out): the four brokers of example A,
// the trader of example B, the identity at a spread of 0 and the clamp to 1 and
// 10 000.
//
// Requirement: FR-MER-017, FR-MER-005
func TestFeeShare_Weyl(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		base, spread, id int64
		want             int64
		comment          string
	}{
		// u(32) = 7 770 bp: 5 000 + 2 500 * (2 * 7 770 - 10 000) / 10 000 = 5 000 + 1 385.
		{"agent 32", 5_000, 2_500, 32, 6_385, "example A"},
		// u(33) = 3 951: 5 000 + 2 500 * (7 902 - 10 000) / 10 000 = 5 000 - 524 (524.5 truncated toward zero).
		{"agent 33", 5_000, 2_500, 33, 4_476, "example A"},
		// u(34) = 131: 5 000 + 2 500 * (262 - 10 000) / 10 000 = 5 000 - 2 434 (2 434.5 truncated).
		{"agent 34", 5_000, 2_500, 34, 2_566, "example A"},
		// u(35) = 6 311: 5 000 + 2 500 * (12 622 - 10 000) / 10 000 = 5 000 + 655.
		{"agent 35", 5_000, 2_500, 35, 5_655, "example A"},
		// u(18) = 1 246: 5 000 + 2 500 * (2 492 - 10 000) / 10 000 = 5 000 - 1 877 (1 877 truncated).
		{"agent 18", 5_000, 2_500, 18, 3_123, "example B"},
		{"no spread", 5_000, 0, 32, 5_000, "the identity"},
		{"clamped below", 1, 5_000, 34, 1, "1 + 5 000 * (262 - 10 000) / 10 000 is negative"},
		{"clamped above", 10_000, 5_000, 32, 10_000, "10 000 + 5 000 * 0.554 is above the ceiling"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := merchant.FeeShare(tt.base, tt.spread, tt.id); got != tt.want {
				t.Errorf("FeeShare(%d, %d, %d) = %d, want %d (%s)", tt.base, tt.spread, tt.id, got, tt.want, tt.comment)
			}
		})
	}
}

// TestFeeShare_EvenSpread verifies that the shares of consecutive agent IDs cover
// the interval evenly: over 1 000 consecutive IDs every tenth of the spread
// holds between 80 and 120 of them, and the extremes are within the spread.
//
// Requirement: FR-MER-017
//
// Guard: the spec chooses a function of the ID over a draw because consecutive
// IDs are spread evenly; a change of the multiplier that clumped them would
// herd the merchants (L93).
func TestFeeShare_EvenSpread(t *testing.T) {
	t.Parallel()
	var bins [10]int
	for id := int64(1); id <= 1_000; id++ {
		phi := merchant.FeeShare(5_000, 2_500, id)
		if phi < 2_500 || phi > 7_500 {
			t.Fatalf("FeeShare(5000, 2500, %d) = %d, want within 2500 to 7500", id, phi)
		}
		bins[(phi-2_500)*10/5_001]++
	}
	for i, n := range bins {
		if n < 80 || n > 120 {
			t.Errorf("tenth %d of the spread holds %d of 1 000 consecutive IDs, want 80 to 120", i, n)
		}
	}
}
