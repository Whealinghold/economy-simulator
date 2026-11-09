package economysim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestHashConfig_Information verifies the bytes of the information module under
// the ids of spec-config.md 8.2.2 (InfoSpeed 1, RumourErrorBP 2, RumourMaxAge 3),
// that the module is the pair of id 6 between Resources (5) and Subsistence (8)
// whatever the order of building, that each of the three fields changes the
// hash, that a module that is on and empty differs from one that is off, and that
// Role.Informed (id 19) is a pair only with the module on and the flag true.
//
// Requirement: FR-INF-044, FR-CFG-105
func TestHashConfig_Information(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	u64 := func(n uint64) []byte { return binary.BigEndian.AppendUint64(nil, n) }
	num := func(n uint16, v uint64) []byte { return bytes.Join([][]byte{id(n), u64(v)}, nil) }

	e := &encoder{}
	pairsInformation(e, Information{InfoSpeed: 40, RumourErrorBP: 500, RumourMaxAge: 60})
	if want := bytes.Join([][]byte{num(1, 40), num(2, 500), num(3, 60)}, nil); !bytes.Equal(e.buf, want) {
		t.Errorf("pairsInformation(40, 500, 60) = % x, want % x", e.buf, want)
	}

	e = &encoder{}
	pairsModules(e, Modules{Subsistence: &SubsistenceConfig{}, Information: &Information{}, Resources: &Resources{}, Population: &Population{}})
	// An Information with every field 0 writes the id and the terminator only; the
	// pointer is written even when the struct is zero (spec-config.md 8.2.1).
	// Subsistence, Population and Resources write their own pairs after the id,
	// which the empty value leaves out except for their ids and terminators.
	got := e.buf
	var ids []uint16
	for i := 0; i+4 <= len(got); i += 2 {
		if v := binary.BigEndian.Uint16(got[i:]); v == 3 || v == 5 || v == 6 || v == 8 {
			ids = append(ids, v)
		}
	}
	if len(ids) < 4 || ids[0] != 3 || ids[1] != 5 || ids[2] != 6 || ids[3] != 8 {
		t.Errorf("pairsModules(Population, Resources, Information, Subsistence) ids = %v in % x, want 3, 5, 6, 8 in ascending order", ids, got)
	}

	base := Config{}
	on := base
	on.Modules.Information = &Information{InfoSpeed: 40, RumourErrorBP: 500, RumourMaxAge: 60}
	if HashConfig(on) == HashConfig(base) {
		t.Errorf("HashConfig of the module on equals the module off, want different")
	}
	empty := base
	empty.Modules.Information = &Information{}
	if HashConfig(empty) == HashConfig(base) || HashConfig(empty) == HashConfig(on) {
		t.Errorf("HashConfig of the module on and empty equals the module off or the module on with fields, want different")
	}
	for name, mutate := range map[string]func(m *Information){
		"InfoSpeed":     func(m *Information) { m.InfoSpeed++ },
		"RumourErrorBP": func(m *Information) { m.RumourErrorBP++ },
		"RumourMaxAge":  func(m *Information) { m.RumourMaxAge++ },
	} {
		changed := base
		m := *on.Modules.Information
		mutate(&m)
		changed.Modules.Information = &m
		if HashConfig(changed) == HashConfig(on) {
			t.Errorf("HashConfig did not change with %s, want it hashed", name)
		}
	}
}

// TestHashConfig_RoleInformed verifies that Role.Informed is hashed as the flag
// of id 19 only while the information module is on and only when true: with the
// module off a true flag hashes as false, and with it on the flag changes the
// hash.
//
// Requirement: FR-INF-044, FR-CFG-105
func TestHashConfig_RoleInformed(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	tail := func(on, informed bool) []byte {
		e := &encoder{}
		pairsRole(e, Role{ID: "a", Informed: informed}, on)
		return e.buf
	}
	if !bytes.Equal(tail(false, true), tail(false, false)) {
		t.Errorf("pairsRole with the module off and Informed true = % x, want the bytes of Informed false", tail(false, true))
	}
	if !bytes.Equal(tail(true, false), tail(false, false)) {
		t.Errorf("pairsRole with the module on and Informed false = % x, want no pair of id 19", tail(true, false))
	}
	// The flag is the last pair of the role: id 19, then the byte 1 (the list writes the terminator).
	if want := append(id(19), 1); !bytes.HasSuffix(tail(true, true), want) {
		t.Errorf("pairsRole with the module on and Informed true = % x, want it to end with % x", tail(true, true), want)
	}

	role := func(informed bool) Config {
		return Config{Roles: []Role{{ID: "a", Informed: informed}}}
	}
	off, offTrue := role(false), role(true)
	if HashConfig(off) != HashConfig(offTrue) {
		t.Errorf("HashConfig changed with Informed while the module is off, want an off module to leave no trace")
	}
	on, onTrue := role(false), role(true)
	on.Modules.Information, onTrue.Modules.Information = &Information{InfoSpeed: 1}, &Information{InfoSpeed: 1}
	if HashConfig(on) == HashConfig(onTrue) {
		t.Errorf("HashConfig did not change with Informed while the module is on, want it hashed")
	}
}
