package experiment

import (
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/reduce"
)

// A Bin is the used readings of one setting of a factor: a level for a grid or a
// one-at-a-time design, a range of cells for a spread.
type Bin struct {
	// Index is the level, or the bin of the spread.
	Index int
	// Used is the used final readings pooled in the bin.
	Used int
	// Mean is the pooled sum over Used in one division, never a mean of rounded
	// point means; undefined for an empty bin.
	Mean Reading
}

// An Effect is the main effect of one factor on a measure (spec-experiments.md
// 8.5.1).
type Effect struct {
	// Factor is the index of the factor.
	Factor int
	// Bins are the bins in index order, empty ones included.
	Bins []Bin
	// Range is the largest bin mean less the smallest over the non-empty bins, in
	// the measure's unit; undefined with fewer than two.
	Range Reading
	// ShareBP is the factor's part of the sum of the ranges of all factors, in
	// basis points; undefined when the range is, or the sum is 0.
	ShareBP Reading
}

// MainEffects returns the main effect of every factor on a measure, in factor
// order (spec-experiments.md 8.5.1). The points are binned by level for a grid and
// for a one-at-a-time design (where a factor's bins hold the baseline and the
// points of that factor alone), and by floor(cell * bins / points) for a spread,
// where bins is a datum of the call in [2, points]; it is ignored for the other
// designs. A bin pools every used final reading of every run of its points. It
// is not a variance decomposition: factors that cancel in a bin look small.
func MainEffects(r Result, measure string, bins int) ([]Effect, error) {
	m := slices.Index(r.Measures, measure)
	if m < 0 {
		return nil, fmt.Errorf("main effects of %q: %w", measure, ErrUnknownMeasure)
	}
	// A spread of one point, or with no factor, has nothing to bin.
	if r.Design == Spread && len(r.Points) > 1 && len(r.Points[0].Levels) > 0 && (bins < 2 || bins > len(r.Points)) {
		return nil, fmt.Errorf("main effects: %d bins of %d points: %w", bins, len(r.Points), ErrBadDesign)
	}
	if len(r.Points) == 0 {
		return nil, nil
	}
	out := make([]Effect, len(r.Points[0].Levels))
	for f := range out {
		out[f] = r.effectOf(f, m, bins)
	}
	shares(out)
	return out, nil
}

// shares sets each effect's share of the sum of the ranges.
func shares(effects []Effect) {
	var total reduce.Int128
	for _, e := range effects {
		if e.Range.OK {
			total, _ = total.Add(reduce.FromInt64(e.Range.Value)) // a sum of ranges below 2^41 each
		}
	}
	if total.Sign() <= 0 {
		return
	}
	for f, e := range effects {
		if e.Range.OK {
			scaled, _ := reduce.FromInt64(e.Range.Value).Mul(reduce.FromInt64(10_000)) // < 2^55
			effects[f].ShareBP = quotient(scaled, total)
		}
	}
}

// effectOf bins the points by factor f and measure m.
func (r Result) effectOf(f, m, bins int) Effect {
	var mos []reduce.Moments
	for _, p := range r.Points {
		b, ok := r.binOf(p, f, bins)
		if !ok {
			continue
		}
		for len(mos) <= b {
			mos = append(mos, reduce.Moments{})
		}
		p.addUsed(&mos[b], m)
	}
	e := Effect{Factor: f}
	var means []int64
	for i, mo := range mos {
		mean, ok := mo.Mean()
		e.Bins = append(e.Bins, Bin{Index: i, Used: int(mo.Count()), Mean: readingOf(mean, ok)})
		if ok {
			means = append(means, mean)
		}
	}
	if len(means) >= 2 {
		e.Range = Reading{slices.Max(means) - slices.Min(means), true} // two means within 2^40: no wrap
	}
	return e
}

// addUsed adds the used final readings of measure m of every run of the point.
func (p PointResult) addUsed(mo *reduce.Moments, m int) {
	for _, v := range p.Variants {
		for _, run := range v.Runs {
			if rd := reading(run, m); run.Status == StatusOK && rd.OK {
				mo.Add(rd.Value) // cannot overflow for readings within MaxReading
			}
		}
	}
}

// binOf returns the bin of point p for factor f, and false when the design leaves
// the point out of that factor's effect.
func (r Result) binOf(p PointResult, f, bins int) (int, bool) {
	if f >= len(p.Levels) || p.Levels[f] < 0 || p.Levels[f] > MaxPoints { // a hand-built Result
		return 0, false
	}
	switch r.Design {
	case Spread:
		return p.Levels[f] * bins / len(r.Points), true // cell < points <= MaxPoints, bins <= points
	case OneAtATime:
		if !onlyFactor(p, f) {
			return 0, false
		}
	}
	return p.Levels[f], true
}

