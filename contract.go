package economysim

import (
	"cmp"
	"context"
	"log/slog"
	"maps"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/merchant"
	"github.com/Whealinghold/economy-sim/internal/transit"
)

// This file is the merchants module (spec-merchants.md, MER): the delivery
// contract and the data of a merchant role. It holds the validation of the
// merchant block and of the module, the derived sets of a merchant role, the
// warnings of the configuration, the round and the escrow; the settlement is
// contract_settle.go. The arithmetic is internal/merchant;
// this file converts at the call (architecture.md 2.9, 2.10.1).

// merchantStrategy is the registered name of the merchant strategy
// (spec-merchants.md 3.1.1): a role that names it must carry the block.
const merchantStrategy = "merchant"

// merchantMaxFeeSpread is the largest FeeShareSpreadBP (spec-merchants.md 3.2).
const merchantMaxFeeSpread = 5_000

// modeIndex returns the modes of the transport module by ID, nil when it is off.
func modeIndex(c *Config) map[string]*Mode {
	t := c.Modules.Transport
	if t == nil {
		return nil
	}
	out := make(map[string]*Mode, len(t.Modes))
	for i := range t.Modes {
		out[t.Modes[i].ID] = &t.Modes[i]
	}
	return out
}

// merchantTripTicks is Kmax of FR-WEL-009 (spec-merchants.md 12.3): the longest travel
// time, at the factor 10 000, over the region graph and the self-carried modes of the
// role's merchant block, capped at MaxTravelTicks so that WindowCycles times it cannot
// overflow and still passes MaxWelfareWindow to be refused there. A hired mode is not the
// merchant's own trip, a mode with a crew above 1 is one it cannot carry itself
// (FR-MER-003), and a speed outside its bound has no travel time (tripWarnings does the
// same). It is 0 for a role with no block, with the transport module off, with no path or
// when the graph does not build: the rules that blame those causes run first.
func merchantTripTicks(c *Config, r *Role) int64 {
	if r.Merchant == nil || c.Modules.Transport == nil {
		return 0
	}
	g, err := NewGraph(*c)
	if err != nil {
		return 0
	}
	var longest int64
	for i := range c.Modules.Transport.Modes {
		md := &c.Modules.Transport.Modes[i]
		if md.hired() || md.Crew > 1 || md.Speed < 1 || md.Speed > MaxSpeed || !slices.Contains(r.Merchant.Modes, md.ID) {
			continue
		}
		if _, k, any := tripExtremes(g, md.Speed); any {
			longest = max(longest, k)
		}
	}
	return min(longest, MaxTravelTicks)
}

// merchantTradeInputs returns the trade inputs of spec-merchants.md 3.2.2: the
// haul commodity of each hired mode of the block and the vehicle and the feed
// commodities of each self-carried mode. A mode that is not defined adds nothing.
func merchantTradeInputs(m *Merchant, modes map[string]*Mode) map[string]bool {
	in := map[string]bool{}
	for _, id := range m.Modes {
		md := modes[id]
		switch {
		case md == nil:
		case md.hired():
			in[md.Haul] = true
		default:
			if md.Vehicle != "" {
				in[md.Vehicle] = true
			}
			for _, f := range md.Feed {
				in[f.Commodity] = true
			}
		}
	}
	return in
}

