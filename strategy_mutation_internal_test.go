package economysim

import (
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/household"
)

// The tests in this file pin edges of strategy.go, strategy_work.go,
// strategy_view.go and strategy_default.go that a mutation run (gomu, inside
// Docker) showed the other tests did not reach.

// TestDirected_TradeIgnoresADirectiveThatIsNotPending verifies that a view whose
// HasDirective is false is planned as the fallback plans it, even when the
// Directive group holds data.
//
// Requirement: FR-AGT-054
//
// Guard: the flag, not the content, says a directive applies; a stale group
// must not hold a commodity or add an order.
func TestDirected_TradeIgnoresADirectiveThatIsNotPending(t *testing.T) {
	t.Parallel()
	fb := &fakeStrategy{intents: []Intent{bid(1, 1)}}
	v := &AgentView{Directive: Directive{
		Intents:  []Intent{bid(2, 20)},
		Hold:     []CommodityID{1},
		Discards: []Discard{{Commodity: 3, Quantity: 1}},
	}}
	p := &TradePlan{}
	Directed{Fallback: fb}.Trade(v, rand.New(rand.NewPCG(1, 2)), p)
	if want := []Intent{bid(1, 1)}; !reflect.DeepEqual(p.Intents, want) {
		t.Errorf("Directed.Trade without HasDirective intents = %+v, want %+v", p.Intents, want)
	}
	if len(p.Discards) != 0 {
		t.Errorf("Directed.Trade without HasDirective discards = %+v, want none", p.Discards)
	}
}

// TestDirected_TradeReplacesAndInsertsAtAnyPosition verifies the merge when the
// matching directive intent is not the first (index 1), when the fallback's
// replaced intent is not the first of its plan, and when new intents go to the
// front and the middle of the plan.
//
// The fallback plans 2 and 4. The directive is [3 (new), 4 (replaces), 1
// (new)]. Replacing 4 reads index 1 of the directive; 3 goes between 2 and 4
// and 1 to the front, so the plan is [1, 2, 3, 4'].
//
// Requirement: FR-AGT-054
func TestDirected_TradeReplacesAndInsertsAtAnyPosition(t *testing.T) {
	t.Parallel()
	fb := &fakeStrategy{intents: []Intent{bid(2, 1), bid(4, 2)}}
	v := &AgentView{HasDirective: true, Directive: Directive{Intents: []Intent{bid(3, 30), bid(4, 40), bid(1, 10)}}}
	p := &TradePlan{}
	Directed{Fallback: fb}.Trade(v, rand.New(rand.NewPCG(1, 2)), p)
	want := []Intent{bid(1, 10), bid(2, 1), bid(3, 30), bid(4, 40)}
	if !reflect.DeepEqual(p.Intents, want) {
		t.Errorf("Directed.Trade intents = %+v, want %+v", p.Intents, want)
	}
}

// TestInsertIntent verifies the insertion into a sorted tail: before all, in
// the middle, after all and after an equal twin, and that intents before from
// are never moved.
//
// Requirement: FR-AGT-054
func TestInsertIntent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		list []Intent
		from int
		in   Intent
		want []Intent
	}{
		{"before all", []Intent{bid(2, 1), bid(3, 1)}, 0, bid(1, 9), []Intent{bid(1, 9), bid(2, 1), bid(3, 1)}},
		{"in the middle", []Intent{bid(1, 1), bid(3, 1)}, 0, bid(2, 9), []Intent{bid(1, 1), bid(2, 9), bid(3, 1)}},
		{"after all", []Intent{bid(1, 1), bid(2, 1)}, 0, bid(3, 9), []Intent{bid(1, 1), bid(2, 1), bid(3, 9)}},
		{"after an equal twin", []Intent{bid(2, 1), bid(3, 1)}, 0, bid(2, 9), []Intent{bid(2, 1), bid(2, 9), bid(3, 1)}},
		{"never before from", []Intent{bid(5, 1), bid(6, 1)}, 1, bid(1, 9), []Intent{bid(5, 1), bid(1, 9), bid(6, 1)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := insertIntent(append([]Intent(nil), tt.list...), tt.from, tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("insertIntent(%+v, %d, %+v) = %+v, want %+v", tt.list, tt.from, tt.in, got, tt.want)
			}
		})
	}
}

