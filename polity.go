package economysim

import (
	"cmp"
	"errors"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/polity"
)

// This file is the polity module (spec-polity.md; architecture.md 2.10.1): the
// validator of its configuration and its two warnings. The tables, the queries and
// the combiners are in internal/polity, and this file converts at the call. Outside
// this file the module appears only in the catalogue files (config.go, hash.go,
// bounds.go, defaults.go, errors.go, validate.go) and at the named hooks of ADR 0010.

// validatePolity checks the polity module (spec-polity.md 9.1, FR-POL-002 to 005,
// 014). It reports nothing when the module is off. The rules run in the order of
// 9.1 and a phase that finds an error ends the run, so the first error is the same
// for the same mistake: syntax and duplicates; counts; parents (unknown, then
// cycle, then depth); members (unknown region, unknown jurisdiction, duplicate
// region); seats (unknown region, not covered, outside node); StartMoney (without
// seat, then range, with the label bound).
func (v *validator) validatePolity() {
	m := v.c.Modules.Polity
	if m == nil {
		return
	}
	base := []string{"modules", "polity"}
	steps := []func() bool{
		func() bool { return v.polityIDs(base, m) },
		func() bool { return v.polityCounts(base, m) },
		func() bool { return v.polityParents(base, m) },
		func() bool { return v.polityMembers(base, m) },
		func() bool { return v.politySeats(base, m) },
		func() bool { return v.polityMoney(base, m) },
	}
	for _, step := range steps {
		if !step() {
			return
		}
	}
}

// polityIDs is the syntax of the jurisdiction IDs and their uniqueness.
func (v *validator) polityIDs(base []string, m *PolityConfig) bool {
	seen := make(map[string]bool, len(m.Jurisdictions)) // a membership test only
	n := len(v.rep.errs)
	for i := range m.Jurisdictions {
		p := path(base, "jurisdictions["+itoa(i)+"]", "id")
		id := m.Jurisdictions[i].ID
		if !validID(id) {
			v.rep.add(ErrInvalidID, p, id, "")
		}
		if seen[id] {
			v.rep.add(ErrDuplicateID, p, id, "")
		}
		seen[id] = true
	}
	return len(v.rep.errs) == n
}

// polityCounts is the count of jurisdictions and of membership rows.
func (v *validator) polityCounts(base []string, m *PolityConfig) bool {
	n := len(v.rep.errs)
	if len(m.Jurisdictions) > MaxJurisdictions {
		v.rep.add(ErrTooMany, path(base, "jurisdictions"), itoa(len(m.Jurisdictions)), itoa(MaxJurisdictions))
	}
	if len(m.Members) > MaxRegions {
		v.rep.add(ErrTooMany, path(base, "members"), itoa(len(m.Members)), itoa(MaxRegions))
	}
	return len(v.rep.errs) == n
}

// polityIndex maps a jurisdiction ID to its index in the given order, for the
// rules that report the path of a row. It is a lookup only.
func polityIndex(m *PolityConfig) map[string]int {
	idx := make(map[string]int, len(m.Jurisdictions))
	for i := range m.Jurisdictions {
		idx[m.Jurisdictions[i].ID] = i
	}
	return idx
}

// polityParents checks the parent links: an undefined parent, then a cycle, then
// the depth.
func (v *validator) polityParents(base []string, m *PolityConfig) bool {
	idx := polityIndex(m)
	n := len(v.rep.errs)
	for i := range m.Jurisdictions {
		if p := m.Jurisdictions[i].Parent; p != "" {
			if _, ok := idx[p]; !ok {
				v.rep.add(ErrUnknownJurisdiction, path(base, "jurisdictions["+itoa(i)+"]", "parent"), p, "")
			}
		}
	}
	if len(v.rep.errs) != n {
		return false
	}
	cc := m.canonical()
	_, err := polity.Build(polityParentArray(cc), nil, MaxPolityDepth)
	var be *polity.BuildError
	if !errors.As(err, &be) { // nil: the forest is sound
		return true
	}
	at := idx[cc.Jurisdictions[be.Index].ID]
	p := path(base, "jurisdictions["+itoa(at)+"]", "parent")
	if be.Err == polity.ErrCycle {
		v.rep.add(ErrJurisdictionCycle, p, cc.Jurisdictions[be.Index].Parent, "")
	} else {
		v.rep.add(ErrOutOfBounds, p, cc.Jurisdictions[be.Index].Parent, itoa(MaxPolityDepth))
	}
	return false
}

