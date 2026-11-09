package economysim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestHash_MigrationFieldIds verifies the bytes of the migration module under the
// ids of spec-config.md 8.2.2 (MoveSpeed 1 to MoveInShare 9, Regions 10 with
// Region 1, OutShare 2, InShare 3), that the module is the pair of id 9 after
// Subsistence (8) and before Estates (25) whatever the order of building, that
// each field changes the hash, that a module that is on and empty differs from
// one that is off, and that the order of the region rows does not.
//
// Requirement: FR-MIG-001, FR-CFG-078, FR-CFG-105
//
// Guard: another module on with a higher id and with a lower id; a pair written
// out of id order would hash a configuration differently from the one the spec
// defines.
func TestHash_MigrationFieldIds(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	u64 := func(n uint64) []byte { return binary.BigEndian.AppendUint64(nil, n) }
	num := func(n uint16, v uint64) []byte { return bytes.Join([][]byte{id(n), u64(v)}, nil) }

	e := &encoder{}
	pairsMigration(e, Migration{MoveSpeed: 20, MoveReviewTicks: 30, MoveBelow: 10_000, MoveGain: 1_000, MoveWidth: 5_000,
		MoveCooldown: 60, MoveHorizon: 180, MoveOutShare: 50, MoveInShare: 100})
	want := bytes.Join([][]byte{num(1, 20), num(2, 30), num(3, 10_000), num(4, 1_000), num(5, 5_000),
		num(6, 60), num(7, 180), num(8, 50), num(9, 100)}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsMigration(defaults, speed 20) = % x, want % x", e.buf, want)
	}

	// A row is Region 1 (a string: its length as 8 bytes, then the bytes), OutShare 2 and InShare 3, in a list of id 10.
	e = &encoder{}
	pairsMigration(e, Migration{Regions: []RegionMove{{Region: "n", OutShare: 7, InShare: 9}}})
	row := bytes.Join([][]byte{id(1), u64(1), []byte("n"), num(2, 7), num(3, 9)}, nil)
	if !bytes.Contains(e.buf, row) || !bytes.Contains(e.buf, id(10)) {
		t.Errorf("pairsMigration(one row) = % x, want it to hold the list id 10 and the row % x", e.buf, row)
	}

	e = &encoder{}
	pairsModules(e, Modules{Estates: &Estates{}, Migration: &Migration{}, Subsistence: &SubsistenceConfig{}, Population: &Population{}})
	// Each empty module writes its id and the terminator 0 0 only, but Migration also writes
	// its nine fields at 0: a number is written even at zero, so the stream is read by prefix.
	if got := e.buf; !bytes.HasPrefix(got, bytes.Join([][]byte{id(3), {0, 0}, id(8), {0, 0}, id(9)}, nil)) || !bytes.HasSuffix(got, bytes.Join([][]byte{id(25), {0, 0}}, nil)) {
		t.Errorf("pairsModules(Population, Subsistence, Migration, Estates) = % x, want ids 3, 8, 9, 25 in ascending order", got)
	}

	base := Config{}
	on := base
	m := DefaultMigration()
	m.MoveSpeed = 20
	m.Regions = []RegionMove{{Region: "a", OutShare: 1, InShare: 2}, {Region: "b", OutShare: 3, InShare: 4}}
	on.Modules.Migration = &m
	empty := base
	empty.Modules.Migration = &Migration{}
	if HashConfig(on) == HashConfig(base) || HashConfig(empty) == HashConfig(base) || HashConfig(empty) == HashConfig(on) {
		t.Errorf("HashConfig of the module off, on and empty must all differ")
	}
	for name, mutate := range map[string]func(m *Migration){
		"MoveSpeed":       func(m *Migration) { m.MoveSpeed++ },
		"MoveReviewTicks": func(m *Migration) { m.MoveReviewTicks++ },
		"MoveBelow":       func(m *Migration) { m.MoveBelow++ },
		"MoveGain":        func(m *Migration) { m.MoveGain++ },
		"MoveWidth":       func(m *Migration) { m.MoveWidth++ },
		"MoveCooldown":    func(m *Migration) { m.MoveCooldown++ },
		"MoveHorizon":     func(m *Migration) { m.MoveHorizon++ },
		"MoveOutShare":    func(m *Migration) { m.MoveOutShare++ },
		"MoveInShare":     func(m *Migration) { m.MoveInShare++ },
		"Regions.Region":  func(m *Migration) { m.Regions[0].Region = "c" },
		"Regions.Out":     func(m *Migration) { m.Regions[0].OutShare++ },
		"Regions.In":      func(m *Migration) { m.Regions[1].InShare++ },
	} {
		changed := base
		c := *on.Modules.Migration
		c.Regions = append([]RegionMove(nil), c.Regions...)
		mutate(&c)
		changed.Modules.Migration = &c
		if HashConfig(changed) == HashConfig(on) {
			t.Errorf("HashConfig did not change with %s, want it hashed", name)
		}
	}
	swapped := on
	s := *on.Modules.Migration
	s.Regions = []RegionMove{s.Regions[1], s.Regions[0]}
	swapped.Modules.Migration = &s
	if HashConfig(swapped) != HashConfig(on) {
		t.Errorf("HashConfig changed with the order of the region rows, want the rows hashed by region")
	}
}

// TestHash_RoleFixed verifies that Role.Fixed is hashed as the flag of id 20 only
// while the migration module is on and only when true, and that it is the last
// pair of the role.
//
// Requirement: FR-MIG-001, FR-CFG-078, FR-CFG-105
func TestHash_RoleFixed(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	tail := func(on, fixed bool) []byte {
		e := &encoder{}
		pairsRole(e, Role{ID: "a", Fixed: fixed}, false, on)
		return e.buf
	}
	if !bytes.Equal(tail(false, true), tail(false, false)) {
		t.Errorf("pairsRole with the module off and Fixed true = % x, want the bytes of Fixed false", tail(false, true))
	}
	if !bytes.Equal(tail(true, false), tail(false, false)) {
		t.Errorf("pairsRole with the module on and Fixed false = % x, want no pair of id 20", tail(true, false))
	}
	if want := append(id(20), 1); !bytes.HasSuffix(tail(true, true), want) {
		t.Errorf("pairsRole with the module on and Fixed true = % x, want it to end with % x", tail(true, true), want)
	}
	role := func(fixed bool) Config { return Config{Roles: []Role{{ID: "a", Fixed: fixed}}} }
	off, offTrue := role(false), role(true)
	if HashConfig(off) != HashConfig(offTrue) {
		t.Errorf("HashConfig changed with Fixed while the module is off, want an off module to leave no trace")
	}
	on, onTrue := role(false), role(true)
	on.Modules.Migration, onTrue.Modules.Migration = &Migration{MoveSpeed: 1}, &Migration{MoveSpeed: 1}
	if HashConfig(on) == HashConfig(onTrue) {
		t.Errorf("HashConfig did not change with Fixed while the module is on, want it hashed")
	}
}
