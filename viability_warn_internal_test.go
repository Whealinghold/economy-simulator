package economysim

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// viaTwoRegions adds a south region with a household of traders (a role with no
// recipe, exempt from role_cannot_work) to the toy, and makes the farm sited with
// a yield in the north only, so south has no way to make grain.
func viaTwoRegions(c *Config) {
	south := DefaultRegion()
	south.ID, south.ReferenceBasket = "south", 100
	c.Regions = append(c.Regions, south)
	trader := DefaultRole()
	trader.ID = "trader"
	c.Roles = append(c.Roles, trader)
	c.Groups = append(c.Groups, Group{Region: "south", Role: "trader", Households: 5})
	c.Recipes[0].Sited = true
	c.Yields = []YieldProfile{{Region: "north", Recipe: "farm", HasAll: true, All: 10_000}}
}

// TestValidate_ViabilityWarnings verifies each viability warning on a
// configuration that draws only it and that never fails Validate (FR-VIA-009,
// FR-CFG-042), and that a configuration just short of the condition does not draw
// it.
//
// Requirement: FR-VIA-009, FR-VIA-008, FR-CFG-042, FR-CFG-044
func TestValidate_ViabilityWarnings(t *testing.T) {
	t.Parallel()
	// The seed cycle of the cycle_cost_gain cases: grow turns 1 seed into 5 grain,
	// thresh 4 grain into 2 seed, and a wild source gives 1 seed. The ratio around
	// the cycle is (1 / 5) * (4 / 2) = 0.4, so the gain CostFloor / 10 000 * 0.4 is
	// 1 at a CostFloor of 25 000.
	seedCycle := func(floor int64) func(c *Config) {
		return func(c *Config) {
			c.World.CostFloor, c.World.CostCeiling = floor, 30_000
			c.Commodities = append(c.Commodities, viaCommodity("seed", 50, nil))
			c.Recipes = []Recipe{
				viaRecipe("gather_seed", nil, []Output{viaOut("seed", 1)}, nil),
				viaRecipe("grow", []Item{{Commodity: "seed", Quantity: 1}}, []Output{viaOut("grain", 5)}, nil),
				viaRecipe("thresh", []Item{{Commodity: "grain", Quantity: 4}}, []Output{viaOut("seed", 2)}, nil),
			}
			c.Roles[0].Recipes = []string{"gather_seed", "grow", "thresh"}
		}
	}
	tests := []struct {
		name   string
		mutate func(c *Config)
		rule   string
		path   []string
		fires  bool
	}{
		{
			// grain is made in the north only; south's households need it and cannot
			// make it.
			name:   "region_not_self_sufficient names the import",
			mutate: viaTwoRegions,
			rule:   "region_not_self_sufficient", path: []string{"regions", "south", "commodities", "grain"}, fires: true,
		},
		{
			name:   "region_not_self_sufficient is not raised for the producing region",
			mutate: viaTwoRegions,
			rule:   "region_not_self_sufficient", path: []string{"regions", "north", "commodities", "grain"}, fires: false,
		},
		{
			// south farms at a yield of 2 000: 4 * 0.2 = 0.8 grain a household-tick
			// against the 1 eaten, so its autarky surplus is 8 000 < 10 000, while the
			// pooled realm (with the north) is fine.
			name: "region_not_self_sufficient for a surplus below 1",
			mutate: func(c *Config) {
				viaTwoRegions(c)
				c.Roles[1].Recipes = []string{"farm"}
				c.Yields = append(c.Yields, YieldProfile{Region: "south", Recipe: "farm", HasAll: true, All: 2_000})
			},
			rule: "region_not_self_sufficient", path: []string{"regions", "south"}, fires: true,
		},
		{
			name: "region_not_self_sufficient is not raised at a surplus of exactly 1",
			mutate: func(c *Config) {
				viaTwoRegions(c)
				c.Roles[1].Recipes = []string{"farm"}
				// 4 * 0.25 = 1 grain a tick for the 1 eaten.
				c.Yields = append(c.Yields, YieldProfile{Region: "south", Recipe: "farm", HasAll: true, All: 2_500})
			},
			rule: "region_not_self_sufficient", path: []string{"regions", "south"}, fires: false,
		},
		{
			// The expected work of 20 ticks is 20 * 10 000 / 365 = 547 basis points
			// of the year; the 2 500 the household must work to subsist is above it.
			name: "subsistence_exceeds_expected_work fires above the expected work",
			mutate: func(c *Config) {
				c.Roles[0].ExpectedWorkTicks, c.Roles[0].MinWorkShare = 20, 0
			},
			rule: "subsistence_exceeds_expected_work", path: []string{"regions", "north"}, fires: true,
		},
		{
			// 250 ticks is 6 849 basis points, above the 2 500 needed.
			name:   "subsistence_exceeds_expected_work is not raised below the expected work",
			mutate: func(c *Config) {},
			rule:   "subsistence_exceeds_expected_work", path: []string{"regions", "north"}, fires: false,
		},
		{
			// Farming in spring only with 8 a tick: the start-capacity row binds
			// (x <= 10 * 92 / 365 = 2.52) so the labour share is absent
			// (FR-VIA-012), and a share cannot exceed the 547 expected.
			name: "subsistence_exceeds_expected_work is not raised when a capacity row is tight",
			mutate: func(c *Config) {
				c.Roles[0].ExpectedWorkTicks, c.Roles[0].MinWorkShare = 20, 0
				c.Recipes[0].Sited = true
				c.Recipes[0].Outputs[0].Quantity = 8
				c.Yields = []YieldProfile{viaSpring("north", "farm")}
			},
			rule: "subsistence_exceeds_expected_work", path: []string{"regions", "north"}, fires: false,
		},
		{
			// 2 grain a tick: the nominal programme works 250 of 365 ticks, 2 * 250 /
			// 365 = 1.37 < the mean target ratio of 15 000 basis points = 1.5.
			name: "target_above_surplus fires below the target",
			mutate: func(c *Config) {
				c.Recipes[0].Outputs[0].Quantity = 2
			},
			rule: "target_above_surplus", path: []string{"regions", "north"}, fires: true,
		},
		{
			// 4 * 250 / 365 = 2.74 > 1.5.
			name:   "target_above_surplus is not raised above the target",
			mutate: func(c *Config) {},
			rule:   "target_above_surplus", path: []string{"regions", "north"}, fires: false,
		},
		{
			// berry lasts 100 ticks, is picked in spring only (the longest run
			// without a pick is 273) and feeds jam, which runs all year.
			name: "storage_gap_input fires for an input of an all-year activity",
			mutate: func(c *Config) {
				c.Commodities = append(c.Commodities, viaCommodity("berry", 50, func(m *Commodity) { m.ShelfLife = 100 }))
				c.Recipes = []Recipe{
					viaRecipe("pick", nil, []Output{viaOut("berry", 8)}, func(r *Recipe) { r.Sited = true }),
					viaRecipe("jam", []Item{{Commodity: "berry", Quantity: 1}}, []Output{viaOut("grain", 4)}, nil),
				}
				c.Roles[0].Recipes = []string{"pick", "jam"}
				c.Yields = []YieldProfile{viaSpring("north", "pick")}
			},
			rule: "storage_gap_input", path: []string{"regions", "north", "commodities", "berry"}, fires: true,
		},
		{
			name: "storage_gap_input is not raised when the shelf life holds the gap",
			mutate: func(c *Config) {
				c.Commodities = append(c.Commodities, viaCommodity("berry", 50, func(m *Commodity) { m.ShelfLife = 273 }))
				c.Recipes = []Recipe{
					viaRecipe("pick", nil, []Output{viaOut("berry", 8)}, func(r *Recipe) { r.Sited = true }),
					viaRecipe("jam", []Item{{Commodity: "berry", Quantity: 1}}, []Output{viaOut("grain", 4)}, nil),
				}
				c.Roles[0].Recipes = []string{"pick", "jam"}
				c.Yields = []YieldProfile{viaSpring("north", "pick")}
			},
			rule: "storage_gap_input", path: []string{"regions", "north", "commodities", "berry"}, fires: false,
		},
		{
			name:   "cycle_cost_gain fires at a gain of exactly 1",
			mutate: seedCycle(25_000),
			rule:   "cycle_cost_gain", path: []string{"commodities", "grain"}, fires: true,
		},
		{
			name:   "cycle_cost_gain is not raised below a gain of 1",
			mutate: seedCycle(24_999),
			rule:   "cycle_cost_gain", path: []string{"commodities", "grain"}, fires: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := viaToy()
			tt.mutate(&c)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate(%s) = %v, want nil: a warning never fails validation", tt.name, err)
			}
			got := c.Warnings()
			if has := viaWarned(got, tt.rule, tt.path...); has != tt.fires {
				t.Errorf("Warnings(%s) has %s at %v = %v, want %v; warnings: %v", tt.name, tt.rule, tt.path, has, tt.fires, got)
			}
		})
	}
}

