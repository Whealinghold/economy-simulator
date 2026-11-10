package strategy

import (
	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/merchant"
)

// takes appends the delivery takes of a consignee (spec-merchants.md 8.2): for
// each commodity of the cargo set in ascending order that is not the cargo of the
// trip planned this tick, a take at a discount to the local price for the units it
// lacks, counting what is on the way, within the cash its orders leave. A merchant
// whose role does not consign, one that rests and one in the own-use mode post
// none.
func (s *state) takes(p *economysim.TradePlan) {
	if s.idle || !s.m.Params.Consigns {
		return
	}
	earmarked := reserved(p)
	for _, c := range s.m.Cargo {
		if s.planned && c == s.best.cargo {
			continue
		}
		i := s.at(c)
		if i < 0 {
			continue
		}
		limit, qty, ok := merchant.Take(merchant.TakeInput{
			Price: s.price(i), MarginBP: s.m.Params.TakeMarginBP,
			Stock: s.v.Commodities[i].Stock, Keep: s.keep(i),
			Units: int64(s.m.Params.TakeUnits), Inbound: s.inbound(c),
			Money: s.v.Money, Reserve: s.reserve, Reserved: earmarked, MaxQty: economysim.MaxQuantity,
		})
		if !ok {
			continue
		}
		p.Takes = append(p.Takes, economysim.DeliveryTake{Commodity: c, Quantity: economysim.Quantity(qty), Limit: economysim.Price(limit)})
		s.notes.Taken++
	}
}

// inbound is the units of the commodity on the way to the agent as consignee.
func (s *state) inbound(c economysim.CommodityID) int64 {
	var total int64
	for _, r := range s.v.Inbound {
		if r.Commodity == c {
			total += int64(r.Quantity)
		}
	}
	return total
}
