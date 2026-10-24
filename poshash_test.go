package economysim_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// posTicks is the length of the scripted run of the positions golden (D178).
const posTicks = 30

// newPackWorld builds a world of cfg with the seed and options.
func newPackWorld(t testing.TB, cfg economysim.Config, seed uint64, opts ...economysim.Option) *economysim.World {
	t.Helper()
	w, err := economysim.NewWorld(cfg, append([]economysim.Option{economysim.WithSeed(seed)}, opts...)...)
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", seed, err)
	}
	return w
}

// positionLines replays ticks steps of a bare world of cfg and returns the lines.
func positionLines(t testing.TB, cfg economysim.Config, seed uint64, ticks int, opts ...economysim.Option) []string {
	t.Helper()
	lines, err := economysim.PositionHashes(newPackWorld(t, cfg, seed, opts...), ticks)
	if err != nil {
		t.Fatalf("PositionHashes(seed %d) error = %v, want nil", seed, err)
	}
	return lines
}

// divergenceNote names the first tick and position at which two recorded
// sequences differ, for the message of a failing golden.
func divergenceNote(want, got []string) string {
	if tick, pos, ok := economysim.FirstDivergence(want, got); ok {
		return fmt.Sprintf("first divergence at tick %d position %s", tick, pos)
	}
	return ""
}

