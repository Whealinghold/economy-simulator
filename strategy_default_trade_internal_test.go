package economysim

import (
	"math/rand/v2"
	"reflect"
	"testing"
)

// TestDefaultStrategyTrade_FirstBuyerOnAnotherCommodity verifies the cross-
// commodity side of the ruled combination (D98): the order that first meets a
// short budget is lowered to what is left and the later orders get nothing,
// the number of words taken does not depend on the money (D93), and the plan
// keeps one order per commodity.
//
// 16.1 with money 120: fuel (tier 1) buys 1 at 105, leaving 15. Ore (tier 2,
// the first tier of its order to buy) cannot pay 195, so its limit is lowered to
// 15 for 1 unit, leaving 0. Staple is a candidate for the buffer, takes its word
// and buys nothing. The reservations are 105 + 15 = 120, the money.
//
// Requirement: FR-AGT-039, FR-AGT-041, FR-AGT-044, FR-AGT-045
func TestDefaultStrategyTrade_FirstBuyerOnAnotherCommodity(t *testing.T) {
	t.Parallel()
	ask := Intent{Commodity: xItem, Side: Ask, Quantity: 1, Limit: 900, Target: 1, Purpose: PurposeOther}
	tests := []struct {
		name  string
		money int64
		want  []Intent
	}{
		{"tier 2 is the first buyer of its order after tier 1 of another", 120, []Intent{
			{Commodity: xFuel, Side: Bid, Quantity: 1, Limit: 105, Target: 3, Reserve: 1, Purpose: PurposeUpkeep},
			{Commodity: xOre, Side: Bid, Quantity: 1, Limit: 15, Target: 2, Purpose: PurposeInput},
			ask,
		}},
		{"tier 1 is lowered and the others get nothing", 50, []Intent{
			{Commodity: xFuel, Side: Bid, Quantity: 1, Limit: 50, Target: 3, Reserve: 1, Purpose: PurposeUpkeep},
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
			if _, rich := runTrade(exampleView(10_000_000), exampleWords()); drawn != rich {
				t.Errorf("Trade() took %d words with money %d and %d with ample money, want the same", drawn, tt.money, rich)
			}
		})
	}
}

// TestDefaultStrategyTrade_TiersOneAndTwoComeFirst verifies over a grid that the
// buffer (tier 3) is bought after tiers 1 and 2 and never at their expense: with
// the buffer wants set to 0 the same view buys the same limits, no row buys more
// than with the buffer, and a row buys into the buffer only when its tiers 1 and
// 2 were bought in full.
//
// Requirement: FR-AGT-041
func TestDefaultStrategyTrade_TiersOneAndTwoComeFirst(t *testing.T) {
	t.Parallel()
	for _, money := range []int64{0, 1, 50, 105, 120, 250, 1000, 2500, 3105, 5000, 8000, 100000} {
		for _, pos := range []int64{0, 5000, 10000} {
			v := exampleView(money)
			for i := range v.Commodities {
				v.Commodities[i].Window = PriceWindowView{Position: pos, HasWindow: true}
			}
			full, _ := considered(v, exampleWords())
			bare, _ := consideredHook(exampleView(money), exampleWords(), func(rows []tradeRow) {
				for i := range rows {
					rows[i].t3 = 0
				}
			})
			for i := range full.rows {
				f, b := full.rows[i], bare.rows[i]
				if f.kind != bidOrder {
					continue
				}
				if f.limit != b.limit || f.qty < b.qty {
					t.Fatalf("money %d, pos %d, commodity %d: with the buffer %+v, without %+v", money, pos, i, f, b)
				}
				if f.qty > b.qty && b.qty != f.t1+f.t2 {
					t.Errorf("money %d, pos %d, commodity %d: the buffer was bought with tiers 1 and 2 at %d of %d", money, pos, i, b.qty, f.t1+f.t2)
				}
			}
		}
	}
}

