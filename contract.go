package economysim

import (
	"maps"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/transit"
)

// This file is the merchants module (spec-merchants.md, MER): the delivery
// contract and the data of a merchant role. It holds, so far, the validation of
// the merchant block and of the module, the derived sets of a merchant role and
// the warnings of the configuration; the round, the escrow and the settlement
// come with the later phases of plan 2-15. The arithmetic is internal/merchant;
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
	for i := range rt.c.Commodities {
		if rt.c.Commodities[i].ID == id {
			return &rt.c.Commodities[i]
		}
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
}

// newMerchants builds the state for a canonical configuration that passed
// Validate: nil when no role has a merchant block and the module is off.
func newMerchants(c *Config, rt *roleTable, t *transportState) *merchants {
	m := &merchants{rows: make([]merchantRoleRow, len(c.Roles)), tally: make([]plannedTally, len(c.Regions))}
	for i := range c.Roles {
		if c.Roles[i].Merchant == nil {
			continue
		}
		m.roles++
		m.rows[i] = newMerchantRow(c, &c.Roles[i], &rt.rows[i], t)
		m.goods = slices.Grow(m.goods, len(m.rows[i].goods))
	}
	if m.roles == 0 && c.Modules.Merchants == nil {
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
