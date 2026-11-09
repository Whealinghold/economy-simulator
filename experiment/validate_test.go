package experiment_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/experiment"
)

// TestValidate_EachRule verifies that each row of the table of
// spec-experiments.md 3.6 is refused with its own sentinel, and that the valid
// base experiment passes: every case breaks exactly one rule of valid().
//
// Requirement: FR-EXP-004
func TestValidate_EachRule(t *testing.T) {
	t.Parallel()
	rng := func(path string, min, max int64, steps int) experiment.Factor {
		return experiment.Factor{Path: path, Min: min, Max: max, Steps: steps}
	}
	tests := []struct {
		name string
		edit func(*experiment.Spec)
		want error // nil: valid
	}{
		{"valid", func(*experiment.Spec) {}, nil},
		{"no replicas", func(e *experiment.Spec) { e.Replicas = 0 }, experiment.ErrBadSchedule},
		{"too many replicas", func(e *experiment.Spec) { e.Replicas = experiment.MaxReplicas + 1 }, experiment.ErrBadSchedule},
		{"no ticks", func(e *experiment.Spec) { e.Ticks = 0 }, experiment.ErrBadSchedule},
		{"transient at ticks", func(e *experiment.Spec) { e.Transient = 10 }, experiment.ErrBadSchedule},
		{"negative transient", func(e *experiment.Spec) { e.Transient = -1 }, experiment.ErrBadSchedule},
		{"negative every", func(e *experiment.Spec) { e.Every = -1 }, experiment.ErrBadSchedule},
		{"too many windows", func(e *experiment.Spec) { e.Ticks, e.Every = experiment.MaxWindows+1, 1 }, experiment.ErrBadSchedule},
		{"script tick at ticks", func(e *experiment.Spec) { e.Script = []experiment.Step{step(10)} }, experiment.ErrBadSchedule},
		{"script tick negative", func(e *experiment.Spec) { e.Script = []experiment.Step{step(-1)} }, experiment.ErrBadSchedule},
		{"script step without a function", func(e *experiment.Spec) { e.Script = []experiment.Step{{Tick: 1}} }, experiment.ErrBadSchedule},
		{"level step out of range", func(e *experiment.Spec) {
			f := twoLevels("a.b")
			f.Levels[1].Steps = []experiment.Step{step(10)}
			e.Factors = []experiment.Factor{f}
		}, experiment.ErrBadSchedule},
		{"one level", func(e *experiment.Spec) {
			e.Factors = []experiment.Factor{{Levels: []experiment.Level{level("a.b")}}}
		}, experiment.ErrBadFactor},
		{"levels and path", func(e *experiment.Spec) {
			f := twoLevels("a.b")
			f.Path = "a.c"
			e.Factors = []experiment.Factor{f}
		}, experiment.ErrBadFactor},
		{"neither levels nor path", func(e *experiment.Spec) { e.Factors = []experiment.Factor{{}} }, experiment.ErrBadFactor},
		{"range min above max", func(e *experiment.Spec) { e.Factors = []experiment.Factor{rng("a.b", 5, 4, 3)} }, experiment.ErrBadFactor},
		{"grid range without steps", func(e *experiment.Spec) { e.Factors = []experiment.Factor{rng("a.b", 1, 4, 1)} }, experiment.ErrBadFactor},
		{"range path syntax", func(e *experiment.Spec) { e.Factors = []experiment.Factor{rng("A.b", 1, 4, 3)} }, experiment.ErrBadFactor},
		{"change path syntax", func(e *experiment.Spec) {
			e.Factors = []experiment.Factor{{Levels: []experiment.Level{level("a.b"), level("nodot")}}}
		}, experiment.ErrBadFactor},
		{"too many factors", func(e *experiment.Spec) {
			for i := 0; i <= experiment.MaxFactors; i++ {
				e.Factors = append(e.Factors, twoLevels(fmt.Sprintf("a.k%d", i)))
			}
			e.Design = experiment.OneAtATime
		}, experiment.ErrBadFactor},
		{"two factors one path", func(e *experiment.Spec) {
			e.Factors = []experiment.Factor{twoLevels("a.b"), twoLevels("a.b")}
		}, experiment.ErrFactorOverlap},
		{"level sets a path twice", func(e *experiment.Spec) {
			e.Factors = []experiment.Factor{{Levels: []experiment.Level{level("a.b", "a.b"), level("a.c")}}}
		}, experiment.ErrFactorOverlap},
		{"range and level share a path", func(e *experiment.Spec) {
			e.Factors = []experiment.Factor{rng("a.b", 1, 2, 2), twoLevels("a.b")}
		}, experiment.ErrFactorOverlap},
		{"unknown design", func(e *experiment.Spec) { e.Design = 9 }, experiment.ErrBadDesign},
		{"unknown seeding", func(e *experiment.Spec) { e.Seeding = 9 }, experiment.ErrBadDesign},
		{"spread of no points", func(e *experiment.Spec) { e.Design = experiment.Spread }, experiment.ErrBadDesign},
		{"duplicate measure", func(e *experiment.Spec) { e.Measures = append(e.Measures, e.Measures[0]) }, economysim.ErrDuplicateID},
		{"duplicate criterion", func(e *experiment.Spec) {
			c := experiment.Criterion{Name: "c", Measure: "m"}
			e.Criteria = []experiment.Criterion{c, c}
		}, economysim.ErrDuplicateID},
		{"criterion names no measure", func(e *experiment.Spec) {
			e.Criteria = []experiment.Criterion{{Name: "c", Measure: "x"}}
		}, experiment.ErrUnknownMeasure},
		{"measure without a function", func(e *experiment.Spec) { e.Measures[0].Fn = nil }, experiment.ErrUnknownMeasure},
		{"branch at 0", func(e *experiment.Spec) {
			e.Branch = &experiment.Branch{At: 0, Variants: []experiment.Variant{{}}}
		}, experiment.ErrBadBranch},
		{"branch at ticks", func(e *experiment.Spec) {
			e.Branch = &experiment.Branch{At: 10, Variants: []experiment.Variant{{}}}
		}, experiment.ErrBadBranch},
		{"branch without variants", func(e *experiment.Spec) { e.Branch = &experiment.Branch{At: 5} }, experiment.ErrBadBranch},
		{"branch with too many variants", func(e *experiment.Spec) {
			e.Branch = &experiment.Branch{At: 5, Variants: make([]experiment.Variant, experiment.MaxVariants+1)}
		}, experiment.ErrBadBranch},
		{"branch with too many continuations", func(e *experiment.Spec) {
			e.Branch = &experiment.Branch{At: 5, Variants: []experiment.Variant{{}}, Continuations: experiment.MaxContinuations + 1}
		}, experiment.ErrBadBranch},
		{"branch with negative continuations", func(e *experiment.Spec) {
			e.Branch = &experiment.Branch{At: 5, Variants: []experiment.Variant{{}}, Continuations: -1}
		}, experiment.ErrBadBranch},
		{"variant step below the branch point", func(e *experiment.Spec) {
			e.Branch = &experiment.Branch{At: 5, Variants: []experiment.Variant{{Steps: []experiment.Step{step(4)}}}}
		}, experiment.ErrBadBranch},
		{"variant step at the branch point", func(e *experiment.Spec) {
			e.Branch = &experiment.Branch{At: 5, Variants: []experiment.Variant{{Steps: []experiment.Step{step(5)}}}}
		}, nil},
		{"variant step beyond the run", func(e *experiment.Spec) {
			e.Branch = &experiment.Branch{At: 5, Variants: []experiment.Variant{{Steps: []experiment.Step{step(10)}}}}
		}, experiment.ErrBadSchedule},
		{"too many measures", func(e *experiment.Spec) {
			e.Measures = nil
			for i := 0; i <= experiment.MaxMeasures; i++ {
				e.Measures = append(e.Measures, experiment.Measure{Name: fmt.Sprint(i), Fn: valid().Measures[0].Fn})
			}
		}, experiment.ErrOversize},
		{"too many points", func(e *experiment.Spec) {
			e.Design, e.DesignN = experiment.Spread, experiment.MaxPoints+1
			e.Factors = []experiment.Factor{twoLevels("a.b")}
		}, experiment.ErrOversize},
		{"too many runs", func(e *experiment.Spec) {
			e.Design, e.DesignN, e.Replicas = experiment.Spread, experiment.MaxPoints, experiment.MaxReplicas
			e.Factors = []experiment.Factor{twoLevels("a.b")}
		}, experiment.ErrOversize},
		{"too many cells", func(e *experiment.Spec) {
			e.Replicas, e.Ticks, e.Every, e.Keep.Series = 1000, 1000, 1, true
			e.Design, e.DesignN = experiment.Spread, 100
			e.Factors = []experiment.Factor{twoLevels("a.b")}
		}, experiment.ErrOversize},
		{"grid product overflows", func(e *experiment.Spec) {
			for i := 0; i < experiment.MaxFactors; i++ {
				e.Factors = append(e.Factors, rng(fmt.Sprintf("a.k%d", i), 0, 9, experiment.MaxLevels))
			}
		}, experiment.ErrOversize},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := valid()
			tt.edit(&e)
			err := e.Validate()
			if tt.want == nil {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("Validate() = %v, want an error that is %v", err, tt.want)
			}
			var ve *economysim.ValidationError
			if !errors.As(err, &ve) || len(ve.Path) == 0 {
				t.Errorf("Validate() = %v, want a *ValidationError with a Go path", err)
			}
		})
	}
}

