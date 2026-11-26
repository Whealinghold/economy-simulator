package economysim

import (
	"slices"
	"testing"
)

// This file holds the two tests the plan of Phase 6 listed that the first cut left out
// (plan 3-ins, N-470 and NFR-INS-002): a held arrival settles later, and the settlement
// finds its policies by the shipment index and never by a scan of the agents.

// footWorld builds the world of the held-arrival test: a self-carried trip on foot with a
// route policy formed at tick 0, and the tick at which it arrives.
func footWorld(t *testing.T) (*World, *evLog, int64) {
	t.Helper()
	c := insConfig(defaultInsure(), defaultUnderwrite())
	c.Modules.Risk.Routes = append(c.Modules.Risk.Routes, riskFoot("a_home", "b_far", 10_000, 10_000, false))
	w, rec := newInsWorld(t, c, nil)
	give(t, w, insHome, commodityOf(t, w, "grain"), 2)
	ship, reason := ask(w, insHome, 0, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: insFar,
		Lines: []Line{{Commodity: commodityOf(t, w, "grain"), Quantity: 2}}})
	if reason != "" {
		t.Fatalf("self-carried dispatch = (%d, %q), want dispatched", ship, reason)
	}
	planCover(w, insHome, LaneCover{Mode: modeFoot, To: regFar, MinCoverBP: 5_000, MaxLoadingBP: 20_000})
	planOffer(w, insBacker, bigOffer)
	w.p33cCoverRound()
	if len(w.insurance.open) != 1 {
		t.Fatalf("policies formed = %d, want 1", len(w.insurance.open))
	}
	return w, rec, w.transport.ships[0].arrives
}

// TestRoute_HeldArrivalSettlesLater verifies spec-insurance.md 8.1.2: an arrival that the
// order guard of the transport module holds (another obligation of the self-carrying
// traveller refuses the region change) has no row in the list of arrivals, so its policy
// stays open with no event and no payout, and it settles once, at the arrival that
// completes, with the loss of that arrival.
//
// Worked: a trip on foot with a certain row of severity 10 000, so the premium equals the
// maximum payout, 9 600 (cover share 8 000 bp of a sum of 12 000, the cargo and what the
// traveller's account holds); each side starts with 10 000, so the insured holds 400 and the
// underwriter 19 600 once the premium moved. The loss value is 9 000 and the due is the cover
// share of it, 8 000 bp of 9 000 = 7 200, under the maximum. The arrival is held by an
// obligation of the traveller (the hired coach of the other tests is never held), then
// released.
//
// Requirement: FR-INS-040, FR-INS-050
// Guard: a loss noted at a held attempt would settle the policy twice (once held, once at
// the retry), or settle it with the loss of an arrival that applied nothing.
func TestRoute_HeldArrivalSettlesLater(t *testing.T) {
	t.Parallel()
	w, rec, arrives := footWorld(t)
	holding := true
	w.bounds.add("test_hold", func(a *agent, _ []string) bool { return holding && a.id == insHome })
	step(t, w, int(arrives)+1) // the arrival is held
	if got := len(of[ShipmentArrived](rec)); got != 0 {
		t.Fatalf("ShipmentArrived events while held = %d, want 0", got)
	}
	if got, want := shape(w, insBacker, insHome), [4]int64{1, 0, 9_600, 1}; got != want {
		t.Errorf("lists while held (open, stores, liability, policies) = %v, want %v: the policy is still open", got, want)
	}
	if got := len(of[CoverEnded](rec)) + len(of[CoverSettled](rec)); got != 0 {
		t.Errorf("cover events while held = %d, want 0", got)
	}
	// the premium moved at tick 0 and no payout has.
	if got, want := moneyOfAgents(t, w, insHome, insBacker), []int64{400, 19_600}; !slices.Equal(got, want) {
		t.Errorf("money of the insured and the underwriter while held = %v, want %v", got, want)
	}
	holding = false
	step(t, w, 1) // the retry completes the arrival
	e := only(t, "CoverEnded after the retry", of[CoverEnded](rec))
	if got, want := figures(e), [5]Money{7_200, 7_200, 0, 0, 0}; got != want || e.LossValue != 9_000 || e.Tick != arrives+1 || e.Cause != CoverClaimed {
		t.Errorf("CoverEnded = %+v, want tick %d, claimed, figures %v", e, arrives+1, want)
	}
	if got := shape(w, insBacker, insHome); got != [4]int64{} {
		t.Errorf("lists after the retry = %v, want the policy gone", got)
	}
	auditNow(t, w)
}

// TestInsurance_WorkIsCounted verifies NFR-INS-002 for P1.3a: the position finds the
// policies of the arrivals of the tick by the shipment index and reads no other policy and
// no other agent. A policy of a stand-in agent that does not exist stands in for the read:
// reading its agent indexes past the agents and panics, so it is read when its shipment
// arrived (the control) and never when only another shipment arrived or none did.
//
// Requirement: NFR-INS-002
// Guard: P1.3a visiting every open policy, or every agent, in a tick with one arrival.
func TestInsurance_WorkIsCounted(t *testing.T) {
	t.Parallel()
	w, _ := settleWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), 8)
	in := w.insurance
	real := coverArrival{ship: in.open[0].shipment, from: regHome}
	ghost := AgentID(len(w.agents))
	const ghostShip = ShipmentID(9_999)
	in.add(coverPolicy{id: 99, kind: CoverRoute, insured: ghost, underwriter: ghost, shipment: ghostShip, sum: 1, maxPayout: 1, coverBP: 8_000})
	reads := func() (read bool) {
		defer func() { read = recover() != nil }()
		w.p13aSettleRoutes()
		return false
	}
	in.arrived = append(in.arrived[:0], coverArrival{ship: ghostShip, from: regHome})
	if !reads() {
		t.Fatal("P1.3a with the arrival of the stand-in's shipment did not read its agent, want the control to read it")
	}
	in.arrived = in.arrived[:0]
	if reads() {
		t.Error("P1.3a with no arrival read an agent, want none")
	}
	in.arrived = append(in.arrived[:0], real)
	if reads() {
		t.Error("P1.3a with the arrival of another shipment read the stand-in's agent, want the shipment index only")
	}
	if _, open := in.route(0); open {
		t.Error("the policy of the arrived shipment is still open after P1.3a, want it settled")
	}
	if _, open := in.route(99); !open {
		t.Error("the policy of the stand-in was settled by the arrival of another shipment, want it open")
	}
}
