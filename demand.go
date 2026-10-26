package economysim

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/household"
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
// (spec-demand.md 12). It holds what is derived from the configuration (the
// ladder of each role and the wants as the rules read them), the saved state (the
// carries of each agent, indexed by agent slot, N-244) and the scratch of a tick
// (the plan being checked, the tallies of P3.6 and the planned units that P6.2
// consumes).
type demandState struct {
	// ladder[role] is the wants of the role's ladder, by rank and then by want ID,
	// indexed by RoleID. A role with no taste row of nonzero scale has an empty
	// ladder and no want state (FR-DEM-004).
	ladder [][]rung
	// wants are the wants by WantID; regionBP the region scale of each (region,
	// want), region * len(wants) + want, 10 000 where no row says otherwise.
	wants    []wantRT
	regionBP []int64
	// carry[slot] is the carries of the agent in that slot, one per rung of its
	// role's ladder, each below 10 000 (spec-demand.md 4.4).
	carry [][]int64
	// off[role] is the first tally cell of the role within a region and block the
	// cells of a region: the rungs of all the ladders. byWant[role] is the ladder
	// positions of the role in ascending want ID, the order of the event.
	off    []int
	block  int
	byWant [][]int
	tally  []wantTally
	// cur is the plan of the agent last filled: the filler writes it and the
	// checker commits it. use is the planned units of the agents checked this tick,
	// by agent then commodity, which P6.2 consumes.
	cur wantCur
	use []wantUse
	// scratch of the filler, sized once from the largest ladder.
	viewWants []WantView
	viewItems []WantItemView
	hItems    []household.WantItem
	shares    []int64
	split     []int64
	merged    []wantUse
	funded    []int64
	oldIDs    []int64
	newIDs    []int64
	kept      []int64
}

// rung is one want of a ladder: the want and the taste scale of the role for it
// in basis points.
type rung struct {
	want  WantID
	scale int64
}

// wantRT is a want as the rules read it: its items by commodity ID and its
// numbers.
type wantRT struct {
	items                                               []wantItemRT
	quantity, period, elasticity, ceiling, substitution int64
}

// wantItemRT is one item of a want: the commodity and its weight in basis points.
type wantItemRT struct {
	commodity CommodityID
	weight    int64
}

// wantTally is what P3.2 counts for one (region, role, want) in a tick.
type wantTally struct{ households, choked, planned, funded int64 }

// wantUse is the planned units of one commodity of one agent.
type wantUse struct {
	agent     AgentID
	commodity CommodityID
	units     int64
}

// wantCur is the plan of one agent that the filler formed: the new carries, the
// planned units and the choke of each rung, and the units of each item, rung
// after rung. It is valid between the filler and the checker of one agent.
type wantCur struct {
	agent   AgentID
	valid   bool
	next    []int64
	planned []int64
	choked  []bool
	items   []wantUse // the commodity and units of each item (the agent is unused)
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
	st.build(c)
	return st
}

// build forms the runtime tables of the wants, the region scales, the tally
// layout and the scratch.
func (st *demandState) build(c *Config) {
	d := c.Modules.Demand
	st.wants = make([]wantRT, len(d.Wants))
	for i, w := range d.Wants {
		rt := wantRT{quantity: w.Quantity, period: w.Period, elasticity: w.Elasticity, ceiling: w.Ceiling, substitution: w.Substitution}
		for _, it := range w.Items {
			id, ok := c.commodityID(it.Commodity)
			if !ok {
				invariant("a want item names an unknown commodity after validation")
			}
			rt.items = append(rt.items, wantItemRT{commodity: id, weight: it.Weight})
		}
		st.wants[i] = rt
	}
	st.regionBP = slices.Repeat([]int64{wholeBP}, len(c.Regions)*len(d.Wants))
	for _, r := range d.RegionTastes {
		region, rok := c.regionID(r.Region)
		want, wok := c.wantID(r.Want)
		if !rok || !wok {
			invariant("a region taste names an unknown region or want after validation")
		}
		st.regionBP[int(region)*len(d.Wants)+int(want)] = r.Scale
	}
	st.layout(len(c.Regions))
}

