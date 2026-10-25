package economysim

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/steady"
)

// regionMill is the chain of millConfig with a second region, away, that has its
// own group of millers. IDs sort away (0), home (1); the estates are agents 0 and
// 1, the millers of away 2 and 3 and those of home 4 and 5. Commodities sort flour
// (0) at 3 000 and grain (1) at 1 000.
func regionMill(prices ...RegionPrice) Config {
	c := millConfig(1, 2)
	away := DefaultRegion()
	away.ID = "away"
	c.Regions = append(c.Regions, away)
	c.Groups = append(c.Groups, Group{Region: "away", Role: "miller", Households: 2})
	c.RegionPrices = prices
	return c
}

// TestReferencePrice_Override verifies that a RegionPrices row is the reference
// price in its region and nowhere else, and is the one price the construction
// reads: the seed price of the market, the reference of the view that bounds a
// belief, and the first price of an estate listing. The warm-start seed
// replaces the seed only (spec-regions.md 5.2.2).
//
// Arithmetic: flour is 3 000 and grain 1 000 by default. Away gets flour at
// 5 000 and grain at 1 050; the basket of away, 1 000, is within the tolerance
// of its upkeep (one grain, 1 050): |1 000 - 1 050| = 50 <= 100. The warm start
// seeds grain in away at 1 100, which is inside the band of 1 050 (105 to 10 500).
//
// Requirement: FR-REG-032, FR-REG-036, FR-REG-038
func TestReferencePrice_Override(t *testing.T) {
	t.Parallel()
	c := regionMill(RegionPrice{Region: "away", Commodity: "flour", Price: 5_000}, RegionPrice{Region: "away", Commodity: "grain", Price: 1_050})
	c.WarmStart = &WarmStart{Prices: []SeedPrice{{Region: "away", Commodity: "grain", Price: 1_100}}}
	w := newMill(t, c)
	const away, home RegionID = 0, 1
	want := []struct {
		r      RegionID
		k      CommodityID
		ref    Price
		market Price // the seed price of the market
	}{
		{away, flour, 5_000, 5_000},
		{away, grain, 1_050, 1_100}, // the warm start replaces the seed, not the reference
		{home, flour, 3_000, 3_000},
		{home, grain, 1_000, 1_000},
	}
	for _, tt := range want {
		if got := w.ReferencePrice(tt.r, tt.k); got != tt.ref {
			t.Errorf("ReferencePrice(%d, %d) = %d, want %d", tt.r, tt.k, got, tt.ref)
		}
		if got := w.markets.reports[w.markets.index(tt.r, tt.k)].Price; got != tt.market {
			t.Errorf("seed price of market (%d, %d) = %d, want %d", tt.r, tt.k, got, tt.market)
		}
	}
	w.buildRotation()
	w.refreshViewCtx()
	for _, id := range []AgentID{2, 4} {
		var v AgentView
		if err := w.fillAgentView(&v, &w.agents[id]); err != nil {
			t.Fatalf("fillAgentView(%d) error = %v, want nil", id, err)
		}
		if len(v.Commodities) == 0 {
			t.Fatalf("the view of agent %d has no commodity, so the test would prove nothing", id)
		}
		for _, cv := range v.Commodities {
			if ref := w.ReferencePrice(v.Region, cv.Commodity); cv.Ref != ref {
				t.Errorf("view of agent %d in region %d: Ref of commodity %d = %d, want the effective reference price %d", id, v.Region, cv.Commodity, cv.Ref, ref)
			}
		}
	}
}

// TestReferencePrice_EstateFirstPrice verifies that the estate of a region lists
// a lot at the region's reference price (spec-goods.md 6.3), for a good that does
// not wear and for one that does (scaled by the durability left), and that
// another region's estate is unchanged.
//
// Arithmetic: every commodity is 100; the override gives region b the staple at 160
// and the tool (durability 10) at 200. The staple lot is listed at 160 in b and
// 100 in a; the tool lot with durability 3 left at 200 * 3 / 10 = 60 in b and
// 100 * 3 / 10 = 30 in a.
//
// Requirement: FR-REG-032
func TestReferencePrice_EstateFirstPrice(t *testing.T) {
	t.Parallel()
	f := newListFixture(t, 2)
	f.cfg.RegionPrices = []RegionPrice{
		{Region: "b", Commodity: f.cfg.Commodities[lstStaple].ID, Price: 160},
		{Region: "b", Commodity: f.cfg.Commodities[lstTool].ID, Price: 200},
	}
	for r := range RegionID(2) {
		f.give(t, r, lstStaple, 0, 5)
		f.give(t, r, lstTool, 3, 2)
	}
	prices := func(r RegionID) map[int]Price {
		orders, err := f.e.asks(f.l, f.cfg, r, nil)
		if err != nil {
			t.Fatalf("asks(region %d) error = %v, want nil", r, err)
		}
		out := map[int]Price{}
		for _, o := range orders {
			out[int(o.Commodity)] = o.Price
		}
		return out
	}
	a, b := prices(0), prices(1)
	if a[lstStaple] != 100 || a[lstTool] != 30 || b[lstStaple] != 160 || b[lstTool] != 60 {
		t.Errorf("first prices a = %v, b = %v, want staple 100 and tool 30 in a, staple 160 and tool 60 in b", a, b)
	}
}

