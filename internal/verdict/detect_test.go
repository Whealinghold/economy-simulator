package verdict_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// check compares a verdict to the class, onset and reason expected, with the
// detector, the case and the verdict in the message (testing.md 3.3).
func check(t *testing.T, name string, got verdict.Verdict, class verdict.Class, onset int64, reason string) {
	t.Helper()
	if got.Class != class || got.Reason != reason || (class == verdict.Detected && got.Onset != onset) {
		t.Errorf("%s: got %s (class %s onset %d reason %q), want class %s onset %d reason %q",
			name, got, got.Class, got.Onset, got.Reason, class, onset, reason)
	}
}

// TestDetect_R1_RegimeAndHealthy verifies R1 on the rows of detectors.out and
// at its threshold.
//
// detectors.out line 5: inflation of 40% a year is detected at "onset window
// 2", the third year window (tick 730); line 2: the healthy seasonal series is
// none. Boundary (at or above fires): links of 10 000 + 3 000 - 1 in each of
// two windows are none, 13 000 fires; a single window above is none. The money
// clause: M at 30 000 bp of its start at two successive closes fires, once does
// not. A clamped link fires at its tick.
//
// Requirement: FR-STY-061
func TestDetect_R1_RegimeAndHealthy(t *testing.T) {
	t.Parallel()
	th := priors()
	run := func(in verdict.Input) verdict.Verdict { return verdict.Detect(verdict.R1, in, th) }
	healthy := priceSeries(4, 2500, 0, 200, 1)
	inflating := priceSeries(4, 2500, 4000, 200, 1)
	check(t, "detectors.out:2 healthy seasonal", run(verdict.Input{Links: []verdict.Series{linksOf(healthy)}}), verdict.None, 0, "")
	check(t, "detectors.out:5 inflation 40%", run(verdict.Input{Links: []verdict.Series{linksOf(inflating)}}), verdict.Detected, winTick(2), "")

	link := func(vals ...int64) verdict.Series {
		s := verdict.Series{}
		for i, v := range vals {
			s.Points = append(s.Points, verdict.Point{Tick: winTick(i + 1), Value: v})
		}
		return s
	}
	for _, tc := range []struct {
		name  string
		in    verdict.Input
		class verdict.Class
		onset int64
	}{
		{"one below the threshold", verdict.Input{Links: []verdict.Series{link(12999, 12999)}}, verdict.None, 0},
		{"at the threshold", verdict.Input{Links: []verdict.Series{link(13000, 13000)}}, verdict.Detected, winTick(2)},
		{"one above", verdict.Input{Links: []verdict.Series{link(13001, 13001)}}, verdict.Detected, winTick(2)},
		{"a single window above is not a run", verdict.Input{Links: []verdict.Series{link(20000, 10000)}}, verdict.None, 0},
		{"a break resets the run", verdict.Input{Links: []verdict.Series{link(13000, 10000, 13000)}}, verdict.None, 0},
		{"a class chain fires too", verdict.Input{Links: []verdict.Series{link(10000, 10000), link(13000, 13000)}}, verdict.Detected, winTick(2)},
		{"money at two closes", verdict.Input{Links: []verdict.Series{link(10000, 10000)}, MInit: 100, M: []verdict.Point{{Tick: 0, Value: 100}, {Tick: 365, Value: 300}, {Tick: 730, Value: 300}}}, verdict.Detected, 730},
		{"money one close below", verdict.Input{Links: []verdict.Series{link(10000, 10000)}, MInit: 100, M: []verdict.Point{{Tick: 0, Value: 100}, {Tick: 365, Value: 299}, {Tick: 730, Value: 300}}}, verdict.None, 0},
		{"money without a start is ignored", verdict.Input{Links: []verdict.Series{link(10000, 10000)}, M: []verdict.Point{{Tick: 0, Value: 100}, {Tick: 365, Value: 900}, {Tick: 730, Value: 900}}}, verdict.None, 0},
		{"a clamped link fires", verdict.Input{Links: []verdict.Series{{Points: link(10000, 10000).Points, Clamped: []bool{false, true}}}}, verdict.Detected, winTick(2)},
	} {
		check(t, tc.name, run(tc.in), tc.class, tc.onset, "")
	}
	// A clamped link before a run is the earlier onset.
	c := link(13000, 13000, 13000)
	c.Clamped = []bool{true}
	check(t, "clamped before the run", run(verdict.Input{Links: []verdict.Series{c}}), verdict.Detected, winTick(1), "")
}

