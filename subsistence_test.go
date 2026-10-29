package economysim_test

import (
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the subsistence module's configuration: its validation, its
// warnings, its effective lists and its tradable items. The module has no world
// effect in this file; the hash, the canonical order and the loader are tested
// beside their code (spec-subsistence.md 3, 12, 13).

// ownUseIDs are the three own-use recipes withSubsistence adds, one for each
// upkeep good of the reference configuration.
var ownUseIDs = []string{"own_fuel", "own_prepared", "own_staple"}

// withSubsistence switches the subsistence module on in c, a copy of
// referenceConfig, with the three sited source recipes of the overlay
// overlay/own-use.toml (own_staple 2 staple, own_prepared 1 prepared, own_fuel 1
// fuel, duration 1, a factor of 10 000 in north) on the default list, the closed
// role visitor on an empty row, and the default tunables. Each covers exactly one
// tick of its upkeep (2 * 10 000 / (1 * 2) = 10 000), so none of the warnings of
// the module is raised.
func withSubsistence(c *economysim.Config) {
	own := func(id, commodity string, q economysim.Quantity) {
		r := economysim.DefaultRecipe()
		r.ID, r.Sited = id, true
		r.Outputs = []economysim.Output{{Commodity: commodity, Quantity: q, CostShare: 10_000}}
		c.Recipes = append(c.Recipes, r)
		c.Yields = append(c.Yields, economysim.YieldProfile{Region: "north", Recipe: id, HasAll: true, All: 10_000})
	}
	own("own_fuel", "fuel", 1)
	own("own_prepared", "prepared", 1)
	own("own_staple", "staple", 2)
	m := economysim.DefaultSubsistence()
	m.Recipes = slices.Clone(ownUseIDs)
	m.Roles = []economysim.RoleOwnUse{{Role: "visitor"}}
	c.Modules.Subsistence = &m
}

// subsistenceConfig returns referenceConfig with the subsistence module on.
func subsistenceConfig() economysim.Config {
	c := referenceConfig()
	withSubsistence(&c)
	return c
}

// subsistenceFieldCases are the failing cases of the rows of the subsistence
// module in the parameter-owner table (spec-subsistence.md 12, 13, FR-CFG-031):
// each breaks one field of a valid configuration. buildFieldCases appends them
// to the table that the registry test reads.
func subsistenceFieldCases() []fieldCase {
	base := []string{"modules", "subsistence"}
	sub := func(field, name string, rule error, mutate func(m *economysim.SubsistenceConfig), path ...string) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{path},
			mutate: func(c *economysim.Config) { withSubsistence(c); mutate(c.Modules.Subsistence) }}
	}
	nine := slices.Repeat([]string{"own_fuel"}, economysim.MaxOwnUseRecipes+1)
	return []fieldCase{
		sub("SubsistenceConfig.Recipes", "more than MaxOwnUseRecipes", economysim.ErrOutOfBounds,
			func(m *economysim.SubsistenceConfig) { m.Recipes = nine }, append(slices.Clone(base), "recipes")...),
		sub("SubsistenceConfig.Recipes", "a recipe that is not defined", economysim.ErrUnknownRecipe,
			func(m *economysim.SubsistenceConfig) { m.Recipes = append(m.Recipes, "nope") },
			append(slices.Clone(base), "recipes", "nope")...),
		sub("SubsistenceConfig.Recipes", "a recipe twice", economysim.ErrDuplicateID,
			func(m *economysim.SubsistenceConfig) { m.Recipes = append(m.Recipes, "own_fuel") },
			append(slices.Clone(base), "recipes", "own_fuel")...),
		{field: "SubsistenceConfig.Recipes", name: "a recipe with inputs", rules: []error{economysim.ErrOwnUseNotSource},
			paths: [][]string{append(slices.Clone(base), "recipes", "own_fuel")},
			mutate: func(c *economysim.Config) {
				withSubsistence(c)
				recipeOf(c, "own_fuel").Inputs = []economysim.Item{{Commodity: "staple", Quantity: 1}}
			}},
		sub("SubsistenceConfig.Roles", "more than MaxRoles", economysim.ErrOutOfBounds,
			func(m *economysim.SubsistenceConfig) { m.Roles = make([]economysim.RoleOwnUse, economysim.MaxRoles+1) },
			append(slices.Clone(base), "roles")...),
		sub("SubsistenceConfig.EnterTicks", "below 1", economysim.ErrOutOfBounds,
			func(m *economysim.SubsistenceConfig) { m.EnterTicks = 0 }, append(slices.Clone(base), "enter_ticks")...),
		sub("SubsistenceConfig.EnterTicks", "above MaxOwnUseTicks", economysim.ErrOutOfBounds,
			func(m *economysim.SubsistenceConfig) { m.EnterTicks = economysim.MaxOwnUseTicks + 1 },
			append(slices.Clone(base), "enter_ticks")...),
		sub("SubsistenceConfig.ExitTicks", "not above EnterTicks", economysim.ErrOwnUseBand,
			func(m *economysim.SubsistenceConfig) { m.ExitTicks = m.EnterTicks }, append(slices.Clone(base), "exit_ticks")...),
		sub("SubsistenceConfig.ExitTicks", "above MaxOwnUseTicks", economysim.ErrOutOfBounds,
			func(m *economysim.SubsistenceConfig) { m.ExitTicks = economysim.MaxOwnUseTicks + 1 },
			append(slices.Clone(base), "exit_ticks")...),
		sub("SubsistenceConfig.RetryJobs", "below 0", economysim.ErrOutOfBounds,
			func(m *economysim.SubsistenceConfig) { m.RetryJobs = -1 }, append(slices.Clone(base), "retry_jobs")...),
		sub("SubsistenceConfig.RetryJobs", "above MaxRetryJobs", economysim.ErrOutOfBounds,
			func(m *economysim.SubsistenceConfig) { m.RetryJobs = economysim.MaxRetryJobs + 1 },
			append(slices.Clone(base), "retry_jobs")...),
		sub("RoleOwnUse.Role", "a role that is not defined", economysim.ErrUnknownRole,
			func(m *economysim.SubsistenceConfig) { m.Roles = append(m.Roles, economysim.RoleOwnUse{Role: "ghost"}) },
			append(slices.Clone(base), "role", "ghost", "role")...),
		sub("RoleOwnUse.Role", "a role with two rows", economysim.ErrDuplicateID,
			func(m *economysim.SubsistenceConfig) {
				m.Roles = append(m.Roles, economysim.RoleOwnUse{Role: "visitor"})
			},
			append(slices.Clone(base), "role", "visitor", "role")...),
		sub("RoleOwnUse.Recipes", "more than MaxOwnUseRecipes", economysim.ErrOutOfBounds,
			func(m *economysim.SubsistenceConfig) { m.Roles[0].Recipes = nine },
			append(slices.Clone(base), "role", "visitor", "recipes")...),
		sub("RoleOwnUse.Recipes", "a recipe that is not defined", economysim.ErrUnknownRecipe,
			func(m *economysim.SubsistenceConfig) { m.Roles[0].Recipes = []string{"nope"} },
			append(slices.Clone(base), "role", "visitor", "recipes", "nope")...),
	}
}