// TestValidate_PathInTheMessage verifies that an error names the Go field,
// segment by segment, as spec-experiments.md 3.6 shows.
//
// Requirement: FR-EXP-004
func TestValidate_PathInTheMessage(t *testing.T) {
	t.Parallel()
	e := valid()
	e.Factors = []experiment.Factor{twoLevels("a.b"), {Levels: []experiment.Level{level("a.b"), level("a.c", "nodot")}}}
	err := e.Validate()
	var ve *economysim.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("Validate() = %v, want a ValidationError", err)
	}
	if got := ve.Error(); !strings.Contains(got, "Factors[1]") {
		t.Errorf("Validate() error %q, want the path Factors[1]...", got)
	}
	want := []string{"Factors[1]", "Levels[1]", "Changes[1]", "Path"}
	if !errors.Is(err, experiment.ErrBadFactor) || fmt.Sprint(ve.Path) == "" {
		t.Fatalf("Validate() = %v, want ErrBadFactor", err)
	}
	for _, e := range err.(interface{ Unwrap() []error }).Unwrap() {
		var v *economysim.ValidationError
		if errors.As(e, &v) && errors.Is(e, experiment.ErrBadFactor) {
			if fmt.Sprint(v.Path) != fmt.Sprint(want) {
				t.Errorf("path of the bad change = %v, want %v", v.Path, want)
			}
			return
		}
	}
	t.Error("no ErrBadFactor in the joined errors")
}

