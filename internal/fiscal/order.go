package fiscal

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/moneyflow"
)

// Limit returns the limit price of a standing order (spec-fiscal.md 9.2):
// min(maxPrice, max(1, ref * limitBP / 10^4)), truncated toward zero. ref is the
// effective reference price of the commodity in the seat region. ok is false for
// a negative argument, a maxPrice below 1 or a product that leaves int64.
func Limit(ref, limitBP, maxPrice int64) (price int64, ok bool) {
	if ref < 0 || limitBP < 0 || maxPrice < 1 {
		return 0, false
	}
	p, ok := checked.MulDiv(ref, limitBP, Basis)
	if !ok {
		return 0, false
	}
	return min(maxPrice, max(1, p)), true
}

// PairOK reports whether a purchase and a release of one commodity of one
// authority never cross: the buy limit is strictly below the sell limit, on the
// integer limit prices and not on the basis points (spec-fiscal.md 9.4.1;
// spec-clearing.md 4.3: a bid not below the ask trades).
func PairOK(buyLimit, sellLimit int64) bool { return buyLimit < sellLimit }

// Wish returns the units a standing order wants this tick and the carry that
// remains (spec-fiscal.md 9.3.1): wish = floor((qty*scale + carry) / 10^4),
// carry = (qty*scale + carry) mod 10^4. A scale of 5 000 and a quantity of 1
// wants one unit every second tick. A wish above int64 saturates at MaxInt64, as a
// due of internal/moneyflow does; a validated quantity is far below it. ok is false
// for a negative argument, or a carry of 10^4 or more.
func Wish(qty, scale, carry int64) (wish, newCarry int64, ok bool) {
	if carry >= Basis {
		return 0, 0, false
	}
	return moneyflow.Spread(1, qty, scale, carry, 1, math.MaxInt64)
}

// Capped is the quantity of a standing order after its caps and the units each
// cap cut, counted in TreasuryOrdered (spec-fiscal.md 9.3.2).
type Capped struct {
	Qty      int64 // the quantity to place; an order is placed only if it is at least 1
	CutCash  int64 // purchase: units cut because the cash does not buy them at the limit
	CutHold  int64 // purchase: units cut because the hold would be passed
	CutStock int64 // release: units cut because the unreserved stock does not cover them
}

// CapPurchase applies the caps of a purchase in the order of spec-fiscal.md
// 9.3.2: the cash, floor(cash / limit), then the hold, hold - held when hold is
// above 0. A limit below 1 leaves nothing to buy.
func CapPurchase(wish, cash, limit, hold, held int64) Capped {
	c := Capped{Qty: max(0, wish)}
	byCash := int64(0)
	if limit >= 1 {
		byCash = max(0, cash) / limit
	}
	if c.Qty > byCash {
		c.CutCash, c.Qty = c.Qty-byCash, byCash
	}
	if hold > 0 {
		if room := max(0, hold-max(0, held)); c.Qty > room { // a negative held counts as 0: hold - held could leave int64
			c.CutHold, c.Qty = c.Qty-room, room
		}
	}
	return c
}

// CapRelease applies the cap of a release (spec-fiscal.md 9.3.2): the unreserved
// stock, held - reserve, never negative.
func CapRelease(wish, held, reserve int64) Capped {
	c := Capped{Qty: max(0, wish)}
	if stock := max(0, held) - max(0, reserve); c.Qty > max(0, stock) { // negatives count as 0: held - reserve could leave int64
		c.CutStock, c.Qty = c.Qty-max(0, stock), max(0, stock)
	}
	return c
}
