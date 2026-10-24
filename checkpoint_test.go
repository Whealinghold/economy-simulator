package economysim_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file tests the write path of the checkpoint through the public API
// (plan 2-2 Phase 4): the frame, the refusals, the copies, the extension
// entries and the options. It reads the bytes with a decoder of its own, so a
// layout the library gets wrong cannot hide behind its own reader. The restore
// and the bit-identity matrix are the restore change's.

// ckptSeed is the seed of the worlds of this file.
const ckptSeed = 77

// hashVersionOfFormat1 is the version of the configuration encoding that format
// 1 carries (spec-config.md 8.2, hash.go): the second number of the header.
const hashVersionOfFormat1 = 1

// uv reads one unsigned varint of b from off and returns it and the new offset.
// It uses encoding/binary because it is the test's reader of canonical input;
// the library's own decoder checks the minimal form (spec-checkpoint.md 5.2.1).
func uv(t *testing.T, b []byte, off int) (uint64, int) {
	t.Helper()
	v, n := binary.Uvarint(b[off:])
	if n <= 0 {
		t.Fatalf("varint at offset %d of %d bytes: Uvarint returned %d", off, len(b), n)
	}
	return v, off + n
}

// header is the decoded frame of a checkpoint.
type header struct {
	format, hashVersion uint64
	hash                [32]byte
	tick, seed          uint64
	sections            []section
}

// section is one section of a frame: its tag and its body.
type section struct {
	tag  uint64
	body []byte
}

// decodeFrame reads the frame of b: the magic, the header, the sections and the
// trailer, which it checks.
func decodeFrame(t *testing.T, b []byte) header {
	t.Helper()
	if !bytes.HasPrefix(b, []byte("ESCK")) {
		t.Fatalf("checkpoint starts %x, want the magic 45 53 43 4b", b[:min(4, len(b))])
	}
	end := len(b) - sha256.Size
	if sum := sha256.Sum256(b[:end]); !bytes.Equal(sum[:], b[end:]) {
		t.Fatalf("trailer = %x, want the SHA-256 of the %d bytes before it, %x", b[end:], end, sum)
	}
	var h header
	off := 4
	h.format, off = uv(t, b, off)
	h.hashVersion, off = uv(t, b, off)
	off += copy(h.hash[:], b[off:off+32])
	h.tick, off = uv(t, b, off)
	h.seed, off = uv(t, b, off)
	for off < end {
		var tag, n uint64
		tag, off = uv(t, b, off)
		n, off = uv(t, b, off)
		h.sections = append(h.sections, section{tag, b[off : off+int(n)]})
		off += int(n)
	}
	if off != end {
		t.Fatalf("sections end at %d, want %d (the trailer starts there)", off, end)
	}
	return h
}

// sectionOf returns the body of the section with the tag.
func (h header) sectionOf(t *testing.T, tag uint64) []byte {
	t.Helper()
	for _, s := range h.sections {
		if s.tag == tag {
			return s.body
		}
	}
	t.Fatalf("no section with tag %d", tag)
	return nil
}

// ckptWorld builds the world of millCfg(3, 4) with the seed and options, steps it
// ticks times and returns it.
func ckptWorld(t *testing.T, ticks int, opts ...economysim.Option) *economysim.World {
	t.Helper()
	w, err := economysim.NewWorld(millCfg(3, 4), append([]economysim.Option{economysim.WithSeed(ckptSeed)}, opts...)...)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	for i := range ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("Step at tick %d error = %v, want nil", i, err)
		}
	}
	return w
}

// mustCheckpoint takes a checkpoint of w.
func mustCheckpoint(t *testing.T, w *economysim.World) economysim.Checkpoint {
	t.Helper()
	c, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() at tick %d error = %v, want nil", w.Tick(), err)
	}
	return c
}

// mustMarshal returns the bytes of c.
func mustMarshal(t *testing.T, c economysim.Checkpoint) []byte {
	t.Helper()
	b, err := c.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v, want nil", err)
	}
	return b
}

