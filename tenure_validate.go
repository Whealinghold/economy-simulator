package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// This file is the validation of the tenure module (spec-land.md 11; a continuation
// file of the module, architecture.md 2.6): the rules of 11.1 that Validate checks and
// the warnings of 11.2. The module has no state yet: it is configured and checked and
// does nothing at run time.

// tenureBase is the path of the module in a configuration.
var tenureBase = []string{"modules", "tenure"}

// tenureRow is the path of row i of a list of the module, such as "term[0]", with
// the keys that follow it.
func tenureRow(list string, i int, key ...string) []string {
	return path(tenureBase, append([]string{list + "[" + itoa(i) + "]"}, key...)...)
}

// effectiveMaxHolderShareBP is the cap the module runs with: the field, or its
// default, 10 000 (the cap is off), when it is 0 (N-345).
func (m *TenureConfig) effectiveMaxHolderShareBP() int64 {
	if m.MaxHolderShareBP == 0 {
		return DefaultMaxHolderShareBP
	}
	return m.MaxHolderShareBP
}

// validateTenure checks the tenure module (spec-land.md 11.1, FR-LND-038). It reports
// nothing when the module is off. The rules run in phases and a phase that finds an
// error ends the run, so the first error is the same for the same mistake: the
// dependency, the counts and the tunables, the keys of the rows, the shares and the
// cap, which read the census, and last the overflow term, which reads the rest.
func (v *validator) validateTenure() {
	m := v.c.Modules.Tenure
	if m == nil {
		return
	}
	if v.c.Modules.Resources == nil {
		v.rep.add(ErrModuleDependency, tenureBase, "resources", "")
		return
	}
	steps := []func(*TenureConfig) bool{
		v.tenureCounts, v.tenureTunables, v.tenureTerms, v.tenureHoldings,
		v.tenureShares, v.tenureCap, v.tenureOverflow,
	}
	for _, step := range steps {
		if !step(m) {
			return
		}
	}
}

// tenureCounts is ErrOutOfBounds for more rows than MaxHoldings in either list.
func (v *validator) tenureCounts(m *TenureConfig) bool {
	n := len(v.rep.errs)
	for _, l := range []struct {
		name string
		n    int
	}{{"terms", len(m.Terms)}, {"holdings", len(m.Holdings)}} {
		if l.n > MaxHoldings {
			v.rep.add(ErrOutOfBounds, path(tenureBase, l.name), itoa(l.n), itoa(MaxHoldings))
		}
	}
	return len(v.rep.errs) == n
}

// tenureTunables checks the five tunables against their bounds (spec-land.md 10).
// A ceiling and a cap of 0 are the defaults and are in bound.
func (v *validator) tenureTunables(m *TenureConfig) bool {
	n := len(v.rep.errs)
	v.ranges(tenureBase, []fieldRange{
		{"rent_step_bp", m.RentStepBP, 0, MaxRentStepBP},
		{"rent_floor_bp", m.RentFloorBP, 0, wholeBP},
	})
	if m.RentCeilingBP != 0 {
		v.inRange(path(tenureBase, "rent_ceiling_bp"), m.RentCeilingBP, wholeBP, MaxRentCeilingBP)
	}
	if m.MaxHolderShareBP != 0 {
		v.inRange(path(tenureBase, "max_holder_share_bp"), m.MaxHolderShareBP, 1, wholeBP)
	}
	return len(v.rep.errs) == n
}

// tenureRegion checks the region of a row: it is defined and not an outside node.
func (v *validator) tenureRegion(p []string, region string, outside map[string]bool) {
	switch {
	case !v.region[region]:
		v.rep.add(ErrUnknownRegion, path(p, "region"), region, "")
	case outside[region]:
		v.rep.add(ErrOutsideNode, path(p, "region"), region, "")
	}
}

