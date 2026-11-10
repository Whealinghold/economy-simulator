package merchant

import "github.com/Whealinghold/economy-sim/internal/checked"

// Escrow returns price * qty, the money a contract holds from formation to
// settlement (spec-merchants.md 9.5). ok is false on overflow.
func Escrow(price, qty int64) (int64, bool) { return checked.Mul(price, qty) }

// Settle returns the two amounts of the settlement of a contract on arrival
// (spec-merchants.md 9.7): out = price * reached to the deliverer and refund =
// price * (spoiled + lost) back to the payer, or to the estate of the destination
// when the payer is gone. reached is the units that arrived, which counts a cargo
// diverted to the estate, and reached + spoiled + lost must equal qty. out +
// refund = price * qty exactly. ok is false when the units do not add up, one is
// negative or the escrow price * qty does not fit; the two parts are then at most
// the escrow, so neither product can overflow.
func Settle(price, qty, reached, spoiled, lost int64) (out, refund int64, ok bool) {
	if price < 0 || reached < 0 || spoiled < 0 || lost < 0 {
		return 0, 0, false
	}
	sum, ok := checked.Add(reached, spoiled)
	if ok {
		sum, ok = checked.Add(sum, lost)
	}
	if !ok || sum != qty {
		return 0, 0, false
	}
	if _, ok = checked.Mul(price, qty); !ok { // the escrow itself did not fit
		return 0, 0, false
	}
	return price * reached, price * (spoiled + lost), true
}
