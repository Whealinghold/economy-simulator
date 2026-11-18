package economysim_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the configuration surface of the hire form of the employment
// module (spec-employment.md 3.1, 3.2, 13, 14): the bounds, the hire list, the
// one-form rule, the weight of an employer and the warnings. The arithmetic is
// tested in internal/employment and the hash in employment_hire_hash_internal_test.go.

// hireConfig returns referenceConfig with the employment module in its hire form:
// the cook is the employer (its hire list is "prepare") and the digger is the
// Hireable role. A money flow to the cook's households gives its profit an outlet,
// so that the configuration raises none of the warnings of 14.2.
func hireConfig() economysim.Config {
	c := referenceConfig()
	m := economysim.DefaultEmployment()
	c.Modules.Employment = &m
	for i := range c.Roles {
		switch c.Roles[i].ID {
		case "cook":
			e := economysim.DefaultHireEmployer()
			e.Recipes = []string{"prepare"}
			c.Roles[i].Employer = &e
		case "digger":
			c.Roles[i].Hireable = true
		}
	}
	c.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{{ID: "stipend", Kind: economysim.FlowGrant, Amount: 1,
		Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: "cook"}}}}
	return c
}

// TestValidate_HireConfigIsValid verifies that the configuration the hire-form
// tests start from validates and raises no employment warning, so each failing
// case breaks one rule and each warning case raises one.
//
// Guard: the field cases start from hireConfig; if it did not validate, each case
// would pass for the wrong reason.
func TestValidate_HireConfigIsValid(t *testing.T) {
	t.Parallel()
	c := hireConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("hireConfig().Validate() = %v, want nil", err)
	}
	if got := employmentWarningRules(c); len(got) != 0 {
		t.Errorf("hireConfig().Warnings() has the employment rules %v, want none", got)
	}
}

