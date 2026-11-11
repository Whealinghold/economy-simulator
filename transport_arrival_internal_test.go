package economysim

import (
	"errors"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file tests the arrival side of the transport module (transport_arrival.go,
// plan 2-12 Phase 5) over the world of transport_request_internal_test.go. Edge 40,
// foot speed 20 and coach speed 30 both give k = 2: a shipment dispatched in tick t
// arrives in the first phase of tick t + 2.

// onTick returns a ship function that asks for reqs from the household home at the
// tick at, once.
func onTick(at int64, reqs ...ShipmentRequest) func(v *AgentView) []ShipmentRequest {
	return func(v *AgentView) []ShipmentRequest {
		if v.Agent == shipHome && v.Tick.Tick == at {
			return reqs
		}
		return nil
	}
}

// toFar is a foot request of lines to the household far.
func toFar(lines ...Line) ShipmentRequest {
	return ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: shipFar, Lines: lines}
}

// TestArrival_ShipmentOrderAndEvents verifies P1.3 (spec-transport.md 5.7, 13.1): a
// self-carried trip with a line arrives in the first phase of tick 2, the traveller
// moves with AgentRelocated (cause transport) before ShipmentArrived, the cargo goes
// to the consignee under transit_in, the Transit account joins the free list, and
// both events precede the relief and the trade of the tick.
//
// Arithmetic: 3 grain shipped, k = ceil(40 / 20) = 2, dispatched in tick 0, so the
// arrival is in tick 2; the consignee held 10 grain and holds 13.
//
// Requirement: FR-TRN-024, FR-TRN-035
func TestArrival_ShipmentOrderAndEvents(t *testing.T) {
	t.Parallel()
	w, rec := newShipWorld(t, nil, onTick(0, toFar(Line{Commodity: grain, Quantity: 3})))
	step(t, w, 2)
	if len(w.transport.ships) != 1 || !w.transport.isAway(shipHome) {
		t.Fatalf("after tick 1: %d shipments in flight, away = %v, want 1 and true", len(w.transport.ships), w.transport.isAway(shipHome))
	}
	acct := w.transport.ships[0].account
	rec.events = nil
	step(t, w, 1)
	var seq []string
	for _, e := range rec.events {
		switch e := e.(type) {
		case AgentRelocated:
			seq = append(seq, "relocated")
			if e.Agent != shipHome || e.From != regHome || e.To != regFar || e.Cause != CauseTransport {
				t.Errorf("AgentRelocated = %+v, want agent %d home to far by transport", e, shipHome)
			}
		case ShipmentArrived:
			seq = append(seq, "arrived")
			want := ShipmentArrived{Tick: 2, Shipment: 0, Mode: modeFoot, Region: regHome, To: regFar, Receiver: shipFar,
				Lines: []ArrivedLine{{Commodity: grain, Shipped: 3, Delivered: 3}}}
			if !reflect.DeepEqual(e, want) {
				t.Errorf("ShipmentArrived = %+v, want %+v", e, want)
			}
		case OrderPlaced, TickStarted:
			seq = append(seq, "-")
		}
	}
	if got := slices.DeleteFunc(seq, func(s string) bool { return s == "-" }); !slices.Equal(got, []string{"relocated", "arrived"}) {
		t.Errorf("events = %v, want relocated then arrived", got)
	}
	if got := held(t, w, shipFar, grain); got != 10 { // 10 - 2 ticks of upkeep + 3 delivered - 1 in tick 2
		t.Errorf("consignee grain = %d, want 10 - 2 + 3 - 1", got)
	}
	if held(t, w, shipHome, grain) == 0 || w.regionOf(&w.agents[shipHome]) != regFar || w.transport.isAway(shipHome) {
		t.Error("the traveller did not arrive: region, goods or away flag")
	}
	if w.transport.free[regHome].Len() != 1 {
		t.Errorf("free list = %d, want the Transit account %d back", w.transport.free[regHome].Len(), acct)
	}
	if v, _ := w.ledger.Account(acct); v.Closed {
		t.Error("the Transit account was closed, want it kept open")
	}
	if got := flowOf(w, ledger.TransitIn, grain); got != 0 {
		t.Errorf("transit_in grain in tick 3 totals = %d, want 0 (the delivery was in tick 2)", got)
	}
}

// TestArrival_SpoilageInTransit verifies spec-transport.md 5.6.2: a lot born in
// tick 0 with shelf life 3 spoils at the end of tick 2, so it is delivered by a trip
// that arrives in tick 2 (dispatched in tick 0) and spoiled in transit by one that
// arrives in tick 3 (dispatched in tick 1); the spoilage is attributed to the origin
// region (FR-TRN-040).
//
// Arithmetic: delivered iff t + k <= b + S - 1: 0 + 2 <= 0 + 3 - 1 yes; 1 + 2 <= 2 no.
//
// Requirement: FR-TRN-025, FR-TRN-026, FR-TRN-040
func TestArrival_SpoilageInTransit(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name              string
		at                int64
		delivered, spoilt Quantity
	}{{"dispatched at birth", 0, 2, 0}, {"dispatched a tick late", 1, 0, 2}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var perish CommodityID
			w, rec := newShipWorld(t, nil, func(v *AgentView) []ShipmentRequest {
				return onTick(tt.at, toFar(Line{Commodity: perish, Quantity: 2}))(v)
			})
			perish = commodityOf(t, w, "perish")
			give(t, w, shipHome, perish, 2) // born in tick 0
			step(t, w, int(tt.at)+3)
			got := of[ShipmentArrived](rec)
			if len(got) != 1 {
				t.Fatalf("ShipmentArrived events = %d, want 1", len(got))
			}
			l := got[0].Lines[0]
			if l.Delivered != tt.delivered || l.Spoiled != tt.spoilt || l.Shipped != 2 || l.Diverted != 0 {
				t.Errorf("arrived line = %+v, want delivered %d spoiled %d of 2", l, tt.delivered, tt.spoilt)
			}
			for _, b := range of[BoundaryFlow](rec) {
				if tt.spoilt > 0 && b.Reason == ReasonSpoilage && b.Commodity == perish && b.Region != regHome {
					t.Errorf("spoilage attributed to region %d, want the origin %d", b.Region, regHome)
				}
			}
		})
	}
}

