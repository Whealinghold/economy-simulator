package economysim

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the arrival side of the transport module (spec-transport.md 5.6 to
// 5.9, 7.4.1): P1.3, the arrival of a shipment (the traveller's move, the beliefs
// reseeded, the cargo delivered or diverted, the account released), the guard of a
// relocation by the shipments an agent holds, the step of a removal and the audit.

// openShipment is the name of the region_bound refusal of the module
// (spec-regions.md 6.5.2).
const openShipment = "open_shipment"

// arrival is the record of one arrived shipment that P1.3 keeps for P1.4 and P1.3a
// (spec-transport.md 13.6): the lines are a range of arrivedLines. It is kept
// whether or not an observer was supplied.
type arrival struct {
	shipment    ShipmentID
	first, size int
}

// holds reports whether the agent holds an instance of the open_shipment
// obligation that skip does not name by key: it is the traveller of an away trip, or
// the shipper or the consignee of a shipment in flight. The instance key of each is
// the shipment ID.
func (t *transportState) holds(a *agent, skip []string) bool {
	for i := range t.ships {
		s := &t.ships[i]
		bound := s.shipper == a.id || s.consignee == a.id // a traveller is its trip's shipper
		if bound && !skipHas(skip, openShipment, int64(s.id)) {
			return true
		}
	}
	return false
}

// p13Arrivals is P1.3: the shipments whose arrival tick has come, in ascending
// shipment ID, each delivered completely before the next (spec-transport.md 5.7). The
// schedule gives them in that order, so the work of a tick follows the arrivals and
// not the shipments in flight (NFR-TRN-002). The arrivals are recorded in the
// scratch of the module for the later positions.
func (w *World) p13Arrivals() {
	t := w.transport
	t.arrived, t.arrivedLines = t.arrived[:0], t.arrivedLines[:0]
	tick := w.Tick()
	for {
		id, ok := t.due.Due(tick)
		if !ok {
			return
		}
		i, found := slices.BinarySearchFunc(t.ships, ShipmentID(id), func(s shipment, id ShipmentID) int { return cmp.Compare(s.id, id) })
		if !found {
			continue // dropped with its traveller (dropTraveller)
		}
		if w.arrive(t.ships[i]) {
			t.ships = slices.Delete(t.ships, i, i+1)
			continue
		}
		t.ships[i].arrives = tick + 1 // held by a refusal of the move: retried next tick
		t.due.Push(tick+1, id)
	}
}

// arrive processes one arrival and reports whether the shipment is complete. A
// traveller whose account cannot move now (another obligation refuses) holds the
// whole arrival: nothing is delivered and the shipment is retried next tick.
func (w *World) arrive(s shipment) bool {
	t := w.transport
	if w.travellerLive(&s) && !w.moveTraveller(&w.agents[s.traveller], &s) {
		return false
	}
	if !s.hired {
		t.setAway(s.traveller, false)
	}
	first := len(t.arrivedLines)
	receiver, diverted := w.deliver(&s)
	if s.account >= 0 {
		t.free[s.from].Put(s.account)
	}
	t.arrived = append(t.arrived, arrival{shipment: s.id, first: first, size: len(t.arrivedLines) - first})
	if w.obs.supplied {
		w.deliverArrived(&s, receiver, diverted, t.arrivedLines[first:])
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "shipment arrived",
			slog.Int64(logkeys.Shipment, int64(s.id)), slog.Int64(logkeys.Dest, int64(s.to)))
	}
	t.take(s.lines)
	return true
}

// travellerLive reports whether the shipment is self-carried and its traveller still
// exists.
func (w *World) travellerLive(s *shipment) bool {
	return !s.hired && w.liveAgent(s.traveller)
}

// liveAgent reports whether the ID is an agent that exists: not a region's estate
// pseudo-agent, issued, and of weight 1 or more.
func (w *World) liveAgent(id AgentID) bool {
	return id >= AgentID(len(w.cfg.Regions)) && id < w.state.nextAgent && w.agents[id].weight >= 1
}

