package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	fcy "github.com/Whealinghold/economy-sim/internal/facility"
	"github.com/Whealinghold/economy-sim/internal/production"
)

// This file is the recipe clauses of the infrastructure module at work
// (spec-infrastructure.md 4): the limits of a start (4.2.2), the use wear of a site
// (5.1.3) and the completion of a build or a repair (4.3). The hooks in production.go and
// phase_production.go are one nil-safe call each into the methods below; the effect tables
// they read are in infrastructure_step.go.

// Indexes of the build limits in facilityCaps, in the master order of 4.2.2 (limits 2 to 6).
const (
	capGate = iota
	capRoom
	capTarget
	capStore
	capRepair
	capLimits
)

// facilityCaps is what the module tells a start: the capacity of each needed class (limit 1)
// and the build and repair limits that apply (limits 2 to 6), each only when it is set. Plain
// values, so a request allocates nothing.
type facilityCaps struct {
	needs  [MaxRecipeNeeds]int64
	nNeeds int
	caps   [capLimits]int64
	on     [capLimits]bool
	// wantCut asks productionScratch to count the executions the needs took off (14.1,
	// FacilityLimited.Cut); set only under an observer.
	wantCut bool
}

// recipeClauses is the three clauses of a recipe resolved to indexes at construction.
type recipeClauses struct {
	needs  []FacilityClassID
	build  FacilityClassID // -1 when the recipe does not build
	repair FacilityClassID // -1 when the recipe does not repair
	toward RegionID        // the far end of a built route, NoRegion for every edge leaving
	units  int64           // condition one execution of a repair restores
}

// clausesOf resolves the clauses of every recipe of the configuration. The configuration
// passed Validate, so every name is a class (13.1).
func clausesOf(c *Config) []recipeClauses {
	m := c.Modules.Infrastructure
	class := func(id string) FacilityClassID {
		return FacilityClassID(slices.IndexFunc(m.Classes, func(k FacilityClass) bool { return k.ID == id }))
	}
	out := make([]recipeClauses, len(c.Recipes))
	for i := range c.Recipes {
		r := &c.Recipes[i]
		rc := recipeClauses{build: -1, repair: -1, toward: NoRegion}
		for _, id := range r.Needs {
			rc.needs = append(rc.needs, class(id))
		}
		if r.Builds != nil {
			rc.build = class(r.Builds.Class)
			if r.Builds.Toward != "" {
				rc.toward = RegionID(slices.IndexFunc(c.Regions, func(g Region) bool { return g.ID == r.Builds.Toward }))
			}
		}
		if r.Repairs != nil {
			rc.repair, rc.units = class(r.Repairs.Class), r.Repairs.Units
		}
		out[i] = rc
	}
	return out
}

// request fills the facility limits of a start for the principal in the region of the
// executor (4.2.2). It is nil-safe, reads the effect table and the counters of the tick and
// writes only q. A recipe with no clause costs one test.
func (n *infrastructure) request(q *startReq) {
	if n == nil {
		return
	}
	region, rc := q.region, &n.rec[q.recipe]
	q.fac = facilityCaps{wantCut: n.world.obs.supplied} // a request read twice keeps nothing of the first
	for i, c := range rc.needs {
		q.fac.needs[i] = max(n.site[siteKey{q.principal, region, c}]-n.used[ownerClass{q.principal, c}], 0)
	}
	q.fac.nNeeds = len(rc.needs)
	if rc.build >= 0 {
		n.buildLimits(q, region, rc)
	}
	if rc.repair >= 0 {
		q.fac.caps[capRepair], q.fac.on[capRepair] = n.repairCap(q.principal, region, rc), true
	}
}

