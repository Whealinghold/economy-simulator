package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// hireForkConfig is the pack one-demand with the hire overlay, the cooks', diggers' and
// foresters' strategy the consumer's "hire_pack" and the recipe the cooks hire for set to
// 3 ticks, so that a hire is in progress at a boundary (the pack's own recipes take one
// tick, and a hire there forms and ends inside one tick).
func hireForkConfig(t *testing.T) (economysim.Config, hirePackStrategy) {
	t.Helper()
	cfg := loadPack(t, packOneDemandHire...)
	s := hirePackStrategyFor(t, cfg)
	for i := range cfg.Roles {
		switch cfg.Roles[i].ID {
		case "cook", "digger", "forester":
			cfg.Roles[i].Strategy = "hire_pack"
		}
	}
	cfg.Recipes[s.recipe].Duration = 3
	return cfg, s
}

// TestFork_EmploymentHire verifies FR-EMP-043 and the fork-at-N of CHK 8.1 with the hire
// form on: a world with hires of 3 ticks is checkpointed before any hire (N 0), in the
// tick of the first formations (N 1) and in the middle of the jobs (N 2 and 3), and the
// original, the restore and the restore of the restore give the same events, snapshot,
// audit and checkpoint bytes for the next ticks, with the logs compared. A restore that
// lost a hire, its Escrow account or the wages paid would differ at the first release
// or at the end of the hire.
//
// Requirement: FR-EMP-043, FR-EMP-044
//
// Guard: the case at N 2 or 3 fails if no hire was in progress at the boundary, so the
// test cannot pass on a world that never held one.
func TestFork_EmploymentHire(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg, s := hireForkConfig(t)
	for _, n := range []int{0, 1, 2, 3} {
		t.Run(fmt.Sprintf("N %d", n), func(t *testing.T) {
			t.Parallel()
			held := 0
			forkAtN(t, forkCase{
				Name: "hire pack", Cfg: cfg, Seed: 1, N: n, M: 12, Logs: true,
				Opts: func() []economysim.Option {
					return []economysim.Option{economysim.WithAudit(), economysim.WithStrategy("hire_pack", s)}
				},
				Before: func(w *economysim.World) {
					snap, err := w.Snapshot()
					if err != nil {
						t.Fatalf("Snapshot() = %v, want nil", err)
					}
					held = len(snap.Hires)
				},
			})
			if n >= 2 && held == 0 {
				t.Errorf("N %d: no hire in progress at the boundary, want at least one", n)
			}
		})
	}
}
