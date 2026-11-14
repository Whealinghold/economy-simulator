package economysim

import (
	"fmt"
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/room"
)

// This file is the storage module (spec-storage.md; architecture.md 2.10.1): what
// the module reads of its configuration, its state (the stores of each agent, kept
// in a slice indexed by agent ID, N-244), the room of an account, the two calls
// AddStore and RemoveStore and the operations they share with the infrastructure
// module. The validator and the warnings are in storage_validate.go, the
// commitments of room at a bid, a start and a dispatch in storage_commit.go and the
// events, the snapshot rows and the audit in storage_report.go. Outside this file
// and its continuations the module appears only in the catalogue files (config.go,
// hash.go, bounds.go, defaults.go, errors.go, event.go, validate.go), in value.go
// (StoreClassID), in role.go (the resolved base room of a role) and at the named
// hooks of ADR 0010.

// hasStoreName is the name of the region_bound obligation of an owner of a store
// (spec-storage.md 9.2); it is the reason of the refusal of a self-carried trip.
const hasStoreName = string(ReasonHasStore)

// storage is the state of the module, nil when it is off. The room of an account is
// not state (spec-storage.md 3.3.4): it is a function of the role, the weight and
// the stores, and the ledger holds it for the operations that read it. The stores are
// the only saved state (checkpoint_*.go); the inbound load is derived from the
// shipments in flight and the counters are scratch.
type storage struct {
	cfg     *Config
	roles   *roleTable
	recipes *recipeTable
	// classes is the capacity of one store of each class, by StoreClassID, and sizes
	// the size of one unit of each commodity (spec-goods.md 3.1).
	classes, sizes []room.Pair
	// given is, per role, the classes its row gives at creation, in class order
	// with repeats (spec-storage.md 3.6.1).
	given [][]int
	// stores is, per agent ID, the count of stores of each class: nil for an agent
	// that owns none and for an agent that is not a household.
	stores [][]uint8
	// inbound is, per agent ID, the load of the lines of the shipments in flight that
	// name the agent as consignee (spec-storage.md 3.4.2); derived, not saved.
	inbound []room.Pair
	// counts is the per-tick counters of RoomLimited per (region, role), cleared at
	// the end of every step (storage_commit.go).
	counts []roomCount
	// world is the world the state belongs to, for the agents a job's principal and
	// residue owner name (a trainee's job is its trainer's).
	world *World
	// jobLoad is, per executor, the load its job pledged on its principal; jobPrin
	// is that principal; held is, per principal, the sum of the loads pledged on it.
	// They are derived from the jobs and never saved.
	jobLoad []room.Pair
	jobPrin []AgentID
	held    []room.Pair
	// ended holds the stores that a role change ended until the world has emitted
	// the RoleChanged they follow (spec-storage.md 3.6.1a); empty without an observer.
	ended []StoreChanged
	// deferred says construction has not yet set the rooms: the goods of the warm
	// start are placed first and the rooms are set once they are, so that the
	// transient grant of a warm job's inputs meets no room (construct_warm.go).
	deferred bool
	// gate says whether a class may be given in a region (spec-storage.md 3.2.3):
	// nil is open, which it is until the settlements module sets it (D8).
	gate func(region RegionID, class int) bool
	// facilityCount is the installed store facilities of a class an agent has (D9);
	// nil is none, until the infrastructure module sets it.
	facilityCount func(a *agent, class StoreClassID) int
	// scratch for the stores of one account and the lists of a recipe.
	caps []room.Pair
	in   []room.Item
	out  []room.Output
}

