package economysim_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s28Link is a pair of regions whose edges, both ways, a scenario closes.
type s28Link struct{ a, b string }

// TestScenarioS28_MiddleLinkClosure verifies the graph leg of scenario S28 (REG-A1,
// spec-reference-chain.md 7.6): on a pack with the edges of the trade overlay
// (spec-regions.md 12.4) the middle link closes and then reopens, between ticks,
// through the unexported status setter (the seam SetEdgeStatusesForTest: the public
// call is plan 2-17's, as the Stage 1 yield seam is N-296's). At each of the three
// steps (open, closed, reopened) the distances, the path, the components and the
// answers of RelocationRefusal are checked, and after the reopening the graph
// equals the one before the closure. The price leg waits for TRN and MER.
//
// Cases: on three the link is south to north, so north is cut off from the rest and
// the graph splits in two, with north (component 0) alone and south with far_south
// (component 1); on five the link is centre to north, which has a detour through
// east, 100 (centre to east) + 130 (east to north) = 230 against the 100 it had.
//
// Requirement: FR-REG-049, FR-REG-050, FR-REG-051, FR-REG-016, FR-REG-027
// Guard: a closure that left a stale table, a reopening that did not restore the
// first answers, or a closure that touched the relocation of an account.
func TestScenarioS28_MiddleLinkClosure(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cases := []struct {
		pack       string
		link       s28Link
		components [2]int // open and closed
		probe      s28Link
		probeOpen  int64
		probeShut  int64 // -1: unreachable
	}{
		{"three", s28Link{"south", "north"}, [2]int{1, 2}, s28Link{"north", "far_south"}, 200, -1},
		{"five", s28Link{"centre", "north"}, [2]int{1, 1}, s28Link{"centre", "north"}, 100, 230},
	}
	for _, tt := range cases {
		t.Run(tt.pack, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, packLayers[tt.pack]...)
			cfg.Edges = tradeEdges(t, tt.pack)
			for _, seed := range seedList(scenarioSeedCount()) {
				t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
					t.Parallel()
					scenSem <- struct{}{}
					defer func() { <-scenSem }()
					s28Seed(t, cfg, tt.pack, seed, tt.link, tt.probe, tt.probeOpen, tt.probeShut, tt.components)
				})
			}
		})
	}
}

// s28Seed runs one seed: 30 ticks open, the closure, 30 ticks closed, the
// reopening, 30 ticks open.
func s28Seed(t *testing.T, cfg economysim.Config, pack string, seed uint64, link, probe s28Link, probeOpen, probeShut int64, components [2]int) {
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed))
	if err != nil {
		t.Fatalf("pack %s seed %d: NewWorld() = %v, want nil", pack, seed, err)
	}
	run := func(ticks int) {
		for range ticks {
			if err := w.Step(); err != nil {
				t.Fatalf("pack %s seed %d: Step() = %v, want nil", pack, seed, err)
			}
		}
	}
	set := func(closed bool) {
		g := w.Graph()
		a, _ := g.Lookup(link.a)
		b, _ := g.Lookup(link.b)
		changes := []economysim.EdgeStatus{{From: a, To: b, Closed: closed, FactorBP: 10_000}, {From: b, To: a, Closed: closed, FactorBP: 10_000}}
		if err := economysim.SetEdgeStatusesForTest(w, changes, economysim.CauseConsumer); err != nil {
			t.Fatalf("pack %s seed %d: set closed=%v: %v", pack, seed, closed, err)
		}
	}
	// refusals is the answer of RelocationRefusal for each agent and region now.
	refusals := func() []string {
		var out []string
		n := w.Graph().Len()
		for a := range economysim.AgentID(n + 3) {
			for r := range economysim.RegionID(n) {
				reason, err := w.RelocationRefusal(a, r)
				out = append(out, fmt.Sprintf("%d>%d:%s:%v", a, r, reason, err))
			}
		}
		return out
	}
	distance := func(g economysim.Graph, x, y string) int64 {
		a, _ := g.Lookup(x)
		b, _ := g.Lookup(y)
		if d, ok := g.Distance(a, b); ok {
			return d
		}
		return -1
	}

	run(30)
	open := w.Graph()
	refusalsOpen := refusals()
	if got := distance(open, probe.a, probe.b); got != probeOpen || open.Components() != components[0] {
		t.Errorf("pack %s seed %d, open: Distance(%s, %s) = %d with %d components, want %d with %d", pack, seed, probe.a, probe.b, got, open.Components(), probeOpen, components[0])
	}

	set(true)
	shut := w.Graph()
	if got := distance(shut, probe.a, probe.b); got != probeShut || shut.Components() != components[1] {
		t.Errorf("pack %s seed %d, closed: Distance(%s, %s) = %d with %d components, want %d with %d", pack, seed, probe.a, probe.b, got, shut.Components(), probeShut, components[1])
	}
	a, _ := shut.Lookup(link.a)
	b, _ := shut.Lookup(link.b)
	if e, ok := shut.Edge(a, b); !ok || !e.Closed || e.Distance != 0 {
		t.Errorf("pack %s seed %d, closed: Edge(%s, %s) = %+v, %v, want the edge, closed", pack, seed, link.a, link.b, e, ok)
	}
	if shut.Components() == 2 {
		ca, _ := shut.Component(a)
		cb, _ := shut.Component(b)
		if ca == cb {
			t.Errorf("pack %s seed %d, closed: %s and %s are in one component %d, want two", pack, seed, link.a, link.b, ca)
		}
	}
	if got := refusals(); !slices.Equal(got, refusalsOpen) {
		t.Errorf("pack %s seed %d, closed: RelocationRefusal answers differ from the open graph's: a closure must not touch relocation", pack, seed)
	}
	run(30)

	set(false)
	if again := w.Graph(); !reflect.DeepEqual(open, again) {
		t.Errorf("pack %s seed %d, reopened: the graph differs from the one before the closure", pack, seed)
	}
	run(30)
}