// TestDetect_R2_RegimeAndHealthy verifies R2.
//
// detectors.out lines 22 and 23: a pool share of 60% twice with volume falling
// to 36% a year is detected, and with flat volume is none. Boundary: the volume
// of the last window against 8 000 bp of the first (below fires, equal does
// not); the share at 5 000 for two closes; an unsold share at 5 000 and rising
// for two closes; the top-decile share at 7 000.
//
// Requirement: FR-STY-062
func TestDetect_R2_RegimeAndHealthy(t *testing.T) {
	t.Parallel()
	th := priors()
	pool := ticksOf([]int64{3000, 6000, 6500})
	fall := volumeOf(volumeDecaying(4, 6000))
	flat := volumeOf(volumeFlat(4))
	run := func(in verdict.Input) verdict.Verdict { return verdict.Detect(verdict.R2, in, th) }
	check(t, "detectors.out:22 pool share, volume falling", run(verdict.Input{Volume: fall, MoneyOut: pool}), verdict.Detected, 2, "")
	check(t, "detectors.out:23 pool share, volume flat", run(verdict.Input{Volume: flat, MoneyOut: pool}), verdict.None, 0, "")

	vol := func(first, last int64) []verdict.Point {
		return []verdict.Point{{Tick: 0, Value: first}, {Tick: 365, Value: first}, {Tick: 730, Value: last}}
	}
	for _, tc := range []struct {
		name  string
		in    verdict.Input
		class verdict.Class
		onset int64
	}{
		{"volume equal to 8000 bp does not fall", verdict.Input{Volume: vol(10000, 8000), MoneyOut: ticksOf([]int64{9000, 9000})}, verdict.None, 0},
		{"volume one below 8000 bp falls", verdict.Input{Volume: vol(10000, 7999), MoneyOut: ticksOf([]int64{9000, 9000})}, verdict.Detected, 1},
		{"share one below", verdict.Input{Volume: vol(10000, 100), MoneyOut: ticksOf([]int64{4999, 4999})}, verdict.None, 0},
		{"share at", verdict.Input{Volume: vol(10000, 100), MoneyOut: ticksOf([]int64{5000, 5000})}, verdict.Detected, 1},
		{"share once is not enough", verdict.Input{Volume: vol(10000, 100), MoneyOut: ticksOf([]int64{9000, 0, 9000})}, verdict.None, 0},
		{"unsold at threshold and rising", verdict.Input{Volume: vol(10000, 100), Unsold: [][]verdict.Point{ticksOf([]int64{4000, 5000, 5001})}}, verdict.Detected, 2},
		{"unsold above but flat is not rising", verdict.Input{Volume: vol(10000, 100), Unsold: [][]verdict.Point{ticksOf([]int64{6000, 6000, 6000})}}, verdict.None, 0},
		{"unsold one below", verdict.Input{Volume: vol(10000, 100), Unsold: [][]verdict.Point{ticksOf([]int64{4000, 4500, 4999})}}, verdict.None, 0},
		{"top decile at 7000", verdict.Input{Volume: vol(10000, 100), Top: ticksOf([]int64{6999, 7000})}, verdict.Detected, 1},
		{"top decile one below", verdict.Input{Volume: vol(10000, 100), Top: ticksOf([]int64{6999, 6999})}, verdict.None, 0},
		{"a first window of 0 never falls", verdict.Input{Volume: vol(0, 0), MoneyOut: ticksOf([]int64{9000, 9000})}, verdict.None, 0},
	} {
		check(t, tc.name, run(tc.in), tc.class, tc.onset, "")
	}
}

// hh600 returns the household series of detectors.py discrimination(): 600
// falling to 200 over a year, then flat or recovering by one a tick.
func hh600(years int, recover bool) []verdict.Point {
	v := make([]int64, years*tpy)
	for t := range v {
		switch {
		case t < tpy:
			v[t] = 600 - (400*int64(t))/tpy
		case recover:
			v[t] = min(600, 200+int64(t-tpy))
		default:
			v[t] = 200
		}
	}
	return ticksOf(v)
}

// TestDetect_R3_RegimeAndHealthy verifies R3 on the rows of detectors.out and
// at its thresholds.
//
// detectors.out lines 11 to 13: 600 households falling to 200 with no recovery
// is "unknown, needs a longer run (horizon 915, available 729)" over 3 years,
// "detected, onset tick 365" over 6 years, and none when it recovers. The
// horizon is max(730, 5 * t_min) with t_min from 200 to 540 at 20 000 bp a year
// (183 ticks, so 915). Boundary: a fall to exactly 8 000 bp of the start fires,
// one tick above does not; a role at 0 for 365 ticks fires, 364 does not; a
// role that never started is ignored.
//
// Requirement: FR-STY-063
func TestDetect_R3_RegimeAndHealthy(t *testing.T) {
	t.Parallel()
	th := priors()
	in := func(h []verdict.Point) verdict.Input {
		return verdict.Input{Households: [][]verdict.Point{h}, Rate: 20000, TicksPerYear: tpy}
	}
	run := func(in verdict.Input) verdict.Verdict { return verdict.Detect(verdict.R3, in, th) }
	check(t, "detectors.out:11 three years", run(in(hh600(3, false))), verdict.Unknown, 0, verdict.ReasonNeedsLongerRun)
	check(t, "detectors.out:12 six years", run(in(hh600(6, false))), verdict.Detected, 365, "")
	check(t, "detectors.out:13 recovers", run(in(hh600(3, true))), verdict.None, 0, "")

	tail := func(first, low int64) []verdict.Point {
		return []verdict.Point{{Tick: 0, Value: first}, {Tick: 10, Value: low}, {Tick: 3000, Value: low}}
	}
	check(t, "fall to exactly 8000 bp", run(in(tail(1000, 800))), verdict.Detected, 10, "")
	check(t, "one above 8000 bp", run(in(tail(1000, 801))), verdict.None, 0, "")
	// 1 000 -> 900 is the recovery target: 10 000 - 1 000 bp of the start.
	rec := []verdict.Point{{Tick: 0, Value: 1000}, {Tick: 10, Value: 500}, {Tick: 20, Value: 900}, {Tick: 3000, Value: 500}}
	check(t, "back at the recovery target", run(in(rec)), verdict.None, 0, "")
	rec[2].Value = 899
	check(t, "one below the recovery target", run(in(rec)), verdict.Detected, 10, "")
	// a zero speed (no rate): the horizon is the floor of 730 ticks.
	noRate := verdict.Input{Households: [][]verdict.Point{tail(1000, 500)}}
	check(t, "no rate reads the horizon floor", run(noRate), verdict.Detected, 10, "")
	short := verdict.Input{Households: [][]verdict.Point{{{Tick: 0, Value: 1000}, {Tick: 10, Value: 500}, {Tick: 739, Value: 500}}}}
	check(t, "run ends one tick before the floor", run(short), verdict.Unknown, 0, verdict.ReasonNeedsLongerRun)
	short.Households[0][2].Tick = 740
	check(t, "run ends at the floor", run(short), verdict.Detected, 10, "")
	check(t, "a single point", run(verdict.Input{Households: [][]verdict.Point{{{Tick: 0, Value: 5}}}}), verdict.Unknown, 0, verdict.ReasonWindows)
	check(t, "a region with no start is not judged", run(in([]verdict.Point{{Tick: 0, Value: 0}, {Tick: 1, Value: 0}})), verdict.None, 0, "")
	check(t, "unreachable speed uses the floor", run(verdict.Input{Households: [][]verdict.Point{tail(1000, 1)}, Rate: 1, TicksPerYear: tpy}), verdict.Detected, 10, "")

	role := func(zeros int) verdict.Input {
		v := []int64{3}
		for i := 0; i < zeros; i++ {
			v = append(v, 0)
		}
		return verdict.Input{Roles: [][]verdict.Point{ticksOf(v)}}
	}
	check(t, "a role at 0 for 365 ticks", run(role(365)), verdict.Detected, 365, "")
	check(t, "a role at 0 for 364 ticks", run(role(364)), verdict.None, 0, "")
	check(t, "a role that never started", run(verdict.Input{Roles: [][]verdict.Point{ticksOf([]int64{0, 0, 0})}}), verdict.None, 0, "")
	check(t, "a role wins over an unknown", run(verdict.Input{Roles: role(365).Roles, Households: [][]verdict.Point{{{Tick: 0, Value: 1}}}}), verdict.Detected, 365, "")
	check(t, "an unknown stays unknown beside a none", run(verdict.Input{Households: [][]verdict.Point{{{Tick: 0, Value: 1}}, tail(1000, 900)}}), verdict.Unknown, 0, verdict.ReasonWindows)
}