// mutateRecipe returns a mutation of the config that changes one recipe.
func mutateRecipe(id string, f func(r *economysim.Recipe)) func(c *economysim.Config) {
	return func(c *economysim.Config) { f(recipeOf(c, id)) }
}

// wantSubsistenceError verifies that the subsistence configuration, mutated,
// is rejected with exactly the rule, at exactly the path.
func wantSubsistenceError(t *testing.T, mutate func(c *economysim.Config), rule error, path ...string) {
	t.Helper()
	c := subsistenceConfig()
	mutate(&c)
	err := c.Validate()
	if !errors.Is(err, rule) {
		t.Fatalf("Validate() = %v, want %v", err, rule)
	}
	vs, other := violations(err)
	if len(other) != 0 || len(vs) != 1 || !slices.Equal(vs[0].Path, path) || !errors.Is(vs[0], rule) {
		t.Errorf("Validate() = %v, want only %v at %v", err, rule, path)
	}
}

// TestValidate_SubsistenceValid verifies that the module with its three source
// recipes validates, and that the same recipes on no list are an ordinary pack.
//
// Requirement: FR-SUB-001
func TestValidate_SubsistenceValid(t *testing.T) {
	t.Parallel()
	c := subsistenceConfig()
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(subsistence on) = %v, want nil", err)
	}
	c.Modules.Subsistence = nil
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(the recipes with the module off) = %v, want nil", err)
	}
}

