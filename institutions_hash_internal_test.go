package economysim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestHash_InstitutionsFieldIds verifies the bytes of the institutions module under
// the ids of the Modules row (Institutions 21, between Polity 19 and Technology 24),
// that the pointer is written even when empty, that the fields of the five structs
// have the ids the encoder documents, that a Label has none and that the module off
// leaves no trace.
//
// Requirement: FR-ORG-051, FR-CFG-105
//
// Guard: a pair written out of id order, an empty module that writes nothing or a
// field that shares an id would hash a configuration differently from the one the
// spec defines, and an off module that leaves a trace would move the pinned hash of
// the all-off reference.
func TestHash_InstitutionsFieldIds(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	u64 := func(n uint64) []byte { return binary.BigEndian.AppendUint64(nil, n) }
	str := func(s string) []byte { return append(u64(uint64(len(s))), s...) }

	e := &encoder{}
	pairsModules(e, Modules{Polity: &PolityConfig{}, Institutions: &Institutions{}, Technology: &Technology{}})
	at := bytes.Index(e.buf, append(id(21), 0, 0))
	if at < 0 || bytes.Index(e.buf, append(id(19), 0, 0)) > at || at > bytes.Index(e.buf, append(id(24), 0, 0)) {
		t.Errorf("pairsModules(Polity, Institutions, Technology) = % x, want the empty pair 00 15 00 00 after id 19 and before id 24", e.buf)
	}

	e = &encoder{}
	pairsInstitutions(e, Institutions{
		Bodies:      []Body{{ID: "g", Roles: []string{"r"}, Regions: []string{"n"}, Jurisdictions: []string{"j"}, Places: 3, MaxTrainees: 2, Needs: "f", Label: "ignored"}},
		Rights:      []Right{{ID: "p", Body: "g", Activity: "produce", Subject: "s", Region: "n", Jurisdiction: "j", Who: "both", Label: "ignored"}},
		Controls:    []Control{{ID: "c", Commodity: "k", Region: "n", Jurisdiction: "j", CapBP: 5, FloorBP: 4, Label: "ignored"}},
		RateLimits:  []RateLimit{{ID: "l", Region: "n", Jurisdiction: "j", CapBP: 6, Label: "ignored"}},
		OwnUseBound: true,
	})
	list := func(id uint16, n uint64) []byte { return append(append([]byte{}, idBytes(id)...), u64(n)...) }
	want := bytes.Join([][]byte{
		list(1, 1), // Bodies
		id(1), str("g"), id(2), u64(1), str("r"), id(3), u64(1), str("n"), id(4), u64(1), str("j"), id(5), u64(3), id(6), u64(2), id(7), str("f"), {0, 0},
		list(2, 1), // Rights
		id(1), str("p"), id(2), str("g"), id(3), str("produce"), id(4), str("s"), id(5), str("n"), id(6), str("j"), id(7), str("both"), {0, 0},
		list(3, 1), // Controls
		id(1), str("c"), id(2), str("k"), id(3), str("n"), id(4), str("j"), id(5), u64(5), id(6), u64(4), {0, 0},
		list(4, 1), // RateLimits
		id(1), str("l"), id(2), str("n"), id(3), str("j"), id(4), u64(6), {0, 0},
		id(5), {1}, // OwnUseBound
	}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsInstitutions = % x, want % x", e.buf, want)
	}

	off := Config{}
	if !bytes.Equal(encodeConfig(off.canonical()), encodeConfig((Config{Modules: Modules{Institutions: nil}}).canonical())) {
		t.Error("encodeConfig of the module off differs from one without the field")
	}
}

// idBytes is the big-endian bytes of a field id.
func idBytes(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
