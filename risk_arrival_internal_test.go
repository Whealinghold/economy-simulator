package economysim

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file tests step 0 of an arrival and what it reaches (plan 3-rsk Phase 5): the
// loss of the Transit account and of the traveller's own account, the order guard,
// step 0, move, delivery, the per-line identity, the contract and the events
// (spec-risk.md 5, 7.3). The world is shipConfig: edge 40, foot and coach both take
// two ticks, so a shipment dispatched in tick 0 arrives in the first phase of tick 2.

// rheTest is a * b / 10 000 rounded half to even, written apart from the module so a
// test does not check the rule with itself.
func rheTest(a, b int64) int64 {
	q, r := a*b/10_000, a*b%10_000
	if 2*r > 10_000 || (2*r == 10_000 && q%2 == 1) {
		q++
	}
	return q
}

// lossBoundary returns the units of a commodity the boundary of the reason loss has
// received, by the snapshot.
func lossBoundary(t *testing.T, w *World, c CommodityID) int64 {
	t.Helper()
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	for _, b := range snap.Boundary {
		if b.Reason == ReasonLoss && b.Commodity == c {
			return b.Balance
		}
	}
	t.Fatalf("no boundary row of loss for commodity %d", c)
	return 0
}

// TestArrival_LossOfLinesAndIdentity verifies the worked example of spec-risk.md 18.1:
// a hired shipment of four lines struck at LossBP 2 500 loses rhe(held * 2 500, 10 000)
// of each, so 37 of one commodity lose 9, 5 lose 1, 2 lose 0 (a tie, to the even 0) and
// 6 lose 2 (a tie, to the even 2); the per-line identity shipped = delivered + diverted
// + spoiled + lost holds, the consignee receives the survivors, the boundary of loss
// holds the units and the BoundaryFlow of the tick names the origin region.
//
// Requirement: FR-RSK-012, FR-RSK-014, FR-RSK-015, FR-RSK-022
func TestArrival_LossOfLinesAndIdentity(t *testing.T) {
	t.Parallel()
	w, rec := newShipWorld(t, riskShip(riskCoach("a_home", "b_far", 10_000, 2_500, false)), nil)
	flour, grain, hay, perish := commodityOf(t, w, "flour"), commodityOf(t, w, "grain"), commodityOf(t, w, "hay"), commodityOf(t, w, "perish")
	lines := []Line{{flour, 5}, {grain, 37}, {hay, 2}, {perish, 6}}
	for _, ln := range lines {
		clearGoods(t, w, shipHome, ln.Commodity)
		give(t, w, shipHome, ln.Commodity, int64(ln.Quantity))
	}
	give(t, w, shipHome, commodityOf(t, w, "haul"), 4)
	if id, reason := ask(w, shipHome, 0, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: lines}); reason != "" || w.transport.ships[id].loss != 2_500 {
		t.Fatalf("dispatch = (%d, %q), loss %d, want a shipment struck at 2500", id, reason, w.transport.ships[id].loss)
	}
	step(t, w, 3)
	got := of[ShipmentArrived](rec)
	if len(got) != 1 || got[0].AccountLost != nil {
		t.Fatalf("ShipmentArrived = %+v, want one with no AccountLost (a hired shipment)", got)
	}
	want := map[CommodityID]Quantity{flour: 1, grain: 9, hay: 0, perish: 2}
	for _, l := range got[0].Lines {
		requireLostLine(t, w, l, want[l.Commodity])
	}
	requireLossFlows(t, rec, want)
}

// requireLostLine fails the test unless a line lost want units, kept the identity
// shipped = delivered + diverted + spoiled + lost, delivered the survivors and sent
// the units lost to the boundary.
func requireLostLine(t *testing.T, w *World, l ArrivedLine, want Quantity) {
	t.Helper()
	if l.Lost != want || l.Shipped != l.Delivered+l.Diverted+l.Spoiled+l.Lost || l.Diverted != 0 || l.Spoiled != 0 || l.Delivered != l.Shipped-l.Lost {
		t.Errorf("line %+v, want lost %d, the identity shipped = delivered + diverted + spoiled + lost and the survivors delivered", l, want)
	}
	if b := lossBoundary(t, w, l.Commodity); b != int64(l.Lost) {
		t.Errorf("boundary of loss for commodity %d = %d, want %d", l.Commodity, b, l.Lost)
	}
}

