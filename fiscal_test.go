package economysim_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the fiscal module's configuration surface (spec-fiscal.md 14, 15):
// the failing case of each rule, their order and the ten warnings. The arithmetic is
// tested in internal/fiscal, the hash in fiscal_hash_test.go and the neutrality of the
// module with it off in the golden files.

// fiscalConfig returns polityConfig with the fiscal module on: the crown (seat capital)
// taxes the cooks of north a head tax of 3 a tick above an exemption of 1 000, which is
// above their upkeep of 999 (staple 2 at 96, prepared at 628, fuel at 179), and every
// household there a rate of 500 basis points above the same exemption; it takes 1 000
// basis points of the treasury of the inland province; it takes 200 basis points of the
// sales of the seller in north; it pays the cooks 2 a tick; it buys staple at 8 000 and
// sells it at 12 000 basis points of the reference price of 96, which are the limits 76
// and 115, up to a hold of 100 with a reserve of 20. None of the module's warnings is
// raised.
func fiscalConfig() economysim.Config {
	c := polityConfig()
	north := func(role string) economysim.FlowTarget {
		return economysim.FlowTarget{Kind: economysim.TargetHouseholds, Region: "north", Role: role}
	}
	c.Modules.Fiscal = &economysim.Fiscal{
		Taxes: []economysim.Tax{
			{ID: "poll", Authority: "crown", Target: north("cook"), Amount: 3, Exempt: 1_000, Label: "Head tax"},
			{ID: "tithe", Authority: "crown", Target: north(""), RateBP: 500, Exempt: 1_000},
			{ID: "remit", Authority: "crown", Target: economysim.FlowTarget{Kind: economysim.TargetTreasury}, RateBP: 1_000},
		},
		Duties:    []economysim.Duty{{ID: "sales", Authority: "crown", Region: "north", Payer: economysim.DutySeller, RateBP: 200}},
		Payouts:   []economysim.Payout{{ID: "alms", Authority: "crown", Target: north("cook"), Amount: 2}},
		Purchases: []economysim.Purchase{{ID: "buy", Authority: "crown", Commodity: "staple", Quantity: 10, LimitBP: 8_000, Hold: 100}},
		Releases:  []economysim.Release{{ID: "sell", Authority: "crown", Commodity: "staple", Quantity: 10, LimitBP: 12_000, Reserve: 20}},
	}
	return c
}

// fiscalTolled is fiscalConfig with the transport module on (one mode, foot) and the
// edges that cross the border of the inland province, and a toll of the crown on the
// shipments that leave it.
func fiscalTolled() economysim.Config {
	c := fiscalConfig()
	c.Edges = []economysim.Edge{{From: "capital", To: "mill", Distance: 40}, {From: "mill", To: "capital", Distance: 40}}
	c.Modules.Transport = &economysim.Transport{Modes: []economysim.Mode{footMode()}}
	c.Modules.Fiscal.Tolls = []economysim.Toll{{ID: "bridge", Authority: "inland", On: economysim.TollExit, Amount: 2}}
	return c
}

// fiscalOutside is the configuration with the external module on and the realm governed
// from north, so that an import duty has an outside to meet.
func fiscalOutside() economysim.Config {
	c := polityOutsideConfig()
	c.Modules.Polity.Jurisdictions[0].Seat = "north"
	c.Modules.Fiscal = &economysim.Fiscal{
		Duties:  []economysim.Duty{{ID: "tariff", Authority: "realm", Payer: economysim.DutyBuyer, Scope: economysim.ScopeImport, RateBP: 100}},
		Payouts: []economysim.Payout{{ID: "alms", Authority: "realm", Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Region: "north", Role: "cook"}, Amount: 1}},
	}
	return c
}

// fiscalPathOf is the path of a key of a flow of a list.
// A key with a dot stands for the segments it joins.
func fiscalPathOf(list, id string, key ...string) []string {
	p := []string{"modules", "fiscal", list, id}
	for _, k := range key {
		p = append(p, strings.Split(k, ".")...)
	}
	return p
}

// fisSet is the change of a case: it gets the configuration and its module.
type fisSet func(c *economysim.Config, m *economysim.Fiscal)

// fisCase builds one failing case that starts from fiscalConfig.
func fisCase(field, name string, rule error, p []string, value, bound string, set fisSet) fieldCase {
	return fisCaseFrom(fiscalConfig, field, name, rule, p, value, bound, set)
}

