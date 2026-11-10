package verdict

import "math/bits"

// maxRegrowYears bounds the iteration of RegrowTicks: a level that needs more
// than this many years at the rate is not reachable (a rule constant, the same
// bound as the model's, detectors.py speed_limit_ticks).
const maxRegrowYears = 1000

// maxTicksPerYear keeps the iteration bound of RegrowTicks finite; no
// calendar of the library has a year longer than this.
const maxTicksPerYear = 1 << 12

// RegrowTicks returns the speed limit t_min of STY-S02: the ticks a count
// needs to regrow from trough to level*(10 000 - bandBP)/10 000 (rounded
// down) at the fastest rate of ratePerYearBP basis points a year, by integer
// iteration per tick with a carried remainder over 10 000 * ticksPerYear
// (POP 3.8, STY 17.4). The second result is false when the level is not
// reachable: a rate or a trough of 0 or less, or more than maxRegrowYears
// years. A trough already at the level takes 0 ticks.
func RegrowTicks(trough, level, bandBP, ratePerYearBP, ticksPerYear int64) (int64, bool) {
	m := min(max(bpScale-bandBP, 0), bpScale)
	// Cannot fail: level >= 0 and 0 <= m <= bpScale, so the quotient is at most level.
	target, _ := mulDivFloor(max(level, 0), m, bpScale)
	if trough >= target {
		return 0, true
	}
	if trough <= 0 || ratePerYearBP <= 0 || ticksPerYear <= 0 || ticksPerYear > maxTicksPerYear {
		return 0, false
	}
	den := uint64(bpScale) * uint64(ticksPerYear)
	h, carry := uint64(trough), uint64(0)
	for t := int64(1); t <= maxRegrowYears*ticksPerYear; t++ {
		hi, lo := bits.Mul64(h, uint64(ratePerYearBP))
		lo, c := bits.Add64(lo, carry, 0)
		hi += c
		if hi >= den {
			return 0, false // the quotient does not fit: far beyond any level
		}
		var add uint64
		add, carry = bits.Div64(hi, lo, den)
		h += add
		if h >= uint64(target) {
			return t, true
		}
	}
	return 0, false
}
