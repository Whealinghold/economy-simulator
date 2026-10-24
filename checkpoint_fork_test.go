package economysim_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// forkCase is one fork-at-N of the checkpoint tests (plan 2-2 D258). Every module
// plan names its test TestFork_<Module> and calls forkAtN with its overlay, so the
// comparison is written once.
type forkCase struct {
	Name string
	Cfg  economysim.Config
	// Opts returns the options of one world, fresh on every call: a stateful extension
	// is never shared between worlds (architecture.md 7.7). Nil means none.
	Opts func() []economysim.Option
	Seed uint64
	// N ticks run before the boundary, M ticks after it.
	N, M int
	// Before is a call between ticks, made on the original after its N ticks and
	// before the checkpoint (a directive, a shock). Nil means none.
	Before func(*economysim.World)
	// Reseed, when set, restores with WithReseed(*Reseed). The restored worlds then
	// differ from the original by design, so they are compared with a second restore
	// of the same checkpoint and the same salt instead.
	Reseed *uint64
	// Logs compares the log records of the worlds since the boundary, at every level.
	Logs bool
	// Inspect, when set, is called at the end with all the events the restored world
	// delivered and the events the reference delivered after the boundary.
	Inspect func(t *testing.T, restored, since []economysim.Event)
}

// forked is a world of a fork with what it recorded since the boundary.
type forked struct {
	name string
	w    *economysim.World
	rec  *recorder
	log  *captureHandler
	// evSeen and logSeen count the events and records already compared.
	evSeen, logSeen int
}

// newForked builds the world a fork case describes: a fresh world when ck is nil,
// else a restore of ck (with the salt, when given).
func newForked(t *testing.T, c forkCase, name string, ck *economysim.Checkpoint, salt *uint64) *forked {
	t.Helper()
	f := &forked{name: name, rec: &recorder{}}
	var opts []economysim.Option
	if c.Opts != nil {
		opts = c.Opts()
	}
	opts = append(opts, economysim.WithObserver(f.rec))
	if c.Logs {
		f.log = newCaptureHandler()
		opts = append(opts, economysim.WithLogger(slog.New(f.log)))
	}
	var err error
	if ck == nil {
		f.w, err = economysim.NewWorld(c.Cfg, append(opts, economysim.WithSeed(c.Seed))...)
	} else {
		if salt != nil {
			opts = append(opts, economysim.WithReseed(*salt))
		}
		f.w, err = economysim.RestoreWorld(c.Cfg, *ck, opts...)
	}
	if err != nil {
		t.Fatalf("%s: seed %d, N %d: building the %s world error = %v, want nil", c.Name, c.Seed, c.N, name, err)
	}
	return f
}

// newEvents returns the events recorded since the last call, leaving the first
// skip events of a restored world (its WorldRestored) out of the first return.
func (f *forked) newEvents(skip int) []economysim.Event {
	if f.evSeen < skip {
		f.evSeen = skip
	}
	out := f.rec.events[f.evSeen:]
	f.evSeen = len(f.rec.events)
	return out
}

// newLogs returns the records written since the last call.
func (f *forked) newLogs() []logLine {
	all := f.log.records()
	out := all[f.logSeen:]
	f.logSeen = len(all)
	return out
}

// forkBoundary checks the first event of the restored world b against the original a
// at the boundary: it is WorldRestored, with the checkpoint's tick, hash and format,
// the reseed it was given, and the snapshot of the boundary (the effective seed
// aside, which a reseed replaces).
func forkBoundary(t *testing.T, c forkCase, a, b *forked) {
	t.Helper()
	at := fmt.Sprintf("%s: seed %d, N %d", c.Name, c.Seed, c.N)
	if len(b.rec.events) == 0 {
		t.Fatalf("%s: the restored world delivered no event, want WorldRestored first", at)
	}
	ev, ok := b.rec.events[0].(economysim.WorldRestored)
	if !ok {
		t.Fatalf("%s: first event of the restored world is %T, want WorldRestored", at, b.rec.events[0])
	}
	if ev.Tick != int64(c.N) || ev.ConfigHash != a.w.ConfigHash() || ev.Format != 1 {
		t.Errorf("%s: WorldRestored = tick %d, hash %x, format %d, want tick %d, hash %x, format 1", at, ev.Tick, ev.ConfigHash, ev.Format, c.N, a.w.ConfigHash())
	}
	if want := c.Reseed != nil; ev.Reseeded != want || (want && ev.Salt != *c.Reseed) {
		t.Errorf("%s: WorldRestored reseeded = %t, salt %d, want %t, %v", at, ev.Reseeded, ev.Salt, want, c.Reseed)
	}
	want, err := a.w.Snapshot()
	if err != nil {
		t.Fatalf("%s: Snapshot() of the original error = %v, want nil", at, err)
	}
	got := ev.Snapshot
	if c.Reseed != nil {
		got.Seed = want.Seed
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s: the snapshot in WorldRestored differs from the original's at the boundary", at)
	}
}