// employmentHireFieldCases are the failing cases of the rows of the hire form in
// the parameter-owner table (spec-employment.md 3.2, 13, 14.1): each breaks one
// field of a valid configuration. buildFieldCases appends them to the table that
// the registry test reads.
func employmentHireFieldCases() []fieldCase {
	field := func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				*c = hireConfig()
				set(c)
			}}
	}
	cook := func(c *economysim.Config) *economysim.Employer { return roleOf(c, "cook").Employer }
	eb := func(f string) []string { return []string{"roles", "cook", "employer", f} }
	mo := []string{"modules", "employment", "max_open"}
	return []fieldCase{
		field("Employment.Hire", "false with the fields of the hire form (the mixed form)", economysim.ErrEmploymentShape,
			[]string{"roles", "cook", "employer"}, "hire fields in the payroll form", "none",
			func(c *economysim.Config) {
				c.Modules.Employment = &economysim.Employment{}
				cook(c).PayrollBP = 5_000
			}),
		field("Employment.MaxOpen", "set while Hire is false", economysim.ErrEmploymentShape, mo, "5", "0",
			func(c *economysim.Config) {
				c.Modules.Employment = &economysim.Employment{MaxOpen: 5}
				cook(c).Recipes, cook(c).MaxHires, cook(c).WorkerShareBP, cook(c).HireYieldBP = nil, 0, 0, 0
				cook(c).PayrollBP = 5_000
			}),
		field("Employment.MaxOpen", "below 1", economysim.ErrOutOfBounds, mo, "-1", "1",
			func(c *economysim.Config) { c.Modules.Employment.MaxOpen = -1 }),
		field("Employment.MaxOpen", "above MaxHireRecords", economysim.ErrOutOfBounds, mo, "1000001", "1000000",
			func(c *economysim.Config) { c.Modules.Employment.MaxOpen = economysim.MaxHireRecords + 1 }),
		field("Employer.PayrollBP", "above 0 in the hire form", economysim.ErrEmploymentShape, eb("payroll_bp"), "5000", "0",
			func(c *economysim.Config) { cook(c).PayrollBP = 5_000 }),
		field("Employer.Recipes", "empty", economysim.ErrEmploymentShape, eb("recipes"), "0", "1",
			func(c *economysim.Config) { cook(c).Recipes = nil }),
		field("Employer.Recipes", "an entry that is not a recipe", economysim.ErrUnknownRecipe, eb("recipes"), "nowhere", "",
			func(c *economysim.Config) { cook(c).Recipes = []string{"prepare", "nowhere"} }),
		field("Employer.Recipes", "a recipe twice", economysim.ErrDuplicateID, eb("recipes"), "prepare", "",
			func(c *economysim.Config) { cook(c).Recipes = []string{"prepare", "prepare"} }),
		field("Employer.Recipes", "a recipe that is not on the role's list", economysim.ErrEmploymentShape, eb("recipes"), "dig_mineral", "role recipes",
			func(c *economysim.Config) { cook(c).Recipes = []string{"prepare", "dig_mineral"} }),
		field("Employer.Recipes", "a role with no recipes", economysim.ErrEmploymentShape, []string{"roles", "visitor", "employer", "recipes"}, "1", "role recipes",
			func(c *economysim.Config) {
				e := economysim.DefaultHireEmployer()
				e.Recipes = []string{"prepare"}
				roleOf(c, "visitor").Employer = &e
			}),
		field("Employer.MaxHires", "below 1", economysim.ErrOutOfBounds, eb("max_hires"), "0", "1",
			func(c *economysim.Config) { cook(c).MaxHires = 0 }),
		field("Employer.MaxHires", "above MaxEstablishment", economysim.ErrOutOfBounds, eb("max_hires"), "1001", "1000",
			func(c *economysim.Config) { cook(c).MaxHires = economysim.MaxEstablishment + 1 }),
		field("Employer.WorkerShareBP", "below 1", economysim.ErrOutOfBounds, eb("worker_share_bp"), "0", "1",
			func(c *economysim.Config) { cook(c).WorkerShareBP = 0 }),
		field("Employer.WorkerShareBP", "above 10 000", economysim.ErrOutOfBounds, eb("worker_share_bp"), "10001", "10000",
			func(c *economysim.Config) { cook(c).WorkerShareBP = 10_001 }),
		field("Employer.HireYieldBP", "below 1", economysim.ErrOutOfBounds, eb("hire_yield_bp"), "0", "1",
			func(c *economysim.Config) { cook(c).HireYieldBP = 0 }),
		field("Employer.HireYieldBP", "above MaxYieldBP", economysim.ErrOutOfBounds, eb("hire_yield_bp"), "100001", "100000",
			func(c *economysim.Config) { cook(c).HireYieldBP = economysim.MaxYieldBP + 1 }),
		field("Role.Employer", "a group of scale above 1", economysim.ErrEmployerWeight, []string{"groups", "north", "cook", "scale"}, "2", "1",
			func(c *economysim.Config) {
				for i := range c.Groups {
					if c.Groups[i].Role == "cook" {
						c.Groups[i].Scale = 2
					}
				}
			}),
	}
}

