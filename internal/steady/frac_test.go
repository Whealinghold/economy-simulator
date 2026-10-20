package steady_test

import (
	"math/big"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// TestFracAtLeast verifies the exact comparison of a fraction with an int64
// ratio, including an exact tie and a product that exceeds int64.
//
// Requirement: FR-CFG-044
func TestFracAtLeast(t *testing.T) {
	t.Parallel()
	huge := new(big.Int).Lsh(big.NewInt(1), 100) // 2^100, far above int64
	tests := []struct {
		name     string
		f        steady.Frac
		num, den int64
		want     bool
	}{
		// 4/5 against 10 000/12 500 = 4/5: an exact tie counts as at least.
		{"tie", steady.Frac{Num: big.NewInt(4), Den: big.NewInt(5)}, 10_000, 12_500, true},
		// 1/45 against 10 000/9 000 = 1.11: far below.
		{"below", steady.Frac{Num: big.NewInt(1), Den: big.NewInt(45)}, 10_000, 9_000, false},
		// 3/2 = 1.5 against 1/1: above.
		{"above", steady.Frac{Num: big.NewInt(3), Den: big.NewInt(2)}, 1, 1, true},
		// 2^100 / (2^100 + 1) is just under 1, and 2^100 * 1 is one short of
		// (2^100 + 1) * 1 only in exact arithmetic.
		{"huge just below", steady.Frac{Num: huge, Den: new(big.Int).Add(huge, big.NewInt(1))}, 1, 1, false},
		{"huge equal", steady.Frac{Num: huge, Den: huge}, 1, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.f.AtLeast(tt.num, tt.den); got != tt.want {
				t.Errorf("(%v/%v).AtLeast(%d, %d) = %v, want %v", tt.f.Num, tt.f.Den, tt.num, tt.den, got, tt.want)
			}
		})
	}
}

// TestFracAtLeast_RejectsBadDenominator verifies that a denominator below 1 is
// a programmer invariant that panics with a message that says so.
//
// Guard: the comparison divides by nothing, but a zero or negative denominator
// would flip or void the inequality silently.
func TestFracAtLeast_RejectsBadDenominator(t *testing.T) {
	t.Parallel()
	defer func() {
		msg, _ := recover().(string)
		if !strings.Contains(msg, "invariant") {
			t.Errorf("AtLeast(1, 0) panic = %q, want a message containing %q", msg, "invariant")
		}
	}()
	steady.Frac{Num: big.NewInt(1), Den: big.NewInt(1)}.AtLeast(1, 0)
}
