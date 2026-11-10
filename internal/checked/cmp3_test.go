package checked_test

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

func refCmp3(a1, a2, a3, b1, b2, b3 int64) int {
	a := new(big.Int).Mul(new(big.Int).Mul(big.NewInt(a1), big.NewInt(a2)), big.NewInt(a3))
	b := new(big.Int).Mul(new(big.Int).Mul(big.NewInt(b1), big.NewInt(b2)), big.NewInt(b3))
	return a.Cmp(b)
}

// TestCmp3_AgainstBig checks the exact comparison of two three-factor products
// against math/big at the edges of 189 bits, in ties and with a zero factor.
//
// Requirement: FR-CLB-061
func TestCmp3_AgainstBig(t *testing.T) {
	t.Parallel()
	m := int64(math.MaxInt64)
	tests := []struct {
		name string
		a, b [3]int64
	}{
		{"equal", [3]int64{2, 3, 5}, [3]int64{5, 6, 1}},
		{"less", [3]int64{2, 3, 5}, [3]int64{5, 6, 2}},
		{"greater", [3]int64{7, 3, 5}, [3]int64{5, 6, 3}},
		{"zero factor", [3]int64{0, m, m}, [3]int64{1, 1, 1}},
		{"both max", [3]int64{m, m, m}, [3]int64{m, m, m}},
		{"max against max minus one", [3]int64{m, m, m}, [3]int64{m, m, m - 1}},
		{"low word decides", [3]int64{1 << 62, 4, 1 << 61}, [3]int64{1 << 62, 4, 1<<61 + 1}},
		{"carry between words", [3]int64{m, m, 2}, [3]int64{1 << 62, 1 << 62, 8}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := checked.Cmp3(tt.a[0], tt.a[1], tt.a[2], tt.b[0], tt.b[1], tt.b[2])
			want := refCmp3(tt.a[0], tt.a[1], tt.a[2], tt.b[0], tt.b[1], tt.b[2])
			if !ok || got != want {
				t.Errorf("Cmp3(%v, %v) = %d, %v, want %d, true", tt.a, tt.b, got, ok, want)
			}
		})
	}
}

// TestCmp3_Random checks 2000 random products of mixed width against math/big,
// with a fixed seed.
//
// Requirement: FR-CLB-061
func TestCmp3_Random(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(7, 11))
	for i := range 2000 {
		var v [6]int64
		for j := range v {
			v[j] = int64(r.Uint64() >> (1 + r.UintN(63))) // widths from 0 to 63 bits
		}
		got, ok := checked.Cmp3(v[0], v[1], v[2], v[3], v[4], v[5])
		if want := refCmp3(v[0], v[1], v[2], v[3], v[4], v[5]); !ok || got != want {
			t.Fatalf("case %d: Cmp3(%v) = %d, %v, want %d, true", i, v, got, ok, want)
		}
	}
}

// TestCmp3_Negative verifies that a negative factor is refused, not compared.
//
// Requirement: FR-CLB-061
func TestCmp3_Negative(t *testing.T) {
	t.Parallel()
	for i := range 6 {
		v := [6]int64{1, 1, 1, 1, 1, 1}
		v[i] = -1
		if got, ok := checked.Cmp3(v[0], v[1], v[2], v[3], v[4], v[5]); ok || got != 0 {
			t.Errorf("Cmp3(%v) = %d, %v, want 0, false", v, got, ok)
		}
	}
}
