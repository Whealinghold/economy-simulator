package economysim_test

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the insurance module's configuration surface (spec-insurance.md 4, 13,
// 14): the failing case of each rule of 14.1, their order and the warnings of 14.2.
// The hash is in insurance_internal_test.go and the loader in config/insurance_test.go.

// insuranceConfig returns riskConfig (two regions, the foot mode, the storage classes
// and the risk rows) with the credit module on and the insurance module on: the cooks
// underwrite (a loading of 12 000, a store rate of 50, 8 000 of a loss, a cap of
// 5 000 and half of the free capacity) and the growers insure (a route when the lane's
// expected loss is 1 000, a ruin share of 3 000, a loading up to 15 000, at least 2 000
// of a loss, half of the held value in store and a store rate up to 200). Every number
// is a placeholder (N-015). The configuration raises none of the module's warnings.
func insuranceConfig() economysim.Config {
	c := riskConfig()
	c.Modules.Credit = &economysim.Credit{MinPrincipal: 5_000, ShieldTicks: 2, MaxClaims: 1_000}
	c.Modules.Insurance = &economysim.Insurance{ValueCapBP: 20_000, PayoutTicks: 1, MinSum: 100, MaxPolicies: 1_000, MaxCoverRequests: 4}
	for i := range c.Roles {
		switch c.Roles[i].ID {
		case "grower":
			c.Roles[i].Cover = &economysim.CoverTerms{Insure: &economysim.InsureTerms{
				MinLossBP: 1_000, RuinShareBP: 3_000, MaxLoadingBP: 15_000, MinCoverBP: 2_000, StoreShareBP: 5_000, MaxStoreRateBP: 200}}
		case "cook":
			c.Roles[i].Cover = &economysim.CoverTerms{Underwrite: &economysim.UnderwriteTerms{
				LoadingBP: 12_000, StoreRateBP: 50, CoverBP: 8_000, CapBP: 5_000, ShareBP: 5_000}}
		}
	}
	return c
}

// insuranceWarningRules returns the rules of the insurance warnings of a
// configuration.
func insuranceWarningRules(c economysim.Config) map[string]bool {
	out := map[string]bool{}
	for _, w := range c.Warnings() {
		p := strings.Join(w.Path, ".")
		if strings.HasPrefix(p, "modules.insurance") || strings.Contains(p, ".cover") {
			out[w.Rule] = true
		}
	}
	return out
}

// TestValidateInsurance_ConfigIsValid verifies that the configuration the insurance
// tests start from validates and raises none of the module's warnings, so every
// failing case breaks one rule, and that the module with no block validates (it is on
// and does nothing, with the warning no_cover_roles).
//
// Requirement: FR-INS-001, FR-INS-010, FR-INS-011
//
// Guard: the field cases start from insuranceConfig; if it did not validate, each
// case would pass for the wrong reason.
func TestValidateInsurance_ConfigIsValid(t *testing.T) {
	t.Parallel()
	c := insuranceConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("insuranceConfig().Validate() = %v, want nil", err)
	}
	if got := insuranceWarningRules(c); len(got) != 0 {
		t.Errorf("insuranceConfig().Warnings() has the insurance warnings %v, want none", got)
	}
	bare := riskConfig()
	bare.Modules.Credit = insuranceConfig().Modules.Credit
	bare.Modules.Insurance = &economysim.Insurance{ValueCapBP: 1, PayoutTicks: 1, MinSum: 1, MaxPolicies: 1, MaxCoverRequests: 1}
	if err := bare.Validate(); err != nil {
		t.Errorf("the module with no block .Validate() = %v, want nil", err)
	}
	if got := insuranceWarningRules(bare); !got["no_cover_roles"] {
		t.Errorf("the module with no block has the warnings %v, want no_cover_roles", got)
	}
}

// insuranceField builds a failing case from the valid configuration: the field, the
// case, the rule, the path, the value and bound, and what breaks it.
type insuranceField func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config)) fieldCase

