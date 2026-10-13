package economysim_test

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode"

	economysim "github.com/Whealinghold/economy-sim"
)

// violations splits the error of Validate into its *ValidationError values, in
// the order returned, and returns the rest (the ErrTooManyErrors entry) apart.
func violations(err error) (found []*economysim.ValidationError, other []error) {
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return nil, []error{err}
	}
	for _, e := range joined.Unwrap() {
		var ve *economysim.ValidationError
		if errors.As(e, &ve) {
			found = append(found, ve)
		} else {
			other = append(other, e)
		}
	}
	return found, other
}

// pathsOf returns the Path of each error.
func pathsOf(vs []*economysim.ValidationError) [][]string {
	out := make([][]string, len(vs))
	for i, v := range vs {
		out[i] = v.Path
	}
	return out
}

// recipeOf returns the recipe with the given ID of c, or nil.
func recipeOf(c *economysim.Config, id string) *economysim.Recipe {
	for i := range c.Recipes {
		if c.Recipes[i].ID == id {
			return &c.Recipes[i]
		}
	}
	return nil
}

// roleOf returns the role with the given ID of c, or nil.
func roleOf(c *economysim.Config, id string) *economysim.Role {
	for i := range c.Roles {
		if c.Roles[i].ID == id {
			return &c.Roles[i]
		}
	}
	return nil
}

// commodityOf returns the commodity with the given ID of c, or nil.
func commodityOf(c *economysim.Config, id string) *economysim.Commodity {
	for i := range c.Commodities {
		if c.Commodities[i].ID == id {
			return &c.Commodities[i]
		}
	}
	return nil
}

// referenceWarnings are the warnings of the reference configuration, in the
// order Warnings returns them. The pack is written to pass its errors, and the
// warnings it draws are informational or about its starting money:
//
//   - tool_unaffordable: the default StartMoney of 10 000 is below the price of
//     tool_b (17 105, the cook's tool) and of tool_a (13 915, the grower's);
//   - cold_start_lockstep: the grower has a recipe of 60 ticks, 253 agents of
//     weight 1 and no job row in the warm start (spec-reference-chain.md 5);
//   - window_extended: the grower's window is 2 * 60 = 120 against 30;
//   - entry_share_window: EntryShare 50 times the longest window 120 is 6 000,
//     above 2 500.
var referenceWarnings = []economysim.Warning{
	{Path: []string{"roles", "cook", "start_money"}, Rule: "tool_unaffordable"},
	{Path: []string{"roles", "grower", "recipes"}, Rule: "cold_start_lockstep"},
	{Path: []string{"roles", "grower", "recipes"}, Rule: "window_extended"},
	{Path: []string{"roles", "grower", "start_money"}, Rule: "tool_unaffordable"},
	{Path: []string{"world", "entry_share"}, Rule: "entry_share_window"},
}

// ruleCase is one configuration that breaks exactly one rule: Validate must
// return errors of rule only, at exactly paths, in that order.
type ruleCase struct {
	name   string
	mutate func(c *economysim.Config)
	paths  [][]string
}

// TestValidate_ReferenceConfigurationIsValid verifies that the reference
// configuration validates and draws exactly the warnings of referenceWarnings,
// and that the same value with
// every entity list reversed (a different written order) validates too, so
// the failing cases below break one rule and nothing else.
//
// Guard: every failing case starts from this configuration. If it were not
// valid, a case could pass by the wrong rule.
//
// Requirement: FR-CFG-005
func TestValidate_ReferenceConfigurationIsValid(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(reference) = %v, want nil", err)
	}
	if w := c.Warnings(); !reflect.DeepEqual(w, referenceWarnings) {
		t.Errorf("Warnings(reference) = %v, want %v", w, referenceWarnings)
	}

	r := referenceConfig()
	slices.Reverse(r.Commodities)
	slices.Reverse(r.Recipes)
	slices.Reverse(r.Roles)
	slices.Reverse(r.Groups)
	slices.Reverse(r.Yields)
	slices.Reverse(r.WarmStart.Prices)
	if err := r.Validate(); err != nil {
		t.Errorf("Validate(reference, lists reversed) = %v, want nil: the written order has no effect", err)
	}
}

// TestValidate_DoesNotChangeTheConfiguration verifies that Validate leaves the
// receiver's slices untouched, including the order of a list that the rules
// sort internally.
//
// Guard: the validator sorts recipes and commodity IDs to be deterministic. A
// sort in place would reorder the caller's configuration and so change its
// hash order and the typed IDs of a world built from it.
func TestValidate_DoesNotChangeTheConfiguration(t *testing.T) {
	t.Parallel()
	build := func() economysim.Config {
		c := referenceConfig()
		slices.Reverse(c.Recipes)
		slices.Reverse(c.Commodities)
		c.Commodities = append(c.Commodities, economysim.Commodity{ID: "dust", ReferencePrice: 1})
		return c
	}
	c := build()
	if err := c.Validate(); err == nil {
		t.Fatalf("Validate(config with an orphan) = nil, want an error to make the validator run its sorts")
	}
	if want := build(); !reflect.DeepEqual(c, want) {
		t.Errorf("Validate changed its receiver: got %+v, want %+v", c, want)
	}
}

