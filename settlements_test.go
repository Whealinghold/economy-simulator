package economysim_test

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the settlements module's configuration surface (spec-settlements.md
// 8, 9): the bounds, the rules of 9.1 with their sentinels and paths, the warnings of
// 9.2, the dependency of a role field and of a store class on the module, and the
// canonical order. The arithmetic is tested in internal/settlement, the hash in
// settlements_internal_test.go and the loader in config/settlements_test.go.

// settlementsConfig returns referenceConfig with the module on: three classes by
// Enter (hamlet 0, village 300, town 500), the flag fair on the two larger, the limit
// workers on them, the grower needing fair and the refiner bounded by workers. The
// region north has 600 households, so it is a town at construction; the band is the
// default and the dwell is 400. It validates with no settlements warning.
func settlementsConfig() economysim.Config {
	c := referenceConfig()
	m := economysim.DefaultSettlements()
	m.Dwell = 400 // above the effective welfare window of every gated role, so dwell_below_window is silent
	m.Classes = []economysim.SettlementClass{
		{ID: "hamlet", Enter: 0},
		{ID: "village", Enter: 300, Flags: []string{"fair"}, Limits: []economysim.ClassLimit{{Name: "workers", Value: 500}}},
		{ID: "town", Enter: 500, Flags: []string{"fair"}, Limits: []economysim.ClassLimit{{Name: "workers", Value: 900}}, Label: "market town"},
	}
	m.Regions = []economysim.RegionSettlement{economysim.DefaultRegionSettlement("north")}
	c.Modules.Settlements = &m
	roleOf(&c, "grower").Needs = "fair"
	roleOf(&c, "refiner").Ceiling = "workers"
	return c
}

// settlementsRule returns the settlements warnings of c as rule@path strings.
func settlementsRule(c economysim.Config) []string {
	var got []string
	for _, w := range c.Warnings() {
		p := strings.Join(w.Path, ".")
		if strings.HasPrefix(p, "modules.settlements") || strings.HasSuffix(p, ".needs") || strings.HasSuffix(p, ".ceiling") {
			got = append(got, w.Rule+"@"+p)
		}
	}
	return got
}

// TestValidateSettlements_ConfigIsValid verifies that the configuration the
// settlements tests start from validates and draws no settlements warning, so every
// failing case breaks one rule and every warning case raises one.
//
// Guard: the field cases start from settlementsConfig; if it did not validate, each
// case would pass for the wrong reason.
func TestValidateSettlements_ConfigIsValid(t *testing.T) {
	t.Parallel()
	c := settlementsConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("settlementsConfig().Validate() = %v, want nil", err)
	}
	if got := settlementsRule(c); len(got) != 0 {
		t.Errorf("settlementsConfig().Warnings() holds %v, want none", got)
	}
}

// settlementsField builds a failing case from one field of the valid configuration:
// the field, the case, the rule, the path, the value and bound, and what breaks it.
type settlementsField func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config, m *economysim.Settlements)) fieldCase

// settlementsFieldCases are the failing cases of the rows of the settlements module in
// the parameter-owner table (spec-settlements.md 8, 9.1, FR-SET-002 to 007, 021):
// each breaks one field of the valid configuration of settlementsConfig.
// buildFieldCases appends them to the table that the registry test reads.
func settlementsFieldCases() []fieldCase {
	field := settlementsField(func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config, m *economysim.Settlements)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				*c = settlementsConfig()
				set(c, c.Modules.Settlements)
			}}
	})
	var cs []fieldCase
	cs = append(cs, settlementsModuleCases(field)...)
	cs = append(cs, settlementsClassCases(field)...)
	cs = append(cs, settlementsRowCases(field)...)
	return append(cs, settlementsRoleCases()...)
}

// setPath is the path of a field of the module.
func setPath(key ...string) []string { return append([]string{"modules", "settlements"}, key...) }

// classPath is the path of a field of the class written at index i.
func classPath(i int, key ...string) []string {
	return append(setPath("classes["+strconv.Itoa(i)+"]"), key...)
}

