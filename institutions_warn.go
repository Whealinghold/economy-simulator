package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/institution"
	"github.com/Whealinghold/economy-sim/internal/production"
)

// This file is the warnings of the institutions module (spec-institutions.md 10.2; a
// continuation file of institutions_validate.go, architecture.md 2.6). None fails
// validation. Each is raised once, at construction, with the path of the body, right,
// control or rate limit by its ID. The functions read the configuration as given and
// never panic on one that does not validate: a name that is not declared is skipped.

// orgWarner holds what the warnings read: the module, the scope expansion and the
// households of each (region, role) at the start.
type orgWarner struct {
	c      *Config
	lk     lookup
	m      *Institutions
	scope  *orgScoper
	groups map[[2]string]int64 // households by (region, role); a lookup only
	seeds  map[string]seedMemo // the seed table of a region, read once; a lookup only
	floors map[[2]string]int64 // the highest cost floor by (region, commodity), read once; a lookup only
}

// institutionsWarnings returns the warnings of spec-institutions.md 10.2: it reports
// nothing when the module is off.
func institutionsWarnings(c *Config, lk lookup) []Warning {
	m := c.Modules.Institutions
	if m == nil {
		return nil
	}
	w := &orgWarner{c: c, lk: lk, m: m, scope: orgScoperOf(c, outsideNodeSet(c)), groups: make(map[[2]string]int64, len(c.Groups)),
		seeds: map[string]seedMemo{}, floors: map[[2]string]int64{}}
	for _, g := range c.Groups {
		key := [2]string{g.Region, g.Role}
		w.groups[key] = satAdd(w.groups[key], g.Households)
	}
	out := w.bodyWarnings()
	out = append(out, w.rightWarnings()...)
	out = append(out, w.controlWarnings()...)
	return append(out, w.rateLimitWarnings()...)
}

// orgWarning is the warning of a rule about the entity of a kind with an ID.
func orgWarning(rule, kind, id string) Warning {
	return Warning{Path: []string{"modules", "institutions", kind, id}, Rule: rule}
}

// households is the households of the roles that start in the region.
func (w *orgWarner) households(region string, roles []string) int64 {
	var sum int64
	for _, r := range roles {
		sum = satAdd(sum, w.groups[[2]string{region, r}])
	}
	return sum
}

// bodyWarnings are body_empty, places_below_start and body_needs_unmet_at_start.
func (w *orgWarner) bodyWarnings() []Warning {
	var out []Warning
	census, hasCensus := w.settlementCensus()
	for i := range w.m.Bodies {
		b := &w.m.Bodies[i]
		scope := w.scope.regions(b.Regions, b.Jurisdictions)
		var most int64
		for _, r := range scope {
			most = max(most, w.households(r, b.Roles))
		}
		out = orgIf(out, len(scope) > 0 && most == 0, "body_empty", "body", b.ID)
		out = orgIf(out, b.Places > 0 && most > b.Places, "places_below_start", "body", b.ID)
		unmet := hasCensus && b.Needs != "" && len(scope) > 0 && !census.carries(scope, b.Needs)
		out = orgIf(out, unmet, "body_needs_unmet_at_start", "body", b.ID)
	}
	return out
}

// orgIf appends the warning of a rule when its condition holds.
func orgIf(out []Warning, cond bool, rule, kind, id string) []Warning {
	if cond {
		return append(out, orgWarning(rule, kind, id))
	}
	return out
}

// orgCensus is the class each region starts in, from the census of the settlements
// module, and the flags that class carries.
type orgCensus struct {
	flags map[string]map[string]bool // region to the flags its class carries at the start
}

// carries reports whether the class of some region of the scope carries the flag.
func (o orgCensus) carries(scope []string, flag string) bool {
	return slices.ContainsFunc(scope, func(r string) bool { return o.flags[r][flag] })
}

