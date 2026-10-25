package economysim_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2" // nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used -- only the *rand.Rand type of the injected source in a test fake (architecture.md 7.7)
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file tests the restore through the public API (plan 2-2 Phase 5): the
// sentinels, the rules of the encoding on a real checkpoint edited and re-digested,
// the audit, the extension protocol, the options and the reseed, the construction
// phase and a continuation of a restored world. The full fork-at-N matrix and the
// fixtures are Phase 6 and 7.

// reframe returns the checkpoint of raw with its sections passed through fn and its
// trailer recomputed, so a test can edit a section of a real checkpoint and still
// pass the digest.
func reframe(t *testing.T, raw []byte, fn func(secs []section) []section) economysim.Checkpoint {
	t.Helper()
	h := decodeFrame(t, raw)
	b := append([]byte("ESCK"), binary.AppendUvarint(nil, h.format)...)
	b = binary.AppendUvarint(b, h.hashVersion)
	b = append(b, h.hash[:]...)
	b = binary.AppendUvarint(b, h.tick)
	b = binary.AppendUvarint(b, h.seed)
	for _, s := range fn(slices.Clone(h.sections)) {
		b = binary.AppendUvarint(b, s.tag)
		b = binary.AppendUvarint(b, uint64(len(s.body)))
		b = append(b, s.body...)
	}
	var c economysim.Checkpoint
	if err := c.UnmarshalBinary(redigest(b)); err != nil {
		t.Fatalf("edited frame: UnmarshalBinary() error = %v, want nil (the edit must keep the frame valid)", err)
	}
	return c
}

// editSection replaces the body of the section with the tag.
func editSection(secs []section, tag uint64, fn func(body []byte) []byte) []section {
	for i := range secs {
		if secs[i].tag == tag {
			secs[i].body = fn(slices.Clone(secs[i].body))
		}
	}
	return secs
}

// varintStarts returns the offsets of the first n varints of b from off, and the
// offset after the n-th.
func varintStarts(t *testing.T, b []byte, off, n int) (starts []int, end int) {
	t.Helper()
	for range n {
		starts = append(starts, off)
		_, off = uv(t, b, off)
	}
	return starts, off
}

// setVarint replaces the varint of b at off with v.
func setVarint(t *testing.T, b []byte, off int, v uint64) []byte {
	t.Helper()
	_, next := uv(t, b, off)
	return append(append(slices.Clone(b[:off]), binary.AppendUvarint(nil, v)...), b[next:]...)
}

// agentRecords returns the offset of each field of each agent record of the body of
// tag 3: the table of names, the count of agents and then the 15 fields of a record
// (id, account, role, weight, flags, name, unfed, hunger, privation, since, carry,
// own-use, own-use jobs, journey, directive), none of which is longer than a
// varint in a world with no directive.
func agentRecords(t *testing.T, body []byte) [][]int {
	t.Helper()
	names, off := uv(t, body, 0)
	for range names {
		var l uint64
		l, off = uv(t, body, off)
		off += int(l)
	}
	n, off := uv(t, body, off)
	recs := make([][]int, n)
	for i := range recs {
		recs[i], off = varintStarts(t, body, off, 15)
	}
	return recs
}

// restoreMill checkpoints the mill at tick 5 and returns the bytes.
func restoreMillBytes(t *testing.T) []byte {
	t.Helper()
	return mustMarshal(t, mustCheckpoint(t, ckptWorld(t, 5)))
}

