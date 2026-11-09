package experiment

import (
	"slices"
)

// steps returns the calls of one run in the order they are applied at an equal
// tick: the script, then the steps of the levels in factor order, then the
// variant's (spec-experiments.md 3.5.2); the merge by tick is stable. values is
// the Values of the point: for an explicit factor the level that applies (a range
// factor's entry is a knob value and is ignored); variant is -1 for none.
func (e Spec) steps(values []int64, variant int) []Step {
	out := slices.Clone(e.Script)
	for f, fac := range e.Factors {
		if fac.Path == "" {
			out = append(out, fac.Levels[values[f]].Steps...)
		}
	}
	if variant >= 0 {
		out = append(out, e.Branch.Variants[variant].Steps...)
	}
	slices.SortStableFunc(out, func(a, b Step) int { return cmpInt64(a.Tick, b.Tick) })
	return out
}

func cmpInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// boundaries returns the ticks a run stops at, ascending and distinct: 0, the
// end of the transient, every window end, every step tick, the branch point and
// Ticks (spec-experiments.md 5.1.3 step 3).
func (e Spec) boundaries(steps []Step) []int64 {
	out := []int64{0, e.Transient, e.Ticks}
	if e.Every > 0 {
		// Ticks - t > Every is t + Every < Ticks without the overflow of a huge Every.
		for t := e.Transient; e.Ticks-t > e.Every; {
			t += e.Every
			out = append(out, t)
		}
	}
	for _, s := range steps {
		out = append(out, s.Tick)
	}
	if e.Branch != nil {
		out = append(out, e.Branch.At)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// closesWindow reports whether a sample window ends at tick b: every Every ticks
// from the end of the transient, and at Ticks, which closes the last one.
func (e Spec) closesWindow(b int64) bool {
	if b <= e.Transient || e.Every == 0 {
		return false
	}
	return b == e.Ticks || (b-e.Transient)%e.Every == 0
}

// opensWindow reports whether a sample window starts at tick b: at the end of
// the transient and at every window end before Ticks.
func (e Spec) opensWindow(b int64) bool {
	return b < e.Ticks && (b == e.Transient || e.closesWindow(b))
}
