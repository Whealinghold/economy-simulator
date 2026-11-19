package policy_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/policy"
)

// TestGauge_Examples verifies the gauge numbers of spec-policy.md 14.2: an unfed
// share of rdiv(10 000 * 90, 1 000) = 900 bp, a mean ratio of
// rdiv(7 400 000, 600) = 12 333 (12 333.33) and a price of
// rdiv(400 * 24 + 450 * 20, 850) = 22 (21.88) where the row of age 5 is not
// counted at MaxAge 3.
//
// Requirement: FR-PLC-034
func TestGauge_Examples(t *testing.T) {
	t.Parallel()
	if got, ok := policy.UnfedShare(90, 1_000); !ok || got != 900 {
		t.Errorf("UnfedShare(90, 1000) = %d, %v; want 900, true", got, ok)
	}
	ratios := []policy.Ratio{{BP: 12_000, Households: 300}, {BP: 8_000, Households: 100}, {BP: 15_000, Households: 200}}
	if got, ok := policy.MeanRatio(ratios); !ok || got != 12_333 {
		t.Errorf("MeanRatio = %d, %v; want 12 333, true", got, ok)
	}
	prices := []policy.PriceRow{
		{Households: 400, Price: 24, Age: 1},
		{Households: 150, Price: 30, Age: 5},
		{Households: 450, Price: 20, Age: 2},
	}
	if got, ok := policy.Price(prices, 3); !ok || got != 22 {
		t.Errorf("Price(maxAge 3) = %d, %v; want 22, true", got, ok)
	}
}

// TestGauge_RoundsHalfToEven verifies the rounding rule of 8.3.4 on exact ties.
//
// Requirement: FR-PLC-034
func TestGauge_RoundsHalfToEven(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		unfed, house int64
		want         int64
	}{
		{"0.5 goes to 0", 1, 20_000, 0},
		{"1.5 goes to 2", 3, 20_000, 2},
		{"2.5 goes to 2", 5, 20_000, 2},
		{"just above a half rounds up", 10_001, 40_000_000, 3},
	}
	for _, tt := range tests {
		got, ok := policy.UnfedShare(tt.unfed, tt.house)
		if !ok || got != tt.want {
			t.Errorf("%s: UnfedShare(%d, %d) = %d, %v; want %d", tt.name, tt.unfed, tt.house, got, ok, tt.want)
		}
	}
	rows := []policy.PriceRow{{Households: 1, Price: 5}, {Households: 1, Price: 6}}
	if got, ok := policy.Price(rows, 0); !ok || got != 6 { // 5.5 goes to the even 6
		t.Errorf("Price(5, 6) = %d, %v; want 6 (half to even)", got, ok)
	}
	rows = []policy.PriceRow{{Households: 1, Price: 4}, {Households: 1, Price: 5}}
	if got, ok := policy.Price(rows, 0); !ok || got != 4 { // 4.5 goes to the even 4
		t.Errorf("Price(4, 5) = %d, %v; want 4 (half to even)", got, ok)
	}
}

// TestGauge_UndefinedTreasuryAndShare verifies the "undefined when" column of
// 8.3.2 for the treasury and unfed_share gauges.
//
// Requirement: FR-PLC-034
func TestGauge_UndefinedTreasuryAndShare(t *testing.T) {
	t.Parallel()
	if _, ok := policy.TreasuryReading(500, false); ok {
		t.Error("TreasuryReading of an ungoverned jurisdiction is defined, want undefined")
	}
	if got, ok := policy.TreasuryReading(-40, true); !ok || got != -40 {
		t.Errorf("TreasuryReading(-40, governed) = %d, %v; want -40, true", got, ok)
	}
	for _, tt := range []struct{ unfed, house int64 }{{5, 0}, {0, 0}, {-1, 10}, {1, -3}} {
		if got, ok := policy.UnfedShare(tt.unfed, tt.house); ok {
			t.Errorf("UnfedShare(%d, %d) = %d, true; want undefined", tt.unfed, tt.house, got)
		}
	}
}

