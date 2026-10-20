package economysim

import (
	"fmt"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/steady"
)

// This file builds the warm-start table of SolveSteadyState from the programmes
// solved: the seed prices and wages (spec-viability.md 6.3, 6.4, 8.2), the money
// (8.3), the stock (8.4) and the jobs (8.5), by the composition rules of D134. The
// diagnostics that are not part of the table are viability_report.go's. Every
// conversion to int64 is checked against int64 and the field's bound
// (architecture.md 3.5); nothing here draws a random number or reads a map in
// an order that reaches a result.

// maxInt64 is the largest int64, spelled without the math package, which the
// viability files do not import (FR-VIA-015).
const maxInt64 = 1<<63 - 1

// regionCtx is one region of one programme while its rows are made.
type regionCtx struct {
	s       *viaSolve
	p       *viaProgramme
	r       RegionID
	k       int
	id      string
	scale   steady.Frac
	price   []int64 // by commodity; 0 is a commodity with no row
	wage    int64   // 0 is no starting wage
	acts    []viaActivity
	actOf   []int   // by recipe: its index in acts, or -1
	listing []int64 // by recipe: households of the groups of the region that list it
	table   []int   // the commodities with a row, ascending
	groups  []groupCtx
	nVirt   map[RecipeID]int64 // by recipe: the count of its virtual start offsets, once asked
}

// groupCtx is one group of a region.
type groupCtx struct {
	g       *Group
	role    RoleID
	row     *roleRow
	up      []int64 // the upkeep of a household, by commodity
	active  []activeRecipe
	reserve []int64 // the lean reserve per household, by commodity
	short   bool    // the job shares were scaled down to 10 000
}

// activeRecipe is a recipe of a group that has a positive flow at lambda 1: its
// index in the region's activities and the part of the flow the group runs
// (D121: each household of the groups that list the recipe is equally likely to
// run it).
type activeRecipe struct {
	i     int
	share steady.Frac
}

// build makes the table and the report of the programmes (D120 to D134).
func (s *viaSolve) build() error {
	for _, p := range s.progs {
		if err := s.programmeTable(p); err != nil {
			return err
		}
	}
	s.advantage()
	s.surpluses()
	return nil
}

// programmeTable makes the rows of every region of a programme. A programme
// with no surplus (lambda 0: a region that cannot make its own basket) has no
// price vector, since the normalisation of the duals holds only for a positive
// lambda (5.2), and its regions are listed unpriced.
func (s *viaSolve) programmeTable(p *viaProgramme) error {
	s.rep.Degenerate = s.rep.Degenerate || p.out.Degenerate
	s.rep.AlternativeOptima = s.rep.AlternativeOptima || p.out.Alternative
	for _, k := range p.nominalSeed {
		s.rep.NominalSeed = append(s.rep.NominalSeed, s.m.c.Regions[p.regions[k]].ID)
	}
	scale, ok := s.scale(p)
	if !ok || p.out.Lambda.Sign() == 0 { // the duals are not a price vector without a surplus
		s.unprogrammed(p.regions)
		return nil
	}
	for k := range p.regions {
		rc := s.newRegion(p, k, scale)
		if err := rc.rows(); err != nil {
			return err
		}
		rc.report()
	}
	return nil
}

// scale returns the price scale of 6.3: the households times the reference
// basket of the programme's regions over the cost of their upkeep at the
// solver's prices, so the regional basket (autarky) or the household-weighted
// baskets (pooled) equal the configured ones (FR-VIA-024). ok is false when the
// upkeep costs nothing.
func (s *viaSolve) scale(p *viaProgramme) (steady.Frac, bool) {
	num, den := steady.Rat(0, 1), steady.Rat(0, 1)
	for _, r := range p.regions {
		num = num.Add(steady.Rat(int64(s.m.c.Regions[r].ReferenceBasket), 1).Mul(s.m.households[r], 1))
		for ci, d := range s.m.upkeep[r] {
			den = den.Add(p.pr.Price[ci].Mul(d, 1))
		}
	}
	return num.Quo(den)
}

