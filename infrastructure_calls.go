package economysim

import (
	"cmp"
	"fmt"
	"slices"
)

// This file is the public face of the infrastructure module (spec-infrastructure.md 11):
// three calls between ticks, AddFacilities, AdjustFacilities and TransferFacilities, and
// two queries, Facility and Facilities, with the step of a removal (7.1) and the ruin of a
// facility, which the calls and the later steps share. The checks of a call follow the
// order of spec-integration.md 3.2 and each call is atomic: it validates every entry and
// only then applies any, takes no random number and moves no money.

// NewFacility is one facility to create between ticks (spec-infrastructure.md 11.1).
type NewFacility struct {
	Class  FacilityClassID
	Region RegionID
	// Toward is the other end of the edge a route speeds; NoRegion for every edge that
	// leaves Region, and for every kind but a route.
	Toward RegionID
	// Owner is a household or external agent, or the estate pseudo-agent of Region.
	Owner AgentID
	// Condition is the condition of the new facility: 0 means the class's full
	// Durability, else 1 to Durability.
	Condition int64
}

// FacilityAdjustment changes the condition of one facility by a signed delta
// (spec-infrastructure.md 11.2).
type FacilityAdjustment struct {
	ID FacilityID
	// Delta is signed; the result is clamped to [0, Durability] and 0 ruins the
	// facility. |Delta| is at most MaxDurability.
	Delta int64
}

// FacilityTransfer gives one facility to another owner (spec-infrastructure.md 11.3).
type FacilityTransfer struct {
	ID FacilityID
	// To is a live household or external agent, or the estate of the facility's region.
	To AgentID
}

// addPlan is one entry of AddFacilities after its checks: the facility to create.
type addPlan struct {
	class     FacilityClassID
	region    RegionID
	to        RegionID
	owner     AgentID
	condition int64
}

// addBudget is what the entries of one AddFacilities call that were already checked take
// from the bounds, so that an entry is checked against the call as it would be applied.
type addBudget struct {
	total    int
	perOwner map[AgentID]int // lookup only
	stores   map[AgentID]int // lookup only
}

// AddFacilities creates the facilities in slice order and returns their IDs, which are
// the next IDs in that order (spec-infrastructure.md 11.1). It acts between ticks and is
// atomic. It returns ErrInStep, ErrWorldFailed, ErrModuleOff, then for the first entry
// that fails, as a *ValidationError, ErrUnknownFacilityClass, ErrUnknownRegion,
// ErrOutsideNode, ErrSameRegion, ErrUnknownEdge, ErrFacilityShape, ErrOutOfBounds (a
// condition above the Durability or below 0), ErrUnknownAgent, ErrAgentRemoved,
// ErrFacilityOwner, ErrFacilityRegion, ErrFacilityLocked, ErrFacilityTarget,
// ErrFacilityCap or the error of a store (ErrNotLimited, ErrAgentAway, ErrStoreLocked,
// ErrOutOfBounds). A store facility is installed in the call: storage emits StoreChanged
// (cause facility) and then FacilityChanged (added, cause consumer) is emitted for it.
func (w *World) AddFacilities(adds []NewFacility) ([]FacilityID, error) {
	if err := w.enter("AddFacilities", "infrastructure", w.infra != nil); err != nil {
		return nil, err
	}
	n := w.infra
	if len(adds) > MaxFacilities {
		return nil, fmt.Errorf("AddFacilities: %w", argError(ErrOutOfBounds, itoa(len(adds)), "MaxFacilities", "adds"))
	}
	plan := make([]addPlan, len(adds))
	budget := addBudget{total: len(n.fac), perOwner: map[AgentID]int{}, stores: map[AgentID]int{}}
	for i := range adds {
		p, err := n.checkAdd(i, &adds[i], &budget)
		if err != nil {
			return nil, fmt.Errorf("AddFacilities: %w", err)
		}
		plan[i] = p
	}
	ids := make([]FacilityID, len(plan))
	for i, p := range plan {
		ids[i] = n.apply(p)
	}
	n.rebuild()
	return ids, nil
}