// TestRestore_Sentinels verifies the sentinel of each way a restore is refused
// before a section is read: the empty checkpoint and a newer format and a flipped
// digest (corrupt and version, from UnmarshalBinary), another configuration (with
// both hashes in the message), an invalid configuration (the ValidationError
// NewWorld gives), WithSeed (option context), a strategy name nobody registered, a
// registered Stateful extension with no entry, an entry nobody registered, and a
// LoadState that fails.
//
// Requirement: FR-CHK-006, FR-CHK-018, FR-CHK-019, FR-CHK-020, FR-CHK-021, FR-CHK-022, FR-CHK-060
func TestRestore_Sentinels(t *testing.T) {
	t.Parallel()
	raw := restoreMillBytes(t)
	good := func() economysim.Checkpoint {
		var c economysim.Checkpoint
		if err := c.UnmarshalBinary(raw); err != nil {
			t.Fatalf("UnmarshalBinary() error = %v, want nil", err)
		}
		return c
	}
	withZeta := economysim.WithStrategy("zeta", statefulStrategy{memo: &memo{version: 1, state: []byte{1}}})
	zetaRaw := mustMarshal(t, mustCheckpoint(t, ckptWorld(t, 2, withZeta)))
	var zetaCk economysim.Checkpoint
	if err := zetaCk.UnmarshalBinary(zetaRaw); err != nil {
		t.Fatalf("UnmarshalBinary() error = %v, want nil", err)
	}
	ghost := reframe(t, raw, func(s []section) []section {
		return editSection(s, 3, func(b []byte) []byte {
			// The table has one name of seven bytes ("default"): one name of five.
			return append([]byte{1, 5, 'g', 'h', 'o', 's', 't'}, b[1+1+len("default"):]...)
		})
	})
	failing := errors.New("refused by the extension")
	tests := []struct {
		name string
		run  func() error
		want []error
	}{
		{"zero checkpoint", func() error {
			_, err := economysim.RestoreWorld(millCfg(3, 4), economysim.Checkpoint{})
			return err
		}, []error{economysim.ErrCheckpointCorrupt}},
		{"another configuration", func() error {
			_, err := economysim.RestoreWorld(millCfg(3, 5), good())
			return err
		}, []error{economysim.ErrCheckpointConfig}},
		{"WithSeed on a restore", func() error {
			_, err := economysim.RestoreWorld(millCfg(3, 4), good(), economysim.WithSeed(1))
			return err
		}, []error{economysim.ErrOptionContext}},
		{"strategy nobody registered", func() error {
			_, err := economysim.RestoreWorld(millCfg(3, 4), ghost)
			return err
		}, []error{economysim.ErrCheckpointExtension, economysim.ErrUnknownStrategy}},
		{"stateful extension with no entry", func() error {
			_, err := economysim.RestoreWorld(millCfg(3, 4), good(), withZeta)
			return err
		}, []error{economysim.ErrCheckpointExtension}},
		{"entry nobody registered", func() error {
			_, err := economysim.RestoreWorld(millCfg(3, 4), zetaCk)
			return err
		}, []error{economysim.ErrCheckpointExtension}},
		{"LoadState fails", func() error {
			_, err := economysim.RestoreWorld(millCfg(3, 4), zetaCk,
				economysim.WithStrategy("zeta", &loaderStrategy{loader: &loader{err: failing}}))
			return err
		}, []error{economysim.ErrCheckpointExtension, failing}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.run()
			for _, want := range tt.want {
				if !errors.Is(err, want) {
					t.Errorf("RestoreWorld() error = %v, want it to match %v", err, want)
				}
			}
		})
	}
	t.Run("configuration hashes are named", func(t *testing.T) {
		t.Parallel()
		_, err := economysim.RestoreWorld(millCfg(3, 5), good())
		other, werr := economysim.NewWorld(millCfg(3, 5))
		if werr != nil {
			t.Fatalf("NewWorld() error = %v, want nil", werr)
		}
		h := other.ConfigHash()
		if err == nil || !strings.Contains(err.Error(), hex.EncodeToString(h[:])) {
			t.Errorf("RestoreWorld() error = %v, want it to name the configuration hash %x", err, h)
		}
	})
	t.Run("invalid configuration is the error of NewWorld", func(t *testing.T) {
		t.Parallel()
		bad := millCfg(3, 4)
		bad.Commodities = nil
		_, want := economysim.NewWorld(bad)
		_, err := economysim.RestoreWorld(bad, good())
		var ve *economysim.ValidationError
		if !errors.As(err, &ve) || want == nil {
			t.Errorf("RestoreWorld() error = %v, want a *ValidationError like NewWorld's %v", err, want)
		}
	})
	t.Run("newer format is a version error naming both", func(t *testing.T) {
		t.Parallel()
		var c economysim.Checkpoint
		err := c.UnmarshalBinary(frameOf(2, hashVersionOfFormat1, 3))
		if !errors.Is(err, economysim.ErrCheckpointVersion) || !strings.Contains(err.Error(), "2") || !strings.Contains(err.Error(), "1") {
			t.Errorf("UnmarshalBinary(format 2) error = %v, want ErrCheckpointVersion naming 2 and 1", err)
		}
	})
	t.Run("flipped digest is corrupt", func(t *testing.T) {
		t.Parallel()
		flipped := slices.Clone(raw)
		flipped[len(flipped)-1] ^= 1
		var c economysim.Checkpoint
		if err := c.UnmarshalBinary(flipped); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
			t.Errorf("UnmarshalBinary(flipped digest) error = %v, want ErrCheckpointCorrupt", err)
		}
	})
}

// loader is a hand-written Stateful that logs the order of its LoadState calls
// and keeps what it was given.
type loader struct {
	name    string
	log     *[]string
	version uint32
	state   []byte
	err     error
	gotV    uint32
	got     []byte
	loads   int
}

// SaveState returns the loader's state.
func (l *loader) SaveState() (uint32, []byte, error) { return l.version, l.state, nil }

// LoadState records the call.
func (l *loader) LoadState(v uint32, b []byte) error {
	if l.log != nil {
		*l.log = append(*l.log, l.name)
	}
	l.loads++
	l.gotV, l.got = v, b
	return l.err
}

// loaderStrategy, loaderSpawner and loaderClearing are the defaults with a loader.
type loaderStrategy struct {
	economysim.DefaultStrategy
	*loader
}

type loaderSpawner struct {
	economysim.DefaultSpawner
	*loader
}

type loaderClearing struct {
	economysim.AverageClearing
	*loader
}

// TestRestore_FailureReturnsNilAndEmitsNothing verifies that a restore that fails
// returns no world and delivers nothing to the observer and the logger it was
// given, at the first check (another configuration), in a section (corrupt) and at
// the last step (a LoadState error).
//
// Requirement: FR-CHK-017, FR-CHK-080
func TestRestore_FailureReturnsNilAndEmitsNothing(t *testing.T) {
	t.Parallel()
	raw := restoreMillBytes(t)
	corrupt := reframe(t, raw, func(s []section) []section {
		return editSection(s, 9, func([]byte) []byte { return []byte{1, 0, 0, 0, 0, 0, 0, 0} })
	})
	var good economysim.Checkpoint
	if err := good.UnmarshalBinary(raw); err != nil {
		t.Fatalf("UnmarshalBinary() error = %v, want nil", err)
	}
	zetaCk := mustCheckpoint(t, ckptWorld(t, 2, economysim.WithStrategy("zeta", statefulStrategy{memo: &memo{}})))
	tests := []struct {
		name string
		cfg  economysim.Config
		ck   economysim.Checkpoint
		opts []economysim.Option
	}{
		{"another configuration", millCfg(3, 5), good, nil},
		{"corrupt section", millCfg(3, 4), corrupt, nil},
		{"LoadState error", millCfg(3, 4), zetaCk, []economysim.Option{
			economysim.WithStrategy("zeta", &loaderStrategy{loader: &loader{err: errors.New("no")}})}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := &recorder{}
			h := newCaptureHandler()
			opts := append([]economysim.Option{economysim.WithObserver(rec), economysim.WithLogger(slog.New(h))}, tt.opts...)
			w, err := economysim.RestoreWorld(tt.cfg, tt.ck, opts...)
			if err == nil || w != nil {
				t.Fatalf("RestoreWorld() = (%v, %v), want (nil, an error)", w, err)
			}
			if len(rec.events) != 0 || len(h.records()) != 0 {
				t.Errorf("a failed restore delivered %d events and %d log lines, want none", len(rec.events), len(h.records()))
			}
		})
	}
}

