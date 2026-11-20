package economysim

import (
	"slices"
	"testing"
)

// TestComplete_Build verifies 4.3.1: a build of two ticks completes at P2.1 of the second tick
// and makes k facilities, in ascending new IDs, owned by the principal in its region, each at full
// Durability, with RecipeExecuted first and then one FacilityChanged (built, cause recipe) each; a
// request of 4 by a household of weight 5 makes four (the cohort rule of 4.2.3), the grain of the
// four is spent at the start and not returned.
//
// Requirement: FR-CAP-014
func TestComplete_Build(t *testing.T) {
	t.Parallel()
	w, s, rec := bldWorld(t, nil)
	m := bldMiller(w, 0)
	bldAsk(t, w, s, "zz_build", 4)
	facStep(t, w)
	if n := len(w.infra.fac); n != 0 {
		t.Fatalf("facilities after the first tick of a two-tick build = %d, want 0", n)
	}
	facStep(t, w)
	built := slices.DeleteFunc(of[FacilityChanged](rec), func(e FacilityChanged) bool { return e.Kind != FacilityBuilt || e.Owner != m })
	if len(built) != 4 {
		t.Fatalf("built events = %d, want 4 (k = 4 of a cohort of 5)", len(built))
	}
	for i, e := range built {
		want := FacilityChanged{Tick: e.Tick, Facility: FacilityID(i + 1), Class: clsMill, Region: 0, To: NoRegion, Owner: m,
			Kind: FacilityBuilt, Cause: CauseRecipe, After: 1_000, Effect: 2}
		if e != want {
			t.Errorf("built[%d] = %+v, want %+v", i, e, want)
		}
	}
	if ex, bi := eventIndex(rec, "RecipeExecuted"), eventIndex(rec, "FacilityChanged"); ex < 0 || bi < ex {
		t.Errorf("RecipeExecuted at %d, first FacilityChanged at %d, want RecipeExecuted first", ex, bi)
	}
	if got := facRecord(w, built[0].Facility).condition; got != 999 {
		t.Errorf("condition of a mill built this tick after P8.3b = %d, want 999 (it decays at the end of its tick)", got)
	}
}

// TestComplete_Repair verifies 4.3.2: a repair of k executions restores k * Units to the
// principal's facilities most worn first, each to at most its Durability, events in ascending ID
// with the condition before and after.
//
// Arithmetic: three mills at 500, 990 and 980; the repair gives 100 an execution and the request
// is 3 (k0 is ceil(530 / 100) = 6, the request bounds it), so 300 units go to the worn mill (500
// to 800): it misses 500, which holds them all.
//
// Requirement: FR-CAP-015
func TestComplete_Repair(t *testing.T) {
	t.Parallel()
	w, s, rec := bldWorld(t, nil)
	ids := bldBuildRepairWorld(t, w, s, 3, 500, 990, 980)
	facStep(t, w)
	if got, want := bldConds(w, ids), []int64{799, 989, 979}; !slices.Equal(got, want) {
		t.Errorf("conditions after a repair of 300 units and the decay = %v, want %v (seed %d)", got, want, facSeed)
	}
	var ev []FacilityChanged
	for _, e := range of[FacilityChanged](rec) {
		if e.Kind == FacilityRepaired {
			ev = append(ev, e)
		}
	}
	if len(ev) != 1 || ev[0].Facility != ids[0] || ev[0].Before != 500 || ev[0].After != 800 || ev[0].Cause != CauseRecipe {
		t.Errorf("repaired events = %+v, want one: mill %d from 500 to 800", ev, ids[0])
	}
}

// bldBuildRepairWorld adds one mill per condition for the home miller and asks for the repair.
func bldBuildRepairWorld(t *testing.T, w *World, s *bldScript, k int64, conds ...int64) []FacilityID {
	t.Helper()
	ids := bldAdd(t, w, clsMill, bldMiller(w, 0), len(conds), conds...)
	bldAsk(t, w, s, "zz_fix", k)
	return ids
}

// TestComplete_RepairSpreadsAndVoids verifies the distribution when the units exceed what is
// missing: two mills at 990 and 980 take 10 and 20 (most worn first: the 980 one), the rest is
// lost, and the executions that restored nothing are counted Voided. Written as a job of two
// ticks whose facilities are filled between the start and the completion, so the completion can
// do nothing (4.3.4): it neither fails nor holds the job open.
//
// Requirement: FR-CAP-015, FR-CAP-017
func TestComplete_RepairSpreadsAndVoids(t *testing.T) {
	t.Parallel()
	w, s, rec := bldWorld(t, func(c *Config) {
		for i := range c.Recipes {
			if c.Recipes[i].ID == "zz_fix" {
				c.Recipes[i].Duration = 2
			}
		}
	})
	ids := bldBuildRepairWorld(t, w, s, 5, 990, 980)
	facStep(t, w)
	if err := w.AdjustFacilities([]FacilityAdjustment{{ID: ids[0], Delta: 1_000}, {ID: ids[1], Delta: 1_000}}); err != nil {
		t.Fatalf("AdjustFacilities error = %v, want nil", err)
	}
	facStep(t, w)
	var voided int64
	for _, e := range of[FacilityLimited](rec) {
		voided += e.Voided
	}
	if voided != 1 {
		t.Errorf("Voided after a repair that found nothing to restore = %d, want 1 (k0 was ceil(30 / 100) = 1)", voided)
	}
	if a, err := w.Agent(bldMiller(w, 0)); err != nil || a.HasJob && a.Recipe == bldRecipe(t, w, "zz_fix") && a.Remaining > 1 {
		t.Errorf("the job of a completion that did nothing is held open: %+v, error %v", a, err)
	}
}