// checkAdd is the checks of one entry (spec-infrastructure.md 11.1): the class, the region
// and target, the condition, the owner, and what the class needs of the place and the owner.
func (n *infrastructure) checkAdd(i int, e *NewFacility, b *addBudget) (addPlan, error) {
	name := entry("adds", i)
	if e.Class < 0 || int(e.Class) >= len(n.cls) {
		return addPlan{}, argError(ErrUnknownFacilityClass, itoa(int(e.Class)), "", name, "class")
	}
	c := &n.cls[e.Class]
	if err := n.checkAddPlace(name, e, c); err != nil {
		return addPlan{}, err
	}
	if e.Condition < 0 || e.Condition > c.durability {
		return addPlan{}, argError(ErrOutOfBounds, i64(e.Condition), i64(c.durability), name, "condition")
	}
	if err := n.checkCallOwner(name, "owner", e.Owner, e.Region, c.kind == kindStore); err != nil {
		return addPlan{}, err
	}
	if err := n.checkAddNeeds(name, e, c, b); err != nil {
		return addPlan{}, err
	}
	if err := n.checkAddCap(name, e.Owner, b); err != nil {
		return addPlan{}, err
	}
	cond := cmp.Or(e.Condition, c.durability)
	return addPlan{class: e.Class, region: e.Region, to: e.Toward, owner: e.Owner, condition: cond}, nil
}

// checkAddCap counts the entry against the bounds of the world and of its owner, with the
// entries of the call before it (ErrFacilityCap).
func (n *infrastructure) checkAddCap(name string, owner AgentID, b *addBudget) error {
	b.total++
	b.perOwner[owner]++
	if b.total > MaxFacilities {
		return argError(ErrFacilityCap, itoa(b.total), itoa(MaxFacilities), name)
	}
	if held := n.held(owner) + b.perOwner[owner]; !n.estateOwner(owner) && held > MaxFacilitiesPerAgent {
		return argError(ErrFacilityCap, itoa(held), itoa(MaxFacilitiesPerAgent), name, "owner")
	}
	return nil
}

// checkAddPlace checks the region of an entry and its target: an ordinary region, and for
// a route a target that is another region joined to it by an edge in either direction;
// no other kind has one.
func (n *infrastructure) checkAddPlace(name string, e *NewFacility, c *facClass) error {
	if err := n.world.facilityRegion(name, "region", e.Region); err != nil {
		return err
	}
	if e.Toward == NoRegion {
		return nil
	}
	if c.kind != kindRoute {
		return argError(ErrFacilityShape, itoa(int(e.Toward)), "", name, "toward")
	}
	if err := n.world.facilityRegion(name, "toward", e.Toward); err != nil {
		return err
	}
	switch {
	case e.Toward == e.Region:
		return argError(ErrSameRegion, itoa(int(e.Toward)), "", name, "toward")
	case !n.joined(e.Region, e.Toward):
		return argError(ErrUnknownEdge, itoa(int(e.Region))+" to "+itoa(int(e.Toward)), "", name, "toward")
	}
	return nil
}

// facilityRegion checks that r is a region of the configuration and not an outside node.
func (w *World) facilityRegion(name, field string, r RegionID) error {
	if r < 0 || int(r) >= len(w.cfg.Regions) {
		return argError(ErrUnknownRegion, itoa(int(r)), "", name, field)
	}
	if w.isOutsideNode(r) {
		return argError(ErrOutsideNode, itoa(int(r)), "", name, field)
	}
	return nil
}

// checkCallOwner checks an owner named by a call: a live household or external agent, or the
// estate pseudo-agent of the facility's region (spec-infrastructure.md 11.1, 11.3). Any
// other pseudo-agent is ErrFacilityOwner, an ID never issued ErrUnknownAgent and a removed
// agent ErrAgentRemoved. A store facility has no estate owner (ErrFacilityShape).
func (n *infrastructure) checkCallOwner(name, field string, owner AgentID, region RegionID, store bool) error {
	w := n.world
	switch {
	case owner < 0 || int(owner) >= len(w.agents):
		return argError(ErrUnknownAgent, itoa(int(owner)), "", name, field)
	case owner == AgentID(region) && owner < AgentID(n.regions):
		if store {
			return argError(ErrFacilityShape, itoa(int(owner)), "", name, field)
		}
		return nil
	case int(owner) < w.firstHousehold:
		return argError(ErrFacilityOwner, itoa(int(owner)), "", name, field)
	case w.agents[owner].weight < 1:
		return argError(ErrAgentRemoved, itoa(int(owner)), "", name, field)
	}
	return nil
}