// TestDefaultStrategyTrade_BufferCapIsSharedAcrossCommodities verifies that the
// spend cap is one budget for the tier 3 of all the bids (FR-AGT-042).
//
// 16.2 with money 8 000 and the price at the bottom of its window (buy share
// 10 000): after 105 and 390, e = 4 505 and the cap is 1 552. Fuel buys its whole
// buffer of 3 at 105 = 315, leaving 1 237; staple wants 8 but 1 237 / 310 = 3
// units, 930, so the buffer spends 1 245. A cap spent separately for each
// commodity would let staple buy 5 (1 550) and the buffer 1 865.
//
// Requirement: FR-AGT-042
func TestDefaultStrategyTrade_BufferCapIsSharedAcrossCommodities(t *testing.T) {
	t.Parallel()
	v := exampleView(8000)
	for i := range v.Commodities {
		v.Commodities[i].Window = PriceWindowView{Position: 0, HasWindow: true}
	}
	got, _ := considered(v, exampleWords())
	var spent int64
	for _, c := range []struct {
		id   CommodityID
		want int64
	}{{xFuel, 3}, {xStaple, 3}} {
		r := got.rows[c.id]
		q3 := r.qty - r.t1 - r.t2
		if q3 != c.want {
			t.Errorf("commodity %d buffer = %d units, want %d", c.id, q3, c.want)
		}
		spent += q3 * r.limit
	}
	if spent != 1245 || spent > 1552 {
		t.Errorf("buffer spent %d, want 1 245 within the cap of 1 552", spent)
	}
}

// TestDefaultStrategyTrade_ReusedScratchDoesNotLeak verifies that the scratch
// rows the world's filler shares between agents carry nothing from one agent to
// the next: a busy view and then a view with nothing to do, on one scratch,
// plan what a fresh scratch plans, over seeds.
//
// Guard: a row left over from the previous agent would be emitted for a
// commodity that has no order.
func TestDefaultStrategyTrade_ReusedScratchDoesNotLeak(t *testing.T) {
	t.Parallel()
	quiet := func() *AgentView {
		return &AgentView{
			Weight: 1, Money: 8000, StartsIn: 1,
			Commodities: []CommodityView{cview(xFuel, 0, 100, 1000), cview(xStaple, 0, 300, 1000), cview(xOre, 0, 200, 1000), cview(xItem, 0, 900, 0)},
			Tunables:    testTunables(),
		}
	}
	for seed := uint64(0); seed < 100; seed++ {
		r := rand.New(rand.NewPCG(seed, 11))
		words := []uint64{r.Uint64(), r.Uint64(), r.Uint64(), r.Uint64()}
		shared := &viewScratch{}
		for round, mk := range []func() *AgentView{func() *AgentView { return exampleView(8000) }, quiet, func() *AgentView { return exampleView(2500) }, quiet} {
			fresh, _ := runTrade(mk(), words)
			v := mk()
			v.scratch = shared
			got, _ := runTrade(v, words)
			if !reflect.DeepEqual(got.Intents, fresh.Intents) {
				t.Fatalf("seed %d, round %d: on a shared scratch Trade() = %+v, on a fresh one %+v", seed, round, got.Intents, fresh.Intents)
			}
		}
	}
}

// TestDefaultStrategyTrade_UrgentBidNeedsATierOneWant verifies that the urgent
// bid (hunger) takes the top of the range only for a bid with a tier 1 want
// (7.5.3): a hungry agent's bid that is for the inputs of its job alone draws
// its word like any other, and the word is still taken.
//
// Requirement: FR-AGT-045
func TestDefaultStrategyTrade_UrgentBidNeedsATierOneWant(t *testing.T) {
	t.Parallel()
	// Centre 100, width 1 000: the range is 90 to 110. Ore is an input, not upkeep,
	// so the bid is tier 2 and tier 3 only (T1 = 0).
	v := &AgentView{
		Weight: 1, Money: 100_000, StartsIn: 1, Hunger: 1,
		Commodities: []CommodityView{cview(xOre, 0, 100, 1000), cview(xItem, 0, 900, 0)},
		Recipes: []RecipeView{{
			Recipe: 0, Duration: 1, Effective: 10000, MaxExecutions: 1,
			Inputs:  []RecipeLine{{Commodity: xOre, Quantity: 1}},
			Outputs: []RecipeOutput{{Commodity: xItem, Quantity: 1, CostShare: 10000}},
		}},
		Tunables: testTunables(),
	}
	got, drawn := runTrade(v, []uint64{0})
	if len(got.Intents) == 0 || got.Intents[0].Commodity != xOre || got.Intents[0].Side != Bid {
		t.Fatalf("Trade() = %+v, want a bid for ore first", got.Intents)
	}
	if in := got.Intents[0]; in.Limit != 90 || in.Purpose != PurposeInput {
		t.Errorf("hungry bid without a tier 1 want = %+v, want the drawn limit 90 and PurposeInput", in)
	}
	if drawn != 1 {
		t.Errorf("Trade() took %d words, want 1 (ore; the item is no candidate)", drawn)
	}
}

