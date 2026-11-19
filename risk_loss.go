package economysim

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	rsk "github.com/Whealinghold/economy-sim/internal/risk"
	"github.com/Whealinghold/economy-sim/internal/room"
)

// This file is the loss of the risk module (spec-risk.md 5, 6): step 0 of an arrival,
// the loss of the Transit account and then of the traveller's own account, applied
// after the guard of the arrival and before the traveller's region change; and the
// store side, P8.1a, which decides every strike first and applies the losses after.

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

// storeStrike is a store class of one agent struck at P8.1a (spec-risk.md 6.3): the
// stores of the class struck, whether the row is common, the share of the account's
// goods the class takes and the range of the amounts it lost in the scratch.
type storeStrike struct {
	agent  AgentID
	region RegionID
	class  StoreClassID
	stores int64
	common bool
	share  int64
	lo, hi int
}

// p81aStores is P8.1a (spec-risk.md 6.2, 6.3): the common store rows draw, then the
// stores of each limited agent in ascending agent ID, and only then are the losses
// applied, so the loss of an earlier agent changes no later draw or share. A tick in
// which no store row has a probability above 0 reads nothing but the rows. The
// StoreLoss events come after the whole position is applied.
func (w *World) p81aStores() {
	r := w.risk
	r.strikes, r.amounts = r.strikes[:0], r.amounts[:0]
	if !w.commonStoreOutcomes() {
		return
	}
	for id, counts := range w.sto.stores {
		if counts != nil {
			w.decideStores(&w.agents[id])
		}
	}
	w.applyStrikes()
	w.reportStrikes()
}

// commonStoreOutcomes draws the outcome of each common store row with a probability
// above 0, in canonical order, one word each (D15 (i)), and clears the outcome of
// every other row. It reports whether any row has a probability above 0.
func (w *World) commonStoreOutcomes() (live bool) {
	tick := w.Tick()
	for i := range w.risk.stores {
		row := &w.risk.stores[i]
		row.struck = false
		if row.p == 0 {
			continue
		}
		live = true
		if row.common {
			w.rnd.keyedPair(siteRiskStoreCommon, tick, int64(row.region), int64(row.class))
			row.struck = rsk.Struck(w.rnd.draw(wholeBP), row.p)
		}
	}
	return live
}

// decideStores decides the strikes of one limited agent (spec-risk.md 6.2.1 step 2):
// for each class it owns with a row of its region with a probability above 0, in
// ascending class, one word for each store of a unit row (D15 (ii)), and a strike for
// the stores struck. An agent with no stored load is not drawn and is not struck. The
// stored fraction is read from the state at the start of the position.
func (w *World) decideStores(a *agent) {
	if !w.sto.limited(a) {
		return
	}
	r := w.risk
	region := w.regionOf(a)
	var stored, load, sumCap room.Pair
	exposed := false
	for class, n := range w.sto.stores[a.id] {
		row := r.findStore(region, StoreClassID(class))
		if n == 0 || row == nil || row.p == 0 {
			continue
		}
		if !exposed {
			if stored, load, sumCap = w.storeExposure(a); stored == (room.Pair{}) {
				return
			}
			exposed = true
		}
		hit := w.storesStruck(a, row, int64(n))
		if hit == 0 {
			continue
		}
		phi := w.storeFraction(stored, sumCap, w.sto.classes[class], load)
		share, ok := rsk.Share(hit, phi, row.severityBP)
		if !ok {
			invariant("the share of a struck store class did not fit after a checked fraction")
		}
		r.strikes = append(r.strikes, storeStrike{agent: a.id, region: region, class: row.class, stores: hit, common: row.common, share: share})
	}
}

// storesStruck is the number of an agent's n stores of a row struck: all of them for
// a common row that struck, none for one that did not, and for a unit row one word of
// the stream keyed by the tick, the agent and the class for each store.
func (w *World) storesStruck(a *agent, row *storeRow, n int64) (hit int64) {
	if row.common {
		if row.struck {
			return n
		}
		return 0
	}
	w.rnd.keyedPair(siteRiskStore, w.Tick(), int64(a.id), int64(row.class))
	for range n {
		if rsk.Struck(w.rnd.draw(wholeBP), row.p) {
			hit++
		}
	}
	return hit
}

