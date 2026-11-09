package economysim

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// This file tests the state side of the transport module (transport_state.go, plan
// 2-12 Phase 7) over the world of transport_request_internal_test.go: the snapshot
// rows, the checkpoint section of tag 17 and the cases a restore refuses.

// twoTrips is the shipping of the tests below, in tick 0: agent 3 sends 3 grain to
// agent 5 on foot, agent 4 goes to far with its own goods (a trip to oneself, no
// Transit account). Both are away for two ticks.
func twoTrips(v *AgentView) []ShipmentRequest {
	switch {
	case v.Tick.Tick != 0:
		return nil
	case v.Agent == shipHome:
		return []ShipmentRequest{toFar(Line{Commodity: grain, Quantity: 3})}
	case v.Agent == shipHome2:
		return []ShipmentRequest{{Mode: modeFoot, To: regFar, ToSelf: true}}
	}
	return nil
}

// restoreShip restores a checkpoint of a ship world, whose roles name the strategy
// "script".
func restoreShip(w *World) (*World, error) {
	c, err := w.Checkpoint()
	if err != nil {
		return nil, err
	}
	return RestoreWorld(w.Config(), c, WithStrategy("script", &shipScript{ship: twoTrips}))
}

// TestSnapshot_Transport verifies the rows of spec-observability.md 5.3.7 and
// 5.3.8: the shipments in flight with their ticks left and their lines, the agents
// away and the factors that are not 10 000, empty with nothing in flight, and the goods
// in transit in the shipment row and not in a market row.
//
// Arithmetic: dispatched in tick 0 with k = ceil(40 / 20) = 2, so after the step of
// tick 0 the arrival is at tick 2 and the ticks left are 2 - 1 = 1. Agent 3 held 10
// grain and ships 3.
//
// Requirement: FR-TRN-034, FR-OBS-020
func TestSnapshot_Transport(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, nil, twoTrips)
	s0, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	if s0.Shipments != nil || s0.Away != nil || s0.TransitFactors != nil {
		t.Errorf("snapshot before any dispatch = %+v %+v %+v, want nothing in flight", s0.Shipments, s0.Away, s0.TransitFactors)
	}
	if err := w.setTransitFactor(regFar, regHome, modeFoot, 5_000, CauseConsumer); err != nil {
		t.Fatalf("setTransitFactor() = %v, want nil", err)
	}
	step(t, w, 1)
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	acct := w.transport.ships[0].account
	want := []ShipmentRow{
		{Shipment: 0, Mode: modeFoot, Region: regHome, To: regFar, Shipper: shipHome, Traveller: shipHome, Consignee: shipFar,
			Dispatched: 0, Arrives: 2, TicksLeft: 1, Account: acct, Units: 1, Lines: []Amount{{Commodity: grain, Quantity: 3}}},
		{Shipment: 1, Mode: modeFoot, Region: regHome, To: regFar, Shipper: shipHome2, Traveller: shipHome2, Consignee: shipHome2,
			Dispatched: 0, Arrives: 2, TicksLeft: 1, Account: -1, Units: 1, Lines: []Amount{}},
	}
	if !reflect.DeepEqual(s.Shipments, want) {
		t.Errorf("Snapshot().Shipments = %+v, want %+v", s.Shipments, want)
	}
	if want := []AgentID{shipHome, shipHome2}; !reflect.DeepEqual(s.Away, want) {
		t.Errorf("Snapshot().Away = %v, want %v", s.Away, want)
	}
	if want := []TransitFactorRow{{From: regFar, To: regHome, Mode: modeFoot, BP: 5_000}}; !reflect.DeepEqual(s.TransitFactors, want) {
		t.Errorf("Snapshot().TransitFactors = %+v, want %+v", s.TransitFactors, want)
	}
	// The 3 grain in the Transit account are in no market row: the market rows
	// count what the agents hold, which the shipment's lines are not.
	var inMarkets, inAgents int64
	for _, m := range s.Markets {
		if m.State.Commodity == grain {
			inMarkets += int64(m.HeldByAgents)
		}
	}
	for id := range w.agents {
		if w.agents[id].weight >= 1 {
			inAgents += held(t, w, AgentID(id), grain)
		}
	}
	if inMarkets != inAgents {
		t.Errorf("grain in the market rows = %d, held by the agents = %d, want equal: the shipped grain is in neither", inMarkets, inAgents)
	}
	s.Shipments[0].Lines[0].Quantity = 99
	if w.transport.ships[0].lines[0].Quantity != 3 {
		t.Error("Snapshot().Shipments shares its lines with the world, want a deep copy")
	}
}

