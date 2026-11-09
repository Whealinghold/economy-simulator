package experiment

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strconv"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/design"
)

// report collects the errors of one Validate call.
type report struct{ errs []*economysim.ValidationError }

// add records one rule broken at the Go path of a field. The path is copied, so
// no two errors alias one slice.
func (r *report) add(rule error, value, bound string, path ...string) {
	r.errs = append(r.errs, &economysim.ValidationError{
		Path: slices.Clone(path), Rule: rule, Value: value, Bound: bound,
	})
}

// err returns nil for no error, and otherwise the errors sorted by path then
// rule, cut at MaxErrors, followed by one ErrTooManyErrors that says how many
// were left out (spec-config.md 7.1.3). The root's helpers are unexported, so
// this is a copy of theirs.
func (r *report) err() error {
	if len(r.errs) == 0 {
		return nil
	}
	slices.SortStableFunc(r.errs, func(a, b *economysim.ValidationError) int {
		return cmp.Or(
			slices.Compare(a.Path, b.Path),
			cmp.Compare(a.Rule.Error(), b.Rule.Error()),
			cmp.Compare(a.Value, b.Value),
			cmp.Compare(a.Bound, b.Bound),
		)
	})
	shown := r.errs
	if len(shown) > MaxErrors {
		shown = shown[:MaxErrors]
	}
	out := make([]error, 0, len(shown)+1)
	for _, e := range shown {
		out = append(out, e)
	}
	if left := len(r.errs) - len(shown); left > 0 {
		out = append(out, fmt.Errorf("%w: %d more not shown", economysim.ErrTooManyErrors, left))
	}
	return errors.Join(out...)
}

func itoa(n int) string  { return strconv.Itoa(n) }
func i64(n int64) string { return strconv.FormatInt(n, 10) }

// idx names element i of the list field name, as in Factors[2].
func idx(name string, i int) string { return name + "[" + itoa(i) + "]" }

// Validate checks what is a property of the value alone (spec-experiments.md
// 3.6): it runs no world, parses no document and calls no function value. It
// returns every error it finds, each a *economysim.ValidationError whose Path is
// the Go path of the field, sorted by path then rule, at most MaxErrors and then
// one ErrTooManyErrors; the rules are the sentinels of errors.go.
func (e Spec) Validate() error {
	var r report
	e.validateSchedule(&r)
	e.validateFactors(&r)
	e.validateDesign(&r)
	e.validateNames(&r)
	e.validateBranch(&r)
	e.validateSize(&r)
	return r.err()
}

func (e Spec) validateSchedule(r *report) {
	if e.Replicas < 1 || e.Replicas > MaxReplicas {
		r.add(ErrBadSchedule, itoa(e.Replicas), "1.."+itoa(MaxReplicas), "Replicas")
	}
	e.validateLength(r)
	e.validateSteps(r, e.Script, []string{"Script"}, 0)
	for f, fac := range e.Factors {
		for l, lv := range fac.Levels {
			e.validateSteps(r, lv.Steps, []string{idx("Factors", f), idx("Levels", l), "Steps"}, 0)
		}
	}
}

// validateLength checks Ticks, Transient and Every.
func (e Spec) validateLength(r *report) {
	if e.Ticks < 1 {
		r.add(ErrBadSchedule, i64(e.Ticks), "at least 1", "Ticks")
	}
	if e.Transient < 0 || e.Transient >= e.Ticks {
		r.add(ErrBadSchedule, i64(e.Transient), "0..Ticks-1", "Transient")
	}
	if e.Every < 0 {
		r.add(ErrBadSchedule, i64(e.Every), "at least 0", "Every")
	} else if e.Every > 0 && e.Ticks/e.Every > MaxWindows {
		r.add(ErrBadSchedule, i64(e.Ticks/e.Every), "at most "+itoa(MaxWindows)+" windows", "Every")
	}
}

// validateSteps checks the steps at the path: a tick in [0, Ticks) and a
// function; a branch variant's steps are also held to the branch point (floor
// is that tick, 0 for the others), which is ErrBadBranch.
func (e Spec) validateSteps(r *report, steps []Step, path []string, floor int64) {
	for i, s := range steps {
		at := append(slices.Clone(path[:len(path)-1]), idx(path[len(path)-1], i))
		switch {
		case s.Tick < 0 || s.Tick >= e.Ticks:
			r.add(ErrBadSchedule, i64(s.Tick), "0..Ticks-1", append(at, "Tick")...)
		case s.Tick < floor:
			r.add(ErrBadBranch, i64(s.Tick), "at least Branch.At "+i64(floor), append(at, "Tick")...)
		}
		if s.Do == nil {
			r.add(ErrBadSchedule, "nil", "a function", append(at, "Do")...)
		}
	}
}

