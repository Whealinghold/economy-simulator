package welfare

import (
	"math"
	"strings"
	"testing"
)

const (
	maxI = math.MaxInt64
	minI = math.MinInt64
)

// TestSatAdd_Extremes verifies that a sum that leaves int64 becomes the
// extreme of the operands' sign and any other sum is exact.
//
// Requirement: FR-WEL-042
func TestSatAdd_Extremes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b int64
		want int64
	}{
		{"in range", 2, 3, 5},
		{"max plus one saturates up", maxI, 1, maxI},
		{"min plus minus one saturates down", minI, -1, minI},
		{"max plus min is minus one", maxI, minI, -1},
		{"both max saturates up", maxI, maxI, maxI},
		{"both min saturates down", minI, minI, minI},
		{"zero plus min", 0, minI, minI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := satAdd(tt.a, tt.b); got != tt.want {
				t.Errorf("satAdd(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestSatSub_SignOfExactResult verifies that a difference that leaves int64
// takes the sign of the exact result, never of a sum with the negated second
// term.
//
// Requirement: FR-WEL-042
//
// Guard: -MinInt64 is not an int64, so a satSub written as satAdd(a, -b)
// would turn -5 - MinInt64 (exactly MaxInt64 - 4, in range) into a wrapped
// value.
func TestSatSub_SignOfExactResult(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b int64
		want int64
	}{
		{"in range", 5, 7, -2},
		{"max less minus one saturates up", maxI, -1, maxI},
		{"min less one saturates down", minI, 1, minI},
		{"zero less min saturates up", 0, minI, maxI},
		{"min less min is zero", minI, minI, 0},
		{"max less max is zero", maxI, maxI, 0},
		// MinInt64 = -2^63 and MaxInt64 = 2^63 - 1, so -5 - (-2^63) = 2^63 - 5 =
		// MaxInt64 - 4, which fits.
		{"minus five less min is max minus four", -5, minI, maxI - 4},
		{"min less minus one is in range", minI, -1, minI + 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := satSub(tt.a, tt.b); got != tt.want {
				t.Errorf("satSub(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestSatMulDiv_OnlyTheFinalValueSaturates verifies that a quotient whose
// product leaves int64 is exact when the quotient fits, saturates only when
// it does not, and takes the sign of a * b * c in each of the eight sign
// combinations.
//
// Requirement: FR-WEL-042
//
// Guard: a saturate-then-divide helper would turn MaxInt64 * 10 000 / 20 000
// into MaxInt64 / 20 000 instead of MaxInt64 / 2.
func TestSatMulDiv_OnlyTheFinalValueSaturates(t *testing.T) {
	t.Parallel()
	// 2^62 * 2^62 is 2^124, far above int64; divided by 1 it cannot fit.
	const big = int64(1) << 62
	tests := []struct {
		name    string
		a, b, c int64
		want    int64
	}{
		// MaxInt64 * 10 000 / 20 000 = MaxInt64 / 2 = 4 611 686 018 427 387 903.
		{"product above int64, quotient in range", maxI, 10_000, 20_000, maxI / 2},
		{"in range", 7, 6, 4, 10},
		{"zero numerator with a saturating factor", 0, maxI, 1, 0},
		{"zero factor with a saturating factor", maxI, 0, -1, 0},
		{"negative quotient in range", -maxI, 10_000, 20_000, -(maxI / 2)},
		{"+ + +", big, big, 1, maxI},
		{"+ + -", big, big, -1, minI},
		{"+ - +", big, -big, 1, minI},
		{"+ - -", big, -big, -1, maxI},
		{"- + +", -big, big, 1, minI},
		{"- + -", -big, big, -1, maxI},
		{"- - +", -big, -big, 1, maxI},
		{"- - -", -big, -big, -1, minI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := satMulDiv(tt.a, tt.b, tt.c); got != tt.want {
				t.Errorf("satMulDiv(%d, %d, %d) = %d, want %d", tt.a, tt.b, tt.c, got, tt.want)
			}
		})
	}
}

// TestSatMul_IsAQuotientByOne verifies satMul as satMulDiv(a, b, 1).
//
// Requirement: FR-WEL-042
func TestSatMul_IsAQuotientByOne(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b int64
		want int64
	}{
		{"in range", -6, 7, -42},
		{"overflow up", maxI, 2, maxI},
		{"overflow down", maxI, -2, minI},
		{"min times minus one", minI, -1, maxI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := satMul(tt.a, tt.b); got != tt.want {
				t.Errorf("satMul(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestSatMulDivCeil_RoundsUpAndSaturates verifies the ceiling quotient: it
// rounds toward positive infinity, is exact in range from a product above
// int64, and saturates by the sign rule.
//
// Requirement: FR-WEL-003, FR-WEL-042
func TestSatMulDivCeil_RoundsUpAndSaturates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		a, b, c int64
		want    int64
	}{
		{"exact", 6, 10, 4, 15},
		{"rounds up", 3, 10, 4, 8}, // 30 / 4 = 7.5
		{"negative rounds toward zero", -3, 10, 4, -7},
		// MaxInt64 * 10 000 / 20 000 = 4 611 686 018 427 387 903.5, ceiling ...904.
		{"in range from a product above int64", maxI, 10_000, 20_000, maxI/2 + 1},
		{"saturates up", 1 << 62, 1 << 62, 1, maxI},
		{"saturates down", 1 << 62, -(1 << 62), 1, minI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := satMulDivCeil(tt.a, tt.b, tt.c); got != tt.want {
				t.Errorf("satMulDivCeil(%d, %d, %d) = %d, want %d", tt.a, tt.b, tt.c, got, tt.want)
			}
		})
	}
}

// TestSat_ZeroDivisorIsAnInvariant verifies that a zero divisor, which the
// callers handle as their absent case first, panics with a message that says
// it is a violated programmer invariant.
//
// Guard: a zero divisor reaching a saturating quotient would otherwise be
// read as an overflow and return an extreme as if it were a result.
func TestSat_ZeroDivisorIsAnInvariant(t *testing.T) {
	t.Parallel()
	for name, f := range map[string]func(){
		"satMulDiv":     func() { satMulDiv(1, 1, 0) },
		"satMulDivCeil": func() { satMulDivCeil(1, 1, 0) },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				msg, _ := recover().(string)
				if !strings.Contains(msg, "violated programmer invariant") {
					t.Errorf("%s with a zero divisor panicked with %q, want a violated programmer invariant", name, msg)
				}
			}()
			f()
		})
	}
}

// TestSatQuot_ZeroIsNonNegative verifies the sign rule of the saturating
// extreme directly: the sign of a * b * c decides, and a zero operand counts as
// non-negative.
//
// Requirement: FR-WEL-042
//
// Guard: callers reach satQuot only on overflow, where no operand is zero, so
// the zero rule is observable only by calling it directly.
func TestSatQuot_ZeroIsNonNegative(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		a, b, c int64
		want    int64
	}{
		{"zero a", 0, 1, 1, maxI},
		{"zero b", 1, 0, 1, maxI},
		{"zero c", 1, 1, 0, maxI},
		{"zero a with one negative", 0, -1, 1, minI},
		{"zero b with one negative", -1, 0, 1, minI},
		{"zero c with one negative", 1, -1, 0, minI},
		{"all positive", 1, 1, 1, maxI},
		{"one negative", -1, 1, 1, minI},
		{"two negative", -1, -1, 1, maxI},
		{"three negative", -1, -1, -1, minI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := satQuot(tt.a, tt.b, tt.c); got != tt.want {
				t.Errorf("satQuot(%d, %d, %d) = %d, want %d", tt.a, tt.b, tt.c, got, tt.want)
			}
		})
	}
}

// TestSatMulDivCeil_NegativeDivisorIsNotInvariantViolation verifies that only
// a zero divisor panics: a negative divisor is a valid quotient.
//
// Requirement: FR-WEL-042
func TestSatMulDivCeil_NegativeDivisorIsNotInvariantViolation(t *testing.T) {
	t.Parallel()
	// 30 / -4 = -7.5, ceiling -7.
	if got := satMulDivCeil(3, 10, -4); got != -7 {
		t.Errorf("satMulDivCeil(3, 10, -4) = %d, want -7", got)
	}
}
