package economysim

import (
	"bytes"
	"log/slog"
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// TestViews_CarryNoOtherPolicyNoOfferNoLoading verifies that the cover group of the
// view holds only the fields of spec-insurance.md 10.1: no other agent's policy, no
// offer, no loading of another agent, no count of underwriters. The group is read by
// reflection, so a field added later must be named here.
//
// Guard: an insured that could see the offers would choose its price, and an
// underwriter that could see the requests would price against them (N-106). Without
// the test a convenient field would leak the market for cover into a strategy.
//
// Requirement: FR-INS-060
func TestViews_CarryNoOtherPolicyNoOfferNoLoading(t *testing.T) {
	t.Parallel()
	want := []string{"Underwrite", "Insure", "MinSum", "MaxCoverRequests", "ValueCapBP", "Open", "Money", "Capacity", "Owed", "Policies", "LaneLoss", "UnitValue"}
	typ := reflect.TypeOf(CoverView{})
	var got []string
	for i := range typ.NumField() {
		got = append(got, typ.Field(i).Name)
	}
	if !slices.Equal(got, want) {
		t.Errorf("CoverView fields = %v, want %v", got, want)
	}
}

// TestViews_CoverGroupOnlyForARoleWithABlock verifies that the group is present in the
// Trade view of an agent whose role has a cover block, absent for a role with none, and
// carries the module's figures and a copy of the block.
//
// Worked: the underwriter holds M money at a cap of 20 000 bp, so its capacity is 2M with
// nothing open; the insured's unit values are min(published, 20 000/10 000 of the
// reference): grain 1 000 (the seed price), flour 3 000.
//
// Requirement: FR-INS-060
func TestViews_CoverGroupOnlyForARoleWithABlock(t *testing.T) {
	t.Parallel()
	w, _ := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	t.Run("an insured has the Insure block only", func(t *testing.T) {
		t.Parallel()
		ins := viewOf(t, w, insHome)
		if ins.Insurance == nil || ins.Insurance.Insure == nil || ins.Insurance.Underwrite != nil {
			t.Fatalf("insured's group = %+v, want an Insure block only", ins.Insurance)
		}
		if g := ins.Insurance; g.MinSum != 10 || g.MaxCoverRequests != 4 || g.ValueCapBP != 20_000 || g.Money != ins.Money {
			t.Errorf("insured's figures = %+v, want MinSum 10, MaxCoverRequests 4, ValueCapBP 20000 and the money of the view", g)
		}
		if got := ins.Insurance.UnitValue; len(got) != len(w.cfg.Commodities) || got[flour] != 3_000 || got[grain] != 1_000 {
			t.Errorf("unit values = %v, want flour 3000 and grain 1000 of %d commodities", got, len(w.cfg.Commodities))
		}
		if len(ins.Insurance.LaneLoss) != len(ins.Routes) {
			t.Errorf("LaneLoss has %d rows, want one for each of the %d routes", len(ins.Insurance.LaneLoss), len(ins.Routes))
		}
	})
	t.Run("an underwriter has the Underwrite block and a capacity", func(t *testing.T) {
		t.Parallel()
		uw := viewOf(t, w, insBacker)
		if uw.Insurance == nil || uw.Insurance.Underwrite == nil || uw.Insurance.Insure != nil {
			t.Fatalf("underwriter's group = %+v, want an Underwrite block only", uw.Insurance)
		}
		if got, want := uw.Insurance.Capacity, 2*uw.Money; got != want || uw.Insurance.Open != 0 {
			t.Errorf("underwriter capacity = %d, open %d, want %d and 0", got, uw.Insurance.Open, want)
		}
	})
	t.Run("a role with no block has no group", func(t *testing.T) {
		t.Parallel()
		bare, _ := newInsWorld(t, insConfig(nil, nil), nil)
		if v := viewOf(t, bare, insHome); v.Insurance != nil {
			t.Errorf("a role with no cover block has the group %+v, want nil", v.Insurance)
		}
	})
}

// coverView is a hand-built view of an insured at home with 100 000 money, a route to
// far by coach whose expected loss is lossBP, and unit values grain 1 000 and flour 3 000.
func coverView(in InsureTerms, lossBP int64) *AgentView {
	return &AgentView{
		Agent: 4, Region: regHome, Weight: 1, Money: 100_000,
		Routes: []RouteRow{{To: regFar, Mode: modeCoach, Has: true, Ticks: 2, LossBP: lossBP}},
		Insurance: &CoverView{
			Insure: &in, MinSum: 10, MaxCoverRequests: 4, ValueCapBP: 20_000, Money: 100_000,
			LaneLoss: []int64{lossBP}, UnitValue: []int64{flour: 3_000, grain: 1_000},
		},
	}
}

// coachTrip is a plan with one hired shipment of qty grain to far.
func coachTrip(qty Quantity) *TradePlan {
	return &TradePlan{Shipments: []ShipmentRequest{{Mode: modeCoach, To: regFar, Consignee: 6, Lines: []Line{{Commodity: grain, Quantity: qty}}}}}
}

// TestDefaultCover_RuinBasedRule verifies the default insured of spec-insurance.md 10.2:
// a request for a lane only when MinLossBP is above 0 and the lane's expected loss is at
// least it, and the plan's goods on the lane are worth at least RuinShareBP of the money;
// a lane named twice gets one request; a store request when StoreShareBP is above 0 and
// the agent owns a store; and the request carries the block's cover and loading limits.
//
// Worked: 60 grain at a unit value of 1 000 is 60 000. Against 100 000 money, a ruin share
// of 5 000 bp asks for 50 000, so it is bought; 6 000 bp asks for 60 000 and 7 000 for 70
// 000. 10 000 * 60 000 >= 6 000 * 100 000 is equal, so 6 000 buys and 6 001 does not.
//
// Requirement: FR-INS-061
func TestDefaultCover_RuinBasedRule(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		in    InsureTerms
		loss  int64
		trip  Quantity
		store bool
		want  []LaneCover
		wantS int
	}{
		{"asks no route cover with MinLossBP 0", InsureTerms{MinCoverBP: 5_000, MaxLoadingBP: 20_000}, 500, 60, false, nil, 0},
		{"loss below the minimum", InsureTerms{MinLossBP: 600, MinCoverBP: 5_000, MaxLoadingBP: 20_000}, 500, 60, false, nil, 0},
		{"loss at the minimum buys", InsureTerms{MinLossBP: 500, MinCoverBP: 5_000, MaxLoadingBP: 20_000}, 500, 60, false, []LaneCover{{Mode: modeCoach, To: regFar, MinCoverBP: 5_000, MaxLoadingBP: 20_000}}, 0},
		{"cargo at the ruin share buys", InsureTerms{MinLossBP: 1, RuinShareBP: 6_000, MinCoverBP: 1, MaxLoadingBP: 7}, 500, 60, false, []LaneCover{{Mode: modeCoach, To: regFar, MinCoverBP: 1, MaxLoadingBP: 7}}, 0},
		{"cargo below the ruin share", InsureTerms{MinLossBP: 1, RuinShareBP: 6_001, MinCoverBP: 1}, 500, 60, false, nil, 0},
		{"store request with a store", InsureTerms{MinCoverBP: 4_000, StoreShareBP: 3_000, MaxStoreRateBP: 90}, 0, 0, true, nil, 1},
		{"no store request without a store", InsureTerms{MinCoverBP: 4_000, StoreShareBP: 3_000, MaxStoreRateBP: 90}, 0, 0, false, nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := coverView(tt.in, tt.loss)
			if tt.store {
				v.Room.Stores = []StoreCount{{Class: 0, Count: 1}}
			}
			p := coachTrip(tt.trip)
			if tt.trip == 0 {
				p = &TradePlan{}
			}
			DefaultCover(v, p)
			if !slices.Equal(p.Cover, tt.want) || len(p.Store) != tt.wantS {
				t.Errorf("DefaultCover plan: cover %+v, store %+v, want %+v and %d store request(s)", p.Cover, p.Store, tt.want, tt.wantS)
			}
		})
	}
}

