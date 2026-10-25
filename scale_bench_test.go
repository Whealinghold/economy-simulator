package economysim_test

import (
	"fmt"
	"os"
	"sync"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// MEASUREMENT RECORD (plan 2-4 Phase 5, 2026-10-25). Nothing in this file asserts a
// figure (testing.md 8.2); the figures are one machine and a collapsing pack.
//
//	Machine: Intel Core i9-14900KF (32 logical CPUs), Linux 7.0.0-31, Go 1.27.3
//	linux/amd64, GOMAXPROCS=4, load average 1.0 at the start (a shared
//	workstation, not an isolated one). Command:
//	go test -run '^$' -bench 'Step(Agents|Weights|Five)' -benchmem -count 6
//	then benchstat over the six: the median and the spread of the six.
//	Window: 3 warm-up steps from the warm start, 30 timed steps, a fresh world.
//
//	Live agents (the logLive line): n=600 600 -> 598, n=1200 1 200 -> 1 198, n=2400
//	2 400 -> 2 395, five 5 000 -> 4 961 from the start to the end of the window, with
//	802, 2 383, 4 845 and 10 735 trades in the last warm-up step: the window is a
//	working market, not the extinct economy of a long loop (see below).
//
//	Benchmark       sec/step         agents  ns/agent-tick  agent-ticks/s  mean|T_a|  B/step    allocs/step
//	n=600           2.342 ms +-3%      600     3 904 +-3%      256 k          4.515    310 KiB   71
//	n=1200          4.755 ms +-1%    1 200     3 962 +-1%      252 k          4.515    629 KiB   93
//	n=2400          10.41 ms +-1%    2 400     4 336 +-1%      231 k          4.515    1.29 MiB  123
//	w=1  (120 ag.)  465.9 us +-4%      120     3 883 +-4%      258 k          4.500    64.5 KiB  49
//	w=7             521.6 us +-4%      120     4 346 +-4%      230 k          4.500    74.7 KiB  52
//	w=40            529.1 us +-6%      120     4 409 +-6%      227 k          4.500    75.4 KiB  53
//	five            33.66 ms +-1%    5 000     6 732 +-1%      149 k          11.69    4.84 MiB  791
//	Under -race (allocation counts only): n=600 87, n=1200 121, n=2400 169, five 901.
//
//	NFR-SCL-001: time at 1 200 over 600 agents = 4.755 / 2.342 = 2.03; at 2 400 over
//	1 200 = 10.41 / 4.755 = 2.19; the limit of spec-scale.md 2.2 is 2.2.
//	FR-SCL-016 (run half, reported): the same 120 agents at weight 7 and 40 take
//	1.12 and 1.14 times the time of weight 1 and 3 or 4 more allocations; the
//	economies differ (110, 119 and 120 agents alive at the end of the window), so
//	the difference is not attributed to the weight.
//	Cost model, from the two points n=600 (|T_a| 4.515, 3.904 us) and five (|T_a|
//	11.69, 6.732 us): c_t = (6.732 - 3.904) / (11.69 - 4.515) = 0.394 us per item of
//	the tradable set and c_0 = 3.904 - 4.515 * 0.394 = 2.125 us per agent-tick. With
//	n=2400 as the first point they are c_t = 0.334 us and c_0 = 2.83 us. The two
//	worlds differ in more than |T_a| (five has five regions and 60 groups), so these
//	are not separated costs. Placeholders of spec-scale.md 6.2: c_0 2 us, c_t 1 us
//	(10 us per agent-tick at |T_a| 8), 100 000 agent-ticks/s; measured 149 k to
//	257 k.
//	Clearing share (BenchmarkPhase, a 4-household world, ns per phase): P4 487 of
//	a step of 5 861 (P0 185, P1 346, P2 1 497, P3 1 255, P4 487, P5 502, P6 620, P7
//	769, P8 200) is 8.3%; the 6% of spec-scale.md 6.3 is not read from a world of
//	this size.
//	CPU profile of n=2400 (30 s, 3 487 steps, Step = 40.11 s of 41.06 s): (*stream).keyed
//	0.06 s and keyedPair 0.37 s are 1.07% of Step; ordering.Finalise 0.33 s is
//	inside them; reseed is inlined and has no frame of its own; PCG.Uint64 0.18 s
//	(0.44%) and PCG.Seed 0.01 s. BenchmarkStream_KeyedDraw: 5.83 ns/op. The rest of
//	a step: p32Trade 37%, fillAgentView 23%, clearing 19%, work 15%, in the ledger's
//	lot search (BinarySearchFunc 14%) and the book's duplicate check (12%).
//	Existing benchmarks (benchOne loops on one world, so its b.N of about 100 000
//	steps times an economy that has died out; do not read them as the cost of a
//	step of `one`): BenchmarkStep_OneNoObserver 9.69 us +-3% 6 allocs,
//	BenchmarkStep_OneRecorder 12.94 us +-12% 33 allocs.
//	five steps: base, lines, five, warm-five (no edges) builds and steps; with
//	overlay/trade-five.toml (12 edges) it builds and steps too.

// This file holds the cost benchmarks of a Step at a size and at a weight (plan 2-4
// D289 to D293, NFR-SCL-001, NFR-SCL-002, NFR-SCL-003). The window is the same for
// every size: 3 warm-up steps from the warm start, then benchWindow timed steps, and
// then a fresh world again: b.Loop runs as many steps as it needs (42 000 at 600
// agents), and the pack `one` collapses (N-312, N-314), so a loop on one world would
// time an empty economy. Every figure describes the first ticks of a collapsing
// economy until plan 8 and is re-measured by plan 24; each benchmark logs once the
// live agents, the households and the trades at the start and at the end of the window.
//
// The cost model of spec-scale.md 6.1 is c_0 + |T_a| * c_t per agent-tick, with
// |T_a| the size of the tradable set of the agent's role. The benchmarks report
// the mean |T_a| over the agents beside ns/agent-tick; two worlds of different mean
// |T_a| (the scaled `one` and `five`) fix c_0 and c_t:
//
//	c_t = (cost_five - cost_one) / (|T|_five - |T|_one);  c_0 = cost_one - |T|_one * c_t

// scaleBenchSeed is the seed of every benchmark here, and benchWindow the timed
// steps after the 3 warm-up steps before the world is built again.
const (
	scaleBenchSeed uint64 = 1
	benchWindow           = 30
)

// benchLayers are the layers of the benchmarked packs.
var (
	benchLayersOne  = packOne
	benchLayersFive = []string{"base.toml", "lines.toml", "five.toml", "warm-five.toml"}
)

// benchConfig loads the layers and applies edit.
func benchConfig(tb testing.TB, layers []string, edit func(*economysim.Config)) economysim.Config {
	tb.Helper()
	cfg, err := config.LoadFS(os.DirFS(chainsDir), layers...)
	if err != nil {
		tb.Fatalf("LoadFS(%q) error = %v, want nil", layers, err)
	}
	if edit != nil {
		edit(&cfg)
	}
	if err := cfg.Validate(); err != nil {
		tb.Fatalf("Validate(%q) error = %v, want nil", layers, err)
	}
	return cfg
}

// multiplyHouseholds multiplies every group's households by k at the scale of the pack.
func multiplyHouseholds(k int64) func(*economysim.Config) {
	return func(c *economysim.Config) {
		for i := range c.Groups {
			c.Groups[i].Households *= k
		}
	}
}

// weighted gives every group 20 * w households at scale w: 20 agents of weight w
// per group, so the agents are the same at every weight.
func weighted(w int64) func(*economysim.Config) {
	return func(c *economysim.Config) {
		c.World.Scale, c.World.MinAgents = w, 1
		for i := range c.Groups {
			c.Groups[i].Households = 20 * w
		}
	}
}

// meanTradable is the mean size of the tradable set of the role of an agent, over
// the agents of cfg (a group of H households at scale S has ceil(H / S) agents).
func meanTradable(cfg economysim.Config) float64 {
	c := economysim.Canonical(cfg)
	rt := economysim.NewRoleTable(&c)
	var sum, agents int64
	for _, g := range c.Groups {
		scale := g.Scale
		if scale == 0 {
			scale = c.World.Scale
		}
		n := max(min((g.Households+scale-1)/scale, g.Households), min(c.World.MinAgents, g.Households))
		for id, r := range c.Roles {
			if r.ID == g.Role {
				sum += n * int64(len(economysim.RoleTableRow(&rt, economysim.RoleID(id)).Tradable))
			}
		}
		agents += n
	}
	return float64(sum) / float64(agents)
}

// tradeCounter counts the trades of the events it sees.
type tradeCounter struct{ trades int64 }

// Observe counts TradeExecuted.
func (c *tradeCounter) Observe(e economysim.Event) {
	if _, ok := e.(economysim.TradeExecuted); ok {
		c.trades++
	}
}

// warmUp steps a fresh world 3 times, as benchOne does.
func warmUp(tb testing.TB, cfg economysim.Config, opts ...economysim.Option) *economysim.World {
	tb.Helper()
	w, err := economysim.NewWorld(cfg, append([]economysim.Option{economysim.WithSeed(scaleBenchSeed)}, opts...)...)
	if err != nil {
		tb.Fatalf("seed %d: NewWorld error = %v, want nil", scaleBenchSeed, err)
	}
	for range 3 {
		if err := w.Step(); err != nil {
			tb.Fatalf("seed %d: Step() error = %v, want nil", scaleBenchSeed, err)
		}
	}
	return w
}

var logOnce sync.Map

// logLive logs once per name the live agents and households and the trades of the
// last step after the 3 warm-up steps and after the window, from a second world with
// a counting observer (the timed world has none), so the reader can tell a working
// market from a collapsed one.
func logLive(b *testing.B, name string, cfg economysim.Config) {
	b.Helper()
	if _, done := logOnce.LoadOrStore(name, true); done {
		return
	}
	var tc tradeCounter
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(scaleBenchSeed), economysim.WithObserver(&tc))
	if err != nil {
		b.Fatalf("seed %d: NewWorld error = %v, want nil", scaleBenchSeed, err)
	}
	state := func() string {
		s, err := w.Snapshot()
		if err != nil {
			b.Fatalf("Snapshot() error = %v, want nil", err)
		}
		return fmt.Sprintf("%d live agents, %d households", s.Agents, s.Households)
	}
	var last int64
	step := func() {
		before := tc.trades
		if err := w.Step(); err != nil {
			b.Fatalf("seed %d: Step() error = %v, want nil", scaleBenchSeed, err)
		}
		last = tc.trades - before
	}
	for range 3 {
		step()
	}
	start, startTrades := state(), last
	for range benchWindow {
		step()
	}
	b.Logf("%s: start of window %s, %d trades in its last step; end of window %s, %d trades; mean |T_a| %.2f (collapsing pack)",
		name, start, startTrades, state(), last, meanTradable(cfg))
}

