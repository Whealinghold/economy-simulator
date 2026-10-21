package economysim

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// This file tests the viability rules of phase 6 of Validate on small
// configuration literals (plan 1-10 A9): the pack tests are viability_pack_test.go.
// Every case starts from viaToy, one region of ten farming households, and
// breaks one thing.

// viaToy returns a viable one-region configuration. A farmer household makes 4
// grain in one tick with no input, eats 1 a tick, and works 250 of 365 ticks, so
// the surplus factor at full labour is 4 (SurplusBP 40 000, labour share
// ceil(10^8 / 40 000) = 2 500) and 2.74 at the expected work (4 * 250 / 365).
func viaToy() Config {
	world := DefaultWorldParams()
	world.Upkeep = []Item{{Commodity: "grain", Quantity: 1}}
	grain := DefaultCommodity()
	grain.ID, grain.ReferencePrice = "grain", 100
	farm := DefaultRecipe()
	farm.ID = "farm"
	farm.Outputs = []Output{{Commodity: "grain", Quantity: 4, CostShare: 10_000}}
	role := DefaultRole()
	role.ID, role.Recipes = "farmer", []string{"farm"}
	region := DefaultRegion()
	region.ID, region.ReferenceBasket = "north", 100
	return Config{
		World:       world,
		Calendar:    DefaultCalendar(),
		Commodities: []Commodity{grain},
		Recipes:     []Recipe{farm},
		Roles:       []Role{role},
		Regions:     []Region{region},
		Groups:      []Group{{Region: "north", Role: "farmer", Households: 10}},
	}
}

// viaCommodity returns a commodity literal.
func viaCommodity(id string, price Price, edit func(*Commodity)) Commodity {
	c := DefaultCommodity()
	c.ID, c.ReferencePrice = id, price
	if edit != nil {
		edit(&c)
	}
	return c
}

// viaRecipe returns a recipe literal with the given inputs and outputs.
func viaRecipe(id string, in []Item, out []Output, edit func(*Recipe)) Recipe {
	r := DefaultRecipe()
	r.ID, r.Inputs, r.Outputs = id, in, out
	if edit != nil {
		edit(&r)
	}
	return r
}

// viaOut is a recipe output with the whole cost share.
func viaOut(commodity string, q Quantity) Output {
	return Output{Commodity: commodity, Quantity: q, CostShare: 10_000}
}

// viaSpring is the yield profile of a recipe that runs in spring only.
func viaSpring(region, recipe string) YieldProfile {
	return YieldProfile{Region: region, Recipe: recipe, Seasons: []SeasonYield{{Season: "spring", BP: 10_000}}}
}

// viaRun validates the configuration, which must pass the structural phases, and
// returns the viability errors in the order returned and the warnings of
// Warnings(). It fails the test on an error that is not a *ViabilityError.
func viaRun(t *testing.T, c Config) (errs []*ViabilityError, warns []Warning) {
	t.Helper()
	err := c.Validate()
	if err != nil {
		joined, ok := err.(interface{ Unwrap() []error })
		if !ok {
			joined = joinedOf(err)
		}
		for _, e := range joined.Unwrap() {
			var ve *ViabilityError
			if !errors.As(e, &ve) {
				t.Fatalf("Validate = %v, want only viability errors; got %v", err, e)
			}
			errs = append(errs, ve)
		}
	}
	return errs, c.Warnings()
}

// joinedOf wraps a single error as a list of one.
func joinedOf(err error) interface{ Unwrap() []error } {
	return errors.Join(err).(interface{ Unwrap() []error })
}

// viaWarned reports whether the warnings hold a rule at a path.
func viaWarned(ws []Warning, rule string, path ...string) bool {
	return slices.ContainsFunc(ws, func(w Warning) bool { return w.Rule == rule && slices.Equal(w.Path, path) })
}

