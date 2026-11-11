package economysim_test

import (
	"fmt"
	"testing"
)

// s41Ticks is the length of a run of S41.
const s41Ticks = 400

// TestScenarioS41_ResourcePopulationLoop runs scenario S41 (RES-S1, owed to this plan
// by plan 9 D398) on pack one with overlay/resources-binding.toml (the fixed factor
// at 1.1 times the draw) and overlay/population.toml, 400 ticks, seeds 1 to
// scenarioSeedCount(), the audit on. It asserts the exact checks: conservation
// (the identity of an endowment is audited every tick), no negative balance or
// stock, the counts of PopulationChanged add up, and that the households drew from
// the endowment. The loop of households, draw and yield
// is reported: the households at the end, the units drawn and refused, the stock
// of the first endowment as a share of its capacity and the mean ratio; the
// comparison with the plan of endowment.Plan and the bands are STY's.
//
// Requirement: FR-POP-022, FR-POP-037, FR-RES-024, FR-REF-023
//
// Guard: a household removed or added without its draw following breaks the
// identity of the endowment the audit checks.
func TestScenarioS41_ResourcePopulationLoop(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S41", "one+resources-binding+population"
	cfg := loadValidPack(t, packOnePopResources...)
	rep := newPackScenReport("one", fmt.Sprintf("S41 %s seeds 1-%d, %d ticks", pack, scenarioSeedCount(), s41Ticks))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				st := runPopulation(t, cfg, seed, s41Ticks, nil)
				reportPop(rep, seed, "loop", st)
				rep.num(seed, "units drawn", st.Drawn)
				rep.num(seed, "units refused by the ration", st.Short)
				rep.num(seed, "stock of the first endowment at the end, bp of capacity", st.StockShareBP)
				checkExact(t, rep, scenario, pack, seed, "counts of PopulationChanged add up", st.Bad == "", "first bad event %s", st.Bad)
				checkExact(t, rep, scenario, pack, seed, "the loop drew from the endowment", st.Drawn > 0, "%d units drawn", st.Drawn)
			})
		}
	})
	rep.flush(t)
}
