package economysim

import (
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/fiscal"
)

// This file is the validation of the fiscal module (spec-fiscal.md 15.1; the
// continuation file of fiscal.go, architecture.md 2.6): the rules that Validate
// checks, in the order of 15.1. The warnings of 15.2 are in fiscal_warnings.go. Nothing
// at run time returns an error from the module (FR-FIS-012).

// fiscalBase is the path of the module in a configuration.
var fiscalBase = []string{"modules", "fiscal"}

// The names of the six lists in a path, in the order the module's calls walk them.
const (
	fiscalTaxes     = "taxes"
	fiscalDuties    = "duties"
	fiscalTolls     = "tolls"
	fiscalPayouts   = "payouts"
	fiscalPurchases = "purchases"
	fiscalReleases  = "releases"
)

// fiscalFlow is what a flow of any kind has in common: the part the common phases
// read.
type fiscalFlow struct {
	list      string
	id        string
	authority string
	window    Window
}

// fiscalPath is the path of a key of a flow: the list, the ID, then the key.
func fiscalPath(f fiscalFlow, key ...string) []string {
	return path(fiscalBase, append([]string{f.list, f.id}, key...)...)
}

// flows returns the flows of the six lists as given, taxes first, then duties, tolls,
// payouts, purchases and releases.
func (m *Fiscal) flows() []fiscalFlow {
	out := make([]fiscalFlow, 0, len(m.Taxes)+len(m.Duties)+len(m.Tolls)+len(m.Payouts)+len(m.Purchases)+len(m.Releases))
	for _, f := range m.Taxes {
		out = append(out, fiscalFlow{fiscalTaxes, f.ID, f.Authority, f.Window})
	}
	for _, f := range m.Duties {
		out = append(out, fiscalFlow{fiscalDuties, f.ID, f.Authority, f.Window})
	}
	for _, f := range m.Tolls {
		out = append(out, fiscalFlow{fiscalTolls, f.ID, f.Authority, f.Window})
	}
	for _, f := range m.Payouts {
		out = append(out, fiscalFlow{fiscalPayouts, f.ID, f.Authority, f.Window})
	}
	for _, f := range m.Purchases {
		out = append(out, fiscalFlow{fiscalPurchases, f.ID, f.Authority, f.Window})
	}
	for _, f := range m.Releases {
		out = append(out, fiscalFlow{fiscalReleases, f.ID, f.Authority, f.Window})
	}
	return out
}

// fiscalCheck is the state the phases of validateFiscal share: the module, its flows,
// and the polity and the canonical configuration it was built from (zero until the
// dependency phase has found the polity sound).
type fiscalCheck struct {
	v     *validator
	m     *Fiscal
	flows []fiscalFlow
	cc    Config
	pol   Polity
}

// validateFiscal checks the fiscal module (spec-fiscal.md 15.1, FR-FIS-004, 007, 008,
// 011, 012). It reports nothing when the module is off. The rules run in the order of
// 15.1 and a phase that finds an error ends the run, so the first error is the same for
// the same mistake: the dependencies, the counts, the IDs (syntax, duplicates, reasons),
// the authorities, the windows, the shapes and last the classes.
func (v *validator) validateFiscal() {
	m := v.c.Modules.Fiscal
	if m == nil {
		return
	}
	k := &fiscalCheck{v: v, m: m, flows: m.flows()}
	for _, step := range []func() bool{
		k.dependencies, k.counts, k.ids, k.authorities, k.windows, k.shapes, k.classes,
	} {
		if !step() {
			return
		}
	}
}

// dependencies is ErrModuleDependency: the module needs the polity module with a
// governed jurisdiction, a pool target the estates module, an import or export scope
// the external module and a toll the transport module. It also builds the polity the
// later phases read; a polity that is itself invalid ends the run here, its own errors
// having been reported by validatePolity.
func (k *fiscalCheck) dependencies() bool {
	v, n := k.v, len(k.v.rep.errs)
	p := v.c.Modules.Polity
	if p == nil || !slices.ContainsFunc(p.Jurisdictions, func(j Jurisdiction) bool { return j.Seat != "" }) {
		v.rep.add(ErrModuleDependency, fiscalBase, "polity", "")
		return false
	}
	k.needs()
	if len(v.rep.errs) != n {
		return false
	}
	return k.buildPolity()
}