// tenureTerms checks the term rows: the key names a region that is not an outside
// node and a resource, appears once, and a rent applies within MaxPrice.
func (v *validator) tenureTerms(m *TenureConfig) bool {
	n := len(v.rep.errs)
	outside := outsideNodeSet(&v.c)
	seen := make(map[[2]string]bool, len(m.Terms)) // a membership test only
	for i := range m.Terms {
		t := &m.Terms[i]
		p := tenureRow("term", i)
		v.tenureRegion(p, t.Region, outside)
		if v.lk.resource[t.Resource] == nil {
			v.rep.add(ErrUnknownResource, path(p, "resource"), t.Resource, "")
		}
		key := [2]string{t.Region, t.Resource}
		if seen[key] {
			v.rep.add(ErrDuplicateID, path(p, "resource"), t.Resource, "")
		}
		seen[key] = true
		if t.HasRent {
			v.inRange(path(p, "rent"), t.Rent, 0, MaxPrice)
		}
	}
	return len(v.rep.errs) == n
}

// tenureHoldings checks the keys and the holders of the holding rows: the region and
// the resource, the kind and the role of the holder, and a key that appears once.
func (v *validator) tenureHoldings(m *TenureConfig) bool {
	n := len(v.rep.errs)
	outside := outsideNodeSet(&v.c)
	seen := make(map[[4]string]bool, len(m.Holdings)) // a membership test only
	for i := range m.Holdings {
		h := &m.Holdings[i]
		p := tenureRow("holding", i)
		v.tenureRegion(p, h.Region, outside)
		if v.lk.resource[h.Resource] == nil {
			v.rep.add(ErrUnknownResource, path(p, "resource"), h.Resource, "")
		}
		v.tenureHolder(p, h.Holder)
		key := [4]string{h.Region, h.Resource, h.Holder.Kind, h.Holder.Role}
		if seen[key] {
			v.rep.add(ErrDuplicateID, path(p, "holder"), h.Holder.Role, "")
		}
		seen[key] = true
	}
	return len(v.rep.errs) == n
}

// tenureHolder checks the holder of a holding: the pool names no role, a role holder
// names a defined one, and any other kind is outside the closed set.
func (v *validator) tenureHolder(p []string, h HolderRef) {
	switch h.Kind {
	case HolderKindPool:
		if h.Role != "" {
			v.rep.add(ErrOutOfBounds, path(p, "holder", "role"), h.Role, "")
		}
	case HolderKindRole:
		if !v.role[h.Role] {
			v.rep.add(ErrUnknownRole, path(p, "holder", "role"), h.Role, "")
		}
	default:
		v.rep.add(ErrOutOfBounds, path(p, "holder", "kind"), h.Kind, HolderKindRole+" or "+HolderKindPool)
	}
}

// tenureHouseholds is the households of the groups of each (region, role), the
// census of spec-land.md 6.1.1, and the largest weight of an agent of each. A group
// that the scale rules refuse is skipped: that rule has spoken.
func (v *validator) tenureHouseholds() (households, weight map[[2]string]int64) {
	households = make(map[[2]string]int64, len(v.c.Groups))
	weight = make(map[[2]string]int64, len(v.c.Groups))
	for _, g := range v.c.Groups {
		if _, w, _, ok := groupShape(v.c.World, g); ok {
			key := [2]string{g.Region, g.Role}
			households[key] += g.Households
			weight[key] = max(weight[key], w)
		}
	}
	return households, weight
}

// tenureShares is ErrTitleShares: a share outside 1 to 10 000, and a total over one
// endowment above 10 000 (3.1.2). A role holder contributes its share times the
// households of the role's groups in the region, so the sum is the census sum of
// 6.1.1. The error of a total is at the holding that takes it over.
func (v *validator) tenureShares(m *TenureConfig) bool {
	n := len(v.rep.errs)
	households, _ := v.tenureHouseholds()
	total := make(map[[2]string]int64, len(m.Holdings))
	over := make(map[[2]string]bool) // a membership test only
	for i := range m.Holdings {
		h := &m.Holdings[i]
		if !v.inRangeAs(ErrTitleShares, tenureRow("holding", i, "share_bp"), h.ShareBP, 1, wholeBP) {
			continue
		}
		units := int64(1)
		if h.Holder.Kind == HolderKindRole {
			units = households[[2]string{h.Region, h.Holder.Role}]
		}
		key := [2]string{h.Region, h.Resource}
		if over[key] {
			continue // one error per endowment, at the row that crosses
		}
		row, mulOK := checked.Mul(h.ShareBP, units)
		sum, addOK := checked.Add(total[key], row)
		total[key] = sum
		if !mulOK || !addOK || sum > wholeBP {
			v.rep.add(ErrTitleShares, tenureRow("holding", i, "share_bp"), i64(sum), i64(wholeBP))
			over[key] = true
		}
	}
	return len(v.rep.errs) == n
}

