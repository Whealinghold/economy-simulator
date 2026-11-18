package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/institution"
)

// This file is the validation of the institutions module (spec-institutions.md 10; the
// continuation file of institutions.go, architecture.md 2.6): the rules of 10.1 that
// Validate checks. The warnings of 10.2 are in institutions_warn.go.

// orgBase is the path of the module in a configuration.
var orgBase = []string{"modules", "institutions"}

// orgBodyPath, orgRightPath and orgControlPath are the paths of element i of a list of
// the module, or of one key of it.
func orgBodyPath(i int, key ...string) []string {
	return path(orgBase, append([]string{"bodies[" + itoa(i) + "]"}, key...)...)
}

func orgRightPath(i int, key ...string) []string {
	return path(orgBase, append([]string{"rights[" + itoa(i) + "]"}, key...)...)
}

func orgControlPath(i int, key ...string) []string {
	return path(orgBase, append([]string{"controls[" + itoa(i) + "]"}, key...)...)
}

func orgLimitPath(i int, key ...string) []string {
	return path(orgBase, append([]string{"rate_limits[" + itoa(i) + "]"}, key...)...)
}

// validateInstitutions checks the institutions module (spec-institutions.md 10.1,
// FR-ORG-002 to 004, 047). It reports nothing when the module is off. The rules run
// in phases and a phase that finds an error ends the run, so the first error is the
// same for the same mistake: the counts, then the IDs and their duplicates, the
// module dependencies, the references, the shapes, the overlap and last the bounds.
func (v *validator) validateInstitutions() {
	m := v.c.Modules.Institutions
	if m == nil {
		return
	}
	steps := []func(*Institutions) bool{
		v.orgCounts, v.orgIDs, v.orgDependencies, v.orgReferences, v.orgShapes, v.orgOverlap, v.orgBounds,
	}
	for _, step := range steps {
		if !step(m) {
			return
		}
	}
}

// orgCounts is ErrTooMany: more bodies, rights or controls than the bound; the controls
// and the rate limits share MaxControls.
func (v *validator) orgCounts(m *Institutions) bool {
	n := len(v.rep.errs)
	if len(m.Bodies) > MaxBodies {
		v.rep.add(ErrTooMany, path(orgBase, "bodies"), itoa(len(m.Bodies)), itoa(MaxBodies))
	}
	if len(m.Rights) > MaxRights {
		v.rep.add(ErrTooMany, path(orgBase, "rights"), itoa(len(m.Rights)), itoa(MaxRights))
	}
	if total := len(m.Controls) + len(m.RateLimits); total > MaxControls {
		v.rep.add(ErrTooMany, path(orgBase, "controls"), itoa(total), itoa(MaxControls))
	}
	return len(v.rep.errs) == n
}

// orgIDs is ErrInvalidID and ErrDuplicateID: an ID that does not match spec-config.md
// 3.3 or is used twice in its namespace (a control and a rate limit share one), an
// entry twice in a list of a body, two
// rights with one (activity, subject, scope key), two controls with one (commodity,
// scope key) and two rate limits with one scope key.
func (v *validator) orgIDs(m *Institutions) bool {
	n := len(v.rep.errs)
	bodies := make(map[string]bool, len(m.Bodies)) // membership tests only
	for i := range m.Bodies {
		v.orgID(bodies, orgBodyPath(i, "id"), m.Bodies[i].ID)
		v.orgBodyLists(i, m.Bodies[i])
	}
	rights, keys := make(map[string]bool, len(m.Rights)), make(map[[4]string]bool, len(m.Rights))
	for i, r := range m.Rights {
		v.orgID(rights, orgRightPath(i, "id"), r.ID)
		v.orgKey(keys, orgRightPath(i, "subject"), [4]string{r.Activity, r.Subject, r.Region, r.Jurisdiction})
	}
	controls, rows := make(map[string]bool, len(m.Controls)+len(m.RateLimits)), make(map[[4]string]bool)
	for i, c := range m.Controls {
		v.orgID(controls, orgControlPath(i, "id"), c.ID)
		v.orgKey(rows, orgControlPath(i, "commodity"), [4]string{"control", c.Commodity, c.Region, c.Jurisdiction})
	}
	for i, l := range m.RateLimits {
		v.orgID(controls, orgLimitPath(i, "id"), l.ID)
		v.orgKey(rows, orgLimitPath(i, "region"), [4]string{"rate_limit", "", l.Region, l.Jurisdiction})
	}
	return len(v.rep.errs) == n
}

