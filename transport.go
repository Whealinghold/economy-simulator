package economysim

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/transit"
)

// This file is the transport module (spec-transport.md): modes of carriage, the
// shipments that go on them and the goods in transit. The pure rules are in
// internal/transit. The file holds, in the order of architecture.md 2.10.1: the
// configuration surface (the canonical form, the validator and the warnings) and
// the ledger reasons the module declares. The request and the dispatch are in
// transport_request.go, the arrival in transport_arrival.go, the view in
// transport_view.go, the snapshot rows and the checkpoint section in
// transport_state.go and the lanes in transport_lanes.go.

// shipment is the record of one journey in flight (spec-transport.md 5.2). Plain
// data; the lines are a buffer the state lends it and takes back at the arrival.
type shipment struct {
	id                  ShipmentID
	mode                ModeID
	from, to            RegionID
	shipper, traveller  AgentID
	consignee           AgentID
	hired               bool
	dispatched, arrives int64
	account             int // the Transit ledger account; -1 for a trip to oneself
	units, haulUnits    int64
	lines               []Line
}

// modeRes is a mode with its commodities resolved to typed IDs, once, at
// construction (the canonical configuration holds strings).
type modeRes struct {
	vehicle, haul CommodityID // NoCommodity when the mode has none
	feed          []Line
}

// transportState is the state of the transport module, nil when it is off
// (spec-transport.md 14): the next shipment ID, the shipments in flight in ID
// order, and the transit factors. The away flags, the free lists of the Transit
// accounts and the scratch are derived and never saved.
type transportState struct {
	next    ShipmentID
	ships   []shipment
	factors []factorRec
	modes   []modeRes
	// free is, per origin region, the open Transit accounts no shipment uses.
	free []transit.Free
	// away is, by agent ID, whether the agent is the traveller of a live
	// self-carried shipment.
	away []bool
	// pool lends the line buffers of the shipments; qty and held are the scratch
	// of one request, by commodity, and sorted the lines of the request in
	// commodity order.
	pool   [][]Line
	qty    []Quantity
	sorted []Line
	loaded []Quantity
	// ctx is the context of the request being checked.
	ctx shipCtx
	// tables are what P3.1 fixes for the views of the tick.
	tables viewTables
	// arrived and arrivedLines are the arrivals of P1.3, kept for the later positions
	// with or without an observer; key and skip build the guard query of an arrival.
	arrived      []arrival
	arrivedLines []ArrivedLine
	key          []byte
	skip         []string
}

// factorRec is one transit factor that differs from 10 000 (spec-transport.md
// 10.3).
type factorRec struct {
	from, to RegionID
	mode     ModeID
	bp       int64
}

// defaultTransitBP is the transit factor of a pair no one has set.
const defaultTransitBP = 10_000

// newTransport builds the state of the module for a canonical configuration, nil
// when the module is off.
func newTransport(c *Config) *transportState {
	m := c.Modules.Transport
	if m == nil {
		return nil
	}
	t := &transportState{
		modes:  make([]modeRes, len(m.Modes)),
		free:   make([]transit.Free, len(c.Regions)),
		qty:    make([]Quantity, len(c.Commodities)),
		loaded: make([]Quantity, len(c.Commodities)),
		sorted: make([]Line, 0, MaxShipmentLines),
	}
	for i := range m.Modes {
		md := &m.Modes[i]
		r := modeRes{vehicle: NoCommodity, haul: NoCommodity}
		if id, ok := c.commodityID(md.Vehicle); ok {
			r.vehicle = id
		}
		if id, ok := c.commodityID(md.Haul); ok {
			r.haul = id
		}
		for _, f := range md.Feed {
			id, _ := c.commodityID(f.Commodity)
			r.feed = append(r.feed, Line{Commodity: id, Quantity: f.Quantity})
		}
		t.modes[i] = r
	}
	return t
}

// isAway reports whether the agent is the traveller of a live self-carried
// shipment.
func (t *transportState) isAway(id AgentID) bool {
	return int(id) < len(t.away) && t.away[id]
}

// setAway sets the away flag of an agent, growing the table as agents join.
func (t *transportState) setAway(id AgentID, away bool) {
	for int(id) >= len(t.away) {
		t.away = append(t.away, false)
	}
	t.away[id] = away
}

