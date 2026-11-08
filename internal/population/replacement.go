package population

import (
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Replacement returns the replacement ratio r*, the ratio at which birth -
// death changes sign from negative to positive, found on the union of the two
// schedules' ratios (spec-population.md 5.7). It is on the first segment
// [lo, hi] of the union with g(lo) < 0 <= g(hi), g being the difference of the
// two schedules, at r* = lo + trunc(-g(lo) * (hi - lo) / (g(hi) - g(lo))), so a
// g of exactly 0 at a knot is the ratio at that knot. ok is false when there is
// no such segment (g never negative, never non-negative above a negative, or a
// crossing from positive to negative only) or on overflow.
func Replacement(birth, death []Knot) (ratio, lo, hi int64, ok bool) {
	xs, g, ok := differences(birth, death)
	if !ok {
		return 0, 0, 0, false
	}
	for i := 1; i < len(xs); i++ {
		if g[i-1] >= 0 || g[i] < 0 {
			continue
		}
		// Every operand is checked; the quotient is at most dx, so it cannot overflow.
		dg, okg := checked.Sub(g[i], g[i-1])
		dx, okx := checked.Sub(xs[i], xs[i-1])
		neg, okn := checked.Sub(0, g[i-1])
		if !okg || !okx || !okn {
			return 0, 0, 0, false
		}
		step, _ := checked.MulDiv(neg, dx, dg)
		return xs[i-1] + step, xs[i-1], xs[i], true
	}
	return 0, 0, 0, false
}

// differences returns the ascending union of the two schedules' ratios and
// birth - death at each. ok is false for an empty schedule or an overflow.
func differences(birth, death []Knot) (xs, g []int64, ok bool) {
	xs = make([]int64, 0, len(birth)+len(death))
	for _, k := range birth {
		xs = append(xs, k.Ratio)
	}
	for _, k := range death {
		xs = append(xs, k.Ratio)
	}
	slices.Sort(xs)
	xs = slices.Compact(xs)
	g = make([]int64, len(xs))
	for i, x := range xs {
		b, okb := eval(birth, x)
		d, okd := eval(death, x)
		if !okb || !okd {
			return nil, nil, false
		}
		if g[i], ok = checked.Sub(b, d); !ok {
			return nil, nil, false
		}
	}
	return xs, g, true
}

// GainWarn reports whether the condition of the warning population_gain holds:
// rstar * dRate * window > 10 000 * dRatio * ticksPerYear, with dRate / dRatio
// the slope of birth - death at r* and window the longest effective welfare
// window in ticks (spec-population.md 5.4). Equality is false. A left side that
// overflows is true and a right side that does, with the left not, false: the
// validator's bounds keep both far below int64.
func GainWarn(rstar, dRate, dRatio, window, ticksPerYear int64) bool {
	l, okl := mul3(rstar, dRate, window)
	r, okr := mul3(10_000, dRatio, ticksPerYear)
	switch {
	case !okl:
		return true
	case !okr:
		return false
	}
	return l > r
}

func mul3(a, b, c int64) (int64, bool) {
	ab, ok := checked.Mul(a, b)
	if !ok {
		return 0, false
	}
	return checked.Mul(ab, c)
}

// Gain returns the gain k of spec-population.md 5.4 at eta 1 in basis points a
// year: trunc(rstar * dRate / dRatio). ok is false for a dRatio below 1 or an
// overflow.
func Gain(rstar, dRate, dRatio int64) (gain int64, ok bool) {
	if dRatio < 1 {
		return 0, false
	}
	return checked.MulDiv(rstar, dRate, dRatio)
}
