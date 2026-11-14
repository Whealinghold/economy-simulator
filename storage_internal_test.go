package economysim

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"strings"
	"testing"
)

// storageHashConfig is a Config with only the storage module: the tests below read
// its bytes and its resolved rooms, not a world.
func storageHashConfig() Config {
	return Config{Modules: Modules{Storage: &Storage{
		Base: &Capacity{Mass: 100, Bulk: 50},
		Roles: []RoleRoom{
			{Role: "grower", Base: &Capacity{Mass: 9, Bulk: 8}, Stores: []string{"shed", "bin", "bin"}},
			{Role: "cook"},
		},
		Classes: []StoreClass{{ID: "shed", Mass: 5, Needs: "", Label: "a"}, {ID: "bin", Mass: 4, Bulk: 3}},
	}}}
}

// TestHash_StorageFieldIds verifies the bytes of the storage module under id 12 of
// the Modules row (between Merchants 10 and Employment 13), that the pointer is
// written even when empty, that the fields have the ids the encoder documents and
// that a Base is a pointer: written when non-nil even if both of its fields are zero.
//
// Requirement: FR-STO-002, FR-CFG-085
//
// Guard: a pair written out of id order, an empty module that writes nothing, or a
// zero base room that hashes like an unlimited role would hash a configuration
// differently from the one the spec defines.
func TestHash_StorageFieldIds(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	u64 := func(n uint64) []byte { return binary.BigEndian.AppendUint64(nil, n) }
	str := func(s string) []byte { return append(u64(uint64(len(s))), s...) }

	e := &encoder{}
	pairsModules(e, Modules{Merchants: &Merchants{}, Storage: &Storage{}, Employment: &Employment{}})
	at := bytes.Index(e.buf, append(id(12), 0, 0))
	if at < 0 || bytes.Index(e.buf, id(10)) > at || at > bytes.Index(e.buf, append(id(13), 0, 0)) {
		t.Errorf("pairsModules(Merchants, Storage, Employment) = % x, want the empty pair 000c 0000 after id 10 and before id 13", e.buf)
	}

	e = &encoder{}
	pairsStorage(e, Storage{
		Base:    &Capacity{Mass: 7},
		Roles:   []RoleRoom{{Role: "r", Base: &Capacity{}, Stores: []string{"a", "b"}}},
		Classes: []StoreClass{{ID: "c", Mass: 1, Bulk: 2, Needs: "n", Label: "ignored"}},
	})
	want := bytes.Join([][]byte{
		id(1), id(1), u64(7), {0, 0}, // Base: Mass 7, end
		id(2), u64(1), // Roles: one element
		id(1), str("r"), id(2), {0, 0}, id(3), u64(2), str("a"), str("b"), {0, 0}, // Role, Base (zero, written), Stores, end
		id(3), u64(1), // Classes: one element
		id(1), str("c"), id(2), u64(1), id(3), u64(2), id(4), str("n"), {0, 0},
	}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsStorage = % x, want % x", e.buf, want)
	}
}