// TestRestore_Corrupt verifies the rules of the encoding on a real checkpoint of the
// mill at tick 5, edited in one place and re-digested so that only the section
// reader can find the fault: a non-minimal varint, a count above the bytes left, a
// bad boolean, a value above its bound, an agent ID repeated, descending and below
// the estates, an agent account that is another agent's, a hosted field that must
// be 0 while its module is off, a sparse row at its default, a missing tag, a
// surplus section, a section that ends early and one that runs over, a bit that a
// live agent needs unset, and a listing for a lot the estate does not hold. Each is
// ErrCheckpointCorrupt.
//
// Arithmetic: the mill has one estate (agent 0) and four households (agents 1 to
// 4), so the first two records of tag 3 have the IDs 1 and 2; MaxQuantity is
// 1 000 000 000, and 1 000 000 001 is one above it.
//
// Requirement: FR-CHK-008, FR-CHK-012, FR-CHK-013, FR-CHK-014, FR-CHK-023, FR-CHK-049
func TestRestore_Corrupt(t *testing.T) {
	t.Parallel()
	raw := restoreMillBytes(t)
	field := func(rec, f int, v uint64) func(t *testing.T, b []byte) []byte {
		return func(t *testing.T, b []byte) []byte { return setVarint(t, b, agentRecords(t, b)[rec][f], v) }
	}
	fieldOf := func(rec, f int, from int) func(t *testing.T, b []byte) []byte {
		return func(t *testing.T, b []byte) []byte {
			recs := agentRecords(t, b)
			v, _ := uv(t, b, recs[from][f])
			return setVarint(t, b, recs[rec][f], v)
		}
	}
	constant := func(body ...byte) func(*testing.T, []byte) []byte {
		return func(*testing.T, []byte) []byte { return body }
	}
	tests := []struct {
		name string
		tag  uint64
		edit func(t *testing.T, b []byte) []byte
	}{
		{"non-minimal varint", 1, func(_ *testing.T, b []byte) []byte { return append([]byte{b[0] | 0x80, 0}, b[1:]...) }},
		{"boolean that is neither", 1, func(_ *testing.T, b []byte) []byte { b[len(b)-1] = 2; return b }},
		{"count above the bytes left", 3, func(_ *testing.T, b []byte) []byte { return append(binary.AppendUvarint(nil, 1<<40), b[1:]...) }},
		{"market sum above int64", 4, func(t *testing.T, b []byte) []byte {
			starts, _ := varintStarts(t, b, 0, 3)
			return setVarint(t, b, starts[2], uint64(math.MaxInt64)+1)
		}},
		{"agent ID repeated", 3, fieldOf(1, 0, 0)},
		{"agent ID descending", 3, field(0, 0, 9)},
		{"agent ID below the estates", 3, field(0, 0, 0)},
		{"account of another agent", 3, fieldOf(0, 1, 1)},
		{"own-use state while the module is off", 3, field(0, 11, 1)},
		{"journey while the module is off", 3, field(0, 13, 1)},
		{"death carry that is not its start", 3, field(0, 10, 2)},
		{"weight 0", 3, field(0, 3, 0)},
		{"role out of range", 3, field(0, 2, 7)},
		{"flags set", 3, field(0, 4, 1)},
		{"unfed above the weight", 3, field(0, 6, 5)},
		{"sparse row at its default", 9, constant(1, 0, 0, 0, 0, 0, 0, 0)},
		{"had-an-agent bit unset for a live role", 9, constant(0, 0, 0, 0, 0)},
		{"listing for a lot the estate does not hold", 8, constant(1, 0, 0, 0, 5)},
		{"section that ends early", 5, func(_ *testing.T, b []byte) []byte { return b[:len(b)-1] }},
		{"section that runs over", 5, func(_ *testing.T, b []byte) []byte { return append(b, 0) }},
		{"rows in a skeleton", 11, constant(1)},
		{"rows in the skills skeleton", 10, constant(0, 0, 1, 0)},
		{"welfare field beyond the clamp", 7, func(t *testing.T, b []byte) []byte {
			// The body starts with the count of records of agent 1, then its first record.
			starts, _ := varintStarts(t, b, 0, 2)
			if b[starts[0]] == 0 {
				t.Fatalf("agent 1 holds no welfare record at tick 5, want at least one")
			}
			return setVarint(t, b, starts[1], 1<<62)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := reframe(t, raw, func(s []section) []section {
				return editSection(s, tt.tag, func(b []byte) []byte { return tt.edit(t, b) })
			})
			if _, err := economysim.RestoreWorld(millCfg(3, 4), c); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt", tt.name, err)
			}
		})
	}
	frames := []struct {
		name string
		fn   func(s []section) []section
	}{
		{"missing tag", func(s []section) []section { return slices.DeleteFunc(s, func(x section) bool { return x.tag == 5 }) }},
		{"surplus section", func(s []section) []section { return append(s, section{tag: 12}) }},
		{"sections out of order", func(s []section) []section { s[3], s[4] = s[4], s[3]; return s }},
	}
	for _, tt := range frames {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := reframe(t, raw, tt.fn)
			if _, err := economysim.RestoreWorld(millCfg(3, 4), c); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt", tt.name, err)
			}
		})
	}
}

// TestRestore_RunsAboveTheirBound verifies that a sparse run above the ticks
// elapsed (6 at tick 5) and a renewal carry at or above 10 000 are corrupt, each
// found by the bound on the value and not by another rule.
//
// Arithmetic: the unfed run closes tag 3 and the clamp run closes tag 7, each
// written as a count of 0 at tick 5 in the mill; the rows (0, 0, 6) replace it.
// The renewal carry opens tag 9 as the row (0, 0, 10 000), 10 000 being 0x90 0x4e.
//
// Requirement: FR-CHK-013
func TestRestore_RunsAboveTheirBound(t *testing.T) {
	t.Parallel()
	raw := restoreMillBytes(t)
	endRow := func(b []byte) []byte {
		if b[len(b)-1] != 0 {
			t.Fatalf("section ends in %d, want a count of 0", b[len(b)-1])
		}
		return append(b[:len(b)-1:len(b)-1], 1, 0, 0, 6)
	}
	tests := []struct {
		name string
		tag  uint64
		edit func([]byte) []byte
		want string
	}{
		{"unfed run", 3, endRow, "run 6 is above 5"},
		{"clamp run", 7, endRow, "run 6 is above 5"},
		{"renewal carry", 9, func([]byte) []byte { return []byte{1, 0, 0, 0x90, 0x4e} }, "run 10000 is above 9999"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := reframe(t, raw, func(s []section) []section { return editSection(s, tt.tag, tt.edit) })
			_, err := economysim.RestoreWorld(millCfg(3, 4), c)
			if !errors.Is(err, economysim.ErrCheckpointCorrupt) || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt naming %q", tt.name, err, tt.want)
			}
		})
	}
}