// storeExposure returns the stored load of an agent's account, its load and the sum of
// the capacities of its stores (spec-risk.md 6.1.2). A zero stored load is no
// exposure, and so is a sum that does not fit int64, which storedLoad treats alike.
func (w *World) storeExposure(a *agent) (stored, load, sumCap room.Pair) {
	rm, _, ok := w.sto.roomState(w.ledger, a)
	if !ok {
		return room.Pair{}, room.Pair{}, room.Pair{}
	}
	stored = w.storedLoad(a)
	for class, n := range w.sto.stores[a.id] {
		m, okm := checked.Mul(w.sto.classes[class].Mass, int64(n))
		b, okb := checked.Mul(w.sto.classes[class].Bulk, int64(n))
		var oks bool
		if sumCap, oks = addPair(sumCap, room.Pair{Mass: m, Bulk: b}); !okm || !okb || !oks {
			return room.Pair{}, room.Pair{}, room.Pair{}
		}
	}
	return stored, room.Pair{Mass: rm.LoadMass, Bulk: rm.LoadBulk}, sumCap
}

// storeFraction is phi of a store of a class (spec-risk.md 6.1.2): the larger of the
// fractions of the account's load in the two dimensions.
func (w *World) storeFraction(stored, sumCap, capK, load room.Pair) int64 {
	mass, okm := rsk.Phi(stored.Mass, sumCap.Mass, capK.Mass, load.Mass)
	bulk, okb := rsk.Phi(stored.Bulk, sumCap.Bulk, capK.Bulk, load.Bulk)
	if !okm || !okb {
		invariant("the stored fraction of a store class did not fit after a checked load")
	}
	return rsk.Larger(mass, bulk)
}

// applyStrikes applies the decided strikes (spec-risk.md 6.3.2): an agent's classes in
// ascending class, each taking rhe(q0 * share, 10 000) of each commodity from the
// account to the boundary under loss, oldest lots first, with q0 the quantity at the
// start of the position and the total taken of a commodity capped at q0. Earmarked
// units are not goods that can be lost, so q0 is the units not earmarked.
func (w *World) applyStrikes() {
	r := w.risk
	for i := 0; i < len(r.strikes); {
		j := i
		for j < len(r.strikes) && r.strikes[j].agent == r.strikes[i].agent {
			j++
		}
		w.applyAgent(&w.agents[r.strikes[i].agent], r.strikes[i:j])
		i = j
	}
}

// applyAgent applies the strikes of one agent, in class order.
func (w *World) applyAgent(a *agent, strikes []storeStrike) {
	r := w.risk
	r.held, r.tripLost = r.held[:0], r.tripLost[:0]
	for c := range w.cfg.Commodities {
		held, reserved, err := w.ledger.Holding(a.account, c)
		if err != nil {
			invariant("the ledger refused the holding of a struck account: " + err.Error())
		}
		r.held = append(r.held, held-reserved)
		r.tripLost = append(r.tripLost, 0)
	}
	for k := range strikes {
		st := &strikes[k]
		st.lo = len(r.amounts)
		for c, q0 := range r.held {
			loss, ok := rsk.Apply(q0, st.share)
			if !ok {
				invariant("the loss of a struck store class did not fit after a checked share")
			}
			if loss = rsk.Capped(q0, r.tripLost[c], loss); loss > 0 {
				w.loseGoods(a.account, CommodityID(c), loss)
				r.tripLost[c] += loss
				r.amounts = append(r.amounts, Amount{Commodity: CommodityID(c), Quantity: Quantity(loss)})
			}
		}
		st.hi = len(r.amounts)
	}
}

// reportStrikes tells the strikes of the position, in ascending agent then class, after
// the whole position has been applied (spec-risk.md 6.4.1): one StoreLoss each, built
// only when an observer was supplied, and a Debug line each.
func (w *World) reportStrikes() {
	r := w.risk
	for i := range r.strikes {
		st := &r.strikes[i]
		w.logStoreStruck(st)
		if !w.obs.supplied {
			continue
		}
		var lost []Amount
		if st.hi > st.lo {
			lost = slices.Clone(r.amounts[st.lo:st.hi])
		}
		w.obs.deliver(StoreLoss{Tick: w.Tick(), Agent: st.agent, Region: st.region, Class: st.class, Stores: st.stores, Common: st.common, ShareBP: st.share, Lost: lost})
	}
}
