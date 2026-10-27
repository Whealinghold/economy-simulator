package economysim

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/production"
	"github.com/Whealinghold/economy-sim/internal/steady"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file holds the diagnostics of SolveSteadyState that are not part of the
// warm-start table (spec-viability.md 9): the rows of the report, their canonical
// order, and the pass of each programme that fills them. The table made from the
// solves is viability_table.go's, the solves and the rest of the exported surface
// viability.go's.

// RegionCommodity names a commodity in a region.
type RegionCommodity struct {
	// Region is the ID of the region.
	Region string
	// Commodity is the ID of the commodity.
	Commodity string
}

// RegionRecipe names a recipe in a region.
type RegionRecipe struct {
	// Region is the ID of the region.
	Region string
	// Recipe is the ID of the recipe.
	Recipe string
}

// PriceRecord is a seed price that left the reference band: the price before
// the band acted and the seed after it.
type PriceRecord struct {
	// Region and Commodity name the seed.
	Region, Commodity string
	// Price is the rounded price before the band, after the resolution floor and
	// MaxPrice.
	Price int64
	// Seed is the price in the table.
	Seed int64
}

// CostRecord is a seed price outside the cost band at the seed prices.
type CostRecord struct {
	// Region and Commodity name the seed.
	Region, Commodity string
	// Seed is the price in the table.
	Seed int64
	// UnitCost is the full unit cost of the commodity (spec-viability.md 8.2.3).
	UnitCost int64
}

// DriftRow is the drift of a seed from the effective reference price, in basis
// points of the reference price, negative below it.
type DriftRow struct {
	// Region and Commodity name the seed.
	Region, Commodity string
	// DriftBP is (seed - reference) * 10 000 / reference, toward zero.
	DriftBP int64
}

// ResidualRow is the residual of an active activity, in basis points.
type ResidualRow struct {
	// Region and Recipe name the activity.
	Region, Recipe string
	// BP is |value added - wage * duration| * 10 000 / (wage * duration), rounded down.
	BP int64
}

// CensusRow is a line of the balanced census (spec-viability.md 5.2). A group
// row has a Role; a recipe row has a Recipe and no Role.
type CensusRow struct {
	// Region is the ID of the region.
	Region string
	// Role is the ID of the role of a group row, empty for a recipe row.
	Role string
	// Recipe is the ID of the recipe of a recipe row, empty for a group row.
	Recipe string
	// Households is the configured households of the group, or of all the groups
	// that list the recipe.
	Households int64
	// Balanced is the households the steady state needs: the apportionment of
	// the region's households for a group, the rounded-up need for a recipe.
	Balanced int64
	// Imbalance is Households less Balanced.
	Imbalance int64
	// Ambiguous marks a recipe on the lists of several roles of the region.
	Ambiguous bool
	// Short marks a group whose job shares were scaled down to 10 000.
	Short bool
	// Unlisted marks an active recipe that no group of the region lists.
	Unlisted bool
	// IncomeBP is the static income per household of a group at the solved flows
	// and prices, in basis points of the group's upkeep cost: w * labour * 10 000 /
	// (Households * U), with w the starting wage, labour the sum of duration times
	// flow of the group's recipes at lambda 1 and U the upkeep cost of a household
	// at the seed prices, rounded half to even. It is a report field: no table and
	// no census figure reads it. Valid only when HasIncome.
	IncomeBP int64
	// HasIncome is false for a recipe row, for a group with no households, no
	// upkeep cost or no active recipe, for a group with an active recipe on the lists of several roles
	// (its flow is not the group's alone), for a region with no wage or a
	// programme with no labour share (a tight capacity row, FR-VIA-012), and when
	// the figure does not fit int64.
	HasIncome bool
}

// ActiveIn lists the regions in which a recipe is active in the pooled solution.
type ActiveIn struct {
	// Recipe is the ID of the recipe.
	Recipe string
	// Regions are the IDs of the regions, ascending.
	Regions []string
}

// PairRatio is the pooled relative price of two commodities against the range
// of their autarky ratios (spec-viability.md 9.2).
type PairRatio struct {
	// A and B are the commodities; the ratio is price(A) / price(B).
	A, B string
	// PooledBP, LowBP and HighBP are the pooled ratio and the lowest and highest
	// autarky ratio, in basis points, rounded down.
	PooledBP, LowBP, HighBP int64
	// Outside is ratio_outside_autarky: the pooled ratio is outside [low, high],
	// compared exactly.
	Outside bool
}

