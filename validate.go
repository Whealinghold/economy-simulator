package economysim

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// maxReportedErrors is the most errors Validate returns before the one
// ErrTooManyErrors that says how many were left out (spec-config.md 7.1.3). It
// is a rule constant of the report, not a tunable.
const maxReportedErrors = 100

// Warning is a configuration that is legal but probably a mistake
// (spec-config.md 7.2). It never fails validation. Path is the path of the Go
// field as a ValidationError's Path is, and Rule is the name of the warning,
// such as "no_upkeep" (spec-config.md 7.4.2).
type Warning struct {
	// Path is the segments of the path of the field the warning is about.
	Path []string
	// Rule is the name of the warning.
	Rule string
}

// Validate checks the configuration in full and returns every error it finds,
// not the first, joined with errors.Join (spec-config.md 7.1). Each is a
// *ValidationError, or a *ViabilityError for a viability rule, so errors.Is
// matches a rule's sentinel (ErrNotViable for every viability rule) and
// errors.As finds the path, value and bound. The errors are sorted by path and
// then by rule, so the result is the same on every run, and at most 100 are
// returned, followed by one ErrTooManyErrors that says how many were left out. A
// valid configuration returns nil.
//
// Validate is pure and bounded: it changes nothing in c, starts no goroutine and
// draws no random number. It runs in the phases of spec-config.md 7.1.6. The
// counts of every collection are checked first; a count over its bound is
// ErrTooMany and stops the other rules, so no sort, set or fixpoint is built on
// a collection that is not known to be small. Then it checks the IDs and the
// shape of each entity, the duplicates and references between them, and the
// producibility of every commodity (spec-config.md 7.5).
//
// It checks the structural rules of spec-config.md 7.3 that need no module and
// the rules of 7.4.1 that the slice's sections own (fields and bounds in phase
// 2, the rules that read two sections in phase 4, the derived products of 4.2 in
// phase 5). Phase 6, the viability rules of spec-config.md 7.6 and the empty set
// of virtual start offsets of a warm-start job, runs only on a configuration the
// earlier phases found clean, on its canonical copy, under the size gate. The
// three name checks of 7.1.4 are NewWorld's, and the checks that need a module
// come with the module that builds them.
func (c Config) Validate() error {
	v := &validator{c: c}
	v.structural()
	if len(v.rep.errs) == 0 {
		v.viability()
	}
	return v.rep.err()
}

// Warnings returns the warnings of the configuration in order of path and then
// rule (spec-config.md 7.2). It never fails and has no effect on Validate. It is
// safe to call on a configuration that does not validate. The structural
// warnings are linear in the size of the configuration (the lists a role holds
// are read once per role); the viability check is not linear and is run again
// here, so a caller that needs both Validate and Warnings pays for it twice. The
// viability warnings of spec-config.md 7.4.2 are
// appended only when phases 1 to 5 of Validate come back clean on a scratch
// report, so Validate and Warnings agree on when the viability check runs, and
// they are bounded by the size gate of spec-config.md 7.6. `group_below_floor` is
// a run-time warning and is not here.
func (c Config) Warnings() []Warning {
	out := warnings(&c)
	v := &validator{c: c}
	v.structural()
	if len(v.rep.errs) > 0 {
		return out
	}
	cc := c.canonical()
	out = append(out, checkViability(&cc, newLookup(&cc)).warnings...)
	sortWarnings(out)
	return out
}

// sortWarnings orders warnings by path and then rule.
func sortWarnings(w []Warning) {
	slices.SortStableFunc(w, func(a, b Warning) int {
		return cmp.Or(slices.Compare(a.Path, b.Path), cmp.Compare(a.Rule, b.Rule))
	})
}

// structural runs phases 1 to 5 of spec-config.md 7.1.6.
func (v *validator) structural() {
	v.counts()
	if len(v.rep.errs) > 0 {
		return
	}
	v.declare()
	v.local()
	v.fields()
	v.references()
	v.structure()
	v.economics()
	v.overflow()
}

// viability is phase 6 (spec-config.md 7.1.6): on the canonical copy of a
// configuration phases 1 to 5 found clean, the empty set of virtual start offsets
// of a job and the viability rules (D125). A defect of the solver, which no
// input can cause, is reported as ErrSolveInconsistent.
func (v *validator) viability() {
	cc := v.c.canonical()
	m := newViaModel(&cc)
	v.emptyOffsets(m)
	v.addOutcome(m.check(newLookup(&cc)))
}

// addOutcome puts the errors of the viability checks in the report.
func (v *validator) addOutcome(out viabilityOutcome) {
	for _, e := range out.errs {
		v.rep.addViability(e)
	}
	if out.fault != nil {
		v.rep.add(out.fault, []string{"viability"}, "", "")
	}
}

// emptyOffsets reports ErrWarmStartJob for a job row of the warm start whose
// recipe has no virtual start offset in the group's region (spec-viability.md
// 8.5.2, 8.7, FR-VIA-033). It needs the yield and recipe tables, so it runs at
// the entry of phase 6 and not with the other warm-start rules of phase 4.
func (v *validator) emptyOffsets(m *viaModel) {
	ws := m.c.WarmStart
	if ws == nil {
		return
	}
	seen := map[[2]int]bool{} // a lookup only; no result depends on its order
	for _, g := range ws.Groups {
		region, _ := m.c.regionID(g.Region)
		for _, j := range g.Jobs {
			recipe, _ := m.c.recipeID(j.Recipe)
			key := [2]int{int(region), int(recipe)}
			empty, known := seen[key]
			if !known {
				empty = noVirtualStart(m.runsFor(region, recipe), m.year, m.c.Recipes[recipe].Duration)
				seen[key] = empty
			}
			if empty {
				v.rep.add(ErrWarmStartJob, []string{"warm_start", "groups", g.Region, g.Role, "jobs", j.Recipe}, j.Recipe, "")
			}
		}
	}
}

// finding is one rejected rule of a report: the error to return and the key the
// one sorted list orders by (path, rule name, value, bound), common to the two
// error types.
type finding struct {
	err          error
	path         []string
	rule         string
	value, bound string
}

// report collects the errors of one Validate call.
type report struct {
	errs []finding
}

// add records one rejected rule. The value and bound are text, as in
// ValidationError.
func (r *report) add(rule error, path []string, value, bound string) {
	r.errs = append(r.errs, finding{
		err:  &ValidationError{Path: path, Rule: rule, Value: value, Bound: bound},
		path: path, rule: rule.Error(), value: value, bound: bound,
	})
}

// addViability records one rejected viability rule in the same list.
func (r *report) addViability(e *ViabilityError) {
	r.errs = append(r.errs, finding{err: e, path: e.Path, rule: e.Rule, value: e.Value, bound: e.Bound})
}

// err returns nil for no error, and otherwise the errors sorted by path then
// rule, cut at maxReportedErrors, followed by one ErrTooManyErrors that says how
// many were left out, joined (spec-config.md 7.1.2, 7.1.3). The sort is total
// (it ends in the value and the bound), so the order does not depend on the
// order the rules ran in.
func (r *report) err() error {
	if len(r.errs) == 0 {
		return nil
	}
	slices.SortStableFunc(r.errs, func(a, b finding) int {
		return cmp.Or(
			slices.Compare(a.path, b.path),
			cmp.Compare(a.rule, b.rule),
			cmp.Compare(a.value, b.value),
			cmp.Compare(a.bound, b.bound),
		)
	})
	shown := r.errs
	if len(shown) > maxReportedErrors {
		shown = shown[:maxReportedErrors]
	}
	out := make([]error, 0, len(shown)+1)
	for _, e := range shown {
		out = append(out, e.err)
	}
	if left := len(r.errs) - len(shown); left > 0 {
		out = append(out, fmt.Errorf("%w: %d more not shown", ErrTooManyErrors, left))
	}
	return errors.Join(out...)
}

// validator is the state of one Validate call: the configuration, the report
// and the sets of declared IDs that the reference rules read. The sets are
// membership tests only; no result depends on the iteration order of a map.
type validator struct {
	c   Config
	rep report

	commodity, recipe, role, region map[string]bool
	override                        map[[2]string]Price
	dearest                         map[string]Price // the highest in-bounds override of each commodity
	lk                              lookup
	cal                             calendarFacts
}

// path returns a new slice of base followed by segs. It never shares base's
// backing array, so two errors never alias one path.
func path(base []string, segs ...string) []string {
	out := make([]string, 0, len(base)+len(segs))
	out = append(out, base...)
	return append(out, segs...)
}

// itoa writes an int as a decimal for a Value or a Bound.
func itoa(n int) string { return strconv.Itoa(n) }

// validID reports whether s is an ID of spec-config.md 3.3: [a-z][a-z0-9_]*, at
// most MaxIDLength bytes.
func validID(s string) bool {
	if len(s) == 0 || len(s) > MaxIDLength {
		return false
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch >= 'a' && ch <= 'z':
		case i > 0 && (ch >= '0' && ch <= '9' || ch == '_'):
		default:
			return false
		}
	}
	return true
}

// counts is phase 1 (spec-config.md 7.1.6): the length of every collection
// against its bound. The top-level collections are a len each, constant work;
// the nested lists are walked only if all of those hold, so the walk is
// bounded by them. Every count over its bound is ErrTooMany with its path
// (FR-CFG-101), whatever the owner spec's row says for a list of items.
func (v *validator) counts() {
	c, r := v.c, &v.rep
	over := func(n, bound int, p ...string) {
		if n > bound {
			r.add(ErrTooMany, p, itoa(n), itoa(bound))
		}
	}
	over(len(c.Commodities), MaxCommodities, "commodities")
	over(len(c.Recipes), MaxRecipes, "recipes")
	over(len(c.Roles), MaxRoles, "roles")
	over(len(c.Regions), MaxRegions, "regions")
	over(len(c.Edges), MaxEdges, "edges")
	over(len(c.RegionPrices), MaxRegionPrices, "region_prices")
	over(len(c.Groups), MaxGroups, "groups")
	over(len(c.Yields), MaxYields, "yields")
	over(len(c.Calendar.Months), MaxMonths, "calendar", "months")
	over(len(c.Calendar.Seasons), MaxSeasons, "calendar", "seasons")
	over(len(c.World.Upkeep), MaxUpkeepItems, "world", "upkeep")
	if w := c.WarmStart; w != nil {
		over(len(w.Prices), MaxSeedPrices, "warm_start", "prices")
		over(len(w.Wages), MaxRegions, "warm_start", "wages")
		over(len(w.Groups), MaxGroups, "warm_start", "groups")
		over(len(w.Pools), MaxRegions, "warm_start", "pools")
	}
	if len(r.errs) > 0 {
		return
	}

	for _, rc := range c.Recipes {
		over(len(rc.Inputs), MaxRecipeItems, "recipes", rc.ID, "inputs")
		over(len(rc.Outputs), MaxRecipeItems, "recipes", rc.ID, "outputs")
		over(len(rc.Requires), MaxRequires, "recipes", rc.ID, "requires")
	}
	for _, ro := range c.Roles {
		over(len(ro.Recipes), MaxRecipes, "roles", ro.ID, "recipes")
		over(len(ro.Upkeep), MaxUpkeepItems, "roles", ro.ID, "upkeep")
		over(len(ro.Trades), MaxCommodities, "roles", ro.ID, "trades")
		over(len(ro.StartStock), MaxCommodities, "roles", ro.ID, "start_stock")
	}
	for _, rg := range c.Regions {
		over(len(rg.ExtraUpkeep), MaxUpkeepItems, "regions", rg.ID, "extra_upkeep")
	}
	for _, y := range c.Yields {
		over(len(y.Seasons), MaxSeasons, "yields", y.Region, y.Recipe, "seasons")
	}
	if w := c.WarmStart; w != nil {
		for _, g := range w.Groups {
			over(len(g.Stock), MaxCommodities, "warm_start", "groups", g.Region, g.Role, "stock")
			over(len(g.Jobs), MaxRecipes, "warm_start", "groups", g.Region, g.Role, "jobs")
		}
	}
}