// insuranceRole returns a pointer to the role of the configuration, for a test to
// change.
func insuranceRole(c *economysim.Config, id string) *economysim.Role {
	for i := range c.Roles {
		if c.Roles[i].ID == id {
			return &c.Roles[i]
		}
	}
	panic("no role " + id)
}

// insuranceFieldCases are the failing cases of the rows of the insurance module in the
// parameter-owner table (spec-insurance.md 13, 14.1, FR-CFG-091): each breaks one
// field of insuranceConfig. buildFieldCases appends them to the table that the
// registry test reads.
func insuranceFieldCases() []fieldCase {
	field := insuranceField(func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				*c = insuranceConfig()
				set(c)
			}}
	})
	cs := insuranceModuleCases(field)
	cs = append(cs, insuranceShapeCases(field)...)
	cs = append(cs, insuranceUnderwriteCases(field)...)
	return append(cs, insuranceInsureCases(field)...)
}

// insuranceModuleCases are the cases of the five fields of the module.
func insuranceModuleCases(field insuranceField) []fieldCase {
	at := func(key string) []string { return []string{"modules", "insurance", key} }
	num := func(v int64) string { return strconv.FormatInt(v, 10) }
	set := func(f func(m *economysim.Insurance)) func(c *economysim.Config) {
		return func(c *economysim.Config) { f(c.Modules.Insurance) }
	}
	return []fieldCase{
		field("Insurance.ValueCapBP", "below 1", economysim.ErrOutOfBounds, at("value_cap_bp"), "0", "1",
			set(func(m *economysim.Insurance) { m.ValueCapBP = 0 })),
		field("Insurance.ValueCapBP", "above MaxValueCapBP", economysim.ErrOutOfBounds, at("value_cap_bp"), num(economysim.MaxValueCapBP+1), num(economysim.MaxValueCapBP),
			set(func(m *economysim.Insurance) { m.ValueCapBP = economysim.MaxValueCapBP + 1 })),
		field("Insurance.PayoutTicks", "below 1", economysim.ErrOutOfBounds, at("payout_ticks"), "0", "1",
			set(func(m *economysim.Insurance) { m.PayoutTicks = 0 })),
		field("Insurance.PayoutTicks", "above MaxCreditTerm", economysim.ErrOutOfBounds, at("payout_ticks"), num(economysim.MaxCreditTerm+1), num(economysim.MaxCreditTerm),
			set(func(m *economysim.Insurance) { m.PayoutTicks = economysim.MaxCreditTerm + 1 })),
		field("Insurance.MinSum", "below 1", economysim.ErrOutOfBounds, at("min_sum"), "0", "1",
			set(func(m *economysim.Insurance) { m.MinSum = 0 })),
		field("Insurance.MinSum", "above MaxMoney", economysim.ErrOutOfBounds, at("min_sum"), num(economysim.MaxMoney+1), num(economysim.MaxMoney),
			set(func(m *economysim.Insurance) { m.MinSum = economysim.MaxMoney + 1 })),
		field("Insurance.MaxPolicies", "below 1", economysim.ErrOutOfBounds, at("max_policies"), "0", "1",
			set(func(m *economysim.Insurance) { m.MaxPolicies = 0 })),
		field("Insurance.MaxPolicies", "above MaxOpenPolicies", economysim.ErrOutOfBounds, at("max_policies"), num(economysim.MaxOpenPolicies+1), num(economysim.MaxOpenPolicies),
			set(func(m *economysim.Insurance) { m.MaxPolicies = economysim.MaxOpenPolicies + 1 })),
		field("Insurance.MaxCoverRequests", "below 1", economysim.ErrOutOfBounds, at("max_cover_requests"), "0", "1",
			set(func(m *economysim.Insurance) { m.MaxCoverRequests = 0 })),
		field("Insurance.MaxCoverRequests", "above MaxCoverRequestsBound", economysim.ErrOutOfBounds, at("max_cover_requests"), num(economysim.MaxCoverRequestsBound+1), num(economysim.MaxCoverRequestsBound),
			set(func(m *economysim.Insurance) { m.MaxCoverRequests = economysim.MaxCoverRequestsBound + 1 })),
	}
}

