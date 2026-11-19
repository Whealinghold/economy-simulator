package economysim

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// facGroundDecaying is the mutation of facConfig that gives the drain a decay of 100 a tick.
func facGroundDecaying(m *Infrastructure) {
	for i := range m.Classes {
		if m.Classes[i].ID == "drain" {
			m.Classes[i].Decay = 100
		}
	}
}

// facDrain adds a full drain to the estate of home and returns its ID. The drain has a
// Durability of 500 and a BoostBP of 3 000, so at full condition eff = 3 000 and the
// endowment of ore at home (capacity 1 000) has an effective capacity of
// 1 000 * 10 000 * 13 000 / 10^8 = 1 300.
func facDrain(t *testing.T, w *World) FacilityID {
	t.Helper()
	ids, err := w.AddFacilities([]NewFacility{{Class: clsDrain, Region: 0, Toward: NoRegion, Owner: 0}})
	if err != nil {
		t.Fatalf("AddFacilities error = %v, want nil", err)
	}
	return ids[0]
}

// facAdjustedBy reads the EndowmentAdjusted events of the log of one kind.
func facAdjustedBy(obs *evLog, kind AdjustKind) []EndowmentAdjusted {
	var out []EndowmentAdjusted
	for _, e := range of[EndowmentAdjusted](obs) {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

// TestGround_EffectiveCapacity verifies the capacity of an endowment with a ground
// facility (spec-infrastructure.md 5.3.2): floor(Capacity * scale * (10 000 + B) / 10^8)
// capped at MaxEndowment, the facility never writes the scale slot, a consumer's scale and
// the gain multiply, a configured capacity of 0 stays 0 whatever the gain, and the stock
// does not rise with the capacity.
//
// Requirement: FR-CAP-024
func TestGround_EffectiveCapacity(t *testing.T) {
	t.Parallel()
	w := facWorld(t, nil)
	r := w.res
	if got := r.effectiveCapacity(0); got != 1_000 {
		t.Fatalf("capacity before any facility = %d, want 1000", got)
	}
	facDrain(t, w)
	if got, stock := r.effectiveCapacity(0), r.stock[0]; got != 1_300 || stock != 1_000 {
		t.Errorf("capacity %d and stock %d with a full drain, want 1300 (1000 * 13 000 / 10 000) and the stock unmoved at 1000", got, stock)
	}
	if r.capScale[0] != wholeBP {
		t.Errorf("capacity scale slot = %d, want %d: a facility never writes the slot", r.capScale[0], wholeBP)
	}
	// A drought of 5 000 and the gain multiply: 1 000 * 5 000 * 13 000 / 10^8 = 650.
	w.setCapacityScales(0, 0, 5_000)
	if got := r.effectiveCapacity(0); got != 650 {
		t.Errorf("capacity with a drought scale of 5000 and the drain = %d, want 650", got)
	}
	if r.groundBP(1) != 0 || r.effectiveCapacity(1) != 1_000 {
		t.Errorf("yonder: gain %d and capacity %d, want 0 and 1000: the drain stands at home", r.groundBP(1), r.effectiveCapacity(1))
	}
	r.capacity[1] = 0
	w.infra.ground[1] = MaxTotalBoostBP
	if got := r.effectiveCapacity(1); got != 0 {
		t.Errorf("capacity of an endowment configured at 0 with a gain = %d, want 0", got)
	}
	r.capacity[1] = MaxEndowment
	if got := r.effectiveCapacity(1); got != MaxEndowment {
		t.Errorf("capacity at the bound with a gain = %d, want MaxEndowment %d", got, int64(MaxEndowment))
	}
}

// TestGround_Clamp verifies the clamp of a ground stock (spec-infrastructure.md 5.3.3): a
// fall of capacity at a call lowers the stock to it, counts the excess as lost in the
// identity of spec-resources.md 6.4 and emits EndowmentAdjusted of kind facility; a fall
// that leaves the stock inside the capacity and a rise emit nothing; and at P8.3b a decay
// and then the ruin of the drain clamp the stock in the same way, after the regeneration.
//
// Requirement: FR-CAP-025
func TestGround_Clamp(t *testing.T) {
	t.Parallel()
	t.Run("at a call", func(t *testing.T) {
		t.Parallel()
		obs := &evLog{}
		w := facWorld(t, nil, WithObserver(obs))
		id := facDrain(t, w)
		w.adjustEndowments(0, 0, 300) // the stock rises to the improved capacity of 1 300
		obs.events = nil
		// A fall of 250: condition 250 of 500, eff = 1 500, capacity = 1 000 * 11 500 / 10 000 = 1 150.
		if err := w.AdjustFacilities([]FacilityAdjustment{{ID: id, Delta: -250}}); err != nil {
			t.Fatalf("AdjustFacilities error = %v, want nil", err)
		}
		want := EndowmentAdjusted{Tick: w.Tick(), Kind: AdjustKindFacility, CapacityBP: wholeBP, RegenBP: wholeBP, StockBefore: 1_300, StockAfter: 1_150, Lost: 150}
		if got := facAdjustedBy(obs, AdjustKindFacility); len(got) != 1 || got[0] != want {
			t.Fatalf("EndowmentAdjusted(facility) = %+v, want [%+v]", got, want)
		}
		if w.res.stock[0] != 1_150 || w.res.lost[0] != 150 {
			t.Errorf("stock %d and lost %d, want 1150 and 150", w.res.stock[0], w.res.lost[0])
		}
		if err := w.Audit(); err != nil {
			t.Errorf("Audit() = %v, want nil: base 1000 + adjusted 300 - lost 150 = 1150", err)
		}
		obs.events = nil
		w.adjustEndowments(0, 0, -500) // the stock is 650, inside every capacity below
		obs.events = nil
		for _, delta := range []int64{-100, 200} { // a fall inside the stock, then a rise
			if err := w.AdjustFacilities([]FacilityAdjustment{{ID: id, Delta: delta}}); err != nil {
				t.Fatalf("AdjustFacilities error = %v, want nil", err)
			}
		}
		if got := facAdjustedBy(obs, AdjustKindFacility); len(got) != 0 {
			t.Errorf("EndowmentAdjusted(facility) = %+v, want none when no unit was lost", got)
		}
	})
	t.Run("at the facility step", func(t *testing.T) {
		t.Parallel()
		obs := &evLog{}
		w := facWorld(t, facGroundDecaying, WithObserver(obs))
		facDrain(t, w)
		w.adjustEndowments(0, 0, 300)
		obs.events = nil
		facStep(t, w) // decay 100: condition 400, eff = 2 400, capacity = 1 000 * 12 400 / 10 000 = 1 240
		got := facAdjustedBy(obs, AdjustKindFacility)
		if len(got) != 1 || got[0].StockAfter != 1_240 || got[0].Lost != got[0].StockBefore-1_240 || got[0].Tick != 0 {
			t.Fatalf("EndowmentAdjusted(facility) = %+v, want one at tick 0 with the stock lowered to 1240 and the excess lost", got)
		}
		for range 4 { // conditions 300, 200, 100 and 0: the drain is ruined in the fourth step
			facStep(t, w)
		}
		if w.infra.index(1) >= 0 || w.res.stock[0] > 1_000 || w.res.effectiveCapacity(0) != 1_000 {
			t.Errorf("after the ruin: drain index %d, stock %d, capacity %d, want it gone, the stock at most 1000 and the capacity back to 1000",
				w.infra.index(1), w.res.stock[0], w.res.effectiveCapacity(0))
		}
	})
}

// TestCheckpoint_FacilitiesGround verifies the case deferred from the records PR (Q-C): a
// stock above the facility-less capacity restores while the drain that lifts the capacity
// is in the checkpoint (the stock check of a restore is a cross-check that reads the table
// after tag 27), and a checkpoint whose records hold no drain for such a stock is
// ErrCheckpointCorrupt naming the endowment.
//
// Requirement: FR-CAP-041, FR-CAP-025
func TestCheckpoint_FacilitiesGround(t *testing.T) {
	t.Parallel()
	w := facWorld(t, nil)
	facDrain(t, w)
	w.adjustEndowments(0, 0, 300)
	cp := errVal(w.Checkpoint())
	restored, err := RestoreWorld(w.Config(), cp, WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld with a stock above the facility-less capacity = %v, want nil", err)
	}
	if got, c := restored.res.stock[0], restored.res.effectiveCapacity(0); got != 1_300 || c != 1_300 {
		t.Errorf("restored stock %d and capacity %d, want 1300 and 1300", got, c)
	}
	if again := errVal(restored.Checkpoint()); !bytes.Equal(again.b, cp.b) {
		t.Errorf("checkpoint of the restored world differs from the original's")
	}
	w.infra.remove(0) // the drain leaves the records without the clamp that a call makes
	w.infra.rebuild()
	if _, err := RestoreWorld(w.Config(), errVal(w.Checkpoint())); !errors.Is(err, ErrCheckpointCorrupt) {
		t.Errorf("RestoreWorld of a stock above the capacity with no drain = %v, want ErrCheckpointCorrupt", err)
	}
}

// routeWorld is the world of shipConfig with the edge between home and far 60 long and a
// class of road for the mode foot (Durability 100, BoostBP 5 000): on foot at speed 20 a
// journey of 60 takes ceil(60 * 10 000 / (20 * 10 000)) = 3 ticks, and 2 with the road,
// ceil(60 * 10 000 / (20 * 15 000)) = 2 (spec-infrastructure.md 18.2).
func routeWorld(t *testing.T, mut func(c *Config)) (*World, *evLog) {
	t.Helper()
	return newShipWorld(t, func(c *Config) {
		c.Edges = []Edge{{From: "a_home", To: "b_far", Distance: 60}, {From: "b_far", To: "a_home", Distance: 60}}
		c.Modules.Infrastructure = &Infrastructure{Classes: []FacilityClass{
			{ID: "road", Kind: "route", Durability: 100, BoostBP: 5_000, Modes: []string{"foot"}},
		}}
		if mut != nil {
			mut(c)
		}
	}, nil)
}

// facRoad adds a road of the class 0 to the estate of the region with the target.
func facRoad(t *testing.T, w *World, region, toward RegionID) {
	t.Helper()
	if _, err := w.AddFacilities([]NewFacility{{Class: 0, Region: region, Toward: toward, Owner: AgentID(region)}}); err != nil {
		t.Fatalf("AddFacilities error = %v, want nil", err)
	}
}

// ticksOf is Travel's time for the mode foot, failing the test when there is no route.
func ticksOf(t *testing.T, w *World, from, to RegionID) int64 {
	t.Helper()
	k, _, ok := w.Travel(from, to, modeFoot)
	if !ok {
		t.Fatalf("Travel(%d, %d, foot) has no route", from, to)
	}
	return k
}

// TestRoute_Layer verifies the transit factor of a pair with a route facility
// (spec-infrastructure.md 5.2.2): the gain lifts F, so a journey of 60 on foot takes 2
// ticks instead of 3; Slots returns the value the consumer set and never the layer; a slot
// of 0 stays closed; a gain adds and the factor is capped at MaxTransitFactorBP; and another
// mode is not speeded.
//
// Requirement: FR-CAP-021
func TestRoute_Layer(t *testing.T) {
	t.Parallel()
	w, _ := routeWorld(t, nil)
	if k := ticksOf(t, w, regHome, regFar); k != 3 {
		t.Fatalf("Travel before the road = %d ticks, want 3", k)
	}
	facRoad(t, w, regHome, regFar)
	if k := ticksOf(t, w, regHome, regFar); k != 2 {
		t.Errorf("Travel with the road = %d ticks, want 2 (F = 15 000)", k)
	}
	if k, _, ok := w.Travel(regHome, regFar, modeWagon); !ok || k != 2 {
		t.Errorf("Travel on the wagon = %d, %v, want 2 and true: speed 30 over 60 is 2 ticks and the road names foot only", k, ok)
	}
	if err := w.SetTransitFactors([]TransitFactor{{From: regHome, To: regFar, Mode: modeFoot, BP: 20_000}}); err != nil {
		t.Fatalf("SetTransitFactors error = %v, want nil", err)
	}
	// F = 20 000 * 15 000 / 10 000 = 30 000: ceil(60 * 10 000 / (20 * 30 000)) = 1.
	if k := ticksOf(t, w, regHome, regFar); k != 1 {
		t.Errorf("Travel with a slot of 20 000 and the road = %d, want 1", k)
	}
	if s := errVal(w.Slots()); len(s.TransitFactors) != 1 || s.TransitFactors[0].BP != 20_000 {
		t.Errorf("Slots().TransitFactors = %+v, want the consumer's 20 000 and not the layer's 30 000", s.TransitFactors)
	}
	if got := w.infra.layer(regHome, regFar, modeFoot, MaxTransitFactorBP); got != MaxTransitFactorBP {
		t.Errorf("layer at the slot bound = %d, want it capped at %d", got, int64(MaxTransitFactorBP))
	}
	if err := w.SetTransitFactors([]TransitFactor{{From: regHome, To: regFar, Mode: modeFoot, BP: 0}}); err != nil {
		t.Fatalf("SetTransitFactors error = %v, want nil", err)
	}
	if _, _, ok := w.Travel(regHome, regFar, modeFoot); ok {
		t.Errorf("Travel of a closed pair with a road has a route, want none: 0 stays 0")
	}
	facRoad(t, w, regHome, regFar) // two roads: the gains add to 10 000 and stay a sum
	if got := w.infra.route[(int(regHome)*w.infra.regions+int(regFar))*w.infra.modes+int(modeFoot)]; got != 10_000 {
		t.Errorf("gain of two roads = %d, want the sum 10000", got)
	}
}

// TestRoute_BothEdges verifies which edges a road speeds (5.2.1): a road with a target
// speeds both directed edges, a road with none every edge that leaves its region and not
// the one that arrives; and a shipment in flight keeps its arrival tick (5.2.3), while the
// next dispatch takes the gain.
//
// Requirement: FR-CAP-022
func TestRoute_BothEdges(t *testing.T) {
	t.Parallel()
	t.Run("a target", func(t *testing.T) {
		t.Parallel()
		w, _ := routeWorld(t, nil)
		facRoad(t, w, regHome, regFar)
		if a, b := ticksOf(t, w, regHome, regFar), ticksOf(t, w, regFar, regHome); a != 2 || b != 2 {
			t.Errorf("Travel there %d and back %d, want 2 and 2", a, b)
		}
	})
	t.Run("no target", func(t *testing.T) {
		t.Parallel()
		w, _ := routeWorld(t, nil)
		facRoad(t, w, regHome, NoRegion)
		if a, b := ticksOf(t, w, regHome, regFar), ticksOf(t, w, regFar, regHome); a != 2 || b != 3 {
			t.Errorf("Travel out of the region %d and into it %d, want 2 and 3: a departure facility speeds departures", a, b)
		}
	})
	t.Run("a shipment in flight", func(t *testing.T) {
		t.Parallel()
		w, _ := routeWorld(t, nil)
		grain := commodityOf(t, w, "grain")
		line := []Line{{Commodity: grain, Quantity: 1}}
		first, reason := ask(w, shipHome, 0, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: shipFar, Lines: line})
		if reason != "" {
			t.Fatalf("dispatch refused: %q", reason)
		}
		facRoad(t, w, regHome, regFar)
		second, reason := ask(w, shipHome2, 0, ShipmentRequest{Mode: modeFoot, To: regFar, Consignee: shipFar, Lines: line})
		if reason != "" {
			t.Fatalf("dispatch refused: %q", reason)
		}
		arrives := map[ShipmentID]int64{}
		for _, s := range w.transport.ships {
			arrives[s.id] = s.arrives - s.dispatched
		}
		if arrives[first] != 3 || arrives[second] != 2 {
			t.Errorf("journeys in ticks: first %d, second %d, want 3 (before the road, unchanged) and 2", arrives[first], arrives[second])
		}
	})
}

// TestRoute_NoNews verifies that a road speeds goods and not news (5.2.3): the shortest
// distance of the pair, which the lag of a rumour and a migrant's journey are functions
// of, and the first hop of the path are what they were without the road.
//
// Requirement: FR-CAP-023
func TestRoute_NoNews(t *testing.T) {
	t.Parallel()
	w, _ := routeWorld(t, nil)
	d0, _ := w.geo.distance(regHome, regFar)
	h0, _ := w.geo.next(regHome, regFar)
	facRoad(t, w, regHome, regFar)
	d1, _ := w.geo.distance(regHome, regFar)
	h1, _ := w.geo.next(regHome, regFar)
	if d0 != 60 || d1 != d0 || h1 != h0 {
		t.Errorf("distance %d then %d, next hop %d then %d, want them unchanged at 60", d0, d1, h0, h1)
	}
}

// TestRoute_PairRule verifies the pair rule of 5.2.5 on the three regions home, far and
// isle: a road on an edge that is not the shortest path of its pair speeds nothing, a pair
// with no direct edge gets no gain from the roads on the edges of its path, and a road on a
// direct edge that is the shortest does speed its pair.
//
// Requirement: FR-CAP-021
func TestRoute_PairRule(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		edges []Edge
		road  [2]RegionID
		pair  [2]RegionID
		want  int64
	}{
		// home-far is 100 long and home-isle-far is 30 + 30: the direct edge is not the
		// shortest path (D = 60), so the road on it speeds nothing: ceil(60 / 20) = 3.
		{"a direct edge that is not shortest", []Edge{
			{From: "a_home", To: "b_far", Distance: 100}, {From: "b_far", To: "a_home", Distance: 100},
			{From: "a_home", To: "c_isle", Distance: 30}, {From: "c_isle", To: "a_home", Distance: 30},
			{From: "c_isle", To: "b_far", Distance: 30}, {From: "b_far", To: "c_isle", Distance: 30}},
			[2]RegionID{regHome, regFar}, [2]RegionID{regHome, regFar}, 3},
		// The road on home-isle (30, the shortest) does not speed the pair home-far, which
		// has no direct edge of its own in this graph.
		{"a pair with no direct edge", []Edge{
			{From: "a_home", To: "c_isle", Distance: 30}, {From: "c_isle", To: "a_home", Distance: 30},
			{From: "c_isle", To: "b_far", Distance: 30}, {From: "b_far", To: "c_isle", Distance: 30}},
			[2]RegionID{regHome, regIsle}, [2]RegionID{regHome, regFar}, 3},
		// The same road does speed its own pair: ceil(30 * 10 000 / (20 * 15 000)) = 1.
		{"a direct edge that is shortest", []Edge{
			{From: "a_home", To: "c_isle", Distance: 30}, {From: "c_isle", To: "a_home", Distance: 30},
			{From: "c_isle", To: "b_far", Distance: 30}, {From: "b_far", To: "c_isle", Distance: 30}},
			[2]RegionID{regHome, regIsle}, [2]RegionID{regHome, regIsle}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _ := routeWorld(t, func(c *Config) { c.Edges = tt.edges })
			facRoad(t, w, tt.road[0], tt.road[1])
			if got := ticksOf(t, w, tt.pair[0], tt.pair[1]); got != tt.want {
				t.Errorf("Travel(%d, %d) with the road on %v = %d ticks, want %d", tt.pair[0], tt.pair[1], tt.road, got, tt.want)
			}
		})
	}
}

