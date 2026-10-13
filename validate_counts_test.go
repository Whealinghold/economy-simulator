package economysim_test

import (
	"errors"
	"slices"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestValidate_EveryCountBound verifies, for every count bound Validate checks
// (spec-config.md 4.1, 7.1.6), that a collection one over its bound is
// ErrTooMany at exactly its path with its count and bound, that it is the only
// error (the later rules did not run, though the configuration is also wrong
// elsewhere), and that a collection at its bound is not ErrTooMany.
//
// Requirement: FR-CFG-017, FR-CFG-101
func TestValidate_EveryCountBound(t *testing.T) {
	// Not parallel: a case holds up to ten million zero entries.
	const bigBound = economysim.MaxYields
	if bigBound != economysim.MaxRegions*economysim.MaxRecipes {
		t.Fatalf("MaxYields = %d, want MaxRegions*MaxRecipes", bigBound)
	}
	tests := []struct {
		name  string
		bound int
		path  []string
		// set makes the collection n entries long; the entries are zero values,
		// which no rule beyond the count phase may be asked about.
		set func(c *economysim.Config, n int)
		// big skips the case at the bound: Validate would run its other rules
		// over every entry, and the bound itself is one comparison shared by all.
		big bool
	}{
		{"commodities", economysim.MaxCommodities, []string{"commodities"},
			func(c *economysim.Config, n int) { c.Commodities = make([]economysim.Commodity, n) }, false},
		{"recipes", economysim.MaxRecipes, []string{"recipes"},
			func(c *economysim.Config, n int) { c.Recipes = make([]economysim.Recipe, n) }, false},
		{"roles", economysim.MaxRoles, []string{"roles"},
			func(c *economysim.Config, n int) { c.Roles = make([]economysim.Role, n) }, false},
		{"regions", economysim.MaxRegions, []string{"regions"},
			func(c *economysim.Config, n int) { c.Regions = make([]economysim.Region, n) }, false},
		{"groups", economysim.MaxGroups, []string{"groups"},
			func(c *economysim.Config, n int) { c.Groups = make([]economysim.Group, n) }, false},
		{"yields", economysim.MaxYields, []string{"yields"},
			func(c *economysim.Config, n int) { c.Yields = make([]economysim.YieldProfile, n) }, true},
		{"months", economysim.MaxMonths, []string{"calendar", "months"},
			func(c *economysim.Config, n int) { c.Calendar.Months = make([]economysim.Month, n) }, false},
		{"seasons", economysim.MaxSeasons, []string{"calendar", "seasons"},
			func(c *economysim.Config, n int) { c.Calendar.Seasons = make([]economysim.Season, n) }, false},
		{"world upkeep", economysim.MaxUpkeepItems, []string{"world", "upkeep"},
			func(c *economysim.Config, n int) { c.World.Upkeep = make([]economysim.Item, n) }, false},
		{"warm start prices", economysim.MaxSeedPrices, []string{"warm_start", "prices"},
			func(c *economysim.Config, n int) { c.WarmStart.Prices = make([]economysim.SeedPrice, n) }, true},
		{"warm start wages", economysim.MaxRegions, []string{"warm_start", "wages"},
			func(c *economysim.Config, n int) { c.WarmStart.Wages = make([]economysim.SeedWage, n) }, false},
		{"warm start groups", economysim.MaxGroups, []string{"warm_start", "groups"},
			func(c *economysim.Config, n int) { c.WarmStart.Groups = make([]economysim.WarmGroup, n) }, false},
		{"warm start pools", economysim.MaxRegions, []string{"warm_start", "pools"},
			func(c *economysim.Config, n int) { c.WarmStart.Pools = make([]economysim.WarmPool, n) }, false},
		{"recipe inputs", economysim.MaxRecipeItems, []string{"recipes", "refine", "inputs"},
			func(c *economysim.Config, n int) { recipeOf(c, "refine").Inputs = make([]economysim.Item, n) }, false},
		{"recipe outputs", economysim.MaxRecipeItems, []string{"recipes", "refine", "outputs"},
			func(c *economysim.Config, n int) { recipeOf(c, "refine").Outputs = make([]economysim.Output, n) }, false},
		{"recipe requires", economysim.MaxRequires, []string{"recipes", "refine", "requires"},
			func(c *economysim.Config, n int) { recipeOf(c, "refine").Requires = make([]economysim.Tool, n) }, false},
		{"role recipes", economysim.MaxRecipes, []string{"roles", "cook", "recipes"},
			func(c *economysim.Config, n int) { roleOf(c, "cook").Recipes = make([]string, n) }, false},
		{"role upkeep", economysim.MaxUpkeepItems, []string{"roles", "cook", "upkeep"},
			func(c *economysim.Config, n int) { roleOf(c, "cook").Upkeep = make([]economysim.Item, n) }, false},
		{"role trades", economysim.MaxCommodities, []string{"roles", "cook", "trades"},
			func(c *economysim.Config, n int) { roleOf(c, "cook").Trades = make([]string, n) }, false},
		{"role start stock", economysim.MaxCommodities, []string{"roles", "cook", "start_stock"},
			func(c *economysim.Config, n int) { roleOf(c, "cook").StartStock = make([]economysim.Item, n) }, false},
		{"region extra upkeep", economysim.MaxUpkeepItems, []string{"regions", "north", "extra_upkeep"},
			func(c *economysim.Config, n int) { c.Regions[0].ExtraUpkeep = make([]economysim.Item, n) }, false},
		{"yield seasons", economysim.MaxSeasons, []string{"yields", "north", "cut_fuel", "seasons"},
			func(c *economysim.Config, n int) { c.Yields[0].Seasons = make([]economysim.SeasonYield, n) }, false},
		{"warm start stock", economysim.MaxCommodities, []string{"warm_start", "groups", "north", "cook", "stock"},
			func(c *economysim.Config, n int) {
				c.WarmStart.Groups = []economysim.WarmGroup{{Region: "north", Role: "cook", Stock: make([]economysim.Item, n)}}
			}, false},
		{"warm start jobs", economysim.MaxRecipes, []string{"warm_start", "groups", "north", "cook", "jobs"},
			func(c *economysim.Config, n int) {
				c.WarmStart.Groups = []economysim.WarmGroup{{Region: "north", Role: "cook", Jobs: make([]economysim.JobSpec, n)}}
			}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name+" over", func(t *testing.T) {
			c := referenceConfig()
			// Other mistakes that the later phases would report.
			c.Roles[len(c.Roles)-1].ID = "BAD"
			c.Groups = append(c.Groups, economysim.Group{Region: "nowhere", Role: "ghost"})
			tt.set(&c, tt.bound+1)
			err := c.Validate()
			if !errors.Is(err, economysim.ErrTooMany) {
				t.Fatalf("errors.Is(Validate(%s one over), ErrTooMany) = false, want true; error: %v", tt.name, err)
			}
			vs, other := violations(err)
			if len(vs) != 1 || len(other) != 0 {
				t.Fatalf("Validate(%s one over) = %d errors and %d others, want exactly one ErrTooMany: %v", tt.name, len(vs), len(other), err)
			}
			if !slices.Equal(vs[0].Path, tt.path) || vs[0].Value != strconv.Itoa(tt.bound+1) || vs[0].Bound != strconv.Itoa(tt.bound) {
				t.Errorf("Validate(%s one over) = path %q value %q bound %q, want path %q value %d bound %d",
					tt.name, vs[0].Path, vs[0].Value, vs[0].Bound, tt.path, tt.bound+1, tt.bound)
			}
		})
		if tt.big {
			continue
		}
		t.Run(tt.name+" at the bound", func(t *testing.T) {
			c := referenceConfig()
			tt.set(&c, tt.bound)
			if err := c.Validate(); errors.Is(err, economysim.ErrTooMany) {
				t.Errorf("Validate(%s at its bound %d) is ErrTooMany, want the bound itself allowed: %v", tt.name, tt.bound, err)
			}
		})
	}
}
