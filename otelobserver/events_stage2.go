package otelobserver

import economysim "github.com/Whealinghold/economy-sim"

// observeFlows handles the events of the money flows, the wants, the population,
// the relocations and the information module.
func (o *Observer) observeFlows(e economysim.Event) bool {
	switch ev := e.(type) {
	case economysim.AgentRelocated:
		o.agentRelocated(ev)
	case economysim.WantShortfall:
		o.count(m2(iWantUnfunded, ix(ev.Want), ix(ev.Role)), satSub(ev.Planned, ev.Funded))
		o.count(m2(iWantChoked, ix(ev.Want), ix(ev.Role)), ev.Choked)
	case economysim.MoneyFlowLimited:
		o.count(m1(iMoneyFlowUnapplied, o.reason(ev.Flow)), satSub(int64(ev.Due), int64(ev.Applied)))
		o.counts(m2(iMoneyFlowLimited, o.reason(ev.Flow), 0), 1, ev.Short, ev.Headroom, ev.Limit)
	case economysim.PayrollPaid:
		o.count(m2(iPayrollPaid, ix(ev.Role), ix(ev.Region)), int64(ev.Paid))
		o.count(m2(iPayrollCut, ix(ev.Role), ix(ev.Region)), int64(ev.Cut))
	case economysim.PopulationChanged:
		o.count(m1(iHouseholdBirths, ix(ev.Role)), ev.BirthsDone)
		o.count(m1(iHouseholdDeaths, ix(ev.Role)), ev.DeathsDone)
		o.counts(m1(iPopulationLimited, 0), 0, ev.DeathsCut, ev.DeathsHeld, ev.BirthsCapped, ev.BirthsUnfunded, ev.RateClamped)
	case economysim.RumoursServed:
		region := ix(ev.Region)
		o.counts(m2(iInformationRumours, 0, region), 0, ev.Offered, ev.NoPath, ev.TooOld, ev.NotYet)
		o.count(m1(iInformationReaders, region), ev.Readers)
	default:
		return false
	}
	return true
}

// agentRelocated counts the households moved by pair of regions and cause; a
// journey that ends is off the road (spec-observability.md 6.2.1).
func (o *Observer) agentRelocated(ev economysim.AgentRelocated) {
	cause := closedIndex(keyCause, string(ev.Cause))
	o.count(m3(iHouseholdRelocations, ix(ev.From), ix(ev.To), cause), ev.Weight)
	if cause == causeMigration {
		o.count(m0(iMigrationOnJourney), -ev.Weight)
	}
}

// observeResources handles the events of the endowments, the own-use mode and
// the outside nodes.
func (o *Observer) observeResources(e economysim.Event) bool {
	switch ev := e.(type) {
	case economysim.EndowmentChanged:
		o.endowmentChanged(ev)
	case economysim.EndowmentAdjusted:
		o.set(m2(iEndowmentStock, ix(ev.Resource), ix(ev.Region)), ev.StockAfter)
	case economysim.OwnUseChanged:
		o.set(m2(iHouseholdOwnUse, ix(ev.Role), ix(ev.Region)), ev.OnOwnUse)
		o.count(m1(iOwnUseBlocked, ix(ev.Role)), ev.Blocked)
		o.counts(m2(iHouseholdOwnUseChanges, ix(ev.Role), 0), 1, ev.Entered, ev.Left)
		for _, a := range ev.Output {
			o.count(m1(iOwnUseUnits, ix(a.Commodity)), int64(a.Quantity))
		}
	case economysim.OutsideTraded:
		o.outsideTraded(ev)
	default:
		return false
	}
	return true
}

func (o *Observer) endowmentChanged(ev economysim.EndowmentChanged) {
	res, reg := ix(ev.Resource), ix(ev.Region)
	o.count(m2(iEndowmentDrawn, res, reg), ev.Drawn)
	o.count(m2(iEndowmentRegenerated, res, reg), ev.Regenerated)
	o.count(m2(iEndowmentShort, res, reg), ev.Short)
	o.set(m2(iEndowmentStock, res, reg), ev.Stock)
}

// outsideTraded counts the units and the money of an outside line and holds the
// price it quoted, by node, gate region, commodity and side.
func (o *Observer) outsideTraded(ev economysim.OutsideTraded) {
	node, gate, c, side := ix(ev.Node), ix(ev.Region), ix(ev.Commodity), ix(ev.Side)
	o.count(m3(iExternalUnits, node, c, side), ev.Traded)
	o.count(m3(iExternalValue, node, c, side), int64(ev.Value))
	if ev.Price > 0 { // 0 is no quote (a closed edge), which external.cut counts
		o.set(m4(iExternalPrice, node, gate, c, side), int64(ev.Price))
	}
	o.counts(m4(iExternalCut, node, c, side, 0), 3, ev.CutClosed, ev.CutCap, ev.CutPrice)
	o.count(m3(iExternalUnfilled, node, c, side), satSub(ev.Offered, ev.Traded))
	if ev.PriceClamped {
		o.count(m2(iExternalClamped, node, c), 1)
	}
}

