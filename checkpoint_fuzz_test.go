package economysim_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"runtime"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// fuzzMill returns the directed mill after ticks steps with a sticky directive on
// agent 1, the world whose checkpoints seed the fuzz targets and the allocation test.
func fuzzMill(t testing.TB, ticks int) *economysim.World {
	t.Helper()
	w, err := economysim.NewWorld(directiveMill(), economysim.WithSeed(5))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	economysim.SetAgentDirective(w, 1, &economysim.Directive{Hold: []economysim.CommodityID{0}, Sticky: true})
	for range ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	return w
}

// fuzzSeeds are valid checkpoints of fuzzMill, before the first step and after five,
// and the frames and fragments that the frame check refuses.
func fuzzSeeds(f *testing.F) [][]byte {
	f.Helper()
	var out [][]byte
	for _, ticks := range []int{0, 5} {
		c, err := fuzzMill(f, ticks).Checkpoint()
		if err != nil {
			f.Fatalf("Checkpoint() error = %v, want nil", err)
		}
		b, err := c.MarshalBinary()
		if err != nil {
			f.Fatalf("MarshalBinary() error = %v, want nil", err)
		}
		out = append(out, b)
	}
	return append(out, frameOf(1, 1, 0), []byte("ESCK"), nil)
}

// FuzzUnmarshalCheckpoint fuzzes the frame check: no input panics, and an accepted one
// marshals back to the bytes it was given and has a header Info can read.
//
// Requirement: FR-CHK-016, NFR-CHK-003
func FuzzUnmarshalCheckpoint(f *testing.F) {
	for _, s := range fuzzSeeds(f) {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		var c economysim.Checkpoint
		if err := c.UnmarshalBinary(data); err != nil {
			if !errors.Is(err, economysim.ErrCheckpointCorrupt) && !errors.Is(err, economysim.ErrCheckpointVersion) {
				t.Fatalf("UnmarshalBinary(%x) error = %v, want ErrCheckpointCorrupt or ErrCheckpointVersion", data, err)
			}
			return
		}
		got, err := c.MarshalBinary()
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("MarshalBinary() after UnmarshalBinary(%x) = %x, %v, want the same bytes, nil", data, got, err)
		}
		if info := c.Info(); info.Tick < 0 {
			t.Fatalf("Info().Tick = %d for an accepted frame, want at least 0", info.Tick)
		}
	})
}

// FuzzRestoreWorld fuzzes the restore of the mill from bytes whose trailer is
// recomputed, so a mutation of a section reaches the section readers instead of
// stopping at the digest. No input panics; a refusal is ErrCheckpointCorrupt,
// ErrCheckpointVersion, ErrCheckpointConfig or ErrCheckpointExtension; and an accepted
// input is a fixed point of restore-then-save with a world that passes its audit.
//
// Requirement: FR-CHK-005, FR-CHK-016, NFR-CHK-003
func FuzzRestoreWorld(f *testing.F) {
	for _, s := range fuzzSeeds(f) {
		f.Add(s)
	}
	cfg := directiveMill()
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) >= 32 {
			data = redigest(data[:len(data)-32])
		}
		var c economysim.Checkpoint
		if err := c.UnmarshalBinary(data); err != nil {
			return
		}
		w, err := economysim.RestoreWorld(cfg, c)
		if err != nil {
			if !errors.Is(err, economysim.ErrCheckpointCorrupt) && !errors.Is(err, economysim.ErrCheckpointConfig) &&
				!errors.Is(err, economysim.ErrCheckpointVersion) && !errors.Is(err, economysim.ErrCheckpointExtension) {
				t.Fatalf("RestoreWorld(%x) error = %v, want a checkpoint sentinel", data, err)
			}
			return
		}
		if err := w.Audit(); err != nil {
			t.Fatalf("Audit() of the world restored from %x error = %v, want nil", data, err)
		}
		if again := mustMarshal(t, mustCheckpoint(t, w)); !bytes.Equal(again, data) {
			t.Fatalf("restore then save of %x gave %x, want the same bytes", data, again)
		}
	})
}

// TestRestore_CraftedCountsAllocateLittle verifies that a count or an agent ID the
// input chose, set to 2^40, makes the restore fail having allocated little: the
// agent table is bounded by the ledger's accounts and every count by the bytes left.
//
// Requirement: FR-CHK-016, NFR-CHK-001
//
// Guard: a decoder that allocated by a number it read would let eleven bytes of input
// ask for terabytes; nothing else would notice until a fuzzer or an attacker did.
func TestRestore_CraftedCountsAllocateLittle(t *testing.T) {
	// No t.Parallel: the figure is the process-wide allocation counter.
	raw := mustMarshal(t, mustCheckpoint(t, fuzzMill(t, 5)))
	huge := binary.AppendUvarint(nil, 1<<40)
	cfg := directiveMill()
	for _, tt := range []struct {
		name string
		tag  uint64
		edit func(body []byte) []byte
	}{
		{"agent count", 3, func(b []byte) []byte { return append(huge, b[1:]...) }},
		{"next agent ID", 1, func(b []byte) []byte { return append(huge, b[len(binary.AppendUvarint(nil, 4+1)):]...) }},
		{"market window", 4, func(b []byte) []byte { return append(huge, b[1:]...) }},
		{"jobs", 6, func(b []byte) []byte { return append(huge, b[1:]...) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := reframe(t, raw, func(secs []section) []section { return editSection(secs, tt.tag, tt.edit) })
			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			_, err := economysim.RestoreWorld(cfg, c)
			runtime.ReadMemStats(&after)
			if !errors.Is(err, economysim.ErrCheckpointCorrupt) {
				t.Fatalf("RestoreWorld() error = %v, want ErrCheckpointCorrupt", err)
			}
			if got := after.TotalAlloc - before.TotalAlloc; got > 8<<20 {
				t.Errorf("RestoreWorld() allocated %d bytes, want at most %d for an input of %d bytes", got, 8<<20, len(raw))
			}
		})
	}
}