// needs reports the module dependencies of the flows: a pool target needs the estates
// module, an import or export scope the external module and a toll the transport
// module.
func (k *fiscalCheck) needs() {
	v := k.v
	for _, t := range k.m.Taxes {
		k.needPool(fiscalFlow{list: fiscalTaxes, id: t.ID}, t.Target)
	}
	for _, t := range k.m.Payouts {
		k.needPool(fiscalFlow{list: fiscalPayouts, id: t.ID}, t.Target)
	}
	for _, d := range k.m.Duties {
		if (d.Scope == ScopeImport || d.Scope == ScopeExport) && v.c.Modules.External == nil {
			v.rep.add(ErrModuleDependency, fiscalPath(fiscalFlow{list: fiscalDuties, id: d.ID}, "scope"), d.Scope, "external")
		}
	}
	if v.c.Modules.Transport == nil && len(k.m.Tolls) > 0 {
		v.rep.add(ErrModuleDependency, path(fiscalBase, fiscalTolls), "transport", "")
	}
}

// needPool reports a pool target while the estates module is off.
func (k *fiscalCheck) needPool(f fiscalFlow, t FlowTarget) {
	if t.Kind == TargetPool && k.v.c.Modules.Estates == nil {
		k.v.rep.add(ErrModuleDependency, fiscalPath(f, "target", "kind"), t.Kind, "estates")
	}
}

// buildPolity builds the polity of the configuration when the polity module is sound,
// and reports false when it is not.
func (k *fiscalCheck) buildPolity() bool {
	sub := &validator{c: k.v.c, lk: k.v.lk}
	sub.validatePolity()
	if sub.rep.err() != nil {
		return false
	}
	k.cc = k.v.c.canonical()
	k.pol = buildPolity(&k.cc)
	return true
}

// counts is ErrTooMany: more than MaxFiscalFlows flows of all kinds together.
func (k *fiscalCheck) counts() bool {
	if n := len(k.flows); n > MaxFiscalFlows {
		k.v.rep.add(ErrTooMany, fiscalBase, itoa(n), itoa(MaxFiscalFlows))
		return false
	}
	return true
}

// ids is ErrInvalidID, ErrDuplicateID and ErrReasonCollision: the IDs of the six lists
// and of the money flows are one namespace, and none is a built-in reason.
func (k *fiscalCheck) ids() bool {
	n := len(k.v.rep.errs)
	money := map[string]bool{} // membership tests only
	if ms := k.v.c.Modules.MoneySupply; ms != nil {
		for _, f := range ms.Flows {
			money[f.ID] = true
		}
	}
	seen := make(map[string]bool, len(k.flows))
	for _, f := range k.flows {
		p := fiscalPath(f, "id")
		switch {
		case !validID(f.id):
			k.v.rep.add(ErrInvalidID, p, f.id, "")
		case slices.Contains(reservedReasons, f.id) || money[f.id]:
			k.v.rep.add(ErrReasonCollision, p, f.id, "")
		case seen[f.id]:
			k.v.rep.add(ErrDuplicateID, p, f.id, "")
		}
		seen[f.id] = true
	}
	return len(k.v.rep.errs) == n
}

// authorities is ErrUnknownJurisdiction and ErrNotGoverned: the authority of every flow
// is a governed jurisdiction.
func (k *fiscalCheck) authorities() bool {
	n := len(k.v.rep.errs)
	for _, f := range k.flows {
		k.governed(fiscalPath(f, "authority"), f.authority)
	}
	return len(k.v.rep.errs) == n
}