// TestArrival_Diverted verifies spec-transport.md 5.8: a cargo whose consignee was
// removed goes to the estate of the destination, flagged, with no wait, and the
// receiver of the event is the estate pseudo-agent of the destination.
//
// Requirement: FR-TRN-027, FR-EST-003
func TestArrival_Diverted(t *testing.T) {
	t.Parallel()
	w, rec := newShipWorld(t, nil, onTick(0, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 3}}}))
	give(t, w, shipHome, commodityOf(t, w, "haul"), 4)
	step(t, w, 1)
	acct := w.transport.ships[0].account
	if err := w.removeAgent(&w.agents[shipFar], CauseConsumer); err != nil {
		t.Fatalf("removeAgent() = %v, want nil", err)
	}
	step(t, w, 2)
	got := of[ShipmentArrived](rec)
	if len(got) != 1 || !got[0].Diverted || got[0].Receiver != AgentID(regFar) {
		t.Fatalf("ShipmentArrived = %+v, want one diverted to the estate %d", got, regFar)
	}
	if l := got[0].Lines[0]; l.Diverted != 3 || l.Delivered != 0 {
		t.Errorf("line = %+v, want 3 diverted", l)
	}
	if n, _, _ := w.ledger.Holding(acct, grain); n != 0 {
		t.Errorf("the Transit account still holds %d grain", n)
	}
}

