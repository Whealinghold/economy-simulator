package economysim_test

import (
	"encoding/binary"
	"errors"
	"log/slog"
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the credit module's configuration surface (spec-credit.md 4, 14, 15):
// the failing case of each rule of 15.1, their order and the warnings of 15.2. The
// hash is in credit_internal_test.go and the loader in config/credit_test.go.

// creditConfig returns referenceConfig with the credit module on: the growers
// borrow (a 60-tick job, a limit of 1 500 basis points, 90 ticks of room, 4 ticks of
// slack) and the cooks lend (600 basis points, 90 ticks, keeping 3 ticks of upkeep,
// half of what is lendable). MinPrincipal 5 000 is large enough that the interest of
// a one-tick loan at 600 basis points does not round to 0, a shield of 2 ticks, and
// every default is a placeholder (N-015). The configuration raises none of the
// module's warnings.
func creditConfig() economysim.Config {
	c := referenceConfig()
	c.Modules.Credit = &economysim.Credit{MinPrincipal: 5_000, ShieldTicks: 2, MaxClaims: 1_000}
	for i := range c.Roles {
		switch c.Roles[i].ID {
		case "grower":
			c.Roles[i].Credit = &economysim.CreditTerms{Borrow: &economysim.BorrowTerms{MaxRateBP: 1_500, MaxDebtTicks: 90, SlackTicks: 4}}
		case "cook":
			c.Roles[i].Credit = &economysim.CreditTerms{Lend: &economysim.LendTerms{RateBP: 600, MaxTermTicks: 90, KeepTicks: 3, ShareBP: 5_000}}
		}
	}
	return c
}

// role returns a pointer to the role of the configuration, for a test to change.
func creditRole(c *economysim.Config, id string) *economysim.Role {
	for i := range c.Roles {
		if c.Roles[i].ID == id {
			return &c.Roles[i]
		}
	}
	panic("no role " + id)
}

// creditWarningRules returns the rules of the credit warnings of a configuration,
// by path.
func creditWarningRules(c economysim.Config) map[string]bool {
	out := map[string]bool{}
	for _, w := range c.Warnings() {
		p := strings.Join(w.Path, ".")
		if strings.HasPrefix(p, "modules.credit") || strings.Contains(p, ".credit.") {
			out[w.Rule] = true
		}
	}
	return out
}

// TestValidateCredit_ConfigIsValid verifies that the configuration the credit
// tests start from validates and raises none of the module's warnings, so every
// failing case breaks one rule, and that the module with no block validates (it is
// on and does nothing, with the warning no_credit_roles).
//
// Requirement: FR-CRD-010, FR-CRD-011
//
// Guard: the field cases start from creditConfig; if it did not validate, each case
// would pass for the wrong reason.
func TestValidateCredit_ConfigIsValid(t *testing.T) {
	t.Parallel()
	c := creditConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("creditConfig().Validate() = %v, want nil", err)
	}
	if got := creditWarningRules(c); len(got) != 0 {
		t.Errorf("creditConfig().Warnings() has the credit warnings %v, want none", got)
	}
	bare := referenceConfig()
	bare.Modules.Credit = &economysim.Credit{MinPrincipal: 1, MaxClaims: 1}
	if err := bare.Validate(); err != nil {
		t.Errorf("the module with no block .Validate() = %v, want nil", err)
	}
	if got := creditWarningRules(bare); !got["no_credit_roles"] {
		t.Errorf("the module with no block has the warnings %v, want no_credit_roles", got)
	}
}

// creditField builds a failing case from the valid configuration: the field, the
// case, the rule, the path, the value and bound, and what breaks it.
type creditField func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config)) fieldCase

// creditFieldCases are the failing cases of the rows of the credit module in the
// parameter-owner table (spec-credit.md 14, 15.1, FR-CFG-088): each breaks one field
// of creditConfig. buildFieldCases appends them to the table that the registry test
// reads.
func creditFieldCases() []fieldCase {
	field := creditField(func(field, name string, rule error, p []string, value, bound string, set func(c *economysim.Config)) fieldCase {
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
			mutate: func(c *economysim.Config) {
				*c = creditConfig()
				set(c)
			}}
	})
	cs := creditModuleCases(field)
	cs = append(cs, creditShapeCases(field)...)
	cs = append(cs, creditLendCases(field)...)
	return append(cs, creditBorrowCases(field)...)
}

