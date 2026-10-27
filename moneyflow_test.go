package economysim_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the money supply module's configuration: its validation, its
// warnings, its hash and the staged rows that name it. The step is tested with
// the module file's behaviour, not here.

// withMoney switches the module on in c, a copy of referenceConfig, with a rate
// grant of 20 a tick to the growers of the north and a rate levy of 3 650 basis
// points a year on the cooks, exempt 3 000 (the overlay faucet-sunk.toml of
// spec-money-supply.md 17.6). The exemption is above the cook's upkeep of 999
// a tick, and the levy is a rate, so no warning of the module is raised.
func withMoney(c *economysim.Config) {
	c.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{
		{ID: "charge", Kind: economysim.FlowLevy, Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: "cook"},
			RateBP: 3_650, Per: economysim.PerYear, Exempt: 3_000},
		{ID: "stipend", Kind: economysim.FlowGrant, Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: "grower", Region: "north"},
			Amount: 20},
	}}
}

// moneyFlowOf returns the flow with the ID of c, or nil.
func moneyFlowOf(c *economysim.Config, id string) *economysim.MoneyFlow {
	for i := range c.Modules.MoneySupply.Flows {
		if c.Modules.MoneySupply.Flows[i].ID == id {
			return &c.Modules.MoneySupply.Flows[i]
		}
	}
	return nil
}

// moneyFieldCases are the failing cases of the rows of the money supply module
// in the parameter-owner table (spec-money-supply.md 11.1, FR-CFG-031): each
// breaks one field of a valid configuration. buildFieldCases appends them to the
// table that the registry test reads.
func moneyFieldCases() []fieldCase {
	// mon breaks one field of the flow that the path names; an ID case renames the
	// flow, so the flow it breaks is the grant "stipend" whatever the path says.
	mon := func(field, name string, rule error, mutate func(f *economysim.MoneyFlow), id, leaf string) fieldCase {
		p := []string{"modules", "money_supply", "flows", id}
		return fieldCase{field: field, name: name, rules: []error{rule}, paths: [][]string{append(p, strings.Split(leaf, ".")...)},
			mutate: func(c *economysim.Config) {
				withMoney(c)
				if moneyFlowOf(c, id) == nil {
					id = "stipend"
				}
				mutate(moneyFlowOf(c, id))
			}}
	}
	bounds := economysim.ErrOutOfBounds
	return []fieldCase{
		{field: "MoneySupply.Flows", name: "more than MaxMoneyFlows", rules: []error{economysim.ErrTooMany},
			paths: [][]string{{"modules", "money_supply", "flows"}},
			mutate: func(c *economysim.Config) {
				withMoney(c)
				for i := range economysim.MaxMoneyFlows + 1 {
					f := *moneyFlowOf(c, "stipend")
					f.ID = fmt.Sprintf("f%03d", i)
					c.Modules.MoneySupply.Flows = append(c.Modules.MoneySupply.Flows, f)
				}
			}},
		mon("MoneyFlow.ID", "upper case letter", economysim.ErrInvalidID, func(f *economysim.MoneyFlow) { f.ID = "Bad" }, "Bad", "id"),
		mon("MoneyFlow.ID", "a reason of the registry", economysim.ErrReasonCollision, func(f *economysim.MoneyFlow) { f.ID = "trade" }, "trade", "id"),
		{field: "MoneyFlow.ID", name: "a flow twice", rules: []error{economysim.ErrDuplicateID},
			paths: [][]string{{"modules", "money_supply", "flows", "stipend", "id"}},
			mutate: func(c *economysim.Config) {
				withMoney(c)
				c.Modules.MoneySupply.Flows = append(c.Modules.MoneySupply.Flows, *moneyFlowOf(c, "stipend"))
			}},
		mon("MoneyFlow.Kind", "neither grant nor levy", bounds, func(f *economysim.MoneyFlow) { f.Kind = "tax" }, "stipend", "kind"),
		mon("MoneyFlow.Target", "a kind whose module is not built", economysim.ErrModuleDependency,
			func(f *economysim.MoneyFlow) { f.Target = economysim.FlowTarget{Kind: "pool", Region: "north"} }, "stipend", "target.kind"),
		mon("MoneyFlow.Amount", "above MaxFlowAmount", bounds, func(f *economysim.MoneyFlow) { f.Amount = economysim.MaxFlowAmount + 1 }, "stipend", "amount"),
		mon("MoneyFlow.Level", "above MaxFlowAmount", bounds, func(f *economysim.MoneyFlow) { f.Amount, f.Level = 0, economysim.MaxFlowAmount+1 }, "stipend", "level"),
		mon("MoneyFlow.RateBP", "above MaxFlowRateBP", bounds, func(f *economysim.MoneyFlow) { f.RateBP = economysim.MaxFlowRateBP + 1 }, "charge", "rate_bp"),
		mon("MoneyFlow.Exempt", "above MaxFlowAmount", bounds, func(f *economysim.MoneyFlow) { f.Exempt = economysim.MaxFlowAmount + 1 }, "charge", "exempt"),
		mon("MoneyFlow.Per", "neither tick nor year", bounds, func(f *economysim.MoneyFlow) { f.Per = "week" }, "stipend", "per"),
		mon("MoneyFlow.Start", "below 0", bounds, func(f *economysim.MoneyFlow) { f.Start = -1 }, "stipend", "start"),
		mon("MoneyFlow.Until", "not above Start", bounds, func(f *economysim.MoneyFlow) { f.Start, f.Until = 5, 5 }, "stipend", "until"),
		mon("MoneyFlow.Limit", "above MaxFlowTotal", bounds, func(f *economysim.MoneyFlow) { f.Limit = economysim.MaxFlowTotal + 1 }, "stipend", "limit"),
		mon("MoneyFlow.NotIncome", "on a levy", economysim.ErrFlowShape, func(f *economysim.MoneyFlow) { f.NotIncome = true }, "charge", "not_income"),
		mon("FlowTarget.Kind", "not a kind", bounds, func(f *economysim.MoneyFlow) { f.Target.Kind = "bogus" }, "stipend", "target.kind"),
		mon("FlowTarget.Region", "unknown region", economysim.ErrUnknownRegion, func(f *economysim.MoneyFlow) { f.Target.Region = "nowhere" }, "stipend", "target.region"),
		mon("FlowTarget.Role", "unknown role", economysim.ErrUnknownRole, func(f *economysim.MoneyFlow) { f.Target.Role = "nobody" }, "stipend", "target.role"),
	}
}

