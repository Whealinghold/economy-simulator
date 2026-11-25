package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The scenarios of the policy module (spec-policy.md 14, spec-reference-chain.md 7.12, plan 3-plc
// Phase 8) run on the pack of the golden, `five` with overlay/policy.toml. They assert what is exact
// (the arithmetic of the schedule, the conservation audit, the equality of two worlds) and report,
// never assert, what the bands of STY will judge (N-027, N-485). A scenario keeps the events of the
// module and a few counters, not snapshots (plan 3-plc, gate economics).

// policyKept is what a scenario run keeps of the events: the module's own and the three slot events
// it moves, in order, and the number of events of every other type.
type policyKept struct {
	decided []economysim.PolicyDecided
	flows   []economysim.MoneyFlowScaled
	transit []economysim.TransitFactorChanged
	edges   []economysim.EdgeStatusChanged
	others  map[string]int
}

// Observe keeps the events of the module and counts the rest.
func (k *policyKept) Observe(e economysim.Event) {
	switch v := e.(type) {
	case economysim.PolicyDecided:
		k.decided = append(k.decided, v)
	case economysim.MoneyFlowScaled:
		k.flows = append(k.flows, v)
	case economysim.TransitFactorChanged:
		k.transit = append(k.transit, v)
	case economysim.EdgeStatusChanged:
		k.edges = append(k.edges, v)
	default:
		k.others[fmt.Sprintf("%T", e)]++
	}
}

// policyWorld builds the world of cfg for seed with the audit on and a policyKept observer.
func policyWorld(t testing.TB, cfg economysim.Config, seed uint64, opts ...economysim.Option) (*economysim.World, *policyKept) {
	t.Helper()
	k := &policyKept{others: map[string]int{}}
	w, err := economysim.NewWorld(cfg, append([]economysim.Option{economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(k)}, opts...)...)
	if err != nil {
		t.Fatalf("seed %d: NewWorld() error = %v, want nil", seed, err)
	}
	return w, k
}

// policyStepTo steps w until its tick is n, calling before(tick) ahead of each step when it is set.
func policyStepTo(t testing.TB, w *economysim.World, n int64, before func(tick int64)) {
	t.Helper()
	for w.Tick() < n {
		if before != nil {
			before(w.Tick())
		}
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", w.Tick(), err)
		}
	}
}

// policyScenario runs fn for the seeds of the scenario as parallel subtests and records the block of
// the pack five under title.
func policyScenario(t *testing.T, title string, seeds int, fn func(t *testing.T, rep *scenReport, seed uint64)) {
	t.Helper()
	rep := newPackScenReport("five", fmt.Sprintf("%s seeds 1-%d", title, seeds))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(seeds) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				fn(t, rep, seed)
			})
		}
	})
	rep.flush(t)
}

// regionRank returns the RegionID of the named region: its rank in the sorted IDs (spec-regions.md 3.1).
func regionRank(cfg economysim.Config, name string) economysim.RegionID {
	ids := make([]string, len(cfg.Regions))
	for i, r := range cfg.Regions {
		ids[i] = r.ID
	}
	slices.Sort(ids)
	return economysim.RegionID(slices.Index(ids, name))
}

// scheduleTicks is the length of S155's run: the actor of the schedule is due at ticks 0 to 250.
const scheduleTicks = 260

// TestScenarioS155_AScheduleIsExact reports PLC-S1 of spec-policy.md 14.1 on the pack of the
// golden, seeds 1 to 10, 260 ticks, audit on: council_a decides every 10 ticks (ActorID 0, bucket 0)
// and holds three levers by a schedule. Every effect is exact: the actor is called 26 times, once
// at each multiple of 10 up to 250; the stipend is scaled at 100 and 200 (the steps of 100 and 200,
// applied at the first due tick at or after each), 10 000 to 15 000 and back; the road from the
// centre to the east at 50 and 250 (15 000, then the default 10 000 that clears the key); and the
// eight keys of the border against group_b at 150 (the factor 20 000), 200 (closed) and 250
// (reopened), one batch each; every one of these events has the cause policy, and no other tick
// has one of council_a's.
//
// Requirement: FR-PLC-043, FR-PLC-050, FR-PLC-051, FR-PLC-061, FR-PLC-080, FR-PLC-085, FR-PLC-001
//
// Guard: a step applied at the wrong tick, an edge written key by key, or a default that left a
// value behind fails here.
func TestScenarioS155_AScheduleIsExact(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, fivePolicy...)
	centre, east := regionRank(cfg, "centre"), regionRank(cfg, "east")
	policyScenario(t, "S155 five+policy schedule, 260 ticks", scenarioSeedCount(), func(t *testing.T, rep *scenReport, seed uint64) {
		w, k := policyWorld(t, cfg, seed)
		policyStepTo(t, w, scheduleTicks, nil)
		c := s155Check{t, rep, seed}
		c.called(k)
		c.stipend(k)
		c.road(k, centre, east)
		c.border(k)
		c.check("every slot event of council_a has the cause policy", allCausePolicy(k), "an event of the module has the cause consumer")
		rep.num(seed, "events of the module", int64(len(k.decided)+len(k.flows)+len(k.transit)+len(k.edges)))
	})
}