// TestSnapshot_TransportOff verifies that a world with the module off has no
// shipment, away or factor row.
//
// Requirement: FR-OBS-020
func TestSnapshot_TransportOff(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, func(c *Config) { c.Modules.Transport = nil }, nil)
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	if s.Shipments != nil || s.Away != nil || s.TransitFactors != nil {
		t.Errorf("snapshot with the module off = %+v %+v %+v, want empty", s.Shipments, s.Away, s.TransitFactors)
	}
}

// TestCheckpoint_TransportRoundTrips verifies tag 17: a world with a cargo and a trip
// to oneself in flight and a factor set restores to the same shipments, factors,
// away flags and free lists, its own checkpoint equals the one it came from, and the
// two worlds then arrive the same.
//
// Requirement: FR-TRN-036, FR-CHK-005
func TestCheckpoint_TransportRoundTrips(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, nil, twoTrips)
	if err := w.setTransitFactor(regHome, regFar, modeCoach, 7_500, CauseConsumer); err != nil {
		t.Fatalf("setTransitFactor() = %v, want nil", err)
	}
	// A free Transit account: one cargo has come and gone before the checkpoint.
	id, err := w.ledger.Open(ledger.KindTransit, 0, int(regFar))
	if err != nil {
		t.Fatalf("Open() = %v, want nil", err)
	}
	w.transport.free[regFar].Put(id)
	step(t, w, 1)
	r, err := restoreShip(w)
	if err != nil {
		t.Fatalf("RestoreWorld() = %v, want nil", err)
	}
	if !reflect.DeepEqual(r.transport.ships, w.transport.ships) {
		t.Errorf("restored shipments = %+v, want %+v", r.transport.ships, w.transport.ships)
	}
	if !reflect.DeepEqual(r.transport.factors, w.transport.factors) || r.transport.next != w.transport.next {
		t.Errorf("restored factors and next = %+v %d, want %+v %d", r.transport.factors, r.transport.next, w.transport.factors, w.transport.next)
	}
	for id := range w.agents {
		if r.transport.isAway(AgentID(id)) != w.transport.isAway(AgentID(id)) {
			t.Errorf("agent %d away = %t after the restore, want %t", id, r.transport.isAway(AgentID(id)), w.transport.isAway(AgentID(id)))
		}
	}
	for g := range w.transport.free {
		if r.transport.free[g].Len() != w.transport.free[g].Len() {
			t.Errorf("free Transit accounts of region %d = %d after the restore, want %d", g, r.transport.free[g].Len(), w.transport.free[g].Len())
		}
	}
	if r.transport.free[regFar].Len() != 1 {
		t.Errorf("free list of far = %d, want the account opened before the checkpoint", r.transport.free[regFar].Len())
	}
	a, _ := w.Checkpoint()
	b, _ := r.Checkpoint()
	if ab, _ := a.MarshalBinary(); len(ab) == 0 {
		t.Fatal("empty checkpoint")
	}
	if !reflect.DeepEqual(a, b) {
		t.Error("the restored world's checkpoint differs from the one it was restored from")
	}
	step(t, w, 2)
	step(t, r, 2)
	if !reflect.DeepEqual(r.transport.ships, w.transport.ships) || r.transport.next != w.transport.next || r.Tick() != w.Tick() {
		t.Errorf("after the arrivals: shipments %+v next %d, want %+v next %d", r.transport.ships, r.transport.next, w.transport.ships, w.transport.next)
	}
	if len(w.transport.ships) != 0 {
		t.Errorf("shipments in flight at tick %d = %d, want 0: both trips took 2 ticks", w.Tick(), len(w.transport.ships))
	}
}

