package occupation_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/occupation"
)

// TestMutation_Funded_OneEntrantUnlimited verifies that a single entrant at an
// amount of 0 is created: the n <= 0 guard does not catch n = 1.
//
// Requirement: FR-SPN-008
func TestMutation_Funded_OneEntrantUnlimited(t *testing.T) {
	t.Parallel()
	if got := occupation.Funded(1, 0, 0); got != 1 {
		t.Errorf("Funded(1, 0, 0) = %d, want 1", got)
	}
}

// TestMutation_Exits_CutCountedPastFirstExhausted verifies that every leaver
// that finds the allowance used up adds its want to the cut: three leavers
// behind an exhausted allowance cut 1 + 1 + 1.
//
// Requirement: FR-SPN-014
func TestMutation_Exits_CutCountedPastFirstExhausted(t *testing.T) {
	t.Parallel()
	ls := []occupation.Leaver{
		{Ref: 0, Privation: 9, Weight: 1, Unfed: 1},
		{Ref: 1, Privation: 8, Weight: 1, Unfed: 1},
		{Ref: 2, Privation: 7, Weight: 1, Unfed: 1},
		{Ref: 3, Privation: 6, Weight: 1, Unfed: 1},
	}
	out, cut := occupation.Exits(nil, ls, 1, 1000)
	if len(out) != 1 || cut != 3 {
		t.Errorf("Exits = %+v, cut %d, want 1 exit, cut 3", out, cut)
	}
}

// TestMutation_ReviewDue_NegativeTickKeepsResidue verifies that a negative tick
// reviews exactly when its residue class does (the remainder of a negative sum
// is never taken as a review).
//
// Requirement: FR-SPN-018
func TestMutation_ReviewDue_NegativeTickKeepsResidue(t *testing.T) {
	t.Parallel()
	const period = 5
	for id := int64(0); id < 50; id++ {
		if got, want := occupation.ReviewDue(id, -1, period), occupation.ReviewDue(id, period-1, period); got != want {
			t.Errorf("ReviewDue(%d, -1, %d) = %v, want %v", id, period, got, want)
		}
	}
}

// TestMutation_Expected_NegativeTicksPerYear verifies that a negative
// ticksPerYear gives no signal, as 0 does.
//
// Requirement: FR-SPN-017
func TestMutation_Expected_NegativeTicksPerYear(t *testing.T) {
	t.Parallel()
	if e, has := occupation.Expected(100, 1, 1, -1); e != 0 || has {
		t.Errorf("Expected(100, 1, 1, -1) = %d, %v, want 0, false", e, has)
	}
}

// TestMutation_Expected_SingleQuotientWhenDenominatorFits verifies that when
// upkeep * ticksPerYear fits, the exact single quotient is used and not the
// two-step one, whose first quotient 1e21 leaves int64 although the answer
// 1e12 * 1e9 / 1e5 = 1e16 does not.
//
// Requirement: FR-SPN-017
func TestMutation_Expected_SingleQuotientWhenDenominatorFits(t *testing.T) {
	t.Parallel()
	if e, has := occupation.Expected(1e12, 100_000, 1, 100_000); e != 1e16 || !has {
		t.Errorf("Expected = %d, %v, want 1e16, true", e, has)
	}
}

// TestMutation_CapitalOK_NegativeInputsDenied verifies that a negative
// toolPrice or capitalReturn denies the test even when the unsigned products
// would pass: with diff * ut = 2^62 * 2^62 the wrapped product of the other
// factor with 1 is far below it.
//
// Requirement: FR-SPN-058
func TestMutation_CapitalOK_NegativeInputsDenied(t *testing.T) {
	t.Parallel()
	const big = int64(1) << 62
	tests := []struct {
		name                 string
		toolPrice, capReturn int64
	}{
		{"negative tool price", -1, 1},
		{"negative capital return", 1, -1},
		{"negative tool price, zero return", -1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if occupation.CapitalOK(big, 0, tt.toolPrice, tt.capReturn, big, 1) {
				t.Errorf("CapitalOK(tool %d, return %d) = true, want false", tt.toolPrice, tt.capReturn)
			}
		})
	}
	if !occupation.CapitalOK(math.MaxInt64, 0, 1, 1, 1, 1) {
		t.Error("CapitalOK positive control = false, want true")
	}
}
