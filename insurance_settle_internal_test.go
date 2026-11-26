package economysim

import (
	"slices"
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
	if got := only(t, "CoverEnded at tick 2", coverEndedAt(rec, 2)); got != wantEnded {
		t.Errorf("CoverEnded = %+v, want %+v", got, wantEnded)
	}
	wantSettled := CoverSettled{Tick: 2, Region: regHome, Stage: CoverStageArrival, Ended: 1, Claimed: 1, LossValue: 2_000, Due: 1_600, Paid: 1_600}
	if got := only(t, "CoverSettled at tick 2", coverSettledAt(rec, 2)); got != wantSettled {
		t.Errorf("CoverSettled = %+v, want %+v", got, wantSettled)
	}
	if !inOrder[ShipmentArrived, CoverEnded, CoverSettled](rec) {
		t.Errorf("events are not in the order ShipmentArrived, CoverEnded, CoverSettled (spec-insurance.md 12.1.1)")
	}
	if got := moneyOfAgents(t, w, insHome, insBacker); !slices.Equal(got, []int64{10_000, 10_000}) {
		t.Errorf("money of the insured and the underwriter = %v, want 10000 each (premium 1600 paid, payout 1600 received)", got)
	}
	in := w.insurance
	if got := shape(w, insBacker, insHome); got != [4]int64{} || in.covered(0) {
		t.Errorf("after the end: lists %v, covered %v, want all empty", got, in.covered(0))
	}
	if want := (coverTotals{formed: 1, premiums: 1_600, due: 1_600, paid: 1_600}); in.tot != want {
		t.Errorf("totals = %+v, want %+v", in.tot, want)
	}
	// N_t: the premium of tick 0 is a cost of the insured and an income of the underwriter, the payout of tick 2 the reverse.
	for id, want := range map[AgentID][5]int64{insHome: {-1_600, 0, 1_600, -1_600, 1_600}, insBacker: {1_600, 0, -1_600, 1_600, -1_600}} {
		r := welfareRecords(w, id)
		if got := [5]int64{r[0].Unearned, r[1].Unearned, r[2].Unearned, r[0].Income, r[2].Income}; got != want {
			t.Errorf("agent %d: unearned of ticks 0 to 2 and income of ticks 0 and 2 = %v, want %v", id, got, want)
		}
	}
	auditNow(t, w)
}

// TestSettle_ShortfallFormsAPayoutClaim verifies the pot and the shield: an underwriter with
// 1 500 has a pot of 1 500 - 1 000 = 500, pays that, and the 1 100 left is a claim of origin
// payout of the insured on it, due one tick on. The identity of the due holds, the shield
// the position kept is told, and the claim is the ledger's, so the next pass of the credit
// module pays it.
//
// Requirement: FR-INS-043, FR-INS-044, FR-INS-045, FR-INS-006, FR-INS-064
func TestSettle_ShortfallFormsAPayoutClaim(t *testing.T) {
	t.Parallel()
	w, rec := settleWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), 8)
	step(t, w, 2)
	moneyOf(t, w, insBacker, 1_500)
	step(t, w, 1)

	e := only(t, "CoverEnded at tick 2", coverEndedAt(rec, 2))
	if got, want := figures(e), [5]Money{1_600, 500, 1_100, 0, 0}; got != want || e.Claim != 0 || e.Cause != CoverClaimed {
		t.Errorf("CoverEnded = %+v, want due 1600 = paid 500 + short 1100, claim 0 (figures %v)", e, want)
	}
	s := only(t, "CoverSettled at tick 2", coverSettledAt(rec, 2))
	if got, want := [3]Money{s.Short, s.Shielded, s.Paid}, [3]Money{1_100, 1_000, 500}; got != want {
		t.Errorf("CoverSettled short, shielded, paid = %v, want %v (shielded is min(1500, 1600) - 500)", got, want)
	}
	c := openClaim(t, w, 0)
	got := [5]int64{int64(c.Origin), int64(c.Creditor), int64(c.Debtor), c.PrincipalOwed, c.Due}
	if want := [5]int64{ledger.OriginPayout, int64(w.agents[insHome].account), int64(w.agents[insBacker].account), 1_100, 3}; got != want || c.Overdue != -1 {
		t.Errorf("claim = %v, want a payout claim of home on backer, 1100 owed, due at tick 3", got)
	}
	// N_t is in G_t, so it is out of the observed wage: the numerator is 0 whatever the income of the window.
	h := w.agents[insHome].history
	if num, _ := h.WageParts(); num != 0 || welfareRecords(w, insHome)[2].Income != 500 {
		t.Errorf("wage numerator = %d, last income %d, want 0 and 500: the premium and the payout are unearned", num, welfareRecords(w, insHome)[2].Income)
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

	end := only(t, "LoanEnded", of[LoanEnded](rec))
	wantEnd := LoanEnded{Tick: 3, Claim: 0, Origin: OriginPayout, Creditor: insHome, Debtor: insBacker, Region: regHome, CreditorRegion: regHome, Principal: 1_100, Cause: CauseRepaid}
	if end != wantEnd {
		t.Errorf("LoanEnded = %+v, want %+v", end, wantEnd)
	}
	want := CoverSettled{Tick: 3, Region: regHome, Stage: CoverStageClaim, ClaimEnded: 1, ClaimPaid: 1_100, ClaimReceived: 1_100}
	if got := only(t, "CoverSettled at tick 3", coverSettledAt(rec, 3)); got != want {
		t.Errorf("CoverSettled at tick 3 = %+v, want %+v", got, want)
	}
	if _, open := w.ledger.Claim(0); open {
		t.Errorf("the claim is still open after it was paid")
	}
	got := [2]int64{welfareRecords(w, insHome)[3].Unearned, welfareRecords(w, insBacker)[3].Unearned}
	if got != [2]int64{1_100, -1_100} {
		t.Errorf("unearned of tick 3: creditor and debtor %v, want +1100 and -1100 (the principal of a payout claim is N_t)", got)
	}
	if !inOrder[CoverEnded, LoanEnded, CoverSettled](rec) {
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
	if got, want := figures(byInsured[insHome]), [5]Money{1_600, 400, 1_200, 0, 0}; got != want {
		t.Errorf("home due, paid, short, dropped, cut = %v, want %v", got, want)
	}
	if got, want := figures(byInsured[insHome2]), [5]Money{2_400, 601, 1_799, 0, 0}; got != want {
		t.Errorf("home2 due, paid, short, dropped, cut = %v, want %v", got, want)
	}
	s := only(t, "CoverSettled at tick 2", coverSettledAt(rec, 2))
	if got, want := [3]int64{s.Ended, int64(s.Short), int64(s.Paid)}, [3]int64{2, 2_999, 1_001}; got != want {
		t.Errorf("CoverSettled ended, short, paid = %v, want %v (the pot is paid, 1001)", got, want)
	}
	auditNow(t, w)
}
