package config_test

import (
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// employmentDoc is a [modules.employment] table, a payer role with its block and
// a Hireable role, in the form of spec-employment.md 24.5.2.
const employmentDoc = minimal + `
[modules.employment]
[role.boss]
[role.boss.employer]
payroll_bp = 3000
[role.hand]
hireable = true
[role.plain]
`

// TestLoad_Employment verifies that the keys of spec-employment.md 24.5.2 load
// the payroll form: the table with no key switches the module on, the block is
// read over DefaultEmployer (an absent payroll_bp takes 5 000), hireable is
// read, a role with neither keeps both off, with no table the module is off, an
// unknown key of the block is rejected, and a file and the same Go value hash
// alike.
//
// Requirement: FR-EMP-080, FR-EMP-079, FR-CFG-086
func TestLoad_Employment(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(employmentDoc)
	if err != nil {
		t.Fatalf("Parse(employmentDoc) error = %v, want nil", err)
	}
	if got.Modules.Employment == nil || len(got.Roles) != 3 {
		t.Fatalf("Parse(employmentDoc) = %+v, want the module on and three roles", got)
	}
	boss, hand, plain := got.Roles[0], got.Roles[1], got.Roles[2]
	if !reflect.DeepEqual(boss.Employer, &economysim.Employer{PayrollBP: 3_000}) || boss.Hireable {
		t.Errorf("Parse(employmentDoc) boss = %+v, want the block with 3000 and not Hireable", boss)
	}
	if !hand.Hireable || hand.Employer != nil || plain.Hireable || plain.Employer != nil {
		t.Errorf("Parse(employmentDoc) hand = %+v, plain = %+v, want hand Hireable with no block and plain with neither", hand, plain)
	}
	def, err := parseStrings(minimal + "[modules.employment]\n[role.a.employer]\n")
	if err != nil {
		t.Fatalf("Parse(defaults) error = %v, want nil", err)
	}
	if want := economysim.DefaultEmployer(); !reflect.DeepEqual(*def.Roles[0].Employer, want) {
		t.Errorf("Parse(defaults).Roles[0].Employer = %+v, want %+v", def.Roles[0].Employer, want)
	}
	if off, _ := parseStrings(minimal); off.Modules.Employment != nil {
		t.Errorf("Parse(minimal).Modules.Employment = %+v, want nil: the module is off", off.Modules.Employment)
	}
	if _, err := parseStrings(minimal + "[role.a.employer]\npayroll_bps = 1\n"); err == nil {
		t.Errorf("Parse(unknown key payroll_bps) error = nil, want it rejected")
	}
	role := func(id string) economysim.Role {
		r := economysim.DefaultRole()
		r.ID = id
		return r
	}
	b, h, p := role("boss"), role("hand"), role("plain")
	b.Employer, h.Hireable = &economysim.Employer{PayrollBP: 3_000}, true
	goValue := economysim.Config{World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar(), Roles: []economysim.Role{b, h, p}}
	goValue.Modules.Employment = &economysim.Employment{}
	if economysim.HashConfig(got) != economysim.HashConfig(goValue) {
		t.Errorf("HashConfig(file) != HashConfig(the same Go value), want equal (FR-EMP-079)")
	}
}

// TestLoad_EmploymentRejects verifies that the payroll form's loader refuses what
// the file must not say: a key of the hire form (the form this one is a subset of),
// a payroll_bp that is not an integer, a key outside the employer block, and an
// employment table that is not a table. Each case breaks one thing in an otherwise
// valid document.
//
// Requirement: FR-EMP-080
func TestLoad_EmploymentRejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, doc string
	}{
		{"hire-form key of the module", minimal + "[modules.employment]\nmax_open = 4\n"},
		{"hire-form key of the block", minimal + "[modules.employment]\n[role.a.employer]\nmax_hires = 2\n"},
		{"payroll_bp as a float", minimal + "[modules.employment]\n[role.a.employer]\npayroll_bp = 5000.5\n"},
		{"payroll_bp as a string", minimal + "[modules.employment]\n[role.a.employer]\npayroll_bp = \"5000\"\n"},
		{"hireable as an integer", minimal + "[modules.employment]\n[role.a]\nhireable = 1\n"},
		{"employment as a scalar", minimal + "[modules]\nemployment = 1\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if cfg, err := parseStrings(tt.doc); err == nil {
				t.Errorf("Parse(%s) = %+v, nil, want an error", tt.name, cfg)
			}
		})
	}
}

// FuzzLoadEmployment is FuzzLoad with the seeds of the payroll form: the document
// above and documents that break each key of the module. It asserts the same
// properties.
//
// Requirement: FR-EMP-080
//
// Fix: found by this fuzz test in its first run. A document with the module on and
// an employer role but no Hireable role validates with ErrEmploymentShape, a rule
// that validationSentinels (helpers_test.go) did not list, so classified rejected
// a correct validation error. The committed input in testdata/fuzz/FuzzLoadEmployment
// pins it. The defect was in the test's list of sentinels, not in the loader.
func FuzzLoadEmployment(f *testing.F) {
	f.Add([]byte(employmentDoc), []byte("[role.boss.employer]\npayroll_bp = 0\n"), []byte("[modules.employment]\nbogus = 1\n"))
	f.Add([]byte(minimal), []byte("[role.a.employer]\npayroll_bp = 10001\n"), []byte("[role.a]\nhireable = \"yes\"\n"))
	f.Fuzz(loadProperties)
}
