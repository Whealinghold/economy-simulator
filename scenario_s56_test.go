package economysim_test

import (
	"fmt"
	"testing"
)

// s56 sizes: 40 households of the forage world, an outage of the market recipe
// from tick 10 to tick 49 (scripts/s56.toml) and 80 ticks. A household starts with
// 6 grain, three ticks of its upkeep of 2, so the outage reaches it at tick 13.
const (
	s56Households int64 = 40
	s56Ticks      int64 = 80
)

// s56Variant is one run of S56: the module off, or on at the coverage of the
// own-use recipe (its yield in basis points), RetryJobs and MinWorkShare.
type s56Variant struct {
	name    string
	module  bool
	cov     int64
	retry   int64
	minWork int64
}

// coverage is the yield of the own-use recipe in basis points, which is Cov for the
// recipe of the world (2 units for a household that needs 2 a tick, one tick of
// work): the module off needs no recipe and keeps the default.
func (v s56Variant) coverage() int64 {
	if !v.module {
		return 10_000
	}
	return v.cov
}

// s56Variants are the runs of the scenario: off, and on at the three coverages of
// spec-subsistence.md 19 (6 667, 10 000 and 13 333), with RetryJobs 0 and 10 and
// MinWorkShare 0 and 4 000.
func s56Variants() []s56Variant {
	vs := []s56Variant{{name: "off"}, {name: "off, floor", minWork: 4_000}}
	for _, cov := range []int64{6_667, 10_000, 13_333} {
		for _, retry := range []int64{0, 10} {
			for _, mw := range []int64{0, 4_000} {
				vs = append(vs, s56Variant{name: fmt.Sprintf("on cov %d retry %d floor %d", cov, retry, mw), module: true, cov: cov, retry: retry, minWork: mw})
			}
		}
	}
	return vs
}

// TestScenarioS56_StapleShortage runs scenario S56 (SUB-S1, plan 2-10 D417) on the
// forage world: an outage of the market recipe, the module off and on, at the
// coverages 6 667, 10 000 and 13 333, RetryJobs 0 and 10 and MinWorkShare 0 and
// 4 000, seeds 1 to scenarioSeedCount(), with the audit on. It asserts the shape of
// the model of spec-subsistence.md 11.2, not its numbers: the unfed household-ticks
// of every variant with a coverage of at least 10 000 are fewer than the module
// off, none of the households is removed, and a coverage of 6 667 is no better than
// the module off (a household that makes two thirds of its upkeep is unfed every
// tick, and the removals are no fewer). It reports each variant's unfed
// household-ticks, removals, units output and the households in the mode. The
// outcomes are written to testdata/chains/expected/subsistence.scenarios.txt with
// -update.
//
// Setup: the world is the small forage world of subsistence_world_test.go, not
// pack one, whose scenarios take minutes and whose households hold stock for
// hundreds of ticks; the price regime of the staple is not reported, because the
// foragers trade nothing.
//
// Requirement: FR-SUB-019, FR-SUB-020, FR-SUB-021, FR-SUB-022, FR-SUB-024
//
// Guard: a floor that does not feed the household, or one that does so only at a
// coverage the model says is too low, fails here.
func TestScenarioS56_StapleShortage(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S56", "forage"
	rep := newPackScenReport("subsistence", fmt.Sprintf("S56 forage outage seeds 1-%d, %d ticks", scenarioSeedCount(), s56Ticks))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				script := outage(t)
				unfed := map[string]int64{}
				removed := map[string]int64{}
				for _, v := range s56Variants() {
					st := runForage(t, forageWorld(forageSpec{Households: s56Households, StartGrain: 6, OwnUseBP: v.coverage(), Duration: 1,
						Module: v.module, RetryJobs: v.retry, MinWork: v.minWork}), seed, s56Ticks, -1, script)
					unfed[v.name], removed[v.name] = st.Unfed, st.Removed
					rep.num(seed, "unfed household-ticks "+v.name, st.Unfed)
					rep.num(seed, "households removed "+v.name, st.Removed)
					if v.module {
						rep.num(seed, "units output "+v.name, st.Units)
						rep.num(seed, "peak households in the mode "+v.name, st.PeakOn)
					}
				}
				for _, v := range s56Variants() {
					if !v.module || v.cov < 10_000 {
						continue
					}
					checkExact(t, rep, scenario, pack, seed, "unfed fewer than off at a coverage of 10 000 or more", unfed[v.name] < unfed["off"],
						"%s: unfed %d, off %d", v.name, unfed[v.name], unfed["off"])
					checkExact(t, rep, scenario, pack, seed, "nobody removed at a coverage of 10 000 or more", removed[v.name] == 0,
						"%s: removed %d", v.name, removed[v.name])
				}
				for _, v := range s56Variants() {
					if v.module && v.cov < 10_000 {
						checkExact(t, rep, scenario, pack, seed, "a coverage of 6 667 is no better than none", unfed[v.name] >= unfed["off"] && removed[v.name] >= removed["off"],
							"%s: unfed %d removed %d, off %d and %d", v.name, unfed[v.name], removed[v.name], unfed["off"], removed["off"])
					}
				}
			})
		}
	})
	rep.flush(t)
}