// inRangeAs is inRange with the error of the rule instead of ErrOutOfBounds.
func (v *validator) inRangeAs(rule error, p []string, val, lo, hi int64) bool {
	switch {
	case val < lo:
		v.rep.add(rule, p, i64(val), i64(lo))
		return false
	case val > hi:
		v.rep.add(rule, p, i64(val), i64(hi))
		return false
	}
	return true
}

// tenureCap is ErrTitleCap: with MaxHolderShareBP below 10 000, the largest agent
// of a role holder's group holds weight times the share, which may not exceed it
// (6.1.2). The pool is exempt.
func (v *validator) tenureCap(m *TenureConfig) bool {
	limit := m.effectiveMaxHolderShareBP()
	if limit >= wholeBP {
		return true
	}
	n := len(v.rep.errs)
	_, weight := v.tenureHouseholds()
	for i := range m.Holdings {
		h := &m.Holdings[i]
		if h.Holder.Kind != HolderKindRole {
			continue
		}
		held, ok := checked.Mul(weight[[2]string{h.Region, h.Holder.Role}], h.ShareBP)
		if !ok || held > limit {
			v.rep.add(ErrTitleCap, tenureRow("holding", i, "share_bp"), overflowOr(held, ok), i64(limit))
		}
	}
	return len(v.rep.errs) == n
}

// tenureOverflow is the derived check of ErrOverflow (11.1): a term's rent times the
// largest draw of a recipe on its resource times the largest weight of a group in
// its region may not exceed MaxMoney.
func (v *validator) tenureOverflow(m *TenureConfig) bool {
	n := len(v.rep.errs)
	_, weight := v.tenureHouseholds()
	for i := range m.Terms {
		t := &m.Terms[i]
		if !t.HasRent || t.Rent < 1 {
			continue
		}
		var wt int64
		for key, w := range weight {
			if key[0] == t.Region {
				wt = max(wt, w)
			}
		}
		product, ok := checked.Mul(largestDraw(&v.c, t.Resource), wt)
		if ok {
			product, ok = checked.Mul(product, t.Rent)
		}
		if !ok || product > MaxMoney {
			v.rep.add(ErrOverflow, tenureRow("term", i, "rent"), overflowOr(product, ok), i64(MaxMoney))
		}
	}
	return len(v.rep.errs) == n
}

// overflowOr is the Value of an ErrOverflow: the product, or overflowValue when it
// did not fit in int64.
func overflowOr(product int64, ok bool) string {
	if !ok {
		return overflowValue
	}
	return i64(product)
}

// largestDraw is the largest quantity of the resource that one execution of a recipe
// draws, 0 when no recipe draws on it.
func largestDraw(c *Config, resource string) int64 {
	var top int64
	for i := range c.Recipes {
		for _, d := range c.Recipes[i].Draws {
			if d.Resource == resource && d.Quantity >= 1 && int64(d.Quantity) <= MaxQuantity {
				top = max(top, int64(d.Quantity))
			}
		}
	}
	return top
}

// tenureFacts are the facts of a configuration that the warnings of 11.2 read: the
// effective term of each endowment, the census of each (region, role), the capacity
// of each endowment and the titles of each. It reads the configuration as written, so
// it is safe on one that does not validate.
type tenureFacts struct {
	c          *Config
	lk         lookup
	m          *TenureConfig
	term       map[[2]string]int   // the first term row of each endowment
	first      map[[2]string]int   // the first holding row of each endowment
	households map[[2]string]int64 // households of the groups of each (region, role)
	total      map[[2]string]int64 // the census sum of the titles of each endowment
	roleTitle  map[[3]string]bool  // (region, resource, role) holds a title
}

