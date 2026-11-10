package experiment_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/experiment"
)

// reduceOut are the ten readings of spec-experiments.md 19.8 (models/exp/reduce.out):
// the price range of a market in basis points of the base.
var reduceOut = []int64{10100, 10200, 10300, 10400, 10500, 10600, 10700, 10900, 11800, 12100}

// reductionOf assembles one point of the readings, failing the runs listed in
// failed, and returns the reduction of the one measure and the pass share of "calm".
func reductionOf(t testing.TB, vals []int64, failed ...int) (experiment.Reduction, experiment.PassShare) {
	t.Helper()
	e := toySpec(len(vals), experiment.Grid)
	var recs []experiment.Record
	for r, v := range vals {
		rec := okRec(0, r, uint64(1+r), v)
		for _, f := range failed {
			if f == r {
				rec = failedRec(0, r, uint64(1+r), experiment.StatusInvalidTrade)
			}
		}
		recs = append(recs, rec)
	}
	v := assemble(t, e, recs...).Points[0].Variants[0]
	return v.Reductions[0], v.Pass[0]
}

func readings(vs ...int64) [5]experiment.Reading {
	var out [5]experiment.Reading
	for i, v := range vs {
		out[i] = experiment.Reading{Value: v, OK: true}
	}
	return out
}

// TestReduce_AgainstReduceOut verifies the reductions of spec-experiments.md 19.8
// against the numbers of models/exp/reduce.out: ten replicas, then the same with the
// run that read 10 600 failed (a survivor mean of 10 778 and a pass share over the
// planned 10 of 7 000), then with the run that read 12 100 failed.
//
// Requirement: FR-EXP-040, FR-EXP-041, FR-EXP-042, FR-EXP-043
func TestReduce_AgainstReduceOut(t *testing.T) {
	t.Parallel()
	rd := func(v int64) experiment.Reading { return experiment.Reading{Value: v, OK: true} }
	tests := []struct {
		name   string
		failed []int
		red    experiment.Reduction
		share  experiment.PassShare
	}{
		// Sum 107 600 over 10 is 10 760; the nearest ranks of 10 are 1, 3, 5, 8 and 9
		// (ceil(bp * 10 / 10 000)); at most 11 000 are the first 8.
		{"all ten", nil, experiment.Reduction{Planned: 10, Used: 10, Min: rd(10100), Max: rd(12100), Mean: rd(10760), Std: rd(674),
			Q: readings(10100, 10300, 10500, 10900, 11800)}, experiment.PassShare{Passes: 8, Planned: 10, ShareBP: 8000}},
		// 107 600 - 10 600 = 97 000 over 9 = 10 777.8; the variance is 506 944 = 712^2; the
		// ranks of 9 are 1, 3, 5, 7 and 9; 7 of the 10 planned pass (the failed run does not).
		{"10 600 failed", []int{5}, experiment.Reduction{Planned: 10, Used: 9, Failed: 1, Min: rd(10100), Max: rd(12100), Mean: rd(10778), Std: rd(712),
			Q: readings(10100, 10300, 10500, 10900, 12100)}, experiment.PassShare{Passes: 7, Planned: 10, ShareBP: 7000}},
		// 107 600 - 12 100 = 95 500 over 9 = 10 611.1; the variance is 261 111.1, rounded to
		// 261 111, whose root is 511.0; the eight passing runs stay.
		{"12 100 failed", []int{9}, experiment.Reduction{Planned: 10, Used: 9, Failed: 1, Min: rd(10100), Max: rd(11800), Mean: rd(10611), Std: rd(511),
			Q: readings(10100, 10300, 10500, 10700, 11800)}, experiment.PassShare{Passes: 8, Planned: 10, ShareBP: 8000}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			red, share := reductionOf(t, reduceOut, tt.failed...)
			if red != tt.red {
				t.Errorf("Reduction(%v, failed %v) =\n %+v\nwant\n %+v", reduceOut, tt.failed, red, tt.red)
			}
			if share != tt.share {
				t.Errorf("PassShare(%v, failed %v) = %+v, want %+v", reduceOut, tt.failed, share, tt.share)
			}
		})
	}
}

