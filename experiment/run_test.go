package experiment_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/experiment"
)

func short(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
}

// badClearing is a Clearing whose trade list the world refuses once on is set:
// a buyer and a seller that are one agent.
type badClearing struct {
	economysim.AverageClearing
	on *bool
}

func (b badClearing) Clear(bids, asks []economysim.Order, rng *rand.Rand) []economysim.Trade {
	if b.on != nil && *b.on {
		return []economysim.Trade{{Commodity: 0, Buyer: 1, Seller: 1, Price: 1, Quantity: 1}}
	}
	return b.AverageClearing.Clear(bids, asks, rng)
}

// useClearing returns a pack-one level whose document selects the clearing
// "faulty", and an Options that registers it with the flag.
func faultyOn(on *bool) func(experiment.Key) []economysim.Option {
	return func(experiment.Key) []economysim.Option {
		return []economysim.Option{economysim.WithClearing("faulty", badClearing{on: on})}
	}
}

var faultyDoc = []byte("[world]\nclearing = \"faulty\"\n")

// TestRunUnit_PureFunction verifies that a unit run twice gives equal records,
// and that the record is that of a healthy run: the status, the digest, the
// readings and the hash.
//
// Requirement: FR-EXP-001, NFR-EXP-001
func TestRunUnit_PureFunction(t *testing.T) {
	short(t)
	e := packSpec(t)
	e.Replicas = 2
	a, b := runSweep(t, e), runSweep(t, e)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("two runs of one experiment gave different records:\n%+v\n%+v", a, b)
	}
	if ra, rb := experiment.Render(assemble(t, e, a...)), experiment.Render(assemble(t, e, b...)); ra != rb {
		t.Fatalf("two runs of one experiment rendered differently:\n%s\n%s", ra, rb)
	}
	r := a[0]
	if r.Status != experiment.StatusOK || r.FailTick != -1 || r.Digest == [32]byte{} || r.Hash == [32]byte{} {
		t.Errorf("record = %+v, want ok, FailTick -1, a digest and a hash", r)
	}
	if want := (experiment.Reading{Value: 600, OK: true}); r.Final[0] != want {
		t.Errorf("households reading = %+v, want %+v", r.Final[0], want)
	}
	if a[0].Digest == a[1].Digest || a[0].Seed != 1 || a[1].Seed != 2 {
		t.Errorf("replicas share a digest or have seeds %d, %d, want 1, 2 and different digests", a[0].Seed, a[1].Seed)
	}
}

// TestResolve_LoaderRejections verifies that the loader's own refusals are the
// point's: an unknown key, a float and a scalar over a table are unresolved, a
// value out of its bound is invalid_config, the other point runs, and Check
// says the same without a tick.
//
// Requirement: FR-EXP-002, FR-EXP-050, FR-EXP-070
func TestResolve_LoaderRejections(t *testing.T) {
	short(t)
	e := packSpec(t)
	bad := func(doc []byte) experiment.Level { return experiment.Level{Docs: [][]byte{doc}} }
	e.Factors = []experiment.Factor{{Levels: []experiment.Level{
		{Changes: []experiment.Change{{Path: "world.belief_step", Value: 150}}},
		{Changes: []experiment.Change{{Path: "world.cost_flor", Value: 1}}},
		bad([]byte("[world]\nbelief_step = 1.5\n")),
		{Changes: []experiment.Change{{Path: "region.north", Value: 1}}},
		{Changes: []experiment.Change{{Path: "world.belief_step", Value: 9999}}},
	}}}
	want := []experiment.Status{experiment.StatusOK, experiment.StatusUnresolved, experiment.StatusUnresolved, experiment.StatusUnresolved, experiment.StatusInvalidConfig}
	recs := runSweep(t, e)
	checks, err := experiment.Check(context.Background(), e)
	if err != nil || len(checks) != len(want) {
		t.Fatalf("Check() = %v, %v, want %d points", checks, err, len(want))
	}
	for p, st := range want {
		if recs[p].Status != st || checks[p].Status != st {
			t.Errorf("point %d: run status %q, Check status %q, want %q (detail %q)", p, recs[p].Status, checks[p].Status, st, recs[p].Detail)
		}
		if st != experiment.StatusOK && (recs[p].Detail == "" || recs[p].FailTick != 0 || recs[p].Hash != [32]byte{} || recs[p].Digest != [32]byte{}) {
			t.Errorf("point %d: record %+v, want detail, FailTick 0, no hash, no digest", p, recs[p])
		}
		if st != experiment.StatusOK && (recs[p].Final[0].OK || recs[p].Pass == nil) {
			t.Errorf("point %d: Final %+v, want undefined", p, recs[p].Final)
		}
	}
}

