package economysim

import (
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/curve"
)

// This file is the validator of the resources module (spec-resources.md 13): the
// bounds of each field, the identity and references of the rows and of the draws
// of the recipes, and the abundance curves. The advice of 13.3 is in
// resources_warn.go.

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
		v.endowmentValues(eb, e, res)
	}
}

// endowmentValues checks the replaced values of one endowment row: its capacity,
// its regeneration scale and its starting stock against the row's capacity.
func (v *validator) endowmentValues(eb []string, e *Endowment, res *Resource) {
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
