package economysim

import (
	"slices"
	"testing"
)

// bldConds is the conditions of the facilities in the order of the IDs.
func bldConds(w *World, ids []FacilityID) []int64 {
	out := make([]int64, len(ids))
	for i, id := range ids {
		if w.infra.index(id) >= 0 {
			out[i] = facRecord(w, id).condition
		} else {
			out[i] = -1
		}
	}
	return out
}

// TestWear_MostWornFirst verifies 5.1.3: the executions of a start go to the owner's facilities
// of the class most worn first by the condition of the table, each wearing WearPerUse for what
// it serves, and the fall of P8.3b follows.
//
// Arithmetic: the mills are at 500 and 900 of 1 000, so their capacities are ceil(2 * 500 / 1 000)
// = 1 and ceil(2 * 900 / 1 000) = 2, three in all. The request of 3 puts 1 on the worn mill and
// 2 on the other; the wear is 2 an execution, so 500 - 2 = 498 and 900 - 4 = 896, and the decay of
// 1 at P8.3b gives 497 and 895.
//
// Requirement: FR-CAP-019
func TestWear_MostWornFirst(t *testing.T) {
	t.Parallel()
	w, s, _ := bldWorld(t, nil)
	ids := bldAdd(t, w, clsMill, bldMiller(w, 0), 2, 500, 900)
	bldAsk(t, w, s, "zz_grind", 3)
	facStep(t, w)
	if got, want := bldConds(w, ids), []int64{497, 895}; !slices.Equal(got, want) {
		t.Errorf("conditions after a start of 3 and the step = %v, want %v (seed %d)", got, want, facSeed)
	}
}

// TestWear_CapPerFacility verifies that no facility serves more than its cap of the table in a
// tick: a request of 2 on the mills of 500 and 900 puts 1 on the worn one (its cap) and 1 on the
// other, and so 500 - 2 - 1 = 497 and 900 - 2 - 1 = 897, where without the per-facility cap the
// worn one would have taken both.
//
// Requirement: FR-CAP-019
func TestWear_CapPerFacility(t *testing.T) {
	t.Parallel()
	w, s, _ := bldWorld(t, nil)
	ids := bldAdd(t, w, clsMill, bldMiller(w, 0), 2, 500, 900)
	bldAsk(t, w, s, "zz_grind", 2)
	facStep(t, w)
	if got, want := bldConds(w, ids), []int64{497, 897}; !slices.Equal(got, want) {
		t.Errorf("conditions after a start of 2 and the step = %v, want %v (seed %d)", got, want, facSeed)
	}
}

// TestWear_TiesGoToTheLowerID verifies that equal conditions are served in ascending ID: two
// mills at 1 000 with capacity 2 each and a request of 2 wear the first only (2 uses, 4 units).
//
// Requirement: FR-CAP-019
func TestWear_TiesGoToTheLowerID(t *testing.T) {
	t.Parallel()
	w, s, _ := bldWorld(t, nil)
	ids := bldAdd(t, w, clsMill, bldMiller(w, 0), 2)
	bldAsk(t, w, s, "zz_grind", 2)
	facStep(t, w)
	if got, want := bldConds(w, ids), []int64{995, 999}; !slices.Equal(got, want) {
		t.Errorf("conditions after a start of 2 on equal mills = %v, want %v (the first takes both, 1 000 - 4 - 1)", got, want)
	}
}

// TestWear_NotRepeated verifies that the wear is at the start and not for the duration: a job of
// three ticks on one mill wears it once (2 executions, 4 units) and the decay takes 1 a tick, so
// after the three steps 1 000 - 4 - 3 = 993.
//
// Requirement: FR-CAP-019
func TestWear_NotRepeated(t *testing.T) {
	t.Parallel()
	w, s, _ := bldWorld(t, func(c *Config) {
		for i := range c.Recipes {
			if c.Recipes[i].ID == "zz_grind" {
				c.Recipes[i].Duration = 3
			}
		}
	})
	ids := bldAdd(t, w, clsMill, bldMiller(w, 0), 1)
	bldAsk(t, w, s, "zz_grind", 2)
	for range 3 {
		facStep(t, w)
	}
	if got := facRecord(w, ids[0]).condition; got != 993 {
		t.Errorf("condition after a job of 3 ticks = %d, want 993 (one wear of 4 and 3 ticks of decay)", got)
	}
}