// TestShipment_Conservation verifies spec-transport.md 7.3.2 over seeds: on every
// ShipmentArrived, for every line, shipped = delivered + diverted + spoiled, and the
// audit (on) holds every tick.
//
// Requirement: FR-TRN-028
func TestShipment_Conservation(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 5; seed++ {
		var perish CommodityID
		w, rec := newShipWorld(t, nil, func(v *AgentView) []ShipmentRequest {
			if v.Agent != shipHome || v.Tick.Tick%3 != 0 {
				return nil
			}
			return []ShipmentRequest{{Mode: modeCoach, To: regFar, Consignee: shipFar,
				Lines: []Line{{Commodity: grain, Quantity: 1}, {Commodity: perish, Quantity: 2}}}}
		}, WithSeed(seed))
		perish = commodityOf(t, w, "perish")
		haul := commodityOf(t, w, "haul")
		give(t, w, shipHome, haul, 40)
		give(t, w, shipHome, perish, 20)
		step(t, w, 12)
		arrived := of[ShipmentArrived](rec)
		if len(arrived) == 0 {
			t.Fatalf("seed %d: no shipment arrived", seed)
		}
		for _, a := range arrived {
			for _, l := range a.Lines {
				if l.Shipped != l.Delivered+l.Diverted+l.Spoiled {
					t.Errorf("seed %d: shipment %d line %+v: shipped != delivered + diverted + spoiled", seed, a.Shipment, l)
				}
			}
		}
	}
}

// TestShipment_HaulUnitsAttributedToOrigin verifies spec-transport.md 7.3.3: the
// haul units a hired shipment redeems are in the flow of the origin region.
//
// Requirement: FR-MER-048
func TestShipment_HaulUnitsAttributedToOrigin(t *testing.T) {
	t.Parallel()
	w, rec := newShipWorld(t, nil, onTick(0, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 1}}}))
	haul := commodityOf(t, w, "haul")
	give(t, w, shipHome, haul, 4)
	step(t, w, 1)
	var got int64
	for _, b := range of[BoundaryFlow](rec) {
		if b.Reason == ReasonHaulUse && b.Commodity == haul && b.Region == regHome {
			got += b.Total
		}
	}
	if got != 4 {
		t.Errorf("haul_use of the origin region = %d, want 4", got)
	}
}

// TestArrival_ReseedsBeliefsOfSelfCarried verifies spec-transport.md 5.7.1: the
// traveller's beliefs are those of a fresh seeding at the destination, with no draw.
//
// Requirement: FR-MER-047
func TestArrival_ReseedsBeliefsOfSelfCarried(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, nil, onTick(0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}))
	step(t, w, 2)
	a := &w.agents[shipHome]
	row := w.roles.row(a.role)
	w.fillSeeds(regFar, row)
	want := make([]beliefSeed, len(w.ex.seeds))
	copy(want, w.ex.seeds)
	probe := agent{}
	reseedBeliefs(&probe, row, nil, want, w.bp)
	step(t, w, 0)
	if len(a.beliefs) != len(probe.beliefs) {
		t.Fatalf("beliefs = %d, want %d", len(a.beliefs), len(probe.beliefs))
	}
	for i := range a.beliefs {
		if a.beliefs[i].Width != probe.beliefs[i].Width {
			t.Errorf("belief %d width = %d, want %d (BeliefWidthInit)", i, a.beliefs[i].Width, probe.beliefs[i].Width)
		}
	}
}

