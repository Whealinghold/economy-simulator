package economysim

import (
	"log/slog"
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file tests the request and the dispatch of the transport module
// (transport_request.go, plan 2-12 Phase 4) over a small world: three regions,
// two of them joined by edges of 40 and one with no edge; two households in each of
// the first two. The goods are those of spec-transport.md 17 where they are
// needed: grain (mass 3, bulk 2, the upkeep of one household), fodder (2, 3),
// fresh (1, 1, shelf life 3), a wagon (durability 400) and haul units.

// shipScript is a Strategy that requests no job and asks for the shipments that
// ship says (a function gates itself on the tick).
type shipScript struct {
	ship func(v *AgentView) []ShipmentRequest
}

func (*shipScript) Work(*AgentView, *rand.Rand) JobRequest { return JobRequest{NoJob: true} }

func (s *shipScript) Trade(v *AgentView, _ *rand.Rand, p *TradePlan) {
	if s.ship != nil {
		p.Shipments = append(p.Shipments, s.ship(v)...)
	}
}

func (*shipScript) Revise(o *OutcomeView, _ *rand.Rand) Belief { return o.Belief }

var _ Strategy = (*shipScript)(nil)

// The households of shipConfig: two in home (agents 3 and 4) and two in far (5 and
// 6); the three estates are agents 0 to 2.
const (
	shipHome  AgentID = 3
	shipHome2 AgentID = 4
	shipFar   AgentID = 5
)

// The regions and the modes of shipConfig, by ID.
const (
	regHome RegionID = 0
	regFar  RegionID = 1
	regIsle RegionID = 2
)

const (
	modeCoach ModeID = iota
	modeFoot
	modeWagon
)

// shipConfig returns the world described above, with the modes coach (hired,
// span 10), foot (one household a unit) and wagon (crew 2, a wagon item, wear 5
// and 3 fodder a tick), the role's strategy named "script".
func shipConfig() Config {
	c := millConfig(1, 0)
	c.Commodities[0].Mass, c.Commodities[0].Bulk = 3, 2 // grain
	c.Commodities[1].Mass, c.Commodities[1].Bulk = 3, 2 // flour
	c.Commodities = append(c.Commodities,
		Commodity{ID: "hay", Mass: 2, Bulk: 3, Raw: true, ReferencePrice: 50},
		Commodity{ID: "perish", Mass: 1, Bulk: 1, ShelfLife: 3, Raw: true, ReferencePrice: 100},
		Commodity{ID: "haul", Mass: 0, Bulk: 0, Raw: true, ReferencePrice: 400},
		Commodity{ID: "wagon", Mass: 10, Bulk: 10, Durability: 400, Raw: true, ReferencePrice: 6_000})
	// A recipe that uses the four, so that none is an orphan commodity.
	c.Recipes = append(c.Recipes, Recipe{ID: "pack", Duration: 1, TraineeYield: 5_000,
		Inputs:   []Item{{Commodity: "hay", Quantity: 1}, {Commodity: "perish", Quantity: 1}},
		Outputs:  []Output{{Commodity: "haul", Quantity: 1, CostShare: 10_000}},
		Requires: []Tool{{Commodity: "wagon", WearPerUse: 1}}})
	c.Roles[0].Recipes = []string{"mill", "pack"}
	// The regions are named so that the canonical order (by ID) is home, far, isle.
	c.Regions[0].ID = "a_home"
	isle := DefaultRegion()
	isle.ID = "c_isle"
	far := DefaultRegion()
	far.ID = "b_far"
	c.Regions = append(c.Regions, far, isle)
	c.Edges = []Edge{{From: "a_home", To: "b_far", Distance: 40}, {From: "b_far", To: "a_home", Distance: 40}}
	c.Roles[0].Strategy = "script"
	c.Groups = []Group{{Region: "a_home", Role: "miller", Households: 2}, {Region: "b_far", Role: "miller", Households: 2}}
	c.Modules.Transport = &Transport{Modes: []Mode{
		{ID: "coach", Speed: 30, UnitMass: 1_200, UnitBulk: 900, Haul: "haul", HaulSpan: 10},
		{ID: "foot", Speed: 20, UnitMass: 40, UnitBulk: 30, Crew: 1},
		{ID: "wagon", Speed: 30, UnitMass: 1_200, UnitBulk: 900, Crew: 2, Vehicle: "wagon", WearPerTick: 5,
			Feed: []Item{{Commodity: "hay", Quantity: 3}}},
	}}
	return c
}

// newShipWorld builds the world of shipConfig, changed by mut, with a recording
// observer and the audit on, whose households follow ship.
func newShipWorld(t *testing.T, mut func(c *Config), ship func(v *AgentView) []ShipmentRequest, opts ...Option) (*World, *evLog) {
	t.Helper()
	c := shipConfig()
	if mut != nil {
		mut(&c)
	}
	rec := &evLog{}
	opts = append([]Option{WithObserver(rec), WithAudit(), WithStrategy("script", &shipScript{ship: ship})}, opts...)
	w := newMill(t, c, opts...)
	// Flows are totalled per tick from here, not in the construction bucket.
	if err := w.ledger.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction() = %v, want nil", err)
	}
	return w, rec
}

// commodityOf returns the ID of a commodity of the world by its string.
func commodityOf(t *testing.T, w *World, id string) CommodityID {
	t.Helper()
	c, ok := w.cfg.commodityID(id)
	if !ok {
		t.Fatalf("commodity %q is not in the configuration", id)
	}
	return c
}

