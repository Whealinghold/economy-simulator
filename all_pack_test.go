package economysim_test

import (
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/strategy"
)

// allLayers is the all-on pack of plan 2-24 D683: the warm `five` (base, lines,
// five, warm-five), its edges, and one layer for each version 1 module in the order of the
// `Modules` pointers (spec-config.md 3.1), each module after what it needs (the
// merchants need transport and information, the migration needs information). A
// layer under all/ is a retarget of a module overlay to five; the others are the
// overlays of the module plans as they are. A module added in Stage 3 appends a
// layer here and moves testdata/golden/all/ once (testing.md 7.2).
var allLayers = []string{
	"base.toml", "lines.toml", "five.toml", "warm-five.toml",
	"overlay/trade-five.toml",
	"all/demand.toml",
	"overlay/faucet.toml",
	"overlay/population.toml",
	"overlay/transport-five.toml",
	"overlay/resources-five.toml",
	"overlay/information-five.toml",
	"overlay/external.toml",
	"all/subsistence.toml",
	"overlay/migration.toml",
	"all/merchants.toml",
	"overlay/estates.toml",
}

// allColdLayers is the all-on pack without the warm start, the set the solver
// reads for the expected file.
func allColdLayers() []string {
	return slices.DeleteFunc(slices.Clone(allLayers), func(l string) bool { return l == "warm-five.toml" })
}

// allOptions are the options the all-on world is built with besides the seed: the
// registry of the strategies the pack names (the merchants), and nothing else.
func allOptions() []economysim.Option { return strategy.Options() }