// TestHash_StorageOnOff verifies that the module on, even empty, differs from the
// module off, that an unlimited role differs from one with a zero base room, that
// the Label of a class and the order the lists are written in change nothing, and
// that Store of a flow target changes the hash.
//
// Requirement: FR-STO-002, FR-CFG-085
func TestHash_StorageOnOff(t *testing.T) {
	t.Parallel()
	off := Config{}
	on := Config{Modules: Modules{Storage: &Storage{}}}
	if HashConfig(off) == HashConfig(on) {
		t.Error("HashConfig of the module on and empty = that of the module off, want different")
	}
	base := storageHashConfig()
	hash := HashConfig(base)

	zero := storageHashConfig()
	zero.Modules.Storage.Roles[1].Base = &Capacity{}
	if HashConfig(zero) == hash {
		t.Error("HashConfig of a zero base room = that of an unlimited role, want different")
	}

	label := storageHashConfig()
	label.Modules.Storage.Classes[0].Label = "another"
	if HashConfig(label) != hash {
		t.Error("HashConfig changed with a StoreClass.Label, want presentation only")
	}

	shuffled := storageHashConfig()
	s := shuffled.Modules.Storage
	s.Roles[0], s.Roles[1] = s.Roles[1], s.Roles[0]
	s.Classes[0], s.Classes[1] = s.Classes[1], s.Classes[0]
	s.Roles[1].Stores = []string{"bin", "bin", "shed"}
	if HashConfig(shuffled) != hash {
		t.Error("HashConfig changed with the order of the rows, the classes or the stores, want canonical order")
	}

	moved := storageHashConfig()
	moved.Modules.Storage.Roles[1].Stores = []string{"bin"}
	moved.Modules.Storage.Roles[1].Base = &Capacity{Mass: 1}
	if HashConfig(moved) == hash {
		t.Error("HashConfig did not change with a store list, want it hashed")
	}

	flow := func(store string) Config {
		return Config{Modules: Modules{MoneySupply: &MoneySupply{Flows: []MoneyFlow{
			{ID: "f", Kind: FlowLevy, Target: FlowTarget{Kind: "stores", Store: store}, Amount: 1},
		}}}}
	}
	if HashConfig(flow("bin")) == HashConfig(flow("")) || HashConfig(flow("bin")) == HashConfig(flow("shed")) {
		t.Error("HashConfig of a stores target does not depend on FlowTarget.Store, want it hashed")
	}
}

// TestHash_StorageAfterLowerModuleIds verifies that the module's pair sits after
// those of ids 1 to 10 and before those of 13, 19 and 25 whatever the order the
// pointers were built in, so the hash id follows the Modules row.
//
// Requirement: FR-CFG-085
func TestHash_StorageAfterLowerModuleIds(t *testing.T) {
	t.Parallel()
	e := &encoder{}
	pairsModules(e, Modules{
		Demand: &Demand{}, Subsistence: &SubsistenceConfig{}, Migration: &Migration{}, Merchants: &Merchants{},
		Storage: &Storage{}, Employment: &Employment{}, Polity: &PolityConfig{}, Estates: &Estates{},
	})
	var ids []uint16
	for i := 0; i+2 <= len(e.buf); {
		n := binary.BigEndian.Uint16(e.buf[i:])
		if n == 0 {
			i += 2
			continue
		}
		ids = append(ids, n)
		// every module here is empty except the tables of Demand, Subsistence and
		// Migration, whose bodies are skipped by scanning to the next 0000.
		i += 2
		for i+2 <= len(e.buf) && binary.BigEndian.Uint16(e.buf[i:]) != 0 {
			i++
		}
	}
	want12 := -1
	for k, n := range ids {
		if n == 12 {
			want12 = k
		}
	}
	if want12 < 0 || ids[want12-1] > 12 || ids[want12+1] < 12 {
		t.Errorf("module ids in order of writing = %v, want 12 between the lower and the higher", ids)
	}
}

// TestEffectiveRoom_RowReplacesBase verifies the resolved base room of a role
// (spec-storage.md 3.3.1): the row's Base replaces the module's, a row with no Base
// makes its role unlimited even though the module has one, a role with no row has
// the module's, the module off or with no Base leaves every role unlimited, and the
// room in the role table is a copy.
//
// Requirement: FR-STO-003
func TestEffectiveRoom_RowReplacesBase(t *testing.T) {
	t.Parallel()
	m := storageHashConfig().Modules.Storage
	tests := []struct {
		name string
		m    *Storage
		role string
		want *Capacity
		row  int
	}{
		{"a row replaces the base", m, "grower", &Capacity{Mass: 9, Bulk: 8}, 0},
		{"an empty row means unlimited", m, "cook", nil, 1},
		{"no row takes the module's base", m, "maker", &Capacity{Mass: 100, Bulk: 50}, -1},
		{"no base and no row is unlimited", &Storage{}, "maker", nil, -1},
		{"the module off is unlimited", nil, "maker", nil, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, row := tt.m.baseOf(tt.role)
			if !reflect.DeepEqual(got, tt.want) || row != tt.row {
				t.Errorf("baseOf(%q) = %v, %d, want %v, %d", tt.role, got, row, tt.want, tt.row)
			}
		})
	}
}

