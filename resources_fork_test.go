package economysim_test

import (
	"reflect"
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

// twoByTwoOre is the world of the multi-cell tests: two regions ("home" and "far")
// of ten households each, a mill that draws 3 units of "ore" and 2 of "wood" at once,
// and four endowments of different shapes and stocks, so that the rationed cells, the
// events and the saved section have more than one row in each of the two keys.
func twoByTwoOre() economysim.Config {
	sh := shapes()
	c := oreWorld(sh["refill"], 400)
	wood := sh["logistic"]
	wood.ID, wood.Capacity = "wood", 800
	far := economysim.DefaultRegion()
	far.ID = "far"
	c.Regions = append(c.Regions, far)
	c.Groups = append(c.Groups, economysim.Group{Region: "far", Role: "miller", Households: 10})
	c.Recipes[0].Draws = []economysim.Draw{{Resource: "ore", Quantity: 3}, {Resource: "wood", Quantity: 2}}
	m := c.Modules.Resources
	m.Resources = append(m.Resources, wood)
	m.Endowments = []economysim.Endowment{
		{Region: "far", Resource: "ore", HasInitial: true, Initial: 90},
		{Region: "far", Resource: "wood", HasInitial: true, Initial: 300},
		{Region: "home", Resource: "ore", HasInitial: true, Initial: 400},
		{Region: "home", Resource: "wood", HasInitial: true, Initial: 40},
	}
	return c
}

// TestIntegration_TwoRegionsTwoResources verifies the module through the public API
// over four endowments at once: the identity of each cell every tick (audit on), that a
// start draws all its resources or none (the units drawn of ore and wood in one region
// stand 3 to 2), that the EndowmentChanged events come in (region, resource) order, that
// two runs of one seed are identical, and that a fork at a boundary with the cells
// mid-regeneration runs on as the original does.
//
// Requirement: FR-RES-018, FR-RES-025, FR-RES-042, FR-RES-043
func TestIntegration_TwoRegionsTwoResources(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	run := func() []economysim.Event {
		rec := &recorder{}
		w, err := economysim.NewWorld(twoByTwoOre(), economysim.WithSeed(5), economysim.WithObserver(rec), economysim.WithAudit())
		if err != nil {
			t.Fatalf("NewWorld() error = %v, want nil", err)
		}
		if _, err := w.Run(t.Context(), 30); err != nil {
			t.Fatalf("Run(30) error = %v, want nil", err)
		}
		return rec.events
	}
	events := run()
	type key struct {
		tick             int64
		region, resource int
	}
	drawn := map[key]int64{}
	last := map[int64]key{}
	for _, e := range events {
		c, ok := e.(economysim.EndowmentChanged)
		if !ok {
			continue
		}
		k := key{c.Tick, int(c.Region), int(c.Resource)}
		drawn[k] = c.Drawn
		if p, seen := last[c.Tick]; seen && (k.region < p.region || (k.region == p.region && k.resource <= p.resource)) {
			t.Errorf("tick %d: EndowmentChanged for (%d, %d) after (%d, %d), want ascending (region, resource)", c.Tick, k.region, k.resource, p.region, p.resource)
		}
		last[c.Tick] = k
	}
	if len(drawn) == 0 {
		t.Fatalf("no EndowmentChanged in 30 ticks, want draws")
	}
	total := map[int]int64{}
	for k, n := range drawn {
		total[k.region] += n
	}
	if total[0] == 0 || total[1] == 0 {
		t.Errorf("units drawn by region = %v, want both regions to draw", total)
	}
	for k, ore := range drawn {
		// Resource 0 is ore and 1 is wood in canonical order; a start draws 3 and 2.
		if k.resource != 0 {
			continue
		}
		if wood := drawn[key{k.tick, k.region, 1}]; ore*2 != wood*3 {
			t.Errorf("tick %d region %d: ore drawn %d, wood drawn %d, want 3 : 2 (all or nothing)", k.tick, k.region, ore, wood)
		}
	}
	if again := run(); !reflect.DeepEqual(events, again) {
		t.Errorf("two runs of seed 5 differ in their events, want identical")
	}
	forkAtN(t, forkCase{Name: "two by two", Cfg: twoByTwoOre(), Seed: 5, N: 12, M: 15,
		Opts: func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} }})
}