// newStorage builds the state from a canonical configuration that passed Validate,
// or nil when the module is off.
func newStorage(c *Config, rt *roleTable, rc *recipeTable) *storage {
	m := c.Modules.Storage
	if m == nil {
		return nil
	}
	s := &storage{
		cfg: c, roles: rt, recipes: rc, deferred: true,
		classes: make([]room.Pair, len(m.Classes)),
		sizes:   make([]room.Pair, len(c.Commodities)),
		given:   make([][]int, len(c.Roles)),
		counts:  make([]roomCount, len(c.Regions)*len(c.Roles)),
	}
	for i, cl := range m.Classes {
		s.classes[i] = room.Pair{Mass: cl.Mass, Bulk: cl.Bulk}
	}
	for i := range c.Commodities {
		s.sizes[i] = room.Pair{Mass: c.Commodities[i].Mass, Bulk: c.Commodities[i].Bulk}
	}
	for _, rr := range m.Roles {
		ri := slices.IndexFunc(c.Roles, func(r Role) bool { return r.ID == rr.Role })
		for _, id := range rr.Stores {
			s.given[ri] = append(s.given[ri], slices.IndexFunc(m.Classes, func(k StoreClass) bool { return k.ID == id }))
		}
	}
	return s
}

// ledgerSizes is the table of unit sizes the ledger is given when the module is on
// (spec-storage.md 4.1.1), one per commodity in index order; nil while it is off, so
// that every commodity takes no room.
func ledgerSizes(c *Config) []ledger.Size {
	if c.Modules.Storage == nil {
		return nil
	}
	sizes := make([]ledger.Size, len(c.Commodities))
	for i := range c.Commodities {
		sizes[i] = ledger.Size{Mass: c.Commodities[i].Mass, Bulk: c.Commodities[i].Bulk}
	}
	return sizes
}

// baseOf is the resolved base room per household of a role (spec-storage.md 3.3.1):
// the Base of the role's row when it has one, else the module's Base; nil is an
// unlimited role. The second result is the index of the row, or -1 when the
// module's Base is the answer. It is nil, -1 for a nil module (off: no role is
// limited).
func (m *Storage) baseOf(role string) (*Capacity, int) {
	if m == nil {
		return nil, -1
	}
	for i := range m.Roles {
		if m.Roles[i].Role == role {
			return m.Roles[i].Base, i
		}
	}
	return m.Base, -1
}

// classSet is the IDs of the classes of the module, a membership test only.
func (m *Storage) classSet() map[string]bool {
	set := make(map[string]bool, len(m.Classes))
	for _, c := range m.Classes {
		set[c.ID] = true
	}
	return set
}

// roomOf is a copy of the resolved base room of a role, nil for an unlimited role
// and for the module off: the role table holds a value of its own, so a change of
// the consumer's Config after construction moves no room.
func (m *Storage) roomOf(role string) *Capacity {
	b, _ := m.baseOf(role)
	if b == nil {
		return nil
	}
	room := *b
	return &room
}

// limited reports whether the agent's role has a base room (spec-storage.md 3.3.1,
// FR-STO-003): the agent is limited if and only if it does.
func (s *storage) limited(a *agent) bool { return s.roles.rows[a.role].room != nil }

// cellOf is the index of the (region, role) cell of agent a in region, the index of
// the per-tick counters and of the cells of the world.
func (w *World) cellOf(region RegionID, a *agent) int {
	return int(region)*len(w.cfg.Roles) + int(a.role)
}

// grow makes the per-agent slices cover the agent ID.
func (s *storage) grow(id AgentID) {
	for len(s.stores) <= int(id) {
		s.stores = append(s.stores, nil)
		s.inbound = append(s.inbound, room.Pair{})
		s.jobLoad = append(s.jobLoad, room.Pair{})
		s.jobPrin = append(s.jobPrin, AgentID(len(s.jobPrin)))
		s.held = append(s.held, room.Pair{})
	}
}

// owned is the number of stores the agent owns, of every class.
func (s *storage) owned(id AgentID) int {
	if int(id) >= len(s.stores) {
		return 0
	}
	n := 0
	for _, c := range s.stores[id] {
		n += int(c)
	}
	return n
}

// count is the number of stores of one class the agent owns.
func (s *storage) count(id AgentID, class int) int {
	if int(id) >= len(s.stores) || s.stores[id] == nil {
		return 0
	}
	return int(s.stores[id][class])
}

// give adds one store of the class to the agent's count, without a room change.
func (s *storage) give(id AgentID, class int) {
	if s.stores[id] == nil {
		s.stores[id] = make([]uint8, len(s.classes))
	}
	s.stores[id][class]++
}