// checkAddNeeds checks what the class needs of the place and of the owner: the settlement
// flag of its Needs (ErrFacilityLocked), an endowment of the resource of a ground class
// with a capacity above 0 (ErrFacilityTarget), and for a store class an owner whose
// account is in the region (ErrFacilityRegion) and that storage would give the store to,
// counting the stores of the entries before this one (spec-infrastructure.md 5.4.3).
func (n *infrastructure) checkAddNeeds(name string, e *NewFacility, c *facClass, b *addBudget) error {
	w := n.world
	if c.flag >= 0 && !w.regionHas(e.Region, c.flag) {
		return argError(ErrFacilityLocked, itoa(int(e.Region)), "", name, "class")
	}
	switch c.kind {
	case kindGround:
		if w.res == nil || w.res.capacity[w.res.cellOf(e.Region, ResourceID(c.resource))] < 1 {
			return argError(ErrFacilityTarget, itoa(int(e.Region)), "", name, "region")
		}
	case kindStore:
		a := w.agentOf(e.Owner)
		if w.regionOf(a) != e.Region {
			return argError(ErrFacilityRegion, itoa(int(e.Owner)), itoa(int(e.Region)), name, "owner")
		}
		if err := w.storeRefusal(a, c.store, b.stores[e.Owner]); err != nil {
			return argError(err, itoa(int(e.Owner)), "", name, "class")
		}
		b.stores[e.Owner]++
	}
	return nil
}

// apply creates the facility of a checked entry, installs its store when it is a store
// class, and emits its events: storage's StoreChanged and then FacilityChanged (added,
// consumer). It returns the new ID.
func (n *infrastructure) apply(p addPlan) FacilityID {
	w := n.world
	f := n.create(p.class, p.region, p.to, p.owner, p.condition, w.Tick())
	if c := &n.cls[p.class]; c.kind == kindStore {
		if err := w.addStore(w.agentOf(p.owner), c.store, CauseFacility); err != nil {
			invariant("a store facility could not be installed after its dry run: " + err.Error())
		}
		n.install(len(n.fac) - 1)
	}
	w.reportFacility(&n.fac[len(n.fac)-1], FacilityAdded, CauseConsumer, 0, 0)
	return f.id
}

// AdjustFacilities changes the condition of facilities by signed deltas, clamped to [0,
// Durability]; a result of 0 ruins the facility at once (spec-infrastructure.md 11.2). It
// acts between ticks and is atomic. It returns ErrInStep, ErrWorldFailed, ErrModuleOff,
// then for the first entry that fails, as a *ValidationError, ErrUnknownFacility or
// ErrOutOfBounds (a delta above MaxDurability in magnitude), and ErrDuplicateID for a
// facility that appears twice. An entry whose result equals the condition is accepted and
// is not a change. It emits FacilityChanged (cause consumer) for each change in ascending
// facility ID, after storage's StoreChanged for the store of a ruined store facility.
func (w *World) AdjustFacilities(changes []FacilityAdjustment) error {
	if err := w.enter("AdjustFacilities", "infrastructure", w.infra != nil); err != nil {
		return err
	}
	n := w.infra
	ids, err := n.checkIDs("AdjustFacilities", "changes", len(changes), func(i int) (FacilityID, error) {
		if d := changes[i].Delta; d > MaxDurability || d < -MaxDurability {
			return 0, argError(ErrOutOfBounds, i64(d), itoa(MaxDurability), entry("changes", i), "delta")
		}
		return changes[i].ID, nil
	})
	if err != nil {
		return err
	}
	for _, i := range byID(ids) {
		n.adjust(changes[i])
	}
	n.rebuild()
	return nil
}

