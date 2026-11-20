package economysim_test

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The tests of the two slots of the institutions module through the public API (plan 3-org
// Phase 8, spec-institutions.md 7.1, 7.4, 12.2): the errors in their order, the refusal of
// a call inside a step, the slot state in a checkpoint and in a fork. The tables a call
// rebuilds are tested in institutions_slots_internal_test.go. The world is institutionsConfig,
// whose controls are, by ControlID, assize 0 (a price control) and usury 1 (a rate limit),
// and whose rights are make_tools 0 and sell_tools 1.

// pathOf returns the path of the *ValidationError in err, nil when err is none.
func pathOf(err error) []string {
	var ve *economysim.ValidationError
	if errors.As(err, &ve) {
		return ve.Path
	}
	return nil
}

// TestSetLegalLevels_ErrorOrder verifies the errors of the call in the order of
// spec-institutions.md 7.1.1: the first entry that fails decides, an unknown control before
// the bounds of its levels, and a repeated key is ErrDuplicateID. Each error names the entry
// and the field, and a refused call changes nothing.
//
// Arithmetic: the cap of a price control is at most MaxBasisPoints and its floor at most the
// cap when a cap is set; a rate limit has a cap of at most MaxCreditRateBP and no floor.
//
// Requirement: FR-ORG-036, FR-INT-032
func TestSetLegalLevels_ErrorOrder(t *testing.T) {
	t.Parallel()
	ok := economysim.LegalLevel{Control: 0, CapBP: 10_000}
	for _, tc := range []struct {
		name    string
		changes []economysim.LegalLevel
		want    error
		path    []string
	}{
		{"unknown control above the list", []economysim.LegalLevel{{Control: 2}}, economysim.ErrUnknownControl, []string{"changes[0]", "control"}},
		{"unknown control below zero", []economysim.LegalLevel{ok, {Control: -1}}, economysim.ErrUnknownControl, []string{"changes[1]", "control"}},
		{"cap above the bound", []economysim.LegalLevel{{Control: 0, CapBP: economysim.MaxBasisPoints + 1}}, economysim.ErrOutOfBounds, []string{"changes[0]", "cap_bp"}},
		{"floor above the cap", []economysim.LegalLevel{{Control: 0, CapBP: 8_000, FloorBP: 9_000}}, economysim.ErrOutOfBounds, []string{"changes[0]", "cap_bp"}},
		{"negative cap", []economysim.LegalLevel{{Control: 0, CapBP: -1}}, economysim.ErrOutOfBounds, []string{"changes[0]", "cap_bp"}},
		{"negative floor", []economysim.LegalLevel{{Control: 0, CapBP: 0, FloorBP: -1}}, economysim.ErrOutOfBounds, []string{"changes[0]", "cap_bp"}},
		{"smallest cap", []economysim.LegalLevel{{Control: 0, CapBP: math.MinInt64}}, economysim.ErrOutOfBounds, []string{"changes[0]", "cap_bp"}},
		{"largest cap", []economysim.LegalLevel{{Control: 0, CapBP: math.MaxInt64}}, economysim.ErrOutOfBounds, []string{"changes[0]", "cap_bp"}},
		{"largest floor with no cap", []economysim.LegalLevel{{Control: 0, FloorBP: math.MaxInt64}}, economysim.ErrOutOfBounds, []string{"changes[0]", "cap_bp"}},
		{"smallest rate cap", []economysim.LegalLevel{{Control: 1, CapBP: math.MinInt64}}, economysim.ErrOutOfBounds, []string{"changes[0]", "cap_bp"}},
		{"largest rate cap", []economysim.LegalLevel{{Control: 1, CapBP: math.MaxInt64}}, economysim.ErrOutOfBounds, []string{"changes[0]", "cap_bp"}},
		{"negative floor on a rate limit", []economysim.LegalLevel{{Control: 1, CapBP: 600, FloorBP: math.MinInt64}}, economysim.ErrOutOfBounds, []string{"changes[0]", "floor_bp"}},
		{"smallest control", []economysim.LegalLevel{{Control: math.MinInt32}}, economysim.ErrUnknownControl, []string{"changes[0]", "control"}},
		{"floor on a rate limit", []economysim.LegalLevel{{Control: 1, CapBP: 600, FloorBP: 1}}, economysim.ErrOutOfBounds, []string{"changes[0]", "floor_bp"}},
		{"rate cap above the bound", []economysim.LegalLevel{{Control: 1, CapBP: economysim.MaxCreditRateBP + 1}}, economysim.ErrOutOfBounds, []string{"changes[0]", "cap_bp"}},
		{"the first failure decides", []economysim.LegalLevel{ok, {Control: 9}, {Control: 1, CapBP: -5}}, economysim.ErrUnknownControl, []string{"changes[1]", "control"}},
		{"a repeated key", []economysim.LegalLevel{{Control: 0, CapBP: 9_000}, {Control: 0, CapBP: 8_000}}, economysim.ErrDuplicateID, []string{"changes[1]", "control"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := newPackWorld(t, institutionsConfig(), 1)
			err := w.SetLegalLevels(tc.changes)
			if !errors.Is(err, tc.want) {
				t.Fatalf("SetLegalLevels(%+v) error = %v, want %v", tc.changes, err, tc.want)
			}
			if got := pathOf(err); !slices.Equal(got, tc.path) {
				t.Errorf("SetLegalLevels(%+v) error path = %v, want %v", tc.changes, got, tc.path)
			}
			if s, _ := w.Slots(); s.LegalLevels != nil {
				t.Errorf("Slots().LegalLevels after a refused call = %+v, want none", s.LegalLevels)
			}
		})
	}
}

// TestSetRightSuspensions_ErrorOrder verifies the errors of the call: an unknown right is
// ErrUnknownRight, a repeated key ErrDuplicateID, the first failing entry decides, and a
// refused call changes nothing.
//
// Requirement: FR-ORG-037, FR-INT-032
func TestSetRightSuspensions_ErrorOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		changes []economysim.RightSuspension
		want    error
		path    []string
	}{
		{"unknown right above the list", []economysim.RightSuspension{{Right: 2, Suspended: true}}, economysim.ErrUnknownRight, []string{"changes[0]", "right"}},
		{"unknown right below zero", []economysim.RightSuspension{{Right: 0, Suspended: true}, {Right: -1}}, economysim.ErrUnknownRight, []string{"changes[1]", "right"}},
		{"smallest right", []economysim.RightSuspension{{Right: math.MinInt32}}, economysim.ErrUnknownRight, []string{"changes[0]", "right"}},
		{"largest right", []economysim.RightSuspension{{Right: math.MaxInt32}}, economysim.ErrUnknownRight, []string{"changes[0]", "right"}},
		{"a repeated key", []economysim.RightSuspension{{Right: 1, Suspended: true}, {Right: 1}}, economysim.ErrDuplicateID, []string{"changes[1]", "right"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := newPackWorld(t, institutionsConfig(), 1)
			err := w.SetRightSuspensions(tc.changes)
			if !errors.Is(err, tc.want) {
				t.Fatalf("SetRightSuspensions(%+v) error = %v, want %v", tc.changes, err, tc.want)
			}
			if got := pathOf(err); !slices.Equal(got, tc.path) {
				t.Errorf("SetRightSuspensions(%+v) error path = %v, want %v", tc.changes, got, tc.path)
			}
			if s, _ := w.Slots(); s.RightSuspensions != nil {
				t.Errorf("Slots().RightSuspensions after a refused call = %+v, want none", s.RightSuspensions)
			}
		})
	}
}

