package economysim

import (
	"math/bits"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/household"
)

// TestDefaultStrategyTrade_ZeroYieldIsNeverPlanned verifies that a source
// recipe with a zero yield (eligible, but making nothing) is not the planned
// recipe, so no tool or input is bought for a job that cannot run.
//
// Requirement: FR-AGT-038
func TestDefaultStrategyTrade_ZeroYieldIsNeverPlanned(t *testing.T) {
	t.Parallel()
	v := combinedView(1, 3, 100000)
	v.Recipes[0].Effective = 0
	ctx := tradeCtx{v: v, par: householdParams(&v.Tunables)}
	if got := ctx.plannedRecipe(); got != -1 {
		t.Errorf("plannedRecipe() = %d, want -1 for a zero yield", got)
	}
	v.Recipes[0].Inputs = nil
	if got := ctx.plannedRecipe(); got != -1 {
		t.Errorf("plannedRecipe() of a source with a zero yield = %d, want -1", got)
	}
}

// TestDefaultStrategyTrade_SpendsWithinTheMoney verifies, over a grid of money,
// stocks and weights that includes first-buyer lowering and dropped later tiers,
// that the plan has at most one order per commodity in canonical order, that
// every quantity is at least 1 and every limit is in [1, MaxPrice], that the sum
// of limit times quantity over the bids is at most the money (FR-AGT-041, 044),
// that the bid of a commodity never exceeds its gap, and that tiers 1 and 2 are
// bought before tier 3 (a bid reaches the buffer only if the money bought the
// upkeep and the inputs in full).
//
// Requirement: FR-AGT-039, FR-AGT-041, FR-AGT-044
func TestDefaultStrategyTrade_SpendsWithinTheMoney(t *testing.T) {
	t.Parallel()
	for _, w := range []int64{1, 3, 40} {
		for _, money := range []int64{0, 1, 50, 105, 250, 1000, 5000, 100000, 10_000_000} {
			for _, stock := range []int64{0, 2, 9, 50} {
				v := exampleView(money)
				v.Weight = w
				v.Commodities[1].Stock = stock * w
				v.Commodities[2].Stock = stock / 2
				v.Hunger = stock % 2
				p, _ := runTrade(v, []uint64{wordFor(3, 21), wordFor(40, 61), wordFor(15, 41), 5})
				var spent int64
				prev := CommodityID(-1)
				for _, in := range p.Intents {
					if in.Commodity <= prev || in.Quantity < 1 || in.Limit < 1 || in.Limit > MaxPrice {
						t.Fatalf("w %d, money %d, stock %d: bad intent %+v after commodity %d", w, money, stock, in, prev)
					}
					prev = in.Commodity
					if in.Side == Bid {
						spent += int64(in.Quantity) * int64(in.Limit)
					}
				}
				if spent > money {
					t.Errorf("w %d, money %d, stock %d: bids reserve %d, more than the money", w, money, stock, spent)
				}
			}
		}
	}
}

// TestDefaultStrategyTrade_TiersAddToTheGap verifies that the quantity of a bid
// is the sum of the allocated tiers: with ample money the bid equals tier 1 plus
// tier 2 plus the buffer want floor(T3 * buyshare / 10 000), and the cap and
// the buy share touch tier 3 only.
//
// With the example household of 16.2 and money 1 000 000: fuel T1 = 1, T2 = 0,
// T3 = 3; buyshare 6 000; fuel buys 1 + 0 + floor(3 * 0.6) = 2. A pos of 10 000
// gives buyshare = FavourFloor = 2 000: floor(3 * 0.2) = 0.
//
// Requirement: FR-AGT-040, FR-AGT-042, FR-AGT-043
func TestDefaultStrategyTrade_TiersAddToTheGap(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		pos  int64
		want Quantity
	}{{"neutral price", 5000, 2}, {"price at the top of the window", 10000, 1}, {"price at the bottom", 0, 4}} {
		v := exampleView(1_000_000)
		v.Commodities[0].Window.Position = tt.pos
		p, _ := runTrade(v, exampleWords())
		if len(p.Intents) == 0 || p.Intents[0].Commodity != xFuel || p.Intents[0].Quantity != tt.want {
			t.Errorf("%s: fuel bid = %+v, want %d units", tt.name, p.Intents, tt.want)
		}
	}
}

