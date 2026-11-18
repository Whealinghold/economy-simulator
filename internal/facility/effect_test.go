package facility_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/facility"
)

// TestEff_Rounding verifies the effect fraction of a gain: floor(boost *
// condition / durability), 0 at condition 0, the whole boost at full
// condition, and ok false outside the range.
//
// Requirement: FR-CAP-006, FR-CAP-046
func TestEff_Rounding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                   string
		boost, cond, dur, want int64
		ok                     bool
	}{
		// 6000 * 1999 / 2000 = 5997.0 (worked.py part 1, a road built at 1999).
		{"one short of full", 6000, 1999, 2000, 5997, true},
		{"full", 6000, 2000, 2000, 6000, true},
		{"half", 6000, 1000, 2000, 3000, true},
		{"zero condition", 6000, 0, 2000, 0, true},
		// 1 * 1 / 2 = 0.5 floors to 0: a gain that rounded to nothing still stands.
		{"rounds to nothing", 1, 1, 2, 0, true},
		{"zero boost", 0, 5, 10, 0, true},
		{"durability zero", 1, 0, 0, 0, false},
		{"condition above durability", 1, 3, 2, 0, false},
		{"negative condition", 1, -1, 2, 0, false},
		{"negative boost", -1, 1, 2, 0, false},
		// The product 10^5 * 10^6 = 10^11 and a 128-bit one both fit.
		{"bounds", 100_000, 999_999, 1_000_000, 99_999, true},
		{"128 bit product", math.MaxInt64, math.MaxInt64, math.MaxInt64, math.MaxInt64, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := facility.Eff(tt.boost, tt.cond, tt.dur)
			if got != tt.want || ok != tt.ok {
				t.Errorf("Eff(%d, %d, %d) = %d, %v, want %d, %v", tt.boost, tt.cond, tt.dur, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestCap_Rounding verifies the capacity of a site: ceil(capacity *
// condition / durability), at least 1 above condition 0 and 0 at 0.
//
// Requirement: FR-CAP-006
func TestCap_Rounding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                      string
		capacity, cond, dur, want int64
		ok                        bool
	}{
		// 6 * 1 / 1000 = 0.006 rounds up to 1: an almost worn mill is still a mill.
		{"almost worn", 6, 1, 1000, 1, true},
		{"ruined", 6, 0, 1000, 0, true},
		{"full", 6, 1000, 1000, 6, true},
		// 6 * 500 / 1000 = 3 exactly: no rounding.
		{"half exact", 6, 500, 1000, 3, true},
		// 6 * 501 / 1000 = 3.006 rounds up to 4.
		{"just above half", 6, 501, 1000, 4, true},
		{"durability zero", 6, 0, 0, 0, false},
		{"condition above durability", 6, 1001, 1000, 0, false},
		{"negative capacity", -1, 1, 2, 0, false},
		{"128 bit product", math.MaxInt64, math.MaxInt64 - 1, math.MaxInt64, math.MaxInt64 - 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := facility.Cap(tt.capacity, tt.cond, tt.dur)
			if got != tt.want || ok != tt.ok {
				t.Errorf("Cap(%d, %d, %d) = %d, %v, want %d, %v", tt.capacity, tt.cond, tt.dur, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestGain_Capped verifies that the gain is a capped sum: ten facilities do
// not multiply, the cap holds, a negative term counts as 0 and nothing
// overflows.
//
// Requirement: FR-CAP-021, FR-CAP-024
func TestGain_Capped(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		effs []int64
		max  int64
		want int64
	}{
		{"empty", nil, 200_000, 0},
		{"sum below cap", []int64{3000, 2000}, 200_000, 5000},
		// 30 * 10 000 = 300 000 is capped at 200 000.
		{"ten roads do not multiply", slicesOf(30, 10_000), 200_000, 200_000},
		{"exactly the cap", []int64{100, 100}, 200, 200},
		{"negative term ignored", []int64{-5, 7}, 100, 7},
		{"negative cap is zero", []int64{5}, -1, 0},
		{"no overflow", []int64{math.MaxInt64, math.MaxInt64}, math.MaxInt64, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := facility.Gain(tt.effs, tt.max); got != tt.want {
				t.Errorf("Gain(%v, %d) = %d, want %d", tt.effs, tt.max, got, tt.want)
			}
		})
	}
}

// slicesOf returns n copies of v.
func slicesOf(n int, v int64) []int64 {
	s := make([]int64, n)
	for i := range s {
		s[i] = v
	}
	return s
}

// TestLayer_Factor verifies the transit layer with the rows of
// spec-infrastructure.md 18.2 (a 60-unit edge at speed 20, BoostBP 6000,
// Durability 2000): the slot 0 stays 0, the factor floors, and the cap acts.
//
// Requirement: FR-CAP-021
func TestLayer_Factor(t *testing.T) {
	t.Parallel()
	const maxFactor = 100_000
	tests := []struct {
		name             string
		slot, gain, want int64
		ok               bool
	}{
		// 10 000 * (10 000 + 6000) / 10 000 = 16 000: the travel time 3 falls to 2.
		{"one road at full", 10_000, 6000, 16_000, true},
		// 5000 * 16 000 / 10 000 = 8000.
		{"damaged slot", 5000, 6000, 8000, true},
		{"closed stays closed", 0, 6000, 0, true},
		{"no gain", 10_000, 0, 10_000, true},
		// 90 000 * 210 000 / 10 000 = 1 890 000 is capped at 100 000.
		{"capped", 90_000, 200_000, maxFactor, true},
		// 3 * 10 001 / 10 000 = 3.0003 floors to 3.
		{"floors", 3, 1, 3, true},
		{"negative slot", -1, 0, 0, false},
		{"negative gain", 1, -1, 0, false},
		{"beyond int64 is capped", math.MaxInt64, math.MaxInt64 - 10_000, maxFactor, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := facility.Layer(tt.slot, tt.gain, maxFactor)
			if got != tt.want || ok != tt.ok {
				t.Errorf("Layer(%d, %d, %d) = %d, %v, want %d, %v", tt.slot, tt.gain, int64(maxFactor), got, ok, tt.want, tt.ok)
			}
		})
	}
	if _, ok := facility.Layer(1, 1, -1); ok {
		t.Errorf("Layer(1, 1, -1) ok = true, want false for a negative cap")
	}
}