// declare fills the sets of declared IDs. An entity with an invalid ID is
// declared too, so a reference to it is not also reported as unknown.
func (v *validator) declare() {
	c := v.c
	v.commodity = make(map[string]bool, len(c.Commodities))
	for _, e := range c.Commodities {
		v.commodity[e.ID] = true
	}
	v.recipe = make(map[string]bool, len(c.Recipes))
	for _, e := range c.Recipes {
		v.recipe[e.ID] = true
	}
	v.role = make(map[string]bool, len(c.Roles))
	for _, e := range c.Roles {
		v.role[e.ID] = true
	}
	v.region = make(map[string]bool, len(c.Regions))
	for _, e := range c.Regions {
		v.region[e.ID] = true
	}
	v.lk = newLookup(&v.c)
	v.override = overrides(c.RegionPrices)
	v.dearest = make(map[string]Price, len(c.RegionPrices))
	for _, p := range c.RegionPrices {
		if p.Price <= MaxPrice && v.region[p.Region] && v.commodity[p.Commodity] {
			v.dearest[p.Commodity] = max(v.dearest[p.Commodity], p.Price)
		}
	}
}

// local is phase 2: the rules that read one entity. It reports ErrNoRegions,
// ErrInvalidID for every entity ID and ErrRecipeNoOutput. The field bounds of
// spec-config.md 7.4.1 belong here too.
func (v *validator) local() {
	c, r := v.c, &v.rep
	if len(c.Regions) == 0 {
		r.add(ErrNoRegions, []string{"regions"}, "", "")
	}
	id := func(p []string, s string) {
		if !validID(s) {
			r.add(ErrInvalidID, p, s, "")
		}
	}
	for _, e := range c.Commodities {
		id([]string{"commodities", e.ID, "id"}, e.ID)
	}
	for _, e := range c.Recipes {
		id([]string{"recipes", e.ID, "id"}, e.ID)
		// A recipe needs a non-discard output (FR-PRO-003); a discard output is
		// created and disposed of in one step and makes nothing.
		kept := false
		for _, o := range e.Outputs {
			kept = kept || !o.Discard
		}
		if !kept {
			r.add(ErrRecipeNoOutput, []string{"recipes", e.ID, "outputs"}, "", "")
		}
	}
	for _, e := range c.Roles {
		id([]string{"roles", e.ID, "id"}, e.ID)
	}
	for _, e := range c.Regions {
		id([]string{"regions", e.ID, "id"}, e.ID)
	}
	for i, e := range c.Calendar.Months {
		id([]string{"calendar", "months[" + itoa(i) + "]", "id"}, e.ID)
	}
	for i, e := range c.Calendar.Seasons {
		id([]string{"calendar", "seasons[" + itoa(i) + "]", "id"}, e.ID)
	}
}

// references is phase 3: duplicates and unknown references, by sets over
// collections that phase 1 bounded.
func (v *validator) references() {
	c, r := v.c, &v.rep

	dupIDs := func(kind string, ids []string) {
		seen := make(map[string]bool, len(ids))
		for _, id := range ids {
			if seen[id] {
				r.add(ErrDuplicateID, []string{kind, id, "id"}, id, "")
			}
			seen[id] = true
		}
	}
	dupIDs("commodities", mapSlice(c.Commodities, func(e Commodity) string { return e.ID }))
	dupIDs("recipes", mapSlice(c.Recipes, func(e Recipe) string { return e.ID }))
	dupIDs("roles", mapSlice(c.Roles, func(e Role) string { return e.ID }))
	dupIDs("regions", mapSlice(c.Regions, func(e Region) string { return e.ID }))
	// The word "pool" is reserved as a role ID (spec-config.md 3.3).
	for _, e := range c.Roles {
		if e.ID == "pool" {
			r.add(ErrDuplicateID, []string{"roles", e.ID, "id"}, e.ID, "")
		}
	}
	// A month or season ID repeats within its own list (FR-CAL-014).
	listDup := func(list string, ids []string) {
		seen := make(map[string]bool, len(ids))
		for i, id := range ids {
			if seen[id] {
				r.add(ErrDuplicateID, []string{"calendar", list + "[" + itoa(i) + "]", "id"}, id, "")
			}
			seen[id] = true
		}
	}
	listDup("months", mapSlice(c.Calendar.Months, func(e Month) string { return e.ID }))
	listDup("seasons", mapSlice(c.Calendar.Seasons, func(e Season) string { return e.ID }))

	// refs checks one list of IDs: a repeat is ErrDuplicateID and an ID that is
	// not in known is the unknown rule. field is the last path segment of an
	// item of a (commodity, quantity) list and "" for a list of plain IDs.
	refs := func(base []string, ids []string, known map[string]bool, unknown error, field string) {
		seen := make(map[string]bool, len(ids))
		for _, id := range ids {
			p := path(base, id)
			if field != "" {
				p = append(p, field)
			}
			if seen[id] {
				r.add(ErrDuplicateID, p, id, "")
			}
			seen[id] = true
			if !known[id] {
				r.add(unknown, p, id, "")
			}
		}
	}
	// one checks a single reference field.
	one := func(p []string, id string, known map[string]bool, unknown error) {
		if !known[id] {
			r.add(unknown, p, id, "")
		}
	}
	items := func(base []string, list []Item) {
		refs(base, mapSlice(list, func(e Item) string { return e.Commodity }), v.commodity, ErrUnknownCommodity, "commodity")
	}

	items([]string{"world", "upkeep"}, c.World.Upkeep)
	for _, e := range c.Recipes {
		base := []string{"recipes", e.ID}
		items(path(base, "inputs"), e.Inputs)
		refs(path(base, "outputs"), mapSlice(e.Outputs, func(o Output) string { return o.Commodity }), v.commodity, ErrUnknownCommodity, "commodity")
		refs(path(base, "requires"), mapSlice(e.Requires, func(t Tool) string { return t.Commodity }), v.commodity, ErrUnknownCommodity, "commodity")
	}
	for _, e := range c.Roles {
		base := []string{"roles", e.ID}
		refs(path(base, "recipes"), e.Recipes, v.recipe, ErrUnknownRecipe, "")
		items(path(base, "upkeep"), e.Upkeep)
		refs(path(base, "trades"), e.Trades, v.commodity, ErrUnknownCommodity, "")
		items(path(base, "start_stock"), e.StartStock)
	}
	for _, e := range c.Regions {
		items([]string{"regions", e.ID, "extra_upkeep"}, e.ExtraUpkeep)
	}

	v.graphRules()

	yield := make(map[[2]string]bool, len(c.Yields))
	for _, y := range c.Yields {
		key := [2]string{y.Region, y.Recipe}
		if yield[key] {
			r.add(ErrDuplicateID, []string{"yields", y.Region, y.Recipe}, y.Region+"/"+y.Recipe, "")
		}
		yield[key] = true
		one([]string{"yields", y.Region, y.Recipe, "region"}, y.Region, v.region, ErrUnknownRegion)
		one([]string{"yields", y.Region, y.Recipe, "recipe"}, y.Recipe, v.recipe, ErrUnknownRecipe)
	}

	group := make(map[[2]string]bool, len(c.Groups))
	for _, g := range c.Groups {
		key := [2]string{g.Region, g.Role}
		if group[key] {
			r.add(ErrDuplicateID, []string{"groups", g.Region, g.Role}, g.Region+"/"+g.Role, "")
		}
		group[key] = true
		one([]string{"groups", g.Region, g.Role, "region"}, g.Region, v.region, ErrUnknownRegion)
		one([]string{"groups", g.Region, g.Role, "role"}, g.Role, v.role, ErrUnknownRole)
	}

	if w := c.WarmStart; w != nil {
		price := make(map[[2]string]bool, len(w.Prices))
		for _, s := range w.Prices {
			key := [2]string{s.Region, s.Commodity}
			if price[key] {
				r.add(ErrDuplicateID, []string{"warm_start", "prices", s.Region, s.Commodity}, s.Region+"/"+s.Commodity, "")
			}
			price[key] = true
			one([]string{"warm_start", "prices", s.Region, s.Commodity, "region"}, s.Region, v.region, ErrUnknownRegion)
			one([]string{"warm_start", "prices", s.Region, s.Commodity, "commodity"}, s.Commodity, v.commodity, ErrUnknownCommodity)
		}
		regionRows := func(list string, ids []string) {
			seen := make(map[string]bool, len(ids))
			for _, id := range ids {
				if seen[id] {
					r.add(ErrDuplicateID, []string{"warm_start", list, id}, id, "")
				}
				seen[id] = true
				one([]string{"warm_start", list, id, "region"}, id, v.region, ErrUnknownRegion)
			}
		}
		regionRows("wages", mapSlice(w.Wages, func(e SeedWage) string { return e.Region }))
		regionRows("pools", mapSlice(w.Pools, func(e WarmPool) string { return e.Region }))
		wg := make(map[[2]string]bool, len(w.Groups))
		for _, g := range w.Groups {
			key := [2]string{g.Region, g.Role}
			base := []string{"warm_start", "groups", g.Region, g.Role}
			if wg[key] {
				r.add(ErrDuplicateID, base, g.Region+"/"+g.Role, "")
			}
			wg[key] = true
			one(path(base, "region"), g.Region, v.region, ErrUnknownRegion)
			one(path(base, "role"), g.Role, v.role, ErrUnknownRole)
			items(path(base, "stock"), g.Stock)
			refs(path(base, "jobs"), mapSlice(g.Jobs, func(j JobSpec) string { return j.Recipe }), v.recipe, ErrUnknownRecipe, "recipe")
		}
	}
}