// newTenureFacts indexes the module of c, which is not nil.
func newTenureFacts(c *Config, lk lookup) *tenureFacts {
	m := c.Modules.Tenure
	f := &tenureFacts{c: c, lk: lk, m: m,
		term: make(map[[2]string]int, len(m.Terms)), first: make(map[[2]string]int, len(m.Holdings)),
		households: make(map[[2]string]int64, len(c.Groups)), total: make(map[[2]string]int64, len(m.Holdings)),
		roleTitle: make(map[[3]string]bool, len(m.Holdings))}
	for i := range m.Terms {
		key := [2]string{m.Terms[i].Region, m.Terms[i].Resource}
		if _, seen := f.term[key]; !seen {
			f.term[key] = i
		}
	}
	for _, g := range c.Groups {
		// a group the scale rules refuse is clamped, so a sum cannot wrap
		f.households[[2]string{g.Region, g.Role}] += min(max(g.Households, 0), MaxHouseholds)
	}
	for i := range m.Holdings {
		f.addHolding(i, &m.Holdings[i])
	}
	return f
}

// addHolding records a holding in the indexes. A share outside its bound adds
// nothing to the census sum.
func (f *tenureFacts) addHolding(i int, h *Holding) {
	key := [2]string{h.Region, h.Resource}
	if _, seen := f.first[key]; !seen {
		f.first[key] = i
	}
	if h.Holder.Kind == HolderKindRole {
		f.roleTitle[[3]string{h.Region, h.Resource, h.Holder.Role}] = true
	}
	if h.ShareBP < 1 || h.ShareBP > wholeBP {
		return
	}
	units := int64(1)
	if h.Holder.Kind == HolderKindRole {
		units = f.households[[2]string{h.Region, h.Holder.Role}]
	}
	if sum, ok := checked.Add(f.total[key], h.ShareBP*min(units, MaxHouseholds)); ok {
		f.total[key] = sum
	}
}

// rent is the effective rent of an endowment and whether it is closed: a row says
// either by its Has flag, else the default, no rent and open (3.3.1).
func (f *tenureFacts) rent(region, resource string) (rent int64, closed bool) {
	i, ok := f.term[[2]string{region, resource}]
	if !ok {
		return 0, false
	}
	t := &f.m.Terms[i]
	return t.Rent * boolInt(t.HasRent), t.HasClosed && t.Closed
}

// boolInt is 1 for true and 0 for false.
func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// capacity is the effective capacity of an endowment: the row's, else the resource's
// default, 0 for an unknown resource (spec-resources.md 3.3.1).
func (f *tenureFacts) capacity(region, resource string) int64 {
	res := f.lk.resource[resource]
	if m := f.c.Modules.Resources; m != nil {
		for i := range m.Endowments {
			if e := &m.Endowments[i]; e.Region == region && e.Resource == resource {
				return rowCapacity(e, res)
			}
		}
	}
	if res != nil {
		return res.Capacity
	}
	return 0
}

// tenureWarnings are the warnings of spec-land.md 11.2 that need no solver: no_titles,
// rent_without_titles, titles_without_rent, holding_unused, role_locked_out,
// entitlement_below_unit, rent_above_output, rent_hoarded and rent_step_inert
// (rent_off_dual is the solver's). A warning never fails validation and is safe on a
// configuration that does not validate: an undeclared ID is skipped.
func tenureWarnings(c *Config, lk lookup) []Warning {
	if c.Modules.Tenure == nil {
		return nil
	}
	f := newTenureFacts(c, lk)
	var out []Warning
	out = append(out, f.moduleWarnings()...)
	out = append(out, f.termWarnings()...)
	out = append(out, f.holdingWarnings()...)
	out = append(out, f.regionWarnings()...)
	out = append(out, f.recipeWarnings()...)
	return out
}

