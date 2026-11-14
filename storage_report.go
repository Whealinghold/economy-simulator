package economysim

import (
	"errors"
	"fmt"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/room"
)

// This file is the reporting part of the storage module (spec-storage.md 14, 4.4):
// the StoreChanged event, the rows of the snapshot and of the agent query, and the
// audit of the pledge. It is a continuation of storage.go (architecture.md 2.6).

// StoreCount is the stores of one class an agent owns.
type StoreCount struct {
	// Class is the store class.
	Class StoreClassID
	// Count is the number of stores of the class, at least 1.
	Count int64
}

// RoomState is the room group of AgentState (spec-storage.md 14.5): the room, the
// load, the pledge and the free room of the agent's account, each a pair, and its
// stores by class in ascending class order. It is the zero value for an agent that
// is not limited and with the module off.
type RoomState struct {
	// Limited says the account has a room.
	Limited bool
	// Room, Load, Pledge and Free are pairs (mass, bulk). Free is the room less the
	// load and the pledge, read as 0 where it would be negative after a cut of the
	// room (spec-storage.md 3.4.4).
	Room, Load, Pledge, Free Capacity
	// Stores are the agent's stores by class.
	Stores []StoreCount
}

// StoreRow is the stores of one class in one region (spec-storage.md 14.5): a row for
// each (region, class) with at least one store, by region then class.
type StoreRow struct {
	// Region is the region of the owners' accounts and Class the store class.
	Region RegionID
	Class  StoreClassID
	// Stores is the number of stores, not of agents or households.
	Stores int64
}

// storeEvent builds the StoreChanged of agent a for class c: the stores of the
// class and the room and load of the account after the change.
func (w *World) storeEvent(a *agent, region RegionID, c int, kind Kind, cause Cause) StoreChanged {
	ev := StoreChanged{
		Tick: w.Tick(), Agent: a.id, Region: region, Role: a.role, Class: StoreClassID(c),
		Kind: kind, Cause: cause, Stores: int64(w.sto.count(a.id, c)),
	}
	if rm, _, ok := w.sto.roomState(w.ledger, a); ok {
		ev.Room, ev.Load = Capacity{Mass: rm.Mass, Bulk: rm.Bulk}, Capacity{Mass: rm.LoadMass, Bulk: rm.LoadBulk}
	}
	return ev
}

// emitStore delivers the StoreChanged of a call, when an observer was supplied. It is
// built after the change is complete (observability-and-logging.md 4.4).
func (w *World) emitStore(a *agent, c int, kind Kind, cause Cause) {
	if w.obs.supplied {
		w.obs.deliver(w.storeEvent(a, w.regionOf(a), c, kind, cause))
	}
}

// deliverEnded delivers the StoreChanged events of the stores that the last role
// change ended, in class order, after its RoleChanged (spec-storage.md 3.6.1a, 11.3).
func (w *World) deliverEnded() {
	for _, e := range w.sto.ended {
		w.obs.deliver(e)
	}
	w.sto.ended = w.sto.ended[:0]
}

// addRoomRow adds agent a to the room figures of its role row (spec-storage.md
// 14.5): the households of the limited agents, and their room and load, with the
// weight folded into the room. A sum that does not fit saturates.
func (w *World) addRoomRow(row *RoleRow, a *agent) {
	rm, _, ok := w.sto.roomState(w.ledger, a)
	if !ok {
		return
	}
	row.LimitedHouseholds = satAdd(row.LimitedHouseholds, a.weight)
	row.RoomMass, row.RoomBulk = satAdd(row.RoomMass, rm.Mass), satAdd(row.RoomBulk, rm.Bulk)
	row.LoadMass, row.LoadBulk = satAdd(row.LoadMass, rm.LoadMass), satAdd(row.LoadBulk, rm.LoadBulk)
}

// storeRows appends the StoreRows of region g, by class, from the agents of the
// region's view.
func (w *World) storeRows(rows []StoreRow, g RegionID, v *RegionView) []StoreRow {
	s := w.sto
	n := make([]int64, len(s.classes))
	for i := range v.Agents {
		for c := range n {
			n[c] += int64(s.count(v.Agents[i].Agent, c)) // at most MaxStoresPerAgent each
		}
	}
	for c, k := range n {
		if k > 0 {
			rows = append(rows, StoreRow{Region: g, Class: StoreClassID(c), Stores: k})
		}
	}
	return rows
}

// fillRoomState sets the room group of the agent query (spec-storage.md 14.5).
func (w *World) fillRoomState(st *AgentState, a *agent) {
	s := w.sto
	rm, free, ok := s.roomState(w.ledger, a)
	if !ok {
		return
	}
	r := &st.Room
	r.Limited = true
	r.Room, r.Load = Capacity{Mass: rm.Mass, Bulk: rm.Bulk}, Capacity{Mass: rm.LoadMass, Bulk: rm.LoadBulk}
	r.Pledge, r.Free = Capacity{Mass: rm.PledgeMass, Bulk: rm.PledgeBulk}, Capacity{Mass: free.Mass, Bulk: free.Bulk}
	for c := range s.classes {
		if n := s.count(a.id, c); n > 0 {
			r.Stores = append(r.Stores, StoreCount{Class: StoreClassID(c), Count: int64(n)})
		}
	}
}

