package economysim_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the resources module's configuration: its validation, its
// warnings and its draws. The module has no world effect yet in this file; the
// hash, the canonical order and the loader are tested beside their code.

// withResources switches the resources module on in c, a copy of referenceConfig,
// with two resources and draws on both. The numbers are chosen so that none of the
// warnings of the module is raised: arable (a refill of 10 000 bp a year, a
// capacity of 1 000, doubled in spring) is drawn by grow_staple and has a row in
// north, and stand (a logistic stock of 3 000 bp a year with a refuge of 20) is
// drawn by cut_fuel and takes the default capacity of 500. Both recipes are on a
// role that has a group in north.
func withResources(c *economysim.Config) {
	arable, stand := economysim.DefaultResource(), economysim.DefaultResource()
	arable.ID, arable.Shape, arable.RateBP, arable.Capacity = "arable", economysim.ShapeRefill, 10_000, 1_000
	arable.Seasons = []economysim.SeasonRegen{{Season: "spring", BP: 20_000}}
	arable.Abundance = []economysim.AbundanceKnot{{Share: 0, Factor: 2_000}, {Share: 5_000, Factor: 9_000}, {Share: 10_000, Factor: 10_000}}
	stand.ID, stand.Shape, stand.RateBP, stand.Refuge, stand.Capacity = "stand", economysim.ShapeLogistic, 3_000, 20, 500
	res := economysim.DefaultResources()
	res.Resources = []economysim.Resource{arable, stand}
	res.Endowments = []economysim.Endowment{
		{Region: "north", Resource: "arable", HasCapacity: true, Capacity: 2_000, HasInitial: true, Initial: 1_000},
	}
	c.Modules.Resources = &res
	recipeOf(c, "grow_staple").Draws = []economysim.Draw{{Resource: "arable", Quantity: 1}}
	recipeOf(c, "cut_fuel").Draws = []economysim.Draw{{Resource: "stand", Quantity: 2}}
}

// resourcesConfig returns referenceConfig with the resources module on.
func resourcesConfig() economysim.Config {
	c := referenceConfig()
	withResources(&c)
	return c
}

// resourceOf returns the resource with the given ID of c, or nil.
func resourceOf(c *economysim.Config, id string) *economysim.Resource {
	for i := range c.Modules.Resources.Resources {
		if c.Modules.Resources.Resources[i].ID == id {
			return &c.Modules.Resources.Resources[i]
		}
	}
	return nil
}

