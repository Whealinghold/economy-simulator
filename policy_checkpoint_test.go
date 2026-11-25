package economysim_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim"
)

// The tests of tag 32 through the public API and a real frame (spec-policy.md 13;
// spec-checkpoint.md 4.2.1): the section is present with the module and absent without it, and
// the restore refuses a checkpoint that disagrees with the configuration.

// policyCheckpointBytes is the checkpoint of a world of c that has run ticks steps.
func policyCheckpointBytes(t *testing.T, c economysim.Config) []byte {
	t.Helper()
	w, _ := policyRun(t, c, 3)
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	raw, err := ck.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v, want nil", err)
	}
	return raw
}

// TestCheckpoint_PolicySectionPresence verifies that a world with the module writes the section
// of tag 32 (empty with no directive), that a checkpoint without it is refused by the world that
// has the module, and that a checkpoint with it is refused by a world that has not.
//
// Requirement: FR-PLC-090
func TestCheckpoint_PolicySectionPresence(t *testing.T) {
	t.Parallel()
	on := policyGoverned()
	raw := policyCheckpointBytes(t, on)
	if at := slices.IndexFunc(decodeFrame(t, raw).sections, func(s section) bool { return s.tag == 32 }); at < 0 {
		t.Fatalf("the checkpoint of a world with the module has no section of tag 32")
	}
	missing := reframe(t, raw, func(secs []section) []section {
		return slices.DeleteFunc(secs, func(s section) bool { return s.tag == 32 })
	})
	if _, err := economysim.RestoreWorld(on, missing); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
		t.Errorf("RestoreWorld() without the section: error = %v, want ErrCheckpointCorrupt", err)
	}

	off := policyGoverned()
	off.Modules.Policy = nil
	offRaw := policyCheckpointBytes(t, off)
	if at := slices.IndexFunc(decodeFrame(t, offRaw).sections, func(s section) bool { return s.tag == 32 }); at >= 0 {
		t.Fatalf("the checkpoint of a world without the module has a section of tag 32")
	}
	extra := reframe(t, offRaw, func(secs []section) []section {
		at := slices.IndexFunc(secs, func(s section) bool { return s.tag > 32 })
		return slices.Insert(secs, at, section{tag: 32})
	})
	if _, err := economysim.RestoreWorld(off, extra); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
		t.Errorf("RestoreWorld() with a section the module does not write: error = %v, want ErrCheckpointCorrupt", err)
	}
}