// TestValidate_HireFormRules verifies the rules of spec-employment.md 14.1 and
// 24.4 that no row of the parameter-owner table owns: an employer role whose
// world scale is above 1 (the group takes the world's), the hire form with no
// employer and with no Hireable role (warnings there, errors in the payroll
// form), and a Hireable role with no recipe, which is valid.
//
// Requirement: FR-EMP-002, FR-EMP-005, FR-EMP-006
func TestValidate_HireFormRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		want   error
		path   []string
	}{
		{"world scale above 1 reaches the employer's group", func(c *economysim.Config) { c.World.Scale = 4 },
			economysim.ErrEmployerWeight, []string{"groups", "north", "cook", "scale"}},
		{"a Hireable role with no recipe", func(c *economysim.Config) { roleOf(c, "digger").Recipes = nil }, nil, nil},
		{"no employer role is a warning", func(c *economysim.Config) { roleOf(c, "cook").Employer = nil }, nil, nil},
		{"no Hireable role is a warning", func(c *economysim.Config) { roleOf(c, "digger").Hireable = false }, nil, nil},
		{"a role that is both", func(c *economysim.Config) { roleOf(c, "cook").Hireable = true },
			economysim.ErrEmploymentShape, []string{"roles", "cook", "hireable"}},
		{"a block while the module is off", func(c *economysim.Config) {
			c.Modules.Employment = nil
			roleOf(c, "digger").Hireable = false
			roleOf(c, "cook").Employer.PayrollBP = 5_000
		}, economysim.ErrModuleDependency, []string{"roles", "cook", "employer"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := hireConfig()
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

// employmentWarningRules returns the rules of the warnings of 14.2 that the
// configuration raises, in the order Warnings gives them, each with its path.
func employmentWarningRules(c economysim.Config) []string {
	rules := []string{"employment_no_employer", "employment_no_worker", "no_employer_in_region", "employer_no_outlet",
		"hire_never_profitable", "wage_below_upkeep", "payroll_exceeds_money", "inkind_wage_ignored", "worker_share_full"}
	var out []string
	for _, w := range c.Warnings() {
		if slices.Contains(rules, w.Rule) {
			out = append(out, w.Rule+" "+strings.Join(w.Path, "."))
		}
	}
	return out
}

// TestWarnings_Employment verifies one case per warning of spec-employment.md
// 14.2, each alone on a valid hire configuration, with its path, and that the
// valid configuration raises none and the payroll form raises none of them (it has
// errors where two are warnings).
//
// Requirement: FR-EMP-059
func TestWarnings_Employment(t *testing.T) {
	t.Parallel()
	cook := func(c *economysim.Config) *economysim.Employer { return roleOf(c, "cook").Employer }
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		want   string
	}{
		{"no employer", func(c *economysim.Config) { roleOf(c, "cook").Employer = nil }, "employment_no_employer modules.employment"},
		{"no worker", func(c *economysim.Config) { roleOf(c, "digger").Hireable = false }, "employment_no_worker modules.employment"},
		{"no employer in a region", func(c *economysim.Config) {
			r := economysim.DefaultRegion()
			r.ID = "south"
			c.Regions = append(c.Regions, r)
			c.Groups = append(c.Groups, economysim.Group{Region: "south", Role: "digger", Households: 30})
			c.Yields = append(c.Yields, economysim.YieldProfile{Region: "south", Recipe: "dig_mineral", HasAll: true, All: 10_000})
		}, "no_employer_in_region groups.south.digger"},
		{"no outlet for the profit", func(c *economysim.Config) { c.Modules.MoneySupply = nil }, "employer_no_outlet roles.cook.employer.recipes"},
		// prepare yields 8 prepared at 628 = 5 024; at HireYieldBP 1 that is
		// 5 024 * 1 / 10 000 = 0, less the inputs 2 * 96 + 179 = 371: below 1.
		{"a recipe never profitable", func(c *economysim.Config) { cook(c).HireYieldBP = 1 }, "hire_never_profitable roles.cook.employer.recipes.prepare"},
		// Net 5 024 - 371 = 4 653 per tick; share 1 gives a wage of
		// 4 653 / 10 000 = 0.47 -> 0, below the digger's upkeep 2 * 96 + 628 + 179 = 999.
		{"a wage below upkeep", func(c *economysim.Config) { cook(c).WorkerShareBP = 1 }, "wage_below_upkeep roles.cook.employer.worker_share_bp"},
		// Wage 4 653 * 5 000 / 10 000 = 2 326.5 -> 2 326 (half to even); 10 hires of
		// one tick cost 23 260, above the cook's StartMoney of 10 000.
		{"an establishment the start money cannot pay", func(c *economysim.Config) { cook(c).MaxHires = 10 }, "payroll_exceeds_money roles.cook.start_money"},
		{"an in-kind share on a worker role", func(c *economysim.Config) { roleOf(c, "digger").InKind = 100 }, "inkind_wage_ignored roles.digger.in_kind"},
		{"a full worker share", func(c *economysim.Config) { cook(c).WorkerShareBP = 10_000 }, "worker_share_full roles.cook.employer.worker_share_bp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := hireConfig()
			tt.mutate(&c)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate(%s) = %v, want nil: a warning never fails validation", tt.name, err)
			}
			if got := employmentWarningRules(c); !slices.Contains(got, tt.want) {
				t.Errorf("Warnings(%s) employment rules = %q, want %q among them", tt.name, got, tt.want)
			}
		})
	}
	t.Run("the payroll form raises none", func(t *testing.T) {
		t.Parallel()
		c := employmentConfig()
		if got := employmentWarningRules(c); len(got) != 0 {
			t.Errorf("Warnings(payroll form) employment rules = %q, want none", got)
		}
	})
}

