package economysim

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"
)

// The recipes of the own-use work fixture: the role's market recipe is 0 (gather,
// item) and the own-use recipes are 5 and 6, forage and forage2, both making
// staple.
const (
	rForage  RecipeID = 5
	rForage2 RecipeID = 6
)

// ownView is workView in the own-use mode: the household eats 3 staple a tick
// and holds 10 (Cover 3), the exit cover is 6 and the retry is off. Besides the
// market recipe it has the own-use recipe forage that makes 3 staple a tick, so
// Cov(forage, staple) = 3 * 10 000 / (1 * 3) = 10 000. Its ratio is above its
// target, so out of the mode it rests.
func ownView() *AgentView {
	v := workView()
	v.Ratio = 20_000
	v.OwnUse, v.OwnUseJobs, v.Cover, v.Secure, v.ExitTicks = true, 0, 3, 3, 6
	v.Recipes = append(v.Recipes, forageRecipe(rForage, 3, 1))
	return v
}

// forageRecipe is an own-use recipe of the fixture: duration d, q staple an
// execution at the yield 10 000, one execution possible.
func forageRecipe(id RecipeID, q, d int64) RecipeView {
	return RecipeView{
		Recipe: id, Duration: d, Effective: 10_000, MaxExecutions: 1, OwnUse: true,
		Outputs: []RecipeOutput{{Commodity: cStaple, Quantity: q, CostShare: 10_000}},
	}
}

// reportOf returns the work report of v.
func reportOf(v *AgentView) workReportBuf {
	buf := workReportBuf{noList: true}
	workReport(v, &buf)
	return buf
}

// TestWork_ModeShareIs10000 verifies that an agent in the mode works with the
// share 10 000 whatever its ratio says, and requests the own-use recipe: the
// household's ratio is above its target (it would rest out of the mode).
//
// Requirement: FR-SUB-019
func TestWork_ModeShareIs10000(t *testing.T) {
	t.Parallel()
	v := ownView()
	r := reportOf(v)
	if r.share != wholeBP || r.rest || r.chosen < 0 || v.Recipes[r.chosen].Recipe != rForage {
		t.Errorf("report in the mode = share %d, rest %v, chosen %d, want share 10000 and the recipe forage", r.share, r.rest, r.chosen)
	}
	v.OwnUse = false
	if r := reportOf(v); !r.rest || r.chosen >= 0 {
		t.Errorf("report out of the mode = rest %v, chosen %d, want a rest and no request", r.rest, r.chosen)
	}
}

// TestWork_NotInModeNeverOwnUse verifies that an own-use recipe is never
// requested by an agent out of the mode, even when it is the only recipe that
// could run and the agent is hungry.
//
// Requirement: FR-SUB-048
func TestWork_NotInModeNeverOwnUse(t *testing.T) {
	t.Parallel()
	v := ownView()
	v.OwnUse, v.Hunger, v.Ratio = false, 2, 1_000
	v.Recipes[0].MaxExecutions = 0 // the market recipe cannot run
	r := reportOf(v)
	if r.chosen >= 0 {
		t.Errorf("chosen = %d (recipe %d), want none out of the mode", r.chosen, v.Recipes[r.chosen].Recipe)
	}
	if r.reason != ReasonNoRecipe && r.reason == "" {
		t.Errorf("reason = %q, want a decline reason", r.reason)
	}
	if got := (DefaultStrategy{}).Work(v, nil); !got.NoJob {
		t.Errorf("Work out of the mode = %+v, want no job", got)
	}
}

