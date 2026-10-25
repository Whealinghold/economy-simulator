package economysim_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand/v2"
	"os"
	"reflect"
	"slices"
	"sync"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file holds the event tests that need a world (plan-phase-1-11.md Phase 10,
// spec-observability.md 9.1, 9.3 and 9.7): the catalogue is emitted, the order of
// a step, the run lifecycle and the bracketing of a tick, composition, the
// defaults, the pseudo-agents, the silent aggregates and the non-interference of
// the observers. Every scenario is seeded with obsSeed (observe_scenarios_test.go).

// typeName is the name of the type of an event.
func typeName(e economysim.Event) string { return reflect.TypeOf(e).Name() }

// tickOf is the Tick field of an event: the first field of every event.
func tickOf(e economysim.Event) int64 { return reflect.ValueOf(e).FieldByName("Tick").Int() }

// scenarioNamed returns the scenario of the list with the name.
func scenarioNamed(t *testing.T, list []scenario, name string) scenario {
	t.Helper()
	for _, sc := range list {
		if sc.name == name {
			return sc
		}
	}
	t.Fatalf("no scenario named %q", name)
	return scenario{}
}

// emittedBy names, for each event the slice emits, the scenario that emits it
// (plan-phase-1-11.md Phase 10, item 3). The faulty clearing's TickEnded with
// Failed true is its own row in TestScenarios_EmitEveryDeclaredEventOfTheSlice.
var emittedBy = []struct{ event, scenario string }{
	{"WorldCreated", "one"}, {"AgentSpawned", "one"}, {"RunStarted", "one"}, {"RunEnded", "one"},
	{"TickStarted", "one"}, {"TickEnded", "one"}, {"OrderPlaced", "one"}, {"TradeExecuted", "one"},
	{"MarketCleared", "one"}, {"BeliefsUpdated", "one"}, {"RecipeExecuted", "one"},
	{"BoundaryFlow", "one"}, {"WearApplied", "one"}, {"ProductionHalted", "one"},
	{"UpkeepShortfall", "starving"}, {"AgentBankrupted", "starving"}, {"EstateOpened", "starving"},
	{"RecipeRefused", "starving"},
	{"RecipeAbandoned", "spawner script"}, {"WeightChanged", "spawner script"},
	{"RoleChanged", "spawner script"}, {"EntryLimited", "spawner script"},
	{"OrderRejected", "bad order"}, {"WorldRestored", "restored"},
}

// sliceNotEmitted names the declared events that no scenario of the slice emits:
// the staged list of FR-OBS-003. Every other declared event is emitted
// (WorldRestored by the scenario "restored" that emittedTypes adds); an entry that
// a scenario does emit fails the test as stale. AgentRelocated is emitted by no
// scenario because a scenario reaches the world through its public API, and the
// public relocation is INT's (plan 2-17): TestRelocate_EventAndOrder in
// regions_relocate_internal_test.go emits it until then, and plan 17 deletes this
// entry with its scenario.
var sliceNotEmitted = []string{"AgentRelocated"}

// emittedTypes runs the scenarios with an observer and returns, for each, the
// names of the event types it emitted.
func emittedTypes(t *testing.T, list []scenario) map[string]map[string]bool {
	t.Helper()
	seen := map[string]map[string]bool{}
	for _, sc := range list {
		run := play(t, sc, who{observer: true})
		seen[sc.name] = map[string]bool{}
		for _, e := range run.events {
			seen[sc.name][typeName(e)] = true
		}
	}
	seen["restored"] = restoredTypes(t, scenarioNamed(t, list, "one"))
	return seen
}

// restoredTypes checkpoints the world of sc after its ticks and returns the names
// of the event types a restore of it delivers to an observer: the scenario
// "restored" of the staged emission test (plan 2-2 Phase 5).
func restoredTypes(t *testing.T, sc scenario) map[string]bool {
	t.Helper()
	w, _, _ := build(t, sc, who{})
	for range sc.ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("scenario %q (seed %d): Step() error = %v, want nil", sc.name, obsSeed, err)
		}
	}
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("scenario %q (seed %d): Checkpoint() error = %v, want nil", sc.name, obsSeed, err)
	}
	rec := &recorder{}
	opts := []economysim.Option{economysim.WithObserver(rec)}
	if sc.opts != nil {
		opts = append(opts, sc.opts()...)
	}
	if _, err := economysim.RestoreWorld(sc.cfg(t), ck, opts...); err != nil {
		t.Fatalf("scenario %q (seed %d): RestoreWorld() error = %v, want nil", sc.name, obsSeed, err)
	}
	out := map[string]bool{}
	for _, e := range rec.events {
		out[typeName(e)] = true
	}
	return out
}

