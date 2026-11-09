package rumour_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/rumour"
)

// TestLag_FloorAndCeiling verifies Lag = 1 + ceil(distance / speed): the floor
// of 2 for any distance of at least 1, the ceiling on a part tick, an exact
// division, and the asymmetric table of spec-information.md 12.1 at speed 40.
//
// Requirement: FR-INF-008, FR-INF-009
func TestLag_FloorAndCeiling(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		distance, speed int64
		want            int64
	}{
		{"distance 1 at the largest speed", 1, 1_000_000_000, 2},
		{"distance 1 at speed 1", 1, 1, 2},
		{"exact division, 40 / 40 = 1", 40, 40, 2},
		{"one over, ceil(41 / 40) = 2", 41, 40, 3},
		{"40 at speed 25, ceil(1.6) = 2", 40, 25, 3},
		{"55 at speed 40, lower to middle", 55, 40, 3},
		{"120 at speed 40, lower to upper", 120, 40, 4},
		{"30 at speed 40, middle to lower", 30, 40, 2},
		{"70 at speed 40, middle to upper", 70, 40, 3},
		{"70 at speed 40, upper to lower", 70, 40, 3},
		{"40 at speed 40, upper to middle", 40, 40, 2},
		{"speed 1 is the distance plus 1", 99, 1, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := rumour.Lag(tt.distance, tt.speed)
			if !ok || got != tt.want {
				t.Errorf("Lag(%d, %d) = %d, %v, want %d, true", tt.distance, tt.speed, got, ok, tt.want)
			}
		})
	}
}

// TestLag_Overflow verifies that a lag that does not fit int64 is reported as
// ok false and not wrapped, and that a large distance whose lag fits is not
// refused because of an intermediate sum.
//
// Fix: found by FuzzLag against its math/big oracle at once. The first version
// added speed - 1 to the distance before dividing, and Lag(MaxInt64, 2) was
// refused although its lag, 2^62 + 1, fits.
//
// Requirement: FR-INF-008
func TestLag_Overflow(t *testing.T) {
	t.Parallel()
	if got, ok := rumour.Lag(math.MaxInt64, 1); ok {
		t.Errorf("Lag(MaxInt64, 1) = %d, true, want ok false (1 + MaxInt64 does not fit)", got)
	}
	// 1 + ceil((2^63 - 1) / 2) = 1 + 2^62 = 4611686018427387905.
	if got, ok := rumour.Lag(math.MaxInt64, 2); !ok || got != 4611686018427387905 {
		t.Errorf("Lag(MaxInt64, 2) = %d, %v, want 4611686018427387905, true", got, ok)
	}
	// The largest distance whose lag fits: MaxInt64-1 at speed 1 gives MaxInt64.
	if got, ok := rumour.Lag(math.MaxInt64-1, 1); !ok || got != math.MaxInt64 {
		t.Errorf("Lag(MaxInt64-1, 1) = %d, %v, want MaxInt64, true", got, ok)
	}
}

// TestLag_BadInput verifies that a distance below 1 (the same region, or no
// path) and a speed below 1 have no lag.
//
// Requirement: FR-INF-008
func TestLag_BadInput(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ distance, speed int64 }{
		{0, 40}, {-1, 40}, {55, 0}, {55, -3}, {0, 0}, {math.MinInt64, math.MinInt64},
	} {
		if got, ok := rumour.Lag(tt.distance, tt.speed); ok || got != 0 {
			t.Errorf("Lag(%d, %d) = %d, %v, want 0, false", tt.distance, tt.speed, got, ok)
		}
	}
}

// FuzzLag verifies, for any distance and speed, that Lag never panics, that
// it is at least 2 for a distance and a speed of at least 1 whenever it fits,
// and that it agrees with a math/big oracle, ok being false exactly when an
// input is below 1 or the exact lag does not fit int64.
//
// Requirement: NFR-INF-004
func FuzzLag(f *testing.F) {
	f.Add(int64(55), int64(40))
	f.Add(int64(1), int64(1_000_000_000))
	f.Add(int64(math.MaxInt64), int64(1))
	f.Add(int64(math.MaxInt64), int64(2))
	f.Add(int64(0), int64(0))
	f.Add(int64(-7), int64(3))
	f.Fuzz(func(t *testing.T, distance, speed int64) {
		got, ok := rumour.Lag(distance, speed)
		if distance < 1 || speed < 1 {
			if ok {
				t.Fatalf("Lag(%d, %d) = %d, true, want ok false for an input below 1", distance, speed, got)
			}
			return
		}
		d, s := big.NewInt(distance), big.NewInt(speed)
		want := new(big.Int).Add(d, new(big.Int).Sub(s, big.NewInt(1)))
		want.Div(want, s).Add(want, big.NewInt(1))
		if ok != want.IsInt64() || ok && (got != want.Int64() || got < 2) {
			t.Fatalf("Lag(%d, %d) = %d, %v, want %v (ok %v), at least 2", distance, speed, got, ok, want, want.IsInt64())
		}
	})
}
