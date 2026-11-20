package economysim

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"testing"
)

// TestView_Facilities verifies 9.1: the Work view carries the agent's own facilities in its
// region in ascending ID (a mill in another region and another agent's mill are not in it), each
// with its condition, durability and, for a site, cap(f) at its condition, and the recipe's
// MaxExecutions is the k0 the world derives, capacity included.
//
// Arithmetic: the mills are at 1 000 and 500 (capacities 2 and 1), so a request for a recipe that
// needs a mill has a k0 of min(5, 3) = 3.
//
// Requirement: FR-CAP-033
func TestView_Facilities(t *testing.T) {
	t.Parallel()
	w, s, _ := bldWorld(t, nil)
	s.seen, s.k0 = map[AgentID][]FacilityView{}, map[AgentID]int64{}
	m := bldMiller(w, 0)
	own := bldAdd(t, w, clsMill, m, 2, 1_000, 500)
	road, err := w.AddFacilities([]NewFacility{{Class: clsRoad, Region: 0, Toward: 1, Owner: m}})
	if err != nil {
		t.Fatalf("AddFacilities(road) error = %v, want nil", err)
	}
	if _, err := w.AddFacilities([]NewFacility{{Class: clsMill, Region: 1, Toward: NoRegion, Owner: bldMiller(w, 1)}}); err != nil {
		t.Fatalf("AddFacilities(yonder mill) error = %v, want nil", err)
	}
	bldAsk(t, w, s, "zz_grind", 5)
	facStep(t, w)
	want := []FacilityView{
		{ID: own[0], Class: clsMill, To: NoRegion, Condition: 1_000, Durability: 1_000, Capacity: 2},
		{ID: own[1], Class: clsMill, To: NoRegion, Condition: 500, Durability: 1_000, Capacity: 1},
		{ID: road[0], Class: clsRoad, To: 1, Condition: 2_000, Durability: 2_000},
	}
	if got := s.seen[m]; !slices.Equal(got, want) {
		t.Errorf("Work view facilities of the owner = %+v, want %+v (seed %d)", got, want, facSeed)
	}
	if got := s.k0[m]; got != 3 {
		t.Errorf("MaxExecutions of the recipe that needs a mill = %d, want 3", got)
	}
	if got := s.seen[bldMiller(w, 1)]; len(got) != 1 || got[0].Capacity != 2 {
		t.Errorf("view of the yonder miller = %+v, want its one mill with capacity 2", got)
	}
}

// TestView_FacilitiesOff verifies that with the module off the view has no Facilities.
//
// Requirement: FR-CAP-033
func TestView_FacilitiesOff(t *testing.T) {
	t.Parallel()
	s := &bldScript{recipe: 0, max: 1, seen: map[AgentID][]FacilityView{}, k0: map[AgentID]int64{}}
	c := skillsConfig()
	c.Roles[0].Strategy = "script"
	w := newMill(t, c, WithSeed(facSeed), WithStrategy("script", s))
	step(t, w, 2)
	if len(s.seen) == 0 {
		t.Fatal("the scripted strategy saw no view, want at least one")
	}
	for id, f := range s.seen {
		if f != nil {
			t.Errorf("agent %d view Facilities = %+v with the module off, want nil", id, f)
		}
	}
}

// TestDefault_BuildsNothing verifies 4.5.1: a world whose roles use the default strategy and
// none a Saver block builds and repairs nothing in sixty ticks, though the role lists the build
// and repair recipes: they have an input and no valued output, so their net value is negative.
//
// Requirement: FR-CAP-018
func TestDefault_BuildsNothing(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newMill(t, bldConfig(func(c *Config) { c.Roles[0].Strategy = "default" }), WithSeed(facSeed), WithAudit(), WithObserver(rec))
	step(t, w, 60)
	for _, e := range of[FacilityChanged](rec) {
		t.Errorf("FacilityChanged %+v under the default strategy, want none", e)
	}
	if n := len(w.infra.fac); n != 0 {
		t.Errorf("facilities after sixty ticks = %d, want 0 (seed %d)", n, facSeed)
	}
}

