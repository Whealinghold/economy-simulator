package economysim

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/curve"
)

// This file is the validator and the warnings of the resources module
// (spec-resources.md 13): the bounds of each field, the identity and references
// of the rows and of the draws of the recipes, the abundance curves, and the
// advice of 13.3 that does not need the solver.

// validateResources is the module's part of phases 2, 3 and 5 of spec-config.md
// 7.1.6 (spec-resources.md 13). The draws of the recipes are checked whether or
// not the module is on, because a draw with the module off is
// ErrModuleDependency (FR-RES-001). The module's own rules report nothing when
// it is off.
func (v *validator) validateResources() {
	v.validateDraws()
	m := v.c.Modules.Resources
	if m == nil {
		return
	}
	base := []string{"modules", "resources"}
	over := false
	if len(m.Resources) > MaxResources {
		v.rep.add(ErrOutOfBounds, path(base, "resources"), itoa(len(m.Resources)), itoa(MaxResources))
		over = true
	}
	if len(m.Endowments) > MaxEndowments {
		v.rep.add(ErrOutOfBounds, path(base, "endowments"), itoa(len(m.Endowments)), itoa(MaxEndowments))
		over = true
	}
	v.inRange(path(base, "horizon_years"), m.HorizonYears, 1, MaxHorizonYears)
	if over {
		return // the rest is linear in the counts it just refused
	}
	seen := make(map[string]bool, len(m.Resources))
	for i := range m.Resources {
		r := &m.Resources[i]
		rb := path(base, "resources", r.ID)
		if !validID(r.ID) {
			v.rep.add(ErrInvalidID, path(rb, "id"), r.ID, "")
		}
		if seen[r.ID] {
			v.rep.add(ErrDuplicateID, path(rb, "id"), r.ID, "")
		}
		seen[r.ID] = true
		v.resourceFields(rb, r)
		v.resourceCurve(rb, r.Abundance)
		v.resourceSeasons(rb, r.Seasons)
	}
	v.endowmentRows(base, m)
	v.resourceRefuges(base, m)
}

// validateDraws checks the Draws of every recipe (spec-production.md 10.2,
// spec-resources.md 13.2): the module is on, at most MaxDraws, each resource
// defined and listed once, each quantity in its bound. The products with the
// weight are the overflow pass's.
func (v *validator) validateDraws() {
	for i := range v.c.Recipes {
		rc := &v.c.Recipes[i]
		if len(rc.Draws) == 0 {
			continue
		}
		base := []string{"recipes", rc.ID, "draws"}
		if v.c.Modules.Resources == nil {
			v.rep.add(ErrModuleDependency, base, itoa(len(rc.Draws)), "0")
			continue
		}
		if len(rc.Draws) > MaxDraws {
			v.rep.add(ErrOutOfBounds, base, itoa(len(rc.Draws)), itoa(MaxDraws))
			continue
		}
		seen := make(map[string]bool, len(rc.Draws))
		for _, d := range rc.Draws {
			dp := path(base, d.Resource)
			if v.lk.resource[d.Resource] == nil {
				v.rep.add(ErrUnknownResource, path(dp, "resource"), d.Resource, "")
			}
			if seen[d.Resource] {
				v.rep.add(ErrDuplicateID, path(dp, "resource"), d.Resource, "")
			}
			seen[d.Resource] = true
			v.inRange(path(dp, "quantity"), int64(d.Quantity), 1, MaxQuantity)
		}
	}
}