// TestDetect_R4_RegimeAndHealthy verifies R4.
//
// detectors.out lines 20 and 21: volume falling to 36% a year with households
// intact is detected at "onset window 3" (tick 1 095); with households at 70% of
// their start the collapse is R3's and R4 is none. Boundary: volume at exactly
// 5 000 bp of the first window does not count (below), one under does.
//
// Requirement: FR-STY-064
func TestDetect_R4_RegimeAndHealthy(t *testing.T) {
	t.Parallel()
	th := priors()
	fall := volumeOf(volumeDecaying(4, 6000))
	hh := func(end int64) [][]verdict.Point {
		return [][]verdict.Point{{{Tick: 0, Value: 600}, {Tick: 1000, Value: end}}}
	}
	run := func(in verdict.Input) verdict.Verdict { return verdict.Detect(verdict.R4, in, th) }
	check(t, "detectors.out:20 households fine", run(verdict.Input{Volume: fall, Households: hh(600)}), verdict.Detected, 1095, "")
	check(t, "detectors.out:21 collapse is R3", run(verdict.Input{Volume: fall, Households: hh(420)}), verdict.None, 0, "")
	check(t, "households at exactly 8000 bp are a collapse", run(verdict.Input{Volume: fall, Households: hh(480)}), verdict.None, 0, "")
	check(t, "households one above 8000 bp are no collapse", run(verdict.Input{Volume: fall, Households: hh(481)}), verdict.Detected, 1095, "")
	vol := func(a, b int64) []verdict.Point {
		return []verdict.Point{{Tick: 0, Value: 1000}, {Tick: 365, Value: a}, {Tick: 730, Value: b}}
	}
	check(t, "volume at 5000 bp twice", run(verdict.Input{Volume: vol(500, 500), Households: hh(600)}), verdict.None, 0, "")
	check(t, "volume one under twice", run(verdict.Input{Volume: vol(499, 499), Households: hh(600)}), verdict.Detected, 730, "")
	check(t, "volume under once", run(verdict.Input{Volume: vol(499, 900), Households: hh(600)}), verdict.None, 0, "")
	check(t, "no households series", run(verdict.Input{Volume: fall}), verdict.Unknown, 0, verdict.ReasonReferenceMissing)
}

// TestDetect_R5_RegimeAndHealthy verifies R5a and R5b.
//
// detectors.out lines 14 to 16: a pinned share for 80 ticks is detected, for 40
// ticks none, a frozen run of 70 ticks is detected. Boundary: 59 pinned ticks
// are none and 60 fire (the onset is the 60th tick); a share one under 5 000
// bp does not pin; the longest run is the Value of the verdict.
//
// Requirement: FR-STY-065
func TestDetect_R5_RegimeAndHealthy(t *testing.T) {
	t.Parallel()
	th := priors()
	pin := func(n int, share int64) verdict.Input {
		v := make([]int64, n+100)
		for i := 0; i < n; i++ {
			v[i] = share
		}
		return verdict.Input{Pinned: [][]verdict.Point{ticksOf(v)}, Outcomes: 5}
	}
	a := func(in verdict.Input) verdict.Verdict { return verdict.Detect(verdict.R5a, in, th) }
	check(t, "detectors.out:14 pinned 80", a(pin(80, 8000)), verdict.Detected, 59, "")
	if got := a(pin(80, 8000)); got.Value != 80 || got.Threshold != 60 {
		t.Errorf("R5a pinned 80: value %d threshold %d, want 80 and 60", got.Value, got.Threshold)
	}
	check(t, "detectors.out:15 pinned 40", a(pin(40, 8000)), verdict.None, 0, "")
	check(t, "59 pinned ticks", a(pin(59, 5000)), verdict.None, 0, "")
	check(t, "60 pinned ticks", a(pin(60, 5000)), verdict.Detected, 59, "")
	check(t, "share one below", a(pin(80, 4999)), verdict.None, 0, "")

	frozen := func(n int) verdict.Input {
		v := make([]int64, n+30)
		for i := 0; i < n; i++ {
			v[i] = 1
		}
		return verdict.Input{Frozen: [][]verdict.Point{ticksOf(v)}}
	}
	b := func(in verdict.Input) verdict.Verdict { return verdict.Detect(verdict.R5b, in, th) }
	check(t, "detectors.out:16 frozen 70", b(frozen(70)), verdict.Detected, 59, "")
	check(t, "59 frozen ticks", b(frozen(59)), verdict.None, 0, "")
	check(t, "60 frozen ticks", b(frozen(60)), verdict.Detected, 59, "")
	// a second market rescues nothing and a detected market wins over a short one.
	two := verdict.Input{Frozen: [][]verdict.Point{ticksOf([]int64{1, 1}), frozen(61).Frozen[0]}}
	check(t, "a detected market wins over a short series", b(two), verdict.Detected, 59, "")
	two.Frozen[1] = ticksOf([]int64{0, 0, 0})
	check(t, "a short series is unknown", b(two), verdict.Unknown, 0, verdict.ReasonWindows)
}