// mapSlice returns f applied to every element of s, in order.
func mapSlice[T, U any](s []T, f func(T) U) []U {
	out := make([]U, len(s))
	for i, e := range s {
		out[i] = f(e)
	}
	return out
}

// structure is phase 4: the rules that read the recipe graph as a whole. It
// reports ErrOrphanCommodity and, by the fixpoint of spec-config.md 7.5,
// ErrUnproducible.
//
// A reference to an undeclared commodity is ignored here: phase 3 reported it,
// and treating it as unproducible would blame every output downstream of it for
// one mistake.
func (v *validator) structure() {
	c, r := v.c, &v.rep

	// consumed holds what a recipe input, a tool or an upkeep list names;
	// mentioned also holds what a recipe makes (spec-config.md 7.3).
	consumed := make(map[string]bool, len(v.commodity))
	mentioned := make(map[string]bool, len(v.commodity))
	use := func(id string) {
		consumed[id] = true
		mentioned[id] = true
	}
	for _, it := range c.World.Upkeep {
		use(it.Commodity)
	}
	for _, ro := range c.Roles {
		for _, it := range ro.Upkeep {
			use(it.Commodity)
		}
	}
	for _, rg := range c.Regions {
		for _, it := range rg.ExtraUpkeep {
			use(it.Commodity)
		}
	}

	// The recipes in ID order, so the witnesses below are the same on every run.
	recipes := slices.Clone(c.Recipes)
	slices.SortStableFunc(recipes, func(a, b Recipe) int { return cmp.Compare(a.ID, b.ID) })

	// needs lists, for each recipe, the declared commodities it needs: its inputs
	// and its tools (a required tool counts as an input, spec-config.md 7.5).
	needs := make([][]string, len(recipes))
	makers := make(map[string][]int)    // commodity -> recipes that output it, by ID
	consumers := make(map[string][]int) // commodity -> one entry per occurrence as a need
	remaining := make([]int, len(recipes))
	for i, rc := range recipes {
		for _, it := range rc.Inputs {
			use(it.Commodity)
			needs[i] = append(needs[i], it.Commodity)
		}
		for _, t := range rc.Requires {
			use(t.Commodity)
			needs[i] = append(needs[i], t.Commodity)
		}
		for _, o := range rc.Outputs {
			mentioned[o.Commodity] = true
			if v.commodity[o.Commodity] {
				makers[o.Commodity] = append(makers[o.Commodity], i)
			}
		}
		for _, id := range needs[i] {
			if v.commodity[id] {
				remaining[i]++
				consumers[id] = append(consumers[id], i)
			}
		}
	}

	// The fixpoint of 7.5 as a worklist: a commodity is producible if it is Raw
	// or some recipe that makes it has every need producible. The set is the
	// least fixpoint, so it does not depend on the order of the work.
	producible := make(map[string]bool, len(v.commodity))
	var queue []string
	mark := func(id string) {
		if v.commodity[id] && !producible[id] {
			producible[id] = true
			queue = append(queue, id)
		}
	}
	for _, cm := range c.Commodities {
		if cm.Raw {
			mark(cm.ID)
		}
	}
	for i, rc := range recipes {
		if remaining[i] == 0 {
			for _, o := range rc.Outputs {
				mark(o.Commodity)
			}
		}
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, i := range consumers[id] {
			remaining[i]--
			if remaining[i] == 0 {
				for _, o := range recipes[i].Outputs {
					mark(o.Commodity)
				}
			}
		}
	}

	for _, id := range slices.Sorted(maps.Keys(v.commodity)) {
		if !mentioned[id] {
			r.add(ErrOrphanCommodity, []string{"commodities", id}, id, "")
		}
		if consumed[id] && !producible[id] {
			p, why := witness(id, recipes, needs, makers, v.commodity, producible)
			r.add(ErrUnproducible, p, why, "")
		}
	}
}

// witness explains why a commodity is not producible: the path is
// "commodities", the commodity and then, alternately, a recipe that makes the
// commodity before it and the unproducible commodity that recipe needs, in ID
// order. It follows the lowest-ID recipe of each commodity and the lowest-ID
// unproducible need of that recipe, and stops at a commodity already on the
// chain, which is a cycle with no way in, or at a commodity no recipe makes
// (spec-config.md 7.5). The path is cut at MaxNestingDepth segments, the most an
// error text shows; the second result says which end it reached.
func witness(id string, recipes []Recipe, needs [][]string, makers map[string][]int, declared, producible map[string]bool) ([]string, string) {
	chain := []string{"commodities", id}
	seen := map[string]bool{id: true}
	why := "no recipe makes it"
	cur := id
	for {
		rs := makers[cur]
		if len(rs) == 0 {
			break
		}
		i := rs[0]
		next := ""
		for _, n := range needs[i] {
			if declared[n] && !producible[n] && (next == "" || n < next) {
				next = n
			}
		}
		if next == "" {
			// Cannot happen: cur is not producible, so each recipe that makes it
			// has a need that is not. Stop rather than loop.
			break
		}
		chain = append(chain, recipes[i].ID, next)
		if seen[next] {
			why = "cycle with no source"
			break
		}
		seen[next] = true
		cur = next
	}
	if len(chain) > MaxNestingDepth {
		chain = chain[:MaxNestingDepth]
	}
	return chain, why
}

// i64 writes an int64 as a decimal for a Value or a Bound.
func i64(n int64) string { return strconv.FormatInt(n, 10) }

// overflowValue is the Value of an ErrOverflow whose product does not even fit
// in int64, and overflowBound is the bound that product was held against.
const overflowValue = "overflow"

var overflowBound = i64(math.MaxInt64)

// lookup holds the entities of a configuration by ID. It is built without
// validating: an ID that repeats keeps its first entity, and a reference to an
// ID that is not declared is absent. The pointers point into the caller's
// slices, so a lookup is read-only.
type lookup struct {
	commodity map[string]*Commodity
	recipe    map[string]*Recipe
	role      map[string]*Role
	region    map[string]*Region
}

// newLookup indexes c. The result is valid while c's slices are unchanged.
func newLookup(c *Config) lookup {
	l := lookup{
		commodity: make(map[string]*Commodity, len(c.Commodities)),
		recipe:    make(map[string]*Recipe, len(c.Recipes)),
		role:      make(map[string]*Role, len(c.Roles)),
		region:    make(map[string]*Region, len(c.Regions)),
	}
	for i := range c.Commodities {
		if e := &c.Commodities[i]; l.commodity[e.ID] == nil {
			l.commodity[e.ID] = e
		}
	}
	for i := range c.Recipes {
		if e := &c.Recipes[i]; l.recipe[e.ID] == nil {
			l.recipe[e.ID] = e
		}
	}
	for i := range c.Roles {
		if e := &c.Roles[i]; l.role[e.ID] == nil {
			l.role[e.ID] = e
		}
	}
	for i := range c.Regions {
		if e := &c.Regions[i]; l.region[e.ID] == nil {
			l.region[e.ID] = e
		}
	}
	return l
}

// effectiveUpkeep is the upkeep list of a household of the role before the
// region's additions: the role's own list when it has one, else the world's
// (spec-agents.md 5.2).
func effectiveUpkeep(c *Config, r *Role) []Item {
	if len(r.Upkeep) > 0 {
		return r.Upkeep
	}
	return c.World.Upkeep
}

// tradableSets returns the tradable set of each role of c, in the order of
// c.Roles (spec-agents.md 3.4): the commodities of the role's effective upkeep
// in any region, the inputs, tools and non-discard outputs of the recipes on its
// list, and its Trades. The demand module's want items and the own-use
// outputs are added with those modules.
func tradableSets(c *Config, lk lookup) []map[string]bool {
	sets := make([]map[string]bool, len(c.Roles))
	for i := range c.Roles {
		r := &c.Roles[i]
		set := map[string]bool{}
		for _, it := range effectiveUpkeep(c, r) {
			set[it.Commodity] = true
		}
		for j := range c.Regions {
			for _, it := range c.Regions[j].ExtraUpkeep {
				set[it.Commodity] = true
			}
		}
		for _, id := range r.Recipes {
			rc := lk.recipe[id]
			if rc == nil {
				continue
			}
			for _, it := range rc.Inputs {
				set[it.Commodity] = true
			}
			for _, t := range rc.Requires {
				set[t.Commodity] = true
			}
			for _, o := range rc.Outputs {
				if !o.Discard {
					set[o.Commodity] = true
				}
			}
		}
		for _, id := range r.Trades {
			set[id] = true
		}
		sets[i] = set
	}
	return sets
}

// resolutionFloor is ceil(10 000 / BeliefStep), the least price a belief still
// moves at (spec-currency.md 3.1). The second result is false when BeliefStep is
// outside its bound, where the floor is not defined.
func resolutionFloor(w WorldParams) (int64, bool) {
	if w.BeliefStep < 1 || w.BeliefStep > 2_500 {
		return 0, false
	}
	return (wholeBP + w.BeliefStep - 1) / w.BeliefStep, true
}

// groupShape returns the number of agents of a group and the largest weight of
// one (spec-scale.md 3.2): n = ceil(H / S), raised to min(K, H) when below the
// floor K, and weights of H / n and one more for the first H mod n, so the
// largest is ceil(H / n). The last result is the scale in force: the group's, or
// the world's when the group gives none. ok is false when a household count, a
// scale or the floor is outside its bound, where the shape is not defined.
func groupShape(w WorldParams, g Group) (agents, maxWeight, scale int64, ok bool) {
	h, s := g.Households, g.Scale
	if s == 0 {
		s = w.Scale
	}
	if h < 1 || h > MaxHouseholds || s < 1 || s > MaxHouseholds || w.MinAgents < 1 || w.MinAgents > MaxHouseholds {
		return 0, 0, 0, false
	}
	n := (h + s - 1) / s
	if n < w.MinAgents {
		n = min(w.MinAgents, h)
	}
	return n, (h + n - 1) / n, s, true
}

// longestDuration is the longest Duration among the declared recipes of a role,
// and whether every declared recipe has a Duration inside its bound. A
// Duration outside it is reported by its own rule and the rules that read a
// window skip the role.
func longestDuration(lk lookup, r *Role) (longest int64, ok bool) {
	ok = true
	for _, id := range r.Recipes {
		rc := lk.recipe[id]
		if rc == nil {
			continue
		}
		if rc.Duration < 1 || rc.Duration > MaxRecipeTicks {
			ok = false
			continue
		}
		longest = max(longest, rc.Duration)
	}
	return longest, ok
}

