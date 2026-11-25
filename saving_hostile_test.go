package economysim_test

import (
	"math"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestSavingWarnings_HostileConfigsDoNotPanic verifies that Warnings, which is safe on a
// configuration that does not validate, returns for a saving configuration with values
// at the edge of int64, no household, no group and an unpriced commodity.
//
// Requirement: FR-SAV-035
//
// Guard: savingConfig raises no warning, so a panic or a spurious result is the case's.
func TestSavingWarnings_HostileConfigsDoNotPanic(t *testing.T) {
	t.Parallel()
	cases := map[string]func(c *economysim.Config){
		"hurdle at min int64":  func(c *economysim.Config) { roleOf(c, "grower").Saver.HurdleBP = math.MinInt64 },
		"hurdle at max int64":  func(c *economysim.Config) { roleOf(c, "grower").Saver.HurdleBP = math.MaxInt64 },
		"spread at min int64":  func(c *economysim.Config) { c.Modules.Saving.HurdleSpreadBP = math.MinInt64 },
		"safety at max int64":  func(c *economysim.Config) { c.Modules.Saving.SafetyTicks = math.MaxInt64 },
		"payback at max int64": func(c *economysim.Config) { roleOf(c, "grower").Saver.PaybackTicks = math.MaxInt64 },
		"payback negative":     func(c *economysim.Config) { roleOf(c, "grower").Saver.PaybackTicks = math.MinInt64 },
		"reserve ticks at max": func(c *economysim.Config) { c.World.ReserveTicks = math.MaxInt64 },
		"start money at max":   func(c *economysim.Config) { roleOf(c, "grower").StartMoney = economysim.Money(math.MaxInt64) },
		"households at max":    func(c *economysim.Config) { c.Groups[0].Households = math.MaxInt64 },
		"households at min":    func(c *economysim.Config) { c.Groups[0].Households = math.MinInt64 },
		"no groups":            func(c *economysim.Config) { c.Groups = nil },
		"no regions":           func(c *economysim.Config) { c.Regions = nil },
		"no commodities":       func(c *economysim.Config) { c.Commodities = nil },
		"worth with max numbers": func(c *economysim.Config) {
			roleOf(c, "grower").Saver.Worth = []economysim.ProjectWorth{{Class: "mill", PerTick: math.MaxInt64, Count: math.MaxInt64}}
		},
		"worth for an unknown class": func(c *economysim.Config) {
			roleOf(c, "grower").Saver.Worth = []economysim.ProjectWorth{{Class: "nope", PerTick: -1, Count: -1}}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := savingConfig()
			mutate(&c)
			_ = c.Warnings()
			_ = c.Validate()
		})
	}
}

// TestSavingWarnings_HurdleBelowBoundsSaturatesMargin verifies that a hurdle at the
// bottom of int64 does not wrap the margin of spread_below_margin to a negative value
// that hides the warning.
//
// Requirement: FR-SAV-035
func TestSavingWarnings_HurdleBelowBoundsSaturatesMargin(t *testing.T) {
	t.Parallel()
	c := savingConfig()
	roleOf(&c, "grower").Saver.HurdleBP = math.MinInt64
	want := "spread_below_margin@roles.grower.saver"
	if got := savingWarningsOf(c); !slices.Contains(got, want) {
		t.Errorf("warnings = %v, want %q", got, want)
	}
}