// TestValidate_MoneySupply verifies the rules of spec-money-supply.md 12.1 that
// need more than one error or a shape of their own: the module on with the
// reference configuration validates, every shape error of 4.1.1, the collision of
// an ID with each reason of the registry, a target kind whose module is not built
// (plan 2-6 D320) and the count that stops the other rules. The field cases of
// the registry cover one bound each.
//
// Requirement: FR-MON-005, FR-MON-034, FR-MON-035, FR-MON-036, FR-MON-037, FR-MON-038
func TestValidate_MoneySupply(t *testing.T) {
	t.Parallel()
	flow := func(mutate func(f *economysim.MoneyFlow)) func(c *economysim.Config) {
		return func(c *economysim.Config) { withMoney(c); mutate(moneyFlowOf(c, "stipend")) }
	}
	levy := func(mutate func(f *economysim.MoneyFlow)) func(c *economysim.Config) {
		return func(c *economysim.Config) { withMoney(c); mutate(moneyFlowOf(c, "charge")) }
	}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		want   error
		path   []string
	}{
		{"valid", withMoney, nil, nil},
		{"module on with no flow", func(c *economysim.Config) { c.Modules.MoneySupply = &economysim.MoneySupply{} }, nil, nil},
		{"top-up", flow(func(f *economysim.MoneyFlow) { f.Amount, f.Level = 0, 5_000 }), nil, nil},
		{"fixed levy", levy(func(f *economysim.MoneyFlow) { f.RateBP, f.Amount = 0, 33 }), nil, nil},
		{"an empty Per is a tick", flow(func(f *economysim.MoneyFlow) { f.Per = "" }), nil, nil},
		{"grant with a rate", flow(func(f *economysim.MoneyFlow) { f.RateBP = 5 }), economysim.ErrFlowShape, []string{"rate_bp"}},
		{"grant with an exemption", flow(func(f *economysim.MoneyFlow) { f.Exempt = 5 }), economysim.ErrFlowShape, []string{"exempt"}},
		{"grant with Amount and Level", flow(func(f *economysim.MoneyFlow) { f.Level = 5 }), economysim.ErrFlowShape, []string{"amount"}},
		{"grant with neither", flow(func(f *economysim.MoneyFlow) { f.Amount = 0 }), economysim.ErrFlowShape, []string{"amount"}},
		{"top-up per year", flow(func(f *economysim.MoneyFlow) { f.Amount, f.Level, f.Per = 0, 5, economysim.PerYear }), economysim.ErrFlowShape, []string{"per"}},
		{"levy with a level", levy(func(f *economysim.MoneyFlow) { f.Level = 5 }), economysim.ErrFlowShape, []string{"level"}},
		{"levy with Amount and RateBP", levy(func(f *economysim.MoneyFlow) { f.Amount = 5 }), economysim.ErrFlowShape, []string{"amount"}},
		{"levy with neither", levy(func(f *economysim.MoneyFlow) { f.RateBP = 0 }), economysim.ErrFlowShape, []string{"amount"}},
		{"households with no role", flow(func(f *economysim.MoneyFlow) { f.Target.Role = "" }), economysim.ErrFlowShape, []string{"target", "role"}},
		{"pool target", flow(func(f *economysim.MoneyFlow) { f.Target = economysim.FlowTarget{Kind: "pool"} }), economysim.ErrModuleDependency, []string{"target", "kind"}},
		{"stores target", flow(func(f *economysim.MoneyFlow) { f.Target = economysim.FlowTarget{Kind: "stores"} }), economysim.ErrModuleDependency, []string{"target", "kind"}},
		{"treasury target", flow(func(f *economysim.MoneyFlow) { f.Target = economysim.FlowTarget{Kind: "treasury"} }), economysim.ErrModuleDependency, []string{"target", "kind"}},
		{"id that is not an identifier", flow(func(f *economysim.MoneyFlow) { f.ID = "Not An ID" }), economysim.ErrInvalidID, []string{"id"}},
		{"duplicate id", func(c *economysim.Config) {
			withMoney(c)
			fl := &c.Modules.MoneySupply.Flows
			dup := (*fl)[0]
			*fl = append(*fl, dup)
		}, economysim.ErrDuplicateID, []string{"id"}},
		{"until not after start", flow(func(f *economysim.MoneyFlow) { f.Start, f.Until = 5, 5 }), economysim.ErrOutOfBounds, []string{"until"}},
		{"unknown kind", flow(func(f *economysim.MoneyFlow) { f.Kind = "tax" }), economysim.ErrOutOfBounds, []string{"kind"}},
		{"unknown per", flow(func(f *economysim.MoneyFlow) { f.Per = "decade" }), economysim.ErrOutOfBounds, []string{"per"}},
		{"unknown region", flow(func(f *economysim.MoneyFlow) { f.Target.Region = "nowhere" }), economysim.ErrUnknownRegion, []string{"target", "region"}},
		{"unknown target kind", flow(func(f *economysim.MoneyFlow) { f.Target = economysim.FlowTarget{Kind: "moon"} }), economysim.ErrOutOfBounds, []string{"target", "kind"}},
		{"grant NotIncome to a pool", flow(func(f *economysim.MoneyFlow) { f.NotIncome, f.Target = true, economysim.FlowTarget{Kind: "pool"} }), economysim.ErrFlowShape, []string{"not_income"}},
		{"levy NotIncome", levy(func(f *economysim.MoneyFlow) { f.NotIncome = true }), economysim.ErrFlowShape, []string{"not_income"}},
		{"too many flows stops the other rules", func(c *economysim.Config) {
			withMoney(c)
			c.Modules.MoneySupply.Flows = make([]economysim.MoneyFlow, economysim.MaxMoneyFlows+1)
		}, economysim.ErrTooMany, []string{"modules", "money_supply", "flows"}},
	}
	for _, r := range []string{"trade", "initial", "want_consumption", "relief", "outside_budget", "wage_pay", "loan_out", "rent", "payout", "loss", "deaths"} {
		tests = append(tests, struct {
			name   string
			mutate func(c *economysim.Config)
			want   error
			path   []string
		}{"collision with " + r, flow(func(f *economysim.MoneyFlow) { f.ID = r }), economysim.ErrReasonCollision, []string{"id"}})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			err := c.Validate()
			if tt.want == nil {
				if err != nil {
					t.Fatalf("Validate = %v, want nil", err)
				}
				return
			}
			var found bool
			for _, ve := range validationErrors(err) {
				found = found || (errors.Is(ve, tt.want) && strings.HasSuffix(strings.Join(ve.Path, "."), strings.Join(tt.path, ".")))
			}
			if !found {
				t.Errorf("Validate = %v, want %v at a path ending %v", err, tt.want, tt.path)
			}
		})
	}
}

