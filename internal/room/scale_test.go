package room_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/room"
)

// TestScale_Floors verifies f of 6.2.2: the least floor(Avail*D/TL) over the
// dimensions that bind, D where none binds, floored not rounded.
//
// Targets 10 of size (3, 1) and 4 of size (1, 5) load (30+4, 10+20) = (34, 30).
// Room (100, 20) with pledge (10, 0) and upkeep (6, 0) leaves Avail (84, 20):
// mass fits (34 <= 84), bulk binds: f = floor(20*10000/30) = 6666 (6666.67).
//
// Requirement: FR-STO-022
func TestScale_Floors(t *testing.T) {
	t.Parallel()
	targets := []room.Item{{Qty: 10, Size: room.Pair{Mass: 3, Bulk: 1}}, {Qty: 4, Size: room.Pair{Mass: 1, Bulk: 5}}}
	for _, tt := range []struct {
		name                string
		roomP, pledge, upkp room.Pair
		want                int64
	}{
		{"bulk binds", room.Pair{Mass: 100, Bulk: 20}, room.Pair{Mass: 10}, room.Pair{Mass: 6}, 6666},
		{"neither binds", room.Pair{Mass: 100, Bulk: 100}, room.Pair{}, room.Pair{}, 10_000},
		{"exactly full", room.Pair{Mass: 34, Bulk: 30}, room.Pair{}, room.Pair{}, 10_000},
		{"mass binds more", room.Pair{Mass: 17, Bulk: 70}, room.Pair{}, room.Pair{}, 5_000},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := room.Scale(targets, tt.roomP, tt.pledge, tt.upkp); got != tt.want {
				t.Errorf("Scale(%v, room %v, pledge %v, upkeep %v) = %d, want %d", targets, tt.roomP, tt.pledge, tt.upkp, got, tt.want)
			}
		})
	}
}

// TestScale_ClampsAvailAtZero verifies that Avail is clamped at 0 after a cut
// of the room (6.2.3): a pledge above the room gives f = 0, and so does a load
// that overflows.
//
// Requirement: FR-STO-022
func TestScale_ClampsAvailAtZero(t *testing.T) {
	t.Parallel()
	one := []room.Item{{Qty: 5, Size: room.Pair{Mass: 1, Bulk: 1}}}
	for _, tt := range []struct {
		name    string
		targets []room.Item
		roomP   room.Pair
		pledge  room.Pair
		upkeep  room.Pair
	}{
		{"pledge above room", one, room.Pair{Mass: 700, Bulk: 700}, room.Pair{Mass: 940, Bulk: 940}, room.Pair{}},
		{"upkeep above avail", one, room.Pair{Mass: 10, Bulk: 10}, room.Pair{Mass: 5, Bulk: 5}, room.Pair{Mass: 6, Bulk: 6}},
		{"load overflows", []room.Item{{Qty: math.MaxInt64, Size: room.Pair{Mass: 2}}}, room.Pair{Mass: 9, Bulk: 9}, room.Pair{}, room.Pair{}},
		{"difference overflows", one, room.Pair{Mass: math.MinInt64, Bulk: 1}, room.Pair{Mass: 1}, room.Pair{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := room.Scale(tt.targets, tt.roomP, tt.pledge, tt.upkeep); got != 0 {
				t.Errorf("Scale(%v, %v, %v, %v) = %d, want 0", tt.targets, tt.roomP, tt.pledge, tt.upkeep, got)
			}
		})
	}
}

// TestScaleQty_Floors verifies Target' = floor(Target * f / D) and the guards
// of the helper.
//
// Requirement: FR-STO-022
func TestScaleQty_Floors(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		qty, f, want int64
	}{
		{10, 7142, 7},                        // 71 420 / 10 000 = 7.142
		{10, 10_000, 10},                     // no scaling
		{10, 0, 0},                           // f = 0
		{0, 5_000, 0},                        // nothing targeted
		{-1, 5_000, 0},                       // a negative target is none
		{5, -1, 0},                           // a negative f is none
		{math.MaxInt64, 2, 1844674407370955}, // 9.22e18 * 2 / 10 000 = 1.8446744e15, exact in 128 bits
		{math.MaxInt64, math.MaxInt64, 0},    // f above D with a quotient that does not fit int64: guarded
	} {
		if got := room.ScaleQty(tt.qty, tt.f); got != tt.want {
			t.Errorf("ScaleQty(%d, %d) = %d, want %d", tt.qty, tt.f, got, tt.want)
		}
	}
}
