package economysim

import (
	"encoding/hex"
	"slices"
	"testing"
)

// specExampleBytes is the 350-byte encoding of the five-entity configuration of
// spec-config.md 8.2.4, copied from the spec.
const specExampleBytes = "" +
	"010003000000000000000100010000000000000005677261696e000200000000" +
	"0000000100030000000000000001000800000000000000640000000400000000" +
	"000000010001000000000000000467726f770002000000000000000100040000" +
	"00000000000100010000000000000005677261696e0002000000000000000100" +
	"0300000000000027100000000000060000000000000001000100000000000000" +
	"066661726d657200020000000000000001000000000000000467726f77000500" +
	"000000000027100007000000000000000764656661756c740009000000000000" +
	"3a98000b0000000000002710000d00000000000000fa000e0000000000000fa0" +
	"000000070000000000000001000100000000000000056e6f7274680002000000" +
	"00000000640000000a0000000000000001000100000000000000056e6f727468" +
	"000200000000000000066661726d65720003000000000000000a00000000"

// TestEncodeConfig_SpecExampleBytes verifies the bytes, not only their digest,
// of the five-entity worked example of spec-config.md 8.2.4: 350 bytes, the
// version byte first and the configuration's terminator last.
//
// Requirement: FR-CFG-104, FR-CFG-105
func TestEncodeConfig_SpecExampleBytes(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Commodities: []Commodity{{ID: "grain", Mass: 1, Bulk: 1, ReferencePrice: 100}},
		Recipes: []Recipe{{
			ID: "grow", Duration: 1,
			Outputs: []Output{{Commodity: "grain", Quantity: 1, CostShare: 10_000}},
		}},
		Roles: []Role{{
			ID: "farmer", Recipes: []string{"grow"}, StartMoney: 10_000, Strategy: "default",
			TargetRatio: 15_000, DefaultRatio: 10_000, ExpectedWorkTicks: 250, MinWorkShare: 4_000,
		}},
		Regions: []Region{{ID: "north", ReferenceBasket: 100}},
		Groups:  []Group{{Region: "north", Role: "farmer", Households: 10}},
	}
	want, err := hex.DecodeString(specExampleBytes)
	if err != nil || len(want) != 350 {
		t.Fatalf("spec example = %d bytes, error %v, want 350 bytes", len(want), err)
	}
	if got := encodeConfig(cfg.canonical()); !slices.Equal(got, want) {
		t.Errorf("encodeConfig(five entities) = %x, want %x", got, want)
	}
}

// TestConfig_TypedIDsFollowIDOrder verifies that the typed IDs are the indices
// of the entities sorted by ID string, whatever order they were written in, that
// an unknown ID is not found, and that canonicalisation leaves the receiver
// as it was.
//
// Requirement: FR-CFG-005
func TestConfig_TypedIDsFollowIDOrder(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Commodities: []Commodity{{ID: "wood"}, {ID: "ale"}, {ID: "iron"}},
		Recipes:     []Recipe{{ID: "smelt"}, {ID: "brew"}},
		Roles:       []Role{{ID: "smith"}, {ID: "brewer"}, {ID: "cook"}},
		Regions:     []Region{{ID: "south"}, {ID: "north"}},
	}
	c := cfg.canonical()
	if cfg.Commodities[0].ID != "wood" || cfg.Roles[0].ID != "smith" || cfg.Regions[0].ID != "south" {
		t.Errorf("canonical() changed the order of its receiver: %+v", cfg)
	}
	tests := []struct {
		name   string
		got    int
		ok     bool
		want   int
		wantOK bool
	}{
		{"commodity ale", int(first(c.commodityID("ale"))), second(c.commodityID("ale")), 0, true},
		{"commodity iron", int(first(c.commodityID("iron"))), second(c.commodityID("iron")), 1, true},
		{"commodity wood", int(first(c.commodityID("wood"))), second(c.commodityID("wood")), 2, true},
		{"commodity unknown", 0, second(c.commodityID("gold")), 0, false},
		{"recipe brew", int(first(c.recipeID("brew"))), second(c.recipeID("brew")), 0, true},
		{"recipe smelt", int(first(c.recipeID("smelt"))), second(c.recipeID("smelt")), 1, true},
		{"role brewer", int(first(c.roleID("brewer"))), second(c.roleID("brewer")), 0, true},
		{"role smith", int(first(c.roleID("smith"))), second(c.roleID("smith")), 2, true},
		{"role unknown", 0, second(c.roleID("king")), 0, false},
		{"region north", int(first(c.regionID("north"))), second(c.regionID("north")), 0, true},
		{"region south", int(first(c.regionID("south"))), second(c.regionID("south")), 1, true},
		{"region unknown", 0, second(c.regionID("east")), 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.ok != tt.wantOK || (tt.ok && tt.got != tt.want) {
				t.Errorf("%s = %d, %v, want %d, %v", tt.name, tt.got, tt.ok, tt.want, tt.wantOK)
			}
		})
	}
}

// first returns the first of two values.
func first[A, B any](a A, _ B) A { return a }

// second returns the second of two values.
func second[A, B any](_ A, b B) B { return b }

// TestConfigCanonical_SharesNothing verifies that the canonical copy shares no
// slice with the original: changing the copy leaves the original alone, and
// the other way round.
//
// Guard: a world keeps the canonical copy while the caller keeps editing the
// original (spec-config.md 3.5); a shared backing array would let a later edit
// change a running world and its hash.
func TestConfigCanonical_SharesNothing(t *testing.T) {
	t.Parallel()
	cfg := Config{
		World:       World{Upkeep: []Item{{Commodity: "b", Quantity: 1}, {Commodity: "a", Quantity: 2}}},
		Calendar:    Calendar{Months: []Month{{ID: "m02", Days: 28}, {ID: "m01", Days: 31}}},
		Commodities: []Commodity{{ID: "b", Classes: []string{"y", "x"}}, {ID: "a"}},
		Recipes:     []Recipe{{ID: "r", Inputs: []Item{{Commodity: "b"}, {Commodity: "a"}}}},
		WarmStart:   &WarmStart{Groups: []WarmGroup{{Region: "n", Role: "r", Stock: []Item{{Commodity: "b"}, {Commodity: "a"}}}}},
	}
	c := cfg.canonical()
	if got := c.Calendar.Months[0].ID; got != "m02" {
		t.Errorf("canonical().Calendar.Months[0].ID = %q, want the written order m02", got)
	}
	if got := c.World.Upkeep[0].Commodity; got != "a" {
		t.Errorf("canonical().World.Upkeep[0].Commodity = %q, want a", got)
	}
	c.World.Upkeep[0].Quantity = 99
	c.Commodities[1].Classes[0] = "changed"
	c.Calendar.Months[0].Days = 99
	c.Recipes[0].Inputs[0].Quantity = 99
	c.WarmStart.Groups[0].Stock[0].Quantity = 99
	c.WarmStart.Wages = append(c.WarmStart.Wages, SeedWage{})
	if cfg.World.Upkeep[0].Quantity != 1 || cfg.Commodities[0].Classes[0] != "y" ||
		cfg.Calendar.Months[0].Days != 28 || cfg.Recipes[0].Inputs[0].Quantity != 0 ||
		cfg.WarmStart.Groups[0].Stock[0].Quantity != 0 || cfg.WarmStart.Wages != nil {
		t.Errorf("changing the canonical copy changed the original: %+v", cfg)
	}
}