// factorOf returns the transit factor of (from, to, mode) in basis points: the
// slot, or 10 000 for a pair no one has set (spec-transport.md 10.3). The slot
// is sorted by (from, to, mode), so the search is binary.
func (t *transportState) factorOf(from, to RegionID, mode ModeID) int64 {
	i, ok := slices.BinarySearchFunc(t.factors, factorRec{from: from, to: to, mode: mode}, cmpFactorKey)
	if !ok {
		return defaultTransitBP
	}
	return t.factors[i].bp
}

// cmpFactorKey orders factor records by (from, to, mode).
func cmpFactorKey(a, b factorRec) int {
	return cmp.Or(cmp.Compare(a.from, b.from), cmp.Compare(a.to, b.to), cmp.Compare(a.mode, b.mode))
}

// lend returns an empty line buffer, from the pool when one is there.
func (t *transportState) lend() []Line {
	if n := len(t.pool); n > 0 {
		b := t.pool[n-1]
		t.pool = t.pool[:n-1]
		return b[:0]
	}
	return make([]Line, 0, MaxShipmentLines)
}

// take returns a line buffer to the pool.
func (t *transportState) take(b []Line) {
	if cap(b) > 0 {
		t.pool = append(t.pool, b[:0])
	}
}

// transportReasons is the ledger reasons the module declares, in the order of
// their rows in the table of spec-ledger.md 7.2. Only the two sinks open a
// boundary account (spec-world.md 5.2 row 3a).
func transportReasons() []ledger.Reason {
	return []ledger.Reason{ledger.TransitOut, ledger.TransitIn, ledger.HaulUse, ledger.TransitUse}
}

// canonical returns a copy of the module with the modes in ID order and the feed
// of each in commodity order (spec-config.md 3.4). A nil module stays nil.
func (t *Transport) canonical() *Transport {
	if t == nil {
		return nil
	}
	out := *t
	out.Modes = sortedBy(t.Modes, func(a, b Mode) int { return cmp.Compare(a.ID, b.ID) })
	for i := range out.Modes {
		out.Modes[i].Feed = sortedBy(out.Modes[i].Feed, byCommodity)
	}
	return &out
}

// hired reports whether the mode is hired: it has a haul commodity.
func (m *Mode) hired() bool { return m.Haul != "" }

// validateTransport checks the transport module by spec-transport.md 12.1, in
// the order of its table: the bounds, the duplicates, the references, the
// vehicle and the shape. It reports nothing when the module is off. The region
// graph is always built, so the rule "the module on while the region graph is
// off" has no case (ErrModuleDependency is never returned by this module).
func (v *validator) validateTransport() {
	m := v.c.Modules.Transport
	if m == nil {
		return
	}
	base := []string{"modules", "transport"}
	if len(m.Modes) > MaxModes {
		v.rep.add(ErrOutOfBounds, path(base, "modes"), itoa(len(m.Modes)), itoa(MaxModes))
		return // the rest is linear in the count it just refused
	}
	seen := make(map[string]bool, len(m.Modes))
	for i := range m.Modes {
		md := &m.Modes[i]
		mb := path(base, "mode", md.ID)
		if !validID(md.ID) {
			v.rep.add(ErrInvalidID, path(mb, "id"), md.ID, "")
		}
		if seen[md.ID] {
			v.rep.add(ErrDuplicateID, path(mb, "id"), md.ID, "")
		}
		seen[md.ID] = true
		v.modeBounds(mb, md)
		v.modeRefs(mb, md)
		v.modeShape(mb, md)
		v.label(mb, "label", md.Label)
	}
}

// modeBounds checks the numbers of one mode against section 11.
func (v *validator) modeBounds(mb []string, md *Mode) {
	v.inRange(path(mb, "speed"), md.Speed, 1, MaxSpeed)
	massOK := v.inRange(path(mb, "unit_mass"), md.UnitMass, 0, MaxCapacity)
	bulkOK := v.inRange(path(mb, "unit_bulk"), md.UnitBulk, 0, MaxCapacity)
	if massOK && bulkOK && md.UnitMass == 0 && md.UnitBulk == 0 {
		v.rep.add(ErrOutOfBounds, path(mb, "unit_mass"), "0", "1")
	}
	if !md.hired() {
		v.inRange(path(mb, "crew"), md.Crew, 1, MaxWeight)
	}
	if md.HaulSpan < 0 || md.HaulSpan > MaxHaulSpan {
		v.inRange(path(mb, "haul_span"), md.HaulSpan, 1, MaxHaulSpan)
	}
	if md.WearPerTick < 0 {
		v.rep.add(ErrOutOfBounds, path(mb, "wear_per_tick"), i64(md.WearPerTick), "1")
	}
	v.feedBounds(mb, md.Feed)
}

