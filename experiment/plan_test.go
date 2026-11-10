package experiment_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/experiment"
)

func keys(units []experiment.Unit) []experiment.Key {
	var out []experiment.Key
	for _, u := range units {
		out = append(out, u.Keys...)
	}
	return out
}

// TestPlan_PointMajorOrder verifies that the units come point ascending, then
// replica ascending (spec-experiments.md 5.1.1): a grid of 2 by 3 levels with 2
// replicas is 12 units, in that order, and Plan is repeatable.
//
// Requirement: FR-EXP-005
func TestPlan_PointMajorOrder(t *testing.T) {
	t.Parallel()
	e := valid()
	e.Factors = []experiment.Factor{
		twoLevels("a.b"),
		{Levels: []experiment.Level{level("a.c"), level("a.c"), level("a.c")}},
	}
	units, err := experiment.Plan(e)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(units) != 12 {
		t.Fatalf("Plan() gave %d units, want 6 points * 2 replicas = 12", len(units))
	}
	for i, u := range units {
		wantP, wantR := i/2, i%2
		if u.Point != wantP || u.Replica != wantR || !slices.Equal(u.Keys, []experiment.Key{{Point: wantP, Replica: wantR}}) {
			t.Errorf("unit %d = %+v, want point %d replica %d with one key", i, u, wantP, wantR)
		}
	}
	again, _ := experiment.Plan(e)
	if !slices.EqualFunc(units, again, func(a, b experiment.Unit) bool {
		return a.Point == b.Point && a.Replica == b.Replica && slices.Equal(a.Keys, b.Keys)
	}) {
		t.Error("Plan() twice gave different units")
	}
}

// TestPlan_BranchKeysByVariantThenContinuation verifies that with a branch a
// unit holds the keys by variant then continuation: 2 variants and 3
// continuations give six keys, and 0 or 1 continuations give one per variant.
//
// Requirement: FR-EXP-005, FR-EXP-032
func TestPlan_BranchKeysByVariantThenContinuation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		conts int
		want  []experiment.Key
	}{
		{3, []experiment.Key{{Variant: 0, Cont: 0}, {Variant: 0, Cont: 1}, {Variant: 0, Cont: 2}, {Variant: 1, Cont: 0}, {Variant: 1, Cont: 1}, {Variant: 1, Cont: 2}}},
		{1, []experiment.Key{{Variant: 0}, {Variant: 1}}},
		{0, []experiment.Key{{Variant: 0}, {Variant: 1}}},
	}
	for _, tt := range tests {
		e := valid()
		e.Replicas = 1
		e.Branch = &experiment.Branch{At: 5, Variants: []experiment.Variant{{}, {}}, Continuations: tt.conts}
		units, err := experiment.Plan(e)
		if err != nil || len(units) != 1 {
			t.Fatalf("Plan() with %d continuations = %v, %v, want one unit", tt.conts, units, err)
		}
		if !slices.Equal(units[0].Keys, tt.want) {
			t.Errorf("Plan() keys with %d continuations = %v, want %v", tt.conts, units[0].Keys, tt.want)
		}
	}
}

// TestPlan_FactorFreeOneUnit verifies that an experiment with no factor has one
// point whatever its design says, so a replicas-only experiment is Replicas units.
//
// Requirement: FR-EXP-005
func TestPlan_FactorFreeOneUnit(t *testing.T) {
	t.Parallel()
	for _, d := range []experiment.Design{experiment.Grid, experiment.OneAtATime, experiment.Spread} {
		e := valid()
		e.Design, e.DesignN, e.Replicas = d, 7, 3
		units, err := experiment.Plan(e)
		if err != nil {
			t.Fatalf("Plan() design %d error = %v", d, err)
		}
		if got := keys(units); len(got) != 3 || got[2] != (experiment.Key{Replica: 2}) {
			t.Errorf("Plan() design %d keys = %v, want one point and 3 replicas", d, got)
		}
	}
}

// TestPlan_OversizeBeforeAnything verifies that a design whose count overflows
// an int, or whose runs or cells pass a bound, is ErrOversize from Plan before
// any unit is built.
//
// Requirement: FR-EXP-064
func TestPlan_OversizeBeforeAnything(t *testing.T) {
	t.Parallel()
	over := func(mut func(*experiment.Spec)) error {
		e := valid()
		mut(&e)
		units, err := experiment.Plan(e)
		if units != nil {
			t.Errorf("Plan() built %d units for an oversize experiment", len(units))
		}
		return err
	}
	cases := map[string]func(*experiment.Spec){
		"grid product overflows int": func(e *experiment.Spec) {
			for i := 0; i < 16; i++ {
				e.Factors = append(e.Factors, experiment.Factor{Path: "a.k" + string(rune('a'+i)), Max: 1, Steps: 1000})
			}
		},
		"runs": func(e *experiment.Spec) {
			e.Design, e.DesignN, e.Replicas = experiment.Spread, 100000, 100
			e.Factors = []experiment.Factor{twoLevels("a.b")}
		},
		"cells with the series": func(e *experiment.Spec) {
			e.Design, e.DesignN, e.Replicas, e.Ticks, e.Every, e.Keep.Series = experiment.Spread, 10000, 10, 1000, 1, true
			e.Factors = []experiment.Factor{twoLevels("a.b")}
		},
	}
	for name, mut := range cases {
		if err := over(mut); !errors.Is(err, experiment.ErrOversize) {
			t.Errorf("Plan() %s = %v, want ErrOversize", name, err)
		}
	}
}