// TestWarnings_ViabilityNeedsACleanConfiguration verifies that Warnings returns
// the structural warnings and no viability warning, and does not panic, for a
// configuration that does not validate, and the viability warnings too for the
// same configuration once it does.
//
// Requirement: FR-CFG-042
//
// Guard: the viability tables panic on an unresolved reference, and Warnings is
// documented safe on a configuration that does not validate; the gate is what
// keeps both true (D125).
func TestWarnings_ViabilityNeedsACleanConfiguration(t *testing.T) {
	t.Parallel()
	c := viaToy()
	viaTwoRegions(&c)
	c.Calendar.TicksPerDay = 5 // inexact_hour, a structural warning
	for i := range c.Roles {
		c.Roles[i].ExpectedWorkTicks, c.Roles[i].MinWorkShare = 50, 0 // inside the longer year
	}
	if got := c.Warnings(); !viaWarned(got, "region_not_self_sufficient", "regions", "south", "commodities", "grain") || !viaWarned(got, "inexact_hour", "calendar", "ticks_per_day") {
		t.Fatalf("Warnings(valid) = %v, want inexact_hour and region_not_self_sufficient", got)
	}
	c.Yields = append(c.Yields, YieldProfile{Region: "nowhere", Recipe: "farm", HasAll: true, All: 1})
	got := c.Warnings() // must not panic
	if viaWarned(got, "region_not_self_sufficient", "regions", "south", "commodities", "grain") {
		t.Errorf("Warnings(unknown region in a yield) = %v, want no viability warning", got)
	}
	if !viaWarned(got, "inexact_hour", "calendar", "ticks_per_day") {
		t.Errorf("Warnings(unknown region in a yield) = %v, want the structural warnings", got)
	}
}