// TestAgentView_LookupIsExactNotOrdered verifies that a commodity absent from
// the view is not mistaken for a listed one with a higher ID.
//
// Requirement: FR-AGT-071
func TestAgentView_LookupIsExactNotOrdered(t *testing.T) {
	t.Parallel()
	v := &AgentView{Commodities: []CommodityView{cview(cOre, 4, 50, 0)}}
	if got := v.commodityAt(cStaple); got != -1 {
		t.Errorf("commodityAt(%d) with only %d listed = %d, want -1", cStaple, cOre, got)
	}
	if v.centreOf(cStaple) != 0 || v.stockOf(cStaple) != 0 {
		t.Errorf("centreOf, stockOf of an unlisted commodity = %d, %d, want 0, 0", v.centreOf(cStaple), v.stockOf(cStaple))
	}
}

// TestUpkeepCost_OverflowIsNotOK verifies that an upkeep whose total does not
// fit reports not ok and a cost of 0, and that an ample reserve is not read as
// the largest value.
//
// Requirement: FR-AGT-072
func TestUpkeepCost_OverflowIsNotOK(t *testing.T) {
	t.Parallel()
	v := workView()
	v.Weight = math.MaxInt64
	v.Upkeep = []UpkeepRow{{Commodity: cStaple, Quantity: 2}}
	if got, ok := v.upkeepCost(); ok || got != 0 {
		t.Errorf("upkeepCost() with an overflowing total = %d, %v, want 0, false", got, ok)
	}
	if got := v.reserveMoney(householdParams(&v.Tunables)); got != math.MaxInt64 {
		t.Errorf("reserveMoney() with an overflowing cost = %d, want MaxInt64", got)
	}
	if got := workView().reserveMoney(householdParams(&v.Tunables)); got != 2700 {
		t.Errorf("reserveMoney() = %d, want 2 700 (3 ticks * 3 staple * 300)", got)
	}
}

// TestRecipeNet_DiscardOutputsAndToolOverflow verifies that a discard output
// adds no value and that a tool whose wear does not fit returns 0 and not ok.
//
// A kept output of 2 at 100 is worth 200; the discard output of 5 is not
// counted, so the net stays 200.
//
// Requirement: FR-AGT-035
func TestRecipeNet_DiscardOutputsAndToolOverflow(t *testing.T) {
	t.Parallel()
	v := workView()
	r := RecipeView{Duration: 1, Effective: 10000, Outputs: []RecipeOutput{
		{Commodity: cItem, Quantity: 2, CostShare: 10000},
		{Commodity: cItem, Quantity: 5, Discard: true},
	}}
	if got, ok := v.recipeNet(&r); !ok || got != 200 {
		t.Errorf("recipeNet(with a discard output) = %d, %v, want 200, true", got, ok)
	}
	r.Tools = []RecipeTool{{Commodity: cTool, WearPerUse: math.MaxInt64, Durability: 1}}
	if got, ok := v.recipeNet(&r); ok || got != 0 {
		t.Errorf("recipeNet(overflowing tool) = %d, %v, want 0, false", got, ok)
	}
}

// TestRecipeHalted_ByproductsDoNotStopTheScan verifies that an output with no
// cost share is passed over without ending the scan, that a cost share of 1
// counts, and that a negative one does not.
//
// Stock 100 against an output target of 1 * 1 * 10 000 * 1 / 10 000 = 1 is far
// over the halt ratio of 30 000 / 10 000.
//
// Requirement: FR-AGT-036
func TestRecipeHalted_ByproductsDoNotStopTheScan(t *testing.T) {
	t.Parallel()
	v := workView()
	v.Commodities[1].Stock = 100 // item
	p := householdParams(&v.Tunables)
	tests := []struct {
		name    string
		outputs []RecipeOutput
		want    bool
	}{
		{"a byproduct before the main output", []RecipeOutput{{Commodity: cOre, Quantity: 1, CostShare: 0}, {Commodity: cItem, Quantity: 1, CostShare: 10000}}, true},
		{"a cost share of 1 counts", []RecipeOutput{{Commodity: cItem, Quantity: 1, CostShare: 1}}, true},
		{"a negative cost share does not", []RecipeOutput{{Commodity: cItem, Quantity: 1, CostShare: -1}}, false},
		{"a byproduct alone never halts", []RecipeOutput{{Commodity: cItem, Quantity: 1, CostShare: 0}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := RecipeView{Duration: 1, Effective: 10000, Outputs: tt.outputs}
			if got := v.recipeHalted(&r, p); got != tt.want {
				t.Errorf("recipeHalted(%+v) = %v, want %v", tt.outputs, got, tt.want)
			}
		})
	}
}

