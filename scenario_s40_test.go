package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s40Ticks is the length of a run of S40.
const s40Ticks = 400

// TestScenarioS40_ClosedCeiling runs scenario S40 (POP-S4, plan 2-11 D437) on pack
// one with the demand module off. It asserts, as exact checks: the defaults of the
// module (the schedules cross at 12 000) give the warning
// replacement_above_closed_ceiling, because a household of a closed economy without
// the demand module cannot read more than 10 000; overlay/population-low.toml
// (crossing at 9 500) does not, and its run (400 ticks, seeds 1 to
// scenarioSeedCount(), the audit on) has a share of unfed households among those
// the schedule saw and the ladder left (Unfed over Exposed plus Unfed) that is
// reported and is a share; Renewal 0 gives neither Renewal warning, a non-default
// Renewal above 0 gives renewal_with_population only, the default 10 000 gives
// renewal_ignored only, and a run with Renewal 5 000 ends in the same household
// count and the same births and deaths as the run with Renewal 0, because the
// module reads an effective renewal of 0 (FR-POP-020).
//
// Requirement: FR-POP-020, FR-POP-028
//
// Guard: a Renewal that still fed the spawner while the module was on would move
// the second run away from the first.
func TestScenarioS40_ClosedCeiling(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S40", "one+population-low"
	rep := newPackScenReport("one", fmt.Sprintf("S40 %s seeds 1-%d, %d ticks", pack, scenarioSeedCount(), s40Ticks))
	has := func(cfg economysim.Config, rule string) bool {
		return slices.Contains(warnRules(cfg, "modules", "population"), rule) || slices.Contains(warnRules(cfg, "world", "renewal"), rule)
	}
	defaults := loadValidPack(t, packOne...)
	mod := economysim.DefaultPopulation()
	defaults.Modules.Population = &mod
	low := loadValidPack(t, packOnePopulation...)
	checkExact(t, rep, scenario, pack, 0, "defaults warn replacement_above_closed_ceiling", has(defaults, "replacement_above_closed_ceiling"), "warnings %v", warnRules(defaults, "modules", "population"))
	checkExact(t, rep, scenario, pack, 0, "population-low does not", !has(low, "replacement_above_closed_ceiling"), "warnings %v", warnRules(low, "modules", "population"))
	for _, r := range []struct {
		renewal       int64
		ignored, with bool
	}{{0, false, false}, {5_000, false, true}, {10_000, true, false}} {
		cfg := low
		cfg.World.Renewal = r.renewal
		checkExact(t, rep, scenario, pack, 0, fmt.Sprintf("renewal %d warnings", r.renewal),
			has(cfg, "renewal_ignored") == r.ignored && has(cfg, "renewal_with_population") == r.with,
			"renewal_ignored %v, renewal_with_population %v, want %v, %v", has(cfg, "renewal_ignored"), has(cfg, "renewal_with_population"), r.ignored, r.with)
	}
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				st := runPopulation(t, low, seed, s40Ticks, nil)
				reportPop(rep, seed, "population-low", st)
				share := st.Unfed * 10_000 / max(st.Exposed+st.Unfed, 1)
				rep.num(seed, "unfed share of the exposed, bp", share)
				checkExact(t, rep, scenario, pack, seed, "counts of PopulationChanged add up", st.Bad == "", "first bad event %s", st.Bad)
				checkExact(t, rep, scenario, pack, seed, "unfed share is a share", st.Exposed+st.Unfed > 0 && share <= 10_000, "unfed %d, exposed %d", st.Unfed, st.Exposed)
				other := low
				other.World.Renewal = 5_000
				a, b := runPopulation(t, low, seed, 100, nil), runPopulation(t, other, seed, 100, nil)
				checkExact(t, rep, scenario, pack, seed, "Renewal 5 000 is ignored",
					a.End == b.End && a.Births == b.Births && a.Deaths == b.Deaths, "Renewal 0: %d households, %d births, %d deaths; 5 000: %d, %d, %d", a.End, a.Births, a.Deaths, b.End, b.Births, b.Deaths)
			})
		}
	})
	rep.flush(t)
}