// governed reports ErrUnknownJurisdiction for an ID that names no jurisdiction and
// ErrNotGoverned for one with no seat, and returns the jurisdiction when it is governed.
func (k *fiscalCheck) governed(p []string, id string) (JurisdictionID, bool) {
	j, ok := k.pol.Lookup(id)
	if !ok {
		k.v.rep.add(ErrUnknownJurisdiction, p, id, "")
		return 0, false
	}
	if _, seat := k.pol.Seat(j); !seat {
		k.v.rep.add(ErrNotGoverned, p, id, "")
		return 0, false
	}
	return j, true
}

// windows is ErrOutOfBounds, ErrUnknownSeason and ErrDuplicateID: the bounds of Start
// and Until, and the seasons of the calendar a window names, each at most once.
func (k *fiscalCheck) windows() bool {
	v, n := k.v, len(k.v.rep.errs)
	for _, f := range k.flows {
		w := f.window
		v.inRange(fiscalPath(f, "window", "start"), w.Start, 0, math.MaxInt64-1)
		if w.Until != 0 && w.Until <= w.Start {
			v.rep.add(ErrOutOfBounds, fiscalPath(f, "window", "until"), i64(w.Until), i64(w.Start+1))
		}
		seen := make(map[string]bool, len(w.Seasons)) // a membership test only
		for i, s := range w.Seasons {
			p := fiscalPath(f, "window", "seasons["+itoa(i)+"]")
			switch {
			case !slices.ContainsFunc(v.c.Calendar.Seasons, func(c Season) bool { return c.ID == s }):
				v.rep.add(ErrUnknownSeason, p, s, "")
			case seen[s]:
				v.rep.add(ErrDuplicateID, p, s, "")
			}
			seen[s] = true
		}
	}
	return len(v.rep.errs) == n
}

// shapes is ErrOutOfBounds and ErrFiscalShape: the bounds and closed sets of every
// field, the combinations of fields of each kind and the rules across the flows of
// one authority.
func (k *fiscalCheck) shapes() bool {
	n := len(k.v.rep.errs)
	for _, t := range k.m.Taxes {
		k.taxShape(t)
	}
	for _, d := range k.m.Duties {
		k.dutyShape(d)
	}
	for _, t := range k.m.Tolls {
		k.tollShape(t)
	}
	for _, p := range k.m.Payouts {
		k.payoutShape(p)
	}
	k.orderShapes()
	return len(k.v.rep.errs) == n
}

// closedSet reports ErrOutOfBounds for a string outside its set, "" being allowed when
// it is among them.
func (k *fiscalCheck) closedSet(p []string, s string, set ...string) {
	if !slices.Contains(set, s) {
		k.v.rep.add(ErrOutOfBounds, p, s, "")
	}
}

// shape reports ErrFiscalShape at p for a rule that is broken.
func (k *fiscalCheck) shape(broken bool, p []string, value string) {
	if broken {
		k.v.rep.add(ErrFiscalShape, p, value, "")
	}
}

// taxShape checks a tax: exactly one of Amount and RateBP, the bounds, Per and the
// kind of the target (spec-fiscal.md 5.1.2).
func (k *fiscalCheck) taxShape(t Tax) {
	f := fiscalFlow{list: fiscalTaxes, id: t.ID}
	k.v.ranges(fiscalPath(f), []fieldRange{
		{"amount", t.Amount, 0, MaxFlowAmount}, {"rate_bp", t.RateBP, 0, MaxFlowRateBP}, {"exempt", t.Exempt, 0, MaxFlowAmount},
	})
	k.shape((t.Amount != 0) == (t.RateBP != 0), fiscalPath(f, "amount"), i64(t.Amount))
	k.closedSet(fiscalPath(f, "per"), t.Per, "", PerTick, PerYear)
	k.v.label(fiscalPath(f), "label", t.Label)
	k.targetShape(f, t.Target)
}

