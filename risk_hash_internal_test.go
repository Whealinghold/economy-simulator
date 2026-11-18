package economysim

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"
)

// riskHashConfig is a Config with only the risk module: the tests below read its
// bytes, not a world.
func riskHashConfig() Config {
	return Config{Modules: Modules{Risk: &Risk{
		Routes: []RouteRisk{
			{From: "b", To: "a", Mode: "m", StrikeBP: 30, SeverityBP: 40, Common: true},
			{From: "a", To: "b", Mode: "m", StrikeBP: 10, SeverityBP: 20},
		},
		Stores: []StoreRisk{
			{Region: "b", Class: "k", StrikeBP: 5, SeverityBP: 6},
			{Region: "a", Class: "k", StrikeBP: 7, SeverityBP: 8, Common: true},
		},
	}}}
}

// TestHash_RiskFieldIds verifies the bytes of the risk module under id 16 of the
// Modules row (after Credit 15 and before Polity 19), that the pointer is written
// even when empty, and that the fields have the ids spec-config.md 8.2.2 gives:
// Routes 1 and Stores 2, RouteRisk From 1 to Common 6, StoreRisk Region 1 to Common 5.
//
// Requirement: FR-RSK-001, FR-CFG-089
//
// Guard: a pair written out of id order, an empty module that writes nothing, or a
// field written under another id would hash a configuration differently from the one
// the spec defines.
func TestHash_RiskFieldIds(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	u64 := func(n uint64) []byte { return binary.BigEndian.AppendUint64(nil, n) }
	str := func(s string) []byte { return append(u64(uint64(len(s))), s...) }

	e := &encoder{}
	pairsModules(e, Modules{Credit: &Credit{}, Risk: &Risk{}, Polity: &PolityConfig{}})
	at := bytes.Index(e.buf, append(id(16), 0, 0))
	if at < 0 || bytes.Index(e.buf, id(15)) > at || at > bytes.Index(e.buf, append(id(19), 0, 0)) {
		t.Errorf("pairsModules(Credit, Risk, Polity) = % x, want the empty pair 0010 0000 after id 15 and before id 19", e.buf)
	}

	e = &encoder{}
	pairsRisk(e, Risk{
		Routes: []RouteRisk{{From: "f", To: "t", Mode: "m", StrikeBP: 1, SeverityBP: 2, Common: true}},
		Stores: []StoreRisk{{Region: "r", Class: "c", StrikeBP: 3, SeverityBP: 4}},
	})
	want := bytes.Join([][]byte{
		id(1), u64(1), // Routes: one element
		id(1), str("f"), id(2), str("t"), id(3), str("m"), id(4), u64(1), id(5), u64(2), id(6), {1}, {0, 0},
		id(2), u64(1), // Stores: one element
		id(1), str("r"), id(2), str("c"), id(3), u64(3), id(4), u64(4), {0, 0},
	}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsRisk = % x, want % x", e.buf, want)
	}
}

// TestHash_RiskOnOff verifies that the module on, even empty, differs from the module
// off, that the order the rows are written in changes nothing, that each number and
// each flag of a row is hashed, and that the module off leaves the hash of the
// configuration as it was.
//
// Requirement: FR-RSK-001, FR-RSK-002, FR-CFG-089
func TestHash_RiskOnOff(t *testing.T) {
	t.Parallel()
	off := Config{}
	if HashConfig(off) == HashConfig(Config{Modules: Modules{Risk: &Risk{}}}) {
		t.Error("HashConfig of the module on and empty = that of the module off, want different")
	}
	hash := HashConfig(riskHashConfig())

	shuffled := riskHashConfig()
	r := shuffled.Modules.Risk
	slices.Reverse(r.Routes)
	slices.Reverse(r.Stores)
	if HashConfig(shuffled) != hash {
		t.Error("HashConfig changed with the order of the rows, want canonical order")
	}

	tests := []struct {
		name string
		set  func(r *Risk)
	}{
		{"route strike", func(r *Risk) { r.Routes[0].StrikeBP++ }},
		{"route severity", func(r *Risk) { r.Routes[0].SeverityBP++ }},
		{"route common", func(r *Risk) { r.Routes[0].Common = !r.Routes[0].Common }},
		{"route mode", func(r *Risk) { r.Routes[0].Mode = "n" }},
		{"route from", func(r *Risk) { r.Routes[0].From = "z" }},
		{"route to", func(r *Risk) { r.Routes[0].To = "z" }},
		{"store strike", func(r *Risk) { r.Stores[0].StrikeBP++ }},
		{"store severity", func(r *Risk) { r.Stores[0].SeverityBP++ }},
		{"store common", func(r *Risk) { r.Stores[0].Common = !r.Stores[0].Common }},
		{"store class", func(r *Risk) { r.Stores[0].Class = "z" }},
		{"store region", func(r *Risk) { r.Stores[0].Region = "z" }},
		{"a row less", func(r *Risk) { r.Routes = r.Routes[:1] }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := riskHashConfig()
			tt.set(c.Modules.Risk)
			if HashConfig(c) == hash {
				t.Errorf("HashConfig did not change with %s, want it hashed", tt.name)
			}
		})
	}
}

// TestHash_RiskNilWritesNothing verifies that a configuration with the module nil
// hashes as one that never had the field: the bytes of the Modules row hold no id
// 16 (spec-config.md 8.2: a module that is off leaves no trace).
//
// Requirement: FR-RSK-001
//
// Guard: the pinned hash of the all-off reference configuration must not move when
// a module is added.
func TestHash_RiskNilWritesNothing(t *testing.T) {
	t.Parallel()
	e := &encoder{}
	pairsModules(e, Modules{Credit: &Credit{}, Polity: &PolityConfig{}})
	if bytes.Contains(e.buf, binary.BigEndian.AppendUint16(nil, 16)) {
		t.Errorf("pairsModules with Risk nil = % x, want no id 16", e.buf)
	}
}