// TestCheckpoint_FrameLayout verifies the frame of a checkpoint at tick 0 and at
// tick 5: the magic, the format 1, the version of the configuration encoding, the
// configuration hash, the tick, the effective seed, the sections of tags 1 to 11
// and 34 in order, and a trailer that is the SHA-256 of every byte before it.
//
// Requirement: FR-CHK-009, FR-CHK-063
func TestCheckpoint_FrameLayout(t *testing.T) {
	t.Parallel()
	for _, ticks := range []int{0, 5} {
		w := ckptWorld(t, ticks)
		h := decodeFrame(t, mustMarshal(t, mustCheckpoint(t, w)))
		if h.format != 1 || h.hashVersion != hashVersionOfFormat1 || h.hash != w.ConfigHash() || h.tick != uint64(ticks) || h.seed != ckptSeed {
			t.Errorf("tick %d: header = format %d, hash version %d, hash %x, tick %d, seed %d, want 1, %d, %x, %d, %d",
				ticks, h.format, h.hashVersion, h.hash, h.tick, h.seed, hashVersionOfFormat1, w.ConfigHash(), ticks, ckptSeed)
		}
		var tags []uint64
		for _, s := range h.sections {
			tags = append(tags, s.tag)
		}
		if want := []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 34}; !slices.Equal(tags, want) {
			t.Errorf("tick %d: section tags = %v, want %v", ticks, tags, want)
		}
	}
}

// TestCheckpoint_TwiceEqualAndNoSideEffect verifies that two checkpoints of one
// boundary are byte-equal and that taking checkpoints between the steps changes
// nothing: the events, the snapshot, the final checkpoint and the log of a run
// with the calls equal those of a run without, and a call itself emits no event
// and no line above Debug.
//
// Arithmetic: both worlds step 8 ticks of millCfg(3, 4) with seed 77; the
// second takes a checkpoint after each of the first 6 steps.
//
// Requirement: FR-CHK-001, FR-CHK-004, FR-CHK-038, FR-CHK-081
//
// Guard: a checkpoint that drew a random word, reordered a slice or logged at
// Info would make the run that takes one differ from the run that does not.
func TestCheckpoint_TwiceEqualAndNoSideEffect(t *testing.T) {
	t.Parallel()
	run := func(checkpoints bool) (*economysim.World, *recorder, *captureHandler) {
		rec, h := &recorder{}, newCaptureHandler()
		w := ckptWorld(t, 0, economysim.WithObserver(rec), economysim.WithLogger(slog.New(h)))
		for i := range 8 {
			if checkpoints && i < 6 {
				a, b := mustMarshal(t, mustCheckpoint(t, w)), mustMarshal(t, mustCheckpoint(t, w))
				if !bytes.Equal(a, b) {
					t.Fatalf("tick %d: two checkpoints of one boundary differ, want byte-equal", i)
				}
			}
			if err := w.Step(); err != nil {
				t.Fatalf("Step at tick %d error = %v, want nil", i, err)
			}
		}
		return w, rec, h
	}
	with, recWith, logWith := run(true)
	without, recWithout, logWithout := run(false)
	if !reflect.DeepEqual(recWith.events, recWithout.events) {
		t.Errorf("events of the run with checkpoints differ from those of the run without (%d and %d events)", len(recWith.events), len(recWithout.events))
	}
	snapWith, _ := with.Snapshot()
	snapWithout, _ := without.Snapshot()
	if !reflect.DeepEqual(snapWith, snapWithout) {
		t.Errorf("snapshot after the run with checkpoints differs from the one without")
	}
	if !bytes.Equal(mustMarshal(t, mustCheckpoint(t, with)), mustMarshal(t, mustCheckpoint(t, without))) {
		t.Errorf("final checkpoints of the run with and the run without checkpoints differ, want byte-equal")
	}
	// quiet returns the lines less the Debug lines of the Checkpoint calls: the
	// final comparison above is a call too, on both worlds.
	quiet := func(h *captureHandler) (kept []logLine) {
		for _, l := range h.records() {
			if l.msg == "checkpoint taken" {
				if l.level != slog.LevelDebug {
					t.Errorf("line %q has level %v, want Debug", l.msg, l.level)
				}
				continue
			}
			kept = append(kept, l)
		}
		return kept
	}
	if a, b := quiet(logWith), quiet(logWithout); !reflect.DeepEqual(a, b) {
		t.Errorf("log of the run with checkpoints, less the Debug lines of the calls, differs from the run without (%d and %d lines)", len(a), len(b))
	}
}