// targetShape checks the kind of a target and the fields that kind does not take
// (spec-fiscal.md 3.2): a pool has no role, a treasury no role or region, and only a
// treasury has a jurisdiction.
func (k *fiscalCheck) targetShape(f fiscalFlow, t FlowTarget) {
	p := fiscalPath(f, "target")
	k.closedSet(path(p, "kind"), t.Kind, TargetHouseholds, TargetPool, TargetTreasury)
	k.shape(t.Store != "", path(p, "store"), t.Store)
	k.shape(t.Jurisdiction != "" && t.Kind != TargetTreasury, path(p, "jurisdiction"), t.Jurisdiction)
	k.shape(t.Role != "" && (t.Kind == TargetPool || t.Kind == TargetTreasury), path(p, "role"), t.Role)
	k.shape(t.Region != "" && t.Kind == TargetTreasury, path(p, "region"), t.Region)
}

// dutyShape checks a duty: the bounds, Payer, Scope and the combinations that can
// never match (spec-fiscal.md 6.1.1).
func (k *fiscalCheck) dutyShape(d Duty) {
	f := fiscalFlow{list: fiscalDuties, id: d.ID}
	k.v.inRange(fiscalPath(f, "rate_bp"), d.RateBP, 1, MaxDutyBP)
	k.closedSet(fiscalPath(f, "payer"), d.Payer, DutySeller, DutyBuyer)
	k.closedSet(fiscalPath(f, "scope"), d.Scope, "", ScopeDomestic, ScopeImport, ScopeExport, ScopeAny)
	k.shape(d.Scope == ScopeImport && d.Payer == DutySeller || d.Scope == ScopeExport && d.Payer == DutyBuyer,
		fiscalPath(f, "scope"), d.Scope)
	k.v.label(fiscalPath(f), "label", d.Label)
}

// tollShape checks a toll: the bounds and On.
func (k *fiscalCheck) tollShape(t Toll) {
	f := fiscalFlow{list: fiscalTolls, id: t.ID}
	k.v.inRange(fiscalPath(f, "amount"), t.Amount, 1, MaxFlowAmount)
	k.closedSet(fiscalPath(f, "on"), t.On, TollExit, TollEnter)
	k.v.label(fiscalPath(f), "label", t.Label)
}

// payoutShape checks a payout: exactly one of Amount and Level, Level for money and a
// tick only, NotIncome for households and money or goods as the shape of an in-kind
// payout says (spec-fiscal.md 8.2).
func (k *fiscalCheck) payoutShape(p Payout) {
	f := fiscalFlow{list: fiscalPayouts, id: p.ID}
	k.v.ranges(fiscalPath(f), []fieldRange{{"amount", p.Amount, 0, MaxFlowAmount}, {"level", p.Level, 0, MaxFlowAmount}})
	k.shape((p.Amount != 0) == (p.Level != 0), fiscalPath(f, "amount"), i64(p.Amount))
	k.shape(p.Level != 0 && (p.Per == PerYear || p.Commodity != ""), fiscalPath(f, "level"), i64(p.Level))
	k.closedSet(fiscalPath(f, "per"), p.Per, "", PerTick, PerYear)
	k.shape(p.NotIncome && (p.Target.Kind != TargetHouseholds || p.Commodity != ""), fiscalPath(f, "not_income"), "true")
	k.v.label(fiscalPath(f), "label", p.Label)
	k.targetShape(f, p.Target)
	k.inKind(f, p)
}

// inKind checks that an in-kind payout has households of the seat region as its target:
// a good does not cross a region outside a shipment (spec-fiscal.md 8.2).
func (k *fiscalCheck) inKind(f fiscalFlow, p Payout) {
	if p.Commodity == "" {
		return
	}
	seat := ""
	if r, ok := k.seatRegion(p.Authority); ok {
		seat = k.cc.Regions[r].ID
	}
	k.shape(p.Target.Kind != TargetHouseholds || p.Target.Region != seat, fiscalPath(f, "target"), p.Target.Region)
}