// creditModuleCases are the cases of the four fields of the module.
func creditModuleCases(field creditField) []fieldCase {
	at := func(key string) []string { return []string{"modules", "credit", key} }
	over := func(v int64) string { return strconv.FormatInt(v, 10) }
	return []fieldCase{
		field("Credit.MinPrincipal", "below 1", economysim.ErrOutOfBounds, at("min_principal"), "0", "1",
			func(c *economysim.Config) { c.Modules.Credit.MinPrincipal = 0 }),
		field("Credit.MinPrincipal", "above MaxMoney", economysim.ErrOutOfBounds, at("min_principal"), over(economysim.MaxMoney+1), over(economysim.MaxMoney),
			func(c *economysim.Config) { c.Modules.Credit.MinPrincipal = economysim.MaxMoney + 1 }),
		field("Credit.ShieldTicks", "above MaxCreditTerm", economysim.ErrOutOfBounds, at("shield_ticks"), over(economysim.MaxCreditTerm+1), over(economysim.MaxCreditTerm),
			func(c *economysim.Config) { c.Modules.Credit.ShieldTicks = economysim.MaxCreditTerm + 1 }),
		field("Credit.WriteOffTicks", "below 0", economysim.ErrOutOfBounds, at("write_off_ticks"), "-1", "0",
			func(c *economysim.Config) { c.Modules.Credit.WriteOffTicks = -1 }),
		field("Credit.MaxClaims", "below 1", economysim.ErrOutOfBounds, at("max_claims"), "0", "1",
			func(c *economysim.Config) { c.Modules.Credit.MaxClaims = 0 }),
		field("Credit.MaxClaims", "above MaxOpenClaims", economysim.ErrOutOfBounds, at("max_claims"), over(economysim.MaxOpenClaims+1), over(economysim.MaxOpenClaims),
			func(c *economysim.Config) { c.Modules.Credit.MaxClaims = economysim.MaxOpenClaims + 1 }),
	}
}

// creditShapeCases are the cases of the role block: the module off, both
// sub-blocks and neither.
func creditShapeCases(field creditField) []fieldCase {
	block := []string{"roles", "grower", "credit"}
	return []fieldCase{
		field("Role.Credit", "a block with the module off", economysim.ErrModuleDependency, block, "credit block", "modules.credit",
			func(c *economysim.Config) {
				c.Modules.Credit = nil
				creditRole(c, "cook").Credit = nil
			}),
		field("CreditTerms.Lend", "both Lend and Borrow", economysim.ErrCreditShape, block, "both", "1",
			func(c *economysim.Config) {
				creditRole(c, "grower").Credit.Lend = &economysim.LendTerms{MaxTermTicks: 1, ShareBP: 1}
			}),
		field("CreditTerms.Borrow", "neither Lend nor Borrow", economysim.ErrCreditShape, block, "neither", "1",
			func(c *economysim.Config) { creditRole(c, "grower").Credit.Borrow = nil }),
	}
}

// creditLendCases are the cases of the four fields of a Lend block.
func creditLendCases(field creditField) []fieldCase {
	at := func(key string) []string { return []string{"roles", "cook", "credit", "lend", key} }
	set := func(f func(l *economysim.LendTerms)) func(c *economysim.Config) {
		return func(c *economysim.Config) { f(creditRole(c, "cook").Credit.Lend) }
	}
	return []fieldCase{
		field("LendTerms.RateBP", "above MaxCreditRateBP", economysim.ErrOutOfBounds, at("rate_bp"), "100001", "100000",
			set(func(l *economysim.LendTerms) { l.RateBP = economysim.MaxCreditRateBP + 1 })),
		field("LendTerms.MaxTermTicks", "below 1", economysim.ErrOutOfBounds, at("max_term_ticks"), "0", "1",
			set(func(l *economysim.LendTerms) { l.MaxTermTicks = 0 })),
		field("LendTerms.KeepTicks", "above MaxCreditTerm", economysim.ErrOutOfBounds, at("keep_ticks"), "3651", "3650",
			set(func(l *economysim.LendTerms) { l.KeepTicks = economysim.MaxCreditTerm + 1 })),
		field("LendTerms.ShareBP", "below 1", economysim.ErrOutOfBounds, at("share_bp"), "0", "1",
			set(func(l *economysim.LendTerms) { l.ShareBP = 0 })),
		field("LendTerms.ShareBP", "above 10 000", economysim.ErrOutOfBounds, at("share_bp"), "10001", "10000",
			set(func(l *economysim.LendTerms) { l.ShareBP = 10_001 })),
	}
}

