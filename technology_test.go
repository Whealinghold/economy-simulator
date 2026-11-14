package economysim_test

import (
	"errors"
	"slices"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the technology module's configuration surface (spec-technology.md 9,
// 10): the failing case of each rule of 10.1, their order, the overflow term and the
// warnings of 10.2 to 10.4. The hash is in technology_hash_test.go and the loader in
// config/technology_test.go.

// technologyConfig returns referenceConfig with the technology module on: a field
// farming (grow_staple and gather_seed, learning only) and a field forestry
// (cut_fuel, a trend only). Every recipe is on a role's list, so none of the
// module's warnings is raised.
func technologyConfig() economysim.Config {
	c := referenceConfig()
	c.Modules.Technology = &economysim.Technology{Fields: []economysim.TechField{
		{ID: "farming", Recipes: []string{"gather_seed", "grow_staple"}, CapBP: 3_000, Base: 1_000, DoublingBP: 300, Label: "Farming"},
		{ID: "forestry", Recipes: []string{"cut_fuel"}, CapBP: 2_000, TrendBP: 100},
	}}
	return c
}

// technologyFieldCases are the failing cases of the rows of the technology module in
// the parameter-owner table (spec-technology.md 9.2, 10.1, FR-TEC-002, FR-TEC-003):
// each breaks one field of the valid configuration of technologyConfig.
// buildFieldCases appends them to the table that the registry test reads.
func technologyFieldCases() []fieldCase {
	tech := func(field, name string, rule error, p []string, value, bound string, set func(m *economysim.Technology)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				*c = technologyConfig()
				set(c.Modules.Technology)
			}}
	}
	at := func(i int, key ...string) []string {
		return append([]string{"modules", "technology", "fields[" + strconv.Itoa(i) + "]"}, key...)
	}
	farming := func(f func(*economysim.TechField)) func(*economysim.Technology) {
		return func(m *economysim.Technology) { f(&m.Fields[0]) }
	}
	var cs []fieldCase
	cs = append(cs, technologyListCases(tech, at)...)
	cs = append(cs, technologyRateCases(tech, at, farming)...)
	return append(cs, technologyPairCases(tech, at, farming)...)
}

// technologyCase builds one failing case.
type technologyCase func(field, name string, rule error, p []string, value, bound string, set func(m *economysim.Technology)) fieldCase

// technologyListCases are the cases of the field list, the IDs and the recipes.
func technologyListCases(tech technologyCase, at func(int, ...string) []string) []fieldCase {
	return []fieldCase{
		tech("Technology.Fields", "more than MaxTechFields", economysim.ErrTooMany, []string{"modules", "technology", "fields"}, "65", "64",
			func(m *economysim.Technology) {
				m.Fields = nil
				for i := range economysim.MaxTechFields + 1 {
					m.Fields = append(m.Fields, economysim.TechField{ID: "f" + strconv.Itoa(i), Recipes: []string{"cut_fuel"}, CapBP: 1})
				}
			}),
		tech("TechField.ID", "upper case letter", economysim.ErrInvalidID, at(0, "id"), "Farming", "",
			func(m *economysim.Technology) { m.Fields[0].ID = "Farming" }),
		tech("TechField.ID", "twice", economysim.ErrDuplicateID, at(1, "id"), "farming", "",
			func(m *economysim.Technology) { m.Fields[1].ID = "farming" }),
		tech("TechField.Recipes", "empty", economysim.ErrTechShape, at(0, "recipes"), "0", "1",
			func(m *economysim.Technology) { m.Fields[0].Recipes = nil }),
		tech("TechField.Recipes", "more than MaxFieldRecipes", economysim.ErrTooMany, at(0, "recipes"), "65", "64",
			func(m *economysim.Technology) { m.Fields[0].Recipes = slices.Repeat([]string{"cut_fuel"}, 65) }),
		tech("TechField.Recipes", "a recipe named twice in one field", economysim.ErrDuplicateID, at(0, "recipes[1]"), "gather_seed", "",
			func(m *economysim.Technology) { m.Fields[0].Recipes = []string{"gather_seed", "gather_seed"} }),
		tech("TechField.Recipes", "a recipe that is not defined", economysim.ErrUnknownRecipe, at(0, "recipes[0]"), "nowhere", "",
			func(m *economysim.Technology) { m.Fields[0].Recipes = []string{"nowhere"} }),
		tech("TechField.Recipes", "a recipe in two fields", economysim.ErrTechRecipeOverlap, at(1, "recipes[1]"), "grow_staple", "farming",
			func(m *economysim.Technology) { m.Fields[1].Recipes = []string{"cut_fuel", "grow_staple"} }),
	}
}

