package economysim

import (
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/institution"
)

// This file is the institutions module (spec-institutions.md, ORG): the bodies, the
// households that are members of them, and the tables derived from the configuration
// that the rights and the legal prices of later changes read. It holds the state, its
// construction, the test of a membership and the names of the module's identifiers.
// The rules are in internal/institution over primitives; the round of P1.9a is in
// institutions_round.go, the resolved rights in institutions_rights.go, the legal
// ranges in institutions_legal.go, and the events, logs, snapshot rows and the
// checkpoint in institutions_report.go (architecture.md 2.6, 2.10.1).

// institutions is the state of the module, nil when Modules.Institutions is nil. The
// membership of a household is a pair of fields indexed by agent slot, kept here and
// not in the agent (architecture.md 2.10.1, N-244): they and the Warn runs are the
// saved state. Everything else is derived from the configuration and rebuilt by a
// restore, and the scratch of the round is never saved (spec-institutions.md 12.4).
type institutions struct {
	m *Institutions
	c *Config
	// pol is the polity (nil when off) and outside marks the outside nodes by region:
	// what the tables of rights and ranges are resolved against.
	pol     *polity
	outside []bool
	// nb, nr and nroles are the numbers of bodies, regions and roles.
	nb, nr, nroles int
	// bodyOf is the body of each (region, role), -1 for none, at region * nroles +
	// role: a (role, region) is in at most one body (spec-institutions.md 3.3.2).
	bodyOf []int32
	// onList says whether a role is on the list of a body, at body * nroles + role,
	// and inScope whether a region is in its scope, at body * nr + region.
	onList, inScope []bool
	// scope is the regions of each body, ascending.
	scope [][]RegionID
	// need is, per body, the index into the sorted flags of the settlements module of
	// the flag its Needs names, or -1 for none; places is its Places.
	need   []int
	places []int64
	// member and memberRegion are the membership of the agent in each slot, as the body
	// and the region of its admission plus 1, 0 for none. A field is a membership only
	// as isMember tests it (spec-institutions.md 4.4). They are saved in the tail of tag 3.
	member, memberRegion []int32
	// run is the consecutive ticks in which each (region, body), at region * nb + body,
	// had households waiting: the run of the Warn (spec-institutions.md 11.2). Diagnostic
	// and saved in tag 31.
	run []int64
	// rights and legal are the resolved tables (institutions_rights.go,
	// institutions_legal.go); rd is the scratch of the round (institutions_round.go).
	rights rightTable
	legal  legalTable
	rd     memberRound
	// class is the headroom the class limit leaves a role in a region, the slot that the
	// settlements module filled before this module wrapped it, nil when none did
	// (spec-institutions.md 3.4.3). cnt is the households of each role in region cntRegion
	// at tick cntTick, read once for each (tick, region) at the first read and kept exact by
	// noteRoleMove, as the settlements module keeps its own count (spec-institutions.md 3.4.2).
	class     func(RegionID, RoleID) (int64, bool)
	cnt       []int64
	cntOK     bool
	cntTick   int64
	cntRegion RegionID
}

// newInstitutions builds the state of the module from a canonical, validated
// configuration, the polity and the settlements module (nil when off) and the outside
// marks of the regions, or returns nil when the module is off. No household is a
// member until the first round (spec-institutions.md 4.1.3).
func newInstitutions(c *Config, pol *polity, set *settlements, outside []bool) *institutions {
	m := c.Modules.Institutions
	if m == nil {
		return nil
	}
	o := &institutions{m: m, c: c, pol: pol, outside: outside, nb: len(m.Bodies), nr: len(c.Regions), nroles: len(c.Roles)}
	o.bodyOf = make([]int32, o.nr*o.nroles)
	for i := range o.bodyOf {
		o.bodyOf[i] = -1
	}
	o.onList = make([]bool, o.nb*o.nroles)
	o.inScope = make([]bool, o.nb*o.nr)
	o.scope = make([][]RegionID, o.nb)
	o.need = make([]int, o.nb)
	o.places = make([]int64, o.nb)
	roles, regions := indexOf(c.Roles, func(r Role) string { return r.ID }), indexOf(c.Regions, func(r Region) string { return r.ID })
	for b := range m.Bodies {
		body := &m.Bodies[b]
		o.places[b], o.need[b] = body.Places, -1
		if set != nil {
			o.need[b] = nameIndex(set.flags, body.Needs)
		}
		o.scope[b] = scopeRegions(c, pol, outside, body.Regions, body.Jurisdictions, regions)
		for _, id := range body.Roles {
			o.onList[b*o.nroles+roles[id]] = true
		}
		for _, g := range o.scope[b] {
			o.inScope[b*o.nr+int(g)] = true
			for _, id := range body.Roles {
				o.bodyOf[int(g)*o.nroles+roles[id]] = int32(b)
			}
		}
	}
	o.run = make([]int64, o.nr*o.nb)
	o.cnt = make([]int64, o.nroles)
	o.rd = newMemberRound(o.nr * o.nb)
	o.resolve()
	return o
}

