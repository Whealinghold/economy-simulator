package verdict_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// detectedWith checks the Value and Threshold of a detected verdict.
func detectedWith(t *testing.T, name string, got verdict.Verdict, onset, value, threshold int64) {
	t.Helper()
	if got.Class != verdict.Detected || got.Onset != onset || got.Value != value || got.Threshold != threshold {
		t.Errorf("%s: got %s, want detected onset %d value %d threshold %d", name, got, onset, value, threshold)
	}
}

// TestDetect_EdgeKeys verifies the corners of the keys that the regime rows do
// not reach: a run length of 1 (and of 0, read as 1), keys with distinct
// values so that a swap between them is seen, equal and rising comparisons,
// and the earliest of two onsets.
//
// Guard: a boundary mutation of a detector (an index off by one, a comparison
// loosened) must change a verdict here; the priors repeat values (5 000 for
// both R2 shares, 3 000 for both R8 keys), which hides a swapped key.
func TestDetect_EdgeKeys(t *testing.T) {
	t.Parallel()
	th := priors()
	pts := func(vals ...int64) []verdict.Point {
		out := make([]verdict.Point, len(vals))
		for i, v := range vals {
			out[i] = verdict.Point{Tick: winTick(i), Value: v}
		}
		return out
	}
	links := func(vals ...int64) verdict.Series {
		p := pts(vals...)
		for i := range p {
			p[i].Tick = winTick(i + 1)
		}
		return verdict.Series{Points: p}
	}

	t.Run("R1", func(t *testing.T) {
		one := with(th, "r1.windows", 1)
		// the first of several qualifying links is the onset (windows 1).
		detectedWith(t, "windows 1 at the first link", verdict.Detect(verdict.R1, verdict.Input{Links: []verdict.Series{links(13000, 14000, 15000)}}, one), winTick(1), 3000, 3000)
		zero := with(th, "r1.windows", 0)
		check(t, "windows 0 reads as 1", verdict.Detect(verdict.R1, verdict.Input{Links: []verdict.Series{links(10000, 13000, 10000)}}, zero), verdict.Detected, winTick(2), "")
		// a clamped link after the run is not the onset.
		c := links(13000, 13000, 13000)
		c.Clamped = []bool{false, false, true}
		check(t, "clamped after the run", verdict.Detect(verdict.R1, verdict.Input{Links: []verdict.Series{c}}, th), verdict.Detected, winTick(2), "")
		// money: strictly above the factor, a negative start ignored, and the
		// stale check reached when money is present but quiet.
		mo := func(minit, a, b int64) verdict.Input {
			return verdict.Input{Links: []verdict.Series{links(10000, 10000)}, MInit: minit, M: pts(a, b), Stale: 5000}
		}
		check(t, "money above twice", verdict.Detect(verdict.R1, mo(100, 301, 400), th), verdict.Detected, winTick(1), "")
		check(t, "negative start", verdict.Detect(verdict.R1, mo(-100, 301, 400), th), verdict.Unknown, 0, verdict.ReasonStale)
		check(t, "quiet money, stale", verdict.Detect(verdict.R1, mo(100, 100, 100), th), verdict.Unknown, 0, verdict.ReasonStale)
	})

	t.Run("R9", func(t *testing.T) {
		fall := volumeOf(volumeDecaying(4, 6000))
		rising := volumeOf(volumeDecaying(4, 14000))
		def := linksOf(priceSeries(4, 2500, -4000, 200, 1))
		healthy := linksOf(priceSeries(4, 2500, 0, 200, 1))
		check(t, "deflation with rising volume", verdict.Detect(verdict.R9, verdict.Input{Links: []verdict.Series{def}, Volume: rising}, th), verdict.None, 0, "")
		extra := append(append([]verdict.Point(nil), fall...), verdict.Point{Tick: winTick(4), Value: 1})
		check(t, "more volume windows than links", verdict.Detect(verdict.R9, verdict.Input{Links: []verdict.Series{def}, Volume: extra}, th), verdict.Detected, winTick(2), "")
		check(t, "stale at the bound", verdict.Detect(verdict.R9, verdict.Input{Links: []verdict.Series{healthy}, Volume: fall[:4], Stale: 1000}, th), verdict.None, 0, "")
		check(t, "stale one above", verdict.Detect(verdict.R9, verdict.Input{Links: []verdict.Series{healthy}, Volume: fall[:4], Stale: 1001}, th), verdict.Unknown, 0, verdict.ReasonStale)
		one := with(th, "r9.windows", 1)
		check(t, "windows 1", verdict.Detect(verdict.R9, verdict.Input{Links: []verdict.Series{links(6000, 10000, 10000)}, Volume: pts(100, 50, 50, 50)}, one), verdict.Detected, winTick(1), "")
	})

	t.Run("R2", func(t *testing.T) {
		d := th
		d.Set("r2.share_bp", 4000)
		d.Set("r2.unsold_bp", 6000)
		d.Set("r2.top_bp", 8000)
		d.Set("r2.windows", 1)
		vol := pts(10000, 9000, 100)
		check(t, "share 4000 fires at 4000", verdict.Detect(verdict.R2, verdict.Input{Volume: vol, MoneyOut: pts(4000)}, d), verdict.Detected, 0, "")
		check(t, "share does not read the unsold key", verdict.Detect(verdict.R2, verdict.Input{Volume: vol, MoneyOut: pts(3999), Unsold: [][]verdict.Point{pts(1, 5999)}}, d), verdict.None, 0, "")
		check(t, "unsold at 6000 rising", verdict.Detect(verdict.R2, verdict.Input{Volume: vol, Unsold: [][]verdict.Point{pts(1, 6000)}}, d), verdict.Detected, winTick(1), "")
		check(t, "unsold above but falling", verdict.Detect(verdict.R2, verdict.Input{Volume: vol, Unsold: [][]verdict.Point{pts(9000, 8000)}}, d), verdict.None, 0, "")
		check(t, "top 8000", verdict.Detect(verdict.R2, verdict.Input{Volume: vol, Top: pts(8000)}, d), verdict.Detected, 0, "")
		check(t, "top 8001", verdict.Detect(verdict.R2, verdict.Input{Volume: vol, Top: pts(8001)}, d), verdict.Detected, 0, "")
		check(t, "top 7999", verdict.Detect(verdict.R2, verdict.Input{Volume: vol, Top: pts(7999)}, d), verdict.None, 0, "")
		// the first window decides the fall, not the second.
		check(t, "first window, not the second", verdict.Detect(verdict.R2, verdict.Input{Volume: pts(10000, 100, 8500), MoneyOut: pts(9000)}, d), verdict.None, 0, "")
		// an unsold series that does not fire does not hide the top share.
		check(t, "unsold quiet, top fires", verdict.Detect(verdict.R2, verdict.Input{Volume: vol, Unsold: [][]verdict.Point{pts(1, 2)}, Top: pts(9000)}, d), verdict.Detected, 0, "")
		check(t, "unsold quiet, share fires", verdict.Detect(verdict.R2, verdict.Input{Volume: vol, MoneyOut: pts(9000), Unsold: [][]verdict.Point{pts(1, 2)}}, d), verdict.Detected, 0, "")
	})

	t.Run("R3", func(t *testing.T) {
		in := func(hs ...[]verdict.Point) verdict.Input {
			return verdict.Input{Households: hs, Rate: 20000, TicksPerYear: tpy}
		}
		long := hh600(6, false)
		single := []verdict.Point{{Tick: 0, Value: 5}}
		// a detected series wins whichever side an unknown stands on.
		check(t, "unknown before detected", verdict.Detect(verdict.R3, in(single, long), th), verdict.Detected, 365, "")
		check(t, "detected before unknown", verdict.Detect(verdict.R3, in(long, single), th), verdict.Detected, 365, "")
		short := hh600(3, false)
		check(t, "the first unknown is kept", verdict.Detect(verdict.R3, in(single, short), th), verdict.Unknown, 0, verdict.ReasonWindows)
		check(t, "the first unknown is kept (other order)", verdict.Detect(verdict.R3, in(short, single), th), verdict.Unknown, 0, verdict.ReasonNeedsLongerRun)
		check(t, "none then unknown", verdict.Detect(verdict.R3, in(hh600(3, true), short), th), verdict.Unknown, 0, verdict.ReasonNeedsLongerRun)
		// value: 200 of 600 left is 3 333 bp, a fall of 6 667.
		detectedWith(t, "fall measured", verdict.Detect(verdict.R3, in(long), th), 365, 6667, 2000)
		// back above the target after the minimum is a recovery.
		rec := []verdict.Point{{Tick: 0, Value: 1000}, {Tick: 10, Value: 500}, {Tick: 20, Value: 901}, {Tick: 3000, Value: 500}}
		check(t, "recovered above the target", verdict.Detect(verdict.R3, in(rec), th), verdict.None, 0, "")
		// a factor that overflows leaves a horizon that no run reaches.
		huge := with(th, "r3.horizon_factor", 1<<62)
		check(t, "overflowing horizon", verdict.Detect(verdict.R3, in(long), huge), verdict.Unknown, 0, verdict.ReasonNeedsLongerRun)
		// roles: empty and never-started series are skipped, a start of 0 is not a start.
		role := func(first int64, zeros int) []verdict.Point {
			v := []int64{first}
			for i := 0; i < zeros; i++ {
				v = append(v, 0)
			}
			return ticksOf(v)
		}
		ri := func(rs ...[]verdict.Point) verdict.Input { return verdict.Input{Roles: rs} }
		check(t, "empty role series", verdict.Detect(verdict.R3, ri(nil), th), verdict.None, 0, "")
		check(t, "never started, long run of zeros", verdict.Detect(verdict.R3, ri(role(0, 400)), th), verdict.None, 0, "")
		detectedWith(t, "role", verdict.Detect(verdict.R3, ri(role(1, 400)), th), 365, 0, 365)
		// a region with a start of 0 is not a collapse for R3 or for R4.
		zero := []verdict.Point{{Tick: 0, Value: 0}, {Tick: 10, Value: 0}, {Tick: 3000, Value: 0}}
		check(t, "start of 0", verdict.Detect(verdict.R3, in(zero), th), verdict.None, 0, "")
	})

	t.Run("R4", func(t *testing.T) {
		vol := pts(1000, 400, 300)
		hh := func(hs ...[]verdict.Point) verdict.Input { return verdict.Input{Volume: vol, Households: hs} }
		zero := []verdict.Point{{Tick: 0, Value: 0}, {Tick: 10, Value: 0}}
		detectedWith(t, "volume share", verdict.Detect(verdict.R4, hh(zero), th), winTick(2), 3000, 5000)
		check(t, "a series with no start is no collapse", verdict.Detect(verdict.R4, hh(zero), th), verdict.Detected, winTick(2), "")
		check(t, "an empty households series", verdict.Detect(verdict.R4, hh(nil), th), verdict.Detected, winTick(2), "")
		check(t, "a collapse in a second series", verdict.Detect(verdict.R4, hh(zero, []verdict.Point{{Value: 10}, {Value: 5}}), th), verdict.None, 0, "")
		// the first window is the reference.
		check(t, "first window, not the second", verdict.Detect(verdict.R4, verdict.Input{Volume: pts(1000, 100, 100), Households: [][]verdict.Point{{{Value: 5}, {Value: 5}}}}, with(th, "r4.windows", 1)), verdict.Detected, winTick(1), "")
	})

	t.Run("R5", func(t *testing.T) {
		in := verdict.Input{Pinned: [][]verdict.Point{ticksOf(make([]int64, 70))}, Outcomes: -1}
		check(t, "negative outcomes", verdict.Detect(verdict.R5a, in, th), verdict.Unknown, 0, verdict.ReasonOutcomes)
		exact := make([]int64, 60)
		check(t, "a series of exactly the count is not short", verdict.Detect(verdict.R5b, verdict.Input{Frozen: [][]verdict.Point{ticksOf(exact)}}, th), verdict.None, 0, "")
		one := with(th, "r5.freeze_ticks", 1)
		detectedWith(t, "count of 1 at the first tick", verdict.Detect(verdict.R5b, verdict.Input{Frozen: [][]verdict.Point{ticksOf([]int64{1, 1, 1})}}, one), 0, 3, 1)
	})

	t.Run("R6", func(t *testing.T) {
		grow := func(vals ...int64) verdict.Input {
			s := verdict.Series{Points: pts(vals...)}
			return verdict.Input{Ranges: []verdict.Series{s}, Active: 10000}
		}
		// growing but below the factor, then different from it: not a blow-up.
		check(t, "growing below the factor", verdict.Detect(verdict.R6, grow(5000, 10000, 20000), th), verdict.None, 0, "")
		check(t, "above the factor but shrinking", verdict.Detect(verdict.R6, grow(90000, 80000, 70000), th), verdict.None, 0, "")
	})

	t.Run("R7", func(t *testing.T) {
		ones := make([]int64, 60)
		for i := range ones {
			ones[i] = 1
		}
		spike := append([]int64(nil), ones...)
		spike[9] = 9
		late := append(append([]int64(nil), ones...), 99)
		quiet := [][]verdict.Point{{{Tick: 0, Value: 900}}}
		run := func(in verdict.Input) verdict.Verdict { return verdict.Detect(verdict.R7, in, th) }
		check(t, "dispersion quiet, spike fires", run(verdict.Input{Dispersion: quiet, Outcomes: 1, Completions: [][]verdict.Point{ticksOf(spike)}}), verdict.Detected, 9, "")
		check(t, "a tick after first_ticks is ignored", run(verdict.Input{Completions: [][]verdict.Point{ticksOf(late)}}), verdict.None, 0, "")
		check(t, "dispersion quiet, completions short", run(verdict.Input{Dispersion: quiet, Outcomes: 1, Completions: [][]verdict.Point{ticksOf([]int64{1})}}), verdict.Unknown, 0, verdict.ReasonWindows)
		check(t, "no outcomes, spike still fires", run(verdict.Input{Dispersion: quiet, Completions: [][]verdict.Point{ticksOf(spike)}}), verdict.Detected, 9, "")
		check(t, "negative outcomes", run(verdict.Input{Dispersion: quiet, Outcomes: -1}), verdict.Unknown, 0, verdict.ReasonOutcomes)
		check(t, "the first unknown is kept", run(verdict.Input{Dispersion: quiet, Completions: [][]verdict.Point{ticksOf([]int64{1}), ticksOf(ones[:2])}}), verdict.Unknown, 0, verdict.ReasonOutcomes)
		check(t, "a short series then a quiet one", run(verdict.Input{Completions: [][]verdict.Point{ticksOf([]int64{1}), ticksOf(ones)}}), verdict.Unknown, 0, verdict.ReasonWindows)
		check(t, "a quiet series then a short one", run(verdict.Input{Completions: [][]verdict.Point{ticksOf(ones), ticksOf([]int64{1})}}), verdict.Unknown, 0, verdict.ReasonWindows)
		// exactly twice the mean is not above it: one 2, fifty-eight 1 and one 0 (sum 60, mean 1).
		eq := append([]int64{2}, ones[:58]...)
		eq = append(eq, 0)
		check(t, "exactly twice the mean", run(verdict.Input{Completions: [][]verdict.Point{ticksOf(eq)}}), verdict.None, 0, "")
	})

	t.Run("R8", func(t *testing.T) {
		d := th
		d.Set("r8.offset_bp", 2000)
		d.Set("r8.unmet_bp", 4000)
		off := func(unmet, offset []int64) verdict.Input {
			return verdict.Input{
				Offsets: []verdict.Series{{Points: ticksOf(offset)}},
				Unmet:   ticksOf(unmet), Active: 5000, HasReference: true,
			}
		}
		check(t, "keys are not swapped", verdict.Detect(verdict.R8, off([]int64{3000, 3000, 3000}, []int64{3500, 3500, 3500}), d), verdict.None, 0, "")
		check(t, "unmet 4000 offset 2000", verdict.Detect(verdict.R8, off([]int64{0, 0, 4000}, []int64{0, 0, 2000}), d), verdict.Detected, 2, "")
		check(t, "more windows than the minimum", verdict.Detect(verdict.R8, off([]int64{0, 0, 0, 0, 4000}, []int64{0, 0, 0, 0, 2500}), d), verdict.Detected, 4, "")
		// a market that stops early does not stop the loop or the next market.
		in := off([]int64{0, 0, 0, 4000, 4000}, []int64{1})
		in.Offsets = append(in.Offsets, verdict.Series{Points: ticksOf([]int64{0, 0, 0, 0, 2600})})
		check(t, "short market, then another", verdict.Detect(verdict.R8, in, d), verdict.Detected, 4, "")
		// a negative threshold is always met.
		neg := with(th, "r8.offset_bp", -1)
		check(t, "negative offset key", verdict.Detect(verdict.R8, off([]int64{0, 0, 9000}, []int64{0, 0, 0}), neg), verdict.Detected, 2, "")
		check(t, "activity at the bound", verdict.Detect(verdict.R8, off([]int64{0, 0, 0}, []int64{0, 0, 0}), th), verdict.None, 0, "")
	})
}

