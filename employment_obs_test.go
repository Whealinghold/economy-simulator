package economysim_test

import (
	economysim "github.com/Whealinghold/economy-sim"
	"testing"
)

// hireLogScenario is the pack one-demand with the hire form, 60 ticks, for the log
// tests: the default strategy offers and posts, so the Debug lines of the round, the
// events of the hires and the two Warn lines of the module (when a run reaches ten) are
// written (spec-employment.md 15.2).
func hireLogScenario() scenario {
	return scenario{name: "employment hire", ticks: 60, tail: 2, cfg: func(t *testing.T) economysim.Config {
		return loadValidPack(t, packOneDemandHire...)
	}}
}
