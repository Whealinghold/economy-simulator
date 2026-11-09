package economysim

import (
	"cmp"
	"context"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/transit"
)

// This file is the view side of the transport module (spec-transport.md 9, 10.3):
// the tables fixed at P3.1, the filler of the AgentView, the four queries and the
// transit-factor slot.

// inRec is one row of the inbound table: a commodity on the way to an agent. outRec
// is one row of the outbound table: a commodity on the way from an agent to a
// region. Both are sorted and merged at P3.1.
type inRec struct {
	agent     AgentID
	commodity CommodityID
	qty       Quantity
	earliest  int64
}

type outRec struct {
	agent     AgentID
	to        RegionID
	commodity CommodityID
	qty       Quantity
	earliest  int64
}

// viewTables is what P3.1 fixes for the tick: the inbound and outbound rows of every
// agent, the route rows of every region that holds an agent, and the scratch the
// filler copies a view's rows into. at is the tick they are valid for.
type viewTables struct {
	at      int64
	fixed   bool
	in      []inRec
	out     []outRec
	routes  [][]RouteRow
	inView  []InboundRow
	outView []OutboundRow
}

// p31Fix is P3.1: the shipments in flight, which fix Inbound and Outbound for every
// agent of the tick before any agent is called, so that they do not depend on the
// order of agents (spec-transport.md 5.3.3, 9.1), and the table of routes of every
// region that holds an agent. Nothing is drawn.
func (w *World) p31Fix() {
	t := w.transport
	v := &t.tables
	v.at, v.fixed = w.Tick(), true
	v.in, v.out = v.in[:0], v.out[:0]
	for i := range t.ships {
		s := &t.ships[i]
		for _, ln := range s.lines {
			v.in = append(v.in, inRec{s.consignee, ln.Commodity, ln.Quantity, s.arrives})
			v.out = append(v.out, outRec{s.shipper, s.to, ln.Commodity, ln.Quantity, s.arrives})
		}
	}
	slices.SortFunc(v.in, func(a, b inRec) int {
		return cmp.Or(cmp.Compare(a.agent, b.agent), cmp.Compare(a.commodity, b.commodity))
	})
	v.in = mergeRows(v.in, func(a, b inRec) bool { return a.agent == b.agent && a.commodity == b.commodity },
		func(a *inRec, b inRec) { a.qty += b.qty; a.earliest = min(a.earliest, b.earliest) })
	slices.SortFunc(v.out, func(a, b outRec) int {
		return cmp.Or(cmp.Compare(a.agent, b.agent), cmp.Compare(a.to, b.to), cmp.Compare(a.commodity, b.commodity))
	})
	v.out = mergeRows(v.out, func(a, b outRec) bool { return a.agent == b.agent && a.to == b.to && a.commodity == b.commodity },
		func(a *outRec, b outRec) { a.qty += b.qty; a.earliest = min(a.earliest, b.earliest) })
	w.buildRoutes()
}

// mergeRows merges the adjacent rows of a sorted slice that same calls equal, with
// add folding the later into the earlier, in place.
func mergeRows[T any](rows []T, same func(a, b T) bool, add func(a *T, b T)) []T {
	out := rows[:0]
	for _, r := range rows {
		if n := len(out); n > 0 && same(out[n-1], r) {
			add(&out[n-1], r)
			continue
		}
		out = append(out, r)
	}
	return out
}

// buildRoutes fills the route rows of every region that holds an agent, in region
// then mode order, at the transit factor in force. The rows of a region without an
// agent are cleared.
func (w *World) buildRoutes() {
	t := w.transport
	v := &t.tables
	regions, modes := len(w.cfg.Regions), len(t.modes)
	for len(v.routes) < regions {
		v.routes = append(v.routes, nil)
	}
	roles := len(w.cfg.Roles)
	for g := range regions {
		v.routes[g] = v.routes[g][:0]
		if !w.regionHolds(g, roles) {
			continue
		}
		for to := range regions {
			if to == g {
				continue
			}
			for m := range modes {
				ticks, haul, ok := w.travel(RegionID(g), RegionID(to), ModeID(m))
				v.routes[g] = append(v.routes[g], RouteRow{To: RegionID(to), Mode: ModeID(m), Has: ok, Ticks: ticks, HaulPerUnit: haul})
			}
		}
	}
}