// TestDetect_R6_RegimeAndHealthy verifies R6.
//
// detectors.out lines 3 and 8: the healthy seasonal series read by year windows
// is none, and a cobweb whose range doubles each year is detected at "onset
// window 3" (tick 1 095). Boundary: the range must be above 30 000 (equal
// does not count) and above the previous by more than 2 500 bp (equal does not
// count), twice in a row; a role-share swing of 1 000 bp in two windows fires,
// 999 does not.
//
// Requirement: FR-STY-066
func TestDetect_R6_RegimeAndHealthy(t *testing.T) {
	t.Parallel()
	th := priors()
	var cobweb []int64
	for y := 0; y < 4; y++ {
		cobweb = append(cobweb, priceSeries(1, min(1500*int64(1)<<y, 9000), 0, 200, uint64(7+y))...)
	}
	healthy := priceSeries(4, 2500, 0, 200, 1)
	run := func(in verdict.Input) verdict.Verdict { return verdict.Detect(verdict.R6, in, th) }
	full := func(s verdict.Series) verdict.Input { return verdict.Input{Ranges: []verdict.Series{s}, Active: 10000} }
	check(t, "detectors.out:3 healthy seasonal", run(full(rangesOf(healthy))), verdict.None, 0, "")
	check(t, "detectors.out:8 cobweb doubling", run(full(rangesOf(cobweb))), verdict.Detected, winTick(3), "")

	rng := func(vals ...int64) verdict.Input {
		s := verdict.Series{}
		for i, v := range vals {
			s.Points = append(s.Points, verdict.Point{Tick: winTick(i), Value: v})
		}
		return full(s)
	}
	// 20 000 -> 30 001 -> 40 000: 30 001 is above the factor and above 25 000 (+25%).
	check(t, "factor one above, growth above", run(rng(20000, 30001, 40001)), verdict.Detected, winTick(2), "")
	check(t, "factor at 30000 does not count", run(rng(20000, 30000, 40000)), verdict.None, 0, "")
	check(t, "growth at exactly +25% does not count", run(rng(40000, 50000, 62500)), verdict.None, 0, "")
	check(t, "growth one above +25% twice", run(rng(40000, 50001, 62502)), verdict.Detected, winTick(2), "")
	check(t, "one window is not a run", run(rng(20000, 40000, 40000)), verdict.None, 0, "")
	sw := func(a, b int64) verdict.Input {
		return verdict.Input{Swings: [][]verdict.Point{{{Tick: 0, Value: a}, {Tick: 365, Value: b}}}, Active: 10000}
	}
	check(t, "swing at 1000 bp twice", run(sw(1000, 1000)), verdict.Detected, 365, "")
	check(t, "swing 999 bp", run(sw(999, 1000)), verdict.None, 0, "")
	check(t, "swing once", run(sw(5000, 0)), verdict.None, 0, "")
}

// TestDetect_SeasonalIsNotACobweb verifies the discrimination of STY 4.1.3 and
// 17.8: a stationary seasonal series has no pair of year windows that grows,
// while windows of 120 ticks read growth in several pairs, and at a swing of
// 60% five pairs fail both conjuncts of STY-P03; R6 over either reading is
// none. This documents why the caller passes year windows.
//
// detectors.out lines 57 to 60 (amp, highest year range, year pairs that
// grow, 120-tick pairs, grow, fact): 2500 17153 0 23 3 0; 4000 23979 0 23 6 0;
// 5000 30802 0 23 4 0; 6000 41015 0 23 5 5.
//
// Requirement: FR-STY-008
func TestDetect_SeasonalIsNotACobweb(t *testing.T) {
	t.Parallel()
	th := priors()
	for _, tc := range []struct {
		amp                int64
		maxRange           int64
		grow120, fact120   int
		yearGrow, yearFact int
		pairs120           int
	}{
		{2500, 17153, 3, 0, 0, 0, 23},
		{4000, 23979, 6, 0, 0, 0, 23},
		{5000, 30802, 4, 0, 0, 0, 23},
		{6000, 41015, 5, 5, 0, 0, 23},
	} {
		s := priceSeries(8, tc.amp, 0, 200, 3)
		yr := rangesOf(s)
		hi := int64(0)
		for _, p := range yr.Points {
			hi = max(hi, p.Value)
		}
		if hi != tc.maxRange {
			t.Errorf("amp %d: highest year range = %d, want %d (detectors.out)", tc.amp, hi, tc.maxRange)
		}
		var r120 []verdict.Point
		for i := 0; i+120 <= len(s); i += 120 {
			r120 = append(r120, verdict.Point{Tick: int64(i), Value: rangeBP(s[i : i+120])})
		}
		grow := func(p []verdict.Point) (pairs, g, f int) {
			for k := 1; k < len(p); k++ {
				pairs++
				if p[k].Value*10000 > p[k-1].Value*11000 {
					g++
					if p[k].Value > 20000 {
						f++
					}
				}
			}
			return
		}
		pairs, g, f := grow(r120)
		if pairs != tc.pairs120 || g != tc.grow120 || f != tc.fact120 {
			t.Errorf("amp %d: 120-tick windows pairs/grow/fact = %d/%d/%d, want %d/%d/%d", tc.amp, pairs, g, f, tc.pairs120, tc.grow120, tc.fact120)
		}
		if _, g, f := grow(yr.Points); g != tc.yearGrow || f != tc.yearFact {
			t.Errorf("amp %d: year windows grow/fact = %d/%d, want %d/%d", tc.amp, g, f, tc.yearGrow, tc.yearFact)
		}
		for name, ser := range map[string]verdict.Series{"year": yr, "120-tick": {Points: r120}} {
			if v := verdict.Detect(verdict.R6, verdict.Input{Ranges: []verdict.Series{ser}, Active: 10000}, th); v.Class != verdict.None {
				t.Errorf("amp %d: R6 over %s windows = %s, want none (the regime's rule is stricter than the fact's)", tc.amp, name, v)
			}
		}
	}
}

