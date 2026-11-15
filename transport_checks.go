package economysim

import (
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/transit"
)

// This file holds the shipment checks of rows 18 to 21 that size the carrying
// capacity, the haul and the feed of a trip; the table, the other rows and row 22
// are in transport_request.go.

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