// TestCheckpoint_DebugLine verifies the Debug line of a checkpoint: the message
// "checkpoint taken" with the tick and the size in bytes, once per call.
//
// Requirement: FR-CHK-081
func TestCheckpoint_DebugLine(t *testing.T) {
	t.Parallel()
	h := newCaptureHandler()
	w := ckptWorld(t, 3, economysim.WithLogger(slog.New(h)))
	h.forget()
	c := mustCheckpoint(t, w)
	lines := h.records()
	if len(lines) != 1 || lines[0].msg != "checkpoint taken" || lines[0].level != slog.LevelDebug {
		t.Fatalf("log after Checkpoint = %+v, want one Debug line \"checkpoint taken\"", lines)
	}
	if tick, _ := lines[0].attr("tick"); tick.Int64() != 3 {
		t.Errorf("tick attribute = %v, want 3", tick)
	}
	if n, _ := lines[0].attr("bytes"); n.Int64() != int64(c.Info().Bytes) {
		t.Errorf("bytes attribute = %v, want %d", n, c.Info().Bytes)
	}
}

// failingClearing is a Clearing whose result the world refuses, and that fails
// the step: it makes a failed world.
type failingClearing struct{ badClearing }

// callsInStep is an observer that takes a checkpoint when a step begins.
type callsInStep struct {
	w   *economysim.World
	err error
}

// Observe takes the checkpoint at TickStarted and keeps the error.
func (o *callsInStep) Observe(e economysim.Event) {
	if _, ok := e.(economysim.TickStarted); ok {
		_, o.err = o.w.Checkpoint()
	}
}

// memo is the state of a hand-written Stateful: what SaveState returns and the
// calls it saw.
type memo struct {
	version  uint32
	state    []byte
	err      error
	saves    int
	loadedAs []uint32
}

// SaveState returns the memo's state and counts the call.
func (m *memo) SaveState() (uint32, []byte, error) {
	m.saves++
	return m.version, m.state, m.err
}

// LoadState records the version it was given.
func (m *memo) LoadState(version uint32, _ []byte) error {
	m.loadedAs = append(m.loadedAs, version)
	return nil
}

// statefulStrategy, statefulSpawner and statefulClearing are the library's
// defaults with a memo: the fakes at the extension points (architecture.md 10.3).
type statefulStrategy struct {
	economysim.DefaultStrategy
	*memo
}

type statefulSpawner struct {
	economysim.DefaultSpawner
	*memo
}

type statefulClearing struct {
	economysim.AverageClearing
	*memo
}

