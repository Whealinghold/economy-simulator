package economysim

import (
	"encoding/binary"
	"reflect"
	"slices"
	"testing"
)

// TestEncodeConfig_EdgesAndPricesBytes verifies the bytes of the two lists, built
// here by hand from spec-config.md 8.2.2 and not by the encoder's helpers: the
// version byte, then field id 8 (Edges) with its count and, per edge, From 1, To
// 2, Distance 3 and the terminator, then id 9 (RegionPrices) with Region 1,
// Commodity 2, Price 3, then the configuration's terminator. Ids are 2 bytes,
// counts and numbers 8, a string its length as 8 bytes and its bytes.
//
// Requirement: FR-REG-053, FR-CFG-104
func TestEncodeConfig_EdgesAndPricesBytes(t *testing.T) {
	t.Parallel()
	be := binary.BigEndian
	u16 := func(b []byte, v uint16) []byte { return be.AppendUint16(b, v) }
	u64 := func(b []byte, v uint64) []byte { return be.AppendUint64(b, v) }
	str := func(b []byte, s string) []byte { return append(u64(b, uint64(len(s))), s...) }

	want := []byte{0x01}
	want = u16(want, 8)
	want = u64(want, 1)
	want = u16(want, 1)
	want = str(want, "a")
	want = u16(want, 2)
	want = str(want, "b")
	want = u16(want, 3)
	want = u64(want, 5)
	want = u16(want, 0)
	want = u16(want, 9)
	want = u64(want, 1)
	want = u16(want, 1)
	want = str(want, "a")
	want = u16(want, 2)
	want = str(want, "x")
	want = u16(want, 3)
	want = u64(want, 9)
	want = u16(want, 0)
	want = u16(want, 0)

	cfg := Config{
		Edges:        []Edge{{From: "a", To: "b", Distance: 5}},
		RegionPrices: []RegionPrice{{Region: "a", Commodity: "x", Price: 9}},
	}
	if got := encodeConfig(cfg.canonical()); !slices.Equal(got, want) {
		t.Errorf("encodeConfig(one edge, one override) = %x, want %x", got, want)
	}
}

// TestCanonical_EdgesAndPricesSorted verifies that canonical orders the edges by
// (From, To) and the overrides by (Region, Commodity), as strings, by a stable
// sort, and shares nothing with the receiver, which keeps the order written.
//
// Requirement: FR-REG-002, FR-REG-014, FR-REG-035
func TestCanonical_EdgesAndPricesSorted(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Edges: []Edge{{"b", "a", 1}, {"a", "c", 2}, {"a", "b", 3}, {"b", "c", 4}},
		RegionPrices: []RegionPrice{
			{"b", "x", 1}, {"a", "y", 2}, {"a", "x", 3},
		},
	}
	c := cfg.canonical()
	wantEdges := []Edge{{"a", "b", 3}, {"a", "c", 2}, {"b", "a", 1}, {"b", "c", 4}}
	wantPrices := []RegionPrice{{"a", "x", 3}, {"a", "y", 2}, {"b", "x", 1}}
	if !slices.Equal(c.Edges, wantEdges) {
		t.Errorf("canonical().Edges = %+v, want %+v", c.Edges, wantEdges)
	}
	if !slices.Equal(c.RegionPrices, wantPrices) {
		t.Errorf("canonical().RegionPrices = %+v, want %+v", c.RegionPrices, wantPrices)
	}
	if cfg.Edges[0].From != "b" || cfg.RegionPrices[0].Region != "b" {
		t.Errorf("canonical() reordered the receiver: %+v %+v", cfg.Edges, cfg.RegionPrices)
	}
	c.Edges[0].Distance, c.RegionPrices[0].Price = 99, 99
	if cfg.Edges[1].Distance != 2 || cfg.RegionPrices[2].Price != 3 {
		t.Errorf("changing the canonical copy changed the original: %+v %+v", cfg.Edges, cfg.RegionPrices)
	}
	if empty := (Config{}).canonical(); empty.Edges != nil || empty.RegionPrices != nil {
		t.Errorf("canonical() of empty lists = %v, %v, want nil", empty.Edges, empty.RegionPrices)
	}
}