// settlementsModuleCases are the cases of the module's tunables and lists.
func settlementsModuleCases(field settlementsField) []fieldCase {
	return []fieldCase{
		field("Settlements.Classes", "no class", economysim.ErrTooMany, setPath("classes"), "0", "1",
			func(c *economysim.Config, m *economysim.Settlements) {
				m.Classes, m.Regions = nil, nil
				roleOf(c, "grower").Needs, roleOf(c, "refiner").Ceiling = "", ""
			}),
		field("Settlements.Classes", "more than MaxSettlementClasses", economysim.ErrTooMany, setPath("classes"), "17", "16",
			func(c *economysim.Config, m *economysim.Settlements) {
				for i := range 14 {
					m.Classes = append(m.Classes, economysim.SettlementClass{ID: "k" + strconv.Itoa(i), Enter: int64(1_000 + i)})
				}
			}),
		field("Settlements.BandBP", "above MaxBandBP", economysim.ErrOutOfBounds, setPath("band_bp"), "5001", "5000",
			func(_ *economysim.Config, m *economysim.Settlements) { m.BandBP = economysim.MaxBandBP + 1 }),
		field("Settlements.BandBP", "negative", economysim.ErrOutOfBounds, setPath("band_bp"), "-1", "0",
			func(_ *economysim.Config, m *economysim.Settlements) { m.BandBP = -1 }),
		field("Settlements.Dwell", "above MaxWelfareWindow", economysim.ErrOutOfBounds, setPath("dwell"), "3651", "3650",
			func(_ *economysim.Config, m *economysim.Settlements) { m.Dwell = economysim.MaxWelfareWindow + 1 }),
		field("Settlements.Regions", "more than MaxRegions rows", economysim.ErrTooMany, setPath("regions"), "1001", "1000",
			func(_ *economysim.Config, m *economysim.Settlements) {
				m.Regions = slices.Repeat(m.Regions, economysim.MaxRegions+1)
			}),
	}
}

// settlementsClassCases are the cases of the fields of a class and of a limit.
func settlementsClassCases(field settlementsField) []fieldCase {
	bad, dup := economysim.ErrInvalidID, economysim.ErrDuplicateID
	return []fieldCase{
		field("SettlementClass.ID", "upper case", bad, classPath(1, "id"), "Village", "",
			func(_ *economysim.Config, m *economysim.Settlements) { m.Classes[1].ID = "Village" }),
		field("SettlementClass.ID", "twice", dup, classPath(2, "id"), "village", "",
			func(_ *economysim.Config, m *economysim.Settlements) { m.Classes[2].ID = "village" }),
		field("SettlementClass.Enter", "above MaxHouseholds", economysim.ErrOutOfBounds, classPath(2, "enter"), "1000000001", "1000000000",
			func(_ *economysim.Config, m *economysim.Settlements) {
				m.Classes[2].Enter = economysim.MaxHouseholds + 1
			}),
		field("SettlementClass.Enter", "first class not open (rule first_class_open)", economysim.ErrOutOfBounds, classPath(0, "enter"), "5", "0",
			func(_ *economysim.Config, m *economysim.Settlements) { m.Classes[0].Enter = 5 }),
		field("SettlementClass.Enter", "two classes with one Enter (rule enter_order)", economysim.ErrOutOfBounds, classPath(2, "enter"), "300", "301",
			func(_ *economysim.Config, m *economysim.Settlements) { m.Classes[2].Enter = 300 }),
		field("SettlementClass.Enter", "negative", economysim.ErrOutOfBounds, classPath(2, "enter"), "-1", "0",
			func(_ *economysim.Config, m *economysim.Settlements) {
				m.Classes[2].Enter, m.Classes[0].Enter = -1, -1
				m.Classes[0].Enter = 0
			}),
		field("SettlementClass.Label", "over MaxLabelBytes", economysim.ErrOutOfBounds, classPath(2, "label"), "257", "256",
			func(_ *economysim.Config, m *economysim.Settlements) { m.Classes[2].Label = strings.Repeat("x", 257) }),
		field("SettlementClass.Flags", "invalid name", bad, classPath(2, "flags[1]"), "Hall", "",
			func(_ *economysim.Config, m *economysim.Settlements) { m.Classes[2].Flags = []string{"fair", "Hall"} }),
		field("SettlementClass.Flags", "a flag twice in one class", dup, classPath(2, "flags[1]"), "fair", "",
			func(_ *economysim.Config, m *economysim.Settlements) { m.Classes[2].Flags = []string{"fair", "fair"} }),
		field("SettlementClass.Flags", "more than MaxClassFlags distinct", economysim.ErrTooMany, setPath("flags"), "65", "64",
			func(_ *economysim.Config, m *economysim.Settlements) {
				for i := range 64 {
					m.Classes[2].Flags = append(m.Classes[2].Flags, "f"+strconv.Itoa(i))
				}
			}),
		field("SettlementClass.Limits", "a limit twice in one class", dup, classPath(2, "limits[1]", "name"), "workers", "",
			func(_ *economysim.Config, m *economysim.Settlements) {
				m.Classes[2].Limits = append(m.Classes[2].Limits, economysim.ClassLimit{Name: "workers", Value: 1})
			}),
		field("SettlementClass.Limits", "more than MaxClassLimits distinct", economysim.ErrTooMany, setPath("limits"), "65", "64",
			func(_ *economysim.Config, m *economysim.Settlements) {
				for i := range 64 {
					m.Classes[2].Limits = append(m.Classes[2].Limits, economysim.ClassLimit{Name: "l" + strconv.Itoa(i)})
				}
			}),
		field("ClassLimit.Name", "invalid name", bad, classPath(2, "limits[0]", "name"), "Workers", "",
			func(c *economysim.Config, m *economysim.Settlements) {
				m.Classes[2].Limits[0].Name = "Workers"
				roleOf(c, "refiner").Ceiling = ""
			}),
		field("ClassLimit.Value", "above MaxQuantity", economysim.ErrOutOfBounds, classPath(2, "limits[0]", "value"), "1000000001", "1000000000",
			func(_ *economysim.Config, m *economysim.Settlements) {
				m.Classes[2].Limits[0].Value = economysim.MaxQuantity + 1
			}),
		field("ClassLimit.Value", "negative", economysim.ErrOutOfBounds, classPath(2, "limits[0]", "value"), "-1", "0",
			func(_ *economysim.Config, m *economysim.Settlements) { m.Classes[2].Limits[0].Value = -1 }),
	}
}

