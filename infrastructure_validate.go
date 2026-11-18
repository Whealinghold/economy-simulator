package economysim

// This file is the validation of the infrastructure module (spec-infrastructure.md
// 13.1; the continuation file of infrastructure.go, architecture.md 2.6): the rules
// of the classes, the initial rows and the recipe clauses that Validate checks. The
// warnings of 13.2 are in infrastructure_report.go.

// The kinds of a facility class (spec-infrastructure.md 3.1.1): a closed set of
// constants, not identifiers. A rule constant beside its code (architecture.md 9.6).
const (
	kindSite   = "site"
	kindRoute  = "route"
	kindGround = "ground"
	kindStore  = "store"
)

// facilityPath is the path of the module's block.
var facilityPath = []string{"modules", "infrastructure"}

// validateInfrastructure checks the infrastructure module and the clauses of the
// recipes (spec-infrastructure.md 13.1, FR-CAP-003, FR-CAP-004, FR-CAP-010). The
// rules run in phases and a phase that finds an error ends the run, so the first
// error is the same for the same mistake: the bounds, the shape, the references, the
// module dependencies, then the counts. With the module off only a clause is
// reported.
func (v *validator) validateInfrastructure() {
	m := v.c.Modules.Infrastructure
	if m == nil {
		v.facilityClausesOff()
		return
	}
	steps := []func() bool{
		func() bool { return v.facilityBounds(m) },
		func() bool { return v.facilityShapes(m) },
		func() bool { return v.facilityReferences(m) },
		func() bool { return v.facilityDependencies(m) },
		func() bool { return v.facilityCounts(m) },
	}
	for _, step := range steps {
		if !step() {
			return
		}
	}
}

// facilityClause reports whether the recipe builds or repairs a facility.
func facilityClause(r *Recipe) bool { return r.Builds != nil || r.Repairs != nil }

// facilityClausesOff reports ErrModuleDependency for a recipe with a clause while the
// module is off (spec-infrastructure.md 13.1).
func (v *validator) facilityClausesOff() {
	for i := range v.c.Recipes {
		rc := &v.c.Recipes[i]
		base := []string{"recipes", rc.ID}
		if len(rc.Needs) > 0 {
			v.rep.add(ErrModuleDependency, path(base, "needs"), itoa(len(rc.Needs)), "infrastructure")
		}
		if rc.Builds != nil {
			v.rep.add(ErrModuleDependency, path(base, "builds"), rc.Builds.Class, "infrastructure")
		}
		if rc.Repairs != nil {
			v.rep.add(ErrModuleDependency, path(base, "repairs"), rc.Repairs.Class, "infrastructure")
		}
	}
}

// facilityClassPath is the path of a class: its ID when it is a valid one, else its
// position.
func facilityClassPath(i int, c FacilityClass) []string {
	if validID(c.ID) {
		return []string{"modules", "infrastructure", "class", c.ID}
	}
	return []string{"modules", "infrastructure", "classes[" + itoa(i) + "]"}
}

// facilityRowPath is the path of an initial row: its position.
func facilityRowPath(i int) []string {
	return []string{"modules", "infrastructure", "initial[" + itoa(i) + "]"}
}

// facilityBounds is phase 1: the counts, each class's ID, kind and numbers against
// their bounds, each row's count, and the numbers of the clauses.
func (v *validator) facilityBounds(m *Infrastructure) bool {
	if len(m.Classes) > MaxFacilityClasses {
		v.rep.add(ErrOutOfBounds, path(facilityPath, "classes"), itoa(len(m.Classes)), itoa(MaxFacilityClasses))
		return false
	}
	if len(m.Initial) > MaxInitialRows {
		v.rep.add(ErrOutOfBounds, path(facilityPath, "initial"), itoa(len(m.Initial)), itoa(MaxInitialRows))
		return false
	}
	n := len(v.rep.errs)
	seen := make(map[string]bool, len(m.Classes)) // a membership test only
	for i, c := range m.Classes {
		p := facilityClassPath(i, c)
		switch {
		case !validID(c.ID):
			v.rep.add(ErrInvalidID, path(p, "id"), c.ID, "")
		case seen[c.ID]:
			v.rep.add(ErrDuplicateID, path(p, "id"), c.ID, "")
		}
		seen[c.ID] = true
		v.facilityClassBounds(p, c)
	}
	for i, r := range m.Initial {
		v.inRange(path(facilityRowPath(i), "count"), r.Count, 1, MaxInitialCount)
	}
	v.facilityClauseBounds()
	return len(v.rep.errs) == n
}

