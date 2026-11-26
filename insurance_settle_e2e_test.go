package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file is the end-to-end test of the settlement through the public API alone (plan
// 3-ins Phase 6; FR-INS-006, FR-INS-040 to 046, FR-INS-070 to 072, NFR-INS-003): the shipper
// pack of insurance_e2e_test.go run long enough for strikes, arrivals and payouts, under the
// audit that runs every tick (the identity of money, FR-INS-002). The settlement is told by
// CoverEnded and CoverSettled.

// settleSums are the figures of CoverEnded events added up.
type settleSums struct {
	ended, claimed                       int64
	loss, due, paid, short, dropped, cut economysim.Money
}

// coverEndedSums adds the CoverEnded events of a run by (tick, region), checking the
// identity of each event as it goes.
func coverEndedSums(t *testing.T, events []economysim.Event, seed uint64) map[[2]int64]settleSums {
	t.Helper()
	sums := map[[2]int64]settleSums{}
	for _, e := range events {
		c, ok := e.(economysim.CoverEnded)
		if !ok {
			continue
		}
		if c.Due != c.Paid+c.Short+c.Dropped+c.Cut {
			t.Errorf("seed %d policy %d: due %d != paid %d + short %d + dropped %d + cut %d", seed, c.Policy, c.Due, c.Paid, c.Short, c.Dropped, c.Cut)
		}
		if (c.Short > 0) != (c.Claim >= 0) {
			t.Errorf("seed %d policy %d: short %d with claim %d, want a claim exactly when there is a shortfall", seed, c.Policy, c.Short, c.Claim)
		}
		k := [2]int64{c.Tick, int64(c.Region)}
		s := sums[k]
		s.ended++
		if c.Cause == economysim.CoverClaimed {
			s.claimed++
		}
		s.loss, s.due, s.paid, s.short = s.loss+c.LossValue, s.due+c.Due, s.paid+c.Paid, s.short+c.Short
		s.dropped, s.cut = s.dropped+c.Dropped, s.cut+c.Cut
		sums[k] = s
	}
	return sums
}

// TestInvariants_Insurance_ClaimIdentity verifies over seeds with the audit on that the due
// of every policy is the sum of its parts (FR-INS-006, spec-insurance.md 9.4 item 1), a claim
// is formed exactly for a shortfall, each CoverSettled of the arrival stage is the total of
// the CoverEnded events of its tick and region, and the run pays at least one policy (a probe
// that fails an empty run).
//
// Requirement: FR-INS-006, FR-INS-041, FR-INS-042, FR-INS-043, FR-INS-071, FR-INS-072
func TestInvariants_Insurance_ClaimIdentity(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("multi-seed end-to-end test, skipped with -short")
	}
	paid := false
	for _, seed := range seedList(invariantSeeds(4)) {
		events := coverRun(t, coverKnobsDefault(), seed, 40)
		sums := coverEndedSums(t, events, seed)
		for _, e := range events {
			s, ok := e.(economysim.CoverSettled)
			if !ok || s.Stage != economysim.CoverStageArrival {
				continue
			}
			want := sums[[2]int64{s.Tick, int64(s.Region)}]
			got := settleSums{s.Ended, s.Claimed, s.LossValue, s.Due, s.Paid, s.Short, s.Dropped, s.Cut}
			if got != want {
				t.Errorf("seed %d tick %d region %d: CoverSettled %+v, want the totals of its CoverEnded events %+v", seed, s.Tick, s.Region, got, want)
			}
			paid = paid || s.Paid > 0
		}
	}
	if !paid {
		t.Error("no CoverSettled with a payout in any seed: the probe of the settlement has nothing to check")
	}
}
