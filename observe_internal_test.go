package economysim

import (
	"log/slog"
	"math"
	"math/rand/v2"
	"reflect"
	"runtime"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// This file holds the observability tests of plan-phase-1-11.md Phase 10 that
// need the unexported methods of the world: the two Warns that a run reaches only
// through the welfare ring (rows 4 and 26), the non-interference of the beliefs and
// the call-site allocation test (rules 8.2 (a)). Every world is seeded with
// obsInternalSeed, which the failure messages name (testing.md 6.2).

// obsInternalSeed is the seed of the worlds of this file.
const obsInternalSeed = 20261023

// allocStrategy is a consumer Strategy that wraps the default for Work and Revise
// and trades by script: in an even tick agent 1 bids 1 flour at 2 500 and agent 2
// asks 1 flour at 2 000, in an odd tick the two swap sides, so the money and the
// flour go back and forth and the trade never runs out.
type allocStrategy struct{ DefaultStrategy }

// Trade places the scripted orders.
func (allocStrategy) Trade(v *AgentView, _ *rand.Rand, p *TradePlan) {
	buyer := AgentID(1 + v.Tick.Tick%2)
	switch v.Agent {
	case buyer:
		p.Intents = append(p.Intents, Intent{Commodity: flour, Side: Bid, Quantity: 1, Limit: 2_500})
	case 3 - buyer:
		p.Intents = append(p.Intents, Intent{Commodity: flour, Side: Ask, Quantity: 1, Limit: 2_000})
	}
}

// matchFirst is a Clearing that allocates nothing: it matches the first bid that
// crosses the first ask it can, one unit at the ask's limit. The default
// AverageClearing builds a new list on every call, by the contract of the
// interface, so its cost is the baseline of the full step of row 12.
type matchFirst struct{ buf [1]Trade }

// Clear returns at most one trade, in a buffer the clearing owns.
func (m *matchFirst) Clear(bids, asks []Order, _ *rand.Rand) []Trade {
	for _, b := range bids {
		for _, a := range asks {
			if a.Agent != b.Agent && a.Price <= b.Price {
				m.buf[0] = Trade{Commodity: a.Commodity, Buyer: b.Agent, Seller: a.Agent, Price: a.Price, Quantity: 1, Lot: a.Lot}
				return m.buf[:1]
			}
		}
	}
	return nil
}

// obsMill is millConfig with the scripted spawner "script" and the clearing
// "match" selected.
func obsMill(duration, households int64) Config {
	c := millConfig(duration, households)
	c.World.Spawner, c.World.Clearing = "script", "match"
	return c
}

// obsWorld builds the world of c with the seed obsInternalSeed, the registered
// spawner "script" (the plan of sp, or none), the strategy "alloc" and the
// clearing "match", and the options.
func obsWorld(t *testing.T, c Config, sp *scriptSpawner, opts ...Option) *World {
	t.Helper()
	if sp == nil {
		sp = &scriptSpawner{}
	}
	base := []Option{WithSeed(obsInternalSeed), WithSpawner("script", sp), WithStrategy("alloc", allocStrategy{}),
		WithClearing("match", &matchFirst{})}
	return newMill(t, c, append(base, opts...)...)
}

// ringWorld builds a world of the given households whose role requests no job,
// with a capturing logger and, when observed, an observer.
func ringWorld(t *testing.T, households int64, observed bool) (*World, capture) {
	t.Helper()
	lg := newCapture()
	opts := []Option{WithLogger(slog.New(lg)), WithSeed(obsInternalSeed), WithSpawner("script", &scriptSpawner{}), WithClearing("match", &matchFirst{})}
	if observed {
		opts = append(opts, WithObserver(&evLog{}))
	}
	c := obsMill(1, households)
	c.Roles[0].Strategy = "script"
	return newScripted(t, c, &reviseScript{}, opts...), lg
}

// floorLines steps a world of 3 households 20 times, giving every ring a record of
// high income and no work before each step, and returns the lines "work floor
// binding" with the tick of each. Each line is checked against the snapshot: the
// counterpart RoleRow.HeldByFloor equals the households the line reports.
func floorLines(t *testing.T, observed bool) (lines []logRec, ticks []int64) {
	t.Helper()
	w, lg := ringWorld(t, 3, observed)
	for range 20 {
		for id := 1; id < len(w.agents); id++ {
			w.agents[id].history.Push(welfare.Record{Income: 1_000_000, Cost: 1_000})
		}
		tick := w.Tick()
		step(t, w, 1)
		for _, l := range lg.take("work floor binding") {
			lines, ticks = append(lines, l), append(ticks, tick)
			snap, err := w.Snapshot()
			if err != nil || snap.Roles[0].HeldByFloor != l.attrs["households"] {
				t.Errorf("seed %d tick %d: RoleRow.HeldByFloor = %d (%v), line households %d, want equal", obsInternalSeed, tick, snap.Roles[0].HeldByFloor, err, l.attrs["households"])
			}
		}
	}
	return lines, ticks
}

// TestWarn_WorkFloorBindingOverARun verifies the Warn "work floor binding" (row 4
// of spec-observability.md 7.3, pattern B) in a run of the world, and its
// counterpart: with every household's ring holding a high income and no working
// tick, the floor holds all 3 households to work in every tick, so the line comes
// at run 10 and 20 (ticks 9 and 19) with households 3, a logger with no observer
// writes the same lines, and RoleRow.HeldByFloor of the snapshot equals the
// households the line reports.
//
// Arithmetic: before each of the 20 steps every household's ring is given a record
// of income 1 000 000, cost 1 000 and no work; the ratio is far above the target
// and the working ticks 0 are below the floor share, so floorHeld is the 3
// households of weight 1 each tick and the run is t + 1.
//
// Requirement: FR-WEL-033, FR-OBS-051, FR-OBS-071
func TestWarn_WorkFloorBindingOverARun(t *testing.T) {
	t.Parallel()
	lines, ticks := floorLines(t, true)
	bare, bareTicks := floorLines(t, false)
	if !slices.Equal(ticks, []int64{9, 19}) || !reflect.DeepEqual(lines, bare) || !slices.Equal(ticks, bareTicks) {
		t.Fatalf("seed %d: Warn ticks %v, and without an observer %v, want [9 19] and the same lines", obsInternalSeed, ticks, bareTicks)
	}
	for _, l := range lines {
		if l.level != slog.LevelWarn || l.attrs["region"] != 0 || l.attrs["role"] != 0 || l.attrs["households"] != 3 {
			t.Errorf("seed %d: line %+v, want a Warn with region 0, role 0, households 3", obsInternalSeed, l)
		}
	}
}

// clampCalls calls the method of P6.3 20 times on a world of 2 households, with a
// record of income MaxInt64 pushed into agent 1's ring before each call, and
// returns the calls at which "welfare record clamped" was written. Each line is
// checked against the snapshot: the counterpart RoleRow.ClampedHouseholds equals
// the households the line reports.
func clampCalls(t *testing.T, observed bool) (calls []int64) {
	t.Helper()
	w, lg := ringWorld(t, 2, observed)
	for call := int64(1); call <= 20; call++ {
		if !w.agents[1].history.Push(welfare.Record{Income: math.MaxInt64}) {
			t.Fatalf("seed %d: Push(MaxInt64 income) reported no clamp, want one", obsInternalSeed)
		}
		w.p63Welfare()
		for _, l := range lg.take("welfare record clamped") {
			calls = append(calls, call)
			snap, err := w.Snapshot()
			if err != nil || l.level != slog.LevelWarn || l.attrs["households"] != 1 || snap.Roles[0].ClampedHouseholds != 1 {
				t.Errorf("seed %d call %d: line %+v, ClampedHouseholds %d (%v), want a Warn with households 1 and the same count in the snapshot", obsInternalSeed, call, l, snap.Roles[0].ClampedHouseholds, err)
			}
		}
	}
	return calls
}

// TestWarn_WelfareRecordClampedThroughTheMethodOfP63 verifies the Warn "welfare
// record clamped" (row 26, pattern A), which a run cannot reach: the ring state
// is set through the agent table and the method of P6.3 is called. A record of
// income MaxInt64 is pushed before each call, so agent 1 (weight 1) holds a
// field at the clamp at every call: the run is the number of calls, and the lines
// come at calls 1, 10 and 20 with households 1, the same with no observer, and
// RoleRow.ClampedHouseholds of the snapshot equals the line.
//
// Requirement: FR-WEL-042, FR-OBS-051, FR-OBS-071
func TestWarn_WelfareRecordClampedThroughTheMethodOfP63(t *testing.T) {
	t.Parallel()
	want := []int64{1, 10, 20}
	if got, bare := clampCalls(t, true), clampCalls(t, false); !slices.Equal(got, want) || !slices.Equal(bare, want) {
		t.Errorf("seed %d: Warn at calls %v, and without an observer %v, want %v", obsInternalSeed, got, bare, want)
	}
}

// allocConfig is the chain of the allocation test and of the beliefs test: four
// households that hold 1 000 grain and 10 flour, a recipe of 2 ticks, the
// strategy "alloc", the clearing "match" and a band of beliefs so narrow that every
// belief sits on its bound, so the Warn of row 3 and the trades are reached.
func allocConfig() Config {
	c := obsMill(2, 4)
	c.Roles[0].Strategy = "alloc"
	c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 1_000}, {Commodity: "flour", Quantity: 10}}
	c.World.RefBandLow, c.World.RefBandHigh = 10_000, 10_000
	return c
}

