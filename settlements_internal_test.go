package economysim

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"
)

// settlementsHashConfig is a Config with only the settlements module, written out of
// order on purpose: the classes by ID and not by Enter, the flags and limits and the
// region rows unsorted.
func settlementsHashConfig() Config {
	return Config{Modules: Modules{Settlements: &Settlements{
		BandBP: 1_000, Dwell: 30,
		Classes: []SettlementClass{
			{ID: "city", Enter: 800, Flags: []string{"hall", "fair"}, Limits: []ClassLimit{{Name: "b", Value: 2}, {Name: "a", Value: 1}}, Label: "a"},
			{ID: "hamlet", Enter: 0},
			{ID: "town", Enter: 300, Flags: []string{"fair"}},
		},
		Regions: []RegionSettlement{{Region: "west", ThresholdBP: 6_000}, {Region: "north", ThresholdBP: 10_000, Floor: "town"}},
	}}}
}

// TestCanonical_ClassesSortedByEnterNotID verifies that the canonical order of the
// classes is by Enter and not by ID (the case of spec-settlements.md 3.2.1: `city`
// sorts before `hamlet` as a string and is the last class), that flags, limits and
// region rows are sorted, and that the written configuration is left as it was.
//
// Requirement: FR-SET-002, FR-SET-030
func TestCanonical_ClassesSortedByEnterNotID(t *testing.T) {
	t.Parallel()
	c := settlementsHashConfig()
	got := c.canonical().Modules.Settlements
	var ids []string
	for _, k := range got.Classes {
		ids = append(ids, k.ID)
	}
	if want := []string{"hamlet", "town", "city"}; !slices.Equal(ids, want) {
		t.Errorf("canonical classes = %v, want %v (by Enter)", ids, want)
	}
	if !slices.Equal(got.Classes[2].Flags, []string{"fair", "hall"}) || got.Classes[2].Limits[0].Name != "a" {
		t.Errorf("canonical city = %+v, want flags and limits sorted by name", got.Classes[2])
	}
	if got.Regions[0].Region != "north" {
		t.Errorf("canonical regions = %+v, want north first", got.Regions)
	}
	if c.Modules.Settlements.Classes[0].ID != "city" || c.Modules.Settlements.Classes[0].Flags[0] != "hall" {
		t.Error("canonical changed the configuration it was called on, want a copy")
	}
	if (*Settlements)(nil).canonical() != nil {
		t.Error("canonical of nil is not nil")
	}
}

// TestHash_SettlementsFieldIds verifies the bytes of the module under id 11 of the
// Modules row (between Merchants 10 and Storage 12), that the pointer is written
// even when empty, that the fields have the ids spec-config.md 8.2.2 gives and that
// Role.Needs and Role.Ceiling take 26 and 27.
//
// Requirement: FR-SET-030
//
// Guard: a pair out of id order, an empty module that writes nothing or a field with
// another id would hash a configuration differently from the one the spec defines.
func TestHash_SettlementsFieldIds(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	u64 := func(n uint64) []byte { return binary.BigEndian.AppendUint64(nil, n) }
	str := func(s string) []byte { return append(u64(uint64(len(s))), s...) }

	e := &encoder{}
	pairsModules(e, Modules{Merchants: &Merchants{}, Settlements: &Settlements{}, Storage: &Storage{}})
	at := bytes.Index(e.buf, append(id(11), 0, 0))
	if at < 0 || bytes.Index(e.buf, id(10)) > at || at > bytes.Index(e.buf, append(id(12), 0, 0)) {
		t.Errorf("pairsModules(Merchants, Settlements, Storage) = % x, want the empty pair 000b 0000 between ids 10 and 12", e.buf)
	}

	e = &encoder{}
	pairsSettlements(e, Settlements{
		Classes: []SettlementClass{{ID: "c", Enter: 5, Flags: []string{"f"}, Limits: []ClassLimit{{Name: "l", Value: 3}}, Label: "ignored"}},
		BandBP:  7, Dwell: 9,
		Regions: []RegionSettlement{{Region: "r", ThresholdBP: 8, Floor: "c"}},
	})
	want := bytes.Join([][]byte{
		id(1), u64(1), // Classes: one element
		id(1), str("c"), id(2), u64(5), id(3), u64(1), str("f"),
		id(4), u64(1), id(1), str("l"), id(2), u64(3), {0, 0}, // Limits: one element, end of the limit
		{0, 0},                       // end of the class
		id(2), u64(7), id(3), u64(9), // BandBP, Dwell
		id(4), u64(1), id(1), str("r"), id(2), u64(8), id(3), str("c"), {0, 0},
	}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsSettlements = % x, want % x", e.buf, want)
	}

	e = &encoder{}
	pairsRole(e, Role{ID: "x", Needs: "n", Ceiling: "k"}, false, false)
	tail := bytes.Join([][]byte{id(26), str("n"), id(27), str("k")}, nil)
	if !bytes.HasSuffix(e.buf, tail) {
		t.Errorf("pairsRole(Needs, Ceiling) = % x, want it to end with % x", e.buf, tail)
	}
}

