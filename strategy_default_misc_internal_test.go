package economysim

import (
	"errors"
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/household"
)

// TestBeliefTunables_MatchBeliefParams verifies that the belief group of the
// view, converted at the call, equals beliefParams field by field, so the two
// cannot drift (D82).
//
// Guard: DefaultStrategy.Revise and Draw read the group and the world reads
// beliefParams; a parameter added to one and not the other would make a
// consumer's call of the default differ from the world's.
func TestBeliefTunables_MatchBeliefParams(t *testing.T) {
	t.Parallel()
	w := DefaultWorldParams()
	bt := beliefTunablesOf(beliefParams(w))
	if got, want := beliefParamsOf(&bt), beliefParams(w); got != want {
		t.Errorf("beliefParamsOf = %+v, want %+v", got, want)
	}
	if n := reflect.TypeOf(BeliefTunables{}).NumField(); n != reflect.TypeOf(beliefs.Params{}).NumField()-1 {
		t.Errorf("BeliefTunables has %d fields, want one fewer than beliefs.Params (MaxPrice)", n)
	}
}

// wrapsDefault is a consumer-style strategy that holds the default in a named
// field and delegates Revise to it (architecture.md 4.7).
type wrapsDefault struct{ inner DefaultStrategy }

func (w wrapsDefault) Revise(o *OutcomeView, r *rand.Rand) Belief { return w.inner.Revise(o, r) }

// TestDefaultStrategyRevise_EqualsUpdate verifies that Revise returns what
// beliefs.Update returns, over a grid of sides, fills, words, stocks and
// targets, and that a wrapper holding the default in a named field and
// delegating gives the same result (FR-AGT-050, spec-beliefs.md 10.5).
//
// Requirement: FR-AGT-050
func TestDefaultStrategyRevise_EqualsUpdate(t *testing.T) {
	t.Parallel()
	bt := beliefTunablesOf(beliefParams(DefaultWorldParams()))
	par := beliefParamsOf(&bt)
	wrapped := wrapsDefault{inner: DefaultStrategy{}}
	for _, side := range []Side{Bid, Ask} {
		for _, filled := range []Quantity{0, 3, 10} {
			for _, draw := range []uint64{0, 1 << 63, math.MaxUint64} {
				for _, stock := range []int64{0, 5, 100} {
					for _, target := range []Quantity{0, 20} {
						o := &OutcomeView{
							Side: side, Limit: 120, Quantity: 10, Filled: filled,
							Belief: Belief{Centre: 100, Width: 1000}, Bounds: Bounds{Floor: 50, Ceiling: 5000},
							Market: MarketReport{Price: 110, Age: 1, BestBid: 105, BestAsk: 115, HasBestBid: true, HasBestAsk: true},
							Stock:  stock, Target: target, Draw: draw, Params: bt,
						}
						want, _ := beliefs.Update(beliefs.Belief{Centre: 100, Width: 1000}, beliefs.Bounds{Floor: 50, Ceiling: 5000},
							beliefs.Outcome{Bid: side == Bid, Limit: 120, Quantity: 10, Filled: int64(filled)},
							beliefs.Quote{Price: 110, Age: 1, BestBid: 105, BestAsk: 115, HasBid: true, HasAsk: true},
							stock, int64(target), par, draw)
						w := Belief{Centre: Price(want.Centre), Width: want.Width}
						if got := (DefaultStrategy{}).Revise(o, nil); got != w {
							t.Errorf("Revise(%v, filled %d, draw %d, stock %d, target %d) = %+v, want %+v", side, filled, draw, stock, target, got, w)
						}
						if got := wrapped.Revise(o, nil); got != w {
							t.Errorf("wrapped Revise = %+v, want %+v", got, w)
						}
					}
				}
			}
		}
	}
}

// TestDefaultStrategy_AllocatesNothingWithScratch verifies the allocation limit
// of the default strategy in the world's path: with a warmed plan and a view
// that carries scratch, Work, Trade and Revise allocate nothing.
//
// Requirement: NFR-AGT-001
func TestDefaultStrategy_AllocatesNothingWithScratch(t *testing.T) {
	v := exampleView(8000)
	v.scratch = &viewScratch{}
	plan := &TradePlan{Intents: make([]Intent, 0, 16)}
	rng := newRand(&wordSource{words: exampleWords()})
	o := &OutcomeView{Side: Bid, Limit: 120, Quantity: 10, Filled: 3, Belief: Belief{Centre: 100, Width: 1000},
		Bounds: Bounds{Floor: 50, Ceiling: 5000}, Params: beliefTunablesOf(beliefParams(DefaultWorldParams()))}
	tests := []struct {
		name string
		fn   func()
	}{
		{"Work", func() { DefaultStrategy{}.Work(v, rng) }},
		{"Trade", func() { plan.Intents = plan.Intents[:0]; DefaultStrategy{}.Trade(v, rng, plan) }},
		{"Revise", func() { DefaultStrategy{}.Revise(o, rng) }},
	}
	for _, tt := range tests {
		tt.fn()
		if got := testing.AllocsPerRun(100, tt.fn); got != 0 {
			t.Errorf("%s allocates %v times per call, want 0", tt.name, got)
		}
	}
}