// take takes one store of the class from the agent's count. The count must be
// above 0.
func (s *storage) take(id AgentID, class int) {
	s.stores[id][class]--
}

// roomFor is the room of a limited agent of the weight with the stores it owns:
// weight times the base room per household plus the capacity of each store, per
// dimension (spec-storage.md 3.3.2, FR-STO-004), with checked arithmetic. A room
// that does not fit int64 is above any load the ledger can hold and is saturated.
func (s *storage) roomFor(a *agent) room.Pair { return s.roomIn(a, a.role) }

// roomIn is roomFor with the agent in another role: the role must have a base room.
func (s *storage) roomIn(a *agent, role RoleID) room.Pair {
	base := s.roles.rows[role].room
	s.caps = s.caps[:0]
	if int(a.id) < len(s.stores) {
		for c, n := range s.stores[a.id] {
			for range n {
				s.caps = append(s.caps, s.classes[c])
			}
		}
	}
	r, ok := room.Of(a.weight, room.Pair{Mass: base.Mass, Bulk: base.Bulk}, s.caps)
	if !ok {
		return room.Pair{Mass: math.MaxInt64, Bulk: math.MaxInt64}
	}
	return r
}

// apply sets the room of agent a in the ledger: the room of a limited agent, or
// none for an unlimited one. A limited account must hold what is in it, else the
// ledger refuses with ErrNoRoom and nothing changes.
func (s *storage) apply(l *ledger.Ledger, a *agent) error {
	if !s.limited(a) {
		return l.SetRoom(a.account, false, 0, 0)
	}
	r := s.roomFor(a)
	return l.SetRoom(a.account, true, r.Mass, r.Bulk)
}

// roomState returns the room, load, pledge and free room of an agent's account as
// the ledger holds them, and false for an account that is not limited.
func (s *storage) roomState(l *ledger.Ledger, a *agent) (rm ledger.Room, free room.Pair, ok bool) {
	rm, err := l.RoomOf(a.account)
	if err != nil {
		invariant("the room of an agent's account is unreadable: " + err.Error())
	}
	if !rm.Limited {
		return rm, room.Pair{}, false
	}
	m, b, err := l.FreeRoom(a.account)
	if err != nil {
		invariant("the free room of a limited account is unreadable: " + err.Error())
	}
	return rm, room.Pair{Mass: m, Bulk: b}, true
}

// joinedStorage is the step of the agent-lifecycle fan-in agentJoined for the
// module (spec-storage.md 3.6.1): the agent is given the stores of its role's row,
// in class order and with no event, and its account the room (the room waits for
// the end of construction while the warm start is applied). The gate of a class's
// Needs is the settlements module's: until it exists every class is open and no
// store is withheld (D8).
func (w *World) joinedStorage(a *agent) {
	s := w.sto
	s.grow(a.id)
	for _, c := range s.given[a.role] {
		s.give(a.id, c)
	}
	if s.deferred {
		return
	}
	if err := s.apply(w.ledger, a); err != nil {
		invariant("the room of a created agent is below its goods: " + err.Error())
	}
}

// roleChangedStorage is the step of agentRoleChanged for the module (spec-storage.md
// 3.6.1a, 5.5.2): the new role's room is set, and a role with no base room makes the
// account unlimited and ends the stores of the agent, in class order, each reported
// by a StoreChanged that the world emits after the RoleChanged (deliverEnded). The
// goods outside the new room have already gone to the estate (cutForRole).
func (w *World) roleChangedStorage(a *agent, region RegionID) {
	s := w.sto
	s.ended = s.ended[:0]
	if err := s.apply(w.ledger, a); err != nil {
		invariant("the room of an agent that changed role is below its goods: " + err.Error())
	}
	if !s.limited(a) {
		for c := range s.classes {
			for s.count(a.id, c) > 0 {
				s.take(a.id, c)
				if w.obs.supplied {
					s.ended = append(s.ended, w.storeEvent(a, region, c, KindRemoved, CauseRoleChanged))
				}
			}
		}
	}
	w.syncJob(a)
}

