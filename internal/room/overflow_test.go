package room_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/room"
)

// TestKRoom_OutputLoadSumThatOverflowsGivesZero verifies that outputs whose
// loads each fit int64 but whose sum does not give k = 0, the safe side, not a
// wrapped sum that reads as a small load.
//
// Requirement: FR-STO-012
func TestKRoom_OutputLoadSumThatOverflowsGivesZero(t *testing.T) {
	t.Parallel()
	// Each output is 4e14 units at E = 10 000 basis points, so its load times D
	// is 4e18, below MaxInt64 (about 9.22e18); three of them sum to 1.2e19.
	one := room.Output{Item: room.Item{Qty: 4e14, Size: room.Pair{Mass: 1}}}
	outs := []room.Output{one, one, one}
	free := room.Pair{Mass: math.MaxInt64, Bulk: math.MaxInt64}
	if got := room.KRoom(nil, outs, room.Denominator, free, room.Pair{}, 5); got != 0 {
		t.Errorf("KRoom(output load sum above int64) = %d, want 0", got)
	}
	prod := []room.Output{{Item: room.Item{Qty: math.MaxInt64, Size: room.Pair{Mass: 2}}}}
	if got := room.KRoom(nil, prod, room.Denominator, free, room.Pair{}, 5); got != 0 {
		t.Errorf("KRoom(output load product above int64) = %d, want 0", got)
	}
}

// TestExcessOrder_NegativeLoadBelowAnyShortfall verifies that a load so far
// below the room that load - room overflows downward is no shortfall: nothing
// goes and the order is met.
//
// Requirement: FR-STO-018
func TestExcessOrder_NegativeLoadBelowAnyShortfall(t *testing.T) {
	t.Parallel()
	cands := []room.Candidate{{Commodity: 1, Size: room.Pair{Mass: 1, Bulk: 1}, Lots: []int64{7}}}
	steps, met := room.ExcessOrder(room.Pair{Mass: math.MinInt64, Bulk: 0}, room.Pair{Mass: 1, Bulk: 1}, cands)
	if !met || len(steps) != 0 {
		t.Errorf("ExcessOrder(load MinInt64) = %v, %v, want no steps, true", steps, met)
	}
}

// TestScale_AvailThatOverflowsIsZero verifies that a room less the pledge, or
// less the upkeep room, that does not fit int64 leaves no room: the scale is 0
// for a target that has a size, and 10 000 for targets that take none.
//
// Guard: a wrapped avail would read as a large room and scale the targets up
// to their full value in a room that was cut below zero.
//
// Requirement: FR-STO-022
func TestScale_AvailThatOverflowsIsZero(t *testing.T) {
	t.Parallel()
	targets := []room.Item{{Qty: 1, Size: room.Pair{Mass: 1, Bulk: 1}}}
	for _, tt := range []struct {
		name                 string
		room, pledge, upkeep room.Pair
		targets              []room.Item
		want                 int64
	}{
		{"room less pledge underflows", room.Pair{Mass: math.MinInt64, Bulk: 5}, room.Pair{Mass: 1}, room.Pair{}, targets, 0},
		{"less upkeep underflows", room.Pair{Mass: math.MinInt64 + 1, Bulk: 5}, room.Pair{Mass: 1}, room.Pair{Mass: 1}, targets, 0},
		{"targets that take none keep the full scale", room.Pair{Mass: math.MinInt64}, room.Pair{Mass: 1}, room.Pair{}, []room.Item{{Qty: 4}}, room.Denominator},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := room.Scale(tt.targets, tt.room, tt.pledge, tt.upkeep); got != tt.want {
				t.Errorf("Scale(%v, %v, %v, %v) = %d, want %d", tt.targets, tt.room, tt.pledge, tt.upkeep, got, tt.want)
			}
		})
	}
}