// TestRemoval_ShipperTravellerConsignee verifies spec-transport.md 5.9: a removed
// traveller's trip to itself is dropped with the away flag cleared and its money
// goes to the pool of the departure region; a trip with lines continues to its
// consignee; the removal of a shipper changes nothing in a hired shipment.
//
// Requirement: FR-TRN-029, FR-EST-003
func TestRemoval_ShipperTravellerConsignee(t *testing.T) {
	t.Parallel()
	t.Run("traveller to itself", func(t *testing.T) {
		t.Parallel()
		w, _ := newShipWorld(t, nil, onTick(0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}))
		step(t, w, 1)
		pool := w.estates.pool(w.ledger, regHome)
		money, _ := balance(t, w, shipHome)
		if err := w.removeAgent(&w.agents[shipHome], CauseConsumer); err != nil {
			t.Fatalf("removeAgent() = %v, want nil", err)
		}
		if len(w.transport.ships) != 0 || w.transport.isAway(shipHome) {
			t.Errorf("after the removal: %d shipments, away %v, want 0 and false", len(w.transport.ships), w.transport.isAway(shipHome))
		}
		if got := w.estates.pool(w.ledger, regHome); int64(got-pool) != money {
			t.Errorf("pool of the departure region rose by %d, want the traveller's %d", got-pool, money)
		}
		step(t, w, 2)
	})
	t.Run("traveller with lines", func(t *testing.T) {
		t.Parallel()
		w, rec := newShipWorld(t, nil, onTick(0, toFar(Line{Commodity: grain, Quantity: 2})))
		step(t, w, 1)
		if err := w.removeAgent(&w.agents[shipHome], CauseConsumer); err != nil {
			t.Fatalf("removeAgent() = %v, want nil", err)
		}
		if w.transport.isAway(shipHome) || len(w.transport.ships) != 1 {
			t.Fatalf("after the removal: away %v, %d shipments, want false and 1", w.transport.isAway(shipHome), len(w.transport.ships))
		}
		step(t, w, 2)
		if got := of[ShipmentArrived](rec); len(got) != 1 || got[0].Diverted || got[0].Lines[0].Delivered != 2 {
			t.Errorf("ShipmentArrived = %+v, want the 2 delivered to the consignee", got)
		}
		if len(of[AgentRelocated](rec)) != 0 {
			t.Error("a removed traveller was relocated")
		}
	})
	t.Run("shipper of a hired shipment", func(t *testing.T) {
		t.Parallel()
		w, rec := newShipWorld(t, nil, onTick(0, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 2}}}))
		give(t, w, shipHome, commodityOf(t, w, "haul"), 4)
		step(t, w, 1)
		if err := w.removeAgent(&w.agents[shipHome], CauseConsumer); err != nil {
			t.Fatalf("removeAgent() = %v, want nil", err)
		}
		step(t, w, 2)
		if got := of[ShipmentArrived](rec); len(got) != 1 || got[0].Lines[0].Delivered != 2 {
			t.Errorf("ShipmentArrived = %+v, want the 2 delivered", got)
		}
	})
}

// TestRelocation_OpenShipmentRefusal verifies spec-transport.md 5.5.3: the public
// relocation of an away traveller, of a shipper and of a consignee is refused with
// ErrRegionBound naming open_shipment; the instance key of the shipment skips it; an
// arrival whose traveller holds another obligation is held and retried each tick.
//
// Requirement: FR-TRN-022, FR-REG-059, FR-REG-060, FR-WLD-055
func TestRelocation_OpenShipmentRefusal(t *testing.T) {
	t.Parallel()
	w, rec := newShipWorld(t, nil, onTick(0, toFar(Line{Commodity: grain, Quantity: 1})))
	step(t, w, 1)
	for _, id := range []AgentID{shipHome, shipFar} {
		reason, err := w.RelocationRefusal(id, regIsle)
		var rb *RegionBoundError
		if reason != string(ReasonRegionBound) || !errors.As(err, &rb) || rb.Obligation != openShipment {
			t.Errorf("RelocationRefusal(%d) = (%q, %v), want region_bound naming open_shipment", id, reason, err)
		}
	}
	if reason, err := w.RelocationRefusal(shipHome, regIsle, "open_shipment:0"); err != nil {
		t.Errorf("RelocationRefusal with the instance key = (%q, %v), want none", reason, err)
	}
	if _, err := w.RelocationRefusal(shipHome, regIsle, "open_shipment:1"); err == nil {
		t.Error("RelocationRefusal skipping another shipment's key = nil, want the refusal")
	}
	// Another obligation of the traveller holds the arrival.
	w.bounds.add("other", func(a *agent, _ []string) bool { return a.id == shipHome })
	step(t, w, 2)
	if len(of[AgentRelocated](rec)) != 0 || len(w.transport.ships) != 1 || !w.transport.isAway(shipHome) {
		t.Fatalf("an arrival with another obligation moved: ships %d, away %v", len(w.transport.ships), w.transport.isAway(shipHome))
	}
	step(t, w, 1)
	if len(w.transport.ships) != 1 || w.transport.ships[0].arrives != w.Tick()+0 {
		t.Errorf("held shipment arrives at %d, want it retried at tick %d", w.transport.ships[0].arrives, w.Tick())
	}
}