// TestValidate_OwnUseNotSource verifies that an own-use recipe with inputs, and
// one that requires a tool, is rejected with ErrOwnUseNotSource at its list: a
// floor that needs a purchased input depends on the market it is under.
//
// Requirement: FR-SUB-004
func TestValidate_OwnUseNotSource(t *testing.T) {
	t.Parallel()
	path := []string{"modules", "subsistence", "recipes", "own_fuel"}
	t.Run("inputs", func(t *testing.T) {
		t.Parallel()
		wantSubsistenceError(t, mutateRecipe("own_fuel", func(r *economysim.Recipe) {
			r.Inputs = []economysim.Item{{Commodity: "staple", Quantity: 1}}
		}), economysim.ErrOwnUseNotSource, path...)
	})
	t.Run("a tool", func(t *testing.T) {
		t.Parallel()
		wantSubsistenceError(t, mutateRecipe("own_fuel", func(r *economysim.Recipe) {
			r.Requires = []economysim.Tool{{Commodity: "tool_a", WearPerUse: 1}}
		}), economysim.ErrOwnUseNotSource, path...)
	})
	t.Run("on a role row", func(t *testing.T) {
		t.Parallel()
		wantSubsistenceError(t, func(c *economysim.Config) {
			c.Modules.Subsistence.Roles[0].Recipes = []string{"own_fuel"}
			recipeOf(c, "own_fuel").Inputs = []economysim.Item{{Commodity: "staple", Quantity: 1}}
			c.Modules.Subsistence.Recipes = []string{"own_prepared", "own_staple"}
		}, economysim.ErrOwnUseNotSource, "modules", "subsistence", "role", "visitor", "recipes", "own_fuel")
	})
}

// TestValidate_OwnUseNotSited verifies that an own-use recipe without Sited is
// rejected with ErrOwnUseNotSited: an unsited source would run everywhere at a
// factor of 10 000.
//
// Requirement: FR-SUB-005
func TestValidate_OwnUseNotSited(t *testing.T) {
	t.Parallel()
	wantSubsistenceError(t, mutateRecipe("own_staple", func(r *economysim.Recipe) { r.Sited = false }),
		economysim.ErrOwnUseNotSited, "modules", "subsistence", "recipes", "own_staple")
}

// TestValidate_OwnUseOverlap verifies that a recipe on the effective own-use list
// and on Role.Recipes of the same role is rejected with ErrOwnUseOverlap, under
// the role, and that the same recipe on the Recipes of a role whose row replaces
// the list is not an overlap.
//
// Requirement: FR-SUB-006
func TestValidate_OwnUseOverlap(t *testing.T) {
	t.Parallel()
	wantSubsistenceError(t, func(c *economysim.Config) {
		for i := range c.Roles {
			if c.Roles[i].ID == "forester" {
				c.Roles[i].Recipes = append(c.Roles[i].Recipes, "own_fuel")
			}
		}
	}, economysim.ErrOwnUseOverlap, "modules", "subsistence", "role", "forester", "recipes", "own_fuel")

	c := subsistenceConfig()
	for i := range c.Roles {
		if c.Roles[i].ID == "forester" {
			c.Roles[i].Recipes = append(c.Roles[i].Recipes, "own_fuel")
		}
	}
	c.Modules.Subsistence.Roles = append(c.Modules.Subsistence.Roles, economysim.RoleOwnUse{Role: "forester", Recipes: []string{"own_staple"}})
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(own_fuel a forester recipe, the forester's row on own_staple) = %v, want nil", err)
	}
}

// TestValidate_OwnUseBand verifies that an ExitTicks equal to or below EnterTicks
// is rejected with ErrOwnUseBand, and one just above is accepted.
//
// Requirement: FR-SUB-007
func TestValidate_OwnUseBand(t *testing.T) {
	t.Parallel()
	for _, exit := range []int64{2, 1} {
		wantSubsistenceError(t, func(c *economysim.Config) { c.Modules.Subsistence.ExitTicks = exit },
			economysim.ErrOwnUseBand, "modules", "subsistence", "exit_ticks")
	}
	c := subsistenceConfig()
	c.Modules.Subsistence.ExitTicks = c.Modules.Subsistence.EnterTicks + 1
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(ExitTicks one above EnterTicks) = %v, want nil", err)
	}
}

