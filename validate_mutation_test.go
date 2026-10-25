package economysim_test

import (
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestValidate_IDLengthAndAlphabetBounds verifies the edges of the ID rule of
// spec-config.md 3.3: an ID of exactly MaxIDLength bytes is valid, and after
// the first byte only [a-z0-9_] is allowed, so the bytes just above '_' in
// ASCII ('`', '{', '~') and a non-ASCII byte are refused.
//
// Requirement: FR-CFG-006
func TestValidate_IDLengthAndAlphabetBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		id      string
		invalid bool
	}{
		{"exactly MaxIDLength bytes", strings.Repeat("a", economysim.MaxIDLength), false},
		{"backtick after the first byte", "vis`tor", true},
		{"left brace after the first byte", "vis{tor", true},
		{"tilde after the first byte", "vis~tor", true},
		{"non-ASCII after the first byte", "visétor", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			// The visitor role has no group, so nothing refers to it.
			roleOf(&c, "visitor").ID = tt.id
			err := c.Validate()
			if got := errors.Is(err, economysim.ErrInvalidID); got != tt.invalid {
				t.Errorf("errors.Is(Validate(%s), ErrInvalidID) = %t, want %t; error: %v", tt.name, got, tt.invalid, err)
			}
			if !tt.invalid && err != nil {
				t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
			}
		})
	}
}

// TestValidate_CountOverStopsNestedWalk verifies that a top-level count over
// its bound is the only error even when a nested list is also over its bound:
// the nested lists are walked only if every top-level count holds.
//
// Requirement: FR-CFG-101
func TestValidate_CountOverStopsNestedWalk(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	c.Commodities = make([]economysim.Commodity, economysim.MaxCommodities+1)
	recipeOf(&c, "refine").Inputs = make([]economysim.Item, economysim.MaxRecipeItems+1)
	vs, other := violations(c.Validate())
	if len(vs) != 1 || len(other) != 0 || len(vs[0].Path) != 1 || vs[0].Path[0] != "commodities" {
		t.Errorf("Validate(commodities and recipe inputs over) = %v (others %v), want exactly one ErrTooMany at commodities", vs, other)
	}
}

// TestValidate_ProducibilityFixpointNeedsEveryNeed verifies the worklist of
// spec-config.md 7.5: mid_c is made from a producible good and from gap_d,
// which nothing makes, so mid_c (consumed by the recipe that makes end_e) is
// ErrUnproducible. A recipe is satisfied only when its last need is, and a
// commodity that is marked twice (a raw good that a source recipe also makes,
// or a good that two source recipes make) must not count twice.
//
// Requirement: FR-CFG-015
func TestValidate_ProducibilityFixpointNeedsEveryNeed(t *testing.T) {
	t.Parallel()
	add := func(c *economysim.Config, raw bool, extraSources int) {
		for _, id := range []string{"src_a", "gap_d", "mid_c", "end_e"} {
			cm := economysim.DefaultCommodity()
			cm.ID, cm.ReferencePrice = id, 100
			cm.Raw = raw && id == "src_a"
			c.Commodities = append(c.Commodities, cm)
		}
		out := func(id string) []economysim.Output {
			return []economysim.Output{{Commodity: id, Quantity: 1, CostShare: 10_000}}
		}
		mid, end := economysim.DefaultRecipe(), economysim.DefaultRecipe()
		mid.ID, end.ID = "make_mid", "make_end"
		mid.Inputs = []economysim.Item{{Commodity: "src_a", Quantity: 1}, {Commodity: "gap_d", Quantity: 1}}
		mid.Outputs = out("mid_c")
		end.Inputs = []economysim.Item{{Commodity: "mid_c", Quantity: 1}}
		end.Outputs = out("end_e")
		c.Recipes = append(c.Recipes, mid, end)
		for i := 0; i < extraSources; i++ {
			s := economysim.DefaultRecipe()
			s.ID = "source_" + string(rune('a'+i))
			s.Outputs = out("src_a")
			c.Recipes = append(c.Recipes, s)
		}
	}
	tests := []struct {
		name         string
		raw          bool
		extraSources int
	}{
		{"one raw need and one unmade need", true, 0},
		{"raw good that a source recipe also makes", true, 1},
		{"good made by two source recipes", false, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			add(&c, tt.raw, tt.extraSources)
			vs, _ := violations(c.Validate())
			found := false
			for _, v := range vs {
				if errors.Is(v, economysim.ErrUnproducible) && len(v.Path) > 1 && v.Path[1] == "mid_c" {
					found = true
				}
			}
			if !found {
				t.Errorf("Validate(%s) = %v, want ErrUnproducible at commodities/mid_c", tt.name, vs)
			}
		})
	}
}

// TestValidate_UnproducibleWitnessIsLowestUnproducibleNeed verifies the
// witness of spec-config.md 7.5: of the needs of the recipe that makes the
// commodity, the path names the unproducible need with the lowest ID, not a
// producible one and not the first or last listed. mid_c is made from aa_ok
// (raw, producible), bb_gap and zz_gap (made by nothing); the path ends at
// bb_gap whichever order the inputs are written in.
//
// Requirement: FR-CFG-015
func TestValidate_UnproducibleWitnessIsLowestUnproducibleNeed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		needs []string
	}{
		{"lowest unproducible first", []string{"bb_gap", "aa_ok", "zz_gap"}},
		{"lowest unproducible last", []string{"zz_gap", "aa_ok", "bb_gap"}},
		{"producible need last", []string{"zz_gap", "bb_gap", "aa_ok"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			for _, id := range []string{"aa_ok", "bb_gap", "zz_gap", "mid_c", "end_e"} {
				cm := economysim.DefaultCommodity()
				cm.ID, cm.ReferencePrice, cm.Raw = id, 100, id == "aa_ok"
				c.Commodities = append(c.Commodities, cm)
			}
			mid, end := economysim.DefaultRecipe(), economysim.DefaultRecipe()
			mid.ID, end.ID = "make_mid", "make_end"
			for _, n := range tt.needs {
				mid.Inputs = append(mid.Inputs, economysim.Item{Commodity: n, Quantity: 1})
			}
			mid.Outputs = []economysim.Output{{Commodity: "mid_c", Quantity: 1, CostShare: 10_000}}
			end.Inputs = []economysim.Item{{Commodity: "mid_c", Quantity: 1}}
			end.Outputs = []economysim.Output{{Commodity: "end_e", Quantity: 1, CostShare: 10_000}}
			c.Recipes = append(c.Recipes, mid, end)
			vs, _ := violations(c.Validate())
			want := []string{"commodities", "mid_c", "make_mid", "bb_gap"}
			if v := hasPath(vs, economysim.ErrUnproducible, want); v == nil {
				t.Errorf("Validate(%s) = %v, want ErrUnproducible at %q", tt.name, vs, want)
			}
		})
	}
}

// heavyCook makes the reference cook group the heaviest the bounds allow: 10^7
// households at scale 10^7 is one agent raised to the floor K = 10, weight
// 10^6 = MaxWeight, so the targets of a role read the extra upkeep of its
// regions at the largest weight (spec-scale.md 3.2, spec-agents.md 7.3).
func heavyCook(c *economysim.Config) {
	for i := range c.Groups {
		if c.Groups[i].Role == "cook" {
			c.Groups[i].Households, c.Groups[i].Scale = 10_000_000, 10_000_000
		}
	}
}

// extraStaple is a copy of the region id that adds the most staple a household
// may be given (MaxQuantity / MaxWeight = 1 000) to its upkeep.
func extraStaple(id string) economysim.Region {
	r := economysim.DefaultRegion()
	r.ID = id
	r.ExtraUpkeep = []economysim.Item{{Commodity: "staple", Quantity: 1_000}}
	return r
}

// TestValidate_DuplicateIDKeepsTheFirstEntity verifies that a rule that reads
// an entity through its ID reads the first of two entities with that ID
// (ErrDuplicateID reports the second): each case adds a second entity that, if
// it were the one read, would break a rule the reference configuration meets.
//
// Requirement: FR-CFG-006
func TestValidate_DuplicateIDKeepsTheFirstEntity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		path   []string
	}{
		{"second commodity not durable", func(c *economysim.Config) {
			// tool_a (Durability 10) is the grower's required tool.
			d := economysim.DefaultCommodity()
			d.ID, d.ReferencePrice, d.Durability = "tool_a", 13915, 0
			c.Commodities = append(c.Commodities, d)
		}, []string{"recipes", "grow_staple", "requires", "tool_a", "commodity"}},
		{"second recipe of the longest duration", func(c *economysim.Config) {
			// The first cut_fuel lasts 1 tick; the second would make the
			// forester's window 2 * 100 000.
			d := *recipeOf(c, "cut_fuel")
			d.Duration = economysim.MaxRecipeTicks
			c.Recipes = append(c.Recipes, d)
		}, []string{"roles", "forester", "recipes"}},
		{"second role with a traineeship", func(c *economysim.Config) {
			// The first cook has no TrainingTicks, so scale 2 is allowed.
			d := *roleOf(c, "cook")
			d.TrainingTicks = 5
			c.Roles = append(c.Roles, d)
			for i := range c.Groups {
				if c.Groups[i].Role == "cook" {
					c.Groups[i].Scale = 2
				}
			}
		}, []string{"groups", "north", "cook", "scale"}},
		{"second region with extra upkeep", func(c *economysim.Config) {
			// North has no extra upkeep: the cook's targets stay at 10^6 * 10.
			heavyCook(c)
			c.Regions = append(c.Regions, extraStaple("north"))
		}, []string{"roles", "cook", "targets", "staple"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			vs, _ := violations(c.Validate())
			for _, v := range vs {
				if len(v.Path) == len(tt.path) && strings.Join(v.Path, "/") == strings.Join(tt.path, "/") &&
					!errors.Is(v, economysim.ErrDuplicateID) {
					t.Errorf("Validate(%s) has %v at %q, want none: the first entity of the ID is the one read", tt.name, v.Rule, tt.path)
				}
			}
		})
	}
}

// TestValidate_InvalidGroupAddsNoRegionToTheRole verifies that a group whose
// household count is outside its bound has no shape, so its region does not
// add its extra upkeep to the role's targets, while a valid group in the same
// region does (the control case).
//
// Requirement: FR-SCL-011
func TestValidate_InvalidGroupAddsNoRegionToTheRole(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		households int64
		want       bool
	}{
		{"valid group in the region with extra upkeep", 1, true},
		{"group of 0 households", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			heavyCook(&c)
			c.Regions = append(c.Regions, extraStaple("east"))
			c.Groups = append(c.Groups, economysim.Group{Region: "east", Role: "cook", Households: tt.households})
			vs, _ := violations(c.Validate())
			path := []string{"roles", "cook", "targets", "staple"}
			found := false
			for _, v := range vs {
				found = found || strings.Join(v.Path, "/") == strings.Join(path, "/")
			}
			if found != tt.want {
				t.Errorf("Validate(%s): violation at %q = %t, want %t; paths: %v", tt.name, path, found, tt.want, pathsOf(vs))
			}
		})
	}
}

