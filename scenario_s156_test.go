package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// shockTicks is the length of S156's run, shockEnds the tick at which the yield shock is lifted and
// sampleEvery the ticks between two readings of the unfed share.
const (
	shockTicks  = 240
	shockEnds   = 60
	sampleEvery = 10
)

// shockOutcome is what S156 measures of one world: the highest and the last unfed share of the two
// regions of council_b, in basis points of its households, and its moves of the lane.
type shockOutcome struct {
	peak, last int64
	laneMoves  int
	laneInside bool // every factor of the lane is within [5 000, 20 000]
	policyOps  int  // slot events with the cause policy
}

// unfedShare is the share of the households of the regions that went unfed at the boundary, in basis
// points, rounded half to even as the gauge is (spec-policy.md 8.3.2); 0 for no household.
func unfedShare(t testing.TB, w *economysim.World, regions ...economysim.RegionID) int64 {
	t.Helper()
	var unfed, households int64
	for _, g := range regions {
		rs, err := w.Region(g)
		if err != nil {
			t.Fatalf("Region(%d) error = %v, want nil", g, err)
		}
		households += rs.Region.Households
		for _, r := range rs.Roles {
			unfed += r.Unfed
		}
	}
	if households == 0 {
		return 0
	}
	return (10_000*unfed + households/2) / households
}

// runShock runs cfg for seed with the yield shock of S79 on grow_staple in the east and the south
// (3 000 of 10 000 until tick 60), reading the unfed share of council_b's regions every 10 ticks.
func runShock(t testing.TB, cfg economysim.Config, seed uint64) shockOutcome {
	t.Helper()
	east, south := regionRank(cfg, "east"), regionRank(cfg, "south")
	ids := make([]string, len(cfg.Recipes))
	for i, r := range cfg.Recipes {
		ids[i] = r.ID
	}
	slices.Sort(ids)
	grow := economysim.RecipeID(slices.Index(ids, "grow_staple"))
	w, k := policyWorld(t, cfg, seed)
	set := func(bp int64) {
		t.Helper()
		if err := w.SetYieldShocks([]economysim.YieldShock{{Region: east, Recipe: grow, BP: bp}, {Region: south, Recipe: grow, BP: bp}}); err != nil {
			t.Fatalf("SetYieldShocks(%d) error = %v, want nil", bp, err)
		}
	}
	var out shockOutcome
	set(3_000)
	policyStepTo(t, w, shockTicks, func(tick int64) {
		if tick == shockEnds {
			set(10_000)
		}
		if tick%sampleEvery == 0 {
			out.last = unfedShare(t, w, east, south)
			out.peak = max(out.peak, out.last)
		}
	})
	centre := regionRank(cfg, "centre")
	out.laneInside = true
	for _, f := range k.transit {
		if f.From == east && f.To == centre {
			out.laneMoves++
			out.laneInside = out.laneInside && f.AfterBP >= 5_000 && f.AfterBP <= 20_000
		}
	}
	out.policyOps = len(k.flows) + len(k.transit) + len(k.edges)
	return out
}

// holdAll is cfg with every actor of the module on hold: the same levers, nothing decided.
func holdAll(cfg economysim.Config) economysim.Config {
	h := cfg
	m := *cfg.Modules.Policy
	m.Actors = slices.Clone(m.Actors)
	for i := range m.Actors {
		m.Actors[i].Policy, m.Actors[i].Schedule, m.Actors[i].Rules, m.Actors[i].Cycle = "hold", nil, nil, 0
	}
	h.Modules.Policy = &m
	return h
}

// TestScenarioS156_ARuleAnswersAShock reports PLC-S2 of spec-policy.md 14.2 on the pack of the golden,
// seeds 1 to 10, 240 ticks: the yield of grow_staple in the east and the south is cut to 3 000 of 10 000
// by SetYieldShocks until tick 60 (the shock of S79), with council_b's rule on the unfed share of
// the two regions against the same world with every actor on hold. Exact: both runs pass the
// conservation audit at every tick, the held world writes no slot with the cause policy, the world
// with the actors writes its slots only through them, and council_b's lane never leaves its bounds
// [5 000, 20 000]. Reported, never asserted (N-485): the highest and the last unfed share of
// the two regions in both worlds and the moves of the lane. The fiscal flow of the spec's example is
// the faucet's stipend here (D17): the pack has no fiscal module.
//
// Requirement: FR-PLC-047, FR-PLC-060, FR-PLC-061, FR-PLC-062, FR-PLC-001
//
// Guard: an actor that wrote a slot while on hold, or a lever moved outside its bounds, fails here;
// the stability of the loop is not asserted.
func TestScenarioS156_ARuleAnswersAShock(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, fivePolicy...)
	policyScenario(t, "S156 five+policy rule against hold after a yield shock, 240 ticks", scenarioSeedCount(), func(t *testing.T, rep *scenReport, seed uint64) {
		ruled, held := runShock(t, cfg, seed), runShock(t, holdAll(cfg), seed)
		rep.num(seed, "peak unfed share with the rule bp", ruled.peak)
		rep.num(seed, "peak unfed share on hold bp", held.peak)
		rep.num(seed, "last unfed share with the rule bp", ruled.last)
		rep.num(seed, "last unfed share on hold bp", held.last)
		rep.num(seed, "moves of the lane with the rule", int64(ruled.laneMoves))
		checkExact(t, rep, "S156", "five", seed, "a world on hold writes no slot", held.policyOps == 0, "%d slot events with the cause policy on hold", held.policyOps)
		checkExact(t, rep, "S156", "five", seed, "the lane stays within its bounds", ruled.laneInside, "a factor of the lane left [5 000, 20 000]")
		checkExact(t, rep, "S156", "five", seed, "the actors write slots", ruled.policyOps > 0, "no slot event from the actors")
	})
}