// newRegion builds the context of region k of a programme.
func (s *viaSolve) newRegion(p *viaProgramme, k int, scale steady.Frac) *regionCtx {
	m := s.m
	r := p.regions[k]
	rc := &regionCtx{
		s: s, p: p, r: r, k: k, id: m.c.Regions[r].ID, scale: scale, acts: m.acts[r],
		price: make([]int64, len(m.c.Commodities)), actOf: make([]int, len(m.c.Recipes)),
		listing: make([]int64, len(m.c.Recipes)),
	}
	for i := range rc.actOf {
		rc.actOf[i] = -1
	}
	for i, a := range rc.acts {
		rc.actOf[a.recipe] = i
	}
	for gi := range m.c.Groups {
		g := &m.c.Groups[gi]
		if gr, _ := m.c.regionID(g.Region); gr != r {
			continue
		}
		ro, _ := m.c.roleID(g.Role)
		gc := groupCtx{g: g, role: ro, row: m.roles.row(ro), up: make([]int64, len(m.c.Commodities)), reserve: make([]int64, len(m.c.Commodities))}
		for _, it := range gc.row.upkeepIn(r) {
			gc.up[it.commodity] = int64(it.quantity)
		}
		for _, k := range gc.row.recipes {
			rc.listing[k] += g.Households
		}
		rc.groups = append(rc.groups, gc)
	}
	rc.tableCommodities()
	rc.activate()
	return rc
}

// tableCommodities lists the commodities that have a row: those in the upkeep,
// an input, a tool or a non-discard output of the region (D134 (7)).
func (rc *regionCtx) tableCommodities() {
	used := make([]bool, len(rc.price))
	for ci, d := range rc.s.m.upkeep[rc.r] {
		used[ci] = d > 0
	}
	for _, a := range rc.acts {
		for _, it := range a.act.Inputs {
			used[it.Commodity] = true
		}
		for _, t := range a.act.Tools {
			used[t.Commodity] = true
		}
		for _, o := range a.act.Outputs {
			used[o.Commodity] = used[o.Commodity] || !o.Discard
		}
	}
	for ci, u := range used {
		if u {
			rc.table = append(rc.table, ci)
		}
	}
}

// flow returns the flow at lambda 1 of the region's activity i (D120).
func (rc *regionCtx) flow(i int) steady.Frac { return rc.p.out.Flow[rc.p.base[rc.k]+i] }

// activeAt reports a positive flow of the region's activity i.
func (rc *regionCtx) activeAt(i int) bool { return rc.p.out.Active[rc.p.base[rc.k]+i] }

// activate lists, for each group, its recipes with a positive flow and its part
// of that flow.
func (rc *regionCtx) activate() {
	for gi := range rc.groups {
		gc := &rc.groups[gi]
		for _, k := range gc.row.recipes {
			if i := rc.actOf[k]; i >= 0 && rc.activeAt(i) {
				gc.active = append(gc.active, activeRecipe{i: i, share: rc.flow(i).Mul(gc.g.Households, rc.listing[k])})
			}
		}
	}
}

// rows makes the prices, the wage and the groups of the region.
func (rc *regionCtx) rows() error {
	if err := rc.prices(); err != nil {
		return err
	}
	if err := rc.wages(); err != nil {
		return err
	}
	rc.reserves()
	for gi := range rc.groups {
		gc := &rc.groups[gi]
		stock, err := rc.stock(gc)
		if err != nil {
			return err
		}
		money, err := rc.money(gc)
		if err != nil {
			return err
		}
		rc.s.ws.Groups = append(rc.s.ws.Groups, WarmGroup{
			Region: rc.id, Role: rc.s.m.c.Roles[gc.role].ID, Money: Money(money), Stock: stock, Jobs: rc.jobs(gc),
		})
	}
	return nil
}

