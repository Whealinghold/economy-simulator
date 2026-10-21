package economysim

import (
	"math"
	"slices"
	"strings"
	"testing"
)

// roleConfig is a small canonical configuration for the role table.
// Commodities sort to ash 0, fuel 1, gem 2, ingot 3, oven 4, ore 5, staple 6.
// Roles sort to idle 0, smith 1. Regions sort to north 0, south 1.
func roleConfig() *Config {
	c := Config{
		World: WorldParams{WelfareWindow: 30, WindowCycles: 2, Upkeep: []Item{{Commodity: "staple", Quantity: 3}, {Commodity: "fuel", Quantity: 1}}},
		Commodities: []Commodity{{ID: "staple"}, {ID: "ore"}, {ID: "oven"}, {ID: "ingot"},
			{ID: "gem"}, {ID: "fuel"}, {ID: "ash"}},
		Recipes: []Recipe{{ID: "smelt", Duration: 1,
			Inputs: []Item{{Commodity: "ore", Quantity: 2}},
			// ash is a discard: it must not join the set.
			Outputs:  []Output{{Commodity: "ingot", Quantity: 1, CostShare: 10000}, {Commodity: "ash", Quantity: 1, Discard: true}},
			Requires: []Tool{{Commodity: "oven", WearPerUse: 1}}}},
		Roles: []Role{
			// smith's own upkeep replaces the world's.
			{ID: "smith", Recipes: []string{"smelt"}, Trades: []string{"gem"}, Upkeep: []Item{{Commodity: "staple", Quantity: 2}}},
			{ID: "idle"},
		},
		Regions: []Region{
			{ID: "south"},
			{ID: "north", ExtraUpkeep: []Item{{Commodity: "fuel", Quantity: 1}}},
		},
	}
	cc := c.canonical()
	return &cc
}

// TestNewRoleTable_TradableSetAndUpkeep verifies a hand-computed table: the
// own upkeep replaces the world's, a region's extra adds to it, the inputs,
// tool and non-discard output of the recipes and a Trades entry join the set,
// the discard does not, and the set is ascending with no duplicate.
//
// Requirement: FR-AGT-003, FR-AGT-011
func TestNewRoleTable_TradableSetAndUpkeep(t *testing.T) {
	t.Parallel()
	tab := newRoleTable(roleConfig())
	smith, idle := tab.row(1), tab.row(0)

	// smith: staple 6 (upkeep), fuel 1 (north's extra, any region), ore 5 and
	// oven 4 (input, tool), ingot 3 (output), gem 2 (trade); not ash 0.
	if want := []CommodityID{1, 2, 3, 4, 5, 6}; !slices.Equal(smith.tradable, want) {
		t.Errorf("smith tradable = %v, want %v", smith.tradable, want)
	}
	if smith.isTradable(0) || !smith.isTradable(3) {
		t.Errorf("smith isTradable(ash)=%v isTradable(ingot)=%v, want false, true", smith.isTradable(0), smith.isTradable(3))
	}
	if want := []itemRow{{6, 2}}; !slices.Equal(smith.upkeep, want) {
		t.Errorf("smith upkeep = %v, want %v", smith.upkeep, want)
	}
	// north: base 2 staple plus 1 fuel.
	if got, want := smith.upkeepIn(0), []itemRow{{1, 1}, {6, 2}}; !slices.Equal(got, want) {
		t.Errorf("smith upkeepIn(north) = %v, want %v", got, want)
	}
	// south has no extra: the base list.
	if got, want := smith.upkeepIn(1), smith.upkeep; !slices.Equal(got, want) {
		t.Errorf("smith upkeepIn(south) = %v, want %v", got, want)
	}
	// idle: world's 3 staple and 1 fuel; north adds 1 fuel, so fuel is 2.
	if got, want := idle.upkeepIn(0), []itemRow{{1, 2}, {6, 3}}; !slices.Equal(got, want) {
		t.Errorf("idle upkeepIn(north) = %v, want %v", got, want)
	}
	if got, want := idle.upkeepIn(1), []itemRow{{1, 1}, {6, 3}}; !slices.Equal(got, want) {
		t.Errorf("idle upkeepIn(south) = %v, want %v", got, want)
	}
	if len(idle.regionUpkeep) != 1 {
		t.Errorf("idle regionUpkeep rows = %d, want 1 (only north has an extra)", len(idle.regionUpkeep))
	}
}

// TestNewRoleTable_NoRecipes verifies that a role with no recipes has an empty
// recipe list and still has its upkeep and trades in its set.
//
// Requirement: FR-AGT-010
func TestNewRoleTable_NoRecipes(t *testing.T) {
	t.Parallel()
	tab := newRoleTable(roleConfig())
	idle := tab.row(0)
	if len(idle.recipes) != 0 {
		t.Errorf("idle recipes = %v, want empty", idle.recipes)
	}
	if want := []CommodityID{1, 6}; !slices.Equal(idle.tradable, want) {
		t.Errorf("idle tradable = %v, want %v (fuel, staple)", idle.tradable, want)
	}
}

// TestRoleTable_TradableSomewhere verifies the union flag: every commodity in
// some set is flagged, and a discard-only commodity is not.
//
// Requirement: FR-GDS-002
func TestRoleTable_TradableSomewhere(t *testing.T) {
	t.Parallel()
	tab := newRoleTable(roleConfig())
	for id, want := range []bool{false, true, true, true, true, true, true} {
		if got := tab.tradableSomewhere(CommodityID(id)); got != want {
			t.Errorf("tradableSomewhere(%d) = %v, want %v", id, got, want)
		}
	}
	if tab.tradableSomewhere(99) {
		t.Errorf("tradableSomewhere(99) = true, want false for an ID out of range")
	}
}