// merchantCargo returns the cargo set of a merchant role in ascending order: its
// Trades less the trade inputs (spec-merchants.md 3.2.2).
func merchantCargo(r *Role, inputs map[string]bool) []string {
	var out []string
	for _, id := range r.Trades {
		if !inputs[id] && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

// hasHiredMode reports whether the block lists a hired mode of modes.
func (m *Merchant) hasHiredMode(modes map[string]*Mode) bool {
	for _, id := range m.Modes {
		if md := modes[id]; md != nil && md.hired() {
			return true
		}
	}
	return false
}

// validateMerchants is the module's part of phases 2 and 3 of spec-config.md
// 7.1.6 (spec-merchants.md 14.1): the bounds of the module and of each block, the
// module dependencies, the shape of a merchant role and its weight. It reports
// nothing about a role that has no block beyond MerchantFeeShare and a missing
// block under the merchant strategy.
func (v *validator) validateMerchants() {
	c := &v.c
	if m := c.Modules.Merchants; m != nil {
		base := []string{"modules", "merchants"}
		if c.Modules.Transport == nil {
			v.rep.add(ErrModuleDependency, base, "transport", "")
		}
		v.inRange(path(base, "max_open"), m.MaxOpen, 1, MaxContracts)
	}
	modes := modeIndex(c)
	for i := range c.Roles {
		r := &c.Roles[i]
		rb := []string{"roles", r.ID}
		v.inRange(path(rb, "merchant_fee_share"), r.MerchantFeeShare, 0, wholeBP)
		switch {
		case r.Merchant != nil:
			v.merchantRole(rb, r, modes)
		case r.Strategy == merchantStrategy:
			v.rep.add(ErrMerchantShape, path(rb, "merchant"), "missing", "")
		}
	}
}

// merchantRole checks the block of one role: the module dependencies, the shape
// of the role, the weight of its groups, the modes, the cargo set and the
// bounds of each field.
func (v *validator) merchantRole(rb []string, r *Role, modes map[string]*Mode) {
	c := &v.c
	b := r.Merchant
	mb := path(rb, "merchant")
	if c.Modules.Transport == nil {
		v.rep.add(ErrModuleDependency, mb, "transport", "")
	}
	if c.Modules.Information == nil {
		v.rep.add(ErrModuleDependency, mb, "information", "")
	}
	if len(r.Recipes) > 0 {
		v.rep.add(ErrMerchantShape, path(rb, "recipes"), itoa(len(r.Recipes)), "0")
	}
	if !r.Informed {
		v.rep.add(ErrMerchantShape, path(rb, "informed"), "false", "true")
	}
	v.merchantWeight(r)
	v.merchantModes(mb, b, modes)
	if c.Modules.Transport != nil {
		v.merchantCargoSet(rb, r, modes)
	}
	v.merchantBounds(mb, r, b)
	if b.Consigns && c.Modules.Merchants == nil {
		v.rep.add(ErrModuleDependency, path(mb, "consigns"), "true", "")
	}
}

// merchantWeight reports a group of the role whose agents weigh more than one
// household (spec-merchants.md 3.6, 14.1).
func (v *validator) merchantWeight(r *Role) {
	c := &v.c
	for _, g := range c.Groups {
		if g.Role != r.ID {
			continue
		}
		_, w, _, ok := groupShape(c.World, g)
		if scaleOf(c.World, g) > 1 || (ok && w > 1) {
			v.rep.add(ErrMerchantWeight, []string{"groups", g.Region, g.Role, "scale"}, i64(scaleOf(c.World, g)), "1")
		}
	}
}

// merchantModes checks the list of modes of the block: not empty, at most
// MaxModes, each defined, none twice, a self-carried one with a crew of 1 and a
// hired one only with the module on (spec-merchants.md 14.1). With the transport
// module off there is no mode to look up; the dependency is already reported.
func (v *validator) merchantModes(mb []string, b *Merchant, modes map[string]*Mode) {
	p := path(mb, "modes")
	if n := len(b.Modes); n < 1 || n > MaxModes {
		v.rep.add(ErrOutOfBounds, p, itoa(n), itoa(MaxModes))
		if n > MaxModes {
			return // the rest is linear in the count it just refused
		}
	}
	if v.c.Modules.Transport == nil {
		return
	}
	seen := make(map[string]bool, len(b.Modes))
	for _, id := range b.Modes {
		v.merchantMode(p, id, modes[id], seen[id])
		seen[id] = true
	}
}

// merchantMode checks one entry of the list of modes: md is the mode, nil when it
// is not defined, and again says it was listed before.
func (v *validator) merchantMode(p []string, id string, md *Mode, again bool) {
	switch {
	case md == nil:
		v.rep.add(ErrUnknownMode, p, id, "")
	case again:
		v.rep.add(ErrDuplicateID, p, id, "")
	case !md.hired() && md.Crew > 1:
		v.rep.add(ErrMerchantShape, path(p, id, "crew"), i64(md.Crew), "1")
	case md.hired() && v.c.Modules.Merchants == nil:
		v.rep.add(ErrModuleDependency, p, id, "")
	}
}

// merchantCargoSet checks that every trade input is in Trades and that the cargo
// set is not empty (spec-merchants.md 3.2.2, 14.1).
func (v *validator) merchantCargoSet(rb []string, r *Role, modes map[string]*Mode) {
	inputs := merchantTradeInputs(r.Merchant, modes)
	trades := make(map[string]bool, len(r.Trades))
	for _, id := range r.Trades {
		trades[id] = true
	}
	for _, id := range slices.Sorted(maps.Keys(inputs)) {
		if !trades[id] {
			v.rep.add(ErrMerchantShape, path(rb, "trades"), id, "")
		}
	}
	if len(merchantCargo(r, inputs)) == 0 {
		v.rep.add(ErrMerchantShape, path(rb, "trades"), "no cargo commodity", "")
	}
}

// merchantBounds checks the numbers of the block against spec-merchants.md 13.
func (v *validator) merchantBounds(mb []string, r *Role, b *Merchant) {
	v.inRange(path(mb, "lot_units"), b.LotUnits, 1, MaxLotUnits)
	v.merchantSpread(mb, r, b)
	v.inRange(path(mb, "max_source_age"), b.MaxSourceAge, 0, MaxRumourAge)
	v.inRange(path(mb, "time_cost_bp"), b.TimeCostBP, 0, MaxTimeCostBP)
	v.inRange(path(mb, "consignee_cut_bp"), b.ConsigneeCutBP, 0, 5_000)
	v.inRange(path(mb, "take_margin_bp"), b.TakeMarginBP, 0, 9_000)
	if b.Consigns && b.TakeUnits == 0 {
		v.rep.add(ErrMerchantShape, path(mb, "take_units"), "0", "1")
		return
	}
	v.inRange(path(mb, "take_units"), int64(b.TakeUnits), b2i(b.Consigns), MaxQuantity)
}

// merchantSpread checks FeeShareSpreadBP against its bound, the role's fee share
// and the ceiling of 10 000 (spec-merchants.md 3.2, 14.1).
func (v *validator) merchantSpread(mb []string, r *Role, b *Merchant) {
	p := path(mb, "fee_share_spread_bp")
	if !v.inRange(p, b.FeeShareSpreadBP, 0, merchantMaxFeeSpread) || r.MerchantFeeShare < 0 || r.MerchantFeeShare > wholeBP {
		return // the share has its own error, and the comparisons below read a share in range
	}
	switch {
	case b.FeeShareSpreadBP > r.MerchantFeeShare:
		v.rep.add(ErrOutOfBounds, p, i64(b.FeeShareSpreadBP), i64(r.MerchantFeeShare))
	case r.MerchantFeeShare+b.FeeShareSpreadBP > wholeBP:
		v.rep.add(ErrOutOfBounds, p, i64(b.FeeShareSpreadBP), i64(wholeBP-r.MerchantFeeShare))
	}
}

// merchantAgents returns the agents of a role in the whole world (spec-scale.md
// 3.2), from the groups; a group outside its bounds counts none.
func merchantAgents(c *Config, r *Role) int64 {
	n := int64(0)
	for _, g := range c.Groups {
		if g.Role != r.ID {
			continue
		}
		if a, _, _, ok := groupShape(c.World, g); ok {
			n += a
		}
	}
	return n
}

// merchantWarnings returns the warnings of spec-merchants.md 14.2 that read the
// configuration alone; lane_without_consignee, cargo_never_delivered and
// consigns_without_cargo need the shortest routes and are merchantRouteWarnings.
// They read a configuration that may not validate, so each guards its inputs.
func merchantWarnings(c *Config) []Warning {
	modes := modeIndex(c)
	consigns := false
	for i := range c.Roles {
		consigns = consigns || (c.Roles[i].Merchant != nil && c.Roles[i].Merchant.Consigns)
	}
	var out []Warning
	for i := range c.Roles {
		r := &c.Roles[i]
		if r.Merchant == nil {
			continue
		}
		rb := []string{"roles", r.ID}
		out = append(out, merchantShareWarnings(rb, r)...)
		if !consigns && r.Merchant.hasHiredMode(modes) {
			out = append(out, Warning{Path: path(rb, "merchant", "modes"), Rule: "merchant_no_consignee"})
		}
		if merchantAgents(c, r) == 1 {
			out = append(out, Warning{Path: rb, Rule: "merchants_one_agent"})
		}
	}
	return out
}

// merchantShareWarnings returns merchant_spread_narrow, merchant_fee_share_high
// and merchant_fee_share_low for one role (spec-merchants.md 14.2): the bars of
// its merchants are 10 000 over the fee share, a share of the cost.
func merchantShareWarnings(rb []string, r *Role) []Warning {
	share, spread := r.MerchantFeeShare, r.Merchant.FeeShareSpreadBP
	p := path(rb, "merchant", "fee_share_spread_bp")
	var out []Warning
	if lo, hi := share-spread, share+spread; lo >= 1 && hi >= 1 && wholeBP*wholeBP/lo-wholeBP*wholeBP/hi < wholeBP {
		out = append(out, Warning{Path: p, Rule: "merchant_spread_narrow"})
	}
	if share+spread > 9_000 {
		out = append(out, Warning{Path: p, Rule: "merchant_fee_share_high"})
	}
	if share-spread < 500 {
		out = append(out, Warning{Path: p, Rule: "merchant_fee_share_low"})
	}
	return out
}

// merchantRouteWarnings returns lane_without_consignee, cargo_never_delivered and
// consigns_without_cargo (spec-merchants.md 14.2): the warnings that read the
// routes of the configured graph, no edge status applied. Warnings appends them
// after the structural phases come back clean, as it appends the migration route
// warnings; the module being off or a graph that does not build gives nothing.
func merchantRouteWarnings(c *Config) []Warning {
	modes := modeIndex(c)
	if modes == nil {
		return nil
	}
	g, err := NewGraph(*c)
	if err != nil {
		return nil
	}
	rt := newMerchantRoutes(c, g, modes)
	var out []Warning
	for i := range c.Roles {
		if r := &c.Roles[i]; r.Merchant != nil {
			out = append(out, rt.role(r)...)
		}
	}
	return out
}

// merchantRoutes is what the route warnings read: the graph, the modes, where
// each role has a group and the regions that have a group of a consigning role.
type merchantRoutes struct {
	c          *Config
	g          Graph
	modes      map[string]*Mode
	at         map[[2]string]bool
	consigning map[string]bool
}

// newMerchantRoutes indexes where each role has a group and which regions have a
// group of a consigning role.
func newMerchantRoutes(c *Config, g Graph, modes map[string]*Mode) merchantRoutes {
	rt := merchantRoutes{c: c, g: g, modes: modes, at: map[[2]string]bool{}, consigning: map[string]bool{}}
	consigns := map[string]bool{}
	for i := range c.Roles {
		consigns[c.Roles[i].ID] = c.Roles[i].Merchant != nil && c.Roles[i].Merchant.Consigns
	}
	for _, grp := range c.Groups {
		rt.at[[2]string{grp.Region, grp.Role}] = true
		rt.consigning[grp.Region] = rt.consigning[grp.Region] || consigns[grp.Role]
	}
	return rt
}

// role returns the route warnings of one merchant role.
func (rt merchantRoutes) role(r *Role) []Warning {
	mp := []string{"roles", r.ID, "merchant"}
	var out []Warning
	if n := rt.lanesWithoutConsignee(r); n > 0 {
		out = append(out, Warning{Path: path(mp, "modes"), Rule: "lane_without_consignee", Count: n})
	}
	if n := rt.neverFresh(r); n > 0 {
		out = append(out, Warning{Path: path(mp, "modes"), Rule: "cargo_never_delivered", Count: n})
	}
	if r.Merchant.Consigns && !rt.anyCargoArrives(r) {
		out = append(out, Warning{Path: path(mp, "consigns"), Rule: "consigns_without_cargo"})
	}
	return out
}

// regionID returns the ID of the region at index i of the canonical regions.
func (rt merchantRoutes) regionID(i RegionID) string { return rt.c.Regions[int(i)].ID }

// lanesWithoutConsignee counts the ordered pairs of regions with a route, whose
// origin has a group of the role and which the role lists a hired mode for, where
// the destination has no group of a role that consigns.
func (rt merchantRoutes) lanesWithoutConsignee(r *Role) int64 {
	if !r.Merchant.hasHiredMode(rt.modes) {
		return 0
	}
	n := int64(0)
	for from := range RegionID(rt.g.Len()) {
		for to := range RegionID(rt.g.Len()) {
			_, ok := rt.g.Distance(from, to)
			if ok && from != to && rt.at[[2]string{rt.regionID(from), r.ID}] && !rt.consigning[rt.regionID(to)] {
				n++
			}
		}
	}
	return n
}

// neverFresh counts the (commodity, mode) pairs of the role with a cargo
// commodity of shelf life S above 0 for which every route from a region the role
// is in takes more than S - 2 ticks (spec-merchants.md 5.3.1: the youngest lot
// it can send is one tick old, so the life left is S - 2): no lot can arrive
// fresh. A role with no route gives none.
func (rt merchantRoutes) neverFresh(r *Role) int64 {
	n := int64(0)
	cargo := merchantCargo(r, merchantTradeInputs(r.Merchant, rt.modes))
	for _, id := range r.Merchant.Modes {
		md := rt.modes[id]
		minK, any := rt.shortest(r, md)
		if md == nil || !any {
			continue
		}
		for _, cid := range cargo {
			if cm := rt.cmdty(cid); cm != nil && cm.ShelfLife > 0 && minK > cm.ShelfLife-2 {
				n++
			}
		}
	}
	return n
}

// cmdty returns the commodity of an ID, nil when none.
func (rt merchantRoutes) cmdty(id string) *Commodity {
	if i, ok := rt.c.commodityID(id); ok {
		return &rt.c.Commodities[i]
	}
	return nil
}

// shortest returns the least travel time of the mode over the routes that start
// where the role has a group, and whether there is one.
func (rt merchantRoutes) shortest(r *Role, md *Mode) (minK int64, any bool) {
	if md == nil || md.Speed < 1 {
		return 0, false
	}
	for from := range RegionID(rt.g.Len()) {
		for to := range RegionID(rt.g.Len()) {
			if k, ok := rt.travel(r, md, from, to); ok && (!any || k < minK) {
				minK, any = k, true
			}
		}
	}
	return minK, any
}

// travel returns the travel time of the mode on the route from one region to
// another that starts where the role has a group.
func (rt merchantRoutes) travel(r *Role, md *Mode, from, to RegionID) (int64, bool) {
	d, ok := rt.g.Distance(from, to)
	if !ok || from == to || !rt.at[[2]string{rt.regionID(from), r.ID}] {
		return 0, false
	}
	return transit.Ticks(d, md.Speed, wholeBP)
}

// anyCargoArrives reports whether some merchant role has a group in a region
// other than a region of the consigning role, a route to it and a cargo
// commodity in common with the consigning role's own cargo set.
func (rt merchantRoutes) anyCargoArrives(r *Role) bool {
	want := merchantCargo(r, merchantTradeInputs(r.Merchant, rt.modes))
	for i := range rt.c.Roles {
		m := &rt.c.Roles[i]
		if m.Merchant == nil || !rt.shares(merchantCargo(m, merchantTradeInputs(m.Merchant, rt.modes)), want) {
			continue
		}
		if rt.reaches(m, r) {
			return true
		}
	}
	return false
}

// shares reports whether two sorted commodity lists have a member in common.
func (rt merchantRoutes) shares(a, b []string) bool {
	for _, id := range a {
		if slices.Contains(b, id) {
			return true
		}
	}
	return false
}

// reaches reports whether the merchant role m has a group in some region with a
// route to a different region where the consigning role has a group.
func (rt merchantRoutes) reaches(m, consigning *Role) bool {
	for from := range RegionID(rt.g.Len()) {
		for to := range RegionID(rt.g.Len()) {
			_, ok := rt.g.Distance(from, to)
			if ok && from != to && rt.at[[2]string{rt.regionID(from), m.ID}] && rt.at[[2]string{rt.regionID(to), consigning.ID}] {
				return true
			}
		}
	}
	return false
}

// DeliveryOffer offers goods for delivery to another region (spec-merchants.md
// 9.2). It is indivisible and lasts one tick. It names no consignee: the round
// chooses the payer.
type DeliveryOffer struct {
	// Mode is the hired mode that carries the lot and To the destination region.
	Mode ModeID
	To   RegionID
	// Commodity and Quantity are the goods, whole units, all or nothing.
	Commodity CommodityID
	Quantity  Quantity
	// Limit is the least delivered price per unit the deliverer accepts.
	Limit Price
}

// DeliveryTake asks for goods to be delivered to the taker's region (spec-merchants.md
// 9.2). It names no origin, may be filled in parts and lasts one tick.
type DeliveryTake struct {
	// Commodity is the commodity wanted.
	Commodity CommodityID
	// Quantity is the most units wanted.
	Quantity Quantity
	// Limit is the most per unit the taker pays.
	Limit Price
}

// PlanNotes is what a strategy reports of its decisions, as counts
// (spec-merchants.md 15.1.2). A strategy that has nothing to report leaves it
// zero; the world sums it per region into MerchantsPlanned and reads it for
// nothing else, so it influences no result.
type PlanNotes struct {
	// Considered is the candidates and NoRumour to BelowShare the first case that
	// decided each.
	Considered, NoRumour, Stale, Locked, Perishable, NoMargin, Unaffordable, BelowShare int64
	// Planned is the trips planned, Dispatched the shipment requests, Offered the
	// delivery offers and Taken the delivery takes appended to the plan.
	Planned, Dispatched, Offered, Taken int64
	// Rested and Waiting count a rest and a stay, and LotCapped a lot cut by the
	// cap of the block.
	Rested, Waiting, LotCapped int64
}

// MerchantView is the merchant group of an AgentView (spec-merchants.md 4.1):
// plain data, present only in the view passed to Trade for an agent whose role has
// a merchant block. Outbound is not a field of it: it is AgentView.Outbound.
type MerchantView struct {
	// Params is the role's block by value, with Modes empty (the modes are in
	// Modes), and FeeShare is the role's MerchantFeeShare.
	Params   Merchant
	FeeShare int64
	// Modes are the modes of the block in mode order, by value.
	Modes []MerchantMode
	// Goods are the commodities of the tradable set in canonical order, and Cargo
	// the cargo set (spec-merchants.md 3.2.2): the commodities the role trades less
	// the trade inputs of its modes, ascending, shared by the agents of the role.
	Goods []MerchantGood
	Cargo []CommodityID
}

// MerchantMode is a mode of the block as the merchant plans with it: the
// definitions of spec-transport.md 3.1 by value, with the commodities resolved.
type MerchantMode struct {
	// Mode is the mode and Hired says it is carried by haul units.
	Mode  ModeID
	Hired bool
	// UnitMass and UnitBulk are the capacity of a carrying unit; Crew the
	// households it needs.
	UnitMass, UnitBulk, Crew int64
	// Vehicle and Haul are the commodities of the vehicle and of the haul unit,
	// NoCommodity when the mode has none; WearPerTick is the vehicle's wear.
	Vehicle     CommodityID
	WearPerTick int64
	Haul        CommodityID
	// HaulSpan is the distance one haul unit carries one carrying unit.
	HaulSpan int64
	// Feed is the goods one carrying unit consumes per tick of travel.
	Feed []Line
}

// MerchantGood is a commodity of the tradable set as the merchant sees it: the
// definitions of spec-goods.md 8 and the age of the oldest lot it holds.
type MerchantGood struct {
	// Commodity is the commodity.
	Commodity CommodityID
	// Mass, Bulk, ShelfLife and Durability are its definitions.
	Mass, Bulk, ShelfLife, Durability int64
	// OldestAge is the ticks since the born tick of the oldest lot the agent holds,
	// -1 when it holds none; it is 0 for a held commodity that never spoils, whose
	// lots have no born tick to read.
	OldestAge int64
}

// plannedTally is the sum of the plan notes of the plans of one region in a tick
// and the agents with a merchant block given a view there, for MerchantsPlanned.
type plannedTally struct {
	merchants int64
	notes     PlanNotes
}

// merchantRoleRow is what the filler copies into a view for a role with a block:
// the tables built once at construction.
type merchantRoleRow struct {
	has    bool
	params Merchant
	share  int64
	modes  []MerchantMode
	goods  []MerchantGood
	cargo  []CommodityID
}

// merchants is the state of the merchants module and of the merchant roles: the
// per-role view tables and the per-region tally of the plan notes. The tally is
// kept only when an observer was supplied, and is cleared by P3.3a.
type merchants struct {
	// roles is the number of roles with a merchant block (MerchantsPlanned is gated
	// by it, not by the module).
	roles int
	rows  []merchantRoleRow
	tally []plannedTally
	// goods and lots are the scratch of the filler.
	goods []MerchantGood
	lots  []ledger.Lot
	// The state of the contracts, set while the module is on (on): the next
	// contract ID, the open contracts in ascending ID order, the cap MaxOpen and, per
	// region, the open Escrow accounts no open contract uses.
	on      bool
	maxOpen int64
	next    ContractID
	open    []contract
	free    []transit.Free
	// The scratch of the round, cleared at its end: the entities the plans
	// submitted, the admitted ones, their conversions for internal/merchant, the
	// pairs it formed and the counts of ContractRound by destination region (dump
	// takes the counts when no observer was supplied).
	subOffers []submittedOffer
	subTakes  []submittedTake
	offers    []roundOffer
	takes     []roundTake
	rOffers   []merchant.Offer
	rTakes    []merchant.Bid
	pairs     []merchant.Pair
	counts    []roundCount
	dump      roundCount
	line      [1]Line
}

// newMerchants builds the state for a canonical configuration that passed
// Validate: nil when no role has a merchant block and the module is off.
func newMerchants(c *Config, rt *roleTable, t *transportState) *merchants {
	m := &merchants{rows: make([]merchantRoleRow, len(c.Roles)), tally: make([]plannedTally, len(c.Regions))}
	if mod := c.Modules.Merchants; mod != nil {
		m.on, m.maxOpen = true, mod.MaxOpen
		m.free, m.counts = make([]transit.Free, len(c.Regions)), make([]roundCount, len(c.Regions))
	}
	for i := range c.Roles {
		if c.Roles[i].Merchant == nil {
			continue
		}
		m.roles++
		m.rows[i] = newMerchantRow(c, &c.Roles[i], &rt.rows[i], t)
		m.goods = slices.Grow(m.goods, len(m.rows[i].goods))
	}
	if m.roles == 0 && !m.on {
		return nil
	}
	return m
}

// newMerchantRow builds the view tables of one role with a block.
func newMerchantRow(c *Config, r *Role, row *roleRow, t *transportState) merchantRoleRow {
	b := *r.Merchant
	b.Modes = nil
	out := merchantRoleRow{has: true, params: b, share: r.MerchantFeeShare}
	ids := make([]ModeID, 0, len(r.Merchant.Modes))
	for i := range c.Modules.Transport.Modes {
		if slices.Contains(r.Merchant.Modes, c.Modules.Transport.Modes[i].ID) {
			ids = append(ids, ModeID(i))
		}
	}
	for _, id := range ids {
		md, res := &c.Modules.Transport.Modes[id], &t.modes[id]
		out.modes = append(out.modes, MerchantMode{
			Mode: id, Hired: md.hired(), UnitMass: md.UnitMass, UnitBulk: md.UnitBulk, Crew: md.Crew,
			Vehicle: res.vehicle, WearPerTick: md.WearPerTick, Haul: res.haul, HaulSpan: md.HaulSpan,
			Feed: slices.Clone(res.feed),
		})
	}
	for _, id := range merchantCargo(r, merchantTradeInputs(r.Merchant, modeIndex(c))) {
		cm, _ := c.commodityID(id)
		out.cargo = append(out.cargo, cm)
	}
	slices.Sort(out.cargo)
	for _, cm := range row.tradable {
		d := &c.Commodities[cm]
		out.goods = append(out.goods, MerchantGood{Commodity: cm, Mass: d.Mass, Bulk: d.Bulk, ShelfLife: d.ShelfLife, Durability: d.Durability})
	}
	return out
}

// fillMerchantView sets the merchant group of the view passed to Trade for an
// agent whose role has a block, and counts the agent for MerchantsPlanned when an
// observer was supplied. It is the module's filler of the fillAgentView fan-in;
// the goods are a scratch copy valid for the call (spec-merchants.md 4.1).
func (w *World) fillMerchantView(v *AgentView, a *agent) {
	m := w.mer
	row := &m.rows[a.role]
	if !row.has {
		return
	}
	lots, err := w.ledger.AppendLots(m.lots[:0], a.account)
	if err != nil {
		// Excluded from coverage (testing.md 9.5): fillCore has read this account.
		invariant("the ledger refused the lots of a live agent: " + err.Error())
	}
	m.lots = lots
	m.goods = append(m.goods[:0], row.goods...)
	oldestAges(m.goods, lots, w.Tick())
	if w.obs.supplied {
		m.tally[v.Region].merchants++
	}
	v.HasMerchant = true
	v.Merchant = MerchantView{Params: row.params, FeeShare: row.share, Modes: row.modes[:len(row.modes):len(row.modes)], Goods: m.goods[:len(m.goods):len(m.goods)], Cargo: row.cargo[:len(row.cargo):len(row.cargo)]}
}

// oldestAges sets the OldestAge of each good from the lots of the account, which
// are in (commodity, born) order, so the first lot of a commodity is its oldest.
// Both goods and lots are in ascending commodity order.
func oldestAges(goods []MerchantGood, lots []ledger.Lot, tick int64) {
	j := 0
	for i := range goods {
		g := &goods[i]
		g.OldestAge = -1
		for j < len(lots) && lots[j].Commodity < int(g.Commodity) {
			j++
		}
		if j < len(lots) && lots[j].Commodity == int(g.Commodity) {
			if g.ShelfLife > 0 {
				g.OldestAge = tick - lots[j].Born
			} else {
				g.OldestAge = 0
			}
		}
	}
}

// checkMerchantPlan is the module's check of the trade plan of agent a (the
// checkTradePlan fan-in): for an agent with a merchant block it adds the plan
// notes into the tally of the agent's region, when an observer was supplied. The
// notes are read for nothing else.
func (w *World) checkMerchantPlan(a *agent, p *TradePlan) {
	m := w.mer
	if len(p.Offers) > 0 || len(p.Takes) > 0 {
		w.collectContracts(a, p)
	}
	if !w.obs.supplied || !m.rows[a.role].has {
		return
	}
	t := &m.tally[w.regionOf(a)].notes
	n := &p.Notes
	t.Considered += n.Considered
	t.NoRumour += n.NoRumour
	t.Stale += n.Stale
	t.Locked += n.Locked
	t.Perishable += n.Perishable
	t.NoMargin += n.NoMargin
	t.Unaffordable += n.Unaffordable
	t.BelowShare += n.BelowShare
	t.Planned += n.Planned
	t.Dispatched += n.Dispatched
	t.Offered += n.Offered
	t.Taken += n.Taken
	t.Rested += n.Rested
	t.Waiting += n.Waiting
	t.LotCapped += n.LotCapped
}

// p33aPlanned is P3.3a: one MerchantsPlanned per region in ascending RegionID in
// which an agent with a merchant block was given a view, from the tally the filler
// and the plan check kept. It emits only when an observer was supplied, which is
// also the only time the tally is kept, and it clears the tally for the next tick.
func (w *World) p33aPlanned() {
	if !w.obs.supplied {
		return
	}
	for r := range w.mer.tally {
		t := &w.mer.tally[r]
		if t.merchants > 0 {
			n := &t.notes
			w.obs.deliver(MerchantsPlanned{
				Tick: w.Tick(), Region: RegionID(r), Merchants: t.merchants,
				Considered: n.Considered, NoRumour: n.NoRumour, Stale: n.Stale, Locked: n.Locked,
				Perishable: n.Perishable, NoMargin: n.NoMargin, Unaffordable: n.Unaffordable, BelowShare: n.BelowShare,
				Planned: n.Planned, Dispatched: n.Dispatched, Offered: n.Offered, Taken: n.Taken,
				Rested: n.Rested, Waiting: n.Waiting, LotCapped: n.LotCapped,
			})
		}
		*t = plannedTally{}
	}
}

// The delivery contract (spec-merchants.md 9). The plan check collects the offers
// and takes of every plan; P3.3 admits them, draws their keys, matches them lane by
// lane and forms the contracts: the shipment is dispatched by the deliverer, the
// payer's money moves to an Escrow account and the record is kept. Settlement,
// removal and the checkpoint are contract_settle.go.

// contract is a delivery contract in flight (spec-merchants.md 16): its parties,
// the shipment that carries the goods, the goods and the price, the Escrow account
// that holds the money, the tick it was formed and the destination region.
type contract struct {
	id               ContractID
	payer, deliverer AgentID
	shipment         ShipmentID
	commodity        CommodityID
	qty              Quantity
	price            Price
	escrow           int
	formed           int64
	to               RegionID
}

// submittedOffer and submittedTake are an offer and a take as a plan returned them:
// the agent, the index in the plan and, for an offer, the plan's shipment requests.
type submittedOffer struct {
	agent AgentID
	idx   int
	ships int
	o     DeliveryOffer
}

type submittedTake struct {
	agent AgentID
	idx   int
	t     DeliveryTake
}

// roundOffer and roundTake are an admitted offer and take with the key drawn for
// it and the region of its agent (for an offer the deliverer's, for a take the
// payer's, which is the destination of its contracts).
type roundOffer struct {
	agent  AgentID
	region RegionID
	o      DeliveryOffer
	key    uint64
}

type roundTake struct {
	agent  AgentID
	region RegionID
	t      DeliveryTake
	key    uint64
}

// roundCount is what ContractRound reports for one destination region.
type roundCount struct {
	offers, takes, offersRefused, takesRefused                     int64
	matched, formed, noTaker, payerFunds, tooMany, dispatchRefused int64
	units                                                          int64
	value                                                          Money
}

// cnt returns the counts of the region, or the dump when no observer was supplied
// or the region is not a region (an offer to an unknown one): a counter that only
// builds an event is kept only for an observer (observability-and-logging.md 4.5).
func (m *merchants) cnt(supplied bool, r RegionID) *roundCount {
	if !supplied || r < 0 || int(r) >= len(m.counts) {
		return &m.dump
	}
	return &m.counts[r]
}

// collectContracts is the plan check's part for the offers and takes of agent a:
// with the module absent each is refused at once as contracts_off (a Debug line,
// no event); else it is kept, with its index in the plan, for the round of P3.3.
func (w *World) collectContracts(a *agent, p *TradePlan) {
	m := w.mer
	if !m.on {
		for range p.Offers {
			w.logContractRefused("offer refused", a.id, ReasonContractsOff)
		}
		for range p.Takes {
			w.logContractRefused("take refused", a.id, ReasonContractsOff)
		}
		return
	}
	for i, o := range p.Offers {
		m.subOffers = append(m.subOffers, submittedOffer{agent: a.id, idx: i, ships: len(p.Shipments), o: o})
	}
	for i, t := range p.Takes {
		m.subTakes = append(m.subTakes, submittedTake{agent: a.id, idx: i, t: t})
	}
}

// logContractRefused writes the Debug line of a refused offer or take.
func (w *World) logContractRefused(msg string, agent AgentID, reason Reason) {
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, msg,
			slog.Int64(logkeys.Agent, int64(agent)), slog.String(logkeys.Reason, string(reason)))
	}
}