// checkStagedEmission fails on a declared event that no scenario emitted and that
// sliceNotEmitted does not list, and on a listed event that a scenario emitted.
func checkStagedEmission(t *testing.T, seen map[string]map[string]bool) {
	t.Helper()
	emitted := map[string]bool{}
	for _, m := range seen {
		for name := range m {
			emitted[name] = true
		}
	}
	for _, e := range declaredEvents {
		name, listed := typeName(e), slices.Contains(sliceNotEmitted, typeName(e))
		switch {
		case emitted[name] && listed:
			t.Errorf("sliceNotEmitted lists %s, which a scenario emits: delete the stale entry", name)
		case !emitted[name] && !listed:
			t.Errorf("declared event %s is emitted by no scenario and is not in sliceNotEmitted (seed %d)", name, obsSeed)
		}
	}
}

// TestScenarios_EmitEveryDeclaredEventOfTheSlice verifies that the world emits
// every event declared in the root package at least once over the scenarios, each
// in the scenario that the plan names for it, and that the faulty clearing's step
// ends with a TickEnded that has Failed true; and that sliceNotEmitted fails on a
// stale entry and on a declared event that is neither emitted nor listed.
//
// Requirement: FR-OBS-003
func TestScenarios_EmitEveryDeclaredEventOfTheSlice(t *testing.T) {
	t.Parallel()
	list := scenarios()
	seen := emittedTypes(t, list)
	for _, row := range emittedBy {
		if !seen[row.scenario][row.event] {
			t.Errorf("scenario %q (seed %d) emitted no %s, want at least one", row.scenario, obsSeed, row.event)
		}
	}
	faulty := play(t, scenarioNamed(t, list, "faulty clearing"), who{observer: true})
	if ended := eventsOf[economysim.TickEnded](faulty); len(ended) != 1 || !ended[0].Failed {
		t.Errorf("faulty clearing (seed %d): TickEnded events = %+v, want one with Failed true", obsSeed, ended)
	}
	checkStagedEmission(t, seen)
}

// stepRank is the position of the event types of the slice in the order of one
// step (spec-world.md 6.11), by type name: TickStarted, the recipes and the halts
// of phase 2, the orders of phase 3, the markets of phase 4, the beliefs of phase
// 5, the shortfall of phase 6, the events of the entry step of phase 7 and
// TickEnded. BoundaryFlow and WearApplied are before TickStarted at Tick -1 (the
// bucket) and at P8.2 after, so flowRank gives theirs.
var stepRank = map[string]int{
	"TickStarted": 1, "RecipeExecuted": 2, "ProductionHalted": 3, "RecipeRefused": 4,
	"OrderPlaced": 5, "OrderRejected": 5, "TradeExecuted": 6, "MarketCleared": 6,
	"BeliefsUpdated": 7, "UpkeepShortfall": 8,
	"RecipeAbandoned": 9, "WeightChanged": 9, "RoleChanged": 9, "AgentSpawned": 9,
	"AgentBankrupted": 9, "EstateOpened": 9, "EntryLimited": 9,
	"TickEnded": 11,
}

// rankOf is the rank of an event in its step, or -1 for a type of no rank.
func rankOf(e economysim.Event) int {
	switch e := e.(type) {
	case economysim.BoundaryFlow:
		return flowRank(e.Tick)
	case economysim.WearApplied:
		return flowRank(e.Tick)
	}
	if r, ok := stepRank[typeName(e)]; ok {
		return r
	}
	return -1
}

// marketKey names a market by its region and commodity.
type marketKey [2]int64

// mkey is the key of the market of a region and a commodity.
func mkey(r economysim.RegionID, c economysim.CommodityID) marketKey {
	return marketKey{int64(r), int64(c)}
}

// flowRank is the rank of a BoundaryFlow or a WearApplied: the bucket of
// construction (Tick -1) comes before TickStarted and the flows of a tick at P8.2.
func flowRank(tick int64) int {
	if tick < 0 {
		return 0
	}
	return 10
}