// Advantage is the comparative advantage of a pooled solution (FR-VIA-045); it
// is empty without WithFreeTrade.
type Advantage struct {
	// Active lists, by recipe, the regions in which it is active.
	Active []ActiveIn
	// Ratios lists the pairs of basket commodities priced in the pooled
	// programme and in at least one autarky programme.
	Ratios []PairRatio
}

// RegionSurplus is the surplus factor of a region in autarky and, with
// WithFreeTrade, the pooled one.
type RegionSurplus struct {
	// Region is the ID of the region.
	Region string
	// AutarkyBP is SurplusBP of the region alone.
	AutarkyBP int64
	// PooledBP is SurplusBP of the pooled programme, when HasPooled.
	PooledBP int64
	// HasPooled says the pooled programme was solved.
	HasPooled bool
}

// RoleWage is the role-economics check of spec-welfare-and-wages.md 7: the
// seed wage of the region against the wage the role's target ratio needs. It is
// a report and never an error.
type RoleWage struct {
	// Region and Role name the group.
	Region, Role string
	// Wage is the seed wage of the region.
	Wage int64
	// Target is TargetRatio * ReferenceBasket * TicksPerYear / (10 000 * ExpectedWorkTicks).
	Target int64
}

// Report holds the diagnostics of a solve (spec-viability.md 10.3), every list
// in ascending region then commodity or recipe order and no map. The fields a
// module owns (Headroom, Rent, FacilityDecay, SurplusWantsBP) are appended with
// the module.
type Report struct {
	// Degenerate and AlternativeOptima say a programme had a basic variable at 0
	// or a non-basic variable at a zero reduced cost (FR-VIA-019).
	Degenerate, AlternativeOptima bool
	// UnpricedCommodities are the rows priced by a rule other than the dual: a
	// free good (dual 0), and the basket goods of a programme that has none.
	UnpricedCommodities []RegionCommodity
	// UnpricedWages are the regions with no starting wage (FR-VIA-050).
	UnpricedWages []string
	// Floored are the seeds raised to the resolution floor or into the band.
	Floored []RegionCommodity
	// OutsideReferenceBand are the seeds the reference band changed.
	OutsideReferenceBand []PriceRecord
	// SeedBelowCostFloor and SeedAboveCostCeiling are the seeds outside the cost
	// band at the seed prices (FR-VIA-027).
	SeedBelowCostFloor, SeedAboveCostCeiling []CostRecord
	// Drift is the drift of each seed from its reference price (FR-VIA-025).
	Drift []DriftRow
	// ResidualBP is the largest residual, and ResidualOver the activities above
	// the ResidualWarnBP option (6.5).
	ResidualBP   int64
	ResidualOver []ResidualRow
	// Census is the balanced census (FR-VIA-028); it changes no table.
	Census []CensusRow
	// CapacityTight are the activities whose capacity row is tight.
	CapacityTight []RegionRecipe
	// Advantage is the comparative advantage of the pooled solution.
	Advantage Advantage
	// CapShortened are the stock covers or reserves cut to the shelf life (8.4.2).
	CapShortened []RegionCommodity
	// NominalSeed are the regions whose table came from the nominal programme.
	NominalSeed []string
	// RegionSurplusBP is the surplus factor of each region with a programme.
	RegionSurplusBP []RegionSurplus
	// WageBelowTarget lists the roles whose seed wage is below the wage their
	// target ratio needs.
	WageBelowTarget []RoleWage
	// SurplusWantsBP is lambda* of the programme with the wants (the lowest of the
	// programmes solved with them), in basis points, when HasSurplusWants: the
	// demand module is on and a programme with the wants was feasible
	// (spec-viability.md 3.2, 10.3).
	SurplusWantsBP  int64
	HasSurplusWants bool
}

