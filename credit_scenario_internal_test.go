package economysim

import (
	"bytes"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// claimsLifeEnd is the last tick of the claims-in-life scenario: every claim planted
// is closed well before it.
const claimsLifeEnd = 30

// claimsLife plants the portfolio of the scenario in a credit world at tick 1 and
// returns the claim IDs. Two weavers lend to millers and a miller owes a payout:
//
//	repaid:        loan weaver(home) -> homeA, 500 + 50, due 5, homeA is rich from 4
//	written off:   loan weaver(home) -> homeB, 300 + 30, due 8, homeB is never paid
//	                (WriteOffTicks 3 gives the loss at tick 11)
//	late, repaid:  loan weaver(yonder) -> yonderA, 400, due 6, yonderA is paid at 10
//	payout:        homeC is owed 200 by yonderB, due 10, yonderB is removed at tick 7
func claimsLife(t *testing.T, w *World) (repaid, writtenOff, late, payout int64) {
	t.Helper()
	setBalances(t, w, map[AgentID]int64{homeWeaver: 2_000, yonderWeaver: 1_000})
	repaid = plantLoan(t, w, homeWeaver, homeA, 500, 50, 5)
	writtenOff = plantLoan(t, w, homeWeaver, homeB, 300, 30, 8)
	late = plantLoan(t, w, yonderWeaver, yonderA, 400, 0, 6)
	payout = plantPayout(t, w, homeC, yonderB, 200, 10)
	return repaid, writtenOff, late, payout
}

// claimsLifeTick drives the tick n of the scenario: it sets the balances the story
// needs (homeA is rich from 4, homeB lives on 1 a tick, yonderA and yonderB are rich from 10), removes yonderB
// with money in hand at tick 7 and the home weaver, a creditor, at tick 10, and steps once.
func claimsLifeTick(t *testing.T, w *World, n int64) {
	t.Helper()
	bal := map[AgentID]int64{}
	if n >= 4 && n <= 6 {
		bal[homeA] = 5_000
	}
	if n >= 4 {
		bal[homeB] = 1 // enough to live on, not to pay: the loan goes into arrears
	}
	if n >= 10 {
		bal[yonderA], bal[yonderB] = 5_000, 5_000
	}
	if n == 8 {
		bal[yonderB] = 300
	}
	setBalances(t, w, bal)
	switch n {
	case 8:
		if err := w.removeAgent(&w.agents[yonderB], CauseConsumer); err != nil {
			t.Fatalf("removeAgent(yonderB) at tick %d error = %v, want nil", n, err)
		}
	case 11:
		if err := w.removeAgent(&w.agents[homeWeaver], CauseConsumer); err != nil {
			t.Fatalf("removeAgent(homeWeaver) at tick %d error = %v, want nil", n, err)
		}
	}
	step(t, w, 1)
}

// driveClaimsLife runs the scenario from the world's tick to claimsLifeEnd.
func driveClaimsLife(t *testing.T, w *World) {
	t.Helper()
	for w.Tick() < claimsLifeEnd {
		claimsLifeTick(t, w, w.Tick()+1)
	}
}

// TestScenario_CreditClaimsInLife verifies the claims of a credit world through a
// whole life, as a consumer sees it from the public calls and the events: a loan is
// repaid at its due tick, a loan of a debtor who cannot pay goes overdue and is
// written off, a late debtor pays what the pot allows once rich, a payout claim ends
// when its debtor is removed with money, and the claims of a removed creditor pass to
// the estate, which is paid and bears the write-off. Every tick the audit is on, the
// money over the accounts is unchanged, and at the end no claim is open and the
// totals of the ledger balance: lent and awarded equal repaid plus lost.
//
// Arithmetic (the tick of an event is the tick the position ran in): homeA owes 550
// and holds 5 000 from tick 4, so 500 of principal and 50 of interest are paid and
// the loan ends repaid. homeB owes 330 and holds 1 a tick: it pays 1 of interest
// at each of the four settlements (ticks 8 to 11, the interest first), so 326 are
// still owed at the write-off at 8 + 3 = 11, 300 of principal and 26 of interest,
// and the claim owes the estate by then, passed at the removal of the weaver at
// tick 10 with 328 owed. yonderA is overdue from 6 and holds 5 000 from 10, so it
// pays 400 and the loan ends repaid. yonderB holds 300 when removed at tick 7 and
// owes 200: the waterfall pays in full, the claim ends debtor_removed with nothing
// lost. Totals: 1 200 lent (500 + 300 + 400), 200 awarded, 54 of interest paid
// (50 + 4), 300 + 26 lost.
//
// Requirement: FR-CRD-030, FR-CRD-034, FR-CRD-035, FR-CRD-040, FR-CRD-042, FR-CRD-003, FR-CRD-006, FR-CRD-082, FR-CRD-071, FR-CRD-072
func TestScenario_CreditClaimsInLife(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	rec := &evLog{}
	w := creditWorld(t, 0, 3, WithObserver(rec), WithAudit())
	step(t, w, 1)
	repaid, writtenOff, late, payout := claimsLife(t, w)
	money := moneyTotal(w)
	for w.Tick() < claimsLifeEnd {
		claimsLifeTick(t, w, w.Tick()+1)
		if got := moneyTotal(w); got != money {
			t.Fatalf("money over the accounts = %d at tick %d, want %d unchanged", got, w.Tick(), money)
		}
	}

	if open := w.ledger.AppendClaims(nil); len(open) != 0 {
		t.Errorf("open claims at tick %d = %+v, want none", w.Tick(), open)
	}
	if err := w.credit.idx.check(nil); err != nil {
		t.Errorf("claim indexes at the end: %v, want empty", err)
	}
	checkClaimsLifeTotals(t, w)
	checkClaimsLifeEvents(t, rec, repaid, writtenOff, late, payout)
}

// checkClaimsLifeTotals verifies the money the claims of the scenario moved.
func checkClaimsLifeTotals(t *testing.T, w *World) {
	t.Helper()
	tot := w.ledger.ClaimTotals()
	if tot.Lent+tot.Awarded != tot.Repaid+tot.LostPrincipal {
		t.Errorf("ClaimTotals = %+v: lent plus awarded is %d, want repaid plus lost principal %d",
			tot, tot.Lent+tot.Awarded, tot.Repaid+tot.LostPrincipal)
	}
	if tot.Lent != 1_200 || tot.Awarded != 200 || tot.InterestPaid != 54 || tot.LostPrincipal != 300 || tot.LostInterest != 26 {
		t.Errorf("ClaimTotals = %+v, want 1200 lent, 200 awarded, 54 interest paid, 300 principal and 26 interest lost", tot)
	}
}

// checkClaimsLifeEvents verifies how each claim of the scenario ended and the one
// claim that passed to the estate.
func checkClaimsLifeEvents(t *testing.T, rec *evLog, repaid, writtenOff, late, payout int64) {
	t.Helper()
	want := map[int64]Cause{repaid: CauseRepaid, writtenOff: CauseWrittenOff, late: CauseRepaid, payout: CauseDebtorRemoved}
	ended := map[int64]LoanEnded{}
	for _, e := range of[LoanEnded](rec) {
		ended[int64(e.Claim)] = e
	}
	if len(ended) != len(want) {
		t.Errorf("LoanEnded events name %d claims, want %d", len(ended), len(want))
	}
	for id, cause := range want {
		if e, ok := ended[id]; !ok || e.Cause != cause {
			t.Errorf("claim %d ended with %+v, want the cause %q", id, e, cause)
		}
	}
	checkClaimsLifeLosses(t, ended[payout], ended[writtenOff])
	checkClaimsLifePassed(t, of[LoanPassed](rec), ClaimID(writtenOff))
}

// checkClaimsLifePassed verifies that the one claim that passed to the estate is
// the written-off one, from the removed weaver.
func checkClaimsLifePassed(t *testing.T, passed []LoanPassed, claim ClaimID) {
	t.Helper()
	if len(passed) != 1 || passed[0].Claim != claim || passed[0].Creditor != homeWeaver || passed[0].Tick != 10 || passed[0].Owed != 328 {
		t.Errorf("LoanPassed events = %+v, want one: the written-off claim, from the home weaver at tick 10, owing 328", passed)
	}
}

// checkClaimsLifeLosses verifies the payment of the removal and the loss of the
// write-off.
func checkClaimsLifeLosses(t *testing.T, payout, writtenOff LoanEnded) {
	t.Helper()
	if payout.PaidPrincipal != 200 || payout.LostPrincipal != 0 || payout.Tick != 7 {
		t.Errorf("payout LoanEnded = %+v, want 200 paid and nothing lost at tick 7", payout)
	}
	if !writtenOff.ToEstate || writtenOff.Tick != 11 || writtenOff.LostPrincipal != 300 || writtenOff.LostInterest != 26 {
		t.Errorf("written-off LoanEnded = %+v, want the estate as creditor, tick 11, 300 and 26 lost", writtenOff)
	}
}

// TestFork_CreditClaimsInLife verifies that a checkpoint taken in the middle of the
// scenario, with a claim repaid, others overdue or not yet due and the removals still
// to come, restores to a world that finishes with the bytes of the original, and
// that two fresh runs of the scenario agree: the claims, the indexes and the warn
// runs are all in the section (spec-credit.md 17) and nothing else moves a word.
//
// Requirement: FR-CRD-087, FR-CRD-088, FR-CRD-062
// Guard: a field of the credit state left out of tag 26 would survive in the original
// and be zero in the restore; the two runs would differ only after the claim that
// needs it, many ticks past the checkpoint.
func TestFork_CreditClaimsInLife(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	fresh := func() *World {
		w := creditWorld(t, 0, 3, WithAudit())
		step(t, w, 1)
		claimsLife(t, w)
		return w
	}
	a := fresh()
	for a.Tick() < 7 {
		claimsLifeTick(t, a, a.Tick()+1)
	}
	cp, err := a.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint at tick %d error = %v, want nil", a.Tick(), err)
	}
	b, err := RestoreWorld(a.Config(), cp, WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	driveClaimsLife(t, a)
	driveClaimsLife(t, b)
	c := fresh()
	driveClaimsLife(t, c)

	ca, _ := a.Checkpoint()
	cb, _ := b.Checkpoint()
	cc, _ := c.Checkpoint()
	if !bytes.Equal(ca.b, cb.b) {
		t.Errorf("checkpoint at tick %d: the restored run has %d bytes and differs from the original's %d", claimsLifeEnd, len(cb.b), len(ca.b))
	}
	if !bytes.Equal(ca.b, cc.b) {
		t.Errorf("checkpoint at tick %d: two runs of the same scenario differ", claimsLifeEnd)
	}
	if got := a.ledger.ClaimTotals(); got == (ledger.ClaimTotals{}) {
		t.Error("ClaimTotals are zero at the end, want the scenario's claims counted")
	}
}