// give puts qty units of a commodity into the account of an agent, from the
// initial boundary, as a test fixture.
func give(t *testing.T, w *World, id AgentID, commodity CommodityID, qty int64) {
	t.Helper()
	src, ok := w.ledger.Boundary(ledger.Initial, int(commodity))
	if !ok {
		t.Fatalf("no initial boundary for commodity %d", commodity)
	}
	if err := w.ledger.Move(src, w.agents[id].account, int(commodity), qty, ledger.Initial); err != nil {
		t.Fatalf("give(%d x %d to agent %d) = %v, want nil", qty, commodity, id, err)
	}
}

// clearGoods empties an agent's account of a commodity, to the disposal boundary.
func clearGoods(t *testing.T, w *World, id AgentID, commodity CommodityID) {
	t.Helper()
	n, _, err := w.ledger.Holding(w.agents[id].account, int(commodity))
	if err != nil {
		t.Fatalf("Holding() = %v, want nil", err)
	}
	sink, ok := w.ledger.Boundary(ledger.Disposal, int(commodity))
	if !ok {
		t.Fatalf("no disposal boundary for commodity %d", commodity)
	}
	if n > 0 {
		if err := w.ledger.Move(w.agents[id].account, sink, int(commodity), n, ledger.Disposal); err != nil {
			t.Fatalf("Move to disposal = %v, want nil", err)
		}
	}
}

// ask dispatches one request of agent id as request number idx, from its own
// region.
func ask(w *World, id AgentID, idx int, r ShipmentRequest) (ShipmentID, Reason) {
	a := &w.agents[id]
	return w.dispatch(a, idx, &r, w.regionOf(a))
}

// TestRequest_RefusalOrder verifies the order of the refusals of
// spec-transport.md 5.4: for each case the first reason that holds is the one
// returned, and a request that breaks only that rule is refused with it.
//
// Arithmetic: the household holds 10 grain (mass 3, bulk 2: load (30, 20)), so on
// foot (unit (40, 30), crew 1, one unit) the load fits. 11 grain is (33, 22): still
// fits; 14 grain is (42, 28): mass 42 over 40, so capacity_exceeded. Hired, a line
// of 3 grain needs ceil(9 / 1200) = 1 unit and 1 * ceil(40 / 10) = 4 haul units; the
// household holds none, so insufficient_haul.
//
// Requirement: FR-TRN-006, FR-TRN-008, FR-TRN-017, FR-TRN-018, FR-TRN-019, FR-TRN-045, FR-TRN-046
func TestRequest_RefusalOrder(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(t *testing.T, w *World)
		req   ShipmentRequest
		want  Reason
	}{
		{"mode below zero", nil, ShipmentRequest{Mode: -1, To: regFar, ToSelf: true}, ReasonUnknownMode},
		{"mode past the last", nil, ShipmentRequest{Mode: 3, To: regFar, ToSelf: true}, ReasonUnknownMode},
		{"same region", nil, ShipmentRequest{Mode: modeFoot, To: regHome, ToSelf: true}, ReasonSameRegion},
		{"unknown region", nil, ShipmentRequest{Mode: modeFoot, To: 9, ToSelf: true}, ReasonUnknownRegion},
		{"region below zero", nil, ShipmentRequest{Mode: modeFoot, To: -1, ToSelf: true}, ReasonUnknownRegion},
		{"no route", nil, ShipmentRequest{Mode: modeFoot, To: regIsle, ToSelf: true}, ReasonNoRoute},
		{"unknown consignee below the households", nil, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: 1, Lines: []Line{{Commodity: grain, Quantity: 1}}}, ReasonUnknownConsignee},
		{"unknown consignee past the last", nil, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: 99, Lines: []Line{{Commodity: grain, Quantity: 1}}}, ReasonUnknownConsignee},
		{"consignee elsewhere", nil, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: shipHome2, Lines: []Line{{Commodity: grain, Quantity: 1}}}, ReasonConsigneeElsewhere},
		{"hired to oneself", nil, ShipmentRequest{Mode: modeCoach, To: regFar, ToSelf: true}, ReasonConsigneeElsewhere},
		{"lines with oneself", nil, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true, Lines: []Line{{Commodity: grain, Quantity: 1}}}, ReasonLinesWithSelf},
		{"empty to another", nil, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: shipFar}, ReasonEmptyShipment},
		{"too many lines", nil, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: shipFar, Lines: make([]Line, MaxShipmentLines+1)}, ReasonTooManyLines},
		{"unknown commodity", nil, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: 99, Quantity: 1}}}, ReasonUnknownCommodity},
		{"quantity zero", nil, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 0}}}, ReasonInvalidQuantity},
		{"quantity over the bound", nil, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: MaxQuantity + 1}}}, ReasonInvalidQuantity},
		{"commodity twice", nil, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 1}, {Commodity: grain, Quantity: 1}}}, ReasonDuplicateLine},
		{"stock short", nil, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 11}}}, ReasonInsufficientStock},
		{"capacity exceeded", func(t *testing.T, w *World) { give(t, w, shipHome, grain, 4) },
			ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}, ReasonCapacityExceeded},
		{"hired without haul", nil, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 3}}}, ReasonInsufficientHaul},
		{"unprovisioned", func(t *testing.T, w *World) { clearGoods(t, w, shipHome, grain) },
			ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}, ReasonUnprovisioned},
		{"upkeep sent away", nil, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 10}}}, ReasonUnprovisioned},
		{"crew short", nil, ShipmentRequest{Mode: modeWagon, To: regFar, ToSelf: true}, ReasonCrewShort},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _ := newShipWorld(t, nil, nil)
			if tt.setup != nil {
				tt.setup(t, w)
			}
			if _, got := ask(w, shipHome, 0, tt.req); got != tt.want {
				t.Errorf("dispatch(%+v) reason = %q, want %q", tt.req, got, tt.want)
			}
		})
	}
}

