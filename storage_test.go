package economysim_test

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the storage module's configuration surface (spec-storage.md 12, 13):
// the failing case of each rule of 13.1, their order and the warnings of 13.2. The
// hash is in storage_internal_test.go and the loader in config/storage_test.go.

// storageConfig returns referenceConfig with the storage module on: a base room of
// 100 and 100 for every role, the cooks unlimited by a row with no base, and the
// growers with a base room of 2 000 and 2 000, a bin and a shed. Two classes, bin and
// shed. Every default commodity has a mass and a bulk of 1, so the upkeep of a
// household (4) and the largest start stock (60, the growers') fit, and the module
// raises no warning.
func storageConfig() economysim.Config {
	c := referenceConfig()
	c.Modules.Storage = &economysim.Storage{
		Base: &economysim.Capacity{Mass: 100, Bulk: 100},
		Roles: []economysim.RoleRoom{
			{Role: "cook"},
			{Role: "grower", Base: &economysim.Capacity{Mass: 2_000, Bulk: 2_000}, Stores: []string{"bin", "shed"}},
		},
		Classes: []economysim.StoreClass{
			{ID: "bin", Mass: 400, Bulk: 300},
			{ID: "shed", Mass: 50, Bulk: 50, Label: "a lean-to"},
		},
	}
	return c
}

// storageLevy adds a levy whose target is the stores of a class to the money supply
// module, and returns the path of the flow.
func storageLevy(c *economysim.Config, target economysim.FlowTarget) []string {
	c.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{
		{ID: "store_levy", Kind: economysim.FlowLevy, Target: target, Amount: 1},
	}}
	return []string{"modules", "money_supply", "flows", "store_levy"}
}

// TestValidateStorage_ConfigIsValid verifies that the configuration the storage
// tests start from validates and raises none of the module's warnings, so every
// failing case breaks one rule.
//
// Requirement: FR-STO-038
//
// Guard: the field cases start from storageConfig; if it did not validate, each
// case would pass for the wrong reason.
func TestValidateStorage_ConfigIsValid(t *testing.T) {
	t.Parallel()
	c := storageConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("storageConfig().Validate() = %v, want nil", err)
	}
	for _, w := range c.Warnings() {
		if strings.HasPrefix(strings.Join(w.Path, "."), "modules.storage") {
			t.Errorf("storageConfig().Warnings() has %v, want none of the module's", w)
		}
	}
	empty := referenceConfig()
	empty.Modules.Storage = &economysim.Storage{}
	if err := empty.Validate(); err != nil {
		t.Errorf("the empty module .Validate() = %v, want nil: it is on, limits no role and does nothing", err)
	}
}

// storageFieldCases are the failing cases of the rows of the storage module in the
// parameter-owner table (spec-storage.md 12, 13.1, FR-CFG-085): each breaks one
// field of the valid configuration of storageConfig. buildFieldCases appends them to
// the table that the registry test reads.
func storageFieldCases() []fieldCase {
	field := storageField(func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config, m *economysim.Storage)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				*c = storageConfig()
				set(c, c.Modules.Storage)
			}}
	})
	base := []string{"modules", "storage", "base"}
	row := func(i int, key ...string) []string {
		return append([]string{"modules", "storage", "roles[" + strconv.Itoa(i) + "]"}, key...)
	}
	class := func(i int, key ...string) []string {
		return append([]string{"modules", "storage", "classes[" + strconv.Itoa(i) + "]"}, key...)
	}
	over, maxRoom := strconv.FormatInt(economysim.MaxRoom+1, 10), strconv.FormatInt(economysim.MaxRoom, 10)
	var cs []fieldCase
	cs = append(cs, storageModuleCases(field, row)...)
	cs = append(cs, storageClassCases(field, class, over, maxRoom)...)
	return append(cs, storageRoomCases(field, base, row, over, maxRoom)...)
}

