package economysim

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
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
	w.logStore(a, c, kind, cause)
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
// region's view; with the storage module off it returns rows unchanged.
func (w *World) storeRows(rows []StoreRow, g RegionID, v *RegionView) []StoreRow {
	s := w.sto
	if s == nil {
		return rows
	}
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

// pairText writes a pair of room as "(mass M, bulk B)": the audit says which dimension
// differs, which "{5 3}" does not.
func pairText(p room.Pair) string { return fmt.Sprintf("(mass %d, bulk %d)", p.Mass, p.Bulk) }

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
		if err := w.auditAgent(a, jobs); err != nil {
			return fmt.Errorf("storage: agent %d: %w", a.id, err)
		}
	}
	return s.auditInbound(w)
}

// auditAgent is auditStorage for one household slot: a live agent's books and room, a
// removed agent's books at rest.
func (w *World) auditAgent(a *agent, jobs []room.Pair) error {
	if a.weight < 1 {
		return w.sto.auditGone(a)
	}
	if err := w.sto.auditBooks(a, jobs); err != nil {
		return err
	}
	return w.auditRoom(a, jobs)
}

// auditBooks checks the module's own books of a live agent, a limited one or not (a
// trainer in a role with no room carries the pledge its trainees' jobs put on it until
// it changes role, and the ledger does not hold it): the sum pledged on the agent is
// that of the jobs that name it, and the load its own job pledged, and on whom, is that
// of its pending output now. A book that drifts would release too much or too little
// at the next change of the job.
func (s *storage) auditBooks(a *agent, jobs []room.Pair) error {
	if s.held[a.id] != jobs[a.id] {
		return fmt.Errorf("the pledge held for the jobs that name it is %s, the jobs imply %s", pairText(s.held[a.id]), pairText(jobs[a.id]))
	}
	load, ok := s.pendingLoad(&a.job)
	prin := a.id
	if a.job.executions > 0 {
		prin = a.job.principal
	}
	if !ok || s.jobLoad[a.id] != load || s.jobPrin[a.id] != prin {
		return fmt.Errorf("its job's pledge is %s on agent %d, its job implies %s on agent %d", pairText(s.jobLoad[a.id]), s.jobPrin[a.id], pairText(load), prin)
	}
	return nil
}