// TestDirective_Build verifies 4.5.2: the directive of a player names a build recipe on the
// role's list and the player pays (its grain is taken at the start, the facility is its own);
// a recipe the role does not list is ErrNotOnRole.
//
// Requirement: FR-CAP-018
func TestDirective_Build(t *testing.T) {
	t.Parallel()
	w, _, _ := bldWorld(t, nil)
	grain, _ := w.cfg.commodityID("grain")
	id, err := w.AddAgent(NewAgent{Region: 0, Role: 0, Money: 100_000, Stock: []StockItem{{Commodity: grain, Quantity: 10}}})
	if err != nil {
		t.Fatalf("AddAgent error = %v, want nil", err)
	}
	if err := w.Direct(id, Directive{Job: &JobRequest{Recipe: bldRecipe(t, w, "zz_build"), Executions: 1}}); err != nil {
		t.Fatalf("Direct(build) error = %v, want nil", err)
	}
	step(t, w, 2)
	got := w.infra.byOwner[id]
	if len(got) != 1 {
		t.Fatalf("facilities of the player after a two-tick build = %d, want 1", len(got))
	}
	if held, _, _ := w.ledger.Holding(w.agents[id].account, int(grain)); held >= 10 {
		t.Errorf("grain held by the player = %d, want less than the 10 it started with (the input of one execution is taken)", held)
	}
	off, _, _ := bldWorld(t, func(c *Config) { c.Roles[0].Recipes = []string{"mill", "zz_grind"} })
	id2, err := off.AddAgent(NewAgent{Region: 0, Role: 0, Money: 1})
	if err != nil {
		t.Fatalf("AddAgent error = %v, want nil", err)
	}
	if err := off.Direct(id2, Directive{Job: &JobRequest{Recipe: bldRecipe(t, off, "zz_build"), Executions: 1}}); !errors.Is(err, ErrNotOnRole) {
		t.Errorf("Direct(build not on the role) error = %v, want ErrNotOnRole", err)
	}
}

// TestWarmStart_Needs verifies 13.4: a warm-start job of a recipe that needs a site class the
// group has no initial facility of is a *ViabilityError that matches ErrNotViable, and an initial
// row of the group's role in its region lets it through; FacilityDecay lists the classes with a
// Decay or WearPerUse above 0 and is nil with the module off.
//
// Requirement: FR-CAP-040
func TestWarmStart_Needs(t *testing.T) {
	t.Parallel()
	warm := func(c *Config) {
		c.WarmStart = &WarmStart{Groups: []WarmGroup{{Region: "home", Role: "miller", Stock: []Item{{Commodity: "grain", Quantity: 50}},
			Jobs: []JobSpec{{Recipe: "zz_grind", InProgressBP: 10_000}}}}}
	}
	_, err := NewWorld(bldConfig(warm), WithSeed(facSeed), WithStrategy("script", &bldScript{}))
	var ve *ViabilityError
	if !errors.Is(err, ErrNotViable) || !errors.As(err, &ve) || ve.Rule != "facility_missing" || ve.Value != "zz_grind" || ve.Bound != "mill" {
		t.Fatalf("NewWorld with a warm job that needs a mill and none = %v, want a facility_missing ViabilityError for zz_grind and mill", err)
	}
	withRow := func(c *Config) {
		warm(c)
		c.Modules.Infrastructure.Initial = []InitialFacility{{Region: "home", Class: "mill", Role: "miller", Count: 1}}
	}
	if _, err := NewWorld(bldConfig(withRow), WithSeed(facSeed), WithStrategy("script", &bldScript{})); err != nil {
		t.Errorf("NewWorld with an initial mill for the group error = %v, want nil", err)
	}
	c := bldConfig(nil)
	if got := facilityDecay(&c); !slices.Equal(got, []string{"mill"}) {
		t.Errorf("facilityDecay = %v, want [mill] (Decay 1 and WearPerUse 2)", got)
	}
	c.Modules.Infrastructure = nil
	if got := facilityDecay(&c); got != nil {
		t.Errorf("facilityDecay with the module off = %v, want nil", got)
	}
}

// TestFacility_Snapshot verifies the rows of 14.5: one per (region, class) with a facility, the
// count, the sum of the condition in basis points, the effect and the pending stores, and a
// restored world's snapshot equals the original's.
//
// Arithmetic: the mills at 1 000 and 500 of 1 000 are 10 000 + 5 000 basis points, with the
// effects 2 + 1; the barn of the weaver has no room to be installed and is pending.
//
// Requirement: FR-CAP-037
func TestFacility_Snapshot(t *testing.T) {
	t.Parallel()
	w, _, _ := bldWorld(t, nil)
	bldAdd(t, w, clsMill, bldMiller(w, 0), 2, 1_000, 500)
	facPlant(w, clsBarn, 0, facAgents(w, 0, "weaver")[0], 400)
	w.infra.rebuild()
	got := errVal(w.Snapshot()).Facilities
	want := []FacilityRow{
		{Region: 0, Class: clsBarn, Facilities: 1, ConditionBP: 5_000, Pending: 1},
		{Region: 0, Class: clsMill, Facilities: 2, ConditionBP: 15_000, Effect: 3},
	}
	if !slices.Equal(got, want) {
		t.Errorf("Snapshot.Facilities = %+v, want %+v", got, want)
	}
	cp, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint error = %v, want nil", err)
	}
	restored, err := RestoreWorld(w.Config(), cp, WithStrategy("script", &bldScript{}))
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	if back := errVal(restored.Snapshot()).Facilities; !reflect.DeepEqual(back, got) {
		t.Errorf("restored Snapshot.Facilities = %+v, want %+v", back, got)
	}
}