// checkStepOrder checks the events of one step, from TickStarted to TickEnded:
// the ranks never fall, every event carries the tick of the step, the markets
// are in order, a removal's AgentBankrupted is followed at once by its
// EstateOpened, and EntryLimited is the last event of the entry step.
func checkStepOrder(t *testing.T, name string, step []economysim.Event) {
	t.Helper()
	tick, prev := tickOf(step[0]), 0
	for _, e := range step {
		r := rankOf(e)
		if r < 0 || r < prev || tickOf(e) != tick {
			t.Errorf("%s (seed %d) tick %d: %s of rank %d and tick %d after rank %d, want the order of 6.11 and the tick of the step",
				name, obsSeed, tick, typeName(e), r, tickOf(e), prev)
		}
		prev = max(prev, r)
	}
	checkMarketOrder(t, name, tick, step)
	checkEntryOrder(t, name, tick, step)
}

// checkEntryOrder checks the events of the entry step: AgentBankrupted is
// followed at once by its EstateOpened and EntryLimited is the last of the step.
func checkEntryOrder(t *testing.T, name string, tick int64, step []economysim.Event) {
	t.Helper()
	for i, e := range step {
		switch e := e.(type) {
		case economysim.AgentBankrupted:
			if next, ok := step[i+1].(economysim.EstateOpened); !ok || next.Agent != e.Agent {
				t.Errorf("%s (seed %d) tick %d: AgentBankrupted %d is followed by %+v, want its EstateOpened", name, obsSeed, tick, e.Agent, step[i+1])
			}
		case economysim.EntryLimited:
			if rankOf(step[i+1]) == 9 {
				t.Errorf("%s (seed %d) tick %d: %s follows EntryLimited, want it last of the entry step", name, obsSeed, tick, typeName(step[i+1]))
			}
		}
	}
}

// checkMarketOrder checks that the trades of a market come before its
// MarketCleared and that the markets are cleared in ascending (region,
// commodity) order.
func checkMarketOrder(t *testing.T, name string, tick int64, step []economysim.Event) {
	t.Helper()
	last := marketKey{-1, -1}
	cleared := map[marketKey]bool{}
	for _, e := range step {
		switch e := e.(type) {
		case economysim.TradeExecuted:
			if cleared[mkey(e.Region, e.Commodity)] {
				t.Errorf("%s (seed %d) tick %d: a trade of market (%d, %d) comes after its MarketCleared", name, obsSeed, tick, e.Region, e.Commodity)
			}
		case economysim.MarketCleared:
			key := mkey(e.Region, e.Commodity)
			if key[0] < last[0] || (key[0] == last[0] && key[1] <= last[1]) {
				t.Errorf("%s (seed %d) tick %d: market %v cleared after %v, want ascending (region, commodity)", name, obsSeed, tick, key, last)
			}
			cleared[key], last = true, key
		}
	}
}

// splitSteps returns the events of each step, from its TickStarted to its
// TickEnded, and the events before the first TickStarted.
func splitSteps(events []economysim.Event) (before []economysim.Event, steps [][]economysim.Event) {
	for _, e := range events {
		switch e.(type) {
		case economysim.TickStarted:
			steps = append(steps, []economysim.Event{e})
		default:
			if len(steps) == 0 {
				before = append(before, e)
			} else if _, ended := steps[len(steps)-1][len(steps[len(steps)-1])-1].(economysim.TickEnded); !ended {
				steps[len(steps)-1] = append(steps[len(steps)-1], e)
			}
		}
	}
	return before, steps
}

