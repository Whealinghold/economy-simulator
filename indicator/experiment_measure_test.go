package indicator_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/experiment"
	"github.com/Whealinghold/economy-sim/indicator"
	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// tallyState is what a tally measure keeps for one world: its tally, the snapshot
// that opened the window and the events the tally has seen since, so that a branch
// can rebuild an equal tally (a tally has no copy, spec-experiments.md 6.1.1).
type tallyState struct {
	tally  *indicator.Tally
	open   economysim.Snapshot
	events []economysim.Event
}

// feed gives the state one event, ignoring WorldRestored: a measure that needs the
// window to straddle the branch point does not let a restore start a new one.
func (s *tallyState) feed(e economysim.Event) {
	if _, restored := e.(economysim.WorldRestored); restored {
		return
	}
	s.events = append(s.events, e)
	s.tally.Observe(e)
}

// tallyMeasure is the measure a consumer writes (spec-experiments.md 3.4.0): the
// Observer builds a Tally, Begin opens it, Fn closes it and computes a figure of
// the window, and Fork copies the burn-in's tally into a branch.
func tallyMeasure(t testing.TB) experiment.Measure {
	states := map[experiment.Key]*tallyState{}
	var burn tallyState // the state of the burn-in when the first restore began
	return experiment.Measure{
		Name: "traded_and_executed",
		Observer: func(k experiment.Key) economysim.Observer {
			s, ok := states[k]
			if ok { // branch (0, 0) has the key of the burn-in: its second Observer continues it
				burn = tallyState{open: s.open, events: slices.Clone(s.events)}
			} else {
				s = &tallyState{tally: indicator.NewTally()}
				states[k] = s
			}
			return economysim.ObserverFunc(s.feed)
		},
		Begin: func(k experiment.Key, open economysim.Snapshot, _ int64, _ bool) {
			s := states[k]
			s.tally, s.open, s.events = indicator.NewTally(), open, nil
			if err := s.tally.Open(open); err != nil {
				t.Errorf("Tally.Open() error = %v, want nil", err)
			}
		},
		Fork: func(from, to experiment.Key) {
			if from == to {
				return
			}
			s := states[to]
			s.tally, s.open, s.events = indicator.NewTally(), burn.open, nil
			if err := s.tally.Open(burn.open); err != nil {
				t.Errorf("Tally.Open() error = %v, want nil", err)
			}
			for _, e := range burn.events {
				s.feed(e)
			}
		},
		Fn: func(s experiment.Sample) (int64, bool, error) {
			w, err := states[s.Key].tally.Close(s.Close)
			if err != nil {
				return 0, false, err
			}
			return figure(w), true, nil
		},
	}
}

// figure is the figure a tally measure reads: the volume cleared plus the recipe
// executions of the window.
func figure(w indicator.Window) int64 {
	var sum int64
	for _, m := range w.Markets {
		sum += m.SumV
	}
	for _, r := range w.Recipe {
		sum += r.Executions
	}
	return sum
}

// onePackDocs reads the layers of pack one as the harness takes them.
func onePackDocs(t testing.TB) [][]byte {
	t.Helper()
	var docs [][]byte
	for _, name := range runPacks[0].files {
		b, err := os.ReadFile(filepath.Join("..", "testdata", "chains", name))
		if err != nil {
			t.Fatalf("read pack layer %s: %v", name, err)
		}
		docs = append(docs, b)
	}
	return docs
}

// directFigure runs pack one for ticks with a Tally fed by the world and returns
// the figure of the one window [0, ticks): the run a consumer makes without a harness.
func directFigure(t testing.TB, seed uint64, ticks int64) int64 {
	t.Helper()
	w, rec := newRun(t, loadPack(t, runPacks[0].files...), seed, false)
	wins := stepWindows(t, w, rec, ticks, 1, nil)
	return figure(wins[0].w)
}