// TestReferencePrice_NoOverrideIsCommodityPrice verifies that with no override
// every effective price is the commodity's, an ID outside its range gives 0, and
// the table is the one World.ReferencePrice reads.
//
// Requirement: FR-REG-036, FR-REG-032
func TestReferencePrice_NoOverrideIsCommodityPrice(t *testing.T) {
	t.Parallel()
	w := newMill(t, regionMill())
	for r := range w.cfg.Regions {
		for k, cm := range w.cfg.Commodities {
			if got := w.ReferencePrice(RegionID(r), CommodityID(k)); got != cm.ReferencePrice {
				t.Errorf("ReferencePrice(%d, %d) = %d, want the commodity's %d", r, k, got, cm.ReferencePrice)
			}
		}
	}
	for _, bad := range [][2]int{{-1, 0}, {2, 0}, {0, -1}, {0, 2}} {
		if got := w.ReferencePrice(RegionID(bad[0]), CommodityID(bad[1])); got != 0 {
			t.Errorf("ReferencePrice(%d, %d) = %d, want 0 outside the range", bad[0], bad[1], got)
		}
	}
}

// TestGraph_ClosedEdgeIsAbsentButViewable verifies the graph built with a status
// table: a closed edge is absent from Out, In, Distance, NextHop, Path, Within and
// the components and still answers Edge and EdgeAt with Closed set and no
// effective length; a factor changes the length of an open edge by the rule of
// spec-regions.md 7.2.1 and leaves its EdgeID.
//
// Arithmetic: a>b 30, b>c 30, a>c 90. With a>b at 15 000 its length is
// ceil(30 * 15 000 / 10 000) = 45 and a>c is the shortest route to c, 75 against 90
// (45 + 30), with b>c closed leaving only a>c at 90.
//
// Requirement: FR-REG-049, FR-REG-014
func TestGraph_ClosedEdgeIsAbsentButViewable(t *testing.T) {
	t.Parallel()
	c := Config{
		Regions: []Region{{ID: "a"}, {ID: "b"}, {ID: "c"}},
		Edges:   []Edge{{From: "a", To: "b", Distance: 30}, {From: "a", To: "c", Distance: 90}, {From: "b", To: "c", Distance: 30}},
	}
	c = c.canonical()
	open := []edgeState{{factorBP: 10_000}, {factorBP: 10_000}, {factorBP: 10_000}}
	g, err := newGraph(&c, open)
	if err != nil {
		t.Fatalf("newGraph(open) error = %v, want nil", err)
	}
	if d, ok := g.Distance(0, 2); !ok || d != 60 {
		t.Errorf("Distance(a, c) all open = %d, %v, want 60, true (a>b>c)", d, ok)
	}

	status := []edgeState{{factorBP: 15_000}, {factorBP: 10_000}, {closed: true, factorBP: 10_000}}
	g, err = newGraph(&c, status)
	if err != nil {
		t.Fatalf("newGraph(status) error = %v, want nil", err)
	}
	if d, ok := g.Distance(0, 2); !ok || d != 90 {
		t.Errorf("Distance(a, c) with b>c closed = %d, %v, want 90, true (the direct edge)", d, ok)
	}
	if got := g.Out(1); len(got) != 0 {
		t.Errorf("Out(b) = %v, want none: its one edge is closed", got)
	}
	if got, want := g.In(2), []Hop{{Region: 0, Distance: 90}}; !slices.Equal(got, want) {
		t.Errorf("In(c) = %v, want %v: b>c is closed", got, want)
	}
	if got, want := g.Out(0), []Hop{{Region: 1, Distance: 45}, {Region: 2, Distance: 90}}; !slices.Equal(got, want) {
		t.Errorf("Out(a) = %v, want %v (30 at 15 000 is 45)", got, want)
	}
	if got, ok := g.Path(1, 2); ok || got != nil {
		t.Errorf("Path(b, c) = %v, %v, want none: b has no open edge out", got, ok)
	}
	if next, ok := g.NextHop(1, 2); ok {
		t.Errorf("NextHop(b, c) = %d, %v, want false", next, ok)
	}
	if got, want := g.Within(0, 100), []Hop{{Region: 1, Distance: 45}, {Region: 2, Distance: 90}}; !slices.Equal(got, want) {
		t.Errorf("Within(a, 100) = %v, want %v", got, want)
	}
	if got, want := g.Components(), 3; got != want {
		t.Errorf("Components() = %d, want %d: no cycle among a, b, c", got, want)
	}
	e, ok := g.Edge(1, 2)
	if want := (EdgeView{ID: 2, From: 1, To: 2, Base: 30, Distance: 0, Closed: true, FactorBP: 10_000}); !ok || e != want {
		t.Errorf("Edge(b, c) = %+v, %v, want %+v, true", e, ok, want)
	}
	if at, ok := g.EdgeAt(0); !ok || at.Distance != 45 || at.Base != 30 || at.FactorBP != 15_000 || at.ID != 0 {
		t.Errorf("EdgeAt(0) = %+v, %v, want base 30, distance 45, factor 15 000", at, ok)
	}
}

