package economysim

import (
	"maps"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file is the removal probe of spec-integration.md 4.2, split from integration.go (plan 2-17 D547).

// estateFits is the read-only probe of a removal (plan 2-17 D547): whether the
// removal can pass no balance bound or flow total. The ledger has no undo, and the
// removal changes the job before the intake. It covers what removeAgent moves: the
// escrow of the contracts the agent delivers goes back to the payer or the estate
// of the destination (dropContracts), then the agent's money and goods go to the
// estate of its region (estates.receive).
func (w *World) estateFits(a *agent) bool {
	region := w.regionOf(a)
	estate := w.estates.account(region)
	mine, err := w.ledger.Account(a.account)
	if err != nil {
		invariant("the account of an agent being removed is unreadable: " + err.Error())
	}
	// credit is what each account receives and refund what each region's refund flow
	// adds, so two contracts into one account are tested together. Only whether every
	// entry fits is read, and the keys are walked in ascending order so that a ledger
	// fault is the same one every run.
	credit := map[int]int64{estate: mine.Balance}
	refund := map[int]int64{}
	toEstate := w.contractRefunds(a, estate, mine.Balance, credit, refund)
	return w.creditsFit(credit) && w.refundsFit(refund) &&
		(toEstate <= 0 || w.ledger.FlowFits(int(region), ledger.EstateMoney, ledger.Money, toEstate)) &&
		w.goodsFit(a, region, estate)
}

// contractRefunds adds the escrow of the contracts the agent delivers to credit and
// refund and returns what the estate of the agent's region gets by EstateMoney
// (start plus the refunds that pass through the agent's own account).
func (w *World) contractRefunds(a *agent, estate int, start int64, credit, refund map[int]int64) int64 {
	toEstate := start
	if w.mer == nil || !w.mer.on {
		return toEstate
	}
	for _, ct := range w.mer.open {
		if ct.deliverer != a.id {
			continue
		}
		to := w.estates.account(ct.to)
		back := int64(ct.price) * int64(ct.qty)
		if ct.payer == a.id {
			to = estate // the refund passes through the agent's account to its estate
			toEstate += back
		} else if w.liveAgent(ct.payer) {
			to = w.agents[ct.payer].account
		}
		credit[to] += back
		refund[int(ct.to)] += back
	}
	return toEstate
}

// creditsFit is whether no account of credit passes the balance bound.
func (w *World) creditsFit(credit map[int]int64) bool {
	for _, acct := range slices.Sorted(maps.Keys(credit)) {
		v, err := w.ledger.Account(acct)
		if err != nil {
			invariant("an account a removal pays is unreadable: " + err.Error())
		}
		if credit[acct] > MaxMoney-v.Balance {
			return false
		}
	}
	return true
}

// refundsFit is whether no escrow refund flow of refund passes its total.
func (w *World) refundsFit(refund map[int]int64) bool {
	for _, g := range slices.Sorted(maps.Keys(refund)) {
		if !w.ledger.FlowFits(g, ledger.EscrowRefund, ledger.Money, refund[g]) {
			return false
		}
	}
	return true
}

// goodsFit is whether the goods of the agent fit the holding and the EstateGoods
// flow of the estate of its region.
func (w *World) goodsFit(a *agent, region RegionID, estate int) bool {
	lots, err := w.ledger.AppendLots(nil, a.account)
	if err != nil {
		invariant("the lots of an agent being removed are unreadable: " + err.Error())
	}
	held := make([]int64, len(w.cfg.Commodities))
	for _, lt := range lots {
		held[lt.Commodity] += lt.Quantity
	}
	for c, q := range held {
		if q == 0 {
			continue
		}
		have, _, err := w.ledger.Holding(estate, c)
		if err != nil {
			invariant("the estate holding is unreadable: " + err.Error())
		}
		if q > MaxQuantity-have || !w.ledger.FlowFits(int(region), ledger.EstateGoods, c, q) {
			return false
		}
	}
	return true
}