// TestDefaultStrategyTrade_AbsurdViewsDoNotPanic verifies that a view a world
// would never build is survived without a panic and without an order for the
// commodity (D96): a weight and an upkeep whose product overflows, a bid above
// MaxQuantity (which gets no order, where one at the bound is legal), and a view
// with nothing in it.
//
// Requirement: FR-AGT-059
func TestDefaultStrategyTrade_AbsurdViewsDoNotPanic(t *testing.T) {
	t.Parallel()
	huge := func(mutate func(v *AgentView)) *AgentView {
		v := combinedView(1, 1, math.MaxInt64)
		mutate(v)
		return v
	}
	tests := []struct {
		name string
		view *AgentView
		ok   func(p TradePlan) bool
	}{
		{"an empty view", &AgentView{}, func(p TradePlan) bool { return len(p.Intents) == 0 }},
		{"weight times upkeep overflows", huge(func(v *AgentView) { v.Weight, v.Upkeep[0].Quantity = math.MaxInt64, 3 }),
			func(p TradePlan) bool { return len(p.Intents) == 0 }},
		{"upkeep cost overflows", huge(func(v *AgentView) {
			v.Upkeep[0].Quantity, v.Commodities[0].Belief.Centre = MaxQuantity, MaxPrice
			v.Tunables.ReserveTicks = math.MaxInt64
		}), func(p TradePlan) bool { return len(p.Intents) == 0 }},
		{"only the upkeep cost sum overflows", func() *AgentView {
			// Ten rows of MaxQuantity at MaxPrice sum past MaxInt64; fuel alone would be bid for.
			v := combinedView(1, 3, MaxMoney)
			v.Commodities[1].Belief.Centre = MaxPrice
			for range 10 {
				v.Upkeep = append(v.Upkeep, UpkeepRow{Commodity: xItem, Quantity: MaxQuantity})
			}
			return v
		}(), func(p TradePlan) bool { return len(p.Intents) == 0 }},
		{"a bid at MaxQuantity and MaxPrice is legal", huge(func(v *AgentView) {
			v.Upkeep[0].Quantity, v.Commodities[0].Belief.Centre = MaxQuantity, MaxPrice
			v.Money = MaxMoney
		}), func(p TradePlan) bool {
			return len(p.Intents) == 1 && p.Intents[0].Quantity == MaxQuantity && p.Intents[0].Limit == MaxPrice
		}},
		{"a bid above MaxQuantity gets no order (D96)", huge(func(v *AgentView) {
			v.Upkeep[0].Quantity, v.Commodities[0].Belief.Centre = MaxQuantity-1, 1
			v.Money = MaxMoney
		}), func(p TradePlan) bool { return len(p.Intents) == 0 }},
		{"negative everything", huge(func(v *AgentView) {
			v.Money, v.Weight, v.Commodities[0].Stock, v.Commodities[0].Expiring = -5, -1, -9, -3
		}), func(p TradePlan) bool { return true }},
		{"bounds that are no bounds", huge(func(v *AgentView) { v.Commodities[0].Bounds = Bounds{} }),
			func(p TradePlan) bool { return len(p.Intents) == 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, _ := runTrade(tt.view, nil)
			if !tt.ok(p) {
				t.Errorf("Trade() = %+v", p.Intents)
			}
			(DefaultStrategy{}).Work(tt.view, nil)
		})
	}
}

// TestDefaultStrategyTrade_NoWindowUsesEmptyPos verifies that a commodity with
// no price window takes the position EmptyPos (5 000) for the tier-3 buy share,
// whatever Position holds: the 16.2 example without windows still buys
// floor(3 * 6 000 / 10 000) = 1 fuel buffer and 4 staple.
//
// Requirement: FR-AGT-043
func TestDefaultStrategyTrade_NoWindowUsesEmptyPos(t *testing.T) {
	t.Parallel()
	v := exampleView(8000)
	for i := range v.Commodities {
		v.Commodities[i].Window = PriceWindowView{Position: 9999}
	}
	got, _ := runTrade(v, exampleWords())
	want := []Intent{
		{Commodity: xFuel, Side: Bid, Quantity: 2, Limit: 105, Target: 3, Reserve: 1, Purpose: PurposeUpkeep},
		{Commodity: xStaple, Side: Bid, Quantity: 4, Limit: 310, Target: 9, Reserve: 3, Purpose: PurposeUpkeep},
		{Commodity: xOre, Side: Bid, Quantity: 2, Limit: 195, Target: 2, Purpose: PurposeInput},
		{Commodity: xItem, Side: Ask, Quantity: 1, Limit: 900, Target: 1, Purpose: PurposeOther},
	}
	if !reflect.DeepEqual(got.Intents, want) {
		t.Errorf("Trade() intents = %+v, want %+v", got.Intents, want)
	}
}