// normalise puts every list of the report in ascending region then commodity or
// recipe order and drops repeats, so the order never depends on the order the
// groups and programmes were visited in.
func (r *Report) normalise() {
	r.UnpricedCommodities = sortedUnique(r.UnpricedCommodities, func(a, b RegionCommodity) int { return cmpPair(a.Region, a.Commodity, b.Region, b.Commodity) })
	r.UnpricedWages = sortedUnique(r.UnpricedWages, cmpString)
	r.Floored = sortedUnique(r.Floored, func(a, b RegionCommodity) int { return cmpPair(a.Region, a.Commodity, b.Region, b.Commodity) })
	r.OutsideReferenceBand = sortedUnique(r.OutsideReferenceBand, func(a, b PriceRecord) int { return cmpPair(a.Region, a.Commodity, b.Region, b.Commodity) })
	r.SeedBelowCostFloor = sortedUnique(r.SeedBelowCostFloor, func(a, b CostRecord) int { return cmpPair(a.Region, a.Commodity, b.Region, b.Commodity) })
	r.SeedAboveCostCeiling = sortedUnique(r.SeedAboveCostCeiling, func(a, b CostRecord) int { return cmpPair(a.Region, a.Commodity, b.Region, b.Commodity) })
	r.Drift = sortedUnique(r.Drift, func(a, b DriftRow) int { return cmpPair(a.Region, a.Commodity, b.Region, b.Commodity) })
	r.ResidualOver = sortedUnique(r.ResidualOver, func(a, b ResidualRow) int { return cmpPair(a.Region, a.Recipe, b.Region, b.Recipe) })
	r.CapacityTight = sortedUnique(r.CapacityTight, func(a, b RegionRecipe) int { return cmpPair(a.Region, a.Recipe, b.Region, b.Recipe) })
	r.CapShortened = sortedUnique(r.CapShortened, func(a, b RegionCommodity) int { return cmpPair(a.Region, a.Commodity, b.Region, b.Commodity) })
	r.NominalSeed = sortedUnique(r.NominalSeed, cmpString)
	r.Census = sortedUnique(r.Census, func(a, b CensusRow) int {
		return cmpPair(a.Region, a.Role+"/"+a.Recipe, b.Region, b.Role+"/"+b.Recipe)
	})
	r.RegionSurplusBP = sortedUnique(r.RegionSurplusBP, func(a, b RegionSurplus) int { return cmpString(a.Region, b.Region) })
	r.WageBelowTarget = sortedUnique(r.WageBelowTarget, func(a, b RoleWage) int { return cmpPair(a.Region, a.Role, b.Region, b.Role) })
}

// cmpString orders two strings.
func cmpString(a, b string) int { return cmp.Compare(a, b) }

// cmpPair orders two (first, second) pairs of strings.
func cmpPair(a1, a2, b1, b2 string) int { return cmp.Or(cmp.Compare(a1, b1), cmp.Compare(a2, b2)) }

// sortedUnique returns s sorted by cmp with equal neighbours dropped, nil when
// empty.
func sortedUnique[T comparable](s []T, order func(a, b T) int) []T {
	if len(s) == 0 {
		return nil
	}
	slices.SortStableFunc(s, order)
	return slices.Compact(s)
}

// report adds the diagnostics of the region that are not part of the table: the
// tight capacity rows, the residual, the seeds outside the cost band, the
// census and the roles whose wage is below their target.
func (rc *regionCtx) report() {
	for i, a := range rc.acts {
		if rc.p.out.CapTight[rc.p.base[rc.k]+i] {
			rc.s.rep.CapacityTight = append(rc.s.rep.CapacityTight, RegionRecipe{rc.id, rc.s.m.c.Recipes[a.recipe].ID})
		}
	}
	rc.residual()
	rc.costBand()
	rc.census()
	rc.wageTargets()
}

// residual recomputes the value added of each activity active at the solver's
// prices at the integer seeds and compares it with the wage times the duration
// plus the scaled capacity rent, in basis points (6.5, FR-VIA-025). An activity
// of a region with no wage has nothing to compare with.
func (rc *regionCtx) residual() {
	rep := &rc.s.rep
	for i, a := range rc.acts {
		at := rc.p.base[rc.k] + i
		wd, ok := checked.Mul(rc.wage, a.act.Duration)
		if !rc.p.pr.Active[at] || rc.wage == 0 || !ok {
			continue
		}
		target := steady.Rat(wd, 1).Add(rc.p.pr.Rent[at].Times(rc.scale))
		dev, ok := a.act.ValueAdded(rc.price).Add(target.Mul(-1, 1)).Abs().Quo(target)
		if !ok {
			continue
		}
		bp, fits := dev.Mul(wholeBP, 1).Floor(0, maxInt64)
		if !fits {
			bp = maxInt64
		}
		rep.ResidualBP = max(rep.ResidualBP, bp)
		if bp > rc.s.o.residualWarn {
			rep.ResidualOver = append(rep.ResidualOver, ResidualRow{rc.id, rc.s.m.c.Recipes[a.recipe].ID, bp})
		}
	}
}