// TestComplete_Voided verifies 4.3.4: a build whose room was used meanwhile creates the
// facilities that fit and loses the rest, without failing; the Voided count and the grain are
// as stated: the five grain taken at the start are not returned.
//
// Arithmetic: the build starts for 5 and two ticks; before the completion 62 mills are added, so
// the room at the completion is 64 - 62 = 2 and three executions are lost.
//
// Requirement: FR-CAP-017
func TestComplete_Voided(t *testing.T) {
	t.Parallel()
	w, s, rec := bldWorld(t, nil)
	m := bldMiller(w, 0)
	bldAsk(t, w, s, "zz_build", 5)
	facStep(t, w)
	grain, _ := w.cfg.commodityID("grain")
	before, _, _ := w.ledger.Holding(w.agents[m].account, int(grain))
	bldAdd(t, w, clsMill, m, MaxFacilitiesPerAgent-2)
	facStep(t, w)
	if got := w.infra.held(m); got != MaxFacilitiesPerAgent {
		t.Errorf("facilities held after the completion = %d, want %d (the room of 2 filled)", got, MaxFacilitiesPerAgent)
	}
	var voided int64
	for _, e := range of[FacilityLimited](rec) {
		voided += e.Voided
	}
	if voided != 3 {
		t.Errorf("Voided = %d, want 3 (5 executions, room for 2)", voided)
	}
	if after, _, _ := w.ledger.Holding(w.agents[m].account, int(grain)); after > before {
		t.Errorf("grain after the completion = %d, before = %d, want no refund of the inputs", after, before)
	}
}

// TestBuild_StoreInstalledAtTheNextStep verifies that a store facility built at the completion
// exists with Installed false and gets its store at P8.3b of the same tick (5.4.2): StoreChanged
// first, then FacilityChanged installed.
//
// Requirement: FR-CAP-014, FR-CAP-026
func TestBuild_StoreInstalledAtTheNextStep(t *testing.T) {
	t.Parallel()
	w, s, rec := bldWorld(t, nil)
	bldAsk(t, w, s, "zz_barn", 1)
	facStep(t, w)
	fac := w.infra.fac
	if len(fac) != 2 || !fac[0].installed || !fac[1].installed {
		t.Fatalf("facilities after a one-tick store build by the two millers = %+v, want two installed at P8.3b of their tick", fac)
	}
	want := []string{"F:built", "F:built", "S:added:facility", "F:installed", "S:added:facility", "F:installed"}
	var got []string
	for _, k := range facEvents(rec) {
		if k != "T" && k != "L" {
			got = append(got, k)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("events of the tick = %v, want %v (built at P2.7, then the install at P8.3b with the store's event first)", got, want)
	}
}

// TestBuild_NeedsRegionGate verifies the gate of 8.2 for the build recipe: a class that needs a
// settlement flag the region's class does not carry refuses the start not_unlocked, and the gate
// is read at the start only (a facility that exists is not removed when the flag falls).
//
// Requirement: FR-CAP-032
func TestBuild_NeedsRegionGate(t *testing.T) {
	t.Parallel()
	w, _, _ := bldWorld(t, func(c *Config) {
		c.Groups[1].Households = 3 // home has 8 households, yonder 6
		m := DefaultSettlements()
		m.Classes = []SettlementClass{{ID: "hamlet", Enter: 0}, {ID: "village", Enter: 7, Flags: []string{"fair"}}}
		c.Modules.Settlements = &m
		for i := range c.Modules.Infrastructure.Classes {
			if c.Modules.Infrastructure.Classes[i].ID == "mill" {
				c.Modules.Infrastructure.Classes[i].Needs = "fair"
			}
		}
	})
	home, yonder := bldMiller(w, 0), bldMiller(w, 1)
	if k, reason := bldK0(t, w, home, "zz_build"); k != 5 || reason != "" {
		t.Errorf("build in the region with the flag k0, reason = %d, %q, want 5, empty", k, reason)
	}
	if k, reason := bldK0(t, w, yonder, "zz_build"); k != 0 || reason != ReasonNotUnlocked {
		t.Errorf("build in the region without it k0, reason = %d, %q, want 0, %q", k, reason, ReasonNotUnlocked)
	}
}

// eventIndex is the position of the first event of the named type in the log, or -1.
func eventIndex(rec *evLog, name string) int {
	return slices.IndexFunc(rec.events, func(e Event) bool {
		switch e.(type) {
		case RecipeExecuted:
			return name == "RecipeExecuted"
		case FacilityChanged:
			return name == "FacilityChanged"
		}
		return false
	})
}
