package economysim

import (
	"bytes"
	"fmt"
	"testing"
)

// This file closes two gaps of the pyramid of the insurance module found at PR F (N-554 items 1
// and 5): the premium of an insured that holds no money and pays it from the loan the credit
// round forms in the same tick (N-212, INS-A6), and a restore that straddles the removal of an
// underwriter (no golden pack removes one).

// TestCoverRound_PremiumPaidFromALoanOfTheSameTick verifies that an insured with no money, whose
// plan asks for a loan and for cover, gets the loan at P3.3b and pays the premium from it at
// P3.3c: the loan is formed before the policy, the premium is moved from the borrowed money, and
// what stays with the insured is the principal less the premium.
//
// Requirement: FR-INS-023, FR-INS-036
//
// Guard: a cover round that read the money of the start of the tick would refuse the premium
// and the insured would hold the loan unspent.
func TestCoverRound_PremiumPaidFromALoanOfTheSameTick(t *testing.T) {
	t.Parallel()
	plan := func(v *AgentView, p *TradePlan) {
		if v.Tick.Tick != 0 {
			return
		}
		switch v.Agent {
		case insHome:
			p.Borrow = append(p.Borrow, CreditRequest{Principal: 2_000, Term: 30, MaxRateBP: 1_000})
			p.Cover = append(p.Cover, coachLane)
		case insBacker:
			p.Offer = append(p.Offer, bigOffer)
			p.Lend = append(p.Lend, CreditOffer{Principal: 5_000, MaxTerm: 90, RateBP: 600})
		}
	}
	c := insConfig(defaultInsure(), defaultUnderwrite())
	c.Roles[0].Credit = &CreditTerms{Borrow: &BorrowTerms{MaxRateBP: 1_500, MaxDebtTicks: 90, SlackTicks: 4}}
	c.Roles[1].Credit = &CreditTerms{Lend: &LendTerms{RateBP: 600, MaxTermTicks: 90, KeepTicks: 3, ShareBP: 5_000}}
	w, rec := newInsWorld(t, c, plan)
	insDispatch(t, w, insHome, 8)
	moneyOf(t, w, insHome, 0)
	moneyOf(t, w, insBacker, 20_000)
	step(t, w, 1)

	loan := only(t, "LoanFormed", formedOf(rec))
	pol := only(t, "CoverFormed", of[CoverFormed](rec))
	if loan.Debtor != insHome || pol.Insured != insHome {
		t.Fatalf("loan debtor %d, policy insured %d, want both %d", loan.Debtor, pol.Insured, insHome)
	}
	if a, b := indexOfEvent[LoanFormed](rec), indexOfEvent[CoverFormed](rec); a >= b {
		t.Errorf("LoanFormed at %d, CoverFormed at %d, want the loan first", a, b)
	}
	if pol.Premium <= 0 || pol.Premium > loan.Principal {
		t.Fatalf("premium %d, loan %d, want 0 < premium <= principal", pol.Premium, loan.Principal)
	}
	if got, want := heldMoney(t, w, insHome), int64(loan.Principal-pol.Premium); got != want {
		t.Errorf("insured's money = %d, want principal less premium %d", got, want)
	}
	auditNow(t, w)
}

// forkAfterUnderwriterRemoval runs a world of the fork test, removes the underwriter at tick 1
// either before the save (the checkpoint holds the ended policies) or after the restore (the
// restored world removes it), and requires the two worlds to agree to the end.
func forkAfterUnderwriterRemoval(t *testing.T, removeBeforeSave bool) {
	t.Helper()
	a := &forkRun{}
	a.w, a.rec = settleWorld(t, insCkConfig(), 8, 12)
	step(t, a.w, 1)
	if removeBeforeSave {
		remove(t, a.w, insBacker)
	}
	a.since = len(a.rec.events)
	ck, err := a.w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint error = %v, want nil", err)
	}
	b := &forkRun{rec: &evLog{}}
	b.w, err = RestoreWorld(insCkConfig(), ck, WithStrategy("script", &insScript{}), WithAudit(), WithObserver(b.rec))
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	if !removeBeforeSave {
		remove(t, a.w, insBacker)
		remove(t, b.w, insBacker)
	}
	for a.w.Tick() < forkEnd {
		step(t, a.w, 1)
		step(t, b.w, 1)
		forkRequireSame(t, 1, a, b)
	}
	forkRequireRemoved(t, a, b)
}

// forkRequireRemoved fails the test unless the run ended the two policies underwriter_removed,
// left no policy or liability, and the two worlds write the same final checkpoint.
func forkRequireRemoved(t *testing.T, a, b *forkRun) {
	t.Helper()
	ends := of[CoverEnded](a.rec)
	if len(ends) != 2 || ends[0].Cause != CoverUnderwriterRemoved || ends[1].Cause != CoverUnderwriterRemoved {
		t.Errorf("CoverEnded = %+v, want the two policies ended underwriter_removed", ends)
	}
	if n := len(a.w.insurance.open); n != 0 || a.w.insurance.liabilityOf(insBacker) != 0 {
		t.Errorf("after the removal: open %d, liability %d, want none", n, a.w.insurance.liabilityOf(insBacker))
	}
	ca, cb := errVal(a.w.Checkpoint()), errVal(b.w.Checkpoint())
	if !bytes.Equal(errVal(ca.MarshalBinary()), errVal(cb.MarshalBinary())) {
		t.Errorf("the restored world's final checkpoint differs from the uninterrupted world's")
	}
}

// TestFork_InsuranceUnderwriterRemoval verifies the cell of the plan "at a removal of an
// underwriter": the removal at tick 1, with two route policies open, before the save and after
// the restore, leaves the original and the restored world with the same events, snapshot,
// checkpoint bytes and audit to the end of the run.
//
// Requirement: FR-INS-050, FR-INS-080, FR-INS-087
//
// Guard: TestFork_Insurance cuts at the removal of an insured because no golden pack removes an
// underwriter; this one ends the policies of an underwriter and restores across it.
func TestFork_InsuranceUnderwriterRemoval(t *testing.T) {
	t.Parallel()
	for _, before := range []bool{true, false} {
		t.Run(fmt.Sprintf("removed before the save %v", before), func(t *testing.T) {
			t.Parallel()
			forkAfterUnderwriterRemoval(t, before)
		})
	}
}
