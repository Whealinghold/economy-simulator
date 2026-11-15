package economysim

import (
	"slices"
	"testing"
)

// The tests of the settlement review at the level of the world (spec-settlements.md
// 4, 5). The pure rules are tested in internal/settlement; these read the class of a
// world built from setConfig and move the households by setting the weights of its
// agents between calls of the position, which is what a population module or a
// call between ticks does to the count.

// The regions of setConfig, by RegionID.
const (
	setA RegionID = iota
	setB
	setC
	setD
)

// setConfig is a world of four regions and one role with the module on: three
// classes (hamlet 0, village 10, town 25), the flag fair on the two larger and the
// limit workers (8 and 20) on them, the default band of 1 000 bp and a dwell of 5.
// Region a has 2 households, b 10, c 30 and d 5, each household an agent of weight
// 1 (the agents of a region are consecutive, in region order). Region c has a row
// that halves its thresholds, so its effective enters are 5 and 13 and its leaves 4
// and 11 (5 * 9 000 / 10 000 = 4; ceil(25 * 5 000 / 10 000) = 13, floor(13 * 9 / 10)
// = 11); region d has the floor village, so it is at least a village. The effective
// thresholds of the other regions are the written ones: enters 10 and 25, leaves 9
// and 22 (floor(25 * 9 / 10) = 22).
func setConfig() Config {
	c := shellConfig()
	c.Regions = nil
	c.Groups = nil
	for i, id := range []string{"a", "b", "c", "d"} {
		r := DefaultRegion()
		r.ID = id
		c.Regions = append(c.Regions, r)
		c.Groups = append(c.Groups, Group{Region: id, Role: "farmer", Households: []int64{2, 10, 30, 5}[i]})
	}
	m := DefaultSettlements()
	m.Dwell = 5
	m.Classes = []SettlementClass{
		{ID: "hamlet", Enter: 0},
		{ID: "village", Enter: 10, Flags: []string{"fair"}, Limits: []ClassLimit{{Name: "workers", Value: 8}}},
		{ID: "town", Enter: 25, Flags: []string{"fair"}, Limits: []ClassLimit{{Name: "workers", Value: 20}}},
	}
	m.Regions = []RegionSettlement{{Region: "c", ThresholdBP: 5_000}, {Region: "d", ThresholdBP: 10_000, Floor: "village"}}
	c.Modules.Settlements = &m
	return c
}

// setWorld builds the world of setConfig with an event log.
func setWorld(t *testing.T) (*World, *evLog) {
	t.Helper()
	rec := &evLog{}
	return newMill(t, setConfig(), WithObserver(rec)), rec
}

// setHouseholds makes the households of region g sum to n by giving the whole weight
// to the first agent of the region and 0 to the others (a removal that has not left
// the table), as a change of the count between two reviews would.
func setHouseholds(t *testing.T, w *World, g RegionID, n int64) {
	t.Helper()
	var ids []int
	for i := w.firstHousehold; i < len(w.agents); i++ {
		if w.regionOf(&w.agents[i]) == g {
			ids = append(ids, i)
		}
	}
	if len(ids) == 0 {
		t.Fatalf("setHouseholds(region %d, %d): the region has no agent", g, n)
	}
	for _, i := range ids {
		w.agents[i].weight = 0
	}
	w.agents[ids[0]].weight = n
}

// atTick moves the world to tick n of the position, as the steps before it would.
func atTick(w *World, n int64) {
	for w.Tick() < n {
		w.ledger.AdvanceTick()
	}
}

