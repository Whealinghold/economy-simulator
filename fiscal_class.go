package economysim

import (
	"slices"
	"strings"
)

// This file is the part of the fiscal module that names the class a flow acts on
// (spec-fiscal.md 3.2; a continuation file of fiscal.go, architecture.md 2.6). It
// holds the class rules of validateFiscal, the last phase of 15.1.

// classes is ErrUnknownRole, ErrUnknownRegion, ErrUnknownCommodity, ErrUnknownMode,
// ErrUnknownJurisdiction, ErrNotGoverned and ErrNotCovered: every name a flow uses is
// defined, every region is one its authority covers and a treasury target is a governed
// proper descendant of the authority.
func (k *fiscalCheck) classes() bool {
	n := len(k.v.rep.errs)
	for _, t := range k.m.Taxes {
		k.target(fiscalFlow{list: fiscalTaxes, id: t.ID, authority: t.Authority}, t.Target)
	}
	for _, p := range k.m.Payouts {
		f := fiscalFlow{list: fiscalPayouts, id: p.ID, authority: p.Authority}
		k.target(f, p.Target)
		k.commodity(f, p.Commodity, true)
	}
	for _, d := range k.m.Duties {
		f := fiscalFlow{list: fiscalDuties, id: d.ID, authority: d.Authority}
		k.region(f, "region", d.Region)
		k.commodity(f, d.Commodity, true)
	}
	for _, t := range k.m.Tolls {
		f := fiscalFlow{list: fiscalTolls, id: t.ID}
		k.mode(f, t.Mode)
	}
	for _, o := range k.m.Purchases {
		k.commodity(fiscalFlow{list: fiscalPurchases, id: o.ID}, o.Commodity, false)
	}
	for _, o := range k.m.Releases {
		k.commodity(fiscalFlow{list: fiscalReleases, id: o.ID}, o.Commodity, false)
	}
	return len(k.v.rep.errs) == n
}

// target checks the names of a target: the role of households, the region of
// households and of a pool, and the jurisdiction of a treasury.
func (k *fiscalCheck) target(f fiscalFlow, t FlowTarget) {
	if t.Role != "" && !k.v.role[t.Role] {
		k.v.rep.add(ErrUnknownRole, fiscalPath(f, "target", "role"), t.Role, "")
	}
	k.region(f, "target.region", t.Region)
	if t.Jurisdiction == "" || t.Kind != TargetTreasury {
		return
	}
	p := fiscalPath(f, "target", "jurisdiction")
	j, ok := k.governed(p, t.Jurisdiction)
	if !ok {
		return
	}
	if a, ok := k.pol.Lookup(f.authority); ok && !k.below(j, a) {
		k.v.rep.add(ErrNotCovered, p, t.Jurisdiction, "")
	}
}

// below reports whether j is a proper descendant of a in the tree of jurisdictions.
func (k *fiscalCheck) below(j, a JurisdictionID) bool {
	for p, ok := k.pol.Parent(j); ok; p, ok = k.pol.Parent(p) {
		if p == a {
			return true
		}
	}
	return false
}

// region checks a region name: defined (ErrUnknownRegion) and covered by the authority
// (ErrNotCovered). key is the dotted path under the flow; "" is every covered region.
func (k *fiscalCheck) region(f fiscalFlow, key, id string) {
	if id == "" {
		return
	}
	p := append(fiscalPath(f), strings.Split(key, ".")...)
	r, ok := k.cc.regionID(id)
	if !ok || !k.v.region[id] {
		k.v.rep.add(ErrUnknownRegion, p, id, "")
		return
	}
	if a, ok := k.pol.Lookup(f.authority); ok && !k.pol.Covers(a, r) {
		k.v.rep.add(ErrNotCovered, p, id, "")
	}
}

// commodity checks a commodity name: defined (ErrUnknownCommodity); an empty name is
// allowed where a flow means every commodity or money.
func (k *fiscalCheck) commodity(f fiscalFlow, id string, optional bool) {
	if id == "" && optional {
		return
	}
	if !k.v.commodity[id] {
		k.v.rep.add(ErrUnknownCommodity, fiscalPath(f, "commodity"), id, "")
	}
}

// mode checks a transport mode name: defined by the transport module (ErrUnknownMode);
// an empty name is every mode.
func (k *fiscalCheck) mode(f fiscalFlow, id string) {
	if id == "" {
		return
	}
	for _, m := range k.v.c.Modules.Transport.Modes {
		if m.ID == id {
			return
		}
	}
	k.v.rep.add(ErrUnknownMode, fiscalPath(f, "mode"), id, "")
}