// resourceFields checks the scalar fields of one resource and the rules that tie
// the shape to the rate, the refuge and the capacity (spec-resources.md 12).
func (v *validator) resourceFields(rb []string, r *Resource) {
	shape := r.Shape
	switch shape {
	case ShapeConstant, ShapeRefill, ShapeLogistic:
	default:
		v.rep.add(ErrResourceShape, path(rb, "shape"), shape, "")
	}
	v.ranges(rb, []fieldRange{
		{"rate_per_year", r.RatePerYear, 0, MaxRegenRatePerYear},
		{"rate_bp", r.RateBP, 0, MaxRegenBP},
		{"refuge", r.Refuge, 0, MaxEndowment},
		{"capacity", r.Capacity, 0, MaxEndowment},
		{"initial_bp", r.InitialBP, 0, wholeBP},
		{"plan_share_bp", r.PlanShareBP, 1, wholeBP},
	})
	v.label(rb, "label", r.Label)
	if r.Capacity >= 0 {
		v.shockProduct(path(rb, "capacity"), r.Capacity)
	}
	if shape != ShapeConstant && r.RatePerYear > 0 {
		v.rep.add(ErrResourceShape, path(rb, "rate_per_year"), i64(r.RatePerYear), "0")
	}
	if shape == ShapeConstant && r.RateBP > 0 {
		v.rep.add(ErrResourceShape, path(rb, "rate_bp"), i64(r.RateBP), "0")
	}
	if shape != ShapeLogistic && r.Refuge > 0 {
		v.rep.add(ErrResourceShape, path(rb, "refuge"), i64(r.Refuge), "0")
	}
}

// shockProduct records ErrOverflow when a capacity times the largest shock
// scale over 10 000 is above MaxEndowment (spec-resources.md 13.2).
func (v *validator) shockProduct(p []string, capacity int64) {
	if capacity > MaxEndowment {
		return // its own bound error
	}
	if scaled, ok := checked.MulDiv(capacity, MaxShockBP, wholeBP); !ok || scaled > MaxEndowment {
		v.rep.add(ErrOverflow, p, i64(scaled), i64(MaxEndowment))
	}
}

// resourceCurve checks the abundance curve of a resource: at most MaxKnots
// knots, each share in 0 to 10 000 and above the one before, each factor in 0
// to MaxYieldBP (ErrCurve, spec-resources.md 13.2). A curve that falls, or that
// does not start at 0 or end at 10 000, is legal: the first is the warning
// abundance_not_monotone and the curve is flat outside its end knots (4.7.1).
func (v *validator) resourceCurve(rb []string, knots []AbundanceKnot) {
	if len(knots) > MaxKnots {
		v.rep.add(ErrOutOfBounds, path(rb, "abundance"), itoa(len(knots)), itoa(MaxKnots))
		return
	}
	if bad := curve.Check(curveKnots(knots), wholeBP, MaxYieldBP); bad >= 0 {
		k := knots[bad]
		field, value, limit := "factor", k.Factor, int64(MaxYieldBP)
		if k.Share < 0 || k.Share > wholeBP || bad > 0 && k.Share <= knots[bad-1].Share {
			field, value, limit = "share", k.Share, int64(wholeBP)
		}
		v.rep.add(ErrCurve, path(rb, "abundance["+itoa(bad)+"]", field), i64(value), i64(limit))
	}
}

// curveKnots converts the knots of the configuration to those of internal/curve.
func curveKnots(knots []AbundanceKnot) []curve.Knot {
	out := make([]curve.Knot, len(knots))
	for i, k := range knots {
		out[i] = curve.Knot{X: k.Share, Y: k.Factor}
	}
	return out
}

// resourceSeasons checks the regeneration profile of a resource: each season is
// in the calendar and listed once, each factor in its bound (spec-resources.md
// 4.5, 13.2).
func (v *validator) resourceSeasons(rb []string, seasons []SeasonRegen) {
	known := make(map[string]bool, len(v.c.Calendar.Seasons))
	for _, s := range v.c.Calendar.Seasons {
		known[s.ID] = true
	}
	seen := make(map[string]bool, len(seasons))
	for _, s := range seasons {
		sp := path(rb, "seasons", s.Season)
		if !known[s.Season] {
			v.rep.add(ErrUnknownSeason, path(sp, "season"), s.Season, "")
		}
		if seen[s.Season] {
			v.rep.add(ErrDuplicateID, path(sp, "season"), s.Season, "")
		}
		seen[s.Season] = true
		v.inRange(path(sp, "bp"), s.BP, 0, MaxSeasonRegenBP)
	}
}

