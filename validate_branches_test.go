package economysim_test

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// withProduct adds a commodity of the given reference price to c.
func withProduct(c *economysim.Config, id string, price economysim.Price, raw bool) {
	cm := economysim.DefaultCommodity()
	cm.ID, cm.ReferencePrice, cm.Raw = id, price, raw
	c.Commodities = append(c.Commodities, cm)
}

// hasPath reports whether some error of the rule has exactly the path.
func hasPath(vs []*economysim.ValidationError, rule error, path []string) *economysim.ValidationError {
	for _, v := range vs {
		if errors.Is(v, rule) && reflect.DeepEqual(v.Path, path) {
			return v
		}
	}
	return nil
}

// TestValidate_DerivedProductsBeyondInt64 verifies the two derived products
// that overflow int64 itself, not only a bound: the cost of ten inputs at the
// largest price and quantity (10 * 10^18), and the value one execution of a
// recipe sells at the largest price and the largest yield (10^9 units at ten
// times the yield is 10^10 units, at 10^9 each, 10^19). Each is reported as
// ErrOverflow with the value "overflow", not as a wrapped number.
//
// Requirement: FR-CFG-010, FR-CFG-036, FR-PRO-039, FR-WEL-010
func TestValidate_DerivedProductsBeyondInt64(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		path   []string
	}{
		{
			name: "cost of ten largest inputs",
			mutate: func(c *economysim.Config) {
				r := economysim.DefaultRecipe()
				r.ID = "big"
				for i := range 10 {
					id := "raw" + strconv.Itoa(i)
					withProduct(c, id, economysim.MaxPrice, true)
					r.Inputs = append(r.Inputs, economysim.Item{Commodity: id, Quantity: economysim.MaxQuantity})
				}
				withProduct(c, "made", 50, false)
				r.Outputs = []economysim.Output{{Commodity: "made", Quantity: 1, CostShare: 10_000}}
				c.Recipes = append(c.Recipes, r)
				c.Roles[0].Recipes = []string{"big"}
			},
			path: []string{"recipes", "big", "inputs"},
		},
		{
			name: "income of one execution at the largest yield",
			mutate: func(c *economysim.Config) {
				withProduct(c, "gem", economysim.MaxPrice, false)
				r := economysim.DefaultRecipe()
				r.ID = "mine"
				r.Outputs = []economysim.Output{{Commodity: "gem", Quantity: economysim.MaxQuantity, CostShare: 10_000}}
				c.Recipes = append(c.Recipes, r)
				c.Roles[0].Recipes = []string{"mine"}
			},
			path: []string{"roles", "farmer", "recipes"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := minimalConfig()
			tt.mutate(&c)
			vs, _ := violations(c.Validate())
			v := hasPath(vs, economysim.ErrOverflow, tt.path)
			if v == nil {
				t.Fatalf("Validate(%s) has no ErrOverflow at %q, want one; errors: %v", tt.name, tt.path, pathsOf(vs))
			}
			if v.Value != "overflow" {
				t.Errorf("Validate(%s) overflow value at %q = %q, want %q", tt.name, tt.path, v.Value, "overflow")
			}
		})
	}
}

// TestValidate_CostSharesWhoseSumOverflowsTheWhole verifies that output cost
// shares whose sum overflows int64 are reported as ErrCostShares even when the
// shares after the overflow add up to exactly 10 000. The shares are MaxInt64,
// MaxInt64 and 10 000: the checked sum of the first two fails and restarts
// from 0, and 0 + 10 000 is the whole.
//
// Requirement: FR-PRO-004
//
// Guard: the overflow is remembered in its own flag. Without it the running
// sum after the overflow looks like a valid 10 000 and the rule passes by
// accident (found by mutation testing of the overflow branch).
func TestValidate_CostSharesWhoseSumOverflowsTheWhole(t *testing.T) {
	t.Parallel()
	c := minimalConfig()
	withProduct(&c, "a", 50, false)
	withProduct(&c, "b", 50, false)
	r := economysim.DefaultRecipe()
	r.ID = "wrap"
	r.Outputs = []economysim.Output{
		{Commodity: "a", Quantity: 1, CostShare: 1<<63 - 1},
		{Commodity: "b", Quantity: 1, CostShare: 1<<63 - 1},
		{Commodity: "grain", Quantity: 1, CostShare: 10_000},
	}
	c.Recipes = append(c.Recipes, r)
	c.Roles[0].Recipes = []string{"wrap"}
	vs, _ := violations(c.Validate())
	if hasPath(vs, economysim.ErrCostShares, []string{"recipes", "wrap", "outputs"}) == nil {
		t.Errorf("Validate(shares overflowing then adding to 10 000) has no ErrCostShares at recipes.wrap.outputs; errors: %v", pathsOf(vs))
	}
}