// TestValidate_AtMost100Errors verifies that 150 faults give 100 errors in path
// order and one ErrTooManyErrors that says 50 were left out.
//
// Requirement: FR-EXP-004
func TestValidate_AtMost100Errors(t *testing.T) {
	t.Parallel()
	e := valid()
	for i := 0; i < 150; i++ {
		e.Script = append(e.Script, step(100)) // outside [0, Ticks)
	}
	err := e.Validate()
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("Validate() = %T, want a joined error", err)
	}
	errs := joined.Unwrap()
	if len(errs) != experiment.MaxErrors+1 {
		t.Fatalf("Validate() gave %d errors, want %d and one ErrTooManyErrors", len(errs), experiment.MaxErrors)
	}
	last := errs[len(errs)-1]
	if !errors.Is(last, economysim.ErrTooManyErrors) || !strings.Contains(last.Error(), "50 more") {
		t.Errorf("last error = %v, want ErrTooManyErrors saying 50 more", last)
	}
	var first, second *economysim.ValidationError
	if !errors.As(errs[0], &first) || !errors.As(errs[1], &second) {
		t.Fatalf("first errors are not ValidationErrors: %v, %v", errs[0], errs[1])
	}
	// Path order compares the segments as text, as the root's validator does:
	// "Script[0]" < "Script[100]" < "Script[101]", because '0' sorts before ']'.
	if first.Path[0] != "Script[0]" || second.Path[0] != "Script[100]" {
		t.Errorf("paths = %v, %v, want Script[0] then Script[100]", first.Path, second.Path)
	}
}

