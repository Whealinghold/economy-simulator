package main

import (
	"context"
	"fmt"

	economysim "github.com/Whealinghold/economy-sim"
)

// session is a world being stepped, with its tracker. It can go back by
// building the world again from the seed and stepping to the tick: the
// simulator is deterministic, so the same tick has the same state.
type session struct {
	o  options
	w  *economysim.World
	tr *tracker

	// win is the event window; last holds the counters of the last complete
	// window, ending before tick lastAt.
	win    int64
	last   []string
	lastAt int64
}

// newSession builds the world of o.
func newSession(o options) (*session, error) {
	s := &session{o: o, win: o.win}
	return s, s.reset()
}

// reset builds the world again from the layers and the seed, at tick 0.
func (s *session) reset() error {
	cfg, err := s.o.load()
	if err != nil {
		return err
	}
	tr := newTracker()
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(s.o.seed), economysim.WithObserver(tr))
	if err != nil {
		return err
	}
	s.w, s.tr = w, tr
	s.last, s.lastAt = nil, 0
	return nil
}

// tick is the tick the next step will run.
func (s *session) tick() int64 { return s.w.Tick() }

// advance steps n ticks, closing an event window at each multiple of the
// window length.
func (s *session) advance(n int64) error {
	for ; n > 0; n-- {
		if _, err := s.w.Run(context.Background(), 1); err != nil {
			return fmt.Errorf("tick %d: %w", s.tick(), err)
		}
		if t := s.tick(); t%s.win == 0 {
			s.last, s.lastAt = s.tr.flush(nil), t
		}
	}
	return nil
}

// seek moves to tick t: forward by stepping, backward by starting again.
func (s *session) seek(t int64) error {
	t = max(t, 0)
	if t < s.tick() {
		if err := s.reset(); err != nil {
			return err
		}
	}
	return s.advance(t - s.tick())
}

// view reads the snapshot at the current boundary.
func (s *session) view() (economysim.Snapshot, snapOut, error) {
	snap, err := s.w.Snapshot()
	if err != nil {
		return snap, snapOut{}, err
	}
	return snap, buildSnap(snap, s.tr), nil
}