// TestValidate_UnproducibleChainIsCutToTheNestingDepth verifies that the
// witness chain of a cycle with no source is cut to MaxNestingDepth elements,
// so an error path cannot grow with the cycle. Ten commodities that each need
// the next make a chain of 2*10+1 elements before the cut.
//
// Guard: the chain is part of an error path that consumers print and that the
// loader bounds elsewhere (CFG 6.3); an uncut chain would let a large cycle
// build an error as long as the configuration.
func TestValidate_UnproducibleChainIsCutToTheNestingDepth(t *testing.T) {
	t.Parallel()
	const n = 10
	c := minimalConfig()
	for i := range n {
		id := "c" + strconv.Itoa(i)
		withProduct(&c, id, 50, false)
		r := economysim.DefaultRecipe()
		r.ID = "make_" + id
		r.Inputs = []economysim.Item{{Commodity: "c" + strconv.Itoa((i+1)%n), Quantity: 1}}
		r.Outputs = []economysim.Output{{Commodity: id, Quantity: 1, CostShare: 10_000}}
		c.Recipes = append(c.Recipes, r)
	}
	vs, _ := violations(c.Validate())
	var found int
	for _, v := range vs {
		if !errors.Is(v, economysim.ErrUnproducible) {
			continue
		}
		found++
		// "commodities" followed by the chain, cut to MaxNestingDepth - 1
		// elements after the first commodity: the path is MaxNestingDepth long.
		if len(v.Path) != economysim.MaxNestingDepth {
			t.Errorf("ErrUnproducible path %q has %d segments, want %d", v.Path, len(v.Path), economysim.MaxNestingDepth)
		}
	}
	if found != n {
		t.Errorf("Validate(cycle of %d) has %d ErrUnproducible errors, want %d", n, found, n)
	}
}

// TestValidate_StartMoneyBeyondItsBoundIsReportedOnce verifies that a role
// whose StartMoney is above its bound is reported on the role only, and that
// the total money at the start (households times StartMoney, which would
// overflow int64 here) skips the group instead of reporting a second error.
//
// Guard: a field outside its bound is reported by its own rule; the derived
// sums skip it (spec-config.md 4.2) so one mistake gives one error.
func TestValidate_StartMoneyBeyondItsBoundIsReportedOnce(t *testing.T) {
	t.Parallel()
	c := minimalConfig()
	c.Roles[0].StartMoney = economysim.MaxMoney
	c.Groups[0].Households = economysim.MaxHouseholds
	vs, other := violations(c.Validate())
	if len(other) != 0 {
		t.Fatalf("Validate returned errors that are not a *ValidationError: %v", other)
	}
	want := [][]string{{"roles", "farmer", "start_money"}}
	if got := pathsOf(vs); !reflect.DeepEqual(got, want) {
		t.Errorf("Validate(start money above its bound) paths = %q, want %q", got, want)
	}
}

// TestWarnings_YieldRowWithNoEntryIsTheDefaultOfTheRecipe verifies that a
// yield profile with no all-season factor and no season rows means the
// recipe's default (spec-production.md 6.2): a recipe that is not Sited runs
// at 10 000, so nothing is said; a Sited recipe does not run, so it is
// reported as unrunnable.
//
// Requirement: FR-PRO-020, FR-PRO-040
func TestWarnings_YieldRowWithNoEntryIsTheDefaultOfTheRecipe(t *testing.T) {
	t.Parallel()
	for _, sited := range []bool{false, true} {
		t.Run("sited="+strconv.FormatBool(sited), func(t *testing.T) {
			t.Parallel()
			c := minimalConfig()
			withProduct(&c, "ore", 50, false)
			r := economysim.DefaultRecipe()
			r.ID, r.Sited = "dig", sited
			r.Outputs = []economysim.Output{{Commodity: "ore", Quantity: 1, CostShare: 10_000}}
			c.Recipes = append(c.Recipes, r)
			c.Roles[0].Recipes = []string{"dig"}
			c.Yields = []economysim.YieldProfile{{Region: "home", Recipe: "dig"}}
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate(empty yield row) = %v, want nil", err)
			}
			got := slices.ContainsFunc(c.Warnings(), func(w economysim.Warning) bool {
				return w.Rule == "unrunnable_recipe" && reflect.DeepEqual(w.Path, []string{"recipes", "dig"})
			})
			if got != sited {
				t.Errorf("Warnings(empty yield row, sited=%v) has unrunnable_recipe = %v, want %v", sited, got, sited)
			}
		})
	}
}

// TestWarnings_UpkeepSourcedFromARegionIsCounted verifies that a good that
// only a region's extra upkeep consumes, and nothing makes, raises
// upkeep_unsourced like one in the world or a role upkeep.
//
// Requirement: FR-AGT-060
//
// Guard: three lists feed the upkeep; the region's list was the one no test
// reached, so dropping it would silence the warning for a region-only good.
func TestWarnings_UpkeepSourcedFromARegionIsCounted(t *testing.T) {
	t.Parallel()
	c := minimalConfig()
	withProduct(&c, "ore_y", 100, false)
	c.Regions[0].ExtraUpkeep = []economysim.Item{{Commodity: "ore_y", Quantity: 1}}
	got := slices.ContainsFunc(c.Warnings(), func(w economysim.Warning) bool {
		return w.Rule == "upkeep_unsourced" && reflect.DeepEqual(w.Path, []string{"commodities", "ore_y"})
	})
	if !got {
		t.Errorf("Warnings(region extra upkeep of an unmade good) = %v, want upkeep_unsourced at commodities.ore_y", c.Warnings())
	}
}