// TestGraph_ExtraEdgesQueries verifies the queries against a graph with a gap:
// an unknown ID, an edge that is absent and an ID outside the range give the
// zero result and false, and never panic.
//
// Requirement: FR-REG-025, FR-REG-004
func TestGraph_ExtraEdgesQueries(t *testing.T) {
	t.Parallel()
	c := Config{
		Regions: []Region{{ID: "a"}, {ID: "b"}},
		Edges:   []Edge{{From: "a", To: "b", Distance: 5}},
	}
	cc := c.canonical()
	g, err := newGraph(&cc, nil)
	if err != nil {
		t.Fatalf("newGraph() error = %v, want nil", err)
	}
	for _, bad := range []RegionID{-1, 2, 1 << 40, NoRegion} {
		if g.Name(bad) != "" || g.Out(bad) != nil || g.In(bad) != nil || len(g.Within(bad, 10)) != 0 {
			t.Errorf("a query with region %d gave a non-zero result", bad)
		}
		if _, ok := g.Distance(bad, 0); ok {
			t.Errorf("Distance(%d, 0) ok = true, want false", bad)
		}
		if _, ok := g.Distance(0, bad); ok {
			t.Errorf("Distance(0, %d) ok = true, want false", bad)
		}
		if _, ok := g.NextHop(bad, 1); ok {
			t.Errorf("NextHop(%d, 1) ok = true, want false", bad)
		}
		if _, ok := g.Path(0, bad); ok {
			t.Errorf("Path(0, %d) ok = true, want false", bad)
		}
		if _, ok := g.Component(bad); ok {
			t.Errorf("Component(%d) ok = true, want false", bad)
		}
		if _, ok := g.Edge(bad, 1); ok {
			t.Errorf("Edge(%d, 1) ok = true, want false", bad)
		}
		if _, ok := g.Edge(0, bad); ok {
			t.Errorf("Edge(0, %d) ok = true, want false", bad)
		}
	}
	for _, bad := range []EdgeID{-1, 1, 1 << 40} {
		if _, ok := g.EdgeAt(bad); ok {
			t.Errorf("EdgeAt(%d) ok = true, want false", bad)
		}
	}
	if _, ok := g.Edge(1, 0); ok {
		t.Errorf("Edge(b, a) ok = true, want false: there is no edge from b to a")
	}
	if _, ok := g.Lookup("zz"); ok {
		t.Errorf("Lookup(zz) ok = true, want false")
	}
}

// threeRegionMill is regionMill with a third region, zone, that has no group, so
// the estates must exist for a region without households. IDs sort away (0), home
// (1), zone (2).
func threeRegionMill() Config {
	c := regionMill()
	zone := DefaultRegion()
	zone.ID = "zone"
	c.Regions = append(c.Regions, zone)
	return c
}

