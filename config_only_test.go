package economysim_test

import (
	"os"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// TestTables_ConfigurationOnlyAddition verifies that a commodity, a recipe
// that makes it and a role that runs the recipe, added to the second pack by
// configuration alone, appear in the commodity count, the recipe table's
// makers, the role table's recipe list and its tradable set, with no Go change.
//
// Requirement: FR-PRO-001, FR-PRO-032, FR-AGT-003
//
// Guard: architecture.md 9.4 and AGENTS.md "supply chains are data"; a table
// that branched on an identifier would not see an entity it never heard of.
func TestTables_ConfigurationOnlyAddition(t *testing.T) {
	t.Parallel()
	cfg, err := config.LoadFS(os.DirFS("testdata/chains"), "second/second.toml", "second/warm-second.toml")
	if err != nil {
		t.Fatalf("LoadFS error = %v, want nil", err)
	}
	baseCommodities, baseRecipes, baseRoles := len(cfg.Commodities), len(cfg.Recipes), len(cfg.Roles)
	cfg.Commodities = append(cfg.Commodities, economysim.Commodity{ID: "zz_new_good", ReferencePrice: 100, Raw: true})
	cfg.Recipes = append(cfg.Recipes, economysim.Recipe{
		ID: "zz_make_new_good", Duration: 1, TraineeYield: 10000,
		Outputs: []economysim.Output{{Commodity: "zz_new_good", Quantity: 1, CostShare: 10000}},
	})
	role := cfg.Roles[0]
	role.ID = "zz_new_role"
	role.Recipes = []string{"zz_make_new_good"}
	role.Trades = nil
	cfg.Roles = append(cfg.Roles, role)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate error = %v, want nil", err)
	}
	c := economysim.Canonical(cfg)
	if len(c.Commodities) != baseCommodities+1 || len(c.Recipes) != baseRecipes+1 || len(c.Roles) != baseRoles+1 {
		t.Fatalf("counts = %d, %d, %d, want one more of each", len(c.Commodities), len(c.Recipes), len(c.Roles))
	}
	rt, ot := economysim.NewRecipeTable(&c), economysim.NewRoleTable(&c)
	var cid economysim.CommodityID
	for i, x := range c.Commodities {
		if x.ID == "zz_new_good" {
			cid = economysim.CommodityID(i)
		}
	}
	var rid economysim.RecipeID
	for i, x := range c.Recipes {
		if x.ID == "zz_make_new_good" {
			rid = economysim.RecipeID(i)
		}
	}
	if got := economysim.RecipeMakers(&rt, cid); !slices.Equal(got, []economysim.RecipeID{rid}) {
		t.Errorf("makers(new good) = %v, want [%d]", got, rid)
	}
	for i, x := range c.Roles {
		if x.ID != "zz_new_role" {
			continue
		}
		row := economysim.RoleTableRow(&ot, economysim.RoleID(i))
		if !slices.Equal(row.Recipes, []int{int(rid)}) {
			t.Errorf("role recipes = %v, want [%d]", row.Recipes, rid)
		}
		if !slices.Contains(row.Tradable, int(cid)) {
			t.Errorf("tradable set %v lacks the new good %d", row.Tradable, cid)
		}
	}
	if !economysim.RoleTradableSomewhere(&ot, cid) {
		t.Error("tradableSomewhere(new good) = false, want true")
	}
}
