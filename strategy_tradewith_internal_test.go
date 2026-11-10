package economysim

import (
	"reflect"
	"testing"
)

// runTradeWith is runTrade through the seam with a planned trip.
func runTradeWith(v *AgentView, words []uint64, trip []TripInput) (TradePlan, int) {
	src := &wordSource{words: words}
	var p TradePlan
	DefaultStrategy{}.TradeWith(v, newRand(src), &p, trip)
	return p, src.n
}

// TestTradeWith_NilIsTrade verifies that Trade is TradeWith with a nil trip: the
// same plan and the same number of words drawn, at the money of the worked
// examples of spec-agents.md 16.1 and 16.2.
//
// Requirement: FR-MER-002
func TestTradeWith_NilIsTrade(t *testing.T) {
	t.Parallel()
	for _, money := range []int64{50, 120, 2_500, 8_000} {
		want, wantDrawn := runTrade(exampleView(money), exampleWords())
		got, drawn := runTradeWith(exampleView(money), exampleWords(), nil)
		if !reflect.DeepEqual(got, want) || drawn != wantDrawn {
			t.Errorf("TradeWith(nil) with money %d = %+v after %d words, want Trade's %+v after %d", money, got, drawn, want, wantDrawn)
		}
	}
}

// TestTradeWith_TripInputsAreTierTwo verifies the planned trip of
// spec-merchants.md 3.4.2: the sum of the entries for a commodity is Job_c, the
// planned recipe is none, the order's purpose is the first entry's, and the
// inputs are tier 2, bought before the buffer and not held back by the saving
// rule.
//
// 16.1 with money 2 500: upkeep cost 1 000, reserve money 3 000, so the money is
// below the reserve and there is no buffer. Fuel (tier 1) buys 1 at 105, 2 395
// left. The trip needs 5 ore (3 + 2 in two entries, the first of purpose Input)
// at 195: 975, affordable, so the bid is 5 units of purpose Input. The recipe
// make_item is not planned, so the ore of its input is not bought for it.
//
// Requirement: FR-MER-006
func TestTradeWith_TripInputsAreTierTwo(t *testing.T) {
	t.Parallel()
	trip := []TripInput{{Commodity: xOre, Quantity: 3, Purpose: PurposeInput}, {Commodity: xOre, Quantity: 2, Purpose: PurposeTool}}
	got, drawn := runTradeWith(exampleView(2_500), exampleWords(), trip)
	want := []Intent{
		{Commodity: xFuel, Side: Bid, Quantity: 1, Limit: 105, Target: 3, Reserve: 1, Purpose: PurposeUpkeep},
		{Commodity: xOre, Side: Bid, Quantity: 5, Limit: 195, Target: 5, Purpose: PurposeInput},
		{Commodity: xItem, Side: Ask, Quantity: 1, Limit: 900, Target: 0, Purpose: PurposeOther},
	}
	if !reflect.DeepEqual(got.Intents, want) {
		t.Errorf("TradeWith(trip) intents = %+v, want %+v", got.Intents, want)
	}
	if _, base := runTrade(exampleView(2_500), exampleWords()); drawn != base {
		t.Errorf("TradeWith(trip) took %d words, want %d as Trade takes for the same commodities", drawn, base)
	}
}

// TestTradeWith_EmptyTripPlansNothing verifies that a non-nil empty trip is a
// trip with no input: the planned recipe is none, so the ore its input is not
// bought, and the surplus is offered (spec-merchants.md 3.4.3).
//
// Requirement: FR-MER-006
func TestTradeWith_EmptyTripPlansNothing(t *testing.T) {
	t.Parallel()
	got, _ := runTradeWith(exampleView(2_500), exampleWords(), []TripInput{})
	for _, in := range got.Intents {
		if in.Commodity == xOre {
			t.Errorf("TradeWith(empty trip) bid for ore: %+v, want none (no recipe is planned)", in)
		}
	}
	if len(got.Intents) == 0 || got.Intents[len(got.Intents)-1].Side != Ask {
		t.Errorf("TradeWith(empty trip) intents = %+v, want the surplus offered", got.Intents)
	}
}

// TestTradeWith_UpkeepAndTripSum verifies that a commodity that is both upkeep and
// cargo takes the sum of the lines (spec-merchants.md 3.4.2): the staple the
// household eats is also the cargo, so its Job_c adds to the upkeep target and
// the order keeps the purpose of the trip's entry.
//
// The staple is eaten 3 a tick and held 4; with a trip of 6 more the target grows
// by 6, so the bid is larger than without the trip.
//
// Requirement: FR-MER-006
func TestTradeWith_UpkeepAndTripSum(t *testing.T) {
	t.Parallel()
	qtyOf := func(p TradePlan) (Quantity, Purpose) {
		for _, in := range p.Intents {
			if in.Commodity == xStaple && in.Side == Bid {
				return in.Quantity, in.Purpose
			}
		}
		return 0, PurposeOther
	}
	base, _ := runTradeWith(exampleView(8_000), exampleWords(), []TripInput{})
	with, _ := runTradeWith(exampleView(8_000), exampleWords(), []TripInput{{Commodity: xStaple, Quantity: 6, Purpose: PurposeInput}})
	q0, _ := qtyOf(base)
	q1, purpose := qtyOf(with)
	if q1 <= q0 || purpose != PurposeInput {
		t.Errorf("staple bid with a trip of 6 = %d of purpose %v, without = %d, want a larger bid of purpose Input", q1, purpose, q0)
	}
}