// fisCaseFrom builds one failing case that starts from the configuration start builds.
func fisCaseFrom(start func() economysim.Config, field, name string, rule error, p []string, value, bound string, set fisSet) fieldCase {
	return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{p}, value: value, bound: bound,
		mutate: func(c *economysim.Config) {
			*c = start()
			set(c, c.Modules.Fiscal)
		}}
}

// TestValidateFiscal_ConfigIsValid verifies that the configurations the fiscal tests
// start from validate and raise none of the module's warnings, so every failing case
// breaks one rule.
//
// Guard: the field cases start from these configurations; if one did not validate,
// each case would pass for the wrong reason.
func TestValidateFiscal_ConfigIsValid(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]economysim.Config{"fiscalConfig": fiscalConfig(), "fiscalTolled": fiscalTolled(), "fiscalOutside": fiscalOutside()} {
		if err := c.Validate(); err != nil {
			t.Errorf("%s().Validate() = %v, want nil", name, err)
		}
		for _, w := range c.Warnings() {
			if slices.Contains(fiscalWarningRules, w.Rule) {
				t.Errorf("%s().Warnings() has %v, want none of the module's", name, w)
			}
		}
	}
}

// fiscalWarningRules are the ten warnings of spec-fiscal.md 15.2.
var fiscalWarningRules = []string{
	"no_fiscal_flows", "treasury_unfunded", "treasury_idle", "tax_below_upkeep", "payout_pool_no_outlet",
	"purchase_unbounded", "release_unsourced", "duty_stack_above_whole", "toll_no_border", "fiscal_class_empty",
}

// fiscalFieldCases are the failing cases of the rows of the fiscal module in the
// parameter-owner table (spec-fiscal.md 14, 15.1, FR-FIS-004, 007, 008, 011, 012): each
// breaks one field of the valid configuration of fiscalConfig. buildFieldCases appends
// them to the table that the registry test reads.
func fiscalFieldCases() []fieldCase {
	var cs []fieldCase
	cs = append(cs, fisCountCases()...)
	cs = append(cs, fisIdentityCases()...)
	cs = append(cs, fisWindowCases()...)
	cs = append(cs, fisTaxCases()...)
	cs = append(cs, fisDutyTollCases()...)
	cs = append(cs, fisPayoutCases()...)
	return append(cs, fisOrderCases()...)
}

// fisMany returns n flows of a list with IDs f00, f01, ..., each a copy of the given
// flow; set gives the copy its ID.
func fisMany[T any](n int, flow T, set func(*T, string)) []T {
	out := make([]T, n)
	for i := range out {
		out[i] = flow
		set(&out[i], "f"+strconv.Itoa(i/10)+strconv.Itoa(i%10))
	}
	return out
}

// fisCountCases are the cases of the six lists: more than MaxFiscalFlows flows of all
// kinds together is ErrTooMany.
func fisCountCases() []fieldCase {
	base := []string{"modules", "fiscal"}
	over := economysim.MaxFiscalFlows + 1
	cases := []struct {
		field string
		set   fisSet
	}{
		{"Fiscal.Taxes", func(_ *economysim.Config, m *economysim.Fiscal) {
			m.Taxes = fisMany(over, m.Taxes[0], func(t *economysim.Tax, id string) { t.ID = id })
		}},
		{"Fiscal.Duties", func(_ *economysim.Config, m *economysim.Fiscal) {
			m.Duties = fisMany(over, m.Duties[0], func(d *economysim.Duty, id string) { d.ID = id })
		}},
		{"Fiscal.Tolls", func(_ *economysim.Config, m *economysim.Fiscal) {
			m.Tolls = fisMany(over, economysim.Toll{Authority: "crown", On: economysim.TollExit, Amount: 1}, func(t *economysim.Toll, id string) { t.ID = id })
		}},
		{"Fiscal.Payouts", func(_ *economysim.Config, m *economysim.Fiscal) {
			m.Payouts = fisMany(over, m.Payouts[0], func(p *economysim.Payout, id string) { p.ID = id })
		}},
		{"Fiscal.Purchases", func(_ *economysim.Config, m *economysim.Fiscal) {
			m.Purchases = fisMany(over, m.Purchases[0], func(p *economysim.Purchase, id string) { p.ID = id })
		}},
		{"Fiscal.Releases", func(_ *economysim.Config, m *economysim.Fiscal) {
			m.Releases = fisMany(over, m.Releases[0], func(r *economysim.Release, id string) { r.ID = id })
		}},
	}
	var out []fieldCase
	for _, tt := range cases {
		start := fiscalConfig
		if tt.field == "Fiscal.Tolls" {
			start = fiscalTolled
		}
		out = append(out, fisCaseFrom(start, tt.field, "above MaxFiscalFlows", economysim.ErrTooMany, base, "", "64", tt.set))
	}
	return out
}