// buildLimits sets limits 2 to 5 of 4.2.2 for a recipe that builds. The gate, the target and
// the store are all or nothing, so they are set (to 0) only when closed; the room always is.
func (n *infrastructure) buildLimits(q *startReq, region RegionID, rc *recipeClauses) {
	w, c := n.world, &n.cls[rc.build]
	if c.flag >= 0 && !w.regionHas(region, c.flag) {
		q.fac.on[capGate] = true
	}
	q.fac.caps[capRoom], q.fac.on[capRoom] = n.room(q.principal, rc.build), true
	q.fac.on[capTarget] = !n.hasTarget(region, rc)
	if c.kind == kindStore {
		q.fac.on[capStore] = w.storeRefusal(w.agentOf(q.principal), c.store, 0) != nil
	}
}

// hasTarget reports whether a build has something to act on (limit 4): a route needs an edge
// from the region to Toward, or any edge leaving it, and a ground improvement an endowment of
// the class's resource with an effective capacity above 0. The other kinds have no target.
func (n *infrastructure) hasTarget(region RegionID, rc *recipeClauses) bool {
	c := &n.cls[rc.build]
	switch c.kind {
	case kindRoute:
		if rc.toward != NoRegion {
			return n.edge[[2]RegionID{region, rc.toward}]
		}
		for to := range n.regions {
			if n.edge[[2]RegionID{region, RegionID(to)}] {
				return true
			}
		}
		return false
	case kindGround:
		r := n.world.res
		return r != nil && c.resource >= 0 && r.effectiveCapacity(int(region)*n.resources+c.resource) > 0
	}
	return true
}

// room is limit 3: the facilities the principal may still order, the least of what the owner
// cap and the world bound leave, each less the builds already started in the tick, and for a
// store class the stores the agent may still own less its pending store facilities (5.4.1).
// An estate is exempt from the owner cap.
func (n *infrastructure) room(principal AgentID, class FacilityClassID) int64 {
	r := int64(MaxFacilities) - int64(len(n.fac)) - n.orderedAll
	if !n.estateOwner(principal) {
		r = min(r, int64(MaxFacilitiesPerAgent-n.held(principal))-n.ordered[principal])
		if n.cls[class].kind == kindStore {
			w := n.world
			r = min(r, int64(MaxStoresPerAgent-w.sto.owned(principal)-n.pendingStores(principal))-n.ordered[principal])
		}
	}
	return r
}

// pendingStores is the number of store facilities of the owner that are not installed.
func (n *infrastructure) pendingStores(owner AgentID) int {
	p := 0
	for _, id := range n.byOwner[owner] {
		if f := &n.fac[n.index(id)]; n.cls[f.class].kind == kindStore && !f.installed {
			p++
		}
	}
	return p
}

// repairCap is limit 6: the least number of executions whose units cover what is missing
// from the principal's facilities of the class in the region, 0 when nothing is.
func (n *infrastructure) repairCap(principal AgentID, region RegionID, rc *recipeClauses) int64 {
	var missing int64
	for _, id := range n.byOwner[principal] {
		if f := &n.fac[n.index(id)]; f.class == rc.repair && f.region == region {
			missing += n.cls[f.class].durability - f.condition
		}
	}
	return upDiv(missing, rc.units)
}

// upDiv is a / b rounded up for a >= 0, without the overflow of a + b - 1; 0 for a b below 1.
func upDiv(a, b int64) int64 {
	if a <= 0 || b < 1 {
		return 0
	}
	q := a / b
	if a%b != 0 {
		q++
	}
	return q
}

// appendNeedLimits appends limit 1 of 4.2.2, one per needed class, after the tools and before
// the inputs.
func appendNeedLimits(limits []production.Limit, q *startReq) []production.Limit {
	for _, c := range q.fac.needs[:q.fac.nNeeds] {
		limits = append(limits, production.Limit{Cap: c, Reason: limitNoFacility})
	}
	return limits
}

// appendBuildLimits appends limits 2 to 6 of 4.2.2 that the request set, after the tenure
// limits and before the room reason.
func appendBuildLimits(limits []production.Limit, q *startReq) []production.Limit {
	codes := [capLimits]int{limitNotUnlocked, limitFacilityCap, limitNoTarget, limitStoreRefused, limitNothingToRepair}
	for i, on := range q.fac.on {
		if on {
			limits = append(limits, production.Limit{Cap: q.fac.caps[i], Reason: codes[i]})
		}
	}
	return limits
}

