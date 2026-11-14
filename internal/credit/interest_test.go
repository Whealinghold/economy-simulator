package credit_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/credit"
)

const (
	maxMoney   = int64(1_000_000_000_000_000_000)
	maxRateBP  = int64(100_000)
	maxTerm    = int64(3_650)
	ticksYear  = int64(365)
	basisTicks = 10_000
)

// exactInterest is the oracle: the exact quotient of the numerator by the
// denominator in big integers, rounded half to even.
func exactInterest(x, rate, term, tpy int64) *big.Int {
	num := new(big.Int).Mul(big.NewInt(x), big.NewInt(rate))
	num.Mul(num, big.NewInt(term))
	den := new(big.Int).Mul(big.NewInt(basisTicks), big.NewInt(tpy))
	q, m := new(big.Int).DivMod(num, den, new(big.Int))
	twice := new(big.Int).Lsh(m, 1)
	switch c := twice.Cmp(den); {
	case c > 0, c == 0 && q.Bit(0) == 1:
		q.Add(q, big.NewInt(1))
	}
	return q
}

// TestInterest_Exact verifies the interest against the big-integer quotient: the
// worked examples of spec-credit.md 20.1 and 20.3, the half-to-even midpoint, and
// the largest legal numerator MaxMoney * MaxCreditRateBP * MaxCreditTerm, which
// needs 128 bits.
//
// Requirement: FR-CRD-027
func TestInterest_Exact(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		x, rate, term, tpy int64
		want               int64
	}{
		{"example A of 20.1", 5_700, 600, 65, 365, 61},
		{"example C of 20.3", 1_470, 900, 33, 400, 11},
		{"interest free", 5_700, 0, 65, 365, 0},
		{"zero term", 5_700, 600, 0, 365, 0},
		// 10 000 * 1 * 1 / (10 000 * 2) = 0.5: the tie goes to the even 0.
		{"midpoint 0.5 goes to even 0", 10_000, 1, 1, 2, 0},
		// 30 000 * 1 * 1 / 20 000 = 1.5: the tie goes to the even 2.
		{"midpoint 1.5 goes to even 2", 30_000, 1, 1, 2, 2},
		// 50 000 / 20 000 = 2.5: even 2.
		{"midpoint 2.5 goes to even 2", 50_000, 1, 1, 2, 2},
		{"just above the midpoint rounds up", 10_001, 1, 1, 2, 1},
		{"just below the midpoint rounds down", 9_999, 1, 1, 2, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := credit.Interest(tt.x, tt.rate, tt.term, tt.tpy)
			if !ok || got != tt.want {
				t.Errorf("Interest(%d, %d, %d, %d) = %d, %v, want %d", tt.x, tt.rate, tt.term, tt.tpy, got, ok, tt.want)
			}
			if oracle := exactInterest(tt.x, tt.rate, tt.term, tt.tpy); oracle.Int64() != got {
				t.Errorf("Interest(%d, %d, %d, %d) = %d, the exact half-even quotient is %s", tt.x, tt.rate, tt.term, tt.tpy, got, oracle)
			}
		})
	}
}

// TestInterest_LargestNumerator verifies that the largest legal product, which
// does not fit in 64 bits, is exact, and that the result is the one the
// big-integer quotient gives.
//
// Requirement: FR-CRD-027
func TestInterest_LargestNumerator(t *testing.T) {
	t.Parallel()
	// A one-tick year makes the interest the principal times 10 * 3 650 / 1: it
	// does not fit int64, which must be reported.
	if got, ok := credit.Interest(maxMoney, maxRateBP, maxTerm, 1); ok {
		t.Errorf("Interest at the largest numerator with a one-tick year = %d, true, want overflow reported", got)
	}
	// At 365 ticks the quotient is 10^18 * 10^5 * 3650 / (10^4 * 365) = 10^20: also overflow.
	if got, ok := credit.Interest(maxMoney, maxRateBP, maxTerm, ticksYear); ok {
		t.Errorf("Interest(MaxMoney, MaxCreditRateBP, MaxCreditTerm, 365) = %d, true, want overflow reported", got)
	}
	// A year of 365 000 ticks brings the same numerator, MaxMoney * MaxCreditRateBP
	// * MaxCreditTerm = 3.65e26, to a quotient of 10^17 that fits.
	longYear := int64(365_000)
	gotBig, okBig := credit.Interest(maxMoney, maxRateBP, maxTerm, longYear)
	if wantBig := exactInterest(maxMoney, maxRateBP, maxTerm, longYear); !okBig || gotBig != wantBig.Int64() {
		t.Errorf("Interest at the largest numerator = %d, %v, want %s", gotBig, okBig, wantBig)
	}
	// A principal of 10^12 keeps the numerator beyond 64 bits (10^12 * 10^5 * 3650 = 3.65e20)
	// and the quotient inside it.
	x := int64(1_000_000_000_000)
	got, ok := credit.Interest(x, maxRateBP, maxTerm, ticksYear)
	want := exactInterest(x, maxRateBP, maxTerm, ticksYear)
	if !ok || got != want.Int64() {
		t.Errorf("Interest(%d, %d, %d, %d) = %d, %v, want %s", x, maxRateBP, maxTerm, ticksYear, got, ok, want)
	}
}

// TestInterest_Overflow verifies that an input outside the domain and a quotient
// that does not fit in int64 are reported, never wrapped.
//
// Requirement: FR-CRD-027
func TestInterest_Overflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		x, rate, term, tpy int64
	}{
		{"negative principal", -1, 600, 65, 365},
		{"negative rate", 1, -600, 65, 365},
		{"negative term", 1, 600, -65, 365},
		{"no ticks in a year", 1, 600, 65, 0},
		{"rate times term overflows", 1, math.MaxInt64, 2, 365},
		{"year times basis overflows", 1, 600, 65, math.MaxInt64},
		{"quotient overflows", math.MaxInt64, math.MaxInt64, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := credit.Interest(tt.x, tt.rate, tt.term, tt.tpy); ok || got != 0 {
				t.Errorf("Interest(%d, %d, %d, %d) = %d, %v, want 0, false", tt.x, tt.rate, tt.term, tt.tpy, got, ok)
			}
		})
	}
}

// TestEffective_ScaleAndClamp verifies the effective rate: the posted rate times
// the scale over 10 000 rounded half to even, clamped to the bound, 0 for a scale
// of 0, and overflow of the product far above the cap reported as the cap.
//
// Requirement: FR-CRD-024
func TestEffective_ScaleAndClamp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                   string
		rate, scale, max, want int64
		wantOK                 bool
	}{
		{"unscaled", 600, 10_000, maxRateBP, 600, true},
		{"doubled", 600, 20_000, maxRateBP, 1_200, true},
		{"scaled to zero is interest free", 600, 0, maxRateBP, 0, true},
		{"0.5 goes to even 0", 1, 5_000, maxRateBP, 0, true},
		{"1.5 goes to even 2", 3, 5_000, maxRateBP, 2, true},
		{"2.5 goes to even 2", 5, 5_000, maxRateBP, 2, true},
		{"clamped to the bound", 90_000, 20_000, maxRateBP, maxRateBP, true},
		{"product beyond int64 is the cap", math.MaxInt64, math.MaxInt64, maxRateBP, maxRateBP, true},
		{"negative rate", -1, 10_000, maxRateBP, 0, false},
		{"negative scale", 1, -1, maxRateBP, 0, false},
		{"negative bound", 1, 1, -1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := credit.Effective(tt.rate, tt.scale, tt.max)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Effective(%d, %d, %d) = %d, %v, want %d, %v", tt.rate, tt.scale, tt.max, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
