package indicator_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/indicator"
)

// TestQuotient_UndefinedIsNotAnError verifies the quotient of
// spec-indicators.md 5.2.2 and 5.2.4: half to even and symmetric for a negative
// numerator, a zero divisor reported as not ok with no error and no panic, and
// the one quotient that does not fit int64 reported as ErrOverflow.
//
// Requirement: FR-IND-012
func TestQuotient_UndefinedIsNotAnError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		n, d   int64
		want   int64
		wantOK bool
		err    error
	}{
		{"minus five over two", -5, 2, -2, true, nil},
		{"minus three over two", -3, 2, -2, true, nil},
		{"twenty five over ten", 25, 10, 2, true, nil},
		{"thirty five over ten", 35, 10, 4, true, nil},
		{"zero divisor", 7, 0, 0, false, nil},
		{"zero over zero", 0, 0, 0, false, nil},
		{"min over minus one", math.MinInt64, -1, 0, false, indicator.ErrOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok, err := indicator.Quotient(tt.n, tt.d)
			if got != tt.want || ok != tt.wantOK || !errors.Is(err, tt.err) {
				t.Errorf("Quotient(%d, %d) = %d, %v, %v, want %d, %v, %v", tt.n, tt.d, got, ok, err, tt.want, tt.wantOK, tt.err)
			}
		})
	}
}

// TestAnnualise_ExactFromSums verifies the annualisation of
// spec-indicators.md 5.7.2: rdiv(total * TicksPerYear, n) with the product in
// 128 bits. A total of 1 150 over a window of 2 ticks in a year of 365 is
// 1 150 * 365 / 2 = 209 875 exactly; an odd window rounds half to even; a window
// with no ticks is undefined, a year below one tick is a range error, and a
// result past int64 is an overflow, although the product needs 128 bits.
//
// Requirement: FR-IND-053
func TestAnnualise_ExactFromSums(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		total, y, n int64
		want        int64
		wantOK      bool
		err         error
	}{
		{"exact", 1150, 365, 2, 209875, true, nil},
		// 5 * 365 / 4 = 456.25, rounds to 456
		{"rounds down", 5, 365, 4, 456, true, nil},
		// 1 * 365 / 2 = 182.5, an exact half goes to the even 182
		{"half to even", 1, 365, 2, 182, true, nil},
		{"negative total", -1, 365, 2, -182, true, nil},
		{"product past int64 but quotient inside", math.MaxInt64, 100000, 100000, math.MaxInt64, true, nil},
		{"no ticks", 5, 365, 0, 0, false, nil},
		{"negative ticks", 5, 365, -3, 0, false, nil},
		{"year below one tick", 5, 0, 2, 0, false, indicator.ErrRange},
		{"result past int64", math.MaxInt64, 100000, 1, 0, false, indicator.ErrOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok, err := indicator.Annualise(tt.total, tt.y, tt.n)
			if got != tt.want || ok != tt.wantOK || !errors.Is(err, tt.err) {
				t.Errorf("Annualise(%d, %d, %d) = %d, %v, %v, want %d, %v, %v", tt.total, tt.y, tt.n, got, ok, err, tt.want, tt.wantOK, tt.err)
			}
		})
	}
}