// TestValidate_OwnUseBounds verifies the numeric bounds and the list length: the
// largest values are accepted, one beyond is ErrOutOfBounds at the field.
//
// Requirement: FR-SUB-007
func TestValidate_OwnUseBounds(t *testing.T) {
	t.Parallel()
	c := subsistenceConfig()
	m := c.Modules.Subsistence
	m.EnterTicks, m.ExitTicks, m.RetryJobs = economysim.MaxOwnUseTicks-1, economysim.MaxOwnUseTicks, economysim.MaxRetryJobs
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(the largest tunables) = %v, want nil", err)
	}
	m.EnterTicks, m.ExitTicks, m.RetryJobs = 1, 2, 0
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(the smallest tunables) = %v, want nil", err)
	}
	wantSubsistenceError(t, func(c *economysim.Config) { c.Modules.Subsistence.RetryJobs = economysim.MaxRetryJobs + 1 },
		economysim.ErrOutOfBounds, "modules", "subsistence", "retry_jobs")
	wantSubsistenceError(t, func(c *economysim.Config) {
		c.Modules.Subsistence.Recipes = slices.Repeat([]string{"own_fuel"}, economysim.MaxOwnUseRecipes+1)
	}, economysim.ErrOutOfBounds, "modules", "subsistence", "recipes")
}

// TestValidate_OwnUseUnknownRecipe verifies that a recipe in a list that is not
// defined is ErrUnknownRecipe, in the default list and in a row.
//
// Requirement: FR-SUB-008
func TestValidate_OwnUseUnknownRecipe(t *testing.T) {
	t.Parallel()
	wantSubsistenceError(t, func(c *economysim.Config) { c.Modules.Subsistence.Recipes = []string{"own_fuel", "nope"} },
		economysim.ErrUnknownRecipe, "modules", "subsistence", "recipes", "nope")
	wantSubsistenceError(t, func(c *economysim.Config) { c.Modules.Subsistence.Roles[0].Recipes = []string{"nope"} },
		economysim.ErrUnknownRecipe, "modules", "subsistence", "role", "visitor", "recipes", "nope")
}

// TestValidate_OwnUseUnknownRole verifies that a row naming a role that is not
// defined is ErrUnknownRole.
//
// Requirement: FR-SUB-008
func TestValidate_OwnUseUnknownRole(t *testing.T) {
	t.Parallel()
	wantSubsistenceError(t, func(c *economysim.Config) {
		c.Modules.Subsistence.Roles = append(c.Modules.Subsistence.Roles, economysim.RoleOwnUse{Role: "ghost"})
	}, economysim.ErrUnknownRole, "modules", "subsistence", "role", "ghost", "role")
}

// TestValidate_OwnUseDuplicateRow verifies that two rows for one role, and a
// recipe twice in one list, are ErrDuplicateID.
//
// Requirement: FR-SUB-002, FR-SUB-008
func TestValidate_OwnUseDuplicateRow(t *testing.T) {
	t.Parallel()
	wantSubsistenceError(t, func(c *economysim.Config) {
		c.Modules.Subsistence.Roles = append(c.Modules.Subsistence.Roles, economysim.RoleOwnUse{Role: "visitor"})
	}, economysim.ErrDuplicateID, "modules", "subsistence", "role", "visitor", "role")
	wantSubsistenceError(t, func(c *economysim.Config) {
		c.Modules.Subsistence.Roles[0].Recipes = []string{"own_fuel", "own_fuel"}
	}, economysim.ErrDuplicateID, "modules", "subsistence", "role", "visitor", "recipes", "own_fuel")
}

// TestValidate_DrawsNeedResources verifies that an own-use recipe with Draws
// while the resources module is off is rejected with ErrModuleDependency, the
// rule of the resources module applied to a recipe on an own-use list.
//
// Requirement: FR-SUB-050
func TestValidate_DrawsNeedResources(t *testing.T) {
	t.Parallel()
	wantSubsistenceError(t, mutateRecipe("own_staple", func(r *economysim.Recipe) {
		r.Draws = []economysim.Draw{{Resource: "arable", Quantity: 1}}
	}), economysim.ErrModuleDependency, "recipes", "own_staple", "draws")
}

