package economysim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// The hash of the hire form of the employment module (spec-employment.md 24.5.1):
// the pairs appended to id 13 of Modules and to id 24 of Role, with the next free
// ids, and the proof that no payroll hash moves.

// empHash builds the bytes the spec defines, by hand, so that a test compares
// the encoder with the text and not with itself.
type empHash struct{}

func (empHash) id(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
func (empHash) u64(n uint64) []byte {
	return binary.BigEndian.AppendUint64(nil, n)
}
func (h empHash) num(n uint16, v uint64) []byte { return bytes.Join([][]byte{h.id(n), h.u64(v)}, nil) }
func (h empHash) text(s string) []byte {
	return bytes.Join([][]byte{h.u64(uint64(len(s))), []byte(s)}, nil)
}

// TestHash_EmploymentPayrollUnchanged verifies that the payroll form hashes to the
// bytes it hashed to before the hire form existed: the module as the id 13 and the
// terminator with no pair, and an employer block as id 24, PayrollBP as id 1 and
// the terminator, with none of the hire pairs.
//
// Requirement: FR-EMP-079, FR-CFG-086
//
// Guard: a hire pair written at its zero value, or PayrollBP moved, would change
// the hash of every payroll world and of every checkpoint written for one.
func TestHash_EmploymentPayrollUnchanged(t *testing.T) {
	t.Parallel()
	var h empHash
	e := &encoder{}
	pairsEmployment(e, Employment{})
	if len(e.buf) != 0 {
		t.Errorf("pairsEmployment(payroll form) = % x, want no pair", e.buf)
	}
	e = &encoder{}
	pairsEmployer(e, Employer{PayrollBP: 5_000})
	if want := h.num(1, 5_000); !bytes.Equal(e.buf, want) {
		t.Errorf("pairsEmployer(PayrollBP 5000) = % x, want % x", e.buf, want)
	}
	e = &encoder{}
	pairsRole(e, Role{ID: "a", Employer: &Employer{PayrollBP: 5_000}}, false, false)
	if want := bytes.Join([][]byte{h.id(24), h.num(1, 5_000), {0, 0}}, nil); !bytes.HasSuffix(e.buf, want) {
		t.Errorf("pairsRole(payroll employer) = % x, want it to end with % x", e.buf, want)
	}
}

// TestHash_EmploymentHireEncodingPinned verifies the bytes of the hire form: the
// module's Hire as id 1 (a flag) and MaxOpen as id 2, and the block's Recipes as
// id 2 (the count, then each string), MaxHires 3, WorkerShareBP 4 and
// HireYieldBP 5, each written only when non-zero and PayrollBP absent.
//
// Requirement: FR-EMP-079, FR-CFG-086
//
// Guard: the ids are part of the hash of every hire world; a renumbering or a
// reordering would hash a configuration differently from the one the spec defines.
func TestHash_EmploymentHireEncodingPinned(t *testing.T) {
	t.Parallel()
	var h empHash
	e := &encoder{}
	pairsEmployment(e, Employment{Hire: true, MaxOpen: 1_000})
	if want := bytes.Join([][]byte{h.id(1), {1}, h.num(2, 1_000)}, nil); !bytes.Equal(e.buf, want) {
		t.Errorf("pairsEmployment(hire, 1000) = % x, want % x", e.buf, want)
	}
	e = &encoder{}
	pairsEmployer(e, Employer{Recipes: []string{"a", "bc"}, MaxHires: 4, WorkerShareBP: 5_000, HireYieldBP: 10_000})
	want := bytes.Join([][]byte{
		h.id(2), h.u64(2), h.text("a"), h.text("bc"), h.num(3, 4), h.num(4, 5_000), h.num(5, 10_000)}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsEmployer(hire block) = % x, want % x", e.buf, want)
	}
	e = &encoder{}
	pairsModules(e, Modules{Employment: &Employment{Hire: true, MaxOpen: 7}})
	if want := bytes.Join([][]byte{h.id(13), h.id(1), {1}, h.num(2, 7), {0, 0}}, nil); !bytes.Contains(e.buf, want) {
		t.Errorf("pairsModules(hire employment) = % x, want it to contain % x", e.buf, want)
	}
}

// TestHash_EmploymentHireFieldsMoveTheHash verifies that each field of the hire
// form changes the hash of an otherwise equal configuration, that the order of the
// hire list does not (it is a set), and that a MaxOpen beside Hire false leaves the
// payroll hash as it was.
//
// Requirement: FR-EMP-079
//
// Guard: a field left out of the hash would let two worlds that behave differently
// share a hash, and a checkpoint of one restore into the other.
func TestHash_EmploymentHireFieldsMoveTheHash(t *testing.T) {
	t.Parallel()
	with := func(mutate func(c *Config)) Config {
		c := Config{Roles: []Role{{ID: "a", Recipes: []string{"x", "y"}, Employer: &Employer{
			Recipes: []string{"x", "y"}, MaxHires: 4, WorkerShareBP: 5_000, HireYieldBP: 10_000}}}}
		c.Modules.Employment = &Employment{Hire: true, MaxOpen: 1_000}
		if mutate != nil {
			mutate(&c)
		}
		return c
	}
	on := HashConfig(with(nil))
	for name, mutate := range map[string]func(c *Config){
		"Hire":          func(c *Config) { c.Modules.Employment.Hire = false },
		"MaxOpen":       func(c *Config) { c.Modules.Employment.MaxOpen++ },
		"Recipes":       func(c *Config) { c.Roles[0].Employer.Recipes = []string{"x"} },
		"MaxHires":      func(c *Config) { c.Roles[0].Employer.MaxHires++ },
		"WorkerShareBP": func(c *Config) { c.Roles[0].Employer.WorkerShareBP++ },
		"HireYieldBP":   func(c *Config) { c.Roles[0].Employer.HireYieldBP++ },
		"PayrollBP":     func(c *Config) { c.Roles[0].Employer.PayrollBP = 5_000 },
	} {
		if HashConfig(with(mutate)) == on {
			t.Errorf("HashConfig does not move with %s, want every field hashed", name)
		}
	}
	if got := HashConfig(with(func(c *Config) { c.Roles[0].Employer.Recipes = []string{"y", "x"} })); got != on {
		t.Errorf("HashConfig(hire list y, x) != HashConfig(hire list x, y), want equal: the list is a set")
	}
	payroll := Config{Roles: []Role{{ID: "a", Employer: &Employer{PayrollBP: 5_000}}}}
	payroll.Modules.Employment = &Employment{}
	stray := payroll
	stray.Modules.Employment = &Employment{MaxOpen: 9}
	if HashConfig(stray) != HashConfig(payroll) {
		t.Errorf("HashConfig(MaxOpen 9 with Hire false) != HashConfig(payroll form), want equal: MaxOpen is written only with Hire")
	}
}
