package facility_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/facility"
)

// BenchmarkAssign reports the cost and allocations of one start over the
// largest list an owner can hold (MaxFacilitiesPerAgent, 64).
//
// Requirement: NFR-CAP-004
func BenchmarkAssign(b *testing.B) {
	b.ReportAllocs()
	fac := make([]facility.Fac, 64)
	dst := make([]facility.Placement, 0, 64)
	for b.Loop() {
		for i := range fac {
			fac[i] = facility.Fac{Ref: i, Table: int64(1000 - i), Condition: int64(1000 - i), Cap: 3, WearPerUse: 2}
		}
		dst, _ = facility.Assign(dst, 100, fac)
	}
}

// BenchmarkEff reports the cost of the effect fraction.
//
// Requirement: NFR-CAP-004
func BenchmarkEff(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		facility.Eff(6000, 1999, 2000)
	}
}

// TestAssign_AllocsPerRun verifies that a start with a reused destination
// allocates nothing.
//
// Requirement: NFR-CAP-002
func TestAssign_AllocsPerRun(t *testing.T) {
	fac := make([]facility.Fac, 16)
	dst := make([]facility.Placement, 0, 16)
	got := testing.AllocsPerRun(50, func() {
		for i := range fac {
			fac[i] = facility.Fac{Ref: i, Table: int64(100 - i), Condition: 100, Cap: 2, WearPerUse: 1}
		}
		dst, _ = facility.Assign(dst, 20, fac)
	})
	if got != 0 {
		t.Errorf("Assign allocated %v times per run, want 0", got)
	}
}
