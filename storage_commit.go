package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/room"
)

// This file is the part of the storage module that commits and cuts room
// (spec-storage.md 5): the pledge of an account, derived from its job and the
// cargo in flight, and the move of goods to the estate that a cut of the room
// needs. It is a continuation of storage.go (architecture.md 2.6).

// loadOfLots is the load of the lots, quantity times size per dimension, with
// checked arithmetic; ok is false when a dimension overflows int64.
func (s *storage) loadOfLots(lots []ledger.Lot) (load room.Pair, ok bool) {
	ok = true
	for _, lt := range lots {
		load, ok = s.addUnits(load, ok, lt.Commodity, lt.Quantity)
	}
	return load, ok
}

// addUnits adds the load of qty units of commodity c to load, quantity times size per
// dimension with checked arithmetic; ok stays false once any step overflowed.
func (s *storage) addUnits(load room.Pair, ok bool, c int, qty int64) (room.Pair, bool) {
	sz := s.sizes[c]
	m, okm := checked.Mul(qty, sz.Mass)
	b, okb := checked.Mul(qty, sz.Bulk)
	sum, oks := addPair(load, room.Pair{Mass: m, Bulk: b})
	return sum, ok && okm && okb && oks
}

// pendingLoad is the load of the pending output of a job: the outputs that sit in the
// account, quantity times size per dimension. A discard output is created and
// disposed at the completion and takes no room (spec-storage.md 5.3.2). ok is false
// when a dimension overflows int64.
func (s *storage) pendingLoad(j *job) (load room.Pair, ok bool) {
	if j.executions <= 0 {
		return room.Pair{}, true
	}
	ok = true
	for i, o := range s.recipes.row(j.recipe).outputs {
		if o.discard {
			continue
		}
		load, ok = s.addUnits(load, ok, int(o.commodity), j.pending[i])
	}
	return load, ok
}

// syncJob brings the pledge up to date with the job of executor a (spec-storage.md
// 3.4.2, 4.3): the load its job had pledged on its principal is taken off, the load
// of its pending output now is put on the principal it has now (the executor itself,
// or a trainer for a trainee's job), and both principals' pledges are derived again.
// It is called at every change of a job: a start, a completion, an abandon, a change
// of weight.
func (w *World) syncJob(a *agent) {
	s := w.sto
	s.grow(a.id)
	old, op := s.jobLoad[a.id], s.jobPrin[a.id]
	rest, ok := subPair(s.held[op], old)
	if !ok {
		s.broke(a.id, "the pledge of a principal is below the load of a job it pledged")
	}
	s.held[op] = rest
	load, okj := s.pendingLoad(&a.job)
	np := a.id
	if a.job.executions > 0 {
		np = a.job.principal
	}
	sum, oks := addPair(s.held[np], load)
	if !okj || !oks {
		s.broke(a.id, "the pledge of a principal does not fit int64")
	}
	s.held[np], s.jobLoad[a.id], s.jobPrin[a.id] = sum, load, np
	w.repledge(&w.agents[op])
	if np != op {
		w.repledge(&w.agents[np])
	}
}

// subPair is the checked difference of two pairs, ok false below 0 or on overflow.
func subPair(a, b room.Pair) (room.Pair, bool) {
	m, okm := checked.Sub(a.Mass, b.Mass)
	k, okk := checked.Sub(a.Bulk, b.Bulk)
	return room.Pair{Mass: m, Bulk: k}, okm && okk && m >= 0 && k >= 0
}

// repledge sets the pledge of agent a to what the jobs pledged on it and the cargo in
// flight to it imply (spec-storage.md 3.4.2, 4.3), absolute. The value is not tested
// against the room: the commitments that made it were each tested, and after a cut
// it may exceed it (spec-storage.md 4.1.2). A pledge that does not fit int64 is a
// violated invariant: every part of it was bounded by the room when it was committed.
// It does nothing for an account that is not limited.
func (w *World) repledge(a *agent) {
	s := w.sto
	if !s.limited(a) || a.weight < 1 {
		return
	}
	p, ok := addPair(s.held[a.id], s.inbound[a.id])
	if !ok {
		s.broke(a.id, "the pledge of an account does not fit int64")
	}
	if err := w.ledger.SetPledge(a.account, p.Mass, p.Bulk); err != nil {
		s.broke(a.id, "the ledger refused the pledge of a limited account: "+err.Error())
	}
}