// TestResolve_PrepareAfterOverlaysBeforeValidate verifies that Prepare sees the
// point's overlays and that its error is invalid_config with no world built.
//
// Requirement: FR-EXP-013
func TestResolve_PrepareAfterOverlaysBeforeValidate(t *testing.T) {
	short(t)
	e := packSpec(t)
	e.Factors = []experiment.Factor{{Levels: []experiment.Level{
		{Changes: []experiment.Change{{Path: "world.belief_step", Value: 150}}},
		{Changes: []experiment.Change{{Path: "world.belief_step", Value: 200}}},
	}}}
	var seen []int64
	boom := errors.New("prepare refuses")
	e.Prepare = func(_ context.Context, cfg economysim.Config) (economysim.Config, error) {
		seen = append(seen, cfg.World.BeliefStep)
		if cfg.World.BeliefStep == 200 {
			return cfg, boom
		}
		cfg.World.BeliefStep = 9999 // out of its bound: only Validate, after Prepare, can see it
		return cfg, nil
	}
	recs := runSweep(t, e)
	if !slices.Equal(seen, []int64{150, 200}) {
		t.Errorf("Prepare saw belief_step %v, want [150 200]: the overlays of each point", seen)
	}
	for p, r := range recs {
		if r.Status != experiment.StatusInvalidConfig || r.Hash != [32]byte{} {
			t.Errorf("point %d: status %q hash %x, want invalid_config and no world (detail %q)", p, r.Status, r.Hash, r.Detail)
		}
	}
	if !strings.Contains(recs[1].Detail, "prepare refuses") {
		t.Errorf("point 1 detail = %q, want the Prepare error", recs[1].Detail)
	}
}

// TestRun_StepsMergedByTick verifies that the script's step comes before the
// level's at an equal tick, that each runs at the boundary of its tick (the
// world has done exactly that many ticks) and never inside a step, and that a
// step at tick 0 runs before the first tick.
//
// Requirement: FR-EXP-014
func TestRun_StepsMergedByTick(t *testing.T) {
	short(t)
	var log []string
	do := func(label string) func(*economysim.World) error {
		return func(w *economysim.World) error { log = append(log, fmt.Sprintf("%s@%d", label, w.Tick())); return nil }
	}
	e := packSpec(t)
	e.Script = []experiment.Step{{Tick: 2, Label: "script2", Do: do("script2")}, {Tick: 0, Label: "script0", Do: do("script0")}}
	e.Factors = []experiment.Factor{{Levels: []experiment.Level{
		{Steps: []experiment.Step{{Tick: 2, Do: do("level2")}, {Tick: 1, Do: do("level1")}}},
		{Changes: []experiment.Change{{Path: "world.belief_step", Value: 150}}},
	}}}
	e.Design = experiment.OneAtATime
	runSweep(t, e)
	// Point 0 is the baseline, level 0: its steps. Point 1 is level 1 with no level steps.
	want := []string{"script0@0", "level1@1", "script2@2", "level2@2", "script0@0", "script2@2"}
	if !slices.Equal(log, want) {
		t.Errorf("steps applied = %v, want %v", log, want)
	}
}

