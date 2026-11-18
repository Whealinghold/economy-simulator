package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// tenureLogScenario is pack one with the resources and tenure overlays for 200 ticks: the
// log tests read it with the other scenarios (logScenarios), so the Warns and the Debug
// lines of the module are held to the keys of the rules table and to the rule that no Info
// or Error is written in a tick. The pack charges no rent (the growers draw on their own
// entitlements, packFiveTenure is the pack that does) and a pack of five thousand households
// with a Debug logger that keeps every line would be too large for the log tests; the
// world-level tests of the Warn and the Debug line are tenure_rentstep_internal_test.go.
func tenureLogScenario() scenario {
	return scenario{name: "tenure", ticks: 200, tail: 2, cfg: func(t *testing.T) economysim.Config {
		return loadValidPack(t, append(slices.Clone(packOneResources), "overlay/tenure.toml")...)
	}}
}
