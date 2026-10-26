package economysim_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the demand module's configuration: its validation, its warnings,
// the tradable set and the ladders it derives, and the packs that switch it on.
// The module has no world effect yet (ADR 0010 hooks are not built), so nothing
// here steps a world beyond construction and a few inert ticks.

// withDemand switches the demand module on in c, a copy of referenceConfig, with
// two wants. The numbers are chosen so that no warning of the module is raised:
// comfort (refined, 3 190) at 1 unit in 4 ticks and treat (prepared 628 and
// staple 96) at 1 unit in 5 ticks. The cook's want spending at the reference is
// 3 190 * 15 000 / 10 000 / 4 + 362 * 5 000 / 10 000 / 5 = 1 196 + 36 = 1 232 a
// tick against an upkeep of 2 * 96 + 628 + 179 = 999, so the ceiling of the
// closed economy is about 2.2 against the target of 1.5.
func withDemand(c *economysim.Config) {
	comfort, treat := economysim.DefaultWant(), economysim.DefaultWant()
	comfort.ID, comfort.Quantity, comfort.Period, comfort.Rank = "comfort", 1, 4, 1
	comfort.Items = []economysim.WantItem{{Commodity: "refined", Weight: 10_000}}
	treat.ID, treat.Quantity, treat.Period, treat.Rank = "treat", 1, 5, 2
	treat.Items = []economysim.WantItem{{Commodity: "prepared", Weight: 10_000}, {Commodity: "staple", Weight: 10_000}}
	taste := func(role, want string, scale int64) economysim.Taste {
		return economysim.Taste{Role: role, Want: want, Scale: scale}
	}
	c.Modules.Demand = &economysim.Demand{
		Wants:        []economysim.Want{comfort, treat},
		Tastes:       []economysim.Taste{taste("cook", "comfort", 10_000), taste("cook", "treat", 5_000), taste("grower", "comfort", 10_000)},
		RegionTastes: []economysim.RegionTaste{{Region: "north", Want: "comfort", Scale: 15_000}},
	}
}

// demandConfig returns referenceConfig with the demand module on.
func demandConfig() economysim.Config {
	c := referenceConfig()
	withDemand(&c)
	return c
}

// wantOf returns the want with the given ID of c, or nil.
func wantOf(c *economysim.Config, id string) *economysim.Want {
	for i := range c.Modules.Demand.Wants {
		if c.Modules.Demand.Wants[i].ID == id {
			return &c.Modules.Demand.Wants[i]
		}
	}
	return nil
}

// manyWants returns n wants with distinct IDs, each valid on its own.
func manyWants(n int) []economysim.Want {
	out := make([]economysim.Want, n)
	for i := range out {
		w := economysim.DefaultWant()
		w.ID, w.Quantity = fmt.Sprintf("w%04d", i), 1
		w.Items = []economysim.WantItem{{Commodity: "refined", Weight: 10_000}}
		out[i] = w
	}
	return out
}

