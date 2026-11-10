package experiment

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
)

// A failure is what stopped a run: a status and the tick it is recorded at, or a
// cancellation, which is not a failure and produces no record
// (spec-experiments.md 5.4.2).
type failure struct {
	status    Status
	tick      int64
	detail    string
	cancelled bool
}

// progress is what a run has done that a branch inherits: the tick, the open
// window and span, and the readings of the windows closed. It is plain data; a
// branch copies it (clone), the snapshots and rows being read-only.
type progress struct {
	tick     int64
	winFrom  int64
	open     economysim.Snapshot
	spanOpen economysim.Snapshot
	ticks    []int64
	series   [][]Reading
	final    []Reading
	detail   string // the first measure error
}

// An engine runs one world through the boundaries of a key: it advances the
// ticks, reads the windows, applies the steps and, at the end, takes the
// digest. It holds one world and is used by one goroutine.
type engine struct {
	e      *Spec
	ctx    context.Context
	cfg    economysim.Config
	w      *economysim.World
	key    Key // the key of the world held, handed to the measures
	steps  []Step
	bounds []int64
	next   int                  // index of the first step not yet applied
	last   *economysim.Snapshot // the closing snapshot at Ticks, kept for Keep.Snapshot
	progress
}

func newEngine(ctx context.Context, e *Spec, cfg economysim.Config, w *economysim.World, key Key, steps []Step, p progress) *engine {
	g := &engine{e: e, ctx: ctx, cfg: cfg, w: w, key: key, steps: steps, bounds: e.boundaries(steps), progress: p}
	g.next, _ = slices.BinarySearchFunc(steps, p.tick, func(s Step, t int64) int { return cmp.Compare(s.Tick, t) })
	return g
}

// advance runs the world to tick to. A cancelled context is not a failure.
func (g *engine) advance(to int64) *failure {
	if to <= g.tick {
		return nil
	}
	done, err := g.w.Run(g.ctx, to-g.tick)
	if err == nil {
		g.tick = to
		return nil
	}
	if g.w.Err() == nil && g.ctx.Err() != nil {
		return &failure{cancelled: true}
	}
	st, tick := classify(stageTick, err, g.tick+done)
	return &failure{status: st, tick: tick, detail: err.Error()}
}

// snapshot takes a snapshot of the healthy world. An error here has no row of
// the table of 9.1; it is a step failure.
func (g *engine) snapshot() (economysim.Snapshot, *failure) {
	s, err := g.w.Snapshot()
	if err != nil {
		return s, &failure{status: StatusStep, tick: g.tick, detail: "snapshot: " + err.Error()}
	}
	return s, nil
}

// boundary is the whole of one boundary b: advance to it, close the window that
// ends there, apply the steps of the tick, and open the window that starts
// there (spec-experiments.md 5.1.3 step 3; FR-EXP-019).
func (g *engine) boundary(b int64) *failure {
	if f := g.advance(b); f != nil {
		return f
	}
	if f := g.closeAt(b); f != nil {
		return f
	}
	if b == g.e.Ticks {
		return nil
	}
	if f := g.stepsAt(b); f != nil {
		return f
	}
	return g.openAt(b)
}

// runAll runs the key from its current tick through every boundary.
func (g *engine) runAll() *failure {
	for _, b := range g.bounds {
		if b < g.tick {
			continue
		}
		if f := g.boundary(b); f != nil {
			return f
		}
	}
	return nil
}

// stepsAt applies the steps whose tick is b, in the order of the merged list.
func (g *engine) stepsAt(b int64) *failure {
	for g.next < len(g.steps) && g.steps[g.next].Tick == b {
		s := g.steps[g.next]
		g.next++
		if err := s.Do(g.w); err != nil {
			st, tick := classify(stageScript, err, b)
			return &failure{status: st, tick: tick, detail: fmt.Sprintf("step %q at tick %d: %v", s.Label, b, err)}
		}
	}
	return nil
}