// creditBorrowCases are the cases of the three fields of a Borrow block.
func creditBorrowCases(field creditField) []fieldCase {
	at := func(key string) []string { return []string{"roles", "grower", "credit", "borrow", key} }
	set := func(f func(b *economysim.BorrowTerms)) func(c *economysim.Config) {
		return func(c *economysim.Config) { f(creditRole(c, "grower").Credit.Borrow) }
	}
	return []fieldCase{
		field("BorrowTerms.MaxRateBP", "below 0", economysim.ErrOutOfBounds, at("max_rate_bp"), "-1", "0",
			set(func(b *economysim.BorrowTerms) { b.MaxRateBP = -1 })),
		field("BorrowTerms.MaxDebtTicks", "below 1", economysim.ErrOutOfBounds, at("max_debt_ticks"), "0", "1",
			set(func(b *economysim.BorrowTerms) { b.MaxDebtTicks = 0 })),
		field("BorrowTerms.SlackTicks", "above MaxCreditTerm", economysim.ErrOutOfBounds, at("slack_ticks"), "3651", "3650",
			set(func(b *economysim.BorrowTerms) { b.SlackTicks = economysim.MaxCreditTerm + 1 })),
	}
}

// TestValidateCredit_Errors verifies each row of 15.1 that Validate checks: every case
// of creditFieldCases, which breaks only its own field of a valid configuration, is
// refused with its rule (errors.Is) at its path with its value and bound, and with
// no other credit error.
//
// Requirement: FR-CRD-010, FR-CRD-011
func TestValidateCredit_Errors(t *testing.T) {
	t.Parallel()
	for _, tc := range creditFieldCases() {
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

// TestValidateCredit_Order verifies the order of 15.1: two mistakes at once report
// the earlier rule only, the bounds before the shape and the shape before the
// dependency, and the module's dependency error never hides a bounds error.
//
// Requirement: FR-CRD-011
func TestValidateCredit_Order(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		break_ func(c *economysim.Config)
		want   error
		not    error
	}{
		{"bounds before shape", func(c *economysim.Config) {
			creditRole(c, "grower").Credit.Lend = &economysim.LendTerms{}
			c.Modules.Credit.MaxClaims = 0
		}, economysim.ErrOutOfBounds, economysim.ErrCreditShape},
		{"shape before dependency", func(c *economysim.Config) {
			creditRole(c, "grower").Credit.Borrow = nil
			c.Modules.Credit = nil
		}, economysim.ErrCreditShape, economysim.ErrModuleDependency},
		{"a block bound before the shape of another", func(c *economysim.Config) {
			creditRole(c, "cook").Credit.Lend.ShareBP = 0
			creditRole(c, "grower").Credit.Borrow = nil
		}, economysim.ErrOutOfBounds, economysim.ErrCreditShape},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := creditConfig()
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

// TestValidateCredit_ReasonCollision verifies that a money flow whose ID is one of
// the three loan reasons is ErrReasonCollision, with the module on and off.
//
// Requirement: FR-CRD-011
func TestValidateCredit_ReasonCollision(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"loan_out", "loan_repay", "loan_interest"} {
		for _, on := range []bool{true, false} {
			c := creditConfig()
			if !on {
				c = referenceConfig()
			}
			c.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{
				{ID: id, Kind: economysim.FlowGrant, Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: "cook"}, Amount: 1},
			}}
			if err := c.Validate(); !errors.Is(err, economysim.ErrReasonCollision) {
				t.Errorf("Validate() with a flow %q and the module on=%v = %v, want ErrReasonCollision", id, on, err)
			}
		}
	}
}

// TestCreditWarnings verifies each warning of 15.2 that this module raises: the
// condition raises it, the quiet case does not, and none fails validation.
//
// Requirement: FR-CRD-076
func TestCreditWarnings(t *testing.T) {
	t.Parallel()
	for _, tt := range creditWarningCases() {
		t.Run(tt.rule, func(t *testing.T) {
			t.Parallel()
			c := creditConfig()
			tt.raise(&c)
			if err := c.Validate(); err != nil {
				t.Fatalf("the configuration of %s does not validate: %v", tt.rule, err)
			}
			if got := creditWarningRules(c); !got[tt.rule] {
				t.Errorf("Warnings() = %v, want %s", got, tt.rule)
			}
			if got := creditWarningRules(creditConfig()); got[tt.rule] {
				t.Errorf("the quiet configuration has %s", tt.rule)
			}
		})
	}
}