// TestHash_SettlementsOnOff verifies that the module on, even empty, differs from the
// module off; that the Label and the written order of the classes, flags, limits and
// rows change nothing (two configurations of equal content hash alike); that each
// hashed field changes the hash; and that Role.Needs and Role.Ceiling do.
//
// Requirement: FR-SET-030, FR-SET-001
func TestHash_SettlementsOnOff(t *testing.T) {
	t.Parallel()
	if HashConfig(Config{}) == HashConfig(Config{Modules: Modules{Settlements: &Settlements{}}}) {
		t.Error("HashConfig of the module on and empty = that of the module off, want different")
	}
	hash := HashConfig(settlementsHashConfig())

	shuffled := settlementsHashConfig()
	s := shuffled.Modules.Settlements
	s.Classes[0], s.Classes[2] = s.Classes[2], s.Classes[0]
	s.Classes[2].Flags[0], s.Classes[2].Flags[1] = s.Classes[2].Flags[1], s.Classes[2].Flags[0]
	s.Regions[0], s.Regions[1] = s.Regions[1], s.Regions[0]
	s.Classes[2].Label = "another"
	if HashConfig(shuffled) != hash {
		t.Error("HashConfig changed with the written order or with a Label, want canonical order and presentation only")
	}

	for name, mutate := range map[string]func(m *Settlements){
		"band":      func(m *Settlements) { m.BandBP++ },
		"dwell":     func(m *Settlements) { m.Dwell++ },
		"enter":     func(m *Settlements) { m.Classes[0].Enter++ },
		"flag":      func(m *Settlements) { m.Classes[0].Flags = nil },
		"limit":     func(m *Settlements) { m.Classes[0].Limits[0].Value++ },
		"threshold": func(m *Settlements) { m.Regions[0].ThresholdBP++ },
		"floor":     func(m *Settlements) { m.Regions[0].Floor = "hamlet" },
	} {
		c := settlementsHashConfig()
		mutate(c.Modules.Settlements)
		if HashConfig(c) == hash {
			t.Errorf("HashConfig did not change with the %s, want it hashed", name)
		}
	}

	role := func(needs, ceiling string) Config {
		return Config{Roles: []Role{{ID: "r", Needs: needs, Ceiling: ceiling}}}
	}
	if HashConfig(role("", "")) == HashConfig(role("f", "")) || HashConfig(role("f", "")) == HashConfig(role("", "f")) {
		t.Error("HashConfig of a role does not depend on Needs and Ceiling, or does not tell them apart")
	}
}

// TestHash_SettlementsAfterLowerModuleIds verifies that the module's pair sits after
// those of ids below 11 and before those of 12, 13, 19 and 25 whatever the order the
// pointers were built in, so the hash id follows the Modules row.
//
// Requirement: FR-CFG-085
func TestHash_SettlementsAfterLowerModuleIds(t *testing.T) {
	t.Parallel()
	e := &encoder{}
	pairsModules(e, Modules{Merchants: &Merchants{}, Storage: &Storage{}, Settlements: &Settlements{}, Employment: &Employment{}, Polity: &PolityConfig{}, Estates: &Estates{}})
	var ids []uint16
	for i := 0; i+2 <= len(e.buf); {
		n := binary.BigEndian.Uint16(e.buf[i:])
		i += 2
		if n == 0 {
			continue
		}
		ids = append(ids, n)
		for i+2 <= len(e.buf) && binary.BigEndian.Uint16(e.buf[i:]) != 0 {
			i++
		}
	}
	if !slices.IsSorted(ids) || !slices.Contains(ids, 11) {
		t.Errorf("module ids in order of writing = %v, want ascending and holding 11", ids)
	}
}

// TestSettlementsThresholds_RegionScale verifies the effective thresholds the
// warnings read: a region with no row has the scale 10 000, one with a row has its
// own, and an outside-node-free list of regions gives one table each.
//
// Requirement: FR-SET-008
func TestSettlementsThresholds_RegionScale(t *testing.T) {
	t.Parallel()
	c := Config{Regions: []Region{{ID: "north"}, {ID: "west"}}}
	m := &Settlements{BandBP: 1_000, Classes: []SettlementClass{{ID: "a"}, {ID: "b", Enter: 300}}, Regions: []RegionSettlement{{Region: "west", ThresholdBP: 6_000}}}
	e, l, ok := settlementThresholds(&c, m)
	if !ok || !slices.Equal(e[0], []int64{0, 300}) || !slices.Equal(e[1], []int64{0, 180}) || !slices.Equal(l[1], []int64{0, 162}) {
		t.Errorf("settlementThresholds = %v, %v, %v, want north 300 and west 180 with leave 162", e, l, ok)
	}
	if _, _, ok := settlementThresholds(&c, &Settlements{BandBP: -1, Classes: []SettlementClass{{ID: "a"}}}); ok {
		t.Error("settlementThresholds with a negative band ok = true, want false")
	}
}