// TestEstates_PseudoAgentIDsAndAccountOrder verifies the estates of the regions
// (spec-regions.md 3.6.1): one estate account of kind Estate per region, in the
// region it serves, created in ascending RegionID before any household account;
// the pseudo-agent of region r has the agent ID r and is the zero agent (it is
// no household); and a household's ID is at least the region count.
//
// Requirement: FR-REG-005
func TestEstates_PseudoAgentIDsAndAccountOrder(t *testing.T) {
	t.Parallel()
	w := newMill(t, threeRegionMill())
	n := len(w.cfg.Regions)
	lastEstate := -1
	for r := range n {
		acct := w.estates.account(RegionID(r))
		v, err := w.ledger.Account(acct)
		if err != nil || v.Kind != ledger.KindEstate || v.Region != r {
			t.Errorf("estate account of region %d = %+v, %v, want kind Estate in region %d", r, v, err, r)
		}
		if acct <= lastEstate {
			t.Errorf("estate account of region %d is %d, not above the previous %d", r, acct, lastEstate)
		}
		lastEstate = acct
		if id := w.estates.agentID(RegionID(r)); id != AgentID(r) {
			t.Errorf("estate pseudo-agent of region %d has the ID %d, want %d", r, id, r)
		}
		if a := w.agents[r]; a.weight != 0 || a.account != 0 || a.beliefs != nil {
			t.Errorf("agents[%d] = %+v, want the zero agent: an estate is no household", r, w.agents[r])
		}
	}
	for id := n; id < len(w.agents); id++ {
		if w.agents[id].account <= lastEstate {
			t.Errorf("household agent %d has the account %d, not above the last estate account %d", id, w.agents[id].account, lastEstate)
		}
	}
}

// TestEstates_ExistWhetherOrNotModuleOn verifies that a region with no group has
// its estate all the same (spec-regions.md 3.6.2): the region zone of
// threeRegionMill has no household, and its estate account is open and empty,
// and no estates module is on (no module is built).
//
// Requirement: FR-REG-006, FR-REG-007
func TestEstates_ExistWhetherOrNotModuleOn(t *testing.T) {
	t.Parallel()
	w := newMill(t, threeRegionMill())
	const zone RegionID = 2
	v, err := w.ledger.Account(w.estates.account(zone))
	if err != nil || v.Kind != ledger.KindEstate || v.Closed || v.Region != int(zone) || v.Balance != 0 {
		t.Errorf("estate of the region with no group = %+v, %v, want an open, empty Estate account in region %d", v, err, zone)
	}
	if w.live.inRole[int(zone)*len(w.cfg.Roles)] != 0 {
		t.Errorf("the region with no group has %d agents, want 0", w.live.inRole[int(zone)*len(w.cfg.Roles)])
	}
}

// TestRegions_SetIsFixed verifies that the regions of a world are those of its
// configuration and that nothing changes them: the graph, the snapshot and the
// configuration give the same region names before and after steps, and the
// configuration given by the caller can be changed afterwards without effect.
//
// Requirement: FR-REG-008
func TestRegions_SetIsFixed(t *testing.T) {
	t.Parallel()
	c := threeRegionMill()
	w := newMill(t, c)
	names := func() []string {
		g := w.Graph()
		var out []string
		for r := range RegionID(g.Len()) {
			out = append(out, g.Name(r))
		}
		return out
	}
	want := []string{"away", "home", "zone"}
	if got := names(); !slices.Equal(got, want) {
		t.Fatalf("Graph names = %q, want %q", got, want)
	}
	c.Regions[0].ID = "changed"
	c.Regions = append(c.Regions, Region{ID: "added"})
	step(t, w, 3)
	if got := names(); !slices.Equal(got, want) {
		t.Errorf("Graph names after steps and a change of the caller's configuration = %q, want %q", got, want)
	}
	if got := w.Config().Regions; len(got) != 3 || got[0].ID != "away" {
		t.Errorf("Config().Regions = %+v, want the three regions as built", got)
	}
}

// TestRegions_IterationOrder verifies that what is ordered by region is ordered
// by ascending RegionID, whatever order the regions were written in: the markets
// of the snapshot, and the AgentSpawned events of construction, which go region by
// region.
//
// Requirement: FR-REG-003, FR-REG-002
func TestRegions_IterationOrder(t *testing.T) {
	t.Parallel()
	forward, reversed := threeRegionMill(), threeRegionMill()
	slices.Reverse(reversed.Regions)
	slices.Reverse(reversed.Groups)
	rec := &evLog{}
	w := newMill(t, forward, WithObserver(rec))
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	last := RegionID(-1)
	for _, m := range snap.Markets {
		if m.State.Region < last {
			t.Errorf("snapshot market of region %d after region %d, want ascending RegionID", m.State.Region, last)
		}
		last = m.State.Region
	}
	lastSpawn := RegionID(-1)
	spawns := 0
	for _, e := range rec.events {
		if s, ok := e.(AgentSpawned); ok {
			spawns++
			if s.Region < lastSpawn {
				t.Errorf("AgentSpawned of region %d after region %d, want ascending RegionID", s.Region, lastSpawn)
			}
			lastSpawn = s.Region
		}
	}
	if spawns == 0 {
		t.Fatal("no AgentSpawned event, so the test would prove nothing")
	}
	other := newMill(t, reversed)
	if w.hash != other.hash {
		t.Errorf("hash with the regions and groups reversed = %x, want %x", other.hash, w.hash)
	}
}

