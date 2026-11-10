package main

import "testing"

// benchWindow is the timed steps after 3 warm-up steps before the game is built
// again, the window of the root module's cost benchmarks (scale_bench_test.go).
const benchWindow = 30

// BenchmarkStepRealm reports a tick of the realm pack (the all-on `five` renamed, the
// default logger and no observer) through Game.Advance, the cost the server pays per
// step. It reports and asserts nothing; the figure is recorded against x3.ms_per_tick
// of bands/five.toml (plan 2-24 D692, NFR-SCL-002).
func BenchmarkStepRealm(b *testing.B) {
	cfg, err := packOnce()
	if err != nil {
		b.Fatalf("Load() = %v, want nil", err)
	}
	fresh := func() *Game {
		g, err := NewGameFrom(cfg, 1)
		if err != nil {
			b.Fatalf("NewGameFrom(seed 1) = %v, want nil", err)
		}
		if _, err := g.Advance(3); err != nil {
			b.Fatalf("Advance(3) = %v, want nil", err)
		}
		return g
	}
	g, steps := fresh(), 0
	b.ReportAllocs()
	for b.Loop() {
		if steps == benchWindow {
			b.StopTimer()
			g, steps = fresh(), 0
			b.StartTimer()
		}
		steps++
		if _, err := g.Advance(1); err != nil {
			b.Fatalf("Advance(1) = %v, want nil", err)
		}
	}
}