// p33Round is P3.3 (spec-merchants.md 9.3 to 9.5): the admission of every offer and
// take of the plans in agent and plan order, the keys, the lanes in ascending
// (destination, commodity) order with the matching and the formation, and
// ContractRound per destination region. Every scratch is cleared at its end.
func (w *World) p33Round() {
	m := w.mer
	w.admitOffers()
	w.admitTakes()
	w.matchLanes()
	w.emitRounds()
	m.subOffers, m.subTakes, m.offers, m.takes = m.subOffers[:0], m.subTakes[:0], m.offers[:0], m.takes[:0]
	m.rOffers, m.rTakes, m.pairs = m.rOffers[:0], m.rTakes[:0], m.pairs[:0]
	m.dump = roundCount{}
}

// offerChecks are the checks of an offer in the order of the table of
// spec-merchants.md 9.3 (rows 2 to 7). Rows 4 (outside_destination, plan 2-16) and
// 5a (not_permitted, institutions) have no check yet and take their place in the
// table when their module lands. The table is never written.
var offerChecks = [...]func(w *World, a *agent, s *submittedOffer, used int) Reason{
	ckOfferCount,   // 2
	ckOfferMode,    // 3
	ckOfferRegion,  // 4: unknown_region, same_region
	ckOfferGoods,   // 5
	ckOfferAmounts, // 6
	ckOfferStock,   // 7
}