// freeOfNeeds is the executions a start would take without the capacity of the needed
// classes, for the count of FacilityLimited.Cut. It is 0 unless the request wants it and
// has needs, so with no observer it costs one test.
func (s *productionScratch) freeOfNeeds(q *startReq) int64 {
	if !q.fac.wantCut || q.fac.nNeeds == 0 {
		return 0
	}
	s.free = s.free[:0]
	for _, l := range s.limits {
		if l.Reason != limitNoFacility {
			s.free = append(s.free, l)
		}
	}
	k, _ := production.Count(q.weight, q.maximum, s.free)
	return k
}

// binding is the index of the needed class that bounded the start: the first with the least
// capacity left, which FacilityLimited.Cut is counted against once, as a refusal is
// (refusalClass). It is 0 for a request with no needs.
func (f *facilityCaps) binding() int {
	b := 0
	for i := 1; i < f.nNeeds; i++ {
		if f.needs[i] < f.needs[b] {
			b = i
		}
	}
	return b
}

// started is what the module does at the start of a job that began with k executions, after
// k is final (4.2.2, 5.1.3): the use wear of each needed class, the running counts of the
// tick that the second evaluation reads, and the counters under an observer, cut being the
// executions the capacity took off (productionScratch.facCut). It is nil-safe.
func (n *infrastructure) started(region RegionID, q *startReq, k, cut int64) {
	if n == nil {
		return
	}
	rc := &n.rec[q.recipe]
	bind := q.fac.binding()
	for i, c := range rc.needs {
		n.wear(q.principal, region, c, k)
		n.used[ownerClass{q.principal, c}] += k
		if n.world.obs.supplied {
			l := &n.limits[n.cell(region, c)]
			l.served += k
			if i == bind {
				l.cut += cut
			}
		}
	}
	if rc.build >= 0 {
		n.ordered[q.principal] += k
		n.orderedAll += k
	}
}

// refused counts a start refused whole for a reason of 4.2.2 under an observer (14.1):
// Refused by one, and Cut by the executions the capacity took off when the reason is
// no_facility. It is nil-safe and ignores every other reason.
func (n *infrastructure) refused(principal AgentID, region RegionID, recipe RecipeID, reason Reason, free int64) {
	if n == nil || !n.world.obs.supplied {
		return
	}
	class, ok := n.refusalClass(principal, region, recipe, reason)
	if !ok {
		return
	}
	l := &n.limits[n.cell(region, class)]
	l.refused++
	if reason == ReasonNoFacility {
		l.cut += free
	}
}

// refusalClass is the class a refusal is counted against: the first needed class with no
// capacity left for no_facility, the class of the clause for the build reasons. It reports
// false for a reason that is not the module's.
func (n *infrastructure) refusalClass(principal AgentID, region RegionID, recipe RecipeID, reason Reason) (FacilityClassID, bool) {
	switch reason {
	case ReasonNoFacility:
		if recipe < 0 || int(recipe) >= len(n.rec) || len(n.rec[recipe].needs) == 0 {
			return 0, false // not a reason this recipe can give: count nothing rather than index past the clauses
		}
		rc := &n.rec[recipe]
		for _, c := range rc.needs {
			if n.facilityRoom(principal, region, c) <= 0 {
				return c, true
			}
		}
		return rc.needs[0], true
	case ReasonNotUnlocked, ReasonFacilityCap, ReasonNoTarget, ReasonStoreRefused:
		return n.clauseClass(recipe, func(rc *recipeClauses) FacilityClassID { return rc.build })
	case ReasonNothingToRepair:
		return n.clauseClass(recipe, func(rc *recipeClauses) FacilityClassID { return rc.repair })
	}
	return 0, false
}