// TestRun_StepErrorIsScript verifies that an error out of Step.Do fails its run
// with script at the step's tick, keeps the earlier calls, gives no digest and
// no reading, and that the next run goes on.
//
// Requirement: FR-EXP-014, FR-EXP-050
func TestRun_StepErrorIsScript(t *testing.T) {
	short(t)
	boom := errors.New("step refuses")
	e := packSpec(t)
	e.Replicas = 2
	calls := 0
	e.Script = []experiment.Step{{Tick: 1, Label: "bad", Do: func(*economysim.World) error {
		calls++
		if calls == 1 {
			return boom
		}
		return nil
	}}}
	recs := runSweep(t, e)
	if r := recs[0]; r.Status != experiment.StatusScript || r.FailTick != 1 || r.Digest != [32]byte{} || r.Final[0].OK || !strings.Contains(r.Detail, "step refuses") || r.Hash == [32]byte{} {
		t.Errorf("failed record = %+v, want script at tick 1, no digest, undefined readings, the error, and a hash", r)
	}
	if recs[1].Status != experiment.StatusOK {
		t.Errorf("next run status = %q, want ok", recs[1].Status)
	}
}

// call is one hook call a measure saw, for the tests of the windows.
type call struct {
	kind     string // begin, fn, do
	from, to int64
	span     bool
	key      experiment.Key
}

// recorder returns a measure that logs its Begin and Fn calls to log and reads
// To of its sample, and the log.
func recorder(log *[]call) experiment.Measure {
	return experiment.Measure{
		Name: "r",
		Begin: func(k experiment.Key, open economysim.Snapshot, from int64, span bool) {
			*log = append(*log, call{kind: "begin", from: from, span: span, key: k})
		},
		Fn: func(s experiment.Sample) (int64, bool, error) {
			*log = append(*log, call{kind: "fn", from: s.From, to: s.To, span: s.Span, key: s.Key})
			return s.To, true, nil
		},
	}
}

// TestRun_WindowsAndFinalReading verifies the windows of a run of 25 ticks with
// a transient of 5 and a window of 8: [5,13), [13,21), [21,25) (the last
// shorter), the span [5,25) for the final reading, the series with Keep.Series,
// and, with Every 0, one window that is the span with no series.
//
// Requirement: FR-EXP-016
func TestRun_WindowsAndFinalReading(t *testing.T) {
	short(t)
	var log []call
	e := packSpec(t)
	e.Ticks, e.Transient, e.Every, e.Keep.Series = 25, 5, 8, true
	e.Measures = []experiment.Measure{recorder(&log)}
	r := runSweep(t, e)[0]
	if want := []int64{13, 21, 25}; !slices.Equal(r.Ticks, want) {
		t.Errorf("window ends = %v, want %v", r.Ticks, want)
	}
	if len(r.Series) != 3 || r.Series[0][0].Value != 13 || r.Series[2][0].Value != 25 {
		t.Errorf("series = %v, want the To of each window: 13, 21, 25", r.Series)
	}
	if want := (experiment.Reading{Value: 25, OK: true}); r.Final[0] != want {
		t.Errorf("final = %+v, want the span read at 25", r.Final[0])
	}
	var fns []call
	for _, c := range log {
		if c.kind == "fn" {
			fns = append(fns, c)
		}
	}
	want := []call{{"fn", 5, 13, false, experiment.Key{}}, {"fn", 13, 21, false, experiment.Key{}}, {"fn", 21, 25, false, experiment.Key{}}, {"fn", 5, 25, true, experiment.Key{}}}
	if !slices.Equal(fns, want) {
		t.Errorf("Fn calls = %v, want %v", fns, want)
	}
	log = nil
	e.Every, e.Keep.Series = 0, true
	r = runSweep(t, e)[0]
	if r.Ticks != nil || r.Series != nil || r.Final[0].Value != 25 {
		t.Errorf("Every 0: Ticks %v Series %v Final %v, want nil, nil and the reading at 25", r.Ticks, r.Series, r.Final)
	}
}

