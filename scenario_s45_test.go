package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s45 sizes. A stock of capacity 10 000 that nobody draws, in the field world of 5
// reapers whose recipe draws nothing here, so that only the regeneration and the
// calls between ticks move it. The year is 73 ticks. The blight sets the stock to
// 5% of the capacity and the drought a regeneration scale of 3 000 bp (30%), both
// through the helpers that the public calls of the integration stage will wrap.
const (
	s45Capacity = 10_000
	s45Ticks    = 200
	s45Blighted = s45Capacity / 20
	s45Drought  = 3_000
)

// s45Shape is one row of S45: a shape and the rates that make it recover in 200
// ticks at full scale.
type s45Shape struct {
	name   string
	res    economysim.Resource
	stock  int64 // the stock the blight leaves
	expect func(t *testing.T, seed uint64, run, control []int64)
}

// s45Shapes lists the shapes of S45.
func s45Shapes() []s45Shape {
	constant, refill, logistic, refuge := economysim.DefaultResource(), economysim.DefaultResource(), economysim.DefaultResource(), economysim.DefaultResource()
	constant.Shape, constant.RatePerYear = economysim.ShapeConstant, 730 // 10 a tick
	refill.Shape, refill.RateBP = economysim.ShapeRefill, 20_000
	logistic.Shape, logistic.RateBP = economysim.ShapeLogistic, 40_000
	refuge.Shape, refuge.RateBP, refuge.Refuge = economysim.ShapeLogistic, 40_000, 10
	return []s45Shape{
		// 730 a year is 10 a tick; the drought scales it to 3 a tick exactly. The
		// blight comes between the first two steps, so 199 ticks regenerate: the stock
		// is 500 + 3 * 199 = 1 097 after 200 ticks, and 500 + 10 * 199 = 2 490 without
		// the drought.
		{"constant", constant, s45Blighted, func(t *testing.T, seed uint64, run, control []int64) {
			if got, want := run[s45Ticks-1], int64(s45Blighted+3*(s45Ticks-1)); got != want {
				t.Errorf("seed %d: constant stock after the drought = %d, want %d (the rate as the scale allows)", seed, got, want)
			}
			if got, want := control[s45Ticks-1], int64(s45Blighted+10*(s45Ticks-1)); got != want {
				t.Errorf("seed %d: constant stock with no drought = %d, want %d", seed, got, want)
			}
		}},
		// A refill of 200% a year closes 2/73 = 2.74% of the gap a tick, and 0.82%
		// under the drought (a scale of 3 000 bp). Over the 199 ticks after the
		// blight the stock is 500 + 9 500 * (1 - 0.9918^199) = 8 150 under the drought
		// (8 161 run) and 500 + 9 500 * (1 - 0.9726^199) = 9 960 without it (9 962).
		{"refill", refill, s45Blighted, func(t *testing.T, seed uint64, run, control []int64) {
			s45Recovers(t, seed, "refill", run, control, 8_000, 9_800)
		}},
		// A logistic stock grows from any positive stock: 5% of the capacity is 500
		// units, w = ceil(S*(C-S)/C) is at least 1, and at 400% a year (5.5% a tick)
		// it grows 1.64% a tick under the drought: C / (1 + 19 * e^(-0.0164 * 199)) =
		// 5 780 after the 199 ticks (5 777 run) against 9 997 without it. (The spec's
		// reading "a logistic stock with no refuge does not recover" is of a stock at
		// 0, tested below.)
		{"logistic from 5%", logistic, s45Blighted, func(t *testing.T, seed uint64, run, control []int64) {
			s45Recovers(t, seed, "logistic", run, control, 5_000, 9_900)
		}},
	}
}

// s45Recovers checks that a recovery rose after the blight without ever falling,
// ended at or above floor under the drought and below the end of the run with no
// drought, and that the run with no drought ended at or above controlFloor.
func s45Recovers(t *testing.T, seed uint64, name string, run, control []int64, floor, controlFloor int64) {
	t.Helper()
	for i := 2; i < len(run); i++ { // run[0] is the full stock before the blight
		if run[i] < run[i-1] {
			t.Errorf("seed %d: %s stock fell from %d to %d at tick %d, want it to rise (nobody draws it)", seed, name, run[i-1], run[i], i)
			break
		}
	}
	if got := run[len(run)-1]; got < floor || got >= control[len(control)-1] {
		t.Errorf("seed %d: %s stock after the drought = %d, want at least %d and below the %d of the run with no drought", seed, name, got, floor, control[len(control)-1])
	}
	if got := control[len(control)-1]; got < controlFloor {
		t.Errorf("seed %d: %s stock with no drought = %d, want at least %d", seed, name, got, controlFloor)
	}
}