// regionHolds reports whether a live household has its account in region g.
func (w *World) regionHolds(g, roles int) bool {
	for r := range roles {
		if w.live.inRole[g*roles+r] > 0 {
			return true
		}
	}
	return false
}

// fillTransport sets the three transport groups of a view (spec-transport.md 9.1)
// from the tables of P3.1, as copies into the scratch the module reuses. While the
// tables are not fixed for this tick (the views of phase 2) the groups are empty.
func (w *World) fillTransport(v *AgentView, a *agent) {
	t := w.transport
	tb := &t.tables
	v.Inbound, v.Outbound, v.Routes = nil, nil, nil
	if !tb.fixed || tb.at != w.Tick() {
		return
	}
	tb.fillInOut(v, a.id)
	if g := w.regionOf(a); int(g) < len(tb.routes) && len(tb.routes[g]) > 0 {
		v.Routes = tb.routes[g][:len(tb.routes[g]):len(tb.routes[g])]
	}
}

// fillInOut copies the inbound and outbound rows of an agent into the scratch and
// points the view at them; a group with no row stays nil.
func (tb *viewTables) fillInOut(v *AgentView, id AgentID) {
	lo, _ := slices.BinarySearchFunc(tb.in, id, func(r inRec, id AgentID) int { return cmp.Compare(r.agent, id) })
	tb.inView = tb.inView[:0]
	for _, r := range tb.in[lo:] {
		if r.agent != id {
			break
		}
		tb.inView = append(tb.inView, InboundRow{Commodity: r.commodity, Quantity: r.qty, Earliest: r.earliest})
	}
	lo, _ = slices.BinarySearchFunc(tb.out, id, func(r outRec, id AgentID) int { return cmp.Compare(r.agent, id) })
	tb.outView = tb.outView[:0]
	for _, r := range tb.out[lo:] {
		if r.agent != id {
			break
		}
		tb.outView = append(tb.outView, OutboundRow{To: r.to, Commodity: r.commodity, Quantity: r.qty, Earliest: r.earliest})
	}
	if n := len(tb.inView); n > 0 {
		v.Inbound = tb.inView[:n:n]
	}
	if n := len(tb.outView); n > 0 {
		v.Outbound = tb.outView[:n:n]
	}
}

// travel returns the travel time of a mode between two regions at the transit
// factor in force, the haul units one carrying unit costs (0 for a self-carried
// mode) and whether there is a route: both regions and the mode known, a path, a
// factor above 0 and a time within MaxTravelTicks.
func (w *World) travel(from, to RegionID, mode ModeID) (ticks, haulPerUnit int64, ok bool) {
	if w.factorKey(from, to, mode) != nil || from == to {
		return 0, 0, false
	}
	d, ok := w.geo.distance(from, to)
	f := w.transport.factorOf(from, to, mode)
	md := &w.cfg.Modules.Transport.Modes[mode]
	if !ok || f == 0 {
		return 0, 0, false
	}
	k, ok := transit.Ticks(d, md.Speed, f)
	if !ok || k > MaxTravelTicks {
		return 0, 0, false
	}
	if md.hired() {
		haulPerUnit, _ = transit.HaulPerUnit(d, md.HaulSpan)
	}
	return k, haulPerUnit, true
}

// Travel returns the travel time in ticks of a mode from one region to another at
// the transit factor in force, and the haul units one carrying unit costs for a
// hired mode (0 for a self-carried one); ok is false when there is no route: the
// module is off, a region or the mode is unknown, the regions are the same, the
// graph has no path, the factor is 0 or the time is over MaxTravelTicks. It is pure,
// changes nothing and draws nothing (spec-transport.md 9.2).
func (w *World) Travel(from, to RegionID, mode ModeID) (ticks, haulPerUnit int64, ok bool) {
	return w.travel(from, to, mode)
}