// unitCosts returns, by commodity, the least full unit cost over the activities
// of the region that a role there lists and that output it (8.2.3): inputs, tool
// wear and the duration at the region's wage, rounded up, from the seed prices.
// 0 is unknown.
func (rc *regionCtx) unitCosts() []int64 {
	m := rc.s.m
	best := make([]int64, len(rc.price))
	for _, a := range rc.acts {
		if !m.listed[rc.r][a.recipe] || a.count.Starts < 1 {
			continue
		}
		terms, wear := productionTerms(a.act)
		for _, o := range m.rt.row(a.recipe).outputs {
			if o.discard || o.costShare <= 0 {
				continue
			}
			b := production.Basis{
				Share: o.costShare, OutQty: int64(o.quantity), YieldBP: a.count.FactorSum / a.count.Starts,
				Duration: a.act.Duration, Wage: rc.wage,
			}
			if u := production.UnitCost(terms, wear, rc.price, b); u > 0 && (best[o.commodity] == 0 || u < best[o.commodity]) {
				best[o.commodity] = u
			}
		}
	}
	return best
}

// productionTerms converts the inputs and tools of an activity to the terms of
// internal/production, inline at the call (architecture.md 2.9).
func productionTerms(a steady.Activity) ([]production.Term, []production.Wear) {
	var terms []production.Term
	for _, it := range a.Inputs {
		terms = append(terms, production.Term{Commodity: it.Commodity, Quantity: it.Qty})
	}
	var wear []production.Wear
	for _, t := range a.Tools {
		wear = append(wear, production.Wear{Commodity: t.Commodity, WearPerUse: t.Wear, Durability: t.Durability})
	}
	return terms, wear
}

// costBand lists the seeds below UnitCost * CostFloor or above UnitCost *
// CostCeiling at the seed prices (FR-VIA-027, FR-VIA-054).
func (rc *regionCtx) costBand() {
	rep, w := &rc.s.rep, rc.s.m.c.World
	for c, unit := range rc.unitCosts() {
		seed := rc.price[c]
		if unit == 0 || seed == 0 {
			continue
		}
		have, ok := checked.Mul(seed, wholeBP)
		lo, ok1 := checked.Mul(unit, w.CostFloor)
		hi, ok2 := checked.Mul(unit, w.CostCeiling)
		if !ok || !ok1 || !ok2 {
			continue
		}
		rec := CostRecord{rc.id, rc.s.m.c.Commodities[c].ID, seed, unit}
		switch {
		case have < lo:
			rep.SeedBelowCostFloor = append(rep.SeedBelowCostFloor, rec)
		case have > hi:
			rep.SeedAboveCostCeiling = append(rep.SeedAboveCostCeiling, rec)
		}
	}
}

// need returns the households the flows of lambda 1 need of a recipe run by a
// part of its flow: the larger of the labour d * x and the start capacity
// x * TicksPerYear / J, which has no term for a recipe without a capacity row
// (5.2, D132).
func (rc *regionCtx) need(a *viaActivity, flow steady.Frac) steady.Frac {
	need := flow.Mul(a.act.Duration, 1)
	if jd, ok := checked.Mul(a.count.Starts, a.act.Duration); ok && jd < rc.s.m.year {
		if capacity := flow.Mul(rc.s.m.year, a.count.Starts); capacity.Cmp(need) > 0 {
			need = capacity
		}
	}
	return need
}

// census adds the balanced census of the region (5.2, FR-VIA-028): the region's
// households apportioned over its groups in proportion to need, and a row for a
// recipe on the lists of several roles (ambiguous) or of none (unlisted).
func (rc *regionCtx) census() {
	rep := &rc.s.rep
	needs := make([]steady.Frac, len(rc.groups))
	labour := make([]steady.Frac, len(rc.groups))
	var configured int64
	for gi := range rc.groups {
		gc := &rc.groups[gi]
		needs[gi] = steady.Rat(0, 1)
		configured += gc.g.Households
		var capacity steady.Frac
		for n, ar := range gc.active {
			a := &rc.acts[ar.i]
			needs[gi] = needs[gi].Add(ar.share.Mul(a.act.Duration, 1))
			if c := rc.need(a, ar.share); n == 0 || c.Cmp(capacity) > 0 {
				capacity = c
			}
		}
		labour[gi] = needs[gi]
		if len(gc.active) > 0 && capacity.Cmp(needs[gi]) > 0 {
			needs[gi] = capacity
		}
	}
	balanced := steady.Apportion(configured, needs)
	for gi := range rc.groups {
		gc := &rc.groups[gi]
		h, b := gc.g.Households, balanced[gi]
		if slices.Max(balanced) == 0 { // no flow anywhere: nothing to balance against
			b = h
		}
		row := CensusRow{
			Region: rc.id, Role: rc.s.m.c.Roles[gc.role].ID, Households: h, Balanced: b, Imbalance: h - b, Short: gc.short,
		}
		row.IncomeBP, row.HasIncome = rc.income(gc, labour[gi])
		rep.Census = append(rep.Census, row)
	}
	rc.recipeCensus()
}

