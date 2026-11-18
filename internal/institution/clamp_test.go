package institution_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/institution"
)

// TestClamp_OneSide verifies that a bid above the cap is lowered to it, an ask below
// the floor is raised to it, an order inside the range is unchanged, and a level of 0
// is none.
//
// Requirement: FR-ORG-030
func TestClamp_OneSide(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		limit       int64
		bid         bool
		cap, floor  int64
		want        int64
		wantChanged bool
	}{
		{"bid above the cap", 1500, true, 1200, 0, 1200, true},
		{"bid at the cap", 1200, true, 1200, 0, 1200, false},
		{"bid below the cap", 900, true, 1200, 0, 900, false},
		{"bid and no cap", 1500, true, 0, 800, 1500, false},
		{"a floor does not touch a bid", 100, true, 0, 800, 100, false},
		{"ask below the floor", 700, false, 1200, 800, 800, true},
		{"ask at the floor", 800, false, 0, 800, 800, false},
		{"ask above the floor", 900, false, 0, 800, 900, false},
		{"ask and no floor", 100, false, 1200, 0, 100, false},
		{"a cap does not touch an ask", 5000, false, 1200, 0, 5000, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, changed := institution.Clamp(tt.limit, tt.bid, tt.cap, tt.floor)
			if got != tt.want || changed != tt.wantChanged {
				t.Errorf("Clamp(%d, bid=%v, %d, %d) = %d, %v, want %d, %v", tt.limit, tt.bid, tt.cap, tt.floor, got, changed, tt.want, tt.wantChanged)
			}
		})
	}
}

// TestClamp_ExampleB verifies example B of spec-institutions.md 13.2: with a cap of
// 1 200 six bids and five asks give three bids lowered, nine units clamped, no ask
// raised, and every limit inside the range afterwards.
//
// Requirement: FR-ORG-030, FR-ORG-031, FR-ORG-032
func TestClamp_ExampleB(t *testing.T) {
	t.Parallel()
	bids := [][2]int64{{1500, 3}, {1400, 2}, {1250, 4}, {1200, 3}, {1100, 5}, {900, 2}}
	asks := [][2]int64{{800, 4}, {1000, 4}, {1150, 3}, {1300, 5}, {1500, 4}}
	var tally institution.Tally
	for _, b := range bids {
		got, changed := institution.Clamp(b[0], true, 1200, 0)
		tally.Add(true, changed, b[1])
		if got > 1200 {
			t.Errorf("bid %d clamped to %d, want at most the cap 1200", b[0], got)
		}
	}
	for _, a := range asks {
		got, changed := institution.Clamp(a[0], false, 1200, 0)
		tally.Add(false, changed, a[1])
		if got != a[0] {
			t.Errorf("ask %d changed to %d with no floor, want unchanged", a[0], got)
		}
	}
	// The three bids above 1 200 are (1 500, 3), (1 400, 2) and (1 250, 4): 9 units.
	if want := (institution.Tally{Lowered: 3, Units: 9}); tally != want {
		t.Errorf("tally = %+v, want %+v", tally, want)
	}
}

// TestTally_AddSaturates verifies that the units of a tally saturate instead of
// wrapping, and that an unchanged order counts nothing.
//
// Guard: architecture.md 3.5, a count that wraps would tell a negative number of units.
func TestTally_AddSaturates(t *testing.T) {
	t.Parallel()
	var tally institution.Tally
	if tally.Add(true, false, 5) || tally != (institution.Tally{}) {
		t.Errorf("an unchanged order counted: %+v", tally)
	}
	const big = int64(^uint64(0) >> 1)
	tally.Add(true, true, big)
	tally.Add(false, true, 1)
	if tally.Units != big || tally.Lowered != 1 || tally.Raised != 1 {
		t.Errorf("tally = %+v, want Units saturated at %d, one lowered, one raised", tally, big)
	}
	tally.Add(true, true, -3)
	if tally.Units != big {
		t.Errorf("a negative quantity changed the units: %d", tally.Units)
	}
}