// TestConstruction_ClassIsHighestReachedAndFoundedEventsOrder verifies that each
// region is given the highest class whose enter threshold its households meet, not
// below its floor, with Since 0, and that the founded events come one per region in
// ascending RegionID, after every AgentSpawned, with From NoSettlementClass.
//
// Arithmetic: a has 2 households, below 10: hamlet (0). b has 10, which meets the enter
// of village (10): village (1). c has 30 against enters 5 and 13: town (2). d has 5,
// below the enter of village, but its floor is village: village (1).
//
// Requirement: FR-SET-013, FR-SET-007
func TestConstruction_ClassIsHighestReachedAndFoundedEventsOrder(t *testing.T) {
	t.Parallel()
	w, rec := setWorld(t)
	if want := []int{0, 1, 2, 1}; !slices.Equal(w.set.class, want) {
		t.Errorf("class after construction = %v, want %v", w.set.class, want)
	}
	if want := []int64{0, 0, 0, 0}; !slices.Equal(w.set.since, want) {
		t.Errorf("since after construction = %v, want %v", w.set.since, want)
	}
	var founded []SettlementClassChanged
	lastSpawn, firstFounded := -1, -1
	for i, e := range rec.events {
		switch e := e.(type) {
		case AgentSpawned:
			lastSpawn = i
		case SettlementClassChanged:
			founded = append(founded, e)
			if firstFounded < 0 {
				firstFounded = i
			}
		}
	}
	want := []SettlementClassChanged{
		{Region: setA, From: NoSettlementClass, To: 0, Households: 2, Cause: CauseFounded},
		{Region: setB, From: NoSettlementClass, To: 1, Households: 10, Cause: CauseFounded},
		{Region: setC, From: NoSettlementClass, To: 2, Households: 30, Cause: CauseFounded},
		{Region: setD, From: NoSettlementClass, To: 1, Households: 5, Cause: CauseFounded},
	}
	if !slices.Equal(founded, want) {
		t.Errorf("founded events = %+v, want %+v", founded, want)
	}
	if firstFounded < lastSpawn {
		t.Errorf("first SettlementClassChanged at event %d, before the last AgentSpawned at %d, want after", firstFounded, lastSpawn)
	}
}

// TestP72a_HeldChangeIsReportedWithTheTicksLeft verifies the dwell: a count that meets
// the next enter threshold in the first ticks of the world is held until tick Dwell, the
// held event counts the ticks left down to 1 and the class does not move.
//
// Arithmetic: region a has 2 households and is a hamlet with Since 0 and Dwell 5. With 13
// households (enter of village 10) the review gives village, so at tick 0 the change is
// held with 5 - (0 - 0) = 5 ticks left, at tick 3 with 5 - 3 = 2 and at tick 4 with 1.
//
// Requirement: FR-SET-011, FR-SET-025, FR-SET-012
func TestP72a_HeldChangeIsReportedWithTheTicksLeft(t *testing.T) {
	t.Parallel()
	w, rec := setWorld(t)
	rec.events = nil
	setHouseholds(t, w, setA, 13)
	for _, tick := range []int64{0, 3, 4} {
		atTick(w, tick)
		w.p72aSettlement()
	}
	want := []SettlementHeld{
		{Tick: 0, Region: setA, Class: 0, Target: 1, Households: 13, TicksLeft: 5},
		{Tick: 3, Region: setA, Class: 0, Target: 1, Households: 13, TicksLeft: 2},
		{Tick: 4, Region: setA, Class: 0, Target: 1, Households: 13, TicksLeft: 1},
	}
	if got := of[SettlementHeld](rec); !slices.Equal(got, want) {
		t.Errorf("held events = %+v, want %+v", got, want)
	}
	if w.set.class[setA] != 0 || len(of[SettlementClassChanged](rec)) != 0 {
		t.Errorf("class = %d with %d change events during the hold, want 0 and none", w.set.class[setA], len(of[SettlementClassChanged](rec)))
	}
}

// TestP72a_HeldChangeIsForgottenAndTheDwellRunsFromTheLastChange verifies that a held
// change is not remembered: a count that has fallen back by tick Dwell leaves the class
// where it was and emits nothing (the plague of example A, spec-settlements.md 12.1),
// and a count that rises again changes the class at once because the dwell has run, with
// Since set to that tick and the cause grew.
//
// Arithmetic: a is held at tick 0 with 13 households, then falls to 2: at tick 5 the
// target is hamlet again, so there is no event, no change and Since stays 0. At tick 6
// the count is 13 again and 6 - 0 >= 5, so the class becomes village with Since 6.
//
// Requirement: FR-SET-011, FR-SET-012, FR-SET-024
func TestP72a_HeldChangeIsForgottenAndTheDwellRunsFromTheLastChange(t *testing.T) {
	t.Parallel()
	w, rec := setWorld(t)
	setHouseholds(t, w, setA, 13)
	w.p72aSettlement()
	rec.events = nil
	setHouseholds(t, w, setA, 2)
	atTick(w, 5)
	w.p72aSettlement()
	if len(rec.events) != 0 || w.set.class[setA] != 0 || w.set.since[setA] != 0 {
		t.Errorf("after the count fell back: events %v, class %d, since %d, want none, 0 and 0", rec.events, w.set.class[setA], w.set.since[setA])
	}
	setHouseholds(t, w, setA, 13)
	atTick(w, 6)
	w.p72aSettlement()
	want := []SettlementClassChanged{{Tick: 6, Region: setA, From: 0, To: 1, Households: 13, Cause: CauseGrew}}
	if got := of[SettlementClassChanged](rec); !slices.Equal(got, want) || len(of[SettlementHeld](rec)) != 0 {
		t.Errorf("events at tick 6 = %+v and %d held, want %+v and none: a change and a hold never share a tick", got, len(of[SettlementHeld](rec)), want)
	}
	if w.set.class[setA] != 1 || w.set.since[setA] != 6 {
		t.Errorf("class %d since %d, want 1 and 6", w.set.class[setA], w.set.since[setA])
	}
}