// fisIdentityCases are the cases of the ID and the authority of each kind: the syntax
// of an ID, its place in the one namespace of the six lists and the money flows, and an
// authority that is not defined or has no seat.
func fisIdentityCases() []fieldCase {
	bad := func(field, list, id string, set fisSet) fieldCase {
		return fisCaseFrom(fiscalTolled, field, "upper case letter", economysim.ErrInvalidID, fiscalPathOf(list, "Bad", "id"), "Bad", "", set)
	}
	unknown := func(field, list, id string, set fisSet) fieldCase {
		return fisCaseFrom(fiscalTolled, field, "an authority that is not defined", economysim.ErrUnknownJurisdiction,
			fiscalPathOf(list, id, "authority"), "nowhere", "", set)
	}
	return []fieldCase{
		bad("Tax.ID", "taxes", "poll", func(_ *economysim.Config, m *economysim.Fiscal) { m.Taxes[0].ID = "Bad" }),
		bad("Duty.ID", "duties", "sales", func(_ *economysim.Config, m *economysim.Fiscal) { m.Duties[0].ID = "Bad" }),
		bad("Toll.ID", "tolls", "bridge", func(_ *economysim.Config, m *economysim.Fiscal) { m.Tolls[0].ID = "Bad" }),
		bad("Payout.ID", "payouts", "alms", func(_ *economysim.Config, m *economysim.Fiscal) { m.Payouts[0].ID = "Bad" }),
		bad("Purchase.ID", "purchases", "buy", func(_ *economysim.Config, m *economysim.Fiscal) { m.Purchases[0].ID = "Bad" }),
		bad("Release.ID", "releases", "sell", func(_ *economysim.Config, m *economysim.Fiscal) { m.Releases[0].ID = "Bad" }),
		fisCase("Tax.ID", "the ID of a duty", economysim.ErrDuplicateID, fiscalPathOf("duties", "poll", "id"), "poll", "",
			func(_ *economysim.Config, m *economysim.Fiscal) { m.Duties[0].ID = "poll" }),
		fisCase("Payout.ID", "a built-in reason", economysim.ErrReasonCollision, fiscalPathOf("payouts", "trade", "id"), "trade", "",
			func(_ *economysim.Config, m *economysim.Fiscal) { m.Payouts[0].ID = "trade" }),
		fisCase("Duty.ID", "the ID of a money flow", economysim.ErrReasonCollision, fiscalPathOf("duties", "levy", "id"), "levy", "",
			func(c *economysim.Config, m *economysim.Fiscal) {
				m.Duties[0].ID = "levy"
				c.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{{ID: "levy", Kind: economysim.FlowLevy,
					Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: "cook"}, Amount: 1}}}
			}),
		unknown("Tax.Authority", "taxes", "poll", func(_ *economysim.Config, m *economysim.Fiscal) { m.Taxes[0].Authority = "nowhere" }),
		unknown("Duty.Authority", "duties", "sales", func(_ *economysim.Config, m *economysim.Fiscal) { m.Duties[0].Authority = "nowhere" }),
		unknown("Toll.Authority", "tolls", "bridge", func(_ *economysim.Config, m *economysim.Fiscal) { m.Tolls[0].Authority = "nowhere" }),
		unknown("Payout.Authority", "payouts", "alms", func(_ *economysim.Config, m *economysim.Fiscal) { m.Payouts[0].Authority = "nowhere" }),
		unknown("Purchase.Authority", "purchases", "buy", func(_ *economysim.Config, m *economysim.Fiscal) { m.Purchases[0].Authority = "nowhere" }),
		unknown("Release.Authority", "releases", "sell", func(_ *economysim.Config, m *economysim.Fiscal) { m.Releases[0].Authority = "nowhere" }),
		fisCase("Tax.Authority", "a grouping with no seat", economysim.ErrNotGoverned, fiscalPathOf("taxes", "poll", "authority"), "coast", "",
			func(_ *economysim.Config, m *economysim.Fiscal) { m.Taxes[0].Authority = "coast" }),
	}
}

