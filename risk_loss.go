package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	rsk "github.com/Whealinghold/economy-sim/internal/risk"
)

// This file is the loss of the risk module (spec-risk.md 5): step 0 of an arrival,
// the loss of the Transit account and then of the traveller's own account, applied
// after the guard of the arrival and before the traveller's region change.

// arrivalLoss is step 0 of an arrival (spec-risk.md 5.1.1): with a loss share above
// 0, rhe(held * LossBP, 10 000) units of each commodity the Transit account holds, in
// commodity order, move to the boundary under loss, oldest lots first; then, for a
// self-carried shipment whose traveller still exists, the same over every commodity
// of the traveller's account but the mode's vehicle (5.4.1). The units lost of each
// line and of the account are kept for the delivery and the event. It runs once per
// arrival: the guard of the arrival comes first, so a held arrival applies nothing
// (5.1.3). It takes no word.
func (w *World) arrivalLoss(s *shipment) {
	r := w.risk
	r.lost, r.accountLost = r.lost[:0], r.accountLost[:0]
	if s.loss == 0 {
		return
	}
	for _, ln := range s.lines {
		held, _, err := w.ledger.Holding(s.account, int(ln.Commodity))
		if err != nil {
			invariant("the ledger refused the holding of a Transit account at an arrival: " + err.Error())
		}
		lost, ok := rsk.Line(held, s.loss)
		if !ok {
			invariant("the loss of a line did not fit after a checked shipment")
		}
		w.loseGoods(s.account, ln.Commodity, lost)
		r.lost = append(r.lost, lost)
	}
	if w.travellerLive(s) {
		w.tripLoss(s)
	}
}

// tripLoss takes the share of the traveller's own account (spec-risk.md 5.4.1): of
// every commodity it holds but the vehicle of the mode, which is capital and not
// cargo. The units lost are recorded in commodity order.
func (w *World) tripLoss(s *shipment) {
	r := w.risk
	acct := w.agents[s.traveller].account
	r.held = r.held[:0]
	for c := range w.cfg.Commodities {
		held, _, err := w.ledger.Holding(acct, c)
		if err != nil {
			invariant("the ledger refused the holding of a traveller's account at an arrival: " + err.Error())
		}
		r.held = append(r.held, held)
	}
	var ok bool
	if r.tripLost, ok = rsk.Account(r.tripLost, r.held, int(w.transport.modes[s.mode].vehicle), s.loss); !ok {
		invariant("the loss of a traveller's account did not fit after a checked holding")
	}
	for c, lost := range r.tripLost {
		if lost > 0 {
			w.loseGoods(acct, CommodityID(c), lost)
			r.accountLost = append(r.accountLost, Amount{Commodity: CommodityID(c), Quantity: Quantity(lost)})
		}
	}
}

// loseGoods moves qty units of a commodity from an account to the boundary of the
// reason loss, oldest lots first as the ledger orders them (spec-risk.md 5.3.3).
func (w *World) loseGoods(account int, c CommodityID, qty int64) {
	if qty == 0 {
		return
	}
	sink, ok := w.ledger.Boundary(ledger.Loss, int(c))
	if !ok {
		invariant("the risk module has no boundary account of loss for a commodity")
	}
	if err := w.ledger.Move(account, sink, int(c), qty, ledger.Loss); err != nil {
		invariant("the ledger refused a checked loss: " + err.Error())
	}
}

// lostOf returns the units lost of the line at index i of the arriving shipment,
// 0 with the module off or no strike. A nil receiver is the module off.
func (r *risk) lostOf(i int) int64 {
	if r == nil || i >= len(r.lost) {
		return 0
	}
	return r.lost[i]
}

// accountLostCopy returns a copy of what the traveller's account lost at the arrival,
// for ShipmentArrived: nil when nothing was lost or the module is off.
func (r *risk) accountLostCopy() []Amount {
	if r == nil || len(r.accountLost) == 0 {
		return nil
	}
	return slices.Clone(r.accountLost)
}
