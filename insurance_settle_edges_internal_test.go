package economysim

import (
	"testing"
)

// This file tests the edges of the settlement of a route policy (plan 3-ins Phase 6): a
// payment cut at the insured's headroom, a full claim list, a shipment with no loss, the
// units kept at the arrival, and the cases where the insured or the underwriter is gone.
// The world and the arithmetic are those of insurance_settle_internal_test.go.

// TestSettle_HeadroomCutIsDroppedNotClaimed verifies spec-insurance.md 8.3.4: a payment that
// would take the insured above MaxMoney is cut to the headroom, the cut is counted and no
// claim is formed for it.
//
// Worked: the insured holds MaxMoney - 100 at the arrival, so of the 1 600 due only 100
// moves and 1 500 is cut; the identity 1 600 = 100 + 0 + 0 + 1 500 holds.
//
// Requirement: FR-INS-043, FR-INS-006
func TestSettle_HeadroomCutIsDroppedNotClaimed(t *testing.T) {
	t.Parallel()
	w, rec := settleWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), 8)
	step(t, w, 2)
	moneyOf(t, w, insHome, MaxMoney-100)
	step(t, w, 1)
	e := only(t, "CoverEnded at tick 2", coverEndedAt(rec, 2))
	if got, want := figures(e), [5]Money{1_600, 100, 0, 0, 1_500}; got != want || e.Claim != -1 {
		t.Errorf("CoverEnded figures = %v, claim %d, want due 1600 = paid 100 + cut 1500, no claim", got, e.Claim)
	}
	if _, open := w.ledger.Claim(0); open {
		t.Errorf("a claim was formed for a cut payment")
	}
	if tot := w.insurance.tot; tot != (coverTotals{formed: 1, premiums: 1_600, due: 1_600, paid: 100, cut: 1_500}) {
		t.Errorf("totals = %+v, want due 1600, paid 100, cut 1500", tot)
	}
	auditNow(t, w)
}

// TestClaim_ListFullDropped verifies spec-insurance.md 8.4.2: with the claim list full the
// shortfall cannot be recorded, so it is dropped and counted, the event says so and no error
// is returned.
//
// Requirement: FR-INS-045, FR-INS-006, FR-INS-004
func TestClaim_ListFullDropped(t *testing.T) {
	t.Parallel()
	c := insConfig(defaultInsure(), defaultUnderwrite())
	c.Modules.Credit.MaxClaims = 1
	w, rec := settleWorld(t, c, 8)
	plantPayout(t, w, insFar, insHome2, 5, 100) // the one claim the list holds
	w.insurance.tot.due += 5                    // the module's Awarded counts it, so its due does (9.4)
	step(t, w, 2)
	moneyOf(t, w, insBacker, 1_500)
	step(t, w, 1)
	e := only(t, "CoverEnded at tick 2", coverEndedAt(rec, 2))
	if got, want := figures(e), [5]Money{1_600, 500, 0, 1_100, 0}; got != want || e.Claim != -1 {
		t.Errorf("CoverEnded figures = %v, claim %d, want paid 500, dropped 1100, no claim", got, e.Claim)
	}
	if tot := w.insurance.tot; tot.dropped != 1_100 || tot.due-5 != tot.paid+tot.dropped+tot.cut {
		t.Errorf("totals = %+v, want dropped 1100 and due = paid + dropped + cut (the planted claim of 5 aside)", tot)
	}
	auditNow(t, w)
}