// TestSubsistence_NoRegionKeyedField verifies, by reflection over the two
// structs, that the module has no region-keyed table and no per-region field: a
// region differs through its yield rows and endowment rows alone.
//
// Requirement: FR-SUB-009
func TestSubsistence_NoRegionKeyedField(t *testing.T) {
	t.Parallel()
	for _, typ := range []reflect.Type{reflect.TypeFor[economysim.SubsistenceConfig](), reflect.TypeFor[economysim.RoleOwnUse]()} {
		for i := range typ.NumField() {
			f := typ.Field(i)
			elem := f.Type
			if elem.Kind() == reflect.Slice {
				elem = elem.Elem()
			}
			if strings.Contains(f.Name, "Region") || strings.Contains(elem.Name(), "Region") {
				t.Errorf("%s.%s (%s) is region-keyed, want none", typ.Name(), f.Name, f.Type)
			}
		}
	}
}

// TestSubsistence_RegionOffersByYieldRow verifies that whether a region offers
// subsistence is whether its yield row for an own-use recipe is positive: a
// role's group in a region with no row for any of its own-use recipes draws
// own_use_not_available, and with the rows it does not.
//
// Requirement: FR-SUB-009, FR-SUB-038
func TestSubsistence_RegionOffersByYieldRow(t *testing.T) {
	t.Parallel()
	c := subsistenceConfig()
	if ws := ownUseWarnings(c.Warnings()); len(ws) != 0 {
		t.Fatalf("Warnings(with the yield rows) = %v, want no subsistence warning", ws)
	}
	c.Yields = slices.DeleteFunc(c.Yields, func(y economysim.YieldProfile) bool { return strings.HasPrefix(y.Recipe, "own_") })
	ws := ownUseWarnings(c.Warnings())
	if !hasWarning(ws, "own_use_not_available", "regions", "north", "roles", "forester") {
		t.Errorf("Warnings(with no yield row) = %v, want own_use_not_available for the forester in north", ws)
	}
}

// ownUseWarnings returns the warnings of the module.
func ownUseWarnings(ws []economysim.Warning) []economysim.Warning {
	return slices.DeleteFunc(slices.Clone(ws), func(w economysim.Warning) bool {
		return !strings.HasPrefix(w.Rule, "own_use_") && w.Rule != "subsistence_empty"
	})
}

// TestWarnings_Eight verifies the eight warnings of spec-subsistence.md 13.3, one
// case each, and that the configuration of withSubsistence and the overlay of
// overlay/own-use.toml draw none of them. A warning never fails validation.
//
// Requirement: FR-SUB-038
func TestWarnings_Eight(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   string
		path   []string
	}{
		{"subsistence_empty", func(c *economysim.Config) { c.Modules.Subsistence.Recipes = nil },
			"subsistence_empty", []string{"modules", "subsistence"}},
		{"own_use_no_upkeep", func(c *economysim.Config) {
			r := economysim.DefaultRecipe()
			r.ID, r.Sited = "own_ore", true
			r.Outputs = []economysim.Output{{Commodity: "mineral", Quantity: 1, CostShare: 10_000}}
			c.Recipes = append(c.Recipes, r)
			c.Yields = append(c.Yields, economysim.YieldProfile{Region: "north", Recipe: "own_ore", HasAll: true, All: 10_000})
			c.Modules.Subsistence.Roles = append(c.Modules.Subsistence.Roles, economysim.RoleOwnUse{Role: "cook", Recipes: []string{"own_ore"}})
		}, "own_use_no_upkeep", []string{"modules", "subsistence", "role", "cook"}},
		{"own_use_beats_market", func(c *economysim.Config) { recipeOf(c, "own_staple").Outputs[0].Quantity = 1_000 },
			"own_use_beats_market", []string{"regions", "north", "roles", "forester", "recipes", "own_staple"}},
		{"own_use_partial", func(c *economysim.Config) { recipeOf(c, "own_staple").Outputs[0].Quantity = 1 },
			"own_use_partial", []string{"regions", "north", "roles", "forester", "recipes", "own_staple"}},
		{"own_use_trap", func(c *economysim.Config) {
			recipeOf(c, "own_staple").Outputs[0].Quantity = 1
			c.Modules.Subsistence.RetryJobs = 0
		}, "own_use_trap", []string{"regions", "north", "roles", "forester", "recipes", "own_staple"}},
		{"own_use_exit_unreachable", func(c *economysim.Config) { c.Modules.Subsistence.ExitTicks = c.World.CoverTicks + 1 },
			"own_use_exit_unreachable", []string{"modules", "subsistence", "exit_ticks"}},
		{"own_use_slower_than_ladder", func(c *economysim.Config) { recipeOf(c, "own_staple").Duration = c.World.ChangePrivation + 1 },
			"own_use_slower_than_ladder", []string{"recipes", "own_staple"}},
		{"own_use_not_available", func(c *economysim.Config) {
			c.Yields = slices.DeleteFunc(c.Yields, func(y economysim.YieldProfile) bool { return y.Recipe == "own_staple" })
			c.Modules.Subsistence.Recipes = []string{"own_staple"}
		}, "own_use_not_available", []string{"regions", "north", "roles", "forester"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := subsistenceConfig()
			tt.mutate(&c)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate() = %v, want nil: a warning never fails validation", err)
			}
			if ws := c.Warnings(); !hasWarning(ws, tt.rule, tt.path...) {
				t.Errorf("Warnings() = %v, want %s at %v", ownUseWarnings(ws), tt.rule, tt.path)
			}
		})
	}
	t.Run("silent", func(t *testing.T) {
		t.Parallel()
		c := subsistenceConfig()
		if ws := ownUseWarnings(c.Warnings()); len(ws) != 0 {
			t.Errorf("Warnings(withSubsistence) = %v, want none", ws)
		}
		pack := loadValidPack(t, append(slices.Clone(packOne), "overlay/own-use.toml")...)
		if ws := ownUseWarnings(pack.Warnings()); len(ws) != 0 {
			t.Errorf("Warnings(pack one with overlay/own-use.toml) = %v, want none", ws)
		}
	})
}