// TestRun_SpanCalls verifies the calls of Begin and Fn: with Every 0 one of
// each with Span true; with Every 8 a Begin with Span false for each window and
// one with Span true at the transient, and an Fn with Span true last.
//
// Requirement: FR-EXP-016
func TestRun_SpanCalls(t *testing.T) {
	short(t)
	var log []call
	e := packSpec(t)
	e.Ticks, e.Transient = 17, 1
	e.Measures = []experiment.Measure{recorder(&log)}
	runSweep(t, e)
	if want := []call{{"begin", 1, 0, true, experiment.Key{}}, {"fn", 1, 17, true, experiment.Key{}}}; !slices.Equal(log, want) {
		t.Errorf("Every 0 calls = %v, want %v", log, want)
	}
	log = nil
	e.Every = 8
	runSweep(t, e)
	want := []call{
		{"begin", 1, 0, false, experiment.Key{}}, {"begin", 1, 0, true, experiment.Key{}},
		{"fn", 1, 9, false, experiment.Key{}}, {"begin", 9, 0, false, experiment.Key{}},
		{"fn", 9, 17, false, experiment.Key{}}, {"fn", 1, 17, true, experiment.Key{}},
	}
	if !slices.Equal(log, want) {
		t.Errorf("Every 8 calls = %v, want %v", log, want)
	}
}

// TestRun_BoundaryOrderCloseStepsOpen verifies the order at a boundary: the
// window that ends there is closed, then the steps of the tick are applied,
// then the next window opens (so its opening snapshot is taken after the steps).
//
// The state half of FR-EXP-019 (a call that adds a pending total) needs a call
// of INT, which plan 2-17 builds; the order of the hooks is what this tests.
//
// Requirement: FR-EXP-019
func TestRun_BoundaryOrderCloseStepsOpen(t *testing.T) {
	short(t)
	var log []call
	e := packSpec(t)
	e.Ticks, e.Every = 4, 2
	e.Measures = []experiment.Measure{recorder(&log)}
	e.Script = []experiment.Step{{Tick: 2, Do: func(*economysim.World) error {
		log = append(log, call{kind: "do"})
		return nil
	}}}
	runSweep(t, e)
	var kinds []string
	for _, c := range log {
		if c.kind == "do" || (c.kind == "fn" && c.to == 2) || (c.kind == "begin" && c.from == 2) {
			kinds = append(kinds, c.kind)
		}
	}
	if want := []string{"fn", "do", "begin"}; !slices.Equal(kinds, want) {
		t.Errorf("hooks at tick 2 = %v, want %v: close, steps, open", kinds, want)
	}
}

// TestRun_CriterionOnFinalReading verifies the three operators on the final
// reading, that Pass is in criterion order, and that a failed run or an
// undefined reading does not pass.
//
// Requirement: FR-EXP-015
func TestRun_CriterionOnFinalReading(t *testing.T) {
	short(t)
	e := packSpec(t) // households is 600 at the end
	crit := func(op experiment.Op, a, b int64) experiment.Criterion {
		return experiment.Criterion{Name: fmt.Sprintf("c%d_%d_%d", op, a, b), Measure: "households", Op: op, A: a, B: b}
	}
	e.Criteria = []experiment.Criterion{
		crit(experiment.AtMost, 600, 0), crit(experiment.AtMost, 599, 0),
		crit(experiment.AtLeast, 600, 0), crit(experiment.AtLeast, 601, 0),
		crit(experiment.Between, 600, 600), crit(experiment.Between, 601, 700),
	}
	got := runSweep(t, e)[0].Pass
	if want := []bool{true, false, true, false, true, false}; !slices.Equal(got, want) {
		t.Errorf("Pass = %v, want %v", got, want)
	}
	e.Measures = []experiment.Measure{{Name: "households", Fn: func(experiment.Sample) (int64, bool, error) { return 0, false, nil }}}
	if got := runSweep(t, e)[0].Pass; slices.Contains(got, true) {
		t.Errorf("Pass of an undefined reading = %v, want all false", got)
	}
	e.Script = []experiment.Step{{Tick: 1, Do: func(*economysim.World) error { return errors.New("x") }}}
	if r := runSweep(t, e)[0]; slices.Contains(r.Pass, true) || len(r.Pass) != 6 {
		t.Errorf("Pass of a failed run = %v, want 6 false", r.Pass)
	}
}