// TestFork_Infrastructure_Build verifies that a checkpoint taken in the middle of a build, with
// a store facility pending, restores to a world that finishes with the original's bytes: the
// same checkpoint, the same snapshot and the same facility events over the next four ticks.
//
// Requirement: FR-CAP-041
func TestFork_Infrastructure_Build(t *testing.T) {
	t.Parallel()
	w, s, rec := bldWorld(t, nil)
	bldAdd(t, w, clsMill, bldMiller(w, 0), 1)
	facPlant(w, clsBarn, 0, facAgents(w, 0, "weaver")[0], 400)
	w.infra.rebuild()
	bldAsk(t, w, s, "zz_build", 3)
	facStep(t, w)
	cp, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint error = %v, want nil", err)
	}
	rec2 := &evLog{}
	s2 := &bldScript{recipe: s.recipe, max: s.max}
	restored, err := RestoreWorld(w.Config(), cp, WithAudit(), WithObserver(rec2), WithStrategy("script", s2))
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	mark := len(rec.events)
	step(t, w, 4)
	step(t, restored, 4)
	a, b := errVal(w.Checkpoint()), errVal(restored.Checkpoint())
	if !bytes.Equal(a.b, b.b) {
		t.Error("checkpoints of the original and the restored world differ after four ticks")
	}
	if !reflect.DeepEqual(errVal(w.Snapshot()), errVal(restored.Snapshot())) {
		t.Error("snapshots of the original and the restored world differ after four ticks")
	}
	if !reflect.DeepEqual(facilityEvents(rec.events[mark:]), facilityEvents(rec2.events)) {
		t.Errorf("facility events differ: original %v, restored %v", facilityEvents(rec.events[mark:]), facilityEvents(rec2.events))
	}
}

// facilityEvents is the FacilityChanged and FacilityLimited events of the log, in order.
func facilityEvents(events []Event) []Event {
	return slices.DeleteFunc(slices.Clone(events), func(e Event) bool {
		switch e.(type) {
		case FacilityChanged, FacilityLimited:
			return false
		}
		return true
	})
}

// TestFacility_MetricsDerivable verifies 14.4: the five metrics are derivable from the events
// and the snapshot with integers only. The count by (class, region) folded from FacilityChanged
// (+1 added and built, -1 ruined) equals the snapshot's rows; the limited and refused counters
// are the sums of FacilityLimited; the pending and condition gauges are the snapshot's rows,
// which a restored world's snapshot seeds equal.
//
// Requirement: FR-CAP-038, FR-CAP-041
func TestFacility_MetricsDerivable(t *testing.T) {
	t.Parallel()
	w, s, rec := bldWorld(t, nil)
	bldAdd(t, w, clsMill, bldMiller(w, 0), 2, 500, 900)
	bldAsk(t, w, s, "zz_grind", 3)
	for range 3 {
		facStep(t, w)
	}
	bldAsk(t, w, s, "zz_build", 2)
	for range 3 {
		facStep(t, w)
	}
	checkFoldedCounts(t, rec, errVal(w.Snapshot()).Facilities)
	var cut, refused int64
	for _, e := range of[FacilityLimited](rec) {
		if e.Region == 1 && e.Class == clsMill {
			cut, refused = cut+e.Cut, refused+e.Refused
		}
	}
	if refused < 3 || cut < 9 {
		t.Errorf("limited counters of the yonder mill from FacilityLimited = cut %d, refused %d, want at least 9 and 3 (three ticks of a request of 3 with no mill)", cut, refused)
	}
}

// checkFoldedCounts folds the count by (class, region) from FacilityChanged (+1 added and built,
// -1 ruined) and compares it to the rows of the snapshot, both ways.
func checkFoldedCounts(t *testing.T, rec *evLog, rows []FacilityRow) {
	t.Helper()
	folded := map[[2]int64]int64{}
	for _, e := range of[FacilityChanged](rec) {
		k := [2]int64{int64(e.Class), int64(e.Region)}
		switch e.Kind {
		case FacilityAdded, FacilityBuilt:
			folded[k]++
		case FacilityRuined:
			folded[k]--
		}
	}
	for _, r := range rows {
		k := [2]int64{int64(r.Class), int64(r.Region)}
		if folded[k] != r.Facilities {
			t.Errorf("count of class %d in region %d from the events = %d, snapshot row = %d", r.Class, r.Region, folded[k], r.Facilities)
		}
		delete(folded, k)
	}
	for k, n := range folded {
		if n != 0 {
			t.Errorf("events leave %d facilities of class %d in region %d, the snapshot has no row", n, k[0], k[1])
		}
	}
}