// TestDetect_R7_RegimeAndHealthy verifies R7.
//
// detectors.out line 17: a lowest dispersion of 10 bp is detected. Boundary:
// 49 bp fires and 50 does not (below); completions exceeding twice their mean
// fire, equal to twice the mean do not: over 60 ticks of one completion a tick
// the mean is 1 and a spike of 20 000 bp is a tick of 2 (not above) or 3.
//
// Requirement: FR-STY-067
func TestDetect_R7_RegimeAndHealthy(t *testing.T) {
	t.Parallel()
	th := priors()
	run := func(in verdict.Input) verdict.Verdict { return verdict.Detect(verdict.R7, in, th) }
	disp := func(v int64) verdict.Input {
		return verdict.Input{Dispersion: [][]verdict.Point{{{Tick: 0, Value: 900}, {Tick: 365, Value: v}}}, Outcomes: 4}
	}
	check(t, "detectors.out:17 lockstep 10 bp", run(disp(10)), verdict.Detected, 365, "")
	check(t, "49 bp", run(disp(49)), verdict.Detected, 365, "")
	check(t, "50 bp", run(disp(50)), verdict.None, 0, "")

	comp := func(spike int64) verdict.Input {
		v := make([]int64, 60)
		for i := range v {
			v[i] = 1
		}
		v[7] = spike
		return verdict.Input{Completions: [][]verdict.Point{ticksOf(v)}}
	}
	// 59 ones and a spike s: mean (59 + s)/60; s > 2 * (59 + s)/60 <=> 60 s > 118 + 2 s <=> s > 2.03.
	check(t, "a tick of 2 is not above twice the mean", run(comp(2)), verdict.None, 0, "")
	check(t, "a tick of 3 is above", run(comp(3)), verdict.Detected, 7, "")
	short := verdict.Input{Completions: [][]verdict.Point{ticksOf([]int64{1, 9, 1})}}
	check(t, "fewer ticks than first_ticks", run(short), verdict.Unknown, 0, verdict.ReasonWindows)
	both := comp(1)
	both.Dispersion = disp(10).Dispersion
	both.Outcomes = 4
	check(t, "either part fires", run(both), verdict.Detected, 365, "")
	nothing := comp(1)
	nothing.Dispersion = disp(900).Dispersion
	check(t, "outcomes 0 makes the dispersion part unknown", run(nothing), verdict.Unknown, 0, verdict.ReasonOutcomes)
	nothing.Outcomes = 1
	check(t, "neither part fires", run(nothing), verdict.None, 0, "")
	check(t, "a first_ticks of 0 is unknown", run(verdict.Input{Completions: [][]verdict.Point{ticksOf([]int64{1})}}), verdict.Unknown, 0, verdict.ReasonWindows)
	zero := with(th, "r7.first_ticks", 0)
	if got := verdict.Detect(verdict.R7, comp(1), zero); got.Class != verdict.Unknown {
		t.Errorf("R7 with first_ticks 0 = %s, want unknown", got)
	}
}

// TestDetect_R8_RegimeAndHealthy verifies R8.
//
// detectors.out lines 18 and 19: an offset of 4 000 bp with 4 000 bp unmet is
// detected, and with no steady-state price is unknown. Boundary: offset 2 999
// is none and 3 000 fires (at or above), also negative; unmet 2 999 is none
// and 3 000 fires; the two must hold in the same window.
//
// Requirement: FR-STY-068
func TestDetect_R8_RegimeAndHealthy(t *testing.T) {
	t.Parallel()
	th := priors()
	in := func(off, unmet int64) verdict.Input {
		return verdict.Input{
			Offsets: []verdict.Series{{Points: []verdict.Point{{Tick: 0, Value: 0}, {Tick: 365, Value: off}, {Tick: 730, Value: 0}}}},
			Unmet:   []verdict.Point{{Tick: 0, Value: 0}, {Tick: 365, Value: unmet}, {Tick: 730, Value: 0}},
			Active:  10000, HasReference: true,
		}
	}
	run := func(in verdict.Input) verdict.Verdict { return verdict.Detect(verdict.R8, in, th) }
	check(t, "detectors.out:18 4000 and 4000", run(in(4000, 4000)), verdict.Detected, 365, "")
	check(t, "offset 2999", run(in(2999, 4000)), verdict.None, 0, "")
	check(t, "offset 3000", run(in(3000, 4000)), verdict.Detected, 365, "")
	check(t, "offset -3000", run(in(-3000, 4000)), verdict.Detected, 365, "")
	check(t, "unmet 2999", run(in(4000, 2999)), verdict.None, 0, "")
	check(t, "unmet 3000", run(in(4000, 3000)), verdict.Detected, 365, "")
	apart := in(4000, 0)
	apart.Unmet[2].Value = 9000 // unmet in window 2, offset in window 1
	check(t, "the two in different windows", run(apart), verdict.None, 0, "")
	short := in(4000, 4000)
	short.Offsets[0].Points = short.Offsets[0].Points[:1]
	check(t, "a market with no point in the window", run(short), verdict.None, 0, "")
}

// TestDetect_R9_RegimeAndHealthy verifies R9.
//
// detectors.out lines 4 to 7: deflation of 40% a year with volume falling is
// detected at "onset window 2" (tick 730); with flat volume none (a falling
// price with growing real volume is the pressure of MON 8.4); the healthy series
// is none. Boundary: a link of 7 001 does not fall by 3 000 bp, 7 000 does.
//
// Requirement: FR-STY-069
func TestDetect_R9_RegimeAndHealthy(t *testing.T) {
	t.Parallel()
	th := priors()
	deflating := linksOf(priceSeries(4, 2500, -4000, 200, 1))
	healthy := linksOf(priceSeries(4, 2500, 0, 200, 1))
	fall, flat := volumeOf(volumeDecaying(4, 6000)), volumeOf(volumeFlat(4))
	run := func(in verdict.Input) verdict.Verdict { return verdict.Detect(verdict.R9, in, th) }
	check(t, "detectors.out:6 deflation, volume falling", run(verdict.Input{Links: []verdict.Series{deflating}, Volume: fall}), verdict.Detected, winTick(2), "")
	check(t, "detectors.out:7 deflation, volume flat", run(verdict.Input{Links: []verdict.Series{deflating}, Volume: flat}), verdict.None, 0, "")
	check(t, "detectors.out:4 healthy", run(verdict.Input{Links: []verdict.Series{healthy}, Volume: flat}), verdict.None, 0, "")

	link := func(v int64) verdict.Series {
		return verdict.Series{Points: []verdict.Point{{Tick: 365, Value: v}, {Tick: 730, Value: v}}}
	}
	for _, tc := range []struct {
		name  string
		v     int64
		class verdict.Class
	}{{"7001", 7001, verdict.None}, {"7000", 7000, verdict.Detected}, {"6999", 6999, verdict.Detected}} {
		check(t, "link "+tc.name, run(verdict.Input{Links: []verdict.Series{link(tc.v)}, Volume: fall[:3]}), tc.class, 730, "")
	}
}