// demandFieldCases are the failing cases of the rows of the demand module in
// the parameter-owner table (spec-demand.md 9, FR-CFG-031): each breaks one
// field of a valid demand configuration. buildFieldCases appends them to the
// table that the registry test reads.
func demandFieldCases() []fieldCase {
	bounds := []error{economysim.ErrOutOfBounds}
	dem := func(field, name string, rule error, mutate func(c *economysim.Config), path ...string) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{path},
			mutate: func(c *economysim.Config) { withDemand(c); mutate(c) }}
	}
	wantBound := func(field string, bad int64, set func(w *economysim.Want, v int64)) fieldCase {
		name := strings.ToLower(field[strings.IndexByte(field, '.')+1:])
		return fieldCase{field: field, name: "outside its range", rules: bounds,
			paths:  [][]string{{"modules", "demand", "wants", "comfort", name}},
			mutate: func(c *economysim.Config) { withDemand(c); set(wantOf(c, "comfort"), bad) }}
	}
	itemAt := func(c *economysim.Config, want string, i int) *economysim.WantItem {
		return &wantOf(c, want).Items[i]
	}
	cs := []fieldCase{
		dem("Demand.Wants", "more than MaxWants", economysim.ErrTooMany,
			func(c *economysim.Config) { c.Modules.Demand.Wants = manyWants(economysim.MaxWants + 1) },
			"modules", "demand", "wants"),
		dem("Want.ID", "upper case letter", economysim.ErrInvalidID, func(c *economysim.Config) {
			w := economysim.DefaultWant()
			w.ID, w.Quantity = "Bad", 1
			w.Items = []economysim.WantItem{{Commodity: "refined", Weight: 10_000}}
			c.Modules.Demand.Wants = append(c.Modules.Demand.Wants, w)
		}, "modules", "demand", "wants", "Bad", "id"),
		dem("Want.ID", "a want twice", economysim.ErrDuplicateID, func(c *economysim.Config) {
			c.Modules.Demand.Wants = append(c.Modules.Demand.Wants, *wantOf(c, "comfort"))
		}, "modules", "demand", "wants", "comfort", "id"),
		dem("Want.Items", "none", economysim.ErrWantItems,
			func(c *economysim.Config) { wantOf(c, "comfort").Items = nil },
			"modules", "demand", "wants", "comfort", "items"),
		dem("Want.Items", "more than MaxWantItems", economysim.ErrTooMany, func(c *economysim.Config) {
			w := wantOf(c, "comfort")
			w.Items = nil
			for _, id := range []string{"fuel", "mineral", "prepared", "refined", "seed"} {
				w.Items = append(w.Items, economysim.WantItem{Commodity: id, Weight: 1})
			}
		}, "modules", "demand", "wants", "comfort", "items"),
		wantBound("Want.Quantity", 0, func(w *economysim.Want, v int64) { w.Quantity = v }),
		wantBound("Want.Period", 0, func(w *economysim.Want, v int64) { w.Period = v }),
		wantBound("Want.Rank", -1, func(w *economysim.Want, v int64) { w.Rank = v }),
		wantBound("Want.Elasticity", economysim.MaxElasticityBP+1, func(w *economysim.Want, v int64) { w.Elasticity = v }),
		wantBound("Want.Ceiling", 9_999, func(w *economysim.Want, v int64) { w.Ceiling = v }),
		wantBound("Want.Substitution", economysim.MaxSubstitution+1, func(w *economysim.Want, v int64) { w.Substitution = v }),
		dem("Want.Label", "label over 256 bytes", economysim.ErrOutOfBounds,
			func(c *economysim.Config) { wantOf(c, "comfort").Label = strings.Repeat("x", 257) },
			"modules", "demand", "wants", "comfort", "label"),
		dem("WantItem.Commodity", "unknown commodity", economysim.ErrUnknownCommodity,
			func(c *economysim.Config) { itemAt(c, "comfort", 0).Commodity = "nothing" },
			"modules", "demand", "wants", "comfort", "items[0]", "commodity"),
		dem("WantItem.Commodity", "durable commodity", economysim.ErrWantDurable,
			func(c *economysim.Config) { itemAt(c, "comfort", 0).Commodity = "tool_a" },
			"modules", "demand", "wants", "comfort", "items[0]", "commodity"),
		dem("WantItem.Commodity", "a commodity twice in one want", economysim.ErrDuplicateID,
			func(c *economysim.Config) { itemAt(c, "treat", 1).Commodity = "prepared" },
			"modules", "demand", "wants", "treat", "items[1]", "commodity"),
		dem("WantItem.Weight", "below 1 with another item usable", economysim.ErrOutOfBounds,
			func(c *economysim.Config) { itemAt(c, "treat", 1).Weight = 0 },
			"modules", "demand", "wants", "treat", "items[1]", "weight"),
		dem("WantItem.Weight", "above MaxBasisPoints", economysim.ErrOutOfBounds,
			func(c *economysim.Config) { itemAt(c, "treat", 1).Weight = economysim.MaxBasisPoints + 1 },
			"modules", "demand", "wants", "treat", "items[1]", "weight"),
		dem("Taste.Role", "unknown role", economysim.ErrUnknownRole, func(c *economysim.Config) {
			c.Modules.Demand.Tastes = append(c.Modules.Demand.Tastes, economysim.Taste{Role: "nobody", Want: "comfort", Scale: 10_000})
		}, "modules", "demand", "tastes", "nobody", "comfort", "role"),
		dem("Taste.Role", "a (role, want) pair twice", economysim.ErrDuplicateID, func(c *economysim.Config) {
			c.Modules.Demand.Tastes = append(c.Modules.Demand.Tastes, c.Modules.Demand.Tastes[0])
		}, "modules", "demand", "tastes", "cook", "comfort"),
		dem("Taste.Want", "unknown want", economysim.ErrUnknownWant, func(c *economysim.Config) {
			c.Modules.Demand.Tastes = append(c.Modules.Demand.Tastes, economysim.Taste{Role: "grower", Want: "nothing", Scale: 10_000})
		}, "modules", "demand", "tastes", "grower", "nothing", "want"),
		dem("Taste.Scale", "above MaxTasteScale", economysim.ErrOutOfBounds,
			func(c *economysim.Config) { c.Modules.Demand.Tastes[0].Scale = economysim.MaxTasteScale + 1 },
			"modules", "demand", "tastes", "cook", "comfort", "scale"),
		dem("Taste.Scale", "below 0", economysim.ErrOutOfBounds,
			func(c *economysim.Config) { c.Modules.Demand.Tastes[0].Scale = -1 },
			"modules", "demand", "tastes", "cook", "comfort", "scale"),
		dem("RegionTaste.Region", "unknown region", economysim.ErrUnknownRegion, func(c *economysim.Config) {
			c.Modules.Demand.RegionTastes = append(c.Modules.Demand.RegionTastes, economysim.RegionTaste{Region: "nowhere", Want: "comfort", Scale: 10_000})
		}, "modules", "demand", "region_tastes", "nowhere", "comfort", "region"),
		dem("RegionTaste.Region", "a (region, want) pair twice", economysim.ErrDuplicateID, func(c *economysim.Config) {
			c.Modules.Demand.RegionTastes = append(c.Modules.Demand.RegionTastes, c.Modules.Demand.RegionTastes[0])
		}, "modules", "demand", "region_tastes", "north", "comfort"),
		dem("RegionTaste.Want", "unknown want", economysim.ErrUnknownWant, func(c *economysim.Config) {
			c.Modules.Demand.RegionTastes = append(c.Modules.Demand.RegionTastes, economysim.RegionTaste{Region: "north", Want: "nothing", Scale: 10_000})
		}, "modules", "demand", "region_tastes", "north", "nothing", "want"),
		dem("RegionTaste.Scale", "above MaxTasteScale", economysim.ErrOutOfBounds,
			func(c *economysim.Config) { c.Modules.Demand.RegionTastes[0].Scale = economysim.MaxTasteScale + 1 },
			"modules", "demand", "region_tastes", "north", "comfort", "scale"),
	}
	return cs
}