// TestRequest_RefusalsThatNeedState verifies the refusals that need a state to
// hold: blocked (a factor of 0), too_far (more than MaxTravelTicks), job_in_progress,
// traveller_away, no_vehicle, vehicle_in_cargo, insufficient_feed, the request
// number past MaxDispatchPerTick and every request with the module off.
//
// Arithmetic: the edge is 40; at speed 1 and the factor 1 the time is 40 ticks, so a
// speed of 1 with a factor of 1 (10 000 / 100 000 of the default) is
// ceil(40 * 10 000 / (1 * 1)) = 400 000 ticks, over MaxTravelTicks (100 000).
//
// Requirement: FR-TRN-005, FR-TRN-006, FR-TRN-007, FR-TRN-016, FR-TRN-017, FR-TRN-021
func TestRequest_RefusalsThatNeedState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		mut   func(c *Config)
		setup func(t *testing.T, w *World)
		idx   int
		req   ShipmentRequest
		want  Reason
	}{
		{"blocked", nil, func(_ *testing.T, w *World) {
			w.transport.factors = []factorRec{{from: regHome, to: regFar, mode: modeFoot, bp: 0}}
		},
			0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}, ReasonBlocked},
		{"too far", func(c *Config) { c.Modules.Transport.Modes[1].Speed = 1 },
			func(_ *testing.T, w *World) {
				w.transport.factors = []factorRec{{from: regHome, to: regFar, mode: modeFoot, bp: 1}}
			},
			0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}, ReasonTooFar},
		{"traveller away", nil, func(_ *testing.T, w *World) { w.transport.setAway(shipHome, true) },
			0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}, ReasonTravellerAway},
		{"hired is not stopped by away", nil, func(_ *testing.T, w *World) { w.transport.setAway(shipHome, true) },
			0, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 1}}}, ReasonInsufficientHaul},
		{"job in progress", nil, func(_ *testing.T, w *World) { w.agents[shipHome].job = job{executions: 1, remaining: 2} },
			0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}, ReasonJobInProgress},
		{"one-tick job started this tick", nil, func(_ *testing.T, w *World) { w.welfare.at(int(shipHome)).k = 1 },
			0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}, ReasonJobInProgress},
		{"no vehicle", nil, nil, 0, ShipmentRequest{Mode: modeWagon, To: regFar, ToSelf: true}, ReasonCrewShort},
		{"too many requests", nil, nil, MaxDispatchPerTick, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}, ReasonTooManyRequests},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _ := newShipWorld(t, tt.mut, nil)
			if tt.setup != nil {
				tt.setup(t, w)
			}
			if _, got := ask(w, shipHome, tt.idx, tt.req); got != tt.want {
				t.Errorf("dispatch(%+v) reason = %q, want %q", tt.req, got, tt.want)
			}
		})
	}
}

// TestRequest_VehicleAndFeedRefusals verifies the refusals of a wagon trip: a pair
// of households with no wagon (no_vehicle), a wagon named as cargo (vehicle_in_cargo),
// fodder short of the 3 * n * k of the trip (insufficient_feed).
//
// Arithmetic: the pair has weight 2 and crew 2, so one unit; the edge is 40 at speed
// 30, so k = ceil(40 / 30) = 2 and the feed is 3 * 1 * 2 = 6 fodder.
//
// Requirement: FR-TRN-011, FR-TRN-013, FR-TRN-018
func TestRequest_VehicleAndFeedRefusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(t *testing.T, w *World, wagon, fodder CommodityID)
		req   func(wagon CommodityID) ShipmentRequest
		want  Reason
	}{
		{"no wagon", nil, func(CommodityID) ShipmentRequest { return ShipmentRequest{Mode: modeWagon, To: regFar, ToSelf: true} }, ReasonNoVehicle},
		{"wagon as cargo", func(t *testing.T, w *World, wagon, _ CommodityID) { give(t, w, shipHome, wagon, 1) },
			func(wagon CommodityID) ShipmentRequest {
				return ShipmentRequest{Mode: modeWagon, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: wagon, Quantity: 1}}}
			}, ReasonVehicleInCargo},
		{"feed short", func(t *testing.T, w *World, wagon, fodder CommodityID) {
			give(t, w, shipHome, wagon, 1)
			give(t, w, shipHome, fodder, 5)
		},
			func(CommodityID) ShipmentRequest { return ShipmentRequest{Mode: modeWagon, To: regFar, ToSelf: true} }, ReasonInsufficientFeed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _ := newShipWorld(t, func(c *Config) { c.World.MinAgents = 1; c.Groups[0].Scale = 2 }, nil)
			wagon, fodder := commodityOf(t, w, "wagon"), commodityOf(t, w, "hay")
			if w.agents[shipHome].weight != 2 {
				t.Fatalf("agent %d weight = %d, want 2", shipHome, w.agents[shipHome].weight)
			}
			if tt.setup != nil {
				tt.setup(t, w, wagon, fodder)
			}
			if _, got := ask(w, shipHome, 0, tt.req(wagon)); got != tt.want {
				t.Errorf("dispatch() reason = %q, want %q", got, tt.want)
			}
		})
	}
}

