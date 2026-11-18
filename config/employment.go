package config

import economysim "github.com/Whealinghold/economy-sim"

// The keys of the employment module (spec-employment.md 24.5.2, 13), in a file of
// their own so that mapping.go does not grow.

// employment reads [modules.employment] (spec-employment.md 24.5.2, 13): hire
// selects the hire form, and max_open is read over DefaultEmployment's value when
// hire is true. The table with no key is the payroll form. A max_open without hire
// is kept and left to the validator (ErrEmploymentShape).
func (l *loader) employment(t *tbl) *economysim.Employment {
	var m economysim.Employment
	t.flag("hire", &m.Hire)
	if m.Hire {
		m.MaxOpen = economysim.DefaultEmployment().MaxOpen
	}
	integer(t, "max_open", &m.MaxOpen)
	return &m
}

// employer reads [role.<id>.employer]: over DefaultHireEmployer when any key of
// the hire form is present, else over DefaultEmployer (the payroll form). The keys
// say the form, so the order the tables are read in does not matter; the validator
// refuses the mixed form.
func employer(t *tbl) economysim.Employer {
	e := economysim.DefaultEmployer()
	if t.has("recipes", "max_hires", "worker_share_bp", "hire_yield_bp") {
		e = economysim.DefaultHireEmployer()
	}
	integer(t, "payroll_bp", &e.PayrollBP)
	t.strings("recipes", &e.Recipes)
	readInts(t, []intField{
		{"max_hires", &e.MaxHires}, {"worker_share_bp", &e.WorkerShareBP}, {"hire_yield_bp", &e.HireYieldBP},
	})
	return e
}