// TestReduce_PlannedIdentity verifies, over many deterministic patterns of
// successes, failures, undefined readings and absent records, that
// Planned = Used + Failed + Undefined + Missing and Planned is the replica count.
//
// Requirement: FR-EXP-041
func TestReduce_PlannedIdentity(t *testing.T) {
	t.Parallel()
	state := uint64(12345)
	next := func() uint64 { // a small LCG: the test needs patterns, not randomness
		state = state*6364136223846793005 + 1442695040888963407
		return state >> 33
	}
	for pattern := range 300 {
		replicas := 1 + int(next()%12)
		e := toySpec(replicas, experiment.Grid)
		var recs []experiment.Record
		for r := range replicas {
			switch next() % 4 {
			case 0:
				recs = append(recs, failedRec(0, r, uint64(1+r), experiment.StatusStep))
			case 1:
				rec := okRec(0, r, uint64(1+r), 1)
				rec.Final[0] = experiment.Reading{}
				recs = append(recs, rec)
			case 2: // absent: no record
			default:
				recs = append(recs, okRec(0, r, uint64(1+r), int64(next()%20000)))
			}
		}
		red := assemble(t, e, recs...).Points[0].Variants[0].Reductions[0]
		if red.Planned != replicas || red.Planned != red.Used+red.Failed+red.Undefined+red.Missing {
			t.Fatalf("pattern %d: %+v, want Planned = %d = Used + Failed + Undefined + Missing", pattern, red, replicas)
		}
	}
}

// FuzzReduce verifies that any readings, statuses and absences give a reduction
// that does not panic and keeps Planned = Used + Failed + Undefined + Missing.
//
// Requirement: NFR-EXP-005
func FuzzReduce(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	f.Add([]byte{255, 0, 255, 0})
	f.Fuzz(func(t *testing.T, data []byte) {
		replicas := min(len(data)/2, 40)
		if replicas == 0 {
			return
		}
		e := toySpec(replicas, experiment.Grid)
		var recs []experiment.Record
		for r := range replicas {
			kind, v := data[2*r]%4, int64(int8(data[2*r+1]))*1_000_000_000
			switch kind {
			case 0:
				recs = append(recs, failedRec(0, r, uint64(1+r), experiment.StatusWorld))
			case 1:
				rec := okRec(0, r, uint64(1+r), v)
				rec.Final[0].OK = false
				recs = append(recs, rec)
			case 2:
			default:
				recs = append(recs, okRec(0, r, uint64(1+r), v))
			}
		}
		res, err := experiment.Assemble(e, recs)
		if err != nil {
			t.Fatalf("Assemble() error = %v, want nil", err)
		}
		red := res.Points[0].Variants[0].Reductions[0]
		if red.Planned != replicas || red.Planned != red.Used+red.Failed+red.Undefined+red.Missing {
			t.Fatalf("Reduction(%v) = %+v, want the Planned identity over %d replicas", data, red, replicas)
		}
		_ = experiment.Render(res)
	})
}

// TestQuantile_ClosedList verifies the closed list of quantiles of a Reduction and
// that an index outside it is 0.
//
// Requirement: FR-EXP-042
func TestQuantile_ClosedList(t *testing.T) {
	t.Parallel()
	for i, want := range []int{1000, 2500, 5000, 7500, 9000} {
		if got := experiment.Quantile(i); got != want {
			t.Errorf("Quantile(%d) = %d, want %d", i, got, want)
		}
	}
	for _, i := range []int{-1, 5} {
		if got := experiment.Quantile(i); got != 0 {
			t.Errorf("Quantile(%d) = %d, want 0", i, got)
		}
	}
}