// cutForRole is the step of a role change that moves the goods above the new
// role's room to the estate (spec-storage.md 5.5.2, FR-STO-018): called with the
// agent still in its old role, it resolves the room of the role to, and if that role
// is limited and the load is above it, moves goods by the rule of 5.5.2.1. It
// returns the number of lots moved, which RoleChanged.GoodsListed counts. A role
// with no base room has nothing to compare.
func (w *World) cutForRole(a *agent, region RegionID, to RoleID) (int64, error) {
	s := w.sto
	base := s.roles.rows[to].room
	if base == nil {
		return 0, nil
	}
	return w.cutLoad(a, region, s.roomIn(a, to))
}

// cutLoad moves whole lots of agent a to the estate of its region, under
// estate_goods, until its load fits the room to, by the rule of 5.5.2.1: the
// binding dimension is the one whose shortfall is the largest relative to the room,
// the commodity of the greatest load in it goes first (ties by the lower commodity
// ID), its lots oldest first, and the ranking is taken again before each commodity.
// It returns the number of lots moved and moves nothing when the load fits. A cargo
// in flight is not moved: the arrival diverts it if the room cannot hold it.
func (w *World) cutLoad(a *agent, region RegionID, to room.Pair) (int64, error) {
	s := w.sto
	lots, err := w.ledger.AppendLots(s.lots[:0], a.account)
	s.lots = lots
	if err != nil {
		s.broke(a.id, "the account of an agent that loses room is unreadable: "+err.Error())
	}
	load, ok := s.loadOfLots(lots)
	if !ok {
		s.broke(a.id, "the load of an account does not fit int64")
	}
	if load.Mass <= to.Mass && load.Bulk <= to.Bulk {
		return 0, nil
	}
	cands, at := s.candidates(lots)
	steps, met := room.ExcessOrder(load, to, cands)
	if !met {
		s.broke(a.id, "a load above the room has no commodity of a size to move")
	}
	var n int64
	for _, st := range steps {
		lt := lots[at[st.Cand][st.Lot]]
		key := ledger.Key{Commodity: lt.Commodity, Born: lt.Born, Durability: lt.Durability}
		if err := w.ledger.MoveLot(a.account, w.estates.accounts[region], key, lt.Quantity, ledger.EstateGoods); err != nil {
			return n, refused("move a lot above the room to the estate", region, a.account, err)
		}
		n++
	}
	return n, nil
}

// candidates groups the lots, which the ledger holds in canonical order (commodity
// first, then oldest first), into the candidates of room.ExcessOrder, and returns
// beside them, for each, the index in lots of each of its lots.
func (s *storage) candidates(lots []ledger.Lot) (cands []room.Candidate, at [][]int) {
	for i, lt := range lots {
		if n := len(cands); n == 0 || cands[n-1].Commodity != lt.Commodity {
			cands = append(cands, room.Candidate{Commodity: lt.Commodity, Size: s.sizes[lt.Commodity]})
			at = append(at, nil)
		}
		last := len(cands) - 1
		cands[last].Lots = append(cands[last].Lots, lt.Quantity)
		at[last] = append(at[last], i)
	}
	return cands, at
}

// linesLoad is the load of the lines of a shipment, quantity times size per dimension,
// with checked arithmetic; ok is false when a dimension overflows int64.
func (s *storage) linesLoad(lines []Line) (load room.Pair, ok bool) {
	ok = true
	for _, ln := range lines {
		load, ok = s.addUnits(load, ok, int(ln.Commodity), int64(ln.Quantity))
	}
	return load, ok
}