// TestValidate_IDRules verifies that an ID that does not match
// [a-z][a-z0-9_]* or exceeds 64 bytes is ErrInvalidID, with the path of the ID
// as given, for every kind of entity and for the calendar's months and seasons.
//
// Requirement: FR-CFG-006, FR-CAL-014
func TestValidate_IDRules(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", economysim.MaxIDLength+1)
	rule := economysim.ErrInvalidID
	for _, tt := range []ruleCase{
		{
			// The only user of "waste" is a recipe output; the output is renamed
			// with it so the reference stays valid.
			name: "commodity with an upper case letter",
			mutate: func(c *economysim.Config) {
				commodityOf(c, "waste").ID = "Waste"
				recipeOf(c, "refine").Outputs[1].Commodity = "Waste"
			},
			paths: [][]string{{"commodities", "Waste", "id"}},
		},
		{
			name: "recipe starting with a digit",
			mutate: func(c *economysim.Config) {
				recipeOf(c, "save_seed").ID = "1save"
				for i := range c.Roles {
					for j, r := range c.Roles[i].Recipes {
						if r == "save_seed" {
							c.Roles[i].Recipes[j] = "1save"
						}
					}
				}
			},
			paths: [][]string{{"recipes", "1save", "id"}},
		},
		{
			// The visitor role has no group, so nothing refers to it.
			name:   "role of 65 bytes",
			mutate: func(c *economysim.Config) { roleOf(c, "visitor").ID = long },
			paths:  [][]string{{"roles", long, "id"}},
		},
		{
			name:   "role with a control character",
			mutate: func(c *economysim.Config) { roleOf(c, "visitor").ID = "vis\x1bitor" },
			paths:  [][]string{{"roles", "vis\x1bitor", "id"}},
		},
		{
			name: "empty region ID",
			mutate: func(c *economysim.Config) {
				// 999 is the cost of the baseline upkeep, so the basket rule is met.
				c.Regions = append(c.Regions, economysim.Region{ID: "", ReferenceBasket: 999})
			},
			paths: [][]string{{"regions", "", "id"}},
		},
		{
			name:   "month with a space",
			mutate: func(c *economysim.Config) { c.Calendar.Months[3].ID = "m 04" },
			paths:  [][]string{{"calendar", "months[3]", "id"}},
		},
		{
			// Winter is not named by a yield entry of the reference configuration.
			name:   "season in upper case",
			mutate: func(c *economysim.Config) { c.Calendar.Seasons[3].ID = "Winter" },
			paths:  [][]string{{"calendar", "seasons[3]", "id"}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("Validate(%s) = nil, want %v at %v", tt.name, rule, tt.paths)
			}
			if !errors.Is(err, rule) {
				t.Errorf("errors.Is(Validate(%s), %v) = false, want true; error: %v", tt.name, rule, err)
			}
			var ve *economysim.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("errors.As(Validate(%s), *ValidationError) = false, want true", tt.name)
			}
			vs, other := violations(err)
			if len(other) != 0 {
				t.Errorf("Validate(%s) returned %d errors that are not a *ValidationError, want none: %v", tt.name, len(other), other)
			}
			for _, v := range vs {
				if !errors.Is(v, rule) {
					t.Errorf("Validate(%s) returned %v, want only %v", tt.name, v, rule)
				}
			}
			if got := pathsOf(vs); !reflect.DeepEqual(got, tt.paths) {
				t.Errorf("Validate(%s) paths = %q, want %q", tt.name, got, tt.paths)
			}
		})
	}
}

// TestValidate_DuplicateRules verifies that two entities of one kind with one
// ID, a commodity twice in a list, two rows for one key and the reserved role
// ID "pool" are ErrDuplicateID, with the path of the repeat.
//
// Requirement: FR-CFG-006, FR-CAL-014, FR-PRO-038, FR-AGT-012, FR-VIA-041, FR-VIA-051
func TestValidate_DuplicateRules(t *testing.T) {
	t.Parallel()
	rule := economysim.ErrDuplicateID
	for _, tt := range []ruleCase{
		{
			name: "commodity twice",
			mutate: func(c *economysim.Config) {
				c.Commodities = append(c.Commodities, *commodityOf(c, "seed"))
			},
			paths: [][]string{{"commodities", "seed", "id"}},
		},
		{
			name: "recipe twice",
			mutate: func(c *economysim.Config) {
				c.Recipes = append(c.Recipes, *recipeOf(c, "save_seed"))
			},
			paths: [][]string{{"recipes", "save_seed", "id"}},
		},
		{
			name: "role twice",
			mutate: func(c *economysim.Config) {
				c.Roles = append(c.Roles, *roleOf(c, "visitor"))
			},
			paths: [][]string{{"roles", "visitor", "id"}},
		},
		{
			name: "region twice",
			mutate: func(c *economysim.Config) {
				c.Regions = append(c.Regions, c.Regions[0])
			},
			paths: [][]string{{"regions", "north", "id"}},
		},
		{
			name:   "reserved role ID pool",
			mutate: func(c *economysim.Config) { roleOf(c, "visitor").ID = "pool" },
			paths:  [][]string{{"roles", "pool", "id"}},
		},
		{
			name:   "month twice",
			mutate: func(c *economysim.Config) { c.Calendar.Months[1].ID = "m01" },
			paths:  [][]string{{"calendar", "months[1]", "id"}},
		},
		{
			name:   "season twice",
			mutate: func(c *economysim.Config) { c.Calendar.Seasons[2].ID = "spring" },
			paths:  [][]string{{"calendar", "seasons[2]", "id"}},
		},
		{
			name: "commodity twice in a recipe input list",
			mutate: func(c *economysim.Config) {
				r := recipeOf(c, "refine")
				r.Inputs = append(r.Inputs, economysim.Item{Commodity: "mineral", Quantity: 1})
			},
			paths: [][]string{{"recipes", "refine", "inputs", "mineral", "commodity"}},
		},
		{
			// staple costs 96, so the basket 999 + 96 = 1 095 is within the 10%
			// tolerance of 1 000 and only the repeat is wrong.
			name: "commodity twice in the world upkeep",
			mutate: func(c *economysim.Config) {
				c.World.Upkeep = append(c.World.Upkeep, economysim.Item{Commodity: "staple", Quantity: 1})
			},
			paths: [][]string{{"world", "upkeep", "staple", "commodity"}},
		},
		{
			name: "recipe twice in a role list",
			mutate: func(c *economysim.Config) {
				r := roleOf(c, "cook")
				r.Recipes = append(r.Recipes, "prepare")
			},
			paths: [][]string{{"roles", "cook", "recipes", "prepare"}},
		},
		{
			name: "two yield profiles for one region and recipe",
			mutate: func(c *economysim.Config) {
				c.Yields = append(c.Yields, c.Yields[0])
			},
			paths: [][]string{{"yields", "north", "cut_fuel"}},
		},
		{
			// More profiles than regions times recipes is no count error: the
			// duplicate is the violation (spec-config.md 7.4.1).
			name: "yield profile table past regions times recipes by a duplicate",
			mutate: func(c *economysim.Config) {
				for _, r := range c.Recipes {
					if !slices.ContainsFunc(c.Yields, func(y economysim.YieldProfile) bool { return y.Recipe == r.ID }) {
						c.Yields = append(c.Yields, economysim.YieldProfile{Region: "north", Recipe: r.ID, HasAll: true, All: 10_000})
					}
				}
				c.Yields = append(c.Yields, c.Yields[0])
			},
			paths: [][]string{{"yields", "north", "cut_fuel"}},
		},
		{
			name: "two groups for one region and role",
			mutate: func(c *economysim.Config) {
				c.Groups = append(c.Groups, c.Groups[0])
			},
			paths: [][]string{{"groups", "north", "cook"}},
		},
		{
			name: "two seed prices for one region and commodity",
			mutate: func(c *economysim.Config) {
				c.WarmStart.Prices = append(c.WarmStart.Prices, c.WarmStart.Prices[0])
			},
			paths: [][]string{{"warm_start", "prices", "north", "fuel"}},
		},
		{
			name: "second starting wage for one region",
			mutate: func(c *economysim.Config) {
				c.WarmStart.Wages = append(c.WarmStart.Wages, c.WarmStart.Wages[0])
			},
			paths: [][]string{{"warm_start", "wages", "north"}},
		},
		{
			name: "two warm groups for one region and role",
			mutate: func(c *economysim.Config) {
				c.WarmStart.Groups = append(c.WarmStart.Groups, c.WarmStart.Groups[0])
			},
			paths: [][]string{{"warm_start", "groups", "north", "cook"}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("Validate(%s) = nil, want %v at %v", tt.name, rule, tt.paths)
			}
			if !errors.Is(err, rule) {
				t.Errorf("errors.Is(Validate(%s), %v) = false, want true; error: %v", tt.name, rule, err)
			}
			var ve *economysim.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("errors.As(Validate(%s), *ValidationError) = false, want true", tt.name)
			}
			vs, other := violations(err)
			if len(other) != 0 {
				t.Errorf("Validate(%s) returned %d errors that are not a *ValidationError, want none: %v", tt.name, len(other), other)
			}
			for _, v := range vs {
				if !errors.Is(v, rule) {
					t.Errorf("Validate(%s) returned %v, want only %v", tt.name, v, rule)
				}
			}
			if got := pathsOf(vs); !reflect.DeepEqual(got, tt.paths) {
				t.Errorf("Validate(%s) paths = %q, want %q", tt.name, got, tt.paths)
			}
		})
	}
}