// fisWindowCases are the cases of the window of each kind.
func fisWindowCases() []fieldCase {
	season := func(field, list, id string, set fisSet) fieldCase {
		return fisCaseFrom(fiscalTolled, field, "a season the calendar lacks", economysim.ErrUnknownSeason,
			fiscalPathOf(list, id, "window", "seasons[0]"), "monsoon", "", set)
	}
	monsoon := economysim.Window{Seasons: []string{"monsoon"}}
	return []fieldCase{
		season("Tax.Window", "taxes", "poll", func(_ *economysim.Config, m *economysim.Fiscal) { m.Taxes[0].Window = monsoon }),
		season("Duty.Window", "duties", "sales", func(_ *economysim.Config, m *economysim.Fiscal) { m.Duties[0].Window = monsoon }),
		season("Toll.Window", "tolls", "bridge", func(_ *economysim.Config, m *economysim.Fiscal) { m.Tolls[0].Window = monsoon }),
		season("Payout.Window", "payouts", "alms", func(_ *economysim.Config, m *economysim.Fiscal) { m.Payouts[0].Window = monsoon }),
		season("Purchase.Window", "purchases", "buy", func(_ *economysim.Config, m *economysim.Fiscal) { m.Purchases[0].Window = monsoon }),
		season("Release.Window", "releases", "sell", func(_ *economysim.Config, m *economysim.Fiscal) { m.Releases[0].Window = monsoon }),
		fisCase("Window.Start", "below 0", economysim.ErrOutOfBounds, fiscalPathOf("taxes", "poll", "window", "start"), "-1", "0",
			func(_ *economysim.Config, m *economysim.Fiscal) { m.Taxes[0].Window.Start = -1 }),
		fisCase("Window.Until", "not above Start", economysim.ErrOutOfBounds, fiscalPathOf("taxes", "poll", "window", "until"), "10", "11",
			func(_ *economysim.Config, m *economysim.Fiscal) {
				m.Taxes[0].Window = economysim.Window{Start: 10, Until: 10}
			}),
		fisCase("Window.Seasons", "a season twice", economysim.ErrDuplicateID, fiscalPathOf("taxes", "poll", "window", "seasons[1]"), "spring", "",
			func(_ *economysim.Config, m *economysim.Fiscal) {
				m.Taxes[0].Window.Seasons = []string{"spring", "spring"}
			}),
	}
}