// band returns the reference band of a price in minor units, as warmRules checks
// it: from RefBandLow to RefBandHigh basis points of the reference price, the
// lower bound rounded up and the upper down.
func (s *viaSolve) band(ref int64) (lo, hi int64, err error) {
	w := s.m.c.World
	lo, ok1 := checked.MulDivCeil(ref, w.RefBandLow, wholeBP)
	hi, ok2 := checked.MulDiv(ref, w.RefBandHigh, wholeBP)
	if !ok1 || !ok2 {
		return 0, 0, fmt.Errorf("%w: reference band of a price of %d", ErrOverflow, ref)
	}
	return lo, hi, nil
}

// seedPrice makes the seed of a dual (6.3, 6.4): the dual times the scale rounded
// half to even, raised to the resolution floor, capped at MaxPrice, and lifted
// into (or capped to) the reference band so the table passes Validate (O11).
// unclamped is the price before the band, and lifted says the floor or the band
// raised it.
func (s *viaSolve) seedPrice(dual, scale steady.Frac, cm *Commodity) (seed, unclamped int64, lifted bool, err error) {
	floor, _ := resolutionFloor(s.m.c.World) // in bounds: Validate passed
	p0, ok := dual.Times(scale).Round(0, MaxPrice)
	if !ok {
		p0 = MaxPrice
	}
	p1 := min(max(p0, floor), MaxPrice)
	lo, hi, err := s.band(int64(cm.ReferencePrice))
	if err != nil {
		return 0, 0, false, err
	}
	seed = min(max(p1, lo), hi)
	return seed, p1, p1 > p0 || seed > p1, nil
}

// prices makes the seed price rows of the region and the records of the report.
func (rc *regionCtx) prices() error {
	rep := &rc.s.rep
	for _, c := range rc.table {
		cm := &rc.s.m.c.Commodities[c]
		dual := rc.p.pr.Price[c]
		seed, p1, lifted, err := rc.s.seedPrice(dual, rc.scale, cm)
		if err != nil {
			return err
		}
		rc.price[c] = seed
		rc.s.ws.Prices = append(rc.s.ws.Prices, SeedPrice{Region: rc.id, Commodity: cm.ID, Price: Price(seed)})
		key := RegionCommodity{rc.id, cm.ID}
		// shortcut: only dual 0 is unpriced; a non-unique non-zero dual is not detected (N-311).
		if dual.Sign() == 0 {
			rep.UnpricedCommodities = append(rep.UnpricedCommodities, key)
		}
		if lifted {
			rep.Floored = append(rep.Floored, key)
		}
		if seed != p1 {
			rep.OutsideReferenceBand = append(rep.OutsideReferenceBand, PriceRecord{key.Region, key.Commodity, p1, seed})
		}
		rep.Drift = append(rep.Drift, driftOf(key, seed, int64(cm.ReferencePrice)))
	}
	return nil
}

// driftOf is the drift of a seed from its reference price in basis points,
// rounded toward zero.
func driftOf(rc RegionCommodity, seed, ref int64) DriftRow {
	d := seed - ref
	bp, _ := checked.MulDiv(max(d, -d), wholeBP, ref) // |d| < 10^9, ref >= 1: cannot overflow
	if d < 0 {
		bp = -bp
	}
	return DriftRow{rc.Region, rc.Commodity, bp}
}

// wages makes the starting wage of the region: the labour dual times the scale
// rounded half to even; none when it is 0 (FR-VIA-050).
func (rc *regionCtx) wages() error {
	w, ok := rc.p.pr.Wage[rc.k].Times(rc.scale).Round(0, MaxMoney)
	if !ok {
		return fmt.Errorf("%w: starting wage of region %q", ErrOverflow, rc.id)
	}
	if w < 1 {
		rc.s.rep.UnpricedWages = append(rc.s.rep.UnpricedWages, rc.id)
		return nil
	}
	rc.wage = w
	rc.s.ws.Wages = append(rc.s.ws.Wages, SeedWage{Region: rc.id, Wage: Money(w)})
	return nil
}