// TestValidate_Demand verifies, in a table of its own and not only through the
// registry's field cases, every rule of spec-demand.md 10.1 that needs more than
// one error or a large input: the counts (ErrTooMany stops the other rules), the
// weights that are all 0, a want that is not usable and the two overflow
// products. The reference configuration with the module on validates.
//
// Requirement: FR-DEM-003, FR-DEM-032, FR-DEM-041, FR-CFG-061, FR-CFG-062
func TestValidate_Demand(t *testing.T) {
	t.Parallel()
	onDemand := func(mutate func(d *economysim.Demand)) func(c *economysim.Config) {
		return func(c *economysim.Config) { mutate(c.Modules.Demand) }
	}
	many := func(role string, n int) []economysim.Taste {
		out := make([]economysim.Taste, n)
		for i := range out {
			out[i] = economysim.Taste{Role: role, Want: fmt.Sprintf("w%04d", i), Scale: 10_000}
		}
		return out
	}
	heavy := func(c *economysim.Config) {
		c.World.MinAgents = 1
		for i := range c.Groups {
			if c.Groups[i].Role == "cook" {
				c.Groups[i].Households, c.Groups[i].Scale = 1_000_000, 1_000_000
			}
		}
	}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		want   error
		path   []string
		value  string
	}{
		{"valid", func(*economysim.Config) {}, nil, nil, ""},
		{"more wants than MaxWants", onDemand(func(d *economysim.Demand) { d.Wants = manyWants(economysim.MaxWants + 1) }),
			economysim.ErrTooMany, []string{"modules", "demand", "wants"}, "1001"},
		{"more tastes than MaxTastes", onDemand(func(d *economysim.Demand) {
			d.Tastes = make([]economysim.Taste, economysim.MaxTastes+1)
		}), economysim.ErrTooMany, []string{"modules", "demand", "tastes"}, "32001"},
		{"more region tastes than MaxRegionTastes", onDemand(func(d *economysim.Demand) {
			d.RegionTastes = make([]economysim.RegionTaste, economysim.MaxRegionTastes+1)
		}), economysim.ErrTooMany, []string{"modules", "demand", "region_tastes"}, "1000001"},
		{"more tastes of one role than MaxWantsPerRole", func(c *economysim.Config) {
			c.Modules.Demand.Wants, c.Modules.Demand.RegionTastes = manyWants(economysim.MaxWantsPerRole+1), nil
			c.Modules.Demand.Tastes = many("cook", economysim.MaxWantsPerRole+1)
		}, economysim.ErrTooMany, []string{"modules", "demand", "tastes", "cook"}, "33"},
		{"exactly MaxWantsPerRole tastes of one role", func(c *economysim.Config) {
			c.Modules.Demand.Wants, c.Modules.Demand.RegionTastes = manyWants(economysim.MaxWantsPerRole), nil
			c.Modules.Demand.Tastes = many("cook", economysim.MaxWantsPerRole)
		}, nil, nil, ""},
		{"every weight 0", func(c *economysim.Config) {
			c.Modules.Demand.Wants[0].Items = []economysim.WantItem{{Commodity: "refined"}}
		}, economysim.ErrWantItems, []string{"modules", "demand", "wants", "comfort", "items"}, "all weights 0"},
		{"units over MaxQuantity", func(c *economysim.Config) {
			heavy(c)
			c.Modules.Demand.Wants[0].Quantity, c.Modules.Demand.Wants[0].Period = 1_000, 1
		}, economysim.ErrOverflow, []string{"modules", "demand", "tastes", "cook", "comfort"}, ""},
		{"spending over MaxMoney", func(c *economysim.Config) {
			heavy(c)
			c.RegionPrices = append(c.RegionPrices, economysim.RegionPrice{Region: "north", Commodity: "refined", Price: economysim.MaxPrice})
			// Two wants of refined at 400 units a tick for a weight of 10^6 and a ceiling of 1.5:
			// 6 * 10^8 units each, 1.2 * 10^18 at 10^9 apiece, above 10^18.
			for i := range c.Modules.Demand.Wants {
				w := &c.Modules.Demand.Wants[i]
				w.Quantity, w.Period = 400, 1
				w.Items = []economysim.WantItem{{Commodity: "refined", Weight: 10_000}}
			}
			c.Modules.Demand.Tastes[1].Scale = 10_000
		}, economysim.ErrOverflow, []string{"modules", "demand", "tastes", "cook"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := demandConfig()
			tt.mutate(&c)
			err := c.Validate()
			if tt.want == nil {
				if err != nil {
					t.Fatalf("Validate(%s) = %v, want nil", tt.name, err)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("errors.Is(Validate(%s), %v) = false, want true; error: %v", tt.name, tt.want, err)
			}
			vs, _ := violations(err)
			found := false
			for _, v := range vs {
				if errors.Is(v, tt.want) && slices.Equal(v.Path, tt.path) && (tt.value == "" || v.Value == tt.value) {
					found = true
				}
			}
			if !found {
				t.Errorf("Validate(%s) paths = %q, want %v at %q (value %q)", tt.name, pathsOf(vs), tt.want, tt.path, tt.value)
			}
			if errors.Is(tt.want, economysim.ErrTooMany) && len(vs) != 1 {
				t.Errorf("Validate(%s) returned %d errors %q, want only the count: ErrTooMany stops the other rules", tt.name, len(vs), pathsOf(vs))
			}
		})
	}
}