// polityParentArray converts the canonical jurisdictions to the parent array of
// internal/polity: the index of each parent in the sorted order, -1 for a root.
func polityParentArray(cc *PolityConfig) []int32 {
	parent := make([]int32, len(cc.Jurisdictions))
	for i, j := range cc.Jurisdictions {
		parent[i] = -1
		if j.Parent != "" {
			at, _ := slices.BinarySearchFunc(cc.Jurisdictions, j.Parent, func(e Jurisdiction, id string) int { return cmp.Compare(e.ID, id) })
			parent[i] = int32(at)
		}
	}
	return parent
}

// polityMembers checks the membership rows: an undefined region, then an undefined
// jurisdiction, then a second row for one region.
func (v *validator) polityMembers(base []string, m *PolityConfig) bool {
	idx := polityIndex(m)
	n := len(v.rep.errs)
	for i, r := range m.Members {
		if v.lk.region[r.Region] == nil {
			v.rep.add(ErrUnknownRegion, path(base, "members["+itoa(i)+"]", "region"), r.Region, "")
		}
	}
	if len(v.rep.errs) != n {
		return false
	}
	for i, r := range m.Members {
		if _, ok := idx[r.Jurisdiction]; !ok {
			v.rep.add(ErrUnknownJurisdiction, path(base, "members["+itoa(i)+"]", "jurisdiction"), r.Jurisdiction, "")
		}
	}
	if len(v.rep.errs) != n {
		return false
	}
	seen := make(map[string]bool, len(m.Members)) // a membership test only
	for i, r := range m.Members {
		if seen[r.Region] {
			v.rep.add(ErrDuplicateID, path(base, "members["+itoa(i)+"]", "region"), r.Region, "")
		}
		seen[r.Region] = true
	}
	return len(v.rep.errs) == n
}

// polityCovers reports whether jurisdiction id covers the region: it is the
// direct jurisdiction of the region or an ancestor of it. direct maps a region ID
// to its direct jurisdiction ID; the parent links are acyclic by now.
func polityCovers(m *PolityConfig, idx map[string]int, direct map[string]string, id, region string) bool {
	for cur, ok := direct[region]; ok && cur != ""; cur = m.Jurisdictions[idx[cur]].Parent {
		if cur == id {
			return true
		}
	}
	return false
}

// politySeats checks the seats: an undefined region, then a region the jurisdiction
// does not cover, then an outside node.
func (v *validator) politySeats(base []string, m *PolityConfig) bool {
	idx := polityIndex(m)
	direct := make(map[string]string, len(m.Members))
	for _, r := range m.Members {
		direct[r.Region] = r.Jurisdiction
	}
	outside := v.outsideSet()
	rules := []struct {
		err error
		bad func(j *Jurisdiction) bool
	}{
		{ErrUnknownRegion, func(j *Jurisdiction) bool { return v.lk.region[j.Seat] == nil }},
		{ErrSeatNotCovered, func(j *Jurisdiction) bool { return !polityCovers(m, idx, direct, j.ID, j.Seat) }},
		{ErrSeatOutsideNode, func(j *Jurisdiction) bool { return outside[j.Seat] }},
	}
	for _, rule := range rules {
		n := len(v.rep.errs)
		for i := range m.Jurisdictions {
			j := &m.Jurisdictions[i]
			if j.Seat != "" && rule.bad(j) {
				v.rep.add(rule.err, path(base, "jurisdictions["+itoa(i)+"]", "seat"), j.Seat, "")
			}
		}
		if len(v.rep.errs) != n {
			return false
		}
	}
	return true
}

