package employment_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/employment"
)

// TestCircuit_ClosedForm_MatchesModelCell verifies the closed form of
// spec-employment.md 11.2 against the sixteen cells of the model
// (models/emp/circuit.py: 10 employers of upkeep 3, 6 units an execution, 100
// places, SpendHalfTicks 10). The expected values are the bound
// h* = min(100, 10 * 3 * (10 000 + S * 10) / ((10 000 - phi) * 6)) in
// thousandths of a household, rounded down.
//
// Requirement: FR-EMP-062
func TestCircuit_ClosedForm_MatchesModelCell(t *testing.T) {
	t.Parallel()
	// Each row is phi; each column S in 0, 2 500, 5 000, 9 000. The numerator
	// is 30 * (10 000 + 10 S) * 1 000 and the denominator (10 000 - phi) * 6:
	//   phi 3 000: den 42 000: 300 000 000 -> 7 142; 1 050 000 000 -> 25 000;
	//              1 800 000 000 -> 42 857; 3 000 000 000 -> 71 428
	//   phi 5 000: den 30 000: 10 000; 35 000; 60 000; 100 000 (the cap, 100 places)
	//   phi 7 500: den 15 000: 20 000; 70 000; 120 000 -> 100 000 (cap); 200 000 -> 100 000
	//   phi 9 000: den  6 000: 50 000; 175 000 -> 100 000; 300 000 -> 100 000; 500 000 -> 100 000
	phis := []int64{3000, 5000, 7500, 9000}
	shares := []int64{0, 2500, 5000, 9000}
	want := [4][4]int64{
		{7142, 25000, 42857, 71428},
		{10000, 35000, 60000, 100000},
		{20000, 70000, 100000, 100000},
		{50000, 100000, 100000, 100000},
	}
	for i, phi := range phis {
		for j, s := range shares {
			c := employment.Circuit{Employers: 10, Upkeep: 3, Units: 6, PhiBP: phi, SpendMaxShareBP: s, SpendHalfTicks: 10, Places: 100}
			got, ok := employment.HiredMilli(c)
			if !ok || got != want[i][j] {
				t.Errorf("HiredMilli(phi %d, S %d) = %d, %v, want %d, true", phi, s, got, ok, want[i][j])
			}
		}
	}
}

// TestCircuit_Invalid verifies that an input out of range is reported.
//
// Requirement: FR-EMP-062
func TestCircuit_Invalid(t *testing.T) {
	t.Parallel()
	good := employment.Circuit{Employers: 10, Upkeep: 3, Units: 6, PhiBP: 5000, Places: 100}
	tests := map[string]func(c *employment.Circuit){
		"phi at the whole": func(c *employment.Circuit) { c.PhiBP = 10000 },
		"negative phi":     func(c *employment.Circuit) { c.PhiBP = -1 },
		"zero units":       func(c *employment.Circuit) { c.Units = 0 },
		"negative places":  func(c *employment.Circuit) { c.Places = -1 },
		"negative upkeep":  func(c *employment.Circuit) { c.Upkeep = -1 },
		"cap product big":  func(c *employment.Circuit) { c.SpendMaxShareBP = math.MaxInt64; c.SpendHalfTicks = 2 },
		"employers big":    func(c *employment.Circuit) { c.Employers = math.MaxInt64; c.Upkeep = 2 },
		"places milli big": func(c *employment.Circuit) { c.Places = math.MaxInt64 },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := good
			change(&c)
			if got, ok := employment.HiredMilli(c); ok {
				t.Errorf("HiredMilli(%+v) = %d, true, want false", c, got)
			}
		})
	}
}