// TestFoodCap_Edges verifies that only an input that is itself an upkeep row
// cuts the executions, whichever side of it the other upkeep rows sort on, and
// that an upkeep total that does not fit allows none.
//
// Requirement: FR-AGT-037
func TestFoodCap_Edges(t *testing.T) {
	t.Parallel()
	t.Run("an input that is not upkeep is not cut", func(t *testing.T) {
		t.Parallel()
		v := workView()
		v.Upkeep = []UpkeepRow{{Commodity: cOre, Quantity: 3}} // a row sorting above the input
		v.Commodities[2].Stock = 0
		r := RecipeView{MaxExecutions: 4, Inputs: []RecipeLine{{Commodity: cStaple, Quantity: 1}}}
		if got := v.foodCap(&r); got != 4 {
			t.Errorf("foodCap() = %d, want 4", got)
		}
	})
	t.Run("an upkeep row of a higher ID than the input is not the input", func(t *testing.T) {
		t.Parallel()
		v := workView()
		v.Upkeep = []UpkeepRow{{Commodity: cStaple + 100, Quantity: 1000}} // a wrong match would cut by this total and cap at 0
		r := RecipeView{MaxExecutions: 4, Inputs: []RecipeLine{{Commodity: cStaple, Quantity: 1}}}
		if got := v.foodCap(&r); got != 4 {
			t.Errorf("foodCap() = %d, want 4", got)
		}
	})
	t.Run("an upkeep input is cut to what it leaves", func(t *testing.T) {
		t.Parallel()
		v := workView() // stock 10 staple, upkeep 3
		r := RecipeView{MaxExecutions: 9, Inputs: []RecipeLine{{Commodity: cStaple, Quantity: 2}}}
		if got := v.foodCap(&r); got != 3 { // (10 - 3) / 2
			t.Errorf("foodCap() = %d, want 3", got)
		}
	})
	t.Run("a total past int64 allows none", func(t *testing.T) {
		t.Parallel()
		v := workView()
		v.Weight = math.MaxInt64
		r := RecipeView{MaxExecutions: 4, Inputs: []RecipeLine{{Commodity: cStaple, Quantity: 1}}}
		if got := v.foodCap(&r); got != 0 {
			t.Errorf("foodCap() = %d, want 0", got)
		}
	})
}

// TestOffer_IneligibleIsZero verifies that a recipe at a negative net returns
// zero values and not ok.
//
// Requirement: FR-AGT-035
func TestOffer_IneligibleIsZero(t *testing.T) {
	t.Parallel()
	v := workView()
	r := RecipeView{Duration: 1, Effective: 10000, MaxExecutions: 1,
		Inputs:  []RecipeLine{{Commodity: cOre, Quantity: 10}},                     // cost 500
		Outputs: []RecipeOutput{{Commodity: cItem, Quantity: 1, CostShare: 10000}}} // value 100
	if net, n, ok := v.offer(&r, 1); ok || net != 0 || n != 0 {
		t.Errorf("offer(net -400) = %d, %d, %v, want 0, 0, false", net, n, ok)
	}
}

// TestWorkReport_PicksAndDeclines verifies, over recipe lists, that the first
// recipe that cannot run gives the decline reason even after recipes that could
// (a stale reason on a runnable one is ignored), that the best of three is kept
// whatever its position, and that an unblocked report names no recipe.
//
// Requirement: FR-AGT-035, FR-AGT-072
func TestWorkReport_PicksAndDeclines(t *testing.T) {
	t.Parallel()
	losing := func(id RecipeID, maxExec int64, reason Reason) RecipeView {
		// Inputs worth 500 for an output worth 100: a net of -400, ineligible.
		return RecipeView{Recipe: id, Duration: 1, Effective: 10000, MaxExecutions: maxExec, Reason: reason,
			Inputs:  []RecipeLine{{Commodity: cOre, Quantity: 10}},
			Outputs: []RecipeOutput{{Commodity: cItem, Quantity: 1, CostShare: 10000}}}
	}
	source := func(id RecipeID, q int64) RecipeView {
		return RecipeView{Recipe: id, Duration: 1, Effective: 10000, MaxExecutions: 1,
			Outputs: []RecipeOutput{{Commodity: cItem, Quantity: q, CostShare: 10000}}}
	}
	t.Run("the first recipe that cannot run gives the reason", func(t *testing.T) {
		t.Parallel()
		v := workView()
		v.Recipes = []RecipeView{losing(0, 1, "stale_one"), losing(1, 2, "stale_two"), losing(2, 0, "no_input"), losing(3, 0, "later")}
		var b workReportBuf
		workReport(v, &b)
		if b.chosen != -1 || b.reason != "no_input" || b.blocked != 2 {
			t.Errorf("workReport() chosen %d reason %q blocked %d, want -1, no_input, 2", b.chosen, b.reason, b.blocked)
		}
	})
	t.Run("the best of three is kept", func(t *testing.T) {
		t.Parallel()
		v := workView()
		v.Recipes = []RecipeView{source(0, 1), source(1, 5), source(2, 2)} // nets 100, 500, 200
		var b workReportBuf
		workReport(v, &b)
		if b.chosen != 1 {
			t.Errorf("workReport() chosen %d, want 1 (the net of 500)", b.chosen)
		}
	})
	t.Run("no recipes names none", func(t *testing.T) {
		t.Parallel()
		v := workView()
		v.Recipes = nil
		var b workReportBuf
		workReport(v, &b)
		if b.chosen != -1 || b.blocked != -1 || b.reason != ReasonNoRecipe {
			t.Errorf("workReport() chosen %d blocked %d reason %q, want -1, -1, no_recipe", b.chosen, b.blocked, b.reason)
		}
	})
}

