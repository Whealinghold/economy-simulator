package economysim_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// forkFiveN and forkFiveM are the ticks before and after the boundary of the five-sized
// fork tests (REF 8.2: a checkpoint at 200 of 400, X5).
const forkFiveN, forkFiveM = 200, 200

// fiveFork is the fork case of a five-sized world at 200 and 200, seed 1.
func fiveFork(t *testing.T, w fiveWorld) forkCase {
	t.Helper()
	return forkCase{Name: w.name, Cfg: w.config(t), Opts: func() []economysim.Option { return w.opts }, Seed: 1, N: forkFiveN, M: forkFiveM}
}

// TestCheckpoint_FiveForkAt200 verifies X5 on the warm `five` with every module off: the
// original and two restores (one a restore of a restore) give the same events, snapshot,
// audit and checkpoint bytes at every one of the 200 ticks after a checkpoint taken at 200.
//
// Requirement: FR-REF-032, FR-CHK-050, FR-CHK-057
//
// Guard: the fork at 200 on 5 000 households is the one that sees a restore of a large
// population (the modules-off checkpoint is about 1 MB).
func TestCheckpoint_FiveForkAt200(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	forkAtN(t, fiveFork(t, fiveOff))
}

// TestCheckpoint_AllOnForkAt200 verifies X5 on the all-on pack: the same fork at 200 with
// every module on, so every section of the checkpoint (the demand carries, the money
// flows, the journeys, the shipments, the contracts, the outside node) is saved and
// restored at a boundary where it holds state.
//
// Requirement: FR-REF-032, FR-CHK-050, FR-CHK-057
//
// Guard: a module whose state is not in its section shows here as a difference between
// the original and the restore, and nowhere else at this size.
func TestCheckpoint_AllOnForkAt200(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	forkAtN(t, fiveFork(t, allOn))
}

// The format 1 fixture of the all-on pack is the checkpoint of allLayers, seed 1, at tick
// fixtureTick, with a sidecar of the form of format-1-one.txt (plan 2-24 D693, CHK 9.2.1
// item 3).
const (
	fixtureAllBinary = "format-1-all.bin"
	fixtureAllSide   = "format-1-all.txt"
)

// writeFixtureAll records the all-on fixture: the checkpoint at tick fixtureTick and the
// digest of the world fixtureLater ticks after a restore of those bytes.
func writeFixtureAll(t *testing.T, cfg economysim.Config) {
	t.Helper()
	w, err := economysim.NewWorld(cfg, append(allOptions(), economysim.WithSeed(fixtureSeed))...)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	for range fixtureTick {
		if err := w.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
	}
	c := mustCheckpoint(t, w)
	raw := mustMarshal(t, c)
	r, err := economysim.RestoreWorld(cfg, c, allOptions()...)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	for range fixtureLater {
		if err := r.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
	}
	later := trailerHex(mustMarshal(t, mustCheckpoint(t, r)))
	if err := os.MkdirAll(filepath.Dir(fixturePath(fixtureAllBinary)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixturePath(fixtureAllBinary), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixturePath(fixtureAllSide), []byte(fixtureSidecar(c, raw, later)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestFixture_Format1All verifies that the committed all-on format 1 checkpoint still
// restores and continues: its sidecar matches its frame, the restored world checkpoints to
// the same bytes, and fixtureLater ticks on its digest is the sidecar's. A change that
// breaks it changed the format (bump it) or the behaviour (explain it, testing.md 7.2).
// -update rewrites both files from a fresh run.
//
// Requirement: FR-CHK-062, FR-CHK-060
//
// Guard: DELIBERATE NEW GOLDEN, new files. The fixture is the one all-modules-on checkpoint
// an older build wrote that the next build must read; a Stage 3 module that changes the
// format supersedes it with a fixture of its own.
func TestFixture_Format1All(t *testing.T) {
	t.Parallel()
	skipFiveSized(t)
	cfg := loadValidPack(t, allLayers...)
	if *updateFlag {
		writeFixtureAll(t, cfg)
	}
	raw, err := os.ReadFile(fixturePath(fixtureAllBinary))
	if err != nil {
		t.Fatalf("read fixture: %v (record it with -update after reading testing.md 7.2)", err)
	}
	var c economysim.Checkpoint
	if err := c.UnmarshalBinary(raw); err != nil {
		t.Fatalf("UnmarshalBinary(fixture) error = %v, want nil", err)
	}
	w, err := economysim.RestoreWorld(cfg, c, allOptions()...)
	if err != nil {
		t.Fatalf("RestoreWorld(fixture) error = %v, want nil", err)
	}
	if again := mustMarshal(t, mustCheckpoint(t, w)); !bytes.Equal(again, raw) {
		t.Errorf("restore then Checkpoint of the fixture differs from the fixture (digest %s, want %s)", trailerHex(again), trailerHex(raw))
	}
	for range fixtureLater {
		if err := w.Step(); err != nil {
			t.Fatalf("Step at tick %d error = %v, want nil", w.Tick(), err)
		}
	}
	later := trailerHex(mustMarshal(t, mustCheckpoint(t, w)))
	want, err := os.ReadFile(fixturePath(fixtureAllSide))
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	if got := fixtureSidecar(c, raw, later); got != string(want) {
		t.Errorf("sidecar differs from the fixture\n got:\n%s want:\n%s", got, want)
	}
}