// layout fixes the tally cells of the roles and sizes the scratch from the
// largest ladder.
func (st *demandState) layout(regions int) {
	var rungs, items, perWant int
	st.off, st.byWant = make([]int, len(st.ladder)), make([][]int, len(st.ladder))
	for role, l := range st.ladder {
		st.off[role] = rungs
		rungs += len(l)
		n := 0
		for _, rg := range l {
			n += len(st.wants[rg.want].items)
			perWant = max(perWant, len(st.wants[rg.want].items))
		}
		items = max(items, n)
		st.byWant[role] = slices.SortedFunc(slices.Values(positions(len(l))), func(a, b int) int {
			return cmp.Compare(l[a].want, l[b].want)
		})
	}
	st.block, st.tally = rungs, make([]wantTally, regions*rungs)
	widest := 0
	for _, l := range st.ladder {
		widest = max(widest, len(l))
	}
	st.viewWants, st.viewItems = make([]WantView, widest), make([]WantItemView, items)
	st.hItems, st.shares, st.split = make([]household.WantItem, 0, perWant), make([]int64, 0, perWant), make([]int64, 0, perWant)
	st.cur = wantCur{next: make([]int64, widest), planned: make([]int64, widest), choked: make([]bool, widest), items: make([]wantUse, items)}
	st.merged, st.funded = make([]wantUse, 0, items), make([]int64, widest)
	st.oldIDs, st.newIDs, st.kept = make([]int64, 0, widest), make([]int64, 0, widest), make([]int64, 0, widest)
}