// TestCredit_WarningsAreLogged verifies that each credit warning reaches the injected
// logger once, at Warn, with the constant message and the rule and the dotted path
// of the warning, before "world created", that the lines are the ones
// World.Warnings returns, and that a configuration that draws none logs none.
//
// Requirement: FR-CRD-076
//
// Guard: the log is the only signal of a credit configuration that is valid but
// does nothing (no shield, interest that rounds to 0) to a consumer that never calls
// Warnings; without this test a change to the construction line could drop them.
func TestCredit_WarningsAreLogged(t *testing.T) {
	t.Parallel()
	isCredit := func(path string) bool {
		return strings.HasPrefix(path, "modules.credit") || strings.Contains(path, ".credit.")
	}
	c := creditConfig()
	c.Modules.Credit.ShieldTicks = 0
	c.Modules.Credit.MinPrincipal = 100
	creditRole(&c, "grower").Credit.Borrow.MaxDebtTicks = 60
	h := newCaptureHandler()
	w, err := economysim.NewWorld(c, economysim.WithLogger(slog.New(h)))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil (a warning never fails construction)", err)
	}
	var want []string
	for _, wn := range w.Warnings() {
		if p := strings.Join(wn.Path, "."); isCredit(p) {
			want = append(want, wn.Rule+"@"+p)
		}
	}
	for _, rule := range []string{"no_shield", "interest_rounds_to_zero", "cap_below_horizon"} {
		if !slices.ContainsFunc(want, func(s string) bool { return strings.HasPrefix(s, rule+"@") }) {
			t.Fatalf("Warnings() = %v, want a %s warning for the scenario to mean anything", want, rule)
		}
	}
	var got []string
	created := -1
	for i, l := range h.sink.lines {
		if l.msg == "world created" {
			created = i
		}
		if l.level == slog.LevelError {
			t.Errorf("log line %v at Error, want none", l)
		}
		if l.level != slog.LevelWarn {
			continue
		}
		attrs := map[string]string{}
		for _, a := range l.attrs {
			attrs[a.Key] = a.Value.String()
		}
		if !isCredit(attrs["path"]) {
			continue // the reference chain's own warnings
		}
		if l.msg != "configuration warning" || created >= 0 {
			t.Errorf("credit warning line %v (after \"world created\": %v), want message %q before the summary", l, created >= 0, "configuration warning")
		}
		got = append(got, attrs["rule"]+"@"+attrs["path"])
	}
	if !slices.Equal(got, want) {
		t.Errorf("logged credit warnings = %v, want World.Warnings() = %v", got, want)
	}

	quiet := newCaptureHandler()
	if _, err := economysim.NewWorld(creditConfig(), economysim.WithLogger(slog.New(quiet))); err != nil {
		t.Fatalf("NewWorld(valid credit) error = %v, want nil", err)
	}
	for _, l := range quiet.sink.lines {
		for _, a := range l.attrs {
			if a.Key == "path" && isCredit(a.Value.String()) {
				t.Errorf("a clean credit configuration logged %v, want no credit warning", l)
			}
		}
	}
}

// creditWarningCase is one warning and a change that raises it.
type creditWarningCase struct {
	rule  string
	raise func(c *economysim.Config)
}

