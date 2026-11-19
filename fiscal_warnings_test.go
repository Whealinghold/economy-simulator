package economysim_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestFiscalWarnings verifies, for each of the ten warnings of spec-fiscal.md 15.2, a
// configuration that raises it once at the path of the flow (or of the module) and the
// quiet case beside it, and that no warning fails validation.
//
// Requirement: FR-FIS-013
func TestFiscalWarnings(t *testing.T) {
	t.Parallel()
	north := economysim.FlowTarget{Kind: economysim.TargetHouseholds, Region: "north", Role: "cook"}
	tests := []struct {
		name   string
		start  func() economysim.Config
		set    func(c *economysim.Config, m *economysim.Fiscal)
		rule   string
		path   []string
		quiet  func(c *economysim.Config, m *economysim.Fiscal)
		quietS string
	}{
		{"no flow at all", fiscalConfig, func(_ *economysim.Config, m *economysim.Fiscal) { *m = economysim.Fiscal{} },
			"no_fiscal_flows", []string{"modules", "fiscal"}, nil, ""},
		{"a payout and nothing that funds the treasury", fiscalConfig, func(c *economysim.Config, m *economysim.Fiscal) {
			m.Taxes, m.Duties, m.Releases = nil, nil, nil
			c.Modules.Polity.Jurisdictions[0].StartMoney = 0
		}, "treasury_unfunded", fiscalPathOf("payouts", "alms"),
			func(c *economysim.Config, _ *economysim.Fiscal) { c.Modules.Polity.Jurisdictions[0].StartMoney = 1 }, "a start money"},
		{"a tax and no outlet", fiscalConfig, func(_ *economysim.Config, m *economysim.Fiscal) {
			m.Payouts, m.Purchases, m.Releases = nil, nil, nil
		}, "treasury_idle", fiscalPathOf("taxes", "poll"),
			func(_ *economysim.Config, m *economysim.Fiscal) {
				m.Payouts = []economysim.Payout{{ID: "alms", Authority: "crown", Target: north, Amount: 2}}
			}, "a payout"},
		{"an exemption below the upkeep", fiscalConfig, func(_ *economysim.Config, m *economysim.Fiscal) { m.Taxes[0].Exempt = 998 },
			"tax_below_upkeep", fiscalPathOf("taxes", "poll"),
			func(_ *economysim.Config, m *economysim.Fiscal) { m.Taxes[0].Exempt = 999 }, "an exemption equal to the upkeep of 999"},
		{"a pool payout with no pool release", fiscalConfig, func(c *economysim.Config, m *economysim.Fiscal) {
			c.Modules.Estates = &economysim.Estates{}
			c.Regions[0].PoolRelease = 0
			m.Payouts = []economysim.Payout{{ID: "aid", Authority: "crown", Target: economysim.FlowTarget{Kind: economysim.TargetPool, Region: "north"}, Amount: 5}}
		}, "payout_pool_no_outlet", fiscalPathOf("payouts", "aid"),
			func(c *economysim.Config, _ *economysim.Fiscal) { c.Regions[0].PoolRelease = 100 }, "a pool release"},
		{"a purchase of a good nothing draws on, with no hold", fiscalConfig, func(_ *economysim.Config, m *economysim.Fiscal) {
			m.Purchases[0].Commodity, m.Purchases[0].Hold = "waste", 0
		}, "purchase_unbounded", fiscalPathOf("purchases", "buy"),
			func(_ *economysim.Config, m *economysim.Fiscal) { m.Purchases[0].Hold = 100 }, "a hold"},
		{"a release with no purchase", fiscalConfig, func(_ *economysim.Config, m *economysim.Fiscal) { m.Purchases = nil },
			"release_unsourced", fiscalPathOf("releases", "sell"), nil, ""},
		{"duties of one payer above the whole value", fiscalConfig, func(_ *economysim.Config, m *economysim.Fiscal) {
			m.Duties[0].RateBP = 6_000
			m.Duties = append(m.Duties, economysim.Duty{ID: "more", Authority: "crown", Region: "north", Payer: economysim.DutySeller, RateBP: 6_000})
		}, "duty_stack_above_whole", fiscalPathOf("duties", "sales"),
			func(_ *economysim.Config, m *economysim.Fiscal) { m.Duties[1].Payer = economysim.DutyBuyer }, "duties of the two payers"},
		{"a toll on a border no edge crosses", fiscalTolled, func(_ *economysim.Config, m *economysim.Fiscal) { m.Tolls[0].Authority = "crown" },
			"toll_no_border", fiscalPathOf("tolls", "bridge"), nil, ""},
		{"a tax on a class with no households", fiscalConfig, func(_ *economysim.Config, m *economysim.Fiscal) { m.Taxes[0].Target.Region = "capital" },
			"fiscal_class_empty", fiscalPathOf("taxes", "poll"), nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tt.start()
			tt.set(&c, c.Modules.Fiscal)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate() = %v, want nil: a warning never fails validation", err)
			}
			if n := countWarning(c.Warnings(), tt.rule, tt.path); n != 1 {
				t.Errorf("Warnings() has %d of %s at %v, want 1: %v", n, tt.rule, tt.path, c.Warnings())
			}
			if tt.quiet == nil {
				return
			}
			tt.quiet(&c, c.Modules.Fiscal)
			if n := countWarning(c.Warnings(), tt.rule, tt.path); n != 0 {
				t.Errorf("Warnings() with %s has %d of %s, want 0", tt.quietS, n, tt.rule)
			}
		})
	}
}

// countWarning counts the warnings of a rule at a path.
func countWarning(ws []economysim.Warning, rule string, path []string) int {
	n := 0
	for _, w := range ws {
		if w.Rule == rule && slices.Equal(w.Path, path) {
			n++
		}
	}
	return n
}

