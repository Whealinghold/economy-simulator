package indicator

import "github.com/Whealinghold/economy-sim/internal/measure"

// Point is one value of a consumer's series at a tick. The library keeps no
// series: the consumer appends a Point after each window
// (spec-indicators.md 10.3).
type Point struct{ Tick, Value int64 }

// SeriesStats summarises a series. OK is false for an empty series, and then
// every other field is 0. Low and High are the first points, in slice order, of
// the lowest and highest value.
type SeriesStats struct {
	OK                      bool
	Count                   int64
	First, Last, Low, High  Point
	Mean                    int64 // the mean value, half to even
	SpreadOK, SpreadClamped bool  // SpreadOK is false for Low.Value <= 0
	Spread                  BP    // High / Low in basis points, clamped at MaxIndexBP
}

// Stats returns the first, last, lowest and highest point of a series, its
// count, its mean rounded half to even and the ratio High * 10 000 / Low
// (spec-indicators.md 10.3.2), which is undefined for a Low at or below 0. The
// sums are 128 bits, so no series overflows them.
func Stats(points []Point) SeriesStats {
	if len(points) == 0 {
		return SeriesStats{}
	}
	s := SeriesStats{OK: true, Count: int64(len(points)), First: points[0], Last: points[len(points)-1], Low: points[0], High: points[0]}
	sum := measure.FromInt64(0)
	for _, p := range points {
		if p.Value < s.Low.Value {
			s.Low = p
		}
		if p.Value > s.High.Value {
			s.High = p
		}
		sum, _ = sum.Add(measure.FromInt64(p.Value)) // 2^64 points would be needed to pass 2^127
	}
	mean, _ := measure.RoundDiv(sum, measure.FromInt64(s.Count))
	s.Mean, _ = measure.ToInt64(mean) // a mean lies between the lowest and highest value
	if s.Low.Value > 0 {
		s.Spread, s.SpreadClamped = ratioBP(s.High.Value, s.Low.Value)
		s.SpreadOK = true
	}
	return s
}

// ratioBP returns 10 000 * n / d, half to even, clamped at MaxIndexBP, for d > 0
// and any n; the product is 128 bits, so it cannot overflow.
func ratioBP(n, d int64) (BP, bool) {
	num, _ := measure.MulBP(measure.FromInt64(n)) // an int64 is far below the guard
	q, _ := measure.RoundDiv(num, measure.FromInt64(d))
	v, clamped := measure.ClampBP(q)
	return BP(v), clamped
}

// Rate is a ratio in basis points that can be undefined or clamped.
type Rate struct {
	OK      bool
	Clamped bool
	BP      BP
}

// Growth returns the growth from point from to point to in basis points,
// rdiv(10 000 * (to - from), from) signed and clamped to the range of
// MaxIndexBP with Clamped set (spec-indicators.md 5.7.1). It is undefined (OK
// false) for a base at or below 0.
func Growth(from, to Point) Rate {
	if from.Value <= 0 {
		return Rate{}
	}
	delta, _ := measure.FromInt64(to.Value).Sub(measure.FromInt64(from.Value)) // far below 2^127
	num, _ := measure.MulBP(delta)
	q, _ := measure.RoundDiv(num, measure.FromInt64(from.Value))
	v, clamped := measure.ClampBP(q)
	return Rate{OK: true, Clamped: clamped, BP: BP(v)}
}

// Diff is the change of a series between its first and last point.
type Diff struct {
	OK    bool
	Value int64 // last value less first value
	Ticks int64 // last tick less first tick
}

// Change returns the last value less the first value of a series and the ticks
// between them (spec-indicators.md 10.3.2). It is undefined (OK false) for a
// series of fewer than two points; err is ErrOverflow when the difference does
// not fit int64.
func Change(points []Point) (Diff, error) {
	if len(points) < 2 {
		return Diff{}, nil
	}
	first, last := points[0], points[len(points)-1]
	delta, ok := measure.ToInt64(mustSub(last.Value, first.Value))
	ticks, tok := measure.ToInt64(mustSub(last.Tick, first.Tick))
	if !ok || !tok {
		return Diff{}, wrap("Change", "difference", ErrOverflow)
	}
	return Diff{OK: true, Value: delta, Ticks: ticks}, nil
}

// mustSub returns a - b in 128 bits; two int64 values never pass 2^127.
func mustSub(a, b int64) measure.Int128 {
	v, _ := measure.FromInt64(a).Sub(measure.FromInt64(b))
	return v
}