// TestCheckpoint_TransportCorrupt verifies that a restore refuses a section 17 that
// no world could have written: each case edits the state of a world with two
// shipments in flight and one trip to oneself before it is saved, and the restore is
// ErrCheckpointCorrupt. The Transit account that holds more than was shipped, that is
// of another region or shared is FR-CHK-023 (the audit of the restore).
//
// Requirement: FR-CHK-023, FR-CHK-016, FR-TRN-036
func TestCheckpoint_TransportCorrupt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		edit func(w *World)
	}{
		{"next shipment not above the shipments", func(w *World) { w.transport.next = 1 }},
		{"shipments out of order", func(w *World) { w.transport.ships[0].id, w.transport.ships[1].id = 1, 0 }},
		{"mode beyond the configuration", func(w *World) { w.transport.ships[0].mode = 9 }},
		{"hired flag against the mode", func(w *World) { w.transport.ships[0].hired = true }},
		{"between a region and itself", func(w *World) { w.transport.ships[0].to = w.transport.ships[0].from }},
		{"shipper that was never created", func(w *World) { w.transport.ships[0].shipper = w.state.nextAgent }},
		{"consignee that was never created", func(w *World) { w.transport.ships[0].consignee = w.state.nextAgent + 5 }},
		{"traveller other than the shipper", func(w *World) { w.transport.ships[0].traveller = shipHome2 }},
		{"haul units on a self-carried shipment", func(w *World) { w.transport.ships[0].haulUnits = 3 }},
		{"trip to oneself with another consignee", func(w *World) { w.transport.ships[1].consignee = shipFar }},
		{"dispatch in the future", func(w *World) { w.transport.ships[0].dispatched = w.Tick() }},
		{"arrival not after the dispatch", func(w *World) { w.transport.ships[0].arrives = 0 }},
		{"arrival too far ahead", func(w *World) { w.transport.ships[0].arrives = w.Tick() + MaxTravelTicks + 1 }},
		{"no carrying units", func(w *World) { w.transport.ships[0].units = 0 }},
		{"Transit account beyond the ledger", func(w *World) { w.transport.ships[0].account = w.ledger.Len() + 3 }},
		{"lines without an account", func(w *World) { w.transport.ships[0].account = -1 }},
		{"trip to oneself with an account", func(w *World) { w.transport.ships[1].account = w.transport.ships[0].account }},
		{"line of 0 units", func(w *World) { w.transport.ships[0].lines[0].Quantity = 0 }},
		{"line of an unknown commodity", func(w *World) { w.transport.ships[0].lines[0].Commodity = CommodityID(len(w.cfg.Commodities)) }},
		{"Transit account of another region", func(w *World) { w.transport.ships[0].from = regFar; w.transport.ships[0].to = regHome }},
		{"account that is not a Transit account", func(w *World) { w.transport.ships[0].account = w.agents[shipHome].account }},
		{"account that holds more than was shipped", func(w *World) {
			if err := w.ledger.Move(w.agents[shipFar].account, w.transport.ships[0].account, grain, 5, ledger.TransitOut); err != nil {
				t.Fatalf("Move() = %v, want nil", err)
			}
		}},
		{"free Transit account that holds goods", func(w *World) {
			id, err := w.ledger.Open(ledger.KindTransit, 0, int(regHome))
			if err == nil {
				err = w.ledger.Move(w.agents[shipFar].account, id, grain, 1, ledger.TransitOut)
			}
			if err != nil {
				t.Fatalf("filling a free Transit account = %v, want nil", err)
			}
		}},
		{"factor at the default", func(w *World) {
			w.transport.factors = []factorRec{{from: regHome, to: regFar, mode: modeFoot, bp: defaultTransitBP}}
		}},
		{"factor above the bound", func(w *World) {
			w.transport.factors = []factorRec{{from: regHome, to: regFar, mode: modeFoot, bp: MaxTransitFactorBP + 1}}
		}},
		{"factor between a region and itself", func(w *World) {
			w.transport.factors = []factorRec{{from: regHome, to: regHome, mode: modeFoot, bp: 5_000}}
		}},
		{"factor of an unknown mode", func(w *World) {
			w.transport.factors = []factorRec{{from: regHome, to: regFar, mode: 7, bp: 5_000}}
		}},
		{"factors out of order", func(w *World) {
			w.transport.factors = []factorRec{{from: regFar, to: regHome, mode: modeFoot, bp: 5_000}, {from: regHome, to: regFar, mode: modeFoot, bp: 5_000}}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _ := newShipWorld(t, nil, twoTrips)
			step(t, w, 1)
			tt.edit(w)
			_, err := restoreShip(w)
			if !errors.Is(err, ErrCheckpointCorrupt) {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt", tt.name, err)
			}
			t.Log(err)
		})
	}
}

// TestCheckpoint_TransportHeldArrival verifies that a shipment whose arrival was held
// (the arrival tick moved to the next tick) restores with that tick, and arrives as
// the original does.
//
// Requirement: FR-TRN-036, FR-TRN-022
func TestCheckpoint_TransportHeldArrival(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, nil, twoTrips)
	step(t, w, 1)
	w.transport.ships[0].arrives++ // as an arrival held by a refusal does
	c, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() = %v, want nil", err)
	}
	r, err := RestoreWorld(w.Config(), c, WithStrategy("script", &shipScript{}))
	if err != nil {
		t.Fatalf("RestoreWorld() = %v, want nil", err)
	}
	if got, want := r.transport.ships[0].arrives, w.transport.ships[0].arrives; got != want {
		t.Errorf("restored arrival tick = %d, want %d", got, want)
	}
}