// accounts returns the number of ledger accounts, closed ones included.
func accounts(w *World) int { return w.ledger.Len() }

// TestRequest_RefusalLeavesNoTrace verifies that a refused request opens no
// account, moves nothing, uses no shipment ID and leaves the away flags alone, even
// when it fails at the last check.
//
// Requirement: FR-TRN-017
func TestRequest_RefusalLeavesNoTrace(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, nil, nil)
	before, n := w.ledger.AppendFlows(nil, false), accounts(w)
	grain := commodityOf(t, w, "grain")
	if _, got := ask(w, shipHome, 0, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 3}}}); got != ReasonInsufficientHaul {
		t.Fatalf("reason = %q, want %q", got, ReasonInsufficientHaul)
	}
	if after := w.ledger.AppendFlows(nil, false); !reflect.DeepEqual(after, before) || accounts(w) != n || w.transport.next != 0 || len(w.transport.ships) != 0 {
		t.Errorf("after a refusal: flows %v accounts %d next %d ships %d, want flows %v accounts %d and nothing in flight", after, accounts(w), w.transport.next, len(w.transport.ships), before, n)
	}
	if held(t, w, shipHome, int(grain)) != 10 {
		t.Errorf("grain held = %d, want 10", held(t, w, shipHome, int(grain)))
	}
}

// TestDispatch_SelfCarriedLoad verifies the worked example of spec-transport.md
// 17.2: the whole account is load, so (36, 30) fits one foot unit and 3 fuel in
// place of 1 does not, and a trip to oneself uses no Transit account and keeps the
// goods in the traveller's account.
//
// Arithmetic: 10 grain is (30, 20); 1 fodder is (2, 3) and 4 fresh is (4, 4): (36, 27)
// fits (40, 30), n = max(1, ceil(36 / 40), ceil(27 / 30)) = 1. With 3 fodder in place
// of 1 the bulk is 20 + 9 + 4 = 33 over 30: refused. The edge is 40 at speed 20, so
// k = 2 and the upkeep for the trip is 1 grain * 1 household * 2 = 2, held.
//
// Requirement: FR-TRN-009, FR-TRN-010, FR-TRN-011, FR-TRN-019
func TestDispatch_SelfCarriedLoad(t *testing.T) {
	t.Parallel()
	w, rec := newShipWorld(t, nil, nil)
	fodder, fresh, grain := commodityOf(t, w, "hay"), commodityOf(t, w, "perish"), commodityOf(t, w, "grain")
	give(t, w, shipHome, fodder, 1)
	give(t, w, shipHome, fresh, 4)
	id, reason := ask(w, shipHome, 0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true})
	if reason != "" || id != 0 {
		t.Fatalf("dispatch() = (%d, %q), want (0, \"\")", id, reason)
	}
	s := w.transport.ships[0]
	if s.units != 1 || s.arrives != 2 || s.account != -1 || len(s.lines) != 0 || s.consignee != shipHome || s.traveller != shipHome || s.hired {
		t.Errorf("shipment = %+v, want 1 unit, arrives 2, no account, no line, to itself, not hired", s)
	}
	if !w.transport.isAway(shipHome) || !w.offStage(&w.agents[shipHome]) {
		t.Error("the traveller is not away after the dispatch")
	}
	if held(t, w, shipHome, int(grain)) != 10 || held(t, w, shipHome, int(fresh)) != 4 {
		t.Errorf("the goods left the traveller's account: grain %d, fresh %d, want 10 and 4", held(t, w, shipHome, int(grain)), held(t, w, shipHome, int(fresh)))
	}
	if d := of[ShipmentDispatched](rec); len(d) != 1 || len(d[0].Lines) != 0 {
		t.Errorf("ShipmentDispatched = %+v, want one with no line", d)
	}

	w2, _ := newShipWorld(t, nil, nil)
	give(t, w2, shipHome, fodder, 3)
	give(t, w2, shipHome, fresh, 4)
	if _, got := ask(w2, shipHome, 0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}); got != ReasonCapacityExceeded {
		t.Errorf("with 3 fodder reason = %q, want %q", got, ReasonCapacityExceeded)
	}
}

// TestDispatch_VehicleWearAndFeed verifies the worked example of spec-transport.md
// 17.3: a pair with a wagon ships a line to a consignee; the feed 3 * n * k goes to
// transit_use, the wagon wears 5 * k, and the line goes to a Transit account of the
// origin region under transit_out.
//
// Arithmetic: n = 1 (the load, less the vehicle and the 6 fodder, is (20, 21): one
// unit holds (1 200, 900)), k = ceil(40 / 30) = 2, so the feed is 3 * 1 * 2 = 6
// fodder and the wear is 5 * 2 = 10: the wagon of durability 400 falls to 390.
//
// Requirement: FR-TRN-013, FR-TRN-014, FR-TRN-009
func TestDispatch_VehicleWearAndFeed(t *testing.T) {
	t.Parallel()
	w, rec := newShipWorld(t, func(c *Config) { c.World.MinAgents = 1; c.Groups[0].Scale = 2 }, nil)
	wagon, fodder := commodityOf(t, w, "wagon"), commodityOf(t, w, "hay")
	give(t, w, shipHome, wagon, 1)
	give(t, w, shipHome, fodder, 9)
	id, reason := ask(w, shipHome, 0, ShipmentRequest{Mode: modeWagon, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 4}}})
	if reason != "" {
		t.Fatalf("dispatch() reason = %q, want none", reason)
	}
	s := w.transport.ships[id]
	if s.units != 1 || s.arrives != 2 || s.account < 0 || !slices.Equal(s.lines, []Line{{Commodity: grain, Quantity: 4}}) {
		t.Errorf("shipment = %+v, want 1 unit, arrives 2, an account and 4 grain", s)
	}
	if got := held(t, w, shipHome, int(fodder)); got != 3 {
		t.Errorf("fodder held = %d, want 9 - 6 = 3", got)
	}
	if got := flowOf(w, ledger.TransitUse, int(fodder)); got != 6 {
		t.Errorf("transit_use fodder = %d, want 6", got)
	}
	if got := flowOf(w, ledger.TransitOut, int(grain)); got != 4 {
		t.Errorf("transit_out grain = %d, want 4", got)
	}
	if got, err := w.ledger.WearUnits(int(wagon), false); err != nil || got != 10 {
		t.Errorf("wear units of the wagon = %d, %v, want 5 * 2 = 10", got, err)
	}
	v, _ := w.ledger.Account(s.account)
	if v.Kind != ledger.KindTransit || v.Region != int(regHome) {
		t.Errorf("transit account = %+v, want a Transit account of region %d", v, regHome)
	}
	if d := of[ShipmentDispatched](rec); len(d) != 1 || d[0].Units != 1 || d[0].Traveller != shipHome || d[0].Hired {
		t.Errorf("ShipmentDispatched = %+v, want one self-carried by agent %d", d, shipHome)
	}
}