// TestValidate_GroupShapeBounds verifies the edges of the inputs of the shape
// of a group (spec-scale.md 3.2): the shape exists, so the traineeship rule
// reads its scale, for exactly 1 and MaxHouseholds households, a scale of
// MaxHouseholds and a floor of 1 and MaxHouseholds, and it does not for one
// past the upper bound.
//
// Requirement: FR-CFG-054, FR-SCL-001, FR-SCL-020
func TestValidate_GroupShapeBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		households int64
		scale      int64
		minAgents  int64
		want       bool
		world      int64 // the world's Scale when not 0
	}{
		{"no household", 0, 2, 10, false, 0},
		{"one household", 1, 2, 10, true, 0},
		{"MaxHouseholds households", economysim.MaxHouseholds, 2, 10, true, 0},
		{"one household too many", economysim.MaxHouseholds + 1, 2, 10, false, 0},
		{"scale of MaxHouseholds", 1, economysim.MaxHouseholds, 10, true, 0},
		{"scale one too large", 136, economysim.MaxHouseholds + 1, 10, false, 0},
		{"floor of 1", 136, 2, 1, true, 0},
		{"floor of MaxHouseholds", 136, 2, economysim.MaxHouseholds, true, 0},
		{"floor one too large", 136, 2, economysim.MaxHouseholds + 1, false, 0},
		// A negative group scale is not "none given": it does not fall back to
		// the world's 2, so the group has no shape.
		{"negative group scale with a world scale of 2", 136, -1, 10, false, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			roleOf(&c, "cook").TrainingTicks = 5
			c.World.MinAgents = tt.minAgents
			if tt.world != 0 {
				c.World.Scale = tt.world
			}
			for i := range c.Groups {
				if c.Groups[i].Role == "cook" {
					c.Groups[i].Households, c.Groups[i].Scale = tt.households, tt.scale
				}
			}
			vs, _ := violations(c.Validate())
			var got bool
			for _, v := range vs {
				if errors.Is(v, economysim.ErrOutOfBounds) && v.Bound == "traineeship_needs_weight_one" {
					got = true
				}
			}
			if got != tt.want {
				t.Errorf("Validate(%s): traineeship_needs_weight_one = %t, want %t; paths: %v", tt.name, got, tt.want, pathsOf(vs))
			}
		})
	}
}

// TestValidate_GroupWeightIsCeilOfHouseholdsOverAgents verifies the weight the
// weight bound reads: 6 000 000 households at scale 3 000 000 and a floor of 1
// are n = ceil(6e6 / 3e6) = 2 agents, so the largest weight is
// ceil(6e6 / 2) = 3 000 000, above MaxWeight, and the error carries it.
//
// Requirement: FR-SCL-002
func TestValidate_GroupWeightIsCeilOfHouseholdsOverAgents(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	c.World.MinAgents = 1
	for i := range c.Groups {
		if c.Groups[i].Role == "cook" {
			c.Groups[i].Households, c.Groups[i].Scale = 6_000_000, 3_000_000
		}
	}
	vs, _ := violations(c.Validate())
	v := hasPath(vs, economysim.ErrOutOfBounds, []string{"groups", "north", "cook", "scale"})
	if v == nil || v.Value != "3000000" || v.Bound != "1000000" {
		t.Errorf("Validate(6e6 households at scale 3e6) = %v, want ErrOutOfBounds at groups/north/cook/scale, value 3000000, bound 1000000", pathsOf(vs))
	}
}

// TestValidate_BeliefFloorEdges verifies the edges of BeliefStep for the
// resolution floor ceil(10 000 / BeliefStep): at 1 the floor is 10 000, at 2 500
// it is 4, and above 2 500 (itself an error) no floor is derived.
//
// Requirement: FR-BEL-029
func TestValidate_BeliefFloorEdges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		step  int64
		price economysim.Price
		bound string // "" when no floor error is wanted
	}{
		{"BeliefStep of 1: floor 10 000", 1, 179, "10000"},
		{"BeliefStep of 2 500: floor 4", 2_500, 3, "4"},
		{"BeliefStep above 2 500: no floor", 2_501, 3, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			c.World.BeliefStep = tt.step
			commodityOf(&c, "fuel").ReferencePrice = tt.price
			vs, _ := violations(c.Validate())
			v := hasPath(vs, economysim.ErrOutOfBounds, []string{"commodities", "fuel", "reference_price"})
			switch {
			case tt.bound == "" && v != nil:
				t.Errorf("Validate(%s) has ErrOutOfBounds at commodities/fuel/reference_price (bound %s), want none", tt.name, v.Bound)
			case tt.bound != "" && (v == nil || v.Bound != tt.bound):
				t.Errorf("Validate(%s) = %v, want ErrOutOfBounds at commodities/fuel/reference_price with bound %s", tt.name, pathsOf(vs), tt.bound)
			}
		})
	}
}

// TestValidate_EffectiveWindowInputs verifies when the effective welfare window
// of a role is computed (spec-config.md FR-CFG-052): only when WelfareWindow is
// in [1, 1 000], WindowCycles in [1, 100] and every Duration on the list in
// [1, MaxRecipeTicks]; the window is then max(WelfareWindow, WindowCycles * the
// longest Duration) and a window above MaxWelfareWindow is an error. The
// grower's list is grow_staple, save_seed and gather_seed.
//
// Requirement: FR-CFG-052, FR-WEL-009
func TestValidate_EffectiveWindowInputs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		grow, save   int64
		window, cyc  int64
		wantTooLarge bool
	}{
		// 2 * 2 000 = 4 000 > 3 650.
		{"control: window of 4 000", 2_000, 1, 30, 2, true},
		{"longest Duration of exactly MaxRecipeTicks", economysim.MaxRecipeTicks, 1, 30, 2, true},
		// The Duration beyond its bound is reported by its own rule; the other
		// recipe's 2 000 must not make a window.
		{"a Duration above MaxRecipeTicks", economysim.MaxRecipeTicks + 1, 2_000, 30, 2, false},
		{"WelfareWindow of 0", 2_000, 1, 0, 2, false},
		{"WelfareWindow of 100 000", 2_000, 1, 100_000, 2, false},
		{"WelfareWindow of 1", 2_000, 1, 1, 2, true},
		// 1 000 * 10 = 10 000 would be above the bound, but 1 000 cycles is not a value.
		{"WindowCycles of 1 000", 10, 1, 30, 1_000, false},
		// 1 * 4 000 = 4 000 > 3 650.
		{"WindowCycles of 1", 4_000, 1, 30, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			recipeOf(&c, "grow_staple").Duration = tt.grow
			recipeOf(&c, "save_seed").Duration = tt.save
			c.World.WelfareWindow, c.World.WindowCycles = tt.window, tt.cyc
			vs, _ := violations(c.Validate())
			got := hasPath(vs, economysim.ErrOutOfBounds, []string{"roles", "grower", "recipes"}) != nil
			if got != tt.wantTooLarge {
				t.Errorf("Validate(%s): window error at roles/grower/recipes = %t, want %t; paths: %v", tt.name, got, tt.wantTooLarge, pathsOf(vs))
			}
		})
	}
}

// TestValidate_WindowProductNotCheckedWithAnInvalidWindowField verifies that
// the overflow of window times income is checked only when the window is
// defined: with WelfareWindow 0 the window has no value, and the income that
// overflows int64 is not reported at the role's recipes (the world field is
// reported at its own path).
//
// Requirement: FR-WEL-010, FR-CFG-052
func TestValidate_WindowProductNotCheckedWithAnInvalidWindowField(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		window int64
		want   bool
	}{
		{"defined window: the income overflows", 30, true},
		{"WelfareWindow of 0", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := minimalConfig()
			withProduct(&c, "gem", economysim.MaxPrice, false)
			r := economysim.DefaultRecipe()
			r.ID = "mine"
			r.Outputs = []economysim.Output{{Commodity: "gem", Quantity: economysim.MaxQuantity, CostShare: 10_000}}
			c.Recipes = append(c.Recipes, r)
			c.Roles[0].Recipes = []string{"mine"}
			c.World.WelfareWindow = tt.window
			vs, _ := violations(c.Validate())
			got := hasPath(vs, economysim.ErrOverflow, []string{"roles", "farmer", "recipes"}) != nil
			if got != tt.want {
				t.Errorf("Validate(%s): ErrOverflow at roles/farmer/recipes = %t, want %t; paths: %v", tt.name, got, tt.want, pathsOf(vs))
			}
		})
	}
}

// TestValidate_WorldOrderEdges verifies the orders between two fields of the
// world (spec-agents.md 12, spec-beliefs.md 11): a pair that is equal is valid
// where the order is non-strict, and a pair that is the wrong way round is
// refused with the value and the bound of the rule.
//
// Requirement: FR-CFG-007, FR-SPN-038
func TestValidate_WorldOrderEdges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(w *economysim.WorldParams)
		path   string // the field refused, "" when the world is valid
		value  string
		bound  string
	}{
		{"stock low above stock high", func(w *economysim.WorldParams) { w.BeliefStockLow = 16_000 }, "belief_stock_low", "16000", "14999"},
		{"width min equal to width init", func(w *economysim.WorldParams) { w.BeliefWidthMin = w.BeliefWidthInit }, "", "", ""},
		{"width init equal to width max", func(w *economysim.WorldParams) { w.BeliefWidthInit = w.BeliefWidthMax }, "", "", ""},
		{"width step equal to width max", func(w *economysim.WorldParams) { w.BeliefWidthStep = w.BeliefWidthMax }, "", "", ""},
		{"cost ceiling equal to cost floor", func(w *economysim.WorldParams) { w.CostCeiling = w.CostFloor }, "", "", ""},
		{"change privation above remove privation", func(w *economysim.WorldParams) { w.ChangePrivation = w.RemovePrivation + 1 }, "change_privation", "11", "ladder_order"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c.World)
			err := c.Validate()
			if tt.path == "" {
				if err != nil {
					t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
				}
				return
			}
			vs, _ := violations(err)
			v := hasPath(vs, economysim.ErrOutOfBounds, []string{"world", tt.path})
			if v == nil {
				t.Fatalf("Validate(%s): no ErrOutOfBounds at world/%s; paths: %v", tt.name, tt.path, pathsOf(vs))
			}
			if v.Value != tt.value || v.Bound != tt.bound {
				t.Errorf("Validate(%s) value, bound = %q, %q, want %q, %q", tt.name, v.Value, v.Bound, tt.value, tt.bound)
			}
		})
	}
}

// TestValidate_CalendarEdges verifies the edges of the derived year
// (spec-calendar.md 7.2): a month of exactly MaxDaysPerMonth days and a year of
// exactly MaxDaysPerYear days are valid, a season longer than the year is
// refused even when the year is one day short of the bound, and the bound shown
// for a tick that does not divide the year is the ticks of a year in days.
//
// Requirement: FR-CAL-014
func TestValidate_CalendarEdges(t *testing.T) {
	t.Parallel()
	months := func(days ...int64) []economysim.Month {
		ms := make([]economysim.Month, len(days))
		for i, d := range days {
			ms[i] = economysim.Month{ID: "m" + string(rune('a'+i)), Days: d}
		}
		return ms
	}
	rep := func(n int, d int64) []int64 {
		out := make([]int64, n)
		for i := range out {
			out[i] = d
		}
		return out
	}
	tests := []struct {
		name   string
		mutate func(cal *economysim.Calendar)
		rule   error // nil: valid
		path   []string
		bound  string
	}{
		{"one month of exactly MaxDaysPerMonth", func(cal *economysim.Calendar) {
			cal.Months = months(economysim.MaxDaysPerMonth)
			cal.Seasons = []economysim.Season{{ID: "spring", Days: economysim.MaxDaysPerMonth}}
			cal.SeasonStart = 0
		}, nil, nil, ""},
		{"year of exactly MaxDaysPerYear", func(cal *economysim.Calendar) {
			cal.Months = months(rep(10, economysim.MaxDaysPerMonth)...)
			cal.Seasons = []economysim.Season{{ID: "spring", Days: economysim.MaxDaysPerYear}}
			cal.SeasonStart = 0
		}, nil, nil, ""},
		{"season past the year of MaxDaysPerYear less one", func(cal *economysim.Calendar) {
			// 9 * 10 000 + 9 999 = 99 999 days; the season is 100 005.
			cal.Months = months(append(rep(9, economysim.MaxDaysPerMonth), economysim.MaxDaysPerMonth-1)...)
			cal.Seasons = []economysim.Season{{ID: "all", Days: economysim.MaxDaysPerYear + 5}}
			cal.SeasonStart = 0
		}, economysim.ErrCalendarSeasons, []string{"calendar", "seasons"}, "99999"},
		{"tick of 7 days in a year of 365", func(cal *economysim.Calendar) {
			cal.DaysPerTick = 7
		}, economysim.ErrCalendarTick, []string{"calendar", "days_per_tick"}, "365"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c.Calendar)
			// The work floors of the pack are shares of a 365-tick year.
			for i := range c.Roles {
				c.Roles[i].MinWorkShare = 0
			}
			err := c.Validate()
			if tt.rule == nil {
				if err != nil {
					t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
				}
				return
			}
			vs, _ := violations(err)
			v := hasPath(vs, tt.rule, tt.path)
			if v == nil {
				t.Fatalf("Validate(%s): no %v at %v; paths: %v", tt.name, tt.rule, tt.path, pathsOf(vs))
			}
			if v.Bound != tt.bound {
				t.Errorf("Validate(%s) bound = %q, want %q", tt.name, v.Bound, tt.bound)
			}
		})
	}
}

