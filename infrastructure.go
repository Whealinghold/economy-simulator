package economysim

import (
	"slices"

	fcy "github.com/Whealinghold/economy-sim/internal/facility"
)

// This file is the infrastructure module (spec-infrastructure.md; architecture.md
// 2.10.1): its state (the facility records in ascending ID, the next ID and the
// indexes that find a facility by its owner), the construction of the initial
// facilities (step 8b of spec-world.md 5.1.2) and the effect fractions. The records'
// bookkeeping is in infrastructure_state.go, the effect tables in
// infrastructure_step.go, the calls and queries in infrastructure_calls.go, the events,
// logs and snapshot rows in infrastructure_report.go and the validator in
// infrastructure_validate.go. Outside these files the module appears only in the
// catalogue files (config.go, hash.go, bounds.go, defaults.go, errors.go, event.go,
// validate.go) and at the named hooks of ADR 0010: the field of World, the construction
// call, the step of a removal in removeAgent, the audit, the section row of the
// checkpoint, the snapshot rows, the agent's state and the seam of storage.

// FacilityState is one facility at the tick boundary (spec-infrastructure.md 3.2.1,
// 11.4): plain data with no pointer.
type FacilityState struct {
	ID    FacilityID
	Class FacilityClassID
	// Region is where the facility stands, fixed at creation. To is the region at the
	// other end of a route that has one and NoRegion for every other facility.
	Region RegionID
	To     RegionID
	// Owner is the agent that holds it: a household or external agent, or the estate
	// pseudo-agent of Region (whose ID is the RegionID).
	Owner AgentID
	// Condition is the wear units left, 1 to the class's Durability.
	Condition int64
	// Built is the tick of creation.
	Built int64
	// Installed says, for a store facility, that its store has been given to the owner.
	Installed bool
	// Effect is eff (basis points) for a route or ground facility and cap (executions a
	// tick) for a site, at the current condition; 0 for a store.
	Effect int64
}

// facility is one record. The class parameters are read from infrastructure.cls.
type facility struct {
	id        FacilityID
	class     FacilityClassID
	region    RegionID
	to        RegionID
	owner     AgentID
	condition int64
	built     int64
	installed bool
}

// facClass is the derived form of a FacilityClass: the kind and numbers, and the
// configured references resolved to indexes (-1 for a field the kind does not use).
type facClass struct {
	kind                                           string
	durability, decay, wearPerUse, capacity, boost int64
	modes                                          []int
	resource                                       int
	store                                          StoreClassID
	flag                                           int
}

// ownerClass keys the installed store facilities of an owner by class.
type ownerClass struct {
	owner AgentID
	class FacilityClassID
}

// siteKey keys the capacity table of the sites (spec-infrastructure.md 6.4.1).
type siteKey struct {
	owner  AgentID
	region RegionID
	class  FacilityClassID
}

// infrastructure is the state of the module, nil when it is off. Only fac and next are
// saved; everything else is derived from them and the configuration (6.4, 15.2).
type infrastructure struct {
	world *World
	cls   []facClass
	// fac are the facilities in ascending ID and next the ID the next creation takes.
	// Saved.
	fac  []facility
	next FacilityID
	// byOwner lists the IDs of each owner's facilities in ascending order, and
	// installed counts the installed store facilities of an owner and class. They are
	// lookup tables only: nothing that affects results iterates them (the audit reads
	// their keys in sorted order).
	byOwner   map[AgentID][]FacilityID
	installed map[ownerClass]int
	// edge is the set of directed edges of the configuration, for the route classes.
	edge map[[2]RegionID]bool
	// The effect tables (spec-infrastructure.md 6.4.1), rebuilt from fac by rebuild:
	// the capacity of the sites by (owner, region, class); the gain of the routes by
	// directed pair and mode, at (from * regions + to) * modes + mode; and the gain of
	// the ground by endowment, at region * resources + resource. Never saved.
	site                      map[siteKey]int64
	route, ground             []int64
	regions, modes, resources int
	// limits are the per-tick counts of FacilityLimited by (region, class), at
	// region * classes + class, kept only under an observer and cleared at P8.6; never
	// saved (spec-infrastructure.md 14.1).
	limits []facilityLimit
}

