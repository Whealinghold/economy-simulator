package strategy

import (
	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/merchant"
)

// carriage prices the carriage of a lot of q units of the cargo by the mode of the
// trip (spec-merchants.md 5.4) and returns the carrying units the lot needs. ok is
// false when the lot does not fit arithmetically or a price is missing.
func (s *state) carriage(t *trip, q int64) (c merchant.Carriage, n int64, ok bool) {
	g, md := &s.m.Goods[t.ci], t.mode
	n, ok = merchant.Units(q, g.Mass, g.Bulk, md.UnitMass, md.UnitBulk)
	if !ok {
		return c, 0, false
	}
	c = merchant.Carriage{N: n, K: t.k, Hired: md.Hired, S: t.s}
	if md.Hired {
		hi := s.at(md.Haul)
		if hi < 0 {
			return c, 0, false
		}
		c.HaulPrice = s.price(hi)
		return c, n, true
	}
	c.WearPerTick, c.UpkeepCost, c.TimeCostBP = md.WearPerTick, s.cost, s.m.Params.TimeCostBP
	if md.Vehicle != economysim.NoCommodity {
		vi := s.at(md.Vehicle)
		if vi < 0 {
			return c, 0, false
		}
		c.VehiclePrice, c.Durability = s.price(vi), s.m.Goods[vi].Durability
	}
	c.Feed = s.feedBuf[:0]
	for _, f := range md.Feed {
		fi := s.at(f.Commodity)
		if fi < 0 {
			return c, 0, false
		}
		c.Feed = append(c.Feed, merchant.Feed{Quantity: int64(f.Quantity), Price: s.price(fi)})
	}
	return c, n, true
}

// margin applies the test of the fee share to a lot of q units: the cost of
// carriage, the carrying units and the result of merchant.Margin.
func (s *state) margin(t *trip, q int64) (res merchant.Result, cost, n int64, ok bool) {
	c, n, ok := s.carriage(t, q)
	if !ok {
		return res, 0, 0, false
	}
	if cost, ok = c.Cost(); !ok {
		return res, 0, 0, false
	}
	res, ok = merchant.Margin(cost, s.phi, q, t.rnet, t.pacq)
	return res, cost, n, ok
}

// capUnits is the most units of the cargo a lot may hold (spec-merchants.md 6.2.1
// item 1): LotUnits carrying units of the mode, never more than MaxQuantity. For a
// self-carried mode the carrying units are shared with everything else the agent
// holds, so what the other holdings take of the mass and the bulk is left out.
func (s *state) capUnits(t *trip) int64 {
	g, md := &s.m.Goods[t.ci], t.mode
	lot := s.m.Params.LotUnits
	mass, ok1 := checked.Mul(lot, md.UnitMass)
	bulk, ok2 := checked.Mul(lot, md.UnitBulk)
	if !ok1 || !ok2 {
		return 0
	}
	if !md.Hired {
		mass, bulk = s.afterHoldings(t, mass, bulk)
	}
	return min(fits(mass, g.Mass), fits(bulk, g.Bulk), economysim.MaxQuantity)
}

// fits is the units of a good of the given per-unit figure that a capacity holds,
// a capacity or a figure of 0 leaving the term out (it holds without limit).
func fits(capacity, per int64) int64 {
	if capacity == 0 || per == 0 {
		return economysim.MaxQuantity
	}
	return capacity / per
}

// afterHoldings takes from a self-carried mode's capacity the mass and the bulk of
// everything the agent holds but the cargo itself, floored at 0: the pool is shared
// (spec-transport.md 3.2.2). A capacity of 0 means unlimited and stays so.
func (s *state) afterHoldings(t *trip, mass, bulk int64) (int64, int64) {
	var m, b int64
	for i := range s.v.Commodities {
		if i == t.ci {
			continue
		}
		g, stock := &s.m.Goods[i], s.v.Commodities[i].Stock
		dm, ok1 := checked.Mul(stock, g.Mass)
		db, ok2 := checked.Mul(stock, g.Bulk)
		if !ok1 || !ok2 {
			return 0, 0
		}
		m, b = m+dm, b+db
	}
	return leave(mass, m), leave(bulk, b)
}