// feedBounds checks the length of a feed and the quantity of each entry.
func (v *validator) feedBounds(mb []string, feed []Item) {
	if len(feed) > MaxFeed {
		v.rep.add(ErrOutOfBounds, path(mb, "feed"), itoa(len(feed)), itoa(MaxFeed))
	}
	for j, it := range feed {
		v.inRange(path(mb, "feed", itoa(j), "quantity"), int64(it.Quantity), 1, MaxQuantity)
	}
}

// modeRefs checks the commodities one mode names: each defined, no feed twice,
// a vehicle with a durability that its wear does not pass.
func (v *validator) modeRefs(mb []string, md *Mode) {
	seen := make(map[string]bool, len(md.Feed))
	for j, it := range md.Feed {
		fb := path(mb, "feed", itoa(j), "commodity")
		if v.lk.commodity[it.Commodity] == nil {
			v.rep.add(ErrUnknownCommodity, fb, it.Commodity, "")
		}
		if seen[it.Commodity] {
			v.rep.add(ErrDuplicateID, fb, it.Commodity, "")
		}
		seen[it.Commodity] = true
	}
	if md.Haul != "" && v.lk.commodity[md.Haul] == nil {
		v.rep.add(ErrUnknownCommodity, path(mb, "haul"), md.Haul, "")
	}
	if md.Vehicle == "" {
		return
	}
	vc := v.lk.commodity[md.Vehicle]
	switch {
	case vc == nil:
		v.rep.add(ErrUnknownCommodity, path(mb, "vehicle"), md.Vehicle, "")
	case vc.Durability == 0:
		v.rep.add(ErrVehicleNotDurable, path(mb, "vehicle"), md.Vehicle, "")
	case md.WearPerTick > vc.Durability:
		v.rep.add(ErrOutOfBounds, path(mb, "wear_per_tick"), i64(md.WearPerTick), i64(vc.Durability))
	}
}

// modeShape checks that a mode is exactly self-carried or hired
// (spec-transport.md 3.1): a hired mode has no crew, vehicle, wear or feed and
// has a haul span; a self-carried mode has no haul span; a wear needs a vehicle
// and a vehicle a wear.
func (v *validator) modeShape(mb []string, md *Mode) {
	bad := func(field, value string) { v.rep.add(ErrModeShape, path(mb, field), value, "") }
	switch {
	case md.hired():
		hiredShape(md, bad)
	case md.HaulSpan != 0:
		bad("haul_span", i64(md.HaulSpan))
	}
	if !md.hired() && md.Vehicle == "" && md.WearPerTick != 0 {
		bad("wear_per_tick", i64(md.WearPerTick))
	}
	if !md.hired() && md.Vehicle != "" && md.WearPerTick == 0 {
		bad("wear_per_tick", "0")
	}
}

// hiredShape reports the fields a hired mode must not set and the one it must.
func hiredShape(md *Mode, bad func(field, value string)) {
	if md.Crew != 0 {
		bad("crew", i64(md.Crew))
	}
	if md.Vehicle != "" {
		bad("vehicle", md.Vehicle)
	}
	if md.WearPerTick != 0 {
		bad("wear_per_tick", i64(md.WearPerTick))
	}
	if len(md.Feed) > 0 {
		bad("feed", itoa(len(md.Feed)))
	}
	if md.HaulSpan == 0 {
		bad("haul_span", "0")
	}
}

