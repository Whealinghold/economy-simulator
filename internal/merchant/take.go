package merchant

import "github.com/Whealinghold/economy-sim/internal/checked"

// TakeInput is what the take of a consignee reads (spec-merchants.md 8.2) for one
// cargo commodity: the local price, the discount, the units it holds above its
// keep, the units it wants to hold or have on the way, the units already on the
// way, and the cash it may spend.
type TakeInput struct {
	// Price is the local price of the commodity and MarginBP the discount at which
	// the consignee takes, in basis points.
	Price, MarginBP int64
	// Stock and Keep are the units held and the units the household keeps.
	Stock, Keep int64
	// Units is TakeUnits and Inbound the units already on the way to the agent.
	Units, Inbound int64
	// Money, Reserve and Reserved are the cash, the default's reserve and the money
	// the agent's orders of this plan already reserve.
	Money, Reserve, Reserved int64
	// MaxQty is the most units of an order.
	MaxQty int64
}

// valid reports whether the inputs are in range: shares in 0 to 10 000 and no
// negative quantity or amount.
func (in TakeInput) valid() bool {
	return in.MarginBP >= 0 && in.MarginBP <= wholeBP &&
		min(in.Price, in.Units, in.Inbound, in.Stock, in.Keep, in.Money, in.Reserve, in.Reserved, in.MaxQty) >= 0
}

// open returns the units the consignee lacks, units - free - inbound, and the
// money it may spend, money - reserve - reserved. With every input non-negative
// the first difference of each cannot overflow; ok is false when the second does.
func (in TakeInput) open() (want, budget int64, ok bool) {
	want, wok := checked.Sub(in.Units-max(0, in.Stock-in.Keep), in.Inbound)
	budget, bok := checked.Sub(in.Money-in.Reserve, in.Reserved)
	return want, budget, wok && bok
}

// Take returns the limit and the quantity of a take (spec-merchants.md 8.2):
//
//	limit = max(1, price * (10 000 - marginBP) / 10 000)             truncated
//	free  = max(0, stock - keep)
//	want  = units - free - inbound
//	afford = floor(max(0, money - reserve - reserved) / limit)
//	qty   = min(want, afford, maxQty)
//
// ok is false, with zeros, when qty is below 1 or an input is out of range; the
// take lasts one tick, like an order.
func Take(in TakeInput) (limit, qty int64, ok bool) {
	if !in.valid() {
		return 0, 0, false
	}
	limit, ok = checked.MulDiv(in.Price, wholeBP-in.MarginBP, wholeBP)
	if !ok {
		return 0, 0, false
	}
	limit = max(1, limit)
	want, budget, ok := in.open()
	if !ok {
		return 0, 0, false
	}
	if qty = min(want, max(0, budget)/limit, in.MaxQty); qty < 1 {
		return 0, 0, false
	}
	return limit, qty, true
}