// roomCount is the counters of one (region, role) for RoomLimited (spec-storage.md
// 14.1): plain integer adds made always, because the start sizing needs them with no
// observer, cleared at the end of every step (P8.6).
type roomCount struct {
	// startsCut is the executions not started for room though the inputs allowed
	// them, startsRefused the requests refused whole (no_room), bidsRejected the
	// orders rejected no_room and shipmentsRefused the shipments refused
	// consignee_full.
	startsCut, startsRefused, bidsRejected, shipmentsRefused int64
	// bidsCut is the intents the default strategy sized down or dropped for room and
	// discarded the units it discarded to make room for tier 1 (6.3).
	bidsCut, discarded int64
	// limitedHouseholds is the households of limited agents given a view for Trade
	// and fullHouseholds those whose free room was 0 in a dimension then; they ride
	// along and trigger nothing.
	limitedHouseholds, fullHouseholds int64
}

// clearTick clears the counters at the end of a step (P8.6): none survives it.
func (s *storage) clearTick() { clear(s.counts) }

// roomFacts fills the room facts of a bid for the admission of the market (spec-storage.md
// 5.2.1): whether the bidder's account is limited, its free room and the size of one
// unit of the commodity. An ask, an unknown commodity and an account that is not
// limited leave them at their zero value.
func (s *storage) roomFacts(l *ledger.Ledger, a *agent, o Order, f *orderFacts) {
	if o.Side != Bid || o.Commodity < 0 || int(o.Commodity) >= len(s.sizes) {
		return
	}
	_, free, ok := s.roomState(l, a)
	if !ok {
		return
	}
	sz := s.sizes[o.Commodity]
	f.Limited, f.FreeMass, f.FreeBulk, f.SizeMass, f.SizeBulk = true, free.Mass, free.Bulk, sz.Mass, sz.Bulk
}

// bidRejected counts a bid rejected for room in the cell of its agent.
func (s *storage) bidRejected(cell int, reason Reason) {
	if reason == ReasonNoRoom {
		s.counts[cell].bidsRejected++
	}
}

// refusedStart counts a job request refused whole for room in the cell of its agent.
func (s *storage) refusedStart(cell int, reason Reason) {
	if reason == ReasonNoRoom {
		s.counts[cell].startsRefused++
	}
}

// pledgeForm is the form in which the output of a job is credited to the principal's
// account (spec-storage.md 4.1.3): from the pledge when the account is limited, the
// free form otherwise.
func (s *storage) pledgeForm(l *ledger.Ledger, account int) ledger.Credit {
	if rm, err := l.RoomOf(account); err == nil && rm.Limited {
		return ledger.CreditPledge
	}
	return ledger.CreditFree
}

// boundExecutions is the room bound of a start (spec-storage.md 5.3.1, 5.3.2,
// FR-STO-011, 012): the executions k0 the request could start, at most the largest k
// whose pending output fits the principal's free room less its upkeep room once the
// inputs are taken. A k of 0 where k0 was not is refused with no_room, which comes
// last in the order of the reasons. *cut is set to the executions the room took off.
// A principal whose account is not limited is not bounded.
func (s *storage) boundExecutions(l *ledger.Ledger, q *startReq, k0 int64, reason Reason, cut *int64) (int64, Reason) {
	if k0 < 1 || q.exec == nil {
		return k0, reason
	}
	rm, err := l.RoomOf(q.account)
	if err != nil || !rm.Limited {
		return k0, reason
	}
	view, err := l.Account(q.account)
	if err != nil {
		s.broke(q.principal, "the account of a principal is unreadable: "+err.Error())
	}
	m, b, _ := l.FreeRoom(q.account)
	up, ok := s.upkeepRoom(&s.world.agents[q.principal], RegionID(view.Region))
	if !ok {
		return 0, ReasonNoRoom
	}
	k := room.KRoom(s.recipeInputs(q.recipe), s.recipeOutputs(q), q.effective, room.Pair{Mass: m, Bulk: b}, up, k0)
	*cut = k0 - k
	if k == 0 {
		return 0, ReasonNoRoom
	}
	return k, reason
}

// recipeInputs lists the inputs of a recipe as quantities with the size of a unit, in
// a scratch the next call overwrites.
func (s *storage) recipeInputs(recipe RecipeID) []room.Item {
	s.in = s.in[:0]
	for _, in := range s.recipes.row(recipe).inputs {
		s.in = append(s.in, room.Item{Qty: int64(in.quantity), Size: s.sizes[in.commodity]})
	}
	return s.in
}