// ckOfferCount is row 2: the offer would be request number MaxDispatchPerTick + 1
// or later of the plan.
func ckOfferCount(_ *World, _ *agent, s *submittedOffer, used int) Reason {
	if s.ships+used >= MaxDispatchPerTick {
		return ReasonTooManyRequests
	}
	return ""
}

// ckOfferMode is row 3: a mode of the configuration.
func ckOfferMode(w *World, _ *agent, s *submittedOffer, _ int) Reason {
	if s.o.Mode < 0 || int(s.o.Mode) >= len(w.transport.modes) {
		return ReasonUnknownMode
	}
	return ""
}

// ckOfferRegion is row 4: the destination is a region and not the deliverer's own.
func ckOfferRegion(w *World, a *agent, s *submittedOffer, _ int) Reason {
	switch {
	case s.o.To < 0 || int(s.o.To) >= len(w.cfg.Regions):
		return ReasonUnknownRegion
	case s.o.To == w.regionOf(a):
		return ReasonSameRegion
	case w.isOutsideNode(s.o.To):
		return ReasonOutsideDestination
	}
	return ""
}

// ckOfferGoods is row 5: a commodity of the configuration, in the deliverer's
// tradable set.
func ckOfferGoods(w *World, a *agent, s *submittedOffer, _ int) Reason {
	return w.goodsRefusal(a, s.o.Commodity)
}

