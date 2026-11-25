package economysim

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file tests the settlement of a route policy at P1.3a (plan 3-ins Phase 6,
// spec-insurance.md 8.1 to 8.5) over the world of insConfig: a coach shipment of 8 grain
// is struck at the certain route row, which takes 25% of the cargo, and arrives at tick 2.
//
// Worked (the numbers of every test below unless it says otherwise): 8 grain at a unit
// value of 1 000 is a sum of 8 000; the policy's maximum payout at a cover share of 8 000 bp
// is 6 400 and its premium 1 600. The strike takes rhe(8 * 2 500 / 10 000) = 2 units, so the
// loss value is 2 000 and the due is halfEven(8 000 * 2 000 / 10 000) = 1 600, under the
// maximum. The upkeep cost of a household is 1 000 a tick and the shield is one tick of it.

// TestSettle_RoutePaidInFull verifies P1.3a for an underwriter whose pot covers the due:
// the payout moves to the insured with the reason payout, the policy ends claimed and leaves
// every index, the totals and the two events carry the figures, and N_t is in the record of
// each party at its tick.
//
// Requirement: FR-INS-040, FR-INS-041, FR-INS-042, FR-INS-043, FR-INS-064, FR-INS-070, FR-INS-071, FR-INS-072
func TestSettle_RoutePaidInFull(t *testing.T) {
	t.Parallel()
	w, rec := settleWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), 8)
	step(t, w, 3)

	wantEnded := CoverEnded{Tick: 2, Policy: 0, Kind: CoverRoute, Insured: insHome, Underwriter: insBacker, Region: regHome, UnderwriterRegion: regHome,
		Cause: CoverClaimed, LossValue: 2_000, Due: 1_600, Paid: 1_600, Claim: -1}
	if got := coverEndedAt(rec, 2); len(got) != 1 || got[0] != wantEnded {
		t.Errorf("CoverEnded at tick 2 = %+v, want one %+v", got, wantEnded)
	}
	wantSettled := CoverSettled{Tick: 2, Region: regHome, Stage: CoverStageArrival, Ended: 1, Claimed: 1, LossValue: 2_000, Due: 1_600, Paid: 1_600}
	if got := coverSettledAt(rec, 2); len(got) != 1 || got[0] != wantSettled {
		t.Errorf("CoverSettled at tick 2 = %+v, want one %+v", got, wantSettled)
	}
	if home, under := heldMoney(t, w, insHome), heldMoney(t, w, insBacker); home != 10_000 || under != 10_000 {
		t.Errorf("money after the payout: insured %d underwriter %d, want 10000 each (premium 1600 paid, payout 1600 received)", home, under)
	}
	in := w.insurance
	if len(in.open) != 0 || in.covered(0) || in.liabilityOf(insBacker) != 0 || in.policiesOf(insHome) != 0 {
		t.Errorf("after the end: open %d, covered %v, liability %d, policies %d, want all empty", len(in.open), in.covered(0), in.liabilityOf(insBacker), in.policiesOf(insHome))
	}
	if want := (coverTotals{formed: 1, premiums: 1_600, due: 1_600, paid: 1_600}); in.tot != want {
		t.Errorf("totals = %+v, want %+v", in.tot, want)
	}
	// N_t: the premium of tick 0 is a cost of the insured and an income of the underwriter, the payout of tick 2 the reverse.
	for _, c := range []struct {
		id   AgentID
		want [3]int64
	}{{insHome, [3]int64{-1_600, 0, 1_600}}, {insBacker, [3]int64{1_600, 0, -1_600}}} {
		recs := welfareRecords(w, c.id)
		if got := [3]int64{recs[0].Unearned, recs[1].Unearned, recs[2].Unearned}; got != c.want || recs[0].Income != c.want[0] || recs[2].Income != c.want[2] {
			t.Errorf("agent %d: unearned of ticks 0 to 2 = %v (income %d, %d), want %v and the same income", c.id, got, recs[0].Income, recs[2].Income, c.want)
		}
	}
	auditNow(t, w)
}

// TestSettle_ShortfallFormsAPayoutClaim verifies the pot and the shield: an underwriter with
// 1 500 has a pot of 1 500 - 1 000 = 500, pays that, and the 1 100 left is a claim of origin
// payout of the insured on it, due one tick on. The identity of the due holds, the shield
// the position kept is told, and the claim is the ledger's, so the next pass of the credit
// module pays it and both parties' record has the principal.
//
// Requirement: FR-INS-043, FR-INS-044, FR-INS-045, FR-INS-006, FR-INS-064
func TestSettle_ShortfallFormsAPayoutClaim(t *testing.T) {
	t.Parallel()
	w, rec := settleWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), 8)
	step(t, w, 2)
	moneyOf(t, w, insBacker, 1_500)
	step(t, w, 1)

	ended := coverEndedAt(rec, 2)
	if len(ended) != 1 {
		t.Fatalf("CoverEnded at tick 2 = %+v, want one", ended)
	}
	if e := ended[0]; e.Due != 1_600 || e.Paid != 500 || e.Short != 1_100 || e.Claim != 0 || e.Dropped != 0 || e.Cut != 0 || e.Cause != CoverClaimed {
		t.Errorf("CoverEnded = %+v, want due 1600 = paid 500 + short 1100, claim 0", e)
	}
	if got := coverSettledAt(rec, 2); len(got) != 1 || got[0].Short != 1_100 || got[0].Shielded != 1_000 || got[0].Paid != 500 {
		t.Errorf("CoverSettled = %+v, want short 1100, shielded min(1500, 1600) - 500 = 1000, paid 500", got)
	}
	c := openClaim(t, w, 0)
	if c.Origin != ledger.OriginPayout || c.Creditor != w.agents[insHome].account || c.Debtor != w.agents[insBacker].account || c.PrincipalOwed != 1_100 || c.Due != 3 || c.Overdue != -1 {
		t.Errorf("claim = %+v, want a payout claim of home on backer, 1100 owed, due at tick 3", c)
	}
	auditNow(t, w)
}