// creditWarningCases are the warnings of 15.2 with a configuration change each.
func creditWarningCases() []creditWarningCase {
	cook := func(c *economysim.Config) *economysim.LendTerms { return creditRole(c, "cook").Credit.Lend }
	grower := func(c *economysim.Config) *economysim.BorrowTerms { return creditRole(c, "grower").Credit.Borrow }
	return []creditWarningCase{
		{"no_credit_roles", func(c *economysim.Config) {
			creditRole(c, "cook").Credit, creditRole(c, "grower").Credit = nil, nil
		}},
		{"credit_no_lender", func(c *economysim.Config) { creditRole(c, "cook").Credit = nil }},
		{"credit_no_borrower", func(c *economysim.Config) { creditRole(c, "grower").Credit = nil }},
		{"rate_above_limits", func(c *economysim.Config) { cook(c).RateBP = 2_000 }},
		{"term_exceeds_lenders", func(c *economysim.Config) { cook(c).MaxTermTicks = 60 }},
		{"cap_below_horizon", func(c *economysim.Config) { grower(c).MaxDebtTicks = 60 }},
		{"lender_without_cash", func(c *economysim.Config) { cook(c).KeepTicks = 20 }},
		{"lender_without_income", func(c *economysim.Config) { creditRole(c, "cook").Recipes = nil }},
		{"lender_keep_below_term", func(c *economysim.Config) { creditRole(c, "cook").Recipes = nil }},
		{"no_shield", func(c *economysim.Config) { c.Modules.Credit.ShieldTicks = 0 }},
		{"credit_no_upkeep", func(c *economysim.Config) {
			c.World.Upkeep = nil
			c.Regions[0].ExtraUpkeep = nil
			c.Regions[0].ReferenceBasket = 0
		}},
		{"credit_free", func(c *economysim.Config) { cook(c).RateBP = 0 }},
		{"interest_rounds_to_zero", func(c *economysim.Config) { c.Modules.Credit.MinPrincipal = 100 }},
	}
}

// TestCreditWarnings_Late verifies that the horizon of a borrower counts the tick of
// Late where the household can be too short of money to buy its inputs at once, so
// the same configuration warns term_exceeds_lenders only when it is.
//
// Requirement: FR-CRD-076
func TestCreditWarnings_Late(t *testing.T) {
	t.Parallel()
	c := creditConfig()
	creditRole(&c, "cook").Credit.Lend.MaxTermTicks = 65 // H = 60 + 1 + 4 = 65 fits exactly
	if got := creditWarningRules(c); got["term_exceeds_lenders"] {
		t.Fatalf("a lender term of 65 for a horizon of 65 warns term_exceeds_lenders")
	}
	creditRole(&c, "grower").StartMoney = 1_000 // below one tick of upkeep and the inputs: Late can be 1, H = 66
	if got := creditWarningRules(c); !got["term_exceeds_lenders"] {
		t.Errorf("a grower that can be late and a lender term of 65 do not warn term_exceeds_lenders, want the horizon 66")
	}
}

// TestCreditWarnings_UnvalidatedStartMoney verifies that a start money far below 0,
// which Warnings accepts because it runs on a configuration that does not validate,
// is read as short of the keep and of the inputs, not wrapped to a huge balance: the
// lender warns lender_without_cash and the grower can be late, so its horizon is 66
// and a lender term of 65 warns term_exceeds_lenders.
//
// Requirement: FR-CRD-076
//
// Fix: dp3-crd-1. StartMoney less the cost was a bare subtraction, so math.MinInt64
// less one tick of upkeep wrapped to a large positive and both warnings stayed quiet.
func TestCreditWarnings_UnvalidatedStartMoney(t *testing.T) {
	t.Parallel()
	c := creditConfig()
	creditRole(&c, "cook").StartMoney = math.MinInt64
	if got := creditWarningRules(c); !got["lender_without_cash"] {
		t.Errorf("a lender with StartMoney MinInt64 does not warn lender_without_cash: %v", got)
	}
	c = creditConfig()
	creditRole(&c, "cook").Credit.Lend.MaxTermTicks = 65 // H = 60 + 1 + 4 fits exactly, unless Late is 1
	creditRole(&c, "grower").StartMoney = math.MinInt64
	if got := creditWarningRules(c); !got["term_exceeds_lenders"] {
		t.Errorf("a grower with StartMoney MinInt64 does not warn term_exceeds_lenders: %v", got)
	}
}

// TestCreditWarnings_LenderIncome verifies the two warnings about a lender's
// income: a lender with no recipe and no flow that targets it warns
// lender_without_income, and also lender_keep_below_term unless its keep is as long
// as its term, and a flow that pays the role silences both.
//
// Requirement: FR-CRD-076
func TestCreditWarnings_LenderIncome(t *testing.T) {
	t.Parallel()
	noIncome := func(c *economysim.Config) { creditRole(c, "cook").Recipes = nil }
	tests := []struct {
		name         string
		mutate       func(c *economysim.Config)
		income, keep bool
	}{
		{"no recipe", noIncome, true, true},
		{"keep as long as the term", func(c *economysim.Config) {
			noIncome(c)
			creditRole(c, "cook").Credit.Lend.KeepTicks = 90
		}, true, false},
		{"a flow pays the role", func(c *economysim.Config) {
			noIncome(c)
			c.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{
				{ID: "pension", Kind: economysim.FlowGrant, Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: "cook"}, Amount: 1},
			}}
		}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := creditConfig()
			tt.mutate(&c)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate() = %v, want nil", err)
			}
			got := creditWarningRules(c)
			if got["lender_without_income"] != tt.income || got["lender_keep_below_term"] != tt.keep {
				t.Errorf("warnings = %v, want lender_without_income %v and lender_keep_below_term %v", got, tt.income, tt.keep)
			}
		})
	}
}

