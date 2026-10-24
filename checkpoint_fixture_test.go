package economysim_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The format 1 fixture is the checkpoint of pack one (the three layers, warm start
// included: the golden world), seed 1, at tick fixtureTick, with a sidecar of the
// facts a reader checks before trusting it (plan 2-2 D259).
const (
	fixtureTick   int64  = 50
	fixtureLater  int64  = 20
	fixtureSeed   uint64 = 1
	fixtureBinary        = "format-1-one.bin"
	fixtureSide          = "format-1-one.txt"
)

func fixturePath(name string) string { return filepath.Join("testdata", "checkpoints", name) }

// trailerHex is the hex of the digest (the trailer) of an encoded checkpoint.
func trailerHex(b []byte) string { return hex.EncodeToString(b[len(b)-sha256.Size:]) }

// fixtureSidecar renders the sidecar of the fixture: tick, config hash, digest, and
// the digest of the world fixtureLater ticks after a restore.
func fixtureSidecar(c economysim.Checkpoint, b []byte, later string) string {
	info := c.Info()
	return fmt.Sprintf("format %d\ntick %d\nconfig_hash %s\ndigest %s\ndigest_after %d %s\n",
		info.Format, info.Tick, hex.EncodeToString(info.ConfigHash[:]), trailerHex(b), fixtureLater, later)
}

// TestFixture_Format1 verifies that the committed format 1 checkpoint still restores
// and continues: its sidecar matches its frame, the restored world checkpoints to the
// same bytes, and fixtureLater ticks on its digest is the sidecar's. A change that
// breaks the fixture changed the format (bump it) or the behaviour (explain it,
// testing.md 7.2). -update rewrites both files from a fresh run.
//
// Requirement: FR-CHK-062, FR-CHK-060
//
// Guard: the fixture is the one checkpoint written by an older build that the next
// build must read; without it the format could change with no test saying so.
func TestFixture_Format1(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	if *updateFlag {
		writeFixture(t, cfg)
	}
	raw, err := os.ReadFile(fixturePath(fixtureBinary))
	if err != nil {
		t.Fatalf("read fixture: %v (record it with -update after reading testing.md 7.2)", err)
	}
	var c economysim.Checkpoint
	if err := c.UnmarshalBinary(raw); err != nil {
		t.Fatalf("UnmarshalBinary(fixture) error = %v, want nil", err)
	}
	w, err := economysim.RestoreWorld(cfg, c)
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
	want, err := os.ReadFile(fixturePath(fixtureSide))
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	if got := fixtureSidecar(c, raw, later); got != string(want) {
		t.Errorf("sidecar differs from the fixture\n got:\n%s want:\n%s", got, want)
	}
}

// writeFixture records the fixture: pack one, seed fixtureSeed, fixtureTick ticks. The
// digest after the continuation comes from a restore of the bytes just written, the
// path the test takes.
func writeFixture(t *testing.T, cfg economysim.Config) {
	t.Helper()
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(fixtureSeed))
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
	r, err := economysim.RestoreWorld(cfg, c)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	for range fixtureLater {
		if err := r.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
	}
	later := trailerHex(mustMarshal(t, mustCheckpoint(t, r)))
	if err := os.MkdirAll(filepath.Dir(fixturePath(fixtureBinary)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixturePath(fixtureBinary), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixturePath(fixtureSide), []byte(fixtureSidecar(c, raw, later)), 0o644); err != nil {
		t.Fatal(err)
	}
}