// settlementCensus reads the class each region starts in from SET's census of the
// groups (spec-settlements.md 4.5.1); false when the module is off or has no census.
func (w *orgWarner) settlementCensus() (orgCensus, bool) {
	cm := w.c.Modules.Settlements.canonical()
	if cm == nil || len(cm.Classes) == 0 {
		return orgCensus{}, false
	}
	s, ok := newSettlementCensus(w.c, cm)
	if !ok {
		return orgCensus{}, false
	}
	out := orgCensus{flags: make(map[string]map[string]bool, len(s.regions))}
	for i := range s.regions {
		set := map[string]bool{}
		for _, f := range cm.Classes[s.class[i]].Flags {
			set[f] = true
		}
		out.flags[s.regions[i].ID] = set
	}
	return out, true
}

// rightWarnings are right_unreachable, right_over_upkeep and train_right_no_training.
func (w *orgWarner) rightWarnings() []Warning {
	var out []Warning
	for i := range w.m.Rights {
		r := &w.m.Rights[i]
		ro := w.lk.role[r.Subject]
		out = orgIf(out, w.unreachable(r), "right_unreachable", "right", r.ID)
		out = orgIf(out, w.overUpkeep(r), "right_over_upkeep", "right", r.ID)
		out = orgIf(out, r.Activity == ActivityTrain && ro != nil && ro.TrainingTicks == 0, "train_right_no_training", "right", r.ID)
	}
	return out
}

// scopeOf is the regions of a scope key.
func (w *orgWarner) scopeOf(region, jurisdiction string) []string {
	return w.scope.regions(orgNonEmpty(region), orgNonEmpty(jurisdiction))
}

// orgNonEmpty is the one-element list of a name, nil for an empty one.
func orgNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

// unreachable is right_unreachable: the holder has no eligible household in a region of
// the right's scope, so the activity is closed to everyone there (5.4.4).
func (w *orgWarner) unreachable(r *Right) bool {
	i := slices.IndexFunc(w.m.Bodies, func(b Body) bool { return b.ID == r.Body })
	if r.Body == "" || i < 0 {
		return false
	}
	b := &w.m.Bodies[i]
	inBody := w.scope.regions(b.Regions, b.Jurisdictions)
	for _, region := range w.scopeOf(r.Region, r.Jurisdiction) {
		if !slices.Contains(inBody, region) || w.households(region, b.Roles) == 0 {
			return true
		}
	}
	return false
}

// overUpkeep is right_over_upkeep (5.5.3, ORG-A6): a buy right on a commodity that is in
// the effective upkeep of a role with a group in the scope, a sell right whose subject
// is the only commodity a role in the scope produces for sale, or a produce right on
// the only recipe that outputs a commodity of such a role's upkeep.
func (w *orgWarner) overUpkeep(r *Right) bool {
	regions := w.scopeOf(r.Region, r.Jurisdiction)
	for i := range w.c.Roles {
		ro := &w.c.Roles[i]
		inScope := slices.ContainsFunc(regions, func(g string) bool { return w.groups[[2]string{g, ro.ID}] > 0 })
		if inScope && w.bindsRole(r, ro) {
			return true
		}
	}
	return false
}

// bindsRole reports whether the right over upkeep or a sole output touches the role.
func (w *orgWarner) bindsRole(r *Right, ro *Role) bool {
	upkeep := effectiveUpkeep(w.c, ro)
	switch r.Activity {
	case ActivityBuy:
		return slices.ContainsFunc(upkeep, func(it Item) bool { return it.Commodity == r.Subject })
	case ActivitySell:
		outs := w.outputsOf(ro)
		return len(outs) == 1 && outs[0] == r.Subject
	case ActivityProduce:
		return slices.ContainsFunc(upkeep, func(it Item) bool {
			src := w.sourcesOf(it.Commodity)
			return len(src) == 1 && src[0] == r.Subject
		})
	}
	return false
}

