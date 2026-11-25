package economysim

import (
	"testing"
)

// This file tests the admission of the cover round (plan 3-ins Phase 5, spec-insurance.md
// 5.2): the refusal of a request or an offer is the first failing check of the table, in
// its order, and an admitted offer is never cut to its room.

// TestAdmission_RequestRefusals verifies, one case for each row of the table of 5.2 that a
// request can fail, that the first failing check is the reason, and that a request that
// breaks only that rule is the case.
//
// Requirement: FR-INS-012, FR-INS-013
func TestAdmission_RequestRefusals(t *testing.T) {
	t.Parallel()
	w, _ := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	lane := func(mut func(l *LaneCover)) LaneCover {
		l := coachLane
		mut(&l)
		return l
	}
	tests := []struct {
		name  string
		agent AgentID
		req   coverRequest
		want  Reason
	}{
		{"admitted", insHome, coverRequest{lane: coachLane}, ""},
		{"a role with no Insure block", insBacker, coverRequest{lane: coachLane}, ReasonNotInsuredRole},
		{"the request after the last allowed", insHome, coverRequest{seq: 4, lane: coachLane}, ReasonTooManyRequests},
		{"the last allowed request", insHome, coverRequest{seq: 3, lane: coachLane}, ""},
		{"cover share 0", insHome, coverRequest{lane: lane(func(l *LaneCover) { l.MinCoverBP = 0 })}, ReasonInvalidCover},
		{"cover share above 10 000", insHome, coverRequest{lane: lane(func(l *LaneCover) { l.MinCoverBP = 10_001 })}, ReasonInvalidCover},
		{"loading below 0", insHome, coverRequest{lane: lane(func(l *LaneCover) { l.MaxLoadingBP = -1 })}, ReasonInvalidCover},
		{"loading above its bound", insHome, coverRequest{lane: lane(func(l *LaneCover) { l.MaxLoadingBP = MaxLoadingBound + 1 })}, ReasonInvalidCover},
		{"store share 0", insHome, coverRequest{store: true, sc: StoreCover{ShareBP: 0, MinCoverBP: 1}}, ReasonInvalidCover},
		{"store rate above its bound", insHome, coverRequest{store: true, sc: StoreCover{ShareBP: 1, MinCoverBP: 1, MaxStoreRateBP: MaxStoreRateBound + 1}}, ReasonInvalidCover},
		{"a mode that is not configured", insHome, coverRequest{lane: lane(func(l *LaneCover) { l.Mode = 9 })}, ReasonUnknownLane},
		{"a negative mode", insHome, coverRequest{lane: lane(func(l *LaneCover) { l.Mode = -1 })}, ReasonUnknownLane},
		{"the agent's own region", insHome, coverRequest{lane: lane(func(l *LaneCover) { l.To = regHome })}, ReasonUnknownLane},
		{"a destination that is not a region", insHome, coverRequest{lane: lane(func(l *LaneCover) { l.To = 9 })}, ReasonUnknownLane},
		{"a store request with no store", insHome, coverRequest{store: true, sc: StoreCover{ShareBP: 1, MinCoverBP: 1}}, ReasonNoStore},
		// 4 before 5: both invalid_cover and unknown_lane hold, the earlier row is the reason.
		{"first failure wins", insHome, coverRequest{lane: lane(func(l *LaneCover) { l.MinCoverBP, l.To = 0, 9 })}, ReasonInvalidCover},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.req.agent = tt.agent
			if got := w.coverRequestRefusal(&w.agents[tt.agent], &tt.req); got != tt.want {
				t.Errorf("coverRequestRefusal(%+v) = %q, want %q", tt.req, got, tt.want)
			}
		})
	}
}

// TestAdmission_DuplicateRequestAndOffer verifies row 6: a second request for the same lane,
// a second store request and a second offer of an agent are refused duplicate, whatever the
// terms; another lane is admitted.
//
// Requirement: FR-INS-012
func TestAdmission_DuplicateRequestAndOffer(t *testing.T) {
	t.Parallel()
	w, _ := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	planCover(w, insHome, coachLane)
	planOffer(w, insBacker, fairOffer)
	other := coachLane
	other.MinCoverBP = 9_000 // the same lane, other terms
	planCover(w, insHome, other)
	planOffer(w, insBacker, CoverOffer{MaxLiability: 11, CoverBP: 1})
	w.admitCoverRequests()
	w.admitCoverOffers()
	c := w.insurance.round.counts[regHome]
	if c.requests != 1 || c.requestsRefused != 1 || c.offers != 1 || c.offersRefused != 1 {
		t.Errorf("counts = %+v, want one admitted and one refused request and offer", c)
	}
}

