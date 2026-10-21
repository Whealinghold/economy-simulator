package economysim

import (
	"math/bits"
	"math/rand/v2"
	"reflect"
	"testing"
)

// wordSource is a rand.Source that returns the words it was given in order and
// then 0, and counts the words taken.
type wordSource struct {
	words []uint64
	n     int
}

func (s *wordSource) Uint64() uint64 {
	var w uint64
	if s.n < len(s.words) {
		w = s.words[s.n]
	}
	s.n++
	return w
}

func newRand(s *wordSource) *rand.Rand { return rand.New(s) }

// wordFor returns a word with which beliefs.Draw picks the k-th price (from 0)
// of a range of n prices: the high word of word * n is k. It needs 0 <= k < n.
func wordFor(k, n uint64) uint64 {
	if k == 0 {
		return 0
	}
	q, _ := bits.Div64(k, 0, n)
	return q + 1
}

// Commodity IDs of the examples of spec-agents.md 16.1 and 16.2, in canonical
// order (fuel before staple, as 16.2 allocates).
const (
	xFuel CommodityID = iota
	xStaple
	xOre
	xItem
)

// exampleView is the household of 16.1: upkeep staple 3 and fuel 1, one recipe
// make_item (duration 1, ore 2 into item 1, yield 10 000), beliefs of the
// centre staple 300, fuel 100, ore 200 and item 900, holding staple 4, item 1,
// free to start a job (StartsIn 1). The belief widths give ranges that contain
// the limit draws of the example: fuel 90 to 110, staple 270 to 330, ore 180 to
// 220; item has a range of one price.
func exampleView(money int64) *AgentView {
	win := PriceWindowView{Position: 5000, HasWindow: true}
	fuel, staple := cview(xFuel, 0, 100, 1000), cview(xStaple, 4, 300, 1000)
	fuel.Window, staple.Window = win, win
	return &AgentView{
		Weight: 1, Money: money, StartsIn: 1,
		Upkeep:      []UpkeepRow{{Commodity: xFuel, Quantity: 1}, {Commodity: xStaple, Quantity: 3}},
		Commodities: []CommodityView{fuel, staple, cview(xOre, 0, 200, 1000), cview(xItem, 1, 900, 0)},
		Recipes: []RecipeView{{
			Recipe: 0, Duration: 1, Effective: 10000, MaxExecutions: 1,
			Inputs:  []RecipeLine{{Commodity: xOre, Quantity: 2}},
			Outputs: []RecipeOutput{{Commodity: xItem, Quantity: 1, CostShare: 10000}},
		}},
		Tunables: testTunables(),
	}
}

// exampleWords are the words that draw fuel 105 (of 90 to 110), staple 310 (of
// 270 to 330) and ore 195 (of 180 to 220), and a word for the item ask.
func exampleWords() []uint64 {
	return []uint64{wordFor(15, 21), wordFor(40, 61), wordFor(15, 41), 0}
}

func runTrade(v *AgentView, words []uint64) (TradePlan, int) {
	src := &wordSource{words: words}
	var p TradePlan
	DefaultStrategy{}.Trade(v, newRand(src), &p)
	return p, src.n
}

