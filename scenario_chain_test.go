package economysim_test

import (
	"context"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// chainPayrollKnobs is the data of trial T1 of plan 2-29 that is not a census: the
// yield of the staple in every season, the tool wear of 24 uses, and the privation
// bounds of the stability work (N-352). The census is not here: the test sets it from
// the payroll-aware solver, so the layer cannot disagree with it.
const chainPayrollKnobs = `
[region.north.yield.grow_staple]
all = 10000
[commodity.tool_b]
durability = 24
[world]
remove_privation = 40
change_privation = 12
`

// payrollSeedCold loads the layers of trial T1 (the cold layers of one-demand, the
// knobs and the payroll overlay), sets every group's households to the balanced
// census the payroll-aware solver gives (twice, so the census is the fixed point of
// the solve), and puts the solver's table on the configuration as its warm start.
// It returns the configuration and the last solve.
func payrollSeedCold(t *testing.T) (economysim.Config, economysim.SteadyState) {
	t.Helper()
	cfg := loadPackBytes(t, []byte(chainPayrollKnobs), append(append([]string{}, packOneDemandCold...), "overlay/employment.toml")...)
	var st economysim.SteadyState
	for range 2 {
		var err error
		st, err = economysim.SolveSteadyState(context.Background(), cfg)
		if err != nil {
			t.Fatalf("SolveSteadyState error = %v, want nil", err)
		}
		for _, row := range st.Report.Census {
			for gi := range cfg.Groups {
				if cfg.Groups[gi].Region == row.Region && cfg.Groups[gi].Role == row.Role {
					cfg.Groups[gi].Households = row.Balanced
				}
			}
		}
	}
	table := st.Table
	cfg.WarmStart = &table
	return cfg, st
}

// TestScenarioChain_PayrollReceiptsMatchStatic compares, for the opening ticks of
// trial T1, the money the payers owed by the payroll rule (the sum of
// PayrollPaid.Due) with the receipts the payroll-aware solver assumed (assumption
// A1 of plan 2-29: a payer's receipts are its net sales at the duals, spec-viability.md
// 5.6). It prints the ratio of measured to static and does not assert a bound: the
// reading is the falsifier of H-P, and the 10% of plan 2-29 Phase 4 is applied by the
// reader of the log. It asserts the exact facts: the module is on and pays, and
// Due = Paid + Cut in every event (no money is created or lost by the rule).
//
// Static per tick = sum over payer groups of PayrollOutBP / 10 000 times a
// household's upkeep cost at the seed prices times the group's households (the
// definition of PayrollOutBP in viability_report.go). Upkeep is staple 2, prepared 1,
// fuel 1 (base.toml line 5), priced at the table.
//
// Requirement: FR-VIA-066
func TestScenarioChain_PayrollReceiptsMatchStatic(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg, st := payrollSeedCold(t)
	price := map[string]int64{}
	for _, p := range st.Table.Prices {
		price[p.Commodity] = int64(p.Price)
	}
	var upkeep int64
	for _, it := range cfg.World.Upkeep {
		upkeep += int64(it.Quantity) * price[it.Commodity]
	}
	var perTick int64
	for _, row := range st.Report.Census {
		if row.PayrollOutBP > 0 {
			perTick += row.PayrollOutBP * upkeep * row.Households / 10000
		}
	}
	if perTick == 0 {
		t.Fatal("static payroll per tick = 0, want a payer with a payroll")
	}
	const ticks = 10
	var due, paid, cut int64
	events := 0
	obs := economysim.ObserverFunc(func(e economysim.Event) {
		if p, ok := e.(economysim.PayrollPaid); ok {
			events++
			due += int64(p.Due)
			paid += int64(p.Paid)
			cut += int64(p.Cut)
			if p.Due != p.Paid+p.Cut {
				t.Errorf("PayrollPaid at tick %d: Due %d != Paid %d + Cut %d", p.Tick, p.Due, p.Paid, p.Cut)
			}
		}
	})
	runScenario(t, cfg, scenarioSpec{Seed: 101, Ticks: ticks, Options: []economysim.Option{economysim.WithObserver(obs)}})
	if events == 0 || due == 0 {
		t.Fatalf("PayrollPaid events = %d, due = %d over %d ticks, want a payroll paid", events, due, ticks)
	}
	t.Logf("static payroll per tick %d, measured Due per tick %d (paid %d, cut %d) over %d ticks: measured/static = %d bp",
		perTick, due/ticks, paid/ticks, cut/ticks, ticks, due*10000/(perTick*ticks))
}
