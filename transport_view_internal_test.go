package economysim

import (
	"errors"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"
)

// This file tests the view side of the transport module (transport_view.go, plan
// 2-12 Phase 6) over the world of transport_request_internal_test.go.

// viewProbe is a Strategy that keeps the views it is given at Work and at Trade,
// by agent and tick, and asks for the shipments ship says. bid, when set, is the
// consignee's rule for how many grain to bid for.
type viewProbe struct {
	ship        func(v *AgentView) []ShipmentRequest
	bid         func(v *AgentView) Quantity
	work, trade map[[2]int64]AgentView
}

func newViewProbe() *viewProbe {
	return &viewProbe{work: map[[2]int64]AgentView{}, trade: map[[2]int64]AgentView{}}
}

func (s *viewProbe) Work(v *AgentView, _ *rand.Rand) JobRequest {
	s.work[[2]int64{int64(v.Agent), v.Tick.Tick}] = cloneView(v)
	return JobRequest{NoJob: true}
}

func (s *viewProbe) Trade(v *AgentView, _ *rand.Rand, p *TradePlan) {
	s.trade[[2]int64{int64(v.Agent), v.Tick.Tick}] = cloneView(v)
	if s.ship != nil {
		p.Shipments = append(p.Shipments, s.ship(v)...)
	}
	if s.bid != nil {
		if q := s.bid(v); q > 0 {
			p.Intents = append(p.Intents, Intent{Commodity: grain, Side: Bid, Quantity: q, Limit: 900})
		}
	}
}

func (*viewProbe) Revise(o *OutcomeView, _ *rand.Rand) Belief { return o.Belief }

var _ Strategy = (*viewProbe)(nil)

// cloneView copies the transport groups of a view, which the world reuses.
func cloneView(v *AgentView) AgentView {
	return AgentView{Agent: v.Agent, Tick: v.Tick, Inbound: slices.Clone(v.Inbound), Outbound: slices.Clone(v.Outbound), Routes: slices.Clone(v.Routes)}
}

// probeWorld builds the ship world with the probe as the role's strategy.
func probeWorld(t *testing.T, s *viewProbe, opts ...Option) (*World, *evLog) {
	t.Helper()
	rec := &evLog{}
	w := newMill(t, shipConfig(), append([]Option{WithObserver(rec), WithAudit(), WithStrategy("script", s)}, opts...)...)
	return w, rec
}