// countingObserver counts the events it sees.
type countingObserver struct{ n *int }

func (c countingObserver) Observe(economysim.Event) { *c.n++ }

// TestRun_NoObserverWithoutAnObserverMeasure verifies that a sweep whose
// measures have no Observer registers none (the world constructs no event:
// fewer allocations than with one), and that the two end at one digest.
//
// Requirement: FR-EXP-017, FR-EXP-065
func TestRun_NoObserverWithoutAnObserverMeasure(t *testing.T) {
	short(t)
	events := 0
	plain := packSpec(t)
	watched := packSpec(t)
	watched.Measures = append(slices.Clone(watched.Measures), experiment.Measure{
		Name:     "events",
		Observer: func(experiment.Key) economysim.Observer { return countingObserver{&events} },
		Fn:       func(experiment.Sample) (int64, bool, error) { return int64(events), true, nil },
	})
	a, b := runSweep(t, plain), runSweep(t, watched)
	if a[0].Digest != b[0].Digest {
		t.Errorf("digest with an observer = %x, without = %x, want equal (FR-EXP-065)", b[0].Digest, a[0].Digest)
	}
	if events == 0 {
		t.Fatal("the observer measure saw no event")
	}
	seen := events
	runSweep(t, plain)
	if events != seen {
		t.Errorf("a sweep with no observer measure delivered %d events to an observer", events-seen)
	}
	allocs := func(e experiment.Spec) float64 {
		return testing.AllocsPerRun(1, func() { runSweep(t, e) })
	}
	if p, w := allocs(plain), allocs(watched); p >= w {
		t.Errorf("allocations without an observer = %v, with one = %v, want fewer without", p, w)
	}
}

// TestRun_HarnessSeedWins verifies that the harness's seed is last: an Options
// that returns WithSeed(99) does not change the digest of seed 1.
//
// Requirement: FR-EXP-018
func TestRun_HarnessSeedWins(t *testing.T) {
	short(t)
	e := packSpec(t)
	want := runSweep(t, e)[0].Digest
	e.Options = func(experiment.Key) []economysim.Option { return []economysim.Option{economysim.WithSeed(99)} }
	if got := runSweep(t, e)[0].Digest; got != want {
		t.Errorf("digest with Options WithSeed(99) = %x, want %x of seed 1", got, want)
	}
}

// TestRun_ReplayIsOneWithSeedCall verifies that a run is reproduced by its
// documents and one WithSeed call: the digest of a sweep record equals that of
// NewWorld on the loader's Config, Run and Checkpoint, and two experiments of
// another Seed0 share no digest.
//
// Requirement: FR-EXP-010, FR-EXP-011
func TestRun_ReplayIsOneWithSeedCall(t *testing.T) {
	short(t)
	e := packSpec(t)
	e.Replicas = 2
	e.Factors = []experiment.Factor{{Levels: []experiment.Level{
		{Changes: []experiment.Change{{Path: "world.belief_step", Value: 150}}},
		{Changes: []experiment.Change{{Path: "world.belief_step", Value: 250}}},
	}}}
	recs := runSweep(t, e)
	for _, r := range recs {
		over := fmt.Sprintf("[world]\nbelief_step = %d\n", []int{150, 250}[r.Key.Point])
		cfg := parseDocs(t, append(slices.Clone(e.Base), []byte(over))...)
		w, err := economysim.NewWorld(cfg, economysim.WithSeed(r.Seed))
		if err != nil {
			t.Fatalf("NewWorld(key %+v) error = %v", r.Key, err)
		}
		if _, err := w.Run(context.Background(), e.Ticks); err != nil {
			t.Fatalf("Run(key %+v) error = %v", r.Key, err)
		}
		if got := digestOf(t, w); got != r.Digest {
			t.Errorf("key %+v: replay digest %x, want the record's %x", r.Key, got, r.Digest)
		}
	}
	e.Seed0 = 1001
	for i, r := range runSweep(t, e) {
		if r.Digest == recs[i].Digest {
			t.Errorf("key %+v: Seed0 1001 gave the digest of Seed0 1", r.Key)
		}
	}
}