// onlyForester makes the module's Base the only room of one role: every other role
// gets an unlimited row, so a break of the Base is one error. The module's Base
// becomes mass and bulk, and the forester's start stock is dropped when noStock, so
// a base below the upkeep breaks that rule alone.
func onlyForester(c *economysim.Config, m *economysim.Storage, mass, bulk int64, noStock bool) {
	for _, id := range []string{"digger", "maker", "refiner", "visitor"} {
		m.Roles = append(m.Roles, economysim.RoleRoom{Role: id})
	}
	m.Base = &economysim.Capacity{Mass: mass, Bulk: bulk}
	for i := range c.Roles {
		if c.Roles[i].ID == "forester" && noStock {
			c.Roles[i].StartStock = nil
		}
	}
}

// storageField builds a failing case from one field of the valid configuration: the
// field, the case, the rule, the path, the value and bound, and what breaks it.
type storageField func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config, m *economysim.Storage)) fieldCase

// storageModuleCases are the cases of the module's lists and of the role rows: the
// duplicate and unknown rows, the stores of a row.
func storageModuleCases(
	field storageField,
	row func(i int, key ...string) []string,
) []fieldCase {
	return []fieldCase{
		field("Storage.Roles", "two rows for one role", economysim.ErrDuplicateID, row(2, "role"), "grower", "",
			func(_ *economysim.Config, m *economysim.Storage) {
				m.Roles = append(m.Roles, economysim.RoleRoom{Role: "grower", Base: &economysim.Capacity{Mass: 2_000, Bulk: 2_000}})
			}),
		field("RoleRoom.Role", "a role that is not defined", economysim.ErrUnknownRole, row(1, "role"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Storage) { m.Roles[1].Role = "nowhere" }),
		field("RoleRoom.Stores", "a class that is not defined", economysim.ErrUnknownStoreClass, row(1, "stores[0]"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Storage) { m.Roles[1].Stores = []string{"nowhere"} }),
		field("RoleRoom.Stores", "more than MaxStoresPerAgent", economysim.ErrOutOfBounds, row(1, "stores"), "9", "8",
			func(_ *economysim.Config, m *economysim.Storage) {
				m.Roles[1].Stores = slices.Repeat([]string{"shed"}, 9)
			}),
		field("RoleRoom.Stores", "on a role with no base room", economysim.ErrNotLimited, row(0, "stores"), "1", "0",
			func(_ *economysim.Config, m *economysim.Storage) { m.Roles[0].Stores = []string{"bin"} }),
		field("Storage.Classes", "more than MaxStoreClasses", economysim.ErrOutOfBounds, []string{"modules", "storage", "classes"}, "66", "64",
			func(_ *economysim.Config, m *economysim.Storage) {
				for i := range economysim.MaxStoreClasses {
					m.Classes = append(m.Classes, economysim.StoreClass{ID: "k" + strconv.Itoa(i), Mass: 1})
				}
			}),
	}
}

// storageClassCases are the cases of the fields of a store class.
func storageClassCases(
	field storageField,
	class func(i int, key ...string) []string, over, maxRoom string,
) []fieldCase {
	return []fieldCase{
		field("StoreClass.ID", "not an ID", economysim.ErrInvalidID, class(1, "id"), "Bad ID", "",
			func(_ *economysim.Config, m *economysim.Storage) { m.Classes[1].ID = "Bad ID" }),
		field("StoreClass.ID", "twice", economysim.ErrDuplicateID, class(1, "id"), "bin", "",
			func(_ *economysim.Config, m *economysim.Storage) { m.Classes[1].ID = "bin" }),
		field("StoreClass.Mass", "above MaxRoom", economysim.ErrStoreShape, class(0, "mass"), over, maxRoom,
			func(_ *economysim.Config, m *economysim.Storage) { m.Classes[0].Mass = economysim.MaxRoom + 1 }),
		field("StoreClass.Mass", "below 0", economysim.ErrStoreShape, class(0, "mass"), "-1", maxRoom,
			func(_ *economysim.Config, m *economysim.Storage) { m.Classes[0].Mass = -1 }),
		field("StoreClass.Bulk", "above MaxRoom", economysim.ErrStoreShape, class(0, "bulk"), over, maxRoom,
			func(_ *economysim.Config, m *economysim.Storage) { m.Classes[0].Bulk = economysim.MaxRoom + 1 }),
		field("StoreClass.Bulk", "both capacities 0", economysim.ErrStoreShape, class(1), "0", "1",
			func(_ *economysim.Config, m *economysim.Storage) { m.Classes[1].Mass, m.Classes[1].Bulk = 0, 0 }),
		field("StoreClass.Needs", "a flag while the settlements module is not built", economysim.ErrModuleDependency, class(0, "needs"), "market", "settlements",
			func(_ *economysim.Config, m *economysim.Storage) { m.Classes[0].Needs = "market" }),
		field("StoreClass.Label", "over MaxLabelBytes", economysim.ErrOutOfBounds, class(0, "label"), "257", "256",
			func(_ *economysim.Config, m *economysim.Storage) { m.Classes[0].Label = strings.Repeat("x", 257) }),
	}
}