// TestValidate_CalendarFactsFeedTheRoleWorkBound verifies that the work a role
// expects is held against the derived ticks of a year only when the calendar
// is wholly valid: a calendar with one invalid field leaves the widest bound
// (MaxTicksPerYear), and a valid one with 73 ticks a year or one tick a year
// holds a role to that.
//
// Requirement: FR-CFG-048
func TestValidate_CalendarFactsFeedTheRoleWorkBound(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		work   int64
		mutate func(cal *economysim.Calendar)
		want   bool // the role's expected work is refused
	}{
		{"valid calendar, work above the year", 400, func(*economysim.Calendar) {}, true},
		{"days per week out of bounds", 400, func(cal *economysim.Calendar) { cal.DaysPerWeek = 101 }, false},
		{"season start out of bounds", 400, func(cal *economysim.Calendar) { cal.SeasonStart = 365 }, false},
		{"seasons that do not cover the year", 400, func(cal *economysim.Calendar) { cal.Seasons[0].Days = 93 }, false},
		{"tick of 5 days: 73 ticks, work of 74", 74, func(cal *economysim.Calendar) { cal.DaysPerTick = 5 }, true},
		{"tick of 5 days: 73 ticks, work of 73", 73, func(cal *economysim.Calendar) { cal.DaysPerTick = 5 }, false},
		{"one tick a year, work of 2", 2, func(cal *economysim.Calendar) {
			cal.Months = []economysim.Month{{ID: "m01", Days: 1}}
			cal.Seasons, cal.SeasonStart = nil, 0
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c.Calendar)
			roleOf(&c, "cook").ExpectedWorkTicks = tt.work
			vs, _ := violations(c.Validate())
			got := hasPath(vs, economysim.ErrOutOfBounds, []string{"roles", "cook", "expected_work_ticks"}) != nil
			if got != tt.want {
				t.Errorf("Validate(%s): ErrOutOfBounds at roles/cook/expected_work_ticks = %t, want %t; paths: %v", tt.name, got, tt.want, pathsOf(vs))
			}
		})
	}
}

// TestValidate_CommodityAndRecipeEdges verifies the edges of three rules: a
// commodity with exactly eight classes is valid and one with nine is not, a
// discarded output with a negative cost share is a cost share error (not only
// a range error), and a tool of durability 1 is durable.
//
// Requirement: FR-GDS-006, FR-PRO-012
func TestValidate_CommodityAndRecipeEdges(t *testing.T) {
	t.Parallel()
	classes := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = "class_" + string(rune('a'+i))
		}
		return out
	}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   error // nil: valid
		path   []string
	}{
		{"eight classes", func(c *economysim.Config) { commodityOf(c, "seed").Classes = classes(8) }, nil, nil},
		{"nine classes", func(c *economysim.Config) { commodityOf(c, "seed").Classes = classes(9) },
			economysim.ErrOutOfBounds, []string{"commodities", "seed", "classes"}},
		{"discarded output with a negative cost share", func(c *economysim.Config) { recipeOf(c, "refine").Outputs[1].CostShare = -1 },
			economysim.ErrCostShares, []string{"recipes", "refine", "outputs", "waste", "cost_share"}},
		{"tool of durability 1", func(c *economysim.Config) {
			commodityOf(c, "tool_a").Durability = 1
			recipeOf(c, "grow_staple").Requires[0].WearPerUse = 1
		}, nil, nil},
		{"tool of durability 0", func(c *economysim.Config) {
			commodityOf(c, "tool_a").Durability = 0
		}, economysim.ErrToolNotDurable, []string{"recipes", "grow_staple", "requires", "tool_a", "commodity"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			err := c.Validate()
			if tt.rule == nil {
				if err != nil {
					t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
				}
				return
			}
			vs, _ := violations(err)
			if hasPath(vs, tt.rule, tt.path) == nil {
				t.Errorf("Validate(%s): no %v at %v; paths: %v", tt.name, tt.rule, tt.path, pathsOf(vs))
			}
		})
	}
}

// boundsAt returns the Bound of every error of vs of the rule at the path.
func boundsAt(vs []*economysim.ValidationError, rule error, path []string) []string {
	var out []string
	for _, v := range vs {
		if errors.Is(v, rule) && slices.Equal(v.Path, path) {
			out = append(out, v.Bound)
		}
	}
	return out
}

// TestValidate_LowestTargetSpreadEdges verifies the lowest-target rule of
// FR-WEL-025: the rule reads a TargetRatio from 10 000 to MaxBasisPoints and a
// TargetSpread from 0 to 9 999 only, so a ratio of exactly 10 000 refuses a
// spread of 1 (bound 0), a ratio of MaxBasisPoints refuses a spread of 9 999
// (bound 10 000 - ceil(10^8 / 10^6) = 9 900), and a ratio out of its own range
// or a spread of 10 000 is reported once, by its range, not by this rule too.
// The ceiling of the bound is exact: ratio 20 000 gives 10 000 - 5 000.
//
// Requirement: FR-WEL-025
func TestValidate_LowestTargetSpreadEdges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		ratio  int64
		spread int64
		bounds []string // the Bounds of the ErrOutOfBounds errors at target_spread
	}{
		{"ratio 10 000, spread 0: target 10 000", 10_000, 0, nil},
		// 10 000 * 9 999 / 10 000 = 9 999 < 10 000; most = 10 000 - 10 000 = 0.
		{"ratio 10 000, spread 1", 10_000, 1, []string{"0"}},
		// 1 000 000 * 1 / 10 000 = 100 < 10 000; most = 10 000 - 100 = 9 900.
		{"ratio MaxBasisPoints, spread 9 999", economysim.MaxBasisPoints, 9_999, []string{"9900"}},
		// The ratio is out of range (reported at target_ratio); this rule is silent.
		{"ratio above MaxBasisPoints, spread 9 999", economysim.MaxBasisPoints + 1, 9_999, nil},
		{"ratio below 10 000, spread 0", 9_999, 0, nil},
		// The spread is out of range: reported once, by its range (bound 9 999).
		{"spread 10 000 is reported once", 15_000, 10_000, []string{"9999"}},
		{"spread 10 001 is reported once", 15_000, 10_001, []string{"9999"}},
		// 20 000 * 4 000 / 10 000 = 8 000 < 10 000; most = 10 000 - 10^8/20 000 = 5 000.
		{"ratio 20 000, spread 6 000", 20_000, 6_000, []string{"5000"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			ro := roleOf(&c, "cook")
			ro.TargetRatio, ro.TargetSpread = tt.ratio, tt.spread
			vs, _ := violations(c.Validate())
			got := boundsAt(vs, economysim.ErrOutOfBounds, []string{"roles", "cook", "target_spread"})
			if !slices.Equal(got, tt.bounds) {
				t.Errorf("Validate(%s) bounds at target_spread = %q, want %q", tt.name, got, tt.bounds)
			}
		})
	}
}

// TestValidate_TraineeshipNeedsWeightOneEdges verifies that the rule of
// FR-CFG-054 reads a TrainingTicks above 0 (not 0, not a negative one, which
// is a range error of its own) and a scale above 1 (a scale of exactly 1 is
// valid).
//
// Requirement: FR-CFG-054
func TestValidate_TraineeshipNeedsWeightOneEdges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		training int64
		scale    int64
		want     bool
	}{
		{"training 1, scale 2", 1, 2, true},
		{"training 1, scale 1", 1, 1, false},
		{"training 0, scale 2", 0, 2, false},
		{"training -1, scale 2", -1, 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			roleOf(&c, "cook").TrainingTicks = tt.training
			c.Groups[0].Scale = tt.scale // the cook's group
			vs, _ := violations(c.Validate())
			got := slices.Contains(boundsAt(vs, economysim.ErrOutOfBounds, []string{"groups", "north", "cook", "scale"}),
				"traineeship_needs_weight_one")
			if got != tt.want {
				t.Errorf("Validate(%s): traineeship_needs_weight_one = %t, want %t", tt.name, got, tt.want)
			}
		})
	}
}

// TestValidate_SeedPriceBandEdges verifies the edges of the seed price rules
// of FR-VIA-030. The reference band reads RefBandLow from 1 and RefBandHigh up
// to MaxBasisPoints (outside them the band is not defined and the check is
// off), a price of 1 is not "below 1" (it is refused by the floor, bound 50, and the band),
// and a commodity whose reference price is exactly 1 or exactly MaxPrice has
// its band checked.
//
// Requirement: FR-VIA-030
func TestValidate_SeedPriceBandEdges(t *testing.T) {
	t.Parallel()
	fuel := []string{"warm_start", "prices", "north", "fuel", "price"}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		bounds []string // the Bounds of the ErrWarmStartPrice errors at the fuel price
	}{
		// RefBandLow 1: the band is [179 * 1 / 10 000, 179 * 100 000 / 10 000] =
		// [1, 1 790]; 100 000 is above its top.
		{"RefBandLow of 1 is a band", func(c *economysim.Config) {
			c.World.RefBandLow = 1
			c.WarmStart.Prices[0].Price = 100_000
		}, []string{"1790"}},
		{"RefBandLow of 0 is no band", func(c *economysim.Config) {
			c.World.RefBandLow = 0
			c.WarmStart.Prices[0].Price = 100_000
		}, nil},
		// RefBandHigh of MaxBasisPoints: the top is 179 * 1 000 000 / 10 000 = 17 900.
		{"RefBandHigh of MaxBasisPoints is a band", func(c *economysim.Config) {
			c.World.RefBandHigh = economysim.MaxBasisPoints
			c.WarmStart.Prices[0].Price = 20_000
		}, []string{"17900"}},
		{"RefBandHigh above MaxBasisPoints is no band", func(c *economysim.Config) {
			c.World.RefBandHigh = economysim.MaxBasisPoints + 1
			c.WarmStart.Prices[0].Price = 20_000
		}, nil},
		// The floor is ceil(10 000 / 200) = 50 and the band bottom is
		// ceil(179 * 1 000 / 10 000) = 18: a price of 1 is below both, and a
		// "below 1" refusal would stop at the bound 1.
		{"price of 1 is refused by the floor and the band", func(c *economysim.Config) {
			c.WarmStart.Prices[0].Price = 1
		}, []string{"18", "50"}},
		// Reference price 1: the band is [1 000 / 10 000, 100 000 / 10 000] = up
		// to 10; 1 000 is above it.
		{"reference price of 1 has a band", func(c *economysim.Config) {
			commodityOf(c, "fuel").ReferencePrice = 1
			c.WarmStart.Prices[0].Price = 1_000
		}, []string{"10"}},
		// Reference price MaxPrice: the bottom is 10^9 * 1 000 / 10 000 = 10^8; 1 000 is below.
		{"reference price of MaxPrice has a band", func(c *economysim.Config) {
			commodityOf(c, "fuel").ReferencePrice = economysim.MaxPrice
			c.WarmStart.Prices[0].Price = 1_000
		}, []string{"100000000"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			vs, _ := violations(c.Validate())
			got := boundsAt(vs, economysim.ErrWarmStartPrice, fuel)
			if !slices.Equal(got, tt.bounds) {
				t.Errorf("Validate(%s) bounds at the fuel seed price = %q, want %q", tt.name, got, tt.bounds)
			}
		})
	}
}