// TestDefaultStrategyTrade_WorkedExamples verifies the trade phases of
// spec-agents.md 16.1 and 16.2. The limit draws are 105 (fuel), 310 (staple),
// 195 (ore).
//
// 16.1, money 2 500: planned recipe make_item (net 1 * 900 - 2 * 200 = 500).
// UpkeepCost = 3 * 300 + 1 * 100 = 1 000, reserve money 3 000. Tier 1: fuel 1 at
// 105, 2 395 left. Tier 2: ore 2 at 195 = 390, 2 005 left. The excess over 3 000
// is 0, so the cap is 0 and no buffer is bought; the ask is the 1 item.
//
// 16.2, money 8 000: after 105 and 390, 7 505 is left; e = 4 505, H = 10 * 1 000,
// cap = 5 000 * (4 505 * 10 000 / 14 505 = 3 105) / 10 000 = 1 552. buyshare =
// 2 000 + 8 000 * 5 000 / 10 000 = 6 000: fuel floor(3 * 0.6) = 1 at 105 (cap
// 1 447 left), staple floor(8 * 0.6) = 4 at 310 = 1 240 (cap 207 left).
//
// Requirement: FR-AGT-038, FR-AGT-039, FR-AGT-040, FR-AGT-041, FR-AGT-042, FR-AGT-043, FR-AGT-048
func TestDefaultStrategyTrade_WorkedExamples(t *testing.T) {
	t.Parallel()
	ask := Intent{Commodity: xItem, Side: Ask, Quantity: 1, Limit: 900, Target: 1, Purpose: PurposeOther}
	tests := []struct {
		name  string
		money int64
		want  []Intent
	}{
		{"16.1 a poor household buys no buffer", 2500, []Intent{
			{Commodity: xFuel, Side: Bid, Quantity: 1, Limit: 105, Target: 3, Reserve: 1, Purpose: PurposeUpkeep},
			{Commodity: xOre, Side: Bid, Quantity: 2, Limit: 195, Target: 2, Purpose: PurposeInput},
			ask,
		}},
		{"16.2 a richer household buys the buffer", 8000, []Intent{
			{Commodity: xFuel, Side: Bid, Quantity: 2, Limit: 105, Target: 3, Reserve: 1, Purpose: PurposeUpkeep},
			{Commodity: xStaple, Side: Bid, Quantity: 4, Limit: 310, Target: 9, Reserve: 3, Purpose: PurposeUpkeep},
			{Commodity: xOre, Side: Bid, Quantity: 2, Limit: 195, Target: 2, Purpose: PurposeInput},
			ask,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, drawn := runTrade(exampleView(tt.money), exampleWords())
			if !reflect.DeepEqual(got.Intents, tt.want) {
				t.Errorf("Trade() intents = %+v, want %+v", got.Intents, tt.want)
			}
			if len(got.Discards) != 0 {
				t.Errorf("Trade() discards = %+v, want none", got.Discards)
			}
			// Four commodities are candidates (fuel, staple and ore bid, item ask).
			if drawn != 4 {
				t.Errorf("Trade() took %d words, want 4", drawn)
			}
		})
	}
}

// combinedView is one commodity that is both upkeep (3 a household) and the
// input of the planned recipe (2 an execution), at a belief centre of 105 with
// a range of one price: the wants are T1 = 3, T2 = 2 and T3 = 9 (keep = 3 +
// 3 * 3 + 2 = 14 against a stock of 0). The recipe makes an item worth 900, so
// its net is 900 - 2 * 105 = 690 and it is planned.
func combinedView(weight, upkeepPer, money int64) *AgentView {
	return &AgentView{
		Weight: weight, Money: money, StartsIn: 1,
		Upkeep:      []UpkeepRow{{Commodity: xFuel, Quantity: upkeepPer}},
		Commodities: []CommodityView{cview(xFuel, 0, 105, 0), cview(xItem, 0, 900, 0)},
		Recipes: []RecipeView{{
			Recipe: 0, Duration: 1, Effective: 10000, MaxExecutions: 1,
			Inputs:  []RecipeLine{{Commodity: xFuel, Quantity: 2}},
			Outputs: []RecipeOutput{{Commodity: xItem, Quantity: 1, CostShare: 10000}},
		}},
		Tunables: testTunables(),
	}
}

// TestDefaultStrategyTrade_TierCombination verifies the ruled combination of the
// tiers of one order (D98, spec-agents.md 7.5.3): every tier buys at the order's
// limit, only the first tier to buy may have the limit lowered, and a later tier
// that cannot pay one unit at the order's limit is dropped, never lowered.
//
//   - Money 50, limit 105: tier 1 (3 units) cannot pay one unit, is lowered to
//     its money and buys 1 unit at 50, leaving 0; no later tier buys.
//   - Money 250, limit 105, T1 = 3, T2 = 2: tier 1 buys floor(250 / 105) = 2 at
//     105 = 210, leaving 40; tier 2 cannot pay 105 and is dropped, not lowered to
//     40. The order is 2 units at 105 (reserving 210 of 250). The rejected
//     least-limit rule would have made it 3 units at 40.
//   - Money 100 000 buys everything: tier 1 3 + tier 2 2 + the buffer.
//
// Requirement: FR-AGT-040, FR-AGT-041, FR-AGT-044
func TestDefaultStrategyTrade_TierCombination(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		money     int64
		wantQty   Quantity
		wantLimit Price
	}{
		{"the first buyer is lowered to its money", 50, 1, 50},
		{"a later tier that cannot pay is dropped not lowered", 250, 2, 105},
		{"exactly the first tier", 315, 3, 105},
		{"a later tier buys at the limit", 525, 5, 105},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, _ := runTrade(combinedView(1, 3, tt.money), nil)
			if len(got.Intents) != 1 {
				t.Fatalf("Trade() = %+v, want one bid", got.Intents)
			}
			in := got.Intents[0]
			if in.Quantity != tt.wantQty || in.Limit != tt.wantLimit || in.Side != Bid {
				t.Errorf("Trade() order = %d at %d (%v), want %d at %d", in.Quantity, in.Limit, in.Side, tt.wantQty, tt.wantLimit)
			}
			if int64(in.Quantity)*int64(in.Limit) > tt.money {
				t.Errorf("Trade() reserves %d * %d, more than the money %d", in.Quantity, in.Limit, tt.money)
			}
		})
	}
}

