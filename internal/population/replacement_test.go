package population_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/population"
)

// TestReplacement verifies the replacement ratio on the knots of the two
// schedules: the worked examples, a crossing inside a union segment, g = 0 at a
// knot, and the schedules that never cross from negative to positive.
//
// Requirement: FR-POP-029
func TestReplacement(t *testing.T) {
	t.Parallel()
	k := func(xy ...int64) []population.Knot {
		var ks []population.Knot
		for i := 0; i < len(xy); i += 2 {
			ks = append(ks, population.Knot{Ratio: xy[i], Rate: xy[i+1]})
		}
		return ks
	}
	tests := []struct {
		name         string
		birth, death []population.Knot
		want, lo, hi int64
		wantOK       bool
	}{
		// g(8 000) = -200, g(16 000) = +200: 8 000 + trunc(200 * 8 000 / 400) = 12 000 (12.1).
		{"defaults", defBirth, defDeath, 12_000, 8_000, 16_000, true},
		// g(6 000) = -600, g(18 000) = +600: 6 000 + 600 * 12 000 / 1 200 = 12 000 (12.2).
		{"habitat", habBirth, habDeath, 12_000, 6_000, 18_000, true},
		// Union {0, 100, 300}: g = -10 at 0, -10 at 100, +30 at 300 (birth flat 40, death 50, 50, 10).
		// 100 + trunc(10 * 200 / 40) = 150.
		{"crossing in a later union segment", k(0, 40), k(0, 50, 100, 50, 300, 10), 150, 100, 300, true},
		// g(0) = 5-10 = -5, g(10) = 10-10 = 0: x1 = 10, r* = 0 + trunc(5 * 10 / 5) = 10.
		{"g is 0 at a knot", k(0, 5, 10, 10), k(0, 10, 10, 10), 10, 0, 10, true},
		{"birth never below death", k(0, 100, 10, 100), k(0, 50, 10, 50), 0, 0, 0, false},
		{"birth always below death", k(0, 10, 10, 10), k(0, 50, 10, 50), 0, 0, 0, false},
		// g goes from +20 to -20: a crossing from positive to negative only.
		{"positive to negative only", k(0, 70, 10, 30), k(0, 50, 10, 50), 0, 0, 0, false},
		{"empty schedule", nil, defDeath, 0, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, lo, hi, _, ok := population.Replacement(tt.birth, tt.death)
			if ok != tt.wantOK || got != tt.want || lo != tt.lo || hi != tt.hi {
				t.Errorf("Replacement(%v, %v) = %d, [%d, %d], %v, want %d, [%d, %d], %v",
					tt.birth, tt.death, got, lo, hi, ok, tt.want, tt.lo, tt.hi, tt.wantOK)
			}
		})
	}
}

// TestGainWarn verifies the condition of population_gain on the two worked
// examples and at the boundary: equality does not fire, one more does.
//
// Requirement: FR-POP-030
func TestGainWarn(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                                  string
		rstar, dRate, dRatio, window, perYear int64
		want                                  bool
	}{
		// 12 000 * 400 * 30 = 144 000 000 against 10 000 * 8 000 * 365 = 29 200 000 000 (12.1).
		{"agrarian realm", 12_000, 400, 8_000, 30, 365, false},
		// 12 000 * 800 * 30 = 288 000 000 against 10 000 * 12 000 * 640 = 76 800 000 000 (12.2).
		{"habitat", 12_000, 800, 12_000, 30, 640, false},
		// 10 * 10 * 10 = 1 000 against 10 000 * 1 * 1 = 10 000 is false; with window 100 it is 10 000: equal.
		{"equality is false", 10, 10, 1, 100, 1, false},
		{"one more is true", 10, 10, 1, 101, 1, true},
		{"left overflows", 1 << 62, 1 << 62, 1, 1 << 62, 1, true},
		{"right overflows", 1, 1, 1 << 62, 1, 1 << 62, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := population.GainWarn(tt.rstar, tt.dRate, tt.dRatio, tt.window, tt.perYear); got != tt.want {
				t.Errorf("GainWarn(%d, %d, %d, %d, %d) = %v, want %v", tt.rstar, tt.dRate, tt.dRatio, tt.window, tt.perYear, got, tt.want)
			}
		})
	}
}

// TestGain verifies the snapshot gain, trunc(r* * dRate / dRatio).
//
// Requirement: FR-POP-034
func TestGain(t *testing.T) {
	t.Parallel()
	// 12 000 * 400 / 8 000 = 600 basis points a year (k = 0.06 per year, 5.4).
	if got, ok := population.Gain(12_000, 400, 8_000); !ok || got != 600 {
		t.Errorf("Gain(12000, 400, 8000) = %d, %v, want 600, true", got, ok)
	}
	// 7 * 1 / 2 = 3.5 truncates to 3.
	if got, ok := population.Gain(7, 1, 2); !ok || got != 3 {
		t.Errorf("Gain(7, 1, 2) = %d, %v, want 3, true", got, ok)
	}
	if _, ok := population.Gain(1, 1, 0); ok {
		t.Errorf("Gain(1, 1, 0) ok, want not ok")
	}
	if _, ok := population.Gain(1<<62, 1<<62, 1); ok {
		t.Errorf("Gain(2^62, 2^62, 1) ok, want overflow")
	}
}

// TestMeans verifies the household-weighted means of the snapshot over the
// agents with a ratio, each truncated toward zero.
//
// Requirement: FR-POP-034
func TestMeans(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		w, b, d             []int64
		present             []bool
		wantB, wantD, wantN int64
		wantOK              bool
	}{
		// (40 * 250 + 10 * 100) / 50 = 220; (40 * 150 + 10 * 450) / 50 = 210; (40 * 100 + 10 * -350) / 50 = 10.
		{"two agents", []int64{40, 10}, []int64{250, 100}, []int64{150, 450}, []bool{true, true}, 220, 210, 10, true},
		{"absent ratio is left out", []int64{40, 10}, []int64{250, 100}, []int64{150, 450}, []bool{true, false}, 250, 150, 100, true},
		// b = (1*1 + 2*0)/3 = 0.33 -> 0; d = (1*2 + 2*1)/3 = 1.33 -> 1; n = (1 - 4)/3 = -1 exactly.
		{"truncation", []int64{1, 2}, []int64{1, 0}, []int64{2, 1}, []bool{true, true}, 0, 1, -1, true},
		{"nobody present", []int64{5}, []int64{100}, []int64{50}, []bool{false}, 0, 0, 0, true},
		{"length mismatch", []int64{5, 5}, []int64{1}, []int64{1, 1}, []bool{true, true}, 0, 0, 0, false},
		{"overflow", []int64{1 << 62}, []int64{4}, []int64{1}, []bool{true}, 0, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b, d, n, ok := population.Means(tt.w, tt.b, tt.d, tt.present)
			if ok != tt.wantOK || b != tt.wantB || d != tt.wantD || n != tt.wantN {
				t.Errorf("Means(%v, %v, %v, %v) = %d, %d, %d, %v, want %d, %d, %d, %v",
					tt.w, tt.b, tt.d, tt.present, b, d, n, ok, tt.wantB, tt.wantD, tt.wantN, tt.wantOK)
			}
		})
	}
}
