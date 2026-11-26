package economysim

import (
	"slices"
	"testing"
)

// This file tests that an observer with no agent table rebuilds the money of the household
// sector by region from the events of the insurance module (plan 3-ins Phase 7,
// spec-insurance.md 15.1.2, FR-INS-079).

// householdMoney returns the money of the households of each region at the boundary.
func householdMoney(t *testing.T, w *World) []Money {
	t.Helper()
	s := errVal(w.Snapshot())
	out := make([]Money, len(s.Regions))
	for g := range s.Regions {
		out[g] = s.Regions[g].MoneyHouseholds
	}
	return out
}

// rebuiltDelta is the change of the household money by region that the events of a tick give:
// CoverFormed moves the premium from the insured's region to the underwriter's, CoverEnded the
// part paid at once the other way, and CoverSettled of the stage claim the principal of a payout
// claim from the debtors' region to the insureds'.
func rebuiltDelta(rec *evLog, tick int64, regions int) []Money {
	d := make([]Money, regions)
	for _, e := range of[CoverFormed](rec) {
		if e.Tick == tick {
			d[e.Region] -= e.Premium
			d[e.UnderwriterRegion] += e.Premium
		}
	}
	for _, e := range coverEndedAt(rec, tick) {
		d[e.UnderwriterRegion] -= e.Paid
		d[e.Region] += e.Paid
	}
	for _, e := range coverSettledAt(rec, tick) {
		if e.Stage == CoverStageClaim {
			d[e.Region] += e.ClaimReceived - e.ClaimPaid
		}
	}
	return d
}

// TestObserver_RebuildsRegionMoneyFromCoverEvents verifies that the change of the money of the
// households of each region over two positions of the module (a part paid at once by an
// underwriter that moved to another region, and the principal of the payout claim paid later) is the
// change the events give, so an observer that holds only the levels of a snapshot and the
// events keeps the household money by region.
//
// Arithmetic: the premium of 1 600 is paid inside home, before the underwriter moves to far.
// At the arrival the due of 1 600 is paid 500 from the pot of the underwriter in far to the
// insured in home, and the claim of 1 100 is repaid a tick later the same way.
//
// Requirement: FR-INS-079
func TestObserver_RebuildsRegionMoneyFromCoverEvents(t *testing.T) {
	t.Parallel()
	const insured, backer = insHome, insBacker
	w, rec := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	insDispatch(t, w, insured, 8)
	planCover(w, insured, coachLane)
	planOffer(w, backer, bigOffer)
	w.p33cCoverRound()
	if err := w.relocate(backer, regFar, CauseConsumer); err != nil {
		t.Fatalf("relocate error = %v, want nil: the underwriter moves to far with its policy open", err)
	}
	step(t, w, 2)
	moneyOf(t, w, backer, 1_500)
	before := householdMoney(t, w)
	step(t, w, 1)
	checkDelta(t, w, rec, "the arrival", before, 2)
	moneyOf(t, w, backer, 5_000)
	before = householdMoney(t, w)
	step(t, w, 1)
	checkDelta(t, w, rec, "the payment of the claim", before, 3)
	if len(of[LoanEnded](rec)) != 1 {
		t.Errorf("LoanEnded events = %d, want 1 for the payout claim", len(of[LoanEnded](rec)))
	}
}

// checkDelta fails the test unless the household money by region moved from before by what the
// events of the tick give, and unless that is not all zero.
func checkDelta(t *testing.T, w *World, rec *evLog, what string, before []Money, tick int64) {
	t.Helper()
	got := householdMoney(t, w)
	want := rebuiltDelta(rec, tick, len(got))
	for g := range got {
		got[g] -= before[g]
	}
	if !slices.Equal(got, want) || slices.Equal(want, make([]Money, len(want))) {
		t.Errorf("%s: change of household money by region = %v, events give %v, want equal and not all 0", what, got, want)
	}
}
