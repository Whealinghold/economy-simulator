package occupation_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/occupation"
)

// TestRefuse_Cases verifies one failing check per slice reason, a decision that
// passes, and the order of the checks when several fail.
//
// Requirement: FR-SPN-004, FR-SPN-010
func TestRefuse_Cases(t *testing.T) {
	t.Parallel()
	ok := occupation.Facts{AgentKnown: true, RoleKnown: true, CauseSupported: true, Households: 2, Weight: 5}
	with := func(f func(*occupation.Facts)) occupation.Facts { x := ok; f(&x); return x }
	tests := []struct {
		name string
		kind occupation.Kind
		f    occupation.Facts
		want occupation.Refusal
	}{
		{"remove passes", occupation.KindRemove, ok, occupation.RefusalNone},
		{"reduce passes", occupation.KindReduce, ok, occupation.RefusalNone},
		{"switch passes", occupation.KindSwitch, ok, occupation.RefusalNone},
		{"enter passes", occupation.KindEnter, ok, occupation.RefusalNone},
		{"unknown agent", occupation.KindRemove, with(func(f *occupation.Facts) { f.AgentKnown = false }), occupation.UnknownAgent},
		{"enter needs no agent", occupation.KindEnter, with(func(f *occupation.Facts) { f.AgentKnown = false }), occupation.RefusalNone},
		{"unknown role on a switch", occupation.KindSwitch, with(func(f *occupation.Facts) { f.RoleKnown = false }), occupation.UnknownRole},
		{"unknown role on an enter", occupation.KindEnter, with(func(f *occupation.Facts) { f.RoleKnown = false }), occupation.UnknownRole},
		{"a reduce names no role", occupation.KindReduce, with(func(f *occupation.Facts) { f.RoleKnown = false }), occupation.RefusalNone},
		{"external agent", occupation.KindSwitch, with(func(f *occupation.Facts) { f.External = true }), occupation.ExternalAgent},
		{"enter ignores external", occupation.KindEnter, with(func(f *occupation.Facts) { f.External = true }), occupation.RefusalNone},
		{"same role", occupation.KindSwitch, with(func(f *occupation.Facts) { f.SameRole = true }), occupation.SameRole},
		{"same role on a remove is ignored", occupation.KindRemove, with(func(f *occupation.Facts) { f.SameRole = true }), occupation.RefusalNone},
		{"unknown agent on a reduce", occupation.KindReduce, with(func(f *occupation.Facts) { f.AgentKnown = false }), occupation.UnknownAgent},
		{"external agent on a reduce", occupation.KindReduce, with(func(f *occupation.Facts) { f.External = true }), occupation.ExternalAgent},
		{"reduce by 0", occupation.KindReduce, with(func(f *occupation.Facts) { f.Households = 0 }), occupation.BadHouseholds},
		{"reduce by more than the weight", occupation.KindReduce, with(func(f *occupation.Facts) { f.Households = 6 }), occupation.BadHouseholds},
		{"reduce by the whole weight", occupation.KindReduce, with(func(f *occupation.Facts) { f.Households = 5 }), occupation.RefusalNone},
		{"enter of 0", occupation.KindEnter, with(func(f *occupation.Facts) { f.Households = 0 }), occupation.BadHouseholds},
		{"unsupported cause", occupation.KindReduce, with(func(f *occupation.Facts) { f.CauseSupported = false }), occupation.UnsupportedCause},
		{"unsupported cause on a remove is ignored", occupation.KindRemove, with(func(f *occupation.Facts) { f.CauseSupported = false }), occupation.RefusalNone},
		{
			"order: unknown agent first", occupation.KindSwitch,
			occupation.Facts{External: true, SameRole: true}, occupation.UnknownAgent,
		},
		{
			"order: unknown role before external", occupation.KindSwitch,
			occupation.Facts{AgentKnown: true, External: true, SameRole: true}, occupation.UnknownRole,
		},
		{
			"order: external before same role", occupation.KindSwitch,
			occupation.Facts{AgentKnown: true, RoleKnown: true, External: true, SameRole: true}, occupation.ExternalAgent,
		},
		{
			"order: bad households before unsupported cause", occupation.KindReduce,
			occupation.Facts{AgentKnown: true, Households: 0, Weight: 5}, occupation.BadHouseholds,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := occupation.Refuse(tt.kind, tt.f); got != tt.want {
				t.Errorf("Refuse(%d, %+v) = %d, want %d", tt.kind, tt.f, got, tt.want)
			}
		})
	}
}

// TestRefusal_UnsupportedCauseIsLast verifies that the order of the Refusal
// values is the order of the checks of SPN 4, unsupported_cause appended last.
//
// Guard: the root maps each value to a Reason and the executor reports the
// first failing check; a value inserted before the last would renumber
// reasons a later module appends.
//
// Requirement: FR-SPN-004
func TestRefusal_UnsupportedCauseIsLast(t *testing.T) {
	t.Parallel()
	want := []occupation.Refusal{occupation.UnknownAgent, occupation.UnknownRole, occupation.ExternalAgent,
		occupation.SameRole, occupation.BadHouseholds, occupation.UnsupportedCause}
	for i, r := range want {
		if int(r) != i+1 {
			t.Errorf("Refusal %d = %d, want %d", i, r, i+1)
		}
	}
	if occupation.RefusalEnd != occupation.UnsupportedCause+1 {
		t.Errorf("RefusalEnd = %d, want %d", occupation.RefusalEnd, occupation.UnsupportedCause+1)
	}
}