// TestFiscalWarnings_DutySumSaturates verifies that two duty rates whose sum overflows
// int64 raise duty_stack_above_whole and do not wrap into a sum below the whole value.
// Warnings reads a configuration that does not validate.
//
// Requirement: FR-FIS-013
func TestFiscalWarnings_DutySumSaturates(t *testing.T) {
	t.Parallel()
	c := fiscalConfig()
	m := c.Modules.Fiscal
	m.Duties[0].RateBP = math.MaxInt64
	m.Duties = append(m.Duties, economysim.Duty{ID: "more", Authority: "crown", Region: "north", Payer: m.Duties[0].Payer, RateBP: math.MaxInt64})
	if n := countWarning(c.Warnings(), "duty_stack_above_whole", fiscalPathOf("duties", m.Duties[0].ID)); n != 1 {
		t.Errorf("Warnings() has %d of duty_stack_above_whole, want 1 for rates that overflow int64 when added", n)
	}
}

// TestFiscalWarnings_OffRaisesNone verifies that a configuration with the module off
// raises none of the ten warnings.
//
// Requirement: FR-FIS-001, FR-FIS-013
func TestFiscalWarnings_OffRaisesNone(t *testing.T) {
	t.Parallel()
	c := fiscalConfig()
	c.Modules.Fiscal = nil
	for _, w := range c.Warnings() {
		if slices.Contains(fiscalWarningRules, w.Rule) {
			t.Errorf("Warnings() with the module off has %v, want none of the module's", w)
		}
	}
}

// TestValidateFiscal_Dependencies verifies that the module is refused without the polity
// module, and with a polity that has no governed jurisdiction, with ErrModuleDependency
// at the module's path.
//
// Requirement: FR-FIS-007, FR-FIS-012
func TestValidateFiscal_Dependencies(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		set  func(c *economysim.Config)
	}{
		{"the polity module off", func(c *economysim.Config) { c.Modules.Polity = nil }},
		{"no jurisdiction", func(c *economysim.Config) { c.Modules.Polity = &economysim.PolityConfig{} }},
		{"no governed jurisdiction", func(c *economysim.Config) {
			for i := range c.Modules.Polity.Jurisdictions {
				c.Modules.Polity.Jurisdictions[i].Seat, c.Modules.Polity.Jurisdictions[i].StartMoney = "", 0
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := fiscalConfig()
			tt.set(&c)
			err := c.Validate()
			if !errors.Is(err, economysim.ErrModuleDependency) {
				t.Fatalf("Validate() = %v, want ErrModuleDependency", err)
			}
			if vs, _ := violations(err); len(vs) != 1 || !slices.Equal(vs[0].Path, []string{"modules", "fiscal"}) {
				t.Errorf("Validate() errors = %v, want one at modules.fiscal", vs)
			}
		})
	}
}

// TestValidateFiscal_Order verifies that the rules run in the order of spec-fiscal.md
// 15.1 and a phase that finds an error ends the run: of two mistakes the earlier phase
// is reported alone.
//
// Requirement: FR-FIS-012
func TestValidateFiscal_Order(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		start func() economysim.Config
		set   fisSet
		first error
		later error
	}{
		{"a dependency before an ID", fiscalConfig, func(_ *economysim.Config, m *economysim.Fiscal) {
			m.Tolls = []economysim.Toll{{ID: "Bad", Authority: "crown", On: "exit", Amount: 1}}
		}, economysim.ErrModuleDependency, economysim.ErrInvalidID},
		{"an ID before an authority", fiscalConfig, func(_ *economysim.Config, m *economysim.Fiscal) {
			m.Taxes[0].ID, m.Taxes[1].Authority = "Bad", "nowhere"
		}, economysim.ErrInvalidID, economysim.ErrUnknownJurisdiction},
		{"an authority before a window", fiscalConfig, func(_ *economysim.Config, m *economysim.Fiscal) {
			m.Taxes[0].Authority, m.Taxes[1].Window.Start = "nowhere", -1
		}, economysim.ErrUnknownJurisdiction, economysim.ErrOutOfBounds},
		{"a window before a shape", fiscalConfig, func(_ *economysim.Config, m *economysim.Fiscal) {
			m.Taxes[0].Window.Start, m.Taxes[1].RateBP, m.Taxes[1].Amount = -1, 0, 0
		}, economysim.ErrOutOfBounds, economysim.ErrFiscalShape},
		{"a shape before a class", fiscalConfig, func(_ *economysim.Config, m *economysim.Fiscal) {
			m.Taxes[0].Amount, m.Taxes[0].RateBP = 0, 0
			m.Taxes[1].Target.Role = "nowhere"
		}, economysim.ErrFiscalShape, economysim.ErrUnknownRole},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tt.start()
			tt.set(&c, c.Modules.Fiscal)
			err := c.Validate()
			if !errors.Is(err, tt.first) {
				t.Errorf("Validate() = %v, want %v", err, tt.first)
			}
			if errors.Is(err, tt.later) {
				t.Errorf("Validate() = %v, want %v not reported: a phase that finds an error ends the run", err, tt.later)
			}
		})
	}
}

// TestFiscal_ModuleOnBuildsAWorld verifies that a world with the module on builds and
// steps with the audit on: the module has declared its ledger reasons and does nothing
// else yet, so a run is the run without it.
//
// Requirement: FR-FIS-001, FR-FIS-002
func TestFiscal_ModuleOnBuildsAWorld(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	w, err := economysim.NewWorld(fiscalConfig(), economysim.WithSeed(1), economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewWorld(fiscalConfig) error = %v, want nil", err)
	}
	for range 5 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
}
