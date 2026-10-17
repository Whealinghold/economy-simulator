package production_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/production"
)

// TestSite_NoneSeasonSkipsEntries verifies that a season of -1 (none) matches
// no entry even when a malformed table lists an entry with season -1.
//
// Requirement: FR-PRO-020
//
// Guard: dropping the season >= 0 test would let the none season read that
// entry's 1 234 instead of All.
func TestSite_NoneSeasonSkipsEntries(t *testing.T) {
	t.Parallel()
	p := &production.Profile{HasAll: true, All: 6000, Seasons: []production.SeasonBP{{-1, 1234}, {spring, 9000}}}
	if got, want := production.Site(p, true, -1), int64(6000); got != want {
		t.Errorf("Site(%+v, true, -1) = %d, want %d", p, got, want)
	}
}

// TestOutput_ZeroOperandsAndBounds verifies that a zero k, q or e is valid and
// leaves the residue, and that the residue and operand bounds are exact.
//
// Requirement: FR-PRO-024
func TestOutput_ZeroOperandsAndBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		k, q, e, residue int64
		wantN, wantRes   int64
		wantOK           bool
	}{
		// 0 * 5 * 8 500 + 1 234 = 1 234: 0 units, residue 1 234.
		{"k 0", 0, 5, 8500, 1234, 0, 1234, true},
		// 3 * 0 * 8 500 + 1 234 = 1 234: 0 units, residue 1 234.
		{"q 0", 3, 0, 8500, 1234, 0, 1234, true},
		// 3 * 2 * 0 + 77 = 77: 0 units, residue 77.
		{"e 0", 3, 2, 0, 77, 0, 77, true},
		// Residue 9 999 is the largest valid one: 0 + 9 999 < 10 000.
		{"residue 9 999", 0, 0, 0, 9999, 0, 9999, true},
		{"residue 10 001", 1, 1, 10000, 10001, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n, r, ok := production.Output(tt.k, tt.q, tt.e, tt.residue)
			if n != tt.wantN || r != tt.wantRes || ok != tt.wantOK {
				t.Errorf("Output(%d, %d, %d, %d) = %d, %d, %v, want %d, %d, %v",
					tt.k, tt.q, tt.e, tt.residue, n, r, ok, tt.wantN, tt.wantRes, tt.wantOK)
			}
		})
	}
}

// TestRescale_NegativeK verifies that a negative k is refused and nothing
// changes, as a k of 0 is.
//
// Requirement: FR-PRO-019
func TestRescale_NegativeK(t *testing.T) {
	t.Parallel()
	pending := []int64{7}
	got, ok := production.Rescale(-1, 5, pending)
	if got != -1 || ok || !slices.Equal(pending, []int64{7}) {
		t.Errorf("Rescale(-1, 5) = %d, %v with pending %v, want -1, false with [7]", got, ok, pending)
	}
}