// TestDefaultStrategyTrade_InputAndToolCommodity verifies the purpose and the
// tiers of a commodity that is both an input and a tool of the planned recipe.
//
// Weight 1, InputJobs 1: the recipe uses 2 ore as input and 1 ore as a tool, so
// Job = 1 * 1 * 2 + 1 = 3, there is no upkeep (Reserve 0) and the stock is 0:
// Keep = Target = 3, the gap is 3, all of it tier 2 (T1 = 0, T3 = 0). The
// purpose is PurposeInput, since an input with a job to buy for comes first.
//
// Requirement: FR-AGT-038, FR-AGT-048
func TestDefaultStrategyTrade_InputAndToolCommodity(t *testing.T) {
	t.Parallel()
	v := &AgentView{
		Weight: 1, Money: 100_000, StartsIn: 1,
		Commodities: []CommodityView{cview(xOre, 0, 100, 0), cview(xItem, 0, 900, 0)},
		Recipes: []RecipeView{{
			Recipe: 0, Duration: 1, Effective: 10000, MaxExecutions: 1,
			Inputs:  []RecipeLine{{Commodity: xOre, Quantity: 2}},
			Tools:   []RecipeTool{{Commodity: xOre, WearPerUse: 1, Durability: 100}},
			Outputs: []RecipeOutput{{Commodity: xItem, Quantity: 1, CostShare: 10000}},
		}},
		Tunables: testTunables(),
	}
	got, _ := runTrade(v, nil)
	want := Intent{Commodity: xOre, Side: Bid, Quantity: 3, Limit: 100, Target: 3, Purpose: PurposeInput}
	if len(got.Intents) == 0 || got.Intents[0] != want {
		t.Errorf("Trade() = %+v, want first %+v", got.Intents, want)
	}
	ctx, _ := considered(v, nil)
	if r := ctx.rows[0]; r.t1 != 0 || r.t2 != 3 || r.t3 != 0 {
		t.Errorf("tiers = %d, %d, %d, want 0, 3, 0", r.t1, r.t2, r.t3)
	}
}

// TestDefaultStrategyTrade_TierOneIgnoresPrices verifies that the quantity
// wanted for upkeep does not depend on any price or belief (FR-AGT-069): the same
// stock and weight give the same tier 1 at every belief centre and every price
// window position, and the price shows only in the limit.
//
// Requirement: FR-AGT-069
func TestDefaultStrategyTrade_TierOneIgnoresPrices(t *testing.T) {
	t.Parallel()
	for _, w := range []int64{1, 7} {
		for _, stock := range []int64{0, 1, 5} {
			var first tradeRow
			for i, centre := range []int64{1, 100, 5_000, 900_000} {
				for _, pos := range []int64{0, 5000, 10000} {
					v := exampleView(1_000_000_000)
					v.Weight = w
					v.Commodities[0].Stock = stock
					v.Commodities[0].Belief.Centre = Price(centre)
					v.Commodities[0].Window.Position = pos
					ctx, _ := considered(v, exampleWords())
					r := ctx.rows[xFuel]
					if i == 0 && pos == 0 {
						first = r
					}
					if r.t1 != first.t1 || r.reserve != first.reserve {
						t.Errorf("w %d, stock %d, centre %d, pos %d: tier 1 = %d, reserve %d, want %d, %d", w, stock, centre, pos, r.t1, r.reserve, first.t1, first.reserve)
					}
				}
			}
		}
	}
}