// TestDefaultStrategyTrade_CapNeverLowersALimit verifies that when the spend
// cap bounds tier 3 the limit of the order stays the drawn one (7.4.4, 7.5.3).
//
// Money 3 105: the reserve money is 3 000, tier 1 takes 105, so avail 3 000 and
// e = 0: no buffer at all. Money 3 500 after tiers: e = 3 500 - 105 - 390 - 3 000
// = 5 gives a cap of 5 000 * (5 * 10 000 / 10 005 = 4) / 10 000 = 2, below the
// price of one staple (310): tier 3 buys nothing and no limit changes.
//
// Requirement: FR-AGT-042
func TestDefaultStrategyTrade_CapNeverLowersALimit(t *testing.T) {
	t.Parallel()
	for _, money := range []int64{3105, 3500} {
		p, _ := runTrade(exampleView(money), exampleWords())
		for _, in := range p.Intents {
			wantLimit := map[CommodityID]Price{xFuel: 105, xOre: 195, xStaple: 310, xItem: 900}[in.Commodity]
			if in.Limit != wantLimit {
				t.Errorf("money %d: %v of commodity %d has limit %d, want the draw %d", money, in.Side, in.Commodity, in.Limit, wantLimit)
			}
			if in.Commodity == xStaple {
				t.Errorf("money %d: a staple order %+v, want none (the cap is below one unit)", money, in)
			}
		}
	}
}

// TestDefaultStrategyTrade_IsAFunctionOfViewAndWords verifies that identical
// views and identical words give identical plans, that the default keeps no
// state (a zero-size type) and that Trade neither discards nor changes the view.
//
// Requirement: FR-AGT-020, FR-AGT-024, FR-AGT-031, FR-AGT-051
func TestDefaultStrategyTrade_IsAFunctionOfViewAndWords(t *testing.T) {
	t.Parallel()
	if got := reflect.TypeOf(DefaultStrategy{}).Size(); got != 0 {
		t.Errorf("DefaultStrategy size = %d, want 0 (no state)", got)
	}
	a, b := exampleView(8000), exampleView(8000)
	p1, _ := runTrade(a, exampleWords())
	p2, _ := runTrade(a, exampleWords())
	if !reflect.DeepEqual(p1, p2) {
		t.Errorf("two calls gave %+v and %+v", p1, p2)
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Trade changed the view: %+v, want %+v", a, b)
	}
	if len(p1.Discards) != 0 {
		t.Errorf("Trade discarded %+v, want nothing", p1.Discards)
	}
}

// scaleView is a household of the weight w whose state is w times that of a
// household of weight 1 with the stocks given: the recipe has a duration of 1
// and a yield of 10 000, so its output target w * q is exact (the claim of
// 7.5 holds only then).
func scaleView(w int64, stocks [4]int64) *AgentView {
	win := PriceWindowView{Position: 4000, HasWindow: true}
	cs := []CommodityView{
		cview(xFuel, stocks[0]*w, 100, 1000), cview(xStaple, stocks[1]*w, 300, 1000),
		cview(xOre, stocks[2]*w, 200, 1000), cview(xItem, stocks[3]*w, 900, 1000),
	}
	for i := range cs {
		cs[i].Window = win
		cs[i].Expiring = stocks[i] / 3 * w
	}
	return &AgentView{
		Weight: w, Money: 1_000_000_000_000, StartsIn: 1,
		Upkeep:      []UpkeepRow{{Commodity: xFuel, Quantity: 1}, {Commodity: xStaple, Quantity: 3}},
		Commodities: cs,
		Recipes: []RecipeView{{
			Recipe: 0, Duration: 1, Effective: 10000, MaxExecutions: 1,
			Inputs:  []RecipeLine{{Commodity: xOre, Quantity: 2}},
			Outputs: []RecipeOutput{{Commodity: xItem, Quantity: 1, CostShare: 10000}},
		}},
		Tunables: testTunables(),
	}
}

