package economysim_test

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// tradeOverlay is the overlay file of a variant (spec-regions.md 12.4).
func tradeOverlay(pack string) string { return "overlay/trade-" + pack + ".toml" }

// tradeWarm is the layers of the warm pack with its trade overlay.
func tradeWarm(pack string) []string {
	return []string{"base.toml", pack + ".toml", tradeOverlay(pack), "warm-" + pack + ".toml"}
}

// withTrade is the layers of a pack followed by its trade overlay.
func withTrade(pack string) []string {
	return append(slices.Clone(packLayers[pack]), tradeOverlay(pack))
}

// TestPacks_TradeOverlaysLoad verifies that each trade overlay of spec-regions.md
// 12.4 loads over its variant, validates, has the directed edges the table gives
// (two: 2, three: 4, five: 12), makes the pack one strongly connected component,
// leaves the two overlay-free warnings out, and gives another configuration hash
// than the variant alone.
//
// Requirement: FR-REG-054, FR-REG-015, FR-REG-053
func TestPacks_TradeOverlaysLoad(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	// The tables of spec-regions.md 12.4, one line for each pair, both directions.
	table := map[string][][3]any{
		"two":   {{"north", "south", 100}},
		"three": {{"south", "north", 100}, {"south", "far_south", 100}},
		"five": {{"centre", "east", 100}, {"centre", "north", 100}, {"centre", "south", 100}, {"centre", "west", 100},
			{"north", "east", 130}, {"south", "west", 130}},
	}
	for pack, pairs := range table {
		t.Run(pack, func(t *testing.T) {
			t.Parallel()
			bare := loadValidPack(t, packLayers[pack]...)
			cfg := loadValidPack(t, withTrade(pack)...)
			var want []economysim.Edge
			for _, p := range pairs {
				a, b, d := p[0].(string), p[1].(string), int64(p[2].(int))
				want = append(want, edgeOf(a, b, d), edgeOf(b, a, d))
			}
			byEnds := func(x, y economysim.Edge) int { return cmp.Or(cmp.Compare(x.From, y.From), cmp.Compare(x.To, y.To)) }
			slices.SortFunc(want, byEnds)
			if got := slices.SortedFunc(slices.Values(cfg.Edges), byEnds); !slices.Equal(got, want) {
				t.Errorf("Edges of %s with its overlay = %v, want %v (spec-regions.md 12.4)", pack, got, want)
			}
			g, err := economysim.NewGraph(cfg)
			if err != nil {
				t.Fatalf("NewGraph(%s + overlay) error = %v, want nil", pack, err)
			}
			if g.Components() != 1 {
				t.Errorf("Components(%s + overlay) = %d, want 1", pack, g.Components())
			}
			ws := cfg.Warnings()
			for _, rule := range []string{"region_isolated", "graph_not_strongly_connected", "edge_not_shortest"} {
				if got := warned(ws, rule); len(got) != 0 {
					t.Errorf("%s of %s + overlay = %v, want none", rule, pack, got)
				}
			}
			if economysim.HashConfig(cfg) == economysim.HashConfig(bare) {
				t.Errorf("HashConfig(%s + overlay) equals the hash of %s alone, want different", pack, pack)
			}
		})
	}
}

// TestPacks_RegionIsolatedWithoutOverlay verifies that a variant with several
// regions and no edge (the probe of Stage 1) loads, validates and draws
// region_isolated for each region and graph_not_strongly_connected, with as many
// components as regions (three: 3), and that a one-region pack draws neither.
//
// Requirement: FR-REG-031, FR-REG-007, FR-REG-015
func TestPacks_RegionIsolatedWithoutOverlay(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	tests := []struct {
		pack    string
		regions []string
	}{
		{"one", nil},
		{"two", []string{"regions/north", "regions/south"}},
		{"three", []string{"regions/far_south", "regions/north", "regions/south"}},
		{"five", []string{"regions/centre", "regions/east", "regions/north", "regions/south", "regions/west"}},
	}
	for _, tt := range tests {
		t.Run(tt.pack, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, packLayers[tt.pack]...)
			ws := cfg.Warnings()
			if got := warned(ws, "region_isolated"); !slices.Equal(got, tt.regions) {
				t.Errorf("region_isolated of %s = %v, want %v", tt.pack, got, tt.regions)
			}
			wantNot := 0
			if len(tt.regions) > 0 {
				wantNot = 1
			}
			if got := warned(ws, "graph_not_strongly_connected"); len(got) != wantNot {
				t.Errorf("graph_not_strongly_connected of %s = %v, want %d line(s)", tt.pack, got, wantNot)
			}
			g, err := economysim.NewGraph(cfg)
			if err != nil {
				t.Fatalf("NewGraph(%s) error = %v, want nil", tt.pack, err)
			}
			if g.Components() != g.Len() {
				t.Errorf("Components(%s) = %d, want %d", tt.pack, g.Components(), g.Len())
			}
		})
	}
}