// storageRoomCases are the cases of the base room and of the rules that read it:
// its bounds, the upkeep and the start stock, and the target of a flow.
func storageRoomCases(
	field storageField,
	base []string, row func(i int, key ...string) []string, over, maxRoom string,
) []fieldCase {
	flow := func(name string, rule error, tail []string, value, bound string, target economysim.FlowTarget, modules bool) fieldCase {
		return fieldCase{field: "FlowTarget.Store", name: name, rules: []error{rule}, value: value, bound: bound,
			paths: [][]string{append([]string{"modules", "money_supply", "flows", "store_levy"}, tail...)},
			mutate: func(c *economysim.Config) {
				*c = storageConfig()
				if !modules {
					c.Modules.Storage = nil
				}
				storageLevy(c, target)
			}}
	}
	return []fieldCase{
		field("Storage.Base", "mass above MaxRoom", economysim.ErrOutOfBounds, append(slices.Clone(base), "mass"), over, maxRoom,
			func(_ *economysim.Config, m *economysim.Storage) { m.Base.Mass = economysim.MaxRoom + 1 }),
		field("Storage.Base", "below one tick's upkeep", economysim.ErrRoomShape, append(slices.Clone(base), "mass"), "3", "4",
			func(c *economysim.Config, m *economysim.Storage) { onlyForester(c, m, 3, 100, true) }),
		field("Storage.Base", "below the start stock", economysim.ErrRoomShape, base, "start_stock", "room",
			func(c *economysim.Config, m *economysim.Storage) { onlyForester(c, m, 10, 10, false) }),
		field("Capacity.Mass", "below 0", economysim.ErrOutOfBounds, row(1, "base", "mass"), "-1", "0",
			func(_ *economysim.Config, m *economysim.Storage) { m.Roles[1].Base.Mass = -1 }),
		field("Capacity.Bulk", "above MaxRoom", economysim.ErrOutOfBounds, row(1, "base", "bulk"), over, maxRoom,
			func(_ *economysim.Config, m *economysim.Storage) { m.Roles[1].Base.Bulk = economysim.MaxRoom + 1 }),
		field("RoleRoom.Base", "below one tick's upkeep, bulk", economysim.ErrRoomShape, row(1, "base", "bulk"), "3", "4",
			func(_ *economysim.Config, m *economysim.Storage) { m.Roles[1].Base.Bulk = 3 }),
		field("RoleRoom.Base", "below the start stock without a store to make up", economysim.ErrRoomShape, row(1, "base"), "start_stock", "room",
			func(_ *economysim.Config, m *economysim.Storage) {
				m.Roles[1].Base = &economysim.Capacity{Mass: 10, Bulk: 10}
				m.Roles[1].Stores = nil
			}),
		flow("a class that is not defined", economysim.ErrUnknownStoreClass, []string{"target", "store"}, "nowhere", "",
			economysim.FlowTarget{Kind: "stores", Store: "nowhere"}, true),
		flow("the module off", economysim.ErrModuleDependency, []string{"target", "kind"}, "stores", "",
			economysim.FlowTarget{Kind: "stores", Store: "bin"}, false),
		flow("a role", economysim.ErrFlowShape, []string{"target", "role"}, "grower", "",
			economysim.FlowTarget{Kind: "stores", Role: "grower"}, true),
		flow("on a target that is not stores", economysim.ErrFlowShape, []string{"target", "store"}, "bin", "",
			economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: "grower", Store: "bin"}, true),
		flow("not built while every other rule holds", economysim.ErrModuleDependency, []string{"target", "kind"}, "stores", "built",
			economysim.FlowTarget{Kind: "stores", Store: "bin"}, true),
	}
}

