package fiscal_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/fiscal"
)

// TestToll_Example verifies the toll of spec-fiscal.md 7.3 and 18.2: with one
// truncation, Amount 3 at scale 5 000 for 3 carrying units is
// floor(3 * 5 000 * 3 / 10^4) = floor(4.5) = 4, not 3 * floor(1.5) = 3; and the
// ferry of 4 a unit for 3 units is 12, a toll of 2 for 3 units is 6.
//
// Requirement: FR-FIS-034
func TestToll_Example(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		amount, scale int64
		n, want       int64
	}{
		{"one truncation at half scale", 3, 5_000, 3, 4},
		{"ferry on exit", 4, 10_000, 3, 12},
		{"gate on entry", 2, 10_000, 3, 6},
		{"zero units count as one", 4, 10_000, 0, 4},
		{"scale 0 charges nothing", 4, 0, 3, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := fiscal.Toll(tt.amount, tt.scale, tt.n); !ok || got != tt.want {
				t.Errorf("Toll(%d, %d, %d) = %d, %v; want %d, true", tt.amount, tt.scale, tt.n, got, ok, tt.want)
			}
		})
	}
}

// TestToll_Domain verifies that a negative argument and a toll that leaves int64
// are reported, the refusal overflow of spec-transport.md 5.4 row 23.
//
// Requirement: FR-FIS-034, NFR-FIS-003
func TestToll_Domain(t *testing.T) {
	t.Parallel()
	for _, c := range [][3]int64{{-1, 1, 1}, {1, -1, 1}, {1, 1, -1}, {math.MaxInt64, math.MaxInt64, 1}, {math.MaxInt64 / 2, 10_000, math.MaxInt64}} {
		if got, ok := fiscal.Toll(c[0], c[1], c[2]); ok {
			t.Errorf("Toll(%d, %d, %d) = %d, true; want ok false", c[0], c[1], c[2], got)
		}
	}
}

// TestToll_WideProduct verifies that amount times scale may pass int64 while the
// toll fits: the product is held wide.
//
// Requirement: FR-FIS-034, NFR-FIS-003
func TestToll_WideProduct(t *testing.T) {
	t.Parallel()
	if got, ok := fiscal.Toll(math.MaxInt64, 2, 1); !ok || got != 1_844_674_407_370_955 {
		t.Errorf("Toll(MaxInt64, 2, 1) = %d, %v; want 1844674407370955, true", got, ok)
	}
}

// TestTotal verifies the sum of the crossings of a shipment and that an overflow
// or a negative amount is reported.
//
// Requirement: FR-FIS-035
func TestTotal(t *testing.T) {
	t.Parallel()
	if got, ok := fiscal.Total([]int64{12, 6, 0, 4}); !ok || got != 22 {
		t.Errorf("Total(12, 6, 0, 4) = %d, %v; want 22, true", got, ok)
	}
	if got, ok := fiscal.Total(nil); !ok || got != 0 {
		t.Errorf("Total(nil) = %d, %v; want 0, true", got, ok)
	}
	for _, in := range [][]int64{{math.MaxInt64, 1}, {1, -1}} {
		if got, ok := fiscal.Total(in); ok {
			t.Errorf("Total(%v) = %d, true; want ok false", in, got)
		}
	}
}