// TestRun_FailureCodes verifies the status of a world that fails a tick
// (invalid_trade, at the tick of the step that set the flag), a nil extension
// (world), a Stateful whose SaveState errors (checkpoint), and that the partial
// series of a failure is kept with Keep.Series and the digest is not.
//
// Requirement: FR-EXP-050, FR-EXP-051
func TestRun_FailureCodes(t *testing.T) {
	short(t)
	on := false
	e := packSpec(t)
	e.Ticks, e.Every, e.Keep.Series = 4, 1, true
	e.Base = append(slices.Clone(e.Base), faultyDoc)
	e.Options = faultyOn(&on)
	e.Script = []experiment.Step{{Tick: 3, Do: func(*economysim.World) error { on = true; return nil }}}
	r := runSweep(t, e)[0]
	if r.Status != experiment.StatusInvalidTrade || r.FailTick != 3 || r.Digest != [32]byte{} || r.Final[0].OK || r.Hash == [32]byte{} {
		t.Errorf("faulty clearing record = %+v, want invalid_trade at tick 3, no digest, undefined, a hash", r)
	}
	if !slices.Equal(r.Ticks, []int64{1, 2, 3}) || len(r.Series) != 3 {
		t.Errorf("partial series ticks %v rows %d, want the windows 1, 2, 3 that closed before the failure", r.Ticks, len(r.Series))
	}
	e.Keep.Series = false
	if r := runSweep(t, e)[0]; r.Series != nil || r.Ticks != nil {
		t.Errorf("without Keep.Series a failed run keeps %v %v, want no reading", r.Ticks, r.Series)
	}

	e = packSpec(t)
	e.Base = append(slices.Clone(e.Base), []byte("[world]\nclearing = \"nobody\"\n"))
	e.Options = func(experiment.Key) []economysim.Option {
		return []economysim.Option{economysim.WithClearing("nobody", nil)}
	}
	if r := runSweep(t, e)[0]; r.Status != experiment.StatusWorld {
		t.Errorf("nil clearing record = %+v, want world (ErrNilExtension)", r)
	}

	e = packSpec(t)
	e.Base = append(slices.Clone(e.Base), []byte("[world]\nclearing = \"stateful\"\n"))
	e.Options = func(experiment.Key) []economysim.Option {
		return []economysim.Option{economysim.WithClearing("stateful", failingSave{})}
	}
	if r := runSweep(t, e)[0]; r.Status != experiment.StatusCheckpoint || r.FailTick != e.Ticks || r.Digest != [32]byte{} || r.Final[0].OK {
		t.Errorf("failing SaveState record = %+v, want checkpoint at the last tick with undefined readings (the whole span did not complete)", r)
	}
}

// failingSave is a Stateful clearing whose SaveState fails: a checkpoint of its
// world is an error.
type failingSave struct{ economysim.AverageClearing }

func (failingSave) SaveState() (uint32, []byte, error) { return 0, nil, errors.New("cannot save") }
func (failingSave) LoadState(uint32, []byte) error     { return nil }