// TestValidate_DemandOffCostsNothing verifies that the rules of the module read
// nothing when it is off: a configuration whose demand module is nil validates
// exactly as before, and a bad row is reported only when the module is on.
//
// Requirement: FR-DEM-001, FR-CFG-002
func TestValidate_DemandOffCostsNothing(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate(referenceConfig) = %v, want nil", err)
	}
	c.Modules.Demand = &economysim.Demand{Wants: []economysim.Want{{ID: "Bad"}}}
	if err := c.Validate(); !errors.Is(err, economysim.ErrInvalidID) {
		t.Errorf("Validate(module on, bad want) = %v, want ErrInvalidID", err)
	}
	c.Modules.Demand = nil
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(module off again) = %v, want nil", err)
	}
}

// TestWarnings_Demand verifies the seven warnings of spec-demand.md 10.2: each
// is raised at its path by a configuration that is valid, none is raised by the
// base configuration, and the variants that spec-demand.md 10.2 exempts (a role
// with a recipe, with Trades, or with money to spare) raise nothing.
//
// Requirement: FR-DEM-033, FR-CFG-065
func TestWarnings_Demand(t *testing.T) {
	t.Parallel()
	rules := []string{"want_unused", "want_unsourced", "want_inelastic", "wants_exceed_spend_cap", "target_above_demand",
		"class_without_income", "region_taste_without_taste"}
	spare := func(c *economysim.Config, item string) {
		w := economysim.DefaultWant()
		w.ID, w.Quantity = "spare", 1
		w.Items = []economysim.WantItem{{Commodity: item, Weight: 10_000}}
		c.Modules.Demand.Wants = append(c.Modules.Demand.Wants, w)
	}
	pensioner := func(c *economysim.Config) {
		r := economysim.DefaultRole()
		r.ID = "pensioner"
		c.Roles = append(c.Roles, r)
		c.Groups = append(c.Groups, economysim.Group{Region: "north", Role: "pensioner", Households: 5})
		c.Modules.Demand.Tastes = append(c.Modules.Demand.Tastes, economysim.Taste{Role: "pensioner", Want: "comfort", Scale: 10_000})
	}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   string // "" for none of the seven
		path   []string
	}{
		{"base raises none", func(*economysim.Config) {}, "", nil},
		{"want in no taste row", func(c *economysim.Config) { spare(c, "refined") }, "want_unused",
			[]string{"modules", "demand", "wants", "spare"}},
		{"item nobody makes", func(c *economysim.Config) { spare(c, "waste") }, "want_unsourced",
			[]string{"modules", "demand", "wants", "spare", "items[0]", "commodity"}},
		{"raw item is sourced", func(c *economysim.Config) {
			spare(c, "fuel")
			c.Modules.Demand.Tastes = append(c.Modules.Demand.Tastes, economysim.Taste{Role: "grower", Want: "spare", Scale: 10_000})
		}, "", nil},
		{"elasticity 10 000 on a good a role makes", func(c *economysim.Config) { wantOf(c, "comfort").Elasticity = 10_000 }, "want_inelastic",
			[]string{"modules", "demand", "wants", "comfort", "items[0]", "commodity"}},
		{"elasticity above 10 000", func(c *economysim.Config) { wantOf(c, "comfort").Elasticity = 10_001 }, "", nil},
		{"wants can never be fully funded", func(c *economysim.Config) { wantOf(c, "comfort").Quantity = 10 }, "wants_exceed_spend_cap",
			[]string{"modules", "demand", "tastes", "cook"}},
		{"target above the closed economy ceiling", func(c *economysim.Config) { roleOf(c, "cook").TargetRatio = 40_000 }, "target_above_demand",
			[]string{"roles", "cook", "target_ratio"}},
		{"class with a ladder and no income", pensioner, "class_without_income", []string{"roles", "pensioner", "start_money"}},
		{"class whose only taste has scale 0 has no ladder", func(c *economysim.Config) {
			pensioner(c)
			c.Modules.Demand.Tastes[len(c.Modules.Demand.Tastes)-1].Scale = 0
		}, "", nil},
		{"class with Trades", func(c *economysim.Config) { pensioner(c); roleOf(c, "pensioner").Trades = []string{"fuel"} }, "", nil},
		{"class with a recipe", func(c *economysim.Config) { pensioner(c); roleOf(c, "pensioner").Recipes = []string{"cut_fuel"} }, "", nil},
		{"class with money for a year of wants", func(c *economysim.Config) {
			pensioner(c)
			roleOf(c, "pensioner").StartMoney = 1_000_000_000
		}, "", nil},
		{"region taste for a want no role likes", func(c *economysim.Config) {
			spare(c, "refined")
			c.Modules.Demand.RegionTastes = append(c.Modules.Demand.RegionTastes, economysim.RegionTaste{Region: "north", Want: "spare", Scale: 10_000})
		}, "region_taste_without_taste", []string{"modules", "demand", "region_tastes", "north", "spare"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := demandConfig()
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
				t.Errorf("Warnings(%s) raised %q, want none of the demand warnings", tt.name, raised)
			case want != "" && !slices.Contains(raised, want):
				t.Errorf("Warnings(%s) raised %q, want %q among them", tt.name, raised, want)
			}
		})
	}
}