// cost is the sum of quantity times seed price over a list, checked.
func (rc *regionCtx) cost(items []steady.Item) (int64, bool) {
	var sum int64
	for _, it := range items {
		v, ok := checked.Mul(it.Qty, rc.price[it.Commodity])
		if !ok {
			return 0, false
		}
		if sum, ok = checked.Add(sum, v); !ok {
			return 0, false
		}
	}
	return sum, true
}

// mainRecipe returns the main recipe of a group and the longest duration among
// its active recipes (8.3.1): the longest Duration, ties broken by the largest
// inputs cost, then by the lower recipe ID (the active list is in recipe order,
// so the first of equals wins). main is -1 when the group has none.
func (rc *regionCtx) mainRecipe(gc *groupCtx) (main int, dmax int64, inputs int64, err error) {
	main = -1
	for _, ar := range gc.active {
		a := &rc.acts[ar.i]
		cost, ok := rc.cost(a.act.Inputs)
		if !ok {
			return -1, 0, 0, fmt.Errorf("%w: money of group %q in region %q", ErrOverflow, rc.s.m.c.Roles[gc.role].ID, rc.id)
		}
		d := a.act.Duration
		if main < 0 || d > dmax || (d == dmax && cost > inputs) {
			main, inputs = ar.i, cost
		}
		dmax = max(dmax, d)
	}
	return main, dmax, inputs, nil
}

// money makes the money of a household of the group (8.3.1): the inputs of one
// execution of its main recipe plus its upkeep at seed prices for the longest
// active job and the buffer, capped at MaxMoney.
func (rc *regionCtx) money(gc *groupCtx) (int64, error) {
	_, dmax, inputs, err := rc.mainRecipe(gc)
	if err != nil {
		return 0, err
	}
	var u int64
	for c, q := range gc.up {
		v, ok := checked.Mul(q, rc.price[c])
		if !ok {
			return 0, fmt.Errorf("%w: upkeep of group %q in region %q", ErrOverflow, rc.s.m.c.Roles[gc.role].ID, rc.id)
		}
		if u, ok = checked.Add(u, v); !ok {
			return 0, fmt.Errorf("%w: upkeep of group %q in region %q", ErrOverflow, rc.s.m.c.Roles[gc.role].ID, rc.id)
		}
	}
	span, ok1 := checked.Add(dmax, rc.s.o.moneyBuffer)
	hold, ok2 := checked.Mul(u, span)
	total, ok3 := checked.Add(inputs, hold)
	if !ok1 || !ok2 || !ok3 {
		return 0, fmt.Errorf("%w: money of group %q in region %q", ErrOverflow, rc.s.m.c.Roles[gc.role].ID, rc.id)
	}
	return min(total, MaxMoney), nil
}

// stock makes the stock of a household of the group (8.4.1, D134 (2)): for each
// commodity the larger of the cover of its upkeep and the inputs of one
// execution of each active recipe, plus the lean reserve and one item of each
// required tool. The cover is cut to the shelf life (8.4.2, FR-VIA-039).
func (rc *regionCtx) stock(gc *groupCtx) ([]Item, error) {
	nc := len(rc.price)
	inputs, tools := make([]int64, nc), make([]int64, nc)
	for _, ar := range gc.active {
		for _, it := range rc.acts[ar.i].act.Inputs {
			inputs[it.Commodity] += it.Qty // at most MaxRequires items of MaxQuantity
		}
		for _, t := range rc.acts[ar.i].act.Tools {
			tools[t.Commodity] = 1
		}
	}
	var out []Item
	for _, c := range rc.table {
		cover, ok := rc.cover(gc, c)
		q, ok2 := checked.Add(max(cover, inputs[c]), gc.reserve[c]+tools[c])
		if !ok || !ok2 {
			return nil, fmt.Errorf("%w: stock of %q in group %q of region %q", ErrOverflow, rc.s.m.c.Commodities[c].ID, rc.s.m.c.Roles[gc.role].ID, rc.id)
		}
		if q > 0 {
			out = append(out, Item{Commodity: rc.s.m.c.Commodities[c].ID, Quantity: Quantity(q)})
		}
	}
	return out, nil
}

