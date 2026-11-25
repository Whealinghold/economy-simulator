package economysim_test

import (
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// hirePackTicks is the length of the runs of the scenarios of the hire form on one-demand.
// The pack collapses after about tick 270 (N-352), so the figures are read over the first 400.
const hirePackTicks = 400

// meanOver is the mean of a series over [from, to), or 0 where the window is empty.
func meanOver(s []int64, from, to int64) int64 {
	m, _ := windowMean(s, from, min(to, int64(len(s))))
	return m
}

// TestScenarioS97_HiredHouseholdsWithWantsOffAndOn is EMP-S1 (spec-employment.md 17): the pack
// one-demand with overlay/employment-hire.toml, whose cooks are the employers and whose
// diggers and foresters are the Hireable households, run for 400 ticks over seeds 1 to 10
// with the audit on, with the employers' wants off (no taste row for the cooks) and then on
// (the pack as it is). Exact, every tick and seed: the money of the world moves only by the
// boundary flows, no hire is formed without a bill, each hire is paid and refunded to the
// bill, the escrow is the money of the hires in progress, and a second run of the seed gives
// the same hires and the same final snapshot. Recorded, not asserted: the households hired,
// the positions posted and unfilled and the households with no vacancy, wants off beside
// wants on. The closed form h* of 11.2 holds for a one-good
// circuit, which the pack is not, and the pack collapses (N-352), so what the wants change
// is a measure here, not a band.
//
// Requirement: FR-EMP-036, FR-EMP-039, FR-EMP-062
//
// Guard: a wage that created or lost a unit of money, or a hire with no bill, is seen in
// every tick and every seed, not only in the golden's seed.
func TestScenarioS97_HiredHouseholdsWithWantsOffAndOn(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	for _, v := range []struct {
		name string
		cfg  economysim.Config
	}{{"wants off", hirePackWantsOff(t)}, {"wants on", loadValidPack(t, packOneDemandHire...)}} {
		hireScenario(t, "one-demand", "S97 one-demand+hire employers' "+v.name+", 400 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
			a := runEmp(t, v.cfg, seed, hirePackTicks, nil)
			a.invariants(t, rep, "S97", "one-demand", seed, true)
			a.report(rep, seed)
			rep.num(seed, "mean households hired a tick, ticks 10 to 100", meanOver(a.tape.formedHH, 10, 100))
			b := runEmp(t, v.cfg, seed, hirePackTicks, nil)
			same := a.tape.formed == b.tape.formed && reflect.DeepEqual(a.tape.formedHH, b.tape.formedHH) && reflect.DeepEqual(a.run.Final, b.run.Final)
			checkExact(t, rep, "S97", "one-demand", seed, "a second run of the seed gives the same hires and the same final snapshot", same, "%d and %d hires formed", a.tape.formed, b.tape.formed)
		})
	}
}
