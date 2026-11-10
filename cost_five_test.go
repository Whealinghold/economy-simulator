package economysim_test

import (
	"sync"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file holds the cost benchmarks of the Stage 2 exit (plan 2-24 Phase 6f, D692,
// criterion 6 of vision.md 5). Nothing here asserts a time (testing.md 8.2, 10.2):
// the figures are a benchmark result on a named machine, recorded against
// `x3.ms_per_tick` of bands/five.toml and read by the owner. The window and the
// helpers (benchConfig, stepBench, warmUp) are those of scale_bench_test.go.

// MEASUREMENT RECORD (plan 2-24 Phase 6f, 2026-11-10, branch stage-2-24c). Nothing in this
// file asserts a figure (testing.md 8.2); the figures are one machine, a shared one.
//
//	Machine: Intel Core i9-14900KF (32 logical CPUs), Linux 7.0.0-31, go1.27.4 linux/amd64,
//	GOMAXPROCS=4. Another gate F ran on the host: load average 44 at the start of the run,
//	33 at its end, 30 for the profile. Plan 4's quiet record (load 1.0) had five at 33.66 ms,
//	so five (50.2 ms) was 1.5 times its quiet figure. No quiet run of the other worlds exists:
//	the 1.5 is not applied to them, and none of the figures below is shown to be a bound.
//	Command: go test -run '^$' -bench 'StepFive$|StepFiveAll|StepFiveScale|StepAgents'
//	-benchtime 50x -benchmem -count 5 . (and, in examples/realm, -bench StepRealm). Window:
//	3 warm-up steps from the warm start, 30 timed steps, a fresh world; seed 1.
//
//	World             ms/tick (median of 5; min-max)   agents at start   B/tick    allocs/tick
//	five (modules off)   50.2   (45.9-52.7)               5 000          4.86 MiB   778
//	five all-on         106.9  (100.6-114.8)               5 339          3.91 MiB   764
//	realm (all-on)       68.7   (64.1-73.1)               5 339          3.90 MiB   769
//	all-on n=quarter     28.9   (22.9-35.7)               1 504          1.26 MiB   517
//	all-on n=half        59.9   (50.9-81.4)               2 759          2.26 MiB   621
//	all-on n=full        90.3   (75.0-103.7)              5 339          3.91 MiB   764
//
//	Budget of spec-scale.md 6.2: about 53 ms a tick for 5 000 agents. five is at it (0.95);
//	the all-on world is 2.0 times it in the five-run median and 1.3 times (70.0 ms, one run
//	of 100 steps) when the load had fallen to 30. The cause of that gap was NOT isolated:
//	load and run order may explain it, and so may the harness. realm is the same economy
//	renamed but is not the same measurement: it runs through Game.Advance with no strategy
//	options list of its own, in a later slot, and its live agents are not logged. Do not read
//	realm 68.7 ms as the all-on world with the load removed. Retake on a quiet host; the
//	owner reads it (R3, D692).
//	NFR-SCL-001: time per doubling of agents quarter to half 2.07, half to full 1.51 (agents
//	1.83 and 1.94 times; the limit of spec-scale.md 2.2 is on time, not exceeded), 19.2, 21.7
//	and 16.9 us per agent-tick; the spread of the five runs is larger than the differences.
//	Cost model: no c_0 and c_t are fitted. five (|T_a| 11.69, 10.0 us per agent-tick) and
//	the all-on world (|T_a| 13.41, 20.0 us) differ in the modules, not only in |T_a|, so
//	two points give a slope (5.8 us per item) and a negative c_0 that mean nothing.
//	Live agents: the all-on world goes from 5 339 to 4 617 over the 33 steps (collapsing,
//	N-325), so the window times a working market that thins out.
//	Largest book (seed 1, 33 steps, OrderPlaced per tick per (region, commodity), both
//	sides together and counting orders the book may reject; no per-side split, so a side is
//	at most this): 1 268 orders, below the "few thousand" trigger of N-241; the duplicate scan is still 8.4% of a
//	step in the profile below.
//	CPU profile of BenchmarkStepFiveAll (100 steps; Step = 12.80 s of 30.95 s sampled, the
//	rest the solver of Validate): p32Trade 49.9% of Step, fillAgentView 42.6% (called from
//	trade and work), work 17.3%, p41Clear 13.2% (book.admit 11.6%, book.duplicate 8.4%
//	flat), against the 6% of spec-scale.md 6.3; reseed: stream.drawRange 1.7% and
//	PCG.Uint64 1.6% (N-222: the keyed draw is inlined and has no frame of its own).

// costScaleSteps are the worlds of BenchmarkStepFiveScale: the all-on pack with the
// households of every group unchanged and the Scale of the world 4, 2 and 1, so the
// agents are about a quarter, a half and all of the 5 000 (spec-scale.md 6.3,
// FR-SCL-016). Households are never divided: that would change the economy.
var costScaleSteps = []struct {
	name  string
	scale int64
}{{"n=quarter", 4}, {"n=half", 2}, {"n=full", 1}}

// scaled sets the Scale of the world and holds the groups of a merchant role at 1: a
// merchant agent has a weight of 1 (ErrMerchantWeight), so those groups keep 1 household
// per agent at every size.
func scaled(k int64) func(*economysim.Config) {
	return func(c *economysim.Config) {
		c.World.Scale = k
		for i := range c.Groups {
			for _, r := range c.Roles {
				if r.ID == c.Groups[i].Role && r.Merchant != nil {
					c.Groups[i].Scale = 1
				}
			}
		}
	}
}

// bookObserver counts the orders placed in each (region, commodity) book in a tick
// and keeps the largest count, to set against the "few thousand" trigger of N-241.
type bookObserver struct {
	cur  map[[2]int]int64
	most int64
}

// Observe counts OrderPlaced and closes the tick at TickEnded.
func (o *bookObserver) Observe(e economysim.Event) {
	switch ev := e.(type) {
	case economysim.TickStarted:
		o.cur = map[[2]int]int64{}
	case economysim.OrderPlaced:
		k := [2]int{int(ev.Region), int(ev.Commodity)}
		o.cur[k]++
		o.most = max(o.most, o.cur[k])
	}
}

var logBookOnce sync.Map

// logBook logs once per name the largest number of orders placed in one (region,
// commodity) book in a tick over the benchmark window, from a second world with the
// counting observer (the timed world has none).
func logBook(b *testing.B, name string, cfg economysim.Config, opts ...economysim.Option) {
	b.Helper()
	if _, done := logBookOnce.LoadOrStore(name, true); done {
		return
	}
	var bo bookObserver
	w, err := economysim.NewWorld(cfg, append([]economysim.Option{economysim.WithSeed(scaleBenchSeed), economysim.WithObserver(&bo)}, opts...)...)
	if err != nil {
		b.Fatalf("seed %d: NewWorld error = %v, want nil", scaleBenchSeed, err)
	}
	for range 3 + benchWindow {
		if err := w.Step(); err != nil {
			b.Fatalf("seed %d: Step() error = %v, want nil", scaleBenchSeed, err)
		}
	}
	b.Logf("%s: largest book of one (region, commodity) in a tick over %d steps = %d orders placed (N-241 trigger: a few thousand)",
		name, 3+benchWindow, bo.most)
}

// BenchmarkStepFiveAll reports a Step of the all-on `five` (5 000 agents, every
// version 1 module on, the default logger and observer) and the largest book per
// tick. It reports and asserts nothing; the figure is recorded against x3.ms_per_tick.
//
// Requirement: NFR-SCL-002, NFR-SCL-003, FR-REF-039
func BenchmarkStepFiveAll(b *testing.B) {
	cfg := benchConfig(b, allLayers, nil)
	logBook(b, "five-all", cfg, allOptions()...)
	stepBench(b, "five-all", cfg, 5000, allOptions()...)
}

// BenchmarkStepFiveScale reports a Step of the all-on `five` at the Scale of the
// world 4, 2 and 1; each sub-benchmark reports ns/agent-tick over the agents it
// has, so the ratio per doubling of agents (NFR-SCL-001) is read from the output
// and recorded, not asserted.
//
// Requirement: NFR-SCL-001, NFR-SCL-003, FR-SCL-016
func BenchmarkStepFiveScale(b *testing.B) {
	for _, tt := range costScaleSteps {
		b.Run(tt.name, func(b *testing.B) {
			cfg := benchConfig(b, allLayers, scaled(tt.scale))
			stepBench(b, "five-all "+tt.name, cfg, costAgents(b, cfg), allOptions()...)
		})
	}
}

// costAgents is the number of agents the all-on world of cfg builds.
func costAgents(tb testing.TB, cfg economysim.Config) int64 {
	tb.Helper()
	w, err := economysim.NewWorld(cfg, append([]economysim.Option{economysim.WithSeed(scaleBenchSeed)}, allOptions()...)...)
	if err != nil {
		tb.Fatalf("seed %d: NewWorld error = %v, want nil", scaleBenchSeed, err)
	}
	s, err := w.Snapshot()
	if err != nil {
		tb.Fatalf("Snapshot() error = %v, want nil", err)
	}
	return s.Agents
}