// requireLossFlows fails the test unless every BoundaryFlow of the reason loss names
// the origin region and the units of its commodity.
func requireLossFlows(t *testing.T, rec *evLog, want map[CommodityID]Quantity) {
	t.Helper()
	for _, f := range of[BoundaryFlow](rec) {
		if f.Reason == ReasonLoss && (f.Region != regHome || f.Total != int64(want[f.Commodity])) {
			t.Errorf("BoundaryFlow %+v, want loss attributed to the origin region with the units of the line", f)
		}
	}
}

// TestArrival_TripAccountExceptVehicle verifies spec-risk.md 5.4.1 and example C: a
// self-carried trip by a mode with a vehicle loses rhe(held * LossBP, 10 000) of every
// commodity its traveller's account holds but the vehicle: 12 of one lose 7 at 6 000
// and 3 of another lose 2, the 2 vehicles lose none (a rule over every commodity
// would take rhe(2 * 6 000, 10 000) = 1 of them), and the vehicles then move with the
// traveller. The household eats on the road, so the grain left is read.
//
// Requirement: FR-RSK-013
func TestArrival_TripAccountExceptVehicle(t *testing.T) {
	t.Parallel()
	row := RouteRisk{From: "a_home", To: "b_far", Mode: "wagon", StrikeBP: 10_000, SeverityBP: 6_000}
	w, rec := newShipWorld(t, func(c *Config) { riskShip(row)(c); c.World.MinAgents = 1; c.Groups[0].Scale = 2 }, nil)
	for c := range w.cfg.Commodities {
		clearGoods(t, w, shipHome, CommodityID(c))
	}
	flour, grain, hay, wagon := commodityOf(t, w, "flour"), commodityOf(t, w, "grain"), commodityOf(t, w, "hay"), commodityOf(t, w, "wagon")
	give(t, w, shipHome, flour, 3)
	give(t, w, shipHome, grain, 12)
	give(t, w, shipHome, hay, 6)
	give(t, w, shipHome, wagon, 2)
	id, reason := ask(w, shipHome, 0, ShipmentRequest{Mode: modeWagon, To: regFar, ToSelf: true})
	if reason != "" || w.transport.ships[id].loss != 6_000 {
		t.Fatalf("dispatch = (%d, %q), want a wagon trip struck at 6000", id, reason)
	}
	// The household eats on the road, so the account holds less grain at the arrival.
	step(t, w, 2)
	inGrain, inFlour := held(t, w, shipHome, int(grain)), held(t, w, shipHome, int(flour))
	step(t, w, 1)
	got := of[ShipmentArrived](rec)
	want := []Amount{{flour, Quantity(rheTest(inFlour, 6_000))}, {grain, Quantity(rheTest(inGrain, 6_000))}}
	if inFlour != 3 || want[0].Quantity != 2 {
		t.Fatalf("flour held %d, lost %d, want 3 and 2: the example of the spec", inFlour, want[0].Quantity)
	}
	if len(got) != 1 || !equalAmounts(got[0].AccountLost, want) {
		t.Fatalf("ShipmentArrived = %+v, want AccountLost %+v", got, want)
	}
	if n := held(t, w, shipHome, int(wagon)); n != 2 {
		t.Errorf("the traveller holds %d wagons after the arrival, want the 2 it started with", n)
	}
	if g := w.regionOf(&w.agents[shipHome]); g != regFar {
		t.Errorf("the traveller is in region %d, want far", g)
	}
	requireLossFlows(t, rec, map[CommodityID]Quantity{flour: want[0].Quantity, grain: want[1].Quantity})
}

