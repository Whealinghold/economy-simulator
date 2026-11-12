package economysim_test

import (
	"errors"
	"slices"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the employment module's configuration surface in its payroll
// form: its role fields, the bound of PayrollBP, the dependencies and the shape
// rules (spec-employment.md 24.3, 24.4, FR-EMP-064 to 067). The arithmetic is
// tested in internal/employment and the hash in employment_hash_internal_test.go.

// employmentConfig returns referenceConfig with the employment module on, the
// cook as the payer (the default block) and the digger as the Hireable role.
func employmentConfig() economysim.Config {
	c := referenceConfig()
	c.Modules.Employment = &economysim.Employment{}
	for i := range c.Roles {
		switch c.Roles[i].ID {
		case "cook":
			e := economysim.DefaultEmployer()
			c.Roles[i].Employer = &e
		case "digger":
			c.Roles[i].Hireable = true
		}
	}
	return c
}

// TestValidate_EmploymentConfigIsValid verifies that the configuration the
// employment tests start from validates, so every failing case breaks one rule.
//
// Guard: the field cases start from employmentConfig; if it did not validate,
// each case would pass for the wrong reason.
func TestValidate_EmploymentConfigIsValid(t *testing.T) {
	t.Parallel()
	if err := employmentConfig().Validate(); err != nil {
		t.Fatalf("employmentConfig().Validate() = %v, want nil", err)
	}
}

// employmentFieldCases are the failing cases of the rows of the employment
// module in the parameter-owner table (spec-employment.md 24.3, 24.4,
// FR-EMP-065 to 067): each breaks one field of a valid configuration.
// buildFieldCases appends them to the table that the registry test reads.
func employmentFieldCases() []fieldCase {
	field := func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				*c = employmentConfig()
				set(c)
			}}
	}
	bp := []string{"roles", "cook", "employer", "payroll_bp"}
	return []fieldCase{
		field("Employer.PayrollBP", "0", economysim.ErrOutOfBounds, bp, "0", "1",
			func(c *economysim.Config) { roleOf(c, "cook").Employer.PayrollBP = 0 }),
		field("Employer.PayrollBP", "above 10 000", economysim.ErrOutOfBounds, bp, "10001", "10000",
			func(c *economysim.Config) { roleOf(c, "cook").Employer.PayrollBP = 10_001 }),
		field("Role.Employer", "a block while the module is off", economysim.ErrModuleDependency, []string{"roles", "cook", "employer"}, "employment", "",
			func(c *economysim.Config) {
				c.Modules.Employment = nil
				roleOf(c, "digger").Hireable = false
			}),
		field("Role.Hireable", "true while the module is off", economysim.ErrModuleDependency, []string{"roles", "digger", "hireable"}, "employment", "",
			func(c *economysim.Config) {
				c.Modules.Employment = nil
				roleOf(c, "cook").Employer = nil
			}),
		field("Role.Hireable", "a role that is also an employer", economysim.ErrEmploymentShape, []string{"roles", "cook", "hireable"}, "true", "false",
			func(c *economysim.Config) { roleOf(c, "cook").Hireable = true }),
	}
}

// TestValidate_EmploymentRules verifies the rules of spec-employment.md 24.4.1
// that no row of the parameter-owner table owns: the module on with no payer
// role and the module on with no Hireable role, each broken alone, and that the
// module off with neither field is valid.
//
// Requirement: FR-EMP-067, FR-EMP-066
func TestValidate_EmploymentRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		want   error
		path   []string
	}{
		{"no payer role", func(c *economysim.Config) { roleOf(c, "cook").Employer = nil }, economysim.ErrEmploymentShape, []string{"modules", "employment", "employer"}},
		{"no Hireable role", func(c *economysim.Config) { roleOf(c, "digger").Hireable = false }, economysim.ErrEmploymentShape, []string{"modules", "employment", "hireable"}},
		{"module off with neither field", func(c *economysim.Config) {
			c.Modules.Employment = nil
			roleOf(c, "cook").Employer = nil
			roleOf(c, "digger").Hireable = false
		}, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := employmentConfig()
			tt.mutate(&c)
			err := c.Validate()
			if tt.want == nil {
				if err != nil {
					t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
				}
				return
			}
			vs, _ := violations(err)
			if !errors.Is(err, tt.want) || len(vs) != 1 || !slices.Equal(vs[0].Path, tt.path) {
				t.Errorf("Validate(%s) = %v, want only %v at %v", tt.name, err, tt.want, tt.path)
			}
		})
	}
}

// TestDefaultEmployer verifies the default block: the placeholder share of
// spec-employment.md 24.3.2 passes validation at the bounds of the field.
//
// Requirement: FR-EMP-065
func TestDefaultEmployer(t *testing.T) {
	t.Parallel()
	if got := economysim.DefaultEmployer().PayrollBP; got != 5_000 {
		t.Errorf("DefaultEmployer().PayrollBP = %d, want 5000", got)
	}
	for _, bp := range []int64{1, 5_000, 10_000} {
		t.Run(strconv.FormatInt(bp, 10), func(t *testing.T) {
			t.Parallel()
			c := employmentConfig()
			roleOf(&c, "cook").Employer.PayrollBP = bp
			if err := c.Validate(); err != nil {
				t.Errorf("Validate(PayrollBP %d) = %v, want nil", bp, err)
			}
		})
	}
}

// TestEmployment_CanonicalCopyIsolatesTheBlock verifies that the canonical
// configuration does not share the employer block with the caller's, so a
// caller that edits its block after NewWorld cannot move the world.
//
// Guard: the hash and the world read the canonical copy; sharing the pointer
// would let a later edit change a configuration that was already validated.
func TestEmployment_CanonicalCopyIsolatesTheBlock(t *testing.T) {
	t.Parallel()
	c := employmentConfig()
	before := economysim.HashConfig(c)
	w, err := economysim.NewWorld(c)
	if err != nil {
		t.Fatalf("NewWorld(employmentConfig()) = %v, want nil", err)
	}
	roleOf(&c, "cook").Employer.PayrollBP = 1
	if after := economysim.HashConfig(employmentConfig()); after != before {
		t.Errorf("HashConfig of an unchanged configuration moved: %x, want %x", after, before)
	}
	if err := w.Step(); err != nil {
		t.Errorf("Step after the caller edited its block = %v, want nil", err)
	}
}