// TestWork_PicksBestCoverage verifies that the own-use recipe with the greatest
// coverage of the short goods is requested, ties to the lower recipe ID, and a
// recipe that makes nothing the household is short of is not requested.
//
// Arithmetic: forage makes 3 a tick, Cov = 3 * 10 000 / (1 * 3) = 10 000;
// forage2 makes 6, Cov = 20 000; a forage2 of duration 2 has Cov = 6 * 10 000 /
// (2 * 3) = 10 000, a tie with forage, which the lower ID wins.
//
// Requirement: FR-SUB-020
func TestWork_PicksBestCoverage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		edit func(v *AgentView)
		want RecipeID
		none bool
	}{
		{"the greater coverage", func(v *AgentView) { v.Recipes = append(v.Recipes, forageRecipe(rForage2, 6, 1)) }, rForage2, false},
		{"a tie goes to the lower ID", func(v *AgentView) { v.Recipes = append(v.Recipes, forageRecipe(rForage2, 6, 2)) }, rForage, false},
		{"a recipe that cannot run is passed over", func(v *AgentView) {
			v.Recipes = append(v.Recipes, forageRecipe(rForage2, 6, 1))
			v.Recipes[len(v.Recipes)-1].MaxExecutions = 0
		}, rForage, false},
		{"cover at exit leaves nothing short", func(v *AgentView) { v.Commodities[cStaple].Stock = 18 }, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := ownView()
			tt.edit(v)
			idx, _, ok := v.ownChoice()
			if tt.none {
				if ok {
					t.Errorf("ownChoice() = recipe %d, want none", v.Recipes[idx].Recipe)
				}
				return
			}
			if !ok || v.Recipes[idx].Recipe != tt.want {
				t.Errorf("ownChoice() = (%d, ok %v), want recipe %d", idx, ok, tt.want)
			}
		})
	}
}

// TestWork_RampForCohort verifies the executions of a cohort: k = min(w,
// MaxExecutions, max(1, ceil(w * (Exit - Cover) / Exit))).
//
// Arithmetic: w 10, Exit 6, Cover 2: ceil(10 * 4 / 6) = 7; MaxExecutions 10 and
// weight 10 do not cut it. A MaxExecutions of 5 cuts it to 5. A weight-1 agent
// runs 1.
//
// Requirement: FR-SUB-021
func TestWork_RampForCohort(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name       string
		weight     int64
		cover, max int64
		want       int64
	}{
		{"a cohort puts in its short share", 10, 2, 10, 7},
		{"the executions are cut to what can run", 10, 2, 5, 5},
		{"a weight one agent runs one", 1, 2, 1, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := ownView()
			v.Weight, v.Cover, v.Recipes[1].MaxExecutions = tt.weight, tt.cover, tt.max
			v.Commodities[cStaple].Stock = 3 * tt.weight * tt.cover // Cover = floor(stock / (weight * 3))
			if r := reportOf(v); r.executions != tt.want {
				t.Errorf("executions = %d, want %d", r.executions, tt.want)
			}
		})
	}
}

// TestWork_RetryChoosesRoleRecipe verifies the retry: with RetryJobs 2 and two
// own-use jobs started, the role recipe the normal decision chooses is
// requested instead and the household is counted as retried; one job short of
// the count, the own-use recipe goes on.
//
// Requirement: FR-SUB-022
func TestWork_RetryChoosesRoleRecipe(t *testing.T) {
	t.Parallel()
	v := ownView()
	v.Ratio, v.RetryJobs = 1_000, 2 // the household wants to work, so the role recipe is eligible
	v.OwnUseJobs = 1
	if r := reportOf(v); r.ownRetried || v.Recipes[r.chosen].Recipe != rForage {
		t.Errorf("one job in: retried %v, recipe %d, want no retry and forage", r.ownRetried, v.Recipes[r.chosen].Recipe)
	}
	v.OwnUseJobs = 2
	r := reportOf(v)
	if !r.ownRetried || r.chosen < 0 || v.Recipes[r.chosen].OwnUse {
		t.Errorf("two jobs in: retried %v, chosen %d, want the role recipe and a retry", r.ownRetried, r.chosen)
	}
}

