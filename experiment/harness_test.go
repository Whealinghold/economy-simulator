package experiment_test

import (
	"context"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/experiment"
)

// toyNoise is a deterministic integer noise in [9000, 11000] of a seed: the
// SplitMix64 finaliser of the seed, modulo 2001. It stands for "a measure of a
// run", with a known spread (uniform: standard deviation about 577), so the tests
// of the scenarios assert the harness's arithmetic and not the economy (N-027).
func toyNoise(seed uint64) int64 {
	z := seed + 0x9E3779B97F4A7C15
	z = (z ^ z>>30) * 0xBF58476D1CE4E5B9
	z = (z ^ z>>27) * 0x94D049BB133111EB
	z ^= z >> 31
	return 9000 + int64(z%2001)
}

// toyColumn assembles points x replicas of toy noise: the seed of replica r of
// point p is base + p*replicas + r, so every run of every point has its own.
func toyColumn(t testing.TB, points, replicas int, base uint64) experiment.Result {
	t.Helper()
	e := toySpec(replicas, experiment.Grid) // one point: the base
	if points > 1 {
		e = toySpec(replicas, experiment.Grid, knob(1, int64(points), points))
	}
	var recs []experiment.Record
	for p := range points {
		for r := range replicas {
			seed := base + uint64(p*replicas+r)
			rec := okRec(p, r, seed, toyNoise(seed))
			rec.Pass[0] = rec.Final[0].Value <= 10000
			recs = append(recs, rec)
		}
	}
	return assemble(t, e, recs...)
}

// TestHarness_SpreadFallsAsRootR is the scenario EXP-S1 (spec-experiments.md 17,
// register entry S84): the standard deviation of a mean over R seeds of a toy
// process falls as 1/sqrt(R). 64 batches of R = 4, 16 and 64 runs each give 64
// means; the standard deviation of those means, read through the harness's own
// reduction, halves when R quadruples to within a factor the placeholders allow
// (the sampling error of a standard deviation of 64 values is about 9%).
//
// Requirement: FR-EXP-040, FR-EXP-042
func TestHarness_SpreadFallsAsRootR(t *testing.T) {
	t.Parallel()
	stdOfMeans := func(replicas int) int64 {
		means := make([][]int64, 1)
		for b := range 64 {
			res := toyColumn(t, 1, replicas, uint64(1+b*1000))
			means[0] = append(means[0], res.Points[0].Variants[0].Reductions[0].Mean.Value)
		}
		e := toySpec(64, experiment.Grid)
		return column(t, e, means).Points[0].Variants[0].Reductions[0].Std.Value
	}
	s4, s16, s64 := stdOfMeans(4), stdOfMeans(16), stdOfMeans(64)
	// 577/sqrt(4) = 289, 577/sqrt(16) = 144, 577/sqrt(64) = 72: each step halves.
	for _, step := range []struct{ big, small int64 }{{s4, s16}, {s16, s64}} {
		if step.big*10 < step.small*15 || step.big*10 > step.small*27 {
			t.Errorf("std of the means over R = 4, 16, 64 seeds = %d, %d, %d: a step does not halve (ratio %d/%d outside 1.5 to 2.7)", s4, s16, s64, step.big, step.small)
		}
	}
}

// TestHarness_WinnersCurse is the scenario EXP-S2 (register entry S85): of 100
// points with one true pass share (a toy run passes when its noise is at most
// 10 000, about one in two), the point with the best share on the choosing seeds
// has a lower share on other seeds, and the average share on other seeds is near
// the true one. The reductions are the harness's own.
//
// Requirement: FR-EXP-043
func TestHarness_WinnersCurse(t *testing.T) {
	t.Parallel()
	choosing, confirm := toyColumn(t, 100, 10, 1), toyColumn(t, 100, 10, 1_000_000)
	best, sum := 0, 0
	for p, pt := range choosing.Points {
		if pt.Variants[0].Pass[0].ShareBP > choosing.Points[best].Variants[0].Pass[0].ShareBP {
			best = p
		}
		sum += confirm.Points[p].Variants[0].Pass[0].ShareBP
	}
	chosen, again := choosing.Points[best].Variants[0].Pass[0].ShareBP, confirm.Points[best].Variants[0].Pass[0].ShareBP
	if chosen < 7000 || again >= chosen {
		t.Errorf("the chosen point %d passes %d bp on the choosing seeds and %d on others, want a high share that falls", best, chosen, again)
	}
	if mean := sum / 100; mean < 4500 || mean > 5500 {
		t.Errorf("the mean pass share on the confirming seeds = %d bp, want about 5 000", mean)
	}
}

// ramp is a toy process with a known approach: y(t) = 100 * min(t, 10), which
// settles at 1 000 from tick 10. A window's reading is the mean of y over its
// ticks (exact for the windows below).
var ramp = experiment.Measure{Name: "ramp", Fn: func(s experiment.Sample) (int64, bool, error) {
	var sum int64
	for tick := s.From; tick < s.To; tick++ {
		sum += 100 * min(tick, 10)
	}
	return sum / (s.To - s.From), true, nil
}}