// TestDetect_UnknownReasons verifies every unknown branch: fewer windows than
// min_windows, stale (R1, R9), low activity (R6, R8), an unset key, a missing
// reference (R8), a one-tick series shorter than its count key, clamped and no
// outcomes; that the volume detectors are not made unknown by low activity; and
// that Stale turns a none into unknown but never a detected into unknown.
//
// Requirement: FR-STY-060, FR-STY-007
func TestDetect_UnknownReasons(t *testing.T) {
	t.Parallel()
	th := priors()
	fall, flat := volumeOf(volumeDecaying(4, 6000)), volumeOf(volumeFlat(4))
	healthyLinks := linksOf(priceSeries(4, 2500, 0, 200, 1))
	infLinks := linksOf(priceSeries(4, 2500, 4000, 200, 1))
	defLinks := linksOf(priceSeries(4, 2500, -4000, 200, 1))
	two := verdict.Series{Points: healthyLinks.Points[:1]} // 2 windows
	rangesOK := verdict.Series{Points: rangesOf(priceSeries(4, 2500, 0, 200, 1)).Points}
	run := func(d verdict.Detector, in verdict.Input) verdict.Verdict { return verdict.Detect(d, in, th) }

	for _, tc := range []struct {
		name   string
		d      verdict.Detector
		in     verdict.Input
		th     verdict.Thresholds
		reason string
	}{
		{"R1 two windows", verdict.R1, verdict.Input{Links: []verdict.Series{two}}, th, verdict.ReasonWindows},
		{"R1 no chain", verdict.R1, verdict.Input{}, th, verdict.ReasonWindows},
		{"R1 stale", verdict.R1, verdict.Input{Links: []verdict.Series{healthyLinks}, Stale: 1001}, th, verdict.ReasonStale},
		{"R1 stale at the bound is fine", verdict.R1, verdict.Input{Links: []verdict.Series{healthyLinks}, Stale: 1000}, th, ""},
		{"R1 key unset", verdict.R1, verdict.Input{Links: []verdict.Series{healthyLinks}}, without(th, "r1.rise_bp"), verdict.ReasonThresholdUnset},
		{"R1 shared key unset", verdict.R1, verdict.Input{Links: []verdict.Series{healthyLinks}}, without(th, "min_windows"), verdict.ReasonThresholdUnset},
		{"R2 two windows", verdict.R2, verdict.Input{Volume: fall[:2]}, th, verdict.ReasonWindows},
		{"R2 no volume", verdict.R2, verdict.Input{}, with(th, "min_windows", 0), verdict.ReasonWindows},
		{"R2 key unset", verdict.R2, verdict.Input{Volume: fall}, without(th, "r2.top_bp"), verdict.ReasonThresholdUnset},
		{"R3 nothing to judge", verdict.R3, verdict.Input{}, th, verdict.ReasonWindows},
		{"R3 key unset", verdict.R3, verdict.Input{Roles: [][]verdict.Point{{{Tick: 0, Value: 1}}}}, without(th, "r3.role_ticks"), verdict.ReasonThresholdUnset},
		{"R4 two windows", verdict.R4, verdict.Input{Volume: fall[:2]}, th, verdict.ReasonWindows},
		{"R4 no volume", verdict.R4, verdict.Input{}, with(th, "min_windows", 0), verdict.ReasonWindows},
		{"R4 key unset", verdict.R4, verdict.Input{Volume: fall}, without(th, "r3.fall_bp"), verdict.ReasonThresholdUnset},
		{"R5a short series", verdict.R5a, verdict.Input{Pinned: [][]verdict.Point{ticksOf(make([]int64, 59))}, Outcomes: 1}, th, verdict.ReasonWindows},
		{"R5a no series", verdict.R5a, verdict.Input{Outcomes: 1}, th, verdict.ReasonWindows},
		{"R5a no outcomes", verdict.R5a, verdict.Input{Pinned: [][]verdict.Point{ticksOf(make([]int64, 90))}}, th, verdict.ReasonOutcomes},
		{"R5a key unset", verdict.R5a, verdict.Input{Pinned: [][]verdict.Point{ticksOf(make([]int64, 90))}, Outcomes: 1}, without(th, "r5.pinned_bp"), verdict.ReasonThresholdUnset},
		{"R5b short series", verdict.R5b, verdict.Input{Frozen: [][]verdict.Point{ticksOf(make([]int64, 10))}}, th, verdict.ReasonWindows},
		{"R5b no series", verdict.R5b, verdict.Input{}, th, verdict.ReasonWindows},
		{"R5b key unset", verdict.R5b, verdict.Input{Frozen: [][]verdict.Point{ticksOf(make([]int64, 90))}}, without(th, "r5.freeze_ticks"), verdict.ReasonThresholdUnset},
		{"R6 low activity", verdict.R6, verdict.Input{Ranges: []verdict.Series{rangesOK}, Active: 4999}, th, verdict.ReasonActive},
		{"R6 activity at the bound is fine", verdict.R6, verdict.Input{Ranges: []verdict.Series{rangesOK}, Active: 5000}, th, ""},
		{"R6 two windows", verdict.R6, verdict.Input{Ranges: []verdict.Series{{Points: rangesOK.Points[:2]}}, Active: 10000}, th, verdict.ReasonWindows},
		{"R6 nothing", verdict.R6, verdict.Input{Active: 10000}, th, verdict.ReasonWindows},
		{"R6 clamped", verdict.R6, verdict.Input{Ranges: []verdict.Series{{Points: rangesOK.Points, Clamped: []bool{false, true}}}, Active: 10000}, th, verdict.ReasonClamped},
		{"R6 key unset", verdict.R6, verdict.Input{Ranges: []verdict.Series{rangesOK}, Active: 10000}, without(th, "active_min_bp"), verdict.ReasonThresholdUnset},
		{"R7 nothing", verdict.R7, verdict.Input{}, th, verdict.ReasonWindows},
		{"R7 key unset", verdict.R7, verdict.Input{Dispersion: [][]verdict.Point{{{Tick: 0, Value: 5}}}, Outcomes: 1}, without(th, "r7.disp_bp"), verdict.ReasonThresholdUnset},
		{"R8 no reference", verdict.R8, verdict.Input{Active: 10000}, th, verdict.ReasonReferenceMissing},
		{"R8 low activity", verdict.R8, verdict.Input{HasReference: true, Active: 100}, th, verdict.ReasonActive},
		{"R8 two windows", verdict.R8, verdict.Input{HasReference: true, Active: 10000, Unmet: make([]verdict.Point, 2), Offsets: []verdict.Series{{}}}, th, verdict.ReasonWindows},
		{"R8 no markets", verdict.R8, verdict.Input{HasReference: true, Active: 10000, Unmet: make([]verdict.Point, 3)}, th, verdict.ReasonWindows},
		{"R8 clamped", verdict.R8, verdict.Input{HasReference: true, Active: 10000, Unmet: make([]verdict.Point, 3), Offsets: []verdict.Series{{Points: make([]verdict.Point, 3), Clamped: []bool{true}}}}, th, verdict.ReasonClamped},
		{"R8 key unset", verdict.R8, verdict.Input{}, without(th, "r8.unmet_bp"), verdict.ReasonThresholdUnset},
		{"R9 two windows", verdict.R9, verdict.Input{Links: []verdict.Series{two}, Volume: fall}, th, verdict.ReasonWindows},
		{"R9 no chain", verdict.R9, verdict.Input{Volume: fall}, th, verdict.ReasonWindows},
		{"R9 volume does not cover the links", verdict.R9, verdict.Input{Links: []verdict.Series{healthyLinks}, Volume: fall[:3]}, th, verdict.ReasonWindows},
		{"R9 stale", verdict.R9, verdict.Input{Links: []verdict.Series{healthyLinks}, Volume: flat, Stale: 2500}, th, verdict.ReasonStale},
		{"R9 clamped", verdict.R9, verdict.Input{Links: []verdict.Series{{Points: healthyLinks.Points, Clamped: []bool{true}}}, Volume: flat}, th, verdict.ReasonClamped},
		{"R9 key unset", verdict.R9, verdict.Input{Links: []verdict.Series{healthyLinks}, Volume: flat}, without(th, "stale_max_bp"), verdict.ReasonThresholdUnset},
	} {
		got := verdict.Detect(tc.d, tc.in, tc.th)
		class := verdict.Unknown
		if tc.reason == "" {
			class = verdict.None
		}
		check(t, tc.name, got, class, 0, tc.reason)
		if got.Detector != tc.d {
			t.Errorf("%s: Detector = %q, want %q", tc.name, got.Detector, tc.d)
		}
	}

	// Stale turns none into unknown but never a detected into unknown.
	check(t, "R1 inflation with stale prices", run(verdict.R1, verdict.Input{Links: []verdict.Series{infLinks}, Stale: 9000}), verdict.Detected, winTick(2), "")
	check(t, "R9 deflation with stale prices", run(verdict.R9, verdict.Input{Links: []verdict.Series{defLinks}, Volume: fall, Stale: 9000}), verdict.Detected, winTick(2), "")
	// The volume detectors are not made unknown by low activity.
	for _, d := range []verdict.Detector{verdict.R2, verdict.R4, verdict.R5b, verdict.R9} {
		in := verdict.Input{Volume: flat, Links: []verdict.Series{healthyLinks}, Households: [][]verdict.Point{{{Value: 600}, {Value: 600}}}, Frozen: [][]verdict.Point{ticksOf(make([]int64, 90))}, Active: 0}
		if got := run(d, in); got.Class != verdict.None {
			t.Errorf("%s with Active 0 = %s, want none: a fall of activity is its subject", d, got)
		}
	}
	// A clamped value is detected by R1 and never unknown.
	check(t, "R1 clamped", run(verdict.R1, verdict.Input{Links: []verdict.Series{{Points: healthyLinks.Points, Clamped: []bool{false, false, true}}}}), verdict.Detected, winTick(3), "")
}