// TestWork_RetryWithNoEligibleRoleRecipeKeepsOwnUse verifies that when the retry
// is due and no role recipe is eligible, the own-use job goes on and the
// household is not counted as retried.
//
// Requirement: FR-SUB-022
func TestWork_RetryWithNoEligibleRoleRecipeKeepsOwnUse(t *testing.T) {
	t.Parallel()
	v := ownView()
	v.RetryJobs, v.OwnUseJobs = 2, 2
	v.Recipes[0].MaxExecutions = 0
	r := reportOf(v)
	if r.ownRetried || r.chosen < 0 || v.Recipes[r.chosen].Recipe != rForage || r.share != wholeBP || r.reason != "" {
		t.Errorf("report = retried %v, chosen %d, share %d, reason %q, want forage at 10000 and no retry", r.ownRetried, r.chosen, r.share, r.reason)
	}
}

// TestWork_BlockedActsAsOutOfMode verifies case (a): with no own-use recipe that
// can run, the household is blocked, the flag stays, and Work requests as it
// would out of the mode, a role recipe if one is eligible.
//
// Requirement: FR-SUB-023
func TestWork_BlockedActsAsOutOfMode(t *testing.T) {
	t.Parallel()
	v := ownView()
	v.Ratio = 1_000
	v.Recipes[1].MaxExecutions = 0
	r := reportOf(v)
	if !r.ownBlocked || r.chosen != 0 {
		t.Errorf("report = blocked %v, chosen %d, want blocked and the role recipe 0", r.ownBlocked, r.chosen)
	}
	v.Ratio = 20_000
	if r := reportOf(v); !r.ownBlocked || !r.rest {
		t.Errorf("a satisfied household: blocked %v, rest %v, want blocked and a rest as out of the mode", r.ownBlocked, r.rest)
	}
}

// TestWork_HaltDoesNotApply verifies that the production halt does not stop an
// own-use recipe: the household holds more staple than its output target, which
// halts a market recipe of staple, and still requests the own-use recipe.
//
// Arithmetic: output target = weight * 3 * E / Duration * OutputCoverTicks (1) =
// 3; the halt ratio is 3.0, so a stock of 10 >= 9 halts a recipe whose output
// has a cost share. Cover is 3 < Exit 6, so the household is short.
//
// Requirement: FR-SUB-024
func TestWork_HaltDoesNotApply(t *testing.T) {
	t.Parallel()
	v := ownView()
	if !v.recipeHalted(&v.Recipes[1], householdParams(&v.Tunables)) {
		t.Fatalf("recipeHalted(forage) = false, want true: the fixture must be one the halt would stop")
	}
	if r := reportOf(v); r.chosen < 0 || v.Recipes[r.chosen].Recipe != rForage || r.halted != 0 {
		t.Errorf("report = chosen %d, halted %d, want forage and no halt", r.chosen, r.halted)
	}
}

// TestWork_NoRecipeRoleRequestsOwnUse verifies that a role with no market recipe
// and an own-use list requests an own-use job in the mode.
//
// Requirement: FR-SUB-025
func TestWork_NoRecipeRoleRequestsOwnUse(t *testing.T) {
	t.Parallel()
	v := ownView()
	v.Recipes = v.Recipes[1:]
	got := (DefaultStrategy{}).Work(v, nil)
	if got.NoJob || got.Recipe != rForage || got.Executions != 1 {
		t.Errorf("Work = %+v, want one execution of forage", got)
	}
}

// TestWork_DirectedOverrides_ExternalEvaluated verifies that a Directed
// strategy's directive overrides the default Work in the mode, and that an
// external agent is evaluated by the flag step like any other.
//
// Requirement: FR-SUB-034
func TestWork_DirectedOverrides_ExternalEvaluated(t *testing.T) {
	t.Parallel()
	v := ownView()
	v.HasDirective, v.Directive = true, Directive{Job: &JobRequest{Recipe: 0, Executions: 1}}
	got := (Directed{Fallback: DefaultStrategy{}}).Work(v, rand.New(rand.NewPCG(1, 1)))
	if got.Recipe != 0 || got.NoJob {
		t.Errorf("Directed.Work = %+v, want the directive's recipe 0", got)
	}
	w := newForage(t, 1, nil)
	w.agents[1].external = true
	w.sub.grow(len(w.agents))
	w.p24OwnUse()
	if !w.sub.ownUse[1] {
		t.Errorf("flag of an external agent = false, want it evaluated like any other")
	}
}