// s45Trace runs the field world of res for s45Ticks ticks with the blight and,
// when drought is set, the drought applied between the first two steps, with the
// audit on, and returns the stock after each tick. The identity of the endowment
// (FR-RES-025) is checked every tick from the events and the public read. stock0 is
// the stock the world starts with and blight the stock the call leaves.
func s45Trace(t *testing.T, seed uint64, res economysim.Resource, stock0, blight int64, drought bool) []int64 {
	t.Helper()
	cfg := fieldWorld(res, s45Capacity, stock0, 5, 1_000, 1)
	cfg.Recipes[0].Draws = nil
	r := newFieldRun(t, cfg, seed)
	trace := make([]int64, 0, s45Ticks)
	prev := r.stock()
	for tick := range s45Ticks {
		// The calls between ticks belong to the tick that follows them, so their
		// events and those of the step are summed together.
		mark := len(r.rec.events)
		if tick == 1 {
			economysim.AdjustStock(r.w, 0, 0, blight-r.stock())
			if drought {
				economysim.ShockRegen(r.w, 0, 0, s45Drought)
			}
		}
		r.step(t, seed)
		want := prev
		for _, e := range r.rec.events[mark:] {
			switch e := e.(type) {
			case economysim.EndowmentAdjusted:
				want += e.Delta
			case economysim.EndowmentChanged:
				want += -e.Drawn + e.Regenerated - e.Lost
			}
		}
		if got := r.stock(); got != want {
			t.Fatalf("seed %d tick %d: stock = %d, want %d (stock_start %d - drawn + regenerated - lost + adjusted, FR-RES-025)", seed, tick, got, want, prev)
		}
		prev = r.stock()
		trace = append(trace, prev)
	}
	return trace
}

// TestScenarioS45_BlightAndDrought verifies RES-S5 (spec-resources.md 19.4): a
// blight (the stock set to 5% of the capacity by AdjustStock) and a drought (the
// regeneration scale set to 3 000 bp) on each shape: a refill stock recovers, more
// slowly than with no drought; a logistic stock recovers from 5%, and a logistic
// stock at 0 with no refuge stays 0 while one with a refuge regrows; a constant
// stock regains exactly its scaled rate a tick; the identity of an endowment
// holds in every tick, and the two calls emit EndowmentAdjusted (kinds adjust and
// shock).
//
// Requirement: FR-RES-025, FR-RES-029, FR-RES-030, FR-RES-031
//
// Guard: the identity is checked with the calls between ticks counted; a call that
// moved the stock without an event would break it.
func TestScenarioS45_BlightAndDrought(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	for _, seed := range seedList(scenarioSeedCount()) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			for _, sh := range s45Shapes() {
				run := s45Trace(t, seed, sh.res, s45Capacity, sh.stock, true)
				control := s45Trace(t, seed, sh.res, s45Capacity, sh.stock, false)
				sh.expect(t, seed, run, control)
			}
			// A stock at 0 (the blight took everything): FR-RES-031.
			logistic, refuge := economysim.DefaultResource(), economysim.DefaultResource()
			logistic.Shape, logistic.RateBP = economysim.ShapeLogistic, 40_000
			refuge.Shape, refuge.RateBP, refuge.Refuge = economysim.ShapeLogistic, 40_000, 10
			dead := s45Trace(t, seed, logistic, s45Capacity, 0, true)
			for i, v := range dead[1:] {
				if v != 0 {
					t.Fatalf("seed %d: logistic stock with no refuge at tick %d = %d, want it to stay 0 (FR-RES-031)", seed, i, v)
				}
			}
			if back := s45Trace(t, seed, refuge, s45Capacity, 0, true); back[s45Ticks-1] <= 0 {
				t.Errorf("seed %d: logistic stock with a refuge of 10 after the blight = %d, want it to regrow from 0", seed, back[s45Ticks-1])
			}
			s45Emits(t, seed)
		})
	}
}

// s45Emits checks that the two calls of the blight and the drought emit one
// EndowmentAdjusted each, of the kinds adjust and shock, stamped with the tick the
// next step runs.
func s45Emits(t *testing.T, seed uint64) {
	t.Helper()
	res := economysim.DefaultResource()
	res.Shape, res.RateBP = economysim.ShapeRefill, 20_000
	cfg := fieldWorld(res, s45Capacity, s45Capacity, 5, 1_000, 1)
	cfg.Recipes[0].Draws = nil
	r := newFieldRun(t, cfg, seed)
	r.step(t, seed)
	mark := len(r.rec.events)
	economysim.AdjustStock(r.w, 0, 0, s45Blighted-r.stock())
	economysim.ShockRegen(r.w, 0, 0, s45Drought)
	got := eventsAs[economysim.EndowmentAdjusted](r.rec.events[mark:])
	if len(got) != 2 || got[0].Kind != economysim.AdjustKindAdjust || got[1].Kind != economysim.AdjustKindShock ||
		got[0].Tick != r.w.Tick() || got[1].RegenBP != s45Drought || got[0].StockAfter != s45Blighted {
		t.Errorf("seed %d: events of the blight and the drought = %+v, want an adjust to %d then a shock of %d bp, stamped tick %d", seed, got, s45Blighted, s45Drought, r.w.Tick())
	}
}
