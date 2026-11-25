package economysim

import (
	"testing"
)

// This file tests step 2f of a removal (plan 3-ins Phase 6, spec-insurance.md 8.6): the open
// policies of a removed agent end with a cause and no money moves, in the order of the
// events of spec-world.md 6.11, and what the waterfall of the credit module pays on a payout
// claim is in the creditor's record. The world is that of insurance_settle_internal_test.go.

// remove removes an agent through the one removal path.
func remove(t *testing.T, w *World, id AgentID) {
	t.Helper()
	if err := w.removeAgent(&w.agents[id], CauseInsolvent); err != nil {
		t.Fatalf("removeAgent(%d) error = %v, want nil", id, err)
	}
}

// TestRemoval_Step2fEndsThePoliciesOfAnInsured verifies that a removed insured's open
// policy ends insured_removed with every amount 0 and no money moved (the premium is not
// refunded), that CoverEnded comes before AgentBankrupted, and that the shipment, which
// arrives afterwards, is struck as before and pays nothing.
//
// Requirement: FR-INS-050, FR-INS-051, FR-INS-053, FR-INS-070
func TestRemoval_Step2fEndsThePoliciesOfAnInsured(t *testing.T) {
	t.Parallel()
	w, rec := settleWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), 8)
	step(t, w, 1)
	under := heldMoney(t, w, insBacker)
	remove(t, w, insHome)

	want := CoverEnded{Tick: 1, Policy: 0, Kind: CoverRoute, Insured: insHome, Underwriter: insBacker, Region: regHome, UnderwriterRegion: regHome,
		Cause: CoverInsuredRemoved, Claim: -1}
	if got := of[CoverEnded](rec); len(got) != 1 || got[0] != want {
		t.Errorf("CoverEnded = %+v, want one %+v", got, want)
	}
	if a, b := indexOfEvent[CoverEnded](rec), indexOfEvent[AgentBankrupted](rec); a >= b {
		t.Errorf("CoverEnded at %d, AgentBankrupted at %d, want the policy first", a, b)
	}
	in := w.insurance
	if len(in.open) != 0 || in.covered(0) || in.liabilityOf(insBacker) != 0 || heldMoney(t, w, insBacker) != under {
		t.Errorf("after 2f: open %d, covered %v, liability %d, underwriter's money %d (was %d), want no policy and no money moved", len(in.open), in.covered(0), in.liabilityOf(insBacker), heldMoney(t, w, insBacker), under)
	}
	step(t, w, 2) // the shipment arrives at tick 2 with lines: struck, no payout
	if got := of[CoverEnded](rec); len(got) != 1 || len(of[CoverSettled](rec)) != 0 {
		t.Errorf("events after the arrival: CoverEnded %d, CoverSettled %d, want the one of the removal and no settlement", len(got), len(of[CoverSettled](rec)))
	}
	auditNow(t, w)
}

// TestRemoval_Step2fEndsThePoliciesOfAnUnderwriter verifies the other side: a removed
// underwriter's policy ends underwriter_removed and the insured keeps the premium paid and
// loses the cover: nothing is paid at the arrival.
//
// Requirement: FR-INS-050, FR-INS-051
func TestRemoval_Step2fEndsThePoliciesOfAnUnderwriter(t *testing.T) {
	t.Parallel()
	w, rec := settleWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), 8, 12)
	step(t, w, 1)
	remove(t, w, insBacker)
	got := of[CoverEnded](rec)
	if len(got) != 2 || got[0].Policy != 0 || got[1].Policy != 1 || got[0].Cause != CoverUnderwriterRemoved || got[1].Cause != CoverUnderwriterRemoved {
		t.Fatalf("CoverEnded = %+v, want policies 0 then 1 ended underwriter_removed, in ascending ID", got)
	}
	if len(w.insurance.open) != 0 || heldMoney(t, w, insHome) != 10_000-1_600 {
		t.Errorf("open %d, insured's money %d, want none open and the premium not refunded (8400)", len(w.insurance.open), heldMoney(t, w, insHome))
	}
	step(t, w, 2)
	if len(of[CoverEnded](rec)) != 2 || len(of[CoverSettled](rec)) != 0 {
		t.Errorf("a settlement happened after the underwriter was removed")
	}
	auditNow(t, w)
}

