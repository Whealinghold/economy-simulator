package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s108Seeds is the full seed count of TestScenarioS108_PostedRentStaysInItsBand, reduced by
// invariantSeeds under -race.
const s108Seeds = 2

// s108Ticks is the length of a run and s108Shock the tick of the yield shock, halfway.
const (
	s108Ticks = 160
	s108Shock = 80
)

// TestScenarioS108_PostedRentStaysInItsBand verifies LND-S2 of spec-land.md 16.4: the warm five
// with the binding resources overlay and the tenure overlay, the posted rent stepping by
// 2 500 basis points inside the band 5 000 to 15 000 of its configured rent, and a yield shock
// that halves the recipes drawing on the arable of north at the tick s108Shock. The posted rent
// stays inside its band every tick, each RentPaid obeys the rule of the step on its own counts
// (rises by one step while units are unserved and none was cut, falls while let units lie
// idle, holds otherwise), and the audit holds. Where the rent settles, the cycle against
// 3 * step * (delay + 1) / 4 (LND-A1, LND-D2) and whether it reaches a clearing rent are
// reported, not asserted: the pack collapses (criterion 1, N-352).
//
// Arithmetic: the configured rent is 2, so the step is max(1, 2 * 2 500 / 10 000) = 1, the
// floor 2 * 5 000 / 10 000 = 1 and the ceiling 2 * 15 000 / 10 000 = 3.
//
// Requirement: FR-LND-019
func TestScenarioS108_PostedRentStaysInItsBand(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	for _, seed := range seedList(invariantSeeds(s108Seeds)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			s108Run(t, seed)
		})
	}
}

// s108Pack is the golden pack with the band and the step of the scenario.
func s108Pack(t *testing.T) economysim.Config {
	t.Helper()
	cfg := loadValidPack(t, packFiveTenure...)
	m := *cfg.Modules.Tenure
	m.RentStepBP, m.RentFloorBP, m.RentCeilingBP = 2_500, 5_000, 15_000
	cfg.Modules.Tenure = &m
	return cfg
}

// s108Recipes are the recipes of the configuration that draw on the resource, by ID in the
// names table of the world.
func s108Recipes(t *testing.T, cfg economysim.Config, w *economysim.World, resource string) []economysim.RecipeID {
	t.Helper()
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	var out []economysim.RecipeID
	for _, r := range cfg.Recipes {
		if slices.ContainsFunc(r.Draws, func(d economysim.Draw) bool { return d.Resource == resource }) {
			out = append(out, economysim.RecipeID(slices.Index(snap.Names.Recipes, r.ID)))
		}
	}
	return out
}

// s108Run is one seed of S108.
func s108Run(t *testing.T, seed uint64) {
	t.Helper()
	cfg := s108Pack(t)
	rec := &recorder{}
	w := mustWorld(t, cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(rec))
	g, r := s111Endowment(t, w)
	for tick := range s108Ticks {
		if tick == s108Shock {
			var shocks []economysim.YieldShock
			for _, id := range s108Recipes(t, cfg, w, "arable") {
				shocks = append(shocks, economysim.YieldShock{Region: g, Recipe: id, BP: 5_000})
			}
			if err := w.SetYieldShocks(shocks); err != nil {
				t.Fatalf("seed %d: SetYieldShocks error = %v, want nil", seed, err)
			}
		}
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step error = %v, want nil (the audit failed or the step did)", seed, err)
		}
	}
	s108Check(t, seed, g, r, rec.events)
}

// s108Check holds each RentPaid of the endowment to the rule of the step and the band, and
// reports where the posted rent went.
func s108Check(t *testing.T, seed uint64, g economysim.RegionID, r economysim.ResourceID, events []economysim.Event) {
	t.Helper()
	lo, hi, rises, falls := int64(1<<62), int64(0), int64(0), int64(0)
	for _, e := range events {
		x, ok := e.(economysim.RentPaid)
		if !ok || x.Region != g || x.Resource != r {
			continue
		}
		if want := s108Rule(x, 1, 1, 3); x.RentAfter != want {
			t.Errorf("seed %d, tick %d: rent %d, unserved %d, cut %d, idle %d gave %d, want %d by the rule of the step (band 1 to 3, step 1)", seed, x.Tick, x.Rent, x.Unserved, x.CutUnits, x.Idle, x.RentAfter, want)
		}
		lo, hi = min(lo, int64(x.RentAfter)), max(hi, int64(x.RentAfter))
		rises, falls = rises+btoi(x.RentAfter > x.Rent), falls+btoi(x.RentAfter < x.Rent)
	}
	if lo < 1 || hi > 3 {
		t.Errorf("seed %d: the posted rent went from %d to %d, want it inside the band 1 to 3", seed, lo, hi)
	}
	t.Logf("seed %d: posted rent from %d to %d, %d rises and %d falls in %d ticks, shock at %d; half-span bound 3*1*(delay+1)/4 reported with delay unknown (not asserted: N-352)", seed, lo, hi, rises, falls, s108Ticks, s108Shock)
}

// s108Rule is the posted rent after the step of spec-land.md 5.5.1 for the counts of an
// event, in the band floor to ceil with the given step; the rent in force is the posted rent
// (the scale is 10 000).
func s108Rule(x economysim.RentPaid, step, floor, ceil int64) economysim.Money {
	switch {
	case x.Unserved > 0 && x.CutUnits == 0:
		return min(economysim.Money(ceil), x.Rent+economysim.Money(step))
	case x.Idle > 0:
		return max(economysim.Money(floor), x.Rent-economysim.Money(step))
	}
	return x.Rent
}
