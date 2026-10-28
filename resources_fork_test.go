package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// forkOre is the world of the fork tests: ten households whose recipe draws 3 units
// of a slow refill stock (500 basis points a year, so the carry of the regeneration is
// pending between units), regenerating in summer only (the other seasons are fallow),
// with jobs of three ticks.
func forkOre() economysim.Config {
	res := economysim.DefaultResource()
	res.Shape, res.RateBP = economysim.ShapeRefill, 500
	res.Seasons = []economysim.SeasonRegen{{Season: "summer", BP: 30_000}}
	c := oreWorld(res, 600)
	c.Recipes[0].Duration = 3
	return c
}

// TestFork_Resources verifies that a world forked at tick N, with a stock mid-
// regeneration, a pending carry and a shock scale, runs on exactly as an
// uninterrupted one in every event and in the snapshot: N 0, inside a job, inside a
// fallow season (spring, days 59 to 150), inside the regenerating season (summer,
// from day 151) and after a call between ticks.
//
// Requirement: FR-RES-042
func TestFork_Resources(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, c := range []struct {
		name   string
		n      int
		before func(*economysim.World)
	}{
		{"tick 0", 0, nil},
		{"inside a job", 4, nil},
		{"inside the fallow season", 100, nil},
		{"inside the regenerating season", 170, nil},
		{"after a call between ticks", 20, func(w *economysim.World) {
			economysim.ShockRegen(w, 0, 0, 3_000)
			economysim.ShockCapacity(w, 0, 0, 15_000)
			economysim.AdjustStock(w, 0, 0, -100)
		}},
	} {
		forkAtN(t, forkCase{Name: "ore " + c.name, Cfg: forkOre(), Seed: 2, N: c.n, M: forkM, Before: c.before,
			Opts: func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} }})
	}
}

// TestFork_ShockBeforeCheckpoint verifies that a shock scale and an adjustment made
// just before the checkpoint are saved: the restored world's snapshot rows have the
// scales and the stock, and it runs on as the original does.
//
// The capacity here is only raised: the units a clamp loses belong to the tick that
// follows the call (spec-resources.md 6.4) and are not part of the saved state, so
// the first EndowmentChanged of a restored world would not show them.
//
// Requirement: FR-RES-042
func TestFork_ShockBeforeCheckpoint(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	forkAtN(t, forkCase{Name: "ore shock", Cfg: forkOre(), Seed: 4, N: 12, M: forkM,
		Before: func(w *economysim.World) {
			economysim.ShockRegen(w, 0, 0, 7_000)
			economysim.ShockCapacity(w, 0, 0, 20_000)
			economysim.AdjustStock(w, 0, 0, 40)
		},
		Inspect: func(t *testing.T, restored, since []economysim.Event) {
			if len(since) == 0 {
				t.Error("the reference world delivered no event after the boundary, want some")
			}
		}})
}
