package economysim

import (
	"fmt"
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// This file is the bookkeeping of the facility records (spec-infrastructure.md 3.2):
// creation, removal, change of owner and the indexes that find a facility by its owner.
// It decides nothing: the rules are in the calls and the step. The saved form of the
// records (tag 27) is at the end of the file.

// create appends a facility with the next ID and indexes it. The ID is never reused.
// The caller has checked the bounds.
func (n *infrastructure) create(class FacilityClassID, region, to RegionID, owner AgentID, condition, tick int64) *facility {
	n.fac = append(n.fac, facility{id: n.next, class: class, region: region, to: to, owner: owner, condition: condition, built: tick})
	n.byOwner[owner] = append(n.byOwner[owner], n.next)
	n.next++
	return &n.fac[len(n.fac)-1]
}

// installedOf is the number of installed store facilities of the store class that the
// agent has: the seam of storage's RemoveStore (spec-storage.md 3.6.2, ErrStoreInUse).
// It is the sum over the facility classes that install that store class.
func (n *infrastructure) installedOf(a *agent, store StoreClassID) int {
	total := 0
	for c := range n.cls {
		if n.cls[c].kind == kindStore && n.cls[c].store == store {
			total += n.installed[ownerClass{a.id, FacilityClassID(c)}]
		}
	}
	return total
}

// ownedBy lists the facilities of an owner in ascending ID, as public states.
func (n *infrastructure) ownedBy(owner AgentID) []FacilityState {
	ids := n.byOwner[owner]
	if len(ids) == 0 {
		return nil
	}
	out := make([]FacilityState, len(ids))
	for k, id := range ids {
		out[k] = n.state(&n.fac[n.index(id)])
	}
	return out
}

// facilityClassNames is the string ID of each FacilityClassID, nil with the module off
// (Names.FacilityClasses).
func facilityClassNames(c *Config) []string {
	m := c.Modules.Infrastructure
	if m == nil {
		return nil
	}
	out := make([]string, len(m.Classes))
	for i := range m.Classes {
		out[i] = m.Classes[i].ID
	}
	return out
}

// estateOwner reports whether the owner is the estate pseudo-agent of a region: an ID
// below the region count, equal to the RegionID (spec-infrastructure.md 3.2.1).
func (n *infrastructure) estateOwner(owner AgentID) bool { return owner >= 0 && int(owner) < n.regions }

// checkRecord checks one record on its own (spec-infrastructure.md 15.3): its class, its
// condition, its region and target and its owner. It does not check the order of the IDs
// or the counts, which the caller does over the whole list.
func (n *infrastructure) checkRecord(f *facility) error {
	if f.class < 0 || int(f.class) >= len(n.cls) {
		return fmt.Errorf("facility %d: class %d is not a class of the configuration", f.id, f.class)
	}
	c := &n.cls[f.class]
	if f.condition < 1 || f.condition > c.durability {
		return fmt.Errorf("facility %d: condition %d is outside 1 to %d", f.id, f.condition, c.durability)
	}
	if f.installed && c.kind != kindStore {
		return fmt.Errorf("facility %d: installed is set for a class that is not a store", f.id)
	}
	if err := n.checkPlace(f, c); err != nil {
		return err
	}
	return n.checkOwner(f, c)
}

// checkPlace checks the region of a record, and its target: a route's other end is a region
// joined to its own by an edge, and no other kind has one.
func (n *infrastructure) checkPlace(f *facility, c *facClass) error {
	if f.region < 0 || int(f.region) >= n.regions || n.world.isOutsideNode(f.region) {
		return fmt.Errorf("facility %d: region %d is not an ordinary region", f.id, f.region)
	}
	if f.to == NoRegion {
		return nil
	}
	if c.kind != kindRoute || f.to < 0 || int(f.to) >= n.regions || f.to == f.region || !n.joined(f.region, f.to) {
		return fmt.Errorf("facility %d: target %d is not a region joined to region %d by a route class", f.id, f.to, f.region)
	}
	return nil
}

// checkOwner checks that the owner is a live agent or the estate of the facility's
// region, and that a store facility is held by an agent whose account is in its region
// when installed.
func (n *infrastructure) checkOwner(f *facility, c *facClass) error {
	w := n.world
	if n.estateOwner(f.owner) {
		if f.owner != AgentID(f.region) || c.kind == kindStore {
			return fmt.Errorf("facility %d: owner %d is not an estate that may hold it", f.id, f.owner)
		}
		return nil
	}
	if int(f.owner) < w.firstHousehold || int(f.owner) >= len(w.agents) || w.agents[f.owner].weight < 1 {
		return fmt.Errorf("facility %d: owner %d is not a live agent", f.id, f.owner)
	}
	if f.installed && w.regionOf(&w.agents[f.owner]) != f.region {
		return fmt.Errorf("facility %d: the account of owner %d is not in region %d", f.id, f.owner, f.region)
	}
	return nil
}

// auditInfrastructure checks the identities of the records (spec-infrastructure.md 15.3,
// 3.2.3): ascending unique IDs below the next ID, every record valid, the counts of the
// world and of an owner, the index by owner, and the installed store facilities at most
// the stores of the class the owner holds. It is nil with the module off.
func (w *World) auditInfrastructure() error {
	n := w.infra
	if n == nil {
		return nil
	}
	if len(n.fac) > MaxFacilities {
		return fmt.Errorf("infrastructure: %d facilities, above %d", len(n.fac), MaxFacilities)
	}
	indexed := 0
	for i := range n.fac {
		f := &n.fac[i]
		if f.id < 1 || f.id >= n.next || (i > 0 && n.fac[i-1].id >= f.id) {
			return fmt.Errorf("infrastructure: facility %d is out of order or not below the next ID %d", f.id, n.next)
		}
		if err := n.checkRecord(f); err != nil {
			return fmt.Errorf("infrastructure: %w", err)
		}
		if _, ok := slices.BinarySearch(n.byOwner[f.owner], f.id); !ok {
			return fmt.Errorf("infrastructure: facility %d is not in the list of owner %d", f.id, f.owner)
		}
		indexed++
	}
	return n.auditCounts(indexed)
}

// auditCounts checks the lists by owner and the installed counts against the records.
func (n *infrastructure) auditCounts(indexed int) error {
	listed := 0
	for owner, ids := range n.byOwner {
		listed += len(ids)
		if !n.estateOwner(owner) && len(ids) > MaxFacilitiesPerAgent {
			return fmt.Errorf("infrastructure: owner %d holds %d facilities, above %d", owner, len(ids), MaxFacilitiesPerAgent)
		}
	}
	if listed != indexed {
		return fmt.Errorf("infrastructure: the lists by owner hold %d facilities, the records %d", listed, indexed)
	}
	for k, count := range n.installed {
		c := &n.cls[k.class]
		if count < 1 || n.world.sto == nil || n.world.sto.count(k.owner, int(c.store)) < n.installedOf(&n.world.agents[k.owner], c.store) {
			return fmt.Errorf("infrastructure: owner %d has %d installed facilities of class %d and too few stores", k.owner, count, k.class)
		}
	}
	return nil
}

// fillFacilityState sets the Facilities group of the state of an agent (spec-infrastructure.md
// 11.4): its facilities in all regions in ascending ID. It is nil-safe: with the module off
// the group is empty.
func (w *World) fillFacilityState(st *AgentState, a *agent) {
	if w.infra == nil {
		return
	}
	st.Facilities = w.infra.ownedBy(a.id)
}

// remove deletes the record at position i and unindexes it.
func (n *infrastructure) remove(i int) {
	f := n.fac[i]
	n.unindex(f.owner, f.id)
	if f.installed {
		n.uninstall(f.owner, f.class)
	}
	n.fac = slices.Delete(n.fac, i, i+1)
}

// unindex takes the ID from the list of its owner, dropping the list when it is empty.
func (n *infrastructure) unindex(owner AgentID, id FacilityID) {
	list := n.byOwner[owner]
	if i, ok := slices.BinarySearch(list, id); ok {
		list = slices.Delete(list, i, i+1)
	}
	if len(list) == 0 {
		delete(n.byOwner, owner)
		return
	}
	n.byOwner[owner] = list
}

// give moves the facility at position i to the owner, keeping both lists in ascending ID.
func (n *infrastructure) give(i int, to AgentID) {
	f := &n.fac[i]
	n.unindex(f.owner, f.id)
	list := n.byOwner[to]
	j, _ := slices.BinarySearch(list, f.id)
	n.byOwner[to] = slices.Insert(list, j, f.id)
	f.owner = to
}

// install records that the store of the facility at position i has been given.
func (n *infrastructure) install(i int) {
	f := &n.fac[i]
	f.installed = true
	n.installed[ownerClass{f.owner, f.class}]++
}

// uninstall drops one installed store facility of the owner and class from the count.
func (n *infrastructure) uninstall(owner AgentID, class FacilityClassID) {
	k := ownerClass{owner, class}
	if n.installed[k] <= 1 {
		delete(n.installed, k)
		return
	}
	n.installed[k]--
}

// held is the number of facilities an owner holds.
func (n *infrastructure) held(owner AgentID) int { return len(n.byOwner[owner]) }

// saveFacilities is tag 27 (spec-infrastructure.md 15.1): the ID the next creation takes,
// the count and for each facility in ascending ID its ID, class index, region, target
// (NoRegion written as 0 and a region r as r + 1), owner, condition, built tick and
// installed flag. The effect tables and the lists by owner are derived and never saved.
func (w *World) saveFacilities(cw *codec.Writer) error {
	n := w.infra
	cw.U(uint64(n.next))
	cw.Count(len(n.fac))
	for i := range n.fac {
		f := &n.fac[i]
		cw.U(uint64(f.id))
		cw.U(uint64(f.class))
		cw.U(uint64(f.region))
		cw.U(uint64(f.to + 1))
		cw.U(uint64(f.owner))
		cw.U(uint64(f.condition))
		cw.U(uint64(f.built))
		cw.B(f.installed)
	}
	return nil
}

// loadFacilities reads tag 27 into the state a world with the module has built, which
// holds no facility, and applies the checks of spec-infrastructure.md 15.3 in the order of
// spec-checkpoint.md 6.2: the IDs are strictly ascending and below the next ID, a class
// index is below the count of classes, and each record passes checkRecord (its condition,
// region, target, owner and installed flag). The counts and the installed stores are
// checked by the audit that every restore ends with. The effect tables are rebuilt from the
// records, after the sections they read (agents, resources and transport) are in.
func (w *World) loadFacilities(rs *restoreCtx, r *codec.Reader) error {
	n := w.infra
	next := r.Bounded("next facility ID", math.MaxInt64)
	count := r.Count(8)
	if r.Err() != nil {
		return nil
	}
	if next < 1 || count > MaxFacilities {
		r.Failf("next facility ID %d and %d facilities, want a next ID of at least 1 and at most %d facilities", next, count, MaxFacilities)
		return nil
	}
	n.next = FacilityID(next)
	var prev uint64
	for k := range count {
		f := facility{id: FacilityID(r.Strict(prev, k > 0))}
		prev = uint64(f.id)
		f.class = FacilityClassID(r.Index(uint64(len(n.cls))))
		f.region = RegionID(r.Index(uint64(n.regions)))
		f.to = RegionID(r.Index(uint64(n.regions)+1)) - 1
		f.owner = AgentID(r.Bounded("facility owner", math.MaxInt32))
		f.condition = r.Bounded("facility condition", MaxDurability)
		f.built = r.Bounded("facility built tick", rs.tick)
		f.installed = r.B()
		if r.Err() != nil {
			return nil
		}
		if f.id >= n.next {
			return fmt.Errorf("%w: facility %d is not below the next ID %d", ErrCheckpointCorrupt, f.id, n.next)
		}
		if err := n.checkRecord(&f); err != nil {
			return fmt.Errorf("%w: %w", ErrCheckpointCorrupt, err)
		}
		n.adopt(f)
	}
	n.rebuild()
	return nil
}

// adopt appends a restored record, which is above every record before it, and indexes it.
func (n *infrastructure) adopt(f facility) {
	n.fac = append(n.fac, f)
	n.byOwner[f.owner] = append(n.byOwner[f.owner], f.id)
	if f.installed {
		n.installed[ownerClass{f.owner, f.class}]++
	}
}