// TestCheckpoint_Refusals verifies the cases in which Checkpoint returns an
// error or succeeds: a failed world is ErrWorldFailed, a call inside a step is
// ErrInStep, a SaveState that fails is ErrCheckpointExtension wrapping the error,
// and a world that has not stepped can be checkpointed.
//
// Requirement: FR-CHK-002
func TestCheckpoint_Refusals(t *testing.T) {
	t.Parallel()
	errBoom := errors.New("boom")
	tests := []struct {
		name string
		run  func(t *testing.T) error
		want error
	}{
		{"failed world", func(t *testing.T) error {
			c := millCfg(1, 4)
			c.World.Clearing = "faulty"
			w, err := economysim.NewWorld(c, economysim.WithClearing("faulty", failingClearing{}))
			if err != nil {
				t.Fatalf("NewWorld() error = %v, want nil", err)
			}
			if err := w.Step(); err == nil {
				t.Fatalf("Step() error = nil, want the failure of the faulty clearing")
			}
			_, err = w.Checkpoint()
			return err
		}, economysim.ErrWorldFailed},
		{"inside a step", func(t *testing.T) error {
			o := &callsInStep{}
			w := ckptWorld(t, 0, economysim.WithObserver(o))
			o.w = w
			if err := w.Step(); err != nil {
				t.Fatalf("Step() error = %v, want nil", err)
			}
			return o.err
		}, economysim.ErrInStep},
		{"SaveState fails", func(t *testing.T) error {
			w := ckptWorld(t, 0, economysim.WithStrategy("bad", statefulStrategy{memo: &memo{err: errBoom}}))
			_, err := w.Checkpoint()
			if !errors.Is(err, errBoom) {
				t.Errorf("Checkpoint() error = %v, want it to wrap the SaveState error too", err)
			}
			return err
		}, economysim.ErrCheckpointExtension},
		{"before the first step", func(t *testing.T) error {
			_, err := ckptWorld(t, 0).Checkpoint()
			return err
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.run(t); !errors.Is(err, tt.want) && (tt.want != nil || err != nil) {
				t.Errorf("Checkpoint() error = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestCheckpoint_NoConfigOrExtensionCodeInside verifies that a checkpoint holds
// neither the configuration's own strings nor the name of a registered extension
// that keeps no state: a restore chooses the configuration and the code, and a
// checkpoint must not carry them.
//
// Requirement: FR-CHK-007
func TestCheckpoint_NoConfigOrExtensionCodeInside(t *testing.T) {
	t.Parallel()
	cfg := loadValidPack(t, packOneCold...)
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1),
		economysim.WithStrategy("plain_unused_extension", economysim.DefaultStrategy{}),
		economysim.WithSpawner("stateful_extension", statefulSpawner{memo: &memo{}}))
	if err != nil {
		t.Fatalf("NewWorld(pack one) error = %v, want nil", err)
	}
	for range 3 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
	}
	b := mustMarshal(t, mustCheckpoint(t, w))
	var ids []string
	for _, c := range cfg.Commodities {
		ids = append(ids, c.ID)
	}
	for _, r := range cfg.Roles {
		ids = append(ids, r.ID)
	}
	for _, r := range cfg.Regions {
		ids = append(ids, r.ID)
	}
	for _, r := range cfg.Recipes {
		ids = append(ids, r.ID)
	}
	for _, id := range ids {
		if len(id) >= 4 && bytes.Contains(b, []byte(id)) {
			t.Errorf("checkpoint contains the configuration string %q, want none", id)
		}
	}
	if bytes.Contains(b, []byte("plain_unused_extension")) {
		t.Errorf("checkpoint contains the name of a registered extension with no state, want none")
	}
	if bytes.Count(b, []byte("stateful_extension")) != 1 {
		t.Errorf("checkpoint holds the name of the Stateful extension %d times, want once (its entry)", bytes.Count(b, []byte("stateful_extension")))
	}
}

// TestInfo_ReportsFrame verifies that Info reports what the frame says: the
// format, the hash, the tick, the effective seed and the length, the same after
// an UnmarshalBinary, and the zero values for the zero Checkpoint.
//
// Requirement: FR-CHK-006
func TestInfo_ReportsFrame(t *testing.T) {
	t.Parallel()
	w := ckptWorld(t, 5)
	c := mustCheckpoint(t, w)
	b := mustMarshal(t, c)
	want := economysim.CheckpointInfo{Format: 1, ConfigHash: w.ConfigHash(), Tick: 5, Seed: ckptSeed, Bytes: len(b)}
	if got := c.Info(); got != want {
		t.Errorf("Info() = %+v, want %+v", got, want)
	}
	var back economysim.Checkpoint
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatalf("UnmarshalBinary() error = %v, want nil", err)
	}
	if got := back.Info(); got != want {
		t.Errorf("Info() after UnmarshalBinary = %+v, want %+v", got, want)
	}
	if got := (economysim.Checkpoint{}).Info(); got != (economysim.CheckpointInfo{}) {
		t.Errorf("Info() of the zero Checkpoint = %+v, want the zero value", got)
	}
}

// frameOf builds a frame by hand: the magic, the format, the hash version, 32
// bytes of hash, the tick and the seed, then the trailer over them.
func frameOf(format, hashVersion, tick uint64) []byte {
	b := append([]byte("ESCK"), binary.AppendUvarint(nil, format)...)
	b = binary.AppendUvarint(b, hashVersion)
	b = append(b, make([]byte, 32)...)
	b = binary.AppendUvarint(b, tick)
	b = binary.AppendUvarint(b, 9)
	return redigest(b)
}

// redigest returns b with a trailer that is the SHA-256 of b.
func redigest(b []byte) []byte {
	sum := sha256.Sum256(b)
	return append(slices.Clone(b), sum[:]...)
}

// TestUnmarshalBinary_FrameRefusals verifies that UnmarshalBinary checks the
// frame, in order: too short, the magic, the format before the digest, the
// digest and the ranges of the header, each as the sentinel of its rule, and
// that a hand-built valid frame is accepted.
//
// Arithmetic: a tick of ten bytes of 0xff and a last byte of 1 is 2^64 - 1, which
// is above the largest signed tick, 2^63 - 1.
//
// Requirement: FR-CHK-006, FR-CHK-012, FR-CHK-060, FR-CHK-063
func TestUnmarshalBinary_FrameRefusals(t *testing.T) {
	t.Parallel()
	valid := frameOf(1, hashVersionOfFormat1, 3)
	flipped := slices.Clone(valid)
	flipped[10] ^= 1
	newer := frameOf(2, hashVersionOfFormat1, 3)
	newerBadDigest := slices.Clone(newer)
	newerBadDigest[len(newerBadDigest)-1] ^= 1
	hugeTick := append([]byte("ESCK\x01\x01"), make([]byte, 32)...)
	hugeTick = append(hugeTick, bytes.Repeat([]byte{0xff}, 9)...)
	hugeTick = append(hugeTick, 1, 9)
	tests := []struct {
		name string
		in   []byte
		want error
	}{
		{"valid", valid, nil},
		{"empty", nil, economysim.ErrCheckpointCorrupt},
		{"shorter than the magic", []byte("ESC"), economysim.ErrCheckpointCorrupt},
		{"wrong magic", append([]byte("XSCK"), valid[4:]...), economysim.ErrCheckpointCorrupt},
		{"format varint cut off", []byte("ESCK\x80"), economysim.ErrCheckpointCorrupt},
		{"no room for a trailer", []byte("ESCK\x01"), economysim.ErrCheckpointCorrupt},
		{"newer format with a good digest", newer, economysim.ErrCheckpointVersion},
		{"newer format is reported before the digest", newerBadDigest, economysim.ErrCheckpointVersion},
		{"format 0", frameOf(0, hashVersionOfFormat1, 3), economysim.ErrCheckpointVersion},
		{"bit flip", flipped, economysim.ErrCheckpointCorrupt},
		{"unknown hash version", frameOf(1, 2, 3), economysim.ErrCheckpointCorrupt},
		{"header cut short", redigest([]byte("ESCK\x01\x01abc")), economysim.ErrCheckpointCorrupt},
		{"tick above the signed range", redigest(hugeTick), economysim.ErrCheckpointCorrupt},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var c economysim.Checkpoint
			err := c.UnmarshalBinary(tt.in)
			if !errors.Is(err, tt.want) && (tt.want != nil || err != nil) {
				t.Errorf("UnmarshalBinary(%x) error = %v, want %v", tt.in, err, tt.want)
			}
			if tt.want != nil && !reflect.DeepEqual(c, economysim.Checkpoint{}) {
				t.Errorf("UnmarshalBinary(%x) changed the receiver on an error, want it left as it was", tt.in)
			}
		})
	}
}