// TestValidate_SeedWageLongestRecipeEdges verifies the edges of the starting
// wage rule of spec-viability.md 8.2.5: a wage of exactly MaxMoney is within
// the range, so the product with the longest recipe is what refuses it (bound
// MaxMoney / 60); a wage times the longest recipe of exactly MaxMoney is valid;
// a recipe of exactly MaxRecipeTicks is the longest there is, and one of
// MaxRecipeTicks + 1 (a range error of its own) is not counted.
//
// Guard: the wage times the longest job is at most MaxMoney (spec-viability.md 8.2.5).
func TestValidate_SeedWageLongestRecipeEdges(t *testing.T) {
	t.Parallel()
	wagePath := []string{"warm_start", "wages", "north", "wage"}
	tests := []struct {
		name     string
		duration int64 // of grow_staple; the others last at most 3
		wage     int64
		bounds   []string
	}{
		// 10^18 / 60 = 16 666 666 666 666 666 (truncated).
		{"wage of MaxMoney against 60 ticks", 60, economysim.MaxMoney, []string{"16666666666666666"}},
		// 10^16 * 100 = 10^18 = MaxMoney.
		{"wage times 100 ticks is MaxMoney", 100, 10_000_000_000_000_000, nil},
		// 10^13 + 1 times 100 000 is above MaxMoney; 10^18 / 100 000 = 10^13.
		{"wage above MaxMoney / MaxRecipeTicks", economysim.MaxRecipeTicks, 10_000_000_000_001, []string{"10000000000000"}},
		{"wage of MaxMoney / MaxRecipeTicks", economysim.MaxRecipeTicks, 10_000_000_000_000, nil},
		// 10^13 * 60 is far below MaxMoney, and 10^13 * 100 001 is above it.
		{"recipe above MaxRecipeTicks is not counted", economysim.MaxRecipeTicks + 1, 10_000_000_000_000, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			recipeOf(&c, "grow_staple").Duration = tt.duration
			c.WarmStart.Wages[0].Wage = economysim.Money(tt.wage)
			// The spawner's wage times the expected work is its own rule.
			for i := range c.Roles {
				c.Roles[i].ExpectedWorkTicks, c.Roles[i].MinWorkShare = 1, 0
			}
			vs, _ := violations(c.Validate())
			got := boundsAt(vs, economysim.ErrWarmStartPrice, wagePath)
			if !slices.Equal(got, tt.bounds) {
				t.Errorf("Validate(%s) bounds at the wage = %q, want %q", tt.name, got, tt.bounds)
			}
		})
	}
}

// TestValidate_WarmStartMoneyStockAndJobEdges verifies the lower and upper
// edges of the warm start's own bounds: a pool or a group with money of
// exactly 0 is valid, a stock quantity of exactly MaxQuantity (at weight 1) is
// valid, a job share of exactly 0 or 10 000 is valid and -1 is not, and the
// shares of a group of exactly 10 000 are valid.
//
// Requirement: FR-VIA-033
// Guard: money is at least 0 and a stock quantity at most MaxQuantity (spec-viability.md 8.2).
func TestValidate_WarmStartMoneyStockAndJobEdges(t *testing.T) {
	t.Parallel()
	job := func(r string, bp int64) economysim.JobSpec { return economysim.JobSpec{Recipe: r, InProgressBP: bp} }
	jobsPath := []string{"warm_start", "groups", "north", "maker", "jobs"}
	shareA := append(jobsPath[:len(jobsPath):len(jobsPath)], "make_tool_a", "in_progress")
	tests := []struct {
		name     string
		mutate   func(c *economysim.Config)
		rule     error
		path     []string
		wantErrs int
	}{
		{"pool money of 0", func(c *economysim.Config) {
			c.WarmStart.Pools = []economysim.WarmPool{{Region: "north", Money: 0}}
		}, economysim.ErrOutOfBounds, []string{"warm_start", "pools", "north", "money"}, 0},
		{"group money of 0", func(c *economysim.Config) { c.WarmStart.Groups[0].Money = 0 },
			economysim.ErrOutOfBounds, []string{"warm_start", "groups", "north", "cook", "money"}, 0},
		{"stock of MaxQuantity at weight 1", func(c *economysim.Config) {
			c.Groups[0].Households = 1 // one agent of weight 1
			c.WarmStart.Groups[0].Stock[0].Quantity = economysim.MaxQuantity
		}, economysim.ErrOverflow, []string{"warm_start", "groups", "north", "cook", "stock", "fuel", "quantity"}, 0},
		{"job share of 0", func(c *economysim.Config) {
			c.WarmStart.Groups[4].Jobs = []economysim.JobSpec{job("make_tool_a", 0)}
		}, economysim.ErrWarmStartJob, shareA, 0},
		{"job share of 10 000", func(c *economysim.Config) {
			c.WarmStart.Groups[4].Jobs = []economysim.JobSpec{job("make_tool_a", 10_000)}
		}, economysim.ErrWarmStartJob, shareA, 0},
		{"job share of -1", func(c *economysim.Config) {
			c.WarmStart.Groups[4].Jobs = []economysim.JobSpec{job("make_tool_a", -1)}
		}, economysim.ErrWarmStartJob, shareA, 1},
		{"shares of exactly 10 000", func(c *economysim.Config) {
			c.WarmStart.Groups[4].Jobs = []economysim.JobSpec{job("make_tool_a", 5_000), job("make_tool_b", 5_000)}
		}, economysim.ErrWarmStartJob, jobsPath, 0},
		{"shares of 10 001", func(c *economysim.Config) {
			c.WarmStart.Groups[4].Jobs = []economysim.JobSpec{job("make_tool_a", 5_000), job("make_tool_b", 5_001)}
		}, economysim.ErrWarmStartJob, jobsPath, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			vs, _ := violations(c.Validate())
			var got int
			for _, v := range vs {
				if errors.Is(v, tt.rule) && slices.Equal(v.Path, tt.path) {
					got++
				}
			}
			if got != tt.wantErrs {
				t.Errorf("Validate(%s): %d errors of %v at %v, want %d; paths: %v", tt.name, got, tt.rule, tt.path, tt.wantErrs, pathsOf(vs))
			}
		})
	}
}

// TestValidate_ReferencePriceOneIsBelowTheFloor verifies that the lowest legal
// reference price, 1, of a tradable commodity is checked against the
// resolution floor: with BeliefStep 1 the floor is ceil(10 000 / 1) = 10 000.
//
// Requirement: FR-BEL-029
func TestValidate_ReferencePriceOneIsBelowTheFloor(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	c.World.BeliefStep = 1
	commodityOf(&c, "fuel").ReferencePrice = 1
	vs, _ := violations(c.Validate())
	v := hasPath(vs, economysim.ErrOutOfBounds, []string{"commodities", "fuel", "reference_price"})
	if v == nil || v.Bound != "10000" {
		t.Errorf("Validate(price 1, BeliefStep 1) = %v, want ErrOutOfBounds at commodities/fuel/reference_price with bound 10000", pathsOf(vs))
	}
}

// TestValidate_BasketCheckEdges verifies the edges of the reference basket
// rule (spec-currency.md 5.1): the check runs for a tolerance in [0, 10 000]
// and a basket in [0, MaxMoney] and when every upkeep price and quantity is in
// bounds (price 1..MaxPrice, quantity 1..MaxQuantity/MaxWeight); outside them
// another rule speaks and no basket mismatch is added.
//
// Requirement: FR-CUR-022
func TestValidate_BasketCheckEdges(t *testing.T) {
	t.Parallel()
	perHousehold := economysim.Quantity(economysim.MaxQuantity / economysim.MaxWeight)
	tests := []struct {
		name     string
		tol      int64
		basket   economysim.Money
		price    economysim.Price
		qty      economysim.Quantity
		mismatch bool
	}{
		// Cost is price * qty. Allowed difference is tol * basket / 10 000.
		{"tolerance 0 at the cost", 0, 999, 999, 1, false},
		{"tolerance 0 one off the cost", 0, 1_000, 999, 1, true},
		{"negative tolerance is not evaluated", -1, 1_000, 999, 1, false},
		{"tolerance 10 000 is evaluated", 10_000, 100, 999, 1, true}, // 899 > 100
		{"tolerance above 10 000 is not evaluated", 10_001, 100, 999, 1, false},
		{"basket 0 is evaluated", 1_000, 0, 999, 1, true}, // 999 > 0
		{"basket MaxMoney is evaluated", 1_000, economysim.MaxMoney, 999, 1, true},
		{"basket above MaxMoney is not evaluated", 1_000, economysim.MaxMoney + 1, 999, 1, false},
		{"price 1 is defined", 1_000, 1_000, 1, 1, true}, // cost 1, 999 > 100
		{"price MaxPrice is defined", 1_000, 1_000, economysim.MaxPrice, 1, true},
		{"price above MaxPrice is not defined", 1_000, 1_000, economysim.MaxPrice + 1, 1, false},
		{"quantity at the per-household bound is defined", 1_000, 5_000, 1, perHousehold, true}, // cost 1 000, 4 000 > 500
		{"quantity above the per-household bound is not defined", 1_000, 5_000, 1, perHousehold + 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			c.World.BasketTolerance = tt.tol
			c.World.Upkeep = []economysim.Item{{Commodity: "fuel", Quantity: tt.qty}}
			commodityOf(&c, "fuel").ReferencePrice = tt.price
			for i := range c.Regions {
				c.Regions[i].ReferenceBasket = tt.basket
				c.Regions[i].ExtraUpkeep = nil
			}
			err := c.Validate()
			if got := errors.Is(err, economysim.ErrBasketMismatch); got != tt.mismatch {
				t.Errorf("errors.Is(Validate(%s), ErrBasketMismatch) = %t, want %t; error: %v", tt.name, got, tt.mismatch, err)
			}
		})
	}
}