// TestEffectiveRoom_RoleTableHoldsACopy verifies that the role table's room is the
// resolved base room, nil with the module off, and does not alias the Config the
// consumer owns.
//
// Requirement: FR-STO-003
//
// Guard: an aliased room would let a change of the consumer's Config after
// construction move the room of a role without a call.
func TestEffectiveRoom_RoleTableHoldsACopy(t *testing.T) {
	t.Parallel()
	role := func(id string) Role {
		r := DefaultRole()
		r.ID = id
		return r
	}
	c := Config{World: DefaultWorldParams(), Calendar: DefaultCalendar(), Roles: []Role{role("grower"), role("cook"), role("maker")}}
	c.Modules.Storage = storageHashConfig().Modules.Storage
	cc := c.canonical()
	tbl := newRoleTable(&cc)
	rows := &tbl
	cook, grower, maker := rows.row(0), rows.row(1), rows.row(2) // RoleIDs follow the order of the IDs
	if grower.room == nil || *grower.room != (Capacity{Mass: 9, Bulk: 8}) {
		t.Errorf("room of grower = %v, want {9 8}", grower.room)
	}
	if cook.room != nil {
		t.Errorf("room of cook = %v, want nil (an empty row is unlimited)", cook.room)
	}
	if maker.room == nil || *maker.room != (Capacity{Mass: 100, Bulk: 50}) {
		t.Errorf("room of maker = %v, want the module's base {100 50}", maker.room)
	}
	c.Modules.Storage.Base.Mass = 1
	if maker.room.Mass != 100 {
		t.Errorf("room of maker moved to %d with the consumer's Config, want a copy", maker.room.Mass)
	}
	off := Config{World: DefaultWorldParams(), Calendar: DefaultCalendar(), Roles: []Role{role("grower")}}
	cc = off.canonical()
	offTbl := newRoleTable(&cc)
	if got := offTbl.row(0).room; got != nil {
		t.Errorf("room with the module off = %v, want nil", got)
	}
}

// TestStorage_NoRegionKeyedField verifies, by reflection, that no type of the
// module has a region field or a map: the module has no region-keyed table
// (spec-storage.md 12.1.1, FR-STO-003), so a regional difference is not in the data.
//
// Requirement: FR-STO-003
//
// Guard: a region key added to the room would make the base room a function of the
// region, which spec-storage.md 3.1.2 says it is not.
func TestStorage_NoRegionKeyedField(t *testing.T) {
	t.Parallel()
	for _, ty := range []reflect.Type{reflect.TypeFor[Storage](), reflect.TypeFor[RoleRoom](), reflect.TypeFor[Capacity](), reflect.TypeFor[StoreClass]()} {
		for i := range ty.NumField() {
			f := ty.Field(i)
			if f.Type.Kind() == reflect.Map || strings.Contains(strings.ToLower(f.Name), "region") {
				t.Errorf("%s.%s is a region key or a map, want neither", ty.Name(), f.Name)
			}
		}
	}
}

// TestStorageCanonical_SharesNoRoom verifies that the canonical copy of the module
// holds its own base rooms, so a change of the consumer's Config does not reach it.
//
// Requirement: FR-STO-003
//
// Guard: a shallow copy shares the Base pointers of the module and of each row.
func TestStorageCanonical_SharesNoRoom(t *testing.T) {
	t.Parallel()
	m := &Storage{Base: &Capacity{Mass: 1, Bulk: 2}, Roles: []RoleRoom{{Role: "a", Base: &Capacity{Mass: 3, Bulk: 4}}}}
	cc := m.canonical()
	m.Base.Mass, m.Roles[0].Base.Mass = 90, 91
	if cc.Base.Mass != 1 || cc.Roles[0].Base.Mass != 3 {
		t.Errorf("canonical bases = %+v, %+v, want {1 2} and {3 4} unmoved", *cc.Base, *cc.Roles[0].Base)
	}
}

