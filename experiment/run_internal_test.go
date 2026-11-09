package experiment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestClassify_Table verifies every row of the table of spec-experiments.md
// 9.1: where the error came from and what it is decide the status and the tick,
// in one function.
//
// Requirement: FR-EXP-050
func TestClassify_Table(t *testing.T) {
	t.Parallel()
	other := errors.New("other")
	both := errors.Join(economysim.ErrCheckpointCorrupt, economysim.ErrAuditFailed)
	tests := []struct {
		name     string
		stage    stage
		err      error
		tick     int64
		want     Status
		wantTick int64
	}{
		{"build validation error", stageBuild, &economysim.ValidationError{Rule: other}, 7, StatusInvalidConfig, 0},
		{"build audit is tick 0", stageBuild, economysim.ErrAuditFailed, 7, StatusAudit, 0},
		{"build other error", stageBuild, other, 7, StatusWorld, 0},
		{"build nil extension", stageBuild, economysim.ErrNilExtension, 7, StatusWorld, 0},
		{"restore error", stageRestore, other, 7, StatusWorld, 7},
		{"restore wrapping an audit failure", stageRestore, both, 7, StatusWorld, 7},
		{"restore config mismatch", stageRestore, economysim.ErrCheckpointConfig, 7, StatusWorld, 7},
		{"restore option context", stageRestore, economysim.ErrOptionContext, 7, StatusWorld, 7},
		{"tick invalid trade", stageTick, economysim.ErrInvalidTrade, 7, StatusInvalidTrade, 7},
		{"tick audit", stageTick, economysim.ErrAuditFailed, 7, StatusAudit, 7},
		{"tick other", stageTick, other, 7, StatusStep, 7},
		{"tick world failed", stageTick, economysim.ErrWorldFailed, 7, StatusStep, 7},
		{"script, even an audit failure", stageScript, economysim.ErrAuditFailed, 7, StatusScript, 7},
		{"checkpoint", stageCheckpoint, economysim.ErrCheckpointExtension, 7, StatusCheckpoint, 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, tick := classify(tt.stage, tt.err, tt.tick)
			if got != tt.want || tick != tt.wantTick {
				t.Errorf("classify(%d, %v, %d) = %q, %d, want %q, %d", tt.stage, tt.err, tt.tick, got, tick, tt.want, tt.wantTick)
			}
		})
	}
}

