package economysim

import "slices"

// This file is the warnings of the fiscal module (spec-fiscal.md 15.2, FR-FIS-013; a
// continuation file of fiscal_validate.go, architecture.md 2.6). None fails
// validation. Each is raised once, at construction, with the path of the flow it is
// about, or of the module. The functions read the configuration as given and never
// panic on one that does not validate: a name that is not declared is skipped.

// fiscalWarner holds what the warnings read: the module, the tree of jurisdictions and
// the membership of the regions as plain lookups, read as given.
type fiscalWarner struct {
	c      *Config
	lk     lookup
	m      *Fiscal
	flows  []fiscalFlow      // m.flows(), built once
	parent map[string]string // jurisdiction to parent; a lookup only
	member map[string]string // region to its direct jurisdiction; a lookup only
	out    []Warning
}

// fiscalWarnings returns the warnings of spec-fiscal.md 15.2; it reports nothing when
// the module is off.
func fiscalWarnings(c *Config, lk lookup) []Warning {
	m := c.Modules.Fiscal
	if m == nil {
		return nil
	}
	w := &fiscalWarner{c: c, lk: lk, m: m, flows: m.flows(), parent: map[string]string{}, member: map[string]string{}}
	if p := c.Modules.Polity; p != nil {
		for _, j := range p.Jurisdictions {
			w.parent[j.ID] = j.Parent
		}
		for _, r := range p.Members {
			w.member[r.Region] = r.Jurisdiction
		}
	}
	if len(w.flows) == 0 {
		return []Warning{{Path: fiscalBase, Rule: "no_fiscal_flows"}}
	}
	w.treasuryWarnings()
	w.taxWarnings()
	w.payoutWarnings()
	w.orderWarnings()
	w.dutyWarnings()
	w.tollWarnings()
	return w.out
}

// warn raises a warning about the flow of a list with an ID.
func (w *fiscalWarner) warn(rule, list, id string) {
	w.out = append(w.out, Warning{Path: path(fiscalBase, list, id), Rule: rule})
}

// chain returns the jurisdictions that cover a region, innermost first: its direct
// jurisdiction and the ancestors of it. A cycle in a configuration that does not
// validate ends the walk.
func (w *fiscalWarner) chain(region string) []string {
	var out []string
	for j, ok := w.member[region]; ok && j != "" && len(out) <= len(w.parent); j, ok = w.parent[j] {
		out = append(out, j)
	}
	return out
}

// covers reports whether the authority covers the region.
func (w *fiscalWarner) covers(authority, region string) bool {
	return slices.Contains(w.chain(region), authority)
}

// below reports whether j is a proper descendant of a.
func (w *fiscalWarner) below(j, a string) bool {
	for p, steps := w.parent[j], 0; p != "" && steps <= len(w.parent); p, steps = w.parent[p], steps+1 {
		if p == a {
			return true
		}
	}
	return false
}

// regions returns the regions a class names under an authority: the named one, or
// every region the authority covers.
func (w *fiscalWarner) regions(authority, region string) []string {
	if region != "" {
		return []string{region}
	}
	var out []string
	for _, r := range w.c.Regions {
		if w.covers(authority, r.ID) {
			out = append(out, r.ID)
		}
	}
	return out
}

// groups returns the groups of the households of a class: the role (any if empty) in
// the regions the class names under the authority.
func (w *fiscalWarner) groups(authority string, t FlowTarget) []Group {
	var out []Group
	regions := w.regions(authority, t.Region)
	for _, g := range w.c.Groups {
		if (t.Role == "" || g.Role == t.Role) && slices.Contains(regions, g.Region) {
			out = append(out, g)
		}
	}
	return out
}

// treasuryWarnings raises treasury_unfunded and treasury_idle, once per authority at
// the first flow that makes it so.
func (w *fiscalWarner) treasuryWarnings() {
	var seen []string
	for _, f := range w.flows {
		if slices.Contains(seen, f.authority) {
			continue
		}
		seen = append(seen, f.authority)
		w.funding(f.authority)
	}
}