// TestRestore_UnusedStrategyNameIsCorrupt verifies that a table of strategy names
// with a registered name no agent uses is corrupt, since the table is the sorted
// set of the names agents use and one world has one byte string (R10).
//
// Arithmetic: every agent of the mill uses "default", so the table is the count 1,
// the length 7 and the name; "directed" (length 8) is registered and sorts after it.
//
// Requirement: FR-CHK-012
func TestRestore_UnusedStrategyNameIsCorrupt(t *testing.T) {
	t.Parallel()
	c := reframe(t, restoreMillBytes(t), func(s []section) []section {
		return editSection(s, 3, func(b []byte) []byte {
			head := append([]byte{2, 7}, "default"...)
			if !bytes.HasPrefix(b, append([]byte{1, 7}, "default"...)) {
				t.Fatalf("tag 3 starts % x, want the table of one name, default", b[:9])
			}
			head = append(append(head, 8), "directed"...)
			return append(head, b[9:]...)
		})
	})
	if _, err := economysim.RestoreWorld(millCfg(3, 4), c); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
		t.Errorf("RestoreWorld() error = %v, want ErrCheckpointCorrupt", err)
	}
}

// TestRestore_UnreadableStrategyTableIsCorrupt verifies that a table of strategy
// names that ends early, a count of 2 with one name or a name longer than the
// bytes left, is ErrCheckpointCorrupt and not a fault of the extensions.
//
// Requirement: FR-CHK-012
func TestRestore_UnreadableStrategyTableIsCorrupt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		table []byte
	}{
		{"count above the names", append([]byte{2, 7}, "default"...)},
		{"name beyond the bytes left", []byte{1, 9, 'd'}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := reframe(t, restoreMillBytes(t), func(s []section) []section {
				return editSection(s, 3, func([]byte) []byte { return tt.table })
			})
			_, err := economysim.RestoreWorld(millCfg(3, 4), c)
			if !errors.Is(err, economysim.ErrCheckpointCorrupt) || errors.Is(err, economysim.ErrCheckpointExtension) {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt only", tt.name, err)
			}
		})
	}
}

// TestRestore_AuditAlwaysRuns verifies that a checkpoint whose ledger no longer
// conserves money (one minor unit added to the balance of the estate, edited and
// re-digested) is refused as ErrCheckpointCorrupt wrapping ErrAuditFailed, with
// WithAudit not given.
//
// Arithmetic: tag 2 starts with the count of accounts and then the first account,
// the estate of the only region: kind 1, closed 0, region 0, owner 0 and then its
// balance, which is 0 before the edit and 1 after.
//
// Requirement: FR-CHK-024
func TestRestore_AuditAlwaysRuns(t *testing.T) {
	t.Parallel()
	c := reframe(t, restoreMillBytes(t), func(s []section) []section {
		return editSection(s, 2, func(b []byte) []byte {
			starts, _ := varintStarts(t, b, 0, 6)
			if b[starts[1]] != 1 {
				t.Fatalf("first non-boundary account has kind %d, want 1 (the estate)", b[starts[1]])
			}
			return setVarint(t, b, starts[5], 1)
		})
	})
	_, err := economysim.RestoreWorld(millCfg(3, 4), c)
	if !errors.Is(err, economysim.ErrCheckpointCorrupt) || !errors.Is(err, economysim.ErrAuditFailed) {
		t.Errorf("RestoreWorld() error = %v, want ErrCheckpointCorrupt wrapping ErrAuditFailed", err)
	}
}

// TestRestore_ExtensionOrderAndLoadLast verifies step 11: LoadState is called once
// per entry in (kind, name) order with the version and the bytes SaveState wrote,
// and only after every other step succeeded, so a checkpoint that fails a section
// leaves every instance untouched.
//
// Requirement: FR-CHK-029, FR-CHK-034
func TestRestore_ExtensionOrderAndLoadLast(t *testing.T) {
	t.Parallel()
	var order []string
	mk := func(name string, v uint32, st ...byte) *loader {
		return &loader{name: name, log: &order, version: v, state: st}
	}
	build := func(zeta, alpha, mine, beta *loader) []economysim.Option {
		return []economysim.Option{
			economysim.WithStrategy("zeta", &loaderStrategy{loader: zeta}),
			economysim.WithStrategy("alpha", &loaderStrategy{loader: alpha}),
			economysim.WithSpawner("mine", &loaderSpawner{loader: mine}),
			economysim.WithClearing("beta", &loaderClearing{loader: beta}),
		}
	}
	z, a, m, b := mk("zeta", 4, 1, 2, 3), mk("alpha", 1), mk("mine", 2, 9), mk("beta", 3, 7, 7)
	ck := mustCheckpoint(t, ckptWorld(t, 3, build(z, a, m, b)...))
	z2, a2, m2, b2 := mk("zeta", 0), mk("alpha", 0), mk("mine", 0), mk("beta", 0)
	order = nil
	if _, err := economysim.RestoreWorld(millCfg(3, 4), ck, build(z2, a2, m2, b2)...); err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	if want := []string{"alpha", "zeta", "mine", "beta"}; !slices.Equal(order, want) {
		t.Errorf("LoadState order = %v, want %v (kind, then name)", order, want)
	}
	if z2.gotV != 4 || !bytes.Equal(z2.got, []byte{1, 2, 3}) || m2.gotV != 2 || !bytes.Equal(m2.got, []byte{9}) {
		t.Errorf("LoadState was given (%d, %v) and (%d, %v), want (4, [1 2 3]) and (2, [9])", z2.gotV, z2.got, m2.gotV, m2.got)
	}
	raw := mustMarshal(t, ck)
	bad := reframe(t, raw, func(s []section) []section {
		return editSection(s, 9, func([]byte) []byte { return []byte{1, 0, 0, 0, 0, 0, 0, 0} })
	})
	z3, a3, m3, b3 := mk("zeta", 0), mk("alpha", 0), mk("mine", 0), mk("beta", 0)
	if _, err := economysim.RestoreWorld(millCfg(3, 4), bad, build(z3, a3, m3, b3)...); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
		t.Fatalf("RestoreWorld(corrupt) error = %v, want ErrCheckpointCorrupt", err)
	}
	if z3.loads+a3.loads+m3.loads+b3.loads != 0 {
		t.Errorf("LoadState was called %d times on a restore that failed a section, want 0", z3.loads+a3.loads+m3.loads+b3.loads)
	}
}

