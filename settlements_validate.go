package economysim

import (
	"maps"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/settlement"
)

// This file is the validation of the settlements module (spec-settlements.md 9; the
// continuation file of settlements.go, architecture.md 2.6): the rules of 9.1 that
// Validate checks and the warnings of 9.2. The rules that read a role (needs_unknown,
// ceiling_unknown, the module dependency) run from the module's seam, after the roles
// are validated, and not inside roleRules.

// validateSettlements checks the module and the two role fields it owns
// (spec-settlements.md 9.1, FR-SET-001 to 007, 021). It checks the role fields
// whether or not the module is on, because a Needs or a Ceiling with the module off
// is ErrModuleDependency. The module's rules run in phases and a phase that finds an
// error ends the run, so the first error is the same for the same mistake: the count
// of classes, each class, the tunables, the order of the Enter values, the region
// rows.
func (v *validator) validateSettlements() {
	m := v.c.Modules.Settlements
	if m != nil {
		base := []string{"modules", "settlements"}
		steps := []func() bool{
			func() bool { return v.settlementClasses(base, m) },
			func() bool { return v.settlementTunables(base, m) },
			func() bool { return v.settlementOrder(base, m) },
			func() bool { return v.settlementRows(base, m) },
		}
		for _, step := range steps {
			if !step() {
				break
			}
		}
	}
	v.settlementRoleFields(m)
}

// settlementClasses checks the count of classes (rule no_class and ErrTooMany), then
// each class: ID, duplicate, Enter range, flags, limits and label, and the count of
// distinct flags and limits over the module.
func (v *validator) settlementClasses(base []string, m *Settlements) bool {
	switch n := len(m.Classes); {
	case n == 0:
		v.rep.add(ErrTooMany, path(base, "classes"), "0", "1") // rule no_class
		return false
	case n > MaxSettlementClasses:
		v.rep.add(ErrTooMany, path(base, "classes"), itoa(n), itoa(MaxSettlementClasses))
		return false
	}
	n := len(v.rep.errs)
	seen := make(map[string]bool, len(m.Classes)) // a membership test only
	flags, limits := map[string]bool{}, map[string]bool{}
	for i := range m.Classes {
		c := &m.Classes[i]
		p := path(base, "classes["+itoa(i)+"]")
		switch {
		case !validID(c.ID):
			v.rep.add(ErrInvalidID, path(p, "id"), c.ID, "")
		case seen[c.ID]:
			v.rep.add(ErrDuplicateID, path(p, "id"), c.ID, "")
		}
		seen[c.ID] = true
		v.inRange(path(p, "enter"), c.Enter, 0, MaxHouseholds)
		v.label(p, "label", c.Label)
		v.classFlags(p, c, flags)
		v.classLimits(p, c, limits)
	}
	v.distinctCount(path(base, "flags"), len(flags), MaxClassFlags)
	v.distinctCount(path(base, "limits"), len(limits), MaxClassLimits)
	return len(v.rep.errs) == n
}

// distinctCount reports ErrTooMany for a count of distinct names above its bound.
func (v *validator) distinctCount(p []string, n, hi int) {
	if n > hi {
		v.rep.add(ErrTooMany, p, itoa(n), itoa(hi))
	}
}

// classFlags checks the flags of one class (a name, once) and records them in all.
func (v *validator) classFlags(p []string, c *SettlementClass, all map[string]bool) {
	own := make(map[string]bool, len(c.Flags))
	for j, f := range c.Flags {
		fp := path(p, "flags["+itoa(j)+"]")
		switch {
		case !validID(f):
			v.rep.add(ErrInvalidID, fp, f, "")
		case own[f]:
			v.rep.add(ErrDuplicateID, fp, f, "")
		}
		own[f], all[f] = true, true
	}
}

// classLimits checks the limits of one class (a name, once, a value in 0 .. MaxQuantity)
// and records the names in all.
func (v *validator) classLimits(p []string, c *SettlementClass, all map[string]bool) {
	own := make(map[string]bool, len(c.Limits))
	for j, l := range c.Limits {
		lp := path(p, "limits["+itoa(j)+"]")
		switch {
		case !validID(l.Name):
			v.rep.add(ErrInvalidID, path(lp, "name"), l.Name, "")
		case own[l.Name]:
			v.rep.add(ErrDuplicateID, path(lp, "name"), l.Name, "")
		}
		own[l.Name], all[l.Name] = true, true
		v.inRange(path(lp, "value"), l.Value, 0, MaxQuantity)
	}
}

