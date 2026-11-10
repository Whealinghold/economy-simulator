package otelobserver_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// wantSeries is one series a scripted tick must leave: the metric, its value and
// the attributes, in the pairs of check.
type wantSeries struct {
	name string
	v    int64
	kv   []string
}

// TestDeferred_EventFedCounters verifies the counters added from the deferred
// candidates of the Stage 2 reviews: each is fed by the fields the rules name, by
// the attributes the rules name, and a zero count or a value outside the closed
// set adds no series.
//
// Requirement: FR-OBS-050
func TestDeferred_EventFedCounters(t *testing.T) {
	tests := []struct {
		name   string
		events []economysim.Event
		want   []wantSeries
		none   []wantSeries
	}{
		{"want choked", []economysim.Event{economysim.WantShortfall{Want: 1, Role: 1, Choked: 4, Planned: 9, Funded: 9},
			economysim.WantShortfall{Want: 1, Role: 1, Choked: 2}},
			[]wantSeries{{"want.choked", 6, []string{"want", "w1", "role", "k1"}}},
			[]wantSeries{{"want.choked", 0, []string{"want", "w0", "role", "k0"}}}},
		{"money flow limited by flow and limit", []economysim.Event{
			economysim.MoneyFlowLimited{Flow: "levy", Due: 10, Applied: 4, Short: 3, Headroom: 0, Limit: 2},
			economysim.MoneyFlowLimited{Flow: "levy", Short: 1}},
			[]wantSeries{{"money.flow.limited", 4, []string{"reason", "levy", "flow_limit", "short"}},
				{"money.flow.limited", 2, []string{"reason", "levy", "flow_limit", "limit"}}},
			[]wantSeries{{"money.flow.limited", 0, []string{"reason", "levy", "flow_limit", "headroom"}}}},
		{"population limited", []economysim.Event{economysim.PopulationChanged{Role: 1, DeathsCut: 1, DeathsHeld: 2,
			BirthsCapped: 3, BirthsUnfunded: 4, RateClamped: 5, BirthsDone: 9}},
			[]wantSeries{{"population.limited", 1, []string{"population_limit", "deaths_cut"}},
				{"population.limited", 2, []string{"population_limit", "deaths_held"}},
				{"population.limited", 3, []string{"population_limit", "births_capped"}},
				{"population.limited", 4, []string{"population_limit", "births_unfunded"}},
				{"population.limited", 5, []string{"population_limit", "rate_clamped"}}}, nil},
		{"cargo diverted and shipments rejected", []economysim.Event{
			economysim.ShipmentArrived{Lines: []economysim.ArrivedLine{{Commodity: 2, Shipped: 5, Delivered: 2, Diverted: 3}}},
			economysim.ShipmentRejected{Mode: 1, Reason: economysim.ReasonNoVehicle},
			economysim.ShipmentRejected{Mode: 1, Reason: economysim.ReasonNoVehicle},
			economysim.ShipmentRejected{Mode: 0, Reason: economysim.ReasonBlocked},
			economysim.ShipmentRejected{Mode: 9, Reason: economysim.ReasonBlocked}},
			[]wantSeries{{"transport.cargo.diverted", 3, []string{"commodity", "c2"}},
				{"transport.rejected", 2, []string{"mode", "m1", "reason", "no_vehicle"}},
				{"transport.rejected", 1, []string{"mode", "m0", "reason", "blocked"}}},
			[]wantSeries{{"transport.rejected", 0, []string{"mode", "m9", "reason", "blocked"}}}},
		{"readers by region", []economysim.Event{economysim.RumoursServed{Region: 2, Readers: 5, Offered: 1},
			economysim.RumoursServed{Region: 2, Readers: 1}},
			[]wantSeries{{"information.readers", 6, []string{"region", "r2"}},
				{"information.rumours", 1, []string{"outcome", "offered", "region", "r2"}}}, nil},
		{"outside cut unfilled clamped", []economysim.Event{
			economysim.OutsideTraded{Node: 2, Region: 0, Commodity: 1, Side: economysim.Ask, Price: 30, Offered: 10, Traded: 4,
				CutClosed: 1, CutCap: 2, CutPrice: 3, PriceClamped: true},
			economysim.OutsideTraded{Node: 2, Region: 0, Commodity: 1, Side: economysim.Ask, Price: 30, Offered: 5, Traded: 5}},
			[]wantSeries{{"external.cut", 1, []string{"node", "r2", "commodity", "c1", "side", "ask", "external_cut", "closed"}},
				{"external.cut", 2, []string{"node", "r2", "commodity", "c1", "side", "ask", "external_cut", "cap"}},
				{"external.cut", 3, []string{"node", "r2", "commodity", "c1", "side", "ask", "external_cut", "price"}},
				{"external.unfilled", 6, []string{"node", "r2", "commodity", "c1", "side", "ask"}},
				{"external.clamped", 1, []string{"node", "r2", "commodity", "c1"}}}, nil},
		{"own use blocked and changes", []economysim.Event{economysim.OwnUseChanged{Role: 1, Region: 2, Blocked: 3, Entered: 4, Left: 5}},
			[]wantSeries{{"own_use.blocked", 3, []string{"role", "k1"}},
				{"household.own_use.changes", 4, []string{"role", "k1", "own_use_change", "entered"}},
				{"household.own_use.changes", 5, []string{"role", "k1", "own_use_change", "left"}}}, nil},
		{"migration unsettled", []economysim.Event{economysim.MigrationReviewed{Region: 1, Unsettled: 7, OutCap: 2}},
			[]wantSeries{{"migration.unsettled", 7, []string{""}},
				{"migration.limited", 2, []string{"limit", "out_cap", "region", "r1"}}}, nil},
		{"contracts settled and unformed", []economysim.Event{
			economysim.ContractSettled{Cause: economysim.CauseShort}, economysim.ContractSettled{Cause: economysim.CauseShort},
			economysim.ContractSettled{Cause: economysim.CauseDelivererRemoved}, economysim.ContractSettled{Cause: "other"},
			economysim.ContractRound{NoTaker: 1, PayerFunds: 2, TooMany: 3, DispatchRefused: 4, OffersRefused: 5, TakesRefused: 6}},
			[]wantSeries{{"contract.settled", 2, []string{"cause", "short"}},
				{"contract.settled", 1, []string{"cause", "deliverer_removed"}},
				{"contract.unformed", 1, []string{"contract_failure", "no_taker"}},
				{"contract.unformed", 4, []string{"contract_failure", "dispatch_refused"}},
				{"contract.refused", 5, []string{"contract_request", "offer"}},
				{"contract.refused", 6, []string{"contract_request", "take"}}},
			[]wantSeries{{"contract.settled", 0, []string{"cause", "delivered"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			h.send(created())
			tick(h, 0, tt.events...)
			got := h.collect()
			for _, w := range tt.want {
				check(t, got, w.name, w.v, w.kv...)
			}
			for _, w := range tt.none {
				absent(t, got, w.name, w.kv...)
			}
		})
	}
}

// TestDeferred_ContractSettledOtherCauseAddsNoSeries verifies that a cause outside
// the closed set of the contract events adds no series and does not touch the
// escrow level.
//
// Guard: the cause table is shared with the removal and relocation causes; a
// lookup that fell back to place 0 would count the contract as an insolvency.
func TestDeferred_ContractSettledOtherCauseAddsNoSeries(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.ContractSettled{Cause: "other", Paid: 3})
	if n := len(h.collect()["economysim.contract.settled"]); n != 0 {
		t.Errorf("contract.settled series = %d, want 0 for a cause outside the set", n)
	}
}

// TestDeferred_ExternalPriceSkipsNoQuote verifies that an OutsideTraded with a
// Price of 0 (no quote was made: a closed edge) does not set external.price, so
// the gauge keeps its last quote instead of reading free, and that the cut
// counts the fact.
//
// Guard: a 0 recorded for a closed gate would read as a price of nothing.
func TestDeferred_ExternalPriceSkipsNoQuote(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	key := []string{"node", "r2", "region", "r0", "commodity", "c1", "side", "ask"}
	tick(h, 0, economysim.OutsideTraded{Node: 2, Region: 0, Commodity: 1, Side: economysim.Ask, Price: 30})
	tick(h, 1, economysim.OutsideTraded{Node: 2, Region: 0, Commodity: 1, Side: economysim.Ask, Wanted: 4, CutClosed: 4})
	got := h.collect()
	check(t, got, "external.price", 30, key...)
	check(t, got, "external.cut", 4, "node", "r2", "commodity", "c1", "side", "ask", "external_cut", "closed")
}

// TestDeferred_RelocationCauseOfAContractIsNotAJourney verifies that a relocation
// whose cause is in the shared cause table but is not migration leaves the
// journey level alone.
//
// Guard: the contract causes were appended to the table of the removal and
// relocation causes; a test of "at or after migration" would take them for it.
func TestDeferred_RelocationCauseOfAContractIsNotAJourney(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.AgentRelocated{From: 0, To: 1, Weight: 3, Cause: economysim.CauseDelivered})
	absent(t, h.collect(), "migration.on_journey", "")
}

