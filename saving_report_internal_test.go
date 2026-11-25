package economysim

import (
	"testing"
)

// savingRun runs the world of richMiller for n ticks with an observer and returns the events
// of the module, the world and the home miller. The default strategy runs the saver role.
func savingRun(t testing.TB, n int, opts ...Option) ([]SavingPlanned, *World, AgentID) {
	t.Helper()
	ev := &evLog{}
	w, m := richMiller(t, nil, append([]Option{WithObserver(ev)}, opts...)...)
	for range n {
		facStep(t, w)
	}
	return of[SavingPlanned](ev), w, m
}

// TestEvent_SavingPlannedTriggers verifies the event of 12.1 end to end: it is emitted for
// the miller role, which has a block, in each region where a count is above 0, never for the
// weaver role, which has none, and never with every triggering count 0; the events of a tick
// come in ascending region then role order.
//
// Arithmetic: the two millers have weight 5, so a count is 5 for each household that is in it;
// the home miller is building at tick 0 (the default asked for the build at P2.5, so Builds is
// 1 and the keep is its goal, 20 000) and the yonder miller has a project that passes and is
// not reachable (its ring is empty), so Unreachable is 5.
//
// Requirement: FR-SAV-029
func TestEvent_SavingPlannedTriggers(t *testing.T) {
	t.Parallel()
	evs, _, _ := savingRun(t, 8)
	if len(evs) == 0 {
		t.Fatalf("SavingPlanned events over 8 ticks = 0, want some (seed %d)", facSeed)
	}
	first := evs[0]
	want := SavingPlanned{Tick: 0, Region: 0, Role: 0, Households: 5, Goal: 20_000, Saving: 5, Building: 5, Builds: 1}
	if first != want {
		t.Errorf("first SavingPlanned = %+v, want %+v", first, want)
	}
	if got := evs[1]; got.Tick != 0 || got.Region != 1 || got.Unreachable != 5 || got.Households != 5 {
		t.Errorf("second SavingPlanned = %+v, want tick 0, region 1, 5 households unreachable", got)
	}
	for i := range evs {
		checkSavingEvent(t, evs, i)
	}
}

// checkSavingEvent checks one event of evs against the trigger, role and order rules of
// TestEvent_SavingPlannedTriggers.
func checkSavingEvent(t *testing.T, evs []SavingPlanned, i int) {
	t.Helper()
	e := evs[i]
	q := savingCount{
		saving: e.Saving, buffered: e.Buffered, passed: e.Passed, building: e.Building, noReturn: e.NoReturn,
		standing: e.Standing, unreachable: e.Unreachable, survival: e.Survival, builds: e.Builds, repairs: e.Repairs,
	}
	if q.quiet() {
		t.Errorf("event %d %+v has every triggering count 0, want it not emitted", i, e)
	}
	if e.Role != 0 {
		t.Errorf("event %d is for role %d, want only role 0 (the weaver has no block)", i, e.Role)
	}
	if i > 0 && (evs[i-1].Tick > e.Tick || evs[i-1].Tick == e.Tick && evs[i-1].Region >= e.Region) {
		t.Errorf("events %d and %d out of order: %+v then %+v, want ascending (tick, region)", i-1, i, evs[i-1], e)
	}
}

// TestEvent_OneBarPerHousehold verifies that a household is in at most one of Survival,
// Building, Passed, Unreachable, NoReturn and Standing, that the counts are households (the
// sum of weights) and so are at most the households given a view, and that Funded and
// Committed are parts of Passed (FR-SAV-030).
//
// Requirement: FR-SAV-030
func TestEvent_OneBarPerHousehold(t *testing.T) {
	t.Parallel()
	evs, _, _ := savingRun(t, 12)
	for i, e := range evs {
		sum := e.Survival + e.Building + e.Passed + e.Unreachable + e.NoReturn + e.Standing
		if sum > e.Households {
			t.Errorf("event %d %+v: exclusive counts sum to %d, want at most the %d households", i, e, sum, e.Households)
		}
		if e.Funded > e.Passed || e.Committed > e.Passed {
			t.Errorf("event %d %+v: Funded or Committed above Passed, want parts of it", i, e)
		}
		for _, c := range []int64{e.Households, e.Saving, e.Buffered, e.Passed, e.Building, e.Unreachable, e.NoReturn} {
			if c%5 != 0 {
				t.Errorf("event %d %+v: count %d is not a multiple of the weight 5, want households and never agents", i, e, c)
			}
		}
	}
}

// TestEvent_FundedAndCommittedRideAlong verifies that the miller of tick 1, which holds the
// inputs of its next mill (the first mill took one set of grain and the stock is large), is
// Committed as well as Passed, and that a household that is Passed and not funded for the
// outlay is not Funded.
//
// Arithmetic: at tick 1 the saver has no job (the first build completed at P2.1 of tick 1),
// 5 households pass the test, Committed is 5 because the held value of the inputs exceeds the
// commit threshold, and the money, 17 000 or so, is below the floor plus the outlay, so Funded
// is 0 (the probe of the query shows the same figures, TestQuery_SavingState).
//
// Requirement: FR-SAV-029, FR-SAV-030
func TestEvent_FundedAndCommittedRideAlong(t *testing.T) {
	t.Parallel()
	evs, _, _ := savingRun(t, 3)
	var got *SavingPlanned
	for i := range evs {
		if evs[i].Tick == 1 && evs[i].Region == 0 {
			got = &evs[i]
		}
	}
	if got == nil {
		t.Fatalf("no SavingPlanned for region 0 at tick 1 in %+v", evs)
	}
	if got.Passed != 5 || got.Committed != 5 || got.Funded != 0 || got.Goal != 25_000 {
		t.Errorf("tick 1 event = %+v, want Passed 5, Committed 5, Funded 0, Goal 25000", *got)
	}
}