func (e Spec) validateFactors(r *report) {
	if len(e.Factors) > MaxFactors {
		r.add(ErrBadFactor, itoa(len(e.Factors)), "at most "+itoa(MaxFactors), "Factors")
	}
	owner := map[string]int{} // membership only: the first factor to set a path
	for f, fac := range e.Factors {
		at := []string{idx("Factors", f)}
		fac.validate(r, at, e.Design != Spread)
		for _, p := range fac.paths() {
			if first, ok := owner[p]; ok && first != f {
				r.add(ErrFactorOverlap, p, "set by Factors["+itoa(first)+"]", append(at, "Path")...)
			} else if !ok {
				owner[p] = f
			}
		}
	}
}

// validate checks one factor: levels or a range, not both, and the syntax of
// every path. needSteps is true where a range factor is cut into Steps levels.
func (f Factor) validate(r *report, at []string, needSteps bool) {
	switch {
	case len(f.Levels) > 0 && f.Path != "":
		r.add(ErrBadFactor, f.Path, "levels or a path, not both", append(at, "Path")...)
	case len(f.Levels) == 0 && f.Path == "":
		r.add(ErrBadFactor, "none", "levels or a path", append(at, "Levels")...)
	case f.Path != "":
		f.validateRange(r, at, needSteps)
	default:
		f.validateLevels(r, at)
	}
}

func (f Factor) validateRange(r *report, at []string, needSteps bool) {
	if !design.ValidPath(f.Path) {
		r.add(ErrBadFactor, f.Path, "a path of 2 to 8 segments", append(at, "Path")...)
	}
	if f.Min > f.Max {
		r.add(ErrBadFactor, i64(f.Min), "at most Max "+i64(f.Max), append(at, "Min")...)
	}
	if needSteps && (f.Steps < 2 || f.Steps > MaxLevels) {
		r.add(ErrBadFactor, itoa(f.Steps), "2.."+itoa(MaxLevels), append(at, "Steps")...)
	}
}

func (f Factor) validateLevels(r *report, at []string) {
	if len(f.Levels) < 2 || len(f.Levels) > MaxLevels {
		r.add(ErrBadFactor, itoa(len(f.Levels)), "2.."+itoa(MaxLevels), append(at, "Levels")...)
	}
	for l, lv := range f.Levels {
		seen := map[string]bool{}
		for c, ch := range lv.Changes {
			cat := append(slices.Clone(at), idx("Levels", l), idx("Changes", c), "Path")
			if !design.ValidPath(ch.Path) {
				r.add(ErrBadFactor, ch.Path, "a path of 2 to 8 segments", cat...)
			} else if seen[ch.Path] {
				r.add(ErrFactorOverlap, ch.Path, "set once in a level", cat...)
			}
			seen[ch.Path] = true
		}
	}
}

// paths returns the paths the factor sets in Changes (a range factor sets its
// Path), in level then change order; a Docs level sets paths the harness does
// not read (spec-experiments.md 3.2.4).
func (f Factor) paths() []string {
	if f.Path != "" {
		return []string{f.Path}
	}
	var out []string
	for _, lv := range f.Levels {
		for _, ch := range lv.Changes {
			out = append(out, ch.Path)
		}
	}
	return out
}

func (e Spec) validateDesign(r *report) {
	switch e.Design {
	case Grid, OneAtATime:
	case Spread:
		if e.DesignN < 1 {
			r.add(ErrBadDesign, itoa(e.DesignN), "at least 1", "DesignN")
		}
	default:
		r.add(ErrBadDesign, itoa(int(e.Design)), "Grid, OneAtATime or Spread", "Design")
	}
	if e.Seeding != SameSeeds && e.Seeding != PerPointSeeds {
		r.add(ErrBadDesign, itoa(int(e.Seeding)), "SameSeeds or PerPointSeeds", "Seeding")
	}
}

func (e Spec) validateNames(r *report) {
	names := map[string]bool{}
	for i, m := range e.Measures {
		if names[m.Name] {
			r.add(economysim.ErrDuplicateID, m.Name, "", idx("Measures", i), "Name")
		}
		names[m.Name] = true
		if m.Fn == nil {
			r.add(ErrUnknownMeasure, "nil", "a function", idx("Measures", i), "Fn")
		}
	}
	crit := map[string]bool{}
	for i, c := range e.Criteria {
		if crit[c.Name] {
			r.add(economysim.ErrDuplicateID, c.Name, "", idx("Criteria", i), "Name")
		}
		crit[c.Name] = true
		if !names[c.Measure] {
			r.add(ErrUnknownMeasure, c.Measure, "a Measure.Name", idx("Criteria", i), "Measure")
		}
		if c.Op != AtMost && c.Op != AtLeast && c.Op != Between || c.Op == Between && c.A > c.B {
			r.add(ErrBadDesign, itoa(int(c.Op)), "AtMost, AtLeast or Between with A <= B", idx("Criteria", i), "Op")
		}
	}
}