// TestObservers_BeliefsAndSnapshotsAreTheSameWithAndWithoutALoggerAndAnObserver
// verifies, in an internal test, that a run with a logger, an observer and
// WithAudit gives the same belief of every agent for every commodity, the same
// snapshot and the same random state after each of 6 ticks as a run without
// them: nothing that watches the world is an input of the belief update.
//
// Requirement: FR-BEL-033, FR-WLD-064
func TestObservers_BeliefsAndSnapshotsAreTheSameWithAndWithoutALoggerAndAnObserver(t *testing.T) {
	t.Parallel()
	bare := obsWorld(t, allocConfig(), nil)
	watched := obsWorld(t, allocConfig(), nil, WithLogger(slog.New(newCapture())), WithObserver(&evLog{}), WithAudit())
	for tick := range 6 {
		step(t, bare, 1)
		step(t, watched, 1)
		for id := range bare.agents {
			if !reflect.DeepEqual(bare.agents[id].beliefs, watched.agents[id].beliefs) {
				t.Errorf("seed %d tick %d: agent %d beliefs %v with a logger and an observer, %v without", obsInternalSeed, tick, id, watched.agents[id].beliefs, bare.agents[id].beliefs)
			}
		}
		a, errA := bare.Snapshot()
		b, errB := watched.Snapshot()
		if errA != nil || errB != nil || !reflect.DeepEqual(a, b) {
			t.Errorf("seed %d tick %d: snapshots differ with a logger and an observer (%v, %v)", obsInternalSeed, tick, errA, errB)
		}
		if !reflect.DeepEqual(bare.rnd, watched.rnd) {
			t.Errorf("seed %d tick %d: the random source differs with a logger and an observer", obsInternalSeed, tick)
		}
	}
}

