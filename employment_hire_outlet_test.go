package economysim_test

import (
	"errors"
	"reflect"
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

// TestHirePack_FormIsInertAtRunTime is the config-level integration test of the
// overlay: the same seed on one-demand and on one-demand with the hire form gives
// the same money, prices and population after 3 steps, while the two
// configurations hash differently. The hire form is data in this PR; a hook that
// changed the stream of a world with the form on would show here.
//
// Requirement: FR-EMP-079
//
// Guard: the form must not change a run until the round exists (plan 3-emp,
// Phase 3 exit check); a payroll hook that fired for the hire form would move
// the money.
func TestHirePack_FormIsInertAtRunTime(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	base := loadPack(t, packOneDemand...)
	hire := loadPack(t, packOneDemandHire...)
	if economysim.HashConfig(base) == economysim.HashConfig(hire) {
		t.Fatalf("HashConfig(one-demand) == HashConfig(one-demand + hire), want different")
	}
	run := func(cfg economysim.Config) economysim.Snapshot {
		w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
		if err != nil {
			t.Fatalf("NewWorld() = %v, want nil", err)
		}
		for range 3 {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() = %v, want nil", err)
			}
		}
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		return s
	}
	a, b := run(base), run(hire)
	if a.ConfigHash == b.ConfigHash {
		t.Errorf("Snapshot.ConfigHash is the same with and without the hire form, want it to see the form")
	}
	a.ConfigHash = b.ConfigHash // the one field that must differ
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Snapshot(hire form) != Snapshot(off) after 3 steps, want equal but for ConfigHash: the form is data only")
	}
}