// outputsOf is the distinct commodities that the recipes on a role's list output, apart
// from the discard outputs, sorted.
func (w *orgWarner) outputsOf(ro *Role) []string {
	var out []string
	for _, id := range ro.Recipes {
		if rc := w.lk.recipe[id]; rc != nil {
			for _, o := range rc.Outputs {
				if !o.Discard {
					out = append(out, o.Commodity)
				}
			}
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// sourcesOf is the IDs of the recipes that output the commodity, apart from a discard
// output, sorted.
func (w *orgWarner) sourcesOf(commodity string) []string {
	var out []string
	for i := range w.c.Recipes {
		rc := &w.c.Recipes[i]
		if slices.ContainsFunc(rc.Outputs, func(o Output) bool { return !o.Discard && o.Commodity == commodity }) {
			out = append(out, rc.ID)
		}
	}
	slices.Sort(out)
	return out
}

// controlWarnings are cap_below_band, floor_above_band, cap_below_seed_cost and the
// control half of control_inert.
func (w *orgWarner) controlWarnings() []Warning {
	var out []Warning
	for i := range w.m.Controls {
		c := &w.m.Controls[i]
		if c.CapBP == 0 && c.FloorBP == 0 {
			out = append(out, orgWarning("control_inert", "control", c.ID))
			continue
		}
		cm := w.lk.commodity[c.Commodity]
		if cm == nil {
			continue
		}
		below, above, seed := w.levels(c, cm)
		if below {
			out = append(out, orgWarning("cap_below_band", "control", c.ID))
		}
		if above {
			out = append(out, orgWarning("floor_above_band", "control", c.ID))
		}
		if seed {
			out = append(out, orgWarning("cap_below_seed_cost", "control", c.ID))
		}
	}
	return out
}

// levels reads the range of a control in each region of its scope against the belief
// band (RefBandLow and RefBandHigh of the reference price) and the cost of a seed.
func (w *orgWarner) levels(c *Control, cm *Commodity) (capBelow, floorAbove, belowSeed bool) {
	for _, region := range w.scopeOf(c.Region, c.Jurisdiction) {
		ref := int64(w.lk.price(region, cm))
		capPrice, floorPrice, ok := institution.Range(ref, c.CapBP, c.FloorBP, MaxPrice)
		if !ok {
			continue
		}
		below, above := w.againstBand(ref, capPrice, floorPrice)
		capBelow, floorAbove = capBelow || below, floorAbove || above
		belowSeed = belowSeed || (capPrice > 0 && w.capBelowSeedCost(region, c.Commodity, capPrice))
	}
	return capBelow, floorAbove, belowSeed
}

// againstBand reports whether a cap is below the lowest belief of the reference band
// (RefBandLow basis points of the reference price) and a floor above the highest.
func (w *orgWarner) againstBand(ref, capPrice, floorPrice int64) (capBelow, floorAbove bool) {
	low, lowOK := checked.MulDiv(ref, w.c.World.RefBandLow, wholeBP)
	high, highOK := checked.MulDiv(ref, w.c.World.RefBandHigh, wholeBP)
	return lowOK && capPrice > 0 && capPrice < low, highOK && floorPrice > 0 && floorPrice > high
}

// seedTable is the warm start's seed prices of a region by commodity index and its
// wage, as internal/production reads them.
type seedTable struct {
	index  map[string]int
	prices []int64
	wage   int64
}

// seedMemo is the seed table of a region and whether the warm start gave one.
type seedMemo struct {
	t  seedTable
	ok bool
}

// seedsOf reads the seed table of a region from the warm start, once per region (a
// control keyed by a jurisdiction asks for each region it covers); ok is false without
// a warm start.
func (w *orgWarner) seedsOf(region string) (t seedTable, ok bool) {
	if m, done := w.seeds[region]; done {
		return m.t, m.ok
	}
	t, ok = w.readSeeds(region)
	w.seeds[region] = seedMemo{t, ok}
	return t, ok
}

// readSeeds builds the seed table of a region from the warm start.
func (w *orgWarner) readSeeds(region string) (t seedTable, ok bool) {
	ws := w.c.WarmStart
	if ws == nil {
		return t, false
	}
	t = seedTable{index: make(map[string]int, len(w.c.Commodities)), prices: make([]int64, len(w.c.Commodities))}
	for i := range w.c.Commodities {
		t.index[w.c.Commodities[i].ID] = i
	}
	for _, s := range ws.Prices {
		if i, known := t.index[s.Commodity]; known && s.Region == region {
			t.prices[i] = int64(s.Price)
		}
	}
	for _, s := range ws.Wages {
		if s.Region == region {
			t.wage = int64(s.Wage)
		}
	}
	return t, true
}

// capBelowSeedCost is cap_below_seed_cost for one region: with a warm start, a cap
// below CostFloor of the full unit cost, at the warm start's prices and wage, of a
// recipe of a role with a group in the region that makes the commodity (spec-production.md 9.4, spec-viability.md 8.2.3).
func (w *orgWarner) capBelowSeedCost(region, commodity string, capPrice int64) bool {
	return capPrice < w.seedFloor(region, commodity)
}

// seedFloor is the highest CostFloor of the unit cost, at the seed prices, of a recipe of
// a role with a group in the region that makes the commodity: a cap below it is below
// the floor of some recipe. It is 0 without a warm start or such a recipe, and it is
// worked out once per (region, commodity), so a jurisdiction-wide control costs one pass
// over the roles and recipes however many regions and controls name the commodity.
func (w *orgWarner) seedFloor(region, commodity string) int64 {
	key := [2]string{region, commodity}
	if f, done := w.floors[key]; done {
		return f
	}
	var most int64
	if seeds, ok := w.seedsOf(region); ok {
		for i := range w.c.Roles {
			ro := &w.c.Roles[i]
			if w.groups[[2]string{region, ro.ID}] <= 0 {
				continue
			}
			for _, id := range ro.Recipes {
				rc := w.lk.recipe[id]
				if rc == nil {
					continue
				}
				unit := w.seedUnitCost(rc, commodity, seeds)
				if floor, ok := checked.MulDiv(unit, w.c.World.CostFloor, wholeBP); unit > 0 && ok {
					most = max(most, floor)
				}
			}
		}
	}
	w.floors[key] = most
	return most
}

// seedUnitCost is the full unit cost of the commodity from the recipe at the seed
// prices and wage, at the full yield, 0 when the recipe does not output it or a price
// is unknown.
func (w *orgWarner) seedUnitCost(rc *Recipe, commodity string, seeds seedTable) int64 {
	for _, o := range rc.Outputs {
		if o.Discard || o.Commodity != commodity || o.CostShare <= 0 {
			continue
		}
		var terms []production.Term
		for _, it := range rc.Inputs {
			if _, known := seeds.index[it.Commodity]; !known {
				return 0
			}
			terms = append(terms, production.Term{Commodity: seeds.index[it.Commodity], Quantity: int64(it.Quantity)})
		}
		var wear []production.Wear
		for _, t := range rc.Requires {
			if _, known := seeds.index[t.Commodity]; !known {
				return 0
			}
			if cm := w.lk.commodity[t.Commodity]; cm != nil {
				wear = append(wear, production.Wear{Commodity: seeds.index[t.Commodity], WearPerUse: t.WearPerUse, Durability: cm.Durability})
			}
		}
		b := production.Basis{Share: o.CostShare, OutQty: int64(o.Quantity), YieldBP: wholeBP, Duration: rc.Duration, Wage: seeds.wage}
		return production.UnitCost(terms, wear, seeds.prices, b)
	}
	return 0
}

// rateLimitWarnings are rate_limit_below_floor and the rate limit half of
// control_inert.
func (w *orgWarner) rateLimitWarnings() []Warning {
	var out []Warning
	for i := range w.m.RateLimits {
		l := &w.m.RateLimits[i]
		if l.CapBP >= MaxCreditRateBP {
			out = append(out, orgWarning("control_inert", "rate_limit", l.ID))
		}
		if w.belowEveryLender(l) {
			out = append(out, orgWarning("rate_limit_below_floor", "rate_limit", l.ID))
		}
	}
	return out
}

// belowEveryLender is rate_limit_below_floor: at least one lender has a group in the
// scope and the cap is below the posted RateBP of every one of them, so no loan can
// form (the mirror of rate_above_limits).
func (w *orgWarner) belowEveryLender(l *RateLimit) bool {
	regions := w.scopeOf(l.Region, l.Jurisdiction)
	lenders := 0
	for i := range w.c.Roles {
		ro := &w.c.Roles[i]
		if ro.Credit == nil || ro.Credit.Lend == nil {
			continue
		}
		if !slices.ContainsFunc(regions, func(g string) bool { return w.groups[[2]string{g, ro.ID}] > 0 }) {
			continue
		}
		if ro.Credit.Lend.RateBP <= l.CapBP {
			return false
		}
		lenders++
	}
	return lenders > 0
}
