package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// facingRules is cfg with both actors on a rule over the same two regions' unfed share, face to face on
// one road: council_a lowers the foot factor from the centre to the east by 1 000 for each 500
// basis points the share stands above 50, and council_b raises the factor of the foot lane back from
// the east to the centre by the same. Both read the regions of their own jurisdictions; each decides every 30
// ticks on its own rotation. The grant and the border of council_a have no reaction.
func facingRules(cfg economysim.Config) economysim.Config {
	c := cfg
	m := *cfg.Modules.Policy
	m.Actors = slices.Clone(m.Actors)
	a := &m.Actors[0]
	a.Policy, a.Schedule, a.Every, a.ReadsRegions = "rule", nil, 30, true
	a.Rules = []economysim.Reaction{{Lever: "road", Gauge: economysim.GaugeUnfedShare, Target: 50, Deadband: 50, Span: 500, Move: 1_000, Raise: false}}
	for i := range a.Levers {
		if a.Levers[i].ID == "road" {
			a.Levers[i].MaxStep = 1_000
		}
	}
	c.Modules.Policy = &m
	return c
}

// TestScenarioS157_TwoRulesFaceToFace reports PLC-S3 on the pack of the golden, seeds 1 to 10, 240
// ticks: two rules, one lowering the foot factor of a road in a direction and the other raising the
// factor of the lane in the other, on the unfed share of their own regions. Exact: the audit holds
// at every tick, every factor either of them writes is within its lever's bounds and no step is
// larger than its MaxStep. Reported, never asserted (N-485): the number of moves of each lever, its
// lowest and highest factor and its last. The stability of two such loops is the register's
// (PLC-A1), not this test's.
//
// Requirement: FR-PLC-041, FR-PLC-047, FR-PLC-060, FR-PLC-062
//
// Guard: a factor outside [5 000, 20 000] or a step above the lever's limit fails here.
func TestScenarioS157_TwoRulesFaceToFace(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := facingRules(loadValidPack(t, fivePolicy...))
	centre, east := regionRank(cfg, "centre"), regionRank(cfg, "east")
	policyScenario(t, "S157 five+policy two rules face to face, 240 ticks", scenarioSeedCount(), func(t *testing.T, rep *scenReport, seed uint64) {
		w, k := policyWorld(t, cfg, seed)
		policyStepTo(t, w, shockTicks, nil)
		for _, lever := range []struct {
			name     string
			from, to economysim.RegionID
			step     int64
		}{{"road", centre, east, 1_000}, {"lane", east, centre, 2_000}} {
			var path []int64
			inside, stepped := true, true
			prev := int64(10_000)
			for _, f := range k.transit {
				if f.From != lever.from || f.To != lever.to {
					continue
				}
				path = append(path, f.AfterBP)
				inside = inside && f.AfterBP >= 5_000 && f.AfterBP <= 20_000
				stepped = stepped && abs64(f.AfterBP-prev) <= lever.step
				prev = f.AfterBP
			}
			checkExact(t, rep, "S157", "five", seed, lever.name+" stays within [5 000, 20 000]", inside, "path %v", path)
			checkExact(t, rep, "S157", "five", seed, lever.name+" moves by at most its step", stepped, "path %v", path)
			rep.num(seed, lever.name+" moves", int64(len(path)))
			rep.num(seed, lever.name+" last factor bp", prev)
			rep.num(seed, lever.name+" highest factor bp", slices.Max(append(path, 10_000)))
			rep.num(seed, lever.name+" lowest factor bp", slices.Min(append(path, 10_000)))
		}
	})
}