// goodsRefusal is the check unknown_commodity and not_tradable of an offer or a
// take for the agent.
func (w *World) goodsRefusal(a *agent, c CommodityID) Reason {
	switch {
	case c < 0 || int(c) >= len(w.cfg.Commodities):
		return ReasonUnknownCommodity
	case !w.roles.row(a.role).isTradable(c):
		return ReasonNotTradable
	}
	return ""
}

// ckOfferAmounts is row 6: a quantity and a limit in range.
func ckOfferAmounts(_ *World, _ *agent, s *submittedOffer, _ int) Reason {
	return amountsRefusal(int64(s.o.Quantity), int64(s.o.Limit))
}

// amountsRefusal is the check invalid_quantity and invalid_price.
func amountsRefusal(qty, limit int64) Reason {
	switch {
	case qty < 1 || qty > MaxQuantity:
		return ReasonInvalidQuantity
	case limit < 1 || limit > MaxPrice:
		return ReasonInvalidPrice
	}
	return ""
}

// ckOfferStock is row 7: the units are available, held and not reserved.
func ckOfferStock(w *World, a *agent, s *submittedOffer, _ int) Reason {
	held, reserved, err := w.ledger.Holding(a.account, int(s.o.Commodity))
	if err != nil {
		invariant("the ledger refused the holding of a live agent: " + err.Error())
	}
	if int64(s.o.Quantity) > held-reserved {
		return ReasonInsufficientStock
	}
	return ""
}

