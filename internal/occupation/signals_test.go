package occupation_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/occupation"
)

// TestExpected_Cases verifies E at the worked case, its truncation toward
// zero, the two causes of no signal, the saturation by the sign of the wage
// and the second case of the computation, where upkeep * ticksPerYear leaves
// int64.
//
// Wage 1 200, 250 expected ticks, upkeep 1 000, 365 ticks: 1 200 * 250 * 10 000
// / (1 000 * 365) = 3 000 000 000 / 365 000 = 8 219.17, so 8 219. A wage of
// -500 gives -1 250 000 000 / 365 000 = -3 424.65, truncated to -3 424. Wage
// 1e18 with 1e5 ticks over upkeep 1 and 1 tick is 1e27, beyond int64. With
// upkeep 1e14 and 1e5 ticks the product 1e19 does not fit: wage 1e18 gives
// 1e18 * 1e9 / 1e19 = 1e8, and wage 1e18 - 1 gives 99 999 999.99999, so
// 99 999 999 whichever way it is computed (1e13 - 1 over 1e5 in two steps).
// Expected work of 9e14 with wage MaxInt64 over upkeep 1e14 leaves int64 in the
// first step of case 2, which saturates by the sign of the wage.
//
// Requirement: FR-SPN-017
func TestExpected_Cases(t *testing.T) {
	t.Parallel()
	const maxTicks = 100_000
	tests := []struct {
		name                                 string
		wage, expected, upkeep, ticksPerYear int64
		want                                 int64
		wantHas                              bool
	}{
		{"worked case", 1200, 250, 1000, 365, 8219, true},
		{"zero wage", 0, 250, 1000, 365, 0, true},
		{"negative wage truncates toward zero", -500, 250, 1000, 365, -3424, true},
		{"no upkeep", 1200, 250, 0, 365, 0, false},
		{"negative upkeep", 1200, 250, -1, 365, 0, false},
		{"no ticks per year", 1200, 250, 1000, 0, 0, false},
		{"zero expected ticks", 1200, 0, 1000, 365, 0, true},
		{"positive saturation", 1e18, maxTicks, 1, 1, math.MaxInt64, true},
		{"negative saturation", -1e18, maxTicks, 1, 1, math.MinInt64, true},
		{"product of upkeep and year beyond int64", 1e18, maxTicks, 1e14, maxTicks, 1e8, true},
		{"second case truncates once", 1e18 - 1, maxTicks, 1e14, maxTicks, 99_999_999, true},
		{"second case negative", -(1e18 - 1), maxTicks, 1e14, maxTicks, -99_999_999, true},
		{"second case first step overflows", math.MaxInt64, 900_000_000_000_000, 1e14, maxTicks, math.MaxInt64, true},
		{"second case first step overflows negative", math.MinInt64 + 1, 900_000_000_000_000, 1e14, maxTicks, math.MinInt64, true},
		{"expected ticks times basis overflows", 5, math.MaxInt64 / 2, 1000, 365, math.MaxInt64, true},
		{"expected ticks times basis overflows, zero wage", 0, math.MaxInt64 / 2, 1000, 365, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, has := occupation.Expected(tt.wage, tt.expected, tt.upkeep, tt.ticksPerYear)
			if got != tt.want || has != tt.wantHas {
				t.Errorf("Expected(%d, %d, %d, %d) = %d, %v, want %d, %v",
					tt.wage, tt.expected, tt.upkeep, tt.ticksPerYear, got, has, tt.want, tt.wantHas)
			}
		})
	}
}

// TestViability_Cases verifies V at the worked case, a zero and a negative wage,
// the absence of a signal and the saturation by the sign of the wage.
//
// 1 200 * 10 000 / 1 000 = 12 000. -1 200 * 10 000 / 1 000 = -12 000. 7 * 10 000
// / 3 = 23 333.3, truncated to 23 333. MaxInt64 * 10 000 is beyond int64 over an
// upkeep of 1.
//
// Requirement: FR-SPN-017
func TestViability_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		wage, upkeep int64
		want         int64
		wantHas      bool
	}{
		{"worked case", 1200, 1000, 12000, true},
		{"zero wage", 0, 1000, 0, true},
		{"negative wage", -1200, 1000, -12000, true},
		{"truncates", 7, 3, 23333, true},
		{"negative truncates toward zero", -7, 3, -23333, true},
		{"no upkeep", 1200, 0, 0, false},
		{"negative upkeep", 1200, -5, 0, false},
		{"positive saturation", math.MaxInt64, 1, math.MaxInt64, true},
		{"negative saturation", math.MinInt64, 1, math.MinInt64, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, has := occupation.Viability(tt.wage, tt.upkeep)
			if got != tt.want || has != tt.wantHas {
				t.Errorf("Viability(%d, %d) = %d, %v, want %d, %v", tt.wage, tt.upkeep, got, has, tt.want, tt.wantHas)
			}
		})
	}
}