// TestRegions_AgentLists verifies that the agents of a region, derived from the
// region of each account, are in ascending agent ID, that they partition the
// households, and that a region with no group has none (spec-regions.md 3.1.1).
//
// Requirement: FR-REG-045
func TestRegions_AgentLists(t *testing.T) {
	t.Parallel()
	w := newMill(t, threeRegionMill())
	byRegion := map[RegionID][]AgentID{}
	for id := len(w.cfg.Regions); id < len(w.agents); id++ {
		r := w.regionOf(&w.agents[id])
		byRegion[r] = append(byRegion[r], AgentID(id))
	}
	want := map[RegionID][]AgentID{0: {3, 4}, 1: {5, 6}}
	for r := range RegionID(3) {
		if got := byRegion[r]; !slices.Equal(got, want[r]) {
			t.Errorf("agents of region %d = %v, want %v", r, got, want[r])
		}
	}
}

// TestRegions_ReadersOnAFailedWorld verifies that the two region readers are
// derived tables and still answer on a world that a step failed: ReferencePrice
// gives the effective price and Graph builds the graph (spec-world.md 12.4,
// FR-WLD-110).
//
// Requirement: FR-WLD-110, FR-REG-028, FR-REG-036
func TestRegions_ReadersOnAFailedWorld(t *testing.T) {
	t.Parallel()
	c := regionMill(RegionPrice{Region: "away", Commodity: "flour", Price: 5_000})
	c.Edges = []Edge{{From: "away", To: "home", Distance: 9}}
	w := newMill(t, c)
	w.failed = errors.New("synthetic failure")
	if got := w.ReferencePrice(0, flour); got != 5_000 {
		t.Errorf("ReferencePrice on a failed world = %d, want 5000", got)
	}
	if d, ok := w.Graph().Distance(0, 1); !ok || d != 9 {
		t.Errorf("Graph().Distance(away, home) on a failed world = %d, %v, want 9, true", d, ok)
	}
}

// graphPointers returns the addresses of the backing arrays of a Graph, to tell
// whether two share storage.
func graphPointers(g Graph) []uintptr {
	ptrs := []uintptr{
		reflect.ValueOf(g.names).Pointer(), reflect.ValueOf(g.edges).Pointer(), reflect.ValueOf(g.first).Pointer(),
		reflect.ValueOf(g.in).Pointer(), reflect.ValueOf(g.inFirst).Pointer(),
	}
	tv := reflect.ValueOf(g.t)
	for i := range tv.NumField() {
		if f := tv.Field(i); f.Kind() == reflect.Slice {
			ptrs = append(ptrs, f.Pointer())
		}
	}
	return ptrs
}