// orderShapes checks the standing orders: the bounds, a Hold of at least Quantity, and
// the rules across the orders of one authority and commodity (spec-fiscal.md 9.4.1).
func (k *fiscalCheck) orderShapes() {
	for _, o := range k.m.Purchases {
		f := fiscalFlow{list: fiscalPurchases, id: o.ID}
		k.orderBounds(f, o.Quantity, o.LimitBP, o.Label)
		k.v.inRange(fiscalPath(f, "hold"), o.Hold, 0, MaxQuantity)
		k.shape(o.Hold != 0 && o.Hold < o.Quantity, fiscalPath(f, "hold"), i64(o.Hold))
	}
	for _, o := range k.m.Releases {
		f := fiscalFlow{list: fiscalReleases, id: o.ID}
		k.orderBounds(f, o.Quantity, o.LimitBP, o.Label)
		k.v.inRange(fiscalPath(f, "reserve"), o.Reserve, 0, MaxQuantity)
	}
	k.pairs()
}

// orderBounds checks the fields a purchase and a release share.
func (k *fiscalCheck) orderBounds(f fiscalFlow, quantity, limitBP int64, label string) {
	k.v.inRange(fiscalPath(f, "quantity"), quantity, 1, MaxQuantity)
	k.v.inRange(fiscalPath(f, "limit_bp"), limitBP, 1, MaxLimitBP)
	k.v.label(fiscalPath(f), "label", label)
}

// pairs is the rule of 9.4.1 and its two companions: an authority has at most one
// purchase and one release of a commodity, a purchase and a release of one commodity
// have limit prices strictly ordered on the integers, and a reserve is not above the
// hold of the purchase of the same commodity.
func (k *fiscalCheck) pairs() {
	for i, a := range k.m.Purchases {
		for _, b := range k.m.Purchases[:i] {
			k.shape(a.Authority == b.Authority && a.Commodity == b.Commodity,
				fiscalPath(fiscalFlow{list: fiscalPurchases, id: a.ID}, "commodity"), a.Commodity)
		}
	}
	for i, a := range k.m.Releases {
		for _, b := range k.m.Releases[:i] {
			k.shape(a.Authority == b.Authority && a.Commodity == b.Commodity,
				fiscalPath(fiscalFlow{list: fiscalReleases, id: a.ID}, "commodity"), a.Commodity)
		}
		for _, b := range k.m.Purchases {
			if a.Authority == b.Authority && a.Commodity == b.Commodity {
				k.pair(a, b)
			}
		}
	}
}

// pair checks one release against the purchase of its authority and commodity: the
// integer limits of 9.2 at the effective reference price of the seat region are
// strictly ordered, and Reserve is not above Hold when both are set.
func (k *fiscalCheck) pair(sell Release, buy Purchase) {
	f := fiscalFlow{list: fiscalReleases, id: sell.ID}
	k.shape(buy.Hold != 0 && sell.Reserve > buy.Hold, fiscalPath(f, "reserve"), i64(sell.Reserve))
	region, ok1 := k.seatRegion(sell.Authority)
	c, ok2 := k.cc.commodityID(sell.Commodity)
	if !ok1 || !ok2 || !limitInBounds(buy.LimitBP) || !limitInBounds(sell.LimitBP) {
		return // an unknown commodity is the classes phase's, a limit out of bounds is reported already
	}
	ref := int64(k.cc.refPrice(region, c))
	bid, ok1 := fiscal.Limit(ref, buy.LimitBP, MaxPrice)
	ask, ok2 := fiscal.Limit(ref, sell.LimitBP, MaxPrice)
	k.shape(ok1 && ok2 && !fiscal.PairOK(bid, ask), fiscalPath(f, "limit_bp"), i64(sell.LimitBP))
}

// limitInBounds reports whether a LimitBP is inside 1 to MaxLimitBP.
func limitInBounds(bp int64) bool { return bp >= 1 && bp <= MaxLimitBP }

// seatRegion returns the seat region of a governed authority.
func (k *fiscalCheck) seatRegion(authority string) (RegionID, bool) {
	j, ok := k.pol.Lookup(authority)
	if !ok {
		return 0, false
	}
	return k.pol.Seat(j)
}
