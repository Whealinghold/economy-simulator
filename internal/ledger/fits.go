package ledger

import "github.com/Whealinghold/economy-sim/internal/checked"

// Fund is a quantity of one commodity to create into an account.
type Fund struct {
	Commodity int
	Qty       int64
}

// FundFits reports whether creating money and goods under Initial into a new,
// empty account of the region would pass no bound: the debited boundary, the
// credited balance (MaxMoney, MaxQuantity) and the region's flow total of the
// phase in progress, each as Transfer and Move check them. It reads and changes
// nothing, so a caller can refuse before it opens an account (the ledger has no
// undo). An amount below 1 is skipped, as the funding skips it; a region or a
// commodity that is out of range, or has no boundary account, does not fit.
func (l *Ledger) FundFits(region int, money int64, goods []Fund) bool {
	if region < 0 || region >= l.regions {
		return false
	}
	if money >= 1 && !l.fits(region, Money, money, l.maxMoney) {
		return false
	}
	for _, g := range goods {
		if g.Qty < 1 {
			continue
		}
		if g.Commodity < 0 || g.Commodity >= len(l.comms) || !l.fits(region, g.Commodity, g.Qty, l.maxQty) {
			return false
		}
	}
	return true
}

// fits checks one creation under Initial of amount of a commodity (Money for
// money) into a new account against the bound of the credited side, the debited
// boundary and the flow total.
func (l *Ledger) fits(region, commodity int, amount, bound int64) bool {
	b, ok := l.Boundary(Initial, commodity)
	if !ok || amount > bound {
		return false
	}
	off := l.offOf(Initial, thingMoney)
	if commodity != Money {
		off = l.offOf(Initial, thingGoods)
		if off >= 0 {
			off += commodity
		}
	}
	if off < 0 {
		return false
	}
	if _, ok := checked.Sub(l.accounts[b].balance, amount); !ok {
		return false
	}
	_, ok = checked.Add(l.flows()[region*l.stride+off], amount)
	return ok
}

// FlowFits reports whether adding amount to the flow total of reason in the region
// (commodity Money for money, else a commodity) passes no bound, as Transfer and
// Move check it. It reads and changes nothing. A region out of range, a commodity
// out of range or a reason with no flow slot does not fit.
func (l *Ledger) FlowFits(region int, r Reason, commodity int, amount int64) bool {
	if region < 0 || region >= l.regions {
		return false
	}
	off := l.offOf(r, thingMoney)
	if commodity != Money {
		if commodity < 0 || commodity >= len(l.comms) {
			return false
		}
		if off = l.offOf(r, thingGoods); off >= 0 {
			off += commodity
		}
	}
	if off < 0 {
		return false
	}
	_, ok := checked.Add(l.flows()[region*l.stride+off], amount)
	return ok
}