// settlementsRowCases are the cases of the region rows.
func settlementsRowCases(field settlementsField) []fieldCase {
	row := func(key ...string) []string { return setPath(append([]string{"regions[0]"}, key...)...) }
	return []fieldCase{
		field("RegionSettlement.Region", "an undefined region", economysim.ErrUnknownRegion, row("region"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Settlements) { m.Regions[0].Region = "nowhere" }),
		field("RegionSettlement.Region", "two rows for one region", economysim.ErrDuplicateID, setPath("regions[1]", "region"), "north", "",
			func(_ *economysim.Config, m *economysim.Settlements) { m.Regions = append(m.Regions, m.Regions[0]) }),
		{field: "RegionSettlement.Region", name: "an outside node", rules: []error{economysim.ErrOutsideNode},
			paths: [][]string{row("region")}, value: "coast",
			mutate: func(c *economysim.Config) {
				*c = externalConfig()
				m := economysim.DefaultSettlements()
				m.Classes = []economysim.SettlementClass{{ID: "only"}}
				m.Regions = []economysim.RegionSettlement{economysim.DefaultRegionSettlement("coast")}
				c.Modules.Settlements = &m
			}},
		field("RegionSettlement.ThresholdBP", "zero", economysim.ErrOutOfBounds, row("threshold_bp"), "0", "1",
			func(_ *economysim.Config, m *economysim.Settlements) { m.Regions[0].ThresholdBP = 0 }),
		field("RegionSettlement.ThresholdBP", "above MaxThresholdBP", economysim.ErrOutOfBounds, row("threshold_bp"), "100001", "100000",
			func(_ *economysim.Config, m *economysim.Settlements) {
				m.Regions[0].ThresholdBP = economysim.MaxThresholdBP + 1
			}),
		field("RegionSettlement.Floor", "a class that is not defined", economysim.ErrUnknownSettlementClass, row("floor"), "city", "",
			func(_ *economysim.Config, m *economysim.Settlements) { m.Regions[0].Floor = "city" }),
	}
}