// facilityClassBounds checks the kind and the numbers of one class against the
// bounds of spec-infrastructure.md 12.2. A field a kind does not use is judged by
// the shape rule, so every number is checked here over its widest range.
func (v *validator) facilityClassBounds(p []string, c FacilityClass) {
	switch c.Kind {
	case kindSite, kindRoute, kindGround, kindStore:
	default:
		v.rep.add(ErrOutOfBounds, path(p, "kind"), c.Kind, "site, route, ground or store")
	}
	dur := int64(MaxDurability)
	if v.inRange(path(p, "durability"), c.Durability, 2, MaxDurability) {
		dur = c.Durability
	}
	v.ranges(p, []fieldRange{
		{"decay", c.Decay, 0, dur - 1},
		{"wear_per_use", c.WearPerUse, 0, dur},
		{"capacity", c.Capacity, 0, MaxFacilityCapacity},
		{"boost_bp", c.BoostBP, 0, MaxBoostBP},
	})
	if len(c.Modes) > MaxRouteModes {
		v.rep.add(ErrOutOfBounds, path(p, "modes"), itoa(len(c.Modes)), itoa(MaxRouteModes))
	}
	v.label(p, "label", c.Label)
}

// facilityClauseBounds checks the numbers of the recipes' clauses: the count of
// needs and the units of a repair against the widest durability.
func (v *validator) facilityClauseBounds() {
	for i := range v.c.Recipes {
		rc := &v.c.Recipes[i]
		base := []string{"recipes", rc.ID}
		if len(rc.Needs) > MaxRecipeNeeds {
			v.rep.add(ErrOutOfBounds, path(base, "needs"), itoa(len(rc.Needs)), itoa(MaxRecipeNeeds))
		}
		if rc.Repairs != nil {
			v.inRange(path(base, "repairs", "units"), rc.Repairs.Units, 1, MaxDurability)
		}
	}
}

// facilityUse says which fields a kind uses (spec-infrastructure.md 3.1.2).
type facilityUse struct {
	wear, capacity, boost, modes, resource, store bool
}

// facilityUses returns the fields of a kind; the zero value for an unknown kind,
// which phase 1 rejected.
func facilityUses(kind string) facilityUse {
	switch kind {
	case kindSite:
		return facilityUse{wear: true, capacity: true}
	case kindRoute:
		return facilityUse{boost: true, modes: true}
	case kindGround:
		return facilityUse{boost: true, resource: true}
	case kindStore:
		return facilityUse{store: true}
	}
	return facilityUse{}
}

// facilityShapes is phase 2: a class field that its kind does not use is zero and a
// required one is set (ErrFacilityShape), the rows are one per key, and the
// clauses fit their classes.
func (v *validator) facilityShapes(m *Infrastructure) bool {
	n := len(v.rep.errs)
	for i, c := range m.Classes {
		v.facilityClassShape(facilityClassPath(i, c), c)
	}
	classes := facilityClassKinds(m)
	seen := make(map[[4]string]bool, len(m.Initial)) // a membership test only
	for i, r := range m.Initial {
		p := facilityRowPath(i)
		key := [4]string{r.Region, r.Class, r.Toward, r.Role}
		if seen[key] {
			v.rep.add(ErrDuplicateID, p, r.Class, "")
		}
		seen[key] = true
		v.facilityRowShape(p, r, classes)
	}
	v.facilityClauseShapes(classes)
	return len(v.rep.errs) == n
}

// facilityClassKinds returns the kind of each class by ID; a repeated ID keeps its
// first (phase 1 reported it).
func facilityClassKinds(m *Infrastructure) map[string]string {
	out := make(map[string]string, len(m.Classes))
	for _, c := range m.Classes {
		if _, ok := out[c.ID]; !ok {
			out[c.ID] = c.Kind
		}
	}
	return out
}