// TestDeferred_ExternalPriceOfOneIsAQuote verifies the edge of the no-quote rule:
// a price of 1 is a quote and a price of 0 or below is not.
//
// Guard: the rule is "above 0"; a comparison moved by one would drop the lowest
// legal quote or record a negative one.
func TestDeferred_ExternalPriceOfOneIsAQuote(t *testing.T) {
	h := newHarness(t)
	h.send(created())
	tick(h, 0, economysim.OutsideTraded{Node: 2, Region: 0, Commodity: 1, Side: economysim.Ask, Price: 1},
		economysim.OutsideTraded{Node: 2, Region: 1, Commodity: 1, Side: economysim.Ask, Price: -4})
	got := h.collect()
	check(t, got, "external.price", 1, "node", "r2", "region", "r0", "commodity", "c1", "side", "ask")
	absent(t, got, "external.price", "node", "r2", "region", "r1", "commodity", "c1", "side", "ask")
}

// deferredSnap is a snapshot over miniNames with the three modules on.
func deferredSnap() economysim.Snapshot {
	return economysim.Snapshot{
		Names:      miniNames(),
		Modules:    []string{"population", "transport", "migration"},
		Population: economysim.PopulationState{HasReplacement: true, ReplacementRatioBP: 9_500},
		Regions: []economysim.RegionRow{{Region: 1, HouseholdCap: 40, OnJourneyFrom: 3, OnJourneyTo: 5},
			{Region: 2, HouseholdCap: 0}, {Region: 0, HouseholdCap: 1}},
		Roles: []economysim.RoleRow{{Region: 1, Role: 0, NaturalIncreaseBP: -120}},
		Shipments: []economysim.ShipmentRow{{Mode: 1, TicksLeft: 0}, {Mode: 1, TicksLeft: -2}, {Mode: 1, TicksLeft: 3}, {Mode: 1, TicksLeft: 5}, {Mode: 1, TicksLeft: 7},
			{Mode: 0, TicksLeft: 1}, {Mode: 0, TicksLeft: 0}, {Mode: -1, TicksLeft: 0}, {Mode: 2, TicksLeft: 0}},
		Away: []economysim.AgentID{4, 9},
	}
}