// cover returns the units of a commodity that StockCoverTicks ticks of the
// group's upkeep need, cut to the upkeep of its shelf life (8.4.2, FR-VIA-039),
// which the report lists. ok is false when the product does not fit int64.
func (rc *regionCtx) cover(gc *groupCtx, c int) (cover int64, ok bool) {
	cover, ok = checked.Mul(rc.s.o.stockCover, gc.up[c])
	shelf := rc.s.m.c.Commodities[c].ShelfLife
	if !ok || shelf == 0 {
		return cover, ok
	}
	if life, lok := checked.Mul(gc.up[c], shelf); lok && cover > life {
		rc.s.rep.CapShortened = append(rc.s.rep.CapShortened, RegionCommodity{rc.id, rc.s.m.c.Commodities[c].ID})
		return life, true
	}
	return cover, true
}

// reserves fills the lean reserve of every group for each commodity the region's
// activities output and that has a supply gap (4.5, D123): the largest
// cumulative shortfall of the whole consumption against the production over the
// year, shared among the groups that make it in proportion to their output and
// rounded up per household.
func (rc *regionCtx) reserves() {
	m := rc.s.m
	prods := m.producers(rc.r)
	for c := range m.c.Commodities {
		if len(prods[c]) == 0 || steady.Gap(m.year, prods[c]) == 0 {
			continue
		}
		total, cut := rc.reserveTotal(c)
		if cut {
			rc.s.rep.CapShortened = append(rc.s.rep.CapShortened, RegionCommodity{rc.id, m.c.Commodities[c].ID})
		}
		weights := make([]steady.Frac, len(rc.groups))
		for gi := range rc.groups {
			weights[gi] = steady.Rat(0, 1)
			for _, ar := range rc.groups[gi].active {
				weights[gi] = weights[gi].Add(ar.share.Times(rc.acts[ar.i].act.Yield(c)))
			}
		}
		for gi, share := range steady.Apportion(total, weights) {
			h := rc.groups[gi].g.Households
			rc.groups[gi].reserve[c] = ceilDiv(share, h)
		}
	}
}

// ceilDiv returns the ceiling of n / d for n >= 0, d > 0 and n <= MaxInt64 - d
// (a share is at most the total, which is at most MaxInt64 minus the households).
func ceilDiv(n, d int64) int64 { return (n + d - 1) / d }

// reserveTotal returns the lean reserve of a commodity in whole units for the
// region, cut to what the region consumes within the shelf life (8.4.2).
func (rc *regionCtx) reserveTotal(c int) (total int64, cut bool) {
	m := rc.s.m
	cons := steady.Rat(m.upkeep[rc.r][c], 1)
	var srcs []steady.Source
	for i, a := range rc.acts {
		if !rc.activeAt(i) {
			continue
		}
		cons = cons.Add(rc.flow(i).Times(a.act.Uses(c)))
		if y := outQty(a.act, c); y > 0 {
			srcs = append(srcs, steady.Source{Duration: a.act.Duration, Flow: rc.flow(i), Yield: y, Year: a.runs})
		}
	}
	total, _ = steady.Reserve(m.year, cons, srcs).Ceil(0, maxInt64-MaxHouseholds)
	if shelf := m.c.Commodities[c].ShelfLife; shelf > 0 {
		if life, ok := cons.Mul(shelf, 1).Ceil(0, maxInt64-MaxHouseholds); ok && total > life {
			return life, true
		}
	}
	return total, false
}

