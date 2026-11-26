package economysim_test

import (
	"testing"
)

// TestScenarioINS3_LaneWithAndWithoutCover is INS-S3 (S162, spec-insurance.md 17, INS-A1, A5, A7): the
// route pack, whose scripted shipper stands in for a merchant (no merchants pack carries the transport,
// risk, credit and cover layers), run with the cover blocks and without them, seeds 1 to the scenario
// count, 400 ticks, audit on. Exact: the books of the world with cover add up every tick, and the world
// without a block forms no policy. Recorded, not asserted (N-027): the units shipped and landed, the
// units lost, the premium paid, the payout received and the agents removed, with and without cover.
//
// Requirement: FR-INS-005, FR-INS-006, FR-INS-079
//
// Guard: a world without a block that formed a policy, or one with cover whose books drifted, fails
// here before the numbers the bands will judge are read.
func TestScenarioINS3_LaneWithAndWithoutCover(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	g := insGoldenTwo(t)
	techScenario(t, "two", "S162 two a lane with and without cover over several underwriters, 400 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		on := newInsTrack(t, g.cfg, seed, g.opts)
		msg := on.run(t, goldenTicks, nil)
		checkExact(t, rep, "S162", g.name, seed, "the books of the world with cover add up every tick", msg == "", "%s", msg)
		off := newInsTrack(t, withoutBlocks(g.cfg, false), seed, g.opts)
		msg = off.run(t, goldenTicks, nil)
		checkExact(t, rep, "S162", g.name, seed, "a world without a block forms no policy", msg == "" && off.tl.formed == 0, "%s, %d policies", msg, off.tl.formed)
		if on.tl.formed == 0 {
			t.Errorf("S162 seed %d: the world with cover formed no policy in %d ticks, want the comparison to be between cover and none", seed, goldenTicks)
		}
		rep.num(seed, "with cover: units landed", on.landed)
		rep.num(seed, "with cover: units lost", on.lostUnits)
		rep.num(seed, "with cover: premiums paid", on.premiums)
		rep.num(seed, "with cover: payouts paid at once", on.paid)
		rep.num(seed, "with cover: agents removed", on.removed)
		rep.num(seed, "without cover: units landed", off.landed)
		rep.num(seed, "without cover: units lost", off.lostUnits)
		rep.num(seed, "without cover: agents removed", off.removed)
	})
}