// awayProbe is a Strategy that counts its calls by kind for one agent.
type awayProbe struct {
	ship        func(v *AgentView) []ShipmentRequest
	work, trade map[int64]int
	revised     int
	probe       AgentID
}

func (s *awayProbe) Work(v *AgentView, _ *rand.Rand) JobRequest {
	if v.Agent == s.probe {
		s.work[v.Tick.Tick]++
	}
	return JobRequest{NoJob: true}
}

func (s *awayProbe) Trade(v *AgentView, _ *rand.Rand, p *TradePlan) {
	if v.Agent == s.probe {
		s.trade[v.Tick.Tick]++
		p.Shipments = append(p.Shipments, s.ship(v)...)
		if v.Tick.Tick == 0 {
			p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Ask, Quantity: 1, Limit: 5_000})
		}
	}
}

func (s *awayProbe) Revise(o *OutcomeView, _ *rand.Rand) Belief {
	if o.Agent == s.probe {
		s.revised++
	}
	return o.Belief
}

var _ Strategy = (*awayProbe)(nil)

// TestAway_WorkAndTradeSkipped verifies spec-transport.md 5.5.2 and FR-TRN-021: an
// away traveller is passed to neither Work nor Trade from the tick after the dispatch
// to the tick before the arrival, places no order, still has Revise for the order of
// the dispatch tick, is on a working tick every tick away, and eats from its account.
//
// Arithmetic: dispatched in tick 0, arriving in tick 2: away in ticks 1 only for the
// calls (tick 0 is the dispatch itself, tick 2 starts after the arrival), working in
// ticks 0 and 1 (two), the grain eaten is one a tick: 10 less 3 after ticks 0 to 2
// (the ask of 1 not sold, so none left the account).
//
// Requirement: FR-TRN-021
func TestAway_WorkAndTradeSkipped(t *testing.T) {
	t.Parallel()
	s := &awayProbe{work: map[int64]int{}, trade: map[int64]int{}, probe: shipHome,
		ship: onTick(0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true})}
	c := shipConfig()
	w := newMill(t, c, WithStrategy("script", s), WithAudit())
	step(t, w, 1)
	if got := w.agents[shipHome].history.WorkingTicks(); got != 1 {
		t.Errorf("working ticks after the dispatch tick = %d, want 1", got)
	}
	step(t, w, 1)
	if s.work[1] != 0 || s.trade[1] != 0 {
		t.Errorf("tick 1 calls: work %d, trade %d, want none for an away traveller", s.work[1], s.trade[1])
	}
	if got := w.agents[shipHome].history.WorkingTicks(); got != 2 {
		t.Errorf("working ticks after the tick away = %d, want 2", got)
	}
	step(t, w, 1)
	if s.work[2] != 1 || s.trade[2] != 1 {
		t.Errorf("tick 2 calls: work %d, trade %d, want 1 and 1 after the arrival", s.work[2], s.trade[2])
	}
	if got := w.agents[shipHome].history.WorkingTicks(); got != 2 {
		t.Errorf("working ticks after the arrival tick = %d, want still 2", got)
	}
	if s.revised != 1 {
		t.Errorf("Revise calls = %d, want 1 for the order of tick 0", s.revised)
	}
	if got := held(t, w, shipHome, grain); got != 7 {
		t.Errorf("grain held = %d, want 10 - 3 ticks of upkeep", got)
	}
}