// admitOffers admits the submitted offers in agent and plan order (9.3): each is
// refused whole with the first reason that applies, or kept with the key drawn for
// it, the first word of the stream (siteContract, tick, agent, 2 * index).
func (w *World) admitOffers() {
	m, supplied, tick := w.mer, w.obs.supplied, w.Tick()
	prev, used := AgentID(-1), 0
	for i := range m.subOffers {
		s := &m.subOffers[i]
		if s.agent != prev {
			prev, used = s.agent, 0
		}
		a := &w.agents[s.agent]
		if reason := w.offerRefusal(a, s, used); reason != "" {
			m.cnt(supplied, s.o.To).offersRefused++
			w.logContractRefused("offer refused", s.agent, reason)
			continue
		}
		used++
		w.rnd.keyedPair(siteContract, tick, int64(s.agent), int64(2*s.idx))
		m.offers = append(m.offers, roundOffer{agent: s.agent, region: w.regionOf(a), o: s.o, key: w.rnd.word()})
		m.cnt(supplied, s.o.To).offers++
	}
}

// offerRefusal returns the reason of the first check of the offer table that the
// offer fails, or "". used is the offers of the same plan admitted before it.
func (w *World) offerRefusal(a *agent, s *submittedOffer, used int) Reason {
	for _, ck := range offerChecks {
		if r := ck(w, a, s, used); r != "" {
			return r
		}
	}
	return ""
}