// TestCreditWarnings_Logged verifies that a credit warning of a configuration
// reaches the construction logger once, at Warn, with its rule and path
// (FR-CFG-020), and that it does not stop NewWorld.
//
// Requirement: FR-CRD-076
func TestCreditWarnings_Logged(t *testing.T) {
	t.Parallel()
	c := creditConfig()
	c.Modules.Credit.ShieldTicks = 0
	var found int
	for _, l := range logRecordsOf(t, c) {
		if l.level != slog.LevelWarn || l.msg != "configuration warning" {
			continue
		}
		if attrValue(l, "rule") == "no_shield" && attrValue(l, "path") == "modules.credit.shield_ticks" {
			found++
		}
	}
	if found != 1 {
		t.Errorf("the construction log has the warning no_shield %d times, want 1 at modules.credit.shield_ticks", found)
	}
}

// attrValue returns the string value of an attribute of a log line, or "".
func attrValue(l logLine, key string) string {
	for _, a := range l.attrs {
		if a.Key == key {
			return a.Value.String()
		}
	}
	return ""
}

// creditFuzzSeed encodes the eleven numbers of the module and of the two blocks, in
// the order creditFuzzConfig reads them, as the bytes of the fuzz argument.
func creditFuzzSeed(v ...int64) []byte {
	b := make([]byte, 0, 8*len(v))
	for _, x := range v {
		b = binary.BigEndian.AppendUint64(b, uint64(x))
	}
	return b
}

// creditFuzzConfig sets the eleven fields of the module and of a lender and a
// borrower block of creditConfig from the bytes b, eight bytes each in the order
// MinPrincipal, ShieldTicks, WriteOffTicks, MaxClaims, RateBP, MaxTermTicks,
// KeepTicks, ShareBP, MaxRateBP, MaxDebtTicks, SlackTicks; a short b is padded with 0.
func creditFuzzConfig(b []byte) economysim.Config {
	b = append(b, make([]byte, 88)...)
	n := func(i int) int64 { return int64(binary.BigEndian.Uint64(b[8*i:])) }
	c := creditConfig()
	*c.Modules.Credit = economysim.Credit{MinPrincipal: n(0), ShieldTicks: n(1), WriteOffTicks: n(2), MaxClaims: n(3)}
	*creditRole(&c, "cook").Credit.Lend = economysim.LendTerms{RateBP: n(4), MaxTermTicks: n(5), KeepTicks: n(6), ShareBP: n(7)}
	*creditRole(&c, "grower").Credit.Borrow = economysim.BorrowTerms{MaxRateBP: n(8), MaxDebtTicks: n(9), SlackTicks: n(10)}
	return c
}

// FuzzCreditValidate verifies, for any values of the module and of a lender and a
// borrower block, that Validate and Warnings never panic, that an accepted
// configuration is within every bound of spec-credit.md 14 and that its hash is
// stable.
//
// Requirement: FR-CRD-010, FR-CRD-011
func FuzzCreditValidate(f *testing.F) {
	f.Add(creditFuzzSeed(5_000, 2, 0, 1_000, 600, 90, 3, 5_000, 1_500, 90, 4))
	f.Add(creditFuzzSeed(0, -1, 1<<62, 0, -1, 0, 0, 0, -1, 0, -1))
	f.Fuzz(func(t *testing.T, b []byte) {
		c := creditFuzzConfig(b)
		err := c.Validate()
		_ = c.Warnings()
		if err == nil && !creditInBounds(c) {
			t.Fatalf("Validate accepted a configuration outside the bounds: %+v", c.Modules.Credit)
		}
		if first, second := economysim.HashConfig(c), economysim.HashConfig(c); first != second {
			t.Fatalf("HashConfig is not stable")
		}
	})
}

