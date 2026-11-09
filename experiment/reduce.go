package experiment

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/reduce"
)

// nQuantiles is the length of the closed list of quantiles of a Reduction
// (spec-experiments.md 8.3).
const nQuantiles = 5

// Quantile returns the i-th quantile of a Reduction in basis points, for i in
// [0, 5): 1 000, 2 500, 5 000, 7 500 and 9 000. Any other i returns 0.
func Quantile(i int) int {
	switch i {
	case 0:
		return 1000
	case 1:
		return 2500
	case 2:
		return 5000
	case 3:
		return 7500
	case 4:
		return 9000
	}
	return 0
}

// A Reduction is one measure over the runs of one variant of one point
// (spec-experiments.md 8.3): the counts, always beside the statistics, so that a
// mean over the survivors of a group with failures is never read alone
// (FR-EXP-041). Min, Max, Mean, Std and Q are undefined for Used 0, and Std for
// Used below 2.
type Reduction struct {
	// Planned is Used + Failed + Undefined + Missing: the runs the experiment asked for.
	Planned int
	// Used is the runs with status ok and a defined final reading.
	Used int
	// Failed is the runs with a status other than ok and missing.
	Failed int
	// Undefined is the runs with status ok and an undefined reading.
	Undefined int
	// Missing is the planned runs absent from the assembly.
	Missing int
	// Min and Max are the extremes of the used readings.
	Min, Max Reading
	// Mean is the sum of the used readings over Used, rounded half to even.
	Mean Reading
	// Std is the sample standard deviation: two roundings, the variance and then
	// the root (spec-experiments.md 8.3).
	Std Reading
	// Q holds the nearest-rank quantiles of the used readings, in the order of
	// Quantile: each is a value some run produced.
	Q [5]Reading
}

// A PassShare is one criterion over the runs of one variant of one point.
type PassShare struct {
	// Passes is the runs that passed the criterion.
	Passes int
	// Planned is the runs the experiment asked for, the denominator.
	Planned int
	// ShareBP is rdiv(10 000 * Passes, Planned), half to even; 0 for Planned 0.
	ShareBP int
}

// reduceRuns reduces every measure and every criterion over the runs of a group.
func (e Spec) reduceRuns(runs []Record) ([]Reduction, []PassShare) {
	reds := make([]Reduction, len(e.Measures))
	for m := range e.Measures {
		reds[m] = reduceMeasure(runs, m)
	}
	pass := make([]PassShare, len(e.Criteria))
	for c := range e.Criteria {
		p := PassShare{Planned: len(runs)}
		for _, r := range runs {
			if r.Status == StatusOK && c < len(r.Pass) && r.Pass[c] {
				p.Passes++
			}
		}
		p.ShareBP = int(rdivInt(10_000*int64(p.Passes), int64(p.Planned)))
		pass[c] = p
	}
	return reds, pass
}

// rdivInt is n over d rounded half to even, and 0 for a zero divisor.
func rdivInt(n, d int64) int64 {
	q, ok := reduce.RDiv(reduce.FromInt64(n), reduce.FromInt64(d))
	if !ok {
		return 0
	}
	v, _ := q.Int64() // |q| <= |n|
	return v
}

// reduceMeasure counts the runs by outcome and reduces the used readings of
// measure m (spec-experiments.md 8.3).
func reduceMeasure(runs []Record, m int) Reduction {
	red := Reduction{Planned: len(runs)}
	var vals []int64
	var mo reduce.Moments
	for _, r := range runs {
		switch rd := reading(r, m); {
		case r.Status == StatusMissing:
			red.Missing++
		case r.Status != StatusOK:
			red.Failed++
		case !rd.OK || !mo.Add(rd.Value):
			red.Undefined++
		default:
			vals = append(vals, rd.Value)
		}
	}
	red.Used = len(vals)
	if red.Used == 0 {
		return red
	}
	slices.Sort(vals)
	red.Min, red.Max = Reading{vals[0], true}, Reading{vals[len(vals)-1], true}
	red.Mean = readingOf(mo.Mean())
	red.Std = readingOf(mo.Std())
	for i := range nQuantiles {
		q := Quantile(i)
		red.Q[i] = Reading{reduce.Quantile(vals, q), true}
	}
	return red
}

// readingOf wraps a value and its ok flag.
func readingOf(v int64, ok bool) Reading {
	if !ok {
		return Reading{}
	}
	return Reading{v, true}
}

// A Group is the runs of one variant of one point: the unit of a reduction
// (spec-experiments.md 8.4).
type Group struct {
	// Point is the index of the point and Variant that of the variant.
	Point, Variant int
}

// runs returns the records of a group, and false for a group outside the result.
func (r Result) runs(g Group) ([]Record, bool) {
	if g.Point < 0 || g.Point >= len(r.Points) {
		return nil, false
	}
	vs := r.Points[g.Point].Variants
	if g.Variant < 0 || g.Variant >= len(vs) {
		return nil, false
	}
	return vs[g.Variant].Runs, true
}

// calc is checked 128-bit arithmetic whose overflows are collected in one flag, so
// a formula reads as a formula; a result after a failure is meaningless.
type calc struct{ failed bool }

func (k *calc) mul(x, y reduce.Int128) reduce.Int128 {
	z, ok := x.Mul(y)
	k.failed = k.failed || !ok
	return z
}

func (k *calc) sub(x, y reduce.Int128) reduce.Int128 {
	z, ok := x.Sub(y)
	k.failed = k.failed || !ok
	return z
}

// quotient is n over d rounded half to even as a reading, undefined for a zero
// divisor or a result beyond int64.
func quotient(n, d reduce.Int128) Reading {
	q, ok := reduce.RDiv(n, d)
	if !ok {
		return Reading{}
	}
	return readingOf(q.Int64())
}

// clampedQuotient is quotient cut to +-(2^31 - 1), and whether it was cut
// (spec-indicators.md 5.3.7).
func clampedQuotient(n, d reduce.Int128) (Reading, bool) {
	q, ok := reduce.RDiv(n, d)
	if !ok {
		return Reading{}, false
	}
	v, fits := q.Int64()
	switch {
	case !fits && q.IsNeg(), fits && v < -clampBP:
		return Reading{-clampBP, true}, true
	case !fits, v > clampBP:
		return Reading{clampBP, true}, true
	}
	return Reading{v, true}, false
}
