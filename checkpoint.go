package economysim

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the checkpoint as a value and its write path (spec-checkpoint.md
// 3, 5, 7; plan 2-2 Phase 4): the frame, the format constants, the optional
// Stateful interface and World.Checkpoint. The section table is in
// checkpoint_sections.go, the extension entries in checkpoint_save.go. Nothing
// here restores: RestoreWorld is the restore change.

// The format constants of format 1 (spec-checkpoint.md 5.1, 9.3). They are rule
// constants: the magic and the version are part of the golden digest. A module
// that appends a section does not change the version; only a change to a
// section a released library already wrote does (spec-checkpoint.md 9.3).
const (
	checkpointFormat uint32 = 1
	checkpointMagic         = "ESCK"
	// digestSize is the length of the trailer, the SHA-256 of every byte before it.
	digestSize = sha256.Size
)

// Checkpoint is the complete state of a world at a tick boundary in one
// canonical binary encoding (spec-checkpoint.md 3.1). It is immutable: it holds
// its bytes and never changes them, and one value may be read from any number of
// goroutines. The zero Checkpoint is empty and not valid.
type Checkpoint struct {
	b []byte
}

// CheckpointInfo is what the frame of a checkpoint says, before any configuration
// is chosen: the format version of the bytes, the hash of the effective
// configuration of the world that was saved, the tick the next Step would have
// run, the effective seed (the seed after a reseed, which a restore reads back)
// and the length of the encoding.
type CheckpointInfo struct {
	Format     uint32
	ConfigHash [32]byte
	Tick       int64
	Seed       uint64
	Bytes      int
}

// frame is what the header of a checkpoint holds: the parts of CheckpointInfo
// and the version of the configuration encoding that made the hash, and the
// sections between the header and the trailer.
type frame struct {
	info        CheckpointInfo
	hashVersion uint64
	sections    []byte
}

// parseFrame checks the frame of b and nothing else, in the order of
// spec-checkpoint.md 6.2 step 1: the length, the magic, the format (before the
// digest, so a newer format is reported as such), the digest and the ranges of
// the header. It returns ErrCheckpointCorrupt or ErrCheckpointVersion, wrapped
// with what failed.
func parseFrame(b []byte) (frame, error) { return readFrame(b, true) }

// readFrame is parseFrame with the digest check optional: Info skips it, because
// a checkpoint in hand was verified when it was made or unmarshalled.
func readFrame(b []byte, verify bool) (frame, error) {
	corrupt := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrCheckpointCorrupt, fmt.Sprintf(format, args...))
	}
	if len(b) < len(checkpointMagic) || !bytes.HasPrefix(b, []byte(checkpointMagic)) {
		return frame{}, corrupt("no magic in %d bytes", len(b))
	}
	r := codec.NewReader(b[len(checkpointMagic):])
	format := r.U()
	if err := r.Err(); err != nil {
		return frame{}, fmt.Errorf("%w: header: %w", ErrCheckpointCorrupt, err)
	}
	if format != uint64(checkpointFormat) {
		return frame{}, fmt.Errorf("%w: format %d, this library reads format %d", ErrCheckpointVersion, format, checkpointFormat)
	}
	if len(b) < len(checkpointMagic)+digestSize {
		return frame{}, corrupt("%d bytes is too short for a trailer", len(b))
	}
	body, trailer := b[:len(b)-digestSize], b[len(b)-digestSize:]
	if verify {
		if sum := sha256.Sum256(body); !bytes.Equal(sum[:], trailer) {
			return frame{}, corrupt("the digest does not match the bytes")
		}
	}
	return parseHeader(codec.NewReader(body[len(checkpointMagic):]), format, len(b))
}

// parseHeader reads the header after the magic from r, which holds the bytes up
// to the trailer, and returns the frame with the sections that follow it.
func parseHeader(r *codec.Reader, format uint64, size int) (frame, error) {
	r.U() // the format, read and checked by the caller
	f := frame{info: CheckpointInfo{Format: uint32(format), Bytes: size}}
	f.hashVersion = r.U()
	copy(f.info.ConfigHash[:], r.Raw(len(f.info.ConfigHash)))
	tick := r.U()
	f.info.Seed = r.U()
	if err := r.Err(); err != nil {
		return frame{}, fmt.Errorf("%w: header: %w", ErrCheckpointCorrupt, err)
	}
	if f.hashVersion != hashVersion {
		return frame{}, fmt.Errorf("%w: header: hash version %d, want %d", ErrCheckpointCorrupt, f.hashVersion, hashVersion)
	}
	if tick > maxTick {
		return frame{}, fmt.Errorf("%w: header: tick %d above %d", ErrCheckpointCorrupt, tick, int64(maxTick))
	}
	f.info.Tick = int64(tick)
	f.sections = r.Raw(r.Remaining())
	return f, nil
}