// site is one call of a step that the allocation test measures: a position
// method, or the part of the step before or after the straight line.
type site struct {
	name string
	call func(w *World) error
}

// noError adapts a position with no error result.
func noError(f func(*World)) func(*World) error {
	return func(w *World) error { f(w); return nil }
}

// stepSites is the straight line of a step as a table of sites, in the order of
// positionList: each is measured alone by the test. TestCallSites_TheTableIsTheStraightLine
// keeps it complete.
var stepSites = []site{
	{"P0.2", noError((*World).p02)},
	{"P1.1", (*World).p11Moment},
	{"P1.2", noError((*World).p12Anchors)},
	{"P1.6", (*World).p16Relief},
	{"P1.7", noError((*World).p17Flows)},
	{"P2.1", noError((*World).p21Complete)},
	{"P2.3", noError((*World).p23Snapshot)},
	{"P2.5", (*World).p25Work},
	{"P2.6", noError((*World).p26Ration)},
	{"P2.7", (*World).p27Start},
	{"P2.8a", noError((*World).p28aHalted)},
	{"P3.2", (*World).p32Trade},
	{"P3.4", (*World).p34Estates},
	{"P3.6", noError((*World).p36Aggregates)},
	{"P4.1", (*World).p41Clear},
	{"P5.1", noError((*World).p51Window)},
	{"P5.3", (*World).p53Markdown},
	{"P5.4", noError((*World).p54Outcomes)},
	{"P6.1", (*World).p61Upkeep},
	{"P6.2", (*World).p62Wants},
	{"P6.3", noError((*World).p63Welfare)},
	{"P6.4", noError((*World).p64Shortfall)},
	{"P7.1", (*World).p71Entry},
	{"P8.1", (*World).p81Age},
	{"P8.2", noError((*World).p82Flows)},
	{"P8.4", (*World).p84Audit},
}

// mallocs is the number of heap objects the process has allocated so far.
func mallocs() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Mallocs
}

// measureSites replays steps steps of w by the straight line and returns, for
// the bracket before the line (P0.1 to P0.3), each site and the bracket after it
// (P8.5 and P8.6), the heap objects allocated in all the replays.
func measureSites(t *testing.T, w *World, steps int) map[string]uint64 {
	t.Helper()
	total := map[string]uint64{}
	for range steps {
		before := mallocs()
		if err := w.beginStep(); err != nil {
			t.Fatalf("seed %d: beginStep() error = %v, want nil", obsInternalSeed, err)
		}
		total["P0.1-P0.3"] += mallocs() - before
		var err error
		for _, s := range stepSites {
			before = mallocs()
			err = s.call(w)
			total[s.name] += mallocs() - before
			if err != nil {
				t.Fatalf("seed %d: %s error = %v, want nil", obsInternalSeed, s.name, err)
			}
		}
		before = mallocs()
		if err := w.endStep(nil); err != nil {
			t.Fatalf("seed %d: endStep() error = %v, want nil", obsInternalSeed, err)
		}
		total["P8.5-P8.6"] += mallocs() - before
	}
	return total
}

