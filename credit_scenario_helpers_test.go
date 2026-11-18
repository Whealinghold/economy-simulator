package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// creditTally is an observer that keeps counters of the credit module and of the
// households' hardship, never the events themselves: S114 to S118 run ten seeds in
// parallel and keep no per-seed log (the lesson of S113, plan 3-crd gate economics).
type creditTally struct {
	formed, repaid, writtenOff, removed, forgiven int64
	principal                                     economysim.Money
	interestPaid, lostPrincipal, lostInterest     economysim.Money
	unfed, relief, unfilled, defaulted            int64
	roundTicks                                    []int64 // the ticks of a CreditRound with requests, in order
	weightFalls                                   []economysim.WeightChanged
	ended                                         []economysim.LoanEnded
	keepEnded                                     bool
}

// Observe adds an event to the counters.
func (c *creditTally) Observe(e economysim.Event) {
	switch ev := e.(type) {
	case economysim.LoanFormed:
		c.formed++
		c.principal += ev.Principal
	case economysim.LoanEnded:
		c.countEnded(ev)
	case economysim.CreditSettled:
		c.interestPaid += ev.InterestPaid
		c.defaulted += ev.Defaulted
	case economysim.CreditRound:
		c.unfilled += int64(ev.Unfilled)
		if ev.Requests > 0 {
			c.roundTicks = append(c.roundTicks, ev.Tick)
		}
	case economysim.UpkeepShortfall:
		c.unfed += ev.Unfed
	case economysim.ReliefPaid:
		c.relief += int64(ev.Paid)
	case economysim.WeightChanged:
		if ev.To < ev.From {
			c.weightFalls = append(c.weightFalls, ev)
		}
	}
}

// countEnded adds a LoanEnded to the counters of its cause.
func (c *creditTally) countEnded(ev economysim.LoanEnded) {
	switch ev.Cause {
	case economysim.CauseRepaid:
		c.repaid++
	case economysim.CauseWrittenOff:
		c.writtenOff++
	case economysim.CauseDebtorRemoved:
		c.removed++
	case economysim.CauseForgiven:
		c.forgiven++
	}
	c.lostPrincipal += ev.LostPrincipal
	c.lostInterest += ev.LostInterest
	if c.keepEnded {
		c.ended = append(c.ended, ev)
	}
}

// report is the one line of what a run did, for t.Logf.
func (c *creditTally) report() string {
	return fmt.Sprintf("formed %d (principal %d), repaid %d, written off %d, debtor removed %d, forgiven %d, interest paid %d, "+
		"defaulted %d, unfilled %d, unfed household-ticks %d, relief %d, rounds with requests %d",
		c.formed, c.principal, c.repaid, c.writtenOff, c.removed, c.forgiven, c.interestPaid,
		c.defaulted, c.unfilled, c.unfed, c.relief, len(c.roundTicks))
}

// checkClaimIdentities checks the two identities of spec-credit.md 3.6 item 1 on a
// snapshot, with exact equality: the principal owed over the open claims is the lent
// plus the awarded less the repaid less the principal lost, and the interest owed plus
// the interest paid plus the interest lost is the interest fixed. It also checks item 4
// (no empty claim, no overpaid claim) row by row.
func checkClaimIdentities(t testing.TB, where string, s economysim.Snapshot) {
	t.Helper()
	var owedP, owedI economysim.Money
	for _, c := range s.Claims {
		owedP += c.PrincipalOwed
		owedI += c.InterestOwed
		if emptyOrOverpaid(c) {
			t.Fatalf("%s: claim %+v is empty or overpaid, want owed within [0, fixed] and not both 0", where, c)
		}
	}
	tot := s.Credit
	if want := tot.Lent + tot.Awarded - tot.Repaid - tot.LostPrincipal; owedP != want {
		t.Fatalf("%s: principal owed over %d claims = %d, want lent + awarded - repaid - lost = %d", where, len(s.Claims), owedP, want)
	}
	if got := owedI + tot.InterestPaid + tot.LostInterest; got != tot.InterestFixed {
		t.Fatalf("%s: interest owed + paid + lost = %d, want the interest fixed %d", where, got, tot.InterestFixed)
	}
	if !slices.IsSortedFunc(s.Claims, func(a, b economysim.ClaimRow) int { return int(a.Claim) - int(b.Claim) }) {
		t.Fatalf("%s: claim rows are not in ascending claim ID", where)
	}
}

// stepAudited steps w n times; the world was built with WithAudit, so every step is
// audited, and fn (if any) runs after each with the tick just ended.
func stepAudited(t testing.TB, w *economysim.World, seed uint64, n int, fn func(tick int)) {
	t.Helper()
	for i := range n {
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step at tick %d error = %v, want nil (WithAudit)", seed, i, err)
		}
		if fn != nil {
			fn(i)
		}
	}
}

// creditWorld builds a world of cfg with the audit and the tally.
func creditWorld(t testing.TB, cfg economysim.Config, seed uint64, tally *creditTally) *economysim.World {
	t.Helper()
	return mustWorld(t, cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(tally))
}

// emptyOrOverpaid reports a claim row that breaks item 4 of spec-credit.md 3.6: an owed
// outside [0, fixed], or no principal and no interest owed.
func emptyOrOverpaid(c economysim.ClaimRow) bool {
	return c.PrincipalOwed < 0 || c.PrincipalOwed > c.Principal || c.InterestOwed < 0 || c.InterestOwed > c.Interest ||
		(c.PrincipalOwed == 0 && c.InterestOwed == 0)
}