// TestValidate_UnknownReferenceRules verifies that a reference to a commodity,
// role, recipe or region that is not defined is the sentinel of that kind, with
// the path of the reference, wherever the slice's configuration refers to one.
//
// Requirement: FR-CFG-013, FR-VIA-041
func TestValidate_UnknownReferenceRules(t *testing.T) {
	t.Parallel()
	t.Run("commodity", func(t *testing.T) {
		t.Parallel()
		rule := economysim.ErrUnknownCommodity
		for _, tt := range []ruleCase{
			{
				name: "world upkeep",
				mutate: func(c *economysim.Config) {
					c.World.Upkeep = append(c.World.Upkeep, economysim.Item{Commodity: "nosuch", Quantity: 1})
				},
				paths: [][]string{{"world", "upkeep", "nosuch", "commodity"}},
			},
			{
				name: "recipe input",
				mutate: func(c *economysim.Config) {
					r := recipeOf(c, "refine")
					r.Inputs = append(r.Inputs, economysim.Item{Commodity: "nosuch", Quantity: 1})
				},
				paths: [][]string{{"recipes", "refine", "inputs", "nosuch", "commodity"}},
			},
			{
				name: "recipe output",
				mutate: func(c *economysim.Config) {
					r := recipeOf(c, "refine")
					r.Outputs = append(r.Outputs, economysim.Output{Commodity: "nosuch", Quantity: 1, Discard: true})
				},
				paths: [][]string{{"recipes", "refine", "outputs", "nosuch", "commodity"}},
			},
			{
				name: "recipe tool",
				mutate: func(c *economysim.Config) {
					r := recipeOf(c, "refine")
					r.Requires = append(r.Requires, economysim.Tool{Commodity: "nosuch", WearPerUse: 1})
				},
				paths: [][]string{{"recipes", "refine", "requires", "nosuch", "commodity"}},
			},
			{
				name: "role upkeep",
				mutate: func(c *economysim.Config) {
					r := roleOf(c, "cook")
					r.Upkeep = append(r.Upkeep, economysim.Item{Commodity: "nosuch", Quantity: 1})
				},
				paths: [][]string{{"roles", "cook", "upkeep", "nosuch", "commodity"}},
			},
			{
				name: "role trades",
				mutate: func(c *economysim.Config) {
					r := roleOf(c, "visitor")
					r.Trades = append(r.Trades, "nosuch")
				},
				paths: [][]string{{"roles", "visitor", "trades", "nosuch"}},
			},
			{
				name: "role start stock",
				mutate: func(c *economysim.Config) {
					r := roleOf(c, "cook")
					r.StartStock = append(r.StartStock, economysim.Item{Commodity: "nosuch", Quantity: 1})
				},
				paths: [][]string{{"roles", "cook", "start_stock", "nosuch", "commodity"}},
			},
			{
				name: "region extra upkeep",
				mutate: func(c *economysim.Config) {
					c.Regions[0].ExtraUpkeep = []economysim.Item{{Commodity: "nosuch", Quantity: 1}}
				},
				paths: [][]string{{"regions", "north", "extra_upkeep", "nosuch", "commodity"}},
			},
			{
				name: "seed price",
				mutate: func(c *economysim.Config) {
					c.WarmStart.Prices = append(c.WarmStart.Prices, economysim.SeedPrice{Region: "north", Commodity: "nosuch", Price: 100})
				},
				paths: [][]string{{"warm_start", "prices", "north", "nosuch", "commodity"}},
			},
			{
				name: "warm group stock",
				mutate: func(c *economysim.Config) {
					g := &c.WarmStart.Groups[0]
					g.Stock = append(g.Stock, economysim.Item{Commodity: "nosuch", Quantity: 1})
				},
				paths: [][]string{{"warm_start", "groups", "north", "cook", "stock", "nosuch", "commodity"}},
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := referenceConfig()
				tt.mutate(&c)
				err := c.Validate()
				if err == nil {
					t.Fatalf("Validate(%s) = nil, want %v at %v", tt.name, rule, tt.paths)
				}
				if !errors.Is(err, rule) {
					t.Errorf("errors.Is(Validate(%s), %v) = false, want true; error: %v", tt.name, rule, err)
				}
				var ve *economysim.ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("errors.As(Validate(%s), *ValidationError) = false, want true", tt.name)
				}
				vs, other := violations(err)
				if len(other) != 0 {
					t.Errorf("Validate(%s) returned %d errors that are not a *ValidationError, want none: %v", tt.name, len(other), other)
				}
				for _, v := range vs {
					if !errors.Is(v, rule) {
						t.Errorf("Validate(%s) returned %v, want only %v", tt.name, v, rule)
					}
				}
				if got := pathsOf(vs); !reflect.DeepEqual(got, tt.paths) {
					t.Errorf("Validate(%s) paths = %q, want %q", tt.name, got, tt.paths)
				}
			})
		}
	})
	t.Run("role", func(t *testing.T) {
		t.Parallel()
		rule := economysim.ErrUnknownRole
		for _, tt := range []ruleCase{
			{
				name: "group",
				mutate: func(c *economysim.Config) {
					c.Groups = append(c.Groups, economysim.Group{Region: "north", Role: "ghost", Households: 1})
				},
				paths: [][]string{{"groups", "north", "ghost", "role"}},
			},
			{
				name: "warm group",
				mutate: func(c *economysim.Config) {
					c.WarmStart.Groups = append(c.WarmStart.Groups, economysim.WarmGroup{Region: "north", Role: "ghost", Money: 1})
				},
				paths: [][]string{{"warm_start", "groups", "north", "ghost", "role"}},
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := referenceConfig()
				tt.mutate(&c)
				err := c.Validate()
				if err == nil {
					t.Fatalf("Validate(%s) = nil, want %v at %v", tt.name, rule, tt.paths)
				}
				if !errors.Is(err, rule) {
					t.Errorf("errors.Is(Validate(%s), %v) = false, want true; error: %v", tt.name, rule, err)
				}
				var ve *economysim.ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("errors.As(Validate(%s), *ValidationError) = false, want true", tt.name)
				}
				vs, other := violations(err)
				if len(other) != 0 {
					t.Errorf("Validate(%s) returned %d errors that are not a *ValidationError, want none: %v", tt.name, len(other), other)
				}
				for _, v := range vs {
					if !errors.Is(v, rule) {
						t.Errorf("Validate(%s) returned %v, want only %v", tt.name, v, rule)
					}
				}
				if got := pathsOf(vs); !reflect.DeepEqual(got, tt.paths) {
					t.Errorf("Validate(%s) paths = %q, want %q", tt.name, got, tt.paths)
				}
			})
		}
	})
	t.Run("recipe", func(t *testing.T) {
		t.Parallel()
		rule := economysim.ErrUnknownRecipe
		for _, tt := range []ruleCase{
			{
				name: "role list",
				mutate: func(c *economysim.Config) {
					r := roleOf(c, "cook")
					r.Recipes = append(r.Recipes, "nosuch")
				},
				paths: [][]string{{"roles", "cook", "recipes", "nosuch"}},
			},
			{
				name: "yield profile",
				mutate: func(c *economysim.Config) {
					c.Yields = append(c.Yields, economysim.YieldProfile{Region: "north", Recipe: "nosuch", HasAll: true, All: 10_000})
				},
				paths: [][]string{{"yields", "north", "nosuch", "recipe"}},
			},
			{
				name: "warm job",
				mutate: func(c *economysim.Config) {
					g := &c.WarmStart.Groups[0]
					g.Jobs = append(g.Jobs, economysim.JobSpec{Recipe: "nosuch", InProgressBP: 1})
				},
				paths: [][]string{{"warm_start", "groups", "north", "cook", "jobs", "nosuch", "recipe"}},
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := referenceConfig()
				tt.mutate(&c)
				err := c.Validate()
				if err == nil {
					t.Fatalf("Validate(%s) = nil, want %v at %v", tt.name, rule, tt.paths)
				}
				if !errors.Is(err, rule) {
					t.Errorf("errors.Is(Validate(%s), %v) = false, want true; error: %v", tt.name, rule, err)
				}
				var ve *economysim.ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("errors.As(Validate(%s), *ValidationError) = false, want true", tt.name)
				}
				vs, other := violations(err)
				if len(other) != 0 {
					t.Errorf("Validate(%s) returned %d errors that are not a *ValidationError, want none: %v", tt.name, len(other), other)
				}
				for _, v := range vs {
					if !errors.Is(v, rule) {
						t.Errorf("Validate(%s) returned %v, want only %v", tt.name, v, rule)
					}
				}
				if got := pathsOf(vs); !reflect.DeepEqual(got, tt.paths) {
					t.Errorf("Validate(%s) paths = %q, want %q", tt.name, got, tt.paths)
				}
			})
		}
	})
	t.Run("region", func(t *testing.T) {
		t.Parallel()
		rule := economysim.ErrUnknownRegion
		for _, tt := range []ruleCase{
			{
				name: "group",
				mutate: func(c *economysim.Config) {
					c.Groups = append(c.Groups, economysim.Group{Region: "nowhere", Role: "visitor", Households: 1})
				},
				paths: [][]string{{"groups", "nowhere", "visitor", "region"}},
			},
			{
				name: "yield profile",
				mutate: func(c *economysim.Config) {
					c.Yields = append(c.Yields, economysim.YieldProfile{Region: "nowhere", Recipe: "cut_fuel", HasAll: true, All: 10_000})
				},
				paths: [][]string{{"yields", "nowhere", "cut_fuel", "region"}},
			},
			{
				name: "yield profile past regions times recipes",
				mutate: func(c *economysim.Config) {
					for _, r := range c.Recipes {
						if !slices.ContainsFunc(c.Yields, func(y economysim.YieldProfile) bool { return y.Recipe == r.ID }) {
							c.Yields = append(c.Yields, economysim.YieldProfile{Region: "north", Recipe: r.ID, HasAll: true, All: 10_000})
						}
					}
					c.Yields = append(c.Yields, economysim.YieldProfile{Region: "nowhere", Recipe: "cut_fuel", HasAll: true, All: 10_000})
				},
				paths: [][]string{{"yields", "nowhere", "cut_fuel", "region"}},
			},
			{
				name: "seed price",
				mutate: func(c *economysim.Config) {
					c.WarmStart.Prices = append(c.WarmStart.Prices, economysim.SeedPrice{Region: "nowhere", Commodity: "fuel", Price: 179})
				},
				paths: [][]string{{"warm_start", "prices", "nowhere", "fuel", "region"}},
			},
			{
				name: "starting wage",
				mutate: func(c *economysim.Config) {
					c.WarmStart.Wages = append(c.WarmStart.Wages, economysim.SeedWage{Region: "nowhere", Wage: 1})
				},
				paths: [][]string{{"warm_start", "wages", "nowhere", "region"}},
			},
			{
				name: "pool",
				mutate: func(c *economysim.Config) {
					c.WarmStart.Pools = append(c.WarmStart.Pools, economysim.WarmPool{Region: "nowhere", Money: 1})
				},
				paths: [][]string{{"warm_start", "pools", "nowhere", "region"}},
			},
			{
				name: "warm group",
				mutate: func(c *economysim.Config) {
					c.WarmStart.Groups = append(c.WarmStart.Groups, economysim.WarmGroup{Region: "nowhere", Role: "visitor", Money: 1})
				},
				paths: [][]string{{"warm_start", "groups", "nowhere", "visitor", "region"}},
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := referenceConfig()
				tt.mutate(&c)
				err := c.Validate()
				if err == nil {
					t.Fatalf("Validate(%s) = nil, want %v at %v", tt.name, rule, tt.paths)
				}
				if !errors.Is(err, rule) {
					t.Errorf("errors.Is(Validate(%s), %v) = false, want true; error: %v", tt.name, rule, err)
				}
				var ve *economysim.ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("errors.As(Validate(%s), *ValidationError) = false, want true", tt.name)
				}
				vs, other := violations(err)
				if len(other) != 0 {
					t.Errorf("Validate(%s) returned %d errors that are not a *ValidationError, want none: %v", tt.name, len(other), other)
				}
				for _, v := range vs {
					if !errors.Is(v, rule) {
						t.Errorf("Validate(%s) returned %v, want only %v", tt.name, v, rule)
					}
				}
				if got := pathsOf(vs); !reflect.DeepEqual(got, tt.paths) {
					t.Errorf("Validate(%s) paths = %q, want %q", tt.name, got, tt.paths)
				}
			})
		}
	})
}