// funding raises the two warnings of the treasury of one authority (spec-fiscal.md
// 15.2). A grant of the money supply module never targets a treasury (note N-389), so
// it counts as no source and no outlet.
func (w *fiscalWarner) funding(a string) {
	var inflow, outflow []fiscalFlow
	for _, f := range w.flows {
		if f.authority != a {
			continue
		}
		switch f.list {
		case fiscalTaxes, fiscalDuties, fiscalTolls, fiscalReleases:
			inflow = append(inflow, f)
		default:
			outflow = append(outflow, f)
		}
	}
	if len(outflow) > 0 && !w.funded(a, inflow) {
		w.warn("treasury_unfunded", outflow[0].list, outflow[0].id)
	}
	collects := slices.IndexFunc(inflow, func(f fiscalFlow) bool { return f.list != fiscalReleases })
	if collects >= 0 && !w.drained(a, outflow) {
		w.warn("treasury_idle", inflow[collects].list, inflow[collects].id)
	}
}

// funded reports whether anything funds the treasury of a: a tax, a duty, a toll or a
// release of its own, a payout that targets it from above, or a start money.
func (w *fiscalWarner) funded(a string, inflow []fiscalFlow) bool {
	if len(inflow) > 0 {
		return true
	}
	if p := w.c.Modules.Polity; p != nil && slices.ContainsFunc(p.Jurisdictions, func(j Jurisdiction) bool { return j.ID == a && j.StartMoney > 0 }) {
		return true
	}
	return slices.ContainsFunc(w.m.Payouts, func(p Payout) bool {
		return p.Target.Kind == TargetTreasury && w.reaches(p.Authority, p.Target.Jurisdiction, a)
	})
}

// drained reports whether what the treasury of a collects can leave it: a payout or a
// purchase of its own, or a remittance from above that targets it.
func (w *fiscalWarner) drained(a string, outflow []fiscalFlow) bool {
	if len(outflow) > 0 {
		return true
	}
	return slices.ContainsFunc(w.m.Taxes, func(t Tax) bool {
		return t.Target.Kind == TargetTreasury && w.reaches(t.Authority, t.Target.Jurisdiction, a)
	})
}

// reaches reports whether a flow of authority whose treasury target names j (every
// proper descendant if empty) includes the treasury of a.
func (w *fiscalWarner) reaches(authority, j, a string) bool {
	return w.below(a, authority) && (j == "" || j == a)
}

// taxWarnings raises tax_below_upkeep and fiscal_class_empty for the taxes.
func (w *fiscalWarner) taxWarnings() {
	for _, t := range w.m.Taxes {
		if t.Target.Kind != TargetHouseholds {
			continue
		}
		groups := w.groups(t.Authority, t.Target)
		if len(groups) == 0 {
			w.warn("fiscal_class_empty", fiscalTaxes, t.ID)
		}
		if w.belowUpkeep(groups, t.Exempt) {
			w.warn("tax_below_upkeep", fiscalTaxes, t.ID)
		}
	}
}

// belowUpkeep is the condition of tax_below_upkeep: the exemption is below one tick's
// upkeep cost of a role of the class at the reference prices of a region where the
// class has households.
func (w *fiscalWarner) belowUpkeep(groups []Group, exempt int64) bool {
	for _, g := range groups {
		ro, rg := w.lk.role[g.Role], w.lk.region[g.Region]
		if ro == nil || rg == nil {
			continue
		}
		if cost, ok := listCost(w.lk, g.Region, effectiveUpkeep(w.c, ro), rg.ExtraUpkeep); ok && exempt < cost {
			return true
		}
	}
	return false
}

// payoutWarnings raises payout_pool_no_outlet and fiscal_class_empty for the payouts.
func (w *fiscalWarner) payoutWarnings() {
	for _, p := range w.m.Payouts {
		switch p.Target.Kind {
		case TargetHouseholds:
			if len(w.groups(p.Authority, p.Target)) == 0 {
				w.warn("fiscal_class_empty", fiscalPayouts, p.ID)
			}
		case TargetPool:
			if w.noOutlet(p) {
				w.warn("payout_pool_no_outlet", fiscalPayouts, p.ID)
			}
		}
	}
}

// noOutlet is the condition of payout_pool_no_outlet: a region of a pool target has a
// PoolRelease of 0 (spec-fiscal.md 8.6). It needs the estates module.
func (w *fiscalWarner) noOutlet(p Payout) bool {
	if w.c.Modules.Estates == nil {
		return false
	}
	return slices.ContainsFunc(w.c.Regions, func(r Region) bool {
		return r.PoolRelease == 0 && slices.Contains(w.regions(p.Authority, p.Target.Region), r.ID)
	})
}

