package economysim

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ownuse"
)

// This file is the validator and the advice of the subsistence module
// (spec-subsistence.md 13), split from subsistence.go, which holds the module.

// validateSubsistence is the module's part of phases 2, 3 and 4 of
// spec-config.md 7.1.6 (spec-subsistence.md 13.1, 13.2). It reports nothing
// when the module is off. The order is the bounds, the band, the duplicates, the
// references, and then the rules of each effective list.
func (v *validator) validateSubsistence() {
	m := v.c.Modules.Subsistence
	if m == nil {
		return
	}
	base := []string{"modules", "subsistence"}
	if len(m.Roles) > MaxRoles {
		v.rep.add(ErrOutOfBounds, path(base, "roles"), itoa(len(m.Roles)), itoa(MaxRoles))
		return // the rest is linear in the count it just refused
	}
	enterOK := v.inRange(path(base, "enter_ticks"), m.EnterTicks, 1, MaxOwnUseTicks)
	exitOK := v.inRange(path(base, "exit_ticks"), m.ExitTicks, 1, MaxOwnUseTicks)
	v.inRange(path(base, "retry_jobs"), m.RetryJobs, 0, MaxRetryJobs)
	if enterOK && exitOK && m.ExitTicks <= m.EnterTicks {
		v.rep.add(ErrOwnUseBand, path(base, "exit_ticks"), i64(m.ExitTicks), i64(m.EnterTicks))
	}
	v.ownUseRefs(path(base, "recipes"), m.Recipes)
	seen := make(map[string]bool, len(m.Roles))
	for i := range m.Roles {
		row := &m.Roles[i]
		rb := path(base, "role", row.Role)
		if seen[row.Role] {
			v.rep.add(ErrDuplicateID, path(rb, "role"), row.Role, "")
		}
		seen[row.Role] = true
		if v.lk.role[row.Role] == nil {
			v.rep.add(ErrUnknownRole, path(rb, "role"), row.Role, "")
		}
		v.ownUseRefs(path(rb, "recipes"), row.Recipes)
	}
	if m.withinBounds() { // the rules sort and scan a list once per role
		v.ownUseRules(m, base)
	}
}

// withinBounds reports whether the role rows and every list are within their
// bounds, MaxRoles and MaxOwnUseRecipes (spec-subsistence.md 3.4.4). The rules
// over the effective lists and the warnings run only then, so an oversized list
// of a file is reported once and not sorted and scanned for every role. A nil
// module (off) reports false: it has no lists to read.
func (s *SubsistenceConfig) withinBounds() bool {
	if s == nil || len(s.Recipes) > MaxOwnUseRecipes || len(s.Roles) > MaxRoles {
		return false
	}
	for i := range s.Roles {
		if len(s.Roles[i].Recipes) > MaxOwnUseRecipes {
			return false
		}
	}
	return true
}

// ownUseRefs checks one list: at most MaxOwnUseRecipes, each recipe defined and
// none twice (spec-subsistence.md 3.4.4).
func (v *validator) ownUseRefs(lb []string, list []string) {
	if len(list) > MaxOwnUseRecipes {
		v.rep.add(ErrOutOfBounds, lb, itoa(len(list)), itoa(MaxOwnUseRecipes))
		return
	}
	seen := make(map[string]bool, len(list))
	for _, id := range list {
		if v.lk.recipe[id] == nil {
			v.rep.add(ErrUnknownRecipe, path(lb, id), id, "")
		}
		if seen[id] {
			v.rep.add(ErrDuplicateID, path(lb, id), id, "")
		}
		seen[id] = true
	}
}

