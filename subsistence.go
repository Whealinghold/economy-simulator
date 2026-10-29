package economysim

import (
	"cmp"
	"slices"
)

// This file is the subsistence module (spec-subsistence.md): the canonical copy
// of its configuration, the effective own-use list of a role and its join into
// the role table. The validator and the warnings are in subsistence_validate.go,
// the types in config.go and the pure rules in internal/ownuse.

// canonical returns a copy with the default list and every row's list sorted by
// recipe ID and the rows sorted by role (spec-subsistence.md 3.2.3), so two
// spellings of one configuration hash alike. A nil module stays nil; a module
// that is on and empty stays on.
func (s *SubsistenceConfig) canonical() *SubsistenceConfig {
	if s == nil {
		return nil
	}
	out := &SubsistenceConfig{
		Recipes:    sortedBy(s.Recipes, cmp.Compare[string]),
		Roles:      sortedBy(s.Roles, func(a, b RoleOwnUse) int { return cmp.Compare(a.Role, b.Role) }),
		EnterTicks: s.EnterTicks,
		ExitTicks:  s.ExitTicks,
		RetryJobs:  s.RetryJobs,
	}
	for i := range out.Roles {
		out.Roles[i].Recipes = sortedBy(out.Roles[i].Recipes, cmp.Compare[string])
	}
	return out
}

// ownUseList returns the effective own-use list of a role (spec-subsistence.md
// 3.2.1, 3.3.1): the list of its row if it has one, an empty row meaning none,
// and the module's default list otherwise, sorted by recipe ID. It is nil when
// the module is off or the list is empty, and a copy the caller may keep.
func (s *SubsistenceConfig) ownUseList(role string) []string {
	if s == nil {
		return nil
	}
	list := s.Recipes
	for i := range s.Roles {
		if s.Roles[i].Role == role {
			list = s.Roles[i].Recipes
			break
		}
	}
	return sortedBy(list, cmp.Compare[string])
}

// ownUseJobs resolves the effective own-use list of a role of a configuration
// that passed Validate and returns it with the union of the role's own recipes
// and that list, both ascending by RecipeID (architecture.md 2.10.1: the join
// into the role table). own is nil and jobs is recipes itself when the module is
// off (a nil receiver) or the list is empty, so a module that does nothing
// changes no slice.
func (s *SubsistenceConfig) ownUseJobs(c *Config, role string, recipes []RecipeID) (own, jobs []RecipeID) {
	list := s.ownUseList(role)
	if len(list) == 0 {
		return nil, recipes
	}
	for _, id := range list {
		rid, ok := c.recipeID(id)
		if !ok {
			// Not covered by a test: validation rejects an unknown recipe before this
			// runs, so no valid configuration reaches the panic (testing.md 9.5).
			panic("economysim: invariant violated: role " + role + " has unknown own-use recipe " + id + " after validation")
		}
		own = append(own, rid)
	}
	slices.Sort(own)
	jobs = append(slices.Clone(recipes), own...)
	slices.Sort(jobs)
	return own, jobs
}

// subsistenceItems adds the non-discard outputs of the effective own-use list
// of a role to its tradable set, so a household may hold and sell what it makes
// (spec-subsistence.md 3.6, spec-agents.md 3.4). It is the module's call in the
// fifth fan-in, tradableItems.
func subsistenceItems(c *Config, lk lookup, role *Role, set map[string]bool) {
	for _, id := range c.Modules.Subsistence.ownUseList(role.ID) {
		rc := lk.recipe[id]
		if rc == nil {
			continue
		}
		for _, o := range rc.Outputs {
			if !o.Discard {
				set[o.Commodity] = true
			}
		}
	}
}

// ownUseOnly returns, by recipe, whether it is on the own-use list of some role
// and on no role's Recipes: such a recipe is no activity of the reference
// programme and makes no commodity producible (spec-viability.md 3.1, 4.2,
// FR-SUB-037). The floor is for when the markets fail, so the steady state does
// not stand on it.
func (m *viaModel) ownUseOnly() []bool {
	own, market := make([]bool, len(m.c.Recipes)), make([]bool, len(m.c.Recipes))
	for i := range m.c.Roles {
		row := m.roles.row(RoleID(i))
		for _, k := range row.ownUse {
			own[k] = true
		}
		for _, k := range row.recipes {
			market[k] = true
		}
	}
	for k := range own {
		own[k] = own[k] && !market[k]
	}
	return own
}