// facilityClassShape reports a field the kind does not use that is not zero and a
// field it requires that is missing.
func (v *validator) facilityClassShape(p []string, c FacilityClass) {
	u := facilityUses(c.Kind)
	rules := []struct {
		name         string
		used, set    bool
		required     bool
		value, bound string
	}{
		{"wear_per_use", u.wear, c.WearPerUse != 0, false, i64(c.WearPerUse), "0"},
		{"capacity", u.capacity, c.Capacity != 0, true, i64(c.Capacity), "1"},
		{"boost_bp", u.boost, c.BoostBP != 0, true, i64(c.BoostBP), "1"},
		{"modes", u.modes, len(c.Modes) != 0, true, itoa(len(c.Modes)), "1"},
		{"resource", u.resource, c.Resource != "", true, c.Resource, "1"},
		{"store", u.store, c.Store != "", true, c.Store, "1"},
	}
	for _, r := range rules {
		switch {
		case !r.used && r.set:
			v.rep.add(ErrFacilityShape, path(p, r.name), r.value, "0")
		case r.used && r.required && !r.set:
			v.rep.add(ErrFacilityShape, path(p, r.name), r.value, r.bound)
		}
	}
}

// facilityRowShape reports a Toward on a class that is not a route and a store row
// with no role (spec-infrastructure.md 3.4.2). An unknown class is phase 3's.
func (v *validator) facilityRowShape(p []string, r InitialFacility, classes map[string]string) {
	kind, known := classes[r.Class]
	switch {
	case !known:
	case r.Toward != "" && kind != kindRoute:
		v.rep.add(ErrFacilityShape, path(p, "toward"), r.Toward, "")
	case kind == kindStore && r.Role == "":
		v.rep.add(ErrFacilityShape, path(p, "role"), "", "1")
	}
}

// facilityClauseShapes checks the rules B1, B3, B4, B6 and B7 of
// spec-infrastructure.md 4.1.2 for every recipe.
func (v *validator) facilityClauseShapes(classes map[string]string) {
	for i := range v.c.Recipes {
		rc := &v.c.Recipes[i]
		base := []string{"recipes", rc.ID}
		v.facilityBuildShape(base, rc, classes)
		v.facilityNeedsShape(base, rc, classes)
		if rc.TraineeEligible && (facilityClause(rc) || len(rc.Needs) > 0) {
			v.rep.add(ErrEligibleHasTool, path(base, "trainee_eligible"), "true", "false")
		}
	}
}

// facilityBuildShape checks B1 (one clause), B3 (a discard output only) and B4 (a
// Toward on a route class only).
func (v *validator) facilityBuildShape(base []string, rc *Recipe, classes map[string]string) {
	if rc.Builds != nil && rc.Repairs != nil {
		v.rep.add(ErrFacilityShape, base, "builds and repairs", "one")
	}
	if facilityClause(rc) {
		for _, o := range rc.Outputs {
			if !o.Discard {
				v.rep.add(ErrFacilityShape, path(base, "outputs", o.Commodity), o.Commodity, "discard")
			}
		}
	}
	if b := rc.Builds; b != nil && b.Toward != "" && classes[b.Class] != kindRoute {
		v.rep.add(ErrFacilityShape, path(base, "builds", "toward"), b.Toward, "")
	}
}

// facilityNeedsShape checks B6: each class of Needs is a site and is listed once.
func (v *validator) facilityNeedsShape(base []string, rc *Recipe, classes map[string]string) {
	seen := make(map[string]bool, len(rc.Needs)) // a membership test only
	for _, id := range rc.Needs {
		kind, known := classes[id]
		switch {
		case seen[id]:
			v.rep.add(ErrDuplicateID, path(base, "needs"), id, "")
		case known && kind != kindSite:
			v.rep.add(ErrFacilityShape, path(base, "needs"), id, kindSite)
		}
		seen[id] = true
	}
}

// facilityRefSets are the names the references of phase 3 are looked up in. A set of
// a module that is off is nil and its references are phase 4's.
type facilityRefSets struct {
	classes, modes, stores map[string]bool
}

