package economysim

import (
	"testing"
)

// This file tests the settlement of a store policy at P8.1b and the welfare record it
// amends (plan 3-ins Phase 6, spec-insurance.md 8.1.3, 8.6.3, 11.1.3): the miller of home
// holds 30 grain with a store barn (100, 120) on a base room of (60, 40), so a third of its
// load is stored, and a certain store row of severity 5 000 bp strikes the barn.

// storeInsConfig is insConfig with the storage module, the store row and an insured with a
// share of 5 000 bp.
func storeInsConfig() Config {
	c := insConfig(&InsureTerms{StoreShareBP: 5_000, MinCoverBP: 5_000, MaxStoreRateBP: 200}, defaultUnderwrite())
	c.Modules.Storage = &Storage{Base: &Capacity{Mass: 60, Bulk: 40}, Classes: []StoreClass{{ID: "barn", Mass: 100, Bulk: 120}}}
	c.Modules.Risk.Stores = []StoreRisk{{Region: "a_home", Class: "barn", StrikeBP: 10_000, SeverityBP: 5_000}}
	return c
}

// storeWorld forms a store policy of insHome at tick 0 by the plan check and the round.
func storeWorld(t *testing.T) (*World, *evLog) {
	t.Helper()
	w, rec := newInsWorld(t, storeInsConfig(), nil)
	stockAgent(t, w, insHome, 30, 0)
	planStore(w, insHome, StoreCover{ShareBP: 5_000, MinCoverBP: 5_000, MaxStoreRateBP: 200})
	planOffer(w, insBacker, bigOffer)
	w.p33cCoverRound()
	if len(w.insurance.stores) != 1 {
		t.Fatalf("store policies = %d, want 1", len(w.insurance.stores))
	}
	return w, rec
}

// TestSettle_StorePolicyPaidAtP81b verifies P8.1b: the strike of P8.1a takes 5 units of
// grain from the insured (the StoreLoss event), valued at 1 000 each, so the loss value is
// 5 000 and the due halfEven(8 000 * 5 000 / 10 000) = 4 000, under the maximum payout of
// 12 000 (the sum is half the held 30 000); it is paid, the policy ends claimed, and the
// events come after StoreLoss.
//
// Requirement: FR-INS-040, FR-INS-041, FR-INS-043, FR-INS-070, FR-INS-072
func TestSettle_StorePolicyPaidAtP81b(t *testing.T) {
	t.Parallel()
	w, rec := storeWorld(t)
	w.p81aStores()
	w.p81bSettleStores()
	want := CoverEnded{Tick: 0, Policy: 0, Kind: CoverStore, Insured: insHome, Underwriter: insBacker, Region: regHome, UnderwriterRegion: regHome,
		Cause: CoverClaimed, LossValue: 5_000, Due: 4_000, Paid: 4_000, Claim: -1}
	if got := coverEndedAt(rec, 0); len(got) != 1 || got[0] != want {
		t.Errorf("CoverEnded = %+v, want one %+v", got, want)
	}
	if got := coverSettledAt(rec, 0); len(got) != 1 || got[0].Stage != CoverStageStore || got[0].Ended != 1 || got[0].Paid != 4_000 {
		t.Errorf("CoverSettled = %+v, want one of the stage store, one ended, 4000 paid", got)
	}
	if home := heldMoney(t, w, insHome); home != 10_000-150+4_000 {
		t.Errorf("insured's money = %d, want 13850 (premium 150, payout 4000)", home)
	}
	in := w.insurance
	if len(in.stores) != 0 || in.liabilityOf(insBacker) != 0 {
		t.Errorf("after P8.1b: store policies %d, liability %d, want none", len(in.stores), in.liabilityOf(insBacker))
	}
	if a, b, c := indexOfEvent[StoreLoss](rec), indexOfEvent[CoverEnded](rec), indexOfEvent[CoverSettled](rec); !(a < b && b < c) {
		t.Errorf("event positions StoreLoss %d, CoverEnded %d, CoverSettled %d, want that order", a, b, c)
	}
	auditNow(t, w)
}