// income returns the IncomeBP of a group (D372): the starting wage times the
// labour of its recipes at the flows of lambda 1 (the wage already carries
// lambda*), over its households times the upkeep cost of one at the seed prices,
// in basis points, rounded half to even. ok is false where CensusRow.HasIncome
// says so; an overflow of int64 is an absent figure, never a wrapped one
// (architecture.md 3.5).
func (rc *regionCtx) income(gc *groupCtx, labour steady.Frac) (bp int64, ok bool) {
	if rc.wage == 0 || !rc.p.out.HasLabourShare || gc.g.Households == 0 || len(gc.active) == 0 {
		return 0, false
	}
	for _, ar := range gc.active {
		if rc.rolesListing(rc.acts[ar.i].recipe) > 1 {
			return 0, false
		}
	}
	var upkeep int64
	for c, q := range gc.up {
		v, ok := checked.Mul(q, rc.price[c])
		if upkeep, ok = checked.Add(upkeep, v); !ok {
			return 0, false
		}
	}
	num, ok1 := checked.Mul(rc.wage, wholeBP)
	den, ok2 := checked.Mul(gc.g.Households, upkeep)
	if !ok1 || !ok2 || den < 1 {
		return 0, false
	}
	return labour.Mul(num, den).Round(0, maxInt64)
}

// rolesListing counts the groups of the region whose role lists the recipe.
func (rc *regionCtx) rolesListing(recipe RecipeID) int {
	roles := 0
	for gi := range rc.groups {
		if slices.Contains(rc.groups[gi].row.recipes, recipe) {
			roles++
		}
	}
	return roles
}

// recipeCensus adds the rows of the active recipes that are on the lists of
// several roles of the region or of none.
func (rc *regionCtx) recipeCensus() {
	m := rc.s.m
	for i, a := range rc.acts {
		if !rc.activeAt(i) {
			continue
		}
		roles := rc.rolesListing(a.recipe)
		if roles == 1 {
			continue
		}
		h := rc.listing[a.recipe]
		b, _ := rc.need(&rc.acts[i], rc.flow(i)).Ceil(0, maxInt64)
		rc.s.rep.Census = append(rc.s.rep.Census, CensusRow{
			Region: rc.id, Recipe: m.c.Recipes[a.recipe].ID, Households: h, Balanced: b, Imbalance: h - b,
			Ambiguous: roles > 1, Unlisted: roles == 0,
		})
	}
}

// wageTargets lists the roles whose seed wage is below the wage their target
// ratio needs, TargetRatio * B_r * TicksPerYear / (10 000 * ExpectedWorkTicks)
// (WEL 3.1, FR-WEL-031, 038): a report, never an error.
func (rc *regionCtx) wageTargets() {
	if rc.wage == 0 {
		return
	}
	m := rc.s.m
	for gi := range rc.groups {
		ro := &m.c.Roles[rc.groups[gi].role]
		target, ok := welfare.Wage(ro.TargetRatio, int64(m.c.Regions[rc.r].ReferenceBasket), m.year, ro.ExpectedWorkTicks)
		if ok && rc.wage < target {
			rc.s.rep.WageBelowTarget = append(rc.s.rep.WageBelowTarget, RoleWage{rc.id, ro.ID, rc.wage, target})
		}
	}
}

