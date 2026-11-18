package economysim_test

import (
	"strconv"
	"testing"
)

// TestScenarioS116_Invariants verifies CRD-S3 of spec-credit.md 18.3 over the credit pack,
// seeds 1 to the seed count, 200 ticks, WithAudit: after every tick the total of money
// is the one the world started with (pack one has no faucet or sink after construction,
// so a loan, a payment, a write-off, a forgiven claim and a waterfall move none: FR-CRD-003,
// FR-CRD-006), the pair identity of spec-credit.md 3.6 holds with exact equality, no claim
// is empty or overpaid and the rows are in ascending claim ID. The audit of the world runs
// at every step as well and checks that no claim outlives an account (FR-CRD-082 to 086).
//
// Requirement: FR-CRD-083, FR-CRD-084, FR-CRD-085, FR-CRD-086, FR-CRD-003, FR-CRD-006
func TestScenarioS116_Invariants(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	cfg := loadValidPack(t, packOneCredit...)
	for seed := uint64(1); seed <= uint64(scenarioSeedCount()); seed++ {
		t.Run("seed "+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			tally := &creditTally{}
			w := creditWorld(t, cfg, seed, tally)
			start, err := w.Snapshot()
			if err != nil {
				t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
			}
			stepAudited(t, w, seed, creditScenarioTicks, func(tick int) {
				s, err := w.Snapshot()
				if err != nil {
					t.Fatalf("seed %d tick %d: Snapshot() error = %v, want nil", seed, tick, err)
				}
				if s.Money != start.Money {
					t.Fatalf("seed %d tick %d: total money = %d, want the starting %d (no faucet: a claim operation creates or destroys none)", seed, tick, s.Money, start.Money)
				}
				checkClaimIdentities(t, "S116 seed "+strconv.FormatUint(seed, 10)+" tick "+strconv.Itoa(tick), s)
			})
			if tally.formed == 0 || tally.repaid == 0 {
				t.Errorf("seed %d: %d loans formed and %d repaid in %d ticks, want both above 0 so the identities were held over claims in life", seed, tally.formed, tally.repaid, creditScenarioTicks)
			}
		})
	}
}
