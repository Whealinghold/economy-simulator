package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestScenarioINS4_CommonRowAndCapacity is INS-S4 (S163, spec-insurance.md 17, INS-A2, A4): the route
// pack, whose lane from south to north is a common row (one draw a tick for the whole origin, so every
// shipment of the tick is struck together), with the underwriters' cap at 5 000, 10 000 and 30 000 of
// 10 000 of their capital, seeds 1 to the scenario count, 400 ticks, audit on. Exact: the books add up
// every tick at each cap. Recorded, not asserted (INS-A2 is undamped in version 1, plan 3-ins R7): the
// agents removed, the shortfall and the claims written off, the offers refused for exposure and the
// requests no offer covered.
//
// Requirement: FR-INS-006, FR-INS-013, FR-INS-044, FR-INS-046
//
// Guard: a cap that was read as a share of the free capacity, or an exposure test that let an
// underwriter pass its cap, would show as a book that drifts or a refusal count that does not move.
func TestScenarioINS4_CommonRowAndCapacity(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	g := insGoldenTwo(t)
	for _, capBP := range []int64{5_000, 10_000, 30_000} {
		cfg := withUnderwrite(g.cfg, func(u *economysim.UnderwriteTerms) { u.CapBP = capBP })
		techScenario(t, "two", fmt.Sprintf("S163 two a common row, underwriters' cap %d, 400 ticks", capBP), func(t *testing.T, rep *scenReport, seed uint64) {
			tr := newInsTrack(t, cfg, seed, g.opts)
			msg := tr.run(t, goldenTicks, nil)
			checkExact(t, rep, "S163", g.name, seed, "the books add up every tick", msg == "", "%s", msg)
			if tr.exposure == 0 {
				t.Errorf("S163 seed %d cap %d: no offer was refused for exposure in %d ticks, want the cap to bind (the scenario would pass for nothing)", seed, capBP, goldenTicks)
			}
			_, lost := payoutClaims(tr.rec.events)
			rep.num(seed, "policies formed", tr.tl.formed)
			rep.num(seed, "agents removed", tr.removed)
			rep.num(seed, "shortfall formed", tr.short)
			rep.num(seed, "principal written off", lost)
			rep.num(seed, "offers refused for exposure", tr.exposure)
			rep.num(seed, "requests no offer covered", tr.noOffer)
		})
	}
}
