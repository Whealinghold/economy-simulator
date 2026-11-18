package economysim

import (
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
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

// satAdd64 adds two non-negative counts of households, saturating at the largest
// int64 so that a count never wraps (architecture.md 3.5).
func satAdd64(a, b int64) int64 {
	if v, ok := checked.Add(a, b); ok {
		return v
	}
	return math.MaxInt64
}

// orgNames returns the string ID of each body, right and control, by their typed IDs,
// or nil with the module off (Names.Bodies, Rights, Controls, spec-institutions.md
// 11.4.1). The controls and the rate limits are one list, sorted together by ID.
func orgNames(c *Config) (bodies, rights, controls []string) {
	m := c.Modules.Institutions
	if m == nil {
		return nil, nil, nil
	}
	bodies = make([]string, len(m.Bodies))
	for i := range m.Bodies {
		bodies[i] = m.Bodies[i].ID
	}
	rights = make([]string, len(m.Rights))
	for i := range m.Rights {
		rights[i] = m.Rights[i].ID
	}
	return bodies, rights, controlIDs(m)
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
