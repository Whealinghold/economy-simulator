package economysim_test

import (
	"testing"
)

// TestScenarioRSK2_Accounting is RSK-S2 (S127, spec-risk.md 16, 7.3): each pack of the
// module-alone goldens is run with the module on, audit on, seeds 1 to the scenario count, 400
// ticks, and after every tick the boundary balance of the reason loss equals the units the events
// report as lost (cargo lines, trip accounts and stores), and every cargo line adds up: Shipped =
// Delivered + Diverted + Spoiled + Lost (FR-RSK-014, FR-RSK-022, FR-RSK-023). Exact. Recorded: the
// shipments struck and the units lost.
//
// Requirement: FR-RSK-014, FR-RSK-022, FR-RSK-023, FR-RSK-033
//
// Guard: a loss applied without its event, or an event without the move, would break the books
// at one tick and only there; the check runs after every tick.
func TestScenarioRSK2_Accounting(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	for _, g := range []riskGolden{riskGoldenTwo(t), riskGoldenStore(t)} {
		techScenario(t, riskPackName(g), "S127 "+g.name+" the books of the loss every tick, 400 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
			rw := newRiskWatch(t, g.cfg, seed, g.opts)
			msg := rw.run(t, goldenTicks)
			checkExact(t, rep, "S127", g.name, seed, "the loss boundary equals the events and every line adds up, every tick", msg == "", "%s", msg)
			rep.num(seed, "shipments struck", rw.tally.struck)
			rep.num(seed, "units lost from cargo and trip accounts", rw.tally.routeLost+rw.tally.accountLost)
			rep.num(seed, "units lost from stores", rw.tally.storeLost)
		})
	}
}