// TestValidate_RecipeOutputRule verifies that a recipe with no output, and one
// whose outputs are all discarded, is ErrRecipeNoOutput.
//
// Requirement: FR-CFG-014, FR-PRO-003
func TestValidate_RecipeOutputRule(t *testing.T) {
	t.Parallel()
	rule := economysim.ErrRecipeNoOutput
	for _, tt := range []ruleCase{
		{
			// A new recipe, so no commodity loses its only source.
			name: "no output",
			mutate: func(c *economysim.Config) {
				c.Recipes = append(c.Recipes, economysim.Recipe{
					ID: "idle", Duration: 1, TraineeYield: 5_000, Inputs: []economysim.Item{{Commodity: "staple", Quantity: 1}},
				})
			},
			paths: [][]string{{"recipes", "idle", "outputs"}},
		},
		{
			name: "only a discard output",
			mutate: func(c *economysim.Config) {
				c.Recipes = append(c.Recipes, economysim.Recipe{
					ID: "burn", Duration: 1, TraineeYield: 5_000, Inputs: []economysim.Item{{Commodity: "staple", Quantity: 1}},
					Outputs: []economysim.Output{{Commodity: "waste", Quantity: 1, Discard: true}},
				})
			},
			paths: [][]string{{"recipes", "burn", "outputs"}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("Validate(%s) = nil, want %v at %v", tt.name, rule, tt.paths)
			}
			if !errors.Is(err, rule) {
				t.Errorf("errors.Is(Validate(%s), %v) = false, want true; error: %v", tt.name, rule, err)
			}
			var ve *economysim.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("errors.As(Validate(%s), *ValidationError) = false, want true", tt.name)
			}
			vs, other := violations(err)
			if len(other) != 0 {
				t.Errorf("Validate(%s) returned %d errors that are not a *ValidationError, want none: %v", tt.name, len(other), other)
			}
			for _, v := range vs {
				if !errors.Is(v, rule) {
					t.Errorf("Validate(%s) returned %v, want only %v", tt.name, v, rule)
				}
			}
			if got := pathsOf(vs); !reflect.DeepEqual(got, tt.paths) {
				t.Errorf("Validate(%s) paths = %q, want %q", tt.name, got, tt.paths)
			}
		})
	}
}

