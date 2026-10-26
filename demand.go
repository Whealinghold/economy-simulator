package economysim

import (
	"cmp"
	"maps"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file is the demand module (spec-demand.md; architecture.md 2.10.1): the
// module's state, its validator and counts, the items it adds to the tradable
// set of a role, its warnings and the names of its IDs. The pure rules of the
// per-tick arithmetic are in internal/household/wants.go. The module has no world
// effect yet: nothing steps demandState, which the hooks of ADR 0010 (the
// per-position methods, the filler and the checker, the agent-lifecycle calls)
// will do when they are admitted. Outside this file the module appears only in
// the catalogue files (config.go, hash.go, bounds.go, defaults.go, errors.go,
// validate.go) and the fan-in call in fanin.go.

// demandState is the state of the module in a world, nil when the module is off
// (spec-demand.md 12). It holds what is derived from the configuration: the
// ladder of each role. The per-agent carries and the choke runs are state and
// are added with the hooks that form and advance them.
type demandState struct {
	// ladder[role] is the wants of the role's ladder, by rank and then by want ID,
	// indexed by RoleID. A role with no taste row of nonzero scale has an empty
	// ladder and no want state (FR-DEM-004).
	ladder [][]rung
}

// rung is one want of a ladder: the want and the taste scale of the role for it
// in basis points.
type rung struct {
	want  WantID
	scale int64
}

// newDemandState builds the state of the module for a canonical, validated
// configuration, or returns nil when the module is off. A reference that does
// not resolve is a programmer error after validation and panics.
func newDemandState(c *Config) *demandState {
	d := c.Modules.Demand
	if d == nil {
		return nil
	}
	st := &demandState{ladder: make([][]rung, len(c.Roles))}
	for _, t := range d.Tastes {
		role, rok := c.roleID(t.Role)
		want, wok := c.wantID(t.Want)
		if !rok || !wok {
			invariant("a taste names an unknown role or want after validation")
		}
		if t.Scale > 0 {
			st.ladder[role] = append(st.ladder[role], rung{want: want, scale: t.Scale})
		}
	}
	for _, l := range st.ladder {
		// Rank, then want ID: a total order, as the IDs are unique (architecture.md
		// 7.6). The want ID order is the order of WantID.
		slices.SortFunc(l, func(a, b rung) int {
			return cmp.Or(cmp.Compare(d.Wants[a.want].Rank, d.Wants[b.want].Rank), cmp.Compare(a.want, b.want))
		})
	}
	return st
}

// canonical returns a deep copy of d in canonical order (spec-config.md 3.4,
// spec-demand.md 3.2.2): wants by ID, items by commodity, tastes by role then
// want, region tastes by region then want. A nil module stays nil, and a module
// that is on stays non-nil, so presence survives the copy.
func (d *Demand) canonical() *Demand {
	if d == nil {
		return nil
	}
	out := &Demand{
		Wants: sortedBy(d.Wants, func(a, b Want) int { return cmp.Compare(a.ID, b.ID) }),
		Tastes: sortedBy(d.Tastes, func(a, b Taste) int {
			return cmp.Or(cmp.Compare(a.Role, b.Role), cmp.Compare(a.Want, b.Want))
		}),
		RegionTastes: sortedBy(d.RegionTastes, func(a, b RegionTaste) int {
			return cmp.Or(cmp.Compare(a.Region, b.Region), cmp.Compare(a.Want, b.Want))
		}),
	}
	for i := range out.Wants {
		out.Wants[i].Items = sortedBy(out.Wants[i].Items, func(a, b WantItem) int { return cmp.Compare(a.Commodity, b.Commodity) })
	}
	return out
}

// wantID returns the typed ID of a want, its index in the canonical Wants
// (spec-demand.md 3.2.2a). The receiver must already be canonical.
func (c Config) wantID(id string) (WantID, bool) {
	if c.Modules.Demand == nil {
		return 0, false
	}
	i, ok := slices.BinarySearchFunc(c.Modules.Demand.Wants, id, func(e Want, t string) int { return cmp.Compare(e.ID, t) })
	return WantID(i), ok
}

// demandNames returns the string ID of each WantID for the names table, or nil
// when the module is off.
func demandNames(c *Config) []string {
	d := c.Modules.Demand
	if d == nil {
		return nil
	}
	names := make([]string, len(d.Wants))
	for i := range d.Wants {
		names[i] = d.Wants[i].ID
	}
	return names
}

// demandIndex holds the tables of a configuration by ID, built without
// validating (a repeated ID keeps its first row), for the validator, the
// warnings and the tradable set. It is nil when the module is off. The pointers
// point into the caller's slices, so an index is read-only.
type demandIndex struct {
	want   map[string]*Want
	tastes map[string][]*Taste // by role, in the order given
	region map[[2]string]int64 // the scale of the first region taste by (region, want)
	tasted map[string]bool     // the wants with a taste row
	// regionMax is the largest scale of any region taste of a want that is inside
	// its bound; a row outside it has its own error.
	regionMax map[string]int64
}

// newDemandIndex indexes the module of c, or returns nil when it is off.
func newDemandIndex(c *Config) *demandIndex {
	d := c.Modules.Demand
	if d == nil {
		return nil
	}
	ix := &demandIndex{
		want:      make(map[string]*Want, len(d.Wants)),
		tastes:    make(map[string][]*Taste),
		region:    make(map[[2]string]int64, len(d.RegionTastes)),
		tasted:    make(map[string]bool, len(d.Tastes)),
		regionMax: make(map[string]int64),
	}
	for i := range d.Wants {
		if w := &d.Wants[i]; ix.want[w.ID] == nil {
			ix.want[w.ID] = w
		}
	}
	for i := range d.Tastes {
		t := &d.Tastes[i]
		ix.tastes[t.Role] = append(ix.tastes[t.Role], t)
		ix.tasted[t.Want] = true
	}
	for _, r := range d.RegionTastes {
		key := [2]string{r.Region, r.Want}
		if _, seen := ix.region[key]; !seen {
			ix.region[key] = r.Scale
		}
		if r.Scale >= 0 && r.Scale <= MaxTasteScale {
			ix.regionMax[r.Want] = max(ix.regionMax[r.Want], r.Scale)
		}
	}
	return ix
}

// regionScale is the scale of a want in a region in basis points: its row, or
// 10 000 when it has none.
func (ix *demandIndex) regionScale(region, want string) int64 {
	if s, ok := ix.region[[2]string{region, want}]; ok {
		return s
	}
	return wholeBP
}

// demandItems adds to set the commodities of the items of every want in the
// ladder of the role (spec-demand.md 3.2.3, FR-DEM-006): the wants with a taste
// row of nonzero scale. A want that is not defined is skipped; validation has
// refused it.
func demandItems(lk lookup, role *Role, set map[string]bool) {
	ix := lk.demand
	for _, t := range ix.tastes[role.ID] {
		w := ix.want[t.Want]
		if t.Scale <= 0 || w == nil {
			continue
		}
		for _, it := range w.Items {
			set[it.Commodity] = true
		}
	}
}

// demandCounts is the top-level part of phase 1 for the module (spec-config.md
// 7.1.6): the wants, then the two taste lists. A count over its bound is
// ErrTooMany with its path (FR-CFG-061, FR-DEM-041).
func demandCounts(d *Demand, over func(n, bound int, p ...string)) {
	if d == nil {
		return
	}
	over(len(d.Wants), MaxWants, "modules", "demand", "wants")
	over(len(d.Tastes), MaxTastes, "modules", "demand", "tastes")
	over(len(d.RegionTastes), MaxRegionTastes, "modules", "demand", "region_tastes")
}

// demandNestedCounts is the nested part of phase 1, walked only when the
// top-level counts hold: the items of each want and the taste rows of each role.
func demandNestedCounts(d *Demand, over func(n, bound int, p ...string)) {
	if d == nil {
		return
	}
	for _, w := range d.Wants {
		over(len(w.Items), MaxWantItems, "modules", "demand", "wants", w.ID, "items")
	}
	perRole := map[string]int{}
	for _, t := range d.Tastes {
		perRole[t.Role]++
	}
	for _, role := range slices.Sorted(maps.Keys(perRole)) {
		over(perRole[role], MaxWantsPerRole, "modules", "demand", "tastes", role)
	}
}

// validateDemand is the module's part of phases 2 and 3 and 5 of spec-config.md
// 7.1.6 (spec-demand.md 10.1): the bounds of each field, the identity and
// references of the three lists, the items of each want and the derived
// products. It runs after phase 1 found no count over a bound, so every list is
// small. It reports nothing when the module is off.
func (v *validator) validateDemand() {
	d := v.c.Modules.Demand
	if d == nil {
		return
	}
	ix := v.lk.demand
	v.demandWants(d)
	seen := map[[2]string]bool{}
	for _, t := range d.Tastes {
		v.demandRow(ix, seen, tasteRow{"tastes", t.Role, "role", t.Want, t.Scale}, v.role, ErrUnknownRole)
	}
	seen = map[[2]string]bool{}
	for _, r := range d.RegionTastes {
		v.demandRow(ix, seen, tasteRow{"region_tastes", r.Region, "region", r.Want, r.Scale}, v.region, ErrUnknownRegion)
	}
	v.demandOverflow(ix)
}

// demandWants checks the fields and the items of every want.
func (v *validator) demandWants(d *Demand) {
	seen := make(map[string]bool, len(d.Wants))
	for _, w := range d.Wants {
		base := []string{"modules", "demand", "wants", w.ID}
		if !validID(w.ID) {
			v.rep.add(ErrInvalidID, path(base, "id"), w.ID, "")
		}
		if seen[w.ID] {
			v.rep.add(ErrDuplicateID, path(base, "id"), w.ID, "")
		}
		seen[w.ID] = true
		v.ranges(base, []fieldRange{
			{"quantity", w.Quantity, 1, perHousehold},
			{"period", w.Period, 1, MaxWantPeriod},
			{"rank", w.Rank, 0, MaxWantRank},
			{"elasticity", w.Elasticity, 0, MaxElasticityBP},
			{"ceiling", w.Ceiling, wholeBP, MaxCeilingBP},
			{"substitution", w.Substitution, 0, MaxSubstitution},
		})
		v.label(base, "label", w.Label)
		v.demandWantItems(base, w.Items)
	}
}

// demandWantItems checks the items of one want: at least one, each commodity known,
// not repeated and not durable, each weight in its bound, and not every weight 0
// (ErrWantItems). The index in a path is the position as written.
func (v *validator) demandWantItems(base []string, items []WantItem) {
	items0 := path(base, "items")
	if len(items) == 0 {
		v.rep.add(ErrWantItems, items0, "0", "1")
		return
	}
	seen := make(map[string]bool, len(items))
	positive := false
	for i, it := range items {
		p := path(base, "items["+itoa(i)+"]")
		switch cm := v.lk.commodity[it.Commodity]; {
		case !v.commodity[it.Commodity]:
			v.rep.add(ErrUnknownCommodity, path(p, "commodity"), it.Commodity, "")
		case cm.Durability > 0:
			v.rep.add(ErrWantDurable, path(p, "commodity"), it.Commodity, "")
		}
		if seen[it.Commodity] {
			v.rep.add(ErrDuplicateID, path(p, "commodity"), it.Commodity, "")
		}
		seen[it.Commodity] = true
		v.inRange(path(p, "weight"), it.Weight, 1, MaxBasisPoints)
		positive = positive || it.Weight > 0
	}
	if !positive {
		v.rep.add(ErrWantItems, items0, "all weights 0", "")
	}
}

// tasteRow is a taste or a region taste as demandRow reads it: the list it is in
// (a path segment), the key it shares with the other kind (a role or a region and
// the name of that key in a path), the want and the scale.
type tasteRow struct {
	list, first, field, want string
	scale                    int64
}

// demandRow checks one taste or region taste row: the first key is known, the
// want is defined, the pair is new, and the scale is in its bound.
func (v *validator) demandRow(ix *demandIndex, seen map[[2]string]bool, r tasteRow, known map[string]bool, unknown error) {
	base := []string{"modules", "demand", r.list, r.first, r.want}
	if !known[r.first] {
		v.rep.add(unknown, path(base, r.field), r.first, "")
	}
	if ix.want[r.want] == nil {
		v.rep.add(ErrUnknownWant, path(base, "want"), r.want, "")
	}
	key := [2]string{r.first, r.want}
	if seen[key] {
		v.rep.add(ErrDuplicateID, base, r.first+"/"+r.want, "")
	}
	seen[key] = true
	v.inRange(path(base, "scale"), r.scale, 0, MaxTasteScale)
}

// demandOverflow is phase 5 for the module (spec-config.md 4.2, spec-demand.md
// 10.1): for each role with a group, the products of roleWantProducts. A product
// whose inputs are outside their bounds is not computed: that rule has spoken.
func (v *validator) demandOverflow(ix *demandIndex) {
	weight := map[string]int64{} // the largest weight of a group of the role
	for _, g := range v.c.Groups {
		if _, wt, _, ok := groupShape(v.c.World, g); ok {
			weight[g.Role] = max(weight[g.Role], wt)
		}
	}
	for _, role := range v.c.Roles {
		if wt := weight[role.ID]; wt > 0 {
			v.roleWantProducts(ix, role.ID, wt)
		}
	}
}

// roleWantProducts checks, for the tastes of one role, the units a household of
// the largest weight can want in a tick (tasteUnits) and the sum over the ladder
// of those units times the dearest item price, at most MaxMoney.
func (v *validator) roleWantProducts(ix *demandIndex, role string, weight int64) {
	var spend int64
	spendOK := true
	for _, t := range ix.tastes[role] {
		units, ok := v.tasteUnits(ix, role, t, weight)
		if !ok {
			continue
		}
		line, lok := checked.Mul(units, v.dearestItem(ix.want[t.Want]))
		sum, sok := checked.Add(spend, line)
		spendOK = spendOK && lok && sok
		spend = sum
	}
	if !spendOK || spend > MaxMoney {
		v.rep.add(ErrOverflow, []string{"modules", "demand", "tastes", role}, overflowOrValue(spend, spendOK), i64(MaxMoney))
	}
}

// tasteUnits is the most units of the want that an agent of the weight wants in a
// tick: weight * Quantity * Ceiling * taste scale * region scale / (Period *
// 10^12), at most MaxQuantity (ErrOverflow at the taste if not). The region scale
// is the largest any region gives the want, and at least 10 000. ok is false when
// the product is not computed because a field is outside its bound (that rule has
// spoken) or because it overflowed (reported here).
func (v *validator) tasteUnits(ix *demandIndex, role string, t *Taste, weight int64) (int64, bool) {
	w := ix.want[t.Want]
	if w == nil || !wantInBounds(w) || t.Scale < 0 || t.Scale > MaxTasteScale {
		return 0, false
	}
	units, ok := wantUnits(w, weight, t.Scale, max(wholeBP, ix.regionMax[w.ID]))
	if !ok || units > MaxQuantity {
		v.rep.add(ErrOverflow, []string{"modules", "demand", "tastes", role, t.Want}, overflowOrValue(units, ok), i64(MaxQuantity))
		return 0, false
	}
	return units, true
}

// overflowOrValue is the Value of an ErrOverflow: the product as text, or
// "overflow" when it did not fit int64.
func overflowOrValue(n int64, ok bool) string {
	if !ok {
		return overflowValue
	}
	return i64(n)
}

// wantInBounds reports whether the fields that the units product reads are inside
// their bounds.
func wantInBounds(w *Want) bool {
	return w.Quantity >= 1 && w.Quantity <= perHousehold && w.Period >= 1 && w.Period <= MaxWantPeriod &&
		w.Ceiling >= wholeBP && w.Ceiling <= MaxCeilingBP
}

// wantUnits is the most units of the want that an agent of the weight wants in a
// tick: weight * Quantity * Ceiling * taste * region / (Period * 10^12), with the
// product held in 128 bits and truncated. ok is false when it does not fit int64.
func wantUnits(w *Want, weight, taste, region int64) (int64, bool) {
	a, aok := checked.Mul(weight*w.Quantity, w.Ceiling) // at most 10^6 * 10^3 * 5 * 10^4
	if !aok {
		return 0, false
	}
	return checked.MulDivDiv(a, taste*region, w.Period, 1_000_000_000_000) // taste * region at most 10^10
}

// dearestItem is the highest price any item of the want can have in any region.
func (v *validator) dearestItem(w *Want) int64 {
	var top int64
	for _, it := range w.Items {
		if cm := v.lk.commodity[it.Commodity]; cm != nil && cm.ReferencePrice >= 1 && cm.ReferencePrice <= MaxPrice {
			top = max(top, int64(max(cm.ReferencePrice, v.dearest[it.Commodity])))
		}
	}
	return top
}

// hasIncomeSource reports whether a role has an income other than its sales and
// its starting money, for the warning class_without_income (spec-demand.md 10.2,
// 7.3). The modules that pay a class (a money-supply grant, a fiscal payout, a
// title, an employer's hire, relief) are not built, so none exempts a role yet:
// each later income module appends its exemption here, in its own change.
func hasIncomeSource(*Config, *Role) bool { return false }

// demandWarnings returns the seven warnings of the module (spec-demand.md 10.2,
// FR-DEM-033), or nil when it is off. Like warnings it reads a configuration as
// given, skips what is not defined and never fails.
func demandWarnings(c *Config, lk lookup) []Warning {
	if lk.demand == nil {
		return nil
	}
	var out []Warning
	add := func(rule string, segs ...string) { out = append(out, Warning{Path: segs, Rule: rule}) }
	demandWantWarnings(c, lk, add)
	for _, r := range c.Modules.Demand.RegionTastes {
		if !lk.demand.tasted[r.Want] {
			add("region_taste_without_taste", "modules", "demand", "region_tastes", r.Region, r.Want)
		}
	}
	for i := range c.Roles {
		demandRoleWarnings(c, lk, &c.Roles[i], add)
	}
	return out
}

// wantSources are the facts about the commodities that the warnings on the items
// of a want read: some recipe makes it (kept, not discarded), some role's recipe
// does, or some role starts with it.
type wantSources struct {
	made, byRole, stocked map[string]bool
}

// keep marks the commodities a recipe makes and keeps.
func keep(into map[string]bool, rc *Recipe) {
	for _, o := range rc.Outputs {
		if !o.Discard {
			into[o.Commodity] = true
		}
	}
}

// newWantSources reads the recipes and the roles of c.
func newWantSources(c *Config, lk lookup) wantSources {
	src := wantSources{made: map[string]bool{}, byRole: map[string]bool{}, stocked: map[string]bool{}}
	for i := range c.Recipes {
		keep(src.made, &c.Recipes[i])
	}
	for i := range c.Roles {
		for _, it := range c.Roles[i].StartStock {
			if it.Quantity > 0 {
				src.stocked[it.Commodity] = true
			}
		}
		for _, id := range c.Roles[i].Recipes {
			if rc := lk.recipe[id]; rc != nil {
				keep(src.byRole, rc)
			}
		}
	}
	return src
}

// demandWantWarnings raises want_unused, want_unsourced and want_inelastic.
func demandWantWarnings(c *Config, lk lookup, add func(string, ...string)) {
	src := newWantSources(c, lk)
	for i := range c.Modules.Demand.Wants {
		w := &c.Modules.Demand.Wants[i]
		if !lk.demand.tasted[w.ID] {
			add("want_unused", "modules", "demand", "wants", w.ID)
		}
		wantItemWarnings(lk, src, w, add)
	}
}

// wantItemWarnings raises want_unsourced for an item that is not raw and that no
// recipe makes and no role starts with, so no one can sell it, and want_inelastic
// for a want of elasticity 10 000 or less on an item a role makes under the
// target-income rule (spec-demand.md 6.3, 10.2).
func wantItemWarnings(lk lookup, src wantSources, w *Want, add func(string, ...string)) {
	for i, it := range w.Items {
		p := []string{"modules", "demand", "wants", w.ID, "items[" + itoa(i) + "]", "commodity"}
		if cm := lk.commodity[it.Commodity]; cm != nil && !cm.Raw && !src.made[it.Commodity] && !src.stocked[it.Commodity] {
			add("want_unsourced", p...)
		}
		if w.Elasticity <= wholeBP && src.byRole[it.Commodity] {
			add("want_inelastic", p...)
		}
	}
}

// ladderSpend is the spending of one household of the role in one region per
// tick, summed over its ladder (spec-demand.md 7.1): wref with every schedule at
// 10 000, wmax with every schedule at its ceiling. The price of a want is the
// weighted mean of the effective reference prices of its items; a want with no
// price is skipped. ok is false when no want was priced or the arithmetic does
// not fit.
func ladderSpend(lk lookup, role *Role, region string) (wref, wmax int64, ok bool) {
	ix := lk.demand
	for _, t := range ix.tastes[role.ID] {
		w := ix.want[t.Want]
		if w == nil || t.Scale <= 0 {
			continue
		}
		price, pok := meanRef(lk, w, region)
		if !pok {
			continue
		}
		ref, top, wok := wantSpend(w, price, t.Scale, ix.regionScale(region, w.ID))
		var s1, s2 bool
		wref, s1 = checked.Add(wref, ref)
		wmax, s2 = checked.Add(wmax, top)
		if !wok || !s1 || !s2 {
			return 0, 0, false
		}
		ok = true
	}
	return wref, wmax, ok
}

// wantSpend is the spending of one household on one want per tick at the mean
// reference price: ref = price * Quantity * taste * region / (Period * 10^8) with
// the schedule at 10 000, and top the same times Ceiling / 10 000, truncated.
func wantSpend(w *Want, price, taste, region int64) (ref, top int64, ok bool) {
	a, aok := checked.Mul(price, w.Quantity)
	scales, sok := checked.Mul(taste, region)
	ceil, cok := checked.Mul(a, w.Ceiling)
	ref, rok := checked.MulDivDiv(a, scales, w.Period, 100_000_000)
	top, tok := checked.MulDivDiv(ceil, scales, w.Period, 1_000_000_000_000)
	return ref, top, aok && sok && cok && rok && tok
}

// meanRef is the mean of the effective reference prices of the items of a want
// in a region, weighted by the item weights, truncated: the price at which the
// substitution shares equal the weights (spec-demand.md 4.2, 7.1). ok is false
// when no item has a weight and a declared commodity.
func meanRef(lk lookup, w *Want, region string) (int64, bool) {
	var sum, total int64
	for _, it := range w.Items {
		cm := lk.commodity[it.Commodity]
		if cm == nil || it.Weight <= 0 {
			continue
		}
		line, ok := checked.Mul(it.Weight, int64(lk.price(region, cm)))
		s, sok := checked.Add(sum, line)
		t, tok := checked.Add(total, it.Weight)
		if !ok || !sok || !tok {
			return 0, false
		}
		sum, total = s, t
	}
	if total == 0 {
		return 0, false
	}
	return sum / total, true
}

// roleFlags are the three conditions that read a role with a ladder, true for a
// region of a group of the role.
type roleFlags struct{ capped, high, poor bool }

// or adds the flags of another region.
func (f *roleFlags) or(o roleFlags) {
	f.capped, f.high, f.poor = f.capped || o.capped, f.high || o.high, f.poor || o.poor
}

// regionFlags evaluates the conditions of wants_exceed_spend_cap,
// target_above_demand and class_without_income for a role in the region of one of
// its groups. Nothing is raised where the arithmetic cannot be done.
func regionFlags(c *Config, lk lookup, ro *Role, g Group) roleFlags {
	rg := lk.region[g.Region]
	if rg == nil {
		return roleFlags{}
	}
	upkeep, uok := listCost(lk, g.Region, effectiveUpkeep(c, ro), rg.ExtraUpkeep)
	wref, wmax, wok := ladderSpend(lk, ro, g.Region)
	if !uok || !wok {
		return roleFlags{}
	}
	return roleFlags{
		capped: spendCapExceeded(c.World, upkeep, wmax),
		high:   targetAboveDemand(ro.TargetRatio, upkeep, wref),
		poor:   classWithoutIncome(c, ro, upkeep, wref),
	}
}

// demandRoleWarnings raises the three warnings that read a role with a ladder
// and a group: wants_exceed_spend_cap, target_above_demand and
// class_without_income. Each is raised once per role, for any region of the
// role's groups that trips it.
func demandRoleWarnings(c *Config, lk lookup, ro *Role, add func(string, ...string)) {
	if len(lk.demand.tastes[ro.ID]) == 0 {
		return
	}
	var f roleFlags
	for _, g := range c.Groups {
		if g.Role == ro.ID {
			f.or(regionFlags(c, lk, ro, g))
		}
	}
	if f.capped {
		add("wants_exceed_spend_cap", "modules", "demand", "tastes", ro.ID)
	}
	if f.high {
		add("target_above_demand", "roles", ro.ID, "target_ratio")
	}
	if f.poor {
		add("class_without_income", "roles", ro.ID, "start_money")
	}
}

// spendCapExceeded is the condition of wants_exceed_spend_cap: Wmax is at least
// SpendMaxShare * SpendHalfTicks * UpkeepCost / 10 000, so the wants can never be
// fully funded (spec-demand.md 5.1, 10.2). An upkeep of 0 has no cap to measure.
func spendCapExceeded(w WorldParams, upkeep, wmax int64) bool {
	h, ok := checked.Mul(w.SpendMaxShare, w.SpendHalfTicks)
	limit, lok := checked.MulDiv(h, upkeep, wholeBP)
	return ok && lok && upkeep > 0 && wmax >= limit
}

// targetAboveDemand is the condition of target_above_demand: the role's target
// ratio is above 10 000 + 10 000 * Wref / UpkeepCost (spec-demand.md 7.1, 10.2),
// held as TargetRatio * U > 10 000 * (U + Wref).
func targetAboveDemand(target, upkeep, wref int64) bool {
	left, lok := checked.Mul(target, upkeep)
	sum, sok := checked.Add(upkeep, wref)
	right, rok := checked.Mul(wholeBP, sum)
	return lok && sok && rok && upkeep > 0 && left > right
}

// classWithoutIncome is the condition of class_without_income: the role has a
// ladder, no recipe and no Trades, no other income source, and a StartMoney below
// ReserveTicks times its upkeep cost plus a year of its wants at the reference
// (spec-demand.md 10.2).
func classWithoutIncome(c *Config, ro *Role, upkeep, wref int64) bool {
	if len(ro.Recipes) > 0 || len(ro.Trades) > 0 || hasIncomeSource(c, ro) {
		return false
	}
	year, yok := checked.Mul(wref, c.Calendar.TicksPerYear())
	reserve, rok := checked.Mul(upkeep, c.World.ReserveTicks)
	need, nok := checked.Add(reserve, year)
	return yok && rok && nok && c.Calendar.TicksPerYear() > 0 && int64(ro.StartMoney) < need
}

// demandReasons returns the ledger reasons the demand module declares: the goods
// sink for the consumption of a want (spec-ledger.md 7.2.2, FR-LED-050).
func demandReasons() []ledger.Reason { return []ledger.Reason{ledger.WantConsumption} }