// TestStep_EventsComeInTheOrderOfTheTick verifies, for every step of every
// scenario, the sequence of event types of spec-world.md 6.11 (the ranks above
// and the orders within a rank) and, for the scenario of the consumer spawner,
// that each event comes after the state change it describes: the snapshot taken
// after each step has the agents and households that AgentSpawned,
// AgentBankrupted and WeightChanged add up to, and each market's published price
// and volume equal its last MarketCleared. The construction bucket (Tick -1)
// comes once, before the first TickStarted, and a step's events carry its tick.
//
// Arithmetic: the spawner script starts with 2 carter agents and 3 miller agents
// of weight 2 (5 agents, 8 households); tick 0 reduces agent 3 by 1 household
// (7), switches agent 1 and enters 1 household (the entrant joins an agent of
// the role: 8), tick 1 removes agent 4 of weight 2 (4 agents, 6 households).
//
// Requirement: FR-OBS-022, FR-WLD-053, FR-WLD-054, FR-SPN-032, FR-EST-021
func TestStep_EventsComeInTheOrderOfTheTick(t *testing.T) {
	t.Parallel()
	for _, sc := range scenarios() {
		run := play(t, sc, who{observer: true})
		before, steps := splitSteps(run.events)
		for _, e := range before {
			if _, bucket := e.(economysim.BoundaryFlow); bucket && tickOf(e) != -1 {
				t.Errorf("%s (seed %d): a BoundaryFlow of tick %d before the first TickStarted, want Tick -1", sc.name, obsSeed, tickOf(e))
			}
		}
		if len(steps) == 0 {
			t.Fatalf("%s (seed %d): no step recorded", sc.name, obsSeed)
		}
		for _, step := range steps {
			if _, ended := step[len(step)-1].(economysim.TickEnded); !ended {
				t.Fatalf("%s (seed %d): the step of tick %d does not end with TickEnded", sc.name, obsSeed, tickOf(step[0]))
			}
			checkStepOrder(t, sc.name, step)
		}
	}
}

// eventTally adds up what the events say: the agents and households, and the last
// MarketCleared of each market.
type eventTally struct {
	agents, households int64
	last               map[marketKey]economysim.MarketCleared
}

// addEvent reads one event into the tally.
func addEvent(c *eventTally, e economysim.Event) {
	switch e := e.(type) {
	case economysim.AgentSpawned:
		c.agents, c.households = c.agents+1, c.households+e.Weight
	case economysim.AgentBankrupted:
		c.agents, c.households = c.agents-1, c.households-e.Weight
	case economysim.WeightChanged:
		c.households += e.To - e.From
	case economysim.MarketCleared:
		c.last[mkey(e.Region, e.Commodity)] = e
	}
}

// checkTally compares the snapshot with what the events add up to.
func checkTally(t *testing.T, c *eventTally, name string, snap economysim.Snapshot) {
	t.Helper()
	if snap.Agents != c.agents || snap.Households != c.households {
		t.Errorf("%s (seed %d) after tick %d: snapshot has %d agents and %d households, events add up to %d and %d",
			name, obsSeed, snap.Tick-1, snap.Agents, snap.Households, c.agents, c.households)
	}
	for _, m := range snap.Markets {
		ev := c.last[mkey(m.State.Region, m.State.Commodity)]
		if m.State.Price != ev.Price || m.State.Volume != ev.Volume {
			t.Errorf("%s (seed %d) after tick %d: market (%d, %d) snapshot price %d volume %d, last MarketCleared %d and %d",
				name, obsSeed, snap.Tick-1, m.State.Region, m.State.Commodity, m.State.Price, m.State.Volume, ev.Price, ev.Volume)
		}
	}
}

// TestStep_SnapshotAgreesWithTheEventsAfterEachStep verifies, for the consumer
// spawner scenario, that after every step the snapshot holds what the events so
// far add up to: the agents and households of AgentSpawned, AgentBankrupted and
// WeightChanged, and for each market the price and the volume of its last
// MarketCleared (the event is emitted after the transition it describes).
//
// Requirement: FR-OBS-022, FR-WLD-054
func TestStep_SnapshotAgreesWithTheEventsAfterEachStep(t *testing.T) {
	t.Parallel()
	sc := spawnerScript()
	w, rec, _ := build(t, sc, who{observer: true})
	tally, used := eventTally{last: map[marketKey]economysim.MarketCleared{}}, 0
	for range sc.ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("%s (seed %d): Step() error = %v, want nil", sc.name, obsSeed, err)
		}
		for _, e := range rec.events[used:] {
			addEvent(&tally, e)
		}
		used = len(rec.events)
		checkTally(t, &tally, sc.name, mustSnapshot(t, w))
	}
}