// TestNewRoleTable_OverflowPanics verifies that a merged upkeep that
// overflows is not wrapped: built directly from a configuration Validate
// would reject, the constructor panics with the invariant message.
//
// Requirement: FR-AGT-011
func TestNewRoleTable_OverflowPanics(t *testing.T) {
	t.Parallel()
	c := roleConfig()
	c.Roles[1].Upkeep = []Item{{Commodity: "staple", Quantity: Quantity(math.MaxInt64)}}
	c.Regions[0].ExtraUpkeep = []Item{{Commodity: "staple", Quantity: 1}}
	defer func() {
		r := recover()
		msg, _ := r.(string)
		if !strings.Contains(msg, "invariant") {
			t.Errorf("newRoleTable panic = %v, want a message containing \"invariant\"", r)
		}
	}()
	newRoleTable(c)
	t.Errorf("newRoleTable returned, want a panic on overflow")
}

// TestRoleTable_ReadsAllocateNothing verifies that the reads of the table
// allocate nothing.
//
// Requirement: NFR-GDS-001
func TestRoleTable_ReadsAllocateNothing(t *testing.T) {
	tab := newRoleTable(roleConfig())
	var sink int
	got := testing.AllocsPerRun(100, func() {
		r := tab.row(1)
		sink += len(r.upkeepIn(0)) + len(r.upkeepIn(1))
		if r.isTradable(3) && tab.tradableSomewhere(3) {
			sink++
		}
	})
	if got != 0 {
		t.Errorf("role table reads allocated %v per run, want 0 (sink %d)", got, sink)
	}
}

// TestNewRoleTable_UnknownCommodityPanics verifies that a role naming a
// commodity that does not resolve panics with the invariant message instead
// of building a row with a wrong ID.
//
// Guard: the table builders take a validated configuration; a silent zero ID
// would make a role trade the first commodity.
func TestNewRoleTable_UnknownCommodityPanics(t *testing.T) {
	t.Parallel()
	c := roleConfig()
	c.Roles[1].Trades = []string{"nonesuch"}
	defer func() {
		r := recover()
		msg, _ := r.(string)
		if !strings.Contains(msg, "invariant") {
			t.Errorf("newRoleTable panic = %v, want a message containing \"invariant\"", r)
		}
	}()
	newRoleTable(c)
	t.Errorf("newRoleTable returned, want a panic on an unknown commodity")
}

// TestRoleRow_UpkeepInSearchesEveryRegion verifies the lookup of a region's
// upkeep when several regions have an extra: a region before the first extra,
// between two, after the last and each with an extra gets the right list.
//
// Requirement: FR-AGT-011
func TestRoleRow_UpkeepInSearchesEveryRegion(t *testing.T) {
	t.Parallel()
	c := Config{
		World:       WorldParams{WelfareWindow: 30, WindowCycles: 2},
		Commodities: []Commodity{{ID: "fuel"}, {ID: "staple"}},
		Roles:       []Role{{ID: "worker", Upkeep: []Item{{Commodity: "staple", Quantity: 1}}}},
	}
	// Regions r0..r6; r1, r3 and r5 carry an extra of 1, 3 and 5 fuel.
	for i := range 7 {
		r := Region{ID: "r" + string(rune('0'+i))}
		if i%2 == 1 {
			r.ExtraUpkeep = []Item{{Commodity: "fuel", Quantity: Quantity(i)}}
		}
		c.Regions = append(c.Regions, r)
	}
	cc := c.canonical()
	tab := newRoleTable(&cc)
	row := tab.row(0)
	if len(row.regionUpkeep) != 3 {
		t.Fatalf("regionUpkeep rows = %d, want 3", len(row.regionUpkeep))
	}
	// Commodity 0 (fuel) is in the set through the regions' extras: ID 0 is a
	// valid commodity, not "none".
	if !row.isTradable(0) || !tab.tradableSomewhere(0) {
		t.Errorf("isTradable(0) = %v, tradableSomewhere(0) = %v, want true, true", row.isTradable(0), tab.tradableSomewhere(0))
	}
	base := []itemRow{{1, 1}}
	for g := range 7 {
		want := base
		if g%2 == 1 {
			// fuel is commodity 0, staple 1: the merged list is fuel then staple.
			want = []itemRow{{0, Quantity(g)}, {1, 1}}
		}
		if got := row.upkeepIn(RegionID(g)); !slices.Equal(got, want) {
			t.Errorf("upkeepIn(%d) = %v, want %v", g, got, want)
		}
	}
	// A region past every row, and a negative one, get the base list.
	for _, g := range []RegionID{7, 100, -1} {
		if got := row.upkeepIn(g); !slices.Equal(got, base) {
			t.Errorf("upkeepIn(%d) = %v, want the base %v", g, got, base)
		}
	}
}

// TestRoleTable_OutOfRangeCommodity verifies that a commodity ID below 0, equal
// to the number of commodities and far above it is not tradable, for a role
// and for the union, and does not index out of range.
//
// Guard: an off-by-one at len(member) would panic on the first ID past the
// last commodity, which a caller holding a stale ID can reach.
func TestRoleTable_OutOfRangeCommodity(t *testing.T) {
	t.Parallel()
	cfg := roleConfig()
	tab := newRoleTable(cfg)
	n := CommodityID(len(cfg.Commodities))
	for _, id := range []CommodityID{-1, n, n + 1, 1 << 30} {
		if tab.row(1).isTradable(id) {
			t.Errorf("isTradable(%d) = true, want false", id)
		}
		if tab.tradableSomewhere(id) {
			t.Errorf("tradableSomewhere(%d) = true, want false", id)
		}
	}
}