// recipeOutputs lists the outputs of a recipe with the size of a unit, the executor's
// residue of each (spec-production.md 6.6) and whether it is a discard output, in a
// scratch the next call overwrites. An executor with no residues reads them as 0.
func (s *storage) recipeOutputs(q *startReq) []room.Output {
	s.out = s.out[:0]
	row := s.recipes.row(q.recipe)
	owner := s.world.residueOwner(q.exec) // a trainee's or a hired job takes its principal's residues
	off, _ := residueOffset(s.roles.row(owner.role).jobs, s.recipes, q.recipe)
	for i, o := range row.outputs {
		var residue int64
		if off+i < len(owner.residues) {
			residue = owner.residues[off+i]
		}
		s.out = append(s.out, room.Output{Item: room.Item{Qty: int64(o.quantity), Size: s.sizes[o.commodity]}, Residue: residue, Discard: o.discard})
	}
	return s.out
}

// fitRoom lowers the units of one output line of a completion to what the room the
// account has left holds (spec-storage.md 5.3.4, 4.1.3): whole units, in each
// dimension, after the lines already cut for the same completion. An account that is
// not limited holds all of fit. The pledge is never above the room except after a cut
// of the room, so this is the cut at the credit.
func (s *storage) fitRoom(l *ledger.Ledger, account int, c CommodityID, fit int64, cut []ledger.Line) int64 {
	rm, err := l.RoomOf(account)
	if err != nil || !rm.Limited {
		return fit
	}
	left := s.roomLeftPair(l, account)
	for _, ln := range cut {
		sz := s.sizes[ln.Commodity]
		m, okm := checked.Mul(ln.Quantity, sz.Mass)
		b, okb := checked.Mul(ln.Quantity, sz.Bulk)
		if !okm || !okb {
			return 0
		}
		left.Mass, left.Bulk = max(0, left.Mass-m), max(0, left.Bulk-b)
	}
	sz := s.sizes[c]
	if sz.Mass > 0 {
		fit = min(fit, left.Mass/sz.Mass)
	}
	if sz.Bulk > 0 {
		fit = min(fit, left.Bulk/sz.Bulk)
	}
	return fit
}

// roomLeftPair is RoomLeft of the account of a limited agent as a pair, 0 where it
// would be negative; an unreadable room is a violated invariant.
func (s *storage) roomLeftPair(l *ledger.Ledger, account int) room.Pair {
	m, b, err := l.RoomLeft(account)
	if err != nil {
		invariant("the room left of a limited account is unreadable: " + err.Error())
	}
	return room.Pair{Mass: m, Bulk: b}
}

// startedStorage is the step of a start that the module takes (spec-storage.md 5.3.3):
// the executions the room cut are counted and the principal's pledge is set to the
// pending output's load plus the cargo in flight.
func (w *World) startedStorage(a *agent, region RegionID, res startResult) {
	c := &w.sto.counts[w.cellOf(region, a)]
	c.startsCut = satAdd(c.startsCut, res.cut) // a sum over every start of the cell: saturates
	w.syncJob(a)
}

// weightChangedStorage is the step of a change of an agent's weight (spec-storage.md
// 5.5.3): the room is the new weight's, and the pledge is derived again from the job,
// which a reduction scaled. A reduction never takes the load above the room (the
// goods were scaled by the same factor); the pledge may exceed it and is cut at the
// credit.
func (w *World) weightChangedStorage(a *agent) {
	if err := w.sto.apply(w.ledger, a); err != nil {
		w.sto.broke(a.id, "a change of weight took the room of an account below its load: "+err.Error())
	}
	w.syncJob(a)
}

// tripRefusal is row 3a of the shipment checks (spec-storage.md 9.3, FR-STO-031): a
// self-carried request of an agent that owns a store, which a trip would strand: it
// ends in a change of region that the region_bound refusal has_store would hold for
// ever. A hired request moves goods, not the owner, and is not refused.
func (s *storage) tripRefusal(c *shipCtx) Reason {
	if !c.md.hired() && s.owned(c.a.id) > 0 {
		return ReasonHasStore
	}
	return ""
}