// leave is what a capacity has left after a load, at least 1 so that a full pool
// reads as no room and not as unlimited, and 0 (unlimited) for a capacity of 0.
func leave(capacity, load int64) int64 {
	if capacity == 0 {
		return 0
	}
	return max(capacity-load, 1)
}

// size sets the lot of the candidate (6.2): the cap of the lot and what the money
// affords at the dearest limit the draw can give, with the cost of the trip's other
// inputs it does not yet hold taken out first. It sets t.q, t.n and t.capped and
// returns whether the lot is at least 1 unit and its break-even at least 1.
func (s *state) size(t *trip) bool {
	limit := s.high(t.ci)
	capQ := s.capUnits(t)
	if capQ < 1 || limit < 1 {
		return false
	}
	// The cost of the other inputs depends on the carrying units, which depend on
	// the lot; the first estimate is the lot the money affords with none.
	q0 := min(capQ, max(0, s.v.Money-s.reserve)/limit)
	if q0 < 1 {
		return false
	}
	_, n0, ok := s.carriage(t, q0)
	if !ok {
		return false
	}
	other, ok := s.otherCost(t, q0, n0)
	if !ok {
		return false
	}
	in := merchant.FitInput{Cap: capQ, Money: s.v.Money, Reserve: s.reserve, Other: other, High: limit}
	q := merchant.Fit(in, func(q int64) (int64, bool) {
		res, _, _, ok := s.margin(t, q)
		return res.BreakEven, ok
	})
	if q < 1 {
		return false
	}
	t.q, t.capped = q, q == capQ && in.Afford(in.High) > capQ
	return true
}

// line is one input of a trip: a commodity, the units the trip needs of it and the
// purpose of the order that buys them (spec-merchants.md 3.4.2).
type line struct {
	c       economysim.CommodityID
	qty     int64
	purpose economysim.Purpose
}

// lines returns the inputs of a trip of q units over n carrying units, in the
// order of 3.4.2: the cargo, then the haul units of a hired mode, or the vehicles,
// the feed and the provisions of a self-carried one. A commodity on two lines has
// both; the default takes their sum.
func (s *state) lines(dst []line, t *trip, q, n int64) ([]line, bool) {
	dst = append(dst, line{t.cargo, q, economysim.PurposeInput})
	md := t.mode
	if md.Hired {
		haul, ok := checked.Mul(n, t.s)
		return append(dst, line{md.Haul, haul, economysim.PurposeInput}), ok
	}
	if md.Vehicle != economysim.NoCommodity {
		dst = append(dst, line{md.Vehicle, n, economysim.PurposeTool})
	}
	for _, f := range md.Feed {
		fq, ok1 := checked.Mul(int64(f.Quantity), n)
		fq, ok2 := checked.Mul(fq, t.k)
		if !ok1 || !ok2 {
			return dst, false
		}
		dst = append(dst, line{f.Commodity, fq, economysim.PurposeInput})
	}
	for _, row := range s.v.Upkeep {
		per, ok1 := checked.Mul(s.v.Weight, row.Quantity)
		need, ok2 := checked.Mul(per, max(0, t.k-s.v.Tunables.CoverTicks))
		if !ok1 || !ok2 {
			return dst, false
		}
		if need > 0 {
			dst = append(dst, line{row.Commodity, need, economysim.PurposeUpkeep})
		}
	}
	return dst, true
}

// otherCost is the cost, at the dearest limit of each, of the trip's other inputs
// that the merchant does not yet hold above what the household keeps (6.2.1 item
// 2). ok is false on overflow.
func (s *state) otherCost(t *trip, q, n int64) (int64, bool) {
	var buf [16]line
	ls, ok := s.lines(buf[:0], t, q, n)
	if !ok {
		return 0, false
	}
	var total int64
	for _, l := range ls[1:] {
		i := s.at(l.c)
		if i < 0 {
			return 0, false
		}
		missing := max(0, l.qty-s.free(i))
		cost, ok := checked.Mul(missing, int64(s.v.Commodities[i].Bounds.Ceiling))
		if !ok {
			return 0, false
		}
		if total, ok = checked.Add(total, cost); !ok {
			return 0, false
		}
	}
	return total, true
}