// effectiveWindow is the welfare window of a role, max(WelfareWindow,
// WindowCycles * L) with L the longest Duration on its list
// (spec-welfare-and-wages.md 2.3.3). ok is false when the world's two window
// fields or a Duration are outside their bounds.
func effectiveWindow(c *Config, lk lookup, r *Role) (t int64, ok bool) {
	w := c.World
	if w.WelfareWindow < 1 || w.WelfareWindow > 1_000 || w.WindowCycles < 1 || w.WindowCycles > 100 {
		return 0, false
	}
	l, ok := longestDuration(lk, r)
	if !ok {
		return 0, false
	}
	// At most 100 * 100 000: the product cannot overflow inside the bounds.
	return max(w.WelfareWindow, w.WindowCycles*l), true
}

// calendarFacts are the quantities derived from a calendar that passes the
// bounds and the tick shape of spec-calendar.md 7.2 (spec-calendar.md 3.2).
type calendarFacts struct {
	ticksPerYear int64
	// ok is true when every bound and the tick shape hold, so ticksPerYear is
	// the length of the year in ticks. The rules that read TicksPerYear run only
	// then: the calendar's own rules have already blamed the cause.
	ok bool
}

// inRange reports whether val is in [lo, hi] and, when it is not, records
// ErrOutOfBounds at p with the end it broke as the bound.
func (v *validator) inRange(p []string, val, lo, hi int64) bool {
	switch {
	case val < lo:
		v.rep.add(ErrOutOfBounds, p, i64(val), i64(lo))
		return false
	case val > hi:
		v.rep.add(ErrOutOfBounds, p, i64(val), i64(hi))
		return false
	}
	return true
}

// ranges checks a table of field ranges under base.
func (v *validator) ranges(base []string, rs []fieldRange) {
	for _, f := range rs {
		v.inRange(path(base, f.name), f.value, f.lo, f.hi)
	}
}

// label records ErrOutOfBounds for a presentation label longer than
// MaxLabelBytes (spec-goods.md 6.5, FR-GDS-021).
func (v *validator) label(base []string, name, s string) {
	if len(s) > MaxLabelBytes {
		v.rep.add(ErrOutOfBounds, path(base, name), itoa(len(s)), itoa(MaxLabelBytes))
	}
}

// quantities checks the quantity of each item of a (commodity, quantity) list
// against [1, hi]; the path of an item is base, its commodity, "quantity".
func (v *validator) quantities(base []string, items []Item, hi int64) {
	for _, it := range items {
		v.inRange(path(base, it.Commodity, "quantity"), int64(it.Quantity), 1, hi)
	}
}

// perHousehold is the most of one commodity a household may be given as an
// upkeep or a start quantity: a weight times it must still fit MaxQuantity
// (spec-agents.md 12, spec-config.md 4.1).
const perHousehold = MaxQuantity / MaxWeight

// fields is phase 2: the rules that read one entity or one section, in the
// order of spec-config.md 7.4.1 by owner: the world's parameters, the calendar,
// commodities, recipes, yield profiles, roles, regions, groups and the warm
// start's own bounds.
func (v *validator) fields() {
	v.worldRules()
	v.calendarRules()
	v.commodityRules()
	v.recipeRules()
	v.yieldRules()
	v.roleRules()
	v.regionRules()
	v.edgeRules()
	v.groupRules()
	v.warmRules()
}

// worldRules checks the world's parameters (spec-agents.md 12, spec-beliefs.md
// 11, spec-spawner.md 11, spec-scale.md, spec-goods.md 8, spec-welfare-and-wages.md
// 12.1): each range, then the orders between two fields. An order is checked
// only when both of its fields are inside their own ranges, so one mistake is
// one error.
func (v *validator) worldRules() {
	w := v.c.World
	base := []string{"world"}
	inside := map[string]bool{}
	for _, f := range worldRanges(w) {
		inside[f.name] = v.inRange(path(base, f.name), f.value, f.lo, f.hi)
	}
	// order records an ErrOutOfBounds on name when ok is false; bound is the
	// value the field had to stay within.
	order := func(name, other string, ok bool, value, bound int64) {
		if inside[name] && inside[other] && !ok {
			v.rep.add(ErrOutOfBounds, path(base, name), i64(value), i64(bound))
		}
	}
	order("belief_stock_low", "belief_stock_high", w.BeliefStockLow < w.BeliefStockHigh, w.BeliefStockLow, w.BeliefStockHigh-1)
	order("belief_width_min", "belief_width_init", w.BeliefWidthMin <= w.BeliefWidthInit, w.BeliefWidthMin, w.BeliefWidthInit)
	order("belief_width_init", "belief_width_max", w.BeliefWidthInit <= w.BeliefWidthMax, w.BeliefWidthInit, w.BeliefWidthMax)
	order("belief_width_step", "belief_width_max", w.BeliefWidthStep <= w.BeliefWidthMax, w.BeliefWidthStep, w.BeliefWidthMax)
	order("cost_ceiling", "cost_floor", w.CostCeiling >= w.CostFloor, w.CostCeiling, w.CostFloor)
	// The rule ladder_order: the bound names the rule, as the value is a number
	// that is legal by itself (spec-spawner.md 12, FR-SPN-037).
	if inside["change_privation"] && inside["remove_privation"] && w.ChangePrivation >= w.RemovePrivation {
		v.rep.add(ErrOutOfBounds, path(base, "change_privation"), i64(w.ChangePrivation), "ladder_order")
	}
	v.quantities(path(base, "upkeep"), w.Upkeep, perHousehold)
	v.label(base, "distance_label", w.DistanceLabel)
}

// calendarRules checks spec-calendar.md 7.2 and records in v.cal what the other
// rules may read.
func (v *validator) calendarRules() {
	cal := v.c.Calendar
	base := []string{"calendar"}
	ok := v.inRange(path(base, "hours_per_day"), cal.HoursPerDay, 1, MaxHoursPerDay)
	ok = v.inRange(path(base, "days_per_week"), cal.DaysPerWeek, 1, MaxDaysPerWeek) && ok
	ok = v.inRange(path(base, "ticks_per_day"), cal.TicksPerDay, 1, MaxTicksPerDay) && ok
	ok = v.inRange(path(base, "days_per_tick"), cal.DaysPerTick, 1, math.MaxInt64) && ok

	// The months: at least one, each of 1 to MaxDaysPerMonth days, at most
	// MaxMonths of them (the count is phase 1's) and MaxDaysPerYear days in all.
	var days int64
	monthsOK := true
	if len(cal.Months) == 0 {
		v.rep.add(ErrOutOfBounds, path(base, "months"), "0", "1")
		monthsOK = false
	}
	for i, m := range cal.Months {
		p := path(base, "months["+itoa(i)+"]", "days")
		monthsOK = v.inRange(p, m.Days, 1, MaxDaysPerMonth) && monthsOK
		// Each month is at most MaxDaysPerMonth and there are at most MaxMonths,
		// so the sum cannot overflow; a month outside its bound is not added.
		if m.Days >= 1 && m.Days <= MaxDaysPerMonth {
			days += m.Days
		}
	}
	if monthsOK {
		monthsOK = v.inRange(path(base, "months"), days, 1, MaxDaysPerYear)
	}
	ok = ok && monthsOK
	if monthsOK {
		ok = v.inRange(path(base, "start_day"), cal.StartDay, 0, days-1) && ok
	}

	seasonsOK := true
	var seasonDays int64
	for i, s := range cal.Seasons {
		if s.Days < 1 {
			v.rep.add(ErrCalendarSeasons, path(base, "seasons["+itoa(i)+"]", "days"), i64(s.Days), "1")
			seasonsOK = false
			continue
		}
		// A season longer than the year is caught by the sum below; each is
		// counted at most one day past the bound of the year, so the sum of at
		// most MaxSeasons cannot overflow.
		seasonDays += min(s.Days, MaxDaysPerYear+1)
	}
	if len(cal.Seasons) > 0 && monthsOK {
		if seasonsOK && seasonDays != days {
			v.rep.add(ErrCalendarSeasons, path(base, "seasons"), i64(seasonDays), i64(days))
			seasonsOK = false
		}
		ok = v.inRange(path(base, "season_start"), cal.SeasonStart, 0, days-1) && ok
	}
	ok = ok && seasonsOK

	// The tick: a part of a day or a number of days, never both, and a whole
	// number of ticks in a year.
	var ticks int64
	if cal.TicksPerDay >= 1 && cal.DaysPerTick >= 1 && cal.TicksPerDay <= MaxTicksPerDay {
		switch {
		case cal.TicksPerDay > 1 && cal.DaysPerTick > 1:
			v.rep.add(ErrCalendarTick, path(base, "days_per_tick"), i64(cal.DaysPerTick), "1")
			ok = false
		case monthsOK && (days*cal.TicksPerDay)%cal.DaysPerTick != 0:
			v.rep.add(ErrCalendarTick, path(base, "days_per_tick"), i64(cal.DaysPerTick), i64(days*cal.TicksPerDay))
			ok = false
		case monthsOK:
			ticks = days * cal.TicksPerDay / cal.DaysPerTick
			ok = v.inRange(path(base, "ticks_per_year"), ticks, 1, MaxTicksPerYear) && ok
		}
	}
	v.cal = calendarFacts{ticksPerYear: ticks, ok: ok && ticks >= 1}
}

// commodityRules checks spec-goods.md 8 and 6.1: ranges, classes and labels.
func (v *validator) commodityRules() {
	for _, cm := range v.c.Commodities {
		base := []string{"commodities", cm.ID}
		v.ranges(base, commodityRanges(cm))
		if len(cm.Classes) > maxClasses {
			v.rep.add(ErrOutOfBounds, path(base, "classes"), itoa(len(cm.Classes)), itoa(maxClasses))
		}
		// Only the first maxClasses are read, so the work is bounded whatever
		// the length of the list.
		seen := make(map[string]bool, maxClasses)
		for _, cl := range cm.Classes[:min(len(cm.Classes), maxClasses)] {
			if !validID(cl) {
				v.rep.add(ErrInvalidID, path(base, "classes", cl), cl, "")
			}
			if seen[cl] {
				v.rep.add(ErrDuplicateID, path(base, "classes", cl), cl, "")
			}
			seen[cl] = true
		}
		v.label(base, "label", cm.Label)
		v.label(base, "unit_label", cm.UnitLabel)
	}
}

