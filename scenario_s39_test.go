package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s39Ticks is the length of a run of S39.
const s39Ticks = 400

// TestScenarioS39_ClosedRegionEmptyPool runs scenario S39 (POP-S3, plan 2-11 D437)
// on pack one with overlay/resources-binding.toml and overlay/population.toml: one
// region, so closed, whose pool starts empty and fills only from the estates of the
// households that die. The birth stake is 10 000, 5 000 and 0 basis points of the
// role's start money, 400 ticks, seeds 1 to scenarioSeedCount(), the audit on. It
// asserts the exact checks: conservation, no negative balance or stock, the
// counts of PopulationChanged add up, and that with a stake of 0 no birth is left
// unfunded (a household that brings nothing needs nothing from the pool,
// FR-POP-017), and that a fourth leg, in which every role's start money is 100 000
// times larger, so that a stake of 10 000 asks the pool for more than it ever
// holds, leaves every birth unfunded and the population without a birth. The births, the births unfunded and the pool at the end are
// reported for each stake; the stop of the population against M / stake is read
// from them.
//
// Requirement: FR-POP-017, FR-POP-022
//
// Guard: a birth funded from nothing breaks the audit; a stake of 0 that still
// asked the pool leaves births unfunded.
func TestScenarioS39_ClosedRegionEmptyPool(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S39", "one+resources-binding+population"
	rep := newPackScenReport("one", fmt.Sprintf("S39 %s seeds 1-%d, %d ticks", pack, scenarioSeedCount(), s39Ticks))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			// The last leg multiplies every role's start money by 100 000, so that a
			// stake of 10 000 asks the pool for more than it ever holds.
			for _, leg := range []struct{ stake, money int64 }{{10_000, 1}, {5_000, 1}, {0, 1}, {10_000, 100_000}} {
				stake := leg.stake
				label := fmt.Sprintf("stake %d money x%d", stake, leg.money)
				t.Run(fmt.Sprintf("seed %d %s", seed, label), func(t *testing.T) {
					t.Parallel()
					cfg := loadValidPack(t, packOnePopResources...)
					mod := *cfg.Modules.Population
					mod.BirthStake = stake
					cfg.Modules.Population = &mod
					for i := range cfg.Roles {
						cfg.Roles[i].StartMoney *= economysim.Money(leg.money)
					}
					st := runPopulation(t, cfg, seed, s39Ticks, nil)
					reportPop(rep, seed, label, st)
					rep.num(seed, label+": births unfunded", st.Unfunded)
					rep.num(seed, label+": pool at the end", st.Pool)
					checkExact(t, rep, scenario, pack, seed, label+": counts of PopulationChanged add up", st.Bad == "", "first bad event %s", st.Bad)
					if leg.money > 1 {
						checkExact(t, rep, scenario, pack, seed, "births unfunded when the pool cannot fund the stake", st.Unfunded > 0 && st.Births == 0, "%d unfunded, %d born", st.Unfunded, st.Births)
					}
					if stake == 0 {
						checkExact(t, rep, scenario, pack, seed, "no birth unfunded at a stake of 0", st.Unfunded == 0, "%d births unfunded", st.Unfunded)
					}
				})
			}
		}
	})
	rep.flush(t)
}
