package economysim_test

import (
	"encoding/binary"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// technologyBody returns the body of tag 33 of a checkpoint and whether it has one.
func technologyBody(t *testing.T, raw []byte) (body []byte, ok bool) {
	t.Helper()
	for _, s := range decodeFrame(t, raw).sections {
		if s.tag == 33 {
			return s.body, true
		}
	}
	return nil, false
}

// varintsOf decodes a body that is all unsigned varints.
func varintsOf(t *testing.T, b []byte) []uint64 {
	t.Helper()
	var out []uint64
	for len(b) > 0 {
		v, n := binary.Uvarint(b)
		if n <= 0 {
			t.Fatalf("body %x is not a run of varints", b)
		}
		out, b = append(out, v), b[n:]
	}
	return out
}

// varintBody encodes values as unsigned varints.
func varintBody(vs []uint64) []byte {
	var b []byte
	for _, v := range vs {
		b = binary.AppendUvarint(b, v)
	}
	return b
}

// TestCheckpoint_TechnologyRoundTrip verifies tag 33: a world with the module on, run
// for 60 ticks of pack three, writes three varints for each of the three cells (one
// field) and then one trend carry, 10 values, and no key; that a world restored from
// it writes the same bytes and shows the same snapshot rows (so the gains are
// re-derived from the cells); and that a world with the module off writes no section.
//
// Requirement: FR-TEC-027, FR-TEC-028, FR-TEC-026
func TestCheckpoint_TechnologyRoundTrip(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := fastLearning(t)
	w := newPackWorld(t, cfg, 1)
	step(t, w, 60)
	ck := mustCheckpoint(t, w)
	raw := mustMarshal(t, ck)
	body, ok := technologyBody(t, raw)
	if !ok {
		t.Fatalf("tag 33 of the world with the module on is missing, want a section")
	}
	if got := len(varintsOf(t, body)); got != 3*3+1 {
		t.Errorf("tag 33 holds %d values, want 10 (3 cells of 3, 1 trend carry)", got)
	}
	if v := varintsOf(t, body); v[0] == 0 && v[1] == 0 {
		t.Errorf("tag 33 first cell = %v, want experience above 0 after 60 ticks of learning", v[:3])
	}
	r, err := economysim.RestoreWorld(cfg, ck)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	if got := mustMarshal(t, mustCheckpoint(t, r)); !slices.Equal(got, raw) {
		t.Errorf("checkpoint of the restored world differs from the one it was restored from")
	}
	checkTechRows(t, w, r)
	off := cfg
	off.Modules.Technology = nil
	ow := newPackWorld(t, off, 1)
	step(t, ow, 5)
	if body, ok := technologyBody(t, mustMarshal(t, mustCheckpoint(t, ow))); ok {
		t.Errorf("tag 33 of the world with the module off = %v, want no section", body)
	}
}

// checkTechRows fails if the technology rows of the snapshot of r differ from w's or
// are empty.
func checkTechRows(t *testing.T, w, r *economysim.World) {
	t.Helper()
	want, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	got, err := r.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() of the restored world error = %v, want nil", err)
	}
	if len(want.Tech) == 0 || !reflect.DeepEqual(want.Tech, got.Tech) {
		t.Errorf("restored Snapshot().Tech = %+v, want %+v (non-empty)", got.Tech, want.Tech)
	}
}

// techCorruptCases are the ways a tag 33 body of three cells and one field can be
// wrong. cap is the field's cap, tpy the ticks of a year of the pack.
func techCorruptCases(cap, tpy uint64, good []uint64) []struct {
	name string
	body []byte
} {
	edit := func(i int, v uint64) []byte { g := slices.Clone(good); g[i] = v; return varintBody(g) }
	last := len(good) - 1
	return []struct {
		name string
		body []byte
	}{
		{"experience above MaxExperience", edit(0, economysim.MaxExperience+1)},
		{"advance above the cap", edit(1, cap+1)},
		{"catch-up carry at 10 000 ticks of a year", edit(2, 10_000*tpy)},
		{"trend carry at the ticks of a year", edit(last, tpy)},
		{"a section that ends early", varintBody(good[:last])},
		{"an empty section", nil},
		{"a byte after the last field", append(varintBody(good), 0)},
		{"a varint that is not minimal", append(varintBody(good[:last]), 0x80, 0x00)},
	}
}