// insuranceShapeCases are the cases of the role block: the module off, a missing
// dependency, both sub-blocks and neither.
func insuranceShapeCases(field insuranceField) []fieldCase {
	block := []string{"roles", "grower", "cover"}
	module := []string{"modules", "insurance"}
	return []fieldCase{
		field("Role.Cover", "a block with the module off", economysim.ErrModuleDependency, block, "cover block", "modules.insurance",
			func(c *economysim.Config) {
				c.Modules.Insurance = nil
				insuranceRole(c, "cook").Cover = nil
			}),
		field("Insurance.ValueCapBP", "the module with Risk off", economysim.ErrModuleDependency, module, "modules.insurance", "modules.risk",
			func(c *economysim.Config) { c.Modules.Risk = nil }),
		field("Insurance.MinSum", "the module with Credit off", economysim.ErrModuleDependency, module, "modules.insurance", "modules.credit",
			func(c *economysim.Config) { c.Modules.Credit = nil }),
		field("CoverTerms.Underwrite", "both Underwrite and Insure", economysim.ErrCoverShape, block, "both", "1",
			func(c *economysim.Config) {
				insuranceRole(c, "grower").Cover.Underwrite = &economysim.UnderwriteTerms{CoverBP: 1, CapBP: 1, ShareBP: 1}
			}),
		field("CoverTerms.Insure", "neither Underwrite nor Insure", economysim.ErrCoverShape, block, "neither", "1",
			func(c *economysim.Config) { insuranceRole(c, "grower").Cover.Insure = nil }),
	}
}

// insuranceUnderwriteCases are the cases of the five fields of an Underwrite block.
func insuranceUnderwriteCases(field insuranceField) []fieldCase {
	at := func(key string) []string { return []string{"roles", "cook", "cover", "underwrite", key} }
	set := func(f func(u *economysim.UnderwriteTerms)) func(c *economysim.Config) {
		return func(c *economysim.Config) { f(insuranceRole(c, "cook").Cover.Underwrite) }
	}
	return []fieldCase{
		field("UnderwriteTerms.LoadingBP", "above MaxLoadingBound", economysim.ErrOutOfBounds, at("loading_bp"), "100001", "100000",
			set(func(u *economysim.UnderwriteTerms) { u.LoadingBP = economysim.MaxLoadingBound + 1 })),
		field("UnderwriteTerms.StoreRateBP", "above MaxStoreRateBound", economysim.ErrOutOfBounds, at("store_rate_bp"), "10001", "10000",
			set(func(u *economysim.UnderwriteTerms) { u.StoreRateBP = economysim.MaxStoreRateBound + 1 })),
		field("UnderwriteTerms.CoverBP", "below 1", economysim.ErrOutOfBounds, at("cover_bp"), "0", "1",
			set(func(u *economysim.UnderwriteTerms) { u.CoverBP = 0 })),
		field("UnderwriteTerms.CoverBP", "above 10 000", economysim.ErrOutOfBounds, at("cover_bp"), "10001", "10000",
			set(func(u *economysim.UnderwriteTerms) { u.CoverBP = 10_001 })),
		field("UnderwriteTerms.CapBP", "below 1", economysim.ErrOutOfBounds, at("cap_bp"), "0", "1",
			set(func(u *economysim.UnderwriteTerms) { u.CapBP = 0 })),
		field("UnderwriteTerms.CapBP", "above MaxCapBound", economysim.ErrOutOfBounds, at("cap_bp"), "100001", "100000",
			set(func(u *economysim.UnderwriteTerms) { u.CapBP = economysim.MaxCapBound + 1 })),
		field("UnderwriteTerms.ShareBP", "below 1", economysim.ErrOutOfBounds, at("share_bp"), "0", "1",
			set(func(u *economysim.UnderwriteTerms) { u.ShareBP = 0 })),
		field("UnderwriteTerms.ShareBP", "above 10 000", economysim.ErrOutOfBounds, at("share_bp"), "10001", "10000",
			set(func(u *economysim.UnderwriteTerms) { u.ShareBP = 10_001 })),
	}
}

