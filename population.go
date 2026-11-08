package economysim

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/endowment"
	"github.com/Whealinghold/economy-sim/internal/population"
)

// This file is the population module (spec-population.md): births and deaths as
// the formation and dissolution of households, driven by the household's own
// welfare ratio. The pure rules are in internal/population. This part of the
// file is the configuration surface: the canonical form, the validator and the
// warnings. The step, the executor hooks, the report, the checkpoint section and
// the scale slot follow in the later phases of plan 2-11.

// canonical returns a copy of the module with the schedules cloned in knot order
// and the region rows sorted by region (spec-population.md 8, spec-config.md
// 3.5). A nil module stays nil; a module that is on stays on.
func (p *Population) canonical() *Population {
	if p == nil {
		return nil
	}
	return &Population{
		BirthSchedule: slices.Clone(p.BirthSchedule),
		DeathSchedule: slices.Clone(p.DeathSchedule),
		MinHouseholds: p.MinHouseholds,
		BirthStake:    p.BirthStake,
		Regions:       sortedBy(p.Regions, func(a, b RegionPopulation) int { return cmp.Compare(a.Region, b.Region) }),
	}
}

// popKnots converts a schedule for internal/population, inline at the call
// (architecture.md 2.9).
func popKnots(ks []Knot) []population.Knot {
	out := make([]population.Knot, len(ks))
	for i, k := range ks {
		out[i] = population.Knot{Ratio: k.Ratio, Rate: k.Rate}
	}
	return out
}

// validatePopulation is the module's part of phase 2 and 3 of spec-config.md
// 7.1.6 (spec-population.md 9.1): the bounds of each field, the schedules and the
// region rows. Every error is ErrOutOfBounds, ErrUnknownRegion or ErrDuplicateID
// with the path of the field; the module adds no sentinel.
func (v *validator) validatePopulation() {
	m := v.c.Modules.Population
	if m == nil {
		return
	}
	base := []string{"modules", "population"}
	v.popSchedule(path(base, "birth_schedule"), m.BirthSchedule)
	v.popSchedule(path(base, "death_schedule"), m.DeathSchedule)
	v.inRange(path(base, "min_households"), m.MinHouseholds, 0, MaxHouseholds)
	v.inRange(path(base, "birth_stake"), m.BirthStake, 0, wholeBP)
	if len(m.Regions) > MaxRegions {
		v.rep.add(ErrOutOfBounds, path(base, "regions"), itoa(len(m.Regions)), itoa(MaxRegions))
		return // the rest is linear in the count it just refused
	}
	seen := make(map[string]bool, len(m.Regions))
	for i := range m.Regions {
		r := &m.Regions[i]
		rb := path(base, "region", r.Region)
		if v.lk.region[r.Region] == nil {
			v.rep.add(ErrUnknownRegion, path(rb, "region"), r.Region, "")
		}
		if seen[r.Region] {
			v.rep.add(ErrDuplicateID, path(rb, "region"), r.Region, "")
		}
		seen[r.Region] = true
		v.inRange(path(rb, "birth_scale"), r.BirthScale, 0, MaxBasisPoints)
		v.inRange(path(rb, "death_scale"), r.DeathScale, 0, MaxBasisPoints)
		v.inRange(path(rb, "household_cap"), r.HouseholdCap, 0, MaxHouseholds)
	}
}

// popSchedule checks one schedule: 1 to MaxKnots knots, each ratio and rate in
// its range, and the ratios strictly increasing (spec-population.md 9.1,
// FR-POP-026). A count out of its bound stops the per-knot rules.
func (v *validator) popSchedule(p []string, ks []Knot) {
	if len(ks) < 1 || len(ks) > MaxKnots {
		v.rep.add(ErrOutOfBounds, p, itoa(len(ks)), itoa(MaxKnots))
		return
	}
	for i, k := range ks {
		kp := path(p, itoa(i))
		v.inRange(path(kp, "ratio"), k.Ratio, 0, MaxBasisPoints)
		v.inRange(path(kp, "rate"), k.Rate, 0, population.MaxRate)
		if i > 0 && k.Ratio <= ks[i-1].Ratio {
			v.rep.add(ErrOutOfBounds, path(kp, "ratio"), i64(k.Ratio), i64(ks[i-1].Ratio))
		}
	}
}

// scheduleOK reports whether a schedule is one the rules can read: 1 to MaxKnots
// knots, ratios strictly increasing and in range, rates in range. The warnings
// run on a configuration that may not validate and read only such schedules.
func scheduleOK(ks []Knot) bool {
	if len(ks) < 1 || len(ks) > MaxKnots {
		return false
	}
	for i, k := range ks {
		if k.Ratio < 0 || k.Ratio > MaxBasisPoints || k.Rate < 0 || k.Rate > population.MaxRate ||
			(i > 0 && k.Ratio <= ks[i-1].Ratio) {
			return false
		}
	}
	return true
}