// TestClaim_SettledByCreditUnchanged verifies that the payout claim is an ordinary claim of
// the credit list (spec-insurance.md 8.4.1): at P1.7a of its due tick the debtor, funded
// again, pays its principal from the pot, the claim ends repaid with a LoanEnded of origin
// payout, CoverSettled of the stage claim counts it after it, and the principal enters N_t
// of both parties in the record of tick 3.
//
// Requirement: FR-INS-004, FR-INS-045, FR-INS-064, FR-INS-072, FR-INS-079
func TestClaim_SettledByCreditUnchanged(t *testing.T) {
	t.Parallel()
	w, rec := settleWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), 8)
	step(t, w, 2)
	moneyOf(t, w, insBacker, 1_500)
	step(t, w, 1)
	moneyOf(t, w, insBacker, 5_000) // pot 5 000 - 1 000 shield = 4 000, above the 1 100 owed
	step(t, w, 1)

	ends := of[LoanEnded](rec)
	if len(ends) != 1 || ends[0].Origin != OriginPayout || ends[0].Cause != CauseRepaid || ends[0].Principal != 1_100 || ends[0].Creditor != insHome || ends[0].Debtor != insBacker {
		t.Errorf("LoanEnded = %+v, want one repaid claim of origin payout, principal 1100, home creditor", ends)
	}
	claim := coverSettledAt(rec, 3)
	want := CoverSettled{Tick: 3, Region: regHome, Stage: CoverStageClaim, ClaimEnded: 1, ClaimPaid: 1_100, ClaimReceived: 1_100}
	if len(claim) != 1 || claim[0] != want {
		t.Errorf("CoverSettled at tick 3 = %+v, want one %+v", claim, want)
	}
	if _, open := w.ledger.Claim(0); open {
		t.Errorf("the claim is still open after it was paid")
	}
	if home, under := welfareRecords(w, insHome)[3].Unearned, welfareRecords(w, insBacker)[3].Unearned; home != 1_100 || under != -1_100 {
		t.Errorf("unearned of tick 3: creditor %d, debtor %d, want +1100 and -1100 (the principal of a payout claim is N_t)", home, under)
	}
	if order := indexOfEvent[LoanEnded](rec) - indexOfEvent[CoverSettled](rec); order >= 0 {
		t.Errorf("CoverSettled of the claim stage is not after LoanEnded in the event order")
	}
	auditNow(t, w)
}

// indexOfEvent returns the index in the log of the last event of type T, -1 when none.
func indexOfEvent[T Event](rec *evLog) int {
	at := -1
	for i, e := range rec.events {
		if _, ok := e.(T); ok {
			at = i
		}
	}
	return at
}

// TestSettle_ProRataWithTheLargestRemainder verifies the pass of one underwriter with two
// policies at one position: the dues 1 600 and 2 400 (8 and 12 grain) share the pot pro rata
// by due with the largest remainder, and the shortfalls are claims.
//
// Worked: the underwriter holds 2 001 at the entry, so the pot is 2 001 - 1 000 = 1 001 and
// the owed 4 000. The exact shares are 1 001 * 1 600 / 4 000 = 400.4 and 1 001 * 2 400 /
// 4 000 = 600.6; the floors 400 and 600 leave 1, which goes to the larger remainder, so the
// shares are 400 and 601 and the shortfalls 1 200 and 1 799.
//
// Requirement: FR-INS-043, FR-INS-044
func TestSettle_ProRataWithTheLargestRemainder(t *testing.T) {
	t.Parallel()
	w, rec := settleWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), 8, 12)
	step(t, w, 2)
	moneyOf(t, w, insBacker, 2_001)
	step(t, w, 1)

	byInsured := map[AgentID]CoverEnded{}
	for _, e := range coverEndedAt(rec, 2) {
		byInsured[e.Insured] = e
	}
	home, home2 := byInsured[insHome], byInsured[insHome2]
	if home.Due != 1_600 || home.Paid != 400 || home.Short != 1_200 || home2.Due != 2_400 || home2.Paid != 601 || home2.Short != 1_799 {
		t.Errorf("home %+v, home2 %+v, want dues 1600 and 2400 paid 400 and 601, short 1200 and 1799", home, home2)
	}
	if got := coverSettledAt(rec, 2); len(got) != 1 || got[0].Ended != 2 || got[0].Short != 2_999 || got[0].Paid != 1_001 {
		t.Errorf("CoverSettled = %+v, want two ended, paid 1001 (the pot), short 2999", got)
	}
	auditNow(t, w)
}
