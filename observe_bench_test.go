package economysim_test

import (
	"os"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// This file holds the benchmarks of a Step of the pack `one` (plan-phase-1-11.md
// Phase 10, item 6; testing.md 8.1; NFR-WLD-002, a SHOULD): with no observer and
// with the recorder. They report time and allocations and assert nothing (N-015).
// The benchmarks of each phase are in observe_bench_internal_test.go.

// benchOne builds the world of the pack `one` with the options and steps it
// through its first ticks, so the scratch has grown.
func benchOne(b *testing.B, opts ...economysim.Option) *economysim.World {
	b.Helper()
	cfg, err := config.LoadFS(os.DirFS("testdata/chains"), "base.toml", "one.toml")
	if err != nil {
		b.Fatalf("LoadFS(one) error = %v, want nil", err)
	}
	w, err := economysim.NewWorld(cfg, append([]economysim.Option{economysim.WithSeed(obsSeed)}, opts...)...)
	if err != nil {
		b.Fatalf("seed %d: NewWorld(one) error = %v, want nil", obsSeed, err)
	}
	for range 3 {
		if err := w.Step(); err != nil {
			b.Fatalf("seed %d: Step() error = %v, want nil", obsSeed, err)
		}
	}
	return w
}

// BenchmarkStep_OneNoObserver reports a Step of `one` with the default observer
// and logger.
func BenchmarkStep_OneNoObserver(b *testing.B) {
	w := benchOne(b)
	b.ReportAllocs()
	for b.Loop() {
		if err := w.Step(); err != nil {
			b.Fatalf("seed %d: Step() error = %v, want nil", obsSeed, err)
		}
	}
}

// BenchmarkStep_OneRecorder reports a Step of `one` with the recording observer,
// which keeps the events of the step and drops them before the next.
func BenchmarkStep_OneRecorder(b *testing.B) {
	rec := &recorder{}
	w := benchOne(b, economysim.WithObserver(rec))
	b.ReportAllocs()
	for b.Loop() {
		rec.forget()
		if err := w.Step(); err != nil {
			b.Fatalf("seed %d: Step() error = %v, want nil", obsSeed, err)
		}
	}
}