// endowmentRows checks the region-keyed table (spec-resources.md 3.3.1, 13.2):
// each key names a region and a resource that exist and appears once, each
// replaced value is in its bound, and a starting stock is within its capacity.
func (v *validator) endowmentRows(base []string, m *Resources) {
	seen := make(map[[2]string]bool, len(m.Endowments))
	for i := range m.Endowments {
		e := &m.Endowments[i]
		eb := path(base, "endowments", e.Region, e.Resource)
		res := v.lk.resource[e.Resource]
		if v.lk.region[e.Region] == nil {
			v.rep.add(ErrUnknownRegion, path(eb, "region"), e.Region, "")
		}
		if res == nil {
			v.rep.add(ErrUnknownResource, path(eb, "resource"), e.Resource, "")
		}
		key := [2]string{e.Region, e.Resource}
		if seen[key] {
			v.rep.add(ErrDuplicateID, path(eb, "resource"), e.Resource, "")
		}
		seen[key] = true
		if e.HasCapacity {
			if v.inRange(path(eb, "capacity"), e.Capacity, 0, MaxEndowment) {
				v.shockProduct(path(eb, "capacity"), e.Capacity)
			}
		}
		if e.HasRegenScale {
			v.inRange(path(eb, "regen_scale_bp"), e.RegenScaleBP, 0, MaxRegenScaleBP)
		}
		if e.HasInitial && v.inRange(path(eb, "initial"), e.Initial, 0, MaxEndowment) {
			if capacity := rowCapacity(e, res); e.Initial > capacity {
				v.rep.add(ErrResourceShape, path(eb, "initial"), i64(e.Initial), i64(capacity))
			}
		}
	}
}

// rowCapacity is the capacity of a row: its own, else the resource's default (0
// when the resource is unknown).
func rowCapacity(e *Endowment, res *Resource) int64 {
	switch {
	case e.HasCapacity:
		return e.Capacity
	case res != nil:
		return res.Capacity
	}
	return 0
}

// resourceRefuges rejects a refuge above the effective capacity of an endowment
// of the resource that has one (spec-resources.md 13.2): the default capacity
// and every row's, where it is above 0. A refuge on a pair with no capacity has
// no stock to be above.
func (v *validator) resourceRefuges(base []string, m *Resources) {
	for i := range m.Resources {
		r := &m.Resources[i]
		if r.Refuge <= 0 || r.Refuge > MaxEndowment {
			continue
		}
		lowest, found := int64(0), false
		note := func(capacity int64) {
			if capacity > 0 && (!found || capacity < lowest) {
				lowest, found = capacity, true
			}
		}
		note(r.Capacity)
		for j := range m.Endowments {
			if e := &m.Endowments[j]; e.Resource == r.ID && e.HasCapacity {
				note(e.Capacity)
			}
		}
		if found && r.Refuge > lowest {
			v.rep.add(ErrResourceShape, path(base, "resources", r.ID, "refuge"), i64(r.Refuge), i64(lowest))
		}
	}
}

// withAbundance multiplies the largest yield factor of a recipe that draws by
// the largest abundance factor, MaxYieldBP over 10 000 (spec-production.md
// 10.3); a recipe with no draw, or a world with the module off, is as it was.
func (v *validator) withAbundance(rc *Recipe, yieldMax int64) int64 {
	if len(rc.Draws) == 0 || v.c.Modules.Resources == nil {
		return yieldMax
	}
	return yieldMax * MaxYieldBP / wholeBP // at most 10^6 * 10^5 / 10^4
}

// drawProducts checks the largest weight times each draw quantity against
// MaxQuantity (spec-production.md 10.3, ErrOverflow).
func (v *validator) drawProducts(rc *Recipe, weight int64, over func([]string, int64, bool, int64)) {
	for _, d := range rc.Draws {
		if q := int64(d.Quantity); q >= 1 && q <= MaxQuantity {
			p, ok := checked.Mul(weight, q)
			over([]string{"recipes", rc.ID, "draws", d.Resource, "quantity"}, p, ok, MaxQuantity)
		}
	}
}