// TestValidate_OrphanCommodityRule verifies that a commodity that no recipe
// makes or uses and that no upkeep or tool mentions is ErrOrphanCommodity.
//
// Requirement: FR-CFG-014
func TestValidate_OrphanCommodityRule(t *testing.T) {
	t.Parallel()
	rule := economysim.ErrOrphanCommodity
	for _, tt := range []ruleCase{
		{
			name: "mentioned by nothing",
			mutate: func(c *economysim.Config) {
				d := economysim.DefaultCommodity()
				d.ID, d.ReferencePrice = "dust", 1
				c.Commodities = append(c.Commodities, d)
			},
			paths: [][]string{{"commodities", "dust"}},
		},
		{
			// Raw says the commodity needs no recipe; it does not say anything
			// uses it.
			name: "raw but used by nothing",
			mutate: func(c *economysim.Config) {
				d := economysim.DefaultCommodity()
				d.ID, d.ReferencePrice, d.Raw = "dust", 1, true
				c.Commodities = append(c.Commodities, d)
			},
			paths: [][]string{{"commodities", "dust"}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("Validate(%s) = nil, want %v at %v", tt.name, rule, tt.paths)
			}
			if !errors.Is(err, rule) {
				t.Errorf("errors.Is(Validate(%s), %v) = false, want true; error: %v", tt.name, rule, err)
			}
			var ve *economysim.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("errors.As(Validate(%s), *ValidationError) = false, want true", tt.name)
			}
			vs, other := violations(err)
			if len(other) != 0 {
				t.Errorf("Validate(%s) returned %d errors that are not a *ValidationError, want none: %v", tt.name, len(other), other)
			}
			for _, v := range vs {
				if !errors.Is(v, rule) {
					t.Errorf("Validate(%s) returned %v, want only %v", tt.name, v, rule)
				}
			}
			if got := pathsOf(vs); !reflect.DeepEqual(got, tt.paths) {
				t.Errorf("Validate(%s) paths = %q, want %q", tt.name, got, tt.paths)
			}
		})
	}
}

