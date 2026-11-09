package economysim

import (
	"context"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/transit"
)

// This file is the request and the dispatch of the transport module
// (spec-transport.md 5.3 to 5.5): the 23 ordered checks of a shipment request
// as a table of small functions over one context, and the steps that apply a
// request that passed them. The checks change nothing; a failure after them is
// a violated invariant (5.5.1).

// shipCtx is what the checks of one request read and write: the request and the
// agent, then the numbers each check settles for the ones after it.
type shipCtx struct {
	w   *World
	a   *agent
	r   *ShipmentRequest
	idx int

	t        *transportState
	md       *Mode
	res      *modeRes
	region   RegionID
	distance int64
	// k is the travel time and n the carrying units used; units is the carrying
	// units the traveller fields (self-carried).
	k, n, units int64
	// haul is the haul units redeemed, n * s.
	haul int64
	// lines is the request's lines in commodity order; the quantity of each is
	// also in t.qty by commodity until the request ends.
	lines []Line
}

// shipChecks are the checks of spec-transport.md 5.4 in the order of its table.
// Rows 3a (has_store), 6a (outside_destination), 11a (consignee_full) and 22a
// (toll_unaffordable) are the marked seams of their modules: each is an empty
// position here, filled when its module is built, and none of their reasons is
// declared yet (D446). The table is never written.
var shipChecks = [...]func(c *shipCtx) Reason{
	ckCount,       // 1
	ckMode,        // 2
	ckAway,        // 3
	nil,           // 3a has_store: STO
	ckJob,         // 4
	ckSame,        // 5
	ckRegion,      // 6
	nil,           // 6a outside_destination: EXT
	ckRoute,       // 7
	ckBlocked,     // 8
	ckTooFar,      // 9
	ckConsignee,   // 10
	ckElsewhere,   // 11
	nil,           // 11a consignee_full: STO
	ckLines,       // 12 to 16
	ckStock,       // 17
	ckCrew,        // 18
	ckCapacity,    // 19
	ckHaul,        // 20
	ckFeed,        // 21
	ckProvisioned, // 22
	nil,           // 22a toll_unaffordable: FIS
}

// ckCount is row 1: the request number MaxDispatchPerTick + 1 or later.
func ckCount(c *shipCtx) Reason {
	if c.idx >= MaxDispatchPerTick {
		return ReasonTooManyRequests
	}
	return ""
}

// ckMode is row 2: a mode of the configuration.
func ckMode(c *shipCtx) Reason {
	if c.r.Mode < 0 || int(c.r.Mode) >= len(c.t.modes) {
		return ReasonUnknownMode
	}
	c.md, c.res = &c.w.cfg.Modules.Transport.Modes[c.r.Mode], &c.t.modes[c.r.Mode]
	return ""
}

// ckAway is row 3: a self-carried request of an agent already away.
func ckAway(c *shipCtx) Reason {
	if !c.md.hired() && c.t.isAway(c.a.id) {
		return ReasonTravellerAway
	}
	return ""
}

// ckJob is row 4: a self-carried request of an agent with a job in progress,
// started this tick or earlier. A one-tick job started and finished this tick
// counts: it counts as a working tick (spec-production.md 5.2.1, E8, D447).
func ckJob(c *shipCtx) Reason {
	if c.md.hired() {
		return ""
	}
	if k, _, _ := c.w.welfare.workRead(int(c.a.id)); c.a.hasJob() || k > 0 {
		return ReasonJobInProgress
	}
	return ""
}

// ckSame is row 5: the destination is the shipper's own region.
func ckSame(c *shipCtx) Reason {
	if c.r.To == c.region {
		return ReasonSameRegion
	}
	return ""
}

// ckRegion is row 6: the destination is a region.
func ckRegion(c *shipCtx) Reason {
	if c.r.To < 0 || int(c.r.To) >= len(c.w.cfg.Regions) {
		return ReasonUnknownRegion
	}
	return ""
}

// ckRoute is row 7: the graph has a path from the origin to the destination.
func ckRoute(c *shipCtx) Reason {
	d, ok := c.w.geo.distance(c.region, c.r.To)
	if !ok {
		return ReasonNoRoute
	}
	c.distance = d
	return ""
}

// ckBlocked is row 8: the transit factor of the pair is 0.
func ckBlocked(c *shipCtx) Reason {
	if c.t.factorOf(c.region, c.r.To, c.r.Mode) == 0 {
		return ReasonBlocked
	}
	return ""
}

