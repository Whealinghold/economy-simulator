package economysim_test

import (
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// creditYieldShocks are the shocks of S115, applied between ticks to the growers' recipe:
// the cycle of ticks 60 to 120 loses half its output and the next one gains half more.
var creditYieldShocks = []struct {
	tick int
	bp   int64
}{{60, 5_000}, {120, 15_000}}

// TestScenarioS115_ShocksAndRelief verifies CRD-S2 of spec-credit.md 18.3: the credit
// pack with a yield shock that cuts the output of the 60-tick job at tick 60 and one that
// raises it at tick 120, with the pool's relief on (the estates module, PoolRelease 1 000
// basis points) and off, seeds 1 to the seed count, 200 ticks, the audit on. Exact: every
// step passes the audit and the claim identities hold at the end of every run. Reported:
// the defaults, the arrears, the lenders' loss, the households unfed, the relief paid, the
// removals and the claims passed to the estate (STB-L163, L164, L39).
//
// Requirement: FR-CRD-083, FR-CRD-006
func TestScenarioS115_ShocksAndRelief(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	for _, relief := range []bool{false, true} {
		cfg := loadValidPack(t, packOneCredit...)
		name := "relief off"
		if relief {
			cfg = loadValidPack(t, append(append([]string{}, packOneCredit...), "overlay/estates.toml")...)
			cfg.Regions[0].PoolRelease = 1_000
			name = "relief on"
		}
		recipe := growerRecipe(t, cfg)
		for seed := uint64(1); seed <= uint64(scenarioSeedCount()); seed++ {
			t.Run(name+" seed "+strconv.FormatUint(seed, 10), func(t *testing.T) {
				t.Parallel()
				tally := &creditTally{}
				w := creditWorld(t, cfg, seed, tally)
				stepAudited(t, w, seed, creditScenarioTicks, func(tick int) {
					for _, s := range creditYieldShocks {
						if tick+1 == s.tick {
							if err := w.SetYieldShocks([]economysim.YieldShock{{Region: 0, Recipe: recipe, BP: s.bp}}); err != nil {
								t.Fatalf("seed %d: SetYieldShocks at tick %d error = %v, want nil", seed, s.tick, err)
							}
						}
					}
				})
				snap, err := w.Snapshot()
				if err != nil {
					t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
				}
				checkClaimIdentities(t, "S115 seed "+strconv.FormatUint(seed, 10), snap)
				t.Logf("seed %d %s: %s; lost principal %d, interest %d; arrears rows %d of %d open", seed, name, tally.report(),
					snap.Credit.LostPrincipal, snap.Credit.LostInterest, overdueRows(snap), len(snap.Claims))
			})
		}
	}
}

// growerRecipe is the recipe of the 60-tick job of pack one, grow_staple.
func growerRecipe(t testing.TB, cfg economysim.Config) economysim.RecipeID {
	t.Helper()
	for i, r := range cfg.Recipes {
		if r.ID == "grow_staple" {
			return economysim.RecipeID(i)
		}
	}
	t.Fatalf("pack one has no recipe grow_staple")
	return 0
}

// overdueRows counts the claim rows that went unpaid at their due tick.
func overdueRows(s economysim.Snapshot) int {
	n := 0
	for _, c := range s.Claims {
		if c.Overdue >= 0 {
			n++
		}
	}
	return n
}
