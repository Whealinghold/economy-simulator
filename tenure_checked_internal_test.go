package economysim

import (
	"math"
	"strings"
	"testing"
)

// TestAddCount_CheckedSum verifies that the rent counts of a tick add with checked
// arithmetic: a sum that fits is returned, and one past int64 is a violated invariant and
// never a wrap.
//
// Requirement: FR-LND-037
//
// Guard: an unchecked += on a money count would wrap silently at the bound and the audit of
// the rent would compare a negative sum (architecture.md rule 3.5).
func TestAddCount_CheckedSum(t *testing.T) {
	t.Parallel()
	if got := addCount(40, 2); got != 42 {
		t.Errorf("addCount(40, 2) = %d, want 42", got)
	}
	if got := addCount(math.MaxInt64-1, 1); got != math.MaxInt64 {
		t.Errorf("addCount(MaxInt64-1, 1) = %d, want MaxInt64", got)
	}
	msg := invariantMessage(func() { addCount(math.MaxInt64, 1) })
	if !strings.Contains(msg, "rent count") {
		t.Errorf("addCount(MaxInt64, 1) panic = %q, want the invariant naming the rent count", msg)
	}
}

// TestRentAudited_Identity verifies the identity of FR-LND-037 for the counts of one
// endowment: paid plus forfeited equals due, and a sum past int64 fails instead of wrapping
// to a value that could equal due.
//
// Arithmetic: 4 paid and 2 forfeited are 6 due. MaxInt64 paid and 2 forfeited wrap to
// MinInt64+1; with a due of MinInt64+1 an unchecked sum would pass.
//
// Requirement: FR-LND-037
func TestRentAudited_Identity(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		c    tenureCount
		want bool
	}{
		{"nothing charged", tenureCount{}, true},
		{"paid and forfeited are the due", tenureCount{paid: 4, forfeited: 2, due: 6}, true},
		{"paid above the due", tenureCount{paid: 7, due: 6}, false},
		{"forfeited short of the due", tenureCount{paid: 4, forfeited: 1, due: 6}, false},
		{"sum wraps to the due", tenureCount{paid: math.MaxInt64, forfeited: 2, due: math.MinInt64 + 1}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := rentAudited(tt.c); got != tt.want {
				t.Errorf("rentAudited(%+v) = %v, want %v", tt.c, got, tt.want)
			}
		})
	}
}

// TestRentCounts_NoneBeforeFirstTick verifies that the counts of an endowment read as zero
// before the first tick has built the scratch, and for a cell past it, so the audit of a
// fresh or restored world compares nothing, and that P8.3a does nothing then.
//
// Requirement: FR-LND-037
//
// Guard: the audit and P8.3a index the scratch by endowment; without the check a world
// audited or stepped through P8.3a before P2.3 ran would index an empty slice.
func TestRentCounts_NoneBeforeFirstTick(t *testing.T) {
	t.Parallel()
	w := tenureWorld(t, nil)
	if len(w.tenure.rt.counts) != 0 {
		t.Fatalf("len(counts) = %d before the first step, want 0", len(w.tenure.rt.counts))
	}
	if got := w.tenure.rentCounts(0); got != (tenureCount{}) {
		t.Errorf("rentCounts(0) = %+v before the first step, want the zero count", got)
	}
	w.p83aRent()
	if err := w.auditTenure(); err != nil {
		t.Errorf("auditTenure() error = %v before the first step, want nil", err)
	}
	if err := w.Step(); err != nil {
		t.Fatalf("Step error = %v, want nil", err)
	}
	if got := w.tenure.rentCounts(len(w.tenure.rt.counts)); got != (tenureCount{}) {
		t.Errorf("rentCounts(past the end) = %+v, want the zero count", got)
	}
}

// TestAudit_RentSumPastInt64Fails verifies that the tenure audit reports a paid and a
// forfeited count whose sum does not fit int64.
//
// Requirement: FR-LND-037
func TestAudit_RentSumPastInt64Fails(t *testing.T) {
	t.Parallel()
	w := newMill(t, rationConfig(10, 5_000, false), WithSeed(1), WithAudit())
	toStart(t, w)
	c := &w.tenure.rt.counts[0]
	c.paid, c.forfeited, c.due = math.MaxInt64, 2, math.MinInt64+1
	if err := w.auditTenure(); err == nil {
		t.Errorf("auditTenure() error = nil, want an error for a sum past int64")
	}
}