// resourceWarnings returns the warnings of spec-resources.md 13.3 that need no
// solver, in no particular order. They read a configuration that may not
// validate, so each guards its lookups. no_comparative_advantage (7.4.1) is a
// property of the yield table and is reported with the module off.
func resourceWarnings(c *Config, lk lookup) []Warning {
	var out []Warning
	add := func(rule string, segs ...string) { out = append(out, Warning{Path: segs, Rule: rule}) }
	if noComparativeAdvantage(c) {
		add("no_comparative_advantage", "yields")
	}
	m := c.Modules.Resources
	if m == nil {
		return out
	}
	drawn := map[string]bool{}
	for i := range c.Recipes {
		for _, d := range c.Recipes[i].Draws {
			drawn[d.Resource] = true
		}
	}
	rows := make(map[[2]string]*Endowment, len(m.Endowments))
	for i := range m.Endowments {
		e := &m.Endowments[i]
		if rows[[2]string{e.Region, e.Resource}] == nil {
			rows[[2]string{e.Region, e.Resource}] = e
		}
	}
	for i := range m.Resources {
		resourceShapeWarnings(c, &m.Resources[i], rows, drawn, add)
	}
	out = append(out, endowmentWarnings(c, lk, m, rows)...)
	out = append(out, exhaustibleWarnings(c, lk)...)
	return out
}

// resourceShapeWarnings raises the warnings that read one resource alone:
// resource_unused, abundance_not_monotone, resource_inert and
// regen_per_tick_high (spec-resources.md 13.3).
func resourceShapeWarnings(c *Config, r *Resource, rows map[[2]string]*Endowment, drawn map[string]bool, add func(string, ...string)) {
	rb := []string{"modules", "resources", "resources", r.ID}
	if !drawn[r.ID] {
		add("resource_unused", rb...)
	}
	if k := sortedKnots(r.Abundance); !curve.NonDecreasing(k) {
		add("abundance_not_monotone", append(slices.Clone(rb), "abundance")...)
	}
	if r.Shape != ShapeRefill && r.Shape != ShapeLogistic {
		return
	}
	if r.RateBP == 0 {
		add("resource_inert", append(slices.Clone(rb), "rate_bp")...)
		return
	}
	tpy := c.Calendar.TicksPerYear()
	if tpy < 1 || r.RateBP < 0 || r.RateBP > MaxRegenBP {
		return
	}
	// RateBP * m / (10^4 * TPY) above 2 000 basis points a tick, with m the
	// regeneration scale of the resource's row, else 10 000 (7.4.1).
	scales := []int64{wholeBP}
	for key, e := range rows {
		if key[1] == r.ID && e.HasRegenScale && e.RegenScaleBP >= 0 && e.RegenScaleBP <= MaxRegenScaleBP {
			scales = append(scales, e.RegenScaleBP)
		}
	}
	if r.RateBP*slices.Max(scales) > regenPerTickHighBP*wholeBP*tpy {
		add("regen_per_tick_high", append(slices.Clone(rb), "rate_bp")...)
	}
}

// regenPerTickHighBP is the per-tick fraction, in basis points, above which a
// refill or logistic rate is stiff against the tick (spec-resources.md 4.6, 13.3).
// It is a rule constant: a warning changes no result.
const regenPerTickHighBP = 2_000

// sortedKnots returns the knots of a curve by share as internal/curve knots.
func sortedKnots(knots []AbundanceKnot) []curve.Knot {
	out := curveKnots(knots)
	slices.SortStableFunc(out, func(a, b curve.Knot) int { return cmp.Compare(a.X, b.X) })
	return out
}