// TestFillCore_MarketsAreSeparatePerRegion verifies that two regions' reports
// and price windows for one commodity reach their own agents' views and not
// each other's.
//
// Guard: the markets are one flat table indexed by region and commodity; a
// wrong index would give a household another region's price without any error.
func TestFillCore_MarketsAreSeparatePerRegion(t *testing.T) {
	t.Parallel()
	h := NewViewHarness(t, viewConfig())
	north := h.Spawn(t, "farmer", "north", 1)
	south := h.Spawn(t, "farmer", "south", 1)
	h.SetReport(t, "north", "grain", MarketReport{Price: 50, Age: 1})
	h.SetReport(t, "south", "grain", MarketReport{Price: 70, Age: 2})
	h.PushWindow(t, "north", "grain", 50)
	h.PushWindow(t, "south", "grain", 70)
	grain := h.commodity(t, "grain")
	for _, tt := range []struct {
		name  string
		agent int
		price int64
	}{{"north", north, 50}, {"south", south, 70}} {
		v, err := h.Fill(tt.agent)
		if err != nil {
			t.Fatalf("Fill(%s) = %v", tt.name, err)
		}
		i := v.commodityAt(grain)
		if i < 0 {
			t.Fatalf("grain not in the view of %s", tt.name)
		}
		if got := v.Commodities[i].Market.Price; got != tt.price {
			t.Errorf("%s grain market price = %d, want %d", tt.name, got, tt.price)
		}
	}
}

// TestFillCommodities_BeliefsLongerThanTheTradableSetIsAnError verifies that a
// belief list with more entries than the tradable set is refused, as a shorter
// one is.
//
// Requirement: FR-AGT-071
func TestFillCommodities_BeliefsLongerThanTheTradableSetIsAnError(t *testing.T) {
	t.Parallel()
	h := NewViewHarness(t, viewConfig())
	i := h.Spawn(t, "farmer", "north", 1)
	b := h.agents[i].beliefs
	h.agents[i].beliefs = append(b[:len(b):len(b)], b[0])
	if _, err := h.Fill(i); err == nil {
		t.Errorf("Fill with one belief too many = nil, want an error")
	}
}