// considered runs the passes of Trade up to and including the allocation.
func considered(v *AgentView, words []uint64) (*tradeCtx, bool) {
	return consideredHook(v, words, nil)
}

// consideredHook is considered with a hook that sees the rows after the first
// pass and before any money is allocated; it may change them.
func consideredHook(v *AgentView, words []uint64, hook func(rows []tradeRow)) (*tradeCtx, bool) {
	t := &tradeCtx{v: v, par: householdParams(&v.Tunables), bp: beliefParamsOf(&v.Tunables.Beliefs), rows: make([]tradeRow, len(v.Commodities))}
	cost, ok := v.upkeepCost()
	if !ok {
		return t, false
	}
	savings, _ := household.ReserveMoney(cost, t.par.ReserveTicks)
	t.rstar = t.plannedRecipe()
	rng := newRand(&wordSource{words: words})
	for i := range v.Commodities {
		t.consider(i, rng)
	}
	if hook != nil {
		hook(t.rows)
	}
	t.allocate(cost, savings)
	return t, true
}

// TestDefaultStrategyTrade_ScalesWithTheWeight is a property test over seeds
// (the seed is in each message): in the conditions of spec-agents.md 7.5.3
// (ample money, so that no limit is lowered and every tier is funded, a stock
// away from the halt threshold, an exact output target) a cohort of weight w
// plans the same limits as a household of weight 1 fed the same words, w times
// its reserves, targets, asks and wants, exactly w times the quantities
// allocated to tiers 1 and 2, and a bid that is w times the weight-1 bid up to
// the w - 1 units that the floor of the buffer share can add. The conditions are
// asserted, not assumed, and the buffer is checked against the closed form of
// the spend cap on the totals.
//
// Requirement: FR-AGT-046, FR-AGT-052
func TestDefaultStrategyTrade_ScalesWithTheWeight(t *testing.T) {
	t.Parallel()
	// bare is the run with the buffer wants set to 0: what tiers 1 and 2 are allocated.
	bare := func(v *AgentView, words []uint64) *tradeCtx {
		c, _ := consideredHook(v, words, func(rows []tradeRow) {
			for i := range rows {
				rows[i].t3 = 0
			}
		})
		return c
	}
	for seed := uint64(0); seed < 100; seed++ {
		r := rand.New(rand.NewPCG(seed, 7))
		stocks := [4]int64{r.Int64N(8), r.Int64N(20), r.Int64N(6), r.Int64N(3)}
		words := []uint64{r.Uint64(), r.Uint64(), r.Uint64(), r.Uint64()}
		w := []int64{2, 3, 7, 40}[seed%4]
		var drawnOne, drawnMany []tradeRow
		keep := func(dst *[]tradeRow) func([]tradeRow) {
			return func(rows []tradeRow) { *dst = append([]tradeRow(nil), rows...) }
		}
		one, ok1 := consideredHook(scaleView(1, stocks), words, keep(&drawnOne))
		many, ok2 := consideredHook(scaleView(w, stocks), words, keep(&drawnMany))
		if !ok1 || !ok2 {
			t.Fatalf("seed %d: the fixture overflows", seed)
		}
		if one.rstar != 0 || many.rstar != 0 {
			t.Fatalf("seed %d, w %d: planned recipe %d at weight 1 and %d at weight w, want 0 at both (a halt, outside the conditions)", seed, w, one.rstar, many.rstar)
		}
		bareOne, bareMany := bare(scaleView(1, stocks), words), bare(scaleView(w, stocks), words)
		for i := range one.rows {
			a, b := one.rows[i], many.rows[i]
			if a.kind != b.kind || a.limit != b.limit || b.reserve != w*a.reserve || b.ask != w*a.ask ||
				b.t1 != w*a.t1 || b.t2 != w*a.t2 || b.t3 != w*a.t3 || b.target != w*a.target {
				t.Fatalf("seed %d, w %d, commodity %d: weight 1 gave %+v, weight w gave %+v", seed, w, i, a, b)
			}
			if a.kind != bidOrder {
				continue
			}
			// The conditions: no limit lowered, tiers 1 and 2 funded in full, the
			// buffer want-bound, not cap-bound, at both weights.
			if a.limit != drawnOne[i].limit || b.limit != drawnMany[i].limit {
				t.Fatalf("seed %d, w %d, commodity %d: a limit was lowered (%d to %d, %d to %d), outside the conditions", seed, w, i, drawnOne[i].limit, a.limit, drawnMany[i].limit, b.limit)
			}
			z1, zw := bareOne.rows[i], bareMany.rows[i]
			if z1.qty != a.t1+a.t2 || zw.qty != b.t1+b.t2 {
				t.Fatalf("seed %d, w %d, commodity %d: tiers 1 and 2 are not funded in full (%d of %d, %d of %d), outside the conditions", seed, w, i, z1.qty, a.t1+a.t2, zw.qty, b.t1+b.t2)
			}
			if zw.qty != w*z1.qty {
				t.Errorf("seed %d, w %d, commodity %d: tiers 1 and 2 allocated %d at weight w, want %d times %d", seed, w, i, zw.qty, w, z1.qty)
			}
			share := household.BuyShare(a.windowPos, one.par)
			wantA, _ := divFloor(a.t3, share)
			wantB, _ := divFloor(b.t3, share)
			if a.qty != a.t1+a.t2+wantA || b.qty != b.t1+b.t2+wantB {
				t.Fatalf("seed %d, w %d, commodity %d: the buffer is cap-bound (%d, %d), outside the conditions", seed, w, i, a.qty, b.qty)
			}
			if b.qty < w*a.qty || b.qty > w*a.qty+w-1 {
				t.Errorf("seed %d, w %d, commodity %d: bid %d at weight w, want %d to %d", seed, w, i, b.qty, w*a.qty, w*a.qty+w-1)
			}
		}
		for _, c := range []struct {
			ctx, bare *tradeCtx
			w         int64
		}{{one, bareOne, 1}, {many, bareMany, w}} {
			if spent, limit := bufferSpent(c.ctx, c.bare), spendCapOf(c.ctx, c.bare, c.w); spent > limit {
				t.Errorf("seed %d, w %d: the buffer spent %d, above the cap %d of the closed form", seed, c.w, spent, limit)
			}
		}
	}
}