// TestWarnings_DemandOffRaisesNone verifies that a configuration without the
// module raises none of the seven warnings, however the other fields look.
//
// Requirement: FR-DEM-001, FR-DEM-033
func TestWarnings_DemandOffRaisesNone(t *testing.T) {
	t.Parallel()
	for _, w := range referenceConfig().Warnings() {
		switch w.Rule {
		case "want_unused", "want_unsourced", "want_inelastic", "wants_exceed_spend_cap", "target_above_demand",
			"class_without_income", "region_taste_without_taste":
			t.Errorf("Warnings(module off) raised %s at %q, want none of the demand warnings", w.Rule, w.Path)
		}
	}
}

// TestTradableItems_WantItemsJoinTheSet verifies that the items of the wants in
// a role's ladder join its tradable set, in the validator's sets and in the role
// table alike, that a role with no taste row and a taste of scale 0 get nothing,
// that the set is the same with the module off, and that an agent of the role has
// a belief for the item from creation.
//
// Requirement: FR-DEM-006, FR-AGT-003, FR-AGT-004
func TestTradableItems_WantItemsJoinTheSet(t *testing.T) {
	t.Parallel()
	refined := func(c *economysim.Config) int {
		for i, cm := range c.Commodities {
			if cm.ID == "refined" {
				return i
			}
		}
		t.Fatal("no commodity refined")
		return -1
	}
	roleIndex := func(c *economysim.Config, id string) economysim.RoleID {
		for i, r := range c.Roles {
			if r.ID == id {
				return economysim.RoleID(i)
			}
		}
		t.Fatalf("no role %q", id)
		return -1
	}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		role   string
		has    bool
	}{
		{"off", func(c *economysim.Config) { c.Modules.Demand = nil }, "grower", false},
		{"role with a taste", func(*economysim.Config) {}, "grower", true},
		{"role with no taste row", func(*economysim.Config) {}, "digger", false},
		{"taste of scale 0 removes the want", func(c *economysim.Config) { c.Modules.Demand.Tastes[2].Scale = 0 }, "grower", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := demandConfig()
			tt.mutate(&cfg)
			c := economysim.Canonical(cfg)
			id, k := roleIndex(&c, tt.role), refined(&c)
			if got := economysim.TradableSets(&c)[id]["refined"]; got != tt.has {
				t.Errorf("tradable set of %s has refined = %v, want %v", tt.role, got, tt.has)
			}
			table := economysim.NewRoleTable(&c)
			row := economysim.RoleTableRow(&table, id)
			if got := slices.Contains(row.Tradable, k); got != tt.has || row.Member[k] != tt.has {
				t.Errorf("role table of %s tradable has refined = %v, member = %v, want %v", tt.role, got, row.Member[k], tt.has)
			}
			if tt.has && !economysim.RoleTradableSomewhere(&table, economysim.CommodityID(k)) {
				t.Errorf("tradableSomewhere(refined) = false, want true")
			}
			w, err := economysim.NewWorld(cfg)
			if err != nil {
				t.Fatalf("NewWorld(%s) error = %v, want nil", tt.name, err)
			}
			for _, a := range economysim.ConstructionRows(w) {
				if a.Role != id {
					continue
				}
				if len(a.Beliefs) != len(row.Tradable) {
					t.Fatalf("agent %d of %s has %d beliefs, want %d: one for each commodity of the set", a.ID, tt.role, len(a.Beliefs), len(row.Tradable))
				}
				if tt.has && a.Beliefs[slices.Index(row.Tradable, k)].Width <= 0 {
					t.Errorf("agent %d of %s has no belief for refined at creation", a.ID, tt.role)
				}
				break
			}
		})
	}
}