// ckTooFar is row 9: the travel time is within MaxTravelTicks.
func ckTooFar(c *shipCtx) Reason {
	k, ok := transit.Ticks(c.distance, c.md.Speed, c.t.factorOf(c.region, c.r.To, c.r.Mode))
	if !ok || k > MaxTravelTicks {
		return ReasonTooFar
	}
	c.k = k
	return ""
}

// ckConsignee is row 10: the consignee is an existing agent. A request to
// oneself names none.
func ckConsignee(c *shipCtx) Reason {
	if c.r.ToSelf {
		return ""
	}
	if !c.w.liveAgent(c.r.Consignee) {
		return ReasonUnknownConsignee
	}
	return ""
}

// ckElsewhere is row 11: the consignee is in the destination. A hired request to
// oneself is this reason; a self-carried one is allowed.
func ckElsewhere(c *shipCtx) Reason {
	if c.r.ToSelf {
		if c.md.hired() {
			return ReasonConsigneeElsewhere
		}
		return ""
	}
	if c.w.regionOf(&c.w.agents[c.r.Consignee]) != c.r.To {
		return ReasonConsigneeElsewhere
	}
	return ""
}

// ckLines is rows 12 to 16: the lines are consistent with the kind of request,
// at most MaxShipmentLines, each a known commodity and a whole quantity, none
// twice and none the mode's own vehicle. It leaves the lines in commodity order.
func ckLines(c *shipCtx) Reason {
	if reason := c.lineCount(); reason != "" {
		return reason
	}
	c.lines = c.t.sorted[:0]
	for _, ln := range c.r.Lines {
		if reason := c.addLine(ln); reason != "" {
			return reason
		}
	}
	for _, ln := range c.lines {
		if ln.Commodity == c.res.vehicle {
			return ReasonVehicleInCargo
		}
	}
	for i := 1; i < len(c.lines); i++ { // insertion sort: at most MaxShipmentLines
		for j := i; j > 0 && c.lines[j].Commodity < c.lines[j-1].Commodity; j-- {
			c.lines[j], c.lines[j-1] = c.lines[j-1], c.lines[j]
		}
	}
	c.t.sorted = c.lines
	return ""
}

// lineCount is rows 12 to 14: lines only where the kind of request takes them, at
// least one where it needs one, and at most MaxShipmentLines.
func (c *shipCtx) lineCount() Reason {
	r := c.r
	switch {
	case r.ToSelf && len(r.Lines) > 0:
		return ReasonLinesWithSelf
	case !r.ToSelf && len(r.Lines) == 0:
		return ReasonEmptyShipment
	case len(r.Lines) > MaxShipmentLines:
		return ReasonTooManyLines
	}
	return ""
}

// addLine checks one line and keeps it: its quantity goes to the scratch by
// commodity, which also finds a commodity named twice.
func (c *shipCtx) addLine(ln Line) Reason {
	switch {
	case ln.Commodity < 0 || int(ln.Commodity) >= len(c.w.cfg.Commodities):
		return ReasonUnknownCommodity
	case ln.Quantity < 1 || ln.Quantity > MaxQuantity:
		return ReasonInvalidQuantity
	case c.t.qty[ln.Commodity] != 0:
		return ReasonDuplicateLine
	}
	c.t.qty[ln.Commodity] = ln.Quantity
	c.lines = append(c.lines, ln)
	return ""
}

// available returns the units of a commodity the agent holds and has not earmarked.
func (c *shipCtx) available(commodity CommodityID) int64 {
	held, reserved, err := c.w.ledger.Holding(c.a.account, int(commodity))
	if err != nil {
		invariant("the ledger refused the holding of a live agent: " + err.Error())
	}
	return held - reserved
}

// ckStock is row 17: each line is within the available stock. The haul and the
// feed of the same commodity are taken from what the line leaves (rows 20, 21).
func ckStock(c *shipCtx) Reason {
	for _, ln := range c.lines {
		if int64(ln.Quantity) > c.available(ln.Commodity) {
			return ReasonInsufficientStock
		}
	}
	return ""
}

// ckCrew is row 18: a self-carried traveller fields at least one carrying unit.
func ckCrew(c *shipCtx) Reason {
	if c.md.hired() {
		return ""
	}
	if c.a.weight < c.md.Crew {
		return ReasonCrewShort
	}
	c.units = c.a.weight / c.md.Crew
	if c.res.vehicle != NoCommodity {
		v := c.available(c.res.vehicle)
		if v < 1 {
			return ReasonNoVehicle
		}
		c.units = min(c.units, v)
	}
	return ""
}

