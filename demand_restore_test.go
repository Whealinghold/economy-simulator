package economysim_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// demandCheckpoint steps the demand world 5 ticks and returns its checkpoint
// bytes, the offset in the body of tag 12 where the sparse runs begin, and the
// number of rungs of the cook's ladder.
func demandCheckpoint(t *testing.T) (raw []byte, runsAt int, cook economysim.RoleID) {
	t.Helper()
	cfg := demandConfig()
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	for range 5 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step error = %v, want nil", err)
		}
	}
	raw = mustMarshal(t, mustCheckpoint(t, w))
	body := tagBody(t, raw, 12)
	off := 0
	for id := range economysim.AgentCount(w) {
		role, weight := economysim.AgentRole(w, economysim.AgentID(id))
		if weight < 1 || len(economysim.LadderOf(w, role)) == 0 {
			continue
		}
		var n uint64
		n, off = uv(t, body, off)
		for range n {
			_, off = uv(t, body, off)
		}
	}
	for i, r := range cfg.Roles {
		if r.ID == "cook" {
			cook = economysim.RoleID(i)
		}
	}
	return raw, off, cook
}

// bodyOf returns the body of the section with the tag in a checkpoint.
func tagBody(t *testing.T, raw []byte, tag uint64) []byte {
	t.Helper()
	for _, s := range decodeFrame(t, raw).sections {
		if s.tag == tag {
			return s.body
		}
	}
	t.Fatalf("no section %d in the checkpoint", tag)
	return nil
}

// TestCheckpointTable_Tag12 verifies that tag 12 is in a checkpoint of a world with
// the demand module on, between the regions and the extensions, and in none of a
// world with it off.
//
// Requirement: FR-DEM-034
func TestCheckpointTable_Tag12(t *testing.T) {
	t.Parallel()
	tags := func(cfg economysim.Config) []uint64 {
		w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
		if err != nil {
			t.Fatalf("NewWorld error = %v, want nil", err)
		}
		var out []uint64
		for _, s := range decodeFrame(t, mustMarshal(t, mustCheckpoint(t, w))).sections {
			out = append(out, s.tag)
		}
		return out
	}
	on, off := tags(demandConfig()), tags(referenceConfig())
	if want := []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 34}; !slices.Equal(on, want) {
		t.Errorf("tags with the module on = %v, want %v", on, want)
	}
	if slices.Contains(off, 12) {
		t.Errorf("tags with the module off = %v, want no 12", off)
	}
}

// TestRestore_DemandCorruptCarry verifies that tag 12 is refused when a carry is
// 10 000, when an agent's carry count is not its ladder's length, and when a
// sparse run row is at its default, out of order, above the tick, names a want
// outside the role's ladder or names a role out of range.
//
// Requirement: FR-DEM-034, FR-CHK-013
func TestRestore_DemandCorruptCarry(t *testing.T) {
	t.Parallel()
	raw, runsAt, cook := demandCheckpoint(t)
	rows := func(rows ...[4]uint64) func(*testing.T, []byte) []byte {
		return func(t *testing.T, b []byte) []byte {
			b = slices.Clone(b[:runsAt])
			b = append(b, byte(len(rows)))
			for _, r := range rows {
				for _, v := range r {
					b = append(b, byte(v)) // every value here is below 128
				}
			}
			return b
		}
	}
	c := uint64(cook)
	tests := []struct {
		name string
		edit func(t *testing.T, b []byte) []byte
	}{
		// Carries are signed (zigzag): 10 000 is 20 000 and -1 is 1.
		{"carry of 10 000", func(t *testing.T, b []byte) []byte { return setVarint(t, b, 1, 20_000) }},
		{"negative carry", func(t *testing.T, b []byte) []byte { return setVarint(t, b, 1, 1) }},
		{"carry count above the ladder", func(t *testing.T, b []byte) []byte { return setVarint(t, b, 0, 3) }},
		{"carry count below the ladder", func(t *testing.T, b []byte) []byte { return setVarint(t, b, 0, 1) }},
		{"run at its default", rows([4]uint64{0, c, 0, 0})},
		{"run above the tick", rows([4]uint64{0, c, 0, 6})},
		{"keys descending", rows([4]uint64{0, c, 1, 2}, [4]uint64{0, c, 0, 2})},
		{"key repeated", rows([4]uint64{0, c, 0, 2}, [4]uint64{0, c, 0, 3})},
		{"role out of range", rows([4]uint64{0, 90, 0, 2})},
		{"want out of range", rows([4]uint64{0, c, 90, 2})},
		{"want not in the ladder", rows([4]uint64{0, 1, 1, 2})}, // grower (role 1 at the reference chain) has comfort only
	}
	if _, err := economysim.RestoreWorld(demandConfig(), reframe(t, raw, func(s []section) []section { return s })); err != nil {
		t.Fatalf("the unedited checkpoint: RestoreWorld error = %v, want nil", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ck := reframe(t, raw, func(s []section) []section {
				return editSection(s, 12, func(b []byte) []byte { return tt.edit(t, b) })
			})
			if _, err := economysim.RestoreWorld(demandConfig(), ck); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt", tt.name, err)
			}
		})
	}
}