// TestGraph_IndependentOfWorld verifies that a Graph is plain data that shares no
// storage with the world or with another Graph: one taken before the world's graph
// changes answers the same after, a list it returned can be changed without
// effect, and one taken after reflects the change. The change is made here by
// replacing the world's statuses and tables, as the status call does.
//
// Arithmetic: a>b 30 and b>c 30 and a>c 90; with a>b closed the shortest a to c is
// the direct 90 instead of 30 + 30 = 60.
//
// Requirement: FR-REG-026, FR-REG-028, FR-REG-030
func TestGraph_IndependentOfWorld(t *testing.T) {
	t.Parallel()
	c := regionMill()
	zone := DefaultRegion()
	zone.ID = "zone"
	c.Regions = append(c.Regions, zone)
	c.Edges = []Edge{{From: "away", To: "home", Distance: 30}, {From: "away", To: "zone", Distance: 90}, {From: "home", To: "zone", Distance: 30}}
	w := newMill(t, c)
	before := w.Graph()
	if d, ok := before.Distance(0, 2); !ok || d != 60 {
		t.Fatalf("Distance(away, zone) = %d, %v, want 60, true (through home)", d, ok)
	}
	mine, worlds, others := graphPointers(before), graphPointers(w.geo.g), graphPointers(w.Graph())
	for i, p := range mine {
		if p != 0 && (p == worlds[i] || p == others[i]) {
			t.Errorf("Graph storage %d is shared with the world or with another Graph", i)
		}
	}
	got := before.Out(0)
	got[0].Distance = 1
	if before.Out(0)[0].Distance != 30 {
		t.Errorf("changing a list returned by Out changed the Graph")
	}

	w.geo.status = []edgeState{{closed: true, factorBP: 10_000}, {factorBP: 10_000}, {factorBP: 10_000}}
	g, err := newGraph(&w.cfg, w.geo.status)
	if err != nil {
		t.Fatalf("newGraph() error = %v, want nil", err)
	}
	w.geo.g = g
	if d, ok := before.Distance(0, 2); !ok || d != 60 {
		t.Errorf("a Graph taken before the change: Distance(away, zone) = %d, %v, want 60, true", d, ok)
	}
	after := w.Graph()
	if d, ok := after.Distance(0, 2); !ok || d != 90 {
		t.Errorf("a Graph taken after the change: Distance(away, zone) = %d, %v, want 90, true", d, ok)
	}
	if d, ok := w.geo.distance(0, 2); !ok || d != 90 {
		t.Errorf("the world's own distance(away, zone) = %d, %v, want 90, true", d, ok)
	}
}

// TestRegionState_Reads verifies the world's own reads of the graph that the later
// modules call (spec-regions.md 4.4.1): distance, next, edge, out and components
// answer as the Graph value does, and out appends to its argument.
//
// Requirement: FR-REG-016, FR-REG-018, FR-REG-023
func TestRegionState_Reads(t *testing.T) {
	t.Parallel()
	c := regionMill()
	c.Edges = []Edge{{From: "away", To: "home", Distance: 30}, {From: "home", To: "away", Distance: 40}}
	w := newMill(t, c)
	if d, ok := w.geo.distance(0, 1); !ok || d != 30 {
		t.Errorf("distance(away, home) = %d, %v, want 30, true", d, ok)
	}
	if d, ok := w.geo.distance(1, 0); !ok || d != 40 {
		t.Errorf("distance(home, away) = %d, %v, want 40, true", d, ok)
	}
	if v, ok := w.geo.next(0, 1); !ok || v != 1 {
		t.Errorf("next(away, home) = %d, %v, want 1, true", v, ok)
	}
	if e, ok := w.geo.edge(1, 0); !ok || e.Base != 40 {
		t.Errorf("edge(home, away) = %+v, %v, want base 40", e, ok)
	}
	got := w.geo.out([]Hop{{Region: 9, Distance: 9}}, 0)
	if want := []Hop{{Region: 9, Distance: 9}, {Region: 1, Distance: 30}}; !slices.Equal(got, want) {
		t.Errorf("out(dst, away) = %v, want %v (appended)", got, want)
	}
	if got := w.geo.out(nil, 7); got != nil {
		t.Errorf("out(nil, 7) = %v, want nil for a region that is not one", got)
	}
	if w.geo.components() != 1 {
		t.Errorf("components() = %d, want 1: away and home reach each other", w.geo.components())
	}
}

// dearFlour is regionMill with flour at 50 000 in away against 3 000 elsewhere, a
// warm-start seed of 40 000 for it there, and 10 units of flour in the start stock
// so that the millers ask for flour in the first tick. 40 000 is inside the band
// of the override (5 000 to 500 000) and above the ceiling of the band of the
// commodity's own price (30 000), so a belief of 40 000 can be held only when the
// band is taken from the effective reference price.
func dearFlour() Config {
	c := regionMill(RegionPrice{Region: "away", Commodity: "flour", Price: 50_000})
	c.WarmStart = &WarmStart{Prices: []SeedPrice{{Region: "away", Commodity: "flour", Price: 40_000}}}
	c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 10}, {Commodity: "flour", Quantity: 10}}
	return c
}

// flourBeliefs returns the centre of the flour belief of each agent in ids.
func flourBeliefs(t *testing.T, w *World, ids ...AgentID) []int64 {
	t.Helper()
	var out []int64
	for _, id := range ids {
		i, ok := slices.BinarySearch(w.roles.row(w.agents[id].role).tradable, CommodityID(flour))
		if !ok {
			t.Fatalf("flour is not in the tradable set of agent %d", id)
		}
		out = append(out, w.agents[id].beliefs[i].Centre)
	}
	return out
}