// TestViaToy_IsViable verifies that the base of every case below validates and
// has the surplus factor its comment states, so a case that breaks one thing is
// not breaking two.
//
// Guard: every later case reads its expectation off this base; a base that
// stopped validating would turn each case into a test of the wrong thing.
func TestViaToy_IsViable(t *testing.T) {
	t.Parallel()
	c := viaToy()
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate(viaToy) = %v, want nil", err)
	}
	cc := c.canonical()
	m := newViaModel(&cc)
	res, skipped, err := m.solve(m.model(m.withHouseholds, steady.Ratio{Num: 1, Den: 1}))
	if err != nil || skipped {
		t.Fatalf("solve(viaToy) skipped = %v, error = %v, want a result", skipped, err)
	}
	// 10 households make 4 grain a tick each and eat 1: lambda = 4.
	if res.SurplusBP != 40_000 || !res.HasLabourShare || res.LabourShareBP != 2_500 {
		t.Errorf("solve(viaToy) = surplus %d, labour share %d (present %v), want 40 000 and 2 500", res.SurplusBP, res.LabourShareBP, res.HasLabourShare)
	}
}

// TestValidate_ViabilityRules verifies that each of the five error rules fires
// on a configuration that breaks only it, as a *ViabilityError that matches
// ErrNotViable, with the path, value and bound of spec-viability.md 10.1.
//
// Requirement: FR-VIA-004, FR-VIA-005, FR-VIA-006, FR-VIA-007, FR-VIA-008, FR-VIA-011, FR-CFG-033, FR-CFG-041
func TestValidate_ViabilityRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *Config)
		want   []ViabilityError
	}{
		{
			// ore is wanted by every household and made by a sited recipe with no
			// yield row, which exists nowhere: the structural pass (which ignores
			// yield) lets it through and V1 does not. V3 is gated off by V1, so
			// labour_short is not added.
			name: "unproducible_in_realm",
			mutate: func(c *Config) {
				c.Commodities = append(c.Commodities, viaCommodity("ore", 50, nil))
				c.Recipes = append(c.Recipes, viaRecipe("dig", nil, []Output{viaOut("ore", 1)}, func(r *Recipe) { r.Sited = true }))
				c.Roles[0].Recipes = []string{"dig", "farm"}
				c.World.Upkeep = append(c.World.Upkeep, Item{Commodity: "ore", Quantity: 1})
				c.Regions[0].ReferenceBasket = 150 // 100 grain + 50 ore
			},
			want: []ViabilityError{{Rule: "unproducible_in_realm", Path: []string{"commodities", "ore"}, Value: "ore"}},
		},
		{
			// A tool is an input of the fixpoint (4.2): the plough the farm requires is
			// forged from iron, iron is dug by a sited recipe that exists nowhere, so
			// the plough, the iron and the grain the farm makes without it are all
			// unproducible, in commodity order.
			name: "unproducible_in_realm through a tool",
			mutate: func(c *Config) {
				c.Commodities = append(c.Commodities,
					viaCommodity("iron", 50, nil),
					viaCommodity("plough", 50, func(m *Commodity) { m.Durability = 10 }))
				c.Recipes[0].Requires = []Tool{{Commodity: "plough", WearPerUse: 1}}
				c.Recipes = append(c.Recipes,
					viaRecipe("forge", []Item{{Commodity: "iron", Quantity: 1}}, []Output{viaOut("plough", 1)}, nil),
					viaRecipe("mine", nil, []Output{viaOut("iron", 1)}, func(r *Recipe) { r.Sited = true }))
				c.Roles[0].Recipes = []string{"farm", "forge", "mine"}
			},
			want: []ViabilityError{
				{Rule: "unproducible_in_realm", Path: []string{"commodities", "grain"}, Value: "grain"},
				{Rule: "unproducible_in_realm", Path: []string{"commodities", "iron"}, Value: "iron"},
				{Rule: "unproducible_in_realm", Path: []string{"commodities", "plough"}, Value: "plough"},
			},
		},
		{
			// idle lists a recipe that exists nowhere and has households where it
			// would have to run; the farmers are fine, so V3 still runs and passes.
			name: "role_cannot_work",
			mutate: func(c *Config) {
				c.Commodities = append(c.Commodities, viaCommodity("rock", 50, nil))
				c.Recipes = append(c.Recipes, viaRecipe("quarry", nil, []Output{viaOut("rock", 1)}, func(r *Recipe) { r.Sited = true }))
				idle := DefaultRole()
				idle.ID, idle.Recipes = "idle", []string{"quarry"}
				c.Roles = append(c.Roles, idle)
				c.Groups = append(c.Groups, Group{Region: "north", Role: "idle", Households: 5})
			},
			want: []ViabilityError{{Rule: "role_cannot_work", Path: []string{"roles", "idle"}, Value: "north"}},
		},
		{
			// seed is a Raw marker, so it is sourced; grow turns 2 seed into 5 grain
			// and thresh turns 5 grain into 1 seed. Per unit mix x of grow and y of
			// thresh: seed 2y... net seed y - 2x, net grain 5x - 5y; both at least s
			// needs x - y >= s/5 and y - 2x >= s, so -x >= 1.2 s and s* = 0.
			name: "cycle_not_productive",
			mutate: func(c *Config) {
				c.Commodities = append(c.Commodities, viaCommodity("seed", 50, func(m *Commodity) { m.Raw = true }))
				c.Recipes = []Recipe{
					viaRecipe("grow", []Item{{Commodity: "seed", Quantity: 2}}, []Output{viaOut("grain", 5)}, nil),
					viaRecipe("thresh", []Item{{Commodity: "grain", Quantity: 5}}, []Output{viaOut("seed", 1)}, nil),
				}
				c.Roles[0].Recipes = []string{"grow", "thresh"}
			},
			want: []ViabilityError{{Rule: "cycle_not_productive", Path: []string{"commodities", "grain"}, Value: "0", Bound: "0"}},
		},
		{
			// A farmer makes 1 grain in 2 ticks, 0.5 a tick against the 1 eaten.
			name: "labour_short",
			mutate: func(c *Config) {
				c.Recipes[0].Duration = 2
				c.Recipes[0].Outputs[0].Quantity = 1
			},
			want: []ViabilityError{{Rule: "labour_short", Path: []string{"regions"}, Value: "5000", Bound: "10000"}},
		},
		{
			// Grain lasts 100 ticks and is made in spring only: the 92 spring ticks
			// each complete a one-tick job, so the longest run without one is
			// 365 - 92 = 273. 8 a tick keeps the chain productive (lambda 2.02).
			name: "storage_gap",
			mutate: func(c *Config) {
				c.Commodities[0].ShelfLife = 100
				c.Recipes[0].Sited = true
				c.Recipes[0].Outputs[0].Quantity = 8
				c.Yields = []YieldProfile{viaSpring("north", "farm")}
			},
			want: []ViabilityError{{Rule: "storage_gap", Path: []string{"regions", "north", "commodities", "grain"}, Value: "273", Bound: "100"}},
		},
		{
			// The d5 numbers: a 60-tick job begun in spring (days 59 to 150) starts at
			// 59 and 119 and completes at 118 and 178, so the longest run without a
			// completion is 365 - 60 - 1 = 304 (counting every open tick would give
			// 273). 600 a job is 10 a tick, enough for the 1 eaten.
			name: "storage_gap of a 60-tick spring job",
			mutate: func(c *Config) {
				c.Commodities[0].ShelfLife = 200
				c.Recipes[0].Sited, c.Recipes[0].Duration = true, 60
				c.Recipes[0].Outputs[0].Quantity = 600
				c.Yields = []YieldProfile{viaSpring("north", "farm")}
			},
			want: []ViabilityError{{Rule: "storage_gap", Path: []string{"regions", "north", "commodities", "grain"}, Value: "304", Bound: "200"}},
		},
		{
			// Spring and autumn (days 243 to 333): starts 59, 119, 243, 303 complete
			// at 118, 178, 302, 362, and the longest run is 178 to 302, 123 ticks.
			name: "storage_gap with two plantings",
			mutate: func(c *Config) {
				c.Commodities[0].ShelfLife = 100
				c.Recipes[0].Sited, c.Recipes[0].Duration = true, 60
				c.Recipes[0].Outputs[0].Quantity = 600
				c.Yields = []YieldProfile{{Region: "north", Recipe: "farm", Seasons: []SeasonYield{{Season: "autumn", BP: 10_000}, {Season: "spring", BP: 10_000}}}}
			},
			want: []ViabilityError{{Rule: "storage_gap", Path: []string{"regions", "north", "commodities", "grain"}, Value: "123", Bound: "100"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := viaToy()
			tt.mutate(&c)
			got, _ := viaRun(t, c)
			if len(got) != len(tt.want) {
				t.Fatalf("Validate(%s) = %d viability errors %v, want %d", tt.name, len(got), got, len(tt.want))
			}
			for i, w := range tt.want {
				if !reflect.DeepEqual(*got[i], w) {
					t.Errorf("Validate(%s) error %d = %+v, want %+v", tt.name, i, *got[i], w)
				}
				if !errors.Is(got[i], ErrNotViable) {
					t.Errorf("errors.Is(%v, ErrNotViable) = false, want true", got[i])
				}
			}
			if err := c.Validate(); !errors.Is(err, ErrNotViable) {
				t.Errorf("errors.Is(Validate(%s), ErrNotViable) = false, want true: %v", tt.name, err)
			}
		})
	}
}