// indexOf maps the ID of each element to its position, a lookup that is never
// iterated.
func indexOf[T any](s []T, id func(T) string) map[string]int {
	out := make(map[string]int, len(s))
	for i := range s {
		out[id(s[i])] = i
	}
	return out
}

// scopeRegions returns the regions of a scope given as regions and jurisdictions: the
// named regions and the regions each named jurisdiction covers, outside nodes left
// out, ascending and listed once (spec-institutions.md 3.1.1). The scope of a right or
// a control, with one region or one jurisdiction, is the same function.
func scopeRegions(c *Config, pol *polity, outside []bool, named, jurisdictions []string, regions map[string]int) []RegionID {
	var out []RegionID
	for _, id := range named {
		out = append(out, RegionID(regions[id]))
	}
	if pol != nil {
		for _, id := range jurisdictions {
			j, _ := slices.BinarySearch(pol.v.names, id)
			out = append(out, pol.v.Regions(JurisdictionID(j))...)
		}
	}
	out = slices.DeleteFunc(out, func(g RegionID) bool { return outside[g] })
	slices.Sort(out)
	return slices.Compact(out)
}

// isMember reports whether the household in agent slot i, which is in region g now,
// is a member of body b at a read: its membership field is (b, g) and it is eligible
// for b, that is, its role is on b's list and g is in b's scope (spec-institutions.md
// 4.4.1). Three integer comparisons and an array read: a field that a role change or
// a move has made stale is not a membership.
func (o *institutions) isMember(i int, role RoleID, g RegionID, b int) bool {
	return i < len(o.member) && o.member[i] == int32(b)+1 && o.memberRegion[i] == int32(g)+1 &&
		o.bodyOf[int(g)*o.nroles+int(role)] == int32(b)
}

// grow makes the membership slices cover n agent slots; the new slots hold no
// membership.
func (o *institutions) grow(n int) {
	if n > len(o.member) {
		o.member = slices.Grow(o.member, n-len(o.member))[:n]
		o.memberRegion = slices.Grow(o.memberRegion, n-len(o.memberRegion))[:n]
	}
}

// eligibleBody returns the body a household of the role is eligible for in region g,
// or NoBody: the body of a household is a function of its role and its region
// (spec-institutions.md 3.3.2).
func (o *institutions) eligibleBody(role RoleID, g RegionID) BodyID {
	return BodyID(o.bodyOf[int(g)*o.nroles+int(role)])
}

// Membership is the membership group of an AgentState (spec-institutions.md 4.4.1,
// spec-integration.md 4.7.2): Member is true only for a household that is a member of a
// body at the boundary, and Body is the BodyID of that body. The zero value is no
// membership, so a world with the module off renders as it did before the module existed
// (architecture.md 2.10.5; the spec's NoBody is Member false).
type Membership struct {
	Member bool
	Body   BodyID
}

// fillMembership is the filler of AgentState.Membership (hook H-11): the body the household
// is a member of as isMember tests it, read at the boundary. A world with the module off
// and an agent that is no household in a region leave the zero value.
func (w *World) fillMembership(st *AgentState, a *agent) {
	o := w.org
	if o == nil {
		return
	}
	g, ok := w.householdRegion(a)
	if !ok {
		return
	}
	if b := o.eligibleBody(a.role, g); b >= 0 && o.isMember(int(a.id), a.role, g, int(b)) {
		st.Membership = Membership{Member: true, Body: b}
	}
}

// wireOrgHeadroom is the construction call that makes the headroom of a completion the least
// of the class limit and the body's (hook H-3, spec-institutions.md 3.4.3): the slot of the
// skills module becomes orgHeadroom, whose first half is the class limit of the settlements
// module when it is on. It runs at allocation, which a restore shares with construction, and
// again after the settlements module wired its own slot, which it would otherwise replace. It
// does nothing without both modules.
func (w *World) wireOrgHeadroom() {
	if w.org == nil || w.skills == nil {
		return
	}
	w.org.class = nil
	if w.set != nil {
		w.org.class = w.roleHeadroom
	}
	w.skills.headroom = w.orgHeadroom
}

// orgHeadroom is the headroom of role r in region g for the completion of a trainee: the least
// of the class limit's, when the settlements module gives one, and of the body's, when the role
// is in a body with places (spec-institutions.md 3.4.3). A role in no body keeps its class
// limit and a body with no places gives no figure.
func (w *World) orgHeadroom(g RegionID, r RoleID) (headroom int64, noLimit bool) {
	o := w.org
	headroom, noLimit = 0, true
	if o.class != nil {
		headroom, noLimit = o.class(g, r)
	}
	if b := o.bodyOf[int(g)*o.nroles+int(r)]; b >= 0 {
		bh, bn := o.bodyHeadroom(w, g, int(b))
		headroom, noLimit = institution.Least(headroom, noLimit, bh, bn)
	}
	return headroom, noLimit
}

