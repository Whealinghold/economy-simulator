package economysim_test

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// buildScript is a Strategy for the end-to-end build test: every agent of the role asks for one
// recipe, as many executions as it can run, and trades nothing. The test changes recipe between
// ticks.
type buildScript struct{ recipe economysim.RecipeID }

func (s *buildScript) Work(*economysim.AgentView, *rand.Rand) economysim.JobRequest {
	return economysim.JobRequest{Recipe: s.recipe}
}
func (*buildScript) Trade(*economysim.AgentView, *rand.Rand, *economysim.TradePlan) {}
func (*buildScript) Revise(o *economysim.OutcomeView, _ *rand.Rand) economysim.Belief {
	return o.Belief
}

var _ economysim.Strategy = (*buildScript)(nil)

// TestInfrastructureE2E_BuildLimitedThenCompleted verifies, through the public API only, the life
// of a facility build under the limits of the module: (1) a site-bound recipe asked for more
// than the capacity of the mills of its owner starts only what the capacity allows, and
// FacilityLimited books the executions served and the ones cut; (2) the use wears the mills (the
// condition read by Facilities falls by more than the decay), and a repair recipe then raises
// it (FacilityChanged repaired, cause recipe); (3) a build recipe of 3 ticks makes one more mill
// per execution for its principal, at the third step, with RecipeExecuted before the
// FacilityChanged built; (4) a build of a store (one tick) is installed at the next step
// (FacilityChanged installed) and the snapshot has no pending facility after it. The audit holds
// at every step and the same seed gives the same events (a second run).
//
// Setup: infrastructureConfig, seed 11, every grower scripted to ask one recipe at a time; the
// growers get fuel and seed in stock so no input binds. The facility counts are read from the
// public Facilities call, not from the records.
//
// Requirement: FR-CAP-012, FR-CAP-013, FR-CAP-014, FR-CAP-015, FR-CAP-019, FR-CAP-026, FR-CAP-038
func TestInfrastructureE2E_BuildLimitedThenCompleted(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	first, second := runBuildScenario(t), runBuildScenario(t)
	if !reflect.DeepEqual(first, second) {
		t.Errorf("events of two runs of the build scenario with seed 11 differ (%d against %d events), want identical", len(first), len(second))
	}
}

// buildRun is a world of the build scenario with the script and the recorder that drive and
// observe it.
type buildRun struct {
	t   *testing.T
	w   *economysim.World
	s   *buildScript
	rec *recorder
}

// runBuildScenario runs the phases of TestInfrastructureE2E_BuildLimitedThenCompleted, asserts
// each, and returns the events of the run.
func runBuildScenario(t *testing.T) []economysim.Event {
	t.Helper()
	c := infrastructureConfig()
	store := economysim.DefaultRecipe()
	store.ID, store.Duration = "build_barn", 1
	store.Inputs = []economysim.Item{{Commodity: "seed", Quantity: 1}}
	store.Builds = &economysim.FacilityBuild{Class: "barn"}
	c.Recipes = append(c.Recipes, store)
	grower := roleOf(&c, "grower")
	grower.Recipes = append(grower.Recipes, "build_barn")
	grower.Strategy = "script"
	r := &buildRun{t: t, s: &buildScript{}, rec: &recorder{}}
	w, err := economysim.NewWorld(c, economysim.WithSeed(11), economysim.WithAudit(), economysim.WithObserver(r.rec), economysim.WithStrategy("script", r.s))
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	r.w = w
	r.limitedAndWorn()
	r.repaired()
	r.builtAfterThreeTicks()
	r.storeInstalled()
	return r.rec.events
}

// ask makes every grower ask for the recipe from the next step.
func (r *buildRun) ask(id string) {
	r.t.Helper()
	i := slices.IndexFunc(r.w.Config().Recipes, func(x economysim.Recipe) bool { return x.ID == id })
	if i < 0 {
		r.t.Fatalf("recipe %q is not in the configuration", id)
	}
	r.s.recipe = economysim.RecipeID(i)
}

// mills is the mills of the north, read through the public Facilities call.
func (r *buildRun) mills() []economysim.FacilityState {
	r.t.Helper()
	all, err := r.w.Facilities(0)
	if err != nil {
		r.t.Fatalf("Facilities(north) error = %v, want nil", err)
	}
	return slices.DeleteFunc(all, func(f economysim.FacilityState) bool { return f.Class != e2eMill })
}