// TestCheckpoint_TechnologyRefusesCorrupt verifies that the decoder of tag 33 refuses
// each way the section can be wrong, as ErrCheckpointCorrupt naming section 33:
// Experience above MaxExperience, Advance above the cap, CatchUp at 10 000 ticks of a
// year, TrendCarry at the ticks of a year, a section that ends early or is empty, a byte
// after the last field and a varint that is not minimal; a missing section with the
// module on and a section present with it off; and that the bounds themselves restore.
//
// Arithmetic: the overlay's cap is 3 000 and a year is 365 ticks, so the largest
// Advance is 3 000, the largest CatchUp 10 000 * 365 - 1 = 3 649 999 and the largest
// TrendCarry 364.
//
// Requirement: FR-TEC-028, FR-CHK-021
func TestCheckpoint_TechnologyRefusesCorrupt(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := fastLearning(t)
	capBP, tpy := uint64(cfg.Modules.Technology.Fields[0].CapBP), uint64(cfg.Calendar.TicksPerYear())
	w := newPackWorld(t, cfg, 1)
	step(t, w, 5)
	raw := mustMarshal(t, mustCheckpoint(t, w))
	body, _ := technologyBody(t, raw)
	good := varintsOf(t, body)
	for _, tt := range techCorruptCases(capBP, tpy, good) {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := reframe(t, raw, func(s []section) []section {
				return editSection(s, 33, func([]byte) []byte { return tt.body })
			})
			_, err := economysim.RestoreWorld(cfg, c)
			if !errors.Is(err, economysim.ErrCheckpointCorrupt) || !strings.Contains(err.Error(), "section 33") {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt naming section 33", tt.name, err)
			}
		})
	}
	t.Run("a section missing", func(t *testing.T) {
		t.Parallel()
		c := reframe(t, raw, func(s []section) []section {
			return slices.DeleteFunc(s, func(x section) bool { return x.tag == 33 })
		})
		if _, err := economysim.RestoreWorld(cfg, c); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
			t.Errorf("RestoreWorld(no section 33) error = %v, want ErrCheckpointCorrupt", err)
		}
	})
	t.Run("a section with the module off", func(t *testing.T) {
		t.Parallel()
		off := cfg
		off.Modules.Technology = nil
		c := reframe(t, raw, func(s []section) []section {
			i := slices.IndexFunc(s, func(x section) bool { return x.tag > 33 })
			return slices.Insert(s, i, section{tag: 33, body: body})
		})
		if _, err := economysim.RestoreWorld(off, c); !errors.Is(err, economysim.ErrCheckpointCorrupt) &&
			!errors.Is(err, economysim.ErrCheckpointConfig) {
			t.Errorf("RestoreWorld(a section 33 with the module off) error = %v, want a refusal", err)
		}
	})
	t.Run("the bounds restore", func(t *testing.T) {
		t.Parallel()
		edge := slices.Clone(good)
		edge[0], edge[1], edge[2], edge[len(edge)-1] = economysim.MaxExperience, capBP, 10_000*tpy-1, tpy-1
		c := reframe(t, raw, func(s []section) []section {
			return editSection(s, 33, func([]byte) []byte { return varintBody(edge) })
		})
		if _, err := economysim.RestoreWorld(cfg, c); err != nil {
			t.Errorf("RestoreWorld(the bounds of tag 33) error = %v, want nil", err)
		}
	})
}

// TestCheckpoint_TechnologyGainRederived verifies that the gain is not saved and is
// derived on restore: a world given an advance by a call, checkpointed, restored and
// stepped once produces the same TechnologyAdvanced BeforeBP as the original, which is
// the gain the restore derived from Experience and Advance.
//
// Requirement: FR-TEC-028
func TestCheckpoint_TechnologyGainRederived(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := fastLearning(t)
	var a, b recorder
	w := newPackWorld(t, cfg, 1, economysim.WithObserver(&a))
	step(t, w, 20)
	adjustRegion0(t, w, 700)
	r, err := economysim.RestoreWorld(cfg, mustCheckpoint(t, w), economysim.WithObserver(&b))
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	a.events, b.events = nil, nil
	step(t, w, 3)
	step(t, r, 3)
	var ga, gb []economysim.TechnologyAdvanced
	for _, e := range a.events {
		if ev, ok := e.(economysim.TechnologyAdvanced); ok {
			ga = append(ga, ev)
		}
	}
	for _, e := range b.events {
		if ev, ok := e.(economysim.TechnologyAdvanced); ok {
			gb = append(gb, ev)
		}
	}
	if len(ga) == 0 || !reflect.DeepEqual(ga, gb) {
		t.Errorf("TechnologyAdvanced after the restore = %+v, want %+v (non-empty)", gb, ga)
	}
}

