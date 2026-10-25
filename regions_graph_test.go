package economysim_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// tradeEdges returns the edges of the trade overlay of a variant as
// spec-regions.md 12.4 proposes them: placeholder distances, both directions.
// The overlay files wait for Phase 9; the tables are the same.
func tradeEdges(pack string) []economysim.Edge {
	both := func(a, b string, d int64) []economysim.Edge {
		return []economysim.Edge{edgeOf(a, b, d), edgeOf(b, a, d)}
	}
	var out []economysim.Edge
	switch pack {
	case "two":
		out = both("north", "south", 100)
	case "three":
		out = append(both("south", "north", 100), both("south", "far_south", 100)...)
	case "five":
		for _, r := range []string{"east", "north", "south", "west"} {
			out = append(out, both("centre", r, 100)...)
		}
		out = append(out, both("north", "east", 130)...)
		out = append(out, both("south", "west", 130)...)
	}
	return out
}

// TestGraph_QueriesOnPacks verifies the graph of the packs of the reference chain
// (spec-reference-chain.md 4): Name and Lookup invert each other for every
// region and refuse what is not one; a pack with no edge has a component for each
// region, one edge-free region being its own; and with the edges of 12.4 the
// variants two, three and five are one component, with the distances the spec
// gives: north and far_south are 200 apart in three, east and west 200 through
// centre in five, east and north 130 directly.
//
// Requirement: FR-REG-004, FR-REG-015, FR-REG-016
func TestGraph_QueriesOnPacks(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, name := range []string{"one", "second", "two", "three", "five"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := loadPack(t, packLayers[name]...)
			g, err := economysim.NewGraph(cfg)
			if err != nil {
				t.Fatalf("NewGraph(%s) error = %v, want nil", name, err)
			}
			for r := range economysim.RegionID(g.Len()) {
				id := g.Name(r)
				if back, ok := g.Lookup(id); !ok || back != r || id == "" {
					t.Errorf("Lookup(Name(%d) = %q) = %d, %v, want %d, true", r, id, back, ok, r)
				}
			}
			if _, ok := g.Lookup("no_such_region"); ok || g.Name(economysim.RegionID(g.Len())) != "" || g.Name(economysim.NoRegion) != "" {
				t.Errorf("Lookup of an unknown ID or Name of an ID outside the range gave a region")
			}
			if g.Components() != g.Len() || g.Edges() != 0 {
				t.Errorf("Graph(%s) with no edge: %d components and %d edges, want %d and 0", name, g.Components(), g.Edges(), g.Len())
			}
			edges := tradeEdges(name)
			if len(edges) == 0 {
				return
			}
			cfg.Edges = edges
			traded, err := economysim.NewGraph(cfg)
			if err != nil {
				t.Fatalf("NewGraph(%s with trade edges) error = %v, want nil", name, err)
			}
			if traded.Components() != 1 {
				t.Errorf("Graph(%s with trade edges) has %d components, want 1", name, traded.Components())
			}
			dist := func(a, b string) int64 {
				ra, _ := traded.Lookup(a)
				rb, _ := traded.Lookup(b)
				d, ok := traded.Distance(ra, rb)
				if !ok {
					t.Fatalf("Distance(%s, %s) in %s is unreachable, want a distance", a, b, name)
				}
				return d
			}
			switch name {
			case "three":
				if d := dist("north", "far_south"); d != 200 {
					t.Errorf("Distance(north, far_south) = %d, want 200 (through south)", d)
				}
			case "five":
				if d := dist("east", "west"); d != 200 {
					t.Errorf("Distance(east, west) = %d, want 200 (through centre)", d)
				}
				if d := dist("east", "north"); d != 130 {
					t.Errorf("Distance(east, north) = %d, want 130 (the direct edge, shorter than 100 + 100)", d)
				}
				east, _ := traded.Lookup("east")
				west, _ := traded.Lookup("west")
				centre, _ := traded.Lookup("centre")
				if p, ok := traded.Path(east, west); !ok || !slices.Equal(p, []economysim.RegionID{east, centre, west}) {
					t.Errorf("Path(east, west) = %v, %v, want [east centre west]", p, ok)
				}
			}
		})
	}
}