// TestDefaultStrategyRevise_SideMatters verifies that the side of the outcome
// reaches the belief rule: an unfilled bid and an unfilled ask at the same
// limit, price and stock give different centres, and each equals what
// beliefs.Update returns for its own side.
//
// With limit 60 at a market price of 60, centre 100, no stock and an unfilled
// quantity, the bid moves the centre to 102 and the ask to 101.
//
// Requirement: FR-AGT-050
func TestDefaultStrategyRevise_SideMatters(t *testing.T) {
	t.Parallel()
	bt := beliefTunablesOf(beliefParams(DefaultWorld()))
	par := beliefParamsOf(&bt)
	view := func(s Side) *OutcomeView {
		return &OutcomeView{Side: s, Limit: 60, Quantity: 10, Filled: 0,
			Belief: Belief{Centre: 100, Width: 1000}, Bounds: Bounds{Floor: 50, Ceiling: 5000},
			Market: MarketReport{Price: 60, Age: 1, BestBid: 55, BestAsk: 65, HasBestBid: true, HasBestAsk: true},
			Stock:  0, Target: 20, Draw: 0, Params: bt}
	}
	got := map[Side]Belief{}
	for _, s := range []Side{Bid, Ask} {
		want, _ := beliefs.Update(beliefs.Belief{Centre: 100, Width: 1000}, beliefs.Bounds{Floor: 50, Ceiling: 5000},
			beliefs.Outcome{Bid: s == Bid, Limit: 60, Quantity: 10, Filled: 0},
			beliefs.Quote{Price: 60, Age: 1, BestBid: 55, BestAsk: 65, HasBid: true, HasAsk: true}, 0, 20, par, 0)
		got[s] = DefaultStrategy{}.Revise(view(s), nil)
		if w := (Belief{Centre: Price(want.Centre), Width: want.Width}); got[s] != w {
			t.Errorf("Revise(%v) = %+v, want %+v", s, got[s], w)
		}
	}
	if got[Bid] == got[Ask] {
		t.Errorf("Revise gave %+v for a bid and an ask, want different beliefs", got[Bid])
	}
}

// ctxFor is a trade context for v with the planned recipe rstar.
func ctxFor(v *AgentView, rstar int) *tradeCtx {
	return &tradeCtx{v: v, par: householdParams(&v.Tunables), rstar: rstar}
}

