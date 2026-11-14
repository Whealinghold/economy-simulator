package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/curve"
	"github.com/Whealinghold/economy-sim/internal/endowment"
	"github.com/Whealinghold/economy-sim/internal/graph"
)

// This file is the validation of the technology module (spec-technology.md 10; the
// continuation file of technology.go, architecture.md 2.6): the rules of 10.1 that
// Validate checks, the overflow term of FR-TEC-020 and the warnings of 10.2 to 10.4.

// techBase is the path of the module in a configuration.
var techBase = []string{"modules", "technology"}

// techPath is the path of field i of the module, or of one key of it.
func techPath(i int, key ...string) []string {
	return path(techBase, append([]string{"fields[" + itoa(i) + "]"}, key...)...)
}

// validateTechnology checks the technology module (spec-technology.md 10.1,
// FR-TEC-002, FR-TEC-003). It reports nothing when the module is off. The rules run
// in phases and a phase that finds an error ends the run, so the first error is the
// same for the same mistake: the shape, then the IDs and counts, the bounds, the
// recipe references and last the overlap, which reads the rest.
func (v *validator) validateTechnology() {
	m := v.c.Modules.Technology
	if m == nil {
		return
	}
	steps := []func(*Technology) bool{
		v.techShape, v.techIDs, v.techBounds, v.techRecipes, v.techOverlap,
	}
	for _, step := range steps {
		if !step(m) {
			return
		}
	}
}

// techShape is ErrTechShape: no recipe, no cap, a value that an off mechanism does
// not use that is not 0, or one that an on mechanism needs that is 0 (3.1.2).
func (v *validator) techShape(m *Technology) bool {
	n := len(v.rep.errs)
	for i, f := range m.Fields {
		if len(f.Recipes) == 0 {
			v.rep.add(ErrTechShape, techPath(i, "recipes"), "0", "1")
		}
		if f.CapBP == 0 {
			v.rep.add(ErrTechShape, techPath(i, "cap_bp"), "0", "1")
		}
		v.techPair(i, "base", f.Base, f.DoublingBP)
		v.techPair(i, "step_bp", f.StepBP, f.DiscoveryBP)
		v.techPair(i, "reach", f.Reach, f.CatchUpBP)
	}
	return len(v.rep.errs) == n
}

// techPair is the rule of one pair of 3.1.2: the value must be set exactly when its
// mechanism's rate is above 0.
func (v *validator) techPair(i int, key string, value, rate int64) {
	switch {
	case rate > 0 && value == 0:
		v.rep.add(ErrTechShape, techPath(i, key), "0", "1")
	case rate <= 0 && value != 0:
		v.rep.add(ErrTechShape, techPath(i, key), i64(value), "0")
	}
}

// techIDs checks the count of fields, each ID and its duplicates, and the count and
// duplicates of each recipe list.
func (v *validator) techIDs(m *Technology) bool {
	if len(m.Fields) > MaxTechFields {
		v.rep.add(ErrTooMany, path(techBase, "fields"), itoa(len(m.Fields)), itoa(MaxTechFields))
		return false
	}
	n := len(v.rep.errs)
	seen := make(map[string]bool, len(m.Fields)) // a membership test only
	for i, f := range m.Fields {
		switch {
		case !validID(f.ID):
			v.rep.add(ErrInvalidID, techPath(i, "id"), f.ID, "")
		case seen[f.ID]:
			v.rep.add(ErrDuplicateID, techPath(i, "id"), f.ID, "")
		}
		seen[f.ID] = true
		v.techRecipeCount(i, f)
	}
	return len(v.rep.errs) == n
}

// techRecipeCount checks the length of one field's recipe list and a recipe named
// twice in it.
func (v *validator) techRecipeCount(i int, f TechField) {
	if len(f.Recipes) > MaxFieldRecipes {
		v.rep.add(ErrTooMany, techPath(i, "recipes"), itoa(len(f.Recipes)), itoa(MaxFieldRecipes))
		return
	}
	seen := make(map[string]bool, len(f.Recipes)) // a membership test only
	for j, id := range f.Recipes {
		if seen[id] {
			v.rep.add(ErrDuplicateID, techPath(i, "recipes["+itoa(j)+"]"), id, "")
		}
		seen[id] = true
	}
}