// ckCapacity is row 19: the load fits the pooled capacity, n <= units. A hired
// load is its lines; a self-carried load is everything the account holds
// (spec-transport.md 3.2.2).
func ckCapacity(c *shipCtx) Reason {
	if c.md.hired() {
		return c.hiredUnits()
	}
	return c.carriedUnits()
}

// hiredUnits settles n for a hired request: the binding fill of its lines.
func (c *shipCtx) hiredUnits() Reason {
	q := c.t.loaded
	clear(q)
	for _, ln := range c.lines {
		q[ln.Commodity] = ln.Quantity
	}
	mass, bulk, ok := load(&c.w.cfg, q)
	clear(q)
	if !ok {
		return ReasonOverflow
	}
	n, ok := transit.UnitsNeeded(mass, bulk, c.md.UnitMass, c.md.UnitBulk)
	if !ok {
		return ReasonOverflow
	}
	c.n = n
	return ""
}

// carriedUnits settles n for a self-carried request: the least n for which the
// load, less the vehicles and the feed the n units consume over k ticks, needs at
// most n units. The need falls as n rises, so the test is monotone and the least
// n is found by bisection. It refuses when n is above the units fielded.
func (c *shipCtx) carriedUnits() Reason {
	q := c.t.loaded
	for i := range q {
		held, _, err := c.w.ledger.Holding(c.a.account, i)
		if err != nil {
			invariant("the ledger refused the holding of a live agent: " + err.Error())
		}
		q[i] = Quantity(held)
	}
	if c.res.vehicle != NoCommodity {
		q[c.res.vehicle] = 0
	}
	mass, bulk, ok := load(&c.w.cfg, q)
	if !ok {
		return ReasonOverflow
	}
	full, ok := transit.UnitsNeeded(mass, bulk, c.md.UnitMass, c.md.UnitBulk)
	if !ok {
		return ReasonOverflow
	}
	lo, hi := int64(1), full
	for lo < hi {
		mid := lo + (hi-lo)/2
		if need, ok := c.needAfterFeed(q, mass, bulk, mid); ok && need <= mid {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	c.n = lo
	if c.n > c.units {
		return ReasonCapacityExceeded
	}
	return ""
}

// needAfterFeed returns the carrying units the load needs once the feed that n
// units consume over the trip is taken out of it (spec-transport.md 3.2.2): the
// quantity of each feed commodity falls by what it consumes, at most what is held.
func (c *shipCtx) needAfterFeed(q []Quantity, mass, bulk, n int64) (int64, bool) {
	for _, f := range c.res.feed {
		use := min(int64(q[f.Commodity]), c.feedUse(f, n))
		cm := &c.w.cfg.Commodities[f.Commodity]
		mass, bulk = mass-use*cm.Mass, bulk-use*cm.Bulk
	}
	return transit.UnitsNeeded(mass, bulk, c.md.UnitMass, c.md.UnitBulk)
}

// feedUse returns the units of a feed line that n carrying units consume over the
// trip, q * n * k, saturating at MaxQuantity (no account can hold more).
func (c *shipCtx) feedUse(f Line, n int64) int64 {
	a, ok := checked.Mul(int64(f.Quantity), n)
	if ok {
		a, ok = checked.Mul(a, c.k)
	}
	if !ok || a > MaxQuantity {
		return MaxQuantity
	}
	return a
}

// ckHaul is row 20: a hired shipper holds the n * s haul units, not counting a
// line of the haul commodity.
func ckHaul(c *shipCtx) Reason {
	if !c.md.hired() {
		return ""
	}
	s, ok := transit.HaulPerUnit(c.distance, c.md.HaulSpan)
	if !ok {
		return ReasonOverflow
	}
	need, ok := checked.Mul(c.n, s)
	if !ok || need > c.available(c.res.haul)-int64(c.t.qty[c.res.haul]) {
		return ReasonInsufficientHaul
	}
	c.haul = need
	return ""
}

// ckFeed is row 21: a self-carried traveller holds the feed of the trip, not
// counting a line of the same commodity.
func ckFeed(c *shipCtx) Reason {
	for _, f := range c.res.feed {
		if c.feedUse(f, c.n) > c.available(f.Commodity)-int64(c.t.qty[f.Commodity]) {
			return ReasonInsufficientFeed
		}
	}
	return ""
}

// ckProvisioned is row 22: after the moves of the dispatch a self-carried
// traveller still holds the upkeep of its households for the k ticks away, because
// it cannot buy while away. A hired request of a shipper already away on a trip of
// the same plan keeps that guarantee for the ticks the trip has left.
func ckProvisioned(c *shipCtx) Reason {
	ticks := c.k
	if c.md.hired() {
		trip := c.t.tripOf(c.a.id)
		if trip == nil {
			return ""
		}
		ticks = trip.arrives - c.w.Tick()
	}
	for _, u := range c.w.roles.row(c.a.role).upkeepIn(c.region) {
		left := c.available(u.commodity) - int64(c.t.qty[u.commodity])
		for _, f := range c.res.feed {
			if f.Commodity == u.commodity {
				left -= c.feedUse(f, c.n)
			}
		}
		if c.md.hired() && u.commodity == c.res.haul {
			left -= c.haul
		}
		need, ok := checked.Mul(int64(u.quantity), c.a.weight)
		if ok {
			need, ok = checked.Mul(need, ticks)
		}
		if !ok || left < need {
			return ReasonUnprovisioned
		}
	}
	return ""
}

// request runs the checks in order and returns the first reason that holds, and
// the context with the numbers the dispatch needs. It changes nothing in the
// world; its scratch is cleared before it returns. The context is held by the
// state, so that the calls through the table of checks allocate nothing.
func (w *World) request(a *agent, idx int, r *ShipmentRequest, region RegionID) (*shipCtx, Reason) {
	t := w.transport
	c := &t.ctx
	*c = shipCtx{w: w, a: a, r: r, idx: idx, t: t, region: region}
	reason := c.run()
	for _, ln := range r.Lines {
		if ln.Commodity >= 0 && int(ln.Commodity) < len(t.qty) {
			t.qty[ln.Commodity] = 0
		}
	}
	return c, reason
}

// run applies the checks in order and returns the first reason that holds.
func (c *shipCtx) run() Reason {
	for _, check := range shipChecks {
		if check == nil {
			continue
		}
		if reason := check(c); reason != "" {
			return reason
		}
	}
	return ""
}

// executeShipments executes the shipment requests of a checked plan, in plan
// order, after its discards and before its orders (spec-world.md 6.5 P3.2(c),
// spec-transport.md 5.3.2). With the module off every request is refused as
// unknown_mode and only logged: no event of an absent module is emitted.
func (w *World) executeShipments(a *agent, p *TradePlan) {
	if len(p.Shipments) == 0 {
		return
	}
	region := w.regionOf(a)
	for i := range p.Shipments {
		w.dispatch(a, i, &p.Shipments[i], region)
	}
}

// dispatch checks one request and, if it passes, dispatches it: the steps of
// spec-transport.md 5.5.1 in order. It returns the shipment ID, or the reason the
// request was refused; a refusal leaves no trace (6.4). It emits ShipmentRejected
// or ShipmentDispatched when an observer was supplied, and logs at Debug.
func (w *World) dispatch(a *agent, idx int, r *ShipmentRequest, region RegionID) (ShipmentID, Reason) {
	if w.transport == nil {
		w.shipmentRefused(a, r, ReasonUnknownMode)
		return 0, ReasonUnknownMode
	}
	c, reason := w.request(a, idx, r, region)
	if reason != "" {
		if w.obs.supplied {
			w.obs.deliver(ShipmentRejected{Tick: w.Tick(), Region: region, Agent: a.id, Mode: r.Mode, To: r.To, Reason: reason})
		}
		w.shipmentRefused(a, r, reason)
		return 0, reason
	}
	s := w.send(c)
	if w.obs.supplied {
		w.deliverDispatched(&s)
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "shipment dispatched",
			slog.Int64(logkeys.Shipment, int64(s.id)), slog.Int64(logkeys.Mode, int64(s.mode)),
			slog.Int64(logkeys.Region, int64(s.from)), slog.Int64(logkeys.Dest, int64(s.to)), slog.Int64(logkeys.Agent, int64(s.shipper)))
	}
	return s.id, ""
}

// shipmentRefused logs the Debug line of a refused request.
func (w *World) shipmentRefused(a *agent, r *ShipmentRequest, reason Reason) {
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "shipment refused",
			slog.Int64(logkeys.Agent, int64(a.id)), slog.Int64(logkeys.Mode, int64(r.Mode)), slog.String(logkeys.Reason, string(reason)))
	}
}