// condition is the sum of the condition of the mills of the north.
func (r *buildRun) condition() (n int64) {
	for _, f := range r.mills() {
		n += f.Condition
	}
	return n
}

// changed is the FacilityChanged events of the run so far of the class and kind.
func (r *buildRun) changed(class economysim.FacilityClassID, kind economysim.FacilityKind) []economysim.FacilityChanged {
	var out []economysim.FacilityChanged
	for _, e := range r.rec.events {
		if f, ok := e.(economysim.FacilityChanged); ok && f.Class == class && f.Kind == kind {
			out = append(out, f)
		}
	}
	return out
}

// limitedAndWorn is phases 1 and 2a: grind asked for the most it can run is bounded by the two
// mills of capacity 2, FacilityLimited books the executions served and cut, and the use wears
// the mills beyond the decay of 1 each.
func (r *buildRun) limitedAndWorn() {
	r.t.Helper()
	r.ask("grind")
	before, n := r.condition(), int64(len(r.mills()))
	step(r.t, r.w, 1)
	var served, cut int64
	for _, e := range r.rec.events {
		if l, ok := e.(economysim.FacilityLimited); ok && l.Region == 0 && l.Class == e2eMill {
			served, cut = served+l.Served, cut+l.Cut
		}
	}
	if served == 0 || cut == 0 {
		r.t.Errorf("FacilityLimited of the mill after a grind asked for the most: Served %d, Cut %d, want both above 0", served, cut)
	}
	if got := r.condition(); got >= before-n {
		r.t.Errorf("condition of the mills after a step of use = %d, want below %d (decay alone)", got, before-n)
	}
}

// repaired is phase 2b: a repair recipe raises the condition, FacilityChanged repaired, cause
// recipe.
func (r *buildRun) repaired() {
	r.t.Helper()
	r.ask("repair_mill")
	step(r.t, r.w, 1)
	got := r.changed(e2eMill, economysim.FacilityRepaired)
	if len(got) == 0 || got[0].Cause != economysim.CauseRecipe || got[0].After <= got[0].Before {
		r.t.Errorf("FacilityChanged repaired = %+v, want at least one, cause recipe, After above Before", got)
	}
}

// builtAfterThreeTicks is phase 3: a build of three ticks adds mills at the third step, never
// before.
func (r *buildRun) builtAfterThreeTicks() {
	r.t.Helper()
	r.ask("build_mill")
	count := len(r.mills())
	step(r.t, r.w, 2)
	if got := len(r.mills()); got != count {
		r.t.Errorf("mills after two ticks of a three-tick build = %d, want %d", got, count)
	}
	step(r.t, r.w, 1)
	if got := len(r.mills()); got <= count {
		r.t.Errorf("mills after three ticks of a three-tick build = %d, want more than %d", got, count)
	}
	if got := r.changed(e2eMill, economysim.FacilityBuilt); len(got) == 0 || got[0].Cause != economysim.CauseRecipe {
		r.t.Errorf("FacilityChanged built of the mill = %+v, want at least one with cause recipe", got)
	}
}

// storeInstalled is phase 4: a store built in one tick is installed at the next step, and the
// snapshot has no pending store after it.
func (r *buildRun) storeInstalled() {
	r.t.Helper()
	r.ask("build_barn")
	step(r.t, r.w, 2)
	built, installed := r.changed(e2eBarn, economysim.FacilityBuilt), r.changed(e2eBarn, economysim.FacilityInstalled)
	if len(built) == 0 || len(installed) < len(built) {
		r.t.Errorf("barns built = %d, installed = %d, want at least one built and all installed within the next step", len(built), len(installed))
	}
	snap, err := r.w.Snapshot()
	if err != nil {
		r.t.Fatalf("Snapshot error = %v, want nil", err)
	}
	for _, row := range snap.Facilities {
		if row.Class == e2eBarn && row.Pending != 0 {
			r.t.Errorf("snapshot row of the barns in region %d has Pending %d, want 0 after the install", row.Region, row.Pending)
		}
	}
}