func internalSpec(t testing.TB) Spec {
	t.Helper()
	var docs [][]byte
	for _, name := range []string{"base.toml", "one.toml", "warm-one.toml"} {
		b, err := os.ReadFile(filepath.Join("..", "testdata", "chains", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		docs = append(docs, b)
	}
	return Spec{
		Base: docs, Replicas: 3, Seed0: 1, Ticks: 2,
		Measures: []Measure{{Name: "h", Fn: func(s Sample) (int64, bool, error) { return s.Close.Households, true, nil }}},
	}
}

// TestRun_PriorCompletesByteForByte verifies that a sweep cancelled after its
// first unit, then run again with those records as the prior, gives the records
// of the uninterrupted sweep exactly, runs only the missing units, and calls the
// record function in plan order for the new records only.
//
// The Result-level form is the second half: Run of the cancelled sweep returns the
// Result of its complete units with the keys not run missing, and Run with that
// prior renders as the uninterrupted one (spec-experiments.md 5.2.3).
//
// Requirement: FR-EXP-020, FR-EXP-052
func TestRun_PriorCompletesByteForByte(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	e := internalSpec(t)
	whole, err := e.runAll(context.Background())
	if err != nil || len(whole) != 3 {
		t.Fatalf("runAll() = %d records, %v, want 3", len(whole), err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	part, err := e.runAll(ctx, WithRecordFunc(func(Record) { cancel() }))
	if !errors.Is(err, context.Canceled) || len(part) != 1 {
		t.Fatalf("cancelled runAll() = %d records, %v, want 1 and context.Canceled (the unit in flight is complete)", len(part), err)
	}
	var fresh []Key
	got, err := e.runAll(context.Background(), WithPrior(part), WithRecordFunc(func(r Record) { fresh = append(fresh, r.Key) }))
	if err != nil || !reflect.DeepEqual(got, whole) {
		t.Errorf("runAll() with the prior = %v, want the records of the uninterrupted sweep (error %v)", got, err)
	}
	if want := []Key{{Replica: 1}, {Replica: 2}}; !reflect.DeepEqual(fresh, want) {
		t.Errorf("record function saw %v, want only the new keys %v", fresh, want)
	}
	wholeRes, err := Run(context.Background(), e)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	ctx2, cancel2 := context.WithCancel(context.Background())
	partRes, err := Run(ctx2, e, WithRecordFunc(func(Record) { cancel2() }))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Run() error = %v, want context.Canceled", err)
	}
	if m := partRes.Points[0].Variants[0].Runs[1].Status; m != StatusMissing {
		t.Errorf("cancelled Run() leaves replica 1 with status %q, want missing", m)
	}
	resumed, err := Run(context.Background(), e, WithPrior(part))
	if err != nil || Render(resumed) != Render(wholeRes) {
		t.Errorf("Run() with the prior renders\n%s\nwant the uninterrupted\n%s\n(error %v)", Render(resumed), Render(wholeRes), err)
	}
}

// TestRun_OptionsOncePerWorld verifies that Options is called once for each
// world built, with the key of that world, and that an Options returning a
// fresh stateful extension each call never gives one instance to two worlds.
//
// Requirement: FR-EXP-024
func TestRun_OptionsOncePerWorld(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	e := internalSpec(t)
	seen := map[Key]int{}
	e.Options = func(k Key) []economysim.Option {
		seen[k]++
		return nil
	}
	if _, err := e.runAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 3 {
		t.Fatalf("Options saw keys %v, want the 3 keys of the plan", seen)
	}
	for k, n := range seen {
		if n != 1 {
			t.Errorf("Options(%+v) called %d times, want once", k, n)
		}
	}
}

// TestRun_HarnessWorkDoesNotGrowWithAgentsOrTicks verifies a counted operation
// (testing.md 8.2): the calls of Begin, Fn, Options and Observer are equal for
// a world of 1 household per agent and of 4, and for 6 and 12 ticks at two
// windows each. The harness does work in proportion to windows and runs, not to
// agents or ticks.
//
// Requirement: NFR-EXP-003
func TestRun_HarnessWorkDoesNotGrowWithAgentsOrTicks(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	count := func(scale int64, ticks int64) [4]int {
		var n [4]int
		e := internalSpec(t)
		e.Replicas, e.Ticks, e.Every = 1, ticks, ticks/2
		e.Factors = []Factor{{Levels: []Level{
			{Changes: []Change{{Path: "world.scale", Value: scale}}},
			{Changes: []Change{{Path: "world.scale", Value: scale}}},
		}}}
		e.Options = func(Key) []economysim.Option { n[2]++; return nil }
		e.Measures = []Measure{{
			Name:     "h",
			Begin:    func(Key, economysim.Snapshot, int64, bool) { n[0]++ },
			Observer: func(Key) economysim.Observer { n[3]++; return economysim.ObserverFunc(func(economysim.Event) {}) },
			Fn:       func(Sample) (int64, bool, error) { n[1]++; return 0, true, nil },
		}}
		recs, err := e.runAll(context.Background())
		if err != nil || recs[0].Status != StatusOK {
			t.Fatalf("scale %d ticks %d: %v, %+v", scale, ticks, err, recs)
		}
		return n
	}
	base := count(1, 6)
	if grown := count(4, 6); grown != base {
		t.Errorf("calls with scale 4 = %v, want %v of scale 1", grown, base)
	}
	if longer := count(1, 12); longer != base {
		t.Errorf("calls with 12 ticks = %v, want %v of 6", longer, base)
	}
}

// BenchmarkRunUnit_HarnessShare reports the allocations of one unit of two
// ticks over pack one, for the trend; it asserts nothing (N-015).
func BenchmarkRunUnit_HarnessShare(b *testing.B) {
	e := internalSpec(b)
	units, err := Plan(e)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := RunUnit(context.Background(), e, units[0]); err != nil {
			b.Fatal(err)
		}
	}
}

// TestRunAll_RefusesBadPrior verifies that a prior record whose key is outside the
// plan, or that repeats a key with different contents, is refused as Assemble
// refuses it, not dropped or resolved to the last one.
//
// Requirement: FR-EXP-020
func TestRunAll_RefusesBadPrior(t *testing.T) {
	t.Parallel()
	e := internalSpec(t)
	a := Record{Key: Key{Replica: 0}, Seed: 1}
	b := Record{Key: Key{Replica: 0}, Seed: 2}
	tests := []struct {
		name  string
		prior []Record
		want  error
	}{
		{"outside the plan", []Record{{Key: Key{Point: 9}}}, ErrUnknownRun},
		{"twice, different", []Record{a, b}, ErrDuplicateRun},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := e.runAll(context.Background(), WithPrior(tt.prior)); !errors.Is(err, tt.want) {
				t.Errorf("runAll(prior %s) error = %v, want %v", tt.name, err, tt.want)
			}
		})
	}
}

// TestRunAll_CancelKeepsLaterPrior verifies that a sweep cancelled before a unit
// whose records the caller supplied still returns those records, so saving the
// partial result as the next prior loses nothing.
//
// Requirement: FR-EXP-020
func TestRunAll_CancelKeepsLaterPrior(t *testing.T) {
	t.Parallel()
	e := internalSpec(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	later := e.baseRecord(Key{Replica: 2})
	got, err := e.runAll(ctx, WithPrior([]Record{later}))
	if !errors.Is(err, context.Canceled) || len(got) != 1 || got[0].Key != later.Key {
		t.Errorf("runAll(cancelled, prior of replica 2) = %v, %v, want that record and context.Canceled", got, err)
	}
}
