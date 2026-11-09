package economysim

import (
	"encoding/binary"
	"log/slog"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/rumour"
)

// TestCheckpoint_InformationRoundTrip verifies the bytes of section 16 against an
// expectation written from the markets' own reports, one per tick, and that a
// restore from the checkpoint holds the same record and goes on to make the same
// rumours: the section is the sources times the counts and the entries, the
// ticks run capped at the depth, and nothing derived.
//
// Arithmetic: with a horizon of 10, a world at tick 7 holds 7 observations per
// source and one at tick 12 holds the last 10 (ticks 2 to 11); each observation is
// the three uvarints Price, Mean, Age, after one uvarint count, for the 5 sources.
//
// Requirement: FR-INF-029, FR-INF-030
func TestCheckpoint_InformationRoundTrip(t *testing.T) {
	t.Parallel()
	const depth = 10
	for _, ticks := range []int{0, 7, 12} {
		w, s := newInfoWorld(t, func(c *Config) { c.Modules.Information.RumourMaxAge = depth })
		hist := make([][]rumour.Obs, len(w.info.sources))
		for range ticks {
			step(t, w, 1)
			for i, src := range w.info.sources {
				rep := &w.markets.reports[src.market]
				mean, _ := w.markets.windows[src.market].Mean()
				hist[i] = append(hist[i], rumour.Obs{Price: int64(rep.Price), Mean: mean, Age: rep.Age})
			}
		}
		var want []byte
		for i := range hist {
			held := hist[i][max(0, len(hist[i])-depth):]
			want = binary.AppendUvarint(want, uint64(len(held)))
			for _, o := range held {
				for _, v := range []int64{o.Price, o.Mean, o.Age} {
					want = binary.AppendUvarint(want, uint64(v))
				}
			}
		}
		cw := codec.NewWriter(nil)
		if err := w.saveInformation(cw); err != nil {
			t.Fatalf("ticks %d: saveInformation() = %v, want nil", ticks, err)
		}
		if got := cw.Out(); !slices.Equal(got, want) {
			t.Errorf("ticks %d: section 16 = % x, want % x", ticks, got, want)
		}

		ck, err := w.Checkpoint()
		if err != nil {
			t.Fatalf("ticks %d: Checkpoint() = %v, want nil", ticks, err)
		}
		s2 := &infoScript{}
		r, err := RestoreWorld(w.Config(), ck, WithStrategy("script", s2))
		if err != nil {
			t.Fatalf("ticks %d: RestoreWorld() = %v, want nil", ticks, err)
		}
		for i := range w.info.sources {
			if a, b := w.info.rec.Entries(nil, i), r.info.rec.Entries(nil, i); !slices.Equal(a, b) {
				t.Errorf("ticks %d source %d: restored record = %v, want %v", ticks, i, b, a)
			}
		}
		step(t, w, 4)
		step(t, r, 4)
		if a, b := lastTrades(s, ticks), s2.trades; len(a) == 0 || !viewsEqual(a, b) {
			t.Errorf("ticks %d: the restored world's views after the boundary differ from the original's (%d and %d views)", ticks, len(a), len(b))
		}
	}
}

// lastTrades returns the views of the script from a tick on.
func lastTrades(s *infoScript, from int) []seenView {
	return slices.DeleteFunc(slices.Clone(s.trades), func(v seenView) bool { return v.tick < int64(from) })
}

// viewsEqual compares two lists of recorded views, rumours included.
func viewsEqual(a, b []seenView) bool {
	return slices.EqualFunc(a, b, func(x, y seenView) bool {
		return x.agent == y.agent && x.region == y.region && x.tick == y.tick && x.weight == y.weight && slices.Equal(x.rumours, y.rumours)
	})
}

// TestCheckpoint_InformationNoSources verifies that a world with the module on and
// no source (no group, so no market is active) restores and goes on: the record has
// no cell to load, only the ticks run.
//
// Requirement: FR-INF-029
func TestCheckpoint_InformationNoSources(t *testing.T) {
	t.Parallel()
	w, _ := newInfoWorld(t, func(c *Config) { c.Groups = nil })
	if len(w.info.sources) != 0 {
		t.Fatalf("sources = %d, want none", len(w.info.sources))
	}
	step(t, w, 5)
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() = %v, want nil", err)
	}
	r, err := RestoreWorld(w.Config(), ck, WithStrategy("script", &infoScript{}))
	if err != nil {
		t.Fatalf("RestoreWorld() = %v, want nil", err)
	}
	step(t, r, 3)
	if h := r.info.rec.Held(); h != 8 {
		t.Errorf("restored record holds %d ticks after 3 more, want 8", h)
	}
}

// TestHashWorld_Information verifies that the state hash of a world with the
// module on does not depend on how the world was driven or watched: a bare world, a
// world with an observer and a logger and a world driven through the positions one
// by one hash alike. The scratch of the module (the view buffer, the tally) is left
// out of the walk. (A restored world does not hash like one that ran on, with the
// module off as well: that is not this module's.)
//
// Guard: a scratch field in the walk would make an observed world, whose tally is
// counted, differ from a bare one, and the determinism tests that compare state
// hashes would fail on a world that is the same.
func TestHashWorld_Information(t *testing.T) {
	t.Parallel()
	build := func(opts ...Option) *World {
		w, _ := newInfoWorld(t, func(c *Config) { c.Modules.Information.RumourErrorBP = 500 }, opts...)
		return w
	}
	bare := build()
	watched := build(WithObserver(&evLog{}), WithLogger(slog.New(newCapture())))
	replayed := build()
	step(t, bare, 12)
	step(t, watched, 12)
	for range 12 {
		if err := replayStep(replayed, func(string) {}); err != nil {
			t.Fatalf("replayStep() = %v, want nil", err)
		}
	}
	want := hashWorld(bare)
	for name, w := range map[string]*World{"observed": watched, "replayed": replayed} {
		if got := hashWorld(w); got != want {
			t.Errorf("state hash of the %s world = %x, want the bare world's %x", name, got, want)
		}
	}
}
