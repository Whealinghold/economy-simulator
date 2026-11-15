package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// techGains returns the gain of each (region, field) row of the snapshot of w, by region.
func techGains(t *testing.T, w *economysim.World) map[economysim.RegionID]int64 {
	t.Helper()
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	out := map[economysim.RegionID]int64{}
	for _, r := range s.Tech {
		out[r.Region] = r.GainBP
	}
	return out
}

// setEdges applies the edge statuses to w.
func setEdges(t *testing.T, w *economysim.World, statuses []economysim.EdgeStatus) {
	t.Helper()
	if err := w.SetEdgeStatuses(statuses); err != nil {
		t.Fatalf("SetEdgeStatuses() error = %v, want nil", err)
	}
}

// runTicks steps w n ticks.
func runTicks(t *testing.T, w *economysim.World, n int) {
	t.Helper()
	for range n {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
}

// TestContact_ClosedEdgeStopsAndResumes verifies, on the pack three with diffusion alone
// (every other mechanism of technology-four.toml set to 0), that a region lagging behind
// a leader with an advance of 2 000 gains while its edges are open, stops gaining the tick
// its only edge to the leader is closed by SetEdgeStatuses (the distance table honours
// the status, so contact is lost), and gains again when the edge is reopened.
//
// Requirement: FR-TEC-036, FR-TEC-013
func TestContact_ClosedEdgeStopsAndResumes(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, technologyThree...)
	f := &cfg.Modules.Technology.Fields[0]
	f.Base, f.DoublingBP, f.TrendBP, f.DiscoveryBP, f.StepBP = 0, 0, 0, 0, 0
	w := mustWorld(t, cfg, economysim.WithSeed(1), economysim.WithAudit())
	// The regions in ID order: far_south 0, north 1, south 2; north and far_south reach
	// each other only through south. North leads.
	const far, north, south = economysim.RegionID(0), economysim.RegionID(1), economysim.RegionID(2)
	if err := w.AdjustTechnology([]economysim.TechnologyAdjustment{{Region: north, Field: 0, DeltaBP: 2_000}}); err != nil {
		t.Fatalf("AdjustTechnology() error = %v, want nil", err)
	}
	runTicks(t, w, 30)
	open := techGains(t, w)
	if !(open[south] != 0 && open[far] != 0 && open[north] == 2_000) {
		t.Fatalf("gains after 30 ticks = %v, want the lagging regions above 0 and north at 2 000", open)
	}
	cut := []economysim.EdgeStatus{{From: north, To: south, Closed: true, FactorBP: 10_000}, {From: south, To: north, Closed: true, FactorBP: 10_000}}
	setEdges(t, w, cut)
	// With north cut off, the frontier of south and far_south is the better of the two:
	// neither can pass what the better one held at the cut.
	ceiling := max(open[south], open[far])
	runTicks(t, w, 60)
	after := techGains(t, w)
	if after[north] != 2_000 || max(after[south], after[far]) > ceiling {
		t.Errorf("gains with north cut off = %v, want north at 2 000 and the others at most %d (the frontier of what they reach)", after, ceiling)
	}
	reopen := []economysim.EdgeStatus{{From: north, To: south, FactorBP: 10_000}, {From: south, To: north, FactorBP: 10_000}}
	setEdges(t, w, reopen)
	runTicks(t, w, 30)
	if got := techGains(t, w); got[south] <= ceiling {
		t.Errorf("gains after reopening = %v, want south, in contact with the leader again, above the ceiling %d", got, ceiling)
	}
}