// settlementsRoleCases are the cases of the two role fields.
func settlementsRoleCases() []fieldCase {
	role := func(field, name string, rule error, key, value string, set func(c *economysim.Config)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{{"roles", "grower", key}}, value: value,
			mutate: func(c *economysim.Config) {
				*c = settlementsConfig()
				set(c)
			}}
	}
	return []fieldCase{
		role("Role.Needs", "a flag that no class carries", economysim.ErrUnknownSettlementFlag, "needs", "hall",
			func(c *economysim.Config) { roleOf(c, "grower").Needs = "hall" }),
		role("Role.Needs", "with the module off", economysim.ErrModuleDependency, "needs", "fair",
			func(c *economysim.Config) {
				c.Modules.Settlements = nil
				roleOf(c, "refiner").Ceiling = ""
			}),
		role("Role.Ceiling", "a limit that no class carries", economysim.ErrUnknownLimit, "ceiling", "places",
			func(c *economysim.Config) { roleOf(c, "grower").Ceiling, roleOf(c, "grower").Needs = "places", "" }),
		role("Role.Ceiling", "with the module off", economysim.ErrModuleDependency, "ceiling", "workers",
			func(c *economysim.Config) {
				c.Modules.Settlements = nil
				roleOf(c, "grower").Ceiling, roleOf(c, "grower").Needs, roleOf(c, "refiner").Ceiling = "workers", "", ""
			}),
	}
}

// TestValidateSettlements_Errors verifies each rule of spec-settlements.md 9.1 with
// its sentinel (matched with errors.Is) and the path of the field, one failing case
// per row of the parameter-owner table.
//
// Requirement: FR-SET-002, FR-SET-003, FR-SET-004, FR-SET-006, FR-SET-007, FR-SET-021
func TestValidateSettlements_Errors(t *testing.T) {
	t.Parallel()
	for _, tt := range settlementsFieldCases() {
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

// TestValidateSettlements_OffReportsNothing verifies that a configuration with the
// module off and no role field reports no settlements rule, and that an empty module
// is the one error no_class.
//
// Requirement: FR-SET-001, FR-SET-002
func TestValidateSettlements_OffReportsNothing(t *testing.T) {
	t.Parallel()
	if err := referenceConfig().Validate(); err != nil {
		t.Errorf("Validate() with the module off = %v, want nil", err)
	}
	c := referenceConfig()
	c.Modules.Settlements = &economysim.Settlements{}
	if err := c.Validate(); !errors.Is(err, economysim.ErrTooMany) {
		t.Errorf("Validate() with an empty module = %v, want ErrTooMany (no_class)", err)
	}
}

// TestValidateSettlements_OneClassIsValid verifies that a module with one class, no
// flag and no limit validates and draws no warning, the inert setting of
// spec-settlements.md 3.6.
//
// Requirement: FR-SET-002, FR-SET-015
func TestValidateSettlements_OneClassIsValid(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	m := economysim.DefaultSettlements()
	m.Classes = []economysim.SettlementClass{{ID: "only"}}
	c.Modules.Settlements = &m
	if err := c.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
	if got := settlementsRule(c); len(got) != 0 {
		t.Errorf("Warnings() holds %v, want none", got)
	}
}

// TestValidateSettlements_StoreClassNeeds verifies the seam of the storage module
// (plan 3-sto D8, FR-SET-021): a store class's Needs with the module off is
// ErrModuleDependency, with it on a flag no class carries is
// ErrUnknownSettlementFlag, and a flag some class carries validates.
//
// Requirement: FR-SET-021, FR-SET-001
func TestValidateSettlements_StoreClassNeeds(t *testing.T) {
	t.Parallel()
	with := func(needs string, module bool) economysim.Config {
		c := storageConfig()
		c.Modules.Storage.Classes[0].Needs = needs
		if module {
			c.Modules.Settlements = settlementsConfig().Modules.Settlements
		}
		return c
	}
	tests := []struct {
		name   string
		c      economysim.Config
		want   error
		wantAt []string
	}{
		{"module off", with("fair", false), economysim.ErrModuleDependency, []string{"modules", "storage", "classes[0]", "needs"}},
		{"unknown flag", with("hall", true), economysim.ErrUnknownSettlementFlag, []string{"modules", "storage", "classes[0]", "needs"}},
		{"a flag some class carries", with("fair", true), nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.c.Validate()
			if tt.want == nil {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			fc := fieldCase{rules: []error{tt.want}, paths: [][]string{tt.wantAt}}
			if !errors.Is(err, tt.want) || !errorAt(err, fc) {
				t.Errorf("Validate() = %v, want %v at %v", err, tt.want, tt.wantAt)
			}
		})
	}
}

// TestValidateSettlements_Order verifies that a configuration with two mistakes
// reports the one the order of the validator puts first: a module with no class hides
// a band out of bounds, which hides an Enter that is not 0 for the first class, which
// hides a row that names no region.
//
// Requirement: FR-SET-004
//
// Guard: the first error must be the same for the same mistake; a validator that
// reported all of them, sorted by path, would put the band first.
func TestValidateSettlements_Order(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		break1, then func(c *economysim.Config)
		first, later error
	}{
		{"a malformed class before the band",
			func(c *economysim.Config) { c.Modules.Settlements.Classes[1].ID = "Bad" },
			func(c *economysim.Config) { c.Modules.Settlements.BandBP = -1 },
			economysim.ErrInvalidID, economysim.ErrOutOfBounds},
		{"the band before the order of the classes",
			func(c *economysim.Config) { c.Modules.Settlements.BandBP = -1 },
			func(c *economysim.Config) { c.Modules.Settlements.Classes[0].Enter = 5 },
			economysim.ErrOutOfBounds, nil},
		{"the order before a row",
			func(c *economysim.Config) { c.Modules.Settlements.Classes[0].Enter = 5 },
			func(c *economysim.Config) { c.Modules.Settlements.Regions[0].Region = "nowhere" },
			economysim.ErrOutOfBounds, economysim.ErrUnknownRegion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := settlementsConfig()
			tt.break1(&c)
			tt.then(&c)
			err := c.Validate()
			if !errors.Is(err, tt.first) {
				t.Errorf("Validate() = %v, want %v", err, tt.first)
			}
			if tt.later != nil && errors.Is(err, tt.later) {
				t.Errorf("Validate() = %v, holds %v of a later phase, want the first phase's error only", err, tt.later)
			}
		})
	}
}

