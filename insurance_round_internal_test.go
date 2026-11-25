package economysim

import (
	"reflect"
	"slices"
	"testing"
)

// This file tests the cover round of P3.3c (plan 3-ins Phase 5, spec-insurance.md 5.2 and
// 7) over the world of insConfig: a shipment is dispatched by the ledger path of the
// transport tests, the requests and offers are planted in the round as a plan check
// would keep them, and the round is run by its position method. Nothing here settles a
// policy: a route policy stays open and a store policy ends at the next round.

// insDispatch gives an agent the haul units and qty grain for one hired coach shipment to
// far (the consignee is the first miller of far) and dispatches it at the current tick.
func insDispatch(t *testing.T, w *World, id AgentID, qty int64) ShipmentID {
	t.Helper()
	give(t, w, id, commodityOf(t, w, "haul"), 8)
	give(t, w, id, commodityOf(t, w, "grain"), qty)
	s, reason := ask(w, id, 0, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: insFar,
		Lines: []Line{{Commodity: commodityOf(t, w, "grain"), Quantity: Quantity(qty)}}})
	if reason != "" {
		t.Fatalf("dispatch for agent %d = (%d, %q), want dispatched", id, s, reason)
	}
	return s
}

// planCover keeps a lane request of the agent for the round, as checkCoverPlan does.
func planCover(w *World, id AgentID, l LaneCover) {
	w.checkCoverPlan(&w.agents[id], &TradePlan{Cover: []LaneCover{l}})
}

// planStore keeps a store request of the agent for the round.
func planStore(w *World, id AgentID, s StoreCover) {
	w.checkCoverPlan(&w.agents[id], &TradePlan{Store: []StoreCover{s}})
}

// planOffer keeps an offer of the agent for the round.
func planOffer(w *World, id AgentID, o CoverOffer) {
	w.checkCoverPlan(&w.agents[id], &TradePlan{Offer: []CoverOffer{o}})
}

// coachLane is the lane request of the tests: any cover at most a loading of 20 000.
var coachLane = LaneCover{Mode: modeCoach, To: regFar, MinCoverBP: 5_000, MaxLoadingBP: 20_000}

// fairOffer is the offer of the tests: fair loading, a store rate of 1%, 80% cover and a
// liability of 10 000.
var fairOffer = CoverOffer{MaxLiability: 10_000, LoadingBP: 10_000, StoreRateBP: 100, CoverBP: 8_000}

// coverEvents returns the CoverFormed and the CoverRound events of the recorder.
func coverEvents(rec *evLog) ([]CoverFormed, []CoverRound) {
	return of[CoverFormed](rec), of[CoverRound](rec)
}

// TestCoverRound_FormsARoutePolicy verifies the worked round of a hired shipment: the sum
// insured, the premium, the maximum payout, the money moved, the policy, the liability and
// the two events.
//
// Worked: 8 grain at the unit value min(1 000, 20 000/10 000 of 1 000) = 1 000 is a sum of
// 8 000. The lane's expected loss is the certain strike of 2 500 bp. At a cover of 8 000 bp
// and a loading of 10 000 the premium is halfEven(8 000 * 2 500 * 8 000 * 10 000 / 10^12)
// = 1 600, the rate halfEven(2 500 * 8 000 * 10 000 / 10^8) = 2 000 bp, and the maximum
// payout halfEven(8 000 * 8 000 / 10 000) = 6 400.
//
// Requirement: FR-INS-021, FR-INS-022, FR-INS-023, FR-INS-030, FR-INS-034, FR-INS-035, FR-INS-036, FR-INS-070, FR-INS-072
func TestCoverRound_FormsARoutePolicy(t *testing.T) {
	t.Parallel()
	w, rec := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	ship := insDispatch(t, w, insHome, 8)
	planCover(w, insHome, coachLane)
	planOffer(w, insBacker, fairOffer)
	insured, under := heldMoney(t, w, insHome), heldMoney(t, w, insBacker)
	w.p33cCoverRound()

	formed, rounds := coverEvents(rec)
	want := CoverFormed{Tick: 0, Policy: 0, Kind: CoverRoute, Insured: insHome, Underwriter: insBacker, Region: regHome, UnderwriterRegion: regHome,
		Shipment: ship, Mode: modeCoach, To: regFar, Sum: 8_000, MaxPayout: 6_400, Premium: 1_600, RateBP: 2_000, CoverBP: 8_000}
	if !reflect.DeepEqual(formed, []CoverFormed{want}) {
		t.Fatalf("CoverFormed = %+v, want one %+v", formed, want)
	}
	wantRound := CoverRound{Tick: 0, Region: regHome, Requests: 1, Offers: 1, Candidates: 1, Formed: 1, Sum: 8_000, Premium: 1_600}
	if !reflect.DeepEqual(rounds, []CoverRound{wantRound}) {
		t.Errorf("CoverRound = %+v, want one %+v", rounds, wantRound)
	}
	in := w.insurance
	got := []int64{heldMoney(t, w, insHome), heldMoney(t, w, insBacker), int64(len(in.open)), in.liabilityOf(insBacker), in.policiesOf(insHome)}
	wantState := []int64{insured - 1_600, under + 1_600, 1, 6_400, 1}
	if !slices.Equal(got, wantState) || !in.covered(ship) {
		t.Errorf("insured's money, underwriter's money, open policies, liability, insured's policies = %v (covered %v), want %v and covered", got, in.covered(ship), wantState)
	}
	auditNow(t, w)
}