// TestDefaultStrategyTrade_NotInvariantInWeight pins what is not invariant in
// the weight, so that the claim is not re-added: a limit lowered for a cohort
// of 1 and not of 7, a quantity bought that is not w times, an output target
// that is a ceiling over the whole product, and a halt threshold that differs.
//
//   - Money 50 against a want of 1 at 105: lowered to 50. Money 350, w 7, want 7:
//     floor(350 / 105) = 3 units at 105, not lowered.
//   - Money 250, want 3: 2 units. Money 1 750, w 7, want 21: floor(1 750 / 105) =
//     16 units, not 7 * 2 = 14.
//   - Duration 3, output 1, cover 1: the output target is ceil(1 / 3) = 1 at w 1
//     and ceil(7 / 3) = 3 at w 7, not 7.
//   - The halt: at w 1 the target is 1 and a stock of 2 is not over 3 * 1; at w 7
//     the target is 3 and a stock of 14 is over 3 * 3 = 9.
//
// Guard: FR-AGT-052 holds on the pure rules only in the conditions of 7.5; a
// reader who removed the divergences would claim an exactness the rules do not have.
func TestDefaultStrategyTrade_NotInvariantInWeight(t *testing.T) {
	t.Parallel()
	buy := func(w, per, money int64) Intent {
		v := &AgentView{
			Weight: w, Money: money, StartsIn: 1,
			Upkeep:      []UpkeepRow{{Commodity: xFuel, Quantity: per}},
			Commodities: []CommodityView{cview(xFuel, 0, 105, 0)},
			Tunables:    testTunables(),
		}
		got, _ := runTrade(v, nil)
		if len(got.Intents) != 1 {
			t.Fatalf("Trade(w %d, money %d) = %+v, want one bid", w, money, got.Intents)
		}
		return got.Intents[0]
	}
	for _, tt := range []struct {
		name        string
		w, per, mon int64
		qty         Quantity
		limit       Price
	}{
		{"w 1 money 50 lowered", 1, 1, 50, 1, 50},
		{"w 7 money 350 not lowered", 7, 1, 350, 3, 105},
		{"w 1 money 250", 1, 3, 250, 2, 105},
		{"w 7 money 1750", 7, 3, 1750, 16, 105},
	} {
		if in := buy(tt.w, tt.per, tt.mon); in.Quantity != tt.qty || in.Limit != tt.limit {
			t.Errorf("%s: order = %d at %d, want %d at %d", tt.name, in.Quantity, in.Limit, tt.qty, tt.limit)
		}
	}
	maker := func(w, stock int64) *AgentView {
		return &AgentView{
			Weight: w, Money: 1000, StartsIn: 1,
			Commodities: []CommodityView{cview(xItem, stock, 100, 0)},
			Recipes: []RecipeView{{
				Recipe: 0, Duration: 3, Effective: 10000, MaxExecutions: 1,
				Outputs: []RecipeOutput{{Commodity: xItem, Quantity: 1, CostShare: 10000}},
			}},
			Tunables: testTunables(),
		}
	}
	for _, tt := range []struct{ w, target int64 }{{1, 1}, {7, 3}} {
		// A stock of w is below the halt: 7 * 10 000 < 30 000 * 3.
		got, _ := runTrade(maker(tt.w, tt.w), nil)
		if len(got.Intents) != 1 || int64(got.Intents[0].Target) != tt.target {
			t.Errorf("output target at w %d = %+v, want %d", tt.w, got.Intents, tt.target)
		}
	}
	halt := func(w, stock int64) int {
		v := maker(w, stock)
		ctx := tradeCtx{v: v, par: householdParams(&v.Tunables)}
		return ctx.plannedRecipe()
	}
	if got := halt(1, 2); got != 0 {
		t.Errorf("planned recipe at w 1, stock 2 = %d, want 0 (not halted)", got)
	}
	if got := halt(7, 14); got != -1 {
		t.Errorf("planned recipe at w 7, stock 14 = %d, want -1 (halted)", got)
	}
}

