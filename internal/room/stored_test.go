package room_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/room"
)

// TestStoredLoad_Formula verifies 8.3.2: min(stores' capacity, max(0, load -
// weight * base)) per dimension, in a table of the three regimes (the load is
// within the household's own room, between it and the stores, and above both)
// and the overflow guard.
//
// Requirement: FR-STO-042
func TestStoredLoad_Formula(t *testing.T) {
	t.Parallel()
	base := room.Pair{Mass: 100, Bulk: 50}
	caps := room.Pair{Mass: 200, Bulk: 80}
	for _, tt := range []struct {
		name   string
		load   room.Pair
		weight int64
		base   room.Pair
		want   room.Pair
	}{
		{"within the own room", room.Pair{Mass: 250, Bulk: 100}, 3, base, room.Pair{}},
		{"into the stores", room.Pair{Mass: 350, Bulk: 190}, 3, base, room.Pair{Mass: 50, Bulk: 40}},
		{"above both", room.Pair{Mass: 900, Bulk: 900}, 3, base, caps},
		{"weight times base overflows", room.Pair{Mass: 5, Bulk: 5}, math.MaxInt64, base, room.Pair{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := room.StoredLoad(caps, tt.load, tt.base, tt.weight); got != tt.want {
				t.Errorf("StoredLoad(%v, %v, %v, %d) = %v, want %v", caps, tt.load, tt.base, tt.weight, got, tt.want)
			}
		})
	}
}

// TestOf_Formula verifies the room of an account (3.3.2): weight times the base
// room plus each store's capacity, a store not multiplied by the weight, and the
// overflow flag.
//
// Example 18.4: weight 10, base (100, 100), one store (200, 200): room (1200,
// 1200); at weight 5 it falls to (700, 700).
//
// Requirement: FR-STO-004
func TestOf_Formula(t *testing.T) {
	t.Parallel()
	base := room.Pair{Mass: 100, Bulk: 100}
	stores := []room.Pair{{Mass: 200, Bulk: 200}}
	for _, tt := range []struct {
		name   string
		weight int64
		base   room.Pair
		stores []room.Pair
		want   room.Pair
		ok     bool
	}{
		{"weight 10", 10, base, stores, room.Pair{Mass: 1200, Bulk: 1200}, true},
		{"weight 5", 5, base, stores, room.Pair{Mass: 700, Bulk: 700}, true},
		{"no store", 5, base, nil, room.Pair{Mass: 500, Bulk: 500}, true},
		{"two stores of a class", 1, base, []room.Pair{{Mass: 1, Bulk: 2}, {Mass: 1, Bulk: 2}}, room.Pair{Mass: 102, Bulk: 104}, true},
		{"base overflows", math.MaxInt64, base, nil, room.Pair{}, false},
		{"stores overflow", 1, room.Pair{Mass: math.MaxInt64}, stores, room.Pair{}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := room.Of(tt.weight, tt.base, tt.stores)
			if ok != tt.ok || (ok && got != tt.want) {
				t.Errorf("Of(%d, %v, %v) = %v, %v, want %v, %v", tt.weight, tt.base, tt.stores, got, ok, tt.want, tt.ok)
			}
		})
	}
}