// auditGone checks that a removed agent leaves nothing behind in the module: no
// store, no cargo expected, no pledge held or placed (dropStorage).
func (s *storage) auditGone(a *agent) error {
	if int(a.id) >= len(s.stores) {
		return nil
	}
	if s.owned(a.id) > 0 || s.inbound[a.id] != (room.Pair{}) || s.held[a.id] != (room.Pair{}) ||
		s.jobLoad[a.id] != (room.Pair{}) || s.jobPrin[a.id] != a.id {
		return errors.New("a removed agent still has stores, cargo or a pledge in the module's books")
	}
	return nil
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
		return fmt.Errorf("reserved room %s at a boundary", pairText(room.Pair{Mass: rm.ReservedMass, Bulk: rm.ReservedBulk}))
	}
	if err := s.auditPledge(rm, a, jobs); err != nil {
		return err
	}
	if r := s.roomFor(a); rm.Mass != r.Mass || rm.Bulk != r.Bulk {
		return fmt.Errorf("room %s, its role, weight and stores give %s", pairText(room.Pair{Mass: rm.Mass, Bulk: rm.Bulk}), pairText(r))
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
		return fmt.Errorf("pledge %s, its job and cargo imply %s", pairText(room.Pair{Mass: rm.PledgeMass, Bulk: rm.PledgeBulk}), pairText(want))
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
			return fmt.Errorf("storage: agent %d expects cargo of load %s, the shipments in flight carry %s", id, pairText(s.inbound[id]), pairText(want[id]))
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

// viewed counts a household group given a Trade view: the households of a limited
// agent, and those whose free room is 0 in a dimension (spec-storage.md 14.1). They
// ride along in RoomLimited and are counted whether or not an observer is supplied.
func (s *storage) viewed(cell int, weight int64, free room.Pair) {
	c := &s.counts[cell]
	c.limitedHouseholds = satAdd(c.limitedHouseholds, weight)
	if free.Mass <= 0 || free.Bulk <= 0 {
		c.fullHouseholds = satAdd(c.fullHouseholds, weight)
	}
}

// checkRoomPlan is the module's step of checkTradePlan: it counts what the default
// strategy cut for room (the bids sized down or dropped) and what it discards, in the
// cell of the agent, and logs each bid cut at Debug. Only the default strategy's
// discard of 6.3.2 is counted (roomDiscarded): a directive's or a consumer's discard
// is not (spec-storage.md 5.3.4). It changes nothing in the plan and takes no random
// word.
func (w *World) checkRoomPlan(a *agent, p *TradePlan) {
	if len(p.Cuts) == 0 && p.roomDiscarded == 0 {
		return
	}
	if rm, _, ok := w.sto.roomState(w.ledger, a); !ok || !rm.Limited {
		return
	}
	c := &w.sto.counts[w.cellOf(w.regionOf(a), a)]
	c.bidsCut = satAdd(c.bidsCut, int64(len(p.Cuts)))
	c.discarded = satAdd(c.discarded, p.roomDiscarded)
	ctx := context.Background()
	if !w.log.Enabled(ctx, slog.LevelDebug) {
		return
	}
	for _, cut := range p.Cuts {
		w.log.LogAttrs(ctx, slog.LevelDebug, "bid cut for room",
			slog.Int64(logkeys.Tick, w.Tick()), slog.Int64(logkeys.Agent, int64(a.id)),
			slog.Int64(logkeys.Commodity, int64(cut.Commodity)), slog.Int64(logkeys.Quantity, int64(cut.Quantity)))
	}
}

// logRefusedJob is the Debug line of a job request refused for room (spec-storage.md
// 14.3), behind Enabled; the reason is no_room.
func (w *World) logRefusedJob(a *agent, recipe RecipeID, reason Reason) {
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "job refused for room",
			slog.Int64(logkeys.Tick, w.Tick()), slog.Int64(logkeys.Agent, int64(a.id)),
			slog.Int64(logkeys.Recipe, int64(recipe)), slog.String(logkeys.Reason, string(reason)))
	}
}

// logStore is the Debug line of a store added or removed, with the class and the
// cause, behind Enabled (spec-storage.md 14.3); it is written with or without an
// observer.
func (w *World) logStore(a *agent, c int, kind Kind, cause Cause) {
	ctx := context.Background()
	if !w.log.Enabled(ctx, slog.LevelDebug) {
		return
	}
	msg := "store added"
	if kind == KindRemoved {
		msg = "store removed"
	}
	w.log.LogAttrs(ctx, slog.LevelDebug, msg,
		slog.Int64(logkeys.Tick, w.Tick()), slog.Int64(logkeys.Agent, int64(a.id)),
		slog.Int64(logkeys.StoreClass, int64(c)), slog.String(logkeys.Cause, string(cause)))
}

// p36aRoomLimited is P3.6a, after WantShortfall: the RoomLimited of each (region,
// role) in which a room cut or refused something, in ascending region then role
// (spec-storage.md 14.1, FR-STO-036). The counters were made as the acts happened,
// with or without an observer; the event is built only when one was supplied
// (observability-and-logging.md 4.5). They are cleared at P8.6, so none survives it.
func (w *World) p36aRoomLimited() {
	if !w.obs.supplied {
		return
	}
	roles := len(w.cfg.Roles)
	for cell := range w.sto.counts {
		c := &w.sto.counts[cell]
		if c.startsCut == 0 && c.startsRefused == 0 && c.bidsCut == 0 && c.bidsRejected == 0 && c.shipmentsRefused == 0 && c.discarded == 0 {
			continue
		}
		w.obs.deliver(RoomLimited{
			Tick: w.Tick(), Region: RegionID(cell / roles), Role: RoleID(cell % roles),
			LimitedHouseholds: c.limitedHouseholds, FullHouseholds: c.fullHouseholds,
			StartsCut: c.startsCut, StartsRefused: c.startsRefused, BidsCut: c.bidsCut,
			BidsRejected: c.bidsRejected, ShipmentsRefused: c.shipmentsRefused, Discarded: c.discarded,
		})
	}
}