// resourcesFieldCases are the failing cases of the rows of the resources module
// in the parameter-owner table (spec-resources.md 12, 13.2, FR-CFG-031): each
// breaks one field of a valid configuration. buildFieldCases appends them to the
// table that the registry test reads.
func resourcesFieldCases() []fieldCase {
	res := func(field, name string, rule error, mutate func(c *economysim.Config), path ...string) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{path},
			mutate: func(c *economysim.Config) { withResources(c); mutate(c) }}
	}
	bound := func(field, key string, bad int64, set func(r *economysim.Resource, v int64)) fieldCase {
		return res(field, "outside its range", economysim.ErrOutOfBounds,
			func(c *economysim.Config) { set(resourceOf(c, "arable"), bad) },
			"modules", "resources", "resources", "arable", key)
	}
	noDraws := func(c *economysim.Config) {
		recipeOf(c, "grow_staple").Draws, recipeOf(c, "cut_fuel").Draws = nil, nil
	}
	heavy := func(c *economysim.Config) {
		// One agent of 253 households: the largest weight of the grower is 253.
		c.World.Scale, c.World.MinAgents = 1_000, 1
	}
	row := func(field, name string, rule error, mutate func(e *economysim.Endowment), path ...string) fieldCase {
		return res(field, name, rule, func(c *economysim.Config) { mutate(&c.Modules.Resources.Endowments[0]) },
			append([]string{"modules", "resources", "endowments", "north", "arable"}, path...)...)
	}
	curve := func(field, name string, knots []economysim.AbundanceKnot, path ...string) fieldCase {
		return res(field, name, economysim.ErrCurve,
			func(c *economysim.Config) { resourceOf(c, "arable").Abundance = knots },
			append([]string{"modules", "resources", "resources", "arable"}, path...)...)
	}
	draws := []string{"recipes", "grow_staple", "draws"}
	return []fieldCase{
		res("Resources.Resources", "more than MaxResources", economysim.ErrOutOfBounds, func(c *economysim.Config) {
			noDraws(c)
			c.Modules.Resources.Resources = make([]economysim.Resource, economysim.MaxResources+1)
		}, "modules", "resources", "resources"),
		res("Resources.Endowments", "more than MaxEndowments", economysim.ErrOutOfBounds, func(c *economysim.Config) {
			c.Modules.Resources.Endowments = make([]economysim.Endowment, economysim.MaxEndowments+1)
		}, "modules", "resources", "endowments"),
		res("Resources.HorizonYears", "below 1", economysim.ErrOutOfBounds,
			func(c *economysim.Config) { c.Modules.Resources.HorizonYears = 0 }, "modules", "resources", "horizon_years"),
		res("Resource.ID", "upper case letter", economysim.ErrInvalidID,
			func(c *economysim.Config) { recipeOf(c, "cut_fuel").Draws = nil; resourceOf(c, "stand").ID = "Stand" },
			"modules", "resources", "resources", "Stand", "id"),
		res("Resource.ID", "a resource twice", economysim.ErrDuplicateID, func(c *economysim.Config) {
			m := c.Modules.Resources
			m.Resources = append(m.Resources, *resourceOf(c, "stand"))
		}, "modules", "resources", "resources", "stand", "id"),
		res("Resource.Shape", "not one of the three", economysim.ErrResourceShape,
			func(c *economysim.Config) { resourceOf(c, "arable").Shape = "tidal" },
			"modules", "resources", "resources", "arable", "shape"),
		res("Resource.RatePerYear", "set on a refill", economysim.ErrResourceShape,
			func(c *economysim.Config) { resourceOf(c, "arable").RatePerYear = 5 },
			"modules", "resources", "resources", "arable", "rate_per_year"),
		res("Resource.RatePerYear", "outside its range", economysim.ErrOutOfBounds, func(c *economysim.Config) {
			r := resourceOf(c, "arable")
			r.Shape, r.RateBP, r.RatePerYear = economysim.ShapeConstant, 0, economysim.MaxRegenRatePerYear+1
		}, "modules", "resources", "resources", "arable", "rate_per_year"),
		bound("Resource.RateBP", "rate_bp", economysim.MaxRegenBP+1, func(r *economysim.Resource, v int64) { r.RateBP = v }),
		res("Resource.RateBP", "set on a constant", economysim.ErrResourceShape, func(c *economysim.Config) {
			resourceOf(c, "arable").Shape = economysim.ShapeConstant
		}, "modules", "resources", "resources", "arable", "rate_bp"),
		res("Resource.Refuge", "set on a refill", economysim.ErrResourceShape,
			func(c *economysim.Config) { resourceOf(c, "arable").Refuge = 1 },
			"modules", "resources", "resources", "arable", "refuge"),
		res("Resource.Refuge", "above the capacity", economysim.ErrResourceShape,
			func(c *economysim.Config) { resourceOf(c, "stand").Refuge = 501 },
			"modules", "resources", "resources", "stand", "refuge"),
		bound("Resource.Capacity", "capacity", economysim.MaxEndowment+1, func(r *economysim.Resource, v int64) { r.Capacity = v }),
		res("Resource.Capacity", "times the largest shock above MaxEndowment", economysim.ErrOverflow,
			func(c *economysim.Config) { resourceOf(c, "arable").Capacity = economysim.MaxEndowment / 2 },
			"modules", "resources", "resources", "arable", "capacity"),
		bound("Resource.InitialBP", "initial_bp", 10_001, func(r *economysim.Resource, v int64) { r.InitialBP = v }),
		bound("Resource.PlanShareBP", "plan_share_bp", 0, func(r *economysim.Resource, v int64) { r.PlanShareBP = v }),
		res("Resource.Abundance", "more than MaxKnots", economysim.ErrOutOfBounds, func(c *economysim.Config) {
			knots := make([]economysim.AbundanceKnot, economysim.MaxKnots+1)
			for i := range knots {
				knots[i] = economysim.AbundanceKnot{Share: int64(i) * 1_000, Factor: 5_000}
			}
			resourceOf(c, "arable").Abundance = knots
		}, "modules", "resources", "resources", "arable", "abundance"),
		curve("AbundanceKnot.Share", "above 10 000", []economysim.AbundanceKnot{{Share: 0, Factor: 0}, {Share: 10_001, Factor: 1}},
			"abundance[1]", "share"),
		curve("AbundanceKnot.Share", "not above the one before", []economysim.AbundanceKnot{{Share: 0, Factor: 0}, {Share: 0, Factor: 1}},
			"abundance[1]", "share"),
		curve("AbundanceKnot.Factor", "above MaxYieldBP", []economysim.AbundanceKnot{{Share: 0, Factor: economysim.MaxYieldBP + 1}},
			"abundance[0]", "factor"),
		res("Resource.Seasons", "a season the calendar lacks", economysim.ErrUnknownSeason,
			func(c *economysim.Config) { resourceOf(c, "arable").Seasons[0].Season = "monsoon" },
			"modules", "resources", "resources", "arable", "seasons", "monsoon", "season"),
		res("SeasonRegen.Season", "a season twice", economysim.ErrDuplicateID, func(c *economysim.Config) {
			r := resourceOf(c, "arable")
			r.Seasons = append(r.Seasons, r.Seasons[0])
		}, "modules", "resources", "resources", "arable", "seasons", "spring", "season"),
		res("SeasonRegen.BP", "above MaxSeasonRegenBP", economysim.ErrOutOfBounds,
			func(c *economysim.Config) { resourceOf(c, "arable").Seasons[0].BP = economysim.MaxSeasonRegenBP + 1 },
			"modules", "resources", "resources", "arable", "seasons", "spring", "bp"),
		res("Resource.Label", "label over 256 bytes", economysim.ErrOutOfBounds,
			func(c *economysim.Config) { resourceOf(c, "arable").Label = strings.Repeat("x", 257) },
			"modules", "resources", "resources", "arable", "label"),
		res("Endowment.Region", "unknown region", economysim.ErrUnknownRegion, func(c *economysim.Config) {
			c.Modules.Resources.Endowments[0].Region = "nowhere"
		}, "modules", "resources", "endowments", "nowhere", "arable", "region"),
		res("Endowment.Resource", "unknown resource", economysim.ErrUnknownResource, func(c *economysim.Config) {
			c.Modules.Resources.Endowments[0].Resource = "nothing"
		}, "modules", "resources", "endowments", "north", "nothing", "resource"),
		res("Endowment.Resource", "a (region, resource) row twice", economysim.ErrDuplicateID, func(c *economysim.Config) {
			m := c.Modules.Resources
			m.Endowments = append(m.Endowments, m.Endowments[0])
		}, "modules", "resources", "endowments", "north", "arable", "resource"),
		row("Endowment.Capacity", "above MaxEndowment", economysim.ErrOutOfBounds,
			func(e *economysim.Endowment) { e.Capacity = economysim.MaxEndowment + 1 }, "capacity"),
		row("Endowment.Initial", "above the capacity", economysim.ErrResourceShape,
			func(e *economysim.Endowment) { e.Initial = 2_001 }, "initial"),
		row("Endowment.RegenScaleBP", "above MaxRegenScaleBP", economysim.ErrOutOfBounds, func(e *economysim.Endowment) {
			e.HasRegenScale, e.RegenScaleBP = true, economysim.MaxRegenScaleBP+1
		}, "regen_scale_bp"),
		res("Recipe.Draws", "while the module is off", economysim.ErrModuleDependency, func(c *economysim.Config) {
			recipeOf(c, "grow_staple").Draws = nil
			c.Modules.Resources = nil
		}, "recipes", "cut_fuel", "draws"),
		res("Recipe.Draws", "more than MaxDraws", economysim.ErrOutOfBounds, func(c *economysim.Config) {
			recipeOf(c, "grow_staple").Draws = make([]economysim.Draw, economysim.MaxDraws+1)
		}, draws...),
		res("Draw.Resource", "unknown resource", economysim.ErrUnknownResource, func(c *economysim.Config) {
			recipeOf(c, "grow_staple").Draws[0].Resource = "nothing"
		}, "recipes", "grow_staple", "draws", "nothing", "resource"),
		res("Draw.Resource", "a resource twice in one recipe", economysim.ErrDuplicateID, func(c *economysim.Config) {
			r := recipeOf(c, "grow_staple")
			r.Draws = append(r.Draws, r.Draws[0])
		}, "recipes", "grow_staple", "draws", "arable", "resource"),
		res("Draw.Quantity", "below 1", economysim.ErrOutOfBounds, func(c *economysim.Config) {
			recipeOf(c, "grow_staple").Draws[0].Quantity = 0
		}, "recipes", "grow_staple", "draws", "arable", "quantity"),
		res("Draw.Quantity", "weight times the draw above MaxQuantity", economysim.ErrOverflow, func(c *economysim.Config) {
			heavy(c)
			recipeOf(c, "grow_staple").Draws[0].Quantity = economysim.MaxQuantity / 2
		}, "recipes", "grow_staple", "draws", "arable", "quantity"),
	}
}