// TestRestore_TwoReadersAfterFailure verifies that after a restore that failed at
// step 11 a new restore with new instances succeeds: the checkpoint is unchanged.
//
// Requirement: FR-CHK-017
func TestRestore_TwoReadersAfterFailure(t *testing.T) {
	t.Parallel()
	zeta := func(l *loader) economysim.Option { return economysim.WithStrategy("zeta", &loaderStrategy{loader: l}) }
	ck := mustCheckpoint(t, ckptWorld(t, 2, zeta(&loader{state: []byte{5}})))
	if _, err := economysim.RestoreWorld(millCfg(3, 4), ck, zeta(&loader{err: errors.New("no")})); !errors.Is(err, economysim.ErrCheckpointExtension) {
		t.Fatalf("first RestoreWorld() error = %v, want ErrCheckpointExtension", err)
	}
	ok := &loader{}
	if _, err := economysim.RestoreWorld(millCfg(3, 4), ck, zeta(ok)); err != nil || !bytes.Equal(ok.got, []byte{5}) {
		t.Errorf("second RestoreWorld() = (%v, state %v), want success with the state [5]", err, ok.got)
	}
}

// TestRestore_AliasedStateIsACopy verifies that the bytes LoadState receives are a
// copy: an extension that writes into them does not change the checkpoint.
//
// Requirement: FR-CHK-003, FR-CHK-034
func TestRestore_AliasedStateIsACopy(t *testing.T) {
	t.Parallel()
	zeta := func(l *loader) economysim.Option { return economysim.WithStrategy("zeta", &loaderStrategy{loader: l}) }
	ck := mustCheckpoint(t, ckptWorld(t, 2, zeta(&loader{state: []byte{5, 6}})))
	before := mustMarshal(t, ck)
	l := &loader{}
	if _, err := economysim.RestoreWorld(millCfg(3, 4), ck, zeta(l)); err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	l.got[0] = 99
	if !bytes.Equal(before, mustMarshal(t, ck)) {
		t.Errorf("the checkpoint changed when the extension wrote into the state it was given")
	}
}

// TestRestore_Reseed verifies the option on the sixteen vectors of spec-checkpoint.md
// 17.5: the restored world's own checkpoint carries seed' = finalise(seed xor
// finalise(salt)), and WorldRestored reports the reseed and the salt, also for seed
// 0 with salt 0; a world restored without the option keeps the header's seed; one
// salt twice gives equal checkpoints and two salts differ; and a reseeded world's
// checkpoint restores with no option to the same seed.
//
// Requirement: FR-CHK-070, FR-CHK-071, FR-CHK-072
func TestRestore_Reseed(t *testing.T) {
	t.Parallel()
	vectors := []struct {
		seed, salt, want uint64
	}{
		{0, 0, 0}, {0, 1, 0x7ab40e090f363a7d}, {0, 2, 0x52ead7e36ea7fea8}, {0, 42, 0x97ea87f7e45c00a5},
		{1, 0, 0x5692161d100b05e5}, {1, 1, 0x83ec686c1600460a}, {1, 2, 0x3ec2d42f3a45cc6e}, {1, 42, 0x901883d2427428f4},
		{42, 0, 0xa759ea27d4727622}, {42, 1, 0xc2a6eebdf3976ad0}, {42, 2, 0x7bfa87c92aa0cff0}, {42, 42, 0x36cdb0593534a37b},
		{1<<64 - 1, 0, 0xb4d055fcf2cbbd7b}, {1<<64 - 1, 1, 0xf3759af4852e5904},
		{1<<64 - 1, 2, 0xab47721b00f39ddb}, {1<<64 - 1, 42, 0xeab06b8e3b2074d7},
	}
	for _, v := range vectors {
		t.Run(fmt.Sprintf("seed %d salt %d", v.seed, v.salt), func(t *testing.T) {
			t.Parallel()
			w, err := economysim.NewWorld(millCfg(3, 4), economysim.WithSeed(v.seed))
			if err != nil {
				t.Fatalf("NewWorld() error = %v, want nil", err)
			}
			rec := &recorder{}
			r, err := economysim.RestoreWorld(millCfg(3, 4), mustCheckpoint(t, w), economysim.WithReseed(v.salt), economysim.WithObserver(rec))
			if err != nil {
				t.Fatalf("RestoreWorld(WithReseed(%d)) error = %v, want nil", v.salt, err)
			}
			ck := mustCheckpoint(t, r)
			if got := ck.Info().Seed; got != v.want {
				t.Errorf("seed of the restored world's checkpoint = %#x, want %#x", got, v.want)
			}
			ev, ok := rec.events[0].(economysim.WorldRestored)
			if !ok || !ev.Reseeded || ev.Salt != v.salt || ev.Snapshot.Seed != v.want {
				t.Errorf("first event = %+v, want WorldRestored with Reseeded, Salt %d and Snapshot.Seed %#x", rec.events[0], v.salt, v.want)
			}
			again, err := economysim.RestoreWorld(millCfg(3, 4), ck)
			if err != nil || mustCheckpoint(t, again).Info().Seed != v.want {
				t.Errorf("restore of the reseeded checkpoint with no option = (%v), want success with seed %#x", err, v.want)
			}
		})
	}
	t.Run("no option keeps the seed, salts differ and one salt twice is equal", func(t *testing.T) {
		t.Parallel()
		w := ckptWorld(t, 3)
		ck := mustCheckpoint(t, w)
		plain, err := economysim.RestoreWorld(millCfg(3, 4), ck)
		if err != nil || mustCheckpoint(t, plain).Info().Seed != ckptSeed {
			t.Fatalf("restore with no reseed = (%v), want seed %d", err, ckptSeed)
		}
		r1 := mustMarshal(t, mustCheckpoint(t, mustRestore(t, ck, economysim.WithReseed(7))))
		r1b := mustMarshal(t, mustCheckpoint(t, mustRestore(t, ck, economysim.WithReseed(7))))
		r2 := mustMarshal(t, mustCheckpoint(t, mustRestore(t, ck, economysim.WithReseed(8))))
		if !bytes.Equal(r1, r1b) || bytes.Equal(r1, r2) {
			t.Errorf("checkpoints of salt 7 twice equal = %v, of salts 7 and 8 equal = %v, want true and false", bytes.Equal(r1, r1b), bytes.Equal(r1, r2))
		}
	})
}