// creditInBounds reports whether every field of the credit module and of the two
// blocks of creditConfig is within its bound of spec-credit.md 14.
func creditInBounds(c economysim.Config) bool {
	m, l, b := c.Modules.Credit, creditRole(&c, "cook").Credit.Lend, creditRole(&c, "grower").Credit.Borrow
	for _, f := range []fieldRange{
		{"min_principal", m.MinPrincipal, 1, economysim.MaxMoney}, {"shield_ticks", m.ShieldTicks, 0, economysim.MaxCreditTerm},
		{"write_off_ticks", m.WriteOffTicks, 0, economysim.MaxCreditTerm}, {"max_claims", m.MaxClaims, 1, economysim.MaxOpenClaims},
		{"rate_bp", l.RateBP, 0, economysim.MaxCreditRateBP}, {"max_term_ticks", l.MaxTermTicks, 1, economysim.MaxCreditTerm},
		{"keep_ticks", l.KeepTicks, 0, economysim.MaxCreditTerm}, {"share_bp", l.ShareBP, 1, 10_000},
		{"max_rate_bp", b.MaxRateBP, 0, economysim.MaxCreditRateBP}, {"max_debt_ticks", b.MaxDebtTicks, 1, economysim.MaxCreditTerm},
		{"slack_ticks", b.SlackTicks, 0, economysim.MaxCreditTerm},
	} {
		if f.value < f.lo || f.value > f.hi {
			return false
		}
	}
	return true
}

// fieldRange is a field name, its value and its bounds, for creditInBounds.
type fieldRange struct {
	name          string
	value, lo, hi int64
}

