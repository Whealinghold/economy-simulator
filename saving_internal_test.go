package economysim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestHash_SavingFieldIds verifies the bytes of the saving module under id 23 of the
// Modules row (after Policy 22 and before Technology 24), that the pointer is written
// even when empty (`0017 0000`), that the fields have the ids spec-config.md 8.2.2
// gives, each written only when non-zero, and the bytes of a Saver block under id 29 of
// Role, which is written even when empty.
//
// Requirement: FR-CFG-096
//
// Guard: a pair out of id order, an empty module that writes nothing, a zero field that
// is written or a block with the wrong ids would hash a configuration differently from
// the one the spec defines.
func TestHash_SavingFieldIds(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	u64 := func(n uint64) []byte { return binary.BigEndian.AppendUint64(nil, n) }
	str := func(s string) []byte { return append(u64(uint64(len(s))), s...) }

	e := &encoder{}
	pairsModules(e, Modules{Policy: &PolicyConfig{}, Saving: &Saving{}, Technology: &Technology{}})
	at := bytes.Index(e.buf, append(id(23), 0, 0))
	if at < 0 || bytes.Index(e.buf, append(id(22), 0, 0)) > at || at > bytes.Index(e.buf, append(id(24), 0, 0)) {
		t.Errorf("pairsModules(Policy, Saving, Technology) = % x, want the empty pair 0017 0000 after id 22 and before id 24", e.buf)
	}

	e = &encoder{}
	pairsSaving(e, Saving{SafetyTicks: 1, CommitBandBP: 3})
	want := bytes.Join([][]byte{id(1), u64(1), id(3), u64(3)}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsSaving with a zero spread = % x, want % x (a zero field is not written)", e.buf, want)
	}

	e = &encoder{}
	pairsSaver(e, Saver{
		PrecautionShareBP: 1, PrecautionCapTicks: 2, HurdleBP: 3, PaybackTicks: 4, RepairBelowBP: 5,
		Worth: []ProjectWorth{{Class: "mill", PerTick: 6, Count: 7}},
	})
	want = bytes.Join([][]byte{
		id(1), u64(1), id(2), u64(2), id(3), u64(3), id(4), u64(4), id(5), u64(5),
		id(6), u64(1), id(1), str("mill"), id(2), u64(6), id(3), u64(7), {0, 0},
	}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsSaver = % x, want % x", e.buf, want)
	}

	e = &encoder{}
	pairsRole(e, Role{Saver: &Saver{}}, false, false)
	if !bytes.HasSuffix(e.buf, append(id(29), 0, 0)) {
		t.Errorf("pairsRole of an empty block = % x, want it to end with the empty pair 001d 0000", e.buf)
	}
	e = &encoder{}
	pairsRole(e, Role{}, false, false)
	if bytes.Contains(e.buf, id(29)) {
		t.Errorf("pairsRole of a role with no block = % x, want no id 29", e.buf)
	}
}

// TestCanonical_SavingCopiesAndSortsWorth verifies that the canonical copy holds a block
// of its own with the Worth rows in class-ID order, and the module of its own: changing
// the consumer's Config after canonical moves nothing in the copy, and the given order
// of the rows leaves the hash unchanged.
//
// Requirement: FR-CFG-096
func TestCanonical_SavingCopiesAndSortsWorth(t *testing.T) {
	t.Parallel()
	build := func(rows ...ProjectWorth) Config {
		return Config{
			Roles:   []Role{{ID: "a", Saver: &Saver{Worth: rows}}},
			Modules: Modules{Saving: &Saving{SafetyTicks: 2}},
		}
	}
	c := build(ProjectWorth{Class: "road", PerTick: 1, Count: 1}, ProjectWorth{Class: "barn", PerTick: 2, Count: 1})
	cp := c.canonical()
	if got := cp.Roles[0].Saver.Worth[0].Class; got != "barn" {
		t.Errorf("canonical Worth[0].Class = %q, want barn (class-ID order)", got)
	}
	c.Roles[0].Saver.Worth[0].PerTick = 99
	c.Roles[0].Saver.HurdleBP = 5
	c.Modules.Saving.SafetyTicks = 9
	if cp.Roles[0].Saver.Worth[1].PerTick != 1 || cp.Roles[0].Saver.HurdleBP != 0 || cp.Modules.Saving.SafetyTicks != 2 {
		t.Errorf("canonical copy moved with the consumer's Config: %+v %+v", *cp.Roles[0].Saver, *cp.Modules.Saving)
	}
	swapped := build(ProjectWorth{Class: "barn", PerTick: 2, Count: 1}, ProjectWorth{Class: "road", PerTick: 1, Count: 1})
	given := build(ProjectWorth{Class: "road", PerTick: 1, Count: 1}, ProjectWorth{Class: "barn", PerTick: 2, Count: 1})
	if HashConfig(swapped) != HashConfig(given) {
		t.Error("HashConfig changed with the order of the Worth rows, want canonical order")
	}
	if (*Saving)(nil).canonical() != nil || (*Saver)(nil).canonical() != nil {
		t.Error("canonical of a nil module or block is not nil")
	}
}