// technologyRateCases are the bound cases of the numbers of a field.
func technologyRateCases(tech technologyCase, at func(int, ...string) []string, farming func(func(*economysim.TechField)) func(*economysim.Technology)) []fieldCase {
	return []fieldCase{
		tech("TechField.CapBP", "0", economysim.ErrTechShape, at(0, "cap_bp"), "0", "1",
			farming(func(f *economysim.TechField) { f.CapBP = 0 })),
		tech("TechField.CapBP", "above MaxGainBP", economysim.ErrOutOfBounds, at(0, "cap_bp"), "100001", "100000",
			farming(func(f *economysim.TechField) { f.CapBP = economysim.MaxGainBP + 1 })),
		tech("TechField.CapBP", "below 0", economysim.ErrOutOfBounds, at(0, "cap_bp"), "-1", "100000",
			farming(func(f *economysim.TechField) { f.CapBP = -1 })),
		tech("TechField.Base", "above MaxTechBase", economysim.ErrOutOfBounds, at(0, "base"), "1000000000001", "1000000000000",
			farming(func(f *economysim.TechField) { f.Base = economysim.MaxTechBase + 1 })),
		tech("TechField.DoublingBP", "above MaxDoublingBP", economysim.ErrOutOfBounds, at(0, "doubling_bp"), "10001", "10000",
			farming(func(f *economysim.TechField) { f.DoublingBP = economysim.MaxDoublingBP + 1 })),
		tech("TechField.TrendBP", "above MaxTechTrendBP", economysim.ErrOutOfBounds, at(1, "trend_bp"), "10001", "10000",
			func(m *economysim.Technology) { m.Fields[1].TrendBP = economysim.MaxTechTrendBP + 1 }),
		tech("TechField.DiscoveryBP", "above MaxDiscoveryBP", economysim.ErrOutOfBounds, at(0, "discovery_bp"), "100001", "100000",
			farming(func(f *economysim.TechField) { f.DiscoveryBP, f.StepBP = economysim.MaxDiscoveryBP+1, 10 })),
		tech("TechField.StepBP", "above MaxStepBP", economysim.ErrOutOfBounds, at(0, "step_bp"), "100001", "100000",
			farming(func(f *economysim.TechField) { f.DiscoveryBP, f.StepBP = 500, economysim.MaxStepBP+1 })),
		tech("TechField.CatchUpBP", "above MaxCatchUpBP", economysim.ErrOutOfBounds, at(0, "catch_up_bp"), "10001", "10000",
			farming(func(f *economysim.TechField) { f.CatchUpBP, f.Reach = economysim.MaxCatchUpBP+1, 100 })),
		tech("TechField.Reach", "above MaxDistance", economysim.ErrOutOfBounds, at(0, "reach"), "1000000001", "1000000000",
			farming(func(f *economysim.TechField) { f.CatchUpBP, f.Reach = 500, economysim.MaxDistance+1 })),
	}
}

// technologyPairCases are the cases of 3.1.2: a value an on mechanism needs and one
// an off mechanism must not use.
func technologyPairCases(tech technologyCase, at func(int, ...string) []string, farming func(func(*economysim.TechField)) func(*economysim.Technology)) []fieldCase {
	return []fieldCase{
		tech("TechField.Base", "0 with learning on", economysim.ErrTechShape, at(0, "base"), "0", "1",
			farming(func(f *economysim.TechField) { f.Base = 0 })),
		tech("TechField.Base", "set with learning off", economysim.ErrTechShape, at(0, "base"), "1000", "0",
			farming(func(f *economysim.TechField) { f.DoublingBP = 0 })),
		tech("TechField.StepBP", "0 with discovery on", economysim.ErrTechShape, at(0, "step_bp"), "0", "1",
			farming(func(f *economysim.TechField) { f.DiscoveryBP = 500 })),
		tech("TechField.StepBP", "set with discovery off", economysim.ErrTechShape, at(0, "step_bp"), "5", "0",
			farming(func(f *economysim.TechField) { f.StepBP = 5 })),
		tech("TechField.Reach", "0 with diffusion on", economysim.ErrTechShape, at(0, "reach"), "0", "1",
			farming(func(f *economysim.TechField) { f.CatchUpBP = 500 })),
		tech("TechField.Reach", "set with diffusion off", economysim.ErrTechShape, at(0, "reach"), "100", "0",
			farming(func(f *economysim.TechField) { f.Reach = 100 })),
		tech("TechField.Label", "over MaxLabelBytes", economysim.ErrOutOfBounds, at(0, "label"), "257", "256",
			farming(func(f *economysim.TechField) { f.Label = string(make([]byte, economysim.MaxLabelBytes+1)) })),
	}
}

