package economysim_test

import (
	"context"
	"os"
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
