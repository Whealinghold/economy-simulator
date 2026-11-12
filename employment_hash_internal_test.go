package economysim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestHash_EmploymentFieldIds verifies the bytes of the employment module and
// of the role fields under the ids of spec-config.md 8.2.2 (Modules.Employment
// 13, between Merchants 10 and Estates 25; Role.Hireable 23, written only when
// true; Role.Employer 24, written when non-nil even at zero; Employer.PayrollBP
// 1), that each field changes the hash, and that the module off and the fields
// at their off value leave the hash of the same configuration unchanged.
//
// Requirement: FR-EMP-079, FR-CFG-086, FR-CFG-105
//
// Guard: a pair written out of id order, a field left out of the hash or an off
// field that leaves a trace would hash a configuration differently from the one
// the spec defines, and would move the pinned hash of the all-off reference.
func TestHash_EmploymentFieldIds(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	u64 := func(n uint64) []byte { return binary.BigEndian.AppendUint64(nil, n) }
	num := func(n uint16, v uint64) []byte { return bytes.Join([][]byte{id(n), u64(v)}, nil) }

	e := &encoder{}
	pairsModules(e, Modules{Estates: &Estates{}, Merchants: &Merchants{MaxOpen: 1_000}, Employment: &Employment{}})
	got := e.buf
	pair := append(id(13), 0, 0)
	if !bytes.Contains(got, pair) || bytes.Index(got, id(10)) > bytes.Index(got, pair) || bytes.Index(got, pair) > bytes.Index(got, append(id(25), 0, 0)) {
		t.Errorf("pairsModules(Merchants, Employment, Estates) = % x, want the pair % x after id 10 and before id 25", got, pair)
	}

	tail := func(r Role) []byte {
		e := &encoder{}
		pairsRole(e, r, false, false)
		return e.buf
	}
	bare := tail(Role{ID: "a"})
	if !bytes.Equal(bare, tail(Role{ID: "a", Hireable: false, Employer: nil})) {
		t.Errorf("pairsRole(off fields) differs from a role without them")
	}
	if got := tail(Role{ID: "a", Hireable: true}); !bytes.HasSuffix(got, append(id(23), 1)) {
		t.Errorf("pairsRole(Hireable) = % x, want it to end with the flag of id 23", got)
	}
	if got := tail(Role{ID: "a", Employer: &Employer{PayrollBP: 5_000}}); !bytes.HasSuffix(got, bytes.Join([][]byte{id(24), num(1, 5_000), {0, 0}}, nil)) {
		t.Errorf("pairsRole(Employer 5000) = % x, want the block of id 24 with PayrollBP 1 and the terminator", got)
	}
	if got := tail(Role{ID: "a", Employer: &Employer{}}); !bytes.HasSuffix(got, append(id(24), 0, 0)) {
		t.Errorf("pairsRole(zero block) = % x, want the id 24 and the terminator, written even at zero (a zero PayrollBP is no pair)", got)
	}

	base := Config{Roles: []Role{{ID: "a"}}}
	with := func(mutate func(c *Config)) Config {
		c := Config{Roles: []Role{{ID: "a", Hireable: true, Employer: &Employer{PayrollBP: 5_000}}}}
		c.Modules.Employment = &Employment{}
		if mutate != nil {
			mutate(&c)
		}
		return c
	}
	on := HashConfig(with(nil))
	if on == HashConfig(base) {
		t.Errorf("HashConfig of the module on equals that of the module off, want it hashed")
	}
	for name, mutate := range map[string]func(c *Config){
		"module":    func(c *Config) { c.Modules.Employment = nil },
		"Hireable":  func(c *Config) { c.Roles[0].Hireable = false },
		"Employer":  func(c *Config) { c.Roles[0].Employer = nil },
		"PayrollBP": func(c *Config) { c.Roles[0].Employer.PayrollBP++ },
	} {
		if HashConfig(with(mutate)) == on {
			t.Errorf("HashConfig does not move with %s, want every field hashed", name)
		}
	}
}
