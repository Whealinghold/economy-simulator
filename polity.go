package economysim

import (
	"cmp"
	"errors"
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	pol "github.com/Whealinghold/economy-sim/internal/polity"
)

// This file is the polity module (spec-polity.md; architecture.md 2.10.1): the
// validator of its configuration and its two warnings, the Polity value its
// queries are read through, and the module state. The tables, the queries and
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
	_, err := pol.Build(polityParentArray(cc), nil, MaxPolityDepth)
	var be *pol.BuildError
	if !errors.As(err, &be) { // nil: the forest is sound
		return true
	}
	at := idx[cc.Jurisdictions[be.Index].ID]
	p := path(base, "jurisdictions["+itoa(at)+"]", "parent")
	if be.Err == pol.ErrCycle {
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

// JurisdictionID is the index of a jurisdiction in ascending order of its ID
// string. It is not stable across configurations: adding a jurisdiction can
// renumber the others. Use the string ID to persist or exchange one.
type JurisdictionID int

// NoJurisdiction means "none": a query returns it where a jurisdiction does not
// exist, or where no jurisdiction covers both of two regions.
const NoJurisdiction JurisdictionID = -1

// Crossing is what a good leaves and enters in going from one region to another
// (spec-polity.md 4.1.1).
type Crossing struct {
	// Exit lists the jurisdictions that cover the origin and not the destination,
	// innermost first.
	Exit []JurisdictionID
	// Enter lists the jurisdictions that cover the destination and not the origin,
	// outermost first.
	Enter []JurisdictionID
}

// Polity is the jurisdiction tree and the membership of the regions: plain data,
// immutable and without a mutating method. It cannot change during a run
// (spec-polity.md 7.4), so the world returns an equal value every time. Like Graph,
// each call of World.Polity returns a fresh deep copy that shares no storage with
// the world, and so do the lists its methods return (architecture.md 12.4). The
// zero value is the polity of a world without the module: no jurisdiction, and
// every query answers none.
type Polity struct {
	names []string // the jurisdiction IDs, ascending: the index is the JurisdictionID
	seats []int32  // the seat region of each jurisdiction, or -1 for a grouping
	gov   []JurisdictionID
	t     pol.Tables
}

// NewPolity checks the polity rules of spec-polity.md 9 on cfg and builds the value
// the module on that configuration would have. It reports the first rule that fails
// in the order of 9.1 as the framework of Validate does, and it checks nothing else
// of cfg. A configuration with no polity module gives the zero Polity.
func NewPolity(cfg Config) (Polity, error) {
	if cfg.Modules.Polity == nil {
		return Polity{}, nil
	}
	v := &validator{c: cfg}
	v.lk = newLookup(&v.c)
	v.validatePolity()
	if err := v.rep.err(); err != nil {
		return Polity{}, err
	}
	cc := cfg.canonical()
	return buildPolity(&cc), nil
}

// buildPolity derives the Polity of a canonical configuration whose polity module
// is on and valid: the regions are indexed by their position in the sorted list.
func buildPolity(c *Config) Polity {
	m := c.Modules.Polity
	member := make([]int32, len(c.Regions))
	for i := range member {
		member[i] = -1
	}
	for _, row := range m.Members {
		r, _ := c.regionID(row.Region)
		j, _ := slices.BinarySearchFunc(m.Jurisdictions, row.Jurisdiction, func(e Jurisdiction, id string) int { return cmp.Compare(e.ID, id) })
		member[r] = int32(j)
	}
	t, err := pol.Build(polityParentArray(m), member, MaxPolityDepth)
	if err != nil {
		invariant("the polity of a validated configuration failed to build: " + err.Error())
	}
	p := Polity{names: make([]string, len(m.Jurisdictions)), seats: make([]int32, len(m.Jurisdictions)), t: t}
	for i, j := range m.Jurisdictions {
		p.names[i], p.seats[i] = j.ID, -1
		if j.Seat != "" {
			r, _ := c.regionID(j.Seat)
			p.seats[i] = int32(r)
			p.gov = append(p.gov, JurisdictionID(i))
		}
	}
	return p
}

// narrow converts a public ID to the int32 of internal/polity, -1 (none) when it
// does not fit, so an ID beyond int32 cannot alias a low index by truncation.
func narrow(i int) int32 {
	if i < math.MinInt32 || i > math.MaxInt32 {
		return -1
	}
	return int32(i)
}

// clone returns a deep copy that shares no storage with p.
func (p Polity) clone() Polity {
	return Polity{names: slices.Clone(p.names), seats: slices.Clone(p.seats), gov: slices.Clone(p.gov), t: p.t.Clone()}
}

// jurisdictions converts a list of internal indexes to a fresh list of IDs, nil
// when empty.
func jurisdictions(in []int32) []JurisdictionID {
	if len(in) == 0 {
		return nil
	}
	out := make([]JurisdictionID, len(in))
	for i, j := range in {
		out[i] = JurisdictionID(j)
	}
	return out
}

// Len returns the number of jurisdictions.
func (p Polity) Len() int { return len(p.names) }

// Name returns the ID string of a jurisdiction, or "" if j is not one.
func (p Polity) Name(j JurisdictionID) string {
	if j < 0 || int(j) >= len(p.names) {
		return ""
	}
	return p.names[j]
}

// Lookup returns the JurisdictionID of the jurisdiction with the given ID string.
func (p Polity) Lookup(id string) (JurisdictionID, bool) {
	i, ok := slices.BinarySearch(p.names, id)
	if !ok {
		return NoJurisdiction, false
	}
	return JurisdictionID(i), true
}

// Parent returns the parent of j; false for a root or a j that is not a
// jurisdiction.
func (p Polity) Parent(j JurisdictionID) (JurisdictionID, bool) {
	par, ok := p.t.Parent(narrow(int(j)))
	if !ok {
		return NoJurisdiction, false
	}
	return JurisdictionID(par), true
}

// Children returns the children of j, ascending by JurisdictionID.
func (p Polity) Children(j JurisdictionID) []JurisdictionID {
	return jurisdictions(p.t.Children(narrow(int(j))))
}

// Depth returns the number of ancestors of j, a root being 0; false for a j that is
// not a jurisdiction.
func (p Polity) Depth(j JurisdictionID) (int, bool) { return p.t.Depth(narrow(int(j))) }

// Of returns the direct jurisdiction of region r; false if it has none.
func (p Polity) Of(r RegionID) (JurisdictionID, bool) {
	j, ok := p.t.Of(narrow(int(r)))
	if !ok {
		return NoJurisdiction, false
	}
	return JurisdictionID(j), true
}

// Chain returns the jurisdictions of region r, its direct jurisdiction first and
// the root last; empty if it has none.
func (p Polity) Chain(r RegionID) []JurisdictionID { return jurisdictions(p.t.Chain(narrow(int(r)))) }

// Covers reports whether jurisdiction j covers region r: j is in the chain of r.
func (p Polity) Covers(j JurisdictionID, r RegionID) bool {
	return p.t.Covers(narrow(int(j)), narrow(int(r)))
}

// Common returns the innermost jurisdiction that covers both regions, or
// NoJurisdiction if there is none.
func (p Polity) Common(a, b RegionID) JurisdictionID {
	return JurisdictionID(p.t.Common(narrow(int(a)), narrow(int(b))))
}

// Regions returns the regions jurisdiction j covers, ascending by RegionID, the
// direct members of its descendants included.
func (p Polity) Regions(j JurisdictionID) []RegionID {
	in := p.t.Covered(narrow(int(j)))
	if len(in) == 0 {
		return nil
	}
	out := make([]RegionID, len(in))
	for i, r := range in {
		out[i] = RegionID(r)
	}
	return out
}

// Crossing returns what a good leaves and enters in going from one region to the
// other (spec-polity.md 4.1.1). It is defined for any two regions, whether or not
// an edge joins them, and does not depend on the status of an edge.
func (p Polity) Crossing(from, to RegionID) Crossing {
	exit, enter := p.t.Crossing(narrow(int(from)), narrow(int(to)))
	return Crossing{Exit: jurisdictions(exit), Enter: jurisdictions(enter)}
}

// Between lists, in ascending EdgeID, the edges of g whose crossing has a in Exit
// and b in Enter: an edge that leaves a region a covers and enters one b covers and
// is internal to neither. A closed edge is listed. Two nested jurisdictions, or one
// with itself, have no edge between them. It costs O(edges * depth).
func (p Polity) Between(g Graph, a, b JurisdictionID) []EdgeID {
	var out []EdgeID
	for e := range g.Edges() {
		v, _ := g.EdgeAt(EdgeID(e))
		if p.t.EdgeBetween(int32(v.From), int32(v.To), narrow(int(a)), narrow(int(b))) {
			out = append(out, v.ID)
		}
	}
	return out
}

// Seat returns the region jurisdiction j is governed from; false for a grouping or a
// j that is not a jurisdiction.
func (p Polity) Seat(j JurisdictionID) (RegionID, bool) {
	if j < 0 || int(j) >= len(p.seats) || p.seats[j] < 0 {
		return NoRegion, false
	}
	return RegionID(p.seats[j]), true
}

// Governed returns the governed jurisdictions, ascending by JurisdictionID.
func (p Polity) Governed() []JurisdictionID { return slices.Clone(p.gov) }

// polity is the state of the module, nil while it is off (spec-polity.md 4.4.1):
// the derived tables, built once at construction from the configuration and saved
// nowhere (spec-polity.md 11).
type polity struct {
	v Polity
}

// newPolity builds the state of a canonical configuration the validator accepted,
// nil when the module is off.
func newPolity(c *Config) *polity {
	if c.Modules.Polity == nil {
		return nil
	}
	return &polity{v: buildPolity(c)}
}

// Polity returns the jurisdiction tree and the membership of the regions: a deep
// copy built fresh on every call, the empty Polity if the module is off. It cannot
// change in a run, so a consumer calls it once and keeps the value. It works on a
// failed world (spec-polity.md 4.2.4).
func (w *World) Polity() Polity {
	if w.polity == nil {
		return Polity{}
	}
	return w.polity.v.clone()
}

// polityNames returns the names table of the jurisdictions, nil while the module is
// off: the IDs in ascending order, the index being the JurisdictionID.
func polityNames(c *Config) []string {
	m := c.Modules.Polity
	if m == nil {
		return nil
	}
	out := make([]string, len(m.Jurisdictions))
	for i := range m.Jurisdictions {
		out[i] = m.Jurisdictions[i].ID
	}
	return out
}

// polityCounts returns the number of jurisdictions and of governed ones, for the
// construction summary.
func polityCounts(c *Config) (total, governed int) {
	for i := range c.Modules.Polity.Jurisdictions {
		if c.Modules.Polity.Jurisdictions[i].Seat != "" {
			governed++
		}
	}
	return len(c.Modules.Polity.Jurisdictions), governed
}