// TestAllocs_Work verifies that the choice of the default Work in the mode, with
// Pick and Ramp, allocates nothing.
//
// Requirement: NFR-SUB-001
func TestAllocs_Work(t *testing.T) {
	v := ownView()
	v.Recipes = append(v.Recipes, forageRecipe(rForage2, 6, 1))
	s := DefaultStrategy{}
	if n := testing.AllocsPerRun(100, func() { _ = s.Work(v, nil) }); n != 0 {
		t.Errorf("allocations per Work in the mode = %v, want 0", n)
	}
}

// TestRecipes_OwnUseNotPlannedNotMadeNoAnchor verifies that a marked recipe is
// not the planned recipe of Trade, is not a recipe the role makes for the
// per-commodity cost anchor or the output cover, even when it is the best.
//
// Requirement: FR-SUB-018, FR-SUB-027
func TestRecipes_OwnUseNotPlannedNotMadeNoAnchor(t *testing.T) {
	t.Parallel()
	v := ownView()
	tc := &tradeCtx{v: v, par: householdParams(&v.Tunables)}
	if got := tc.plannedRecipe(); got != 0 {
		t.Errorf("plannedRecipe() = %d, want 0 (the market recipe, not forage at index 1)", got)
	}
	if tc.makes(cStaple) {
		t.Errorf("makes(staple) = true, want false: only forage makes it and it is marked")
	}
	if makesIn(v.Recipes, cStaple) {
		t.Errorf("makesIn(staple) = true, want false: a marked recipe sets no anchor")
	}
	if !makesIn(v.Recipes, cItem) {
		t.Errorf("makesIn(item) = false, want true: the market recipe makes it")
	}
}

// TestView_CarriesOwnUseGroup verifies that the world's view of a household in
// the mode carries the flag, the counter, Cover, Secure and the two tunables, and
// the own-use recipes marked after the role's in ascending RecipeID; and that
// with the module off every field is zero and no recipe is marked.
//
// Requirement: FR-SUB-018
func TestView_CarriesOwnUseGroup(t *testing.T) {
	t.Parallel()
	w := newForage(t, 1, nil)
	a := &w.agents[1]
	holdGrain(t, w, a, 2)
	w.sub.grow(len(w.agents))
	w.p24OwnUse()
	w.sub.ownUseJobs[a.id] = 3
	w.refreshViewCtx()
	var v AgentView
	if err := w.fillAgentView(&v, a); err != nil {
		t.Fatalf("fillAgentView error = %v, want nil", err)
	}
	if !v.OwnUse || v.OwnUseJobs != 3 || v.Cover != 1 || v.ExitTicks != 3 || v.RetryJobs != 0 || v.Secure != 1 {
		t.Errorf("view = flag %v, jobs %d, cover %d, secure %d, exit %d, retry %d, want true, 3, 1, 1, 3, 0 (stock 2, upkeep 2, no money)",
			v.OwnUse, v.OwnUseJobs, v.Cover, v.Secure, v.ExitTicks, v.RetryJobs)
	}
	var marks []bool
	for _, r := range v.Recipes {
		marks = append(marks, r.OwnUse)
	}
	if want := []bool{false, true}; !slices.Equal(marks, want) {
		t.Errorf("marks of the recipes = %v, want %v: the role's tend, then the own-use gather", marks, want)
	}
	off := newForage(t, 1, func(c *Config) { c.Modules.Subsistence = nil })
	off.refreshViewCtx()
	var vo AgentView
	if err := off.fillAgentView(&vo, &off.agents[1]); err != nil {
		t.Fatalf("fillAgentView (off) error = %v, want nil", err)
	}
	if vo.OwnUse || vo.OwnUseJobs != 0 || vo.Cover != 0 || vo.Secure != 0 || vo.ExitTicks != 0 || vo.RetryJobs != 0 || len(vo.Recipes) != 1 || vo.Recipes[0].OwnUse {
		t.Errorf("view with the module off = %+v, want the own-use group zero and the role's one recipe", vo)
	}
}

