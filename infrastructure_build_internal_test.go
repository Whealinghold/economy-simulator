package economysim

import (
	"testing"
)

// bldMiller is the weight-5 miller of a region (agents in region order: home 1, yonder 2).
func bldMiller(w *World, region RegionID) AgentID { return facAgents(w, region, "miller")[0] }

// TestStart_Limits_Order verifies the facility limits of 4.2.2 in their master order: one case
// per row, each reading the executions the miller (weight 5) could start and the reason. The
// first of the order is the reason: no_facility before no_input, no_input before the build
// reasons; the capacity bounds k0 without refusing; every refusal consumes nothing (the grain
// is read after).
//
// Arithmetic: a mill of full condition has capacity 2 (Durability 1 000, Capacity 2), so one
// mill bounds a weight of 5 to 2, three to min(5, 6) = 5. A mill at 700 misses 300 units and
// the repair gives 100 a execution, so three executions cover it.
//
// Requirement: FR-CAP-012, FR-CAP-013
func TestStart_Limits_Order(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		setup  func(t *testing.T, w *World, m AgentID)
		recipe string
		who    func(w *World, m AgentID) AgentID
		wantK  int64
		want   Reason
	}{
		{"no site: no_facility", nil, "zz_grind", nil, 0, ReasonNoFacility},
		{"one mill bounds k to its capacity", func(t *testing.T, w *World, m AgentID) { bldAdd(t, w, clsMill, m, 1) }, "zz_grind", nil, 2, ""},
		{"three mills do not bound a weight of 5", func(t *testing.T, w *World, m AgentID) { bldAdd(t, w, clsMill, m, 3) }, "zz_grind", nil, 5, ""},
		{"no_facility comes before no_input", func(t *testing.T, w *World, m AgentID) { bldGrain(t, w, m) }, "zz_grind", nil, 0, ReasonNoFacility},
		{"a build with no grain: no_input before facility_cap", func(t *testing.T, w *World, m AgentID) {
			bldAdd(t, w, clsMill, m, MaxFacilitiesPerAgent)
			bldGrain(t, w, m)
		}, "zz_build", nil, 0, ReasonNoInput},
		{"owner at the cap: facility_cap", func(t *testing.T, w *World, m AgentID) { bldAdd(t, w, clsMill, m, MaxFacilitiesPerAgent) }, "zz_build", nil, 0, ReasonFacilityCap},
		{"room for two bounds a build to two", func(t *testing.T, w *World, m AgentID) { bldAdd(t, w, clsMill, m, MaxFacilitiesPerAgent-2) }, "zz_build", nil, 2, ""},
		{"a route toward the region itself has no edge: no_target", nil, "zz_road", func(w *World, m AgentID) AgentID { return bldMiller(w, 1) }, 0, ReasonNoTarget},
		{"a route toward yonder has an edge", nil, "zz_road", nil, 5, ""},
		{"a household with no room cannot own a store: store_refused", nil, "zz_barn", func(w *World, m AgentID) AgentID { return facAgents(w, 0, "weaver")[0] }, 0, ReasonStoreRefused},
		{"nothing worn: nothing_to_repair", func(t *testing.T, w *World, m AgentID) { bldAdd(t, w, clsMill, m, 2) }, "zz_fix", nil, 0, ReasonNothingToRepair},
		{"300 units missing: three executions", func(t *testing.T, w *World, m AgentID) { bldAdd(t, w, clsMill, m, 1, 700) }, "zz_fix", nil, 3, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _, _ := bldWorld(t, func(c *Config) {
				c.Roles[1].StartStock = []Item{{Commodity: "grain", Quantity: 10}}
			})
			m := bldMiller(w, 0)
			if tt.setup != nil {
				tt.setup(t, w, m)
			}
			who := m
			if tt.who != nil {
				who = tt.who(w, m)
			}
			k, reason := bldK0(t, w, who, tt.recipe)
			if k != tt.wantK || reason != tt.want {
				t.Errorf("k0, reason of %s for agent %d = %d, %q, want %d, %q (seed %d)", tt.recipe, who, k, reason, tt.wantK, tt.want, facSeed)
			}
		})
	}
}

// TestStart_CapPerPrincipal verifies that the capacity is a bound per principal and tick shared
// by every start of it: with the capacity of 6 (three mills), requests of 4, 5 and 3 executions
// in turn take 4, 2 and 0 (CAP 17 part 3b), and the third is refused no_facility; the running
// count is cleared at P8.6.
//
// Requirement: FR-CAP-012
func TestStart_CapPerPrincipal(t *testing.T) {
	t.Parallel()
	w, _, _ := bldWorld(t, nil)
	m := bldMiller(w, 0)
	bldAdd(t, w, clsMill, m, 3)
	a := &w.agents[m]
	var got []int64
	var last Reason
	for _, weight := range []int64{4, 5, 3} {
		q := w.startReq(a, 0, bldRecipe(t, w, "zz_grind"), 0)
		q.weight = weight
		k, reason, err := w.prod.maxExecutions(w.ledger, &q)
		if err != nil {
			t.Fatalf("maxExecutions error = %v, want nil", err)
		}
		got, last = append(got, k), reason
		if k > 0 {
			w.infra.started(0, &q, k, 0)
		}
	}
	if want := []int64{4, 2, 0}; got[0] != want[0] || got[1] != want[1] || got[2] != want[2] || last != ReasonNoFacility {
		t.Errorf("executions of three starts against a capacity of 6 = %v, last reason %q, want %v, %q (seed %d)", got, last, want, ReasonNoFacility, facSeed)
	}
	if room := w.infra.facilityRoom(m, 0, clsMill); room != 0 {
		t.Errorf("facilityRoom after the three starts = %d, want 0", room)
	}
	w.infra.clearTick()
	if room := w.infra.facilityRoom(m, 0, clsMill); room != 6 {
		t.Errorf("facilityRoom after the tick is cleared = %d, want the capacity 6", room)
	}
}

