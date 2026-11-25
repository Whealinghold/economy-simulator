package economysim_test

import (
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// savingS146Legs are the three legs of S146 on the growers, who receive a grant of the money supply
// each tick: no outlet (a buffer and no project), an outlet (the mill, valued by a Worth row) and a
// holding charge (the sink of overlay/faucet-sunk.toml, no outlet).
var savingS146Legs = []struct {
	name   string
	layer  string
	outlet bool
}{
	{"no outlet", "overlay/faucet.toml", false},
	{"a project", "overlay/faucet.toml", true},
	{"a holding charge", "overlay/faucet-sunk.toml", false},
}

// TestScenarioS146_SurplusWithNoOutlet reports SAV-S5 of spec-saving-and-investment.md 14.3 on pack
// one with the infrastructure and saving overlays and the money supply of overlay/faucet.toml, 200
// ticks, audit on: a role with a grant each tick and a surplus no want absorbs (the growers), first
// with no outlet, then with a project that a Worth row values, then with the holding charge of
// overlay/faucet-sunk.toml. The identities are asserted (the audit holds; no idle money is
// negative); the effect is reported: the sampled idle money (money above the target, spec 8.1) of
// the roles with a block, and the facilities built (the cooks build in every leg; the growers only
// in the leg with the Worth row).
//
// Arithmetic: idle money is max(0, Money - S*) (spec 8.1) and a household with a project funds it
// from the idle money, so the outlet leg should hold less of it than the leg with none; whether it
// does is for STY, and the placeholders assert no band (N-015, N-027).
//
// Requirement: FR-SAV-031, FR-SAV-032, FR-SAV-033, FR-SAV-036
//
// Guard: a negative idle row, or a step that fails the audit while the surplus is held, taxed or
// spent on a facility, fails here.
func TestScenarioS146_SurplusWithNoOutlet(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	savingScenario(t, "S146 one+saving a surplus with no outlet, a project and a charge, 200 ticks", min(5, scenarioSeedCount()), func(t *testing.T, rep *scenReport, seed uint64) {
		for _, leg := range savingS146Legs {
			cfg := loadValidPack(t, append(slices.Clone(packOneSaving), leg.layer)...)
			if leg.outlet {
				g := roleOf(&cfg, "grower")
				g.Recipes = append(g.Recipes, "build_mill")
				g.Saver.Worth = []economysim.ProjectWorth{{Class: "mill", PerTick: 5_000, Count: 1}}
			}
			st := savingRun(t, cfg, seed, 200)
			name := leg.name
			rep.num(seed, name+": idle money, sampled", st.IdleSamples)
			rep.num(seed, name+": built", st.Built)
			checkExact(t, rep, "S146", "one", seed, name+": the idle money is not negative", st.IdleNegative == 0, "%d rows with negative idle money", st.IdleNegative)
		}
	})
}