// FuzzDecodeTechnology verifies that a restore never panics on any body of tag 33 and
// either restores or refuses with ErrCheckpointCorrupt, and that a body it accepts is
// written back byte for byte (the encoding is minimal and the count is implied).
//
// Requirement: FR-TEC-028, FR-CHK-021
func FuzzDecodeTechnology(f *testing.F) {
	cfg := loadValidPack(f, technologyThree...)
	w := newPackWorld(f, cfg, 1)
	for range 5 {
		if err := w.Step(); err != nil {
			f.Fatalf("Step() error = %v, want nil", err)
		}
	}
	ck, err := w.Checkpoint()
	if err != nil {
		f.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	raw, err := ck.MarshalBinary()
	if err != nil {
		f.Fatalf("MarshalBinary() error = %v, want nil", err)
	}
	var good []byte
	for _, s := range decodeFrame(f, raw).sections {
		if s.tag == 33 {
			good = s.body
		}
	}
	f.Add(good)
	f.Add([]byte{})
	f.Add(append(slices.Clone(good), 1))
	f.Fuzz(func(t *testing.T, body []byte) {
		c := reframe(t, raw, func(s []section) []section {
			return editSection(s, 33, func([]byte) []byte { return body })
		})
		r, err := economysim.RestoreWorld(cfg, c)
		if err != nil {
			if !errors.Is(err, economysim.ErrCheckpointCorrupt) {
				t.Fatalf("RestoreWorld(%x) error = %v, want nil or ErrCheckpointCorrupt", body, err)
			}
			return
		}
		got, _ := technologyBody(t, mustMarshal(t, mustCheckpoint(t, r)))
		if !slices.Equal(got, body) {
			t.Fatalf("RestoreWorld(%x) then Checkpoint wrote %x, want the same body", body, got)
		}
	})
}

// discoveryTicks steps w for ticks ticks and returns the ticks of the TechnologyAdvanced
// events with a discovery, from rec.
func discoveryTicks(t *testing.T, w *economysim.World, rec *recorder, ticks int) []int64 {
	t.Helper()
	rec.forget()
	step(t, w, ticks)
	var out []int64
	for _, e := range rec.events {
		if a, ok := e.(economysim.TechnologyAdvanced); ok && a.Discoveries > 0 {
			out = append(out, a.Tick*10+int64(a.Region))
		}
	}
	return out
}

// TestFork_Technology verifies that a world forked inside the module runs on exactly as
// an uninterrupted one in every event, in the snapshot rows, in the checkpoint bytes and
// in the log records: before the first step, one tick before and one tick after a
// discovery, one tick before and after a catch-up add, after a discovery much later and
// after an AdjustTechnology call that raised the advance of a cell.
//
// Setup: pack three with the four-mechanism overlay and a base of 10, seed 1 (probed:
// a discovery in region 2 at tick 6 and in region 1 at tick 140, a catch-up add in
// region 1 at ticks 10, 15, 19, ...), M = 30 ticks after the boundary.
//
// Requirement: FR-TEC-027, FR-TEC-028, FR-CHK-050
func TestFork_Technology(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := fastLearning(t)
	opts := func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} }
	advanced := func(t *testing.T, restored, since []economysim.Event) {
		n := 0
		for _, e := range since {
			if _, ok := e.(economysim.TechnologyAdvanced); ok {
				n++
			}
		}
		if n == 0 {
			t.Errorf("no TechnologyAdvanced after the boundary, want some so the fork carries the cells")
		}
	}
	call := func(w *economysim.World) {
		if err := w.AdjustTechnology([]economysim.TechnologyAdjustment{{Region: 0, Field: 0, DeltaBP: 500}}); err != nil {
			panic("AdjustTechnology: " + err.Error()) // the world of the test accepts it; Before has no *testing.T
		}
	}
	for _, c := range []struct {
		name   string
		n      int
		before func(*economysim.World)
	}{
		{"before the first step", 0, nil},
		{"a tick before a discovery", 6, nil},
		{"a tick after a discovery", 7, nil},
		{"a tick before a catch-up add", 9, nil},
		{"a tick after a catch-up add", 11, nil},
		{"after a late discovery", 141, nil},
		{"after a call", 20, call},
	} {
		forkAtN(t, forkCase{Name: "technology " + c.name, Cfg: cfg, Seed: 1, N: c.n, M: forkM, Logs: true,
			Opts: opts, Before: c.before, Inspect: advanced})
	}
}

// TestFork_TechnologyReseed verifies FR-TEC-012: a world restored with WithReseed
// discovers on other ticks than the original, because the key follows from the
// effective seed, and two restores of one checkpoint with one salt agree on events,
// snapshot, audit and checkpoint bytes for 30 ticks.
//
// Requirement: FR-TEC-011, FR-TEC-012, FR-CHK-050
func TestFork_TechnologyReseed(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := fastLearning(t)
	salt := uint64(7)
	opts := func() []economysim.Option { return []economysim.Option{economysim.WithAudit()} }
	forkAtN(t, forkCase{Name: "technology reseeded", Cfg: cfg, Seed: 1, N: 5, M: forkM, Reseed: &salt, Opts: opts})

	var ra, rb recorder
	w := newPackWorld(t, cfg, 1, economysim.WithObserver(&ra))
	step(t, w, 2)
	r, err := economysim.RestoreWorld(cfg, mustCheckpoint(t, w), economysim.WithReseed(salt), economysim.WithObserver(&rb))
	if err != nil {
		t.Fatalf("RestoreWorld(WithReseed) error = %v, want nil", err)
	}
	same := discoveryTicks(t, w, &ra, 200)
	other := discoveryTicks(t, r, &rb, 200)
	if len(same) == 0 || slices.Equal(same, other) {
		t.Errorf("discoveries of the original %v and of the reseeded restore %v, want some and different", same, other)
	}
}