// send applies a request that passed the checks: the steps 1 to 5 of
// spec-transport.md 5.5.1. The seams of the toll (3a) and the risk (4a) are empty
// positions here, each filled by its module. A refusal of the ledger below is a
// violated invariant: every move was checked.
func (w *World) send(c *shipCtx) shipment {
	t, a := c.t, c.a
	s := shipment{
		id: t.next, mode: c.r.Mode, from: c.region, to: c.r.To, shipper: a.id, consignee: c.r.Consignee,
		hired: c.md.hired(), dispatched: w.Tick(), arrives: w.Tick() + c.k, account: -1, units: c.n, haulUnits: c.haul,
	}
	if !s.hired {
		s.traveller = a.id
		if c.r.ToSelf {
			s.consignee = a.id
		}
	}
	if len(c.lines) > 0 {
		s.account = w.transitAccount(c.region)
		s.lines = append(t.lend(), c.lines...)
		w.moveLines(a, &s)
	}
	if s.hired {
		w.redeemHaul(a, c)
	} else {
		w.spendTrip(a, c)
	}
	// 3a: the toll (FIS). 4a: the risk (RSK).
	t.next++
	t.ships = append(t.ships, s)
	t.due.Push(s.arrives, int64(s.id))
	if !s.hired {
		t.setAway(a.id, true)
	}
	return s
}

