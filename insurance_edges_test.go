package economysim_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The edge tests of the insurance module's configuration surface: the numbers the
// spec fixes (13), the cases in which a warning of 14.2 must stay silent, the
// dependencies a block does not have, and a world built with the module on that runs
// like one with it off, because no position of the module exists yet.

// TestInsurance_BoundsAndDefaultsMatchSpec verifies the seven bounds of spec-insurance.md
// 13 and the two defaults of DefaultInsurance by value: the bound tests elsewhere read the
// constants, so a constant that drifted would move the test with it.
//
// Requirement: FR-INS-010, FR-CFG-091
//
// Guard: the bounds are placeholders (N-015) and the premium product fits 128 bits only
// at these values (spec 6.4.1); a change to one is a decision, not a side effect.
func TestInsurance_BoundsAndDefaultsMatchSpec(t *testing.T) {
	t.Parallel()
	for _, b := range []struct {
		name      string
		got, want int64
	}{
		{"MaxValueCapBP", economysim.MaxValueCapBP, 100_000},
		{"MaxLoadingBound", economysim.MaxLoadingBound, 100_000},
		{"MaxStoreRateBound", economysim.MaxStoreRateBound, 10_000},
		{"MaxCapBound", economysim.MaxCapBound, 100_000},
		{"MaxRuinBound", economysim.MaxRuinBound, 100_000},
		{"MaxOpenPolicies", economysim.MaxOpenPolicies, 1_000_000},
		{"MaxCoverRequestsBound", economysim.MaxCoverRequestsBound, 64},
	} {
		if b.got != b.want {
			t.Errorf("%s = %d, want %d (spec-insurance.md 13)", b.name, b.got, b.want)
		}
	}
	want := economysim.Insurance{PayoutTicks: 1, MaxCoverRequests: 4}
	if got := economysim.DefaultInsurance(); got != want {
		t.Errorf("DefaultInsurance() = %+v, want %+v: only PayoutTicks and MaxCoverRequests have a default", got, want)
	}
}

// TestDefaultInsurance_NeedsThreeRequiredFields verifies that the module as
// DefaultInsurance returns it does not validate until the three fields with no default
// are set, that Validate names exactly those three, and that setting them makes the
// configuration valid with the two defaults kept.
//
// Requirement: FR-INS-010, FR-INS-011
func TestDefaultInsurance_NeedsThreeRequiredFields(t *testing.T) {
	t.Parallel()
	c := insuranceConfig()
	m := economysim.DefaultInsurance()
	c.Modules.Insurance = &m
	err := c.Validate()
	var got []string
	for _, e := range unwrapAll(err) {
		var ve *economysim.ValidationError
		if errors.As(e, &ve) && errors.Is(e, economysim.ErrOutOfBounds) {
			got = append(got, strings.Join(ve.Path, "."))
		}
	}
	slices.Sort(got)
	want := []string{"modules.insurance.max_policies", "modules.insurance.min_sum", "modules.insurance.value_cap_bp"}
	if !slices.Equal(got, want) {
		t.Errorf("Validate(DefaultInsurance()) out-of-bounds paths = %v, want %v", got, want)
	}
	m.ValueCapBP, m.MinSum, m.MaxPolicies = 20_000, 100, 1_000
	if err := c.Validate(); err != nil {
		t.Errorf("Validate(DefaultInsurance() with the three required fields) = %v, want nil", err)
	}
}