// TestWear_SiteServesItsTick verifies that a site that wears to 0 serves the tick it wore out in
// and is ruined at P8.3b with the cause use: a mill at 2 has capacity 1, the start wears it by 2
// to 0, the job of one tick completes (RecipeExecuted) and the ruin follows in the same step.
//
// Requirement: FR-CAP-020
func TestWear_SiteServesItsTick(t *testing.T) {
	t.Parallel()
	w, s, rec := bldWorld(t, nil)
	m := bldMiller(w, 0)
	ids := bldAdd(t, w, clsMill, m, 1, 2)
	bldAsk(t, w, s, "zz_grind", 5)
	facStep(t, w)
	if w.infra.index(ids[0]) >= 0 {
		t.Errorf("the mill worn to 0 is still a record after P8.3b, want it ruined")
	}
	var done int64
	for _, e := range of[RecipeExecuted](rec) {
		if e.Agent == m {
			done += e.Executions
		}
	}
	if done != 1 {
		t.Errorf("executions completed by the owner = %d, want 1 (the site served its tick)", done)
	}
	ruins := of[FacilityChanged](rec)
	last := ruins[len(ruins)-1]
	if last.Kind != FacilityRuined || last.Cause != CauseUse || last.Facility != ids[0] {
		t.Errorf("last FacilityChanged = %+v, want a ruin of %d by use", last, ids[0])
	}
}

// TestEvents_FacilityLimitedCounts verifies the counters of a start under an observer: with no
// mill the request of 5 is refused no_facility (Refused 1, Cut 5); with one mill (capacity 2) it
// starts 2 (Served 2, Cut 3).
//
// Requirement: FR-CAP-038, FR-CAP-013
func TestEvents_FacilityLimitedCounts(t *testing.T) {
	t.Parallel()
	w, s, rec := bldWorld(t, nil)
	bldAsk(t, w, s, "zz_grind", 5)
	facStep(t, w)
	got := of[FacilityLimited](rec)
	if len(got) == 0 || got[0].Class != clsMill || got[0].Refused < 1 || got[0].Cut < 5 || got[0].Served != 0 {
		t.Fatalf("FacilityLimited after a refused start = %+v, want the mill with Refused >= 1, Cut >= 5 and Served 0", got)
	}
	rec.events = nil
	bldAdd(t, w, clsMill, bldMiller(w, 0), 1)
	facStep(t, w)
	served, cut := bldLimited(rec)
	if served != 2 || cut < 3 {
		t.Errorf("FacilityLimited after a start of 2 of 5 = Served %d, Cut %d, want 2 and at least 3 (seed %d)", served, cut, facSeed)
	}
}

// bldLimited sums Served and Cut of the FacilityLimited events of the mill in region 0.
func bldLimited(rec *evLog) (served, cut int64) {
	for _, e := range of[FacilityLimited](rec) {
		if e.Region == 0 && e.Class == clsMill {
			served += e.Served
			cut += e.Cut
		}
	}
	return served, cut
}

// TestInfrastructure_StartAllocsPerRun verifies that the facility part of a start allocates
// nothing: the read of the limits at P2.5, the wear assignment and the running counts at P2.7,
// over the arrays and slices sized at construction. Not parallel (AllocsPerRun) and not skipped
// under the race detector.
//
// Requirement: NFR-CAP-002
func TestInfrastructure_StartAllocsPerRun(t *testing.T) {
	w, _, _ := bldWorld(t, nil)
	m := bldMiller(w, 0)
	bldAdd(t, w, clsMill, m, 3)
	a := &w.agents[m]
	q := w.startReq(a, 0, bldRecipe(t, w, "zz_grind"), 0)
	got := testing.AllocsPerRun(50, func() {
		w.infra.rebuild() // a new tick: the served counts are 0
		if _, _, err := w.prod.maxExecutions(w.ledger, &q); err != nil {
			t.Fatalf("maxExecutions error = %v, want nil", err)
		}
		w.infra.started(0, &q, 2, 0)
		w.infra.clearTick()
	})
	if got != 0 {
		t.Errorf("allocations of the limits, the wear and the counts of a start = %v, want 0", got)
	}
}