// TestRun_FailedStepEndsTheRunAndAFailedWorldEmitsNothing verifies the run
// lifecycle of a failed step: with the faulty clearing Run emits RunStarted, the
// step's TickStarted and TickEnded with Failed true, and RunEnded with Failed
// true and Completed 0; a Run, a Step and a negative-ticks Run on the failed
// world then emit neither a RunStarted nor a TickStarted and return
// ErrWorldFailed (the guard of P0.1 emits neither bracket event).
//
// Requirement: FR-OBS-012, FR-OBS-013, FR-WLD-006
func TestRun_FailedStepEndsTheRunAndAFailedWorldEmitsNothing(t *testing.T) {
	t.Parallel()
	w, rec, _ := build(t, scenarioNamed(t, scenarios(), "faulty clearing"), who{observer: true})
	rec.forget()
	got, err := w.Run(context.Background(), 5)
	if got != 0 || !errors.Is(err, economysim.ErrInvalidTrade) || w.Err() == nil {
		t.Fatalf("Run(5) = (%d, %v), Err = %v (seed %d), want (0, ErrInvalidTrade) and a failed world", got, err, w.Err(), obsSeed)
	}
	want := []string{"RunStarted ticks=5", "TickStarted 0", "TickEnded 0 failed=true", "RunEnded completed=0 cancelled=false failed=true"}
	if lifecycle := render(rec.events); !slices.Equal(lifecycle, want) {
		t.Errorf("events of the failed Run (seed %d) = %q, want %q", obsSeed, lifecycle, want)
	}
	rec.forget()
	for name, call := range map[string]func() error{
		"Run":      func() error { _, err := w.Run(context.Background(), 1); return err },
		"Negative": func() error { _, err := w.Run(context.Background(), -1); return err },
		"Step":     w.Step,
	} {
		if err := call(); err == nil {
			t.Errorf("%s on a failed world (seed %d) returned nil, want an error", name, obsSeed)
		}
	}
	if len(rec.events) != 0 {
		t.Errorf("calls on a failed world emitted %d events (seed %d), want none", len(rec.events), obsSeed)
	}
}

// panicStrategy is a consumer Strategy whose Work panics.
type panicStrategy struct{ economysim.DefaultStrategy }

// Work panics: a consumer's bug inside a position.
func (panicStrategy) Work(*economysim.AgentView, *rand.Rand) economysim.JobRequest {
	panic("consumer strategy panic")
}

// TestStep_EveryStartedStepEndsAndAPanicEmitsNoTickEnded verifies the bracketing
// of a tick over the scenarios: every TickStarted has a TickEnded, one with
// Failed true only in the scenario of the faulty clearing and only as its last,
// and a panic inside a position (a consumer's Work) emits TickStarted and no
// TickEnded and fails the world.
//
// Requirement: FR-OBS-013, FR-WLD-006, FR-WLD-012
func TestStep_EveryStartedStepEndsAndAPanicEmitsNoTickEnded(t *testing.T) {
	t.Parallel()
	for _, sc := range scenarios() {
		run := play(t, sc, who{observer: true})
		started, ended := eventsOf[economysim.TickStarted](run), eventsOf[economysim.TickEnded](run)
		if len(started) != len(ended) {
			t.Errorf("%s (seed %d): %d TickStarted and %d TickEnded, want equal", sc.name, obsSeed, len(started), len(ended))
		}
		for i, e := range ended {
			if want := sc.name == "faulty clearing" && i == len(ended)-1; e.Failed != want {
				t.Errorf("%s (seed %d): TickEnded %d has Failed %t, want %t", sc.name, obsSeed, e.Tick, e.Failed, want)
			}
		}
	}
	sc := scenario{name: "panic", ticks: 1,
		cfg: func(*testing.T) economysim.Config {
			c := millCfg(1, 2)
			c.Roles[0].Strategy = "panic"
			return c
		},
		opts: func() []economysim.Option {
			return []economysim.Option{economysim.WithStrategy("panic", panicStrategy{})}
		}}
	w, rec, _ := build(t, sc, who{observer: true})
	rec.forget()
	if v := recoverPanic(func() { _ = w.Step() }); v == nil {
		t.Fatalf("Step() (seed %d) did not panic, want the consumer's panic to propagate", obsSeed)
	}
	if got := render(rec.events); !slices.Equal(got, []string{"TickStarted 0"}) || w.Err() == nil {
		t.Errorf("after a panic (seed %d): events %q, Err %v, want only TickStarted and a failed world", obsSeed, got, w.Err())
	}
}

