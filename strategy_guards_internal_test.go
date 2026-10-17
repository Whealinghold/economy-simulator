package economysim

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestConsume_NoConsumptionBoundaryIsAnError verifies that eating goods from a
// ledger that has no consumption boundary account for the commodity returns an
// error and does not panic.
//
// Guard: the boundary accounts are opened at construction, so a world never
// meets this; without the check a missing account would be dereferenced as
// account 0 and the goods would be moved to the wrong place.
func TestConsume_NoConsumptionBoundaryIsAnError(t *testing.T) {
	t.Parallel()
	l, err := ledger.New(ledger.Config{Commodities: []ledger.Commodity{{}}, Regions: 1, MaxMoney: MaxMoney, MaxQuantity: MaxQuantity})
	if err != nil {
		t.Fatalf("ledger.New: %v", err)
	}
	// Only the initial boundary exists, so the agent can be given goods but
	// cannot eat them.
	src, err := l.OpenBoundary(ledger.Initial, 0)
	if err != nil {
		t.Fatalf("OpenBoundary: %v", err)
	}
	acct, err := l.Open(ledger.KindAgent, 0, 0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := l.Move(src, acct, 0, 5, ledger.Initial); err != nil {
		t.Fatalf("Move: %v", err)
	}
	a := agent{weight: 1}
	var sc consumeScratch
	if err := consume(l, &a, acct, []itemRow{{0, 2}}, &sc); err == nil {
		t.Errorf("consume with no consumption boundary = nil, want an error")
	}
}

// TestSum_AddRemembersAnOverflow verifies that the checked sum of the trade
// pass reports an overflow once and keeps reporting it, whatever is added
// afterwards, and that a term which did not fit is never added.
//
// Guard: the job terms are summed from view numbers; if an overflow were
// forgotten by the next term, a wrapped total would size a bid.
func TestSum_AddRemembersAnOverflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		steps []struct {
			x  int64
			ok bool
		}
		wantV   int64
		wantBad bool
	}{
		{"terms that fit add up", []struct {
			x  int64
			ok bool
		}{{2, true}, {3, true}}, 5, false},
		{"a term that did not fit marks the sum bad", []struct {
			x  int64
			ok bool
		}{{2, true}, {7, false}}, 2, true},
		{"the sum stays bad after a good term", []struct {
			x  int64
			ok bool
		}{{7, false}, {1, true}}, 0, true},
		{"an addition past int64 marks the sum bad", []struct {
			x  int64
			ok bool
		}{{math.MaxInt64, true}, {1, true}}, math.MaxInt64, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var s sum
			for _, st := range tt.steps {
				s.add(st.x, st.ok)
			}
			if s.bad != tt.wantBad {
				t.Errorf("sum.add(%v).bad = %v, want %v", tt.steps, s.bad, tt.wantBad)
			}
			if !tt.wantBad && s.v != tt.wantV {
				t.Errorf("sum.add(%v).v = %d, want %d", tt.steps, s.v, tt.wantV)
			}
		})
	}
}

// TestAgentView_CommodityOutsideTheTradableSet verifies that a commodity the
// view does not list has no index, a belief centre of 0 and a stock of 0, and
// that a listed one reads its own row.
//
// Guard: the work rules price inputs and tools by the belief centre; a recipe
// may name a commodity the role does not trade, and it must then read as
// worth nothing rather than index past the slice.
func TestAgentView_CommodityOutsideTheTradableSet(t *testing.T) {
	t.Parallel()
	v := &AgentView{Commodities: []CommodityView{cview(cStaple, 7, 300, 0)}}
	if got := v.commodityAt(cOre); got != -1 {
		t.Errorf("commodityAt(unlisted) = %d, want -1", got)
	}
	if got := v.centreOf(cOre); got != 0 {
		t.Errorf("centreOf(unlisted) = %d, want 0", got)
	}
	if got := v.stockOf(cOre); got != 0 {
		t.Errorf("stockOf(unlisted) = %d, want 0", got)
	}
	if v.centreOf(cStaple) != 300 || v.stockOf(cStaple) != 7 || v.commodityAt(cStaple) != 0 {
		t.Errorf("listed commodity: centre %d stock %d index %d, want 300, 7, 0", v.centreOf(cStaple), v.stockOf(cStaple), v.commodityAt(cStaple))
	}
}

