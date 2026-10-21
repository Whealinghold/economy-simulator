package economysim_test

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file builds the scenarios that the observability tests of plan-phase-1-11.md
// Phase 10 share (spec-observability.md 9.7): the pack `one`, and Go-built
// configurations that make each condition of the Warn rows and each event of the
// slice happen. The packs `famine` and `five` are Stage 2 (N-295). Every scenario
// is seeded with obsSeed, which the failure messages name (testing.md 6.2). The
// only fakes are the sanctioned ones at the extension points (D142): a consumer
// Spawner that scripts a plan, a consumer Strategy and a faulty Clearing.

// obsSeed is the seed of every scenario of the observability tests.
const obsSeed = 20261022

// stampedLine is a log record with the tick of the step that wrote it; tick is
// -1 for a line of construction.
type stampedLine struct {
	tick int64
	logLine
}

// observedRun is what a scenario produced: the world, every event, every line and
// the error of the step that failed it, if any.
type observedRun struct {
	w      *economysim.World
	events []economysim.Event
	lines  []stampedLine
	err    error
	steps  int64
}

// scenario is one run: a configuration, the options that give it its extensions
// and a number of ticks. cfg and opts are functions because a Config is a value
// that a test may change and a fake keeps state: each run gets fresh ones.
type scenario struct {
	name  string
	cfg   func(t *testing.T) economysim.Config
	opts  func() []economysim.Option
	ticks int64
	// tail is the ticks that one Run call steps after the ticks stepped one by
	// one, so that RunStarted and RunEnded occur in the scenario.
	tail int64
}

// who selects what a run attaches besides the options of the scenario.
type who struct{ observer, logger bool }

// build makes the world of sc with the seed obsSeed and the attachments of with,
// and returns it with the recorder and the capturing handler it was given.
func build(t *testing.T, sc scenario, with who, extra ...economysim.Option) (*economysim.World, *recorder, *captureHandler) {
	t.Helper()
	rec, h := &recorder{}, newCaptureHandler()
	opts := []economysim.Option{economysim.WithSeed(obsSeed)}
	if sc.opts != nil {
		opts = append(opts, sc.opts()...)
	}
	if with.observer {
		opts = append(opts, economysim.WithObserver(rec))
	}
	if with.logger {
		opts = append(opts, economysim.WithLogger(slog.New(h)))
	}
	w, err := economysim.NewWorld(sc.cfg(t), append(opts, extra...)...)
	if err != nil {
		t.Fatalf("%s (seed %d): NewWorld() error = %v, want nil", sc.name, obsSeed, err)
	}
	return w, rec, h
}

// play builds the world of sc, steps it sc.ticks times, or until a step fails,
// and collects what it recorded. Each log line is stamped with the tick of the
// step that wrote it.
func play(t *testing.T, sc scenario, with who, extra ...economysim.Option) observedRun {
	t.Helper()
	w, rec, h := build(t, sc, with, extra...)
	run := observedRun{w: w}
	seen := 0
	collect := func(tick int64) {
		recs := h.records()
		for _, l := range recs[seen:] {
			run.lines = append(run.lines, stampedLine{tick: tick, logLine: l})
		}
		seen = len(recs)
	}
	collect(-1)
	for range sc.ticks {
		tick := w.Tick()
		err := w.Step()
		collect(tick)
		if err != nil {
			run.err = err
			break
		}
		run.steps++
	}
	if run.err == nil && sc.tail > 0 {
		tick := w.Tick()
		n, err := w.Run(context.Background(), sc.tail)
		collect(tick)
		run.steps, run.err = run.steps+n, err
	}
	run.events = rec.events
	return run
}

// warnLines returns the Warn lines of a run with the message, in order.
func (r observedRun) warnLines(msg string) []stampedLine {
	var out []stampedLine
	for _, l := range r.lines {
		if l.level == slog.LevelWarn && l.msg == msg {
			out = append(out, l)
		}
	}
	return out
}