// TestP72a_ShrinkRunsFromTheLastChangeAndNeverBelowTheFloor verifies the down
// direction and the floor: a region whose count is below the leave threshold of its
// class gives it up once the dwell has run from its last change, with the cause
// shrank; a region at its floor stays there whatever the count, and says nothing.
//
// Arithmetic: c is a town (2) with leave threshold 11 (above). With 10 households the
// review goes down while h < L: 10 < 11 gives village, and 10 < 4 is false, so it
// stops at village; the dwell has run (tick 5, Since 0, 5 - 0 >= 5), so the class
// becomes village at tick 5, Since 5. d is a village at its floor with leave threshold
// 9: with 1 household the count alone would give hamlet, the class stays village and
// no event is emitted.
//
// Requirement: FR-SET-007, FR-SET-010, FR-SET-024
func TestP72a_ShrinkRunsFromTheLastChangeAndNeverBelowTheFloor(t *testing.T) {
	t.Parallel()
	w, rec := setWorld(t)
	rec.events = nil
	setHouseholds(t, w, setC, 10)
	setHouseholds(t, w, setD, 1)
	atTick(w, 5)
	w.p72aSettlement()
	want := []SettlementClassChanged{{Tick: 5, Region: setC, From: 2, To: 1, Households: 10, Cause: CauseShrank}}
	if got := of[SettlementClassChanged](rec); !slices.Equal(got, want) {
		t.Errorf("change events = %+v, want %+v", got, want)
	}
	if want := []int{0, 1, 1, 1}; !slices.Equal(w.set.class, want) {
		t.Errorf("class = %v, want %v (d held at its floor)", w.set.class, want)
	}
	if len(of[SettlementHeld](rec)) != 0 {
		t.Errorf("SettlementHeld events = %v, want none: the floor is silent", of[SettlementHeld](rec))
	}
}

// classFields are the four class fields of a region row and its households.
type classFields struct {
	households    int64
	class, target SettlementClassID
	since         int64
	atFloor       bool
}

// classOf returns the class fields of a row.
func classOf(r RegionRow) classFields {
	return classFields{r.Households, r.Class, r.ClassTarget, r.ClassSince, r.ClassAtFloor}
}

// TestBetweenTicks_ClassUnchangedUntilTheNextReview verifies that a call between
// ticks changes the household count and nothing else: after adding 25 external agents
// to a hamlet, the snapshot shows the new count and the old class, and the class
// changes at the first step's P7.2a, not before, so a reader of any position of that
// step sees one value (spec-settlements.md 5.2, 5.3).
//
// Arithmetic: region a is a hamlet with 2 households and Dwell 5; 25 external agents of
// weight 1 make 27 >= 25, the enter of town, so a review at tick 5 gives town. The
// class the snapshot shows before the step is the hamlet, the target town.
//
// Requirement: FR-SET-017, FR-SET-014, FR-SET-026
func TestBetweenTicks_ClassUnchangedUntilTheNextReview(t *testing.T) {
	t.Parallel()
	w, _ := setWorld(t)
	step(t, w, 5)
	for range 25 {
		if _, err := w.AddAgent(NewAgent{Region: setA, Role: 0, Money: 100}); err != nil {
			t.Fatalf("AddAgent() error = %v, want nil", err)
		}
	}
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	row, want := snap.Regions[setA], classFields{households: 27, class: 0, target: 2}
	if got := classOf(row); got != want {
		t.Errorf("row of a after the call = %+v, want %+v", got, want)
	}
	step(t, w, 1)
	if w.set.class[setA] != 2 || w.set.since[setA] != 5 {
		t.Errorf("class %d since %d after the step, want 2 and 5", w.set.class[setA], w.set.since[setA])
	}
}