// TestDetect_R3_SpeedLimit verifies the horizon of R3: r3.horizon_factor times
// t_min, at least r3.horizon, and that a run too short is unknown.
//
// 200 households falling from 600: t_min to 540 at 20 000 bp a year is 183
// ticks (detectors.out line 11 prints horizon 915 = 5 * 183). A run that ends
// 914 ticks after the minimum is unknown and 915 detects. A slower rate
// (4 000 bp a year, t_min 911) pushes the horizon to 4 555.
//
// Requirement: FR-STY-063
func TestDetect_R3_SpeedLimit(t *testing.T) {
	t.Parallel()
	th := priors()
	tm, ok := verdict.RegrowTicks(200, 600, 1000, 20000, tpy)
	if !ok || tm != 183 {
		t.Fatalf("RegrowTicks(200, 600, 1000, 20000, 365) = %d, %v, want 183, true (detectors.out line 11: horizon 915 = 5 * 183)", tm, ok)
	}
	in := func(end int64, rate int64) verdict.Input {
		return verdict.Input{Households: [][]verdict.Point{{{Tick: 0, Value: 600}, {Tick: 100, Value: 200}, {Tick: end, Value: 200}}}, Rate: rate, TicksPerYear: tpy}
	}
	run := func(in verdict.Input) verdict.Verdict { return verdict.Detect(verdict.R3, in, th) }
	check(t, "914 ticks after the minimum", run(in(100+914, 20000)), verdict.Unknown, 0, verdict.ReasonNeedsLongerRun)
	check(t, "915 ticks after the minimum", run(in(100+915, 20000)), verdict.Detected, 100, "")
	tm2, _ := verdict.RegrowTicks(200, 600, 1000, 4000, tpy)
	horizon := tm2 * 5
	check(t, "slow rate, one short of the horizon", run(in(100+horizon-1, 4000)), verdict.Unknown, 0, verdict.ReasonNeedsLongerRun)
	check(t, "slow rate, at the horizon", run(in(100+horizon, 4000)), verdict.Detected, 100, "")
	check(t, "a factor of 0 leaves the floor", verdict.Detect(verdict.R3, in(100+730, 4000), with(th, "r3.horizon_factor", 0)), verdict.Detected, 100, "")
}