// TestValidate_StorageSkipsAnImportedGood verifies that an upkeep good no
// activity of a region makes has no supply gap there (V4, spec-viability.md 4.5
// as implemented): salt lasts 300 ticks, is made in the south in spring only (gap
// 273, within its shelf life) and is made nowhere in the north, which imports it.
// Counting the north as a whole-year gap (365 > 300) would reject the pack.
//
// Requirement: FR-VIA-008
func TestValidate_StorageSkipsAnImportedGood(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.Commodities = append(c.Commodities, viaCommodity("salt", 50, func(m *Commodity) { m.ShelfLife = 300 }))
	c.Recipes = append(c.Recipes, viaRecipe("mine", nil, []Output{viaOut("salt", 8)}, func(r *Recipe) { r.Sited = true }))
	miner := DefaultRole()
	miner.ID, miner.Recipes = "miner", []string{"mine"}
	c.Roles = append(c.Roles, miner)
	south := DefaultRegion()
	south.ID = "south"
	c.Regions = append(c.Regions, south)
	c.Regions[0].ReferenceBasket, c.Regions[1].ReferenceBasket = 150, 150
	c.World.Upkeep = append(c.World.Upkeep, Item{Commodity: "salt", Quantity: 1})
	c.Groups = append(c.Groups, Group{Region: "south", Role: "miner", Households: 10}, Group{Region: "south", Role: "farmer", Households: 10})
	c.Yields = []YieldProfile{viaSpring("south", "mine")}
	got, _ := viaRun(t, c)
	for _, e := range got {
		t.Errorf("Validate = %+v, want no viability error: the north makes no salt, so it has no gap to bridge", *e)
	}
}