// observeTransport handles the events of the shipments and the contracts.
func (o *Observer) observeTransport(e economysim.Event) bool {
	switch ev := e.(type) {
	case economysim.ShipmentDispatched:
		o.count(m1(iTransportShipments, ix(ev.Mode)), 1)
		for _, a := range ev.Lines {
			o.count(m1(iTransportInTransit, ix(a.Commodity)), int64(a.Quantity))
		}
	case economysim.ShipmentArrived:
		for _, l := range ev.Lines {
			o.count(m1(iTransportInTransit, ix(l.Commodity)), -int64(l.Shipped))
			o.count(m1(iTransportCargoSpoiled, ix(l.Commodity)), int64(l.Spoiled))
			o.count(m1(iTransportCargoDiverted, ix(l.Commodity)), int64(l.Diverted))
		}
	case economysim.ShipmentRejected:
		o.count(m2(iTransportRejected, ix(ev.Mode), o.reason(string(ev.Reason))), 1)
	case economysim.ContractFormed:
		o.count(m1(iContractFormed, ix(ev.Commodity)), 1)
		o.count(m1(iContractValue, ix(ev.Commodity)), int64(ev.Escrowed))
		o.count(m0(iContractEscrow), int64(ev.Escrowed))
	case economysim.ContractSettled:
		o.count(m0(iContractEscrow), satSub(satSub(0, int64(ev.Paid)), int64(ev.Refunded)))
		o.count(m1(iContractSettled, closedIndex(keyCause, string(ev.Cause))), 1)
	case economysim.ContractRound:
		o.counts(m1(iContractUnformed, 0), 0, ev.NoTaker, ev.PayerFunds, ev.TooMany, ev.DispatchRefused)
		o.counts(m1(iContractRefused, 0), 0, ev.OffersRefused, ev.TakesRefused)
	default:
		return false
	}
	return true
}

// observeMigration handles the events of the migration module and the merchants.
func (o *Observer) observeMigration(e economysim.Event) bool {
	switch ev := e.(type) {
	case economysim.MigrationStarted:
		o.count(m2(iMigrationStarted, ix(ev.From), ix(ev.To)), ev.Weight)
		o.count(m0(iMigrationOnJourney), ev.Weight)
	case economysim.MigrationReviewed:
		o.counts(m2(iMigrationLimited, 0, ix(ev.Region)), 0, ev.NoEstimate, ev.NoDestination, ev.NoRumour, ev.TooFar,
			ev.NoViable, ev.NoGain, ev.Bound, ev.OutCap, ev.Unprovisioned, ev.InCap, ev.Held, ev.Ended)
		o.count(m0(iMigrationOnJourney), -ev.Ended)
		o.count(m0(iMigrationUnsettled), ev.Unsettled)
	case economysim.MerchantsPlanned:
		o.counts(m1(iMerchantDecisions, 0), 0, ev.NoRumour, ev.Stale, ev.Locked, ev.Perishable, ev.NoMargin,
			ev.Unaffordable, ev.BelowShare, ev.Planned, ev.Rested, ev.Waiting, ev.LotCapped)
	default:
		return false
	}
	return true
}

// seedStage2 sets the levels of the modules from the snapshot at the restored
// boundary (spec-observability.md 6.4.2): the units in flight, the households on
// a journey, the escrow, and the held gauges of the endowments, the own-use mode
// and the outside quotes.
func (o *Observer) seedStage2(s economysim.Snapshot) {
	for _, sh := range s.Shipments {
		for _, a := range sh.Lines {
			o.count(m1(iTransportInTransit, ix(a.Commodity)), int64(a.Quantity))
		}
	}
	for _, j := range s.Journeys {
		o.count(m0(iMigrationOnJourney), j.Weight)
	}
	o.count(m0(iContractEscrow), int64(s.Escrow))
	for _, e := range s.Endowments {
		o.set(m2(iEndowmentStock, ix(e.Resource), ix(e.Region)), e.Stock)
	}
	for _, r := range s.Roles {
		o.set(m2(iHouseholdOwnUse, ix(r.Role), ix(r.Region)), r.OnOwnUse)
	}
	for _, q := range s.Outside.Sides {
		o.set(m4(iExternalPrice, ix(q.Node), ix(q.Gate), ix(q.Commodity), ix(q.Side)), int64(q.Quote))
	}
}