// orgBodyLists reports an entry that a body lists twice in its Roles, Regions or
// Jurisdictions. A repeat would build the same world as the single entry yet hash
// differently (canonical sorts a list and does not drop a repeat), so it is a mistake and
// not a second spelling.
func (v *validator) orgBodyLists(i int, b Body) {
	for _, l := range []struct {
		key  string
		list []string
	}{{"roles", b.Roles}, {"regions", b.Regions}, {"jurisdictions", b.Jurisdictions}} {
		seen := make(map[string]bool, len(l.list)) // membership tests only
		for j, id := range l.list {
			if seen[id] {
				v.rep.add(ErrDuplicateID, orgBodyPath(i, l.key+"["+itoa(j)+"]"), id, "")
			}
			seen[id] = true
		}
	}
}

// orgID reports an ID that is not valid or is already in seen, and records it.
func (v *validator) orgID(seen map[string]bool, p []string, id string) {
	switch {
	case !validID(id):
		v.rep.add(ErrInvalidID, p, id, "")
	case seen[id]:
		v.rep.add(ErrDuplicateID, p, id, "")
	}
	seen[id] = true
}

// orgKey reports a key that is already in seen, and records it.
func (v *validator) orgKey(seen map[[4]string]bool, p []string, key [4]string) {
	if seen[key] {
		v.rep.add(ErrDuplicateID, p, key[1]+key[2]+key[3], "")
	}
	seen[key] = true
}

// orgDependencies is ErrModuleDependency: a jurisdiction named, or a right, control or
// rate limit keyed by one, while Modules.Polity is absent; a Needs while
// Modules.Settlements is absent; a rate limit while Modules.Credit is absent.
func (v *validator) orgDependencies(m *Institutions) bool {
	n := len(v.rep.errs)
	noPolity, noSettlements := v.c.Modules.Polity == nil, v.c.Modules.Settlements == nil
	for i, b := range m.Bodies {
		v.orgNeed(noPolity, orgBodyPath(i, "jurisdictions"), orgFirst(b.Jurisdictions), "polity")
		v.orgNeed(noSettlements, orgBodyPath(i, "needs"), b.Needs, "settlements")
	}
	for i, r := range m.Rights {
		v.orgNeed(noPolity, orgRightPath(i, "jurisdiction"), r.Jurisdiction, "polity")
	}
	for i, c := range m.Controls {
		v.orgNeed(noPolity, orgControlPath(i, "jurisdiction"), c.Jurisdiction, "polity")
	}
	for i, l := range m.RateLimits {
		v.orgNeed(noPolity, orgLimitPath(i, "jurisdiction"), l.Jurisdiction, "polity")
		v.orgNeed(v.c.Modules.Credit == nil, orgLimitPath(i, "cap_bp"), i64(l.CapBP), "credit")
	}
	return len(v.rep.errs) == n
}

