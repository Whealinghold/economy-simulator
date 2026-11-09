package measure_test

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/measure"
)

// TestIsqrtNearest_NoTie verifies that the result is the integer nearest to the
// square root for twenty thousand pseudo-random products below 2^62 (the bound
// of two indices of at most 2^31 - 1), and on the examples of the spec.
//
// A result r is nearest when (r - 1/2)^2 < n < (r + 1/2)^2, that is
// 4r^2 - 4r + 1 < 4n < 4r^2 + 4r + 1; neither side can equal 4n, because 4n is
// even and both bounds are odd, which is why there is no tie to break.
//
// Requirement: FR-IND-021
func TestIsqrtNearest_NoTie(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(9, 10))
	for range 20000 {
		n := (r.Uint64() >> 2) >> uint(r.IntN(40))
		got := measure.IsqrtNearest(n)
		if !(4*got*got-4*got+1 < 4*n && 4*n < 4*got*got+4*got+1) {
			t.Fatalf("IsqrtNearest(%d) = %d, which is not the nearest integer to the square root", n, got)
		}
	}
	tests := []struct {
		n, want uint64
	}{
		{0, 0}, {1, 1}, {2, 1}, {3, 2}, {4, 2}, {6, 2}, {7, 3},
		// 12 105 * 12 083 = 146 264 715; sqrt = 12 093.995, so 12 094 (spec 18.4)
		{146264715, 12094},
		{math.MaxUint64, 1 << 32},
	}
	for _, tt := range tests {
		if got := measure.IsqrtNearest(tt.n); got != tt.want {
			t.Errorf("IsqrtNearest(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}