// equalAmounts reports whether two lists of amounts are equal.
func equalAmounts(a, b []Amount) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestArrival_HeldMoveTakesNoLossTwice verifies the order of spec-risk.md 5.1.3: an
// arrival that the guard refuses (another obligation of the traveller) is held with
// nothing applied, so the loss is not taken at the first attempt and is taken once at
// the retry. The trip carries 2 grain to a consignee and the traveller's account holds
// the rest; at 5 000 the line loses 1 and the account half of what it holds.
//
// Fix: no bug existed yet. A loss applied before a refused move would be applied again
// at the retry, and the boundary would hold the loss twice.
//
// Requirement: FR-RSK-012
func TestArrival_HeldMoveTakesNoLossTwice(t *testing.T) {
	t.Parallel()
	row := RouteRisk{From: "a_home", To: "b_far", Mode: "foot", StrikeBP: 10_000, SeverityBP: 5_000}
	w, rec := newShipWorld(t, riskShip(row), onTick(0, toFar(Line{Commodity: grain, Quantity: 2})))
	grainID := commodityOf(t, w, "grain")
	holding := true
	w.bounds.add("test_hold", func(a *agent, _ []string) bool { return holding && a.id == shipHome })
	step(t, w, 3) // the arrival of tick 2 is held
	if len(of[ShipmentArrived](rec)) != 0 {
		t.Fatal("the held shipment arrived")
	}
	if b := lossBoundary(t, w, grainID); b != 0 {
		t.Errorf("a held arrival applied a loss: boundary of loss for grain = %d, want 0", b)
	}
	account := held(t, w, shipHome, int(grainID))
	if account < 4 || w.transport.ships[0].lines[0].Quantity != 2 {
		t.Fatalf("the traveller holds %d grain while held, want a few to lose", account)
	}
	holding = false
	step(t, w, 1) // the retry
	got := of[ShipmentArrived](rec)
	if len(got) != 1 || got[0].Lines[0].Lost != 1 || !equalAmounts(got[0].AccountLost, []Amount{{grainID, Quantity(rheTest(account, 5_000))}}) {
		t.Fatalf("ShipmentArrived = %+v, want the line to lose 1 and the account %d of %d grain", got, rheTest(account, 5_000), account)
	}
	if b := lossBoundary(t, w, grainID); b != 1+rheTest(account, 5_000) {
		t.Errorf("boundary of loss for grain = %d, want %d (the line and the account, once)", b, 1+rheTest(account, 5_000))
	}
}

// TestArrival_RemovedTravellerStrikesTransitOnly verifies spec-risk.md 5.4.3: a trip
// with lines whose traveller was removed continues, step 0 strikes its Transit account
// as for any shipment and the event has an empty AccountLost.
//
// Requirement: FR-RSK-013
func TestArrival_RemovedTravellerStrikesTransitOnly(t *testing.T) {
	t.Parallel()
	row := RouteRisk{From: "a_home", To: "b_far", Mode: "foot", StrikeBP: 10_000, SeverityBP: 5_000}
	w, rec := newShipWorld(t, riskShip(row), onTick(0, toFar(Line{Commodity: grain, Quantity: 4})))
	step(t, w, 1)
	if err := w.removeAgent(&w.agents[shipHome], CauseConsumer); err != nil {
		t.Fatalf("removeAgent() = %v, want nil", err)
	}
	step(t, w, 2)
	got := of[ShipmentArrived](rec)
	if len(got) != 1 || got[0].Lines[0].Lost != 2 || got[0].Lines[0].Delivered != 2 || got[0].AccountLost != nil {
		t.Errorf("ShipmentArrived = %+v, want 2 of 4 lost, 2 delivered and no AccountLost", got)
	}
}