// ownUseRules checks the three rules of an own-use recipe (spec-subsistence.md
// 3.4.1 to 3.4.3) over the effective list of every role: a source, sited, and
// not also a recipe of the same role. A recipe on the default list breaks the
// first two once, at the default list; one on a row, at the row. The overlap is a
// property of a (role, recipe) pair and is reported under the role.
func (v *validator) ownUseRules(m *SubsistenceConfig, base []string) {
	for _, id := range slices.Compact(sortedBy(m.Recipes, cmp.Compare[string])) {
		v.ownUseKind(path(base, "recipes", id), id)
	}
	for i := range m.Roles {
		row := &m.Roles[i]
		for _, id := range slices.Compact(sortedBy(row.Recipes, cmp.Compare[string])) {
			v.ownUseKind(path(base, "role", row.Role, "recipes", id), id)
		}
	}
	for i := range v.c.Roles {
		ro := &v.c.Roles[i]
		for _, id := range slices.Compact(m.ownUseList(ro.ID)) {
			if slices.Contains(ro.Recipes, id) {
				v.rep.add(ErrOwnUseOverlap, path(base, "role", ro.ID, "recipes", id), id, "")
			}
		}
	}
}

// ownUseKind reports a recipe that is not a sited source (spec-subsistence.md
// 3.4.1, 3.4.2) at the given path; an undefined recipe is the reference check's.
func (v *validator) ownUseKind(p []string, id string) {
	rc := v.lk.recipe[id]
	if rc == nil {
		return
	}
	if len(rc.Inputs) > 0 || len(rc.Requires) > 0 {
		v.rep.add(ErrOwnUseNotSource, p, id, "")
	}
	if !rc.Sited {
		v.rep.add(ErrOwnUseNotSited, p, id, "")
	}
}

// subsistenceWarnings returns the warnings of spec-subsistence.md 13.3 that need
// no solver and no yield table, in no particular order. They read a
// configuration that may not validate, so each guards its lookups. The four that
// read the activities of a region are the viability pass's (ownUse).
func subsistenceWarnings(c *Config, lk lookup) []Warning {
	m := c.Modules.Subsistence
	if !m.withinBounds() { // off, or out of bounds, which Validate reports
		return nil
	}
	var out []Warning
	add := func(rule string, segs ...string) { out = append(out, Warning{Path: segs, Rule: rule}) }
	base := []string{"modules", "subsistence"}
	any := false
	for i := range c.Roles {
		ro := &c.Roles[i]
		list := m.ownUseList(ro.ID)
		if len(list) == 0 {
			continue
		}
		any = true
		if len(coveredSet(c, lk, ro, list)) == 0 {
			add("own_use_no_upkeep", path(base, "role", ro.ID)...)
		}
		for _, id := range list {
			if rc := lk.recipe[id]; rc != nil && rc.Duration-1 >= c.World.ChangePrivation {
				add("own_use_slower_than_ladder", "recipes", id)
			}
		}
	}
	if !any {
		add("subsistence_empty", base...)
	}
	if m.ExitTicks > c.World.CoverTicks {
		add("own_use_exit_unreachable", path(base, "exit_ticks")...)
	}
	return dedupWarnings(out)
}

// dedupWarnings drops a warning whose rule and path repeat an earlier one: a
// recipe on the lists of many roles is one warning.
func dedupWarnings(w []Warning) []Warning {
	out := w[:0]
	for _, x := range w {
		if !slices.ContainsFunc(out, func(y Warning) bool { return y.Rule == x.Rule && slices.Equal(y.Path, x.Path) }) {
			out = append(out, x)
		}
	}
	return out
}