// techBounds checks every number of a field against its bound (9.2) and its label.
func (v *validator) techBounds(m *Technology) bool {
	n := len(v.rep.errs)
	for i, f := range m.Fields {
		for _, r := range []fieldRange{
			{"cap_bp", f.CapBP, 1, MaxGainBP}, {"base", f.Base, 0, MaxTechBase},
			{"doubling_bp", f.DoublingBP, 0, MaxDoublingBP}, {"trend_bp", f.TrendBP, 0, MaxTechTrendBP},
			{"discovery_bp", f.DiscoveryBP, 0, MaxDiscoveryBP}, {"step_bp", f.StepBP, 0, MaxStepBP},
			{"catch_up_bp", f.CatchUpBP, 0, MaxCatchUpBP}, {"reach", f.Reach, 0, MaxDistance},
		} {
			if r.value < r.lo || r.value > r.hi {
				v.rep.add(ErrOutOfBounds, techPath(i, r.name), i64(r.value), i64(r.hi))
			}
		}
		v.label(techPath(i), "label", f.Label)
	}
	return len(v.rep.errs) == n
}

// techRecipes is ErrUnknownRecipe: a recipe of a field that is not configured.
func (v *validator) techRecipes(m *Technology) bool {
	n := len(v.rep.errs)
	for i, f := range m.Fields {
		for j, id := range f.Recipes {
			if !v.recipe[id] {
				v.rep.add(ErrUnknownRecipe, techPath(i, "recipes["+itoa(j)+"]"), id, "")
			}
		}
	}
	return len(v.rep.errs) == n
}

// techOverlap is ErrTechRecipeOverlap: a recipe in two fields (3.1.3). The error is
// at the second use, and its bound names the field that has it first.
func (v *validator) techOverlap(m *Technology) bool {
	n := len(v.rep.errs)
	first := make(map[string]int, len(m.Fields)) // the field of each recipe
	for i, f := range m.Fields {
		for j, id := range f.Recipes {
			if k, taken := first[id]; taken && k != i {
				v.rep.add(ErrTechRecipeOverlap, techPath(i, "recipes["+itoa(j)+"]"), id, m.Fields[k].ID)
				continue
			}
			first[id] = i
		}
	}
	return len(v.rep.errs) == n
}

// withTech multiplies the largest yield factor of a recipe that is in a field by the
// largest technology factor, MaxTechBP over 10 000 (spec-technology.md 9.5,
// FR-TEC-020); a recipe in no field, or a world with the module off, is as it was.
func (v *validator) withTech(recipe string, yieldMax int64) int64 {
	m := v.c.Modules.Technology
	if m == nil {
		return yieldMax
	}
	in := slices.ContainsFunc(m.Fields, func(f TechField) bool { return slices.Contains(f.Recipes, recipe) })
	if !in {
		return yieldMax
	}
	// rounded up: the result is a bound. The caller's yieldMax is truncated, so the bound
	// can sit below the exact maximum by under MaxTechBP/10 000 units, far inside the
	// slack of the int64 checks that read it
	return (yieldMax*MaxTechBP + wholeBP - 1) / wholeBP // at most 10^6 * 1.1 * 10^5 / 10^4
}

// technologyWarnings are the warnings of spec-technology.md 10.2 to 10.4: field_inert,
// field_unused, diffusion_no_contact, technology_past_maximum_yield and
// trend_outpaces_schedule. A warning never fails validation and is safe on a
// configuration that does not validate: an undeclared ID is skipped.
func technologyWarnings(c *Config, lk lookup) []Warning {
	m := c.Modules.Technology
	if m == nil {
		return nil
	}
	var out []Warning
	used := techUsedRecipes(c)
	contact := techContact(c, m)
	for i := range m.Fields {
		for _, rule := range techFieldRules(c, lk, &m.Fields[i], used, contact) {
			out = append(out, Warning{Path: techPath(i), Rule: rule})
		}
	}
	return out
}

// inert reports whether every mechanism of the field is off (3.1.2).
func (f *TechField) inert() bool {
	return f.DoublingBP <= 0 && f.TrendBP <= 0 && f.DiscoveryBP <= 0 && f.CatchUpBP <= 0
}

// noContact reports whether diffusion is on and the reach has no pair of regions in
// contact.
func (f *TechField) noContact(contact func(int64) bool) bool {
	return f.CatchUpBP > 0 && f.Reach >= 1 && !contact(f.Reach)
}

// techFieldRules are the rules of 10.2 to 10.4 that fire for one field.
func techFieldRules(c *Config, lk lookup, f *TechField, used map[string]bool, contact func(int64) bool) []string {
	var out []string
	if f.inert() {
		out = append(out, "field_inert")
	}
	if !slices.ContainsFunc(f.Recipes, func(id string) bool { return used[id] }) {
		out = append(out, "field_unused")
	}
	if f.noContact(contact) {
		out = append(out, "diffusion_no_contact")
	}
	if techPastMaximumYield(c, lk, f) {
		out = append(out, "technology_past_maximum_yield")
	}
	if techTrendOutpaces(c, f) {
		out = append(out, "trend_outpaces_schedule")
	}
	return out
}