// TestPlannedRecipe_Choice verifies the planned recipe over lists that put a
// skipped recipe before a good one, a tiny yield, an ineligible recipe with
// inputs, a source at a negative net, a tie and a better recipe in the middle.
//
// Nets at the workView centres (item 100, ore 50): a source making 1 item is
// worth 100 and one making 5 is worth 500; a recipe that eats 10 ore (500) to
// make 1 item (100) is worth -400.
//
// Requirement: FR-AGT-035
func TestPlannedRecipe_Choice(t *testing.T) {
	t.Parallel()
	src := func(q int64, eff int64) RecipeView {
		return RecipeView{Duration: 1, Effective: eff, MaxExecutions: 1,
			Outputs: []RecipeOutput{{Commodity: cItem, Quantity: q, CostShare: 10000}}}
	}
	eating := func(oreQty int64, q int64) RecipeView {
		r := src(q, 10000)
		r.Inputs = []RecipeLine{{Commodity: cOre, Quantity: oreQty}}
		return r
	}
	tests := []struct {
		name    string
		recipes []RecipeView
		want    int
	}{
		{"none", nil, -1},
		{"a zero yield before a good recipe", []RecipeView{src(5, 0), src(1, 10000)}, 1},
		{"a yield of 1 is still a yield", []RecipeView{src(5, 1)}, 0},
		{"an ineligible recipe before a good one", []RecipeView{eating(10, 1), src(1, 10000)}, 1},
		{"an ineligible recipe alone", []RecipeView{eating(10, 1)}, -1},
		{"a recipe with inputs at a positive net", []RecipeView{eating(1, 5)}, 0}, // 500 - 50
		{"a source at a negative net is eligible", []RecipeView{src(0, 10000)}, 0},
		{"a tie keeps the first", []RecipeView{src(2, 10000), src(2, 10000)}, 0},
		{"the best in the middle is kept", []RecipeView{src(1, 10000), src(5, 10000), src(2, 10000)}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := workView()
			v.Recipes = tt.recipes
			if got := ctxFor(v, -1).plannedRecipe(); got != tt.want {
				t.Errorf("plannedRecipe() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestJobAndOutputTerms verifies the job and output terms of the planned
// recipe: the job counts inputs and tools only when the job starts next tick,
// an output counts only if it is not a discard and is the commodity asked for,
// the second recipe of the list is read when it is the planned one, and an
// overflow in either term is not ok.
//
// Requirement: FR-AGT-038
func TestJobAndOutputTerms(t *testing.T) {
	t.Parallel()
	v := workView()
	v.Weight = 3
	v.Tunables.InputJobs = 2
	v.Recipes = []RecipeView{
		{Duration: 1, Effective: 10000, MaxExecutions: 1},
		{Duration: 1, Effective: 10000, MaxExecutions: 1,
			Inputs:  []RecipeLine{{Commodity: cOre, Quantity: 5}},
			Tools:   []RecipeTool{{Commodity: cTool, WearPerUse: 1, Durability: 10}},
			Outputs: []RecipeOutput{{Commodity: cItem, Quantity: 4, CostShare: 10000}, {Commodity: cOre, Quantity: 9, Discard: true}}},
	}
	t.Run("job of an input is jobs times weight times quantity", func(t *testing.T) {
		t.Parallel()
		got, ok := ctxFor(v, 1).jobTerm(cOre)
		if !ok || got != 2*3*5 {
			t.Errorf("jobTerm(ore) = %d, %v, want 30, true", got, ok)
		}
	})
	t.Run("job of a tool is the weight", func(t *testing.T) {
		t.Parallel()
		got, ok := ctxFor(v, 1).jobTerm(cTool)
		if !ok || got != 3 {
			t.Errorf("jobTerm(tool) = %d, %v, want 3, true", got, ok)
		}
	})
	t.Run("a commodity that is neither has no job", func(t *testing.T) {
		t.Parallel()
		// item (1) is below the tool (3) and ore (2): an ordered comparison would count it.
		for _, c := range []CommodityID{cStaple, cItem} {
			if got, ok := ctxFor(v, 1).jobTerm(c); !ok || got != 0 {
				t.Errorf("jobTerm(%d) = %d, %v, want 0, true", c, got, ok)
			}
		}
		w := *v
		w.Recipes = slices.Clone(v.Recipes)
		w.Recipes[1].Inputs = nil
		if got, _ := ctxFor(&w, 1).jobTerm(cOre); got != 0 {
			t.Errorf("jobTerm(ore) with only a tool of a higher ID = %d, want 0", got)
		}
	})
	t.Run("a job that does not start next tick has none", func(t *testing.T) {
		t.Parallel()
		for _, starts := range []int64{0, 2} {
			w := *v
			w.StartsIn = starts
			if got, ok := ctxFor(&w, 1).jobTerm(cOre); !ok || got != 0 {
				t.Errorf("jobTerm(ore) with StartsIn %d = %d, %v, want 0, true", starts, got, ok)
			}
		}
	})
	t.Run("the output term skips discards and other commodities", func(t *testing.T) {
		t.Parallel()
		tc := ctxFor(v, 1)
		want, _ := household.OutputTarget(3, 4, 10000, tc.par.OutputCoverTicks, 1)
		if got, ok := tc.outputTerm(cItem); !ok || got != want || want == 0 {
			t.Errorf("outputTerm(item) = %d, %v, want %d (non-zero), true", got, ok, want)
		}
		if got, ok := tc.outputTerm(cOre); !ok || got != 0 {
			t.Errorf("outputTerm(ore, a discard) = %d, %v, want 0, true", got, ok)
		}
		if got, ok := tc.outputTerm(cStaple); !ok || got != 0 {
			t.Errorf("outputTerm(staple) = %d, %v, want 0, true", got, ok)
		}
	})
	t.Run("no planned recipe gives zero terms", func(t *testing.T) {
		t.Parallel()
		if job, out, ok := ctxFor(v, -1).jobAndOutput(cItem); job != 0 || out != 0 || !ok {
			t.Errorf("jobAndOutput with no recipe = %d, %d, %v, want 0, 0, true", job, out, ok)
		}
		tc := ctxFor(v, 1)
		job, out, ok := tc.jobAndOutput(cItem)
		if want, _ := tc.outputTerm(cItem); job != 0 || out != want || !ok {
			t.Errorf("jobAndOutput(item) on the second recipe = %d, %d, %v, want 0, %d, true", job, out, ok, want)
		}
	})
	t.Run("an overflow in one term is not ok", func(t *testing.T) {
		t.Parallel()
		w := *v
		w.Tunables.InputJobs = math.MaxInt64
		if _, ok := ctxFor(&w, 1).jobTerm(cOre); ok {
			t.Errorf("jobTerm(ore) with InputJobs MaxInt64 ok = true, want false")
		}
		if _, _, ok := ctxFor(&w, 1).jobAndOutput(cOre); ok {
			t.Errorf("jobAndOutput(ore) with an overflowing job ok = true, want false")
		}
		x := *v
		x.Weight = math.MaxInt64
		x.Recipes = []RecipeView{x.Recipes[0], {Duration: 1, Effective: 10000, MaxExecutions: 1,
			Outputs: []RecipeOutput{{Commodity: cItem, Quantity: 2, CostShare: 10000}}}}
		if job, _, ok := ctxFor(&x, 1).jobAndOutput(cItem); ok || job != 0 {
			t.Errorf("jobAndOutput(item) with an overflowing output = job %d, ok %v, want job 0 and not ok", job, ok)
		}
	})
}

// TestMakesAndPurposeAndCandidate verifies the small deciders of the trade
// pass at their ordered neighbours: makes reads non-discard outputs of exactly
// the commodity, purposeOf names an input or a tool only for exactly that
// commodity, and candidate takes no ask for a negative expiring count.
//
// Requirement: FR-AGT-048
func TestMakesAndPurposeAndCandidate(t *testing.T) {
	t.Parallel()
	v := workView()
	v.Recipes = []RecipeView{{Duration: 1, Effective: 10000, MaxExecutions: 1,
		Inputs:  []RecipeLine{{Commodity: cItem, Quantity: 1}},
		Tools:   []RecipeTool{{Commodity: cTool, WearPerUse: 1, Durability: 10}},
		Outputs: []RecipeOutput{{Commodity: cOre, Quantity: 1, CostShare: 10000}, {Commodity: cStaple, Quantity: 1, Discard: true}}}}
	tc := ctxFor(v, 0)
	for _, tt := range []struct {
		c    CommodityID
		want bool
	}{{cOre, true}, {cStaple, false}, {cItem, false}, {cTool, false}} {
		if got := tc.makes(tt.c); got != tt.want {
			t.Errorf("makes(%d) = %v, want %v", tt.c, got, tt.want)
		}
	}
	for _, tt := range []struct {
		name     string
		c        CommodityID
		job      int64
		isUpkeep bool
		rstar    int
		want     Purpose
	}{
		{"input with a job", cItem, 1, false, 0, PurposeInput},
		{"input without a job is other", cItem, 0, false, 0, PurposeOther},
		{"tool", cTool, 0, false, 0, PurposeTool},
		{"a commodity above the input is other", cOre, 1, false, 0, PurposeOther},
		{"a commodity below the tool is other", cOre, 0, false, 0, PurposeOther},
		{"upkeep", cStaple, 0, true, 0, PurposeUpkeep},
		{"no planned recipe is not an input", cItem, 1, false, -1, PurposeOther},
		{"the second recipe is read when planned", cItem, 1, false, 1, PurposeInput},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := *v
			w.Recipes = append([]RecipeView{{Duration: 1, Effective: 10000}}, v.Recipes...)
			if tt.rstar == 0 {
				w.Recipes = v.Recipes
			}
			if got := ctxFor(&w, tt.rstar).purposeOf(tt.c, tt.job, tt.isUpkeep); got != tt.want {
				t.Errorf("purposeOf(%d, job %d, upkeep %v, recipe %d) = %v, want %v", tt.c, tt.job, tt.isUpkeep, tt.rstar, got, tt.want)
			}
		})
	}
	c := &CommodityView{Commodity: cItem}
	if kind, ask := tc.candidate(c, 0, 0, -3); kind != noOrder || ask != 0 {
		t.Errorf("candidate(no gap, no surplus, expiring -3) = %d, %d, want no order and 0", kind, ask)
	}
	if kind, ask := tc.candidate(c, 0, 4, -3); kind != askOrder || ask != 4 {
		t.Errorf("candidate(surplus 4, expiring -3) = %d, %d, want an ask of 4", kind, ask)
	}
	if kind, ask := tc.candidate(c, 0, 0, 5); kind != askOrder || ask != 5 {
		t.Errorf("candidate(expiring 5) = %d, %d, want an ask of 5", kind, ask)
	}
}

// TestCombineTiers verifies the money left and the quantity of a tier for an
// order that has not bought (the limit may be lowered to the money), for one
// that has (a later tier buys at the same limit or not at all), and the guards
// for a limit or a want below 1.
//
// Requirement: FR-AGT-044
func TestCombineTiers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		row               tradeRow
		avail, want       int64
		wantLeft, wantQty int64
		wantLimit         int64
		wantBought        bool
	}{
		{"first tier buys at the limit", tradeRow{limit: 10}, 25, 5, 5, 2, 10, true},
		{"first tier lowers the limit to the money", tradeRow{limit: 10}, 4, 3, 0, 1, 4, true},
		{"a later tier buys at the same limit", tradeRow{limit: 10, bought: true, qty: 2}, 35, 5, 5, 5, 10, true},
		{"a later tier that cannot pay one unit buys nothing", tradeRow{limit: 10, bought: true, qty: 2}, 7, 5, 7, 2, 10, true},
		{"no money buys nothing and leaves the row unbought", tradeRow{limit: 10}, 0, 5, 0, 0, 10, false},
		{"a limit of 0 is left alone", tradeRow{limit: 0, bought: true}, 10, 5, 10, 0, 0, true},
		{"a want of 0 is left alone", tradeRow{limit: 10, bought: true, qty: 1}, 10, 0, 10, 1, 10, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := tt.row
			left := r.combineTiers(tt.avail, tt.want)
			if left != tt.wantLeft || r.qty != tt.wantQty || r.limit != tt.wantLimit || r.bought != tt.wantBought {
				t.Errorf("combineTiers(%d, %d) on %+v: left %d qty %d limit %d bought %v, want %d %d %d %v",
					tt.avail, tt.want, tt.row, left, r.qty, r.limit, r.bought, tt.wantLeft, tt.wantQty, tt.wantLimit, tt.wantBought)
			}
		})
	}
}

