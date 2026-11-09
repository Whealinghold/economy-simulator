package experiment_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/experiment"
)

// eventCounts is the state of the recording-observer measure of the golden: the
// events seen by a world, and the count at the start of the open window and of the
// span.
type eventCounts struct{ total, win, span int64 }

// countingMeasure returns a measure that reads events: it counts those a world
// emits, per window and over the span. It is the shape of a measure that builds a
// tally (spec-experiments.md 3.4.0, 6.1.1): Observer(key) gives the world its
// observer, Begin opens a window, Fn closes it, and Fork copies the burn-in's
// count into a branch. For branch (0, 0), whose key is the burn-in's, the second
// Observer(k) call continues the state the first built (Fork(k, k) means keep).
func countingMeasure() experiment.Measure {
	live := map[experiment.Key]*eventCounts{}
	var burn eventCounts // the state of the burn-in when the first restore began
	return experiment.Measure{
		Name: "events",
		Observer: func(k experiment.Key) economysim.Observer {
			c, ok := live[k]
			if ok {
				burn = *c
			} else {
				c = &eventCounts{}
				live[k] = c
			}
			return economysim.ObserverFunc(func(e economysim.Event) {
				if _, restored := e.(economysim.WorldRestored); !restored {
					c.total++
				}
			})
		},
		Begin: func(k experiment.Key, _ economysim.Snapshot, _ int64, span bool) {
			if c := live[k]; span {
				c.span = c.total
			} else {
				c.win = c.total
			}
		},
		Fork: func(from, to experiment.Key) {
			if from != to {
				*live[to] = burn
			}
		},
		Fn: func(s experiment.Sample) (int64, bool, error) {
			c := live[s.Key]
			if s.Span {
				return c.total - c.span, true, nil
			}
			return c.total - c.win, true, nil
		},
	}
}

// goldenSpec is the experiment of testdata/golden/experiment/one.txt (plan 2-20
// D615): pack one, belief_step at 200, 300 and 400 (the plan says 100, 200 and 400, but 100 is
// invalid_config on the current pack: its warm-start price 96 is outside a bound of 100), three seeds, 120 ticks in
// windows of 30, a branch at tick 60 of two variants, three snapshot measures and
// the event counter. The spec's 400 ticks and ten seeds are the scenarios' size,
// not the harness's: this proves the bytes, not the economy (N-027).
func goldenSpec(t testing.TB) experiment.Spec {
	t.Helper()
	f := experiment.Factor{Name: "belief_step"}
	for _, v := range []int64{200, 300, 400} {
		f.Levels = append(f.Levels, experiment.Level{Changes: []experiment.Change{{Path: "world.belief_step", Value: v}}})
	}
	agents := experiment.Measure{Name: "agents", Fn: func(s experiment.Sample) (int64, bool, error) {
		return s.Close.Agents, true, nil
	}}
	return experiment.Spec{
		Name:     "pack_one_belief_step",
		Base:     onePack(t),
		Factors:  []experiment.Factor{f},
		Replicas: 3,
		Seed0:    1,
		Ticks:    120,
		Every:    30,
		Keep:     experiment.Keep{Series: true},
		Measures: []experiment.Measure{households, money, agents, countingMeasure()},
		Criteria: []experiment.Criterion{{Name: "stays_small", Measure: "households", Op: experiment.AtMost, A: 650}},
		Branch: &experiment.Branch{At: 60, Variants: []experiment.Variant{
			{Label: "none"},
			// One more tick from the branch point stands in for a shock: the public
			// SetYieldShocks is plan 2-17's. Plan 8's acceptance re-records the golden.
			{Label: "extra tick", Steps: []experiment.Step{{Tick: 60, Label: "step", Do: func(w *economysim.World) error { return w.Step() }}}},
		}},
	}
}

// goldenResult runs the golden experiment once for the tests of this file; the
// result is the same value for each, and a failing run fails each of them.
func goldenResult(t testing.TB) (experiment.Result, error) {
	goldenOnce.Do(func() {
		goldenRes, goldenErr = experiment.Run(context.Background(), goldenSpec(t))
	})
	return goldenRes, goldenErr
}

var (
	goldenOnce sync.Once
	goldenRes  experiment.Result
	goldenErr  error
)

// TestGolden_ExperimentOnPackOne verifies that Render of the sweep of the golden
// experiment equals testdata/golden/experiment/one.txt, byte for byte. A change of
// the harness, the pack or the library that moves it is a behaviour change:
// regenerate with -update only after reading testing.md 7.2. The digests in it are
// to be compared on two architectures by hand until plan 1's CI decision.
//
// This golden is recorded on the current reference pack: plan 8's acceptance
// re-records it when the pack is rebalanced.
//
// Requirement: NFR-EXP-001, FR-EXP-060
func TestGolden_ExperimentOnPackOne(t *testing.T) {
	short(t)
	if raceEnabled {
		t.Skip("the bytes are proven by the run without -race (gate F and the quick run); no goroutine is involved")
	}
	res, err := goldenResult(t)
	if err != nil {
		t.Fatalf("Run(golden) error = %v, want nil", err)
	}
	got := experiment.Render(res)
	path := filepath.Join("..", "testdata", "golden", "experiment", "one.txt")
	if *updateFlag {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v, want nil", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v, want nil", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v; record it with -update after reading testing.md 7.2", path, err)
	}
	if string(want) == got {
		return
	}
	wl, gl := strings.Split(string(want), "\n"), strings.Split(got, "\n")
	for i := range max(len(wl), len(gl)) {
		var a, b string
		if i < len(wl) {
			a = wl[i]
		}
		if i < len(gl) {
			b = gl[i]
		}
		if a != b {
			t.Fatalf("golden %s differs at line %d:\n want %q\n  got %q\n(%d lines wanted, %d got; regenerate with -update only after reading testing.md 7.2)", path, i+1, a, b, len(wl), len(gl))
		}
	}
}

// TestGolden_RenderTwiceEqual verifies that Render is a pure function of a Result:
// twice on one result, and on the result assembled from the same records in
// another order, it gives one string, and the golden result has the shape the file
// claims (three points, two variants, three replicas, four measures).
//
// Requirement: NFR-EXP-001, FR-EXP-060
func TestGolden_RenderTwiceEqual(t *testing.T) {
	short(t)
	res, err := goldenResult(t)
	if err != nil {
		t.Fatalf("Run(golden) error = %v, want nil", err)
	}
	var recs []experiment.Record
	for _, p := range res.Points {
		for _, v := range p.Variants {
			recs = append(recs, v.Runs...)
		}
	}
	slices.Reverse(recs)
	again, err := experiment.Assemble(goldenSpec(t), recs)
	if err != nil {
		t.Fatalf("Assemble() error = %v, want nil", err)
	}
	if a, b := experiment.Render(res), experiment.Render(again); a != b || a != experiment.Render(res) {
		t.Errorf("Render differs between two calls or two orders of the records")
	}
	if len(res.Points) != 3 || len(res.Points[0].Variants) != 2 || len(res.Points[0].Variants[0].Runs) != 3 || len(res.Measures) != 4 {
		t.Errorf("golden result shape = %d points, %d measures, want 3 points of 2 variants of 3 runs and 4 measures", len(res.Points), len(res.Measures))
	}
	for _, p := range res.Points {
		for _, v := range p.Variants {
			for _, r := range v.Runs {
				if r.Status != experiment.StatusOK {
					t.Errorf("golden run %+v has status %s (%s), want ok: a golden of failures proves nothing", r.Key, r.Status, r.Detail)
				}
			}
		}
	}
}
