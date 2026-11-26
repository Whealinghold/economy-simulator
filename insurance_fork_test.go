package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// insForkTicks are the ticks of a bare run of a set at which the cover events a fork cell must
// straddle first happen, or -1 when the run holds none.
type insForkTicks struct {
	formed, payout, shortfall, claimPaid, removed int
}

// insForkProbe returns the first tick of each cover event of a bare run of g: a policy formed, a
// policy ended with a due above 0, one ended with a shortfall, a payout claim paid at P1.7a, and a
// policy ended by the removal of its insured.
func insForkProbe(t testing.TB, g insGolden, seed uint64, ticks int64) insForkTicks {
	t.Helper()
	res := runPack(t, g.cfg, runSpec{Seed: seed, Ticks: ticks, Drive: driveStep, Observe: true, Extra: g.opts})
	out := insForkTicks{-1, -1, -1, -1, -1}
	first := func(dst *int, tick int64) {
		if *dst < 0 {
			*dst = int(tick)
		}
	}
	for _, e := range res.Events {
		switch e := e.(type) {
		case economysim.CoverFormed:
			first(&out.formed, e.Tick)
		case economysim.CoverEnded:
			if e.Due > 0 {
				first(&out.payout, e.Tick)
			}
			if e.Short > 0 {
				first(&out.shortfall, e.Tick)
			}
			if e.Cause == economysim.CoverInsuredRemoved {
				first(&out.removed, e.Tick)
			}
		case economysim.CoverSettled:
			if e.Stage == economysim.CoverStageClaim && e.ClaimPaid > 0 {
				first(&out.claimPaid, e.Tick)
			}
		}
	}
	return out
}

// insForkInspect fails a cell whose restored world told no settlement after the boundary (a policy
// ended, or a payout claim paid) when want says it should: a cell that straddles nothing proves
// nothing.
func insForkInspect(want bool) func(t *testing.T, restored, since []economysim.Event) {
	return func(t *testing.T, restored, _ []economysim.Event) {
		t.Helper()
		settled := slices.ContainsFunc(restored, func(e economysim.Event) bool {
			switch e := e.(type) {
			case economysim.CoverEnded:
				return true
			case economysim.CoverSettled:
				return e.Stage == economysim.CoverStageClaim
			}
			return false
		})
		if want && !settled {
			t.Errorf("the restored world settled nothing after the boundary, want the cell to straddle a settlement")
		}
	}
}

// TestFork_Insurance verifies the fork-at-N of the two packs of the module-alone goldens
// (FR-INS-080, FR-INS-087; INS-S6): before the first step, after a formation with a route policy
// in flight, at an arrival that pays and the tick after it, after a shortfall claim formed (and
// at the tick its repayment is made), after a P8.1b payout of a store policy and at a removal of
// an insured, the original and two restores give the same events, snapshot, audit, checkpoint
// bytes and log records for 20 ticks, so a restored world settles, pays and removes as the
// uninterrupted one. Removal of an underwriter does not occur on the packs of the goldens in 400
// ticks of seeds 1 to 4; the unit tests of step 2f cover it.
//
// Requirement: FR-INS-080, FR-INS-087, FR-INS-006, FR-CHK-050
//
// Guard: the unit tests compare the restored state field by field; this one compares what the
// restored world does next, so a policy, a claim or a total the codec loses shows some ticks later.
func TestFork_Insurance(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	two, store := insGoldenTwo(t), insGoldenStore(t)
	rt, st := insForkProbe(t, two, 1, 40), insForkProbe(t, store, 1, goldenTicks)
	for _, p := range []struct {
		set  string
		tick int
	}{{"two formed", rt.formed}, {"two payout", rt.payout}, {"two shortfall", rt.shortfall}, {"two claim paid", rt.claimPaid}, {"store payout", st.payout}, {"store removal", st.removed}} {
		if p.tick < 0 {
			t.Fatalf("%s: no such event in the bare run, want the cell to straddle it", p.set)
		}
	}
	type cell struct {
		name string
		g    insGolden
		n    int
		want bool
	}
	for _, tc := range []cell{
		{"two before the first step", two, 0, false},
		{"two after a formation, a policy in flight", two, rt.formed + 1, true},
		{"two at an arrival that pays", two, rt.payout, true},
		{"two the tick after an arrival that pays", two, rt.payout + 1, true},
		{"two after a shortfall claim formed", two, rt.shortfall + 1, true},
		{"two at the repayment of a claim", two, rt.claimPaid, true},
		{"store before the first step", store, 0, false},
		{"store after a P8.1b payout", store, st.payout + 1, true},
		{"store at a removal of an insured", store, st.removed, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{
				Name: "insurance " + tc.name, Cfg: tc.g.cfg, Seed: 1, N: tc.n, M: 20, Logs: true,
				Opts:    func() []economysim.Option { return tc.g.opts },
				Inspect: insForkInspect(tc.want),
			})
		})
	}
}
