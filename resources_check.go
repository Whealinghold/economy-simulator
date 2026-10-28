package economysim

import (
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/curve"
	"github.com/Whealinghold/economy-sim/internal/steady"
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

// resSupply reads the endowments of a configuration for the viability rules
// (spec-resources.md 9.2, 11): which recipes can start where, the rows of the
// programme and its abundance. It reads the configuration as written, so it is
// safe on one that does not validate (Warnings), where a repeated ID keeps its
// first definition. It is nil while the module is off.
type resSupply struct {
	c    *Config
	idx  map[string]int
	rows map[[2]string]*Endowment
}

// newResSupply indexes the resources and the rows of c, or returns nil when the
// module is off.
func newResSupply(c *Config) *resSupply {
	m := c.Modules.Resources
	if m == nil {
		return nil
	}
	s := &resSupply{c: c, idx: make(map[string]int, len(m.Resources)), rows: make(map[[2]string]*Endowment, len(m.Endowments))}
	for i := range m.Resources {
		if _, seen := s.idx[m.Resources[i].ID]; !seen {
			s.idx[m.Resources[i].ID] = i
		}
	}
	for i := range m.Endowments {
		key := [2]string{m.Endowments[i].Region, m.Endowments[i].Resource}
		if s.rows[key] == nil {
			s.rows[key] = &m.Endowments[i]
		}
	}
	return s
}

// levels returns the capacity and the starting stock of the endowment of resource
// r in a region at tick 0: the row's capacity, else the resource's, and the
// row's starting stock, else the capacity times InitialBP over 10 000, at most the
// capacity (spec-resources.md 3.3, 3.4; the shock scale and the facility layer
// are neutral).
func (s *resSupply) levels(region string, r *Resource) (capacity, stock int64) {
	row := s.rows[[2]string{region, r.ID}]
	capacity = r.Capacity
	if row != nil && row.HasCapacity {
		capacity = row.Capacity
	}
	if row != nil && row.HasInitial {
		stock = row.Initial
	} else {
		stock, _ = checked.MulDiv(capacity, r.InitialBP, wholeBP) // in bounds: Validate passed
	}
	return capacity, min(stock, capacity)
}

// regrows reports that a resource with the starting stock gives units back: a
// positive rate and, for a logistic stock, a stock or a refuge to grow from
// (spec-resources.md 9.2).
func (r *Resource) regrows(stock int64) bool {
	switch r.Shape {
	case ShapeRefill:
		return r.RateBP > 0
	case ShapeLogistic:
		return r.RateBP > 0 && (stock > 0 || r.Refuge > 0)
	}
	return r.RatePerYear > 0
}

// supplies is the producibility predicate of spec-resources.md 9.2: in the
// region every draw can start once, because the effective capacity is at least
// the draw and the starting stock is positive or the shape regrows. A recipe with
// no draw is supplied, and so is every recipe while the module is off.
func (s *resSupply) supplies(region string, draws []Draw) bool {
	if s == nil {
		return true
	}
	for _, d := range draws {
		i, ok := s.idx[d.Resource]
		if !ok {
			return false
		}
		r := &s.c.Modules.Resources.Resources[i]
		capacity, stock := s.levels(region, r)
		if capacity < int64(d.Quantity) || stock == 0 && !r.regrows(stock) {
			return false
		}
	}
	return true
}

// apply sets the draws of a programme activity from the draws of its recipe and
// its mean factor's abundance: the least over the draws of the curve at the plan
// share, 10 000 for a flat curve (spec-resources.md 4.7.4, 11.2).
func (s *resSupply) apply(a *steady.Activity, draws []Draw) {
	for i, d := range draws {
		j := s.idx[d.Resource]
		a.Draws = append(a.Draws, steady.Draw{Resource: j, Qty: int64(d.Quantity)})
		r := &s.c.Modules.Resources.Resources[j]
		y := int64(wholeBP)
		if len(r.Abundance) > 0 {
			y, _ = curve.Eval(curveKnots(r.Abundance), r.PlanShareBP) // in range: Validate passed
		}
		if i == 0 || y < a.AbundanceBP {
			a.AbundanceBP = y
		}
	}
}

// endows returns the resource rows of a programme over the regions: one for each
// resource that an activity of the region draws on, ascending by region then
// resource, from the configuration (spec-resources.md 11.1). seasons is the
// year as stretches of one season.
func (s *resSupply) endows(regions []RegionID, acts [][]viaActivity, seasons []seasonRun) []steady.Endow {
	if s == nil {
		return nil
	}
	var out []steady.Endow
	for k, reg := range regions {
		var used []int
		for _, a := range acts[reg] {
			for _, d := range a.act.Draws {
				if !slices.Contains(used, d.Resource) {
					used = append(used, d.Resource)
				}
			}
		}
		slices.Sort(used)
		for _, j := range used {
			out = append(out, s.endow(k, s.c.Regions[reg].ID, j, seasons))
		}
	}
	return out
}

// endow is the resource row of resource j in region k (named id).
func (s *resSupply) endow(k int, id string, j int, seasons []seasonRun) steady.Endow {
	m := s.c.Modules.Resources
	r := &m.Resources[j]
	capacity, stock := s.levels(id, r)
	e := steady.Endow{
		Resource: j, Region: k, Capacity: capacity, Stock: stock, PlanShareBP: r.PlanShareBP,
		ScaleBP: wholeBP, Horizon: m.HorizonYears, Seasons: s.seasonFactors(r, seasons),
	}
	switch r.Shape {
	case ShapeRefill:
		e.Shape, e.Rate = steady.Refill, r.RateBP
	case ShapeLogistic:
		e.Shape, e.Rate = steady.Logistic, r.RateBP
	default:
		e.Shape, e.Rate = steady.Constant, r.RatePerYear
	}
	if row := s.rows[[2]string{id, r.ID}]; row != nil && row.HasRegenScale {
		e.ScaleBP = row.RegenScaleBP
	}
	return e
}

// seasonFactors turns the regeneration profile of a resource into runs of the
// year: the factor of the season each stretch is in, 0 for a season the profile
// does not list (spec-resources.md 4.5.1). It is nil, 10 000 on every tick, for
// a resource with no profile.
func (s *resSupply) seasonFactors(r *Resource, seasons []seasonRun) []steady.Run {
	if len(r.Seasons) == 0 {
		return nil
	}
	out := make([]steady.Run, len(seasons))
	for i, sr := range seasons {
		out[i] = steady.Run{Length: sr.length}
		if sr.season < 0 {
			continue
		}
		id := s.c.Calendar.Seasons[sr.season].ID
		if k := slices.IndexFunc(r.Seasons, func(x SeasonRegen) bool { return x.Season == id }); k >= 0 {
			out[i].FactorBP = r.Seasons[k].BP
		}
	}
	return out
}

// cause names the first resource row of a solved model that limits the surplus,
// its draw holding the sustainable draw with a positive rent, as the path
// regions, <region>, resources, <resource> (spec-resources.md 11.3); nil when
// none does, or while the module is off. regions are the regions of the model.
func (s *resSupply) cause(regions []RegionID, md steady.Model, out steady.Outcome) []string {
	if s == nil {
		return nil
	}
	for j, e := range md.Resources {
		if out.ResTight[j] && out.ResRent[j].Sign() > 0 {
			return []string{"regions", s.c.Regions[regions[e.Region]].ID, "resources", s.c.Modules.Resources.Resources[e.Resource].ID}
		}
	}
	return nil
}

// peaks reports the season peak of one draw of an activity a in region k of the
// model md, whose flow at lambda 1 is flow executions a tick: some window of its
// starts draws more than the stock at the window's opening (spec-resources.md
// 11.3, steady.Endow.PeakExceeds).
func (s *resSupply) peaks(md steady.Model, k int, a viaActivity, d steady.Draw, flow steady.Frac) bool {
	var row steady.Endow
	for _, e := range md.Resources {
		if e.Region == k && e.Resource == d.Resource {
			row = e
		}
	}
	perTick := flow.Mul(d.Qty, 1)
	for _, w := range startWindows(a) {
		if row.PeakExceeds(s.c.Calendar.TicksPerYear(), w, a.count.Starts, perTick) {
			return true
		}
	}
	return false
}

// startWindows returns the windows of an activity's year: the maximal stretches
// of ticks with a positive yield factor, each with the starts that fall in it.
// A start is the tick its job completes at less the duration plus 1.
func startWindows(a viaActivity) []steady.Window {
	var ws []steady.Window
	var at int64
	open := false
	for _, r := range a.runs {
		switch {
		case r.FactorBP <= 0:
			open = false
		case !open:
			ws, open = append(ws, steady.Window{Start: at}), true
			fallthrough
		default:
			ws[len(ws)-1].End = at + r.Length
		}
		at += r.Length
	}
	for _, c := range a.count.Completions {
		start := c - a.act.Duration + 1
		for i := range ws {
			if start >= ws[i].Start && start < ws[i].End {
				ws[i].Starts++
				break
			}
		}
	}
	return ws
}

// endowments adds the rows of the resources module to the report for the
// programme p: the headroom and the rent of each resource row and the run-down
// time of each finite stock that the reference programme draws (spec-resources.md
// 11.3). scale is the price scale of the programme, as for the wages.
func (s *viaSolve) endowments(p *viaProgramme, scale steady.Frac) {
	for j, e := range p.md.Resources {
		region := s.m.c.Regions[p.regions[e.Region]].ID
		res := &s.m.c.Modules.Resources.Resources[e.Resource]
		rent, ok := p.out.ResRent[j].Times(scale).Round(0, MaxMoney)
		if !ok {
			rent = MaxMoney
		}
		s.rep.Rent = append(s.rep.Rent, RentRow{Region: region, Resource: res.ID, Rent: rent})
		if draw := p.out.ResDraw[j]; draw.Sign() > 0 {
			s.headroom(region, res, e, p.out.ResPhi[j], draw)
		}
	}
}

// headroom adds the headroom of a resource row that the reference programme
// draws on, phi over the draw in basis points, and for a finite stock its
// runway, the starting stock over the draw in ticks.
func (s *viaSolve) headroom(region string, res *Resource, e steady.Endow, phi, draw steady.Frac) {
	q, _ := phi.Quo(draw) // the draw is positive
	bp, fits := q.Mul(wholeBP, 1).Floor(0, math.MaxInt64)
	if !fits {
		bp = math.MaxInt64
	}
	s.rep.Headroom = append(s.rep.Headroom, HeadroomRow{Region: region, Resource: res.ID, BP: bp})
	if res.RatePerYear != 0 || res.RateBP != 0 {
		return
	}
	runway, _ := steady.Rat(e.Stock, 1).Quo(draw)
	ticks, fits := runway.Floor(0, math.MaxInt64)
	if !fits {
		ticks = math.MaxInt64
	}
	s.rep.Runway = append(s.rep.Runway, RunwayRow{Region: region, Resource: res.ID, Ticks: ticks})
}