// consigneeRefusal is row 11a of the shipment checks (spec-storage.md 5.4.1,
// FR-STO-014, FR-STO-046): a consignee other than the shipper whose limited account
// has a free room, less its upkeep room, below the load of the lines in either
// dimension. The free room is read now, after the pledges of every earlier request of
// the tick. The rows 12 to 16 have not run, so a line the later rows will refuse (an
// unknown commodity, a quantity out of its bounds) adds nothing here. A load or an
// upkeep room that does not fit int64 is above any room.
func (s *storage) consigneeRefusal(c *shipCtx) Reason {
	r := c.r
	if r.ToSelf || r.Consignee == c.a.id {
		return ""
	}
	cons := &c.w.agents[r.Consignee]
	_, free, ok := s.roomState(c.w.ledger, cons)
	if !ok {
		return ""
	}
	load, okl := s.tolerantLoad(r.Lines)
	if !okl {
		return s.refuseShipment(c)
	}
	up, okup := s.upkeepRoom(cons, c.w.regionOf(cons))
	if !okup || free.Mass-up.Mass < load.Mass || free.Bulk-up.Bulk < load.Bulk {
		return s.refuseShipment(c)
	}
	return ""
}

// tolerantLoad is the load of the lines of a request that rows 12 to 16 have not
// checked: a line that names an unknown commodity or a quantity out of its bounds
// adds nothing, as the later rows refuse it. ok is false when the sum overflows.
func (s *storage) tolerantLoad(lines []Line) (load room.Pair, ok bool) {
	for _, ln := range lines {
		if ln.Commodity < 0 || int(ln.Commodity) >= len(s.sizes) || ln.Quantity < 1 || ln.Quantity > MaxQuantity {
			continue
		}
		var okl bool
		if load, okl = s.addUnits(load, true, int(ln.Commodity), int64(ln.Quantity)); !okl {
			return room.Pair{}, false
		}
	}
	return load, true
}

// refuseShipment counts a shipment refused for the consignee's room in the cell of its
// shipper and returns the reason.
func (s *storage) refuseShipment(c *shipCtx) Reason {
	s.counts[c.w.cellOf(c.region, c.a)].shipmentsRefused++
	return ReasonConsigneeFull
}

// shippedStorage is the step of a dispatch that the module takes (spec-storage.md
// 5.4.1, 4.3): the load of the lines is added to the cargo in flight to the consignee
// and its pledge is derived again. A consignee that is not limited has no pledge; the
// cargo is tracked all the same, so a role change to a limited role finds it.
func (w *World) shippedStorage(sh *shipment) {
	s := w.sto
	load, ok := s.linesLoad(sh.lines)
	sum, ok2 := addPair(s.inbound[sh.consignee], load)
	if !ok || !ok2 {
		s.broke(sh.consignee, "the load of the cargo in flight to an agent does not fit int64")
	}
	s.inbound[sh.consignee] = sum
	w.repledge(&w.agents[sh.consignee])
}

// arrivedStorage is the step of an arrival that the module takes (spec-storage.md
// 5.4.4): the load of the lines, whatever part was delivered, spoiled or diverted,
// leaves the cargo in flight to the consignee and its pledge is derived again. A
// consignee that was removed on the way has no state to change.
func (w *World) arrivedStorage(sh *shipment) {
	s := w.sto
	if len(sh.lines) == 0 || !w.liveAgent(sh.consignee) {
		return
	}
	load, ok := s.linesLoad(sh.lines)
	left, ok2 := checked.Sub(s.inbound[sh.consignee].Mass, load.Mass)
	leftB, ok3 := checked.Sub(s.inbound[sh.consignee].Bulk, load.Bulk)
	if !ok || !ok2 || !ok3 || left < 0 || leftB < 0 {
		s.broke(sh.consignee, "an arrival is larger than the cargo in flight to its consignee")
	}
	s.inbound[sh.consignee] = room.Pair{Mass: left, Bulk: leftB}
	w.repledge(&w.agents[sh.consignee])
}