// InTransit returns, in commodity order, the units on the way in the shipments of
// which the agent is the shipper or the consignee, each shipment once
// (spec-transport.md 7.4.2). A trip to oneself carries its goods in the traveller's
// own account and is not in it. The slice is the caller's.
func (w *World) InTransit(agent AgentID) []Amount {
	if w.transport == nil {
		return nil
	}
	var out []Amount
	for i := range w.transport.ships {
		s := &w.transport.ships[i]
		if s.shipper != agent && s.consignee != agent {
			continue
		}
		for _, ln := range s.lines {
			j, found := slices.BinarySearchFunc(out, ln.Commodity, func(a Amount, c CommodityID) int { return cmp.Compare(a.Commodity, c) })
			if found {
				out[j].Quantity += ln.Quantity
			} else {
				out = slices.Insert(out, j, Amount(ln))
			}
		}
	}
	return out
}

// Away reports whether the agent is the traveller of a self-carried shipment in
// flight: it is in no market and is passed to neither Work nor Trade.
func (w *World) Away(agent AgentID) bool {
	return w.transport != nil && w.transport.isAway(agent)
}

// RouteLoss returns the expected loss in basis points of a route: 0 with ok true
// for a route while the risk module is off, and ok false when there is no route
// (the formula is the risk module's, which replaces the body).
func (w *World) RouteLoss(from, to RegionID, mode ModeID) (bp int64, ok bool) {
	_, _, ok = w.travel(from, to, mode)
	return 0, ok
}

// factorKey checks that the module is on and the mode and both regions are
// defined: ErrUnknownMode or ErrUnknownRegion, else nil.
func (w *World) factorKey(from, to RegionID, mode ModeID) error {
	t := w.transport
	switch {
	case t == nil || mode < 0 || int(mode) >= len(t.modes):
		return ErrUnknownMode
	case from < 0 || int(from) >= len(w.cfg.Regions) || to < 0 || int(to) >= len(w.cfg.Regions):
		return ErrUnknownRegion
	}
	return nil
}

// setTransitFactor sets the transit factor of (from, to, mode) to bp basis points
// and emits TransitFactorChanged once, if it changed (spec-transport.md 10.3). A
// factor of 0 blocks new dispatches and not those in flight. It returns
// ErrUnknownMode with the module off or for a mode not defined, ErrUnknownRegion for
// a region not defined, ErrSameRegion when the regions are the same and
// ErrOutOfBounds for a factor outside 0 to MaxTransitFactorBP; it changes nothing
// then. The public call that wraps it, and the policy actor's write, are later
// plans'.
func (w *World) setTransitFactor(from, to RegionID, mode ModeID, bp int64, cause Cause) error {
	if err := w.factorKey(from, to, mode); err != nil {
		return err
	}
	switch {
	case from == to:
		return ErrSameRegion
	case bp < 0 || bp > MaxTransitFactorBP:
		return ErrOutOfBounds
	}
	t := w.transport
	before := t.factorOf(from, to, mode)
	if before == bp {
		return nil
	}
	rec := factorRec{from: from, to: to, mode: mode, bp: bp}
	i, found := slices.BinarySearchFunc(t.factors, rec, cmpFactorKey)
	switch {
	case bp == defaultTransitBP:
		t.factors = slices.Delete(t.factors, i, i+1)
	case found:
		t.factors[i].bp = bp
	default:
		t.factors = slices.Insert(t.factors, i, rec)
	}
	if w.obs.supplied {
		w.obs.deliver(TransitFactorChanged{Tick: w.Tick(), From: from, To: to, Mode: mode, BeforeBP: before, AfterBP: bp, Cause: cause})
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "transit factor changed",
			slog.Int64(logkeys.FromRegion, int64(from)), slog.Int64(logkeys.ToRegion, int64(to)), slog.Int64(logkeys.Mode, int64(mode)),
			slog.Int64(logkeys.From, before), slog.Int64(logkeys.To, bp), slog.String(logkeys.Cause, string(cause)))
	}
	return nil
}
