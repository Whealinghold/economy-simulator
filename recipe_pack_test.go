package economysim_test

import (
	"os"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// TestRecipeTable_TwoSettings verifies over the packs one and second that
// every row's commodity IDs name the strings the configuration lists, that the
// source flag is "no inputs", and that the makers of every commodity are
// exactly the recipes that name it as a non-discard output.
//
// Requirement: FR-PRO-002, FR-PRO-003, FR-PRO-032
//
// Guard: architecture.md 9.4, the table must work for any setting; a rule
// tuned to one pack's names would pass the reference chain and fail the other.
func TestRecipeTable_TwoSettings(t *testing.T) {
	t.Parallel()
	fsys := os.DirFS("testdata/chains")
	packs := map[string][]string{
		"one":    {"base.toml", "one.toml", "warm-one.toml"},
		"second": {"second/second.toml", "second/warm-second.toml"},
	}
	for name, layers := range packs {
		raw, err := config.LoadFS(fsys, layers...)
		if err != nil {
			t.Fatalf("pack %s: LoadFS error = %v, want nil", name, err)
		}
		cfg := economysim.Canonical(raw)
		tab := economysim.NewRecipeTable(&cfg)
		for ri, r := range cfg.Recipes {
			v := economysim.RecipeRow(&tab, economysim.RecipeID(ri))
			var in, out, tools []string
			for _, id := range v.Inputs {
				in = append(in, cfg.Commodities[id].ID)
			}
			for _, id := range v.Outputs {
				out = append(out, cfg.Commodities[id].ID)
			}
			for _, id := range v.Tools {
				tools = append(tools, cfg.Commodities[id].ID)
			}
			var wantIn, wantOut, wantTools []string
			for _, x := range r.Inputs {
				wantIn = append(wantIn, x.Commodity)
			}
			for _, x := range r.Outputs {
				wantOut = append(wantOut, x.Commodity)
			}
			for _, x := range r.Requires {
				wantTools = append(wantTools, x.Commodity)
			}
			if !slices.Equal(in, wantIn) || !slices.Equal(out, wantOut) || !slices.Equal(tools, wantTools) {
				t.Errorf("pack %s recipe %s: rows (%v, %v, %v), want (%v, %v, %v)", name, r.ID, in, out, tools, wantIn, wantOut, wantTools)
			}
			if v.Source != (len(r.Inputs) == 0) {
				t.Errorf("pack %s recipe %s: source = %v, want %v", name, r.ID, v.Source, len(r.Inputs) == 0)
			}
		}
		for ci, c := range cfg.Commodities {
			var want []economysim.RecipeID
			for ri, r := range cfg.Recipes {
				for _, o := range r.Outputs {
					if o.Commodity == c.ID && !o.Discard {
						want = append(want, economysim.RecipeID(ri))
						break
					}
				}
			}
			if got := economysim.RecipeMakers(&tab, economysim.CommodityID(ci)); !slices.Equal(got, want) {
				t.Errorf("pack %s: makers(%s) = %v, want %v", name, c.ID, got, want)
			}
		}
	}
}