// mustRestore restores the mill from ck with the options.
func mustRestore(t *testing.T, ck economysim.Checkpoint, opts ...economysim.Option) *economysim.World {
	t.Helper()
	w, err := economysim.RestoreWorld(millCfg(3, 4), ck, opts...)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	return w
}

// TestRestore_InConstruction verifies the construction phase across a restore: a
// world checkpointed before its first Step is restored still in construction and
// emits its bucket as BoundaryFlow events with Tick -1 at its first step, as the
// original does; one checkpointed after the first Step emits none.
//
// Requirement: FR-CHK-026
func TestRestore_InConstruction(t *testing.T) {
	t.Parallel()
	bucketTicks := func(rec *recorder) (n int) {
		for _, e := range rec.events {
			if b, ok := e.(economysim.BoundaryFlow); ok && b.Tick == -1 {
				n++
			}
		}
		return n
	}
	orig := &recorder{}
	w := ckptWorld(t, 0, economysim.WithObserver(orig))
	ck := mustCheckpoint(t, w)
	before := &recorder{}
	r := mustRestore(t, ck, economysim.WithObserver(before))
	for _, x := range []*economysim.World{w, r} {
		if err := x.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	if o, b := bucketTicks(orig), bucketTicks(before); o == 0 || o != b {
		t.Errorf("bucket events at the first step: original %d, restored %d, want equal and above 0", o, b)
	}
	after := &recorder{}
	r2 := mustRestore(t, mustCheckpoint(t, r), economysim.WithObserver(after))
	if err := r2.Step(); err != nil {
		t.Fatalf("Step() error = %v, want nil", err)
	}
	if n := bucketTicks(after); n != 0 {
		t.Errorf("a world restored after its first step emitted %d bucket events, want none", n)
	}
}

// TestRestore_ConfigAndHash verifies that the restored world reports the effective
// configuration and the hash of the header, and the tick and seed.
//
// Requirement: FR-CHK-082
func TestRestore_ConfigAndHash(t *testing.T) {
	t.Parallel()
	w := ckptWorld(t, 4)
	r := mustRestore(t, mustCheckpoint(t, w))
	if r.ConfigHash() != w.ConfigHash() || !reflect.DeepEqual(r.Config(), w.Config()) || r.Tick() != 4 || r.Err() != nil {
		t.Errorf("restored world: hash %x tick %d err %v, want the original's %x, 4 and nil, with an equal Config()", r.ConfigHash(), r.Tick(), r.Err(), w.ConfigHash())
	}
}

// TestRestore_SharesNothing verifies that two worlds restored from one checkpoint
// and the original share no state: stepping one leaves the others as they were.
//
// Requirement: FR-CHK-003, FR-CHK-025
func TestRestore_SharesNothing(t *testing.T) {
	t.Parallel()
	w := ckptWorld(t, 4)
	ck := mustCheckpoint(t, w)
	a, b := mustRestore(t, ck), mustRestore(t, ck)
	want := mustMarshal(t, mustCheckpoint(t, b))
	for range 3 {
		if err := a.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	if got := mustMarshal(t, mustCheckpoint(t, b)); !bytes.Equal(got, want) {
		t.Errorf("a sibling restored from one checkpoint changed when the other was stepped")
	}
	if !bytes.Equal(mustMarshal(t, ck), mustMarshal(t, mustCheckpoint(t, ckptWorld(t, 4)))) {
		t.Errorf("the checkpoint changed after two restores and three steps")
	}
}

// follower is a world restored at a boundary of the run, stepped beside the original
// for a few ticks: the events the original delivered after the boundary start at mark.
type follower struct {
	w    *economysim.World
	rec  *recorder
	mark int
	last int
}

// TestRestore_ResaveIsAFixedPointAndContinues verifies, for the mill and for pack
// one of the reference chain, that a world restored at a boundary of a run checkpoints
// to the same bytes (the one spelling, FR-CHK-005), and that it then steps beside the
// original to the same bytes, snapshot and events for ten ticks: nothing the
// continuation needs is missing from the sections. The long matrix with the fork
// helper, the logs and the observers added later is Phase 6.
//
// Requirement: FR-CHK-005, FR-CHK-050, FR-CHK-051
func TestRestore_ResaveIsAFixedPointAndContinues(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, tt := range []struct {
		name  string
		cfg   economysim.Config
		ticks int
		every int
	}{
		{"mill", millCfg(3, 4), 30, 1},
		{"pack one", loadValidPack(t, packOne...), 60, 7},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			orig := &recorder{}
			w, err := economysim.NewWorld(tt.cfg, economysim.WithSeed(3), economysim.WithObserver(orig))
			if err != nil {
				t.Fatalf("NewWorld() error = %v, want nil", err)
			}
			var followers []*follower
			for tick := range tt.ticks {
				if tick%tt.every == 0 {
					ck := mustCheckpoint(t, w)
					rec := &recorder{}
					r, err := economysim.RestoreWorld(tt.cfg, ck, economysim.WithObserver(rec))
					if err != nil {
						t.Fatalf("tick %d: RestoreWorld() error = %v, want nil", tick, err)
					}
					if !bytes.Equal(mustMarshal(t, ck), mustMarshal(t, mustCheckpoint(t, r))) {
						t.Fatalf("tick %d: the restored world's checkpoint differs from the one it was restored from", tick)
					}
					followers = append(followers, &follower{w: r, rec: rec, mark: len(orig.events), last: tick + 10})
				}
				if err := w.Step(); err != nil {
					t.Fatalf("tick %d: Step() error = %v, want nil", tick, err)
				}
				want := mustMarshal(t, mustCheckpoint(t, w))
				wantSnap, _ := w.Snapshot()
				for _, f := range followers {
					if tick >= f.last {
						continue
					}
					if err := f.w.Step(); err != nil {
						t.Fatalf("tick %d: Step() of a world restored at %d error = %v, want nil", tick, f.last-10, err)
					}
					gotSnap, _ := f.w.Snapshot()
					switch {
					case !bytes.Equal(want, mustMarshal(t, mustCheckpoint(t, f.w))):
						t.Fatalf("tick %d: the checkpoint of a world restored at %d differs from the original's", tick, f.last-10)
					case !reflect.DeepEqual(wantSnap, gotSnap):
						t.Fatalf("tick %d: the snapshot of a world restored at %d differs from the original's", tick, f.last-10)
					case !reflect.DeepEqual(orig.events[f.mark:], f.rec.events[1:]):
						t.Fatalf("tick %d: the events of a world restored at %d differ from the original's since the boundary", tick, f.last-10)
					}
				}
			}
		})
	}
}