// fisTaxCases are the cases of the fields of a tax.
func fisTaxCases() []fieldCase {
	tax := func(field, name string, rule error, key, value, bound string, set func(t *economysim.Tax)) fieldCase {
		return fisCase(field, name, rule, fiscalPathOf("taxes", "poll", key), value, bound,
			func(_ *economysim.Config, m *economysim.Fiscal) { set(&m.Taxes[0]) })
	}
	return []fieldCase{
		tax("Tax.Amount", "with a rate", economysim.ErrFiscalShape, "amount", "3", "", func(t *economysim.Tax) { t.RateBP = 100 }),
		tax("Tax.Amount", "with neither", economysim.ErrFiscalShape, "amount", "0", "", func(t *economysim.Tax) { t.Amount = 0 }),
		tax("Tax.Amount", "above MaxFlowAmount", economysim.ErrOutOfBounds, "amount", "1000000000001", "1000000000000",
			func(t *economysim.Tax) { t.Amount = economysim.MaxFlowAmount + 1 }),
		tax("Tax.RateBP", "above 10 000", economysim.ErrOutOfBounds, "rate_bp", "10001", "10000",
			func(t *economysim.Tax) { t.Amount, t.RateBP = 0, 10_001 }),
		tax("Tax.Exempt", "above MaxFlowAmount", economysim.ErrOutOfBounds, "exempt", "1000000000001", "1000000000000",
			func(t *economysim.Tax) { t.Exempt = economysim.MaxFlowAmount + 1 }),
		tax("Tax.Per", "a period that is neither", economysim.ErrOutOfBounds, "per", "week", "", func(t *economysim.Tax) { t.Per = "week" }),
		tax("Tax.Label", "over MaxLabelBytes", economysim.ErrOutOfBounds, "label", "257", "256",
			func(t *economysim.Tax) { t.Label = strings.Repeat("x", economysim.MaxLabelBytes+1) }),
		tax("Tax.Target", "a role that is not defined", economysim.ErrUnknownRole, "target.role", "nowhere", "",
			func(t *economysim.Tax) { t.Target.Role = "nowhere" }),
		tax("Tax.Target", "a kind that is not one", economysim.ErrOutOfBounds, "target.kind", "stores", "",
			func(t *economysim.Tax) { t.Target.Kind = "stores" }),
		fisCase("Tax.Target", "a pool with a role", economysim.ErrFiscalShape, fiscalPathOf("taxes", "poll", "target.role"), "cook", "",
			func(c *economysim.Config, m *economysim.Fiscal) {
				c.Modules.Estates = &economysim.Estates{}
				m.Taxes[0].Target.Kind, m.Taxes[0].Target.Region = economysim.TargetPool, ""
			}),
		fisCase("Tax.Target", "a pool with the estates module off", economysim.ErrModuleDependency, fiscalPathOf("taxes", "poll", "target", "kind"), "pool", "estates",
			func(_ *economysim.Config, m *economysim.Fiscal) {
				m.Taxes[0].Target = economysim.FlowTarget{Kind: economysim.TargetPool}
			}),
		tax("Tax.Target", "a region the authority does not cover", economysim.ErrNotCovered, "target.region", "harbour", "",
			func(t *economysim.Tax) { t.Authority, t.Target.Region = "inland", "harbour" }),
		tax("Tax.Target", "a region that is not defined", economysim.ErrUnknownRegion, "target.region", "nowhere", "",
			func(t *economysim.Tax) { t.Target.Region = "nowhere" }),
		fisCase("FlowTarget.Jurisdiction", "a treasury that is not below the authority", economysim.ErrNotCovered,
			fiscalPathOf("taxes", "remit", "target", "jurisdiction"), "crown", "",
			func(_ *economysim.Config, m *economysim.Fiscal) { m.Taxes[2].Target.Jurisdiction = "crown" }),
		fisCase("FlowTarget.Jurisdiction", "a jurisdiction that is not defined", economysim.ErrUnknownJurisdiction,
			fiscalPathOf("taxes", "remit", "target", "jurisdiction"), "nowhere", "",
			func(_ *economysim.Config, m *economysim.Fiscal) { m.Taxes[2].Target.Jurisdiction = "nowhere" }),
		fisCase("FlowTarget.Jurisdiction", "a treasury with no seat", economysim.ErrNotGoverned,
			fiscalPathOf("taxes", "remit", "target", "jurisdiction"), "uplands", "",
			func(_ *economysim.Config, m *economysim.Fiscal) { m.Taxes[2].Target.Jurisdiction = "uplands" }),
		fisCase("FlowTarget.Jurisdiction", "households with a jurisdiction", economysim.ErrFiscalShape,
			fiscalPathOf("taxes", "poll", "target", "jurisdiction"), "inland", "",
			func(_ *economysim.Config, m *economysim.Fiscal) { m.Taxes[0].Target.Jurisdiction = "inland" }),
	}
}