// jobs makes the jobs in progress of a group (8.5.1, D134): for each active
// recipe of two ticks or more with a virtual start, the executions in progress
// x * TicksPerYear * |V| / N_open over the households of the groups that list
// the recipe, in basis points rounded down and clamped to 10 000; shares that
// sum above 10 000 are scaled down in proportion (D121).
func (rc *regionCtx) jobs(gc *groupCtx) []JobSpec {
	m := rc.s.m
	var jobs []JobSpec
	var sum int64
	for _, ar := range gc.active {
		a := &rc.acts[ar.i]
		if a.act.Duration < 2 { // a one-tick recipe has no job in progress at tick 0
			continue
		}
		v := rc.virtualStarts(a.recipe)
		den, ok := checked.Mul(openTicks(a.runs), rc.listing[a.recipe])
		if v == 0 || !ok {
			continue
		}
		num, _ := checked.Mul(m.year, v) // at most MaxTicksPerYear * MaxRecipeTicks
		bp, fits := rc.flow(ar.i).Mul(num, den).Mul(wholeBP, 1).Floor(0, maxInt64)
		if !fits { // beyond int64: far above the clamp
			bp = wholeBP
		}
		if bp = min(bp, wholeBP); bp > 0 {
			jobs = append(jobs, JobSpec{Recipe: m.c.Recipes[a.recipe].ID, InProgressBP: bp})
			sum += bp // at most 10 000 per job
		}
	}
	if sum > wholeBP {
		gc.short = true
		jobs = scaleJobs(jobs, sum)
	}
	return jobs
}

// virtualStarts is the number of virtual start offsets of a recipe in the
// region, memoised in nVirt.
func (rc *regionCtx) virtualStarts(recipe RecipeID) int64 {
	if v, seen := rc.nVirt[recipe]; seen {
		return v
	}
	if rc.nVirt == nil {
		rc.nVirt = map[RecipeID]int64{}
	}
	m := rc.s.m
	v := int64(len(virtualStartOffsets(m.c, &m.yt, rc.r, recipe)))
	rc.nVirt[recipe] = v
	return v
}

// scaleJobs scales the shares of jobs, which sum to sum above 10 000, down in
// proportion, rounded down; a share scaled to 0 is no job.
func scaleJobs(jobs []JobSpec, sum int64) []JobSpec {
	scaled := jobs[:0]
	for _, j := range jobs {
		if j.InProgressBP = j.InProgressBP * wholeBP / sum; j.InProgressBP > 0 {
			scaled = append(scaled, j)
		}
	}
	return scaled
}

// outQty is the quantity of a commodity one execution outputs before the yield
// factor, over the non-discard outputs.
func outQty(a steady.Activity, c int) int64 {
	var q int64
	for _, o := range a.Outputs {
		if o.Commodity == c && !o.Discard {
			q += o.Qty
		}
	}
	return q
}

// openTicks is the number of ticks of the year with a positive yield factor
// (N_open of 8.5.1).
func openTicks(runs []steady.Run) int64 {
	var n int64
	for _, r := range runs {
		if r.FactorBP > 0 {
			n += r.Length
		}
	}
	return n
}

// finish assembles the result: the programmes, the table in canonical order and
// the report in stable order. It runs the structural rules of Validate on the
// configuration with the table, so a table that breaks a bound (a wage, a stock
// times a weight, the money at start, an empty set of offsets) is that error and
// never returned.
func (s *viaSolve) finish() (SteadyState, error) {
	tmp := Config{WarmStart: &s.ws}
	table := *tmp.canonical().WarmStart
	full := *s.m.c
	full.WarmStart = &table
	v := &validator{c: full}
	v.structural()
	mm := *s.m
	mm.c = &full
	if len(v.rep.errs) == 0 {
		v.emptyOffsets(&mm)
	}
	if err := v.rep.err(); err != nil {
		return SteadyState{}, err
	}
	st := SteadyState{Table: table, Report: s.rep}
	for _, p := range s.progs {
		st.Programmes = append(st.Programmes, s.result(p))
	}
	st.Report.normalise()
	return st, nil
}

// result is the public form of a programme.
func (s *viaSolve) result(p *viaProgramme) ProgrammeResult {
	res := ProgrammeResult{
		SurplusBP: p.out.SurplusBP, LabourShareBP: p.out.LabourShareBP, HasLabourShare: p.out.HasLabourShare,
		NominalSurplusBP: p.nominalBP, Pivots: p.out.Pivots,
	}
	for _, r := range p.regions {
		res.Regions = append(res.Regions, s.m.c.Regions[r].ID)
	}
	return res
}
