package economysim_test

import (
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"log/slog"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// eventHash is an observer that folds every event after the first (WorldCreated carries
// the module) into one running hash, so ten parallel runs keep a number and not a log.
type eventHash struct {
	h hash.Hash64
	n int
}

// Observe folds the event.
func (e *eventHash) Observe(ev economysim.Event) {
	if e.n++; e.n > 1 {
		fmt.Fprintf(e.h, "%#v;", ev)
	}
}

// inertRun runs cfg for ticks ticks under seed and returns the hash of the events after
// WorldCreated (0 without an observer) and the hash of the snapshot at every tenth
// boundary and at the end, with the three carriers of the module blanked
// (FR-TEC-001: ConfigHash, Modules, Names).
func inertRun(t *testing.T, cfg economysim.Config, seed uint64, ticks int, observe, debugLog bool) (events, states uint64) {
	t.Helper()
	opts := []economysim.Option{economysim.WithSeed(seed), economysim.WithAudit()}
	rec := &eventHash{h: fnv.New64a()}
	if observe {
		opts = append(opts, economysim.WithObserver(rec))
	}
	if debugLog {
		opts = append(opts, economysim.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))))
	}
	w := mustWorld(t, cfg, opts...)
	sh := fnv.New64a()
	for i := 1; i <= ticks; i++ {
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, i, err)
		}
		if i%10 == 0 || i == ticks {
			s, err := w.Snapshot()
			if err != nil {
				t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
			}
			s.ConfigHash, s.Modules, s.Names = [32]byte{}, nil, economysim.Names{}
			fmt.Fprintf(sh, "%#v;", s)
		}
	}
	if observe {
		events = rec.h.Sum64()
	}
	return events, sh.Sum64()
}

// TestScenarioS148_InertEqualsOff verifies TEC-S1 of spec-technology.md 14.4 (FR-TEC-001):
// the module on with no field, and on with a field whose every rate is 0, equals the
// module off in events after WorldCreated and in the snapshot at every tenth boundary
// (ConfigHash, Modules and Names excepted), with and without an observer, and with a
// Debug logger. Pack one with its warm start, seeds 1 to 10, 100 ticks, audit on. The
// run keeps hashes per seed, never a snapshot per seed.
//
// Requirement: FR-TEC-001, NFR-TEC-001
func TestScenarioS148_InertEqualsOff(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	t.Parallel()
	off := loadValidPack(t, "base.toml", "one.toml", "warm-one.toml")
	empty, inert := off, off
	empty.Modules.Technology = &economysim.Technology{}
	inert.Modules.Technology = &economysim.Technology{Fields: []economysim.TechField{{ID: "farming", Recipes: []string{"grow_staple"}, CapBP: 3_000}}}
	for seed := uint64(1); seed <= uint64(scenarioSeedCount()); seed++ {
		t.Run("seed "+strconv.FormatUint(seed, 10), func(t *testing.T) {
			t.Parallel()
			wantE, wantS := inertRun(t, off, seed, 100, true, false)
			for name, cfg := range map[string]economysim.Config{"no field": empty, "every rate 0": inert} {
				if e, s := inertRun(t, cfg, seed, 100, true, false); e != wantE || s != wantS {
					t.Errorf("seed %d, %s: events %x states %x, want %x and %x of the module off", seed, name, e, s, wantE, wantS)
				}
				if _, s := inertRun(t, cfg, seed, 100, false, true); s != wantS {
					t.Errorf("seed %d, %s, no observer and a Debug logger: states %x, want %x of the module off", seed, name, s, wantS)
				}
			}
		})
	}
}
