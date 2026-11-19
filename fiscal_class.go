package economysim

import "strings"

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