// forkView is what the reference world shows after one Step, taken once so that a
// call with a log line (Checkpoint) is made as often on every world.
type forkView struct {
	tick int64
	snap economysim.Snapshot
	ck   []byte
	ev   []economysim.Event
	logs []logLine
}

// viewOf takes the view of f after a Step; the events and records are those since
// the last view, the first event of a restored world left out.
func viewOf(t *testing.T, c forkCase, at string, f *forked) forkView {
	t.Helper()
	if err := f.w.Audit(); err != nil {
		t.Fatalf("%s: Audit() of the %s world error = %v, want nil", at, f.name, err)
	}
	snap, err := f.w.Snapshot()
	if err != nil {
		t.Fatalf("%s: Snapshot() of the %s world error = %v, want nil", at, f.name, err)
	}
	v := forkView{tick: f.w.Tick(), snap: snap, ck: mustMarshal(t, mustCheckpoint(t, f.w)), ev: f.newEvents(1)}
	if c.Logs {
		v.logs = f.newLogs()
	}
	return v
}

// forkCompare fails if the view of a world differs from the reference's: tick,
// snapshot, checkpoint bytes, events and (with Logs) log records.
func forkCompare(t *testing.T, at string, want, got forkView, name string) {
	t.Helper()
	switch {
	case want.tick != got.tick:
		t.Fatalf("%s: Tick() of the %s world = %d, want %d", at, name, got.tick, want.tick)
	case !reflect.DeepEqual(want.snap, got.snap):
		t.Fatalf("%s: the snapshot of the %s world differs from the reference's", at, name)
	case !bytes.Equal(want.ck, got.ck):
		t.Fatalf("%s: the checkpoint of the %s world differs from the reference's", at, name)
	case !reflect.DeepEqual(want.ev, got.ev):
		t.Fatalf("%s: the events of the %s world differ from the reference's (%d and %d events)", at, name, len(got.ev), len(want.ev))
	case !reflect.DeepEqual(want.logs, got.logs):
		t.Fatalf("%s: the log records of the %s world differ from the reference's (%d and %d records)", at, name, len(got.logs), len(want.logs))
	}
}

// forkAtN runs the fork-at-N check of CHK 8.1 and 8.2: a world A runs N ticks and
// makes its call; two checkpoints of the boundary are byte-equal; B is restored from
// it and its first event is WorldRestored; C is restored from B's checkpoint (a
// restore of a restore); and for M more ticks A, B and C give, after every Step, the
// same events, snapshot, tick, audit and checkpoint bytes. With a reseed A is left
// out and a second restore of the same checkpoint and salt is the reference. A
// failure names the seed, N and the first tick that differs.
func forkAtN(t *testing.T, c forkCase) {
	t.Helper()
	where := func(tick int64) string { return fmt.Sprintf("%s: seed %d, N %d, tick %d", c.Name, c.Seed, c.N, tick) }
	a := newForked(t, c, "original", nil, nil)
	for i := range c.N {
		if err := a.w.Step(); err != nil {
			t.Fatalf("%s: Step() of the original error = %v, want nil", where(int64(i)), err)
		}
	}
	if c.Before != nil {
		c.Before(a.w)
	}
	ck := mustCheckpoint(t, a.w)
	ckBytes := mustMarshal(t, ck)
	if !bytes.Equal(ckBytes, mustMarshal(t, mustCheckpoint(t, a.w))) {
		t.Fatalf("%s: two checkpoints of one boundary differ", where(int64(c.N)))
	}
	a.newEvents(0)
	if c.Logs {
		a.newLogs()
	}

	b := newForked(t, c, "restored", &ck, c.Reseed)
	forkBoundary(t, c, a, b)
	ckB := mustCheckpoint(t, b.w)
	if c.Reseed == nil && !bytes.Equal(ckBytes, mustMarshal(t, ckB)) {
		t.Fatalf("%s: the restored world's checkpoint differs from the one it was restored from", where(int64(c.N)))
	}
	third := newForked(t, c, "restore of a restore", &ckB, nil)
	ref := a
	if c.Reseed != nil {
		ref = newForked(t, c, "second restore", &ck, c.Reseed)
	}
	for _, f := range []*forked{b, third, ref} {
		if f != a {
			f.newEvents(1)
			if c.Logs {
				f.newLogs()
			}
		}
	}
	var since []economysim.Event
	for i := range c.M {
		tick := int64(c.N + i)
		for _, f := range []*forked{ref, b, third} {
			if err := f.w.Step(); err != nil {
				t.Fatalf("%s: Step() of the %s world error = %v, want nil", where(tick), f.name, err)
			}
		}
		want := viewOf(t, c, where(tick), ref)
		forkCompare(t, where(tick), want, viewOf(t, c, where(tick), b), b.name)
		forkCompare(t, where(tick), want, viewOf(t, c, where(tick), third), third.name)
		since = append(since, want.ev...)
	}
	if c.Inspect != nil {
		c.Inspect(t, b.rec.events, since)
	}
}

