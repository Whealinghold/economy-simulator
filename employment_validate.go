package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	payroll "github.com/Whealinghold/economy-sim/internal/employment"
)

// This file is the validator and the warnings of the employment module, both
// forms (spec-employment.md 14, 24.4). The state, the payroll rule and the
// position of the payroll form are employment.go.

// validateEmployment checks the module and the role fields of both forms
// (spec-employment.md 14.1, 24.4.1): the bounds of the module's and the block's
// fields, the module dependency of Hireable and of the block, a role that is both,
// the one-form rule, the hire list, the weight of an employer, and, in the payroll
// form only, the module on with no payer role or no Hireable role (a no-op that
// would look like a result; in the hire form these are the warnings of 14.2).
func (v *validator) validateEmployment() {
	c := &v.c
	m := c.Modules.Employment
	on := m != nil
	hire := on && m.Hire
	base := []string{"modules", "employment"}
	v.employmentMaxOpen(path(base, "max_open"), m)
	var payers, workers int
	for i := range c.Roles {
		r := &c.Roles[i]
		v.employmentRole([]string{"roles", r.ID}, r, on, hire)
		if r.Employer != nil {
			payers++
		}
		if r.Hireable {
			workers++
		}
	}
	if !on || hire {
		return
	}
	if payers == 0 {
		v.rep.add(ErrEmploymentShape, path(base, "employer"), "0", "1")
	}
	if workers == 0 {
		v.rep.add(ErrEmploymentShape, path(base, "hireable"), "0", "1")
	}
}

// employmentMaxOpen checks the module's MaxOpen: in range in the hire form, zero
// in the payroll form (m is nil when the module is off, which has no field).
func (v *validator) employmentMaxOpen(p []string, m *Employment) {
	switch {
	case m == nil:
	case m.Hire:
		v.inRange(p, m.MaxOpen, 1, MaxHireRecords)
	case m.MaxOpen != 0:
		v.rep.add(ErrEmploymentShape, p, i64(m.MaxOpen), "0")
	}
}

// employmentRole checks the employment fields of one role; on says the module is
// present and hire that it is the hire form.
func (v *validator) employmentRole(rb []string, r *Role, on, hire bool) {
	if r.Hireable && !on {
		v.rep.add(ErrModuleDependency, path(rb, "hireable"), "employment", "")
	}
	if r.Hireable && r.Employer != nil {
		v.rep.add(ErrEmploymentShape, path(rb, "hireable"), "true", "false")
	}
	if r.Employer == nil {
		return
	}
	eb := path(rb, "employer")
	if !on {
		v.rep.add(ErrModuleDependency, eb, "employment", "")
	}
	if hire {
		v.hireEmployer(eb, r)
		return
	}
	v.inRange(path(eb, "payroll_bp"), r.Employer.PayrollBP, 1, payroll.WholeBP)
	if on && hasHireFields(r.Employer) {
		v.rep.add(ErrEmploymentShape, eb, "hire fields in the payroll form", "none")
	}
}

// hasHireFields says the block sets a field of the hire form.
func hasHireFields(b *Employer) bool {
	return len(b.Recipes) > 0 || b.MaxHires != 0 || b.WorkerShareBP != 0 || b.HireYieldBP != 0
}

// hireEmployer checks the block of a role in the hire form: no PayrollBP, the four
// fields in their bounds, the hire list and the weight of the role's groups.
func (v *validator) hireEmployer(eb []string, r *Role) {
	b := r.Employer
	if b.PayrollBP != 0 {
		v.rep.add(ErrEmploymentShape, path(eb, "payroll_bp"), i64(b.PayrollBP), "0")
	}
	v.ranges(eb, []fieldRange{
		{"max_hires", b.MaxHires, 1, MaxEstablishment},
		{"worker_share_bp", b.WorkerShareBP, 1, payroll.WholeBP},
		{"hire_yield_bp", b.HireYieldBP, 1, MaxYieldBP},
	})
	v.hireList(path(eb, "recipes"), r)
	v.employerWeight(r)
}

// hireList checks the hire list: not empty, on a role that has recipes, each
// entry a known recipe on the role's own list, none twice (spec-employment.md
// 3.2, 14.1).
func (v *validator) hireList(p []string, r *Role) {
	b := r.Employer
	if len(r.Recipes) == 0 { // a hire list needs a recipe of the role to name
		v.rep.add(ErrEmploymentShape, p, itoa(len(b.Recipes)), "role recipes")
		return
	}
	if len(b.Recipes) == 0 {
		v.rep.add(ErrEmploymentShape, p, "0", "1")
		return
	}
	seen := make(map[string]bool, len(b.Recipes))
	for _, id := range b.Recipes {
		switch {
		case !v.recipe[id]:
			v.rep.add(ErrUnknownRecipe, p, id, "")
		case seen[id]:
			v.rep.add(ErrDuplicateID, p, id, "")
		case !slices.Contains(r.Recipes, id):
			v.rep.add(ErrEmploymentShape, p, id, "role recipes")
		}
		seen[id] = true
	}
}