// recipeRules checks spec-production.md 10.2 and 3.2: ranges, tools, cost
// shares and trainee eligibility. The duplicate and reference rules are phase 3's.
func (v *validator) recipeRules() {
	for _, rc := range v.c.Recipes {
		base := []string{"recipes", rc.ID}
		v.ranges(base, recipeRanges(rc))
		v.label(base, "label", rc.Label)
		for _, it := range rc.Inputs {
			v.inRange(path(base, "inputs", it.Commodity, "quantity"), int64(it.Quantity), 1, MaxQuantity)
		}
		var shares int64
		shareFits, kept := true, false
		for _, o := range rc.Outputs {
			ob := path(base, "outputs", o.Commodity)
			v.inRange(path(ob, "quantity"), int64(o.Quantity), 1, MaxQuantity)
			v.inRange(path(ob, "cost_share"), o.CostShare, 0, wholeBP)
			if o.Discard {
				if o.CostShare != 0 {
					v.rep.add(ErrCostShares, path(ob, "cost_share"), i64(o.CostShare), "0")
				}
				continue
			}
			kept = true
			var ok bool
			if shares, ok = checked.Add(shares, o.CostShare); !ok {
				shareFits = false
			}
		}
		// A recipe with no kept output is ErrRecipeNoOutput; its shares have
		// nothing to add up to.
		if kept && (!shareFits || shares != wholeBP) {
			v.rep.add(ErrCostShares, path(base, "outputs"), i64(shares), i64(wholeBP))
		}
		for _, t := range rc.Requires {
			cm := v.lk.commodity[t.Commodity]
			if cm == nil {
				continue // an unknown commodity is phase 3's
			}
			tp := path(base, "requires", t.Commodity)
			if cm.Durability < 1 {
				v.rep.add(ErrToolNotDurable, path(tp, "commodity"), t.Commodity, "")
				continue
			}
			v.inRange(path(tp, "wear_per_use"), t.WearPerUse, 1, cm.Durability)
		}
		if rc.TraineeEligible && len(rc.Requires) > 0 {
			v.rep.add(ErrEligibleHasTool, path(base, "requires"), itoa(len(rc.Requires)), "0")
		}
	}
}

// yieldRules checks the yield profiles' factors and seasons
// (spec-production.md 6.1, 10.2). The unknown region and recipe and the
// duplicate profile are phase 3's.
func (v *validator) yieldRules() {
	seasons := make(map[string]bool, len(v.c.Calendar.Seasons))
	for _, s := range v.c.Calendar.Seasons {
		seasons[s.ID] = true
	}
	for _, y := range v.c.Yields {
		base := []string{"yields", y.Region, y.Recipe}
		v.inRange(path(base, "all"), y.All, 0, MaxYieldBP)
		seen := make(map[string]bool, len(y.Seasons))
		for _, sy := range y.Seasons {
			sp := path(base, "seasons", sy.Season)
			if !seasons[sy.Season] {
				v.rep.add(ErrUnknownSeason, path(sp, "season"), sy.Season, "")
			}
			if seen[sy.Season] {
				v.rep.add(ErrDuplicateID, path(sp, "season"), sy.Season, "")
			}
			seen[sy.Season] = true
			v.inRange(path(sp, "bp"), sy.BP, 0, MaxYieldBP)
		}
	}
}

// roleRules checks spec-agents.md 12, spec-welfare-and-wages.md 12 and
// spec-skill-barriers.md 5 for each role: ranges, the work expectation against
// the year, the lowest target, and the effective window.
func (v *validator) roleRules() {
	for i := range v.c.Roles {
		ro := &v.c.Roles[i]
		base := []string{"roles", ro.ID}
		v.ranges(base, roleRanges(*ro))
		v.label(base, "label", ro.Label)
		v.quantities(path(base, "upkeep"), ro.Upkeep, perHousehold)
		v.quantities(path(base, "start_stock"), ro.StartStock, perHousehold)

		// ExpectedWorkTicks is at most the derived TicksPerYear and never scaled
		// silently (FR-CFG-048); with no valid calendar the bound of the year is
		// the widest it could be.
		year := int64(MaxTicksPerYear)
		if v.cal.ok {
			year = v.cal.ticksPerYear
		}
		okE := v.inRange(path(base, "expected_work_ticks"), ro.ExpectedWorkTicks, 1, year)
		okM := v.inRange(path(base, "min_work_share"), ro.MinWorkShare, 0, wholeBP)
		// MinWorkShare * TicksPerYear / 10 000 is at most ExpectedWorkTicks
		// (FR-WEL-028): in integers MinWorkShare * TicksPerYear is at most
		// ExpectedWorkTicks * 10 000. The bound shown is the largest share.
		if okE && okM && v.cal.ok && ro.MinWorkShare*v.cal.ticksPerYear > ro.ExpectedWorkTicks*wholeBP {
			v.rep.add(ErrOutOfBounds, path(base, "min_work_share"), i64(ro.MinWorkShare),
				i64(ro.ExpectedWorkTicks*wholeBP/v.cal.ticksPerYear))
		}

		// The lowest target an agent can be drawn, TargetRatio * (10 000 - spread)
		// / 10 000, is at least 10 000 (FR-WEL-025): the spread is at most
		// 10 000 - ceil(10 000^2 / TargetRatio).
		if ro.TargetRatio >= wholeBP && ro.TargetRatio <= MaxBasisPoints && ro.TargetSpread >= 0 && ro.TargetSpread < wholeBP {
			if ro.TargetRatio*(wholeBP-ro.TargetSpread)/wholeBP < wholeBP {
				most := wholeBP - (wholeBP*wholeBP+ro.TargetRatio-1)/ro.TargetRatio
				v.rep.add(ErrOutOfBounds, path(base, "target_spread"), i64(ro.TargetSpread), i64(most))
			}
		}

		if t, ok := effectiveWindow(&v.c, v.lk, ro); ok && t > MaxWelfareWindow {
			v.rep.add(ErrOutOfBounds, path(base, "recipes"), i64(t), i64(MaxWelfareWindow))
		}
	}
}

// regionRules checks the ranges, the extra upkeep and the label of each region.
func (v *validator) regionRules() {
	for _, rg := range v.c.Regions {
		base := []string{"regions", rg.ID}
		v.ranges(base, regionRanges(rg))
		v.quantities(path(base, "extra_upkeep"), rg.ExtraUpkeep, perHousehold)
		v.label(base, "label", rg.Label)
	}
}

// groupRules checks the ranges of each group, that a traineeship needs agents of
// weight 1 (rule traineeship_needs_weight_one, spec-skill-barriers.md 4.1.1)
// and that no agent holds more than MaxWeight households.
func (v *validator) groupRules() {
	w := v.c.World
	for _, g := range v.c.Groups {
		base := []string{"groups", g.Region, g.Role}
		v.ranges(base, groupRanges(g))
		_, weight, scale, ok := groupShape(w, g)
		if !ok {
			continue
		}
		if ro := v.lk.role[g.Role]; ro != nil && ro.TrainingTicks > 0 && scale > 1 {
			v.rep.add(ErrOutOfBounds, path(base, "scale"), i64(scale), "traineeship_needs_weight_one")
		}
		if weight > MaxWeight {
			v.rep.add(ErrOutOfBounds, path(base, "scale"), i64(weight), i64(MaxWeight))
		}
	}
}

// warmRules checks the warm start's own bounds (spec-viability.md 8.2, 8.7): the
// seed prices against the bounds, the resolution floor and the reference band,
// the starting wages, the money and the stock, and the jobs. The rules that
// need the solver, and the virtual start offsets that need the calendar's
// season lookup (spec-viability.md 8.5.2), come with Stage 1.
func (v *validator) warmRules() {
	ws := v.c.WarmStart
	if ws == nil {
		return
	}
	w := v.c.World
	floor, floorOK := resolutionFloor(w)
	bandOK := w.RefBandLow >= 1 && w.RefBandLow <= wholeBP && w.RefBandHigh >= wholeBP && w.RefBandHigh <= MaxBasisPoints

	for _, s := range ws.Prices {
		p := path([]string{"warm_start", "prices", s.Region, s.Commodity}, "price")
		price := int64(s.Price)
		switch {
		case price < 1:
			v.rep.add(ErrWarmStartPrice, p, i64(price), "1")
			continue
		case price > MaxPrice:
			v.rep.add(ErrWarmStartPrice, p, i64(price), i64(MaxPrice))
			continue
		}
		if floorOK && price < floor {
			v.rep.add(ErrWarmStartPrice, p, i64(price), i64(floor))
		}
		ref, refOK := v.refPrice(s.Region, s.Commodity)
		if !refOK || !bandOK {
			continue
		}
		// The band runs from RefBandLow to RefBandHigh basis points of the
		// effective reference price of the seed's region (spec-beliefs.md 3.2.1,
		// spec-regions.md 5.2.2); the products are at most 10^9 * 10^6.
		lo, hi := ref*w.RefBandLow, ref*w.RefBandHigh
		switch scaled := price * wholeBP; {
		case scaled < lo:
			v.rep.add(ErrWarmStartPrice, p, i64(price), i64((lo+wholeBP-1)/wholeBP))
		case scaled > hi:
			v.rep.add(ErrWarmStartPrice, p, i64(price), i64(hi/wholeBP))
		}
	}

	// The labour cost of the longest recipe: a wage times it is at most MaxMoney
	// (spec-viability.md 8.2.5).
	var longest int64
	for _, rc := range v.c.Recipes {
		if rc.Duration >= 1 && rc.Duration <= MaxRecipeTicks {
			longest = max(longest, rc.Duration)
		}
	}
	for _, sw := range ws.Wages {
		p := []string{"warm_start", "wages", sw.Region, "wage"}
		wage := int64(sw.Wage)
		switch {
		case wage < 1:
			v.rep.add(ErrWarmStartPrice, p, i64(wage), "1")
		case wage > MaxMoney:
			v.rep.add(ErrWarmStartPrice, p, i64(wage), i64(MaxMoney))
		default:
			if cost, ok := checked.Mul(wage, longest); !ok || cost > MaxMoney {
				v.rep.add(ErrWarmStartPrice, p, i64(wage), i64(MaxMoney/max(longest, 1)))
			}
		}
	}

	money := func(p []string, m Money) {
		switch {
		case m < 0:
			v.rep.add(ErrOutOfBounds, p, i64(int64(m)), "0")
		case m > MaxMoney:
			v.rep.add(ErrOverflow, p, i64(int64(m)), i64(MaxMoney))
		}
	}
	for _, wp := range ws.Pools {
		money([]string{"warm_start", "pools", wp.Region, "money"}, wp.Money)
	}
	for _, g := range ws.Groups {
		base := []string{"warm_start", "groups", g.Region, g.Role}
		money(path(base, "money"), g.Money)
		for _, it := range g.Stock {
			p := path(base, "stock", it.Commodity, "quantity")
			switch q := int64(it.Quantity); {
			case q < 1:
				v.rep.add(ErrOutOfBounds, p, i64(q), "1")
			case q > MaxQuantity:
				v.rep.add(ErrOverflow, p, i64(q), i64(MaxQuantity))
			}
		}
		ro := v.lk.role[g.Role]
		var shares int64
		sharesFit := true
		for _, j := range g.Jobs {
			jp := path(base, "jobs", j.Recipe)
			if j.InProgressBP < 0 || j.InProgressBP > wholeBP {
				v.rep.add(ErrWarmStartJob, path(jp, "in_progress"), i64(j.InProgressBP), i64(wholeBP))
				sharesFit = false
			} else {
				shares += j.InProgressBP
			}
			// A recipe that is not declared is phase 3's; one that is declared and
			// not on the role's list is a job the role cannot hold.
			if ro != nil && v.lk.recipe[j.Recipe] != nil && !slices.Contains(ro.Recipes, j.Recipe) {
				v.rep.add(ErrWarmStartJob, path(jp, "recipe"), j.Recipe, "")
			}
		}
		if sharesFit && shares > wholeBP {
			v.rep.add(ErrWarmStartJob, path(base, "jobs"), i64(shares), i64(wholeBP))
		}
	}
}