// TestConfigRefPrice_OverrideElseCommodity verifies the config-level effective
// reference price (spec-regions.md 5.2.2): the override of the (region,
// commodity) if there is a row, else the commodity's price, for every pair of a
// small canonical configuration.
//
// Requirement: FR-REG-036, FR-REG-032
func TestConfigRefPrice_OverrideElseCommodity(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Commodities: []Commodity{{ID: "a", ReferencePrice: 10}, {ID: "b", ReferencePrice: 20}, {ID: "c", ReferencePrice: 30}},
		Regions:     []Region{{ID: "n"}, {ID: "s"}},
		RegionPrices: []RegionPrice{
			{Region: "n", Commodity: "b", Price: 21}, {Region: "s", Commodity: "a", Price: 11}, {Region: "s", Commodity: "c", Price: 31},
		},
	}.canonical()
	want := [2][3]Price{{10, 21, 30}, {11, 20, 31}}
	for r := range RegionID(2) {
		for k := range CommodityID(3) {
			if got := cfg.refPrice(r, k); got != want[r][k] {
				t.Errorf("refPrice(%d, %d) = %d, want %d", r, k, got, want[r][k])
			}
		}
	}
}

// TestReadGeometry_SkipsWhatIsInvalid verifies that the warnings' reader of the
// graph keeps the valid edges in (From, To) order with the index of their row as
// written, skips an unknown end, a loop, a distance outside [1, MaxDistance] and
// the second row of a pair, and refuses a configuration over a count bound.
//
// Requirement: FR-REG-031
func TestReadGeometry_SkipsWhatIsInvalid(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Regions: []Region{{ID: "b"}, {ID: "a"}, {ID: "c"}, {ID: "a"}},
		Edges: []Edge{
			{"c", "a", 4}, {"a", "zz", 1}, {"a", "a", 1}, {"a", "b", 0}, {"a", "b", MaxDistance + 1},
			{"b", "c", 7}, {"b", "c", 8}, {"a", "b", MaxDistance},
		},
	}
	g, ok := readGeometry(&cfg)
	if !ok {
		t.Fatal("readGeometry(small configuration) = false, want true")
	}
	if want := []string{"a", "b", "c"}; !slices.Equal(g.ids, want) {
		t.Errorf("readGeometry ids = %q, want %q (sorted, repeated ID once)", g.ids, want)
	}
	// a>b (row 7), b>c (row 5, the first of its pair), c>a (row 0).
	wantRows := []int{7, 5, 0}
	if !slices.Equal(g.row, wantRows) || len(g.edges) != 3 || g.edges[0].Length != MaxDistance || g.edges[1].Length != 7 || g.edges[2].Length != 4 {
		t.Errorf("readGeometry rows = %v, edges = %+v, want rows %v and lengths %d, 7, 4", g.row, g.edges, wantRows, int64(MaxDistance))
	}
	big := Config{Regions: make([]Region, MaxRegions+1)}
	if _, ok := readGeometry(&big); ok {
		t.Errorf("readGeometry(MaxRegions+1 regions) = true, want false")
	}
	big = Config{Edges: make([]Edge, MaxEdges+1)}
	if _, ok := readGeometry(&big); ok {
		t.Errorf("readGeometry(MaxEdges+1 edges) = true, want false")
	}
}

// TestRegionWarnings_ExternalModuleNotBuilt verifies that the external module
// has no entry in Modules yet.
//
// Guard: regionWarnings exempts no region from region_without_agents because
// the outside node of the external module (spec-regions.md 9.2) does not exist.
// When the module is built, this test fails and the exemption in regionWarnings
// has to be filled in the same change; without it every pack with an outside
// node would draw a spurious warning.
func TestRegionWarnings_ExternalModuleNotBuilt(t *testing.T) {
	t.Parallel()
	if _, ok := reflect.TypeOf(Modules{}).FieldByName("External"); ok {
		t.Fatal("Modules.External exists: fill the outside-node exemption of region_without_agents in regionWarnings")
	}
}
