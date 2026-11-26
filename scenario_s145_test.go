package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// savingS145Inputs are the staple quantities of the build recipe of the mill in S145: the overlay's
// 2, a figure whose cost a cook's surplus covers in a few ticks, and one it cannot reach.
var savingS145Inputs = []int64{2, 400, 20_000}

// TestScenarioS145_AnOutlayAboveTheSurplus reports SAV-S4 of spec-saving-and-investment.md 14.3 on
// pack one with the infrastructure and saving overlays, 200 ticks, audit on: the inputs of
// build_mill at 2, 400 and 20 000 units of the staple, so that the outlay (inputs, the upkeep of the
// job and the floor, spec 5.6.1) rises from far below a cook's surplus to far above it. The
// identities are asserted (the audit holds; no facility is built without a request); the effect
// is reported: the facilities built, the households that save, and those barred by `Unreachable`,
// which buy buffers and no wants for a project they cannot fund, and by `NoReturn`. The bars are in
// the order of spec 5.5, so a cost so large that the margin test fails first is counted `NoReturn`,
// not `Unreachable`: the `Unreachable` count does not rise with the outlay without bound.
//
// Arithmetic: reach needs Need * n <= max(0, C_n) * T (spec 5.6.5): the outlay a cook can fund in
// the record of its surplus. The pack's placeholders decide where the legs fall (N-015, N-027).
//
// Requirement: FR-SAV-019, FR-SAV-045, FR-SAV-033
//
// Guard: a facility built with no request, or a step that fails the audit across a project the
// household cannot fund, fails here.
func TestScenarioS145_AnOutlayAboveTheSurplus(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	savingScenario(t, "S145 one+saving the outlay of a build, 200 ticks", scenarioSeedCount(), func(t *testing.T, rep *scenReport, seed uint64) {
		for _, q := range savingS145Inputs {
			cfg := savingPack(t, func(c *economysim.Config) { setRecipeInput(t, c, "build_mill", q) })
			st := savingRun(t, cfg, seed, 200)
			name := fmt.Sprintf("inputs %d", q)
			rep.num(seed, name+": built", st.Built)
			rep.num(seed, name+": saving household-ticks", st.Saving)
			rep.num(seed, name+": unreachable household-ticks", st.Unreachable)
			rep.num(seed, name+": no return household-ticks", st.NoReturn)
			// Not in the report: the leg with the overlay's cost is the control; a module that planned and
			// asked for nothing would pass the check below with 0 against 0.
			if q == savingS145Inputs[0] && (st.Builds == 0 || st.Built == 0) {
				t.Fatalf("seed %d, %s: %d build requests and %d built, want both above 0 for the cheap leg", seed, name, st.Builds, st.Built)
			}
			checkExact(t, rep, "S145", "one", seed, name+": no facility is built without a build request", st.Built <= st.Builds, "%d built against %d requested", st.Built, st.Builds)
		}
	})
}

// setRecipeInput sets the quantity of the first input of the recipe id and fails the test when the
// recipe or its input is missing, so a leg never runs on the unchanged pack.
func setRecipeInput(t testing.TB, c *economysim.Config, id string, quantity int64) {
	t.Helper()
	for i := range c.Recipes {
		if c.Recipes[i].ID == id && len(c.Recipes[i].Inputs) > 0 {
			c.Recipes[i].Inputs[0].Quantity = economysim.Quantity(quantity)
			return
		}
	}
	t.Fatalf("recipe %q with an input is not in the configuration", id)
}
