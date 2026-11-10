package strategy

import (
	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/merchant"
)

// prepare decides whether the planned trip is ready (6.3.1) and, if it is, tests
// it again at the lot the merchant can send (6.3.2). It returns false when a ready
// trip fails the second test: nothing is sent, nothing is held back for it and the
// default sells what is not needed (3.4.3). A trip that is not ready is prepared:
// the default buys its inputs from the lines of 3.4.2.
func (s *state) prepare() bool {
	t := &s.best
	qr, ok := s.readyLot(t)
	if !ok {
		return true
	}
	res, cost, _, ok := s.margin(t, qr)
	if !ok || !merchant.Pass(cost, s.phi, res.G) {
		return false
	}
	t.ready, t.qr, t.rr = true, qr, res
	return true
}

// readyLot returns the lot the merchant can send now: the cargo above what the
// household keeps, less what other lines of the trip take of the same commodity,
// at most the planned lot; ok is false when the merchant does not hold all the
// inputs of that lot (the haul units or the vehicles, the feed and the provisions).
func (s *state) readyLot(t *trip) (qr int64, ok bool) {
	var buf [16]line
	ls, ok := s.lines(buf[:0], t, t.q, t.n)
	if !ok {
		return 0, false
	}
	qr = min(t.q, s.free(t.ci)-sumFor(ls[1:], t.cargo))
	if qr < 1 {
		return 0, false
	}
	n, ok := merchant.Units(qr, s.m.Goods[t.ci].Mass, s.m.Goods[t.ci].Bulk, t.mode.UnitMass, t.mode.UnitBulk)
	if !ok {
		return 0, false
	}
	if ls, ok = s.lines(buf[:0], t, qr, n); !ok {
		return 0, false
	}
	for _, l := range ls[1:] {
		if i := s.at(l.c); i < 0 || sumFor(ls, l.c) > s.free(i) {
			return 0, false
		}
	}
	return qr, true
}

// sumFor is the units the lines need of the commodity.
func sumFor(ls []line, c economysim.CommodityID) int64 {
	var total int64
	for _, l := range ls {
		if l.c == c {
			total += l.qty
		}
	}
	return total
}

// inputs returns the lines of the trip as the entries the default takes through
// TradeWith (spec-merchants.md 3.4.2): at the planned lot, or at the lot it can
// send when the trip is ready, so that a ready trip buys nothing more.
func (s *state) inputs() []economysim.TripInput {
	t := &s.best
	q, n := t.q, t.n
	if t.ready {
		q = t.qr
		n, _ = merchant.Units(q, s.m.Goods[t.ci].Mass, s.m.Goods[t.ci].Bulk, t.mode.UnitMass, t.mode.UnitBulk)
	}
	var buf [16]line
	ls, _ := s.lines(buf[:0], t, q, n)
	out := s.tripBuf[:0]
	for _, l := range ls {
		out = append(out, economysim.TripInput{Commodity: l.c, Quantity: economysim.Quantity(min(l.qty, economysim.MaxQuantity)), Purpose: l.purpose})
	}
	return out
}

// lower sets the limit of the bids the default made from index n0 on: the bid for
// the cargo to the break-even of the lot and the bid for haul units to the ceiling
// of FR-CAR-004 (5.6). A bid whose limit falls below 1 is dropped, keeping the
// order of the rest.
func (s *state) lower(p *economysim.TradePlan, n0 int) {
	t := &s.best
	ceiling := int64(-1)
	if t.mode.Hired {
		ceiling, _ = merchant.HaulCeiling(t.res.G, 0, t.n, t.s, s.phi)
	}
	kept := p.Intents[:n0]
	for _, in := range p.Intents[n0:] {
		if in.Side == economysim.Bid {
			switch {
			case in.Commodity == t.cargo:
				in.Limit = economysim.Price(min(int64(in.Limit), t.res.BreakEven))
			case t.mode.Hired && in.Commodity == t.mode.Haul:
				in.Limit = economysim.Price(min(int64(in.Limit), ceiling))
			}
			if in.Limit < 1 {
				continue
			}
		}
		kept = append(kept, in)
	}
	p.Intents = kept
}

// send appends the one shipment request or delivery offer of a ready trip (6.3.2):
// the request to travel with everything the merchant holds for a self-carried mode,
// the offer of the lot at the least delivered price that passes for a hired one.
func (s *state) send(p *economysim.TradePlan) {
	t := &s.best
	if !t.ready {
		return
	}
	if !t.mode.Hired {
		p.Shipments = append(p.Shipments, economysim.ShipmentRequest{Mode: t.mode.Mode, To: t.dest, ToSelf: true})
		s.notes.Dispatched++
		return
	}
	if t.rr.AskLimit < 1 || t.rr.AskLimit > economysim.MaxPrice {
		return
	}
	p.Offers = append(p.Offers, economysim.DeliveryOffer{
		Mode: t.mode.Mode, To: t.dest, Commodity: t.cargo,
		Quantity: economysim.Quantity(t.qr), Limit: economysim.Price(t.rr.AskLimit),
	})
	s.notes.Offered++
}

// reserved is the money the bids of the plan reserve, at their limits, saturating:
// a consignee does not take with money its orders have earmarked (8.2).
func reserved(p *economysim.TradePlan) int64 {
	var total int64
	for _, in := range p.Intents {
		if in.Side != economysim.Bid {
			continue
		}
		v, ok := checked.Mul(int64(in.Limit), int64(in.Quantity))
		if !ok {
			return economysim.MaxMoney
		}
		if total, ok = checked.Add(total, v); !ok {
			return economysim.MaxMoney
		}
	}
	return total
}