// TestRole_NoTasteNoWantState verifies that the ladder of a role is its wants
// with a taste row of nonzero scale, by rank and then by ID, that a role with no
// such row has an empty ladder, that a class is a role (a role with no recipe and
// a ladder is valid), and that a world holds the module's state if and only if
// the module is on.
//
// Requirement: FR-DEM-004, FR-DEM-005
func TestRole_NoTasteNoWantState(t *testing.T) {
	t.Parallel()
	cfg := demandConfig()
	// Comfort has rank 1 and treat rank 2, so the cook's ladder is comfort, treat. A third
	// want, "aaa", of rank 1, ties with comfort and sorts before it by ID. The maker's taste
	// for treat has scale 0, which removes the want.
	aaa := economysim.DefaultWant()
	aaa.ID, aaa.Quantity, aaa.Rank = "aaa", 1, 1
	aaa.Items = []economysim.WantItem{{Commodity: "fuel", Weight: 10_000}}
	cfg.Modules.Demand.Wants = append(cfg.Modules.Demand.Wants, aaa)
	cfg.Modules.Demand.Tastes = append(cfg.Modules.Demand.Tastes,
		economysim.Taste{Role: "cook", Want: "aaa", Scale: 10_000}, economysim.Taste{Role: "maker", Want: "treat", Scale: 0})
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate = %v, want nil", err)
	}
	c := economysim.Canonical(cfg)
	ladders, on := economysim.DemandLadders(&c)
	if !on {
		t.Fatal("DemandLadders(module on) on = false, want true")
	}
	idOf := func(want string) economysim.WantID {
		for i, w := range c.Modules.Demand.Wants {
			if w.ID == want {
				return economysim.WantID(i)
			}
		}
		t.Fatalf("no want %q", want)
		return -1
	}
	role := func(id string) economysim.RoleID {
		for i, r := range c.Roles {
			if r.ID == id {
				return economysim.RoleID(i)
			}
		}
		t.Fatalf("no role %q", id)
		return -1
	}
	tests := []struct {
		role string
		want []economysim.WantID
	}{
		{"cook", []economysim.WantID{idOf("aaa"), idOf("comfort"), idOf("treat")}},
		{"grower", []economysim.WantID{idOf("comfort")}},
		{"digger", nil},
		{"maker", nil},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			t.Parallel()
			if got := ladders[role(tt.role)]; !slices.Equal(got, tt.want) {
				t.Errorf("ladder of %s = %v, want %v (rank, then ID; scale 0 removes)", tt.role, got, tt.want)
			}
		})
	}
	// A class is a role: a role with no recipe and a ladder validates.
	r := economysim.DefaultRole()
	r.ID = "pensioner"
	cfg.Roles = append(cfg.Roles, r)
	cfg.Modules.Demand.Tastes = append(cfg.Modules.Demand.Tastes, economysim.Taste{Role: "pensioner", Want: "comfort", Scale: 10_000})
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate(a role with no recipe and a ladder) = %v, want nil", err)
	}
	for _, off := range []bool{false, true} {
		cfg := demandConfig()
		if off {
			cfg.Modules.Demand = nil
		}
		w, err := economysim.NewWorld(cfg)
		if err != nil {
			t.Fatalf("NewWorld(off=%v) error = %v, want nil", off, err)
		}
		if got := economysim.WorldHasDemandState(w); got == off {
			t.Errorf("world has demand state = %v with the module off = %v, want it only when on", got, off)
		}
	}
}