// TestAdmission_OfferRefusals verifies the rows of an offer: not an underwriter, invalid
// terms, in arrears (an overdue payout claim as debtor), and exposure, and that an offer
// above its room is refused whole and never cut to it.
//
// Worked: the underwriter holds M money and a cap of 20 000 bp, so its room is 2M with
// nothing open: an offer of 2M is admitted and one of 2M + 1 is refused exposure.
//
// Requirement: FR-INS-012, FR-INS-014, FR-INS-015
func TestAdmission_OfferRefusals(t *testing.T) {
	t.Parallel()
	w, _ := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	room := 2 * heldMoney(t, w, insBacker)
	offer := func(mut func(o *CoverOffer)) CoverOffer {
		o := fairOffer
		mut(&o)
		return o
	}
	tests := []struct {
		name  string
		agent AgentID
		o     CoverOffer
		want  Reason
	}{
		{"admitted", insBacker, fairOffer, ""},
		{"a role with no Underwrite block", insHome, fairOffer, ReasonNotUnderwriter},
		{"liability below MinSum", insBacker, offer(func(o *CoverOffer) { o.MaxLiability = 9 }), ReasonInvalidOffer},
		{"cover share 0", insBacker, offer(func(o *CoverOffer) { o.CoverBP = 0 }), ReasonInvalidOffer},
		{"cover share above 10 000", insBacker, offer(func(o *CoverOffer) { o.CoverBP = 10_001 }), ReasonInvalidOffer},
		{"loading above its bound", insBacker, offer(func(o *CoverOffer) { o.LoadingBP = MaxLoadingBound + 1 }), ReasonInvalidOffer},
		{"store rate below 0", insBacker, offer(func(o *CoverOffer) { o.StoreRateBP = -1 }), ReasonInvalidOffer},
		{"liability at the room", insBacker, offer(func(o *CoverOffer) { o.MaxLiability = room }), ""},
		{"liability one above the room", insBacker, offer(func(o *CoverOffer) { o.MaxLiability = room + 1 }), ReasonExposure},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := w.coverOfferRefusal(&w.agents[tt.agent], &coverOffer{agent: tt.agent, o: tt.o}); got != tt.want {
				t.Errorf("coverOfferRefusal(%+v) = %q, want %q", tt.o, got, tt.want)
			}
		})
	}
	t.Run("in arrears", func(t *testing.T) {
		t.Parallel()
		w, _ := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
		id := plantPayout(t, w, insHome, insBacker, 50, 1)
		if err := w.ledger.MarkOverdue(id, 1); err != nil {
			t.Fatalf("MarkOverdue error = %v, want nil", err)
		}
		if got := w.coverOfferRefusal(&w.agents[insBacker], &coverOffer{agent: insBacker, o: fairOffer}); got != ReasonInArrears {
			t.Errorf("coverOfferRefusal with an overdue payout claim = %q, want %q", got, ReasonInArrears)
		}
	})
}

// TestAdmission_OfferCountsInArrearsAndExposure verifies that the two refusals the event
// counts apart are subsets of OffersRefused.
//
// Requirement: FR-INS-012, FR-INS-014
func TestAdmission_OfferCountsInArrearsAndExposure(t *testing.T) {
	t.Parallel()
	w, rec := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	planOffer(w, insBacker, CoverOffer{MaxLiability: 1 << 40, CoverBP: 8_000})
	w.p33cCoverRound()
	if _, rounds := coverEvents(rec); len(rounds) != 1 || rounds[0].OffersRefused != 1 || rounds[0].Exposure != 1 || rounds[0].InArrears != 0 {
		t.Errorf("CoverRound = %+v, want one with OffersRefused 1 and Exposure 1", rounds)
	}
}

// TestAdmission_AwayAgentAdmitted verifies that a request of an agent that is away on a
// self-carried trip is admitted: its plan was built before it left, and its lane is the
// region of its account, which has not moved.
//
// Requirement: FR-INS-012, FR-INS-016
func TestAdmission_AwayAgentAdmitted(t *testing.T) {
	t.Parallel()
	c := insConfig(defaultInsure(), defaultUnderwrite())
	c.Modules.Risk.Routes = append(c.Modules.Risk.Routes, riskFoot("a_home", "b_far", 10_000, 2_500, false))
	w, rec := newInsWorld(t, c, nil)
	if _, reason := ask(w, insHome, 0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}); reason != "" {
		t.Fatalf("trip = %q, want dispatched", reason)
	}
	if !w.Away(insHome) {
		t.Fatal("the traveller is not away after the dispatch")
	}
	planCover(w, insHome, LaneCover{Mode: modeFoot, To: regFar, MinCoverBP: 1, MaxLoadingBP: MaxLoadingBound})
	planOffer(w, insBacker, CoverOffer{MaxLiability: 20_000, LoadingBP: 10_000, CoverBP: 10_000})
	w.p33cCoverRound()
	if formed, _ := coverEvents(rec); len(formed) != 1 || formed[0].Insured != insHome {
		t.Errorf("CoverFormed = %+v, want one policy of the away traveller", formed)
	}
}