// TestDefaultStrategyTrade_OutputTargetCeilsOnTheTotal verifies that the output
// target of a cohort is the ceiling of w times the exact quantity, not w times
// the ceiling (the packs' duration-3, output-1 recipes): ceil(w * q * E *
// OutputCoverTicks / (10 000 * Duration)).
//
// Requirement: FR-AGT-052
func TestDefaultStrategyTrade_OutputTargetCeilsOnTheTotal(t *testing.T) {
	t.Parallel()
	for _, dur := range []int64{1, 2, 3, 7} {
		for _, w := range []int64{1, 2, 3, 7, 40} {
			v := &AgentView{
				Weight: w, Money: 1_000_000, StartsIn: 1,
				Commodities: []CommodityView{cview(xItem, 1, 100, 0)},
				Recipes: []RecipeView{{
					Recipe: 0, Duration: dur, Effective: 10000, MaxExecutions: 1,
					Outputs: []RecipeOutput{{Commodity: xItem, Quantity: 1, CostShare: 10000}},
				}},
				Tunables: testTunables(),
			}
			ctx, _ := considered(v, nil)
			if want := (w + dur - 1) / dur; ctx.rows[0].target != want {
				t.Errorf("duration %d, w %d: output target = %d, want ceil(%d / %d) = %d", dur, w, ctx.rows[0].target, w, dur, want)
			}
		}
	}
}

// TestDefaultStrategyWork_ScalesWithTheWeight verifies the closed form of the
// request on the totals: a cohort of weight w in the band of the ratio asks for
// ceil(w * share / 10 000) executions, whatever w is (7.1.2, FR-AGT-052).
//
// Requirement: FR-AGT-034, FR-AGT-052
func TestDefaultStrategyWork_ScalesWithTheWeight(t *testing.T) {
	t.Parallel()
	for seed := uint64(0); seed < 100; seed++ {
		r := rand.New(rand.NewPCG(seed, 13))
		w := []int64{1, 2, 3, 7, 40}[seed%5]
		ratio := 15000 + r.Int64N(750)
		share := (15750 - ratio) * 10000 / 750
		v := workView()
		v.Weight, v.Money, v.Ratio = w, 1_000_000_000, ratio
		v.Commodities[cStaple].Stock = 1000 * w
		v.Recipes[0].MaxExecutions = w
		want := JobRequest{Recipe: 0, Executions: (w*share + 9999) / 10000}
		if share == 0 {
			want = JobRequest{NoJob: true}
		}
		if got := (DefaultStrategy{}).Work(v, nil); got != want {
			t.Fatalf("seed %d, w %d, ratio %d: Work() = %+v, want %+v", seed, w, ratio, got, want)
		}
	}
}

// TestDefaultStrategyTrade_ReservationNeverExceedsTheMoneyCharged verifies, over
// a grid of money and weights that includes a first buyer lowered to what is
// left and later tiers dropped, the rule of 7.5.3 (D98): every bid ends at a
// limit that is at least 1 and at most the limit drawn (a limit is lowered,
// never raised), and the reservation limit times quantity of each bid, and the
// sum of them over the bids, is at most the money the allocation charged, which
// is at most the money held. The charges are rebuilt from the rows: a bid is
// charged limit times quantity at its one final limit, tier by tier, so no tier
// may have been charged at a limit above the final one.
//
// Requirement: FR-AGT-041, FR-AGT-044
func TestDefaultStrategyTrade_ReservationNeverExceedsTheMoneyCharged(t *testing.T) {
	t.Parallel()
	for _, w := range []int64{1, 3} {
		for _, money := range []int64{0, 1, 15, 50, 105, 120, 250, 500, 1000, 2500, 8000, 100000} {
			v := exampleView(money)
			v.Weight = w
			var drawn []int64
			tc, ok := consideredHook(v, exampleWords(), func(rows []tradeRow) {
				for _, r := range rows {
					drawn = append(drawn, r.limit)
				}
			})
			if !ok {
				t.Fatalf("w %d, money %d: the example view is not tradable", w, money)
			}
			var total int64
			for k, r := range tc.rows {
				if r.kind != bidOrder || r.qty < 1 {
					continue
				}
				if r.limit < 1 || r.limit > drawn[k] {
					t.Errorf("w %d, money %d, commodity %d: final limit %d, drawn %d, want 1 <= final <= drawn", w, money, r.commodity, r.limit, drawn[k])
				}
				res := r.limit * r.qty
				if res > money {
					t.Errorf("w %d, money %d, commodity %d: reservation %d exceeds the money", w, money, r.commodity, res)
				}
				total += res
			}
			if total > money {
				t.Errorf("w %d, money %d: reservations total %d, want at most the money charged", w, money, total)
			}
		}
	}
}

