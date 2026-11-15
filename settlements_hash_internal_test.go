package economysim

import (
	"bytes"
	"testing"
)

// TestHash_SettlementsEveryNameMovesIt verifies that each identifier and each list
// membership of the module is part of the hash, beyond the numbers that
// TestHash_SettlementsOnOff mutates: a class ID, a flag name, a limit name, a region
// name, and the addition or removal of a class, a flag, a limit or a region row. Two
// configurations that differ in any of them build different worlds, so they must not
// share a hash (a checkpoint of one must not restore into the other).
//
// Requirement: FR-SET-030
func TestHash_SettlementsEveryNameMovesIt(t *testing.T) {
	t.Parallel()
	base := HashConfig(settlementsHashConfig())
	if again := HashConfig(settlementsHashConfig()); again != base {
		t.Fatalf("HashConfig of the same configuration twice = %x, %x, want equal", base, again)
	}
	tests := []struct {
		name   string
		mutate func(m *Settlements)
	}{
		{"class id", func(m *Settlements) { m.Classes[1].ID = "village" }},
		{"flag name", func(m *Settlements) { m.Classes[0].Flags[0] = "market" }},
		{"flag added", func(m *Settlements) { m.Classes[1].Flags = []string{"hall"} }},
		{"limit name", func(m *Settlements) { m.Classes[0].Limits[0].Name = "z" }},
		{"limit added", func(m *Settlements) { m.Classes[1].Limits = []ClassLimit{{Name: "a", Value: 0}} }},
		{"limit zero against absent", func(m *Settlements) { m.Classes[0].Limits = m.Classes[0].Limits[:1] }},
		{"class added", func(m *Settlements) { m.Classes = append(m.Classes, SettlementClass{ID: "capital", Enter: 5_000}) }},
		{"class removed", func(m *Settlements) { m.Classes = m.Classes[:2] }},
		{"region name", func(m *Settlements) { m.Regions[0].Region = "east" }},
		{"region row added", func(m *Settlements) {
			m.Regions = append(m.Regions, RegionSettlement{Region: "east", ThresholdBP: 10_000})
		}},
		{"region row removed", func(m *Settlements) { m.Regions = m.Regions[:1] }},
		{"floor removed", func(m *Settlements) { m.Regions[1].Floor = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := settlementsHashConfig()
			tt.mutate(c.Modules.Settlements)
			if got := HashConfig(c); got == base {
				t.Errorf("HashConfig after %s = the base hash %x, want a different one", tt.name, base)
			}
		})
	}
}

// TestHash_SettlementsOffAddsNothing verifies that a configuration with the module off
// hashes as one with no trace of it: a nil module writes no pair, and a role's Needs
// is a pair appended after the bytes of a role without it, so a configuration of an earlier release keeps its hash.
//
// Requirement: FR-SET-001, FR-SET-030
//
// Guard: TestHashConfig_ReferencePinned pins one configuration; this states the rule
// for any, so a pair written for an empty value would fail here with a name.
func TestHash_SettlementsOffAddsNothing(t *testing.T) {
	t.Parallel()
	bare := Config{Roles: []Role{{ID: "r"}}}
	e1, e2 := &encoder{}, &encoder{}
	pairsModules(e1, bare.Modules)
	pairsModules(e2, Modules{Settlements: nil})
	if string(e1.buf) != string(e2.buf) || len(e1.buf) != 0 {
		t.Errorf("pairsModules with every module off = % x, want no bytes", e1.buf)
	}
	empty, set := &encoder{}, &encoder{}
	pairsRole(empty, Role{ID: "r"}, false, false)
	pairsRole(set, Role{ID: "r", Needs: "n"}, false, false)
	if !bytes.HasPrefix(set.buf, empty.buf) || len(set.buf) == len(empty.buf) {
		t.Errorf("pairsRole(Needs set) = % x, want the bytes of the role without it % x plus a pair at the end", set.buf, empty.buf)
	}
}