// TestWarnings_InsuranceStaysSilent verifies, for each warning of spec-insurance.md 14.2
// that has an edge, the nearest configuration that must not raise it: the value on the
// edge, the other half of an either-or and the case the warning is defined away from.
// TestWarnings_Insurance holds the configuration that raises each; this holds the other
// side, so a warning that fires one step early or on a different case is caught.
//
// Requirement: FR-INS-010, FR-CFG-091
func TestWarnings_InsuranceStaysSilent(t *testing.T) {
	t.Parallel()
	pool := func(c *economysim.Config, role string) {
		c.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{
			{ID: "stipend", Kind: economysim.FlowGrant, Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: role}, Amount: 1},
		}}
	}
	tests := []struct {
		name string
		set  func(c *economysim.Config)
		not  string
	}{
		{"a cap of exactly the whole money", func(c *economysim.Config) {
			r := insuranceRole(c, "cook")
			r.StartStock = nil
			r.Cover.Underwrite.CapBP = 10_000
		}, "underwriter_cap_above_money"},
		{"a cap above the money with a start stock", func(c *economysim.Config) {
			insuranceRole(c, "cook").Cover.Underwrite.CapBP = 20_000
		}, "underwriter_cap_above_money"},
		{"no recipes but a flow to the role", func(c *economysim.Config) {
			insuranceRole(c, "cook").Recipes = nil
			pool(c, "cook")
		}, "underwriter_without_income"},
		{"as many claims as policies", func(c *economysim.Config) {
			c.Modules.Credit.MaxClaims = c.Modules.Insurance.MaxPolicies
		}, "claims_below_policies"},
		{"no shield and no underwriter", func(c *economysim.Config) {
			c.Modules.Credit.ShieldTicks = 0
			insuranceRole(c, "cook").Cover = nil
		}, "no_shield_underwriter"},
		{"the value cap one below its bound", func(c *economysim.Config) {
			c.Modules.Insurance.ValueCapBP = economysim.MaxValueCapBP - 1
		}, "value_cap_unbound"},
		{"a minimum of exactly the start money", func(c *economysim.Config) {
			c.Modules.Insurance.MinSum = int64(insuranceRole(c, "grower").StartMoney)
		}, "cover_below_minimum"},
		{"a loading equal to the highest limit", func(c *economysim.Config) {
			insuranceRole(c, "cook").Cover.Underwrite.LoadingBP = insuranceRole(c, "grower").Cover.Insure.MaxLoadingBP
		}, "loading_above_limits"},
		{"a second underwriter below the limit", func(c *economysim.Config) {
			second := *insuranceRole(c, "cook")
			second.ID = "second"
			second.Cover = &economysim.CoverTerms{Underwrite: &economysim.UnderwriteTerms{LoadingBP: 20_000, StoreRateBP: 50, CoverBP: 8_000, CapBP: 5_000, ShareBP: 5_000}}
			c.Roles = append(c.Roles, second)
			insuranceRole(c, "cook").Cover.Underwrite.LoadingBP = 10_000
		}, "loading_above_limits"},
		{"no route cover and no route row", func(c *economysim.Config) {
			c.Modules.Risk.Routes = nil
			insuranceRole(c, "grower").Cover.Insure.MinLossBP = 0
		}, "cover_without_risk"},
		{"no store cover and no store row", func(c *economysim.Config) {
			c.Modules.Risk.Stores = nil
			insuranceRole(c, "grower").Cover.Insure.StoreShareBP = 0
		}, "cover_without_risk"},
		{"both sets empty is one warning", func(c *economysim.Config) {
			insuranceRole(c, "grower").Cover, insuranceRole(c, "cook").Cover = nil, nil
		}, "cover_no_underwriter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := insuranceConfig()
			tt.set(&c)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate(%s) = %v, want nil", tt.name, err)
			}
			if got := insuranceWarningRules(c); got[tt.not] {
				t.Errorf("Warnings(%s) has the insurance warnings %v, want no %s", tt.name, got, tt.not)
			}
		})
	}
}