// TestScanRuns_WindowsKeyOfZero verifies through R2 that a run key of 0 is read
// as 1, not as "always".
func TestScanRuns_WindowsKeyOfZero(t *testing.T) {
	t.Parallel()
	th := with(priors(), "r2.windows", 0)
	in := verdict.Input{
		Volume:   []verdict.Point{{Tick: 0, Value: 1000}, {Tick: 365, Value: 100}, {Tick: 730, Value: 100}},
		MoneyOut: []verdict.Point{{Tick: 0, Value: 0}, {Tick: 365, Value: 9000}},
	}
	check(t, "windows 0 reads as 1", verdict.Detect(verdict.R2, in, th), verdict.Detected, 365, "")
	in.MoneyOut = ticksOf([]int64{0, 0})
	check(t, "windows 0 is not always", verdict.Detect(verdict.R2, in, th), verdict.None, 0, "")
}

// TestDetect_EdgeCases2 verifies the remaining corners found by mutation
// testing: an overflowing R3 horizon from a long speed limit, a quiet role
// beside a collapse, a start of 1, a frozen flag of -1, a first_ticks of 1 and
// an offset key of 1.
//
// Guard: each case changes a verdict if the guard it exercises is dropped.
func TestDetect_EdgeCases2(t *testing.T) {
	t.Parallel()
	th := priors()
	// 200 -> 540 at 100 bp a year takes about 36 000 ticks: times the largest
	// factor the product does not fit, and the horizon is the largest int64.
	long := []verdict.Point{{Tick: 0, Value: 600}, {Tick: 100, Value: 200}, {Tick: 1 << 40, Value: 200}}
	in := verdict.Input{Households: [][]verdict.Point{long}, Rate: 100, TicksPerYear: tpy}
	check(t, "overflowing horizon is never reached", verdict.Detect(verdict.R3, in, with(th, "r3.horizon_factor", 1<<63-1)), verdict.Unknown, 0, verdict.ReasonNeedsLongerRun)
	check(t, "the same run with a factor of 5", verdict.Detect(verdict.R3, in, th), verdict.Detected, 100, "")

	// a quiet role does not hide a collapse (the role is judged first).
	quiet := verdict.Input{Roles: [][]verdict.Point{ticksOf([]int64{3, 3, 3})}, Households: [][]verdict.Point{hh600(6, false)}, Rate: 20000, TicksPerYear: tpy}
	check(t, "quiet role, collapse", verdict.Detect(verdict.R3, quiet, th), verdict.Detected, 365, "")
	check(t, "a role with a negative start", verdict.Detect(verdict.R3, verdict.Input{Roles: [][]verdict.Point{ticksOf(append([]int64{-1}, make([]int64, 400)...))}}, th), verdict.None, 0, "")

	// a start of 1 is a start: it falls to 0, so R4 yields to R3.
	vol := []verdict.Point{{Tick: 0, Value: 1000}, {Tick: 365, Value: 100}, {Tick: 730, Value: 100}}
	check(t, "start of 1 falling to 0", verdict.Detect(verdict.R4, verdict.Input{Volume: vol, Households: [][]verdict.Point{{{Value: 1}, {Tick: 10, Value: 0}}}}, th), verdict.None, 0, "")

	// the frozen flag is any non-zero value.
	neg := ticksOf(make([]int64, 70))
	for i := range neg {
		neg[i].Value = -1
	}
	check(t, "frozen flag of -1", verdict.Detect(verdict.R5b, verdict.Input{Frozen: [][]verdict.Point{neg}}, th), verdict.Detected, 59, "")

	// first_ticks of 1: one tick is its own mean, so never above twice it.
	one := with(th, "r7.first_ticks", 1)
	check(t, "first_ticks 1", verdict.Detect(verdict.R7, verdict.Input{Completions: [][]verdict.Point{ticksOf([]int64{5, 50})}}, one), verdict.None, 0, "")

	// an offset key of 1: an offset of 0 does not reach it.
	k1 := with(th, "r8.offset_bp", 1)
	in8 := verdict.Input{
		Offsets: []verdict.Series{{Points: ticksOf([]int64{0, 0, 0})}}, Unmet: ticksOf([]int64{9000, 9000, 9000}),
		Active: 10000, HasReference: true,
	}
	check(t, "offset 0 against a key of 1", verdict.Detect(verdict.R8, in8, k1), verdict.None, 0, "")
}

// TestSeedRules_Medians verifies the median of five and of seven values, and
// the Need of SR1 for n of 0.
//
// Requirement: FR-STY-076
func TestSeedRules_Medians(t *testing.T) {
	t.Parallel()
	if got := verdict.SR3(5, []int64{9, 1, 7, 3, 5}, 0, 99).Median; got != 5 {
		t.Errorf("SR3 median of five values = %d, want 5", got)
	}
	if got := verdict.SR3(7, []int64{70, 10, 40, 20, 60, 30, 50}, 0, 99).Median; got != 40 {
		t.Errorf("SR3 median of seven values = %d, want 40", got)
	}
	if got := verdict.SR1(0, nil); got.Need != 0 || got.Pass {
		t.Errorf("SR1(0, none) = %+v, want need 0 and fail", got)
	}
}