// coveredSet returns the covered set of a role for a list (spec-subsistence.md
// 3.3.2): the commodities of its upkeep in any region that are a non-discard
// output of a recipe on the list, ascending.
func coveredSet(c *Config, lk lookup, ro *Role, list []string) []string {
	upkeep := map[string]bool{}
	for _, it := range effectiveUpkeep(c, ro) {
		upkeep[it.Commodity] = true
	}
	for i := range c.Regions {
		for _, it := range c.Regions[i].ExtraUpkeep {
			upkeep[it.Commodity] = true
		}
	}
	var out []string
	for _, id := range list {
		rc := lk.recipe[id]
		if rc == nil {
			continue
		}
		for _, o := range rc.Outputs {
			if !o.Discard && upkeep[o.Commodity] {
				out = append(out, o.Commodity)
			}
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// ownUse raises the warnings of spec-subsistence.md 13.3 that read the
// activities of a region, once per (role, own-use recipe) with a household in the
// region: own_use_partial and own_use_trap for a floor under 10 000 (4.5), and
// own_use_beats_market for a floor at or above the autarky surplus factor of the
// region (4.2); and own_use_not_available once per role with a household where
// none of its own-use recipes can run. surplus is the autarky SurplusBP of the
// region, read when haveSurplus says the programme was solved (the pooled figure
// is in the solver's report alone, spec-viability.md 10.3).
func (m *viaModel) ownUse(o *viabilityOutcome, r RegionID, surplus int64, haveSurplus bool) {
	if m.c.Modules.Subsistence == nil {
		return
	}
	rid := m.c.Regions[r].ID
	var done []RoleID
	for _, g := range m.c.Groups {
		if g.Region != rid {
			continue
		}
		ro, _ := m.c.roleID(g.Role)
		if slices.Contains(done, ro) {
			continue
		}
		done = append(done, ro)
		m.ownUseRole(o, r, ro, surplus, haveSurplus)
	}
}

// ownUseRole raises the warnings of one role with a household in a region: one
// per own-use recipe that runs there, and own_use_not_available when none does.
func (m *viaModel) ownUseRole(o *viabilityOutcome, r RegionID, ro RoleID, surplus int64, haveSurplus bool) {
	row := m.roles.row(ro)
	if len(row.ownUse) == 0 {
		return
	}
	runnable := false
	for _, k := range row.ownUse {
		if m.ownUseRecipe(o, r, ro, k, surplus, haveSurplus) {
			runnable = true
		}
	}
	if !runnable {
		o.warn("own_use_not_available", "regions", m.c.Regions[r].ID, "roles", m.c.Roles[ro].ID)
	}
}

// ownUseRecipe raises the floor warnings of one own-use recipe for a role in a
// region and reports whether the recipe runs there. A recipe that makes none of
// the role's upkeep goods has no floor figure and draws none (own_use_no_upkeep
// is the role's).
func (m *viaModel) ownUseRecipe(o *viabilityOutcome, r RegionID, ro RoleID, k RecipeID, surplus int64, haveSurplus bool) bool {
	a, ok := m.activity(r, k)
	if !ok {
		return false
	}
	floor, covers := m.floorBP(a, m.roles.row(ro), r)
	if !covers {
		return true
	}
	p := []string{"regions", m.c.Regions[r].ID, "roles", m.c.Roles[ro].ID, "recipes", m.c.Recipes[k].ID}
	if floor < wholeBP {
		o.warn("own_use_partial", p...)
		if m.c.Modules.Subsistence.RetryJobs == 0 {
			o.warn("own_use_trap", p...)
		}
	}
	if haveSurplus && floor >= surplus {
		o.warn("own_use_beats_market", p...)
	}
	return true
}

// floorBP is the floor figure of an activity for a role in a region: the
// greatest coverage over the role's upkeep goods that the recipe makes, at the
// mean factor of the region (spec-subsistence.md 4.1, 4.2, FR-SUB-039). covers is
// false when the recipe makes none of them.
func (m *viaModel) floorBP(a viaActivity, row *roleRow, r RegionID) (floor int64, covers bool) {
	e := a.act.FactorSum / a.act.Starts
	if len(a.act.Draws) > 0 {
		e, _ = checked.MulDivDiv(a.act.FactorSum, a.act.AbundanceBP, a.act.Starts, wholeBP)
	}
	var covs []int64
	for _, u := range row.upkeepIn(r) {
		for _, out := range a.act.Outputs {
			if out.Discard || out.Commodity != int(u.commodity) {
				continue
			}
			cov, _ := ownuse.Cov(out.Qty, e, a.act.Duration, int64(u.quantity))
			covs = append(covs, cov)
		}
	}
	return ownuse.FloorBP(covs), len(covs) > 0
}
