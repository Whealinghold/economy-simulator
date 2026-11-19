package economysim

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/institution"
)

// This file is the legal prices of the institutions module (spec-institutions.md 6):
// the controls and rate limits as one list in ControlID order, their levels in force,
// and the legal range in minor units resolved for every (region, commodity) and the
// rate cap for every region (6.2.2). The tables are derived and rebuilt, never saved;
// only the levels that differ from the configured ones are state (12.2). It is a
// continuation file of institutions.go (architecture.md 2.6).

// legalTable holds the controls and the rate limits by ControlID, ascending ID, with
// controls and rate limits sorted together. capBP and floorBP are the levels in force
// and baseCap and baseFloor the configured ones; a rate limit has a cap and no floor.
// capPrice, floorPrice and control are indexed by region * commodities + commodity (0
// is none) and rate by region.
type legalTable struct {
	ids                []string
	isRate             []bool
	capBP, floorBP     []int64
	baseCap, baseFloor []int64
	capPrice           []int64
	floorPrice         []int64
	control            []int32
	rate               []int64
	rateAt             []int32
	// tally is the clamps of the tick by (region, commodity) cell and rateTally those of the
	// rate limit by region, kept only when an observer was supplied and told at P7.2b
	// (spec-institutions.md 11.1). order is the scratch of the events of one region.
	tally     []institution.Tally
	rateTally []institution.Tally
	order     []controlRef
}

// controlRef names a tally of one region by its control, for the order of the events.
type controlRef struct {
	control int32
	cell    int32 // the (region, commodity) cell, -1 for the region's rate limit
}

// resolve derives every table from the configuration, the suspensions and the levels
// in force: the right in force per activity, subject and region, and the legal range
// per region and commodity. A construction, a restore and a slot call each call it
// once, so a read in the tick is an array read (spec-institutions.md 5.4.2, 6.2.2).
func (o *institutions) resolve() {
	o.initLevels()
	chains := o.chains()
	o.resolveRights(chains)
	o.resolveLegal(chains)
}

// initLevels builds the list of controls and rate limits in ControlID order with the
// configured levels, the first time it is called; the levels in force of a later call
// are what the slot or a restore left.
func (o *institutions) initLevels() {
	l := &o.legal
	if l.ids != nil {
		return
	}
	m := o.m
	l.ids = controlIDs(m)
	n := len(l.ids)
	l.isRate = make([]bool, n)
	l.baseCap, l.baseFloor = make([]int64, n), make([]int64, n)
	for i := range m.Controls {
		c := &m.Controls[i]
		k := o.controlIndex(c.ID)
		l.baseCap[k], l.baseFloor[k] = c.CapBP, c.FloorBP
	}
	for i := range m.RateLimits {
		r := &m.RateLimits[i]
		k := o.controlIndex(r.ID)
		l.isRate[k], l.baseCap[k] = true, r.CapBP
	}
	l.capBP, l.floorBP = slices.Clone(l.baseCap), slices.Clone(l.baseFloor)
}

// controlIndex is the ControlID of the control or rate limit with the ID.
func (o *institutions) controlIndex(id string) int {
	k, _ := slices.BinarySearch(o.legal.ids, id)
	return k
}

// resolveLegal fills the range of every (region, commodity) and the rate cap of every
// region from the levels in force. A range is the cap and the floor in minor units of
// the commodity's effective reference price in the region, innermost row first
// (spec-institutions.md 6.1.3, 6.2.1); an outside node has none.
func (o *institutions) resolveLegal(chains [][]int32) {
	l := &o.legal
	c := o.c
	nc := len(c.Commodities)
	l.capPrice, l.floorPrice = make([]int64, o.nr*nc), make([]int64, o.nr*nc)
	l.control, l.rate, l.rateAt = make([]int32, o.nr*nc), make([]int64, o.nr), make([]int32, o.nr)
	regions := indexOf(c.Regions, func(r Region) string { return r.ID })
	commodities := indexOf(c.Commodities, func(k Commodity) string { return k.ID })
	var rows, rates []institution.Row
	var rowID, rateID []int32
	for i := range o.m.Controls {
		ct := &o.m.Controls[i]
		rows = append(rows, institution.Row{Subject: int32(commodities[ct.Commodity]), Scope: o.scopeOf(ct.Region, ct.Jurisdiction, regions)})
		rowID = append(rowID, int32(o.controlIndex(ct.ID)))
	}
	for i := range o.m.RateLimits {
		r := &o.m.RateLimits[i]
		rates = append(rates, institution.Row{Scope: o.scopeOf(r.Region, r.Jurisdiction, regions)})
		rateID = append(rateID, int32(o.controlIndex(r.ID)))
	}
	for g := range o.nr {
		if o.outside[g] {
			continue
		}
		o.resolveRegion(g, chains[g], rows, rowID)
		if ref, ok := institution.Resolve(rates, 0, int32(g), chains[g]); ok {
			l.rateAt[g], l.rate[g] = rateID[ref]+1, l.capBP[rateID[ref]]
		}
	}
}

// resolveRegion fills the ranges of region g for every commodity that has a control.
func (o *institutions) resolveRegion(g int, chain []int32, rows []institution.Row, rowID []int32) {
	l := &o.legal
	nc := len(o.c.Commodities)
	for k := range nc {
		ref, ok := institution.Resolve(rows, int32(k), int32(g), chain)
		if !ok {
			continue
		}
		id := rowID[ref]
		capPrice, floorPrice, ok := institution.Range(int64(o.c.refPrice(RegionID(g), CommodityID(k))), l.capBP[id], l.floorBP[id], MaxPrice)
		if !ok {
			invariant("the legal range of a validated control does not fit after validation")
		}
		cell := g*nc + k
		l.capPrice[cell], l.floorPrice[cell], l.control[cell] = capPrice, floorPrice, id+1
	}
}