// TestBuffer_SharesTheMoneyAndTheCapInOrder verifies that tier 3 takes each
// bid's buffer in canonical order out of the one money left and the one cap,
// that a limit of 1 still buys, and that an ask row never buys.
//
// With a full buy share (window position 0, favour floor 0): the first bid wants
// 5 at 10 and the second 5 at 10. With 70 money the first takes 5 (50) and the
// second the 2 that the other 20 buys; with a cap of 60 and ample money the
// first takes 5 (50) and the second the 1 that the other 10 of the cap buys.
//
// Requirement: FR-AGT-042, FR-AGT-043
func TestBuffer_SharesTheMoneyAndTheCapInOrder(t *testing.T) {
	t.Parallel()
	par := household.Params{FavourFloor: 0}
	bid := func(limit, t3 int64) tradeRow { return tradeRow{kind: bidOrder, limit: limit, t3: t3} }
	tests := []struct {
		name       string
		rows       []tradeRow
		avail, cap int64
		want       []int64
	}{
		{"money runs out in the second bid", []tradeRow{bid(10, 5), bid(10, 5)}, 70, 1000, []int64{5, 2}},
		{"the cap runs out in the second bid", []tradeRow{bid(10, 5), bid(10, 5)}, 1000, 60, []int64{5, 1}},
		{"a limit of 1 buys", []tradeRow{bid(1, 3)}, 100, 100, []int64{3}},
		{"an ask row never buys", []tradeRow{{kind: askOrder, limit: 10, t3: 5}, bid(10, 2)}, 100, 100, []int64{0, 2}},
		{"a bid with no money buys nothing", []tradeRow{bid(10, 5)}, 5, 100, []int64{0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rows := append([]tradeRow(nil), tt.rows...)
			tc := &tradeCtx{par: par, rows: rows}
			tc.buffer(tt.avail, tt.cap)
			got := make([]int64, len(rows))
			for i := range rows {
				got[i] = rows[i].qty
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("buffer(%d, %d) quantities = %v, want %v", tt.avail, tt.cap, got, tt.want)
			}
		})
	}
}