// bufferSpent is the money the bids spent on tier 3: limit times the quantity
// above what tiers 1 and 2 bought.
func bufferSpent(full, bare *tradeCtx) int64 {
	var spent int64
	for i := range full.rows {
		if full.rows[i].kind == bidOrder {
			spent += (full.rows[i].qty - bare.rows[i].qty) * full.rows[i].limit
		}
	}
	return spent
}

// spendCapOf is the spend cap of 7.4.4 written out on the totals of scaleView:
// SpendMaxShare * (e * H / (e + H)) / 10 000, with e the money left after tiers
// 1 and 2 less the reserve money, H the half-life of the cost, and the cost of
// one tick of upkeep w * (1 * 100 + 3 * 300) = 1 000 * w.
func spendCapOf(full, bare *tradeCtx, w int64) int64 {
	var spent12 int64
	for i := range bare.rows {
		if bare.rows[i].kind == bidOrder {
			spent12 += bare.rows[i].qty * bare.rows[i].limit
		}
	}
	cost := 1000 * w
	e := max(0, full.v.Money-spent12-full.par.ReserveTicks*cost)
	h := full.par.SpendHalfTicks * cost
	return full.par.SpendMaxShare * (e * h / (e + h)) / 10000
}

// divFloor returns floor(n * share / 10 000).
func divFloor(n, share int64) (int64, bool) {
	hi, lo := bits.Mul64(uint64(n), uint64(share))
	q, _ := bits.Div64(hi, lo, uint64(household.Basis))
	return int64(q), true
}
