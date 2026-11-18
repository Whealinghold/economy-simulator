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

// TestLoad_EmploymentRejects verifies that the loader refuses what the file must
// not say: a payroll_bp that is not an integer, a key of the hire form with the
// wrong type, an unknown key, a key outside the employer block, and an employment
// table that is not a table. Each case breaks one thing in an otherwise valid
// document.
//
// Requirement: FR-EMP-080
func TestLoad_EmploymentRejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, doc string
	}{
		{"hire as an integer", minimal + "[modules.employment]\nhire = 1\n"},
		{"max_open as a float", minimal + "[modules.employment]\nhire = true\nmax_open = 4.5\n"},
		{"unknown key of the module", minimal + "[modules.employment]\nhires = true\n"},
		{"recipes as a string", minimal + "[modules.employment]\nhire = true\n[role.a.employer]\nrecipes = \"prepare\"\n"},
		{"max_hires as a string", minimal + "[modules.employment]\nhire = true\n[role.a.employer]\nmax_hires = \"2\"\n"},
		{"unknown key of the block", minimal + "[modules.employment]\nhire = true\n[role.a.employer]\nworker_share = 5000\n"},
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

// hireDoc is a hire-form document: the module with its MaxOpen, an employer role
// with the four fields of the block and a Hireable role, in the form of
// spec-employment.md 24.5.2 and 13.
const hireDoc = minimal + `
[modules.employment]
hire = true
max_open = 50
[role.boss]
recipes = ["prepare", "grind"]
[role.boss.employer]
recipes = ["grind", "prepare"]
max_hires = 6
worker_share_bp = 4000
hire_yield_bp = 8000
[role.hand]
hireable = true
`

// TestLoad_EmploymentHire verifies that the keys of spec-employment.md 13 load the
// hire form: hire and max_open on the module, the four fields on the block read
// over DefaultHireEmployer (an absent key takes its default and PayrollBP stays 0,
// so the mixed form is never made by a default), a block with only payroll_bp
// stays the payroll form, an absent max_open with hire takes the default, and a
// file and the same Go value hash alike, whatever the order of the hire list.
//
// Requirement: FR-EMP-079, FR-EMP-080, FR-CFG-086
func TestLoad_EmploymentHire(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(hireDoc)
	if err != nil {
		t.Fatalf("Parse(hireDoc) error = %v, want nil", err)
	}
	if want := (economysim.Employment{Hire: true, MaxOpen: 50}); !reflect.DeepEqual(*got.Modules.Employment, want) {
		t.Errorf("Parse(hireDoc).Modules.Employment = %+v, want %+v", *got.Modules.Employment, want)
	}
	wantBlock := economysim.Employer{Recipes: []string{"grind", "prepare"}, MaxHires: 6, WorkerShareBP: 4_000, HireYieldBP: 8_000}
	if boss := got.Roles[0]; !reflect.DeepEqual(*boss.Employer, wantBlock) {
		t.Errorf("Parse(hireDoc) boss block = %+v, want %+v", *boss.Employer, wantBlock)
	}
	def, err := parseStrings(minimal + "[modules.employment]\nhire = true\n[role.a.employer]\nrecipes = [\"x\"]\n")
	if err != nil {
		t.Fatalf("Parse(defaults) error = %v, want nil", err)
	}
	wantDef := economysim.DefaultHireEmployer()
	wantDef.Recipes = []string{"x"}
	if def.Modules.Employment.MaxOpen != economysim.DefaultMaxOpen || !reflect.DeepEqual(*def.Roles[0].Employer, wantDef) {
		t.Errorf("Parse(defaults) = %+v and %+v, want MaxOpen %d and block %+v",
			*def.Modules.Employment, *def.Roles[0].Employer, economysim.DefaultMaxOpen, wantDef)
	}
	pay, err := parseStrings(minimal + "[modules.employment]\n[role.a.employer]\npayroll_bp = 3000\n")
	if err != nil {
		t.Fatalf("Parse(payroll) error = %v, want nil", err)
	}
	if want := (economysim.Employer{PayrollBP: 3_000}); !reflect.DeepEqual(*pay.Roles[0].Employer, want) || pay.Modules.Employment.Hire {
		t.Errorf("Parse(payroll) = %+v and %+v, want the payroll block %+v and Hire false", *pay.Roles[0].Employer, *pay.Modules.Employment, want)
	}
	boss := economysim.DefaultRole()
	boss.ID, boss.Recipes = "boss", []string{"prepare", "grind"}
	boss.Employer = &economysim.Employer{Recipes: []string{"prepare", "grind"}, MaxHires: 6, WorkerShareBP: 4_000, HireYieldBP: 8_000}
	hand := economysim.DefaultRole()
	hand.ID, hand.Hireable = "hand", true
	goValue := economysim.Config{World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar(), Roles: []economysim.Role{boss, hand}}
	goValue.Modules.Employment = &economysim.Employment{Hire: true, MaxOpen: 50}
	if economysim.HashConfig(got) != economysim.HashConfig(goValue) {
		t.Errorf("HashConfig(file) != HashConfig(the same Go value), want equal (FR-EMP-079)")
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
	f.Add([]byte(hireDoc), []byte("[role.boss.employer]\nrecipes = [1]\n"), []byte("[modules.employment]\nmax_open = -1\n"))
	f.Add([]byte(minimal), []byte("[modules.employment]\nhire = true\nmax_open = 9223372036854775807\n"), []byte("[role.a.employer]\nmax_hires = 0\npayroll_bp = 1\n"))
	f.Fuzz(loadProperties)
}