// TestSettle_NoLossEndsTheRoutePolicy verifies the cause no_loss: a strike that took no unit
// (severity 100 bp of 8 grain rounds to 0) ends the policy with no payout, the premium is
// kept, and the event is still emitted for a route policy.
//
// Requirement: FR-INS-040, FR-INS-050, FR-INS-071
func TestSettle_NoLossEndsTheRoutePolicy(t *testing.T) {
	t.Parallel()
	c := insConfig(defaultInsure(), defaultUnderwrite())
	c.Modules.Risk.Routes[0].SeverityBP = 100
	w, rec := settleWorld(t, c, 8)
	step(t, w, 3)
	e := only(t, "CoverEnded at tick 2", coverEndedAt(rec, 2))
	if got := [3]Money{e.Due, e.Paid, e.LossValue}; got != [3]Money{} || e.Cause != CoverNoLoss {
		t.Errorf("CoverEnded = %+v, want one no_loss with nothing due", e)
	}
	if s := only(t, "CoverSettled at tick 2", coverSettledAt(rec, 2)); [2]int64{s.Ended, s.Claimed} != [2]int64{1, 0} {
		t.Errorf("CoverSettled = %+v, want one ended and none claimed", s)
	}
	if got := moneyOfAgents(t, w, insBacker); len(w.insurance.open) != 0 || got[0] != 10_064 {
		t.Errorf("open %d, underwriter's money %v, want no open policy and the premium of 64 kept (halfEven(8000 * 100 * 8000 * 10000 / 10^12))", len(w.insurance.open), got)
	}
	auditNow(t, w)
}

// TestNoteArrival_KeepsOnlyInsuredShipments is the Guard test of hook H-11 (risk_loss.go):
// the units lost at an arrival are kept for P1.3a only for a shipment that has a route
// policy, the lost units of the lines then those of the traveller's account, and a nil
// module keeps nothing. The list is kept without an observer.
//
// Requirement: FR-INS-040
func TestNoteArrival_KeepsOnlyInsuredShipments(t *testing.T) {
	t.Parallel()
	var none *insurance
	none.noteArrival(&shipment{id: 1}, []int64{3}, nil) // nil-safe, no panic
	in := &insurance{byShipment: map[ShipmentID]int64{7: 0}}
	s := &shipment{id: 7, from: 2, lines: []Line{{Commodity: 1, Quantity: 8}, {Commodity: 2, Quantity: 4}}}
	in.noteArrival(&shipment{id: 8, lines: s.lines}, []int64{1, 1}, nil)
	if len(in.arrived) != 0 {
		t.Fatalf("an uninsured shipment left %d rows, want none", len(in.arrived))
	}
	in.noteArrival(s, []int64{0, 3}, []Amount{{Commodity: 4, Quantity: 5}})
	want := []Amount{{Commodity: 2, Quantity: 3}, {Commodity: 4, Quantity: 5}}
	if len(in.arrived) != 1 || in.arrived[0].ship != 7 || in.arrived[0].from != 2 || len(in.amounts) != 2 || in.amounts[0] != want[0] || in.amounts[1] != want[1] {
		t.Errorf("rows %+v amounts %+v, want one row of shipment 7 from region 2 with %+v (a line that lost nothing is not kept)", in.arrived, in.amounts, want)
	}
}

// TestRoute_ArrivalListKeptWithoutObserver verifies the settlement needs no observer: a
// world with none pays the same policy at the same tick (FR-INS-040, spec-insurance.md
// 8.1.2), and the arrivals are cleared once P1.3a has run.
//
// Requirement: FR-INS-040, FR-INS-043
func TestRoute_ArrivalListKeptWithoutObserver(t *testing.T) {
	t.Parallel()
	w := newMill(t, insConfig(defaultInsure(), defaultUnderwrite()), WithAudit(), WithStrategy("script", &insScript{}))
	w.obs = observers{}
	if err := w.ledger.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction() = %v, want nil", err)
	}
	insDispatch(t, w, insHome, 8)
	planCover(w, insHome, coachLane)
	planOffer(w, insBacker, bigOffer)
	w.p33cCoverRound()
	step(t, w, 3)
	if home := heldMoney(t, w, insHome); home != 10_000 {
		t.Errorf("insured's money = %d, want 10000 (premium 1600 paid, payout 1600 received) with no observer", home)
	}
	if len(w.insurance.arrived) != 0 || len(w.insurance.amounts) != 0 {
		t.Errorf("arrivals kept after P1.3a: %d rows, %d amounts, want none", len(w.insurance.arrived), len(w.insurance.amounts))
	}
}