// TestDefaultCover_OneRequestForTwoShipmentsOnALane verifies that two shipments and a
// delivery offer on one lane are valued together and asked once.
//
// Worked: 30 grain and 30 grain on the lane are 60 000 together, which at a ruin share of
// 6 000 bp of 100 000 money buys; each alone (30 000) would not.
//
// Requirement: FR-INS-061
func TestDefaultCover_OneRequestForTwoShipmentsOnALane(t *testing.T) {
	t.Parallel()
	v := coverView(InsureTerms{MinLossBP: 1, RuinShareBP: 6_000, MinCoverBP: 1, MaxLoadingBP: 7}, 500)
	p := coachTrip(30)
	p.Shipments = append(p.Shipments, p.Shipments[0])
	DefaultCover(v, p)
	if len(p.Cover) != 1 {
		t.Errorf("DefaultCover gave %d lane requests for two shipments of one lane, want 1 (60 000 reaches 6 000 bp of 100 000)", len(p.Cover))
	}
	q := &TradePlan{Offers: []DeliveryOffer{{Mode: modeCoach, To: regFar, Commodity: grain, Quantity: 60, Limit: 1}}}
	DefaultCover(v, q)
	if len(q.Cover) != 1 {
		t.Errorf("DefaultCover gave %d lane requests for a delivery offer of 60 grain, want 1", len(q.Cover))
	}
}

