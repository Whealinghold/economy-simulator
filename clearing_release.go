package economysim

import (
	"fmt"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/clearing"
	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// release returns the remainder of every reservation of the book: the money of
// a bid not spent and the units of an ask not sold, given the units filled per
// slot (nil: nothing was filled). A remainder of 0 is not released, because
// the ledger refuses an amount below 1 (D33).
func (b *book) release(l *ledger.Ledger, sums []int64) {
	for i := range b.entries {
		e := &b.entries[i]
		var filled int64
		if sums != nil {
			filled = sums[e.slot]
		}
		rem := int64(e.qty) - filled
		if rem < 1 {
			continue
		}
		var err error
		switch {
		case e.side == Bid:
			// limit * remaining <= limit * quantity, which fit at admission.
			money, ok := checked.Mul(int64(e.price), rem)
			if !ok {
				invariant("the remainder of an admitted bid has a product that overflows")
			}
			err = l.ReleaseMoney(e.account, money)
			if err == nil && e.roomed {
				err = b.releaseRoom(l, e, rem)
			}
		case e.lot.Named:
			err = l.ReleaseLot(e.account, ledger.Key{Commodity: int(b.commodity), Born: e.lot.Born, Durability: e.lot.Durability}, rem)
		default:
			err = l.ReleaseGoods(e.account, int(b.commodity), rem)
		}
		if err != nil {
			invariant("the ledger refused the release of a reservation the book holds: " + err.Error())
		}
	}
}

// releaseRoom returns the room of the unmatched part rem of a bid that reserved it
// (spec-storage.md 5.2.2, FR-STO-009).
func (b *book) releaseRoom(l *ledger.Ledger, e *bookEntry, rem int64) error {
	mass, okm := checked.Mul(rem, e.sizeMass)
	bulk, okb := checked.Mul(rem, e.sizeBulk)
	if !okm || !okb {
		invariant("the room of the remainder of an admitted bid overflows")
	}
	return l.ReleaseRoom(e.account, mass, bulk)
}

// tradeListError is the error of a list of trades that fails the check of
// spec-clearing.md 8.2. It wraps ErrInvalidTrade and names the market.
type tradeListError struct {
	tick      int64
	region    RegionID
	commodity CommodityID
	what      string
}

// Error describes the market and the rule broken.
func (e *tradeListError) Error() string {
	return fmt.Sprintf("%s: tick %d region %d commodity %d: %s",
		ErrInvalidTrade.Error(), e.tick, e.region, e.commodity, e.what)
}

// Unwrap returns ErrInvalidTrade.
func (e *tradeListError) Unwrap() error { return ErrInvalidTrade }

// listFailure describes a failure code of clearing.Check.
func listFailure(c clearing.Code) string {
	switch c {
	case clearing.BadCommodity:
		return "a trade names another commodity than the market's"
	case clearing.NoOrder:
		return "a trade names a buyer with no bid or a seller with no ask"
	case clearing.BadQuantity:
		return "a trade has a quantity below 1"
	case clearing.BadPrice:
		return "a trade price is outside the limits of its orders"
	case clearing.SameOwner:
		return "a trade has one agent as buyer and seller"
	case clearing.OverFilled:
		return "the trades of an order exceed its quantity"
	}
	return "unknown failure" // a Code Check does not return; kept so a new code is not silent
}

// mustSettle panics when the ledger refuses a step of settlement. The list
// passed the check and admission closed the overflow cases (D33, D34), so a
// refusal is a violated invariant, never an input.
func mustSettle(err error, step string) {
	if err != nil {
		invariant("the ledger refused " + step + " of a checked trade: " + err.Error())
	}
}
