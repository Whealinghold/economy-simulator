package economysim_test

import (
	"context"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// payrollOffPacks are the packs whose solver output the payroll-aware solver
// must leave byte-equal (plan 2-29 D3): the cold layers the solver reads and the
// warm document of the pack, where it has one (`all` keeps the warm start of
// `five`, so only its report is checked).
var payrollOffPacks = []struct {
	name       string
	cold, warm []string
}{
	{"one", []string{"base.toml", "one.toml"}, []string{"base.toml", "one.toml", "warm-one.toml"}},
	{"one-demand", packOneDemandCold, packOneDemand},
	{"two", []string{"base.toml", "two.toml"}, []string{"base.toml", "two.toml", "warm-two.toml"}},
	{"three", []string{"base.toml", "three.toml"}, []string{"base.toml", "three.toml", "warm-three.toml"}},
	{"five", []string{"base.toml", "lines.toml", "five.toml"}, []string{"base.toml", "lines.toml", "five.toml", "warm-five.toml"}},
	{"famine", packLayers["famine"], append(slices.Clone(packLayers["famine"]), "warm-famine.toml")},
	{"second", []string{"second/second.toml"}, []string{"second/second.toml", "second/warm-second.toml"}},
	{"all", nil, nil},
}

// TestSolveSteadyState_PayrollOffDifferential is the off proof of plan 2-29
// Phase 3: for every existing pack (none turns on the employment module) the
// table the solver makes equals the committed warm document of the pack field for
// field, and the report carries no payroll diagnostic and no payroll column. The
// committed documents, `golden/off/` and the pinned hash are not regenerated.
//
// Requirement: FR-VIA-068, NFR-VIA-005
func TestSolveSteadyState_PayrollOffDifferential(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, tt := range payrollOffPacks {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cold := tt.cold
			if tt.name == "all" {
				cold = allColdLayers()
			}
			cfg := loadPack(t, cold...)
			if cfg.Modules.Employment != nil {
				t.Fatalf("pack %s turns the employment module on, want off", tt.name)
			}
			st, err := economysim.SolveSteadyState(context.Background(), cfg)
			if err != nil {
				t.Fatalf("SolveSteadyState(%s) = %v, want nil", tt.name, err)
			}
			if tt.warm != nil {
				want := loadWarm(t, tt.warm...)
				if tt.name == "famine" {
					// The document of famine is not the solver's output on main either (its two maker
					// job rows read 2 272 and 8 522 there; the solver gives 2 104 and 7 895, measured
					// on origin/main df4cea7 before this change), so the differential pins main's rows.
					want = setJob(setJob(want, "north", "maker", "make_tool_a", 2_104), "north", "maker", "make_tool_b", 7_895)
				}
				if !reflect.DeepEqual(st.Table, want) {
					t.Errorf("table of %s differs from its warm document\ngot  %+v\nwant %+v", tt.name, st.Table, want)
				}
			}
			if st.Report.PayrollInfeasible != nil {
				t.Errorf("PayrollInfeasible of %s = %+v, want nil with the module off", tt.name, st.Report.PayrollInfeasible)
			}
			for _, r := range st.Report.Census {
				if r.PayrollOutBP != 0 || r.PayrollInBP != 0 {
					t.Errorf("census row %s/%s of %s has payroll columns %d, %d, want 0", r.Role, r.Recipe, tt.name, r.PayrollOutBP, r.PayrollInBP)
				}
			}
		})
	}
}

