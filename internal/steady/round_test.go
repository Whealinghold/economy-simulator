package steady_test

import (
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// bigOf parses a decimal integer.
func bigOf(t *testing.T, s string) *big.Int {
	t.Helper()
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		t.Fatalf("bad integer %q", s)
	}
	return v
}

// TestRound_Rules verifies the three rounding rules on exact halves, thirds,
// negative values and a negative denominator.
//
// Requirement: FR-VIA-015
func TestRound_Rules(t *testing.T) {
	t.Parallel()
	// Each row: num/den, then half-to-even, floor and ceil. 5/2 = 2.5 is a half
	// between 2 and 3, so half-even takes the even 2; 7/2 = 3.5 takes 4;
	// -5/2 = -2.5 takes -2; -7/2 = -3.5 takes -4; 1/3 and 2/3 are not halves.
	tests := []struct {
		num, den          int64
		even, floor, ceil int64
	}{
		{5, 2, 2, 2, 3}, {7, 2, 4, 3, 4}, {-5, 2, -2, -3, -2}, {-7, 2, -4, -4, -3},
		{1, 2, 0, 0, 1}, {-1, 2, 0, -1, 0}, {3, 2, 2, 1, 2}, {-3, 2, -2, -2, -1},
		{1, 3, 0, 0, 1}, {2, 3, 1, 0, 1}, {-1, 3, 0, -1, 0}, {-2, 3, -1, -1, 0},
		{6, 3, 2, 2, 2}, {-6, 3, -2, -2, -2}, {0, 5, 0, 0, 0},
		{5, -2, -2, -3, -2}, {-5, -2, 2, 2, 3}, {7, -3, -2, -3, -2},
	}
	for _, tt := range tests {
		num, den := big.NewInt(tt.num), big.NewInt(tt.den)
		for name, c := range map[string]struct {
			f    func(a, b *big.Int, lo, hi int64) (int64, bool)
			want int64
		}{
			"half even": {steady.RoundHalfEven, tt.even},
			"floor":     {steady.Floor, tt.floor},
			"ceil":      {steady.Ceil, tt.ceil},
		} {
			got, ok := c.f(num, den, math.MinInt64, math.MaxInt64)
			if !ok || got != c.want {
				t.Errorf("%s(%d, %d) = %d, %v, want %d, true", name, tt.num, tt.den, got, ok, c.want)
			}
		}
	}
}

// TestRound_Bounds verifies that a result outside int64 or outside the caller's
// bound is reported, with the value 0, by all three functions.
//
// Requirement: FR-VIA-015
func TestRound_Bounds(t *testing.T) {
	t.Parallel()
	over := bigOf(t, "9223372036854775808")   // MaxInt64 + 1
	under := bigOf(t, "-9223372036854775809") // MinInt64 - 1
	one := big.NewInt(1)
	tests := []struct {
		name   string
		num    *big.Int
		den    *big.Int
		lo, hi int64
		want   int64
		ok     bool
	}{
		{"one past MaxInt64", over, one, math.MinInt64, math.MaxInt64, 0, false},
		{"one past MinInt64", under, one, math.MinInt64, math.MaxInt64, 0, false},
		{"MaxInt64", bigOf(t, "9223372036854775807"), one, math.MinInt64, math.MaxInt64, math.MaxInt64, true},
		{"MinInt64", bigOf(t, "-9223372036854775808"), one, math.MinInt64, math.MaxInt64, math.MinInt64, true},
		{"huge quotient of huge numbers", new(big.Int).Lsh(one, 200), new(big.Int).Lsh(one, 100), math.MinInt64, math.MaxInt64, 0, false},
		{"huge ratio that fits", new(big.Int).Lsh(one, 200), new(big.Int).Lsh(one, 199), math.MinInt64, math.MaxInt64, 2, true},
		{"at the upper bound", big.NewInt(10), one, 0, 10, 10, true},
		{"above the upper bound", big.NewInt(11), one, 0, 10, 0, false},
		{"at the lower bound", big.NewInt(0), one, 0, 10, 0, true},
		{"below the lower bound", big.NewInt(-1), one, 0, 10, 0, false},
		{"empty interval", big.NewInt(5), one, 6, 4, 0, false},
	}
	for _, tt := range tests {
		for name, f := range map[string]func(a, b *big.Int, lo, hi int64) (int64, bool){
			"RoundHalfEven": steady.RoundHalfEven, "Floor": steady.Floor, "Ceil": steady.Ceil,
		} {
			got, ok := f(tt.num, tt.den, tt.lo, tt.hi)
			if got != tt.want || ok != tt.ok {
				t.Errorf("%s(%s) [%s / %s in %d..%d] = %d, %v, want %d, %v",
					name, tt.name, tt.num, tt.den, tt.lo, tt.hi, got, ok, tt.want, tt.ok)
			}
		}
	}
}

// TestRound_ZeroDenominatorPanics verifies that a zero denominator is a
// programmer invariant that panics with that word, not a result.
//
// Guard: a zero denominator would otherwise be an overflow report that hides a
// defect of the caller.
func TestRound_ZeroDenominatorPanics(t *testing.T) {
	t.Parallel()
	for name, f := range map[string]func(a, b *big.Int, lo, hi int64) (int64, bool){
		"RoundHalfEven": steady.RoundHalfEven, "Floor": steady.Floor, "Ceil": steady.Ceil,
	} {
		func() {
			defer func() {
				msg, _ := recover().(string)
				if !strings.Contains(msg, "invariant") {
					t.Errorf("%s(1, 0) panic = %q, want a message with %q", name, msg, "invariant")
				}
			}()
			f(big.NewInt(1), new(big.Int), math.MinInt64, math.MaxInt64)
		}()
	}
}