// dropStorage is the step of a removal for the module (spec-storage.md 3.5.1): the
// stores of the agent end with it, with no event (AgentBankrupted is the event of a
// removal).
func (w *World) dropStorage(a *agent) {
	s := w.sto
	if int(a.id) < len(s.stores) {
		s.stores[a.id], s.inbound[a.id] = nil, room.Pair{}
		s.jobLoad[a.id], s.held[a.id] = room.Pair{}, room.Pair{}
	}
}

// holds is the test of the region_bound obligation has_store (spec-storage.md 9.2):
// the agent owns at least one store. It has no instance key.
func (s *storage) holds(a *agent, _ []string) bool { return s.owned(a.id) > 0 }

// away reports whether the agent is away on a self-carried trip or on a migration
// journey (spec-storage.md 3.6.2, ErrAgentAway).
func (w *World) away(a *agent) bool {
	return (w.transport != nil && w.transport.isAway(a.id)) || w.onJourney(a)
}

// AddStore gives the agent one more store of the class (spec-storage.md 3.6.2,
// FR-STO-032). It acts between ticks and is atomic: a refused call changes nothing.
// It returns ErrInStep, ErrWorldFailed, ErrModuleOff, ErrUnknownAgent (an ID never
// issued or a pseudo-agent), ErrAgentRemoved, ErrUnknownStoreClass, ErrStoreLocked,
// ErrNotLimited (the agent's role has no base room), ErrAgentAway (the agent is away
// on a trip or a journey) or ErrOutOfBounds (it would own more than
// MaxStoresPerAgent). On success it emits StoreChanged with the cause consumer.
func (w *World) AddStore(a AgentID, class StoreClassID) error {
	if err := w.enter("AddStore", "storage", w.sto != nil); err != nil {
		return err
	}
	ag, err := w.queryAgent(a)
	if err != nil {
		return fmt.Errorf("AddStore: agent %d: %w", a, err)
	}
	if err := w.addStore(ag, class, CauseConsumer); err != nil {
		return fmt.Errorf("AddStore: agent %d, class %d: %w", a, class, err)
	}
	return nil
}

// RemoveStore takes one store of the class from the agent (spec-storage.md 3.6.2,
// FR-STO-033). It acts between ticks and is atomic. It returns ErrInStep,
// ErrWorldFailed, ErrModuleOff, ErrUnknownAgent, ErrAgentRemoved,
// ErrUnknownStoreClass, ErrOutOfBounds (the agent owns none of the class),
// ErrStoreInUse or ErrNoRoom (without the store the load plus the pledge, or the
// pledge plus the upkeep room, would exceed the room). It is allowed for an agent
// that is away. On success it emits StoreChanged with the cause consumer.
func (w *World) RemoveStore(a AgentID, class StoreClassID) error {
	if err := w.enter("RemoveStore", "storage", w.sto != nil); err != nil {
		return err
	}
	ag, err := w.queryAgent(a)
	if err != nil {
		return fmt.Errorf("RemoveStore: agent %d: %w", a, err)
	}
	if _, err := w.removeStore(ag, class, CauseConsumer, false); err != nil {
		return fmt.Errorf("RemoveStore: agent %d, class %d: %w", a, class, err)
	}
	return nil
}

// addStore is the operation behind AddStore and the infrastructure module's
// install (spec-storage.md 3.6.3): the checks of the table of 3.6.2 in its order,
// the store, the new room and the event. The away test is in the operation, so a
// facility completed for an agent that is away meets it too.
func (w *World) addStore(a *agent, class StoreClassID, cause Cause) error {
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
	case s.owned(a.id) >= MaxStoresPerAgent:
		return ErrOutOfBounds
	}
	s.give(a.id, int(class))
	if err := s.apply(w.ledger, a); err != nil {
		invariant("a store lowered the room of an account: " + err.Error())
	}
	w.emitStore(a, int(class), KindAdded, cause)
	return nil
}