// stepBench times b.N steps of the world of cfg and reports the cost per agent-tick.
func stepBench(b *testing.B, name string, cfg economysim.Config, agents int64) {
	b.Helper()
	logLive(b, name, cfg)
	w := warmUp(b, cfg)
	steps := 0
	b.ReportAllocs()
	for b.Loop() {
		if steps == benchWindow {
			b.StopTimer()
			w, steps = warmUp(b, cfg), 0
			b.StartTimer()
		}
		steps++
		if err := w.Step(); err != nil {
			b.Fatalf("seed %d: Step() error = %v, want nil", scaleBenchSeed, err)
		}
	}
	perTick := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	b.ReportMetric(perTick/float64(agents), "ns/agent-tick")
	b.ReportMetric(float64(agents)*1e9/perTick, "agent-ticks/s")
	b.ReportMetric(meanTradable(cfg), "mean|T_a|")
}

// BenchmarkStepAgents reports a Step of `one` with its households multiplied by 1,
// 2 and 4 at scale 1, so the agents are 600, 1 200 and 2 400: the time of a step
// should double with the agents (NFR-SCL-001). It reports and asserts nothing.
//
// Requirement: NFR-SCL-001, NFR-SCL-002, NFR-SCL-003
func BenchmarkStepAgents(b *testing.B) {
	for _, tt := range []struct {
		name string
		k    int64
	}{{"n=600", 1}, {"n=1200", 2}, {"n=2400", 4}} {
		b.Run(tt.name, func(b *testing.B) {
			stepBench(b, "one "+tt.name, benchConfig(b, benchLayersOne, multiplyHouseholds(tt.k)), 600*tt.k)
		})
	}
}

