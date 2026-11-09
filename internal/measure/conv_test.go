package measure_test

import (
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/measure"
)

// TestToInt64_Overflow verifies the checked conversion at both ends of int64
// and one past each.
//
// Requirement: FR-IND-009
func TestToInt64_Overflow(t *testing.T) {
	t.Parallel()
	two63 := new(big.Int).Lsh(big.NewInt(1), 63)
	tests := []struct {
		name   string
		v      *big.Int
		want   int64
		wantOK bool
	}{
		{"zero", big.NewInt(0), 0, true},
		{"max", big.NewInt(math.MaxInt64), math.MaxInt64, true},
		{"min", big.NewInt(math.MinInt64), math.MinInt64, true},
		{"one past max", new(big.Int).Set(two63), 0, false},
		{"one past min", new(big.Int).Sub(new(big.Int).Neg(two63), big.NewInt(1)), 0, false},
		{"in the high word", new(big.Int).Lsh(big.NewInt(1), 100), 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := fromBig(t, tt.v)
			got, ok := measure.ToInt64(v)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("ToInt64(%v) = %d, %v, want %d, %v", tt.v, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// fromBig builds the Int128 of a big.Int of magnitude below 2^127 from the
// public operations, so the tests need no constructor the package does not have.
func fromBig(t *testing.T, b *big.Int) measure.Int128 {
	t.Helper()
	mag := new(big.Int).Abs(b)
	hi, lo := new(big.Int).Rsh(mag, 64), new(big.Int).And(mag, new(big.Int).SetUint64(math.MaxUint64))
	u, ok := measure.FromUint128(measure.Uint128{Hi: hi.Uint64(), Lo: lo.Uint64()})
	if !ok {
		t.Fatalf("fromBig(%v): magnitude past 2^127", b)
	}
	if b.Sign() < 0 {
		return u.Neg()
	}
	return u
}

// TestGuardBeforeTimes10000 verifies the guard that stands before every
// multiplication by 10 000: the limit is floor(2^127 / 10 000), a value at it
// is multiplied exactly, and one above it is refused, never wrapped.
//
// Requirement: FR-IND-009
func TestGuardBeforeTimes10000(t *testing.T) {
	t.Parallel()
	limit := new(big.Int).Lsh(big.NewInt(1), 127)
	limit.Quo(limit, big.NewInt(10000))
	tests := []struct {
		name   string
		v      *big.Int
		wantOK bool
	}{
		{"zero", big.NewInt(0), true},
		{"at the limit", new(big.Int).Set(limit), true},
		{"negative at the limit", new(big.Int).Neg(limit), true},
		{"one above the limit", new(big.Int).Add(limit, big.NewInt(1)), false},
		{"negative one above", new(big.Int).Neg(new(big.Int).Add(limit, big.NewInt(1))), false},
		{"2^126", new(big.Int).Lsh(big.NewInt(1), 126), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := measure.MulBP(fromBig(t, tt.v))
			if ok != tt.wantOK {
				t.Fatalf("MulBP(%v) ok = %v, want %v", tt.v, ok, tt.wantOK)
			}
			if want := new(big.Int).Mul(tt.v, big.NewInt(10000)); ok && bigOf(got).Cmp(want) != 0 {
				t.Errorf("MulBP(%v) = %v, want %v", tt.v, bigOf(got), want)
			}
		})
	}
	// the product at the limit is below 2^127, the largest Int128
	if p := new(big.Int).Mul(limit, big.NewInt(10000)); p.Cmp(new(big.Int).Lsh(big.NewInt(1), 127)) >= 0 {
		t.Errorf("limit * 10000 = %v, want below 2^127", p)
	}
}

// TestClampBP verifies the clamp of an index, ratio or growth at 2^31 - 1: the
// value at the bound is cut and flagged, a negative growth is cut at the
// negative bound, and anything below passes unchanged, including a value that
// does not fit int64.
//
// Requirement: FR-IND-023
func TestClampBP(t *testing.T) {
	t.Parallel()
	const m = measure.MaxIndexBP
	if m != 1<<31-1 {
		t.Fatalf("MaxIndexBP = %d, want 2^31 - 1", m)
	}
	tests := []struct {
		name        string
		v           *big.Int
		want        int64
		wantClamped bool
	}{
		{"zero", big.NewInt(0), 0, false},
		{"just below", big.NewInt(m - 1), m - 1, false},
		{"at the bound", big.NewInt(m), m, true},
		{"above", big.NewInt(m + 1), m, true},
		{"negative just below", big.NewInt(-(m - 1)), -(m - 1), false},
		{"negative at the bound", big.NewInt(-m), -m, true},
		{"negative above", big.NewInt(-m - 5), -m, true},
		{"past int64", new(big.Int).Lsh(big.NewInt(1), 90), m, true},
		{"negative past int64", new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 90)), -m, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, clamped := measure.ClampBP(fromBig(t, tt.v))
			if got != tt.want || clamped != tt.wantClamped {
				t.Errorf("ClampBP(%v) = %d, %v, want %d, %v", tt.v, got, clamped, tt.want, tt.wantClamped)
			}
		})
	}
}

// TestErrors_AreNamed verifies that each sentinel error of the package has a
// message that names the package, so an error that reaches a log says where it
// came from.
//
// Guard: the sentinels are matched with errors.Is, so an empty message would
// pass every other test and print nothing useful.
func TestErrors_AreNamed(t *testing.T) {
	t.Parallel()
	for _, err := range []error{measure.ErrOverflow, measure.ErrNegative, measure.ErrRange} {
		if msg := err.Error(); !strings.HasPrefix(msg, "measure: ") || len(msg) < 12 {
			t.Errorf("error message %q, want %q followed by a reason", msg, "measure: ")
		}
	}
}