// checkIDs is the checks that AdjustFacilities and TransferFacilities share: the length of
// the slice, that each entry names a facility (after the entry's own check) and that no
// facility appears twice. It returns the facility ID of each entry.
func (n *infrastructure) checkIDs(call, field string, size int, own func(i int) (FacilityID, error)) ([]FacilityID, error) {
	if size > MaxFacilities {
		return nil, fmt.Errorf("%s: %w", call, argError(ErrOutOfBounds, itoa(size), "MaxFacilities", field))
	}
	ids := make([]FacilityID, size)
	for i := range ids {
		id, err := own(i)
		if err == nil && n.index(id) < 0 {
			err = argError(ErrUnknownFacility, i64(int64(id)), "", entry(field, i), "id")
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", call, err)
		}
		ids[i] = id
	}
	if d := keyedOnce(ids); d >= 0 {
		return nil, fmt.Errorf("%s: %w", call, argError(ErrDuplicateID, i64(int64(ids[d])), "", entry(field, d), "id"))
	}
	return ids, nil
}

// byID is the indexes of ids in ascending facility ID.
func byID(ids []FacilityID) []int {
	idx := make([]int, len(ids))
	for i := range idx {
		idx[i] = i
	}
	slices.SortFunc(idx, func(a, b int) int { return cmp.Compare(ids[a], ids[b]) })
	return idx
}

// adjust applies one checked entry: the new condition is clamped to [0, Durability]; an
// unchanged condition is no change, a result of 0 ruins the facility and any other change
// is reported with the kind adjusted.
func (n *infrastructure) adjust(c FacilityAdjustment) {
	i := n.index(c.ID)
	f := &n.fac[i]
	dur := n.cls[f.class].durability
	after := min(max(f.condition+c.Delta, 0), dur)
	before := f.condition
	switch {
	case after == before:
	case after == 0:
		n.world.ruinFacility(i, CauseConsumer)
	default:
		f.condition = after
		n.world.reportFacility(f, FacilityAdjusted, CauseConsumer, before, 0)
	}
}

// ruinFacility ends the facility at position i with the cause (spec-infrastructure.md
// 6.2): a store's store is removed by the forced form of storage's removal, which emits
// StoreChanged (removed, cause facility) first and lists whole lots to the estate; the
// record is then removed and FacilityChanged (ruined) carries the lots listed. The caller
// rebuilds the tables.
func (w *World) ruinFacility(i int, cause Cause) {
	n := w.infra
	f := n.fac[i]
	var goods int64
	if f.installed {
		n.uninstall(f.owner, f.class) // the facility no longer holds its store, so the forced removal is not refused
		n.fac[i].installed = false
		lots, err := w.removeStore(w.agentOf(f.owner), n.cls[f.class].store, CauseFacility, true)
		if err != nil {
			invariant("the store of a ruined store facility could not be removed: " + err.Error())
		}
		goods = lots
	}
	n.remove(i)
	ended := f
	ended.condition = 0
	w.reportFacility(&ended, FacilityRuined, cause, f.condition, goods)
}

// TransferFacilities changes the owners of facilities (spec-infrastructure.md 11.3). It
// moves no money and acts between ticks, atomically. It returns ErrInStep, ErrWorldFailed,
// ErrModuleOff, then for the first entry that fails, as a *ValidationError,
// ErrUnknownFacility, ErrFacilityNotTransferable (a store facility), ErrUnknownAgent,
// ErrAgentRemoved, ErrFacilityOwner (a pseudo-agent that is not the estate of the
// facility's region) or ErrFacilityCap (the new owner would hold more than
// MaxFacilitiesPerAgent), and ErrDuplicateID for a facility that appears twice. A transfer
// to the owner it has is no change. It emits FacilityChanged (transferred, cause consumer)
// for each change in ascending facility ID.
func (w *World) TransferFacilities(moves []FacilityTransfer) error {
	if err := w.enter("TransferFacilities", "infrastructure", w.infra != nil); err != nil {
		return err
	}
	n := w.infra
	ids, err := n.checkIDs("TransferFacilities", "moves", len(moves), func(i int) (FacilityID, error) {
		return moves[i].ID, nil
	})
	if err != nil {
		return err
	}
	if err := n.checkMoves(moves); err != nil {
		return fmt.Errorf("TransferFacilities: %w", err)
	}
	for _, i := range byID(ids) {
		n.move(moves[i])
	}
	n.rebuild()
	return nil
}

