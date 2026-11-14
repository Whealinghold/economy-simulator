package economysim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestHash_PolityFieldIds verifies the bytes of the polity module under the ids of
// the Modules row (Polity 19, between Employment 13 and Estates 25), that the
// pointer is written even when empty, that the fields have the ids the encoder
// documents and that the module off leaves no trace.
//
// Requirement: FR-POL-022, FR-CFG-105
//
// Guard: a pair written out of id order, or an empty module that writes nothing,
// would hash a configuration differently from the one the spec defines, and an off
// module that leaves a trace would move the pinned hash of the all-off reference.
func TestHash_PolityFieldIds(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	u64 := func(n uint64) []byte { return binary.BigEndian.AppendUint64(nil, n) }
	str := func(s string) []byte { return append(u64(uint64(len(s))), s...) }

	e := &encoder{}
	pairsModules(e, Modules{Estates: &Estates{}, Employment: &Employment{}, Polity: &PolityConfig{}})
	got := e.buf
	pair := append(id(19), 0, 0)
	at := bytes.Index(got, pair)
	if at < 0 || bytes.Index(got, append(id(13), 0, 0)) > at || at > bytes.Index(got, append(id(25), 0, 0)) {
		t.Errorf("pairsModules(Employment, Polity, Estates) = % x, want the empty pair % x after id 13 and before id 25", got, pair)
	}

	e = &encoder{}
	pairsPolity(e, PolityConfig{
		Jurisdictions: []Jurisdiction{{ID: "a", Parent: "b", Seat: "r", StartMoney: 7, Label: "ignored"}},
		Members:       []Member{{Region: "r", Jurisdiction: "a"}},
	})
	want := bytes.Join([][]byte{
		id(1), u64(1), // Jurisdictions: one element
		id(1), str("a"), id(2), str("b"), id(3), str("r"), id(4), u64(7), {0, 0}, // ID, Parent, Seat, StartMoney, end
		id(2), u64(1), // Members: one element
		id(1), str("r"), id(2), str("a"), {0, 0},
	}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsPolity = % x, want % x", e.buf, want)
	}

	off := Config{}
	if !bytes.Equal(encodeConfig(off.canonical()), encodeConfig((Config{Modules: Modules{Polity: nil}}).canonical())) {
		t.Error("encodeConfig of the module off differs from one without the field")
	}
}