// insuranceInsureCases are the cases of the six fields of an Insure block, and the two
// dependencies of a block that asks for route or for store cover.
func insuranceInsureCases(field insuranceField) []fieldCase {
	at := func(key string) []string { return []string{"roles", "grower", "cover", "insure", key} }
	set := func(f func(s *economysim.InsureTerms)) func(c *economysim.Config) {
		return func(c *economysim.Config) { f(insuranceRole(c, "grower").Cover.Insure) }
	}
	return []fieldCase{
		field("InsureTerms.MinLossBP", "above 10 000", economysim.ErrOutOfBounds, at("min_loss_bp"), "10001", "10000",
			set(func(s *economysim.InsureTerms) { s.MinLossBP = 10_001 })),
		field("InsureTerms.MinLossBP", "route cover with Transport off", economysim.ErrModuleDependency, at("min_loss_bp"), "route cover", "modules.transport",
			func(c *economysim.Config) {
				*c = insuranceConfig()
				c.Modules.Transport, c.Modules.Risk.Routes = nil, nil
			}),
		field("InsureTerms.RuinShareBP", "above MaxRuinBound", economysim.ErrOutOfBounds, at("ruin_share_bp"), "100001", "100000",
			set(func(s *economysim.InsureTerms) { s.RuinShareBP = economysim.MaxRuinBound + 1 })),
		field("InsureTerms.MaxLoadingBP", "above MaxLoadingBound", economysim.ErrOutOfBounds, at("max_loading_bp"), "100001", "100000",
			set(func(s *economysim.InsureTerms) { s.MaxLoadingBP = economysim.MaxLoadingBound + 1 })),
		field("InsureTerms.MinCoverBP", "below 1", economysim.ErrOutOfBounds, at("min_cover_bp"), "0", "1",
			set(func(s *economysim.InsureTerms) { s.MinCoverBP = 0 })),
		field("InsureTerms.MinCoverBP", "above 10 000", economysim.ErrOutOfBounds, at("min_cover_bp"), "10001", "10000",
			set(func(s *economysim.InsureTerms) { s.MinCoverBP = 10_001 })),
		field("InsureTerms.StoreShareBP", "above 10 000", economysim.ErrOutOfBounds, at("store_share_bp"), "10001", "10000",
			set(func(s *economysim.InsureTerms) { s.StoreShareBP = 10_001 })),
		field("InsureTerms.StoreShareBP", "store cover with Storage off", economysim.ErrModuleDependency, at("store_share_bp"), "store cover", "modules.storage",
			func(c *economysim.Config) {
				*c = insuranceConfig()
				c.Modules.Storage, c.Modules.Risk.Stores = nil, nil
			}),
		field("InsureTerms.MaxStoreRateBP", "above MaxStoreRateBound", economysim.ErrOutOfBounds, at("max_store_rate_bp"), "10001", "10000",
			set(func(s *economysim.InsureTerms) { s.MaxStoreRateBP = economysim.MaxStoreRateBound + 1 })),
	}
}

