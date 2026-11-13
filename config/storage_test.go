package config_test

import (
	"reflect"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// storageDoc is the file form of spec-storage.md 12.5: a base room, a role row with
// its own base and stores, a role row with no base (an unlimited role) and two
// classes, one with a label.
const storageDoc = minimal + `
[modules.storage.base]
mass = 400
bulk = 200

[modules.storage.role.grower]
stores = ["bin", "bin"]
[modules.storage.role.grower.base]
mass = 600
bulk = 300

[modules.storage.role.cook]

[modules.storage.class.bin]
mass = 400
bulk = 300
label = "a grain bin"

[modules.storage.class.shed]
mass = 50
bulk = 50
`

// storageSame is storageDoc written the other way: the classes first, the tables in
// another order and the base of the row as a dotted key.
const storageSame = minimal + `
[modules.storage.class.shed]
bulk = 50
mass = 50
[modules.storage.class.bin]
label = "a grain bin"
bulk = 300
mass = 400
[modules.storage.role.cook]
[modules.storage.role.grower]
stores = ["bin", "bin"]
base = { mass = 600, bulk = 300 }
[modules.storage.base]
bulk = 200
mass = 400
`

// TestLoad_Storage verifies that the tables of spec-storage.md 12.5 load: the base
// room, the role rows in role order with their bases and stores, a row with no base
// table as an unlimited role, the classes in ID order with their label, a key-less
// [modules.storage] table switching the module on, an empty [base] table as a base
// room of 0, no table leaving the module off, an unknown key or a wrong type
// rejected, two spellings of one module loading to one Config and one hash, and the
// file and the Go value hashing alike.
//
// Requirement: FR-STO-002, FR-CFG-085
func TestLoad_Storage(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(storageDoc)
	if err != nil {
		t.Fatalf("Parse(storageDoc) error = %v, want nil", err)
	}
	want := &economysim.Storage{
		Base: &economysim.Capacity{Mass: 400, Bulk: 200},
		Roles: []economysim.RoleRoom{
			{Role: "cook"},
			{Role: "grower", Base: &economysim.Capacity{Mass: 600, Bulk: 300}, Stores: []string{"bin", "bin"}},
		},
		Classes: []economysim.StoreClass{
			{ID: "bin", Mass: 400, Bulk: 300, Label: "a grain bin"},
			{ID: "shed", Mass: 50, Bulk: 50},
		},
	}
	if !reflect.DeepEqual(got.Modules.Storage, want) {
		t.Errorf("Parse(storageDoc).Modules.Storage = %+v, want %+v", got.Modules.Storage, want)
	}
	if same, err := parseStrings(storageSame); err != nil || !reflect.DeepEqual(same, got) ||
		economysim.HashConfig(same) != economysim.HashConfig(got) {
		t.Errorf("Parse(storageSame) = %+v, %v, want the same Config and hash as storageDoc", same, err)
	}
	// The file and the Go value hash alike (the label is not hashed either way).
	goCfg := economysim.Config{World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar()}
	goCfg.Modules.Storage = want
	if economysim.HashConfig(goCfg) != economysim.HashConfig(got) {
		t.Error("HashConfig of the file differs from that of the same Go value")
	}
}

// TestLoad_StorageOnAndOff verifies that a key-less [modules.storage] table switches
// the module on, an empty [base] table is a base room of 0 and not an unlimited one,
// and no table leaves the module off.
//
// Requirement: FR-STO-002, FR-CFG-085
func TestLoad_StorageOnAndOff(t *testing.T) {
	t.Parallel()
	if on, err := parseStrings(minimal + "[modules.storage]\n"); err != nil || !reflect.DeepEqual(on.Modules.Storage, &economysim.Storage{}) {
		t.Errorf("Parse(empty table).Modules.Storage = %+v, %v, want the module on and empty", on.Modules.Storage, err)
	}
	if zero, err := parseStrings(minimal + "[modules.storage.base]\n"); err != nil || !reflect.DeepEqual(zero.Modules.Storage.Base, &economysim.Capacity{}) {
		t.Errorf("Parse(empty base).Modules.Storage.Base = %+v, %v, want a base room of 0, not unlimited", zero.Modules.Storage, err)
	}
	if off, _ := parseStrings(minimal); off.Modules.Storage != nil {
		t.Errorf("Parse(minimal).Modules.Storage = %+v, want nil: the module is off", off.Modules.Storage)
	}
}

// TestLoad_StorageRejections verifies the keys and types the loader rejects in the
// storage tables, each with the rule and the path of the key.
//
// Requirement: FR-CFG-085
func TestLoad_StorageRejections(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"an unknown key of the module", minimal + "[modules.storage]\nlevel = 1\n", economysim.ErrUnknownKey, []string{"modules", "storage", "level"}},
		{"an unknown key of a class", minimal + "[modules.storage.class.a]\ncost = 1\n", economysim.ErrUnknownKey, []string{"modules", "storage", "class", "a", "cost"}},
		{"an unknown key of a base", minimal + "[modules.storage.base]\nvolume = 1\n", economysim.ErrUnknownKey, []string{"modules", "storage", "base", "volume"}},
		{"a region key of a row", minimal + "[modules.storage.role.a]\nregion = \"r\"\n", economysim.ErrUnknownKey, []string{"modules", "storage", "role", "a", "region"}},
		{"a float mass", minimal + "[modules.storage.class.a]\nmass = 1.5\n", economysim.ErrNotInteger, []string{"modules", "storage", "class", "a", "mass"}},
		{"a string bulk of a base", minimal + "[modules.storage.base]\nbulk = \"1\"\n", economysim.ErrWrongType, []string{"modules", "storage", "base", "bulk"}},
		{"a number in stores", minimal + "[modules.storage.role.a]\nstores = [ 1 ]\n", economysim.ErrWrongType, []string{"modules", "storage", "role", "a", "stores", "[0]"}},
		{"a scalar where a class table", minimal + "[modules.storage.class]\na = 1\n", economysim.ErrWrongType, []string{"modules", "storage", "class", "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseStrings(tt.doc)
			if err == nil || !hasError(err, tt.rule, tt.path) {
				t.Errorf("Parse(%q) errors = %s, want %v at %q", tt.doc, describe(err), tt.rule, tt.path)
			}
		})
	}
}

