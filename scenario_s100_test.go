package economysim_test

import (
	"testing"
)

// The step of S100: the yield of cut_fuel, a input of the cooks' recipes, is halved from tick
// 60 and restored at tick 100, in a run of 200 ticks (the pack collapses after about 230,
// N-352, so the recovery is read before it).
const (
	s100From, s100To, s100Ticks = 60, 100, 200
)

// TestScenarioS100_StepInTheYieldOfTheEmployersInput is EMP-S4 (spec-employment.md 17): the
// pack one-demand with the hire overlay, a step in the yield of the fuel the cooks use
// (cut_fuel at 5 000 basis points from tick 60 to tick 100), 200 ticks, seeds 1 to 10, audit
// on. Exact: the invariants of every employment scenario hold through the step and after it.
// Recorded, not asserted: the mean positions posted and the households with no vacancy
// before the step, during it and after it, the fewest positions of a window of 10 ticks in
// the step, and the recovery time of the positions posted, in ticks after the step ends, to
// within 25% of their mean before it (EMP-A2: report only, as the pack collapses and the
// positions may never return to their old level).
//
// Requirement: FR-EMP-015
//
// Guard: a step that left a hire open, or a wage paid on a job the shock cut to nothing,
// would show in the escrow and the money of the ticks around it.
func TestScenarioS100_StepInTheYieldOfTheEmployersInput(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, packOneDemandHire...)
	script := []scriptStep{
		{Tick: s100From, Region: "north", Recipe: "cut_fuel", BP: 5_000},
		{Tick: s100To, Region: "north", Recipe: "cut_fuel", BP: 10_000},
	}
	hireScenario(t, "one-demand", "S100 one-demand+hire step in the yield of the cooks' fuel, 200 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		r := runEmp(t, cfg, seed, s100Ticks, script)
		r.invariants(t, rep, "S100", "one-demand", seed, true)
		l := r.tape
		rep.num(seed, "mean positions posted a tick, ticks 30 to 60", meanOver(l.posted, 30, s100From))
		rep.num(seed, "mean positions posted a tick, in the step", meanOver(l.posted, s100From, s100To))
		rep.num(seed, "mean positions posted a tick, ticks 100 to 140", meanOver(l.posted, s100To, 140))
		rep.num(seed, "mean households with no vacancy a tick, ticks 30 to 60", meanOver(l.noVacancy, 30, s100From))
		rep.num(seed, "mean households with no vacancy a tick, in the step", meanOver(l.noVacancy, s100From, s100To))
		rep.num(seed, "mean households with no vacancy a tick, ticks 100 to 140", meanOver(l.noVacancy, s100To, 140))
		low := int64(-1)
		for k := int64(s100From); k+10 <= s100To; k++ {
			if m := meanOver(l.posted, k, k+10); low < 0 || m < low {
				low = m
			}
		}
		rep.num(seed, "fewest positions posted in a window of 10 ticks in the step", low)
		p0, ok := windowMean(l.posted, 30, s100From)
		if !ok || p0 == 0 {
			rep.absent(seed, "ticks to recover the positions posted after the step")
			return
		}
		if n, ok := recoveryTicks(l.posted[:min(len(l.posted), s100Ticks)], s100To, p0, 2_500, 10); ok {
			rep.num(seed, "ticks to recover the positions posted after the step", n)
		} else {
			rep.absent(seed, "ticks to recover the positions posted after the step")
		}
	})
}
