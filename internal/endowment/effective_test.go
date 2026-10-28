package endowment_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/curve"
	"github.com/Whealinghold/economy-sim/internal/endowment"
)

// TestEffective_CapacityAndScale verifies the effective values of
// spec-resources.md 3.4: the capacity times the capacity scale times
// (10 000 + B) over 10^8, truncated and cut at the maximum, and the regeneration
// scale times its shock scale over 10 000, truncated.
//
// Requirement: FR-RES-005, FR-RES-006
func TestEffective_CapacityAndScale(t *testing.T) {
	t.Parallel()
	const maxCap = 1_000_000_000_000_000
	capTests := []struct {
		name                     string
		row, scale, facility, mx int64
		want                     int64
		wantOK                   bool
	}{
		{"unscaled", 1000, 10_000, 0, maxCap, 1000, true},
		{"half scale", 1000, 5000, 0, maxCap, 500, true},
		// 1 000 * 10 000 * 12 000 / 10^8 = 1 200: B = 2 000 raises it by a fifth.
		{"facility layer", 1000, 10_000, 2000, maxCap, 1200, true},
		// 1 001 * 3 333 / 10^4 = 333.6, truncated to 333.
		{"truncates", 1001, 3333, 0, maxCap, 333, true},
		{"cut at the maximum", maxCap, 100_000, 0, maxCap, maxCap, true},
		// 10^15 * 10^5 * 10^5 / 10^8 = 10^17 does not fit the maximum 10^15 and is cut to it.
		{"largest inputs are cut", maxCap, 100_000, 90_000, maxCap, maxCap, true},
		{"negative row", -1, 10_000, 0, maxCap, 0, false},
		{"negative scale", 1, -1, 0, maxCap, 0, false},
		{"negative facility", 1, 1, -1, maxCap, 0, false},
		{"negative maximum", 1, 1, 0, -1, 0, false},
		{"zero row", 0, 10_000, 0, maxCap, 0, true},
		{"zero scale", 1000, 0, 0, maxCap, 0, true},
		{"zero maximum cuts everything", 1000, 10_000, 0, 0, 0, true},
		// 9.2e18 * 10^5 * 10^4 / 10^8 = 9.2e19 needs more than 64 bits: cut to the maximum.
		{"quotient beyond 64 bits", math.MaxInt64, 100_000, 0, maxCap, maxCap, true},
		{"beyond 128 bits", math.MaxInt64, math.MaxInt64, math.MaxInt64, maxCap, 0, false},
	}
	for _, tt := range capTests {
		t.Run("capacity "+tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := endowment.EffectiveCapacity(tt.row, tt.scale, tt.facility, tt.mx)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("EffectiveCapacity(%d, %d, %d, %d) = %d, %v, want %d, %v", tt.row, tt.scale, tt.facility, tt.mx, got, ok, tt.want, tt.wantOK)
			}
		})
	}
	scaleTests := []struct {
		name       string
		row, scale int64
		want       int64
		wantOK     bool
	}{
		{"neutral", 10_000, 10_000, 10_000, true},
		{"halved by the shock", 10_000, 5000, 5000, true},
		// 3 * 3 333 / 10 000 = 0.9999, truncated to 0.
		{"truncates", 3, 3333, 0, true},
		{"negative row", -1, 10_000, 0, false},
		{"negative shock", 10_000, -1, 0, false},
		{"zero row", 0, 10_000, 0, true},
		{"zero shock", 10_000, 0, 0, true},
		{"beyond int64", math.MaxInt64, math.MaxInt64, 0, false},
	}
	for _, tt := range scaleTests {
		t.Run("scale "+tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := endowment.EffectiveScale(tt.row, tt.scale)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("EffectiveScale(%d, %d) = %d, %v, want %d, %v", tt.row, tt.scale, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestAbundance_CurveShareAndMinimum verifies the abundance of a stock
// (spec-resources.md 4.7.1): the curve at the floor of the stock share, a flat
// 10 000 with no knots, and 0 whatever the curve for an endowment with no
// capacity.
//
// Requirement: FR-RES-013, FR-RES-014
func TestAbundance_CurveShareAndMinimum(t *testing.T) {
	t.Parallel()
	knots := []curve.Knot{{X: 0, Y: 2000}, {X: 2000, Y: 8000}, {X: 10_000, Y: 10_000}}
	tests := []struct {
		name            string
		knots           []curve.Knot
		stock, capacity int64
		want            int64
		wantOK          bool
	}{
		// share = floor(1 000 * 10 000 / 2 000) = 5 000; 8 000 + trunc(2 000 * 3 000 / 8 000) = 8 750.
		{"interior of the curve", knots, 1000, 2000, 8750, true},
		{"empty stock reads the first knot", knots, 0, 2000, 2000, true},
		{"full stock reads the last knot", knots, 2000, 2000, 10_000, true},
		// share = floor(1 * 10 000 / 3) = 3 333; 8 000 + trunc(2 000 * 1 333 / 8 000) = 8 333.
		{"share is floored", knots, 1, 3, 8333, true},
		{"no knots is flat", nil, 5, 100, 10_000, true},
		{"no capacity is 0 whatever the curve", knots, 0, 0, 0, true},
		{"no capacity and no knots is 0", nil, 0, 0, 0, true},
		{"stock above the capacity reads the capacity", knots, 3000, 2000, 10_000, true},
		{"negative stock", knots, -1, 2000, 0, false},
		{"negative capacity", knots, 1, -1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := endowment.Abundance(tt.knots, tt.stock, tt.capacity)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Abundance(%v, %d, %d) = %d, %v, want %d, %v", tt.knots, tt.stock, tt.capacity, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