// TestValidateInsurance_Errors verifies each row of 14.1 that Validate checks: every
// case of insuranceFieldCases, which breaks only its own field of a valid
// configuration, is refused with its rule (errors.Is) at its path with its value and
// bound.
//
// Requirement: FR-INS-010, FR-INS-011
func TestValidateInsurance_Errors(t *testing.T) {
	t.Parallel()
	for _, tc := range insuranceFieldCases() {
		t.Run(tc.field+" "+tc.name, func(t *testing.T) {
			t.Parallel()
			var c economysim.Config
			tc.mutate(&c)
			err := c.Validate()
			if !errors.Is(err, tc.rules[0]) {
				t.Fatalf("Validate() = %v, want %v", err, tc.rules[0])
			}
			var found bool
			for _, e := range unwrapAll(err) {
				var ve *economysim.ValidationError
				if errors.As(e, &ve) && errors.Is(e, tc.rules[0]) && strings.Join(ve.Path, ".") == strings.Join(tc.paths[0], ".") {
					found = found || (ve.Value == tc.value && ve.Bound == tc.bound)
				}
			}
			if !found {
				t.Errorf("Validate() = %v, want %v at %v with value %q and bound %q", err, tc.rules[0], tc.paths[0], tc.value, tc.bound)
			}
		})
	}
}

// TestValidateInsurance_Order verifies the order of 14.1: two mistakes at once report
// the earlier rule only, the bounds before the shape and the shape before the
// dependency.
//
// Requirement: FR-INS-011
func TestValidateInsurance_Order(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		break_ func(c *economysim.Config)
		want   error
		not    error
	}{
		{"bounds before shape", func(c *economysim.Config) {
			insuranceRole(c, "grower").Cover.Underwrite = &economysim.UnderwriteTerms{}
			c.Modules.Insurance.MaxPolicies = 0
		}, economysim.ErrOutOfBounds, economysim.ErrCoverShape},
		{"shape before dependency", func(c *economysim.Config) {
			insuranceRole(c, "grower").Cover.Insure = nil
			c.Modules.Insurance = nil
		}, economysim.ErrCoverShape, economysim.ErrModuleDependency},
		{"a block bound before the shape of another", func(c *economysim.Config) {
			insuranceRole(c, "cook").Cover.Underwrite.ShareBP = 0
			insuranceRole(c, "grower").Cover.Insure = nil
		}, economysim.ErrOutOfBounds, economysim.ErrCoverShape},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := insuranceConfig()
			tt.break_(&c)
			err := c.Validate()
			if !errors.Is(err, tt.want) {
				t.Errorf("Validate() = %v, want %v", err, tt.want)
			}
			if errors.Is(err, tt.not) {
				t.Errorf("Validate() = %v, want %v not reported while an earlier rule fails", err, tt.not)
			}
		})
	}
}

// TestValidateInsurance_StoreOnlyInsuredNeedsNoTransport verifies spec-insurance.md
// 4.1.1: an insured with MinLossBP 0 and store cover is valid with Storage and no
// Transport, and the dependency on Transport is the route request's alone.
//
// Requirement: FR-INS-010
func TestValidateInsurance_StoreOnlyInsuredNeedsNoTransport(t *testing.T) {
	t.Parallel()
	c := insuranceConfig()
	c.Modules.Transport, c.Modules.Risk.Routes = nil, nil
	insuranceRole(&c, "grower").Cover.Insure.MinLossBP = 0
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate(store-only insured, no Transport) = %v, want nil", err)
	}
}

// TestSentinels_Insurance verifies that ErrCoverShape is a root sentinel of its own,
// not the credit module's, that its text names the cover block, and that the
// collision of a money flow ID with premium or payout is ErrReasonCollision.
//
// Requirement: FR-INS-010, FR-CFG-091
func TestSentinels_Insurance(t *testing.T) {
	t.Parallel()
	if errors.Is(economysim.ErrCoverShape, economysim.ErrCreditShape) || errors.Is(economysim.ErrCreditShape, economysim.ErrCoverShape) {
		t.Errorf("ErrCoverShape and ErrCreditShape are the same error, want two sentinels")
	}
	if got := economysim.ErrCoverShape.Error(); !strings.Contains(got, "cover") {
		t.Errorf("ErrCoverShape.Error() = %q, want it to name the cover block", got)
	}
	for _, id := range []string{"premium", "payout"} {
		c := insuranceConfig()
		c.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{
			{ID: id, Kind: economysim.FlowGrant, Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: "cook"}, Amount: 1},
		}}
		if err := c.Validate(); !errors.Is(err, economysim.ErrReasonCollision) {
			t.Errorf("Validate(a money flow %q) = %v, want ErrReasonCollision", id, err)
		}
	}
}