// moveTraveller relocates the traveller to the destination: the guard query run
// skipping the arriving shipment and the seam of the contract, then the one path
// that moves an account (cause transport), then the beliefs reseeded with no draw.
// It reports false, changing nothing, when another obligation refuses.
func (w *World) moveTraveller(a *agent, s *shipment) bool {
	t := w.transport
	key := string(strconv.AppendInt(append(t.key[:0], openShipment+":"...), int64(s.id), 10))
	t.skip = append(append(t.skip[:0], key), w.arrivalSkips(s)...)
	if reason, err := w.relocateRefusal(a.id, s.to, t.skip); err != nil {
		if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
			w.log.LogAttrs(ctx, slog.LevelDebug, "arrival held",
				slog.Int64(logkeys.Shipment, int64(s.id)), slog.Int64(logkeys.Agent, int64(a.id)), slog.String(logkeys.Reason, string(reason)))
		}
		return false
	}
	if err := w.moveAgent(a, s.to, CauseTransport); err != nil {
		invariant("the ledger refused the relocation of an arriving traveller: " + err.Error())
	}
	w.reseedAtArrival(a, s.to)
	return true
}

// reseedAtArrival reseeds the price beliefs of an agent that has just arrived in
// region to by the rule of spec-beliefs.md 8.3: for every commodity of its
// tradable set the centre is the clamp of the destination's window mean (the market
// price while the window is empty) and the width is BeliefWidthInit. It draws
// nothing; plan 14 shares it for the arrival of a migrant.
func (w *World) reseedAtArrival(a *agent, to RegionID) {
	row := w.roles.row(a.role)
	w.fillSeeds(to, row)
	reseedBeliefs(a, row, nil, w.ex.seeds, w.bp)
}

// deliver moves the cargo of the Transit account, line by line in commodity order,
// to the consignee when it can hold it, else to the estate of the destination
// (spec-transport.md 5.7 item 2, 5.8). What the account no longer holds spoiled on
// the way. It records the lines and returns the receiver and whether any went to
// the estate.
func (w *World) deliver(s *shipment) (receiver AgentID, diverted bool) {
	t := w.transport
	cons := w.consigneeAccount(s)
	receiver = s.consignee
	for _, ln := range s.lines {
		have, _, err := w.ledger.Holding(s.account, int(ln.Commodity))
		if err != nil {
			invariant("the ledger refused the holding of a Transit account: " + err.Error())
		}
		rec := ArrivedLine{Commodity: ln.Commodity, Shipped: ln.Quantity, Spoiled: ln.Quantity - Quantity(have)}
		to, ok := cons, cons >= 0 && w.consigneeHolds(cons, ln.Commodity, have)
		if !ok {
			to, diverted, receiver = w.estates.accounts[s.to], true, AgentID(s.to)
			rec.Diverted = Quantity(have)
			w.logDiverted(s)
		} else {
			rec.Delivered = Quantity(have)
		}
		if have > 0 {
			if err := w.ledger.Move(s.account, to, int(ln.Commodity), have, ledger.TransitIn); err != nil {
				invariant("the ledger refused the delivery of a cargo: " + err.Error())
			}
		}
		t.arrivedLines = append(t.arrivedLines, rec)
	}
	return receiver, diverted
}

// consigneeAccount returns the account of the consignee when the agent still
// exists and is in the destination, else -1.
func (w *World) consigneeAccount(s *shipment) int {
	if !w.liveAgent(s.consignee) {
		return -1
	}
	if a := &w.agents[s.consignee]; w.regionOf(a) == s.to {
		return a.account
	}
	return -1
}

// consigneeHolds reports whether the account can take qty more units of the
// commodity within MaxQuantity (the room of the storage module is its own seam, 11a).
func (w *World) consigneeHolds(account int, c CommodityID, qty int64) bool {
	held, _, err := w.ledger.Holding(account, int(c))
	if err != nil {
		invariant("the ledger refused the holding of a consignee: " + err.Error())
	}
	return held+qty <= MaxQuantity
}

// logDiverted writes the Debug line of a cargo that goes to the estate.
func (w *World) logDiverted(s *shipment) {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "cargo diverted",
			slog.Int64(logkeys.Shipment, int64(s.id)), slog.Int64(logkeys.Dest, int64(s.to)), slog.Int64(logkeys.Agent, int64(s.consignee)))
	}
}

// deliverArrived builds and delivers ShipmentArrived; the lines are a copy.
func (w *World) deliverArrived(s *shipment, receiver AgentID, diverted bool, lines []ArrivedLine) {
	w.obs.deliver(ShipmentArrived{
		Tick: w.Tick(), Shipment: s.id, Mode: s.mode, Region: s.from, To: s.to,
		Receiver: receiver, Diverted: diverted, Lines: slices.Clone(lines),
	})
}