// TestRemoval_NoPoliciesIsCheap verifies the removal of an agent with no policy touches
// nothing and emits nothing, and a world with the module off removes as before (the call is
// nil-safe).
//
// Requirement: FR-INS-050
func TestRemoval_NoPoliciesIsCheap(t *testing.T) {
	t.Parallel()
	w, rec := settleWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), 8)
	remove(t, w, insFar)
	if len(of[CoverEnded](rec)) != 0 || len(w.insurance.open) != 1 {
		t.Errorf("CoverEnded %d, open %d, want no event and the policy still open", len(of[CoverEnded](rec)), len(w.insurance.open))
	}
	var off *World
	off, _ = newInsWorld(t, shipOffConfig(), nil)
	off.dropPolicies(&off.agents[insHome]) // nil-safe with the module off
}

// shipOffConfig is insConfig with the module off.
func shipOffConfig() Config {
	c := insConfig(nil, nil)
	c.Modules.Insurance = nil
	return c
}

// TestRemoval_StorePolicyOfTheTickEndsWithNoPayout verifies spec-insurance.md 8.6.3: a store
// policy of the tick, removed before P8.1b, ends with the cause and no payout, the premium
// stays with the underwriter, and P8.1b finds nothing to settle.
//
// Requirement: FR-INS-050, FR-INS-052
func TestRemoval_StorePolicyOfTheTickEndsWithNoPayout(t *testing.T) {
	t.Parallel()
	w, rec := storeWorld(t)
	remove(t, w, insHome)
	w.p81aStores()
	w.p81bSettleStores()
	got := of[CoverEnded](rec)
	if len(got) != 1 || got[0].Cause != CoverInsuredRemoved || got[0].Kind != CoverStore || got[0].Paid != 0 || got[0].Due != 0 {
		t.Errorf("CoverEnded = %+v, want one store policy ended insured_removed with nothing paid", got)
	}
	if len(of[CoverSettled](rec)) != 0 || heldMoney(t, w, insBacker) != 10_150 {
		t.Errorf("CoverSettled %d, underwriter's money %d, want no settlement and the premium kept", len(of[CoverSettled](rec)), heldMoney(t, w, insBacker))
	}
}

// TestRemoval_WaterfallPaysAPayoutClaimIntoTheRecord verifies the order of a removal and
// the late term of N_t (spec-insurance.md 11.1.3): an underwriter with two payout claims
// open and a policy is removed after the record of the tick was pushed; step 2d's waterfall
// pays the claims from its whole balance, then 2f ends the policy, then the agent is
// bankrupted, and the principal paid amends the last record of each creditor.
//
// Worked: at the arrival the underwriter holds 1 500, so its pot is 500 shared 200 and 300
// between dues of 1 600 and 2 400; the claims are 1 400 and 2 100. It is then given 20 000
// and forms a third policy; the waterfall pays both claims whole, 1 400 and 2 100.
//
// Requirement: FR-INS-050, FR-INS-064
func TestRemoval_WaterfallPaysAPayoutClaimIntoTheRecord(t *testing.T) {
	t.Parallel()
	w, rec := settleWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), 8, 12)
	step(t, w, 2)
	moneyOf(t, w, insBacker, 1_500)
	step(t, w, 1)
	moneyOf(t, w, insBacker, 20_000)
	insDispatch(t, w, insHome, 8)
	planCover(w, insHome, coachLane)
	planOffer(w, insBacker, bigOffer)
	w.p33cCoverRound()
	if len(w.insurance.open) != 1 {
		t.Fatalf("open policies = %d, want the third", len(w.insurance.open))
	}
	before := [2]int64{welfareRecords(w, insHome)[2].Unearned, welfareRecords(w, insHome2)[2].Unearned}
	remove(t, w, insBacker)

	if ends := of[LoanEnded](rec); len(ends) != 2 || ends[0].Cause != CauseDebtorRemoved || ends[0].Origin != OriginPayout || ends[1].PaidPrincipal+ends[0].PaidPrincipal != 3_500 {
		t.Fatalf("LoanEnded = %+v, want the two payout claims ended debtor_removed, 3500 paid in all", ends)
	}
	if a, b, c := indexOfEvent[LoanEnded](rec), indexOfEvent[CoverEnded](rec), indexOfEvent[AgentBankrupted](rec); !(a < b && b < c) {
		t.Errorf("LoanEnded at %d, CoverEnded at %d, AgentBankrupted at %d, want that order", a, b, c)
	}
	after := [2]int64{welfareRecords(w, insHome)[2].Unearned, welfareRecords(w, insHome2)[2].Unearned}
	if after != [2]int64{before[0] + 1_400, before[1] + 2_100} {
		t.Errorf("last records' unearned %v from %v, want the principal paid, 1400 and 2100, added", after, before)
	}
	auditNow(t, w)
}