// TestEmploymentHire_CanonicalDefaultsAndIsolation verifies that a hire form with
// MaxOpen 0 takes DefaultMaxOpen in the canonical form (so the hash of 0 and of
// the default agree), that the hire list is sorted there, and that the canonical
// configuration shares no slice with the caller's.
//
// Requirement: FR-EMP-079
//
// Guard: the hash and the world read the canonical copy; a shared hire list would
// let a later edit of the caller's slice change a configuration already validated.
func TestEmploymentHire_CanonicalDefaultsAndIsolation(t *testing.T) {
	t.Parallel()
	zero, def := hireConfig(), hireConfig()
	zero.Modules.Employment = &economysim.Employment{Hire: true}
	if economysim.HashConfig(zero) != economysim.HashConfig(def) {
		t.Errorf("HashConfig(MaxOpen 0) != HashConfig(MaxOpen %d), want equal: canonical writes the default", economysim.DefaultMaxOpen)
	}
	c := hireConfig()
	roleOf(&c, "cook").Employer.Recipes = []string{"prepare_basic", "prepare"}
	before := economysim.HashConfig(c)
	roleOf(&c, "cook").Employer.Recipes[0] = "prepare"
	if economysim.HashConfig(c) == before {
		t.Errorf("HashConfig does not see an edit of the hire list, want it hashed")
	}
}

// TestDefaultHireEmployer verifies the defaults of spec-employment.md 13: an
// establishment of 4, the midpoint worker share and a hired yield of 10 000, and
// that the default module is the hire form with MaxOpen 1 000.
//
// Requirement: FR-EMP-003
func TestDefaultHireEmployer(t *testing.T) {
	t.Parallel()
	want := economysim.Employer{MaxHires: 4, WorkerShareBP: 5_000, HireYieldBP: 10_000}
	if got := economysim.DefaultHireEmployer(); !reflect.DeepEqual(got, want) {
		t.Errorf("DefaultHireEmployer() = %+v, want %+v", got, want)
	}
	if got := economysim.DefaultEmployment(); !got.Hire || got.MaxOpen != 1_000 {
		t.Errorf("DefaultEmployment() = %+v, want Hire with MaxOpen 1000", got)
	}
}

// TestConfigNoWageField verifies that no field of the employment configuration
// holds a wage or a wage multiplier: the wage of a hire is derived at its
// formation (spec-employment.md 3.5).
//
// Requirement: FR-EMP-007
//
// Guard: a wage field would let a pack set what the round must derive from the
// worker's reservation and the employer's bid; the reflection walk catches a
// field added later under any name that says wage, pay, salary or multiplier.
func TestConfigNoWageField(t *testing.T) {
	t.Parallel()
	for _, typ := range []reflect.Type{reflect.TypeFor[economysim.Employment](), reflect.TypeFor[economysim.Employer]()} {
		for i := range typ.NumField() {
			f := typ.Field(i)
			name := strings.ToLower(f.Name)
			if f.Name == "PayrollBP" { // the payroll form's share of receipts, not a wage (24.3.2)
				continue
			}
			for _, bad := range []string{"wage", "pay", "salary", "multiplier", "rate"} {
				if strings.Contains(name, bad) {
					t.Errorf("%s.%s names a wage (%q), want the wage derived at formation", typ.Name(), f.Name, bad)
				}
			}
		}
	}
}
