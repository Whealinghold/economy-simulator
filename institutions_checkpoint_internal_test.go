package economysim

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

// The tests of the membership field in the agent record, the tail of tag 3
// (spec-institutions.md 12.1, 12.3). The field is state of the module, so a test sets it
// through the module's slices, as the round would, and reads it back after a restore.

// orgRestore checkpoints w and restores it with cfg.
func orgRestore(t *testing.T, w *World, cfg Config) (*World, error) {
	t.Helper()
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	return RestoreWorld(cfg, ck)
}

// TestCheckpoint_MembershipFieldPerAgent verifies that the membership of each agent is
// saved in its record and read back: the restored world holds the same body and region for
// every live agent, and runs on as the original does.
//
// Requirement: FR-ORG-049, FR-ORG-033
func TestCheckpoint_MembershipFieldPerAgent(t *testing.T) {
	t.Parallel()
	cfg := orgToy()
	w := newMill(t, cfg)
	step(t, w, 3)
	r, err := orgRestore(t, w, cfg)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	w.eachLive(func(a *agent) {
		i := int(a.id)
		if r.org.member[i] != w.org.member[i] || r.org.memberRegion[i] != w.org.memberRegion[i] {
			t.Errorf("agent %d: restored membership = (%d, %d), want (%d, %d)", i, r.org.member[i], r.org.memberRegion[i], w.org.member[i], w.org.memberRegion[i])
		}
	})
	if got := len(orgMembers(r, 0)) + len(orgMembers(r, 1)); got != 15 {
		t.Errorf("members after the restore = %d, want 15 (4 + 5 smiths of guild and 6 farmers of lodge)", got)
	}
	step(t, w, 5)
	step(t, r, 5)
	a, errA := w.Snapshot()
	b, errB := r.Snapshot()
	if errA != nil || errB != nil || !reflect.DeepEqual(a, b) {
		t.Errorf("the restored world differs from the original after 5 ticks (errors %v, %v)", errA, errB)
	}
}

// TestCheckpoint_MembershipRefusesCorrupt verifies that the decoder of the tail refuses a
// membership that is half zero, names a body that is not defined, or names a region outside
// the body's scope, as ErrCheckpointCorrupt, and accepts the membership of a household that
// is no longer eligible, which the next round lapses and counts.
//
// Arithmetic: guild (BodyID 0, stored as 1) has the scope a and b; c is region 2, stored as 3.
// A farmer of b, stored as a member of guild in b (1, 2), is on no list of guild and is legal.
//
// Requirement: FR-ORG-050
func TestCheckpoint_MembershipRefusesCorrupt(t *testing.T) {
	t.Parallel()
	cfg := orgToy()
	smith := func(w *World) int { return orgAgents(w, orgB, orgSmith)[0] }
	farmer := func(w *World) int { return orgAgents(w, orgB, orgFarmer)[0] }
	for _, tc := range []struct {
		name    string
		set     func(w *World)
		corrupt bool
	}{
		{"a body and no region", func(w *World) { w.org.member[smith(w)], w.org.memberRegion[smith(w)] = 1, 0 }, true},
		{"a region and no body", func(w *World) { w.org.member[smith(w)], w.org.memberRegion[smith(w)] = 0, 2 }, true},
		{"a body that is not defined", func(w *World) { w.org.member[smith(w)], w.org.memberRegion[smith(w)] = 3, 2 }, true},
		{"a region outside the scope", func(w *World) { w.org.member[smith(w)], w.org.memberRegion[smith(w)] = 1, 3 }, true},
		{"a region that is not defined", func(w *World) { w.org.member[smith(w)], w.org.memberRegion[smith(w)] = 1, 9 }, true},
		{"a household that is no longer eligible", func(w *World) { w.org.member[farmer(w)], w.org.memberRegion[farmer(w)] = 1, 2 }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := newMill(t, cfg)
			step(t, w, 2)
			tc.set(w)
			r, err := orgRestore(t, w, cfg)
			if tc.corrupt != errors.Is(err, ErrCheckpointCorrupt) || (!tc.corrupt && err != nil) {
				t.Fatalf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt = %v", tc.name, err, tc.corrupt)
			}
			if tc.corrupt {
				return
			}
			r.org.round(r)
			cell := r.org.rd.out[int(orgB)*r.org.nb+0]
			if cell.LapsedList != 1 || r.org.member[farmer(r)] != 0 {
				t.Errorf("the round after the restore: (b, guild) counts = %+v, member field = %d, want one LapsedList and the field cleared", cell, r.org.member[farmer(r)])
			}
		})
	}
}

// TestRegionBound_NoneRegistered verifies that the module registers no region_bound
// obligation: a membership is a status, so the table of the world with the module on lists
// the same obligations as the world without it (spec-institutions.md 3.6, FR-ORG-009).
//
// Requirement: FR-ORG-009
func TestRegionBound_NoneRegistered(t *testing.T) {
	t.Parallel()
	off := orgToy()
	off.Modules.Institutions = nil
	names := func(w *World) []string {
		var out []string
		for _, r := range w.bounds.rows {
			out = append(out, r.err.Obligation)
		}
		return out
	}
	if on, none := names(newMill(t, orgToy())), names(newMill(t, off)); !slices.Equal(on, none) {
		t.Errorf("region-bound obligations with the module on = %v, off = %v, want the same", on, none)
	}
}
