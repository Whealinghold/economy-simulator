package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestScenarioS34_LevyBelowTheUpkeepLine verifies MON-S2 (spec-money-supply.md 16,
// 9.4): a fixed levy equal to a household's upkeep cost, without an exemption and
// with an exemption of the upkeep cost.
//
// Exact, on the configuration (pack one, a levy of 999 on the cooks, the cost of
// one tick's upkeep at the reference prices): the warning levy_below_upkeep is
// raised without the exemption and not with it. Exact, on a small world whose
// farmers hold 1 500 each and are levied 1 000 a tick: with an exemption of 1 000
// the levy takes 500 once, leaves each household its 1 000 and then collects
// nothing (the households are Short and the levy is counted, not carried); without
// the exemption it takes all, 1 000 then 500, and the money held is 0 from the
// second tick. Reported only, on pack one over 200 ticks and seeds 1 to 3: the
// household-ticks unfed, in all roles, with no levy, with the levy and with the
// exemption. The spec's "no household goes unfed because of the levy" is not an
// observable of a run (the households' later trades differ), so it is not
// asserted; the exact version is that a levy never takes the money held below the
// exemption (FR-MON-020).
//
// Requirement: FR-MON-020, FR-MON-039, FR-REF-023
//
// Guard: the exemption is what keeps a levy from taking a household's upkeep
// money; without it a levy equal to the upkeep leaves a household nothing to buy
// its upkeep with.
func TestScenarioS34_LevyBelowTheUpkeepLine(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario, pack = "S34", "one"
	levyOn := func(cfg economysim.Config, exempt int64) economysim.Config {
		cfg.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{{ID: "levy", Kind: economysim.FlowLevy,
			Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Role: "cook"}, Amount: 999, Exempt: exempt}}}
		return cfg
	}
	base := loadValidPack(t, packOne...)
	warns := func(cfg economysim.Config) bool {
		for _, w := range cfg.Warnings() {
			if w.Rule == "levy_below_upkeep" {
				return true
			}
		}
		return false
	}
	if !warns(levyOn(base, 0)) || warns(levyOn(base, 999)) {
		t.Errorf("levy_below_upkeep raised = %v without the exemption and %v with it, want true and false", warns(levyOn(base, 0)), warns(levyOn(base, 999)))
	}
	t.Run("the exemption keeps the money held", func(t *testing.T) {
		t.Parallel()
		for _, tt := range []struct {
			exempt int64
			held   []economysim.Money // money held by the households after tick 0, 1, 2
		}{
			{1_000, []economysim.Money{10_000, 10_000, 10_000}},
			{0, []economysim.Money{5_000, 0, 0}},
		} {
			levy := flowTo("levy", economysim.FlowLevy, "farmer")
			levy.Amount, levy.Exempt = 1_000, tt.exempt
			c := moneyWorldConfig(10, 10, levy)
			c.Roles[0].StartMoney = 1_500
			w := moneyWorld(t, c)
			// The estates hold money too (the starting pool): it does not move.
			pool := roleMoney(t, w) - 10*1_500
			for tick, want := range tt.held {
				step(t, w, 1)
				if got := roleMoney(t, w) - pool; got != want {
					t.Errorf("exempt %d: money held by the households after tick %d = %d, want %d", tt.exempt, tick, got, want)
				}
			}
		}
	})
	rep := newScenReport(fmt.Sprintf("S34 one+levy seeds 1-%d, 200 ticks", min(3, scenarioSeedCount())))
	t.Run("seeds", func(t *testing.T) {
		for _, seed := range seedList(min(3, scenarioSeedCount())) {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()
				for _, v := range []struct {
					name string
					cfg  economysim.Config
				}{{"no levy", base}, {"levy", levyOn(base, 0)}, {"levy with exemption", levyOn(base, 999)}} {
					run := runScenario(t, v.cfg, scenarioSpec{Seed: seed, Ticks: 200})
					var unfed int64
					for _, u := range run.Obs.unfed {
						unfed += u
					}
					rep.num(seed, "household-ticks unfed, "+v.name, unfed)
					if v.cfg.Modules.MoneySupply != nil {
						checkExact(t, rep, scenario, pack, seed, "the levy collected money ("+v.name+")",
							len(run.Final.Flows) == 1 && run.Final.Flows[0].Total > 0, "flows %+v", run.Final.Flows)
					}
				}
			})
		}
	})
	rep.flush(t)
}
