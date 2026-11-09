package economysim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestHashConfig_TransportIds verifies the bytes of one mode under the ids of
// spec-transport.md 11 and spec-config.md 8.2.2 (Modes 1; ID 1, Speed 2,
// UnitMass 3, UnitBulk 4, Crew 5, Vehicle 6, WearPerTick 7, Feed 8, Haul 9,
// HaulSpan 10), that Label is not hashed, that the module is the pair of id 4
// between Population (3) and Resources (5), and that a module that is on and
// empty differs from one that is off.
//
// Requirement: FR-TRN-037, FR-CFG-071
func TestHashConfig_TransportIds(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	u64 := func(n uint64) []byte { return binary.BigEndian.AppendUint64(nil, n) }
	text := func(n uint16, s string) []byte {
		return bytes.Join([][]byte{id(n), u64(uint64(len(s))), []byte(s)}, nil)
	}
	num := func(n uint16, v uint64) []byte { return bytes.Join([][]byte{id(n), u64(v)}, nil) }

	mode := Mode{ID: "wagon", Speed: 30, UnitMass: 1_200, UnitBulk: 900, Crew: 2, Vehicle: "tool", WearPerTick: 5,
		Feed: []Item{{Commodity: "fodder", Quantity: 3}}, Label: "A wagon"}
	e := &encoder{}
	pairsTransport(e, Transport{Modes: []Mode{mode}})
	want := bytes.Join([][]byte{
		id(1), u64(1), // Modes: one element
		text(1, "wagon"), num(2, 30), num(3, 1_200), num(4, 900), num(5, 2), text(6, "tool"), num(7, 5),
		id(8), u64(1), text(1, "fodder"), num(2, 3), id(0), // Feed: one Item, then its terminator
		id(0), // end of the mode
	}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsTransport(one mode) = % x, want % x", e.buf, want)
	}

	hired := Mode{ID: "coach", Speed: 30, UnitMass: 1, Haul: "haul", HaulSpan: 10}
	e = &encoder{}
	pairsTransport(e, Transport{Modes: []Mode{hired}})
	want = bytes.Join([][]byte{id(1), u64(1), text(1, "coach"), num(2, 30), num(3, 1), text(9, "haul"), num(10, 10), id(0)}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsTransport(hired mode) = % x, want % x", e.buf, want)
	}

	e = &encoder{}
	pairsModules(e, Modules{Population: &Population{}, Transport: &Transport{}, Resources: &Resources{}})
	// Population and Resources write their own pairs after the id; with no field
	// set a module writes the id and the terminator only, so the bytes are 3 0 4 0 5 0.
	if want := bytes.Join([][]byte{id(3), id(0), id(4), id(0), id(5), id(0)}, nil); !bytes.Equal(e.buf, want) {
		t.Errorf("pairsModules(Population, Transport, Resources) = % x, want % x: ids 3, 4, 5 in ascending order", e.buf, want)
	}

	base := Config{}
	on, labelled := base, base
	on.Modules.Transport = &Transport{Modes: []Mode{mode}}
	l := mode
	l.Label = "another"
	labelled.Modules.Transport = &Transport{Modes: []Mode{l}}
	empty := base
	empty.Modules.Transport = &Transport{}
	if HashConfig(on) != HashConfig(labelled) {
		t.Errorf("HashConfig changed with Mode.Label, want it presentation only")
	}
	if HashConfig(empty) == HashConfig(base) || HashConfig(on) == HashConfig(base) {
		t.Errorf("HashConfig of the module on equals the module off, want different")
	}
}