// TestAway_OutOfRangeIDIsNotAway verifies that the public Away answers false, and
// does not panic, for an ID below 0 or past the issued agents.
//
// Guard: Away takes any AgentID from a consumer; a negative one indexed the away
// table and panicked.
func TestAway_OutOfRangeIDIsNotAway(t *testing.T) {
	t.Parallel()
	w := newMill(t, shipConfig(), WithStrategy("script", &awayProbe{work: map[int64]int{}, trade: map[int64]int{}, probe: shipHome}))
	for _, id := range []AgentID{-1, 1 << 40} {
		if w.Away(id) {
			t.Errorf("Away(%d) = true, want false", id)
		}
	}
}

// TestAudit_Transport verifies that each check of spec-transport.md 7.4.1 fails when
// its state is corrupted and names the check, wrapping ErrAuditFailed.
//
// Requirement: FR-TRN-032
func TestAudit_Transport(t *testing.T) {
	t.Parallel()
	// put adds a unit of grain from nowhere to the Transit account of the shipment.
	put := func(w *World) {
		give(t, w, shipHome, grain, 1)
		if err := w.ledger.Move(w.agents[shipHome].account, w.transport.ships[0].account, grain, 1, ledger.TransitOut); err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
	tests := []struct {
		name    string
		corrupt func(w *World)
	}{
		{"arrives in the past", func(w *World) { w.transport.ships[0].arrives = -1 }},
		{"ids out of order", func(w *World) { w.transport.ships = append(w.transport.ships, w.transport.ships[0]) }},
		{"away without a trip", func(w *World) { w.transport.setAway(shipFar, true) }},
		{"traveller not away", func(w *World) { w.transport.setAway(shipHome, false) }},
		{"free account not empty", func(w *World) {
			put(w)
			w.transport.ships = w.transport.ships[:0]
			w.transport.setAway(shipHome, false)
		}},
		{"account holds more than shipped", put},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _ := newShipWorld(t, nil, onTick(0, toFar(Line{Commodity: grain, Quantity: 1})))
			step(t, w, 1)
			if err := w.Audit(); err != nil {
				t.Fatalf("Audit() before the corruption = %v, want nil", err)
			}
			tt.corrupt(w)
			if err := w.Audit(); !errors.Is(err, ErrAuditFailed) {
				t.Errorf("Audit() = %v, want one wrapping ErrAuditFailed", err)
			}
		})
	}
}

// TestPositions_TransportEntries verifies that P1.3 and P3.1 are positions of the
// straight line owned by TRN, in the order of the table, and are no longer
// listed as not built.
//
// Requirement: FR-WLD-004
func TestPositions_TransportEntries(t *testing.T) {
	t.Parallel()
	var ids []string
	for _, p := range positionList {
		if p.owner == "TRN" {
			ids = append(ids, p.id)
			if p.where != line {
				t.Errorf("%s runs at %v, want a line call", p.id, p.where)
			}
		}
	}
	if !slices.Equal(ids, []string{"P1.3", "P3.1"}) {
		t.Errorf("TRN positions = %v, want [P1.3 P3.1]", ids)
	}
	for _, n := range notBuiltPositions {
		if n == "P1.3" || n == "P3.1" {
			t.Errorf("%s is still in notBuiltPositions", n)
		}
	}
}

// TestArrivals_ScratchWithoutObserver verifies spec-transport.md 13.6 and D459: the
// record of an arrival kept for the later positions is the same with and without an
// observer.
//
// Requirement: NFR-TRN-004
func TestArrivals_ScratchWithoutObserver(t *testing.T) {
	t.Parallel()
	run := func(observed bool) ([]arrival, []ArrivedLine) {
		w, _ := newShipWorld(t, nil, onTick(0, toFar(Line{Commodity: grain, Quantity: 2})))
		if !observed {
			w.obs = observers{}
		}
		step(t, w, 2)
		if err := w.beginStep(); err != nil {
			t.Fatalf("beginStep() = %v", err)
		}
		w.p13Arrivals()
		a, l := slices.Clone(w.transport.arrived), slices.Clone(w.transport.arrivedLines)
		if err := w.endStep(nil); err != nil {
			t.Fatalf("endStep() = %v", err)
		}
		return a, l
	}
	a1, l1 := run(true)
	a2, l2 := run(false)
	if len(a1) != 1 || !reflect.DeepEqual(a1, a2) || !reflect.DeepEqual(l1, l2) || l1[0].Delivered != 2 {
		t.Errorf("arrivals with %v %v and without %v %v, want one record, equal", a1, l1, a2, l2)
	}
}