// TestValidate_ViabilityErrorsAreSorted verifies that two rules in one
// configuration both appear, in the one sorted list: by path, then rule.
//
// Requirement: FR-CFG-041, FR-VIA-011
func TestValidate_ViabilityErrorsAreSorted(t *testing.T) {
	t.Parallel()
	c := viaToy()
	// storage_gap at regions/north/commodities/grain and role_cannot_work at roles/idle.
	c.Commodities[0].ShelfLife = 100
	c.Recipes[0].Sited = true
	c.Recipes[0].Outputs[0].Quantity = 8
	c.Yields = []YieldProfile{viaSpring("north", "farm")}
	c.Commodities = append(c.Commodities, viaCommodity("rock", 50, nil))
	c.Recipes = append(c.Recipes, viaRecipe("quarry", nil, []Output{viaOut("rock", 1)}, func(r *Recipe) { r.Sited = true }))
	idle := DefaultRole()
	idle.ID, idle.Recipes = "idle", []string{"quarry"}
	c.Roles = append(c.Roles, idle)
	c.Groups = append(c.Groups, Group{Region: "north", Role: "idle", Households: 5})
	got, _ := viaRun(t, c)
	var rules []string
	for _, e := range got {
		rules = append(rules, e.Rule+"@"+strings.Join(e.Path, "/"))
	}
	want := []string{"storage_gap@regions/north/commodities/grain", "role_cannot_work@roles/idle"}
	if !slices.Equal(rules, want) {
		t.Errorf("Validate = %v, want %v: sorted by path then rule", rules, want)
	}
}