// TestValidateTechnology_ConfigIsValid verifies that the configuration the technology
// tests start from validates and raises none of the module's warnings, and that the
// empty module is valid: it is on, has no field and does nothing (FR-TEC-001).
//
// Requirement: FR-TEC-002, FR-TEC-029
//
// Guard: the field cases start from technologyConfig; if it did not validate, each
// case would pass for the wrong reason.
func TestValidateTechnology_ConfigIsValid(t *testing.T) {
	t.Parallel()
	c := technologyConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("technologyConfig().Validate() = %v, want nil", err)
	}
	for _, w := range c.Warnings() {
		if len(w.Path) > 1 && w.Path[1] == "technology" {
			t.Errorf("technologyConfig().Warnings() has %v, want none of the module's", w)
		}
	}
	empty := referenceConfig()
	empty.Modules.Technology = &economysim.Technology{}
	if err := empty.Validate(); err != nil {
		t.Errorf("the empty module .Validate() = %v, want nil", err)
	}
}

// TestValidateTechnology_Errors verifies that each rule of spec-technology.md 10.1 has
// a configuration that breaks only that rule: the sentinel by errors.Is, the path of
// the offending field by its index, and the value and bound.
//
// Requirement: FR-TEC-002, FR-TEC-003, FR-CFG-085
func TestValidateTechnology_Errors(t *testing.T) {
	t.Parallel()
	for _, tt := range technologyFieldCases() {
		t.Run(tt.field+" "+tt.name, func(t *testing.T) {
			t.Parallel()
			var c economysim.Config
			tt.mutate(&c)
			err := c.Validate()
			if !errors.Is(err, tt.rules[0]) {
				t.Fatalf("Validate() = %v, want %v", err, tt.rules[0])
			}
			if !errorAt(err, tt) {
				t.Errorf("Validate() = %v, want %v at %v with value %q and bound %q", err, tt.rules[0], tt.paths[0], tt.value, tt.bound)
			}
		})
	}
}

// TestValidateTechnology_PhaseOrder verifies that a phase that finds an error ends
// the run, so the first error is the same for the same mistake: a field with a shape
// error and an out-of-bounds number reports the shape only, and a field with a bound
// error and an unknown recipe reports the bound only.
//
// Requirement: FR-TEC-002, FR-TEC-003
func TestValidateTechnology_PhaseOrder(t *testing.T) {
	t.Parallel()
	c := technologyConfig()
	c.Modules.Technology.Fields[0].Recipes = nil
	c.Modules.Technology.Fields[0].TrendBP = economysim.MaxTechTrendBP + 1
	err := c.Validate()
	if !errors.Is(err, economysim.ErrTechShape) || errors.Is(err, economysim.ErrOutOfBounds) {
		t.Errorf("shape and bound mistakes: Validate() = %v, want ErrTechShape only", err)
	}
	c = technologyConfig()
	c.Modules.Technology.Fields[0].Recipes = []string{"nowhere"}
	c.Modules.Technology.Fields[0].TrendBP = economysim.MaxTechTrendBP + 1
	err = c.Validate()
	if !errors.Is(err, economysim.ErrOutOfBounds) || errors.Is(err, economysim.ErrUnknownRecipe) {
		t.Errorf("bound and reference mistakes: Validate() = %v, want ErrOutOfBounds only", err)
	}
}