// polityMoney checks StartMoney: above 0 without a seat, then outside 0 to
// MaxMoney, then the sum against MaxMoney; the label bound is checked with the
// range. The sum is a rule beyond spec-polity.md 9.1: every treasury is created
// with its StartMoney, and the total of the world's money must stay at most
// MaxMoney (bounds.go), which a thousand treasuries of MaxMoney each would not.
func (v *validator) polityMoney(base []string, m *PolityConfig) bool {
	n := len(v.rep.errs)
	for i := range m.Jurisdictions {
		j := &m.Jurisdictions[i]
		if j.StartMoney > 0 && j.Seat == "" {
			v.rep.add(ErrStartWithoutSeat, path(base, "jurisdictions["+itoa(i)+"]", "start_money"), i64(int64(j.StartMoney)), "")
		}
	}
	if len(v.rep.errs) != n {
		return false
	}
	for i := range m.Jurisdictions {
		j := &m.Jurisdictions[i]
		jb := path(base, "jurisdictions["+itoa(i)+"]")
		v.inRange(path(jb, "start_money"), int64(j.StartMoney), 0, MaxMoney)
		v.label(jb, "label", j.Label)
	}
	if len(v.rep.errs) != n {
		return false
	}
	var total int64
	for i := range m.Jurisdictions {
		var ok bool
		if total, ok = checked.Add(total, int64(m.Jurisdictions[i].StartMoney)); !ok || total > MaxMoney {
			v.rep.add(ErrOverflow, path(base, "jurisdictions"), overflowValue, i64(MaxMoney))
			return false
		}
	}
	return true
}

// polityWarnings returns the warnings of spec-polity.md 9.2 for c read as given:
// jurisdiction_empty for a jurisdiction that covers no region, and
// region_unassigned for each region that is not an outside node and has no row
// when at least one region has one (FR-POL-026). It never fails: an ID that is not
// declared is skipped, and a parent chain is followed for at most the number of
// jurisdictions so that a cycle ends.
func polityWarnings(c *Config) []Warning {
	m := c.Modules.Polity
	if m == nil {
		return nil
	}
	covering, assigned := polityCoverage(c, m)
	var out []Warning
	for i := range m.Jurisdictions {
		if id := m.Jurisdictions[i].ID; !covering[id] {
			out = append(out, Warning{Path: []string{"modules", "polity", "jurisdiction", id}, Rule: "jurisdiction_empty"})
		}
	}
	if len(m.Members) > 0 {
		outside := outsideNodeSet(c)
		for i := range c.Regions {
			if id := c.Regions[i].ID; !assigned[id] && !outside[id] {
				out = append(out, Warning{Path: []string{"modules", "polity", "member", id}, Rule: "region_unassigned"})
			}
		}
	}
	return out
}

// polityCoverage returns the jurisdictions that cover at least one defined region
// and the regions that have a membership row. The parent links are followed for at
// most the number of jurisdictions, so a cycle in a configuration that does not
// validate ends the walk.
func polityCoverage(c *Config, m *PolityConfig) (covering, assigned map[string]bool) {
	idx := polityIndex(m)
	regions := make(map[string]bool, len(c.Regions))
	for i := range c.Regions {
		regions[c.Regions[i].ID] = true
	}
	covering = make(map[string]bool, len(m.Jurisdictions))
	assigned = make(map[string]bool, len(m.Members))
	for _, r := range m.Members {
		assigned[r.Region] = true
		if !regions[r.Region] {
			continue
		}
		cur := r.Jurisdiction
		for step := 0; step <= len(m.Jurisdictions); step++ {
			at, ok := idx[cur]
			if !ok {
				break
			}
			covering[cur] = true
			cur = m.Jurisdictions[at].Parent
		}
	}
	return covering, assigned
}