// facilityReferences is phase 3: every name of a class, a clause or a row is
// defined (spec-infrastructure.md 13.1).
func (v *validator) facilityReferences(m *Infrastructure) bool {
	n := len(v.rep.errs)
	refs := v.facilityRefs(m)
	for i, c := range m.Classes {
		v.facilityClassRefs(facilityClassPath(i, c), c, refs)
	}
	for i, r := range m.Initial {
		v.facilityRowRefs(facilityRowPath(i), r, refs)
	}
	for i := range v.c.Recipes {
		rc := &v.c.Recipes[i]
		v.facilityClauseRefs([]string{"recipes", rc.ID}, rc, refs)
	}
	return len(v.rep.errs) == n
}

// facilityRefs builds the sets of class IDs, transport modes and store classes.
func (v *validator) facilityRefs(m *Infrastructure) facilityRefSets {
	refs := facilityRefSets{classes: make(map[string]bool, len(m.Classes))}
	for _, c := range m.Classes {
		refs.classes[c.ID] = true
	}
	if t := v.c.Modules.Transport; t != nil {
		refs.modes = make(map[string]bool, len(t.Modes))
		for _, md := range t.Modes {
			refs.modes[md.ID] = true
		}
	}
	if s := v.c.Modules.Storage; s != nil {
		refs.stores = s.classSet()
	}
	return refs
}

// facilityClassRefs checks the mode, resource and store class a class names. A
// module that is off has no set to look in and is reported as a dependency.
func (v *validator) facilityClassRefs(p []string, c FacilityClass, refs facilityRefSets) {
	if refs.modes != nil {
		for j, id := range c.Modes {
			if !refs.modes[id] {
				v.rep.add(ErrUnknownMode, path(p, "modes["+itoa(j)+"]"), id, "")
			}
		}
	}
	if c.Resource != "" && v.c.Modules.Resources != nil && v.lk.resource[c.Resource] == nil {
		v.rep.add(ErrUnknownResource, path(p, "resource"), c.Resource, "")
	}
	if c.Store != "" && refs.stores != nil && !refs.stores[c.Store] {
		v.rep.add(ErrUnknownStoreClass, path(p, "store"), c.Store, "")
	}
}

// facilityRowRefs checks the class, regions and role of an initial row, and that a
// route row's pair is joined by an edge.
func (v *validator) facilityRowRefs(p []string, r InitialFacility, refs facilityRefSets) {
	if !refs.classes[r.Class] {
		v.rep.add(ErrUnknownFacilityClass, path(p, "class"), r.Class, "")
	}
	v.facilityRegionRef(path(p, "region"), r.Region)
	if r.Toward != "" {
		v.facilityRegionRef(path(p, "toward"), r.Toward)
		switch {
		case r.Toward == r.Region:
			v.rep.add(ErrSameRegion, path(p, "toward"), r.Toward, "")
		case v.region[r.Region] && v.region[r.Toward] && !v.joined(r.Region, r.Toward):
			v.rep.add(ErrUnknownEdge, path(p, "toward"), r.Toward, r.Region)
		}
	}
	if r.Role != "" && !v.role[r.Role] {
		v.rep.add(ErrUnknownRole, path(p, "role"), r.Role, "")
	}
}

// facilityRegionRef reports a region that is not defined or is an outside node.
func (v *validator) facilityRegionRef(p []string, id string) {
	switch {
	case !v.region[id]:
		v.rep.add(ErrUnknownRegion, p, id, "")
	case v.outsideSet()[id]:
		v.rep.add(ErrOutsideNode, p, id, "")
	}
}

// joined reports whether an edge joins the two regions in either direction.
func (v *validator) joined(a, b string) bool {
	for _, e := range v.c.Edges {
		if (e.From == a && e.To == b) || (e.From == b && e.To == a) {
			return true
		}
	}
	return false
}

// facilityClauseRefs checks the classes a recipe names and a build's Toward.
func (v *validator) facilityClauseRefs(base []string, rc *Recipe, refs facilityRefSets) {
	for _, id := range rc.Needs {
		if !refs.classes[id] {
			v.rep.add(ErrUnknownFacilityClass, path(base, "needs"), id, "")
		}
	}
	if b := rc.Builds; b != nil {
		if !refs.classes[b.Class] {
			v.rep.add(ErrUnknownFacilityClass, path(base, "builds", "class"), b.Class, "")
		}
		if b.Toward != "" {
			v.facilityRegionRef(path(base, "builds", "toward"), b.Toward)
		}
	}
	if r := rc.Repairs; r != nil && !refs.classes[r.Class] {
		v.rep.add(ErrUnknownFacilityClass, path(base, "repairs", "class"), r.Class, "")
	}
}