// TestStorageWarnings_RecipeLoadOverflowWarns verifies that a recipe whose output
// load does not fit in int64 is still room_below_recipe when the role also has an
// upkeep load: the sum of the two is checked, not wrapped.
//
// Requirement: FR-STO-038
//
// Guard: an unchecked upkeep + output sum wraps the overflow sentinel to a negative
// figure, which is below any room, and the warning is silently dropped.
func TestStorageWarnings_RecipeLoadOverflowWarns(t *testing.T) {
	t.Parallel()
	c := Config{
		World:       WorldParams{Upkeep: []Item{{Commodity: "u", Quantity: 1}}},
		Commodities: []Commodity{{ID: "u", Mass: 1, Bulk: 1}, {ID: "g", Mass: math.MaxInt64 / 2}},
		Recipes:     []Recipe{{ID: "r", Outputs: []Output{{Commodity: "g", Quantity: 3}}}},
		Regions:     []Region{{ID: "a"}},
		Roles:       []Role{{ID: "x", Recipes: []string{"r"}}},
		Modules:     Modules{Storage: &Storage{Base: &Capacity{Mass: MaxRoom, Bulk: MaxRoom}}},
	}
	var got bool
	for _, w := range storageWarnings(&c) {
		got = got || w.Rule == "room_below_recipe"
	}
	if !got {
		t.Errorf("storageWarnings = %v, want room_below_recipe for an output load that overflows", storageWarnings(&c))
	}
}

// TestCohortRoom verifies the room of a cohort (spec-storage.md 3.3.2, FR-STO-004):
// the weight times the base room, plus the capacity of each store of the row, a
// store not multiplied by the weight; a class that is not configured adds nothing;
// and a figure that does not fit in int64 is reported, not wrapped.
//
// Requirement: FR-STO-004
func TestCohortRoom(t *testing.T) {
	t.Parallel()
	m := &Storage{
		Roles:   []RoleRoom{{Role: "r", Stores: []string{"a", "a", "gone", "b"}}},
		Classes: []StoreClass{{ID: "a", Mass: 10, Bulk: 1}, {ID: "b", Mass: math.MaxInt64 - 1, Bulk: 0}},
	}
	tests := []struct {
		name       string
		row        int
		w          int64
		room       Capacity
		mass, bulk int64
		ok         bool
	}{
		// 3 * 5 = 15 and 3 * 2 = 6, no row, no store.
		{"no row: the weight times the base", -1, 3, Capacity{5, 2}, 15, 6, true},
		// Half of MaxInt64, plus one, times 2 does not fit.
		{"the weight times the base overflows", -1, 2, Capacity{math.MaxInt64/2 + 1, 0}, 0, 0, false},
		// Row 0 holds a, a, an unknown class and b: 5 + 10 + 10 fits, adding b's
		// MaxInt64 - 1 does not.
		{"the stores overflow the sum", 0, 1, Capacity{5, 0}, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mass, bulk, ok := cohortRoom(m, tt.row, tt.w, &tt.room)
			if mass != tt.mass || bulk != tt.bulk || ok != tt.ok {
				t.Errorf("cohortRoom(row %d, w %d, %+v) = %d, %d, %v, want %d, %d, %v", tt.row, tt.w, tt.room, mass, bulk, ok, tt.mass, tt.bulk, tt.ok)
			}
		})
	}
	// A store is not multiplied by the weight, and an unknown class adds nothing:
	// base 5 * 3 = 15, plus a (10) twice = 35 mass; bulk 0 * 3 + 1 + 1 = 2.
	small := &Storage{Roles: []RoleRoom{{Role: "r", Stores: []string{"a", "gone", "a"}}}, Classes: m.Classes[:1]}
	if mass, bulk, ok := cohortRoom(small, 0, 3, &Capacity{Mass: 5}); mass != 35 || bulk != 2 || !ok {
		t.Errorf("cohortRoom(a, gone, a at weight 3) = %d, %d, %v, want 35, 2, true", mass, bulk, ok)
	}
}

