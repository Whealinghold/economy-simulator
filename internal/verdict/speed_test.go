package verdict_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// handRegrow is the per-tick integer iteration of POP 3.8 written out plainly
// with the carried remainder, for a small case where nothing can overflow.
func handRegrow(h, target, rate, tpy int64) int64 {
	den := 10000 * tpy
	var carry, t int64
	for h < target {
		num := h*rate + carry
		h += num / den
		carry = num % den
		t++
	}
	return t
}

// TestRegrowTicks_CarriedRemainder verifies the speed limit of STY-S02
// against a hand iteration of POP 3.8, with the remainder carried across ticks.
//
// Expected (worked.out line 18, STY 17.4): 820 households regrow to 900, the
// lower edge of the band 1 000 bp under 1 000, at 200 bp a year over 365 ticks
// a year in 1 700 ticks (4 years and 240 ticks); whole-year compounding gives
// 5 years, the coarser reading, not used.
//
// Requirement: FR-STY-036, FR-STY-063
func TestRegrowTicks_CarriedRemainder(t *testing.T) {
	t.Parallel()
	got, ok := verdict.RegrowTicks(820, 1000, 1000, 200, 365)
	if !ok || got != 1700 {
		t.Errorf("RegrowTicks(820, 1000, 1000, 200, 365) = %d, %v, want 1700, true", got, ok)
	}
	for _, tc := range []struct{ trough, level, band, rate, tpy int64 }{
		{820, 1000, 1000, 200, 365},
		{50, 100, 1000, 20000, 12},
		{5, 10, 0, 50000, 4},
		{100, 1000, 5000, 3000, 365},
	} {
		target := tc.level * (10000 - tc.band) / 10000
		want := handRegrow(tc.trough, target, tc.rate, tc.tpy)
		if got, ok := verdict.RegrowTicks(tc.trough, tc.level, tc.band, tc.rate, tc.tpy); !ok || got != want {
			t.Errorf("RegrowTicks(%+v) = %d, %v, want %d, true (hand iteration)", tc, got, ok, want)
		}
	}
	t.Run("a trough at the level takes no ticks", func(t *testing.T) {
		if got, ok := verdict.RegrowTicks(900, 1000, 1000, 200, 365); !ok || got != 0 {
			t.Errorf("RegrowTicks(900, 1000, 1000, ...) = %d, %v, want 0, true", got, ok)
		}
	})
	t.Run("unreachable", func(t *testing.T) {
		// a zero rate, a trough of 0, a rate of 1 bp a year over 1 000 years
		// (9% of 1 000 households never), a negative band, and a bad calendar.
		for _, tc := range []struct{ trough, level, band, rate, tpy int64 }{
			{820, 1000, 1000, 0, 365},
			{820, 1000, 1000, -5, 365},
			{0, 1000, 1000, 200, 365},
			{820, 1000, 1000, 200, 0},
			{820, 1000, 1000, 200, 1 << 20},
			{1, 1000000, 0, 1, 365},
			{820, 1 << 62, 0, 200, 365},
		} {
			if got, ok := verdict.RegrowTicks(tc.trough, tc.level, tc.band, tc.rate, tc.tpy); ok {
				t.Errorf("RegrowTicks(%+v) = %d, true, want unreachable", tc, got)
			}
		}
	})
	t.Run("extremes do not wrap", func(t *testing.T) {
		const maxI = 1<<63 - 1
		if _, ok := verdict.RegrowTicks(maxI-1, maxI, 0, maxI, 365); !ok {
			// reached in one tick or reported unreachable; it must not panic
			t.Log("unreachable at the top, accepted")
		}
		if _, ok := verdict.RegrowTicks(1, maxI, 0, maxI, 365); ok {
			t.Errorf("RegrowTicks(1, max, 0, max, 365) reachable, want the overflow guard")
		}
	})
}