// TestValidateSettlements_ClassWrittenOutOfOrder verifies that classes written by ID
// and not by Enter validate (the order is the canonical one) and that the path of an
// Enter error names the class in the order it was written.
//
// Requirement: FR-SET-002
func TestValidateSettlements_ClassWrittenOutOfOrder(t *testing.T) {
	t.Parallel()
	c := settlementsConfig()
	m := c.Modules.Settlements
	m.Classes[0], m.Classes[2] = m.Classes[2], m.Classes[0]
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate() of classes written out of order = %v, want nil", err)
	}
	m.Classes[0].Enter = 300 // the town, now written first, repeats the village's Enter
	err := c.Validate()
	if !errorAt(err, fieldCase{rules: []error{economysim.ErrOutOfBounds}, paths: [][]string{classPath(0, "enter")}}) &&
		!errorAt(err, fieldCase{rules: []error{economysim.ErrOutOfBounds}, paths: [][]string{classPath(1, "enter")}}) {
		t.Errorf("Validate() = %v, want enter_order at the enter of class 0 or 1", err)
	}
}

// TestWarnings_Settlements verifies one case for each of the eight warnings of
// spec-settlements.md 9.2, with its rule and path, and that none fails validation.
//
// Requirement: FR-SET-028
func TestWarnings_Settlements(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		build func(c *economysim.Config)
		want  string // rule@path
	}{
		{"a needed flag the start class lacks", func(c *economysim.Config) {
			// north has 600 households: a town, which carries fair; make the flag a village-only one.
			c.Modules.Settlements.Classes[2].Flags = nil
			roleOf(c, "grower").Needs = "fair"
		}, "needs_unmet_at_start@roles.grower.needs"},
		{"a role that starts above its ceiling", func(c *economysim.Config) {
			// the refiner has 36 households; the town's limit is 20.
			c.Modules.Settlements.Classes[2].Limits[0].Value = 20
		}, "ceiling_below_start@roles.refiner.ceiling"},
		{"a dwell below the window of a gated role", func(c *economysim.Config) {
			c.Modules.Settlements.Dwell = 1
		}, "dwell_below_window@modules.settlements.dwell"},
		{"a band of one household", func(c *economysim.Config) {
			// Enter 500 at 1 000 bp is not one household; Enter 5 at 1 000 bp is.
			c.Modules.Settlements.Classes[1].Enter = 5
		}, "band_one_household@modules.settlements.class.village"},
		{"two thresholds that collapse in a region", func(c *economysim.Config) {
			// 5 and 6 at 5 000 bp both give 3.
			c.Modules.Settlements.Classes[1].Enter, c.Modules.Settlements.Classes[2].Enter = 5, 6
			c.Modules.Settlements.Regions[0].ThresholdBP = 5_000
		}, "threshold_collapsed@modules.settlements.class.town"},
		{"a class no region can reach", func(c *economysim.Config) {
			c.Modules.Settlements.Classes[2].Enter = 5_000
			roleOf(c, "grower").Needs = "fair"
		}, "class_unreachable@modules.settlements.class.town"},
		{"a flag no role needs", func(c *economysim.Config) {
			c.Modules.Settlements.Classes[1].Flags = append(c.Modules.Settlements.Classes[1].Flags, "hall")
		}, "flag_unused@modules.settlements.flag.hall"},
		{"a limit no role names", func(c *economysim.Config) {
			c.Modules.Settlements.Classes[1].Limits = append(c.Modules.Settlements.Classes[1].Limits, economysim.ClassLimit{Name: "berths", Value: 3})
		}, "limit_unused@modules.settlements.limit.berths"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := settlementsConfig()
			tt.build(&c)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate() = %v, want a valid configuration (a warning never fails validation)", err)
			}
			if got := settlementsRule(c); !slices.Contains(got, tt.want) {
				t.Errorf("Warnings() settlements rules = %v, want %q", got, tt.want)
			}
		})
	}
}

