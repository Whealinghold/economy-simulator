package verdict_test

import (
	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// The helpers of the detector tests: an integer port of the synthetic series of
// doc/specs/models/sty/detectors.py (a 63-bit congruential generator, the
// triangular seasonal swing, the compound yearly drift), so that the tests
// reproduce the rows of detectors.out exactly.

const tpy = 365 // detectors.py TPY

type lcg struct{ s uint64 }

func newLCG(seed uint64) *lcg { return &lcg{s: seed & (1<<63 - 1)} }

// below mirrors Lcg.below: the high bits of a 63-bit LCG modulo n.
func (g *lcg) below(n uint64) uint64 {
	g.s = (g.s*6364136223846793005 + 1442695040888963407) & (1<<63 - 1)
	return (g.s >> 20) % n
}

// floorDiv is Python's // on an int numerator and a positive denominator.
func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && a < 0 {
		q--
	}
	return q
}

// tri mirrors detectors.py tri: a triangular swing in thousandths.
func tri(t int64) int64 {
	u := (t % tpy) * 4000 / tpy
	switch {
	case u < 1000:
		return u
	case u < 3000:
		return 2000 - u
	}
	return u - 4000
}

// priceSeries mirrors detectors.py price_series: base 10 000, a seasonal swing
// of +-amp bp, a compound drift per year and bounded noise.
func priceSeries(years int, amp, drift, noise int64, seed uint64) []int64 {
	g := newLCG(seed)
	out := make([]int64, 0, int(years)*tpy)
	level := int64(10000 * 10000)
	for t := int64(0); t < int64(years)*tpy; t++ {
		if t > 0 && t%tpy == 0 {
			level = level * (10000 + drift) / 10000
		}
		base := level / 10000
		p := base * (10000 + floorDiv(amp*tri(t), 1000)) / 10000
		p = p * (10000 + int64(g.below(uint64(2*noise+1))) - noise) / 10000
		out = append(out, max(p, 1))
	}
	return out
}

func yearWindows(s []int64) [][]int64 {
	n := len(s) / tpy
	out := make([][]int64, n)
	for k := range out {
		out[k] = s[k*tpy : (k+1)*tpy]
	}
	return out
}

func sum(xs []int64) int64 {
	var s int64
	for _, x := range xs {
		s += x
	}
	return s
}

// meanInt mirrors detectors.py mean_int: the mean rounded half to even.
func meanInt(xs []int64) int64 {
	n := int64(len(xs))
	q, r := sum(xs)/n, sum(xs)%n
	if 2*r > n || (2*r == n && q%2 == 1) {
		q++
	}
	return q
}

func rangeBP(xs []int64) int64 {
	lo, hi := xs[0], xs[0]
	for _, x := range xs {
		lo, hi = min(lo, x), max(hi, x)
	}
	return hi * 10000 / lo
}

// winTick is the first tick of year window k.
func winTick(k int) int64 { return int64(k) * tpy }

// linksOf returns the year-mean chain links in bp as detectors.py r1 computes
// them (floor), one point per window after the first, at the window's tick.
func linksOf(prices []int64) verdict.Series {
	w := yearWindows(prices)
	var pts []verdict.Point
	for k := 1; k < len(w); k++ {
		pts = append(pts, verdict.Point{Tick: winTick(k), Value: meanInt(w[k]) * 10000 / meanInt(w[k-1])})
	}
	return verdict.Series{Points: pts}
}

// rangesOf returns the range ratio of each year window.
func rangesOf(prices []int64) verdict.Series {
	var pts []verdict.Point
	for k, w := range yearWindows(prices) {
		pts = append(pts, verdict.Point{Tick: winTick(k), Value: rangeBP(w)})
	}
	return verdict.Series{Points: pts}
}

func volumeFlat(years int) []int64 {
	v := make([]int64, years*tpy)
	for i := range v {
		v[i] = 100
	}
	return v
}

// volumeDecaying mirrors detectors.py volumes_decaying: 100 a tick falling to
// yearlyBP of the year before each year.
func volumeDecaying(years int, yearlyBP int64) []int64 {
	out := make([]int64, 0, years*tpy)
	level := int64(100 * 10000)
	for t := int64(0); t < int64(years)*tpy; t++ {
		if t > 0 && t%tpy == 0 {
			level = level * yearlyBP / 10000
		}
		out = append(out, level/10000)
	}
	return out
}

// volumeOf returns the sum of each year window as a window series.
func volumeOf(v []int64) []verdict.Point {
	var pts []verdict.Point
	for k, w := range yearWindows(v) {
		pts = append(pts, verdict.Point{Tick: winTick(k), Value: sum(w)})
	}
	return pts
}

// ticksOf returns a one-tick series from values.
func ticksOf(v []int64) []verdict.Point {
	pts := make([]verdict.Point, len(v))
	for i, x := range v {
		pts[i] = verdict.Point{Tick: int64(i), Value: x}
	}
	return pts
}

// priors returns every key set to its prior of STY 6.2 (detectors.py R).
func priors() verdict.Thresholds {
	t := verdict.NewThresholds()
	for name, v := range map[string]int64{
		"r1.rise_bp": 3000, "r1.windows": 2, "r1.m_factor_bp": 30000,
		"r2.vol_bp": 8000, "r2.share_bp": 5000, "r2.unsold_bp": 5000, "r2.top_bp": 7000, "r2.windows": 2,
		"r3.fall_bp": 2000, "r3.recover_bp": 1000, "r3.horizon_factor": 50000, "r3.horizon": 730, "r3.role_ticks": 365,
		"r4.vol_bp": 5000, "r4.windows": 2,
		"r5.pinned_bp": 5000, "r5.pinned_ticks": 60, "r5.freeze_ticks": 60,
		"r6.factor": 30000, "r6.growth_bp": 2500, "r6.windows": 2, "r6.swing_bp": 1000,
		"r7.disp_bp": 50, "r7.spike_factor": 20000, "r7.first_ticks": 60,
		"r8.offset_bp": 3000, "r8.unmet_bp": 3000,
		"r9.fall_bp": 3000, "r9.windows": 2,
		"stale_max_bp": 1000, "active_min_bp": 5000, "min_windows": 3,
	} {
		t.Set(name, v)
	}
	return t
}

// without returns the thresholds with the named key unset.
func without(t verdict.Thresholds, name string) verdict.Thresholds {
	out := append(verdict.Thresholds(nil), t...)
	for i := range out {
		if out[i].Name == name {
			out[i].Set = false
		}
	}
	return out
}

// with returns the thresholds with the named key set to v.
func with(t verdict.Thresholds, name string, v int64) verdict.Thresholds {
	out := append(verdict.Thresholds(nil), t...)
	out.Set(name, v)
	return out
}