// surpluses fills RegionSurplusBP: the surplus factor of each region with a
// programme in autarky and, with free trade, the pooled one.
func (s *viaSolve) surpluses() {
	row := make(map[RegionID]*RegionSurplus) // a lookup only; the rows are emitted in region order
	for _, p := range append(slices.Clone(s.progs), s.aux...) {
		for _, r := range p.regions {
			if row[r] == nil {
				row[r] = &RegionSurplus{Region: s.m.c.Regions[r].ID}
			}
			if len(p.regions) > 1 || s.o.freeTrade && p == s.progs[0] {
				row[r].PooledBP, row[r].HasPooled = p.out.SurplusBP, true
			} else {
				row[r].AutarkyBP = p.out.SurplusBP
			}
		}
	}
	for _, r := range s.m.withHouseholds {
		if row[r] != nil {
			s.rep.RegionSurplusBP = append(s.rep.RegionSurplusBP, *row[r])
		}
	}
}

// surplusWants fills SurplusWantsBP: the lowest surplus factor of the programmes
// solved with the wants.
func (s *viaSolve) surplusWants() {
	for _, p := range s.progs {
		if p.wants && (!s.rep.HasSurplusWants || p.out.SurplusBP < s.rep.SurplusWantsBP) {
			s.rep.SurplusWantsBP, s.rep.HasSurplusWants = p.out.SurplusBP, true
		}
	}
}

// advantage fills the comparative advantage of the pooled programme (9.2,
// FR-VIA-045): the regions in which each recipe is active, and for each pair of
// basket commodities priced in the pooled programme and in some autarky
// programme the pooled ratio against the range of the autarky ratios.
func (s *viaSolve) advantage() {
	if !s.o.freeTrade || len(s.progs) == 0 {
		return
	}
	p := s.progs[0]
	for k := range s.m.c.Recipes {
		if regions := s.activeRegions(p, RecipeID(k)); len(regions) > 0 {
			s.rep.Advantage.Active = append(s.rep.Advantage.Active, ActiveIn{Recipe: s.m.c.Recipes[k].ID, Regions: regions})
		}
	}
	basket := s.m.upkeepGoods()
	for x, a := range basket {
		for _, b := range basket[x+1:] {
			if pr, ok := s.pairRatio(p, a, b); ok {
				s.rep.Advantage.Ratios = append(s.rep.Advantage.Ratios, pr)
			}
		}
	}
}

// activeRegions returns the IDs of the regions of the pooled programme in which
// the recipe has a positive flow.
func (s *viaSolve) activeRegions(p *viaProgramme, recipe RecipeID) []string {
	var regions []string
	for ri, r := range p.regions {
		for i, a := range s.m.acts[r] {
			if a.recipe == recipe && p.out.Active[p.base[ri]+i] {
				regions = append(regions, s.m.c.Regions[r].ID)
			}
		}
	}
	return regions
}

// pairRatio compares the pooled ratio price(a) / price(b) with the autarky
// ratios of the regions that price both, exactly; ok is false when the pooled
// programme or every autarky programme leaves one of them unpriced.
func (s *viaSolve) pairRatio(p *viaProgramme, a, b int) (PairRatio, bool) {
	pooled, ok := ratioOf(p.pr, a, b)
	low, high, found := s.autarkyRange(a, b)
	if !ok || !found {
		return PairRatio{}, false
	}
	pr := PairRatio{A: s.m.c.Commodities[a].ID, B: s.m.c.Commodities[b].ID, Outside: pooled.Cmp(low) < 0 || pooled.Cmp(high) > 0}
	var ok1, ok2, ok3 bool
	pr.PooledBP, ok1 = pooled.Mul(wholeBP, 1).Floor(0, maxInt64)
	pr.LowBP, ok2 = low.Mul(wholeBP, 1).Floor(0, maxInt64)
	pr.HighBP, ok3 = high.Mul(wholeBP, 1).Floor(0, maxInt64)
	return pr, ok1 && ok2 && ok3
}

// autarkyRange returns the lowest and highest price(a) / price(b) over the
// autarky programmes that price both.
func (s *viaSolve) autarkyRange(a, b int) (low, high steady.Frac, found bool) {
	for _, q := range s.aux {
		r, ok := ratioOf(q.out, a, b)
		if !ok {
			continue
		}
		if !found || r.Cmp(low) < 0 {
			low = r
		}
		if !found || r.Cmp(high) > 0 {
			high = r
		}
		found = true
	}
	return low, high, found
}

// ratioOf returns price(a) / price(b) of an outcome when both are positive.
func ratioOf(o steady.Outcome, a, b int) (steady.Frac, bool) {
	if o.Price[a].Sign() == 0 || o.Price[b].Sign() == 0 {
		return steady.Frac{}, false
	}
	return o.Price[a].Quo(o.Price[b])
}
