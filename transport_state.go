package economysim

import (
	"fmt"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file is the state side of the transport module (spec-transport.md 14,
// spec-checkpoint.md 4.2 tag 17, spec-observability.md 5.3.7 and 5.3.8): the snapshot
// rows, the checkpoint section and the derived tables a restore rebuilds.

// ShipmentRow is a shipment in flight in the snapshot: the public copy of the
// record of spec-transport.md 5.2. Traveller is 0 for a hired shipment, Account is
// the Transit ledger account and -1 for a trip to oneself, and Lines are the
// commodities and quantities shipped, in commodity order.
type ShipmentRow struct {
	Shipment           ShipmentID
	Mode               ModeID
	Region, To         RegionID
	Shipper, Traveller AgentID
	Consignee          AgentID
	Hired              bool
	// Dispatched and Arrives are ticks; TicksLeft is Arrives less the tick of the
	// snapshot.
	Dispatched, Arrives, TicksLeft int64
	Account                        int
	// Units are the carrying units used and HaulUnits the haul units redeemed.
	Units, HaulUnits int64
	Lines            []Amount
}

// TransitFactorRow is a transit factor that is not the default 10 000 in the
// snapshot, in basis points.
type TransitFactorRow struct {
	From, To RegionID
	Mode     ModeID
	BP       int64
}

// snapshotShipments returns the rows of the shipments in flight, in shipment ID
// order, each with its own copy of the lines. It is nil with the module off.
func (w *World) snapshotShipments() []ShipmentRow {
	t := w.transport
	if t == nil || len(t.ships) == 0 {
		return nil
	}
	rows := make([]ShipmentRow, len(t.ships))
	for i := range t.ships {
		s := &t.ships[i]
		rows[i] = ShipmentRow{
			Shipment: s.id, Mode: s.mode, Region: s.from, To: s.to, Shipper: s.shipper, Traveller: s.traveller,
			Consignee: s.consignee, Hired: s.hired, Dispatched: s.dispatched, Arrives: s.arrives,
			TicksLeft: s.arrives - w.Tick(), Account: s.account, Units: s.units, HaulUnits: s.haulUnits,
			Lines: make([]Amount, len(s.lines)),
		}
		for j, ln := range s.lines {
			rows[i].Lines[j] = Amount(ln)
		}
	}
	return rows
}

// snapshotAway returns the IDs of the agents away on a trip, ascending. It is nil
// when none is.
func (w *World) snapshotAway() []AgentID {
	if w.transport == nil {
		return nil
	}
	var out []AgentID
	for id, away := range w.transport.away {
		if away {
			out = append(out, AgentID(id))
		}
	}
	return out
}

// snapshotFactors returns the transit factors that are not 10 000, by (from, to,
// mode). It is nil when there is none.
func (w *World) snapshotFactors() []TransitFactorRow {
	if w.transport == nil || len(w.transport.factors) == 0 {
		return nil
	}
	rows := make([]TransitFactorRow, len(w.transport.factors))
	for i, f := range w.transport.factors {
		rows[i] = TransitFactorRow{From: f.from, To: f.to, Mode: f.mode, BP: f.bp}
	}
	return rows
}

// saveTransport is tag 17 (spec-checkpoint.md 4.2): the next shipment ID; the
// shipments in flight in ID order, each with its mode and regions by index, its
// Transit account as account + 1 (0 is a trip to oneself) and its lines in
// commodity order; then the transit factors that are not 10 000 as (from, to, mode,
// factor). The away flags, the free lists of Transit accounts and the scratch are
// derived and not saved.
func (w *World) saveTransport(cw *codec.Writer) error {
	t := w.transport
	cw.U(unsigned(t.next))
	cw.Count(len(t.ships))
	for i := range t.ships {
		s := &t.ships[i]
		cw.U(unsigned(s.id))
		cw.U(unsigned(s.mode))
		cw.U(unsigned(s.from))
		cw.U(unsigned(s.to))
		cw.U(unsigned(s.shipper))
		cw.U(unsigned(s.traveller))
		cw.U(unsigned(s.consignee))
		cw.B(s.hired)
		cw.U(unsigned(s.dispatched))
		cw.U(unsigned(s.arrives))
		cw.U(unsigned(s.account + 1))
		cw.U(unsigned(s.units))
		cw.U(unsigned(s.haulUnits))
		cw.Count(len(s.lines))
		for _, ln := range s.lines {
			cw.U(unsigned(ln.Commodity))
			cw.U(unsigned(ln.Quantity))
		}
	}
	cw.Count(len(t.factors))
	for _, f := range t.factors {
		cw.U(unsigned(f.from))
		cw.U(unsigned(f.to))
		cw.U(unsigned(f.mode))
		cw.U(unsigned(f.bp))
	}
	return nil
}

// loadTransport reads tag 17: the next shipment ID, the shipments with strictly
// ascending IDs below it, then the factors with strictly ascending keys, each at
// most MaxTransitFactorBP and not 10 000. A reference to an agent need only be
// below the next agent ID, because a shipment outlives its shipper, traveller and
// consignee (spec-checkpoint.md 4.2.3). That the Transit account exists and holds
// what the record says is the audit of the restore (auditTransport); the away flags
// are derived here from the shipments, and the free lists in transportCross.
func (w *World) loadTransport(rs *restoreCtx, r *codec.Reader) error {
	t := w.transport
	t.next = ShipmentID(r.Bounded("next shipment", maxID))
	w.readShipments(rs, r)
	w.readFactors(r)
	t.due.Reset()
	for i := range t.ships {
		s := &t.ships[i]
		t.due.Push(s.arrives, int64(s.id))
		if !s.hired && w.travellerLive(s) {
			t.setAway(s.traveller, true)
		}
	}
	return nil
}

// readShipments reads the shipments of tag 17 into the state.
func (w *World) readShipments(rs *restoreCtx, r *codec.Reader) {
	t := w.transport
	for i, n := 0, r.Count(14); i < n && r.Err() == nil; i++ {
		s := w.readShipment(rs, r)
		if r.Err() != nil {
			return
		}
		if s.id >= t.next || (i > 0 && s.id <= t.ships[i-1].id) {
			r.Failf("shipment %d is not in strictly ascending order below the next shipment %d", s.id, t.next)
			return
		}
		t.ships = append(t.ships, s)
	}
}

// readFactors reads the transit factors of tag 17 into the state.
func (w *World) readFactors(r *codec.Reader) {
	t := w.transport
	for i := range r.Count(4) {
		f := factorRec{
			from: RegionID(r.Index(uint64(len(w.cfg.Regions)))), to: RegionID(r.Index(uint64(len(w.cfg.Regions)))),
			mode: ModeID(r.Index(uint64(len(t.modes)))), bp: r.Bounded("transit factor", MaxTransitFactorBP),
		}
		switch {
		case r.Err() != nil:
			return
		case f.from == f.to || f.bp == defaultTransitBP:
			r.Failf("factor row %d is between a region and itself or at the default, which is not saved", i)
			return
		case i > 0 && cmpFactorKey(t.factors[i-1], f) >= 0:
			r.Failf("factor row %d is not above the row before it", i)
			return
		}
		t.factors = append(t.factors, f)
	}
}

// maxID is the largest ID a checkpoint carries as an unsigned number.
const maxID = 1<<63 - 1

// readShipment reads one shipment record and checks its fields against the
// configuration, the tick and the shape of 5.2: a hired shipment has no traveller,
// a self-carried one is travelled by its shipper unless it is a trip to oneself (no
// lines, no account), and a shipment with lines has a Transit account.
func (w *World) readShipment(rs *restoreCtx, r *codec.Reader) shipment {
	t := w.transport
	next := int64(w.state.nextAgent)
	s := shipment{id: ShipmentID(r.Bounded("shipment", maxID))}
	s.mode = ModeID(r.Index(uint64(len(t.modes))))
	s.from = RegionID(r.Index(uint64(len(w.cfg.Regions))))
	s.to = RegionID(r.Index(uint64(len(w.cfg.Regions))))
	s.shipper = AgentID(r.Bounded("shipper", next-1))
	s.traveller = AgentID(r.Bounded("traveller", next-1))
	s.consignee = AgentID(r.Bounded("consignee", next-1))
	s.hired = r.B()
	s.dispatched = r.Bounded("dispatch tick", rs.tick-1)
	s.arrives = r.Bounded("arrival tick", rs.tick+MaxTravelTicks)
	s.account = int(r.Bounded("transit account", int64(w.ledger.Len()))) - 1
	s.units = r.Bounded("carrying units", MaxQuantity*MaxShipmentLines)
	s.haulUnits = r.Bounded("haul units", MaxQuantity)
	prev := uint64(0)
	m := r.Count(2)
	for i := 0; i < m && r.Err() == nil; i++ {
		c := r.Strict(prev, i > 0)
		q := r.Bounded("shipped quantity", MaxQuantity)
		if r.Err() == nil && (c >= uint64(len(w.cfg.Commodities)) || q < 1) {
			r.Failf("shipment %d line %d names commodity %d with %d units", s.id, i, c, q)
		}
		prev = c
		s.lines = append(s.lines, Line{Commodity: CommodityID(c), Quantity: Quantity(q)})
	}
	w.checkShipmentShape(r, &s)
	return s
}

// checkShipmentShape reports a record that cannot come from a dispatch.
func (w *World) checkShipmentShape(r *codec.Reader, s *shipment) {
	if r.Err() != nil {
		return
	}
	md := &w.cfg.Modules.Transport.Modes[s.mode]
	faults := [...]struct {
		bad  bool
		what string
	}{
		{s.from == s.to, "is between a region and itself"},
		{s.hired != md.hired(), "is hired, or not, against its mode"},
		{s.hired && s.traveller != 0, "is hired and has a traveller"},
		{!s.hired && s.traveller != s.shipper, "is self-carried and its traveller is not its shipper"},
		{(len(s.lines) == 0) != (s.account < 0), "has lines and no Transit account, or the converse"},
		{s.hired && len(s.lines) == 0, "is hired and has no lines"},
		{s.arrives <= s.dispatched || s.units < 1, "arrives at or before its dispatch, or carries no unit"},
	}
	for _, f := range faults {
		if f.bad {
			r.Failf("shipment %d %s", s.id, f.what)
			return
		}
	}
}

// transportCross is the module's part of step 7: the free list of each region is
// every open Transit account of the region that no shipment names (spec-checkpoint.md
// 6.2 step 8). That every account a shipment names exists and holds what the record
// says is checked by the audit that follows.
func (w *World) transportCross() error {
	t := w.transport
	used := make(map[int]bool, len(t.ships))
	for i := range t.ships {
		if a := t.ships[i].account; a >= 0 {
			if used[a] {
				return fmt.Errorf("%w: Transit account %d is named by two shipments", ErrCheckpointCorrupt, a)
			}
			used[a] = true
		}
	}
	free := make([][]int, len(t.free))
	for id := range w.ledger.Len() {
		v, err := w.ledger.Account(id)
		if err != nil || v.Kind != ledger.KindTransit || v.Closed || used[id] {
			continue
		}
		free[v.Region] = append(free[v.Region], id)
	}
	for g := range t.free {
		t.free[g].Rebuild(free[g])
	}
	return nil
}