// resolve builds the class of a flow's target (spec-fiscal.md 3.2): the roles and
// regions of households and of a pool, and the payer or recipient treasuries of a
// remittance. A name that does not resolve is a programmer error after validation.
func (s *fiscalState) resolve(c *Config, authority JurisdictionID, t FlowTarget) fiscalTarget {
	out := fiscalTarget{kind: t.Kind, role: -1}
	if t.Role != "" {
		out.role = slices.IndexFunc(c.Roles, func(r Role) bool { return r.ID == t.Role })
	}
	switch t.Kind {
	case TargetTreasury:
		out.from = s.treasuries(authority, t.Jurisdiction)
	default:
		out.in = s.regionsOf(c, authority, t.Region)
	}
	return out
}

// regionsOf is the set of regions a class is read in: the region named, or every
// region the authority covers when none is (spec-fiscal.md 3.2.4).
func (s *fiscalState) regionsOf(c *Config, authority JurisdictionID, region string) []bool {
	in := make([]bool, len(c.Regions))
	if region != "" {
		at := slices.IndexFunc(c.Regions, func(r Region) bool { return r.ID == region })
		if at < 0 {
			invariant("a fiscal flow's region is not defined after validation: " + region)
		}
		in[at] = true
		return in
	}
	for _, r := range s.pol.v.Regions(authority) {
		in[r] = true
	}
	return in
}

// treasuries lists the treasuries of a class of treasuries, as indexes into
// polity.tr in ascending JurisdictionID: the one jurisdiction named, or every
// governed proper descendant of the authority (spec-fiscal.md 3.2.3).
func (s *fiscalState) treasuries(authority JurisdictionID, named string) []int {
	if named != "" {
		j, ok := s.pol.v.Lookup(named)
		if !ok || s.pol.byJ[j] < 0 {
			invariant("a fiscal flow's treasury is not governed after validation: " + named)
		}
		return []int{int(s.pol.byJ[j])}
	}
	var out []int
	for i := range s.pol.tr {
		if s.descends(s.pol.tr[i].j, authority) {
			out = append(out, i)
		}
	}
	return out
}

// descends reports whether j is a proper descendant of a in the tree of jurisdictions.
func (s *fiscalState) descends(j, a JurisdictionID) bool {
	for p, ok := s.pol.v.Parent(j); ok; p, ok = s.pol.v.Parent(p) {
		if p == a {
			return true
		}
	}
	return false
}

// gather lists the accounts a flow acts on in the tick, in ascending account ID,
// into the scratch (spec-fiscal.md 3.2.1 to 3.2.3). Membership is read now, so a
// household that changed role or region is in its new class (D4). Estate and
// treasury pseudo-agents are never households; external agents are.
func (s *fiscalState) gather(w *World, f *fiscalFlowState) {
	s.payers = s.payers[:0]
	switch t := &f.target; t.kind {
	case TargetHouseholds:
		s.gatherHouseholds(w, t)
	case TargetPool:
		s.gatherPools(w, t)
	default:
		for _, i := range t.from {
			tr := &s.pol.tr[i]
			s.payers = append(s.payers, fiscalPayer{account: tr.account, weight: 1, region: tr.seat, owner: tr.agent})
		}
	}
}

// gatherHouseholds lists the live agents of the role (any if the class names none) in
// the regions of the class, in ascending agent ID, which is ascending account ID.
func (s *fiscalState) gatherHouseholds(w *World, t *fiscalTarget) {
	for i := w.firstHousehold; i < len(w.agents); i++ {
		a := &w.agents[i]
		if a.weight < 1 || (t.role >= 0 && int(a.role) != t.role) {
			continue
		}
		if g := w.regionOf(a); t.in[g] {
			s.payers = append(s.payers, fiscalPayer{account: a.account, weight: a.weight, region: g, owner: a.id})
		}
	}
}

// gatherPools lists the estate account of each region of the class, in ascending region.
func (s *fiscalState) gatherPools(w *World, t *fiscalTarget) {
	for g, on := range t.in {
		if on {
			s.payers = append(s.payers, fiscalPayer{account: w.estates.account(RegionID(g)), weight: 1, region: RegionID(g), owner: AgentID(g)})
		}
	}
}