// cargoFit is the units of the line of a cargo, of the have units that reached the
// consignee, that its room holds (spec-storage.md 5.4.4): whole units, per dimension,
// of the room left, and all of have for an account that is not limited. It is the
// backstop that a cut of the room makes reachable.
func (s *storage) cargoFit(l *ledger.Ledger, account int, c CommodityID, have int64) int64 {
	rm, err := l.RoomOf(account)
	if err != nil || !rm.Limited {
		return have
	}
	left := s.roomLeftPair(l, account)
	sz := s.sizes[c]
	if sz.Mass > 0 {
		have = min(have, left.Mass/sz.Mass)
	}
	if sz.Bulk > 0 {
		have = min(have, left.Bulk/sz.Bulk)
	}
	return have
}

// storesTarget says whether the target is a class of store owners and which store
// class it names (-1 for every class, 0 when the target is not stores), for newFlowState. The class is validated, so
// it is in the configuration (spec-money-supply.md 4.3.4, FR-STO-026).
func storesTarget(c *Config, t FlowTarget) (stores bool, class int) {
	if t.Kind != targetStores {
		return false, 0 // zero off the kind: the hashed state of a flow that is not a stores flow does not move
	}
	class = -1
	if t.Store != "" {
		if c.Modules.Storage == nil {
			invariant("a stores flow's module is off after validation: " + t.Store)
		}
		if class = slices.IndexFunc(c.Modules.Storage.Classes, func(k StoreClass) bool { return k.ID == t.Store }); class < 0 {
			invariant("a stores flow's store class does not resolve after validation: " + t.Store) // -1 would mean every class
		}
	}
	return true, class
}

// scanOwners lists, per region, the live agents in ascending AgentID for the flows
// of a stores target: a stores class is every open agent account that owns a store
// of the class in the region, in ascending account ID, whatever its role. Membership
// is read at the position of the flow, so a store added between ticks counts from the
// next P1.7 (spec-money-supply.md 4.3.4). Nothing is built when no flow has one.
func (w *World) scanOwners(m *moneyFlows) {
	if m.owners == nil {
		return
	}
	for g := range m.owners {
		m.owners[g] = m.owners[g][:0]
	}
	for id := m.regions; id < len(w.agents); id++ {
		if a := &w.agents[id]; a.weight >= 1 {
			g := int(w.regionOf(a))
			m.owners[g] = append(m.owners[g], AgentID(id))
		}
	}
}

// visitStores applies flow i of a stores target to the owners in region g, in
// ascending account ID: each owner of n matching stores is due n times the amount
// (a store is not a household), with its carry kept per account. An agent with no
// matching store is not visited.
func (w *World) visitStores(m *moneyFlows, i, g int) {
	f := &m.flows[i]
	for _, id := range m.owners[g] {
		a := &w.agents[id]
		n := w.sto.owned(a.id)
		if f.class >= 0 {
			n = w.sto.count(a.id, f.class)
		}
		if n < 1 {
			continue
		}
		w.visitFlow(m, i, g, recipient{account: a.account, weight: int64(n), carry: &f.carry[a.id], agent: a})
	}
}

// storeRefusal is the checks of addStore in the order of the table of 3.6.2, with no
// effect: nil when a store of the class can be added to the agent. pending is the stores
// that the caller has decided to add to the agent before this one and has not yet added,
// so a call that adds several (AddFacilities) can check them all before it applies any.
func (w *World) storeRefusal(a *agent, class StoreClassID, pending int) error {
	s := w.sto
	switch {
	case class < 0 || int(class) >= len(s.classes):
		return ErrUnknownStoreClass
	case s.gate != nil && !s.gate(w.regionOf(a), int(class)):
		return ErrStoreLocked
	case !s.limited(a):
		return ErrNotLimited
	case w.away(a):
		return ErrAgentAway
	case s.owned(a.id)+pending >= MaxStoresPerAgent:
		return ErrOutOfBounds
	}
	return nil
}