// TestCheckpoint_ImmutableCopies verifies that a Checkpoint is immutable:
// MarshalBinary and AppendBinary return copies, UnmarshalBinary copies its
// argument, and changing what they returned or were given changes nothing.
//
// Requirement: FR-CHK-003
func TestCheckpoint_ImmutableCopies(t *testing.T) {
	t.Parallel()
	c := mustCheckpoint(t, ckptWorld(t, 2))
	b := mustMarshal(t, c)
	m := slices.Clone(b)
	m[0] ^= 0xff
	if got := mustMarshal(t, c); !bytes.Equal(got, b) {
		t.Errorf("MarshalBinary() after the result was changed differs from the original, want a copy")
	}
	app, err := c.AppendBinary([]byte("prefix"))
	if err != nil || !bytes.Equal(app, append([]byte("prefix"), b...)) {
		t.Errorf("AppendBinary(prefix) = %x, %v, want the prefix and then the encoding", app, err)
	}
	app[len("prefix")] ^= 0xff
	if got := mustMarshal(t, c); !bytes.Equal(got, b) {
		t.Errorf("MarshalBinary() after AppendBinary's result was changed differs, want a copy")
	}
	in := slices.Clone(b)
	var back economysim.Checkpoint
	if err := back.UnmarshalBinary(in); err != nil {
		t.Fatalf("UnmarshalBinary() error = %v, want nil", err)
	}
	in[0] ^= 0xff
	if got := mustMarshal(t, back); !bytes.Equal(got, b) {
		t.Errorf("MarshalBinary() after the input of UnmarshalBinary was changed differs, want a copy")
	}
}