// TestSettle_StoreNotStruckEndsWithNoEvent verifies spec-insurance.md 15.1.1: a store
// policy that ends no_loss emits no CoverEnded and is counted in CoverSettled.Ended.
//
// Requirement: FR-INS-071, FR-INS-072
func TestSettle_StoreNotStruckEndsWithNoEvent(t *testing.T) {
	t.Parallel()
	w, rec := storeWorld(t)
	for i := range w.risk.stores {
		w.risk.stores[i].scale, w.risk.stores[i].p = 0, 0 // no strike at P8.1a
	}
	w.p81aStores()
	w.p81bSettleStores()
	if got := coverEndedAt(rec, 0); len(got) != 0 {
		t.Errorf("CoverEnded = %+v, want none for a store policy that ended no_loss", got)
	}
	if got := coverSettledAt(rec, 0); len(got) != 1 || got[0].Ended != 1 || got[0].Claimed != 0 || got[0].Paid != 0 {
		t.Errorf("CoverSettled = %+v, want one ended, none claimed", got)
	}
	if heldMoney(t, w, insBacker) != 10_150 {
		t.Errorf("underwriter's money = %d, want the premium of 150 kept", heldMoney(t, w, insBacker))
	}
}

// TestWelfare_StorePayoutAmendsPushedRecord verifies spec-insurance.md 11.1.3: the payout of
// P8.1b is after P6.3, so it is added to the record of the same tick already pushed, as the
// premium of P3.3c is in the scratch. Run through a whole step: the insured's record of
// tick 0 has an unearned sum of payout less premium, and its income the same.
//
// Requirement: FR-INS-064
func TestWelfare_StorePayoutAmendsPushedRecord(t *testing.T) {
	t.Parallel()
	var w *World
	plan := func(v *AgentView, p *TradePlan) {
		if v.Tick.Tick != 0 {
			return
		}
		switch v.Agent {
		case insHome:
			p.Store = append(p.Store, StoreCover{ShareBP: 5_000, MinCoverBP: 5_000, MaxStoreRateBP: 200})
		case insBacker:
			p.Offer = append(p.Offer, bigOffer)
		}
	}
	w, rec := newInsWorld(t, storeInsConfig(), plan)
	stockAgent(t, w, insHome, 30, 0)
	step(t, w, 1)
	ended := coverEndedAt(rec, 0)
	if len(ended) != 1 || ended[0].Paid <= 0 {
		t.Fatalf("CoverEnded = %+v, want one paid policy", ended)
	}
	paid := int64(ended[0].Paid)
	for _, c := range []struct {
		id   AgentID
		want int64
	}{{insHome, paid - 150}, {insBacker, 150 - paid}} {
		if r := welfareRecords(w, c.id)[0]; r.Unearned != c.want || r.Income != c.want {
			t.Errorf("agent %d: record of tick 0 = %+v, want unearned and income %d (the payout of P8.1b amends it)", c.id, r, c.want)
		}
	}
}

// TestWelfare_NoRecordNotAmended verifies an agent with no record of the tick is not
// amended and nothing is written (spec-insurance.md 11.1.3).
//
// Requirement: FR-INS-064
func TestWelfare_NoRecordNotAmended(t *testing.T) {
	t.Parallel()
	w, _ := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	a := &w.agents[insFar]
	a.history = newHistory(w.roles.row(a.role))
	w.coverWelfare(insFar, 500, true)
	if a.history.Len() != 0 {
		t.Errorf("history length = %d, want 0: an agent with no record is not amended", a.history.Len())
	}
	w.coverWelfare(insFar, 500, false)
	if got := w.welfare.at(int(insFar)).unearned; got != 500 {
		t.Errorf("scratch unearned = %d, want 500 for a flow before P6.3", got)
	}
}
