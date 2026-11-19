package economysim_test

import (
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestFiscalHash verifies the hash of the fiscal module (spec-fiscal.md 14, FR-FIS-005):
// the module off leaves the hash of the configuration as it was, on it is part of the
// hash, an empty module differs from none, the lists hash in ID order and the seasons of
// a window in the calendar's order, a flow scale and a Label are not hashed, and a change
// of a rate, an exemption, a limit or a target changes it.
//
// Requirement: FR-FIS-005
func TestFiscalHash(t *testing.T) {
	t.Parallel()
	base := fiscalConfig()
	off := fiscalConfig()
	off.Modules.Fiscal = nil
	if economysim.HashConfig(base) == economysim.HashConfig(off) {
		t.Error("HashConfig with the module on equals the hash with it off, want a different hash")
	}
	empty := fiscalConfig()
	empty.Modules.Fiscal = &economysim.Fiscal{}
	if economysim.HashConfig(empty) == economysim.HashConfig(off) {
		t.Error("HashConfig with an empty non-nil Fiscal equals the hash with none, want a different hash")
	}

	tests := []struct {
		name   string
		mutate func(m *economysim.Fiscal)
		moves  bool
	}{
		{"a label", func(m *economysim.Fiscal) { m.Taxes[0].Label = "other"; m.Releases[0].Label = "x" }, false},
		{"the lists in the other order", func(m *economysim.Fiscal) { m.Taxes[0], m.Taxes[2] = m.Taxes[2], m.Taxes[0] }, false},
		{"seasons added to a window", func(m *economysim.Fiscal) { m.Taxes[0].Window.Seasons = []string{"winter", "spring"} }, true},
		{"an empty Per spelled tick", func(m *economysim.Fiscal) { m.Taxes[0].Per = economysim.PerTick }, false},
		{"an empty Scope spelled any", func(m *economysim.Fiscal) { m.Duties[0].Scope = economysim.ScopeAny }, false},
		{"a rate", func(m *economysim.Fiscal) { m.Taxes[1].RateBP++ }, true},
		{"an exemption", func(m *economysim.Fiscal) { m.Taxes[1].Exempt++ }, true},
		{"a limit", func(m *economysim.Fiscal) { m.Releases[0].LimitBP++ }, true},
		{"a target jurisdiction", func(m *economysim.Fiscal) { m.Taxes[2].Target.Jurisdiction = "inland" }, true},
		{"an authority", func(m *economysim.Fiscal) { m.Payouts[0].Authority = "inland" }, true},
		{"a window start", func(m *economysim.Fiscal) { m.Purchases[0].Window.Start = 5 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := fiscalConfig()
			tt.mutate(c.Modules.Fiscal)
			if moved := economysim.HashConfig(c) != economysim.HashConfig(base); moved != tt.moves {
				t.Errorf("HashConfig after %s moved = %v, want %v", tt.name, moved, tt.moves)
			}
		})
	}
}

// TestFiscalHash_SeasonsInCalendarOrder verifies that two spellings of one window, the
// seasons listed in either order, hash alike.
//
// Requirement: FR-FIS-005
func TestFiscalHash_SeasonsInCalendarOrder(t *testing.T) {
	t.Parallel()
	a, b := fiscalConfig(), fiscalConfig()
	a.Modules.Fiscal.Taxes[0].Window.Seasons = []string{"spring", "winter"}
	b.Modules.Fiscal.Taxes[0].Window.Seasons = []string{"winter", "spring"}
	if economysim.HashConfig(a) != economysim.HashConfig(b) {
		t.Error("HashConfig of the seasons [spring winter] and [winter spring] differ, want equal: the calendar's order is canonical")
	}
}
