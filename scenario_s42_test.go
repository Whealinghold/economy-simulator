package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s42Ticks is the length of a run of S42. A year is 73 ticks, so the 400 ticks are
// five and a half years: the collapse takes about 130 ticks and a recovery from
// the refuge about 300.
const s42Ticks = 400

// s42 sizes: a logistic stock of capacity 2 000 growing 300% a year (30 000 bp), a
// year of 73 ticks. Its maximum sustainable draw is r*C/4 = 3*2000/4 = 1 500 a
// year = 20.5 a tick (spec-resources.md 4.2, 16.1). A household draws 1 a tick, so
// 15 households draw 15 a tick (0.73 of the maximum) and 25 households draw 25 a
// tick (1.22 of it).
const (
	s42Capacity = 2_000
	s42Below    = 15
	s42Above    = 25
	// s42Equilibrium is the stable stock of the 15 households: the larger root of
	// 15 = (3/73) * S * (1 - S/2000), S = 1000 + sqrt(1000^2 - 15*2000*73/3) =
	// 1000 + sqrt(270 000) = 1 519.6. The regrowth w is rounded up (FR-RES-009), so
	// the run settles a little above it, at 1 535.
	s42Equilibrium = 1_520
)

// logisticField is the logistic stock of S42 with the given refuge.
func logisticField(refuge int64) economysim.Resource {
	res := economysim.DefaultResource()
	res.Shape, res.RateBP, res.Refuge = economysim.ShapeLogistic, 30_000, refuge
	return res
}

// TestScenarioS42_LogisticAboveAndBelowMaximum verifies RES-S2 (spec-resources.md
// 19.4) on a world of reapers who eat what they draw: a draw below the maximum
// sustainable one holds the stock on the stable branch, above it the stock
// collapses; with no refuge a stock that reached 0 stays 0 and the Warn is bounded
// by its period, with a refuge it regrows from 0 (the entrants the spawner sends
// in draw it again); while the stock is short the recipe reports no_endowment, the households go unfed and the
// role fails by the ladder (cause starved), after the stock began to fall short.
//
// Requirement: FR-RES-009, FR-RES-028, FR-RES-031, FR-RES-022
//
// Guard: a regeneration that rounded down would let a logistic stock die near 0
// and a draw that was not rationed would drive the stock negative.
func TestScenarioS42_LogisticAboveAndBelowMaximum(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	for _, seed := range seedList(scenarioSeedCount()) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			s42Below15(t, seed)
			none := s42Collapse(t, seed, logisticField(0))
			if none.stockAtEnd != 0 || none.regrew {
				t.Errorf("seed %d: with no refuge the stock at the end = %d and regrew = %v, want 0 and false (FR-RES-031)", seed, none.stockAtEnd, none.regrew)
			}
			refuge := s42Collapse(t, seed, logisticField(10))
			if !refuge.regrew {
				t.Errorf("seed %d: with a refuge of 10 the stock never rose from 0 again, want it to regrow (FR-RES-031)", seed)
			}
		})
	}
}

// s42Below15 runs the 15 households below the maximum and checks the stable stock.
func s42Below15(t *testing.T, seed uint64) {
	t.Helper()
	cfg := fieldWorld(logisticField(0), s42Capacity, s42Capacity/2, s42Below, 6, 2)
	r := newFieldRun(t, cfg, seed)
	var floor int64 = s42Capacity
	for range s42Ticks {
		r.step(t, seed)
		floor = min(floor, r.stock())
	}
	if floor <= 0 {
		t.Errorf("seed %d: below the maximum the stock fell to %d, want it to stay positive", seed, floor)
	}
	if got := r.stock(); got < s42Equilibrium*95/100 || got > s42Equilibrium*105/100 {
		t.Errorf("seed %d: below the maximum the stock at the end = %d, want %d within 5%% (the stable root)", seed, got, s42Equilibrium)
	}
	var short int64
	for _, e := range r.changed() {
		short += e.Short
	}
	if n, _ := removedStarved(r.rec.events); short != 0 || n != 0 || r.warns() != 0 {
		t.Errorf("seed %d: below the maximum short = %d, starved removals = %d, Warn lines = %d, want 0, 0 and 0", seed, short, n, r.warns())
	}
}

// s42Result is what a run above the maximum shows: the stock at the end and
// whether it rose again after it was first 0.
type s42Result struct {
	stockAtEnd int64
	regrew     bool
}

// s42Collapse runs the 25 households above the maximum and checks the collapse, the
// refusals, the ladder and the Warn; the caller checks the end state, which is what
// the refuge changes.
func s42Collapse(t *testing.T, seed uint64, res economysim.Resource) s42Result {
	t.Helper()
	cfg := fieldWorld(res, s42Capacity, s42Capacity/2, s42Above, 6, 2)
	r := newFieldRun(t, cfg, seed)
	zero, regrew := int64(-1), false
	for tick := range s42Ticks {
		r.step(t, seed)
		if zero < 0 && r.stock() == 0 {
			zero = int64(tick) + 1
		}
		regrew = regrew || (zero >= 0 && r.stock() > 0)
	}
	if zero < 0 || zero > 250 {
		t.Errorf("seed %d: refuge %d: the stock first reached 0 at tick %d, want a collapse within 250 ticks", seed, res.Refuge, zero)
	}
	if n := noEndowmentRefusals(r.rec.events); n == 0 {
		t.Errorf("seed %d: refuge %d: no household was refused for no_endowment, want some once the stock fell short", seed, res.Refuge)
	}
	ev := r.changed()
	firstShort := int64(-1)
	for _, e := range ev {
		if e.Short > 0 {
			firstShort = e.Tick
			break
		}
	}
	// With a refuge the spawner replaces the removed households, who draw the
	// regrown stock and starve in their turn, so more than the first 25 are removed.
	n, firstStarved := removedStarved(r.rec.events)
	if n < s42Above || firstStarved < firstShort || firstShort < 0 {
		t.Errorf("seed %d: refuge %d: starved removals = %d (first at tick %d), first short at tick %d, want at least %d removed by the ladder after the stock fell short", seed, res.Refuge, n, firstStarved, firstShort, s42Above)
	}
	// One endowment: at most one line a tick, and with no refuge one run of short
	// ticks, so one line at its first tick and one in every ten after it.
	bound := s42Ticks
	if res.Refuge == 0 {
		bound = 1 + s42Ticks/exhaustedWarnPeriod
	}
	if got, want := r.warns(), wantWarns(ev); got != want || got == 0 || got > bound {
		t.Errorf("seed %d: refuge %d: Warn lines = %d, want %d (tick 1 of a run and every tenth, FR-RES-028), at least 1 and at most %d", seed, res.Refuge, got, want, bound)
	}
	return s42Result{stockAtEnd: r.stock(), regrew: regrew}
}
