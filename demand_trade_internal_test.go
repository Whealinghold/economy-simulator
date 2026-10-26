package economysim

import "testing"

// wantsView is a household with upkeep of fuel and staple and no recipe, with a
// two-rung ladder whose items are ore (rank 1) and item (rank 2), the planned
// units given. It holds none of the four commodities.
func wantsView(money, planned1, planned2 int64) *AgentView {
	cs := []CommodityView{cview(xFuel, 0, 100, 1000), cview(xStaple, 0, 300, 1000), cview(xOre, 0, 200, 1000), cview(xItem, 0, 900, 1000)}
	for i := range cs {
		cs[i].Window = PriceWindowView{Position: 4000, HasWindow: true}
	}
	return &AgentView{
		Weight: 1, Money: money, StartsIn: 1,
		Upkeep:      []UpkeepRow{{Commodity: xFuel, Quantity: 1}, {Commodity: xStaple, Quantity: 3}},
		Commodities: cs, Tunables: testTunables(),
		Wants: []WantView{
			{Want: 0, Planned: Quantity(planned1), Items: []WantItemView{{Commodity: xOre, Share: 10_000, Units: Quantity(planned1)}}},
			{Want: 1, Planned: Quantity(planned2), Items: []WantItemView{{Commodity: xItem, Share: 10_000, Units: Quantity(planned2)}}},
		},
	}
}

// tradeOf runs the default strategy's Trade on v with every word 0.
func tradeOf(v *AgentView) *TradePlan {
	var p TradePlan
	DefaultStrategy{}.Trade(v, newRand(&wordSource{words: make([]uint64, 8)}), &p)
	return &p
}

// fundedOf is the units of want w that the plan funded.
func fundedOf(p *TradePlan, w WantID) int64 {
	for _, f := range p.WantFunded {
		if f.Want == w {
			return int64(f.Units)
		}
	}
	return 0
}

// spend is the money the bids of the plan can cost.
func spend(p *TradePlan) (all, upkeep int64) {
	for _, in := range p.Intents {
		if in.Side != Bid {
			continue
		}
		cost := int64(in.Limit) * int64(in.Quantity)
		all += cost
		if in.Commodity == xFuel || in.Commodity == xStaple {
			upkeep += cost
		}
	}
	return all, upkeep
}

// TestDemand_BidMerge verifies that a want unit is a part of the agent's single
// bid for the commodity and not a second order: one intent per commodity, its
// WantUnits at most its Quantity, its Target at least the planned units, its
// purpose Other for a commodity that is only a want item, the funded units of the
// plan equal to the WantUnits of the intents, and the sum of limit times quantity
// within the money. With no wants in the view the bid is exactly the bid without
// them.
//
// Requirement: FR-DEM-019, FR-DEM-021, FR-DEM-022, FR-DEM-024, FR-AGT-076
func TestDemand_BidMerge(t *testing.T) {
	t.Parallel()
	v := wantsView(1_000_000, 4, 3)
	p := tradeOf(v)
	seen := map[CommodityID]bool{}
	var wantUnits int64
	for _, in := range p.Intents {
		if seen[in.Commodity] {
			t.Errorf("Trade() placed two orders for commodity %d, want one", in.Commodity)
		}
		seen[in.Commodity] = true
		if in.WantUnits > in.Quantity {
			t.Errorf("intent %+v has WantUnits above Quantity", in)
		}
		wantUnits += int64(in.WantUnits)
		if (in.Commodity == xOre || in.Commodity == xItem) && in.Purpose != PurposeOther {
			t.Errorf("want bid %+v has purpose %v, want PurposeOther", in, in.Purpose)
		}
	}
	if got := fundedOf(p, 0) + fundedOf(p, 1); got != wantUnits || wantUnits != 7 {
		t.Errorf("funded %d, WantUnits %d, want both 7 (4 + 3 planned, ample money)", got, wantUnits)
	}
	for _, in := range p.Intents {
		if in.Commodity == xOre && int64(in.Target) < 4 {
			t.Errorf("ore Target = %d, want the planned 4 added to it", in.Target)
		}
	}
	if all, _ := spend(p); all > v.Money {
		t.Errorf("bids cost %d, want at most the money %d", all, v.Money)
	}
	off := wantsView(1_000_000, 4, 3)
	off.Wants = nil
	if q := tradeOf(off); len(q.WantFunded) != 0 {
		t.Errorf("Trade() with no wants funded %+v, want none", q.WantFunded)
	} else {
		for _, in := range q.Intents {
			if in.WantUnits != 0 || in.Commodity == xOre || in.Commodity == xItem {
				t.Errorf("Trade() with no wants placed %+v, want only the upkeep bids", in)
			}
		}
	}
}

// TestDemand_ReserveGate verifies that a household whose money is at or below
// its reserve plus the upkeep tiers buys no want, and one with money above it
// does.
//
// Requirement: FR-DEM-020, FR-DEM-016
func TestDemand_ReserveGate(t *testing.T) {
	t.Parallel()
	if p := tradeOf(wantsView(2_500, 4, 3)); len(p.WantFunded) != 0 {
		t.Errorf("Trade() with money below the reserve funded %+v, want no want", p.WantFunded)
	}
	if p := tradeOf(wantsView(1_000_000, 4, 3)); len(p.WantFunded) == 0 {
		t.Errorf("Trade() with ample money funded no want")
	}
}

// TestDemand_Engel verifies Engel's law of the rules (spec-demand.md 5.3): as the
// money rises the upkeep share of the spending falls, and a want of higher rank
// is funded only once the want of lower rank is funded in full.
//
// Requirement: FR-DEM-029, FR-DEM-017
func TestDemand_Engel(t *testing.T) {
	t.Parallel()
	prev := int64(10_001)
	var sawTop bool
	for _, money := range []int64{2_000, 4_000, 6_000, 8_000, 12_000, 20_000, 50_000, 1_000_000} {
		p := tradeOf(wantsView(money, 6, 6))
		all, up := spend(p)
		if all == 0 {
			continue
		}
		share := up * 10_000 / all
		if share > prev {
			t.Errorf("money %d: upkeep share %d rose above %d at the lower money", money, share, prev)
		}
		prev = share
		if fundedOf(p, 1) > 0 {
			sawTop = true
			if fundedOf(p, 0) != 6 {
				t.Errorf("money %d: the second want is funded while the first has %d of 6 units", money, fundedOf(p, 0))
			}
		}
	}
	if !sawTop {
		t.Errorf("no money level funded the second want, want one that does")
	}
}