// TestEvents_FacilityLimitedCutIsBookedOnTheBindingClass verifies that the executions the
// capacity took off a start are counted once, against the class that bounded the start, not
// against every needed class: a recipe that needs the mill (capacity 2) and the kiln (capacity
// 10), asked for 5 by a household of weight 5, starts 2, so the mill books Served 2 and Cut 3
// and the kiln has no event (a row is emitted only for a non-zero Cut, Refused, Voided or
// Pending, 14.1), where booking the cut on both classes gave it a Cut of 3.
//
// Arithmetic: free of the needs the start would take the weight, 5; the mill allows 2, the kiln
// 10; k = 2 and the cut is 5 - 2 = 3, the mill's.
//
// Requirement: FR-CAP-013, FR-CAP-038
//
// Fix: review of CAP PR 4. Cut was booked on every needed class, so the kiln showed 3
// executions cut that it never limited.
func TestEvents_FacilityLimitedCutIsBookedOnTheBindingClass(t *testing.T) {
	t.Parallel()
	w, s, rec := bldWorld(t, func(c *Config) {
		m := c.Modules.Infrastructure
		m.Classes = append(m.Classes, FacilityClass{ID: "kiln", Kind: "site", Durability: 1_000, Capacity: 10})
		c.Recipes = append(c.Recipes, Recipe{ID: "zz_two", TraineeYield: 5_000, Duration: 1, Inputs: []Item{{Commodity: "grain", Quantity: 1}},
			Needs: []string{"mill", "kiln"}, Outputs: []Output{{Commodity: "flour", Quantity: 2, CostShare: 10_000}}})
		c.Roles[0].Recipes = append(c.Roles[0].Recipes, "zz_two")
	})
	// The configuration sorts the classes, so the IDs are read by name.
	classes := w.cfg.Modules.Infrastructure.Classes
	mill := FacilityClassID(slices.IndexFunc(classes, func(c FacilityClass) bool { return c.ID == "mill" }))
	kiln := FacilityClassID(slices.IndexFunc(classes, func(c FacilityClass) bool { return c.ID == "kiln" }))
	m := bldMiller(w, 0)
	bldAdd(t, w, mill, m, 1)
	bldAdd(t, w, kiln, m, 1)
	bldAsk(t, w, s, "zz_two", 5)
	facStep(t, w)
	var onMill, onKiln FacilityLimited
	for _, e := range of[FacilityLimited](rec) {
		if e.Region != 0 {
			continue
		}
		switch e.Class {
		case mill:
			onMill = e
		case kiln:
			onKiln = e
		}
	}
	if onMill.Served != 2 || onMill.Cut != 3 {
		t.Errorf("FacilityLimited of the mill = Served %d, Cut %d, want 2 and 3 (seed %d)", onMill.Served, onMill.Cut, facSeed)
	}
	if onKiln != (FacilityLimited{}) {
		t.Errorf("FacilityLimited of the kiln = %+v, want none: the kiln did not bind and an event is emitted only for a non-zero Cut, Refused, Voided or Pending (seed %d)", onKiln, facSeed)
	}
}

// TestRefusal_CountsAgainstThePrincipalsClass verifies that the class a no_facility refusal is
// counted against is read from the principal's facilities: the recipe needs the arbor and the
// mill, the principal holds an arbor and no mill, so the class is the mill, while a worker with
// neither is told the first needed class, the arbor.
//
// Requirement: FR-CAP-013
//
// Guard: a hired start is refused for the employer's full class, and World.refuseFor passes the
// principal here; read as the executor, the class would be the first needed one whatever bound.
// The wiring of refuseFor itself has no test, since a hire cannot form without the facility.
func TestRefusal_CountsAgainstThePrincipalsClass(t *testing.T) {
	t.Parallel()
	w, _, _ := bldWorld(t, func(c *Config) {
		m := c.Modules.Infrastructure
		m.Classes = append(m.Classes, FacilityClass{ID: "arbor", Kind: "site", Durability: 1_000, Capacity: 10})
		c.Recipes = append(c.Recipes, Recipe{ID: "zz_two", TraineeYield: 5_000, Duration: 1, Inputs: []Item{{Commodity: "grain", Quantity: 1}},
			Needs: []string{"arbor", "mill"}, Outputs: []Output{{Commodity: "flour", Quantity: 2, CostShare: 10_000}}})
		c.Roles[0].Recipes = append(c.Roles[0].Recipes, "zz_two")
	})
	classes := w.cfg.Modules.Infrastructure.Classes
	arbor := FacilityClassID(slices.IndexFunc(classes, func(c FacilityClass) bool { return c.ID == "arbor" }))
	mill := FacilityClassID(slices.IndexFunc(classes, func(c FacilityClass) bool { return c.ID == "mill" }))
	principal, worker := bldMiller(w, 0), bldMiller(w, 1)
	bldAdd(t, w, arbor, principal, 1)
	w.infra.rebuild()
	recipe := bldRecipe(t, w, "zz_two")
	got, ok := w.infra.refusalClass(principal, 0, recipe, ReasonNoFacility)
	if !ok || got != mill {
		t.Errorf("refusalClass(principal with an arbor and no mill) = %d, %v, want the mill %d (seed %d)", got, ok, mill, facSeed)
	}
	if got, _ := w.infra.refusalClass(worker, 0, recipe, ReasonNoFacility); got != arbor {
		t.Errorf("refusalClass(worker with neither) = %d, want the first needed class %d, the arbor (seed %d)", got, arbor, facSeed)
	}
}