// TestValidateCredit_BoundEdges verifies both edges of every bounded field of the
// module and its blocks: the smallest and the largest valid value validate, and the
// value one step outside refuses with ErrOutOfBounds at the field's path. The field
// cases of TestValidateCredit_Errors hold one failing side of each field; this holds
// the other side and the accepted edges, so an off-by-one in a bound is caught.
//
// Requirement: FR-CRD-010
func TestValidateCredit_BoundEdges(t *testing.T) {
	t.Parallel()
	const maxTerm, maxRate = economysim.MaxCreditTerm, economysim.MaxCreditRateBP
	lend := func(f func(l *economysim.LendTerms)) func(c *economysim.Config) {
		return func(c *economysim.Config) { f(creditRole(c, "cook").Credit.Lend) }
	}
	borrow := func(f func(b *economysim.BorrowTerms)) func(c *economysim.Config) {
		return func(c *economysim.Config) { f(creditRole(c, "grower").Credit.Borrow) }
	}
	module := func(f func(m *economysim.Credit)) func(c *economysim.Config) {
		return func(c *economysim.Config) { f(c.Modules.Credit) }
	}
	tests := []struct {
		name string
		path string
		set  func(c *economysim.Config)
		ok   bool
	}{
		{"MinPrincipal at 1", "modules.credit.min_principal", module(func(m *economysim.Credit) { m.MinPrincipal = 1 }), true},
		{"MinPrincipal at MaxMoney", "modules.credit.min_principal", module(func(m *economysim.Credit) { m.MinPrincipal = economysim.MaxMoney }), true},
		{"ShieldTicks at 0", "modules.credit.shield_ticks", module(func(m *economysim.Credit) { m.ShieldTicks = 0 }), true},
		{"ShieldTicks at the maximum", "modules.credit.shield_ticks", module(func(m *economysim.Credit) { m.ShieldTicks = maxTerm }), true},
		{"ShieldTicks below 0", "modules.credit.shield_ticks", module(func(m *economysim.Credit) { m.ShieldTicks = -1 }), false},
		{"WriteOffTicks at the maximum", "modules.credit.write_off_ticks", module(func(m *economysim.Credit) { m.WriteOffTicks = maxTerm }), true},
		{"WriteOffTicks above the maximum", "modules.credit.write_off_ticks", module(func(m *economysim.Credit) { m.WriteOffTicks = maxTerm + 1 }), false},
		{"MaxClaims at 1", "modules.credit.max_claims", module(func(m *economysim.Credit) { m.MaxClaims = 1 }), true},
		{"MaxClaims at MaxOpenClaims", "modules.credit.max_claims", module(func(m *economysim.Credit) { m.MaxClaims = economysim.MaxOpenClaims }), true},
		{"RateBP at 0", "roles.cook.credit.lend.rate_bp", lend(func(l *economysim.LendTerms) { l.RateBP = 0 }), true},
		{"RateBP at the maximum", "roles.cook.credit.lend.rate_bp", lend(func(l *economysim.LendTerms) { l.RateBP = maxRate }), true},
		{"RateBP below 0", "roles.cook.credit.lend.rate_bp", lend(func(l *economysim.LendTerms) { l.RateBP = -1 }), false},
		{"MaxTermTicks at 1", "roles.cook.credit.lend.max_term_ticks", lend(func(l *economysim.LendTerms) { l.MaxTermTicks = 1 }), true},
		{"MaxTermTicks at the maximum", "roles.cook.credit.lend.max_term_ticks", lend(func(l *economysim.LendTerms) { l.MaxTermTicks = maxTerm }), true},
		{"MaxTermTicks above the maximum", "roles.cook.credit.lend.max_term_ticks", lend(func(l *economysim.LendTerms) { l.MaxTermTicks = maxTerm + 1 }), false},
		{"KeepTicks at 0", "roles.cook.credit.lend.keep_ticks", lend(func(l *economysim.LendTerms) { l.KeepTicks = 0 }), true},
		{"KeepTicks at the maximum", "roles.cook.credit.lend.keep_ticks", lend(func(l *economysim.LendTerms) { l.KeepTicks = maxTerm }), true},
		{"KeepTicks below 0", "roles.cook.credit.lend.keep_ticks", lend(func(l *economysim.LendTerms) { l.KeepTicks = -1 }), false},
		{"ShareBP at 1", "roles.cook.credit.lend.share_bp", lend(func(l *economysim.LendTerms) { l.ShareBP = 1 }), true},
		{"ShareBP at 10 000", "roles.cook.credit.lend.share_bp", lend(func(l *economysim.LendTerms) { l.ShareBP = 10_000 }), true},
		{"MaxRateBP at 0", "roles.grower.credit.borrow.max_rate_bp", borrow(func(b *economysim.BorrowTerms) { b.MaxRateBP = 0 }), true},
		{"MaxRateBP at the maximum", "roles.grower.credit.borrow.max_rate_bp", borrow(func(b *economysim.BorrowTerms) { b.MaxRateBP = maxRate }), true},
		{"MaxRateBP above the maximum", "roles.grower.credit.borrow.max_rate_bp", borrow(func(b *economysim.BorrowTerms) { b.MaxRateBP = maxRate + 1 }), false},
		{"MaxDebtTicks at 1", "roles.grower.credit.borrow.max_debt_ticks", borrow(func(b *economysim.BorrowTerms) { b.MaxDebtTicks = 1 }), true},
		{"MaxDebtTicks at the maximum", "roles.grower.credit.borrow.max_debt_ticks", borrow(func(b *economysim.BorrowTerms) { b.MaxDebtTicks = maxTerm }), true},
		{"MaxDebtTicks above the maximum", "roles.grower.credit.borrow.max_debt_ticks", borrow(func(b *economysim.BorrowTerms) { b.MaxDebtTicks = maxTerm + 1 }), false},
		{"SlackTicks at 0", "roles.grower.credit.borrow.slack_ticks", borrow(func(b *economysim.BorrowTerms) { b.SlackTicks = 0 }), true},
		{"SlackTicks at the maximum", "roles.grower.credit.borrow.slack_ticks", borrow(func(b *economysim.BorrowTerms) { b.SlackTicks = maxTerm }), true},
		{"SlackTicks below 0", "roles.grower.credit.borrow.slack_ticks", borrow(func(b *economysim.BorrowTerms) { b.SlackTicks = -1 }), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := creditConfig()
			tt.set(&c)
			err := c.Validate()
			if tt.ok {
				if err != nil {
					t.Errorf("Validate(%s) = %v, want nil", tt.name, err)
				}
				return
			}
			if !errors.Is(err, economysim.ErrOutOfBounds) {
				t.Fatalf("Validate(%s) = %v, want ErrOutOfBounds", tt.name, err)
			}
			var at bool
			for _, e := range unwrapAll(err) {
				var ve *economysim.ValidationError
				at = at || (errors.As(e, &ve) && strings.Join(ve.Path, ".") == tt.path)
			}
			if !at {
				t.Errorf("Validate(%s) = %v, want the error at %s", tt.name, err, tt.path)
			}
		})
	}
}
