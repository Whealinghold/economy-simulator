package otelobserver_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestRelocations_ByPairAndCause verifies that households relocated are counted
// by the two regions and the cause, and that a relocation of cause migration
// takes its weight off the journey level.
//
// Requirement: FR-OBS-050
func TestRelocations_ByPairAndCause(t *testing.T) {
	h := newHarness(t)
	h.send(created(), economysim.MigrationStarted{From: 0, To: 1, Weight: 5})
	tick(h, 0,
		economysim.AgentRelocated{From: 0, To: 1, Weight: 3, Cause: economysim.CauseTransport},
		economysim.AgentRelocated{From: 0, To: 1, Weight: 2, Cause: economysim.CauseMigration},
		economysim.AgentRelocated{From: 1, To: 2, Weight: 4, Cause: economysim.CauseConsumer},
		economysim.AgentRelocated{From: 1, To: 2, Weight: 4, Cause: "other"})
	got := h.collect()
	check(t, got, "household.relocations", 3, "region.from", "r0", "region.to", "r1", "cause", "transport")
	check(t, got, "household.relocations", 2, "region.from", "r0", "region.to", "r1", "cause", "migration")
	check(t, got, "household.relocations", 4, "region.from", "r1", "region.to", "r2", "cause", "consumer")
	if n := len(got["economysim.household.relocations"]); n != 3 {
		t.Errorf("household.relocations series = %d, want 3: a cause outside the closed set is dropped", n)
	}
	check(t, got, "migration.on_journey", 3, "")
}

// TestWants_Unfunded verifies that the units planned and not funded are counted
// by want and role, and that a want funded in full or over adds nothing.
//
// Requirement: FR-OBS-050
func TestWants_Unfunded(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.WantShortfall{Want: 1, Role: 0, Planned: 10, Funded: 7},
		economysim.WantShortfall{Want: 1, Role: 0, Planned: 4, Funded: 4},
		economysim.WantShortfall{Want: 0, Role: 1, Planned: 3, Funded: 5})
	got := h.collect()
	check(t, got, "want.unfunded", 3, "want", "w1", "role", "k0")
	if n := len(got["economysim.want.unfunded"]); n != 1 {
		t.Errorf("want.unfunded series = %d, want 1", n)
	}
}

// TestMoneyFlow_Unapplied verifies that the money a flow computed and could not
// apply is counted by flow, as due less applied.
//
// Requirement: FR-OBS-050
func TestMoneyFlow_Unapplied(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.MoneyFlowLimited{Flow: "stipend", Region: 0, Due: 100, Applied: 60},
		economysim.MoneyFlowLimited{Flow: "stipend", Region: 1, Due: 50, Applied: 50},
		economysim.MoneyFlowLimited{Flow: "levy", Due: 9, Applied: 4})
	got := h.collect()
	check(t, got, "money.flow.unapplied", 40, "reason", "stipend")
	check(t, got, "money.flow.unapplied", 5, "reason", "levy")
}

// TestEndowment_FlowsAndStockHeld verifies the units drawn, regenerated and
// short, the stock after the tick, the stock after an adjustment, and that a tick
// with no event records the stock again.
//
// Requirement: FR-OBS-050
func TestEndowment_FlowsAndStockHeld(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.EndowmentChanged{Region: 1, Resource: 1, Stock: 90, Drawn: 10, Regenerated: 4, Short: 2},
		economysim.EndowmentChanged{Region: 1, Resource: 1, Stock: 85, Drawn: 5})
	before := h.calls["economysim.endowment.stock"]
	tick(h, 1)
	if n := h.calls["economysim.endowment.stock"] - before; n != 1 {
		t.Errorf("endowment.stock records in a silent tick = %d, want 1", n)
	}
	got := h.collect()
	check(t, got, "endowment.drawn", 15, "resource", "s1", "region", "r1")
	check(t, got, "endowment.regenerated", 4, "resource", "s1", "region", "r1")
	check(t, got, "endowment.short", 2, "resource", "s1", "region", "r1")
	check(t, got, "endowment.stock", 85, "resource", "s1", "region", "r1")
	h.send(economysim.EndowmentAdjusted{Region: 1, Resource: 1, StockAfter: 60})
	check(t, h.collect(), "endowment.stock", 60, "resource", "s1", "region", "r1")
}

