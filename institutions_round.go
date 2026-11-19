package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/institution"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// This file is the membership round of the institutions module (spec-institutions.md
// 4): P1.9a, the lapse of the memberships that are no longer valid and the admission
// of the eligible households that are not members, by seniority, into the places a
// body has. It is the continuation file of institutions.go (architecture.md 2.6).

// memberRound is the scratch of the round, allocated once and reused. out holds the
// counts of each (region, body) at region * nb + body and is the source of the events
// of P7.2b; members and eligible are the households of the cell after the round.
type memberRound struct {
	out               []institution.Outcome
	members, eligible []int64
	// region is the region of the household in each agent slot, -1 for a slot that is
	// no household.
	region []int32
	// start are the offsets of the candidates of each cell in cands, and cands the
	// eligible non-members, grouped by cell.
	start []int32
	cands []institution.Candidate
	// adm is the agents admitted to one cell, in the order admitted.
	adm []int32
	// visited is the household slots the last round visited, the work of NFR-ORG-002.
	visited int64
}

// newMemberRound allocates the scratch for cells (region, body) pairs.
func newMemberRound(cells int) memberRound {
	return memberRound{
		out: make([]institution.Outcome, cells), members: make([]int64, cells), eligible: make([]int64, cells),
		start: make([]int32, cells+1),
	}
}

// p19aMembership is P1.9a: the membership round (spec-institutions.md 4.1). It takes no
// draw and runs every tick; there is none at construction.
func (w *World) p19aMembership() {
	w.org.round(w)
}

// round is one round: the lapse pass over every household, then, for each body in
// ascending BodyID and each region of its scope in ascending RegionID, the admission
// (spec-institutions.md 4.1.1). The lapse of every body runs before any admission, so
// a household that lapses from one body and is eligible for another is a candidate of
// it in the same round whatever the order of the bodies (4.2.1, FR-ORG-019). The result
// is a function of the state and not of the order of the agents.
func (o *institutions) round(w *World) {
	if o.nb == 0 {
		return // the module present and empty has no body to lapse from or admit to
	}
	rd := &o.rd
	o.grow(len(w.agents))
	clear(rd.out)
	clear(rd.members)
	clear(rd.eligible)
	o.lapse(w)
	o.classify(w)
	for b := range o.nb {
		for _, g := range o.scope[b] {
			o.admit(w, b, g)
		}
	}
}

// lapse is the first pass over the households: it records the region of each, and
// clears the membership of one that is no longer valid, counting it LapsedRole when its
// role left the body's list and LapsedRegion when it is out of the scope or has moved,
// in the cell it was admitted in (spec-institutions.md 4.2.1).
func (o *institutions) lapse(w *World) {
	rd := &o.rd
	rd.region = slices.Grow(rd.region[:0], len(w.agents))[:len(w.agents)]
	rd.visited = 0
	for i := range w.agents {
		rd.region[i] = -1
		if i < w.firstHousehold {
			continue
		}
		a := &w.agents[i]
		g, ok := w.householdRegion(a)
		if !ok {
			continue
		}
		rd.region[i] = int32(g)
		rd.visited++
		if o.member[i] == 0 {
			continue
		}
		b, at := int(o.member[i])-1, int32(o.memberRegion[i])-1
		k := institution.Lapse(o.onList[b*o.nroles+int(a.role)], o.inScope[b*o.nr+int(g)], at, int32(g))
		if k == institution.Keep {
			continue
		}
		rd.out[int(at)*o.nb+b].Lapsed(k, a.weight)
		o.member[i], o.memberRegion[i] = 0, 0
	}
}

// classify is the second pass: for each household that is eligible for a body it
// counts it in its cell, as a member or as a candidate, and groups the candidates by
// cell in cands (spec-institutions.md 4.3.1 steps 1 and 2).
func (o *institutions) classify(w *World) {
	rd := &o.rd
	clear(rd.start)
	for i, g := range rd.region {
		if g < 0 {
			continue
		}
		a := &w.agents[i]
		b := o.bodyOf[int(g)*o.nroles+int(a.role)]
		if b < 0 {
			continue
		}
		cell := int(g)*o.nb + int(b)
		rd.eligible[cell] = satAdd64(rd.eligible[cell], a.weight)
		if o.isMember(i, a.role, RegionID(g), int(b)) {
			rd.members[cell] = satAdd64(rd.members[cell], a.weight)
		} else {
			rd.start[cell+1]++
		}
	}
	for c := range len(rd.start) - 1 {
		rd.start[c+1] += rd.start[c]
	}
	rd.cands = slices.Grow(rd.cands[:0], int(rd.start[len(rd.start)-1]))[:rd.start[len(rd.start)-1]]
	o.fill(w)
}

// fill is the third pass: it writes each candidate at the next free place of its
// cell, in ascending agent ID within the cell. The offsets are consumed and restored
// by shifting them back, so no second table is needed.
func (o *institutions) fill(w *World) {
	rd := &o.rd
	t := w.Tick()
	for i, g := range rd.region {
		if g < 0 {
			continue
		}
		a := &w.agents[i]
		b := o.bodyOf[int(g)*o.nroles+int(a.role)]
		if b < 0 || o.isMember(i, a.role, RegionID(g), int(b)) {
			continue
		}
		cell := int(g)*o.nb + int(b)
		rd.cands[rd.start[cell]] = institution.Candidate{
			ID: int32(i), Weight: a.weight, Since: a.roleSince, Key: ordering.Key(t, int64(i)),
		}
		rd.start[cell]++
	}
	for c := len(rd.start) - 1; c > 0; c-- {
		rd.start[c] = rd.start[c-1]
	}
	rd.start[0] = 0
}

// admit admits the candidates of body b in region g into its free places, in the
// order of spec-institutions.md 4.3.1: the free places are the places less the
// members, a body whose need the region's class does not meet admits nobody and
// counts its candidates Barred, and a candidate that does not fit is Skipped and the
// round goes on. It sets the membership of each household admitted.
func (o *institutions) admit(w *World, b int, g RegionID) {
	rd := &o.rd
	cell := int(g)*o.nb + b
	cands := rd.cands[rd.start[cell]:rd.start[cell+1]]
	free := o.places[b] - rd.members[cell]
	barred := o.need[b] >= 0 && !w.regionHas(g, o.need[b])
	var out institution.Outcome
	rd.adm, out = institution.Admit(rd.adm[:0], cands, free, o.places[b] > 0, barred)
	res := &rd.out[cell]
	res.Admitted, res.Skipped, res.Barred, res.Waiting = out.Admitted, out.Skipped, out.Barred, out.Waiting
	for _, id := range rd.adm {
		o.member[id], o.memberRegion[id] = int32(b)+1, int32(g)+1
	}
	rd.members[cell] = satAdd64(rd.members[cell], out.Admitted)
}