// TestDispatch_HiredHaul verifies the worked example of spec-transport.md 17.4: a
// hired shipment redeems n * s haul units to haul_use, takes a Transit account of the
// origin region and leaves the shipper where it is (not away).
//
// Arithmetic: 100 grain is (300, 200); the unit is (1 200, 900), so n = 1; the edge is
// 40 and the span 10, so s = 4 and 4 haul units are redeemed; k = ceil(40 / 30) = 2.
//
// Requirement: FR-TRN-012, FR-TRN-020, FR-TRN-009
func TestDispatch_HiredHaul(t *testing.T) {
	t.Parallel()
	w, rec := newShipWorld(t, nil, nil)
	haul := commodityOf(t, w, "haul")
	give(t, w, shipHome, haul, 5)
	give(t, w, shipHome, grain, 90)
	id, reason := ask(w, shipHome, 0, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 100}}})
	if reason != "" {
		t.Fatalf("dispatch() reason = %q, want none", reason)
	}
	s := w.transport.ships[id]
	if s.units != 1 || s.haulUnits != 4 || s.arrives != 2 || !s.hired || s.traveller != 0 || s.consignee != shipFar {
		t.Errorf("shipment = %+v, want 1 unit, 4 haul units, arrives 2, hired, consignee %d", s, shipFar)
	}
	if got := held(t, w, shipHome, int(haul)); got != 1 {
		t.Errorf("haul held = %d, want 5 - 4 = 1", got)
	}
	if got := flowOf(w, ledger.HaulUse, int(haul)); got != 4 {
		t.Errorf("haul_use = %d, want 4", got)
	}
	if w.transport.isAway(shipHome) {
		t.Error("the shipper of a hired shipment is away")
	}
	if d := of[ShipmentDispatched](rec); len(d) != 1 || d[0].HaulUnits != 4 || !d[0].Hired || d[0].Region != regHome || d[0].To != regFar {
		t.Errorf("ShipmentDispatched = %+v, want one hired, 4 haul units, home to far", d)
	}
}

// TestDispatch_HiredHaulCountsALineOfTheHaulCommodity verifies the stricter
// reading of rows 17 and 20 (decision taken): a line of the haul commodity is not
// redeemable, so 4 haul units held with a line of 1 of them leave 3, short of the 4
// the carriage needs.
//
// Requirement: FR-TRN-012
func TestDispatch_HiredHaulCountsALineOfTheHaulCommodity(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, nil, nil)
	haul := commodityOf(t, w, "haul")
	give(t, w, shipHome, haul, 4)
	if _, got := ask(w, shipHome, 0, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: haul, Quantity: 1}}}); got != ReasonInsufficientHaul {
		t.Errorf("reason = %q, want %q", got, ReasonInsufficientHaul)
	}
}

// TestTransitAccount_Reuse verifies that a shipment takes the lowest open Transit
// account of its origin region that no shipment uses, opens one when there is none,
// and never closes one (spec-transport.md 7.1, 5.5.1 step 1).
//
// Requirement: FR-TRN-020, FR-TRN-023
func TestTransitAccount_Reuse(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, nil, nil)
	haul := commodityOf(t, w, "haul")
	give(t, w, shipHome, haul, 40)
	give(t, w, shipHome, grain, 3)
	line := []Line{{Commodity: grain, Quantity: 1}}
	var ids [3]int
	for i := range ids {
		if _, got := ask(w, shipHome, i, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: line}); got != "" {
			t.Fatalf("dispatch %d reason = %q, want none", i, got)
		}
		ids[i] = w.transport.ships[i].account
	}
	if !(ids[0] < ids[1] && ids[1] < ids[2]) {
		t.Fatalf("accounts = %v, want three distinct accounts, opened in order", ids)
	}
	// The first and the third come back: the next dispatch takes the lowest.
	w.transport.free[regHome].Put(ids[2])
	w.transport.free[regHome].Put(ids[0])
	give(t, w, shipHome, grain, 1)
	if _, got := ask(w, shipHome, 3, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: line}); got != "" {
		t.Fatalf("dispatch 3 reason = %q, want none", got)
	}
	if got := w.transport.ships[3].account; got != ids[0] {
		t.Errorf("fourth account = %d, want the lowest free %d", got, ids[0])
	}
	if w.transport.free[regHome].Len() != 1 {
		t.Errorf("free list holds %d, want 1", w.transport.free[regHome].Len())
	}
}