// TestSlots_ModuleOffAndInStep verifies the two refusals that come before every entry: a
// world without the module returns ErrModuleOff whatever the entries are, and a call from an
// observer inside a step returns ErrInStep (spec-integration.md 3.2).
//
// Requirement: FR-ORG-036, FR-ORG-037, FR-INT-032
func TestSlots_ModuleOffAndInStep(t *testing.T) {
	t.Parallel()
	t.Run("module off", func(t *testing.T) {
		t.Parallel()
		w := newPackWorld(t, loadValidPack(t, packOne...), 1)
		if err := w.SetLegalLevels([]economysim.LegalLevel{{Control: 99, CapBP: -1}}); !errors.Is(err, economysim.ErrModuleOff) {
			t.Errorf("SetLegalLevels() with the module off error = %v, want ErrModuleOff", err)
		}
		if err := w.SetRightSuspensions([]economysim.RightSuspension{{Right: 99}}); !errors.Is(err, economysim.ErrModuleOff) {
			t.Errorf("SetRightSuspensions() with the module off error = %v, want ErrModuleOff", err)
		}
		if s, err := w.Slots(); err != nil || s.LegalLevels != nil || s.RightSuspensions != nil {
			t.Errorf("Slots() with the module off = %+v, %v, want empty slots and nil", s, err)
		}
	})
	t.Run("in step", func(t *testing.T) {
		t.Parallel()
		var w *economysim.World
		var levels, suspensions error
		probe := economysim.ObserverFunc(func(e economysim.Event) {
			if _, ok := e.(economysim.TickStarted); ok {
				levels = w.SetLegalLevels([]economysim.LegalLevel{{Control: 0, CapBP: 5_000}})
				suspensions = w.SetRightSuspensions([]economysim.RightSuspension{{Right: 0, Suspended: true}})
			}
		})
		w = newPackWorld(t, institutionsConfig(), 1, economysim.WithObserver(probe))
		step(t, w, 1)
		if !errors.Is(levels, economysim.ErrInStep) || !errors.Is(suspensions, economysim.ErrInStep) {
			t.Errorf("calls from an observer inside a step = %v and %v, want ErrInStep", levels, suspensions)
		}
	})
}