func (e Spec) validateBranch(r *report) {
	b := e.Branch
	if b == nil {
		return
	}
	if b.At < 1 || b.At >= e.Ticks {
		r.add(ErrBadBranch, i64(b.At), "1..Ticks-1", "Branch", "At")
	}
	if len(b.Variants) < 1 || len(b.Variants) > MaxVariants {
		r.add(ErrBadBranch, itoa(len(b.Variants)), "1.."+itoa(MaxVariants), "Branch", "Variants")
	}
	if b.Continuations < 0 || b.Continuations > MaxContinuations {
		r.add(ErrBadBranch, itoa(b.Continuations), "0.."+itoa(MaxContinuations), "Branch", "Continuations")
	}
	for v, vr := range b.Variants {
		e.validateSteps(r, vr.Steps, []string{"Branch", idx("Variants", v), "Steps"}, b.At)
	}
}

func (e Spec) validateSize(r *report) {
	if len(e.Measures) > MaxMeasures {
		r.add(ErrOversize, itoa(len(e.Measures)), "at most "+itoa(MaxMeasures), "Measures")
	}
	if docs, ok := e.documents(); !ok || docs > economysim.MaxDocuments {
		r.add(ErrOversize, itoa(docs), "at most "+itoa(economysim.MaxDocuments)+" documents of a point", "Base")
	}
	if e.countsSane() { // else a factor already broke its own rule
		e.validateCounts(r)
	}
}

// validateCounts checks the points, the runs and the cells in turn, each from a
// checked product, and reports the first bound passed.
func (e Spec) validateCounts(r *report) {
	points, ok := e.pointCount()
	if !ok || points > MaxPoints {
		r.add(ErrOversize, itoa(points), "at most "+itoa(MaxPoints)+" points", "Factors")
		return
	}
	runs, ok := e.runCount(points)
	if !ok || runs > MaxRuns {
		r.add(ErrOversize, itoa(runs), "at most "+itoa(MaxRuns)+" runs", "Replicas")
		return
	}
	if cells, ok := e.cellCount(runs); !ok || cells > MaxCells {
		r.add(ErrOversize, itoa(cells), "at most "+itoa(MaxCells)+" cells", "Measures")
	}
}

// countsSane reports whether every factor has a level count the design can use:
// at least 1 (a range factor's Steps matter only outside a Spread), and, for
// the spread, DesignN of at least 1.
func (e Spec) countsSane() bool {
	if len(e.Factors) == 0 {
		return true
	}
	if e.Design == Spread {
		return e.DesignN >= 1
	}
	for _, n := range e.levelCounts() {
		if n < 1 {
			return false
		}
	}
	return e.Design == Grid || e.Design == OneAtATime
}

// documents returns the documents one point needs: the base, then for each
// factor the most documents any of its levels brings (one for each Change and
// each Docs; a range factor brings one). It returns false on overflow
// (spec-experiments.md 13.2).
func (e Spec) documents() (int, bool) {
	total := len(e.Base)
	for _, f := range e.Factors {
		most := 0
		if f.Path != "" {
			most = 1
		}
		for _, lv := range f.Levels {
			most = max(most, len(lv.Docs)+len(lv.Changes))
		}
		if total > int(^uint(0)>>1)-most {
			return 0, false
		}
		total += most
	}
	return total, true
}

// runCount returns points times variants times replicas times max(1,
// continuations), checked.
func (e Spec) runCount(points int) (int, bool) {
	variants, conts := 1, 1
	if b := e.Branch; b != nil {
		variants, conts = max(1, len(b.Variants)), max(1, b.Continuations)
	}
	return design.CheckedCount(points, variants, max(1, e.Replicas), conts)
}

// cellCount returns runs times measures times (windows + 1) with the series, and
// runs times measures without it, checked.
func (e Spec) cellCount(runs int) (int, bool) {
	perRun := max(1, len(e.Measures))
	if e.Keep.Series && e.Every > 0 {
		windows := (e.Ticks - e.Transient) / e.Every // ceiling, without overflow for a huge Every
		if (e.Ticks-e.Transient)%e.Every != 0 {
			windows++
		}
		if windows < 0 || windows > MaxWindows+1 {
			return 0, false
		}
		return design.CheckedCount(runs, perRun, int(windows)+1)
	}
	return design.CheckedCount(runs, perRun)
}