// s155Check records the exact checks of one seed of S155.
type s155Check struct {
	t    *testing.T
	rep  *scenReport
	seed uint64
}

// check records one exact check.
func (c s155Check) check(name string, ok bool, format string, args ...any) {
	c.t.Helper()
	checkExact(c.t, c.rep, "S155", "five", c.seed, name, ok, format, args...)
}

// called checks the number of decisions of council_a.
func (c s155Check) called(k *policyKept) {
	c.t.Helper()
	n := 0
	for _, d := range k.decided {
		if d.Actor == 0 {
			n++
		}
	}
	c.check("council_a is called at every multiple of 10 up to 250", n == 26, "%d decisions of actor 0, want 26", n)
}

// stipend checks the two moves of the grant lever.
func (c s155Check) stipend(k *policyKept) {
	c.t.Helper()
	var m []economysim.MoneyFlowScaled
	for _, f := range k.flows {
		if f.Flow == "stipend" {
			m = append(m, f)
		}
	}
	ok := len(m) == 2 && m[0].Tick == 100 && m[0].From == 10_000 && m[0].To == 15_000 && m[1].Tick == 200 && m[1].From == 15_000 && m[1].To == 10_000
	c.check("the stipend moves at 100 and 200", ok, "stipend moves %+v", m)
}

// road checks the two moves of the transit lever from from to to.
func (c s155Check) road(k *policyKept, from, to economysim.RegionID) {
	c.t.Helper()
	var m []economysim.TransitFactorChanged
	for _, f := range k.transit {
		if f.From == from && f.To == to {
			m = append(m, f)
		}
	}
	ok := len(m) == 2 && m[0].Tick == 50 && m[0].AfterBP == 15_000 && m[1].Tick == 250 && m[1].AfterBP == 10_000
	c.check("the road moves at 50 and 250", ok, "road moves %+v", m)
}

// border checks the three batches of the edge group.
func (c s155Check) border(k *policyKept) {
	c.t.Helper()
	byTick := map[int64][]economysim.EdgeStatusChanged{}
	for _, e := range k.edges {
		byTick[e.Tick] = append(byTick[e.Tick], e)
	}
	ticks := make([]int64, 0, len(byTick))
	for tick := range byTick {
		ticks = append(ticks, tick)
	}
	slices.Sort(ticks)
	c.check("the border moves in three batches of eight keys", slices.Equal(ticks, []int64{150, 200, 250}) &&
		len(byTick[150]) == 8 && len(byTick[200]) == 8 && len(byTick[250]) == 8, "edge events at ticks %v", ticks)
	c.check("the border is open at 20 000, then closed, then open", allEdges(byTick[150], false) && allEdges(byTick[200], true) && allEdges(byTick[250], false),
		"closed flags at 150, 200, 250 differ from open, closed, open")
}

// allEdges reports whether every event has the given Closed.
func allEdges(es []economysim.EdgeStatusChanged, closed bool) bool {
	return slices.IndexFunc(es, func(e economysim.EdgeStatusChanged) bool { return e.Closed != closed }) < 0
}

// allCausePolicy reports whether every slot event kept has the cause policy.
func allCausePolicy(k *policyKept) bool {
	return slices.IndexFunc(k.flows, func(e economysim.MoneyFlowScaled) bool { return e.Cause != economysim.CausePolicy }) < 0 &&
		slices.IndexFunc(k.transit, func(e economysim.TransitFactorChanged) bool { return e.Cause != economysim.CausePolicy }) < 0 &&
		slices.IndexFunc(k.edges, func(e economysim.EdgeStatusChanged) bool { return e.Cause != economysim.CausePolicy }) < 0
}