// TestDefaultOffer_ShareOfFreeCapacity verifies the default underwriter of
// spec-insurance.md 10.3: a share of the room its own bids left, at its block's terms.
//
// Worked: money 10 000, bids 2 000, so 8 000 is free of bids; the cap of 20 000 bp is
// 16 000, less 3 000 open is 13 000; a share of 5 000 bp offers 6 500. With bids of
// 10 000 nothing is free (0, less 3 000 open, floors at 0) and the offer is below MinSum.
//
// Requirement: FR-INS-062
func TestDefaultOffer_ShareOfFreeCapacity(t *testing.T) {
	t.Parallel()
	u := defaultUnderwrite()
	v := &AgentView{Money: 10_000, Insurance: &CoverView{Underwrite: u, MinSum: 10, Money: 10_000, Open: 3_000}}
	tests := []struct {
		name string
		bids int64
		want CoverOffer
		ok   bool
	}{
		{"half of the room after the bids", 2_000, CoverOffer{MaxLiability: 6_500, LoadingBP: 10_000, StoreRateBP: 100, CoverBP: 8_000}, true},
		{"bids take all the money", 10_000, CoverOffer{}, false},
		{"bids above the money", 20_000, CoverOffer{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := DefaultOffer(v, tt.bids)
			if got != tt.want || ok != tt.ok {
				t.Errorf("DefaultOffer(view, %d) = %+v, %v, want %+v, %v", tt.bids, got, ok, tt.want, tt.ok)
			}
		})
	}
	small := &AgentView{Insurance: &CoverView{Underwrite: u, MinSum: 6_501, Money: 10_000, Open: 3_000}}
	if o, ok := DefaultOffer(small, 2_000); ok {
		t.Errorf("DefaultOffer below MinSum = %+v, true, want no offer (6 500 < 6 501)", o)
	}
}

// TestDefaultOffer_NoOfferWhileOwing verifies that an underwriter that owes a payout
// claim offers nothing, because the round would refuse it in_arrears.
//
// Requirement: FR-INS-062
func TestDefaultOffer_NoOfferWhileOwing(t *testing.T) {
	t.Parallel()
	v := &AgentView{Insurance: &CoverView{Underwrite: defaultUnderwrite(), MinSum: 10, Money: 10_000, Owed: 1}}
	if o, ok := DefaultOffer(v, 0); ok {
		t.Errorf("DefaultOffer while owing = %+v, true, want no offer", o)
	}
	if o, ok := DefaultOffer(&AgentView{}, 0); ok {
		t.Errorf("DefaultOffer with no group = %+v, true, want no offer", o)
	}
}

// TestDefaultCover_AppendsTheOfferOfAnUnderwriter verifies that DefaultCover appends the
// underwriter's offer from the bids of the plan, so a wrapper needs one call.
//
// Requirement: FR-INS-063
func TestDefaultCover_AppendsTheOfferOfAnUnderwriter(t *testing.T) {
	t.Parallel()
	v := &AgentView{Money: 10_000, Insurance: &CoverView{Underwrite: defaultUnderwrite(), MinSum: 10, Money: 10_000}}
	p := &TradePlan{Intents: []Intent{{Side: Bid, Limit: 100, Quantity: 20}}}
	DefaultCover(v, p)
	// 10 000 less bids 2 000 is 8 000; cap 20 000 bp is 16 000; half is 8 000.
	if len(p.Offer) != 1 || p.Offer[0].MaxLiability != 8_000 {
		t.Errorf("DefaultCover offers = %+v, want one of MaxLiability 8000", p.Offer)
	}
}

// coverFallback is a fallback strategy that appends a lane request.
type coverFallback struct{}

func (coverFallback) Work(*AgentView, *rand.Rand) JobRequest { return JobRequest{NoJob: true} }
func (coverFallback) Trade(_ *AgentView, _ *rand.Rand, p *TradePlan) {
	p.Cover = append(p.Cover, LaneCover{Mode: modeCoach, To: regFar, MinCoverBP: 1})
	p.Borrow = append(p.Borrow, CreditRequest{Principal: 1, Term: 1})
}
func (coverFallback) Revise(o *OutcomeView, _ *rand.Rand) Belief { return o.Belief }