// TestOwnUse_GaugeHeldAndUnits verifies the households in the own-use mode as a
// gauge held across a silent tick, and the units output by commodity.
//
// Requirement: FR-OBS-050
func TestOwnUse_GaugeHeldAndUnits(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.OwnUseChanged{Region: 2, Role: 1, OnOwnUse: 6,
		Output: []economysim.Amount{{Commodity: 0, Quantity: 4}, {Commodity: 2, Quantity: 1}}},
		economysim.OwnUseChanged{Region: 2, Role: 1, OnOwnUse: 7, Output: []economysim.Amount{{Commodity: 0, Quantity: 1}}})
	before := h.calls["economysim.household.own_use"]
	tick(h, 1)
	if n := h.calls["economysim.household.own_use"] - before; n != 1 {
		t.Errorf("household.own_use records in a silent tick = %d, want 1", n)
	}
	got := h.collect()
	check(t, got, "household.own_use", 7, "role", "k1", "region", "r2")
	check(t, got, "own_use.units", 5, "commodity", "c0")
	check(t, got, "own_use.units", 1, "commodity", "c2")
}

// TestPopulation_BirthsAndDeaths verifies that the households born and dead of
// the population module are counted by role, and that a death is not an exit.
//
// Requirement: FR-OBS-050, FR-OBS-059
func TestPopulation_BirthsAndDeaths(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.PopulationChanged{Role: 0, BirthsDone: 3, DeathsDone: 1},
		economysim.PopulationChanged{Role: 0, BirthsDone: 2},
		economysim.WeightChanged{Role: 0, From: 5, To: 4, Cause: economysim.CausePopulation})
	got := h.collect()
	check(t, got, "household.births", 5, "role", "k0")
	check(t, got, "household.deaths", 1, "role", "k0")
	absent(t, got, "household.exits", "role", "k0", "cause", "population")
}

// TestTransport_ShipmentsInTransitSpoiled verifies that a shipment is counted by
// mode, that the units in transit rise by each line at dispatch and fall by the
// units shipped at arrival, back to 0, and that the units spoiled are counted.
//
// Requirement: FR-OBS-050
func TestTransport_ShipmentsInTransitSpoiled(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	lines := []economysim.Amount{{Commodity: 0, Quantity: 5}, {Commodity: 1, Quantity: 3}}
	tick(h, 0, economysim.ShipmentDispatched{Mode: 1, Lines: lines})
	got := h.collect()
	check(t, got, "transport.shipments", 1, "mode", "m1")
	check(t, got, "transport.in_transit", 5, "commodity", "c0")
	check(t, got, "transport.in_transit", 3, "commodity", "c1")
	tick(h, 1, economysim.ShipmentArrived{Mode: 1, Lines: []economysim.ArrivedLine{
		{Commodity: 0, Shipped: 5, Delivered: 4, Spoiled: 1}, {Commodity: 1, Shipped: 3, Delivered: 3}}})
	got = h.collect()
	check(t, got, "transport.in_transit", 0, "commodity", "c0")
	check(t, got, "transport.in_transit", 0, "commodity", "c1")
	check(t, got, "transport.cargo.spoiled", 1, "commodity", "c0")
}

// TestInformation_Rumours verifies the four outcomes of the rumours offered.
//
// Requirement: FR-OBS-050
func TestInformation_Rumours(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.RumoursServed{Offered: 7, NoPath: 3, TooOld: 2, NotYet: 1},
		economysim.RumoursServed{Offered: 1})
	got := h.collect()
	for outcome, want := range map[string]int64{"offered": 8, "no_path": 3, "too_old": 2, "not_yet": 1} {
		check(t, got, "information.rumours", want, "outcome", outcome)
	}
}

// TestMigration_StartedLimitedOnJourney verifies the households started by pair,
// the twelve outcomes of a review, and the journey level: the weight at the start,
// less the weight of the relocation of cause migration, less the journeys that a
// removal ended.
//
// Requirement: FR-OBS-050
func TestMigration_StartedLimitedOnJourney(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.MigrationStarted{From: 0, To: 2, Weight: 6}, economysim.MigrationStarted{From: 0, To: 2, Weight: 1})
	check(t, h.collect(), "migration.on_journey", 7, "")
	tick(h, 1, economysim.AgentRelocated{From: 0, To: 2, Weight: 4, Cause: economysim.CauseMigration},
		economysim.MigrationReviewed{NoEstimate: 1, NoDestination: 2, NoRumour: 3, TooFar: 4, NoViable: 5, NoGain: 6,
			Bound: 7, OutCap: 8, Unprovisioned: 9, InCap: 10, Held: 11, Ended: 2, Started: 99, Pushed: 99})
	got := h.collect()
	check(t, got, "migration.started", 7, "region.from", "r0", "region.to", "r2")
	check(t, got, "migration.on_journey", 1, "")
	for i, l := range []string{"no_estimate", "no_destination", "no_rumour", "too_far", "no_viable", "no_gain",
		"bound", "out_cap", "unprovisioned", "in_cap", "held", "ended"} {
		want := int64(i + 1)
		if l == "ended" {
			want = 2
		}
		check(t, got, "migration.limited", want, "limit", l)
	}
}