// transitAccount takes the open Transit account of the region with the lowest ID
// that no shipment in flight uses, or opens a new one (5.5.1 step 1). Accounts
// are reused and never closed (7.1).
func (w *World) transitAccount(region RegionID) int {
	if id, ok := w.transport.free[region].Take(); ok {
		return id
	}
	id, err := w.ledger.Open(ledger.KindTransit, 0, int(region))
	if err != nil {
		invariant("the ledger refused a Transit account: " + err.Error())
	}
	return id
}

// moveLines moves each line, in commodity order, from the shipper to the Transit
// account under transit_out (step 2). The lots keep their born tick and
// durability.
func (w *World) moveLines(a *agent, s *shipment) {
	for _, ln := range s.lines {
		if err := w.ledger.Move(a.account, s.account, int(ln.Commodity), int64(ln.Quantity), ledger.TransitOut); err != nil {
			invariant("the ledger refused a checked shipment line: " + err.Error())
		}
	}
}

// redeemHaul moves the haul units of a hired shipment to the boundary under
// haul_use (step 3).
func (w *World) redeemHaul(a *agent, c *shipCtx) {
	sink, ok := w.ledger.Boundary(ledger.HaulUse, int(c.res.haul))
	if !ok {
		invariant("the haul commodity has no haul_use boundary account")
	}
	if err := w.ledger.Move(a.account, sink, int(c.res.haul), c.haul, ledger.HaulUse); err != nil {
		invariant("the ledger refused checked haul units: " + err.Error())
	}
}

// spendTrip charges a self-carried trip (step 3): the wear of the n vehicles for
// the whole trip, then the feed to the boundary under transit_use. A vehicle that
// reaches 0 is destroyed and the trip stands.
func (w *World) spendTrip(a *agent, c *shipCtx) {
	if c.res.vehicle != NoCommodity {
		if err := w.ledger.WearItems(a.account, int(c.res.vehicle), c.n, c.md.WearPerTick*c.k); err != nil {
			invariant("the ledger refused the checked wear of a vehicle: " + err.Error())
		}
	}
	for _, f := range c.res.feed {
		sink, ok := w.ledger.Boundary(ledger.TransitUse, int(f.Commodity))
		if !ok {
			invariant("a feed commodity has no transit_use boundary account")
		}
		if err := w.ledger.Move(a.account, sink, int(f.Commodity), c.feedUse(f, c.n), ledger.TransitUse); err != nil {
			invariant("the ledger refused checked feed: " + err.Error())
		}
	}
}

// deliverDispatched builds and delivers ShipmentDispatched; the lines are a copy.
func (w *World) deliverDispatched(s *shipment) {
	lines := make([]Amount, len(s.lines))
	for i, ln := range s.lines {
		lines[i] = Amount(ln)
	}
	w.obs.deliver(ShipmentDispatched{
		Tick: s.dispatched, Shipment: s.id, Mode: s.mode, Region: s.from, To: s.to,
		Shipper: s.shipper, Traveller: s.traveller, Consignee: s.consignee, Hired: s.hired,
		Arrives: s.arrives, Units: s.units, HaulUnits: s.haulUnits, Lines: lines,
	})
}