// TestDemand_ConfigIsCopiedOnConstruction verifies that the world owns a deep
// copy of the module's tables: changing the configuration the caller passed, or
// the copy Config returns, changes nothing in the world, and the copy is in
// canonical order.
//
// Requirement: FR-CFG-003, FR-CFG-005
//
// Guard: canonical() shares a pointer by default; a module's table that was not
// copied would let a caller change a running world through the Config it kept.
func TestDemand_ConfigIsCopiedOnConstruction(t *testing.T) {
	t.Parallel()
	cfg := demandConfig()
	slices.Reverse(cfg.Modules.Demand.Wants)
	w, err := economysim.NewWorld(cfg)
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	cfg.Modules.Demand.Wants[0].Quantity = 77
	got := w.Config()
	if got.Modules.Demand == nil || got.Modules.Demand.Wants[0].ID != "comfort" || got.Modules.Demand.Wants[0].Quantity != 1 {
		t.Fatalf("World.Config().Modules.Demand.Wants[0] = %+v, want comfort with quantity 1 in canonical order", got.Modules.Demand)
	}
	got.Modules.Demand.Wants[0].Quantity = 99
	if again := w.Config(); again.Modules.Demand.Wants[0].Quantity != 1 {
		t.Errorf("World.Config() after changing a copy: quantity = %d, want 1", again.Modules.Demand.Wants[0].Quantity)
	}
	if got.Modules.Demand.Wants[1].Items[0].Commodity > got.Modules.Demand.Wants[1].Items[1].Commodity {
		t.Errorf("items of %s = %v, want canonical order by commodity", got.Modules.Demand.Wants[1].ID, got.Modules.Demand.Wants[1].Items)
	}
}

