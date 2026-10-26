package economysim

import (
	"slices"
	"testing"
)

// TestReservedReasons_CoverTheLedgerNames verifies that every reason name the
// world knows (reasonNames) is in reservedReasons, that the list has no
// duplicate, and that every name is a valid ID, so a flow cannot take the name
// of a reason (spec-money-supply.md 4.5.2, plan 2-6 D325).
//
// Requirement: FR-MON-036
//
// Guard: a module that declares a reason must add its name to reservedReasons,
// or a flow ID could collide with it and two reasons would share one BoundaryFlow
// row. Without this test the omission shows only when a pack names a flow so.
func TestReservedReasons_CoverTheLedgerNames(t *testing.T) {
	t.Parallel()
	for _, name := range reasonNames {
		if name != "" && !slices.Contains(reservedReasons, string(name)) {
			t.Errorf("reason %q of reasonNames is not in reservedReasons", name)
		}
	}
	seen := map[string]bool{}
	for _, name := range reservedReasons {
		if seen[name] {
			t.Errorf("reservedReasons lists %q twice", name)
		}
		seen[name] = true
		if !validID(name) {
			t.Errorf("reservedReasons entry %q is not a valid ID", name)
		}
	}
}

// TestMoneySupply_CanonicalIsACopy verifies that canonical sorts the flows by ID,
// spells an empty period as a tick, returns nil for a module that is off and
// leaves its receiver unchanged.
//
// Requirement: NFR-MON-003
func TestMoneySupply_CanonicalIsACopy(t *testing.T) {
	t.Parallel()
	var off *MoneySupply
	if off.canonical() != nil {
		t.Error("canonical of a nil module is not nil")
	}
	m := &MoneySupply{Flows: []MoneyFlow{{ID: "b"}, {ID: "a", Per: PerYear}}}
	got := m.canonical()
	if got.Flows[0].ID != "a" || got.Flows[0].Per != PerYear || got.Flows[1].ID != "b" || got.Flows[1].Per != PerTick {
		t.Errorf("canonical flows = %+v, want a (year) then b (tick)", got.Flows)
	}
	if m.Flows[0].ID != "b" || m.Flows[1].Per != PerYear || m.Flows[0].Per != "" {
		t.Errorf("canonical changed its receiver: %+v", m.Flows)
	}
}