// rangeOf returns the cap and the floor in minor units of commodity k in region g, 0
// for none, and the ControlID of the control in force; ok is false when none is in
// force. It is an array read and allocates nothing (spec-institutions.md 6.2.2).
func (o *institutions) rangeOf(g RegionID, k CommodityID) (capPrice, floorPrice int64, id ControlID, ok bool) {
	l := &o.legal
	cell := int(g)*len(o.c.Commodities) + int(k)
	if l.control[cell] == 0 {
		return 0, 0, 0, false
	}
	return l.capPrice[cell], l.floorPrice[cell], ControlID(l.control[cell] - 1), true
}

// rangeAt returns the cap and the floor in minor units of commodity k in region g, 0 for
// none: what a book reads at each admission. It is an array read.
func (o *institutions) rangeAt(g RegionID, k CommodityID) (capPrice, floorPrice int64) {
	cell := int(g)*len(o.c.Commodities) + int(k)
	return o.legal.capPrice[cell], o.legal.floorPrice[cell]
}

// attachBooks gives every book of the markets the module, so that each admission reads the
// legal range of its market (hook H-5), and the counts of the clamps when an observer was
// supplied (observability-and-logging.md 4.5).
func (o *institutions) attachBooks(m *marketTable, supplied bool) {
	for _, b := range m.books {
		b.org = o
	}
	if supplied {
		o.legal.tally = make([]institution.Tally, o.nr*len(o.c.Commodities))
		o.legal.rateTally = make([]institution.Tally, o.nr)
	}
}

// noteClamp counts one order of qty units in region g, on the bid side or the ask side,
// whose limit the range replaced. A nil module and a world with no observer count
// nothing.
func (o *institutions) noteClamp(g RegionID, k CommodityID, bid bool, qty int64) {
	if o != nil && o.legal.tally != nil {
		o.legal.tally[int(g)*len(o.c.Commodities)+int(k)].Add(bid, true, qty)
	}
}

// clampLimit returns the limit price an order of the side for commodity k in region g is
// admitted with under the legal range, and whether the range replaced it (hook H-7, the
// clamp of the contract round, spec-institutions.md 6.7.1). A nil module changes nothing.
func (o *institutions) clampLimit(g RegionID, k CommodityID, bid bool, limit Price) (Price, bool) {
	if o == nil {
		return limit, false
	}
	capPrice, floorPrice := o.rangeAt(g, k)
	p, changed := institution.Clamp(int64(limit), bid, capPrice, floorPrice)
	return Price(p), changed
}

// clampOffer admits a delivery offer with its ask limit raised to the floor in force for its
// commodity in the destination region, and counts the clamp (spec-institutions.md 6.7.1).
func (o *institutions) clampOffer(d *DeliveryOffer) {
	if p, changed := o.clampLimit(d.To, d.Commodity, false, d.Limit); changed {
		d.Limit = p
		o.noteClamp(d.To, d.Commodity, false, int64(d.Quantity))
	}
}

// rateLimit is the rate seam of the credit module (hook H-8, spec-institutions.md 6.6): the
// limit a request of a borrower in region g is admitted with, which is the cap in force
// when the request's own is above it, and whether it was lowered. A rate limit bounds what
// a borrower accepts, never what a lender posts.
func (o *institutions) rateLimit(g RegionID, limit int64) (int64, bool) {
	if o.legal.rateAt[g] == 0 || limit <= o.legal.rate[g] {
		return limit, false
	}
	if o.legal.rateTally != nil {
		o.legal.rateTally[g].Add(true, true, 0)
	}
	return o.legal.rate[g], true
}

// reportControls is the PriceControlled events of P7.2b: for each region in ascending
// order, one event for each control with a clamp, in ascending ControlID, after the
// RightRefused events (spec-institutions.md 7.3.3). The tallies are cleared as they are
// told.
func (o *institutions) reportControls(w *World) {
	l := &o.legal
	nc := len(o.c.Commodities)
	for g := range o.nr {
		l.order = l.order[:0]
		for k := range nc {
			if l.tally[g*nc+k] != (institution.Tally{}) {
				l.order = append(l.order, controlRef{control: l.control[g*nc+k] - 1, cell: int32(g*nc + k)})
			}
		}
		if l.rateTally[g] != (institution.Tally{}) {
			l.order = append(l.order, controlRef{control: l.rateAt[g] - 1, cell: -1})
		}
		slices.SortFunc(l.order, func(a, b controlRef) int {
			return cmp.Or(cmp.Compare(a.control, b.control), cmp.Compare(a.cell, b.cell))
		})
		for _, r := range l.order {
			w.obs.deliver(o.controlEvent(w.Tick(), RegionID(g), r))
		}
	}
}

// controlEvent builds the event of one control of a region and clears its tally.
func (o *institutions) controlEvent(tick int64, g RegionID, r controlRef) PriceControlled {
	l := &o.legal
	t := &l.rateTally[g]
	k := NoCommodity
	if r.cell >= 0 {
		t, k = &l.tally[r.cell], CommodityID(int(r.cell)-int(g)*len(o.c.Commodities))
	}
	e := PriceControlled{Tick: tick, Region: g, Control: ControlID(r.control), Commodity: k, Bids: t.Lowered, Asks: t.Raised, Units: t.Units}
	*t = institution.Tally{}
	return e
}
