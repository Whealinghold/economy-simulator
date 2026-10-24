package economysim_test

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// ckptE2ETicks is how long the end-to-end checkpoint runs go: long enough for the
// reference chain to produce, trade, spend and (with the warm start) settle, so
// every section of the frame holds live rows rather than the empty ones of tick 0.
const ckptE2ETicks = 60

// ckptE2EEvery is the interval of the checkpoints taken during a run.
const ckptE2EEvery = 15

// ckptRun steps a pack-one world for ckptE2ETicks and returns the marshalled
// checkpoint taken every ckptE2EEvery ticks (and before the first step), keyed in
// order, with the world. When take is false it takes none, to give the bare run.
func ckptRun(t *testing.T, cfg economysim.Config, seed uint64, take bool) ([][]byte, *economysim.World) {
	t.Helper()
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed))
	if err != nil {
		t.Fatalf("seed %d: NewWorld() error = %v, want nil", seed, err)
	}
	var out [][]byte
	for i := range ckptE2ETicks + 1 {
		if take && i%ckptE2EEvery == 0 {
			out = append(out, mustMarshal(t, mustCheckpoint(t, w)))
		}
		if i == ckptE2ETicks {
			break
		}
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step at tick %d error = %v, want nil", seed, i, err)
		}
	}
	return out, w
}

// TestCheckpoint_ReferenceChainRun verifies the write path as a consumer uses it:
// pack one of the reference chain, run for 60 ticks over several seeds, a
// checkpoint every 15 ticks. Each checkpoint is a valid frame whose header agrees
// with the world (format 1, its configuration hash, its tick and seed) and whose
// sections are the eleven core tags and the table's, in order; it survives
// MarshalBinary and UnmarshalBinary byte for byte; two worlds of one seed give the
// same bytes at every boundary; the state moves between boundaries; and a
// different seed gives different bytes.
//
// Arithmetic: ticks 0, 15, 30, 45 and 60 are 5 checkpoints per run.
//
// Requirement: FR-CHK-001, FR-CHK-004, FR-CHK-006, FR-CHK-009, FR-CHK-063
//
// Guard: the unit tests build small hand-made worlds; only a long run of the real
// pack holds rows in every section (listings, jobs, beliefs, welfare), and a
// section that writes a map-ordered or run-dependent byte would pass them and fail here.
func TestCheckpoint_ReferenceChainRun(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	wantTags := []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 34}
	seeds := seedList(invariantSeeds(3))
	last := make([][]byte, len(seeds)) // the checkpoint at the last boundary, one slot per seed
	for si, seed := range seeds {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			a, wa := ckptRun(t, cfg, seed, true)
			b, _ := ckptRun(t, cfg, seed, true)
			if len(a) != 5 || len(b) != 5 {
				t.Fatalf("seed %d: checkpoints = %d and %d, want 5 each", seed, len(a), len(b))
			}
			for i, raw := range a {
				tick := int64(i * ckptE2EEvery)
				if !bytes.Equal(raw, b[i]) {
					t.Errorf("seed %d: checkpoint at tick %d differs between two worlds of one seed", seed, tick)
				}
				h := decodeFrame(t, raw)
				if h.format != 1 || h.tick != uint64(tick) || h.seed != seed || h.hash != wa.ConfigHash() {
					t.Errorf("seed %d: header at tick %d = format %d, tick %d, seed %d, hash %x; want 1, %d, %d, %x",
						seed, tick, h.format, h.tick, h.seed, h.hash, tick, seed, wa.ConfigHash())
				}
				var tags []uint64
				for _, s := range h.sections {
					tags = append(tags, s.tag)
				}
				if !slices.Equal(tags, wantTags) {
					t.Errorf("seed %d: section tags at tick %d = %v, want %v", seed, tick, tags, wantTags)
				}
				var c economysim.Checkpoint
				if err := c.UnmarshalBinary(raw); err != nil {
					t.Fatalf("seed %d: UnmarshalBinary at tick %d error = %v, want nil", seed, tick, err)
				}
				if got := mustMarshal(t, c); !bytes.Equal(got, raw) {
					t.Errorf("seed %d: MarshalBinary(UnmarshalBinary(b)) at tick %d differs from b", seed, tick)
				}
				if got := c.Info(); got.Tick != tick || got.Seed != seed || got.Bytes != len(raw) {
					t.Errorf("seed %d: Info() at tick %d = %+v, want tick %d, seed %d, %d bytes", seed, tick, got, tick, seed, len(raw))
				}
				if i > 0 && bytes.Equal(raw, a[i-1]) {
					t.Errorf("seed %d: checkpoints at ticks %d and %d are equal, want the state to have moved", seed, tick-ckptE2EEvery, tick)
				}
			}
			last[si] = a[len(a)-1] // each subtest writes its own slot; the cleanup reads after all finish
		})
	}
	t.Cleanup(func() {
		for i := range seeds {
			for j := i + 1; j < len(seeds); j++ {
				if bytes.Equal(last[i], last[j]) {
					t.Errorf("seeds %d and %d give equal checkpoints at tick %d, want different bytes", seeds[i], seeds[j], ckptE2ETicks)
				}
			}
		}
	})
}

// TestCheckpoint_TakingThemDoesNotMoveTheRun verifies that a run that takes a
// checkpoint every 15 ticks ends in the same state as one that takes none: equal
// state hash, equal snapshot, equal tick.
//
// Requirement: FR-CHK-001, FR-CHK-004, FR-OBS-023
//
// Guard: a checkpoint reads the world; if a section writer drew from the random
// source, sorted in place or cleared a scratch mark, only a long run would
// diverge, and the unit test of one boundary would not see it.
func TestCheckpoint_TakingThemDoesNotMoveTheRun(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	for _, seed := range seedList(invariantSeeds(2)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			_, with := ckptRun(t, cfg, seed, true)
			_, bare := ckptRun(t, cfg, seed, false)
			if with.Tick() != bare.Tick() {
				t.Fatalf("seed %d: tick = %d with checkpoints, %d without", seed, with.Tick(), bare.Tick())
			}
			if economysim.HashWorld(with) != economysim.HashWorld(bare) {
				t.Errorf("seed %d: state hash differs between the run with checkpoints and the bare run", seed)
			}
			sw, err := with.Snapshot()
			if err != nil {
				t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
			}
			sb, err := bare.Snapshot()
			if err != nil {
				t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
			}
			if !reflect.DeepEqual(sw, sb) {
				t.Errorf("seed %d: snapshot differs between the run with checkpoints and the bare run", seed)
			}
		})
	}
}