// TestValidateStorage_Errors verifies each rule of spec-storage.md 13.1 with a
// configuration that breaks only that rule: the sentinel by errors.Is, the path of
// the offending row by its index in the order written, and the value.
//
// Requirement: FR-STO-038, FR-STO-002, FR-STO-003, FR-CFG-085
func TestValidateStorage_Errors(t *testing.T) {
	t.Parallel()
	for _, tt := range storageFieldCases() {
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

// errorAt reports whether err holds a *ValidationError of the first rule of the case
// at its first path, with the value and the bound the case names (empty is any).
func errorAt(err error, tt fieldCase) bool {
	for _, e := range unwrapAll(err) {
		var ve *economysim.ValidationError
		if !errors.As(e, &ve) || !errors.Is(ve, tt.rules[0]) || !slices.Equal(ve.Path, tt.paths[0]) {
			continue
		}
		if (tt.value == "" || ve.Value == tt.value) && (tt.bound == "" || ve.Bound == tt.bound) {
			return true
		}
	}
	return false
}

// TestValidateStorage_StartStockUsesTheLargestWeight verifies the rule
// room_below_start_stock for a cohort: a store is not multiplied by the weight, so
// a stock that fits one household's base room plus a store fits no longer when the
// cohort is larger, and the rule reads the largest weight of any group of the role.
//
// Requirement: FR-STO-038, FR-STO-004
//
// Guard: a rule that multiplied the stores by the weight, or read the world's Scale
// and not the group's, would accept a cohort whose start stock cannot be held.
func TestValidateStorage_StartStockUsesTheLargestWeight(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		weight  int64 // the Scale of the grower group
		base    int64 // the grower's base room, both dimensions
		stores  []string
		wantErr bool
	}{
		// The growers' start stock is 60 per household.
		{"one household fits its base", 1, 60, nil, false},
		{"one household over its base", 1, 59, nil, true},
		{"a store makes up the difference at weight 1", 1, 20, []string{"bin"}, false},
		{"a store does not make up the difference of a cohort of 10", 10, 20, []string{"bin"}, true},
		{"a cohort of 10 fits 10 times the base", 10, 60, nil, false},
		{"a cohort of 10 and a store fits a smaller base", 10, 56, []string{"bin"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := storageConfig()
			c.Modules.Storage.Roles[1].Base = &economysim.Capacity{Mass: tt.base, Bulk: tt.base}
			c.Modules.Storage.Roles[1].Stores = tt.stores
			for i := range c.Groups {
				if c.Groups[i].Role == "grower" {
					c.Groups[i].Scale = tt.weight
					c.Groups[i].Households = 10 * tt.weight * 100
				}
			}
			err := c.Validate()
			if got := errors.Is(err, economysim.ErrRoomShape); got != tt.wantErr {
				t.Errorf("Validate() = %v, want ErrRoomShape = %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateStorage_Order verifies that a configuration with two mistakes reports
// the one the order of the validator puts first and none of the later ones: a class
// that is malformed hides a base that is out of bounds, which hides a row that names
// no role, which hides a room that is too small.
//
// Requirement: FR-STO-038
//
// Guard: the first error must be the same for the same mistake; a validator that
// reported all of them, sorted by path, would put the base first.
func TestValidateStorage_Order(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		break_ []func(m *economysim.Storage)
		first  error
		later  error
	}{
		{"a class shape before the base bound", []func(*economysim.Storage){
			func(m *economysim.Storage) { m.Classes[0].Mass, m.Classes[0].Bulk = 0, 0 },
			func(m *economysim.Storage) { m.Base.Mass = economysim.MaxRoom + 1 },
		}, economysim.ErrStoreShape, economysim.ErrOutOfBounds},
		{"the base bound before a row", []func(*economysim.Storage){
			func(m *economysim.Storage) { m.Base.Mass = -1 },
			func(m *economysim.Storage) { m.Roles[1].Role = "nowhere" },
		}, economysim.ErrOutOfBounds, economysim.ErrUnknownRole},
		{"a row before the room rules", []func(*economysim.Storage){
			func(m *economysim.Storage) { m.Roles[1].Role = "nowhere" },
			func(m *economysim.Storage) { m.Base.Mass = 3 },
		}, economysim.ErrUnknownRole, economysim.ErrRoomShape},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := storageConfig()
			for _, b := range tt.break_ {
				b(c.Modules.Storage)
			}
			err := c.Validate()
			if !errors.Is(err, tt.first) || errors.Is(err, tt.later) {
				t.Errorf("Validate() = %v, want %v and not %v", err, tt.first, tt.later)
			}
		})
	}
}

// TestValidateStorage_OffReportsNothing verifies that the module off checks none of
// its rules and that a pointer that is nil validates a configuration with the
// fields of the module absent.
//
// Requirement: FR-STO-001
func TestValidateStorage_OffReportsNothing(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("referenceConfig().Validate() = %v, want nil", err)
	}
	if ws := c.Warnings(); slices.ContainsFunc(ws, func(w economysim.Warning) bool { return w.Rule == "no_limited_role" }) {
		t.Errorf("the module off Warnings() has no_limited_role: %v", ws)
	}
}

// TestWarnings_Storage verifies the five warnings of spec-storage.md 13.2 that need
// no other module, each with its path, none failing validation, and none on the
// valid configuration.
//
// Requirement: FR-STO-038
func TestWarnings_Storage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		build func() economysim.Config
		want  string // rule@path
	}{
		{"no role is limited", func() economysim.Config {
			c := storageConfig()
			c.Modules.Storage.Base = nil
			c.Modules.Storage.Roles = nil
			return c
		}, "no_limited_role@modules.storage"},
		{"the empty module", func() economysim.Config {
			c := referenceConfig()
			c.Modules.Storage = &economysim.Storage{}
			return c
		}, "no_limited_role@modules.storage"},
		{"a base below the cover of the upkeep", func() economysim.Config {
			c := storageConfig()
			c.Modules.Storage.Base = &economysim.Capacity{Mass: 60, Bulk: 60}
			c.Modules.Storage.Roles[1].Base = &economysim.Capacity{Mass: 11, Bulk: 2_000}
			return c
		}, "room_below_cover@modules.storage.roles[1].base"},
		{"a recipe that cannot start", func() economysim.Config {
			c := storageConfig()
			c.Modules.Storage.Roles[1].Base = &economysim.Capacity{Mass: 100, Bulk: 2_000}
			c.Modules.Storage.Roles[1].Stores = nil
			return c
		}, "room_below_recipe@modules.storage.roles[1].base.recipe.grow_staple"},
		{"a class that nothing names", func() economysim.Config {
			c := storageConfig()
			c.Modules.Storage.Classes = append(c.Modules.Storage.Classes, economysim.StoreClass{ID: "vault", Mass: 9})
			return c
		}, "store_class_unused@modules.storage.classes[2]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tt.build()
			var got []string
			for _, w := range c.Warnings() {
				if p := strings.Join(w.Path, "."); strings.HasPrefix(p, "modules.storage") {
					got = append(got, w.Rule+"@"+p)
				}
			}
			if !slices.Contains(got, tt.want) {
				t.Errorf("Warnings() has %q, want it to include %q", got, tt.want)
			}
		})
	}
}