// techUsedRecipes is the set of recipes that some role may run: its list and its
// effective own-use list (spec-technology.md 10.2, field_unused).
func techUsedRecipes(c *Config) map[string]bool {
	used := make(map[string]bool)
	for i := range c.Roles {
		for _, id := range jobIDs(c, &c.Roles[i]) {
			used[id] = true
		}
	}
	return used
}

// techContact returns a function that reports whether some ordered pair of
// ordinary regions has a finite distance below a reach over the edges as
// configured (spec-technology.md 10.2, diffusion_no_contact). It builds the
// shortest paths once, from the geometry of regions.go. When the graph cannot be
// built it reports contact, so that no warning fires on a fact it did not read.
func techContact(c *Config, m *Technology) func(reach int64) bool {
	if !slices.ContainsFunc(m.Fields, func(f TechField) bool { return f.CatchUpBP > 0 }) {
		return func(int64) bool { return true }
	}
	g, ok := readGeometry(c)
	if !ok {
		return func(int64) bool { return true }
	}
	tables, err := graph.Build(len(g.ids), g.edges)
	if err != nil {
		return func(int64) bool { return true }
	}
	ordinary := ordinaryRegions(c, g.ids)
	return func(reach int64) bool {
		for _, a := range ordinary {
			for _, b := range ordinary {
				if d, ok := tables.Dist(a, b); a != b && ok && d >= 1 && d < reach {
					return true
				}
			}
		}
		return false
	}
}

// ordinaryRegions are the indexes, in ids, of the regions that are not outside
// nodes.
func ordinaryRegions(c *Config, ids []string) []int {
	outside := outsideNodeSet(c)
	var out []int
	for i, id := range ids {
		if !outside[id] {
			out = append(out, i)
		}
	}
	return out
}

// techPastMaximumYield is technology_past_maximum_yield (10.3, FR-TEC-030): the
// population module is on, a recipe of the field draws on a logistic resource whose
// abundance curve has a positive slope at the plan share, and a(5 000) *
// (10 000 + CapBP) >= a(sig) * 10 000, with the product held in 128 bits.
func techPastMaximumYield(c *Config, lk lookup, f *TechField) bool {
	if c.Modules.Population == nil || c.Modules.Resources == nil {
		return false
	}
	for _, id := range f.Recipes {
		rc := lk.recipe[id]
		if rc == nil {
			continue
		}
		for _, d := range rc.Draws {
			if r := lk.resource[d.Resource]; r != nil && logisticPastYield(c, r, f.CapBP) {
				return true
			}
		}
	}
	return false
}

// logisticPastYield is the inequality of 10.3.1 for one resource and a cap.
func logisticPastYield(c *Config, r *Resource, capBP int64) bool {
	p, ok := resourcePlan(c, r)
	if !ok || p.Shape != endowment.Logistic || p.ElasticityBP <= 0 || capBP < 0 {
		return false
	}
	knots := sortedKnots(r.Abundance)
	half, ok1 := abundanceAt(knots, wholeBP/2)
	plan, ok2 := abundanceAt(knots, r.PlanShareBP)
	if !ok1 || !ok2 {
		return false
	}
	got, ok := checked.Cmp3(half, wholeBP+capBP, 1, plan, wholeBP, 1)
	return ok && got >= 0
}

// abundanceAt reads the abundance curve at a share in basis points: 10 000 when
// there are no knots (spec-resources.md 4.7).
func abundanceAt(knots []curve.Knot, share int64) (int64, bool) {
	if len(knots) == 0 {
		return wholeBP, true
	}
	return curve.Eval(knots, share)
}

// techTrendOutpaces is trend_outpaces_schedule (10.4, FR-TEC-031): the population
// module is on and TrendBP * dX > (knot_max - r*) * dR, in integers with no
// division, the product held in 128 bits.
func techTrendOutpaces(c *Config, f *TechField) bool {
	p := c.Modules.Population
	if p == nil || f.TrendBP <= 0 {
		return false
	}
	rstar, _, dR, dX, ok := p.replacementOf()
	if !ok {
		return false
	}
	var knotMax int64
	for _, k := range p.BirthSchedule {
		knotMax = max(knotMax, k.Ratio)
	}
	got, ok := checked.Cmp3(f.TrendBP, dX, 1, max(knotMax-rstar, 0), dR, 1)
	return ok && got > 0
}
