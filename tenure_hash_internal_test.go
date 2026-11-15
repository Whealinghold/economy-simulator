package economysim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// hashBytes builds the expected encoding of a run of fields: a field id is 2 bytes
// big-endian, a number 8, a flag 1, a string its length as 8 bytes and then its bytes
// (spec-config.md 8.2.1).
type hashBytes struct{ b []byte }

func (h *hashBytes) id(n uint16) *hashBytes {
	h.b = binary.BigEndian.AppendUint16(h.b, n)
	return h
}

func (h *hashBytes) u64(n uint64) *hashBytes {
	h.b = binary.BigEndian.AppendUint64(h.b, n)
	return h
}

func (h *hashBytes) str(id uint16, s string) *hashBytes {
	h.id(id).u64(uint64(len(s)))
	h.b = append(h.b, s...)
	return h
}

func (h *hashBytes) flag(id uint16) *hashBytes {
	h.id(id)
	h.b = append(h.b, 1)
	return h
}

// TestHash_TenureEncodingPinned verifies the bytes of the tenure module in the
// configuration hash (spec-config.md 8.2.2): the module is id 14, its fields are 1 to 7
// in declaration order, a term has Region 1, Resource 2, HasRent 3, Rent 4, HasClosed 5
// and Closed 6, a holding has Region 1, Resource 2, Holder 3 and ShareBP 4, a holder has
// Kind 1 and Role 2. A zero field writes nothing, a list is its count and its rows each
// ended by a terminator, and the two defaults of the module are written by canonical.
//
// Requirement: FR-CFG-104, FR-LND-001
//
// Guard: the encoding is part of every checkpoint's configuration hash; an id that moved
// would silently invalidate every saved world with the module on.
func TestHash_TenureEncodingPinned(t *testing.T) {
	t.Parallel()
	m := TenureConfig{
		Terms: []TenureTerm{{Region: "n", Resource: "a", HasRent: true, Rent: 9, HasClosed: true, Closed: true}},
		Holdings: []Holding{
			{Region: "n", Resource: "a", Holder: HolderRef{Kind: HolderKindRole, Role: "g"}, ShareBP: 20},
		},
		RentStepBP: 5, RentFloorBP: 6, RentCeilingBP: 11_000, OwnUseExempt: true, MaxHolderShareBP: 7,
	}
	want := (&hashBytes{}).id(14).
		id(1).u64(1).str(1, "n").str(2, "a").flag(3).id(4).u64(9).flag(5).flag(6).id(0).
		id(2).u64(1).str(1, "n").str(2, "a").id(3).str(1, "role").str(2, "g").id(0).id(4).u64(20).id(0).
		id(3).u64(5).id(4).u64(6).id(5).u64(11_000).flag(6).id(7).u64(7).id(0)
	e := &encoder{}
	pairsLaterModules(e, Modules{Tenure: m.canonical()})
	if !bytes.Equal(e.buf, want.b) {
		t.Errorf("pairsLaterModules(tenure) = %x, want %x", e.buf, want.b)
	}
}

// TestHash_TenureDefaultsAreWritten verifies that a module that is on and empty writes
// the two defaults that canonical gives it, RentCeilingBP 20 000 as field 5 and
// MaxHolderShareBP 10 000 as field 7, and that the spelling of a default as 0 and as its
// value hash alike (N-345: the defaults of a module's own field are written).
//
// Requirement: FR-CFG-104
func TestHash_TenureDefaultsAreWritten(t *testing.T) {
	t.Parallel()
	want := (&hashBytes{}).id(14).id(5).u64(DefaultRentCeilingBP).id(7).u64(DefaultMaxHolderShareBP).id(0)
	e := &encoder{}
	pairsLaterModules(e, Modules{Tenure: (&TenureConfig{}).canonical()})
	if !bytes.Equal(e.buf, want.b) {
		t.Errorf("pairsLaterModules(empty tenure) = %x, want %x", e.buf, want.b)
	}
	spelled := &TenureConfig{RentCeilingBP: DefaultRentCeilingBP, MaxHolderShareBP: DefaultMaxHolderShareBP}
	a, b := &encoder{}, &encoder{}
	pairsLaterModules(a, Modules{Tenure: (&TenureConfig{}).canonical()})
	pairsLaterModules(b, Modules{Tenure: spelled.canonical()})
	if !bytes.Equal(a.buf, b.buf) {
		t.Errorf("a default spelled 0 and spelled out hash differently: %x and %x", a.buf, b.buf)
	}
}
