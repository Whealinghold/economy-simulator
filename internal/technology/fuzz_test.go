package technology_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/technology"
)

// FuzzLearned verifies that Learned never panics and that an accepted gain is
// not negative, is 0 at x = 0 and does not fall as x rises.
//
// Requirement: FR-TEC-006, FR-TEC-005
func FuzzLearned(f *testing.F) {
	f.Add(int64(200), int64(100_000), int64(500))
	f.Add(int64(10_000), int64(1), int64(1)<<50)
	f.Add(int64(0), int64(0), int64(-1))
	f.Fuzz(func(t *testing.T, d, base, x int64) {
		g, ok := technology.Learned(d, base, x)
		if !ok {
			return
		}
		if g < 0 {
			t.Fatalf("Learned(%d, %d, %d) = %d, negative", d, base, x, g)
		}
		if zero, zok := technology.Learned(d, base, 0); zok && zero != 0 {
			t.Fatalf("Learned(%d, %d, 0) = %d, want 0", d, base, zero)
		}
		if x < 1<<62 {
			if g2, ok2 := technology.Learned(d, base, x+1); ok2 && g2 < g {
				t.Fatalf("Learned(%d, %d, %d) = %d fell below %d", d, base, x+1, g2, g)
			}
		}
	})
}

// FuzzStep verifies that Step never panics on arbitrary primitives, that a
// refusal leaves the state as it was, and that an acceptance keeps the
// identity of 4.6.3 with the stored advance inside [0, CapBP].
//
// Requirement: FR-TEC-017, NFR-TEC-005
func FuzzStep(f *testing.F) {
	f.Add(int64(3000), int64(1000), int64(50), int64(137), int64(5000), int64(40), int64(2000), int64(100), uint64(7), int64(3))
	f.Add(int64(-1), int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), int64(0), uint64(0), int64(0))
	f.Add(int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, int64(1)<<62, uint64(1), int64(1)<<62)
	f.Fuzz(func(t *testing.T, capBP, x, a, trend, disc, step, catch, reach int64, key uint64, counter int64) {
		field := technology.Field{CapBP: capBP, Base: 1000, DoublingBP: 200, TrendBP: trend, DiscoveryBP: disc, StepBP: step, CatchUpBP: catch, Reach: reach}
		in := technology.Input{
			Cells:         []technology.Cell{{Experience: x, Advance: a}, {}},
			Counters:      []int64{counter, 1},
			TrendCarry:    []int64{0},
			Fields:        []technology.Field{field},
			Regions:       2,
			Tick:          5,
			TicksPerYear:  365,
			MaxExperience: 1 << 50,
			Key:           key,
			Dist:          func(a, b int) (int64, bool) { return 10, true },
		}
		before := []technology.Cell{in.Cells[0], in.Cells[1]}
		out := make([]technology.Result, 2)
		var s technology.Scratch
		if !technology.Step(in, &s, out) {
			if in.Cells[0] != before[0] || in.Cells[1] != before[1] || in.TrendCarry[0] != 0 {
				t.Fatalf("refusal changed the state: %+v", in.Cells)
			}
			return
		}
		checkResults(t, out, capBP)
	})
}

// checkResults asserts the identity and the ranges of an accepted step.
func checkResults(t *testing.T, out []technology.Result, capBP int64) {
	t.Helper()
	for i, r := range out {
		if r.Clamped < 0 || r.After < r.Before || r.Advance < 0 || r.Advance > capBP {
			t.Fatalf("cell %d: result out of range %+v for cap %d", i, r, capBP)
		}
		if r.Before+r.Doing+r.Trend+r.Discovery+r.CatchUp-r.Clamped != r.After {
			t.Fatalf("cell %d: identity broken %+v", i, r)
		}
	}
}
