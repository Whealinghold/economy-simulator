package economysim_test

import (
	"context"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packFiveWarm is the five-region pack with its warm start, the layers the two
// five overlays of the resources module sit on.
var packFiveWarm = []string{"base.toml", "lines.toml", "five.toml", "warm-five.toml"}

// resourceOverlays are the four overlays of the resources module (plan 2-9 D398):
// the layers they sit on, the rows they must report and the headroom they were
// sized for, in basis points of the draw of the reference programme.
var resourceOverlays = []struct {
	name   string
	layers []string
	rows   int
	target int64
}{
	{"resources", packOne, 3, 22_000},
	{"resources-binding", packOne, 3, 11_000},
	{"resources-five", packFiveWarm, 15, 22_000},
	{"resources-five-binding", packFiveWarm, 15, 11_000},
}

// TestOverlays_ResourcesSizedFromTheSolver verifies that each resources overlay
// loads and validates over its pack and that the solver reports, for every
// endowment, the headroom the overlay was sized for: at least 2.2 times the draw
// of the reference programme for the slack files and 1.1 times for the binding
// ones, and at most 5% more (a capacity is rounded up to a whole unit, and the
// smallest, 616 units of arable, moves the headroom by 0.2%; the others differ
// because the headroom is floored where the solver is probed); that no row binds
// in a slack file's resource_short sense (Validate would refuse it); and that the
// stock of each renewing resource starts at the plan share (initial_bp equals
// plan_share_bp; a constant stock has no plan share, its sustainable draw is the
// stock over the horizon).
//
// Arithmetic: each capacity is 1e9 times the target over the headroom the
// solver reported at 1e9 (the headroom is linear in the capacity at a fixed plan
// share), so the headroom of the file is the target up to the rounding up.
//
// Requirement: FR-RES-039, FR-RES-040
//
// Guard: a re-derived pack (a changed census or draw) that leaves a capacity from
// the old programme would silently turn a slack overlay into a binding one.
func TestOverlays_ResourcesSizedFromTheSolver(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, tc := range resourceOverlays {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, append(slices.Clone(tc.layers), "overlay/"+tc.name+".toml")...)
			ss, err := economysim.SolveSteadyState(context.Background(), cfg)
			if err != nil {
				t.Fatalf("SolveSteadyState(%s) error = %v, want nil", tc.name, err)
			}
			if got := len(ss.Report.Headroom); got != tc.rows {
				t.Fatalf("SolveSteadyState(%s) Headroom rows = %d, want %d (one per region and resource)", tc.name, got, tc.rows)
			}
			for _, h := range ss.Report.Headroom {
				if h.BP < tc.target || h.BP > tc.target*105/100 {
					t.Errorf("SolveSteadyState(%s) headroom of %s in %s = %d bp, want %d to %d", tc.name, h.Resource, h.Region, h.BP, tc.target, tc.target*105/100)
				}
			}
			for _, r := range cfg.Modules.Resources.Resources {
				if r.Shape != economysim.ShapeConstant && r.InitialBP != r.PlanShareBP {
					t.Errorf("%s: resource %s initial_bp = %d, want plan_share_bp %d (spec-resources.md 11.4)", tc.name, r.ID, r.InitialBP, r.PlanShareBP)
				}
			}
		})
	}
}