// BenchmarkStepWeights reports a Step of 120 agents of weight 1, 7 and 40 (20 per
// role, six roles): the same agents with other weights, which should cost the same
// (FR-SCL-016, run half). It reports and asserts nothing.
//
// Requirement: FR-SCL-016, NFR-SCL-003
func BenchmarkStepWeights(b *testing.B) {
	for _, tt := range []struct {
		name string
		w    int64
	}{{"w=1", 1}, {"w=7", 7}, {"w=40", 40}} {
		b.Run(tt.name, func(b *testing.B) {
			stepBench(b, "one "+tt.name, benchConfig(b, benchLayersOne, weighted(tt.w)), 120)
		})
	}
}

// BenchmarkStepFive reports a Step of the pack `five` (5 000 agents at scale 1 in
// five regions, no edges), the first consumer's world size and the second point of
// the cost model (plan 2-4 D290, D291). It reports and asserts nothing.
//
// Requirement: NFR-SCL-002, NFR-SCL-003
func BenchmarkStepFive(b *testing.B) {
	stepBench(b, "five", benchConfig(b, benchLayersFive, nil), 5000)
}

// TestScaleBench_BuildsTheScaledWorlds builds the worlds of the three benchmarks once
// and asserts their agents and households, so a broken fixture fails CI and not only
// the benchmark.
//
// Arithmetic: `one` has 600 households; times 1, 2 and 4 they are 600, 1 200 and
// 2 400 agents at scale 1. Six groups of 20 * w households at scale w are 6 * 20 =
// 120 agents holding 120 * w households. `five` has 5 000 households at scale 1.
//
// Requirement: NFR-SCL-003, NFR-SCL-001
func TestScaleBench_BuildsTheScaledWorlds(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, tt := range []struct {
		name               string
		layers             []string
		edit               func(*economysim.Config)
		agents, households int64
	}{
		{"n=600", benchLayersOne, multiplyHouseholds(1), 600, 600},
		{"n=1200", benchLayersOne, multiplyHouseholds(2), 1200, 1200},
		{"n=2400", benchLayersOne, multiplyHouseholds(4), 2400, 2400},
		{"w=1", benchLayersOne, weighted(1), 120, 120},
		{"w=7", benchLayersOne, weighted(7), 120, 840},
		{"w=40", benchLayersOne, weighted(40), 120, 4800},
		{"five", benchLayersFive, nil, 5000, 5000},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, err := economysim.NewWorld(benchConfig(t, tt.layers, tt.edit), economysim.WithSeed(scaleBenchSeed))
			if err != nil {
				t.Fatalf("NewWorld error = %v, want nil", err)
			}
			s, err := w.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() error = %v, want nil", err)
			}
			if s.Agents != tt.agents || s.Households != tt.households {
				t.Errorf("%s: %d agents and %d households, want %d and %d", tt.name, s.Agents, s.Households, tt.agents, tt.households)
			}
		})
	}
}
