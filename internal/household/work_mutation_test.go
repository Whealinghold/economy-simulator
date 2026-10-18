package household_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/household"
)

// TestNet_ValidOutputKeepsOverflow verifies that once a Net has overflowed, a later
// valid Output does not clear the failure.
//
// Two costs of MaxInt64 overflow the sum; the Output of 1 * 10 000 * 10 000 /
// 10 000 = 10 000 is valid on its own and must not reset ok.
//
// Guard: a cleared flag would let a wrapped cost pass as a real net value.
func TestNet_ValidOutputKeepsOverflow(t *testing.T) {
	t.Parallel()
	var n household.Net
	n.Cost(math.MaxInt64)
	n.Cost(math.MaxInt64)
	n.Output(1, 10_000, 10_000)
	if got, ok := n.Value(); ok {
		t.Errorf("Net after an overflow then a valid Output = %d, true, want not ok", got)
	}
}

// TestOutputTarget_NegativeDurationIsNotOK verifies that a duration below 0 is
// refused, not divided by.
//
// 1 * 1 * 10 000 * 3 / (10 000 * -1) would be -3 with ok true if the duration
// were not checked.
//
// Requirement: FR-AGT-036
func TestOutputTarget_NegativeDurationIsNotOK(t *testing.T) {
	t.Parallel()
	got, ok := household.OutputTarget(1, 1, 10_000, 3, -1)
	if got != 0 || ok {
		t.Errorf("OutputTarget(1, 1, 10000, 3, -1) = %d, %v, want 0, false", got, ok)
	}
}