// facilityLimit is the counts of one (region, class) cell for FacilityLimited.
type facilityLimit struct{ served, cut, refused, voided, pending int64 }

// newInfrastructure builds the state from a canonical configuration that passed
// Validate, with no facility, or returns nil when the module is off. The initial
// facilities are made by found once the agents exist.
func newInfrastructure(c *Config, set *settlements) *infrastructure {
	m := c.Modules.Infrastructure
	if m == nil {
		return nil
	}
	n := &infrastructure{
		cls: make([]facClass, len(m.Classes)), next: 1, regions: len(c.Regions),
		byOwner: map[AgentID][]FacilityID{}, installed: map[ownerClass]int{}, site: map[siteKey]int64{},
		limits: make([]facilityLimit, len(c.Regions)*len(m.Classes)),
	}
	if c.Modules.Transport != nil {
		n.modes = len(c.Modules.Transport.Modes)
	}
	if c.Modules.Resources != nil {
		n.resources = len(c.Modules.Resources.Resources)
	}
	for i := range m.Classes {
		n.cls[i] = n.classOf(c, set, &m.Classes[i])
		if n.cls[i].kind == kindRoute && n.route == nil {
			n.route = make([]int64, n.regions*n.regions*max(n.modes, 1))
			n.edge = n.edgeSet(c)
		}
		if n.cls[i].kind == kindGround && n.ground == nil {
			n.ground = make([]int64, n.regions*n.resources)
		}
	}
	return n
}

// classOf resolves one class to its derived form.
func (n *infrastructure) classOf(c *Config, set *settlements, cl *FacilityClass) facClass {
	fc := facClass{kind: cl.Kind, durability: cl.Durability, decay: cl.Decay, wearPerUse: cl.WearPerUse,
		capacity: cl.Capacity, boost: cl.BoostBP, resource: -1, store: -1, flag: -1}
	if c.Modules.Transport != nil {
		for _, id := range cl.Modes {
			fc.modes = append(fc.modes, slices.IndexFunc(c.Modules.Transport.Modes, func(m Mode) bool { return m.ID == id }))
		}
	}
	if cl.Resource != "" && c.Modules.Resources != nil {
		fc.resource = slices.IndexFunc(c.Modules.Resources.Resources, func(r Resource) bool { return r.ID == cl.Resource })
	}
	if cl.Store != "" && c.Modules.Storage != nil {
		fc.store = StoreClassID(slices.IndexFunc(c.Modules.Storage.Classes, func(k StoreClass) bool { return k.ID == cl.Store }))
	}
	if cl.Needs != "" && set != nil {
		fc.flag = nameIndex(set.flags, cl.Needs)
	}
	return fc
}

// edgeSet is the set of directed edges of the configuration by region index.
func (n *infrastructure) edgeSet(c *Config) map[[2]RegionID]bool {
	idx := make(map[string]RegionID, len(c.Regions)) // lookup only
	for i := range c.Regions {
		idx[c.Regions[i].ID] = RegionID(i)
	}
	out := make(map[[2]RegionID]bool, len(c.Edges))
	for _, e := range c.Edges {
		out[[2]RegionID{idx[e.From], idx[e.To]}] = true
	}
	return out
}

// joined reports whether an edge joins the two regions in either direction.
func (n *infrastructure) joined(a, b RegionID) bool {
	return n.edge[[2]RegionID{a, b}] || n.edge[[2]RegionID{b, a}]
}

// eff is the effect of a facility at its condition: eff of a route or ground facility,
// cap of a site, 0 for a store (spec-infrastructure.md 3.3.2). The bounds make the
// product fit, so a failure is a violated invariant.
func (n *infrastructure) eff(f *facility) int64 {
	c := &n.cls[f.class]
	var v int64
	var ok bool
	switch c.kind {
	case kindSite:
		v, ok = fcy.Cap(c.capacity, f.condition, c.durability)
	case kindRoute, kindGround:
		v, ok = fcy.Eff(c.boost, f.condition, c.durability)
	default:
		return 0
	}
	if !ok {
		invariant("the effect of a facility does not fit after validation")
	}
	return v
}

