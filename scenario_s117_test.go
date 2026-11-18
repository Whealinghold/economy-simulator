package economysim_test

import (
	"fmt"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// creditCohortScale is the Scale of the cohort world of S117: the households of every
// group are held in agents of up to four, so a borrower is a cohort whose weight the
// failure ladder can cut.
const creditCohortScale = 4

// TestScenarioS117_CohortBorrower verifies CRD-S4 of spec-credit.md 18.3 and 9.3: the
// credit pack with World.Scale 4, so a borrower is a cohort of up to four households,
// seeds 1 to the seed count, 200 ticks, the audit on. A fall of a debtor's weight by the
// failure ladder leaves every claim of the agent whole (9.3): a claim is not split and the
// debt per household rises. Exact: at every step each claim that was open before the step
// and is open after it keeps its principal, its interest and its debtor (the cause of an end
// is not asserted).
// Reported: the falls of weight of a debtor with an open claim, the debt per household
// before and after the first of them, and the claims ended by the waterfall.
//
// Requirement: FR-CRD-044, FR-CRD-043, FR-CRD-083
func TestScenarioS117_CohortBorrower(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	cfg := loadValidPack(t, packOneCredit...)
	cfg.World.Scale = creditCohortScale
	for seed := uint64(1); seed <= uint64(scenarioSeedCount()); seed++ {
		t.Run("seed "+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			tally := &creditTally{}
			w := creditWorld(t, cfg, seed, tally)
			cohort := &creditCohort{prev: map[economysim.ClaimID]economysim.ClaimRow{}}
			stepAudited(t, w, seed, creditScenarioTicks, func(tick int) {
				snap, err := w.Snapshot()
				if err != nil {
					t.Fatalf("seed %d tick %d: Snapshot() error = %v, want nil", seed, tick, err)
				}
				cohort.step(t, seed, tick, tally, snap.Claims)
			})
			t.Logf("seed %d: %s; weight falls of a debtor with open claims %d; claims ended by the waterfall %d; %s",
				seed, tally.report(), cohort.falls, tally.removed, cohort.first)
		})
	}
}

// creditCohort is the state of S117 across the steps of one seed: the claim rows of the
// step before, the weight falls already looked at, how many hit a debtor with open
// claims and the debt per household around the first of them.
type creditCohort struct {
	prev  map[economysim.ClaimID]economysim.ClaimRow
	seen  int
	falls int
	first string
}

// step checks the claims of one boundary against the step before and looks at the falls
// of weight the step made.
func (c *creditCohort) step(t *testing.T, seed uint64, tick int, tally *creditTally, rows []economysim.ClaimRow) {
	t.Helper()
	owed := map[economysim.AgentID]economysim.Money{}
	for _, p := range c.prev {
		owed[p.Debtor] += p.PrincipalOwed + p.InterestOwed
	}
	for _, f := range tally.weightFalls[c.seen:] {
		if before, ok := owed[f.Agent]; ok && f.From > 0 && f.To > 0 { // a weight of 0 is a removal, not a fall
			c.falls++
			if c.first == "" {
				c.first = fmt.Sprintf("first at tick %d: agent %d weight %d to %d, owed %d before the fall: %d per household before it and %d after it, the claims whole (9.3)",
					f.Tick, f.Agent, f.From, f.To, before, int64(before)/f.From, int64(before)/f.To)
			}
		}
	}
	c.seen = len(tally.weightFalls)
	c.prev = checkClaimsWhole(t, seed, tick, c.prev, rows)
}

// checkClaimsWhole checks that every claim open in prev and open in rows keeps its
// principal, interest and debtor (its creditor may change: a removed creditor's claims
// pass to its estate, 9.1), and returns rows indexed by claim ID for the next step.
func checkClaimsWhole(t *testing.T, seed uint64, tick int, prev map[economysim.ClaimID]economysim.ClaimRow, rows []economysim.ClaimRow) map[economysim.ClaimID]economysim.ClaimRow {
	t.Helper()
	next := make(map[economysim.ClaimID]economysim.ClaimRow, len(rows))
	for _, c := range rows {
		next[c.Claim] = c
		if p, ok := prev[c.Claim]; ok && (p.Principal != c.Principal || p.Interest != c.Interest || p.Debtor != c.Debtor) {
			t.Fatalf("seed %d tick %d: claim %d was %+v and is %+v, want principal, interest and debtor unchanged", seed, tick, c.Claim, p, c)
		}
	}
	return next
}