// TestDeferred_SnapshotGauges verifies the snapshot-fed gauges of the deferred
// candidates: the natural increase of a role, the household cap of a region
// (absent at 0), the replacement ratio where there is one, the shipments overdue
// by mode (0 for a mode with none, a mode outside the names dropped), the agents
// away and the migration pipeline of each region, from and to.
//
// Requirement: FR-OBS-050
func TestDeferred_SnapshotGauges(t *testing.T) {
	h := newHarness(t)
	h.obs.RecordSnapshot(deferredSnap())
	got := h.collect()
	check(t, got, "population.natural_increase", -120, "role", "k0", "region", "r1")
	check(t, got, "population.cap", 40, "region", "r1")
	absent(t, got, "population.cap", "region", "r2")
	check(t, got, "population.replacement_ratio", 9_500)
	check(t, got, "transport.overdue", 2, "mode", "m1")
	check(t, got, "transport.overdue", 1, "mode", "m0")
	check(t, got, "population.cap", 1, "region", "r0")
	check(t, got, "transport.away", 2)
	check(t, got, "migration.pipeline", 3, "region", "r1", "direction", "from")
	check(t, got, "migration.pipeline", 5, "region", "r1", "direction", "to")
	if n := len(got["economysim.transport.overdue"]); n != 2 {
		t.Errorf("transport.overdue series = %d, want 2: one per mode of the names", n)
	}
}

// TestDeferred_SnapshotCapBelowOneIsAbsent verifies that a household cap that is
// not above 0 records no series.
//
// Guard: 0 means no cap in the snapshot, so a series of 0 would draw a cap of
// nothing; a negative cap is a library defect the gauge must not show.
func TestDeferred_SnapshotCapBelowOneIsAbsent(t *testing.T) {
	s := deferredSnap()
	s.Regions = []economysim.RegionRow{{Region: 1, HouseholdCap: -3}, {Region: 2}}
	h := newHarness(t)
	h.obs.RecordSnapshot(s)
	if n := len(h.collect()["economysim.population.cap"]); n != 0 {
		t.Errorf("population.cap series = %d for a cap of -3 and 0, want 0", n)
	}
}

// TestDeferred_SnapshotGaugesAbsentWithModuleOff verifies that the gauges of a
// module that is not in Modules are absent, not zero, and that a replacement ratio
// is absent where the schedules have none.
//
// Guard: the rows exist in every snapshot with the module off at their zero
// value; recording them would put a series of zeros on every region and role.
func TestDeferred_SnapshotGaugesAbsentWithModuleOff(t *testing.T) {
	s := deferredSnap()
	s.Modules = nil
	h := newHarness(t)
	h.obs.RecordSnapshot(s)
	got := h.collect()
	for _, name := range []string{"population.natural_increase", "population.cap", "population.replacement_ratio",
		"transport.overdue", "transport.away", "migration.pipeline"} {
		if n := len(got["economysim."+name]); n != 0 {
			t.Errorf("%s series = %d with the module off, want 0", name, n)
		}
	}
	s.Modules = []string{"population"}
	s.Population.HasReplacement = false
	h = newHarness(t)
	h.obs.RecordSnapshot(s)
	if n := len(h.collect()["economysim.population.replacement_ratio"]); n != 0 {
		t.Errorf("population.replacement_ratio series = %d without HasReplacement, want 0", n)
	}
}