// TestEvent_BuildsCountTheRequests verifies that Builds counts the requests Work returned:
// over the run the default asks for three builds, one for each mill the miller ends with, and
// no repair.
//
// Requirement: FR-SAV-030
func TestEvent_BuildsCountTheRequests(t *testing.T) {
	t.Parallel()
	evs, w, m := savingRun(t, 14)
	var builds, repairs int64
	for _, e := range evs {
		builds, repairs = builds+e.Builds, repairs+e.Repairs
	}
	if want := int64(ownedMills(w, m)); builds != want || builds != 3 {
		t.Errorf("sum of Builds over 14 ticks = %d, want %d (the mills built, 3)", builds, want)
	}
	if repairs != 0 {
		t.Errorf("sum of Repairs over 14 ticks = %d, want 0 (nothing decays in this pack)", repairs)
	}
}

// TestEvent_CountsWhateverTheStrategy verifies that the counts are the world's from the pure
// rules and not the strategy's: a role that runs an inert strategy (no job, no order) has the
// same Households, Passed or bars as the default's role at tick 0, and no Builds.
//
// Requirement: FR-SAV-030
func TestEvent_CountsWhateverTheStrategy(t *testing.T) {
	t.Parallel()
	ev := &evLog{}
	w, _ := richMiller(t, func(c *Config) { c.Roles[0].Strategy = "script" }, WithObserver(ev), WithStrategy("script", &shipScript{}))
	facStep(t, w)
	got := of[SavingPlanned](ev)
	if len(got) == 0 {
		t.Fatalf("SavingPlanned events with an inert strategy = 0, want the counts of the world")
	}
	if got[0].Households != 5 || got[0].Builds != 0 || got[0].Repairs != 0 {
		t.Errorf("first event with an inert strategy = %+v, want 5 households and no request", got[0])
	}
	if got[0].Passed+got[0].Unreachable+got[0].NoReturn+got[0].Standing+got[0].Survival == 0 {
		t.Errorf("first event with an inert strategy = %+v, want the household in one of the bars or Passed", got[0])
	}
}

// TestEvent_NoCountsWithoutObserver verifies that the counts exist only under the observer
// flag (observability-and-logging.md 4.5): with no observer the scratch is nil and the
// counting calls build nothing, and the counts are all 0 after a step either way (they are
// cleared at P8.6).
//
// Requirement: FR-SAV-038
func TestEvent_NoCountsWithoutObserver(t *testing.T) {
	t.Parallel()
	off, _ := richMiller(t, nil)
	facStep(t, off)
	if off.saving.counts != nil {
		t.Errorf("counts without an observer = %v, want nil", off.saving.counts)
	}
	_, on, _ := savingRun(t, 3)
	for cell, c := range on.saving.counts {
		if c != (savingCount{}) {
			t.Errorf("counts of cell %d after a step = %+v, want cleared at P8.6", cell, c)
		}
	}
}

// TestCount_PrecedenceAndWeight verifies the rule of one count for each household at the
// lowest level: the weight is what is counted, a household with no project on its list is in
// no bar, and the precedence is survival, building, passed, unreachable, no return, standing.
//
// Arithmetic: a floor of 150 over a reserve of 90 is a buffer of 60; a project of outlay 600
// held for 0 has a goal of 600; the keep of a job in progress is its goal.
//
// Requirement: FR-SAV-030
func TestCount_PrecedenceAndWeight(t *testing.T) {
	t.Parallel()
	choice := ProjectChoice{Outlay: 600, Funded: true, Committed: true}
	tests := []struct {
		name string
		pk   savingPick
		want savingCount
	}{
		{"no project on the list is in none", savingPick{first: -1, floor: 90, reserve: 90, bar: BarSurvival},
			savingCount{households: 3}},
		{"survival wins", savingPick{first: 0, floor: 90, reserve: 90, bar: BarSurvival, building: true, chosen: true},
			savingCount{households: 3, survival: 3}},
		{"building before passed", savingPick{first: 0, floor: 90, reserve: 90, building: true, keep: 200},
			savingCount{households: 3, building: 3, saving: 3, goal: 200}},
		{"passed funded and committed", savingPick{first: 0, floor: 90, reserve: 90, chosen: true, choice: choice},
			savingCount{households: 3, passed: 3, funded: 3, committed: 3, saving: 3, goal: 600}},
		{"unreachable", savingPick{first: 0, floor: 90, reserve: 90, bar: BarUnreachable},
			savingCount{households: 3, unreachable: 3}},
		{"no return with a buffer above the reserve", savingPick{first: 0, floor: 150, reserve: 90, bar: BarNoReturn},
			savingCount{households: 3, noReturn: 3, buffered: 3, buffer: 60}},
		{"standing", savingPick{first: 0, floor: 90, reserve: 90, bar: BarStanding},
			savingCount{households: 3, standing: 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got savingCount
			got.add(&AgentView{Weight: 3}, &tt.pk)
			if got != tt.want {
				t.Errorf("add(weight 3, %+v) = %+v, want %+v", tt.pk, got, tt.want)
			}
		})
	}
}