// TestObservers_CompositionOverAScenario verifies, over the consumer spawner
// scenario, that with several WithObserver options every event reaches each
// observer in the order the options were given (the first observer sees an event
// before the second does, event by event), that the two full recorders hold the
// same stream, and that an observer that type-switches on a subset sees exactly
// that subset of the full stream, in order.
//
// Requirement: FR-OBS-024, FR-WLD-003
func TestObservers_CompositionOverAScenario(t *testing.T) {
	t.Parallel()
	sc := spawnerScript()
	var order []int
	a, b := &recorder{}, &recorder{}
	var subset []economysim.Event
	opts := []economysim.Option{
		economysim.WithObserver(economysim.ObserverFunc(func(e economysim.Event) { order = append(order, 1); a.Observe(e) })),
		economysim.WithObserver(economysim.ObserverFunc(func(e economysim.Event) { order = append(order, 2); b.Observe(e) })),
		economysim.WithObserver(economysim.ObserverFunc(func(e economysim.Event) {
			switch e.(type) {
			case economysim.OrderPlaced, economysim.TradeExecuted:
				subset = append(subset, e)
			}
		})),
	}
	w, _, _ := build(t, sc, who{}, opts...)
	for range sc.ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() (seed %d) error = %v, want nil", obsSeed, err)
		}
	}
	if len(a.events) == 0 || !reflect.DeepEqual(a.events, b.events) {
		t.Fatalf("recorders hold %d and %d events (seed %d), want the same non-empty stream", len(a.events), len(b.events), obsSeed)
	}
	for i, who := range order {
		if want := 1 + i%2; who != want {
			t.Fatalf("delivery %d went to observer %d (seed %d), want observer %d: each event reaches 1 then 2", i, who, obsSeed, want)
		}
	}
	var want []economysim.Event
	for _, e := range a.events {
		switch e.(type) {
		case economysim.OrderPlaced, economysim.TradeExecuted:
			want = append(want, e)
		}
	}
	if len(want) == 0 || !reflect.DeepEqual(subset, want) {
		t.Errorf("subset observer saw %d events (seed %d), want the %d OrderPlaced and TradeExecuted of the full stream, in order", len(subset), obsSeed, len(want))
	}
}

// captureOutput runs f with os.Stdout and os.Stderr replaced by pipes and returns
// what it wrote to each. It must not run in parallel with another test that
// writes to either.
func captureOutput(t *testing.T, f func()) (stdout, stderr string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()
	var out, errb bytes.Buffer
	var wg sync.WaitGroup
	pipe := func(dst *bytes.Buffer) *os.File {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("os.Pipe() error = %v, want nil", err)
		}
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = io.Copy(dst, r); _ = r.Close() }()
		return w
	}
	wOut, wErr := pipe(&out), pipe(&errb)
	os.Stdout, os.Stderr = wOut, wErr
	f()
	_ = wOut.Close()
	_ = wErr.Close()
	wg.Wait()
	return out.String(), errb.String()
}

// TestDefaults_ANilLoggerAndObserverWriteNothing verifies that a world given no
// logger and no observer, and one given a nil logger and a nil observer, builds,
// steps, runs and reads its snapshot without a panic and writes nothing to
// standard output or standard error, over the scenarios that remove, enter and
// fail (the paths that would log). It is not parallel: it replaces os.Stdout and
// os.Stderr.
//
// Requirement: FR-OBS-090, FR-WLD-003
func TestDefaults_ANilLoggerAndObserverWriteNothing(t *testing.T) {
	for _, name := range []string{"spawner script", "bad order", "starving"} {
		for _, nilOpts := range []bool{false, true} {
			sc := scenarioNamed(t, scenarios(), name)
			var extra []economysim.Option
			if nilOpts {
				extra = []economysim.Option{economysim.WithLogger(nil), economysim.WithObserver(nil)}
			}
			out, errOut := captureOutput(t, func() {
				w, _, _ := build(t, sc, who{}, extra...)
				if _, err := w.Run(context.Background(), sc.ticks); err != nil {
					t.Errorf("%s (seed %d): Run() error = %v, want nil", name, obsSeed, err)
				}
				if _, err := w.Snapshot(); err != nil {
					t.Errorf("%s (seed %d): Snapshot() error = %v, want nil", name, obsSeed, err)
				}
			})
			if out != "" || errOut != "" {
				t.Errorf("%s with nil options %t (seed %d) wrote %q to stdout and %q to stderr, want nothing", name, nilOpts, obsSeed, out, errOut)
			}
		}
	}
}