// TestCoverRound_FormsAStorePolicy verifies a store policy: a share of the held value, priced at
// the store rate, for the one tick, and ended at P8.1b of the tick.
//
// Worked: the household holds 10 grain at a unit value of 1 000, so the held value is
// 10 000 and a share of 5 000 bp is a sum of 5 000. At a store rate of 100 bp the premium
// is halfEven(5 000 * 100 / 10 000) = 50 and the maximum payout at 8 000 bp is 4 000.
//
// Requirement: FR-INS-021, FR-INS-023, FR-INS-034, FR-INS-072
func TestCoverRound_FormsAStorePolicy(t *testing.T) {
	t.Parallel()
	c := insConfig(&InsureTerms{StoreShareBP: 5_000, MinCoverBP: 5_000, MaxStoreRateBP: 200}, defaultUnderwrite())
	c.Modules.Storage = &Storage{Base: &Capacity{Mass: 100_000, Bulk: 100_000}, Classes: []StoreClass{{ID: "bin", Mass: 4, Bulk: 3}}}
	w, rec := newInsWorld(t, c, nil)
	stockAgent(t, w, insHome, 10, 0)
	planStore(w, insHome, StoreCover{ShareBP: 5_000, MinCoverBP: 5_000, MaxStoreRateBP: 200})
	planOffer(w, insBacker, fairOffer)
	w.p33cCoverRound()
	formed, _ := coverEvents(rec)
	want := CoverFormed{Tick: 0, Policy: 0, Kind: CoverStore, Insured: insHome, Underwriter: insBacker, Region: regHome, UnderwriterRegion: regHome,
		Shipment: NoShipment, Mode: NoMode, To: NoRegion, Sum: 5_000, MaxPayout: 4_000, Premium: 50, RateBP: 100, CoverBP: 8_000}
	if len(formed) != 1 || formed[0] != want {
		t.Fatalf("CoverFormed = %+v, want one %+v", formed, want)
	}
	if in := w.insurance; len(in.stores) != 1 || len(in.open) != 0 || in.liabilityOf(insBacker) != 4_000 {
		t.Errorf("store policies %d, route policies %d, liability %d, want 1, 0, 4000", len(in.stores), len(in.open), in.liabilityOf(insBacker))
	}
	w.p81bSettleStores() // P8.1b of the same tick: the store policy ends with no strike
	if in := w.insurance; len(in.stores) != 0 || in.liabilityOf(insBacker) != 0 {
		t.Errorf("after P8.1b: store policies %d, liability %d, want 0 and 0", len(in.stores), in.liabilityOf(insBacker))
	}
	auditNow(t, w)
}

// TestSum_ByShape verifies the sum insured of each shape of a shipment (spec-insurance.md
// 6.3.2): a hired shipment is its lines, a self-carried trip to oneself is the goods the
// traveller holds, and a self-carried shipment with lines is both.
//
// Worked: the household holds 10 grain at 1 000. A trip on foot to oneself is 10 000. A
// hired shipment of 3 grain is the lines, 3 000: the 10 grain at home are not insured. A
// foot shipment of 4 grain to another is the lines 4 000 plus the 6 grain left on the
// traveller, 6 000, which is 10 000.
//
// Requirement: FR-INS-021
func TestSum_ByShape(t *testing.T) {
	t.Parallel()
	footLane := LaneCover{Mode: modeFoot, To: regFar, MinCoverBP: 1, MaxLoadingBP: MaxLoadingBound}
	tests := []struct {
		name     string
		lane     LaneCover
		dispatch func(t *testing.T, w *World)
		want     int64
	}{
		{"self-carried to oneself", footLane, func(t *testing.T, w *World) {
			if _, reason := ask(w, insHome, 0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}); reason != "" {
				t.Fatalf("trip to oneself = %q, want dispatched", reason)
			}
		}, 10_000},
		{"self-carried with lines", footLane, func(t *testing.T, w *World) {
			r := ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: insFar, Lines: []Line{{Commodity: commodityOf(t, w, "grain"), Quantity: 4}}}
			if _, reason := ask(w, insHome, 0, r); reason != "" {
				t.Fatalf("foot shipment with lines = %q, want dispatched", reason)
			}
		}, 10_000},
		{"hired lines only", coachLane, func(t *testing.T, w *World) { insDispatch(t, w, insHome, 3) }, 3_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := insConfig(defaultInsure(), defaultUnderwrite())
			c.Modules.Risk.Routes = append(c.Modules.Risk.Routes, riskFoot("a_home", "b_far", 10_000, 2_500, false))
			w, rec := newInsWorld(t, c, nil)
			tt.dispatch(t, w)
			planCover(w, insHome, tt.lane)
			planOffer(w, insBacker, CoverOffer{MaxLiability: 20_000, LoadingBP: 10_000, CoverBP: 10_000})
			w.p33cCoverRound()
			if formed, _ := coverEvents(rec); len(formed) != 1 || int64(formed[0].Sum) != tt.want {
				t.Errorf("CoverFormed = %+v, want one policy of sum %d", formed, tt.want)
			}
		})
	}
}