// TestView_InboundOutboundRoutes verifies spec-transport.md 9.1: the consignee sees
// the units on the way and the earliest arrival, the shipper sees them as outbound
// by destination, both fixed at P3.1 so that a dispatch of an agent earlier in the
// order of the same tick is not in the view of a later one, the views of phase 2
// have none, and the routes list every other region and mode in order with the time
// and, for a hired mode, the haul units per carrying unit.
//
// Arithmetic: the edge is 40; coach (speed 30) and foot, wagon (20, 30) give
// ceil(40 / 30) = 2, 2 and ceil(40 / 20) = 2 ticks; a coach unit costs
// ceil(40 / 10) = 4 haul units; the isle has no edge, so no route.
//
// Requirement: FR-TRN-033, FR-TRN-049
func TestView_InboundOutboundRoutes(t *testing.T) {
	t.Parallel()
	s := newViewProbe()
	s.ship = func(v *AgentView) []ShipmentRequest {
		if v.Agent == shipHome {
			return []ShipmentRequest{{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 1}}}}
		}
		return nil
	}
	w, _ := probeWorld(t, s)
	give(t, w, shipHome, commodityOf(t, w, "haul"), 8)
	step(t, w, 2)
	// Tick 0: the first dispatch. Tick 1: the second, by the same agent, earlier in
	// the order than the consignee, whose view must not see it.
	far := s.trade[[2]int64{int64(shipFar), 1}]
	if want := []InboundRow{{Commodity: grain, Quantity: 1, Earliest: 2}}; !reflect.DeepEqual(far.Inbound, want) {
		t.Errorf("consignee Inbound at tick 1 = %+v, want %+v (the dispatch of tick 1 is not in it)", far.Inbound, want)
	}
	home := s.trade[[2]int64{int64(shipHome), 1}]
	if want := []OutboundRow{{To: regFar, Commodity: grain, Quantity: 1, Earliest: 2}}; !reflect.DeepEqual(home.Outbound, want) {
		t.Errorf("shipper Outbound at tick 1 = %+v, want %+v", home.Outbound, want)
	}
	wantRoutes := []RouteRow{
		{To: regFar, Mode: modeCoach, Has: true, Ticks: 2, HaulPerUnit: 4},
		{To: regFar, Mode: modeFoot, Has: true, Ticks: 2},
		{To: regFar, Mode: modeWagon, Has: true, Ticks: 2},
		{To: regIsle, Mode: modeCoach}, {To: regIsle, Mode: modeFoot}, {To: regIsle, Mode: modeWagon},
	}
	if !reflect.DeepEqual(home.Routes, wantRoutes) {
		t.Errorf("Routes = %+v, want %+v", home.Routes, wantRoutes)
	}
	if v := s.work[[2]int64{int64(shipFar), 1}]; v.Inbound != nil || v.Outbound != nil || v.Routes != nil {
		t.Errorf("the Work view has transport rows %+v, want none before P3.1", v)
	}
	if v, ok := s.trade[[2]int64{int64(shipFar), 0}]; !ok || v.Inbound != nil {
		t.Errorf("tick 0 consignee view = %+v (seen %v), want no inbound before any dispatch", v, ok)
	}
}

// TestQueries_PureAndBounded verifies the four queries of spec-transport.md 9.2:
// Travel at the factor in force and its failures, InTransit of the shipper and of
// the consignee, Away, RouteLoss 0 with ok for a route, and that none changes
// the state of the world (the position hash) or draws.
//
// Requirement: FR-TRN-034
func TestQueries_PureAndBounded(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, nil, onTick(0, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 2}}}))
	step(t, w, 1)
	before := hashWorld(w)
	for _, tt := range []struct {
		name     string
		from, to RegionID
		mode     ModeID
		ticks    int64
		haul     int64
		ok       bool
	}{
		{"foot", regHome, regFar, modeFoot, 2, 0, true},
		{"coach", regHome, regFar, modeCoach, 2, 4, true},
		{"same region", regHome, regHome, modeFoot, 0, 0, false},
		{"no route", regHome, regIsle, modeFoot, 0, 0, false},
		{"unknown mode", regHome, regFar, 9, 0, 0, false},
		{"unknown region", regHome, 9, modeFoot, 0, 0, false},
	} {
		ticks, haul, ok := w.Travel(tt.from, tt.to, tt.mode)
		if ticks != tt.ticks || haul != tt.haul || ok != tt.ok {
			t.Errorf("Travel(%s) = (%d, %d, %v), want (%d, %d, %v)", tt.name, ticks, haul, ok, tt.ticks, tt.haul, tt.ok)
		}
	}
	want := []Amount{{Commodity: grain, Quantity: 2}}
	for _, id := range []AgentID{shipHome, shipFar} {
		if got := w.InTransit(id); !reflect.DeepEqual(got, want) {
			t.Errorf("InTransit(%d) = %+v, want %+v", id, got, want)
		}
	}
	if got := w.InTransit(shipHome2); len(got) != 0 {
		t.Errorf("InTransit(uninvolved) = %+v, want none", got)
	}
	if !w.Away(shipHome) || w.Away(shipFar) {
		t.Errorf("Away(home) = %v, Away(far) = %v, want true and false", w.Away(shipHome), w.Away(shipFar))
	}
	if bp, ok := w.RouteLoss(regHome, regFar, modeFoot); bp != 0 || !ok {
		t.Errorf("RouteLoss(route) = (%d, %v), want (0, true)", bp, ok)
	}
	if bp, ok := w.RouteLoss(regHome, regIsle, modeFoot); bp != 0 || ok {
		t.Errorf("RouteLoss(no route) = (%d, %v), want (0, false)", bp, ok)
	}
	if after := hashWorld(w); after != before {
		t.Error("a query changed the state of the world")
	}
	off := newMill(t, func() Config { c := shipConfig(); c.Modules.Transport = nil; return c }(), WithStrategy("script", newViewProbe()))
	if _, _, ok := off.Travel(regHome, regFar, 0); ok || off.Away(shipHome) || off.InTransit(shipHome) != nil {
		t.Error("a query of a world with the module off found a route, an away agent or a shipment")
	}
}