// TestArrival_TotalLossNothingToDivert verifies spec-risk.md 5.5.1: a shipment with a
// LossBP of 10 000 whose consignee was removed has nothing to deliver and nothing to
// divert: Diverted is false, the receiver is the consignee and every line's Lost is
// what shipped.
//
// Requirement: FR-RSK-015
func TestArrival_TotalLossNothingToDivert(t *testing.T) {
	t.Parallel()
	row := RouteRisk{From: "a_home", To: "b_far", Mode: "foot", StrikeBP: 10_000, SeverityBP: 10_000}
	w, rec := newShipWorld(t, riskShip(row), onTick(0, toFar(Line{Commodity: grain, Quantity: 2})))
	step(t, w, 1)
	if err := w.removeAgent(&w.agents[shipFar], CauseConsumer); err != nil {
		t.Fatalf("removeAgent() = %v, want nil", err)
	}
	step(t, w, 2)
	got := of[ShipmentArrived](rec)
	if len(got) != 1 || got[0].Diverted || got[0].Lines[0].Lost != 2 || got[0].Lines[0].Delivered != 0 || got[0].Lines[0].Diverted != 0 {
		t.Errorf("ShipmentArrived = %+v, want a total loss with nothing delivered or diverted", got)
	}
}

// TestContract_LostBorneByDeliverer verifies spec-risk.md 5.6.1: a contract of 10 units
// at 100 whose shipment is struck at 2 500 loses rhe(2.5) = 2 units (a tie, to the even
// 2); the 8 that reached are paid 800, 200 are refunded to the payer, the settlement
// says Lost 2 and the cause is short although nothing spoiled.
//
// Requirement: FR-RSK-016
func TestContract_LostBorneByDeliverer(t *testing.T) {
	t.Parallel()
	row := RouteRisk{From: "a_home", To: "b_far", Mode: "coach", StrikeBP: 10_000, SeverityBP: 2_500}
	w, rec, _ := roundWorld(t, func(c *Config) { trio(c); c.Modules.Risk = &Risk{Routes: []RouteRisk{row}} }, oneGrain(), d32)
	step(t, w, 3)
	got := settled(rec)
	want := ContractSettled{Tick: 2, Contract: 0, Payer: p61, Deliverer: d32, Region: regHome, To: regFar, Commodity: cGrain,
		Quantity: 10, Reached: 8, Lost: 2, Paid: 800, Refunded: 200, Cause: CauseShort}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("ContractSettled = %+v, want %+v", got, want)
	}
	if b := lossBoundary(t, w, cGrain); b != 2 {
		t.Errorf("boundary of loss for grain = %d, want 2", b)
	}
}

// TestArrival_NoEarmarkAtStepZero verifies that no account holds an earmark at a tick
// boundary of a self-carried trip, so the loss of step 0 (phase 1, which takes the
// held units of the traveller's account) never meets an earmarked unit that Move
// would refuse: a reservation is made at the admission of an ask in phase 3 or 4 and
// released with its book at clearing, inside the same tick.
//
// Guard: tripLoss counts earmarked units in held and a refusal would panic through
// invariant. If a reservation outlived its tick, step 0 would crash at the next
// arrival; this test fails first, at the boundary, and names the account.
func TestArrival_NoEarmarkAtStepZero(t *testing.T) {
	t.Parallel()
	row := RouteRisk{From: "a_home", To: "b_far", Mode: "foot", StrikeBP: 10_000, SeverityBP: 5_000}
	w, rec := newShipWorld(t, riskShip(row), onTick(0, toFar(Line{Commodity: grain, Quantity: 2})))
	for tick := 0; tick < 3; tick++ {
		step(t, w, 1)
		for id := 0; id < w.ledger.Len(); id++ {
			v, err := w.ledger.Account(id)
			if err != nil || v.Closed || v.Kind == ledger.KindBoundary {
				continue
			}
			for c := range w.cfg.Commodities {
				if _, reserved, err := w.ledger.Holding(id, c); err != nil || reserved != 0 {
					t.Errorf("after tick %d account %d commodity %d: earmarked %d, error %v, want 0 and nil", tick, id, c, reserved, err)
				}
			}
		}
	}
	if len(of[ShipmentArrived](rec)) != 1 {
		t.Fatalf("ShipmentArrived events = %d, want the trip to have arrived and struck step 0", len(of[ShipmentArrived](rec)))
	}
}