// employerWeight reports a group of the role whose agents weigh more than one
// household (spec-employment.md 3.6, 14.1).
func (v *validator) employerWeight(r *Role) {
	c := &v.c
	for _, g := range c.Groups {
		if g.Role != r.ID {
			continue
		}
		_, w, _, ok := groupShape(c.World, g)
		if scaleOf(c.World, g) > 1 || (ok && w > 1) {
			v.rep.add(ErrEmployerWeight, []string{"groups", g.Region, g.Role, "scale"}, i64(scaleOf(c.World, g)), "1")
		}
	}
}

// employmentWarnings are the warnings of spec-employment.md 14.2, raised in the
// hire form only (the payroll form has errors where two of them are warnings,
// 24.4.2).
func employmentWarnings(c *Config, lk lookup) []Warning {
	if m := c.Modules.Employment; m == nil || !m.Hire {
		return nil
	}
	var out []Warning
	add := func(rule string, segs ...string) { out = append(out, Warning{Path: segs, Rule: rule}) }
	employers, workers := splitHireRoles(c)
	if len(employers) == 0 {
		add("employment_no_employer", "modules", "employment")
	}
	if len(workers) == 0 {
		add("employment_no_worker", "modules", "employment")
	}
	for _, r := range workers {
		if r.InKind > 0 {
			add("inkind_wage_ignored", "roles", r.ID, "in_kind")
		}
	}
	out = append(out, noEmployerInRegion(c, employers, workers)...)
	for _, r := range employers {
		out = append(out, employerWarnings(c, lk, r, workers)...)
	}
	return out
}

// splitHireRoles returns the roles with an employer block and the other Hireable
// roles, each in role order.
func splitHireRoles(c *Config) (employers, workers []*Role) {
	for i := range c.Roles {
		if r := &c.Roles[i]; r.Employer != nil {
			employers = append(employers, r)
		} else if r.Hireable {
			workers = append(workers, r)
		}
	}
	return employers, workers
}

// noEmployerInRegion is no_employer_in_region: a region with a group of a Hireable
// role and no group of an employer role. It is raised once per such group.
func noEmployerInRegion(c *Config, employers, workers []*Role) []Warning {
	if len(employers) == 0 {
		return nil // employment_no_employer says it, once
	}
	has := make(map[[2]string]bool, len(c.Groups))
	for _, g := range c.Groups {
		has[[2]string{g.Region, g.Role}] = true
	}
	var out []Warning
	for _, g := range c.Groups {
		if !slices.ContainsFunc(workers, func(r *Role) bool { return r.ID == g.Role }) {
			continue
		}
		if !slices.ContainsFunc(employers, func(r *Role) bool { return has[[2]string{g.Region, r.ID}] }) {
			out = append(out, Warning{Path: []string{"groups", g.Region, g.Role}, Rule: "no_employer_in_region"})
		}
	}
	return out
}

// employerWarnings are the warnings of one employer role: worker_share_full,
// employer_no_outlet and, for a role with a group, the three that read the
// reference prices (hire_never_profitable, wage_below_upkeep and
// payroll_exceeds_money).
func employerWarnings(c *Config, lk lookup, r *Role, workers []*Role) []Warning {
	var out []Warning
	add := func(rule string, segs ...string) { out = append(out, Warning{Path: segs, Rule: rule}) }
	eb := []string{"roles", r.ID, "employer"}
	if r.Employer.WorkerShareBP == payroll.WholeBP {
		add("worker_share_full", append(eb, "worker_share_bp")...)
	}
	if !hasOutlet(c, r) {
		add("employer_no_outlet", append(eb, "recipes")...)
	}
	return append(out, employerPriceWarnings(c, lk, r, workers)...)
}

// hasOutlet says the employer role has a taste row with a scale above 0 or that a
// money flow targets its households: the profit of its hires has somewhere to go
// (spec-employment.md 11.3.1).
func hasOutlet(c *Config, r *Role) bool {
	if d := c.Modules.Demand; d != nil {
		if slices.ContainsFunc(d.Tastes, func(t Taste) bool { return t.Role == r.ID && t.Scale > 0 }) {
			return true
		}
	}
	if m := c.Modules.MoneySupply; m != nil {
		return slices.ContainsFunc(m.Flows, func(f MoneyFlow) bool {
			return f.Target.Kind == TargetHouseholds && (f.Target.Role == "" || f.Target.Role == r.ID)
		})
	}
	return false
}