// TestBeats_Cases verifies the hysteresis comparison at the edge of the worked
// case, with a negative or zero own signal, with no band, and where the ceiling
// leaves int64 (decided by its sign).
//
// ceil(8 219 * 11 000 / 10 000) = ceil(9 040.9) = 9 041, so 9 040 fails and
// 9 041 passes. For an own signal of -1 000 the threshold is ceil(-11 000 000 /
// 10 000) = -1 100. For 0 it is 0. With a band of 0 the threshold is the own
// signal. MaxInt64 * 11 000 / 10 000 is beyond int64: no destination reaches
// it. MinInt64 * 11 000 / 10 000 is below int64: every destination does.
//
// Requirement: FR-SPN-019
func TestBeats_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		eD, eO, hyst int64
		want         bool
	}{
		{"one below the edge", 9040, 8219, 1000, false},
		{"at the edge", 9041, 8219, 1000, true},
		{"above the edge", 12000, 8219, 1000, true},
		{"negative own signal, at the threshold", -1100, -1000, 1000, true},
		{"negative own signal, one below", -1101, -1000, 1000, false},
		{"zero own signal, at zero", 0, 0, 1000, true},
		{"zero own signal, below", -1, 0, 1000, false},
		{"no band, equal", 500, 500, 0, true},
		{"no band, one below", 499, 500, 0, false},
		{"threshold beyond int64, positive", math.MaxInt64, math.MaxInt64, 1000, false},
		{"threshold beyond int64, negative", math.MinInt64, math.MinInt64, 1000, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := occupation.Beats(tt.eD, tt.eO, tt.hyst); got != tt.want {
				t.Errorf("Beats(%d, %d, %d) = %v, want %v", tt.eD, tt.eO, tt.hyst, got, tt.want)
			}
		})
	}
}

// TestCapitalOK_Cases verifies the capital test as the exact inequality diff *
// ut >= price * CapitalReturn: a price of 0 passes first, equal sides pass and
// one more fails, a negative difference fails, an unusable ut is denied, and
// 128-bit products are compared exactly.
//
// diff = 500, ut = 1 000 * 365 = 365 000, left = 182 500 000. Price 1 825 000 at
// 100 basis points is 182 500 000 (equal, passes); 1 825 001 gives 182 500 100
// (fails). Upkeep 1e14 with 1e5 ticks has ut 1e19, beyond int64: denied. The
// full-range difference (MaxInt64 - MinInt64 = 2^64 - 1) times ut 2^62 is about
// 2^126, above MaxInt64 * MaxInt64 = 2^126 - 2^64 + 1 (passes); a difference of
// 1 against MaxInt64 * 2 = 2^64 - 2 fails.
//
// Requirement: FR-SPN-058
func TestCapitalOK_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                                           string
		eD, eO, price, capReturn, upkeep, ticksPerYear int64
		want                                           bool
	}{
		{"price 0 passes before anything", -5, 10, 0, 100, 0, 0, true},
		{"equal sides", 600, 100, 1_825_000, 100, 1000, 365, true},
		{"one more on the right", 600, 100, 1_825_001, 100, 1000, 365, false},
		{"negative difference", 99, 100, 1, 0, 1000, 365, false},
		{"zero difference, zero return", 100, 100, 5, 0, 1000, 365, true},
		{"zero difference, positive return", 100, 100, 5, 1, 1000, 365, false},
		{"ut beyond int64 is denied", 1 << 40, 0, 1, 1, 1e14, 100_000, false},
		{"no upkeep", 1000, 0, 1, 1, 0, 365, false},
		{"no ticks", 1000, 0, 1, 1, 1000, 0, false},
		{"negative price", 1000, 0, -1, 1, 1000, 365, false},
		{"negative return", 1000, 0, 1, -1, 1000, 365, false},
		{"full range difference, high word decides", math.MaxInt64, math.MinInt64, math.MaxInt64, math.MaxInt64, 1 << 31, 1 << 31, true},
		{"high word decides against", 1, 0, math.MaxInt64, 2, 1, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := occupation.CapitalOK(tt.eD, tt.eO, tt.price, tt.capReturn, tt.upkeep, tt.ticksPerYear); got != tt.want {
				t.Errorf("CapitalOK(%d, %d, %d, %d, %d, %d) = %v, want %v",
					tt.eD, tt.eO, tt.price, tt.capReturn, tt.upkeep, tt.ticksPerYear, got, tt.want)
			}
		})
	}
}
