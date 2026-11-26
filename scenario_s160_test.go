package economysim_test

import (
	"testing"
)

// TestScenarioINS1_NeutralityAndAccounting is INS-S1 (S160, spec-insurance.md 17, FR-INS-005,
// FR-INS-006): each pack of the module-alone goldens is run twice. (1) With the module on and no
// role block, 100 ticks, every event but WorldCreated and the whole snapshot but the module's own
// parts equal those of the run with the module off (the module on with no block is the module
// off). (2) With the blocks, audit on, seeds 1 to the scenario count, 400 ticks, after every tick
// the totals of the snapshot equal the sums of the CoverFormed and CoverEnded events and every
// policy's due is the sum of what was paid, owed, dropped and cut. Exact. Recorded: the policies
// formed, the premiums and the payouts.
//
// Requirement: FR-INS-001, FR-INS-005, FR-INS-006, FR-INS-079, NFR-INS-003
//
// Guard: a cover round that took money or a draw with no block, or a total that drifted from its
// events, breaks the equality at the tick it happens.
func TestScenarioINS1_NeutralityAndAccounting(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	for _, g := range []insGolden{insGoldenTwo(t), insGoldenStore(t)} {
		techScenario(t, insNames(g), "S160 "+g.name+" neutrality with no block, accounting with blocks, 400 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
			msg := insNeutral(t, g.cfg, g.opts, seed, 100)
			checkExact(t, rep, "S160", g.name, seed, "the module on with no block equals the module off", msg == "", "%s", msg)
			tr := newInsTrack(t, g.cfg, seed, g.opts)
			msg = tr.run(t, goldenTicks, nil)
			checkExact(t, rep, "S160", g.name, seed, "the totals equal the events and every due adds up, every tick", msg == "", "%s", msg)
			// Not a report check (the recorded files would change): a run that formed or paid nothing proves nothing above.
			if tr.tl.formed == 0 || tr.paid == 0 {
				t.Errorf("S160 pack %s seed %d: %d policies formed and %d paid in %d ticks, want both above 0 (the scenario would pass for nothing)", g.name, seed, tr.tl.formed, tr.paid, goldenTicks)
			}
			rep.num(seed, "policies formed", tr.tl.formed)
			rep.num(seed, "premiums", tr.premiums)
			rep.num(seed, "due", tr.due)
			rep.num(seed, "paid", tr.paid)
		})
	}
}