// removeStore is the operation behind RemoveStore and the infrastructure module's
// ruin (spec-storage.md 3.6.3). The ordinary form is refused with ErrNoRoom where
// the room that is left cannot hold the load and the pledge; the forced form of a
// ruin never is: it cuts the room by the rule of 5.5.2.1, whole lots to the estate
// under estate_goods, and leaves the pledge as it is (5.5.3). It returns the number
// of lots moved.
func (w *World) removeStore(a *agent, class StoreClassID, cause Cause, forced bool) (lots int64, err error) {
	s := w.sto
	switch {
	case class < 0 || int(class) >= len(s.classes):
		return 0, ErrUnknownStoreClass
	case s.count(a.id, int(class)) == 0:
		return 0, ErrOutOfBounds
	case s.count(a.id, int(class)) <= s.facilities(a, class):
		return 0, ErrStoreInUse
	}
	s.take(a.id, int(class))
	after := s.roomFor(a)
	if !forced && !w.roomHolds(a, after) {
		s.give(a.id, int(class))
		return 0, ErrNoRoom
	}
	if forced {
		if lots, err = w.cutLoad(a, w.regionOf(a), after); err != nil {
			s.give(a.id, int(class))
			return 0, err
		}
	}
	if err := s.apply(w.ledger, a); err != nil {
		invariant("the room of an account after a removed store is below its load: " + err.Error())
	}
	w.emitStore(a, int(class), KindRemoved, cause)
	return lots, nil
}

// facilities is the number of installed store facilities of the class that the
// agent's stores must not fall below (spec-storage.md 3.6.2, ErrStoreInUse). It is 0
// until the infrastructure module exists (D9).
func (s *storage) facilities(a *agent, class StoreClassID) int {
	if s.facilityCount == nil {
		return 0
	}
	return s.facilityCount(a, class)
}

// roomHolds reports whether room after holds the load plus the pledge and the
// pledge plus the upkeep room of agent a, per dimension (spec-storage.md 3.6.2,
// ErrNoRoom): the test of an ordinary RemoveStore. All the sums are checked; a sum
// that does not fit is above any room.
func (w *World) roomHolds(a *agent, after room.Pair) bool {
	rm, _, ok := w.sto.roomState(w.ledger, a)
	if !ok {
		return true
	}
	up, upOK := w.sto.upkeepRoom(a, w.regionOf(a))
	lp, lpOK := addPair(room.Pair{Mass: rm.LoadMass, Bulk: rm.LoadBulk}, room.Pair{Mass: rm.PledgeMass, Bulk: rm.PledgeBulk})
	pu, puOK := addPair(room.Pair{Mass: rm.PledgeMass, Bulk: rm.PledgeBulk}, up)
	return upOK && lpOK && puOK && lp.Mass <= after.Mass && lp.Bulk <= after.Bulk && pu.Mass <= after.Mass && pu.Bulk <= after.Bulk
}

// addPair is the checked sum of two pairs; ok is false when a dimension overflows.
func addPair(a, b room.Pair) (room.Pair, bool) {
	m, okm := checked.Add(a.Mass, b.Mass)
	k, okk := checked.Add(a.Bulk, b.Bulk)
	return room.Pair{Mass: m, Bulk: k}, okm && okk
}

// upkeepRoom is the upkeep room of agent a in a region (spec-storage.md 3.4.5): the
// load of the units its agent will eat at the end of the tick, weight times the
// effective upkeep of one household times the size of each commodity, per dimension,
// 0 for a role with no upkeep. ok is false when a sum overflows int64.
func (s *storage) upkeepRoom(a *agent, region RegionID) (room.Pair, bool) {
	var out room.Pair
	for _, it := range s.roles.row(a.role).upkeepIn(region) {
		q, ok := checked.Mul(a.weight, int64(it.quantity))
		sz := s.sizes[it.commodity]
		m, okm := checked.Mul(q, sz.Mass)
		b, okb := checked.Mul(q, sz.Bulk)
		var oks, okt bool
		out.Mass, oks = checked.Add(out.Mass, m)
		out.Bulk, okt = checked.Add(out.Bulk, b)
		if !ok || !okm || !okb || !oks || !okt {
			return room.Pair{}, false
		}
	}
	return out, true
}