// checkMoves checks the entries of TransferFacilities one by one: a store facility is not
// transferable, the new owner is valid for the facility's region and does not pass the
// bound of an owner, counting the entries before it that give it a facility.
func (n *infrastructure) checkMoves(moves []FacilityTransfer) error {
	taken := map[AgentID]int{} // lookup only
	for i, m := range moves {
		f := &n.fac[n.index(m.ID)]
		name := entry("moves", i)
		if n.cls[f.class].kind == kindStore {
			return argError(ErrFacilityNotTransferable, i64(int64(m.ID)), "", name, "id")
		}
		if err := n.checkCallOwner(name, "to", m.To, f.region, false); err != nil {
			return err
		}
		if m.To == f.owner {
			continue
		}
		taken[m.To]++
		if !n.estateOwner(m.To) && n.held(m.To)+taken[m.To] > MaxFacilitiesPerAgent {
			return argError(ErrFacilityCap, itoa(n.held(m.To)+taken[m.To]), itoa(MaxFacilitiesPerAgent), name, "to")
		}
	}
	return nil
}

// move gives one checked facility to its new owner: no change when it has that owner.
func (n *infrastructure) move(m FacilityTransfer) {
	i := n.index(m.ID)
	if n.fac[i].owner == m.To {
		return
	}
	n.give(i, m.To)
	n.world.reportFacility(&n.fac[i], FacilityTransferred, CauseConsumer, n.fac[i].condition, 0)
}

// dropFacilities is step 2g of a removal (spec-infrastructure.md 7.1.1): the removed
// agent's facilities, in ascending ID, after its claims and before its money and goods
// move. A site, route or ground facility passes to the estate of its region
// (FacilityChanged transferred, cause owner_removed); a store facility is ruined with the
// same cause, with no StoreChanged and no lots listed, because storage ends the stores
// with the account. The events come before AgentBankrupted. It is nil-safe: with the
// module off it does nothing.
func (w *World) dropFacilities(a *agent) {
	n := w.infra
	if n == nil || len(n.byOwner[a.id]) == 0 {
		return // nothing held: skip the full rebuild of the effect tables
	}
	for _, id := range slices.Clone(n.byOwner[a.id]) {
		i := n.index(id)
		f := &n.fac[i]
		if n.cls[f.class].kind == kindStore {
			ended := *f
			before := f.condition
			ended.condition = 0
			n.remove(i)
			w.reportFacility(&ended, FacilityRuined, CauseOwnerRemoved, before, 0)
			continue
		}
		n.give(i, AgentID(f.region))
		w.reportFacility(f, FacilityTransferred, CauseOwnerRemoved, f.condition, 0)
	}
	n.rebuild()
}

// Facility returns one facility: its record, its class, region, target, owner, condition,
// the tick it was built, whether it is installed, and the derived Effect (eff, or cap for
// a site) (spec-infrastructure.md 11.4). It returns ErrInStep, ErrWorldFailed,
// ErrModuleOff and ErrUnknownFacility. It draws nothing, emits nothing and returns a copy.
func (w *World) Facility(id FacilityID) (FacilityState, error) {
	if err := w.enter("Facility", "infrastructure", w.infra != nil); err != nil {
		return FacilityState{}, err
	}
	i := w.infra.index(id)
	if i < 0 {
		return FacilityState{}, fmt.Errorf("Facility: facility %d: %w", id, ErrUnknownFacility)
	}
	return w.infra.state(&w.infra.fac[i]), nil
}

// Facilities returns the facilities that stand in the region, in ascending ID
// (spec-infrastructure.md 11.4). It returns ErrInStep, ErrWorldFailed, ErrModuleOff,
// ErrUnknownRegion and ErrOutsideNode. It draws nothing, emits nothing and returns a copy.
func (w *World) Facilities(r RegionID) ([]FacilityState, error) {
	if err := w.enter("Facilities", "infrastructure", w.infra != nil); err != nil {
		return nil, err
	}
	if err := w.facilityRegion("region", "region", r); err != nil {
		return nil, fmt.Errorf("Facilities: %w", err)
	}
	var out []FacilityState
	for i := range w.infra.fac {
		if f := &w.infra.fac[i]; f.region == r {
			out = append(out, w.infra.state(f))
		}
	}
	return out, nil
}