// validationErrors returns the *ValidationError of each error joined in err.
func validationErrors(err error) []*economysim.ValidationError {
	var out []*economysim.ValidationError
	var walk func(error)
	walk = func(e error) {
		if j, ok := e.(interface{ Unwrap() []error }); ok {
			for _, x := range j.Unwrap() {
				walk(x)
			}
			return
		}
		var ve *economysim.ValidationError
		if errors.As(e, &ve) {
			out = append(out, ve)
		}
	}
	walk(err)
	return out
}

// TestValidate_MoneySupplyOffCostsNothing verifies that with the module off no
// rule of it speaks: the reference configuration validates and a flow list that
// would be invalid is never read.
//
// Requirement: FR-MON-005
func TestValidate_MoneySupplyOffCostsNothing(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate(referenceConfig) = %v, want nil", err)
	}
	for _, w := range c.Warnings() {
		if w.Path[0] == "modules" && len(w.Path) > 1 && w.Path[1] == "money_supply" {
			t.Errorf("Warnings(module off) raised %s at %q, want none", w.Rule, w.Path)
		}
	}
}

// TestWarnings_MoneySupply verifies the five warnings of spec-money-supply.md
// 12.2 that this build raises, each with its path and none for the valid base
// (plan 2-6 D338). The census of the reference configuration is 253 growers and
// 136 cooks and a year is 365 ticks: a balanced pair is a grant of 136 and a
// levy of 253 a household, both 136 * 253 * 365 a year.
//
// Requirement: FR-MON-039
func TestWarnings_MoneySupply(t *testing.T) {
	t.Parallel()
	pair := func(grant, levy int64) func(c *economysim.Config) {
		return func(c *economysim.Config) {
			withMoney(c)
			moneyFlowOf(c, "stipend").Amount = grant
			f := moneyFlowOf(c, "charge")
			f.RateBP, f.Amount, f.Per, f.Exempt = 0, levy, economysim.PerTick, 3_000
		}
	}
	tests := []struct {
		name   string
		mutate func(c *economysim.Config)
		rule   string // "" for none of the five
		path   []string
	}{
		{"base raises none", withMoney, "", nil},
		{"module on with no flow", func(c *economysim.Config) { c.Modules.MoneySupply = &economysim.MoneySupply{} }, "no_flows",
			[]string{"modules", "money_supply"}},
		{"a grant and no levy", func(c *economysim.Config) {
			withMoney(c)
			c.Modules.MoneySupply.Flows = c.Modules.MoneySupply.Flows[1:]
		}, "faucet_without_sink", []string{"modules", "money_supply", "flows", "stipend"}},
		{"a grant with a limit is exempt", func(c *economysim.Config) {
			withMoney(c)
			c.Modules.MoneySupply.Flows = c.Modules.MoneySupply.Flows[1:]
			c.Modules.MoneySupply.Flows[0].Limit = 1_000
		}, "", nil},
		{"a levy that exempts less than the upkeep", func(c *economysim.Config) {
			withMoney(c)
			moneyFlowOf(c, "charge").Exempt = 998
		}, "levy_below_upkeep", []string{"modules", "money_supply", "flows", "charge"}},
		{"a levy that exempts the upkeep", func(c *economysim.Config) {
			withMoney(c)
			moneyFlowOf(c, "charge").Exempt = 999
		}, "", nil},
		{"a class with no households", func(c *economysim.Config) {
			withMoney(c)
			r := economysim.DefaultRole()
			r.ID = "idle"
			c.Roles = append(c.Roles, r)
			moneyFlowOf(c, "stipend").Target.Role = "idle"
		}, "class_empty", []string{"modules", "money_supply", "flows", "stipend"}},
		{"a pair of the same yearly total", pair(136, 253), "", nil},
		{"a pair 1% apart is within the tolerance", pair(136, 255), "", nil},
		{"a pair that drifts", pair(136, 300), "pair_unbalanced", []string{"modules", "money_supply"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := referenceConfig()
			tt.mutate(&c)
			var got []economysim.Warning
			for _, w := range c.Warnings() {
				if len(w.Path) > 1 && w.Path[0] == "modules" && w.Path[1] == "money_supply" {
					got = append(got, w)
				}
			}
			switch {
			case tt.rule == "" && len(got) != 0:
				t.Errorf("Warnings = %v, want none of the module's", got)
			case tt.rule != "" && (len(got) != 1 || got[0].Rule != tt.rule || !slices.Equal(got[0].Path, tt.path)):
				t.Errorf("Warnings = %v, want one %s at %v", got, tt.rule, tt.path)
			}
		})
	}
}