// TestCallSites_TheTableIsTheStraightLine verifies that the table of sites has
// one entry for each position of the straight line of tick.go, in order, so a
// position added to the line without a row here fails.
//
// Requirement: NFR-OBS-001
func TestCallSites_TheTableIsTheStraightLine(t *testing.T) {
	t.Parallel()
	var want []string
	for _, p := range positionList {
		if p.where == line {
			want = append(want, p.id)
		}
	}
	var got []string
	for _, s := range stepSites {
		got = append(got, s.name)
	}
	if !slices.Equal(got, want) {
		t.Errorf("sites = %v, want the positions of the straight line %v", got, want)
	}
}

// TestCallSites_AllocateNothingWithTheDefaultLoggerAndObserver verifies rules 8.2
// (a): with the default logger and observer, every event-emission and logging call
// site that the tick loop reaches allocates nothing, each measured through its
// unexported position method over 20 replayed steps after 8 steps of warm-up. The
// worlds reach the sites together: the trades, the belief update, the draw and the
// window push, the production phase with its tally, the completion of a job, the
// executor of the spawner's plan with a refused and an unfunded decision
// (NFR-SPN-001), the Warns of rows 2, 3, 6, 8 and 19 and the unfed path. It is not
// parallel: it counts the heap objects of the process.
//
// Arithmetic: a site allocates nothing when its heap objects over the 20 replays,
// divided by 20 and truncated as testing.AllocsPerRun does, is 0.
//
// Requirement: NFR-OBS-001, NFR-BEL-001, NFR-SPN-001
func TestCallSites_AllocateNothingWithTheDefaultLoggerAndObserver(t *testing.T) {
	removeAll := func(v *RegionView) []Decision {
		if v.Tick != 0 {
			return nil
		}
		var d []Decision
		for _, a := range v.Agents {
			d = append(d, Decision{Kind: DecideRemove, Agent: a.Agent, Cause: CauseStarved})
		}
		return d
	}
	// The plan is built once: a literal in the func would allocate in the
	// consumer's code on every call, which is not a call site of the world.
	plan := []Decision{{Kind: DecideEnter, Role: 0, Households: 1, Cause: CauseConsumer}, {Kind: DecideRemove, Agent: 99, Cause: CauseStarved}}
	refused := func(*RegionView) []Decision { return plan }
	// reached says the world did get to the site the case is for, read from the
	// snapshot after the replays; the disposal of an output is asserted by the
	// Warn case "disposed" of the external tests, which has this configuration.
	worlds := []struct {
		name    string
		cfg     func() Config
		plan    func(*RegionView) []Decision
		reached func(Snapshot) bool
	}{
		{"trades and beliefs", allocConfig, refused, func(s Snapshot) bool {
			return slices.ContainsFunc(s.Markets, func(m MarketRow) bool { return m.State.Volume > 0 })
		}},
		{"unfed", func() Config {
			c := obsMill(1, 3)
			c.Roles[0].StartStock = nil
			return c
		}, nil, func(s Snapshot) bool { return s.Roles[0].Unfed > 0 }},
		{"extinct", func() Config { return obsMill(1, 3) }, removeAll, func(s Snapshot) bool { return s.Roles[0].Agents == 0 }},
		{"group below floor", func() Config {
			c := obsMill(1, 6)
			c.World.MinAgents = 3
			c.Groups[0].Scale = 3
			return c
		}, func(v *RegionView) []Decision {
			if v.Tick == 0 {
				return []Decision{{Kind: DecideRemove, Agent: 1, Cause: CauseStarved}}
			}
			return nil
		}, func(s Snapshot) bool { return s.Roles[0].Agents == 2 }},
		{"output disposed", func() Config {
			c := obsMill(1, MaxWeight)
			c.World.Scale = MaxWeight
			c.Roles[0].StartStock = append(c.Roles[0].StartStock, Item{Commodity: "flour", Quantity: MaxQuantity / MaxWeight})
			return c
		}, nil, nil},
	}
	prev := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(prev)
	const warm, runs = 8, 20
	for _, wd := range worlds {
		t.Run(wd.name, func(t *testing.T) {
			w := obsWorld(t, wd.cfg(), &scriptSpawner{plan: wd.plan})
			step(t, w, warm)
			for name, n := range measureSites(t, w, runs) {
				if n/runs != 0 {
					t.Errorf("seed %d: site %s allocated %d heap objects over %d steps, want 0 per step with the default logger and observer", obsInternalSeed, name, n, runs)
				}
			}
			if snap, err := w.Snapshot(); wd.reached != nil && (err != nil || !wd.reached(snap)) {
				t.Errorf("seed %d: the world did not reach the sites of the case (%v), want the condition of %q to hold", obsInternalSeed, err, wd.name)
			}
		})
	}
}