// clauseClass is the class a recipe's clause names, false for a recipe outside the
// configuration or one with no such clause (the index is -1).
func (n *infrastructure) clauseClass(recipe RecipeID, pick func(*recipeClauses) FacilityClassID) (FacilityClassID, bool) {
	if recipe < 0 || int(recipe) >= len(n.rec) {
		return 0, false
	}
	c := pick(&n.rec[recipe])
	return c, c >= 0
}

// wear assigns k executions of a start to the owner's facilities of the class in the region,
// most worn first by the condition of the effect table, ties to the lower ID, each at most
// its cap of the table less what earlier starts of the tick put on it, and takes WearPerUse
// from each for what it serves (5.1.3). A start never asks for more than the capacity, so
// nothing is left over.
func (n *infrastructure) wear(owner AgentID, region RegionID, class FacilityClassID, k int64) {
	c := &n.cls[class]
	n.scratch = n.scratch[:0]
	for _, id := range n.byOwner[owner] {
		if f := &n.fac[n.index(id)]; f.class == class && f.region == region {
			n.scratch = append(n.scratch, fcy.Fac{Ref: int(f.id), Table: f.tabCond, Condition: f.condition, Cap: f.tabCap,
				Served: f.served, WearPerUse: c.wearPerUse})
		}
	}
	var rest int64
	n.placed, rest = fcy.Assign(n.placed, k, n.scratch)
	if rest != 0 {
		invariant("a start took more executions of a site than its capacity")
	}
	for _, g := range n.scratch {
		f := &n.fac[n.index(FacilityID(g.Ref))]
		f.condition, f.served = g.Condition, g.Served
	}
}

// facilityRoom is the executions of the class that the principal's facilities in the region
// can still serve this tick: the capacity of the table less the running count (5.1.2). The hire
// formation of the employment module calls it (spec-employment.md 5.5; N-381); nothing in this
// tree does yet.
func (n *infrastructure) facilityRoom(principal AgentID, region RegionID, class FacilityClassID) int64 {
	return max(n.site[siteKey{principal, region, class}]-n.used[ownerClass{principal, class}], 0)
}

// standingReason is the first reason of the master order of 4.2.2, other than no_input and
// no_tool, that would refuse a start of one execution of the recipe for the principal in the
// region, or "" (spec-saving-and-investment.md 5.2): the derivation of the per-recipe
// Standing. The money limits (rent_funds) are judged at the money held, which a request does
// not read. Nothing in this tree calls it yet; the saving module does.
func (n *infrastructure) standingReason(principal AgentID, region RegionID, recipe RecipeID) Reason {
	if n == nil {
		return ""
	}
	q := startReq{recipe: recipe, principal: principal, region: region}
	n.request(&q)
	if slices.ContainsFunc(q.fac.needs[:q.fac.nNeeds], func(c int64) bool { return c <= 0 }) {
		return ReasonNoFacility
	}
	reasons := [capLimits]Reason{ReasonNotUnlocked, ReasonFacilityCap, ReasonNoTarget, ReasonStoreRefused, ReasonNothingToRepair}
	for i, on := range q.fac.on {
		if on && q.fac.caps[i] <= 0 {
			return reasons[i]
		}
	}
	return ""
}

// completed is what the module does at the completion of a job (4.3): a build creates the
// facilities, a repair restores condition. It is nil-safe and acts on the principal, whose
// region is the executor's (4.2.1). The effects enter the tables at the next P8.3b (4.3.3).
func (n *infrastructure) completed(region RegionID, c *completion) {
	if n == nil {
		return
	}
	switch rc := &n.rec[c.recipe]; {
	case rc.build >= 0:
		n.completeBuild(c.principal, region, rc, c.executions)
	case rc.repair >= 0:
		n.completeRepair(c.principal, region, rc, c.executions)
	}
}