// TestValidate_UnproducibleRule verifies the producibility fixpoint of
// spec-config.md 7.5: a commodity that is consumed and cannot be produced is
// ErrUnproducible, a cycle with no source is named in the path in ID order, and
// Raw or a recipe with no inputs is a source.
//
// Requirement: FR-CFG-015, FR-GDS-018, FR-PRO-003
func TestValidate_UnproducibleRule(t *testing.T) {
	t.Parallel()
	// loopWith adds loop_a and loop_b, each made from the other, and when
	// start is true a recipe that makes loop_a from nothing.
	loopWith := func(c *economysim.Config, start bool) {
		a, b := economysim.DefaultCommodity(), economysim.DefaultCommodity()
		a.ID, a.ReferencePrice = "loop_a", 1
		b.ID, b.ReferencePrice = "loop_b", 1
		c.Commodities = append(c.Commodities, a, b)
		ra, rb := economysim.DefaultRecipe(), economysim.DefaultRecipe()
		ra.ID, rb.ID = "make_loop_a", "make_loop_b"
		ra.Inputs = []economysim.Item{{Commodity: "loop_b", Quantity: 1}}
		ra.Outputs = []economysim.Output{{Commodity: "loop_a", Quantity: 1, CostShare: 10_000}}
		rb.Inputs = []economysim.Item{{Commodity: "loop_a", Quantity: 1}}
		rb.Outputs = []economysim.Output{{Commodity: "loop_b", Quantity: 1, CostShare: 10_000}}
		c.Recipes = append(c.Recipes, ra, rb)
		if start {
			rs := economysim.DefaultRecipe()
			rs.ID = "start_loop"
			rs.Outputs = []economysim.Output{{Commodity: "loop_a", Quantity: 1, CostShare: 10_000}}
			c.Recipes = append(c.Recipes, rs)
		}
	}
	loop := func(c *economysim.Config) { loopWith(c, false) }
	orePlusUpkeep := func(raw bool) func(c *economysim.Config) {
		return func(c *economysim.Config) {
			d := economysim.DefaultCommodity()
			d.ID, d.ReferencePrice, d.Raw = "ore_x", 100, raw
			c.Commodities = append(c.Commodities, d)
			c.World.Upkeep = append(c.World.Upkeep, economysim.Item{Commodity: "ore_x", Quantity: 1})
		}
	}

	rule := economysim.ErrUnproducible
	for _, tt := range []ruleCase{
		{
			// Two goods that each need the other: both are consumed, neither can
			// start. The path names the witness chain from each, ending when a
			// commodity repeats.
			name:   "cycle with no source",
			mutate: loop,
			paths: [][]string{
				{"commodities", "loop_a", "make_loop_a", "loop_b", "make_loop_b", "loop_a"},
				{"commodities", "loop_b", "make_loop_b", "loop_a", "make_loop_a", "loop_b"},
			},
		},
		{
			name:   "consumed and made by no recipe",
			mutate: orePlusUpkeep(false),
			paths:  [][]string{{"commodities", "ore_x"}},
		},
		{
			// A tool that is made from a good only the tool can make: a source
			// recipe that needs a tool made from its own product (spec-production.md
			// 10.6). "ring" is made by forge from ingot and forge is the only way
			// to ingot, which needs the ring.
			name: "tool made from its own product",
			mutate: func(c *economysim.Config) {
				ring, ingot := economysim.DefaultCommodity(), economysim.DefaultCommodity()
				ring.ID, ring.ReferencePrice, ring.Durability = "ring", 1, 5
				ingot.ID, ingot.ReferencePrice = "ingot", 1
				c.Commodities = append(c.Commodities, ring, ingot)
				forge, smelt := economysim.DefaultRecipe(), economysim.DefaultRecipe()
				forge.ID, smelt.ID = "forge", "smelt"
				forge.Inputs = []economysim.Item{{Commodity: "ingot", Quantity: 1}}
				forge.Outputs = []economysim.Output{{Commodity: "ring", Quantity: 1, CostShare: 10_000}}
				smelt.Requires = []economysim.Tool{{Commodity: "ring", WearPerUse: 1}}
				smelt.Outputs = []economysim.Output{{Commodity: "ingot", Quantity: 1, CostShare: 10_000}}
				c.Recipes = append(c.Recipes, forge, smelt)
			},
			paths: [][]string{
				{"commodities", "ingot", "smelt", "ring", "forge", "ingot"},
				{"commodities", "ring", "forge", "ingot", "smelt", "ring"},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("Validate(%s) = nil, want %v at %v", tt.name, rule, tt.paths)
			}
			if !errors.Is(err, rule) {
				t.Errorf("errors.Is(Validate(%s), %v) = false, want true; error: %v", tt.name, rule, err)
			}
			var ve *economysim.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("errors.As(Validate(%s), *ValidationError) = false, want true", tt.name)
			}
			vs, other := violations(err)
			if len(other) != 0 {
				t.Errorf("Validate(%s) returned %d errors that are not a *ValidationError, want none: %v", tt.name, len(other), other)
			}
			for _, v := range vs {
				if !errors.Is(v, rule) {
					t.Errorf("Validate(%s) returned %v, want only %v", tt.name, v, rule)
				}
			}
			if got := pathsOf(vs); !reflect.DeepEqual(got, tt.paths) {
				t.Errorf("Validate(%s) paths = %q, want %q", tt.name, got, tt.paths)
			}
		})
	}

	t.Run("reasons", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name   string
			mutate func(c *economysim.Config)
			want   string
		}{
			{"cycle", loop, "cycle with no source"},
			{"no recipe", orePlusUpkeep(false), "no recipe makes it"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := referenceConfig()
				tt.mutate(&c)
				vs, _ := violations(c.Validate())
				if len(vs) == 0 || vs[0].Value != tt.want {
					t.Errorf("Validate(%s) first value = %+v, want %q", tt.name, vs, tt.want)
				}
			})
		}
	})

	t.Run("sources", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name   string
			mutate func(c *economysim.Config)
		}{
			{"raw commodity is sourced", orePlusUpkeep(true)},
			{
				// A recipe with no inputs is a source (FR-PRO-003).
				"recipe with no inputs is a source",
				func(c *economysim.Config) {
					d := economysim.DefaultCommodity()
					d.ID, d.ReferencePrice = "ore_x", 100
					c.Commodities = append(c.Commodities, d)
					c.World.Upkeep = append(c.World.Upkeep, economysim.Item{Commodity: "ore_x", Quantity: 1})
					r := economysim.DefaultRecipe()
					r.ID = "mine_x"
					r.Outputs = []economysim.Output{{Commodity: "ore_x", Quantity: 1, CostShare: 10_000}}
					c.Recipes = append(c.Recipes, r)
				},
			},
			{
				// The cycle of the first case is fine once it has a way in: a
				// recipe that makes loop_a from nothing.
				"cycle with a way in",
				func(c *economysim.Config) { loopWith(c, true) },
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := referenceConfig()
				tt.mutate(&c)
				if err := c.Validate(); err != nil {
					t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
				}
			})
		}
	})
}

// TestValidate_NoRegionsRule verifies that a configuration with no region is
// ErrNoRegions and that one region with no edges is valid.
//
// Requirement: FR-REG-001
func TestValidate_NoRegionsRule(t *testing.T) {
	t.Parallel()
	rule := economysim.ErrNoRegions
	for _, tt := range []ruleCase{
		{
			name: "no region",
			mutate: func(c *economysim.Config) {
				// The rows that refer to the region go with it, so only the
				// absence of any region is wrong.
				c.Regions, c.Groups, c.Yields, c.WarmStart = nil, nil, nil, nil
			},
			paths: [][]string{{"regions"}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("Validate(%s) = nil, want %v at %v", tt.name, rule, tt.paths)
			}
			if !errors.Is(err, rule) {
				t.Errorf("errors.Is(Validate(%s), %v) = false, want true; error: %v", tt.name, rule, err)
			}
			var ve *economysim.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("errors.As(Validate(%s), *ValidationError) = false, want true", tt.name)
			}
			vs, other := violations(err)
			if len(other) != 0 {
				t.Errorf("Validate(%s) returned %d errors that are not a *ValidationError, want none: %v", tt.name, len(other), other)
			}
			for _, v := range vs {
				if !errors.Is(v, rule) {
					t.Errorf("Validate(%s) returned %v, want only %v", tt.name, v, rule)
				}
			}
			if got := pathsOf(vs); !reflect.DeepEqual(got, tt.paths) {
				t.Errorf("Validate(%s) paths = %q, want %q", tt.name, got, tt.paths)
			}
		})
	}
	// With the yield profiles kept, ErrNoRegions is still reported (the
	// yields then also name an unknown region), not hidden by a count error.
	t.Run("no region, yield profiles kept", func(t *testing.T) {
		t.Parallel()
		kept := referenceConfig()
		kept.Regions, kept.Groups, kept.WarmStart = nil, nil, nil
		err := kept.Validate()
		vs, _ := violations(err)
		if !errors.Is(err, economysim.ErrNoRegions) || len(vs) == 0 || !reflect.DeepEqual(vs[0].Path, []string{"regions"}) {
			t.Errorf("Validate(no region, yields kept) = %v, want ErrNoRegions first at [regions]", err)
		}
	})
	c := referenceConfig()
	if len(c.Regions) != 1 {
		t.Fatalf("reference configuration has %d regions, want 1", len(c.Regions))
	}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(one region, no edges) = %v, want nil", err)
	}
}