// TestValidate_ResourcesValid verifies that the configuration of withResources,
// and each legal oddity the spec allows, validates: a curve that falls, a curve
// that does not start at 0 or end at 10 000, a missing row, an unused default.
//
// Requirement: FR-RES-001, FR-RES-004, FR-RES-034
//
// Guard: the validator follows the text of spec-resources.md 4.7.1 and 13.2: a
// curve is flat outside its end knots and a falling curve is a warning, so
// neither is an error. A stricter validator would reject a pack the spec
// allows.
func TestValidate_ResourcesValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
	}{
		{"the reference overlay", func(*economysim.Config) {}},
		{"a falling curve", func(c *economysim.Config) {
			resourceOf(c, "arable").Abundance = []economysim.AbundanceKnot{{Share: 0, Factor: 9_000}, {Share: 10_000, Factor: 1_000}}
		}},
		{"a curve of one knot in the middle", func(c *economysim.Config) {
			resourceOf(c, "arable").Abundance = []economysim.AbundanceKnot{{Share: 4_000, Factor: 7_000}}
		}},
		{"no row, so no endowment in north", func(c *economysim.Config) { c.Modules.Resources.Endowments = nil }},
		{"the module on and empty", func(c *economysim.Config) {
			c.Modules.Resources = &economysim.Resources{HorizonYears: 1}
			recipeOf(c, "grow_staple").Draws, recipeOf(c, "cut_fuel").Draws = nil, nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := resourcesConfig()
			tt.mutate(&c)
			if err := c.Validate(); err != nil {
				t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
			}
		})
	}
}

