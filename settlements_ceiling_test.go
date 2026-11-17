package economysim_test

import (
	"math/rand/v2"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// ceilingRecorder is a spawner that records the view of one role in region 0 at each
// tick and delegates to the default.
type ceilingRecorder struct {
	role  economysim.RoleID
	views []economysim.RoleView
}

func (c *ceilingRecorder) Decide(v *economysim.RegionView, rng *rand.Rand, d *economysim.Decisions) {
	if v.Region == 0 {
		c.views = append(c.views, v.Roles[c.role])
	}
	economysim.DefaultSpawner{}.Decide(v, rng, d)
}

// TestCeiling_ViewAndEntriesOverSeeds verifies, through the public API on the reference
// chain with the module on, that a role with a ceiling sees its headroom as the class
// limit less its households (never below 0), that a role with no ceiling sees no limit,
// and that the role never holds more households than the larger of its start and the
// limit: the spawner and the population module refuse what does not fit, and remove
// nobody.
//
// Arithmetic: the limit is the grower's households at the start, so the headroom at the
// start is 0 and any entry would exceed the limit.
//
// Requirement: FR-SET-023, FR-SPN-056
func TestCeiling_ViewAndEntriesOverSeeds(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, seed := range []uint64{1, 2, 3} {
		c := referenceConfig()
		limit := groupHouseholds(c, "north", "grower")
		m := economysim.DefaultSettlements()
		m.Classes = []economysim.SettlementClass{{ID: "only", Limits: []economysim.ClassLimit{{Name: "places", Value: limit}}}}
		c.Modules.Settlements = &m
		roleOf(&c, "grower").Ceiling = "places"
		grower := setRoleID(t, c, "grower")
		rec := &ceilingRecorder{role: grower}
		c.World.Spawner = "ceil"
		w, err := economysim.NewWorld(c, economysim.WithSeed(seed), economysim.WithSpawner("ceil", rec))
		if err != nil {
			t.Fatalf("seed %d: NewWorld() error = %v, want nil", seed, err)
		}
		for tick := range 40 {
			if err := w.Step(); err != nil {
				t.Fatalf("seed %d: Step() at tick %d = %v, want nil", seed, tick, err)
			}
		}
		if len(rec.views) == 0 {
			t.Fatalf("seed %d: the spawner saw no view of region 0", seed)
		}
		checkCeilingViews(t, seed, rec.views, limit)
	}
}

// groupHouseholds is the households of the group of the region and role in c.
func groupHouseholds(c economysim.Config, region, role string) int64 {
	for _, g := range c.Groups {
		if g.Region == region && g.Role == role {
			return g.Households
		}
	}
	return 0
}

// checkCeilingViews asserts the headroom of every view and that the role never holds more
// than the limit.
func checkCeilingViews(t testing.TB, seed uint64, views []economysim.RoleView, limit int64) {
	t.Helper()
	for i, v := range views {
		if want := max(limit-v.Households, 0); v.NoLimit || v.Headroom != want {
			t.Errorf("seed %d: view %d: Headroom %d NoLimit %v with %d households, want %d, false", seed, i, v.Headroom, v.NoLimit, v.Households, want)
		}
		if v.Households > limit {
			t.Errorf("seed %d: view %d: the grower holds %d households, want at most the limit %d (nothing enters above it)", seed, i, v.Households, limit)
		}
	}
}