// closeAt closes the sample window that ends at b, if any, and at Ticks reads the
// whole span, which is the final reading. With Every 0 only the span is read.
func (g *engine) closeAt(b int64) *failure {
	win := g.e.closesWindow(b)
	if len(g.e.Measures) == 0 || b <= g.e.Transient || (!win && b != g.e.Ticks) {
		return nil
	}
	closing, f := g.snapshot()
	if f != nil {
		return f
	}
	if win {
		row := g.read(Sample{Open: g.open, Close: closing, From: g.winFrom, To: b})
		if g.e.Keep.Series {
			g.ticks = append(g.ticks, b)
			g.series = append(g.series, row)
		}
		g.winFrom = b
	}
	if b == g.e.Ticks {
		g.final = g.read(Sample{Open: g.spanOpen, Close: closing, From: g.e.Transient, To: b, Span: true})
		g.last = &closing
	}
	return nil
}

// openAt opens the sample window that starts at b, if any, from a snapshot taken
// after the steps. At the end of the transient it also opens the span: with
// Every 0 the window and the span are one and Begin is called once, with Span
// true (spec-experiments.md 3.4.3).
func (g *engine) openAt(b int64) *failure {
	if len(g.e.Measures) == 0 || !g.e.opensWindow(b) {
		return nil
	}
	open, f := g.snapshot()
	if f != nil {
		return f
	}
	g.open, g.winFrom = open, b
	span := b == g.e.Transient
	if span {
		g.spanOpen = open
	}
	for _, m := range g.e.Measures {
		if m.Begin == nil {
			continue
		}
		if g.e.Every > 0 {
			m.Begin(g.key, open, b, false)
		}
		if span {
			m.Begin(g.key, open, b, true)
		}
	}
	return nil
}

// read calls Fn of every measure on the sample and returns the readings. A
// returned error or a value beyond MaxReading leaves the reading undefined and
// the first such error in the detail (FR-EXP-047, FR-EXP-053); the run goes on.
func (g *engine) read(s Sample) []Reading {
	s.Config, s.Key = g.cfg, g.key
	out := make([]Reading, len(g.e.Measures))
	for i, m := range g.e.Measures {
		v, ok, err := m.Fn(s)
		switch {
		case err != nil:
			g.noteDetail(fmt.Sprintf("measure %q window [%d, %d): %v", m.Name, s.From, s.To, err))
		case ok && (v > MaxReading || v < -MaxReading):
			g.noteDetail(fmt.Sprintf("measure %q window [%d, %d): %v: %d", m.Name, s.From, s.To, ErrRange, v))
		case ok:
			out[i] = Reading{Value: v, OK: true}
		}
	}
	return out
}

func (g *engine) noteDetail(s string) {
	if g.detail == "" {
		g.detail = s
	}
}

// finish takes the digest of the healthy world at Ticks: the last 32 bytes of
// the checkpoint's encoding, its trailer (spec-experiments.md 5.1.3 step 4), and
// with Keep.Snapshot the final snapshot.
func (g *engine) finish() (digest [32]byte, snap economysim.Snapshot, f *failure) {
	fail := func(err error) *failure {
		st, tick := classify(stageCheckpoint, err, g.tick)
		return &failure{status: st, tick: tick, detail: err.Error()}
	}
	cp, err := g.w.Checkpoint()
	if err != nil {
		return digest, snap, fail(err)
	}
	b, err := cp.MarshalBinary()
	if err != nil {
		return digest, snap, fail(err)
	}
	if len(b) < len(digest) {
		return digest, snap, fail(fmt.Errorf("checkpoint of %d bytes has no trailer", len(b)))
	}
	copy(digest[:], b[len(b)-len(digest):])
	if g.e.Keep.Snapshot && g.last != nil {
		snap = *g.last
	} else if g.e.Keep.Snapshot {
		if snap, f = g.snapshot(); f != nil {
			return digest, snap, f
		}
	}
	return digest, snap, nil
}