// TestTransitFactor_Slot verifies spec-transport.md 10.3 and 4: the default factor is
// 10 000; a factor of 0 blocks new dispatches and not those in flight; a factor of 5 000
// doubles the time; the bounds and the sentinels; one event per changed key and none
// when unchanged; setting 10 000 clears the entry.
//
// Arithmetic: foot over 40 at speed 20 takes ceil(40 * 10 000 / (20 * 5 000)) = 4 ticks.
//
// Requirement: FR-TRN-039, FR-TRN-005, FR-TRN-007
func TestTransitFactor_Slot(t *testing.T) {
	t.Parallel()
	w, rec := newShipWorld(t, nil, nil)
	if got := w.transport.factorOf(regHome, regFar, modeFoot); got != 10_000 {
		t.Fatalf("default factor = %d, want 10 000", got)
	}
	if _, got := ask(w, shipHome, 0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}); got != "" {
		t.Fatalf("first dispatch reason = %q, want none", got)
	}
	set := func(from, to RegionID, mode ModeID, bp int64) error {
		return w.setTransitFactor(from, to, mode, bp, CauseConsumer)
	}
	for _, tt := range []struct {
		name string
		call func() error
		want error
	}{
		{"same region", func() error { return set(regHome, regHome, modeFoot, 1) }, ErrSameRegion},
		{"unknown mode", func() error { return set(regHome, regFar, 9, 1) }, ErrUnknownMode},
		{"unknown region", func() error { return set(regHome, 9, modeFoot, 1) }, ErrUnknownRegion},
		{"negative", func() error { return set(regHome, regFar, modeFoot, -1) }, ErrOutOfBounds},
		{"over the bound", func() error { return set(regHome, regFar, modeFoot, MaxTransitFactorBP+1) }, ErrOutOfBounds},
	} {
		if err := tt.call(); !errors.Is(err, tt.want) {
			t.Errorf("%s: error = %v, want %v", tt.name, err, tt.want)
		}
	}
	if len(of[TransitFactorChanged](rec)) != 0 {
		t.Fatal("a refused call emitted an event")
	}
	if err := set(regHome, regFar, modeFoot, 0); err != nil {
		t.Fatalf("set 0 = %v, want nil", err)
	}
	if err := set(regHome, regFar, modeFoot, 0); err != nil {
		t.Fatalf("set 0 again = %v, want nil", err)
	}
	if _, got := ask(w, shipHome2, 0, ShipmentRequest{Mode: modeFoot, To: regFar, ToSelf: true}); got != ReasonBlocked {
		t.Errorf("dispatch at factor 0 reason = %q, want %q", got, ReasonBlocked)
	}
	if len(w.transport.ships) != 1 || w.transport.ships[0].arrives != 2 {
		t.Errorf("the shipment in flight changed: %+v", w.transport.ships)
	}
	if err := set(regHome, regFar, modeFoot, 5_000); err != nil {
		t.Fatalf("set 5 000 = %v, want nil", err)
	}
	if k, _, _ := w.Travel(regHome, regFar, modeFoot); k != 4 {
		t.Errorf("Travel at 5 000 = %d ticks, want 4", k)
	}
	if err := set(regHome, regFar, modeFoot, MaxTransitFactorBP); err != nil {
		t.Fatalf("set the maximum = %v, want nil", err)
	}
	if k, _, _ := w.Travel(regHome, regFar, modeFoot); k != 1 {
		t.Errorf("Travel at the maximum = %d ticks, want the minimum 1", k)
	}
	if err := set(regHome, regFar, modeFoot, 10_000); err != nil {
		t.Fatalf("set 10 000 = %v, want nil", err)
	}
	if len(w.transport.factors) != 0 {
		t.Errorf("slot = %+v, want empty after the default is set back", w.transport.factors)
	}
	got := of[TransitFactorChanged](rec)
	want := []TransitFactorChanged{
		{From: regHome, To: regFar, Mode: modeFoot, BeforeBP: 10_000, AfterBP: 0, Cause: CauseConsumer},
		{From: regHome, To: regFar, Mode: modeFoot, BeforeBP: 0, AfterBP: 5_000, Cause: CauseConsumer},
		{From: regHome, To: regFar, Mode: modeFoot, BeforeBP: 5_000, AfterBP: MaxTransitFactorBP, Cause: CauseConsumer},
		{From: regHome, To: regFar, Mode: modeFoot, BeforeBP: MaxTransitFactorBP, AfterBP: 10_000, Cause: CauseConsumer},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("TransitFactorChanged = %+v, want %+v", got, want)
	}
	off := newMill(t, func() Config { c := shipConfig(); c.Modules.Transport = nil; return c }(), WithStrategy("script", newViewProbe()))
	if err := off.setTransitFactor(0, 1, 0, 1, CauseConsumer); !errors.Is(err, ErrUnknownMode) {
		t.Errorf("set with the module off = %v, want %v", err, ErrUnknownMode)
	}
}