// viabilityRules returns the rules of the viability errors of err.
func viabilityRules(err error) []string {
	var rules []string
	joined, _ := err.(interface{ Unwrap() []error })
	if joined == nil {
		return nil
	}
	for _, e := range joined.Unwrap() {
		var ve *economysim.ViabilityError
		if errors.As(e, &ve) {
			rules = append(rules, ve.Rule)
		}
	}
	return rules
}

// TestViability_OwnUseOnlyRecipeNotAnActivity verifies that a recipe on an own-use
// list and on no role's Recipes is no activity of the reference programme and
// makes no commodity producible: with the market recipes of prepared (a
// commodity that is not Raw) removed, an upkeep good that only own_prepared makes
// is unproducible_in_realm; the same recipe on a role's Recipes (here the cook's,
// whose own-use row is empty so that the lists do not overlap) is an activity as
// any other and the good is producible.
//
// Requirement: FR-SUB-037
func TestViability_OwnUseOnlyRecipeNotAnActivity(t *testing.T) {
	t.Parallel()
	strip := func(c *economysim.Config) {
		for i := range c.Roles {
			c.Roles[i].Recipes = slices.DeleteFunc(c.Roles[i].Recipes, func(id string) bool { return strings.HasPrefix(id, "prepare") })
		}
		c.Recipes = slices.DeleteFunc(c.Recipes, func(r economysim.Recipe) bool { return strings.HasPrefix(r.ID, "prepare") })
	}
	own := subsistenceConfig()
	strip(&own)
	err := own.Validate()
	if !errors.Is(err, economysim.ErrNotViable) || !slices.Contains(viabilityRules(err), "unproducible_in_realm") {
		t.Errorf("Validate(prepared made only by an own-use recipe) = %v, want unproducible_in_realm", err)
	}

	market := subsistenceConfig()
	strip(&market)
	for i := range market.Roles {
		if market.Roles[i].ID == "cook" {
			market.Roles[i].Recipes = append(market.Roles[i].Recipes, "own_prepared")
		}
	}
	market.Modules.Subsistence.Roles = append(market.Modules.Subsistence.Roles, economysim.RoleOwnUse{Role: "cook"})
	if rules := viabilityRules(market.Validate()); slices.Contains(rules, "unproducible_in_realm") {
		t.Errorf("Validate(prepared made by a role recipe that is also an own-use recipe of other roles) rules = %v, want no unproducible_in_realm", rules)
	}
}