// TestDefaults_Demand verifies the constructors of spec-demand.md 9: the want,
// the item and the two taste rows, and that a want built from them with the
// required fields validates.
//
// Requirement: FR-CFG-007, FR-DEM-041
func TestDefaults_Demand(t *testing.T) {
	t.Parallel()
	if got, want := economysim.DefaultWant(), (economysim.Want{Period: 1, Elasticity: 15_000, Ceiling: 15_000, Substitution: 2}); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("DefaultWant() = %+v, want %+v", got, want)
	}
	if got := economysim.DefaultWantItem(); got.Weight != 10_000 || got.Commodity != "" {
		t.Errorf("DefaultWantItem() = %+v, want weight 10000", got)
	}
	if got := economysim.DefaultTaste(); got.Scale != 10_000 {
		t.Errorf("DefaultTaste().Scale = %d, want 10000", got.Scale)
	}
	if got := economysim.DefaultRegionTaste(); got.Scale != 10_000 {
		t.Errorf("DefaultRegionTaste().Scale = %d, want 10000", got.Scale)
	}
	c := referenceConfig()
	w, it, ta, rt := economysim.DefaultWant(), economysim.DefaultWantItem(), economysim.DefaultTaste(), economysim.DefaultRegionTaste()
	w.ID, w.Quantity = "comfort", 1
	it.Commodity = "refined"
	w.Items = []economysim.WantItem{it}
	ta.Role, ta.Want = "cook", "comfort"
	rt.Region, rt.Want = "north", "comfort"
	c.Modules.Demand = &economysim.Demand{Wants: []economysim.Want{w}, Tastes: []economysim.Taste{ta}, RegionTastes: []economysim.RegionTaste{rt}}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(a want from the constructors) = %v, want nil", err)
	}
}
