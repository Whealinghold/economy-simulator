package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// labourBoundMaxHires is the establishment of a cook in the labour-bound variant of the
// pack: 40 hired executions in progress at once, ten times the overlay's 4, against the 118
// households of the Hireable roles, so the places the employers can post far exceed the
// households that offer.
const labourBoundMaxHires = 40

// hirePackLabourBound is the pack one-demand with the hire overlay, the cooks' establishment
// raised to labourBoundMaxHires and the work floor (Role.MinWorkShare) of the Hireable roles
// set to floor, in basis points.
func hirePackLabourBound(t testing.TB, floor int64) economysim.Config {
	t.Helper()
	cfg := loadValidPack(t, packOneDemandHire...)
	for i := range cfg.Roles {
		switch r := &cfg.Roles[i]; {
		case r.Employer != nil:
			r.Employer.MaxHires = labourBoundMaxHires
		case r.Hireable:
			r.MinWorkShare = floor
		}
	}
	return cfg
}

// windowWage is the mean wage per household of the formations of [from, to), half to even,
// and whether any formed.
func windowWage(l *labourTape, from, to int64) (int64, bool) {
	to = min(to, int64(len(l.wageHH)))
	var wage, hh int64
	for k := max(from, 0); k < to; k++ {
		wage += l.wageHH[k]
		hh += l.formedHH[k]
	}
	return divHE(wage, hh)
}

// TestScenarioS98_LabourBoundWithTheFloorAtDefaultAndZero is EMP-S2 (spec-employment.md 17): a
// labour-bound variant of the pack one-demand with the hire overlay (the cooks may post ten
// times the places, the households that offer are the same), the work floor of the Hireable
// roles at the default of 4 000 and at 0, 400 ticks, seeds 1 to 10, audit on. Exact: the
// published price of every market stays inside the belief interval of FR-BEL-004 (FR-STB-008)
// after every tick, and the invariants of every employment scenario hold. Recorded, not
// asserted: the households that offered and found no vacancy, the positions posted and
// unfilled, the mean wage and the households hired in the first and the second window of 50
// ticks and the arc elasticity of the hired households with respect to the wage between them
// (EMP-A1, the days worked of a target-income worker); the stated condition in which the
// Hireable roles end is their households at the end of the run.
//
// Requirement: FR-EMP-016, FR-EMP-046, FR-EMP-060, FR-STB-008
//
// Guard: a floor that let the hire pull a worker's days to 0 would show as a price out of its
// bounds or a household count that does not close, and only a run with the floor at 0 sees it.
func TestScenarioS98_LabourBoundWithTheFloorAtDefaultAndZero(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	for _, floor := range []int64{4_000, 0} {
		cfg := hirePackLabourBound(t, floor)
		hireScenario(t, "one-demand", fmt.Sprintf("S98 one-demand+hire labour-bound, work floor %d, 400 ticks", floor), func(t *testing.T, rep *scenReport, seed uint64) {
			r := runEmp(t, cfg, seed, hirePackTicks, nil)
			r.invariants(t, rep, "S98", "one-demand", seed, true)
			r.report(rep, seed)
			bv := boundsViolations(cfg, r.run.Names, r.run.Obs, 3)
			checkExact(t, rep, "S98", "one-demand", seed, "price within belief bounds (FR-STB-008)", len(bv.Violations) == 0, "%v", bv.Violations)
			l := r.tape
			rep.num(seed, "mean households hired a tick, ticks 10 to 60", meanOver(l.formedHH, 10, 60))
			rep.num(seed, "mean households hired a tick, ticks 60 to 110", meanOver(l.formedHH, 60, 110))
			p0, ok0 := windowWage(l, 10, 60)
			p1, ok1 := windowWage(l, 60, 110)
			rep.num(seed, "mean wage, ticks 10 to 60", p0)
			rep.num(seed, "mean wage, ticks 60 to 110", p1)
			if e, ok := arcElasticityBP(meanOver(l.formedHH, 10, 60), p0, meanOver(l.formedHH, 60, 110), p1); ok && ok0 && ok1 {
				rep.num(seed, "arc elasticity of hired households to the wage, bp", e)
			} else {
				rep.absent(seed, "arc elasticity of hired households to the wage, bp")
			}
		})
	}
}