// dropTraveller is the transport step of the removal of an agent (spec-transport.md
// 5.9.2, spec-agents.md 3.6 step 2a), before its money moves: a trip to oneself is
// dropped and the away flag cleared; a shipment with lines continues without a
// traveller. A shipper or a consignee changes no record.
func (w *World) dropTraveller(a *agent) {
	t := w.transport
	if !t.isAway(a.id) {
		return
	}
	t.setAway(a.id, false)
	for i := range t.ships {
		if s := &t.ships[i]; !s.hired && s.traveller == a.id {
			if s.account < 0 {
				t.take(s.lines)
				t.ships = slices.Delete(t.ships, i, i+1)
			}
			return
		}
	}
}

// auditTransport is the transport part of the audit (spec-transport.md 7.4.1): the
// IDs of the shipments in flight are unique and ascending and none arrives before
// the current tick; a shipment with an account references an open Transit account of
// its origin region that holds only its commodities and no more of each than was
// shipped; a Transit account no shipment references is empty; an agent is away if
// and only if it is the traveller of a live self-carried shipment. The error names
// the check and the shipment and wraps ErrAuditFailed.
func (w *World) auditTransport() error {
	t := w.transport
	fail := func(format string, args ...any) error {
		return fmt.Errorf("audit at tick %d: %w: transport: %s", w.Tick(), ErrAuditFailed, fmt.Sprintf(format, args...))
	}
	used := map[int]bool{}
	travellers := 0
	for i := range t.ships {
		s := &t.ships[i]
		if i > 0 && s.id <= t.ships[i-1].id {
			return fail("shipment %d is not above shipment %d", s.id, t.ships[i-1].id)
		}
		if s.arrives < w.Tick() {
			return fail("shipment %d arrives at tick %d, before the current tick", s.id, s.arrives)
		}
		if err := w.auditAccount(s, used); err != "" {
			return fail("shipment %d: %s", s.id, err)
		}
		if w.travellerLive(s) {
			if !t.isAway(s.traveller) {
				return fail("shipment %d has a traveller that is not away", s.id)
			}
			travellers++
		}
	}
	if away := t.awayCount(); away != travellers {
		return fail("%d agents are away and %d live self-carried shipments have a traveller", away, travellers)
	}
	return w.auditFreeAccounts(used, fail)
}

// awayCount returns the number of agents that are away.
func (t *transportState) awayCount() (n int) {
	for _, f := range t.away {
		if f {
			n++
		}
	}
	return n
}

// shippedOf returns the units of a commodity the shipment's lines carry.
func (s *shipment) shippedOf(c int) (n int64) {
	for _, ln := range s.lines {
		if int(ln.Commodity) == c {
			n = int64(ln.Quantity)
		}
	}
	return n
}

// auditAccount checks the Transit account of one shipment and marks it used. It
// returns the failure, or "".
func (w *World) auditAccount(s *shipment, used map[int]bool) string {
	if s.account < 0 {
		return ""
	}
	v, err := w.ledger.Account(s.account)
	if err != nil || v.Closed || v.Kind != ledger.KindTransit || v.Region != int(s.from) {
		return fmt.Sprintf("account %d is not an open Transit account of region %d", s.account, s.from)
	}
	if used[s.account] {
		return fmt.Sprintf("account %d is used by two shipments", s.account)
	}
	used[s.account] = true
	for c := range w.cfg.Commodities {
		have, _, err := w.ledger.Holding(s.account, c)
		if err != nil {
			return fmt.Sprintf("account %d: the ledger refused the holding of commodity %d: %v", s.account, c, err)
		}
		if shipped := s.shippedOf(c); have > shipped {
			return fmt.Sprintf("account %d holds %d of commodity %d, more than the %d shipped", s.account, have, c, shipped)
		}
	}
	return ""
}

// auditFreeAccounts checks that every open Transit account that no shipment uses is
// empty.
func (w *World) auditFreeAccounts(used map[int]bool, fail func(string, ...any) error) error {
	for id := range w.ledger.Len() {
		v, err := w.ledger.Account(id)
		if err != nil {
			return fail("the ledger refused account %d: %v", id, err)
		}
		if v.Kind != ledger.KindTransit || v.Closed || used[id] {
			continue
		}
		for c := range w.cfg.Commodities {
			have, _, err := w.ledger.Holding(id, c)
			if err != nil {
				return fail("the ledger refused the holding of commodity %d in account %d: %v", c, id, err)
			}
			if have != 0 {
				return fail("free Transit account %d holds %d of commodity %d", id, have, c)
			}
		}
	}
	return nil
}
