package design

import "math/bits"

// GridCount returns the number of points of the full grid over factors of the
// given level counts: their product (spec-experiments.md 4.2.1). It returns
// false for a level count below 1 or a product that overflows. No factors give
// one point.
func GridCount(levels []int) (int, bool) {
	for _, l := range levels {
		if l < 1 {
			return 0, false
		}
	}
	return CheckedCount(levels...)
}

// GridCoords returns the level index of every factor at point index of the full
// grid, the last factor fastest, in dst (reused when it has the capacity): the
// mixed-radix digits of index (spec-experiments.md 4.2.1). It returns false for
// an index outside [0, GridCount).
func GridCoords(index int, levels []int, dst []int) ([]int, bool) {
	n, ok := GridCount(levels)
	if !ok || index < 0 || index >= n {
		return nil, false
	}
	dst = grow(dst, len(levels))
	for f := len(levels) - 1; f >= 0; f-- {
		dst[f] = index % levels[f]
		index /= levels[f]
	}
	return dst, true
}

// OATCount returns the number of points of the one-at-a-time design,
// 1 + sum(L - 1) (spec-experiments.md 4.3.1). It returns false for a level count
// below 1 or a sum that overflows.
func OATCount(levels []int) (int, bool) {
	n := 1
	for _, l := range levels {
		if l < 1 || n > int(maxInt)-(l-1) {
			return 0, false
		}
		n += l - 1
	}
	return n, true
}

// OATCoords returns the level index of every factor at point index of the
// one-at-a-time design: point 0 is the baseline, level 0 everywhere, and then
// for each factor in order each level above 0 with the others at 0
// (spec-experiments.md 4.3.1). It returns false for an index outside
// [0, OATCount).
func OATCoords(index int, levels []int, dst []int) ([]int, bool) {
	n, ok := OATCount(levels)
	if !ok || index < 0 || index >= n {
		return nil, false
	}
	dst = grow(dst, len(levels))
	clear(dst)
	index-- // the baseline is point 0; after it the points are counted from 0
	for f, l := range levels {
		if index >= 0 && index < l-1 {
			dst[f] = index + 1
			break
		}
		index -= l - 1
	}
	return dst, true
}

func grow(dst []int, n int) []int {
	if cap(dst) < n {
		return make([]int, n)
	}
	return dst[:n]
}

// RangeValue returns the value of level k of a range factor [lo, hi] in a grid of
// steps levels: lo + rdiv(k*(hi-lo), steps-1), half to even, so the end points
// are exact (spec-experiments.md 3.2.3). The product is 128 bits and hi-lo may
// reach 2^64 - 1. It returns false for steps below 2, k outside [0, steps-1] or
// lo above hi.
func RangeValue(k, steps int, lo, hi int64) (int64, bool) {
	if steps < 2 || k < 0 || k >= steps || lo > hi {
		return 0, false
	}
	d := uint64(steps - 1)
	ph, pl := bits.Mul64(uint64(k), uint64(hi)-uint64(lo))
	q, r := bits.Div64(ph, pl, d) // k <= d, so the quotient is at most hi - lo
	// Half to even: up when the remainder is above half of d, and on an exact
	// half when q is odd. r < d, so d - r cannot wrap and r > d - r means 2r > d.
	if r > d-r || (r == d-r && q&1 == 1) {
		q++
	}
	return int64(uint64(lo) + q), true
}