// TestResolveStrategy verifies the name check: "default" is DefaultStrategy,
// "directed" is a Directed with DefaultStrategy as its fallback, a registered
// name finds its extension, and an unknown or empty name is a *ValidationError
// at the path with rule ErrUnknownStrategy. isDirected is true for the built-in
// directed strategy.
//
// Requirement: FR-AGT-030, FR-CFG-038
func TestResolveStrategy(t *testing.T) {
	t.Parallel()
	custom := newFake()
	o, err := applyOptions([]Option{WithStrategy("custom", custom)})
	if err != nil {
		t.Fatalf("applyOptions: %v", err)
	}
	path := []string{"roles", "worker_a", "strategy"}
	tests := []struct {
		name     string
		in       string
		want     Strategy
		directed bool
		fail     bool
	}{
		{"default", "default", DefaultStrategy{}, false, false},
		{"directed", "directed", Directed{Fallback: DefaultStrategy{}}, true, false},
		{"registered", "custom", custom, false, false},
		{"unknown", "other", nil, false, true},
		{"empty", "", nil, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveStrategy(&o, tt.in, path)
			if !tt.fail {
				if err != nil || !reflect.DeepEqual(got, tt.want) || isDirected(got) != tt.directed {
					t.Errorf("resolveStrategy(%q) = %v, %v, want %v, nil (directed %v)", tt.in, got, err, tt.want, tt.directed)
				}
				return
			}
			var ve *ValidationError
			if !errors.Is(err, ErrUnknownStrategy) || !errors.As(err, &ve) {
				t.Fatalf("resolveStrategy(%q) error = %v, want a *ValidationError of ErrUnknownStrategy", tt.in, err)
			}
			if !slices.Equal(ve.Path, path) || ve.Value != tt.in || got != nil {
				t.Errorf("resolveStrategy(%q) = %v, path %v, value %q, want nil, path %v, value %q", tt.in, got, ve.Path, ve.Value, path, tt.in)
			}
		})
	}
}

// BenchmarkDefaultStrategy reports the cost and allocations of the three
// methods on the view of 16.2 (NFR-AGT-003; a benchmark never fails CI, the
// allocation limit is TestDefaultStrategy_AllocatesNothingWithScratch).
func BenchmarkDefaultStrategy(b *testing.B) {
	v := exampleView(8000)
	v.scratch = &viewScratch{}
	plan := &TradePlan{Intents: make([]Intent, 0, 16)}
	rng := newRand(&wordSource{words: exampleWords()})
	o := &OutcomeView{Side: Bid, Limit: 120, Quantity: 10, Filled: 3, Belief: Belief{Centre: 100, Width: 1000},
		Bounds: Bounds{Floor: 50, Ceiling: 5000}, Params: beliefTunablesOf(beliefParams(DefaultWorldParams()))}
	b.Run("Work", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			DefaultStrategy{}.Work(v, rng)
		}
	})
	b.Run("Trade", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			plan.Intents = plan.Intents[:0]
			DefaultStrategy{}.Trade(v, rng, plan)
		}
	})
	b.Run("Revise", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			DefaultStrategy{}.Revise(o, rng)
		}
	})
	benchParts(b)
}

// benchParts reports the passes of the default strategy one by one
// (NFR-AGT-003): the targets, the money allocation, the work share and the
// recipe choice.
func benchParts(b *testing.B) {
	v := exampleView(8000)
	par := householdParams(&v.Tunables)
	var saved []tradeRow
	ctx, _ := consideredHook(v, exampleWords(), func(rows []tradeRow) { saved = append([]tradeRow(nil), rows...) })
	cost, _ := v.upkeepCost()
	savings, _ := household.ReserveMoney(cost, par.ReserveTicks)
	wv := workView()
	wv.Recipes = append(wv.Recipes, wv.Recipes[0], wv.Recipes[0])
	wv.Recipes[1].Recipe, wv.Recipes[2].Recipe = 1, 2
	var buf workReportBuf
	b.Run("Targets", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			household.ComputeTargets(household.Need{Upkeep: 3, Job: 2, OutTgt: 1}, par)
		}
	})
	b.Run("Allocate", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			copy(ctx.rows, saved)
			ctx.allocate(cost, savings)
		}
	})
	b.Run("WorkShare", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			workShare(wv, 2700)
		}
	})
	b.Run("RecipeChoice", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			workReport(wv, &buf)
		}
	})
}