// TestLoad_StoresTarget verifies the key store of a flow target: it loads into
// FlowTarget.Store and changes the hash, and an unknown key of a target is still
// rejected.
//
// Requirement: FR-STO-002, FR-CFG-085
func TestLoad_StoresTarget(t *testing.T) {
	t.Parallel()
	doc := func(store string) string {
		return minimal + "[modules.money_supply.flow.rent]\nkind = \"levy\"\namount = 1\ntarget = { kind = \"stores\", store = " + strconv.Quote(store) + " }\n"
	}
	got, err := parseStrings(doc("bin"))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if tg := got.Modules.MoneySupply.Flows[0].Target; tg != (economysim.FlowTarget{Kind: "stores", Store: "bin"}) {
		t.Errorf("Parse().Target = %+v, want the stores of bin", tg)
	}
	other, _ := parseStrings(doc("shed"))
	if economysim.HashConfig(got) == economysim.HashConfig(other) {
		t.Error("HashConfig does not depend on the store of a target, want it hashed")
	}
}

// FuzzLoadStorage verifies, for random class, row and base values, that the loader
// never panics, that an accepted document loads again to an equal Config with an
// equal hash and that its Validate does not panic.
//
// Requirement: FR-CFG-024
func FuzzLoadStorage(f *testing.F) {
	f.Add("bin", int64(400), int64(300), "bin", int64(10), int64(10))
	f.Add("a", int64(0), int64(0), "b", int64(-1), int64(1_000_000_000_001))
	f.Add("", int64(1), int64(1), "", int64(0), int64(0))
	f.Fuzz(func(t *testing.T, class string, mass, bulk int64, store string, baseMass, baseBulk int64) {
		doc := minimal + "[modules.storage.class." + strconv.Quote(class) + "]\nmass = " + strconv.FormatInt(mass, 10) + "\nbulk = " + strconv.FormatInt(bulk, 10) +
			"\n[modules.storage.role.a]\nstores = [" + strconv.Quote(store) + "]\n[modules.storage.role.a.base]\nmass = " + strconv.FormatInt(baseMass, 10) +
			"\nbulk = " + strconv.FormatInt(baseBulk, 10) + "\n"
		got, err := parseStrings(doc)
		if err != nil {
			return
		}
		again, err := parseStrings(doc)
		if err != nil || !reflect.DeepEqual(got, again) || economysim.HashConfig(got) != economysim.HashConfig(again) {
			t.Fatalf("Parse(%q) is not repeatable: %v", doc, err)
		}
		_ = got.Validate()
	})
}