// moduleWarnings are no_titles and rent_step_inert, the rules about the module as a
// whole.
func (f *tenureFacts) moduleWarnings() []Warning {
	var out []Warning
	m := f.m
	moves := false // some term has a rent and is not closed
	acts := false  // some term has a rent or a closing
	for i := range m.Terms {
		r, closed := f.rent(m.Terms[i].Region, m.Terms[i].Resource)
		acts = acts || r > 0 || closed
		moves = moves || (r > 0 && !closed)
	}
	if len(m.Holdings) == 0 && !acts {
		out = append(out, Warning{Path: tenureBase, Rule: "no_titles"})
	}
	if m.RentStepBP > 0 && !moves {
		out = append(out, Warning{Path: path(tenureBase, "rent_step_bp"), Rule: "rent_step_inert"})
	}
	return out
}

// termWarnings are rent_without_titles, the rule of a term whose rent no one is paid,
// and role_locked_out, the rule of a closed endowment that is fully titled.
func (f *tenureFacts) termWarnings() []Warning {
	var out []Warning
	for i := range f.m.Terms {
		t := &f.m.Terms[i]
		key := [2]string{t.Region, t.Resource}
		rent, closed := f.rent(t.Region, t.Resource)
		if _, titled := f.first[key]; rent > 0 && !titled {
			out = append(out, Warning{Path: tenureRow("term", i), Rule: "rent_without_titles"})
		}
		if closed && f.total[key] == wholeBP {
			out = append(out, f.lockedOut(i, t)...)
		}
	}
	return out
}

// lockedOut is role_locked_out for one closed term: a role with a group in the region
// that lists, or has on its effective own-use list, a recipe that draws on the
// resource and holds no title.
func (f *tenureFacts) lockedOut(i int, t *TenureTerm) []Warning {
	var out []Warning
	for r := range f.c.Roles {
		ro := &f.c.Roles[r]
		if f.households[[2]string{t.Region, ro.ID}] == 0 || f.roleTitle[[3]string{t.Region, t.Resource, ro.ID}] {
			continue
		}
		if f.rolePlays(ro, t.Resource) {
			out = append(out, Warning{Path: tenureRow("term", i, "role", ro.ID), Rule: "role_locked_out"})
		}
	}
	return out
}

// rolePlays reports whether a role lists, or has on its own-use list, a recipe that
// draws on the resource.
func (f *tenureFacts) rolePlays(ro *Role, resource string) bool {
	for _, id := range jobIDs(f.c, ro) {
		rc := f.lk.recipe[id]
		if rc != nil && slices.ContainsFunc(rc.Draws, func(d Draw) bool { return d.Resource == resource }) {
			return true
		}
	}
	return false
}

// holdingWarnings are titles_without_rent (once per endowment, at its first holding),
// holding_unused and entitlement_below_unit.
func (f *tenureFacts) holdingWarnings() []Warning {
	var out []Warning
	for i := range f.m.Holdings {
		h := &f.m.Holdings[i]
		key := [2]string{h.Region, h.Resource}
		p := tenureRow("holding", i)
		capacity := f.capacity(h.Region, h.Resource)
		if rent, closed := f.rent(h.Region, h.Resource); f.first[key] == i && rent == 0 && !closed {
			out = append(out, Warning{Path: p, Rule: "titles_without_rent"})
		}
		noGroup := h.Holder.Kind == HolderKindRole && f.households[[2]string{h.Region, h.Holder.Role}] == 0
		if noGroup || capacity == 0 {
			out = append(out, Warning{Path: p, Rule: "holding_unused"})
		}
		if capacity > 0 && f.belowUnit(h, capacity) {
			out = append(out, Warning{Path: p, Rule: "entitlement_below_unit"})
		}
	}
	return out
}