// employerPriceWarnings are the three warnings that value the hire list at the
// reference prices (spec-employment.md 14.2). The prices are those of the dearest
// region where the role has a group, so a warning means no region of the role
// would do better; a role with no group (a role for external agents) draws none.
// A value that does not fit int64 is unknown, not low: it draws no
// hire_never_profitable for its recipe and no wage warning for the role.
func employerPriceWarnings(c *Config, lk lookup, r *Role, workers []*Role) []Warning {
	regions := groupRegions(c, r.ID)
	if len(regions) == 0 {
		return nil
	}
	var out []Warning
	eb := []string{"roles", r.ID, "employer"}
	best, longest := int64(0), int64(1)
	unknown := false
	for _, id := range r.Employer.Recipes {
		rc := lk.recipe[id]
		if rc == nil || rc.Duration < 1 {
			continue
		}
		net, complete := bestNetPerTick(lk, regions, rc, r.Employer.HireYieldBP)
		if net < 1 && complete { // an incomplete value is a lower bound: no claim about this recipe
			out = append(out, Warning{Path: append(slices.Clone(eb), "recipes", id), Rule: "hire_never_profitable"})
		}
		unknown = unknown || !complete
		best, longest = max(best, net), max(longest, rc.Duration)
	}
	if unknown {
		return out // best is only a lower bound, so the wage warnings would be guesses
	}
	return append(out, wageWarnings(c, lk, r, workers, regions, best, longest)...)
}

// wageWarnings are wage_below_upkeep and payroll_exceeds_money, from the best net
// value per tick and the longest job of the hire list.
func wageWarnings(c *Config, lk lookup, r *Role, workers []*Role, regions []string, best, longest int64) []Warning {
	var out []Warning
	eb := []string{"roles", r.ID, "employer"}
	wage, ok := payroll.Wage(0, best, r.Employer.WorkerShareBP)
	if !ok || best < 1 {
		return out
	}
	if slices.ContainsFunc(workers, func(w *Role) bool { return wage < upkeepValue(c, lk, w, regions) }) {
		out = append(out, Warning{Path: append(slices.Clone(eb), "worker_share_bp"), Rule: "wage_below_upkeep"})
	}
	if bill, ok := payroll.Bill(wage, longest, r.Employer.MaxHires); ok && int64(r.StartMoney) < bill {
		out = append(out, Warning{Path: []string{"roles", r.ID, "start_money"}, Rule: "payroll_exceeds_money"})
	}
	return out
}

// groupRegions are the regions where the role has a group, in group order.
func groupRegions(c *Config, role string) []string {
	var out []string
	for _, g := range c.Groups {
		if g.Role == role && !slices.Contains(out, g.Region) {
			out = append(out, g.Region)
		}
	}
	return out
}

// bestNetPerTick is the most, over the regions, that one tick of a hired execution
// of the recipe is worth at the reference prices: the non-discard outputs at the
// hired yield less the inputs, over the Duration, rounded down (the Bid of
// spec-employment.md 4.3.3 at a yield table entry of 1). complete is false when a
// product did not fit int64 in some region: the value is then a lower bound, and a
// caller must not read a low value as "never profitable".
func bestNetPerTick(lk lookup, regions []string, rc *Recipe, yieldBP int64) (best int64, complete bool) {
	complete = true
	for _, g := range regions {
		out, ok := outputValue(lk, g, rc, yieldBP)
		in, ok2 := itemsValue(lk, g, rc.Inputs)
		if !ok || !ok2 {
			complete = false
			continue
		}
		best = max(best, (out-in)/rc.Duration)
	}
	return best, complete
}

// outputValue is the value of the non-discard outputs of one execution at the
// hired yield in region g, rounded down.
func outputValue(lk lookup, g string, rc *Recipe, yieldBP int64) (int64, bool) {
	var sum int64
	for _, o := range rc.Outputs {
		cm := lk.commodity[o.Commodity]
		if cm == nil || o.Discard {
			continue
		}
		v, ok := checked.Mul(int64(o.Quantity), int64(lk.price(g, cm)))
		if !ok {
			return 0, false
		}
		if sum, ok = checked.Add(sum, v); !ok {
			return 0, false
		}
	}
	return checked.MulDivFloor(sum, yieldBP, payroll.WholeBP)
}

// itemsValue is the value of a list of items at the reference prices of region g.
func itemsValue(lk lookup, g string, items []Item) (int64, bool) {
	var sum int64
	for _, it := range items {
		cm := lk.commodity[it.Commodity]
		if cm == nil {
			continue
		}
		v, ok := checked.Mul(int64(it.Quantity), int64(lk.price(g, cm)))
		if !ok {
			return 0, false
		}
		if sum, ok = checked.Add(sum, v); !ok {
			return 0, false
		}
	}
	return sum, true
}

// upkeepValue is one tick of the upkeep of a household of the role (its own list,
// else the world's) at the reference prices of the dearest of the regions; 0 where
// a product does not fit, so that no warning is raised on arithmetic that cannot
// be done.
func upkeepValue(c *Config, lk lookup, r *Role, regions []string) int64 {
	items := c.World.Upkeep
	if len(r.Upkeep) > 0 {
		items = r.Upkeep
	}
	best := int64(0)
	for _, g := range regions {
		if v, ok := itemsValue(lk, g, items); ok {
			best = max(best, v)
		}
	}
	return best
}