// orgFirst is the first of a list of names, "" for none.
func orgFirst(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

// orgNeed reports ErrModuleDependency at p when the module is off and the value names
// something of it (a value of "" does not).
func (v *validator) orgNeed(off bool, p []string, value, module string) {
	if off && value != "" {
		v.rep.add(ErrModuleDependency, p, value, module)
	}
}

// orgReferences is the unknown-reference rules: a role, region, jurisdiction, settlement
// flag, recipe, commodity or body that names nothing, and a region that is an outside
// node. A right's subject that is defined as another kind is left to the shape rule.
func (v *validator) orgReferences(m *Institutions) bool {
	n := len(v.rep.errs)
	juris := v.orgJurisdictions()
	for i, b := range m.Bodies {
		v.orgBodyRefs(i, b, juris)
	}
	bodies := make(map[string]bool, len(m.Bodies))
	for _, b := range m.Bodies {
		bodies[b.ID] = true
	}
	for i, r := range m.Rights {
		if r.Body != "" {
			v.orgKnown(bodies[r.Body], ErrUnknownBody, orgRightPath(i, "body"), r.Body)
		}
		v.orgScopeRefs(orgRightPath(i), r.Region, r.Jurisdiction, juris)
		if !v.recipe[r.Subject] && !v.commodity[r.Subject] && !v.role[r.Subject] {
			v.rep.add(orgUnknownSubject(r.Activity), orgRightPath(i, "subject"), r.Subject, "")
		}
	}
	for i, c := range m.Controls {
		v.orgKnown(v.commodity[c.Commodity], ErrUnknownCommodity, orgControlPath(i, "commodity"), c.Commodity)
		v.orgScopeRefs(orgControlPath(i), c.Region, c.Jurisdiction, juris)
	}
	for i, l := range m.RateLimits {
		v.orgScopeRefs(orgLimitPath(i), l.Region, l.Jurisdiction, juris)
	}
	return len(v.rep.errs) == n
}

// orgBodyRefs checks the references of body i: its roles, regions, jurisdictions and
// the flag it needs.
func (v *validator) orgBodyRefs(i int, b Body, juris map[string]bool) {
	for j, r := range b.Roles {
		v.orgKnown(v.role[r], ErrUnknownRole, orgBodyPath(i, "roles["+itoa(j)+"]"), r)
	}
	for j, r := range b.Regions {
		v.orgRegion(orgBodyPath(i, "regions["+itoa(j)+"]"), r)
	}
	for j, id := range b.Jurisdictions {
		v.orgKnown(juris[id], ErrUnknownJurisdiction, orgBodyPath(i, "jurisdictions["+itoa(j)+"]"), id)
	}
	if s := v.c.Modules.Settlements; s != nil && b.Needs != "" {
		v.orgKnown(s.flagSet()[b.Needs], ErrUnknownSettlementFlag, orgBodyPath(i, "needs"), b.Needs)
	}
}

// orgUnknownSubject is the sentinel of a subject that names nothing: by the kind the
// activity says, and a recipe for an activity that is not one of the four.
func orgUnknownSubject(activity string) error {
	switch activity {
	case ActivitySell, ActivityBuy:
		return ErrUnknownCommodity
	case ActivityTrain:
		return ErrUnknownRole
	}
	return ErrUnknownRecipe
}

// orgJurisdictions is the set of jurisdiction IDs of the polity module, empty when it
// is off (a dependency error is reported for each use then).
func (v *validator) orgJurisdictions() map[string]bool {
	out := map[string]bool{}
	if p := v.c.Modules.Polity; p != nil {
		for _, j := range p.Jurisdictions {
			out[j.ID] = true
		}
	}
	return out
}

// orgKnown reports rule at p when ok is false. A jurisdiction is not checked while the
// polity module is off, as that is a dependency error.
func (v *validator) orgKnown(ok bool, rule error, p []string, value string) {
	if !ok && !(rule == ErrUnknownJurisdiction && v.c.Modules.Polity == nil) {
		v.rep.add(rule, p, value, "")
	}
}

// orgRegion checks a region of a scope: defined and not an outside node.
func (v *validator) orgRegion(p []string, region string) {
	switch {
	case !v.region[region]:
		v.rep.add(ErrUnknownRegion, p, region, "")
	case v.outsideSet()[region]:
		v.rep.add(ErrOutsideNode, p, region, "")
	}
}

// orgScopeRefs checks the scope key of a right, a control or a rate limit: the region
// it names, and the jurisdiction.
func (v *validator) orgScopeRefs(base []string, region, jurisdiction string, juris map[string]bool) {
	if region != "" {
		v.orgRegion(path(base, "region"), region)
	}
	if jurisdiction != "" {
		v.orgKnown(juris[jurisdiction], ErrUnknownJurisdiction, path(base, "jurisdiction"), jurisdiction)
	}
}

// orgShapes is ErrBodyShape, ErrRightShape and ErrControlShape: a body with no role or
// no region in its scope, a right with an activity outside the four, a subject of the
// wrong kind, a scope key that is not exactly one of region and jurisdiction or a Who
// that does not fit, and a control with a bad scope key or a floor above its cap.
func (v *validator) orgShapes(m *Institutions) bool {
	n := len(v.rep.errs)
	scoper := newOrgScoper(v)
	for i, b := range m.Bodies {
		switch {
		case len(b.Roles) == 0:
			v.rep.add(ErrBodyShape, orgBodyPath(i, "roles"), "0", "1")
		case len(scoper.regions(b.Regions, b.Jurisdictions)) == 0:
			v.rep.add(ErrBodyShape, orgBodyPath(i, "regions"), "0", "1")
		}
	}
	for i, r := range m.Rights {
		v.orgRightShape(i, r)
	}
	for i, c := range m.Controls {
		v.orgScopeShape(orgControlPath(i, "region"), c.Region, c.Jurisdiction)
		if c.CapBP > 0 && c.FloorBP > c.CapBP {
			v.rep.add(ErrControlShape, orgControlPath(i, "floor_bp"), i64(c.FloorBP), i64(c.CapBP))
		}
	}
	for i, l := range m.RateLimits {
		v.orgScopeShape(orgLimitPath(i, "region"), l.Region, l.Jurisdiction)
	}
	return len(v.rep.errs) == n
}

// orgScopeShape is ErrControlShape for a scope key that is both or neither of region and
// jurisdiction.
func (v *validator) orgScopeShape(p []string, region, jurisdiction string) {
	if (region == "") == (jurisdiction == "") {
		v.rep.add(ErrControlShape, p, region+jurisdiction, "1")
	}
}

// orgRightShape is the shape rules of one right.
func (v *validator) orgRightShape(i int, r Right) {
	if (r.Region == "") == (r.Jurisdiction == "") {
		v.rep.add(ErrRightShape, orgRightPath(i, "region"), r.Region+r.Jurisdiction, "1")
	}
	switch {
	case !orgActivity(r.Activity):
		v.rep.add(ErrRightShape, orgRightPath(i, "activity"), r.Activity, "")
	case !v.orgSubjectKind(r.Activity, r.Subject):
		v.rep.add(ErrRightShape, orgRightPath(i, "subject"), r.Subject, r.Activity)
	}
	switch r.Who {
	case "", WhoPrincipal:
	case WhoExecutor, WhoBoth:
		if r.Activity != ActivityProduce {
			v.rep.add(ErrRightShape, orgRightPath(i, "who"), r.Who, WhoPrincipal)
		}
	default:
		v.rep.add(ErrRightShape, orgRightPath(i, "who"), r.Who, WhoPrincipal)
	}
}

// orgActivity reports whether the activity is one of the four.
func orgActivity(a string) bool {
	return a == ActivityProduce || a == ActivitySell || a == ActivityBuy || a == ActivityTrain
}

// orgSubjectKind reports whether the subject is of the kind the activity names and the
// activity is one of the four.
func (v *validator) orgSubjectKind(activity, subject string) bool {
	switch activity {
	case ActivityProduce:
		return v.recipe[subject]
	case ActivitySell, ActivityBuy:
		return v.commodity[subject]
	case ActivityTrain:
		return v.role[subject]
	}
	return false
}

// orgOverlap is ErrBodyOverlap: a (role, region) that is in the scope of two bodies
// (spec-institutions.md 3.3.2), the first pair in the order of the bodies.
func (v *validator) orgOverlap(m *Institutions) bool {
	scoper := newOrgScoper(v)
	roleIx, regionIx := orgIndexes(v.c.Roles, func(r Role) string { return r.ID }), orgIndexes(v.c.Regions, func(r Region) string { return r.ID })
	items, places := make([][]int32, len(m.Bodies)), make([][]int32, len(m.Bodies))
	for i, b := range m.Bodies {
		items[i] = orgIndexList(b.Roles, roleIx)
		places[i] = orgIndexList(scoper.regions(b.Regions, b.Jurisdictions), regionIx)
	}
	a, b, role, region, ok := institution.Overlap(items, places)
	if ok {
		v.rep.add(ErrBodyOverlap, orgBodyPath(b, "roles"), v.c.Roles[role].ID+" in "+v.c.Regions[region].ID, m.Bodies[a].ID)
	}
	return !ok
}

// orgIndexes maps the ID of each element to its position, a lookup only.
func orgIndexes[T any](s []T, id func(T) string) map[string]int32 {
	out := make(map[string]int32, len(s))
	for i := range s {
		out[id(s[i])] = int32(i)
	}
	return out
}

// orgIndexList returns the ascending distinct positions of the names that are in ix.
func orgIndexList(names []string, ix map[string]int32) []int32 {
	out := make([]int32, 0, len(names))
	for _, n := range names {
		if i, ok := ix[n]; ok {
			out = append(out, i)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// orgBounds is ErrOutOfBounds: a Places or MaxTrainees outside its bound, a level above
// its bound, and a label over MaxLabelBytes.
func (v *validator) orgBounds(m *Institutions) bool {
	n := len(v.rep.errs)
	for i, b := range m.Bodies {
		v.inRange(orgBodyPath(i, "places"), b.Places, 0, MaxHouseholds)
		v.inRange(orgBodyPath(i, "max_trainees"), b.MaxTrainees, 0, MaxBodyTrainees)
		v.label(orgBodyPath(i), "label", b.Label)
	}
	for i, r := range m.Rights {
		v.label(orgRightPath(i), "label", r.Label)
	}
	for i, c := range m.Controls {
		v.inRange(orgControlPath(i, "cap_bp"), c.CapBP, 0, MaxBasisPoints)
		v.inRange(orgControlPath(i, "floor_bp"), c.FloorBP, 0, MaxBasisPoints)
		v.label(orgControlPath(i), "label", c.Label)
	}
	for i, l := range m.RateLimits {
		v.inRange(orgLimitPath(i, "cap_bp"), l.CapBP, 0, MaxCreditRateBP)
		v.label(orgLimitPath(i), "label", l.Label)
	}
	return len(v.rep.errs) == n
}

// orgScoper expands a scope given as regions and jurisdictions into the defined regions
// that are not outside nodes (spec-institutions.md 3.1.1). It reads the polity module
// as given and follows a parent chain for at most the number of jurisdictions, so a
// configuration that does not validate cannot loop.
type orgScoper struct {
	all     []string          // the defined regions that are not outside nodes, in the order of the configuration
	defined map[string]bool   // the same, as a set
	direct  map[string]string // region to its direct jurisdiction
	parent  map[string]string // jurisdiction to its parent
	steps   int
}

// newOrgScoper builds the scoper of a configuration for the checks of the validator.
func newOrgScoper(v *validator) *orgScoper { return orgScoperOf(&v.c, v.outsideSet()) }

// orgScoperOf builds the scoper of a configuration.
func orgScoperOf(c *Config, outside map[string]bool) *orgScoper {
	s := &orgScoper{defined: map[string]bool{}, direct: map[string]string{}, parent: map[string]string{}}
	for i := range c.Regions {
		if id := c.Regions[i].ID; !outside[id] {
			s.all = append(s.all, id)
			s.defined[id] = true
		}
	}
	if p := c.Modules.Polity; p != nil {
		for _, m := range p.Members {
			s.direct[m.Region] = m.Jurisdiction
		}
		for _, j := range p.Jurisdictions {
			s.parent[j.ID] = j.Parent
		}
		s.steps = len(p.Jurisdictions)
	}
	return s
}

// covers reports whether the jurisdiction is the direct jurisdiction of the region or
// an ancestor of it.
func (s *orgScoper) covers(jurisdiction, region string) bool {
	cur := s.direct[region]
	for step := 0; cur != "" && step <= s.steps; step++ {
		if cur == jurisdiction {
			return true
		}
		cur = s.parent[cur]
	}
	return false
}

// regions returns the distinct defined regions of the scope, sorted by ID: the named
// regions and those that a named jurisdiction covers.
func (s *orgScoper) regions(regions, jurisdictions []string) []string {
	var out []string
	for _, r := range regions {
		if s.defined[r] {
			out = append(out, r)
		}
	}
	for _, r := range s.all {
		for _, j := range jurisdictions {
			if s.covers(j, r) {
				out = append(out, r)
				break
			}
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}