// TestValidate_CountBounds verifies that a collection over its bound is
// ErrTooMany with its path, value and bound, and that it is the only error
// returned, whatever else is wrong: the other rules do not run
// (spec-config.md 7.1.6).
//
// Requirement: FR-CFG-017, FR-CFG-101
func TestValidate_CountBounds(t *testing.T) {
	t.Parallel()
	manyCommodities := func(n int) []economysim.Commodity {
		out := make([]economysim.Commodity, n)
		for i := range out {
			// Every ID is invalid and repeated, so any rule that ran would speak.
			out[i] = economysim.Commodity{ID: "BAD", ReferencePrice: 1}
		}
		return out
	}
	tests := []struct {
		name      string
		mutate    func(c *economysim.Config)
		wantPath  []string
		wantValue string
		wantBound string
	}{
		{
			name:      "commodities",
			mutate:    func(c *economysim.Config) { c.Commodities = manyCommodities(economysim.MaxCommodities + 1) },
			wantPath:  []string{"commodities"},
			wantValue: "1001", wantBound: "1000",
		},
		{
			// 1 001 roles, each unknown to nothing and with a duplicate ID.
			name: "roles",
			mutate: func(c *economysim.Config) {
				c.Roles = make([]economysim.Role, economysim.MaxRoles+1)
			},
			wantPath:  []string{"roles"},
			wantValue: "1001", wantBound: "1000",
		},
		{
			name: "months",
			mutate: func(c *economysim.Config) {
				c.Calendar.Months = make([]economysim.Month, economysim.MaxMonths+1)
			},
			wantPath:  []string{"calendar", "months"},
			wantValue: "101", wantBound: "100",
		},
		{
			name: "world upkeep",
			mutate: func(c *economysim.Config) {
				c.World.Upkeep = make([]economysim.Item, economysim.MaxUpkeepItems+1)
			},
			wantPath:  []string{"world", "upkeep"},
			wantValue: "17", wantBound: "16",
		},
		{
			name: "recipe inputs",
			mutate: func(c *economysim.Config) {
				recipeOf(c, "refine").Inputs = make([]economysim.Item, economysim.MaxRecipeItems+1)
			},
			wantPath:  []string{"recipes", "refine", "inputs"},
			wantValue: "17", wantBound: "16",
		},
		{
			name: "recipe tools",
			mutate: func(c *economysim.Config) {
				recipeOf(c, "refine").Requires = make([]economysim.Tool, economysim.MaxRequires+1)
			},
			wantPath:  []string{"recipes", "refine", "requires"},
			wantValue: "5", wantBound: "4",
		},
		{
			name: "warm start prices",
			mutate: func(c *economysim.Config) {
				c.WarmStart.Prices = make([]economysim.SeedPrice, economysim.MaxSeedPrices+1)
			},
			wantPath:  []string{"warm_start", "prices"},
			wantValue: "1000001", wantBound: "1000000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			// Other mistakes that the later phases would report.
			c.Roles[0].ID = "BAD"
			c.Groups = append(c.Groups, economysim.Group{Region: "nowhere", Role: "ghost"})
			tt.mutate(&c)
			err := c.Validate()
			if !errors.Is(err, economysim.ErrTooMany) {
				t.Fatalf("errors.Is(Validate(%s), ErrTooMany) = false, want true; error: %v", tt.name, err)
			}
			vs, other := violations(err)
			if len(vs) != 1 || len(other) != 0 {
				t.Fatalf("Validate(%s) = %d errors and %d others, want exactly one ErrTooMany: %v", tt.name, len(vs), len(other), err)
			}
			if !slices.Equal(vs[0].Path, tt.wantPath) || vs[0].Value != tt.wantValue || vs[0].Bound != tt.wantBound {
				t.Errorf("Validate(%s) = path %q value %q bound %q, want path %q value %q bound %q",
					tt.name, vs[0].Path, vs[0].Value, vs[0].Bound, tt.wantPath, tt.wantValue, tt.wantBound)
			}
		})
	}

	t.Run("at the bound is valid", func(t *testing.T) {
		t.Parallel()
		c := referenceConfig()
		c.World.Upkeep = nil
		for i := 0; i < economysim.MaxUpkeepItems; i++ {
			// The list holds the same commodity 16 times, which is a duplicate
			// and not a count; only that the count itself is allowed is under test.
			c.World.Upkeep = append(c.World.Upkeep, economysim.Item{Commodity: "fuel", Quantity: 1})
		}
		if errors.Is(c.Validate(), economysim.ErrTooMany) {
			t.Errorf("Validate(%d upkeep items) is ErrTooMany, want the bound itself to be allowed", economysim.MaxUpkeepItems)
		}
	})
}

// TestValidate_ErrorsAreSortedByPathThenRule verifies that the errors come in
// the order of path and then of rule, whatever order the configuration or the
// rules found them in, and that two rules at one path are ordered by rule.
//
// Requirement: FR-CFG-012
func TestValidate_ErrorsAreSortedByPathThenRule(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	// Found by phase 3 and phase 4 in an order that is not the order of paths.
	c.Groups = append(c.Groups, economysim.Group{Region: "zone", Role: "ghost"})
	c.Commodities = append(c.Commodities,
		economysim.Commodity{ID: "BAD", ReferencePrice: 1}, economysim.Commodity{ID: "BAD", ReferencePrice: 1})
	c.Regions[0].ExtraUpkeep = []economysim.Item{{Commodity: "nosuch", Quantity: 1}}
	err := c.Validate()
	vs, other := violations(err)
	if len(other) != 0 || len(vs) < 5 {
		t.Fatalf("Validate = %d errors and %d others, want at least 5 errors: %v", len(vs), len(other), err)
	}
	less := func(a, b *economysim.ValidationError) bool {
		if c := slices.Compare(a.Path, b.Path); c != 0 {
			return c < 0
		}
		return a.Rule.Error() <= b.Rule.Error()
	}
	for i := 1; i < len(vs); i++ {
		if !less(vs[i-1], vs[i]) {
			t.Errorf("error %d (%v) comes before error %d (%v), want the order of path then rule", i-1, vs[i-1], i, vs[i])
		}
	}
	// The path {"commodities","BAD","id"} holds ErrDuplicateID once and
	// ErrInvalidID twice: the rule decides their order.
	var rules []string
	for _, v := range vs {
		if slices.Equal(v.Path, []string{"commodities", "BAD", "id"}) {
			rules = append(rules, v.Rule.Error())
		}
	}
	if want := []string{"duplicate ID", "invalid ID", "invalid ID"}; !slices.Equal(rules, want) {
		t.Errorf("rules at commodities.BAD.id = %q, want %q", rules, want)
	}
}