// TestValidate_OverflowReadsMaxTechBP verifies the overflow term of
// spec-technology.md 9.5: a recipe in a field has its largest yield factor multiplied
// by MaxTechBP over 10 000, so a product that is at the bound without the module is
// above it with the recipe in a field, and a recipe in no field, or the module off, is
// as it was.
//
// The arithmetic (10 000 households in agents of weight 1 000): weight 1 000 * 10^6
// units = 10^9; the largest yield of a Sited recipe with an entry of 1 000 bp is
// 1 000 * MaxYieldBP / 10 000 = 10 000, so the product is 10^9 * 10 000 / 10 000 =
// 10^9, which is MaxQuantity and passes; with the factor 110 000 / 10 000 it is
// 1.1 * 10^10 and fails.
//
// Requirement: FR-TEC-020, NFR-TEC-005
func TestValidate_OverflowReadsMaxTechBP(t *testing.T) {
	t.Parallel()
	outPath := []string{"recipes", "mine", "outputs", "gem", "quantity"}
	build := func(tech *economysim.Technology) economysim.Config {
		c := minimalConfig()
		c.Groups[0].Households, c.Groups[0].Scale = 10_000, 10_000
		withProduct(&c, "gem", 1_000, false)
		r := economysim.DefaultRecipe()
		r.ID, r.Sited = "mine", true
		r.Outputs = []economysim.Output{{Commodity: "gem", Quantity: 1_000_000, CostShare: 10_000}}
		c.Recipes = append(c.Recipes, r)
		c.Yields = []economysim.YieldProfile{{Region: "home", Recipe: "mine", HasAll: true, All: 1_000}}
		c.Roles[0].Recipes = []string{"mine"}
		c.Modules.Technology = tech
		return c
	}
	field := func(recipe string) *economysim.Technology {
		return &economysim.Technology{Fields: []economysim.TechField{{ID: "f", Recipes: []string{recipe}, CapBP: 1}}}
	}
	tests := []struct {
		name string
		tech *economysim.Technology
		want bool
	}{
		{"the module off", nil, false},
		{"the module on with no field", &economysim.Technology{}, false},
		{"another recipe in the field", field("other"), false},
		{"the recipe in a field", field("mine"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := build(tt.tech)
			other := economysim.DefaultRecipe()
			other.ID = "other"
			c.Recipes = append(c.Recipes, other)
			vs, _ := violations(c.Validate())
			if got := hasPath(vs, economysim.ErrOverflow, outPath) != nil; got != tt.want {
				t.Errorf("Validate(%s): ErrOverflow at %v = %t, want %t; paths: %v", tt.name, outPath, got, tt.want, pathsOf(vs))
			}
		})
	}
}

// TestWarnings_Technology verifies each warning of spec-technology.md 10.2 to 10.4
// fires on its condition at the path of the field, is silent on the case beside it,
// and never fails Validate (FR-TEC-029).
//
// Requirement: FR-TEC-029
func TestWarnings_Technology(t *testing.T) {
	t.Parallel()
	farming := []string{"modules", "technology", "fields[0]"}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   string
		want   bool
	}{
		{"every rate off", func(c *economysim.Config) {
			c.Modules.Technology.Fields[0] = economysim.TechField{ID: "farming", Recipes: []string{"grow_staple"}, CapBP: 5}
		}, "field_inert", true},
		{"one rate on", func(c *economysim.Config) {}, "field_inert", false},
		{"a recipe no role lists", func(c *economysim.Config) {
			r := economysim.DefaultRecipe()
			r.ID = "idle"
			c.Recipes = append(c.Recipes, r)
			c.Modules.Technology.Fields[0].Recipes = []string{"idle"}
		}, "field_unused", true},
		{"one recipe of two is listed", func(c *economysim.Config) {
			r := economysim.DefaultRecipe()
			r.ID = "idle"
			c.Recipes = append(c.Recipes, r)
			c.Modules.Technology.Fields[0].Recipes = []string{"grow_staple", "idle"}
		}, "field_unused", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := technologyConfig()
			tt.mutate(&c)
			if got := hasWarning(c.Warnings(), tt.rule, farming...); got != tt.want {
				t.Errorf("Warnings() has %s at %v = %v, want %v", tt.rule, farming, got, tt.want)
			}
		})
	}
}

// TestWarnings_TechnologyContact verifies diffusion_no_contact: with diffusion on it
// fires when the world has one ordinary region, or no pair of ordinary regions has a
// path shorter than the reach, and is silent when one does. The distance is strict
// (a path equal to the reach has weight 0), an outside node is not an ordinary
// region, and the check is the shortest path over the edges as written.
//
// Requirement: FR-TEC-029
func TestWarnings_TechnologyContact(t *testing.T) {
	t.Parallel()
	farming := []string{"modules", "technology", "fields[0]"}
	two := func(distance, reach int64) economysim.Config {
		c := technologyConfig()
		other := c.Regions[0]
		other.ID = "other"
		c.Regions = append(c.Regions, other)
		c.Edges = []economysim.Edge{{From: c.Regions[0].ID, To: "other", Distance: distance}}
		f := &c.Modules.Technology.Fields[0]
		f.CatchUpBP, f.Reach = 500, reach
		return c
	}
	one := technologyConfig()
	one.Modules.Technology.Fields[0].CatchUpBP, one.Modules.Technology.Fields[0].Reach = 500, 100
	tests := []struct {
		name string
		c    economysim.Config
		want bool
	}{
		{"one region", one, true},
		{"two regions, a path below the reach", two(99, 100), false},
		{"two regions, a path equal to the reach", two(100, 100), true},
		{"two regions, no edge", func() economysim.Config { c := two(1, 100); c.Edges = nil; return c }(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := hasWarning(tt.c.Warnings(), "diffusion_no_contact", farming...); got != tt.want {
				t.Errorf("Warnings() has diffusion_no_contact = %v, want %v", got, tt.want)
			}
		})
	}
	off := two(100, 100)
	off.Modules.Technology.Fields[0].CatchUpBP, off.Modules.Technology.Fields[0].Reach = 0, 0
	if hasWarning(off.Warnings(), "diffusion_no_contact", farming...) {
		t.Errorf("Warnings() has diffusion_no_contact with diffusion off, want none")
	}
}

