package economysim

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/curve"
)

// This file is the warnings of the resources module (spec-resources.md 13.3)
// that need no solver, split from resources_check.go, which holds the validator.

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
	if r.Shape == ShapeRefill || r.Shape == ShapeLogistic {
		regenWarnings(c, r, rows, rb, add)
	}
}

// regenWarnings raises resource_inert and regen_per_tick_high for a refill or
// logistic resource (spec-resources.md 13.3).
func regenWarnings(c *Config, r *Resource, rows map[[2]string]*Endowment, rb []string, add func(string, ...string)) {
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
	if r.RateBP*maxRegenScale(rows, r.ID) > regenPerTickHighBP*wholeBP*tpy {
		add("regen_per_tick_high", append(slices.Clone(rb), "rate_bp")...)
	}
}

// maxRegenScale is the largest regeneration scale, in basis points, of the rows of
// a resource that carry one in bound, and 10 000 when none is larger.
func maxRegenScale(rows map[[2]string]*Endowment, resource string) int64 {
	largest := int64(wholeBP)
	for key, e := range rows {
		if key[1] == resource && e.HasRegenScale && e.RegenScaleBP >= 0 && e.RegenScaleBP <= MaxRegenScaleBP {
			largest = max(largest, e.RegenScaleBP)
		}
	}
	return largest
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
	out := unusedEndowments(c, lk, m, capacity)
	return append(out, impossibleDraws(c, lk, capacity)...)
}

// regionResource names an endowment by its region and resource IDs.
type regionResource struct{ region, resource string }

// unusedEndowments raises endowment_unused, in (region, resource) order.
func unusedEndowments(c *Config, lk lookup, m *Resources, capacity func(region, resource string) int64) []Warning {
	var out []Warning
	listed := map[regionResource]bool{} // a role with a group in the region lists a recipe that draws on it
	for i := range c.Groups {
		g := &c.Groups[i]
		ro := lk.role[g.Role]
		if ro == nil {
			continue
		}
		for _, id := range jobIDs(c, ro) { // a draw of the own-use list counts (D413)
			if rc := lk.recipe[id]; rc != nil {
				for _, d := range rc.Draws {
					listed[regionResource{g.Region, d.Resource}] = true
				}
			}
		}
	}
	for i := range c.Regions {
		region := c.Regions[i].ID
		for j := range m.Resources {
			id := m.Resources[j].ID
			if capacity(region, id) > 0 && !listed[regionResource{region, id}] {
				out = append(out, Warning{Path: []string{"modules", "resources", "endowments", region, id}, Rule: "endowment_unused"})
			}
		}
	}
	return out
}

// impossibleDraws raises draw_impossible, once per recipe draw, in recipe order.
func impossibleDraws(c *Config, lk lookup, capacity func(region, resource string) int64) []Warning {
	var out []Warning
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
	sited := sitedRecipes(c)
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
			if !sameComponent(&g, c.Regions[i].ID, c.Regions[j].ID) {
				continue
			}
			hit, advantage := recipePairsCross(mean, len(sited), i, j)
			if advantage {
				return false
			}
			found = found || hit
		}
	}
	return found
}

// sitedRecipes returns the Sited recipes of the configuration, in recipe order.
func sitedRecipes(c *Config) []*Recipe {
	var sited []*Recipe
	for i := range c.Recipes {
		if c.Recipes[i].Sited {
			sited = append(sited, &c.Recipes[i])
		}
	}
	return sited
}

// sameComponent reports whether two regions exist and share a strongly connected
// component of the region graph.
func sameComponent(g *Graph, a, b string) bool {
	ra, okA := g.Lookup(a)
	rb, okB := g.Lookup(b)
	ca, _ := g.Component(ra)
	cb, _ := g.Component(rb)
	return okA && okB && ca == cb
}

// recipePairsCross applies crossRatio to every pair of the n sited recipes over
// the regions i and j: whether some pair has all four means positive, and
// whether some pair reaches the advantage.
func recipePairsCross(mean func(a, i int) int64, n, i, j int) (positive, advantage bool) {
	for a := range n {
		for b := a + 1; b < n; b++ {
			hit, adv := crossRatio(mean(a, i), mean(b, j), mean(a, j), mean(b, i))
			if adv {
				return false, true
			}
			positive = positive || hit
		}
	}
	return positive, false
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
