package economysim_test

import (
	"context"
	"os"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// TestPacks_TenureOverlayLoadsAndSolves verifies that overlay/tenure.toml, layered after
// overlay/resources.toml, loads and validates on pack one and on pack five, that the
// module is on with the rows the overlay says, and that the steady-state solver still
// solves the layers (the programme is unchanged by tenure, spec-land.md 12.1).
//
// Requirement: FR-LND-001, FR-LND-038, FR-LND-040
func TestPacks_TenureOverlayLoadsAndSolves(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	packs := map[string][]string{
		"one":  {"base.toml", "one.toml", "warm-one.toml", "overlay/resources.toml", "overlay/tenure.toml"},
		"five": {"base.toml", "lines.toml", "five.toml", "warm-five.toml", "overlay/resources-five.toml", "overlay/tenure.toml"},
	}
	for name, layers := range packs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg, err := config.LoadFS(os.DirFS("testdata/chains"), layers...)
			if err != nil {
				t.Fatalf("LoadFS(%v) = %v, want nil", layers, err)
			}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate(%v) = %v, want nil", layers, err)
			}
			if cfg.Modules.Tenure == nil || len(cfg.Modules.Tenure.Holdings) != 2 || len(cfg.Modules.Tenure.Terms) != 1 {
				t.Fatalf("the tenure module of %s = %+v, want one term and two holdings", name, cfg.Modules.Tenure)
			}
			if _, err := economysim.SolveSteadyState(context.Background(), cfg); err != nil {
				t.Errorf("SolveSteadyState(%s with tenure) = %v, want nil: it validates", name, err)
			}
		})
	}
}

// TestSolver_RentOffDualReportsRentAwayFromTheDual verifies FR-LND-040 and spec-land.md
// 5.6.2: the steady-state solver reports Rent for each endowment, and with the tenure module
// on it lists in Report.RentOffDual the endowments whose configured rent is more than 50% of
// the dual away from it, or above 0 on a slack row, and none when the rent is the dual.
//
// Arithmetic: the stand of north binds (its dual is read from the report of the run); the
// overlay's arable of north is slack (dual 0) with a rent of 2, so it is always a row. The
// rent of the stand is set to the dual (no row), to the dual plus half of it (the edge of
// the tolerance of 5 000 basis points, no row) and to twice the dual (off by 100%: a row).
//
// Requirement: FR-LND-040
func TestSolver_RentOffDualReportsRentAwayFromTheDual(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	layers := []string{"base.toml", "lines.toml", "five.toml", "warm-five.toml", "overlay/resources-five-binding.toml", "overlay/tenure.toml"}
	load := func(rent int64) economysim.Config {
		cfg, err := config.LoadFS(os.DirFS("testdata/chains"), layers...)
		if err != nil {
			t.Fatalf("LoadFS(%v) = %v, want nil", layers, err)
		}
		m := cfg.Modules.Tenure
		m.Terms = append(m.Terms, economysim.TenureTerm{Region: "north", Resource: "stand", HasRent: true, Rent: rent})
		m.Holdings = append(m.Holdings, economysim.Holding{Region: "north", Resource: "stand", Holder: economysim.HolderRef{Kind: economysim.HolderKindPool}, ShareBP: 1_000})
		return cfg
	}
	solve := func(rent int64) economysim.Report {
		ss, err := economysim.SolveSteadyState(context.Background(), load(rent))
		if err != nil {
			t.Fatalf("SolveSteadyState(rent %d) = %v, want nil", rent, err)
		}
		return ss.Report
	}
	var dual int64
	for _, r := range solve(1).Rent {
		if r.Region == "north" && r.Resource == "stand" {
			dual = r.Rent
		}
	}
	if dual <= 0 {
		t.Fatalf("the dual of the stand of north = %d, want a binding row (above 0)", dual)
	}
	off := func(rep economysim.Report, resource string, rent, dual int64) bool {
		return slices.ContainsFunc(rep.RentOffDual, func(r economysim.RentOffDualRow) bool {
			return r.Region == "north" && r.Resource == resource && r.Rent == rent && r.Dual == dual
		})
	}
	for _, tt := range []struct {
		name string
		rent int64
		want bool
	}{{"the dual", dual, false}, {"within the tolerance", dual + dual/2, false}, {"twice the dual", 2 * dual, true}} {
		t.Run(tt.name, func(t *testing.T) {
			rep := solve(tt.rent)
			if got := off(rep, "stand", tt.rent, dual); got != tt.want {
				t.Errorf("stand with rent %d against the dual %d: reported = %v, want %v (rows %+v)", tt.rent, dual, got, tt.want, rep.RentOffDual)
			}
			if !off(rep, "arable", 2, 0) {
				t.Errorf("arable with rent 2 on a slack row: not reported, want a row (rows %+v)", rep.RentOffDual)
			}
		})
	}
}