// TestRoomLoad_OverflowIsReported verifies that the load of a list of items whose sum
// does not fit in int64 is reported as not ok, never wrapped, and that an undeclared
// commodity adds nothing.
//
// Requirement: FR-STO-038
//
// Guard: a wrapped load is a small or negative figure that passes a room check the
// real load fails.
func TestRoomLoad_OverflowIsReported(t *testing.T) {
	t.Parallel()
	c := Config{Commodities: []Commodity{{ID: "big", Mass: math.MaxInt64 / 2, Bulk: 1}, {ID: "one", Mass: 1, Bulk: 1}}}
	lk := newLookup(&c)
	tests := []struct {
		name       string
		items      []Item
		mass, bulk int64
		ok         bool
	}{
		{"an undeclared commodity adds nothing", []Item{{Commodity: "none", Quantity: 5}, {Commodity: "one", Quantity: 2}}, 2, 2, true},
		{"one product overflows", []Item{{Commodity: "big", Quantity: 3}}, 0, 0, false},
		{"the sum overflows", []Item{{Commodity: "big", Quantity: 1}, {Commodity: "big", Quantity: 1}, {Commodity: "one", Quantity: 2}}, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mass, bulk, ok := roomLoad(lk, tt.items)
			if mass != tt.mass || bulk != tt.bulk || ok != tt.ok {
				t.Errorf("roomLoad(%v) = %d, %d, %v, want %d, %d, %v", tt.items, mass, bulk, ok, tt.mass, tt.bulk, tt.ok)
			}
		})
	}
}

// TestStorageWarnings_UpkeepOverflowWarnsAndNeverFails verifies that an upkeep whose
// load does not fit in int64 reads as the largest figure: the room is below its
// cover, the warning is given, and validation of the room reports ErrRoomShape rather
// than accepting a wrapped load.
//
// Requirement: FR-STO-038
func TestStorageWarnings_UpkeepOverflowWarnsAndNeverFails(t *testing.T) {
	t.Parallel()
	c := Config{
		World:       WorldParams{Upkeep: []Item{{Commodity: "u", Quantity: 3}}, CoverTicks: 2},
		Commodities: []Commodity{{ID: "u", Mass: math.MaxInt64 / 2, Bulk: 1}},
		Regions:     []Region{{ID: "a"}},
		Roles:       []Role{{ID: "x"}},
		Modules:     Modules{Storage: &Storage{Base: &Capacity{Mass: MaxRoom, Bulk: MaxRoom}}},
	}
	got := map[string]bool{}
	for _, w := range storageWarnings(&c) {
		got[w.Rule] = true
	}
	if !got["room_below_cover"] {
		t.Errorf("storageWarnings = %v, want room_below_cover for an upkeep load that overflows", got)
	}
	if mass, bulk := regionsUpkeepLoad(&c, newLookup(&c), &c.Roles[0]); mass != math.MaxInt64 || bulk != math.MaxInt64 {
		t.Errorf("regionsUpkeepLoad = %d, %d, want the largest int64 for both", mass, bulk)
	}
}

// TestStorageWarnings_RecipeRoomOverflowGivesNone verifies that room_below_recipe is
// not reported for a role whose own room does not fit in int64: that case is not a
// recipe finding, and the sum is never wrapped.
//
// Requirement: FR-STO-038
func TestStorageWarnings_RecipeRoomOverflowGivesNone(t *testing.T) {
	t.Parallel()
	c := Config{
		Commodities: []Commodity{{ID: "g", Mass: 1, Bulk: 1}},
		Recipes:     []Recipe{{ID: "r", Outputs: []Output{{Commodity: "g", Quantity: 1}}}},
		Roles:       []Role{{ID: "x", Recipes: []string{"r", "undeclared"}}},
	}
	m := &Storage{
		Roles:   []RoleRoom{{Role: "x", Stores: []string{"k", "k"}}},
		Classes: []StoreClass{{ID: "k", Mass: math.MaxInt64/2 + 1}},
	}
	lk := newLookup(&c)
	if got := recipeWarnings(lk, m, &c.Roles[0], 0, &Capacity{Mass: 1, Bulk: 1}, nil, 0, 0); got != nil {
		t.Errorf("recipeWarnings with a room that overflows = %v, want none", got)
	}
	// The same role with a room that fits and holds nothing: the declared recipe warns,
	// the undeclared one is skipped.
	fits := &Storage{Roles: []RoleRoom{{Role: "x"}}}
	got := recipeWarnings(lk, fits, &c.Roles[0], 0, &Capacity{}, []string{"p"}, 0, 0)
	if len(got) != 1 || got[0].Rule != "room_below_recipe" || got[0].Path[len(got[0].Path)-1] != "r" {
		t.Errorf("recipeWarnings = %v, want one room_below_recipe for r", got)
	}
}