// TestRun_MeasureErrorAndRange verifies that a measure error leaves its reading
// undefined with the first error in Detail and the run goes on, and that a
// reading beyond 2^40 is undefined with the ErrRange text.
//
// Requirement: FR-EXP-053, FR-EXP-047
func TestRun_MeasureErrorAndRange(t *testing.T) {
	short(t)
	e := packSpec(t)
	e.Measures = []experiment.Measure{
		{Name: "bad", Fn: func(experiment.Sample) (int64, bool, error) { return 5, true, errors.New("no reading") }},
		{Name: "again", Fn: func(experiment.Sample) (int64, bool, error) { return 0, false, errors.New("second") }},
		{Name: "huge", Fn: func(experiment.Sample) (int64, bool, error) { return experiment.MaxReading + 1, true, nil }},
		{Name: "edge", Fn: func(experiment.Sample) (int64, bool, error) { return -experiment.MaxReading, true, nil }},
	}
	r := runSweep(t, e)[0]
	if r.Status != experiment.StatusOK || r.Digest == [32]byte{} {
		t.Fatalf("record = %+v, want the run to go on to ok", r)
	}
	if r.Final[0].OK || r.Final[1].OK || r.Final[2].OK || !r.Final[3].OK {
		t.Errorf("Final = %+v, want the first three undefined and the edge value defined", r.Final)
	}
	if !strings.Contains(r.Detail, `"bad"`) || strings.Contains(r.Detail, "second") {
		t.Errorf("Detail = %q, want the first error only", r.Detail)
	}
	e.Measures = e.Measures[2:3]
	if d := runSweep(t, e)[0].Detail; !strings.Contains(d, experiment.ErrRange.Error()) {
		t.Errorf("Detail = %q, want the ErrRange text", d)
	}
}

// TestRun_CancelDiscardsTheRunInFlight verifies that a context cancelled from a
// measure's Fn stops the run at the next tick with no record and the wrapped
// context error, and that it is not a failure of the next unit.
//
// Requirement: FR-EXP-020, FR-EXP-052
func TestRun_CancelDiscardsTheRunInFlight(t *testing.T) {
	short(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e := packSpec(t)
	e.Ticks, e.Every = 4, 2
	e.Measures = []experiment.Measure{{Name: "m", Fn: func(experiment.Sample) (int64, bool, error) {
		cancel()
		return 1, true, nil
	}}}
	units, err := experiment.Plan(e)
	if err != nil {
		t.Fatal(err)
	}
	recs, err := experiment.RunUnit(ctx, e, units[0])
	if recs != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("RunUnit() = %v, %v, want no records and context.Canceled", recs, err)
	}
	if _, err := experiment.RunUnit(ctx, e, units[0]); !errors.Is(err, context.Canceled) {
		t.Errorf("RunUnit() on a cancelled context = %v, want context.Canceled before the unit", err)
	}
}

// TestRunUnit_ParallelEqualsRun verifies the contract of 5.3 for a consumer's
// own runner: the units of Plan run on eight goroutines give the records of the
// sequential run, and the experiment is not written (a deep copy before and
// after is equal). Run under -race this also finds a shared variable.
//
// Requirement: FR-EXP-021, FR-EXP-023
func TestRunUnit_ParallelEqualsRun(t *testing.T) {
	short(t)
	e := packSpec(t)
	e.Replicas = 2
	e.Factors = []experiment.Factor{twoLevelsOf("world.belief_step", 150, 200)}
	before := fmt.Sprintf("%+v", e.Factors) + fmt.Sprint(e.Base[0][:50])
	want := runSweep(t, e)
	units, _ := experiment.Plan(e)
	got := make([][]experiment.Record, len(units))
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := w; i < len(units); i += 8 {
				recs, err := experiment.RunUnit(context.Background(), e, units[i])
				if err != nil {
					t.Errorf("RunUnit(%+v) error = %v", units[i], err)
				}
				got[i] = recs
			}
		}()
	}
	wg.Wait()
	var flat []experiment.Record
	for _, g := range got {
		flat = append(flat, g...)
	}
	if !reflect.DeepEqual(flat, want) {
		t.Errorf("records of eight goroutines differ from the sequential run")
	}
	if after := fmt.Sprintf("%+v", e.Factors) + fmt.Sprint(e.Base[0][:50]); after != before {
		t.Errorf("the experiment changed during the runs")
	}
}

