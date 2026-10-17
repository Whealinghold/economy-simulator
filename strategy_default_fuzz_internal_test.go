package economysim

import (
	"math"
	"testing"
)

// FuzzTrade verifies, for views built from any int64 values, that
// DefaultStrategy.Trade and Work never panic, that the plan has at most one
// order per commodity in strictly ascending commodity order, that every
// quantity is at least 1 and at most MaxQuantity, that every limit is in
// [1, MaxPrice], that the bids reserve no more than the money, and that the
// plan has no discard (D96, FR-AGT-039, 044, 051).
//
// Requirement: FR-AGT-039, FR-AGT-044, FR-AGT-059
func FuzzTrade(f *testing.F) {
	// Seeds: the example of 16.2; the overflowing view of the robustness test;
	// negative values; the largest values.
	f.Add(int64(1), int64(8000), int64(0), int64(4), int64(0), int64(3), int64(300), int64(1000), int64(1), int64(2), int64(1), uint64(1<<63))
	f.Add(int64(math.MaxInt64), int64(math.MaxInt64), int64(1), int64(0), int64(0), int64(MaxQuantity), int64(MaxPrice), int64(5000), int64(3), int64(MaxQuantity), int64(1), uint64(math.MaxUint64))
	f.Add(int64(-1), int64(-1), int64(-1), int64(-1), int64(-1), int64(-1), int64(-1), int64(-1), int64(-1), int64(-1), int64(-1), uint64(0))
	f.Add(int64(40), int64(0), int64(2), int64(1<<40), int64(1<<40), int64(3), int64(105), int64(0), int64(1), int64(2), int64(10000), uint64(12345))
	f.Fuzz(func(t *testing.T, weight, money, hunger, stock, expiring, upkeep, centre, width, startsIn, inQty, effective int64, word uint64) {
		bounds := Bounds{Floor: 1, Ceiling: MaxPrice}
		v := &AgentView{
			Weight: weight, Money: money, Hunger: hunger, StartsIn: startsIn,
			HasRatio: true, Ratio: money, TargetRatio: stock, Window: 10, WorkingTicks: expiring,
			Upkeep: []UpkeepRow{{Commodity: 0, Quantity: upkeep}},
			Commodities: []CommodityView{
				{Commodity: 0, Stock: stock, Expiring: expiring, Belief: Belief{Centre: Price(centre), Width: width}, Bounds: bounds},
				{Commodity: 1, Stock: expiring, Belief: Belief{Centre: Price(centre), Width: width}, Bounds: bounds, Tool: stock%2 == 0},
				{Commodity: 2, Stock: stock, Belief: Belief{Centre: Price(centre), Width: width}, Bounds: bounds},
			},
			Recipes: []RecipeView{{
				Recipe: 0, Duration: 1 + hunger%5, Effective: effective, MaxExecutions: stock,
				Inputs:  []RecipeLine{{Commodity: 0, Quantity: inQty}},
				Tools:   []RecipeTool{{Commodity: 1, WearPerUse: 1, Durability: 3}},
				Outputs: []RecipeOutput{{Commodity: 2, Quantity: inQty, CostShare: 10000}},
			}},
			Tunables: testTunables(),
		}
		p, _ := runTrade(v, []uint64{word, word >> 1, word << 1})
		var spent int64
		last := CommodityID(-1)
		for _, in := range p.Intents {
			if in.Commodity <= last || in.Quantity < 1 || in.Quantity > MaxQuantity || in.Limit < 1 || in.Limit > MaxPrice {
				t.Fatalf("Trade(%+v) gave the bad intent %+v after commodity %d", v, in, last)
			}
			last = in.Commodity
			if in.Side == Bid {
				spent += int64(in.Quantity) * int64(in.Limit)
				if spent < 0 || spent > max(money, 0) {
					t.Fatalf("Trade(%+v) bids reserve %d, more than the money %d", v, spent, money)
				}
			}
		}
		if len(p.Discards) != 0 {
			t.Fatalf("Trade(%+v) discarded %+v", v, p.Discards)
		}
		(DefaultStrategy{}).Work(v, nil)
	})
}