// pseudoScenario is the consumer spawner scenario with the recording Strategy
// fake as the strategy of both roles, run for 8 ticks so that the estate holds
// the goods of the removed agent and asks for them.
func pseudoScenario(fake *strategyFake) scenario {
	sc := spawnerScript()
	sc.name, sc.ticks = "pseudo", 8
	cfg, opts := sc.cfg, sc.opts
	sc.cfg = func(t *testing.T) economysim.Config {
		c := cfg(t)
		for i := range c.Roles {
			c.Roles[i].Strategy = "fake"
		}
		return c
	}
	sc.opts = func() []economysim.Option {
		return append(opts(), economysim.WithStrategy("fake", fake))
	}
	return sc
}

// TestPseudoAgents_AreNeverTheSubjectOfAnAgentEvent verifies that the estate of
// the region, agent 0, is no subject of AgentSpawned, AgentBankrupted,
// WeightChanged or RoleChanged and no count includes it (the agents and
// households that the events add up to equal the snapshot's), that a recording
// Strategy fake that wraps the default is never called to Work, Trade or Revise
// for the estate's ID, and that the estate's own orders carry its agent ID and
// are asks.
//
// Arithmetic: the scenario has 1 region, so the estate is agent 0; the households
// are agents 1 to 5, and the removal of agent 4 in tick 1 gives the estate the
// goods it asks for from tick 1 on.
//
// Requirement: FR-OBS-007, FR-EST-011, FR-EST-012, FR-AGT-032
func TestPseudoAgents_AreNeverTheSubjectOfAnAgentEvent(t *testing.T) {
	t.Parallel()
	fake := newStrategyFake(false)
	sc := pseudoScenario(fake)
	run := play(t, sc, who{observer: true})
	if run.err != nil {
		t.Fatalf("%s (seed %d): error = %v, want nil", sc.name, obsSeed, run.err)
	}
	estates := int64(len(run.w.Config().Regions))
	var agents, households int64
	for _, e := range run.events {
		var subject economysim.AgentID = -1
		switch e := e.(type) {
		case economysim.AgentSpawned:
			subject, agents, households = e.Agent, agents+1, households+e.Weight
		case economysim.AgentBankrupted:
			subject, agents, households = e.Agent, agents-1, households-e.Weight
		case economysim.WeightChanged:
			subject, households = e.Agent, households+e.To-e.From
		case economysim.RoleChanged:
			subject = e.Agent
		}
		if subject >= 0 && int64(subject) < estates {
			t.Errorf("%s (seed %d): %s has the estate %d as its subject", sc.name, obsSeed, typeName(e), subject)
		}
	}
	snap, err := run.w.Snapshot()
	if err != nil || snap.Agents != agents || snap.Households != households {
		t.Errorf("snapshot = (%d agents, %d households, %v), events add up to (%d, %d), want equal with no estate counted (seed %d)",
			snap.Agents, snap.Households, err, agents, households, obsSeed)
	}
	for _, calls := range []map[economysim.AgentID]int{fake.work, fake.trade, fake.revise} {
		if len(calls) == 0 {
			t.Errorf("the recording strategy was never called (seed %d), want calls for the households", obsSeed)
		}
		for id := range calls {
			if int64(id) < estates {
				t.Errorf("the strategy was called for the estate %d (seed %d), want only households", id, obsSeed)
			}
		}
	}
	asks := 0
	for _, e := range eventsOf[economysim.OrderPlaced](run) {
		if int64(e.Agent) < estates {
			if e.Side != economysim.Ask {
				t.Errorf("the estate's order %+v is not an ask (seed %d)", e, obsSeed)
			}
			asks++
		}
	}
	if asks == 0 {
		t.Errorf("no OrderPlaced carries the estate's agent ID (seed %d), want the estate's asks after the removal", obsSeed)
	}
}

// quietScenario is a world in which nothing is short: every household holds a
// thousand grain and starts a one-tick job every tick.
func quietScenario() scenario {
	return scenario{name: "quiet", ticks: 12, cfg: func(*testing.T) economysim.Config {
		c := millCfg(1, 4)
		c.Roles[0].StartStock = []economysim.Item{{Commodity: "grain", Quantity: 1_000}}
		return c
	}}
}