// TestDispatch_MovesNoMoney verifies that a dispatch moves no money: every balance
// and every money flow of the ledger are equal before and after (spec-transport.md
// 3.3, position 3).
//
// Requirement: FR-TRN-015
func TestDispatch_MovesNoMoney(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, nil, nil)
	haul := commodityOf(t, w, "haul")
	give(t, w, shipHome, haul, 5)
	moneyOf := func() []int64 {
		var out []int64
		for i := range w.ledger.Len() {
			if v, _ := w.ledger.Account(i); v.Kind != ledger.KindBoundary {
				out = append(out, v.Balance)
			}
		}
		return out
	}
	before := moneyOf()
	if _, got := ask(w, shipHome, 0, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 2}}}); got != "" {
		t.Fatalf("dispatch reason = %q, want none", got)
	}
	after := moneyOf()
	// The Transit account is new: its balance is the one extra 0.
	if !slices.Equal(after[:len(before)], before) || len(after) != len(before)+1 || after[len(before)] != 0 {
		t.Errorf("balances after = %v, want %v and a 0", after, before)
	}
	for _, f := range w.ledger.AppendFlows(nil, false) {
		if f.Commodity == ledger.Money && f.Reason != ledger.Initial {
			t.Errorf("money flow %+v after a dispatch, want none beyond the initial endowment", f)
		}
	}
}

// shipLoad is a script that ships on tick 0 what the table says for each agent.
func shipLoad(grain CommodityID, n int, hired bool) func(v *AgentView) []ShipmentRequest {
	return func(v *AgentView) []ShipmentRequest {
		if v.Agent != shipHome || v.Tick.Tick != 0 {
			return nil
		}
		mode := modeFoot
		if hired {
			mode = modeCoach
		}
		out := make([]ShipmentRequest, n)
		for i := range out {
			out[i] = ShipmentRequest{Mode: mode, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 1}}}
		}
		return out
	}
}

// TestPlan_MaxDispatchPerTick verifies that the ninth request of a plan is refused
// too_many_requests, that the first eight are dispatched (the ninth would not
// have been refused for another reason), that the events come in plan order and
// that the shipments come before the plan's orders (spec-transport.md 5.3.2, 13.1).
//
// Requirement: FR-TRN-016, FR-TRN-035
func TestPlan_MaxDispatchPerTick(t *testing.T) {
	t.Parallel()
	w, rec := newShipWorld(t, nil, shipLoad(grain, MaxDispatchPerTick+1, true))
	haul := commodityOf(t, w, "haul")
	give(t, w, shipHome, haul, 4*MaxDispatchPerTick)
	give(t, w, shipHome, grain, MaxDispatchPerTick+1)
	step(t, w, 1)
	var disp, rej int
	for _, e := range rec.events {
		switch e := e.(type) {
		case ShipmentDispatched:
			disp++
		case ShipmentRejected:
			rej++
			if e.Reason != ReasonTooManyRequests || e.Agent != shipHome || e.Region != regHome {
				t.Errorf("ShipmentRejected = %+v, want too_many_requests of agent %d in region %d", e, shipHome, regHome)
			}
		case OrderPlaced:
			if disp+rej != MaxDispatchPerTick+1 {
				t.Errorf("an OrderPlaced came after %d of %d shipment events, want them all first", disp+rej, MaxDispatchPerTick+1)
			}
		}
	}
	if disp != MaxDispatchPerTick || rej != 1 {
		t.Errorf("dispatched %d and rejected %d, want %d and 1", disp, rej, MaxDispatchPerTick)
	}
	for i, s := range w.transport.ships {
		if s.id != ShipmentID(i) {
			t.Errorf("ships[%d].id = %d, want %d", i, s.id, i)
		}
	}
}

// TestModule_OnWithNoRequests verifies that a world with the module on and no
// strategy that ships behaves as a world without it: the same events (apart from
// the construction event, whose names table lists the modes) and the same
// snapshot (apart from the list of modules, the boundary rows of the two sinks, the
// configuration hash and the names of the modes), over ticks.
//
// Requirement: FR-TRN-004
func TestModule_OnWithNoRequests(t *testing.T) {
	t.Parallel()
	run := func(on bool) ([]Event, Snapshot) {
		w, rec := newShipWorld(t, func(c *Config) {
			if !on {
				c.Modules.Transport = nil
			}
		}, nil)
		step(t, w, 5)
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		return rec.events[1:], s // the first event is WorldCreated
	}
	evOn, snapOn := run(true)
	evOff, snapOff := run(false)
	if !reflect.DeepEqual(evOn, evOff) {
		t.Errorf("events differ with the module on and no request: %d on, %d off", len(evOn), len(evOff))
	}
	snapOn.Modules, snapOff.Modules = nil, nil
	snapOn.Boundary, snapOff.Boundary = nil, nil                   // the two sinks add rows
	snapOn.ConfigHash, snapOff.ConfigHash = [32]byte{}, [32]byte{} // the module is in the configuration
	snapOn.Names.Modes = nil
	if !reflect.DeepEqual(snapOn, snapOff) {
		a, b := reflect.ValueOf(snapOn), reflect.ValueOf(snapOff)
		for i := range a.NumField() {
			if !reflect.DeepEqual(a.Field(i).Interface(), b.Field(i).Interface()) {
				t.Errorf("snapshot field %s differs with the module on and no request", a.Type().Field(i).Name)
			}
		}
	}
}