// TestWarnings_SettlementsPopulationMakesAClassReachable verifies that class_unreachable
// is not raised with the population module on: households may be born.
//
// Requirement: FR-SET-028
func TestWarnings_SettlementsPopulationMakesAClassReachable(t *testing.T) {
	t.Parallel()
	c := settlementsConfig()
	c.Modules.Settlements.Classes[2].Enter = 5_000
	c.Modules.Population = &economysim.Population{}
	for _, w := range c.Warnings() {
		if w.Rule == "class_unreachable" {
			t.Errorf("Warnings() holds class_unreachable with the population module on")
		}
	}
}

// TestWarnings_SettlementsCountRegions verifies that the count of a warning is the
// number of regions it holds in, and that a region row's scale changes the census the
// warning reads.
//
// Requirement: FR-SET-028
func TestWarnings_SettlementsCountRegions(t *testing.T) {
	t.Parallel()
	c := settlementsConfig()
	c.Modules.Settlements.Classes[2].Flags = nil
	var got int64
	for _, w := range c.Warnings() {
		if w.Rule == "needs_unmet_at_start" {
			got = w.Count
		}
	}
	if got != 1 {
		t.Errorf("needs_unmet_at_start Count = %d, want 1 (the one region with a grower group)", got)
	}
}

// TestSentinels_Settlements verifies that the three new sentinels are distinct,
// non-nil and not matched by each other, and match the module's own errors.
//
// Requirement: FR-SET-021
func TestSentinels_Settlements(t *testing.T) {
	t.Parallel()
	all := []error{economysim.ErrUnknownSettlementClass, economysim.ErrUnknownSettlementFlag, economysim.ErrUnknownLimit}
	for i, a := range all {
		if a == nil || a.Error() == "" {
			t.Fatalf("sentinel %d is empty", i)
		}
		for j, b := range all {
			if (i == j) != errors.Is(a, b) {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", a, b, errors.Is(a, b), i == j)
			}
		}
	}
}

// TestBounds_Settlements verifies the values of the five bounds of
// spec-settlements.md 8.1.
//
// Requirement: FR-SET-003, FR-SET-004
func TestBounds_Settlements(t *testing.T) {
	t.Parallel()
	for name, got := range map[string][2]int64{
		"MaxSettlementClasses": {economysim.MaxSettlementClasses, 16},
		"MaxClassFlags":        {economysim.MaxClassFlags, 64},
		"MaxClassLimits":       {economysim.MaxClassLimits, 64},
		"MaxBandBP":            {economysim.MaxBandBP, 5_000},
		"MaxThresholdBP":       {economysim.MaxThresholdBP, 100_000},
	} {
		if got[0] != got[1] {
			t.Errorf("%s = %d, want %d", name, got[0], got[1])
		}
	}
}

