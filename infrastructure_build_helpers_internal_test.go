package economysim

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// bldScript is a Strategy that asks for one recipe up to max executions whenever the agent
// is free and trades nothing; a test changes recipe and max between ticks. It also keeps the
// facilities each Work view showed, by agent, so a test reads the view without a pointer into
// the scratch.
type bldScript struct {
	recipe RecipeID
	max    int64
	off    bool
	seen   map[AgentID][]FacilityView
	// k0 is the MaxExecutions of the asked recipe in the view of each agent.
	k0 map[AgentID]int64
}

func (s *bldScript) Work(v *AgentView, _ *rand.Rand) JobRequest {
	if s.seen != nil {
		s.seen[v.Agent] = slices.Clone(v.Facilities)
		for _, r := range v.Recipes {
			if r.Recipe == s.recipe {
				s.k0[v.Agent] = r.MaxExecutions
			}
		}
	}
	return JobRequest{Recipe: s.recipe, Executions: s.max, NoJob: s.off}
}
func (*bldScript) Trade(*AgentView, *rand.Rand, *TradePlan)   {}
func (*bldScript) Revise(o *OutcomeView, _ *rand.Rand) Belief { return o.Belief }

var _ Strategy = (*bldScript)(nil)

// bldConfig is facConfig with five recipes that use the clauses (all listed by the miller
// role, whose strategy is "script"), and one miller of weight 5 in each region with plenty of
// grain: zz_grind (needs the mill, one tick), zz_build (builds a mill, two ticks), zz_fix (repairs
// a mill, 100 units an execution, one tick), zz_road (builds the road toward yonder, one tick)
// and zz_barn (builds a barn, one tick). Every one has one grain as its input; only zz_grind has
// an output. The numbers are placeholders (N-015).
func bldConfig(mut func(c *Config)) Config {
	c := facConfig(nil)
	grain := []Item{{Commodity: "grain", Quantity: 1}}
	c.Recipes = append(c.Recipes,
		Recipe{ID: "zz_barn", TraineeYield: 5_000, Duration: 1, Inputs: grain, Builds: &FacilityBuild{Class: "barn"}},
		Recipe{ID: "zz_build", TraineeYield: 5_000, Duration: 2, Inputs: grain, Builds: &FacilityBuild{Class: "mill"}},
		Recipe{ID: "zz_fix", TraineeYield: 5_000, Duration: 1, Inputs: grain, Repairs: &FacilityRepair{Class: "mill", Units: 100}},
		Recipe{ID: "zz_grind", TraineeYield: 5_000, Duration: 1, Inputs: grain, Needs: []string{"mill"},
			Outputs: []Output{{Commodity: "flour", Quantity: 2, CostShare: 10_000}}},
		Recipe{ID: "zz_road", TraineeYield: 5_000, Duration: 1, Inputs: grain, Builds: &FacilityBuild{Class: "road", Toward: "yonder"}},
	)
	c.Roles[0].Recipes = []string{"mill", "zz_barn", "zz_build", "zz_fix", "zz_grind", "zz_road"}
	c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 100}, {Commodity: "pan", Quantity: 1}}
	c.Roles[0].Strategy = "script"
	c.Groups = []Group{
		{Region: "home", Role: "miller", Households: 5, Scale: 5},
		{Region: "home", Role: "weaver", Households: 1},
		{Region: "yonder", Role: "miller", Households: 5, Scale: 5},
		{Region: "yonder", Role: "weaver", Households: 1},
	}
	if mut != nil {
		mut(&c)
	}
	return c
}

// bldWorld builds the world of bldConfig with the scripted strategy, an event log and the audit.
func bldWorld(t testing.TB, mut func(c *Config)) (*World, *bldScript, *evLog) {
	t.Helper()
	s, rec := &bldScript{}, &evLog{}
	w := newMill(t, bldConfig(mut), WithSeed(facSeed), WithAudit(), WithObserver(rec), WithStrategy("script", s))
	return w, s, rec
}

// bldRecipe is the ID of the recipe of the configuration.
func bldRecipe(t testing.TB, w *World, id string) RecipeID {
	t.Helper()
	i := slices.IndexFunc(w.cfg.Recipes, func(r Recipe) bool { return r.ID == id })
	if i < 0 {
		t.Fatalf("recipe %q is not in the configuration", id)
	}
	return RecipeID(i)
}

// bldAsk sets what the scripted miller asks for.
func bldAsk(t testing.TB, w *World, s *bldScript, recipe string, max int64) {
	t.Helper()
	s.recipe, s.max = bldRecipe(t, w, recipe), max
}

// bldK0 is the executions the agent could start now for the recipe and the reason when none,
// read by the count the start uses.
func bldK0(t testing.TB, w *World, id AgentID, recipe string) (int64, Reason) {
	t.Helper()
	a := &w.agents[id]
	q := w.startReq(a, w.regionOf(a), bldRecipe(t, w, recipe), 0)
	k, reason, err := w.prod.maxExecutions(w.ledger, &q)
	if err != nil {
		t.Fatalf("maxExecutions(agent %d, %s) error = %v, want nil", id, recipe, err)
	}
	return k, reason
}

// bldAdd adds n facilities of the class at full condition in region 0 for the owner.
func bldAdd(t testing.TB, w *World, class FacilityClassID, owner AgentID, n int, condition ...int64) []FacilityID {
	t.Helper()
	adds := make([]NewFacility, n)
	for i := range adds {
		adds[i] = NewFacility{Class: class, Region: 0, Toward: NoRegion, Owner: owner}
		if i < len(condition) {
			adds[i].Condition = condition[i]
		}
	}
	ids, err := w.AddFacilities(adds)
	if err != nil {
		t.Fatalf("AddFacilities(%d of class %d) error = %v, want nil", n, class, err)
	}
	return ids
}

// bldGrain removes every grain of the agent's account, so no recipe has its input.
func bldGrain(t testing.TB, w *World, id AgentID) {
	t.Helper()
	acct := w.agents[id].account
	grain, ok := w.cfg.commodityID("grain")
	if !ok {
		t.Fatal("commodity grain is not in the configuration")
	}
	held, _, err := w.ledger.Holding(acct, int(grain))
	if err != nil {
		t.Fatalf("Holding(grain of agent %d) error = %v, want nil", id, err)
	}
	if err := w.ledger.Transform(acct, []ledger.Line{{Commodity: int(grain), Quantity: held}}, nil); err != nil {
		t.Fatalf("Transform(grain of agent %d out) error = %v, want nil", id, err)
	}
}
