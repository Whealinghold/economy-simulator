package occupation_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/occupation"
)

// TestAllocs_PureFunctionsAllocateNothing verifies that the functions the
// default spawner calls per agent allocate nothing when the caller supplies the
// slices.
//
// Guard: Decide runs once per region per tick over every agent and must
// allocate nothing with the world's scratch (NFR-SPN-001). A closure or a
// slice built inside one of these would show up as a per-agent allocation.
//
// Requirement: NFR-SPN-001
func TestAllocs_PureFunctionsAllocateNothing(t *testing.T) {
	ls := []occupation.Leaver{
		{Ref: 0, Privation: 9, Weight: 40, Unfed: 7},
		{Ref: 1, Privation: 9, Weight: 1, Unfed: 1},
		{Ref: 2, Privation: 12, Weight: 3, Unfed: 2},
	}
	dst := make([]occupation.Exit, 0, len(ls))
	hosts := []occupation.Host{{Ref: 0, Weight: 3}, {Ref: 1, Weight: 1}, {Ref: 2, Weight: 5}}
	var sink int64
	var bsink bool
	tests := []struct {
		name string
		f    func()
	}{
		{"Exits", func() { out, cut := occupation.Exits(dst, ls, 5, 1000); sink += cut + int64(len(out)) }},
		{"Place", func() { r, room, _ := occupation.Place(hosts, 4); sink += int64(r) + room }},
		{"Beats", func() { bsink = occupation.Beats(9041, 8219, 1000) }},
		{"CapitalOK", func() { bsink = occupation.CapitalOK(600, 100, 1_825_000, 100, 1000, 365) }},
		{"Expected", func() { e, _ := occupation.Expected(1200, 250, 1000, 365); sink += e }},
		{"Viability", func() { v, _ := occupation.Viability(1200, 1000); sink += v }},
		{"ReviewDue", func() { bsink = occupation.ReviewDue(17, 10, 10) }},
		{"Rung3", func() { bsink = occupation.Rung3(20, 100, 80, 20) }},
		{"Renewal", func() { n, c, _ := occupation.Renewal(5000, 1, 5000); sink += n + c }},
		{"Remaining", func() { r, _ := occupation.Remaining(12000, 40, 36); sink += r }},
	}
	for _, tt := range tests {
		if got := testing.AllocsPerRun(100, tt.f); got != 0 {
			t.Errorf("%s allocates %v times per call, want 0", tt.name, got)
		}
	}
	_, _ = sink, bsink
}