// admitTakes admits the submitted takes in agent and plan order (9.3), drawing the
// key of each from the stream (siteContract, tick, agent, 2 * index + 1).
func (w *World) admitTakes() {
	m, supplied, tick := w.mer, w.obs.supplied, w.Tick()
	for i := range m.subTakes {
		s := &m.subTakes[i]
		a := &w.agents[s.agent]
		region := w.regionOf(a)
		if reason := w.takeRefusal(a, s); reason != "" {
			m.cnt(supplied, region).takesRefused++
			w.logContractRefused("take refused", s.agent, reason)
			continue
		}
		w.rnd.keyedPair(siteContract, tick, int64(s.agent), int64(2*s.idx+1))
		m.takes = append(m.takes, roundTake{agent: s.agent, region: region, t: s.t, key: w.rnd.word()})
		m.cnt(supplied, region).takes++
	}
}

// takeRefusal returns the reason of the first check of the take table that the
// take fails, or "": the commodity, the amounts, a second take for the commodity,
// a value that does not fit and a value above the available cash (rows 5, 6, 8, 9
// and 10; row 5a is institutions').
func (w *World) takeRefusal(a *agent, s *submittedTake) Reason {
	t := &s.t
	if r := w.goodsRefusal(a, t.Commodity); r != "" {
		return r
	}
	if r := amountsRefusal(int64(t.Quantity), int64(t.Limit)); r != "" {
		return r
	}
	for _, o := range slices.Backward(w.mer.takes) {
		if o.agent != s.agent {
			break
		}
		if o.t.Commodity == t.Commodity {
			return ReasonDuplicateTake
		}
	}
	value, ok := checked.Mul(int64(t.Limit), int64(t.Quantity))
	if !ok || value > MaxMoney {
		return ReasonOverflow
	}
	if value > w.cash(a) {
		return ReasonInsufficientFunds
	}
	return ""
}

// cash is the agent's available money: its balance less what its orders reserve.
func (w *World) cash(a *agent) int64 {
	v, err := w.ledger.Account(a.account)
	if err != nil {
		invariant("the ledger refused the account of a live agent: " + err.Error())
	}
	return v.Balance - v.Reserved
}

// matchLanes runs the lanes in ascending (destination, commodity) order (9.4.2):
// the offers of a lane by limit then key, the takes of the agents in the
// destination by limit descending then key, first fit with the midpoint, and the
// formation of each pair. An offer's lane with no take, or whose takes are all
// taken, finds none.
func (w *World) matchLanes() {
	m := w.mer
	if len(m.offers) == 0 {
		return
	}
	m.sortLanes()
	j := 0
	for lo := 0; lo < len(m.rOffers); {
		l := m.offerLane(m.rOffers[lo].Ref)
		hi := lo + 1
		for hi < len(m.rOffers) && m.offerLane(m.rOffers[hi].Ref) == l {
			hi++
		}
		for j < len(m.rTakes) && laneBefore(m.takeLane(m.rTakes[j].Ref), l) {
			j++
		}
		k := j
		for k < len(m.rTakes) && m.takeLane(m.rTakes[k].Ref) == l {
			k++
		}
		w.matchLane(m.rOffers[lo:hi], m.rTakes[j:k], l)
		lo, j = hi, k
	}
}

// sortLanes converts the admitted offers and takes for internal/merchant and
// orders them so that the entities of a lane are together, lanes ascending, with
// the admission index to end the comparison (architecture.md 7.6).
func (m *merchants) sortLanes() {
	for i, o := range m.offers {
		m.rOffers = append(m.rOffers, merchant.Offer{Ref: i, Limit: int64(o.o.Limit), Qty: int64(o.o.Quantity), Key: o.key})
	}
	for i, t := range m.takes {
		m.rTakes = append(m.rTakes, merchant.Bid{Ref: i, Limit: int64(t.t.Limit), Qty: int64(t.t.Quantity), Key: t.key})
	}
	slices.SortFunc(m.rOffers, func(a, b merchant.Offer) int {
		return laneOrder(m.offerLane(a.Ref), m.offerLane(b.Ref), a.Ref, b.Ref)
	})
	slices.SortFunc(m.rTakes, func(a, b merchant.Bid) int {
		return laneOrder(m.takeLane(a.Ref), m.takeLane(b.Ref), a.Ref, b.Ref)
	})
}

// lane is a pair (destination region, commodity) of the round.
type lane struct {
	region    RegionID
	commodity CommodityID
}