// TestSlots_LegalLevelsSavedNotHashed verifies that the slot state is state and not
// configuration: a call leaves the configuration hash as it was, a checkpoint of the world
// after it restores the levels and the suspension (listed by Slots and by the snapshot), and
// the restored world writes the same bytes again.
//
// Requirement: FR-ORG-033, FR-ORG-038, FR-CHK-045
func TestSlots_LegalLevelsSavedNotHashed(t *testing.T) {
	t.Parallel()
	cfg := institutionsConfig()
	w := newPackWorld(t, cfg, 1)
	step(t, w, 3)
	hash := economysim.HashConfig(w.Config())
	if err := w.SetLegalLevels([]economysim.LegalLevel{{Control: 0, CapBP: 9_000, FloorBP: 4_000}, {Control: 1, CapBP: 700}}); err != nil {
		t.Fatalf("SetLegalLevels() error = %v, want nil", err)
	}
	if err := w.SetRightSuspensions([]economysim.RightSuspension{{Right: 1, Suspended: true}}); err != nil {
		t.Fatalf("SetRightSuspensions() error = %v, want nil", err)
	}
	if got := economysim.HashConfig(w.Config()); got != hash {
		t.Errorf("HashConfig after the calls = %x, want %x: the slot state is not configuration", got, hash)
	}
	ck := mustCheckpoint(t, w)
	restored, err := economysim.RestoreWorld(cfg, ck)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	a, _ := w.Slots()
	b, _ := restored.Slots()
	wantSlots := economysim.SlotState{
		LegalLevels:      []economysim.LegalLevel{{Control: 0, CapBP: 9_000, FloorBP: 4_000}, {Control: 1, CapBP: 700}},
		RightSuspensions: []economysim.RightSuspension{{Right: 1, Suspended: true}},
	}
	if !reflect.DeepEqual(a, wantSlots) || !reflect.DeepEqual(b, wantSlots) {
		t.Errorf("Slots() of the original and of the restore = %+v and %+v, want both %+v", a, b, wantSlots)
	}
	if !slices.Equal(mustMarshal(t, mustCheckpoint(t, restored)), mustMarshal(t, ck)) {
		t.Errorf("the checkpoint of the restored world differs from the one it was restored from")
	}
}

// TestFork_InstitutionsSlots verifies the fork at N of a world that made a slot call before
// its boundary: the original and two restores give the same events, snapshot, audit,
// checkpoint bytes and logs for 30 ticks, so the levels and the suspension in force are the
// ones the uninterrupted world has, and the next round and the next clamp are its own.
//
// Arithmetic: the call lowers the cap of assize to 9 000 bp (96 * 9 000 / 10 000 = 86 for the
// staple), sets its floor to 4 000 and suspends the sale right sell_tools; N 0 makes the call
// before the first step, N 5 after five.
//
// Requirement: FR-ORG-033, FR-ORG-036, FR-ORG-037, FR-CHK-050
func TestFork_InstitutionsSlots(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	call := func(w *economysim.World) {
		if err := w.SetLegalLevels([]economysim.LegalLevel{{Control: 0, CapBP: 9_000, FloorBP: 4_000}, {Control: 1, CapBP: 700}}); err != nil {
			panic("SetLegalLevels: " + err.Error()) // Before has no *testing.T
		}
		if err := w.SetRightSuspensions([]economysim.RightSuspension{{Right: 1, Suspended: true}}); err != nil {
			panic("SetRightSuspensions: " + err.Error())
		}
	}
	for _, n := range []int{0, 5} {
		forkAtN(t, forkCase{Name: "institutions slots", Cfg: orgWaitingConfig(), Seed: 1, N: n, M: forkM, Logs: true, Before: call})
	}
}
