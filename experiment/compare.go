package experiment

import (
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/reduce"
)

// clampBP is the bound of a growth in basis points, 2^31 - 1 (spec-indicators.md
// 5.3.7).
const clampBP = 1<<31 - 1

// A QuantilePair is the nearest-rank quantile of each of two groups and their
// difference, the comparison of distributions over replicas (spec-experiments.md
// 8.4.2).
type QuantilePair struct {
	// A and B are the quantile of each group, and Diff is B - A. Each is undefined
	// when its group has no used reading.
	A, B, Diff Reading
}

// A Comparison is group B against group A for one measure (spec-experiments.md
// 8.4). A field that can be undefined is a Reading.
type Comparison struct {
	// Pairs is the matched pairs in which both readings are defined; 0 for an
	// unpaired comparison.
	Pairs int
	// MeanA and MeanB are the means of all used readings of each group, matched or
	// not, and DiffOfMeans is their difference from the exact sums, in one division.
	MeanA, MeanB, DiffOfMeans Reading
	// MeanDiff is the mean of d = B - A over the pairs and MedianDiff its median.
	MeanDiff, MedianDiff Reading
	// Wins, Losses and Ties count the pairs with d above, below and equal to 0.
	Wins, Losses, Ties int
	// SignTailPPM is the exact two-sided sign test, ties dropped, in parts per
	// million; undefined for no untied pair and for more than 62.
	SignTailPPM Reading
	// RelativeBP is the growth of B over A in basis points, from the exact sums in
	// one division, clamped to +-(2^31 - 1) with Clamped set; undefined for a base
	// at or below 0.
	RelativeBP Reading
	// Clamped says RelativeBP was cut to the clamp.
	Clamped bool
	// Dist has the five quantiles of each group, in the order of Quantile.
	Dist [5]QuantilePair
	// PairingGain is 10 000 * SdIndep / SdDiff, in basis points: the spread of the
	// difference of two independent runs over that of the paired difference.
	// Undefined for SdDiff 0 and for fewer than two pairs.
	PairingGain Reading
	// SdIndep is isqrt_nearest(vA + vB) of the rounded sample variances of the used
	// readings of each group, and SdDiff the sample standard deviation of d.
	SdIndep, SdDiff Reading
	// Identical says every d is 0 (and there is at least one pair).
	Identical bool
}

// side is what Compare reads of one group: its used readings and their moments.
type side struct {
	byKey map[[2]int]Record // replica and continuation to the record
	vals  []int64
	mo    reduce.Moments
}

func newSide(runs []Record, m int) side {
	s := side{byKey: make(map[[2]int]Record, len(runs))}
	for _, r := range runs {
		s.byKey[[2]int{r.Key.Replica, r.Key.Cont}] = r
		if rd := reading(r, m); r.Status == StatusOK && rd.OK && s.mo.Add(rd.Value) {
			s.vals = append(s.vals, rd.Value)
		}
	}
	slices.Sort(s.vals)
	return s
}

// Compare compares group b against group a for a measure (spec-experiments.md 8.4).
// The records of the two groups are matched by replica and continuation, and only
// pairs in which both are used and share a seed enter, so two different points of
// a PerPointSeeds experiment are unpaired (Pairs 0). An unknown measure is
// ErrUnknownMeasure and a group outside the result ErrUnknownRun.
func Compare(r Result, a, b Group, measure string) (Comparison, error) {
	m := slices.Index(r.Measures, measure)
	if m < 0 {
		return Comparison{}, fmt.Errorf("compare %q: %w", measure, ErrUnknownMeasure)
	}
	ra, okA := r.runs(a)
	rb, okB := r.runs(b)
	if !okA || !okB {
		return Comparison{}, fmt.Errorf("compare group %+v with %+v: %w", a, b, ErrUnknownRun)
	}
	sa, sb := newSide(ra, m), newSide(rb, m)
	var c Comparison
	c.MeanA, c.MeanB = readingOf(sa.mo.Mean()), readingOf(sb.mo.Mean())
	for i := range nQuantiles {
		q := Quantile(i)
		c.Dist[i] = distOf(sa.vals, sb.vals, q)
	}
	diffs, d, base, ok := pairUp(ra, sb, m)
	if !ok {
		return Comparison{}, fmt.Errorf("compare %q: sums: %w", measure, ErrRange)
	}
	c.Pairs = len(diffs)
	c.pairStats(diffs, d)
	c.growth(sa, sb, d, base)
	c.gain(sa, sb, d)
	return c, nil
}

