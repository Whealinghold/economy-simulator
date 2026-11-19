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
// by RightID, -1 for a prohibition; suspended is the slot state, by RightID. who is the
// Who of a produce right, by RightID. refused is the count of refusals at region *
// rights + right, kept only when an observer was supplied and told at P7.2b.
type rightTable struct {
	rows      [numActivities][]institution.Row
	ids       [numActivities][]int32
	at        [numActivities][]int32
	holder    []int32
	suspended []bool
	who       []uint8
	refused   []int64
}

// The Who of a right as the table holds it (spec-institutions.md 5.3).
const (
	whoPrincipal uint8 = iota
	whoExecutor
	whoBoth
)

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
	t.holder, t.who = make([]int32, len(m.Rights)), make([]uint8, len(m.Rights))
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
		switch r.Who {
		case WhoExecutor:
			t.who[i] = whoExecutor
		case WhoBoth:
			t.who[i] = whoBoth
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

// outsider reports whether household a, in region own, is barred by right id: the right
// has no holder (a prohibition), or its holder is a body a is not a member of
// (spec-institutions.md 5.1.1, 5.4.4). Three comparisons and an array read.
func (o *institutions) outsider(id RightID, a *agent, own RegionID) bool {
	h := o.rights.holder[id]
	return h < 0 || !o.isMember(int(a.id), a.role, own, int(h))
}

// barredAt returns the right that bars household a, in region own, from the activity on
// the subject (a recipe, a commodity or a role index) in region at, and false when the
// activity is open there or a is a member of the holder. at is the region of the right and
// own the region of the household: they differ for a delivery contract, whose right is the
// destination's (spec-institutions.md 5.2.1).
func (o *institutions) barredAt(a *agent, own, at RegionID, activity, subject int) (RightID, bool) {
	id, ok := o.rightIn(activity, subject, at)
	if !ok || !o.outsider(id, a, own) {
		return 0, false
	}
	return id, true
}

// startBarred reports whether a right bars the start of recipe by the job of executor a
// and principal p in region g, and the right that does (spec-institutions.md 5.2, 5.3).
// Who says whose membership is tested: the principal, the executor or both. A trainee is
// exempt from the executor test and its principal, the trainer, is tested as for principal
// whatever Who says (5.3.2, trainee true); an own-use recipe is exempt unless OwnUseBound is
// set (5.5.2). The right is read at a start only (5.5.1).
func (o *institutions) startBarred(w *World, a, p *agent, trainee bool, g RegionID, recipe RecipeID) (RightID, bool) {
	id, ok := o.rightIn(actProduce, int(recipe), g)
	if !ok || (!o.m.OwnUseBound && isOwn(w.roles.row(a.role), recipe)) {
		return 0, false
	}
	who := o.rights.who[id]
	if (trainee || who != whoExecutor) && o.outsider(id, p, g) {
		return id, true
	}
	return id, who != whoPrincipal && !trainee && o.outsider(id, a, g)
}

// attach is the construction call that gives the module what the world holds and the
// module does not: the counts of the refusals of the rights, which exist only when an
// observer was supplied (observability-and-logging.md 4.5), so a world without one
// keeps none. A nil module does nothing.
func (o *institutions) attach(supplied bool) {
	if o != nil && supplied {
		o.rights.refused = make([]int64, o.nr*len(o.m.Rights))
	}
}

// reportRights is the RightRefused events of P7.2b: one for each (region, right) with a
// count, in ascending (RegionID, RightID) order, after the MembershipChanged events
// (spec-institutions.md 7.3.3). The count of the activity of the right is the one that is
// set, and the counts are cleared as they are told.
func (o *institutions) reportRights(w *World) {
	n := len(o.m.Rights)
	for g := range o.nr {
		for id := range n {
			c := &o.rights.refused[g*n+id]
			if *c == 0 {
				continue
			}
			e := RightRefused{Tick: w.Tick(), Region: RegionID(g), Right: RightID(id)}
			switch activityOf(o.m.Rights[id].Activity) {
			case actProduce:
				e.Produce = *c
			case actSell:
				e.Sell = *c
			case actBuy:
				e.Buy = *c
			default:
				e.Train = *c
			}
			*c = 0
			w.obs.deliver(e)
		}
	}
}

// refuse counts one refusal of right id in region g, when an observer was supplied
// (the counts exist only for the event of P7.2b, FR-ORG-039).
func (o *institutions) refuse(g RegionID, id RightID) {
	if r := o.rights.refused; r != nil {
		r[int(g)*len(o.m.Rights)+int(id)]++
	}
}

// refuseStart is the check of a request at P2.5 and of a trainee's request (hook H-4,
// spec-institutions.md 5.2): it reports whether a right bars the start of recipe by a, with
// p its principal (a itself unless a is a trainee), and, if so, counts the refusal and
// refuses the request with ReasonNotPermitted, which logs the Debug line and counts an
// own-use household as blocked. It runs after not_on_role and before k0 is derived.
func (w *World) refuseStart(a, p *agent, trainee bool, g RegionID, recipe RecipeID) bool {
	id, barred := w.org.startBarred(w, a, p, trainee, g, recipe)
	if !barred {
		return false
	}
	w.org.refuse(g, id)
	w.refuse(a, g, recipe, ReasonNotPermitted)
	if w.sub != nil {
		w.sub.blockedStart(w, a, int(g)*len(w.cfg.Roles)+int(a.role), recipe, ReasonNotPermitted)
	}
	return true
}

// orderBarred reports whether a right bars household a, in region g, from placing the
// order: the right of the sell activity for an ask and of the buy activity for a bid, on
// its commodity (hook H-5, spec-institutions.md 5.2, 5.6.2). The world passes it as the
// fact NotPermitted of the admission, so the count is made only for an order that
// admission refuses for it (refusedOrder).
func (o *institutions) orderBarred(a *agent, g RegionID, ord Order) bool {
	_, barred := o.barredAt(a, g, g, orderActivity(ord.Side), int(ord.Commodity))
	return barred
}

// refusedOrder counts the order that admission refused as not permitted, under the right
// that barred it.
func (o *institutions) refusedOrder(a *agent, g RegionID, ord Order) {
	if id, barred := o.barredAt(a, g, g, orderActivity(ord.Side), int(ord.Commodity)); barred {
		o.refuse(g, id)
	}
}

// orderActivity is the activity a side of an order is: a bid buys and an ask sells.
func orderActivity(s Side) int {
	if s == Bid {
		return actBuy
	}
	return actSell
}

// contractRefusal is row 5a of an offer and of a take of the contract round (hook H-7,
// spec-merchants.md 9.3): a delivery contract is a sale for the deliverer and a purchase
// for the payer, tested for the destination region at (spec-institutions.md 5.2.1, 6.7.1).
// It returns ReasonNotPermitted and counts the refusal, or "" (always "" for a nil module,
// so the row has no effect with the module off).
func (o *institutions) contractRefusal(a *agent, own, at RegionID, activity int, c CommodityID) Reason {
	if o == nil {
		return ""
	}
	id, barred := o.barredAt(a, own, at, activity, int(c))
	if !barred {
		return ""
	}
	o.refuse(at, id)
	return ReasonNotPermitted
}

// ckOfferRight is row 5a of an offer, a check of the table offerChecks of contract.go: the
// deliverer's sell right on the commodity is not held, tested for the destination region
// (spec-institutions.md 5.2.1, 6.7.1). It refuses nothing with the module off.
func ckOfferRight(w *World, a *agent, s *submittedOffer, _ int) Reason {
	return w.org.contractRefusal(a, w.regionOf(a), s.o.To, actSell, s.o.Commodity)
}