// facilityDependencies is phase 4: a route class needs the transport module, a
// ground class the resources module, a store class the storage module and a Needs
// flag the settlements module and a flag some class carries.
func (v *validator) facilityDependencies(m *Infrastructure) bool {
	n := len(v.rep.errs)
	for i, c := range m.Classes {
		p := facilityClassPath(i, c)
		switch {
		case c.Kind == kindRoute && v.c.Modules.Transport == nil:
			v.rep.add(ErrModuleDependency, path(p, "kind"), c.Kind, "transport")
		case c.Kind == kindGround && v.c.Modules.Resources == nil:
			v.rep.add(ErrModuleDependency, path(p, "kind"), c.Kind, "resources")
		case c.Kind == kindStore && v.c.Modules.Storage == nil:
			v.rep.add(ErrModuleDependency, path(p, "kind"), c.Kind, "storage")
		}
		if c.Needs != "" {
			v.settlementFlagRule(path(p, "needs"), c.Needs)
		}
	}
	return len(v.rep.errs) == n
}

// facilityCounts is phase 5: a clause has an input and a repair's units fit the
// class's durability (B2, B5), every row has a holder, and the rows fit the counts
// of the world and of an owner.
func (v *validator) facilityCounts(m *Infrastructure) bool {
	n := len(v.rep.errs)
	durability := make(map[string]int64, len(m.Classes))
	for _, c := range m.Classes {
		if _, ok := durability[c.ID]; !ok {
			durability[c.ID] = c.Durability
		}
	}
	for i := range v.c.Recipes {
		rc := &v.c.Recipes[i]
		base := []string{"recipes", rc.ID}
		if facilityClause(rc) && len(rc.Inputs) == 0 {
			v.rep.add(ErrFacilityNoCost, path(base, "inputs"), "0", "1")
		}
		if r := rc.Repairs; r != nil {
			v.inRange(path(base, "repairs", "units"), r.Units, 1, durability[r.Class])
		}
	}
	v.facilityHolders(m)
	return len(v.rep.errs) == n
}

// facilityHolders reports a row whose role has no agent in the region (or, for a
// store row, no base room), a sum of rows above MaxFacilities and a round robin that
// gives one agent more than MaxFacilitiesPerAgent (spec-infrastructure.md 3.2.3,
// 3.4.2). The agents of a group are those of spec-scale.md 3.2.
func (v *validator) facilityHolders(m *Infrastructure) {
	kinds := facilityClassKinds(m)
	agents := make(map[[2]string]int64, len(v.c.Groups)) // lookup only
	for _, g := range v.c.Groups {
		if a, _, _, ok := groupShape(v.c.World, g); ok {
			// A repeated (region, role) is reported by phase 3 of Validate; its
			// agents add up, as in the settlement census, so the answer does not
			// depend on which of the copies comes last.
			agents[[2]string{g.Region, g.Role}] += a
		}
	}
	perAgent := make(map[[2]string]int64, len(m.Initial)) // the first agent's count; lookup only
	var total int64
	for i, r := range m.Initial {
		p := facilityRowPath(i)
		total += r.Count
		if r.Role == "" {
			continue
		}
		key := [2]string{r.Region, r.Role}
		n := agents[key]
		if n < 1 || (kinds[r.Class] == kindStore && !v.hasBaseRoom(r.Role)) {
			v.rep.add(ErrFacilityHolder, path(p, "role"), r.Role, r.Region)
			continue
		}
		perAgent[key] += (r.Count + n - 1) / n
		if perAgent[key] > MaxFacilitiesPerAgent {
			v.rep.add(ErrFacilityCount, path(p, "count"), i64(perAgent[key]), itoa(MaxFacilitiesPerAgent))
		}
	}
	if total > MaxFacilities {
		v.rep.add(ErrFacilityCount, path(facilityPath, "initial"), i64(total), itoa(MaxFacilities))
	}
}

// hasBaseRoom reports whether the role has a base room under the storage module
// (spec-storage.md 3.3.1).
func (v *validator) hasBaseRoom(role string) bool {
	base, _ := v.c.Modules.Storage.baseOf(role)
	return base != nil
}