// TestHash_ModuleIdsInOrder verifies that the pairs of the modules of ids 15 to 25 are
// written in id order, whatever the order of the struct: the encoding of all eleven
// together is the encoding of each alone, joined from id 15 to id 25, and a Modules with
// none of them writes nothing. The ids pair with the continuation functions
// (pairsLastModules 15 to 22, pairsTailModules 23, pairsEndModules 24 and 25), which the
// Lizard split moves around.
//
// Requirement: FR-CFG-096
//
// Guard: a split of pairsModules that moves a block into the wrong continuation would
// reorder the pairs and change the hash of every configuration with both modules on,
// though each module alone would still hash the same.
func TestHash_ModuleIdsInOrder(t *testing.T) {
	t.Parallel()
	setters := []func(m *Modules){
		func(m *Modules) { m.Credit = &Credit{} },                 // 15
		func(m *Modules) { m.Risk = &Risk{} },                     // 16
		func(m *Modules) { m.Infrastructure = &Infrastructure{} }, // 17
		func(m *Modules) { m.Insurance = &Insurance{} },           // 18
		func(m *Modules) { m.Polity = &PolityConfig{} },           // 19
		func(m *Modules) { m.Fiscal = &Fiscal{} },                 // 20
		func(m *Modules) { m.Institutions = &Institutions{} },     // 21
		func(m *Modules) { m.Policy = &PolicyConfig{} },           // 22
		func(m *Modules) { m.Saving = &Saving{} },                 // 23
		func(m *Modules) { m.Technology = &Technology{} },         // 24
		func(m *Modules) { m.Estates = &Estates{} },               // 25
	}
	encode := func(m Modules) []byte {
		e := &encoder{}
		pairsModules(e, m)
		return e.buf
	}
	if got := encode(Modules{}); len(got) != 0 {
		t.Fatalf("pairsModules(Modules{}) = % x, want nothing", got)
	}
	var all Modules
	var want []byte
	for i, set := range setters {
		var one Modules
		set(&one)
		alone := encode(one)
		wantID := binary.BigEndian.AppendUint16(nil, uint16(15+i))
		if len(alone) < 2 || !bytes.HasPrefix(alone, wantID) {
			t.Fatalf("pairsModules with module %d alone = % x, want it to start with id %d", 15+i, alone, 15+i)
		}
		want = append(want, alone...)
		set(&all)
	}
	if got := encode(all); !bytes.Equal(got, want) {
		t.Errorf("pairsModules(ids 15 to 25) = % x, want the modules alone joined in id order % x", got, want)
	}
}

// TestHash_NilSavingWritesNothing verifies that with no saving module and no Saver
// block the encoding carries neither id 23 of Modules nor id 29 of Role, and that the
// canonical copy keeps both nil: a configuration that does not use the module hashes
// and copies as it did before the module existed (the pinned hash of the reference
// configuration is the end-to-end proof, in hash_test.go).
//
// Requirement: FR-CFG-096, FR-SAV-001
//
// Guard: a pointer that is written when nil, or a canonical copy that turns nil into an
// empty module, would move every existing hash and checkpoint.
func TestHash_NilSavingWritesNothing(t *testing.T) {
	t.Parallel()
	e := &encoder{}
	pairsModules(e, Modules{Policy: &PolicyConfig{}, Technology: &Technology{}})
	with := &encoder{}
	pairsModules(with, Modules{Policy: &PolicyConfig{}, Saving: &Saving{}, Technology: &Technology{}})
	if len(with.buf) != len(e.buf)+4 {
		t.Errorf("pairsModules with an empty module is %d bytes, want %d (the nil module's %d plus the empty pair 0017 0000)", len(with.buf), len(e.buf)+4, len(e.buf))
	}
	c := Config{Roles: []Role{{ID: "a"}}}
	cp := c.canonical()
	if cp.Modules.Saving != nil || cp.Roles[0].Saver != nil {
		t.Errorf("canonical of a nil module and block = %v, %v, want both nil", cp.Modules.Saving, cp.Roles[0].Saver)
	}
	r := &encoder{}
	pairsRole(r, Role{ID: "a"}, false, false)
	if bytes.Contains(r.buf, binary.BigEndian.AppendUint16(nil, 29)) {
		t.Errorf("pairsRole of a role with no block = % x, want no id 29", r.buf)
	}
}