// TestDispatch_ModuleOffRefusesWithoutEvent verifies that with the module off a
// request is refused unknown_mode, logged at Debug, with no ShipmentRejected: the
// events of an absent module are not emitted.
//
// Requirement: FR-TRN-004, FR-TRN-035
func TestDispatch_ModuleOffRefusesWithoutEvent(t *testing.T) {
	t.Parallel()
	cap := newCapture()
	w, rec := newShipWorld(t, func(c *Config) { c.Modules.Transport = nil },
		func(v *AgentView) []ShipmentRequest {
			if v.Agent != shipHome {
				return nil
			}
			return []ShipmentRequest{{Mode: 0, To: regFar, ToSelf: true}}
		}, WithLogger(slog.New(cap)))
	step(t, w, 1)
	if got := of[ShipmentRejected](rec); len(got) != 0 {
		t.Errorf("ShipmentRejected = %v, want none with the module off", got)
	}
	var lines []logRec
	for _, l := range *cap.lines {
		if l.msg == "shipment refused" {
			lines = append(lines, l)
		}
	}
	if len(lines) != 1 || lines[0].strs[logkeys.Reason] != string(ReasonUnknownMode) {
		t.Errorf("shipment refused lines = %+v, want one with reason unknown_mode", lines)
	}
}

// TestDispatch_Deterministic verifies that two worlds with the same seed and
// configuration dispatch the same shipments to the same accounts and emit the same
// events (spec-transport.md FR-TRN-044).
//
// Requirement: FR-TRN-044
func TestDispatch_Deterministic(t *testing.T) {
	t.Parallel()
	run := func() []Event {
		w, rec := newShipWorld(t, nil, shipLoad(grain, 3, true))
		give(t, w, shipHome, commodityOf(t, w, "haul"), 12)
		give(t, w, shipHome, grain, 3)
		step(t, w, 3)
		return rec.events
	}
	e1, e2 := run(), run()
	if !reflect.DeepEqual(e1, e2) {
		t.Errorf("two runs differ: %d and %d events", len(e1), len(e2))
	}
	var sent int
	for _, e := range e1 {
		if _, ok := e.(ShipmentDispatched); ok {
			sent++
		}
	}
	if sent != 3 {
		t.Errorf("dispatched = %d, want 3", sent)
	}
}

// TestEvents_Transport verifies the fields and the order of ShipmentDispatched and
// ShipmentRejected for a scripted plan (spec-transport.md 13.1): in plan order, the
// refusal carrying the request as given, lines in commodity order, and none of them
// absent a module (see TestDispatch_ModuleOffRefusesWithoutEvent).
//
// Requirement: FR-TRN-035
func TestEvents_Transport(t *testing.T) {
	t.Parallel()
	var haul CommodityID
	w, rec := newShipWorld(t, nil, func(v *AgentView) []ShipmentRequest {
		if v.Agent != shipHome {
			return nil
		}
		return []ShipmentRequest{
			{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: haul, Quantity: 1}, {Commodity: grain, Quantity: 2}}},
			{Mode: 7, To: regIsle},
		}
	})
	haul = commodityOf(t, w, "haul")
	give(t, w, shipHome, haul, 6)
	step(t, w, 1)
	var seq []string
	for _, e := range rec.events {
		switch e := e.(type) {
		case ShipmentDispatched:
			seq = append(seq, "dispatched")
			want := ShipmentDispatched{Tick: 0, Shipment: 0, Mode: modeCoach, Region: regHome, To: regFar, Shipper: shipHome, Consignee: shipFar,
				Hired: true, Arrives: 2, Units: 1, HaulUnits: 4, Lines: []Amount{{Commodity: grain, Quantity: 2}, {Commodity: haul, Quantity: 1}}}
			if !reflect.DeepEqual(e, want) {
				t.Errorf("ShipmentDispatched = %+v, want %+v", e, want)
			}
		case ShipmentRejected:
			seq = append(seq, "rejected")
			if want := (ShipmentRejected{Tick: 0, Region: regHome, Agent: shipHome, Mode: 7, To: regIsle, Reason: ReasonUnknownMode}); e != want {
				t.Errorf("ShipmentRejected = %+v, want %+v", e, want)
			}
		}
	}
	if !slices.Equal(seq, []string{"dispatched", "rejected"}) {
		t.Errorf("shipment events = %v, want dispatched then rejected, in plan order", seq)
	}
}

