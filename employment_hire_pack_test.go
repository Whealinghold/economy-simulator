package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestHirePack_OverlayLoadsValidatesAndBuilds verifies that overlay/employment-hire.toml
// layered over the pack one-demand loads as the hire form, validates with no warning
// of 14.2 (the cooks have a taste row, so their profit has an outlet), builds a world,
// and steps it for a few ticks without error: the form is data and does nothing at run
// time yet, so the stream of the pack is untouched.
//
// Requirement: FR-EMP-002, FR-EMP-003, FR-EMP-004, FR-EMP-059
func TestHirePack_OverlayLoadsValidatesAndBuilds(t *testing.T) {
	t.Parallel()
	cfg := loadPack(t, packOneDemandHire...)
	m := cfg.Modules.Employment
	if m == nil || !m.Hire || m.MaxOpen != 1_000 {
		t.Fatalf("pack Modules.Employment = %+v, want the hire form with MaxOpen 1000", m)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate(one-demand + hire overlay) = %v, want nil", err)
	}
	if got := employmentWarningRules(cfg); len(got) != 0 {
		t.Errorf("Warnings(one-demand + hire overlay) employment rules = %q, want none", got)
	}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
	if err != nil {
		t.Fatalf("NewWorld(one-demand + hire overlay) = %v, want nil", err)
	}
	for range 3 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
	}
}