// TestGraph_OutInOrder verifies that Out and In list the open edges in ascending
// To and ascending From with their effective length, that the lists are the
// caller's to change, and that Edge and EdgeAt agree with the configuration
// sorted as the EdgeID is defined.
//
// Requirement: FR-REG-023, FR-REG-014, FR-REG-026
func TestGraph_OutInOrder(t *testing.T) {
	t.Parallel()
	cfg := withRegions(referenceConfig(), "east", "south") // east 0, north 1, south 2
	cfg.Edges = []economysim.Edge{
		edgeOf("south", "east", 7), edgeOf("north", "south", 5), edgeOf("east", "south", 3),
		edgeOf("north", "east", 4), edgeOf("south", "north", 6),
	}
	g, err := economysim.NewGraph(cfg)
	if err != nil {
		t.Fatalf("NewGraph() error = %v, want nil", err)
	}
	wantOut := map[economysim.RegionID][]economysim.Hop{
		0: {{Region: 2, Distance: 3}},
		1: {{Region: 0, Distance: 4}, {Region: 2, Distance: 5}},
		2: {{Region: 0, Distance: 7}, {Region: 1, Distance: 6}},
	}
	wantIn := map[economysim.RegionID][]economysim.Hop{
		0: {{Region: 1, Distance: 4}, {Region: 2, Distance: 7}},
		1: {{Region: 2, Distance: 6}},
		2: {{Region: 0, Distance: 3}, {Region: 1, Distance: 5}},
	}
	for r := range economysim.RegionID(3) {
		if got := g.Out(r); !slices.Equal(got, wantOut[r]) {
			t.Errorf("Out(%d) = %v, want %v", r, got, wantOut[r])
		}
		if got := g.In(r); !slices.Equal(got, wantIn[r]) {
			t.Errorf("In(%d) = %v, want %v", r, got, wantIn[r])
		}
	}
	// The edges in canonical order: east>south, north>east, north>south, south>east,
	// south>north, with the EdgeIDs 0 to 4.
	wantID := []economysim.EdgeView{
		{ID: 0, From: 0, To: 2, Base: 3, Distance: 3, FactorBP: 10_000},
		{ID: 1, From: 1, To: 0, Base: 4, Distance: 4, FactorBP: 10_000},
		{ID: 2, From: 1, To: 2, Base: 5, Distance: 5, FactorBP: 10_000},
		{ID: 3, From: 2, To: 0, Base: 7, Distance: 7, FactorBP: 10_000},
		{ID: 4, From: 2, To: 1, Base: 6, Distance: 6, FactorBP: 10_000},
	}
	if g.Edges() != len(wantID) {
		t.Errorf("Edges() = %d, want %d", g.Edges(), len(wantID))
	}
	for _, want := range wantID {
		if got, ok := g.EdgeAt(want.ID); !ok || got != want {
			t.Errorf("EdgeAt(%d) = %+v, %v, want %+v", want.ID, got, ok, want)
		}
		if got, ok := g.Edge(want.From, want.To); !ok || got != want {
			t.Errorf("Edge(%d, %d) = %+v, %v, want %+v", want.From, want.To, got, ok, want)
		}
	}
	out := g.Out(1)
	out[0].Distance = 99
	if g.Out(1)[0].Distance != 4 {
		t.Errorf("changing a list returned by Out changed the graph: Out(1) = %v", g.Out(1))
	}
}

// TestWorldGraph_MatchesNewGraph verifies that World.Graph builds the same graph as
// NewGraph from the configuration while no status is set, and that it works on a
// world that has run.
//
// Requirement: FR-REG-028, FR-WLD-110
func TestWorldGraph_MatchesNewGraph(t *testing.T) {
	t.Parallel()
	cfg := withRegions(referenceConfig(), "east", "south")
	cfg.Edges = []economysim.Edge{edgeOf("north", "south", 5), edgeOf("south", "north", 6), edgeOf("east", "north", 2)}
	w, err := economysim.NewWorld(cfg)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	want, err := economysim.NewGraph(cfg)
	if err != nil {
		t.Fatalf("NewGraph() error = %v, want nil", err)
	}
	g := w.Graph()
	for a := range economysim.RegionID(3) {
		for b := range economysim.RegionID(3) {
			dg, okg := g.Distance(a, b)
			dw, okw := want.Distance(a, b)
			if dg != dw || okg != okw {
				t.Errorf("World.Graph().Distance(%d, %d) = %d, %v, want %d, %v (NewGraph)", a, b, dg, okg, dw, okw)
			}
		}
	}
	if g.Components() != want.Components() || g.Edges() != want.Edges() {
		t.Errorf("World.Graph() has %d components and %d edges, want %d and %d", g.Components(), g.Edges(), want.Components(), want.Edges())
	}
	if _, err := economysim.NewGraph(economysim.Config{}); err == nil {
		t.Errorf("NewGraph(empty configuration) error = nil, want ErrNoRegions")
	}
}

