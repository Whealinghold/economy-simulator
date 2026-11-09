package population_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/population"
)

// TestCut_FloorAndFed verifies the per-agent cap at the fed households and the
// group floor, with the deaths beyond each counted apart.
//
// Requirement: FR-POP-013, FR-POP-015
func TestCut_FloorAndFed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                        string
		due, fed, group, floor      int64
		wantTake, wantCut, wantHeld int64
	}{
		{"all taken", 2, 36, 100, 1, 2, 0, 0},
		// 4 due, 3 fed: one beyond the fed households is cut.
		{"cut at the fed households", 4, 3, 100, 1, 3, 1, 0},
		// MinHouseholds 1 holds the last household of a group of 1 (12.2).
		{"floor holds the last household", 1, 1, 1, 1, 0, 0, 1},
		// A group of 5 with floor 4 allows 1 of 3 due.
		{"floor allows part", 3, 10, 5, 4, 1, 0, 2},
		{"floor 0 holds nothing", 3, 3, 3, 0, 3, 0, 0},
		// 5 due, 2 fed: 3 cut; the 2 taken then meet a room of 1: 1 held.
		{"cap then floor", 5, 2, 4, 3, 1, 3, 1},
		{"group already below the floor", 2, 5, 0, 1, 0, 0, 2},
		{"negative inputs are 0", -3, 5, 5, 1, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			take, cut, held := population.Cut(tt.due, tt.fed, tt.group, tt.floor)
			if take != tt.wantTake || cut != tt.wantCut || held != tt.wantHeld {
				t.Errorf("Cut(%d, %d, %d, %d) = %d, %d, %d, want %d, %d, %d",
					tt.due, tt.fed, tt.group, tt.floor, take, cut, held, tt.wantTake, tt.wantCut, tt.wantHeld)
			}
		})
	}
}

// TestAllow_Cap verifies the household cap drops the births beyond it, and that
// a cap of 0 is no cap.
//
// Requirement: FR-POP-014
func TestAllow_Cap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                     string
		due, households, cap     int64
		wantAllowed, wantDropped int64
	}{
		// min(3, 600 - 598) = 2 born, 1 dropped (12.2).
		{"two of three", 3, 598, 600, 2, 1},
		{"room for all", 3, 10, 600, 3, 0},
		{"at the cap", 2, 600, 600, 0, 2},
		{"above the cap after a shrink of the cap", 2, 700, 600, 0, 2},
		{"cap 0 is none", 5, 1_000_000, 0, 5, 0},
		{"nothing due", 0, 5, 6, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a, d := population.Allow(tt.due, tt.households, tt.cap)
			if a != tt.wantAllowed || d != tt.wantDropped {
				t.Errorf("Allow(%d, %d, %d) = %d, %d, want %d, %d", tt.due, tt.households, tt.cap, a, d, tt.wantAllowed, tt.wantDropped)
			}
		})
	}
}