// TestSubsistence_DefaultsEqualTheSpecTable verifies that the tunables of
// DefaultSubsistence are the defaults of the parameter table of
// spec-subsistence.md 12 (2, 3 and 10), which the generic registry check cannot
// read because that table names its scope "Module".
//
// Requirement: FR-SUB-007, FR-CFG-030
func TestSubsistence_DefaultsEqualTheSpecTable(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("doc/specs/spec-subsistence.md")
	if err != nil {
		t.Fatalf("read spec-subsistence.md: %v", err)
	}
	d := economysim.DefaultSubsistence()
	want := map[string]int64{"EnterTicks": d.EnterTicks, "ExitTicks": d.ExitTicks, "RetryJobs": d.RetryJobs}
	found := 0
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "|") {
			continue
		}
		scope, field, n, ok := specDefault(strings.Split(line, "|"))
		if w, known := want[field]; ok && known && scope == "Module" {
			found++
			if n != w {
				t.Errorf("spec-subsistence.md 12: the default of %s is %d in the table and %d in DefaultSubsistence()", field, n, w)
			}
		}
	}
	if found != len(want) {
		t.Errorf("found %d of the %d tunable rows of the parameter table", found, len(want))
	}
}

// FuzzValidateSubsistence verifies, for any numbers in the tunables and any
// recipe and role names in the lists, that Validate and Warnings never panic, and
// that a configuration that validates validates a second time to the same answer
// and hashes the same whichever way its lists were written.
//
// Requirement: FR-SUB-007, FR-SUB-008
func FuzzValidateSubsistence(f *testing.F) {
	f.Add(int64(2), int64(3), int64(10), "own_fuel", "visitor", int8(0))
	f.Add(int64(0), int64(0), int64(-1), "nope", "ghost", int8(1))
	f.Add(int64(economysim.MaxOwnUseTicks+1), int64(1<<62), int64(economysim.MaxRetryJobs+1), "own_staple", "cook", int8(2))
	f.Fuzz(func(t *testing.T, enter, exit, retry int64, recipe, role string, shape int8) {
		c := subsistenceConfig()
		m := c.Modules.Subsistence
		m.EnterTicks, m.ExitTicks, m.RetryJobs = enter, exit, retry
		switch uint8(shape) % 3 {
		case 0:
			m.Recipes = append(m.Recipes, recipe)
		case 1:
			m.Roles = append(m.Roles, economysim.RoleOwnUse{Role: role, Recipes: []string{recipe, recipe}})
		default:
			m.Roles[0].Role, m.Roles[0].Recipes = role, []string{recipe}
		}
		err := c.Validate()
		_ = c.Warnings()
		if err != nil {
			return
		}
		if again := c.Validate(); again != nil {
			t.Fatalf("Validate() = %v the second time after nil", again)
		}
		d := c
		d.Modules.Subsistence = &economysim.SubsistenceConfig{
			Recipes: slices.Clone(m.Recipes), Roles: slices.Clone(m.Roles),
			EnterTicks: m.EnterTicks, ExitTicks: m.ExitTicks, RetryJobs: m.RetryJobs,
		}
		slices.Reverse(d.Modules.Subsistence.Recipes)
		slices.Reverse(d.Modules.Subsistence.Roles)
		if economysim.HashConfig(c) != economysim.HashConfig(d) {
			t.Errorf("HashConfig differs after the lists were reversed, want equal (recipe %q, role %q)", recipe, role)
		}
	})
}

// TestLoad_OwnUseOverlay verifies that overlay/own-use.toml loads over pack one to
// the module the file describes: the three source recipes on the default list in
// ID order, the closed role visitor on an empty row, the default tunables, and
// the yield rows of north; and that the layered pack validates.
//
// Requirement: FR-SUB-047, FR-CFG-077
func TestLoad_OwnUseOverlay(t *testing.T) {
	t.Parallel()
	cfg := loadValidPack(t, append(slices.Clone(packOne), "overlay/own-use.toml")...)
	want := economysim.DefaultSubsistence()
	want.Recipes = slices.Clone(ownUseIDs)
	want.Roles = []economysim.RoleOwnUse{{Role: "visitor"}}
	if got := cfg.Modules.Subsistence; got == nil || !reflect.DeepEqual(*got, want) {
		t.Errorf("overlay module = %+v, want %+v", got, want)
	}
	for _, id := range ownUseIDs {
		r := recipeOf(&cfg, id)
		if r == nil || !r.Sited || len(r.Inputs) != 0 || len(r.Requires) != 0 || r.Duration != 1 {
			t.Errorf("recipe %s = %+v, want a sited source of duration 1", id, r)
		}
	}
}
