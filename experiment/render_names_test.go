package experiment_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/experiment"
)

// TestRender_NamesStayOnTheirLine verifies that a line break in the experiment, a
// measure or a criterion name cannot start a line of the rendering.
//
// Guard: a name "x\npoint 9" forged a point line in the canonical text that a golden
// or a diff of two sweeps would read as a point.
//
// Requirement: FR-EXP-060
func TestRender_NamesStayOnTheirLine(t *testing.T) {
	t.Parallel()
	e := toySpec(1, experiment.Grid)
	e.Name = "a\npoint 9"
	e.Measures[0].Name = "m\nreduce z"
	e.Criteria = []experiment.Criterion{{Name: "c\nrun 1", Measure: "m\nreduce z", Op: experiment.AtMost, A: 1}}
	res := column(t, e, [][]int64{{3}})
	for _, l := range strings.Split(experiment.Render(res), "\n") {
		if strings.HasPrefix(l, "point 9") || strings.HasPrefix(l, "reduce z") || strings.HasPrefix(l, "run 1") {
			t.Errorf("Render(names with line breaks) has a forged line %q", l)
		}
	}
}

// TestValidate_HugeEvery verifies that an Every near the int64 limit is one window,
// valid by spec-experiments.md 3.4, and does not overflow the window count.
//
// Guard: (Ticks - Transient + Every - 1) / Every wrapped and refused it as oversize.
//
// Requirement: FR-EXP-004
func TestValidate_HugeEvery(t *testing.T) {
	t.Parallel()
	e := toySpec(1, experiment.Grid)
	e.Transient, e.Every, e.Keep.Series = 1, math.MaxInt64, true
	if err := e.Validate(); err != nil {
		t.Errorf("Validate(Every = MaxInt64, Keep.Series) = %v, want nil", err)
	}
}

// TestValidate_CriterionOp verifies that a criterion with an Op outside the closed
// set, or a Between whose bounds are reversed, is refused rather than failing every
// run silently.
//
// Guard: passes() returns false for both, so the verdicts read "fail" with no error.
//
// Requirement: FR-EXP-004
func TestValidate_CriterionOp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		c    experiment.Criterion
	}{
		{"unknown op", experiment.Criterion{Name: "c", Measure: toyMeasure, Op: experiment.Op(7)}},
		{"reversed between", experiment.Criterion{Name: "c", Measure: toyMeasure, Op: experiment.Between, A: 10, B: 5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := toySpec(1, experiment.Grid)
			e.Criteria = []experiment.Criterion{tt.c}
			if err := e.Validate(); !errors.Is(err, experiment.ErrBadDesign) {
				t.Errorf("Validate(%s) = %v, want ErrBadDesign", tt.name, err)
			}
		})
	}
}