// TestValidate_ZeroValueInvalid verifies that the zero Spec is refused, as its
// doc comment says.
//
// Requirement: FR-EXP-004
func TestValidate_ZeroValueInvalid(t *testing.T) {
	t.Parallel()
	if err := (experiment.Spec{}).Validate(); !errors.Is(err, experiment.ErrBadSchedule) {
		t.Errorf("Spec{}.Validate() = %v, want ErrBadSchedule", err)
	}
	if _, err := experiment.Plan(experiment.Spec{}); err == nil {
		t.Error("Plan(Spec{}) = nil error, want the Validate error")
	}
}

// TestValidate_DocumentCount verifies the document rule of D604: len(Base) plus,
// for each factor, the most documents any level brings (one for each Change and
// each Docs) must not pass MaxDocuments (16), and the boundary itself is valid.
//
// The count: a base of 4 documents, one factor whose widest level has 3 changes
// and 2 docs (5), one range factor (1) and one more factor of 6 changes: 4 + 5
// + 1 + 6 = 16 is valid, one more document is not.
//
// Requirement: FR-EXP-004
func TestValidate_DocumentCount(t *testing.T) {
	t.Parallel()
	build := func(extra int) experiment.Spec {
		e := valid()
		e.Base = make([][]byte, 4)
		wide := level("a.b", "a.c", "a.d")
		wide.Docs = [][]byte{[]byte("x"), []byte("y")}
		six := level("b.a", "b.b", "b.c", "b.d", "b.e", "b.f")
		for range extra {
			six.Docs = append(six.Docs, []byte("z"))
		}
		e.Factors = []experiment.Factor{
			{Levels: []experiment.Level{level("a.b"), wide}},
			{Path: "r.k", Min: 0, Max: 3, Steps: 2},
			{Levels: []experiment.Level{level("b.a"), six}},
		}
		return e
	}
	if err := build(0).Validate(); err != nil {
		t.Errorf("16 documents: Validate() = %v, want nil", err)
	}
	if err := build(1).Validate(); !errors.Is(err, experiment.ErrOversize) {
		t.Errorf("17 documents: Validate() = %v, want ErrOversize", err)
	}
}

// TestValidate_RunsNothing verifies that Validate calls no function value and
// parses no document: a Prepare, an Options, a Step.Do and a Measure.Fn that
// fail the test if called, and a base that is not a valid document.
//
// Requirement: FR-EXP-004
func TestValidate_RunsNothing(t *testing.T) {
	t.Parallel()
	boom := func() { t.Error("Validate called a function value of the experiment") }
	e := valid()
	e.Base = [][]byte{[]byte("\x00 not a document")}
	e.Prepare = func(context.Context, economysim.Config) (economysim.Config, error) {
		boom()
		return economysim.Config{}, nil
	}
	e.Options = func(experiment.Key) []economysim.Option { boom(); return nil }
	e.Script = []experiment.Step{{Tick: 1, Do: func(*economysim.World) error { boom(); return nil }}}
	e.Measures[0].Fn = func(experiment.Sample) (int64, bool, error) { boom(); return 0, false, nil }
	e.Measures[0].Observer = func(experiment.Key) economysim.Observer { boom(); return nil }
	if err := e.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}