// TestCheckViability_SizeGate verifies the gate (D126, FR-VIA-010, FR-CFG-043):
// with the cell limit lowered, the programmes of V2 and V3 are skipped and each
// skip is the warning viability_check_skipped naming its part, V1 and V4 still
// run, and with the limit as it is the same configuration runs the programmes.
//
// Requirement: FR-VIA-010, FR-CFG-043
func TestCheckViability_SizeGate(t *testing.T) {
	t.Parallel()
	// The seed cycle with a wild source (productive) and a grain that spoils
	// between spring pickings, so V4 has something to find with the programmes off.
	build := func() Config {
		c := viaToy()
		c.Commodities[0].ShelfLife = 100
		c.Commodities = append(c.Commodities, viaCommodity("seed", 50, nil))
		c.Recipes = []Recipe{
			viaRecipe("gather_seed", nil, []Output{viaOut("seed", 1)}, nil),
			viaRecipe("grow", []Item{{Commodity: "seed", Quantity: 1}}, []Output{viaOut("grain", 5)}, func(r *Recipe) { r.Sited = true }),
			viaRecipe("thresh", []Item{{Commodity: "grain", Quantity: 4}}, []Output{viaOut("seed", 2)}, nil),
		}
		c.Roles[0].Recipes = []string{"gather_seed", "grow", "thresh"}
		c.Yields = []YieldProfile{viaSpring("north", "grow")}
		return c.canonical()
	}
	skips := func(o viabilityOutcome) []string {
		var parts []string
		for _, w := range o.warnings {
			if w.Rule == "viability_check_skipped" {
				parts = append(parts, w.Path[1])
			}
		}
		slices.Sort(parts)
		return parts
	}
	tests := []struct {
		name      string
		limit     func(m *viaModel)
		wantSkips []string
	}{
		{"limits as they are", func(m *viaModel) {}, nil},
		// One cell: no tableau fits, so V2 of the component, V3 pooled, V3 of the
		// region and its nominal programme are all skipped.
		{"cells", func(m *viaModel) { m.maxCells = 1 }, []string{"nominal", "v2", "v3_pooled", "v3_region"}},
		// A cap of 0 pivots stops every solve that needs one at its first pivot.
		{"pivots", func(m *viaModel) { m.maxPivots = 0 }, []string{"nominal", "v2", "v3_pooled", "v3_region"}},
		// The enumeration of the cycle gain alone.
		{"steps", func(m *viaModel) { m.maxSteps = 1 }, []string{"cycle_gain"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := build()
			m := newViaModel(&c)
			tt.limit(m)
			o := m.check(newLookup(&c))
			if got := skips(o); !slices.Equal(got, tt.wantSkips) {
				t.Errorf("skipped parts (%s) = %v, want %v", tt.name, got, tt.wantSkips)
			}
			// V4 is never skipped: grain lasts 100 ticks and only the spring-only grow
			// makes it, so its gap is 273 whatever the gate.
			if len(o.errs) != 1 || o.errs[0].Rule != "storage_gap" {
				t.Errorf("errors (%s) = %v, want the storage_gap of V4 alone, whatever the gate", tt.name, o.errs)
			}
		})
	}
}