// TestExperiment_TallyMeasureEqualsDirectRun verifies, through the harness and with
// the real tally of this package, that a measure built as spec-experiments.md 3.4.0
// describes reads what a plain recorded run reads: the final reading of a sweep
// unit equals the figure of a tally fed by the world directly, and a branch whose
// Fork copies the tally (two variants, no steps and no reseed) reads the same as
// the uninterrupted run, whichever variant, as FR-EXP-031 asks of the digest.
//
// Requirement: FR-EXP-016, FR-EXP-030
func TestExperiment_TallyMeasureEqualsDirectRun(t *testing.T) {
	if testing.Short() {
		t.Skip("run-level test, skipped with -short")
	}
	const ticks, seed = 8, 1
	e := experiment.Spec{
		Name: "tally", Base: onePackDocs(t), Replicas: 1, Seed0: seed, Ticks: ticks,
		Measures: []experiment.Measure{tallyMeasure(t)},
	}
	whole, err := experiment.Run(context.Background(), e)
	if err != nil {
		t.Fatalf("Run(unbranched) error = %v, want nil", err)
	}
	want := directFigure(t, seed, ticks)
	got := whole.Points[0].Variants[0].Runs[0]
	if got.Status != experiment.StatusOK || got.Final[0] != (experiment.Reading{Value: want, OK: true}) || want == 0 {
		t.Fatalf("sweep reads %+v (%s %s), direct run reads %d, want equal and above 0", got.Final[0], got.Status, got.Detail, want)
	}

	e.Measures = []experiment.Measure{tallyMeasure(t)} // a branched run has its own states
	e.Branch = &experiment.Branch{At: 3, Variants: make([]experiment.Variant, 2)}
	branched, err := experiment.Run(context.Background(), e)
	if err != nil {
		t.Fatalf("Run(branched) error = %v, want nil", err)
	}
	for v, vr := range branched.Points[0].Variants {
		r := vr.Runs[0]
		if r.Status != experiment.StatusOK || r.Final[0] != got.Final[0] || r.Digest != got.Digest {
			t.Errorf("variant %d: status %s (%s), reading %+v, digest %x; want the uninterrupted run's reading %+v and digest %x", v, r.Status, r.Detail, r.Final[0], r.Digest[:4], got.Final[0], got.Digest[:4])
		}
	}
}

// TestSweep_StylisedVerdictAsResponse verifies FR-STY-090 on the harness: a
// one-at-a-time sweep of one knob at its default and two neighbours runs with a
// verdict (internal/verdict.Check) as the measure's reading, so the response of
// each point is STABLE (1) when every conjunct held and NOT STABLE (2) when one
// failed. The conjuncts are stand-ins: this checks the seam, not a band.
//
// Requirement: FR-STY-090
func TestSweep_StylisedVerdictAsResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("run-level test, skipped with -short")
	}
	response := experiment.Measure{
		Name: "verdict",
		Fn: func(s experiment.Sample) (int64, bool, error) {
			outcome := verdict.Held
			if s.Close.Households <= 0 {
				outcome = verdict.Failed
			}
			var got []verdict.Conjunct
			for _, n := range verdict.Required(verdict.CheckBP) {
				got = append(got, verdict.Conjunct{Name: n, Outcome: outcome})
			}
			return int64(verdict.Check(verdict.CheckBP, got).Result), true, nil
		},
	}
	level := func(v int64) experiment.Level {
		return experiment.Level{Changes: []experiment.Change{{Path: "world.belief_step", Value: v}}}
	}
	e := experiment.Spec{
		Name: "verdict", Base: onePackDocs(t), Replicas: 1, Seed0: 1, Ticks: 4,
		Design:   experiment.OneAtATime,
		Factors:  []experiment.Factor{{Levels: []experiment.Level{level(150), level(200), level(250)}}},
		Measures: []experiment.Measure{response},
	}
	res, err := experiment.Run(context.Background(), e)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if len(res.Points) != 3 {
		t.Fatalf("points = %d, want 3 (the default and its two neighbours)", len(res.Points))
	}
	for _, p := range res.Points {
		r := p.Variants[0].Runs[0]
		if r.Status != experiment.StatusOK || r.Final[0] != (experiment.Reading{Value: int64(verdict.Stable), OK: true}) {
			t.Errorf("point %d: status %s (%s), reading %+v, want ok and the verdict STABLE", p.Index, r.Status, r.Detail, r.Final[0])
		}
	}
}
