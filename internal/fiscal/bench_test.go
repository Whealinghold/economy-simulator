package fiscal_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/fiscal"
)

// TestCut_AllocsPerRun verifies that a cut allocates nothing once the scratch and
// the destination have grown, the allocation limit of NFR-FIS-001.
//
// Requirement: NFR-FIS-001
func TestCut_AllocsPerRun(t *testing.T) {
	var s fiscal.Scratch
	rs := almsRecipients()
	dst, _ := s.Cut(nil, 4_135, rs)
	if got := testing.AllocsPerRun(100, func() { dst, _ = s.Cut(dst, 4_135, rs) }); got != 0 {
		t.Errorf("Cut allocates %v per run after growth, want 0", got)
	}
}

// BenchmarkCut reports the cost of the cut of 20 recipients, with allocations.
func BenchmarkCut(b *testing.B) {
	var s fiscal.Scratch
	rs := almsRecipients()
	dst, _ := s.Cut(nil, 4_135, rs)
	b.ReportAllocs()
	for b.Loop() {
		dst, _ = s.Cut(dst, 4_135, rs)
	}
}

// BenchmarkDue reports the cost of the due of one payer, with allocations.
func BenchmarkDue(b *testing.B) {
	r := fiscal.Rule{Shape: fiscal.Rate, RateBP: 100, Exempt: 2_000, Scale: 10_000, Period: 1, MaxMoney: maxMoney}
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.Due(1, 2_940, 0)
	}
}

// BenchmarkDutyDue reports the cost of the due of one duty, with allocations.
func BenchmarkDutyDue(b *testing.B) {
	var base fiscal.Base
	base.Add(1_350)
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = fiscal.DutyDue(base, 300, 10_000, 0, maxMoney)
	}
}
