package strategy_test

import (
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// readyValley is the view of the valley merchant with a trip ready to send (150 fuel and
// 6 haul units, one delivery offer of 150 fuel to the remote region by barge) and an
// insurance group whose insured block buys on any lane with a loss: the route's expected
// loss is 1 000 bp and the fuel unit value 200.
func readyValley(withBlock bool) *economysim.AgentView {
	v := valley()
	v.Commodities[0].Stock, v.Commodities[1].Stock = 150, 6
	v.Routes[0].LossBP = 1_000
	if withBlock {
		v.Insurance = &economysim.CoverView{
			Insure: &economysim.InsureTerms{MinLossBP: 1, MinCoverBP: 5_000, MaxLoadingBP: 20_000},
			MinSum: 1, MaxCoverRequests: 4, ValueCapBP: 20_000, Money: v.Money,
			LaneLoss: []int64{1_000}, UnitValue: []int64{fuel: 200, haul: 700},
		}
	}
	return v
}

// TestDefaultCover_CalledFromMerchant verifies that the merchant appends the cover
// request for the lane of the trip it has just planned (spec-insurance.md 10.4): one
// delivery offer of fuel to remote by barge gives one lane request with the block's
// limits, after the offer it covers.
//
// Requirement: FR-INS-063
func TestDefaultCover_CalledFromMerchant(t *testing.T) {
	t.Parallel()
	p, _ := trade(t, merchantStrategy, readyValley(true))
	want := []economysim.LaneCover{{Mode: barge, To: remote, MinCoverBP: 5_000, MaxLoadingBP: 20_000}}
	if len(p.Offers) != 1 || !reflect.DeepEqual(p.Cover, want) {
		t.Errorf("Trade() offers %v, cover %+v, want one delivery offer and the lane request %+v", p.Offers, p.Cover, want)
	}
}

// TestMerchant_NoCoverRequestWithoutBlock verifies that a merchant whose view has no cover
// group makes no cover request and plans exactly as it did before the module existed.
//
// Guard: the merchant calls the default cover rule on every trip; without the nil check
// of the group a merchant of a pack with no cover block would change its plan and move
// the merchant golden.
//
// Requirement: FR-INS-063
func TestMerchant_NoCoverRequestWithoutBlock(t *testing.T) {
	t.Parallel()
	got, gotN := trade(t, merchantStrategy, readyValley(false))
	if len(got.Cover)+len(got.Store)+len(got.Offer) != 0 {
		t.Errorf("Trade() with no cover group appended cover %+v, store %+v, offer %+v, want none", got.Cover, got.Store, got.Offer)
	}
	plain := valley()
	plain.Commodities[0].Stock, plain.Commodities[1].Stock, plain.Routes[0].LossBP = 150, 6, 1_000
	want, wantN := trade(t, merchantStrategy, plain)
	if !reflect.DeepEqual(got, want) || gotN != wantN {
		t.Errorf("Trade() = %+v after %d words, want %+v after %d (the same view)", got, gotN, want, wantN)
	}
}