// orderWarnings raises purchase_unbounded and release_unsourced.
func (w *fiscalWarner) orderWarnings() {
	for _, o := range w.m.Purchases {
		if o.Hold == 0 && !w.drawn(o.Commodity) {
			w.warn("purchase_unbounded", fiscalPurchases, o.ID)
		}
	}
	for _, o := range w.m.Releases {
		if !slices.ContainsFunc(w.m.Purchases, func(p Purchase) bool {
			return p.Authority == o.Authority && p.Commodity == o.Commodity
		}) {
			w.warn("release_unsourced", fiscalReleases, o.ID)
		}
	}
}

// drawn reports whether a release, an in-kind payout or a recipe of the pack draws on a
// commodity, so that something besides the treasury's cash limits a purchase of it.
func (w *fiscalWarner) drawn(commodity string) bool {
	return slices.ContainsFunc(w.m.Releases, func(r Release) bool { return r.Commodity == commodity }) ||
		slices.ContainsFunc(w.m.Payouts, func(p Payout) bool { return p.Commodity == commodity }) ||
		slices.ContainsFunc(w.c.Recipes, func(r Recipe) bool {
			return slices.ContainsFunc(r.Inputs, func(it Item) bool { return it.Commodity == commodity })
		})
}

// dutyWarnings raises duty_stack_above_whole.
func (w *fiscalWarner) dutyWarnings() {
	flagged := make([]bool, len(w.m.Duties))
	commodities := append([]string{""}, w.dutyCommodities()...)
	for _, r := range w.c.Regions {
		for _, scope := range []string{ScopeDomestic, ScopeImport, ScopeExport} {
			for _, payer := range []string{DutySeller, DutyBuyer} {
				w.stack(flagged, commodities, r.ID, scope, payer)
			}
		}
	}
	for i, d := range w.m.Duties {
		if flagged[i] {
			w.warn("duty_stack_above_whole", fiscalDuties, d.ID)
		}
	}
}

// stack sums the rates of the duties that can match one trade of a region, a scope and
// a payer and flags the first of them when the sum is above the whole value of the
// trade. Each commodity named by a duty is tried, and the trade of a commodity none
// names, so the sum is of the duties that one trade of some commodity meets.
func (w *fiscalWarner) stack(flagged []bool, commodities []string, region, scope, payer string) {
	for _, commodity := range commodities {
		var sum int64
		first := -1
		for i, d := range w.m.Duties {
			if d.Payer != payer || !w.matches(d, region, scope, commodity) {
				continue
			}
			sum += d.RateBP
			if first < 0 {
				first = i
			}
		}
		if sum > wholeBP {
			flagged[first] = true
		}
	}
}

// dutyCommodities returns the commodities that some duty names, without repeats.
func (w *fiscalWarner) dutyCommodities() []string {
	var out []string
	for _, d := range w.m.Duties {
		if d.Commodity != "" && !slices.Contains(out, d.Commodity) {
			out = append(out, d.Commodity)
		}
	}
	return out
}

// matches reports whether a duty can match a trade in a region of a scope (domestic,
// import or export) of a commodity ("" for one no duty names).
func (w *fiscalWarner) matches(d Duty, region, scope, commodity string) bool {
	if d.Region != "" && d.Region != region || d.Region == "" && !w.covers(d.Authority, region) {
		return false
	}
	if d.Commodity != "" && d.Commodity != commodity {
		return false
	}
	return d.Scope == "" || d.Scope == ScopeAny || d.Scope == scope
}

// tollWarnings raises toll_no_border: no edge of the graph leaves (On exit) or enters
// (On enter) the authority, so the toll can never apply.
func (w *fiscalWarner) tollWarnings() {
	for _, t := range w.m.Tolls {
		if !slices.ContainsFunc(w.c.Edges, func(e Edge) bool { return w.crosses(t, e) }) {
			w.warn("toll_no_border", fiscalTolls, t.ID)
		}
	}
}

// crosses reports whether an edge leaves or enters the authority of a toll, as its On
// says: the authority is in the chain of one end and not of the other (spec-polity.md
// 4.1.1).
func (w *fiscalWarner) crosses(t Toll, e Edge) bool {
	from, to := w.covers(t.Authority, e.From), w.covers(t.Authority, e.To)
	if t.On == TollExit {
		return from && !to
	}
	return to && !from
}