// positions returns 0 to n-1.
func positions(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
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
	if !slices.ContainsFunc(lk.demand.tastes[ro.ID], func(t *Taste) bool { return t.Scale > 0 }) {
		return // a scale 0 row is not in the ladder (D304)
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

// commodityRow returns the view's row of the commodity, found by binary search
// in the canonical order of the tradable set. A want item is always in the set
// (FR-DEM-006), so a missing row is a programmer error.
func (v *AgentView) commodityRow(c CommodityID) *CommodityView {
	i, ok := slices.BinarySearchFunc(v.Commodities, c, func(e CommodityView, t CommodityID) int { return cmp.Compare(e.Commodity, t) })
	if !ok {
		invariant("a want item is not in the tradable set of the role")
	}
	return &v.Commodities[i]
}

// fillWants is the filler of the want group of AgentView (spec-demand.md 4.4,
// D308): for each want of the ladder in rank order its planned units, a Choked
// flag and the units split among its items. It commits nothing: the new carries
// wait in d.cur for the checker, so a view built at P2.5 advances no carry.
func (w *World) fillWants(v *AgentView, a *agent) {
	d := w.demand
	ladder := d.ladder[a.role]
	if len(ladder) == 0 {
		return
	}
	cur := &d.cur
	cur.agent, cur.valid = a.id, true
	carries, o := d.carry[a.id], 0
	for k, rg := range ladder {
		its := d.viewItems[o : o+len(d.wants[rg.want].items) : o+len(d.wants[rg.want].items)]
		cur.planned[k], cur.choked[k], cur.next[k] = d.rung(v, a, rg, carries[k], its)
		for j := range its {
			cur.items[o+j] = wantUse{commodity: its[j].Commodity, units: int64(its[j].Units)}
		}
		d.viewWants[k] = WantView{Want: rg.want, Planned: Quantity(cur.planned[k]), Choked: cur.choked[k], Items: its}
		o += len(its)
	}
	v.Wants = d.viewWants[:len(ladder):len(ladder)]
}

// rung forms one want of an agent's ladder: the price ratio of its items, the
// schedule, the whole units from the carry and their split among the items,
// written into its (commodity, share, units). An agent whose ratio or schedule
// cannot be formed, or is 0, is choked and keeps its carry (FR-DEM-011).
func (d *demandState) rung(v *AgentView, a *agent, rg rung, carry int64, its []WantItemView) (planned int64, choked bool, next int64) {
	wt := &d.wants[rg.want]
	hs := d.hItems[:0]
	for _, it := range wt.items {
		cv := v.commodityRow(it.commodity)
		hs = append(hs, household.WantItem{Centre: int64(cv.Belief.Centre), Ref: int64(cv.Ref), Weight: it.weight})
	}
	shares, sched, ok := d.schedule(hs, wt)
	units, next := int64(0), carry
	if ok && sched > 0 {
		var uok bool
		units, next, uok = household.Units(carry, wt.quantity, a.weight, rg.scale, d.regionBP[int(v.Region)*len(d.wants)+int(rg.want)], sched, wt.period)
		if !uok {
			units, next = 0, carry
		}
	}
	choked = !ok || sched == 0
	split, _ := household.Split(d.split[:0], units, shares)
	for j, it := range wt.items {
		its[j] = WantItemView{Commodity: it.commodity}
		if len(shares) > j {
			its[j].Share = shares[j]
		}
		if len(split) > j {
			its[j].Units = Quantity(split[j])
		}
	}
	return units, choked, next
}

// schedule is the shares, the price ratio and the schedule value of a want at the
// items hs. ok is false when any of them cannot be formed (a reference below 1, a
// centre below 1, an overflow), which counts as a choke.
func (d *demandState) schedule(hs []household.WantItem, wt *wantRT) (shares []int64, sched int64, ok bool) {
	shares, ok = household.Shares(d.shares[:0], hs, wt.substitution)
	if !ok {
		return nil, 0, false
	}
	ratio, ok := household.WantRatio(hs, shares)
	if !ok {
		return shares, 0, false
	}
	sched, ok = household.Schedule(ratio, wt.elasticity, wt.ceiling)
	return shares, sched, ok
}

// checkWants is the checker of the TradePlan for the want group (spec-demand.md
// 4.4, 4.6, D308), called once per agent at P3.2 after Trade: it commits the new
// carries, clamps Intent.WantUnits to the quantity and the planned units, tallies
// planned, funded and choked units for the event, and keeps the planned units per
// commodity for P6.2.
func (w *World) checkWants(a *agent, p *TradePlan) {
	d := w.demand
	ladder := d.ladder[a.role]
	if len(ladder) == 0 {
		for i := range p.Intents {
			p.Intents[i].WantUnits = 0
		}
		p.WantFunded = p.WantFunded[:0]
		return
	}
	cur := &d.cur
	if !cur.valid || cur.agent != a.id {
		invariant("the want plan of an agent was not formed by its filler")
	}
	cur.valid = false
	copy(d.carry[a.id], cur.next[:len(ladder)])
	d.mergeItems(a.id, ladder)
	d.clampIntents(p)
	d.tallyFunded(p, ladder)
	d.tallyRungs(int(w.regionOf(a)), a, ladder)
	d.use = append(d.use, d.merged...)
}

// mergeItems sums the planned units of the agent's ladder per commodity into
// d.merged, in ascending commodity ID.
func (d *demandState) mergeItems(agent AgentID, ladder []rung) {
	d.merged = d.merged[:0]
	n := 0
	for _, rg := range ladder {
		n += len(d.wants[rg.want].items)
	}
	for _, it := range d.cur.items[:n] {
		if it.units < 1 {
			continue
		}
		i, found := slices.BinarySearchFunc(d.merged, it.commodity, func(e wantUse, t CommodityID) int { return cmp.Compare(e.commodity, t) })
		if found {
			d.merged[i].units += it.units
			continue
		}
		d.merged = slices.Insert(d.merged, i, wantUse{agent: agent, commodity: it.commodity, units: it.units})
	}
}

// clampIntents limits the WantUnits of each intent to its quantity and to the
// planned units of its commodity; an ask has none.
func (d *demandState) clampIntents(p *TradePlan) {
	for i := range p.Intents {
		in := &p.Intents[i]
		planned := int64(0)
		if j, ok := slices.BinarySearchFunc(d.merged, in.Commodity, func(e wantUse, t CommodityID) int { return cmp.Compare(e.commodity, t) }); ok && in.Side == Bid {
			planned = d.merged[j].units
		}
		in.WantUnits = Quantity(max(0, min(int64(in.WantUnits), int64(in.Quantity), planned)))
	}
}

// tallyFunded reads TradePlan.WantFunded into d.funded, one number per rung,
// each at most the planned units of its want and the sum of the rows of a want
// at most that too. A row for a want not in the ladder counts for nothing.
func (d *demandState) tallyFunded(p *TradePlan, ladder []rung) {
	clear(d.funded[:len(ladder)])
	for _, f := range p.WantFunded {
		for k, rg := range ladder {
			if rg.want == f.Want {
				d.funded[k] = min(d.cur.planned[k], d.funded[k]+max(0, int64(f.Units)))
			}
		}
	}
}

// tallyRungs adds the agent's households, choked households, planned and funded
// units to the cell of each of its wants.
func (d *demandState) tallyRungs(region int, a *agent, ladder []rung) {
	base := region*d.block + d.off[a.role]
	for k := range ladder {
		c := &d.tally[base+k]
		c.households += a.weight
		c.planned += d.cur.planned[k]
		c.funded += d.funded[k]
		if d.cur.choked[k] {
			c.choked += a.weight
		}
	}
}

// p36Wants is the demand module's part of P3.6 (spec-demand.md 11.1, D312): it
// closes the tallies of P3.2 for the tick.
func (w *World) p36Wants() {
	clear(w.demand.tally)
}

// nextUpkeep is the units of the commodity the agent eats next tick (its
// households times the per-household quantity), 0 if it is not upkeep.
func (w *World) nextUpkeep(a *agent, region RegionID, c CommodityID) int64 {
	for _, it := range w.roles.row(a.role).upkeepIn(region) {
		if it.commodity == c {
			n, ok := checked.Mul(a.weight, int64(it.quantity))
			if !ok {
				invariant("upkeep times weight overflows after validation")
			}
			return n
		}
	}
	return 0
}

// p62Wants is P6.2 (spec-demand.md 4.7, FR-DEM-025): for each agent ascending and
// each planned commodity, the lesser of the planned units and what the agent
// holds above the next tick's upkeep of the commodity moves to the
// want_consumption boundary, oldest lots first. The error is a ledger refusal.
func (w *World) p62Wants() error {
	d := w.demand
	if d == nil {
		return nil
	}
	for _, u := range d.use {
		a := &w.agents[u.agent]
		if a.weight < 1 {
			continue
		}
		c := int(u.commodity)
		held, _, err := w.ledger.Holding(a.account, c)
		if err != nil {
			return fmt.Errorf("want consumption at tick %d: %w", w.Tick(), err)
		}
		n := household.Consumed(u.units, held, w.nextUpkeep(a, w.regionOf(a), u.commodity))
		if n < 1 {
			continue
		}
		sink, ok := w.ledger.Boundary(ledger.WantConsumption, c)
		if !ok {
			invariant("a commodity has no want_consumption boundary account")
		}
		if err := w.ledger.Move(a.account, sink, c, n, ledger.WantConsumption); err != nil {
			return fmt.Errorf("want consumption at tick %d: %w", w.Tick(), err)
		}
	}
	d.use = d.use[:0]
	return nil
}

// agentIndex is the 0-based position of the agent in its region's agent list,
// away agents included: the live agents of the region with a lower ID
// (spec-demand.md 4.4, FR-DEM-013).
func (w *World) agentIndex(a *agent, region RegionID) int64 {
	var n int64
	for i := len(w.cfg.Regions); i < int(a.id); i++ {
		if o := &w.agents[i]; o.weight >= 1 && w.regionOf(o) == region {
			n++
		}
	}
	return n
}

// joinedDemand forms the carries of a new agent (FR-DEM-013): the formula of 4.4
// for each rung of its role's ladder, with no draw.
func (w *World) joinedDemand(a *agent, region RegionID) {
	d := w.demand
	for len(d.carry) <= int(a.id) {
		d.carry = append(d.carry, nil)
	}
	ladder := d.ladder[a.role]
	if len(ladder) == 0 {
		d.carry[a.id] = nil
		return
	}
	index := w.agentIndex(a, region)
	carries := make([]int64, len(ladder))
	for k := range ladder {
		c, ok := household.InitialCarry(index, int64(k))
		if !ok {
			invariant("the initial carry of a want overflows")
		}
		carries[k] = c
	}
	d.carry[a.id] = carries
}

// roleChangedDemand re-forms the carries of an agent whose role changed
// (FR-DEM-042): kept by want ID for a want in both ladders, the formula for a
// want only in the new one, dropped for one only in the old.
func (w *World) roleChangedDemand(a *agent, from RoleID, region RegionID) {
	d := w.demand
	old, ladder := d.ladder[from], d.ladder[a.role]
	if len(ladder) == 0 {
		d.carry[a.id] = nil
		return
	}
	d.oldIDs, d.newIDs = d.oldIDs[:0], d.newIDs[:0]
	for _, rg := range old {
		d.oldIDs = append(d.oldIDs, int64(rg.want))
	}
	for _, rg := range ladder {
		d.newIDs = append(d.newIDs, int64(rg.want))
	}
	kept, ok := household.KeepCarries(d.kept[:0], d.oldIDs, d.carry[a.id], d.newIDs, w.agentIndex(a, region))
	if !ok {
		invariant("the carries of a role change overflow")
	}
	d.carry[a.id] = slices.Clone(kept)
}

// dropAgent is the module's step of removeAgent: the carries go with the agent.
func (d *demandState) dropAgent(a *agent) {
	if int(a.id) < len(d.carry) {
		d.carry[a.id] = nil
	}
}

// wantOf is Want_c: the planned units of the commodity over the wants of the
// view (spec-demand.md 4.6).
func (t *tradeCtx) wantOf(c CommodityID) int64 {
	var n int64
	for k := range t.v.Wants {
		for _, it := range t.v.Wants[k].Items {
			if it.Commodity == c {
				n += int64(it.Units)
			}
		}
	}
	return n
}

// wantScratch is the scratch of tier 4 of Trade: the ladder in the form of
// household.Fund and the funded units per want, grown once.
type wantScratch struct {
	rungs          []household.Rung
	shares, limits []int64
	out            []int64
	funded         []WantFunding
}

// rowOf is the trade row of the commodity, parallel to the view's tradable set.
func (t *tradeCtx) rowOf(c CommodityID) *tradeRow {
	i, _ := slices.BinarySearchFunc(t.v.Commodities, c, func(e CommodityView, x CommodityID) int { return cmp.Compare(e.Commodity, x) })
	return &t.rows[i]
}

// fundWants is tier 4 (spec-demand.md 4.5, D310): after the buffers it spends the
// rest of the cap, never more than the money left, down the ladder at each bid's
// own limit. A want item with no bid or no room in its tier 4 gets nothing. The
// units bought join the row's quantity and the funded units per want are kept for
// the plan.
func (t *tradeCtx) fundWants(avail, spendCap int64) {
	v := t.v
	if len(v.Wants) == 0 {
		return
	}
	ws := t.ws
	ws.funded = ws.funded[:0]
	total := 0
	for k := range v.Wants {
		total += len(v.Wants[k].Items)
	}
	ws.shares, ws.limits = slices.Grow(ws.shares[:0], total), slices.Grow(ws.limits[:0], total)
	ws.out, ws.rungs = slices.Grow(ws.out[:0], total)[:total], ws.rungs[:0]
	for k := range v.Wants {
		s0 := len(ws.shares)
		for _, it := range v.Wants[k].Items {
			ws.shares = append(ws.shares, it.Share)
			ws.limits = append(ws.limits, t.bidLimit(it.Commodity))
		}
		ws.rungs = append(ws.rungs, household.Rung{Units: int64(v.Wants[k].Planned), Shares: ws.shares[s0:], Limits: ws.limits[s0:]})
	}
	if _, ok := household.Fund(spendCap, 0, avail, ws.rungs, ws.out); ok {
		t.buyWants()
	}
}

// bidLimit is the limit of the commodity's bid if it has room for want units, else
// 0, which funds nothing.
func (t *tradeCtx) bidLimit(c CommodityID) int64 {
	if r := t.rowOf(c); r.kind == bidOrder && r.t4 > r.q4 {
		return r.limit
	}
	return 0
}

// buyWants adds the units Fund allocated to the rows, at most the tier 4 of each,
// and records the funded units per want.
func (t *tradeCtx) buyWants() {
	v, ws, off := t.v, t.ws, 0
	for k := range v.Wants {
		var funded int64
		for _, it := range v.Wants[k].Items {
			q := ws.out[off]
			off++
			r := t.rowOf(it.Commodity)
			if q = min(q, r.t4-r.q4); q < 1 {
				continue
			}
			r.qty, r.q4, r.bought = r.qty+q, r.q4+q, true
			funded += q
		}
		if funded > 0 {
			ws.funded = append(ws.funded, WantFunding{Want: v.Wants[k].Want, Units: Quantity(funded)})
		}
	}
}
