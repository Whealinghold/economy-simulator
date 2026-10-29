package economysim

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"slices"
	"testing"
)

// TestHash_SubsistenceOnOff verifies that a subsistence module that is off
// leaves the hash of the configuration as it was, that a module that is on and
// empty differs from it (a non-nil pointer is always written), and that a
// default list and a role row each change the hash (spec-config.md 8.2.1).
//
// Requirement: FR-CFG-077, FR-SUB-001
func TestHash_SubsistenceOnOff(t *testing.T) {
	t.Parallel()
	off, empty, listed, rowed := oneRegion(), oneRegion(), oneRegion(), oneRegion()
	empty.Modules.Subsistence = &SubsistenceConfig{}
	listed.Modules.Subsistence = &SubsistenceConfig{Recipes: []string{"a"}}
	rowed.Modules.Subsistence = &SubsistenceConfig{Roles: []RoleOwnUse{{Role: "r"}}}
	for name, c := range map[string]Config{"empty": empty, "listed": listed, "rowed": rowed} {
		if HashConfig(c) == HashConfig(off) {
			t.Errorf("HashConfig(%s) equals the hash of the module off, want different", name)
		}
	}
	if HashConfig(empty) == HashConfig(listed) || HashConfig(listed) == HashConfig(rowed) || HashConfig(empty) == HashConfig(rowed) {
		t.Errorf("HashConfig of an empty module, a default list and an empty row are not all different")
	}
}

// TestHash_SubsistenceFieldIDs verifies the bytes of the module in the hash
// (spec-config.md 8.2.2): the one-region row with the module on and empty is
// `000c 0008 0000 0000` before the last terminator, and a populated module writes
// Recipes 1, Roles 2 (Role 1, Recipes 2), EnterTicks 3, ExitTicks 4 and
// RetryJobs 5 as the byte string below. Both digests were computed by an
// independent encoder (Python's hashlib over the bytes of the one-region row of
// spec-config.md 8.2.4 with the module's pair inserted).
//
// Requirement: FR-CFG-077, FR-CFG-025
func TestHash_SubsistenceFieldIDs(t *testing.T) {
	t.Parallel()
	const base = "0100070000000000000001000100000000000000056e6f7274680000" + "0000"
	tests := []struct {
		name   string
		module *SubsistenceConfig
		insert string
		digest string
	}{
		{"on and empty", &SubsistenceConfig{}, "000c" + "0008" + "0000" + "0000",
			"232ca24344a8fbc9b5361abf1bc088f932196d50f336157641b290300380af86"},
		{"populated", &SubsistenceConfig{
			Recipes: []string{"a"}, Roles: []RoleOwnUse{{Role: "r", Recipes: []string{"b"}}},
			EnterTicks: 2, ExitTicks: 3, RetryJobs: 4,
		}, "000c" + "0008" +
			"0001" + "0000000000000001" + "0000000000000001" + "61" + // Recipes: one string "a"
			"0002" + "0000000000000001" + // Roles: one row
			"0001" + "0000000000000001" + "72" + // Role "r"
			"0002" + "0000000000000001" + "0000000000000001" + "62" + // Recipes: "b"
			"0000" + // the row's terminator
			"0003" + "0000000000000002" + "0004" + "0000000000000003" + "0005" + "0000000000000004" +
			"0000" + "0000",
			"da26b4e520d4b42644b0f48b2166b524a71c459b137758f342b044daf7271925"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := oneRegion()
			cfg.Modules.Subsistence = tt.module
			want := base[:len(base)-4] + tt.insert + base[len(base)-4:]
			got := encodeConfig(cfg.canonical())
			if hex.EncodeToString(got) != want {
				t.Errorf("encodeConfig(one region, Subsistence %s) = %x, want %s", tt.name, got, want)
			}
			sum := sha256.Sum256(got)
			if hex.EncodeToString(sum[:]) != tt.digest || HashConfig(cfg) != sum {
				t.Errorf("HashConfig(one region, Subsistence %s) = %x, want %s", tt.name, HashConfig(cfg), tt.digest)
			}
			r := &idReader{b: got[1:]}
			if err := r.structOf(reflect.TypeFor[Config](), "Config"); err != nil || len(r.b) != 0 {
				t.Errorf("decode of the encoding = %v with %d bytes left, want nil and 0", err, len(r.b))
			}
		})
	}
}

// TestHash_SubsistenceAfterLowerModuleIDs verifies that the pair of the
// subsistence module (id 8) is written after the pairs of the modules with a
// smaller id and before that of the estates module (25), whatever the order in
// which the modules were built.
//
// Requirement: FR-CFG-077, FR-CFG-104
func TestHash_SubsistenceAfterLowerModuleIDs(t *testing.T) {
	t.Parallel()
	c := oneRegion()
	c.Modules.Demand = &Demand{}
	c.Modules.MoneySupply = &MoneySupply{}
	c.Modules.Resources = &Resources{}
	c.Modules.Subsistence = &SubsistenceConfig{}
	c.Modules.Estates = &Estates{}
	r := &idReader{b: encodeConfig(c.canonical())[1:]}
	if err := r.structOf(reflect.TypeFor[Config](), "Config"); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if want := []uint16{1, 2, 5, 8, 25}; !slices.Equal(r.modules, want) {
		t.Errorf("Modules wrote the ids %v, want %v", r.modules, want)
	}
}

// TestHash_ListOrderIrrelevant verifies that two spellings of one subsistence
// configuration, with the default list, the rows and each row's list in another
// order, hash alike, and that a different list does not.
//
// Requirement: FR-CFG-077, FR-SUB-002
func TestHash_ListOrderIrrelevant(t *testing.T) {
	t.Parallel()
	mk := func(recipes []string, rows ...RoleOwnUse) Config {
		c := oneRegion()
		c.Modules.Subsistence = &SubsistenceConfig{Recipes: recipes, Roles: rows, EnterTicks: 2, ExitTicks: 3}
		return c
	}
	a := mk([]string{"x", "y", "z"}, RoleOwnUse{Role: "p", Recipes: []string{"m", "n"}}, RoleOwnUse{Role: "q"})
	b := mk([]string{"z", "x", "y"}, RoleOwnUse{Role: "q"}, RoleOwnUse{Role: "p", Recipes: []string{"n", "m"}})
	other := mk([]string{"x", "y"}, RoleOwnUse{Role: "p", Recipes: []string{"m", "n"}}, RoleOwnUse{Role: "q"})
	if HashConfig(a) != HashConfig(b) {
		t.Errorf("HashConfig differs between two orders of one configuration")
	}
	if HashConfig(a) == HashConfig(other) {
		t.Errorf("HashConfig equal for two different default lists")
	}
	before := slices.Clone(b.Modules.Subsistence.Recipes)
	_ = HashConfig(b)
	if !slices.Equal(before, b.Modules.Subsistence.Recipes) {
		t.Errorf("HashConfig changed the receiver's list: %v, want %v", b.Modules.Subsistence.Recipes, before)
	}
}