// TestWarnings_TechnologyStock verifies technology_past_maximum_yield on the worked
// example of spec-technology.md 16.3: a logistic stock with a linear abundance curve
// and a plan share of 6 000, the population module on, and a field whose recipe draws
// on it. The inequality a(5 000) * (10 000 + CapBP) >= a(6 000) * 10 000 is 5 000 *
// (10 000 + CapBP) >= 6 000 * 10 000, so a cap of 1 999 is silent and 2 000 fires.
// It is silent with the population module off, for a recipe that draws on no logistic
// stock and for a flat curve.
//
// Requirement: FR-TEC-030
func TestWarnings_TechnologyStock(t *testing.T) {
	t.Parallel()
	forestry := []string{"modules", "technology", "fields[1]"}
	with := func(capBP int64, mutate func(c *economysim.Config)) economysim.Config {
		c := resourcesConfig()
		withPopulation(&c)
		c.Modules.Technology = technologyConfig().Modules.Technology
		stand := resourceOf(&c, "stand")
		stand.PlanShareBP = 6_000
		stand.Abundance = []economysim.AbundanceKnot{{Share: 0, Factor: 0}, {Share: 10_000, Factor: 10_000}}
		c.Modules.Technology.Fields[1].CapBP = capBP
		mutate(&c)
		return c
	}
	tests := []struct {
		name   string
		capBP  int64
		mutate func(c *economysim.Config)
		want   bool
	}{
		{"a cap of 1 999", 1_999, func(*economysim.Config) {}, false},
		{"a cap of 2 000", 2_000, func(*economysim.Config) {}, true},
		{"the population module off", 2_000, func(c *economysim.Config) { c.Modules.Population = nil }, false},
		{"a flat curve", 2_000, func(c *economysim.Config) { resourceOf(c, "stand").Abundance = nil }, false},
		{"a refill stock", 2_000, func(c *economysim.Config) { resourceOf(c, "stand").Shape = economysim.ShapeRefill }, false},
		{"a recipe that draws nothing", 2_000, func(c *economysim.Config) { recipeOf(c, "cut_fuel").Draws = nil }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := with(tt.capBP, tt.mutate)
			if got := hasWarning(c.Warnings(), "technology_past_maximum_yield", forestry...); got != tt.want {
				t.Errorf("Warnings() has technology_past_maximum_yield = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestWarnings_TechnologyTrend verifies trend_outpaces_schedule on the worked example
// of spec-technology.md 16.3: with the default schedules (r* 12 000, dR 400, dX 8 000,
// the highest birth knot 16 000) the condition TrendBP * dX > (knot_max - r*) * dR is
// TrendBP * 8 000 > 4 000 * 400 = 1 600 000, so a trend of 200 is silent and 201
// fires. It is silent with the population module off and with the trend off.
//
// Requirement: FR-TEC-031
func TestWarnings_TechnologyTrend(t *testing.T) {
	t.Parallel()
	forestry := []string{"modules", "technology", "fields[1]"}
	tests := []struct {
		name  string
		trend int64
		off   bool
		want  bool
	}{
		{"a trend of 200", 200, false, false},
		{"a trend of 201", 201, false, true},
		{"the population module off", 201, true, false},
		{"no trend", 0, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := technologyConfig()
			withPopulation(&c)
			c.Modules.Technology.Fields[1].TrendBP = tt.trend
			if tt.trend == 0 {
				c.Modules.Technology.Fields[1].DiscoveryBP, c.Modules.Technology.Fields[1].StepBP = 10, 10
			}
			if tt.off {
				c.Modules.Population = nil
			}
			if got := hasWarning(c.Warnings(), "trend_outpaces_schedule", forestry...); got != tt.want {
				t.Errorf("Warnings() has trend_outpaces_schedule = %v, want %v", got, tt.want)
			}
		})
	}
}