// TestDefaultStrategyTrade_NeverDiscards verifies, over a grid of weights,
// money, stocks, expiring units and hunger on the example household, that the
// default plan carries no discard: a surplus is asked, never disposed of, with
// the storage module off (FR-AGT-051). The plan has no shipment field at this
// stage (FR-AGT-079 comes with the transport module), so there is nothing else
// for the default to append.
//
// Requirement: FR-AGT-051
func TestDefaultStrategyTrade_NeverDiscards(t *testing.T) {
	t.Parallel()
	plans := 0
	for _, w := range []int64{1, 4, 40} {
		for _, money := range []int64{0, 120, 2500, 8000, 10_000_000} {
			for _, stock := range []int64{0, 1, 5, 40, 500} {
				for _, expiring := range []int64{0, 1, 500} {
					v := exampleView(money)
					v.Weight, v.Hunger = w, stock%2
					for k := range v.Commodities {
						v.Commodities[k].Stock = stock * w
						v.Commodities[k].Expiring = min(expiring, stock*w)
					}
					p, _ := runTrade(v, exampleWords())
					plans++
					if len(p.Discards) != 0 {
						t.Errorf("w %d, money %d, stock %d, expiring %d: discards %+v, want none", w, money, stock, expiring, p.Discards)
					}
				}
			}
		}
	}
	if plans == 0 {
		t.Fatal("no plan was made")
	}
}

// TestDefaultStrategyTrade_JobKeepsTheNextTicksUpkeep verifies that tier 2 of an
// input that is also an upkeep commodity asks for the next tick's upkeep as well
// as the input: the work rule leaves the upkeep out of the inputs (FR-AGT-037,
// foodCap), so a job needs U + q in stock when it starts.
//
// One household eats 1 fuel a tick, holds 1 and plans a job that needs 1 fuel.
// It eats its 1 at the end of this tick (reserve 1, so tier 1 is 0), and the
// next tick it must hold the upkeep 1 plus the input 1: tier 2 is 1 + 1 = 2. A
// fuel that is not upkeep, with none held, stays the input alone: tier 2 is 1.
//
// Requirement: FR-AGT-038, FR-AGT-040
//
// Fix: N-313. Tier 2 was only the input, so a cook holding one tick of fuel
// bought 1, ate 1, started the next tick with 1 and foodCap was 0: the whole
// role stalled on every tick.
func TestDefaultStrategyTrade_JobKeepsTheNextTicksUpkeep(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		upkeep []UpkeepRow
		stock  int64
		want   int64
	}{
		{"an input that is also upkeep adds the upkeep", []UpkeepRow{{Commodity: xFuel, Quantity: 1}}, 1, 2},
		{"an input that is not upkeep is the input alone", nil, 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := &AgentView{
				Weight: 1, Money: 100_000, StartsIn: 1, Upkeep: tt.upkeep,
				Commodities: []CommodityView{cview(xFuel, tt.stock, 100, 0), cview(xItem, 0, 900, 0)},
				Recipes: []RecipeView{{
					Recipe: 0, Duration: 1, Effective: 10000, MaxExecutions: 1,
					Inputs:  []RecipeLine{{Commodity: xFuel, Quantity: 1}},
					Outputs: []RecipeOutput{{Commodity: xItem, Quantity: 1, CostShare: 10000}},
				}},
				Tunables: testTunables(),
			}
			ctx, _ := considered(v, nil)
			if r := ctx.rows[0]; r.t2 != tt.want {
				t.Errorf("considered(...) tier 2 = %d, want %d", r.t2, tt.want)
			}
		})
	}
}
