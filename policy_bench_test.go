package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// BenchmarkPolicyView reports the cost of what the actors of the module-alone pack read at one tick
// (NFR-PLC-002, plan 3-plc Phase 8): the views of the two actors, on the warm five of
// 5 000 households, on a world of seed 1 after 3 steps. "no regions" builds the levers and the
// treasury only; "regions, shared" builds the states of the five covered regions for the pass
// (council_a covers the centre, the north and the west, council_b the east and the south), with the
// journeys, contracts, treasuries and shipments read once; "regions, per region" builds the whole-world
// parts once for each covered region, as the module did before the review item of PR B. The figures
// are recorded in the log of the PR, not asserted (N-015).
//
// Requirement: NFR-PLC-002
//
// Guard: a view that read the whole world once for each region would show as the gap between the
// two "regions" rows.
func BenchmarkPolicyView(b *testing.B) {
	for _, bc := range []struct {
		name          string
		reads, shared bool
	}{{"no regions", false, true}, {"regions, shared", true, true}, {"regions, per region", true, false}} {
		b.Run(bc.name, func(b *testing.B) {
			w := policyBenchWorld(b, bc.reads)
			b.ReportAllocs()
			var n int
			for b.Loop() {
				n = economysim.PolicyViews(w, true, bc.shared)
			}
			b.ReportMetric(float64(n), "regions/op")
		})
	}
}

// policyBenchWorld is the pack of the golden, seed 1, after 3 steps, with every actor reading
// regions (a rule and a schedule that ignores them) or none (both on hold).
func policyBenchWorld(b *testing.B, reads bool) *economysim.World {
	b.Helper()
	cfg := loadValidPack(b, fivePolicy...)
	for i := range cfg.Modules.Policy.Actors {
		a := &cfg.Modules.Policy.Actors[i]
		a.ReadsRegions = reads
		if !reads { // the rule's gauge needs the regions: hold its place
			a.Policy, a.Rules = "hold", nil
			a.Schedule = nil
		}
	}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
	if err != nil {
		b.Fatalf("NewWorld() error = %v, want nil", err)
	}
	for range 3 {
		if err := w.Step(); err != nil {
			b.Fatalf("Step() error = %v, want nil", err)
		}
	}
	return w
}