// TestWarnings_ClassWithoutIncomeExemptsGrant verifies that a role targeted by a
// grant to households that counts as income raises no class_without_income, while
// the same grant with NotIncome true, and a levy, do not exempt it.
//
// Requirement: FR-DEM-033, FR-MON-039
func TestWarnings_ClassWithoutIncomeExemptsGrant(t *testing.T) {
	t.Parallel()
	build := func(kind string, notIncome bool) economysim.Config {
		c := demandConfig()
		r := economysim.DefaultRole()
		r.ID = "pensioner"
		c.Roles = append(c.Roles, r)
		c.Groups = append(c.Groups, economysim.Group{Region: "north", Role: "pensioner", Households: 5})
		c.Modules.Demand.Tastes = append(c.Modules.Demand.Tastes, economysim.Taste{Role: "pensioner", Want: "comfort", Scale: 10_000})
		f := economysim.MoneyFlow{ID: "pension", Kind: kind, Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: "pensioner"},
			Amount: 5, NotIncome: notIncome && kind == economysim.FlowGrant}
		c.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{f}}
		return c
	}
	tests := []struct {
		name      string
		kind      string
		notIncome bool
		want      bool
	}{
		{"a grant that counts as income", economysim.FlowGrant, false, false},
		{"a grant that is not income", economysim.FlowGrant, true, true},
		{"a levy", economysim.FlowLevy, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := build(tt.kind, tt.notIncome)
			var got bool
			for _, w := range c.Warnings() {
				got = got || w.Rule == "class_without_income" && slices.Equal(w.Path, []string{"roles", "pensioner", "start_money"})
			}
			if got != tt.want {
				t.Errorf("class_without_income for the pensioner = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHashConfig_MoneySupply verifies that the module is part of the hash when
// on (even with no flow), that a flow's fields move it, that Label does not, that
// the spelling of the period and the order of the flows do not, and that the
// all-off reference hash is the pinned one.
//
// Requirement: NFR-MON-003
func TestHashConfig_MoneySupply(t *testing.T) {
	t.Parallel()
	off := hexHash(referenceConfig())
	if off != pinnedReferenceHash {
		t.Fatalf("HashConfig(reference) = %s, want the pinned %s", off, pinnedReferenceHash)
	}
	on := referenceConfig()
	withMoney(&on)
	empty := referenceConfig()
	empty.Modules.MoneySupply = &economysim.MoneySupply{}
	if hexHash(empty) == off || hexHash(on) == hexHash(empty) {
		t.Errorf("HashConfig of the module off, on and empty = %s, %s, %s, want three different hashes", off, hexHash(on), hexHash(empty))
	}
	same := []struct {
		name   string
		mutate func(c *economysim.Config)
	}{
		{"label", func(c *economysim.Config) { moneyFlowOf(c, "stipend").Label = "a label" }},
		{"empty per equals tick", func(c *economysim.Config) { moneyFlowOf(c, "stipend").Per = economysim.PerTick }},
		{"flow order", func(c *economysim.Config) {
			f := c.Modules.MoneySupply.Flows
			f[0], f[1] = f[1], f[0]
		}},
	}
	for _, tt := range same {
		c := referenceConfig()
		withMoney(&c)
		tt.mutate(&c)
		if got, want := hexHash(c), hexHash(on); got != want {
			t.Errorf("HashConfig(%s changed) = %s, want %s (unchanged)", tt.name, got, want)
		}
	}
	moved := []struct {
		name   string
		mutate func(f *economysim.MoneyFlow)
	}{
		{"id", func(f *economysim.MoneyFlow) { f.ID = "other" }},
		{"kind", func(f *economysim.MoneyFlow) { f.Kind = economysim.FlowLevy }},
		{"target role", func(f *economysim.MoneyFlow) { f.Target.Role = "cook" }},
		{"target region", func(f *economysim.MoneyFlow) { f.Target.Region = "" }},
		{"amount", func(f *economysim.MoneyFlow) { f.Amount++ }},
		{"level", func(f *economysim.MoneyFlow) { f.Level++ }},
		{"rate", func(f *economysim.MoneyFlow) { f.RateBP++ }},
		{"exempt", func(f *economysim.MoneyFlow) { f.Exempt++ }},
		{"per", func(f *economysim.MoneyFlow) { f.Per = economysim.PerYear }},
		{"start", func(f *economysim.MoneyFlow) { f.Start++ }},
		{"until", func(f *economysim.MoneyFlow) { f.Until = 9 }},
		{"limit", func(f *economysim.MoneyFlow) { f.Limit++ }},
		{"not income", func(f *economysim.MoneyFlow) { f.NotIncome = true }},
	}
	for _, tt := range moved {
		c := referenceConfig()
		withMoney(&c)
		tt.mutate(moneyFlowOf(&c, "stipend"))
		if hexHash(c) == hexHash(on) {
			t.Errorf("HashConfig(%s changed) = HashConfig(base), want a different hash", tt.name)
		}
	}
}

// FuzzMoneySupplyValidate checks, for any flow fields, that Validate and
// Warnings never panic, that an invalid configuration returns a sentinel
// error, and that the hash does not depend on the spelling of the period.
//
// Requirement: NFR-MON-003
func FuzzMoneySupplyValidate(f *testing.F) {
	f.Add("grant", "households", "grower", "north", int64(20), int64(0), int64(0), int64(0), "tick", int64(0), int64(0), int64(0), false)
	f.Add("levy", "households", "cook", "", int64(0), int64(0), int64(3650), int64(3000), "year", int64(5), int64(9), int64(100), false)
	f.Add("tax", "pool", "", "nowhere", int64(-1), int64(1), int64(99999), int64(-7), "week", int64(-1), int64(0), int64(-1), true)
	f.Fuzz(func(t *testing.T, kind, tkind, role, region string, amount, level, rate, exempt int64, per string, start, until, limit int64, notIncome bool) {
		c := referenceConfig()
		c.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{{
			ID: "fz", Kind: kind, Target: economysim.FlowTarget{Kind: tkind, Role: role, Region: region},
			Amount: amount, Level: level, RateBP: rate, Exempt: exempt, Per: per, Start: start, Until: until, Limit: limit, NotIncome: notIncome,
		}}}
		err := c.Validate()
		_ = c.Warnings()
		if err != nil && len(validationErrors(err)) == 0 {
			t.Fatalf("Validate = %v, want a *ValidationError", err)
		}
		if per == "" || per == economysim.PerTick {
			tick, blank := c, c
			tick.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{c.Modules.MoneySupply.Flows[0]}}
			blank.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{c.Modules.MoneySupply.Flows[0]}}
			tick.Modules.MoneySupply.Flows[0].Per, blank.Modules.MoneySupply.Flows[0].Per = economysim.PerTick, ""
			if hexHash(tick) != hexHash(blank) {
				t.Fatalf("HashConfig differs between Per tick and Per empty")
			}
		}
	})
}