// fisDutyTollCases are the cases of the fields of a duty and of a toll.
func fisDutyTollCases() []fieldCase {
	duty := func(field, name string, rule error, key, value, bound string, set func(d *economysim.Duty)) fieldCase {
		return fisCase(field, name, rule, fiscalPathOf("duties", "sales", key), value, bound,
			func(_ *economysim.Config, m *economysim.Fiscal) { set(&m.Duties[0]) })
	}
	toll := func(field, name string, rule error, key, value, bound string, set func(t *economysim.Toll)) fieldCase {
		return fisCaseFrom(fiscalTolled, field, name, rule, fiscalPathOf("tolls", "bridge", key), value, bound,
			func(_ *economysim.Config, m *economysim.Fiscal) { set(&m.Tolls[0]) })
	}
	return []fieldCase{
		duty("Duty.Region", "a region that is not defined", economysim.ErrUnknownRegion, "region", "nowhere", "", func(d *economysim.Duty) { d.Region = "nowhere" }),
		duty("Duty.Region", "a region the authority does not cover", economysim.ErrNotCovered, "region", "harbour", "",
			func(d *economysim.Duty) { d.Authority, d.Region = "inland", "harbour" }),
		duty("Duty.Commodity", "a commodity that is not defined", economysim.ErrUnknownCommodity, "commodity", "nowhere", "",
			func(d *economysim.Duty) { d.Commodity = "nowhere" }),
		duty("Duty.Payer", "neither seller nor buyer", economysim.ErrOutOfBounds, "payer", "both", "", func(d *economysim.Duty) { d.Payer = "both" }),
		duty("Duty.Scope", "a scope that is not one", economysim.ErrOutOfBounds, "scope", "abroad", "", func(d *economysim.Duty) { d.Scope = "abroad" }),
		duty("Duty.Scope", "import with the external module off", economysim.ErrModuleDependency, "scope", "import", "external",
			func(d *economysim.Duty) { d.Scope = economysim.ScopeImport }),
		duty("Duty.RateBP", "0", economysim.ErrOutOfBounds, "rate_bp", "0", "1", func(d *economysim.Duty) { d.RateBP = 0 }),
		duty("Duty.RateBP", "above MaxDutyBP", economysim.ErrOutOfBounds, "rate_bp", "10001", "10000", func(d *economysim.Duty) { d.RateBP = 10_001 }),
		fisCaseFrom(fiscalOutside, "Duty.Payer", "import with the seller", economysim.ErrFiscalShape, fiscalPathOf("duties", "tariff", "scope"), "import", "",
			func(_ *economysim.Config, m *economysim.Fiscal) { m.Duties[0].Payer = economysim.DutySeller }),
		fisCaseFrom(fiscalOutside, "Duty.Scope", "export with the buyer", economysim.ErrFiscalShape, fiscalPathOf("duties", "tariff", "scope"), "export", "",
			func(_ *economysim.Config, m *economysim.Fiscal) { m.Duties[0].Scope = economysim.ScopeExport }),
		toll("Toll.On", "neither exit nor enter", economysim.ErrOutOfBounds, "on", "pass", "", func(t *economysim.Toll) { t.On = "pass" }),
		toll("Toll.Mode", "a mode that is not defined", economysim.ErrUnknownMode, "mode", "nowhere", "", func(t *economysim.Toll) { t.Mode = "nowhere" }),
		toll("Toll.Amount", "0", economysim.ErrOutOfBounds, "amount", "0", "1", func(t *economysim.Toll) { t.Amount = 0 }),
		fisCase("Toll.Amount", "the transport module off", economysim.ErrModuleDependency, []string{"modules", "fiscal", "tolls"}, "transport", "",
			func(_ *economysim.Config, m *economysim.Fiscal) {
				m.Tolls = []economysim.Toll{{ID: "bridge", Authority: "crown", On: economysim.TollExit, Amount: 1}}
			}),
	}
}

// fisPayoutCases are the cases of the fields of a payout.
func fisPayoutCases() []fieldCase {
	payout := func(field, name string, rule error, key, value, bound string, set func(p *economysim.Payout)) fieldCase {
		return fisCase(field, name, rule, fiscalPathOf("payouts", "alms", key), value, bound,
			func(_ *economysim.Config, m *economysim.Fiscal) { set(&m.Payouts[0]) })
	}
	inKind := func(p *economysim.Payout) {
		p.Commodity, p.Target.Region, p.Target.Role = "staple", "capital", ""
	}
	return []fieldCase{
		payout("Payout.Target", "a treasury with a role", economysim.ErrFiscalShape, "target.role", "cook", "",
			func(p *economysim.Payout) { p.Target.Kind, p.Target.Region = economysim.TargetTreasury, "" }),
		payout("Payout.Target", "an in-kind payout outside the seat region", economysim.ErrFiscalShape, "target", "north", "",
			func(p *economysim.Payout) { inKind(p); p.Target.Region = "north" }),
		payout("Payout.Commodity", "a commodity that is not defined", economysim.ErrUnknownCommodity, "commodity", "nowhere", "",
			func(p *economysim.Payout) { inKind(p); p.Commodity = "nowhere" }),
		payout("Payout.Amount", "with a level", economysim.ErrFiscalShape, "amount", "2", "", func(p *economysim.Payout) { p.Level = 10 }),
		payout("Payout.Amount", "with neither", economysim.ErrFiscalShape, "amount", "0", "", func(p *economysim.Payout) { p.Amount = 0 }),
		payout("Payout.Amount", "above MaxFlowAmount", economysim.ErrOutOfBounds, "amount", "1000000000001", "1000000000000",
			func(p *economysim.Payout) { p.Amount = economysim.MaxFlowAmount + 1 }),
		payout("Payout.Level", "with the period year", economysim.ErrFiscalShape, "level", "10", "",
			func(p *economysim.Payout) { p.Amount, p.Level, p.Per = 0, 10, economysim.PerYear }),
		payout("Payout.Level", "above MaxFlowAmount", economysim.ErrOutOfBounds, "level", "1000000000001", "1000000000000",
			func(p *economysim.Payout) { p.Amount, p.Level = 0, economysim.MaxFlowAmount+1 }),
		payout("Payout.Per", "a period that is neither", economysim.ErrOutOfBounds, "per", "week", "", func(p *economysim.Payout) { p.Per = "week" }),
		payout("Payout.NotIncome", "an in-kind payout", economysim.ErrFiscalShape, "not_income", "true", "",
			func(p *economysim.Payout) { inKind(p); p.NotIncome = true }),
		payout("Payout.Label", "over MaxLabelBytes", economysim.ErrOutOfBounds, "label", "257", "256",
			func(p *economysim.Payout) { p.Label = strings.Repeat("x", economysim.MaxLabelBytes+1) }),
	}
}

