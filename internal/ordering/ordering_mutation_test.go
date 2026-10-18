package ordering_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// Equivalent mutants of ordering.go found by gomu (no test can kill them):
//   - Bucket `p < 1` -> `p <= 1`: for p == 1 the product (Weyl2 * 1) >> 32 is
//     0 because Weyl2 < 2^32, so both branches return 0.
//   - Bucket `return 0` -> `return 0` (identical source).
//   - Bucket `hi<<32 | lo>>32` -> `^`: hi<<32 has its low 32 bits clear and
//     lo>>32 fits in 32 bits, so the operands share no set bit.

// TestBucket_NonPositivePeriod verifies that a period below 1 has no bucket,
// including a negative one, which would otherwise convert to a huge uint64.
//
// Requirement: FR-WLD-080
func TestBucket_NonPositivePeriod(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		p    int64
	}{
		{"zero", 0},
		{"minus one", -1},
		{"min int64", -1 << 63},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// id 1 has Weyl2 = 1779033703, so p = -1 would give a nonzero
			// bucket without the guard.
			for _, id := range []int64{1, 2, 17} {
				if got := ordering.Bucket(id, tt.p); got != 0 {
					t.Errorf("Bucket(%d, %d) = %d, want 0", id, tt.p, got)
				}
			}
		})
	}
}
