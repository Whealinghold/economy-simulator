package economysim_test

import (
	"errors"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// FuzzValidateEmployment drives the fields of both forms of the employment module
// through Validate and Warnings. It asserts that nothing panics, that the two calls
// agree on a second run, that every error is a *ValidationError, that a
// configuration that validates raises only warnings that name a path, and that one
// that validates hashes the same twice.
//
// Requirement: FR-EMP-002, FR-EMP-003, FR-EMP-004, FR-EMP-059
func FuzzValidateEmployment(f *testing.F) {
	f.Add(true, int64(1_000), int64(0), int64(4), int64(5_000), int64(10_000), "prepare", "prepare")
	f.Add(false, int64(0), int64(5_000), int64(0), int64(0), int64(0), "", "")
	f.Add(true, int64(-1), int64(5_000), int64(1_001), int64(0), int64(100_001), "nowhere", "dig_mineral")
	f.Add(true, int64(1<<62), int64(-1), int64(1<<62), int64(1<<62), int64(1<<62), "prepare", "prepare_basic")
	f.Fuzz(func(t *testing.T, hire bool, maxOpen, payrollBP, maxHires, share, yield int64, r1, r2 string) {
		c := hireConfig()
		c.Modules.Employment = &economysim.Employment{Hire: hire, MaxOpen: maxOpen}
		e := roleOf(&c, "cook").Employer
		e.PayrollBP, e.MaxHires, e.WorkerShareBP, e.HireYieldBP = payrollBP, maxHires, share, yield
		e.Recipes = []string{r1, r2}
		err := c.Validate()
		var ve *economysim.ValidationError
		if err != nil && !errors.As(err, &ve) {
			t.Fatalf("Validate() = %v, want a *ValidationError or nil", err)
		}
		w := c.Warnings()
		if again := c.Warnings(); len(again) != len(w) {
			t.Fatalf("Warnings() gave %d then %d, want the same", len(w), len(again))
		}
		if err == nil {
			for _, x := range w {
				if len(x.Path) == 0 || x.Rule == "" {
					t.Fatalf("Warnings() = %+v, want a rule and a path", x)
				}
			}
			first := economysim.HashConfig(c)
			if again := economysim.HashConfig(c); first != again {
				t.Fatalf("HashConfig() of a valid configuration gave %x then %x, want the same", first, again)
			}
		}
	})
}
