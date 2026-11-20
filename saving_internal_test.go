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