// settlementTunables checks the band and the dwell.
func (v *validator) settlementTunables(base []string, m *Settlements) bool {
	n := len(v.rep.errs)
	v.ranges(base, []fieldRange{{"band_bp", m.BandBP, 0, MaxBandBP}, {"dwell", m.Dwell, 0, MaxWelfareWindow}})
	return len(v.rep.errs) == n
}

// settlementOrder checks the Enter values in the sorted order: the smallest is 0
// (rule first_class_open) and no two are equal (rule enter_order), so the order is
// total. A path names the class in the order it was written.
func (v *validator) settlementOrder(base []string, m *Settlements) bool {
	idx := make([]int, len(m.Classes))
	for i := range idx {
		idx[i] = i
	}
	slices.SortStableFunc(idx, func(a, b int) int { return int(sign(m.Classes[a].Enter - m.Classes[b].Enter)) })
	n := len(v.rep.errs)
	if first := idx[0]; m.Classes[first].Enter != 0 {
		v.rep.add(ErrOutOfBounds, path(base, "classes["+itoa(first)+"]", "enter"), i64(m.Classes[first].Enter), "0")
	}
	for k := 1; k < len(idx); k++ {
		if cur := idx[k]; m.Classes[cur].Enter == m.Classes[idx[k-1]].Enter {
			v.rep.add(ErrOutOfBounds, path(base, "classes["+itoa(cur)+"]", "enter"), i64(m.Classes[cur].Enter), i64(m.Classes[cur].Enter+1))
		}
	}
	return len(v.rep.errs) == n
}

