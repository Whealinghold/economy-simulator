package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s44 sizes. A refill stock of capacity 20 000 growing 1 000% a year (the bound
// MaxRegenBP), a year of 73 ticks, so 13.7% of the gap a tick, viable for 1 000
// households that draw 1 each a tick. A blight before the first tick lowers its
// capacity to 3 650 (a scale of 1 825 bp): an empty stock regrows 13.7% of 3 650 =
// 500 a tick, half what the 1 000 households ask, so the ration serves half of
// the demand every tick once the first stock is spent.
const (
	s44Households = 1_000
	s44Ticks      = 120
	// s44Warmup is the ticks before the measure: the capacity clamp leaves 3 650
	// units, which the first four ticks spend.
	s44Warmup  = 10
	s44Blight  = 1_825
	s44Cohort  = 50
	s44Cohorts = s44Households / s44Cohort
)

// grantStats are what one agent was asked and granted over the measured ticks.
type grantStats struct{ asked, granted, run, longest int64 }

// s44Run runs 1 000 households as agents of weight scale (1 000 agents of weight 1,
// or 20 cohorts of weight 50) on the blighted stock and returns, for each agent,
// its asked and granted executions and its longest run of ticks granted nothing,
// the number of requests granted less than floor(w * A / U) in a tick, and the run
// (for its Warn lines). The first s44Warmup ticks are not measured.
func s44Run(t *testing.T, seed uint64, scale int64) (stats map[economysim.AgentID]*grantStats, cohortShort int, r *fieldRun) {
	t.Helper()
	res := economysim.DefaultResource()
	res.Shape, res.RateBP = economysim.ShapeRefill, 100_000
	cfg := fieldWorld(res, 20_000, 20_000, s44Households, 500, 1)
	cfg.World.Scale, cfg.World.MinAgents = scale, s44Households/scale
	r = newFieldRun(t, cfg, seed)
	economysim.ShockCapacity(r.w, 0, 0, s44Blight)
	stats = map[economysim.AgentID]*grantStats{}
	for tick := range s44Ticks {
		var asked, granted int64
		var grants, wants []int64
		err := economysim.StepGrants(r.w, func(a economysim.AgentID, ask, g int64) {
			asked += ask
			granted += g
			grants, wants = append(grants, g), append(wants, ask)
			if tick < s44Warmup {
				return
			}
			c := stats[a]
			if c == nil {
				c = &grantStats{}
				stats[a] = c
			}
			c.asked += ask
			c.granted += g
			if g == 0 {
				c.run++
				c.longest = max(c.longest, c.run)
			} else {
				c.run = 0
			}
		})
		if err != nil {
			t.Fatalf("seed %d: StepGrants() at tick %d error = %v, want nil", seed, tick, err)
		}
		// FR-RES-021: a request is granted at least floor(w * A / U); when the
		// stock is short the units granted are the stock, A = granted.
		if asked > granted {
			for i := range grants {
				if floor := wants[i] * granted / asked; grants[i] < floor {
					cohortShort++
				}
			}
		}
	}
	return stats, cohortShort, r
}

// s44Share is the share of the executions asked that were granted, in basis
// points, over every agent measured.
func s44Share(stats map[economysim.AgentID]*grantStats) (share, longest int64) {
	var asked, granted int64
	for _, c := range stats {
		asked += c.asked
		granted += c.granted
		longest = max(longest, c.longest)
	}
	return granted * 10_000 / asked, longest
}

// TestScenarioS44_RationWithCohorts verifies RES-S4 (spec-resources.md 19.4): a
// thousand households as 1 000 agents of weight 1 and as 20 cohorts of weight 50
// ask a blighted refill stock for twice what it regrows; the ration serves half of
// the demand, the same half to the units and to the cohorts (within
// s44.grant_band_bp of the mean), no agent is granted nothing for more than
// s44.denied_ticks ticks, every request gets at least floor(w * A / U), and the
// Warn is bounded by its period.
//
// Arithmetic: see the constants. A stock of 500 units (an empty stock regrows 13.7%
// of the capacity 3 650) against 1 000 requests of 1 unit gives 500 grants of 1 to
// the units, chosen by the stateless key of each tick (so every agent is served
// about half the ticks), and 25 to each cohort of 50.
//
// Requirement: FR-RES-017, FR-RES-020, FR-RES-021, FR-RES-043, FR-RES-028
//
// Guard: a ration that kept a memory of past draws, or ordered by agent ID, would
// deny the same agents for long runs; one that was not weight-invariant would serve
// cohorts and units differently.
func TestScenarioS44_RationWithCohorts(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	for _, seed := range seedList(scenarioSeedCount()) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			units, unitShort, unitRun := s44Run(t, seed, 1)
			cohorts, cohortShort, cohortRun := s44Run(t, seed, s44Cohort)
			if len(units) != s44Households || len(cohorts) != s44Cohorts {
				t.Fatalf("seed %d: agents = %d units and %d cohorts, want %d and %d", seed, len(units), len(cohorts), s44Households, s44Cohorts)
			}
			us, ul := s44Share(units)
			cs, cl := s44Share(cohorts)
			if us < 5_000-5_000*s44GrantBandBP/10_000 || us > 5_000+5_000*s44GrantBandBP/10_000 {
				t.Errorf("seed %d: the units were granted %d bp of what they asked, want 5000 within %d bp of it (half the demand)", seed, us, s44GrantBandBP)
			}
			if diff := max(us, cs) - min(us, cs); diff*10_000/max(us, cs) > s44GrantBandBP {
				t.Errorf("seed %d: units were granted %d bp and cohorts %d bp of what they asked, want them within %d bp of each other (FR-RES-021)", seed, us, cs, s44GrantBandBP)
			}
			if ul > s44DeniedTicks || cl > s44DeniedTicks {
				t.Errorf("seed %d: longest run granted nothing = %d for a unit and %d for a cohort, want at most %d ticks", seed, ul, cl, s44DeniedTicks)
			}
			if unitShort != 0 || cohortShort != 0 {
				t.Errorf("seed %d: requests granted less than floor(w*A/U): %d of units and %d of cohorts, want 0 (FR-RES-021)", seed, unitShort, cohortShort)
			}
			for name, r := range map[string]*fieldRun{"units": unitRun, "cohorts": cohortRun} {
				if got, want := r.warns(), wantWarns(r.changed()); got != want || got == 0 || got > 1+s44Ticks/exhaustedWarnPeriod {
					t.Errorf("seed %d: %s: Warn lines = %d, want %d (FR-RES-028), at least 1 and at most %d", seed, name, got, want, 1+s44Ticks/exhaustedWarnPeriod)
				}
			}
		})
	}
}