// TestDefaultStrategyTrade_Candidates verifies the candidate rules of 7.3.5 and
// the draw count of 7.5.3: one word for each commodity that is a bid candidate
// (a gap) or an ask candidate (no gap and a surplus, or units expiring beyond
// the reserve), whatever the money, and none for any other, including a tool
// the role does not make.
//
// Requirement: FR-AGT-039, FR-AGT-045, FR-AGT-047
func TestDefaultStrategyTrade_Candidates(t *testing.T) {
	t.Parallel()
	one := func(mutate func(v *AgentView)) *AgentView {
		v := &AgentView{
			Weight: 1, Money: 10000, StartsIn: 1,
			Upkeep:      []UpkeepRow{{Commodity: xFuel, Quantity: 1}},
			Commodities: []CommodityView{cview(xFuel, 0, 100, 0)},
			Tunables:    testTunables(),
		}
		mutate(v)
		return v
	}
	tests := []struct {
		name    string
		view    *AgentView
		words   int
		intents int
		side    Side
		qty     Quantity
	}{
		{"a bid candidate that cannot afford one unit spends its word", one(func(v *AgentView) { v.Money = 0 }), 1, 0, 0, 0},
		{"a bid candidate with one unit of money lowers its limit", one(func(v *AgentView) { v.Money = 1 }), 1, 1, Bid, 1},
		{"a surplus is the whole ask", one(func(v *AgentView) { v.Commodities[0].Stock = 10 }), 1, 1, Ask, 6},
		{"units expiring beyond the reserve are offered", one(func(v *AgentView) {
			// Keep = 1 + 3 = 4 = stock: no gap and no surplus; 3 expire, 1 is eaten.
			v.Commodities[0].Stock, v.Commodities[0].Expiring = 4, 3
		}), 1, 1, Ask, 2},
		{"expiring units within the reserve are no candidate", one(func(v *AgentView) {
			v.Commodities[0].Stock, v.Commodities[0].Expiring = 4, 1
		}), 0, 0, 0, 0},
		{"expiring units are part of the surplus not added to it", one(func(v *AgentView) {
			v.Commodities[0].Stock, v.Commodities[0].Expiring = 10, 5
		}), 1, 1, Ask, 6},
		{"a position at its keep takes no word", one(func(v *AgentView) { v.Commodities[0].Stock = 4 }), 0, 0, 0, 0},
		{"a tool the role does not make is no candidate", one(func(v *AgentView) {
			v.Upkeep = nil
			v.Commodities[0].Tool, v.Commodities[0].Stock = true, 3
		}), 0, 0, 0, 0},
		{"a tool the role makes is sold", one(func(v *AgentView) {
			v.Upkeep = nil
			v.Commodities[0].Tool, v.Commodities[0].Stock = true, 3
			v.Recipes = []RecipeView{{
				Recipe: 0, Duration: 1, Effective: 10000, MaxExecutions: 1,
				Outputs: []RecipeOutput{{Commodity: xFuel, Quantity: 1, CostShare: 10000}},
			}}
		}), 1, 1, Ask, 3},
		{"a tool only discarded is not made", one(func(v *AgentView) {
			v.Upkeep = nil
			v.Commodities[0].Tool, v.Commodities[0].Stock = true, 3
			v.Recipes = []RecipeView{{
				Recipe: 0, Duration: 1, Effective: 10000, MaxExecutions: 1,
				Outputs: []RecipeOutput{{Commodity: xFuel, Quantity: 1, Discard: true}},
			}}
		}), 0, 0, 0, 0},
		{"nothing to do", one(func(v *AgentView) { v.Upkeep = nil }), 0, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, drawn := runTrade(tt.view, nil)
			if drawn != tt.words || len(got.Intents) != tt.intents {
				t.Fatalf("Trade() took %d words and gave %+v, want %d words and %d intents", drawn, got.Intents, tt.words, tt.intents)
			}
			if tt.intents == 1 && (got.Intents[0].Side != tt.side || got.Intents[0].Quantity != tt.qty) {
				t.Errorf("Trade() order = %v %d, want %v %d", got.Intents[0].Side, got.Intents[0].Quantity, tt.side, tt.qty)
			}
		})
	}
}