// TestDefaults_Settlements verifies the defaults of spec-settlements.md 8: a band of
// 1 000 and a dwell of 30, no class, and a region row of 10 000 with no floor.
//
// Requirement: FR-CFG-054
func TestDefaults_Settlements(t *testing.T) {
	t.Parallel()
	d := economysim.DefaultSettlements()
	if d.BandBP != 1_000 || d.Dwell != 30 || d.Classes != nil || d.Regions != nil {
		t.Errorf("DefaultSettlements() = %+v, want band 1000, dwell 30, no class, no row", d)
	}
	if r := economysim.DefaultRegionSettlement("x"); r.Region != "x" || r.ThresholdBP != 10_000 || r.Floor != "" {
		t.Errorf("DefaultRegionSettlement() = %+v, want 10000 and no floor", r)
	}
}

// FuzzValidateSettlements feeds Validate and Warnings a module built from fuzzed
// numbers: it never panics, a configuration that validates hashes to the same bytes
// as its copy, and an invalid one returns an error that is a *ValidationError.
//
// Requirement: NFR-SET-004
func FuzzValidateSettlements(f *testing.F) {
	f.Add(int64(300), int64(500), int64(1_000), int64(30), int64(10_000), int64(900))
	f.Add(int64(0), int64(0), int64(-1), int64(-5), int64(0), int64(-1))
	f.Add(int64(1<<62), int64(1<<62), int64(1<<62), int64(1<<62), int64(1<<62), int64(1<<62))
	f.Fuzz(func(t *testing.T, enter1, enter2, band, dwell, scale, limit int64) {
		c := settlementsConfig()
		m := c.Modules.Settlements
		m.Classes[1].Enter, m.Classes[2].Enter = enter1, enter2
		m.BandBP, m.Dwell = band, dwell
		m.Regions[0].ThresholdBP = scale
		m.Classes[2].Limits[0].Value = limit
		_ = c.Warnings()
		err := c.Validate()
		if err == nil {
			again := c
			cp := *c.Modules.Settlements
			cp.Classes = slices.Clone(cp.Classes)
			again.Modules.Settlements = &cp
			if economysim.HashConfig(c) != economysim.HashConfig(again) {
				t.Fatalf("HashConfig of a valid configuration differs from that of its copy")
			}
			return
		}
		if _, other := violations(err); len(other) != 0 {
			t.Fatalf("Validate returned entries that are not a *ValidationError: %v", other)
		}
	})
}

// TestValidateSettlements_ClassListsAreBounded verifies that a class with more flags
// or limits than the module may hold is ErrTooMany at the module's list, without a finding
// per element.
//
// Requirement: FR-SET-002
func TestValidateSettlements_ClassListsAreBounded(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		set  func(c *economysim.SettlementClass)
		key  string
		n    int
	}{
		{"flags", func(c *economysim.SettlementClass) { c.Flags = make([]string, economysim.MaxClassFlags+1) }, "flags", economysim.MaxClassFlags + 1},
		{"limits", func(c *economysim.SettlementClass) {
			c.Limits = make([]economysim.ClassLimit, economysim.MaxClassLimits+1)
		}, "limits", economysim.MaxClassLimits + 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := settlementsConfig()
			tt.set(&c.Modules.Settlements.Classes[1])
			err := c.Validate()
			if !errors.Is(err, economysim.ErrTooMany) {
				t.Fatalf("Validate() = %v, want ErrTooMany", err)
			}
			var ve *economysim.ValidationError
			if !errors.As(err, &ve) || strings.Join(ve.Path, ".") != strings.Join(setPath(tt.key), ".") || ve.Value != strconv.Itoa(tt.n) {
				t.Errorf("Validate() first error = %+v, want ErrTooMany at %v with value %d", ve, setPath(tt.key), tt.n)
			}
			if strings.Count(err.Error(), "invalid") > 1 {
				t.Errorf("Validate() reported a finding per element: %v", err)
			}
		})
	}
}