// TestValidate_RecipeAmountOverflowEdges verifies the edges of the products of
// a recipe with the weight of the roles that list it (spec-config.md 4.2,
// spec-production.md 10.3): an amount outside [1, MaxQuantity] is not
// multiplied (its own rule has spoken), the largest yield is the largest table
// entry inside [0, MaxYieldBP] (a Sited recipe has no yield of its own, one
// that is not has at least 10 000), a recipe no role lists has no weight, and
// a price outside its bound is left out of the cost of the inputs. The group is
// 10 000 households in agents of weight 1 000.
//
// Requirement: FR-CFG-010, FR-PRO-039
func TestValidate_RecipeAmountOverflowEdges(t *testing.T) {
	t.Parallel()
	outPath := []string{"recipes", "mine", "outputs", "gem", "quantity"}
	inPath := []string{"recipes", "mine", "inputs", "grain", "quantity"}
	costPath := []string{"recipes", "mine", "inputs"}
	out := func(q int64, sited bool, y ...economysim.YieldProfile) func(*economysim.Config, *economysim.Recipe) {
		return func(c *economysim.Config, r *economysim.Recipe) {
			r.Outputs = []economysim.Output{{Commodity: "gem", Quantity: economysim.Quantity(q), CostShare: 10_000}}
			r.Sited = sited
			c.Yields = y
		}
	}
	all := func(bp int64) economysim.YieldProfile {
		return economysim.YieldProfile{Region: "home", Recipe: "mine", HasAll: true, All: bp}
	}
	season := func(bp int64) economysim.YieldProfile {
		return economysim.YieldProfile{Region: "home", Recipe: "mine", Seasons: []economysim.SeasonYield{{Season: "spring", BP: bp}}}
	}
	in := func(q int64) func(*economysim.Config, *economysim.Recipe) {
		return func(c *economysim.Config, r *economysim.Recipe) {
			r.Inputs = []economysim.Item{{Commodity: "grain", Quantity: economysim.Quantity(q)}}
		}
	}
	// Three inputs of 10^9 units: two at the largest price (2 * 10^18 above
	// MaxMoney) and one at a price below the bound, which is left out.
	costs := func(third economysim.Price) func(*economysim.Config, *economysim.Recipe) {
		return func(c *economysim.Config, r *economysim.Recipe) {
			for _, id := range []string{"ore", "coal", "slag"} {
				p := economysim.Price(economysim.MaxPrice)
				if id == "slag" {
					p = third
				}
				withProduct(c, id, p, true)
				r.Inputs = append(r.Inputs, economysim.Item{Commodity: id, Quantity: economysim.MaxQuantity})
			}
		}
	}
	costPlusOne := func(price economysim.Price) func(*economysim.Config, *economysim.Recipe) {
		return func(c *economysim.Config, r *economysim.Recipe) {
			withProduct(c, "ore", economysim.MaxPrice, true)
			withProduct(c, "mite", price, true)
			r.Inputs = []economysim.Item{{Commodity: "ore", Quantity: economysim.MaxQuantity}, {Commodity: "mite", Quantity: 1}}
		}
	}
	tests := []struct {
		name   string
		mutate func(*economysim.Config, *economysim.Recipe)
		path   []string
		want   bool
		noRole bool
	}{
		// Weight 1 000 * 10^6 units * a yield of 1 000 bp * 10 / 10 000 = 10^9.
		{"yield of 1 000 bp is at the bound", out(1_000_000, true, all(1_000)), outPath, false, false},
		{"yield of 1 001 bp is above it", out(1_000_000, true, all(1_001)), outPath, true, false},
		{"All of exactly MaxYieldBP counts", out(1_000_000, true, all(economysim.MaxYieldBP)), outPath, true, false},
		{"All above MaxYieldBP is not an entry", out(1_000_000, true, all(economysim.MaxYieldBP+1)), outPath, false, false},
		{"season of 1 000 bp is at the bound", out(1_000_000, true, season(1_000)), outPath, false, false},
		{"season of 1 001 bp is above it", out(1_000_000, true, season(1_001)), outPath, true, false},
		{"season of exactly MaxYieldBP counts", out(1_000_000, true, season(economysim.MaxYieldBP)), outPath, true, false},
		{"season above MaxYieldBP is not an entry", out(1_000_000, true, season(economysim.MaxYieldBP+1)), outPath, false, false},
		{"Sited recipe with no entry has no yield", out(1_000_000, true), outPath, false, false},
		// 1 000 * 10^5 * 10 = 10^9; one more unit is above.
		{"not Sited: at least 10 000 bp, at the bound", out(100_000, false), outPath, false, false},
		{"not Sited: at least 10 000 bp, above it", out(100_001, false), outPath, true, false},
		// 1 000 * 10^6 = 10^9 at the bound; 10^9 units at weight 1 000 is above.
		{"input of weight times quantity at the bound", in(1_000_000), inPath, false, false},
		{"input one above the bound", in(1_000_001), inPath, true, false},
		{"input of MaxQuantity", in(economysim.MaxQuantity), inPath, true, false},
		{"input above MaxQuantity is not multiplied", in(economysim.MaxQuantity + 1), inPath, false, false},
		{"input of 0 is not multiplied", in(0), inPath, false, false},
		{"negative input is not multiplied", in(-9_000_000_000_000_000_000), inPath, false, false},
		{"output of MaxQuantity", out(economysim.MaxQuantity, false), outPath, true, false},
		{"output above MaxQuantity is not multiplied", out(economysim.MaxQuantity+1, false), outPath, false, false},
		// -10^15 * 1 000 * 10^5 / 10^4 is below int64.
		{"negative output is not multiplied", out(-1_000_000_000_000_000, false), outPath, false, false},
		{"cost of the inputs above MaxMoney", costs(economysim.MaxPrice), costPath, true, false},
		// 2 * 10^18 - 10^18 would be at the bound if the price were counted.
		{"a price below 1 is left out of the cost", costs(-1_000_000_000), costPath, true, false},
		{"a price above MaxPrice is left out of the cost", func(c *economysim.Config, r *economysim.Recipe) {
			costs(economysim.MaxPrice)(c, r)
			r.Inputs = r.Inputs[:1]
			withProduct(c, "dross", economysim.MaxPrice+1, true)
			r.Inputs = append(r.Inputs, economysim.Item{Commodity: "dross", Quantity: economysim.MaxQuantity})
		}, costPath, false, false},
		// 10^9 * 10^9 = MaxMoney is at the bound; one more unit of any price in
		// [1, MaxPrice] is above it.
		{"a price of 1 counts in the cost", costPlusOne(1), costPath, true, false},
		{"a price below MaxPrice counts in the cost", costPlusOne(1_000), costPath, true, false},
		{"a recipe no role lists has no cost check", costs(economysim.MaxPrice), costPath, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := minimalConfig()
			c.Groups[0].Households, c.Groups[0].Scale = 10_000, 10_000
			withProduct(&c, "gem", 1_000, false)
			r := economysim.DefaultRecipe()
			r.ID = "mine"
			tt.mutate(&c, &r)
			c.Recipes = append(c.Recipes, r)
			if !tt.noRole {
				c.Roles[0].Recipes = []string{"mine"}
			}
			vs, _ := violations(c.Validate())
			got := hasPath(vs, economysim.ErrOverflow, tt.path) != nil
			if got != tt.want {
				t.Errorf("Validate(%s): ErrOverflow at %v = %t, want %t; paths: %v", tt.name, tt.path, got, tt.want, pathsOf(vs))
			}
		})
	}
}

// TestValidate_RecipeCostSkipsNonPositiveAndOversizeInputs verifies that an
// input whose quantity is below 1 or above MaxQuantity is left out of the cost
// of a recipe's inputs: a negative quantity must not offset the other inputs,
// and an oversize one is not priced (it has its own error).
//
// Guard: the cost sum counts only inputs of 1 to MaxQuantity units.
func TestValidate_RecipeCostSkipsNonPositiveAndOversizeInputs(t *testing.T) {
	t.Parallel()
	costPath := []string{"recipes", "mine", "inputs"}
	tests := []struct {
		name   string
		inputs func(c *economysim.Config) []economysim.Item
		want   bool
	}{
		// Two inputs of 10^9 units at 10^9 = 2 * 10^18 > MaxMoney. A
		// counted -10^9 units at 10^9 would bring it to MaxMoney exactly.
		{"a negative quantity does not offset the cost", func(c *economysim.Config) []economysim.Item {
			withProduct(c, "ore", economysim.MaxPrice, true)
			withProduct(c, "coal", economysim.MaxPrice, true)
			withProduct(c, "debt", economysim.MaxPrice, true)
			return []economysim.Item{
				{Commodity: "ore", Quantity: economysim.MaxQuantity},
				{Commodity: "coal", Quantity: economysim.MaxQuantity},
				{Commodity: "debt", Quantity: -economysim.MaxQuantity},
			}
		}, true},
		// (10^9 + 1) * 10^9 > MaxMoney only if the oversize quantity is priced.
		{"an oversize quantity is not priced", func(c *economysim.Config) []economysim.Item {
			withProduct(c, "ore", economysim.MaxPrice, true)
			return []economysim.Item{{Commodity: "ore", Quantity: economysim.MaxQuantity + 1}}
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := minimalConfig()
			r := economysim.DefaultRecipe()
			r.ID = "mine"
			r.Inputs = tt.inputs(&c)
			c.Recipes = append(c.Recipes, r)
			c.Roles[0].Recipes = []string{"mine"}
			vs, _ := violations(c.Validate())
			if got := hasPath(vs, economysim.ErrOverflow, costPath) != nil; got != tt.want {
				t.Errorf("Validate(%s): ErrOverflow at %v = %t, want %t; paths: %v", tt.name, costPath, got, tt.want, pathsOf(vs))
			}
		})
	}
}

// TestValidate_RoleWithoutGroupHasNoProductChecks verifies that a role no group
// uses has weight 0 and is not checked for product overflow, even when its
// recipe's income would overflow int64 at a positive weight.
//
// Guard: a role with weight 0 is skipped by the product checks.
func TestValidate_RoleWithoutGroupHasNoProductChecks(t *testing.T) {
	t.Parallel()
	incomePath := []string{"roles", "idle", "recipes"}
	for _, grouped := range []bool{false, true} {
		t.Run(map[bool]string{false: "no group", true: "with a group"}[grouped], func(t *testing.T) {
			t.Parallel()
			c := minimalConfig()
			withProduct(&c, "gem", economysim.MaxPrice, false)
			r := economysim.DefaultRecipe()
			r.ID = "mine"
			r.Sited = true
			r.Outputs = []economysim.Output{{Commodity: "gem", Quantity: economysim.MaxQuantity, CostShare: 10_000}}
			c.Recipes = append(c.Recipes, r)
			c.Yields = []economysim.YieldProfile{{Region: "home", Recipe: "mine", HasAll: true, All: economysim.MaxYieldBP}}
			idle := economysim.DefaultRole()
			idle.ID = "idle"
			idle.Recipes = []string{"mine"}
			c.Roles = append(c.Roles, idle)
			if grouped {
				c.Groups = append(c.Groups, economysim.Group{Region: "home", Role: "idle", Households: 10})
			}
			vs, _ := violations(c.Validate())
			got := hasPath(vs, economysim.ErrOverflow, incomePath) != nil
			if got != grouped {
				t.Errorf("Validate(grouped=%t): ErrOverflow at %v = %t, want %t; paths: %v", grouped, incomePath, got, grouped, pathsOf(vs))
			}
		})
	}
}

