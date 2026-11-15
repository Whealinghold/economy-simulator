package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packOneTechnology is pack one with the fixed factor and the population module of
// S39 and S41 and overlay/technology.toml (learning only on the growers): the pack of
// S149, S151 and S152 (spec-reference-chain.md 4.5, plan 3-tec question 5).
var packOneTechnology = append(slices.Clone(packOnePopResources), "overlay/technology.toml")

// techTicks is the length of the TEC scenarios that run on pack one: a year.
const techTicks = 365

// techRun is what one run of a TEC scenario keeps: the sums of what accrued by
// mechanism, the first break of the chain of gains and the last snapshot, never the
// events (a scenario over ten seeds keeps numbers, not logs).
type techRun struct {
	advanced                         int64
	doing, trend, discovery, catchUp int64
	discoveries, lost                int64
	bad                              string
	final                            economysim.Snapshot
	capBP                            int64
	last                             map[[2]int64]int64
	row                              economysim.TechRow
	hasRow                           bool
	maxGain                          int64
}

// Observe folds a TechnologyAdvanced or a TechnologyAdjusted into the run. The gains of
// a cell chain: each BeforeBP is the AfterBP of the last event of the cell, a call
// included, none is above the cap and none falls without a call.
func (r *techRun) Observe(e economysim.Event) {
	switch ev := e.(type) {
	case economysim.TechnologyAdvanced:
		r.advanced++
		r.doing, r.trend, r.discovery, r.catchUp = r.doing+ev.DoingAddedBP, r.trend+ev.TrendAddedBP, r.discovery+ev.DiscoveryAddedBP, r.catchUp+ev.CatchUpAddedBP
		r.discoveries += ev.Discoveries
		key := [2]int64{int64(ev.Region), int64(ev.Field)}
		switch {
		case ev.BeforeBP != r.last[key]:
			r.bad = fmt.Sprintf("tick %d region %d: BeforeBP %d, want the last AfterBP %d", ev.Tick, ev.Region, ev.BeforeBP, r.last[key])
		case ev.AfterBP < ev.BeforeBP:
			r.bad = fmt.Sprintf("tick %d region %d: the gain fell from %d to %d without a call", ev.Tick, ev.Region, ev.BeforeBP, ev.AfterBP)
		case r.capBP > 0 && ev.AfterBP > r.capBP:
			r.bad = fmt.Sprintf("tick %d region %d: gain %d above the cap %d", ev.Tick, ev.Region, ev.AfterBP, r.capBP)
		}
		r.last[key] = ev.AfterBP
		r.maxGain = max(r.maxGain, ev.AfterBP)
	case economysim.TechnologyAdjusted:
		r.lost += ev.LostBP
		r.last[[2]int64{int64(ev.Region), int64(ev.Field)}] = ev.AfterBP
	}
}

// runTechnology steps a world of cfg under seed for ticks ticks with the audit and a
// techRun observer; call, when set, is made between ticks before the step of tick i.
func runTechnology(t *testing.T, cfg economysim.Config, seed uint64, ticks int, call func(*economysim.World, int)) *techRun {
	t.Helper()
	r := &techRun{last: map[[2]int64]int64{}}
	if m := cfg.Modules.Technology; m != nil && len(m.Fields) > 0 {
		r.capBP = m.Fields[0].CapBP
	}
	w := newPackWorld(t, cfg, seed, economysim.WithAudit(), economysim.WithObserver(r))
	for i := range ticks {
		if call != nil {
			call(w, i)
		}
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, i, err)
		}
	}
	var err error
	if r.final, err = w.Snapshot(); err != nil {
		t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
	}
	if len(r.final.Tech) > 0 {
		r.row, r.hasRow = r.final.Tech[0], true
	}
	return r
}

// reportTech records the measures every TEC scenario reads under seed.
func reportTech(rep *scenReport, seed uint64, r *techRun) {
	rep.num(seed, "households at end", r.final.Households)
	rep.num(seed, "TechnologyAdvanced events", r.advanced)
	rep.num(seed, "gain added by learning bp", r.doing)
	rep.num(seed, "gain added by the trend bp", r.trend)
	rep.num(seed, "gain added by discoveries bp", r.discovery)
	rep.num(seed, "gain added by catch-up bp", r.catchUp)
	if r.hasRow {
		rep.num(seed, "gain of the first cell at end bp", r.row.GainBP)
		rep.num(seed, "experience of the first cell at end", r.row.Experience)
	} else {
		rep.absent(seed, "gain of the first cell at end bp")
		rep.absent(seed, "experience of the first cell at end")
	}
}