// replacementOf returns the replacement ratio r* of the module and the slope of
// birth - death on the segment that holds it: dR the rise in rate and dX the run
// in ratio (spec-population.md 5.4, 5.7). lo is the lower end of the segment. ok
// is false when a schedule is unreadable or the schedules have no replacement
// ratio.
func (p *Population) replacementOf() (rstar, lo, dR, dX int64, ok bool) {
	if !scheduleOK(p.BirthSchedule) || !scheduleOK(p.DeathSchedule) {
		return 0, 0, 0, 0, false
	}
	b, d := popKnots(p.BirthSchedule), popKnots(p.DeathSchedule)
	rstar, lo, hi, ok := population.Replacement(b, d)
	if !ok {
		return 0, 0, 0, 0, false
	}
	g := func(x int64) int64 {
		br, _, _ := population.Rate(b, x, wholeBP)
		dr, _, _ := population.Rate(d, x, wholeBP)
		return br - dr
	}
	return rstar, lo, g(hi) - g(lo), hi - lo, true
}

// populationWarnings returns the warnings of spec-population.md 9.2 and the two
// diagnostics (5.7, 4.6.1), in no particular order. They read a configuration
// that may not validate, so each guards what it reads. None fails Validate.
func populationWarnings(c *Config) []Warning {
	m := c.Modules.Population
	if m == nil {
		return nil
	}
	var out []Warning
	add := func(rule string, segs ...string) { out = append(out, Warning{Path: segs, Rule: rule}) }
	base := []string{"modules", "population"}
	popRenewalWarnings(c, add)
	if !hasUpkeep(c) {
		add("population_without_upkeep", base...)
	}
	if !hasDraws(c) && !hasCap(m) {
		add("no_fixed_factor", base...)
	}
	if scheduleOK(m.BirthSchedule) && scheduleOK(m.DeathSchedule) {
		popScheduleWarnings(c, m, base, add)
	}
	return out
}

// popRenewalWarnings raises renewal_with_population for a non-default Renewal
// above 0 and the informational renewal_ignored at the default (spec-population.md
// 4.6.1, 9.2). At 0 neither fires.
func popRenewalWarnings(c *Config, add func(string, ...string)) {
	switch r := c.World.Renewal; {
	case r == DefaultWorldParams().Renewal:
		add("renewal_ignored", "world", "renewal")
	case r > 0:
		add("renewal_with_population", "world", "renewal")
	}
}

// popScheduleWarnings raises the warnings that read the two schedules:
// schedule_direction, no_replacement_ratio, replacement_ratio,
// replacement_above_closed_ceiling, population_gain and the two plan warnings of
// the resources module.
func popScheduleWarnings(c *Config, m *Population, base []string, add func(string, ...string)) {
	if fallsAnywhere(m.BirthSchedule) {
		add("schedule_direction", append(slices.Clone(base), "birth_schedule")...)
	}
	if risesAnywhere(m.DeathSchedule) {
		add("schedule_direction", append(slices.Clone(base), "death_schedule")...)
	}
	rstar, lo, dR, dX, ok := m.replacementOf()
	if !ok {
		add("no_replacement_ratio", base...)
		return
	}
	// The path ends in the lower knot of the birth schedule at or below the
	// segment, the index of the segment the figure is read at (D434).
	idx := max(0, sortedSearchBelow(m.BirthSchedule, lo))
	add("replacement_ratio", append(slices.Clone(base), "birth_schedule", itoa(idx))...)
	if rstar > wholeBP && c.Modules.Demand == nil {
		add("replacement_above_closed_ceiling", base...)
	}
	if population.GainWarn(rstar, dR, dX, longestWindow(c), c.Calendar.TicksPerYear()) {
		add("population_gain", base...)
	}
	popPlanWarnings(c, rstar, dR, dX, add)
}

// sortedSearchBelow returns the index of the last knot with a ratio at or below
// x, -1 when there is none.
func sortedSearchBelow(ks []Knot, x int64) int {
	i, _ := slices.BinarySearchFunc(ks, x, func(k Knot, t int64) int { return cmp.Compare(k.Ratio, t+1) })
	return i - 1
}

// fallsAnywhere reports whether a schedule's rate falls from one knot to the next.
func fallsAnywhere(ks []Knot) bool {
	for i := 1; i < len(ks); i++ {
		if ks[i].Rate < ks[i-1].Rate {
			return true
		}
	}
	return false
}