// TestEmit_SkipsOrdersOutOfRange verifies that an order with no quantity, a
// quantity above the bound, no limit or no kind gets no intent, and that a valid
// bid and ask are emitted in row order.
//
// Requirement: FR-AGT-039
func TestEmit_SkipsOrdersOutOfRange(t *testing.T) {
	t.Parallel()
	rows := []tradeRow{
		{kind: bidOrder, commodity: 0, qty: 5, limit: 0},
		{kind: bidOrder, commodity: 1, qty: 0, limit: 5},
		{kind: bidOrder, commodity: 2, qty: MaxQuantity + 1, limit: 5},
		{kind: askOrder, commodity: 3, ask: 0, limit: 5},
		{kind: noOrder, commodity: 4, qty: 5, limit: 5},
		{kind: bidOrder, commodity: 5, qty: 1, limit: 1, target: 7, reserve: 2},
		{kind: askOrder, commodity: 6, ask: 3, limit: 9},
		{kind: bidOrder, commodity: 7, qty: MaxQuantity, limit: 1},
	}
	var p TradePlan
	(&tradeCtx{rows: rows}).emit(&p)
	want := []Intent{
		{Commodity: 5, Side: Bid, Quantity: 1, Limit: 1, Target: 7, Reserve: 2},
		{Commodity: 6, Side: Ask, Quantity: 3, Limit: 9},
		{Commodity: 7, Side: Bid, Quantity: MaxQuantity, Limit: 1},
	}
	if !reflect.DeepEqual(p.Intents, want) {
		t.Errorf("emit() intents = %+v, want %+v", p.Intents, want)
	}
}