// endowmentWarnings raises endowment_unused and draw_impossible: an endowment
// with a capacity where no role with a group lists a recipe that draws on it,
// and a recipe that can run in a region whose endowment is below its draw
// (spec-resources.md 13.3).
func endowmentWarnings(c *Config, lk lookup, m *Resources, rows map[[2]string]*Endowment) []Warning {
	var out []Warning
	capacity := func(region, resource string) int64 {
		res := lk.resource[resource]
		if e := rows[[2]string{region, resource}]; e != nil {
			return rowCapacity(e, res)
		}
		if res != nil {
			return res.Capacity
		}
		return 0
	}
	type pair struct{ region, resource string }
	listed := map[pair]bool{} // a role with a group in the region lists a recipe that draws on it
	for i := range c.Groups {
		g := &c.Groups[i]
		ro := lk.role[g.Role]
		if ro == nil {
			continue
		}
		for _, id := range ro.Recipes {
			if rc := lk.recipe[id]; rc != nil {
				for _, d := range rc.Draws {
					listed[pair{g.Region, d.Resource}] = true
				}
			}
		}
	}
	for i := range c.Regions {
		region := c.Regions[i].ID
		for j := range m.Resources {
			id := m.Resources[j].ID
			if capacity(region, id) > 0 && !listed[pair{region, id}] {
				out = append(out, Warning{Path: []string{"modules", "resources", "endowments", region, id}, Rule: "endowment_unused"})
			}
		}
	}
	runs := siteRuns(c, lk)
	for i := range c.Recipes {
		rc := &c.Recipes[i]
		for _, d := range rc.Draws {
			for r := range c.Regions {
				region := c.Regions[r].ID
				if runs(rc, region) && capacity(region, d.Resource) < int64(d.Quantity) {
					out = append(out, Warning{Path: []string{"recipes", rc.ID, "draws", d.Resource}, Rule: "draw_impossible"})
					break
				}
			}
		}
	}
	return out
}

// siteRuns returns whether a recipe has a positive site factor in some season
// of a region: by its yield profile there, else by the default of the recipe
// (spec-production.md 6.2).
func siteRuns(c *Config, lk lookup) func(rc *Recipe, region string) bool {
	profile := make(map[[2]string]*YieldProfile, len(c.Yields))
	for i := range c.Yields {
		y := &c.Yields[i]
		if profile[[2]string{y.Region, y.Recipe}] == nil {
			profile[[2]string{y.Region, y.Recipe}] = y
		}
	}
	return func(rc *Recipe, region string) bool {
		y := profile[[2]string{region, rc.ID}]
		switch {
		case y == nil || !y.HasAll && len(y.Seasons) == 0:
			return !rc.Sited
		case y.HasAll && y.All > 0:
			return true
		}
		return slices.ContainsFunc(y.Seasons, func(s SeasonYield) bool { return s.BP > 0 })
	}
}

// exhaustibleWarnings raises raw_source_exhaustible: a raw commodity whose every
// source (a recipe with no inputs that outputs it) draws on a finite stock, that
// is a constant stock with no rate or a refill or logistic one with no RateBP
// (spec-resources.md 9.3). A raw good with an import or a faucet is EXT's and
// not known here.
func exhaustibleWarnings(c *Config, lk lookup) []Warning {
	var out []Warning
	finite := func(rc *Recipe) bool {
		for _, d := range rc.Draws {
			if r := lk.resource[d.Resource]; r != nil && r.RatePerYear == 0 && r.RateBP == 0 {
				return true
			}
		}
		return false
	}
	for _, cm := range c.Commodities {
		if !cm.Raw {
			continue
		}
		sources, exhaustible := 0, 0
		for i := range c.Recipes {
			rc := &c.Recipes[i]
			if len(rc.Inputs) > 0 || !slices.ContainsFunc(rc.Outputs, func(o Output) bool { return o.Commodity == cm.ID && !o.Discard }) {
				continue
			}
			sources++
			if finite(rc) {
				exhaustible++
			}
		}
		if sources > 0 && sources == exhaustible {
			out = append(out, Warning{Path: []string{"commodities", cm.ID}, Rule: "raw_source_exhaustible"})
		}
	}
	return out
}