// storedLoad is the stored load of an agent's account: the part of its load that
// sits above the base room, up to the capacity of its stores (spec-storage.md 8.3.2,
// FR-STO-042), read by the risk module; 0 for an account that is not limited.
func (w *World) storedLoad(a *agent) room.Pair {
	s := w.sto
	rm, _, ok := s.roomState(w.ledger, a)
	if !ok {
		return room.Pair{}
	}
	var stores room.Pair
	for c, n := range s.stores[a.id] {
		m, okm := checked.Mul(s.classes[c].Mass, int64(n))
		b, okb := checked.Mul(s.classes[c].Bulk, int64(n))
		var oks bool
		stores, oks = addPair(stores, room.Pair{Mass: m, Bulk: b})
		if !okm || !okb || !oks {
			return room.Pair{}
		}
	}
	base := s.roles.rows[a.role].room
	return room.StoredLoad(stores, room.Pair{Mass: rm.LoadMass, Bulk: rm.LoadBulk}, room.Pair{Mass: base.Mass, Bulk: base.Bulk}, a.weight)
}

// storeClassNames is the string ID of each store class in StoreClassID order, nil
// while the module is off and while it has no class (the names table of
// spec-storage.md 14.1: an empty module renders as the module off).
func storeClassNames(c *Config) []string {
	m := c.Modules.Storage
	if m == nil || len(m.Classes) == 0 {
		return nil
	}
	names := make([]string, len(m.Classes))
	for i, cl := range m.Classes {
		names[i] = cl.ID
	}
	return names
}

// finishStorage ends the construction of the module (spec-storage.md 13.1,
// FR-STO-045): the warm start has placed its goods and jobs, and now every
// household's room is set and its job's pledge. A household whose goods, or whose
// goods plus its warm job's output, do not fit its room is ErrRoomShape, never the
// ledger's ErrNoRoom (room_below_start_stock for a warm group and room_below_warm_job).
func (w *World) finishStorage() error {
	s := w.sto
	s.deferred = false
	for i := w.firstHousehold; i < len(w.agents); i++ {
		a := &w.agents[i]
		if err := s.apply(w.ledger, a); err != nil {
			return fmt.Errorf("agent %d of role %s: the goods it starts with do not fit its room: %w: %w",
				a.id, w.cfg.Roles[a.role].ID, ErrRoomShape, err)
		}
		w.syncJob(a)
		if s.limited(a) && !w.roomHolds(a, s.roomFor(a)) {
			return fmt.Errorf("agent %d of role %s: its warm job's output and its goods do not fit its room: %w",
				a.id, w.cfg.Roles[a.role].ID, ErrRoomShape)
		}
	}
	return nil
}

// admitsStock is the room check of AddAgent (spec-storage.md 5.1.1, FR-STO-019): the
// goods of the new agent must fit the room it would have, the base room of its role
// and the stores of the role's row. It returns ErrNoRoom, and AddAgent creates
// nothing. An agent of an unlimited role has no room to test.
func (s *storage) admitsStock(n NewAgent, stock []itemRow) error {
	base := s.roles.rows[n.Role].room
	if base == nil {
		return nil
	}
	r := room.Pair{Mass: base.Mass, Bulk: base.Bulk}
	for _, class := range s.given[n.Role] {
		var ok bool
		if r, ok = addPair(r, s.classes[class]); !ok {
			return nil // a room above int64 holds any load
		}
	}
	var load room.Pair
	for _, it := range stock {
		sz := s.sizes[it.commodity]
		m, okm := checked.Mul(int64(it.quantity), sz.Mass)
		b, okb := checked.Mul(int64(it.quantity), sz.Bulk)
		var ok bool
		load, ok = addPair(load, room.Pair{Mass: m, Bulk: b})
		if !okm || !okb || !ok {
			return ErrNoRoom
		}
	}
	if load.Mass > r.Mass || load.Bulk > r.Bulk {
		return ErrNoRoom
	}
	return nil
}
