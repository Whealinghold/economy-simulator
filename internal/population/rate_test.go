package population_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/population"
)

var (
	// The default schedules of spec-population.md 8.
	defBirth = []population.Knot{{Ratio: 8_000, Rate: 250}, {Ratio: 16_000, Rate: 250}}
	defDeath = []population.Knot{{Ratio: 8_000, Rate: 450}, {Ratio: 16_000, Rate: 50}}
	// The habitat of spec-population.md 12.2.
	habBirth = []population.Knot{{Ratio: 6_000, Rate: 300}, {Ratio: 18_000, Rate: 700}}
	habDeath = []population.Knot{{Ratio: 6_000, Rate: 900}, {Ratio: 18_000, Rate: 100}}
)

// TestRate_Piecewise verifies the schedule is flat outside its first and last
// knot and linear between two, with the increment truncated toward zero also on
// a falling segment.
//
// Requirement: FR-POP-004
func TestRate_Piecewise(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		knots []population.Knot
		ratio int64
		want  int64
	}{
		{"below the first knot", defDeath, 5_000, 450},
		{"above the last knot", defDeath, 20_000, 50},
		// 450 + trunc(-400 * (14 000 - 8 000) / 8 000) = 450 - 300 (12.1).
		{"falling segment", defDeath, 14_000, 150},
		// 450 + trunc(-400 * 1 / 8 000) = 450 + trunc(-0.05) = 450: toward zero, not floor.
		{"truncation toward zero on a fall", defDeath, 8_001, 450},
		// 300 + trunc(400 * 3 000 / 12 000) = 400 (12.2).
		{"rising segment", habBirth, 9_000, 400},
		{"single knot is constant", []population.Knot{{Ratio: 5_000, Rate: 70}}, 99_999, 70},
		{"negative ratio is below the first knot", defBirth, -3, 250},
		{"at a knot", defDeath, 16_000, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, clamped, ok := population.Rate(tt.knots, tt.ratio, 10_000)
			if !ok || clamped || got != tt.want {
				t.Errorf("Rate(%v, %d, 10000) = %d, %v, %v, want %d, false, true", tt.knots, tt.ratio, got, clamped, ok, tt.want)
			}
		})
	}
}

// TestRate_ScaleAndClamp verifies the region scale multiplies the schedule in
// basis points, truncating, and that a result above 10 000 is capped and says so.
//
// Requirement: FR-POP-005
func TestRate_ScaleAndClamp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		knots       []population.Knot
		ratio, bp   int64
		want        int64
		wantClamped bool
	}{
		// min(10 000, trunc(700 * 30 000 / 10 000)) = 2 100 (12.2).
		{"disease scale", habDeath, 9_000, 30_000, 2_100, false},
		// 900 * 2 = 1 800 only; the clamp needs a large rate: 6 000 * 30 000 / 10 000 = 18 000 -> 10 000.
		{"clamps at 10 000", []population.Knot{{Ratio: 0, Rate: 6_000}}, 0, 30_000, 10_000, true},
		{"exactly 10 000 is not clamped", []population.Knot{{Ratio: 0, Rate: 5_000}}, 0, 20_000, 10_000, false},
		// trunc(250 * 3 333 / 10 000) = trunc(83.325) = 83.
		{"scale truncates", defBirth, 10_000, 3_333, 83, false},
		{"scale 0 is no event", defBirth, 10_000, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, clamped, ok := population.Rate(tt.knots, tt.ratio, tt.bp)
			if !ok || got != tt.want || clamped != tt.wantClamped {
				t.Errorf("Rate(%v, %d, %d) = %d, %v, %v, want %d, %v, true", tt.knots, tt.ratio, tt.bp, got, clamped, ok, tt.want, tt.wantClamped)
			}
		})
	}
}

// TestRate_Refuses verifies Rate reports instead of guessing on an empty
// schedule, a negative scale or rate, and an overflow.
//
// Guard: the root treats ok false as a violated invariant after validation; a
// silent 0 would hide a schedule the validator let through.
func TestRate_Refuses(t *testing.T) {
	t.Parallel()
	huge := []population.Knot{{Ratio: 0, Rate: 1 << 62}}
	tests := []struct {
		name  string
		knots []population.Knot
		bp    int64
	}{
		{"empty schedule", nil, 10_000},
		{"negative scale", defBirth, -1},
		{"negative rate", []population.Knot{{Ratio: 0, Rate: -5}}, 10_000},
		{"overflow", huge, 1 << 62},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, _, ok := population.Rate(tt.knots, 0, tt.bp); ok {
				t.Errorf("Rate(%v, 0, %d) = %d, ok true, want ok false", tt.knots, tt.bp, got)
			}
		})
	}
}

// TestRate_NoAllocation verifies the schedule evaluation allocates nothing for
// a schedule within the stack buffer.
//
// Guard: the step evaluates a rate per agent per tick; a conversion that
// escaped would make NFR-POP-001 depend on the number of agents.
func TestRate_NoAllocation(t *testing.T) {
	if n := testing.AllocsPerRun(100, func() { population.Rate(defDeath, 12_000, 10_000) }); n != 0 {
		t.Errorf("Rate allocated %v times per call, want 0", n)
	}
}

// FuzzRate verifies Rate never panics and that an accepted result lies in
// [0, 10 000] with clamped true only at the cap.
//
// Requirement: FR-POP-004, FR-POP-005
func FuzzRate(f *testing.F) {
	f.Add(int64(8_000), int64(450), int64(16_000), int64(50), int64(14_000), int64(10_000))
	f.Add(int64(0), int64(10_000), int64(1), int64(0), int64(-5), int64(1<<62))
	f.Fuzz(func(t *testing.T, x0, y0, x1, y1, ratio, scale int64) {
		knots := []population.Knot{{Ratio: x0, Rate: y0}, {Ratio: x1, Rate: y1}}
		if x1 <= x0 {
			knots = knots[:1]
		}
		got, clamped, ok := population.Rate(knots, ratio, scale)
		if !ok {
			return
		}
		if got < 0 || got > 10_000 {
			t.Errorf("Rate(%v, %d, %d) = %d, outside [0, 10000]", knots, ratio, scale, got)
		}
		if clamped && got != 10_000 {
			t.Errorf("Rate(%v, %d, %d) clamped at %d, want 10000", knots, ratio, scale, got)
		}
	})
}
