package skill_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/skill"
)

// TestOutput_NetPerTick verifies the net value of a trainee's eligible output
// per tick of spec-production.md 8.3: value * T / 10 000 less the input cost,
// over the duration, truncated toward zero, with overflow and bad input reported.
//
// Requirement: FR-SKL-016
func TestOutput_NetPerTick(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                          string
		value, inputs, yieldBP, ticks int64
		want                          int64
		wantOK                        bool
	}{
		// 1 000 * 5 000 / 10 000 = 500; 500 - 200 = 300; one tick.
		{"worked example", 1000, 200, 5000, 1, 300, true},
		// (500 - 200) / 2 = 150.
		{"over a two-tick recipe", 1000, 200, 5000, 2, 150, true},
		// 500 - 800 = -300; -300 / 2 = -150 (a loss is a negative output).
		{"a loss", 1000, 800, 5000, 2, -150, true},
		// -301 / 2 truncates toward zero: -150.
		{"the quotient truncates toward zero", 1000, 801, 5000, 2, -150, true},
		{"zero duration", 1000, 200, 5000, 0, 0, false},
		{"negative value", -1, 0, 5000, 1, 0, false},
		{"negative inputs", 1, -1, 5000, 1, 0, false},
		{"negative yield", 1, 0, -1, 1, 0, false},
		{"product overflow", math.MaxInt64, 0, math.MaxInt64, 1, 0, false},
		// 0 - MaxInt64 fits int64 and is negative.
		{"largest input cost", 0, math.MaxInt64, 5000, 1, -math.MaxInt64, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := skill.Output(tt.value, tt.inputs, tt.yieldBP, tt.ticks)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Output(%d, %d, %d, %d) = %d, %v, want %d, %v", tt.value, tt.inputs, tt.yieldBP, tt.ticks, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestSurplus_AboveUpkeepFloored verifies S_u of spec-skill-barriers.md 4.5: the
// income per tick above the upkeep, from the window ratio, floored at 0 and 0
// while the window has no ratio.
//
// Requirement: FR-SKL-018
func TestSurplus_AboveUpkeepFloored(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		ratio   int64
		present bool
		upkeep  int64
		want    int64
		wantOK  bool
	}{
		// (15 000 - 10 000) * 1 000 / 10 000 = 500, the surplus of 4.8.
		{"worked example of 4.8", 15000, true, 1000, 500, true},
		{"ratio at 10 000", 10000, true, 1000, 0, true},
		{"ratio below 10 000", 4000, true, 1000, 0, true},
		{"no ratio yet", 15000, false, 1000, 0, true},
		{"no upkeep", 15000, true, 0, 0, true},
		{"overflow", math.MaxInt64, true, math.MaxInt64, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := skill.Surplus(tt.ratio, tt.present, tt.upkeep)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Surplus(%d, %v, %d) = %d, %v, want %d, %v", tt.ratio, tt.present, tt.upkeep, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