// TestInfrastructure_LayerAllocsPerRun verifies NFR-CAP-002 for the route layer: Travel with
// the layer on allocates nothing. Not parallel: AllocsPerRun panics in a parallel test.
//
// Requirement: NFR-CAP-002
func TestInfrastructure_LayerAllocsPerRun(t *testing.T) {
	w, _ := routeWorld(t, nil)
	facRoad(t, w, regHome, regFar)
	if got := testing.AllocsPerRun(50, func() { w.Travel(regHome, regFar, modeFoot) }); got != 0 {
		t.Errorf("allocations of Travel with a route facility = %v, want 0", got)
	}
}

// TestWire_GroundTableSizeMismatchIsAnInvariant verifies that wiring a ground table whose
// length differs from the endowments of the resources module stops at once with an
// invariant message, instead of letting clampGround or effectiveCapacity index past the
// end of one of the two tables.
//
// Guard: both tables are sized region * resources at construction, so a mismatch is a
// programmer error. Without the check it would surface as an index panic inside P8.3b or
// a capacity read, far from the cause.
func TestWire_GroundTableSizeMismatchIsAnInvariant(t *testing.T) {
	t.Parallel()
	w := facWorld(t, nil)
	n := w.infra
	n.ground = n.ground[:len(n.ground)-1]
	msg := invariantMessage(func() { n.wire(w) })
	if !strings.Contains(msg, "does not match the endowments") {
		t.Errorf("wire with a short ground table: message = %q, want the invariant naming the mismatch", msg)
	}
}
