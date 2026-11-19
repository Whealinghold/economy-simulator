package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/institution"
)

// This file is the resolved rights of the institutions module (spec-institutions.md
// 5.4): for each activity, subject and region, the right in force, resolved once at
// construction and again after a restore or a slot call into a dense array, so a read
// in the tick is an array read that allocates nothing and is not saved (FR-ORG-014).
// It is a continuation file of institutions.go (architecture.md 2.6).

// The four activities of a right, as the index of their tables.
const (
	actProduce = iota
	actSell
	actBuy
	actTrain
	numActivities
)

// rightTable holds the rights as the resolver reads them. rows and ids list the rights
// of each activity in RightID order; at is the right in force plus 1 at subject *
// regions + region, 0 for none (the activity is open); holder is the body of each right
// by RightID, -1 for a prohibition; suspended is the slot state, by RightID.
type rightTable struct {
	rows      [numActivities][]institution.Row
	ids       [numActivities][]int32
	at        [numActivities][]int32
	holder    []int32
	suspended []bool
}

// activityOf returns the table index of an activity name of a validated right.
func activityOf(name string) int {
	switch name {
	case ActivitySell:
		return actSell
	case ActivityBuy:
		return actBuy
	case ActivityTrain:
		return actTrain
	}
	return actProduce
}

// subjectIndex is the position of each name a right can have as its subject, by
// activity: the recipes of a produce right, the commodities of a sell or buy right and
// the roles of a train right.
func (o *institutions) subjectIndex() [numActivities]map[string]int {
	c := o.c
	recipes, commodities := indexOf(c.Recipes, func(r Recipe) string { return r.ID }), indexOf(c.Commodities, func(k Commodity) string { return k.ID })
	roles := indexOf(c.Roles, func(r Role) string { return r.ID })
	return [numActivities]map[string]int{actProduce: recipes, actSell: commodities, actBuy: commodities, actTrain: roles}
}

// chains returns the jurisdictions of each region, innermost first, as the resolver
// takes them; empty for a region with none or with the polity module off.
func (o *institutions) chains() [][]int32 {
	out := make([][]int32, o.nr)
	if o.pol == nil {
		return out
	}
	for g := range out {
		for _, j := range o.pol.v.Chain(RegionID(g)) {
			out[g] = append(out[g], int32(j))
		}
	}
	return out
}

// scopeOf returns the key of a right's or a control's scope: the region, or the
// jurisdiction when the region is empty.
func (o *institutions) scopeOf(region, jurisdiction string, regions map[string]int) institution.Scope {
	if region != "" {
		return institution.Scope{Key: int32(regions[region])}
	}
	j, _ := slices.BinarySearch(o.pol.v.names, jurisdiction)
	return institution.Scope{Jurisdiction: true, Key: int32(j)}
}

// resolveRights builds the table of the right in force for every (activity, subject,
// region) with a right, from the rights of the configuration, the suspensions and the
// chains of the regions. A suspended right leaves its key open and no outer right is
// consulted (FR-ORG-015); a region with no row and no jurisdiction with one has none.
func (o *institutions) resolveRights(chains [][]int32) {
	t := &o.rights
	m := o.m
	regions := indexOf(o.c.Regions, func(r Region) string { return r.ID })
	subjects := o.subjectIndex()
	bodies := indexOf(m.Bodies, func(b Body) string { return b.ID })
	t.holder = make([]int32, len(m.Rights))
	if len(t.suspended) != len(m.Rights) {
		t.suspended = make([]bool, len(m.Rights))
	}
	for a := range numActivities {
		t.rows[a], t.ids[a] = t.rows[a][:0], t.ids[a][:0]
	}
	for i := range m.Rights {
		r := &m.Rights[i]
		a := activityOf(r.Activity)
		t.holder[i] = -1
		if r.Body != "" {
			t.holder[i] = int32(bodies[r.Body])
		}
		sc := o.scopeOf(r.Region, r.Jurisdiction, regions)
		t.rows[a] = append(t.rows[a], institution.Row{Subject: int32(subjects[a][r.Subject]), Scope: sc, Suspended: t.suspended[i]})
		t.ids[a] = append(t.ids[a], int32(i))
	}
	for a := range numActivities {
		o.resolveActivity(a, len(subjects[a]), chains)
	}
}

// resolveActivity fills the table of activity a: for each subject that has a right and
// each region that is not an outside node, the right in force.
func (o *institutions) resolveActivity(a, subjects int, chains [][]int32) {
	t := &o.rights
	t.at[a] = make([]int32, subjects*o.nr)
	seen := make([]bool, subjects)
	for _, row := range t.rows[a] {
		seen[row.Subject] = true
	}
	for s := range subjects {
		if !seen[s] {
			continue
		}
		for g := range o.nr {
			if o.outside[g] {
				continue
			}
			if ref, ok := institution.Resolve(t.rows[a], int32(s), int32(g), chains[g]); ok {
				t.at[a][s*o.nr+g] = t.ids[a][ref] + 1
			}
		}
	}
}

// rightIn returns the right in force for the activity on the subject (a recipe, a
// commodity or a role index) in region g, and false when the activity is open there.
// It is an array read and allocates nothing (spec-institutions.md 5.4.2).
func (o *institutions) rightIn(activity, subject int, g RegionID) (RightID, bool) {
	at := o.rights.at[activity]
	i := subject*o.nr + int(g)
	if i >= len(at) || at[i] == 0 {
		return 0, false
	}
	return RightID(at[i] - 1), true
}