// sign returns -1, 0 or 1 for the sign of n.
func sign(n int64) int64 {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

// settlementRows checks the region rows (spec-settlements.md 3.5, 9.1): a defined
// region that is not an outside node, once, a scale in 1 .. MaxThresholdBP and a
// Floor that is a defined class.
func (v *validator) settlementRows(base []string, m *Settlements) bool {
	if len(m.Regions) > MaxRegions {
		v.rep.add(ErrTooMany, path(base, "regions"), itoa(len(m.Regions)), itoa(MaxRegions))
		return false
	}
	n := len(v.rep.errs)
	classes := make(map[string]bool, len(m.Classes))
	for i := range m.Classes {
		classes[m.Classes[i].ID] = true
	}
	seen := make(map[string]bool, len(m.Regions))
	for i := range m.Regions {
		r := &m.Regions[i]
		p := path(base, "regions["+itoa(i)+"]")
		v.settlementRowRegion(p, r.Region, seen)
		v.inRange(path(p, "threshold_bp"), r.ThresholdBP, 1, MaxThresholdBP)
		if r.Floor != "" && !classes[r.Floor] {
			v.rep.add(ErrUnknownSettlementClass, path(p, "floor"), r.Floor, "")
		}
	}
	return len(v.rep.errs) == n
}

// settlementRowRegion checks the region of one row: defined, not an outside node,
// not named by an earlier row.
func (v *validator) settlementRowRegion(p []string, region string, seen map[string]bool) {
	switch {
	case v.lk.region[region] == nil:
		v.rep.add(ErrUnknownRegion, path(p, "region"), region, "")
	case v.outsideSet()[region]:
		v.rep.add(ErrOutsideNode, path(p, "region"), region, "")
	case seen[region]:
		v.rep.add(ErrDuplicateID, path(p, "region"), region, "")
	}
	seen[region] = true
}

// settlementRoleFields checks Role.Needs and Role.Ceiling (spec-settlements.md 9.1):
// with the module off a value is ErrModuleDependency; with it on the flag must be one
// some class carries (ErrUnknownSettlementFlag) and the limit one some class carries
// (ErrUnknownLimit).
func (v *validator) settlementRoleFields(m *Settlements) {
	for i := range v.c.Roles {
		r := &v.c.Roles[i]
		rb := []string{"roles", r.ID}
		if r.Needs != "" {
			v.settlementFlagRule(path(rb, "needs"), r.Needs)
		}
		if r.Ceiling == "" {
			continue
		}
		switch {
		case m == nil:
			v.rep.add(ErrModuleDependency, path(rb, "ceiling"), r.Ceiling, "settlements")
		case !m.limitSet()[r.Ceiling]:
			v.rep.add(ErrUnknownLimit, path(rb, "ceiling"), r.Ceiling, "")
		}
	}
}

// settlementFlagRule is the rule of a reader that names a flag, at path p (a role's
// Needs or a store class's Needs): the module must be on (ErrModuleDependency) and
// some class must carry the flag (ErrUnknownSettlementFlag).
func (v *validator) settlementFlagRule(p []string, flag string) {
	m := v.c.Modules.Settlements
	switch {
	case m == nil:
		v.rep.add(ErrModuleDependency, p, flag, "settlements")
	case !m.flagSet()[flag]:
		v.rep.add(ErrUnknownSettlementFlag, p, flag, "")
	}
}

// flagSet returns the flags some class carries (a membership set, never ranged over
// for a result).
func (m *Settlements) flagSet() map[string]bool {
	out := map[string]bool{}
	for i := range m.Classes {
		for _, f := range m.Classes[i].Flags {
			out[f] = true
		}
	}
	return out
}

// limitSet returns the limit names some class carries.
func (m *Settlements) limitSet() map[string]bool {
	out := map[string]bool{}
	for i := range m.Classes {
		for _, l := range m.Classes[i].Limits {
			out[l.Name] = true
		}
	}
	return out
}

// settlementThresholds returns the effective enter and leave thresholds of every
// region that is not an outside node, in the order of c.Regions, from the canonical
// classes of m. ok is false when the module has no class or an arithmetic step does
// not fit, which validation prevents.
func settlementThresholds(c *Config, m *Settlements) (e, l [][]int64, ok bool) {
	enter := make([]int64, len(m.Classes))
	for k := range m.Classes {
		enter[k] = m.Classes[k].Enter
	}
	scale := make(map[string]int64, len(m.Regions)) // lookup only
	for _, row := range m.Regions {
		scale[row.Region] = row.ThresholdBP
	}
	e, l = make([][]int64, len(c.Regions)), make([][]int64, len(c.Regions))
	for i := range c.Regions {
		s, has := scale[c.Regions[i].ID]
		if !has {
			s = 10_000
		}
		var okR bool
		if e[i], l[i], okR = settlement.Thresholds(enter, s, m.BandBP); !okR {
			return nil, nil, false
		}
	}
	return e, l, true
}

// settlementCensus is what the warnings of 9.2 read: the effective thresholds of each
// region, its households and class at construction (spec-settlements.md 4.1.1, 4.5.1:
// the census of the groups, with no band), and the households of each (region, role)
// group. It is built on the canonical copy of the module.
type settlementCensus struct {
	m       *Settlements
	regions []Region
	outside map[string]bool
	e, l    [][]int64
	hh      []int64
	class   []int
	byRole  map[[2]string]int64 // lookup only
}

// newSettlementCensus builds the census of c; ok is false when the module has no
// class or a threshold does not fit.
func newSettlementCensus(c *Config, m *Settlements) (s settlementCensus, ok bool) {
	e, l, ok := settlementThresholds(c, m)
	if !ok {
		return s, false
	}
	s = settlementCensus{m: m, regions: c.Regions, outside: outsideNodeSet(c), e: e, l: l,
		hh: make([]int64, len(c.Regions)), class: make([]int, len(c.Regions)), byRole: make(map[[2]string]int64, len(c.Groups))}
	at := make(map[string]int, len(c.Regions))
	for i := range c.Regions {
		at[c.Regions[i].ID] = i
	}
	for _, g := range c.Groups {
		if i, in := at[g.Region]; in {
			s.hh[i] = satAdd(s.hh[i], g.Households)
			key := [2]string{g.Region, g.Role}
			s.byRole[key] = satAdd(s.byRole[key], g.Households)
		}
	}
	floors := make(map[string]string, len(m.Regions))
	for _, row := range m.Regions {
		floors[row.Region] = row.Floor
	}
	for i := range c.Regions {
		s.class[i] = settlement.Initial(s.hh[i], e[i], m.classIndex(floors[c.Regions[i].ID]))
	}
	return s, true
}

// classIndex returns the index of the class with the ID, 0 for none or unknown.
func (m *Settlements) classIndex(id string) int {
	for k := range m.Classes {
		if m.Classes[k].ID == id {
			return k
		}
	}
	return 0
}

// settlementsWarnings are the warnings of spec-settlements.md 9.2, none of which
// fails validation. It reports nothing when the module is off or has no class, and it
// never panics on a configuration that does not validate.
func settlementsWarnings(c *Config) []Warning {
	cm := c.Modules.Settlements.canonical()
	if cm == nil || len(cm.Classes) == 0 {
		return nil
	}
	s, ok := newSettlementCensus(c, cm)
	if !ok {
		return nil
	}
	lk := newLookup(c)
	out := s.roleWarnings(c)
	out = append(out, s.dwellWarning(c, lk)...)
	out = append(out, s.thresholdWarnings()...)
	out = append(out, s.unreachableWarnings(c)...)
	return append(out, s.unusedWarnings(c)...)
}

// roleWarnings are needs_unmet_at_start and ceiling_below_start: for each role with
// the field, the count of regions where a group of the role starts without the flag or
// above the limit of the region's class.
func (s *settlementCensus) roleWarnings(c *Config) []Warning {
	var out []Warning
	for i := range c.Roles {
		r := &c.Roles[i]
		var unmet, above int64
		for ri := range s.regions {
			h, has := s.byRole[[2]string{s.regions[ri].ID, r.ID}]
			if !has || s.outside[s.regions[ri].ID] {
				continue
			}
			cl := &s.m.Classes[s.class[ri]]
			if r.Needs != "" && !slices.Contains(cl.Flags, r.Needs) {
				unmet++
			}
			if lim, ok := cl.limit(r.Ceiling); r.Ceiling != "" && ok && h > lim {
				above++
			}
		}
		out = appendCount(out, []string{"roles", r.ID, "needs"}, "needs_unmet_at_start", unmet)
		out = appendCount(out, []string{"roles", r.ID, "ceiling"}, "ceiling_below_start", above)
	}
	return out
}

// appendCount appends a warning with the count when the count is above 0.
func appendCount(out []Warning, p []string, rule string, n int64) []Warning {
	if n > 0 {
		out = append(out, Warning{Path: p, Rule: rule, Count: n})
	}
	return out
}

// limit returns the value of the class's limit of that name.
func (c *SettlementClass) limit(name string) (int64, bool) {
	for _, l := range c.Limits {
		if l.Name == name {
			return l.Value, true
		}
	}
	return 0, false
}

// dwellWarning is dwell_below_window: the count of roles with a Needs or a Ceiling
// whose effective welfare window is longer than the dwell.
func (s *settlementCensus) dwellWarning(c *Config, lk lookup) []Warning {
	var n int64
	for i := range c.Roles {
		r := &c.Roles[i]
		if r.Needs == "" && r.Ceiling == "" {
			continue
		}
		if window, ok := effectiveWindow(c, lk, r); ok && s.m.Dwell < window {
			n++
		}
	}
	return appendCount(nil, []string{"modules", "settlements", "dwell"}, "dwell_below_window", n)
}

// thresholdWarnings are band_one_household and threshold_collapsed: per class, the
// count of regions where its band is one household, or where its enter threshold
// equals that of the class below.
func (s *settlementCensus) thresholdWarnings() []Warning {
	var out []Warning
	for k := 1; k < len(s.m.Classes); k++ {
		var one, same int64
		for ri := range s.regions {
			if s.outside[s.regions[ri].ID] {
				continue
			}
			if s.e[ri][k]-s.l[ri][k] == 1 {
				one++
			}
			if s.e[ri][k] == s.e[ri][k-1] {
				same++
			}
		}
		p := []string{"modules", "settlements", "class", s.m.Classes[k].ID}
		out = appendCount(out, p, "band_one_household", one)
		out = appendCount(out, p, "threshold_collapsed", same)
	}
	return out
}

// unreachableWarnings is class_unreachable: a class whose smallest effective enter
// threshold over the regions exceeds the households of the whole realm at
// construction, with the population module off (nothing then adds households).
func (s *settlementCensus) unreachableWarnings(c *Config) []Warning {
	if c.Modules.Population != nil {
		return nil
	}
	var total int64
	for ri := range s.regions {
		total = satAdd(total, s.hh[ri])
	}
	var out []Warning
	for k := 1; k < len(s.m.Classes); k++ {
		least, any := int64(0), false
		for ri := range s.regions {
			if !s.outside[s.regions[ri].ID] && (!any || s.e[ri][k] < least) {
				least, any = s.e[ri][k], true
			}
		}
		if any && least > total {
			out = append(out, Warning{Path: []string{"modules", "settlements", "class", s.m.Classes[k].ID}, Rule: "class_unreachable"})
		}
	}
	return out
}

// unusedWarnings are flag_unused and limit_unused: a flag or a limit that no role
// (and, for a flag, no store class) names.
func (s *settlementCensus) unusedWarnings(c *Config) []Warning {
	used := map[string]bool{} // membership only
	usedLimit := map[string]bool{}
	for i := range c.Roles {
		used[c.Roles[i].Needs], usedLimit[c.Roles[i].Ceiling] = true, true
	}
	if st := c.Modules.Storage; st != nil {
		for i := range st.Classes {
			used[st.Classes[i].Needs] = true
		}
	}
	var out []Warning
	for _, f := range slices.Sorted(maps.Keys(s.m.flagSet())) {
		if !used[f] {
			out = append(out, Warning{Path: []string{"modules", "settlements", "flag", f}, Rule: "flag_unused"})
		}
	}
	for _, name := range slices.Sorted(maps.Keys(s.m.limitSet())) {
		if !usedLimit[name] {
			out = append(out, Warning{Path: []string{"modules", "settlements", "limit", name}, Rule: "limit_unused"})
		}
	}
	return out
}