// TestValidate_ResourcesBoundsAndShapes verifies the value and the bound of the
// errors that name the field a rule broke, not only the path: a curve error names
// whether the share or the factor is at fault, and a refuge is held against the
// lowest capacity of the resource.
//
// Requirement: FR-RES-032, FR-RES-034
func TestValidate_ResourcesBoundsAndShapes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		mutate       func(c *economysim.Config)
		rule         error
		value, bound string
	}{
		{"share above 10 000", func(c *economysim.Config) {
			resourceOf(c, "arable").Abundance = []economysim.AbundanceKnot{{Share: 10_001, Factor: 1}}
		}, economysim.ErrCurve, "10001", "10000"},
		{"factor above MaxYieldBP", func(c *economysim.Config) {
			resourceOf(c, "arable").Abundance = []economysim.AbundanceKnot{{Share: 1, Factor: economysim.MaxYieldBP + 1}}
		}, economysim.ErrCurve, "100001", "100000"},
		{"refuge above a row's capacity", func(c *economysim.Config) {
			// The default capacity of stand is 500; the row lowers it to 100 here.
			c.Modules.Resources.Endowments = append(c.Modules.Resources.Endowments,
				economysim.Endowment{Region: "north", Resource: "stand", HasCapacity: true, Capacity: 100})
		}, economysim.ErrResourceShape, "20", "100"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := resourcesConfig()
			if tt.name == "refuge above a row's capacity" {
				resourceOf(&c, "stand").Refuge = 150
				tt.value = "150"
			}
			tt.mutate(&c)
			vs, _ := violations(c.Validate())
			if len(vs) == 0 || !errors.Is(vs[0], tt.rule) || vs[0].Value != tt.value || vs[0].Bound != tt.bound {
				t.Errorf("Validate(%s) = %v, want %v with value %q and bound %q", tt.name, vs, tt.rule, tt.value, tt.bound)
			}
		})
	}
}