// TestSolveSteadyState_PayrollOn verifies the payroll-aware solve on one-demand
// with overlay/employment: the primal and the balanced census are those of the
// pack without the module, only the duals move (the cooks pay, every other role is
// paid, the seed prices differ), the residual of the net-wage equations at the
// integer seeds is small, nothing is infeasible, and the table passes Validate.
//
// Requirement: FR-VIA-064, FR-VIA-065, FR-VIA-067, FR-VIA-069
func TestSolveSteadyState_PayrollOn(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	ctx := context.Background()
	off, err := economysim.SolveSteadyState(ctx, loadPack(t, packOneDemandCold...))
	if err != nil {
		t.Fatalf("SolveSteadyState(one-demand) = %v, want nil", err)
	}
	cfg := loadPack(t, packOneDemandEmploymentCold...)
	on, err := economysim.SolveSteadyState(ctx, cfg)
	if err != nil {
		t.Fatalf("SolveSteadyState(one-demand + employment) = %v, want nil", err)
	}
	if on.Report.PayrollInfeasible != nil {
		t.Fatalf("PayrollInfeasible = %+v, want none", on.Report.PayrollInfeasible)
	}
	if !reflect.DeepEqual(on.Programmes, off.Programmes) {
		t.Errorf("programmes with payroll = %+v, want those without %+v: the primal does not change", on.Programmes, off.Programmes)
	}
	if len(on.Report.Census) != len(off.Report.Census) {
		t.Fatalf("census rows = %d, want %d", len(on.Report.Census), len(off.Report.Census))
	}
	for i, r := range on.Report.Census {
		o := off.Report.Census[i]
		if r.Role != o.Role || r.Households != o.Households || r.Balanced != o.Balanced || r.Imbalance != o.Imbalance {
			t.Errorf("census row %s = %d/%d/%d, want %d/%d/%d: the balanced census does not change", r.Role, r.Households, r.Balanced, r.Imbalance, o.Households, o.Balanced, o.Imbalance)
		}
		switch {
		case r.Role == "cook" && (r.PayrollOutBP <= 0 || r.PayrollInBP != 0):
			t.Errorf("cook payroll columns out %d in %d, want out > 0 and in 0", r.PayrollOutBP, r.PayrollInBP)
		case r.Role != "cook" && (r.PayrollInBP <= 0 || r.PayrollOutBP != 0):
			t.Errorf("%s payroll columns out %d in %d, want in > 0 and out 0", r.Role, r.PayrollOutBP, r.PayrollInBP)
		}
	}
	if reflect.DeepEqual(on.Table.Prices, off.Table.Prices) {
		t.Errorf("seed prices with payroll equal those without: the duals were not re-solved")
	}
	if on.Report.ResidualBP > 200 {
		t.Errorf("ResidualBP = %d, want at most 200: the seeds at the integer prices satisfy the net-wage equations", on.Report.ResidualBP)
	}
	cfg.WarmStart = &on.Table
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate(one-demand + employment with its table) = %v, want nil", err)
	}
	again, err := economysim.SolveSteadyState(ctx, cfg)
	if err != nil || !reflect.DeepEqual(again.Table, on.Table) || !reflect.DeepEqual(again.Report, on.Report) {
		t.Errorf("a second solve differs from the first (err %v): the solver must be a function of the configuration", err)
	}
}

// TestSolveSteadyState_PayrollReports verifies that a share the chain cannot bear
// and a recipe shared with another role keep the payroll-free duals and are named
// in the report, never an error and never a changed census.
//
// Requirement: FR-VIA-067, FR-VIA-069
func TestSolveSteadyState_PayrollReports(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	ctx := context.Background()
	base := loadPack(t, packOneDemandEmploymentCold...)
	free, err := economysim.SolveSteadyState(ctx, loadPack(t, packOneDemandCold...))
	if err != nil {
		t.Fatalf("SolveSteadyState(one-demand) = %v, want nil", err)
	}
	roleAt := func(cfg economysim.Config, id string) int {
		for i := range cfg.Roles {
			if cfg.Roles[i].ID == id {
				return i
			}
		}
		t.Fatalf("role %q not in the pack", id)
		return -1
	}
	t.Run("share too high", func(t *testing.T) {
		t.Parallel()
		cfg := base
		cfg.Roles = slices.Clone(base.Roles)
		cfg.Roles[roleAt(cfg, "cook")].Employer = &economysim.Employer{PayrollBP: 10_000}
		st, err := economysim.SolveSteadyState(ctx, cfg)
		if err != nil {
			t.Fatalf("SolveSteadyState(payroll 10000) = %v, want nil: a payroll diagnostic is not an error", err)
		}
		if len(st.Report.PayrollInfeasible) != 1 || st.Report.PayrollInfeasible[0].Region != "north" {
			t.Fatalf("PayrollInfeasible = %+v, want one record for north", st.Report.PayrollInfeasible)
		}
		t.Logf("reason at 10 000 bp: %s", st.Report.PayrollInfeasible[0].Reason)
		if !reflect.DeepEqual(st.Table, free.Table) {
			t.Errorf("a kept payroll-free solution must give the payroll-free table")
		}
		for _, r := range st.Report.Census {
			if r.PayrollOutBP != 0 || r.PayrollInBP != 0 {
				t.Errorf("census row %s has payroll columns %d, %d, want 0 where the duals were kept", r.Role, r.PayrollOutBP, r.PayrollInBP)
			}
		}
	})
	t.Run("recipe shared with another role", func(t *testing.T) {
		t.Parallel()
		cfg := base
		cfg.Roles = slices.Clone(base.Roles)
		cook := roleAt(cfg, "cook")
		f := roleAt(cfg, "forester")
		cfg.Roles[f].Recipes = append(slices.Clone(cfg.Roles[f].Recipes), cfg.Roles[cook].Recipes[0])
		st, err := economysim.SolveSteadyState(ctx, cfg)
		if err != nil {
			t.Fatalf("SolveSteadyState(shared recipe) = %v, want nil", err)
		}
		if len(st.Report.PayrollInfeasible) != 1 || st.Report.PayrollInfeasible[0].Reason != "unsupported" {
			t.Errorf("PayrollInfeasible = %+v, want one record, reason unsupported", st.Report.PayrollInfeasible)
		}
	})
}