// TestValidate_RoleTargetProductEdges verifies the edges of the held-units
// check of a role's targets (spec-agents.md 7.3, spec-config.md 4.2): the
// weight times the units a household holds of one commodity, upkeep * (1 +
// CoverTicks) + the inputs of InputJobs jobs, is at most MaxQuantity. The check
// is made only when CoverTicks is in [1, 1 000] and InputJobs is in [1, 100];
// an upkeep or extra-upkeep quantity counts only inside [1, perHousehold =
// 1 000]; an input counts only inside [1, MaxQuantity]. The group has weight
// 1 000, so the largest held amount is 10^6 units.
//
// Requirement: FR-CFG-010, FR-AGT-038
func TestValidate_RoleTargetProductEdges(t *testing.T) {
	t.Parallel()
	targetPath := []string{"roles", "farmer", "targets", "grain"}
	grain := func(q ...int64) []economysim.Item {
		items := make([]economysim.Item, 0, len(q))
		for _, v := range q {
			items = append(items, economysim.Item{Commodity: "grain", Quantity: economysim.Quantity(v)})
		}
		return items
	}
	tests := []struct {
		name     string
		cover    int64
		jobs     int64
		roleUp   []economysim.Item
		regionUp []economysim.Item
		input    int64 // quantity of grain a recipe of the role consumes; 0 for none
		want     bool
	}{
		// 1 000 * (1 + 999) * 1 000 = 10^9 at the bound; one more cover tick is above.
		{"upkeep at the bound", 999, 1, grain(1_000), nil, 0, false},
		{"one cover tick above the bound", 1_000, 1, grain(1_000), nil, 0, true},
		// 1 000 * 1 000 * 1 = 10^9 at the bound; one unit of upkeep at 1 tick adds 2.
		{"inputs at the bound", 1, 1, nil, nil, 1_000_000, false},
		{"upkeep of 1 counts", 1, 1, grain(1), nil, 1_000_000, true},
		{"one input unit counts", 999, 1, grain(1_000), nil, 1, true},
		// 2 jobs * 10^6 = 2 * 10^6 units; a difference or quotient of 10^6 and 2
		// would be 999 998 or 500 000.
		{"jobs multiply the input", 1, 2, nil, nil, 1_000_000, true},
		{"CoverTicks of 5 is inside the bounds", 5, 2, nil, nil, 1_000_000, true},
		{"CoverTicks of 1 000 is inside the bounds", 1_000, 2, nil, nil, 1_000_000, true},
		{"CoverTicks of 0 is not checked", 0, 2, nil, nil, 1_000_000, false},
		{"CoverTicks of 1 001 is not checked", 1_001, 2, nil, nil, 1_000_000, false},
		{"InputJobs of 0 is not checked", 1_000, 0, grain(1_000), nil, 0, false},
		{"InputJobs of 100 is inside the bounds", 1_000, 100, grain(1_000), nil, 0, true},
		{"InputJobs of 101 is not checked", 1_000, 101, grain(1_000), nil, 0, false},
		// 1 000 + (-500) would be 500 units, 500 * 1 001 * 1 000 below the bound.
		{"a negative upkeep is not counted", 1_000, 1, grain(1_000, -500), nil, 0, true},
		{"an upkeep above perHousehold is not counted", 1_000, 1, grain(1_001), nil, 0, false},
		{"extra upkeep counts", 1_000, 1, nil, grain(1_000), 0, true},
		{"extra upkeep of 1 counts", 1, 1, nil, grain(1), 1_000_000, true},
		{"a negative extra is not counted", 1_000, 1, nil, grain(1_000, -500), 0, true},
		{"an extra above perHousehold is not counted", 1_000, 1, nil, grain(1_001), 0, false},
		// 500 + 500 = 1 000 units (a region's rows add up); 500 alone is 500 500 * 1 000 / 1 000.
		{"a region's rows of one commodity add up", 1_000, 1, nil, grain(500, 500), 0, true},
		// 1 + 999 = 1 000 units; their product 999 would be at most 999 * 1 001 * 1 000 / 1 000 below.
		{"role and region upkeep add up", 1_000, 1, grain(1), grain(999), 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := minimalConfig()
			c.Groups[0].Households, c.Groups[0].Scale = 10_000, 10_000
			c.World.CoverTicks, c.World.InputJobs = tt.cover, tt.jobs
			c.World.Upkeep = nil
			c.Roles[0].Upkeep = tt.roleUp
			c.Regions[0].ExtraUpkeep = tt.regionUp
			if tt.input != 0 {
				r := economysim.DefaultRecipe()
				r.ID = "mine"
				r.Inputs = grain(tt.input)
				c.Recipes = append(c.Recipes, r)
				c.Roles[0].Recipes = []string{"mine"}
			}
			vs, _ := violations(c.Validate())
			got := hasPath(vs, economysim.ErrOverflow, targetPath) != nil
			if got != tt.want {
				t.Errorf("Validate(%s): ErrOverflow at %v = %t, want %t; paths: %v", tt.name, targetPath, got, tt.want, pathsOf(vs))
			}
		})
	}
}

// TestValidate_RoleTargetIgnoresAnOversizeNegativeInput verifies that an input
// below 1 is not multiplied by InputJobs: -9 * 10^18 * 2 wraps to a large
// positive number in int64 and would be reported as an overflow.
//
// Guard: the held-units sum counts only inputs of 1 to MaxQuantity units.
func TestValidate_RoleTargetIgnoresAnOversizeNegativeInput(t *testing.T) {
	t.Parallel()
	c := minimalConfig()
	c.Groups[0].Households, c.Groups[0].Scale = 10_000, 10_000
	c.World.Upkeep = nil
	c.World.InputJobs = 2
	r := economysim.DefaultRecipe()
	r.ID = "mine"
	r.Inputs = []economysim.Item{{Commodity: "grain", Quantity: -9_000_000_000_000_000_000}}
	c.Recipes = append(c.Recipes, r)
	c.Roles[0].Recipes = []string{"mine"}
	vs, _ := violations(c.Validate())
	path := []string{"roles", "farmer", "targets", "grain"}
	if hasPath(vs, economysim.ErrOverflow, path) != nil {
		t.Errorf("Validate(negative input): ErrOverflow at %v, want none; paths: %v", path, pathsOf(vs))
	}
}

// TestValidate_WindowProductEdgesAtMaxWelfareWindow verifies that the income
// times window product is checked for an effective window of exactly
// MaxWelfareWindow (3 650) and not for one above it, which has its own error.
// The window is WindowCycles 2 times the longest Duration (1 825 or 2 000).
//
// Requirement: FR-WEL-010, FR-CFG-052
func TestValidate_WindowProductEdgesAtMaxWelfareWindow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		duration int64
		want     bool
	}{
		// 2 * 1 825 = 3 650.
		{"window of exactly MaxWelfareWindow", 1_825, true},
		// 2 * 2 000 = 4 000 > 3 650.
		{"window above MaxWelfareWindow", 2_000, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := minimalConfig()
			withProduct(&c, "gem", economysim.MaxPrice, false)
			r := economysim.DefaultRecipe()
			r.ID = "mine"
			r.Duration = tt.duration
			r.Outputs = []economysim.Output{{Commodity: "gem", Quantity: economysim.MaxQuantity, CostShare: 10_000}}
			c.Recipes = append(c.Recipes, r)
			c.Roles[0].Recipes = []string{"mine"}
			c.World.WindowCycles = 2
			vs, _ := violations(c.Validate())
			path := []string{"roles", "farmer", "recipes"}
			got := hasPath(vs, economysim.ErrOverflow, path) != nil
			if got != tt.want {
				t.Errorf("Validate(%s): ErrOverflow at %v = %t, want %t; paths: %v", tt.name, path, got, tt.want, pathsOf(vs))
			}
		})
	}
}

// roleIncomeConfig is minimalConfig with one recipe "mine" listed by the role
// "farmer", whose group has the given number of households, and with a welfare
// window of exactly one tick (WelfareWindow 1, WindowCycles 1, Duration 1) and a
// single agent (Scale = Households, MinAgents 1), so the weight is households. The
// recipe is not sited, so its yield factor is 10 000 and a unit of Quantity sells
// ten units (MaxYieldBP / 10 000).
func roleIncomeConfig(households int64, price economysim.Price, outs ...economysim.Output) economysim.Config {
	c := minimalConfig()
	withProduct(&c, "gem", price, false)
	r := economysim.DefaultRecipe()
	r.ID, r.Duration = "mine", 1
	r.Outputs = outs
	c.Recipes = append(c.Recipes, r)
	c.Roles[0].Recipes = []string{"mine"}
	c.World.WelfareWindow, c.World.WindowCycles = 1, 1
	c.Groups[0].Households, c.Groups[0].Scale, c.World.MinAgents = households, households, 1
	return c
}

// TestValidate_RoleIncomeOutputSelection verifies which outputs of a recipe the
// income product counts: a discarded output and one above MaxQuantity are
// skipped, one of a single unit counts. With a window of 1 tick and a weight of
// 1, the product is 10 * Quantity * price: 10 * 10^9 * 922 337 203 =
// 9 223 372 030 * 10^9 fits int64 (9 223 372 036 854 775 807), one more unit
// adds 9 223 372 030 and does not.
//
// Requirement: FR-WEL-010
func TestValidate_RoleIncomeOutputSelection(t *testing.T) {
	t.Parallel()
	const fitPrice = 922_337_203
	path := []string{"roles", "farmer", "recipes"}
	huge := economysim.Output{Commodity: "gem", Quantity: economysim.MaxQuantity, CostShare: 10_000}
	tests := []struct {
		name       string
		households int64
		price      economysim.Price
		outs       []economysim.Output
		want       bool
	}{
		{"a discarded output is skipped", 1, economysim.MaxPrice,
			[]economysim.Output{{Commodity: "gem", Quantity: economysim.MaxQuantity, Discard: true}}, false},
		{"a discarded output beside a counted one is skipped", 1, economysim.MaxPrice,
			[]economysim.Output{{Commodity: "gem", Quantity: economysim.MaxQuantity, Discard: true}, {Commodity: "gem", Quantity: 1, CostShare: 10_000}}, false},
		{"MaxQuantity fits", 1, fitPrice, []economysim.Output{huge}, false},
		{"above MaxQuantity is skipped", 1, fitPrice,
			[]economysim.Output{{Commodity: "gem", Quantity: economysim.MaxQuantity + 1, CostShare: 10_000}}, false},
		// 10 * 1 * 10^9 * 10^9 = 10^19 > MaxInt64.
		{"one unit counts", economysim.MaxHouseholds, economysim.MaxPrice,
			[]economysim.Output{{Commodity: "gem", Quantity: 1, CostShare: 10_000}}, true},
		{"zero units add nothing", economysim.MaxHouseholds, economysim.MaxPrice,
			[]economysim.Output{{Commodity: "gem", Quantity: 0, CostShare: 10_000}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := roleIncomeConfig(tt.households, tt.price, tt.outs...)
			vs, _ := violations(c.Validate())
			if got := hasPath(vs, economysim.ErrOverflow, path) != nil; got != tt.want {
				t.Errorf("Validate(%s): ErrOverflow at %v = %t, want %t; paths: %v", tt.name, path, got, tt.want, pathsOf(vs))
			}
		})
	}
}