// TestView_TradeViewCoverRecomputed verifies that Cover and Secure are those of
// the stock at the moment the view is built, not the flag step's: after the
// household eats, the view built for Trade shows the lower cover.
//
// Requirement: FR-SUB-018
func TestView_TradeViewCoverRecomputed(t *testing.T) {
	t.Parallel()
	w := newForage(t, 1, nil)
	a := &w.agents[1]
	holdGrain(t, w, a, 6)
	w.sub.grow(len(w.agents))
	w.p24OwnUse()
	w.refreshViewCtx()
	var v1, v2 AgentView
	if err := w.fillAgentView(&v1, a); err != nil {
		t.Fatal(err)
	}
	takeGrain(t, w, a, 4)
	if err := w.fillAgentView(&v2, a); err != nil {
		t.Fatal(err)
	}
	if v1.Cover != 3 || v2.Cover != 1 {
		t.Errorf("Cover before and after the stock fell by 4 = %d and %d, want 3 and 1", v1.Cover, v2.Cover)
	}
}

// TestStart_CountsOwnUseJobs verifies the counter: an own-use start adds 1 and
// counts the executions in the cell, the start of any other recipe resets it.
//
// Requirement: FR-SUB-022
func TestStart_CountsOwnUseJobs(t *testing.T) {
	t.Parallel()
	w := newForage(t, 1, nil)
	w.sub.grow(len(w.agents))
	a := &w.agents[1]
	gather, tend := RecipeID(0), RecipeID(1)
	cell := int(w.regionOf(a))*len(w.cfg.Roles) + int(a.role)
	w.sub.noteStart(w, a, cell, gather, 1)
	w.sub.noteStart(w, a, cell, gather, 2)
	if w.sub.ownUseJobs[a.id] != 2 || w.sub.started[cell] != 3 {
		t.Errorf("after two own-use starts: counter %d, started %d, want 2 and 3", w.sub.ownUseJobs[a.id], w.sub.started[cell])
	}
	w.sub.noteStart(w, a, cell, tend, 1)
	if w.sub.ownUseJobs[a.id] != 0 {
		t.Errorf("after a start of the market recipe: counter %d, want 0", w.sub.ownUseJobs[a.id])
	}
}

// TestWorld_HouseholdFeedsItselfOnTheFloor runs the small world: a household with
// no grain and no money enters at the first flag step, requests gather at P2.5,
// starts it at P2.7 (a one-tick job completes at once) and so holds its upkeep.
//
// Arithmetic: gather makes 2 grain at the yield 10 000; the upkeep is 2, so the
// household is fed and its counter is 1 after the first tick.
//
// Requirement: FR-SUB-019, FR-SUB-022
func TestWorld_HouseholdFeedsItselfOnTheFloor(t *testing.T) {
	t.Parallel()
	w := newForage(t, 1, nil)
	if err := w.Step(); err != nil {
		t.Fatalf("Step error = %v, want nil", err)
	}
	a := &w.agents[1]
	if !w.sub.ownUse[a.id] || w.sub.ownUseJobs[a.id] != 1 || a.unfed != 0 {
		t.Errorf("after the first tick: flag %v, counter %d, unfed %d, want true, 1, 0", w.sub.ownUse[a.id], w.sub.ownUseJobs[a.id], a.unfed)
	}
	cell := int(w.regionOf(a)) * len(w.cfg.Roles)
	if w.sub.entered[cell+int(a.role)] != 1 || w.sub.started[cell+int(a.role)] != 1 {
		t.Errorf("counts = entered %d, started %d, want 1 and 1", w.sub.entered[cell+int(a.role)], w.sub.started[cell+int(a.role)])
	}
}

