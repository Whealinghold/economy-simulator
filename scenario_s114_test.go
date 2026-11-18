package economysim_test

import (
	"slices"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// creditScenarioTicks is the length of the credit scenarios: the first loans form at
// about tick 58 and the main wave of 202 claims falls due at about tick 123, so 200
// ticks run the whole cycle of the 60-tick job once and a half.
const creditScenarioTicks = 200

// packOneColdCredit is pack one with no warm start and the credit module on.
var packOneColdCredit = append(slices.Clone(packOneCold), "overlay/credit.toml")

// TestScenarioS114_ColdAndWarm verifies CRD-S1 of spec-credit.md 18.3: pack one, the
// role with the 60-tick job a borrower and the cooks lenders (overlay/credit.toml), cold
// and warm, with credit on and off, seeds 1 to the seed count, 200 ticks, the audit on.
// Exact: every step passes the audit; with the module off no claim, no credit event and
// no credit total exists; with it on the identities of spec-credit.md 3.6 hold at the end
// and the principal the events say was lent is the total the snapshot says. Reported, not
// compared (a paired comparison is not valid, the module changes the tie-breaks): unfed
// household-ticks, relief paid, claims formed and ended by cause, interest, defaults, the
// principal unfilled, and the ticks of the first rounds against the job cycle (the pulse of
// the cold start, STB-L167).
//
// Requirement: FR-CRD-083, FR-CRD-006, FR-CRD-001
func TestScenarioS114_ColdAndWarm(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	for _, v := range []struct {
		name   string
		layers []string
		on     bool
	}{
		{"cold on", packOneColdCredit, true}, {"cold off", packOneCold, false},
		{"warm on", packOneCredit, true}, {"warm off", packOne, false},
	} {
		cfg := loadValidPack(t, v.layers...)
		for seed := uint64(1); seed <= uint64(scenarioSeedCount()); seed++ {
			t.Run(v.name+" seed "+strconv.FormatUint(seed, 10), func(t *testing.T) {
				t.Parallel()
				tally := &creditTally{}
				w := creditWorld(t, cfg, seed, tally)
				stepAudited(t, w, seed, creditScenarioTicks, nil)
				snap, err := w.Snapshot()
				if err != nil {
					t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
				}
				checkScenarioS114(t, seed, v.on, tally, snap)
				t.Logf("seed %d %s: %s; first rounds at ticks %v", seed, v.name, tally.report(), tally.roundTicks[:min(5, len(tally.roundTicks))])
			})
		}
	}
}

// checkScenarioS114 asserts the exact half of S114 on one run.
func checkScenarioS114(t *testing.T, seed uint64, on bool, tally *creditTally, snap economysim.Snapshot) {
	t.Helper()
	if !on {
		if tally.formed != 0 || len(snap.Claims) != 0 || snap.Credit != (economysim.CreditTotals{}) || slices.Contains(snap.Modules, "credit") {
			t.Errorf("seed %d: with the module off %d claims formed, %d rows, totals %+v, modules %v, want none", seed, tally.formed, len(snap.Claims), snap.Credit, snap.Modules)
		}
		return
	}
	checkClaimIdentities(t, "S114 seed "+strconv.FormatUint(seed, 10), snap)
	if economysim.Money(tally.principal) != snap.Credit.Lent {
		t.Errorf("seed %d: LoanFormed principal sums to %d, Snapshot().Credit.Lent = %d, want equal", seed, tally.principal, snap.Credit.Lent)
	}
	if tally.formed == 0 {
		t.Errorf("seed %d: no loan formed in %d ticks, want the growers to borrow for their 60-tick job", seed, creditScenarioTicks)
	}
}
