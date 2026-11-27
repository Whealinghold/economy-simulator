package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// claimSettled says an event is the payment of a payout claim at P1.7a or the end of one (repaid, written off or waterfalled).
func claimSettled(e economysim.Event) bool {
	switch e := e.(type) {
	case economysim.CoverSettled:
		return e.Stage == economysim.CoverStageClaim
	case economysim.LoanEnded:
		return e.Origin == economysim.OriginPayout
	}
	return false
}

// TestScenarioINS6_CheckpointWithPoliciesAndClaimOpen is INS-S6 (S165, spec-insurance.md 17, FR-INS-080,
// FR-INS-087): on the route pack, for each seed of 1 to the scenario count that forms a shortfall,
// two boundaries are taken, one tick after the first formation (the route policies are open and
// their shipments on the road) and one tick after the first shortfall (a payout claim is open), and
// at each the original and two restores give the same events, snapshot, audit, checkpoint bytes
// and log records for 20 ticks. Exact: the first restored world holds the open policies at the first
// boundary, and pays or ends the payout claim after the second.
//
// Requirement: FR-INS-080, FR-INS-087, FR-INS-006, FR-CHK-050
//
// Guard: the policies and the totals are the module's saved state and the claim is the ledger's; a
// restore that dropped one would settle a payout differently, or twice, some ticks later.
func TestScenarioINS6_CheckpointWithPoliciesAndClaimOpen(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	g := insGoldenTwo(t)
	probed := 0
	for _, seed := range seedList(scenarioSeedCount()) {
		p := insForkProbe(t, g, seed, 40)
		if p.shortfall < 0 {
			continue // a seed with no shortfall has no payout claim to open
		}
		probed++
		for _, cell := range []struct {
			name   string
			n      int
			policy bool
		}{{"policies open", p.formed + 1, true}, {"claim open", p.shortfall + 1, false}} {
			t.Run(fmt.Sprintf("seed %d %s", seed, cell.name), func(t *testing.T) {
				t.Parallel()
				var open int
				forkAtN(t, forkCase{
					Name: "insurance " + cell.name, Cfg: g.cfg, Seed: seed, N: cell.n, M: 20, Logs: true,
					Opts: func() []economysim.Option { return g.opts },
					Before: func(w *economysim.World) {
						s, err := w.Snapshot()
						if err != nil {
							t.Fatalf("Snapshot() error = %v, want nil", err)
						}
						open = len(s.Policies)
					},
					Inspect: func(t *testing.T, restored, _ []economysim.Event) {
						if cell.policy && open == 0 {
							t.Errorf("seed %d: no route policy open at the boundary, want the policies of the shipments on the road", seed)
						}
						if !cell.policy && !slices.ContainsFunc(restored, claimSettled) {
							t.Errorf("seed %d: the restored world paid or ended no payout claim after the boundary, want the open claim to be settled or written off", seed)
						}
					},
				})
			})
		}
	}
	if probed == 0 {
		t.Fatalf("no seed formed a shortfall in 40 ticks, want at least one so the test does not pass for nothing")
	}
}