// TestPacks_EdgeListOrderIndependence verifies that the packs with their trade
// overlays and every entity list, the edge list included, in reverse order give
// the same graph, the same configuration hash and the same run as the sorted
// ones.
//
// Requirement: FR-REG-002, FR-REG-014, FR-REG-053
// Guard: the tables index typed IDs into the sorted slices; a builder that read
// the caller's edge order would answer differently, or panic.
func TestPacks_EdgeListOrderIndependence(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, pack := range []string{"two", "three", "five"} {
		t.Run(pack, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, withTrade(pack)...)
			rev := cfg
			rev.Regions, rev.Edges, rev.Groups = slices.Clone(cfg.Regions), slices.Clone(cfg.Edges), slices.Clone(cfg.Groups)
			slices.Reverse(rev.Regions)
			slices.Reverse(rev.Edges)
			slices.Reverse(rev.Groups)
			if economysim.HashConfig(rev) != economysim.HashConfig(cfg) {
				t.Errorf("HashConfig(%s reversed) differs from the sorted one", pack)
			}
			g1, err1 := economysim.NewGraph(cfg)
			g2, err2 := economysim.NewGraph(rev)
			if err1 != nil || err2 != nil {
				t.Fatalf("NewGraph(%s) errors = %v, %v, want nil", pack, err1, err2)
			}
			for a := range economysim.RegionID(g1.Len()) {
				for b := range economysim.RegionID(g1.Len()) {
					d1, ok1 := g1.Distance(a, b)
					d2, ok2 := g2.Distance(a, b)
					n1, m1 := g1.NextHop(a, b)
					n2, m2 := g2.NextHop(a, b)
					if d1 != d2 || ok1 != ok2 || n1 != n2 || m1 != m2 {
						t.Errorf("%s: (%d, %d) answers %d,%v,%d,%v and %d,%v,%d,%v, want equal", pack, a, b, d1, ok1, n1, m1, d2, ok2, n2, m2)
					}
				}
			}
			run := func(c economysim.Config) economysim.Snapshot {
				return runPack(t, c, runSpec{Seed: 1, Ticks: 20}).Final
			}
			if a, b := run(cfg), run(rev); !reflect.DeepEqual(a, b) {
				t.Errorf("snapshot of %s after 20 ticks differs between the sorted and the reversed lists", pack)
			}
		})
	}
}

// quiet is a run of cfg with the parts that name the graph removed, so two runs
// that differ only in their edges can be compared: the events from the first
// TickStarted on, and the final snapshot without its configuration hash and
// component count.
func quiet(res result) (string, economysim.Snapshot) {
	var evs []economysim.Event
	started := false
	for _, e := range res.Events {
		started = started || eventType(e) == "TickStarted"
		if started {
			evs = append(evs, e)
		}
	}
	s := res.Final
	s.ConfigHash, s.Components = [32]byte{}, 0
	return fmt.Sprintf("%+v", evs), s
}

// TestWorldRun_EdgesDoNotMoveAPack verifies that three with its trade overlay and
// three without it run the same for 50 ticks from seed 1: the events of the run
// and the snapshot are equal but for the configuration hash and the component
// count, because no module reads the graph yet.
//
// Requirement: FR-REG-056 (spec-regions.md 2, row 11: REG is core and non-interfering)
// Guard: an edge that reached a market, an order or a draw would move a pack
// whose regions do not trade.
func TestWorldRun_EdgesDoNotMoveAPack(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	spec := runSpec{Seed: 1, Ticks: 50, Observe: true}
	bare := runPack(t, loadValidPack(t, packLayers["three"]...), spec)
	traded := runPack(t, loadValidPack(t, withTrade("three")...), spec)
	e1, s1 := quiet(bare)
	e2, s2 := quiet(traded)
	if e1 != e2 {
		t.Errorf("events of three with and without its overlay differ")
	}
	if !reflect.DeepEqual(s1, s2) {
		t.Errorf("final snapshot of three with and without its overlay differs")
	}
	if bare.Final.Components != 3 || traded.Final.Components != 1 {
		t.Errorf("Components = %d without and %d with the overlay, want 3 and 1", bare.Final.Components, traded.Final.Components)
	}
}

// TestPacks_TradeDeterminism verifies, for the packs two and three with their
// trade overlays, that the final state does not depend on the way the run is
// driven or watched: the cases of detCases(true), seed 1, end in the snapshot and
// state hash of a bare Step run. The invariants of testing.md 6.3 over the same
// packs are the two rows of invariantTable (TestInvariants_Packs).
//
// Requirement: FR-REF-025, FR-REG-056
// Guard: the packs gain edges; a run that differs by observation or chunking
// would show first here.
func TestPacks_TradeDeterminism(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, pack := range []string{"two", "three"} {
		t.Run(pack, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, tradeWarm(pack)...)
			base := runPack(t, cfg, detStep(runSpec{Seed: 1, Ticks: detTicks}))
			for _, tc := range detCases(true) {
				spec := tc.spec()
				spec.Seed, spec.Ticks = 1, detTicks
				got := runPack(t, cfg, spec)
				if !reflect.DeepEqual(got.Final, base.Final) {
					t.Errorf("%s case %q: final snapshot differs from the bare run", pack, tc.name)
				}
				if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
					t.Errorf("%s case %q: state hash differs from the bare run", pack, tc.name)
				}
			}
		})
	}
}
