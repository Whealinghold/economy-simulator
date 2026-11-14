package tenure_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/tenure"
)

// TestLeastRent_EntitlementsBeyondInt64DoNotWrap verifies that an own and a
// commons entitlement whose sum passes int64 still cover the draw: the units
// left are 0, not a wrapped positive number.
//
// Requirement: FR-LND-015, NFR-LND-003
func TestLeastRent_EntitlementsBeyondInt64DoNotWrap(t *testing.T) {
	t.Parallel()
	got, ok := tenure.LeastRent(0, 0, math.MaxInt64, math.MaxInt64, 1)
	if got != 0 || !ok {
		t.Errorf("LeastRent(0, 0, Max, Max, 1) = %d, %v, want 0, true", got, ok)
	}
	got, ok = tenure.LeastRent(5, 1, math.MaxInt64, math.MaxInt64, 7)
	if got != 0 || !ok {
		t.Errorf("LeastRent(5, 1, Max, Max, 7) = %d, %v, want 0, true", got, ok)
	}
}

// TestCharge_FreeEntitlementsBeyondInt64AreNotRentFunds verifies that a line
// whose own and commons entitlements sum beyond int64 is free for every k, so
// the whole request is served and no refusal rent_funds is reported.
//
// Requirement: FR-LND-015, NFR-LND-003
func TestCharge_FreeEntitlementsBeyondInt64AreNotRentFunds(t *testing.T) {
	t.Parallel()
	lines := []tenure.Line{{Rent: 10, Draw: 3, Own: math.MaxInt64, Commons: math.MaxInt64}}
	k, rent, ok := tenure.Charge(0, 12, lines)
	if k != 12 || rent != 0 || !ok {
		t.Errorf("Charge(0, 12) = %d, %d, %v, want 12, 0, true", k, rent, ok)
	}
	if total, ok := tenure.TotalRent(lines, 12); total != 0 || !ok {
		t.Errorf("TotalRent = %d, %v, want 0, true", total, ok)
	}
}

// TestTotalRent_NegativeInputIsRefused verifies that a negative k or a negative
// field is refused, not wrapped into a large positive count of units.
//
// Requirement: NFR-LND-003
func TestTotalRent_NegativeInputIsRefused(t *testing.T) {
	t.Parallel()
	ok := []tenure.Line{{Rent: 1, Draw: 1, Own: 1}}
	if total, valid := tenure.TotalRent(ok, math.MinInt64); total != 0 || valid {
		t.Errorf("TotalRent(MinInt64) = %d, %v, want 0, false", total, valid)
	}
	for _, l := range []tenure.Line{{Rent: -1, Draw: 1}, {Rent: 1, Draw: -1}, {Rent: 1, Draw: 1, Own: -1}, {Rent: 1, Draw: 1, Commons: -1}} {
		if total, valid := tenure.TotalRent([]tenure.Line{l}, 3); total != 0 || valid {
			t.Errorf("TotalRent(%+v) = %d, %v, want 0, false", l, total, valid)
		}
	}
}