// TestRecipeNet_ToolWearThatDoesNotFitIsNotOK verifies that a tool whose wear
// cost overflows, or whose durability is not positive, makes the net of the
// recipe report not ok instead of a wrapped value.
//
// Guard: ok false is what makes Work decline the recipe; a wrapped net could
// make an impossible recipe look like the best one.
func TestRecipeNet_ToolWearThatDoesNotFitIsNotOK(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		tool RecipeTool
	}{
		{"wear times price past int64", RecipeTool{Commodity: cTool, WearPerUse: math.MaxInt64, Durability: 1}},
		{"durability of zero", RecipeTool{Commodity: cTool, WearPerUse: 1, Durability: 0}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := workView() // the centre of the tool is 100
			r := RecipeView{Recipe: 0, Duration: 1, Effective: 10000, Tools: []RecipeTool{tt.tool}}
			if got, ok := v.recipeNet(&r); ok {
				t.Errorf("recipeNet(tool %+v) = %d, true, want not ok", tt.tool, got)
			}
		})
	}
}

// TestFillCommodities_UnknownCommodityIsAnError verifies that a role whose
// tradable set names a commodity the ledger does not know makes the fill return
// the ledger's refusal instead of panicking.
//
// Requirement: FR-AGT-071
func TestFillCommodities_UnknownCommodityIsAnError(t *testing.T) {
	t.Parallel()
	h := NewViewHarness(t, viewConfig())
	i := h.Spawn(t, "farmer", "north", 1)
	row := h.roles.row(h.agents[i].role)
	row.tradable = append(row.tradable[:len(row.tradable):len(row.tradable)], CommodityID(1<<20))
	h.agents[i].beliefs = append(h.agents[i].beliefs[:len(h.agents[i].beliefs):len(h.agents[i].beliefs)], h.agents[i].beliefs[0])
	if _, err := h.Fill(i); err == nil {
		t.Errorf("fillCommodities with an unknown tradable commodity = nil, want an error")
	}
}

// TestDefaultStrategyTrade_OverflowingTargetsPlanNoBid verifies that a
// commodity whose upkeep cost, or whose cover target, does not fit in an int64
// gets no bid from the default strategy, and that Trade does not panic.
//
// Weight 2 with an upkeep of MaxInt64 overflows the total (weight * quantity)
// and Trade plans nothing. With a belief centre of 1, an upkeep of MaxInt64/2
// and no reserve ticks the cost fits (MaxInt64/2 * 1) but the cover target
// (3 cover ticks * MaxInt64/2) does not, so only that commodity is skipped.
//
// Guard: a validated configuration bounds these products, so only a hand-built
// view reaches this; a wrapped target would size a bid from garbage.
func TestDefaultStrategyTrade_OverflowingTargetsPlanNoBid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		build func(v *AgentView)
	}{
		{"upkeep total past int64", func(v *AgentView) {
			v.Weight = 2
			v.Upkeep = []UpkeepRow{{Commodity: xFuel, Quantity: math.MaxInt64}}
		}},
		{"cover target past int64", func(v *AgentView) {
			v.Tunables.ReserveTicks = 0
			v.Commodities[0] = cview(xFuel, 0, 1, 0) // fuel is Commodities[0]
			v.Upkeep = []UpkeepRow{{Commodity: xFuel, Quantity: math.MaxInt64 / 2}}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := exampleView(1_000_000)
			tt.build(v)
			plan, _ := runTrade(v, exampleWords())
			for _, in := range plan.Intents {
				if in.Commodity == xFuel && in.Side == Bid {
					t.Errorf("Trade() planned %+v for an upkeep that overflows, want no bid for the commodity", in)
				}
			}
		})
	}
}
