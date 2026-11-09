package economysim_test

import (
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// s49Pack is pack two with the information overlay, every role that has a group
// informed (the overlay) and following the strategy of the given name; errorBP edits
// the error. Without the module (off) the roles are uninformed and the table is gone.
func s49Pack(t testing.TB, errorBP int64, strategy string, off bool) economysim.Config {
	t.Helper()
	cfg := loadValidPack(t, packInformation("two")...)
	if off {
		cfg.Modules.Information = nil
		for i := range cfg.Roles {
			cfg.Roles[i].Informed = false
		}
	} else {
		cfg.Modules.Information.RumourErrorBP = errorBP
	}
	for i := range cfg.Roles {
		cfg.Roles[i].Strategy = strategy
	}
	return cfg
}

// TestScenarioS49_ForkAndStream runs scenario S49 (INF-S4, plan 2-13 D478) on pack two
// with the information overlay. First, with an error of 500 and a strategy that moves
// its limits by the rumours it is shown (so the events depend on the record and its
// draws), the world is checkpointed at tick 80 and resumed for 120 more: the events,
// snapshot, audit and checkpoint bytes of the restore equal the uninterrupted run's
// (forkAtN). Second, with an error of 0 the module takes no draw: the events and
// snapshot of 30 ticks, for seeds 1 to scenarioSeedCount(), equal those of the same
// world with the module off, whose strategy ignores rumours (a draw would move the
// later keyed streams' consumers, and the random source is reseeded at every site, so
// the events are the evidence); with an error of 500 the views are the same and only the
// price and the mean differ, and RumoursServed counts the candidates that took a draw.
//
// Requirement: FR-INF-014, FR-INF-029, FR-INF-046, FR-INF-047
//
// Guard: a draw taken without an error, or a record that a restore does not rebuild
// exactly, would break the reproduction of every run that has the module on.
func TestScenarioS49_ForkAndStream(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Run("fork at 80 resumed", func(t *testing.T) {
		t.Parallel()
		forkAtN(t, forkCase{
			Name: "s49", Cfg: s49Pack(t, 500, "reader", false), Seed: 1, N: 80, M: 120, Logs: true,
			Opts: func() []economysim.Option {
				return []economysim.Option{economysim.WithStrategy("reader", rumourReader{})}
			},
		})
	})
	rep := newPackScenReport("two", "S49 the stream with the module on and the fork (INF-S4)")
	for seed := uint64(1); seed <= uint64(scenarioSeedCount()); seed++ {
		build := func(cfg economysim.Config, rec *recordingStrategy) (*economysim.World, *recorder) {
			r := &recorder{}
			w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithObserver(r), economysim.WithStrategy(infoName, rec))
			if err != nil {
				t.Fatalf("seed %d: NewWorld() error = %v, want nil", seed, err)
			}
			return w, r
		}
		offS, quietS, noisyS := &recordingStrategy{}, &recordingStrategy{}, &recordingStrategy{}
		off, offRec := build(s49Pack(t, 0, infoName, true), offS)
		quiet, quietRec := build(s49Pack(t, 0, infoName, false), quietS)
		noisy, noisyRec := build(s49Pack(t, 500, infoName, false), noisyS)
		for tick := 0; tick < 30; tick++ {
			for _, w := range []*economysim.World{off, quiet, noisy} {
				if err := w.Step(); err != nil {
					t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, tick, err)
				}
			}
		}
		var candidates int64
		for _, e := range noisyRec.events {
			if r, ok := e.(economysim.RumoursServed); ok {
				candidates += r.Offered + r.NoPath + r.TooOld + r.NotYet
			}
		}
		// The rumours of the two module-on worlds are the same views at the same ticks;
		// with an error they differ in the price and in nothing else.
		var priceDiff, otherDiff int64
		if len(quietS.seen) != len(noisyS.seen) {
			t.Fatalf("seed %d: the module-on worlds saw %d and %d views, want equal", seed, len(quietS.seen), len(noisyS.seen))
		}
		for i := range quietS.seen {
			if len(quietS.seen[i].rumours) != len(noisyS.seen[i].rumours) {
				t.Fatalf("seed %d: view %d holds %d and %d rumours, want equal", seed, i, len(quietS.seen[i].rumours), len(noisyS.seen[i].rumours))
			}
			for j, q := range quietS.seen[i].rumours {
				n := noisyS.seen[i].rumours[j]
				if q.Price != n.Price || q.Mean != n.Mean {
					priceDiff++
				}
				q.Price, q.Mean, n.Price, n.Mean = 0, 0, 0, 0
				if q != n {
					otherDiff++
				}
			}
		}
		rep.num(seed, "s49.candidates_that_took_a_draw", candidates)
		rep.num(seed, "s49.rumours_blurred", priceDiff)
		infoCheck(t, rep, seed, "s49.no_error_no_draw", sameWorld(t, off, offRec, quiet, quietRec), "the world with an error of 0 differs from the module-off world")
		infoCheck(t, rep, seed, "s49.error_draws_and_blurs_only_the_price", candidates > 0 && priceDiff > 0 && otherDiff == 0 && len(quietS.seen) == len(noisyS.seen),
			"an error of 500 took no draw, or changed more than the price and the mean")
	}
	rep.flush(t)
}

// sameWorld reports whether two worlds at the same tick have the same events, apart
// from the creation event and RumoursServed that only the module-on world emits, and
// the same snapshot, apart from the hash and the list of modules.
func sameWorld(t testing.TB, a *economysim.World, ra *recorder, b *economysim.World, rb *recorder) bool {
	t.Helper()
	strip := func(es []economysim.Event) []economysim.Event {
		return slices.DeleteFunc(slices.Clone(es), func(e economysim.Event) bool {
			switch e.(type) {
			case economysim.WorldCreated, economysim.RumoursServed:
				return true
			}
			return false
		})
	}
	snap := func(w *economysim.World) economysim.Snapshot {
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		s.ConfigHash, s.Modules = [32]byte{}, nil
		return s
	}
	return reflect.DeepEqual(strip(ra.events), strip(rb.events)) && reflect.DeepEqual(snap(a), snap(b))
}