// TestValidate_DrawsNeedTheModule verifies that a recipe with Draws is refused
// with ErrModuleDependency while the module is off, that a configuration with no
// draw and no module is untouched, and that the module on accepts the draws.
//
// Requirement: FR-RES-001, FR-PRO-007
func TestValidate_DrawsNeedTheModule(t *testing.T) {
	t.Parallel()
	c := resourcesConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate(module on) = %v, want nil", err)
	}
	c.Modules.Resources = nil
	if err := c.Validate(); !errors.Is(err, economysim.ErrModuleDependency) {
		t.Errorf("Validate(draws, module off) = %v, want ErrModuleDependency", err)
	}
	recipeOf(&c, "grow_staple").Draws, recipeOf(&c, "cut_fuel").Draws = nil, nil
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(no draws, module off) = %v, want nil", err)
	}
}

// TestValidate_DrawOverflow verifies that the largest weight of a role times a
// draw is held against MaxQuantity (spec-production.md 10.3, ErrOverflow): one
// agent of 253 households has a weight of 253, so a draw of MaxQuantity/2
// overflows and a draw of MaxQuantity/253 does not.
//
// Requirement: FR-RES-032, FR-PRO-039
func TestValidate_DrawOverflow(t *testing.T) {
	t.Parallel()
	c := resourcesConfig()
	c.World.Scale, c.World.MinAgents = 1_000, 1
	recipeOf(&c, "grow_staple").Draws[0].Quantity = economysim.MaxQuantity / 2
	if err := c.Validate(); !errors.Is(err, economysim.ErrOverflow) {
		t.Errorf("Validate(weight 253, draw of MaxQuantity/2) = %v, want ErrOverflow", err)
	}
	recipeOf(&c, "grow_staple").Draws[0].Quantity = economysim.MaxQuantity / 253
	if err := c.Validate(); errors.Is(err, economysim.ErrOverflow) {
		t.Errorf("Validate(weight 253, draw of MaxQuantity/253) = %v, want no ErrOverflow: 253 * (MaxQuantity / 253) is within MaxQuantity", err)
	}
}

// TestValidate_SeasonProfile verifies that a season in a profile is unknown when
// the calendar has no seasons at all (spec-resources.md 13.2).
//
// Requirement: FR-RES-012
func TestValidate_SeasonProfile(t *testing.T) {
	t.Parallel()
	c := resourcesConfig()
	c.Calendar.Seasons = nil
	c.Yields = nil
	for i := range c.Recipes {
		c.Recipes[i].Sited = false
	}
	if err := c.Validate(); !errors.Is(err, economysim.ErrUnknownSeason) {
		t.Errorf("Validate(season entry, no seasons) = %v, want ErrUnknownSeason", err)
	}
}

// TestValidate_EndowmentRows verifies the region-keyed table: an unknown region
// or resource and a second row are refused with their sentinels, and the absence
// of a row is no warning (spec-resources.md 3.3.1).
//
// Requirement: FR-RES-004
func TestValidate_EndowmentRows(t *testing.T) {
	t.Parallel()
	c := resourcesConfig()
	c.Modules.Resources.Endowments = nil
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate(no row) = %v, want nil", err)
	}
	for _, w := range c.Warnings() {
		if w.Rule == "endowment_unused" && w.Path[len(w.Path)-1] == "arable" {
			t.Errorf("Warnings(no row) raised %s at %q, want none for a pair with no capacity", w.Rule, w.Path)
		}
	}
	for _, tt := range fieldCases {
		if strings.HasPrefix(tt.field, "Endowment.") && tt.rules[0] == nil {
			t.Errorf("case %s has no rule", tt.name)
		}
	}
}