// TestCheckViability_AFaultIsReported verifies that a defect of the solver (a
// failed certificate, which no input can cause) is kept as the first fault, in
// the form ErrSolveInconsistent, and that Validate reports it as an error and not
// as a panic.
//
// Requirement: FR-VIA-014
func TestCheckViability_AFaultIsReported(t *testing.T) {
	t.Parallel()
	var o viabilityOutcome
	o.fail(steady.ErrInconsistent)
	o.fail(errors.New("second"))
	if !errors.Is(o.fault, ErrSolveInconsistent) {
		t.Fatalf("fault = %v, want ErrSolveInconsistent", o.fault)
	}
	var other viabilityOutcome
	other.fail(errors.New("tableau"))
	if !errors.Is(other.fault, ErrSolveInconsistent) {
		t.Errorf("fault of another error = %v, want it to wrap ErrSolveInconsistent", other.fault)
	}
	v := &validator{}
	v.addOutcome(o)
	err := v.rep.err()
	if !errors.Is(err, ErrSolveInconsistent) {
		t.Errorf("report = %v, want ErrSolveInconsistent", err)
	}
}

// TestValidate_EmptyBasketAndRawBasketAreNotErrors verifies D130 and JC5 for
// Validate: a configuration with no upkeep at all, and one whose only upkeep is a
// Raw marker nothing makes, validate (the programmes are skipped silently), and
// V1 still runs on the second.
//
// Requirement: FR-VIA-007, FR-CFG-029
func TestValidate_EmptyBasketAndRawBasketAreNotErrors(t *testing.T) {
	t.Parallel()
	t.Run("no upkeep", func(t *testing.T) {
		t.Parallel()
		c := viaToy()
		c.World.Upkeep = nil
		c.Regions[0].ReferenceBasket = 0
		if err := c.Validate(); err != nil {
			t.Errorf("Validate(no upkeep) = %v, want nil: no_upkeep is a warning", err)
		}
	})
	t.Run("only an unsourced Raw good", func(t *testing.T) {
		t.Parallel()
		c := viaToy()
		c.Commodities[0].Raw = true
		c.Recipes, c.Roles[0].Recipes = nil, nil
		if err := c.Validate(); err != nil {
			t.Errorf("Validate(raw basket) = %v, want nil: the programme is skipped", err)
		}
	})
	t.Run("a raw good with a source is a programme", func(t *testing.T) {
		t.Parallel()
		c := viaToy()
		c.Commodities[0].Raw = true // the farm is a real source, so the programme runs
		c.Recipes[0].Outputs[0].Quantity = 1
		c.Recipes[0].Duration = 2
		if err := c.Validate(); !errors.Is(err, ErrNotViable) {
			t.Errorf("Validate(sourced raw, labour short) = %v, want labour_short", err)
		}
	})
}

// TestValidate_ViabilityDoesNotChangeTheReceiver verifies that Validate and
// Warnings leave a configuration with a warm start and unsorted lists as they
// found it.
//
// Requirement: NFR-CFG-002
func TestValidate_ViabilityDoesNotChangeTheReceiver(t *testing.T) {
	t.Parallel()
	c := viaToy()
	viaTwoRegions(&c)
	slices.Reverse(c.Regions)
	slices.Reverse(c.Roles)
	before := reflect.ValueOf(c).Interface()
	copyOf := Config(c)
	copyOf.Regions = slices.Clone(c.Regions)
	copyOf.Roles = slices.Clone(c.Roles)
	_ = c.Validate()
	_ = c.Warnings()
	if !reflect.DeepEqual(before, any(c)) || !reflect.DeepEqual(copyOf.Regions, c.Regions) || !reflect.DeepEqual(copyOf.Roles, c.Roles) {
		t.Errorf("Validate/Warnings changed the receiver")
	}
}