// TestValidate_ViabilityErrorsAreCut verifies that more than 100 viability
// errors end in one ErrTooManyErrors that counts the rest, as the structural
// errors do.
//
// Requirement: FR-CFG-041
func TestValidate_ViabilityErrorsAreCut(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.Commodities = append(c.Commodities, viaCommodity("rock", 50, nil))
	c.Recipes = append(c.Recipes, viaRecipe("quarry", nil, []Output{viaOut("rock", 1)}, func(r *Recipe) { r.Sited = true }))
	for i := 0; i < 150; i++ {
		id := "idle" + string(rune('a'+i/26)) + string(rune('a'+i%26))
		r := DefaultRole()
		r.ID, r.Recipes = id, []string{"quarry"}
		c.Roles = append(c.Roles, r)
		c.Groups = append(c.Groups, Group{Region: "north", Role: id, Households: 1})
	}
	err := c.Validate()
	joined := err.(interface{ Unwrap() []error }).Unwrap()
	if len(joined) != 101 {
		t.Fatalf("Validate = %d errors, want 100 and one ErrTooManyErrors", len(joined))
	}
	last := joined[100]
	if !errors.Is(last, ErrTooManyErrors) || !strings.Contains(last.Error(), "50 more") {
		t.Errorf("last error = %v, want ErrTooManyErrors counting 50 more", last)
	}
	for _, e := range joined[:100] {
		var ve *ViabilityError
		if !errors.As(e, &ve) || ve.Rule != "role_cannot_work" {
			t.Fatalf("error %v, want a role_cannot_work *ViabilityError", e)
		}
	}
}

// TestValidate_StructuralErrorsStopViability verifies that phase 6 does not run
// on a configuration phases 1 to 5 refused: an unknown region in a yield profile
// is the structural error alone and builds no table.
//
// Requirement: FR-CFG-033
//
// Guard: the tables panic on an unresolved reference, so a viability phase that
// ran after a structural error would crash on the very input Validate must refuse.
func TestValidate_StructuralErrorsStopViability(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.Yields = []YieldProfile{{Region: "nowhere", Recipe: "farm", HasAll: true, All: 10_000}}
	c.Commodities[0].ShelfLife = 1 // viability would complain too
	err := c.Validate()
	if !errors.Is(err, ErrUnknownRegion) || errors.Is(err, ErrNotViable) {
		t.Errorf("Validate = %v, want ErrUnknownRegion and no ErrNotViable", err)
	}
}

// TestViaToy_ListsReverseCanonically verifies that phase 6 canonicalises: the
// toy chain with its entity lists in reverse order validates and warns as the
// sorted one does, where the tables would panic on an unsorted slice.
//
// Requirement: FR-VIA-022
func TestViaToy_ListsReverseCanonically(t *testing.T) {
	t.Parallel()
	c := viaToy()
	c.Commodities = append(c.Commodities, viaCommodity("berry", 50, nil))
	c.Recipes = append(c.Recipes, viaRecipe("pick", nil, []Output{viaOut("berry", 3)}, nil))
	c.Roles[0].Recipes = []string{"pick", "farm"}
	c.World.Upkeep = []Item{{Commodity: "grain", Quantity: 1}, {Commodity: "berry", Quantity: 1}}
	c.Regions[0].ReferenceBasket = 150
	rev := c
	rev.Commodities = slices.Clone(c.Commodities)
	rev.Recipes = slices.Clone(c.Recipes)
	slices.Reverse(rev.Commodities)
	slices.Reverse(rev.Recipes)
	rev.World.Upkeep = slices.Clone(c.World.Upkeep)
	slices.Reverse(rev.World.Upkeep)
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate(sorted) = %v, want nil", err)
	}
	if err := rev.Validate(); err != nil {
		t.Fatalf("Validate(reversed) = %v, want nil", err)
	}
	if !reflect.DeepEqual(c.Warnings(), rev.Warnings()) {
		t.Errorf("Warnings differ between the sorted and the reversed lists:\n%v\n%v", c.Warnings(), rev.Warnings())
	}
}

// TestViaModel_GateIsTheBounds verifies that a model built for Validate gates its
// tableaux at MaxViabilityCells, its pivots at MaxViabilityPivots and the cycle
// enumeration at MaxViabilityCells steps, as spec-config.md 7.6 says.
//
// Requirement: FR-CFG-043, FR-CFG-046
func TestViaModel_GateIsTheBounds(t *testing.T) {
	t.Parallel()
	c := viaToy().canonical()
	m := newViaModel(&c)
	if m.maxCells != MaxViabilityCells || m.maxPivots != MaxViabilityPivots || m.maxSteps != MaxViabilityCells {
		t.Errorf("gate = %d cells, %d pivots, %d steps, want %d, %d, %d", m.maxCells, m.maxPivots, m.maxSteps, MaxViabilityCells, MaxViabilityPivots, MaxViabilityCells)
	}
}