// TestReferencePrice_BandOfTheEffectivePrice verifies that the reference band that
// bounds a belief is the band of the effective price in the agent's region
// (spec-regions.md 5.2.2, FR-REG-038): at construction a belief seeded at the
// warm-start price 40 000 is kept, not clamped to the 30 000 ceiling of the
// commodity's own price; the snapshot counts it at neither bound; and after
// ticks, when the order outcomes revise it inside its bounds, it is still above
// that ceiling. The millers of home keep the commodity's price.
//
// Arithmetic: RefBandHigh is 100 000 basis points, ten times the reference price:
// 3 000 gives 30 000, 50 000 gives 500 000. The flour of the millers of home is
// seeded at the market's price, 3 000.
//
// Requirement: FR-REG-032, FR-REG-038
func TestReferencePrice_BandOfTheEffectivePrice(t *testing.T) {
	t.Parallel()
	w := newMill(t, dearFlour(), WithSeed(7))
	away, home := []AgentID{2, 3}, []AgentID{4, 5}
	if got := flourBeliefs(t, w, away...); !slices.Equal(got, []int64{40_000, 40_000}) {
		t.Errorf("flour beliefs of away at construction = %v, want [40000 40000]", got)
	}
	if got := flourBeliefs(t, w, home...); !slices.Equal(got, []int64{3_000, 3_000}) {
		t.Errorf("flour beliefs of home at construction = %v, want [3000 3000]", got)
	}
	check := func(when string) {
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() %s error = %v, want nil", when, err)
		}
		row := snap.Markets[w.markets.index(0, flour)]
		if row.AtFloor != 0 || row.AtCeiling != 0 {
			t.Errorf("snapshot %s: flour in away has %d beliefs at the floor and %d at the ceiling, want none: 40 000 is inside the band of 50 000", when, row.AtFloor, row.AtCeiling)
		}
	}
	check("at construction")
	step(t, w, 3)
	check("after 3 ticks")
	for _, c := range flourBeliefs(t, w, away...) {
		if c <= 30_000 {
			t.Errorf("flour belief of away after 3 ticks = %d, want above 30 000, the ceiling of the commodity's own band", c)
		}
	}
}

// TestReferencePrice_SnapshotBoundsAreTheEffectiveBand verifies that the snapshot
// counts a belief at the bounds of its own region's band: a belief seeded at the
// lowest price the band of the override allows is at the floor, and the two agents
// of that region are counted, and none of the other region.
//
// Arithmetic: RefBandLow is 1 000 basis points, a tenth: the floor of the band of
// 50 000 is 5 000. Against the commodity's own price the floor would be 300.
//
// Requirement: FR-REG-032, FR-REG-038
func TestReferencePrice_SnapshotBoundsAreTheEffectiveBand(t *testing.T) {
	t.Parallel()
	c := dearFlour()
	c.WarmStart.Prices[0].Price = 5_000
	w := newMill(t, c, WithSeed(7))
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	away, home := snap.Markets[w.markets.index(0, flour)], snap.Markets[w.markets.index(1, flour)]
	if away.AtFloor != 2 || away.AtCeiling != 0 || home.AtFloor != 0 || home.AtCeiling != 0 {
		t.Errorf("beliefs at the bounds: away floor %d ceiling %d, home floor %d ceiling %d, want 2, 0 and 0, 0", away.AtFloor, away.AtCeiling, home.AtFloor, home.AtCeiling)
	}
}

// TestReferencePrice_ValuePricesAndSeeds verifies the two remaining readers of the
// reference price in a tick: the price at which the welfare record values a
// market that has never traded, and the seeds of the beliefs of a new entrant
// (spec-regions.md 5.2.2). Both are the effective price of the region of the
// market, not the commodity's.
//
// Requirement: FR-REG-032
func TestReferencePrice_ValuePricesAndSeeds(t *testing.T) {
	t.Parallel()
	w := newMill(t, dearFlour(), WithSeed(7))
	w.valuePrices()
	for r := range RegionID(2) {
		for k := range CommodityID(2) {
			if got, want := w.oc.valued[w.markets.index(r, k)], int64(w.ReferencePrice(r, k)); got != want {
				t.Errorf("valued price of market (%d, %d) = %d, want the effective reference price %d", r, k, got, want)
			}
		}
	}
	row := w.roles.row(0)
	for g := range RegionID(2) {
		w.fillSeeds(g, row)
		for i, c := range row.tradable {
			if got, want := w.ex.seeds[i].Ref, int64(w.ReferencePrice(g, c)); got != want {
				t.Errorf("seed Ref of commodity %d in region %d = %d, want the effective reference price %d", c, g, got, want)
			}
		}
	}
}

