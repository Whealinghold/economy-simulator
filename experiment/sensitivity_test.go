package experiment_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/experiment"
)

// points1 wraps each value as the one reading of a point.
func points1(vs ...int64) [][]int64 {
	out := make([][]int64, len(vs))
	for i, v := range vs {
		out[i] = []int64{v}
	}
	return out
}

// TestMainEffects_ThreeByThree verifies the effect range and share of the 3 by 3
// grid of spec-experiments.md 19.7.
//
// The point means y(i, j) are 100, 104, 108 / 110, 114, 118 / 140, 144, 148. The
// bin means of the first factor are 104, 114 and 144 (range 40); of the second
// 350/3 = 116.7, 362/3 = 120.7 and 374/3 = 124.7, so 117, 121 and 125 (range 8).
// The shares are rdiv(10 000 * 40, 48) = 8 333 and rdiv(10 000 * 8, 48) = 1 667.
//
// Requirement: FR-EXP-046
func TestMainEffects_ThreeByThree(t *testing.T) {
	t.Parallel()
	e := toySpec(1, experiment.Grid, levels("world.a", 3), levels("world.b", 3))
	res := column(t, e, points1(100, 104, 108, 110, 114, 118, 140, 144, 148))
	got, err := experiment.MainEffects(res, toyMeasure, 0)
	if err != nil {
		t.Fatalf("MainEffects() error = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("MainEffects() = %d effects, want 2", len(got))
	}
	means := [][]int64{{104, 114, 144}, {117, 121, 125}}
	for f, want := range []struct{ rng, share int64 }{{40, 8333}, {8, 1667}} {
		if got[f].Range != rd(want.rng) || got[f].ShareBP != rd(want.share) {
			t.Errorf("MainEffects factor %d Range, ShareBP = %+v, %+v, want %d, %d", f, got[f].Range, got[f].ShareBP, want.rng, want.share)
		}
		for i, m := range means[f] {
			if b := got[f].Bins[i]; b.Mean != rd(m) || b.Used != 3 || b.Index != i {
				t.Errorf("MainEffects factor %d bin %d = %+v, want mean %d over 3 readings", f, i, b, m)
			}
		}
	}
}

// TestMainEffects_DesignsAndErrors verifies the binning of a one-at-a-time design
// (a factor's bins hold the baseline and its own points), the cells of a spread
// into bins, a factor with fewer than two non-empty bins, and the errors.
//
// Requirement: FR-EXP-046
func TestMainEffects_DesignsAndErrors(t *testing.T) {
	t.Parallel()
	// One at a time over two 3-level factors: 5 points, the baseline (0,0), then
	// (1,0), (2,0), (0,1), (0,2). Factor 0 sees points 0, 1, 2 and factor 1 sees 0, 3, 4.
	oat := column(t, toySpec(1, experiment.OneAtATime, levels("world.a", 3), levels("world.b", 3)), points1(10, 20, 30, 12, 18))
	got, err := experiment.MainEffects(oat, toyMeasure, 0)
	if err != nil {
		t.Fatalf("MainEffects(one at a time) error = %v, want nil", err)
	}
	if got[0].Range != rd(20) || got[1].Range != rd(8) || got[0].ShareBP != rd(7143) || got[1].ShareBP != rd(2857) {
		t.Errorf("MainEffects(one at a time) = ranges %+v, %+v shares %+v, %+v, want 20 and 8 and 7143 and 2857 (20/28, 8/28)", got[0].Range, got[1].Range, got[0].ShareBP, got[1].ShareBP)
	}
	// A spread of 4 points in 2 bins: every cell is used once, so two points a bin.
	spread := toySpec(1, experiment.Spread, levels("world.a", 2))
	spread.DesignN = 4
	sp := column(t, spread, points1(1, 2, 3, 4))
	eff, err := experiment.MainEffects(sp, toyMeasure, 2)
	if err != nil {
		t.Fatalf("MainEffects(spread) error = %v, want nil", err)
	}
	if len(eff[0].Bins) != 2 || eff[0].Bins[0].Used != 2 || eff[0].Bins[1].Used != 2 {
		t.Errorf("MainEffects(spread, 2 bins) bins = %+v, want two bins of two readings", eff[0].Bins)
	}
	// All readings equal: two bins, range 0, total range 0: share undefined.
	flat := column(t, toySpec(1, experiment.Grid, levels("world.a", 2)), points1(5, 5))
	f, err := experiment.MainEffects(flat, toyMeasure, 0)
	if err != nil {
		t.Fatalf("MainEffects(flat) error = %v, want nil", err)
	}
	if f[0].Range != rd(0) || f[0].ShareBP.OK {
		t.Errorf("MainEffects(flat) Range, ShareBP = %+v, %+v, want 0 and undefined", f[0].Range, f[0].ShareBP)
	}
	for _, tt := range []struct {
		name string
		res  experiment.Result
		m    string
		bins int
		want error
	}{
		{"unknown measure", sp, "nope", 2, experiment.ErrUnknownMeasure},
		{"spread with one bin", sp, toyMeasure, 1, experiment.ErrBadDesign},
		{"spread with more bins than points", sp, toyMeasure, 5, experiment.ErrBadDesign},
	} {
		if _, err := experiment.MainEffects(tt.res, tt.m, tt.bins); !errors.Is(err, tt.want) {
			t.Errorf("MainEffects(%s) error = %v, want %v", tt.name, err, tt.want)
		}
	}
}

// oatKnob assembles a one-at-a-time sweep of a range knob from min to max in two
// steps, with the readings of the baseline and of the second level.
func oatKnob(t testing.TB, min, max int64, base, lvl []int64) experiment.Result {
	t.Helper()
	return column(t, toySpec(len(base), experiment.OneAtATime, knob(min, max, 2)), [][]int64{base, lvl})
}

// TestElasticity_Worked verifies the elasticity of spec-experiments.md 19.7 and its
// OK conditions: a knob from 7 000 to 8 000 moving a measure from 12 000 to 11 100
// has rdiv(10 000 * (-900) * 7 000, 12 000 * 1 000) = -5 250; it is undefined for a
// baseline at or below 0, a baseline knob of 0 and a level equal to the baseline;
// it is clamped at 2^31 - 1 and flagged when the numerator overflows.
//
// Requirement: FR-EXP-046
func TestElasticity_Worked(t *testing.T) {
	t.Parallel()
	got, err := experiment.Elasticity(oatKnob(t, 7000, 8000, []int64{12000}, []int64{11100}), toyMeasure, 0, 1)
	if err != nil || got != (experiment.Sensitivity{BP: -5250, OK: true}) {
		t.Errorf("Elasticity(19.7) = %+v, %v, want -5250 ok", got, err)
	}
	// With three replicas the sums pool: Y_l = 33 300, Y_0 = 36 000, n = 3 each give the
	// same -900 * 3 * ... = rdiv(10 000 * (33 300*3 - 36 000*3) * 7 000, 36 000 * 3 * 1 000).
	pooled, err := experiment.Elasticity(oatKnob(t, 7000, 8000, []int64{12000, 12000, 12000}, []int64{11100, 11100, 11100}), toyMeasure, 0, 1)
	if err != nil || pooled.BP != -5250 {
		t.Errorf("Elasticity(three replicas) = %+v, %v, want -5250", pooled, err)
	}
	tests := []struct {
		name  string
		res   experiment.Result
		level int
		want  experiment.Sensitivity
	}{
		{"baseline at 0", oatKnob(t, 7000, 8000, []int64{0}, []int64{5}), 1, experiment.Sensitivity{}},
		{"baseline below 0", oatKnob(t, 7000, 8000, []int64{-4}, []int64{5}), 1, experiment.Sensitivity{}},
		{"knob baseline 0", oatKnob(t, 0, 1000, []int64{10}, []int64{12}), 1, experiment.Sensitivity{}},
		{"level is the baseline", oatKnob(t, 7000, 8000, []int64{10}, []int64{12}), 0, experiment.Sensitivity{}},
		// 10 000 * (1e9 - 1) * 7 000 / (1 * 1 000) is about 7e13, beyond 2^31 - 1.
		{"clamped", oatKnob(t, 7000, 8000, []int64{1}, []int64{1_000_000_000}), 1, experiment.Sensitivity{BP: 1<<31 - 1, OK: true, Clamped: true}},
		{"clamped below", oatKnob(t, 7000, 8000, []int64{1}, []int64{-1_000_000_000}), 1, experiment.Sensitivity{BP: -(1<<31 - 1), OK: true, Clamped: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, err := experiment.Elasticity(tt.res, toyMeasure, 0, tt.level); err != nil || got != tt.want {
				t.Errorf("Elasticity(%s) = %+v, %v, want %+v", tt.name, got, err, tt.want)
			}
		})
	}
}

// TestElasticity_Overflow verifies that a numerator beyond 128 bits is undefined
// and flagged Clamped: 1 000 readings of 2^40 against 1 000 of 1, a knob at 2^62,
// give (2^40 * 1 000 * 1 000) * 10 000 * 2^62, about 2^135.
//
// Requirement: FR-EXP-046
func TestElasticity_Overflow(t *testing.T) {
	t.Parallel()
	hi, lo := make([]int64, 1000), make([]int64, 1000)
	for i := range hi {
		hi[i], lo[i] = 1<<40, 1
	}
	res := oatKnob(t, 1<<62, 1<<62+1000, lo, hi)
	got, err := experiment.Elasticity(res, toyMeasure, 0, 1)
	if err != nil || got != (experiment.Sensitivity{Clamped: true}) {
		t.Errorf("Elasticity(overflow) = %+v, %v, want undefined and Clamped", got, err)
	}
}

// TestElasticity_Errors verifies the errors: another design, an unknown measure and
// a factor or level the result does not have.
//
// Requirement: FR-EXP-046
func TestElasticity_Errors(t *testing.T) {
	t.Parallel()
	grid := column(t, toySpec(1, experiment.Grid, knob(1, 2, 2)), points1(1, 2))
	oat := oatKnob(t, 7000, 8000, []int64{12000}, []int64{11100})
	for _, tt := range []struct {
		name          string
		res           experiment.Result
		m             string
		factor, level int
		want          error
	}{
		{"grid design", grid, toyMeasure, 0, 1, experiment.ErrBadDesign},
		{"unknown measure", oat, "nope", 0, 1, experiment.ErrUnknownMeasure},
		{"factor past the result", oat, toyMeasure, 1, 1, experiment.ErrUnknownRun},
		{"negative factor", oat, toyMeasure, -1, 1, experiment.ErrUnknownRun},
		{"level past the factor", oat, toyMeasure, 0, 2, experiment.ErrUnknownRun},
	} {
		if _, err := experiment.Elasticity(tt.res, tt.m, tt.factor, tt.level); !errors.Is(err, tt.want) {
			t.Errorf("Elasticity(%s) error = %v, want %v", tt.name, err, tt.want)
		}
	}
}

// TestElasticity_ExplicitFactor verifies that the knob value x of an explicit
// factor is the value of its level's first change, not the level index (spec-experiments.md
// 8.5.3): levels with changes 7 000 and 8 000 and readings 12 000 and 11 100 give
// the -5 250 of the range factor of 19.7.
//
// Requirement: FR-EXP-046
func TestElasticity_ExplicitFactor(t *testing.T) {
	t.Parallel()
	f := experiment.Factor{Name: "knob", Levels: []experiment.Level{
		{Changes: []experiment.Change{{Path: "world.knob", Value: 7000}}},
		{Changes: []experiment.Change{{Path: "world.knob", Value: 8000}}},
	}}
	res := column(t, toySpec(1, experiment.OneAtATime, f), [][]int64{{12000}, {11100}})
	got, err := experiment.Elasticity(res, toyMeasure, 0, 1)
	if err != nil || got != (experiment.Sensitivity{BP: -5250, OK: true}) {
		t.Errorf("Elasticity(explicit levels 7 000, 8 000) = %+v, %v, want -5250 ok", got, err)
	}
}

// TestMainEffects_SpreadWithoutFactors verifies that a spread with no factor, which
// has one point and nothing to bin, accepts any bins.
//
// Guard: the bins check once refused it with ErrBadDesign.
func TestMainEffects_SpreadWithoutFactors(t *testing.T) {
	t.Parallel()
	e := toySpec(1, experiment.Spread)
	e.DesignN = 1
	res := column(t, e, [][]int64{{5}})
	if got, err := experiment.MainEffects(res, toyMeasure, 0); err != nil || len(got) != 0 {
		t.Errorf("MainEffects(spread, no factors, 0 bins) = %v, %v, want no effects, nil", got, err)
	}
}