// TestValidate_TargetRatioWageOverflowEdges verifies the wage of the target
// ratio, ratio * TicksPerYear * basket / (10 000 * ExpectedWorkTicks), against
// int64. TicksPerYear is 365 and the basket MaxMoney (10^18): at ratio 10 000
// the wage is 365 * 10^18 / W, which is 9.125 * 10^18 at W 40 (fits) and
// 9.36 * 10^18 at W 39 (does not). The rule reads only a ratio from 10 000 to
// MaxBasisPoints, a work of 1 to TicksPerYear and a basket from 0 to MaxMoney.
//
// Requirement: FR-WEL-010
func TestValidate_TargetRatioWageOverflowEdges(t *testing.T) {
	t.Parallel()
	path := []string{"roles", "farmer", "target_ratio"}
	tests := []struct {
		name   string
		ratio  int64
		work   int64
		basket economysim.Money
		want   bool
	}{
		{"wage just under int64", 10_000, 40, economysim.MaxMoney, false},
		{"wage just over int64", 10_000, 39, economysim.MaxMoney, true},
		// 10^6 * 365 * 10^18 / (10^4 * 365) = 10^20.
		{"work of exactly TicksPerYear", economysim.MaxBasisPoints, 365, economysim.MaxMoney, true},
		{"work above TicksPerYear is not checked", economysim.MaxBasisPoints, 366, economysim.MaxMoney, false},
		{"work of 0 is not checked", economysim.MaxBasisPoints, 0, economysim.MaxMoney, false},
		{"ratio of exactly MaxBasisPoints", economysim.MaxBasisPoints, 1, economysim.MaxMoney, true},
		{"ratio above MaxBasisPoints is not checked", economysim.MaxBasisPoints + 1, 1, economysim.MaxMoney, false},
		{"ratio below 10 000 is not checked", 9_999, 1, economysim.MaxMoney, false},
		// 10^6 * 365 * 10^17 / 10^4 = 3.65 * 10^21.
		{"basket below MaxMoney", economysim.MaxBasisPoints, 1, economysim.MaxMoney / 10, true},
		{"basket above MaxMoney is not checked", economysim.MaxBasisPoints, 1, economysim.MaxMoney + 1, false},
		{"a negative basket is not checked", economysim.MaxBasisPoints, 1, -economysim.MaxMoney, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := minimalConfig()
			c.Roles[0].TargetRatio, c.Roles[0].ExpectedWorkTicks = tt.ratio, tt.work
			c.Regions[0].ReferenceBasket = tt.basket
			vs, _ := violations(c.Validate())
			if got := hasPath(vs, economysim.ErrOverflow, path) != nil; got != tt.want {
				t.Errorf("Validate(%s): ErrOverflow at %v = %t, want %t; paths: %v", tt.name, path, got, tt.want, pathsOf(vs))
			}
		})
	}
}

// TestValidate_MoneyAtStartSelectionEdges verifies which money figures count
// towards the money in existence at construction (spec-config.md 4.2). A warm
// row counts only with money in [0, MaxMoney]; otherwise the role's StartMoney
// counts, and only inside [0, MaxMoney / MaxWeight]. A pool counts only in
// [0, MaxMoney]. The case names the arithmetic of the total.
//
// Requirement: FR-CFG-010
func TestValidate_MoneyAtStartSelectionEdges(t *testing.T) {
	t.Parallel()
	const maxStart = economysim.MaxMoney / economysim.MaxWeight // 10^12
	setup := func(h int64, start economysim.Money, warm []economysim.WarmGroup, pools ...economysim.Money) func(c *economysim.Config) {
		return func(c *economysim.Config) {
			c.Groups[0].Households = h
			c.Roles[0].StartMoney = start
			ws := &economysim.WarmStart{Groups: warm}
			for _, m := range pools {
				ws.Pools = append(ws.Pools, economysim.WarmPool{Region: "home", Money: m})
			}
			c.WarmStart = ws
		}
	}
	warmRow := func(m economysim.Money) []economysim.WarmGroup {
		return []economysim.WarmGroup{{Region: "home", Role: "farmer", Money: m}}
	}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		want   bool // ErrOverflow at groups
	}{
		// 2e6 households of the role's 10^12 = 2e18 > MaxMoney when the warm row is not used.
		{"warm money 0 is used: 0 total", setup(2_000_000, maxStart, warmRow(0)), false},
		{"warm money -1 is not used: role money 2e18", setup(2_000_000, maxStart, warmRow(-1)), true},
		{"warm money 1 is used: 2e6 total", setup(2_000_000, maxStart, warmRow(1)), false},
		{"warm money MaxMoney is used: 1 household, 10^18", setup(1, 0, warmRow(economysim.MaxMoney)), false},
		{"warm money MaxMoney+1 is not used: role money 10^12 * 2e6", setup(2_000_000, maxStart, warmRow(economysim.MaxMoney+1)), true},
		// Role money: counted in [0, 10^12].
		{"role money 10^12 counts: 2e6 * 10^12", setup(2_000_000, maxStart, nil), true},
		{"role money 5e11 counts: 4e6 * 5e11", setup(4_000_000, maxStart/2, nil), true},
		{"role money 10^12+1 is skipped", setup(2_000_000, maxStart+1, nil), false},
		{"negative role money is skipped: not -10^21", setup(economysim.MaxHouseholds, -maxStart, nil), false},
		// Households of 1 count.
		{"one household counts: 1 * 10^18 + pool 1", setup(1, 0, warmRow(economysim.MaxMoney), 1), true},
		// Pools: counted in [0, MaxMoney].
		{"pool 1 counts: 10^18 + 1", setup(1, 0, warmRow(economysim.MaxMoney), 1), true},
		{"pool -3 is skipped: 3 * 333333333333333334 = 10^18 + 2", setup(3, 0, warmRow(333_333_333_333_333_334), -3), true},
		{"pool MaxMoney+1 is skipped", setup(1, 0, warmRow(0), economysim.MaxMoney+1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := minimalConfig()
			tt.mutate(&c)
			vs, _ := violations(c.Validate())
			if got := hasPath(vs, economysim.ErrOverflow, []string{"groups"}) != nil; got != tt.want {
				t.Errorf("Validate(%s): ErrOverflow at groups = %t, want %t; paths: %v", tt.name, got, tt.want, pathsOf(vs))
			}
		})
	}
}

// TestValidate_WarmProductsSelectionEdges verifies the edges of the warm
// start's derived products (spec-viability.md 8.7): a stock quantity of exactly
// MaxQuantity times a weight of 2 overflows; a wage of exactly MaxMoney counts;
// a wage row below 1 or above MaxMoney is ignored and leaves the earlier row of
// the region standing; an expected work of exactly MaxTicksPerYear is checked
// and one tick more is not.
//
// Requirement: FR-VIA-033
func TestValidate_WarmProductsSelectionEdges(t *testing.T) {
	t.Parallel()
	stockPath := []string{"warm_start", "groups", "home", "farmer", "stock", "grain", "quantity"}
	workPath := []string{"roles", "farmer", "expected_work_ticks"}
	wages := func(ws ...economysim.Money) func(c *economysim.Config) {
		return func(c *economysim.Config) {
			w := &economysim.WarmStart{}
			for _, m := range ws {
				w.Wages = append(w.Wages, economysim.SeedWage{Region: "home", Wage: m})
			}
			c.WarmStart = w
		}
	}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		path   []string
		want   bool
	}{
		{"stock MaxQuantity at weight 2: 2e9 > 1e9", func(c *economysim.Config) {
			c.Groups[0].Households, c.Groups[0].Scale, c.World.MinAgents = 2, 2, 1
			c.WarmStart = &economysim.WarmStart{Groups: []economysim.WarmGroup{{Region: "home", Role: "farmer",
				Stock: []economysim.Item{{Commodity: "grain", Quantity: economysim.MaxQuantity}}}}}
		}, stockPath, true},
		{"wage MaxMoney times 250 ticks", wages(economysim.MaxMoney), workPath, true},
		{"wage 1 after a wage of MaxMoney replaces it", wages(economysim.MaxMoney, 1), workPath, false},
		{"wage 0 after a wage of MaxMoney is ignored", wages(economysim.MaxMoney, 0), workPath, true},
		{"wage -1 after a wage of MaxMoney is ignored", wages(economysim.MaxMoney, -1), workPath, true},
		{"expected work of MaxTicksPerYear is checked", func(c *economysim.Config) {
			wages(economysim.MaxMoney)(c)
			c.Roles[0].ExpectedWorkTicks = economysim.MaxTicksPerYear
		}, workPath, true},
		{"expected work of MaxTicksPerYear+1 is not checked here", func(c *economysim.Config) {
			wages(economysim.MaxMoney)(c)
			c.Roles[0].ExpectedWorkTicks = economysim.MaxTicksPerYear + 1
		}, workPath, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := minimalConfig()
			tt.mutate(&c)
			vs, _ := violations(c.Validate())
			if got := hasPath(vs, economysim.ErrOverflow, tt.path) != nil; got != tt.want {
				t.Errorf("Validate(%s): ErrOverflow at %v = %t, want %t; paths: %v", tt.name, tt.path, got, tt.want, pathsOf(vs))
			}
		})
	}
}

