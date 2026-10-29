package ownuse_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ownuse"
)

// FuzzStep verifies that Step never panics and, for any inputs it accepts,
// keeps the secure ticks at or below the limit, never leaves the mode with a
// cover below Exit, never enters it with secure ticks at or above Enter,
// resets the counter on a change of the mode and keeps it otherwise, and
// returns the saturated figures with the mode untouched when it refuses.
//
// Requirement: FR-SUB-013, FR-SUB-014, NFR-SUB-002
func FuzzStep(f *testing.F) {
	f.Add(int64(2), int64(3), int64(1000), true, int64(4), int64(3), int64(100), int64(150), false, int64(7))
	f.Add(int64(2), int64(3), int64(1000), false, int64(9), int64(160), int64(12), int64(300), true, int64(200))
	f.Add(int64(1), int64(2), int64(5), true, int64(0), int64(math.MaxInt64), int64(2), int64(0), false, int64(1))
	f.Fuzz(func(t *testing.T, enter, exit, limit int64, on bool, jobs, upkeep, price, money int64, hungry bool, stock int64) {
		p := ownuse.Params{Enter: enter, Exit: exit, Max: limit}
		goods := []ownuse.Good{{Stock: stock, Upkeep: upkeep, Price: price}}
		r, ok := ownuse.Step(p, on, jobs, goods, hungry, money)
		if !ok {
			if r.On != on || r.Jobs != jobs || r.Entered || r.Left || r.Cover != limit || r.Secure != limit {
				t.Fatalf("Step(%+v, on %v, jobs %d, %+v, hungry %v, money %d) refused with %+v, want the mode kept and the figures at the limit", p, on, jobs, goods, hungry, money, r)
			}
			return
		}
		if r.Secure > limit {
			t.Fatalf("Step(%+v, %+v) secure %d above the limit %d", p, goods, r.Secure, limit)
		}
		if r.Left && (!on || r.On || r.Cover < exit) {
			t.Fatalf("Step(%+v, on %v, %+v) left with cover %d below exit %d: %+v", p, on, goods, r.Cover, exit, r)
		}
		if r.Entered && (on || !r.On || r.Secure >= enter) {
			t.Fatalf("Step(%+v, on %v, %+v) entered with secure %d not below enter %d: %+v", p, on, goods, r.Secure, enter, r)
		}
		if changed := r.Entered || r.Left; changed != (r.On != on) || (changed && r.Jobs != 0) || (!changed && r.Jobs != jobs) {
			t.Fatalf("Step(%+v, on %v, jobs %d) = %+v: mode and counter disagree", p, on, jobs, r)
		}
	})
}
