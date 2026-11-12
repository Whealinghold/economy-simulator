package economysim_test

import (
	"context"
	"fmt"
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

// chainClaimSeeds and chainClaimTicks are the seeds and the length of the count
// claim of FR-REF-045 (acceptance criterion 2 of plan 2-29).
const (
	chainClaimFirstSeed, chainClaimLastSeed = 101, 104
	chainClaimTicks                         = 2200
)

// TestScenarioChain_Claim runs the pack `payroll` warm from its start on seeds 101
// to 104 for 2 200 ticks and asserts the household-count claim of FR-REF-045 on
// each: the households after tick 1 100 and after tick 2 200 are within 10% of the
// households after tick 600, and no role of the eight that the pack staffs has zero households after any
// tick (no `role_keep`-type floor is set). It is a claim about the count of the pack
// `payroll` in these four seeds, not about criterion 1 of `vision.md` 5.1 and not
// about the word "stable" (N-027, N-339): the households are flat, but 36% to 39%
// of the non-cook household-ticks are unfed (the prior is 1%), prices are about 40%
// of the reference and the cooks hold four to five times their share of the money,
// so acceptance criterion 4 is not met (N-352). On seeds 1 to 30 the same claim
// held on 25 of 30 (seeds 3, 8, 17, 22 and 27 lose about 19% of the households
// between tick 1 600 and 2 200, with the cooks holding 57% to 74% of the money), so
// the claim is partial and this test does not cover those seeds. Gate F test: it
// is skipped under -short (a 2 200 tick run is an integration test) and under
// -race (gate F runs it without -race, plan 2-1 D229).
//
// Requirement: FR-REF-045
func TestScenarioChain_Claim(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test (2 200 ticks, four seeds), skipped with -short")
	}
	if raceEnabled {
		t.Skip("gate F runs it without -race (plan 2-1 D229)")
	}
	cfg := loadValidPack(t, payrollWarm...)
	// The eight roles of the claim are those the pack gives households; the visitor role is closed and filled
	// only by external agents, so it has no household at the start and is not a role that can go extinct.
	staffed := map[string]bool{}
	for _, g := range cfg.Groups {
		if g.Households > 0 {
			staffed[g.Role] = true
		}
	}
	if len(staffed) != 8 {
		t.Fatalf("roles with households in the pack payroll = %d, want 8", len(staffed))
	}
	for seed := uint64(chainClaimFirstSeed); seed <= chainClaimLastSeed; seed++ {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			var at600, at1100, at2200 int64
			extinct := map[string]int64{}
			runScenario(t, cfg, scenarioSpec{
				Seed: seed, Ticks: chainClaimTicks,
				Snap: func(int64) bool { return true },
				OnSnap: func(tick int64, s economysim.Snapshot) {
					// The harness names a tick by the number of steps taken: after tick index i, At = i+1.
					switch tick + 1 {
					case 600:
						at600 = s.Households
					case 1100:
						at1100 = s.Households
					case chainClaimTicks:
						at2200 = s.Households
					}
					for role, r := range screenRoles(s, len(s.Names.Roles)) {
						if _, done := extinct[s.Names.Roles[role]]; !done && staffed[s.Names.Roles[role]] && r.Households == 0 {
							extinct[s.Names.Roles[role]] = tick + 1
						}
					}
				},
			})
			if at600 == 0 {
				t.Fatalf("seed %d: households after tick 600 = 0, want a population", seed)
			}
			for _, p := range []struct {
				tick, got int64
			}{{1100, at1100}, {chainClaimTicks, at2200}} {
				// within 10%: |got - at600| * 10 <= at600 (integers, no tolerance on money).
				if d := p.got - at600; d*10 > at600 || -d*10 > at600 {
					t.Errorf("seed %d: households after tick %d = %d, after tick 600 = %d, want within 10%%", seed, p.tick, p.got, at600)
				}
			}
			for role, tick := range extinct {
				t.Errorf("seed %d: role %s has zero households after tick %d, want none extinct", seed, role, tick)
			}
			t.Logf("seed %d: households %d after 600, %d after 1100, %d after 2200; extinct roles %v", seed, at600, at1100, at2200, extinct)
		})
	}
}