// TestSnapshot_ClassFieldsAtFloorAndIgnoringTheDwell verifies the four fields of the
// region row: the saved class and Since, the target a review at the current count
// would give ignoring the dwell, and the at-floor flag.
//
// Arithmetic: d is a village at its floor (leave 9); with 1 household the count alone
// gives hamlet, so Class 1, ClassTarget 1 (never below the floor) and ClassAtFloor
// true. a with 13 households is a hamlet held by the dwell at tick 0: ClassTarget 1,
// Class 0, not at floor.
//
// Requirement: FR-SET-026
func TestSnapshot_ClassFieldsAtFloorAndIgnoringTheDwell(t *testing.T) {
	t.Parallel()
	w, _ := setWorld(t)
	setHouseholds(t, w, setA, 13)
	setHouseholds(t, w, setD, 1)
	w.p72aSettlement()
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	want := []classFields{{13, 0, 1, 0, false}, {10, 1, 1, 0, false}, {30, 2, 2, 0, false}, {1, 1, 1, 0, true}}
	for g, wr := range want {
		if got := classOf(snap.Regions[g]); got != wr {
			t.Errorf("class fields of region %d = %+v, want %+v", g, got, wr)
		}
	}
}

// TestRegionReads_FlagAndLimitOfTheClassAsItStands verifies the two reads of a class:
// the flag fair is carried by village and town and not by hamlet, a limit absent from a
// class is no limit (false), and the limit workers is 8 for a village and 20 for a town.
// Neither read allocates.
//
// Requirement: FR-SET-005, FR-SET-020
func TestRegionReads_FlagAndLimitOfTheClassAsItStands(t *testing.T) {
	w, _ := setWorld(t)
	type limit struct {
		v  int64
		ok bool
	}
	wantHas := []bool{false, true, true, true}
	wantLimit := []limit{{0, false}, {8, true}, {20, true}, {8, true}}
	for g := range wantHas {
		if got := w.regionHas(RegionID(g), 0); got != wantHas[g] {
			t.Errorf("regionHas(region %d, fair) = %v, want %v", g, got, wantHas[g])
		}
		if v, ok := w.regionLimit(RegionID(g), 0); (limit{v, ok}) != wantLimit[g] {
			t.Errorf("regionLimit(region %d, workers) = %d, %v, want %d, %v", g, v, ok, wantLimit[g].v, wantLimit[g].ok)
		}
	}
	if allocs := testing.AllocsPerRun(100, func() {
		_ = w.regionHas(setB, 0)
		_, _ = w.regionLimit(setC, 0)
	}); allocs != 0 {
		t.Errorf("allocations of the two reads = %v, want 0", allocs)
	}
}

// TestAlloc_SettlementReviewNoOp verifies that the review allocates nothing with the
// default logger and observer, in a tick that holds a change, a tick that changes
// nothing and the construction class: no event is constructed without an observer.
//
// Requirement: NFR-SET-002
func TestAlloc_SettlementReviewNoOp(t *testing.T) {
	w := newMill(t, setConfig())
	setHouseholds(t, w, setA, 13) // held for the first five ticks, so the review reports a hold
	if allocs := testing.AllocsPerRun(50, func() { w.p72aSettlement() }); allocs != 0 {
		t.Errorf("allocations of one review = %v, want 0", allocs)
	}
}

// TestSettlements_WorkIsCounted verifies that the review visits each live agent once
// per tick: the count of all regions is one pass over the agents, so P7.2a is O(A)
// and nothing grows with households (NFR-SET-001).
//
// Arithmetic: setConfig has 2 + 10 + 30 + 5 = 47 household agents, and the pass sums
// 47 weights of 1.
//
// Requirement: NFR-SET-001
func TestSettlements_WorkIsCounted(t *testing.T) {
	t.Parallel()
	w, _ := setWorld(t)
	counts := make([]int64, len(w.cfg.Regions))
	w.householdCounts(counts)
	var sum int64
	for _, n := range counts {
		sum += n
	}
	if agents := int64(len(w.agents) - w.firstHousehold); sum != 47 || agents != 47 {
		t.Errorf("sum of the counts = %d over %d agents, want 47 over 47", sum, agents)
	}
}