// TestLogs_Transport verifies the Debug lines of the module (spec-transport.md
// 13.2): "shipment dispatched" with the shipment, mode, region, dest and agent,
// "shipment refused" with the agent, mode and reason, and no Info or Warn line in
// the tick.
//
// Requirement: FR-TRN-042
func TestLogs_Transport(t *testing.T) {
	t.Parallel()
	cap := newCapture()
	w, _ := newShipWorld(t, nil, func(v *AgentView) []ShipmentRequest {
		if v.Agent != shipHome {
			return nil
		}
		return []ShipmentRequest{{Mode: modeFoot, To: regFar, ToSelf: true}, {Mode: modeFoot, To: regFar, ToSelf: true}}
	}, WithLogger(slog.New(cap)))
	before := len(*cap.lines)
	step(t, w, 1)
	var got []logRec
	for _, l := range (*cap.lines)[before:] {
		if l.level >= slog.LevelInfo && strings.Contains(l.msg, "shipment") {
			t.Errorf("line %q at level %v in the tick, want Debug only", l.msg, l.level)
		}
		got = append(got, l)
	}
	var sent, refused *logRec
	for i := range got {
		switch got[i].msg {
		case "shipment dispatched":
			sent = &got[i]
		case "shipment refused":
			refused = &got[i]
		}
	}
	if sent == nil || sent.attrs[logkeys.Shipment] != 0 || sent.attrs[logkeys.Mode] != int64(modeFoot) || sent.attrs[logkeys.Region] != int64(regHome) || sent.attrs[logkeys.Dest] != int64(regFar) || sent.attrs[logkeys.Agent] != int64(shipHome) {
		t.Errorf("shipment dispatched line = %+v, want shipment 0, mode foot, region home, dest far, agent %d", sent, shipHome)
	}
	if refused == nil || refused.strs[logkeys.Reason] != string(ReasonTravellerAway) || refused.attrs[logkeys.Agent] != int64(shipHome) {
		t.Errorf("shipment refused line = %+v, want reason traveller_away of agent %d", refused, shipHome)
	}
}

// primeTransport gives the shipment state room for n shipments and n line buffers,
// so that the allocation of the first dispatches (growth, not steady state) is out
// of the measure.
func primeTransport(w *World, n int) {
	t := w.transport
	t.ships = make([]shipment, 0, n)
	for range n {
		t.take(make([]Line, 0, MaxShipmentLines))
	}
}

// TestDispatch_Allocs verifies that with no observer a refused request allocates
// nothing and a dispatch allocates nothing in the module once its state has room:
// the checks, the load sum, the travel time and the sorted lines use the scratch of
// the world (spec-transport.md 13.6). The ledger's own growth (a lot of the Transit
// account) is measured once and held in the baseline: raising it is a reviewed
// change.
//
// Requirement: NFR-TRN-001, NFR-TRN-004
func TestDispatch_Allocs(t *testing.T) {
	grain := Line{Commodity: grain, Quantity: 1}
	t.Run("refusal", func(t *testing.T) {
		w, _ := newShipWorld(t, func(c *Config) { c.Modules.Transport.Modes[0].HaulSpan = 10 }, nil)
		w.obs = observers{}
		req := ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{grain}}
		a := &w.agents[shipHome]
		if got := testing.AllocsPerRun(50, func() { w.dispatch(a, 0, &req, 0) }); got != 0 {
			t.Errorf("AllocsPerRun(refused request) = %v, want 0", got)
		}
	})
	t.Run("dispatch", func(t *testing.T) {
		w, _ := newShipWorld(t, nil, nil)
		w.obs = observers{}
		const runs = 40
		primeTransport(w, runs+2)
		give(t, w, shipHome, commodityOf(t, w, "haul"), 4*(runs+2))
		give(t, w, shipHome, grain.Commodity, runs+2)
		req := ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{grain}}
		a := &w.agents[shipHome]
		w.dispatch(a, 0, &req, 0) // opens the first Transit account
		const baseline = 1
		if got := testing.AllocsPerRun(runs, func() { w.dispatch(a, 0, &req, 0) }); got > baseline {
			t.Errorf("AllocsPerRun(dispatch) = %v, want at most %d (the ledger's lot growth)", got, baseline)
		}
	})
}

// TestEvents_NoObserverNoAlloc verifies that no event of the module is built when
// no observer was supplied: the allocations of a dispatch with an observer exceed
// those without it by at least the event and its copy of the lines.
//
// Requirement: NFR-TRN-004
func TestEvents_NoObserverNoAlloc(t *testing.T) {
	measure := func(observed bool) float64 {
		w, _ := newShipWorld(t, nil, nil)
		if !observed {
			w.obs = observers{}
		}
		const runs = 40
		primeTransport(w, runs+2)
		give(t, w, shipHome, commodityOf(t, w, "haul"), 4*(runs+2))
		give(t, w, shipHome, grain, runs+2)
		req := ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 1}}}
		a := &w.agents[shipHome]
		w.dispatch(a, 0, &req, 0)
		return testing.AllocsPerRun(runs, func() { w.dispatch(a, 0, &req, 0) })
	}
	with, without := measure(true), measure(false)
	if with < without+2 {
		t.Errorf("allocations with an observer = %v, without = %v, want at least 2 more (the event and the copy of its lines)", with, without)
	}
}

// TestRequest_HiredAfterTripKeepsProvision verifies that a hired request made after
// a self-carried trip of the same plan is refused unprovisioned when it sends the
// upkeep the traveller needs on the trip, and is not refused for it when the module
// has no trip in flight (row 22).
//
// Arithmetic: the household holds 10 grain and the trip passes row 22 with all of it.
// The hired line of 10 grain would leave none for the ticks the trip has left.
//
// Fix: review round 1 of plan 2-12. A later request of the plan took the stock that
// row 22 had checked for the trip, so the traveller left with less than its upkeep.
//
// Requirement: FR-TRN-019
func TestRequest_HiredAfterTripKeepsProvision(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, nil, nil)
	give(t, w, shipHome, commodityOf(t, w, "haul"), 4)
	if _, got := ask(w, shipHome, 0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}); got != "" {
		t.Fatalf("trip reason = %q, want none", got)
	}
	hired := ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 10}}}
	if _, got := ask(w, shipHome, 1, hired); got != ReasonUnprovisioned {
		t.Errorf("dispatch(%+v) after a trip reason = %q, want %q", hired, got, ReasonUnprovisioned)
	}
}