// TestView_StrategyAvoidsDoubleOrder verifies TRN-A2 at unit level: a consignee that
// counts Inbound bids only for what is not already on the way, and one that ignores
// it bids again.
//
// Arithmetic: it wants 3 grain more than it holds; 3 are inbound, so the counting
// consignee bids 3 - 3 = 0 and the other bids 3.
//
// Requirement: FR-TRN-033
func TestView_StrategyAvoidsDoubleOrder(t *testing.T) {
	t.Parallel()
	bids := func(count bool) int {
		s := newViewProbe()
		s.ship = func(v *AgentView) []ShipmentRequest {
			if v.Agent == shipHome && v.Tick.Tick == 0 {
				return []ShipmentRequest{{Mode: modeFoot, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 3}}}}
			}
			return nil
		}
		s.bid = func(v *AgentView) Quantity {
			if v.Agent != shipFar || v.Tick.Tick != 1 {
				return 0
			}
			var inbound Quantity
			for _, r := range v.Inbound {
				if count && r.Commodity == grain {
					inbound += r.Quantity
				}
			}
			return 3 - inbound
		}
		w, rec := probeWorld(t, s)
		step(t, w, 2)
		n := 0
		for _, o := range of[OrderPlaced](rec) {
			if o.Agent == shipFar && o.Side == Bid {
				n++
			}
		}
		return n
	}
	if with, without := bids(true), bids(false); with != 0 || without != 1 {
		t.Errorf("bids placed: counting Inbound %d, ignoring it %d, want 0 and 1", with, without)
	}
}

// TestStep_TransportAllocs verifies that a world with the transport module on and no
// shipment allocates no more per step than one with it off, once warm: the fixing of
// P3.1, the routes and the arrivals of an empty pipeline use only the module's scratch.
//
// Requirement: NFR-TRN-001
func TestStep_TransportAllocs(t *testing.T) {
	measure := func(on bool) float64 {
		c := shipConfig()
		if !on {
			c.Modules.Transport = nil
		}
		w := newMill(t, c, WithStrategy("script", &shipScript{}))
		step(t, w, 3)
		return testing.AllocsPerRun(20, func() {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() = %v", err)
			}
		})
	}
	if on, off := measure(true), measure(false); on > off {
		t.Errorf("allocations per step with the module on = %v, off = %v, want no more", on, off)
	}
}