// TestPositionHash_Golden verifies the per-position state hash of pack one, seed 1,
// ticks 0 to 29 (D178), against testdata/golden/off/one.positions.txt: one line
// "t=<tick> <position> <16 hex>" per position that ran.
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-WLD-044, NFR-WLD-004
//
// Guard: when a golden fails, the first differing line names the tick and the
// position at which the state diverged; the file is regenerated with -update
// only after the diff is understood (testing.md 7.2).
func TestPositionHash_Golden(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	lines := positionLines(t, cfg, goldenSeed, posTicks)
	if want := posTicks * 22; len(lines) != want {
		t.Errorf("recorded %d lines, want %d (22 positions per tick)", len(lines), want)
	}
	checkGolden(t, "one.positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// TestPositionHash_ReplayEqualsRun verifies the replay is the step: a world
// stepped 30 ticks by ReplayStep and one by Step end with the same state hash and
// snapshot, and a recorder-attached replay of 400 ticks with a snapshot between
// replayed steps renders the same one.txt bytes as the golden.
//
// Requirement: testing.md 7.1.2, FR-WLD-004, FR-WLD-063, NFR-WLD-004
//
// Guard: a replay that drifts from Step (a position missing from the map, or in
// the wrong order) would make the positions golden describe a different run.
func TestPositionHash_ReplayEqualsRun(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	t.Run("state hash after 30 ticks", func(t *testing.T) {
		t.Parallel()
		a, b := newPackWorld(t, cfg, goldenSeed), newPackWorld(t, cfg, goldenSeed)
		for i := 0; i < posTicks; i++ {
			if err := a.Step(); err != nil {
				t.Fatalf("Step at tick %d error = %v, want nil", i, err)
			}
			if err := economysim.ReplayStep(b); err != nil {
				t.Fatalf("ReplayStep at tick %d error = %v, want nil", i, err)
			}
			// Both worlds are snapshotted, so the scratch Snapshot writes is equal.
			for _, w := range []*economysim.World{a, b} {
				if _, err := w.Snapshot(); err != nil {
					t.Fatalf("Snapshot after tick %d error = %v, want nil (the stepping flag must be clear)", i, err)
				}
			}
		}
		if economysim.HashWorld(a) != economysim.HashWorld(b) {
			t.Errorf("state hash after %d ticks differs between Step and the replay", posTicks)
		}
		if a.Tick() != b.Tick() {
			t.Errorf("Tick = %d after the replay, want %d", b.Tick(), a.Tick())
		}
	})
	t.Run("one.txt of a replayed run", func(t *testing.T) {
		t.Parallel()
		rec := &recorder{}
		w := newPackWorld(t, cfg, goldenSeed, economysim.WithObserver(rec), economysim.WithAudit())
		res := result{World: w, Samples: map[int64]economysim.Snapshot{}, Ticks: goldenTicks}
		spec := runSpec{Ticks: goldenTicks, Snapshots: snapSample}
		for i := int64(0); i < goldenTicks; i++ {
			if err := economysim.ReplayStep(w); err != nil {
				t.Fatalf("ReplayStep at tick %d error = %v, want nil", i, err)
			}
			if spec.isSample(i) {
				s, err := w.Snapshot()
				if err != nil {
					t.Fatalf("Snapshot after tick %d error = %v, want nil", i, err)
				}
				res.Samples[i] = s
			}
		}
		var err error
		if res.Final, err = w.Snapshot(); err != nil {
			t.Fatalf("Snapshot error = %v, want nil", err)
		}
		res.Events = rec.events
		res.Digest = checkpointDigest(t, w)
		want, err := os.ReadFile(goldenPath("one.txt"))
		if err != nil {
			t.Fatalf("read golden one.txt: %v", err)
		}
		got := oneTxt(cfg, goldenSeed, goldenTicks, res, solveCold(t))
		if !bytes.Equal(want, got) {
			t.Errorf("one.txt of the replayed run differs from the golden at %s", firstLineDiff(want, got))
		}
	})
}

// TestPositionHash_NamesTheFirstDivergence verifies a change made to the state at
// a chosen tick and position is reported as the first divergence at exactly that
// tick and position, and that equal runs report none.
//
// Requirement: testing.md 7.1.2
//
// Guard: a hash that did not name the position at which the state first moved
// would leave a failing golden without a lead.
func TestPositionHash_NamesTheFirstDivergence(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	const ticks = 4
	base := positionLines(t, cfg, goldenSeed, ticks)
	if _, _, ok := economysim.FirstDivergence(base, positionLines(t, cfg, goldenSeed, ticks)); ok {
		t.Fatalf("two equal runs report a divergence")
	}
	for _, tc := range []struct {
		tick int64
		pos  string
	}{{0, "begin"}, {1, "P3.2"}, {2, "P8.4"}, {3, "end"}} {
		t.Run(fmt.Sprintf("t=%d %s", tc.tick, tc.pos), func(t *testing.T) {
			t.Parallel()
			got, err := economysim.PositionHashesPerturbed(newPackWorld(t, cfg, goldenSeed), ticks, tc.tick, tc.pos)
			if err != nil {
				t.Fatalf("PositionHashesPerturbed error = %v, want nil", err)
			}
			tick, pos, ok := economysim.FirstDivergence(base, got)
			if !ok || tick != tc.tick || pos != tc.pos {
				t.Errorf("first divergence = tick %d position %q (found %t), want tick %d position %q", tick, pos, ok, tc.tick, tc.pos)
			}
			if note := divergenceNote(base, got); !strings.Contains(note, fmt.Sprintf("tick %d position %s", tc.tick, tc.pos)) {
				t.Errorf("note = %q, want it to name tick %d position %s", note, tc.tick, tc.pos)
			}
		})
	}
}

// TestFirstDivergence_Report verifies the external wrapper on hand-made lines.
//
// Requirement: testing.md 7.1.2
func TestFirstDivergence_Report(t *testing.T) {
	t.Parallel()
	a := []string{"t=0 begin aa", "t=0 P0.2 bb"}
	if _, _, ok := economysim.FirstDivergence(a, a); ok {
		t.Errorf("equal sequences report a divergence")
	}
	tick, pos, ok := economysim.FirstDivergence(a, []string{"t=0 begin aa", "t=0 P0.2 zz"})
	if !ok || tick != 0 || pos != "P0.2" {
		t.Errorf("FirstDivergence = %d, %q, %t, want 0, P0.2, true", tick, pos, ok)
	}
	if got := divergenceNote(a, a[:1]); got != "first divergence at tick 0 position P0.2" {
		t.Errorf("note for a shorter sequence = %q", got)
	}
}

// positionsNote replays the scripted run of D178 on cfg and compares it with the
// recorded positions file, so a failing golden says where the state first moved.
func positionsNote(t testing.TB, cfg economysim.Config) string {
	t.Helper()
	b, err := os.ReadFile(goldenPath("one.positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with one.positions.txt: the divergence is later", posTicks-1)
}