// TestAggregates_AreSilentWhenAllTheirCountsAreZero verifies the aggregates that
// the table of spec-observability.md 4.1 marks "never when all are zero": in a
// quiet world (no household is unfed, none is held, rested or refused) there is
// no UpkeepShortfall, ProductionHalted or RecipeRefused in any tick, and in the
// scenarios that do emit them each event has a nonzero count (the three counts of
// a halt are at most the free households, and an UpkeepShortfall has Unfed above 0).
//
// Requirement: FR-OBS-008, FR-OBS-009, FR-OBS-016, FR-OBS-095, FR-OBS-100
func TestAggregates_AreSilentWhenAllTheirCountsAreZero(t *testing.T) {
	t.Parallel()
	quiet := play(t, quietScenario(), who{observer: true})
	if quiet.err != nil || quiet.steps != 12 {
		t.Fatalf("quiet (seed %d): %d steps, error %v, want 12 and nil", obsSeed, quiet.steps, quiet.err)
	}
	for _, e := range quiet.events {
		switch typeName(e) {
		case "UpkeepShortfall", "ProductionHalted", "RecipeRefused":
			t.Errorf("quiet world (seed %d) emitted %+v, want no aggregate while all its counts are 0", obsSeed, e)
		}
	}
	for _, sc := range scenarios() {
		run := play(t, sc, who{observer: true})
		for _, e := range run.events {
			switch e := e.(type) {
			case economysim.UpkeepShortfall:
				if e.Unfed < 1 || e.Households < e.Unfed {
					t.Errorf("%s (seed %d): %+v, want Unfed in 1 to Households", sc.name, obsSeed, e)
				}
			case economysim.ProductionHalted:
				if n := e.Halted + e.Rested + e.NoRecipe; n < 1 || n > e.Free {
					t.Errorf("%s (seed %d): %+v, want Halted + Rested + NoRecipe in 1 to Free", sc.name, obsSeed, e)
				}
			case economysim.RecipeRefused:
				if e.NotOnRole+e.ZeroYield+e.NoSite+e.NoTool+e.NoInput < 1 {
					t.Errorf("%s (seed %d): %+v, want at least one refusal", sc.name, obsSeed, e)
				}
			}
		}
	}
}

// trace is what a run leaves that must not depend on what watched it.
type trace struct {
	snaps  []economysim.Snapshot
	hash   [32]byte
	pcg    []byte
	events []economysim.Event
}

// traceOf steps the world of sc, with the attachments of with and the extra
// options, ticks times (at most) and records the snapshot after each step, the
// configuration hash and the state of the random source.
func traceOf(t *testing.T, sc scenario, ticks int64, with who, extra ...economysim.Option) trace {
	t.Helper()
	w, rec, _ := build(t, sc, with, extra...)
	var tr trace
	for range ticks {
		if err := w.Step(); err != nil {
			break
		}
		snap, err := w.Snapshot()
		if err != nil {
			t.Fatalf("%s (seed %d): Snapshot() error = %v, want nil", sc.name, obsSeed, err)
		}
		tr.snaps = append(tr.snaps, snap)
	}
	tr.hash, tr.pcg, tr.events = w.ConfigHash(), economysim.PCGState(w), rec.events
	return tr
}

// TestObservers_ALoggerAndAnObserverChangeNothing verifies that one seed and one
// configuration give the same snapshot after every tick, the same configuration
// hash and the same random state whether the world has no logger and no observer
// or a logger, an observer and WithAudit, and that the event streams of two
// observed runs, with and without the logger, are equal. This is the logger and
// observer half of FR-WLD-064; the pack matrix is row 12's.
//
// Requirement: FR-OBS-023, FR-WLD-064, FR-BEL-033
func TestObservers_ALoggerAndAnObserverChangeNothing(t *testing.T) {
	t.Parallel()
	for _, sc := range scenarios() {
		ticks := min(sc.ticks, 6)
		bare := traceOf(t, sc, ticks, who{})
		full := traceOf(t, sc, ticks, who{observer: true, logger: true}, economysim.WithAudit())
		watched := traceOf(t, sc, ticks, who{observer: true})
		if len(bare.snaps) == 0 && sc.name != "faulty clearing" {
			t.Errorf("%s (seed %d): no snapshot recorded", sc.name, obsSeed)
		}
		if !reflect.DeepEqual(bare.snaps, full.snaps) || bare.hash != full.hash || !bytes.Equal(bare.pcg, full.pcg) {
			t.Errorf("%s (seed %d): a logger, an observer and WithAudit changed a snapshot, the hash or the random state", sc.name, obsSeed)
		}
		if !reflect.DeepEqual(full.events, watched.events) || len(watched.events) == 0 {
			t.Errorf("%s (seed %d): the event stream differs with a logger (%d events against %d)", sc.name, obsSeed, len(full.events), len(watched.events))
		}
	}
}
