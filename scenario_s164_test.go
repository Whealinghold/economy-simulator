package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestScenarioINS5_PriceSpikeAndValueCap is INS-S5 (S164, spec-insurance.md 17, 6.2, INS-A3): the store
// pack with a harvest failure of the staple of the north from tick 10 to tick 60 (a yield of 2 000 of
// 10 000, which raises the price that values the stock), with the value cap at 20 000 and at 100 000 of
// 10 000 of the reference price (the bound, the nearest to no cap), seeds 1 to the scenario count, 80
// ticks, audit on. Exact: the books add up every tick at each cap. Recorded, not asserted: the sum
// insured, the premium, the units valued at the cap and the payout, so that the effect of a price that
// rises with the loss it values shows as a difference between the two caps.
//
// Requirement: FR-INS-006, FR-INS-020
//
// Guard: a cap that was not applied would leave the units valued at the cap at 0 in the capped run
// while the spike lifts the price above it.
func TestScenarioINS5_PriceSpikeAndValueCap(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	g := insGoldenStore(t)
	script := []scriptStep{{Tick: 10, Region: "north", Recipe: "grow_staple", BP: 2_000}, {Tick: 60, Region: "north", Recipe: "grow_staple", BP: 10_000}}
	for _, capBP := range []int64{20_000, 100_000} {
		cfg := withValueCap(g.cfg, capBP)
		techScenario(t, "one", fmt.Sprintf("S164 one-store a price spike, value cap %d, 80 ticks", capBP), func(t *testing.T, rep *scenReport, seed uint64) {
			tr := newInsTrack(t, cfg, seed, g.opts)
			names := mustSnapshot(t, tr.w).Names
			msg := tr.run(t, 80, func(w *economysim.World) { applyScript(t, w, names, script) })
			checkExact(t, rep, "S164", g.name, seed, "the books add up every tick", msg == "", "%s", msg)
			rep.num(seed, "sum insured", tr.sum)
			rep.num(seed, "premiums", tr.premiums)
			rep.num(seed, "units valued at the cap", tr.roundCapped)
			rep.num(seed, "due", tr.due)
		})
	}
}
