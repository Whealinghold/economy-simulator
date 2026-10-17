package economysim_test

import (
	"os"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// TestProductionAnchors_TwoSettings verifies over the packs one and second,
// with no code specific to either, that the cost anchor of every (region,
// commodity) at a seed price of 100 for every commodity, a wage of 300 and
// every recipe runnable equals an independent computation from the
// configuration's strings: the least, over the recipes with the commodity as a
// non-discard output of share above 0 and a factor above 0, of
// ceil(cost * share / (outQty * factor)) with cost = sum q*100 + sum
// ceil(wear*100/durability) + duration*300, a tool of durability 0 making the
// cost unknown (0); and that the root's read returns the same cells.
//
// Arithmetic: a recipe of 2 inputs of 1, no tool, duration 1, one output of 1
// at share 10 000 and factor 10 000 costs ceil((200+300)*10000/(1*10000)) =
// 500.
//
// Requirement: FR-PRO-037, FR-PRO-001, FR-PRO-028
//
// Guard: architecture.md 9.4, an anchor tuned to one pack's names would pass
// the reference chain and fail the other.
func TestProductionAnchors_TwoSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	fsys := os.DirFS("testdata/chains")
	packs := map[string][]string{
		"one":    {"base.toml", "one.toml", "warm-one.toml"},
		"second": {"second/second.toml", "second/warm-second.toml"},
	}
	const price, wage = 100, 300
	for name, layers := range packs {
		raw, err := config.LoadFS(fsys, layers...)
		if err != nil {
			t.Fatalf("pack %s: LoadFS error = %v, want nil", name, err)
		}
		cfg := economysim.Canonical(raw)
		nc := len(cfg.Commodities)
		com := map[string]int{}
		for i, c := range cfg.Commodities {
			com[c.ID] = i
		}
		var prices, wages []int64
		var all []economysim.RecipeID
		for i := range cfg.Recipes {
			all = append(all, economysim.RecipeID(i))
		}
		runnable := make([][]economysim.RecipeID, len(cfg.Regions))
		for g := range cfg.Regions {
			for range nc {
				prices = append(prices, price)
			}
			wages = append(wages, wage)
			runnable[g] = all
		}
		table, read := economysim.ProductionAnchors(&cfg, prices, wages, runnable, -1)
		if len(table) != len(cfg.Regions)*nc {
			t.Fatalf("pack %s: table length = %d, want %d", name, len(table), len(cfg.Regions)*nc)
		}
		yt := economysim.NewYieldTable(&cfg)
		nonZero := 0
		for g := range cfg.Regions {
			want := make([]int64, nc)
			for ri, r := range cfg.Recipes {
				e, _ := economysim.YieldSite(&yt, economysim.RegionID(g), economysim.RecipeID(ri), -1, r.Sited)
				if e <= 0 {
					continue
				}
				cost, known := r.Duration*wage, true
				for _, in := range r.Inputs {
					cost += int64(in.Quantity) * price
				}
				for _, tl := range r.Requires {
					d := cfg.Commodities[com[tl.Commodity]].Durability
					if d <= 0 {
						known = false
						continue
					}
					cost += (tl.WearPerUse*price + d - 1) / d
				}
				if !known {
					continue
				}
				for _, o := range r.Outputs {
					if o.Discard || o.CostShare <= 0 {
						continue
					}
					den := int64(o.Quantity) * e
					u := (cost*o.CostShare + den - 1) / den
					if c := com[o.Commodity]; want[c] == 0 || u < want[c] {
						want[c] = u
					}
				}
			}
			for c := range nc {
				got := table[g*nc+c]
				if got != want[c] {
					t.Errorf("pack %s: anchor(%s, %s) = %d, want %d", name, cfg.Regions[g].ID, cfg.Commodities[c].ID, got, want[c])
				}
				if r := read(economysim.RegionID(g), economysim.CommodityID(c)); r != got {
					t.Errorf("pack %s: unitCost(%s, %s) = %d, want %d", name, cfg.Regions[g].ID, cfg.Commodities[c].ID, r, got)
				}
				if got > 0 {
					nonZero++
				}
			}
		}
		if nonZero == 0 {
			t.Errorf("pack %s: every anchor is 0, want some non-zero", name)
		}
	}
}
