package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packSecondHire is pack second, the crater colony, with the employment module in its hire form.
var packSecondHire = append(append([]string(nil), packSecond...), "second/employment-hire.toml")

// TestConfigOnly_EmploymentHireNeedsOnlyConfiguration verifies architecture.md 9.4 for the hire
// form: pack second, a setting that shares nothing with the reference chain, takes an employer
// role with a hire list and a Hireable role from a document alone, loads, validates, raises no
// warning of the employment module but the informational employer_no_outlet, runs 100 ticks for seeds 1 and 2 with the audit on, forms
// hires with the default strategy, and every hire conforms to its bill. The Go of the test names
// no commodity, role or region of the module.
//
// Requirement: FR-EMP-002, FR-EMP-003, FR-EMP-013, FR-EMP-024, FR-EMP-036, FR-EMP-059
//
// Guard: a branch on a commodity, recipe or role identifier in the module would fail on a
// setting it never heard of; the probe fails the test if no hire forms, so it cannot pass for
// nothing.
func TestConfigOnly_EmploymentHireNeedsOnlyConfiguration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, packSecondHire...)
	if m := cfg.Modules.Employment; m == nil || !m.Hire {
		t.Fatalf("Modules.Employment = %+v, want the hire form", m)
	}
	// The one warning left is the informational one: the colony has no demand module, so the employers'
	// profit has no outlet but their upkeep (spec-employment.md 11.3.1, a world a pack may want).
	if got := employmentWarningRules(cfg); len(got) != 1 || got[0] != "employer_no_outlet roles.smelter.employer.recipes" {
		t.Errorf("Warnings(second + hire document) employment rules = %q, want only employer_no_outlet", got)
	}
	for _, seed := range seedList(min(2, scenarioSeedCount())) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			tape := newLabourTape()
			w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(tape))
			if err != nil {
				t.Fatalf("NewWorld(second + hire document, seed %d) = %v, want nil", seed, err)
			}
			for range 100 {
				if err := w.Step(); err != nil {
					t.Fatalf("Step() at tick %d (seed %d) = %v, want nil", w.Tick(), seed, err)
				}
			}
			if tape.formed == 0 {
				t.Errorf("seed %d: no hire formed in 100 ticks, want some (the test would pass for nothing)", seed)
			}
			if p := tape.firstProblem(); p != "" {
				t.Errorf("seed %d: %s", seed, p)
			}
		})
	}
}