// TestGauge_UndefinedMeanRatio verifies the "undefined when" column of 8.3.2
// for the mean_ratio gauge.
//
// Requirement: FR-PLC-034
func TestGauge_UndefinedMeanRatio(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		rows []policy.Ratio
	}{
		{"no rows", nil},
		{"zero households", []policy.Ratio{{BP: 9_000, Households: 0}}},
		{"negative ratio", []policy.Ratio{{BP: -1, Households: 5}}},
		{"negative households", []policy.Ratio{{BP: 1, Households: -5}}},
	}
	for _, tt := range tests {
		if got, ok := policy.MeanRatio(tt.rows); ok {
			t.Errorf("MeanRatio(%s) = %d, true; want undefined", tt.name, got)
		}
	}
}

// TestGauge_UndefinedPrice verifies the "undefined when" column of 8.3.2 for the
// price gauge: no row, only stale rows and no households are undefined, and a
// row exactly at MaxAge counts.
//
// Requirement: FR-PLC-034
func TestGauge_UndefinedPrice(t *testing.T) {
	t.Parallel()
	if _, ok := policy.Price(nil, 3); ok {
		t.Error("Price(no rows) is defined, want undefined")
	}
	old := []policy.PriceRow{{Households: 10, Price: 9, Age: 4}}
	if _, ok := policy.Price(old, 3); ok {
		t.Error("Price(only a stale row) is defined, want undefined")
	}
	if got, ok := policy.Price(old, 4); !ok || got != 9 {
		t.Errorf("Price at the age limit = %d, %v; want 9, true", got, ok)
	}
	if _, ok := policy.Price([]policy.PriceRow{{Households: 0, Price: 9}}, 3); ok {
		t.Error("Price(zero households) is defined, want undefined")
	}
	if _, ok := policy.Price([]policy.PriceRow{{Households: 10, Price: 9, Age: -1}}, 3); ok {
		t.Error("Price(negative age) is defined, want undefined")
	}
}

// TestGauge_OverflowIsUndefined verifies that a sum that leaves its range is an
// undefined reading, never a wrapped one.
//
// Requirement: FR-PLC-034
func TestGauge_OverflowIsUndefined(t *testing.T) {
	t.Parallel()
	if _, ok := policy.UnfedShare(math.MaxInt64, 1); ok {
		t.Error("UnfedShare(MaxInt64, 1) is defined, want undefined (the share leaves int64)")
	}
	big := []policy.Ratio{{BP: math.MaxInt64, Households: math.MaxInt64}, {BP: math.MaxInt64, Households: math.MaxInt64}}
	if _, ok := policy.MeanRatio(big); ok {
		t.Error("MeanRatio(2 x MaxInt64^2) is defined, want undefined")
	}
	hh := []policy.Ratio{{BP: 1, Households: math.MaxInt64}, {BP: 1, Households: 1}}
	if _, ok := policy.MeanRatio(hh); ok {
		t.Error("MeanRatio(households sum leaves int64) is defined, want undefined")
	}
	ph := []policy.PriceRow{{Households: math.MaxInt64, Price: 1}, {Households: 1, Price: 1}}
	if _, ok := policy.Price(ph, 0); ok {
		t.Error("Price(households sum leaves int64) is defined, want undefined")
	}
	pp := []policy.PriceRow{{Households: math.MaxInt64, Price: math.MaxInt64}, {Households: math.MaxInt64, Price: math.MaxInt64}}
	if _, ok := policy.Price(pp, 0); ok {
		t.Error("Price(products leave 128 bits) is defined, want undefined")
	}
}

// TestGauge_OrderIndependent verifies that the sums are exact, so the reading
// does not depend on the order of the rows (8.3.4).
//
// Requirement: FR-PLC-034
func TestGauge_OrderIndependent(t *testing.T) {
	t.Parallel()
	a := []policy.Ratio{{BP: 12_000, Households: 300}, {BP: 8_000, Households: 100}, {BP: 15_000, Households: 200}}
	b := []policy.Ratio{a[2], a[0], a[1]}
	ga, _ := policy.MeanRatio(a)
	gb, _ := policy.MeanRatio(b)
	if ga != gb {
		t.Errorf("MeanRatio depends on the order of the rows: %d and %d", ga, gb)
	}
}