// twoLevelsOf is an explicit factor setting one path to two values.
func twoLevelsOf(path string, a, b int64) experiment.Factor {
	return experiment.Factor{Levels: []experiment.Level{
		{Changes: []experiment.Change{{Path: path, Value: a}}},
		{Changes: []experiment.Change{{Path: path, Value: b}}},
	}}
}

// TestExecuteUnit_SkipsBaseAndPrepare verifies that ExecuteUnit uses the Config
// it is given: a Spec with no Base, factors that would not resolve and a
// Prepare that fails the test give the record of a direct run of the Config.
//
// Requirement: FR-EXP-001
func TestExecuteUnit_SkipsBaseAndPrepare(t *testing.T) {
	short(t)
	e := packSpec(t)
	want := runSweep(t, e)[0]
	cfg := parseDocs(t, e.Base...)
	e.Base = nil
	e.Factors = []experiment.Factor{twoLevelsOf("world.cost_flor", 1, 2)}
	e.Prepare = func(context.Context, economysim.Config) (economysim.Config, error) {
		t.Error("ExecuteUnit called Prepare")
		return economysim.Config{}, nil
	}
	units, err := experiment.Plan(e)
	if err != nil {
		t.Fatal(err)
	}
	recs, err := experiment.ExecuteUnit(context.Background(), cfg, e, units[0])
	if err != nil || len(recs) != 1 || recs[0].Digest != want.Digest || recs[0].Status != experiment.StatusOK {
		t.Errorf("ExecuteUnit() = %+v, %v, want the digest %x of the same Config run directly", recs, err, want.Digest)
	}
	if _, err := experiment.ExecuteUnit(context.Background(), cfg, e, experiment.Unit{Point: 9}); !errors.Is(err, experiment.ErrUnknownRun) {
		t.Errorf("ExecuteUnit() of a unit outside the plan = %v, want ErrUnknownRun", err)
	}
}

// TestCheck_ResolvesWithoutTicks verifies that Check calls no Options and runs
// no tick (a measure that fails the test if called), and reports a point that
// does not resolve; and RangedNarrowerThanDesign: a spread of 5 over a range of
// 3 values warns at the factor.
//
// Requirement: FR-EXP-070, FR-EXP-071
func TestCheck_ResolvesWithoutTicks(t *testing.T) {
	short(t)
	e := packSpec(t)
	e.Options = func(experiment.Key) []economysim.Option { t.Error("Check called Options"); return nil }
	e.Measures = []experiment.Measure{{Name: "m", Fn: func(experiment.Sample) (int64, bool, error) { t.Error("Check ran a window"); return 0, true, nil }}}
	e.Design, e.DesignN = experiment.Spread, 5
	e.Factors = []experiment.Factor{
		{Path: "world.belief_step", Min: 150, Max: 152},
		twoLevelsOf("world.cost_flor", 1, 2),
	}
	cs, err := experiment.Check(context.Background(), e)
	if err != nil || len(cs) != 5 {
		t.Fatalf("Check() = %v, %v, want 5 points", cs, err)
	}
	for _, c := range cs {
		if c.Status != experiment.StatusUnresolved || c.Detail == "" {
			t.Errorf("point %d = %+v, want unresolved with a detail (cost_flor is no key)", c.Point, c)
		}
		want := []experiment.Warning{{Rule: "ranged_narrower_than_design", Path: "Factors[0]"}}
		if !slices.Equal(c.Warnings, want) {
			t.Errorf("point %d warnings = %v, want %v", c.Point, c.Warnings, want)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := experiment.Check(ctx, e); !errors.Is(err, context.Canceled) {
		t.Errorf("Check() on a cancelled context = %v, want context.Canceled", err)
	}
}
