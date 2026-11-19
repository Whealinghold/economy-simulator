package indicator

import (
	"cmp"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file folds the events of the risk module into the tally (spec-indicators.md 9.1
// row 25, FR-IND-059): the units a shipment lost in transit and from the traveller's
// own account (the Lost and AccountLost of ShipmentArrived), the shipments dispatched
// with a loss share (DispatchTally.Struck) and the stores struck (StoreLoss). The
// ContractSettled Lost is the same units as a ShipmentArrived line and is not read.

// dispatchStruck counts a dispatched shipment whose loss share is above 0.
func (t *Tally) dispatchStruck(d *DispatchTally, e economysim.ShipmentDispatched) {
	if e.LossBP > 0 {
		t.add(&d.Struck, 1)
	}
}

// arrivalKnown reports whether every line and every amount the account of an arrival
// lost names a commodity of the table and no amount of the account is negative; a bad
// one drops the whole event.
func (t *Tally) arrivalKnown(e economysim.ShipmentArrived) bool {
	if !t.known(len(e.Lines), func(i int) economysim.CommodityID { return e.Lines[i].Commodity }) ||
		!t.known(len(e.AccountLost), func(i int) economysim.CommodityID { return e.AccountLost[i].Commodity }) {
		return false
	}
	for _, a := range e.AccountLost {
		if !t.nonneg(int64(a.Quantity)) {
			return false
		}
	}
	return true
}

// accountLost adds what the traveller's own account lost at an arrival to the row of
// its commodity on the lane, in the origin, where the account was (spec-risk.md 7.2.1).
func (t *Tally) accountLost(e economysim.ShipmentArrived) {
	for _, l := range e.AccountLost {
		if l.Quantity == 0 {
			continue
		}
		k := ArrivalTally{Origin: e.Region, Destination: e.To, Commodity: l.Commodity}
		t.add(&upsert(&t.w.Arrivals, k, arrivalKey).AccountLost, int64(l.Quantity))
	}
}

func lossKey(r LossTally, k LossTally) int {
	return cmp.Or(cmp.Compare(r.Region, k.Region), cmp.Compare(r.Class, k.Class), cmp.Compare(r.Commodity, k.Commodity))
}

func strikeKey(r StrikeTally, k StrikeTally) int {
	return cmp.Or(cmp.Compare(r.Region, k.Region), cmp.Compare(r.Class, k.Class))
}

// storeLossOK reports whether a StoreLoss names a region and a class of the window, no
// negative count and commodities of the table, and takes the step it belongs to; a bad
// one marks the window stray.
func (t *Tally) storeLossOK(e economysim.StoreLoss) bool {
	ok := e.Class >= 0 && e.Region >= 0 && int(e.Region) < t.w.Regions &&
		t.known(len(e.Lost), func(i int) economysim.CommodityID { return e.Lost[i].Commodity }) && t.nonneg(e.Stores)
	for _, a := range e.Lost {
		ok = ok && t.nonneg(int64(a.Quantity))
	}
	if !ok || !t.step(e.Tick) {
		t.stray = true
		return false
	}
	return true
}

// storeLoss folds a StoreLoss: the strike by (region, class), and each amount lost by
// (region, class, commodity). A strike with no amount is counted and takes no unit.
func (t *Tally) storeLoss(e economysim.StoreLoss) {
	if !t.storeLossOK(e) {
		return
	}
	s := upsert(&t.w.StoreStrikes, StrikeTally{Region: e.Region, Class: e.Class}, strikeKey)
	t.add(&s.Events, 1)
	t.add(&s.Stores, e.Stores)
	for _, a := range e.Lost {
		if a.Quantity > 0 {
			t.add(&upsert(&t.w.StoreLosses, LossTally{Region: e.Region, Class: e.Class, Commodity: a.Commodity}, lossKey).Units, int64(a.Quantity))
		}
	}
}