// TestDetect_CostIsLinear verifies that the allocations of a detector are the
// same for a short and a long series of the same shape, so they do not grow with
// the length, and that the number of agents of a run, which no input carries, can
// not change them.
//
// Requirement: NFR-STY-001
func TestDetect_CostIsLinear(t *testing.T) {
	th := priors()
	build := func(years int) verdict.Input {
		p := priceSeries(years, 2500, 0, 200, 1)
		l := linksOf(p)
		hh := hh600(years, true)
		return verdict.Input{
			Links: []verdict.Series{l}, Volume: volumeOf(volumeFlat(years)), Stale: 0, Active: 10000,
			Households: [][]verdict.Point{hh}, Rate: 20000, TicksPerYear: tpy,
			Pinned: [][]verdict.Point{ticksOf(make([]int64, years*tpy))}, Frozen: [][]verdict.Point{ticksOf(make([]int64, years*tpy))}, Outcomes: 5,
			Ranges: []verdict.Series{rangesOf(p)}, Dispersion: [][]verdict.Point{ticksOf(volumeFlat(years))},
			Offsets: []verdict.Series{rangesOf(p)}, Unmet: ticksOf(make([]int64, years)), HasReference: true,
		}
	}
	small, large := build(4), build(40)
	for _, d := range verdict.Detectors() {
		a := testing.AllocsPerRun(20, func() { verdict.Detect(d, small, th) })
		b := testing.AllocsPerRun(20, func() { verdict.Detect(d, large, th) })
		if a != b {
			t.Errorf("%s allocates %v on 4 years and %v on 40: it must not grow with the series", d, a, b)
		}
	}
}

// FuzzDetectors verifies that no detector panics on any series, that an onset
// is the tick of a point of the input, and that the class of a verdict does not
// depend on the order of the series of a list (equal values in any order give
// one verdict).
//
// Requirement: NFR-STY-003
func FuzzDetectors(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8}, []byte{9, 9, 9, 9}, int64(3000))
	f.Add([]byte{}, []byte{1}, int64(0))
	f.Add([]byte{255, 0, 255, 0, 255, 0, 255}, []byte{0, 0, 0, 0, 0, 0}, int64(-5))
	f.Fuzz(func(t *testing.T, a, b []byte, k int64) {
		pts := func(raw []byte, step int64) []verdict.Point {
			out := make([]verdict.Point, len(raw))
			for i, x := range raw {
				out[i] = verdict.Point{Tick: int64(i) * step, Value: int64(x) * (k%97 + 100)}
			}
			return out
		}
		pa, pb := pts(a, 1), pts(b, 365)
		in := verdict.Input{
			Links: []verdict.Series{{Points: pb}, {Points: pts(a, 365)}}, M: pa, MInit: k, Stale: k % 2000,
			Volume: pb, MoneyOut: pa, Unsold: [][]verdict.Point{pa, pb}, Top: pb,
			Households: [][]verdict.Point{pa, pb}, Roles: [][]verdict.Point{pb, pa}, Rate: k, TicksPerYear: k%400 + 1,
			Pinned: [][]verdict.Point{pa, pb}, Frozen: [][]verdict.Point{pb, pa}, Outcomes: k,
			Ranges: []verdict.Series{{Points: pb}, {Points: pa}}, Swings: [][]verdict.Point{pa, pb}, Active: 10000,
			Dispersion: [][]verdict.Point{pb, pa}, Completions: [][]verdict.Point{pa, pb},
			Offsets: []verdict.Series{{Points: pa}, {Points: pb}}, Unmet: pb, HasReference: true,
		}
		rev := in
		rev.Links = []verdict.Series{in.Links[1], in.Links[0]}
		rev.Households = [][]verdict.Point{pb, pa}
		rev.Roles = [][]verdict.Point{pa, pb}
		rev.Unsold = [][]verdict.Point{pb, pa}
		rev.Pinned = [][]verdict.Point{pb, pa}
		rev.Frozen = [][]verdict.Point{pa, pb}
		rev.Ranges = []verdict.Series{in.Ranges[1], in.Ranges[0]}
		rev.Swings = [][]verdict.Point{pb, pa}
		rev.Dispersion = [][]verdict.Point{pa, pb}
		rev.Completions = [][]verdict.Point{pb, pa}
		rev.Offsets = []verdict.Series{in.Offsets[1], in.Offsets[0]}
		th := priors()
		th.Set("r3.horizon", k%5000)
		th.Set("r1.windows", k%4)
		for _, d := range verdict.Detectors() {
			v := verdict.Detect(d, in, th)
			if v.Class == verdict.Detected && !tickOf(in, v.Onset) {
				t.Fatalf("%s onset %d is not the tick of a point of the input", d, v.Onset)
			}
			r := rev
			if d == verdict.R9 {
				r.Links = in.Links // R9 reads the first chain, the whole economy: its place is fixed
			}
			if w := verdict.Detect(d, r, th); w.Class != v.Class {
				t.Fatalf("%s class %s on the series in one order and %s in the other", d, v.Class, w.Class)
			}
		}
	})
}

// tickOf reports whether some point of the input has the tick.
func tickOf(in verdict.Input, tick int64) bool {
	has := func(p []verdict.Point) bool {
		for _, x := range p {
			if x.Tick == tick {
				return true
			}
		}
		return false
	}
	lists := [][]verdict.Point{in.M, in.Volume, in.MoneyOut, in.Top, in.Unmet}
	lists = append(lists, in.Unsold...)
	lists = append(lists, in.Households...)
	lists = append(lists, in.Roles...)
	lists = append(lists, in.Pinned...)
	lists = append(lists, in.Frozen...)
	lists = append(lists, in.Swings...)
	lists = append(lists, in.Dispersion...)
	lists = append(lists, in.Completions...)
	for _, s := range in.Links {
		lists = append(lists, s.Points)
	}
	for _, s := range in.Ranges {
		lists = append(lists, s.Points)
	}
	for _, s := range in.Offsets {
		lists = append(lists, s.Points)
	}
	for _, l := range lists {
		if has(l) {
			return true
		}
	}
	return false
}