// belowUnit is entitlement_below_unit for one holding: the plan stock of the endowment
// (spec-resources.md 11.1, PlanShareBP of the capacity) times the share, in basis
// points of a unit, is below one unit. The smallest share of a holder is the row's:
// a cohort's weight only raises it.
func (f *tenureFacts) belowUnit(h *Holding, capacity int64) bool {
	res := f.lk.resource[h.Resource]
	if res == nil || res.PlanShareBP < 1 || res.PlanShareBP > wholeBP || h.ShareBP < 1 || h.ShareBP > wholeBP {
		return false
	}
	plan := capacity / wholeBP * res.PlanShareBP // capacity is at most MaxEndowment, so this does not overflow
	plan += capacity % wholeBP * res.PlanShareBP / wholeBP
	product, ok := checked.Mul(plan, h.ShareBP)
	return ok && product < wholeBP
}

// regionWarnings is rent_hoarded, once per region that has a title: the pool may be
// given one by a removal, and the estates module is off or releases nothing there.
func (f *tenureFacts) regionWarnings() []Warning {
	var out []Warning
	done := make(map[string]bool, len(f.c.Regions)) // a membership test only
	for i := range f.m.Holdings {
		region := f.m.Holdings[i].Region
		if done[region] {
			continue
		}
		done[region] = true
		if r := f.lk.region[region]; r != nil && (f.c.Modules.Estates == nil || r.PoolRelease == 0) {
			out = append(out, Warning{Path: tenureRow("holding", i), Rule: "rent_hoarded"})
		}
	}
	return out
}

// recipeWarnings is rent_above_output: a recipe that a role of a region with a term
// can run, whose rent per execution at the configured rate reaches the value of its
// first output at the reference price times the yield at the mean site factor.
// Informational: it is raised once per (term, recipe).
func (f *tenureFacts) recipeWarnings() []Warning {
	var out []Warning
	for i := range f.m.Terms {
		t := &f.m.Terms[i]
		if rent, _ := f.rent(t.Region, t.Resource); rent < 1 {
			continue
		}
		for r := range f.c.Recipes {
			rc := &f.c.Recipes[r]
			if f.rentAboveOutput(t.Region, rc) && slices.ContainsFunc(rc.Draws, func(d Draw) bool { return d.Resource == t.Resource }) {
				out = append(out, Warning{Path: tenureRow("term", i, "recipe", rc.ID), Rule: "rent_above_output"})
			}
		}
	}
	return out
}

// rentAboveOutput reports whether the rent per execution of a recipe in a region, the
// sum of its draws times the rent of each resource, reaches the value of its first
// output. A recipe no role of the region with a group lists, or with no output, is
// not judged.
func (f *tenureFacts) rentAboveOutput(region string, rc *Recipe) bool {
	if len(rc.Outputs) == 0 || !f.runsIn(region, rc.ID) {
		return false
	}
	var rent int64
	for _, d := range rc.Draws {
		r, _ := f.rent(region, d.Resource)
		line, ok := checked.Mul(int64(d.Quantity), r)
		sum, ok2 := checked.Add(rent, line)
		if !ok || !ok2 {
			return false
		}
		rent = sum
	}
	out := rc.Outputs[0]
	cm := f.lk.commodity[out.Commodity]
	if cm == nil || out.Quantity < 1 {
		return false
	}
	value, ok := checked.Mul(int64(out.Quantity), int64(f.lk.price(region, cm)))
	if !ok {
		return false
	}
	return rent >= f.meanYield(region, rc, value)
}

// meanYield is the value scaled by the mean site factor of a sited recipe in the
// region, in basis points of the value; a recipe with no site is as it is.
func (f *tenureFacts) meanYield(region string, rc *Recipe, value int64) int64 {
	if !rc.Sited {
		return value
	}
	site := meanSites(f.c, []*Recipe{rc})
	if site == nil {
		return value
	}
	for i := range f.c.Regions {
		if f.c.Regions[i].ID == region {
			scaled, ok := checked.MulDiv(value, site(0, i), wholeBP)
			if !ok {
				return value
			}
			return scaled
		}
	}
	return value
}

// runsIn reports whether a role with a group in the region lists the recipe.
func (f *tenureFacts) runsIn(region, recipe string) bool {
	for i := range f.c.Roles {
		ro := &f.c.Roles[i]
		if f.households[[2]string{region, ro.ID}] > 0 && slices.Contains(jobIDs(f.c, ro), recipe) {
			return true
		}
	}
	return false
}