// TestDirected_KeepsCoverRequests verifies that the Directed strategy keeps the cover
// requests of its fallback when a directive replaces its credit request: spec-insurance.md
// 10.5, a player insures by the fallback's rules.
//
// Guard: directCredit replaces only the credit fields; a change that rebuilt the plan
// would drop the cover fields silently.
//
// Requirement: FR-INS-063
func TestDirected_KeepsCoverRequests(t *testing.T) {
	t.Parallel()
	v := &AgentView{HasDirective: true, Directive: Directive{Borrow: &CreditRequest{Principal: 5, Term: 2}}}
	p := &TradePlan{}
	Directed{Fallback: coverFallback{}}.Trade(v, nil, p)
	if len(p.Cover) != 1 || len(p.Borrow) != 1 || p.Borrow[0].Principal != 5 {
		t.Errorf("Directed plan: cover %+v, borrow %+v, want the fallback's lane request kept and the directive's borrow", p.Cover, p.Borrow)
	}
}

// TestAdmission_OffCostsNothing verifies that a cover request or offer made while the
// module is off costs nothing: it is a Debug line "cover refused" with the reason
// insurance_off, and a run that makes them writes the same checkpoint as one that makes
// none.
//
// Requirement: FR-INS-016
func TestAdmission_OffCostsNothing(t *testing.T) {
	t.Parallel()
	run := func(plan func(v *AgentView, p *TradePlan)) ([]byte, string) {
		var buf bytes.Buffer
		lg := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		w, _ := newInsWorld(t, shipConfig(), plan, WithLogger(lg))
		step(t, w, 3)
		ck, err := w.Checkpoint()
		if err != nil {
			t.Fatalf("Checkpoint() error = %v, want nil", err)
		}
		return errVal(ck.MarshalBinary()), buf.String()
	}
	ask := func(v *AgentView, p *TradePlan) {
		p.Cover = append(p.Cover, LaneCover{Mode: modeCoach, To: regFar, MinCoverBP: 1})
		p.Store = append(p.Store, StoreCover{ShareBP: 1, MinCoverBP: 1})
		p.Offer = append(p.Offer, CoverOffer{MaxLiability: 1, CoverBP: 1})
	}
	with, logged := run(ask)
	without, _ := run(nil)
	if !bytes.Equal(with, without) {
		t.Error("a run with cover requests while the module is off writes another checkpoint than one without")
	}
	n, off := strings.Count(logged, "cover refused"), strings.Count(logged, "reason=insurance_off")
	if n == 0 || n != off || n%3 != 0 {
		t.Errorf("Debug lines \"cover refused\" = %d, with insurance_off %d, want all of them a multiple of 3 (a request, a store request and an offer each time)", n, off)
	}
}

// TestUnitValue_ReadsPriceBeforePosition verifies the unit value of spec-insurance.md
// 6.2.1: the price a market published at its last clearing, capped at ValueCapBP of the
// effective reference price, the reference price while the market has never traded.
//
// Worked: the cap is 20 000 bp, so grain (reference 1 000) is capped at 2 000. Before any
// trade the value is the seed price 1 000. After a clearing that published 5 000 it is
// min(5 000, 2 000) = 2 000, capped; a price of 1 500 is below the cap and is the value.
//
// Guard: the view's UnitValue and the round's sum must read the price left by P4.1 of the
// previous tick, not a price re-derived from orders; a wrong read changes every sum.
//
// Requirement: FR-INS-020
func TestUnitValue_ReadsPriceBeforePosition(t *testing.T) {
	t.Parallel()
	w, _ := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	i := int(regHome)*len(w.cfg.Commodities) + int(grain)
	if v, capped := w.unitValue(regHome, grain); v != 1_000 || capped {
		t.Errorf("unitValue before any trade = %d, capped %v, want 1000 (the reference price), false", v, capped)
	}
	for _, tt := range []struct {
		published, want int64
		capped          bool
	}{{5_000, 2_000, true}, {1_500, 1_500, false}} {
		w.markets.windows[i].Push(tt.published)
		w.markets.reports[i].Price = Price(tt.published)
		if v, capped := w.unitValue(regHome, grain); v != tt.want || capped != tt.capped {
			t.Errorf("unitValue after a clearing at %d = %d, capped %v, want %d, %v", tt.published, v, capped, tt.want, tt.capped)
		}
	}
	if got := viewOf(t, w, insHome).Insurance.UnitValue[grain]; got != 1_500 {
		t.Errorf("view UnitValue[grain] = %d, want 1500 (the last published price, under the cap)", got)
	}
}