// TestContracts_FormedValueEscrow verifies the contracts formed and their value
// by commodity, and the escrow level: money escrowed less money paid and
// refunded at the settlement.
//
// Requirement: FR-OBS-050
func TestContracts_FormedValueEscrow(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.ContractFormed{Commodity: 2, Escrowed: 100}, economysim.ContractFormed{Commodity: 2, Escrowed: 40})
	got := h.collect()
	check(t, got, "contract.formed", 2, "commodity", "c2")
	check(t, got, "contract.value", 140, "commodity", "c2")
	check(t, got, "contract.escrow", 140, "")
	tick(h, 1, economysim.ContractSettled{Paid: 70, Refunded: 30})
	check(t, h.collect(), "contract.escrow", 40, "")
}

// TestMerchants_Decisions verifies the eleven lane decisions of a merchant, each
// from its own count.
//
// Requirement: FR-OBS-050
func TestMerchants_Decisions(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.MerchantsPlanned{NoRumour: 1, Stale: 2, Locked: 3, Perishable: 4, NoMargin: 5, Unaffordable: 6,
		BelowShare: 7, Planned: 8, Rested: 9, Waiting: 10, LotCapped: 11, Considered: 99, Dispatched: 99})
	got := h.collect()
	for i, d := range []string{"no_rumour", "stale", "locked", "perishable", "no_margin", "unaffordable",
		"below_share", "planned", "rested", "waiting", "lot_capped"} {
		check(t, got, "merchant.decisions", int64(i+1), "decision", d)
	}
}

// TestExternal_UnitsValuePriceHeld verifies the units and money of an outside
// line by node, commodity and side, and the price by node, gate region,
// commodity and side, held across a silent tick.
//
// Requirement: FR-OBS-050
func TestExternal_UnitsValuePriceHeld(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.OutsideTraded{Node: 2, Region: 0, Commodity: 1, Side: economysim.Ask, Price: 30, Traded: 4, Value: 120},
		economysim.OutsideTraded{Node: 2, Region: 0, Commodity: 1, Side: economysim.Ask, Price: 32, Traded: 1, Value: 32},
		economysim.OutsideTraded{Node: 2, Region: 1, Commodity: 1, Side: economysim.Bid, Price: 20, Traded: 2, Value: 40})
	before := h.calls["economysim.external.price"]
	tick(h, 1)
	if n := h.calls["economysim.external.price"] - before; n != 2 {
		t.Errorf("external.price records in a silent tick = %d, want 2", n)
	}
	got := h.collect()
	check(t, got, "external.units", 5, "node", "r2", "commodity", "c1", "side", "ask")
	check(t, got, "external.value", 152, "node", "r2", "commodity", "c1", "side", "ask")
	check(t, got, "external.price", 32, "node", "r2", "region", "r0", "commodity", "c1", "side", "ask")
	check(t, got, "external.price", 20, "node", "r2", "region", "r1", "commodity", "c1", "side", "bid")
}

// TestRestore_Stage2Levels verifies that WorldRestored seeds the levels of the
// modules from its snapshot: the units in flight by commodity, the households on
// a journey, the escrow, the stock of the endowments, the households in the
// own-use mode and the quotes of the outside sides.
//
// Requirement: FR-OBS-054
func TestRestore_Stage2Levels(t *testing.T) {
	h := newHarness(t)
	snap := economysim.Snapshot{
		Shipments: []economysim.ShipmentRow{
			{Lines: []economysim.Amount{{Commodity: 0, Quantity: 4}, {Commodity: 1, Quantity: 2}}},
			{Lines: []economysim.Amount{{Commodity: 0, Quantity: 1}}}},
		Journeys:   []economysim.JourneyRow{{Weight: 3}, {Weight: 2}},
		Escrow:     250,
		Endowments: []economysim.EndowmentRow{{Region: 1, Resource: 0, Stock: 80}},
		Roles:      []economysim.RoleRow{{Region: 2, Role: 1, OnOwnUse: 9}},
		Outside: economysim.OutsideState{Sides: []economysim.OutsideSideRow{
			{Node: 2, Gate: 0, Commodity: 1, Side: economysim.Bid, Quote: 44}}},
	}
	h.send(economysim.WorldRestored{Names: miniNames(), Snapshot: snap})
	got := h.collect()
	check(t, got, "transport.in_transit", 5, "commodity", "c0")
	check(t, got, "transport.in_transit", 2, "commodity", "c1")
	check(t, got, "migration.on_journey", 5, "")
	check(t, got, "contract.escrow", 250, "")
	check(t, got, "endowment.stock", 80, "resource", "s0", "region", "r1")
	check(t, got, "household.own_use", 9, "role", "k1", "region", "r2")
	check(t, got, "external.price", 44, "node", "r2", "region", "r0", "commodity", "c1", "side", "bid")
}