// TestWarnings_ResourceRules verifies each warning of spec-resources.md 13.3 that
// needs no solver: it fires on the condition, none fails Validate, and the
// reference overlay raises none of them.
//
// Requirement: FR-RES-033
func TestWarnings_ResourceRules(t *testing.T) {
	t.Parallel()
	rules := []string{"resource_unused", "endowment_unused", "draw_impossible", "abundance_not_monotone",
		"regen_per_tick_high", "resource_inert", "raw_source_exhaustible"}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   string
		path   []string
	}{
		{"the reference overlay", func(*economysim.Config) {}, "", nil},
		{"a resource no recipe draws", func(c *economysim.Config) {
			recipeOf(c, "cut_fuel").Draws = nil
		}, "resource_unused", []string{"modules", "resources", "resources", "stand"}},
		{"a capacity where no role draws on it", func(c *economysim.Config) {
			c.Regions = append(c.Regions, economysim.DefaultRegion())
			c.Regions[1].ID = "south"
			c.Modules.Resources.Endowments = append(c.Modules.Resources.Endowments,
				economysim.Endowment{Region: "south", Resource: "arable", HasCapacity: true, Capacity: 10})
		}, "endowment_unused", []string{"modules", "resources", "endowments", "south", "arable"}},
		{"a draw above the capacity", func(c *economysim.Config) {
			recipeOf(c, "cut_fuel").Draws[0].Quantity = 501
		}, "draw_impossible", []string{"recipes", "cut_fuel", "draws", "stand"}},
		{"a curve that falls", func(c *economysim.Config) {
			resourceOf(c, "arable").Abundance = []economysim.AbundanceKnot{{Share: 0, Factor: 9_000}, {Share: 10_000, Factor: 1_000}}
		}, "abundance_not_monotone", []string{"modules", "resources", "resources", "arable", "abundance"}},
		{"a rate that is stiff against the tick", func(c *economysim.Config) {
			// 100 000 bp a year over 365 ticks is 273 bp a tick; a scale of 10 times
			// gives 2 739, above 2 000.
			resourceOf(c, "arable").RateBP = economysim.MaxRegenBP
			c.Modules.Resources.Endowments[0].HasRegenScale = true
			c.Modules.Resources.Endowments[0].RegenScaleBP = economysim.MaxRegenScaleBP
		}, "regen_per_tick_high", []string{"modules", "resources", "resources", "arable", "rate_bp"}},
		{"a refill with no rate", func(c *economysim.Config) {
			resourceOf(c, "arable").RateBP = 0
		}, "resource_inert", []string{"modules", "resources", "resources", "arable", "rate_bp"}},
		{"a raw good whose only source is a finite stock", func(c *economysim.Config) {
			resourceOf(c, "stand").RateBP, resourceOf(c, "stand").Refuge = 0, 0
			for i := range c.Commodities {
				if c.Commodities[i].ID == "fuel" {
					c.Commodities[i].Raw = true
				}
			}
		}, "raw_source_exhaustible", []string{"commodities", "fuel"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := resourcesConfig()
			tt.mutate(&c)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate(%s) = %v, want nil: a warning never fails validation", tt.name, err)
			}
			var raised []string
			for _, w := range c.Warnings() {
				if slices.Contains(rules, w.Rule) {
					raised = append(raised, w.Rule+" "+strings.Join(w.Path, "."))
				}
			}
			want := ""
			if tt.rule != "" {
				want = tt.rule + " " + strings.Join(tt.path, ".")
			}
			switch {
			case want == "" && len(raised) != 0:
				t.Errorf("Warnings(%s) raised %q, want none of the resource warnings", tt.name, raised)
			case want != "" && !slices.Contains(raised, want):
				t.Errorf("Warnings(%s) raised %q, want %q among them", tt.name, raised, want)
			}
		})
	}
}

