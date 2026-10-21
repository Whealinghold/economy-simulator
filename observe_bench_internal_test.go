package economysim

import (
	"slices"
	"testing"
)

// This file holds the benchmarks of the tick per phase (plan-phase-1-11.md
// Phase 10, item 6; testing.md 8.1; NFR-WLD-002, a SHOULD): each runs the world of
// the allocation test through the unexported position methods and times one phase
// alone, so the cost of a phase can be read from its own line. No threshold is
// asserted (N-015). The world and the sites are those of observe_internal_test.go.

// benchWorld builds the world of the allocation test: four households that
// produce, trade, update their beliefs and meet the spawner's refusals.
func benchWorld(b *testing.B, opts ...Option) *World {
	b.Helper()
	plan := []Decision{{Kind: DecideEnter, Role: 0, Households: 1, Cause: CauseConsumer}}
	base := []Option{WithSeed(obsInternalSeed), WithSpawner("script", &scriptSpawner{plan: func(*RegionView) []Decision { return plan }}),
		WithStrategy("alloc", allocStrategy{}), WithClearing("match", &matchFirst{})}
	w, err := NewWorld(allocConfig(), append(base, opts...)...)
	if err != nil {
		b.Fatalf("seed %d: NewWorld() error = %v, want nil", obsInternalSeed, err)
	}
	return w
}

// benchPhases are the phases of the tick as groups of the sites of the straight
// line, by position ID: P0 is the empty P0.2 and the bracket of the step, then
// the eight phases.
var benchPhases = []struct {
	name  string
	sites []string
}{
	{"P0", []string{"P0.2"}},
	{"P1", []string{"P1.1", "P1.2"}},
	{"P2", []string{"P2.1", "P2.5", "P2.7", "P2.8a"}},
	{"P3", []string{"P3.2", "P3.4"}},
	{"P4", []string{"P4.1"}},
	{"P5", []string{"P5.1", "P5.3", "P5.4"}},
	{"P6", []string{"P6.1", "P6.3", "P6.4"}},
	{"P7", []string{"P7.1"}},
	{"P8", []string{"P8.1", "P8.2", "P8.4"}},
}

// runSites calls the sites in order and fails the benchmark on an error.
func runSites(b *testing.B, w *World, sites []site) {
	b.Helper()
	for _, s := range sites {
		if err := s.call(w); err != nil {
			b.Fatalf("seed %d: %s error = %v, want nil", obsInternalSeed, s.name, err)
		}
	}
}

// BenchmarkPhase reports the time and the allocations of each phase of a replayed
// step, with the other phases outside the timer. It is the per-phase trend that
// NFR-WLD-002 asks for; a benchmark reports and never fails CI (testing.md 8.2).
//
// Requirement: NFR-WLD-002
func BenchmarkPhase(b *testing.B) {
	for _, ph := range benchPhases {
		first := slices.IndexFunc(stepSites, func(s site) bool { return s.name == ph.sites[0] })
		last := slices.IndexFunc(stepSites, func(s site) bool { return s.name == ph.sites[len(ph.sites)-1] })
		if first < 0 || last < first {
			b.Fatalf("phase %s names sites %v that are not in the table, want a run of the straight line", ph.name, ph.sites)
		}
		b.Run(ph.name, func(b *testing.B) {
			w := benchWorld(b)
			for range 8 {
				if err := w.Step(); err != nil {
					b.Fatalf("seed %d: Step() error = %v, want nil", obsInternalSeed, err)
				}
			}
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				if err := w.beginStep(); err != nil {
					b.Fatalf("seed %d: beginStep() error = %v, want nil", obsInternalSeed, err)
				}
				runSites(b, w, stepSites[:first])
				b.StartTimer()
				runSites(b, w, stepSites[first:last+1])
				b.StopTimer()
				runSites(b, w, stepSites[last+1:])
				if err := w.endStep(nil); err != nil {
					b.Fatalf("seed %d: endStep() error = %v, want nil", obsInternalSeed, err)
				}
				b.StartTimer()
			}
		})
	}
}

// BenchmarkStepWorld reports a whole step of the same world with the default
// observer and logger.
func BenchmarkStepWorld(b *testing.B) {
	w := benchWorld(b)
	b.ReportAllocs()
	for b.Loop() {
		if err := w.Step(); err != nil {
			b.Fatalf("seed %d: Step() error = %v, want nil", obsInternalSeed, err)
		}
	}
}
