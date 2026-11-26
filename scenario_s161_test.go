package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// payoutClaims tallies the payout claims of a run of events: the loans of origin payout that
// ended, and the principal they lost unpaid.
func payoutClaims(events []economysim.Event) (ended, lost int64) {
	for _, e := range events {
		if l, ok := e.(economysim.LoanEnded); ok && l.Origin == economysim.OriginPayout {
			ended++
			lost += int64(l.LostPrincipal)
		}
	}
	return ended, lost
}

// TestScenarioINS2_ShortfallClaim is INS-S2 (S161, spec-insurance.md 17, FR-INS-043 to 046, INS-A6):
// the route pack with an underwriter that cannot pay every payout it owes (a cook offers all its free
// capacity against a book of ten times its capital, cap_bp 100 000), seeds 1 to the scenario count, 400
// ticks, audit on. Exact: the money the debtors paid on payout claims at P1.7a never exceeds the
// shortfall the policies formed, the books of the module add up every tick, and the audit passes (a
// claim is a claim of the ledger, not money). Recorded: the shortfall, the claims paid and ended, the
// principal written off and the loans formed (the premium of an insured may be paid from a loan, A6).
//
// Requirement: FR-INS-043, FR-INS-044, FR-INS-045, FR-INS-046, FR-INS-006, FR-INS-079
//
// Guard: a shortfall paid twice, or a claim that paid more than it was formed for, would break the
// bound of the paid against the short at the tick it happens.
func TestScenarioINS2_ShortfallClaim(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	g := insGoldenTwo(t)
	techScenario(t, "two", "S161 two a payout the underwriter cannot pay in full is a claim, 400 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		tr := newInsTrack(t, g.cfg, seed, g.opts)
		msg := tr.run(t, goldenTicks, nil)
		checkExact(t, rep, "S161", g.name, seed, "the books of the module add up every tick", msg == "", "%s", msg)
		checkExact(t, rep, "S161", g.name, seed, "claims paid never exceed the shortfall formed", tr.claimPaid <= tr.short, "paid %d, short %d", tr.claimPaid, tr.short)
		ended, lost := payoutClaims(tr.rec.events)
		loans := int64(0)
		for _, e := range tr.rec.events {
			if _, ok := e.(economysim.LoanFormed); ok {
				loans++
			}
		}
		rep.num(seed, "shortfall formed", tr.short)
		rep.num(seed, "claims principal paid at P1.7a", tr.claimPaid)
		rep.num(seed, "claims ended repaid", tr.claimEnded)
		rep.num(seed, "payout claims ended", ended)
		rep.num(seed, "principal written off", lost)
		rep.num(seed, "loans formed", loans)
	})
}
