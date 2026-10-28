package economysim

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// TestStartWindows_WrapsAcrossTheYearEnd verifies that a job started late in the
// year and completed early in the next is counted in the window it started in.
//
// Arithmetic: a year of 10 ticks, open on ticks 6 to 9 (the window [6, 10)). A
// job of duration 3 completing at tick 1 started at 1 - 3 + 1 = -1, that is tick
// 9 of the year, inside the window; a completion at tick 8 started at 6.
//
// Fix: stage-2-9c review. The start was not taken modulo the year, so the first
// completion matched no window and the window's draw was undercounted.
func TestStartWindows_WrapsAcrossTheYearEnd(t *testing.T) {
	t.Parallel()
	a := viaActivity{
		runs:  []steady.Run{{Length: 6, FactorBP: 0}, {Length: 4, FactorBP: 10_000}},
		count: steady.StartCount{Completions: []int64{1, 8}},
		act:   steady.Activity{Duration: 3},
	}
	want := []steady.Window{{Start: 6, End: 10, Starts: 2}}
	if got := startWindows(a); !slices.Equal(got, want) {
		t.Errorf("startWindows(year 10, window [6,10), completions 1 and 8, duration 3) = %v, want %v", got, want)
	}
}