// TestWarnings_StorageLevyBelowUpkeep verifies store_levy_below_upkeep: a levy on
// stores whose Exempt is below one tick's upkeep cost of a limited role warns, one
// whose Exempt covers it does not, and a flow that names a class counts as using it.
// A stores flow does not validate yet, so the warning is read on a configuration that
// does not.
//
// Requirement: FR-STO-038
func TestWarnings_StorageLevyBelowUpkeep(t *testing.T) {
	t.Parallel()
	rules := func(c economysim.Config) []string {
		var out []string
		for _, w := range c.Warnings() {
			out = append(out, w.Rule+"@"+strings.Join(w.Path, "."))
		}
		return out
	}
	c := storageConfig()
	path := storageLevy(&c, economysim.FlowTarget{Kind: "stores", Store: "shed"})
	want := "store_levy_below_upkeep@" + strings.Join(path, ".")
	if got := rules(c); !slices.Contains(got, want) {
		t.Errorf("Warnings() = %q, want %q for an Exempt of 0", got, want)
	}
	if got := rules(c); slices.Contains(got, "store_class_unused@modules.storage.classes[1]") {
		t.Errorf("Warnings() = %q, want shed used by the flow", got)
	}
	c.Modules.MoneySupply.Flows[0].Exempt = 1_000_000
	if got := rules(c); slices.Contains(got, want) {
		t.Errorf("Warnings() = %q, want no %q for an Exempt that covers the upkeep", got, want)
	}
}