// TestArrivals_CostFollowsArrivals verifies NFR-TRN-002 at the position: a tick
// with no arrival costs the schedule one comparison, and the tick of two arrivals
// costs the logarithm more with 500 more shipments in flight, because P1.3 asks the
// schedule for what is due and never scans the shipments.
//
// Arithmetic: both trips of twoTrips are dispatched in tick 0 with k = 2, so tick 1
// has nothing due and tick 2 has two arrivals (three Due calls, the last finding none).
//
// Requirement: NFR-TRN-002
func TestArrivals_CostFollowsArrivals(t *testing.T) {
	t.Parallel()
	cost := func(extra int) (idle, busy int64) {
		w, _ := newShipWorld(t, nil, twoTrips)
		for i := range extra {
			w.transport.due.Push(1_000+int64(i), 1_000+int64(i)) // in flight, far from due
		}
		step(t, w, 1)
		before := w.transport.due.Steps()
		w.transport.arrived = w.transport.arrived[:0]
		w.p13Arrivals()
		idle = w.transport.due.Steps() - before
		step(t, w, 1)
		before = w.transport.due.Steps()
		w.p13Arrivals()
		return idle, w.transport.due.Steps() - before
	}
	idle0, busy0 := cost(0)
	idle1, busy1 := cost(500)
	if idle0 != 1 || idle1 != 1 {
		t.Errorf("P1.3 with nothing due cost %d and %d comparisons with 0 and 500 more in flight, want 1 and 1", idle0, idle1)
	}
	if busy1 > busy0+50 {
		t.Errorf("P1.3 with two arrivals cost %d comparisons with 500 more in flight and %d without, want a difference of at most two pops sifting down log2(500) levels (50)", busy1, busy0)
	}
}

// TestArrival_AnchorsLagOneTick verifies N-322 for the arrivals of P1.3: the
// region-keyed tick scratch (rings, runnable) is built at P1.1 and P1.2, before the
// arrival, so the arriving household works its arrival tick on anchors that did not
// count it, and the next tick's P1.2 does.
//
// Arithmetic: two households live in each region; the traveller leaves in tick 0 and
// arrives in tick 2. The far ring has 2 entries in tick 2 (after the move there are
// 3 live in the region) and 3 in tick 3; the home ring has 2 in tick 2 (the traveller
// is gone) and, since it is rebuilt, 1 in tick 3.
//
// Requirement: FR-TRN-024
func TestArrival_AnchorsLagOneTick(t *testing.T) {
	t.Parallel()
	s := &awayProbe{work: map[int64]int{}, trade: map[int64]int{}, probe: shipHome,
		ship: onTick(0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true})}
	w := newMill(t, shipConfig(), WithStrategy("script", s), WithAudit())
	step(t, w, 3) // ticks 0 to 2: the arrival is at P1.3 of tick 2
	if got := w.regionOf(&w.agents[shipHome]); got != regFar {
		t.Fatalf("region of the traveller after tick 2 = %d, want %d", got, regFar)
	}
	if got := len(w.tk.rings[regFar]); got != 2 {
		t.Errorf("far ring in the arrival tick = %d, want 2 (built before the arrival)", got)
	}
	step(t, w, 1)
	if got := len(w.tk.rings[regFar]); got != 3 {
		t.Errorf("far ring the tick after = %d, want 3", got)
	}
}