// forkM is the number of ticks after the boundary of the fork matrix (plan 2-2 Phase 6).
const forkM = 30

// forkCells are the ticks N of the matrix on pack one, seed 1 (probed, see the
// arithmetic of TestFork_OneAtN).
var forkCells = []struct {
	name string
	n    int
}{
	{"before the first step", 0},
	{"after one step", 1},
	{"inside a job", 100},
	{"across a season change", 150},
	{"just after a season change", 60},
}

// TestFork_OneAtN verifies the fork-at-N of pack one of the reference chain, warm
// start, for several seeds: at N in {0, 1, inside a job, across a season change, 60}
// the original and two restores (one a restore of a restore) give the same events,
// snapshot, audit and checkpoint bytes for 30 ticks. For the cell across the season
// change it also checks that a job started before the boundary completes with the
// yield it was started with.
//
// Arithmetic: the default calendar begins its first season on day 59 and a season is
// 92 days, so the season changes at ticks 59 and 151. grow_staple lasts 60 ticks and
// yields in spring only, so at tick 100 jobs of recipe 3 are in progress (170 or
// more executions, probed) and at tick 150 they still are, one tick before the
// season changes: a restore that took the yield from the season of the completion
// tick would give 0 at tick 151 or later instead of the 10 000 it started with.
//
// Requirement: FR-CHK-050, FR-CHK-052, FR-CHK-053, FR-CHK-026
//
// Guard: N-237. The yield of a job is part of the saved job; a reader that
// recomputed it would pass every test that restores outside a season change.
func TestFork_OneAtN(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	for seed := uint64(1); seed <= uint64(invariantSeeds(4)); seed++ {
		for _, cell := range forkCells {
			t.Run(fmt.Sprintf("seed %d %s", seed, cell.name), func(t *testing.T) {
				t.Parallel()
				forkAtN(t, forkCase{
					Name: "one", Cfg: cfg, Seed: seed, N: cell.n, M: forkM,
					Inspect: func(t *testing.T, _, since []economysim.Event) {
						if cell.n != 150 {
							return
						}
						checkCarriedYield(t, seed, int64(cell.n), since)
					},
				})
			})
		}
	}
}

// checkCarriedYield fails if a job that started before tick n and completed after the
// season changed at tick 151 did not complete with the yield of 10 000 it started with.
// A run with no such job passes, so a seed whose jobs all ended earlier proves nothing
// here; the comparison of forkAtN still holds for it.
func checkCarriedYield(t *testing.T, seed uint64, n int64, since []economysim.Event) {
	t.Helper()
	for _, e := range since {
		if ev, ok := e.(economysim.RecipeExecuted); ok && ev.StartTick < n && ev.Tick >= 151 && ev.EffectiveYield != 10_000 {
			t.Errorf("seed %d: job of recipe %d started at %d completed at %d with yield %d, want 10000 (the yield it started with)", seed, ev.Recipe, ev.StartTick, ev.Tick, ev.EffectiveYield)
		}
	}
}

// TestFork_WithObserversAndLogger verifies that a restored world given a new observer
// and a new logger delivers WorldRestored first and replays no AgentSpawned or
// BoundaryFlow of its own, then the same events as the original, and writes the same
// log records (at every level, in order, with equal attributes) as the original for
// the 30 ticks after the boundary.
//
// Requirement: FR-CHK-027, FR-CHK-028, FR-CHK-043, FR-OBS-011
//
// Guard: a restore that replayed the construction events, or logged a different line
// than a world that never stopped, would double-count in an observer that adds up
// AgentSpawned and BoundaryFlow, and no other test reads the logs of a restored world.
func TestFork_WithObserversAndLogger(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	forkAtN(t, forkCase{
		Name: "one with logs", Cfg: cfg, Seed: 1, N: 100, M: forkM, Logs: true,
		Inspect: func(t *testing.T, restored, since []economysim.Event) {
			if len(since) == 0 {
				t.Fatalf("the reference delivered no event after the boundary")
			}
			for i, e := range restored[1:] {
				if _, started := e.(economysim.TickStarted); started {
					break
				}
				t.Errorf("event %d between WorldRestored and the first TickStarted is %T, want none (no AgentSpawned or BoundaryFlow is replayed)", i+1, e)
			}
			for _, e := range restored {
				if bf, ok := e.(economysim.BoundaryFlow); ok && bf.Tick < 0 {
					t.Errorf("the restored world delivered the construction BoundaryFlow (tick -1) of a world past its first step")
				}
			}
		},
	})
}