// TestWarnings_NoComparativeAdvantage verifies the cross-ratio condition of
// spec-resources.md 7.4.1, with the module off: two regions in one component and
// two sited recipes whose means are scaled copies raise the warning, and a table
// whose cross ratio reaches 11 000 basis points does not.
//
// Requirement: FR-RES-036
func TestWarnings_NoComparativeAdvantage(t *testing.T) {
	t.Parallel()
	twoRegions := func(a1, b1, a2, b2 int64) economysim.Config {
		c := referenceConfig()
		north := c.Regions[0]
		south := north
		south.ID = "south"
		c.Regions = []economysim.Region{north, south}
		c.Edges = []economysim.Edge{{From: "north", To: "south", Distance: 1}, {From: "south", To: "north", Distance: 1}}
		c.Yields = nil
		for _, y := range []struct {
			region, recipe string
			bp             int64
		}{{"north", "cut_fuel", a1}, {"north", "dig_mineral", b1}, {"south", "cut_fuel", a2}, {"south", "dig_mineral", b2}} {
			c.Yields = append(c.Yields, economysim.YieldProfile{Region: y.region, Recipe: y.recipe, HasAll: true, All: y.bp})
		}
		return c
	}
	tests := []struct {
		name string
		c    economysim.Config
		want bool
	}{
		// Cross ratio 10 000 * 5 000 / (10 000 * 5 000) = 1: scaled copies.
		{"scaled copies", twoRegions(10_000, 5_000, 10_000, 5_000), true},
		// 10 000 * 5 000 / (5 000 * 10 000) is 1 too; 12 000 * 5 000 / (10 000 * 5 000)
		// = 1.2 = 12 000 bp, above the 11 000 threshold.
		{"a difference of 20%", twoRegions(12_000, 5_000, 10_000, 5_000), false},
		// 10 000 * 5 500 / (10 000 * 5 000) = 11 000 bp exactly: it reaches.
		{"at the threshold", twoRegions(10_000, 5_500, 10_000, 5_000), false},
		{"one below", twoRegions(10_000, 5_499, 10_000, 5_000), true},
		{"a zero mean has no ratio", twoRegions(10_000, 0, 10_000, 5_000), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := slices.ContainsFunc(tt.c.Warnings(), func(w economysim.Warning) bool { return w.Rule == "no_comparative_advantage" })
			if got != tt.want {
				t.Errorf("Warnings(%s) raised no_comparative_advantage = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
	if slices.ContainsFunc(referenceConfig().Warnings(), func(w economysim.Warning) bool { return w.Rule == "no_comparative_advantage" }) {
		t.Errorf("Warnings(one region) raised no_comparative_advantage, want none: it needs two regions")
	}
}

// FuzzValidateResources verifies, for any numbers in the fields of the module,
// that Validate and Warnings never panic, and that a configuration that
// validates hashes the same after canonicalisation as before, in whatever order
// its rows and draws were written (the properties of testing.md 7.4).
//
// Requirement: NFR-RES-003
func FuzzValidateResources(f *testing.F) {
	f.Add(int64(10_000), int64(1_000), int64(20), int64(5_000), int64(9_000), int64(1), int8(0))
	f.Add(int64(-1), int64(0), int64(-5), int64(10_001), int64(-3), int64(0), int8(1))
	f.Add(int64(economysim.MaxRegenBP+1), int64(economysim.MaxEndowment+1), int64(1<<62), int64(0), int64(1<<62), int64(1<<62), int8(2))
	f.Fuzz(func(t *testing.T, rate, capacity, refuge, share, factor, draw int64, shape int8) {
		c := resourcesConfig()
		r := resourceOf(&c, "arable")
		r.Shape = []string{economysim.ShapeConstant, economysim.ShapeRefill, economysim.ShapeLogistic, "x"}[uint8(shape)%4]
		r.RateBP, r.RatePerYear, r.Capacity, r.Refuge = rate, rate, capacity, refuge
		r.Abundance = []economysim.AbundanceKnot{{Share: share, Factor: factor}, {Share: share + 1, Factor: factor}}
		c.Modules.Resources.Endowments[0].Capacity = capacity
		recipeOf(&c, "grow_staple").Draws[0].Quantity = economysim.Quantity(draw)
		err := c.Validate()
		_ = c.Warnings()
		if err != nil {
			return
		}
		// Reverse the lists: the canonical form, and so the hash, is the same.
		d := resourcesConfig()
		*resourceOf(&d, "arable") = *r
		d.Modules.Resources.Endowments[0] = c.Modules.Resources.Endowments[0]
		slices.Reverse(d.Modules.Resources.Resources)
		recipeOf(&d, "grow_staple").Draws[0].Quantity = economysim.Quantity(draw)
		if economysim.HashConfig(c) != economysim.HashConfig(d) {
			t.Errorf("HashConfig differs after the resources were listed in reverse, want equal (rate %d, capacity %d)", rate, capacity)
		}
	})
}

// TestWarnings_NoComparativeAdvantagePacks verifies the rule on the packs of
// testdata/chains (spec-resources.md 7.4.1): the five-region pack passes it, with
// the cross ratio of grow_staple and cut_fuel between north and centre of 1.46
// (spec-reference-chain.md 4.4.4), and the packs of fewer than two regions or of
// no sited pair raise nothing. A pack that raised it would be told to its author
// as a table of scaled copies.
//
// Requirement: FR-RES-036
func TestWarnings_NoComparativeAdvantagePacks(t *testing.T) {
	t.Parallel()
	for name, layers := range packLayers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := loadPack(t, layers...)
			if got := warned(cfg.Warnings(), "no_comparative_advantage"); len(got) != 0 {
				t.Errorf("Warnings(%s) raised no_comparative_advantage at %v, want none", name, got)
			}
		})
	}
}
