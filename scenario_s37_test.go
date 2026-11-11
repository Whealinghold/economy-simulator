package economysim_test

import (
	"fmt"
	"testing"
)

// s37Ticks is the length of a run of S37, a year being 365 ticks of a day.
const s37Ticks = 400

// TestScenarioS37_ApproachFromBothSides runs scenario S37 (POP-S1, plan 2-11
// D437) on pack one with overlay/resources-binding.toml and overlay/population.toml,
// 400 ticks, seeds 1 to scenarioSeedCount(), the audit on: two runs that start at
// 0.6 and 1.6 of the pack's households (the groups' counts scaled; world.scale is
// households per agent, not a count). It asserts the exact checks: conservation
// (the audit fails a Step), no negative balance or stock, the counts of every
// PopulationChanged add up (FR-POP-022, 037) and the households are never negative.
// The least households, the end count, the end mean ratio and the crossings of the
// start are reported; the bands are STY's.
//
// The solver does not report N* (spec-population.md 18 item 3, N-109), and the 2 000-tick
// pre-run the plan names reaches 0 households on this pack with renewal 0 (the
// cold-run collapse of the Stage 1 economy, see the recorded outcomes), so N*
// is not defined here and the two starts are scaled from the pack's own 600.
//
// Requirement: FR-POP-022, FR-POP-037, FR-POP-006, FR-REF-023
//
// Guard: a birth or death the executor did not carry out, or one that created a
// unit from nothing, fails the identities or the audit.
func TestScenarioS37_ApproachFromBothSides(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S37", "one+resources-binding+population"
	base := loadValidPack(t, packOnePopResources...)
	rep := newPackScenReport("one", fmt.Sprintf("S37 %s seeds 1-%d, %d ticks", pack, scenarioSeedCount(), s37Ticks))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			for _, leg := range []struct {
				name     string
				num, den int64
			}{{"0.6", 6, 10}, {"1.6", 16, 10}} {
				t.Run(fmt.Sprintf("seed %d start %s", seed, leg.name), func(t *testing.T) {
					t.Parallel()
					st := runPopulation(t, withHouseholds(base, leg.num, leg.den), seed, s37Ticks, nil)
					reportPop(rep, seed, "start "+leg.name, st)
					checkExact(t, rep, scenario, pack, seed, "counts of PopulationChanged add up", st.Bad == "", "first bad event %s", st.Bad)
					checkExact(t, rep, scenario, pack, seed, "households not negative", st.End >= 0 && st.Min >= 0, "end %d, least %d", st.End, st.Min)
				})
			}
		}
	})
	rep.flush(t)
}
