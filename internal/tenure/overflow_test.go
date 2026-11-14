package tenure_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/tenure"
)

// TestSupply_WeightsBeyondInt64AreRefused verifies that unused entitlements
// whose sum passes int64 are refused with every slot 0, not wrapped into a small
// sum that would let a larger supply through.
//
// Requirement: FR-LND-013, NFR-LND-003
func TestSupply_WeightsBeyondInt64AreRefused(t *testing.T) {
	t.Parallel()
	lessors := []tenure.Part{{Ref: 0, ID: 1, Share: math.MaxInt64}, {Ref: 1, ID: 2, Share: 1}}
	out := []int64{9, 9}
	if tenure.Supply(1, lessors, out) || out[0] != 0 || out[1] != 0 {
		t.Errorf("Supply(1, %+v) = %v, want refused with 0: weights beyond int64", lessors, out)
	}
}

// TestTotalRent_SumOfLinesBeyondInt64IsRefused verifies that two lines, each
// within int64, whose rents sum beyond it, are refused as unaffordable and that
// Charge then serves fewer executions instead of wrapping the total.
//
// Requirement: FR-LND-016, NFR-LND-003
func TestTotalRent_SumOfLinesBeyondInt64IsRefused(t *testing.T) {
	t.Parallel()
	half := int64(math.MaxInt64/2 + 1) // two of them are 2^63
	lines := []tenure.Line{{Rent: half, Draw: 1}, {Rent: half, Draw: 1}}
	if total, ok := tenure.TotalRent(lines, 1); ok || total != 0 {
		t.Errorf("TotalRent(%+v, 1) = %d, %v, want refused: sum beyond int64", lines, total, ok)
	}
	if k, rent, ok := tenure.Charge(math.MaxInt64, 3, lines); !ok || k != 0 || rent != 0 {
		t.Errorf("Charge(Max, 3, %+v) = %d, %d, %v, want 0, 0, true: no k is affordable", lines, k, rent, ok)
	}
}

// TestTiers_EntitlementsBeyondInt64AreRefused verifies that holder entitlements
// whose sum passes int64 are refused with every output 0.
//
// Requirement: FR-LND-007, NFR-LND-003
func TestTiers_EntitlementsBeyondInt64AreRefused(t *testing.T) {
	t.Parallel()
	holders := []tenure.Holder{{ID: 1, Entitled: math.MaxInt64}, {ID: 2, Entitled: 5}}
	reqs := []tenure.Request{{Ref: 0, ID: 1, Principal: 1, Executions: 2, Draw: 1}}
	var s tenure.Scratch
	res, ok := s.Tiers(tenure.Params{Stock: math.MaxInt64}, holders, reqs)
	if ok || res != (tenure.Result{}) || reqs[0].Own != 0 || holders[0].Unused != 0 {
		t.Errorf("Tiers(entitlements beyond int64) = %+v, %v, own %d, want refused with 0", res, ok, reqs[0].Own)
	}
}

// TestCharge_OtherAtMaxInt64 verifies the bisection does not wrap its midpoint
// when other is MaxInt64: a free line and a line whose rent exactly fits the
// balance serve every execution asked, and a rent of 2 per execution serves half.
//
// Requirement: FR-LND-016, NFR-LND-003
//
// Guard: a wrapped midpoint answered k 0, refusing a start the principal can pay.
func TestCharge_OtherAtMaxInt64(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		available int64
		lines     []tenure.Line
		wantK     int64
		wantRent  int64
	}{
		{"free", 0, []tenure.Line{{Rent: 0, Draw: 1}}, math.MaxInt64, 0},
		{"no lines", 0, nil, math.MaxInt64, 0},
		{"exact fit", math.MaxInt64, []tenure.Line{{Rent: 1, Draw: 1}}, math.MaxInt64, math.MaxInt64},
		{"one short", math.MaxInt64 - 1, []tenure.Line{{Rent: 1, Draw: 1}}, math.MaxInt64 - 1, math.MaxInt64 - 1},
		{"half", math.MaxInt64, []tenure.Line{{Rent: 2, Draw: 1}}, math.MaxInt64 / 2, math.MaxInt64 - 1},
		{"unaffordable", 5, []tenure.Line{{Rent: 2, Draw: 1}}, 2, 4},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			k, rent, ok := tenure.Charge(tc.available, math.MaxInt64, tc.lines)
			if !ok || k != tc.wantK || rent != tc.wantRent {
				t.Errorf("Charge(%d, MaxInt64, %+v) = %d, %d, %v, want %d, %d, true", tc.available, tc.lines, k, rent, ok, tc.wantK, tc.wantRent)
			}
		})
	}
}

// TestTotalRent_NegativeKWithoutLinesIsRefused verifies a negative k is refused
// even when there is no line to refuse it.
//
// Requirement: FR-LND-016, NFR-LND-003
func TestTotalRent_NegativeKWithoutLinesIsRefused(t *testing.T) {
	t.Parallel()
	if total, ok := tenure.TotalRent(nil, -1); ok || total != 0 {
		t.Errorf("TotalRent(nil, -1) = %d, %v, want 0, false", total, ok)
	}
}
