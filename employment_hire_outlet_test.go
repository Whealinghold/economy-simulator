package economysim_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestWarnings_EmployerOutlet verifies what counts as an outlet for an
// employer's profit (spec-employment.md 11.3.1, 14.2 employer_no_outlet): a
// money flow to the households of the employer role, or to the households of
// every role, does; a flow to another role or to a target that is not households
// does not.
//
// Requirement: FR-EMP-059
//
// Guard: the outlet test reads the flow's role and kind; if it accepted any flow,
// a pack could pay its employers' profit into nothing and raise no warning.
func TestWarnings_EmployerOutlet(t *testing.T) {
	t.Parallel()
	const rule = "employer_no_outlet roles.cook.employer.recipes"
	flow := func(role string) func(c *economysim.Config) {
		return func(c *economysim.Config) {
			c.Modules.MoneySupply.Flows[0].Target = economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: role}
		}
	}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		want   bool
	}{
		{"a flow to the employer role's households", flow("cook"), false},
		{"a flow to the households of every role (no role named)", flow(""), false},
		{"a flow to another role's households", flow("digger"), true},
		{"no money supply module", func(c *economysim.Config) { c.Modules.MoneySupply = nil }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := hireConfig()
			tt.mutate(&c)
			if got := slices.Contains(employmentWarningRules(c), rule); got != tt.want {
				t.Errorf("Warnings(%s) has %q = %v, want %v", tt.name, rule, got, tt.want)
			}
		})
	}
}

// TestHirePack_TasteIsTheOutlet verifies, on the pack one-demand with the hire
// overlay, that a taste row with a scale above 0 is the cooks' outlet and that
// without one, or at scale 0, employer_no_outlet is raised for the cooks.
//
// Requirement: FR-EMP-059
func TestHirePack_TasteIsTheOutlet(t *testing.T) {
	t.Parallel()
	const rule = "employer_no_outlet roles.cook.employer.recipes"
	tests := []struct {
		name   string
		mutate func(tastes []economysim.Taste) []economysim.Taste
		want   bool
	}{
		{"the pack's tastes", func(ts []economysim.Taste) []economysim.Taste { return ts }, false},
		{"no cook taste", func(ts []economysim.Taste) []economysim.Taste {
			return slices.DeleteFunc(ts, func(x economysim.Taste) bool { return x.Role == "cook" })
		}, true},
		{"cook tastes at scale 0", func(ts []economysim.Taste) []economysim.Taste {
			for i := range ts {
				if ts[i].Role == "cook" {
					ts[i].Scale = 0
				}
			}
			return ts
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := loadPack(t, packOneDemandHire...)
			d := *cfg.Modules.Demand
			d.Tastes = tt.mutate(slices.Clone(d.Tastes))
			cfg.Modules.Demand = &d
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate(%s) = %v, want nil", tt.name, err)
			}
			if got := slices.Contains(employmentWarningRules(cfg), rule); got != tt.want {
				t.Errorf("Warnings(%s) employment rules = %q, want %q present = %v",
					tt.name, employmentWarningRules(cfg), rule, tt.want)
			}
		})
	}
}

// TestValidate_HireableNeedsTheModule verifies that a Hireable role with the
// employment module absent is ErrModuleDependency at the role's path.
//
// Requirement: FR-EMP-002
func TestValidate_HireableNeedsTheModule(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	roleOf(&c, "digger").Hireable = true
	err := c.Validate()
	vs, _ := violations(err)
	if !errors.Is(err, economysim.ErrModuleDependency) || len(vs) != 1 || !slices.Equal(vs[0].Path, []string{"roles", "digger", "hireable"}) {
		t.Errorf("Validate(Hireable with no module) = %v, want only ErrModuleDependency at roles.digger.hireable", err)
	}
}

// TestHirePack_DefaultStrategyHires is the config-level integration test of the overlay:
// one-demand with the hire form, run with the default strategy and the audit, forms
// hires (the default's offers and vacancies meet), while the same seed on one-demand
// alone forms none and the two configurations hash differently. Every hire's wage lies
// in [Reserve, Bid] and the audit holds at every tick.
//
// Requirement: FR-EMP-013, FR-EMP-015, FR-EMP-021
//
// Guard: before the default strategy offered and posted (plan 3-emp, Phase 6) the form
// was inert at run time; this pins that the form now acts, and only with the form.
func TestHirePack_DefaultStrategyHires(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	base := loadPack(t, packOneDemand...)
	hire := loadPack(t, packOneDemandHire...)
	if economysim.HashConfig(base) == economysim.HashConfig(hire) {
		t.Fatalf("HashConfig(one-demand) == HashConfig(one-demand + hire), want different")
	}
	run := func(cfg economysim.Config) (formed int) {
		w, err := economysim.NewWorld(cfg, economysim.WithSeed(1), economysim.WithAudit(),
			economysim.WithObserver(economysim.ObserverFunc(func(e economysim.Event) {
				if h, ok := e.(economysim.HireFormed); ok {
					formed++
					if int64(h.Wage) < int64(h.Reserve) || int64(h.Wage) > int64(h.Bid) {
						t.Errorf("HireFormed %d: Wage = %d, want within [Reserve %d, Bid %d]", h.Hire, h.Wage, h.Reserve, h.Bid)
					}
				}
			})))
		if err != nil {
			t.Fatalf("NewWorld() = %v, want nil", err)
		}
		for range 10 {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() = %v, want nil", err)
			}
		}
		return formed
	}
	if got := run(base); got != 0 {
		t.Errorf("hires formed on one-demand = %d, want 0 (no employment module)", got)
	}
	if got := run(hire); got == 0 {
		t.Errorf("hires formed on one-demand + hire overlay = 0, want some: the default strategy offers and posts")
	}
}