// TestRestore_MillContinuesWithAnObserver verifies, at the integration level, that
// a world restored from the mill at tick 4 steps six more ticks to the same
// checkpoint bytes, snapshot and events as the original, with an Observer wired
// to the restored world and the first event it sees being the restore's own.
//
// Requirement: FR-CHK-050, FR-CHK-051
func TestRestore_MillContinuesWithAnObserver(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	orig := &recorder{}
	w, err := economysim.NewWorld(millCfg(3, 4), economysim.WithSeed(3), economysim.WithObserver(orig))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	for range 4 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	mark := len(orig.events)
	rec := &recorder{}
	r, err := economysim.RestoreWorld(millCfg(3, 4), mustCheckpoint(t, w), economysim.WithObserver(rec))
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	for tick := 4; tick < 10; tick++ {
		for _, x := range []*economysim.World{w, r} {
			if err := x.Step(); err != nil {
				t.Fatalf("tick %d: Step() error = %v, want nil", tick, err)
			}
		}
		ws, _ := w.Snapshot()
		rs, _ := r.Snapshot()
		if !reflect.DeepEqual(ws, rs) || !bytes.Equal(mustMarshal(t, mustCheckpoint(t, w)), mustMarshal(t, mustCheckpoint(t, r))) {
			t.Fatalf("tick %d: the restored world's snapshot or checkpoint differs from the original's", tick)
		}
	}
	if len(rec.events) < 2 || !reflect.DeepEqual(orig.events[mark:], rec.events[1:]) {
		t.Errorf("events after the restore = %d, want the restore event and then the original's %d events", len(rec.events), len(orig.events)-mark)
	}
}

// hiddenCounter is a Strategy that skips its Trade on some calls, by a counter. With
// state set it saves the counter; without, it keeps it in a field that no checkpoint
// holds.
type hiddenCounter struct {
	economysim.DefaultStrategy
	n     *int
	saved bool
}

// Trade skips the trade of two calls in every five.
func (h hiddenCounter) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	*h.n++
	if *h.n%5 < 2 {
		return
	}
	h.DefaultStrategy.Trade(v, rng, p)
}

// SaveState writes the counter when the strategy declares it.
func (h hiddenCounter) SaveState() (uint32, []byte, error) {
	if !h.saved {
		return 0, nil, nil
	}
	return 1, []byte{byte(*h.n)}, nil
}

// LoadState restores the counter.
func (h hiddenCounter) LoadState(_ uint32, b []byte) error {
	if h.saved {
		*h.n = int(b[0])
	}
	return nil
}

// TestRestore_StatefulHiddenState verifies the contract of Stateful end to end: an
// extension that saves its counter continues exactly as the original does, and one
// that keeps it where no checkpoint holds it diverges, which the library cannot
// detect and does not claim to.
//
// Arithmetic: four households call Trade once per tick, so the counter is 4 per tick
// and 12 after 3 ticks; the strategy skips calls whose counter mod 5 is below 2.
// A restored counter of 12 skips what the original does; one that starts again at 0
// skips different calls.
//
// Requirement: FR-CHK-036
func TestRestore_StatefulHiddenState(t *testing.T) {
	t.Parallel()
	run := func(saved bool) (diverged bool) {
		var n int
		orig := economysim.WithStrategy("count", hiddenCounter{n: &n, saved: saved})
		cfg := millCfg(3, 4)
		cfg.Roles[0].Strategy = "count"
		w, err := economysim.NewWorld(cfg, economysim.WithSeed(ckptSeed), orig)
		if err != nil {
			t.Fatalf("NewWorld() error = %v, want nil", err)
		}
		for range 3 {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() error = %v, want nil", err)
			}
		}
		var m int
		r, err := economysim.RestoreWorld(cfg, mustCheckpoint(t, w), economysim.WithStrategy("count", hiddenCounter{n: &m, saved: saved}))
		if err != nil {
			t.Fatalf("RestoreWorld() error = %v, want nil", err)
		}
		for range 6 {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() error = %v, want nil", err)
			}
			if err := r.Step(); err != nil {
				t.Fatalf("Step() error = %v, want nil", err)
			}
			if !bytes.Equal(mustMarshal(t, mustCheckpoint(t, w)), mustMarshal(t, mustCheckpoint(t, r))) {
				return true
			}
		}
		return false
	}
	if run(true) {
		t.Errorf("a Stateful strategy that saves its counter diverged after a restore, want identical continuations")
	}
	if !run(false) {
		t.Errorf("a strategy that keeps its counter outside SaveState did not diverge, want the test to show the hidden state")
	}
}