// A Sensitivity is an elasticity in basis points.
type Sensitivity struct {
	// BP is the elasticity: 10 000 means a 1% rise of the knob moves the measure by 1%.
	BP int64
	// OK is false for a baseline at or below 0, a baseline value of the knob of 0,
	// a level equal to the baseline, no used run, or an overflow.
	OK bool
	// Clamped says BP was cut to +-(2^31 - 1), or the numerator overflowed.
	Clamped bool
}

// Elasticity is the finite-difference elasticity of a measure to a factor at a
// level against the baseline of a one-at-a-time design (spec-experiments.md
// 8.5.3), from the exact sums of the used final readings of the runs of the two
// points:
//
//	rdiv(10 000 * (Y_l * n_0 - Y_0 * n_l) * x_0, Y_0 * n_l * (x_l - x_0))
//
// x is the knob's value: Values of a range factor, the value of the first change
// of the level of an explicit factor (0 when the level has none). It is valid for a
// small step.
// A design other than OneAtATime is ErrBadDesign, an unknown measure
// ErrUnknownMeasure and a factor or level outside the result ErrUnknownRun.
func Elasticity(r Result, measure string, factor, level int) (Sensitivity, error) {
	m := slices.Index(r.Measures, measure)
	if m < 0 {
		return Sensitivity{}, fmt.Errorf("elasticity of %q: %w", measure, ErrUnknownMeasure)
	}
	if r.Design != OneAtATime {
		return Sensitivity{}, fmt.Errorf("elasticity: design %d is not one at a time: %w", int(r.Design), ErrBadDesign)
	}
	base, at := r.oatPoints(factor, level)
	if base < 0 || at < 0 {
		return Sensitivity{}, fmt.Errorf("elasticity: factor %d level %d: %w", factor, level, ErrUnknownRun)
	}
	y0, n0 := r.sum(base, m)
	yl, nl := r.sum(at, m)
	x0, xl, ok := r.knobValues(factor, level, base, at)
	if !ok { // a hand-built Result: the level or the values are not there
		return Sensitivity{}, fmt.Errorf("elasticity: factor %d level %d: %w", factor, level, ErrUnknownRun)
	}
	return elasticity(y0, n0, yl, nl, x0, xl), nil
}

// knobValues returns the knob's value at the baseline and at the level, and false
// when the result does not have them.
func (r Result) knobValues(factor, level, base, at int) (x0, xl int64, ok bool) {
	f := r.firstOf(factor)
	switch {
	case f != nil && level >= 0 && level < len(f): // an explicit factor: Values is the level index
		return f[0], f[level], true
	case f == nil && factor < len(r.Points[base].Values) && factor < len(r.Points[at].Values):
		return r.Points[base].Values[factor], r.Points[at].Values[factor], true
	}
	return 0, 0, false
}

// firstOf returns the first-change values of an explicit factor, or nil for a
// range factor, a factor outside the result or a Result Assemble did not build.
func (r Result) firstOf(factor int) []int64 {
	if factor < 0 || factor >= len(r.first) || len(r.first[factor]) == 0 {
		return nil
	}
	return r.first[factor]
}

// oatPoints returns the indexes of the baseline and of the point where only factor
// is at level, or -1 for either that the result does not have.
func (r Result) oatPoints(factor, level int) (base, at int) {
	base, at = -1, -1
	for i, p := range r.Points {
		if factor < 0 || factor >= len(p.Levels) {
			return -1, -1
		}
		if onlyFactor(p, -1) {
			base = i
		}
		if p.Levels[factor] == level && onlyFactor(p, factor) {
			at = i
		}
	}
	return base, at
}

// onlyFactor reports whether every factor but f is at level 0 (f -1: all of them).
func onlyFactor(p PointResult, f int) bool {
	for g, l := range p.Levels {
		if g != f && l != 0 {
			return false
		}
	}
	return true
}

// sum returns the exact sum and count of the used final readings of point p.
func (r Result) sum(p, m int) (reduce.Int128, int64) {
	var mo reduce.Moments
	r.Points[p].addUsed(&mo, m)
	return mo.Sum(), mo.Count()
}

// elasticity is the formula of Elasticity from the sums and counts.
func elasticity(y0 reduce.Int128, n0 int64, yl reduce.Int128, nl int64, x0, xl int64) Sensitivity {
	if n0 == 0 || nl == 0 || y0.Sign() <= 0 || x0 == 0 || xl == x0 {
		return Sensitivity{}
	}
	i := reduce.FromInt64
	var k calc
	y0nl := k.mul(y0, i(nl))
	diff := k.sub(k.mul(yl, i(n0)), y0nl) // Y_l*n_0 - Y_0*n_l
	num := k.mul(k.mul(diff, i(10_000)), i(x0))
	den := k.mul(y0nl, k.sub(i(xl), i(x0)))
	if k.failed {
		return Sensitivity{Clamped: true}
	}
	q, clamped := clampedQuotient(num, den)
	return Sensitivity{BP: q.Value, OK: q.OK, Clamped: clamped}
}