func (m *merchants) offerLane(ref int) lane {
	return lane{m.offers[ref].o.To, m.offers[ref].o.Commodity}
}

func (m *merchants) takeLane(ref int) lane {
	return lane{m.takes[ref].region, m.takes[ref].t.Commodity}
}

// laneBefore reports whether lane a comes before lane b in ascending (region,
// commodity) order.
func laneBefore(a, b lane) bool {
	return a.region < b.region || (a.region == b.region && a.commodity < b.commodity)
}

// laneOrder is the total order that groups the entities of a lane: the lane, then
// the admission index, which is unique (architecture.md 7.6).
func laneOrder(a, b lane, ra, rb int) int {
	return cmp.Or(cmp.Compare(a.region, b.region), cmp.Compare(a.commodity, b.commodity), cmp.Compare(ra, rb))
}

// matchLane sorts the offers and takes of one lane by the order of 9.4.2 and runs
// the first-fit matching with the formation of each tentative pair. The offers that
// were not matched are NoTaker.
func (w *World) matchLane(offers []merchant.Offer, takes []merchant.Bid, l lane) {
	m := w.mer
	merchant.SortOffers(offers)
	merchant.SortBids(takes)
	c := m.cnt(w.obs.supplied, l.region)
	before := c.matched
	m.pairs = merchant.Match(m.pairs[:0], offers, takes, func(p merchant.Pair) bool { return w.formPair(p) })
	c.noTaker += int64(len(offers)) - (c.matched - before)
}

// formPair is the formation of a tentative match (9.5): the checks in order
// (MaxOpen, the payer's cash), the dispatch of the shipment by the deliverer, the
// Escrow account and the move of the money, then the record and ContractFormed. The
// first failure ends the attempt with its reason and changes nothing. It returns
// whether the contract was formed.
func (w *World) formPair(p merchant.Pair) bool {
	m := w.mer
	of, tk := &m.offers[p.Offer], &m.takes[p.Bid]
	c := m.cnt(w.obs.supplied, of.o.To)
	c.matched++
	value, _ := merchant.Escrow(p.Price, p.Qty) // price <= MaxPrice and qty <= MaxQuantity: it fits MaxMoney
	payer, deliverer := &w.agents[tk.agent], &w.agents[of.agent]
	switch {
	case int64(len(m.open)) >= m.maxOpen:
		c.tooMany++
		w.logContractRefused("offer refused", of.agent, ReasonTooManyContracts)
		return false
	case value > w.cash(payer):
		c.payerFunds++
		w.logContractRefused("offer refused", of.agent, ReasonPayerFunds)
		return false
	}
	m.line[0] = Line{Commodity: of.o.Commodity, Quantity: Quantity(p.Qty)}
	req := ShipmentRequest{Mode: of.o.Mode, To: of.o.To, Consignee: tk.agent, Lines: m.line[:1]}
	id, reason := w.dispatch(deliverer, 0, &req, of.region)
	if reason != "" {
		c.dispatchRefused++
		w.logContractRefused("offer refused", of.agent, reason)
		return false
	}
	ct := contract{id: m.next, payer: tk.agent, deliverer: of.agent, shipment: id, commodity: of.o.Commodity,
		qty: Quantity(p.Qty), price: Price(p.Price), escrow: w.escrowAccount(of.o.To), formed: w.Tick(), to: of.o.To}
	if err := w.ledger.Transfer(payer.account, ct.escrow, value, ledger.EscrowIn); err != nil {
		invariant("the ledger refused the escrow of a checked contract: " + err.Error())
	}
	m.next++
	m.open = append(m.open, ct)
	c.formed++
	c.units += p.Qty
	c.value += Money(value)
	w.announceContract(&ct, of.region)
	return true
}

// escrowAccount takes the open Escrow account of the region with the lowest ID that
// no open contract uses, or opens one (9.6.2). Accounts are reused and never closed.
func (w *World) escrowAccount(region RegionID) int {
	if id, ok := w.mer.free[region].Take(); ok {
		return id
	}
	id, err := w.ledger.Open(ledger.KindEscrow, 0, int(region))
	if err != nil {
		invariant("the ledger refused an Escrow account: " + err.Error())
	}
	return id
}

// announceContract emits ContractFormed and writes the Debug line of a contract.
func (w *World) announceContract(ct *contract, origin RegionID) {
	if w.obs.supplied {
		w.obs.deliver(ContractFormed{
			Tick: w.Tick(), Contract: ct.id, Payer: ct.payer, Deliverer: ct.deliverer, Region: origin, To: ct.to,
			Commodity: ct.commodity, Quantity: ct.qty, Price: ct.price, Escrowed: Money(int64(ct.price) * int64(ct.qty)),
			Shipment: ct.shipment, Mode: w.shipmentOf(ct.shipment).mode, Arrives: w.shipmentOf(ct.shipment).arrives,
		})
	}
	ctx := context.Background()
	if w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "contract formed",
			slog.Int64(logkeys.Contract, int64(ct.id)), slog.Int64(logkeys.Agent, int64(ct.deliverer)),
			slog.Int64(logkeys.Region, int64(origin)), slog.Int64(logkeys.Dest, int64(ct.to)),
			slog.Int64(logkeys.Commodity, int64(ct.commodity)), slog.Int64(logkeys.Quantity, int64(ct.qty)),
			slog.Int64(logkeys.Price, int64(ct.price)))
	}
}

// shipmentOf returns the shipment with the ID, which is in flight: the ships are in
// ascending ID order.
func (w *World) shipmentOf(id ShipmentID) *shipment {
	i, ok := slices.BinarySearchFunc(w.transport.ships, id, func(s shipment, id ShipmentID) int { return cmp.Compare(s.id, id) })
	if !ok {
		invariant("a contract names a shipment that is not in flight")
	}
	return &w.transport.ships[i]
}

// emitRounds emits ContractRound per destination region in ascending RegionID for
// each region in which an offer or a take was submitted, and clears the counts.
func (w *World) emitRounds() {
	if !w.obs.supplied {
		return
	}
	for r := range w.mer.counts {
		c := &w.mer.counts[r]
		if c.offers+c.takes+c.offersRefused+c.takesRefused > 0 {
			w.obs.deliver(ContractRound{
				Tick: w.Tick(), Region: RegionID(r), Offers: c.offers, Takes: c.takes,
				OffersRefused: c.offersRefused, TakesRefused: c.takesRefused, Matched: c.matched, Formed: c.formed,
				NoTaker: c.noTaker, PayerFunds: c.payerFunds, TooMany: c.tooMany, DispatchRefused: c.dispatchRefused,
				Units: c.units, Value: c.value,
			})
		}
		*c = roundCount{}
	}
}