// pairUp matches the records of a with those of b by replica and continuation and
// returns the differences B - A of the pairs in which both are used and share a
// seed, with their moments and the moments of the A readings. It returns false if
// a sum would overflow, which the bound on a reading rules out.
func pairUp(ra []Record, sb side, m int) (diffs []int64, d, base reduce.Moments, ok bool) {
	for _, rec := range ra {
		other, found := sb.byKey[[2]int{rec.Key.Replica, rec.Key.Cont}]
		x, y := reading(rec, m), reading(other, m)
		if !found || rec.Seed != other.Seed || rec.Status != StatusOK || other.Status != StatusOK || !x.OK || !y.OK {
			continue
		}
		diffs = append(diffs, y.Value-x.Value) // |x|, |y| <= 2^40: no wrap
		if !d.Add(y.Value-x.Value) || !base.Add(x.Value) {
			return nil, d, base, false
		}
	}
	return diffs, d, base, true
}

// distOf is the quantile q of two sorted samples and its difference.
func distOf(a, b []int64, q int) QuantilePair {
	var p QuantilePair
	if len(a) > 0 {
		p.A = Reading{reduce.Quantile(a, q), true}
	}
	if len(b) > 0 {
		p.B = Reading{reduce.Quantile(b, q), true}
	}
	if p.A.OK && p.B.OK {
		p.Diff = Reading{p.B.Value - p.A.Value, true}
	}
	return p
}

// pairStats sets the mean and median of the differences, the counts and the sign
// test.
func (c *Comparison) pairStats(diffs []int64, d reduce.Moments) {
	for _, v := range diffs {
		switch {
		case v > 0:
			c.Wins++
		case v < 0:
			c.Losses++
		default:
			c.Ties++
		}
	}
	c.Identical = c.Pairs > 0 && c.Ties == c.Pairs
	if c.Pairs == 0 {
		return
	}
	c.MeanDiff = readingOf(d.Mean())
	slices.Sort(diffs)
	c.MedianDiff = Reading{reduce.Quantile(diffs, 5000), true}
	c.SignTailPPM = readingOf(reduce.SignTailPPM(c.Wins, c.Losses))
}

// growth sets DiffOfMeans and RelativeBP. Over pairs, the growth is the sum of d
// over the sum of A in one division; unpaired it is the difference of the means
// over the mean of A from the exact sums.
func (c *Comparison) growth(a, b side, d, base reduce.Moments) {
	if a.mo.Count() == 0 || b.mo.Count() == 0 {
		return
	}
	na, nb := reduce.FromInt64(a.mo.Count()), reduce.FromInt64(b.mo.Count())
	sa, sb := a.mo.Sum(), b.mo.Sum()
	var k calc
	num := k.sub(k.mul(sb, na), k.mul(sa, nb)) // S_B*n_A - S_A*n_B
	both := k.mul(na, nb)
	unpaired := k.mul(sa, nb)
	if k.failed {
		return
	}
	c.DiffOfMeans = quotient(num, both)
	gnum, gden := num, unpaired
	if c.Pairs > 0 {
		gnum, gden = d.Sum(), base.Sum()
	}
	if gden.Sign() <= 0 {
		return
	}
	scaled := k.mul(gnum, reduce.FromInt64(10_000))
	if k.failed {
		return
	}
	c.RelativeBP, c.Clamped = clampedQuotient(scaled, gden)
}

// gain sets SdIndep, SdDiff and PairingGain (spec-experiments.md 7.4.2).
func (c *Comparison) gain(a, b side, d reduce.Moments) {
	c.SdDiff = readingOf(d.Std())
	va, ok1 := a.mo.Variance()
	vb, ok2 := b.mo.Variance()
	if sum, ok3 := va.Add(vb); ok1 && ok2 && ok3 {
		c.SdIndep = Reading{int64(reduce.IsqrtNearest(sum)), true} // below 2^41
	}
	if !c.SdIndep.OK || !c.SdDiff.OK || c.SdDiff.Value == 0 {
		return
	}
	if scaled, ok := reduce.FromInt64(c.SdIndep.Value).Mul(reduce.FromInt64(10_000)); ok {
		c.PairingGain = quotient(scaled, reduce.FromInt64(c.SdDiff.Value))
	}
}