// TestHarness_TransientWindows is the scenario EXP-S3 (register entry S86): on a
// toy process with a known approach, a window opened at 0 and one opened at the
// settling tick differ by the stated amount, and the windows of a short series
// are the stated means.
//
// Over [0, 20) the sum of y is 4 500 + 10 * 1 000 = 14 500, a mean of 725; over
// [10, 20) it is 1 000. The windows of 5 ticks: (0+100+200+300+400)/5 = 200,
// (500+...+900)/5 = 700, then 1 000 and 1 000.
//
// Requirement: FR-EXP-016
func TestHarness_TransientWindows(t *testing.T) {
	short(t)
	run := func(transient, every int64) experiment.Record {
		e := packSpec(t)
		e.Ticks, e.Transient, e.Every = 20, transient, every
		e.Measures = []experiment.Measure{ramp}
		e.Keep.Series = true
		return runSweep(t, e)[0]
	}
	whole := run(0, 5)
	if whole.Final[0].Value != 725 {
		t.Errorf("final reading from tick 0 = %d, want 725", whole.Final[0].Value)
	}
	var got []int64
	for _, row := range whole.Series {
		got = append(got, row[0].Value)
	}
	if want := []int64{200, 700, 1000, 1000}; !slices.Equal(got, want) || !slices.Equal(whole.Ticks, []int64{5, 10, 15, 20}) {
		t.Errorf("windows of 5 ticks = %v at %v, want %v at 5, 10, 15, 20", got, whole.Ticks, want)
	}
	settled := run(10, 5)
	if settled.Final[0].Value != 1000 || !slices.Equal(settled.Ticks, []int64{15, 20}) {
		t.Errorf("final reading from the settling tick = %d at windows %v, want 1000 and windows ending at 15, 20", settled.Final[0].Value, settled.Ticks)
	}
	if d := settled.Final[0].Value - whole.Final[0].Value; d != 275 {
		t.Errorf("difference of the two openings = %d, want 275: the approach, 1 000 - 725", d)
	}
}

// TestHarness_SurvivorBias is the scenario EXP-S4 (register entry S87): a sweep
// whose points fail at known rates reports Failed and Planned equal to those rates,
// the pass share over Planned, and a survivor mean that differs from the mean of
// all runs by the stated bias.
//
// Point p fails its first 3p runs (0, 3 and 6 of 10). A run r reads 10 000 + 100 r.
// All ten average 10 450; the survivors of point 1 (r = 3..9) average 10 600, and
// those of point 2 (r = 6..9) 10 750: a bias of 150 and 300.
//
// Requirement: FR-EXP-041, FR-EXP-050
func TestHarness_SurvivorBias(t *testing.T) {
	t.Parallel()
	e := toySpec(10, experiment.Grid, knob(1, 3, 3))
	var recs []experiment.Record
	for p := range 3 {
		for r := range 10 {
			if r < 3*p {
				recs = append(recs, failedRec(p, r, uint64(1+r), experiment.StatusWorld))
				continue
			}
			recs = append(recs, okRec(p, r, uint64(1+r), int64(10000+100*r)))
		}
	}
	res := assemble(t, e, recs...)
	for p, want := range []struct{ failed, used, mean, share int }{{0, 10, 10450, 10000}, {3, 7, 10600, 7000}, {6, 4, 10750, 4000}} {
		red, ps := res.Points[p].Variants[0].Reductions[0], res.Points[p].Variants[0].Pass[0]
		if red.Planned != 10 || red.Failed != want.failed || red.Used != want.used || red.Mean != rd(int64(want.mean)) || ps.ShareBP != want.share {
			t.Errorf("point %d: %+v, share %d, want failed %d used %d of 10 planned, mean %d, share %d", p, red, ps.ShareBP, want.failed, want.used, want.mean, want.share)
		}
	}
	if bias := res.Points[2].Variants[0].Reductions[0].Mean.Value - res.Points[0].Variants[0].Reductions[0].Mean.Value; bias != 300 {
		t.Errorf("survivor bias of point 2 = %d, want 300", bias)
	}
}

// TestRecord_WithinTheSizeFormula verifies the size formula of spec-experiments.md
// 8.6.1: a record holds one final reading per measure and, with Keep.Series, one
// more per measure and window, and nothing grows with the ticks without the series.
//
// Requirement: NFR-EXP-004
func TestRecord_WithinTheSizeFormula(t *testing.T) {
	short(t)
	for _, ticks := range []int64{6, 12} {
		e := packSpec(t)
		e.Ticks, e.Every = ticks, 3
		bare := runSweep(t, e)[0]
		if len(bare.Series) != 0 || len(bare.Ticks) != 0 || len(bare.Final) != 2 {
			t.Errorf("record of %d ticks without the series holds %d final readings, %d windows, %d ticks, want 2, 0, 0", ticks, len(bare.Final), len(bare.Series), len(bare.Ticks))
		}
		e.Keep.Series = true
		kept := runSweep(t, e)[0]
		windows := int(ticks / 3)
		if cells := len(kept.Final) + len(kept.Series)*len(kept.Final); len(kept.Series) != windows || cells > len(e.Measures)*(windows+1) {
			t.Errorf("record of %d ticks with the series holds %d windows and %d cells, want %d windows and at most %d cells", ticks, len(kept.Series), cells, windows, len(e.Measures)*(windows+1))
		}
	}
}

// TestRun_InvalidExperiment verifies that Run refuses an invalid experiment with
// the error of Validate and an empty Result, and runs nothing.
//
// Requirement: FR-EXP-004
func TestRun_InvalidExperiment(t *testing.T) {
	t.Parallel()
	called := false
	e := valid()
	e.Replicas = 0
	e.Options = func(experiment.Key) []economysim.Option { called = true; return nil }
	res, err := experiment.Run(context.Background(), e)
	if err == nil || res.Points != nil || called {
		t.Errorf("Run(invalid) = %+v, %v (Options called: %v), want an error, an empty result and no run", res, err, called)
	}
}