// noComparativeAdvantage reports the cross-ratio condition of spec-resources.md
// 7.4.1: at least one pair of regions of one strongly connected component and
// one pair of Sited recipes whose mean site factors are all positive, and no
// cross ratio, or its inverse, that reaches 11 000 basis points. It is false for
// a configuration whose region graph or calendar is invalid.
func noComparativeAdvantage(c *Config) bool {
	var sited []*Recipe
	for i := range c.Recipes {
		if c.Recipes[i].Sited {
			sited = append(sited, &c.Recipes[i])
		}
	}
	if len(sited) < 2 || len(c.Regions) < 2 {
		return false
	}
	g, err := NewGraph(*c)
	if err != nil {
		return false
	}
	mean := meanSites(c, sited)
	if mean == nil {
		return false
	}
	found := false
	for i := range c.Regions {
		for j := i + 1; j < len(c.Regions); j++ {
			ri, okI := g.Lookup(c.Regions[i].ID)
			rj, okJ := g.Lookup(c.Regions[j].ID)
			ci, _ := g.Component(ri)
			cj, _ := g.Component(rj)
			if !okI || !okJ || ci != cj {
				continue
			}
			for a := range sited {
				for b := a + 1; b < len(sited); b++ {
					hit, advantage := crossRatio(mean(a, i), mean(b, j), mean(a, j), mean(b, i))
					if advantage {
						return false
					}
					found = found || hit
				}
			}
		}
	}
	return found
}

// crossRatio returns whether the four mean site factors are all positive, and
// whether the cross ratio ai*bj / (aj*bi) or its inverse reaches 11 000 basis
// points. The means are at most MaxYieldBP, so the products fit in int64.
func crossRatio(ai, bj, aj, bi int64) (positive, advantage bool) {
	if ai <= 0 || bj <= 0 || aj <= 0 || bi <= 0 {
		return false, false
	}
	num, den := ai*bj, aj*bi
	return true, num*wholeBP >= 11_000*den || den*wholeBP >= 11_000*num
}

// meanSites returns the mean site factor, in basis points, of the sited recipe
// a in the region i, over the ticks of a year with the season of each
// (spec-production.md 6.2), truncated; nil when the calendar is not valid. Each
// season is evaluated once and weighted by its ticks.
func meanSites(c *Config, sited []*Recipe) func(a, i int) int64 {
	tpy := c.Calendar.TicksPerYear()
	if tpy < 1 {
		return nil
	}
	ticks := make([]int64, len(c.Calendar.Seasons)+1) // the last cell is no season
	for t := range tpy {
		s := c.Calendar.SeasonAt(t)
		if s < 0 {
			s = len(c.Calendar.Seasons)
		}
		ticks[s]++
	}
	profile := make(map[[2]string]*YieldProfile, len(c.Yields))
	for i := range c.Yields {
		if y := &c.Yields[i]; profile[[2]string{y.Region, y.Recipe}] == nil {
			profile[[2]string{y.Region, y.Recipe}] = y
		}
	}
	return func(a, i int) int64 {
		y := profile[[2]string{c.Regions[i].ID, sited[a].ID}]
		var sum int64
		for s, n := range ticks {
			sum += n * siteAt(y, sited[a].Sited, c.Calendar.Seasons, s)
		}
		return sum / tpy
	}
}

// siteAt is the site factor of a recipe at the season with calendar index s (the
// length of the calendar's list for no season): the entry of the profile, else
// All, else 0 when the profile lists seasons, else the default of the recipe
// (spec-production.md 6.2), the rule of production.Site over the configuration.
func siteAt(y *YieldProfile, sited bool, seasons []Season, s int) int64 {
	if y != nil && s < len(seasons) {
		for _, e := range y.Seasons {
			if e.Season == seasons[s].ID {
				return e.BP
			}
		}
	}
	switch {
	case y != nil && y.HasAll:
		return y.All
	case y != nil && len(y.Seasons) > 0:
		return 0
	case sited:
		return 0
	}
	return wholeBP
}