// completeBuild creates the facilities of k executions that fit the owner cap and the world
// bound, in ascending new IDs, and loses the rest (4.3.1, 4.3.4). One FacilityChanged (built)
// follows per facility, after RecipeExecuted.
func (n *infrastructure) completeBuild(principal AgentID, region RegionID, rc *recipeClauses, k int64) {
	w := n.world
	room := int64(MaxFacilities) - int64(len(n.fac))
	if !n.estateOwner(principal) {
		room = min(room, int64(MaxFacilitiesPerAgent-n.held(principal)))
	}
	made := max(min(k, room), 0)
	for range made {
		f := n.create(rc.build, region, rc.toward, principal, n.cls[rc.build].durability, w.Tick())
		w.reportFacility(f, FacilityBuilt, CauseRecipe, 0, 0)
	}
	n.void(region, rc.build, k-made)
}

// completeRepair restores k * Units of condition to the principal's facilities of the class in
// the region, most worn first, ties to the lower ID, each to at most its durability; what is
// left over is lost (4.3.2, 4.3.4). The events follow in ascending ID.
func (n *infrastructure) completeRepair(principal AgentID, region RegionID, rc *recipeClauses, k int64) {
	n.needs = n.needs[:0]
	for _, id := range n.byOwner[principal] {
		if f := &n.fac[n.index(id)]; f.class == rc.repair && f.region == region {
			n.needs = append(n.needs, fcy.Need{Ref: int(f.id), Condition: f.condition, Missing: n.cls[f.class].durability - f.condition})
		}
	}
	units, ok := checked.Mul(k, rc.units)
	if !ok {
		invariant("the units of a repair do not fit after validation")
	}
	var left int64
	n.placed, left = fcy.Distribute(n.placed, units, n.needs)
	took := upDiv(units-left, rc.units) // the executions that restored something
	slices.SortFunc(n.placed, func(a, b fcy.Placement) int { return a.Ref - b.Ref })
	for _, p := range n.placed {
		f := &n.fac[n.index(FacilityID(p.Ref))]
		before := f.condition
		f.condition += p.N
		n.world.reportFacility(f, FacilityRepaired, CauseRecipe, before, 0)
	}
	n.void(region, rc.repair, k-took)
}

// void counts executions that did nothing at a completion (4.3.4), under an observer.
func (n *infrastructure) void(region RegionID, class FacilityClassID, executions int64) {
	if executions > 0 && n.world.obs.supplied {
		n.limits[n.cell(region, class)].voided += executions
	}
}

// facilityDecay is the facility classes, in ascending ID, whose Decay or WearPerUse is above
// 0: the informational report of the viability programme (spec-infrastructure.md 13.4). It is
// nil with the module off.
func facilityDecay(c *Config) []string {
	if c.Modules.Infrastructure == nil {
		return nil
	}
	var out []string
	for i := range c.Modules.Infrastructure.Classes {
		if k := &c.Modules.Infrastructure.Classes[i]; k.Decay > 0 || k.WearPerUse > 0 {
			out = append(out, k.ID)
		}
	}
	return out
}

// warmJob refuses the job in progress of a warm-start agent whose recipe needs a site class
// the pack gives its group no initial facility of: the warm start creates no job that needs a
// facility that does not exist, a *ViabilityError that matches ErrNotViable, as for a missing
// tool (spec-infrastructure.md 13.4). The initial facilities are made after the warm start, so
// the check reads the rows of the configuration. It is nil-safe.
func (n *infrastructure) warmJob(a *agent) error {
	if n == nil {
		return nil
	}
	w := n.world
	region, role := w.cfg.Regions[w.regionOf(a)].ID, w.cfg.Roles[a.role].ID
	for _, c := range n.rec[a.job.recipe].needs {
		id := w.cfg.Modules.Infrastructure.Classes[c].ID
		if !slices.ContainsFunc(w.cfg.Modules.Infrastructure.Initial, func(r InitialFacility) bool {
			return r.Class == id && r.Region == region && r.Role == role && r.Count > 0
		}) {
			return &ViabilityError{Rule: "facility_missing", Path: []string{"warm_start", "groups", region, role},
				Value: w.cfg.Recipes[a.job.recipe].ID, Bound: id}
		}
	}
	return nil
}