// TestWarnings_InsuranceIncomeFlowIsOfTheRole verifies that an underwriter with no
// recipes is without income unless a money flow pays the households of its own role: a
// flow to another role leaves the warning, a flow to the role removes it.
//
// Requirement: FR-INS-010, FR-CFG-091
func TestWarnings_InsuranceIncomeFlowIsOfTheRole(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		role string
		want bool
	}{{"a flow to another role", "grower", true}, {"a flow to the role", "cook", false}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := insuranceConfig()
			insuranceRole(&c, "cook").Recipes = nil
			c.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{
				{ID: "stipend", Kind: economysim.FlowGrant, Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: tc.role}, Amount: 1},
			}}
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate(%s) = %v, want nil", tc.name, err)
			}
			if got := insuranceWarningRules(c)["underwriter_without_income"]; got != tc.want {
				t.Errorf("Warnings(%s) underwriter_without_income = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// TestWarnings_InsurancePaths verifies where each warning sits: the module-level ones
// at modules.insurance (and value_cap_unbound at its field), the underwriter's two at
// the role's cover block and the cap one at its field, so a consumer can find what the
// warning is about (observability-and-logging.md 3.2.4).
//
// Requirement: FR-INS-010, FR-CFG-091
func TestWarnings_InsurancePaths(t *testing.T) {
	t.Parallel()
	c := insuranceConfig()
	r := insuranceRole(&c, "cook")
	r.Recipes, r.StartStock = nil, nil
	r.Cover.Underwrite.CapBP = 20_000
	c.Modules.Insurance.ValueCapBP = economysim.MaxValueCapBP
	c.Modules.Credit.MaxClaims = 10
	got := map[string]string{}
	for _, w := range c.Warnings() {
		got[w.Rule] = strings.Join(w.Path, ".")
	}
	want := map[string]string{
		"underwriter_without_income":  "roles.cook.cover",
		"underwriter_cap_above_money": "roles.cook.cover.underwrite.cap_bp",
		"value_cap_unbound":           "modules.insurance.value_cap_bp",
		"claims_below_policies":       "modules.insurance",
	}
	for rule, path := range want {
		if got[rule] != path {
			t.Errorf("Warnings() rule %s at %q, want %q", rule, got[rule], path)
		}
	}
}

// TestValidateInsurance_DependenciesAreOfTheRequestOnly verifies spec-insurance.md 4.1.1
// from the other side: an underwriter's block needs neither Transport nor Storage, a
// route-only insured needs no Storage, and a store-only insured needs no Transport, so
// each is valid with the other module off.
//
// Requirement: FR-INS-010
func TestValidateInsurance_DependenciesAreOfTheRequestOnly(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		set  func(c *economysim.Config)
	}{
		{"a route-only insured with Storage off", func(c *economysim.Config) {
			insuranceRole(c, "grower").Cover.Insure.StoreShareBP = 0
			c.Modules.Storage, c.Modules.Risk.Stores = nil, nil
		}},
		{"an underwriter alone with Transport and Storage off", func(c *economysim.Config) {
			insuranceRole(c, "grower").Cover = nil
			c.Modules.Transport, c.Modules.Storage = nil, nil
			c.Modules.Risk.Routes, c.Modules.Risk.Stores = nil, nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := insuranceConfig()
			tt.set(&c)
			if err := c.Validate(); err != nil {
				t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
			}
		})
	}
}

// TestInsurance_OnWorldBuildsAndRunsLikeOff verifies, over three seeds, that a world
// built in Go with the insurance module on, an insured role and an underwriter role, is
// accepted by NewWorld, hashes differently from the same configuration with the module
// and the blocks removed, and then emits the same events but for the two of the cover round
// (CoverRound, CoverFormed) and ends in the same snapshot but for the name of the module in
// the list of modules. The module has its round at P3.3c since plan 3-ins PR C, but this pack
// has no shipment and no store offer (the insured role has no store share), so the round
// forms no policy and moves no money: only its own events differ. With the store offers
// the module is not neutral: TestInsurance_StoreOffersLeaveATrace pins that side.
//
// Requirement: FR-INS-001, FR-INS-005
//
// Guard: a module that is on changes a run only where it acts. If a later change let the
// blocks or the reasons of the module reach a draw, an event or a boundary row outside its
// own positions, this comparison would differ. The run must have events, or the comparison
// would pass for nothing.
func TestInsurance_OnWorldBuildsAndRunsLikeOff(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	on := idleInsuranceConfig()
	off := idleInsuranceConfig()
	off.Modules.Insurance = nil
	for i := range off.Roles {
		off.Roles[i].Cover = nil
	}
	if economysim.HashConfig(on) == economysim.HashConfig(off) {
		t.Fatal("HashConfig(insurance on) = HashConfig(insurance off), want the module hashed")
	}
	for _, seed := range []uint64{1, 7, 42} {
		onEvents, onSnap := riskRun(t, on, seed, 25)
		offEvents, offSnap := riskRun(t, off, seed, 25)
		if len(onEvents) == 0 {
			t.Fatalf("seed %d: no event after WorldCreated in 25 ticks, the comparison would pass for nothing", seed)
		}
		onEvents = slices.DeleteFunc(onEvents, func(e economysim.Event) bool {
			switch e.(type) {
			case economysim.CoverRound, economysim.CoverFormed:
				return true
			}
			return false
		})
		if !reflect.DeepEqual(onEvents, offEvents) {
			t.Errorf("seed %d: %d events with the module on, %d with it off, want the same events", seed, len(onEvents), len(offEvents))
		}
		onSnap.Modules = slices.DeleteFunc(slices.Clone(onSnap.Modules), func(m string) bool { return m == "insurance" })
		if !reflect.DeepEqual(onSnap, offSnap) {
			t.Errorf("seed %d: snapshots differ with the module on and off, want equal but for the module's name", seed)
		}
	}
}

// idleInsuranceConfig is insuranceConfig with nothing insurable: the insured role has no store
// share and the pack has no shipment, so the module on has no policy to form.
func idleInsuranceConfig() economysim.Config {
	c := insuranceConfig()
	for i := range c.Roles {
		if in := c.Roles[i].Cover; in != nil && in.Insure != nil {
			in.Insure.StoreShareBP = 0
		}
	}
	return c
}

// TestInsurance_WorldRefusesBrokenBlockAtConstruction verifies that NewWorld refuses a
// configuration whose cover block breaks a rule of spec-insurance.md 14.1, each with its
// sentinel, so a running simulation never meets a bad block (architecture.md 8.5): a
// block with both halves, one with neither, a bound and a module the block needs.
//
// Requirement: FR-INS-010, FR-INS-011
func TestInsurance_WorldRefusesBrokenBlockAtConstruction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		set  func(c *economysim.Config)
		want error
	}{
		{"both halves", func(c *economysim.Config) {
			insuranceRole(c, "cook").Cover.Insure = &economysim.InsureTerms{MinCoverBP: 1}
		}, economysim.ErrCoverShape},
		{"neither half", func(c *economysim.Config) { insuranceRole(c, "cook").Cover = &economysim.CoverTerms{} }, economysim.ErrCoverShape},
		{"a share above its bound", func(c *economysim.Config) { insuranceRole(c, "cook").Cover.Underwrite.ShareBP = 10_001 }, economysim.ErrOutOfBounds},
		{"credit off", func(c *economysim.Config) { c.Modules.Credit = nil }, economysim.ErrModuleDependency},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := insuranceConfig()
			tt.set(&c)
			if _, err := economysim.NewWorld(c); !errors.Is(err, tt.want) {
				t.Errorf("NewWorld(%s) error = %v, want %v", tt.name, err, tt.want)
			}
		})
	}
	w, err := economysim.NewWorld(insuranceConfig())
	if err != nil {
		t.Fatalf("NewWorld(insuranceConfig()) error = %v, want nil", err)
	}
	if _, err := w.Run(context.Background(), 1); err != nil {
		t.Errorf("Run(1) error = %v, want nil", err)
	}
}