// TestDefaultStrategyTrade_UrgentBidUsesTheTop verifies that a bid with hunger
// and a tier-1 want takes the top of the belief's range, High, while still
// taking its word, and that a bid without hunger takes the draw (7.5.3).
//
// Requirement: FR-AGT-045
func TestDefaultStrategyTrade_UrgentBidUsesTheTop(t *testing.T) {
	t.Parallel()
	view := func(hunger int64) *AgentView {
		// Centre 100, width 1 000: the range is 90 to 110.
		return &AgentView{
			Weight: 1, Money: 10000, StartsIn: 1, Hunger: hunger,
			Upkeep:      []UpkeepRow{{Commodity: xFuel, Quantity: 1}},
			Commodities: []CommodityView{cview(xFuel, 0, 100, 1000)},
			Tunables:    testTunables(),
		}
	}
	for _, tt := range []struct {
		name   string
		hunger int64
		limit  Price
	}{{"hungry", 1, 110}, {"fed", 0, 90}} {
		got, drawn := runTrade(view(tt.hunger), []uint64{0})
		if drawn != 1 || len(got.Intents) != 1 || got.Intents[0].Limit != tt.limit {
			t.Errorf("%s: Trade() took %d words and gave %+v, want 1 word and a limit of %d", tt.name, drawn, got.Intents, tt.limit)
		}
	}
}

// TestDefaultStrategyTrade_ToolsAndPurposes verifies the job needs of a recipe
// with a tool (Job of the tool is k*, tier 2, Keep and PurposeTool), the
// purpose of a commodity that is both an input and upkeep (PurposeInput), the
// StartsIn rule (Job is 0 unless the job starts next tick, and the input is then
// not PurposeInput), and that UnitCost passes through unchanged.
//
// The recipe uses 1 ore and the tool and makes an item worth 900: weight 2,
// InputJobs 1. The ore is also upkeep (U = 2), so Job(ore) = 1 * 2 * 1 + U = 4
// (N-313: the work rule keeps the upkeep out of the inputs) and Job(tool) = k* = 2.
//
// Requirement: FR-AGT-038, FR-AGT-048, FR-AGT-057, FR-AGT-073
func TestDefaultStrategyTrade_ToolsAndPurposes(t *testing.T) {
	t.Parallel()
	view := func(startsIn int64) *AgentView {
		ore := cview(xOre, 0, 100, 0)
		ore.UnitCost = 77
		return &AgentView{
			Weight: 2, Money: 1_000_000, StartsIn: startsIn,
			Upkeep:      []UpkeepRow{{Commodity: xOre, Quantity: 1}},
			Commodities: []CommodityView{cview(xFuel, 0, 100, 0), ore, cview(xItem, 0, 900, 0)},
			Recipes: []RecipeView{{
				Recipe: 0, Duration: 1, Effective: 10000, MaxExecutions: 1,
				Inputs:  []RecipeLine{{Commodity: xOre, Quantity: 1}},
				Tools:   []RecipeTool{{Commodity: xFuel, WearPerUse: 1, Durability: 100}},
				Outputs: []RecipeOutput{{Commodity: xItem, Quantity: 1, CostShare: 10000}},
			}},
			Tunables: testTunables(),
		}
	}
	// StartsIn 1: ore: U = 2, Job = 2 + 2 = 4, UseTgt = 2 * 3 + 4 = 10, Keep = 12, Target 10;
	// tier 1 is 2, tier 2 is 4, tier 3 is 6. The tool: Job = 2, Keep = 2, bought as tier 2.
	got, _ := runTrade(view(1), nil)
	byCommodity := func(p TradePlan, c CommodityID) (Intent, bool) {
		for _, in := range p.Intents {
			if in.Commodity == c {
				return in, true
			}
		}
		return Intent{}, false
	}
	tool, ok := byCommodity(got, xFuel)
	if !ok || tool.Quantity != 2 || tool.Purpose != PurposeTool || tool.Target != 2 {
		t.Errorf("tool order = %+v, %v, want 2 units, PurposeTool, target 2", tool, ok)
	}
	ore, ok := byCommodity(got, xOre)
	if !ok || ore.Purpose != PurposeInput || ore.Reserve != 2 || ore.Target != 10 || ore.UnitCost != 77 {
		t.Errorf("ore order = %+v, %v, want PurposeInput, reserve 2, target 10, unit cost 77", ore, ok)
	}
	// StartsIn 3: no job next tick, so nothing is bought for it and the ore is upkeep only.
	got, _ = runTrade(view(3), nil)
	ore, ok = byCommodity(got, xOre)
	if !ok || ore.Purpose != PurposeUpkeep || ore.Target != 6 {
		t.Errorf("ore order at StartsIn 3 = %+v, %v, want PurposeUpkeep, target 6", ore, ok)
	}
	if tool, ok = byCommodity(got, xFuel); ok {
		t.Errorf("tool order at StartsIn 3 = %+v, want none (no job next tick)", tool)
	}
}