// economics is phase 4 for the rules that read two sections: the resolution
// floor against the commodities' reference prices and the reference basket
// against the upkeep (spec-config.md 7.4.1).
func (v *validator) economics() {
	c := &v.c
	if floor, ok := resolutionFloor(c.World); ok {
		v.priceFloor(floor)
	}

	tol := c.World.BasketTolerance
	if tol < 0 || tol > wholeBP {
		return
	}
	for _, rg := range c.Regions {
		if rg.ReferenceBasket < 0 || rg.ReferenceBasket > MaxMoney {
			continue
		}
		cost, ok := v.baselineCost(rg.ID, rg.ExtraUpkeep)
		if !ok {
			continue
		}
		diff := int64(rg.ReferenceBasket) - cost
		if diff < 0 {
			diff = -diff
		}
		// |B - cost| * 10 000 > tolerance * B, held as |B - cost| > the whole
		// units of tolerance * B / 10 000 (the quotient fits: tolerance is at
		// most 10 000).
		allowed, _ := checked.MulDiv(tol, int64(rg.ReferenceBasket), wholeBP)
		if diff > allowed {
			v.rep.add(ErrBasketMismatch, []string{"regions", rg.ID, "reference_basket"}, i64(int64(rg.ReferenceBasket)), i64(cost))
		}
	}
}

// priceFloor checks that the effective reference price of every tradable
// commodity, in every region, clears the resolution floor (spec-regions.md
// 5.2.3). A commodity is tradable if it is in the tradable set of some role
// (spec-agents.md 3.4). A default price below the floor is reported once, at
// the commodity, when some region still uses it; an override below the floor is
// reported at its own row.
func (v *validator) priceFloor(floor int64) {
	c := &v.c
	tradable := map[string]bool{}
	for _, set := range tradableSets(c, v.lk) {
		for id := range set {
			tradable[id] = true
		}
	}
	overridden := v.overridesBelow(floor, tradable)
	for _, cm := range c.Commodities {
		price := int64(cm.ReferencePrice)
		inUse := overridden[cm.ID] == 0 || len(v.region) > overridden[cm.ID]
		if tradable[cm.ID] && inUse && price >= 1 && price <= MaxPrice && price < floor {
			v.rep.add(ErrOutOfBounds, []string{"commodities", cm.ID, "reference_price"}, i64(price), i64(floor))
		}
	}
}

// overridesBelow reports an override of a tradable commodity below the floor at
// its row, and returns the number of regions that override each commodity, a
// repeated row counted once (a
// count only; no result depends on the iteration order of the map).
func (v *validator) overridesBelow(floor int64, tradable map[string]bool) map[string]int {
	overridden := map[string]int{}
	seen := make(map[[2]string]bool, len(v.c.RegionPrices))
	for i, p := range v.c.RegionPrices {
		if !v.region[p.Region] || !v.commodity[p.Commodity] {
			continue
		}
		// A repeated row (ErrDuplicateID, reported elsewhere) is one region, once.
		if key := [2]string{p.Region, p.Commodity}; !seen[key] {
			seen[key] = true
			overridden[p.Commodity]++
		}
		if price := int64(p.Price); tradable[p.Commodity] && price >= 1 && price <= MaxPrice && price < floor {
			v.rep.add(ErrOutOfBounds, []string{"region_prices[" + itoa(i) + "]", "price"}, i64(price), i64(floor))
		}
	}
	return overridden
}

// baselineCost is the cost of the world's upkeep plus a region's extra upkeep
// at the region's effective reference prices (spec-currency.md 5.1,
// spec-regions.md 5.2.3). ok is false when a commodity is not declared, a price
// or a quantity is outside its bound (the rules for those have spoken) so the
// cost is not defined; inside the bounds the sum is at most 32 items of 1 000
// units at 10^9, which fits.
func (v *validator) baselineCost(region string, extra []Item) (cost int64, ok bool) {
	for _, list := range [][]Item{v.c.World.Upkeep, extra} {
		for _, it := range list {
			price, priceOK := v.refPrice(region, it.Commodity)
			q := int64(it.Quantity)
			if !priceOK || q < 1 || q > perHousehold {
				return 0, false
			}
			cost += q * price
		}
	}
	return cost, true
}

// overflow is phase 5: the derived products of spec-config.md 4.2 that the
// field bounds do not already keep inside their limits, with ErrOverflow. A
// product whose inputs are outside their own bounds is not computed: that rule
// has spoken. The viability rules of phase 6 come with Stage 1.
func (v *validator) overflow() {
	c := &v.c
	w := c.World

	// The largest weight of each role's groups (spec-scale.md 3.2), and the
	// regions it has a group in.
	weight := make(map[string]int64, len(c.Roles))
	inRegion := make(map[string][]string, len(c.Roles))
	for _, g := range c.Groups {
		if _, wt, _, ok := groupShape(w, g); ok {
			weight[g.Role] = max(weight[g.Role], wt)
			inRegion[g.Role] = append(inRegion[g.Role], g.Region)
		}
	}

	// Per recipe: the largest yield factor (the largest table entry, times the
	// shock bound, spec-production.md 10.3) in basis points.
	entry := make(map[string]int64, len(c.Recipes))
	hasEntry := make(map[string]bool, len(c.Recipes))
	for _, y := range c.Yields {
		if y.All >= 0 && y.All <= MaxYieldBP {
			entry[y.Recipe] = max(entry[y.Recipe], y.All)
		}
		for _, s := range y.Seasons {
			if s.BP >= 0 && s.BP <= MaxYieldBP {
				entry[y.Recipe] = max(entry[y.Recipe], s.BP)
			}
		}
		hasEntry[y.Recipe] = true
	}
	// yieldMax is the factor a recipe's outputs can be multiplied by, in basis
	// points: a recipe that is not Sited has 10 000 where no entry says
	// otherwise. At most 10^5 * 10^5 / 10^4.
	yieldMax := func(rc *Recipe) int64 {
		top := entry[rc.ID]
		if !rc.Sited {
			top = max(top, wholeBP)
		}
		return top * MaxYieldBP / wholeBP
	}
	// price is the dearest a commodity can be in any region (spec-regions.md
	// 5.2), because a product must fit wherever the household is.
	price := func(id string) (int64, bool) {
		cm := v.lk.commodity[id]
		if cm == nil || cm.ReferencePrice < 1 || cm.ReferencePrice > MaxPrice {
			return 0, false
		}
		return int64(max(cm.ReferencePrice, v.dearest[id])), true
	}
	over := func(p []string, product int64, ok bool, bound int64) {
		switch {
		case !ok:
			v.rep.add(ErrOverflow, p, overflowValue, overflowBound)
		case product > bound:
			v.rep.add(ErrOverflow, p, i64(product), i64(bound))
		}
	}

	// A recipe's weight is the largest of the roles that list it.
	recipeWeight := make(map[string]int64, len(c.Recipes))
	for i := range c.Roles {
		ro := &c.Roles[i]
		for _, id := range ro.Recipes {
			recipeWeight[id] = max(recipeWeight[id], weight[ro.ID])
		}
	}
	for i := range c.Recipes {
		rc := &c.Recipes[i]
		wt := recipeWeight[rc.ID]
		if wt == 0 {
			continue
		}
		base := []string{"recipes", rc.ID}
		ym := yieldMax(rc)
		for _, it := range rc.Inputs {
			if q := int64(it.Quantity); q >= 1 && q <= MaxQuantity {
				p, ok := checked.Mul(wt, q)
				over(path(base, "inputs", it.Commodity, "quantity"), p, ok, MaxQuantity)
			}
		}
		for _, o := range rc.Outputs {
			if q := int64(o.Quantity); q >= 1 && q <= MaxQuantity {
				wq, _ := checked.Mul(wt, q) // at most 10^6 * 10^9
				p, ok := checked.MulDiv(wq, ym, wholeBP)
				over(path(base, "outputs", o.Commodity, "quantity"), p, ok, MaxQuantity)
			}
		}
		// The cost of the inputs at the largest reference price is at most
		// MaxMoney, so UnitCost cannot overflow (spec-production.md 10.3).
		var cost int64
		costOK := true
		for _, it := range rc.Inputs {
			pr, ok := price(it.Commodity)
			q := int64(it.Quantity)
			if !ok || q < 1 || q > MaxQuantity {
				continue
			}
			line, lineOK := checked.Mul(q, pr)
			if sum, sumOK := checked.Add(cost, line); lineOK && sumOK {
				cost = sum
			} else {
				costOK = false
			}
		}
		over(path(base, "inputs"), cost, costOK, MaxMoney)
	}

	for i := range c.Roles {
		ro := &c.Roles[i]
		wt := weight[ro.ID]
		if wt == 0 {
			continue
		}
		v.roleProducts(ro, wt, inRegion[ro.ID], yieldMax, price, over)
	}

	v.moneyAtStart()
	v.warmProducts()
}