// index is the position of the facility in fac, or -1.
func (n *infrastructure) index(id FacilityID) int {
	i, ok := slices.BinarySearchFunc(n.fac, id, func(f facility, id FacilityID) int {
		switch {
		case f.id < id:
			return -1
		case f.id > id:
			return 1
		}
		return 0
	})
	if !ok {
		return -1
	}
	return i
}

// state is the public form of a record.
func (n *infrastructure) state(f *facility) FacilityState {
	return FacilityState{ID: f.id, Class: f.class, Region: f.region, To: f.to, Owner: f.owner,
		Condition: f.condition, Built: f.built, Installed: f.installed, Effect: n.eff(f)}
}

// found makes the initial facilities (spec-infrastructure.md 3.4.2, 3.4.3), in the order
// of the rows: the facilities of a row are made in turn for the agents of the group
// (region, role) in ascending agent ID, or for the estate of the region when the row has
// no role. It takes no random number and emits no event: announce does, after the
// AgentSpawned events. The configuration passed Validate, so a failure here is a
// violated invariant. It is nil-safe.
func (n *infrastructure) found(w *World) {
	if n == nil {
		return
	}
	n.world = w
	m := w.cfg.Modules.Infrastructure
	for i := range m.Initial {
		r := &m.Initial[i]
		class := FacilityClassID(slices.IndexFunc(m.Classes, func(c FacilityClass) bool { return c.ID == r.Class }))
		region := RegionID(slices.IndexFunc(w.cfg.Regions, func(g Region) bool { return g.ID == r.Region }))
		to := NoRegion
		if r.Toward != "" {
			to = RegionID(slices.IndexFunc(w.cfg.Regions, func(g Region) bool { return g.ID == r.Toward }))
		}
		owners := []AgentID{AgentID(region)}
		if r.Role != "" {
			owners = n.holders(w, region, r.Role)
		}
		if len(owners) == 0 {
			invariant("an initial facility row has no holder after validation")
		}
		for k := int64(0); k < r.Count; k++ {
			f := n.create(class, region, to, owners[int(k)%len(owners)], n.cls[class].durability, 0)
			if n.cls[class].kind == kindStore {
				f.installed = w.installAtConstruction(w.agentOf(f.owner), n.cls[class].store)
				if f.installed {
					n.installed[ownerClass{f.owner, class}]++
				}
			}
		}
	}
	n.rebuild()
}

// holders are the agents of the role in the region, in ascending agent ID.
func (n *infrastructure) holders(w *World, region RegionID, role string) []AgentID {
	ri := slices.IndexFunc(w.cfg.Roles, func(r Role) bool { return r.ID == role })
	var out []AgentID
	for k := w.firstHousehold; k < len(w.agents); k++ {
		if a := &w.agents[k]; int(a.role) == ri && a.weight >= 1 && w.regionOf(a) == region {
			out = append(out, a.id)
		}
	}
	return out
}

// agentOf is the live agent with the ID; the caller has checked it is one.
func (w *World) agentOf(id AgentID) *agent { return &w.agents[id] }

// installAtConstruction gives the owner the store of a store facility with no event
// (the stores of a role row have none either: AgentSpawned reports them, and the
// FacilityChanged of the facility reports this one). It reports whether the store was
// given: a closed gate of the store class or a full owner leaves the facility pending,
// to be installed at the facility step (spec-infrastructure.md 5.4.2).
func (w *World) installAtConstruction(a *agent, class StoreClassID) bool {
	s := w.sto
	if s == nil || !s.limited(a) || (s.gate != nil && !s.gate(w.regionOf(a), int(class))) || s.owned(a.id) >= MaxStoresPerAgent {
		return false
	}
	s.give(a.id, int(class))
	if err := s.apply(w.ledger, a); err != nil {
		s.broke(a.id, "a store facility lowered the room of an account: "+err.Error())
	}
	return true
}
