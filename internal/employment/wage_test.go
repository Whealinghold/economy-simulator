package employment_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/employment"
)

// TestWage_ReserveToBidHalfEven_InRange verifies the wage of a pair: the
// reservation plus the worker's share of the margin to the bid, rounded half to
// even, always inside the two limits.
//
// Requirement: FR-EMP-021
func TestWage_ReserveToBidHalfEven_InRange(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		reserve, bid, shareBP int64
		want                  int64
	}{
		// 3 + 7 * 5 000 / 10 000 = 3 + 3.5. The rounding is of the cut
		// (spec-employment.md 5.4.2, round.py): the tie goes to the even 4, so
		// the wage is 7. Rounding the wage 6.5 instead would give 6; the spec's
		// formula and the model both round the cut.
		{"midpoint tie rounds the cut to even", 3, 10, 5000, 7},
		// 4 + 9 * 5 000 / 10 000 = 4 + 4.5: the tie goes to the even 4, wage 8.
		{"tie to the even cut below", 4, 13, 5000, 8},
		// 4 + 7 * 5 000 / 10 000 = 4 + 3.5: the tie goes to the even 4, wage 8.
		{"tie to the even cut above", 4, 11, 5000, 8},
		// share 1: 100 * 1 / 10 000 = 0.01 rounds to 0, the wage is the reserve.
		{"smallest share gives the reserve", 10, 110, 1, 10},
		// share 10 000: the whole margin, the wage is the bid.
		{"whole share gives the bid", 10, 110, 10000, 110},
		// share 0 is outside the configured bound but the function is total: the reserve.
		{"zero share gives the reserve", 10, 110, 0, 10},
		{"bid equal to reserve", 50, 50, 5000, 50},
		// 0 + 1 * 5 000 / 10 000 = 0.5, ties to even 0: the wage is 0 and the round skips the pair.
		{"margin of one rounds to zero", 0, 1, 5000, 0},
		// 0 + 3 * 5 000 / 10 000 = 1.5, ties to even 2.
		{"margin of three", 0, 3, 5000, 2},
		// 1 000 000 000 (MaxPrice) * 10 000 / 10 000 needs the 128-bit product.
		{"largest bid at the whole share", 0, 1_000_000_000, 10000, 1_000_000_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := employment.Wage(tt.reserve, tt.bid, tt.shareBP)
			if !ok || got != tt.want {
				t.Errorf("Wage(%d, %d, %d) = %d, %v, want %d, true", tt.reserve, tt.bid, tt.shareBP, got, ok, tt.want)
			}
			if got < tt.reserve || got > tt.bid {
				t.Errorf("Wage(%d, %d, %d) = %d, outside [reserve, bid]", tt.reserve, tt.bid, tt.shareBP, got)
			}
		})
	}
}

// TestWage_Invalid verifies that an input a caller should have refused at
// admission is reported, not computed.
//
// Requirement: FR-EMP-021
func TestWage_Invalid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		reserve, bid, shareBP int64
	}{
		{"negative reserve", -1, 10, 5000},
		{"bid below reserve", 10, 9, 5000},
		{"share above whole", 1, 10, 10001},
		{"negative share", 1, 10, -1},
		// bid - reserve overflows int64 only when reserve is negative, which is refused first.
		{"extreme negative reserve", math.MinInt64, math.MaxInt64, 5000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := employment.Wage(tt.reserve, tt.bid, tt.shareBP); ok {
				t.Errorf("Wage(%d, %d, %d) = %d, true, want false", tt.reserve, tt.bid, tt.shareBP, got)
			}
		})
	}
}

// TestBill_Product128 verifies the bill of a hire, wage times ticks times
// households, at the values the bounds allow and at a product past int64.
//
// Requirement: FR-EMP-042
func TestBill_Product128(t *testing.T) {
	t.Parallel()
	const (
		maxPrice         = 1_000_000_000 // bounds.go MaxPrice
		maxRecipeTicks   = 1_000         // a placeholder of the longest recipe in this test
		maxEstablishment = 1_000         // spec-employment.md 13
	)
	tests := []struct {
		name                    string
		wage, ticks, households int64
		want                    int64
		wantOK                  bool
	}{
		// 3 * 4 * 5 = 60.
		{"small bill", 3, 4, 5, 60, true},
		// 10^9 * 10^3 * 10^3 = 10^15: at the bounds the bill fits int64 (MaxMoney is 10^18).
		{"bill at the bounds", maxPrice, maxRecipeTicks, maxEstablishment, 1_000_000_000_000_000, true},
		// 2^62 * 2 = 2^63 does not fit int64 by one.
		{"product past int64", 1 << 62, 2, 1, 0, false},
		// the first product fits, the second one does not: 2^40 * 2^20 = 2^60, then * 2^4 = 2^64.
		{"second product past int64", 1 << 40, 1 << 20, 1 << 4, 0, false},
		{"zero wage", 0, 5, 5, 0, true},
		{"negative wage", -1, 5, 5, 0, false},
		{"negative ticks", 1, -5, 5, 0, false},
		{"negative households", 1, 5, -5, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := employment.Bill(tt.wage, tt.ticks, tt.households)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("Bill(%d, %d, %d) = %d, %v, want %d, %v", tt.wage, tt.ticks, tt.households, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// FuzzWage verifies that Wage never panics or overflows and that a wage it
// returns lies in [reserve, bid].
//
// Requirement: NFR-EMP-003
func FuzzWage(f *testing.F) {
	f.Add(int64(3), int64(10), int64(5000))
	f.Add(int64(0), int64(math.MaxInt64), int64(10000))
	f.Add(int64(math.MaxInt64), int64(math.MaxInt64), int64(1))
	f.Add(int64(-1), int64(0), int64(0))
	f.Fuzz(func(t *testing.T, reserve, bid, shareBP int64) {
		got, ok := employment.Wage(reserve, bid, shareBP)
		if !ok {
			return
		}
		if got < reserve || got > bid {
			t.Fatalf("Wage(%d, %d, %d) = %d, outside [reserve, bid]", reserve, bid, shareBP, got)
		}
	})
}
