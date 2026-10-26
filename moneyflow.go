package economysim

import (
	"cmp"
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// This file is the money supply module (spec-money-supply.md; architecture.md
// 2.10.1): the module's canonical form, its validator and counts and its
// warnings. The step, the audit, the snapshot rows and the checkpoint section
// are appended to it by the change that builds them. Its arithmetic is
// internal/moneyflow.

// The target kinds that are recognised but not built (plan 2-6 D320): a flow
// naming one is ErrModuleDependency until its module is built.
const (
	targetPool     = "pool"
	targetStores   = "stores"
	targetTreasury = "treasury"
)

// pairTolerance is the tolerance of the warning pair_unbalanced, in basis points
// of the larger yearly total (spec-money-supply.md 12.2). It is a reporting rule
// constant, not a tunable.
const pairTolerance = 1_000

// reservedReasons is every reason name of the registry of spec-ledger.md 7.2,
// whether or not its module is built: a flow ID equal to one is
// ErrReasonCollision (spec-money-supply.md 4.5.2, plan 2-6 D325). A later module
// that declares a reason adds its name here, and the test of the reasons fails
// until it does.
var reservedReasons = []string{
	"trade", "initial", "recipe_input", "recipe_output", "disposal",
	"consumption", "want_consumption", "spoilage", "wear", "estate_money",
	"estate_goods", "start_money", "relief", "outside_budget",
	"outside_unspent", "outside_receipts", "outside_offer", "outside_unsold",
	"outside_purchases", "escrow_in", "escrow_out", "escrow_refund",
	"wage_escrow", "wage_pay", "wage_refund", "loan_out", "loan_repay",
	"loan_interest", "rent", "title_sale", "premium", "payout", "transit_out",
	"transit_in", "haul_use", "transit_use", "loss", "split", "merge", "births",
	"starvation", "deaths",
}

// canonical returns the module in canonical order with the period spelled: flows
// by ID, and an empty Per as PerTick, so the effective configuration and the hash
// do not depend on the spelling (plan 2-6 D326, NFR-MON-003). It returns nil for
// a module that is off and copies what it changes.
func (m *MoneySupply) canonical() *MoneySupply {
	if m == nil {
		return nil
	}
	out := &MoneySupply{Flows: sortedBy(m.Flows, func(a, b MoneyFlow) int { return cmp.Compare(a.ID, b.ID) })}
	for i := range out.Flows {
		if out.Flows[i].Per == "" {
			out.Flows[i].Per = PerTick
		}
	}
	return out
}

// moneyCounts is the count bound of phase 1 for the module: more than
// MaxMoneyFlows flows is ErrTooMany (spec-money-supply.md 12.1, FR-MON-034).
func moneyCounts(m *MoneySupply, over func(n, bound int, p ...string)) {
	if m != nil {
		over(len(m.Flows), MaxMoneyFlows, "modules", "money_supply", "flows")
	}
}

// validateMoneySupply is the module's part of phases 2, 3 and 5 of
// spec-config.md 7.1.6 (spec-money-supply.md 12.1): the identity of each flow,
// the bounds of its fields, its shape and its target. It runs after phase 1
// found no count over a bound, and reports nothing when the module is off.
func (v *validator) validateMoneySupply() {
	m := v.c.Modules.MoneySupply
	if m == nil {
		return
	}
	seen := make(map[string]bool, len(m.Flows))
	for _, f := range m.Flows {
		base := []string{"modules", "money_supply", "flows", f.ID}
		switch {
		case !validID(f.ID):
			v.rep.add(ErrInvalidID, path(base, "id"), f.ID, "")
		case slices.Contains(reservedReasons, f.ID):
			v.rep.add(ErrReasonCollision, path(base, "id"), f.ID, "")
		}
		if seen[f.ID] {
			v.rep.add(ErrDuplicateID, path(base, "id"), f.ID, "")
		}
		seen[f.ID] = true
		v.flowFields(base, f)
		v.flowShape(base, f)
		v.flowTarget(base, f)
	}
}

// flowFields checks the bounds of the integer fields and the closed sets of the
// strings of one flow.
func (v *validator) flowFields(base []string, f MoneyFlow) {
	v.ranges(base, []fieldRange{
		{"amount", f.Amount, 0, MaxFlowAmount},
		{"level", f.Level, 0, MaxFlowAmount},
		{"rate_bp", f.RateBP, 0, MaxFlowRateBP},
		{"exempt", f.Exempt, 0, MaxFlowAmount},
		{"start", f.Start, 0, math.MaxInt64 - 1},
		{"limit", f.Limit, 0, MaxFlowTotal},
	})
	if f.Until != 0 && f.Until <= f.Start {
		v.rep.add(ErrOutOfBounds, path(base, "until"), i64(f.Until), i64(f.Start+1))
	}
	if f.Kind != FlowGrant && f.Kind != FlowLevy {
		v.rep.add(ErrOutOfBounds, path(base, "kind"), f.Kind, "")
	}
	if f.Per != "" && f.Per != PerTick && f.Per != PerYear {
		v.rep.add(ErrOutOfBounds, path(base, "per"), f.Per, "")
	}
	v.label(base, "label", f.Label)
}

// flowShape checks that the flow is exactly one of the four shapes: a rate grant
// (Amount), a top-up (Level), a fixed levy (Amount) or a rate levy (RateBP), and
// that no field of another shape is set (spec-money-supply.md 4.1.1). A zero
// field is absent.
func (v *validator) flowShape(base []string, f MoneyFlow) {
	bad := func(field string) { v.rep.add(ErrFlowShape, path(base, field), "", "") }
	switch f.Kind {
	case FlowGrant:
		grantShape(f, bad)
	case FlowLevy:
		levyShape(f, bad)
	}
}

// grantShape reports the fields of a grant that break its shape.
func grantShape(f MoneyFlow, bad func(field string)) {
	if f.RateBP != 0 {
		bad("rate_bp")
	}
	if f.Exempt != 0 {
		bad("exempt")
	}
	if (f.Amount != 0) == (f.Level != 0) {
		bad("amount")
	}
	if f.Level != 0 && f.Per == PerYear {
		bad("per")
	}
	if f.NotIncome && f.Target.Kind != TargetHouseholds {
		bad("not_income")
	}
}

// levyShape reports the fields of a levy that break its shape.
func levyShape(f MoneyFlow, bad func(field string)) {
	if f.Level != 0 {
		bad("level")
	}
	if (f.Amount != 0) == (f.RateBP != 0) {
		bad("amount")
	}
	if f.NotIncome {
		bad("not_income")
	}
}

// flowTarget checks the target of one flow: the kind, and for households the
// role and region, which must be defined. A kind that is recognised and not
// built is ErrModuleDependency (plan 2-6 D320).
func (v *validator) flowTarget(base []string, f MoneyFlow) {
	t := f.Target
	tp := path(base, "target")
	switch t.Kind {
	case TargetHouseholds:
		switch {
		case t.Role == "":
			v.rep.add(ErrFlowShape, path(tp, "role"), "", "")
		case !v.role[t.Role]:
			v.rep.add(ErrUnknownRole, path(tp, "role"), t.Role, "")
		}
		if t.Region != "" && !v.region[t.Region] {
			v.rep.add(ErrUnknownRegion, path(tp, "region"), t.Region, "")
		}
	case targetPool, targetStores, targetTreasury:
		v.rep.add(ErrModuleDependency, path(tp, "kind"), t.Kind, "")
	default:
		v.rep.add(ErrOutOfBounds, path(tp, "kind"), t.Kind, "")
	}
}

// moneyGrantsIncome reports whether a grant to the households of the role counts
// as income: a flow of kind grant to the role's households, not NotIncome. It is
// the exemption of class_without_income (spec-demand.md 10.2, FR-DEM-033).
func moneyGrantsIncome(c *Config, ro *Role) bool {
	m := c.Modules.MoneySupply
	if m == nil {
		return false
	}
	return slices.ContainsFunc(m.Flows, func(f MoneyFlow) bool {
		return f.Kind == FlowGrant && f.Target.Kind == TargetHouseholds && f.Target.Role == ro.ID && !f.NotIncome
	})
}

// moneyWarnings returns the five warnings of the module that this build can
// raise (spec-money-supply.md 12.2, plan 2-6 D338), or nil when it is off. Like
// warnings it reads a configuration as given, skips what is not defined and
// never fails.
func moneyWarnings(c *Config, lk lookup) []Warning {
	m := c.Modules.MoneySupply
	if m == nil {
		return nil
	}
	var out []Warning
	add := func(rule string, segs ...string) { out = append(out, Warning{Path: segs, Rule: rule}) }
	if len(m.Flows) == 0 {
		add("no_flows", "modules", "money_supply")
		return out
	}
	hasLevy := slices.ContainsFunc(m.Flows, func(f MoneyFlow) bool { return f.Kind == FlowLevy })
	for _, f := range m.Flows {
		flowWarnings(c, lk, f, hasLevy, add)
	}
	if pairUnbalanced(c, m) {
		add("pair_unbalanced", "modules", "money_supply")
	}
	return out
}

// flowWarnings raises the warnings of one flow: faucet_without_sink,
// class_empty and levy_below_upkeep.
func flowWarnings(c *Config, lk lookup, f MoneyFlow, hasLevy bool, add func(string, ...string)) {
	p := []string{"modules", "money_supply", "flows", f.ID}
	if f.Kind == FlowGrant && f.Limit == 0 && !hasLevy {
		add("faucet_without_sink", p...)
	}
	if f.Target.Kind != TargetHouseholds {
		return
	}
	if !roleHasGroup(c, f.Target) {
		add("class_empty", p...)
	}
	if f.Kind == FlowLevy && levyBelowUpkeep(c, lk, f) {
		add("levy_below_upkeep", p...)
	}
}

// targets reports whether a group is in the class of a household target.
func (t FlowTarget) targets(g Group) bool {
	return g.Role == t.Role && (t.Region == "" || g.Region == t.Region)
}

// roleHasGroup reports whether the class of a household target has households
// in any targeted region at construction.
func roleHasGroup(c *Config, t FlowTarget) bool {
	return slices.ContainsFunc(c.Groups, t.targets)
}

// levyBelowUpkeep is the condition of levy_below_upkeep: the Exempt of a levy is
// below one tick's upkeep cost of the targeted role at the reference prices of a
// region where the class has households (spec-money-supply.md 9.4).
func levyBelowUpkeep(c *Config, lk lookup, f MoneyFlow) bool {
	ro := lk.role[f.Target.Role]
	if ro == nil {
		return false
	}
	for _, g := range c.Groups {
		rg := lk.region[g.Region]
		if rg == nil || !f.Target.targets(g) {
			continue
		}
		if upkeep, ok := listCost(lk, g.Region, effectiveUpkeep(c, ro), rg.ExtraUpkeep); ok && f.Exempt < upkeep {
			return true
		}
	}
	return false
}

// pairUnbalanced is the condition of pair_unbalanced: there is a rate grant to
// households and a fixed levy on households, and their nominal yearly totals
// over the census differ by more than pairTolerance basis points of the larger
// (spec-money-supply.md 9.2). Rate levies and top-ups are not compared.
func pairUnbalanced(c *Config, m *MoneySupply) bool {
	grant, levy := yearSum{}, yearSum{}
	for _, f := range m.Flows {
		switch {
		case f.Target.Kind != TargetHouseholds || f.Amount == 0:
		case f.Kind == FlowGrant:
			grant.add(c, f)
		case f.Kind == FlowLevy:
			levy.add(c, f)
		}
	}
	return grant.usable() && levy.usable() && beyondTolerance(grant.total, levy.total)
}

// beyondTolerance reports whether two totals differ by more than pairTolerance
// basis points of the larger.
func beyondTolerance(a, b int64) bool {
	larger := max(a, b)
	lhs, lok := checked.Mul(larger-min(a, b), wholeBP)
	rhs, rok := checked.Mul(larger, pairTolerance)
	return lok && rok && lhs > rhs
}

// yearSum is the running yearly total of the fixed flows of one kind, and whether
// there was one and whether every sum fitted.
type yearSum struct {
	total    int64
	has, bad bool
}

// usable reports whether the sum has a flow and every sum fitted.
func (y yearSum) usable() bool { return y.has && !y.bad }

// add adds the yearly total of f; a sum that does not fit makes the pair not
// comparable.
func (y *yearSum) add(c *Config, f MoneyFlow) {
	year, yok := yearlyTotal(c, f)
	sum, sok := checked.Add(y.total, year)
	y.total, y.has = sum, true
	y.bad = y.bad || !yok || !sok
}

// yearlyTotal is the nominal amount a fixed flow moves in a year over the census
// of its class: the households of the targeted groups times Amount, times the
// ticks of a year for a period of a tick.
func yearlyTotal(c *Config, f MoneyFlow) (int64, bool) {
	var households int64
	for _, g := range c.Groups {
		if f.Target.targets(g) {
			households += g.Households
		}
	}
	total, ok := checked.Mul(households, f.Amount)
	if f.Per == PerYear {
		return total, ok
	}
	total, tok := checked.Mul(total, c.Calendar.TicksPerYear())
	return total, ok && tok
}