// TestPrices_SeedsFollowTheRegionsReferencePrice verifies that the warm start the
// solver writes uses the effective reference price of each region: the band that
// lifts a seed, the record of a seed outside the band and the drift all come from
// the region's price, not the commodity's.
//
// Arithmetic: one commodity at 100, a dual of 100 and the floor of 50. In region
// a the price is 100, the band is 10 to 1 000 and the seed 100 stays. In region b
// the override is 5 000, the band 500 to 50 000, so the seed is lifted to 500: a
// record of an outside-band seed, and a drift of (500 - 5 000) * 10 000 / 5 000 =
// -9 000 basis points.
//
// Requirement: FR-REG-032, FR-VIA-025
func TestPrices_SeedsFollowTheRegionsReferencePrice(t *testing.T) {
	t.Parallel()
	cfg := (&Config{
		World: DefaultWorldParams(), Regions: []Region{{ID: "a"}, {ID: "b"}}, Commodities: []Commodity{{ID: "x", ReferencePrice: 100}},
		RegionPrices: []RegionPrice{{Region: "b", Commodity: "x", Price: 5_000}},
	}).canonical()
	run := func(r RegionID, id string) (*regionCtx, error) {
		rc := &regionCtx{
			s: &viaSolve{m: &viaModel{c: &cfg}}, p: &viaProgramme{pr: steady.Outcome{Price: []steady.Frac{steady.Rat(100, 1)}}}, r: r, id: id,
			scale: steady.Rat(1, 1), price: make([]int64, 1), table: []int{0},
		}
		return rc, rc.prices()
	}
	a, err := run(0, "a")
	if err != nil || a.price[0] != 100 || len(a.s.rep.OutsideReferenceBand) != 0 || a.s.rep.Drift[0].DriftBP != 0 {
		t.Errorf("prices of a = %v, %v, record %+v, want seed 100, no record and no drift", a.price, err, a.s.rep)
	}
	b, err := run(1, "b")
	if err != nil || b.price[0] != 500 || len(b.s.rep.OutsideReferenceBand) != 1 || b.s.rep.Drift[0].DriftBP != -9_000 {
		t.Errorf("prices of b = %v, %v, record %+v, want seed 500, one outside-band record and a drift of -9000", b.price, err, b.s.rep)
	}
}

// TestValidatorDeclare_DearestIgnoresRowsOfNoRegion verifies that the dearest
// price of a commodity counts only the override rows of a declared region and
// commodity: a stray row for an unknown region must not raise the bound that the
// derived-product overflow rule reads.
//
// Guard: without the filter a row that ErrUnknownRegion already reports would
// also fail the overflow rule of every recipe that reads the commodity.
//
// Requirement: FR-REG-037
func TestValidatorDeclare_DearestIgnoresRowsOfNoRegion(t *testing.T) {
	t.Parallel()
	c := regionMill(
		RegionPrice{Region: "away", Commodity: "grain", Price: 1_500},
		RegionPrice{Region: "nowhere", Commodity: "grain", Price: 900_000},
		RegionPrice{Region: "away", Commodity: "unknown", Price: 800_000},
	)
	v := &validator{c: c.canonical()}
	v.declare()
	if got, want := v.dearest["grain"], Price(1_500); got != want {
		t.Errorf("dearest[grain] = %d, want %d (the unknown region's 900000 is not counted)", got, want)
	}
	if _, ok := v.dearest["unknown"]; ok {
		t.Error("dearest has an entry for an undeclared commodity, want none")
	}
}

// TestOverridesBelow_RepeatedRowCountsOnce verifies that a repeated override row
// counts its region once, so a default below the floor is still reported while
// another region uses it.
//
// Guard: two rows for one region (ErrDuplicateID, reported elsewhere) would
// otherwise equal the region count and hide the default's error until the
// duplicate is fixed.
//
// Requirement: FR-REG-037
func TestOverridesBelow_RepeatedRowCountsOnce(t *testing.T) {
	t.Parallel()
	c := regionMill(
		RegionPrice{Region: "away", Commodity: "grain", Price: 1_100},
		RegionPrice{Region: "away", Commodity: "grain", Price: 1_200},
	)
	v := &validator{c: c.canonical()}
	v.declare()
	got := v.overridesBelow(1, map[string]bool{"grain": true})
	if got["grain"] != 1 {
		t.Errorf("overridesBelow(...)[grain] = %d, want 1 region for two rows of one region", got["grain"])
	}
}
