package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestScenarioS43_FiniteDeposit verifies RES-S3 (spec-resources.md 19.4) on a
// world whose reapers draw a finite deposit and eat what they draw: the tick at
// which the deposit runs short agrees with endowment_runway of SolveSteadyState
// within s43.runway_bp (10%), the deposit never regrows, and from that tick the
// households that depended on it go unfed and are removed by the ladder, so the
// output of the chain stops.
//
// Arithmetic: 20 reapers draw 1 unit each a tick, 20 a tick, and the reference
// programme draws the same (a yield of 1 grain per execution against an upkeep of
// 1), so a deposit of 6 000 lasts 6 000 / 20 = 300 ticks (4.1 years of 73 ticks).
// The horizon is 3 years, so the sustainable draw is 6 000 / 3 = 2 000 a year =
// 27.4 a tick and covers the 20 a tick. The deposit has no abundance curve, so the
// yield stays flat until the last unit (spec-resources.md 16.2).
//
// Requirement: FR-RES-040, FR-RES-028, FR-RES-031
//
// Guard: the runway is what the viability check promises a finite stock; a run that
// ran out at another time would mean the check and the run disagree about the draw.
func TestScenarioS43_FiniteDeposit(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const households, stock, horizon, ticks = 20, 6_000, 3, 420
	res := economysim.DefaultResource()
	res.Shape = economysim.ShapeConstant
	cfg := fieldWorld(res, stock, stock, households, 6, 1)
	cfg.Modules.Resources.HorizonYears = horizon
	ss, err := economysim.SolveSteadyState(t.Context(), cfg)
	if err != nil {
		t.Fatalf("SolveSteadyState() error = %v, want nil", err)
	}
	if len(ss.Report.Runway) != 1 {
		t.Fatalf("SolveSteadyState() Runway = %+v, want one row (a constant stock with no rate)", ss.Report.Runway)
	}
	runway := ss.Report.Runway[0].Ticks
	for _, seed := range seedList(scenarioSeedCount()) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			r := newFieldRun(t, cfg, seed)
			var regrown, lastRecipes, recipesAfter int64
			prev, exhausted := int64(stock), int64(-1)
			for range ticks {
				r.step(t, seed)
				regrown += max(0, r.stock()-prev)
				prev = r.stock()
				if exhausted < 0 && prev == 0 {
					exhausted = r.w.Tick()
				}
			}
			if lo, hi := runway*(10_000-s43RunwayBP)/10_000, runway*(10_000+s43RunwayBP)/10_000; exhausted < lo || exhausted > hi {
				t.Errorf("seed %d: the deposit was spent at tick %d, want %d within %d bp (endowment_runway)", seed, exhausted, runway, s43RunwayBP)
			}
			if regrown != 0 {
				t.Errorf("seed %d: the deposit regrew %d units, want 0 (a constant stock with rate 0, FR-RES-031)", seed, regrown)
			}
			for _, e := range eventsAs[economysim.RecipeExecuted](r.rec.events) {
				if e.Tick >= exhausted {
					recipesAfter += int64(e.Executions)
				}
				lastRecipes = e.Tick
			}
			n, firstStarved := removedStarved(r.rec.events)
			if recipesAfter != 0 || n != households || firstStarved < exhausted {
				t.Errorf("seed %d: executions from tick %d on = %d (last at tick %d), starved removals = %d (first at tick %d), want 0 executions and %d households removed after the deposit was spent", seed, exhausted, recipesAfter, lastRecipes, n, firstStarved, households)
			}
			// The households' view caps their executions at what the stock covers, so
			// they ask nothing of a spent deposit: no tick is short and no line is due.
			if got, want := r.warns(), wantWarns(r.changed()); got != want || want != 0 {
				t.Errorf("seed %d: Warn lines = %d, want %d = 0 (the view stops the requests, so nothing is rationed short)", seed, got, want)
			}
		})
	}
}