// TestValidate_IsDeterministic verifies that two runs on the same value, and
// one on a copy, give identical text and identical typed errors.
//
// Requirement: FR-CFG-012, NFR-CFG-003
func TestValidate_IsDeterministic(t *testing.T) {
	t.Parallel()
	build := func() economysim.Config {
		c := referenceConfig()
		c.Groups = append(c.Groups, economysim.Group{Region: "zone", Role: "ghost"})
		c.Roles = append(c.Roles, economysim.Role{ID: "Bad"}, economysim.Role{ID: "Bad"})
		return c
	}
	first, second := build(), build()
	want := first.Validate()
	if want == nil {
		t.Fatalf("Validate(broken) = nil, want errors")
	}
	for i := 0; i < 5; i++ {
		got := first.Validate()
		if got.Error() != want.Error() {
			t.Fatalf("run %d: Validate text = %q, want %q", i, got.Error(), want.Error())
		}
	}
	if got := second.Validate(); !reflect.DeepEqual(got, want) {
		t.Errorf("Validate(copy) = %v, want %v", got, want)
	}
}

// TestValidate_ReportIsCapped verifies that at most 100 errors are returned,
// the first 100 in sorted order, followed by one ErrTooManyErrors that says how
// many were left out, and that exactly 100 errors have no such entry.
//
// Requirement: FR-CFG-011
func TestValidate_ReportIsCapped(t *testing.T) {
	t.Parallel()
	withUnknownRegions := func(n int) economysim.Config {
		c := referenceConfig()
		for i := 0; i < n; i++ {
			c.Groups = append(c.Groups, economysim.Group{Region: fmt.Sprintf("zone%03d", i), Role: "visitor", Households: 1})
		}
		return c
	}

	t.Run("150 errors", func(t *testing.T) {
		t.Parallel()
		err := withUnknownRegions(150).Validate()
		vs, other := violations(err)
		if len(vs) != 100 || len(other) != 1 {
			t.Fatalf("Validate(150 errors) = %d errors and %d others, want 100 and 1", len(vs), len(other))
		}
		if !errors.Is(other[0], economysim.ErrTooManyErrors) || !errors.Is(err, economysim.ErrTooManyErrors) {
			t.Errorf("the last entry = %v, want ErrTooManyErrors", other[0])
		}
		if !strings.Contains(other[0].Error(), "50") {
			t.Errorf("ErrTooManyErrors text = %q, want it to say 50 were left out", other[0])
		}
		// Sorted by path, so the 100 kept are zone000 to zone099.
		if got := vs[99].Path[1]; got != "zone099" {
			t.Errorf("last kept error is for %q, want zone099", got)
		}
		if last := err.(interface{ Unwrap() []error }).Unwrap(); last[len(last)-1] != other[0] {
			t.Errorf("ErrTooManyErrors is not the last error")
		}
	})
	t.Run("101 errors", func(t *testing.T) {
		t.Parallel()
		err := withUnknownRegions(101).Validate()
		vs, other := violations(err)
		if len(vs) != 100 || len(other) != 1 || !strings.Contains(other[0].Error(), "1 more") {
			t.Errorf("Validate(101 errors) = %d errors and others %v, want 100 and one entry saying 1 more", len(vs), other)
		}
	})
	t.Run("exactly 100 errors", func(t *testing.T) {
		t.Parallel()
		err := withUnknownRegions(100).Validate()
		vs, other := violations(err)
		if len(vs) != 100 || len(other) != 0 || errors.Is(err, economysim.ErrTooManyErrors) {
			t.Errorf("Validate(100 errors) = %d errors and %d others, want 100 and no ErrTooManyErrors", len(vs), len(other))
		}
	})
}

// TestValidate_TextOfInputDerivedPaths verifies that an ID of hundreds of bytes
// and an ID with control characters give an error text whose segments are
// quoted and cut and that holds no control character, while the typed error
// keeps the ID as given.
//
// Requirement: FR-CFG-100
func TestValidate_TextOfInputDerivedPaths(t *testing.T) {
	t.Parallel()
	huge := strings.Repeat("é", 500) + "\x1b[31mred\x00"
	tests := []struct {
		name string
		id   string
	}{
		{"oversized", strings.Repeat("a", 5000)},
		{"control characters", "vis\x1b[2Jitor\n\x00"},
		{"oversized with multi-byte runes and control characters", huge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			roleOf(&c, "visitor").ID = tt.id
			err := c.Validate()
			var ve *economysim.ValidationError
			if !errors.As(err, &ve) || !errors.Is(err, economysim.ErrInvalidID) {
				t.Fatalf("Validate(%s) = %v, want ErrInvalidID as a *ValidationError", tt.name, err)
			}
			if ve.Path[1] != tt.id || ve.Value != tt.id {
				t.Errorf("Validate(%s) fields hold a cut ID, want the ID as given", tt.name)
			}
			text := err.Error()
			// 64 bytes at most are kept of each of the path segment and the
			// value; a byte takes at most four characters of escape, and the
			// rest of the text is a few words.
			if limit := 2*(4*economysim.MaxIDLength+8) + 200; len(text) > limit {
				t.Errorf("len(Error()) = %d, want at most %d", len(text), limit)
			}
			for _, r := range text {
				if unicode.IsControl(r) {
					t.Errorf("Error() holds the control character %U: %q", r, text)
					break
				}
			}
			if len(tt.id) > economysim.MaxIDLength && !strings.Contains(text, `"...`) {
				t.Errorf("Error() = %.100q, want a cut segment ending in quote and three dots", text)
			}
		})
	}
}

// TestValidate_ZeroValueEntityFails verifies that an entity that was never
// filled in (a zero value, as a file with only its header would give) fails
// validation and is not treated as the default.
//
// Requirement: FR-CFG-008
func TestValidate_ZeroValueEntityFails(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		add  func(c *economysim.Config)
	}{
		{"commodity", func(c *economysim.Config) { c.Commodities = append(c.Commodities, economysim.Commodity{}) }},
		{"recipe", func(c *economysim.Config) { c.Recipes = append(c.Recipes, economysim.Recipe{}) }},
		{"role", func(c *economysim.Config) { c.Roles = append(c.Roles, economysim.Role{}) }},
		{"region", func(c *economysim.Config) { c.Regions = append(c.Regions, economysim.Region{}) }},
		{"group", func(c *economysim.Config) { c.Groups = append(c.Groups, economysim.Group{}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.add(&c)
			if err := c.Validate(); err == nil {
				t.Errorf("Validate(reference + zero %s) = nil, want an error", tt.name)
			}
		})
	}
}