// TestOptions_Context verifies the option context rule of NewWorld: WithReseed
// is refused as ErrOptionContext whatever its salt, 0 included, and no world is
// built; WithTargetSpread and a later WithSeed are accepted. The refusal of
// WithSeed on a restore is the restore change's half.
//
// Requirement: FR-CHK-020
func TestOptions_Context(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		opts []economysim.Option
		want error
	}{
		{"WithReseed", []economysim.Option{economysim.WithReseed(5)}, economysim.ErrOptionContext},
		{"WithReseed(0) is a reseed too", []economysim.Option{economysim.WithReseed(0)}, economysim.ErrOptionContext},
		{"WithReseed after WithSeed", []economysim.Option{economysim.WithSeed(1), economysim.WithReseed(2)}, economysim.ErrOptionContext},
		{"WithTargetSpread", []economysim.Option{economysim.WithTargetSpread(100)}, nil},
		{"WithSeed and WithTargetSpread", []economysim.Option{economysim.WithSeed(3), economysim.WithTargetSpread(0)}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, err := economysim.NewWorld(millCfg(1, 4), tt.opts...)
			if !errors.Is(err, tt.want) && (tt.want != nil || err != nil) {
				t.Fatalf("NewWorld(%s) error = %v, want %v", tt.name, err, tt.want)
			}
			if (w == nil) != (tt.want != nil) {
				t.Errorf("NewWorld(%s) world = %v, want a world only without an error", tt.name, w)
			}
		})
	}
}

// entry is one entry of the section of tag 34.
type entry struct {
	kind    uint64
	name    string
	version uint64
	state   []byte
}

// entriesOf decodes the section of tag 34 of a checkpoint.
func entriesOf(t *testing.T, c economysim.Checkpoint) []entry {
	t.Helper()
	body := decodeFrame(t, mustMarshal(t, c)).sectionOf(t, 34)
	n, off := uv(t, body, 0)
	var out []entry
	for range n {
		var e entry
		var l uint64
		e.kind, off = uv(t, body, off)
		l, off = uv(t, body, off)
		e.name, off = string(body[off:off+int(l)]), off+int(l)
		e.version, off = uv(t, body, off)
		l, off = uv(t, body, off)
		e.state, off = slices.Clone(body[off:off+int(l)]), off+int(l)
		out = append(out, e)
	}
	if off != len(body) {
		t.Fatalf("section 34 has %d bytes after its entries, want none", len(body)-off)
	}
	return out
}