// TestWarnings_Insurance verifies each of the eleven warnings of spec-insurance.md
// 14.2: the configuration that raises it, with the rule, and that none fails
// validation. Each case starts from insuranceConfig, which raises none.
//
// Requirement: FR-INS-010, FR-CFG-091
func TestWarnings_Insurance(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		set  func(c *economysim.Config)
		want string
	}{
		{"no roles with a block", func(c *economysim.Config) {
			insuranceRole(c, "grower").Cover, insuranceRole(c, "cook").Cover = nil, nil
		}, "no_cover_roles"},
		{"no underwriter", func(c *economysim.Config) { insuranceRole(c, "cook").Cover = nil }, "cover_no_underwriter"},
		{"no insured", func(c *economysim.Config) { insuranceRole(c, "grower").Cover = nil }, "cover_no_insured"},
		{"route cover with no route row", func(c *economysim.Config) { c.Modules.Risk.Routes = nil }, "cover_without_risk"},
		{"store cover with no store row", func(c *economysim.Config) { c.Modules.Risk.Stores = nil }, "cover_without_risk"},
		{"loading above every limit", func(c *economysim.Config) {
			insuranceRole(c, "cook").Cover.Underwrite.LoadingBP = 20_000
		}, "loading_above_limits"},
		{"a minimum above every start money", func(c *economysim.Config) { c.Modules.Insurance.MinSum = economysim.MaxMoney }, "cover_below_minimum"},
		{"an underwriter with no income", func(c *economysim.Config) { insuranceRole(c, "cook").Recipes = nil }, "underwriter_without_income"},
		{"a cap above the money", func(c *economysim.Config) {
			r := insuranceRole(c, "cook")
			r.StartStock = nil
			r.Cover.Underwrite.CapBP = 20_000
		}, "underwriter_cap_above_money"},
		{"fewer claims than policies", func(c *economysim.Config) { c.Modules.Credit.MaxClaims = 10 }, "claims_below_policies"},
		{"an underwriter with no shield", func(c *economysim.Config) { c.Modules.Credit.ShieldTicks = 0 }, "no_shield_underwriter"},
		{"the value cap at its bound", func(c *economysim.Config) { c.Modules.Insurance.ValueCapBP = economysim.MaxValueCapBP }, "value_cap_unbound"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := insuranceConfig()
			tt.set(&c)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate(%s) = %v, want nil: a warning never fails", tt.name, err)
			}
			if got := insuranceWarningRules(c); !got[tt.want] {
				t.Errorf("Warnings(%s) has the insurance warnings %v, want %s", tt.name, got, tt.want)
			}
		})
	}
}

// TestWarnings_InsuranceOffIsSilent verifies that no insurance warning is raised with
// the module off, with a role block or none, and that the overlay-like configuration
// of insuranceConfig is silent.
//
// Requirement: FR-INS-001, FR-CFG-091
func TestWarnings_InsuranceOffIsSilent(t *testing.T) {
	t.Parallel()
	c := insuranceConfig()
	c.Modules.Insurance = nil
	for _, w := range c.Warnings() {
		if slices.Contains([]string{"no_cover_roles", "cover_no_underwriter", "cover_no_insured", "cover_without_risk", "loading_above_limits",
			"cover_below_minimum", "underwriter_without_income", "underwriter_cap_above_money", "claims_below_policies",
			"no_shield_underwriter", "value_cap_unbound"}, w.Rule) {
			t.Errorf("Warnings() with the module off has %v %q, want no insurance warning", w.Path, w.Rule)
		}
	}
}