// maxTick is the largest tick a header may hold: the tick is a signed count.
const maxTick = 1<<63 - 1

// Info reports what the frame says. It has no error result and decodes no
// section: a checkpoint made by Checkpoint or accepted by UnmarshalBinary has a
// valid frame, and the zero Checkpoint reports the zero CheckpointInfo.
func (c Checkpoint) Info() CheckpointInfo {
	f, err := readFrame(c.b, false)
	if err != nil {
		return CheckpointInfo{}
	}
	return f.info
}

// MarshalBinary implements encoding.BinaryMarshaler: it returns a copy of the
// encoding.
func (c Checkpoint) MarshalBinary() ([]byte, error) {
	return bytes.Clone(c.b), nil
}

// AppendBinary implements encoding.BinaryAppender: it appends a copy of the
// encoding to b.
func (c Checkpoint) AppendBinary(b []byte) ([]byte, error) {
	return append(b, c.b...), nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler: it checks the frame
// (the magic, the format version, the digest and the header) and nothing else,
// then keeps a copy of data, so the caller may reuse the slice. A bad frame is
// ErrCheckpointCorrupt or, for a format this library does not read,
// ErrCheckpointVersion, and leaves c unchanged. RestoreWorld checks the rest.
func (c *Checkpoint) UnmarshalBinary(data []byte) error {
	if _, err := parseFrame(data); err != nil {
		return err
	}
	c.b = bytes.Clone(data)
	return nil
}

// Stateful is implemented by a Strategy, Spawner, Clearing or Policy that keeps
// state between calls, so that a checkpoint can save it and a restore can load
// it. It is optional and is not an extension point: no code requires it, an
// implementation that keeps no state does not implement it, and it has no
// default. An implementation that does not implement it and has state breaks
// bit-identity after a restore, and the library cannot see that.
//
// The contract (spec-checkpoint.md 7.2; architecture.md 7.7). SaveState and
// LoadState are called between ticks only, by World.Checkpoint and RestoreWorld,
// once per registered extension, in the order of (kind, name). SaveState MUST NOT
// change the extension and MUST give equal bytes for equal states: no map order,
// no address, no time. It returns the whole state, every piece of it that can
// influence a result, and the version of its encoding; no state may be in a
// closure, a package variable or a goroutine. The world copies the returned
// slice. LoadState replaces the whole state, as if the instance were new and then
// loaded, MUST validate its input and return an error for a version or bytes it
// cannot read, and MUST NOT keep the slice it is given, which aliases the
// checkpoint. Neither method receives a random source or may create one. An
// instance that holds state MUST NOT be shared between worlds.
type Stateful interface {
	// SaveState returns the version of the encoding and the whole state.
	SaveState() (version uint32, state []byte, err error)
	// LoadState replaces the whole state with the one SaveState wrote under the
	// same version.
	LoadState(version uint32, state []byte) error
}

// Checkpoint returns the checkpoint of the world: its complete state at this
// tick boundary, in the one canonical encoding (spec-checkpoint.md 3). It changes
// no state, draws no random number, emits no event and logs only a Debug line.
// Two checkpoints of one boundary are byte-equal. It returns ErrWorldFailed on a
// failed world (the error wraps the cause) and ErrInStep inside a step, and
// ErrCheckpointExtension wrapping the error of an extension whose SaveState
// failed. It may be taken before the first Step: the world is then still in its
// construction phase and the checkpoint says so.
func (w *World) Checkpoint() (Checkpoint, error) {
	if err := w.guard(); err != nil {
		return Checkpoint{}, err
	}
	out := codec.NewWriter(nil)
	out.Raw([]byte(checkpointMagic))
	out.U(uint64(checkpointFormat))
	out.U(hashVersion)
	out.Raw(w.hash[:])
	out.U(uint64(w.Tick()))
	out.U(w.state.seed)
	for i := range checkpointSections {
		s := &checkpointSections[i]
		// Excluded from coverage (testing.md 9.5): every row of the table is present
		// with no condition until a module appends its section, and that change
		// tests the guard.
		if s.present != nil && !s.present(&w.cfg) {
			continue
		}
		body := codec.NewWriter(nil)
		if err := s.save(w, body); err != nil {
			return Checkpoint{}, err
		}
		out.U(s.tag)
		out.Bytes(body.Out())
	}
	sum := sha256.Sum256(out.Out())
	out.Raw(sum[:])
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "checkpoint taken",
			slog.Int64(logkeys.Tick, w.Tick()), slog.Int64(logkeys.Bytes, int64(out.Len())))
	}
	return Checkpoint{b: out.Out()}, nil
}