// TestStart_SecondEvaluation verifies the running count of builds in a tick (4.2.2, limit 3):
// a principal that holds 62 mills may order two, a second start of the tick finds its room used
// (facility_cap), and with the world's bound used up even a principal with none is refused.
//
// Requirement: FR-CAP-012, FR-CAP-017
func TestStart_SecondEvaluation(t *testing.T) {
	t.Parallel()
	w, _, _ := bldWorld(t, nil)
	m := bldMiller(w, 0)
	bldAdd(t, w, clsMill, m, MaxFacilitiesPerAgent-2)
	k, reason := bldK0(t, w, m, "zz_build")
	if k != 2 || reason != "" {
		t.Fatalf("first build k0, reason = %d, %q, want 2, empty", k, reason)
	}
	a := &w.agents[m]
	q := w.startReq(a, 0, bldRecipe(t, w, "zz_build"), 0)
	w.infra.started(0, &q, 2, 0)
	if k, reason = bldK0(t, w, m, "zz_build"); k != 0 || reason != ReasonFacilityCap {
		t.Errorf("second build k0, reason = %d, %q, want 0, %q after the first ordered its two", k, reason, ReasonFacilityCap)
	}
	w.infra.clearTick()
	other := bldMiller(w, 1)
	w.infra.orderedAll = int64(MaxFacilities - len(w.infra.fac))
	if k, reason = bldK0(t, w, other, "zz_build"); k != 0 || reason != ReasonFacilityCap {
		t.Errorf("build with the world's bound used up k0, reason = %d, %q, want 0, %q", k, reason, ReasonFacilityCap)
	}
}

// TestBuild_YieldIsATest verifies 4.1.3: a sited build recipe with no yield profile for the
// region is refused no_site, and an unsited one, whose factor is positive, is not changed in its
// count by it: k0 is the weight, as for any recipe with no limit.
//
// Requirement: FR-CAP-011
func TestBuild_YieldIsATest(t *testing.T) {
	t.Parallel()
	w, _, _ := bldWorld(t, func(c *Config) {
		for i := range c.Recipes {
			if c.Recipes[i].ID == "zz_build" {
				c.Recipes[i].Sited = true
			}
		}
	})
	m := bldMiller(w, 0)
	if k, reason := bldK0(t, w, m, "zz_build"); k != 0 || reason != ReasonNoSite {
		t.Errorf("sited build with no profile k0, reason = %d, %q, want 0, %q", k, reason, ReasonNoSite)
	}
	if k, reason := bldK0(t, w, m, "zz_road"); k != 5 || reason != "" {
		t.Errorf("unsited build k0, reason = %d, %q, want 5, empty", k, reason)
	}
}

// TestFacilityRoom_ForEMP verifies the helper the hire formation calls: the capacity of the table
// less the running count, never below 0, for a class the principal has none of.
//
// Requirement: FR-CAP-012
func TestFacilityRoom_ForEMP(t *testing.T) {
	t.Parallel()
	w, _, _ := bldWorld(t, nil)
	m := bldMiller(w, 0)
	if got := w.infra.facilityRoom(m, 0, clsMill); got != 0 {
		t.Errorf("facilityRoom with no mill = %d, want 0", got)
	}
	bldAdd(t, w, clsMill, m, 2)
	if got := w.infra.facilityRoom(m, 0, clsMill); got != 4 {
		t.Errorf("facilityRoom with two mills = %d, want 4", got)
	}
	w.infra.used[ownerClass{m, clsMill}] = 9
	if got := w.infra.facilityRoom(m, 0, clsMill); got != 0 {
		t.Errorf("facilityRoom with more used than the capacity = %d, want 0", got)
	}
}

// TestStanding_Reason verifies the derivation of the per-recipe Standing for the saving
// module: the first reason of 4.2.2 other than the inputs and the tools, or empty.
//
// Requirement: FR-CAP-013
func TestStanding_Reason(t *testing.T) {
	t.Parallel()
	w, _, _ := bldWorld(t, nil)
	m := bldMiller(w, 0)
	cases := []struct {
		recipe string
		want   Reason
	}{{"zz_grind", ReasonNoFacility}, {"zz_fix", ReasonNothingToRepair}, {"zz_build", ""}, {"mill", ""}}
	for _, c := range cases {
		if got := w.infra.standingReason(m, 0, bldRecipe(t, w, c.recipe)); got != c.want {
			t.Errorf("standingReason(%s) = %q, want %q", c.recipe, got, c.want)
		}
	}
	bldAdd(t, w, clsMill, m, 1)
	if got := w.infra.standingReason(m, 0, bldRecipe(t, w, "zz_grind")); got != "" {
		t.Errorf("standingReason(zz_grind) with a mill = %q, want empty", got)
	}
	var off *infrastructure
	if got := off.standingReason(m, 0, 0); got != "" {
		t.Errorf("standingReason with the module off = %q, want empty", got)
	}
}