// TestNewGraph_RunsTheGraphRules verifies that NewGraph returns the first error of
// the graph rules as a *ValidationError, and builds nothing for a configuration
// that breaks one: it needs no commodity, role or group.
//
// Requirement: FR-REG-030
func TestNewGraph_RunsTheGraphRules(t *testing.T) {
	t.Parallel()
	regions := []economysim.Region{{ID: "a"}, {ID: "b"}}
	tests := []struct {
		name string
		cfg  economysim.Config
		want error
	}{
		{"no region", economysim.Config{}, economysim.ErrNoRegions},
		{"duplicate region", economysim.Config{Regions: []economysim.Region{{ID: "a"}, {ID: "a"}}}, economysim.ErrDuplicateID},
		{"invalid region ID", economysim.Config{Regions: []economysim.Region{{ID: "A"}}}, economysim.ErrInvalidID},
		{"unknown end", economysim.Config{Regions: regions, Edges: []economysim.Edge{edgeOf("a", "z", 1)}}, economysim.ErrUnknownRegion},
		{"loop", economysim.Config{Regions: regions, Edges: []economysim.Edge{edgeOf("a", "a", 1)}}, economysim.ErrSelfEdge},
		{"distance 0", economysim.Config{Regions: regions, Edges: []economysim.Edge{edgeOf("a", "b", 0)}}, economysim.ErrOutOfBounds},
		{"pair twice", economysim.Config{Regions: regions, Edges: []economysim.Edge{edgeOf("a", "b", 1), edgeOf("a", "b", 2)}}, economysim.ErrDuplicateID},
		{"too many edges", economysim.Config{Regions: regions, Edges: make([]economysim.Edge, economysim.MaxEdges+1)}, economysim.ErrTooMany},
		{"override of an unknown commodity", economysim.Config{Regions: regions, RegionPrices: []economysim.RegionPrice{{Region: "a", Commodity: "z", Price: 1}}}, economysim.ErrUnknownCommodity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g, err := economysim.NewGraph(tt.cfg)
			var ve *economysim.ValidationError
			if !errors.Is(err, tt.want) || !errors.As(err, &ve) || g.Len() != 0 {
				t.Errorf("NewGraph(%s) = %d regions, %v, want no graph and a *ValidationError of %v", tt.name, g.Len(), err, tt.want)
			}
		})
	}
}

// TestGraph_NoAllocation verifies that the query methods named by NFR-REG-001 and
// World.ReferencePrice allocate nothing, with testing.AllocsPerRun: a benchmark
// would report the trend and never fail.
//
// Requirement: NFR-REG-001
func TestGraph_NoAllocation(t *testing.T) {
	cfg := withRegions(referenceConfig(), "east", "south")
	cfg.Edges = []economysim.Edge{edgeOf("north", "south", 5), edgeOf("south", "north", 6), edgeOf("east", "north", 2)}
	cfg.RegionPrices = []economysim.RegionPrice{{Region: "south", Commodity: "seed", Price: 300}}
	w, err := economysim.NewWorld(cfg)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	g := w.Graph()
	var sink int64
	calls := map[string]func(){
		"Len":            func() { sink += int64(g.Len()) },
		"Name":           func() { sink += int64(len(g.Name(1))) },
		"Lookup":         func() { r, _ := g.Lookup("south"); sink += int64(r) },
		"Edges":          func() { sink += int64(g.Edges()) },
		"Edge":           func() { e, _ := g.Edge(1, 2); sink += e.Base },
		"EdgeAt":         func() { e, _ := g.EdgeAt(1); sink += e.Base },
		"Distance":       func() { d, _ := g.Distance(0, 2); sink += d },
		"NextHop":        func() { r, _ := g.NextHop(0, 2); sink += int64(r) },
		"Component":      func() { c, _ := g.Component(1); sink += int64(c) },
		"Components":     func() { sink += int64(g.Components()) },
		"ReferencePrice": func() { sink += int64(w.ReferencePrice(2, 4)) },
	}
	for name, call := range calls {
		if got := testing.AllocsPerRun(100, call); got != 0 {
			t.Errorf("%s allocates %v times per call, want 0", name, got)
		}
	}
	_ = sink
}