// TestSubsistence_OffLeavesNoTrace verifies that with the module off the world
// holds no state of it, Snapshot.Modules does not name it, and with it on and
// nobody ever entering (every household is rich and fed) a run equals the run
// with it off in everything but the module's name and the configuration hash.
//
// Requirement: FR-SUB-001, FR-SUB-035
func TestSubsistence_OffLeavesNoTrace(t *testing.T) {
	t.Parallel()
	rich := func(c *Config) {
		c.Roles[0].StartMoney = 1_000_000
		c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 40}}
	}
	on := newForage(t, 3, rich)
	off := newForage(t, 3, func(c *Config) { rich(c); c.Modules.Subsistence = nil })
	if off.sub != nil || on.sub == nil {
		t.Fatalf("sub = %v (off) and %v (on), want nil and non-nil", off.sub, on.sub)
	}
	for range 6 {
		if err := on.Step(); err != nil {
			t.Fatal(err)
		}
		if err := off.Step(); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range on.sub.ownUse {
		if f {
			t.Fatalf("a household entered the mode: the fixture must be one where nobody does")
		}
	}
	so, err := on.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	sf, err := off.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(so.Modules, []string{"subsistence"}) || len(sf.Modules) != 0 {
		t.Errorf("Modules = %v (on) and %v (off), want [subsistence] and none", so.Modules, sf.Modules)
	}
	so.Modules, sf.Modules, so.ConfigHash, sf.ConfigHash = nil, nil, [32]byte{}, [32]byte{}
	if !reflect.DeepEqual(so, sf) {
		t.Errorf("the snapshot with the module on and nobody entering differs from the one with it off")
	}
}

// TestSpawner_RegionAgentCarriesOwnUse verifies that the per-agent row the
// spawner is given carries the flag, read-only.
//
// Requirement: FR-SUB-051
func TestSpawner_RegionAgentCarriesOwnUse(t *testing.T) {
	t.Parallel()
	w := newForage(t, 2, nil)
	w.sub.grow(len(w.agents))
	w.sub.ownUse[2] = true
	w.ex.view.Agents = w.ex.view.Agents[:0]
	for _, id := range []int{1, 2} {
		w.addAgentRow(&w.agents[id])
	}
	if got := []bool{w.ex.view.Agents[0].OwnUse, w.ex.view.Agents[1].OwnUse}; !slices.Equal(got, []bool{false, true}) {
		t.Errorf("RegionAgent.OwnUse = %v, want [false true]", got)
	}
}

// TestAllocs_FlagStep verifies that the flag step and the filling of the view
// allocate nothing once the world is built (NFR-SUB-001).
//
// Requirement: NFR-SUB-001
func TestAllocs_FlagStep(t *testing.T) {
	w := newForage(t, 4, nil)
	w.sub.grow(len(w.agents))
	w.p24OwnUse()
	w.refreshViewCtx()
	var v AgentView
	if err := w.fillAgentView(&v, &w.agents[1]); err != nil {
		t.Fatal(err)
	}
	if n := testing.AllocsPerRun(50, func() { w.p24OwnUse() }); n != 0 {
		t.Errorf("allocations per flag step = %v, want 0", n)
	}
	if n := testing.AllocsPerRun(50, func() { _ = w.fillAgentView(&v, &w.agents[1]) }); n != 0 {
		t.Errorf("allocations per fillAgentView in the mode = %v, want 0", n)
	}
}

// TestPositions_FlagStepBetweenSnapshotAndWork verifies the place of P2.4 in the
// table: after the snapshot of the endowments, before the first Work.
//
// Requirement: FR-SUB-017
func TestPositions_FlagStepBetweenSnapshotAndWork(t *testing.T) {
	t.Parallel()
	at := func(id string) int {
		for i, p := range positionList {
			if p.id == id {
				return i
			}
		}
		return -1
	}
	if !(at("P2.3") >= 0 && at("P2.3")+1 == at("P2.4") && at("P2.4")+1 == at("P2.5")) || positionList[at("P2.4")].owner != "SUB" {
		t.Errorf("positions P2.3, P2.4, P2.5 are at %d, %d, %d (owner %q), want consecutive and P2.4 owned by SUB", at("P2.3"), at("P2.4"), at("P2.5"), positionList[at("P2.4")].owner)
	}
}
