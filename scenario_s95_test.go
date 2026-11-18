package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// storesLevy is the flow of S95: a levy on every barn, amount a tick per store.
func storesLevy(amount, exempt int64) economysim.MoneyFlow {
	return economysim.MoneyFlow{
		ID: "barn_levy", Kind: economysim.FlowLevy, Target: economysim.FlowTarget{Kind: "stores", Store: "barn"},
		Amount: amount, Exempt: exempt,
	}
}

// TestScenarioS95_LevyAgainstItsPairAndExemption is STO-S5 (spec-storage.md 16): pack one
// with the storage module and a levy on the grain barn of the growers, 200 ticks, seeds 1
// to 10, audit on, in two variants: a levy of 5 a store a tick with no exemption, and the
// same levy with an exemption above any balance. Exact: the audit holds every tick, and
// with it the boundary balance of the flow equals its total (spec-money-supply.md 12.1);
// the flow collected something in the first variant and nothing in the second. Recorded:
// the total, and the households at the end of each.
//
// Requirement: FR-STO-026, FR-STO-027, FR-MON-008
//
// Guard: a levy whose exemption is per store, not per account, or one that bypassed the
// boundary account, would pass a run of one household and fail the audit here.
func TestScenarioS95_LevyAgainstItsPairAndExemption(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	for _, v := range []struct {
		name         string
		exempt       int64
		collectsSome bool
	}{{"no exemption", 0, true}, {"exempt above every balance", economysim.MaxFlowAmount, false}} {
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, packOneStorage...)
			cfg.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{storesLevy(5, v.exempt)}}
			stoScenario(t, "one", fmt.Sprintf("S95 one+storage levy on the barn, %s, 200 ticks", v.name), func(t *testing.T, rep *scenReport, seed uint64) {
				res := runPack(t, cfg, runSpec{Seed: seed, Ticks: 200, Drive: driveStep, Audit: true})
				var total economysim.Money
				for _, f := range res.Final.Flows {
					total += f.Total
				}
				rep.num(seed, "levy collected", int64(total))
				rep.num(seed, "households at end", res.Final.Households)
				checkExact(t, rep, "S95", "one", seed, "the flow collects exactly when its exemption leaves a balance above it", (total > 0) == v.collectsSome, "total %d, want collected %t", total, v.collectsSome)
			})
		})
	}
}