// techScenario runs fn for the seeds of the scenario and records the block of pack
// under title; each seed is a parallel subtest and the report is flushed after all.
func techScenario(t *testing.T, pack, title string, fn func(t *testing.T, rep *scenReport, seed uint64)) {
	t.Helper()
	rep := newPackScenReport(pack, fmt.Sprintf("%s seeds 1-%d", title, scenarioSeedCount()))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(scenarioSeedCount()) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				fn(t, rep, seed)
			})
		}
	})
	rep.flush(t)
}

// checkChain asserts the exact check every TEC scenario shares: the gains of every cell
// chained and within the cap.
func checkChain(t *testing.T, rep *scenReport, scenario, pack string, seed uint64, r *techRun) {
	t.Helper()
	checkExact(t, rep, scenario, pack, seed, "the gains of each cell chain, never fall and stay within the cap", r.bad == "", "%s", r.bad)
}

// TestScenarioS149_LearningWithPopulation reports TEC-S2 of spec-technology.md 14.4:
// learning only on pack one with the population module and the fixed factor of
// resources, a year, seeds 1 to 10, audit on. It asserts what is exact (the books balance
// every tick, the gains chain and stay within the cap) and records, never asserts, what
// the bands of STY will judge: the gain and the experience of the cell, the households at
// the end. The pack collapses under its own rules (N-352), so the growth a gain would
// cause cannot be shown here (plan 3-tec R4).
//
// Requirement: FR-TEC-005, FR-TEC-006, FR-TEC-007, FR-TEC-017, FR-TEC-024
//
// Guard: a gain that fell without a call, or passed the cap, fails here.
func TestScenarioS149_LearningWithPopulation(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, packOneTechnology...)
	techScenario(t, "one", "S149 one+population+resources+technology learning, 365 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		r := runTechnology(t, cfg, seed, techTicks, nil)
		reportTech(rep, seed, r)
		checkChain(t, rep, "S149", "one", seed, r)
		checkExact(t, rep, "S149", "one", seed, "only learning accrues", r.trend+r.discovery+r.catchUp == 0, "trend %d, discovery %d, catch-up %d", r.trend, r.discovery, r.catchUp)
	})
}

// TestScenarioS151_CapAroundTheStockThreshold reports TEC-S4: the same pack with the cap
// of the field at 1 000 and at 9 000 basis points, a year, seeds 1 to 10. The two caps
// straddle the threshold of spec-technology.md 10.3 for a refilling stock; the gain of
// the lower never passes 1 000 and the higher is not capped by the same rule. The stock
// share and the households at the end are recorded, not asserted (N-352).
//
// Requirement: FR-TEC-005, FR-TEC-030
func TestScenarioS151_CapAroundTheStockThreshold(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	for _, capBP := range []int64{1_000, 9_000} {
		t.Run(fmt.Sprintf("cap %d", capBP), func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, packOneTechnology...)
			cfg.Modules.Technology.Fields[0].CapBP = capBP
			techScenario(t, "one", fmt.Sprintf("S151 one+population+resources cap %d, 365 ticks", capBP), func(t *testing.T, rep *scenReport, seed uint64) {
				r := runTechnology(t, cfg, seed, techTicks, nil)
				reportTech(rep, seed, r)
				if len(r.final.Endowments) > 0 {
					rep.num(seed, "stock of the first endowment at end", r.final.Endowments[0].Stock)
				}
				checkChain(t, rep, "S151", "one", seed, r)
				checkExact(t, rep, "S151", "one", seed, "the gain stays within the cap of the variant", r.maxGain <= capBP, "largest gain %d, cap %d", r.maxGain, capBP)
			})
		})
	}
}

// TestScenarioS152_StapleWithAndWithoutTheFloor reports TEC-S5: pack one with learning
// only, a year, seeds 1 to 10, with the work-share floor of every role at its default and
// at 0. The households and the gain at the end are recorded for both; the claim "a gain is
// rest first" of spec-technology.md 13.5 is a finding for STY, not an assertion here.
//
// Requirement: FR-TEC-005, FR-TEC-018
func TestScenarioS152_StapleWithAndWithoutTheFloor(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	for _, floor := range []string{"default", "zero"} {
		t.Run(floor, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, packOneTechnology...)
			if floor == "zero" {
				for i := range cfg.Roles {
					cfg.Roles[i].MinWorkShare = 0
				}
			}
			techScenario(t, "one", fmt.Sprintf("S152 one+technology work floor %s, 365 ticks", floor), func(t *testing.T, rep *scenReport, seed uint64) {
				r := runTechnology(t, cfg, seed, techTicks, nil)
				reportTech(rep, seed, r)
				checkChain(t, rep, "S152", "one", seed, r)
			})
		})
	}
}