// auditStorage is the storage part of the audit (spec-storage.md 4.4, FR-STO-040),
// run after the ledger's own identities of the room (the load is the sum of the lots
// and the load plus the reserved room is within the room): for every household, in
// ID order, a limited agent has a room and an unlimited one has none and owns no
// store, the reserved room is 0 (the audit runs at a tick boundary), the pledge is
// what the job and the cargo in flight imply, and the cargo the module tracked is the
// cargo of the shipments in flight. It names the first that fails.
func (w *World) auditStorage() error {
	s := w.sto
	jobs, err := s.pledgedByJobs(w)
	if err != nil {
		return err
	}
	for i := w.firstHousehold; i < len(w.agents); i++ {
		a := &w.agents[i]
		if a.weight < 1 {
			continue
		}
		if err := w.auditRoom(a, jobs); err != nil {
			return fmt.Errorf("storage: agent %d: %w", a.id, err)
		}
	}
	return s.auditInbound(w)
}

// auditRoom is auditStorage for one agent.
func (w *World) auditRoom(a *agent, jobs []room.Pair) error {
	s := w.sto
	rm, err := w.ledger.RoomOf(a.account)
	if err != nil {
		return err
	}
	if rm.Limited != s.limited(a) {
		return fmt.Errorf("limited is %t, its role says %t", rm.Limited, s.limited(a))
	}
	if !rm.Limited {
		if s.owned(a.id) > 0 {
			return fmt.Errorf("an unlimited account owns %d stores", s.owned(a.id))
		}
		return nil
	}
	if rm.ReservedMass != 0 || rm.ReservedBulk != 0 {
		return fmt.Errorf("reserved room (%d, %d) at a boundary", rm.ReservedMass, rm.ReservedBulk)
	}
	if err := s.auditPledge(rm, a, jobs); err != nil {
		return err
	}
	if r := s.roomFor(a); rm.Mass != r.Mass || rm.Bulk != r.Bulk {
		return fmt.Errorf("room (%d, %d), its role, weight and stores give (%d, %d)", rm.Mass, rm.Bulk, r.Mass, r.Bulk)
	}
	return nil
}

// auditPledge checks that the pledge of a limited account is what its job and its
// cargo in flight imply (spec-storage.md 4.4 identity 4).
func (s *storage) auditPledge(rm ledger.Room, a *agent, jobs []room.Pair) error {
	want, okp := addPair(jobs[a.id], s.inbound[a.id])
	if !okp {
		return errors.New("the pledge its job and cargo imply does not fit int64")
	}
	if rm.PledgeMass != want.Mass || rm.PledgeBulk != want.Bulk {
		return fmt.Errorf("pledge (%d, %d), its job and cargo imply (%d, %d)", rm.PledgeMass, rm.PledgeBulk, want.Mass, want.Bulk)
	}
	return nil
}

// auditInbound checks that the inbound load the module tracks for each agent is the
// load of the lines of the shipments in flight that name it as consignee.
func (s *storage) auditInbound(w *World) error {
	if w.transport == nil {
		return nil
	}
	want, err := s.cargoInFlight(w)
	if err != nil {
		return err
	}
	for id := range s.inbound {
		if w.liveAgent(AgentID(id)) && s.inbound[id] != want[id] {
			return fmt.Errorf("storage: agent %d expects cargo of load %v, the shipments in flight carry %v", id, s.inbound[id], want[id])
		}
	}
	return nil
}

// cargoInFlight is, per agent ID, the load of the lines of the shipments in flight
// that name the agent as consignee, derived from the transport module's table.
func (s *storage) cargoInFlight(w *World) ([]room.Pair, error) {
	want := make([]room.Pair, len(s.inbound))
	for i := range w.transport.ships {
		sh := &w.transport.ships[i]
		if int(sh.consignee) >= len(want) || len(sh.lines) == 0 || !w.liveAgent(sh.consignee) {
			continue
		}
		load, ok := s.linesLoad(sh.lines)
		sum, ok2 := addPair(want[sh.consignee], load)
		if !ok || !ok2 {
			return nil, fmt.Errorf("storage: the cargo in flight to agent %d does not fit int64", sh.consignee)
		}
		want[sh.consignee] = sum
	}
	return want, nil
}

// pledgedByJobs is, per principal, the load of the pending output of the jobs in
// progress that name it, scanned from the agents' jobs (the audit's check of the
// pledge that syncJob keeps).
func (s *storage) pledgedByJobs(w *World) ([]room.Pair, error) {
	out := make([]room.Pair, len(w.agents))
	for i := w.firstHousehold; i < len(w.agents); i++ {
		a := &w.agents[i]
		if a.weight < 1 || a.job.executions <= 0 {
			continue
		}
		load, ok := s.pendingLoad(&a.job)
		sum, ok2 := addPair(out[a.job.principal], load)
		if !ok || !ok2 {
			return nil, fmt.Errorf("storage: the pledge of the job of agent %d does not fit int64", a.id)
		}
		out[a.job.principal] = sum
	}
	return out, nil
}
