package economysim

// This file is the employment module in its payroll form (spec-employment.md 24,
// EMP; ADR 0011): after clearing, an agent of an employer role pays a share of
// its receipts of the tick to the households of the Hireable roles that
// worked. It holds the module's state, its validator and, with the position,
// its rule; the arithmetic is internal/employment, and this file converts at the
// call (architecture.md 2.9, 2.10.1). The module has no hire, no escrow, no
// per-agent state and no checkpoint section.

// employment is the state of the module: the roles it reads, by RoleID. It is
// derived from the configuration and saved nowhere (spec-employment.md 24.1.7).
// It is nil when the module is off.
type employment struct {
	// payrollBP is the PayrollBP of each role with an employer block, 0 for a
	// role with none.
	payrollBP []int64
	// hireable says the role is Hireable.
	hireable []bool
}

// newEmployment builds the state for a canonical configuration that passed
// Validate: nil when the module is off.
func newEmployment(c *Config) *employment {
	if c.Modules.Employment == nil {
		return nil
	}
	e := &employment{payrollBP: make([]int64, len(c.Roles)), hireable: make([]bool, len(c.Roles))}
	for i := range c.Roles {
		e.hireable[i] = c.Roles[i].Hireable
		if b := c.Roles[i].Employer; b != nil {
			e.payrollBP[i] = b.PayrollBP
		}
	}
	return e
}

// validateEmployment checks the module and the role fields of the payroll form
// (spec-employment.md 24.4.1): the bound of PayrollBP, the module dependency of
// Hireable and of the block, a role that is both, and the module on with no
// payer role or no Hireable role (a no-op that would look like a result).
func (v *validator) validateEmployment() {
	c := &v.c
	on := c.Modules.Employment != nil
	var payers, workers int
	for i := range c.Roles {
		r := &c.Roles[i]
		v.employmentRole([]string{"roles", r.ID}, r, on)
		if r.Employer != nil {
			payers++
		}
		if r.Hireable {
			workers++
		}
	}
	if !on {
		return
	}
	base := []string{"modules", "employment"}
	if payers == 0 {
		v.rep.add(ErrEmploymentShape, path(base, "employer"), "0", "1")
	}
	if workers == 0 {
		v.rep.add(ErrEmploymentShape, path(base, "hireable"), "0", "1")
	}
}

// employmentRole checks the employment fields of one role; on says the module
// is present.
func (v *validator) employmentRole(rb []string, r *Role, on bool) {
	if r.Employer != nil {
		eb := path(rb, "employer")
		if !on {
			v.rep.add(ErrModuleDependency, eb, "employment", "")
		}
		v.inRange(path(eb, "payroll_bp"), r.Employer.PayrollBP, 1, wholeBP)
	}
	if r.Hireable && !on {
		v.rep.add(ErrModuleDependency, path(rb, "hireable"), "employment", "")
	}
	if r.Hireable && r.Employer != nil {
		v.rep.add(ErrEmploymentShape, path(rb, "hireable"), "true", "false")
	}
}