// risesAnywhere reports whether a schedule's rate rises from one knot to the next.
func risesAnywhere(ks []Knot) bool {
	for i := 1; i < len(ks); i++ {
		if ks[i].Rate > ks[i-1].Rate {
			return true
		}
	}
	return false
}

// hasUpkeep reports whether any level of the configuration names an upkeep, the
// no_upkeep case of spec-agents.md 13.
func hasUpkeep(c *Config) bool {
	if len(c.World.Upkeep) > 0 {
		return true
	}
	for i := range c.Roles {
		if len(c.Roles[i].Upkeep) > 0 {
			return true
		}
	}
	for i := range c.Regions {
		if len(c.Regions[i].ExtraUpkeep) > 0 {
			return true
		}
	}
	return false
}

// hasDraws reports whether any recipe draws on an endowment (the first-level test
// of no_fixed_factor, spec-population.md 9.2 and open question 9).
func hasDraws(c *Config) bool {
	for i := range c.Recipes {
		if len(c.Recipes[i].Draws) > 0 {
			return true
		}
	}
	return false
}

// hasCap reports whether any region row sets a household cap.
func hasCap(m *Population) bool {
	for i := range m.Regions {
		if m.Regions[i].HouseholdCap > 0 {
			return true
		}
	}
	return false
}

// longestWindow is the longest effective welfare window in ticks: the world's, or
// a role's WindowCycles times its longest job (spec-welfare-and-wages.md 2.3.3).
func longestWindow(c *Config) int64 {
	t := c.World.WelfareWindow
	lk := newLookup(c)
	for i := range c.Roles {
		if p, ok := checked.Mul(c.World.WindowCycles, longestDurationAny(c, lk, &c.Roles[i])); ok {
			t = max(t, p)
		}
	}
	return t
}

// popPlanWarnings fills endowment.Plan of each resource from the schedules and
// raises resource_overshoot and resource_unstable_plan when the resources module
// is on (spec-resources.md 8.3.1, FR-RES-044; spec-population.md 5.2). The
// regeneration scale is taken at its default 10 000, since a row's scale is per
// region and the warning is per resource.
func popPlanWarnings(c *Config, rstar, dR, dX int64, add func(string, ...string)) {
	m := c.Modules.Resources
	if m == nil {
		return
	}
	for i := range m.Resources {
		r := &m.Resources[i]
		plan, ok := resourcePlan(c, r)
		if !ok {
			continue
		}
		plan.RStar, plan.DR, plan.DX = rstar, dR, dX
		rb := []string{"modules", "resources", "resources", r.ID}
		if endowment.Overshoot(plan) {
			add("resource_overshoot", rb...)
		}
		if endowment.UnstablePlan(plan) {
			add("resource_unstable_plan", rb...)
		}
	}
}

// resourcePlan returns the part of endowment.Plan that a resource gives: its
// shape, the effective annual rate in basis points (the rate times the mean
// season factor), the plan share and the elasticity of the abundance curve there.
// ok is false for a share or rate outside its bound or an overflow.
func resourcePlan(c *Config, r *Resource) (p endowment.Plan, ok bool) {
	if r.PlanShareBP < 1 || r.PlanShareBP > wholeBP || r.RateBP < 0 || r.RateBP > MaxRegenBP {
		return p, false
	}
	eps, ok := endowment.Elasticity(sortedKnots(r.Abundance), r.PlanShareBP)
	if !ok {
		return p, false
	}
	p = endowment.Plan{ShareBP: r.PlanShareBP, ElasticityBP: eps}
	switch r.Shape {
	case ShapeRefill:
		p.Shape = endowment.Refill
	case ShapeLogistic:
		p.Shape = endowment.Logistic
	default:
		p.Shape = endowment.Constant
	}
	p.RateBPEff = r.RateBP * meanSeasonBP(c, r) / wholeBP
	return p, true
}

// meanSeasonBP is the mean regeneration factor of a resource over the calendar's
// seasons in basis points, truncated: 10 000 when it has no profile or the
// calendar no season, 0 for a season the profile does not list.
func meanSeasonBP(c *Config, r *Resource) int64 {
	if len(r.Seasons) == 0 || len(c.Calendar.Seasons) == 0 {
		return wholeBP
	}
	var sum int64
	for _, s := range c.Calendar.Seasons {
		for _, sr := range r.Seasons {
			if sr.Season == s.ID {
				sum += min(max(sr.BP, 0), MaxSeasonRegenBP)
				break
			}
		}
	}
	return sum / int64(len(c.Calendar.Seasons))
}