// FuzzValidateStorage verifies, for any base room, class capacities, store list and
// class ID written into the valid configuration, that Validate and Warnings never
// panic, that every error of the module is a *ValidationError whose path starts at
// the module or a flow, and that a configuration Validate accepts is accepted again
// after a canonical round (the hash does not depend on it).
//
// Requirement: FR-STO-038
func FuzzValidateStorage(f *testing.F) {
	f.Add(int64(100), int64(100), int64(400), int64(300), "bin", 1, "bin")
	f.Add(int64(-1), int64(economysim.MaxRoom+1), int64(0), int64(0), "Bad", 9, "nowhere")
	f.Add(int64(0), int64(0), int64(1), int64(0), "", 0, "")
	f.Fuzz(func(t *testing.T, baseMass, baseBulk, mass, bulk int64, id string, n int, store string) {
		c := storageConfig()
		m := c.Modules.Storage
		m.Base = &economysim.Capacity{Mass: baseMass, Bulk: baseBulk}
		m.Classes = []economysim.StoreClass{{ID: id, Mass: mass, Bulk: bulk}}
		m.Roles[1].Stores = slices.Repeat([]string{store}, max(0, min(n, 12)))
		err := c.Validate()
		_ = c.Warnings()
		onlyValidationErrors(t, err)
		if err == nil && c.Validate() != nil {
			t.Fatal("Validate() accepted the configuration once and rejected it again")
		}
	})
}

// onlyValidationErrors fails the test for an entry of err that is not a
// *ValidationError.
func onlyValidationErrors(t *testing.T, err error) {
	t.Helper()
	for _, e := range unwrapAll(err) {
		var ve *economysim.ValidationError
		if e != nil && !errors.As(e, &ve) {
			t.Fatalf("Validate() has %v, want only *ValidationError", e)
		}
	}
}

// TestSentinels_Storage verifies that the eight sentinels of the storage module have
// the text and the message of their own, are matched by errors.Is through a
// ValidationError that carries them, and match no other of the eight.
//
// Requirement: FR-STO-032, FR-STO-033, FR-STO-038
//
// Guard: two sentinels with one message cannot be told apart in a log, and one that
// does not round-trip would make a rule unmatchable.
func TestSentinels_Storage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		err  error
		want string
	}{
		{economysim.ErrStoreShape, "store class has the wrong shape"},
		{economysim.ErrRoomShape, "room is too small for the role"},
		{economysim.ErrNotLimited, "role is not limited"},
		{economysim.ErrUnknownStoreClass, "unknown store class"},
		{economysim.ErrStoreLocked, "store class is locked in the region"},
		{economysim.ErrAgentAway, "agent is away"},
		{economysim.ErrStoreInUse, "store is in use by a facility"},
		{economysim.ErrNoRoom, "no room"},
	}
	for _, tt := range tests {
		if got := tt.err.Error(); got != tt.want {
			t.Errorf("%v message = %q, want %q", tt.err, got, tt.want)
		}
		wrapped := error(&economysim.ValidationError{Path: []string{"x"}, Rule: tt.err})
		for _, other := range tests {
			if got := errors.Is(wrapped, other.err); got != (other.err == tt.err) {
				t.Errorf("errors.Is(ValidationError{%v}, %v) = %v, want %v", tt.err, other.err, got, other.err == tt.err)
			}
		}
	}
}
