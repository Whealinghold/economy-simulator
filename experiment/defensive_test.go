package experiment_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/experiment"
)

// handBuilt is a Result a consumer assembled or decoded by hand, with fewer
// names than rows, a level index out of range and a reading beyond MaxReading.
func handBuilt() experiment.Result {
	run := func(v int64) experiment.Record {
		return experiment.Record{Status: experiment.StatusOK, Final: []experiment.Reading{{Value: v, OK: true}}}
	}
	return experiment.Result{
		Measures: []string{"m"},
		Design:   experiment.OneAtATime,
		Points: []experiment.PointResult{
			{Index: 0, Levels: []int{0}, Values: []int64{10}, Variants: []experiment.VariantResult{{
				Runs:       []experiment.Record{run(math.MinInt64)},
				Reductions: make([]experiment.Reduction, 3), Pass: make([]experiment.PassShare, 2)}}},
			{Index: 1, Levels: []int{1 << 40}, Variants: []experiment.VariantResult{{Runs: []experiment.Record{run(math.MaxInt64)}}}},
			{Index: 2},
		},
	}
}

// TestHandBuiltResult_NeverPanics verifies that Render, MainEffects, Elasticity
// and Compare take a Result whose lists disagree, whose level is out of range and
// whose reading is beyond MaxReading, and give an error or an undefined value.
//
// Guard: a Result is plain data, and Assemble trusts the records it is given
// (spec-experiments.md 5.3.4); without the guards an index or a wrapped
// difference panics or gives a wrong number.
func TestHandBuiltResult_NeverPanics(t *testing.T) {
	r := handBuilt()
	if out := experiment.Render(r); strings.Contains(out, "reduce m") == false {
		t.Errorf("Render(hand-built) = %q, want the reduce line of the one named measure", out)
	}
	if _, err := experiment.MainEffects(r, "m", 0); err != nil {
		t.Errorf("MainEffects(hand-built) error = %v, want nil", err)
	}
	if _, err := experiment.Elasticity(r, "m", 0, 1); !errors.Is(err, experiment.ErrUnknownRun) {
		t.Errorf("Elasticity(hand-built) error = %v, want ErrUnknownRun", err)
	}
	g0, g1 := experiment.Group{Point: 0}, experiment.Group{Point: 1}
	c, err := experiment.Compare(r, g0, g1, "m")
	if err != nil || c.Pairs != 0 || c.MeanA.OK || c.MeanB.OK {
		t.Errorf("Compare(readings beyond MaxReading) = %+v, %v, want no used reading", c, err)
	}
}