// roleProducts checks the products of one role with a group: the targets of
// spec-agents.md 7.3 times the weight, the welfare window times the largest
// income, and the calibration product of spec-welfare-and-wages.md 3.1.
func (v *validator) roleProducts(ro *Role, wt int64, regions []string, yieldMax func(*Recipe) int64,
	price func(string) (int64, bool), over func([]string, int64, bool, int64)) {
	c := &v.c
	w := c.World
	base := []string{"roles", ro.ID}

	// The units a household holds of a commodity: the upkeep over CoverTicks and
	// the tick, the inputs of InputJobs jobs and a tool (7.3, with k* = 1).
	if w.CoverTicks >= 1 && w.CoverTicks <= 1_000 && w.InputJobs >= 1 && w.InputJobs <= 100 {
		// The upkeep: the role's or the world's list plus the largest extra of
		// the regions the role has a group in.
		upkeep := map[string]int64{}
		for _, it := range effectiveUpkeep(c, ro) {
			if q := int64(it.Quantity); q >= 1 && q <= perHousehold {
				upkeep[it.Commodity] += q
			}
		}
		extra := map[string]int64{}
		for _, id := range regions {
			if rg := v.lk.region[id]; rg != nil {
				per := map[string]int64{}
				for _, it := range rg.ExtraUpkeep {
					if q := int64(it.Quantity); q >= 1 && q <= perHousehold {
						per[it.Commodity] += q
					}
				}
				for cm, q := range per {
					extra[cm] = max(extra[cm], q)
				}
			}
		}
		for cm, q := range extra {
			upkeep[cm] += q
		}
		jobs := map[string]int64{}
		for _, id := range ro.Recipes {
			rc := v.lk.recipe[id]
			if rc == nil {
				continue
			}
			for _, it := range rc.Inputs {
				if q := int64(it.Quantity); q >= 1 && q <= MaxQuantity {
					jobs[it.Commodity] = max(jobs[it.Commodity], q*w.InputJobs)
				}
			}
			for _, t := range rc.Requires {
				jobs[t.Commodity] = max(jobs[t.Commodity], 1)
			}
		}
		ids := map[string]bool{}
		for id := range upkeep {
			ids[id] = true
		}
		for id := range jobs {
			ids[id] = true
		}
		for _, id := range slices.Sorted(maps.Keys(ids)) {
			// upkeep is at most 10^3 * 10^3 (a household's list, 1 001 ticks); the
			// job part is at most 10^9 * 10^2: neither overflows.
			held := upkeep[id]*(1+w.CoverTicks) + jobs[id]
			p, ok := checked.Mul(wt, held)
			over(path(base, "targets", id), p, ok, MaxQuantity)
		}
	}

	// The window times the largest income a tick can bring: the most one
	// execution of any of the role's recipes sells, at the largest reference
	// price and the largest yield, times the weight (spec-welfare-and-wages.md
	// 2.3.2, 2.3.4).
	if t, ok := effectiveWindow(c, v.lk, ro); ok && t <= MaxWelfareWindow {
		var income int64
		incomeOK := true
		for _, id := range ro.Recipes {
			rc := v.lk.recipe[id]
			if rc == nil {
				continue
			}
			ym := yieldMax(rc)
			var line int64
			for _, o := range rc.Outputs {
				pr, pok := price(o.Commodity)
				q := int64(o.Quantity)
				if o.Discard || !pok || q < 1 || q > MaxQuantity {
					continue
				}
				units, _ := checked.MulDiv(q, ym, wholeBP) // at most 10^9 * 10^6 / 10^4
				val, vok := checked.Mul(units, pr)
				sum, sok := checked.Add(line, val)
				if !vok || !sok {
					incomeOK = false
					break
				}
				line = sum
			}
			income = max(income, line)
		}
		// The product is held in int64 whatever the bound, so over reports only
		// a product that does not fit.
		var p int64
		if incomeOK {
			scaled, scaledOK := checked.Mul(income, wt)
			p, incomeOK = checked.Mul(scaled, t)
			incomeOK = incomeOK && scaledOK
		}
		over(path(base, "recipes"), p, incomeOK, math.MaxInt64)
	}

	// The wage of the target ratio, r_bp * B_r * D_year / (10 000 * D_work), uses
	// the 128-bit helper; the check is that the result fits (FR-WEL-010).
	okR := ro.TargetRatio >= wholeBP && ro.TargetRatio <= MaxBasisPoints
	okE := v.cal.ok && ro.ExpectedWorkTicks >= 1 && ro.ExpectedWorkTicks <= v.cal.ticksPerYear
	if okR && okE {
		for _, id := range slices.Compact(slices.Sorted(slices.Values(regions))) {
			rg := v.lk.region[id]
			if rg == nil || rg.ReferenceBasket < 0 || rg.ReferenceBasket > MaxMoney {
				continue
			}
			// TargetRatio * TicksPerYear is at most 10^6 * 10^5; the divisor at
			// most 10^4 * 10^5.
			_, ok := checked.MulDiv(ro.TargetRatio*v.cal.ticksPerYear, int64(rg.ReferenceBasket), wholeBP*ro.ExpectedWorkTicks)
			if !ok {
				v.rep.add(ErrOverflow, path(base, "target_ratio"), overflowValue, overflowBound)
				break
			}
		}
	}
}

// moneyAtStart checks that the money in existence at construction fits MaxMoney:
// the households of each group times their money, from the warm start where a
// row gives it and from the role otherwise, and the pools (spec-config.md 4.2).
func (v *validator) moneyAtStart() {
	c := &v.c
	type key struct{ region, role string }
	warm := map[key]Money{}
	var pools int64
	ok := true
	if ws := c.WarmStart; ws != nil {
		for _, g := range ws.Groups {
			if g.Money >= 0 && g.Money <= MaxMoney {
				warm[key{g.Region, g.Role}] = g.Money
			}
		}
		for _, p := range ws.Pools {
			if p.Money >= 0 && p.Money <= MaxMoney {
				var sok bool
				if pools, sok = checked.Add(pools, int64(p.Money)); !sok {
					ok = false
				}
			}
		}
	}
	total := pools
	for _, g := range c.Groups {
		ro := v.lk.role[g.Role]
		if ro == nil || g.Households < 1 || g.Households > MaxHouseholds {
			continue
		}
		m, has := warm[key{g.Region, g.Role}]
		if !has {
			if ro.StartMoney < 0 || ro.StartMoney > MaxMoney/MaxWeight {
				continue
			}
			m = ro.StartMoney
		}
		part, pok := checked.Mul(g.Households, int64(m))
		sum, sok := checked.Add(total, part)
		if !pok || !sok {
			ok = false
			break
		}
		total = sum
	}
	switch {
	case !ok:
		v.rep.add(ErrOverflow, []string{"groups"}, overflowValue, i64(MaxMoney))
	case total > MaxMoney:
		v.rep.add(ErrOverflow, []string{"groups"}, i64(total), i64(MaxMoney))
	}
}

// warmProducts checks the products of the warm start (spec-viability.md 8.7):
// a stock quantity times the weight of its group, and a starting wage times a
// role's expected work (spec-spawner.md 12).
func (v *validator) warmProducts() {
	ws := v.c.WarmStart
	if ws == nil {
		return
	}
	w := v.c.World
	type key struct{ region, role string }
	weight := map[key]int64{}
	for _, g := range v.c.Groups {
		if _, wt, _, ok := groupShape(w, g); ok {
			weight[key{g.Region, g.Role}] = wt
		}
	}
	for _, g := range ws.Groups {
		wt := weight[key{g.Region, g.Role}]
		for _, it := range g.Stock {
			if q := int64(it.Quantity); q >= 1 && q <= MaxQuantity && wt > 0 {
				p, _ := checked.Mul(wt, q) // at most 10^9 * 10^9: a weight is at most MaxHouseholds
				if p > MaxQuantity {
					v.rep.add(ErrOverflow, []string{"warm_start", "groups", g.Region, g.Role, "stock", it.Commodity, "quantity"},
						i64(p), i64(MaxQuantity))
				}
			}
		}
	}
	wage := map[string]int64{}
	for _, sw := range ws.Wages {
		if sw.Wage >= 1 && sw.Wage <= MaxMoney {
			wage[sw.Region] = int64(sw.Wage)
		}
	}
	for _, g := range v.c.Groups {
		ro, wg := v.lk.role[g.Role], wage[g.Region]
		if ro == nil || wg == 0 || ro.ExpectedWorkTicks < 1 || ro.ExpectedWorkTicks > MaxTicksPerYear {
			continue
		}
		if p, ok := checked.Mul(wg, ro.ExpectedWorkTicks); !ok || p > MaxMoney {
			v.rep.add(ErrOverflow, []string{"roles", ro.ID, "expected_work_ticks"}, i64(ro.ExpectedWorkTicks), i64(MaxMoney/wg))
		}
	}
}

// entryShareWindow is the product of EntryShare and the longest effective
// window above which `entry_share_window` is raised: the model's condition of
// spec-spawner.md 12, a placeholder.
const entryShareWindow = 2_500