// TestWarnings_BoundaryAndSourceEdges verifies the edges of the warnings of
// spec-config.md 7.4.2 that TestWarnings leaves open: the bounds of the
// comparisons (a window or share exactly at its limit draws nothing, one past
// does), the three sources of upkeep, the yield rows that run a recipe, and
// which recipes count as a source of a raw good. The configurations are read
// by Warnings only, which never fails on one that does not validate.
//
// Requirement: FR-CFG-018, FR-BEL-038, FR-AGT-060, FR-PRO-020, FR-PRO-040, FR-CAL-021, FR-SPN-039
func TestWarnings_BoundaryAndSourceEdges(t *testing.T) {
	t.Parallel()
	// The reference calendar has 365 days; a tick of 5 days over seasons that
	// sum to 365.
	seasons := func(a, b int64) func(c *economysim.Config) {
		return func(c *economysim.Config) {
			c.Calendar.DaysPerTick = 5
			c.Calendar.Seasons = []economysim.Season{{ID: "spring", Days: a}, {ID: "winter", Days: b}}
		}
	}
	// goodX adds a no-input recipe make_x that makes good_x, which no role uses.
	goodX := func(c *economysim.Config) {
		cm := economysim.DefaultCommodity()
		cm.ID, cm.ReferencePrice = "good_x", 100
		c.Commodities = append(c.Commodities, cm)
		r := economysim.DefaultRecipe()
		r.ID = "make_x"
		r.Outputs = []economysim.Output{{Commodity: "good_x", Quantity: 1, CostShare: 10_000}}
		c.Recipes = append(c.Recipes, r)
	}
	// oreY adds a good that only the world upkeep consumes and nothing makes,
	// and puts qty of it in the start stock of the cook.
	oreY := func(qty economysim.Quantity) func(c *economysim.Config) {
		return func(c *economysim.Config) {
			cm := economysim.DefaultCommodity()
			cm.ID, cm.ReferencePrice = "ore_y", 100
			c.Commodities = append(c.Commodities, cm)
			c.World.Upkeep = append(c.World.Upkeep, economysim.Item{Commodity: "ore_y", Quantity: 1})
			cook := roleOf(c, "cook")
			cook.StartStock = append(cook.StartStock, economysim.Item{Commodity: "ore_y", Quantity: qty})
		}
	}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   string
		path   []string
		fires  bool
	}{
		{
			name: "no_upkeep is not raised for a region extra upkeep",
			mutate: func(c *economysim.Config) {
				c.World.Upkeep = nil
				c.Regions[0].ExtraUpkeep = []economysim.Item{{Commodity: "staple", Quantity: 1}}
			},
			rule: "no_upkeep", path: []string{"world", "upkeep"}, fires: false,
		},
		{
			name:   "no_open_role is not raised with no roles at all",
			mutate: func(c *economysim.Config) { c.Roles, c.Groups = nil, nil },
			rule:   "no_open_role", path: []string{"roles"}, fires: false,
		},
		{
			name:   "cost_floor_at_full_cost fires above 10 000",
			mutate: func(c *economysim.Config) { c.World.CostFloor = 10_001 },
			rule:   "cost_floor_at_full_cost", path: []string{"world", "cost_floor"}, fires: true,
		},
		{
			name:   "season_shorter_than_tick is not raised for a season of exactly a tick",
			mutate: seasons(5, 360),
			rule:   "season_shorter_than_tick", path: []string{"calendar", "seasons"}, fires: false,
		},
		{
			name:   "season_shorter_than_tick fires for a season one day short",
			mutate: seasons(4, 361),
			rule:   "season_shorter_than_tick", path: []string{"calendar", "seasons"}, fires: true,
		},
		{
			// make_x keeps good_x first, then spoil only discards it: the good
			// is made, so it is not a discard-only good and is untraded.
			name: "commodity_untraded fires for a kept output that is also discarded",
			mutate: func(c *economysim.Config) {
				goodX(c)
				r := economysim.DefaultRecipe()
				r.ID = "spoil"
				r.Outputs = []economysim.Output{{Commodity: "good_x", Quantity: 1, Discard: true}}
				c.Recipes = append(c.Recipes, r)
			},
			rule: "commodity_untraded", path: []string{"commodities", "good_x"}, fires: true,
		},
		{
			name: "unrunnable_recipe fires when every season factor is 0",
			mutate: func(c *economysim.Config) {
				c.Yields[0] = economysim.YieldProfile{Region: "north", Recipe: "cut_fuel", Seasons: []economysim.SeasonYield{{Season: "spring", BP: 0}}}
			},
			rule: "unrunnable_recipe", path: []string{"recipes", "cut_fuel"}, fires: true,
		},
		{
			// The profile at north is 0, so cut_fuel does not start there.
			name:   "raw_without_source fires when the source has a profile of 0",
			mutate: func(c *economysim.Config) { c.Yields[0].All = 0 },
			rule:   "raw_without_source", path: []string{"commodities", "fuel"}, fires: true,
		},
		{
			// No profile: a recipe that is not Sited runs at its default.
			name: "raw_without_source is not raised for a source that is not Sited and has no profile",
			mutate: func(c *economysim.Config) {
				recipeOf(c, "cut_fuel").Sited = false
				c.Yields = slices.Delete(c.Yields, 0, 1)
			},
			rule: "raw_without_source", path: []string{"commodities", "fuel"}, fires: false,
		},
		{
			// No profile: a Sited recipe does not run.
			name:   "raw_without_source fires for a Sited source with no profile",
			mutate: func(c *economysim.Config) { c.Yields = slices.Delete(c.Yields, 0, 1) },
			rule:   "raw_without_source", path: []string{"commodities", "fuel"}, fires: true,
		},
		{
			name: "raw_without_source fires when the only recipe that makes the good has an input",
			mutate: func(c *economysim.Config) {
				recipeOf(c, "cut_fuel").Inputs = []economysim.Item{{Commodity: "seed", Quantity: 1}}
			},
			rule: "raw_without_source", path: []string{"commodities", "fuel"}, fires: true,
		},
		{
			// cut_fuel runs and makes fuel, which sorts below zz_raw: only an
			// output of the raw good itself is a source of it.
			name: "raw_without_source fires for a raw good only another good's source makes",
			mutate: func(c *economysim.Config) {
				cm := economysim.DefaultCommodity()
				cm.ID, cm.ReferencePrice, cm.Raw = "zz_raw", 100, true
				c.Commodities = append(c.Commodities, cm)
			},
			rule: "raw_without_source", path: []string{"commodities", "zz_raw"}, fires: true,
		},
		{
			name:   "upkeep_unsourced is not raised for an upkeep good in a start stock",
			mutate: oreY(1),
			rule:   "upkeep_unsourced", path: []string{"commodities", "ore_y"}, fires: false,
		},
		{
			name:   "upkeep_unsourced fires for an upkeep good whose start stock is 0",
			mutate: oreY(0),
			rule:   "upkeep_unsourced", path: []string{"commodities", "ore_y"}, fires: true,
		},
		{
			// 2 cycles of 15 ticks is the world window of 30.
			name:   "window_extended is not raised when the job is exactly the window",
			mutate: func(c *economysim.Config) { recipeOf(c, "grow_staple").Duration = 15 },
			rule:   "window_extended", path: []string{"roles", "grower", "recipes"}, fires: false,
		},
		{
			name:   "window_extended fires one tick past the window",
			mutate: func(c *economysim.Config) { recipeOf(c, "grow_staple").Duration = 16 },
			rule:   "window_extended", path: []string{"roles", "grower", "recipes"}, fires: true,
		},
		{
			// The longest window is 2 * 50 = 100; 25 * 100 is 2 500.
			name: "entry_share_window is not raised at exactly 2 500",
			mutate: func(c *economysim.Config) {
				recipeOf(c, "grow_staple").Duration = 50
				c.World.EntryShare = 25
			},
			rule: "entry_share_window", path: []string{"world", "entry_share"}, fires: false,
		},
		{
			name: "entry_share_window fires one share past 2 500",
			mutate: func(c *economysim.Config) {
				recipeOf(c, "grow_staple").Duration = 50
				c.World.EntryShare = 26
			},
			rule: "entry_share_window", path: []string{"world", "entry_share"}, fires: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			got := c.Warnings()
			if has := hasWarning(got, tt.rule, tt.path...); has != tt.fires {
				t.Errorf("Warnings(%s) has %s at %v = %v, want %v; warnings: %v", tt.name, tt.rule, tt.path, has, tt.fires, got)
			}
		})
	}
}

// TestWarnings_RoleStockMoneyAndLockstepEdges verifies the edges of the
// role-level warnings of spec-config.md 7.4.2 that TestWarnings leaves open:
// cold_start_lockstep (a recipe of Duration 0 is not long, a job row of another
// role does not count, a group whose shape is refused is judged by its
// households), strategy_default_unused_trades (only the default strategy),
// start_stock_below_target (a stock exactly at the upkeep or at the tool is
// enough, a tool does not stand in for a missing input), reserve_exceeds_money
// (a cost that is not computable draws nothing) and tool_unaffordable (money
// above the price of the tool draws nothing). Warnings reads a configuration
// that does not validate, so the cases may leave the valid range.
//
// Requirement: FR-CFG-018, FR-AGT-060, FR-SKL-004
func TestWarnings_RoleStockMoneyAndLockstepEdges(t *testing.T) {
	t.Parallel()
	// withStock puts the digger on the single recipe make_tool_a, which needs 2
	// refined and the tool tool_b, with a stock that covers one tick of upkeep
	// (2 staple, 1 prepared, 1 fuel) exactly, plus the given refined and tool.
	withStock := func(refined, tool economysim.Quantity) func(c *economysim.Config) {
		return func(c *economysim.Config) {
			recipeOf(c, "make_tool_a").Requires = []economysim.Tool{{Commodity: "tool_b", WearPerUse: 1}}
			d := roleOf(c, "digger")
			d.Recipes = []string{"make_tool_a"}
			d.StartStock = []economysim.Item{
				{Commodity: "fuel", Quantity: 1}, {Commodity: "prepared", Quantity: 1}, {Commodity: "staple", Quantity: 2},
				{Commodity: "refined", Quantity: refined}, {Commodity: "tool_b", Quantity: tool},
			}
		}
	}
	// shape puts the grower on a group whose shape is refused (MinAgents 0), so
	// only its households decide whether the first jobs start together.
	shape := func(households int64) func(c *economysim.Config) {
		return func(c *economysim.Config) {
			c.World.MinAgents = 0
			c.Groups[3].Households = households
		}
	}
	// upkeepOf sets the world upkeep and the money of the digger.
	upkeepOf := func(money economysim.Money, items ...economysim.Item) func(c *economysim.Config) {
		return func(c *economysim.Config) {
			c.World.Upkeep = items
			roleOf(c, "digger").StartMoney = money
		}
	}
	lockstep := []string{"roles", "grower", "recipes"}
	trades := []string{"roles", "digger", "trades"}
	stock := []string{"roles", "digger", "start_stock"}
	reserve := []string{"roles", "digger", "start_money"}
	withStrategy := func(s string) func(c *economysim.Config) {
		return func(c *economysim.Config) {
			d := roleOf(c, "digger")
			d.Trades, d.Strategy = []string{"fuel"}, s
		}
	}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   string
		path   []string
		fires  bool
	}{
		{"lockstep: recipes of Duration 0 are not long", func(c *economysim.Config) {
			for _, id := range roleOf(c, "grower").Recipes {
				recipeOf(c, id).Duration = 0
			}
		}, "cold_start_lockstep", lockstep, false},
		{"lockstep: a job row of an earlier role does not count", func(c *economysim.Config) {
			c.WarmStart.Groups[0].Jobs = []economysim.JobSpec{{Recipe: "prepare", InProgressBP: 100}}
		}, "cold_start_lockstep", lockstep, true},
		{"lockstep: a refused shape of 2 households is many", shape(2), "cold_start_lockstep", lockstep, true},
		{"lockstep: a refused shape of 1 household is not many", shape(1), "cold_start_lockstep", lockstep, false},
		{"lockstep: a refused shape of 0 households is not many", shape(0), "cold_start_lockstep", lockstep, false},

		{"trades: an earlier strategy name draws nothing", withStrategy("aaa"), "strategy_default_unused_trades", trades, false},
		{"trades: a later strategy name draws nothing", withStrategy("zzz"), "strategy_default_unused_trades", trades, false},
		{"trades: the empty strategy name draws nothing", withStrategy(""), "strategy_default_unused_trades", trades, false},

		// 2 staple, 1 prepared and 1 fuel of stock against the same upkeep: a stock
		// exactly at the upkeep is not below it.
		{"stock: exactly one tick of upkeep is enough", withStock(2, 1), "start_stock_below_target", stock, false},
		{"stock: one tool of 1 required starts the recipe", withStock(2, 1), "start_stock_below_target", stock, false},
		{"stock: two tools start the recipe", withStock(2, 2), "start_stock_below_target", stock, false},
		{"stock: no tool does not start the recipe", withStock(2, 0), "start_stock_below_target", stock, true},
		{"stock: a tool does not stand in for the input", withStock(0, 1), "start_stock_below_target", stock, true},

		// A cost that cannot be computed is not compared with the money: a negative
		// StartMoney is below the 0 that stands in for it.
		{"reserve: an undeclared upkeep good draws nothing", upkeepOf(-1, economysim.Item{Commodity: "ghost", Quantity: 1}), "reserve_exceeds_money", reserve, false},
		{"reserve: a reserve that overflows draws nothing", func(c *economysim.Config) {
			upkeepOf(-1, economysim.Item{Commodity: "staple", Quantity: 1})(c)
			c.World.ReserveTicks = math.MaxInt64
		}, "reserve_exceeds_money", reserve, false},
		{"reserve: a line that overflows draws nothing", upkeepOf(0,
			economysim.Item{Commodity: "staple", Quantity: 1},
			economysim.Item{Commodity: "fuel", Quantity: math.MaxInt64}), "reserve_exceeds_money", reserve, false},
		{"reserve: money below the cost of 96 draws it", upkeepOf(287, economysim.Item{Commodity: "staple", Quantity: 1}), "reserve_exceeds_money", reserve, true},

		{"tool: money above the price of the tool draws nothing", func(c *economysim.Config) {
			roleOf(c, "cook").StartMoney = 17_106
		}, "tool_unaffordable", []string{"roles", "cook", "start_money"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			got := c.Warnings()
			if has := hasWarning(got, tt.rule, tt.path...); has != tt.fires {
				t.Errorf("Warnings(%s) has %s at %v = %v, want %v; warnings: %v", tt.name, tt.rule, tt.path, has, tt.fires, got)
			}
		})
	}
}