// fisOrderCases are the cases of the fields of a purchase and of a release.
func fisOrderCases() []fieldCase {
	buy := func(field, name string, rule error, key, value, bound string, set func(p *economysim.Purchase)) fieldCase {
		return fisCase(field, name, rule, fiscalPathOf("purchases", "buy", key), value, bound,
			func(_ *economysim.Config, m *economysim.Fiscal) { set(&m.Purchases[0]) })
	}
	sell := func(field, name string, rule error, key, value, bound string, set func(r *economysim.Release)) fieldCase {
		return fisCase(field, name, rule, fiscalPathOf("releases", "sell", key), value, bound,
			func(_ *economysim.Config, m *economysim.Fiscal) { set(&m.Releases[0]) })
	}
	return []fieldCase{
		buy("Purchase.Commodity", "a commodity that is not defined", economysim.ErrUnknownCommodity, "commodity", "nowhere", "",
			func(p *economysim.Purchase) { p.Commodity = "nowhere" }),
		buy("Purchase.Quantity", "0", economysim.ErrOutOfBounds, "quantity", "0", "1", func(p *economysim.Purchase) { p.Quantity = 0 }),
		buy("Purchase.LimitBP", "above MaxLimitBP", economysim.ErrOutOfBounds, "limit_bp", "100001", "100000", func(p *economysim.Purchase) { p.LimitBP = 100_001 }),
		fisCase("Purchase.Hold", "below Quantity", economysim.ErrFiscalShape, fiscalPathOf("purchases", "buy", "hold"), "5", "",
			func(_ *economysim.Config, m *economysim.Fiscal) { m.Purchases[0].Hold, m.Releases[0].Reserve = 5, 0 }),
		sell("Release.Commodity", "a commodity that is not defined", economysim.ErrUnknownCommodity, "commodity", "nowhere", "",
			func(r *economysim.Release) { r.Commodity = "nowhere" }),
		sell("Release.Quantity", "0", economysim.ErrOutOfBounds, "quantity", "0", "1", func(r *economysim.Release) { r.Quantity = 0 }),
		sell("Release.LimitBP", "0", economysim.ErrOutOfBounds, "limit_bp", "0", "1", func(r *economysim.Release) { r.LimitBP = 0 }),
		sell("Release.LimitBP", "limits that are equal on the integers", economysim.ErrFiscalShape, "limit_bp", "8010", "",
			func(r *economysim.Release) { r.LimitBP = 8_010 }),
		sell("Release.Reserve", "above the hold of the purchase", economysim.ErrFiscalShape, "reserve", "101", "",
			func(r *economysim.Release) { r.Reserve = 101 }),
		fisCase("Purchase.Commodity", "two purchases of one commodity", economysim.ErrFiscalShape, fiscalPathOf("purchases", "buy2", "commodity"), "staple", "",
			func(_ *economysim.Config, m *economysim.Fiscal) {
				p := m.Purchases[0]
				p.ID = "buy2"
				m.Purchases = append(m.Purchases, p)
			}),
		fisCase("Release.Commodity", "two releases of one commodity", economysim.ErrFiscalShape, fiscalPathOf("releases", "sell2", "commodity"), "staple", "",
			func(_ *economysim.Config, m *economysim.Fiscal) {
				r := m.Releases[0]
				r.ID = "sell2"
				m.Releases = append(m.Releases, r)
			}),
	}
}
