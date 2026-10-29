package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s57 sizes: pack famine (cold, the estates module on) for 300 ticks with the
// harvest failure of S14 at depth 0 from tick 100 to tick 189, the module off and
// on (overlay/own-use.toml), and a snapshot at tick 150, inside the failure.
const (
	s57Ticks int64 = 300
	s57Mid   int64 = 150
)

// s57Script is the failure of S14 at depth 0, moved to ticks 100 to 189 so that
// a run of 300 ticks covers it and the recovery.
var s57Script = []shockStep{
	{Tick: 100, Region: "north", Recipe: "grow_staple", BP: 0},
	{Tick: 190, Region: "north", Recipe: "grow_staple", BP: 10_000},
}

// TestScenarioS57_FamineWithTheFloor runs scenario S57 (SUB-S2, plan 2-10 D417) on
// pack famine with the estates module on, 300 ticks and the failure of S14 at
// depth 0 from tick 100 to 189, the module off and on, seeds 1 to
// scenarioSeedCount(), with the audit on. It asserts that with the module on the
// unfed household-ticks, the households removed and the pool at the end are lower
// (the plan's reading of the spec; the relief paid is reported, not asserted, since
// it is higher with the module on in some seeds: the households that survive draw
// it for longer); that at tick 150 every role with households in the mode has a mean
// welfare ratio above 0 (a household that is fed by the floor reads what the floor
// covers, FR-SUB-028, not 0); and that in the closed upkeep-only world of the
// forage world, where every household makes exactly its upkeep and nothing else,
// the mean ratio of the role is 10 000 (spec-subsistence.md 7.2). It reports the
// unfed household-ticks, the removals, the relief paid, the pool at the end and the
// units output. The outcomes are written to
// testdata/chains/expected/famine.scenarios.txt with -update.
//
// Requirement: FR-SUB-028, FR-SUB-033
//
// Guard: a floor that feeds nobody, or whose output the welfare record counts as
// nothing, leaves the famine as it was.
func TestScenarioS57_FamineWithTheFloor(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S57", "famine+own-use"
	off := loadValidPack(t, packLayers["famine"]...)
	on := loadValidPack(t, append(slices.Clone(packLayers["famine"]), "overlay/own-use.toml")...)
	rep := newPackScenReport("famine", fmt.Sprintf("S57 famine+own-use seeds 1-%d, %d ticks", scenarioSeedCount(), s57Ticks))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				base := runFamine(t, off, seed)
				with := runFamine(t, on, seed)
				for _, r := range []struct {
					name string
					m    famineStats
				}{{"module off", base}, {"module on", with}} {
					rep.num(seed, "unfed household-ticks, "+r.name, r.m.Unfed)
					rep.num(seed, "households removed, "+r.name, r.m.Removed)
					rep.num(seed, "relief paid, "+r.name, r.m.Relief)
					rep.num(seed, "pool at the end, "+r.name, r.m.Pool)
				}
				rep.num(seed, "units output by own-use jobs", with.Units)
				checkExact(t, rep, scenario, pack, seed, "unfed fewer with the module on", with.Unfed < base.Unfed, "on %d, off %d", with.Unfed, base.Unfed)
				checkExact(t, rep, scenario, pack, seed, "removals fewer with the module on", with.Removed < base.Removed, "on %d, off %d", with.Removed, base.Removed)
				checkExact(t, rep, scenario, pack, seed, "pool at the end lower with the module on", with.Pool < base.Pool, "on %d, off %d", with.Pool, base.Pool)
				var fed, zero int
				for _, row := range with.Mid.Roles {
					if row.OnOwnUse > 0 && row.HasMeanRatio {
						fed++
						if row.MeanRatioBP == 0 {
							zero++
						}
					}
				}
				checkExact(t, rep, scenario, pack, seed, "a role in the mode reads a ratio above 0", fed > 0 && zero == 0,
					"%d roles in the mode with a ratio, %d of them at 0", fed, zero)
			})
		}
	})
	rep.flush(t)
}

// famineStats are the measures of one run of S57.
type famineStats struct {
	forageStats
	Relief, Pool int64
}

// runFamine runs cfg for s57Ticks with the failure and returns its measures; the
// relief is summed from the events.
func runFamine(t testing.TB, cfg economysim.Config, seed uint64) famineStats {
	t.Helper()
	var relief int64
	obs := economysim.ObserverFunc(func(e economysim.Event) {
		if r, ok := e.(economysim.ReliefPaid); ok {
			relief += int64(r.Paid)
		}
	})
	st := runForage(t, cfg, seed, s57Ticks, s57Mid, s57Script, economysim.WithObserver(obs))
	var pool int64
	for _, g := range st.Final.Regions {
		pool += int64(g.MoneyPool)
	}
	return famineStats{forageStats: st, Relief: relief, Pool: pool}
}

// TestScenarioS57_ClosedUpkeepOnlyReadsTenThousand verifies, in the forage world
// with the market recipe dead from tick 0, that a household that makes exactly its
// upkeep and has nothing else reads a mean welfare ratio of 10 000 at any price:
// the receipt is the units created valued at the cost price, and the cost is the
// upkeep at the same price (spec-subsistence.md 7.2).
//
// Arithmetic: gather makes 2 grain in 1 tick, the upkeep is 2 grain, so the
// receipt over the ring is 2 * p and the cost 2 * p, a ratio of 10 000.
//
// Requirement: FR-SUB-028
func TestScenarioS57_ClosedUpkeepOnlyReadsTenThousand(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	script := []shockStep{{Tick: 0, Region: "home", Recipe: "tend", BP: 0}}
	for _, seed := range seedList(min(3, scenarioSeedCount())) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			st := runForage(t, forageWorld(forageSpec{Households: 20, OwnUseBP: 10_000, Duration: 1, Module: true, RetryJobs: 10}), seed, 60, 59, script)
			var row *economysim.RoleRow
			for i := range st.Final.Roles {
				row = &st.Final.Roles[i]
			}
			if row == nil || !row.HasMeanRatio || row.MeanRatioBP != 10_000 || row.OnOwnUse != 20 || st.Unfed != 0 {
				t.Errorf("seed %d: role row = %+v, unfed household-ticks = %d, want 20 households in the mode, a mean ratio of 10000 and nobody unfed", seed, row, st.Unfed)
			}
		})
	}
}