// eventsOf returns the events of one type of a run, in order.
func eventsOf[T economysim.Event](r observedRun) []T {
	var out []T
	for _, e := range r.events {
		if v, ok := e.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

// intAttr returns the integer attribute of a line, or -1 when the line has none.
func (l logLine) intAttr(key string) int64 {
	if v, ok := l.attr(key); ok && v.Kind() == slog.KindInt64 {
		return v.Int64()
	}
	return -1
}

// scriptedSpawner is a consumer Spawner that appends what plan returns for the
// view it is given. A nil plan is a spawner that does nothing, so no household
// leaves or enters and a condition persists.
type scriptedSpawner struct {
	plan func(v *economysim.RegionView) []economysim.Decision
}

// Decide appends the scripted plan.
func (s scriptedSpawner) Decide(v *economysim.RegionView, _ *rand.Rand, d *economysim.Decisions) {
	if s.plan != nil {
		d.List = append(d.List, s.plan(v)...)
	}
}

// strategyFake is a consumer Strategy that wraps the default and adds an
// invalid intent for agent 1 each tick, and counts the agents it was called for.
// calls is by the agent ID of the call; the fake is not shared between worlds.
type strategyFake struct {
	invalid bool
	work    map[economysim.AgentID]int
	trade   map[economysim.AgentID]int
	revise  map[economysim.AgentID]int
}

// newStrategyFake returns a fake that counts its calls and, with invalid, places
// an order of an unknown commodity for agent 1.
func newStrategyFake(invalid bool) *strategyFake {
	return &strategyFake{invalid: invalid,
		work: map[economysim.AgentID]int{}, trade: map[economysim.AgentID]int{}, revise: map[economysim.AgentID]int{}}
}

// Work counts the call and defers to the default.
func (s *strategyFake) Work(v *economysim.AgentView, rng *rand.Rand) economysim.JobRequest {
	s.work[v.Agent]++
	return economysim.DefaultStrategy{}.Work(v, rng)
}

// Trade counts the call, defers to the default and adds the invalid order.
func (s *strategyFake) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	s.trade[v.Agent]++
	economysim.DefaultStrategy{}.Trade(v, rng, p)
	if s.invalid && v.Agent == 1 {
		p.Intents = append(p.Intents, economysim.Intent{Commodity: 99, Side: economysim.Bid, Quantity: 1, Limit: 900})
	}
}

// Revise counts the call and defers to the default.
func (s *strategyFake) Revise(o *economysim.OutcomeView, rng *rand.Rand) economysim.Belief {
	s.revise[o.Agent]++
	return economysim.DefaultStrategy{}.Revise(o, rng)
}

// badClearing is a consumer Clearing that returns a trade of one agent with
// itself: the world refuses it as an invalid trade list (D151).
type badClearing struct{}

// Clear returns the one invalid trade.
func (badClearing) Clear(_, _ []economysim.Order, _ *rand.Rand) []economysim.Trade {
	return []economysim.Trade{{Commodity: 0, Buyer: 1, Seller: 1, Price: 1, Quantity: 1}}
}

// millCfg is the small chain of the scenarios: one region "home" and one role
// "miller" whose recipe "mill" takes 2 "grain" and gives 3 "flour" in duration
// ticks; each household holds 10 000 minor units and 10 grain; grain is also the
// upkeep. The households are in groups of weight 1 (World.Scale 1, MinAgents 1),
// so the agent IDs are the estate 0 and the households 1 to households.
func millCfg(duration, households int64) economysim.Config {
	c := minimalConfig()
	c.Commodities = append(c.Commodities, economysim.Commodity{ID: "flour", ReferencePrice: 3_000})
	c.Recipes = []economysim.Recipe{{
		ID: "mill", Duration: duration, TraineeYield: 5_000,
		Inputs:  []economysim.Item{{Commodity: "grain", Quantity: 2}},
		Outputs: []economysim.Output{{Commodity: "flour", Quantity: 3, CostShare: 10_000}},
	}}
	c.World.OutputCoverTicks = 0
	c.World.Scale, c.World.MinAgents = 1, 1
	c.Roles[0].ID, c.Roles[0].Recipes = "miller", []string{"mill"}
	c.Roles[0].StartStock = []economysim.Item{{Commodity: "grain", Quantity: 10}}
	c.Groups = []economysim.Group{{Region: "home", Role: "miller", Households: households}}
	return c
}

// scripted returns the options that register a spawner under "script" and select
// it in the configuration through the returned function.
func scripted(plan func(v *economysim.RegionView) []economysim.Decision) func() []economysim.Option {
	return func() []economysim.Option {
		return []economysim.Option{economysim.WithSpawner("script", scriptedSpawner{plan: plan})}
	}
}

// withSpawner selects the registered spawner "script".
func withSpawner(c economysim.Config) economysim.Config {
	c.World.Spawner = "script"
	return c
}

// removeEveryone removes every agent of the region in the first tick: no replacement.
func removeEveryone(v *economysim.RegionView) []economysim.Decision {
	if v.Tick != 0 {
		return nil
	}
	var d []economysim.Decision
	for _, a := range v.Agents {
		d = append(d, economysim.Decision{Kind: economysim.DecideRemove, Agent: a.Agent, Cause: economysim.CauseStarved})
	}
	return d
}

// scenarios returns the scenarios of the observability tests, by name. Each call
// gives fresh fakes.
func scenarios() []scenario {
	return []scenario{
		{name: "one", ticks: 12, tail: 2, cfg: func(t *testing.T) economysim.Config { return loadPack(t, "base.toml", "one.toml") }},
		{name: "starving", ticks: 25, cfg: func(*testing.T) economysim.Config {
			c := millCfg(1, 6)
			c.Roles[0].StartMoney, c.Roles[0].StartStock = 10, nil
			return c
		}},
		spawnerScript(),
		{name: "bad order", ticks: 3,
			cfg: func(*testing.T) economysim.Config {
				c := millCfg(1, 4)
				c.Roles[0].Strategy = "fake"
				return c
			},
			opts: func() []economysim.Option {
				return []economysim.Option{economysim.WithStrategy("fake", newStrategyFake(true))}
			}},
		{name: "faulty clearing", ticks: 3,
			cfg: func(*testing.T) economysim.Config {
				c := millCfg(1, 4)
				c.World.Clearing = "faulty"
				return c
			},
			opts: func() []economysim.Option {
				return []economysim.Option{economysim.WithClearing("faulty", badClearing{})}
			}},
	}
}

// spawnerScript is the scenario of the consumer spawner: a plan of a reduction
// (cause starved), a switch, an entry and a removal of an agent with a job in
// progress.
//
// Arithmetic: roles are in canonical order carter (0), miller (1); the carter
// group has 2 households of weight 1 (agents 1 and 2) and the miller group 6
// households in 3 agents of weight 2 (agents 3, 4 and 5). In tick 0 agent 3 is
// reduced by 1 household (its 10 000 * 2 / 2 = 10 000 money goes to the pool, so
// the pool holds 10 000 and EntryPoolShare 10 000 makes the whole of it the
// budget), agent 1 switches to the miller role and 1 household enters the miller
// role at the start money 10 000 (funded). The recipe takes 3 ticks, so in tick 1
// agent 4 holds the job it started in tick 0 and its removal abandons it.
func spawnerScript() scenario {
	plan := func(v *economysim.RegionView) []economysim.Decision {
		switch v.Tick {
		case 0:
			return []economysim.Decision{
				{Kind: economysim.DecideReduce, Agent: 3, Households: 1, Cause: economysim.CauseStarved},
				{Kind: economysim.DecideSwitch, Agent: 1, Role: 1, Cause: economysim.CauseDistress},
				{Kind: economysim.DecideEnter, Role: 1, Households: 1, Cause: economysim.CauseConsumer},
			}
		case 1:
			return []economysim.Decision{{Kind: economysim.DecideRemove, Agent: 4, Cause: economysim.CauseStarved}}
		}
		return nil
	}
	return scenario{name: "spawner script", ticks: 4, opts: scripted(plan),
		cfg: func(*testing.T) economysim.Config {
			c := withSpawner(millCfg(3, 6))
			carter := economysim.DefaultRole()
			carter.ID, carter.StartStock = "carter", []economysim.Item{{Commodity: "grain", Quantity: 10}}
			c.Roles = append(c.Roles, carter)
			c.World.EntryPoolShare = 10_000
			c.Groups = []economysim.Group{
				{Region: "home", Role: "carter", Households: 2},
				{Region: "home", Role: "miller", Households: 6, Scale: 2},
			}
			return c
		}}
}

// warnCase is a scenario that makes the condition of one Warn row hold from the
// tick start on, so that the run of the condition is tick - start + 1 at tick.
// pattern is 'A' (the first tick of a run and every tenth) or 'B' (every tenth),
// spec-observability.md 7.3. cell names the attributes that identify the cell
// whose lines are bounded (FR-OBS-072); a row with no run has no pattern.
type warnCase struct {
	scenario
	msg     string
	pattern byte
	start   int64
	cell    []string
}

// warnTicks is the ticks in [start, start+n) at which a run that began at start
// writes its line under the pattern: A at run 1 and every multiple of 10, B at
// every multiple of 10 (warnRun, observer.go).
func warnTicks(pattern byte, start, n int64) []int64 {
	var out []int64
	for tick := start; tick < start+n; tick++ {
		run := tick - start + 1
		if run%10 == 0 || (pattern == 'A' && run == 1) {
			out = append(out, tick)
		}
	}
	return out
}

// warnCases are the scenarios of the rows of the Warn table that a run reaches
// from the root package (rows 2, 3, 6, 7, 8 and 19; row 1 is every scenario with
// a warning in its configuration; rows 4 and 26 need the ring of a household and
// are an internal test). Each runs 25 ticks, so a run that holds from its first
// tick reaches its tenth and twentieth.
func warnCases() []warnCase {
	idle := func(c economysim.Config) economysim.Config { return withSpawner(c) }
	return []warnCase{
		{msg: "households unfed", pattern: 'B', start: 0, cell: []string{"region", "role"}, scenario: scenario{
			name: "unfed", ticks: 25, opts: scripted(nil),
			cfg: func(*testing.T) economysim.Config {
				c := idle(millCfg(1, 2))
				c.Roles[0].StartStock = nil
				return c
			}}},
		{msg: "role extinct", pattern: 'A', start: 0, cell: []string{"region", "role"}, scenario: scenario{
			name: "extinct", ticks: 25, opts: scripted(removeEveryone),
			cfg: func(*testing.T) economysim.Config { return idle(millCfg(1, 3)) }}},
		{msg: "entry unfunded", pattern: 'B', start: 0, cell: []string{"region"}, scenario: scenario{
			name: "unfunded", ticks: 25,
			opts: scripted(func(*economysim.RegionView) []economysim.Decision {
				return []economysim.Decision{{Kind: economysim.DecideEnter, Role: 0, Households: 1, Cause: economysim.CauseConsumer}}
			}),
			cfg: func(*testing.T) economysim.Config { return idle(millCfg(1, 3)) }}},
		{msg: "group below floor", pattern: 'A', start: 0, cell: []string{"region", "role"}, scenario: scenario{
			name: "below floor", ticks: 25,
			opts: scripted(func(v *economysim.RegionView) []economysim.Decision {
				if v.Tick != 0 {
					return nil
				}
				return []economysim.Decision{{Kind: economysim.DecideRemove, Agent: 1, Cause: economysim.CauseStarved}}
			}),
			cfg: func(*testing.T) economysim.Config {
				c := idle(millCfg(1, 6))
				c.World.MinAgents = 3
				c.Groups[0].Scale = 3
				return c
			}}},
		{msg: "recipe output disposed", scenario: scenario{
			name: "disposed", ticks: 3, opts: scripted(nil),
			cfg: func(*testing.T) economysim.Config {
				c := idle(millCfg(1, economysim.MaxWeight))
				c.World.Scale = economysim.MaxWeight
				c.Roles[0].StartStock = append(c.Roles[0].StartStock,
					economysim.Item{Commodity: "flour", Quantity: economysim.MaxQuantity / economysim.MaxWeight})
				return c
			}}},
		{msg: "price beliefs at bound", pattern: 'B', start: 0, cell: []string{"region", "commodity"}, scenario: scenario{
			name: "bound", ticks: 25, opts: scripted(nil),
			cfg: func(*testing.T) economysim.Config {
				c := idle(millCfg(1, 6))
				c.World.RefBandLow, c.World.RefBandHigh = 10_000, 10_000
				return c
			}}},
	}
}