// transportWarnings returns the warnings of spec-transport.md 12.2, in no
// particular order. They read a configuration that may not validate, so each
// guards its lookups; the ones that read travel times need the region graph and
// are left out when it cannot be built.
func transportWarnings(c *Config, lk lookup) []Warning {
	m := c.Modules.Transport
	if m == nil {
		return nil
	}
	var out []Warning
	add := func(rule string, segs ...string) { out = append(out, Warning{Path: segs, Rule: rule}) }
	base := []string{"modules", "transport"}
	if len(c.Regions) < 2 || len(m.Modes) == 0 {
		add("transport_without_routes", base...)
	}
	made := map[string]bool{}
	for i := range c.Recipes {
		for _, o := range c.Recipes[i].Outputs {
			made[o.Commodity] = true
		}
	}
	for i := range m.Modes {
		out = append(out, haulWarnings(&m.Modes[i], lk, made, base)...)
	}
	g, err := NewGraph(*c)
	if err != nil || len(m.Modes) > MaxModes {
		return out
	}
	return append(out, tripWarnings(c, m, lk, g, base)...)
}

// haulWarnings raises haul_unsourced, haul_never_spoils and haul_durable for one
// hired mode (spec-transport.md 12.2).
func haulWarnings(md *Mode, lk lookup, made map[string]bool, base []string) []Warning {
	hc := lk.commodity[md.Haul]
	if !md.hired() || hc == nil {
		return nil
	}
	hb := path(base, "mode", md.ID, "haul")
	var out []Warning
	if !made[md.Haul] && !hc.Raw {
		out = append(out, Warning{Path: hb, Rule: "haul_unsourced"})
	}
	if hc.ShelfLife == 0 {
		out = append(out, Warning{Path: hb, Rule: "haul_never_spoils"})
	}
	if hc.Durability > 0 {
		out = append(out, Warning{Path: hb, Rule: "haul_durable"})
	}
	return out
}

// tripWarnings raises the three warnings that read the travel times over the
// graph: vehicle_wears_in_one_trip, trip_longer_than_window and
// route_too_slow_for_shelf_life (spec-transport.md 12.2). A mode with a speed
// outside its bound has no travel time and is skipped.
func tripWarnings(c *Config, m *Transport, lk lookup, g Graph, base []string) []Warning {
	var out []Warning
	for i := range m.Modes {
		md := &m.Modes[i]
		if md.Speed < 1 || md.Speed > MaxSpeed {
			continue
		}
		minK, maxK, any := tripExtremes(g, md.Speed)
		if any {
			out = append(out, modeTripWarnings(c, md, lk, path(base, "mode", md.ID), minK, maxK)...)
		}
	}
	return out
}

// modeTripWarnings raises the trip warnings of one mode, given the shortest and
// the longest travel time over the graph.
func modeTripWarnings(c *Config, md *Mode, lk lookup, mb []string, minK, maxK int64) []Warning {
	var out []Warning
	if vc := lk.commodity[md.Vehicle]; vc != nil && md.WearPerTick > 0 {
		if w, ok := checked.Mul(md.WearPerTick, maxK); ok && w > vc.Durability {
			out = append(out, Warning{Path: path(mb, "wear_per_tick"), Rule: "vehicle_wears_in_one_trip"})
		}
	}
	if !md.hired() && maxK > c.World.WelfareWindow {
		out = append(out, Warning{Path: mb, Rule: "trip_longer_than_window"})
	}
	for _, cm := range c.Commodities {
		if cm.ShelfLife > 0 && minK >= cm.ShelfLife {
			out = append(out, Warning{Path: path(mb, "shelf_life", cm.ID), Rule: "route_too_slow_for_shelf_life"})
		}
	}
	return out
}

// tripExtremes returns the shortest and the longest travel time, at the default
// factor, over the ordered pairs of regions that have a path, and whether there
// is one.
func tripExtremes(g Graph, speed int64) (minK, maxK int64, any bool) {
	for from := range RegionID(g.Len()) {
		for to := range RegionID(g.Len()) {
			d, ok := g.Distance(from, to)
			if !ok || from == to {
				continue
			}
			k, ok := transit.Ticks(d, speed, 10_000)
			if !ok {
				continue
			}
			if !any {
				minK, maxK, any = k, k, true
			}
			minK, maxK = min(minK, k), max(maxK, k)
		}
	}
	return minK, maxK, any
}

// modeNames returns the string ID of each mode, in ModeID order; nil with the
// module off.
func modeNames(c *Config) []string {
	m := c.Modules.Transport
	if m == nil {
		return nil
	}
	out := make([]string, len(m.Modes))
	for i := range m.Modes {
		out[i] = m.Modes[i].ID
	}
	return out
}
