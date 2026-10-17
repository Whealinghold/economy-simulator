package economysim_test

import (
	"os"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
	"github.com/Whealinghold/economy-sim/internal/production"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// TestRoleTable_EffectiveWindowOverPacks verifies, for the packs one and second
// and every overlay that validates, that the stored window of every role equals
// max(WelfareWindow, WindowCycles * L), computed here from the Duration of the
// role's recipes, and is at most MaxWelfareWindow.
//
// Requirement: FR-WEL-009
func TestRoleTable_EffectiveWindowOverPacks(t *testing.T) {
	t.Parallel()
	fsys := os.DirFS("testdata/chains")
	checked, longer := 0, 0
	for name, layers := range rolePackLayers(t) {
		raw, err := config.LoadFS(fsys, layers...)
		if err != nil {
			if strings.HasPrefix(name, "overlay ") {
				continue // an overlay may name a module that is not built yet
			}
			t.Fatalf("pack %s: LoadFS error = %v, want nil", name, err)
		}
		if raw.Validate() != nil {
			continue
		}
		cfg := economysim.Canonical(raw)
		tab := economysim.NewRoleTable(&cfg)
		for ri, role := range cfg.Roles {
			var l int64
			for _, id := range role.Recipes {
				for _, rc := range cfg.Recipes {
					if rc.ID == id {
						l = max(l, rc.Duration)
					}
				}
			}
			want := max(cfg.World.WelfareWindow, cfg.World.WindowCycles*l)
			got := economysim.RoleWindow(&tab, economysim.RoleID(ri))
			if got != want || got > economysim.MaxWelfareWindow {
				t.Errorf("pack %s role %s: window = %d, want %d (at most %d)", name, role.ID, got, want, economysim.MaxWelfareWindow)
			}
			checked++
			if want > cfg.World.WelfareWindow {
				longer++
			}
		}
	}
	if checked == 0 || longer == 0 {
		t.Errorf("checked %d roles, %d with a window above the world's, want both above 0", checked, longer)
	}
}

// TestWearCost_AgreesWithProduction verifies that the wear cost of an execution
// in internal/welfare equals the wear term of UnitCost in internal/production
// for every tool of the packs one and second at its seed price. UnitCost does
// not expose the term alone, so each tool gets a recipe with no inputs, a
// duration of 1, a cost share of 10 000, one output unit and a yield factor of
// 10 000: the unit cost is the wear plus 1 * w for a known wage w, and the wear
// is that value less w.
//
// Guard: the two packages keep one copy of the formula each (plan 1-6, D66);
// the test is what stops them drifting.
//
// Requirement: FR-WEL-003
func TestWearCost_AgreesWithProduction(t *testing.T) {
	t.Parallel()
	const w = 1000
	fsys := os.DirFS("testdata/chains")
	packs := map[string][]string{
		"one":    {"base.toml", "one.toml", "warm-one.toml"},
		"second": {"second/second.toml", "second/warm-second.toml"},
	}
	tools := 0
	for name, layers := range packs {
		raw, err := config.LoadFS(fsys, layers...)
		if err != nil {
			t.Fatalf("pack %s: LoadFS error = %v, want nil", name, err)
		}
		cfg := economysim.Canonical(raw)
		prices := map[string][]int64{}
		for _, c := range cfg.Commodities {
			prices[c.ID] = []int64{int64(c.ReferencePrice)}
		}
		if cfg.WarmStart != nil {
			for _, sp := range cfg.WarmStart.Prices {
				prices[sp.Commodity] = append(prices[sp.Commodity], int64(sp.Price))
			}
		}
		durability := map[string]int64{}
		for _, c := range cfg.Commodities {
			durability[c.ID] = c.Durability
		}
		for _, rc := range cfg.Recipes {
			for _, tl := range rc.Requires {
				for _, price := range prices[tl.Commodity] {
					tools++
					d := durability[tl.Commodity]
					unit := production.UnitCost(nil, []production.Wear{{Commodity: 0, WearPerUse: tl.WearPerUse, Durability: d}}, []int64{price},
						production.Basis{Share: 10000, OutQty: 1, YieldBP: 10000, Duration: 1, Wage: w})
					got := welfare.WearCost(tl.WearPerUse, price, d, 1)
					if want := unit - w; got != want {
						t.Errorf("pack %s tool %s (wear %d, durability %d, price %d): welfare.WearCost = %d, production wear term = %d",
							name, tl.Commodity, tl.WearPerUse, d, price, got, want)
					}
				}
			}
		}
	}
	if tools == 0 {
		t.Errorf("checked %d tool prices, want above 0", tools)
	}
}