// TestOptions_ContextOnRestore verifies the options of a restore (spec-checkpoint.md
// 6.4): WithSeed is ErrOptionContext, WithReseed is accepted with any salt, 0
// included, and WithTargetSpread is accepted and is part of the effective
// configuration, so a restore that does not repeat the original's value is
// ErrCheckpointConfig.
//
// Requirement: FR-CHK-020
func TestOptions_ContextOnRestore(t *testing.T) {
	t.Parallel()
	plain := mustCheckpoint(t, ckptWorld(t, 2))
	spread := mustCheckpoint(t, ckptWorld(t, 2, economysim.WithTargetSpread(100)))
	tests := []struct {
		name string
		ck   economysim.Checkpoint
		opts []economysim.Option
		want error
	}{
		{"WithSeed", plain, []economysim.Option{economysim.WithSeed(1)}, economysim.ErrOptionContext},
		{"WithSeed after WithReseed", plain, []economysim.Option{economysim.WithReseed(1), economysim.WithSeed(1)}, economysim.ErrOptionContext},
		{"WithReseed(0)", plain, []economysim.Option{economysim.WithReseed(0)}, nil},
		{"WithReseed(5)", plain, []economysim.Option{economysim.WithReseed(5)}, nil},
		{"WithTargetSpread the original had", spread, []economysim.Option{economysim.WithTargetSpread(100)}, nil},
		{"WithTargetSpread the original lacked", plain, []economysim.Option{economysim.WithTargetSpread(100)}, economysim.ErrCheckpointConfig},
		{"WithTargetSpread missing", spread, nil, economysim.ErrCheckpointConfig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, err := economysim.RestoreWorld(millCfg(3, 4), tt.ck, tt.opts...)
			if !errors.Is(err, tt.want) && (tt.want != nil || err != nil) {
				t.Fatalf("RestoreWorld(%s) error = %v, want %v", tt.name, err, tt.want)
			}
			if (w == nil) != (tt.want != nil) {
				t.Errorf("RestoreWorld(%s) world = %v, want a world only without an error", tt.name, w)
			}
		})
	}
}

// TestRestore_ReportsEventAndLog verifies step 12: the observer given to a restore
// receives WorldRestored first and alone (no AgentSpawned, no WorldCreated), with
// the tick, the hash, the format, the names and a snapshot equal to the original's
// at that boundary; the logger receives the configuration warnings of construction, once each, and
// the Info line "world restored" with its seven attributes; and a restore with no observer builds no event.
//
// Requirement: FR-CHK-027, FR-CHK-028, FR-CHK-081
func TestRestore_ReportsEventAndLog(t *testing.T) {
	t.Parallel()
	w := ckptWorld(t, 4)
	ck := mustCheckpoint(t, w)
	rec, h := &recorder{}, newCaptureHandler()
	r := mustRestore(t, ck, economysim.WithObserver(rec), economysim.WithLogger(slog.New(h)))
	if len(rec.events) != 1 {
		t.Fatalf("a restore delivered %d events, want exactly WorldRestored", len(rec.events))
	}
	ev, ok := rec.events[0].(economysim.WorldRestored)
	want, _ := w.Snapshot()
	if !ok || ev.Tick != 4 || ev.ConfigHash != w.ConfigHash() || ev.Format != 1 || ev.Reseeded || ev.Salt != 0 ||
		!reflect.DeepEqual(ev.Snapshot, want) || !reflect.DeepEqual(ev.Names, want.Names) {
		t.Errorf("first event = %+v, want WorldRestored at tick 4 with the original's hash, format 1, no reseed and the original's snapshot and names", rec.events[0])
	}
	// The mill has configuration warnings; construction logs each once at Warn, and a
	// restore must log the same ones, once each.
	hc := newCaptureHandler()
	if _, err := economysim.NewWorld(millCfg(3, 4), economysim.WithSeed(ckptSeed), economysim.WithLogger(slog.New(hc))); err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	var info, warns, wantWarns []logLine
	for _, l := range h.records() {
		switch l.level {
		case slog.LevelInfo:
			info = append(info, l)
		case slog.LevelWarn:
			warns = append(warns, l)
		}
	}
	for _, l := range hc.records() {
		if l.level == slog.LevelWarn {
			wantWarns = append(wantWarns, l)
		}
	}
	if len(info) != 1 || info[0].msg != "world restored" {
		t.Fatalf("Info lines = %+v, want the one line %q", info, "world restored")
	}
	if !reflect.DeepEqual(warns, wantWarns) || len(warns) == 0 {
		t.Errorf("Warn lines of the restore = %+v, want the %d of construction, %+v", warns, len(wantWarns), wantWarns)
	}
	for _, key := range []string{"tick", "seed", "config_hash", "agents", "households", "format", "bytes"} {
		if _, ok := info[0].attr(key); !ok {
			t.Errorf("line %q has no attribute %q", info[0].msg, key)
		}
	}
	if v, _ := info[0].attr("bytes"); v.Int64() != int64(ck.Info().Bytes) {
		t.Errorf("bytes = %d, want %d", v.Int64(), ck.Info().Bytes)
	}
	if err := r.Step(); err != nil {
		t.Errorf("Step() of the restored world error = %v, want nil", err)
	}
}

// ExampleRestoreWorld takes the checkpoint of a world after three ticks and
// restores a second world from it: the restored world is at the same tick and its
// own checkpoint is the same bytes.
func ExampleRestoreWorld() {
	cfg := millCfg(3, 4)
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
	if err != nil {
		panic(err)
	}
	for range 3 {
		if err := w.Step(); err != nil {
			panic(err)
		}
	}
	c, err := w.Checkpoint()
	if err != nil {
		panic(err)
	}
	r, err := economysim.RestoreWorld(cfg, c)
	if err != nil {
		panic(err)
	}
	again, err := r.Checkpoint()
	if err != nil {
		panic(err)
	}
	a, _ := c.MarshalBinary()
	b, _ := again.MarshalBinary()
	fmt.Println(r.Tick(), bytes.Equal(a, b))
	// Output: 3 true
}