// warnings returns the warnings of spec-config.md 7.4.2 of the slice, sorted by
// path and then rule. It reads c as given and never fails: an ID that is not
// declared is skipped, and arithmetic that cannot be done is not a warning.
func warnings(c *Config) []Warning {
	lk := newLookup(c)
	var out []Warning
	add := func(rule string, segs ...string) {
		out = append(out, Warning{Path: segs, Rule: rule})
	}

	// no_upkeep: no upkeep at the world, role or region level, so no household
	// can starve (spec-agents.md 5.2, 13).
	hasUpkeep := len(c.World.Upkeep) > 0
	for _, r := range c.Roles {
		hasUpkeep = hasUpkeep || len(r.Upkeep) > 0
	}
	for _, r := range c.Regions {
		hasUpkeep = hasUpkeep || len(r.ExtraUpkeep) > 0
	}
	if !hasUpkeep {
		add("no_upkeep", "world", "upkeep")
	}
	// no_open_role: every role is Closed, so the default spawner places nobody
	// (spec-spawner.md 12).
	open := false
	for _, r := range c.Roles {
		open = open || !r.Closed
	}
	if len(c.Roles) > 0 && !open {
		add("no_open_role", "roles")
	}

	// cost_floor_at_full_cost: the floor equals the full unit cost at the steady
	// state and the wage loop has gain 1 (spec-beliefs.md FR-BEL-038).
	if c.World.CostFloor >= wholeBP {
		add("cost_floor_at_full_cost", "world", "cost_floor")
	}
	// inexact_hour and season_shorter_than_tick (spec-calendar.md 7.3).
	cal := c.Calendar
	if cal.TicksPerDay > 1 && cal.HoursPerDay%cal.TicksPerDay != 0 {
		add("inexact_hour", "calendar", "ticks_per_day")
	}
	if cal.DaysPerTick > 1 {
		for _, s := range cal.Seasons {
			if s.Days < cal.DaysPerTick {
				add("season_shorter_than_tick", "calendar", "seasons")
				break
			}
		}
	}

	// The recipe and commodity warnings of spec-production.md 10.4 and
	// spec-agents.md 13.
	listed := map[string]bool{}  // a recipe on some role's list
	started := map[string]bool{} // a recipe some role with a group can run
	required := map[string]bool{}
	for i := range c.Roles {
		for _, id := range c.Roles[i].Recipes {
			listed[id] = true
		}
	}
	for i := range c.Recipes {
		for _, t := range c.Recipes[i].Requires {
			required[t.Commodity] = true
		}
	}
	made := map[string]bool{}        // some recipe outputs it, kept
	discardOnly := map[string]bool{} // some recipe outputs it, and only discarded
	for i := range c.Recipes {
		for _, o := range c.Recipes[i].Outputs {
			if o.Discard {
				if !made[o.Commodity] {
					discardOnly[o.Commodity] = true
				}
				continue
			}
			made[o.Commodity] = true
			delete(discardOnly, o.Commodity)
		}
	}
	// positive[recipe] is true when a yield profile of the recipe has a factor
	// above 0 in some season; profiled[recipe] when it has a profile at all.
	type regionRecipe struct{ region, recipe string }
	runsIn := map[regionRecipe]bool{}
	profiledAt := map[regionRecipe]bool{}
	positive := map[string]bool{}
	profiled := map[string]bool{}
	for _, y := range c.Yields {
		profiled[y.Recipe] = true
		profiledAt[regionRecipe{y.Region, y.Recipe}] = true
		pos := y.HasAll && y.All > 0
		for _, s := range y.Seasons {
			pos = pos || s.BP > 0
		}
		// A profile with no entry at all is the default of the recipe
		// (spec-production.md 6.2): 10 000 unless the recipe is Sited.
		if !y.HasAll && len(y.Seasons) == 0 {
			if rc := lk.recipe[y.Recipe]; rc != nil {
				pos = !rc.Sited
			}
		}
		positive[y.Recipe] = positive[y.Recipe] || pos
		if pos {
			runsIn[regionRecipe{y.Region, y.Recipe}] = true
		}
	}
	// runs reports whether a recipe can start in a region: by its profile
	// there if it has one, else by the default of the recipe (spec-production.md
	// 6.2, FR-PRO-020).
	runs := func(rc *Recipe, region string) bool {
		if profiledAt[regionRecipe{region, rc.ID}] {
			return runsIn[regionRecipe{region, rc.ID}]
		}
		return !rc.Sited
	}

	groups := map[string][]*Group{}
	for i := range c.Groups {
		g := &c.Groups[i]
		groups[g.Role] = append(groups[g.Role], g)
	}
	for id, gs := range groups {
		ro := lk.role[id]
		if ro == nil {
			continue
		}
		for _, g := range gs {
			for _, rid := range ro.Recipes {
				if rc := lk.recipe[rid]; rc != nil && runs(rc, g.Region) {
					started[rid] = true
				}
			}
		}
	}

	for _, cm := range c.Commodities {
		if cm.Durability > 0 && !required[cm.ID] {
			add("durable_not_required", "commodities", cm.ID)
		}
		if cm.Raw {
			// raw_without_source: nothing real stands behind the marker, that is,
			// no recipe with no inputs that makes it runs for a role that has
			// households (spec-viability.md 4.2.1).
			sourced := false
			for i := range c.Recipes {
				rc := &c.Recipes[i]
				if len(rc.Inputs) > 0 || !started[rc.ID] {
					continue
				}
				for _, o := range rc.Outputs {
					sourced = sourced || (o.Commodity == cm.ID && !o.Discard)
				}
			}
			if !sourced {
				add("raw_without_source", "commodities", cm.ID)
			}
		}
	}
	for i := range c.Recipes {
		rc := &c.Recipes[i]
		if !listed[rc.ID] && !rc.TraineeEligible {
			add("unused_recipe", "recipes", rc.ID)
		}
		if !positive[rc.ID] && (rc.Sited || profiled[rc.ID]) {
			add("unrunnable_recipe", "recipes", rc.ID)
		}
		if len(rc.Inputs) == 0 && !rc.Sited {
			add("source_unbounded", "recipes", rc.ID)
		}
	}

	// The tradable set and the sources of an upkeep commodity.
	sets := tradableSets(c, lk)
	tradable := map[string]bool{}
	for _, set := range sets {
		for id := range set {
			tradable[id] = true
		}
	}
	for _, cm := range c.Commodities {
		if !tradable[cm.ID] && !discardOnly[cm.ID] {
			add("commodity_untraded", "commodities", cm.ID)
		}
	}
	stocked := map[string]bool{}
	for i := range c.Roles {
		for _, it := range c.Roles[i].StartStock {
			if it.Quantity > 0 {
				stocked[it.Commodity] = true
			}
		}
	}
	upkept := map[string]bool{}
	for _, it := range c.World.Upkeep {
		upkept[it.Commodity] = true
	}
	for i := range c.Roles {
		for _, it := range c.Roles[i].Upkeep {
			upkept[it.Commodity] = true
		}
	}
	for i := range c.Regions {
		for _, it := range c.Regions[i].ExtraUpkeep {
			upkept[it.Commodity] = true
		}
	}
	for _, cm := range c.Commodities {
		if upkept[cm.ID] && !cm.Raw && !made[cm.ID] && !stocked[cm.ID] {
			add("upkeep_unsourced", "commodities", cm.ID)
		}
	}

	longestWindow := c.World.WelfareWindow
	for i := range c.Roles {
		ro := &c.Roles[i]
		base := []string{"roles", ro.ID}
		l := longestDurationAny(lk, ro)
		if t, ok := checked.Mul(c.World.WindowCycles, l); ok && t > c.World.WelfareWindow {
			// window_extended: the role's window is longer than the world's
			// because of its longest job (spec-welfare-and-wages.md 11.4).
			add("window_extended", "roles", ro.ID, "recipes")
			longestWindow = max(longestWindow, t)
		}
		gs := groups[ro.ID]
		if len(gs) == 0 {
			// A role with no group is how a role for external agents is
			// written, and draws no warning (spec-config.md 7.3, FR-CFG-066).
			continue
		}
		roleWarnings(c, lk, ro, gs, base, add)
	}
	// entry_share_window: occupation changes may swing (spec-spawner.md 12).
	if p, ok := checked.Mul(c.World.EntryShare, longestWindow); ok && p > entryShareWindow {
		add("entry_share_window", "world", "entry_share")
	}

	out = append(out, regionWarnings(c)...)
	sortWarnings(out)
	return out
}

// longestDurationAny is longestDuration without the bound: the longest Duration
// of the declared recipes on a role's list, whatever its value, for the
// warnings, which read a configuration that may not validate.
func longestDurationAny(lk lookup, r *Role) int64 {
	var longest int64
	for _, id := range r.Recipes {
		if rc := lk.recipe[id]; rc != nil {
			longest = max(longest, rc.Duration)
		}
	}
	return longest
}

// roleWarnings raises the warnings that read a role that has households: the
// cold start, the start stock and money, the strategy and the tool.
func roleWarnings(c *Config, lk lookup, ro *Role, gs []*Group, base []string, add func(string, ...string)) {
	w := c.World

	// cold_start_lockstep (informational): a recipe of Duration above 1 and more
	// than one agent in a group, with no warm start or a warm start that gives
	// the role no job row, so the first jobs of the group start and complete
	// together (spec-world.md 15.2).
	long := false
	for _, id := range ro.Recipes {
		if rc := lk.recipe[id]; rc != nil && rc.Duration > 1 {
			long = true
		}
	}
	if long {
		many := false
		for _, g := range gs {
			if n, _, _, ok := groupShape(w, *g); ok {
				many = many || n > 1
			} else {
				many = many || g.Households > 1
			}
		}
		jobRow := false
		if ws := c.WarmStart; ws != nil {
			for _, wg := range ws.Groups {
				jobRow = jobRow || (wg.Role == ro.ID && len(wg.Jobs) > 0)
			}
		}
		if many && !jobRow {
			add("cold_start_lockstep", "roles", ro.ID, "recipes")
		}
	}

	// strategy_default_unused_trades: the default strategy sells a Trades entry
	// at once (spec-agents.md 3.4.2, 13).
	if len(ro.Trades) > 0 && ro.Strategy == "default" {
		add("strategy_default_unused_trades", "roles", ro.ID, "trades")
	}

	// start_stock_below_target: below one tick of upkeep of a household, or no
	// recipe of the role can start from the start stock (spec-agents.md 13, 19).
	held := map[string]int64{}
	for _, it := range ro.StartStock {
		held[it.Commodity] += int64(it.Quantity)
	}
	short := false
	for _, it := range effectiveUpkeep(c, ro) {
		short = short || held[it.Commodity] < int64(it.Quantity)
	}
	if !short && len(ro.Recipes) > 0 {
		startable, declared := false, false
		for _, id := range ro.Recipes {
			rc := lk.recipe[id]
			if rc == nil {
				continue
			}
			declared = true
			ok := true
			for _, it := range rc.Inputs {
				ok = ok && held[it.Commodity] >= int64(it.Quantity)
			}
			for _, t := range rc.Requires {
				ok = ok && held[t.Commodity] >= 1
			}
			startable = startable || ok
		}
		short = declared && !startable
	}
	if short {
		add("start_stock_below_target", "roles", ro.ID, "start_stock")
	}

	// reserve_exceeds_money: StartMoney is below ReserveTicks times the cost of
	// the effective upkeep at the region's reference prices (spec-agents.md 13).
	for _, g := range gs {
		rg := lk.region[g.Region]
		if rg == nil {
			continue
		}
		cost, ok := listCost(lk, effectiveUpkeep(c, ro), rg.ExtraUpkeep)
		if need, nok := checked.Mul(cost, w.ReserveTicks); ok && nok && int64(ro.StartMoney) < need {
			add("reserve_exceeds_money", "roles", ro.ID, "start_money")
			break
		}
	}

	// tool_unaffordable: StartMoney cannot buy a required tool at the reference
	// price, so the role could never be entered (spec-skill-barriers.md 2.3).
	unaffordable := false
	for _, id := range ro.Recipes {
		rc := lk.recipe[id]
		if rc == nil {
			continue
		}
		for _, t := range rc.Requires {
			if cm := lk.commodity[t.Commodity]; cm != nil && int64(ro.StartMoney) < int64(cm.ReferencePrice) {
				unaffordable = true
			}
		}
	}
	if unaffordable {
		add("tool_unaffordable", "roles", ro.ID, "start_money")
	}
}

// listCost is the cost of the items of two lists at the commodities' reference
// prices, in checked arithmetic. ok is false when a commodity is not declared or
// the sum does not fit.
func listCost(lk lookup, a, b []Item) (cost int64, ok bool) {
	for _, list := range [][]Item{a, b} {
		for _, it := range list {
			cm := lk.commodity[it.Commodity]
			if cm == nil {
				return 0, false
			}
			line, lok := checked.Mul(int64(it.Quantity), int64(cm.ReferencePrice))
			sum, sok := checked.Add(cost, line)
			if !lok || !sok {
				return 0, false
			}
			cost = sum
		}
	}
	return cost, true
}