// bodyHeadroom is the headroom of body b in region g: max(0, Places - H), H the households of
// the body's roles in the region, members or not (spec-institutions.md 3.4.2), and no limit for
// a body with no places.
func (o *institutions) bodyHeadroom(w *World, g RegionID, b int) (int64, bool) {
	if o.places[b] <= 0 {
		return 0, true
	}
	o.count(w, g)
	var h int64
	for r := range o.nroles {
		if o.bodyOf[int(g)*o.nroles+r] == int32(b) {
			h = satAdd64(h, o.cnt[r])
		}
	}
	return institution.Headroom(o.places[b], h)
}

// count fills the households of each role in region g, once for each (tick, region): the
// scan of the agents that the headroom of the settlements module makes (spec-institutions.md
// 3.4.2). noteRoleMove keeps it exact after.
func (o *institutions) count(w *World, g RegionID) {
	if t := w.Tick(); o.cntOK && o.cntTick == t && o.cntRegion == g {
		return
	}
	clear(o.cnt)
	for i := w.firstHousehold; i < len(w.agents); i++ {
		a := &w.agents[i]
		if rg, ok := w.householdRegion(a); ok && rg == g {
			o.cnt[a.role] = satAdd64(o.cnt[a.role], a.weight)
		}
	}
	o.cntOK, o.cntTick, o.cntRegion = true, w.Tick(), g
}

// noteRoleMove is called (hook H-13) after a completed traineeship moved weight households of
// region g from one role to another, so a count read before it stays exact for the next held
// trainee. It does nothing for a nil module or when no count is held for the region.
func (o *institutions) noteRoleMove(g RegionID, from, to RoleID, weight int64) {
	if o != nil && o.cntOK && o.cntRegion == g {
		o.cnt[from] = max(o.cnt[from]-weight, 0) // a count never goes below 0, so headroom is never inflated
		o.cnt[to] = satAdd64(o.cnt[to], weight)
	}
}

// noteHeld is called (hook H-13) when a completion of weight households into role r of region
// g is held for want of headroom, and counts it for the body of the role when the body's own
// places are what is short, for MembershipChanged.Held (spec-institutions.md 4.5.2). The count
// exists only for the event, so a world with no observer keeps none.
func (o *institutions) noteHeld(w *World, g RegionID, r RoleID, weight int64) {
	if o == nil || !w.obs.supplied {
		return
	}
	b := o.bodyOf[int(g)*o.nroles+int(r)]
	if b < 0 {
		return
	}
	if room, none := o.bodyHeadroom(w, g, int(b)); !none && room < weight {
		o.rd.out[int(g)*o.nb+int(b)].Held++
	}
}

// fillRoleBodies is the reader of a body's places in the view of region g (hook H-14,
// spec-institutions.md 3.4.2, 4.5.1): the Headroom of each role in a body with places is the
// least of what its class limit left and the places less the households of the body's roles,
// at the start of the entry step, and its NoLimit is cleared. A role in no body, and a body with
// no places, are untouched. The spawner and the population module read the same fields.
func (w *World) fillRoleBodies(g RegionID) {
	o := w.org
	roles := w.ex.view.Roles
	for b := range o.nb {
		if o.places[b] <= 0 || !o.inScope[b*o.nr+int(g)] {
			continue
		}
		var h int64
		for r := range roles {
			if o.bodyOf[int(g)*o.nroles+r] == int32(b) {
				h = satAdd64(h, roles[r].Households)
			}
		}
		room, none := institution.Headroom(o.places[b], h)
		for r := range roles {
			if o.bodyOf[int(g)*o.nroles+r] == int32(b) {
				roles[r].Headroom, roles[r].NoLimit = institution.Least(roles[r].Headroom, roles[r].NoLimit, room, none)
			}
		}
	}
}

// satAdd64 adds two non-negative counts of households, saturating at the largest
// int64 so that a count never wraps (architecture.md 3.5).
func satAdd64(a, b int64) int64 {
	if v, ok := checked.Add(a, b); ok {
		return v
	}
	return math.MaxInt64
}

// orgNames returns the string ID of each body, right and control, by their typed IDs,
// or nil with the module off or with none of the kind declared (Names.Bodies, Rights,
// Controls, spec-institutions.md 11.4.1), so a module that is present and empty leaves
// the names as they are without it. The controls and the rate limits are one list,
// sorted together by ID.
func orgNames(c *Config) (bodies, rights, controls []string) {
	m := c.Modules.Institutions
	if m == nil {
		return nil, nil, nil
	}
	for i := range m.Bodies {
		bodies = append(bodies, m.Bodies[i].ID)
	}
	for i := range m.Rights {
		rights = append(rights, m.Rights[i].ID)
	}
	if ids := controlIDs(m); len(ids) > 0 {
		controls = ids
	}
	return bodies, rights, controls
}

// controlIDs returns the IDs of the controls and the rate limits sorted together
// bytewise, the index of each being its ControlID (spec-institutions.md 6.1.2).
func controlIDs(m *Institutions) []string {
	out := make([]string, 0, len(m.Controls)+len(m.RateLimits))
	for i := range m.Controls {
		out = append(out, m.Controls[i].ID)
	}
	for i := range m.RateLimits {
		out = append(out, m.RateLimits[i].ID)
	}
	slices.Sort(out)
	return out
}