// TestExtensions_EntriesSortedAndCopied verifies the entries of tag 34: a
// registered Stateful strategy, spawner and clearing each have one, in the order
// of kind and then name as byte strings, with the built-in strategy "directed"
// among the strategies; SaveState is called once per entry and the world copies
// the slice it returns.
//
// Arithmetic: kinds are 0 strategy, 1 spawner and 2 clearing (spec-checkpoint.md
// 4.2.4). The names "alpha" < "directed" < "zeta" order the strategies, so the
// built-in sits between the two registered ones.
//
// Requirement: FR-CHK-031, FR-CHK-032, FR-CHK-033
func TestExtensions_EntriesSortedAndCopied(t *testing.T) {
	t.Parallel()
	zeta := &memo{version: 4, state: []byte{1, 2, 3}}
	alpha := &memo{version: 1}
	sp := &memo{version: 2, state: []byte{9}}
	cl := &memo{version: 3, state: []byte{7, 7}}
	w := ckptWorld(t, 1,
		economysim.WithStrategy("zeta", statefulStrategy{memo: zeta}),
		economysim.WithStrategy("alpha", statefulStrategy{memo: alpha}),
		economysim.WithStrategy("plain", economysim.DefaultStrategy{}),
		economysim.WithSpawner("mine", statefulSpawner{memo: sp}),
		economysim.WithClearing("beta", statefulClearing{memo: cl}))
	c := mustCheckpoint(t, w)
	want := []entry{
		{0, "alpha", 1, nil}, {0, "directed", 0, nil}, {0, "zeta", 4, []byte{1, 2, 3}},
		{1, "mine", 2, []byte{9}}, {2, "beta", 3, []byte{7, 7}},
	}
	got := entriesOf(t, c)
	for i := range got {
		if len(got[i].state) == 0 {
			got[i].state = nil
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("entries = %+v, want %+v", got, want)
	}
	for name, m := range map[string]*memo{"zeta": zeta, "alpha": alpha, "mine": sp, "beta": cl} {
		if m.saves != 1 {
			t.Errorf("SaveState of %q called %d times by one Checkpoint, want once", name, m.saves)
		}
	}
	zeta.state[0] = 99
	if got := entriesOf(t, c); got[2].state[0] != 1 {
		t.Errorf("state of the entry %q = %v after the extension changed its slice, want the copy 1 2 3", "zeta", got[2].state)
	}
}

// observerWithState is an Observer that also implements Stateful: the world must
// never ask it for state.
type observerWithState struct{ memo }

// Observe ignores the event.
func (*observerWithState) Observe(economysim.Event) {}

// TestCheckpoint_StatefulContractFlags verifies the optional nature of Stateful:
// a strategy with no state has no entry, an Observer is never saved even when it
// implements Stateful, and the version SaveState returned is written unchanged.
//
// Requirement: FR-CHK-030, FR-CHK-034
func TestCheckpoint_StatefulContractFlags(t *testing.T) {
	t.Parallel()
	obs := &observerWithState{memo{version: 5}}
	st := &memo{version: 4_000_000_000}
	w := ckptWorld(t, 0, economysim.WithObserver(obs),
		economysim.WithStrategy("plain", economysim.DefaultStrategy{}),
		economysim.WithStrategy("kept", statefulStrategy{memo: st}))
	got := entriesOf(t, mustCheckpoint(t, w))
	if len(got) != 2 || got[0].name != "directed" || got[1].name != "kept" || got[1].version != 4_000_000_000 {
		t.Errorf("entries = %+v, want \"directed\" and \"kept\" with version 4000000000 and none for \"plain\"", got)
	}
	if obs.saves != 0 {
		t.Errorf("the observer's SaveState was called %d times, want 0: an observer is never saved", obs.saves)
	}
}

// TestStateful_BuiltInDirectedEntry verifies that every world has the entry of
// the built-in strategy "directed": kind 0, version 0 and no bytes, whether or
// not a role uses it.
//
// Requirement: FR-CHK-037
func TestStateful_BuiltInDirectedEntry(t *testing.T) {
	t.Parallel()
	got := entriesOf(t, mustCheckpoint(t, ckptWorld(t, 0)))
	if len(got) != 1 || got[0].kind != 0 || got[0].name != "directed" || got[0].version != 0 || len(got[0].state) != 0 {
		t.Errorf("entries of a world with no registration = %+v, want the one entry of \"directed\" with version 0 and no bytes", got)
	}
}

// TestDirected_StatefulDelegates verifies that Directed saves and loads what its
// fallback does: a Stateful fallback's state in both directions, no state and no
// error for a fallback that has none, and a refusal of any bytes or version for a
// fallback that has none.
//
// Requirement: FR-CHK-035; N-247
func TestDirected_StatefulDelegates(t *testing.T) {
	t.Parallel()
	m := &memo{version: 6, state: []byte{1}}
	d := economysim.Directed{Fallback: statefulStrategy{memo: m}}
	if v, s, err := d.SaveState(); v != 6 || !bytes.Equal(s, []byte{1}) || err != nil {
		t.Errorf("SaveState over a Stateful fallback = %d, %x, %v, want 6, 01, nil", v, s, err)
	}
	if err := d.LoadState(6, []byte{1}); err != nil || !slices.Equal(m.loadedAs, []uint32{6}) {
		t.Errorf("LoadState over a Stateful fallback = %v and the fallback saw %v, want nil and [6]", err, m.loadedAs)
	}
	for name, plain := range map[string]economysim.Directed{
		"plain fallback": {Fallback: economysim.DefaultStrategy{}},
		"nil fallback":   {},
	} {
		if v, s, err := plain.SaveState(); v != 0 || s != nil || err != nil {
			t.Errorf("%s: SaveState = %d, %x, %v, want 0, nil, nil", name, v, s, err)
		}
		if err := plain.LoadState(0, nil); err != nil {
			t.Errorf("%s: LoadState(0, nil) = %v, want nil", name, err)
		}
		if err := plain.LoadState(0, []byte{1}); err == nil {
			t.Errorf("%s: LoadState with bytes = nil, want an error: the fallback has none to load", name)
		}
		if err := plain.LoadState(1, nil); err == nil {
			t.Errorf("%s: LoadState with version 1 = nil, want an error", name)
		}
	}
}

// TestSave_SizeIndependentOfWeights verifies that the size of a checkpoint is a
// function of the agents and not of the households they stand for: two worlds of
// one configuration with households of different weights have checkpoints of
// equal length.
//
// Arithmetic: Scale 5 and 8 households make 2 agents of weight 4, and 10
// households make 2 agents of weight 5. Each holds weight times the 1 000 start
// money of the default role and the same stock, so every number is at most
// 5 000 * 10 and stays in two varint bytes (below 16 384) at tick 0.
//
// Requirement: NFR-CHK-002
func TestSave_SizeIndependentOfWeights(t *testing.T) {
	t.Parallel()
	size := func(households int64) int {
		c := millCfg(3, households)
		c.World.Scale, c.World.MinAgents = 5, 1
		c.Roles[0].StartMoney = 500
		w, err := economysim.NewWorld(c, economysim.WithSeed(ckptSeed))
		if err != nil {
			t.Fatalf("NewWorld(%d households) error = %v, want nil", households, err)
		}
		return mustCheckpoint(t, w).Info().Bytes
	}
	if a, b := size(8), size(10); a != b {
		t.Errorf("checkpoint of 8 households (weights 4, 4) = %d bytes, of 10 (weights 5, 5) = %d bytes, want equal", a, b)
	}
}

// TestSave_AllocationOutsideTheStep verifies that taking a checkpoint adds
// nothing to what a Step allocates: the allocations of one Step before and after
// a Checkpoint are equal. Not parallel: AllocsPerRun panics in a parallel test.
//
// Requirement: NFR-CHK-001
//
// Guard: a checkpoint that kept a buffer in the world, or grew a scratch the step
// uses, would raise the baseline of the tick loop.
func TestSave_AllocationOutsideTheStep(t *testing.T) {
	w := ckptWorld(t, 12)
	step := func() {
		if err := w.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
	}
	before := testing.AllocsPerRun(5, step)
	mustCheckpoint(t, w)
	if after := testing.AllocsPerRun(5, step); after > before {
		t.Errorf("allocs per Step after a Checkpoint = %v, before = %v, want no more", after, before)
	}
}

// snapshotAtTicks are the boundaries at which TestSnapshot_FunctionOfSavedState
// compares the snapshots: before any step, after one, mid-job, across a season change.
var snapshotAtTicks = []int{0, 1, 60, 100, 150}

// TestSnapshot_FunctionOfSavedState verifies that the snapshot of a restored world
// equals the original's at the same boundary, field for field, on pack one over
// several seeds and at several ticks.
//
// Requirement: FR-CHK-053, FR-OBS-043
//
// Fix: N-125. The market rows counted the beliefs at the floor of their range, and the
// range of a made good depends on the cost anchor, which is scratch (derived at P1.2 of
// every tick, not saved). The original read the anchors left by the last step and a
// restored world had none, so AtFloor and AtCeiling differed at the boundary (55 and 0
// on seed 1 at tick 60) although every saved byte was equal.
func TestSnapshot_FunctionOfSavedState(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	for seed := uint64(1); seed <= uint64(invariantSeeds(8)); seed++ {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed))
			if err != nil {
				t.Fatalf("seed %d: NewWorld() error = %v, want nil", seed, err)
			}
			for tick := 0; tick <= snapshotAtTicks[len(snapshotAtTicks)-1]; tick++ {
				if slices.Contains(snapshotAtTicks, tick) {
					r, err := economysim.RestoreWorld(cfg, mustCheckpoint(t, w))
					if err != nil {
						t.Fatalf("seed %d, tick %d: RestoreWorld() error = %v, want nil", seed, tick, err)
					}
					want, _ := w.Snapshot()
					got, _ := r.Snapshot()
					if !reflect.DeepEqual(want, got) {
						t.Errorf("seed %d, tick %d: the snapshot of the restored world differs from the original's", seed, tick)
					}
				}
				if err := w.Step(); err != nil {
					t.Fatalf("seed %d, tick %d: Step() error = %v, want nil", seed, tick, err)
				}
			}
		})
	}
}
